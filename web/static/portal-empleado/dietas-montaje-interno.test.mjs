import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { crearCoordinadorModulosPortal } from "./portal-modulos-coordinador.js";
import { versionDe } from "./versiones-cache.test-helper.mjs";

test("el catálogo interno monta Dietas con clientes HTTP, ruta y mapa, sin dependencias demo", async () => {
  const recibidas = [];
  const transporte = () => {};
  const clienteBorradores = Object.freeze({ tipo: "borradores" });
  const clienteAsignacion = Object.freeze({ tipo: "asignacion", async obtenerRelaciones() {
    return { relaciones_autorizadas: [], fecha_referencia: "2026-09-25" };
  } });
  const calculador = Object.freeze({ tipo: "calculador" });
  const visor = Object.freeze({ tipo: "visor" });
  const coordinador = crearCoordinadorModulosPortal({
    escaparHTML: String,
    entorno: { fetch: transporte },
    cargarCatalogoInterno: async () => Object.freeze([{ clave: "dietas" }]),
    cargadoresInternos: {
      contratacion_temporal: async () => { throw new Error("no se debe cargar CT"); },
      dietas: async () => Object.freeze({
        contrato: Object.freeze({}),
        clienteBorradores: { crearClienteBorradoresDietasHTTP({ fetchImpl }) {
          assert.equal(typeof fetchImpl, "function");
          return clienteBorradores;
        } },
        clienteAsignacion: { crearClienteAsignacionDietasHTTP({ fetchImpl }) {
          assert.equal(typeof fetchImpl, "function");
          return clienteAsignacion;
        } },
        calculador: { crearCalculadorRutasDietasHTTP({ fetchImpl }) {
          assert.equal(typeof fetchImpl, "function");
          return calculador;
        } },
        mapa: { crearVisorRutaDietas({ permitirTeselas }) {
          assert.equal(permitirTeselas, true);
          return visor;
        } },
        recorridos: { montarVistaRecorridosDietas(raiz, opciones) {
          recibidas.push({ raiz, opciones });
          return Object.freeze({ desmontar() {} });
        } },
      }),
    },
  });
  await coordinador.cargarInterno();
  assert.equal(coordinador.vistaDisponible("dietas"), true);
  const raiz = { innerHTML: "", replaceChildren() {} };
  assert.equal(await coordinador.montarVista("dietas", raiz), true);
  assert.equal(recibidas.length, 1);
  assert.equal(recibidas[0].raiz, raiz);
  assert.strictEqual(recibidas[0].opciones.clienteBorradores, clienteBorradores);
  assert.strictEqual(recibidas[0].opciones.clienteAsignacion, clienteAsignacion);
  assert.equal(Object.hasOwn(recibidas[0].opciones, "montarItinerario"), false);
  assert.strictEqual(recibidas[0].opciones.calculadorRuta, calculador);
  assert.strictEqual(recibidas[0].opciones.visorRuta, visor);
  assert.equal(recibidas[0].opciones.estadoRelaciones, "disponible");
});

test("el cargador interno importa la ruta real y ningún dato de presentación", async () => {
  const [coordinador, composicion] = await Promise.all([
    readFile(new URL("./portal-modulos-coordinador.js", import.meta.url), "utf8"),
    readFile(new URL("./portal-composicion-empleado.js", import.meta.url), "utf8"),
  ]);
  const interno = coordinador.split("const CARGADORES_INTERNOS_PREDETERMINADOS =")[1]
    .split("function componerModuloAislado")[0];
  for (const recurso of ["cliente-borradores-http", "cliente-asignacion-http", "calculador-rutas-http", "mapa-ruta"])
    assert.ok(versionDe(interno, `./modulos/dietas/${recurso}.js`), recurso);
  assert.doesNotMatch(interno, /calculador-rutas-presentacion|vista-itinerario|datos-presentacion|adaptador-presentacion/u);
  assert.doesNotMatch(composicion, /datos-sinteticos-rrhh|componerDietasVisible|centroSalidaSintetico/u);
});
