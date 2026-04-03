#!/usr/bin/env node

import { execFileSync } from 'node:child_process'
import { chmodSync, cpSync, existsSync, mkdirSync, rmSync } from 'node:fs'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const scriptDir = dirname(fileURLToPath(import.meta.url))
const packageDir = resolve(scriptDir, '..')
const runtimeDir = join(packageDir, 'npm', 'runtime')
const runtimeBinDir = join(runtimeDir, 'bin')

const targets = [
  { rustTarget: 'aarch64-apple-darwin', suffix: 'darwin-arm64' },
  { rustTarget: 'x86_64-apple-darwin', suffix: 'darwin-amd64' },
]

function run(command, args, options = {}) {
  execFileSync(command, args, {
    stdio: 'inherit',
    ...options,
  })
}

function installedTargets() {
  const output = execFileSync('rustup', ['target', 'list', '--installed'], {
    encoding: 'utf8',
  })
  return new Set(output.split(/\s+/).filter(Boolean))
}

rmSync(runtimeDir, { recursive: true, force: true })
mkdirSync(runtimeBinDir, { recursive: true })

const installed = installedTargets()

for (const target of targets) {
  if (!installed.has(target.rustTarget)) {
    run('rustup', ['target', 'add', target.rustTarget], {
      cwd: packageDir,
    })
  }

  run('cargo', ['build', '--locked', '--release', '--target', target.rustTarget], {
    cwd: packageDir,
  })

  const sourceBinary = join(packageDir, 'target', target.rustTarget, 'release', 'claude-message')
  if (!existsSync(sourceBinary)) {
    throw new Error(`expected built binary at ${sourceBinary}`)
  }

  const outputBinary = join(runtimeBinDir, `claude-message-${target.suffix}`)
  cpSync(sourceBinary, outputBinary)
  chmodSync(outputBinary, 0o755)
}

console.log(`Prepared npm bundle in ${runtimeDir}`)
