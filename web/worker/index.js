const API_ROUTE_PREFIXES = ['/api/', '/static/uploads/']
const API_ROUTE_EXACT_MATCHES = new Set(['/api', '/static/uploads'])

export default {
  async fetch(request, env) {
    const url = new URL(request.url)

    if (shouldProxyToAPI(url.pathname)) {
      return proxyAPIRequest(request, env)
    }

    return env.ASSETS.fetch(request)
  },
}

function shouldProxyToAPI(pathname) {
  return API_ROUTE_EXACT_MATCHES.has(pathname) || API_ROUTE_PREFIXES.some((prefix) => pathname.startsWith(prefix))
}

function normalizeOrigin(value) {
  if (typeof value !== 'string') {
    return ''
  }

  return value.trim().replace(/\/+$/, '')
}

async function proxyAPIRequest(request, env) {
  const apiOrigin = normalizeOrigin(env.API_ORIGIN)
  const requestURL = new URL(request.url)

  if (apiOrigin === '') {
    if (requestURL.pathname === '/api' || requestURL.pathname.startsWith('/api/')) {
      return Response.json(
        {
          error: 'agent-message cloud service is not configured yet',
        },
        {
          status: 503,
          headers: {
            'cache-control': 'no-store',
          },
        },
      )
    }

    return new Response('Not found', {
      status: 404,
      headers: {
        'cache-control': 'no-store',
      },
    })
  }

  const upstreamURL = new URL(request.url)
  const apiOriginURL = new URL(apiOrigin)
  upstreamURL.protocol = apiOriginURL.protocol
  upstreamURL.host = apiOriginURL.host
  upstreamURL.username = ''
  upstreamURL.password = ''

  return fetch(new Request(upstreamURL, request))
}
