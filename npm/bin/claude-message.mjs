#!/usr/bin/env node

import { spawnSync } from 'node:child_process'
import { existsSync, readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import process from 'node:process'
import { fileURLToPath } from 'node:url'

const scriptDir = dirname(fileURLToPath(import.meta.url))
const packageRoot = resolve(scriptDir, '..', '..')
const crateDir = resolve(packageRoot, 'claude-message')
const manifestPath = resolve(crateDir, 'Cargo.toml')
const binaryPath = resolve(crateDir, 'target', 'debug', process.platform === 'win32' ? 'claude-message.exe' : 'claude-message')
const upgradeCommand = 'upgrade'
const packageJsonPath = resolve(crateDir, 'package.json')

if (process.argv[2] === upgradeCommand) {
  runGlobalUpgrade(['agent-message@latest', 'claude-message@latest'])
}

if (process.argv[2] === '--version') {
  printVersion(packageJsonPath)
}

if (!existsSync(manifestPath)) {
  console.error(`claude-message sources were not found at ${crateDir}`)
  process.exit(1)
}

if (!existsSync(binaryPath)) {
  const build = spawnSync('cargo', ['build', '--manifest-path', manifestPath], {
    stdio: 'inherit',
    cwd: packageRoot,
  })
  if (build.status !== 0) {
    process.exit(build.status ?? 1)
  }
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

  process.exit(result.status ?? 1)
}

function printVersion(path) {
  const packageJson = JSON.parse(readFileSync(path, 'utf8'))
  console.log(`${packageJson.name} ${packageJson.version}`)
  process.exit(0)
}

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
