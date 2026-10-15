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
  assert.equal(llamadas[0].ruta, "/api/vec/meritos/cartera?filtro=retirados&limite=25&lang=en");
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

test("carga el PDF con la clave de la declaración y exige admisión real", async () => {
  const llamadas = [];
  const documento = new Blob(["%PDF-1.7"], { type: "application/pdf" });
  documento.name = "titulo.pdf";
  const cliente = crearClienteCartera({ fetchImpl: async (ruta, opciones) => {
    llamadas.push({ ruta, opciones });
    return respuesta(200, { data: { comprobante_ref: "ref:original", carga_id: "ref:carga",
      registrado_en: "2026-10-14T12:00:00Z", estado: "admitido" } });
  } });
  assert.equal((await cliente.subirComprobante("declaracion:misma-clave", documento)).comprobante_ref, "ref:original");
  assert.equal(llamadas[0].ruta, "/api/vec/meritos/cartera/comprobantes");
  assert.equal(llamadas[0].opciones.body.get("clave_idempotencia"), "declaracion:misma-clave");
  assert.equal(await llamadas[0].opciones.body.get("documento").text(), "%PDF-1.7");
  assert.deepEqual([...llamadas[0].opciones.body.keys()], ["clave_idempotencia", "documento"]);
  assert.equal(llamadas[0].opciones.headers["Content-Type"], undefined);
  assert.equal(llamadas[0].opciones.credentials, "same-origin");
  assert.equal(llamadas[0].opciones.redirect, "error");
  const falso = crearClienteCartera({ fetchImpl: async () => respuesta(200,
    { data: { comprobante_ref: "ref:original", carga_id: "ref:carga", registrado_en: "2026-10-14T12:00:00Z", estado: "pendiente" } }) });
  await assert.rejects(falso.subirComprobante("declaracion:misma-clave", documento),
    (error) => error instanceof ErrorCartera && error.codigo === "respuesta_invalida");
});

test("descarga el PDF original por ruta fija y rechaza otro tipo de respuesta", async () => {
  const llamadas = [];
  const cliente = crearClienteCartera({ fetchImpl: async (ruta, opciones) => {
    llamadas.push({ ruta, opciones });
    return { ok: true, headers: { get: () => "application/pdf" }, blob: async () => new Blob(["%PDF"], { type: "application/pdf" }) };
  } });
  assert.equal(await (await cliente.descargarComprobante("ref:original")).text(), "%PDF");
  assert.equal(llamadas[0].ruta, "/api/vec/meritos/cartera/comprobantes/ref%3Aoriginal/original");
  assert.equal(llamadas[0].opciones.cache, "no-store");
  assert.equal(llamadas[0].opciones.credentials, "same-origin");
  assert.equal(llamadas[0].opciones.referrerPolicy, "no-referrer");
  await assert.rejects(cliente.descargarComprobante("https://otro.local/documento"), TypeError);
});

test("la carga pendiente y el rechazo se consultan por la misma clave sin simular admisión", async () => {
  const rutas = [];
  const documento = new Blob(["%PDF-1.7"], { type: "application/pdf" });
  documento.name = "titulo.pdf";
  const cliente = crearClienteCartera({ fetchImpl: async (ruta, opciones) => {
    rutas.push({ ruta, opciones });
    if (opciones.method === "POST") return respuesta(202, { data: {
      carga_id: "ref:carga1", estado: "recibido", recibo_ref: "ref:recibo1",
    } });
    return respuesta(200, { data: { carga_id: "ref:carga1", estado: "rechazado",
      recibo_ref: "ref:recibo1", codigo_rechazo: "pdf_no_valido" } });
  } });
  assert.equal((await cliente.subirComprobante("declaracion:clave1", documento)).estado, "recibido");
  const estado = await cliente.consultarComprobante("declaracion:clave1");
  assert.equal(estado.estado, "rechazado");
  assert.equal(estado.comprobante_ref, undefined);
  assert.equal(rutas[1].ruta, "/api/vec/meritos/cartera/comprobantes/estado?clave_idempotencia=declaracion%3Aclave1");
  assert.equal(rutas[1].opciones.credentials, "same-origin");
  assert.equal(rutas[1].opciones.cache, "no-store");
});
