import { createReadStream } from 'node:fs'
import { access, readFile, stat } from 'node:fs/promises'
import http from 'node:http'
import { basename, extname, join, normalize, resolve } from 'node:path'

const host = process.env.AGENT_GATEWAY_HOST ?? '127.0.0.1'
const port = Number(process.env.AGENT_GATEWAY_PORT ?? '8788')
const apiOrigin = process.env.AGENT_API_ORIGIN ?? 'http://127.0.0.1:18080'
const distDir = resolve(process.env.AGENT_WEB_DIST ?? join(process.cwd(), 'web', 'dist'))
const indexPath = join(distDir, 'index.html')

const contentTypes = new Map([
  ['.css', 'text/css; charset=utf-8'],
  ['.html', 'text/html; charset=utf-8'],
  ['.ico', 'image/x-icon'],
  ['.js', 'text/javascript; charset=utf-8'],
  ['.json', 'application/json; charset=utf-8'],
  ['.mjs', 'text/javascript; charset=utf-8'],
  ['.png', 'image/png'],
  ['.svg', 'image/svg+xml'],
  ['.txt', 'text/plain; charset=utf-8'],
  ['.webmanifest', 'application/manifest+json; charset=utf-8'],
  ['.woff2', 'font/woff2'],
])

function setForwardHeaders(headers, req) {
  headers.set('x-forwarded-for', req.socket.remoteAddress ?? '')
  headers.set('x-forwarded-host', req.headers.host ?? '')
  headers.set('x-forwarded-proto', 'https')
}

async function proxyRequest(req, res) {
  const targetURL = new URL(req.url ?? '/', apiOrigin)
  const headers = new Headers()

  for (const [key, value] of Object.entries(req.headers)) {
    if (value === undefined) {
      continue
    }
    if (Array.isArray(value)) {
      for (const item of value) {
        headers.append(key, item)
      }
      continue
    }
    headers.set(key, value)
  }

  setForwardHeaders(headers, req)

  const init = {
    method: req.method,
    headers,
    body: req.method === 'GET' || req.method === 'HEAD' ? undefined : req,
    duplex: 'half',
  }

  const upstream = await fetch(targetURL, init)

  res.writeHead(
    upstream.status,
    Object.fromEntries(upstream.headers.entries()),
  )

  if (!upstream.body) {
    res.end()
    return
  }

  for await (const chunk of upstream.body) {
    res.write(chunk)
  }
  res.end()
}

function resolveStaticPath(requestPath) {
  const decodedPath = decodeURIComponent(requestPath.split('?')[0])
  const normalizedPath = normalize(decodedPath).replace(/^(\.\.[/\\])+/, '')
  const trimmedPath = normalizedPath.replace(/^[/\\]+/, '')
  return join(distDir, trimmedPath)
}

async function fileExists(path) {
  try {
    await access(path)
    return true
  } catch {
    return false
  }
}

async function serveFile(res, path) {
  const fileStats = await stat(path)
  if (!fileStats.isFile()) {
    return false
  }

  const type = contentTypes.get(extname(path)) ?? 'application/octet-stream'
  const isServiceWorkerScript = basename(path) === 'sw.js'
  res.writeHead(200, {
    'cache-control':
      path === indexPath || isServiceWorkerScript ? 'no-cache' : 'public, max-age=31536000, immutable',
    'content-length': String(fileStats.size),
    'content-type': type,
  })
  createReadStream(path).pipe(res)
  return true
}

async function serveApp(req, res) {
  const requestPath = req.url ?? '/'
  let candidatePath = resolveStaticPath(requestPath)

  if (await fileExists(candidatePath)) {
    const served = await serveFile(res, candidatePath)
    if (served) {
      return
    }
  }

  candidatePath = indexPath
  res.writeHead(200, {
    'cache-control': 'no-cache',
    'content-type': 'text/html; charset=utf-8',
  })
  res.end(await readFile(candidatePath))
}

const server = http.createServer(async (req, res) => {
  try {
    const requestPath = req.url ?? '/'
    if (requestPath.startsWith('/api/') || requestPath === '/api' || requestPath.startsWith('/static/uploads/')) {
      await proxyRequest(req, res)
      return
    }

    await serveApp(req, res)
  } catch (error) {
    console.error('gateway request failed', error)
    if (!res.headersSent) {
      res.writeHead(502, { 'content-type': 'text/plain; charset=utf-8' })
    }
    res.end('Bad gateway')
  }
})

server.listen(port, host, () => {
  console.log(`agent gateway listening on http://${host}:${port}`)
  console.log(`serving dist from ${distDir}`)
  console.log(`proxying API to ${apiOrigin}`)
})
