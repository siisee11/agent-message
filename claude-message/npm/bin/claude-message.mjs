#!/usr/bin/env node

import { accessSync, constants, existsSync } from 'node:fs'
import { delimiter, dirname, join, resolve } from 'node:path'
import process from 'node:process'
import { spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const scriptDir = dirname(fileURLToPath(import.meta.url))
const packageRoot = resolve(scriptDir, '..', '..')
const runtimeBinDir = resolve(packageRoot, 'npm', 'runtime', 'bin')

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
