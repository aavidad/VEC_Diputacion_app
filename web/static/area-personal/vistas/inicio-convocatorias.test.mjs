import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { iniciarI18nAreaPersonal } from "../i18n.js";
import { lectorCatalogos } from "../textos-prueba.test-helper.mjs";
import { renderizarInicio } from "./inicio-convocatorias.js";

test("Inicio enlaza Mi bolsa sin presentar convocatorias ni solicitudes sin servidor", async () => {
  const [html, arranque, aplicacion] = await Promise.all([
    readFile(new URL("../index.html", import.meta.url), "utf8"),
    readFile(new URL("../arranque.js", import.meta.url), "utf8"),
    readFile(new URL("../aplicacion.js", import.meta.url), "utf8"),
  ]);
  assert.match(html, /arranque\.js\?v=20261008-b4-v6/u);
  assert.match(arranque, /aplicacion\.js\?v=20261008-b4-v6/u);
  assert.match(aplicacion, /inicio-convocatorias\.js\?v=20261008-b4-v6/u);
  for (const [idioma, boton] of [["es", "Ver Mi bolsa"], ["en", "View my job pool"]]) {
    await iniciarI18nAreaPersonal({ querySelectorAll: () => [], documentElement: {} }, {
      leer: lectorCatalogos(), ubicacion: { href: `https://vec.example/area-personal/?lang=${idioma}` },
    });
    const vista = renderizarInicio();
    assert.match(vista, /data-ruta="llamamientos"/u);
    assert.ok(vista.includes(boton));
    assert.doesNotMatch(vista, /iniciar-solicitud|abrir-expediente|resumen-cifras|filtros-convocatorias|<p>/u);
  }
});
