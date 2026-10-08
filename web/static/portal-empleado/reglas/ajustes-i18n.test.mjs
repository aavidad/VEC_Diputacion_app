import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { cargarTextosAjustes, existeClaveAjustes, idiomaAjustes, traducirAjustes } from "./ajustes-i18n.js";

test("los textos del panel se cargan a demanda y el idioma de respaldo es recuperable", async () => {
  let intentos = 0;
  const leer = async (url) => {
    intentos++;
    if (url.pathname.includes("/en/")) throw new Error("catálogo no disponible");
    return JSON.parse(await readFile(url, "utf8"));
  };
  const respaldado = await cargarTextosAjustes({ idioma: "en", porDefecto: "es", leer, avisar: () => {} });
  assert.equal(respaldado.idioma, "es");
  assert.equal(idiomaAjustes(), "es");
  assert.ok(existeClaveAjustes("ajustesFechaFutura"));
  assert.equal(traducirAjustes("ajustesDesdeAhora"), "Desde ahora");
  assert.equal(intentos, 2);
  const recuperado = await cargarTextosAjustes({ idioma: "en", porDefecto: "es" });
  assert.equal(recuperado.idioma, "en");
  assert.equal(traducirAjustes("ajustesDesdeAhora"), "From now");
});
