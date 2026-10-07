#!/usr/bin/env node
// Guardia de transporte sobre dos servidores efímeros propios, sin VEC.
import http from 'node:http';
import { once } from 'node:events';
import assert from 'node:assert/strict';
import { pathToFileURL } from 'node:url';
import { interceptar } from './recorrer.mjs';

const { chromium } = await import(pathToFileURL(process.env.VEC_PLAYWRIGHT_MODULE).href);
let destinos = 0, efectos = 0;
const destino = http.createServer((_req, res) => { destinos++; res.end(); });
destino.listen(0, '127.0.0.1'); await once(destino, 'listening');
const destinoUrl = `http://127.0.0.1:${destino.address().port}`;
const servidor = http.createServer((req, res) => {
  if (req.method === 'POST') efectos++;
  if (req.url === '/salto') { res.writeHead(302, { Location: destinoUrl }); res.end(); }
  else { res.writeHead(200, { 'Content-Type': 'text/html' }); res.end('<!doctype html><html lang="es"><title>Fixture</title><body></body></html>'); }
});
servidor.listen(0, '127.0.0.1'); await once(servidor, 'listening');
const origen = `http://127.0.0.1:${servidor.address().port}`;
let browser;
try {
  browser = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
  const context = await browser.newContext({ serviceWorkers: 'block' });
  const datos = { bloqueadas: 0, red_fallida: 0 };
  await context.route('**/*', route => interceptar(route, origen, datos));
  const page = await context.newPage();
  await page.goto(origen);
  assert.equal(await page.evaluate(() => fetch('/salto').then(() => false, () => true)), true);
  assert.equal(await page.evaluate(() => fetch('/api/vec/bolsa/llamamientos', { method: 'POST', body: '{}' }).then(() => false, () => true)), true);
  assert.equal(destinos, 0); assert.equal(efectos, 0); assert.equal(datos.bloqueadas, 2);
  console.log(JSON.stringify({ chrome_sistema: true, redirecciones_destino: destinos, escrituras_fixture: efectos }));
} finally {
  if (browser) await browser.close();
  servidor.closeAllConnections(); destino.closeAllConnections();
  await Promise.all([new Promise(r => servidor.close(r)), new Promise(r => destino.close(r))]);
}
