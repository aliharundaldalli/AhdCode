# Identity

`Identity.id()` produces a public identifier. It is not a secret.

```
bring Identity

value: String := Identity.id()
```

The value is 16 cryptographically random bytes encoded as unpadded URL-safe
Base64: 22 characters from `A-Za-z0-9-_`.

Use it for public record identifiers such as a `public_id` column. Do not use
it for passwords, session secrets, or CSRF tokens. Those belong to
`Security.token()` and `Security.passwordHash()`.

There is no decode API. The identifier is an opaque string.

AhdCode Web starters store a private numeric `id` for the database and a
separate `public_id` for URLs. Routes never expose the numeric id.
