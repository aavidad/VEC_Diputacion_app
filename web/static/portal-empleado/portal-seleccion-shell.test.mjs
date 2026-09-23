import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { crearCoordinadorModulosPortal, rutaDeVistaPortal } from "./portal-modulos-coordinador.js";
import { categoriaDeVistaBolsa, VISTAS_INTERNAS_BOLSA } from "./portal-menu-bolsa.js";
import { traducirPortal } from "./portal-i18n.js";
import { renderizarVistaInscripciones } from "./modulos/seleccion/inscripciones/vista.js";
import { renderizarVistaPruebas } from "./modulos/seleccion/pruebas/vista.js";
import { renderizarVistaSeleccionComunicaciones } from "./modulos/seleccion/comunicaciones/vista.js";

const vistas = ["seleccion-inscripciones", "seleccion-pruebas", "seleccion-comunicaciones", "contratos"];

test("las cuatro rutas internas usan el coordinador común y desmontan una sola vez", async () => {
  const limpiezas = [];
  const montajes = [];
  const raiz = { innerHTML: "", replaceChildren() { this.innerHTML = ""; } };
  const coordinador = crearCoordinadorModulosPortal({
    escaparHTML: String,
    montajeBolsa: {
      disponible: (vista) => vistas.includes(vista),
      montar: ({ vista }) => {
        montajes.push(vista);
        return { desmontar: () => limpiezas.push(vista) };
      },
    },
  });
  for (const vista of vistas) {
    assert.ok(VISTAS_INTERNAS_BOLSA.includes(vista));
    assert.equal(categoriaDeVistaBolsa(vista), vista === "contratos" ? "contratos" : "bolsas-candidatos");
    assert.equal(rutaDeVistaPortal(vista), `#bolsa/${vista}`);
    assert.equal(await coordinador.montarVista(vista, raiz), true);
  }
  coordinador.desmontarVistaActual();
  coordinador.desmontarVistaActual();
  assert.deepEqual(montajes, vistas);
  assert.deepEqual(limpiezas, vistas);
});

test("Selección llega sin fuente y sin filas ni efectos inventados", () => {
  const html = [
    renderizarVistaInscripciones({ estado: "no_configurado" }),
    renderizarVistaPruebas({ estado: "no_configurado" }),
    renderizarVistaSeleccionComunicaciones({ estadoFuente: "no_configurado" }),
  ];
  for (const vista of html) {
    assert.match(vista, /configurad|fuente pendiente|sin fuente|pendiente de conexión/i);
    assert.doesNotMatch(vista, /DEMO-|data-inscripciones-detalle="0"|data-prueba-detalle=|data-s6-seleccionar=/);
  }
  assert.match(html[1], /disabled/);
  assert.match(html[2], /disabled/);
});

test("el shell conecta los tres montajes locales y Contratos solo consume contratos_fuente", async () => {
  const [portal, html] = await Promise.all([
    readFile(new URL("portal.js", import.meta.url), "utf8"),
    readFile(new URL("index.html", import.meta.url), "utf8"),
  ]);
  for (const [vista, montaje] of [
    ["seleccion-inscripciones", "montarVistaInscripciones"],
    ["seleccion-pruebas", "montarVistaPruebas"],
    ["seleccion-comunicaciones", "montarVistaSeleccionComunicaciones"],
  ]) {
    assert.match(portal, new RegExp(`vista === "${vista}"[\\s\\S]*?${montaje}\\(`));
    assert.match(html, new RegExp(`data-vista="${vista}"`));
  }
  for (const clave of ["seleccion_inscripciones_titulo", "seleccion_pruebas_titulo",
    "seleccion_comunicaciones_titulo", "contratos_consulta_estado"]) {
    assert.match(html, new RegExp(`data-i18n-portal="${clave}"`));
    assert.ok(traducirPortal(clave));
  }
  assert.match(portal, /renderizarContratos\(\{ contratos_fuente: \{ estado: "no_configurado" \} \}\)/);
  assert.doesNotMatch(portal, /renderizarContratos\(\{[^}]*contratos:\s*DATOS_PANEL/);
  assert.match(portal, /VISTAS_BOLSA_SIN_LECTURA = new Set\(\[[\s\S]*?"contratos", "seleccion-inscripciones", "seleccion-pruebas", "seleccion-comunicaciones"/);
});
