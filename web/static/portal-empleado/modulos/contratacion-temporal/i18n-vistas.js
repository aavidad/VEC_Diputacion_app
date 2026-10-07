/** Lecturas de textos que corresponden a la vista CT que se abre. */
import { IDIOMA_ACTUAL, IDIOMA_POR_DEFECTO, INDICE_IDIOMAS, prepararIdiomas } from "../../../comun/idioma.js";
import { cargarTextos, reintentarTextos } from "../../../comun/textos.js";
import { FASES_RRHH } from "./fases-rrhh-datos.js";

const CUADRO = Object.freeze([
  ["portal", "fases_rrhh"],
  ["portal", "general"],
  ["portal", "textos"],
  ["contratacion-temporal-ficha-lista", "general"],
  ["contratacion-temporal-lista-plazos", "lista"],
]);
const EXPEDIENTE = Object.freeze([
  ...CUADRO,
  ["contratacion-temporal-analisis-catalogo", "general"],
  ["contratacion-temporal-cambios-expediente", "general"],
  ["contratacion-temporal-firma-incorporacion-expedientes", "incorporacion"],
  ["contratacion-temporal-firma-incorporacion-expedientes", "hito"],
  ["contratacion-temporal-firma-incorporacion-expedientes", "continuidad"],
  ["contratacion-temporal-firma-incorporacion-expedientes", "firma"],
  ["contratacion-temporal-firma-incorporacion-expedientes", "borrador"],
  ["contratacion-temporal-firma-incorporacion-expedientes", "firma_pendiente"],
]);
const VISTAS = Object.freeze({
  cuadro: CUADRO,
  expediente: EXPEDIENTE,
  alta: Object.freeze([
    ["portal", "fases_rrhh"],
    ["contratacion-temporal-moad", "general"],
    ["contratacion-temporal-textos-vistas", "general"],
    ["contratacion-temporal-analisis-catalogo", "general"],
  ]),
  firma: Object.freeze([
    ...EXPEDIENTE,
    ["contratacion-temporal-circuito-firma", "general"],
    ["contratacion-temporal-firma-incorporacion-portal", "general"],
  ]),
});

async function cargarGrupo(fuentes, idioma, { leer, reintentar }) {
  const modulos = [...new Set(fuentes.map(([modulo]) => modulo))];
  const entradas = await Promise.all(modulos.map(async (modulo) => {
    const catalogo = reintentar && leer === undefined
      ? await reintentarTextos(modulo, { idioma })
      : await cargarTextos(modulo, { idioma, ...(leer ? { leer } : {}) });
    return [modulo, catalogo];
  }));
  return Object.fromEntries(entradas);
}

/**
 * Devuelve secciones congeladas y el idioma efectivo. Una caída del elegido
 * lleva a recargar el conjunto completo en el idioma por defecto.
 */
export async function prepararTextosContratacionVista(vista, { idioma, leer, reintentar = false } = {}) {
  if (!Object.hasOwn(VISTAS, vista)) throw new TypeError("vista CT de textos no válida");
  let incidenciaPreparacion = null;
  try { await prepararIdiomas(); }
  catch (error) { incidenciaPreparacion = error; }
  if (idioma !== undefined && !INDICE_IDIOMAS.idiomas.some(({ codigo }) => codigo === idioma)) {
    throw new RangeError("idioma CT no admitido");
  }
  const solicitado = idioma ?? IDIOMA_ACTUAL;
  let textos = await cargarGrupo(VISTAS[vista], solicitado, { leer, reintentar });
  if (Object.values(textos).some((catalogo) => catalogo.incidenciaIndice === undefined
    || catalogo.incidenciaCatalogo === undefined)) throw new TypeError("metadatos de textos CT incompletos");
  const incidencias = [incidenciaPreparacion, ...Object.values(textos).flatMap((catalogo) =>
    [catalogo.incidenciaIndice, catalogo.incidenciaCatalogo])].filter((valor) => valor !== null);
  if (Object.values(textos).some((catalogo) => catalogo.idioma !== solicitado)) {
    if (solicitado === IDIOMA_POR_DEFECTO) throw new Error("catálogos CT del idioma por defecto no disponibles");
    textos = await cargarGrupo(VISTAS[vista], IDIOMA_POR_DEFECTO, { leer, reintentar });
  }
  const efectivos = new Set(Object.values(textos).map((catalogo) => catalogo.idioma));
  if (efectivos.size !== 1) throw new Error("catálogos CT con idiomas distintos");
  const secciones = Object.freeze(Object.fromEntries(VISTAS[vista].map(([modulo, seccion]) =>
    [`${modulo}.${seccion}`, textos[modulo].seccion(seccion)])));
  return Object.freeze({ vista, idioma: [...efectivos][0], secciones,
    incidencias: Object.freeze(incidencias), reintentar: incidencias.length > 0 });
}

/** Traducción sin lecturas adicionales para la lista y su paginación. */
export function crearTraductorCuadroCT(preparado) {
  if (!preparado || !["cuadro", "expediente", "firma"].includes(preparado.vista)) {
    throw new TypeError("textos de cuadro CT no preparados");
  }
  const secciones = preparado.secciones;
  const fases = secciones["portal.fases_rrhh"];
  const mensajes = {
    ...secciones["contratacion-temporal-ficha-lista.general"],
    ...secciones["contratacion-temporal-lista-plazos.lista"],
  };
  for (const fase of FASES_RRHH) mensajes[`etiqueta_fase_${fase}`] = fases[`fase_${fase}`];
  for (const estado of ["pendiente", "en_curso", "espera", "completado", "incidencia", "cancelado"]) {
    mensajes[`fase_${estado}`] = fases[`estado_${estado}`];
  }
  mensajes.fase_rrhh_orden = fases.fase_de;
  const traducir = (clave, variables = {}) => {
    const plantilla = mensajes[clave] ?? secciones["portal.general"]?.[clave]
      ?? secciones["portal.textos"]?.[clave];
    if (typeof plantilla !== "string") throw new Error(`clave CT de cuadro desconocida: ${clave}`);
    return plantilla.replace(/\{([A-Za-z_][A-Za-z0-9_]*)\}/gu,
      (_coincidencia, nombre) => String(variables[nombre] ?? ""));
  };
  return Object.freeze(traducir);
}
