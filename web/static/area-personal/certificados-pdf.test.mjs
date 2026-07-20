import assert from "node:assert/strict";
import test from "node:test";
import { crearDescriptorPDFRecibo } from "./aplicacion.js";
import { crearDescargadorRecibosPresentacion } from "../portal-empleado/documentos/descarga-recibos-presentacion.js";

test("el certificado DEMO se descarga como PDF real con sus datos y QR individual", async () => {
  const descriptor = crearDescriptorPDFRecibo({
    recibo: {
      accion: "solicitar_certificado",
      actor: "DEMO-PER-ASPIRANTE-001",
      advertencia: "RECIBO DEMO",
      fecha: "2026-07-20T12:30:00.000Z",
      objetivo: "DEMO-CER-001",
      presentacion: true,
      referencia: "DEMO-REC-0042",
      resultado: "Certificado preparado en el escenario de demostración",
    },
    certificados: [{
      id: "DEMO-CER-001",
      tipo: "Certificado de inscripción en bolsa",
    }],
    origen: "https://portal.example.test",
  });

  assert.equal(descriptor.tipo_documento, "CERTIFICADO");
  assert.equal(descriptor.nombre_archivo, "certificado-demo-rec-0042.pdf");
  assert.match(descriptor.texto_certificacion, /CERTIFICA/u);
  assert.match(descriptor.texto_certificacion, /Certificado de inscripción en bolsa/u);
  assert.equal(
    descriptor.comprobacion.qr_contenido,
    "https://portal.example.test/verificar/?ref=DEMO-REC-0042&presentacion=rrhh",
  );
  assert.doesNotMatch(descriptor.comprobacion.qr_contenido, /ASPIRANTE|persona/iu);

  let blob;
  let nombre;
  const entorno = {
    URL: { createObjectURL(valor) { blob = valor; return "blob:certificado-demo"; }, revokeObjectURL() {} },
    location: { origin: "https://portal.example.test" },
    document: {
      body: { append() {} },
      createElement() {
        return { hidden: false, click() {}, remove() {}, set href(_valor) {}, set download(valor) { nombre = valor; } };
      },
    },
    setTimeout(funcion) { funcion(); },
  };
  const resultado = await crearDescargadorRecibosPresentacion(entorno)(descriptor);
  assert.equal(resultado.formato, "application/pdf");
  assert.equal(nombre, "certificado-demo-rec-0042.pdf");
  assert.equal(blob.type, "application/pdf");
  const pdf = Buffer.from(await blob.arrayBuffer()).toString("latin1");
  assert.match(pdf, /^%PDF-1\.4/u);
  assert.match(pdf, /CERTIFICADO/u);
  assert.match(pdf, /Certificado de inscripcion en bolsa/u);
  assert.match(pdf, /DEMO-REC-0042/u);
  assert.match(pdf, /Diputacion de Granada/u);
  assert.match(pdf, /%%EOF$/u);
  assert.ok(blob.size > 10_000, "el certificado debe contener el QR vectorial real");
});

test("un recibo de certificado sin catálogo autorizado falla cerrado", () => {
  assert.throws(() => crearDescriptorPDFRecibo({
    recibo: {
      accion: "solicitar_certificado", objetivo: "DEMO-CER-AJENO",
      referencia: "DEMO-REC-0043", presentacion: true,
    },
    certificados: [],
    origen: "https://portal.example.test",
  }), /no pertenece al expediente visible/u);
});

test("el QR rechaza una referencia DEMO que pueda codificar identidad", () => {
  assert.throws(() => crearDescriptorPDFRecibo({
    recibo: {
      accion: "solicitar_certificado", objetivo: "DEMO-CER-001",
      referencia: "DEMO-PER-ASPIRANTE-001", presentacion: true,
    },
    certificados: [{ id: "DEMO-CER-001", tipo: "Certificado de inscripción en bolsa" }],
    origen: "https://portal.example.test",
  }), /referencia de comprobación.*no es opaca/iu);
});
