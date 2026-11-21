'use strict';

const VERSION = '1';
const PUBLIC_ASSETS = /* PUBLIC_ASSET_LIST */[];
const ORIGIN = self.location.origin;
const CACHE_PREFIX = 'ahdcode:pwa:' + self.registration.scope + ':';
const PUBLIC_TYPES = new Set([
  'image/png', 'image/jpeg', 'image/webp', 'text/css',
  'application/javascript', 'text/javascript', 'font/woff2',
]);

// Changing an asset or the version changes the cache identity. An update
// with identical assets can reuse only the same integrity-pinned bytes.
async function cacheName() {
  const encoded = new TextEncoder().encode(JSON.stringify([VERSION, PUBLIC_ASSETS]));
  const digest = await crypto.subtle.digest('SHA-256', encoded);
  return CACHE_PREFIX + Array.from(new Uint8Array(digest), byte =>
    byte.toString(16).padStart(2, '0')).join('');
}

async function publicResponse(asset) {
  if (!/^\/ahd-pwa\/[a-zA-Z0-9_-][a-zA-Z0-9_.-]*$/.test(asset.path) ||
      !PUBLIC_TYPES.has(asset.mime) ||
      !/^sha256-[A-Za-z0-9+/]{43}=$/.test(asset.integrity)) {
    throw new Error('Invalid public PWA asset: ' + asset.path);
  }
  const response = await fetch(ORIGIN + asset.path, {
    credentials: 'omit',
    redirect: 'error',
    cache: 'no-store',
    integrity: asset.integrity,
  });
  const mime = (response.headers.get('Content-Type') || '').split(';')[0].trim().toLowerCase();
  const cacheControl = response.headers.get('Cache-Control') || '';
  const vary = response.headers.get('Vary') || '';
  if (response.status !== 200 || mime !== asset.mime ||
      /\b(private|no-store|no-cache)\b/i.test(cacheControl) ||
      /(?:^|,)\s*(?:cookie|authorization|\*)\s*(?:,|$)/i.test(vary)) {
    throw new Error('Public PWA asset refused: ' + asset.path);
  }
  return response;
}

self.addEventListener('install', event => {
  event.waitUntil((async () => {
    // A failed download installs nothing and leaves the active worker alone.
    const responses = await Promise.all(PUBLIC_ASSETS.map(publicResponse));
    const cache = await caches.open(await cacheName());
    await Promise.all(responses.map((response, index) =>
      cache.put(ORIGIN + PUBLIC_ASSETS[index].path, response)));
  })());
});

self.addEventListener('activate', event => {
  event.waitUntil((async () => {
    const current = await cacheName();
    const keys = await caches.keys();
    await Promise.all(keys.filter(key => key.startsWith(CACHE_PREFIX) && key !== current)
      .map(key => caches.delete(key)));
    await self.clients.claim();
  })());
});

function offlinePage() {
  return new Response(`<!doctype html><html lang="en"><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Offline</title><main><h1>You are offline</h1>
<p>Reconnect and reload to use this application.</p></main></html>`, {
    status: 503,
    headers: {
      'Content-Type': 'text/html; charset=utf-8',
      'Cache-Control': 'no-store',
      'Content-Security-Policy': "default-src 'none'; base-uri 'none'; frame-ancestors 'none'",
    },
  });
}

async function publicAsset(request) {
  try {
    return await fetch(request, { cache: 'no-store' });
  } catch (error) {
    const cached = await caches.match(request.url, { cacheName: await cacheName() });
    if (cached) return cached;
    throw error;
  }
}

self.addEventListener('fetch', event => {
  const request = event.request;
  const url = new URL(request.url);
  // This guard deliberately returns without respondWith for live reload,
  // private routes, API calls, writes, cross-origin traffic and queries.
  if (request.method !== 'GET' || url.origin !== ORIGIN ||
      url.href !== ORIGIN + url.pathname || request.headers.has('Range') ||
      request.headers.has('Authorization') || url.pathname.startsWith('/__ahdcode/')) {
    return;
  }
  if (request.mode === 'navigate' && url.pathname === '/') {
    event.respondWith(fetch(request, { cache: 'no-store' }).catch(offlinePage));
    return;
  }
  if (request.destination && PUBLIC_ASSETS.some(asset => asset.path === url.pathname)) {
    event.respondWith(publicAsset(request));
  }
});
