#!/usr/bin/env node
// Fixtures propios: no acredita instalación ni lecturas contra VEC.
import http from 'node:http';
import fs from 'node:fs';
import path from 'node:path';
import assert from 'node:assert/strict';
import { once } from 'node:events';
import { pathToFileURL } from 'node:url';
import { chromeAccesible, comprobarAccesibilidad, activarLectura } from './a11y.mjs';
import { idiomas } from './config.mjs';

assert.ok(process.argv.slice(2).length === 0 || (process.argv.length === 3 && process.argv[2] === '--regresiones-ux'));
const regresionesUX = process.argv[2] === '--regresiones-ux';
const { chromium } = await import(pathToFileURL(process.env.VEC_PLAYWRIGHT_MODULE).href);
const scratch = process.env.VEC_F_TEST_SCRATCH;
assert.ok(scratch && path.isAbsolute(scratch));
const salida = fs.mkdtempSync(path.join(scratch, 'smoke-a11y-'));
let efectos = 0;
const server = http.createServer((req, res) => {
  if (req.method !== 'GET') efectos++;
  res.writeHead(200, { 'Content-Type': 'text/html' });
  res.end('<!doctype html><html><body></body></html>');
});
server.listen(0, '127.0.0.1'); await once(server, 'listening');
const origen = `http://127.0.0.1:${server.address().port}`;
let chrome;
let casos = 0;
try {
  chrome = await chromeAccesible(chromium, salida);
  const probar = async (lang, width, factor, defecto = '') => {
    await chrome.zoom(factor);
    const context = await chrome.browser.newContext({ viewport: { width, height: 900 }, serviceWorkers: 'block' });
    try {
      await context.route('**/*', route => new URL(route.request().url()).origin === origen && route.request().method() === 'GET' ? route.continue() : route.abort());
      await context.routeWebSocket('**/*', route => route.close());
      const page = await context.newPage();
      let errores = 0;
      page.on('pageerror', () => errores++);
      await page.goto(origen);
      const t = idiomas.disponibles[lang].accesibilidad_fixture;
      await page.setContent(`<!doctype html><html lang="${lang}"><title>${t.titulo}</title>
        <style>body{margin:24px}button,input{margin:12px;max-width:85%;box-sizing:border-box} :focus-visible{outline:3px solid black;outline-offset:2px}
        ${defecto === 'foco' ? ':focus-visible{outline:none}' : ''}
        ${defecto === 'redondeado' ? 'button{border-radius:12px;padding:12px}' : ''}
        ${defecto === 'etiqueta_opacidad' ? 'label{opacity:0}' : ''}
        ${defecto === 'etiqueta_ancestro_opacidad' ? '.etiqueta-contenedor{opacity:0}' : ''}
        ${defecto === 'contorno_permanente' ? 'button,input{outline:3px solid black;outline-offset:2px}' : ''}</style>
        <main id="lectura"><h1>${t.titulo}</h1>
        ${defecto === 'etiqueta' ? '' : `<div class="etiqueta-contenedor"><label for="buscar">${t.etiqueta}</label></div>`}<input id="buscar" ${defecto === 'etiqueta' ? `aria-label="${t.etiqueta}"` : ''}>
        <button ${defecto === 'nombre' ? '' : `aria-label="${t.abrir}"`} id="abrir" ${defecto === 'tab' ? 'tabindex="-1"' : ''}></button>
        <div style="height:1100px"></div><button id="final">${t.volver}</button>
        </main>${defecto === 'tapado' ? '<div style="position:fixed;inset:0;z-index:1000"></div>' : ''}</html>`);
      if (['trampa_ambas', 'trampa_avance'].includes(defecto)) await page.evaluate(ambas => {
        document.querySelector('#final').addEventListener('keydown', e => {
          if (e.key === 'Tab' && (ambas || !e.shiftKey)) e.preventDefault();
        });
      }, defecto === 'trampa_ambas');
      const fallo = Boolean(defecto) && defecto !== 'redondeado';
      const prueba = await comprobarAccesibilidad(page, '#lectura', 'fixture', width, factor);
      assert.equal(prueba.estado, fallo ? 'CORTADO' : 'COMPROBADO', JSON.stringify(prueba));
      if (defecto === 'etiqueta') assert.equal(prueba.etiquetas_ausentes, 1);
      if (defecto === 'nombre') assert.equal(prueba.nombres_ausentes, 1);
      if (defecto === 'foco') assert.equal(prueba.foco_invisible, 3);
      if (defecto === 'tapado') assert.equal(prueba.foco_tapado, 3);
      if (defecto === 'tab') assert.equal(prueba.controles_fuera_de_tab, 1);
      if (['etiqueta_opacidad', 'etiqueta_ancestro_opacidad'].includes(defecto)) assert.equal(prueba.etiquetas_ausentes, 1);
      if (defecto === 'contorno_permanente') assert.equal(prueba.foco_invisible, 3);
      if (['trampa_ambas', 'trampa_avance'].includes(defecto)) assert.equal(prueba.trampas_teclado, 1);
      if (!fallo) {
        assert.equal(prueba.teclado_comprobados, 3);
        await page.evaluate(() => { document.querySelector('#abrir').addEventListener('click', () => { document.body.dataset.abierto = 'true'; }); });
        await activarLectura(page, page.locator('#abrir'));
        assert.equal(await page.locator('body').getAttribute('data-abierto'), 'true');
      }
      assert.equal(errores, 0); assert.deepEqual(await context.cookies(), []);
      assert.equal(await page.evaluate(() => localStorage.length + sessionStorage.length), 0);
      // El informe no conserva textos de etiquetas ni nombres accesibles.
      const raw = JSON.stringify(prueba);
      for (const texto of Object.values(t)) assert.ok(!raw.includes(texto));
      casos++;
    } finally { await context.close(); }
  };
  if (!regresionesUX) {
    for (const lang of Object.keys(idiomas.disponibles)) for (const [width, factor] of [[1440, 1], [390, 1], [1440, 2]]) await probar(lang, width, factor);
    for (const defecto of ['etiqueta', 'nombre', 'foco', 'tapado', 'tab']) await probar(idiomas.respaldo, 1440, 1, defecto);
  }
  for (const defecto of ['redondeado', 'etiqueta_opacidad', 'etiqueta_ancestro_opacidad', 'contorno_permanente', 'trampa_ambas', 'trampa_avance']) await probar(idiomas.respaldo, 1440, 1, defecto);
  assert.equal(efectos, 0);
  await chrome.cerrar(); chrome = null;
  assert.deepEqual(fs.readdirSync(salida), []);
  console.log(JSON.stringify({ chrome_sistema: true, fixtures_comprobados: casos, zoom_nativo_200: !regresionesUX, regresiones_ux_comprobadas: 6, escrituras_fixture: efectos, perfil_eliminado: true }));
} finally {
  await chrome?.cerrar();
  server.closeAllConnections();
  await new Promise(resolve => server.close(resolve));
  fs.rmSync(salida, { recursive: true, force: true });
}
