# v0.18 Web starters

`ahdcode init web` writes one of three starters into the current directory.

```bash
mkdir my-app
cd my-app
ahdcode init web
ahdcode dev app.ahd
```

- **Empty** — welcome application. No database, no login.
- **Basic** — welcome application plus `Config/Mail.ahd`.
- **Admin** — Home, Login, Dashboard, one administrator, SQLite or MySQL.

The generated source belongs to the application. Auth policy is ordinary
code (`signedIn`), not a framework feature.
