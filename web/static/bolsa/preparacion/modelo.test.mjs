import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { validarIdentificador, rutaDetalle, validarLimites, seleccionarFicheros, crearLectorDetalle, crearResumen } from './modelo.js';
import { textoResumen } from './arranque.js';
import { cargarTextos } from '../../comun/textos.js';

const huella = 'a'.repeat(64);
const fecha = '2026-10-01T10:00:00Z';
const limites = validarLimites({ maximo_archivos: '8', maximo_bytes_archivo: '10485760' });
function fixture() {
  const catalogo = { catalogo_id: 'categorias', version: 1, huella_sha256: huella, huella_proyeccion_sha256: huella };
  const valor = { clave: 'bolsa', version: 1, etiqueta: 'Bolsa de prueba', semantica: 'informacion' };
  return {
    esquema: 'vec.bolsa.publico.convocatoria.v2',
    fuente: { revision: 'fixture-test', actualizada_en: fecha, demostracion: false },
    convocatoria: { identificador_publico: 'auxiliares-2026', version: 'v2', huella_sha256: huella,
      titulo: '<img src=x onerror=alert(1)>', resumen: 'Resumen sintético', publicada_en: fecha,
      tipo: valor, estado: valor, categorias: [{ clave: 'auxiliares', version: 1 }], catalogo_categorias: catalogo,
      numero_requisitos: 1, numero_documentos: 1, numero_ayudas: 0 },
    diccionario_categorias: [{ clave: 'auxiliares', version: 1, etiqueta: 'Auxiliares', semantica: 'informacion', catalogo_categorias: catalogo }],
    descripcion: 'Información sintética.', requisitos: [{ titulo: 'Titulación', descripcion: 'Literal de las bases.', obligatorio: true }],
    plazos: [{ titulo: 'Solicitudes', tipo: valor, abre_en: fecha, cierra_en: '2026-11-01T10:00:00Z', etiqueta_situacion: 'Publicado', semantica_situacion: 'informacion', descripcion: 'Texto literal.' }],
    documentos: [{ titulo: 'Bases', descripcion: 'Documento público', formato: 'pdf', tipo: valor, url: '/bolsa/documentos/bases.pdf' }], ayuda: [],
  };
}
const respuesta = (datos) => new Response(JSON.stringify(datos), { headers: { 'content-type': 'application/json' } });

test('identificador cerrado y ruta pública canónica con idioma', () => {
  for (const id of ['../rrhh', 'AUXILIARES', 'https://otro.invalid', 'abc?x=1', '', null]) assert.equal(validarIdentificador(id), false);
  assert.equal(rutaDetalle('auxiliares-2026', 'en'), '/api/publico/bolsa/convocatorias/auxiliares-2026?idioma=en');
  assert.throws(() => rutaDetalle('../a', 'es'));
});

test('GET anónimo sin caché/redirecciones valida identidad del detalle antes de usarlo', async () => {
  let llamada;
  const lector = crearLectorDetalle({ fetchImpl: async (...args) => { llamada = args; return respuesta(fixture()); } });
  assert.equal((await lector.leer('auxiliares-2026', 'es')).convocatoria.version, 'v2');
  assert.equal(llamada[1].method, 'GET');
  assert.equal(llamada[1].credentials, 'omit');
  assert.equal(llamada[1].cache, 'no-store');
  assert.equal(llamada[1].redirect, 'error');
  assert.equal(llamada[1].referrerPolicy, 'no-referrer');
  const ajeno = fixture(); ajeno.convocatoria.identificador_publico = 'otro-2026';
  await assert.rejects(crearLectorDetalle({ fetchImpl: async () => respuesta(ajeno) }).leer('auxiliares-2026', 'es'));
});

test('no incorpora demo ni datos con enlace fuera de documentos públicos', async () => {
  for (const modificar of [d => { d.fuente.demostracion = true; }, d => { d.documentos[0].url = 'https://otro.invalid/bases.pdf'; }]) {
    const datos = fixture(); modificar(datos);
    await assert.rejects(crearLectorDetalle({ fetchImpl: async () => respuesta(datos) }).leer('auxiliares-2026', 'es'));
  }
});

test('descarta respuesta tardía al seleccionar otra convocatoria o desmontar', async () => {
  const pendientes = [];
  const lector = crearLectorDetalle({ fetchImpl: (_ruta, opciones) => new Promise(resolve => pendientes.push({ resolve, opciones })) });
  const antigua = lector.leer('auxiliares-2026', 'es');
  const actual = lector.leer('auxiliares-2026', 'en');
  assert.equal(pendientes[0].opciones.signal.aborted, true);
  pendientes[0].resolve(respuesta(fixture()));
  assert.equal(await antigua, null);
  lector.desmontar();
  assert.equal(pendientes[1].opciones.signal.aborted, true);
  pendientes[1].resolve(respuesta(fixture()));
  assert.equal(await actual, null);
  await assert.rejects(lector.leer('auxiliares-2026', 'es'));
});

test('errores y respuesta excesiva cierran el lector sin detalles inventados', async () => {
  await assert.rejects(crearLectorDetalle({ fetchImpl: async () => new Response('', { status: 503 }) }).leer('auxiliares-2026', 'es'), /no_fuente/);
  await assert.rejects(crearLectorDetalle({ fetchImpl: async () => new Response('x'.repeat(2 * 1024 * 1024 + 1), { headers: { 'content-type': 'application/json' } }) }).leer('auxiliares-2026', 'es'));
});

test('ficheros opcionales conservan solo nombre/tamaño sin leer bytes ni ruta', () => {
  const original = { name: 'titulo.pdf', size: 3, webkitRelativePath: '/privado/titulo.pdf', text() { throw new Error('no leer'); } };
  assert.deepEqual(seleccionarFicheros([original], limites), [{ nombre: 'titulo.pdf', tamano: 3 }]);
  assert.deepEqual(seleccionarFicheros([], limites), []);
  assert.throws(() => seleccionarFicheros(Array(9).fill(original), limites), /cantidad_error/);
  assert.throws(() => seleccionarFicheros([{ ...original, size: limites.maximoBytesArchivo + 1 }], limites), /tamano_error/);
  for (const name of ['a/b.pdf', 'a\\b.pdf', 'a\n.pdf', '..']) assert.throws(() => seleccionarFicheros([{ ...original, name }], limites), /nombre_error/);
  assert.throws(() => validarLimites({ maximo_archivos: '0', maximo_bytes_archivo: '-1' }));
});

test('resumen independiente conserva versión/huella y pendientes; documento público no crea adjunto', () => {
  const datos = fixture();
  const resumen = crearResumen(datos, { limites, fecha: new Date(fecha), lectura: true });
  assert.equal(resumen.estado, 'sin_presentar');
  assert.equal(resumen.convocatoria.huella_sha256, huella);
  assert.equal(resumen.convocatoria.version, 'v2');
  assert.equal(resumen.requisitos[0].cumplimiento, 'pendiente');
  assert.equal(resumen.requisitos[0].descripcion, 'Literal de las bases.');
  assert.equal(resumen.plazos[0].descripcion, 'Texto literal.');
  assert.deepEqual(resumen.archivos_locales, []);
  datos.requisitos[0].descripcion = 'mutado';
  assert.equal(resumen.requisitos[0].descripcion, 'Literal de las bases.');
});

test('descarga localizada con traductor real, fechas y metadatos; catálogos simétricos', async () => {
  const resumen = crearResumen(fixture(), { limites, archivos: [{ nombre: 'titulo.pdf', tamano: 23 }], fecha: new Date(fecha), lectura: true });
  for (const idioma of ['es', 'en']) {
    const textos = await cargarTextos('convoca-preparacion', { idioma });
    assert.deepEqual(textos.faltantes, []);
    const contenido = textoResumen(resumen, textos);
    assert.match(contenido, /v2/);
    assert.ok(contenido.includes(huella));
    assert.ok(contenido.includes('Literal de las bases.'));
    assert.ok(contenido.includes('titulo.pdf'));
    assert.ok(contenido.includes(textos.traducir('resumen.limite')));
    assert.ok(!contenido.includes('undefined'));
  }
  const es = JSON.parse(await readFile(new URL('../../textos/es/convoca-preparacion.json', import.meta.url)));
  const en = JSON.parse(await readFile(new URL('../../textos/en/convoca-preparacion.json', import.meta.url)));
  const claves = objeto => Object.entries(objeto).flatMap(([k,v]) => typeof v === 'object' ? claves(v).map(p => `${k}.${p}`) : [k]).sort();
  assert.deepEqual(claves(es), claves(en));
});
