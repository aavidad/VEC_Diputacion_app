import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { crearTraductorComunicaciones, MENSAJES_COMUNICACIONES_ES } from "./i18n.js";
import { montarVistaComunicaciones, normalizarConsultaComunicaciones, renderizarVistaComunicaciones } from "./vista.js";

const COMUNICACION = Object.freeze({
  referencia: "ref-prueba-1", asunto: "Aviso de prueba <script>", canal_previsto: "Buzón corporativo",
  evidencias: { transporte: { estado: "aceptado", recibo_ref: "recibo-transporte-1" } },
});
const RESPUESTA = Object.freeze({ estado: "disponible", autorizado: true, comunicaciones: [COMUNICACION] });
const turno = () => new Promise((resolver) => setImmediate(resolver));
function raizFalsa() {
  const eventos = new Map(); const focos = [];
  return {
    eventos, focos,
    innerHTML: "",
    addEventListener(tipo, fn) { eventos.set(tipo, fn); },
    removeEventListener(tipo, fn) { assert.equal(eventos.get(tipo), fn); eventos.delete(tipo); },
    replaceChildren() { this.innerHTML = ""; },
    querySelector(selector) { return { focus() { focos.push(selector); } }; },
  };
}

test("sin fuente muestra no_configurado y cero, con envío bloqueado", () => {
  const html = renderizarVistaComunicaciones();
  assert.match(html, /No configurado · ver dependencias/u);
  assert.match(html, /0 visibles/u);
  assert.match(html, /la fuente todavía no está conectada/u);
  assert.match(html, /<button[^>]*disabled aria-disabled="true"[^>]*>Enviar<\/button>/u);
  assert.doesNotMatch(html, /Antonio López|COM-2026-|@dipgra\.es/u);
});

test("la respuesta exige autorización positiva, cardinalidad y cadena de evidencias", () => {
  assert.throws(() => normalizarConsultaComunicaciones({ estado: "disponible", comunicaciones: [COMUNICACION] }), /inválida/u);
  assert.deepEqual(normalizarConsultaComunicaciones({ estado: "denegado", comunicaciones: [COMUNICACION] }).comunicaciones, []);
  assert.equal(normalizarConsultaComunicaciones({ estado: "disponible", autorizado: true, comunicaciones: [] }).estado, "vacio");
  assert.throws(() => normalizarConsultaComunicaciones({ ...RESPUESTA, comunicaciones: [COMUNICACION, COMUNICACION] }), /duplicada/u);
  assert.throws(() => normalizarConsultaComunicaciones({ ...RESPUESTA, comunicaciones: [{ ...COMUNICACION, evidencias: { entrega: { estado: "entregado", recibo_ref: "recibo-entrega-1" } } }] }), /antecedente/u);
  assert.throws(() => normalizarConsultaComunicaciones({ ...RESPUESTA, comunicaciones: [{ ...COMUNICACION, evidencias: { transporte: { estado: "aceptado", recibo_ref: "recibo-transporte-1" }, lectura: { estado: "leido", recibo_ref: "recibo-lectura-1" } } }] }), /antecedente/u);
  assert.throws(() => normalizarConsultaComunicaciones({ ...RESPUESTA, comunicaciones: [{ ...COMUNICACION, evidencias: { transporte: { estado: "aceptado" } } }] }), /recibo transporte/u);
});

test("transporte aceptado no acredita entrega ni lectura y el texto de fuente se escapa", () => {
  const consulta = normalizarConsultaComunicaciones(RESPUESTA);
  const html = renderizarVistaComunicaciones({ estado: consulta.estado, comunicaciones: consulta.comunicaciones });
  assert.match(html, /1 visible/u);
  assert.match(html, /Aceptado/u);
  assert.match(html, /Recibo: recibo-transporte-1/u);
  assert.equal((html.match(/Sin evidencia/g) || []).length, 4);
  assert.match(html, /Aviso de prueba &lt;script&gt;/u);
  assert.doesNotMatch(html, /<script>/u);
  assert.match(html, /Envío no disponible/u);
});

test("la bandeja filtra y selecciona sin convertir el transporte en entrega", () => {
  const dos = normalizarConsultaComunicaciones({ estado: "disponible", autorizado: true, comunicaciones: [
    COMUNICACION,
    { referencia: "ref-prueba-2", asunto: "Segundo aviso", canal_previsto: "Correo corporativo", evidencias: { transporte: { estado: "aceptado", recibo_ref: "recibo-transporte-2" }, entrega: { estado: "entregado", recibo_ref: "recibo-entrega-2" }, lectura: { estado: "leido", recibo_ref: "recibo-lectura-2" } } },
  ] });
  const html = renderizarVistaComunicaciones({ estado: dos.estado, comunicaciones: dos.comunicaciones, filtro: "SEGUNDO", seleccion: "ref-prueba-2" });
  assert.match(html, /Segundo aviso/u);
  assert.doesNotMatch(html, /Aviso de prueba/u);
  assert.match(html, /Leído/u);
  assert.match(html, /data-comunicaciones-seleccionar="ref-prueba-2" aria-current="true"/u);
  const vacio = renderizarVistaComunicaciones({ estado: dos.estado, comunicaciones: dos.comunicaciones, filtro: "sin coincidencia" });
  assert.match(vacio, /No hay comunicaciones que coincidan con el filtro/u);
  assert.doesNotMatch(vacio, /data-comunicaciones-seleccionar/u);
});

test("montaje pasa de carga a consulta autorizada y conserva foco al seleccionar", async () => {
  let resolver;
  const raiz = raizFalsa();
  const vista = montarVistaComunicaciones({ raiz, fuente: { listarAutorizadas: ({ signal }) => { assert.equal(signal.aborted, false); return new Promise((ok) => { resolver = ok; }); } } });
  assert.match(raiz.innerHTML, /Cargando comunicaciones/u);
  resolver(RESPUESTA);
  await turno();
  assert.match(raiz.innerHTML, /1 visible/u);
  raiz.eventos.get("click")({ target: { closest(selector) { return selector === "[data-comunicaciones-seleccionar]" ? { dataset: { comunicacionesSeleccionar: "ref-prueba-1" } } : null; } } });
  assert.deepEqual(raiz.focos, ["#comunicaciones-detalle-titulo"]);
  vista.desmontar();
  assert.equal(raiz.eventos.size, 0);
});

test("denegación, fallo y respuesta tardía no publican datos anteriores", async () => {
  const raiz = raizFalsa();
  const resoluciones = []; const señales = [];
  const vista = montarVistaComunicaciones({ raiz, fuente: { listarAutorizadas: ({ signal }) => { señales.push(signal); return new Promise((ok) => resoluciones.push(ok)); } } });
  const segunda = vista.refrescar();
  assert.equal(señales[0].aborted, true);
  resoluciones[0](RESPUESTA);
  resoluciones[1]({ estado: "denegado", comunicaciones: [COMUNICACION] });
  await segunda; await turno();
  assert.match(raiz.innerHTML, /Consulta denegada/u);
  assert.doesNotMatch(raiz.innerHTML, /Aviso de prueba/u);
  const tercera = vista.refrescar();
  resoluciones[2]({ estado: "disponible", comunicaciones: [COMUNICACION] });
  await tercera;
  assert.match(raiz.innerHTML, /No se pudo consultar/u);
  assert.doesNotMatch(raiz.innerHTML, /Aviso de prueba/u);
  vista.desmontar();
});

test("desmontar aborta la consulta y descarta su respuesta tardía", async () => {
  const raiz = raizFalsa(); let señal; let resolver;
  const vista = montarVistaComunicaciones({ raiz, fuente: { listarAutorizadas: ({ signal }) => { señal = signal; return new Promise((ok) => { resolver = ok; }); } } });
  vista.desmontar();
  assert.equal(señal.aborted, true);
  resolver(RESPUESTA);
  await turno();
  assert.equal(raiz.innerHTML, "");
});

test("las pestañas usan flechas, conservan foco y se desmontan", () => {
  const raiz = raizFalsa(); const anuncios = [];
  const vista = montarVistaComunicaciones({ raiz, anunciar: (texto) => anuncios.push(texto) });
  let prevenido = false;
  raiz.eventos.get("keydown")({ key: "ArrowRight", preventDefault() { prevenido = true; }, target: { closest() { return { dataset: { comunicacionesPestana: "bandeja" } }; } } });
  assert.equal(prevenido, true);
  assert.match(raiz.innerHTML, /data-comunicaciones-pestana="preferencias" aria-selected="true"/u);
  assert.deepEqual(raiz.focos, ['[data-comunicaciones-pestana="preferencias"]']);
  assert.match(anuncios[0], /Preferencias/u);
  vista.desmontar();
  assert.equal(raiz.eventos.size, 0);
  assert.equal(raiz.innerHTML, "");
});

test("catálogo cerrado, sin red ni almacenamiento en la vista", async () => {
  const t = crearTraductorComunicaciones();
  assert.equal(t("visibles", { numero: 0 }), "0 visibles");
  assert.throws(() => t("clave_inexistente"), /desconocida/u);
  assert.throws(() => crearTraductorComunicaciones({ ...MENSAJES_COMUNICACIONES_ES, titulo: "" }), /incompleto/u);
  const fuente = await readFile(new URL("./vista.js", import.meta.url), "utf8");
  assert.doesNotMatch(fuente, /(?:fetch\(|localStorage|sessionStorage|document\.cookie|datos-presentacion)/u);
});
