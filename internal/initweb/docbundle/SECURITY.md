# Security standard module

[English] · Türkçe

[Back to README](README.md) · [Modules](MODULES.md) · [HTTP](HTTP.md) · [SQLite](SQLITE.md) · [Student Guide](STUDENT_GUIDE_EN.md#50-security-password-hashing-and-secure-tokens)

`Security` is the compiler-registered `builtin:Security` module, introduced in
AhdCode v0.10.0. It is explicit and a sibling `Security.ahd` cannot shadow it:

```ahd
bring Security
from Security bring SecurityError
```

`Security` is a narrow, opinionated set of cryptographic primitives: Argon2id
password hashing, opaque random tokens, constant-time comparison, SHA-2
digests, HMAC, the common encodings, RS256 signatures and AES-256-GCM
encryption. It is **not** a full authentication framework and not a JWT
library: `rsaSignSHA256` and `base64UrlEncode` give you the pieces a JWT is
built from, but assembling, validating and rotating tokens is your program's
job.

Where a cryptographic choice exists it is made once, safely, and is not exposed
as a knob — one digest family, one MAC, one signature scheme, one authenticated
cipher. You cannot select a broken mode or forget an authentication tag.

## ⚠ Critical warnings

- **Never store plaintext passwords.** Always store the PHC string from `passwordHash`.
- **Never use `Security.token` as a JWT.** Tokens carry no claims and are not signed.
- **Never log passwords or raw tokens.** Error messages from this module never include them.
- **Never use generic hash functions (SHA-256, MD5) for password storage.**
  Those functions are designed for speed; Argon2id is designed to be slow.
- `Security` provides primitives. Build a complete auth system on top of them.

## Public surface

```text
Security.passwordHash(password: String)                  -> String
Security.passwordVerify(password: String, encodedHash: String) -> Bool
Security.token()                                         -> String
Security.secureEqual(expected: String, received: String) -> Bool

Security.sha256(text: String)                            -> String
Security.sha512(text: String)                            -> String
Security.hmacSHA256(key: String, message: String)        -> String
Security.hmacVerify(key: String, message: String, receivedHex: String) -> Bool

Security.base64Encode(text: String)                      -> String
Security.base64Decode(encoded: String)                   -> String
Security.base64UrlEncode(text: String)                   -> String
Security.base64UrlDecode(encoded: String)                -> String
Security.hexEncode(text: String)                         -> String
Security.hexDecode(encoded: String)                      -> String

Security.randomHex(count: Int)                           -> String

Security.rsaSignSHA256(privateKeyPem: String, message: String)   -> String
Security.rsaVerifySHA256(publicKeyPem: String, message: String, signature: String) -> Bool
Security.aesEncrypt(keyHex: String, plaintext: String)   -> String
Security.aesDecrypt(keyHex: String, payload: String)     -> String
```

### Digests, MACs and encodings

`sha256` and `sha512` return lowercase hex. `hmacSHA256` returns lowercase hex
too. Compare a received MAC with `hmacVerify`, never with `==`: `hmacVerify`
uses a constant-time comparison, so it does not leak how many leading
characters matched.

`base64UrlEncode` emits the URL-safe alphabet (`-` and `_`) **without**
padding, which is what JWT and JWS segments require. `base64UrlDecode` accepts
padded input as well, because other systems emit it.

`randomHex(count)` returns `count` cryptographically random bytes as hex, so
the string is twice as long as the byte count. `count` must be between 1 and
1024. Thirty-two bytes is the right size for an AES-256 key or an HMAC secret.

### Signatures

`rsaSignSHA256` is RSASSA-PKCS1-v1_5 over SHA-256 — the algorithm JSON Web
Tokens call **RS256** — and returns the signature as unpadded base64url. The
private key is PEM, in either PKCS#1 (`RSA PRIVATE KEY`) or PKCS#8
(`PRIVATE KEY`) form; service-account files use PKCS#8. `rsaVerifySHA256`
accepts a PKIX or PKCS#1 public key and returns `false` for a bad signature,
raising only when the key or the encoding itself is unusable.

Together with `base64UrlEncode` these are enough to build a signed JWT:

```ahd
header: String := Security.base64UrlEncode("{\"alg\":\"RS256\",\"typ\":\"JWT\"}")
claims: String := Security.base64UrlEncode(claimsJson)
signature: String := Security.rsaSignSHA256(privateKeyPem, header + "." + claims)
token: String := header + "." + claims + "." + signature
```

### Symmetric encryption

`aesEncrypt` and `aesDecrypt` are AES-256-GCM. The key is 32 bytes given as 64
hex characters; any other length is rejected. The nonce is generated per call
and carried inside the returned base64 payload, so the caller never manages it
— reusing a nonce with the same key destroys GCM's security, and this API makes
that mistake impossible.

Decryption is authenticated: a modified or truncated payload raises
`SecurityError` instead of returning attacker-influenced plaintext. There is no
mode selection, so a broken mode such as ECB cannot be chosen by accident.

### Error type

```ahd
from Security bring SecurityError
```

`SecurityError` extends `Error`. It is raised on:
- Malformed or truncated PHC strings
- Unsupported algorithm or version in a stored hash
- Parameters outside safe bounds (checked before running Argon2)
- Entropy failure during random generation (extremely rare)

Wrong passwords return `false`; they **never** raise `SecurityError`.

## passwordHash

```ahd
hash: String := Security.passwordHash("fake_password_example")
```

Hashes `password` with Argon2id and returns a PHC (Password Hashing
Competition) encoded string. The encoding stores the algorithm, parameters,
salt, and derived key together so that `passwordVerify` needs only this one
string.

**Argon2id parameters (v0.10.0):**

| Parameter | Value | Meaning |
|-----------|-------|---------|
| algorithm | argon2id | Memory-hard, side-channel-resistant |
| version | v19 (0x13) | RFC 9106 |
| memory | 65 536 KiB | 64 MiB per hash |
| iterations | 3 | Time cost |
| parallelism | 1 | Thread count |
| salt | 16 bytes | Cryptographically random, per-hash |
| derived key | 32 bytes | Output length |

**PHC string shape:**

```
$argon2id$v=19$m=65536,t=3,p=1$<base64-salt>$<base64-key>
```

Both `<base64-salt>` and `<base64-key>` use standard unpadded Base64
(`base64.RawStdEncoding`), not base64url.

**Password size limit:** 1 MiB (1 048 576 bytes). Larger inputs raise
`SecurityError` before any hashing. Empty passwords are allowed.

**UTF-8 behavior:** The password is treated as raw UTF-8 bytes, the same as
every other `String` in AhdCode.

## passwordVerify

```ahd
ok: Bool := Security.passwordVerify(candidate, storedHash)
```

Parses `storedHash`, validates its parameters, recomputes Argon2id with the
stored salt, and compares the result using `crypto/subtle.ConstantTimeCompare`.

| Input condition | Result |
|-----------------|--------|
| Correct password | `true` |
| Wrong password | `false` |
| Malformed PHC string | raises `SecurityError` |
| Unsupported algorithm (not argon2id) | raises `SecurityError` |
| Unsupported version (not v19) | raises `SecurityError` |
| Parameters outside safe bounds | raises `SecurityError` (before Argon2 runs) |

**Safe parameter bounds (verification):**

| Parameter | Minimum | Maximum |
|-----------|---------|---------|
| memory | 8 192 KiB | 262 144 KiB |
| iterations | 1 | 10 |
| parallelism | 1 | 16 |
| salt length | 8 bytes | 64 bytes |
| hash length | 16 bytes | 64 bytes |

These bounds prevent a stored hash from forcing the verifier to spend
excessive resources or use pathologically weak parameters.

## token

```ahd
tok: String := Security.token()
```

Generates 32 random bytes from `crypto/rand` and encodes them with
`base64.RawURLEncoding` (no padding). The result is always 43 characters,
uses only URL-safe characters (`A–Z`, `a–z`, `0–9`, `-`, `_`), and carries
256 bits of entropy.

`Security.token` fails fatally on entropy failure; it never falls back to a
weaker source.

Use tokens for:
- CSRF hidden fields
- Password-reset links
- Session IDs (if you are not using `HTTP.sessionStore`)

Do **not** use tokens as JWTs — they carry no claims, no expiry, and are not signed.

## secureEqual

```ahd
same: Bool := Security.secureEqual(expected, received)
```

Compares two strings in constant time using `crypto/subtle.ConstantTimeCompare`.
Returns `true` only when both strings are byte-for-byte identical. Never panics
on different-length inputs.

Use `secureEqual` whenever you compare a value from an untrusted source
against a known secret (CSRF token, API key, webhook signature). Ordinary
`==` is not constant-time and may leak information about the secret through
timing differences.

## CSRF protection pattern

```ahd
bring HTTP
bring Security

app := HTTP.server("127.0.0.1", 8080)
store := HTTP.sessionStore("SESSID", Env.getOr("SESSION_SECRET", "dev-only"))

app.get("/form", fn(req) -> Response {
    session := store.session(req)
    tok := Security.token()
    session.set("csrf", tok)
    return HTTP.html("<form method='POST' action='/submit'>" +
        "<input type='hidden' name='csrf' value='" + tok + "'/>" +
        "<button>Submit</button></form>")
})

app.post("/submit", fn(req) -> Response {
    session := store.session(req)
    stored: String?    := session.get("csrf")
    submitted: String? := req.field("csrf")
    if stored == null or submitted == null {
        return HTTP.text("rejected", 403)
    }
    if Security.secureEqual(stored, submitted) {
        session.set("csrf", Security.token())   // rotate after use
        return HTTP.text("ok")
    }
    return HTTP.text("rejected", 403)
})
```

## SQLite storage example

Store only the PHC string. No separate salt column is needed.

```ahd
bring Security
bring SQLite
from Security bring SecurityError

db := SQLite.open("users.db")
db.execute("CREATE TABLE IF NOT EXISTS users (username TEXT PRIMARY KEY, hash TEXT NOT NULL)")

// Register
fn register(db: Database, username: String, password: String) {
    db.execute("INSERT INTO users (username, hash) VALUES (?, ?)",
        [SQLite.fromString(username), SQLite.fromString(Security.passwordHash(password))])
}

// Login
fn login(db: Database, username: String, attempt: String) -> Bool {
    rows := db.query("SELECT hash FROM users WHERE username = ?",
        [SQLite.fromString(username)])
    if len(rows) == 0 { return false }
    return Security.passwordVerify(attempt, rows[0]["hash"].string())
}
```

## Error messages

| Message | Meaning |
|---------|---------|
| `Security password hash is malformed` | PHC string did not parse |
| `Security password hash uses an unsupported algorithm` | Not argon2id / not v19 |
| `Security password hash has unsafe parameters` | Parameters out of safe bounds |
| `Security password input is too large` | Password exceeded 1 MiB |
| `Security random token generation failed` | OS entropy failure |

Passwords never appear in error messages.

## See also

- v0.10 examples
- [Student Guide — Security](STUDENT_GUIDE_EN.md#50-security-password-hashing-and-secure-tokens)
- [HTTP module](HTTP.md) — sessions, CSRF context
- [SQLite module](SQLITE.md) — storing hashes
- [Env module](ENV.md) — loading secrets from environment
