import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { crearTextos } from '../../../comun/textos.js';
import { crearClienteCarrera, crearBorrador, validarEscenario } from './cliente.js';
const fixture = () => ({ alcance: 'preparacion_sintetica', version: 'ejercicio-1', fuente: { referencia: 'fuente-ejemplo', version: 'fuente-2' }, pendientes: ['carrera.pendiente.politica'], casos: [{ referencia: 'caso-1', via: 'grado', persona_nombre: 'Persona sintética', regimen: 'funcionario_carrera', estado_global: 'pendiente', nivel_puesto: 22, grado_personal: 20, fuentes: [{ referencia: 'personal-ejemplo', version: 'version-3' }], comprobaciones: [{ clave: 'carrera.comprobacion.politica', estado: 'pendiente', motivo_clave: 'carrera.motivo.politica_pendiente' }], pendientes: ['carrera.pendiente.politica'] }] });
test('el transporte solo lee el escenario fijo sin credenciales ni redirección', async () => {
  const ac = new AbortController(); let peticion;
  const cliente = crearClienteCarrera({ fetchImpl: async (url, opts) => { peticion = { url, opts }; return { ok: true, text: async () => JSON.stringify(fixture()) }; } });
  assert.deepEqual(await cliente.listar({ signal: ac.signal }), fixture());
  assert.equal(peticion.url.pathname.endsWith('/carrera/escenario.json'), true);
  assert.equal(peticion.url.search, '?v=20261001-carrera-preparacion-v1');
  assert.equal(peticion.opts.credentials, 'omit'); assert.equal(peticion.opts.redirect, 'error');
  assert.equal(peticion.opts.cache, 'no-store'); assert.equal(peticion.opts.referrerPolicy, 'no-referrer');
  assert.equal(peticion.opts.method, 'GET'); assert.equal(peticion.opts.signal, ac.signal);
});
test('rechaza error, tamaño excesivo y datos que presenten reconocimiento', async () => {
  for (const respuesta of [{ ok: false }, { ok: true, text: async () => ' '.repeat(1024 * 1024 + 1) }]) {
    await assert.rejects(crearClienteCarrera({ fetchImpl: async () => respuesta }).listar());
  }
  const dto = fixture(); dto.casos[0].estado_global = 'aprobado'; assert.throws(() => validarEscenario(dto));
  dto.casos[0].estado_global = 'pendiente'; dto.casos[0].nivel_puesto = {}; assert.throws(() => validarEscenario(dto));
  dto.casos[0].nivel_puesto = 22; dto.fuente.version = null; assert.throws(() => validarEscenario(dto));
});
test('el borrador conserva versiones, estados y separa grado de nivel sin modificar el escenario', () => {
  const dto = fixture(); const previo = structuredClone(dto); const salida = JSON.parse(crearBorrador(dto));
  assert.deepEqual(salida, previo); assert.deepEqual(dto, previo);
  assert.equal(salida.casos[0].nivel_puesto, 22); assert.equal(salida.casos[0].grado_personal, 20);
  assert.equal(salida.casos[0].estado_global, 'pendiente');
});
test('vacío y vías distintas conservan el contrato de preparación', () => {
  const dto = fixture(); dto.casos = []; assert.equal(validarEscenario(dto), dto);
  for (const via of ['grado', 'progresion', 'promocion']) { const caso = fixture(); caso.casos[0].via = via; assert.equal(validarEscenario(caso), caso); }
});
test('catálogos completos y motivos traducidos con el lector i18n común', async () => {
  const es = JSON.parse(await readFile(new URL('../../../textos/es/carrera.json', import.meta.url), 'utf8'));
  const en = JSON.parse(await readFile(new URL('../../../textos/en/carrera.json', import.meta.url), 'utf8'));
  const textos = crearTextos({ modulo: 'carrera', idioma: 'en', localizacion: 'en-GB', respaldo: es, propio: en });
  assert.deepEqual(textos.faltantes, []);
  function revisar(obj, ruta = '') { for (const [clave, valor] of Object.entries(obj)) { const k = ruta ? `${ruta}.${clave}` : clave; if (typeof valor === 'string') assert.ok(textos.traducir(k, { cuenta: 3, version: 'v1' }).length); else revisar(valor, k); } }
  revisar(es);
  assert.notEqual(textos.traducir('detalle.nivel_puesto'), textos.traducir('detalle.grado_personal'));
});
test('el respaldo de catálogo conserva su idioma desde datos y limita el transporte', async () => {
  const { leerErrorCatalogo } = await import('./cliente.js');
  const datos = JSON.parse(await readFile(new URL('../../../textos/es/carrera-error.json', import.meta.url), 'utf8'));
  let opciones;
  const resultado = await leerErrorCatalogo({ fetchImpl: async (url, opts) => { assert.equal(url.pathname.endsWith('/carrera/error-catalogo.json'), true); opciones = opts; return { ok: true, text: async () => JSON.stringify(datos) }; } });
  assert.deepEqual(resultado, datos); assert.equal(opciones.credentials, 'omit'); assert.equal(opciones.redirect, 'error'); assert.equal(opciones.cache, 'no-store'); assert.equal(opciones.referrerPolicy, 'no-referrer');
  await assert.rejects(leerErrorCatalogo({ fetchImpl: async () => ({ ok: false }) }));
  await assert.rejects(leerErrorCatalogo({ fetchImpl: async () => ({ ok: true, text: async () => JSON.stringify({ ...datos, idioma: null }) }) }));
});
test('un caso incompleto llega como pendiente y el borrador conserva las carencias de origen', async () => {
  const dto = fixture(); const c = dto.casos[0]; c.persona_nombre = ''; c.regimen = ''; c.fuentes[0].version = ''; dto.fuente.version = '';
  c.comprobaciones = [{ clave: 'carrera.comprobacion.datos', estado: 'pendiente', motivo_clave: 'carrera.motivo.datos_incompletos' }, { clave: 'carrera.comprobacion.fuentes', estado: 'pendiente', motivo_clave: 'carrera.motivo.fuente_incompleta' }];
  c.pendientes = ['carrera.pendiente.datos', 'carrera.pendiente.fuentes'];
  const cliente = crearClienteCarrera({ fetchImpl: async () => ({ ok: true, text: async () => JSON.stringify(dto) }) });
  const recibido = await cliente.listar(); assert.equal(recibido.casos[0].estado_global, 'pendiente');
  assert.deepEqual(JSON.parse(crearBorrador(recibido)), dto);
  const es = JSON.parse(await readFile(new URL('../../../textos/es/carrera.json', import.meta.url), 'utf8'));
  const t = crearTextos({ modulo: 'carrera', idioma: 'es', localizacion: 'es-ES', respaldo: es });
  assert.ok(t.traducir('detalle.persona_pendiente')); assert.ok(t.traducir('detalle.regimen_pendiente')); assert.ok(t.traducir('detalle.sin_dato'));
});
