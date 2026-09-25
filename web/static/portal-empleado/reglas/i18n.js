/** Textos de la pantalla de reglas vigentes. Todo texto visible pasa por aquí. */
export const MENSAJES_REGLAS_ES = Object.freeze({
  documentTitle: "Reglas vigentes · Portal del Empleado",
  miga: "Portal del Empleado → Bolsa y Contratación temporal → Reglas vigentes",
  titulo: "Reglas vigentes",
  volver: "Volver al portal",
  ayudaAbrir: "Ayuda",
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

const CLAVES = Object.freeze(Object.keys(MENSAJES_REGLAS_ES));

export function crearTraductorReglas(catalogo = MENSAJES_REGLAS_ES) {
  if (!catalogo || typeof catalogo !== "object"
    || CLAVES.some((clave) => typeof catalogo[clave] !== "string" || catalogo[clave] === "")) {
    throw new Error("catálogo i18n de reglas incompleto");
  }
  return (clave, variables = {}) => {
    if (!Object.hasOwn(catalogo, clave)) throw new Error(`clave i18n de reglas desconocida: ${clave}`);
    return catalogo[clave].replace(/\{([a-z_]+)\}/g, (_c, v) => String(variables[v] ?? ""));
  };
}

/** ¿Existe la clave? Para textos que dependen de un valor del servidor. */
export const existeClaveReglas = (clave) => Object.hasOwn(MENSAJES_REGLAS_ES, clave);

export function formatearNumero(n) {
  return new Intl.NumberFormat("es-ES").format(n);
}
