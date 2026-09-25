import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { TRAMITES_AYUDANTE_PORTAL } from "./ayuda-contenido.js";
import {
  CLAVES_SIN_ENTRADA_PORTAL,
  crearCoordinadorModulosPortal,
  vistaConEntradaPortal,
} from "./portal-modulos-coordinador.js";
import { crearVistaInicioPortal } from "./portal-inicio.js";

// El portal solo ofrece Bolsa y la contratación temporal: Personal, Cronos y
// Dietas se siguen cargando (su URL directa funciona), pero no tienen entrada
// en el menú lateral, en Inicio ni en el ayudante de trámites.

const OCULTOS = ["personal", "cronos", "dietas"];
const modulo = (clave, sigla) => Object.freeze({ clave, sigla, titulo: `Módulo ${sigla}`, texto: `Texto ${sigla}` });
const CATALOGO = Object.freeze([
  modulo("personal", "PER"), modulo("cronos", "CRO"), modulo("dietas", "DIE"),
  modulo("bolsa", "BOL"), modulo("contratacion_temporal", "CT"),
]);
const esperarTurnos = async (turnos = 5) => {
  for (let i = 0; i < turnos; i += 1) await new Promise((seguir) => setImmediate(seguir));
};
const sinTemporizador = Object.freeze({ setTimeout: () => 0, clearTimeout() {} });

function coordinadorConModulosCargando() {
  // Personal, Cronos y Dietas quedan «cargando»: sin el filtro se ofrecerían.
  const nunca = () => new Promise(() => {});
  return crearCoordinadorModulosPortal({
    escaparHTML: String,
    cargarCatalogoInterno: async () => CATALOGO,
    entorno: { fetch: async () => { throw new Error("sin red"); } },
    temporizadores: sinTemporizador,
    cargadoresInternos: {
      contratacion_temporal: async () => { throw new Error("sin CT"); },
      personal: nunca, cronos: nunca, dietas: nunca,
    },
  });
}

test("Personal, Cronos y Dietas no tienen entrada propia en el portal", () => {
  assert.deepEqual([...CLAVES_SIN_ENTRADA_PORTAL].sort(), [...OCULTOS].sort());
  for (const vista of ["personal", "personal-registro", "cronos", "cronos-permisos", "dietas"])
    assert.equal(vistaConEntradaPortal(vista), false, vista);
  for (const vista of ["contratacion-temporal", "resumen", "llamamientos"])
    assert.equal(vistaConEntradaPortal(vista), true, vista);
});

test("el menú lateral e Inicio solo ofrecen Bolsa y la contratación temporal", async () => {
  const coordinador = coordinadorConModulosCargando();
  void coordinador.cargarInterno().catch(() => {});
  await esperarTurnos();
  // El módulo sigue en carga (su vista directa se conserva), pero no se ofrece.
  assert.equal(coordinador.resolverAcceso("cronos").estado, "cargando");
  assert.deepEqual(coordinador.obtenerCatalogo().map(({ clave }) => clave), ["bolsa", "contratacion_temporal"]);

  const menu = coordinador.renderizarNavegacion(true);
  for (const clave of OCULTOS) assert.doesNotMatch(menu, new RegExp(`data-modulo-portal="${clave}"`, "u"));

  for (const esPerfilRRHH of [() => false, () => true]) {
    const inicio = crearVistaInicioPortal({
      encabezadoVista: (_s, titulo) => `<h2>${titulo}</h2>`,
      escaparHTML: String,
      obtenerCatalogo: coordinador.obtenerCatalogo,
      resolverAcceso: (clave) => coordinador.resolverAcceso(clave),
      esPerfilRRHH,
    })();
    for (const clave of OCULTOS) {
      assert.doesNotMatch(inicio, new RegExp(`data-modulo-catalogo="${clave}"`, "u"));
      assert.doesNotMatch(inicio, new RegExp(`data-vista="${clave}`, "u"));
    }
    assert.doesNotMatch(inicio, /Módulo (PER|CRO|DIE)/u);
  }
});

test("el ayudante de trámites y el resumen de accesos no ofrecen los módulos ocultos", async () => {
  const ofrecidos = TRAMITES_AYUDANTE_PORTAL.filter((tramite) => vistaConEntradaPortal(tramite.vista));
  assert.ok(ofrecidos.length > 0, "Bolsa conserva sus trámites guiados");
  for (const tramite of ofrecidos) {
    assert.doesNotMatch(tramite.vista, /^(personal|cronos|dietas)/u, tramite.id);
    for (const paso of tramite.pasos) if (paso.vista) assert.equal(vistaConEntradaPortal(paso.vista), true, tramite.id);
  }
  const portal = await readFile(new URL("./portal.js", import.meta.url), "utf8");
  assert.match(portal, /TRAMITES_AYUDANTE_PORTAL\.filter\(\(tramite\) => vistaConEntradaPortal\(tramite\.vista\)\)/u);
  assert.match(portal, /\.filter\(\(clave\) => !CLAVES_SIN_ENTRADA_PORTAL\.includes\(clave\)\)/u);
});
