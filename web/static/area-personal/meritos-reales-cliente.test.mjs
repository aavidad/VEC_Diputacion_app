import test from "node:test";
import assert from "node:assert/strict";
import { crearClienteCartera, ErrorCartera } from "./meritos-reales-cliente.js";

const respuesta = (status, data) => ({ status, ok: status >= 200 && status < 300,
  async json() { return data; } });

test("GET propio usa un único filtro del servidor, sin identidad ni cookies", async () => {
  const llamadas = [];
  const cliente = crearClienteCartera({ fetchImpl: async (ruta, opciones) => {
    llamadas.push({ ruta, opciones });
    return respuesta(200, { data: { hechos: [], total: 0, filtro: "retirados",
      motivos_retirada: [{ etiqueta: "Incorrect document", motivo: { entrada_clave: "motivo_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" } }],
      corte_en: "2026-10-12T15:00:00Z", recibo_acceso: { auditoria_ref: "aud:1" } } });
  } });
  const pagina = await cliente.consultar({ filtro: "retirados", idioma: "en", limite: 25 });
  assert.equal(pagina.cursor_siguiente, "");
  assert.equal(llamadas.length, 1);
  assert.equal(llamadas[0].ruta, "/api/meritos/cartera?filtro=retirados&limite=25&lang=en");
  assert.equal(llamadas[0].opciones.credentials, "omit");
  assert.equal(llamadas[0].opciones.cache, "no-store");
  assert.equal(llamadas[0].opciones.method, "GET");
  assert.doesNotMatch(llamadas[0].ruta, /persona_ref|perfil_ref/u);
});

test("POST incierto conserva la clave entregada y nunca hace otro intento solo", async () => {
  const llamadas = [];
  const cliente = crearClienteCartera({ fetchImpl: async (ruta, opciones) => {
    llamadas.push({ ruta, opciones });
    throw new Error("conexión perdida tras posible COMMIT");
  } });
  const orden = { hecho_ref: "hecho:1", clave_idempotencia: "clave:retirada:1" };
  await assert.rejects(cliente.retirar(orden), (err) => err instanceof ErrorCartera
    && err.codigo === "resultado_indeterminado");
  assert.equal(llamadas.length, 1);
  assert.equal(JSON.parse(llamadas[0].opciones.body).clave_idempotencia, orden.clave_idempotencia);
  assert.equal(llamadas[0].opciones.credentials, "omit");
});

test("POST distingue alta y replay por respuesta y rechaza recibo incoherente", async () => {
  let repetida = false;
  const cliente = crearClienteCartera({ fetchImpl: async () => respuesta(repetida ? 200 : 201,
    { data: { repetida, recibo: { referencia: "recibo:original" }, auditoria_acceso_ref: "auditoria:nueva" } }) });
  const orden = { hecho_ref: "hecho:1", clave_idempotencia: "clave:retirada:1" };
  assert.equal((await cliente.retirar(orden)).repetida, false);
  repetida = true;
  assert.equal((await cliente.retirar(orden)).recibo.referencia, "recibo:original");
  const falso = crearClienteCartera({ fetchImpl: async () => respuesta(200,
    { data: { repetida: false, recibo: { referencia: "recibo:otro" }, auditoria_acceso_ref: "aud:1" } }) });
  await assert.rejects(falso.retirar(orden), (err) => err instanceof ErrorCartera
    && err.codigo === "resultado_indeterminado");
});
