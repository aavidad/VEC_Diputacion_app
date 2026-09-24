import assert from "node:assert/strict";
import test from "node:test";
import { crearClienteAsignacionDietasHTTP } from "./cliente-asignacion-http.js";

const relacion = "rel_1234567890123456789012";
const unidad = "unidad:servicio:uno";
const asignacion = Object.freeze({
  asignacion: { asignacion_ref: "ads_1234567890123456789012", relacion_ref: relacion,
    persona_ref: "per_aaaaaaaaaaaaaaaaaaaaaa", unidad_ref: unidad, centro_ref: "centro:uno",
    administrativo_persona_ref: "per_bbbbbbbbbbbbbbbbbbbbbb", responsable_persona_ref: "per_cccccccccccccccccccccc",
    grupo_dieta: 2, vigente_desde: "2026-09-01", version: 3 },
  recibo_ref: `rad_${"a".repeat(32)}`, decision_ref: "decision:uno", efecto_ref: "efecto:uno",
  consumo_huella_sha256: "b".repeat(64), auditoria_ref: "auditoria:uno",
  registrada_en: "2026-09-24T09:00:00.000000Z", estado_local: "consultada",
});
const respuesta = (cuerpo, estado = 200) => new Response(JSON.stringify(cuerpo), {
  status: estado, headers: { "Content-Type": "application/json; charset=utf-8" },
});

test("consulta relaciones Personal y asignación exacta por relación, unidad y fecha acreditada", async () => {
  const llamadas = [];
  const cliente = crearClienteAsignacionDietasHTTP({ fetchImpl: async (ruta, opciones) => {
    llamadas.push({ ruta, opciones });
    return respuesta(ruta.endsWith("relaciones-dietas") ? {
      relaciones_autorizadas: [{ relacion_ref: relacion, unidad_ref: unidad, version: 4 }],
      fecha_referencia: "2026-09-24",
    } : asignacion);
  } });
  const relaciones = await cliente.obtenerRelaciones();
  const actual = await cliente.obtener(relaciones.relaciones_autorizadas[0].relacion_ref,
    relaciones.relaciones_autorizadas[0].unidad_ref, relaciones.fecha_referencia);
  assert.equal(actual.verificada, true);
  assert.equal(actual.grupo_dieta, 2);
  assert.equal(llamadas[0].ruta, "/api/vec/personal/relaciones-dietas");
  assert.equal(llamadas[1].ruta, `/api/vec/personal/asignaciones-dietas/${relacion}?fecha_referencia=2026-09-24&unidad_ref=unidad%3Aservicio%3Auno`);
  assert.equal(llamadas[1].opciones.credentials, "same-origin");
  assert.equal(llamadas[1].opciones.headers.Cookie, undefined);
});

test("la vista no acepta unidad inventada ni una asignación de otra relación", async () => {
  const cliente = crearClienteAsignacionDietasHTTP({ fetchImpl: async () => respuesta(asignacion) });
  await assert.rejects(() => cliente.obtener(relacion, "", "2026-09-24"), /consulta/u);
  await assert.rejects(() => cliente.obtener(relacion, unidad, "ayer"), /consulta/u);
  const ajena = crearClienteAsignacionDietasHTTP({ fetchImpl: async () => respuesta({
    ...asignacion, asignacion: { ...asignacion.asignacion, relacion_ref: "rel_9999999999999999999999" },
  }) });
  await assert.rejects(() => ajena.obtener(relacion, unidad, "2026-09-24"), /incompatible/u);
});

test("503 de Personal cierra asignación y abortar evita presentar respuesta tardía", async () => {
  const caida = crearClienteAsignacionDietasHTTP({ fetchImpl: async () => respuesta({ error: "personal.error.no_disponible" }, 503) });
  await assert.rejects(() => caida.obtener(relacion, unidad, "2026-09-24"), (error) => error.codigo === "no_disponible");
  const controlador = new AbortController();
  const tardio = crearClienteAsignacionDietasHTTP({ fetchImpl: async (_ruta, opciones) => {
    controlador.abort();
    assert.equal(opciones.signal.aborted, true);
    return respuesta(asignacion);
  } });
  await assert.rejects(() => tardio.obtener(relacion, unidad, "2026-09-24", { signal: controlador.signal }), /cancelada/u);
});
