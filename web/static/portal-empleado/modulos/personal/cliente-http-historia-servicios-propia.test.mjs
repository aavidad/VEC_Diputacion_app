import assert from "node:assert/strict";
import test from "node:test";
import { crearClienteHistoriaServiciosPropia, validarRespuestaHistoriaServicios, RUTA_HISTORIA_SERVICIOS_PROPIA } from "./cliente-http-historia-servicios-propia.js";

const filtros = { efectosDesde: "2024-01-01", efectosHasta: "2025-01-01" };
function revision(version = 1) {
  return { servicio_ref: "srv_AAAAAAAAAAAAAAAAAAAAAA", relacion_ref: "rel_CCCCCCCCCCCCCCCCCCCCCC", periodo_desde: "2019-01-01", periodo_hasta: "2019-12-31", dias_reconocidos: 365, estado: "reconocido", clase: "Servicios propios", traza: { desde: "2024-01-01", registrada_en: "2024-03-01T09:00:00.000000Z", version, acto_ref: "acto:uno", fuente_ref: "fuente:uno", fuente_version: 1 } };
}
const sobre = () => ({ data: { historia: { corte: { efectos_desde: filtros.efectosDesde, efectos_hasta: filtros.efectosHasta, conocido_en: "2026-10-04T08:00:00.123456Z" }, cobertura: "parcial", revisiones: [revision(2), revision(1)] }, consultada_en: "2026-10-04T08:00:01.000000Z", recibo_ref: "aud_v3_abcdef0123456789abcdef0123456789" } });
const respuesta = (datos, status = 200) => new Response(JSON.stringify(datos), { status, headers: { "Content-Type": "application/json; charset=utf-8" } });

test("POST sólo dos fechas: conocimiento, actor y empleado los resuelve el servidor; conserva todas las revisiones", async () => {
  const llamadas = [];
  const cliente = crearClienteHistoriaServiciosPropia({ fetchImpl: async (...args) => { llamadas.push(args); return respuesta(sobre()); } });
  const resultado = await cliente.consultar({ ...filtros, actor: "oculto", empleado: "oculto", conocidoEn: "no_admitido" });
  assert.equal(llamadas.length, 1); const [ruta, opciones] = llamadas[0];
  assert.equal(ruta, RUTA_HISTORIA_SERVICIOS_PROPIA);
  assert.deepEqual(JSON.parse(opciones.body), { efectos_desde: filtros.efectosDesde, efectos_hasta: filtros.efectosHasta });
  assert.deepEqual([opciones.method, opciones.mode, opciones.credentials, opciones.redirect, opciones.cache, opciones.referrerPolicy], ["POST", "same-origin", "same-origin", "error", "no-store", "no-referrer"]);
  assert.deepEqual(resultado.historia.revisiones.map((r) => r.traza.version), [2, 1]); assert.equal(resultado.historia.cobertura, "parcial");
  assert.ok(Object.isFrozen(resultado.historia.revisiones[0].traza));
});

test("rango exclusivo e inexistencia de fecha se rechazan sin red", async () => {
  let llamadas = 0; const cliente = crearClienteHistoriaServiciosPropia({ fetchImpl: async () => { llamadas++; } });
  for (const entrada of [{}, { ...filtros, efectosHasta: filtros.efectosDesde }, { ...filtros, efectosDesde: "2025-02-29" }, { ...filtros, efectosHasta: "0000-01-01" }]) await assert.rejects(cliente.consultar(entrada), (e) => e.codigo === "intervalo_invalido");
  assert.equal(llamadas, 0);
});

test("corte ajeno, recibo de ficha, datos internos, orden, duplicados y revisión posterior al conocimiento son rechazados", () => {
  const cambios = [
    (s) => { s.data.historia.corte.efectos_hasta = "2026-01-01"; },
    (s) => { s.data.recibo_ref = "fichapropia:0f0e0d0c-0b0a-4908-8706-050403020100"; },
    (s) => { s.data.historia.empleado_ref = "empleado:oculto"; },
    (s) => { s.data.historia.revisiones.reverse(); },
    (s) => { s.data.historia.revisiones[1].traza.version = 2; },
    (s) => { s.data.historia.revisiones[0].traza.registrada_en = "2026-10-05T09:00:00.000000Z"; },
    (s) => { s.data.historia.revisiones[0].traza.hasta = "2023-12-31"; },
  ];
  for (const cambiar of cambios) { const datos = sobre(); cambiar(datos); assert.throws(() => validarRespuestaHistoriaServicios(datos, filtros), (e) => e.codigo === "respuesta_no_valida"); }
});

test("vacío conserva cobertura, exceso nunca se trunca", () => {
  const datos = sobre(); datos.data.historia.revisiones = []; datos.data.historia.cobertura = "no_acreditada";
  assert.equal(validarRespuestaHistoriaServicios(datos, filtros).historia.cobertura, "no_acreditada");
  datos.data.historia.revisiones = Array.from({ length: 201 }, () => revision());
  assert.throws(() => validarRespuestaHistoriaServicios(datos, filtros), (e) => e.codigo === "excede_limite");
});

test("referencias propias conservan mayúsculas y admiten exactamente 22 a 128 caracteres después del prefijo", () => {
  for (const longitud of [22, 128]) {
    const datos = sobre();
    for (const fila of datos.data.historia.revisiones) { fila.servicio_ref = `srv_${"A".repeat(longitud)}`; fila.relacion_ref = `rel_${"C".repeat(longitud)}`; }
    assert.equal(validarRespuestaHistoriaServicios(datos, filtros).historia.revisiones.length, 2);
  }
  for (const campo of ["servicio_ref", "relacion_ref"]) for (const longitud of [21, 129]) {
    const datos = sobre(); datos.data.historia.revisiones[0][campo] = `${campo === "servicio_ref" ? "srv_" : "rel_"}${"A".repeat(longitud)}`;
    assert.throws(() => validarRespuestaHistoriaServicios(datos, filtros), (e) => e.codigo === "respuesta_no_valida");
  }
});

test("401/403/422/503 tienen estado propio y respuesta ajena queda opaca", async () => {
  for (const [estado, codigo, esperado] of [[401, "autenticacion_requerida", "sesion_caducada"], [403, "acceso_denegado", "denegado"], [422, "excede_limite", "excede_limite"], [503, "no_disponible", "no_disponible"], [503, "detalle_privado", "respuesta_no_valida"]]) {
    const cliente = crearClienteHistoriaServiciosPropia({ fetchImpl: async () => respuesta({ error: codigo }, estado) });
    await assert.rejects(cliente.consultar(filtros), (e) => e.codigo === esperado && !e.message.includes("detalle_privado"));
  }
});

test("cancelación y plazo no devuelven resultados tardíos", async () => {
  const controlador = new AbortController(); let llamadas = 0;
  const cliente = crearClienteHistoriaServiciosPropia({ fetchImpl: async (_ruta, { signal }) => { llamadas++; return new Promise((_r, rechazar) => signal.addEventListener("abort", () => rechazar(new Error("abort")))); } });
  const pendiente = cliente.consultar({ ...filtros, signal: controlador.signal }); controlador.abort();
  await assert.rejects(pendiente, (e) => e.codigo === "operacion_abortada");
  await assert.rejects(cliente.consultar({ ...filtros, signal: controlador.signal }), (e) => e.codigo === "operacion_abortada"); assert.equal(llamadas, 1);
  const lento = crearClienteHistoriaServiciosPropia({ plazoMs: 5, fetchImpl: async (_r, { signal }) => new Promise((_ok, no) => signal.addEventListener("abort", () => no(new Error("timeout")))) });
  await assert.rejects(lento.consultar(filtros), (e) => e.codigo === "no_disponible");
});

test("cuerpo mayor del máximo cancela su lector y no entrega filas", async () => {
  let cancelada = false;
  const body = new ReadableStream({ start(c) { c.enqueue(new Uint8Array(1024 * 1024 + 1)); }, cancel() { cancelada = true; } });
  const cliente = crearClienteHistoriaServiciosPropia({ fetchImpl: async () => new Response(body, { headers: { "Content-Type": "application/json; charset=utf-8" } }) });
  await assert.rejects(cliente.consultar(filtros), (e) => e.codigo === "respuesta_no_valida"); assert.equal(cancelada, true);
});
