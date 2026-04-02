#!/usr/bin/env node

import { spawnSync } from 'node:child_process'
import { existsSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import process from 'node:process'
import { fileURLToPath } from 'node:url'

const scriptDir = dirname(fileURLToPath(import.meta.url))
const packageRoot = resolve(scriptDir, '..', '..')
const crateDir = resolve(packageRoot, 'codex-message')
const manifestPath = resolve(crateDir, 'Cargo.toml')
const binaryPath = resolve(crateDir, 'target', 'debug', process.platform === 'win32' ? 'codex-message.exe' : 'codex-message')

if (!existsSync(manifestPath)) {
  console.error(`codex-message sources were not found at ${crateDir}`)
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
