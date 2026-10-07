#!/usr/bin/env node
import fs from 'node:fs';
import path from 'node:path';
import { pathToFileURL } from 'node:url';
import { rutaExterna, prepararSalida, validarOrigen, idiomas } from '../../recorridos-f/config.mjs';
import { interceptar } from '../../recorridos-f/recorrer.mjs';
import { chromeAccesible, comprobarAccesibilidad, activarLectura } from '../../recorridos-f/a11y.mjs';

export const plan = JSON.parse(fs.readFileSync(new URL('casos.json', import.meta.url)));
export const API = '/api/interna/cronos/permisos/propio';
const MODULO = '/portal-empleado/modulos/cronos/cliente-solicitudes-http.js';
const VISTA = '/portal-empleado/modulos/cronos/vista-permisos-propios.js';
const HISTORIAL = '[data-cronos-permisos-propios] section[aria-labelledby="cronos-historial-titulo"]';
// Consultas que el shell vigente lanza al abrir su URL directa. Sin comodines
// ni rutas elegidas en configuración: el resto de API se deniega.
const SHELL = new Set(['/api/vec/modules', '/api/vec/session', '/api/vec/usuarios/mis-preferencias',
  '/api/vec/usuarios/mis-correos', '/api/vec/usuarios/mi-imagen', '/api/vec/bolsa/bolsas']);
const preferido = process.env.LANG?.split(/[_.-]/)[0];
const idiomaCLI = Object.hasOwn(idiomas.disponibles, preferido) ? preferido : idiomas.respaldo;
const mensajes = JSON.parse(fs.readFileSync(new URL(`../../recorridos-f/${idiomas.disponibles[idiomaCLI].mensajes}`, import.meta.url)));
function exigir(ok) { if (!ok) throw new Error('config'); }

export function validarConfig(c) {
  exigir(c?.version === 1 && c.sintetico === true && c.entorno_controlado === true);
  const origen = validarOrigen(c.origen);
  exigir(typeof c.commit_servido === 'string' && /^[0-9a-f]{40}$/.test(c.commit_servido));
  exigir(Number.isSafeInteger(c.anio) && c.anio >= 2000 && c.anio <= 2100);
  exigir(c.identidad && Object.keys(c.identidad).length === 2 && ['certificado', 'clave'].every(k => typeof c.identidad[k] === 'string' && path.isAbsolute(c.identidad[k])));
  // No perfil, actor, huella ni permiso solicitados por el operador. El servidor
  // deriva la identidad mTLS y autoriza la lectura con su autoridad central.
  exigir(Object.keys(c).every(k => ['version', 'sintetico', 'entorno_controlado', 'origen', 'commit_servido', 'anio', 'identidad'].includes(k)));
  return { ...c, origen };
}

export async function interceptarHistorial(route, c, datos) {
  const req = route.request(), u = new URL(req.url());
  const canonica = !u.pathname.includes('%') && !u.pathname.includes('//');
  const consulta = u.pathname === API && u.search === `?anio=${c.anio}`;
  const shell = SHELL.has(u.pathname) && !u.search;
  const estatico = /^\/(?:portal-empleado|comun|textos|assets|locales)\//.test(u.pathname);
  if (req.method() !== 'GET' || !canonica || !(consulta || shell || estatico)) {
    datos.bloqueadas++; await route.abort(); return;
  }
  return interceptar(route, c.origen, datos);
}

export async function validarConsulta(page, respuesta, anio) {
  if (respuesta.status() !== 200) throw new Error('http');
  const dto = await respuesta.json();
  const resumen = await page.evaluate(async ({ dto, anio, ruta, vista }) => {
    const montada = ruta => performance.getEntriesByType('resource').map(r => r.name).reverse()
      .find(n => new URL(n).origin === location.origin && new URL(n).pathname === ruta);
    const href = montada(ruta), hrefVista = montada(vista);
    if (!href || !hrefVista) return null;
    try {
      const modulo = await import(href);
      const renderer = await import(hrefVista);
      if (typeof modulo.validarPermisosPropiosCronos !== 'function' || typeof renderer.renderizarPermisosPropiosCronos !== 'function') return null;
      const data = modulo.validarPermisosPropiosCronos(dto, anio);
      const filas = data.solicitudes.map(s => {
        const copia = document.createElement('div');
        copia.innerHTML = renderer.renderizarPermisosPropiosCronos({ estado: 'listo', anio, datos: { ...data, solicitudes: [s] } });
        const row = copia.querySelector('section[aria-labelledby="cronos-historial-titulo"] tbody tr');
        return { ref: s.solicitud_ref, estado: s.estado, clave: JSON.stringify([...row.children].map(c => c.textContent)) };
      });
      // La UI no expone referencias. Sólo puede identificarse una fila cuando
      // su proyección visible corresponde a una única referencia del DTO.
      if (new Set(filas.map(f => f.clave)).size !== filas.length || new Set(filas.map(f => f.ref)).size !== filas.length) return { escenario_insuficiente: true };
      return { total: data.solicitudes.length, concedidos: data.solicitudes.filter(s => s.estado === 'concedido').length,
        filas,
        modulo_version: /^[A-Za-z0-9_.-]{1,128}$/.test(new URL(href).searchParams.get('v') || '') ? new URL(href).searchParams.get('v') : null };
    } catch { return null; }
  }, { dto, anio, ruta: MODULO, vista: VISTA });
  if (!resumen) throw new Error('contrato');
  if (resumen.escenario_insuficiente || resumen.total <= plan.tamano_pagina || !resumen.concedidos) throw new Error('escenario_insuficiente');
  return resumen;
}

export function referenciasVisibles(claves, filas) {
  const mapa = new Map(filas.map(f => [f.clave, f.ref]));
  if (mapa.size !== filas.length || new Set(filas.map(f => f.ref)).size !== filas.length) throw new Error('historial');
  const refs = claves.map(k => mapa.get(k));
  if (refs.some(r => typeof r !== 'string') || new Set(refs).size !== refs.length) throw new Error('historial');
  return refs;
}

export function comprobarReferencias(claves, filas, esperadas) {
  const refs = referenciasVisibles(claves, filas);
  if (refs.length !== esperadas.length || refs.some((r, i) => r !== esperadas[i])) throw new Error('historial');
}

async function alcanzar(page, control) {
  for (let i = 0; i < 256; i++) {
    if (await control.evaluate(el => document.activeElement === el)) return;
    await page.keyboard.press('Tab');
  }
  throw new Error('teclado_accesibilidad');
}

// El portal usa desplazamiento suave. La auditoría común debe observar el foco
// después de ese desplazamiento, sin cambiar CSS ni forzar scrollIntoView.
function tecladoEstable(page) {
  const keyboard = { press: async (...args) => {
    await page.keyboard.press(...args);
    const estable = await page.evaluate(() => new Promise(resolve => {
      let ultimo = '', iguales = 0, frames = 0;
      const medir = () => {
        const posiciones = [scrollX, scrollY];
        for (let e = document.activeElement; e; e = e.parentElement) posiciones.push(e.scrollLeft, e.scrollTop);
        const actual = JSON.stringify(posiciones);
        iguales = actual === ultimo ? iguales + 1 : 0; ultimo = actual;
        if (iguales >= 3) { resolve(true); return; }
        if (++frames >= 120) { resolve(false); return; }
        requestAnimationFrame(medir);
      };
      requestAnimationFrame(medir);
    }));
    if (!estable) throw new Error('teclado_accesibilidad');
  } };
  return new Proxy(page, { get(obj, k) {
    if (k === 'keyboard') return keyboard;
    const value = Reflect.get(obj, k); return typeof value === 'function' ? value.bind(obj) : value;
  } });
}

export async function recorrer(c, chromium, salida, transporte = interceptarHistorial, tipo = 'aplicacion') {
  const resultado = { version: 1, tipo_ejecucion: tipo, estado: 'EN_CURSO', commit_servido_declarado: c.commit_servido,
    servidor_instalado_verificado: false, autenticacion_acreditada: false, persistencia_acreditada: false, flujo_funcional_completado: false, pasos: [] };
  const guardar = () => fs.writeFileSync(path.join(salida, 'resultado.json'), JSON.stringify(resultado, null, 2), { mode: 0o600 });
  let chrome;
  try {
    chrome = await chromeAccesible(chromium, salida);
    for (const idioma of plan.idiomas) for (const { ancho, zoom } of plan.visitas) {
      await chrome.zoom(zoom);
      const datos = { idioma, ancho, zoom_porcentaje: zoom * 100, bloqueadas: 0, red_fallida: 0, errores_js: 0, consola_error: 0, http_fallidos: 0, consultas_api: 0, accesibilidad: [] };
      resultado.pasos.push(datos);
      const context = await chrome.browser.newContext({ viewport: { width: ancho, height: ancho === 390 ? 844 : 900 },
        locale: idiomas.disponibles[idioma].locale, serviceWorkers: 'block',
        clientCertificates: [{ origin: c.origen, certPath: c.identidad.certificado, keyPath: c.identidad.clave }] });
      try {
        await context.route('**/*', route => transporte(route, c, datos));
        await context.routeWebSocket('**/*', route => { datos.bloqueadas++; route.close(); });
        const page = tecladoEstable(await context.newPage());
        page.on('pageerror', () => datos.errores_js++);
        page.on('requestfailed', () => datos.red_fallida++);
        page.on('console', m => { if (m.type() === 'error') datos.consola_error++; });
        page.on('response', r => { if (r.status() >= 400) datos.http_fallidos++; if (new URL(r.url()).pathname === API) datos.consultas_api++; });
        const pendiente = page.waitForResponse(r => new URL(r.url()).pathname === API, { timeout: 20000 });
        const [respuesta, entrada] = await Promise.all([pendiente, page.goto(`${c.origen}/portal-empleado/?lang=${idioma}#cronos-permisos`, { waitUntil: 'domcontentloaded', timeout: 20000 })]);
        if (entrada?.status() !== 200) throw new Error('http');
        const resumen = await validarConsulta(page, respuesta, c.anio);
        datos.consulta_http = 200; datos.modulo_version = resumen.modulo_version;
        await page.locator('[data-cronos-permisos-propios] [data-estado="listo"]').waitFor({ state: 'visible', timeout: 10000 });
        await page.waitForLoadState('networkidle', { timeout: 20000 });
        const scope = page.locator(HISTORIAL);
        const filas = () => scope.locator('tbody tr').count();
        const comprobarFilas = async esperadas => comprobarReferencias(await scope.locator('tbody tr').evaluateAll(rs =>
          rs.map(r => JSON.stringify([...r.children].map(c => c.textContent)))), resumen.filas, esperadas.map(f => f.ref));
        const comprobar = async lectura => {
          if (await page.locator('html').getAttribute('lang') !== idioma) throw new Error('idioma');
          const audit = await comprobarAccesibilidad(page, HISTORIAL, lectura, ancho, zoom);
          datos.accesibilidad.push(audit);
          if (audit.estado !== 'COMPROBADO') throw new Error('accesibilidad');
        };
        if (await filas() !== plan.tamano_pagina) throw new Error('historial');
        await comprobarFilas(resumen.filas.slice(0, plan.tamano_pagina));
        await comprobar('pagina_inicial');
        const siguiente = scope.locator('[data-cronos-historial-pagina="siguiente"]');
        const anterior = scope.locator('[data-cronos-historial-pagina="anterior"]');
        if (await siguiente.isDisabled() || !(await anterior.isDisabled())) throw new Error('historial');
        await activarLectura(page, siguiente);
        await page.waitForFunction(selector => !document.querySelector(selector).disabled, `${HISTORIAL} [data-cronos-historial-pagina="anterior"]`);
        if (await filas() !== Math.min(plan.tamano_pagina, resumen.total - plan.tamano_pagina)) throw new Error('historial');
        await comprobarFilas(resumen.filas.slice(plan.tamano_pagina, plan.tamano_pagina * 2));
        datos.pagina_siguiente = true;
        await activarLectura(page, anterior);
        await page.waitForFunction(selector => document.querySelector(selector).disabled, `${HISTORIAL} [data-cronos-historial-pagina="anterior"]`);
        datos.pagina_anterior = await filas() === plan.tamano_pagina;
        await comprobarFilas(resumen.filas.slice(0, plan.tamano_pagina));
        const filtro = scope.locator('[data-cronos-historial-filtro]');
        await alcanzar(page, filtro);
        await page.keyboard.press('Home');
        const indice = await filtro.evaluate((el, valor) => [...el.options].findIndex(o => o.value === valor), plan.filtro);
        if (indice < 0) throw new Error('historial');
        for (let i = 0; i < indice; i++) await page.keyboard.press('ArrowDown');
        if (await filtro.inputValue() !== plan.filtro || await filas() !== Math.min(plan.tamano_pagina, resumen.concedidos)) throw new Error('historial');
        await comprobarFilas(resumen.filas.filter(f => f.estado === 'concedido').slice(0, plan.tamano_pagina));
        const estados = scope.locator('tbody [data-estado]');
        if (await estados.count() !== await filas()
          || !(await estados.evaluateAll(es => es.every(e => e.dataset.estado === 'concedido')))) throw new Error('historial');
        datos.filtro_teclado = true;
        await comprobar('historial_filtrado');
        datos.cookies = (await context.cookies()).length;
        datos.almacenamiento = await page.evaluate(async () => localStorage.length + sessionStorage.length + (await indexedDB.databases()).length + (await caches.keys()).length);
        if (!datos.pagina_anterior || datos.consultas_api !== 1 || [datos.bloqueadas, datos.red_fallida, datos.errores_js,
          datos.consola_error, datos.http_fallidos, datos.cookies, datos.almacenamiento].some(Boolean)) throw new Error('controles');
        datos.estado = 'COMPROBADO';
      } finally { await context.close(); guardar(); }
    }
    resultado.estado = 'LECTURAS_Y_COMPROBACIONES_PARCIALES';
    return resultado;
  } catch (e) {
    resultado.estado = 'CORTADO';
    resultado.corte = ['http', 'contrato', 'escenario_insuficiente', 'idioma', 'historial', 'accesibilidad', 'controles', 'zoom_nativo', 'teclado_accesibilidad'].includes(e.message) ? e.message : 'navegador';
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
    rutaExterna(c.identidad.certificado, { privado: true }); rutaExterna(c.identidad.clave, { privado: true });
    const { chromium } = await import(pathToFileURL(rutaExterna(process.env.VEC_PLAYWRIGHT_MODULE)).href);
    exigir(typeof chromium?.launchPersistentContext === 'function');
    const salida = prepararSalida(args[3]); iniciado = true;
    await recorrer(c, chromium, salida);
  } catch { console.error(iniciado ? mensajes.fallo : mensajes.entrada); process.exitCode = 1; }
}
if (process.argv[1] && pathToFileURL(path.resolve(process.argv[1])).href === import.meta.url) await main();
