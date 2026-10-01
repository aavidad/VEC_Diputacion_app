import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { validarOrigen, solicitudPermitida, rutaExterna, prepararSalida, CUADRO, DETALLE, BORRADORES, FIRMAS } from './config.mjs';
import { interceptar, respuesta, validarLectura, recorrer } from './recorrer.mjs';

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
  const base = process.env.VEC_F_TEST_SCRATCH || os.tmpdir();
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

test('HTTP200 incompatible de historial y CT corta antes de acreditar la lectura', async () => {
  const casos = [
    ['/api/vec/bolsa/mi-bolsa/historial', { data: {} }],
    [DETALLE, { data: { resumen: { expediente_ref: 'expediente:sintetico', version: 1 }, hitos: [] } }],
    [CUADRO, { data: { expedientes: [] } }],
    ['/api/vec/bolsa/mi-bolsa', { data: { participaciones: [{ bolsa: 'bolsa:sintetica' }] } }],
    ['/api/vec/bolsa/bolsas', { data: { bolsas: [{ bolsa_ref: 'bolsa:sintetica' }] } }],
    ['/api/vec/bolsa/bolsas/bolsa:sintetica/candidatos', { data: { bolsa: { bolsa_ref: 'bolsa:sintetica' }, candidatos: [] } }],
  ];
  for (const [ruta, envelope] of casos) {
    const page = { waitForResponse: async () => ({ status: () => 200, json: async () => envelope }) };
    await assert.rejects(respuesta(page, ruta, async () => {}), /^Error: contrato$/);
  }
});

test('los validadores reales admiten historial vacío y detalle CT completos', () => {
  const historico = { data: { esquema: 'vec.bolsa.mi-bolsa.historial.v1', consultada_en: '2026-10-01T00:00:00.000000Z',
    campos_visibles: [], historial: { pagina: 1, tamano: 20, hay_mas: false, items: [] } } };
  assert.equal(validarLectura('/api/vec/bolsa/mi-bolsa/historial', historico).historial.pagina, 1);
  assert.throws(() => validarLectura('/api/vec/bolsa/mi-bolsa/historial', { data: { ...historico.data, historial: { ...historico.data.historial, pagina: 2 } } }));
  const resumen = { expediente_ref: 'expediente:sintetico', numero_visible: '2026/CT-0001', version: 1,
    flujo_ref: 'flujo:ct:general', flujo_version: 1, flujo_huella_sha256: 'a'.repeat(64), fase_clave: 'solicitud',
    estado_clave: 'pendiente', centro_ref: 'centro:001', categoria_ref: 'categoria:auxiliar',
    creado_en: '2026-10-01T00:00:00Z', actualizado_en: '2026-10-01T00:00:00Z' };
  const detalle = { data: { esquema: 'vec.contratacion-temporal.detalle-rrhh.v1', resumen,
    solicitud: { grupo_subgrupo: 'A2', motivo_clave: 'sustitucion', periodo_inicio: '2026-10-01T00:00:00Z', periodo_fin: '2026-10-31T00:00:00Z' },
    hitos: [{ secuencia: 1, version_expediente: 1, accion_clave: 'registrar_solicitud', realizada_en: '2026-10-01T00:00:00Z', fase_destino: 'solicitud', estado_origen: 'pendiente', estado_destino: 'pendiente' }] } };
  assert.equal(validarLectura(DETALLE, detalle).resumen.version, 1);
  assert.equal(validarLectura(CUADRO, { data: { esquema: 'vec.contratacion-temporal.cuadro-rrhh.v1', generada_en: '2026-10-01T00:00:00Z', expedientes: [resumen], hay_mas: false } }).expedientes.length, 1);
});

test('lecturas automáticas de ficha: cuerpos exactos; descarga y registro bloqueados', async () => {
  assert.equal(solicitudPermitida(request(BORRADORES, 'POST', { expediente_ref: 'expediente:sintetico', version_observada: 1 }), origen), true);
  assert.equal(solicitudPermitida(request(FIRMAS, 'POST', { expediente_ref: 'expediente:sintetico' }), origen), true);
  for (const req of [request(BORRADORES, 'POST', { expediente_ref: 'expediente:sintetico', version_observada: 0 }), request(BORRADORES, 'POST', { expediente_ref: 'expediente:sintetico', version_observada: 1, tipo: 'resolucion' }), request(FIRMAS, 'POST', { expediente_ref: 'expediente:sintetico', resultado: 'firmado' }), request(BORRADORES.replace('/disponibles', ''), 'POST', {}), request(FIRMAS.replace('/consultas', ''), 'POST', {})]) assert.equal(solicitudPermitida(req, origen), false);
  const catalogo = { esquema: 'vec.contratacion-temporal.borradores-disponibles.v1', catalogo_ref: 'catalogo:sintetico', catalogo_huella_sha256: 'a'.repeat(64), procedencia_ref: 'recibo:11111111-1111-1111-1111-111111111111', tipos: [] };
  assert.equal(validarLectura(BORRADORES, catalogo).tipos.length, 0);
  const firmas = { data: { esquema: 'vec.contratacion-temporal.estado-firmas-documento.v1', catalogo_ref: 'catalogo:sintetico', huella_sha256: 'a'.repeat(64), ejemplo: true, firma_eficaz: false, verificacion_disponible: false, documentos: [] } };
  assert.equal(validarLectura(FIRMAS, firmas).firma_eficaz, false);
  for (const ruta of [BORRADORES, FIRMAS]) {
    const datos = { bloqueadas: 0, red_fallida: 0 };
    await interceptar({ request: () => request(ruta, 'POST', ruta === BORRADORES ? { expediente_ref: 'expediente:sintetico', version_observada: 1 } : { expediente_ref: 'expediente:sintetico' }),
      fetch: async () => ({ url: () => origen + ruta, status: () => 200, headersArray: async () => [], json: async () => ({ data: {} }) }), fulfill: async () => {}, abort: async () => {} }, origen, datos);
    assert.equal(datos.contratos_fallidos, 1);
  }
});

test('fallos de launch y certificado/newContext dejan resultado terminal saneado', async () => {
  const base = process.env.VEC_F_TEST_SCRATCH || os.tmpdir();
  const dir = fs.mkdtempSync(path.join(base, 'vec-f-prueba-'));
  const c = { commit_servido: 'a'.repeat(40), origenes: { interno: origen, externo: 'https://127.0.0.1:8444' }, bolsa_ref: 'bolsa:sintetica', expediente_ref: 'expediente:sintetico', idioma: 'es', identidades: { candidato: { certificado: '/privado', clave: '/secreto' } } };
  try {
    for (const fase of ['launch', 'context']) {
      const salida = path.join(dir, fase); fs.mkdirSync(salida, { mode: 0o700 });
      const error = new Error('/secreto certificado URL token=privado');
      const chromium = { launch: async () => { if (fase === 'launch') throw error;
        return { newContext: async () => { throw error; }, close: async () => {} }; } };
      await assert.rejects(recorrer(c, chromium, salida));
      const raw = fs.readFileSync(path.join(salida, 'resultado.json'), 'utf8');
      const resultado = JSON.parse(raw);
      assert.equal(resultado.estado, 'CORTADO'); assert.equal(resultado.corte, 'navegador');
      assert.doesNotMatch(raw, /secreto|token|certificado|URL/);
      if (fase === 'context') assert.equal(resultado.pasos[0].estado, 'CORTADO');
    }
  } finally { fs.rmSync(dir, { recursive: true, force: true }); }
});
