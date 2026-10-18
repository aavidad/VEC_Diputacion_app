import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { datosMinimosMiBolsa } from "../aplicacion.js";
import { iniciarI18nAreaPersonal } from "../i18n.js";
import { lectorCatalogos } from "../textos-prueba.test-helper.mjs";
import * as vista from "./inicio-convocatorias.js";

test("la entrada se renueva y la vista de inicio conserva su URL vigente", async () => {
  const [html, arranque, aplicacion] = await Promise.all([
    readFile(new URL("../index.html", import.meta.url), "utf8"),
    readFile(new URL("../arranque.js", import.meta.url), "utf8"),
    readFile(new URL("../aplicacion.js", import.meta.url), "utf8"),
  ]);
  const entrada = html.match(/src="\/area-personal\/arranque\.js\?v=([a-zA-Z0-9._-]+)"/u);
  const modulo = arranque.match(/from "\.\/aplicacion\.js\?v=([a-zA-Z0-9._-]+)"/u);
  assert.ok(entrada, "falta la entrada versionada del área personal");
  assert.ok(modulo, "falta el módulo de aplicación versionado");
  assert.equal(modulo[1], entrada[1], "entrada y aplicación deben compartir la versión vigente");
  assert.match(aplicacion, /from "\.\/vistas\/inicio-convocatorias\.js\?v=[a-zA-Z0-9._-]+"/u);
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
