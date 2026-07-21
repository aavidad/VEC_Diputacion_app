import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { ErrorAPIBorradores } from "./portal-borradores-api.js";
import { crearControlAccesoBorradores } from "./portal-borradores-acceso.js";
import { opciones } from "./portal-borradores-fixtures.test-helper.mjs";

function diferida() {
  let resolver;
  const promesa = new Promise((resolve) => { resolver = resolve; });
  return { promesa, resolver };
}

function crearControl(consultarOpciones) {
  const cambios = [];
  const control = crearControlAccesoBorradores({
    consultarOpciones,
    alCambiar: (acceso) => cambios.push(acceso),
  });
  return { cambios, control };
}

test("la disponibilidad consulta solo opciones y habilita directamente Elaboración", async () => {
  let consultas = 0;
  const { cambios, control } = crearControl(async ({ signal }) => {
    consultas += 1;
    assert.equal(signal instanceof AbortSignal, true);
    return structuredClone(opciones());
  });
  assert.deepEqual(control.obtenerAcceso(), {
    disponible: false,
    vista: "",
    estado: "cargando",
    etiqueta: "Comprobando acceso a borradores",
  });
  assert.equal(await control.comprobar(), true);
  assert.equal(consultas, 1);
  assert.deepEqual(control.obtenerAcceso(), {
    disponible: true,
    vista: "elaboracion",
    estado: "disponible",
    etiqueta: "Borradores disponibles",
  });
  assert.deepEqual(control.obtenerOpciones(), opciones());
  assert.deepEqual(cambios.map(({ estado }) => estado), ["cargando", "disponible"]);
});

test("dos consumidores comparten una única comprobación en curso", async () => {
  const pendiente = diferida();
  let consultas = 0;
  const { control } = crearControl(async () => { consultas += 1; return pendiente.promesa; });
  const primera = control.comprobar();
  const segunda = control.comprobar();
  assert.equal(primera, segunda);
  assert.equal(consultas, 1);
  pendiente.resolver(structuredClone(opciones()));
  assert.equal(await primera, true);
});

test("una capacidad no concedida o un 403 permanecen denegados y sin reintento engañoso", async () => {
  const sinCapacidad = structuredClone(opciones());
  sinCapacidad.capacidades.consultar = false;
  const primerControl = crearControl(async () => sinCapacidad).control;
  assert.equal(await primerControl.comprobar(), false);
  assert.deepEqual(primerControl.obtenerAcceso(), {
    disponible: false,
    vista: "",
    estado: "denegado",
    etiqueta: "Sin permiso para gestionar borradores",
  });
  assert.equal(primerControl.obtenerError().estado, 403);
  assert.equal(primerControl.obtenerOpciones(), null);

  const segundoControl = crearControl(async () => {
    throw new ErrorAPIBorradores("Autorización denegada.", 403, undefined, {
      codigo: "autorizacion_denegada",
    });
  }).control;
  assert.equal(await segundoControl.comprobar(), false);
  assert.equal(segundoControl.obtenerAcceso().estado, "denegado");
});

test("un fallo técnico se diferencia de una denegación y puede reintentarse", async () => {
  let intentos = 0;
  const { control } = crearControl(async () => {
    intentos += 1;
    if (intentos === 1) throw new ErrorAPIBorradores("No disponible.", 503);
    return structuredClone(opciones());
  });
  assert.equal(await control.comprobar(), false);
  assert.deepEqual(control.obtenerAcceso(), {
    disponible: false,
    vista: "",
    estado: "error",
    etiqueta: "Servicio de borradores no disponible",
    reintentar: true,
  });
  assert.equal(await control.comprobar({ forzar: true }), true);
  assert.equal(intentos, 2);
  assert.equal(control.obtenerAcceso().vista, "elaboracion");
});

test("un 401 o 403 de otra operación invalida opciones y comprobaciones en curso", async () => {
  const pendiente = diferida();
  let signal;
  const { control } = crearControl(async (opcionesConsulta) => {
    signal = opcionesConsulta.signal;
    return pendiente.promesa;
  });
  const comprobacion = control.comprobar();
  assert.equal(control.invalidar(new ErrorAPIBorradores("Sesión retirada.", 401)), true);
  assert.equal(signal.aborted, true);
  assert.equal(control.obtenerAcceso().estado, "denegado");
  assert.equal(control.obtenerOpciones(), null);
  pendiente.resolver(structuredClone(opciones()));
  assert.equal(await comprobacion, false);
  assert.equal(control.invalidar(new ErrorAPIBorradores("Fallo técnico.", 503)), false);
  assert.equal(control.obtenerAcceso().estado, "denegado");
});

test("el control no contiene fallback DEMO, cookies ni almacenamiento local", async () => {
  const fuente = await readFile(new URL("portal-borradores-acceso.js", import.meta.url), "utf8");
  assert.doesNotMatch(fuente, /datos-presentacion|portal-borradores-demo|presentacion=rrhh/);
  assert.doesNotMatch(fuente, /document\.cookie|localStorage|sessionStorage/);
  assert.doesNotMatch(fuente, /listar\s*\(/, "la comprobación no debe leer la bandeja");
});
