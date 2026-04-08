#!/usr/bin/env node

import { accessSync, constants, existsSync, readFileSync } from 'node:fs'
import { delimiter, dirname, join, resolve } from 'node:path'
import process from 'node:process'
import { spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const scriptDir = dirname(fileURLToPath(import.meta.url))
const packageRoot = resolve(scriptDir, '..', '..')
const runtimeBinDir = resolve(packageRoot, 'npm', 'runtime', 'bin')
const upgradeCommand = 'upgrade'
const packageJsonPath = resolve(packageRoot, 'package.json')

if (process.argv[2] === upgradeCommand) {
  runGlobalUpgrade(['agent-message@latest', 'codex-message@latest'])
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

function runGlobalUpgrade(packages) {
  const npmCommand = process.platform === 'win32' ? 'npm.cmd' : 'npm'
  const result = spawnSync(npmCommand, ['install', '-g', ...packages], {
    stdio: 'inherit',
    cwd: process.cwd(),
    env: process.env,
  })

  if (result.error) {
    console.error(result.error.message)
    process.exit(1)
  }

  if (result.status === 0) {
    const installedVersions = readInstalledPackageVersions(npmCommand, packages)
    console.log('\ncodex-message is ready with the latest updates.')
    if (installedVersions.length > 0) {
      console.log(`Installed versions: ${installedVersions.join(', ')}`)
    }
    process.exit(0)
  }

  process.exit(result.status ?? 1)
}

function readInstalledPackageVersions(npmCommand, packages) {
  const packageNames = packages.map(parsePackageName)
  const result = spawnSync(npmCommand, ['list', '-g', '--depth=0', '--json', ...packageNames], {
    cwd: process.cwd(),
    env: process.env,
    encoding: 'utf8',
  })

  if (result.error || result.status !== 0 || !result.stdout) {
    return []
  }

  try {
    const parsed = JSON.parse(result.stdout)
    const dependencies = parsed.dependencies ?? {}
    return packageNames.flatMap((name) => {
      const version = dependencies[name]?.version
      return version ? [`${name} ${version}`] : []
    })
  } catch {
    return []
  }
}

function parsePackageName(specifier) {
  const versionSeparator = specifier.lastIndexOf('@')
  if (versionSeparator > 0) {
    return specifier.slice(0, versionSeparator)
  }
  return specifier
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
