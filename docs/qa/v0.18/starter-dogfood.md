# v0.18 starter dogfood

Measured from an empty directory to a running application. No migration or
seed command is required.

## Empty

Commands:

1. `mkdir my-app && cd my-app`
2. `ahdcode init web` → Empty → application name
3. `ahdcode dev app.ahd`

Tree:

```
app.ahd
.env
.env.example
.gitignore
Config/App.ahd
Components/Navbar.ahd
Components/Footer.ahd
Layouts/Main.ahd
Pages/Home.ahd
public/style.css
public/main.js
public/ahdcode-logo.png
public/vendor/bootstrap/bootstrap.min.css
public/vendor/bootstrap/bootstrap.bundle.min.js
public/vendor/bootstrap/LICENSE
```

No `database/`, `Services/`, `Repositories/`, Login, or Dashboard.

## Basic

Same three commands. Wizard: Basic → application name.

Adds `Config/Mail.ahd` and `MAIL_*` keys. Still no database and no login.

## Admin SQLite

Same three commands. Wizard answers: Admin → application name → SQLite →
database name → admin name / email / password (hidden).

Adds Login, Dashboard, `Services/Auth.ahd`, `Repositories/Users.ahd`,
`Config/Database.ahd`, `database/schema.sql`, and `database/<name>.db`.
Logout is POST `/logout` and redirects to `/`.

## Decisions

- Generated SQLite `database/*.db` is gitignored; `schema.sql` is not.
- Bootstrap 5.3.3 MIT files are vendored and local.
- Existing local files and existing databases stop init before side effects.
