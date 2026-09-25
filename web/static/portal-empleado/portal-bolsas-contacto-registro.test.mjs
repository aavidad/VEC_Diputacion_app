import assert from "node:assert/strict";
import test from "node:test";
import {
  crearControladorRegistroContacto, registrarContacto, renderizarRegistroContacto, rutaRegistroContacto, validarComandoContacto,
} from "./portal-bolsas-contacto-registro.js";

const valido = { correo: "persona@ejemplo.es", telefono_1: "600000001", telefono_2: "", motivo: "Corrección tras llamada", origen: "convoca" };

test("valida el comando como el dominio: correo o teléfono, motivo y origen admitido", () => {
  assert.equal(validarComandoContacto(valido), "");
  assert.equal(validarComandoContacto({ ...valido, origen: "" }), "");
  assert.notEqual(validarComandoContacto({ ...valido, origen: "persona" }), "");
  assert.notEqual(validarComandoContacto({ ...valido, correo: "", telefono_1: "" }), "");
  assert.notEqual(validarComandoContacto({ ...valido, motivo: "" }), "");
  assert.notEqual(validarComandoContacto({ ...valido, telefono_1: "500000000" }), "");
  assert.notEqual(validarComandoContacto({ ...valido, telefono_2: "600000001" }), "");
  assert.equal(rutaRegistroContacto("bolsa:1", "part:1"), "/api/vec/bolsa/bolsas/bolsa:1/candidatos/part:1/datos-contacto");
});

test("envía con clave idempotente y traduce los errores sin guardar nada", async () => {
  let peticion;
  const fetchImpl = async (ruta, opciones) => { peticion = { ruta, opciones }; return { status: 201, json: async () => ({ data: { version: 2, recibo_ref: "recibo:c2", reutilizada: false } }) }; };
  const r = await registrarContacto("bolsa:1", "part:1", valido, "contacto-1", { fetchImpl });
  assert.equal(r.ok, true);
  assert.equal(peticion.opciones.headers["Idempotency-Key"], "contacto-1");
  assert.deepEqual(JSON.parse(peticion.opciones.body), valido);
  const conflicto = await registrarContacto("bolsa:1", "part:1", valido, "contacto-1", { fetchImpl: async () => ({ status: 422, json: async () => ({}) }) });
  assert.match(conflicto.mensaje, /regla/u);
  const invalido = await registrarContacto("bolsa:1", "part:1", { ...valido, motivo: "" }, "contacto-1", { fetchImpl: () => assert.fail("no debe enviar") });
  assert.equal(invalido.ok, false);
});

test("pinta el botón y el formulario con origen propio o CONVOCA, escapando valores", () => {
  assert.match(renderizarRegistroContacto({ estado: {} }), /data-contacto-rrhh-accion="abrir"/u);
  const formulario = renderizarRegistroContacto({ estado: { abierto: true, formulario: { correo: "<x>", origen: "convoca" }, error: "Mal" } });
  assert.match(formulario, /value="convoca" selected/u);
  assert.match(formulario, /role="alert"/u);
  assert.doesNotMatch(formulario, /<x>/u);
});

test("el controlador conserva la clave en el reintento y recarga el origen al registrar", async () => {
  const claves = [];
  let recargado = 0;
  const modal = { candidato: { participacion_ref: "part:1" } };
  const estado = { modalFicha: modal, bolsaSeleccionada: "bolsa:1" };
  let fallar = true;
  const fetchImpl = async (_ruta, opciones) => {
    claves.push(opciones.headers["Idempotency-Key"]);
    if (fallar) throw new Error("corte");
    return { status: 201, json: async () => ({ data: { version: 3, recibo_ref: "recibo:c3", reutilizada: false } }) };
  };
  const c = crearControladorRegistroContacto({ estado, renderizar: () => {}, fetchImpl, alRegistrar: async () => { recargado++; } });
  c.manejarClick({ target: { closest: () => ({ dataset: { contactoRrhhAccion: "abrir" } }) } });
  const datos = new Map(Object.entries(valido));
  const formulario = {};
  const evento = { target: { closest: () => formulario } };
  globalThis.FormData = class { constructor() { this.m = datos; } get(k) { return this.m.get(k); } };
  c.manejarSubmit(evento);
  await new Promise((r) => setTimeout(r, 0));
  assert.match(modal.registroContacto.error, /No se pudo/u);
  fallar = false;
  c.manejarSubmit(evento);
  await new Promise((r) => setTimeout(r, 0));
  assert.equal(claves.length, 2);
  assert.equal(claves[0], claves[1]);
  assert.equal(recargado, 1);
  assert.match(modal.registroContacto.exito, /versión 3/u);
  assert.equal(modal.registroContacto.abierto, false);
});
