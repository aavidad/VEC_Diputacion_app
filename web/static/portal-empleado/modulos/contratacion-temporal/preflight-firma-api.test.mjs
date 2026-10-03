import assert from "node:assert/strict";
import test from "node:test";
import { crearClientePreflightFirma, validarPreflightFirma, RUTA_PREFLIGHT_FIRMA } from "./preflight-firma-api.js";

const solicitud = { expedienteRef: "expediente:prueba", version: 7, documento: "resolucion",
  originalRef: "original:prueba", originalVersion: 1 };
const datos = () => ({ esquema: "vec.contratacion-temporal.preflight-firma.v2", entrada_documento_ref: "original:prueba",
  entrada_documento_version: 1, entrada_documento_sha256: "b".repeat(64), version_expediente: 7, documento: "resolucion", catalogo_ref: "catalogo:prueba",
  catalogo_huella: "a".repeat(64), paso_pendiente: 1, original_ref: "original:prueba", original_version: 1,
  vias_disponibles: ["certificado_vec", "portafirmas_registro_rrhh"] });
const respuesta = (data, status = 200) => new Response(JSON.stringify({ data }), { status,
  headers: { "Content-Type": "application/json" } });

test("preflight solo envía referencias/versiones y conserva transporte cerrado", async () => {
  let captura;
  const signal = new AbortController().signal;
  const cliente = crearClientePreflightFirma({ fetchImpl: async (ruta, opciones) => {
    captura = { ruta, opciones }; return respuesta(datos());
  } });
  const result = await cliente.consultar({ ...solicitud, actor: "descartado" }, { signal });
  assert.equal(captura.ruta, RUTA_PREFLIGHT_FIRMA);
  assert.deepEqual(JSON.parse(captura.opciones.body), { expediente_ref: solicitud.expedienteRef, version_observada: 7,
    documento: "resolucion", original_ref: "original:prueba", original_version: 1 });
  for (const [campo, valor] of Object.entries({ method: "POST", mode: "same-origin", credentials: "same-origin",
    cache: "no-store", redirect: "error", referrerPolicy: "no-referrer", signal })) assert.equal(captura.opciones[campo], valor);
  assert.equal(Object.isFrozen(result.vias_disponibles), true);
});
test("desviación, vínculo cambiado, vías desconocidas o duplicadas nunca habilitan", () => {
  const alteraciones = [
    (d) => { d.actor = "ajeno"; }, (d) => { d.version_expediente = 8; }, (d) => { d.original_ref = "original:otro"; },
    (d) => { d.original_version = 2; }, (d) => { d.documento = "diligencia"; }, (d) => { d.catalogo_huella = "x"; },
    (d) => { d.vias_disponibles = ["certificado_vec", "certificado_vec"]; }, (d) => { d.vias_disponibles = ["firma"]; },
    (d) => { d.paso_pendiente = 0; }, (d) => { d.paso_pendiente = 17; }, (d) => { d.paso_pendiente = 3; },
    (d) => { d.paso_pendiente = 2; },
  ];
  for (const modificar of alteraciones) { const d = datos(); modificar(d); assert.equal(validarPreflightFirma(d, solicitud), null); }
  assert.deepEqual(validarPreflightFirma({ ...datos(), vias_disponibles: [] }, solicitud).vias_disponibles, []);
});
test("denegación, caída, conflicto y cuerpo no confiable son errores cerrados", async () => {
  for (const [fetchImpl, codigo] of [
    [async () => respuesta({}, 404), "acceso_denegado"], [async () => respuesta({}, 403), "acceso_denegado"],
    [async () => respuesta({}, 409), "conflicto"], [async () => respuesta({}, 503), "servicio_no_disponible"],
    [async () => { throw new Error("red"); }, "servicio_no_disponible"],
    [async () => respuesta({ ...datos(), perfil: "ajeno" }), "resultado_no_confiable"],
    [async () => new Response("x".repeat(17000), { headers: { "Content-Type": "application/json" } }), "resultado_no_confiable"],
    [async () => new Response("{}", { headers: { "Content-Type": "text/html" } }), "resultado_no_confiable"],
  ]) await assert.rejects(crearClientePreflightFirma({ fetchImpl }).consultar(solicitud), (e) => e.codigo === codigo);
});
test("cancelación y original ausente no emiten peticiones", async () => {
  let peticiones = 0;
  const cliente = crearClientePreflightFirma({ fetchImpl: async () => { peticiones++; return respuesta(datos()); } });
  await assert.rejects(cliente.consultar({ ...solicitud, originalRef: "" }), (e) => e.codigo === "contenido_no_valido");
  const c = new AbortController(); c.abort();
  await assert.rejects(cliente.consultar(solicitud, { signal: c.signal }), (e) => e.codigo === "operacion_abortada");
  assert.equal(peticiones, 0);
});
