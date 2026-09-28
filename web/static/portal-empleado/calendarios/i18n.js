import { IDIOMA_ACTUAL, LOCALIZACION_ACTUAL } from "../../comun/idioma.js";

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
  kpiHabil: "Día hábil",
  kpiLaborables: "Días laborables del centro",
  kpiLaborable: "Día laborable del centro",
  kpiFestivos: "Festivos oficiales",
  kpiFestivo: "Festivo oficial",
  kpiCierres: "Cierres del centro",
  kpiCierre: "Cierre del centro",
  calendarioTitulo: "Calendario {anio}",
  leyenda: "Leyenda",
  leyendaNacional: "Festivo nacional",
  leyendaAutonomico: "Festivo de Andalucía",
  leyendaLocal: "Festivo local",
  leyendaCentro: "Cierre del centro",
  leyendaFinde: "Sábado o domingo",
  semanaCorta: "L,M,X,J,V,S,D",
  semanaLarga: "lunes,martes,miércoles,jueves,viernes,sábado,domingo",
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
  error_plazo_no_determinado: "Con los calendarios publicados no se puede fijar el vencimiento de este plazo.",
  error_autenticacion_requerida: "Identifíquese con su certificado para consultar los calendarios.",
  error_acceso_denegado: "Su perfil no permite consultar los calendarios.",
  error_respuesta: "La respuesta del servidor no es válida.",
});

export const MENSAJES_CALENDARIOS_EN = Object.freeze({
  documentTitle: "Work calendars · Employee Portal",
  miga: "Employee Portal → Calendars",
  titulo: "Work calendars",
  volver: "Back to the portal",
  ayudaAbrir: "Help",
  ayudaTitulo: "Work calendars",
  ayudaCerrar: "Close",
  ayudaCentro: "Each workplace calendar combines national, Andalusian and local public holidays with closures specific to the workplace.",
  ayudaHabil: "Saturdays, Sundays and public holidays do not count towards administrative deadlines. An internal workplace closure does not change that calculation.",
  ayudaConocido: "‘Known on’ shows the calendar as it stood on that date, before later corrections.",
  ayudaPlazo: "The period starts on the day after notification. If its last day is not a working day, it ends on the next working day.",
  filtros: "Calendar selection",
  centro: "Workplace",
  anio: "Year",
  conocidoEn: "Known on",
  consultar: "View",
  cargando: "Loading calendar…",
  cargandoCentros: "Loading workplaces…",
  sinCentros: "No workplace calendars have been published for this year.",
  kpiHabiles: "Administrative working days",
  kpiHabil: "Administrative working day",
  kpiLaborables: "Workplace working days",
  kpiLaborable: "Workplace working day",
  kpiFestivos: "Official holidays",
  kpiFestivo: "Official holiday",
  kpiCierres: "Workplace closures",
  kpiCierre: "Workplace closure",
  calendarioTitulo: "Calendar {anio}",
  leyenda: "Key",
  leyendaNacional: "National holiday",
  leyendaAutonomico: "Andalusian holiday",
  leyendaLocal: "Local holiday",
  leyendaCentro: "Workplace closure",
  leyendaFinde: "Saturday or Sunday",
  semanaCorta: "Mo,Tu,We,Th,Fr,Sa,Su",
  semanaLarga: "Monday,Tuesday,Wednesday,Thursday,Friday,Saturday,Sunday",
  diaEtiqueta: "{fecha}: {motivos}",
  diaHabil: "working day",
  diaInhabil: "non-working day",
  festivosTitulo: "Holidays and closures",
  colFecha: "Date",
  colDenominacion: "Name",
  colAmbito: "Scope",
  colProcedencia: "Source",
  ambito_nacional: "National",
  ambito_autonomico: "Andalusia",
  ambito_local: "Local",
  ambito_centro: "Workplace",
  sintetico: "Synthetic",
  oficial: "Official",
  versionesTitulo: "Calendar sources",
  colCalendario: "Calendar",
  colVersion: "Version",
  colNorma: "Regulation or agreement",
  colConocidoDesde: "Known since",
  plazoTitulo: "Deadline calculation",
  plazoNotificacion: "Notification date",
  plazoUnidad: "Unit",
  plazoCantidad: "Quantity",
  plazoResidencia: "Municipality of residence",
  plazoMismaSede: "Same as the workplace",
  plazoCalcular: "Calculate",
  unidad_dias_habiles: "Working days",
  unidad_dias_naturales: "Calendar days",
  unidad_meses: "Months",
  unidad_anios: "Years",
  plazoVence: "Due on {fecha}",
  plazoProrrogado: "Extended from {fecha} because that day is non-working",
  plazoInstante: "Until 23:59 on {fecha} (Madrid time)",
  plazoPrimerDia: "First day of the period: {fecha}",
  plazoExcluidos: "Days not counted",
  plazoSinExcluidos: "No days were excluded.",
  plazoFinde: "Weekend",
  municipio_sintetico: "Synthetic municipality {codigo}",
  municipio_ine: "INE municipality {codigo}",
  error_solicitud_invalida: "Check the request details.",
  error_calendario_no_publicado: "No calendar published for {anio}: missing {faltan}.",
  error_servicio_no_disponible: "The calendar service is unavailable. Please try again later.",
  error_plazo_no_determinado: "The published calendars cannot establish this deadline.",
  error_autenticacion_requerida: "Identify yourself with your certificate to view the calendars.",
  error_acceso_denegado: "Your profile does not allow you to view the calendars.",
  error_respuesta: "The server response is invalid.",
});

const CLAVES = Object.freeze(Object.keys(MENSAJES_CALENDARIOS_ES));
const MARCADORES = /\{([a-z_]+)\}/gu;
const catalogoActual = () => IDIOMA_ACTUAL === "en" ? MENSAJES_CALENDARIOS_EN : MENSAJES_CALENDARIOS_ES;

export function crearTraductorCalendarios(catalogo = catalogoActual()) {
  if (!catalogo || typeof catalogo !== "object"
    || Object.keys(catalogo).length !== CLAVES.length
    || CLAVES.some((clave) => typeof catalogo[clave] !== "string" || catalogo[clave] === ""
      || [...catalogo[clave].matchAll(MARCADORES)].map((m) => m[1]).sort().join(",")
        !== [...MENSAJES_CALENDARIOS_ES[clave].matchAll(MARCADORES)].map((m) => m[1]).sort().join(","))) {
    throw new Error("catálogo i18n de Calendarios incompleto");
  }
  return (clave, variables = {}) => {
    if (!Object.hasOwn(catalogo, clave)) throw new Error(`clave i18n de Calendarios desconocida: ${clave}`);
    return catalogo[clave].replace(/\{([a-z_]+)\}/g, (_c, v) => String(variables[v] ?? ""));
  };
}

const ZONA = "Europe/Madrid";

/** Fecha civil AAAA-MM-DD en formato largo local, sin desplazamientos de zona. */
export function formatearFechaCivil(texto, estilo = "long", localizacion = LOCALIZACION_ACTUAL) {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/u.exec(String(texto ?? ""));
  if (!m) throw new TypeError("fecha civil no válida");
  const instante = new Date(Date.UTC(Number(m[1]), Number(m[2]) - 1, Number(m[3]), 12));
  return new Intl.DateTimeFormat(localizacion, { dateStyle: estilo, timeZone: "UTC" }).format(instante);
}

/** Instante RFC 3339 en hora de Madrid. */
export function formatearInstante(texto, localizacion = LOCALIZACION_ACTUAL) {
  const instante = new Date(texto);
  if (Number.isNaN(instante.getTime())) throw new TypeError("instante no válido");
  return new Intl.DateTimeFormat(localizacion, { dateStyle: "medium", timeStyle: "short", timeZone: ZONA }).format(instante);
}

export function formatearNumero(n, localizacion = LOCALIZACION_ACTUAL) {
  return new Intl.NumberFormat(localizacion).format(n);
}

export function nombreMes(mes, localizacion = LOCALIZACION_ACTUAL) {
  return new Intl.DateTimeFormat(localizacion, { month: "long", timeZone: "UTC" }).format(new Date(Date.UTC(2026, mes - 1, 1, 12)));
}
