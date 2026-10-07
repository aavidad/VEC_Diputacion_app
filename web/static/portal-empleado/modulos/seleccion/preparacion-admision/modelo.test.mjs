import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { leerArchivo, leerSalida, MAXIMO_BYTES, CAUSAS, PENDIENTES } from './modelo.js';
import { crearCargaLocal } from './controlador.js';
import { cargarTextos } from '../../../../comun/textos.js';

const bytes = new Uint8Array(await readFile(new URL('./testdata/admision-preparada.json', import.meta.url)));
const base = () => JSON.parse(new TextDecoder().decode(bytes));
const codificar = dto => new TextEncoder().encode(JSON.stringify(dto));
const leer = dto => leerSalida(codificar(dto));

test('salida CLI conserva pendientes, vínculo local sin presentar y soportes declarados', async () => {
  const dto = leerSalida(bytes);
  assert.equal(dto.admision_oficial, false); assert.equal(dto.persistido, false);
  assert.equal(dto.universo_requisitos, 'propuesto_no_cotejado');
  assert.equal(dto.solicitud_contexto.estado, 'sin_presentar');
  assert.ok(dto.requisitos.every(r => r.estado === 'pendiente'));
  const resultado = await leerArchivo({ size: bytes.length, arrayBuffer: async () => bytes.buffer });
  assert.deepEqual(resultado.bytes, bytes);
});
test('sin solicitud, sin paquete y lista vacía siguen siendo una preparación pendiente', () => {
  const dto = base(); delete dto.solicitud_contexto; delete dto.version_paquete_hechos; dto.requisitos = [];
  assert.equal(leer(dto).requisitos.length, 0);
});
test('datos nominales, claves extra, tipos y falso acto se rechazan sin resultado parcial', () => {
  for (const mutar of [
    d => { d.persona = 'dato'; }, d => { d.bases.dni = 'dato'; }, d => { d.estado = 'admitida'; },
    d => { d.persistido = true; }, d => { d.admision_oficial = true; }, d => { d.universo_requisitos = 'completo'; },
    d => { d.requisitos[0].estado = 'cumple'; }, d => { d.requisitos[0].causas = []; },
    d => { d.requisitos[0].causas.push('seleccion.admision.causa.nueva'); },
    d => { d.requisitos[0].accion_propuesta = 'seleccion.admision.accion.admitir'; },
    d => { d.revision = '1'; }, d => { d.bases.referencia = '../ruta'; },
    d => { d.bases.huella_sha256 = 'A'.repeat(64); },
    d => { d.solicitud_contexto.estado = 'presentada'; }, d => { d.solicitud_contexto.version = 'otra'; },
    d => { d.requisitos[0].requisito.hito_fecha = '2026-02-30'; },
    d => { d.requisitos[0].requisito.regla_version = ''; },
    d => { d.requisitos[0].soportes[0].version = 2; },
    d => { d.requisitos[0].soportes[0].evidencias_aportadas = 33; },
    d => { d.requisitos[0].soportes[0].hasta = '2026-01-01'; },
    d => { d.requisitos[0].soportes[0].estado_aportado = 'caducado'; },
  ]) { const dto = base(); mutar(dto); assert.throws(() => leer(dto), /formato/); }
});
test('duplicados de requisitos, causas, pendientes, soportes y referencias son incompatibles', () => {
  for (const mutar of [
    d => { d.requisitos.push(d.requisitos[0]); }, d => { d.pendientes.push(d.pendientes[0]); },
    d => { d.requisitos[0].causas.push(d.requisitos[0].causas[0]); },
    d => { d.requisitos[0].soportes.push(d.requisitos[0].soportes[0]); },
    d => { d.requisitos[0].requisito.hechos_esperados.push(d.requisitos[0].requisito.hechos_esperados[0]); },
    d => { d.requisitos[1].requisito.hechos_esperados = [{ referencia: d.requisitos[0].requisito.hechos_esperados[0].referencia, version: 2 }]; },
  ]) { const dto = base(); mutar(dto); assert.throws(() => leer(dto), /formato/); }
});
test('rechaza claves léxicas duplicadas, prototipos, UTF-8 inválido y profundidad excesiva', () => {
  for (const texto of ['{"esquema":1,"esquema":2}', '{"__proto__":{}}', '{"constructor":{}}',
    '{"prototype":{}}', '{"a": {"b":1,"b":2}}', '['.repeat(34) + '0' + ']'.repeat(34)]) {
    assert.throws(() => leerSalida(new TextEncoder().encode(texto)), /formato/);
  }
  assert.throws(() => leerSalida(new Uint8Array([255])), /formato/);
});
test('títulos aportados tienen límite UTF-8, sin controles ni espacios exteriores', () => {
  for (const titulo of ['', ' nombre', 'nombre ', 'a\nb', 'á'.repeat(129)]) {
    const dto = base(); dto.requisitos[0].requisito.titulo_propuesto = titulo; assert.throws(() => leer(dto), /formato/);
  }
  const dto = base(); dto.requisitos[0].requisito.titulo_propuesto = 'á'.repeat(128); assert.equal(leer(dto).requisitos.length, 2);
});
test('límites de archivo se verifican antes y después de leerlo', async () => {
  for (const size of [0, MAXIMO_BYTES + 1]) {
    await assert.rejects(leerArchivo({ size, arrayBuffer() { throw new Error('no debe leerse'); } }), /tamano/);
  }
  await assert.rejects(leerArchivo({ size: 1, arrayBuffer: async () => new ArrayBuffer(MAXIMO_BYTES + 1) }), /tamano/);
  const dto = base(); dto.requisitos = Array(65).fill(dto.requisitos[0]); assert.throws(() => leer(dto), /formato/);
});
test('catálogos reales traducen causas, acciones y pendientes ES/EN sin claves ausentes', async () => {
  for (const idioma of ['es', 'en']) {
    const t = await cargarTextos('selectivos-admision-visor', { idioma }); assert.deepEqual(t.faltantes, []);
    for (const c of CAUSAS) assert.ok(t.traducir(`causas.${c}`).length);
    for (const p of PENDIENTES) assert.ok(t.traducir(`pendientes.${p}`).length);
    for (const a of ['preparar_revision', 'preparar_aportacion']) assert.ok(t.traducir(`acciones.${a}`).length);
  }
});
test('reimportar retira lo anterior y una lectura tardía no desplaza el archivo posterior', async () => {
  const resolutores = []; const visibles = []; const mensajes = []; let retiradas = 0;
  const c = crearCargaLocal({ leer: () => new Promise(resolve => resolutores.push(resolve)), retirar: () => { ++retiradas; },
    mostrar: r => visibles.push(r), anunciar: (clave, error) => mensajes.push([clave, error]) });
  const a = c.abrir({}); const b = c.abrir({}); resolutores[1]('segundo'); await b; resolutores[0]('primero'); await a;
  assert.deepEqual(visibles, ['segundo']); assert.equal(retiradas, 2); assert.equal(mensajes.at(-1)[0], 'cargado');
});
test('cerrar y desmontar impiden que una lectura tardía reponga el material', async () => {
  for (const accion of ['cerrar', 'desmontar']) {
    let resolver; const visibles = [];
    const c = crearCargaLocal({ leer: () => new Promise(r => { resolver = r; }), retirar() {}, mostrar: r => visibles.push(r), anunciar() {} });
    const carga = c.abrir({}); c[accion](); resolver('tarde'); await carga; assert.deepEqual(visibles, []);
  }
});
test('un error durante lectura o pintado retira datos antes de anunciarlo', async () => {
  for (const enVista of [false, true]) {
    const sucesos = [];
    const c = crearCargaLocal({ leer: async () => { if (!enVista) throw new TypeError('tamano'); return {}; },
      retirar: () => sucesos.push('retirar'), mostrar: () => { throw new TypeError('formato'); },
      anunciar: (clave, error) => sucesos.push([clave, error]) });
    await c.abrir({}); assert.deepEqual(sucesos.at(-2), 'retirar');
    assert.deepEqual(sucesos.at(-1), [enVista ? 'error_formato' : 'error_tamano', true]);
  }
});
