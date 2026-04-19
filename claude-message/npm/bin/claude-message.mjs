#!/usr/bin/env node

import { spawn, spawnSync } from 'node:child_process'
import { accessSync, closeSync, constants, existsSync, mkdirSync, openSync, readFileSync, readdirSync, rmSync, writeFileSync } from 'node:fs'
import { delimiter, dirname, join, resolve } from 'node:path'
import { homedir } from 'node:os'
import process from 'node:process'
import { fileURLToPath } from 'node:url'

const upgradeSpinner = {
  frames: ['⠃', '⠉', '⠘', '⠰', '⢠', '⣀', '⡄', '⠆'],
  interval: 100,
}

const scriptDir = dirname(fileURLToPath(import.meta.url))
const packageRoot = resolve(scriptDir, '..', '..')
const runtimeBinDir = resolve(packageRoot, 'npm', 'runtime', 'bin')
const upgradeCommand = 'upgrade'
const packageJsonPath = resolve(packageRoot, 'package.json')

if (process.argv[2] === upgradeCommand) {
  await runGlobalUpgrade({
    packages: ['agent-message@latest', 'claude-message@latest'],
    primaryPackage: 'claude-message',
  })
}

if (process.argv[2] === '--version') {
  printVersion(packageJsonPath)
}

if (process.argv[2] === 'list' || process.argv[2] === 'ls') {
  listBackgroundSessions('claude-message', process.argv.slice(3))
  process.exit(0)
}

if (process.argv[2] === 'kill' || process.argv[2] === 'stop') {
  const exitCode = await killBackgroundSessions('claude-message', process.argv.slice(3))
  process.exit(exitCode)
}

if (process.argv.includes('--bg')) {
  const backgroundArgs = process.argv.slice(2).filter((arg) => arg !== '--bg')
  if (backgroundArgs.includes('--help') || backgroundArgs.includes('-h')) {
    printBackgroundHelp('claude-message')
    process.exit(0)
  }
  const binaryPath = resolveBinaryPath()
  requireCommand('agent-message', 'Install it first with `npm install -g agent-message`.')
  requireCommand('claude', 'Install the Claude CLI before running claude-message.')
  runInBackground({
    appName: 'claude-message',
    binaryPath,
    forwardedArgs: backgroundArgs,
    argv0: 'claude-message',
  })
}

function resolveBinaryPath() {
  if (process.platform !== 'darwin') {
    throw new Error(`unsupported platform: ${process.platform}. This package currently supports macOS only.`)
  }

  if (process.arch === 'arm64') {
    return join(runtimeBinDir, 'claude-message-darwin-arm64')
  }
  if (process.arch === 'x64') {
    return join(runtimeBinDir, 'claude-message-darwin-amd64')
  }

  throw new Error(`unsupported architecture: ${process.arch}. Expected arm64 or x64 on macOS.`)
}

function findOnPath(command) {
  const pathValue = process.env.PATH ?? ''
  for (const entry of pathValue.split(delimiter)) {
    if (!entry) {
      continue
    }
    const candidate = join(entry, command)
    try {
      accessSync(candidate, constants.X_OK)
      return candidate
    } catch {}
  }
  return null
}

function requireCommand(command, installHint) {
  if (findOnPath(command)) {
    return
  }
  console.error(`${command} was not found on PATH. ${installHint}`)
  process.exit(1)
}

async function runGlobalUpgrade({ packages, primaryPackage }) {
  const npmCommand = process.platform === 'win32' ? 'npm.cmd' : 'npm'
  const currentVersion = readInstalledPackageVersion(npmCommand, primaryPackage)
  const latestVersion = readLatestPackageVersion(npmCommand, primaryPackage)

  printUpgradeHeader()
  printUpgradeLine('●', 'Using method: npm')
  printDivider()
  printUpgradeLine('●', `From ${formatVersion(currentVersion)} → ${formatVersion(latestVersion)}`)
  printDivider()
  const spinner = createStepSpinner('Upgrading packages')
  const result = await runCommand(npmCommand, ['install', '-g', ...packages], {
    cwd: process.cwd(),
    env: process.env,
  })

  if (result.error) {
    spinner.fail('Upgrade failed')
    printUpgradeFooter()
    console.error(result.error.message)
    process.exit(1)
  }

  if (result.status === 0) {
    spinner.succeed('Upgrade complete')
    printUpgradeFooter()
    process.exit(0)
  }

  spinner.fail('Upgrade failed')
  printUpgradeFooter()
  relayUpgradeFailureOutput(result)
  process.exit(result.status ?? 1)
}

function readInstalledPackageVersion(npmCommand, packageName) {
  const result = spawnSync(npmCommand, ['list', '-g', '--depth=0', '--json', packageName], {
    cwd: process.cwd(),
    env: process.env,
    encoding: 'utf8',
  })

  if (result.error || result.status !== 0 || !result.stdout) {
    return null
  }

  try {
    const parsed = JSON.parse(result.stdout)
    const dependencies = parsed.dependencies ?? {}
    return dependencies[packageName]?.version ?? null
  } catch {
    return null
  }
}

function readLatestPackageVersion(npmCommand, packageName) {
  const result = spawnSync(npmCommand, ['view', packageName, 'version', '--json'], {
    cwd: process.cwd(),
    env: process.env,
    encoding: 'utf8',
  })

  if (result.error || result.status !== 0 || !result.stdout) {
    return null
  }

  try {
    const parsed = JSON.parse(result.stdout)
    if (typeof parsed === 'string' && parsed.length > 0) {
      return parsed
    }
  } catch {
    const trimmed = result.stdout.trim()
    if (trimmed.length > 0) {
      return trimmed
    }
  }

  return null
}

function printUpgradeHeader() {
  console.log('┌  Upgrade')
  console.log('│')
}

function printDivider() {
  console.log('│')
}

function printUpgradeLine(marker, message) {
  console.log(`${marker}  ${message}`)
}

function printUpgradeFooter() {
  console.log('│')
  console.log('└  Done')
}

function formatVersion(version) {
  return version ?? 'unknown'
}

function createStepSpinner(message) {
  const spinner = upgradeSpinner
  const supportsAnimation = Boolean(process.stdout.isTTY && spinner?.frames?.length)
  let frameIndex = 0
  let timer = null

  if (supportsAnimation) {
    render()
    timer = setInterval(render, spinner.interval)
  } else {
    printUpgradeLine('●', message)
  }

  return {
    succeed(finalMessage) {
      stop('◇', finalMessage)
    },
    fail(finalMessage) {
      stop('◆', finalMessage)
    },
  }

  function render() {
    const frame = spinner.frames[frameIndex % spinner.frames.length]
    frameIndex += 1
    process.stdout.write(`\r\x1b[2K●  ${frame} ${message}`)
  }

  function stop(marker, finalMessage) {
    if (timer) {
      clearInterval(timer)
      timer = null
    }
    if (supportsAnimation) {
      process.stdout.write(`\r\x1b[2K${marker}  ${finalMessage}\n`)
      return
    }
    printUpgradeLine(marker, finalMessage)
  }
}

function runCommand(command, args, options) {
  return new Promise((resolvePromise) => {
    const child = spawn(command, args, {
      ...options,
      stdio: ['ignore', 'pipe', 'pipe'],
    })

    let stdout = ''
    let stderr = ''

    child.stdout?.setEncoding('utf8')
    child.stderr?.setEncoding('utf8')
    child.stdout?.on('data', (chunk) => {
      stdout += chunk
    })
    child.stderr?.on('data', (chunk) => {
      stderr += chunk
    })
    child.on('error', (error) => {
      resolvePromise({ error, status: null, stdout, stderr })
    })
    child.on('close', (status) => {
      resolvePromise({ error: null, status, stdout, stderr })
    })
  })
}

function relayUpgradeFailureOutput(result) {
  const stderr = result.stderr?.trim()
  const stdout = result.stdout?.trim()
  if (stderr) {
    console.error(stderr)
    return
  }
  if (stdout) {
    console.error(stdout)
  }
}

function printVersion(path) {
  const packageJson = JSON.parse(readFileSync(path, 'utf8'))
  console.log(`${packageJson.name} ${packageJson.version}`)
  process.exit(0)
}

function printBackgroundHelp(appName) {
  console.log(`Run ${appName} detached in the background`)
  console.log('')
  console.log(`Usage: ${appName} --bg [wrapper flags...]`)
  console.log('')
  console.log('Examples:')
  console.log(`  ${appName} --bg --model sonnet --permission-mode accept-edits`)
  console.log(`  ${appName} --bg --to alice --model sonnet --cwd /path/to/worktree`)
  console.log('')
  console.log('All flags except `--bg` are forwarded to the wrapper binary.')
  console.log('')
  console.log('Session commands:')
  console.log(`  ${appName} list [--all] [--json]`)
  console.log(`  ${appName} kill <session-id|pid|all>`)
}

function runInBackground({ appName, binaryPath, forwardedArgs, argv0 }) {
  if (!existsSync(binaryPath)) {
    console.error(`packaged ${appName} binary is missing at ${binaryPath}. Reinstall the package or rebuild the npm bundle.`)
    process.exit(1)
  }

  const wrapperDir = join(homedir(), '.agent-message', 'wrappers', appName)
  mkdirSync(wrapperDir, { recursive: true })

  const sessionId = `${new Date().toISOString().replace(/[:.]/g, '-')}-${process.pid}`
  const logFile = join(wrapperDir, `${sessionId}.log`)
  const metadataFile = join(wrapperDir, `${sessionId}.json`)
  const stdoutFd = openSync(logFile, 'a')
  const stderrFd = openSync(logFile, 'a')

  try {
    const child = spawn(binaryPath, forwardedArgs, {
      argv0,
      cwd: process.cwd(),
      detached: true,
      env: process.env,
      stdio: ['ignore', stdoutFd, stderrFd],
    })
    child.unref()

    if (!Number.isInteger(child.pid) || child.pid <= 0) {
      throw new Error(`failed to launch background process: ${binaryPath}`)
    }

    writeFileSync(
      metadataFile,
      `${JSON.stringify(
        {
          appName,
          pid: child.pid,
          cwd: process.cwd(),
          command: binaryPath,
          args: forwardedArgs,
          logFile,
          startedAt: new Date().toISOString(),
        },
        null,
        2,
      )}\n`,
    )

    console.log(`Started ${appName} in background.`)
    console.log(`PID: ${child.pid}`)
    console.log(`Log: ${logFile}`)
    console.log(`Metadata: ${metadataFile}`)
    process.exit(0)
  } finally {
    closeSync(stdoutFd)
    closeSync(stderrFd)
  }
}

function listBackgroundSessions(appName, args) {
  if (args.includes('--help') || args.includes('-h')) {
    printListUsage(appName)
    return
  }

  const includeAll = args.includes('--all') || args.includes('-a')
  const asJson = args.includes('--json')
  const sessions = readBackgroundSessions(appName)
  const visibleSessions = includeAll ? sessions : sessions.filter((session) => session.alive)

  if (asJson) {
    console.log(JSON.stringify(visibleSessions, null, 2))
    return
  }

  if (visibleSessions.length === 0) {
    const staleCount = sessions.filter((session) => !session.alive).length
    console.log(`No running ${appName} background sessions.`)
    if (!includeAll && staleCount > 0) {
      console.log(`Use \`${appName} list --all\` to show ${staleCount} stale session metadata file(s).`)
    }
    return
  }

  printSessionTable(visibleSessions)
}

async function killBackgroundSessions(appName, args) {
  if (args.includes('--help') || args.includes('-h')) {
    printKillUsage(appName)
    return 0
  }

  const targets = args.filter((arg) => !arg.startsWith('-'))
  const killAll = args.includes('--all') || args.includes('-a') || targets.includes('all')
  if (!killAll && targets.length === 0) {
    printKillUsage(appName)
    return 1
  }

  const sessions = readBackgroundSessions(appName)
  const matches = killAll
    ? sessions.filter((session) => session.alive)
    : uniqueSessions(
        targets.flatMap((target) =>
          sessions.filter((session) => sessionMatchesTarget(session, target)),
        ),
      )

  if (matches.length === 0) {
    console.error(killAll ? `No running ${appName} background sessions.` : 'No matching session found.')
    return killAll ? 0 : 1
  }

  let exitCode = 0
  for (const session of matches) {
    if (!session.alive) {
      rmSync(session.metadataFile, { force: true })
      console.log(`Removed stale ${appName} session ${session.sessionId} (pid ${session.pid}).`)
      continue
    }

    const killed = await terminateProcessTree(session.pid)
    if (killed) {
      rmSync(session.metadataFile, { force: true })
      console.log(`Killed ${appName} session ${session.sessionId} (pid ${session.pid}).`)
    } else {
      console.error(`Failed to kill ${appName} session ${session.sessionId} (pid ${session.pid}).`)
      exitCode = 1
    }
  }

  return exitCode
}

function printListUsage(appName) {
  console.log(`Usage: ${appName} list [--all] [--json]`)
  console.log('')
  console.log('Lists running background sessions created with --bg.')
}

function printKillUsage(appName) {
  console.log(`Usage: ${appName} kill <session-id|pid|all>`)
  console.log('')
  console.log('Examples:')
  console.log(`  ${appName} kill 12345`)
  console.log(`  ${appName} kill 2026-04-19T10-30-00-000Z-12345`)
  console.log(`  ${appName} kill all`)
}

function readBackgroundSessions(appName) {
  const dir = join(homedir(), '.agent-message', 'wrappers', appName)
  if (!existsSync(dir)) {
    return []
  }

  return readdirSync(dir)
    .filter((name) => name.endsWith('.json'))
    .flatMap((name) => {
      const metadataFile = join(dir, name)
      try {
        const metadata = JSON.parse(readFileSync(metadataFile, 'utf8'))
        const pid = Number(metadata.pid)
        if (!Number.isInteger(pid) || pid <= 0) {
          return []
        }
        return [
          {
            sessionId: name.slice(0, -'.json'.length),
            appName: metadata.appName ?? appName,
            pid,
            cwd: metadata.cwd ?? '',
            command: metadata.command ?? '',
            args: Array.isArray(metadata.args) ? metadata.args : [],
            logFile: metadata.logFile ?? '',
            startedAt: metadata.startedAt ?? '',
            metadataFile,
            alive: isPidAlive(pid),
          },
        ]
      } catch {
        return []
      }
    })
    .sort((left, right) => String(left.startedAt).localeCompare(String(right.startedAt)))
}

function printSessionTable(sessions) {
  const rows = [
    ['SESSION', 'PID', 'STATUS', 'STARTED', 'CWD', 'LOG'],
    ...sessions.map((session) => [
      session.sessionId,
      String(session.pid),
      session.alive ? 'running' : 'stale',
      session.startedAt || '-',
      session.cwd || '-',
      session.logFile || '-',
    ]),
  ]
  const widths = rows[0].map((_, columnIndex) =>
    Math.max(...rows.map((row) => row[columnIndex].length)),
  )
  for (const row of rows) {
    console.log(row.map((cell, index) => cell.padEnd(widths[index])).join('  ').trimEnd())
  }
}

function uniqueSessions(sessions) {
  const seen = new Set()
  return sessions.filter((session) => {
    if (seen.has(session.metadataFile)) {
      return false
    }
    seen.add(session.metadataFile)
    return true
  })
}

function sessionMatchesTarget(session, target) {
  const normalized = String(target).trim()
  return (
    normalized.length > 0 &&
    (String(session.pid) === normalized ||
      session.sessionId === normalized ||
      session.sessionId.startsWith(normalized))
  )
}

function isPidAlive(pid) {
  try {
    process.kill(pid, 0)
  } catch (error) {
    return error?.code === 'EPERM'
  }
  return !isZombiePid(pid)
}

function isZombiePid(pid) {
  if (process.platform === 'win32') {
    return false
  }
  const result = spawnSync('ps', ['-o', 'stat=', '-p', String(pid)], {
    encoding: 'utf8',
    stdio: ['ignore', 'pipe', 'ignore'],
  })
  const state = result.stdout.trim()
  return state.startsWith('Z')
}

async function terminateProcessTree(pid) {
  if (!isPidAlive(pid)) {
    return true
  }

  sendSignal(pid, 'SIGTERM')
  if (await waitForProcessExit(pid, 3000)) {
    return true
  }

  sendSignal(pid, 'SIGKILL')
  return waitForProcessExit(pid, 3000)
}

function sendSignal(pid, signal) {
  if (process.platform !== 'win32') {
    try {
      process.kill(-pid, signal)
      return
    } catch {}
  }

  try {
    process.kill(pid, signal)
  } catch {}
}

async function waitForProcessExit(pid, timeoutMs) {
  const startedAt = Date.now()
  while (Date.now() - startedAt < timeoutMs) {
    if (!isPidAlive(pid)) {
      return true
    }
    await sleep(100)
  }
  return !isPidAlive(pid)
}

function sleep(ms) {
  return new Promise((resolvePromise) => {
    setTimeout(resolvePromise, ms)
  })
}

const binaryPath = resolveBinaryPath()
if (!existsSync(binaryPath)) {
  console.error(`packaged claude-message binary is missing at ${binaryPath}. Reinstall the package or rebuild the npm bundle.`)
  process.exit(1)
}

requireCommand('agent-message', 'Install it first with `npm install -g agent-message`.')
requireCommand('claude', 'Install the Claude CLI before running claude-message.')

const child = spawnSync(binaryPath, process.argv.slice(2), {
  stdio: 'inherit',
  cwd: process.cwd(),
  env: process.env,
})

if (child.error) {
  console.error(child.error.message)
  process.exit(1)
}

process.exit(child.status ?? 0)
