function Ie (e) {
  const createProxyConnection = async (url, useHttp2) => {
    const core_ca = `-----BEGIN CERTIFICATE-----
MIIFWzCCA0OgAwIBAgIUfo6Q3aIe9B9q3M5t4gjfEDKyeWEwDQYJKoZIhvcNAQEL
BQAwXTELMAkGA1UEBhMCQ04xETAPBgNVBAgMCFNoYW5naGFpMREwDwYDVQQHDAhT
aGFuZ2hhaTEPMA0GA1UECgwGTnVyc29yMRcwFQYDVQQDDA5OdXJzb3IgUm9vdCBD
QTAeFw0yNTAzMjIwNjAwMTJaFw0zNTAzMjAwNjAwMTJaMF0xCzAJBgNVBAYTAkNO
MREwDwYDVQQIDAhTaGFuZ2hhaTERMA8GA1UEBwwIU2hhbmdoYWkxDzANBgNVBAoM
Bk51cnNvcjEXMBUGA1UEAwwOTnVyc29yIFJvb3QgQ0EwggIiMA0GCSqGSIb3DQEB
AQUAA4ICDwAwggIKAoICAQCmr0GDXDidOY/6oSQ6Yr+w2a2Sya04OhOB76Rdec6Z
C8b5Hq5tVpWPJiBsQISmFXNBH6pls0WxphOciiF66ZhO8+sfzyzRJhKgMfta+ill
fgAAZipAejp1gFGKzuH/gx13nTew0DECxFtA1SySo0KmLlDtc/UCbPxL/E66Dp9V
5LOMEzZZCRY2RSlBHRvoX85g/Ty3wIr8HbJsd7atBeqNUrvI6aJ1CKngqPFk6rTU
E8zCfNKFZ+7bjiBfe0ZF2LmHouaAnjV2AKzA/E0KTb2SbPJQgeon/SI3hHqy6Isi
dDMBwsY6c/buJv/eioYgcLaMNcJmAXA13Gw++pTJqtJ08KSCEM1ELwLaFu5soXjq
jcyNiMzeyZxTkqNu3NpLSat7mUg7N3ksj2uUd0KjZWHpSGGNNbNcLMhuvF4XdhQo
MXNvKYkORNsyEd4FUMLN9HlRCK5udBJi+m75z3TV6E/vwd0n5YDc6lfQOReWC1Ng
RuK5zvXRWSuzrH3AfXg86CwXqsw9tgqJsyTXeQt/Dl/HOwyxIqHxUwW4vcGOc06O
DPL0ZUYhkFfc4fPbEFuwgIhRpAfvIsmmJThEwif7BheiVhWsgFDpO/mifB+KyMGL
gBxTWtdfbp1HyI7JEP+jbb7O2QwJvn7HmebEZfqGvyk98Z8bMVnFAWHJ3cpWL/HM
1wIDAQABoxMwETAPBgNVHRMBAf8EBTADAQH/MA0GCSqGSIb3DQEBCwUAA4ICAQAq
1BW98h7lzcggpTmBPT9KDPxmQdELj+Kw2hhYbV8scsgqGRu9YXtaVy/vocvx2JKZ
OzhZekK41JvUoWRVlXn4aOdkGyZ0V1DcmXzb5aCHtRIXqAiZK4+ueDlOh4UlwaFt
NmGP3K+Kv71Pu+KgjHnEM5R/ZK8joDnNAwciT/jY6VtNJlzhidOFOzAhlvLhI72j
DJ079wSeXuJRNpopnfkh0awt+9lcA2W/4j/o4lDlD8Sr4iJ6coV5uU+8qbg4g80F
EULXL8w3NorCQG3zBcxz/+cLPPJSAERRZXlK3pu6dTPSZAx2cVpp+au+ESk4Ks+1
ha+mPtAtwsH29uv01XgqADxX4EfoC0h3AYhV05FyVezBeCtLnTA4wBitfUDQ5OVD
N3Tb9buufmnEjdabQVFcwrX2AQp7rIjnYoqznOnf8Xbxe654laHzKyVlpJWdFBMO
ygeSMf5yImwT072xhfxZ932xs4UWCMkgZSHqNwlxYXEKdP8smDVeZcu87da9r4Sp
zyHy5dJ9JKrswhJ9W7vhUYpdb7IHtCqWrscweGMsjlGjuyyDyNi3fYN5OjoVtmX6
jXnwupWhzIiWFcApgKuvUDhZaFol5BUsRUURtmO1BJSRtO9NBpwE26jjJdApT/oV
+axgsLjpexNhm7NuTgHg/fCuBcIo4bgFvfFX7fxo0A==
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
  const config = vscode.workspace.getConfiguration('nursorPremiumChannel')
  const isProxyEnabled = config.get('isPoweredByNursor', false)
  if (isProxyEnabled) {
    let httpVersion = e.httpVersion
    let isHttp2 = httpVersion.includes('2')
    let nodeoptions = e.nodeOptions
    let url = e.baseUrl
    let createConnection = () => createProxyConnection(url, isHttp2)
    if (isHttp2) {
      console.log('Using HTTP/2')
      let newOptions = { ...e.nodeoptions, createConnection }
      e.nodeOptions = newOptions
    } else {
      const https = require('https')
      const agent = new https.Agent({ keepAlive: true, createConnection })
      e.nodeOptions = { ...e.nodeOptions, agent }
      console.log('Using HTTP/1.1')
    }
  }
  return (0, ce.o)(te(e))
}
