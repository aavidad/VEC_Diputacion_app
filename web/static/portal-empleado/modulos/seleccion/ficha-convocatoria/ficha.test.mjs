import assert from "node:assert/strict";
import test from "node:test";
import { cargarTextosFicha } from "./i18n.js";
import { ErrorFichaConvocatoria, validarLecturaFicha, validarSelectorFicha } from "./contrato.js?v=20261001-s1-ficha-v1";
import { crearLectorFichaHTTP, RUTA_FICHA_CONVOCATORIA } from "./cliente-http.js";
import { montarFichaConvocatoria } from "./vista.js";
import { montarFichaConvocatoriaHTTP } from "./montaje.js";

const selector = { convocatoria_id: "convocatoria:prueba", secuencia: 2 };
function lectura(s = selector) {
  const h = "a".repeat(64);
  return {
    ficha: { convocatoria_id: s.convocatoria_id, secuencia: s.secuencia, revision: 1, fuente_ref: `${s.convocatoria_id}#${s.secuencia}`, huella_version_sha256: h,
      bases: [{ rol: "bases", publicacion_ref: "publicacion:bases", documento_ref: "documento:bases", version_documento: 1,
        representacion_ref: "representacion:bases", huella_contenido_sha256: h, firma_validada_ref: "firma:prueba", recibo_custodia_ref: "custodia:prueba" }],
      requisitos: [{ referencia: "requisito:prueba", orden: 1, titulo: "<img src=x onerror=alert(1)>", descripcion: "<script>alert(2)</script>", obligatorio: true }],
      flujo_proceso: { id: "flujo-prueba", version: 1, huella_contenido_sha256: h }, reglas_baremacion: { id: "baremo:prueba", version: 2, huella_contenido_sha256: h },
      fases_estado: "pendiente_fuente", referencias_cobertura_estado: "pendiente_fuente" },
    evidencia: { recibo_ref: "recibo:prueba", decision_ref: "decision:prueba", consumo_huella_sha256: h, auditoria_ref: "auditoria:prueba",
      correlacion_ref: "correlacion:prueba", consultada_en: "2026-10-01T12:00:00Z" },
  };
}
function response(v = lectura(), estado = 200) { return new Response(JSON.stringify(v), { status: estado, headers: { "Content-Type": "application/json" } }); }

// DOM mínimo para comprobar el consumidor, no la disposición en Chrome.
class Nodo {
  constructor(tag, doc) { this.tagName = tag; this.ownerDocument = doc; this.children = []; this.dataset = {}; this.attrs = {}; this.handlers = {}; this._texto = ""; }
  set textContent(v) { this._texto = String(v); this.children = []; }
  get textContent() { return this._texto + this.children.map((v) => v.textContent).join(" "); }
  append(...nodos) { this.children.push(...nodos); for (const n of nodos) n.parent = this; }
  replaceChildren(...nodos) { this.children = []; this.append(...nodos); }
  setAttribute(k, v) { this.attrs[k] = v; }
  addEventListener(k, fn) { this.handlers[k] = fn; }
  contains(n) { return this === n || this.children.some((v) => v.contains(n)); }
  remove() { if (this.parent) this.parent.children = this.parent.children.filter((v) => v !== this); }
  focus() { this.ownerDocument.activeElement = this; }
}
function dom() { const doc = { createElement: (tag) => new Nodo(tag, doc) }; const contenedor = new Nodo("main", doc); return { doc, contenedor }; }
function nodos(raiz, etiqueta) { return [raiz, ...raiz.children.flatMap((v) => nodos(v, etiqueta))].filter((v) => !etiqueta || v.tagName === etiqueta); }
function diferido() { let resolver; const promesa = new Promise((r) => { resolver = r; }); return { promesa, resolver }; }

for (const idioma of ["es", "en"]) {
  test(`catálogo real ${idioma}, referencias y límites sin decisiones inferidas`, async () => {
    const textos = await cargarTextosFicha({ idioma }); assert.equal(textos.modulo, "seleccion-ficha-convocatoria"); assert.equal(textos.faltantes.length, 0);
    const { contenedor } = dom(); const vista = montarFichaConvocatoria(contenedor, { textos, lector: { consultarExacta: async () => lectura() } });
    await vista.consultar(selector);
    assert.equal(contenedor.children[0].lang, idioma); assert.equal(contenedor.children[0].dataset.fichaEstado, "listo");
    assert.match(contenedor.textContent, /01\/10\/2026/u); assert.match(contenedor.textContent, /a{64}/u); assert.match(contenedor.textContent, /convocatoria:prueba#2/u);
    assert.match(contenedor.textContent, idioma === "es" ? /Pendientes de la definición|Plazas y oferta/u : /Awaiting the process definition|Positions and public/u);
    assert.equal(nodos(contenedor, "img").length, 0); assert.equal(nodos(contenedor, "script").length, 0); assert.equal(nodos(contenedor, "a").length, 0);
    assert.match(contenedor.textContent, /<script>alert\(2\)<\/script>/u);
    assert.ok(nodos(contenedor, "details").length >= 3); assert.ok(nodos(contenedor, "details").every((n) => !n.open));
  });
}

test("cliente emite sólo selector y controles exactos de transporte", async () => {
  let request;
  const lector = crearLectorFichaHTTP({ fetchImpl: async (ruta, opciones) => { request = { ruta, opciones }; return response(); } });
  const r = await lector.consultarExacta(selector);
  assert.deepEqual(r, lectura()); assert.equal(request.ruta, RUTA_FICHA_CONVOCATORIA);
  assert.deepEqual(JSON.parse(request.opciones.body), selector); assert.equal(request.opciones.method, "POST");
  for (const [clave, valor] of Object.entries({ cache: "no-store", credentials: "same-origin", mode: "same-origin", redirect: "error", referrerPolicy: "no-referrer" })) assert.equal(request.opciones[clave], valor);
  for (const invalida of [{ ...selector, actor: "intruso" }, { ...selector, secuencia: 0 }, { ...selector, convocatoria_id: "foo#2" }]) {
    assert.throws(() => validarSelectorFicha(invalida), /consulta_invalida/u);
  }
});

test("fallos HTTP no leen un cuerpo privado ni exponen datos parciales", async () => {
  for (const [estado, codigo] of [[403, "acceso_denegado"], [401, "acceso_denegado"], [404, "no_encontrada"], [503, "servicio_no_disponible"]]) {
    let leido = false;
    const lector = crearLectorFichaHTTP({ fetchImpl: async () => ({ status: estado, body: { getReader() { leido = true; } } }) });
    await assert.rejects(lector.consultarExacta(selector), (e) => e.codigo === codigo && e.estado === estado); assert.equal(leido, false);
  }
});

test("rechaza otra versión, evidencia incompleta, fases inferidas y cuerpo excesivo", async () => {
  for (const cambiar of [
    (r) => { r.ficha.secuencia = 3; }, (r) => { r.evidencia.auditoria_ref = ""; }, (r) => { r.ficha.fases_estado = "aprobada"; },
    (r) => { r.ficha.referencias_cobertura_estado = "cubierta"; }, (r) => { r.ficha.bases = [null]; }, (r) => { r.ficha.huella_version_sha256 = ""; },
    (r) => { r.evidencia.consultada_en = "2026-02-31T12:00:00Z"; },
  ]) {
    const r = lectura(); cambiar(r); assert.throws(() => validarLecturaFicha(r, selector), ErrorFichaConvocatoria);
    await assert.rejects(crearLectorFichaHTTP({ fetchImpl: async () => response(r) }).consultarExacta(selector), /respuesta_incompatible/u);
  }
  const exceso = new Response("a".repeat(4 * 1024 * 1024 + 1), { headers: { "Content-Type": "application/json" } });
  await assert.rejects(crearLectorFichaHTTP({ fetchImpl: async () => exceso }).consultarExacta(selector), /respuesta_incompatible/u);
});

test("lector cancela y caduca sin convertirlo en éxito", async () => {
  const fetchImpl = async (_r, { signal }) => new Promise((_si, no) => signal.addEventListener("abort", () => no(new Error("abort")), { once: true }));
  const controlador = new AbortController(); const lector = crearLectorFichaHTTP({ fetchImpl }); const p = lector.consultarExacta(selector, { signal: controlador.signal }); controlador.abort();
  await assert.rejects(p, /operacion_abortada/u);
  await assert.rejects(crearLectorFichaHTTP({ fetchImpl, plazoMs: 5 }).consultarExacta(selector), /servicio_no_disponible/u);
});

test("cancelación y respuesta tardía no rellenan otra versión ni una vista desmontada", async () => {
  const textos = await cargarTextosFicha(); const { contenedor } = dom(); const llamadas = [];
  const vista = montarFichaConvocatoria(contenedor, { textos, lector: { consultarExacta: (_s, o) => { const d = diferido(); llamadas.push({ ...d, signal: o.signal }); return d.promesa; } } });
  const a = vista.consultar(selector), otro = { ...selector, secuencia: 3 }, b = vista.consultar(otro);
  assert.equal(llamadas[0].signal.aborted, true); llamadas[1].resolver(lectura(otro)); await b;
  llamadas[0].resolver(lectura()); await a; assert.match(contenedor.textContent, /convocatoria:prueba#3/u); assert.doesNotMatch(contenedor.textContent, /convocatoria:prueba#2/u);
  const c = vista.consultar(selector); vista.desmontar(); llamadas[2].resolver(lectura()); await c;
  assert.equal(llamadas[2].signal.aborted, true); assert.equal(contenedor.children.length, 0);
});

test("fallo tras lectura borra datos anteriores y reintenta conservando selector", async () => {
  const textos = await cargarTextosFicha();
  for (const [codigo, estado] of [["acceso_denegado", "denegado"], ["servicio_no_disponible", "error"], ["no_encontrada", "vacio"]]) {
    const { contenedor } = dom(); let falla = false;
    const vista = montarFichaConvocatoria(contenedor, { textos, lector: { consultarExacta: async (s) => { assert.deepEqual(s, selector); if (falla) throw new ErrorFichaConvocatoria(codigo); return lectura(); } } });
    await vista.consultar(selector); falla = true; await vista.consultar(selector);
    assert.equal(contenedor.children[0].dataset.fichaEstado, estado); assert.doesNotMatch(contenedor.textContent, /a{64}|<script>|recibo:prueba/u);
    falla = false; await vista.consultar(selector); assert.equal(contenedor.children[0].dataset.fichaEstado, "listo");
  }
});


test("montaje devuelve handle inmediato y no ocupa un contenedor reutilizado durante carga de textos", async () => {
  const textos = await cargarTextosFicha(); const { contenedor, doc } = dom(); const carga = diferido(); let lecturas = 0;
  const vista = montarFichaConvocatoriaHTTP(contenedor, { selector, prepararTextos: () => carga.promesa,
    lector: { consultarExacta: async () => { lecturas++; return lectura(); } } });
  assert.equal(typeof vista.desmontar, "function"); vista.desmontar();
  const otraPantalla = doc.createElement("section"); otraPantalla.textContent = "otra vista"; contenedor.replaceChildren(otraPantalla);
  carga.resolver(textos); await vista.ready;
  assert.equal(contenedor.children[0], otraPantalla); assert.equal(lecturas, 0);
});

test("handle permite cancelar primera lectura antes de ready y conserva otra pantalla", async () => {
  const textos = await cargarTextosFicha(); const { contenedor, doc } = dom(); const pendiente = diferido(), llamada = diferido(); let signal;
  const vista = montarFichaConvocatoriaHTTP(contenedor, { selector, prepararTextos: async () => textos,
    lector: { consultarExacta: (_s, opciones) => { signal = opciones.signal; llamada.resolver(); return pendiente.promesa; } } });
  await llamada.promesa; vista.desmontar(); assert.equal(signal.aborted, true);
  const otraPantalla = doc.createElement("section"); contenedor.replaceChildren(otraPantalla);
  pendiente.resolver(lectura()); await vista.ready; assert.equal(contenedor.children[0], otraPantalla);
});
