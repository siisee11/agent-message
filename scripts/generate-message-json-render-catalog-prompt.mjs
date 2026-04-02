#!/usr/bin/env node

import { mkdirSync, readFileSync, writeFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import process from 'node:process'
import { fileURLToPath } from 'node:url'

import { messageJsonRenderCatalog } from '../web/src/components/messageJsonRenderCatalog.shared.mjs'

const scriptDir = dirname(fileURLToPath(import.meta.url))
const rootDir = resolve(scriptDir, '..')
const outputPath = resolve(rootDir, 'server', 'api', 'catalog_prompt.txt')
const nextPrompt = `${messageJsonRenderCatalog.prompt().trim()}\n`

mkdirSync(dirname(outputPath), { recursive: true })

let currentPrompt = ''
try {
  currentPrompt = readFileSync(outputPath, 'utf8')
} catch {
  currentPrompt = ''
}

if (currentPrompt !== nextPrompt) {
  writeFileSync(outputPath, nextPrompt)
}

process.stdout.write(outputPath)
