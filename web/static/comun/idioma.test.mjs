import test from "node:test";
import assert from "node:assert/strict";
import { cambiarIdioma, montarSelectorIdioma, seleccionarIdioma } from "./idioma.js";

test("la elección explícita de interfaz prevalece sobre el navegador (índice del repositorio)", () => {
  assert.equal(seleccionarIdioma("en", ["es-ES"]), "en");
  assert.equal(seleccionarIdioma("fr", ["en-GB", "es-ES"]), "en");
  assert.equal(seleccionarIdioma("", ["fr-FR"]), "es");
});

test("el selector conserva la ruta, otros parámetros y el ancla sin almacenamiento", () => {
  let destino;
  const ubicacion = { href: "https://vec.example/portal-empleado/?vista=bolsa#detalle", assign: (url) => { destino = url; } };
  let alCambiar;
  const selector = { value: "", addEventListener: (_evento, manejador) => { alCambiar = manejador; } };
  assert.equal(montarSelectorIdioma(selector, ubicacion), true);
  assert.equal(selector.value, "es");
  selector.value = "en";
  alCambiar();
  assert.equal(destino, "https://vec.example/portal-empleado/?vista=bolsa&lang=en#detalle");
  assert.equal(cambiarIdioma("fr", ubicacion), false);
});
