#!/usr/bin/env python3
"""Run serial HTTP acceptance checks against an already-built v0.16 fixture.

Usage: python3 tools/qa/v016_forms.py /absolute/path/to/acceptance
No builds, downloads, external services, or third-party Python dependencies.
Only the exact subprocess started here is terminated. Tokens are never printed.
"""
import html
import http.cookiejar
import os
from pathlib import Path
import re
import socket
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.parse
import urllib.request


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


def client():
    jar = http.cookiejar.CookieJar()
    return urllib.request.build_opener(urllib.request.HTTPCookieProcessor(jar), NoRedirect())


def run(binary):
    with socket.socket() as sock:
        sock.bind(('127.0.0.1', 0))
        port = sock.getsockname()[1]
    base = f'http://127.0.0.1:{port}'
    checks = 0

    def check(ok, label):
        nonlocal checks
        if not ok:
            raise AssertionError(label)
        checks += 1

    def request(opener, path, data=None):
        body = None if data is None else urllib.parse.urlencode(data).encode()
        try:
            response = opener.open(base + path, data=body, timeout=5)
        except urllib.error.HTTPError as error:
            response = error
        with response:
            return response.status, response.headers, response.read().decode()

    def token(opener):
        status, headers, body = request(opener, '/register')
        check(status == 200, 'GET form')
        field = re.search(r'<input[^>]*name="_csrf"[^>]*>', body)
        check(field is not None, 'CSRF hidden field')
        value = re.search(r'value="([^"]+)"', field[0])
        check(value is not None and len(value[1]) >= 32, 'nonempty token')
        check('type="hidden"' in field[0], 'hidden CSRF type')
        return html.unescape(value[1]), headers

    with tempfile.TemporaryDirectory(prefix='ahd-v016-relocated-') as cwd:
        with tempfile.TemporaryFile(mode='w+') as log:
            process = subprocess.Popen([str(binary)], cwd=cwd,
                                       env={**os.environ, 'SERVER_PORT': str(port)},
                                       stdout=log, stderr=log)
            try:
                first, second = client(), client()
                deadline = time.monotonic() + 10
                while time.monotonic() < deadline:
                    if process.poll() is not None:
                        raise RuntimeError('acceptance process exited before readiness')
                    try:
                        if request(first, '/ok')[0] == 200:
                            break
                    except (OSError, urllib.error.URLError):
                        time.sleep(.1)
                else:
                    raise RuntimeError('acceptance process did not become ready')
                one, headers = token(first)
                check(len(headers.get_all('Set-Cookie', [])) == 1, 'CSRF session committed once')
                again, headers = token(first)
                check(again == one, 'CSRF token stable across requests')
                check(not headers.get_all('Set-Cookie'), 'unchanged session emits no cookie')
                two, _ = token(second)
                check(two != one, 'independent client CSRF tokens')
                for data in [{}, {'_csrf': ''}, {'_csrf': 'wrong'}, {'_csrf': two}]:
                    status, headers, _ = request(first, '/register', data)
                    check(status == 403, 'missing/empty/wrong/other-client token rejected')
                    check(not headers.get_all('Set-Cookie'), 'invalid CSRF commits no unrelated changes')
                check(request(client(), '/register', {'_csrf': one})[0] == 403,
                      'token without bound session rejected')
                payload = '<script>alert(1)</script>'
                invalid = {'_csrf': one, 'name': '', 'email': payload,
                           'password': 'SuperSecret123', 'password_confirmation': 'different',
                           'reset_token': 'ResetSecret456'}
                status, headers, body = request(first, '/register', invalid)
                check(status == 422, 'validation early response')
                check('&lt;script&gt;alert(1)&lt;/script&gt;' in body and payload not in body,
                      'old input escapes executable markup')
                check('SuperSecret123' not in body and 'ResetSecret456' not in body
                      and 'value="different"' not in body, 'secret inputs not re-rendered')
                check(not headers.get_all('Set-Cookie'), 'validation saves no old input to session')
                check(request(first, '/state')[2] == 'anonymous', 'validation has no login mutation')
                check(token(first)[0] == one, 'validation preserves CSRF')
                data = {**invalid, 'empty': '', 'number': '42', 'badnumber': 'not-an-int',
                        'overflow': '999999999999999999999999', 'checked': 'on', 'unchecked': '0'}
                status, _, body = request(first, '/form-checks', data)
                check(status == 200 and '&lt;script&gt;' in body, 'typed form and explicit old-input checks')
                status, headers, _ = request(first, '/double')
                check(status == 200 and len(headers.get_all('Set-Cookie', [])) == 1,
                      'alias second-finalization rejection without duplicate cookies')
                check(request(first, '/flash-twice')[0] == 200, 'flash takes once within request')
                status, _, _ = request(first, '/register', {'_csrf': one, 'name': 'Ali',
                    'email': 'ali@example.com', 'password': 'SuperSecret123',
                    'password_confirmation': 'SuperSecret123'})
                check(status == 303, 'valid registration redirect')
                check('No pending message.' in request(second, '/profile')[2], 'flash isolation')
                check('Form accepted.' in request(first, '/profile')[2], 'flash survives redirect')
                check('No pending message.' in request(first, '/profile')[2], 'flash removed after next request')
                status, headers, _ = request(first, '/protected/2?x=1', {'_csrf': one})
                check(status == 303 and headers.get('Location') == '/profile', 'wildcard protected POST redirect')
                check(len(headers.get_all('Set-Cookie', [])) == 1, 'rotated login commits once')
                check(request(first, '/state')[2] == 'Ali', 'login Session alias mutations persist')
                check(request(second, '/state')[2] == 'anonymous', 'session isolation')
                check(token(first)[0] == one, 'CSRF survives session rotation')
                check('signed in' in request(first, '/profile')[2], 'login flash')
                check('No pending message.' in request(first, '/profile')[2], 'login flash once')
                check(request(first, '/protected/exact', {})[0] == 200, 'exact route before wildcard')
                check(request(first, '/protected/2/more', {})[0] == 404, 'one segment only')
                check(request(first, '/early')[0] == 404, 'explicit 404 finalization')
                check(request(first, '/legacy')[0] == 200, 'low-level session commit')
                check(request(first, '/legacy-read')[0] == 200, 'context reads low-level session state')
                status, headers, _ = request(first, '/logout', {'_csrf': one})
                check(status == 303 and len(headers.get_all('Set-Cookie', [])) == 1, 'logout deletion committed once')
                check(request(first, '/state')[2] == 'anonymous', 'logout state cleared')
                check(token(first)[0] != one, 'new session receives new CSRF')
                # The fixture runs validationChecks before starting the server;
                # its stdout can remain buffered for the lifetime of the server.
                check(request(first, '/ok')[2] == 'ok', 'validator table completed before serving')
                print(f'PASS: {checks} HTTP assertions plus AhdCode validation/form checks; isolated runtime directory.')
            finally:
                if process.poll() is None:
                    process.terminate()
                    try:
                        process.wait(timeout=5)
                    except subprocess.TimeoutExpired:
                        process.kill()
                        process.wait(timeout=5)


if __name__ == '__main__':
    run(Path(sys.argv[1]).resolve(strict=True))
