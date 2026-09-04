# AhdCode v0.11 examples

[English] · [Türkçe](README_TR.md)

[Back to project README](../../README.md)

These examples demonstrate the `MySQL` standard module introduced in
v0.11.0: connecting, parameterized CRUD, transactions, and using `MySQL`
alongside `SQLite` in one process.

| File | Topic |
|---|---|
| `01_mysql_connect.ahd` | Connect with no default database, list every database the credentials can see |
| `02_mysql_crud.ahd` | Parameterized INSERT/SELECT/UPDATE/DELETE, `affectedRows`, `lastInsertId` |
| `03_mysql_transaction.ahd` | A two-row transfer, committed together or rolled back together |
| `04_mysql_and_sqlite.ahd` | `MySQL` and `SQLite` used in the same program, no collision |

All examples read connection details from `Env` and use placeholder
credentials. Never hardcode a real password.

## Running

```bash
export MYSQL_HOST=127.0.0.1
export MYSQL_USERNAME=app
export MYSQL_PASSWORD=change-me
export MYSQL_DATABASE=ahdcode_demo
export MYSQL_SECURITY=tls   # or "none" for a trusted local development server

ahdcode run 01_mysql_connect.ahd
ahdcode run 02_mysql_crud.ahd
ahdcode run 03_mysql_transaction.ahd
ahdcode run 04_mysql_and_sqlite.ahd
```

These examples need a real, reachable MySQL server; none of them start one.

## See also

- [MySQL module documentation](../../docs/MYSQL.md)
- [Student Guide — MySQL section](../../docs/STUDENT_GUIDE_EN.md#51-mysql-a-network-database-server)
