import assert from "node:assert/strict";
import test from "node:test";
import { renderizarLlamamiento } from "./renderizado-llamamiento.js";
import { crearTraductorContratacionTemporal } from "./i18n.js";
import {
  recibo as seleccion, comunicacionRegistrada, justificante, declaracion,
  reciboResolucion, continuacionConfirmada, avisoSiguienteRegistrado,
  justificanteSiguiente, declaracionSiguiente, reciboResolucionSucesor,
} from "./formulario-llamamiento-pruebas.js";

const fecha = new Intl.DateTimeFormat("es-ES", {
  dateStyle: "medium", timeStyle: "medium", timeZone: "Europe/Madrid",
});
const t = crearTraductorContratacionTemporal();
const paso = (recibo, extra = {}) => ({
  recibo, valores: {}, ocupado: false, calculando: false, solicitud: null,
  bloqueado: false, claveConservada: false, tono: "exito", mensaje: "llamamiento_sin_recibo",
  ...extra,
});
const respuesta = justificante({ ...declaracion(), recibida_en: "2026-09-05T08:30:00Z" });
const respuestaSiguiente = justificanteSiguiente({ ...declaracionSiguiente(), recibida_en: "2026-09-05T09:08:00Z" });
const resolucion = reciboResolucion("renuncia");
const resolucionSiguiente = reciboResolucionSucesor("aceptacion");
const propuesta = {
  estado_local: "confirmado", propuesta_ref: "propuesta:sintetica:001",
  recibo_local_ref: "recibo:propuesta:001", version_resultante: 7,
  confirmada_en: "2026-09-06T10:00:00.123456Z",
};

function estadoCompleto() {
  return {
    enlazado: true, comunicacionAbierta: true,
    seleccion: paso(seleccion, { solicitud: { version_esperada: 6 } }),
    comunicacion: paso(comunicacionRegistrada),
    respuesta: paso(respuesta), resolucion: paso(resolucion),
    siguiente: paso(continuacionConfirmada),
    comunicacion_siguiente: paso(avisoSiguienteRegistrado),
    respuesta_siguiente: paso(respuestaSiguiente),
    resolucion_siguiente: paso(resolucionSiguiente),
    propuesta: paso(propuesta, { aceptacion: resolucionSiguiente, disponible: true }),
  };
}

function seccionRecibo(html, operacion) {
  const hallada = html.match(new RegExp(`<section class="ct-recibo" data-ct-llamamiento-recibo="${operacion}"[\\s\\S]*?<\\/section>`, "u"));
  assert.ok(hallada, `falta recibo de ${operacion}`);
  return hallada[0];
}

test("los nueve recibos localizan sus instantes y conservan ISO exacto y precisión accesible", () => {
  const estado = estadoCompleto();
  const antes = JSON.stringify(estado);
  const html = renderizarLlamamiento(estado, t, fecha);
  const esperados = {
    seleccion: [seleccion.confirmada_en],
    comunicacion: [comunicacionRegistrada.registrada_en],
    respuesta: [respuesta.recibida_en, respuesta.registrada_en],
    resolucion: [resolucion.resuelta_en, resolucion.intencion_siguiente.actualizada_en],
    siguiente: [continuacionConfirmada.confirmada_en],
    comunicacion_siguiente: [avisoSiguienteRegistrado.registrada_en],
    respuesta_siguiente: [respuestaSiguiente.recibida_en, respuestaSiguiente.registrada_en],
    resolucion_siguiente: [resolucionSiguiente.resuelta_en],
    propuesta: [propuesta.confirmada_en],
  };
  assert.equal(Object.keys(esperados).length, 9);
  for (const [operacion, instantes] of Object.entries(esperados)) {
    const recibo = seccionRecibo(html, operacion);
    for (const iso of instantes) {
      const visible = fecha.format(new Date(iso));
      assert.ok(recibo.includes(`<time datetime="${iso}" title="${iso}" aria-label="${visible} · ${iso}">${visible}</time>`),
        `${operacion}: fecha visible y exacta ${iso}`);
      assert.ok(!recibo.includes(`<dd>${iso}</dd>`), `${operacion}: no debe quedar ISO crudo`);
    }
  }
  assert.equal(JSON.stringify(estado), antes, "el renderizado no modifica el recibo original");
});

test("el resumen original y el del sucesor coinciden con sus recibos", () => {
  for (const sucesor of [false, true]) {
    const estado = estadoCompleto();
    if (!sucesor) estado.siguiente = paso(null);
    const html = renderizarLlamamiento(estado, t, fecha);
    const resumen = html.match(/<section class="ct-llamamiento-resultado"[\s\S]*?<\/section>/u)?.[0];
    assert.ok(resumen);
    const declaracionActual = sucesor ? respuestaSiguiente : respuesta;
    const resolucionActual = sucesor ? resolucionSiguiente : resolucion;
    for (const [operacion, iso] of [
      [sucesor ? "respuesta_siguiente" : "respuesta", declaracionActual.registrada_en],
      [sucesor ? "resolucion_siguiente" : "resolucion", resolucionActual.resuelta_en],
    ]) {
      const visible = fecha.format(new Date(iso));
      const marcado = `<time datetime="${iso}" title="${iso}" aria-label="${visible} · ${iso}">${visible}</time>`;
      assert.ok(resumen.includes(marcado));
      assert.ok(seccionRecibo(html, operacion).includes(marcado));
    }
  }
});

test("los datos extraños se escapan en los campos del recibo", () => {
  const estado = estadoCompleto();
  estado.seleccion.recibo = { ...seleccion, confirmada_en: '<img src=x onerror="alert(1)">', recibo_ref: '<script>alert(1)</script>' };
  const html = renderizarLlamamiento(estado, t, fecha);
  const recibo = seccionRecibo(html, "seleccion");
  assert.ok(recibo.includes('&lt;img src=x onerror=&quot;alert(1)&quot;&gt;'));
  assert.ok(recibo.includes('&lt;script&gt;alert(1)&lt;/script&gt;'));
  assert.ok(!recibo.includes("<img") && !recibo.includes("<script"));
});
