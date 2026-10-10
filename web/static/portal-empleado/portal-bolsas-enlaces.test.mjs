import test from "node:test";
import { prepararTextosPortal } from "./portal-i18n.js?v=20261007-pantallas-textos-final-v1";
import { prepararMensajesContratos } from "./portal-bolsas-contratos.js?v=20261007-pantallas-textos-final-v1";
await prepararTextosPortal("bolsa");
await prepararMensajesContratos();
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { runInNewContext } from "node:vm";
import * as rutasGlobales from "./portal-bolsas-ruta-filtros.js?v=20261008-bolsa-global-v2";

import { consultarBolsas, consultarCandidatosBolsa, consultarEstadisticasBolsa, consultarGlobalBolsa, crearControladorBolsas } from "./portal-bolsas-api.js?v=20261010-ct-bolsa-cohorte-v10";
import { crearPresentadorPanelInterno } from "./portal-panel-interno.js?v=20261010-ct-bolsa-cohorte-v10";
import { renderizarGlobalBolsa, prepararTextosGlobalBolsa } from "./portal-bolsas-global.js?v=20261008-bolsa-global-v2";
import {
  leerCandidatosBolsaCompartible, leerGlobalBolsaCompartible, rutaCandidatosBolsaCompartible, rutaGlobalBolsaCompartible,
  rutaResumenBolsasCompartible,
} from "./portal-bolsas-ruta-filtros.js?v=20261008-bolsa-global-v2";
import { SITUACIONES_PARTICIPACION_BOLSA } from "./portal-bolsas-contrato.js";

await prepararTextosGlobalBolsa();
const BOLSA = Object.freeze({
  bolsa_ref: "bolsa:sintetica:1", categoria_clave: "administrativo", categoria: "ADMINISTRATIVO",
  tipo_lista: "rotatoria", vigente_desde: "2026-09-18T00:43:00Z", vigente_hasta: null, total: 2,
  llamamientos_en_curso: 1,
  por_estado: { disponible: 2, no_disponible: 0, trabajando: 0, pendiente_incorporacion: 0, renuncia: 0, excluido: 0, disponible_desde: 0 },
  politica_orden: { politica_ref: "politica:orden:1", version: 1, criterio: "puntuacion_desc_acta", tipo_lista: "rotatoria", reposicion: "misma_posicion", provisional: false, rotulo: "", actor: "sistema", vigente_desde: "2026-09-18T00:43:00Z" },
});

const CORTE_GLOBAL = "a".repeat(64);
test("el enlace global conserva idioma y corte, y rechaza filtros mezclados", () => {
  const ruta = rutaGlobalBolsaCompartible("?lang=en&bolsa_ref=vieja&estado=renuncia&cursor=50", "llamamientos", CORTE_GLOBAL, BOLSA.bolsa_ref);
  const url = new URL(ruta, "https://vec.example/portal-empleado/");
  assert.equal(url.hash, "#bolsa/bolsa-candidatos");
  assert.equal(url.searchParams.get("lang"), "en");
  assert.equal(url.searchParams.has("bolsa_ref"), false);
  assert.equal(url.searchParams.has("cursor"), false);
  assert.deepEqual(leerGlobalBolsaCompartible(url.search), { filtro: "llamamientos", corte: CORTE_GLOBAL, bolsa: BOLSA.bolsa_ref });
  assert.equal(leerGlobalBolsaCompartible("?lang=en"), null);
  for (const search of ["?bolsa_global=todos&bolsa_global=renuncia", "?bolsa_global=renuncia&bolsa_ref=vieja",
    "?bolsa_global=disponible&bolsa_curso=bolsa:1", "?bolsa_global=todos&corte_bolsa=mal", "?corte_bolsa=" + CORTE_GLOBAL]) {
    assert.throws(() => leerGlobalBolsaCompartible(search), search);
  }
});

test("la lista global consulta una página de 50 y exige el mismo corte", async () => {
  const rutas = [];
  const fetchImpl = async (ruta) => {
    rutas.push(new URL(ruta, "https://vec.example"));
    return new Response(JSON.stringify({ data: { esquema: "vec.bolsa.rrhh.global.v1", generado_en: "2026-10-08T10:00:00Z",
      corte_ref: CORTE_GLOBAL, filtro: "disponible", bolsa_ref: null, total: 1, desde: 1, hasta: 1,
      hay_mas: false, cursor_siguiente: null, items: [{ bolsa_ref: BOLSA.bolsa_ref, categoria: BOLSA.categoria,
        participacion_ref: CANDIDATO.participacion_ref, orden_acta: 1, estado_clave: "disponible", estado_desde: CANDIDATO.estado_desde, disponible_desde: null }] } }), { status: 200 });
  };
  const respuesta = await consultarGlobalBolsa("disponible", { corte: CORTE_GLOBAL }, { fetchImpl });
  assert.equal(respuesta.ok, true);
  assert.equal(rutas[0].searchParams.get("limite"), "50");
  assert.equal(rutas[0].searchParams.get("corte"), CORTE_GLOBAL);
  assert.equal(rutas[0].searchParams.has("cursor"), false);
  assert.equal(rutas.length, 1);
});

test("la relación global minimiza referencias y ofrece paginación accesible", () => {
  const html = renderizarGlobalBolsa({ global: true, carga: "listo", filtro: "llamamientos", datos: {
    desde: 1, hasta: 1, total: 51, hay_mas: true, cursor_siguiente: "1", items: [{
      bolsa_ref: BOLSA.bolsa_ref, categoria: BOLSA.categoria, llamamiento_ref: "llamamiento:prueba:00000001",
      referencia: "secreto:referencia", emitido_en: "2026-10-08T10:00:00Z", participaciones: 2,
    }],
  } }, { encabezadoVista: (_s, titulo, _d, acciones) => `<header><h2>${titulo}</h2>${acciones}</header>`,
    escaparHTML: (valor) => String(valor ?? "").replaceAll("&", "&amp;").replaceAll('"', "&quot;") });
  assert.match(html, /class="tabla-contenedor" tabindex="0" role="region"/u);
  assert.match(html, /data-bolsa-accion="pagina-global" data-cursor="1"/u);
  assert.match(html, /data-accion="ver-bolsa"/u);
  assert.doesNotMatch(html, /secreto/u);
  assert.doesNotMatch(html, /data-seguimiento|<th scope="col">Aspirantes<\/th>/u);
});

test("los resúmenes nuevos conservan los campos validados y activan enlaces con corte", async () => {
  const listas = await consultarBolsas({ fetchImpl: async () => new Response(JSON.stringify({ data: {
    esquema: "vec.bolsa.rrhh.bolsas.v1", generado_en: "2026-10-08T10:00:00Z", corte_ref: CORTE_GLOBAL,
    bolsas: [{ ...BOLSA, politica_orden: { ...BOLSA.politica_orden, rotulo: "Orden vigente" }, lista_llamamientos_disponible: true }],
  } }), { status: 200 }) });
  assert.equal(listas.ok, true, listas.mensaje);
  assert.equal(listas.datos.corte_ref, CORTE_GLOBAL);
  assert.equal(listas.datos.bolsas[0].lista_llamamientos_disponible, true);
  const stats = await consultarEstadisticasBolsa({ fetchImpl: async () => new Response(JSON.stringify({ data: {
    esquema: "vec.bolsa.rrhh.estadisticas.v1", generado_en: "2026-10-08T10:00:00Z", corte_ref: CORTE_GLOBAL,
    bolsas: { total: 1, vigentes: 1, sustituidas: 0 }, personas: { total: 2, por_estado: BOLSA.por_estado },
    llamamientos: { total: 1, en_curso_total: 1, lista_llamamientos_disponible: true, por_canal: {}, por_resultado: {} },
    por_bolsa: [{ bolsa_ref: BOLSA.bolsa_ref, categoria: BOLSA.categoria, tipo_lista: "rotatoria", vigente: true, total: 2, por_estado: BOLSA.por_estado }],
  } }), { status: 200 }) });
  assert.equal(stats.ok, true, stats.mensaje);
  assert.equal(stats.datos.llamamientos.en_curso_total, 1);
  assert.equal(stats.datos.lista_llamamientos_disponible, true);
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

function presentador(filtros = { estado: "", texto: "" }, modalFicha = null, globalDisponible = false) {
  return crearPresentadorPanelInterno({
    claseEstado: (clave) => `chip-${clave}`,
    encabezadoVista: (_s, titulo, _d, acciones = "") => `<header><h2>${titulo}</h2>${acciones}</header>`,
    escaparHTML: (valor) => String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll('"', "&quot;"),
    numero: (valor) => String(valor ?? 0),
    obtenerDatosPanel: () => ({ esquema: "vec.bolsa.panel.interno.v1" }),
    tituloVista: (vista) => vista,
    obtenerDatosBolsas: () => ({ carga: "listo", datos: { bolsas: globalDisponible ? [{ ...BOLSA, lista_llamamientos_disponible: true }] : [BOLSA], ...(globalDisponible ? { corte_ref: CORTE_GLOBAL } : {}) } }),
    obtenerDatosCandidatosBolsa: () => ({ carga: "listo", datos: { bolsa: BOLSA, candidatos: [CANDIDATO], contactos: [], hay_mas: false } }),
    obtenerEstadoCandidatos: () => filtros,
    obtenerModalFicha: () => modalFicha,
    obtenerDatosEstadisticas: () => ({ carga: "listo", datos: {
      generado_en: "2026-09-26T09:52:00Z", ...(globalDisponible ? { corte_ref: CORTE_GLOBAL, lista_llamamientos_disponible: true } : {}), bolsas: { total: 1, vigentes: 1, sustituidas: 0 },
      personas: { total: 2, por_estado: { disponible: 2 } }, llamamientos: { total: 1, ...(globalDisponible ? { en_curso_total: 1 } : {}), por_canal: {}, por_resultado: {} },
      por_bolsa: [{ bolsa_ref: BOLSA.bolsa_ref, categoria: BOLSA.categoria, tipo_lista: "rotatoria", vigente: true, total: 2, por_estado: BOLSA.por_estado }],
    } }),
  });
}

test("las cifras globales habilitadas abren la lista exacta sin segunda lectura", () => {
  const vista = presentador(undefined, null, true);
  const cuadro = vista.renderizarSoloBolsas("resumen");
  const estadisticas = vista.renderizarEstadisticasBolsa();
  assert.match(cuadro, /bolsa_global=todos&amp;corte_bolsa=/u);
  assert.match(cuadro, /bolsa_global=disponible&amp;corte_bolsa=/u);
  assert.match(cuadro, /bolsa_global=renuncia&amp;corte_bolsa=/u);
  assert.match(cuadro, /bolsa_global=llamamientos&amp;corte_bolsa=/u);
  assert.match(estadisticas, /bolsa_global=llamamientos&amp;corte_bolsa=/u);
  assert.match(estadisticas, /<strong class="valor-kpi">1<\/strong><span class="etiqueta-kpi">Llamamientos en curso/u);
});

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

test("Personas en bolsa enlaza la lista completa y quita estado, texto y cursor anteriores", () => {
  const anterior = globalThis.location;
  globalThis.location = { search: "?lang=en&bolsa_ref=bolsa%3Aanterior&estado=renuncia&cursor=obsoleto&texto=Sara" };
  try {
    const html = presentador({ estado: "renuncia", texto: "Sara", pestana: "historico", pagina_historico: 2 })
      .renderizarVista("bolsa-candidatos");
    const enlace = html.match(/<dt>(<a[^>]+>Personas en bolsa<\/a>)<\/dt><dd>2<\/dd>/u)?.[1];
    assert.ok(enlace, "la etiqueta del total del servidor se ofrece como enlace accesible");
    assert.match(enlace, /data-accion="ver-bolsa" data-bolsa-ref="bolsa:sintetica:1" data-pestana="candidatos"/u);
    const ruta = enlace.match(/href="([^"]+)"/u)?.[1]?.replaceAll("&amp;", "&");
    const url = new URL(ruta, "https://vec.example/portal-empleado/");
    assert.equal(url.hash, "#bolsa/bolsa-candidatos");
    assert.equal(url.searchParams.get("lang"), "en");
    assert.equal(url.searchParams.get("bolsa_ref"), BOLSA.bolsa_ref);
    for (const clave of ["estado", "texto", "cursor"]) assert.equal(url.searchParams.has(clave), false);
    assert.match(rutaCandidatosBolsaCompartible("?lang=en&texto=busqueda-deliberada", BOLSA.bolsa_ref),
      /texto=busqueda-deliberada/u, "otros enlaces conservan sus búsquedas");
  } finally { globalThis.location = anterior; }
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

test("el controlador de reserva abre Personas en Candidatos con filtros limpios", async () => {
  const escuchas = {};
  const consultas = [];
  const estado = { vista: "bolsa-candidatos", bolsaSeleccionada: BOLSA.bolsa_ref,
    filtrosBolsa: { estado: "renuncia", texto: "Sara", pestana: "historico", pagina_historico: 2 },
    datosCandidatos: { carga: "listo", datos: { bolsa: BOLSA, candidatos: [] } } };
  const documento = { addEventListener(tipo, fn) { escuchas[tipo] = fn; }, querySelector: () => null, querySelectorAll: () => [] };
  const controlador = crearControladorBolsas({ estado, documento, navegar() {}, renderizar() {},
    obtenerFuenteLectura: () => ({ async consultarCandidatosBolsa(ref, filtros) {
      consultas.push({ ref, filtros });
      return { ok: true, datos: { bolsa: BOLSA, candidatos: [], contactos: [], hay_mas: false } };
    } }) });
  controlador.instalar();
  const boton = { dataset: { accion: "ver-bolsa", bolsaRef: BOLSA.bolsa_ref, pestana: "candidatos" } };
  escuchas.click({ preventDefault() {}, target: { closest(selector) {
    return selector === '[data-accion="ver-bolsa"], [data-bolsa-abrir="true"]' ? boton : null;
  } } });
  await new Promise((resolve) => setTimeout(resolve, 0));
  assert.deepEqual(estado.filtrosBolsa, { estado: "", texto: "", pestana: "candidatos" });
  assert.deepEqual(consultas, [{ ref: BOLSA.bolsa_ref, filtros: { estado: "", texto: "", cursor: "" } }]);
});

test("desde Histórico, el enlace Personas fija URL y pestaña Candidatos con una sola lectura", async () => {
  const portal = await readFile(new URL("portal.js", import.meta.url), "utf8");
  const inicioRuta = portal.indexOf("function aplicarRutaCandidatosBolsa()");
  const finRuta = portal.indexOf("function actualizarVistaBolsa(", inicioRuta);
  const inicioEnlace = portal.indexOf("function instalarEnlacesBolsa()");
  const finEnlace = portal.indexOf("function instalarEventosAuditoriaBolsa()", inicioEnlace);
  const location = { origin: "https://vec.example", pathname: "/portal-empleado/",
    search: "?lang=en&bolsa_ref=bolsa%3Asintetica%3A1&estado=renuncia&cursor=viejo&texto=Sara",
    hash: "#bolsa/bolsa-candidatos" };
  Object.defineProperty(location, "href", { get() { return `${location.origin}${location.pathname}${location.search}${location.hash}`; } });
  const history = { pushState(_a, _b, ruta) { const url = new URL(ruta, location.href);
    location.search = url.search; location.hash = url.hash; }, replaceState() { assert.fail("ruta válida"); } };
  const estado = { vista: "bolsa-candidatos", bolsaSeleccionada: BOLSA.bolsa_ref,
    datosBolsas: { carga: "listo", datos: { bolsas: [BOLSA] } },
    filtrosBolsa: { estado: "renuncia", texto: "Sara", pestana: "historico", pagina_historico: 2 },
    datosCandidatos: { carga: "listo", datos: { bolsa: BOLSA, candidatos: [] } } };
  const escuchas = {};
  const documento = { addEventListener(tipo, fn) { escuchas[tipo] = fn; } };
  const enlace = { tagName: "A", dataset: { bolsaRef: BOLSA.bolsa_ref, pestana: "candidatos" },
    classList: { contains: () => false }, closest: () => null, hasAttribute: () => true,
    getAttribute: () => { const parametros = new URLSearchParams(location.search); parametros.delete("texto");
      return rutaCandidatosBolsaCompartible(parametros.toString(), BOLSA.bolsa_ref); } };
  const consultas = [];
  const contexto = { document: documento, window: { location, addEventListener() {} }, history, estado,
    porId: () => ({ contains: (control) => control === enlace, querySelector: () => null }),
    rutasBolsa: rutasGlobales, prepararRutasBolsa: async () => rutasGlobales,
    controladorBolsas: { cargarCandidatosBolsa(ref, opciones) { consultas.push({ ref, opciones }); return Promise.resolve(); } },
    rutaCandidatosAplicada: null, navegar() {}, anunciar(mensaje) { assert.fail(mensaje); },
    traducirPortal: (clave) => clave, moduloDeVistaPortal: () => "bolsa", URL };
  const funciones = runInNewContext(`${portal.slice(inicioRuta, finRuta)}\n${portal.slice(inicioEnlace, finEnlace)};
    ({ aplicarRutaCandidatosBolsa, instalarEnlacesBolsa })`, contexto);
  funciones.instalarEnlacesBolsa();
  await escuchas.click({ target: { closest: (selector) => selector === '[data-accion="ver-bolsa"][data-bolsa-ref]' ? enlace : null },
    stopImmediatePropagation() {}, preventDefault() {}, button: 0 });
  funciones.aplicarRutaCandidatosBolsa();
  assert.deepEqual(JSON.parse(JSON.stringify(consultas)), [{ ref: BOLSA.bolsa_ref, opciones: { enfocarDestino: true } }]);
  assert.deepEqual(JSON.parse(JSON.stringify(estado.filtrosBolsa)), { estado: "", texto: "" });
  assert.equal(location.search, "?lang=en&bolsa_ref=bolsa%3Asintetica%3A1");
  assert.equal(location.hash, "#bolsa/bolsa-candidatos");
});

test("Personas en bolsa limpia datos previos al recibir 401 o 403", async () => {
  for (const status of [401, 403]) {
    const escuchas = {};
    const estado = { bolsaSeleccionada: BOLSA.bolsa_ref,
      filtrosBolsa: { estado: "renuncia", texto: "Sara", pestana: "historico" },
      datosCandidatos: { carga: "listo", datos: { bolsa: BOLSA, candidatos: [CANDIDATO] } } };
    const documento = { addEventListener(tipo, fn) { escuchas[tipo] = fn; }, querySelector: () => null, querySelectorAll: () => [] };
    const controlador = crearControladorBolsas({ estado, documento, navegar() {}, renderizar() {},
      obtenerFuenteLectura: () => ({ async consultarCandidatosBolsa() {
        return { ok: false, status, mensaje: "Acceso denegado" };
      } }) });
    controlador.instalar();
    const boton = { dataset: { accion: "ver-bolsa", bolsaRef: BOLSA.bolsa_ref, pestana: "candidatos" } };
    escuchas.click({ preventDefault() {}, target: { closest(selector) {
      return selector === '[data-accion="ver-bolsa"], [data-bolsa-abrir="true"]' ? boton : null;
    } } });
    await new Promise((resolve) => setTimeout(resolve, 0));
    assert.equal(estado.datosCandidatos.carga, "denegado");
    assert.equal(estado.datosCandidatos.datos, null);
  }
});

test("una lectura tardía de Personas no sustituye la bolsa abierta después", async () => {
  const escuchas = {};
  const segundaRef = "bolsa:sintetica:2";
  const segundaBolsa = { ...BOLSA, bolsa_ref: segundaRef };
  let resolverPrimera;
  const estado = { bolsaSeleccionada: BOLSA.bolsa_ref,
    filtrosBolsa: { estado: "renuncia", texto: "Sara", pestana: "historico" },
    datosCandidatos: { carga: "listo", datos: { bolsa: BOLSA, candidatos: [CANDIDATO] } } };
  const documento = { addEventListener(tipo, fn) { escuchas[tipo] = fn; }, querySelector: () => null, querySelectorAll: () => [] };
  const controlador = crearControladorBolsas({ estado, documento, navegar() {}, renderizar() {},
    obtenerFuenteLectura: () => ({ consultarCandidatosBolsa(ref) {
      if (ref === BOLSA.bolsa_ref) return new Promise((resolver) => { resolverPrimera = resolver; });
      return Promise.resolve({ ok: true, datos: { bolsa: segundaBolsa, candidatos: [], contactos: [], hay_mas: false } });
    } }) });
  controlador.instalar();
  const pulsar = (ref) => { const boton = { dataset: { accion: "ver-bolsa", bolsaRef: ref, pestana: "candidatos" } };
    escuchas.click({ preventDefault() {}, target: { closest(selector) {
      return selector === '[data-accion="ver-bolsa"], [data-bolsa-abrir="true"]' ? boton : null;
    } } }); };
  pulsar(BOLSA.bolsa_ref);
  pulsar(segundaRef);
  await new Promise((resolve) => setTimeout(resolve, 0));
  resolverPrimera({ ok: true, datos: { bolsa: BOLSA, candidatos: [CANDIDATO], contactos: [], hay_mas: false } });
  await new Promise((resolve) => setTimeout(resolve, 0));
  assert.equal(estado.bolsaSeleccionada, segundaRef);
  assert.equal(estado.datosCandidatos.datos.bolsa.bolsa_ref, segundaRef);
});


test("actualizar tras caducar consulta una vez y fija el corte nuevo para F5", async () => {
  const portal = await readFile(new URL("portal.js", import.meta.url), "utf8");
  const inicio = portal.indexOf("function aplicarRutaCandidatosBolsa()");
  const fin = portal.indexOf("function actualizarVistaBolsa(", inicio);
  const corteNuevo = "b".repeat(64);
  const location = { pathname: "/portal-empleado/", search: `?lang=es&bolsa_global=todos&corte_bolsa=${CORTE_GLOBAL}`, hash: "#bolsa/bolsa-candidatos" };
  const history = { replaceState(_a, _b, ruta) { const url = new URL(ruta, "https://vec.example"); location.search = url.search; location.hash = url.hash; } };
  const estado = { vista: "bolsa-candidatos", datosBolsas: { carga: "listo", datos: { bolsas: [BOLSA], corte_ref: CORTE_GLOBAL } }, filtrosBolsa: {} };
  const escuchas = [];
  const documento = { addEventListener(tipo, fn) { if (tipo === "click") escuchas.push(fn); }, querySelector: () => null, querySelectorAll: () => [] };
  const previo = { fetch: globalThis.fetch, location: globalThis.location, history: globalThis.history };
  const consultas = [];
  globalThis.location = location; globalThis.history = history;
  globalThis.fetch = async (ruta) => {
    consultas.push(new URL(ruta, "https://vec.example"));
    if (consultas.length === 1) return new Response("{}", { status: 409 });
    return new Response(JSON.stringify({ data: { esquema: "vec.bolsa.rrhh.global.v1", generado_en: "2026-10-08T10:00:00Z", corte_ref: corteNuevo,
      filtro: "todos", bolsa_ref: null, total: 0, desde: 0, hasta: 0, hay_mas: false, cursor_siguiente: null, items: [] } }), { status: 200 });
  };
  let aplicar = () => {};
  const controlador = crearControladorBolsas({ estado, documento, navegar() {}, renderizar: () => aplicar() });
  const contexto = { estado, rutasBolsa: rutasGlobales, controladorBolsas: controlador, rutaCandidatosAplicada: null, window: { location }, history,
    navegar() {}, anunciar() {}, traducirPortal: (clave) => clave };
  aplicar = runInNewContext(`${portal.slice(inicio, fin)}; aplicarRutaCandidatosBolsa`, contexto);
  const esperar = async (condicion) => { for (let n = 0; n < 100 && !condicion(); n++) await new Promise((r) => setTimeout(r, 5)); assert.ok(condicion()); };
  try {
    controlador.instalar(); aplicar();
    await esperar(() => estado.datosCandidatos?.carga === "caducado");
    const boton = { dataset: { bolsaAccion: "actualizar-global" } };
    for (const escucha of escuchas) escucha({ preventDefault() {}, target: { closest: (selector) => selector === "[data-bolsa-accion]" ? boton : null } });
    await esperar(() => estado.datosCandidatos?.carga === "listo");
    assert.equal(consultas.length, 2);
    assert.equal(consultas[0].searchParams.get("corte"), CORTE_GLOBAL);
    assert.equal(consultas[1].searchParams.has("corte"), false);
    assert.equal(new URLSearchParams(location.search).get("corte_bolsa"), corteNuevo);
    aplicar(); assert.equal(consultas.length, 2, "repintar el corte nuevo no duplica GET");
  } finally { controlador.cancelarPeticiones(); Object.assign(globalThis, previo); }
});


test("una página tardía no cambia la URL ni el resumen al volver durante la carga", async () => {
  const location = { pathname: "/portal-empleado/", search: `?bolsa_global=todos&corte_bolsa=${CORTE_GLOBAL}`, hash: "#bolsa/bolsa-candidatos" };
  const previo = { fetch: globalThis.fetch, location: globalThis.location, history: globalThis.history };
  let resolver, pedidos = 0, escrituras = 0;
  globalThis.location = location;
  globalThis.history = { replaceState() { escrituras++; } };
  globalThis.fetch = () => { pedidos++; return new Promise((r) => { resolver = r; }); };
  const estado = { vista: "bolsa-candidatos", filtrosBolsa: {} };
  const controlador = crearControladorBolsas({ estado, renderizar() {}, navegar() {}, documento: { querySelector: () => null } });
  try {
    const carga = controlador.cargarGlobalBolsa("todos", { corte: CORTE_GLOBAL });
    for (let n = 0; n < 100 && !resolver; n++) await new Promise((r) => setTimeout(r, 5));
    assert.equal(pedidos, 1);
    estado.vista = "resumen"; location.search = ""; location.hash = "#bolsa/resumen";
    resolver(new Response(JSON.stringify({ data: { esquema: "vec.bolsa.rrhh.global.v1", generado_en: "2026-10-08T10:00:00Z", corte_ref: CORTE_GLOBAL,
      filtro: "todos", bolsa_ref: null, total: 0, desde: 0, hasta: 0, hay_mas: false, cursor_siguiente: null, items: [] } }), { status: 200 }));
    await carga;
    assert.equal(escrituras, 0); assert.equal(location.hash, "#bolsa/resumen");
    assert.equal(estado.datosCandidatos, null);
  } finally { controlador.cancelarPeticiones(); Object.assign(globalThis, previo); }
});
