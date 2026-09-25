/** Textos de la pantalla de Calendarios. Todo texto visible pasa por aquí. */
export const MENSAJES_CALENDARIOS_ES = Object.freeze({
  documentTitle: "Calendarios laborales · Portal del Empleado",
  miga: "Portal del Empleado → Calendarios",
  titulo: "Calendarios laborales",
  volver: "Volver al portal",
  ayudaAbrir: "Ayuda",
  ayudaTitulo: "Calendarios laborales",
  ayudaCerrar: "Cerrar",
  ayudaCentro: "Cada calendario de centro combina las fiestas nacionales, las de Andalucía, las locales de su municipio y los cierres propios del centro.",
  ayudaHabil: "Los sábados, domingos y festivos son inhábiles para los plazos administrativos. Un cierre interno del centro no cambia el cómputo de plazos.",
  ayudaConocido: "«Conocido en» muestra el calendario tal como constaba en esa fecha, antes de correcciones posteriores.",
  ayudaPlazo: "El plazo se cuenta desde el día siguiente a la notificación. Si el último día es inhábil, vence el primer día hábil siguiente.",
  filtros: "Selección de calendario",
  centro: "Centro",
  anio: "Año",
  conocidoEn: "Conocido en",
  consultar: "Consultar",
  cargando: "Cargando calendario…",
  cargandoCentros: "Cargando centros…",
  sinCentros: "No hay calendarios de centro publicados para este año.",
  kpiHabiles: "Días hábiles",
  kpiLaborables: "Días laborables del centro",
  kpiFestivos: "Festivos oficiales",
  kpiCierres: "Cierres del centro",
  calendarioTitulo: "Calendario {anio}",
  leyenda: "Leyenda",
  leyendaNacional: "Festivo nacional",
  leyendaAutonomico: "Festivo de Andalucía",
  leyendaLocal: "Festivo local",
  leyendaCentro: "Cierre del centro",
  leyendaFinde: "Sábado o domingo",
  semanaCorta: "L,M,X,J,V,S,D",
  diaEtiqueta: "{fecha}: {motivos}",
  diaHabil: "hábil",
  diaInhabil: "inhábil",
  festivosTitulo: "Festivos y cierres",
  colFecha: "Fecha",
  colDenominacion: "Denominación",
  colAmbito: "Ámbito",
  colProcedencia: "Procedencia",
  ambito_nacional: "Nacional",
  ambito_autonomico: "Andalucía",
  ambito_local: "Local",
  ambito_centro: "Centro",
  sintetico: "Sintético",
  oficial: "Oficial",
  versionesTitulo: "Fuentes del calendario",
  colCalendario: "Calendario",
  colVersion: "Versión",
  colNorma: "Norma o acuerdo",
  colConocidoDesde: "Conocido desde",
  plazoTitulo: "Cálculo de plazo",
  plazoNotificacion: "Fecha de notificación",
  plazoUnidad: "Unidad",
  plazoCantidad: "Cantidad",
  plazoResidencia: "Municipio de residencia",
  plazoMismaSede: "El de la sede",
  plazoCalcular: "Calcular",
  unidad_dias_habiles: "Días hábiles",
  unidad_dias_naturales: "Días naturales",
  unidad_meses: "Meses",
  unidad_anios: "Años",
  plazoVence: "Vence el {fecha}",
  plazoProrrogado: "Prorrogado desde el {fecha} por ser inhábil",
  plazoInstante: "Hasta las 23:59 del {fecha} (hora de Madrid)",
  plazoPrimerDia: "Primer día del cómputo: {fecha}",
  plazoExcluidos: "Días no computados",
  plazoSinExcluidos: "No se ha excluido ningún día.",
  plazoFinde: "Fin de semana",
  municipio_sintetico: "Municipio sintético {codigo}",
  municipio_ine: "Municipio INE {codigo}",
  error_solicitud_invalida: "Revise los datos de la consulta.",
  error_calendario_no_publicado: "No hay calendario publicado para {anio}: falta {faltan}.",
  error_servicio_no_disponible: "El servicio de calendarios no está disponible. Inténtelo más tarde.",
  error_autenticacion_requerida: "Identifíquese con su certificado para consultar los calendarios.",
  error_acceso_denegado: "Su perfil no permite consultar los calendarios.",
  error_respuesta: "La respuesta del servidor no es válida.",
});

const CLAVES = Object.freeze(Object.keys(MENSAJES_CALENDARIOS_ES));

export function crearTraductorCalendarios(catalogo = MENSAJES_CALENDARIOS_ES) {
  if (!catalogo || typeof catalogo !== "object"
    || CLAVES.some((clave) => typeof catalogo[clave] !== "string" || catalogo[clave] === "")) {
    throw new Error("catálogo i18n de Calendarios incompleto");
  }
  return (clave, variables = {}) => {
    if (!Object.hasOwn(catalogo, clave)) throw new Error(`clave i18n de Calendarios desconocida: ${clave}`);
    return catalogo[clave].replace(/\{([a-z_]+)\}/g, (_c, v) => String(variables[v] ?? ""));
  };
}

const LOCALE = "es-ES";
const ZONA = "Europe/Madrid";

/** Fecha civil AAAA-MM-DD en formato largo local, sin desplazamientos de zona. */
export function formatearFechaCivil(texto, estilo = "long") {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/u.exec(String(texto ?? ""));
  if (!m) throw new TypeError("fecha civil no válida");
  const instante = new Date(Date.UTC(Number(m[1]), Number(m[2]) - 1, Number(m[3]), 12));
  return new Intl.DateTimeFormat(LOCALE, { dateStyle: estilo, timeZone: "UTC" }).format(instante);
}

/** Instante RFC 3339 en hora de Madrid. */
export function formatearInstante(texto) {
  const instante = new Date(texto);
  if (Number.isNaN(instante.getTime())) throw new TypeError("instante no válido");
  return new Intl.DateTimeFormat(LOCALE, { dateStyle: "medium", timeStyle: "short", timeZone: ZONA }).format(instante);
}

export function formatearNumero(n) {
  return new Intl.NumberFormat(LOCALE).format(n);
}

export function nombreMes(mes) {
  return new Intl.DateTimeFormat(LOCALE, { month: "long", timeZone: "UTC" }).format(new Date(Date.UTC(2026, mes - 1, 1, 12)));
}
