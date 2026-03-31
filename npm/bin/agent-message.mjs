#!/usr/bin/env node

import { spawn, spawnSync } from 'node:child_process'
import { closeSync, existsSync, mkdirSync, openSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { access, constants } from 'node:fs/promises'
import os from 'node:os'
import { dirname, join, resolve } from 'node:path'
import process from 'node:process'
import { fileURLToPath } from 'node:url'

const DEFAULT_API_HOST = '127.0.0.1'
const DEFAULT_API_PORT = 8080
const DEFAULT_WEB_HOST = '127.0.0.1'
const DEFAULT_WEB_PORT = 8788
const STARTUP_ATTEMPTS = 40
const STARTUP_DELAY_MS = 500
const PROCESS_STOP_DELAY_MS = 1000

const scriptDir = dirname(fileURLToPath(import.meta.url))
const packageRoot = resolve(scriptDir, '..', '..')
const bundleRoot = resolve(packageRoot, 'npm', 'runtime')
const bundleBinDir = join(bundleRoot, 'bin')
const bundleGatewayPath = join(bundleRoot, 'agent_gateway.mjs')
const bundleWebDistDir = join(bundleRoot, 'web-dist')

const lifecycleCommands = new Set(['start', 'stop', 'status'])

async function main() {
  const [command, ...rest] = process.argv.slice(2)
  if (!command) {
    printRootUsage()
    process.exit(1)
  }

  if (command === '--help' || command === '-h' || command === 'help') {
    printRootUsage()
    return
  }

  if (lifecycleCommands.has(command)) {
    const options = parseLifecycleOptions(rest)
    await ensureBundleReady()

    if (command === 'start') {
      await startStack(options)
      return
    }
    if (command === 'stop') {
      await stopStack(options, { quiet: false })
      return
    }
    await printStatus(options)
    return
  }

  await ensureBundleReady()
  delegateToBundledCli([command, ...rest])
}

function printRootUsage() {
  console.error(`Usage:
  agent-message start [--runtime-dir <dir>] [--api-host <host>] [--api-port <port>] [--web-host <host>] [--web-port <port>]
  agent-message stop [--runtime-dir <dir>]
  agent-message status [--runtime-dir <dir>] [--api-host <host>] [--api-port <port>] [--web-host <host>] [--web-port <port>]
  agent-message <existing-cli-command> [...args]`)
}

function parseLifecycleOptions(args) {
  const options = {
    runtimeDir: join(os.homedir(), '.agent-message'),
    apiHost: DEFAULT_API_HOST,
    apiPort: DEFAULT_API_PORT,
    webHost: DEFAULT_WEB_HOST,
    webPort: DEFAULT_WEB_PORT,
  }

  for (let index = 0; index < args.length; index += 1) {
    const arg = args[index]
    if (arg === '--help' || arg === '-h') {
      printRootUsage()
      process.exit(0)
    }

    if (arg === '--runtime-dir') {
      options.runtimeDir = requireOptionValue(args, ++index, arg)
      continue
    }
    if (arg === '--api-host') {
      options.apiHost = requireOptionValue(args, ++index, arg)
      continue
    }
    if (arg === '--web-host') {
      options.webHost = requireOptionValue(args, ++index, arg)
      continue
    }
    if (arg === '--api-port') {
      options.apiPort = parsePort(requireOptionValue(args, ++index, arg), arg)
      continue
    }
    if (arg === '--web-port') {
      options.webPort = parsePort(requireOptionValue(args, ++index, arg), arg)
      continue
    }

    throw new Error(`unknown option: ${arg}`)
  }

  return options
}

function requireOptionValue(args, index, flag) {
  const value = args[index]
  if (!value || value.startsWith('-')) {
    throw new Error(`missing value for ${flag}`)
  }
  return value
}

function parsePort(value, flag) {
  const parsed = Number(value)
  if (!Number.isInteger(parsed) || parsed <= 0 || parsed > 65535) {
    throw new Error(`invalid port for ${flag}: ${value}`)
  }
  return parsed
}

async function ensureBundleReady() {
  const requiredPaths = [
    bundleGatewayPath,
    bundleWebDistDir,
    resolveBinaryPath('agent-message-server'),
    resolveBinaryPath('agent-message-cli'),
  ]

  for (const target of requiredPaths) {
    if (!existsSync(target)) {
      throw new Error(
        `npm bundle is incomplete: missing ${target}. Run "npm run prepare:npm-bundle" before packing or publishing.`,
      )
    }
  }

  await access(resolveBinaryPath('agent-message-server'), constants.X_OK)
  await access(resolveBinaryPath('agent-message-cli'), constants.X_OK)
}

function resolveBinaryPath(baseName) {
  if (process.platform !== 'darwin') {
    throw new Error(`unsupported platform: ${process.platform}. This package currently supports macOS only.`)
  }

  let archSuffix
  if (process.arch === 'arm64') {
    archSuffix = 'darwin-arm64'
  } else if (process.arch === 'x64') {
    archSuffix = 'darwin-amd64'
  } else {
    throw new Error(`unsupported architecture: ${process.arch}. Expected arm64 or x64 on macOS.`)
  }

  return join(bundleBinDir, `${baseName}-${archSuffix}`)
}

function runtimePaths(runtimeDir) {
  return {
    runtimeDir,
    logDir: join(runtimeDir, 'logs'),
    uploadDir: join(runtimeDir, 'uploads'),
    serverLog: join(runtimeDir, 'logs', 'server.log'),
    gatewayLog: join(runtimeDir, 'logs', 'gateway.log'),
    serverPidfile: join(runtimeDir, 'server.pid'),
    gatewayPidfile: join(runtimeDir, 'gateway.pid'),
    sqlitePath: join(runtimeDir, 'agent_message.sqlite'),
  }
}

async function startStack(options) {
  const paths = runtimePaths(options.runtimeDir)
  mkdirSync(paths.runtimeDir, { recursive: true })
  mkdirSync(paths.logDir, { recursive: true })
  mkdirSync(paths.uploadDir, { recursive: true })

  await stopStack(options, { quiet: true })

  writeFileSync(paths.serverLog, '')
  writeFileSync(paths.gatewayLog, '')

  const serverBinary = resolveBinaryPath('agent-message-server')
  const gatewayChild = { pid: null }

  try {
    const serverChild = spawnDetached(serverBinary, [], {
      ...process.env,
      SERVER_ADDR: `${options.apiHost}:${options.apiPort}`,
      DB_DRIVER: 'sqlite',
      SQLITE_DSN: paths.sqlitePath,
      UPLOAD_DIR: paths.uploadDir,
      CORS_ALLOWED_ORIGINS: '*',
    }, paths.serverLog)
    writeFileSync(paths.serverPidfile, `${serverChild.pid}\n`)

    await waitForHttp(`http://${options.apiHost}:${options.apiPort}/healthz`, 'API server')

    gatewayChild.pid = spawnDetached(
      process.execPath,
      [bundleGatewayPath],
      {
        ...process.env,
        AGENT_GATEWAY_HOST: options.webHost,
        AGENT_GATEWAY_PORT: String(options.webPort),
        AGENT_API_ORIGIN: `http://${options.apiHost}:${options.apiPort}`,
        AGENT_WEB_DIST: bundleWebDistDir,
      },
      paths.gatewayLog,
    ).pid
    writeFileSync(paths.gatewayPidfile, `${gatewayChild.pid}\n`)

    await waitForHttp(`http://${options.webHost}:${options.webPort}`, 'web gateway')
  } catch (error) {
    await stopStack(options, { quiet: true })
    throw error
  }

  console.log('Agent Message is up.')
  console.log(`API:  http://${options.apiHost}:${options.apiPort}`)
  console.log(`Web:  http://${options.webHost}:${options.webPort}`)
  console.log(`Logs: ${paths.serverLog} ${paths.gatewayLog}`)
}

async function stopStack(options, { quiet }) {
  const paths = runtimePaths(options.runtimeDir)
  const stoppedServer = await killFromPidfile(paths.serverPidfile)
  const stoppedGateway = await killFromPidfile(paths.gatewayPidfile)

  if (!quiet) {
    if (stoppedServer || stoppedGateway) {
      console.log('Agent Message is stopped.')
    } else {
      console.log('Agent Message is not running.')
    }
  }
}

async function printStatus(options) {
  const paths = runtimePaths(options.runtimeDir)
  const serverPid = readPidfile(paths.serverPidfile)
  const gatewayPid = readPidfile(paths.gatewayPidfile)
  const serverRunning = serverPid !== null && isPidAlive(serverPid)
  const gatewayRunning = gatewayPid !== null && isPidAlive(gatewayPid)
  const apiHealthy = serverRunning ? await isHttpReady(`http://${options.apiHost}:${options.apiPort}/healthz`) : false
  const webHealthy = gatewayRunning ? await isHttpReady(`http://${options.webHost}:${options.webPort}`) : false

  console.log(`API server: ${serverRunning ? 'running' : 'stopped'}${apiHealthy ? ' (healthy)' : ''}`)
  console.log(`Web gateway: ${gatewayRunning ? 'running' : 'stopped'}${webHealthy ? ' (healthy)' : ''}`)
  console.log(`Runtime dir: ${paths.runtimeDir}`)
  console.log(`API URL: http://${options.apiHost}:${options.apiPort}`)
  console.log(`Web URL: http://${options.webHost}:${options.webPort}`)
}

function delegateToBundledCli(args) {
  const cliBinary = resolveBinaryPath('agent-message-cli')
  const result = spawnSync(cliBinary, args, {
    stdio: 'inherit',
    env: process.env,
  })

  if (result.error) {
    throw result.error
  }

  if (result.signal) {
    process.kill(process.pid, result.signal)
    return
  }

  process.exit(result.status ?? 1)
}

function spawnDetached(command, args, env, logFile) {
  const stdoutFd = openSync(logFile, 'a')
  const stderrFd = openSync(logFile, 'a')

  try {
    const child = spawn(command, args, {
      detached: true,
      stdio: ['ignore', stdoutFd, stderrFd],
      env,
    })
    child.unref()

    if (!Number.isInteger(child.pid) || child.pid <= 0) {
      throw new Error(`failed to launch background process: ${command}`)
    }
    return { pid: child.pid }
  } finally {
    closeSync(stdoutFd)
    closeSync(stderrFd)
  }
}

function readPidfile(pidfile) {
  if (!existsSync(pidfile)) {
    return null
  }

  const raw = readFileSync(pidfile, 'utf8').trim()
  const pid = Number(raw)
  if (!Number.isInteger(pid) || pid <= 0) {
    rmSync(pidfile, { force: true })
    return null
  }
  return pid
}

function isPidAlive(pid) {
  try {
    process.kill(pid, 0)
    return true
  } catch (error) {
    if (error && typeof error === 'object' && 'code' in error && error.code === 'ESRCH') {
      return false
    }
    return true
  }
}

async function killFromPidfile(pidfile) {
  const pid = readPidfile(pidfile)
  if (pid === null) {
    return false
  }

  if (isPidAlive(pid)) {
    safeKill(pid, 'SIGTERM')
    await sleep(PROCESS_STOP_DELAY_MS)
    if (isPidAlive(pid)) {
      safeKill(pid, 'SIGKILL')
    }
  }

  rmSync(pidfile, { force: true })
  return true
}

async function waitForHttp(url, label) {
  for (let attempt = 0; attempt < STARTUP_ATTEMPTS; attempt += 1) {
    if (await isHttpReady(url)) {
      return
    }
    await sleep(STARTUP_DELAY_MS)
  }
  throw new Error(`${label} did not become ready: ${url}`)
}

async function isHttpReady(url) {
  try {
    const controller = new AbortController()
    const timeout = setTimeout(() => controller.abort(), 1000)
    const response = await fetch(url, { signal: controller.signal })
    clearTimeout(timeout)
    return response.ok
  } catch {
    return false
  }
}

function sleep(ms) {
  return new Promise((resolvePromise) => {
    setTimeout(resolvePromise, ms)
  })
}

function safeKill(pid, signal) {
  try {
    process.kill(pid, signal)
  } catch (error) {
    if (!(error && typeof error === 'object' && 'code' in error && error.code === 'ESRCH')) {
      throw error
    }
  }
}

main().catch((error) => {
  console.error(error instanceof Error ? error.message : String(error))
  process.exit(1)
})
