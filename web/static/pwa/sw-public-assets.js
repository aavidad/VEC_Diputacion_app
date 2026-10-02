/* Recursos públicos de la PWA. Este fichero se carga desde los tres workers. */
(function () {
  'use strict';

  const VERSION = '20261002-pwa-v1';
  const PORTALS = Object.freeze({
    empleado: '/portal-empleado/',
    personal: '/area-personal/',
    admin: '/administracion-perfiles/'
  });
  const ICONS = new Set([
    '/pwa/icons/vec-192.png',
    '/pwa/icons/vec-512.png',
    '/pwa/icons/vec-maskable-192.png',
    '/pwa/icons/vec-maskable-512.png',
    '/pwa/icons/vec.ico'
  ]);
  const CONTENT_TYPES = Object.freeze({
    css: /^(?:text\/css)(?:;|$)/i,
    js: /^(?:(?:text|application)\/javascript)(?:;|$)/i,
    json: /^application\/json(?:;|$)/i,
    png: /^image\/png(?:;|$)/i,
    ico: /^(?:image\/(?:x-icon|vnd\.microsoft\.icon)|application\/octet-stream)(?:;|$)/i,
    woff: /^font\/woff(?:;|$)/i,
    woff2: /^font\/woff2(?:;|$)/i
  });

  function versionValida(url) {
    const params = [...url.searchParams.entries()];
    return params.length === 1 && params[0][0] === 'v' &&
      /^[a-z0-9][a-z0-9._-]{0,79}$/i.test(params[0][1]);
  }

  function tipoDeRecurso(url, scopePath) {
    const path = url.pathname;
    if (/[%\\]|\/\.|\/\//.test(path) || !/^[a-z0-9/_.-]+\.[a-z0-9]+$/i.test(path)) return null;
    if (path.endsWith('/sw.js')) return null;
    if (/\.(?:test|spec)\.|(?:^|\/)(?:test|tests|fixtures|datos-presentacion|adaptador-presentacion)/i.test(path)) return null;
    if (ICONS.has(path)) return path.endsWith('.ico') ? 'ico' : 'png';
    if (/^\/textos\/(?:es|en)\/[a-z0-9/_-]+\.json$/i.test(path)) return 'json';
    if (path === '/styles.css' || path.startsWith('/comun/')) {
      const ext = path.slice(path.lastIndexOf('.') + 1);
      return ext === 'css' || ext === 'js' || ext === 'woff' || ext === 'woff2' ? ext : null;
    }
    if (!path.startsWith(scopePath)) return null;
    const ext = path.slice(path.lastIndexOf('.') + 1);
    return ext === 'css' || ext === 'js' || ext === 'woff' || ext === 'woff2' ? ext : null;
  }

  function recursoPermitido(request, origin, scopePath) {
    if (request.method !== 'GET') return null;
    const url = new URL(request.url);
    if (url.origin !== origin || url.username || url.password || url.hash || !versionValida(url)) return null;
    const tipo = tipoDeRecurso(url, scopePath);
    if (!tipo) return null;
    if (tipo !== 'json' && request.destination && !({
      css: ['style'], js: ['script', 'worker', 'sharedworker'],
      png: ['image'], ico: ['image'], woff: ['font'], woff2: ['font']
    }[tipo] || []).includes(request.destination)) return null;
    return { url, tipo };
  }

  function respuestaPublica(response, original, tipo) {
    if (!response || response.status !== 200 || !response.ok || response.redirected || response.type !== 'basic') return false;
    if (!response.url || response.url !== original.href) return false;
    if (!CONTENT_TYPES[tipo].test(response.headers.get('Content-Type') || '')) return false;
    const cacheControl = response.headers.get('Cache-Control') || '';
    if (/\bprivate\b/i.test(cacheControl)) return false;
    if (tipo !== 'json' && /\bno-store\b/i.test(cacheControl)) return false;
    if (response.headers.has('Set-Cookie') || response.headers.has('Content-Disposition')) return false;
    if (/(?:\*|cookie|authorization)/i.test(response.headers.get('Vary') || '')) return false;
    return true;
  }

  function textoSeguro(value) {
    return String(value).replace(/[&<>"']/g, char => ({
      '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'
    }[char]));
  }

  function iniciar(portal) {
    const scopePath = PORTALS[portal];
    if (!scopePath || new URL(self.registration.scope).pathname !== scopePath) throw new Error('scope PWA inválido');
    const origin = self.location.origin;
    const cacheName = `vec-pwa-${portal}-public-${VERSION}`;
    const cachePrefix = `vec-pwa-${portal}-public-`;
    const catalogo = idioma => `${origin}/textos/${idioma}/pwa.json?v=${VERSION}`;

    async function pedirPublico(url, tipo) {
      const response = await fetch(url.href, {
        cache: 'no-store', credentials: 'omit', redirect: 'error', referrerPolicy: 'no-referrer'
      });
      return respuestaPublica(response, url, tipo) ? response : null;
    }

    async function prepararCatalogos() {
      const cache = await caches.open(cacheName);
      for (const idioma of ['es', 'en']) {
        const url = new URL(catalogo(idioma));
        const response = await pedirPublico(url, 'json');
        if (!response) throw new Error('catálogo público PWA no disponible');
        await cache.put(url.href, response);
      }
    }

    async function recursoEstatico(request, recurso) {
      const cache = await caches.open(cacheName);
      const guardado = await cache.match(recurso.url.href);
      if (guardado) return guardado;
      let response;
      try { response = await pedirPublico(recurso.url, recurso.tipo); } catch (_) { return fetch(request); }
      if (!response) return fetch(request);
      try { await cache.put(recurso.url.href, response.clone()); } catch (_) { /* Sin cuota: sigue la red. */ }
      return response;
    }

    async function sinConexion(request) {
      const preferido = /^en\b/i.test(request.headers.get('Accept-Language') || '') ? 'en' : 'es';
      const cache = await caches.open(cacheName);
      let contenido = await cache.match(catalogo(preferido));
      let idioma = preferido;
      if (!contenido && preferido !== 'es') {
        contenido = await cache.match(catalogo('es'));
        idioma = 'es';
      }
      if (!contenido) return Response.error();
      let textos;
      try { textos = (await contenido.json()).general; } catch (_) { return Response.error(); }
      if (!textos || !['sin_conexion_titulo', 'sin_conexion_mensaje', 'reintentar'].every(key => typeof textos[key] === 'string')) {
        return Response.error();
      }
      const titulo = textoSeguro(textos.sin_conexion_titulo);
      const mensaje = textoSeguro(textos.sin_conexion_mensaje);
      const reintentar = textoSeguro(textos.reintentar);
      const html = `<!doctype html><html lang="${idioma}"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>${titulo}</title></head><body><main><h1>${titulo}</h1><p>${mensaje}</p><p><a href="${scopePath}">${reintentar}</a></p></main></body></html>`;
      return new Response(html, { status: 503, headers: {
        'Content-Type': 'text/html; charset=utf-8',
        'Cache-Control': 'no-store',
        'Referrer-Policy': 'no-referrer',
        'Content-Security-Policy': "default-src 'none'; base-uri 'none'; form-action 'none'"
      } });
    }

    self.addEventListener('install', event => {
      event.waitUntil(prepararCatalogos().then(() => self.skipWaiting()));
    });
    self.addEventListener('activate', event => {
      event.waitUntil((async () => {
        for (const key of await caches.keys()) {
          if (key.startsWith(cachePrefix) && key !== cacheName) await caches.delete(key);
        }
        await self.clients.claim();
      })());
    });
    self.addEventListener('fetch', event => {
      const request = event.request;
      if (request.method !== 'GET') return;
      const url = new URL(request.url);
      if (url.origin !== origin) return;
      if (request.mode === 'navigate') {
        if (url.pathname.startsWith(scopePath)) {
          event.respondWith(fetch(request).catch(() => sinConexion(request)));
        }
        return;
      }
      const recurso = recursoPermitido(request, origin, scopePath);
      if (recurso) event.respondWith(recursoEstatico(request, recurso));
    });
  }

  self.iniciarVECPublicWorker = iniciar;
})();
