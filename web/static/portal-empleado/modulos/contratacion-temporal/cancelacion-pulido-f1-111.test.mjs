import assert from "node:assert/strict";
import test from "node:test";

import { montarPanelCancelacion } from "./cancelacion-expediente.js?v=VEC_WEB_VERSION";
import { cargarCatalogosContratacionEnIdioma } from "./i18n-catalogos.js?v=VEC_WEB_VERSION";
import { cargarMensajesExpedientesContratacionEnIdioma, crearTraductorExpedientesContratacion } from "./i18n-expedientes.js?v=VEC_WEB_VERSION";
import { renderizarLineaFases } from "./vista-expedientes-ficha.js?v=VEC_WEB_VERSION";

// F1-111: la línea de fases de un expediente cancelado, el desplegable de
// motivos y el mismo nombre de la cancelación en el historial.
const EXP = "expediente:ct:sintetico:cancelacion-111";
const contexto = Object.freeze({ expediente_ref: EXP, version: 2, fase_clave: "solicitud", estado_clave: "en_curso", numero_visible: "2026/CT-000111" });
const motivos = [{ clave: "falta_credito", etiqueta: "Sin crédito", clave_i18n: "contratacion_temporal.cancelacion.motivo.falta_credito" }];
const esperar = () => new Promise((r) => setImmediate(r));
const FASES = ["solicitud", "analisis_rrhh", "gestion_bolsa", "fiscalizacion", "obtencion_candidato", "nombramiento", "incorporacion", "seguimiento"];

function ficha(estados) {
  return { fases: FASES.map((clave, posicion) => ({ fase_ref: `presentacion:ct:${clave}`, orden: posicion + 1,
    etiqueta: `Fase ${clave}`, estado_clave: estados[posicion] ?? "pendiente" })) };
}

async function mensajesCancelacion(idioma) {
  const { actual } = await cargarCatalogosContratacionEnIdioma("contratacion-temporal-cancelacion", idioma);
  return Object.fromEntries(Object.entries(actual).map(([clave, valor]) => [`cancelacion.${clave}`, valor]));
}

function contenedorFalso() {
  const eventos = new Map();
  return { innerHTML: "", hidden: false, eventos, addEventListener: (n, f) => eventos.set(n, f), removeEventListener: (n) => eventos.delete(n),
    querySelector: () => null };
}

for (const [idioma, cancelado, resumen, sinHacer, historial] of [
  ["es", "Cancelado en esta fase", "1 hechas · cancelado en «Fase analisis_rrhh»", "No se hará", "Expediente cancelado"],
  ["en", "Cancelled at this stage", "1 done · cancelled at “Fase analisis_rrhh”", "Will not take place", "Case cancelled"],
]) {
  test(`${idioma}: la línea de fases marca lo hecho, la fase cancelada y lo que ya no se hará`, async () => {
    const mensajes = await cargarMensajesExpedientesContratacionEnIdioma(idioma);
    const t = crearTraductorExpedientesContratacion(mensajes);
    const html = renderizarLineaFases(ficha(["completado", "cancelado"]), t);
    assert.ok(html.includes(resumen), html);
    assert.match(html, /<li class="hecho" data-ct-exp-orden="1">/u);
    assert.match(html, /<li class="cancelado" data-ct-exp-orden="2">/u);
    assert.match(html, /<span class="marca" aria-hidden="true">✕<\/span>/u);
    assert.ok(html.includes(`<small>${cancelado}</small>`));
    assert.equal(html.match(/<li class="sin-hacer"/gu)?.length, 6);
    assert.ok(html.includes(`<small>${sinHacer}</small>`));
    assert.doesNotMatch(html, /class="falta"|aria-current/u);
    assert.equal(t("hito_cancelacion_expediente"), historial);
  });
}

test("un expediente en curso conserva la línea de siempre", async () => {
  const t = crearTraductorExpedientesContratacion(await cargarMensajesExpedientesContratacionEnIdioma("es"));
  const html = renderizarLineaFases(ficha(["completado", "en_curso"]), t);
  assert.ok(html.includes("1 hechas · 1 ahora · 6 faltan"));
  assert.equal(html.match(/<li class="falta"/gu)?.length, 6);
  assert.doesNotMatch(html, /cancelado|sin-hacer/u);
});

for (const [idioma, elegir, etiqueta] of [["es", "Seleccione un motivo", "Motivo de la cancelación (obligatorio)"],
  ["en", "Select a reason", "Reason for cancellation (required)"]]) {
  test(`${idioma}: el desplegable empieza pidiendo un motivo y solo queda un botón de cancelar`, async () => {
    const c = contenedorFalso();
    const desmontar = montarPanelCancelacion({ contenedor: c, contexto, mensajes: await mensajesCancelacion(idioma), locale: idioma,
      cliente: { consultarCancelacion: async () => ({ fases_admitidas: ["solicitud"], motivos, cancelacion: null }) } });
    await esperar();
    assert.match(c.innerHTML, /data-ct-cancelacion-abrir/u);
    c.eventos.get("click")({ target: { closest: () => ({ matches: (s) => s === "[data-ct-cancelacion-abrir]", setAttribute() {} }) } });
    assert.ok(c.innerHTML.includes(`<option value="">${elegir}</option>`));
    assert.doesNotMatch(c.innerHTML, /<option value=""><\/option>/u);
    assert.doesNotMatch(c.innerHTML, /data-ct-cancelacion-abrir/u);
    assert.match(c.innerHTML, /data-ct-cancelacion-cerrar/u);
    assert.ok(c.innerHTML.includes(`<span>${etiqueta}</span>`));
    assert.doesNotMatch(c.innerHTML, /aria-invalid/u);
    // Sin motivo, el desplegable queda marcado y remite al aviso.
    const FormDataOriginal = globalThis.FormData;
    globalThis.FormData = class { constructor(f) { this.f = f; } entries() { return Object.entries(this.f.campos); } };
    try {
      c.eventos.get("submit")({ target: { closest: () => ({ campos: { motivo_clave: "", observaciones: "" } }) }, preventDefault() {} });
      await esperar();
    } finally {
      globalThis.FormData = FormDataOriginal;
    }
    assert.match(c.innerHTML, /aria-invalid="true" aria-describedby="ct-cancelacion-aviso"/u);
    assert.match(c.innerHTML, /id="ct-cancelacion-aviso"/u);
    c.eventos.get("click")({ target: { closest: () => ({ matches: (s) => s === "[data-ct-cancelacion-cerrar]" }) } });
    assert.match(c.innerHTML, /data-ct-cancelacion-abrir/u);
    desmontar();
  });
}
