#!/usr/bin/env node

import { spawn, spawnSync } from 'node:child_process'
import { accessSync, constants, existsSync, readFileSync } from 'node:fs'
import { delimiter, dirname, join, resolve } from 'node:path'
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
    packages: ['agent-message@latest', 'codex-message@latest'],
    primaryPackage: 'codex-message',
  })
}

if (process.argv[2] === '--version') {
  printVersion(packageJsonPath)
}

function resolveBinaryPath() {
  if (process.platform !== 'darwin') {
    throw new Error(`unsupported platform: ${process.platform}. This package currently supports macOS only.`)
  }

  if (process.arch === 'arm64') {
    return join(runtimeBinDir, 'codex-message-darwin-arm64')
  }
  if (process.arch === 'x64') {
    return join(runtimeBinDir, 'codex-message-darwin-amd64')
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

const binaryPath = resolveBinaryPath()
if (!existsSync(binaryPath)) {
  console.error(`packaged codex-message binary is missing at ${binaryPath}. Reinstall the package or rebuild the npm bundle.`)
  process.exit(1)
}

requireCommand('agent-message', 'Install it first with `npm install -g agent-message`.')
requireCommand('codex', 'Install the Codex CLI before running codex-message.')

const child = spawnSync(binaryPath, process.argv.slice(2), {
  argv0: 'codex-message',
  stdio: 'inherit',
  cwd: process.cwd(),
  env: process.env,
})

if (child.error) {
  console.error(child.error.message)
  process.exit(1)
}

process.exit(child.status ?? 0)
