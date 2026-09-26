import assert from "node:assert/strict";
import test from "node:test";

import { numeroExpedienteVisible, renderizarCuadro } from "./componentes-expedientes.js";
import { crearTraductorExpedientesContratacion } from "./i18n-expedientes.js";
import { renderizarCabeceraModulo } from "./vista-expedientes-render.js";

const t = crearTraductorExpedientesContratacion();
const expediente = (sufijo, estado = "en_curso") => ({
  expediente_ref: `expediente:ct:${sufijo}`, numero_visible: `2026/CT-00000${sufijo}`,
  centro: "Centro", categoria: "Auxiliar", modalidad: "Sustitución", estado_clave: estado,
  estado: "En curso", fase_clave: "solicitud", fase_actual: "Solicitud", plazo: "—", version: 1,
});
const indicadores = [
  { clave: "total", etiqueta: "Expedientes", valor: "2", tono: "informacion" },
  { clave: "pendientes", etiqueta: "Pendientes", valor: "0", tono: "aviso" },
  { clave: "en_curso", etiqueta: "En curso", valor: "2", tono: "informacion" },
  { clave: "incidencias", etiqueta: "Incidencias", valor: "0", tono: "peligro" },
];
const estadoCuadro = (paginacion, extra = {}) => ({
  vista: "cuadro", carga: "listo", filtros: { texto: "", estado: "", fase: "" },
  cuadro: { demostracion: false, indicadores, expedientes: [expediente(1), expediente(2)], paginacion },
  ...extra,
});

test("los indicadores usan iconos comunes y rótulos cortos sin «en esta página»", () => {
  const html = renderizarCuadro(estadoCuadro({ pagina: 1, cursor_siguiente: "" }), t);
  // Los cuatro llevan a la lista con su filtro (recuadros pulsables, como en Inicio).
  assert.equal((html.match(/<button type="button" class="tarjeta-kpi/gu) || []).length, 4);
  assert.equal((html.match(/<span class="icono-kpi"><svg aria-hidden="true"/gu) || []).length, 4);
  assert.match(html, /tarjeta-kpi kpi--peligro" data-ct-exp-indicador="incidencias"/u);
  assert.match(html, /<span class="etiqueta-kpi">Expedientes<\/span>/u);
  assert.doesNotMatch(html, /en esta página|▣|ct-exp-nota-tabla|estado-chip info/u);
});

test("una sola página deja la paginación al marco común; con más páginas va junto a la tabla", () => {
  const unica = renderizarCuadro(estadoCuadro({ pagina: 1, cursor_siguiente: "" }), t);
  assert.doesNotMatch(unica, /ct-exp-paginacion|data-ct-exp-pagina/u);
  const varias = renderizarCuadro(estadoCuadro({ pagina: 1, cursor_siguiente: "cursor-b" }), t);
  assert.match(varias, /<\/div>\s*<nav class="ct-exp-paginacion"[\s\S]*<\/nav>\s*<\/section>/u);
  const reinicio = renderizarCuadro(estadoCuadro(
    { pagina: 1, cursor_siguiente: "" }, { paginacion_requiere_reinicio: true },
  ), t);
  assert.match(reinicio, /data-ct-exp-pagina="primera"/u);
});

test("la cabecera no lleva sobrelínea, descripción ni aviso de presentación y ofrece Centros y Peticiones", () => {
  const cuadro = renderizarCabeceraModulo({ vista: "cuadro", cuadro: { demostracion: true } }, t);
  assert.match(cuadro, /<h2>Expedientes de contratación<\/h2>/u);
  assert.match(cuadro, /class="acciones-vista ct-exp-acciones-cabecera"[\s\S]*>Centros<\/a>[\s\S]*>Peticiones<\/a>/u);
  assert.match(cuadro, /href="\/portal-empleado\/calendarios\/"[^>]*>Calendarios<\/a>/u);
  assert.doesNotMatch(cuadro, /sobrelinea|Flujo guiado|Presentación RRHH|sintéticos|ct-exp-aviso-presentacion/u);
  const detalle = renderizarCabeceraModulo({ vista: "expediente" }, t);
  assert.doesNotMatch(detalle, /ct-exp-acciones-cabecera/u);
});

test("el número técnico anterior a la numeración figura sin numerar; el legible no cambia", () => {
  assert.equal(numeroExpedienteVisible("2026/CT-8c17ba0b2be0fa7d84131e1dc93db150"), "Sin numerar");
  assert.equal(numeroExpedienteVisible("2026/CT-000013"), "2026/CT-000013");
  assert.equal(numeroExpedienteVisible(null), "");
});
