import test from "node:test";
import { prepararTextosPortal } from "./portal-i18n.js?v=20261007-pantallas-textos-final-v1";
import { prepararMensajesContratos } from "./portal-bolsas-contratos.js?v=20261007-pantallas-textos-final-v1";
await prepararTextosPortal("bolsa");
await prepararMensajesContratos();
import assert from "node:assert/strict";
import { validarBolsa, validarCandidato, validarRespuestaCandidatosBolsa, validarRespuestaEstadisticas } from "./portal-bolsas-contrato.js";
import { consultarCandidatosBolsa, consultarEstadisticasBolsa, consultarSeleccionMasivaBolsa, seleccionarParticipacionesPorEstado } from "./portal-bolsas-api.js?v=20261008-bolsa-global-v2";
import { crearPresentadorPanelInterno } from "./portal-panel-interno.js?v=20261008-b1-traza-v1";

const estadosLegacy = { disponible: 1, no_disponible: 0, trabajando: 0, pendiente_incorporacion: 0, renuncia: 0, excluido: 0, disponible_desde: 0 };
const bolsa = { bolsa_ref: "bolsa:sintetica", categoria_clave: "auxiliar", categoria: "Auxiliar", tipo_lista: "cerrada", vigente_desde: "2026-09-01", vigente_hasta: null, total: 1, por_estado: estadosLegacy, llamamientos_en_curso: 0,
  politica_orden: { politica_ref: "politica:sintetica", version: 4, criterio: "puntuacion_desc_acta", tipo_lista: "cerrada", reposicion: "misma_posicion", provisional: false, rotulo: "Orden del acta", actor: "rrhh:sintetico", vigente_desde: "2026-09-01" } };
const candidato = { participacion_ref: "participacion:sintetica", orden: null, orden_acta: 1, razon_orden: "sin_turno", nombre_visible: "Persona sintética", documento_enmascarado: "***0001**", estado_clave: "en_revision", estado_desde: "2026-10-02T10:00:00Z", disponible_desde: null, ultimo_llamamiento: null, contactos_total: 0 };
const respuestaLista = (conteos = estadosLegacy) => ({ data: { esquema: "vec.bolsa.rrhh.candidatos.v1", generado_en: candidato.estado_desde, bolsa: { ...bolsa, por_estado: conteos }, candidatos: [candidato], contactos: [], hay_mas: false, cursor_siguiente: null } });
const respuestaEstadisticas = (conteos = estadosLegacy) => ({ data: { esquema: "vec.bolsa.rrhh.estadisticas.v1", generado_en: candidato.estado_desde, bolsas: { total: 1, vigentes: 1, sustituidas: 0 }, personas: { total: 1, por_estado: conteos }, llamamientos: { total: 0, por_canal: {}, por_resultado: {} }, por_bolsa: [{ bolsa_ref: bolsa.bolsa_ref, categoria: bolsa.categoria, tipo_lista: bolsa.tipo_lista, vigente: true, total: 1, por_estado: conteos }] } });

test("RRHH18 admite mapas históricos de siete estados y mapas nuevos de ocho sin abrir el contrato", () => {
  for (const conteos of [estadosLegacy, { ...estadosLegacy, disponible: 0, en_revision: 1 }]) {
    const esperado = conteos.en_revision ?? 0;
    assert.equal(validarBolsa({ ...bolsa, por_estado: conteos }).por_estado.en_revision, esperado);
    const estadisticas = validarRespuestaEstadisticas(respuestaEstadisticas(conteos));
    assert.equal(estadisticas.personas.por_estado.en_revision, esperado);
    assert.equal(estadisticas.por_bolsa[0].por_estado.en_revision, esperado);
    assert.ok(Object.isFrozen(estadisticas.personas.por_estado));
  }
  for (const conteos of [{ ...estadosLegacy, inventado: 1 }, { ...estadosLegacy, en_revision: -1 }, { ...estadosLegacy, en_revision: "1" }, { ...estadosLegacy, en_revision: null }, Object.fromEntries(Object.entries(estadosLegacy).filter(([clave]) => clave !== "trabajando"))]) {
    assert.throws(() => validarBolsa({ ...bolsa, por_estado: conteos }));
    assert.throws(() => validarRespuestaEstadisticas(respuestaEstadisticas(conteos)));
  }
  assert.equal(Object.hasOwn(estadosLegacy, "en_revision"), false);
  assert.equal(validarCandidato(candidato).estado_clave, "en_revision");
  assert.throws(() => validarCandidato({ ...candidato, estado_clave: "inventado" }));
});

test("RRHH18 las consultas HTTP aceptan revisión en lista y estadísticas con contratos cerrados", async () => {
  const conteos = { ...estadosLegacy, disponible: 0, en_revision: 1 };
  const consulta = await consultarCandidatosBolsa(bolsa.bolsa_ref, { estado: "en_revision" }, { fetchImpl: async (ruta, opciones) => {
    assert.match(ruta, /estado=en_revision/);
    assert.equal(opciones.cache, "no-store");
    return { ok: true, status: 200, json: async () => respuestaLista(conteos) };
  } });
  assert.equal(consulta.ok, true);
  assert.equal(consulta.datos.candidatos[0].estado_clave, "en_revision");
  const estadisticas = await consultarEstadisticasBolsa({ fetchImpl: async () => ({ ok: true, status: 200, json: async () => respuestaEstadisticas(conteos) }) });
  assert.equal(estadisticas.ok, true);
  assert.equal(estadisticas.datos.personas.por_estado.en_revision, 1);
});

test("RRHH18 muestra revisión traducida en ficha, filtro, cuadro y estadísticas", () => {
  const conteos = { ...estadosLegacy, disponible: 0, en_revision: 1 };
  const datos = validarRespuestaCandidatosBolsa(respuestaLista(conteos));
  const flujo = {};
  const presentador = crearPresentadorPanelInterno({ claseEstado: () => "info", encabezadoVista: (_a, titulo) => `<h2>${titulo}</h2>`, escaparHTML: (valor) => String(valor ?? "").replaceAll("<", "&lt;").replaceAll('"', "&quot;"), numero: (valor) => String(valor ?? 0), obtenerDatosPanel: () => ({ esquema: "vec.bolsa.panel.interno.v1" }), tituloVista: (v) => v,
    obtenerDatosBolsas: () => ({ carga: "listo", datos: { bolsas: [datos.bolsa] } }),
    obtenerDatosCandidatosBolsa: () => ({ carga: "listo", datos }), obtenerEstadoCandidatos: () => flujo,
    obtenerDatosEstadisticas: () => ({ carga: "listo", datos: validarRespuestaEstadisticas(respuestaEstadisticas(conteos)) }),
    obtenerModalFicha: () => ({ abierto: true, bolsa: datos.bolsa, candidato: datos.candidatos[0] }) });
  const lista = presentador.renderizarVista("bolsa-candidatos");
  assert.match(lista, /value="en_revision">En revisión<\/option>/);
  assert.match(lista, /<dt>Situación<\/dt><dd><span[^>]*>En revisión<\/span>/);
  assert.match(lista, /data-estado="en_revision"/);
  const estadisticas = presentador.renderizarVista("estadisticas");
  assert.match(estadisticas, /<th scope="col">En revisión<\/th>/);
  assert.match(estadisticas, /data-estado="en_revision"/);
  assert.match(estadisticas, /Histórico de llamamientos[\s\S]*Consulte el histórico de llamamientos de cada bolsa\./u);
  assert.doesNotMatch(estadisticas, /Histórico de llamamientos[\s\S]*No disponible<\/p>/u);
  assert.doesNotMatch(estadisticas, /NaN|undefined/);
});

test("RRHH18 bloquea selección individual y masiva aun con filtros manipulados", async () => {
  const estados = ["disponible", "disponible_desde", "en_revision", "trabajando", "excluido"];
  const candidatos = estados.map((estado_clave, indice) => ({ ...candidato, estado_clave, orden: indice + 1, participacion_ref: `participacion:${estado_clave}` }));
  assert.deepEqual(seleccionarParticipacionesPorEstado(candidatos, estados), ["participacion:disponible", "participacion:disponible_desde"]);
  const masiva = await consultarSeleccionMasivaBolsa(bolsa.bolsa_ref, estados, { consultar: async () => ({ ok: true, datos: { bolsa: { ...bolsa, total: candidatos.length }, candidatos, generado_en: candidato.estado_desde, hay_mas: false, cursor_siguiente: null } }) });
  assert.equal(masiva.ok, true);
  assert.deepEqual(masiva.participaciones, ["participacion:disponible", "participacion:disponible_desde"]);
  const flujo = { nuevo_llamamiento: { paso: 2, estados, participaciones: candidatos.map((c) => c.participacion_ref) } };
  const presentador = crearPresentadorPanelInterno({ claseEstado: () => "info", encabezadoVista: () => "", escaparHTML: String, numero: String, obtenerDatosPanel: () => ({ esquema: "vec.bolsa.panel.interno.v1" }), tituloVista: String, obtenerEstadoCandidatos: () => flujo, obtenerDatosCandidatosBolsa: () => ({ carga: "listo", datos: { bolsa, candidatos, contactos: [] } }) });
  const seleccion = presentador.renderizarVista("bolsa-candidatos");
  for (const estado of ["en_revision", "trabajando", "excluido"]) {
    const control = seleccion.match(new RegExp(`<input[^>]*value="participacion:${estado}"[^>]*>`))?.[0];
    assert.ok(control);
    assert.match(control, / disabled/);
    assert.doesNotMatch(control, / checked/);
  }
});
