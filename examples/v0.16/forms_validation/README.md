# Forms and validation — v0.16

```bash
ahdcode run examples/v0.16/forms_validation/app.ahd
```

Open http://127.0.0.1:8160/register. No database or `.env` file is required.
`SERVER_PORT=8162` in the shell changes the port. `.env.example` is explanatory.

- GET `/register`: context, session-bound CSRF, semantic Web.UI form.
- POST `/register`: explicit CSRF check (403), ordered validation, selected
  name/email old input on failure (422), empty password controls.
- Successful POST: generic flash, 303 redirect, explicit response finalization.
- GET `/profile`: take flash, finalize its removal; refresh shows no message.

This demonstrates form state, not account creation or authentication. Passwords
are checked only in request memory. Only `name` and `email` enter OldInput; no
submitted values are copied to the session. Text and attributes use Web.UI
escaping. The local loopback server uses non-Secure cookies for HTTP; choose
Secure cookies for an HTTPS application. Flash remains pending until taken.

Handler naming follows the v0.16 convention: `Pages/Register.ahd` holds
`register` (GET) and `registerSubmit` (POST), `Pages/Profile.ahd` holds
`profile`. `Page` is not a required suffix — the router is handed a `Function`
value and never reads its name. See
[10.1 Naming](../../../docs/WEB.md#101-naming).

See [the full workflow and API](../../../docs/WEB.md).
