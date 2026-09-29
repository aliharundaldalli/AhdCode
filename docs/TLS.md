# TLS standard module

[English] · [Türkçe](TLS_TR.md)

[Back to README](../README.md) · [Modules](MODULES.md) · [DNS](DNS.md) · [Time](TIME.md) · [Errors](ERRORS.md)

`TLS` (v2.6.0) **inspects** a TLS endpoint: it connects, performs a TLS
handshake, reads the certificate the server presents, verifies it, and
reports what it found — whether the certificate is valid for the host, who
issued it, which names it covers, when it expires, and which TLS version was
negotiated. It never runs `openssl` or any other tool, and it is always
bounded by a timeout. Import it explicitly:

```ahd
bring TLS
from TLS bring (TLSInfo, TLSError)
```

The canonical module identity is `builtin:TLS`; a sibling `TLS.ahd` cannot
shadow it. (Since v2.6.0 `TLS` is a standard module name: a local `TLS.ahd`
is no longer what `bring TLS` loads — rename such a file.)

## Surface

```text
TLS.inspect(host: String, port: Int = 443, timeoutSeconds: Int = 10) -> TLSInfo

TLSInfo.host()               -> String          // the normalized host
TLSInfo.port()               -> Int
TLSInfo.valid()              -> Bool            // true only if verification succeeded
TLSInfo.verificationStatus() -> String          // see below
TLSInfo.subject()            -> String          // e.g. "CN=example.com,O=Example"
TLSInfo.issuer()             -> String
TLSInfo.dnsNames()           -> List<String>    // the certificate's DNS names, in certificate order
TLSInfo.notBefore()          -> DateTime        // UTC
TLSInfo.notAfter()           -> DateTime        // UTC
TLSInfo.protocol()           -> String          // "TLS 1.3", "TLS 1.2", ...
TLSInfo.cipherSuite()        -> String          // e.g. "TLS_AES_128_GCM_SHA256"

TLSError  (derives from Error)
```

`TLSInfo` is an opaque value produced only by `TLS.inspect`. The dates are
the [Time](TIME.md) module's `DateTime`, in UTC; `bring TLS` makes them
usable without `bring Time`, but naming the type (`x: DateTime := ...`) or
calling `Time` functions still needs `bring Time` as usual.

```ahd
info: TLSInfo := TLS.inspect(host: "example.com", port: 443, timeoutSeconds: 5)
write(str(info.valid()) + " " + info.verificationStatus())
write(info.issuer())
write(info.notAfter().toISO())
```

## Inspection is not trust

`TLS.inspect` must report on broken certificates too — an expired or
mismatched certificate is exactly what a monitor needs to see. Normal TLS
clients stop at such a certificate and learn nothing more, so inspection
works in two separate steps:

1. **Read.** It connects to `host:port`, sends `host` as the SNI server name
   (no SNI is sent for an IP address), completes the handshake with
   automatic verification switched off, records the presented certificate
   chain and the negotiated protocol, and closes the connection. No
   application data is sent or received and nothing from the server is
   trusted or used.
2. **Verify.** It then verifies the leaf certificate explicitly against the
   operating system's trusted roots, using the intermediates the server
   presented, the requested host name, and the current time.

`valid()` is `true` only when that explicit verification succeeds. An
invalid certificate is never reported as valid and is never "accepted" — it
is only described. There is no `insecure` option.

`verificationStatus()` names the outcome, deciding in this fixed order and
reporting the first problem found:

| Status | Meaning |
|---|---|
| `valid` | verification succeeded; `valid()` is `true` |
| `expired` | the current time is after `notAfter()` |
| `notYetValid` | the current time is before `notBefore()` |
| `hostnameMismatch` | the certificate does not cover the requested host |
| `untrustedIssuer` | the chain does not lead to a trusted root (self-signed, private CA, missing intermediate) |
| `invalid` | any other verification failure (for example a certificate the platform verifier rejects as not standards compliant) |

For every status the metadata — subject, issuer, names, dates, protocol — is
still available.

## Host, SNI, and IP addresses

`host` is validated exactly like [`DNS.lookup`](DNS.md#input-validation):
a host name or an IP address, never a URL, a path, or a `host:port` pair.
The same host is used for the connection, for SNI, and for the name check,
so a certificate is always judged against the name you asked about. An IP
address (`TLS.inspect("203.0.113.10")`) connects to that address, sends no
SNI, and is checked against the certificate's IP addresses.

`port` defaults to **443** and must be between **1 and 65535**. There is no
service discovery.

## Timeout

One timeout covers everything that involves the server — resolving the host,
connecting, and the handshake. `timeoutSeconds` defaults to **10** and must
be between **1 and 60**. A server that accepts the connection and then never
answers raises `TLSError` after the timeout; the connection is always closed
before `TLS.inspect` returns. Verification then runs locally on the
certificates already received; on Windows the platform verifier may itself
fetch a missing intermediate certificate, under the operating system's own
limits.

## Certificate expiry

`TLS.inspect` reports facts, not policy: it does not decide when a renewal is
"urgent". Compute the remaining time with the Time module and apply your own
thresholds:

```ahd
bring Math
bring TLS
bring Time

info := TLS.inspect("example.com")
remaining := Time.between(Time.utc(), info.notAfter())
days := Math.floor(remaining.milliseconds / 86400000)
if not info.valid() {
    write("certificate problem: " + info.verificationStatus())
} else if days <= 7 {
    write("renew now: " + str(days) + " days left")
} else if days <= 30 {
    write("renew soon: " + str(days) + " days left")
}
```

## Errors

`TLSError` means no certificate could be obtained at all. A reachable server
with a bad certificate is **not** an error — it is a `TLSInfo` whose
`valid()` is `false`. Messages have the form
`inspect "<host>:<port>" failed: <reason>`:

```text
inspect "127.0.0.1:8443" failed: connection refused
inspect "example.com:443" failed: timed out after 10 seconds (timeoutSeconds)
inspect "example.com:80" failed: the server did not answer with TLS
inspect "example.com:443" failed: the server closed the connection during the TLS handshake
inspect "missing.example:443" failed: host not found
inspect "example.com:70000" failed: port must be between 1 and 65535; received 70000
```

## Platform notes

Verification uses each operating system's trusted roots and, on macOS and
Windows, the platform's own certificate verifier. The verifiers agree on
ordinary certificates, but platform policies can differ at the edges: for
example, macOS rejects a server certificate whose validity period exceeds
Apple's limits and `TLS.inspect` then reports `invalid` there, while Linux
may report the same certificate differently. `valid()` is always the
platform's verdict and is never relaxed.

## Security: worker-only for untrusted input

`TLS.inspect` opens a network connection to whatever host and port it is
given. **Do not pass an untrusted web request straight to it** — that would
let a visitor make your server connect to internal addresses. Keep it in a
dedicated worker that only inspects validated, allowlisted domains; see the
architecture in [DNS](DNS.md#security-worker-only-for-untrusted-input).
AhdCode has no global allowlist of its own.

## Not in this version

Certificate issuance or renewal (ACME / Let's Encrypt), private keys,
client certificates, choosing a separate connect address and server name,
custom trust roots, OCSP and CRL checks, Certificate Transparency, raw
certificate or extension dumps, and HTTP requests are not part of `TLS`.
