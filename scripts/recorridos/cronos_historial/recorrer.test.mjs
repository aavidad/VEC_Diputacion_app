import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import { pathToFileURL } from 'node:url';
import test from 'node:test';
import { API, plan, validarConfig, interceptarHistorial, validarConsulta, comprobarReferencias, referenciasVisibles, recorrer } from './recorrer.mjs';
import { prepararSalida, rutaExterna, idiomas } from '../../recorridos-f/config.mjs';

const config = { version: 1, sintetico: true, entorno_controlado: true, origen: 'https://127.0.0.1:8443',
  commit_servido: 'a'.repeat(40), anio: plan.fixture.anio,
  identidad: { certificado: '/fixture-no-certificado.crt', clave: '/fixture-no-clave.key' } };
test('plan_6_visitas_12_superficies', () => {
  assert.equal(plan.idiomas.length * plan.visitas.length * 2, 12);
  assert.equal(validarConfig(config).origen, config.origen);
});
for (const [id, cambio] of Object.entries({ sintetico: { sintetico: false }, remoto: { origen: 'https://example.invalid' },
  http: { origen: 'http://127.0.0.1:8443' }, perfil: { perfil: 'administrador' }, actor: { actor_ref: 'persona:otra' },
  anio: { anio: 1999 }, certificado: { identidad: { certificado: 'relativo', clave: '/fixture.key' } },
  clave: { identidad: { certificado: '/fixture.crt' } }, cabeceras: { identidad: { ...config.identidad, headers: {} } } })) {
  test(`config_rechaza_${id}`, () => assert.throws(() => validarConfig({ ...config, ...cambio })));
}

for (const [id, metodo, destino, estado, cookie, permitir] of [
  ['lectura', 'GET', `${config.origen}${API}?anio=${config.anio}`, 200, false, true],
  ['post', 'POST', `${config.origen}${API}`, 200, false, false],
  ['anio_ajeno', 'GET', `${config.origen}${API}?anio=2025`, 200, false, false],
  ['actor', 'GET', `${config.origen}${API}?anio=${config.anio}&actor_ref=otra`, 200, false, false],
  ['otro_cronos', 'GET', `${config.origen}/api/interna/cronos/saldos/propio`, 200, false, false],
  ['administracion', 'GET', `${config.origen}/api/interna/administracion/perfiles`, 200, false, false],
  ['vec_ajena', 'GET', `${config.origen}/api/vec/usuarios/administracion`, 200, false, false],
  ['publico_ajeno', 'GET', `${config.origen}/api/publico/bolsa/convocatorias`, 200, false, false],
  ['api_encoded', 'GET', `${config.origen}/%61pi/interna/administracion/perfiles`, 200, false, false],
  ['slash_encoded', 'GET', `${config.origen}/assets/%2fapi%2finterna%2fadministracion`, 200, false, false],
  ['api_double_slash', 'GET', `${config.origen}//api/interna/administracion`, 200, false, false],
  ['dot_normalizada', 'GET', `${config.origen}/assets/../api/interna/administracion`, 200, false, false],
  ['dot_encoded_normalizada', 'GET', `${config.origen}/assets/%2e%2e/api/interna/administracion`, 200, false, false],
  ['query_duplicada', 'GET', `${config.origen}${API}?anio=${config.anio}&anio=${config.anio}`, 200, false, false],
  ['query_codificada', 'GET', `${config.origen}${API}?%61nio=${config.anio}`, 200, false, false],
  ['shell_query', 'GET', `${config.origen}/api/vec/session?actor_ref=otra`, 200, false, false],
  ...['/api/vec/modules', '/api/vec/session', '/api/vec/usuarios/mis-preferencias', '/api/vec/usuarios/mis-correos',
    '/api/vec/usuarios/mi-imagen', '/api/vec/bolsa/bolsas'].map((ruta, i) => [`shell_${i}`, 'GET', `${config.origen}${ruta}`, 200, false, true]),
  ['otro_origen', 'GET', `https://example.invalid${API}?anio=${config.anio}`, 200, false, false],
  ['redirect', 'GET', `${config.origen}${API}?anio=${config.anio}`, 302, false, false],
  ['set_cookie', 'GET', `${config.origen}${API}?anio=${config.anio}`, 200, true, false],
]) test(`frontera_${id}`, async () => {
  let enviadas = 0, abortadas = 0, entregadas = 0;
  const datos = { bloqueadas: 0, red_fallida: 0 };
  await interceptarHistorial({ request: () => ({ method: () => metodo, url: () => destino }), abort: async () => abortadas++,
    fetch: async () => { enviadas++; return { url: () => destino, status: () => estado, headersArray: async () => cookie ? [{ name: 'Set-Cookie', value: 'sintetica=;Max-Age=0' }] : [] }; },
    fulfill: async () => entregadas++ }, config, datos);
  assert.equal(entregadas, permitir ? 1 : 0); assert.equal(abortadas, permitir ? 0 : 1);
  if (!permitir && !['redirect', 'set_cookie'].includes(id)) assert.equal(enviadas, 0);
});

test('denegacion_no_se_valida_como_datos', async () => {
  let evaluaciones = 0;
  await assert.rejects(validarConsulta({ evaluate: async () => evaluaciones++ }, { status: () => 403 }, config.anio));
  assert.equal(evaluaciones, 0);
});

const filas41 = Array.from({ length: plan.fixture.cantidad_solicitudes }, (_, i) => ({ ref: `${plan.fixture.solicitud.solicitud_ref}-${i}`, clave: String(i) }));
test('refs_pagina_2_mismo_conteo_diferentes_refs', () => {
  const segunda = filas41.slice(plan.tamano_pagina, plan.tamano_pagina * 2);
  comprobarReferencias(segunda.map(f => f.clave), filas41, segunda.map(f => f.ref));
  assert.throws(() => comprobarReferencias(filas41.slice(0, plan.tamano_pagina).map(f => f.clave), filas41, segunda.map(f => f.ref)));
});
test('refs_regreso_y_filtro_no_aceptan_otra_pagina', () => {
  const primera = filas41.slice(0, plan.tamano_pagina);
  comprobarReferencias(primera.map(f => f.clave), filas41, primera.map(f => f.ref));
  assert.throws(() => comprobarReferencias(filas41.slice(plan.tamano_pagina, plan.tamano_pagina * 2).map(f => f.clave), filas41, primera.map(f => f.ref)));
});
test('refs_ambiguas_o_repetidas_se_rechazan', () => {
  assert.throws(() => referenciasVisibles(['0'], [{ ref: 'a', clave: '0' }, { ref: 'b', clave: '0' }]));
  assert.throws(() => referenciasVisibles(['0', '0'], filas41));
  assert.throws(() => referenciasVisibles(['desconocida'], filas41));
});

// Bytes fixture y montaje real; sin VEC, TLS ni certificado real. El espía
// confirma las opciones mTLS que el runner entrega al SDK. Sólo este ensayo
// elimina esos paths ficticios antes de abrir su contexto completamente offline.
async function ensayarChrome() {
  const soloMutante = process.argv.includes('--mutante-solo');
  const salida = prepararSalida(process.env.VEC_F_CRONOS_SALIDA);
  const { chromium } = await import(pathToFileURL(rutaExterna(process.env.VEC_PLAYWRIGHT_MODULE)).href);
  const origenRaiz = path.resolve(new URL('../../../web/static/', import.meta.url).pathname);
  const opcionesCertificado = [];
  let duplicarPagina = false;
  const chromiumFixture = { launchPersistentContext: async (...args) => {
    const real = await chromium.launchPersistentContext(...args);
    const browser = real.browser();
    return new Proxy(real, { get(obj, key) {
      if (key === 'browser') return () => ({ newContext: async options => {
        opcionesCertificado.push(options.clientCertificates);
        const { clientCertificates, ...offline } = options;
        return browser.newContext(offline);
      } });
      const v = Reflect.get(obj, key); return typeof v === 'function' ? v.bind(obj) : v;
    } });
  } };
  const transporte = async (route, c, datos) => {
    const u = new URL(route.request().url());
    let body, contentType;
    if (u.pathname === '/portal-empleado/') {
      const lang = u.searchParams.get('lang'), t = idiomas.disponibles[lang].accesibilidad_fixture;
      body = `<!doctype html><html lang="${lang}"><title>${t.titulo}</title>
        <link rel="stylesheet" href="/portal-empleado/portal.css"><link rel="stylesheet" href="/portal-empleado/portal-componentes.css">
        <link rel="stylesheet" href="/portal-empleado/modulos/cronos/cronos.css"><link rel="stylesheet" href="/portal-empleado/modulos/cronos/vista-solicitudes.css">
        <style>body{margin:16px;min-width:0}main{min-width:0}</style><main id="lectura"></main>
        <script type="module">import {montarPermisosPropiosCronos} from '/portal-empleado/modulos/cronos/vista-permisos-propios.js?v=fixture';
        montarPermisosPropiosCronos({raiz:document.querySelector('#lectura'),anio:${c.anio}});
        ${duplicarPagina ? `document.addEventListener('click',e=>{if(e.target.closest('[data-cronos-historial-pagina="siguiente"]')){
          const selector='section[aria-labelledby="cronos-historial-titulo"] tbody';
          const primera=document.querySelector(selector).innerHTML;
          setTimeout(()=>{document.querySelector(selector).innerHTML=primera;},0);}},true);` : ''}</script></html>`;
      contentType = 'text/html';
    } else if (u.pathname === API) {
      const lang = datos.idioma;
      const f = plan.fixture;
      const dto = { anio: f.anio, permisos: [{ ...f.permiso, nombre: f.nombres[lang] }],
        solicitudes: Array.from({ length: f.cantidad_solicitudes }, (_, i) => {
          const fecha = new Date(`${f.solicitud.desde}T12:00:00Z`); fecha.setUTCDate(fecha.getUTCDate() + i);
          return { ...f.solicitud, desde: fecha.toISOString().slice(0, 10), hasta: fecha.toISOString().slice(0, 10),
            solicitud_ref: `${f.solicitud.solicitud_ref}-${i}`, estado: i === 0 ? f.estado_alternativo : f.solicitud.estado };
        }) };
      body = JSON.stringify(dto); contentType = 'application/json';
    } else {
      const archivo = path.resolve(origenRaiz, `.${u.pathname}`);
      if (!archivo.startsWith(`${origenRaiz}${path.sep}`) || !fs.existsSync(archivo) || !fs.statSync(archivo).isFile()) {
        datos.bloqueadas++; await route.abort(); return;
      }
      body = fs.readFileSync(archivo);
      contentType = { '.js': 'text/javascript', '.css': 'text/css', '.json': 'application/json', '.svg': 'image/svg+xml' }[path.extname(archivo)] || 'application/octet-stream';
    }
    const response = { url: () => u.href, status: () => 200, headersArray: async () => [{ name: 'content-type', value: contentType }] };
    await interceptarHistorial({ request: () => route.request(), abort: () => route.abort(), fetch: async () => response,
      fulfill: () => route.fulfill({ status: 200, contentType, body }) }, c, datos);
  };
  if (!soloMutante) {
    const informe = await recorrer(config, chromiumFixture, salida, transporte, 'prueba_guion');
    assert.equal(informe.pasos.length, 6);
    assert.equal(informe.pasos.flatMap(p => p.accesibilidad).length, 12);
    assert.ok(informe.pasos.every(p => p.filtro_teclado && p.pagina_anterior && p.pagina_siguiente && p.consultas_api === 1));
    assert.equal(informe.autenticacion_acreditada, false); assert.equal(informe.persistencia_acreditada, false);
    assert.deepEqual(fs.readdirSync(salida), ['resultado.json']);
    const raw = JSON.stringify(informe);
    assert.ok(!raw.includes(plan.fixture.solicitud.solicitud_ref) && !raw.includes(plan.fixture.permiso.permiso_ref));
  }
  duplicarPagina = true;
  const salidaMutante = prepararSalida(`${process.env.VEC_F_CRONOS_SALIDA}-mutante`);
  await assert.rejects(recorrer(config, chromiumFixture, salidaMutante, transporte, 'prueba_guion'), { message: 'historial' });
  const mutante = JSON.parse(fs.readFileSync(path.join(salidaMutante, 'resultado.json')));
  assert.equal(mutante.estado, 'CORTADO'); assert.equal(mutante.corte, 'historial');
  assert.ok(opcionesCertificado.every(v => v.length === 1 && v[0].origin === config.origen
    && v[0].certPath === config.identidad.certificado && v[0].keyPath === config.identidad.clave));
  console.log(JSON.stringify({ tipo_ejecucion: 'prueba_guion', visitas_positivas: soloMutante ? 0 : 6, superficies_positivas: soloMutante ? 0 : 12, contextos_mtls_observados: opcionesCertificado.length,
    pagina_duplicada_rechazada: true, pkis_generadas: 0, autenticacion_acreditada: false, servicios_arrancados: 0 }));
}
if (process.argv.includes('--ensayar-chrome')) await ensayarChrome();
