# MySQL raffle demo

A small local raffle: members join and receive a participation code, an
admin starts the draw, then announces a winner. Passwords are stored with
`Security.passwordHash` (Argon2id), never in plaintext.

This is an example program, not a product and not part of AhdDataStudio.

## Start

```bash
cd examples/v0.12/raffle
cp .env.example .env   # then edit placeholders
ahdcode run app.ahd
```

Open [http://127.0.0.1:8082/](http://127.0.0.1:8082/)

Admin: [http://127.0.0.1:8082/admin](http://127.0.0.1:8082/admin)

Default admin (change in `.env`): `admin` / `change-me-local-only`

The first run hashes that password with Argon2id and stores only the hash.
Changing `.env` later does not update an existing `raffle_admin` row.

## Stop

```bash
cd examples/v0.12/raffle
ahdcode kill app.run
```

The server binds **127.0.0.1:8082** only, so it does not collide with
AhdDataStudio on 8081.

## What it shows

- MySQL `CREATE TABLE` / parameterized `INSERT` / `UPDATE` / `SELECT`
- `ORDER BY RAND() LIMIT 1` to pick one winner
- Argon2id admin password hashing
- HTTP sessions + CSRF on POST forms
- `HTML.text` for names and codes

Tables are created in `MYSQL_DATABASE` (`ahdcode_demo` by default). You can
inspect them in AhdDataStudio.

## Language

[English](README.md) · [Türkçe](README_TR.md)
