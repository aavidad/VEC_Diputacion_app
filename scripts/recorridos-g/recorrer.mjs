#!/usr/bin/env node
import fs from 'node:fs';
import path from 'node:path';
import { pathToFileURL } from 'node:url';
import { rutaExterna, prepararSalida, validarOrigen, huella } from '../recorridos-f/config.mjs';

export const catalogo = JSON.parse(fs.readFileSync(new URL('casos.json', import.meta.url), 'utf8'));
const idiomas = JSON.parse(fs.readFileSync(new URL('idiomas.json', import.meta.url), 'utf8'));
const mensajes = Object.fromEntries(Object.entries(idiomas.disponibles).map(([clave, datos]) =>
  [clave, JSON.parse(fs.readFileSync(new URL(datos.mensajes, import.meta.url), 'utf8'))]));
const COMISIONES = '/api/vec/dietas/comisiones';
const CIRCUITO = `${COMISIONES}/circuito`;
const referencia = '[A-Za-z0-9][A-Za-z0-9:._-]{2,159}';
const patrones = Object.freeze({
  empleado_solicitud: [{ metodo: 'POST', ruta: new RegExp(`^${COMISIONES}$`) }],
  empleado_gastos_km: [{ metodo: 'PUT', ruta: new RegExp(`^${COMISIONES}/${referencia}$`) }],
  empleado_calculo_envio: [{ metodo: 'POST', ruta: new RegExp(`^${COMISIONES}/${referencia}/enviar$`) }],
  revision: [{ metodo: 'POST', ruta: new RegExp(`^${CIRCUITO}/${referencia}/decisiones$`) }],
  autorizacion: [{ metodo: 'POST', ruta: new RegExp(`^${CIRCUITO}/${referencia}/decisiones$`) }],
  liquidacion: [{ metodo: 'POST', ruta: new RegExp(`^${CIRCUITO}/${referencia}/decisiones$`) }],
  fiscalizacion: [{ metodo: 'POST', ruta: new RegExp(`^${CIRCUITO}/${referencia}/decisiones$`) }],
  retorno_reenvio: [
    { metodo: 'PUT', ruta: new RegExp(`^${COMISIONES}/${referencia}$`) },
    { metodo: 'POST', ruta: new RegExp(`^${COMISIONES}/${referencia}/enviar$`) },
  ],
});
const estados = Object.freeze({
  empleado_solicitud: ['borrador'], empleado_gastos_km: ['borrador'],
  empleado_calculo_envio: ['enviado_pendiente_revision'],
  revision: ['pendiente_autorizacion'],
  autorizacion: ['pendiente_liquidacion'],
  liquidacion: ['pendiente_fiscalizacion'],
  fiscalizacion: ['fiscalizada'],
  retorno_reenvio: ['borrador', 'enviado_pendiente_revision'],
});
const etapasDecision = Object.freeze({ revision: 'revision', autorizacion: 'autorizacion', liquidacion: 'liquidacion', fiscalizacion: 'fiscalizacion' });

function exigir(ok) { if (!ok) throw new Error('entrada'); }
export function plan() {
  return catalogo.escenarios.map(c => ({ id: c.id, actor: c.actor, tipo: c.tipo, dependencias: c.dependencias, descripcion: c.descripcion, estado: c.tipo === 'pendiente' ? 'CONTRATO_PENDIENTE' : 'PREPARADO_NO_EJECUTADO' }));
}

export function validarCaso(casoId) {
  const caso = catalogo.escenarios.find(c => c.id === casoId);
  exigir(caso && caso.tipo === 'escritura' && patrones[casoId]);
  return caso;
}

export function validarConfiguracion(c, casoId) {
  const caso = validarCaso(casoId);
  exigir(c && c.version === 1 && c.sintetico === true && c.entorno_controlado === true);
  exigir(typeof c.commit_servido === 'string' && /^[0-9a-f]{40}$/.test(c.commit_servido));
  exigir(Object.hasOwn(idiomas.disponibles, c.idioma));
  c.origen = validarOrigen(c.origen);
  exigir(caso.dependencias.every(d => c.dependencias?.[d] === true));
  const pasos = validarPasos(casoId, c.casos?.[casoId]?.pasos);
  const identidad = c.identidades?.[caso.actor];
  exigir(identidad && typeof identidad.certificado === 'string' && typeof identidad.clave === 'string');
  rutaExterna(identidad.certificado); rutaExterna(identidad.clave, { privado: true });
  exigir(fs.statSync('/usr/bin/google-chrome').isFile());
  fs.accessSync('/usr/bin/google-chrome', fs.constants.X_OK);
  return { caso, identidad, pasos };
}

export function validarPasos(casoId, pasos) {
  validarCaso(casoId);
  exigir(Array.isArray(pasos) && pasos.length >= 2 && pasos.length <= 60);
  exigir(pasos.every(p => validarPaso(p)));
  const efectos = pasos.filter(p => p.tipo === 'efecto');
  if (casoId === 'empleado_gastos_km') exigir(pasos.some(p => p.tipo === 'ruta'));
  if (casoId === 'empleado_calculo_envio') exigir(pasos.some(p => p.tipo === 'visible' && p.selector === '[data-dietas-borrador-preparacion]'));
  exigir(efectos.length === patrones[casoId].length);
  exigir(efectos.every((p, i) => p.metodo === patrones[casoId][i].metodo && patrones[casoId][i].ruta.test(p.ruta)
    && (casoId === 'retorno_reenvio' ? p.estado_esperado === estados[casoId][i] : p.estado_esperado === estados[casoId][0])
    && (etapasDecision[casoId] ? p.etapa === etapasDecision[casoId] && p.decision === 'aprobar' : p.etapa === undefined && p.decision === undefined)));
  return pasos;
}

function validarPaso(p) {
  if (!p || typeof p !== 'object' || Array.isArray(p)) return false;
  if (p.tipo === 'visible' || p.tipo === 'click') return selectorValido(p.selector);
  if (p.tipo === 'ruta') return selectorValido(p.selector);
  if (p.tipo === 'fill' || p.tipo === 'select') return selectorValido(p.selector) && typeof p.valor === 'string' && p.valor.length > 0 && p.valor.length <= 500;
  if (p.tipo === 'efecto') return selectorValido(p.selector) && ['POST', 'PUT'].includes(p.metodo) && typeof p.ruta === 'string' && !p.ruta.includes('?')
    && (p.modo === undefined || ['nuevo', 'recuperacion'].includes(p.modo));
  return false;
}
function selectorValido(s) { return typeof s === 'string' && s.length > 0 && s.length <= 160 && (/^\[data-dietas-[a-z0-9-]+(?:="[A-Za-z0-9:_-]+")?\]$/.test(s) || /^\[name="[a-z_]+"\]$/.test(s)); }

export function solicitudPermitida(request, origen, casoId, efectoEsperado) {
  try {
    const u = new URL(request.url());
    if (u.origin !== origen || u.username || u.password) return false;
    const metodo = request.method();
    if (metodo === 'GET') return true;
    if (metodo === 'POST' && u.pathname === '/api/vec/dietas/road-route' && !u.search
        && ['empleado_solicitud', 'empleado_gastos_km'].includes(casoId)) return true;
    if (u.search || !efectoEsperado || efectoEsperado.consumida === true || efectoEsperado.metodo !== metodo || efectoEsperado.ruta !== u.pathname
        || !patrones[casoId]?.some(p => p.metodo === metodo && p.ruta.test(u.pathname))) return false;
    const body = request.postData();
    if (typeof body !== 'string' || Buffer.byteLength(body) > 65536) return false;
    const datos = JSON.parse(body);
    if (!datos || typeof datos !== 'object' || Array.isArray(datos)
        || typeof datos.clave_idempotencia !== 'string' || !/^[A-Za-z0-9:_-]{16,128}$/.test(datos.clave_idempotencia)) return false;
    if (etapasDecision[casoId]) return Object.keys(datos).sort().join(',') === 'clave_idempotencia,decision,etapa,motivo,version_esperada'
      && datos.etapa === efectoEsperado.etapa && datos.decision === efectoEsperado.decision
      && Number.isSafeInteger(datos.version_esperada) && datos.version_esperada >= 1
      && typeof datos.motivo === 'string' && datos.motivo.length <= 600;
    if (casoId !== 'empleado_solicitud' && (!Number.isSafeInteger(datos.version_esperada) || datos.version_esperada < 1)) return false;
    return true;
  } catch { return false; }
}

export async function interceptar(route, origen, casoId, datos) {
  const request = route.request();
  if (!solicitudPermitida(request, origen, casoId, datos.efectoEsperado)) { datos.bloqueadas++; await route.abort(); return; }
  if (datos.efectoEsperado && request.method() !== 'GET' && new URL(request.url()).pathname !== '/api/vec/dietas/road-route') {
    datos.efectoEsperado.consumida = true;
    datos.mutaciones_enviadas = (datos.mutaciones_enviadas || 0) + 1;
  }
  try {
    const response = await route.fetch({ maxRedirects: 0, maxRetries: 0, timeout: 20000 });
    if (new URL(response.url()).origin !== origen || (response.status() >= 300 && response.status() < 400)
        || (await response.headersArray()).some(h => h.name.toLowerCase() === 'set-cookie')) {
      datos.bloqueadas++; await route.abort(); return;
    }
    if (new URL(request.url()).pathname.startsWith('/api/') && response.status() >= 400) datos.http_fallidos++;
    await route.fulfill({ response });
  } catch { datos.red_fallida++; await route.abort(); }
}

export function validarRecibo(body, estadoEsperado, modo = 'nuevo', http) {
  const recibo = body?.recibo;
  const comision = body?.comision;
  exigir(recibo && typeof recibo.referencia === 'string' && /^rcd_[A-Za-z0-9_-]{3,159}$/.test(recibo.referencia)
    && Number.isSafeInteger(recibo.version) && recibo.version > 0
    && typeof recibo.registrado_en === 'string' && /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{6}Z$/.test(recibo.registrado_en)
    && Number.isFinite(Date.parse(recibo.registrado_en)) && typeof recibo.repeticion === 'boolean');
  exigir(comision && typeof comision.referencia === 'string' && Number.isSafeInteger(comision.version)
    && comision.version === recibo.version);
  exigir((modo === 'nuevo' && http === 201 && recibo.repeticion === false)
    || (modo === 'recuperacion' && http === 200 && recibo.repeticion === true));
  exigir(comision.estado === estadoEsperado);
  if (estadoEsperado === 'enviado_pendiente_revision') exigir(comision.documento && Array.isArray(comision.documento.tramos_aceptados));
  return { comision_ref: comision.referencia, recibo_ref: recibo.referencia, registrado_en: recibo.registrado_en,
    version: recibo.version, repeticion: recibo.repeticion, estado: comision.estado,
    huella: huella(JSON.stringify([recibo.referencia, recibo.version, recibo.registrado_en, comision.referencia, comision.estado])) };
}

export function validarRuta(body) {
  exigir(body?.data?.code === 'Ok' && body.data.engine === 'osrm_on_premise'
    && typeof body.data.data_version === 'string' && body.data.data_version.length > 0
    && Array.isArray(body.data.routes) && body.data.routes.length > 0);
}

export function validarDesglose(body) {
  const c = body?.comision;
  exigir(c?.vehiculo_propio === true && Array.isArray(c.rutas) && c.rutas.length > 0
    && Array.isArray(c.documento?.lineas)
    && c.documento.lineas.some(l => l.tipo === 'kilometraje')
    && c.documento.lineas.some(l => l.tipo === 'otro_medio' || l.tipo === 'otro_gasto'));
}

export function validarPuertaPersonal(body) {
  exigir(Array.isArray(body?.relaciones_autorizadas) && body.relaciones_autorizadas.length > 0
    && body.relaciones_autorizadas.length <= 64 && typeof body.fecha_referencia === 'string'
    && Number.isFinite(Date.parse(body.fecha_referencia)));
  exigir(body.relaciones_autorizadas.every(r => typeof r.relacion_ref === 'string' && r.relacion_ref.startsWith('rel_')
    && typeof r.unidad_ref === 'string' && r.unidad_ref.length > 0 && Number.isSafeInteger(r.version) && r.version > 0));
}

export function validarPuertaCompetencias(body, etapa) {
  exigir(body?.fuente === 'acreditada' && Array.isArray(body.etapas) && body.etapas.includes(etapa));
}

async function controles(page, context, datos) {
  const pagina = await page.evaluate(async () => ({
    desbordamiento: document.documentElement.scrollWidth > document.documentElement.clientWidth,
    almacenamiento: localStorage.length + sessionStorage.length + (await indexedDB.databases()).length,
    caches: (await caches.keys()).length,
  }));
  Object.assign(datos, pagina, { cookies: (await context.cookies()).length });
  exigir(!Object.values(pagina).some(Boolean) && !datos.cookies && !datos.errores_js && !datos.bloqueadas && !datos.red_fallida && !datos.http_fallidos);
}

export async function recorrer(c, casoId, chromium, salida) {
  const { caso, identidad, pasos } = validarConfiguracion(c, casoId);
  const resultado = { version: 1, caso: caso.id, estado: 'EN_CURSO', commit_servido_declarado: c.commit_servido,
    escenario_sha256: huella(JSON.stringify([c.origen, casoId, c.idioma])), servidor_instalado_verificado: false,
    reinicio_verificado: false, pasos: [], efectos: [] };
  const guardar = () => fs.writeFileSync(path.join(salida, 'resultado.json'), JSON.stringify(resultado, null, 2), { mode: 0o600 });
  guardar();
  let browser;
  let referenciaRetorno;
  try {
    browser = await chromium.launch({ executablePath: '/usr/bin/google-chrome', headless: true });
    for (const width of [1440, 390]) {
      const datos = { ancho: width, bloqueadas: 0, red_fallida: 0, http_fallidos: 0, errores_js: 0, mutaciones_enviadas: 0, estado: 'EN_CURSO' };
      resultado.pasos.push(datos); guardar();
      const context = await browser.newContext({ clientCertificates: [{ origin: c.origen,
        certPath: identidad.certificado, keyPath: identidad.clave }], ignoreHTTPSErrors: false,
        serviceWorkers: 'block', locale: idiomas.disponibles[c.idioma].locale, timezoneId: 'Europe/Madrid',
        viewport: { width, height: 900 } });
      try {
        await context.route('**/*', route => interceptar(route, c.origen, casoId, datos));
        await context.routeWebSocket('**/*', async route => { datos.bloqueadas++; await route.close(); });
        const page = await context.newPage();
        page.on('pageerror', () => { datos.errores_js++; });
        page.on('dialog', dialog => dialog.dismiss());
        const personalPendiente = page.waitForResponse(r => new URL(r.url()).pathname === '/api/vec/personal/relaciones-dietas', { timeout: 20000 });
        const etapa = etapasDecision[casoId];
        const competenciasPendiente = etapa ? page.waitForResponse(r => new URL(r.url()).pathname === `${CIRCUITO}/competencias`, { timeout: 20000 }) : null;
        const [entrada, personal, competencias] = await Promise.all([
          page.goto(`${c.origen}/portal-empleado/?lang=${c.idioma}#dietas`, { waitUntil: 'domcontentloaded', timeout: 20000 }),
          personalPendiente, competenciasPendiente,
        ]);
        exigir(entrada?.status() === 200);
        await page.locator('[data-dietas-recorridos]').waitFor({ state: 'visible', timeout: 20000 });
        exigir(personal.status() === 200); validarPuertaPersonal(await personal.json());
        if (competencias) { exigir(competencias.status() === 200); validarPuertaCompetencias(await competencias.json(), etapa); }
        exigir((await page.locator('html').getAttribute('lang'))?.startsWith(c.idioma));
        if (width === 1440) {
          for (const paso of pasos) {
            const loc = page.locator(paso.selector).first();
            if (paso.tipo === 'visible') await loc.waitFor({ state: 'visible', timeout: 20000 });
            else if (paso.tipo === 'click') await loc.click();
            else if (paso.tipo === 'fill') await loc.fill(paso.valor);
            else if (paso.tipo === 'select') await loc.selectOption(paso.valor);
            else if (paso.tipo === 'ruta') {
              const pendiente = page.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname === '/api/vec/dietas/road-route', { timeout: 20000 });
              const [response] = await Promise.all([pendiente, loc.click()]);
              exigir(response.status() === 200); validarRuta(await response.json());
            }
            else {
              const antes = datos.mutaciones_enviadas;
              datos.efectoEsperado = { metodo: paso.metodo, ruta: paso.ruta, etapa: paso.etapa, decision: paso.decision, consumida: false };
              try {
                const pendiente = page.waitForResponse(r => r.request().method() === paso.metodo && new URL(r.url()).pathname === paso.ruta, { timeout: 20000 });
                const [response] = await Promise.all([pendiente, loc.click()]);
                exigir(datos.efectoEsperado.consumida === true && datos.mutaciones_enviadas === antes + 1);
                const body = await response.json();
                const recibo = validarRecibo(body, paso.estado_esperado, paso.modo ?? 'nuevo', response.status());
                if (casoId === 'empleado_gastos_km') validarDesglose(body);
                if (casoId === 'retorno_reenvio') {
                  if (referenciaRetorno) exigir(referenciaRetorno === body.comision.referencia && recibo.version > resultado.efectos.at(-1).version);
                  referenciaRetorno = body.comision.referencia;
                }
                resultado.efectos.push({ metodo: paso.metodo, ruta_tipo: caso.efecto, http: response.status(), ...recibo });
                if (['revision', 'autorizacion', 'liquidacion', 'fiscalizacion'].includes(casoId)) {
                  await page.locator('[data-dietas-circuito-estado][data-nivel="exito"]').waitFor({ state: 'visible', timeout: 20000 });
                } else {
                  await page.locator('[data-dietas-borrador-recibo]').waitFor({ state: 'visible', timeout: 20000 });
                }
              } finally { delete datos.efectoEsperado; }
            }
            guardar();
          }
          exigir(resultado.efectos.length === patrones[casoId].length && datos.mutaciones_enviadas === patrones[casoId].length);
        }
        await page.waitForLoadState('networkidle', { timeout: 20000 });
        await controles(page, context, datos);
        datos.estado = 'COMPROBADO'; guardar();
      } finally { await context.close(); }
    }
    resultado.estado = 'RECORRIDO_SINTETICO_COMPROBADO'; guardar(); return resultado;
  } catch {
    resultado.estado = 'CORTADO'; resultado.corte = 'recorrido'; guardar(); throw new Error('recorrido');
  } finally { await browser?.close(); }
}

export async function main(argv = process.argv.slice(2)) {
  if (argv.length === 1 && argv[0] === '--plan') { console.log(JSON.stringify(plan(), null, 2)); return 0; }
  try {
    const opt = {};
    for (let i = 0; i < argv.length; i += 2) {
      exigir(['--config', '--caso', '--salida', '--escrituras-sinteticas'].includes(argv[i]) && argv[i + 1] && !Object.hasOwn(opt, argv[i]));
      opt[argv[i]] = argv[i + 1];
    }
    exigir(opt['--escrituras-sinteticas'] === 'si' && opt['--config'] && opt['--caso'] && opt['--salida']);
    const c = JSON.parse(fs.readFileSync(rutaExterna(opt['--config'], { privado: true }), 'utf8'));
    const idioma = Object.hasOwn(idiomas.disponibles, c.idioma) ? c.idioma : idiomas.respaldo;
    validarConfiguracion(c, opt['--caso']);
    const modulo = rutaExterna(process.env.VEC_PLAYWRIGHT_MODULE);
    const { chromium } = await import(pathToFileURL(modulo).href);
    exigir(chromium && typeof chromium.launch === 'function');
    const salida = prepararSalida(opt['--salida']);
    try { await recorrer(c, opt['--caso'], chromium, salida); }
    catch { console.error(mensajes[idioma].recorrido_cortado); return 1; }
    console.log(mensajes[idioma].recorrido_comprobado); return 0;
  } catch { console.error(mensajes[idiomas.respaldo].entrada_invalida); return 2; }
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) process.exitCode = await main();
