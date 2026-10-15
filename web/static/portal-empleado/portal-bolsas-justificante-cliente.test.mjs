import assert from "node:assert/strict";
import test from "node:test";
import { crearClienteJustificanteRenuncia } from "./portal-bolsas-justificante-cliente.js";

const respuesta = (bytes = "%PDF-1.7\noriginal", headers = {}) => new Response(bytes, { status: 200, headers: {
  "Content-Type": "application/pdf", "Content-Disposition": 'attachment; filename="justificante.pdf"', ...headers,
} });

test("la consulta es solo GET, mismo origen y no almacena el documento", async () => {
  const llamadas = [];
  const cliente = crearClienteJustificanteRenuncia({ fetchImpl: async (ruta, opciones) => {
    llamadas.push({ ruta, opciones });
    return respuesta();
  } });
  assert.equal(llamadas.length, 0);
  const archivo = await cliente.descargar("bolsa:01", "participacion:01");
  assert.equal(llamadas.length, 1);
  assert.equal(llamadas[0].opciones.method, "GET");
  assert.equal(llamadas[0].opciones.credentials, "same-origin");
  assert.equal(llamadas[0].opciones.cache, "no-store");
  assert.equal(llamadas[0].opciones.redirect, "error");
  assert.equal(llamadas[0].opciones.referrerPolicy, "no-referrer");
  assert.equal(archivo.nombre, "justificante.pdf");
  assert.equal(new TextDecoder().decode(archivo.bytes), "%PDF-1.7\noriginal");
});

test("no entrega HTML, nombre inyectado ni PDF vacío", async () => {
  for (const respuestaHTTP of [
    respuesta("<html>", { "Content-Type": "text/html" }),
    respuesta("%PDF-1.7", { "Content-Disposition": 'attachment; filename="../../otro.pdf"' }),
    respuesta(""),
  ]) {
    const cliente = crearClienteJustificanteRenuncia({ fetchImpl: async () => respuestaHTTP });
    await assert.rejects(cliente.descargar("bolsa:01", "participacion:01"));
  }
});
