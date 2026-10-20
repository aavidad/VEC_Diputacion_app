import test from "node:test";
import assert from "node:assert/strict";
import { crearGestorResultadoBolsa } from "./gestor-resultado-bolsa.js";

function escenario(cliente, confirmarOperacion = async () => true, consultarPersonas = async () => new Map(),
  { zona = { innerHTML: "", querySelector: () => ({ after() {} }) }, documento = {} } = {}) {
  const eventos = new Map();
  const resultado = { vinculos: [], emisiones_vinculables: [{ bolsa_ref: "bolsa:prueba",
    llamamiento_ref: "llamamiento:prueba", recibo_emision_ref: "recibo:emision:prueba",
    referencia_visible: "2026/001", emitido_en: "2026-10-09T10:00:00Z" }],
  siguiente_cursor: "2026-10-09T10:00:00.000000Z#llamamiento:" + "a".repeat(64),
  total_vinculos: 21, personas_solicitadas: 1, aceptaciones_firmes: 0 };
  const expediente = { expediente_ref: "expediente:prueba", version: 4, resultado_bolsa: resultado };
  const estado = { vista: "expediente", carga: "listo", expediente_ref: expediente.expediente_ref, expediente };
  const raiz = { contains: () => true, querySelector: (selector) => selector === "[data-ct-resultado-bolsa-zona]" ? zona : null,
    ownerDocument: Object.assign(documento, { createElement: () => ({ setAttribute() {}, textContent: "" }) }),
    addEventListener: (nombre, funcion) => eventos.set(nombre, [...(eventos.get(nombre) ?? []), funcion]),
    removeEventListener: (nombre, funcion) => eventos.set(nombre, (eventos.get(nombre) ?? []).filter((f) => f !== funcion)) };
  const presentador = { obtenerEstado: () => estado, seleccionarExpediente: async () => {} };
  const gestor = crearGestorResultadoBolsa({ raiz, presentador, cliente, t: (clave) => clave,
    locale: "es-ES", zonaHoraria: "Europe/Madrid", confirmarOperacion,
    repintar() {}, generarClave: () => "clave-original-vinculo", consultarPersonas });
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

test("el selector y la confirmación dicen a quién se llamó cuando Bolsa lo devuelve", async () => {
  const advertencias = [];
  let pedidas = 0;
  const consultar = async (llamamientos) => {
    pedidas += 1;
    assert.deepEqual(llamamientos.map((l) => l.llamamiento_ref), ["llamamiento:prueba"]);
    return new Map([["llamamiento:prueba", ["Antonio Reyes Álvarez"]]]);
  };
  const e = escenario({ vincularLlamamientoBolsa: async () => ({}) }, async (opciones) => {
    advertencias.push(opciones.advertencia); return false;
  }, consultar);
  await new Promise((resolver) => setTimeout(resolver, 0));
  assert.match(e.zona.innerHTML, /Antonio Reyes Álvarez · .* · 2026\/001/u);
  e.gestor.actualizar();
  assert.equal(pedidas, 1);
  await e.enviar();
  assert.deepEqual(advertencias, ["resultado_bolsa_confirmar_persona"]);
  e.gestor.desmontar();
});

test("si Bolsa no responde, el selector sigue con referencia y fecha", async () => {
  const e = escenario({ vincularLlamamientoBolsa() {} }, async () => false, async () => { throw new Error("403"); });
  await new Promise((resolver) => setTimeout(resolver, 0));
  assert.match(e.zona.innerHTML, /2026\/001 · /u);
  assert.doesNotMatch(e.zona.innerHTML, /Antonio/u);
  e.gestor.desmontar();
});

test("al llegar los nombres se conserva la opción elegida y el foco del desplegable", async () => {
  // Zona mínima que, como el navegador, crea un desplegable nuevo en cada pintado.
  const documento = { activeElement: null };
  let html = "";
  const crearSelect = () => ({ value: "", matches: (s) => s === "#ct-bolsa-emision", focus() { documento.activeElement = this; } });
  let select = crearSelect();
  const zona = { get innerHTML() { return html; }, set innerHTML(valor) { html = valor; select = crearSelect(); },
    contains: (nodo) => nodo === select,
    querySelector: (s) => (s === "#ct-bolsa-emision" ? select : { after() {} }) };
  let completar;
  const e = escenario({ vincularLlamamientoBolsa() {} }, async () => false,
    () => new Promise((resolver) => { completar = resolver; }), { zona, documento });
  select.value = "0";
  select.focus();
  completar(new Map([["llamamiento:prueba", ["Antonio Reyes Álvarez"]]]));
  await new Promise((resolver) => setTimeout(resolver, 0));
  assert.match(html, /Antonio Reyes Álvarez/u);
  assert.equal(select.value, "0");
  assert.equal(documento.activeElement, select);
  e.gestor.desmontar();
});

test("los nombres ya leídos no se vuelven a pedir y uno que falló se pide en el siguiente pintado", async () => {
  const pedidas = [];
  let fallar = true;
  const consultar = async (llamamientos) => {
    pedidas.push(llamamientos.map((l) => l.llamamiento_ref));
    if (fallar) throw new Error("503");
    return new Map([["llamamiento:prueba", ["Antonio Reyes Álvarez"]]]);
  };
  const e = escenario({ vincularLlamamientoBolsa() {} }, async () => false, consultar);
  await new Promise((resolver) => setTimeout(resolver, 0));
  assert.doesNotMatch(e.zona.innerHTML, /Antonio/u);
  fallar = false;
  e.gestor.actualizar();
  await new Promise((resolver) => setTimeout(resolver, 0));
  assert.match(e.zona.innerHTML, /Antonio Reyes Álvarez/u);
  e.gestor.actualizar();
  e.gestor.actualizar();
  assert.deepEqual(pedidas, [["llamamiento:prueba"], ["llamamiento:prueba"]]);
  e.gestor.desmontar();
});
