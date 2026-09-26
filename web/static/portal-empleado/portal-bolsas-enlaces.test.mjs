import test from "node:test";
import assert from "node:assert/strict";

import { crearControladorBolsas } from "./portal-bolsas-api.js";
import { crearPresentadorPanelInterno } from "./portal-panel-interno.js";

const BOLSA = Object.freeze({
  bolsa_ref: "bolsa:sintetica:1", categoria_clave: "administrativo", categoria: "ADMINISTRATIVO",
  tipo_lista: "rotatoria", vigente_desde: "2026-09-18T00:43:00Z", vigente_hasta: null, total: 2,
  llamamientos_en_curso: 1,
  por_estado: { disponible: 2, no_disponible: 0, trabajando: 0, pendiente_incorporacion: 0, renuncia: 0, excluido: 0, disponible_desde: 0 },
  politica_orden: { politica_ref: "politica:orden:1", version: 1, criterio: "puntuacion_desc_acta", tipo_lista: "rotatoria", reposicion: "misma_posicion", provisional: false, rotulo: "", actor: "sistema", vigente_desde: "2026-09-18T00:43:00Z" },
});
const CANDIDATO = Object.freeze({
  participacion_ref: "part_sintetica_1", orden: 1, orden_acta: 1, razon_orden: "orden_acta",
  nombre_visible: "Antonio Reyes Álvarez", documento_enmascarado: "***0071**", estado_clave: "disponible",
  estado_desde: "2026-09-18T00:43:00Z", disponible_desde: null, contactos_total: 1,
  ultimo_llamamiento: { llamamiento_ref: "llamamiento:1", comunicado_en: "2026-09-20T10:00:00Z", canal: "correo", resultado: "sin_respuesta" },
});

function presentador(filtros = { estado: "", texto: "" }, modalFicha = null) {
  return crearPresentadorPanelInterno({
    claseEstado: (clave) => `chip-${clave}`,
    encabezadoVista: (_s, titulo, _d, acciones = "") => `<header><h2>${titulo}</h2>${acciones}</header>`,
    escaparHTML: (valor) => String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll('"', "&quot;"),
    numero: (valor) => String(valor ?? 0),
    obtenerDatosPanel: () => ({ esquema: "vec.bolsa.panel.interno.v1" }),
    tituloVista: (vista) => vista,
    obtenerDatosBolsas: () => ({ carga: "listo", datos: { bolsas: [BOLSA] } }),
    obtenerDatosCandidatosBolsa: () => ({ carga: "listo", datos: { bolsa: BOLSA, candidatos: [CANDIDATO], contactos: [], hay_mas: false } }),
    obtenerEstadoCandidatos: () => filtros,
    obtenerModalFicha: () => modalFicha,
    obtenerDatosEstadisticas: () => ({ carga: "listo", datos: {
      generado_en: "2026-09-26T09:52:00Z", bolsas: { total: 1, vigentes: 1, sustituidas: 0 },
      personas: { total: 2, por_estado: { disponible: 2 } }, llamamientos: { total: 1, por_canal: {}, por_resultado: {} },
      por_bolsa: [{ bolsa_ref: BOLSA.bolsa_ref, categoria: BOLSA.categoria, tipo_lista: "rotatoria", vigente: true, total: 2, por_estado: BOLSA.por_estado }],
    } }),
  });
}

test("el recuadro de bolsas de las estadísticas lleva al cuadro y los demás siguen como resumen", () => {
  const html = presentador().renderizarEstadisticasBolsa();
  assert.match(html, /<button type="button" class="tarjeta-kpi" data-vista="resumen" aria-label="Bolsas: 1\. Abrir el cuadro de mando con la relación de bolsas">/u);
  assert.equal((html.match(/<button type="button" class="tarjeta-kpi"/gu) || []).length, 1);
  assert.equal((html.match(/<article class="tarjeta-kpi">/gu) || []).length, 4);
});

test("los llamamientos en curso del cuadro abren el histórico de su bolsa", () => {
  const html = presentador().renderizarSoloBolsas("resumen");
  assert.match(html, /<button type="button" class="estado-chip info" data-accion="ver-bolsa" data-bolsa-ref="bolsa:sintetica:1" data-pestana="historico" aria-label="1 llamamientos en curso de ADMINISTRATIVO\. Abrir su histórico de llamamientos">1<\/button>/u);
});

test("en el histórico el nombre abre la ficha y las tablas anchas se desplazan en su propia región", () => {
  const historico = presentador({ estado: "", texto: "", pestana: "historico" }).renderizarVista("bolsa-candidatos");
  assert.match(historico, /data-bolsa-accion="abrir-ficha-historico" data-participacion-ref="part_sintetica_1" aria-label="Abrir la ficha de Antonio Reyes Álvarez">Antonio Reyes Álvarez<\/button>/u);
  assert.match(historico, /<div class="tabla-contenedor" tabindex="0" role="region" aria-label="Aspirantes ordenados por mérito y situación en bolsa">/u);
  const ficha = presentador(undefined, {
    abierto: true, candidato: CANDIDATO, bolsa: BOLSA,
    operacionesB8: { carga: "listo", items: [{ desde: "2026-09-20T10:00:00Z", operacion: "pausar", situacion: "no_disponible", motivo: "Cuidado de familiar", justificante: { tipo: "correo", referencia: "registro:1" }, actor: "actor:rrhh:1", validador: "actor:rrhh:2", validada_en: "2026-09-20T10:05:00Z" }] },
  }).renderizarVista("bolsa-candidatos");
  assert.match(ficha, /<div class="tabla-contenedor" tabindex="0" role="region" aria-label="Historial de operaciones"><table/u);
});

test("abrir la ficha desde el histórico vuelve a candidatos y abre la ficha; el cuadro abre el histórico", () => {
  const escuchas = {};
  const estado = {
    filtrosBolsa: { estado: "", texto: "", pestana: "historico" },
    bolsaSeleccionada: BOLSA.bolsa_ref,
    datosCandidatos: { carga: "listo", datos: { bolsa: BOLSA, candidatos: [CANDIDATO], contactos: [] } },
  };
  const documento = { addEventListener(tipo, fn) { escuchas[tipo] = fn; }, querySelector: () => null, querySelectorAll: () => [] };
  let renderizados = 0;
  const navegaciones = [];
  const controlador = crearControladorBolsas({ estado, renderizar: () => { renderizados += 1; }, navegar: (vista) => navegaciones.push(vista), documento });
  controlador.instalar();
  const pulsar = (dataset, selectorPropio) => escuchas.click({
    preventDefault() {},
    target: { closest: (selector) => (selector === selectorPropio ? { dataset } : null) },
  });
  pulsar({ bolsaAccion: "abrir-ficha-historico", participacionRef: CANDIDATO.participacion_ref }, "[data-bolsa-accion]");
  assert.equal(estado.filtrosBolsa.pestana, "candidatos");
  assert.equal(estado.modalFicha?.candidato?.participacion_ref, CANDIDATO.participacion_ref);
  assert.ok(renderizados >= 1);
  pulsar({ accion: "ver-bolsa", bolsaRef: BOLSA.bolsa_ref, pestana: "historico" }, '[data-accion="ver-bolsa"], [data-bolsa-abrir="true"]');
  assert.equal(estado.filtrosBolsa.pestana, "historico");
  assert.deepEqual(navegaciones, ["bolsa-candidatos"]);
});
