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

test("recupera el recibo inicial con GET propio y rechaza una respuesta que no acredita su origen", async () => {
  const operacion = "123e4567-e89b-42d3-a456-426614174000";
  const cuerpo = { recibo: { solicitud_ref: `correccion:cronos:${operacion}`, actuacion_ref: "correccion:actuacion:original", recibo_ref: "recibo:cronos:original", estado: "pendiente_responsable", version: 1, instante_utc: "2026-09-25T07:00:00Z", replay: true } };
  const llamadas = [];
  const cliente = crearClienteSolicitudesCronosHTTP({ fetchImpl: async (url, opciones) => { llamadas.push({ url, opciones }); return respuestaJSON(200, cuerpo); } });
  assert.equal((await cliente.recuperarReciboCorreccion(operacion)).recibo_ref, cuerpo.recibo.recibo_ref);
  assert.equal(llamadas[0].url, `${RUTAS_SOLICITUDES_CRONOS.reciboCorreccion}?clave_operacion=${operacion}`);
  assert.equal(llamadas[0].opciones.method, "GET");
  assert.equal(llamadas[0].opciones.body, undefined);
  assert.equal(llamadas[0].opciones.credentials, "same-origin");
  assert.equal(llamadas[0].opciones.cache, "no-store");
  assert.equal(llamadas[0].opciones.redirect, "error");
  assert.equal(llamadas[0].opciones.referrerPolicy, "no-referrer");
  await assert.rejects(cliente.recuperarReciboCorreccion("otra-clave"), TypeError);
  assert.equal(llamadas.length, 1);
  for (const recibo of [{ ...cuerpo.recibo, replay: false }, { ...cuerpo.recibo, version: 2 }, { ...cuerpo.recibo, actuacion_ref: "actuacion:cronos:incorrecta" }]) {
    const incompatible = crearClienteSolicitudesCronosHTTP({ fetchImpl: async () => respuestaJSON(200, { recibo }) });
    await assert.rejects(incompatible.recuperarReciboCorreccion(operacion), (e) => e.codigo === "respuesta_incompatible");
  }
  for (const estado of [403, 503]) {
    const fallido = crearClienteSolicitudesCronosHTTP({ fetchImpl: async () => respuestaJSON(estado, { error: "no_disponible" }) });
    await assert.rejects(fallido.recuperarReciboCorreccion(operacion), (e) => e instanceof ErrorClienteSolicitudesCronos && e.estado === estado);
  }
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

test("cada solicitud propia admite el circuito aplicado sólo coherente con su estado", async () => {
  const solicitud = { solicitud_ref: "permiso:cronos:solicitud:a-0000001", catalogo_version_ref: "catalogo:cronos:asuntos-propios:v1", permiso_ref: "permiso:cronos:asuntos-propios",
    desde: "2026-10-05", hasta: "2026-10-05", cantidad: 1, unidad: "dia", estado: "solicitado", version: 1, pendiente_justificar: false, solicitada_en: "2026-09-25T08:00:00Z" };
  const consulta = (extra) => crearClienteSolicitudesCronosHTTP({ fetchImpl: async () => respuestaJSON(200, { anio: 2026, permisos: [], solicitudes: [{ ...solicitud, ...extra }] }) })
    .consultarPermisos({ anio: 2026 });
  for (const extra of [{}, { circuito: "J-A" }, { circuito: "A" }, { circuito: "J-A", pendiente_asignacion: true },
    { estado: "pendiente_administracion", version: 2, circuito: "J-A" }, { estado: "concedido", version: 2 }, { estado: "concedido", version: 2, circuito: "A" }]) {
    assert.equal((await consulta(extra)).solicitudes.length, 1, JSON.stringify(extra));
  }
  for (const extra of [{ circuito: "X" }, { circuito: "A", pendiente_asignacion: true }, { circuito: "J-A", pendiente_asignacion: false },
    { pendiente_asignacion: true }, { estado: "pendiente_administracion", version: 2, circuito: "A" }, { estado: "concedido", version: 2, circuito: "J-A", pendiente_asignacion: true }]) {
    await assert.rejects(consulta(extra), (e) => e instanceof ErrorClienteSolicitudesCronos && e.codigo === "respuesta_incompatible", JSON.stringify(extra));
  }
});
