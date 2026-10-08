import test from "node:test";
import { prepararTextosPortal } from "./portal-i18n.js?v=20261007-pantallas-textos-final-v1";
import { prepararMensajesContratos } from "./portal-bolsas-contratos.js?v=20261007-pantallas-textos-final-v1";
await prepararTextosPortal("bolsa");
await prepararMensajesContratos();
import assert from "node:assert/strict";

import { consultarCandidatosBolsa, crearControladorBolsas } from "./portal-bolsas-api.js?v=20261007-pantallas-textos-final-v1";
import { crearPresentadorPanelInterno } from "./portal-panel-interno.js?v=20261007-pantallas-textos-final-v1";
import {
  leerCandidatosBolsaCompartible, rutaCandidatosBolsaCompartible,
  rutaResumenBolsasCompartible,
} from "./portal-bolsas-ruta-filtros.js";
import { SITUACIONES_PARTICIPACION_BOLSA } from "./portal-bolsas-contrato.js";

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
const BOLSAS_AUTORIZADAS = Object.freeze([{ bolsa_ref: BOLSA.bolsa_ref }, { bolsa_ref: "bolsa:sintetica:2" }]);

test("cada situación conserva bolsa e idioma en una URL restaurable", () => {
  for (const estado of SITUACIONES_PARTICIPACION_BOLSA) {
    const ruta = rutaCandidatosBolsaCompartible("?lang=en&tema=alto&cursor=obsoleto", BOLSA.bolsa_ref, estado);
    const url = new URL(ruta, "https://vec.example/portal-empleado/");
    assert.equal(url.hash, "#bolsa/bolsa-candidatos");
    assert.equal(url.searchParams.get("lang"), "en");
    assert.equal(url.searchParams.get("tema"), "alto");
    assert.deepEqual(leerCandidatosBolsaCompartible(url.search, BOLSAS_AUTORIZADAS),
      { bolsaRef: BOLSA.bolsa_ref, estado });
    assert.equal(url.searchParams.has("cursor"), false);
  }
});

test("volver al resumen retira bolsa y estado sin perder los demás parámetros", () => {
  const ruta = rutaCandidatosBolsaCompartible("?lang=es", BOLSA.bolsa_ref, "renuncia");
  const vuelta = rutaResumenBolsasCompartible(new URL(ruta, "https://vec.example/").search + "&cursor=obsoleto");
  const url = new URL(vuelta, "https://vec.example/portal-empleado/");
  assert.equal(url.hash, "#bolsa/resumen");
  assert.equal(url.searchParams.get("lang"), "es");
  assert.equal(url.searchParams.has("bolsa_ref"), false);
  assert.equal(url.searchParams.has("estado"), false);
  assert.equal(url.searchParams.has("cursor"), false);
  assert.equal(leerCandidatosBolsaCompartible(url.search, BOLSAS_AUTORIZADAS), null);
});

test("duplicados, estado ajeno y bolsa fuera de la lectura autorizada no abren lista", () => {
  for (const search of [
    "?bolsa_ref=bolsa:sintetica:1&bolsa_ref=bolsa:sintetica:2",
    "?bolsa_ref=bolsa:sintetica:1&estado=disponible&estado=renuncia",
    "?estado=disponible", "?bolsa_ref=bolsa:sintetica:1&estado=desconocido",
    "?bolsa_ref=bolsa:ajena:999", "?bolsa_ref=bolsa:sintetica:1%2Fextra",
    "?bolsa_ref=bolsa:sintetica:1&cursor=obsoleto",
  ]) assert.throws(() => leerCandidatosBolsaCompartible(search, BOLSAS_AUTORIZADAS), search);
  assert.throws(() => rutaCandidatosBolsaCompartible("", BOLSA.bolsa_ref, "desconocido"));
  assert.throws(() => leerCandidatosBolsaCompartible(`?bolsa_ref=${BOLSA.bolsa_ref}`, []));
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

test("el cuadro enlaza bolsas y situaciones exactas; el recuento de curso no abre un histórico sin filtro", () => {
  const html = presentador().renderizarSoloBolsas("resumen");
  assert.match(html, /<a class="tarjeta-kpi" href="#bolsa\/resumen" data-vista="resumen">/u);
  assert.match(html, /href="\?bolsa_ref=bolsa%3Asintetica%3A1#bolsa\/bolsa-candidatos" data-accion="ver-bolsa" data-bolsa-ref="bolsa:sintetica:1"/u);
  assert.match(html, /href="\?bolsa_ref=bolsa%3Asintetica%3A1&amp;estado=disponible#bolsa\/bolsa-candidatos" data-accion="ver-bolsa" data-bolsa-ref="bolsa:sintetica:1" data-estado="disponible"/u);
  assert.match(html, /<span class="estado-chip info">1<\/span>/u);
  assert.doesNotMatch(html, /data-pestana="historico"/u);
  assert.equal((html.match(/<article class="tarjeta-kpi/gu) ?? []).length, 3,
    "los totales globales de personas esperan una lista global autorizada");
});

test("el estado de una bolsa pagina 101 personas con el mismo predicado del contador", async () => {
  const bolsa = { ...BOLSA, total: 101,
    por_estado: { ...BOLSA.por_estado, disponible: 101 },
    politica_orden: { ...BOLSA.politica_orden, rotulo: "Orden vigente" } };
  const candidatos = Array.from({ length: 101 }, (_, indice) => ({ ...CANDIDATO,
    participacion_ref: `participacion:${indice + 1}`, orden: indice + 1, orden_acta: indice + 1,
    nombre_visible: `Persona sintética ${indice + 1}`, contactos_total: 0, ultimo_llamamiento: null }));
  const rutas = [];
  const fetchImpl = async (ruta) => {
    const url = new URL(ruta, "https://vec.example");
    rutas.push(url);
    const segunda = url.searchParams.has("cursor");
    return new Response(JSON.stringify({ data: {
      esquema: "vec.bolsa.rrhh.candidatos.v1", generado_en: "2026-10-08T08:00:00Z",
      bolsa, candidatos: segunda ? candidatos.slice(100) : candidatos.slice(0, 100), contactos: [],
      hay_mas: !segunda, cursor_siguiente: segunda ? null : candidatos[99].participacion_ref,
    } }), { status: 200, headers: { "Content-Type": "application/json" } });
  };
  const primera = await consultarCandidatosBolsa(BOLSA.bolsa_ref,
    { estado: "disponible", limite: 100 }, { fetchImpl });
  assert.equal(primera.ok, true, primera.mensaje);
  const segunda = await consultarCandidatosBolsa(BOLSA.bolsa_ref,
    { estado: "disponible", limite: 100, cursor: primera.datos.cursor_siguiente }, { fetchImpl });
  assert.equal(segunda.ok, true, segunda.mensaje);
  assert.equal(primera.datos.candidatos.length + segunda.datos.candidatos.length,
    bolsa.por_estado.disponible);
  assert.deepEqual(rutas.map((url) => url.searchParams.get("estado")), ["disponible", "disponible"]);
  assert.equal(rutas[0].searchParams.get("cursor"), null);
  assert.equal(rutas[1].searchParams.get("cursor"), candidatos[99].participacion_ref);
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
