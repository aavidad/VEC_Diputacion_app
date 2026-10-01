#!/usr/bin/env node
import fs from 'node:fs';
import path from 'node:path';
import { pathToFileURL } from 'node:url';
import { cargarConfig, prepararEstado, guardarEstado, validarEstadoAnterior, huellaEscenario,
  validarReciboFirma, rutaPrivada, sha256 } from './config.mjs';
import { validarConsultaB2, validarReciboB2, validarSolicitudPlanB2, validarSolicitudConfirmacionB2 } from '../../web/static/portal-empleado/modulos/contratacion-temporal/contrato-incorporacion-personal-b2.js';
import { validarEstadoFirmas, RUTA_CONSULTA_FIRMA_DOCUMENTO, RUTA_FIRMA_DOCUMENTO } from '../../web/static/portal-empleado/modulos/contratacion-temporal/firma-documento-cliente.js';
import { RUTA_PLAN_B2, RUTA_CONFIRMAR_B2 } from '../../web/static/portal-empleado/modulos/contratacion-temporal/cliente-http-incorporacion-personal-b2.js';
import { validarCircuitoFirma, RUTA_CIRCUITO_FIRMA } from '../../web/static/portal-empleado/modulos/contratacion-temporal/circuito-firma.js';

const CUADRO = '/api/vec/contratacion-temporal/cuadro/consultas';
const DETALLE = '/api/vec/contratacion-temporal/expedientes/consultas';
const MAX_PAGINAS = 100;
const RELOJ = { navegacion: 20000, lectura: 20000, autofirma: 180000 };
const fallo = codigo => { throw new Error(codigo); };
const permitidos = new Set(['entrada', 'red', 'http', 'contrato', 'expediente', 'b2_no_disponible',
  'b2_opcion', 'b2_pendiente', 'firma_no_disponible', 'firma_pendiente', 'firma_incompleta',
  'recuperacion', 'controles', 'estado_incierto', 'tiempo_agotado']);

export function seleccionarOpciones(consulta, seleccion) {
  const o = consulta.opciones;
  const vacante = x => [x.plaza_ref, x.puesto_ref, x.version_plantilla_ref, x.version_rpt_ref];
  const catalogo = x => [x.ref, x.version];
  const grupos = {
    vacante: [o.vacantes, vacante, vacante(seleccion.vacante)],
    regimen: [o.regimenes, catalogo, catalogo(seleccion.regimen)],
    modalidad: [o.modalidades, catalogo, catalogo(seleccion.modalidad)],
    clase_ocupacion: [o.clases_ocupacion, x => x.valor, seleccion.clase_ocupacion],
    motivo: [o.motivos, x => x, seleccion.motivo],
    documento: [o.documentos, x => [x.documento_ref, x.documento_sha256],
      [seleccion.documento.documento_ref, seleccion.documento.documento_sha256]],
  };
  const valores = {};
  for (const [campo, [lista, identidad, elegida]] of Object.entries(grupos)) {
    const candidatos = lista.map((x, i) => JSON.stringify(identidad(x)) === JSON.stringify(elegida) ? i : -1).filter(i => i >= 0);
    if (candidatos.length !== 1) fallo('b2_opcion');
    valores[campo] = String(candidatos[0]);
  }
  if (o.periodo.fuente_ref && (o.periodo.desde !== seleccion.desde || o.periodo.hasta !== seleccion.hasta)) fallo('b2_opcion');
  return { ...valores, desde: seleccion.desde, hasta: seleccion.hasta };
}

export function resumirRecuperacionFirmas(estado, firmas, inventarioAnterior = null) {
  if (!estado || !Array.isArray(estado.documentos) || !estado.documentos.length
    || estado.documentos.some(d => !d.completo || d.pasos.some(p => p.estado !== 'firmado' || !p.recibo_ref || !p.registrada_en))) fallo('firma_incompleta');
  const inventario = estado.documentos.flatMap(d => d.pasos.map(p => ({ documento: d.documento, orden: p.orden,
    recibo_ref: p.recibo_ref, registrada_en: p.registrada_en, custodiado: p.documento_custodiado ?? null })));
  const claves = inventario.map(p => `${p.documento}:${p.orden}`);
  if (new Set(claves).size !== inventario.length || new Set(inventario.map(p => p.recibo_ref)).size !== inventario.length) fallo('recuperacion');
  inventario.sort((a, b) => a.documento.localeCompare(b.documento) || a.orden - b.orden);
  if (inventarioAnterior && JSON.stringify(inventario) !== JSON.stringify(inventarioAnterior)) fallo('recuperacion');
  const vistos = new Set();
  for (const firma of firmas) {
    const pasos = estado.documentos.filter(d => d.documento === firma.documento)
      .flatMap(d => d.pasos.filter(p => p.orden === firma.orden && p.recibo_ref === firma.recibo_ref));
    if (pasos.length !== 1 || pasos[0].estado !== 'firmado' || pasos[0].registrada_en !== firma.registrada_en
      || vistos.has(firma.recibo_ref)) fallo('recuperacion');
    vistos.add(firma.recibo_ref);
    if (firma.custodiado && JSON.stringify(pasos[0].documento_custodiado) !== JSON.stringify(firma.custodiado)) fallo('recuperacion');
  }
  return { documentos: estado.documentos.length, pasos: inventario.length, recibos_cotejados: inventario.length,
    recibos_del_post_cotejados: vistos.size, inventario };
}

function argumentos(args) {
  const opciones = {};
  for (let i = 0; i < args.length; i += 2) {
    if (!['--config', '--modo', '--estado', '--reinicio'].includes(args[i]) || !args[i + 1] || opciones[args[i]]) fallo('entrada');
    opciones[args[i]] = args[i + 1];
  }
  if (!opciones['--config'] || !opciones['--estado'] || !['preparado', 'firmar', 'incorporar', 'reconciliar', 'recuperar'].includes(opciones['--modo'])) fallo('entrada');
  if (opciones['--modo'] === 'recuperar' ? !opciones['--reinicio'] : Boolean(opciones['--reinicio'])) fallo('entrada');
  return opciones;
}

export function esRespuesta(response, c, ruta, metodo = 'POST') {
  const u = new URL(response.url());
  return u.origin === c.origen && u.pathname === ruta && response.request().method() === metodo;
}

async function respuesta(response, c, ruta, metodo = 'POST') {
  if (!esRespuesta(response, c, ruta, metodo) || response.status() !== 200) fallo('http');
  const body = await response.json().catch(() => fallo('contrato'));
  if (!body || Object.keys(body).length !== 1 || !body.data) fallo('contrato');
  return body.data;
}

export async function consultarFirmas(page, c) {
  const r = await page.evaluate(async ({ ruta, ref }) => {
    const x = await fetch(ruta, { method: 'POST', headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
      body: JSON.stringify({ expediente_ref: ref }), credentials: 'same-origin', cache: 'no-store', redirect: 'error', referrerPolicy: 'no-referrer' });
    return { status: x.status, body: await x.json() };
  }, { ruta: RUTA_CONSULTA_FIRMA_DOCUMENTO, ref: c.expediente_ref });
  if (r.status !== 200 || !r.body || Object.keys(r.body).length !== 1) fallo('firma_no_disponible');
  const estado = validarEstadoFirmas(r.body.data);
  // `ejemplo=true` describe el catálogo de desarrollo, no una respuesta fabricada.
  if (!estado || !estado.verificacion_disponible) fallo('firma_no_disponible');
  return estado;
}

async function consultarCatalogo(page) {
  const r = await page.evaluate(async ruta => {
    const x = await fetch(ruta, { method: 'GET', credentials: 'same-origin', cache: 'no-store',
      redirect: 'error', referrerPolicy: 'no-referrer' });
    return { status: x.status, body: await x.json() };
  }, RUTA_CIRCUITO_FIRMA);
  if (r.status !== 200 || !r.body || Object.keys(r.body).length !== 1) fallo('firma_no_disponible');
  const catalogo = validarCircuitoFirma(r.body.data);
  if (!catalogo) fallo('firma_no_disponible');
  return catalogo;
}

async function consultarB2(page, c) {
  const r = await page.evaluate(async ({ ruta, ref }) => {
    const x = await fetch(`${ruta}?expediente_ref=${encodeURIComponent(ref)}`, {
      method: 'GET', credentials: 'same-origin', cache: 'no-store', redirect: 'error', referrerPolicy: 'no-referrer' });
    return { status: x.status, body: await x.json() };
  }, { ruta: RUTA_PLAN_B2, ref: c.expediente_ref });
  if (r.status !== 200 || !r.body || Object.keys(r.body).length !== 1) fallo('b2_no_disponible');
  return validarConsultaB2(r.body.data, c.expediente_ref);
}

const EFECTOS = new Set([RUTA_FIRMA_DOCUMENTO, RUTA_PLAN_B2, RUTA_CONFIRMAR_B2]);

function persistirAntesDeEnviar(req, c, estado, guardar) {
  const url = new URL(req.url());
  if (url.origin !== c.origen || req.method() !== 'POST' || !EFECTOS.has(url.pathname)) return;
  const body = req.postDataJSON();
  if (!body || body.expediente_ref !== c.expediente_ref || typeof body.clave_idempotencia !== 'string') fallo('entrada');
  if (url.pathname === RUTA_PLAN_B2) {
    validarSolicitudPlanB2(body);
    estado.b2.intencion = body;
  } else if (url.pathname === RUTA_CONFIRMAR_B2) validarSolicitudConfirmacionB2(body);
  else if (body.resultado !== 'firmado' || !/^[A-Za-z0-9][A-Za-z0-9._-]{15,63}$/u.test(body.clave_idempotencia)
    || !Number.isSafeInteger(body.paso_orden) || body.paso_orden < 1
    || typeof body.documento !== 'string') fallo('entrada');
  estado.pendiente = { ruta: url.pathname, clave_idempotencia: body.clave_idempotencia,
    ...(url.pathname === RUTA_FIRMA_DOCUMENTO ? { documento: body.documento, paso_orden: body.paso_orden } :
      { plan_ref: body.plan_ref ?? null, version_plan: body.version_plan ?? null }) };
  guardar();
}

export async function interceptarHTTP(route, c, estado, guardar) {
    const req = route.request(), url = new URL(req.url());
    if (![c.origen, ...c.auxiliares.filter(v => v.startsWith('https:'))].includes(url.origin)
      || url.username || url.password || !['GET', 'POST'].includes(req.method())) {
      estado.red_bloqueada++; await route.abort(); return;
    }
    try {
      persistirAntesDeEnviar(req, c, estado, guardar);
      const res = await route.fetch({ maxRedirects: 0, maxRetries: 0, timeout: RELOJ.lectura });
      const headers = await res.headersArray();
      if (res.status() >= 300 && res.status() < 400 || new URL(res.url()).origin !== url.origin
        || headers.some(h => h.name.toLowerCase() === 'set-cookie')) {
        estado.red_bloqueada++; await route.abort(); return;
      }
      await route.fulfill({ response: res });
    } catch { estado.red_bloqueada++; await route.abort(); }
}

async function vigilarRed(context, c, estado, guardar) {
  await context.route('**/*', route => interceptarHTTP(route, c, estado, guardar));
  await context.routeWebSocket('**/*', async route => {
    if (c.auxiliares.includes(new URL(route.url()).origin)) await route.connect();
    else { estado.red_bloqueada++; await route.close(); }
  });
}

async function controles(page, context, estado) {
  const datos = await page.evaluate(async () => ({ desbordamiento: document.documentElement.scrollWidth > document.documentElement.clientWidth,
    almacenamiento: localStorage.length + sessionStorage.length + (await indexedDB.databases()).length + (await caches.keys()).length }));
  if (datos.desbordamiento || datos.almacenamiento || (await context.cookies()).length || estado.errores_js || estado.red_bloqueada) fallo('controles');
  estado.controles = { ancho: page.viewportSize().width, sin_desbordamiento: true, sin_almacenamiento: true,
    sin_cookies: true, sin_errores_js: true, red_mismo_origen: true };
}

async function captura(page, c, modo, estado) {
  if (!c.capturas) return;
  const ancho = page.viewportSize().width;
  const destino = path.join(c.capturas, `${modo}-${ancho}.png`);
  rutaPrivada(destino, { nueva: true });
  const bytes = await page.screenshot({ fullPage: true, animations: 'disabled', timeout: RELOJ.lectura });
  fs.writeFileSync(destino, bytes, { mode: 0o600, flag: 'wx' });
  estado.capturas ??= [];
  estado.capturas.push({ modo, ancho, bytes: bytes.length, sha256: sha256(bytes) });
}

async function abrirExpediente(page, c) {
  const ingreso = await page.goto(`${c.origen}/portal-empleado/?lang=es#contratacion-temporal`, { waitUntil: 'domcontentloaded', timeout: RELOJ.navegacion });
  if (ingreso?.status() !== 200) fallo('http');
  for (let pagina = 0; pagina < MAX_PAGINAS; pagina++) {
    // La referencia se compara como atributo, sin interpolarla en CSS.
    await page.locator('[data-ct-exp-abrir]').first().waitFor({ state: 'visible', timeout: RELOJ.lectura });
    const botones = page.locator('[data-ct-exp-abrir]');
    for (let i = 0; i < await botones.count(); i++) {
      const boton = botones.nth(i);
      if (await boton.getAttribute('data-ct-exp-abrir') === c.expediente_ref) {
        const espera = page.waitForResponse(r => esRespuesta(r, c, DETALLE), { timeout: RELOJ.lectura });
        await boton.click();
        const detalle = await respuesta(await espera, c, DETALLE);
        if (detalle.resumen?.expediente_ref !== c.expediente_ref) fallo('contrato');
        await page.locator('[data-ct-exp-incorporacion-ejercicio]').waitFor({ state: 'visible', timeout: RELOJ.lectura });
        return;
      }
    }
    const siguiente = page.locator('[data-ct-exp-pagina="siguiente"]');
    if (!await siguiente.count() || !await siguiente.isEnabled()) fallo('expediente');
    const espera = page.waitForResponse(r => esRespuesta(r, c, CUADRO), { timeout: RELOJ.lectura });
    await siguiente.click();
    await respuesta(await espera, c, CUADRO);
  }
  fallo('expediente');
}

async function firmarTodo(page, c, estado, guardar) {
  let firmas = await consultarFirmas(page, c);
  const catalogo = await consultarCatalogo(page);
  if (catalogo.huella_sha256 !== firmas.huella_sha256) fallo('firma_no_disponible');
  const pendientes = firmas.documentos.reduce((n, d) => n + d.pasos.filter(p => p.estado !== 'firmado').length, 0);
  if (pendientes > 100 || firmas.documentos.length < 1) fallo('firma_incompleta');
  for (let i = 0; i < pendientes; i++) {
    const d = firmas.documentos.find(x => x.paso_pendiente > 0);
    if (!d) break;
    const orden = d.paso_pendiente;
    const paso = catalogo.documentos.find(x => x.documento === d.documento)?.pasos.find(p => p.orden === orden);
    if (!paso) fallo('firma_no_disponible');
    if (paso.perfil_ref !== c.identidad.perfil_ref) {
      estado.siguiente_perfil_ref = paso.perfil_ref;
      estado.estado = 'E3_PENDIENTE_OTRA_IDENTIDAD'; guardar(); return;
    }
    delete estado.siguiente_perfil_ref;
    const detalles = page.locator('[data-ct-firma-detalles]');
    await detalles.waitFor({ state: 'visible', timeout: RELOJ.lectura });
    await detalles.evaluate(e => { e.open = true; });
    const botones = page.locator('[data-ct-firma-accion="firmar"]');
    let boton = null;
    for (let j = 0; j < await botones.count(); j++) {
      const b = botones.nth(j);
      if (await b.getAttribute('data-ct-firma-documento') === d.documento
        && Number(await b.getAttribute('data-ct-firma-orden')) === orden) { boton = b; break; }
    }
    if (!boton || !await boton.isEnabled()) fallo('firma_pendiente');
    const espera = page.waitForResponse(r => esRespuesta(r, c, RUTA_FIRMA_DOCUMENTO), { timeout: RELOJ.autofirma })
      .catch(() => fallo('tiempo_agotado'));
    await boton.click();
    const res = await espera;
    if (![200, 201].includes(res.status())) fallo('http');
    const body = await res.json().catch(() => fallo('contrato'));
    const firma = validarReciboFirma(body?.data, c.expediente_ref, d.documento, orden, c.identidad);
    estado.firmas.push(firma); estado.pendiente = null; guardar();
    firmas = await consultarFirmas(page, c);
    const actual = firmas.documentos.find(x => x.documento === d.documento)?.pasos.find(p => p.orden === orden);
    if (actual?.estado !== 'firmado' || actual.recibo_ref !== firma.recibo_ref || actual.registrada_en !== firma.registrada_en) fallo('recuperacion');
    await page.waitForFunction(({ documento, paso }) => ![...document.querySelectorAll('[data-ct-firma-accion="firmar"]')]
      .some(b => b.dataset.ctFirmaDocumento === documento && Number(b.dataset.ctFirmaOrden) === paso),
    { documento: d.documento, paso: orden }, { timeout: RELOJ.lectura });
  }
  estado.firma = resumirRecuperacionFirmas(firmas, estado.firmas);
  estado.estado = 'E3_COMPROBADA'; guardar();
}

async function incorporar(page, c, estado, guardar) {
  const consulta = await consultarB2(page, c);
  if (consulta.estado === 'incorporacion_confirmada') {
    if (!estado.b2.recibo || JSON.stringify(estado.b2.recibo) !== JSON.stringify(consulta.recibo)) fallo('estado_incierto');
    estado.b2.recibo = consulta.recibo; estado.b2.plan = consulta.plan?.plan_ref ?? null;
    estado.estado = 'B2_COMPROBADA'; guardar(); return;
  }
  if (consulta.prerrequisitos.length < 1 || consulta.prerrequisitos.some(p => !p.cumplido)) fallo('b2_pendiente');
  if (consulta.estado === 'sin_plan') {
    const valores = seleccionarOpciones(consulta, c.seleccion);
    const formulario = page.locator('[data-b2-form]');
    await formulario.waitFor({ state: 'visible', timeout: RELOJ.lectura });
    for (const [campo, valor] of Object.entries(valores)) {
      const control = formulario.locator(`[name="${campo}"]`);
      if (!await control.count()) continue; // Única opción o fecha fijada por fuente.
      const tag = await control.evaluate(e => e.tagName);
      if (tag === 'SELECT') await control.selectOption(valor);
      else if (await control.getAttribute('readonly') === null) await control.fill(valor);
    }
    await formulario.locator('[type="submit"]').click();
    await page.locator('[data-b2-accion="registrar"]').waitFor({ state: 'visible', timeout: RELOJ.lectura });
  } else if (consulta.plan) {
    if (JSON.stringify(consulta.plan.intencion) !== JSON.stringify(estado.b2.intencion)) fallo('estado_incierto');
  }
  const esperaPlan = consulta.plan ? null : page.waitForResponse(r => esRespuesta(r, c, RUTA_PLAN_B2), { timeout: RELOJ.lectura }).catch(() => fallo('tiempo_agotado'));
  const esperaConfirmacion = page.waitForResponse(r => esRespuesta(r, c, RUTA_CONFIRMAR_B2), { timeout: RELOJ.lectura }).catch(() => fallo('tiempo_agotado'));
  await page.locator('[data-b2-accion="registrar"]').click();
  if (esperaPlan) {
    const plan = await respuesta(await esperaPlan, c, RUTA_PLAN_B2);
    const validado = validarConsultaB2(plan, c.expediente_ref);
    if (!validado.plan) fallo('contrato');
    estado.b2.plan = validado.plan.plan_ref; estado.b2.intencion = validado.plan.intencion; guardar();
  }
  const confirmacion = await esperaConfirmacion;
  if (![200, 201].includes(confirmacion.status())) fallo('http');
  const body = await confirmacion.json().catch(() => fallo('contrato'));
  const recibo = validarReciboB2(body?.data, { expediente_ref: c.expediente_ref,
    plan_ref: estado.b2.plan, version_plan: consulta.plan?.version ?? body?.data?.plan_version });
  estado.b2.recibo = recibo; estado.pendiente = null; estado.estado = 'B2_COMPROBADA'; guardar();
}

async function reconciliar(page, c, estado, guardar) {
  const pendiente = estado.pendiente;
  if (!pendiente) fallo('entrada');
  if (pendiente.ruta === RUTA_FIRMA_DOCUMENTO) {
    const firmas = await consultarFirmas(page, c);
    const paso = firmas.documentos.find(d => d.documento === pendiente.documento)
      ?.pasos.find(p => p.orden === pendiente.paso_orden);
    if (paso?.estado !== 'firmado' || !paso.recibo_ref || !paso.registrada_en) fallo('estado_incierto');
    aplicarReconciliacionFirma(estado, firmas, paso); guardar(); return;
  }
  const b2 = await consultarB2(page, c);
  if (!b2.plan || JSON.stringify(b2.plan.intencion) !== JSON.stringify(estado.b2.intencion)
    || b2.plan.intencion.clave_idempotencia !== pendiente.clave_idempotencia) fallo('estado_incierto');
  estado.b2.plan = b2.plan.plan_ref;
  if (pendiente.ruta === RUTA_CONFIRMAR_B2) {
    if (b2.estado !== 'incorporacion_confirmada' || !b2.recibo) fallo('estado_incierto');
    estado.b2.recibo = b2.recibo; estado.estado = 'B2_COMPROBADA';
  } else if (pendiente.ruta === RUTA_PLAN_B2) estado.estado = 'PLAN_B2_RECUPERADO';
  else fallo('entrada');
  estado.pendiente = null; guardar();
}

export function aplicarReconciliacionFirma(estado, consulta, paso) {
  const pendiente = estado.pendiente;
  if (!pendiente || pendiente.ruta !== RUTA_FIRMA_DOCUMENTO || paso?.estado !== 'firmado'
    || !paso.recibo_ref || !paso.registrada_en || paso.orden !== pendiente.paso_orden) fallo('estado_incierto');
  // E3 no devuelve la clave idempotente en GET; la consulta no prueba que
  // este recibo pertenezca al POST incierto. Conservamos la barrera de escritura.
  estado.reconciliacion = { documento: pendiente.documento, orden: pendiente.paso_orden,
    recibo_ref_observado: paso.recibo_ref, registrada_en: paso.registrada_en,
    clave_comprobada: false };
  if (consulta.documentos.every(d => d.completo)) {
    estado.reconciliacion.inventario = resumirRecuperacionFirmas(consulta, estado.firmas).inventario;
  }
  estado.estado = 'FIRMA_CONSULTADA_SIN_CLAVE';
  return estado;
}

async function ejecutar(c, modo, estado, guardar) {
  const modulo = rutaPrivada(process.env.VEC_PLAYWRIGHT_MODULE);
  const { chromium } = await import(pathToFileURL(modulo).href);
  if (typeof chromium?.launch !== 'function') fallo('entrada');
  const browser = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: modo !== 'firmar' });
  let context;
  try {
    context = await browser.newContext({ clientCertificates: [{ origin: c.origen,
      certPath: c.identidad.certificado, keyPath: c.identidad.clave }], ignoreHTTPSErrors: false,
      serviceWorkers: 'block', locale: 'es-ES', timezoneId: 'Europe/Madrid', viewport: { width: 1440, height: 900 } });
    await vigilarRed(context, c, estado, guardar);
    const page = await context.newPage();
    page.on('pageerror', () => { estado.errores_js++; guardar(); });
    page.on('dialog', dialog => dialog.dismiss());
    await abrirExpediente(page, c);
    if (modo === 'firmar') await firmarTodo(page, c, estado, guardar);
    if (modo === 'incorporar') await incorporar(page, c, estado, guardar);
    if (modo === 'reconciliar') await reconciliar(page, c, estado, guardar);
    if (modo === 'recuperar') {
      const firmas = await consultarFirmas(page, c);
      if (!Array.isArray(estado.firma.inventario) || !estado.firma.inventario.length) fallo('recuperacion');
      estado.recuperacion_firma = resumirRecuperacionFirmas(firmas, estado.firmas, estado.firma.inventario);
      const b2 = await consultarB2(page, c);
      if (b2.estado !== 'incorporacion_confirmada' || !estado.b2.recibo
        || JSON.stringify(b2.recibo) !== JSON.stringify(estado.b2.recibo)) fallo('recuperacion');
      estado.recuperacion_b2 = { recibo_ref: b2.recibo.recibo_ref, registrada_en: b2.recibo.registrada_en };
      estado.estado = 'LECTURAS_RECUPERADAS'; guardar();
    }
    await controles(page, context, estado);
    await captura(page, c, modo, estado);
    await page.setViewportSize({ width: 390, height: 844 });
    await controles(page, context, estado);
    await captura(page, c, modo, estado);
    guardar();
  } finally { await context?.close(); await browser.close(); }
}

async function main() {
  let estado, archivo;
  try {
    const a = argumentos(process.argv.slice(2));
    const c = cargarConfig(a['--config']); archivo = a['--estado'];
    const existe = fs.existsSync(archivo);
    if (a['--modo'] === 'preparado') {
      prepararEstado(archivo, existe);
      console.log('PREPARADO: configuración sintética comprobada; navegador sin ejecutar.'); return 0;
    }
    estado = existe ? validarEstadoAnterior(prepararEstado(archivo, true), c)
      : { version: 1, escenario_sha256: huellaEscenario(c), commit_servido_declarado: c.commit_servido,
        servidor_instalado_verificado: false, estado: 'NO_EJECUTADO', firmas: [], b2: {},
        pendiente: null, errores_js: 0, red_bloqueada: 0 };
    if (!existe) prepararEstado(archivo);
    const guardar = () => guardarEstado(archivo, estado);
    if ((estado.pendiente && a['--modo'] !== 'reconciliar') || (!estado.pendiente && a['--modo'] === 'reconciliar')
      || estado.red_bloqueada || estado.errores_js) fallo('estado_incierto');
    if (a['--modo'] === 'recuperar') {
      if (!estado.b2.recibo || !estado.firma || !['B2_COMPROBADA', 'CORTE', 'LECTURAS_RECUPERADAS'].includes(estado.estado)) fallo('recuperacion');
      const reinicio = JSON.parse(fs.readFileSync(rutaPrivada(a['--reinicio']), 'utf8'));
      if (reinicio?.expediente_ref !== c.expediente_ref || reinicio.aplicacion_reiniciada !== true
        || reinicio.postgresql_reiniciado !== true || !reinicio.instante_utc || !Number.isFinite(Date.parse(reinicio.instante_utc))) fallo('entrada');
      estado.reinicio_declarado_sha256 = sha256(JSON.stringify(reinicio)); guardar();
    } else if (a['--modo'] === 'firmar' && estado.firma || a['--modo'] === 'incorporar' && !estado.firma) fallo('entrada');
    delete estado.corte; estado.estado = 'EN_CURSO'; guardar();
    await ejecutar(c, a['--modo'], estado, guardar);
    console.log(estado.estado); return 0;
  } catch (e) {
    const codigo = permitidos.has(e?.message) ? e.message : 'entrada';
    if (estado && archivo) { estado.estado = 'CORTE'; estado.corte = codigo; guardarEstado(archivo, estado); }
    console.error(`CORTE: ${codigo}`); return 1;
  }
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) process.exitCode = await main();
