const ORIGIN = 'http://agent.namjaeyoun.com.wordbricks.ai'

export default {
  async fetch(request) {
    const incomingURL = new URL(request.url)
    const upstreamURL = new URL(incomingURL.pathname + incomingURL.search, ORIGIN)
    const proxyRequest = new Request(upstreamURL.toString(), request)
    return fetch(proxyRequest, {
      redirect: 'manual',
    })
  },
}
