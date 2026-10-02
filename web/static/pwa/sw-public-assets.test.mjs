import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import vm from 'node:vm';

const source = await readFile(new URL('./sw-public-assets.js', import.meta.url), 'utf8');
const VERSION = '20261002-pwa-v1';

function crearEntorno(portal = 'empleado') {
  const scopes = { empleado: '/portal-empleado/', personal: '/area-personal/', admin: '/administracion-perfiles/' };
  const scope = scopes[portal];
  const handlers = new Map();
  const almacen = new Map();
  const llamadas = [];
  let respuesta = (url) => ({
    status: 200, ok: true, type: 'basic', redirected: false, url,
    headers: new Headers({ 'Content-Type': url.endsWith('.json?v=' + VERSION) ? 'application/json' :
      url.includes('.css?') ? 'text/css' : 'text/javascript' }),
    clone() { return this; },
    async json() { return { general: {
      sin_conexion_titulo: 'Sin conexión',
      sin_conexion_mensaje: 'Compruebe la conexión y vuelva a intentarlo.',
      reintentar: 'Reintentar'
    } }; }
  });
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
    location: { origin: 'https://vec.example' },
    registration: { scope: `https://vec.example${scope}` },
    clients: { async claim() {} },
    async skipWaiting() {},
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
    destination: options.destination || '',
    headers: new Headers(options.headers || {})
  });
  return { lanzar, solicitud, caches, almacen, llamadas, ponerRespuesta(fn) { respuesta = fn; } };
}

test('instala solo los catálogos públicos y limpia únicamente versiones de su portal', async () => {
  const app = crearEntorno();
  await app.lanzar('install');
  const nombre = `vec-pwa-empleado-public-${VERSION}`;
  assert.deepEqual([...app.almacen.get(nombre).keys()], [
    `https://vec.example/textos/es/pwa.json?v=${VERSION}`,
    `https://vec.example/textos/en/pwa.json?v=${VERSION}`
  ]);
  await app.caches.open('vec-pwa-empleado-public-anterior');
  await app.caches.open('vec-pwa-personal-public-anterior');
  await app.lanzar('activate');
  assert.deepEqual(await app.caches.keys(), [nombre, 'vec-pwa-personal-public-anterior']);
});

test('ADMIN mantiene el scope del proceso privado y no almacena activos de otros portales', async () => {
  const app = crearEntorno('admin');
  await app.lanzar('install');
  const script = app.solicitud('/administracion-perfiles/arranque.js?v=1', { destination: 'script' });
  await app.lanzar('fetch', script);
  assert.equal(app.almacen.get(`vec-pwa-admin-public-${VERSION}`).has(script.url), true);
  assert.equal(await app.lanzar('fetch', app.solicitud('/admin/modulos/arranque.js?v=1', { destination: 'script' })), undefined);
});

test('guarda solo una respuesta pública estática versionada del mismo origen', async () => {
  const app = crearEntorno();
  await app.lanzar('install');
  const js = app.solicitud('/portal-empleado/portal.js?v=20260923-p4-reintento-v2', { destination: 'script' });
  await app.lanzar('fetch', js);
  await app.lanzar('fetch', js);
  assert.equal(app.llamadas.filter(url => url === js.url).length, 1);
  const entradas = app.almacen.get(`vec-pwa-empleado-public-${VERSION}`);
  assert.equal(entradas.has(js.url), true);
});

test('API, documentos, HTML, métodos de escritura, consultas ajenas y otros portales pasan a la red', async () => {
  const app = crearEntorno();
  await app.lanzar('install');
  const rutas = [
    '/api/vec/session?v=1',
    '/portal-empleado/documentos/recibo.pdf?v=1',
    '/portal-empleado/index.html?v=1',
    '/portal-empleado/portal.js?perfil=rrhh',
    '/portal-empleado/portal.js?v=1&usuario=1',
    '/portal-empleado/portal.js',
    '/portal-empleado/portal.test.mjs?v=1',
    '/portal-empleado/sw.js?v=1',
    '/area-personal/arranque.js?v=1',
    'https://otro.example/portal-empleado/portal.js?v=1'
  ];
  for (const ruta of rutas) assert.equal(await app.lanzar('fetch', app.solicitud(ruta)), undefined, ruta);
  assert.equal(await app.lanzar('fetch', app.solicitud('/portal-empleado/portal.js?v=1', { method: 'POST' })), undefined);
  assert.equal(app.almacen.get(`vec-pwa-empleado-public-${VERSION}`).size, 2);
});

test('rechaza redirecciones, HTML disfrazado, respuestas privadas y tipos opacos', async () => {
  const app = crearEntorno();
  await app.lanzar('install');
  const url = '/portal-empleado/portal.css?v=1';
  for (const cambio of [
    { redirected: true },
    { headers: new Headers({ 'Content-Type': 'text/html' }) },
    { headers: new Headers({ 'Content-Type': 'text/css', 'Cache-Control': 'private' }) },
    { type: 'opaque' },
    { url: 'https://otro.example/portal.css?v=1' }
  ]) {
    app.ponerRespuesta(valor => ({ status: 200, ok: true, type: 'basic', redirected: false,
      url: valor, headers: new Headers({ 'Content-Type': 'text/css' }), clone() { return this; }, ...cambio }));
    await app.lanzar('fetch', app.solicitud(url, { destination: 'style' }));
  }
  assert.equal(app.almacen.get(`vec-pwa-empleado-public-${VERSION}`).size, 2);
});

test('un catálogo i18n privado no se almacena, aunque sea JSON público por ruta', async () => {
  const app = crearEntorno();
  await app.lanzar('install');
  const url = '/textos/es/preferencias.json?v=1';
  app.ponerRespuesta(valor => ({ status: 200, ok: true, type: 'basic', redirected: false,
    url: valor, headers: new Headers({ 'Content-Type': 'application/json', 'Cache-Control': 'private, no-store' }),
    clone() { return this; }
  }));
  await app.lanzar('fetch', app.solicitud(url));
  assert.equal(app.almacen.get(`vec-pwa-empleado-public-${VERSION}`).size, 2);
});

test('si la carga anónima del estático falla, recupera la petición de red sin almacenarla', async () => {
  const app = crearEntorno();
  await app.lanzar('install');
  app.ponerRespuesta((valor, input) => {
    if (typeof input === 'string') throw new TypeError('redirección rechazada');
    return { status: 200, ok: true, url: valor };
  });
  const recurso = app.solicitud('/portal-empleado/portal.js?v=1', { destination: 'script' });
  const response = await app.lanzar('fetch', recurso);
  assert.equal(response.url, recurso.url);
  assert.equal(app.llamadas.filter(url => url === recurso.url).length, 2);
  assert.equal(app.almacen.get(`vec-pwa-empleado-public-${VERSION}`).size, 2);
});

test('las navegaciones consultan la red; sin conexión generan HTML sin guardarlo', async () => {
  const app = crearEntorno('personal');
  await app.lanzar('install');
  const privada = app.solicitud('/area-personal/?vista=perfil&persona=123', { mode: 'navigate', headers: { 'Accept-Language': 'es' } });
  const online = await app.lanzar('fetch', privada);
  assert.equal(online.url, privada.url);
  app.ponerRespuesta(() => { throw new TypeError('sin red'); });
  const offline = await app.lanzar('fetch', privada);
  assert.equal(offline.status, 503);
  assert.equal(offline.headers.get('Cache-Control'), 'no-store');
  const html = await offline.text();
  assert.match(html, /Sin conexión/);
  assert.doesNotMatch(html, /persona=123/);
  assert.equal(app.almacen.get(`vec-pwa-personal-public-${VERSION}`).size, 2);
});
