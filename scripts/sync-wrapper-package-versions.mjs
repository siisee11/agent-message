#!/usr/bin/env node

import { readFileSync, writeFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const repoRoot = resolve(fileURLToPath(new URL('.', import.meta.url)), '..')
const rootPackagePath = resolve(repoRoot, 'package.json')

const wrappers = [
  {
    name: 'codex-message',
    packageJsonPath: resolve(repoRoot, 'codex-message', 'package.json'),
    cargoTomlPath: resolve(repoRoot, 'codex-message', 'Cargo.toml'),
    cargoLockPath: resolve(repoRoot, 'codex-message', 'Cargo.lock'),
  },
  {
    name: 'claude-message',
    packageJsonPath: resolve(repoRoot, 'claude-message', 'package.json'),
    cargoTomlPath: resolve(repoRoot, 'claude-message', 'Cargo.toml'),
    cargoLockPath: resolve(repoRoot, 'claude-message', 'Cargo.lock'),
  },
]

const rootPackage = JSON.parse(readFileSync(rootPackagePath, 'utf8'))
const sourceVersion = String(rootPackage.version ?? '').trim()

if (!sourceVersion) {
  throw new Error(`missing version in ${rootPackagePath}`)
}

for (const wrapper of wrappers) {
  const packageJson = JSON.parse(readFileSync(wrapper.packageJsonPath, 'utf8'))
  if (packageJson.version !== sourceVersion) {
    packageJson.version = sourceVersion
    writeFileSync(wrapper.packageJsonPath, `${JSON.stringify(packageJson, null, 2)}\n`)
  }

  const cargoToml = readFileSync(wrapper.cargoTomlPath, 'utf8')
  const pattern = /(\[package\][\s\S]*?^version = ")([^"]+)(")/m
  if (!pattern.test(cargoToml)) {
    throw new Error(`failed to find package version in ${wrapper.cargoTomlPath}`)
  }

  const nextCargoToml = cargoToml.replace(
    pattern,
    `$1${sourceVersion}$3`,
  )

  if (nextCargoToml !== cargoToml) {
    writeFileSync(wrapper.cargoTomlPath, nextCargoToml)
  }

  const cargoLock = readFileSync(wrapper.cargoLockPath, 'utf8')
  const lockPattern = new RegExp(`(name = "${wrapper.name}"\\nversion = ")([^"]+)(")`, 'm')
  if (!lockPattern.test(cargoLock)) {
    throw new Error(`failed to find package version in ${wrapper.cargoLockPath}`)
  }

  const nextCargoLock = cargoLock.replace(
    lockPattern,
    `$1${sourceVersion}$3`,
  )

  if (nextCargoLock !== cargoLock) {
    writeFileSync(wrapper.cargoLockPath, nextCargoLock)
  }
}
