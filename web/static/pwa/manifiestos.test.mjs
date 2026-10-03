import test from "node:test";
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { validarManifiestoPWA } from "./manifiestos-validacion.test-helper.mjs";

const raiz = new URL("../", import.meta.url);
const portales = [
  { archivo: "pwa-portal-empleado", ruta: "/portal-empleado/" },
  { archivo: "pwa-area-personal", ruta: "/area-personal/" },
  { archivo: "pwa-admin", ruta: "/administracion-perfiles/" },
];

async function leerJSON(ruta) {
  return JSON.parse(await readFile(new URL(ruta, raiz), "utf8"));
}

const indiceIdiomas = await leerJSON("textos/idiomas.json");
const idiomas = indiceIdiomas.idiomas.map(({ codigo }) => codigo);

test("el esquema PWA cerrado rechaza iconos y campos ajenos", async () => {
  const idioma = indiceIdiomas.por_defecto;
  const modulo = "pwa-portal-empleado";
  const base = await leerJSON(`textos/${idioma}/${modulo}.json`);
  validarManifiestoPWA(base, modulo, idioma);
  for (const cambiar of [
    datos => { datos.extra = "no admitido"; },
    datos => { datos.icons.push(datos.icons[0]); },
    datos => { datos.icons = {}; },
    datos => { datos.icons[0].src = "https://otro.example/icono.png"; },
    datos => { datos.icons[0].src = "/pwa/icons/vec-512.png?v=20261002-pwa-v1"; },
    datos => { datos.icons[0].sizes = "512x512"; },
    datos => { datos.icons[0].type = "image/svg+xml"; },
    datos => { datos.icons[0].purpose = "maskable"; },
    datos => { datos.icons[0].extra = "no admitido"; },
    datos => { datos.scope = "/otro/"; },
  ]) {
    const alterado = structuredClone(base);
    cambiar(alterado);
    assert.throws(() => validarManifiestoPWA(alterado, modulo, idioma));
  }
});

test("los manifiestos de cada portal tienen identidad, ámbito e idioma propios", async () => {
  for (const idioma of idiomas) {
    for (const portal of portales) {
      const manifiesto = await leerJSON(`textos/${idioma}/${portal.archivo}.json`);
      assert.equal(manifiesto.id, portal.ruta);
      assert.equal(manifiesto.scope, portal.ruta);
      assert.equal(manifiesto.start_url, `${portal.ruta}?lang=${idioma}`);
      assert.equal(manifiesto.lang, idioma);
      assert.equal(manifiesto.display, "standalone");
      assert.ok(manifiesto.name && manifiesto.short_name && manifiesto.description);
      assert.match(manifiesto.theme_color, /^#[0-9a-f]{6}$/u);
      assert.match(manifiesto.background_color, /^#[0-9a-f]{6}$/u);
      assert.deepEqual(
        manifiesto.icons.map(({ sizes, purpose }) => `${sizes}:${purpose}`),
        ["192x192:any", "512x512:any", "192x192:maskable", "512x512:maskable"],
      );
      for (const icono of manifiesto.icons) {
        assert.match(icono.src, /^\/pwa\/icons\/vec-[a-z0-9-]+\.png\?v=20261002-pwa-v1$/u);
        assert.equal(icono.type, "image/png");
      }
    }
  }
});

test("los iconos PNG servidos tienen dimensiones reales y firmas válidas", async () => {
  for (const tamano of [192, 512]) {
    for (const variante of ["", "maskable-"]) {
      const bytes = await readFile(new URL(`pwa/icons/vec-${variante}${tamano}.png`, raiz));
      assert.equal(bytes.subarray(0, 8).toString("hex"), "89504e470d0a1a0a");
      assert.equal(bytes.readUInt32BE(16), tamano);
      assert.equal(bytes.readUInt32BE(20), tamano);
    }
  }
  const ico = await readFile(new URL("pwa/icons/vec.ico", raiz));
  assert.equal(ico.readUInt16LE(0), 0);
  assert.equal(ico.readUInt16LE(2), 1);
  assert.equal(ico.readUInt16LE(4), 3);
});

test("los textos comunes de la aplicación están completos en todos los idiomas publicados", async () => {
  const catalogos = await Promise.all(idiomas.map((idioma) => leerJSON(`textos/${idioma}/pwa.json`)));
  const respaldo = catalogos[idiomas.indexOf(indiceIdiomas.por_defecto)];
  assert.ok(respaldo);
  const claves = Object.keys(respaldo.general);
  for (const catalogo of catalogos) {
    assert.deepEqual(Object.keys(catalogo.general), claves);
    for (const texto of Object.values(catalogo.general)) {
      assert.ok(typeof texto === "string" && texto.trim() === texto && texto.length > 0);
    }
  }
});
