import assert from "node:assert/strict";
import test from "node:test";
import { crearClienteBorradorLlamamiento, ErrorAPIBorradorLlamamiento, RUTA_BORRADORES_LLAMAMIENTO } from "./portal-borrador-llamamiento-api.js";

const ref = `borrador-llamamiento:alta:${"a".repeat(64)}`;
const recibo = { data: { borrador_ref: ref, estado: "borrador_interno", version: "1", resumen: "Preparar cobertura interna", recibo_ref: `recibo:${"a".repeat(64)}`, registrado_en: "2026-09-21T10:30:00Z", reintento_idempotente: false } };
function respuesta(cuerpo, estado = 201) {
  const texto = JSON.stringify(cuerpo);
  return new Response(texto, { status: estado, headers: { "Content-Type": "application/json", "Content-Length": String(new TextEncoder().encode(texto).byteLength) } });
}

test("crea con contrato mínimo, identidad resuelta por servidor y clave criptográfica", async () => {
  let llamada;
  const cliente = crearClienteBorradorLlamamiento({ criptografia: { getRandomValues: (bytes) => bytes.fill(7) }, fetchImpl: async (...args) => { llamada = args; return respuesta(recibo); } });
  const resultado = await cliente.crear({ resumen: "Preparar cobertura interna" });
  assert.equal(resultado.borrador_ref, ref);
  assert.equal(llamada[0], RUTA_BORRADORES_LLAMAMIENTO);
  assert.equal(llamada[1].credentials, "same-origin");
  assert.equal(llamada[1].headers["Idempotency-Key"], `blam-${"07".repeat(24)}`);
  assert.deepEqual(JSON.parse(llamada[1].body), { resumen: "Preparar cobertura interna" });
  assert.equal(Object.hasOwn(llamada[1].headers, "Authorization"), false);
});

test("consulta únicamente una referencia opaca exacta", async () => {
  let ruta;
  const cliente = crearClienteBorradorLlamamiento({ fetchImpl: async (valor) => { ruta = valor; return respuesta(recibo, 200); } });
  await cliente.consultar({ referencia: ref });
  assert.equal(ruta, `${RUTA_BORRADORES_LLAMAMIENTO}/${ref}`);
  await assert.rejects(async () => cliente.consultar({ referencia: "dni:12345678Z" }), /referencia/);
});

test("expone denegación y conflicto sin perder la clave que entrega la interfaz", async () => {
  const cliente = crearClienteBorradorLlamamiento({ fetchImpl: async () => respuesta({ error: { codigo: "clave_idempotencia_reutilizada" } }, 409) });
  await assert.rejects(cliente.crear({ resumen: "Preparar cobertura interna", claveIdempotencia: "blam-12345678" }), (error) => error instanceof ErrorAPIBorradorLlamamiento && error.estado === 409 && error.codigo === "clave_idempotencia_reutilizada");
});

test("clasifica estados de autenticación, autorización, ausencia, límite e infraestructura", async () => {
  for (const estado of [401, 403, 404, 413, 503]) {
    const cliente = crearClienteBorradorLlamamiento({ fetchImpl: async () => respuesta({ error: { codigo: "controlado" } }, estado) });
    await assert.rejects(cliente.crear({ resumen: "Preparar cobertura interna", claveIdempotencia: "blam-12345678" }), (error) => error instanceof ErrorAPIBorradorLlamamiento && error.estado === estado && error.codigo === "controlado");
  }
});

test("mide el resumen en bytes UTF-8, igual que el contrato Go", async () => {
  const cliente = crearClienteBorradorLlamamiento({ fetchImpl: async () => respuesta(recibo) });
  await cliente.crear({ resumen: "€", claveIdempotencia: "blam-12345678" });
  await cliente.crear({ resumen: "á".repeat(1000), claveIdempotencia: "blam-12345679" });
  await assert.rejects(async () => cliente.crear({ resumen: "á".repeat(1001), claveIdempotencia: "blam-12345670" }), /2.000/);
});

test("cancela sin convertir AbortError en éxito o mensaje de conexión", async () => {
  let abortado = false;
  const cliente = crearClienteBorradorLlamamiento({ fetchImpl: (_ruta, opciones) => new Promise((_resolve, reject) => opciones.signal.addEventListener("abort", () => { abortado = true; reject(new DOMException("cancelado", "AbortError")); }, { once: true })) });
  const abortador = new AbortController();
  const operacion = cliente.crear({ resumen: "Preparar cobertura interna", claveIdempotencia: "blam-12345678", signal: abortador.signal });
  abortador.abort();
  await assert.rejects(operacion, /AbortError|cancelado/);
  assert.equal(abortado, true);
});
