import test from "node:test";
import assert from "node:assert/strict";
import { cambiarIdioma, montarSelectorIdioma, prepararIdiomas, resolverIdiomaNavegacion, seleccionarIdioma } from "./idioma.js";

await prepararIdiomas();

test("la elección explícita de interfaz prevalece sobre el navegador (índice del repositorio)", () => {
  assert.equal(seleccionarIdioma("en", ["es-ES"]), "en");
  assert.equal(seleccionarIdioma("fr", ["en-GB", "es-ES"]), "en");
  assert.equal(seleccionarIdioma("", ["fr-FR"]), "es");
});

test("idioma de navegación válido domina servidor, que domina navegador; se ignoran valores ajenos", () => {
  const navegador = { languages: ["en-GB", "es-ES"] };
  const ubicacion = (lang) => ({ href: `https://vec.example/portal-empleado/?lang=${lang}` });
  assert.equal(resolverIdiomaNavegacion({ ubicacion: ubicacion("es"), idiomaPreferido: "en", navegador }), "es");
  assert.equal(resolverIdiomaNavegacion({ ubicacion: ubicacion("fr"), idiomaPreferido: "es", navegador }), "es");
  assert.equal(resolverIdiomaNavegacion({ ubicacion: ubicacion("fr"), idiomaPreferido: "navegador", navegador }), "en");
  assert.equal(resolverIdiomaNavegacion({ ubicacion: ubicacion("fr"), idiomaPreferido: "invalido", navegador: { languages: ["fr-FR"] } }), "es");
  assert.equal(seleccionarIdioma("en", ["es-ES"], "es"), "en");
  const ubicacionSoloLectura = Object.freeze(ubicacion("fr"));
  const navegadorSoloLectura = Object.freeze({ languages: Object.freeze(["en-GB"]) });
  assert.equal(resolverIdiomaNavegacion({ ubicacion: ubicacionSoloLectura, idiomaPreferido: "es", navegador: navegadorSoloLectura }), "es");
  assert.equal(ubicacionSoloLectura.href, "https://vec.example/portal-empleado/?lang=fr");
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
