import assert from "node:assert/strict";
import test from "node:test";

import { crearGestorTramitacion } from "./vista-expedientes-tramitacion.js";

function superficie() {
  const eventos = new Map();
  const zona = { innerHTML: "", eventos,
    addEventListener(tipo, funcion) { eventos.set(tipo, funcion); },
    removeEventListener(tipo) { eventos.delete(tipo); },
    replaceChildren() { this.innerHTML = ""; },
    reintentar() { eventos.get("click")?.({
      target: { closest: (selector) => selector === "[data-ct-exp-preparacion-reintentar]" ? {} : null },
      preventDefault() {},
    }); },
  };
  const alta = { innerHTML: "", replaceChildren() { this.innerHTML = ""; } };
  const raiz = { querySelector(selector) {
    if (selector === "[data-ct-exp-preparacion]") return zona;
    if (selector === "[data-ct-exp-alta]") return alta;
    return null;
  } };
  return { zona, alta, raiz };
}

function gestorPara(superficie, consultarPreparacion) {
  return crearGestorTramitacion({
    raiz: superficie.raiz,
    presentador: { obtenerEstado: () => ({ vista: "alta" }) },
    altaDisponible: true,
    alta: { catalogos: {}, ejecutor: async () => {}, consultarPreparacion },
    mensajes: {
      cobertura_preparacion_cargando: "Consultando requisitos",
      cobertura_preparacion_error: "No se pudieron consultar los requisitos",
      cobertura_preparacion_denegado: "Este perfil no puede consultar los requisitos",
      cobertura_preparacion_reintentar: "Reintentar consulta",
    },
  });
}

test("Nueva petición espera GET y aborta la lectura al salir", async () => {
  const superficieActual = superficie();
  let resolver;
  let signal;
  const gestor = gestorPara(superficieActual, ({ signal: recibido }) => {
    signal = recibido;
    return new Promise((continuar) => { resolver = continuar; });
  });
  gestor.montarAltaSiProcede();
  assert.match(superficieActual.zona.innerHTML, /Consultando requisitos/u);
  assert.equal(superficieActual.alta.innerHTML, "");
  assert.equal(signal.aborted, false);
  gestor.retirarComponentes();
  assert.equal(signal.aborted, true);
  resolver({});
  await Promise.resolve();
  assert.equal(superficieActual.zona.innerHTML, "");
  assert.equal(superficieActual.alta.innerHTML, "");
});

test("403 limpia la vista y reintenta solo la consulta", async () => {
  const superficieActual = superficie();
  let consultas = 0;
  const gestor = gestorPara(superficieActual, async () => {
    consultas += 1;
    throw Object.assign(new Error("denegada"), { estado: 403, envelopeValido: true });
  });
  gestor.montarAltaSiProcede();
  await new Promise((resolver) => setImmediate(resolver));
  assert.match(superficieActual.zona.innerHTML, /Este perfil no puede consultar/u);
  assert.doesNotMatch(superficieActual.zona.innerHTML, /data-ct-preparacion-vias/u);
  assert.equal(superficieActual.alta.innerHTML, "");
  superficieActual.zona.reintentar();
  await new Promise((resolver) => setImmediate(resolver));
  assert.equal(consultas, 2);
  assert.equal(superficieActual.alta.innerHTML, "");
  gestor.retirarComponentes();
});
