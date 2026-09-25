import assert from "node:assert/strict";
import test from "node:test";
import { cuerpoPortalMiBolsa, enviarPortalMiBolsa, validarPortalMiBolsa } from "./mi-bolsa-portal.js";
import { renderizarOfertasMiBolsa, validarOfertasMiBolsa } from "./mi-bolsa-ofertas.js";

const participaciones = [{ bolsa: "bolsa:demo:1", categoria: "Auxiliar de enfermería" }];
const oferta = (extra = {}) => ({
  oferta: `oferta:${"a".repeat(64)}`, bolsa: "bolsa:demo:1", categoria: "Auxiliar de enfermería", centro: "Residencia Sierra",
  fecha_inicio: "2026-10-01", fecha_fin: null, descripcion: "Sustitución por baja", publicada_en: "2026-09-25T08:00:00.000000Z",
  vence_antes_de: "2026-09-27T21:59:59.000000Z", estado: "abierta", disposicion: null, ...extra,
});

test("sin ofertas compuestas no se exige nada y con ellas se validan todas", () => {
  assert.doesNotThrow(() => validarOfertasMiBolsa({ participaciones }));
  assert.doesNotThrow(() => validarPortalMiBolsa({ participaciones, ofertas: [oferta()] }));
  assert.throws(() => validarPortalMiBolsa({ participaciones, ofertas: [oferta({ bolsa: "bolsa:ajena" })] }), /ajena/u);
  assert.throws(() => validarOfertasMiBolsa({ participaciones, ofertas: [oferta({ estado: "adjudicada" })] }), /ajena/u);
  assert.throws(() => validarOfertasMiBolsa({ participaciones, ofertas: [oferta({ fecha_fin: "31/12/2026" })] }), /Fechas/u);
  assert.throws(() => validarOfertasMiBolsa({ participaciones, ofertas: [oferta({ disposicion: { recibo: "r", manifestada_en: "ayer" } })] }), /instante/u);
});

test("ofrece el botón solo en ofertas abiertas sin disposición", () => {
  const abierta = renderizarOfertasMiBolsa([oferta()]);
  assert.match(abierta, /data-portal-mi-bolsa="disposicion"/u);
  assert.match(abierta, /Me ofrezco/u);
  assert.match(abierta, /Sin fecha de fin/u);
  const hecha = renderizarOfertasMiBolsa([oferta({ disposicion: { recibo: "recibo:disposicion:x", manifestada_en: "2026-09-25T09:00:00.000000Z" } })]);
  assert.doesNotMatch(hecha, /data-portal-mi-bolsa/u);
  assert.match(hecha, /recibo:disposicion:x/u);
  const adjudicada = renderizarOfertasMiBolsa([oferta({ estado: "adjudicada_propia", disposicion: { recibo: "r", manifestada_en: "2026-09-25T09:00:00.000000Z" } })]);
  assert.match(adjudicada, /Adjudicada a usted/u);
  assert.equal(renderizarOfertasMiBolsa([]), "");
  assert.doesNotMatch(renderizarOfertasMiBolsa([oferta({ descripcion: "<b>x</b>" })]), /<b>x/u);
});

test("envía solo la oferta y una clave estable, y explica los conflictos", async () => {
  const formulario = { dataset: { portalMiBolsa: "disposicion", oferta: `oferta:${"a".repeat(64)}` }, querySelector: () => null };
  const primera = await cuerpoPortalMiBolsa(formulario, new FormData());
  const segunda = await cuerpoPortalMiBolsa(formulario, new FormData());
  assert.equal(primera.ruta, "/api/vec/bolsa/mi-bolsa/disposiciones");
  assert.deepEqual(Object.keys(primera.cuerpo).sort(), ["clave", "oferta"]);
  assert.equal(primera.cuerpo.clave, segunda.cuerpo.clave);
  assert.equal(await cuerpoPortalMiBolsa({ dataset: { portalMiBolsa: "disposicion", oferta: "oferta:x" } }, new FormData()), null);
  const zona = { textContent: "" };
  const conZona = { ...formulario, querySelector: (s) => (s === "[data-portal-resultado]" ? zona : null) };
  const fetchImpl = async () => ({ status: 409, json: async () => ({ error: { codigo: "oferta_no_abierta" } }) });
  assert.equal(await enviarPortalMiBolsa(conZona, { fetchImpl, datos: new FormData() }), false);
  assert.match(zona.textContent, /ya no está abierta/u);
});
