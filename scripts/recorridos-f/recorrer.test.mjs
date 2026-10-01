import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { validarOrigen, solicitudPermitida, rutaExterna, prepararSalida, CUADRO, DETALLE } from './config.mjs';
import { interceptar } from './recorrer.mjs';

const origen = 'https://127.0.0.1:8443';
const request = (ruta, method = 'GET', body) => ({ url: () => origen + ruta, method: () => method, postData: () => body === undefined ? null : JSON.stringify(body) });

test('origen numérico loopback: sin DNS, destinos remotos, credenciales o consultas', () => {
  assert.equal(validarOrigen(origen), origen);
  assert.equal(validarOrigen('https://[::1]:8443'), 'https://[::1]:8443');
  for (const v of ['https://cidonia.cloud', 'https://localhost:8443', 'http://127.0.0.1:8443', origen + '?token=secreto', 'https://usuario:clave@127.0.0.1:8443']) assert.throws(() => validarOrigen(v));
});

test('POST solo en las dos consultas CT con cuerpos de lectura', () => {
  assert.equal(solicitudPermitida(request(DETALLE, 'POST', { expediente_ref: 'expediente:sintetico', version_observada: 7 }), origen), true);
  assert.equal(solicitudPermitida(request(CUADRO, 'POST', { filtros: { texto: '', estado_clave: '', fase_clave: '' }, paginacion: { limite: 10, cursor: '' } }), origen), true);
  for (const r of [request('/api/vec/contratacion-temporal/solicitudes', 'POST', {}), request(DETALLE, 'DELETE'), request(DETALLE, 'POST', { expediente_ref: 'expediente:sintetico', version_observada: -1 }), request(DETALLE + '?accion=alta', 'POST', {}), request(CUADRO, 'POST', { filtros: {}, paginacion: { limite: 0 } })]) assert.equal(solicitudPermitida(r, origen), false);
});

test('petición externa se aborta antes de fetch; POST de efecto también', async () => {
  for (const req of [{ url: () => 'https://127.0.0.1:8444/', method: () => 'GET' }, request('/api/vec/bolsa/llamamientos', 'POST', {})]) {
    let fetches = 0, aborts = 0;
    const datos = { bloqueadas: 0, red_fallida: 0 };
    await interceptar({ request: () => req, fetch: () => { fetches++; }, abort: async () => { aborts++; } }, origen, datos);
    assert.equal(fetches, 0); assert.equal(aborts, 1); assert.equal(datos.bloqueadas, 1);
  }
});

test('redirecciones y Set-Cookie se cortan sin fulfill', async () => {
  for (const [status, headers] of [[302, []], [200, [{ name: 'Set-Cookie', value: 'secreto' }]]]) {
    let aborts = 0;
    await interceptar({ request: () => request('/portal-empleado/'), fetch: async options => {
      assert.equal(options.maxRedirects, 0); assert.equal(options.maxRetries, 0);
      return { status: () => status, url: () => origen + '/portal-empleado/', headersArray: async () => headers };
    }, abort: async () => { aborts++; }, fulfill: () => assert.fail('no debe entregar respuesta') }, origen, { bloqueadas: 0, red_fallida: 0 });
    assert.equal(aborts, 1);
  }
});

test('config y evidencia excluyen Git, enlaces, archivos compartidos y sobrescritura', () => {
  const base = path.join(os.homedir(), '.local/state');
  const dir = fs.mkdtempSync(path.join(base, 'vec-f-prueba-'));
  try {
    fs.chmodSync(dir, 0o700);
    const file = path.join(dir, 'privado.json');
    fs.writeFileSync(file, '{}', { mode: 0o600 });
    assert.equal(rutaExterna(file, { privado: true }), file);
    fs.symlinkSync(file, path.join(dir, 'enlace'));
    assert.throws(() => rutaExterna(path.join(dir, 'enlace')));
    fs.linkSync(file, path.join(dir, 'duro'));
    assert.throws(() => rutaExterna(file));
    fs.unlinkSync(path.join(dir, 'duro'));
    const salida = prepararSalida(path.join(dir, 'salida'));
    assert.equal(fs.statSync(salida).mode & 0o777, 0o700);
    assert.throws(() => prepararSalida(salida));
    fs.mkdirSync(path.join(dir, '.git'));
    assert.throws(() => rutaExterna(file));
  } finally { fs.rmSync(dir, { recursive: true, force: true }); }
});
