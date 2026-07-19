import assert from "node:assert/strict";
import test from "node:test";
import { generarReciboPDFPresentacion } from "./recibo-pdf-presentacion.js";

const descriptor = {
  referencia: "DEMO-DIE-REC-0007",
  titulo: "Recibo de liquidación de dieta",
  subtitulo: "Comisión de servicio · estructura de documento definitiva",
  filas: [
    ["Expediente", "DEMO-DIE-2026-0007"],
    ["Itinerario", "Granada - Guadix - Granada"],
    ["Kilometraje", "121,00 km · 31,46 EUR"],
    ["Estado", "Aprobada para liquidación DEMO"],
  ],
  urlVerificacion: "http://127.0.0.2:8081/verificar/?ref=DEMO-DIE-REC-0007&presentacion=rrhh",
  nombreArchivo: "recibo-demo-die-0007.pdf",
  textoCertificacion: "La persona titular del órgano competente CERTIFICA que esta actuación figura en el escenario de demostración y carece de efectos administrativos.",
};

test("genera un PDF institucional de presentación con referencia verificable", async () => {
  const blob = generarReciboPDFPresentacion(descriptor);
  assert.equal(blob.type, "application/pdf");
  const contenido = Buffer.from(await blob.arrayBuffer()).toString("latin1");
  assert.match(contenido, /^%PDF-1\.4/);
  assert.match(contenido, /Diputacion de Granada/);
  assert.match(contenido, /Recibo de liquidacion de dieta/);
  assert.match(contenido, /CERTIFICA/);
  assert.match(contenido, /\/Encoding \/WinAnsiEncoding/);
  assert.match(contenido, /DEMO-DIE-REC-0007/);
  assert.match(contenido, /sin validez administrativa/);
  assert.match(contenido, /%%EOF$/);
  assert.ok(blob.size > 10_000, "el PDF debe contener la representación vectorial del QR");
});

test("rechaza referencias, destinos y campos que no respetan el contrato cerrado", () => {
  assert.throws(() => generarReciboPDFPresentacion({ ...descriptor, referencia: "../../dato" }), /texto documental/);
  assert.throws(() => generarReciboPDFPresentacion({ ...descriptor, urlVerificacion: "javascript:alert(1)" }), /URL/);
  assert.throws(() => generarReciboPDFPresentacion({ ...descriptor, filas: [] }), /filas/);
  assert.throws(() => generarReciboPDFPresentacion({ ...descriptor, filas: Array.from({ length: 9 }, (_, indice) => ["Dato", String(indice)]) }), /filas/);
  assert.throws(() => generarReciboPDFPresentacion({ ...descriptor, nombreArchivo: "recibo.html" }), /nombre/);
});
