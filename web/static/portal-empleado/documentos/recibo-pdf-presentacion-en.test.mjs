import assert from "node:assert/strict";
import test from "node:test";

globalThis.location = { href: "https://portal.example.test/?lang=en" };
const { generarReciboPDFPresentacion } = await import("./recibo-pdf-presentacion.js");

test("lang=en traduce los rótulos fijos del PDF DEMO sin alterar los datos", async () => {
  const blob = generarReciboPDFPresentacion({
    referencia: "DEMO-TEST-0001",
    titulo: "Source title",
    filas: [["Source label", "Source value"]],
    urlVerificacion: "https://portal.example.test/verificar/?ref=DEMO-TEST-0001",
    origenInstitucional: "https://portal.example.test",
  });
  const pdf = Buffer.from(await blob.arrayBuffer()).toString("latin1");
  assert.match(pdf, /Employee Portal - DEMO/u);
  assert.match(pdf, /Status: demonstration document with no administrative validity/u);
  assert.match(pdf, /Source label/u);
  assert.match(pdf, /Source value/u);
  assert.match(pdf, /DEMO-TEST-0001/u);
  assert.doesNotMatch(pdf, /Estado: documento de demostracion/u);
});
