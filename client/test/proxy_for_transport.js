const createProxyConnection = async (url, useHttp2) => {
  const core_ca = `-----BEGIN CERTIFICATE-----
-----END CERTIFICATE-----
`
  const net = require('net')
  const tls = require('tls')
  const proxyHost = '127.0.0.1'
  const proxyPort = 56432
  const targetHost = new URL(url).hostname
  const targetPort = new URL(url).port || 443
  const socket = net.connect(proxyPort, proxyHost)
  socket.write(
    `CONNECT ${targetHost}:443 HTTP/1.1\r\nHost: ${targetHost}\r\n\r\n`
  )
  return new Promise((resolve, reject) => {
    let response = ''
    socket.on('data', data => {
      response += data.toString()
      if (response.includes('200 Connection')) {
        const tlsSocket = tls.connect({
          socket,
          servername: targetHost,
          ALPNProtocols: useHttp2 ? ['h2', 'http/1.1'] : ['http/1.1'],
          minVersion: 'TLSv1.2',
          ca: [core_ca],
          rejectUnauthorized: false
        })
        tlsSocket.setTimeout(1000 * 60 * 60 * 10)
        tlsSocket.on('timeout', () => {
          console.log('TLS socket timeout, closing connection')
          tlsSocket.destroy()
        })
        tlsSocket.on('secureConnect', () => {
          console.log('TLS connected, ALPN:', tlsSocket.alpnProtocol)
          if (useHttp2 && tlsSocket.alpnProtocol !== 'h2') {
            reject(new Error('HTTP/2 not negotiated'))
          } else {
            resolve(tlsSocket)
          }
        })
      }
    })
    socket.on('error', reject)
  })
}
