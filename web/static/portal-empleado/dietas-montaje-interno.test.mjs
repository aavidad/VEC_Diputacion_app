import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { crearCoordinadorModulosPortal } from "./portal-modulos-coordinador.js";

test("el catálogo interno monta Dietas con ambos clientes HTTP y sin dependencias demo", async () => {
  const recibidas = [];
  const transporte = () => {};
  const clienteBorradores = Object.freeze({ tipo: "borradores" });
  const clienteAsignacion = Object.freeze({ tipo: "asignacion" });
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
  assert.equal(Object.hasOwn(recibidas[0].opciones, "calculador"), false);
  assert.equal(Object.hasOwn(recibidas[0].opciones, "visorRuta"), false);
});

test("el cargador interno no importa la ruta ni los datos de presentación", async () => {
  const [coordinador, composicion] = await Promise.all([
    readFile(new URL("./portal-modulos-coordinador.js", import.meta.url), "utf8"),
    readFile(new URL("./portal-composicion-empleado.js", import.meta.url), "utf8"),
  ]);
  const interno = coordinador.split("const CARGADORES_INTERNOS_PREDETERMINADOS =")[1]
    .split("function componerModuloAislado")[0];
  assert.match(interno, /cliente-borradores-http\.js\?v=20260924-dietas-montaje-v1/u);
  assert.match(interno, /cliente-asignacion-http\.js\?v=20260924-dietas-montaje-v1/u);
  assert.doesNotMatch(interno, /calculador-rutas-presentacion|vista-itinerario|mapa-ruta/u);
  assert.doesNotMatch(composicion, /datos-sinteticos-rrhh|componerDietasVisible|centroSalidaSintetico/u);
});
