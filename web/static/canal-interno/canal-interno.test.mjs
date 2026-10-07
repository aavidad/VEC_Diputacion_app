import test from "node:test";
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";

const leer = async (ruta) => JSON.parse(await readFile(new URL(ruta, import.meta.url), "utf8"));

test("el paquete de enlaces solo expone los destinos institucionales previstos", async () => {
  const datos = await leer("./destinos.json");
  assert.equal(datos.version, 1);
  assert.deepEqual(Object.keys(datos.destinos).sort(), ["canal", "estrategia", "institucional", "ley", "reglamento"]);
  assert.equal(datos.fuente_institucional, datos.destinos.institucional.url);
  for (const registro of Object.values(datos.destinos)) {
    const url = new URL(registro.url);
    for (const campo of ["organo", "fuente", "publicacion", "vigencia"]) assert.ok(registro[campo]);
    assert.equal(url.protocol, "https:");
    assert.equal(url.username, "");
    assert.equal(url.password, "");
    assert.ok(["www.dipgra.es", "bop.dipgra.es", "www.boe.es", "centinela.lefebvre.es"].includes(url.hostname));
    assert.equal(url.hash, "");
  }
  assert.equal(new URL(datos.destinos.canal.url).search, "");
});

test("los dos catálogos traducen todas las claves visibles de la página", async () => {
  const html = await readFile(new URL("./index.html", import.meta.url), "utf8");
  const claves = [...html.matchAll(/data-t="([a-z_]+)"/g)].map((coincidencia) => coincidencia[1]);
  for (const idioma of ["es", "en"]) {
    const catalogo = await leer(`../textos/${idioma}/canal-interno.json`);
    for (const clave of claves) assert.ok(catalogo[clave], `${idioma}: ${clave}`);
    for (const clave of ["documento", "ayuda_boton", "error", "reintentar"]) assert.ok(catalogo[clave], `${idioma}: ${clave}`);
    const error = await leer(`../textos/${idioma}/canal-interno-error.json`);
    for (const clave of ["documento", "saltar", "marca", "idioma", "pie", "error", "reintentar"]) assert.ok(error[clave], `${idioma}: ${clave}`);
  }
});

test("los enlaces exteriores salen sin referencia y no se carga el proveedor automáticamente", async () => {
  const html = await readFile(new URL("./index.html", import.meta.url), "utf8");
  const js = await readFile(new URL("./canal-interno.js", import.meta.url), "utf8");
  assert.equal((html.match(/target="_blank" rel="noopener noreferrer" referrerpolicy="no-referrer"/g) ?? []).length, 5);
  assert.ok(html.includes('<meta name="referrer" content="no-referrer">'));
  assert.ok(!html.includes("centinela.lefebvre.es"));
  assert.ok(!/localStorage|sessionStorage|document\.cookie|sendBeacon|fetch\(.*centinela/.test(js));
});
