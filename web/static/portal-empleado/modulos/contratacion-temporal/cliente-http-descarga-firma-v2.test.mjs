import assert from "node:assert/strict";
import test from "node:test";
import { createHash, webcrypto } from "node:crypto";
import { cargarTextos } from "../../../comun/textos.js";
import { crearClienteDescargaFirmaV2, RUTA_DESCARGA_FIRMA_V2 } from "./cliente-http-descarga-firma-v2.js";
import { renderizarFaseFirma } from "./fase-firma.js";

const bytes = new TextEncoder().encode("%PDF-1.7\n1 0 obj\n<<>>\nendobj\n%%EOF\n");
const huella = createHash("sha256").update(bytes).digest("hex");
const solicitud = Object.freeze({ expedienteRef: "expediente:ct:" + "a".repeat(64), version: 7,
  documento: "resolucion", firmaRef: "firma:nominal:paso2", custodiaRef: "ref:" + "b".repeat(64),
  custodiaVersion: 2, huellaEsperada: huella });
const datos = Object.freeze({ esquema: "vec.contratacion-temporal.descarga-firma-r5.v2",
  pdf_base64: Buffer.from(bytes).toString("base64"), expediente_ref: solicitud.expedienteRef,
  version_observada: 7, documento: "resolucion", firma_ref: solicitud.firmaRef,
  recibo_ref: "recibo:firma:historica", documento_ref: solicitud.custodiaRef,
  documento_version: 2, huella_sha256: huella });
const json = (data, status = 200) => new Response(JSON.stringify({ data }), {
  status, headers: { "Content-Type": "application/json; charset=utf-8" },
});

test("descarga nominal envía solo la selección CT y entrega los mismos bytes tras cotejar SHA-256", async () => {
  let llamada;
  const cliente = crearClienteDescargaFirmaV2({ cryptoImpl: webcrypto, fetchImpl: async (ruta, opciones) => {
    llamada = { ruta, opciones }; return json(datos);
  } });
  const archivo = await cliente.descargar(solicitud);
  assert.equal(llamada.ruta, RUTA_DESCARGA_FIRMA_V2);
  assert.equal(llamada.opciones.method, "POST");
  assert.equal(llamada.opciones.credentials, "same-origin");
  assert.deepEqual(JSON.parse(llamada.opciones.body), { expediente_ref: solicitud.expedienteRef,
    version_observada: 7, documento: "resolucion", firma_ref: solicitud.firmaRef });
  assert.deepEqual(archivo.contenido, bytes);
  assert.equal(archivo.tipo, "application/pdf");
  assert.equal(archivo.nombre, "resolucion.pdf");
  assert.equal(archivo.reciboRef, datos.recibo_ref, "el recibo devuelto es el histórico");
});

test("bloquea PDF alterado y metadatos ajenos antes de entregar un archivo", async () => {
  for (const cambiado of [
    { ...datos, pdf_base64: Buffer.from("%PDF-1.7\nCAMBIO\n%%EOF").toString("base64") },
    { ...datos, firma_ref: "firma:ajena" },
    { ...datos, documento_ref: "ref:" + "c".repeat(64) },
    { ...datos, documento_version: 3 },
    { ...datos, huella_sha256: "0".repeat(64) },
    { ...datos, recibo_ref: "" },
  ]) {
    const cliente = crearClienteDescargaFirmaV2({ cryptoImpl: webcrypto, fetchImpl: async () => json(cambiado) });
    await assert.rejects(cliente.descargar(solicitud), { codigo: "resultado_no_confiable" });
  }
});

test("rechazo y cancelación no recurren a Documentos", async () => {
  const cliente = crearClienteDescargaFirmaV2({ cryptoImpl: webcrypto, fetchImpl: async () => new Response(null, { status: 403 }) });
  await assert.rejects(cliente.descargar(solicitud), { codigo: "denegado" });
  const abortado = new AbortController(); abortado.abort();
  await assert.rejects(cliente.descargar(solicitud, { signal: abortado.signal }), { codigo: "operacion_abortada" });
});

test("el botón nominal usa la garantía R5 explícita y omite la terna del cliente general", async () => {
  const textos = await cargarTextos("contratacion-temporal-firma", { idioma: "es", porDefecto: "es" });
  const documento = { documento: "resolucion", etiqueta: "Resolución", descarga_nominal: true, paso_pendiente: 0,
    pasos: [{ orden: 1, cargo: "Jefatura", estado: "firmado", registrada_en: "2026-10-14T10:00:00Z",
      firma_ref: solicitud.firmaRef, documento_custodiado: { expediente_ref: "ref:" + "d".repeat(64),
        documento_ref: solicitud.custodiaRef, version: 2, huella_sha256: huella } }] };
  const real = { documentos: [documento], registro: { verificacion: true } };
  const html = renderizarFaseFirma({ catalogo: real, real, textos });
  assert.match(html, /data-ct-firmado-modo="nominal"/u);
  assert.match(html, /data-ct-firmado-tipo="resolucion"/u);
  assert.match(html, /data-ct-firmado-firma-ref="firma:nominal:paso2"/u);
  assert.doesNotMatch(html, /data-ct-firmado-expediente=/u);
  documento.pasos[0].firma_ref = "";
  assert.doesNotMatch(renderizarFaseFirma({ catalogo: real, real, textos }), /data-ct-descargar-firmado/u);
});
