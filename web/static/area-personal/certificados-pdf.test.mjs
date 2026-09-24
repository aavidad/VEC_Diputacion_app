import assert from "node:assert/strict";
import test from "node:test";
import { crearDescriptorPDFRecibo } from "./aplicacion.js";

const origen = "https://portal.example.test";
const reciboReal = Object.freeze({
  accion: "solicitar_certificado", actor: "persona:opaca:1234",
  advertencia: "Pendiente de emisión documental autorizada.",
  fecha: "2026-09-24T12:30:00.000Z", objetivo: "certificado:opaco:1234",
  presentacion: false, referencia: "recibo:opaco:1234", resultado: "Solicitud registrada",
});

test("el área personal rechaza recibos de presentación antes de preparar un PDF", () => {
  assert.throws(() => crearDescriptorPDFRecibo({
    recibo: { ...reciboReal, presentacion: true, referencia: "DEMO-REC-0042" },
    certificados: [{ id: reciboReal.objetivo, tipo: "Certificado de inscripción" }], origen,
  }), /no está admitido/u);
});

test("el descriptor operativo no crea un enlace de presentación ni simula certificación", () => {
  const descriptor = crearDescriptorPDFRecibo({
    recibo: reciboReal,
    certificados: [{ id: reciboReal.objetivo, tipo: "Certificado de inscripción" }], origen,
  });
  assert.equal(descriptor.comprobacion.qr_contenido, `${origen}/verificar/?ref=recibo%3Aopaco%3A1234`);
  assert.doesNotMatch(descriptor.comprobacion.qr_contenido, /presentacion=/u);
  assert.match(descriptor.texto_certificacion, /requiere su firma o sello verificable/u);
  assert.doesNotMatch(descriptor.texto_certificacion, /CERTIFICA|Demostración|sintético/u);
});

test("un certificado ajeno al catálogo visible se rechaza", () => {
  assert.throws(() => crearDescriptorPDFRecibo({ recibo: reciboReal, certificados: [], origen }),
    /no pertenece al expediente visible/u);
});
