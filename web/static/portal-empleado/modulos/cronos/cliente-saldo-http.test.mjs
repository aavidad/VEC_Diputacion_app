import assert from "node:assert/strict";
import test from "node:test";
import { crearClienteSaldoCronosHTTP, ErrorClienteSaldoCronos, validarConsultaSaldoCronos, validarResultadoSaldoCronos } from "./cliente-saldo-http.js";

const resultado = {
  periodo: { tipo: "rango", desde: "2026-09-20", hasta: "2026-09-21" },
  resumen: { previstos_minutos: null, trabajados_minutos: 31, saldo_minutos: null, estado: "no_disponible" },
  detalle: [
    { fecha: "2026-09-20", previstos_minutos: null, trabajados_minutos: 31, pausas_minutos: 0, saldo_minutos: null, estado: "no_disponible", marcajes: [
      { instante_utc: "2026-09-20T09:10:00Z", movimiento: "entrada", origen: null },
    ] },
  ],
};
function respuesta(cuerpo, estado = 200) {
  return new Response(JSON.stringify(cuerpo), { status: estado, headers: { "content-type": "application/json; charset=utf-8" } });
}

test("consulta propia usa solo periodo y fechas civiles inclusivas, sin identidad del cliente", async () => {
  let llamada;
  const cliente = crearClienteSaldoCronosHTTP({ fetchImpl: async (url, opciones) => {
    llamada = { url, opciones }; return respuesta(resultado);
  } });
  const dato = await cliente.consultar({ periodo: "rango", desde: "2026-09-20", hasta: "2026-09-21" });
  assert.deepEqual(dato, resultado);
  assert.equal(llamada.url, "/api/interna/cronos/saldos/propio?periodo=rango&desde=2026-09-20&hasta=2026-09-21");
  assert.equal(llamada.opciones.method, "GET");
  assert.equal(llamada.opciones.cache, "no-store");
  assert.equal(llamada.opciones.credentials, "same-origin");
  assert.equal(llamada.opciones.mode, "same-origin");
  assert.equal(llamada.opciones.body, undefined);
  assert.equal(llamada.opciones.headers.Authorization, undefined);
  assert.throws(() => validarConsultaSaldoCronos({ periodo: "hoy", empleado_ref: "otra_persona" }), TypeError);
  assert.throws(() => validarConsultaSaldoCronos({ periodo: "rango", desde: "2026-02-30", hasta: "2026-03-01" }), TypeError);
});

test("denegación y respuesta inconsistente no se convierten en cifras", async () => {
  const denegado = crearClienteSaldoCronosHTTP({ fetchImpl: async () => respuesta({ error: "denegado" }, 403) });
  await assert.rejects(denegado.consultar({ periodo: "hoy" }), (error) => error instanceof ErrorClienteSaldoCronos && error.codigo === "acceso_denegado");
  const incorrecto = structuredClone(resultado);
  incorrecto.periodo.tipo = "hoy";
  incorrecto.detalle[0].saldo_minutos = 0;
  incorrecto.detalle[0].marcajes[0].instante_utc = "fecha_no_valida";
  const cliente = crearClienteSaldoCronosHTTP({ fetchImpl: async () => respuesta(incorrecto) });
  await assert.rejects(cliente.consultar({ periodo: "hoy" }), (error) => error instanceof ErrorClienteSaldoCronos && error.codigo === "respuesta_incompatible");
  const origenSinTipo = structuredClone(resultado);
  origenSinTipo.detalle[0].marcajes[0].origen = "origen_ref_001";
  const clienteOrigen = crearClienteSaldoCronosHTTP({ fetchImpl: async () => respuesta(origenSinTipo) });
  await assert.rejects(clienteOrigen.consultar({ periodo: "rango", desde: "2026-09-20", hasta: "2026-09-21" }), (error) => error instanceof ErrorClienteSaldoCronos && error.codigo === "respuesta_incompatible");
  const campoReservado = structuredClone(resultado);
  campoReservado.detalle[0].marcajes[0].origen_ref = "referencia_interna";
  const clienteCampo = crearClienteSaldoCronosHTTP({ fetchImpl: async () => respuesta(campoReservado) });
  await assert.rejects(clienteCampo.consultar({ periodo: "rango", desde: "2026-09-20", hasta: "2026-09-21" }), (error) => error instanceof ErrorClienteSaldoCronos && error.codigo === "respuesta_incompatible");
});

test("AbortSignal cancela la consulta y no reintenta otra petición", async () => {
  let llamadas = 0;
  const controlador = new AbortController();
  const cliente = crearClienteSaldoCronosHTTP({ fetchImpl: async (_url, opciones) => {
    llamadas++;
    return new Promise((_resolver, rechazar) => opciones.signal.addEventListener("abort", () => rechazar(new DOMException("aborted", "AbortError")), { once: true }));
  } });
  const pendiente = cliente.consultar({ periodo: "mes" }, { signal: controlador.signal });
  await Promise.resolve(); controlador.abort();
  await assert.rejects(pendiente, (error) => error instanceof ErrorClienteSaldoCronos && error.codigo === "operacion_abortada");
  assert.equal(llamadas, 1);
});

test("admite nanosegundos UTC de Go y rechaza zona distinta o fecha imposible", () => {
  const consulta = { periodo: "rango", desde: "2026-09-20", hasta: "2026-09-21" };
  const dato = structuredClone(resultado);
  dato.detalle[0].marcajes[0].instante_utc = "2026-09-20T09:10:00.123456789Z";
  assert.doesNotThrow(() => validarResultadoSaldoCronos(dato, consulta));
  for (const instante of ["2026-09-20T09:10:00.123456789+00:00", "2026-09-20T09:10:00.1234567890Z", "2026-02-30T09:10:00.123456789Z"]) {
    dato.detalle[0].marcajes[0].instante_utc = instante;
    assert.throws(() => validarResultadoSaldoCronos(dato, consulta), (error) => error instanceof ErrorClienteSaldoCronos && error.codigo === "respuesta_incompatible");
  }
});
