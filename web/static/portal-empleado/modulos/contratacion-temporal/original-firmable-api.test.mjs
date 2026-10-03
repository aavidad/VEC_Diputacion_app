import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import test from "node:test";
import { crearClienteOriginalFirmable, RUTA_ORIGINAL_FIRMABLE } from "./original-firmable-api.js";

const pdf = new TextEncoder().encode("%PDF-1.7\noriginal custodiado\n%%EOF");
const huella = createHash("sha256").update(pdf).digest("hex");
const solicitud = { expedienteRef: "expediente:prueba", version: 7, documento: "resolucion" };
const doc = `ref:${"d".repeat(64)}`;
function datos() {
  return { esquema: "vec.contratacion-temporal.original-firmable.v1", expediente_ref: solicitud.expedienteRef,
    version_observada: 7, documento: "resolucion", original_ref: doc, original_version: 7,
    original_sha256: huella, tipo_ref: "tipo:resolucion_original", mime: "application/pdf",
    pdf_base64: Buffer.from(pdf).toString("base64"), documento_custodiado: {
      expediente_ref: `ref:${"e".repeat(64)}`, documento_ref: doc, version: 7, huella_sha256: huella,
    } };
}
const respuesta = (data, status = 200) => new Response(JSON.stringify({ data }), {
  status, headers: { "Content-Type": "application/json; charset=utf-8" },
});
test("el original se recupera por referencia y versión, con sus bytes y huella exactos", async () => {
  let enviada;
  const cliente = crearClienteOriginalFirmable({ fetchImpl: async (ruta, opciones) => {
    enviada = { ruta, opciones }; return respuesta(datos());
  } });
  const original = await cliente.preparar({ ...solicitud, actor: "descartado", bytes: "descartados" });
  assert.deepEqual(original.contenido, pdf);
  assert.equal(original.originalRef, doc); assert.equal(original.originalHuella, huella);
  assert.equal(enviada.ruta, RUTA_ORIGINAL_FIRMABLE);
  assert.deepEqual(JSON.parse(enviada.opciones.body), { expediente_ref: solicitud.expedienteRef, version_observada: 7, documento: "resolucion" });
  assert.equal(enviada.opciones.credentials, "same-origin"); assert.equal(enviada.opciones.cache, "no-store");
  assert.equal(enviada.opciones.redirect, "error"); assert.equal(enviada.opciones.referrerPolicy, "no-referrer");
});
test("otra versión, otro PDF o custodia ajena no entrega bytes para firmar", async () => {
  for (const alterar of [(d) => { d.original_version++; }, (d) => { d.expediente_ref = "expediente:otro"; },
    (d) => { d.pdf_base64 = Buffer.from("%PDF-1.7\najeno\n%%EOF").toString("base64"); },
    (d) => { d.documento_custodiado.documento_ref = `ref:${"a".repeat(64)}`; },
    (d) => { d.pdf_base64 += "\n"; }, (d) => { d.actor = "ajeno"; }]) {
    const d = datos(); alterar(d);
    await assert.rejects(crearClienteOriginalFirmable({ fetchImpl: async () => respuesta(d) }).preparar(solicitud),
      (e) => e.codigo === "resultado_no_confiable");
  }
});
test("denegación, límite de respuesta y cancelación cierran la descarga", async () => {
  await assert.rejects(crearClienteOriginalFirmable({ fetchImpl: async () => respuesta({}, 403) }).preparar(solicitud),
    (e) => e.codigo === "acceso_denegado");
  const grande = respuesta(datos()); grande.headers.set("Content-Length", "2000000");
  await assert.rejects(crearClienteOriginalFirmable({ fetchImpl: async () => grande }).preparar(solicitud),
    (e) => e.codigo === "resultado_no_confiable");
  const c = new AbortController(); let llamadas = 0; c.abort();
  await assert.rejects(crearClienteOriginalFirmable({ fetchImpl: async () => { llamadas++; } }).preparar(solicitud, { signal: c.signal }),
    (e) => e.codigo === "operacion_abortada");
  assert.equal(llamadas, 0);
});
