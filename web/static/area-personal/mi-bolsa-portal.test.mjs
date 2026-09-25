import assert from "node:assert/strict";
import test from "node:test";
import { cuerpoPortalMiBolsa, enviarPortalMiBolsa, renderizarPortalMiBolsa, validarPortalMiBolsa } from "./mi-bolsa-portal.js";

const acciones = { causas_renuncia: ["enfermedad", "matrimonio_union_hecho"], pausa_maxima: "2027-09-25T21:59:59.000000Z", modo_respuesta: "firme" };
const participaciones = [{ bolsa: "bolsa:demo:1", categoria: "Auxiliar de enfermería" }];

test("sin acciones compuestas no se exige nada y con ellas se valida todo", () => {
  assert.doesNotThrow(() => validarPortalMiBolsa({ participaciones }));
  const datos = { participaciones, acciones_portal: acciones, portal: [{ bolsa: "bolsa:demo:1", llamamiento_abierto: { contacto_en: "2026-09-25T08:00:00.000000Z", vence_antes_de: "2026-09-26T21:59:59.000000Z" }, solicitud_pendiente: null, ultima_respuesta: null }] };
  assert.doesNotThrow(() => validarPortalMiBolsa(datos));
  assert.throws(() => validarPortalMiBolsa({ ...datos, portal: [{ bolsa: "bolsa:ajena" }] }), /ajena/u);
  assert.throws(() => validarPortalMiBolsa({ ...datos, acciones_portal: { ...acciones, modo_respuesta: "automatico" } }), /no son válidas/u);
});

test("pinta pausa, reactivación y la respuesta con su plazo y causas del catálogo", () => {
  const html = renderizarPortalMiBolsa(participaciones, [{ bolsa: "bolsa:demo:1", llamamiento_abierto: { contacto_en: "2026-09-25T08:00:00.000000Z", vence_antes_de: "2026-09-26T21:59:59.000000Z" } }], acciones);
  assert.match(html, /data-portal-mi-bolsa="solicitar" data-tipo="pausa"/u);
  assert.match(html, /max="2027-09-25"/u);
  assert.match(html, /data-portal-mi-bolsa="responder"/u);
  assert.match(html, /Matrimonio o unión de hecho/u);
  assert.match(html, /Respuesta firme/u);
  const pendiente = renderizarPortalMiBolsa(participaciones, [{ bolsa: "bolsa:demo:1", solicitud_pendiente: { tipo: "pausa", recibo: "recibo:solicitud-portal:x", registrada_en: "2026-09-25T08:00:00.000000Z" } }], acciones);
  assert.match(pendiente, /pendiente de RRHH/u);
  assert.doesNotMatch(pendiente, /data-tipo="pausa"/u);
});

test("la renuncia justificada exige causa, referencia y justificante, y envía su huella", async () => {
  const formulario = { dataset: { portalMiBolsa: "responder", bolsa: "bolsa:demo:1" } };
  const incompleta = new FormData();
  incompleta.set("respuesta", "renuncia_justificada");
  assert.equal(await cuerpoPortalMiBolsa(formulario, incompleta), null);
  const datos = new FormData();
  datos.set("respuesta", "renuncia_justificada");
  datos.set("causa", "enfermedad");
  datos.set("justificante_ref", "parte-medico-1");
  datos.set("justificante", new Blob(["abc"]));
  const { ruta, cuerpo } = await cuerpoPortalMiBolsa(formulario, datos);
  assert.equal(ruta, "/api/vec/bolsa/mi-bolsa/respuestas");
  assert.equal(cuerpo.justificante_sha256, "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad");
  const repetida = await cuerpoPortalMiBolsa(formulario, datos);
  assert.equal(repetida.cuerpo.clave, cuerpo.clave, "un reintento repite la clave");
});

test("envía la solicitud y explica los conflictos del servidor", async () => {
  const zona = { textContent: "" };
  const formulario = { dataset: { portalMiBolsa: "solicitar", tipo: "reactivacion", bolsa: "bolsa:demo:1" }, querySelector: (s) => (s === "[data-portal-resultado]" ? zona : null) };
  let peticion;
  const conflicto = async (ruta, opciones) => { peticion = { ruta, opciones }; return { status: 409, json: async () => ({ error: { codigo: "situacion_no_admite" } }) }; };
  assert.equal(await enviarPortalMiBolsa(formulario, { fetchImpl: conflicto, datos: new FormData() }), false);
  assert.match(zona.textContent, /no admite/u);
  assert.equal(peticion.opciones.headers["Content-Type"], "application/json");
  let recargado = false;
  const correcto = async () => ({ status: 201, json: async () => ({ data: { recibo: "recibo:solicitud-portal:z" } }) });
  assert.equal(await enviarPortalMiBolsa(formulario, { fetchImpl: correcto, datos: new FormData(), alRegistrar: () => { recargado = true; } }), true);
  assert.match(zona.textContent, /recibo:solicitud-portal:z/u);
  assert.ok(recargado);
});
