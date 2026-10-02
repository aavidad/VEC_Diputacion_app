import test from "node:test";
import assert from "node:assert/strict";
import { validarCircuitoRRHH, validarConsultaCircuitoRRHH } from "./contrato-circuito-rrhh.js";
import { crearClienteCircuitoRRHH, RUTA_CONSULTA_CIRCUITO_RRHH } from "./cliente-http-circuito-rrhh.js";
import { renderizarCircuitoRRHH } from "./vista-circuito-rrhh.js";
import { insertarConsultaCircuitoRRHH } from "./vista-expedientes.js";

const consulta = { expediente_ref: "expediente:prueba:rrhh", version_observada: 1 };
const flujo = { definicion_ref: "flujo:prueba:rrhh", version: 2, huella_sha256: "a".repeat(64) };
function datos() {
  return { flujo, circuito: { definicion: flujo, estado_actual: "solicitud", hitos: [] },
    version_expediente: 1, transiciones_permitidas: [] };
}
test("la consulta no admite identidad ni permisos enviados por la pantalla", () => {
  assert.throws(() => validarConsultaCircuitoRRHH({ ...consulta, actor_ref: "actor:ajeno" }));
  assert.throws(() => validarConsultaCircuitoRRHH({ ...consulta, perfil_ref: "perfil:direccion" }));
  assert.throws(() => validarConsultaCircuitoRRHH({ ...consulta, version_observada: 0 }));
  assert.throws(() => validarConsultaCircuitoRRHH({ ...consulta, expediente_ref: undefined }));
  assert.throws(() => validarConsultaCircuitoRRHH({ ...consulta, expediente_ref: null }));
});
test("la respuesta debe corresponder al flujo y no exponer evidencias personales", () => {
  const cambiado = datos(); cambiado.circuito.definicion = { ...flujo, version: 3 };
  assert.throws(() => validarCircuitoRRHH(cambiado, consulta));
  const personal = datos(); personal.circuito.hitos = [{ actor_ref: "persona:ajena" }];
  assert.throws(() => validarCircuitoRRHH(personal, consulta));
  assert.throws(() => validarCircuitoRRHH(datos(), { ...consulta, version_observada: 2 }));
  const adelantado = datos(); adelantado.version_expediente = 2;
  assert.throws(() => validarCircuitoRRHH(adelantado, consulta));
});
test("alta sin hitos se presenta pendiente, sin afirmar firmas ni trámites hechos", () => {
  const html = renderizarCircuitoRRHH(validarCircuitoRRHH(datos(), consulta));
  assert.match(html, /Todavía no constan actuaciones/u);
  assert.doesNotMatch(html, /✓|<li>/u);
});
test("el panel se inserta solo en fichas del flujo nuevo", () => {
  const inserciones = [];
  const raiz = { querySelector: (selector) => selector === "[data-ct-exp-ancla-firma]"
    ? { insertAdjacentHTML: (...args) => inserciones.push(args) } : null };
  assert.equal(insertarConsultaCircuitoRRHH(raiz, { expediente_ref: consulta.expediente_ref,
    version: 1, fases: [{ fase_ref: "fase:ct:solicitud" }] }), false);
  assert.equal(insertarConsultaCircuitoRRHH(raiz, { expediente_ref: consulta.expediente_ref,
    version: 1, fases: [{ fase_ref: "fase:ct:circuito_solicitud" }] }), true);
  assert.equal(inserciones.length, 1);
  assert.equal(inserciones[0][0], "beforebegin");
  assert.match(inserciones[0][1], /data-ct-circuito-expediente="expediente:prueba:rrhh"/u);
});
test("el cliente consulta por POST fijo sin identidad libre ni persistencia", async () => {
  let peticion;
  const cliente = crearClienteCircuitoRRHH({ fetchImpl: async (ruta, opciones) => {
    peticion = { ruta, opciones };
    return new Response(JSON.stringify({ data: datos() }), {
      status: 200, headers: { "Content-Type": "application/json" },
    });
  } });
  const resultado = await cliente.consultar(consulta);
  assert.equal(resultado.estado, "disponible");
  assert.equal(peticion.ruta, RUTA_CONSULTA_CIRCUITO_RRHH);
  assert.equal(peticion.opciones.method, "POST");
  assert.equal(peticion.opciones.credentials, "same-origin");
  assert.equal(peticion.opciones.cache, "no-store");
  assert.equal(peticion.opciones.redirect, "error");
  assert.equal(peticion.opciones.referrerPolicy, "no-referrer");
  assert.deepEqual(JSON.parse(peticion.opciones.body), consulta);
});
test("una denegación o una respuesta excesiva no presenta ningún hito", async () => {
  const denegado = crearClienteCircuitoRRHH({ fetchImpl: async () => new Response("{}", { status: 403 }) });
  assert.deepEqual(await denegado.consultar(consulta), { estado: "denegado" });
  const excesivo = crearClienteCircuitoRRHH({ fetchImpl: async () => new Response("x".repeat(128 * 1024 + 1), {
    status: 200, headers: { "Content-Type": "application/json" },
  }) });
  assert.deepEqual(await excesivo.consultar(consulta), { estado: "no_disponible" });
});
