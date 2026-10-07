import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { comprobarFuente, crearServidor, ficheroPermitido, peticionPermitida } from './servidor.mjs';
import { cerrarRecorrido, crearVarianteInforme, evaluarScroll, ejecutar, registrarRespuesta } from './recorrer.mjs';

const casos = JSON.parse(fs.readFileSync(new URL('casos.json', import.meta.url), 'utf8'));
const idiomas = JSON.parse(fs.readFileSync(new URL('idiomas.json', import.meta.url), 'utf8'));

test('lista positiva: solo estáticos públicos y tres JSON exactos', () => {
  assert.equal(ficheroPermitido(casos.paginas.informes), casos.paginas.informes.slice(1));
  assert.equal(ficheroPermitido(casos.datos[0]), casos.datos[0].slice(1));
  for (const ruta of ['/api/vec/dietas', '/.git/config', '/data/demo/dietas/otro.json',
    '/web/static/../../AGENTS.md', '/web/static/.env', '/web/static/app.js?x=1', '/web/static/cert.pem']) {
    assert.equal(ficheroPermitido(ruta), null, ruta);
  }
  const origen = 'http://127.0.0.1:40123';
  assert.equal(peticionPermitida(`${origen}${casos.paginas.informes}?lang=${idiomas.por_defecto}`, 'GET', origen), true);
  assert.equal(peticionPermitida(`http://127.0.0.1:40124${casos.paginas.informes}`, 'GET', origen), false);
  assert.equal(peticionPermitida(`${origen}${casos.paginas.informes}?lang=desconocido`, 'GET', origen), false);
  assert.equal(peticionPermitida(`${origen}${casos.paginas.informes}?lang=${idiomas.por_defecto}&x=1`, 'GET', origen), false);
  assert.equal(peticionPermitida(`${origen}/api/vec/dietas`, 'GET', origen), false);
});

test('una preview ausente no da plan verde ni abre Chrome', async () => {
  const fuente = fs.mkdtempSync(path.join(os.tmpdir(), 'vec-preview-ausente-'));
  try {
    assert.equal(comprobarFuente(fuente, casos).length, 5);
    assert.equal(await ejecutar(['--source', fuente, '--modo', 'plan']), 2);
  } finally { fs.rmSync(fuente, { recursive: true, force: true }); }
});

test('servidor deniega API, escritura, datos extra y enlaces', async () => {
  const fuente = fs.mkdtempSync(path.join(os.tmpdir(), 'vec-preview-servidor-'));
  const pagina = path.join(fuente, casos.paginas.informes.slice(1));
  fs.mkdirSync(path.dirname(pagina), { recursive: true });
  fs.writeFileSync(pagina, '<!doctype html><title>Fixture</title>');
  const enlace = path.join(fuente, 'web/static/enlace.html');
  fs.symlinkSync(pagina, enlace);
  const servidor = crearServidor(fuente);
  try {
    const origen = await servidor.escuchar();
    assert.equal((await fetch(`${origen}${casos.paginas.informes}?lang=${idiomas.por_defecto}`)).status, 200);
    assert.equal((await fetch(`${origen}/api/vec/dietas`)).status, 403);
    assert.equal((await fetch(`${origen}${casos.paginas.informes}`, { method: 'POST' })).status, 403);
    assert.equal((await fetch(`${origen}/data/demo/dietas/otro.json`)).status, 403);
    assert.equal((await fetch(`${origen}/web/static/enlace.html`)).status, 404);
    assert.equal(servidor.contadores().peticiones, 1);
  } finally { if (servidor.servidor.listening) await servidor.cerrar(); fs.rmSync(fuente, { recursive: true, force: true }); }
});

test('variante de liquidación v2 enlazada: referencias e importe exactos', () => {
  const base = {
    naturaleza: 'sintetica', schema: 'dietas-informes-demo', configuracion_ref: 'propuesta:ejemplo',
    configuracion_version: 1, registros: [
      { referencia: casos.informes.referencias[0], situacion: 'liquidado', persona_ref: casos.informes.filtros.persona,
        unidad_ref: casos.informes.filtros.unidad, fecha_liquidacion: casos.informes.filtros.desde,
        conceptos_centimos: { manutencion: casos.informes.total_centimos } },
      { referencia: casos.informes.referencias[1], situacion: 'liquidado', persona_ref: casos.informes.filtros.persona,
        unidad_ref: casos.informes.filtros.unidad, fecha_liquidacion: casos.informes.filtros.hasta,
        conceptos_centimos: { manutencion: 0 } },
    ],
  };
  const config = { naturaleza: 'sintetica', schema: 'dietas-informes-config-demo', referencia: base.configuracion_ref,
    version: 1, historia: [{ version: 1 }] };
  const variante = crearVarianteInforme(base, config, casos.informes);
  assert.equal(variante.datos.configuracion_version, variante.criterio.version);
  assert.equal(variante.criterio.historia.length, variante.criterio.version);
  assert.equal(config.version, 1);
  const incorrecto = structuredClone(casos.informes);
  incorrecto.total_centimos++;
  assert.throws(() => crearVarianteInforme(base, config, incorrecto), /importe_informe/u);
  incorrecto.total_centimos--;
  incorrecto.referencias[1] = 'otra-referencia';
  assert.throws(() => crearVarianteInforme(base, config, incorrecto), /referencias_informe/u);
});

test('R10 separa scroll del documento y del panel; estrecho exige salida alcanzable', () => {
  const pc = { ancho_css: casos.pantalla.pc_min_css, documento: { alto: 900, visible: 900, alcanzable: false, y_tras_end: 0 },
    principal: { alto: 1400, visible: 900, alcanzable: true } };
  assert.equal(evaluarScroll(pc, casos.pantalla).scroll_documento, false);
  assert.equal(evaluarScroll(pc, casos.pantalla).scroll_principal, true);
  assert.throws(() => evaluarScroll({ ...pc, documento: { ...pc.documento, alto: 901 } }, casos.pantalla), /scroll_documento_pc/u);
  assert.throws(() => evaluarScroll({ ...pc, documento: { ...pc.documento, y_tras_end: 1 } }, casos.pantalla), /ventana_pc_tras_end/u);
  const estrecha = { ...pc, ancho_css: casos.pantalla.pc_min_css - 1, principal: { ...pc.principal, alcanzable: false } };
  assert.throws(() => evaluarScroll(estrecha, casos.pantalla), /scroll_estrecho_inaccesible/u);
});

test('un 503 ajeno o repetido nunca queda amparado por el reintento inyectado', () => {
  const red = { respuestasError: 0 };
  const solicitud = {};
  const inyeccion = { ruta: casos.datos[1], emitidas: 1, observadas: 0,
    solicitud, fase: 'carga_inicial' };
  registrarRespuesta(red, inyeccion, 503, casos.datos[0], solicitud, inyeccion.fase);
  assert.equal(red.respuestasError, 1);
  registrarRespuesta(red, inyeccion, 503, casos.datos[1], {}, inyeccion.fase);
  assert.equal(red.respuestasError, 2);
  registrarRespuesta(red, inyeccion, 503, casos.datos[1], solicitud, 'reintento');
  assert.equal(red.respuestasError, 3);
  registrarRespuesta(red, inyeccion, 503, casos.datos[1], solicitud, inyeccion.fase);
  assert.equal(inyeccion.observadas, 1);
  registrarRespuesta(red, inyeccion, 503, casos.datos[1], solicitud, inyeccion.fase);
  assert.equal(red.respuestasError, 4);
});

test('fallo al cerrar Chrome no impide cerrar servidor ni retirar perfil; error original prevalece', async () => {
  const scratch = fs.mkdtempSync(path.join(os.tmpdir(), 'vec-preview-cierre-'));
  let servidorCerrado = false;
  const falloCierre = new Error('chrome_close');
  const chrome = { cerrar: async () => { throw falloCierre; } };
  const server = { servidor: { listening: true }, cerrar: async () => { servidorCerrado = true; } };
  await assert.rejects(cerrarRecorrido(chrome, server, scratch), error => error === falloCierre);
  assert.equal(servidorCerrado, true);
  assert.equal(fs.existsSync(scratch), false);
  const otro = fs.mkdtempSync(path.join(os.tmpdir(), 'vec-preview-cierre-'));
  const falloOriginal = new Error('recorrido');
  await cerrarRecorrido(chrome, server, otro, falloOriginal);
  assert.equal(fs.existsSync(otro), false);
});

test('preflight no sigue un directorio de datos enlazado', () => {
  const fuente = fs.mkdtempSync(path.join(os.tmpdir(), 'vec-preview-enlace-'));
  const otro = fs.mkdtempSync(path.join(os.tmpdir(), 'vec-preview-otro-'));
  try {
    fs.symlinkSync(otro, path.join(fuente, 'data'));
    assert.ok(comprobarFuente(fuente, casos).includes(casos.datos[0]));
  } finally {
    fs.rmSync(fuente, { recursive: true, force: true });
    fs.rmSync(otro, { recursive: true, force: true });
  }
});
