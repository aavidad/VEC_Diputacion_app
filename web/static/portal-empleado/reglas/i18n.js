import { IDIOMA_ACTUAL, LOCALIZACION_ACTUAL } from "../../comun/idioma.js";

/** Textos de la pantalla de reglas vigentes. Todo texto visible pasa por aquí. */
export const MENSAJES_REGLAS_ES = Object.freeze({
  documentTitle: "Reglas vigentes · Portal del Empleado",
  miga: "Portal del Empleado → Bolsa y Contratación temporal → Reglas vigentes",
  titulo: "Reglas vigentes",
  volver: "Volver al portal",
  ayudaAbrir: "Ayuda",
  ayudaSimbolo: "?",
  ayudaTitulo: "Reglas vigentes",
  ayudaCerrar: "Cerrar",
  ayudaQue: "Esta pantalla muestra, sin permitir cambios, los plazos, límites y listas que aplican hoy Bolsa y Contratación temporal.",
  ayudaOrigen: "«Reglamento» cita el artículo del Reglamento de bolsas publicado. «Ejemplo» es un valor de trabajo que RRHH todavía no ha aprobado; la columna «Duda de RRHH» indica qué pregunta lo resolverá.",
  ayudaVersion: "Cada regla se identifica por su catálogo y versión. Un cambio de valor se publica como versión nueva del catálogo, sin tocar el programa.",
  filtros: "Filtros de reglas",
  filtroModulo: "Módulo",
  filtroOrigen: "Origen",
  filtroTexto: "Buscar",
  todos: "Todos",
  origen_reglamento: "Reglamento",
  origen_ejemplo: "Ejemplo",
  cargando: "Cargando reglas…",
  sinResultados: "Ninguna regla coincide con los filtros.",
  kpiTotal: "Reglas vigentes",
  kpiReglamento: "Del Reglamento",
  kpiEjemplo: "De ejemplo",
  modulo_bolsa: "Bolsa",
  modulo_contratacion_temporal: "Contratación temporal",
  catalogoVersion: "Catálogo {catalogo} · versión {version}",
  catalogoHuella: "Huella {huella}",
  paqueteEjemplo: "Paquete de ejemplo",
  estado_sin_catalogo: "Sin catálogo de reglas cargado: el módulo aplica su comportamiento actual.",
  estado_no_disponible: "El catálogo de reglas no está disponible en este momento.",
  contadorReglas: "{cantidad} reglas",
  contadorRegla: "{cantidad} regla",
  colRegla: "Regla",
  colValor: "Valor",
  colUnidad: "Unidad",
  colOrigen: "Origen",
  colDuda: "Duda de RRHH",
  colVersion: "Versión",
  origenArticulo: "Reglamento, {articulo}",
  origenEjemplo: "Ejemplo",
  parteEjemplo: "Parte de ejemplo: {texto}",
  sinValor: "No aplica",
  computo_administrativo: "Cómputo administrativo",
  computo_civil: "Cómputo civil",
  unidad_dias_habiles: "Días hábiles",
  unidad_dias_naturales: "Días naturales",
  unidad_meses: "Meses",
  unidad_anios: "Años",
  unidad_horas: "Horas",
  unidad_minutos_semanales: "Minutos semanales",
  unidad_intentos: "Intentos",
  unidad_procesos: "Procesos",
  unidad_franja_horaria: "Franja horaria",
  unidad_lista: "Lista",
  unidad_ninguna: "Sin cantidad",
  error_solicitud_invalida: "La consulta no es válida.",
  error_servicio_no_disponible: "El servicio de reglas no está disponible. Inténtelo más tarde.",
  error_autenticacion_requerida: "Identifíquese con su certificado para consultar las reglas.",
  error_acceso_denegado: "Su perfil no permite consultar las reglas.",
  error_respuesta: "La respuesta del servidor no es válida.",
});

export const MENSAJES_REGLAS_EN = Object.freeze({
  documentTitle: "Current rules · Employee Portal",
  miga: "Employee Portal → Candidate pool and temporary recruitment → Current rules",
  titulo: "Current rules",
  volver: "Back to the portal",
  ayudaAbrir: "Help",
  ayudaSimbolo: "?",
  ayudaTitulo: "Current rules",
  ayudaCerrar: "Close",
  ayudaQue: "This read-only page shows the deadlines, limits and lists currently used by the candidate pool and temporary recruitment.",
  ayudaOrigen: "‘Regulation’ cites an article of the published candidate pool regulation. ‘Example’ is a working value that HR has not yet approved; the ‘HR question’ column shows the outstanding question.",
  ayudaVersion: "Each rule has a catalogue and version. Changes to values are published as a new catalogue version without changing the program.",
  filtros: "Rule filters",
  filtroModulo: "Module",
  filtroOrigen: "Source",
  filtroTexto: "Search",
  todos: "All",
  origen_reglamento: "Regulation",
  origen_ejemplo: "Example",
  cargando: "Loading rules…",
  sinResultados: "No rules match the filters.",
  kpiTotal: "Current rules",
  kpiReglamento: "From the regulation",
  kpiEjemplo: "Examples",
  modulo_bolsa: "Candidate pool",
  modulo_contratacion_temporal: "Temporary recruitment",
  catalogoVersion: "Catalogue {catalogo} · version {version}",
  catalogoHuella: "Fingerprint {huella}",
  paqueteEjemplo: "Example package",
  estado_sin_catalogo: "No rule catalogue loaded: the module uses its current behaviour.",
  estado_no_disponible: "The rule catalogue is currently unavailable.",
  contadorReglas: "{cantidad} rules",
  contadorRegla: "{cantidad} rule",
  colRegla: "Rule",
  colValor: "Value",
  colUnidad: "Unit",
  colOrigen: "Source",
  colDuda: "HR question",
  colVersion: "Version",
  origenArticulo: "Regulation, {articulo}",
  origenEjemplo: "Example",
  parteEjemplo: "Example excerpt: {texto}",
  sinValor: "Not applicable",
  computo_administrativo: "Administrative calculation",
  computo_civil: "Civil calculation",
  unidad_dias_habiles: "Working days",
  unidad_dias_naturales: "Calendar days",
  unidad_meses: "Months",
  unidad_anios: "Years",
  unidad_horas: "Hours",
  unidad_minutos_semanales: "Minutes per week",
  unidad_intentos: "Attempts",
  unidad_procesos: "Processes",
  unidad_franja_horaria: "Time window",
  unidad_lista: "List",
  unidad_ninguna: "No quantity",
  error_solicitud_invalida: "The request is invalid.",
  error_servicio_no_disponible: "The rule service is unavailable. Please try again later.",
  error_autenticacion_requerida: "Identify yourself with your certificate to view the rules.",
  error_acceso_denegado: "Your profile does not allow you to view the rules.",
  error_respuesta: "The server response is invalid.",
});

const CLAVES = Object.freeze(Object.keys(MENSAJES_REGLAS_ES));
const MARCADORES = /\{([a-z_]+)\}/gu;
const catalogoActual = () => IDIOMA_ACTUAL === "en" ? MENSAJES_REGLAS_EN : MENSAJES_REGLAS_ES;

export function crearTraductorReglas(catalogo = catalogoActual()) {
  if (!catalogo || typeof catalogo !== "object"
    || Object.keys(catalogo).length !== CLAVES.length
    || CLAVES.some((clave) => typeof catalogo[clave] !== "string" || catalogo[clave] === ""
      || [...catalogo[clave].matchAll(MARCADORES)].map((m) => m[1]).sort().join(",")
        !== [...MENSAJES_REGLAS_ES[clave].matchAll(MARCADORES)].map((m) => m[1]).sort().join(","))) {
    throw new Error("catálogo i18n de reglas incompleto");
  }
  return (clave, variables = {}) => {
    if (!Object.hasOwn(catalogo, clave)) throw new Error(`clave i18n de reglas desconocida: ${clave}`);
    return catalogo[clave].replace(/\{([a-z_]+)\}/g, (_c, v) => String(variables[v] ?? ""));
  };
}

/** ¿Existe la clave? Para textos que dependen de un valor del servidor. */
export const existeClaveReglas = (clave) => Object.hasOwn(MENSAJES_REGLAS_ES, clave);

export function formatearNumero(n, localizacion = LOCALIZACION_ACTUAL) {
  return new Intl.NumberFormat(localizacion).format(n);
}
