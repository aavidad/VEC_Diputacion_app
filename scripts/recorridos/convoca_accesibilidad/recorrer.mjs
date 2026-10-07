#!/usr/bin/env node
import fs from 'node:fs';
import path from 'node:path';
import { pathToFileURL } from 'node:url';
import { rutaExterna, prepararSalida, validarOrigen, idiomas } from '../../recorridos-f/config.mjs';
import { interceptar } from '../../recorridos-f/recorrer.mjs';
import { chromeAccesible, comprobarAccesibilidad, activarLectura } from '../../recorridos-f/a11y.mjs';

export const plan = JSON.parse(fs.readFileSync(new URL('casos.json', import.meta.url)));
const preferido = process.env.LANG?.split(/[_.-]/)[0];
const idiomaCLI = Object.hasOwn(idiomas.disponibles, preferido) ? preferido : idiomas.respaldo;
const mensajes = JSON.parse(fs.readFileSync(new URL(`../../recorridos-f/${idiomas.disponibles[idiomaCLI].mensajes}`, import.meta.url)));
const API = '/api/publico/bolsa/convocatorias';
function exigir(ok) { if (!ok) throw new Error('config'); }

export function validarConfig(c) {
  exigir(c?.version === 1 && c.sintetico === true && c.entorno_controlado === true);
  const origen = validarOrigen(c.origen);
  exigir(typeof c.commit_servido === 'string' && /^[0-9a-f]{40}$/.test(c.commit_servido));
  exigir(typeof c.identificador_publico === 'string' && /^[a-z0-9][a-z0-9-]{2,79}$/.test(c.identificador_publico));
  // Superficie anónima: no admite identidad, certificados ni cabeceras del operador.
  exigir(Object.keys(c).every(k => ['version', 'sintetico', 'entorno_controlado', 'origen', 'commit_servido', 'identificador_publico'].includes(k)));
  return { ...c, origen };
}

export async function interceptarPublico(route, origen, datos) {
  if (route.request().method() !== 'GET') { datos.bloqueadas++; await route.abort(); return; }
  return interceptar(route, origen, datos);
}

async function contrato(page, r, tipo, identificador) {
  if (r.status() !== 200) throw new Error('http');
  const dto = await r.json();
  const valido = await page.evaluate(({ dto, tipo, identificador }) => {
    try {
      if (dto?.fuente?.demostracion !== false) return false;
      const contrato = globalThis.VECBolsaContratoV2;
      if (tipo === 'listado') {
        contrato.validarListado(dto);
        return dto.convocatorias.some(c => c.identificador_publico === identificador);
      }
      contrato.validarDetalle(dto);
      return dto.convocatoria.identificador_publico === identificador;
    } catch { return false; }
  }, { dto, tipo, identificador });
  if (!valido) throw new Error('contrato');
}

export async function recorrer(c, chromium, salida, transporte = interceptarPublico, tipo = 'aplicacion') {
  const resultado = { version: 1, tipo_ejecucion: tipo, estado: 'EN_CURSO',
    commit_servido_declarado: c.commit_servido, servidor_instalado_verificado: false,
    flujo_funcional_completado: false, persistencia_acreditada: false, pasos: [] };
  const guardar = () => fs.writeFileSync(path.join(salida, 'resultado.json'), JSON.stringify(resultado, null, 2), { mode: 0o600 });
  let chrome;
  try {
    chrome = await chromeAccesible(chromium, salida);
    for (const idioma of plan.idiomas) for (const { ancho, zoom } of plan.visitas) {
      await chrome.zoom(zoom);
      const datos = { idioma, ancho, zoom_porcentaje: zoom * 100, bloqueadas: 0, red_fallida: 0, errores_js: 0, consola_error: 0, http_fallidos: 0, accesibilidad: [] };
      resultado.pasos.push(datos);
      const context = await chrome.browser.newContext({ viewport: { width: ancho, height: ancho === 390 ? 844 : 900 },
        locale: idioma === 'es' ? 'es-ES' : 'en-GB', serviceWorkers: 'block' });
      try {
        await context.route('**/*', route => transporte(route, c.origen, datos));
        await context.routeWebSocket('**/*', route => { datos.bloqueadas++; route.close(); });
        const page = await context.newPage();
        page.on('pageerror', () => datos.errores_js++);
        page.on('requestfailed', () => datos.red_fallida++);
        page.on('console', message => { if (message.type() === 'error') datos.consola_error++; });
        page.on('response', r => { if (r.status() >= 400) datos.http_fallidos++; });
        const listado = page.waitForResponse(r => new URL(r.url()).pathname === API, { timeout: 20000 });
        const [r, entrada] = await Promise.all([listado, page.goto(`${c.origen}/bolsa/?lang=${idioma}`, { waitUntil: 'domcontentloaded', timeout: 20000 })]);
        if (entrada?.status() !== 200) throw new Error('http');
        await contrato(page, r, 'listado', c.identificador_publico);
        const enlace = page.locator(`article[data-identificador="${c.identificador_publico}"] .enlace-detalle`);
        await enlace.waitFor({ state: 'visible', timeout: 10000 });
        await page.waitForLoadState('networkidle', { timeout: 20000 });
        const comprobar = async superficie => {
          if (await page.locator('html').getAttribute('lang') !== idioma) throw new Error('idioma');
          const lectura = await comprobarAccesibilidad(page, '#contenido-principal', superficie, ancho, zoom);
          datos.accesibilidad.push(lectura);
          if (lectura.estado !== 'COMPROBADO') throw new Error('accesibilidad');
        };
        await comprobar('listado');
        const pendiente = page.waitForResponse(r => new URL(r.url()).pathname === `${API}/${c.identificador_publico}`, { timeout: 20000 });
        const [detalle] = await Promise.all([pendiente, activarLectura(page, enlace)]);
        await contrato(page, detalle, 'detalle', c.identificador_publico);
        await page.locator('#contenido-detalle').waitFor({ state: 'visible', timeout: 10000 });
        await comprobar('detalle');
        datos.cookies = (await context.cookies()).length;
        datos.almacenamiento = await page.evaluate(async () => localStorage.length + sessionStorage.length
          + (await indexedDB.databases()).length + (await caches.keys()).length);
        if ([datos.bloqueadas, datos.red_fallida, datos.errores_js, datos.consola_error, datos.http_fallidos,
          datos.cookies, datos.almacenamiento].some(Boolean)) throw new Error('controles');
        datos.estado = 'COMPROBADO';
      } finally { await context.close(); guardar(); }
    }
    resultado.estado = 'ACCESIBILIDAD_PARCIAL_COMPROBADA';
    return resultado;
  } catch (e) {
    resultado.estado = 'CORTADO';
    resultado.corte = ['http', 'contrato', 'idioma', 'accesibilidad', 'controles', 'zoom_nativo', 'teclado_accesibilidad'].includes(e.message) ? e.message : 'navegador';
    throw e;
  } finally {
    try { await chrome?.cerrar(); }
    catch (e) { resultado.estado = 'CORTADO'; resultado.corte = 'navegador'; throw e; }
    finally { guardar(); }
  }
}

async function main() {
  const args = process.argv.slice(2);
  if (args.length === 1 && args[0] === '--plan') { console.log(JSON.stringify(plan, null, 2)); return; }
  let iniciado = false;
  try {
    exigir(args.length === 4 && args[0] === '--config' && args[2] === '--salida');
    const c = validarConfig(JSON.parse(fs.readFileSync(rutaExterna(args[1], { privado: true }), 'utf8')));
    const { chromium } = await import(pathToFileURL(rutaExterna(process.env.VEC_PLAYWRIGHT_MODULE)).href);
    exigir(typeof chromium?.launchPersistentContext === 'function');
    const salida = prepararSalida(args[3]);
    iniciado = true;
    await recorrer(c, chromium, salida);
  } catch { console.error(iniciado ? mensajes.fallo : mensajes.entrada); process.exitCode = 1; }
}
if (process.argv[1] && pathToFileURL(path.resolve(process.argv[1])).href === import.meta.url) await main();
