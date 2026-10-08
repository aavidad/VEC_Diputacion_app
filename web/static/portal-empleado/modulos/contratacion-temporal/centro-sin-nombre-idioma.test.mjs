import assert from "node:assert/strict";
import test from "node:test";
import { renderizarResultadosCuadro } from "./componentes-expedientes.js";
import { crearTraductorExpedientesContratacion, cargarMensajesExpedientesContratacionEnIdioma } from "./i18n-expedientes.js";

const estado = { cuadro: { generado_en: "2026-10-01T08:00:00Z", expedientes: [{ expediente_ref: "expediente:ct:1",
  numero_visible: "2026/CT-0001", centro: "centro:rpt:600", categoria: "Auxiliar", fase_clave: "analisis",
  fase_actual: "Análisis", estado_clave: "en_curso", estado: "En trámite" }] } };

test("un centro sin nombre se dice en el idioma activo", async () => {
  assert.match(renderizarResultadosCuadro(estado, crearTraductorExpedientesContratacion()), />Centro con código 600<small>/u);
  const en = crearTraductorExpedientesContratacion(await cargarMensajesExpedientesContratacionEnIdioma("en"));
  assert.match(renderizarResultadosCuadro(estado, en), />Centre with code 600<small>/u);
});
