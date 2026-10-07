/* Política pública, finita y separada por portal. Ningún documento entra en CacheStorage. */
(function () {
  'use strict';

  const VERSION = '20261003-pwa-ci-v5';
  const IDIOMAS = `/textos/idiomas.json?v=${VERSION}`;
  const TIPOS = Object.freeze({
    css: /^text\/css(?:;|$)/i,
    js: /^(?:text|application)\/javascript(?:;|$)/i,
    json: /^application\/json(?:;|$)/i,
    png: /^image\/png(?:;|$)/i,
    ico: /^(?:image\/(?:x-icon|vnd\.microsoft\.icon)|application\/octet-stream)(?:;|$)/i,
    woff: /^font\/woff(?:;|$)/i,
    woff2: /^font\/woff2(?:;|$)/i
  });

  function versionValida(url) {
    const parametros = [...url.searchParams.entries()];
    return parametros.length === 1 && parametros[0][0] === 'v' &&
      /^[a-z0-9][a-z0-9._-]{0,79}$/i.test(parametros[0][1]);
  }

  function urlCanonica(ruta, origen) {
    if (typeof ruta !== 'string' || !ruta.startsWith('/') || /[%\\#]/.test(ruta)) return null;
    const url = new URL(ruta, origen);
    return url.href === origen + ruta && versionValida(url) ? url : null;
  }

  function tipoAsset(url, scopePath) {
    const path = url.pathname;
    if (!/^\/[a-z0-9/_.-]+\.[a-z0-9]+$/i.test(path) || /\/\.|\/\//.test(path)) return null;
    if (/\.(?:test|spec)\.|(?:^|\/)(?:api|documentos|test|tests|fixtures|datos-presentacion|adaptador-presentacion)(?:\/|\.)/i.test(path)) return null;
    if (!path.startsWith('/pwa/') && !path.startsWith(scopePath)) return null;
    if (path.endsWith('/sw.js') || path.endsWith('/sw-public-assets.js')) return null;
    const extension = path.slice(path.lastIndexOf('.') + 1).toLowerCase();
    return extension in TIPOS && extension !== 'json' ? extension : null;
  }

  function indiceValido(datos) {
    if (!datos || !Array.isArray(datos.idiomas) || datos.idiomas.length < 1 || datos.idiomas.length > 12) return null;
    const idiomas = datos.idiomas.map(item => item?.codigo);
    if (idiomas.some(codigo => typeof codigo !== 'string' || !/^[a-z]{2,8}(?:-[a-z0-9]{2,8})?$/i.test(codigo))) return null;
    if (new Set(idiomas.map(codigo => codigo.toLowerCase())).size !== idiomas.length || !idiomas.includes(datos.por_defecto)) return null;
    return { idiomas, porDefecto: datos.por_defecto };
  }

  function politicaValida(config, indice, scopePath, origen, configURL) {
    if (!config || config.version !== VERSION || !Array.isArray(config.comunes) || !Array.isArray(config.propios) ||
      Object.keys(config).sort().join(',') !== 'comunes,propios,version') return null;
    const entradas = [...config.comunes, ...config.propios];
    if (entradas.length > 80 || new Set(entradas).size !== entradas.length) return null;
    const assets = new Map();
    for (const [grupo, rutas] of [['comunes', config.comunes], ['propios', config.propios]]) {
      for (const ruta of rutas) {
        const url = urlCanonica(ruta, origen);
        const tipo = url && tipoAsset(url, scopePath);
        if (!tipo || (grupo === 'comunes' && !url.pathname.startsWith('/pwa/')) ||
          (grupo === 'propios' && !url.pathname.startsWith(scopePath))) return null;
        assets.set(url.href, tipo);
      }
    }
    const idiomas = indiceValido(indice);
    if (!idiomas) return null;
    const json = new Set([configURL.href, origen + IDIOMAS]);
    for (const codigo of idiomas.idiomas) json.add(`${origen}/textos/${codigo}/pwa.json?v=${VERSION}`);
    return { assets, json, idiomas };
  }

  function respuestaPublica(response, url, tipo, permiteNoStore) {
    if (!response || response.status !== 200 || !response.ok || response.redirected || response.type !== 'basic') return false;
    if (response.url !== url.href || !TIPOS[tipo].test(response.headers.get('Content-Type') || '')) return false;
    const control = response.headers.get('Cache-Control') || '';
    if (/\bprivate\b/i.test(control) || (!permiteNoStore && /\bno-store\b/i.test(control))) return false;
    if (response.headers.has('Set-Cookie') || response.headers.has('Content-Disposition')) return false;
    const variacion = (response.headers.get('Vary') || '').trim();
    if (variacion && !/^accept-encoding$/i.test(variacion)) return false;
    return true;
  }

  function textoSeguro(valor) {
    return String(valor).replace(/[&<>"']/g, caracter => ({
      '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'
    }[caracter]));
  }

  function iniciar(portal) {
    if (typeof portal !== 'string' || !/^[a-z0-9_-]{1,20}$/.test(portal)) throw new Error('identificador PWA inválido');
    const origen = self.location.origin;
    const scope = new URL(self.registration.scope);
    const scopePath = scope.pathname;
    const script = new URL(self.location.href);
    if (scope.origin !== origen || !/^\/[a-z0-9/_-]+\/$/i.test(scopePath) || scopePath.includes('//') ||
      script.origin !== origen || script.pathname !== `${scopePath}sw.js`) throw new Error('scope PWA inválido');
    const configURL = new URL(`${origen}${scopePath}cache-publica-v1.json?v=${VERSION}`);
    const cacheName = `vec-pwa-${portal}-public-${VERSION}`;
    const cachePrefix = `vec-pwa-${portal}-public-`;
    let politica = null;

    async function pedir(url, tipo, permiteNoStore = false) {
      const response = await fetch(url.href, {
        cache: 'no-store', credentials: 'omit', redirect: 'error', referrerPolicy: 'no-referrer'
      });
      return respuestaPublica(response, url, tipo, permiteNoStore) ? response : null;
    }

    async function instalar() {
      const idiomasURL = new URL(origen + IDIOMAS);
      const configResponse = await pedir(configURL, 'json', true);
      const idiomasResponse = await pedir(idiomasURL, 'json', true);
      if (!configResponse || !idiomasResponse) throw new Error('catálogos públicos PWA no disponibles');
      const config = await configResponse.clone().json();
      const indice = await idiomasResponse.clone().json();
      const aprobada = politicaValida(config, indice, scopePath, origen, configURL);
      if (!aprobada) throw new Error('política pública PWA no válida');
      const cache = await caches.open(cacheName);
      await cache.put(configURL.href, configResponse);
      await cache.put(idiomasURL.href, idiomasResponse);
      for (const codigo of aprobada.idiomas.idiomas) {
        const url = new URL(`${origen}/textos/${codigo}/pwa.json?v=${VERSION}`);
        const response = await pedir(url, 'json', true);
        if (!response) throw new Error('catálogo público PWA no disponible');
        await cache.put(url.href, response);
      }
      politica = aprobada;
      await self.skipWaiting();
    }

    async function recuperarPolitica() {
      if (politica) return politica;
      const cache = await caches.open(cacheName);
      const config = await cache.match(configURL.href);
      const idiomas = await cache.match(origen + IDIOMAS);
      if (!config || !idiomas) return null;
      try { politica = politicaValida(await config.json(), await idiomas.json(), scopePath, origen, configURL); }
      catch (_) { return null; }
      return politica;
    }

    async function recursoEstatico(request, url) {
      const aprobada = await recuperarPolitica();
      const tipo = aprobada?.assets.get(url.href) || (aprobada?.json.has(url.href) ? 'json' : null);
      if (!tipo) return fetch(request);
      const cache = await caches.open(cacheName);
      const guardado = await cache.match(url.href);
      if (guardado) return guardado;
      let response;
      try { response = await pedir(url, tipo, tipo === 'json' && aprobada.json.has(url.href)); }
      catch (_) { return fetch(request); }
      if (!response) return fetch(request);
      try { await cache.put(url.href, response.clone()); } catch (_) { /* Sin cuota: sigue la red. */ }
      return response;
    }

    function elegirIdioma(acceptLanguage, idiomas) {
      const disponibles = new Map(idiomas.idiomas.map(codigo => [codigo.toLowerCase(), codigo]));
      for (const parte of (acceptLanguage || '').split(',')) {
        const solicitado = parte.split(';')[0].trim().toLowerCase();
        if (disponibles.has(solicitado)) return disponibles.get(solicitado);
        const base = solicitado.split('-')[0];
        if (disponibles.has(base)) return disponibles.get(base);
      }
      return idiomas.porDefecto;
    }

    function idiomaDeNavegacion(request, idiomas) {
      const url = new URL(request.url);
      const solicitados = url.searchParams.getAll('lang');
      if (solicitados.length === 1) {
        const encontrado = idiomas.idiomas.find(codigo => codigo.toLowerCase() === solicitados[0].toLowerCase());
        if (encontrado) return encontrado;
      }
      return elegirIdioma(request.headers.get('Accept-Language'), idiomas);
    }

    async function sinConexion(request) {
      const aprobada = await recuperarPolitica();
      if (!aprobada) return Response.error();
      const cache = await caches.open(cacheName);
      let idioma = idiomaDeNavegacion(request, aprobada.idiomas);
      let contenido = await cache.match(`${origen}/textos/${idioma}/pwa.json?v=${VERSION}`);
      if (!contenido && idioma !== aprobada.idiomas.porDefecto) {
        idioma = aprobada.idiomas.porDefecto;
        contenido = await cache.match(`${origen}/textos/${idioma}/pwa.json?v=${VERSION}`);
      }
      if (!contenido) return Response.error();
      let textos;
      try { textos = (await contenido.json()).general; } catch (_) { return Response.error(); }
      if (!textos || !['sin_conexion_titulo', 'sin_conexion_mensaje', 'reintentar'].every(clave => typeof textos[clave] === 'string')) return Response.error();
      const titulo = textoSeguro(textos.sin_conexion_titulo);
      const mensaje = textoSeguro(textos.sin_conexion_mensaje);
      const reintentar = textoSeguro(textos.reintentar);
      const html = `<!doctype html><html lang="${idioma}"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>${titulo}</title></head><body><main><h1>${titulo}</h1><p>${mensaje}</p><p><a href="${scopePath}?lang=${encodeURIComponent(idioma)}">${reintentar}</a></p></main></body></html>`;
      return new Response(html, { status: 503, headers: {
        'Content-Type': 'text/html; charset=utf-8',
        'Cache-Control': 'no-store',
        'Referrer-Policy': 'no-referrer',
        'Content-Security-Policy': "default-src 'none'; base-uri 'none'; form-action 'none'"
      } });
    }

    self.addEventListener('install', event => event.waitUntil(instalar()));
    self.addEventListener('activate', event => {
      event.waitUntil((async () => {
        for (const key of await caches.keys()) {
          if (key.startsWith(cachePrefix) && key !== cacheName) await caches.delete(key);
        }
        await recuperarPolitica();
        await self.clients.claim();
      })());
    });
    self.addEventListener('fetch', event => {
      const request = event.request;
      if (request.method !== 'GET') return;
      const url = new URL(request.url);
      if (url.origin !== origen) return;
      if (request.mode === 'navigate') {
        if (url.pathname.startsWith(scopePath)) event.respondWith(fetch(request).catch(() => sinConexion(request)));
        return;
      }
      if (!versionValida(url)) return;
      if (politica && !politica.assets.has(url.href) && !politica.json.has(url.href)) return;
      if (!politica && !tipoAsset(url, scopePath) && url.href !== configURL.href && url.href !== origen + IDIOMAS && !/^\/textos\//.test(url.pathname)) return;
      event.respondWith(recursoEstatico(request, url));
    });
  }

  self.iniciarVECPublicWorker = iniciar;
})();
