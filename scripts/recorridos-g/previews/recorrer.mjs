#!/usr/bin/env node
import fs from 'node:fs';
import path from 'node:path';
import os from 'node:os';
import assert from 'node:assert/strict';
import { pathToFileURL } from 'node:url';
import { chromeAccesible, verificarZoom } from '../../recorridos-f/a11y.mjs';
import { comprobarFuente, crearServidor, peticionPermitida } from './servidor.mjs';

const casos = JSON.parse(fs.readFileSync(new URL('casos.json', import.meta.url), 'utf8'));
const idiomas = JSON.parse(fs.readFileSync(new URL('idiomas.json', import.meta.url), 'utf8'));
const textoDefecto = idiomas.disponibles[idiomas.por_defecto];

function opciones(argv) {
  const o = {};
  for (let i = 0; i < argv.length; i += 2) {
    const clave = argv[i];
    if (!['--source', '--modo'].includes(clave) || !argv[i + 1] || o[clave]) throw new Error('uso');
    o[clave] = argv[i + 1];
  }
  if (argv.length % 2 || !o['--source'] || !['plan', 'chrome'].includes(o['--modo'])) throw new Error('uso');
  return { fuente: o['--source'], modo: o['--modo'] };
}

export function crearVarianteInforme(informe, configuracion, caso) {
  const variante = caso.variante;
  assert.equal(informe.naturaleza, 'sintetica');
  assert.equal(informe.schema, 'dietas-informes-demo');
  assert.equal(configuracion.naturaleza, 'sintetica');
  assert.equal(configuracion.schema, 'dietas-informes-config-demo');
  assert.equal(informe.configuracion_ref, configuracion.referencia);
  assert.equal(informe.configuracion_version, configuracion.version);
  assert.equal(configuracion.historia.length, configuracion.version);
  assert.equal(variante.version_configuracion, configuracion.version + 1);
  assert.equal(variante.historia_adicional.version, variante.version_configuracion);
  assert.equal(caso.referencias.length, 2);
  const datos = structuredClone(informe);
  const criterio = structuredClone(configuracion);
  datos.configuracion_version = variante.version_configuracion;
  criterio.version = variante.version_configuracion;
  criterio.criterio = structuredClone(variante.criterio);
  criterio.historia.push(structuredClone(variante.historia_adicional));
  const campo = criterio.criterio.campo_fecha;
  const estados = new Set(criterio.criterio.estados_incluidos);
  const { persona, unidad, desde, hasta } = caso.filtros;
  const filas = datos.registros.filter(r => estados.has(r.situacion) && r.persona_ref === persona
    && r.unidad_ref === unidad && r[campo] >= desde && r[campo] <= hasta);
  assert.deepEqual(filas.map(r => r.referencia), caso.referencias, 'referencias_informe');
  const total = filas.reduce((suma, fila) => suma + criterio.criterio.conceptos_incluidos
    .reduce((parte, concepto) => parte + fila.conceptos_centimos[concepto], 0), 0);
  assert.equal(total, caso.total_centimos, 'importe_informe');
  return { datos, criterio };
}

function revisarDatos(fuente) {
  const informe = JSON.parse(fs.readFileSync(path.join(fuente, casos.datos[0].slice(1)), 'utf8'));
  const catalogo = JSON.parse(fs.readFileSync(path.join(fuente, casos.datos[1].slice(1)), 'utf8'));
  const criterio = JSON.parse(fs.readFileSync(path.join(fuente, casos.datos[2].slice(1)), 'utf8'));
  assert.equal(catalogo.estado, 'ejemplo');
  assert.ok(catalogo.version.startsWith('propuesta:'));
  return crearVarianteInforme(informe, criterio, casos.informes);
}

export function registrarRespuesta(red, inyeccion, estado, ruta, solicitud, fase) {
  if (estado < 400) return;
  if (estado === 503 && ruta === inyeccion.ruta && solicitud === inyeccion.solicitud
    && fase === inyeccion.fase && inyeccion.emitidas === 1 && inyeccion.observadas === 0) {
    inyeccion.observadas++;
    return;
  }
  red.respuestasError++;
}

export function evaluarScroll(m, politica) {
  const pc = m.ancho_css >= politica.pc_min_css;
  const docAlto = m.documento.alto > m.documento.visible;
  const mainAlto = m.principal.alto > m.principal.visible;
  if (pc) {
    assert.equal(politica.pc.documento, 'sin_scroll');
    assert.equal(politica.pc.principal, 'scroll_si_desborda');
    assert.equal(docAlto, false, 'scroll_documento_pc');
    assert.equal(m.documento.y_tras_end, 0, 'ventana_pc_tras_end');
    if (mainAlto) assert.equal(m.principal.alcanzable, true, 'scroll_principal_pc');
  } else {
    assert.equal(politica.estrecha.documento, 'permitido');
    assert.equal(politica.estrecha.principal, 'permitido');
    assert.equal(politica.estrecha.contenido_alto, 'scroll_alcanzable');
    if (docAlto || mainAlto) assert.ok(m.documento.alcanzable || m.principal.alcanzable, 'scroll_estrecho_inaccesible');
  }
  return { tipo: pc ? 'pc' : 'estrecha', scroll_documento: m.documento.alcanzable,
    scroll_principal: m.principal.alcanzable, ventana_tras_end: m.documento.y_tras_end };
}

async function comprobarPrivacidad(page, context) {
  assert.deepEqual(await context.cookies(), []);
  const estado = await page.evaluate(async () => ({
    local: localStorage.length, sesion: sessionStorage.length,
    indexed: (await indexedDB.databases()).length,
    caches: (await caches.keys()).length,
  }));
  assert.deepEqual(estado, { local: 0, sesion: 0, indexed: 0, caches: 0 });
}

async function comprobarPantalla(page, ancho, factor) {
  await page.keyboard.press('End');
  const medida = await page.evaluate(() => {
    const doc = document.scrollingElement;
    const main = document.querySelector('#espacio-trabajo');
    const y_tras_end = scrollY;
    const medir = el => {
      const alto = el.scrollHeight, visible = el.clientHeight;
      el.scrollTo({ top: 0, behavior: 'instant' });
      el.scrollTo({ top: alto, behavior: 'instant' });
      return { alto, visible, alcanzable: el.scrollTop > 0 };
    };
    const documento = medir(doc);
    documento.y_tras_end = y_tras_end;
    return {
      ancho_css: innerWidth, dpr: devicePixelRatio, escala_visual: visualViewport.scale,
      zoom_css: getComputedStyle(document.documentElement).zoom,
      desbordamiento: document.documentElement.scrollWidth > document.documentElement.clientWidth,
      documento, principal: medir(main),
    };
  });
  verificarZoom(medida, ancho, factor);
  assert.equal(medida.desbordamiento, false);
  return { ancho, zoom: factor * 100, ...evaluarScroll(medida, casos.pantalla) };
}

async function teclado(page) {
  await page.keyboard.press('Tab');
  const foco = await page.evaluate(() => {
    const el = document.activeElement;
    return el && el !== document.body && el.getClientRects().length > 0 && el.matches(':focus-visible');
  });
  assert.equal(foco, true, 'foco_teclado');
}

async function recorrerPagina(chrome, origen, nombre, idioma, ancho, factor, variante, resultado) {
  await chrome.zoom(factor);
  const context = await chrome.browser.newContext({ viewport: { width: ancho, height: 900 }, serviceWorkers: 'block', acceptDownloads: false });
  const red = { bloqueadas: 0, descargas: 0, cookies: 0, respuestasError: 0, muestra: [] };
  const errores = [];
  const reintento = nombre === casos.reintento.pagina && ancho === casos.reintento.ancho
    && factor === casos.reintento.factor && idioma === idiomas[casos.reintento.idioma];
  const inyeccion = { ruta: casos.datos[1], programada: reintento, emitidas: 0, observadas: 0,
    solicitud: null, fase: 'carga_inicial' };
  let fase = 'carga_inicial';
  const sustituciones = new Map(nombre === 'informes' ? [
    [casos.datos[0], JSON.stringify(variante.datos)], [casos.datos[2], JSON.stringify(variante.criterio)],
  ] : []);
  const sustituidas = new Map([...sustituciones.keys()].map(ruta => [ruta, 0]));
  try {
    await context.route('**/*', async route => {
      const request = route.request();
      let u;
      try { u = new URL(request.url()); } catch { red.bloqueadas++; await route.abort(); return; }
      if (!peticionPermitida(request.url(), request.method(), origen)) {
        red.bloqueadas++; red.muestra.push({ interno: u.origin === origen, ruta: u.origin === origen ? u.pathname + u.search : 'externo' }); await route.abort(); return;
      }
      if (inyeccion.programada && u.pathname === inyeccion.ruta) {
        inyeccion.programada = false;
        inyeccion.emitidas++;
        inyeccion.solicitud = request;
        await route.fulfill({ status: 503, contentType: 'application/json', body: '{}' });
        return;
      }
      if (sustituciones.has(u.pathname)) {
        sustituidas.set(u.pathname, sustituidas.get(u.pathname) + 1);
        await route.fulfill({ status: 200, contentType: 'application/json', body: sustituciones.get(u.pathname) });
        return;
      }
      await route.continue();
    });
    await context.routeWebSocket('**/*', route => { red.bloqueadas++; route.close(); });
    const page = await context.newPage();
    page.on('pageerror', e => errores.push(e.message));
    page.on('download', () => red.descargas++);
    page.on('response', r => {
      registrarRespuesta(red, inyeccion, r.status(), new URL(r.url()).pathname, r.request(), fase);
      if (r.headers()['set-cookie']) red.cookies++;
    });
    const destino = `${origen}${casos.paginas[nombre]}?lang=${idioma}`;
    const respuesta = await page.goto(destino, { waitUntil: 'networkidle', timeout: 20000 });
    assert.equal(respuesta.status(), 200);
    assert.equal(await page.locator('html').getAttribute('lang'), idiomas.disponibles[idioma].codigo);
    const t = idiomas.disponibles[idioma];
    if (nombre === 'informes') {
      await page.locator('[data-dietas-informes]').waitFor();
      await page.getByText(t.texto_sintetico_informes, { exact: true }).waitFor();
      for (const etiqueta of [t.texto_exportar, t.texto_imprimir]) assert.equal(await page.getByRole('button', { name: etiqueta }).isDisabled(), true);
      await page.locator('[data-dietas-informes-filtros]').waitFor({ state: 'visible' });
      await page.locator('select[name=persona]').selectOption(casos.informes.filtros.persona);
      await page.locator('select[name=unidad]').selectOption(casos.informes.filtros.unidad);
      await page.locator('input[name=desde]').fill(casos.informes.filtros.desde);
      await page.locator('input[name=hasta]').fill(casos.informes.filtros.hasta);
      await page.locator('[data-dietas-informes-filtros] button[type=submit]').click();
      const filas = page.locator('[data-dietas-informes-listado] tbody tr');
      await filas.first().waitFor();
      assert.deepEqual(await filas.locator('td:first-child').allInnerTexts(), casos.informes.referencias, 'referencias_renderizadas');
      const importe = await page.locator('[data-dietas-informes-resumen] .tarjeta-kpi .valor-kpi').nth(1).innerText();
      const esperado = new Intl.NumberFormat(t.localizacion, { style: 'currency', currency: variante.datos.moneda })
        .format(casos.informes.total_centimos / 100);
      assert.equal(importe.replace(/\s+/gu, ' ').trim(), esperado.replace(/\s+/gu, ' ').trim(), 'importe_renderizado');
    } else {
      await page.locator('[data-dietas-catalogo]').waitFor();
      if (reintento) {
        assert.equal(inyeccion.observadas, 1, 'fallo_inyectado_observado');
        fase = 'reintento';
        await page.getByRole('button', { name: t.texto_reintentar }).click();
      }
      await page.locator('[data-dietas-catalogo] [role=status]').getByText(t.texto_sintetico_catalogo, { exact: true }).waitFor();
      await page.locator('[data-dietas-catalogo] tbody tr').nth(casos.catalogo.fila).locator('button').click();
      const editor = page.locator('[data-dietas-catalogo-editor]');
      await editor.locator('input[name=importe_propuesto]').fill(casos.catalogo.importes[idioma]);
      await editor.locator('textarea[name=motivo]').fill(casos.catalogo.motivos[idioma]);
      await editor.locator('button[type=submit]').click();
      await page.getByText(t.texto_limite_propuesta, { exact: true }).waitFor();
      assert.ok((await editor.innerText()).includes(casos.catalogo.importes[idioma]));
      assert.equal(await editor.getByRole('button', { name: t.texto_publicar }).isDisabled(), true);
    }
    await teclado(page);
    const pantalla = await comprobarPantalla(page, ancho, factor);
    await comprobarPrivacidad(page, context);
    assert.deepEqual(errores, [], `${nombre}/${idioma}/${ancho}/${factor}: javascript`);
    assert.equal(red.descargas, 0, 'descargas'); assert.equal(red.cookies, 0, 'cookies');
    assert.equal(red.respuestasError, 0, 'respuesta_http'); assert.equal(red.bloqueadas, 0, `peticiones_bloqueadas:${JSON.stringify(red.muestra)}`);
    assert.equal(inyeccion.emitidas, reintento ? 1 : 0, 'inyeccion_503');
    assert.equal(inyeccion.observadas, inyeccion.emitidas, 'respuesta_503');
    for (const cuenta of sustituidas.values()) assert.equal(cuenta, 1, 'variante_interceptada');
    resultado.push({ pagina: nombre, idioma, ...pantalla, peticiones_bloqueadas: red.bloqueadas });
  } finally { await context.close(); }
}

export async function cerrarRecorrido(chrome, server, scratch, falloPrincipal = null) {
  let falloCierre = null;
  try { await chrome?.cerrar(); } catch (error) { falloCierre ??= error; }
  try { if (server.servidor.listening) await server.cerrar(); } catch (error) { falloCierre ??= error; }
  try { fs.rmSync(scratch, { recursive: true, force: true }); } catch (error) { falloCierre ??= error; }
  if (!falloPrincipal && falloCierre) throw falloCierre;
}

export async function ejecutar(argv) {
  const { fuente, modo } = opciones(argv);
  const ausentes = comprobarFuente(fuente, casos);
  if (ausentes.length) {
    process.stderr.write(`${textoDefecto.dependencia}\n${JSON.stringify({ ausentes })}\n`);
    return 2;
  }
  const variante = revisarDatos(fuente);
  if (modo === 'plan') {
    process.stdout.write(`${textoDefecto.titulo_plan}\n`);
    return 0;
  }
  const modulo = process.env.VEC_PLAYWRIGHT_MODULE;
  if (!modulo || !path.isAbsolute(modulo) || !fs.statSync(modulo).isFile()) throw new Error('playwright_local_pendiente');
  const { chromium } = await import(pathToFileURL(modulo).href);
  const scratch = fs.mkdtempSync(path.join(os.tmpdir(), 'vec-dietas-previews-'));
  fs.chmodSync(scratch, 0o700);
  const server = crearServidor(fuente);
  let chrome;
  let falloPrincipal;
  try {
    const origen = await server.escuchar();
    chrome = await chromeAccesible(chromium, scratch);
    const resultado = [];
    for (const idioma of Object.keys(idiomas.disponibles)) for (const nombre of Object.keys(casos.paginas)) {
      for (const [ancho, factor] of [[1440, 1], [390, 1], [1440, 2]]) await recorrerPagina(chrome, origen, nombre, idioma, ancho, factor, variante, resultado);
    }
    assert.equal(server.contadores().denegadas, 0);
    process.stdout.write(`${textoDefecto.resultado}\n${JSON.stringify({ casos: resultado.length, resultado, servidor: server.contadores() })}\n`);
    return 0;
  } catch (error) {
    falloPrincipal = error;
    throw error;
  } finally {
    await cerrarRecorrido(chrome, server, scratch, falloPrincipal);
  }
}

if (process.argv[1] && path.resolve(process.argv[1]) === path.resolve(new URL(import.meta.url).pathname)) {
  ejecutar(process.argv.slice(2)).then(code => { process.exitCode = code; }, error => {
    process.stderr.write(`${error.message}\n`); process.exitCode = 1;
  });
}
