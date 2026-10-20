import assert from "node:assert/strict";
import test from "node:test";

import { contextoCancelacionDesdeEstado, montarPanelCancelacion } from "./cancelacion-expediente.js?v=VEC_WEB_VERSION";
import { presentarEtiquetasHitoRRHH } from "./adaptador-http-expedientes.js?v=VEC_WEB_VERSION";
import { cargarCatalogosContratacionEnIdioma } from "./i18n-catalogos.js?v=VEC_WEB_VERSION";
import { cargarMensajesExpedientesContratacionEnIdioma, crearTraductorExpedientesContratacion } from "./i18n-expedientes.js?v=VEC_WEB_VERSION";
import { renderizarSiguientePasoFicha } from "./vista-expedientes-ficha.js?v=VEC_WEB_VERSION";

// F1-102: el panel de cancelación y el historial se entienden en los dos idiomas.
const EXP = "expediente:ct:sintetico:cancelacion-102";
const PREFIJO = "contratacion_temporal.cancelacion.motivo.";
const motivos = ["necesidad_desaparecida", "desistimiento_centro", "cobertura_otra_via", "error_solicitud",
  "solicitud_duplicada", "falta_credito"].map((clave) => ({ clave, etiqueta: `servidor ${clave}`, clave_i18n: `${PREFIJO}${clave}` }));
const consulta = (cancelacion = null) => ({ fases_admitidas: ["solicitud"], motivos, cancelacion });
const registrada = Object.freeze({ canal: "rrhh", motivo_clave: "necesidad_desaparecida", motivo_etiqueta: "Ha desaparecido la necesidad",
  motivo_clave_i18n: `${PREFIJO}necesidad_desaparecida`, fase_previa: "solicitud", observaciones: "",
  recibo_ref: "recibo:ct-seguimiento:102", registrada_en: "2026-10-20T08:30:00Z" });
const contexto = Object.freeze({ expediente_ref: EXP, version: 3, fase_clave: "solicitud", estado_clave: "en_curso", numero_visible: "2026/CT-000102" });
const esperar = () => new Promise((r) => setImmediate(r));

async function mensajesCancelacion(idioma) {
  const { actual } = await cargarCatalogosContratacionEnIdioma("contratacion-temporal-cancelacion", idioma);
  return Object.fromEntries(Object.entries(actual).map(([clave, valor]) => [`cancelacion.${clave}`, valor]));
}

function contenedorFalso() {
  const eventos = new Map();
  const enfocados = [];
  return { innerHTML: "", hidden: false, eventos, enfocados, addEventListener: (n, f) => eventos.set(n, f), removeEventListener: (n) => eventos.delete(n),
    querySelector(selector) {
      return selector === "[data-ct-cancelacion-aviso]" && this.innerHTML.includes("data-ct-cancelacion-aviso")
        ? { focus: () => enfocados.push(selector) } : null;
    } };
}

for (const [idioma, motivo, resumen] of [
  ["es", "Ha desaparecido la necesidad", "Expediente cancelado por Recursos Humanos: Ha desaparecido la necesidad."],
  ["en", "The need no longer exists", "Case cancelled by Human Resources: The need no longer exists."],
]) {
  test(`${idioma}: los motivos salen en el idioma elegido, en el desplegable y en el resumen`, async () => {
    const mensajes = await mensajesCancelacion(idioma);
    const c = contenedorFalso();
    const desmontar = montarPanelCancelacion({ contenedor: c, contexto, mensajes, locale: idioma,
      cliente: { consultarCancelacion: async () => consulta() } });
    await esperar();
    c.eventos.get("click")({ target: { closest: () => ({ matches: (s) => s === "[data-ct-cancelacion-abrir]", setAttribute() {} }) } });
    assert.ok(c.innerHTML.includes(`>${motivo}</option>`));
    assert.doesNotMatch(c.innerHTML, /servidor /u);
    desmontar();
    const otro = contenedorFalso();
    const fin = montarPanelCancelacion({ contenedor: otro, contexto: { ...contexto, estado_clave: "cancelado" }, mensajes, locale: idioma,
      cliente: { consultarCancelacion: async () => consulta(registrada) } });
    await esperar();
    assert.ok(otro.innerHTML.includes(resumen));
    fin();
  });
}

test("una clave de motivo desconocida usa el nombre que da el servidor", async () => {
  const c = contenedorFalso();
  const desmontar = montarPanelCancelacion({ contenedor: c, contexto, cliente: { consultarCancelacion: async () => consulta({
    ...registrada, motivo_clave_i18n: `${PREFIJO}otro_motivo`, motivo_etiqueta: "Otro motivo del catálogo" }) } });
  await esperar();
  assert.match(c.innerHTML, /Otro motivo del catálogo/u);
  desmontar();
});

test("al volver a montarse tras cancelar, el foco va al mensaje y el justificante sale una vez con fecha corta", async () => {
  const c = contenedorFalso();
  const desmontar = montarPanelCancelacion({ contenedor: c, contexto: { ...contexto, estado_clave: "cancelado" },
    avisoInicial: { tono: "exito", texto: "Expediente cancelado.", recibo: registrada.recibo_ref },
    cliente: { consultarCancelacion: async () => consulta(registrada) } });
  await esperar();
  assert.deepEqual(c.enfocados, ["[data-ct-cancelacion-aviso]"]);
  assert.equal(c.innerHTML.match(/data-copiar-justificante=/gu)?.length, 1);
  assert.match(c.innerHTML, /<dd>20 oct 2026, 10:30<\/dd>/u);
  desmontar();
});

test("el diálogo nombra el expediente por su número y su versión", async () => {
  for (const [idioma, esperado] of [["es", "Expediente 2026/CT-000102, versión 3."], ["en", "Case 2026/CT-000102, version 3."]]) {
    const mensajes = await mensajesCancelacion(idioma);
    const c = contenedorFalso();
    let descriptor = null;
    const FormDataOriginal = globalThis.FormData;
    globalThis.FormData = class { constructor(f) { this.f = f; } entries() { return Object.entries(this.f.campos); } };
    try {
      const desmontar = montarPanelCancelacion({ contenedor: c, contexto, mensajes,
        cliente: { consultarCancelacion: async () => consulta() }, confirmarOperacion: (d) => { descriptor = d; return false; } });
      await esperar();
      c.eventos.get("submit")({ target: { closest: () => ({ campos: { motivo_clave: "falta_credito", observaciones: "" } }) }, preventDefault() {} });
      await esperar();
      assert.equal(descriptor?.referencia, esperado);
      desmontar();
    } finally {
      globalThis.FormData = FormDataOriginal;
    }
  }
});

test("el contexto lleva el número visible del expediente", () => {
  const estado = { vista: "expediente", carga: "listo",
    expediente: { demostracion: false, expediente_ref: EXP, version: 3, numero_visible: "2026/CT-000102" },
    cuadro: { expedientes: [{ expediente_ref: EXP, version: 3, fase_clave: "solicitud", estado_clave: "en_curso" }] } };
  assert.equal(contextoCancelacionDesdeEstado(estado).numero_visible, "2026/CT-000102");
});

test("la ficha cancelada dice que está cancelada, en los dos idiomas", async () => {
  const expediente = { expediente_ref: EXP, version: 4, tareas: [], historial: [], fases: [] };
  const estado = { vista: "expediente", carga: "listo", expediente_ref: EXP,
    cuadro: { expedientes: [{ expediente_ref: EXP, version: 4, fase_clave: "solicitud", estado_clave: "cancelado" }] } };
  for (const [idioma, titulo, que] of [
    ["es", "Expediente cancelado", "El expediente está cancelado y no admite más actuaciones."],
    ["en", "Case cancelled", "The case is cancelled and no further action is possible."],
  ]) {
    const t = crearTraductorExpedientesContratacion(await cargarMensajesExpedientesContratacionEnIdioma(idioma));
    const html = renderizarSiguientePasoFicha(expediente, estado, t);
    assert.ok(html.includes(`>${titulo}</h3>`), idioma);
    assert.ok(html.includes(que), idioma);
    assert.doesNotMatch(html, /terminado|finished|data-ct-exp-accion="ir-tramite"/u);
    assert.ok(!html.includes(t("siguiente_paso_hasta")), idioma);
  }
});

test("el historial nombra la propuesta y la resolución en lenguaje llano", async () => {
  const hito = (accion_clave) => ({ accion_clave, fase_origen: "nombramiento", fase_destino: "nombramiento",
    estado_origen: "en_curso", estado_destino: "en_curso" });
  const en = await cargarMensajesExpedientesContratacionEnIdioma("en");
  assert.equal(presentarEtiquetasHitoRRHH(hito("registrar_propuesta_formalizacion")).accion, "Propuesta de nombramiento o contrato registrada");
  assert.equal(presentarEtiquetasHitoRRHH(hito("registrar_resolucion_formalizacion")).accion, "Resolución de nombramiento o contrato registrada");
  assert.equal(presentarEtiquetasHitoRRHH(hito("registrar_propuesta_formalizacion"), en).accion, "Appointment or contract proposal recorded");
  assert.equal(presentarEtiquetasHitoRRHH(hito("registrar_resolucion_formalizacion"), en).accion, "Appointment or contract decision recorded");
});
