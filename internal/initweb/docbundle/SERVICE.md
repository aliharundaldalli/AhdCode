# Service standard module

[English] · Türkçe

[Back to README](README.md) · [Modules](MODULES.md) · [Disk](DISK.md) · [Process](PROCESS.md) · [Errors](ERRORS.md)

`Service` (v2.7.0) reports the state of a **Linux systemd** unit — whether a
service is active, running, failed, or enabled. It is **read-only**: it never
starts, stops, restarts, enables, disables, or reloads anything. Import it
explicitly:

```ahd
bring Service
from Service bring (ServiceInfo, ServiceError)
```

The canonical module identity is `builtin:Service`; a sibling `Service.ahd`
cannot shadow it. (Since v2.7.0 `Service` is a standard module name: a local
`Service.ahd` is no longer what `bring Service` loads — rename such a file.
Web projects' `Services/` folders, loaded with `require`, are unaffected.)

## Surface

```text
Service.status(name: String) -> ServiceInfo

ServiceInfo.name()        -> String   // the unit systemd resolved, e.g. "nginx.service"
ServiceInfo.activeState() -> String   // normalized: see below
ServiceInfo.subState()    -> String   // systemd's lower-level state, e.g. "running", "exited", "dead"
ServiceInfo.running()     -> Bool     // subState() == "running"
ServiceInfo.enabled()     -> Bool     // the unit is enabled to start at boot

ServiceError  (derives from Error)
```

`ServiceInfo` is an opaque value produced only by `Service.status`.

```ahd
unit: ServiceInfo := Service.status("nginx.service")
write(unit.name() + " is " + unit.activeState() + " (" + unit.subState() + ")")
```

(The member is `activeState()`, not `state()`: `state` is a reserved word in
AhdCode. The name matches systemd's own `ActiveState` property.)

## States

`activeState()` is normalized to a fixed vocabulary:

| `activeState()` | systemd ActiveState |
|---|---|
| `active` | `active`, `reloading`, `refreshing` |
| `inactive` | `inactive` |
| `failed` | `failed` |
| `activating` | `activating` |
| `deactivating` | `deactivating` |
| `unknown` | anything else |

`subState()` is systemd's own lower-level state (`running`, `exited`, `dead`,
`failed`, `start`, `stop-sigterm`, …), reported only when it is plain
lowercase text and otherwise as `unknown`. `running()` is true only for
`running`: an active one-shot unit that has finished (`active`/`exited`) is
not running.

`enabled()` is true when systemd's `UnitFileState` is `enabled` or
`enabled-runtime`. It is `false` for `disabled`, `masked`, and for units that
are `static` or started through a socket or another unit (for example
`systemd-journald.service`, or `ssh.service` on systems where `ssh.socket`
starts it) — those can still be active and running.

A unit name without a suffix is resolved by systemd (`nginx` →
`nginx.service`); `name()` reports the resolved name.

## How it works

`Service.status` runs the absolute `systemctl` executable
(`/usr/bin/systemctl` or `/bin/systemctl`) through the same shell-free
machinery as [`Process.run`](PROCESS.md): one executable, a fixed argument
list, a 10-second timeout, and bounded output. It asks for machine-readable
properties only:

```text
systemctl show --no-pager --property=Id,LoadState,ActiveState,SubState,UnitFileState -- <name>
```

It never parses the human-facing, colored, possibly localized
`systemctl status` output. The name is one literal argument after `--`, and
it is validated first: only ASCII letters, digits, and `:_.@-` are allowed,
at most 255 characters, not starting with `-` or `.`. Spaces, `;`, `$`,
quotes, slashes, and control characters are refused before any process runs,
so no name can become a command or an option.

## Errors

Every failure raises `ServiceError`, with the message
`service "<name>" status failed: <reason>`:

```text
service "ghost.service" status failed: service not found
service "nginx" status failed: systemd is not available on this system
service "nginx" status failed: permission denied
service "nginx" status failed: the inspection timed out after 10 seconds
service "a;b" status failed: the service name contains ';'; only letters, digits, and ':_.@-' are allowed
service "nginx" status failed: service inspection is supported only on Linux with systemd; this system is darwin
```

Raw `systemctl` error text is never part of the message.

## Platform support

| Platform | Behavior |
|---|---|
| Linux with systemd | supported |
| Linux without systemd (containers, other init systems) | `ServiceError`: systemd is not available |
| macOS, Windows, others | `ServiceError`: supported only on Linux with systemd |

AhdCode v2.7.0 deliberately does not guess service state from `launchctl`
or `sc.exe` output on other platforms.

## Read-only by design

There is no `Service.start`, `stop`, `restart`, `enable`, `disable`, or
`reload`. A worker that must change a service does so explicitly, with its
own allowlist, through [`Process.run`](PROCESS.md) — for example
`Process.run("/usr/bin/systemctl", ["restart", validatedUnit])`.

## Security: worker-only for untrusted input

Keep `Service.status` in a dedicated worker that inspects validated,
allowlisted units, never directly behind an untrusted web request:

```text
web application
    -> authenticated control channel
        -> dedicated AhdCode worker
            -> validated / allowlisted service name
                -> Service.status
```

```ahd
bring Service
from Service bring (ServiceInfo, ServiceError)

for unit in ["caddy.service", "ahdcode-org.service"] {
    attempt {
        info: Local ServiceInfo := Service.status(unit)
        if info.activeState() != "active" {
            write("ALERT: " + info.name() + " is " + info.activeState())
        }
    }
    except ServiceError as failure {
        write(failure.message)
    }
}
```

## Not in this version

Changing services, listing units, reading logs (journald), timers' schedules,
and service managers other than systemd are not part of `Service`.
