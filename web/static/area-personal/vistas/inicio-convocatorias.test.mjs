import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { datosMinimosMiBolsa } from "../aplicacion.js";
import { iniciarI18nAreaPersonal } from "../i18n.js";
import { lectorCatalogos } from "../textos-prueba.test-helper.mjs";
import { renderizarConvocatorias, renderizarDetalleConvocatoria, renderizarInicio } from "./inicio-convocatorias.js";

const estado = { filtros: { termino: "", estado: "", categoria: "" }, convocatoriaSeleccionada: "CONV-AJENA" };

test("la vista y sus importadores comparten una URL renovada", async () => {
  const [html, arranque, aplicacion] = await Promise.all([
    readFile(new URL("../index.html", import.meta.url), "utf8"),
    readFile(new URL("../arranque.js", import.meta.url), "utf8"),
    readFile(new URL("../aplicacion.js", import.meta.url), "utf8"),
  ]);
  const versionShell = "20261009-mi-bolsa-pausa-null-v1";
  const versionVista = "20261005-b4-v1";
  assert.match(html, new RegExp(`/area-personal/arranque\\.js\\?v=${versionShell}`));
  assert.match(arranque, new RegExp(`\\./aplicacion\\.js\\?v=${versionShell}`));
  assert.match(aplicacion, new RegExp(`\\./vistas/inicio-convocatorias\\.js\\?v=${versionVista}`));
});

test("la consulta real de Mi bolsa no presenta ceros como plazos o solicitudes confirmados", async () => {
  const datos = datosMinimosMiBolsa({ consultada_en: "2026-09-30T10:00:00Z" });
  assert.equal(datos.meta.origen, "GET /api/vec/bolsa/mi-bolsa");
  for (const [idioma, visible] of [["es", "Ver Mi bolsa"], ["en", "View my job pool"]]) {
    await iniciarI18nAreaPersonal({ querySelectorAll: () => [], documentElement: {} }, {
      leer: lectorCatalogos(), ubicacion: { href: `https://vec.example/area-personal/?lang=${idioma}` },
    });
    for (const html of [renderizarInicio(datos), renderizarConvocatorias(datos, estado)]) {
      assert.match(html, /data-ruta="llamamientos"/u);
      assert.match(html, /href="\?vista=llamamientos"/u);
      assert.ok(html.includes(visible));
      assert.doesNotMatch(html, /data-accion="iniciar-solicitud"|data-accion="abrir-expediente"/u);
      assert.doesNotMatch(html, /class="resumen-cifras"|id="filtros-convocatorias"/u);
    }
    const inicio = renderizarInicio(datos);
    const convocatorias = renderizarConvocatorias(datos, estado);
    if (idioma === "es") {
      assert.match(inicio, /Consulte los datos de su participación que ya están disponibles/u);
      assert.match(convocatorias, /El listado de convocatorias aún no está disponible/u);
      assert.doesNotMatch(inicio, /Plazos, acciones y estado de sus procesos/u);
      assert.doesNotMatch(convocatorias, /Consulte bases, requisitos, plazos y estado/u);
    } else {
      assert.match(inicio, /View the details of your participation that are already available/u);
      assert.match(convocatorias, /The list of calls for applications is not yet available/u);
      assert.doesNotMatch(inicio, /Deadlines, actions and the status of your processes/u);
      assert.doesNotMatch(convocatorias, /Check the terms, requirements, deadlines and status/u);
    }
  }
});

test("un identificador ausente no muestra el detalle de otra convocatoria", async () => {
  await iniciarI18nAreaPersonal({ querySelectorAll: () => [], documentElement: {} }, {
    leer: lectorCatalogos(), ubicacion: { href: "https://vec.example/area-personal/?lang=es" },
  });
  const datos = { convocatorias: [{
    id: "CONV-1", titulo: "Convocatoria de otra persona", estado: "Plazo abierto", requisitos: [], documentos: [],
  }] };
  const html = renderizarDetalleConvocatoria(datos, estado);
  assert.match(html, /Convocatoria no disponible/u);
  assert.match(html, /data-ruta="convocatorias"/u);
  assert.doesNotMatch(html, /Convocatoria de otra persona|data-accion="iniciar-solicitud"/u);

  const consultaBolsa = renderizarDetalleConvocatoria(datosMinimosMiBolsa({ consultada_en: "2026-09-30T10:00:00Z" }), estado);
  assert.match(consultaBolsa, /data-ruta="llamamientos"/u);
  assert.match(consultaBolsa, /Consulte su participación en Mi bolsa/u);
  assert.doesNotMatch(consultaBolsa, /data-ruta="convocatorias"|Vuelva al listado/u);
});

test("un listado vacío y un filtro sin coincidencias tienen mensajes distintos", () => {
  const vacio = renderizarConvocatorias({ convocatorias: [], meta: { origen: "otra consulta" } }, estado);
  assert.match(vacio, /Sin convocatorias en esta consulta/u);
  assert.doesNotMatch(vacio, /Modifique los filtros/u);

  const filtrado = renderizarConvocatorias({ convocatorias: [{
    id: "CONV-1", titulo: "Bolsa de prueba", referencia: "BOP-1", categoria: "Operario/a",
    estado: "Plazo abierto", descripcion: "Bases públicas", plazo: "Plazo vigente",
  }], meta: { origen: "otra consulta" } }, { filtros: { termino: "sin coincidencias", estado: "", categoria: "" } });
  assert.match(filtrado, /Modifique los filtros/u);
  assert.doesNotMatch(filtrado, /Sin convocatorias en esta consulta/u);
});
