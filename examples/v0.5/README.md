# AhdCode v0.5 examples

[English] · [Türkçe](README_TR.md)

[Back to project README](../../README.md)

These programs introduce v0.5.0 web state: HTTP cookies and an in-memory
server-side session. Bind `127.0.0.1`, then open `http://127.0.0.1:8080/`.
`Server.start()` occupies the terminal until you stop the program.

Session values live only while that process runs. Restarting the program
forgets every session. That is expected. HTTP-only examples need no SQLite
helper and no cookie/session helper.

```bash
ahdcode run examples/v0.5/01_cookie.ahd
```

| Example | Topic |
|---|---|
| `01_cookie.ahd` | Read, set, and delete a cookie |
| `02_session_counter.ahd` | Per-browser counter; String-to-Int is explicit |
| `03_session_login.ahd` | rotate on continue, destroy on logout, two-browser independence |

This is not production authentication. Session stores a name you typed.
