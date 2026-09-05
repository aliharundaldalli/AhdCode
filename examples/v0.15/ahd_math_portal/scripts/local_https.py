#!/usr/bin/env python3
"""Start/stop this portal's optional local Caddy HTTPS preview; no auto-builds."""
import json
import os
from pathlib import Path
import shutil
import signal
import socket
import ssl
import subprocess
import sys
import time
import urllib.request

PROJECT = Path(__file__).resolve().parents[1]
LOCAL = PROJECT / '.local'
STATE = LOCAL / 'https-processes.json'
BINARY = LOCAL / 'portal'
CONFIG = PROJECT / 'Caddyfile.local'
ROOT_CERT = LOCAL / 'caddy-data/caddy/pki/authorities/local/root.crt'


def owned_process(record):
    result = subprocess.run(['ps', '-p', str(record['pid']), '-o', 'command='],
                            capture_output=True, text=True)
    command = result.stdout.strip()
    return (command == str(BINARY) if record['kind'] == 'portal'
            else command == record['command'])


def stop(records):
    for record in reversed(records):
        if owned_process(record):
            os.kill(record['pid'], signal.SIGTERM)
    for _ in range(100):
        if not any(owned_process(r) for r in records):
            return
        time.sleep(.1)
    raise RuntimeError('A recorded process did not stop; inspect it manually.')


def start():
    if STATE.exists():
        records = json.loads(STATE.read_text())
        if any(owned_process(r) for r in records):
            raise RuntimeError('Local preview already has a running process. Use status or stop.')
    if not BINARY.is_file():
        raise RuntimeError('Build first: mkdir -p .local && ahdcode build app.ahd -o .local/portal')
    caddy = shutil.which('caddy')
    if not caddy:
        raise RuntimeError('Caddy is required for this optional preview. Install it with brew install caddy.')
    for port in (8443, 8161):
        with socket.socket() as probe:
            probe.bind(('127.0.0.1', port))
    LOCAL.mkdir(mode=0o700, exist_ok=True)
    LOCAL.chmod(0o700)
    environment = os.environ.copy()
    environment.update(APP_ENV='production', APP_HOST='ahdakademi.com.test', APP_PROTOCOL='https', APP_PUBLIC_PORT='8443',
                       SERVER_HOST='127.0.0.1', SERVER_PORT='8161')
    proxy_environment = os.environ.copy()
    proxy_environment.update(XDG_DATA_HOME=str(LOCAL / 'caddy-data'),
                             XDG_CONFIG_HOME=str(LOCAL / 'caddy-config'))
    records = []
    try:
        for kind, args, env in [
            ('portal', [str(BINARY)], environment),
            ('caddy', [caddy, 'run', '--config', str(CONFIG), '--adapter', 'caddyfile'], proxy_environment),
        ]:
            fd = os.open(LOCAL / (kind + '.log'), os.O_WRONLY | os.O_CREAT | os.O_APPEND, 0o600)
            with os.fdopen(fd, 'ab') as output:
                child = subprocess.Popen(args, cwd=PROJECT, env=env, stdout=output,
                                         stderr=subprocess.STDOUT, start_new_session=True)
            records.append({'kind': kind, 'pid': child.pid, 'command': ' '.join(args)})
        STATE.write_text(json.dumps(records, indent=2) + '\n')
        for _ in range(200):
            if not all(owned_process(r) for r in records):
                raise RuntimeError('Preview process exited. Inspect the private .local logs.')
            if ROOT_CERT.exists():
                try:
                    context = ssl.create_default_context(cafile=str(ROOT_CERT))
                    with urllib.request.urlopen('https://ahdakademi.com.test:8443/', context=context, timeout=2) as response:
                        if response.status == 200:
                            print('Ready: https://ahdakademi.com.test:8443 (HTTP 200; certificate verified)')
                            print('Browser trust is separate; see README_TR.md or README.md.')
                            return
                except OSError:
                    pass
            time.sleep(.1)
        raise RuntimeError('HTTPS preview did not become ready.')
    except BaseException:
        stop(records)
        STATE.unlink(missing_ok=True)
        raise


def main():
    action = sys.argv[1] if len(sys.argv) == 2 else 'status'
    if action == 'start':
        start()
    elif action in ('status', 'stop'):
        records = json.loads(STATE.read_text()) if STATE.exists() else []
        if action == 'stop':
            stop(records)
            STATE.unlink(missing_ok=True)
            print('Stopped only this preview\'s recorded, identity-checked processes.')
        else:
            for record in records:
                print(record['kind'], record['pid'], 'running' if owned_process(record) else 'stopped')
            print('https://ahdakademi.com.test:8443' if records else 'Local HTTPS preview is stopped.')
    else:
        raise RuntimeError('Usage: python3 scripts/local_https.py [start|status|stop]')


if __name__ == '__main__':
    try:
        main()
    except (RuntimeError, OSError) as error:
        sys.exit(str(error))
