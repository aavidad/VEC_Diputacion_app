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

test("pausa máxima nula conserva Mi bolsa y las demás acciones sin ofrecer pausa", () => {
  const portal = [{ bolsa: "bolsa:demo:1", llamamiento_abierto: {
    contacto_en: "2026-09-25T08:00:00.000000Z", vence_antes_de: "2026-09-26T21:59:59.000000Z",
  }, solicitud_pendiente: null, ultima_respuesta: null }];
  const sinPausa = { ...acciones, pausa_maxima: null };
  assert.doesNotThrow(() => validarPortalMiBolsa({ participaciones, acciones_portal: sinPausa, portal }));
  assert.doesNotThrow(() => validarPortalMiBolsa({ participaciones,
    acciones_portal: { ...acciones, pausa_maxima: "2028-02-29T08:00:00Z" }, portal }));
  const html = renderizarPortalMiBolsa(participaciones, portal, sinPausa);
  assert.equal(html, renderizarPortalMiBolsa(participaciones, portal, acciones));
  assert.match(html, /data-portal-mi-bolsa="responder"/u);
  assert.match(html, /data-portal-mi-bolsa="documental"/u);
  assert.doesNotMatch(html, /data-tipo="pausa"|data-portal-mi-bolsa="solicitar"/u);
  for (const valor of [undefined, "", "sin fecha", "2026-02-30T08:00:00Z",
    "2025-02-29T08:00:00Z", "2026-10-01T24:00:00Z", 42, {}]) {
    assert.throws(() => validarPortalMiBolsa({ participaciones,
      acciones_portal: { ...acciones, pausa_maxima: valor }, portal }), /pausa_maxima/u);
  }
  for (const incompleta of [{ ...sinPausa, causas_renuncia: undefined },
    { ...sinPausa, modo_respuesta: undefined }]) {
    assert.throws(() => validarPortalMiBolsa({ participaciones, acciones_portal: incompleta, portal }), /no son válidas/u);
  }
  assert.throws(() => validarPortalMiBolsa({ participaciones, acciones_portal: sinPausa }), /no son válidas/u);
});

test("muestra la respuesta y admite solicitud pendiente desde toda participación propia", () => {
  const html = renderizarPortalMiBolsa(participaciones, [{ bolsa: "bolsa:demo:1", llamamiento_abierto: { contacto_en: "2026-09-25T08:00:00.000000Z", vence_antes_de: "2026-09-26T21:59:59.000000Z" } }], acciones);
  assert.doesNotMatch(html, /data-portal-mi-bolsa="solicitar"|data-tipo="pausa"|data-tipo="reactivacion"/u);
  assert.match(html, /data-portal-mi-bolsa="documental"/u);
  const revision = renderizarPortalMiBolsa([{ ...participaciones[0], situacion_actual: { estado: "en_revision" } }], [{ bolsa: "bolsa:demo:1" }], acciones);
  assert.match(revision, /data-portal-mi-bolsa="documental"/u);
  assert.match(revision, /Enviar solicitud a RRHH/u);
  const excluida = renderizarPortalMiBolsa([{ ...participaciones[0], situacion_actual: { estado: "excluido" } }], [{ bolsa: "bolsa:demo:1" }], acciones);
  assert.match(excluida, /data-portal-mi-bolsa="documental"/u);
  const cerrada = renderizarPortalMiBolsa([{ ...participaciones[0], vigente_hasta: "2026-10-01T00:00:00Z" }], [{ bolsa: "bolsa:demo:1" }], acciones);
  assert.doesNotMatch(cerrada, /data-portal-mi-bolsa="documental"/u);
  assert.match(html, /data-portal-mi-bolsa="responder"/u);
  assert.match(html, /Matrimonio o unión de hecho/u);
  assert.match(html, /Respuesta firme/u);
  const pendiente = renderizarPortalMiBolsa(participaciones, [{ bolsa: "bolsa:demo:1", solicitud_pendiente: { tipo: "pausa", recibo: "recibo:solicitud-portal:x", registrada_en: "2026-09-25T08:00:00.000000Z" } }], acciones);
  assert.match(pendiente, /pendiente de RRHH/u);
  assert.doesNotMatch(pendiente, /data-tipo="pausa"/u);
  assert.doesNotMatch(pendiente, /data-portal-mi-bolsa="documental"/u);
});

test("la solicitud documental envía referencia, huella y fin de causa con clave estable", async () => {
  const formulario = { dataset: { portalMiBolsa: "documental", bolsa: "bolsa:demo:1" } };
  const datos = new FormData();
  datos.set("documento_ref", "documento:parte-1");
  datos.set("documento", new Blob(["abc"]));
  datos.set("fecha_fin_causa", "2026-10-02");
  const primera = await cuerpoPortalMiBolsa(formulario, datos);
  assert.equal(primera.ruta, "/api/vec/bolsa/mi-bolsa/solicitudes-documentales");
  assert.deepEqual(Object.keys(primera.cuerpo).sort(), ["bolsa", "clave", "documento_ref", "documento_sha256", "fecha_fin_causa", "tipo"]);
  assert.equal(primera.cuerpo.documento_sha256, "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad");
  assert.equal((await cuerpoPortalMiBolsa(formulario, datos)).cuerpo.clave, primera.cuerpo.clave);
  datos.set("fecha_fin_causa", "2026-02-30");
  assert.equal(await cuerpoPortalMiBolsa(formulario, datos), null);
  datos.set("fecha_fin_causa", "2026-10-02");
  datos.set("documento_ref", "dni:prueba");
  assert.equal(await cuerpoPortalMiBolsa(formulario, datos), null);
  datos.set("documento_ref", "documento:parte-1");
  datos.delete("fecha_fin_causa");
  const sinFecha = await cuerpoPortalMiBolsa(formulario, datos);
  assert.equal(Object.hasOwn(sinFecha.cuerpo, "fecha_fin_causa"), false);
  assert.equal(sinFecha.cuerpo.clave, primera.cuerpo.clave);
});

test("referencia con identidad se corrige antes de enviar y no deja solicitud pendiente", async () => {
  const zona = { textContent: "" };
  const campo = { mensaje: "", foco: false, setCustomValidity(valor) { this.mensaje = valor; }, reportValidity() { return false; }, focus() { this.foco = true; }, addEventListener() {} };
  const formulario = { dataset: { portalMiBolsa: "documental", bolsa: "bolsa:demo:1" },
    querySelector: (selector) => selector === "[data-portal-resultado]" ? zona : selector === '[name="documento_ref"]' ? campo : null };
  const datos = new FormData();
  datos.set("documento_ref", "dni:prueba"); datos.set("documento", new Blob(["abc"]));
  let envios = 0;
  assert.equal(await enviarPortalMiBolsa(formulario, { datos, fetchImpl: async () => { envios++; } }), false);
  assert.equal(envios, 0);
  assert.match(zona.textContent, /sin espacios ni datos de identidad/u);
  assert.equal(campo.mensaje, zona.textContent);
  assert.equal(campo.foco, true);
});

test("la consulta conserva recibo y resolución sin permitir duplicar una pendiente", () => {
  const estado = { bolsa: "bolsa:demo:1", solicitud_documental_pendiente: true,
    ultima_solicitud_documental: { tipo: "documental_rrhh", recibo: `recibo:solicitud-documental:${"a".repeat(64)}`,
      registrada_en: "2026-10-02T10:00:00.000000Z", estado: "pendiente_rrhh", recibo_resolucion_ref: null } };
  const datos = { participaciones: [{ ...participaciones[0], situacion_actual: { estado: "en_revision" } }], acciones_portal: acciones, portal: [estado] };
  assert.doesNotThrow(() => validarPortalMiBolsa(datos));
  const html = renderizarPortalMiBolsa(datos.participaciones, datos.portal, acciones);
  assert.match(html, /Pendiente de RRHH/u);
  assert.doesNotMatch(html, /data-portal-mi-bolsa="documental"/u);
  estado.solicitud_documental_pendiente = false;
  estado.ultima_solicitud_documental.estado = "rechazada";
  estado.ultima_solicitud_documental.recibo_resolucion_ref = `recibo:solicitud-documental-resolucion:${"b".repeat(64)}`;
  assert.doesNotThrow(() => validarPortalMiBolsa(datos));
  assert.match(renderizarPortalMiBolsa(datos.participaciones, datos.portal, acciones), /data-portal-mi-bolsa="documental"/u);
});

test("solo un recibo documental completo confirma el envío pendiente", async () => {
  const zona = { textContent: "" };
  const formulario = { dataset: { portalMiBolsa: "documental", bolsa: "bolsa:demo:1" }, querySelector: (s) => s === "[data-portal-resultado]" ? zona : null };
  const datos = new FormData();
  datos.set("documento_ref", "documento:parte-1"); datos.set("documento", new Blob(["abc"])); datos.set("fecha_fin_causa", "2026-10-02");
  const recibo = { esquema: "vec.bolsa.mi-bolsa.solicitud-documental.v1", referencia: `solicitud-documental:${"a".repeat(64)}`,
    recibo: `recibo:solicitud-documental:${"b".repeat(64)}`, contenido_sha256: "c".repeat(64),
    registrada_en: "2026-10-02T10:00:00.000000Z", version: 1, estado: "pendiente_rrhh", repetida: false };
  let recargas = 0;
  assert.equal(await enviarPortalMiBolsa(formulario, { datos, alRegistrar: () => recargas++,
    fetchImpl: async () => ({ status: 201, json: async () => ({ data: { ...recibo, contenido_sha256: "" } }) }) }), false);
  assert.equal(recargas, 0);
  assert.equal(await enviarPortalMiBolsa(formulario, { datos, alRegistrar: () => recargas++,
    fetchImpl: async () => ({ status: 200, json: async () => ({ data: { ...recibo, repetida: true } }) }) }), true);
  assert.equal(recargas, 1);
  assert.match(zona.textContent, /Solicitud pendiente de RRHH/u);
});

test("tras un resultado incierto inmoviliza los datos y reintenta el mismo cuerpo", async () => {
  const zona = { textContent: "" };
  const campos = [{ disabled: false }, { disabled: false }, { disabled: false }];
  const formulario = { dataset: { portalMiBolsa: "documental", bolsa: "bolsa:demo:1" },
    querySelector: (s) => s === "[data-portal-resultado]" ? zona : null,
    querySelectorAll: () => campos };
  const datos = new FormData();
  datos.set("documento_ref", "documento:parte-1"); datos.set("documento", new Blob(["abc"])); datos.set("fecha_fin_causa", "2026-10-02");
  const cuerpos = [];
  assert.equal(await enviarPortalMiBolsa(formulario, { datos, fetchImpl: async (_ruta, opciones) => {
    cuerpos.push(opciones.body); throw new TypeError("respuesta perdida");
  } }), false);
  assert.ok(campos.every((campo) => campo.disabled));
  assert.match(zona.textContent, /Reintentar conserva los datos originales/u);
  datos.set("fecha_fin_causa", "2026-10-01");
  const recibo = { esquema: "vec.bolsa.mi-bolsa.solicitud-documental.v1", referencia: `solicitud-documental:${"a".repeat(64)}`,
    recibo: `recibo:solicitud-documental:${"b".repeat(64)}`, contenido_sha256: "c".repeat(64),
    registrada_en: "2026-10-02T10:00:00.000000Z", version: 1, estado: "pendiente_rrhh", repetida: true };
  assert.equal(await enviarPortalMiBolsa(formulario, { datos, fetchImpl: async (_ruta, opciones) => {
    cuerpos.push(opciones.body); return { status: 200, json: async () => ({ data: recibo }) };
  } }), true);
  assert.equal(cuerpos[1], cuerpos[0]);
  assert.ok(campos.every((campo) => !campo.disabled));
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

test("ningún formulario heredado de pausa o reactivación envía una petición", async () => {
  const zona = { textContent: "" };
  const formulario = { dataset: { portalMiBolsa: "solicitar", tipo: "reactivacion", bolsa: "bolsa:demo:1" }, querySelector: (s) => (s === "[data-portal-resultado]" ? zona : null) };
  let envios = 0;
  assert.equal(await enviarPortalMiBolsa(formulario, { fetchImpl: async () => { envios++; } }), false);
  assert.equal(envios, 0);
  assert.equal(await cuerpoPortalMiBolsa({ dataset: { portalMiBolsa: "solicitar", tipo: "pausa", bolsa: "bolsa:demo:1" } }, new FormData()), null);
  assert.equal(await cuerpoPortalMiBolsa({ dataset: { portalMiBolsa: "desconocida", bolsa: "bolsa:demo:1" } }, new FormData()), null);
});
