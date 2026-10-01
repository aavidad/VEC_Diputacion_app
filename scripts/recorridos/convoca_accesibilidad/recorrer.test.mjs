import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import { pathToFileURL } from 'node:url';
import test from 'node:test';
import { plan, validarConfig, interceptarPublico, recorrer } from './recorrer.mjs';
import { prepararSalida, rutaExterna } from '../../recorridos-f/config.mjs';
import { idiomas } from '../../recorridos-f/config.mjs';

const configuracion = { version: 1, sintetico: true, entorno_controlado: true, origen: 'https://127.0.0.1:8443',
  commit_servido: 'a'.repeat(40), identificador_publico: 'sintetica-2026' };

test('plan_6_visitas_12_superficies', () => {
  assert.equal(plan.idiomas.length * plan.visitas.length * plan.superficies.length, 12);
  assert.ok(plan.visitas.some(v => v.ancho === 1440 && v.zoom === 2));
  assert.equal(validarConfig(configuracion).origen, configuracion.origen);
});
for (const cambio of [{ sintetico: false }, { origen: 'https://cidonia.cloud' }, { origen: 'http://127.0.0.1:443' },
  { identidad: 'rrhh' }, { identificador_publico: 'x"]' }, { commit_servido: 'HEAD' }]) {
  test(`configuracion_rechazada_${Object.keys(cambio)[0]}_${String(Object.values(cambio)[0])}`, () => {
    assert.throws(() => validarConfig({ ...configuracion, ...cambio }));
  });
}

test('publico_rechaza_post_consulta_ct', async () => {
  let abortadas = 0, enviadas = 0;
  const datos = { bloqueadas: 0 };
  await interceptarPublico({ request: () => ({ method: () => 'POST' }), abort: async () => abortadas++, fetch: async () => enviadas++ }, configuracion.origen, datos);
  assert.equal(abortadas, 1); assert.equal(enviadas, 0); assert.equal(datos.bloqueadas, 1);
});

for (const defecto of ['origen', 'redirect', 'cookie']) test(`interceptor_comun_rechaza_${defecto}`, async () => {
  let abortadas = 0, cumplidas = 0;
  const datos = { bloqueadas: 0, red_fallida: 0 };
  const route = { request: () => ({ url: () => defecto === 'origen' ? 'https://example.invalid/' : `${configuracion.origen}/bolsa/`, method: () => 'GET' }),
    abort: async () => abortadas++, fulfill: async () => cumplidas++, fetch: async () => ({ url: () => `${configuracion.origen}/bolsa/`,
      status: () => defecto === 'redirect' ? 302 : 200, headersArray: async () => defecto === 'cookie' ? [{ name: 'Set-Cookie', value: 'sintetica=;Max-Age=0' }] : [] }) };
  await interceptarPublico(route, configuracion.origen, datos);
  assert.equal(abortadas, 1); assert.equal(cumplidas, 0); assert.equal(datos.bloqueadas, 1);
});

// Ensayo de la herramienta: no servidor, socket ni acceso HTTP real. La frontera
// sigue pasando por el interceptor común; sólo su adaptador fetch usa bytes fixture.
async function ensayarChrome() {
  const salida = prepararSalida(process.env.VEC_F_CONVOCA_SALIDA);
  const { chromium } = await import(pathToFileURL(rutaExterna(process.env.VEC_PLAYWRIGHT_MODULE)).href);
  const fixture = JSON.parse(fs.readFileSync(new URL('../convoca_fixture.json', import.meta.url)));
  const contrato = fs.readFileSync(new URL('../../../web/static/bolsa/contrato-v2.js', import.meta.url));
  const transporte = async (route, origen, datos) => {
    const u = new URL(route.request().url());
    let body, contentType = 'application/json';
    if (u.pathname === '/bolsa/') {
      const lang = u.searchParams.get('lang');
      const t = idiomas.disponibles[lang].accesibilidad_fixture;
      contentType = 'text/html';
      body = `<!doctype html><html lang="${lang}"><title>${t.titulo}</title><style>body{margin:20px}input{max-width:85%}:focus-visible{outline:3px solid black;outline-offset:2px}</style>
        <main id="contenido-principal"><h1>${t.titulo}</h1><label for="buscar">${t.etiqueta}</label><input id="buscar">
        <article data-identificador="${fixture.listado.convocatorias[0].identificador_publico}"><a href="#" class="enlace-detalle">${t.abrir}</a></article>
        <div id="contenido-detalle" hidden><button>${t.volver}</button></div></main>
        <script src="/bolsa/contrato-v2.js"></script><script>
        fetch('/api/publico/bolsa/convocatorias').then(r=>r.json()).then(d=>VECBolsaContratoV2.validarListado(d));
        document.querySelector('.enlace-detalle').addEventListener('click',async e=>{e.preventDefault();await fetch('/api/publico/bolsa/convocatorias/${fixture.listado.convocatorias[0].identificador_publico}');document.querySelector('#contenido-detalle').hidden=false;});
        </script></html>`;
    } else if (u.pathname === '/bolsa/contrato-v2.js') { body = contrato; contentType = 'text/javascript'; }
    else if (u.pathname === '/api/publico/bolsa/convocatorias') body = JSON.stringify(fixture.listado);
    else if (u.pathname === `/api/publico/bolsa/convocatorias/${configuracion.identificador_publico}`) body = JSON.stringify(fixture.detalle);
    else { datos.bloqueadas++; await route.abort(); return; }
    const response = { url: () => u.href, status: () => 200, headersArray: async () => [{ name: 'content-type', value: contentType }] };
    await interceptarPublico({ request: () => route.request(), abort: () => route.abort(), fetch: async () => response,
      fulfill: () => route.fulfill({ status: 200, contentType, body }) }, origen, datos);
  };
  const informe = await recorrer(configuracion, chromium, salida, transporte, 'prueba_guion');
  assert.equal(informe.pasos.length, 6);
  assert.equal(informe.pasos.flatMap(p => p.accesibilidad).length, 12);
  assert.equal(informe.flujo_funcional_completado, false);
  assert.equal(informe.servidor_instalado_verificado, false);
  assert.deepEqual(fs.readdirSync(salida), ['resultado.json']);
  console.log(JSON.stringify({ tipo_ejecucion: 'prueba_guion', visitas: 6, superficies: 12, zoom_nativo: true, servicios_arrancados: 0 }));
}
if (process.argv.includes('--ensayar-chrome')) await ensayarChrome();
