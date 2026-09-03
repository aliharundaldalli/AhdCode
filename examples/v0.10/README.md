# AhdCode v0.10 — Security examples

These examples demonstrate the `Security` standard module introduced in v0.10.0.

| File | Topic |
|------|-------|
| `01_password_hash.ahd` | Argon2id hashing and verification |
| `02_secure_token.ahd` | Secure random tokens and `secureEqual` |
| `03_csrf_session.ahd` | CSRF protection with HTTP sessions |
| `04_sqlite_login.ahd` | SQLite register / login with password hashing |

All examples use fake passwords. Never use real credentials in source code.

## Running

```sh
ahdcode run 01_password_hash.ahd
ahdcode run 02_secure_token.ahd
ahdcode run 03_csrf_session.ahd   # starts HTTP server on :9080
ahdcode run 04_sqlite_login.ahd
```

## See also

- [Security module documentation](../../docs/SECURITY.md)
- [Student Guide — Security section](../../docs/STUDENT_GUIDE_EN.md)
