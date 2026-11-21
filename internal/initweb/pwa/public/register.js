(() => {
  if (window.top !== window || !window.isSecureContext || !('serviceWorker' in navigator)) {
    return;
  }
  navigator.serviceWorker.register('/ahdcode-sw.js', {
    scope: '/',
    updateViaCache: 'none',
  }).catch(error => console.warn('AhdCode PWA registration failed:', error));
})();
