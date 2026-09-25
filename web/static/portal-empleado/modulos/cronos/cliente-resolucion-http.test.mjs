import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { crearClienteResolucionCronosHTTP, ErrorClienteResolucionCronos, RUTAS_RESOLUCION_CRONOS } from "./cliente-resolucion-http.js";

function respuestaJSON(estado, cuerpo) {
  const bytes = new TextEncoder().encode(JSON.stringify(cuerpo));
  return { status: estado, ok: estado >= 200 && estado < 300, redirected: false,
    headers: { get: (n) => (n === "content-type" ? "application/json; charset=utf-8" : null) },
    body: { getReader() { let hecho = false; return { read: async () => (hecho ? { done: true } : (hecho = true, { done: false, value: bytes })), releaseLock() {}, cancel: async () => {} }; } } };
}
const BANDEJA = {
  paso: "responsable",
  pendientes: [{ solicitud_ref: "permiso:cronos:solicitud:perm-va-00001", empleado_ref: "emp_AAAAAAAAAAAAAAAAAAAAAA", empleado_etiqueta: "Persona sintética A",
    permiso_ref: "permiso:cronos:vacaciones", nombre: "Vacaciones", circuito: "J-A", justificante_exigido: false, desde: "2026-10-05", hasta: "2026-10-07",
    cantidad: 3, unidad: "dia", estado: "solicitado", version: 1, solicitada_en: "2026-09-24T08:00:00.123456Z" }],
};
const ENTRADA = { clave_operacion: "res-va-00001", solicitud_ref: "permiso:cronos:solicitud:perm-va-00001", paso: "responsable", decision: "denegar", motivo: "Coincide con el cierre", version_esperada: 1 };
const RECIBO = { recibo: { resolucion_ref: "permiso:cronos:resolucion:res-va-00001", solicitud_ref: "permiso:cronos:solicitud:perm-va-00001", recibo_ref: "recibo:cronos:1", estado: "denegado", version: 2, instante_utc: "2026-09-25T08:00:00Z", replay: false } };

test("la bandeja se pide por paso, mismo origen y sin identidad del cliente", async () => {
  const llamadas = [];
  const cliente = crearClienteResolucionCronosHTTP({ fetchImpl: async (url, opciones) => { llamadas.push({ url, opciones }); return respuestaJSON(200, BANDEJA); } });
  const b = await cliente.consultarBandeja({ paso: "responsable" });
  assert.equal(b.pendientes[0].empleado_etiqueta, "Persona sintética A");
  assert.equal(llamadas[0].url, `${RUTAS_RESOLUCION_CRONOS.bandeja}?paso=responsable`);
  assert.equal(llamadas[0].opciones.credentials, "same-origin");
  assert.equal(llamadas[0].opciones.redirect, "error");
  assert.equal(llamadas[0].opciones.referrerPolicy, "no-referrer");
  await assert.rejects(cliente.consultarBandeja({ paso: "jefatura" }), TypeError);
  await assert.rejects(crearClienteResolucionCronosHTTP({ fetchImpl: async () => respuestaJSON(200, BANDEJA) }).consultarBandeja({ paso: "administracion" }),
    (e) => e instanceof ErrorClienteResolucionCronos && e.codigo === "respuesta_incompatible", "una bandeja de otro paso no se acepta");
  const concedida = structuredClone(BANDEJA); concedida.pendientes[0].estado = "concedido";
  await assert.rejects(crearClienteResolucionCronosHTTP({ fetchImpl: async () => respuestaJSON(200, concedida) }).consultarBandeja({ paso: "responsable" }),
    (e) => e.codigo === "respuesta_incompatible");
});

test("la resolución envía sólo decisión, motivo, versión y clave y exige el recibo del estado que corresponde", async () => {
  let enviado;
  const cliente = crearClienteResolucionCronosHTTP({ fetchImpl: async (url, o) => { enviado = { url, ...o }; return respuestaJSON(201, RECIBO); } });
  const r = await cliente.resolver(ENTRADA);
  assert.equal(r.estado, "denegado");
  assert.equal(enviado.url, RUTAS_RESOLUCION_CRONOS.resoluciones);
  assert.deepEqual(JSON.parse(enviado.body), { clave_operacion: "res-va-00001", solicitud_ref: "permiso:cronos:solicitud:perm-va-00001", paso: "responsable",
    decision: "denegar", version_esperada: "1", motivo: "Coincide con el cierre" });
  for (const malo of [{ ...ENTRADA, motivo: undefined }, { ...ENTRADA, motivo: " con espacios" }, { ...ENTRADA, motivo: "a\nb" }, { ...ENTRADA, motivo: "x".repeat(501) },
    { ...ENTRADA, empleado_ref: "emp_x" }, { ...ENTRADA, version_esperada: 0 }, { ...ENTRADA, decision: "conceder" }]) {
    await assert.rejects(cliente.resolver(malo), TypeError);
  }
  const aprobar = crearClienteResolucionCronosHTTP({ fetchImpl: async () => respuestaJSON(201, RECIBO) });
  await assert.rejects(aprobar.resolver({ ...ENTRADA, decision: "aprobar", motivo: undefined }), (e) => e.codigo === "respuesta_incompatible",
    "un recibo denegado no confirma una conformidad");
});

test("los rechazos nominales llegan con su código y los demás son no disponible", async () => {
  for (const [estado, cuerpo, codigo] of [[403, { error: "no_competente" }, "no_competente"], [403, { error: "sin_empleado" }, "sin_empleado"],
    [403, { error: "acceso_denegado" }, "acceso_denegado"], [409, { error: "estado_cambiado" }, "estado_cambiado"], [409, { error: "conflicto" }, "conflicto"],
    [400, { error: "peticion_invalida" }, "peticion_invalida"], [503, { error: "no_disponible" }, "servicio_no_disponible"]]) {
    const cliente = crearClienteResolucionCronosHTTP({ fetchImpl: async () => respuestaJSON(estado, cuerpo) });
    await assert.rejects(cliente.resolver(ENTRADA), (e) => e instanceof ErrorClienteResolucionCronos && e.codigo === codigo, `${estado} ${cuerpo.error}`);
  }
});

test("avisos y archivo validan la respuesta completa", async () => {
  const avisos = { avisos: [{ aviso_ref: "aviso:cronos:00000000-0000-4000-8000-000000000001", solicitud_ref: "permiso:cronos:solicitud:perm-va-00001", estado: "denegado",
    motivo: "Cierre", resuelto_en: "2026-09-25T08:00:00Z", permiso_ref: "permiso:cronos:vacaciones", nombre: "Vacaciones", desde: "2026-10-05", hasta: "2026-10-07",
    cantidad: 3, unidad: "dia", archivado: false }] };
  const llamadas = [];
  const cliente = crearClienteResolucionCronosHTTP({ fetchImpl: async (url, o) => {
    llamadas.push({ url, o });
    if (url === RUTAS_RESOLUCION_CRONOS.avisos) return respuestaJSON(200, avisos);
    return respuestaJSON(201, { recibo: { archivo_ref: "aviso:cronos:archivo:arch-00000001", aviso_ref: avisos.avisos[0].aviso_ref, recibo_ref: "recibo:cronos:2", instante_utc: "2026-09-25T09:00:00Z", replay: false } });
  } });
  assert.equal((await cliente.consultarAvisos()).avisos.length, 1);
  const r = await cliente.archivarAviso({ clave_operacion: "arch-00000001", aviso_ref: avisos.avisos[0].aviso_ref });
  assert.equal(r.archivo_ref, "aviso:cronos:archivo:arch-00000001");
  assert.deepEqual(JSON.parse(llamadas[1].o.body), { clave_operacion: "arch-00000001", aviso_ref: avisos.avisos[0].aviso_ref });
  const sinMotivo = structuredClone(avisos); delete sinMotivo.avisos[0].motivo;
  await assert.rejects(crearClienteResolucionCronosHTTP({ fetchImpl: async () => respuestaJSON(200, sinMotivo) }).consultarAvisos(), (e) => e.codigo === "respuesta_incompatible");
  const archivadoSinFecha = structuredClone(avisos); archivadoSinFecha.avisos[0].archivado = true;
  await assert.rejects(crearClienteResolucionCronosHTTP({ fetchImpl: async () => respuestaJSON(200, archivadoSinFecha) }).consultarAvisos(), (e) => e.codigo === "respuesta_incompatible");
  await assert.rejects(cliente.archivarAviso({ clave_operacion: "arch-00000001", aviso_ref: "x" }), TypeError);
});

test("el cliente no guarda nada en el navegador", async () => {
  const fuente = await readFile(new URL("./cliente-resolucion-http.js", import.meta.url), "utf8");
  assert.doesNotMatch(fuente, /localStorage|sessionStorage|indexedDB|document\.cookie|credentials:\s*"include"/u);
});
