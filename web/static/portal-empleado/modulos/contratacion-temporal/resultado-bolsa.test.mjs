import test from "node:test";
import assert from "node:assert/strict";
import { requiereRevisionBolsa, continuidadBolsaDisponible, renderizarResultadoBolsa } from "./resultado-bolsa.js";
import { crearTraductorExpedientesContratacion, cargarMensajesExpedientesContratacionEnIdioma } from "./i18n-expedientes.js";
import { renderizarLineaFases } from "./vista-expedientes-ficha.js";
import { validarReciboVinculoBolsa, validarSolicitudVinculoBolsa } from "./cliente-http-vinculo-bolsa.js";

const t = crearTraductorExpedientesContratacion();
function expediente(resultado = {}) {
  return { expediente_ref: "expediente:prueba", version: 4, resultado_bolsa: {
    vinculos: [], emisiones_vinculables: [], siguiente_cursor: null, total_vinculos: 0,
    personas_solicitadas: 2, aceptaciones_firmes: 0, ...resultado,
  } };
}
function vinculo(emitido, respuesta = null, modo = null, situacion = "disponible") {
  return { bolsa_ref: "bolsa:prueba", llamamiento_ref: `llamamiento:${"a".repeat(64)}`,
    recibo_emision_ref: `recibo:llamamiento:${"a".repeat(64)}`, emitido_en: emitido,
    participaciones: [{ respuesta, modo, situacion_actual: situacion, participacion_ref: "participacion:prueba",
      recibo_respuesta_ref: respuesta ? "recibo:respuesta:prueba" : null,
      respondida_en: respuesta ? "2026-10-09T12:00:00Z" : null,
      recibo_situacion_ref: "recibo:situacion:prueba", situacion_desde: "2026-10-09T12:01:00Z" }] };
}

test("el recuento histórico de aceptaciones no cambia la fase confirmada por el servidor", () => {
  const ficha = expediente({ aceptaciones_firmes: 2 });
  ficha.fases = [{ fase_ref: "fase:obtencion_candidato", estado_clave: "pendiente", orden: 4, etiqueta: "Candidato" }];
  assert.match(renderizarLineaFases(ficha, t), /class="falta"/u);
  ficha.fases[0].estado_clave = "completado";
  assert.match(renderizarLineaFases(ficha, t), /class="hecho"/u);
});

test("contacto y renuncia de situación no se presentan como aceptación ni respuesta formal", () => {
  const ficha = expediente({ vinculos: [vinculo("2026-10-09T10:00:00Z", null, null, "renuncia")] });
  const html = renderizarResultadoBolsa(ficha, t, "es-ES", "Europe/Madrid");
  assert.match(html, /Sin respuesta registrada/u);
  assert.match(html, /Renuncia registrada en Bolsa/u);
  assert.doesNotMatch(html, /Aceptación confirmada/u);
  assert.equal(continuidadBolsaDisponible(ficha), true);
});

test("la siguiente emisión pendiente no ofrece otra continuidad tras una renuncia anterior", () => {
  const ficha = expediente({ vinculos: [vinculo("2026-10-09T10:00:00Z", "renuncia", "firme"),
    vinculo("2026-10-09T10:00:00.000001Z")] });
  assert.equal(continuidadBolsaDisponible(ficha), false);
  assert.equal(continuidadBolsaDisponible(expediente({ vinculos: [vinculo("2026-10-09T10:00:00Z", "renuncia", "propuesta_rrhh")] })), false);
});

test("la respuesta pendiente de RRHH y la firme se distinguen en el idioma elegido", async () => {
  const mensajes = await cargarMensajesExpedientesContratacionEnIdioma("en");
  const en = crearTraductorExpedientesContratacion(mensajes);
  const ficha = expediente({ vinculos: [vinculo("2026-10-09T10:00:00Z", "acepta", "propuesta_rrhh")] });
  assert.match(renderizarResultadoBolsa(ficha, en, "en-GB", "Europe/Madrid"), /Response awaiting HR review/u);
  ficha.resultado_bolsa.vinculos[0].participaciones[0].modo = "firme";
  assert.match(renderizarResultadoBolsa(ficha, en, "en-GB", "Europe/Madrid"), /Confirmed acceptance/u);
});

test("un recibo de otro llamamiento no confirma el vínculo solicitado", () => {
  const s = validarSolicitudVinculoBolsa({ expediente_ref: "expediente:prueba", version_esperada: 4,
    bolsa_ref: "bolsa:prueba", llamamiento_ref: "llamamiento:prueba", recibo_emision_ref: "recibo:emision:prueba",
    clave_idempotencia: "clave-vinculo-01" });
  const recibo = { ...s, recibo_vinculo_ref: "recibo:vinculo:prueba", vinculado_en: "2026-10-09T10:00:00Z", reutilizado: false };
  assert.equal(validarReciboVinculoBolsa(recibo, s).reutilizado, false);
  assert.throws(() => validarReciboVinculoBolsa({ ...recibo, llamamiento_ref: "llamamiento:ajeno" }, s));
});

// El contacto telefónico todavía no es una respuesta formal del llamamiento.
test("muestra el contacto con fecha y justificante separado de la respuesta", async () => {
  const v = vinculo("2026-10-09T10:00:00Z");
  Object.assign(v.participaciones[0], { contacto_resultado: "acepta",
    contacto_en: "2026-10-09T11:00:00Z", recibo_contacto_ref: "recibo:contacto:prueba" });
  const ficha = expediente({ vinculos: [v] });
  const html = renderizarResultadoBolsa(ficha, t, "es-ES", "Europe/Madrid");
  assert.match(html, /Contacto registrado/u);
  assert.match(html, /Acepta durante el contacto/u);
  assert.match(html, /Justificante del contacto/u);
  assert.match(html, /recibo:contacto:prueba/u);
  assert.match(html, /Sin respuesta registrada/u);
  assert.doesNotMatch(html, /Aceptación confirmada/u);
  const en = crearTraductorExpedientesContratacion(await cargarMensajesExpedientesContratacionEnIdioma("en"));
  assert.match(renderizarResultadoBolsa(ficha, en, "en-GB", "Europe/Madrid"), /Accepts during contact/u);
});

test("una retirada posterior conserva la aceptación y ofrece revisión sin presumir renuncia formal", () => {
  const v = vinculo("2026-10-09T10:00:00Z", "acepta", "firme", "renuncia");
  const ficha = expediente({ vinculos: [v], personas_solicitadas: 1, aceptaciones_firmes: 1 });
  const html = renderizarResultadoBolsa(ficha, t, "es-ES", "Europe/Madrid");
  assert.match(html, /Aceptación confirmada/u);
  assert.match(html, /Renuncia registrada en Bolsa/u);
  assert.match(html, /recibo:respuesta:prueba/u);
  assert.match(html, /recibo:situacion:prueba/u);
  assert.equal(requiereRevisionBolsa(ficha), true);
  assert.equal(continuidadBolsaDisponible(ficha), false);
  v.participaciones[0].situacion_desde = "2026-10-09T11:59:00Z";
  assert.equal(requiereRevisionBolsa(ficha), false);
});
