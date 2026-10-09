import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { crearUtilidadesVista } from "./portal-vistas-utilidades.js?v=20261001-ct-a-i18n-v1";
import { tabla as tablaPersonal } from "../area-personal/vistas/comunes.js";

const escaparHTML = (valor) => String(valor).replaceAll("&", "&amp;").replaceAll('"', "&quot;").replaceAll("<", "&lt;");
const { tabla } = crearUtilidadesVista({ escaparHTML, numero: String, claseEstado: () => "neutro", encabezadoVista: () => "" });

test("las tablas de consulta permiten scroll con teclado y nombre del contenido, incluso vacío", () => {
  for (const filas of [[], [["1", "Pendiente"]]]) {
    const html = tabla({ titulo: 'Peticiones "nuevas" <RRHH>', cabeceras: ["Número", "Estado"], filas });
    assert.match(html, /tabindex="0" role="region" aria-label="Peticiones &quot;nuevas&quot; &lt;RRHH>"/);
    assert.equal((html.match(/scope="col"/g) || []).length, 2);
    assert.match(html, /<caption>Peticiones &quot;nuevas&quot; &lt;RRHH><\/caption>/);
  }
});

test("las tablas de candidato conservan semántica y cabeceras al convertirse en fichas", () => {
  const html = tablaPersonal({ descripcion: 'Mi bolsa "propia"', columnas: ["Puesto", "Estado"], filas: [["Auxiliar", "Disponible"]] });
  assert.match(html, /tabindex="0" role="region" aria-label="Mi bolsa &quot;propia&quot;"/);
  assert.match(html, /<table class="tabla-administrativa" role="table">/);
  assert.equal((html.match(/scope="col"/g) || []).length, 2);
  assert.match(html, /data-etiqueta="Estado"/);
});

test("el área personal conserva Perfil y Mi bolsa sin el formulario de méritos retirado", async () => {
  const { renderizarPerfil } = await import("../area-personal/vistas/perfil-meritos-solicitud.js");
  const { renderizarLlamamientos } = await import("../area-personal/vistas/seguimiento-tramites.js");
  const { datosMinimosMiBolsa } = await import("../area-personal/aplicacion.js");
  const { iniciarI18nAreaPersonal } = await import("../area-personal/i18n.js");
  const { lectorCatalogos } = await import("../area-personal/textos-prueba.test-helper.mjs");
  const [html, rutas, fuentePerfil] = await Promise.all([
    readFile(new URL("../area-personal/index.html", import.meta.url), "utf8"),
    readFile(new URL("../area-personal/vistas.json", import.meta.url), "utf8"),
    readFile(new URL("../area-personal/vistas/perfil-meritos-solicitud.js", import.meta.url), "utf8"),
  ]);
  assert.equal(JSON.parse(rutas).vistas.meritos, undefined);
  assert.doesNotMatch(html, /data-ruta="meritos"/u);
  assert.doesNotMatch(fuentePerfil, /renderizarMeritos|formulario-merito|data-operacion="incorporar_merito"/u);
  const datos = datosMinimosMiBolsa({ consultada_en: "2026-10-08T10:00:00Z" });
  for (const [idioma, tituloBolsa] of [["es", "Mi bolsa"], ["en", "My job pool"]]) {
    await iniciarI18nAreaPersonal({ querySelectorAll: () => [], documentElement: {} }, {
      leer: lectorCatalogos(), ubicacion: { href: `https://vec.example/area-personal/?lang=${idioma}` },
    });
    const perfil = renderizarPerfil(datos);
    const bolsa = renderizarLlamamientos(datos, { participaciones: [] });
    assert.match(perfil, /id="ficha-aspirante"[\s\S]*id="contacto-propio"/u);
    assert.ok(bolsa.includes(tituloBolsa));
    assert.doesNotMatch(`${perfil}${bolsa}`, /formulario-merito|data-operacion="incorporar_merito"/u);
  }
});
