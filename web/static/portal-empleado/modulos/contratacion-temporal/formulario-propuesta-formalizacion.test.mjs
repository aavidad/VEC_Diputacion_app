import assert from "node:assert/strict";
import test from "node:test";
import { renderizarResumenPropuestaFormalizacion } from "./formulario-propuesta-formalizacion.js";

const textos = new Map([
  ["llamamiento_propuesta_pantalla_sobrelinea", "Pantalla 13"],
  ["llamamiento_propuesta_pantalla_titulo", "Candidatura aceptada"],
  ["llamamiento_propuesta_pantalla_ayuda", "Prepare la propuesta."],
  ["llamamiento_propuesta_pantalla_limite", "Sin entrega."],
  ["llamamiento_resolucion_llamamiento_aceptada_ref", "Resolución"],
  ["llamamiento_recibo_resolucion_aceptada_ref", "Recibo"],
]);
test("la orientación solo aparece para una aceptación y no afirma entrega", () => {
  const html = renderizarResumenPropuestaFormalizacion({ respuesta: "aceptacion", resolucion_ref: "resolucion:<x>", recibo_local_ref: "recibo:&" }, (k) => textos.get(k));
  assert.match(html, /resolucion:&lt;x&gt;/);
  assert.match(html, /recibo:&amp;/);
  assert.doesNotMatch(html.toLowerCase(), /firma|entregad[oa]|destinatario/);
  assert.equal(renderizarResumenPropuestaFormalizacion({ respuesta: "renuncia" }, (k) => textos.get(k)), "");
});
