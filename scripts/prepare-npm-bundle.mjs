#!/usr/bin/env node

import { execFileSync } from 'node:child_process'
import { cpSync, existsSync, mkdirSync, rmSync } from 'node:fs'
import { dirname, join, resolve } from 'node:path'
import process from 'node:process'
import { fileURLToPath } from 'node:url'

const scriptDir = dirname(fileURLToPath(import.meta.url))
const rootDir = resolve(scriptDir, '..')
const runtimeDir = join(rootDir, 'npm', 'runtime')
const runtimeBinDir = join(runtimeDir, 'bin')
const webDistDir = join(runtimeDir, 'web-dist')

function run(command, args, options = {}) {
  execFileSync(command, args, {
    stdio: 'inherit',
    ...options,
  })
}

function buildGoBinary(sourceDir, outputPath, goarch) {
  run(
    'go',
    ['build', '-o', outputPath, '.'],
    {
      cwd: join(rootDir, sourceDir),
      env: {
        ...process.env,
        CGO_ENABLED: '0',
        GOOS: 'darwin',
        GOARCH: goarch,
      },
    },
  )
}

rmSync(runtimeDir, { recursive: true, force: true })
mkdirSync(runtimeBinDir, { recursive: true })

if (!existsSync(join(rootDir, 'web', 'node_modules'))) {
  run('npm', ['ci'], {
    cwd: join(rootDir, 'web'),
  })
}

run('node', ['./scripts/generate-message-json-render-catalog-prompt.mjs'], {
  cwd: rootDir,
})

run('npm', ['run', 'build'], {
  cwd: join(rootDir, 'web'),
  env: {
    ...process.env,
    VITE_AGENT_MESSAGE_SELFHOST: '1',
  },
})

cpSync(join(rootDir, 'web', 'dist'), webDistDir, { recursive: true })
cpSync(join(rootDir, 'deploy', 'agent_gateway.mjs'), join(runtimeDir, 'agent_gateway.mjs'))

buildGoBinary('server', join(runtimeBinDir, 'agent-message-server-darwin-arm64'), 'arm64')
buildGoBinary('server', join(runtimeBinDir, 'agent-message-server-darwin-amd64'), 'amd64')
buildGoBinary('cli', join(runtimeBinDir, 'agent-message-cli-darwin-arm64'), 'arm64')
buildGoBinary('cli', join(runtimeBinDir, 'agent-message-cli-darwin-amd64'), 'amd64')

console.log(`Prepared npm bundle in ${runtimeDir}`)
