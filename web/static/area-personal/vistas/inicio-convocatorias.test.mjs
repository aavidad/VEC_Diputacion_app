import assert from "node:assert/strict";
import test from "node:test";

import { datosMinimosMiBolsa } from "../aplicacion.js";
import { iniciarI18nAreaPersonal } from "../i18n.js";
import { lectorCatalogos } from "../textos-prueba.test-helper.mjs";
import { renderizarConvocatorias, renderizarDetalleConvocatoria, renderizarInicio } from "./inicio-convocatorias.js";

const estado = { filtros: { termino: "", estado: "", categoria: "" }, convocatoriaSeleccionada: "CONV-AJENA" };

test("la consulta real de Mi bolsa no presenta ceros como plazos o solicitudes confirmados", async () => {
  const datos = datosMinimosMiBolsa({ consultada_en: "2026-09-30T10:00:00Z" });
  for (const [idioma, visible] of [["es", "Ver Mi bolsa"], ["en", "View my employment pool"]]) {
    await iniciarI18nAreaPersonal({ querySelectorAll: () => [], documentElement: {} }, {
      leer: lectorCatalogos(), ubicacion: { href: `https://vec.example/area-personal/?lang=${idioma}` },
    });
    for (const html of [renderizarInicio(datos), renderizarConvocatorias(datos, estado)]) {
      assert.match(html, /data-ruta="llamamientos"/u);
      assert.ok(html.includes(visible));
      assert.doesNotMatch(html, /data-accion="iniciar-solicitud"|data-accion="abrir-expediente"/u);
      assert.doesNotMatch(html, /class="resumen-cifras"|id="filtros-convocatorias"/u);
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
