# DNS standard module

[English] · [Türkçe](DNS_TR.md)

[Back to README](../README.md) · [Modules](MODULES.md) · [TLS](TLS.md) · [Errors](ERRORS.md)

`DNS` (v2.6.0) answers one question: **which IP addresses does this host
name resolve to right now?** It uses the operating system's resolver, is
always bounded by a timeout, and never runs a shell or an external tool (no
`dig`, no `nslookup`). Import it explicitly:

```ahd
bring DNS
from DNS bring (DNSResult, DNSError)
```

The canonical module identity is `builtin:DNS`; a sibling `DNS.ahd` cannot
shadow it. (Since v2.6.0 `DNS` is a standard module name: a local `DNS.ahd`
is no longer what `bring DNS` loads — rename such a file.)

## Surface

```text
DNS.lookup(host: String, timeoutSeconds: Int = 5) -> DNSResult

DNSResult.host()      -> String         // the normalized host that was looked up
DNSResult.addresses() -> List<String>   // every address, IPv4 first, then IPv6
DNSResult.ipv4()      -> List<String>
DNSResult.ipv6()      -> List<String>

DNSError  (derives from Error)
```

`DNSResult` is an opaque value produced only by `DNS.lookup`.

```ahd
found: DNSResult := DNS.lookup(host: "example.com", timeoutSeconds: 5)
write(found.host())
write(found.addresses())
```

## Address lookup only

v2.6.0 resolves addresses (the A and AAAA records a connection would use).
It does not query MX, TXT, NS, SRV, or CNAME records, does not choose a DNS
server, and never changes DNS data. Results come from the operating system's
resolver, so they honor the machine's own configuration (hosts file, search
domains, and upstream servers); the exact resolver internals differ between
macOS, Linux, and Windows, but the public result format does not.

## Result semantics

- Addresses are **deduplicated** and **sorted**: every IPv4 address first,
  then every IPv6 address, each family in ascending numeric order. The same
  answer therefore always prints the same way, in the REPL and in a compiled
  program alike.
- Addresses use canonical text (`192.0.2.10`, `2001:db8::1`); an
  IPv4-mapped IPv6 answer is reported as plain IPv4.
- `host()` is the normalized name: lowercase, without a trailing dot
  (`Panel.Example.COM.` → `panel.example.com`).
- A name that resolves to no address is a `DNSError`
  (`no addresses were found for the host`), never an empty success.
- An **IP literal** is returned as itself without any network access:
  `DNS.lookup("127.0.0.1")` gives `["127.0.0.1"]`, `DNS.lookup("[::1]")`
  gives `["::1"]`.

## Input validation

The host is checked before any network work. `DNS.lookup` takes a host name
or an IP address — nothing else:

| Input | Result |
|---|---|
| `""`, `"   "` | `the host is empty` / `the host is only whitespace` |
| `"https://example.com/path"` | `the host must be a host name such as example.com, not a URL` |
| `"example.com/path"`, `"example.com?x"` | `the host must not contain a path, query, or fragment` |
| `"example.com:443"` | `the host must not contain a port; pass the port separately` |
| `"user@example.com"` | `the host must not contain user information` |
| spaces, tabs, NUL, other control characters | rejected |
| `;`, `&`, `$`, `` ` ``, and any other character outside letters, digits, `-`, `_`, `.` | rejected |
| non-ASCII names | rejected — pass the ASCII (`xn--`) form of an internationalized name |
| labels longer than 63, names longer than 253, empty labels, labels starting or ending with `-` | rejected |

There is no shell anywhere, so text such as `example.com;touch X` or
`$(whoami)` is simply an invalid host name.

## Timeout

Every lookup is bounded: `timeoutSeconds` defaults to **5** and must be
between **1 and 60**. A lookup that does not finish in time raises
`DNSError` with `timed out after N seconds (timeoutSeconds)`.

## Errors

Every failure raises `DNSError`, whose message has the form
`lookup "<host>" failed: <reason>`:

```text
lookup "missing.example" failed: host not found
lookup "slow.example" failed: timed out after 5 seconds (timeoutSeconds)
lookup "broken.example" failed: the resolver could not answer (temporary failure)
lookup "empty.example" failed: no addresses were found for the host
lookup "example.com:443" failed: the host must not contain a port; pass the port separately
```

Messages never include the machine's resolver configuration (such as the
address of the DNS server that answered).

```ahd
attempt {
    found: Local DNSResult := DNS.lookup("panel.example.com")
    if not ("203.0.113.10" in found.ipv4()) {
        write("panel.example.com does not point at this server")
    }
}
except DNSError as failure {
    write(failure.message)
}
```

## Security: worker-only for untrusted input

`DNS.lookup` makes the program send network queries for whatever name it is
given. **Do not pass an untrusted web request straight to it.** For a control
panel, the recommended shape is:

```text
web application (unprivileged)
    -> authenticated control channel
        -> dedicated AhdCode worker
            -> validated / allowlisted domain
                -> DNS and TLS inspection
```

The worker decides which domains may be inspected (for example, only domains
registered to the requesting account). AhdCode has no global allowlist of its
own: that policy belongs to the application.

## Not in this version

Other record types (MX, TXT, NS, SRV, CNAME), choosing a DNS server,
DNSSEC validation, reverse lookups, zone transfers, and any DNS change
(records, zones, or provider APIs such as Cloudflare) are not part of `DNS`.
