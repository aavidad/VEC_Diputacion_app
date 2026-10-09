import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { datosMinimosMiBolsa } from "../aplicacion.js";
import { iniciarI18nAreaPersonal } from "../i18n.js";
import { lectorCatalogos } from "../textos-prueba.test-helper.mjs";
import * as vista from "./inicio-convocatorias.js";

test("la vista y sus importadores comparten una URL renovada", async () => {
  const [html, arranque, aplicacion] = await Promise.all([
    readFile(new URL("../index.html", import.meta.url), "utf8"),
    readFile(new URL("../arranque.js", import.meta.url), "utf8"),
    readFile(new URL("../aplicacion.js", import.meta.url), "utf8"),
  ]);
  const version = "20261009-nombre-propio-v1";
  assert.match(html, new RegExp(`/area-personal/arranque\\.js\\?v=${version}`));
  assert.match(arranque, new RegExp(`\\./aplicacion\\.js\\?v=${version}`));
  assert.match(aplicacion, new RegExp(`\\./vistas/inicio-convocatorias\\.js\\?v=${version}`));
});

test("el inicio remite a Mi bolsa y ya no ofrece convocatorias ni expedientes", async () => {
  assert.deepEqual(Object.keys(vista), ["renderizarInicio"]);
  const datos = datosMinimosMiBolsa({ consultada_en: "2026-09-30T10:00:00Z" });
  for (const [idioma, visible, descripcion] of [
    ["es", "Ver Mi bolsa", /Consulte los datos de su participación que ya están disponibles/u],
    ["en", "View my job pool", /View the details of your participation that are already available/u],
  ]) {
    await iniciarI18nAreaPersonal({ querySelectorAll: () => [], documentElement: {} }, {
      leer: lectorCatalogos(), ubicacion: { href: `https://vec.example/area-personal/?lang=${idioma}` },
    });
    const html = vista.renderizarInicio(datos);
    assert.match(html, /data-ruta="llamamientos"/u);
    assert.match(html, /href="\?vista=llamamientos"/u);
    assert.ok(html.includes(visible));
    assert.match(html, descripcion);
    assert.doesNotMatch(html, /data-ruta="(?:convocatorias|seguimiento|meritos)"|data-accion="abrir-expediente"|class="resumen-cifras"/u);
  }
});
