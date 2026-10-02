import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile, writeFile } from 'node:fs/promises';
import { crearResumen } from '../modelo.js';
import { cargarTextos } from '../../../comun/textos.js';
import { recuperarResumen, leerArchivoLocal, MAXIMO_ARCHIVO } from './material.js';
import { rutaElegirConvocatoria } from './vista.js';

const detalle = JSON.parse(await readFile(new URL('./testdata/detalle-publico-sintetico.json', import.meta.url)));
const textos = await cargarTextos('convoca-preparacion');
const limites = textos.seccion('limites');
// Fixture sintética del contrato público; ningún dato acredita una fuente real.
const resumen = crearResumen(detalle, { limites: { maximoArchivos: Number(limites.maximo_archivos), maximoBytesArchivo: Number(limites.maximo_bytes_archivo) },
  archivos: [{ nombre: 'ejemplo.pdf', tamano: 23 }], lectura: true, fecha: new Date('2026-10-01T10:00:00Z') });
const bytes = v => new TextEncoder().encode(JSON.stringify(v, null, 2) + '\n');

test('recupera el DTO original de crearResumen y descarga los mismos bytes, sin alias', () => {
  const original = bytes(resumen); const esperado = original.slice();
  const recibido = recuperarResumen(original, limites);
  assert.deepEqual(recibido.resumen, resumen);
  original.fill(0); const copia = recibido.copiarOriginal(); copia.fill(1);
  assert.deepEqual(recibido.copiarOriginal(), esperado);
  const otra = recuperarResumen(recibido.copiarOriginal(), limites);
  assert.deepEqual(otra.resumen, recibido.resumen);
  assert.equal(otra.resumen.estado, 'sin_presentar');
  assert.equal(otra.resumen.requisitos[0].cumplimiento, 'pendiente');
  assert.equal(otra.resumen.convocatoria.version, 'v2');
});

test('arrays vacíos y descripción de plazo omitida son compatibles con el productor', () => {
  const entrada = structuredClone(resumen);
  entrada.requisitos = []; entrada.documentos_publicos = []; entrada.archivos_locales = [];
  delete entrada.plazos[0].descripcion;
  assert.deepEqual(recuperarResumen(bytes(entrada), limites).resumen, entrada);
});

test('conserva textos largos y listas válidas que el productor original permite', () => {
  const fuente = structuredClone(detalle);
  fuente.convocatoria.titulo = 'x'.repeat(400);
  fuente.requisitos = Array.from({ length: 300 }, (_, i) => ({ ...fuente.requisitos[0], titulo: `r${i}` }));
  fuente.convocatoria.numero_requisitos = fuente.requisitos.length;
  const producido = crearResumen(fuente, { limites: { maximoArchivos: Number(limites.maximo_archivos), maximoBytesArchivo: Number(limites.maximo_bytes_archivo) } });
  assert.deepEqual(recuperarResumen(bytes(producido), limites).resumen, producido);
});

test('rechaza estados elevados, campos de autoridad y requisitos resueltos', () => {
  for (const cambiar of [
    r => { r.estado = 'presentada'; }, r => { r.requisitos[0].cumplimiento = 'cumple'; },
    r => { r.persona_ref = 'persona:ejemplo'; }, r => { r.empleado_ref = 'empleado:ejemplo'; },
    r => { r.convocatoria.version = 2; }, r => { r.convocatoria.huella_sha256 = [r.convocatoria.huella_sha256]; }, r => { r.documentos_publicos[0].url = 'https://otro.invalid/bases.pdf'; },
    r => { r.archivos_locales[0].nombre = '../archivo.pdf'; }, r => { r.generada_en = '2026-02-31T10:00:00Z'; },
  ]) {
    const entrada = structuredClone(resumen); cambiar(entrada);
    assert.throws(() => recuperarResumen(bytes(entrada), limites));
  }
});

test('rechaza duplicados incluso escapados, UTF-8 inválido y archivo excesivo', () => {
  const texto = JSON.stringify(resumen);
  for (const clave of ['estado', '\\u0065stado']) {
    const duplicado = texto.replace('"estado":"sin_presentar"', '"estado":"sin_presentar","' + clave + '":"sin_presentar"');
    assert.throws(() => recuperarResumen(new TextEncoder().encode(duplicado), limites));
  }
  assert.throws(() => recuperarResumen(new Uint8Array([0xff]), limites));
  assert.throws(() => recuperarResumen(new Uint8Array(MAXIMO_ARCHIVO + 1), limites));
});

test('lector acota antes de leer y nunca abre los documentos o archivos mencionados', async () => {
  let lecturas = 0;
  const datos = bytes(resumen);
  const archivo = { size: datos.length, arrayBuffer: async () => { lecturas += 1; return datos.buffer; } };
  const recibido = await leerArchivoLocal(archivo, limites);
  assert.equal(lecturas, 1); assert.deepEqual(recibido.copiarOriginal(), datos);
  await assert.rejects(leerArchivoLocal({ ...archivo, size: MAXIMO_ARCHIVO + 1 }, limites));
  assert.equal(lecturas, 1);
  await assert.rejects(leerArchivoLocal({ ...archivo, size: datos.length - 1 }, limites));
});

test('catálogos ES/EN propios comparten claves y cargan con el traductor común', async () => {
  const es = JSON.parse(await readFile(new URL('../../../textos/es/seleccion-recuperacion.json', import.meta.url)));
  const en = JSON.parse(await readFile(new URL('../../../textos/en/seleccion-recuperacion.json', import.meta.url)));
  assert.deepEqual(Object.keys(es).sort(), Object.keys(en).sort());
  for (const idioma of ['es', 'en']) {
    const t = await cargarTextos('seleccion-recuperacion', { idioma });
    assert.deepEqual(t.faltantes, []); assert.ok(t.traducir('titulo'));
  }
});

test('ayuda dirige al listado y conserva idioma sin selector de archivo ni ruta de preparación inválida', async () => {
  const html = await readFile(new URL('./index.html', import.meta.url), 'utf8');
  assert.match(html, /id="elegir-convocatoria" href="\/bolsa\/"/u);
  assert.ok(!html.includes('href="/bolsa/preparacion/"'));
  for (const idioma of ['es', 'en']) {
    const ruta = new URL(rutaElegirConvocatoria(idioma), 'https://fixture.invalid');
    assert.equal(ruta.pathname, '/bolsa/');
    assert.equal(ruta.searchParams.get('lang'), idioma);
    assert.deepEqual([...ruta.searchParams.keys()], ['lang']);
  }
});

// Generación opcional de fixture para Chrome, por variable del ensayo; no es runtime.
if (process.env.VEC_S3_FIXTURE_SALIDA) await writeFile(process.env.VEC_S3_FIXTURE_SALIDA, bytes(resumen));
