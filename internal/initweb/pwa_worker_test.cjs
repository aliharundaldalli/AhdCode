const assert = require('node:assert/strict');
const { webcrypto } = require('node:crypto');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const vm = require('node:vm');

const origin = 'https://pwa.example.test';
const scope = origin + '/';
const icon = '/ahd-pwa/icon-192.png';
const asset = { path: icon, mime: 'image/png', integrity: 'sha256-' + 'A'.repeat(43) + '=' };
const source = fs.readFileSync(path.join(__dirname, 'pwa', 'worker.js'), 'utf8');

function harness({ version = '1', assets = [asset], storage = new Map(), fetch } = {}) {
  const events = new Map();
  const calls = [];
  let network = fetch || (() => Promise.resolve(new Response('public', {
    headers: { 'Content-Type': 'image/png' },
  })));
  const caches = {
    open: async name => {
      if (!storage.has(name)) storage.set(name, new Map());
      return { put: async (key, value) => storage.get(name).set(key, value) };
    },
    keys: async () => Array.from(storage.keys()),
    delete: async name => storage.delete(name),
    match: async (key, { cacheName }) => storage.get(cacheName)?.get(key)?.clone(),
  };
  const script = source.replace('/* PUBLIC_ASSET_LIST */[]', JSON.stringify(assets))
    .replace("const VERSION = '1';", `const VERSION = ${JSON.stringify(version)};`);
  vm.runInNewContext(script, {
    self: {
      location: { origin },
      registration: { scope },
      addEventListener: (name, handler) => events.set(name, handler),
      clients: { claim: async () => {} },
    },
    caches, crypto: webcrypto, Headers, URL, TextEncoder, Response,
    fetch: (request, options) => {
      calls.push({ request, options });
      return network(request, options);
    },
  });
  return {
    storage, calls,
    setNetwork: value => { network = value; },
    lifecycle: name => {
      let work;
      events.get(name)({ waitUntil: value => { work = value; } });
      return work;
    },
    request: (pathname, extras) => {
      let response;
      const request = {
        url: new URL(pathname, scope).href, method: 'GET', mode: 'navigate',
        destination: 'document', headers: new Headers(), ...extras,
      };
      events.get('fetch')({ request, respondWith: value => { response = value; } });
      return response;
    },
  };
}

test('only integrity-pinned public assets enter the cache and updates own their namespace', async () => {
  const storage = new Map([['another-app', new Map()]]);
  const original = harness({ storage });
  await original.lifecycle('install');
  await original.lifecycle('activate');
  const request = original.calls[0];
  assert.equal(request.request, origin + icon);
  assert.equal(request.options.integrity, asset.integrity);
  assert.equal(request.options.credentials, 'omit');
  assert.equal(request.options.redirect, 'error');
  assert.equal(request.options.cache, 'no-store');
  const previous = [...storage.keys()].find(name => name !== 'another-app');
  const replacement = harness({ version: '2', storage });
  await replacement.lifecycle('install');
  assert.ok(storage.has(previous), 'install must leave the active cache present');
  await replacement.lifecycle('activate');
  assert.ok(!storage.has(previous));
  assert.ok(storage.has('another-app'));
  assert.equal(storage.size, 2);
  assert.deepEqual([...storage.values()].flatMap(items => [...items.keys()]), [origin + icon]);
});

test('invalid, private, varied, or failed asset installs do not replace the live cache', async () => {
  const active = harness();
  await active.lifecycle('install');
  const keys = [...active.storage.keys()];
  for (const response of [
    new Response('private', { headers: { 'Content-Type': 'image/png', 'Cache-Control': 'private' } }),
    new Response('varied', { headers: { 'Content-Type': 'image/png', Vary: 'Cookie' } }),
    new Response('HTML', { headers: { 'Content-Type': 'text/html' } }),
    new Response('missing', { status: 404, headers: { 'Content-Type': 'image/png' } }),
  ]) {
    const update = harness({ version: '2', storage: active.storage, fetch: async () => response });
    await assert.rejects(update.lifecycle('install'), /refused/);
    assert.deepEqual([...active.storage.keys()], keys);
  }
  const invalid = harness({ assets: [{ ...asset, path: '/__ahdcode/dev/reload' }], storage: active.storage });
  await assert.rejects(invalid.lifecycle('install'), /Invalid/);
  assert.deepEqual([...active.storage.keys()], keys);
});

test('online home always uses the network and no user or session response survives offline', async () => {
  const worker = harness();
  await worker.lifecycle('install');
  let state = 'Ada logged in';
  worker.setNetwork(async (request, options) => {
    assert.equal(options.cache, 'no-store');
    return new Response(state, { status: state === 'Logged out' ? 401 : 200 });
  });
  assert.equal(await (await worker.request('/')).text(), state);
  state = 'Ada after session rotation';
  assert.equal(await (await worker.request('/')).text(), state);
  state = 'Logged out';
  assert.equal((await worker.request('/')).status, 401);
  state = 'Other user';
  assert.equal(await (await worker.request('/')).text(), state);
  const before = worker.calls.length;
  for (const [route, extras] of [
    ['/admin'], ['/api/private', { mode: 'cors' }], ['/logout', { method: 'POST' }],
    ['/login', { method: 'POST' }], ['/__ahdcode/dev/reload', { mode: 'cors' }],
    ['/__ahdcode/dev/reload.js', { mode: 'cors', destination: 'script' }],
    ['/?session=one'], ['/?'], ['https://other.test/'],
    [icon, { destination: '' }], [icon + '?x=1', { destination: 'image' }],
    [icon, { headers: new Headers({ Authorization: 'test' }) }],
    [icon, { headers: new Headers({ Range: 'bytes=0-10' }) }],
  ]) {
    assert.equal(worker.request(route, extras), undefined, route);
  }
  assert.equal(worker.calls.length, before);
  worker.setNetwork(async () => { throw new TypeError('disconnected'); });
  const response = await worker.request('/');
  const fallback = await response.text();
  assert.equal(response.status, 503);
  assert.match(fallback, /You are offline/);
  assert.doesNotMatch(fallback, /Ada|Other user/);
  assert.equal(await (await worker.request(icon, { destination: 'image' })).text(), 'public');
  assert.deepEqual([...worker.storage.values()].flatMap(items => [...items.keys()]), [origin + icon]);
  worker.setNetwork(async () => new Response('Online again'));
  assert.equal(await (await worker.request('/')).text(), 'Online again');
});

test('registration uses an explicit root scope in secure top-level windows', async () => {
  const script = fs.readFileSync(path.join(__dirname, 'pwa', 'public', 'register.js'), 'utf8');
  for (const [secure, embedded, supported, count] of [
    [true, false, true, 1], [false, false, true, 0],
    [true, true, true, 0], [true, false, false, 0],
  ]) {
    const calls = [];
    const window = { isSecureContext: secure };
    window.top = embedded ? {} : window;
    const navigator = supported ? { serviceWorker: { register: async (...args) => calls.push(args) } } : {};
    vm.runInNewContext(script, { window, navigator, console });
    assert.equal(calls.length, count);
    if (count) {
      assert.equal(calls[0][0], '/ahdcode-sw.js');
      assert.equal(calls[0][1].scope, '/');
      assert.equal(calls[0][1].updateViaCache, 'none');
    }
  }
});
