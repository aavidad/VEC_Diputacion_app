#!/usr/bin/env node
import fs from 'node:fs';
import path from 'node:path';
import { pathToFileURL } from 'node:url';
import { cargarConfig, rutaExterna, prepararSalida, solicitudPermitida, huella, idiomas, CUADRO, DETALLE, BORRADORES, FIRMAS } from './config.mjs';
import { chromeAccesible, comprobarAccesibilidad, activarLectura } from './a11y.mjs';
import { validarRespuestaMiBolsa } from '../../web/static/area-personal/contrato.js';
import { validarHistorialMiBolsa, RUTA_HISTORIAL_MI_BOLSA } from '../../web/static/area-personal/mi-bolsa-historial.js';
import { validarRespuestaBolsas, validarRespuestaCandidatosBolsa } from '../../web/static/portal-empleado/portal-bolsas-contrato.js';
import { crearConsultasRRHHClienteHTTP } from '../../web/static/portal-empleado/modulos/contratacion-temporal/cliente-http-consultas-rrhh.js';
import { validarBorradoresDisponibles } from '../../web/static/portal-empleado/modulos/contratacion-temporal/cliente-http-borradores-publicados.js';
import { validarEstadoFirmas } from '../../web/static/portal-empleado/modulos/contratacion-temporal/firma-documento-cliente.js';

const preferido = process.env.LANG?.split(/[_.-]/)[0];
const idioma = Object.hasOwn(idiomas.disponibles, preferido) ? preferido : idiomas.respaldo;
const mensajes = JSON.parse(fs.readFileSync(new URL(idiomas.disponibles[idioma].mensajes, import.meta.url)));

// route.fetch evita que Playwright siga el 302 antes de inspeccionar el origen.
export async function interceptar(route, origen, datos) {
  if (!solicitudPermitida(route.request(), origen)) {
    datos.bloqueadas += 1;
    await route.abort();
    return;
  }
  try {
    const response = await route.fetch({ maxRedirects: 0, maxRetries: 0, timeout: 15000 });
    if (new URL(response.url()).origin !== origen || (response.status() >= 300 && response.status() < 400)
        || (await response.headersArray()).some(h => h.name.toLowerCase() === 'set-cookie')) {
      datos.bloqueadas += 1;
      await route.abort();
      return;
    }
    const ruta = new URL(route.request().url()).pathname;
    if ([BORRADORES, FIRMAS].includes(ruta)) {
      datos.auxiliares ??= [];
      datos.auxiliares.push({ lectura: ruta === BORRADORES ? 'borradores_disponibles' : 'firmas_consulta', http: response.status() });
      if (response.status() === 200) {
        try { validarLectura(ruta, await response.json()); }
        catch { datos.contratos_fallidos = (datos.contratos_fallidos || 0) + 1; }
      }
    }
    await route.fulfill({ response });
  } catch {
    datos.red_fallida += 1;
    await route.abort();
  }
}

// La fábrica publica en ejecutar el validador real usado por el cliente CT.
// Este adaptador sólo obtiene contratos: no crea otro transporte ni envía red.
const contratosCT = crearConsultasRRHHClienteHTTP({ ejecutar: solicitud => solicitud, validarOpciones: () => ({}) });
const validarCuadro = contratosCT.consultarCuadroRRHH({ filtros: { texto: '', estado_clave: '', fase_clave: '' }, paginacion: { limite: 10, cursor: '' } }).validarRespuesta;
const validarDetalle = contratosCT.consultarDetalleRRHH({ expediente_ref: 'expediente:contrato', version_observada: 0 }).validarRespuesta;

export function validarLectura(ruta, envelope) {
  if (ruta === '/api/vec/bolsa/mi-bolsa') return validarRespuestaMiBolsa(envelope);
  if (ruta === RUTA_HISTORIAL_MI_BOLSA) return validarHistorialMiBolsa(envelope, 1);
  if (ruta === '/api/vec/bolsa/bolsas') return validarRespuestaBolsas(envelope);
  if (ruta.startsWith('/api/vec/bolsa/bolsas/') && ruta.endsWith('/candidatos')) return validarRespuestaCandidatosBolsa(envelope);
  if (ruta === CUADRO) return validarCuadro(envelope?.data);
  if (ruta === DETALLE) return validarDetalle(envelope?.data);
  if (ruta === BORRADORES) return validarBorradoresDisponibles(envelope);
  if (ruta === FIRMAS) {
    const estado = validarEstadoFirmas(envelope?.data);
    if (!estado) throw new Error('contrato');
    return estado;
  }
  throw new Error('contrato');
}

export async function respuesta(page, ruta, accion) {
  const pendiente = page.waitForResponse(r => new URL(r.url()).pathname === ruta, { timeout: 20000 });
  // Instalar ambas promesas antes de esperar evita rechazos sin observador.
  const [r] = await Promise.all([pendiente, accion()]);
  if (r.status() !== 200) throw new Error('http');
  try { return validarLectura(ruta, await r.json()); }
  catch { throw new Error('contrato'); }
}

async function controles(page, context, datos) {
  const vista = await page.evaluate(async () => ({
    desbordamiento: document.documentElement.scrollWidth > document.documentElement.clientWidth,
    almacenamiento: localStorage.length + sessionStorage.length + (await indexedDB.databases()).length,
    caches: (await caches.keys()).length,
  }));
  Object.assign(datos, vista, { cookies: (await context.cookies()).length });
  if (Object.values(vista).some(Boolean) || datos.cookies || datos.errores_js || datos.bloqueadas || datos.red_fallida || datos.contratos_fallidos) throw new Error('controles');
}

async function abrir(page, destino) {
  const r = await page.goto(destino, { waitUntil: 'domcontentloaded', timeout: 20000 });
  if (r?.status() !== 200) throw new Error('entrada');
}

async function tecladoMovil(page, nombre) {
  const lateral = nombre === 'mi_bolsa' ? '.ap-lateral' : '.portal-lateral';
  const boton = nombre === 'mi_bolsa' ? '[data-accion="alternar-menu"]' : '#boton-menu';
  const skip = page.locator('.salto-contenido');
  await skip.focus();
  await page.keyboard.press('Tab');
  if (await page.evaluate(selector => Boolean(document.activeElement.closest(selector)), lateral)) throw new Error('teclado');
  await page.locator(boton).focus();
  await page.keyboard.press('Enter');
  await page.waitForFunction(() => document.body.dataset.menuAbierto === 'true');
  if (!(await page.evaluate(selector => Boolean(document.activeElement.closest(selector)), lateral))) throw new Error('teclado');
  await page.keyboard.press('Escape');
  await page.waitForFunction(() => document.body.dataset.menuAbierto !== 'true');
  if (!(await page.locator(boton).evaluate(el => document.activeElement === el))) throw new Error('teclado');
}

export async function recorrer(c, chromium, salida, anterior, accesibilidad = false) {
  const resultado = { version: 1, estado: 'EN_CURSO', servidor_instalado_verificado: false,
    commit_servido_declarado: c.commit_servido, escenario_sha256: huella(JSON.stringify([c.origenes, c.bolsa_ref, c.expediente_ref, c.idioma])),
    pasos: [], reinicio_verificado: false };
  const guardar = () => fs.writeFileSync(path.join(salida, 'resultado.json'), JSON.stringify(resultado, null, 2), { mode: 0o600 });
  guardar();
  let browser, chrome;
  try {
    if (accesibilidad) { chrome = await chromeAccesible(chromium, salida); browser = chrome.browser; }
    else browser = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
    for (const [width, factor] of (accesibilidad ? [[1440, 1], [390, 1], [1440, 2]] : [[1440, 1], [390, 1]])) {
      if (accesibilidad) await chrome.zoom(factor);
      for (const nombre of ['mi_bolsa', 'bolsa_rrhh', 'contratacion']) {
        const identidad = c.identidades[nombre === 'mi_bolsa' ? 'candidato' : 'rrhh'];
        const origen = c.origenes[nombre === 'mi_bolsa' ? 'externo' : 'interno'];
        const datos = { nombre, ancho: width, estado: 'ENTRADA', errores_js: 0, bloqueadas: 0, red_fallida: 0 };
        if (accesibilidad) { datos.zoom_porcentaje = factor * 100; datos.accesibilidad = []; }
        resultado.pasos.push(datos);
        let page, context;
        try {
          context = await browser.newContext({ clientCertificates: [{ origin: origen,
            certPath: identidad.certificado, keyPath: identidad.clave }],
            ignoreHTTPSErrors: false, serviceWorkers: 'block',
            locale: idiomas.disponibles[c.idioma].locale, timezoneId: 'Europe/Madrid',
            viewport: { width, height: 900 } });
          await context.route('**/*', route => interceptar(route, origen, datos));
          await context.routeWebSocket('**/*', async route => { datos.bloqueadas += 1; await route.close(); });
          page = await context.newPage();
          page.on('pageerror', () => { datos.errores_js += 1; });
          page.on('dialog', dialog => dialog.dismiss());
          const verificar = async (lectura, selector = '#espacio-trabajo') => {
            if (!accesibilidad) return;
            await page.waitForLoadState('networkidle', { timeout: 20000 });
            const prueba = await comprobarAccesibilidad(page, selector, lectura, width, factor);
            datos.accesibilidad.push(prueba);
            if (prueba.estado !== 'COMPROBADO') throw new Error('accesibilidad');
          };
          datos.estado = 'LECTURA';
          if (nombre === 'mi_bolsa') {
            const historial = page.waitForResponse(r => new URL(r.url()).pathname === '/api/vec/bolsa/mi-bolsa/historial', { timeout: 20000 });
            const [data, historico] = await Promise.all([
              respuesta(page, '/api/vec/bolsa/mi-bolsa', () => abrir(page, `${origen}/area-personal/?vista=llamamientos&lang=${c.idioma}`)), historial,
            ]);
            if (historico.status() !== 200) throw new Error('historial');
            try { validarLectura(RUTA_HISTORIAL_MI_BOLSA, await historico.json()); }
            catch { throw new Error('historial'); }
            if (!data.participaciones?.some(p => p.bolsa === c.bolsa_ref)) throw new Error('bolsa');
            await page.locator('#historial-mi-bolsa').waitFor({ state: 'visible' });
            await page.waitForFunction(() => {
              const contenedor = document.querySelector('#historial-mi-bolsa');
              return contenedor?.querySelector('.historial-mi-bolsa') && !contenedor.querySelector('p[role="status"]');
            });
            if (await page.locator('#historial-mi-bolsa [data-historial-accion="reintentar"]').count()) throw new Error('historial');
            datos.mi_bolsa_http = 200; datos.historial_http = 200;
            await verificar('mi_bolsa');
            await verificar('historial_mi_bolsa', '#historial-mi-bolsa');
          } else if (nombre === 'bolsa_rrhh') {
            const data = await respuesta(page, '/api/vec/bolsa/bolsas', () => abrir(page, `${origen}/portal-empleado/?lang=${c.idioma}#bolsa/resumen`));
            if (!data.bolsas?.some(b => b.bolsa_ref === c.bolsa_ref)) throw new Error('bolsa');
            const boton = page.locator(`tr[data-bolsa-ref="${c.bolsa_ref}"] button[data-accion="ver-bolsa"]`).first();
            await boton.waitFor({ state: 'visible' });
            datos.bolsas_http = 200;
            await verificar('bolsas_rrhh');
            const candidatos = await respuesta(page, `/api/vec/bolsa/bolsas/${c.bolsa_ref}/candidatos`, () => accesibilidad ? activarLectura(page, boton) : boton.click());
            if (candidatos.bolsa?.bolsa_ref !== c.bolsa_ref || !Array.isArray(candidatos.candidatos)) throw new Error('candidatos');
            await page.locator('[data-bolsa-accion="iniciar-b7"]').waitFor({ state: 'visible' });
            datos.bolsas_http = 200; datos.candidatos_http = 200;
            await verificar('candidatos_bolsa');
          } else {
            const data = await respuesta(page, CUADRO, () => abrir(page, `${origen}/portal-empleado/?lang=${c.idioma}#contratacion-temporal`));
            if (!data.expedientes?.some(e => e.expediente_ref === c.expediente_ref)) throw new Error('expediente_no_visible');
            const boton = page.locator(`[data-ct-exp-abrir="${c.expediente_ref}"]`).first();
            await boton.waitFor({ state: 'visible' });
            datos.cuadro_http = 200;
            await verificar('cuadro_ct', '[data-modulo="contratacion-temporal"]');
            const detalle = await respuesta(page, DETALLE, () => accesibilidad ? activarLectura(page, boton) : boton.click());
            if (detalle.resumen?.expediente_ref !== c.expediente_ref || !Number.isSafeInteger(detalle.resumen.version)
                || detalle.resumen.version < 1 || !Array.isArray(detalle.hitos)) throw new Error('detalle');
            await page.locator('[data-modulo="contratacion-temporal"] .ct-exp-ficha-cabecera').waitFor({ state: 'visible' });
            if (await page.locator('[data-modulo="contratacion-temporal"] .ct-exp-estado-global[role="alert"]').count()) throw new Error('detalle');
            datos.cuadro_http = 200; datos.detalle_http = 200;
            datos.version_expediente = detalle.resumen.version;
            datos.detalle_sha256 = huella(JSON.stringify([detalle.resumen, detalle.hitos]));
            await verificar('detalle_ct', '[data-modulo="contratacion-temporal"]');
          }
          await page.waitForLoadState('networkidle', { timeout: 20000 });
          if (!(await page.locator('html').getAttribute('lang'))?.startsWith(c.idioma)) throw new Error('idioma');
          if (width === 390 || (accesibilidad && factor === 2)) { await tecladoMovil(page, nombre); datos.teclado_menu = true; }
          await controles(page, context, datos);
          datos.estado = 'COMPROBADO';
          if (anterior) {
            const previo = anterior.pasos.find(p => p.nombre === nombre && p.ancho === width);
            if (!previo || previo.estado !== 'COMPROBADO'
                || (nombre === 'contratacion' && previo.detalle_sha256 !== datos.detalle_sha256)) throw new Error('comparacion');
            datos.comparacion = true;
          }
        } catch (e) {
          datos.estado = 'CORTADO';
          // Solo códigos internos controlados; los errores Playwright contienen URL/identidad.
          datos.corte = ['http', 'contrato', 'entrada', 'historial', 'bolsa', 'candidatos', 'expediente_no_visible', 'detalle', 'controles', 'comparacion', 'idioma', 'teclado', 'accesibilidad', 'teclado_accesibilidad', 'zoom_nativo'].includes(e.message) ? e.message : 'navegador';
          resultado.estado = 'CORTADO';
          throw e;
        } finally {
          guardar();
          await context?.close();
        }
      }
    }
    resultado.estado = accesibilidad ? 'LECTURAS_Y_ACCESIBILIDAD_COMPROBADAS' : 'LECTURAS_COMPROBADAS';
    guardar();
    return resultado;
  } catch (e) {
    resultado.estado = 'CORTADO';
    resultado.corte = ['accesibilidad', 'teclado_accesibilidad', 'zoom_nativo'].includes(e.message) ? e.message : 'navegador';
    guardar();
    throw e;
  } finally {
    try { if (chrome) await chrome.cerrar(); else await browser?.close(); }
    catch (e) { resultado.estado = 'CORTADO'; resultado.corte = 'navegador'; guardar(); throw e; }
  }
}

async function main() {
  const args = process.argv.slice(2), opciones = {};
  try {
    for (let i = 0; i < args.length; i += 2) {
      if (!['--config', '--modo', '--salida', '--comparar'].includes(args[i]) || !args[i + 1] || Object.hasOwn(opciones, args[i])) throw new Error();
      opciones[args[i]] = args[i + 1];
    }
    if (!opciones['--config'] || !['preflight', 'lectura', 'accesibilidad'].includes(opciones['--modo'])) throw new Error();
    const c = cargarConfig(opciones['--config']);
    // Ruta de una instalación existente. No descarga paquetes ni navegadores.
    const modulo = rutaExterna(process.env.VEC_PLAYWRIGHT_MODULE);
    const { chromium } = await import(pathToFileURL(modulo).href);
    if (!chromium || typeof chromium.launch !== 'function') throw new Error();
    if (opciones['--modo'] === 'preflight') {
      if (opciones['--salida'] || opciones['--comparar']) throw new Error();
      console.log(mensajes.preflight);
      return 0;
    }
    let anterior;
    if (opciones['--comparar']) {
      if (opciones['--modo'] !== 'lectura') throw new Error();
      anterior = JSON.parse(fs.readFileSync(rutaExterna(opciones['--comparar'], { privado: true }), 'utf8'));
      if (anterior.estado !== 'LECTURAS_COMPROBADAS' || !Array.isArray(anterior.pasos)
          || anterior.commit_servido_declarado !== c.commit_servido
          || anterior.escenario_sha256 !== huella(JSON.stringify([c.origenes, c.bolsa_ref, c.expediente_ref, c.idioma]))) throw new Error();
    }
    const salida = prepararSalida(opciones['--salida']);
    try { await recorrer(c, chromium, salida, anterior, opciones['--modo'] === 'accesibilidad'); }
    catch { console.error(mensajes.fallo); return 1; }
    console.log(opciones['--modo'] === 'accesibilidad' ? idiomas.disponibles[c.idioma].accesibilidad_exito : mensajes.exito);
    return 0;
  } catch { console.error(mensajes.entrada); console.error(idiomas.disponibles[idioma].accesibilidad_uso); return 2; }
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) process.exitCode = await main();
