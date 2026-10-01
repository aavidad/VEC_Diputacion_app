import assert from "node:assert/strict";
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

test("el formulario de mérito indica campos obligatorios y opcionales con catálogos reales ES/EN", async () => {
  const { renderizarMeritos } = await import("../area-personal/vistas/perfil-meritos-solicitud.js");
  const { iniciarI18nAreaPersonal } = await import("../area-personal/i18n.js");
  const { lectorCatalogos } = await import("../area-personal/textos-prueba.test-helper.mjs");
  for (const [idioma, obligatorio, opcional] of [["es", "(obligatorio)", "(opcional)"], ["en", "(required)", "(optional)"]]) {
    await iniciarI18nAreaPersonal({ querySelectorAll: () => [], documentElement: {} }, {
      leer: lectorCatalogos(), ubicacion: { href: `https://vec.example/area-personal/?lang=${idioma}` },
    });
    const html = renderizarMeritos({ meritos: [], documentos: [] });
    for (const campo of ["tipo", "titulo"]) {
      const etiqueta = html.match(new RegExp(`<label for="merito-${campo}">(.*?)</label>`))[1];
      assert.ok(etiqueta.includes(obligatorio));
      assert.ok(!etiqueta.includes(opcional));
    }
    for (const campo of ["jornada", "documento"]) {
      const etiqueta = html.match(new RegExp(`<label for="merito-${campo}">(.*?)</label>`))[1];
      assert.ok(etiqueta.includes(opcional));
    }
    assert.match(html, /aria-describedby="merito-documento-ayuda"/);
    assert.match(html, /<small id="merito-documento-ayuda">[^<]+<\/small>/);
    assert.doesNotMatch(html, /areaPersonal\.vista\.comun\.campo/);
  }
});
