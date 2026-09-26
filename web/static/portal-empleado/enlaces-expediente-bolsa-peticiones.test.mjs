import assert from "node:assert/strict";
import test from "node:test";
import { renderizarExpediente } from "./modulos/contratacion-temporal/componentes-expedientes.js";
import { crearTraductorExpedientesContratacion } from "./modulos/contratacion-temporal/i18n-expedientes.js";
import { EVENTO_EXPEDIENTES_CENTRO, renderizarPeticionCentro, renderizarPeticionesCentroRRHH } from "./peticiones-centro/peticiones-centro.js";
import { EVENTO_EXPEDIENTES_CENTRO as EVENTO_INCORPORACIONES, idFilaExpediente } from "./peticiones-centro/incorporaciones-centro.js";

// Los datos que tiene sentido abrir llevan enlace a su pantalla, y solo si el
// perfil la ve: la bolsa del expediente de CT y el expediente de una petición entregada.

const t = crearTraductorExpedientesContratacion({});
const campo = (clave, etiqueta, valor) => ({ clave, etiqueta, valor, tono: "neutro", control: "solo_lectura", obligatorio: false, opciones: [] });
const estadoCT = {
  carga: "listo", tarea_ref: "",
  expediente: { expediente_ref: "expediente:sintetico:1", numero_visible: "2026/CT-00001", version: 3, fases: [], tareas: [], hitos: [],
    cabecera: [campo("via_cobertura", "Vía de cobertura", "Bolsa vigente"), campo("bolsa_cobertura", "Bolsa", "bolsa:sintetica:auxiliar")] },
};

test("la bolsa del expediente enlaza con su histórico de llamamientos si el perfil la ve", () => {
  const html = renderizarExpediente(estadoCT, t, "es-ES", "Europe/Madrid", false,
    (ref) => (ref === "bolsa:sintetica:auxiliar" ? { categoria: "Auxiliar administrativo" } : null));
  assert.match(html, /<button type="button" class="enlace-tabla" data-accion="ver-bolsa" data-bolsa-ref="bolsa:sintetica:auxiliar" data-pestana="historico" aria-label="Bolsa Auxiliar administrativo\. Abrir su histórico de llamamientos">Auxiliar administrativo<\/button>/u);
});

test("sin acceso a la bolsa no se muestra ni enlace ni la referencia opaca", () => {
  for (const resolver of [null, () => null]) {
    const html = renderizarExpediente(estadoCT, t, "es-ES", "Europe/Madrid", false, resolver);
    assert.doesNotMatch(html, /bolsa:sintetica:auxiliar|data-accion="ver-bolsa"/u);
    assert.match(html, /Bolsa vigente/u);
  }
});

const peticion = { referencia: "peticion:centro:sintetica-1", version: 2, estado: "ratificada", solicitud: { centro_ref: "cen_sintetico_001" }, creada_en: "2026-09-20T08:00:00Z" };

test("el centro enlaza cada petición entregada con la fila de su expediente en incorporaciones", () => {
  assert.equal(EVENTO_EXPEDIENTES_CENTRO, EVENTO_INCORPORACIONES);
  const destino = idFilaExpediente({ expediente_ref: "expediente:sintetico:1" });
  assert.equal(destino, "ic-exp-expediente-sintetico-1");
  const contexto = { actor: { referencia: "actor:1", puede_presentar: true, puede_ratificar: false, nombre: "Antonio Reyes Álvarez", cargo: "Dirección", centro: "Centro" } };
  const html = renderizarPeticionCentro({ contexto, peticiones: [peticion],
    expedientes: new Map([[peticion.referencia, { peticion_ref: peticion.referencia, numero_visible: "2026/CT-00001", destino }]]) });
  assert.match(html, /<a class="pc-enlace-expediente" href="#ic-exp-expediente-sintetico-1" data-pc-ir-expediente="ic-exp-expediente-sintetico-1" aria-label="Petición peticion:centro:sintetica-1: ver su expediente 2026\/CT-00001 en las incorporaciones del centro">Expediente 2026\/CT-00001<\/a>/u);
  assert.doesNotMatch(renderizarPeticionCentro({ contexto, peticiones: [peticion] }), /pc-enlace-expediente/u);
});

test("RRHH enlaza cada petición entregada con su expediente en Contratación temporal", () => {
  const recibo = { expediente_ref: "expediente:sintetico:1", numero_visible: "2026/CT-00001", recibo_ref: "recibo:1", confirmada_en: "2026-09-20T08:00:00Z" };
  const html = renderizarPeticionesCentroRRHH({ peticiones: [
    { peticion, estado_entrega: "confirmada", recibo_alta: recibo },
    { peticion: { ...peticion, referencia: "peticion:centro:sintetica-2" }, estado_entrega: "preparada", recibo_alta: null },
  ] });
  assert.equal(html.match(/class="pc-enlace-expediente"/gu)?.length, 1);
  assert.match(html, /href="\/portal-empleado\/\?expediente=expediente%3Asintetico%3A1#contratacion-temporal" aria-label="Petición peticion:centro:sintetica-1: abrir su expediente 2026\/CT-00001 en Contratación temporal">Expediente 2026\/CT-00001<\/a>/u);
});
