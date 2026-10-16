import assert from "node:assert/strict";
import test from "node:test";
import { crearClienteAutoFirma } from "./firma-autofirma.js";

test("AutoFirma recibe el PDF de 2 MiB y rechaza el byte adicional antes de abrir el canal", async () => {
  const maximo = 2 * 1024 * 1024;
  const pdf = new Uint8Array(maximo).fill(0x78);
  pdf.set(new TextEncoder().encode("%PDF-"), 0);
  const exceso = new Uint8Array(maximo + 1); exceso.set(pdf); exceso[maximo] = 0x78;
  let lanzamientos = 0;
  class CanalNoDisponible { constructor() { throw new Error("canal de prueba cerrado"); } }
  const cliente = crearClienteAutoFirma({ WebSocketImpl: CanalNoDisponible,
    lanzar: () => { lanzamientos++; }, aleatorio: (n) => new Uint8Array(n).fill(1),
    intentos: 1, esperaReintentoMs: 0 });
  await assert.rejects(cliente.firmarPDF(pdf), { codigo: "autofirma_no_disponible" });
  assert.equal(lanzamientos, 1);
  await assert.rejects(cliente.firmarPDF(exceso), { codigo: "documento_no_valido" });
  assert.equal(lanzamientos, 1);
});
