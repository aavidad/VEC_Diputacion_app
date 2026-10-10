import test from "node:test";
import assert from "node:assert/strict";
import { crearGestorResultadoBolsa } from "./gestor-resultado-bolsa.js";

function escenario(cliente, confirmarOperacion = async () => true) {
  const eventos = new Map();
  const resultado = { vinculos: [], emisiones_vinculables: [{ bolsa_ref: "bolsa:prueba",
    llamamiento_ref: "llamamiento:prueba", recibo_emision_ref: "recibo:emision:prueba",
    referencia_visible: "2026/001", emitido_en: "2026-10-09T10:00:00Z" }],
  siguiente_cursor: "2026-10-09T10:00:00.000000Z#llamamiento:" + "a".repeat(64),
  total_vinculos: 21, personas_solicitadas: 1, aceptaciones_firmes: 0 };
  const expediente = { expediente_ref: "expediente:prueba", version: 4, resultado_bolsa: resultado };
  const estado = { vista: "expediente", carga: "listo", expediente_ref: expediente.expediente_ref, expediente };
  const zona = { innerHTML: "", querySelector: () => ({ after() {} }) };
  const raiz = { contains: () => true, querySelector: (selector) => selector === "[data-ct-resultado-bolsa-zona]" ? zona : null,
    ownerDocument: { createElement: () => ({ setAttribute() {}, textContent: "" }) },
    addEventListener: (nombre, funcion) => eventos.set(nombre, [...(eventos.get(nombre) ?? []), funcion]),
    removeEventListener: (nombre, funcion) => eventos.set(nombre, (eventos.get(nombre) ?? []).filter((f) => f !== funcion)) };
  const presentador = { obtenerEstado: () => estado, seleccionarExpediente: async () => {} };
  const gestor = crearGestorResultadoBolsa({ raiz, presentador, cliente, t: (clave) => clave,
    locale: "es-ES", zonaHoraria: "Europe/Madrid", confirmarOperacion,
    repintar() {}, generarClave: () => "clave-original-vinculo" });
  gestor.actualizar();
  const formulario = { querySelector: () => ({ value: "0" }) };
  const enviar = () => eventos.get("submit")[0]({ target: { closest: () => formulario }, preventDefault() {} });
  const pulsar = async (atributo) => {
    const boton = { hasAttribute: (a) => a === atributo };
    for (const f of eventos.get("click")) await f({ target: { closest: (s) => s.includes(`[${atributo}]`) ? boton : null }, preventDefault() {} });
  };
  return { estado, gestor, zona, enviar, pulsar, eventos };
}

test("un vínculo incierto conserva su petición y clave al comprobar de nuevo", async () => {
  const solicitudes = [];
  const cliente = { vincularLlamamientoBolsa: async (s) => {
    solicitudes.push(s);
    throw Object.assign(new Error("respuesta_incierta"), { resultadoIndeterminado: true });
  } };
  const e = escenario(cliente);
  await e.enviar();
  await e.pulsar("data-ct-bolsa-vinculo-reintentar");
  // El manejador de clic inicia la recuperación asíncrona.
  await Promise.resolve();
  assert.equal(solicitudes.length, 2);
  assert.equal(solicitudes[0], solicitudes[1]);
  assert.equal(solicitudes[1].clave_idempotencia, "clave-original-vinculo");
  e.gestor.desmontar();
});

test("la segunda página se solicita al pulsar y se cancela al cambiar de expediente", async () => {
  let opciones, consulta, completar;
  const cliente = { vincularLlamamientoBolsa() {}, consultarDetalleRRHH: (s, o) => {
    consulta = s; opciones = o;
    return new Promise((resolver) => { completar = resolver; });
  } };
  const e = escenario(cliente);
  assert.equal(consulta, undefined);
  const vuelo = e.pulsar("data-ct-bolsa-mas");
  await Promise.resolve();
  assert.equal(consulta.expediente_ref, "expediente:prueba");
  assert.equal(consulta.version_observada, 4);
  e.estado.expediente_ref = "expediente:otro";
  e.gestor.actualizar();
  assert.equal(opciones.signal.aborted, true);
  completar({ resumen: { expediente_ref: "expediente:prueba", version: 4 }, resultado_bolsa: {} });
  await vuelo;
  assert.doesNotMatch(e.zona.innerHTML, /pagina_bolsa_ajena/u);
  e.gestor.desmontar();
});

test("confirma mediante el descriptor que consume el portal y cancela sin escribir", async () => {
  let descriptor;
  let escrituras = 0;
  const e = escenario({ vincularLlamamientoBolsa: async () => { escrituras += 1; } }, async (entrada) => {
    descriptor = entrada;
    return false;
  });
  await e.enviar();
  assert.deepEqual(Object.keys(descriptor).sort(), ["advertencia", "referencia", "titulo"]);
  assert.equal(descriptor.titulo, "resultado_bolsa_vincular");
  assert.equal(descriptor.advertencia, "resultado_bolsa_confirmar");
  assert.equal(descriptor.referencia, "2026/001");
  assert.equal(escrituras, 0);
  e.gestor.desmontar();
});
