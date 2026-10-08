import assert from "node:assert/strict";
import test from "node:test";
import { iniciarI18nAreaPersonal } from "./i18n.js";
import { lectorCatalogos } from "./textos-prueba.test-helper.mjs";
import { renderizarAyuda } from "./vistas/comunicaciones-ayuda.js";

test("la ayuda orienta a Mi bolsa sin ofrecer el asistente ni trámites antiguos", async () => {
  for (const idioma of ["es", "en"]) {
    await iniciarI18nAreaPersonal({ querySelectorAll: () => [], documentElement: {} }, {
      leer: lectorCatalogos(), ubicacion: { href: `https://vec.example/area-personal/?lang=${idioma}` },
    });
    const vista = renderizarAyuda();
    assert.match(vista, /data-ruta="llamamientos"/u);
    assert.doesNotMatch(vista, /data-ruta="(?:convocatorias|meritos)"|asistente|Assistant|inscribirme|Submit merits/u);
    assert.match(vista, /<h3 id="titulo-guia-ayuda">/u);
  }
});
