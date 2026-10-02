import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs/promises';
import path from 'node:path';
import os from 'node:os';
import { spawnSync } from 'node:child_process';
import { pathToFileURL } from 'node:url';
import { argumentos, plan, paginaConCamposNativos, html } from './recorrer.mjs';
import { comprobarAccesibilidad } from '../../recorridos-f/a11y.mjs';
import { recursoPermitido, leerEstatico, entregar } from './guardas.mjs';
const casos = await plan();
const ORIGEN = casos.origen;

test('host usa el scroll de espacio-trabajo cuando el documento está cerrado en escritorio',
  { skip: !process.argv.includes('--ensayar-host') }, async () => {
    const { chromium } = await import(pathToFileURL(process.env.VEC_PLAYWRIGHT_MODULE).href);
    const browser = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
    try {
      const contexto = await browser.newContext({ viewport: { width: 1440, height: 900 }, serviceWorkers: 'block' });
      const red = { entregadas: 0, bloqueadas: 0 };
      const raiz = path.resolve(new URL('../../../web/static/', import.meta.url).pathname);
      await contexto.route('**/*', route => entregar(route, raiz, html(casos.idiomas[0], casos), red, casos));
      await contexto.routeWebSocket('**/*', route => route.close());
      const page = await contexto.newPage();
      await page.goto(`${casos.origen}/fixture?lang=${casos.idiomas[0]}`);
      await page.locator('#lectura').evaluate((el, titulo) => {
        const espacio = document.createElement('div'); espacio.style.height = '150vh'; espacio.setAttribute('aria-hidden', 'true');
        const boton = document.createElement('button'); boton.id = 'control-prueba'; boton.textContent = titulo;
        el.append(espacio, boton);
      }, casos.textos[casos.idiomas[0]].titulo);
      assert.deepEqual(await page.evaluate(() => ({ documento: getComputedStyle(document.documentElement).overflow,
        cuerpo: getComputedStyle(document.body).overflow, marco: getComputedStyle(document.querySelector('main')).overflow,
        altura: document.querySelector('main').clientHeight })),
      { documento: 'hidden', cuerpo: 'hidden', marco: 'auto', altura: 900 });
      await paginaConCamposNativos(page).keyboard.press('Tab');
      assert.equal(await page.locator('#control-prueba').evaluate(el => el === document.activeElement), true);
      assert.ok(await page.locator('#espacio-trabajo').evaluate(el => el.scrollTop) > 0);
      const medicion = await comprobarAccesibilidad(paginaConCamposNativos(page), '#lectura', 'contrato_host', 1440, 1);
      assert.equal(medicion.foco_tapado, 0, 'el scroll real revela los cinco puntos del control');
      assert.equal(medicion.desbordamiento, false);
      await page.locator('main').evaluate(el => { el.removeAttribute('id'); el.scrollTop = 0; });
      assert.equal(await page.locator('main').evaluate(el => getComputedStyle(el).overflow), 'visible', 'el mutante pierde la autoridad del scroll');
      assert.equal(red.bloqueadas, 0);
      await contexto.close();
    } finally { await browser.close(); }
  });

test('plan funciona con módulo Playwright inexistente y no abre Chrome', async () => {
  const r = spawnSync(process.execPath, [new URL('./recorrer.mjs', import.meta.url).pathname, '--plan'],
    { env: { PATH: process.env.PATH, VEC_PLAYWRIGHT_MODULE: '/inexistente' }, encoding: 'utf8' });
  assert.equal(r.status, 0); assert.equal(r.stderr, '');
  const p = JSON.parse(r.stdout);
  assert.equal(p.alcance, 'preview_offline_sintetica');
  assert.equal(p.pantallas.length * p.vistas.length * p.idiomas.length, 24);
  assert.deepEqual(argumentos(['--plan']), { plan: true });
  assert.throws(() => argumentos(['--plan', '--salida', '/tmp/x']));
  assert.throws(() => argumentos(['--salida', 'relativa']));
});

test('frontera entrega bytes locales y corta red ajena, API y POST antes de leer', async () => {
  for (const [metodo, url] of [['POST', `${ORIGEN}/fixture?lang=es`], ['GET', 'https://example.invalid/comun/idioma.js'],
    ['GET', `${ORIGEN}/api/interna/cronos/saldo`], ['GET', `${ORIGEN}/textos/%2fsecreto.json`],
    ['GET', `${ORIGEN}//textos/es/cronos.json`], ['GET', `${ORIGEN}/textos/es/cronos.json?actor=otra`],
    ['GET', `${ORIGEN}/fixture?lang=fr`], ['GET', `${ORIGEN}/portal-empleado/../../etc/passwd`]]) {
    let abortadas = 0;
    const contadores = { entregadas: 0, bloqueadas: 0 };
    assert.equal(recursoPermitido(url, metodo, casos), null);
    await entregar({ request: () => ({ url: () => url, method: () => metodo }), abort: async () => abortadas++,
      fulfill: async () => assert.fail('no debe entregar') }, '/inexistente', '', contadores, casos);
    assert.equal(abortadas, 1); assert.equal(contadores.bloqueadas, 1);
  }
  assert.equal(recursoPermitido(`${ORIGEN}/comun/idioma.js?v=version-1`, 'GET', casos), 'comun/idioma.js');
  let entregas = 0;
  await entregar({ request: () => ({ url: () => `${ORIGEN}/fixture?lang=en`, method: () => 'GET' }),
    fulfill: async respuesta => { entregas++; assert.equal(respuesta.status, 200); assert.equal(respuesta.headers['Set-Cookie'], undefined); },
    abort: async () => assert.fail('fixture permitido') }, '/inexistente', '<html></html>', { entregadas: 0, bloqueadas: 0 }, casos);
  assert.equal(entregas, 1);
});

test('lectura no sigue enlaces ni abandona raíz y respeta el límite de tamaño', async () => {
  const scratch = await fs.mkdtemp(path.join(os.tmpdir(), 'cronos-guardas-'));
  try {
    const origen = path.join(scratch, 'static'); await fs.mkdir(origen);
    await fs.writeFile(path.join(origen, 'ok.json'), '{}');
    await fs.symlink(path.join(origen, 'ok.json'), path.join(origen, 'enlace.json'));
    await fs.writeFile(path.join(origen, 'grande.json'), Buffer.alloc(2 * 1024 * 1024 + 1));
    assert.equal((await leerEstatico(origen, 'ok.json')).body.toString(), '{}');
    await assert.rejects(leerEstatico(origen, '../fuera.json'));
    await assert.rejects(leerEstatico(origen, 'enlace.json'));
    await assert.rejects(leerEstatico(origen, 'grande.json'));
  } finally { await fs.rm(scratch, { recursive: true, force: true }); }
});

test('casos dejan previsión y saldo desconocidos y no atribuyen un registro', async () => {
  const casos = await plan();
  assert.equal(casos.saldo.resumen.saldo_minutos, null);
  assert.equal(casos.saldo.resumen.previstos_minutos, null);
  assert.equal(casos.movimientos.absentismos[0].pendiente_justificar, true);
  assert.equal(casos.movimientos.correcciones[0].estado, 'pendiente_responsable');
  assert.equal(Object.hasOwn(casos, 'recibo'), false);
});

test('Chrome sale de fecha nativa y conserva una trampa real al llegar al tope',
  { skip: !process.argv.includes('--ensayar-chrome') }, async () => {
    const { chromium } = await import(pathToFileURL(process.env.VEC_PLAYWRIGHT_MODULE).href);
    const browser = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
    try {
      const contexto = await browser.newContext({ viewport: { width: 1440, height: 900 }, serviceWorkers: 'block' });
      const titulo = casos.textos[casos.idiomas[0]].titulo;
      await contexto.route('**/*', route => recursoPermitido(route.request().url(), route.request().method(), casos) === 'fixture'
        ? route.fulfill({ contentType: 'text/html', body: `<html><main id="lectura"><label>${titulo}<input id="fecha" type="date" value="2026-10-01"></label><button id="salida">${titulo}</button></main></html>` }) : route.abort());
      await contexto.routeWebSocket('**/*', route => route.close());
      const page = await contexto.newPage();
      await page.goto(`${casos.origen}/fixture?lang=${casos.idiomas[0]}`);
      const agrupada = paginaConCamposNativos(page);
      await page.locator('#fecha').focus();
      await agrupada.keyboard.press('Tab');
      assert.equal(await page.locator('#salida').evaluate(el => el === document.activeElement), true);
      await agrupada.keyboard.press('Shift+Tab');
      assert.equal(await page.locator('#fecha').evaluate(el => el === document.activeElement), true);
      await page.locator('#fecha').evaluate(el => el.addEventListener('keydown', e => { if (e.key === 'Tab') e.preventDefault(); }));
      await agrupada.keyboard.press('Tab');
      assert.equal(await page.locator('#fecha').evaluate(el => el === document.activeElement), true);
      const resultado = await comprobarAccesibilidad(agrupada, '#lectura', 'mutante_atrapado', 1440, 1);
      assert.ok(resultado.trampas_teclado > 0);
      assert.equal(resultado.estado, 'CORTADO');
      await contexto.close();
    } finally { await browser.close(); }
  });
