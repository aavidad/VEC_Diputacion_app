import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import vm from 'node:vm';

const source = await readFile(new URL('./sw-public-assets.js', import.meta.url), 'utf8');
const configs = Object.fromEntries(await Promise.all([
  ['empleado', '../portal-empleado/cache-publica-v1.json'],
  ['personal', '../area-personal/cache-publica-v1.json'],
  ['admin', '../administracion-perfiles/cache-publica-v1.json']
].map(async ([portal, ruta]) => [portal, JSON.parse(await readFile(new URL(ruta, import.meta.url), 'utf8'))])));
const VERSION = configs.empleado.version;
const PRECACHE = 4;

function crearEntorno(portal = 'empleado', opciones = {}) {
  const scopes = { empleado: '/portal-empleado/', personal: '/area-personal/', admin: '/administracion-perfiles/' };
  const handlers = new Map();
  const almacen = opciones.almacen || new Map();
  const llamadas = [];
  const config = opciones.config || configs[portal];
  const indice = opciones.indice || {
    por_defecto: 'es', idiomas: [{ codigo: 'es' }, { codigo: 'en' }]
  };
  let respuesta = url => {
    const path = new URL(url).pathname;
    const idioma = path.match(/^\/textos\/([^/]+)\/pwa\.json$/)?.[1];
    const tipo = path.endsWith('.json') ? 'application/json' : path.endsWith('.css') ? 'text/css' : 'text/javascript';
    return {
      status: 200, ok: true, type: 'basic', redirected: false, url,
      headers: new Headers({ 'Content-Type': tipo, ...(tipo === 'application/json' ? { 'Cache-Control': 'no-store' } : {}) }),
      clone() { return this; },
      async json() {
        if (path.endsWith('/cache-publica-v1.json')) return config;
        if (path === '/textos/idiomas.json') return indice;
        return { general: {
          sin_conexion_titulo: `Sin conexión ${idioma}`,
          sin_conexion_mensaje: 'Compruebe la conexión y vuelva a intentarlo.',
          reintentar: 'Reintentar'
        } };
      }
    };
  };
  const caches = {
    async open(name) {
      if (!almacen.has(name)) almacen.set(name, new Map());
      const entradas = almacen.get(name);
      return {
        async put(url, value) { entradas.set(String(url), value); },
        async match(url) { return entradas.get(String(url)); }
      };
    },
    async keys() { return [...almacen.keys()]; },
    async delete(name) { return almacen.delete(name); }
  };
  const self = {
    location: { origin: 'https://vec.example', href: `https://vec.example${scopes[portal]}sw.js?v=${VERSION}` },
    registration: { scope: `https://vec.example${scopes[portal]}` },
    clients: { async claim() {} }, async skipWaiting() {},
    addEventListener(tipo, callback) { handlers.set(tipo, callback); }
  };
  vm.runInNewContext(source, { self, URL, Headers, Response, caches,
    fetch: async input => {
      const url = typeof input === 'string' ? input : input.url;
      llamadas.push(url);
      return respuesta(url, input);
    }
  });
  self.iniciarVECPublicWorker(portal);
  async function lanzar(tipo, request) {
    let promesa;
    handlers.get(tipo)({ request, waitUntil(value) { promesa = value; }, respondWith(value) { promesa = value; } });
    return promesa ? await promesa : undefined;
  }
  const solicitud = (path, options = {}) => ({
    url: new URL(path, 'https://vec.example').href,
    method: options.method || 'GET', mode: options.mode || 'cors',
    destination: options.destination || '', headers: new Headers(options.headers || {})
  });
  return { lanzar, solicitud, caches, almacen, llamadas, ponerRespuesta(fn) { respuesta = fn; } };
}

test('instala únicamente política e idiomas aprobados y limpia versiones del mismo portal', async () => {
  const app = crearEntorno();
  await app.lanzar('install');
  const nombre = `vec-pwa-empleado-public-${VERSION}`;
  assert.deepEqual([...app.almacen.get(nombre).keys()], [
    `https://vec.example/portal-empleado/cache-publica-v1.json?v=${VERSION}`,
    `https://vec.example/textos/idiomas.json?v=${VERSION}`,
    `https://vec.example/textos/es/pwa.json?v=${VERSION}`,
    `https://vec.example/textos/en/pwa.json?v=${VERSION}`
  ]);
  const anterior = await app.caches.open('vec-pwa-empleado-public-20261002-pwa-v1');
  await anterior.put('https://vec.example/textos/es/preferencias.json?v=1', { ok: true });
  await app.caches.open('vec-pwa-empleado-public-20261002-pwa-v2');
  await app.caches.open('vec-pwa-empleado-public-20261002-pwa-v3');
  await app.caches.open('vec-pwa-empleado-public-20261002-pwa-v4');
  await app.caches.open('vec-pwa-personal-public-anterior');
  await app.lanzar('activate');
  assert.deepEqual(await app.caches.keys(), [nombre, 'vec-pwa-personal-public-anterior']);
});

test('ADMIN conserva el scope privado y su CSS aprobado', async () => {
  const app = crearEntorno('admin');
  await app.lanzar('install');
  const css = app.solicitud(configs.admin.propios[0], { destination: 'style' });
  await app.lanzar('fetch', css);
  assert.equal(app.almacen.get(`vec-pwa-admin-public-${VERSION}`).has(css.url), true);
  assert.equal(await app.lanzar('fetch', app.solicitud('/admin/modulos/arranque.js?v=1')), undefined);
});

test('almacena una sola vez el asset exacto aprobado; un JS futuro del mismo directorio no entra', async () => {
  const app = crearEntorno();
  await app.lanzar('install');
  const js = app.solicitud(configs.empleado.propios[2], { destination: 'script' });
  await app.lanzar('fetch', js);
  await app.lanzar('fetch', js);
  assert.equal(app.llamadas.filter(url => url === js.url).length, 1);
  const nuevo = app.solicitud('/portal-empleado/nuevo-modulo.js?v=20991231', { destination: 'script' });
  assert.equal(await app.lanzar('fetch', nuevo), undefined);
  assert.equal(app.almacen.get(`vec-pwa-empleado-public-${VERSION}`).size, PRECACHE + 1);
});

test('API, documentos, HTML, escritura, consultas extra y otros portales pasan a la red', async () => {
  const app = crearEntorno();
  await app.lanzar('install');
  const rutas = [
    '/api/vec/session?v=1', '/portal-empleado/documentos/recibo.pdf?v=1',
    '/portal-empleado/index.html?v=1', '/portal-empleado/portal.js?perfil=rrhh',
    '/portal-empleado/portal.js?v=1&usuario=1', '/portal-empleado/portal.js',
    '/portal-empleado/portal.test.mjs?v=1', '/portal-empleado/sw.js?v=1',
    configs.personal.propios[1], 'https://otro.example/portal-empleado/portal.js?v=1'
  ];
  for (const ruta of rutas) assert.equal(await app.lanzar('fetch', app.solicitud(ruta)), undefined, ruta);
  assert.equal(await app.lanzar('fetch', app.solicitud(configs.empleado.propios[2], { method: 'POST' })), undefined);
  assert.equal(app.almacen.get(`vec-pwa-empleado-public-${VERSION}`).size, PRECACHE);
});

test('rechaza respuesta redirigida, HTML disfrazado, privada, opaca o de otro origen', async () => {
  const app = crearEntorno();
  await app.lanzar('install');
  const url = configs.empleado.propios[0];
  for (const cambio of [
    { redirected: true },
    { headers: new Headers({ 'Content-Type': 'text/html' }) },
    { headers: new Headers({ 'Content-Type': 'text/css', 'Cache-Control': 'private' }) },
    { type: 'opaque' }, { url: 'https://otro.example/portal.css?v=1' }
  ]) {
    app.ponerRespuesta(valor => ({ status: 200, ok: true, type: 'basic', redirected: false,
      url: valor, headers: new Headers({ 'Content-Type': 'text/css' }), clone() { return this; }, ...cambio }));
    await app.lanzar('fetch', app.solicitud(url, { destination: 'style' }));
  }
  assert.equal(app.almacen.get(`vec-pwa-empleado-public-${VERSION}`).size, PRECACHE);
});

test('Vary sólo admite vacío o Accept-Encoding único para la caché pública', async () => {
  for (const [vary, admitida] of [
    ['', true], ['Accept-Encoding', true], ['accept-encoding', true],
    ['*', false], ['Cookie', false], ['Authorization', false], ['X-Persona', false],
    ['Accept-Encoding, Cookie', false], ['Accept-Encoding, Accept-Encoding', false],
  ]) {
    const app = crearEntorno();
    await app.lanzar('install');
    app.ponerRespuesta(url => ({
      status: 200, ok: true, type: 'basic', redirected: false, url,
      headers: new Headers({ 'Content-Type': 'text/css', Vary: vary }), clone() { return this; }
    }));
    const request = app.solicitud(configs.empleado.propios[0], { destination: 'style' });
    await app.lanzar('fetch', request);
    assert.equal(app.almacen.get(`vec-pwa-empleado-public-${VERSION}`).has(request.url), admitida, vary);
  }
});

test('un catálogo offline marcado private se rechaza; otros JSON no figuran en la política', async () => {
  const app = crearEntorno();
  await app.lanzar('install');
  const cache = app.almacen.get(`vec-pwa-empleado-public-${VERSION}`);
  const catalogo = `https://vec.example/textos/es/pwa.json?v=${VERSION}`;
  cache.delete(catalogo);
  app.ponerRespuesta(valor => ({ status: 200, ok: true, type: 'basic', redirected: false,
    url: valor, headers: new Headers({ 'Content-Type': 'application/json', 'Cache-Control': 'private, no-store' }),
    clone() { return this; }
  }));
  await app.lanzar('fetch', app.solicitud(catalogo));
  assert.equal(cache.has(catalogo), false);
  assert.equal(await app.lanzar('fetch', app.solicitud('/textos/es/preferencias.json?v=1')), undefined);
});

test('un fallo al pedir el estático aprobado vuelve a la red original sin cachearlo', async () => {
  const app = crearEntorno();
  await app.lanzar('install');
  app.ponerRespuesta((valor, input) => {
    if (typeof input === 'string') throw new TypeError('redirección rechazada');
    return { status: 200, ok: true, url: valor };
  });
  const recurso = app.solicitud(configs.empleado.propios[2], { destination: 'script' });
  const response = await app.lanzar('fetch', recurso);
  assert.equal(response.url, recurso.url);
  assert.equal(app.llamadas.filter(url => url === recurso.url).length, 2);
  assert.equal(app.almacen.get(`vec-pwa-empleado-public-${VERSION}`).size, PRECACHE);
});

test('la página sin conexión deriva tercer idioma y nuevo predeterminado del índice', async () => {
  const indice = { por_defecto: 'fr', idiomas: [{ codigo: 'es' }, { codigo: 'en' }, { codigo: 'fr' }] };
  const app = crearEntorno('personal', { indice });
  await app.lanzar('install');
  const privada = app.solicitud('/area-personal/?vista=perfil&persona=123', { mode: 'navigate',
    headers: { 'Accept-Language': 'fr-FR,fr;q=0.9' } });
  const online = await app.lanzar('fetch', privada);
  assert.equal(online.url, privada.url);
  app.ponerRespuesta(() => { throw new TypeError('sin red'); });
  const offline = await app.lanzar('fetch', privada);
  const html = await offline.text();
  assert.equal(offline.status, 503);
  assert.equal(offline.headers.get('Cache-Control'), 'no-store');
  assert.match(html, /lang="fr"/);
  assert.match(html, /Sin conexión fr/);
  assert.doesNotMatch(html, /persona=123/);
  const sinPreferencia = await app.lanzar('fetch', app.solicitud('/area-personal/', { mode: 'navigate' }));
  assert.match(await sinPreferencia.text(), /Sin conexión fr/);
  assert.equal(app.almacen.get(`vec-pwa-personal-public-${VERSION}`).size, PRECACHE + 1);
});

test('respeta el código regional canónico y recupera política del CacheStorage tras reiniciar', async () => {
  const indice = { por_defecto: 'es', idiomas: [{ codigo: 'es' }, { codigo: 'pt-BR' }] };
  const primera = crearEntorno('personal', { indice });
  await primera.lanzar('install');
  const reiniciada = crearEntorno('personal', { almacen: primera.almacen });
  reiniciada.ponerRespuesta(() => { throw new TypeError('sin red'); });
  const respuesta = await reiniciada.lanzar('fetch', reiniciada.solicitud('/area-personal/?vista=perfil', {
    mode: 'navigate', headers: { 'Accept-Language': 'pt-BR,pt;q=0.9' }
  }));
  const html = await respuesta.text();
  assert.match(html, /lang="pt-BR"/);
  assert.match(html, /Sin conexión pt-BR/);
  assert.equal(primera.almacen.get(`vec-pwa-personal-public-${VERSION}`).size, PRECACHE);
});

test('la preferencia lang explícita precede al navegador y se conserva al reintentar', async () => {
  const app = crearEntorno('personal');
  await app.lanzar('install');
  app.ponerRespuesta(() => { throw new TypeError('sin red'); });
  const en = await app.lanzar('fetch', app.solicitud('/area-personal/?lang=en&vista=perfil', {
    mode: 'navigate', headers: { 'Accept-Language': 'es-ES' }
  }));
  const enHTML = await en.text();
  assert.match(enHTML, /lang="en"/);
  assert.match(enHTML, /href="\/area-personal\/\?lang=en"/);
  assert.doesNotMatch(enHTML, /vista=perfil/);
  const es = await app.lanzar('fetch', app.solicitud('/area-personal/?lang=es', {
    mode: 'navigate', headers: { 'Accept-Language': 'en-GB' }
  }));
  assert.match(await es.text(), /lang="es"/);
});

test('una configuración que intente añadir API se rechaza en la instalación', async () => {
  const config = structuredClone(configs.empleado);
  config.comunes.push('/api/vec/private.js?v=1');
  const app = crearEntorno('empleado', { config });
  await assert.rejects(app.lanzar('install'), /política pública PWA no válida/);
});

test('config pública y helper común no contienen nombres de scopes internos', () => {
  assert.doesNotMatch(source, /\/(?:portal-empleado|administracion-perfiles)\//);
  assert.doesNotMatch(JSON.stringify(configs.personal), /\/(?:portal-empleado|administracion-perfiles)\//);
  assert.deepEqual([configs.personal.version, configs.admin.version], [VERSION, VERSION]);
});

test('una config propia no puede declarar activos de otro scope', async () => {
  const config = structuredClone(configs.personal);
  config.propios.push(configs.empleado.propios[0]);
  await assert.rejects(crearEntorno('personal', { config }).lanzar('install'), /política pública PWA no válida/);
});
