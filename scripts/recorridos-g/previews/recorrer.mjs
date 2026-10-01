#!/usr/bin/env node
import fs from 'node:fs';
import path from 'node:path';
import os from 'node:os';
import assert from 'node:assert/strict';
import { pathToFileURL } from 'node:url';
import { chromeAccesible, verificarZoom } from '../../recorridos-f/a11y.mjs';
import { comprobarFuente, crearServidor, ficheroPermitido } from './servidor.mjs';

const casos = JSON.parse(fs.readFileSync(new URL('casos.json', import.meta.url), 'utf8'));
const idiomas = JSON.parse(fs.readFileSync(new URL('idiomas.json', import.meta.url), 'utf8'));

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

function revisarDatos(fuente) {
  const informe = JSON.parse(fs.readFileSync(path.join(fuente, casos.datos[0].slice(1)), 'utf8'));
  const catalogo = JSON.parse(fs.readFileSync(path.join(fuente, casos.datos[1].slice(1)), 'utf8'));
  const criterio = JSON.parse(fs.readFileSync(path.join(fuente, casos.datos[2].slice(1)), 'utf8'));
  assert.equal(informe.naturaleza, 'sintetica');
  assert.equal(informe.schema, 'dietas-informes-demo');
  assert.equal(catalogo.estado, 'ejemplo');
  assert.ok(catalogo.version.startsWith('propuesta:'));
  assert.equal(criterio.naturaleza, 'sintetica');
  assert.equal(criterio.schema, 'dietas-informes-config-demo');
  assert.ok(informe.registros.some(r => r.referencia === casos.informes.referencia && r.conceptos_centimos.manutencion === casos.informes.importe_centimos));
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
  const medida = await page.evaluate(() => ({
    ancho_css: innerWidth, dpr: devicePixelRatio, escala_visual: visualViewport.scale,
    zoom_css: getComputedStyle(document.documentElement).zoom,
    desbordamiento: document.documentElement.scrollWidth > document.documentElement.clientWidth,
    alto: document.documentElement.scrollHeight, ventana: innerHeight,
  }));
  verificarZoom(medida, ancho, factor);
  assert.equal(medida.desbordamiento, false);
  const desplazamiento = await page.evaluate(() => {
    const candidatos = [document.scrollingElement, document.querySelector('#espacio-trabajo'), document.querySelector('.portal-superficie')].filter(Boolean);
    return candidatos.map(el => {
      el.scrollTop = 0;
      el.scrollTop = el.scrollHeight;
      const despues = el.scrollTop;
      return { alto: el.scrollHeight, ventana: el.clientHeight, movio: despues > 0, overflow: getComputedStyle(el).overflowY };
    });
  });
  const contenidoAlto = desplazamiento.some(x => x.alto > x.ventana + 2 && ['auto', 'scroll'].includes(x.overflow));
  if (contenidoAlto) assert.ok(desplazamiento.some(x => x.movio), `scroll_documento:${JSON.stringify(desplazamiento)}`);
  return { ancho, zoom: factor * 100, scroll_documento: desplazamiento.some(x => x.movio) };
}

async function teclado(page) {
  await page.keyboard.press('Tab');
  const foco = await page.evaluate(() => {
    const el = document.activeElement;
    return el && el !== document.body && el.getClientRects().length > 0 && el.matches(':focus-visible');
  });
  assert.equal(foco, true, 'foco_teclado');
}

async function recorrerPagina(chrome, origen, nombre, idioma, ancho, factor, resultado) {
  await chrome.zoom(factor);
  const context = await chrome.browser.newContext({ viewport: { width: ancho, height: 900 }, serviceWorkers: 'block', acceptDownloads: false });
  const red = { bloqueadas: 0, descargas: 0, cookies: 0, respuestasError: 0, muestra: [] };
  const errores = [];
  const rutaDatos = casos.datos[nombre === 'informes' ? 0 : 1];
  let falloFixture = nombre === 'catalogo' && idioma === 'es' && ancho === 1440 && factor === 1;
  try {
    await context.route('**/*', async route => {
      const request = route.request();
      let u;
      try { u = new URL(request.url()); } catch { red.bloqueadas++; await route.abort(); return; }
      const consultaIdioma = Object.values(casos.paginas).includes(u.pathname) && /^\?lang=(?:es|en)$/u.test(u.search);
      const version = u.pathname.startsWith('/web/static/') && /^\?v=[A-Za-z0-9._-]{1,80}$/u.test(u.search);
      if (u.origin !== origen || request.method() !== 'GET' || !ficheroPermitido(u.pathname) || (u.search && !consultaIdioma && !version) || u.hash) {
        red.bloqueadas++; red.muestra.push({ interno: u.origin === origen, ruta: u.origin === origen ? u.pathname + u.search : 'externo' }); await route.abort(); return;
      }
      if (falloFixture && u.pathname === rutaDatos) {
        falloFixture = false;
        await route.fulfill({ status: 503, contentType: 'application/json', body: '{}' });
        return;
      }
      await route.continue();
    });
    await context.routeWebSocket('**/*', route => { red.bloqueadas++; route.close(); });
    const page = await context.newPage();
    page.on('pageerror', e => errores.push(e.message));
    page.on('download', () => red.descargas++);
    page.on('response', r => { if (r.status() >= 400 && r.status() !== 503) red.respuestasError++; if (r.headers()['set-cookie']) red.cookies++; });
    const destino = `${origen}${casos.paginas[nombre]}?lang=${idioma}`;
    const respuesta = await page.goto(destino, { waitUntil: 'networkidle', timeout: 20000 });
    assert.equal(respuesta.status(), 200);
    assert.equal(await page.locator('html').getAttribute('lang'), idiomas[idioma].codigo);
    const t = idiomas[idioma];
    if (nombre === 'informes') {
      await page.locator('[data-dietas-informes]').waitFor();
      await page.getByText(t.texto_sintetico_informes, { exact: true }).waitFor();
      for (const etiqueta of [t.texto_exportar, t.texto_imprimir]) assert.equal(await page.getByRole('button', { name: etiqueta }).isDisabled(), true);
      await page.locator('[data-dietas-informes-filtros]').waitFor({ state: 'visible' });
      await page.locator('select[name=persona]').selectOption(casos.informes.persona);
      await page.locator('select[name=unidad]').selectOption(casos.informes.unidad);
      await page.locator('input[name=desde]').fill(casos.informes.desde);
      await page.locator('input[name=hasta]').fill(casos.informes.hasta);
      await page.locator('[data-dietas-informes-filtros] button[type=submit]').click();
      await page.locator('[data-dietas-informes-listado] tbody tr').first().waitFor();
      assert.equal(await page.locator('[data-dietas-informes-listado] tbody tr').count(), 2);
      assert.ok((await page.locator('[data-dietas-informes-listado]').innerText()).includes(casos.informes.referencia));
    } else {
      await page.locator('[data-dietas-catalogo]').waitFor();
      if (idioma === 'es' && ancho === 1440 && factor === 1) {
        await page.getByRole('button', { name: t.texto_reintentar }).click();
      }
      await page.locator('[data-dietas-catalogo] [role=status]').getByText(t.texto_sintetico_catalogo, { exact: true }).waitFor();
      await page.locator('[data-dietas-catalogo] tbody tr').nth(casos.catalogo.fila).locator('button').click();
      const editor = page.locator('[data-dietas-catalogo-editor]');
      await editor.locator('input[name=importe_propuesto]').fill(casos.catalogo[`importe_${idioma}`]);
      await editor.locator('textarea[name=motivo]').fill(casos.catalogo[`motivo_${idioma}`]);
      await editor.locator('button[type=submit]').click();
      await page.getByText(t.texto_limite_propuesta, { exact: true }).waitFor();
      assert.ok((await editor.innerText()).includes(t.texto_importe));
      assert.equal(await editor.getByRole('button', { name: t.texto_publicar }).isDisabled(), true);
    }
    await teclado(page);
    const pantalla = await comprobarPantalla(page, ancho, factor);
    await comprobarPrivacidad(page, context);
    assert.deepEqual(errores, [], `${nombre}/${idioma}/${ancho}/${factor}: javascript`);
    assert.equal(red.descargas, 0, 'descargas'); assert.equal(red.cookies, 0, 'cookies');
    assert.equal(red.respuestasError, 0, 'respuesta_http'); assert.equal(red.bloqueadas, 0, `peticiones_bloqueadas:${JSON.stringify(red.muestra)}`);
    resultado.push({ pagina: nombre, idioma, ...pantalla, peticiones_bloqueadas: red.bloqueadas });
  } finally { await context.close(); }
}

export async function ejecutar(argv) {
  const { fuente, modo } = opciones(argv);
  const ausentes = comprobarFuente(fuente, casos);
  if (ausentes.length) {
    process.stderr.write(`${idiomas.es.dependencia}\n${JSON.stringify({ ausentes })}\n`);
    return 2;
  }
  revisarDatos(fuente);
  if (modo === 'plan') {
    process.stdout.write(`${idiomas.es.titulo_plan}\n`);
    return 0;
  }
  const modulo = process.env.VEC_PLAYWRIGHT_MODULE;
  if (!modulo || !path.isAbsolute(modulo) || !fs.statSync(modulo).isFile()) throw new Error('playwright_local_pendiente');
  const { chromium } = await import(pathToFileURL(modulo).href);
  const scratch = fs.mkdtempSync(path.join(os.tmpdir(), 'vec-dietas-previews-'));
  fs.chmodSync(scratch, 0o700);
  const server = crearServidor(fuente);
  let chrome;
  try {
    const origen = await server.escuchar();
    chrome = await chromeAccesible(chromium, scratch);
    const resultado = [];
    for (const idioma of Object.keys(idiomas)) for (const nombre of Object.keys(casos.paginas)) {
      for (const [ancho, factor] of [[1440, 1], [390, 1], [1440, 2]]) await recorrerPagina(chrome, origen, nombre, idioma, ancho, factor, resultado);
    }
    assert.equal(server.contadores().denegadas, 0);
    process.stdout.write(`${idiomas.es.resultado}\n${JSON.stringify({ casos: resultado.length, resultado, servidor: server.contadores() })}\n`);
    return 0;
  } finally {
    await chrome?.cerrar();
    if (server.servidor.listening) await server.cerrar();
    fs.rmSync(scratch, { recursive: true, force: true });
  }
}

if (process.argv[1] && path.resolve(process.argv[1]) === path.resolve(new URL(import.meta.url).pathname)) {
  ejecutar(process.argv.slice(2)).then(code => { process.exitCode = code; }, error => {
    process.stderr.write(`${error.message}\n`); process.exitCode = 1;
  });
}
