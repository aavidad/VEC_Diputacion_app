import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { crearCoordinadorModulosPortal, rutaDeVistaPortal } from "./portal-modulos-coordinador.js";
import { VISTAS_INTERNAS_BOLSA } from "./portal-menu-bolsa.js";

const vistasSeleccion = ["seleccion-inscripciones", "seleccion-pruebas", "seleccion-comunicaciones"];

test("Selección queda fuera de las rutas productivas hasta disponer de fuente autorizada", async () => {
  const montajes = [];
  const raiz = { innerHTML: "", replaceChildren() { this.innerHTML = ""; } };
  const coordinador = crearCoordinadorModulosPortal({
    escaparHTML: String,
    montajeBolsa: {
      disponible: () => true,
      montar: ({ vista }) => { montajes.push(vista); return { desmontar() {} }; },
    },
  });
  for (const vista of vistasSeleccion) {
    assert.equal(VISTAS_INTERNAS_BOLSA.includes(vista), false);
    assert.equal(rutaDeVistaPortal(vista), "#portal");
    assert.equal(await coordinador.montarVista(vista, raiz), false);
  }
  assert.deepEqual(montajes, []);
});

test("el shell no publica rutas, estilos ni montajes de Selección sin fuente", async () => {
  const [portal, html] = await Promise.all([
    readFile(new URL("portal.js", import.meta.url), "utf8"),
    readFile(new URL("index.html", import.meta.url), "utf8"),
  ]);
  for (const vista of vistasSeleccion) assert.doesNotMatch(html, new RegExp(`data-vista="${vista}"`));
  assert.doesNotMatch(html, /modulos\/seleccion\/(?:inscripciones|pruebas|comunicaciones)\/[^" ]+\.css/u);
  assert.doesNotMatch(portal, /montarVista(?:Inscripciones|Pruebas|SeleccionComunicaciones)/u);
  assert.match(portal, /if \(vista\.startsWith\("seleccion-"\)\) return false/u);
  assert.match(portal, /renderizarContratos\(\{ contratos_fuente: \{ estado: "no_configurado" \} \}\)/u);
});
