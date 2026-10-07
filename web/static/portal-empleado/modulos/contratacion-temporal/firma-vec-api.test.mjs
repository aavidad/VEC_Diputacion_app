import assert from "node:assert/strict";
import test from "node:test";
import { createHash } from "node:crypto";
import { crearClienteFirmaVec, RUTA_REGISTRO_FIRMA_VEC } from "./firma-vec-api.js";

const pdf = new TextEncoder().encode("%PDF-1.7\nprimera firma conservada\nsegunda firma\n%%EOF");
const solicitud = { expedienteRef: "expediente:prueba", version: 7, documento: "resolucion", pasoOrden: 2,
  originalRef: "original:prueba", originalVersion: 1, firmado: pdf, clave: "clave-prueba-123456789" };
function recibo(replay = false) {
  return { esquema: "vec.contratacion-temporal.registro-firma-vec.v2", recibo_ref: "recibo:prueba", firma_ref: "firma:prueba",
    ya_registrada: replay, expediente_ref: "expediente:prueba", version_expediente: 7, documento: "resolucion", paso_orden: 2,
    paso_ref: "paso:prueba", secuencia: 2, registrada_en: "2026-10-03T10:00:00Z", firma_eficaz: false,
    documento_custodiado: { expediente_ref: `ref:${"e".repeat(64)}`, documento_ref: `ref:${"d".repeat(64)}`, version: 2, huella_sha256: createHash("sha256").update(pdf).digest("hex") },
    verificacion_tecnica: { estado: "valida", motivo: "verificada", politica: "politica:prueba", revocacion: "vigente", sello_tiempo: "no_presente",
      original_sha256: "a".repeat(64), firmado_sha256: createHash("sha256").update(pdf).digest("hex") }, material_root_sha256: "c".repeat(64),
    revision_pdf: { orden_firma: 2, entrada_sha256: "d".repeat(64), revision_sha256: createHash("sha256").update(pdf).digest("hex"), evidencia_sha256: "f".repeat(64) } };
}
const respuesta = (data, status = 201) => new Response(JSON.stringify({ data }), { status, headers: { "Content-Type": "application/json" } });
test("VEC V2 envía sólo ocho campos y el PDF intacto, sin original ni actor", async () => {
  let captura;
  const cliente = crearClienteFirmaVec({ fetchImpl: async (ruta, opciones) => { captura = { ruta, opciones }; return respuesta(recibo()); } });
  await cliente.registrar({ ...solicitud, original: pdf, actor: "descartado" });
  assert.equal(captura.ruta, RUTA_REGISTRO_FIRMA_VEC);
  const cuerpo = JSON.parse(captura.opciones.body);
  assert.deepEqual(Object.keys(cuerpo).sort(), ["expediente_ref", "version_expediente", "documento", "paso_orden", "original_ref", "original_version", "firmado_base64", "clave_idempotencia"].sort());
  assert.deepEqual(new Uint8Array(Buffer.from(cuerpo.firmado_base64, "base64")), pdf);
  assert.equal(captura.opciones.credentials, "same-origin"); assert.equal(captura.opciones.cache, "no-store");
  assert.equal(captura.opciones.redirect, "error"); assert.equal(captura.opciones.referrerPolicy, "no-referrer");
});
test("recibo ajeno, revisión rota o eficacia administrativa nunca confirma", async () => {
  for (const alterar of [(d) => { d.expediente_ref = "expediente:otro"; }, (d) => { d.documento = "diligencia"; },
    (d) => { d.firma_eficaz = true; }, (d) => { d.revision_pdf.orden_firma = 1; },
    (d) => { d.revision_pdf.revision_sha256 = "e".repeat(64); }, (d) => { d.documento_custodiado.huella_sha256 = "e".repeat(64); },
    (d) => { d.verificacion_tecnica.revocacion = "desconocida"; }, (d) => { d.actor = "ajeno"; }]) {
    const d = recibo(); alterar(d);
    await assert.rejects(crearClienteFirmaVec({ fetchImpl: async () => respuesta(d) }).registrar(solicitud), (e) => e.codigo === "resultado_no_confiable");
  }
});
test("replay exige 200 y conserva recibo; 201 no acepta ya_registrada", async () => {
  const result = await crearClienteFirmaVec({ fetchImpl: async () => respuesta(recibo(true), 200) }).registrar(solicitud);
  assert.equal(result.recibo_ref, "recibo:prueba");
  await assert.rejects(crearClienteFirmaVec({ fetchImpl: async () => respuesta(recibo(true), 201) }).registrar(solicitud), (e) => e.codigo === "resultado_no_confiable");
});
test("un recibo coherente de otro PDF no confirma los bytes enviados", async () => {
  const d = recibo(); const otra = "e".repeat(64);
  d.documento_custodiado.huella_sha256 = otra;
  d.verificacion_tecnica.firmado_sha256 = otra;
  d.revision_pdf.revision_sha256 = otra;
  await assert.rejects(crearClienteFirmaVec({ fetchImpl: async () => respuesta(d) }).registrar(solicitud),
    (e) => e.codigo === "resultado_no_confiable");
});
test("fallos de red, denegación y cancelación no devuelven recibo", async () => {
  for (const [status, codigo] of [[403, "acceso_denegado"], [503, "servicio_no_disponible"]]) {
    await assert.rejects(crearClienteFirmaVec({ fetchImpl: async () => respuesta({}, status) }).registrar(solicitud), (e) => e.codigo === codigo);
  }
  let peticiones = 0; const c = new AbortController(); c.abort();
  await assert.rejects(crearClienteFirmaVec({ fetchImpl: async () => { peticiones++; } }).registrar(solicitud, { signal: c.signal }), (e) => e.codigo === "operacion_abortada");
  assert.equal(peticiones, 0);
});

test("el recibo conserva las huellas del original raíz y de la entrada preparada", async () => {
  const preparada = { ...solicitud, originalHuella: "a".repeat(64), revisionEntradaHuella: "d".repeat(64) };
  await crearClienteFirmaVec({ fetchImpl: async () => respuesta(recibo()) }).registrar(preparada);
  for (const campo of ["originalHuella", "revisionEntradaHuella"]) {
    await assert.rejects(crearClienteFirmaVec({ fetchImpl: async () => respuesta(recibo()) }).registrar({ ...preparada, [campo]: "b".repeat(64) }),
      (e) => e.codigo === "resultado_no_confiable");
  }
});
