/**
 * Vocabulario visible de los ocho hitos de RRHH. Las claves intermedias del
 * flujo se asignan a un hito para presentar la posición del expediente; esta
 * proyección no altera la definición gobernada ni las decisiones del servidor.
 */
const FASES = [
  ["solicitud", "Solicitud", "Request"],
  ["analisis_rrhh", "Análisis de RRHH", "HR review"],
  ["gestion_bolsa", "Gestión de bolsa", "Job pool management"],
  ["fiscalizacion", "Fiscalización", "Financial review"],
  ["obtencion_candidato", "Obtención del candidato", "Candidate selection"],
  ["nombramiento", "Nombramiento", "Appointment"],
  ["incorporacion", "Incorporación", "Joining"],
  ["seguimiento", "Seguimiento", "Follow-up"],
];

export const FASES_RRHH = Object.freeze(FASES.map(([clave, es, en], indice) =>
  Object.freeze({ clave, orden: indice + 1, es, en })));

const ALIAS_FASE = Object.freeze({
  solicitud_registrada: "solicitud",
  analisis: "analisis_rrhh",
  cobertura: "gestion_bolsa",
  asignacion: "gestion_bolsa",
  asignacion_unidad: "gestion_bolsa",
  informe: "gestion_bolsa",
  informe_juridico: "gestion_bolsa",
  subsanacion_unidad: "fiscalizacion",
  llamamiento: "obtencion_candidato",
  cierre: "seguimiento",
});

function claveNormalizada(valor) {
  if (typeof valor !== "string") return "";
  return valor.trim().toLocaleLowerCase("es-ES")
    .normalize("NFD").replace(/[\u0300-\u036f]/gu, "")
    .replace(/^contratacion_temporal\.fase\./u, "")
    .replace(/^fase[:_-]/u, "")
    .replace(/[\s-]+/gu, "_");
}

export function obtenerFaseRRHH(clave) {
  const normalizada = claveNormalizada(clave);
  const canonica = ALIAS_FASE[normalizada] || normalizada;
  return FASES_RRHH.find((fase) => fase.clave === canonica) || null;
}

export function nombreFaseRRHH(clave, idioma = "es") {
  const fase = obtenerFaseRRHH(clave);
  return fase ? fase[idioma === "en" ? "en" : "es"] : null;
}

export function formatearFaseRRHH(clave, idioma = "es") {
  const fase = obtenerFaseRRHH(clave);
  if (!fase) return null;
  return idioma === "en"
    ? `Phase ${fase.orden} of ${FASES_RRHH.length}: ${fase.en}`
    : `Fase ${fase.orden} de ${FASES_RRHH.length}: ${fase.es}`;
}

const ESTADOS_RRHH = Object.freeze({
  pendiente: Object.freeze({ es: "Pendiente", en: "Pending" }),
  en_curso: Object.freeze({ es: "En tramitación", en: "In progress" }),
  esperando_otro_departamento: Object.freeze({ es: "Esperando a otro departamento", en: "Awaiting another department" }),
  completado: Object.freeze({ es: "Completado", en: "Completed" }),
  incidencia: Object.freeze({ es: "Con incidencia", en: "Needs attention" }),
  cancelado: Object.freeze({ es: "Cancelado", en: "Cancelled" }),
  cerrado: Object.freeze({ es: "Cerrado", en: "Closed" }),
});

const ALIAS_ESTADO = Object.freeze({
  espera: "esperando_otro_departamento",
  esperando_a_otro_departamento: "esperando_otro_departamento",
});

export function nombreEstadoRRHH(clave, idioma = "es") {
  const normalizada = claveNormalizada(clave);
  const estado = ESTADOS_RRHH[ALIAS_ESTADO[normalizada] || normalizada];
  return estado ? estado[idioma === "en" ? "en" : "es"] : null;
}
