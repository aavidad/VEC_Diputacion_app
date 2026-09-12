import assert from "node:assert/strict";
import test from "node:test";
import { crearClienteHTTPBorradorRRHH } from "./cliente-http-informe-definitivo.js";

const solicitud = Object.freeze({ expediente_ref: "expediente:ct:sintetico-009", version_observada: 7 });
const mime = "application/vnd.openxmlformats-officedocument.wordprocessingml.document";
const documentos = Object.freeze([
  ["informe_definitivo", "informe-definitivo-desarrollo", "informe-definitivo-borrador.docx"],
  ["resolucion", "resolucion-desarrollo", "resolucion-borrador.docx"],
  ["diligencia", "diligencia-desarrollo", "diligencia-borrador.docx"],
  ["toma_posesion", "toma-posesion-desarrollo", "toma-posesion-borrador.docx"],
  ["notificacion", "notificacion-desarrollo", "notificacion-borrador.docx"],
  ["comunicacion_centro", "comunicacion-centro-desarrollo", "comunicacion-centro-borrador.docx"],
]);
const docx = new Uint8Array([0x50, 0x4b, 0x03, 0x04, 0x14, 0, 0, 0]);

test("seis DOCX reutilizan la consulta autorizada y conservan bytes y nombre", async () => {
  for (const [tipo, selector, nombre] of documentos) {
    const llamadas = [];
    const cliente = crearClienteHTTPBorradorRRHH({ fetchImpl: async (ruta, opciones) => {
      llamadas.push({ ruta, opciones });
      return new Response(docx, { headers: {
        "Content-Type": mime,
        "Content-Disposition": `attachment; filename="${nombre}"`,
        "Content-Length": String(docx.byteLength),
      } });
    } });
    const descarga = await cliente.descargarBorrador(solicitud, { tipo, formato: "docx" });
    assert.equal(descarga.type, mime);
    assert.deepEqual(new Uint8Array(await descarga.arrayBuffer()), docx);
    assert.equal(llamadas.length, 1);
    assert.equal(llamadas[0].ruta, "/api/vec/contratacion-temporal/expedientes/consultas");
    assert.equal(llamadas[0].opciones.method, "POST");
    assert.equal(llamadas[0].opciones.body, JSON.stringify(solicitud));
    assert.deepEqual(llamadas[0].opciones.headers, {
      "Content-Type": "application/json", Accept: `${mime}; documento=${selector}`,
    });
  }
});

test("DOCX rechaza MIME, nombre o bytes que no son un contenedor Office", async () => {
  const cliente = (respuesta) => crearClienteHTTPBorradorRRHH({ fetchImpl: async () => respuesta });
  const cabeceras = { "Content-Type": mime, "Content-Disposition": 'attachment; filename="informe-definitivo-borrador.docx"' };
  for (const respuesta of [
    new Response(docx, { headers: { ...cabeceras, "Content-Type": "application/pdf" } }),
    new Response(docx, { headers: { ...cabeceras, "Content-Disposition": 'attachment; filename="otro.docx"' } }),
    new Response("<html>error</html>", { headers: cabeceras }),
  ]) await assert.rejects(cliente(respuesta).descargarBorrador(solicitud, { formato: "docx" }), { codigo: "resultado_no_confiable" });
});
