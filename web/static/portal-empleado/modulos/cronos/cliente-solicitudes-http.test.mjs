import assert from "node:assert/strict";
import test from "node:test";
import { crearClienteSolicitudesCronosHTTP, ErrorClienteSolicitudesCronos, RUTAS_SOLICITUDES_CRONOS } from "./cliente-solicitudes-http.js";

function respuestaJSON(estado, cuerpo) {
  const bytes = new TextEncoder().encode(JSON.stringify(cuerpo));
  return { status: estado, ok: estado >= 200 && estado < 300, redirected: false,
    headers: { get: (n) => (n === "content-type" ? "application/json; charset=utf-8" : null) },
    body: { getReader() { let hecho = false; return { read: async () => (hecho ? { done: true } : (hecho = true, { done: false, value: bytes })), releaseLock() {}, cancel: async () => {} }; } } };
}
const MOVIMIENTOS = {
  periodo: { tipo: "anio", desde: "2026-01-01", hasta: "2026-12-31" },
  calendario: { disponible: true, dias: [{ fecha: "2026-01-06", tipo: "festivo", nombre: "Epifanía" }] },
  marcajes_por_dia: [{ fecha: "2026-09-21", marcajes: 4 }],
  absentismos: [{ solicitud_ref: "permiso:cronos:solicitud:c-0000001", permiso_ref: "permiso:cronos:traslado", nombre: "Traslado de domicilio", desde: "2026-09-22", hasta: "2026-09-22", cantidad: 1, unidad: "dia", pendiente_justificar: true }],
  correcciones: [{ solicitud_ref: "correccion:cronos:corr-0000001", fecha_civil: "2026-09-24", hora_pretendida: "08:00", movimiento: "entrada", estado: "pendiente_responsable", version: 1, solicitada_en: "2026-09-25T07:00:00.123456Z" }],
};

test("consulta movimientos por la ruta propia, mismo origen y sin identidad del cliente", async () => {
  const llamadas = [];
  const cliente = crearClienteSolicitudesCronosHTTP({ fetchImpl: async (url, opciones) => { llamadas.push({ url, opciones }); return respuestaJSON(200, MOVIMIENTOS); } });
  const r = await cliente.consultarMovimientos({ periodo: "anio" });
  assert.equal(r.calendario.dias[0].nombre, "Epifanía");
  assert.equal(llamadas[0].url, `${RUTAS_SOLICITUDES_CRONOS.movimientos}?periodo=anio`);
  assert.equal(llamadas[0].opciones.credentials, "same-origin");
  assert.equal(llamadas[0].opciones.redirect, "error");
  assert.equal(llamadas[0].opciones.referrerPolicy, "no-referrer");
  assert.equal(llamadas[0].opciones.body, undefined);
  const inventado = structuredClone(MOVIMIENTOS); inventado.calendario = { disponible: false, dias: MOVIMIENTOS.calendario.dias };
  const malo = crearClienteSolicitudesCronosHTTP({ fetchImpl: async () => respuestaJSON(200, inventado) });
  await assert.rejects(malo.consultarMovimientos({ periodo: "anio" }), (e) => e instanceof ErrorClienteSolicitudesCronos && e.codigo === "respuesta_incompatible");
});

test("la solicitud de corrección envía sólo la declaración y exige recibo pendiente de jefatura", async () => {
  let enviado;
  const recibo = { recibo: { solicitud_ref: "correccion:cronos:corr-clave-0001", actuacion_ref: "correccion:actuacion:x", recibo_ref: "recibo:cronos:1", estado: "pendiente_responsable", version: 1, instante_utc: "2026-09-25T07:00:00Z", replay: false } };
  const cliente = crearClienteSolicitudesCronosHTTP({ fetchImpl: async (_url, o) => { enviado = o; return respuestaJSON(201, recibo); } });
  const r = await cliente.solicitarCorreccion({ clave_operacion: "corr-clave-0001", movimiento: "entrada", fecha_civil: "2026-09-24", hora_pretendida: "08:00" });
  assert.equal(r.estado, "pendiente_responsable");
  assert.equal(enviado.method, "POST");
  assert.deepEqual(JSON.parse(enviado.body), { clave_operacion: "corr-clave-0001", movimiento: "entrada", fecha_civil: "2026-09-24", hora_pretendida: "08:00" });
  await assert.rejects(cliente.solicitarCorreccion({ clave_operacion: "corr-clave-0001", movimiento: "entrada", fecha_civil: "2026-09-24", hora_pretendida: "08:00", empleado_ref: "emp_x" }), TypeError);
});

test("los rechazos nominales del permiso llegan con su código y los demás son no disponible", async () => {
  const entrada = { clave_operacion: "perm-clave-0001", permiso_ref: "permiso:cronos:asuntos-propios", desde: "2026-10-05", hasta: "2026-10-06" };
  for (const [estado, cuerpo, codigo] of [[422, { error: "fuera_de_limites" }, "fuera_de_limites"], [422, { error: "otro" }, "servicio_no_disponible"],
    [409, { error: "conflicto" }, "conflicto"], [403, { error: "sin_empleado" }, "sin_empleado"], [403, { error: "acceso_denegado" }, "acceso_denegado"], [503, { error: "no_disponible" }, "servicio_no_disponible"]]) {
    const cliente = crearClienteSolicitudesCronosHTTP({ fetchImpl: async () => respuestaJSON(estado, cuerpo) });
    await assert.rejects(cliente.solicitarPermiso(entrada), (e) => e instanceof ErrorClienteSolicitudesCronos && e.codigo === codigo, `${estado} ${cuerpo.error}`);
  }
  const concedido = { recibo: { solicitud_ref: "permiso:cronos:solicitud:perm-clave-0001", recibo_ref: "recibo:cronos:1", catalogo_version_ref: "catalogo:cronos:x", version: 1, estado: "concedido", cantidad: 2, unidad: "dia", instante_utc: "2026-09-25T07:00:00Z", replay: false } };
  const cliente = crearClienteSolicitudesCronosHTTP({ fetchImpl: async () => respuestaJSON(201, concedido) });
  await assert.rejects(cliente.solicitarPermiso(entrada), (e) => e.codigo === "respuesta_incompatible");
  await assert.rejects(cliente.solicitarPermiso({ ...entrada, hora_inicio: "09:00", hora_fin: "10:00" }), TypeError);
});
