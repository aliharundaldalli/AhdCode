# Browser refresh under `ahdcode dev`

[English] · [Türkçe](README_TR.md)

This page is ordinary `HTTP.html`. `ahdcode dev` is what refreshes the browser.

```text
ahdcode dev app.ahd
```

Open the address the program prints. Leave the tab open and change the
heading in `app.ahd`. After the dev log shows `→ Restarting...`,
`✓ Running`, and, for a program that `bring Web`, `✓ Browser will refresh`,
the tab shows the new heading without a manual reload. The first start of a
Web program prints a `Browser refresh:` section in the banner and does not
print that checkmark after the first `✓ Running`.

Save a syntax error. The terminal keeps the diagnostic, the last page stays
in the browser, and the reload address still reports the same session and
generation. Fix the file and the tab loads again. Stop `ahdcode dev` and
start it once more: the new process has a new session, so a tab that was
left open loads the new page instead of staying on the old one.

The script is an external file at `/__ahdcode/dev/reload.js`, so a page
whose `Content-Security-Policy` allows `script-src 'self'` can run it.
`.env` cannot turn the hook on. A response labeled `gzip`, `UTF-16`, or a
redirect is left unchanged.

`ahdcode run app.ahd` serves the page and does not inject the script.
`GET /__ahdcode/dev/reload` is then an ordinary 404.

This example uses the live reload behavior shipped in AhdCode v2.10.0.

## Limits

- The refresh is a full page load. Form input and scroll position are not kept.
- Static files and `HTTP.file` responses are not part of the watch, and they
  are not rewritten with the script.
- The hook exists only while the process was started by `ahdcode dev`.
- Local HTTPS is unchanged: `ahdcode dev` still refuses `APP_PROTOCOL=https`.
