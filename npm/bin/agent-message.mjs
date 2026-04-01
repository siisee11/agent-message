#!/usr/bin/env node

import { spawn, spawnSync } from 'node:child_process'
import { generateKeyPairSync } from 'node:crypto'
import { closeSync, existsSync, mkdirSync, openSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { access, constants } from 'node:fs/promises'
import os from 'node:os'
import { dirname, join, resolve } from 'node:path'
import process from 'node:process'
import { fileURLToPath } from 'node:url'

const DEFAULT_API_HOST = '127.0.0.1'
const DEFAULT_API_PORT = 45180
const DEFAULT_WEB_HOST = '127.0.0.1'
const DEFAULT_WEB_PORT = 45788
const STARTUP_ATTEMPTS = 40
const STARTUP_DELAY_MS = 500
const PROCESS_STOP_DELAY_MS = 1000
const DEFAULT_WEB_PUSH_SUBJECT = 'mailto:agent-message@local.invalid'
const DEFAULT_TUNNEL_WEB_PUSH_SUBJECT = 'https://agent-message.namjaeyoun.com'

const scriptDir = dirname(fileURLToPath(import.meta.url))
const packageRoot = resolve(scriptDir, '..', '..')
const bundleRoot = resolve(packageRoot, 'npm', 'runtime')
const bundleBinDir = join(bundleRoot, 'bin')
const bundleGatewayPath = join(bundleRoot, 'agent_gateway.mjs')
const bundleWebDistDir = join(bundleRoot, 'web-dist')
const sourceServerDir = resolve(packageRoot, 'server')
const sourceWebDir = resolve(packageRoot, 'web')
const sourceGatewayPath = resolve(packageRoot, 'deploy', 'agent_gateway.mjs')
const sourceWebDistDir = resolve(sourceWebDir, 'dist')
const tunnelConfigPath = resolve(packageRoot, 'deploy', 'agent_tunnel_config.yml')
const tunnelName = 'agent-namjaeyoun-com'

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
    if (!options.dev) {
      await ensureBundleReady()
    }

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
  agent-message start [--dev] [--with-tunnel] [--runtime-dir <dir>] [--api-host <host>] [--api-port <port>] [--web-host <host>] [--web-port <port>]
  agent-message stop [--dev] [--with-tunnel] [--runtime-dir <dir>]
  agent-message status [--dev] [--runtime-dir <dir>] [--api-host <host>] [--api-port <port>] [--web-host <host>] [--web-port <port>]
  agent-message <existing-cli-command> [...args]`)
}

function parseLifecycleOptions(args) {
  const options = {
    dev: false,
    withTunnel: false,
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

    if (arg === '--dev') {
      options.dev = true
      continue
    }
    if (arg === '--with-tunnel' || arg === '--all') {
      options.withTunnel = true
      continue
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
    binDir: join(runtimeDir, 'bin'),
    logDir: join(runtimeDir, 'logs'),
    uploadDir: join(runtimeDir, 'uploads'),
    serverLog: join(runtimeDir, 'logs', 'server.log'),
    gatewayLog: join(runtimeDir, 'logs', 'gateway.log'),
    tunnelLog: join(runtimeDir, 'logs', 'named-tunnel.log'),
    serverPidfile: join(runtimeDir, 'server.pid'),
    gatewayPidfile: join(runtimeDir, 'gateway.pid'),
    tunnelPidfile: join(runtimeDir, 'named-tunnel.pid'),
    stackMetadataPath: join(runtimeDir, 'stack.json'),
    sqlitePath: join(runtimeDir, 'agent_message.sqlite'),
    webPushConfigPath: join(runtimeDir, 'web-push.json'),
  }
}

async function startStack(options) {
  const paths = runtimePaths(options.runtimeDir)
  mkdirSync(paths.runtimeDir, { recursive: true })
  mkdirSync(paths.binDir, { recursive: true })
  mkdirSync(paths.logDir, { recursive: true })
  mkdirSync(paths.uploadDir, { recursive: true })

  await stopStack(options, { quiet: true })

  writeFileSync(paths.serverLog, '')
  writeFileSync(paths.gatewayLog, '')
  if (options.withTunnel) {
    ensureTunnelTargetMatchesDefaults(options)
    writeFileSync(paths.tunnelLog, '')
  }

  const launchSpec = options.dev ? buildDevLaunchSpec(paths) : buildBundledLaunchSpec()

  try {
    const webPushConfig = ensureWebPushConfig(paths, options)
    const serverChild = spawnDetached(launchSpec.serverCommand, launchSpec.serverArgs, {
      ...process.env,
      SERVER_ADDR: `${options.apiHost}:${options.apiPort}`,
      DB_DRIVER: 'sqlite',
      SQLITE_DSN: paths.sqlitePath,
      UPLOAD_DIR: paths.uploadDir,
      CORS_ALLOWED_ORIGINS: '*',
      WEB_PUSH_VAPID_PUBLIC_KEY: webPushConfig.publicKey,
      WEB_PUSH_VAPID_PRIVATE_KEY: webPushConfig.privateKey,
      WEB_PUSH_SUBJECT: webPushConfig.subject,
    }, paths.serverLog)
    writeFileSync(paths.serverPidfile, `${serverChild.pid}\n`)

    await waitForHttp(`http://${options.apiHost}:${options.apiPort}/healthz`, 'API server')

    const gatewayChild = spawnDetached(
      launchSpec.gatewayCommand,
      launchSpec.gatewayArgs,
      {
        ...process.env,
        AGENT_GATEWAY_HOST: options.webHost,
        AGENT_GATEWAY_PORT: String(options.webPort),
        AGENT_API_ORIGIN: `http://${options.apiHost}:${options.apiPort}`,
        AGENT_WEB_DIST: launchSpec.webDistDir,
      },
      paths.gatewayLog,
    )
    writeFileSync(paths.gatewayPidfile, `${gatewayChild.pid}\n`)

    await waitForHttp(`http://${options.webHost}:${options.webPort}`, 'web gateway')
    if (options.withTunnel) {
      const tunnelChild = spawnDetached(
        'cloudflared',
        ['tunnel', '--config', tunnelConfigPath, 'run', tunnelName],
        process.env,
        paths.tunnelLog,
      )
      writeFileSync(paths.tunnelPidfile, `${tunnelChild.pid}\n`)
      await waitForLogMessage(paths.tunnelLog, 'Registered tunnel connection', 'named tunnel')
    }
    writeStackMetadata(paths.stackMetadataPath, options)
  } catch (error) {
    await stopStack(options, { quiet: true })
    throw error
  }

  console.log('Agent Message is up.')
  console.log(`API:  http://${options.apiHost}:${options.apiPort}`)
  console.log(`Web:  http://${options.webHost}:${options.webPort}`)
  console.log(`Logs: ${paths.serverLog} ${paths.gatewayLog}`)
  console.log(`Web Push: ${paths.webPushConfigPath}`)
  if (options.withTunnel) {
    console.log(`Tunnel: https://agent.namjaeyoun.com`)
    console.log(`Tunnel log: ${paths.tunnelLog}`)
  }
}

function ensureWebPushConfig(paths, options) {
  const envPublicKey = process.env.WEB_PUSH_VAPID_PUBLIC_KEY?.trim() ?? ''
  const envPrivateKey = process.env.WEB_PUSH_VAPID_PRIVATE_KEY?.trim() ?? ''
  const envSubject = process.env.WEB_PUSH_SUBJECT?.trim() ?? ''
  const defaultSubject = resolveDefaultWebPushSubject(options)

  if ((envPublicKey && !envPrivateKey) || (!envPublicKey && envPrivateKey)) {
    throw new Error('WEB_PUSH_VAPID_PUBLIC_KEY and WEB_PUSH_VAPID_PRIVATE_KEY must be set together.')
  }

  if (envPublicKey && envPrivateKey) {
    return {
      publicKey: envPublicKey,
      privateKey: envPrivateKey,
      subject: envSubject || defaultSubject,
    }
  }

  const stored = readStoredWebPushConfig(paths.webPushConfigPath)
  if (stored) {
    const storedSubject = stored.subject || ''
    const shouldMigrateDefaultSubject =
      storedSubject === '' ||
      (storedSubject === DEFAULT_WEB_PUSH_SUBJECT && defaultSubject !== DEFAULT_WEB_PUSH_SUBJECT)
    const resolved = {
      publicKey: stored.publicKey,
      privateKey: stored.privateKey,
      subject: envSubject || (shouldMigrateDefaultSubject ? defaultSubject : storedSubject),
    }
    if (resolved.subject !== stored.subject) {
      writeStoredWebPushConfig(paths.webPushConfigPath, resolved)
    }
    return resolved
  }

  const generated = generateWebPushConfig(envSubject || defaultSubject)
  writeStoredWebPushConfig(paths.webPushConfigPath, generated)
  return generated
}

function resolveDefaultWebPushSubject(options) {
  if (options?.withTunnel) {
    return DEFAULT_TUNNEL_WEB_PUSH_SUBJECT
  }
  return DEFAULT_WEB_PUSH_SUBJECT
}

function readStoredWebPushConfig(path) {
  if (!existsSync(path)) {
    return null
  }

  try {
    const parsed = JSON.parse(readFileSync(path, 'utf8'))
    const publicKey = typeof parsed.publicKey === 'string' ? parsed.publicKey.trim() : ''
    const privateKey = typeof parsed.privateKey === 'string' ? parsed.privateKey.trim() : ''
    const subject = typeof parsed.subject === 'string' ? parsed.subject.trim() : ''
    if (!publicKey || !privateKey) {
      return null
    }
    return {
      publicKey,
      privateKey,
      subject: subject || DEFAULT_WEB_PUSH_SUBJECT,
    }
  } catch {
    return null
  }
}

function writeStoredWebPushConfig(path, config) {
  writeFileSync(path, `${JSON.stringify(config, null, 2)}\n`)
}

function generateWebPushConfig(subject) {
  const { privateKey, publicKey } = generateKeyPairSync('ec', {
    namedCurve: 'prime256v1',
  })
  const privateJWK = privateKey.export({ format: 'jwk' })
  const publicJWK = publicKey.export({ format: 'jwk' })

  if (
    typeof privateJWK.d !== 'string' ||
    typeof publicJWK.x !== 'string' ||
    typeof publicJWK.y !== 'string'
  ) {
    throw new Error('failed to generate VAPID keys')
  }

  const x = Buffer.from(publicJWK.x, 'base64url')
  const y = Buffer.from(publicJWK.y, 'base64url')
  const publicKeyBytes = Buffer.concat([Buffer.from([0x04]), x, y])

  return {
    publicKey: publicKeyBytes.toString('base64url'),
    privateKey: privateJWK.d,
    subject,
  }
}

async function stopStack(options, { quiet }) {
  const paths = runtimePaths(options.runtimeDir)
  const stoppedServer = await killFromPidfile(paths.serverPidfile)
  const stoppedGateway = await killFromPidfile(paths.gatewayPidfile)
  const stoppedTunnel = await killFromPidfile(paths.tunnelPidfile)
  rmSync(paths.stackMetadataPath, { force: true })

  if (!quiet) {
    if (stoppedServer || stoppedGateway || stoppedTunnel) {
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
  const tunnelPid = readPidfile(paths.tunnelPidfile)
  const tunnelRunning = tunnelPid !== null && isPidAlive(tunnelPid)
  const apiHealthy = serverRunning ? await isHttpReady(`http://${options.apiHost}:${options.apiPort}/healthz`) : false
  const webHealthy = gatewayRunning ? await isHttpReady(`http://${options.webHost}:${options.webPort}`) : false

  console.log(`API server: ${serverRunning ? 'running' : 'stopped'}${apiHealthy ? ' (healthy)' : ''}`)
  console.log(`Web gateway: ${gatewayRunning ? 'running' : 'stopped'}${webHealthy ? ' (healthy)' : ''}`)
  if (tunnelPid !== null) {
    console.log(`Named tunnel: ${tunnelRunning ? 'running' : 'stopped'}`)
  }
  console.log(`Runtime dir: ${paths.runtimeDir}`)
  console.log(`API URL: http://${options.apiHost}:${options.apiPort}`)
  console.log(`Web URL: http://${options.webHost}:${options.webPort}`)
}

function buildBundledLaunchSpec() {
  return {
    serverCommand: resolveBinaryPath('agent-message-server'),
    serverArgs: [],
    gatewayCommand: process.execPath,
    gatewayArgs: [bundleGatewayPath],
    webDistDir: bundleWebDistDir,
  }
}

function buildDevLaunchSpec(paths) {
  ensureDevSourcesReady()

  if (!existsSync(join(sourceWebDir, 'node_modules'))) {
    runForeground('npm', ['ci'], { cwd: sourceWebDir })
  }
  runForeground('npm', ['run', 'build'], { cwd: sourceWebDir })

  const serverBinaryPath = join(paths.binDir, 'agent-message-server')
  runForeground('go', ['build', '-o', serverBinaryPath, '.'], { cwd: sourceServerDir })

  return {
    serverCommand: serverBinaryPath,
    serverArgs: [],
    gatewayCommand: process.execPath,
    gatewayArgs: [sourceGatewayPath],
    webDistDir: sourceWebDistDir,
  }
}

function ensureDevSourcesReady() {
  const requiredPaths = [sourceServerDir, sourceWebDir, sourceGatewayPath]
  for (const target of requiredPaths) {
    if (!existsSync(target)) {
      throw new Error(`development mode requires a local checkout with ${target}`)
    }
  }
}

function ensureTunnelTargetMatchesDefaults(options) {
  if (options.webHost !== DEFAULT_WEB_HOST || options.webPort !== DEFAULT_WEB_PORT) {
    throw new Error(
      `--with-tunnel requires the default web listener ${DEFAULT_WEB_HOST}:${DEFAULT_WEB_PORT} to match ${tunnelConfigPath}.`,
    )
  }
}

function delegateToBundledCli(args) {
  const cliBinary = resolveBinaryPath('agent-message-cli')
  const delegatedArgs = buildDelegatedCliArgs(args)
  const result = spawnSync(cliBinary, delegatedArgs, {
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

function buildDelegatedCliArgs(args) {
  if (shouldSkipLocalServerOverride(args) || hasExplicitServerURL(args)) {
    return args
  }

  const localServerURL = readLocalServerURL()
  if (!localServerURL) {
    return args
  }

  return ['--server-url', localServerURL, ...args]
}

function shouldSkipLocalServerOverride(args) {
  const [command] = args
  return command === undefined || command === 'config' || command === 'profile'
}

function hasExplicitServerURL(args) {
  return args.some((arg) => arg === '--server-url' || arg.startsWith('--server-url='))
}

function readLocalServerURL() {
  const paths = runtimePaths(join(os.homedir(), '.agent-message'))
  const serverPid = readPidfile(paths.serverPidfile)
  if (serverPid === null || !isPidAlive(serverPid)) {
    return null
  }

  try {
    const raw = readFileSync(paths.stackMetadataPath, 'utf8')
    const metadata = JSON.parse(raw)
    const apiHost = typeof metadata.apiHost === 'string' ? metadata.apiHost.trim() : ''
    const apiPort = metadata.apiPort
    if (!apiHost || !Number.isInteger(apiPort) || apiPort <= 0 || apiPort > 65535) {
      return null
    }
    return `http://${apiHost}:${apiPort}`
  } catch {
    return null
  }
}

function writeStackMetadata(path, options) {
  const metadata = {
    apiHost: options.apiHost,
    apiPort: options.apiPort,
    webHost: options.webHost,
    webPort: options.webPort,
  }
  writeFileSync(path, `${JSON.stringify(metadata, null, 2)}\n`)
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

function runForeground(command, args, { cwd }) {
  const result = spawnSync(command, args, {
    cwd,
    env: process.env,
    stdio: 'inherit',
  })

  if (result.error) {
    throw result.error
  }

  if (result.signal) {
    process.kill(process.pid, result.signal)
    return
  }

  if ((result.status ?? 1) !== 0) {
    throw new Error(`${command} ${args.join(' ')} failed with exit code ${result.status ?? 1}`)
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

async function waitForLogMessage(logFile, message, label) {
  for (let attempt = 0; attempt < STARTUP_ATTEMPTS; attempt += 1) {
    if (existsSync(logFile) && readFileSync(logFile, 'utf8').includes(message)) {
      return
    }
    await sleep(STARTUP_DELAY_MS)
  }
  throw new Error(`${label} did not become ready: ${message}`)
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
