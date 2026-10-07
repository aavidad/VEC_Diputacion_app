/** Equivalencias de fase para presentación; no cargan textos ni conceden acceso. */
export const FASES_RRHH = Object.freeze([
  "solicitud", "analisis_rrhh", "gestion_bolsa", "fiscalizacion",
  "obtencion_candidato", "nombramiento", "incorporacion", "seguimiento",
]);

export const FASE_RRHH_DE_ORIGEN = Object.freeze({
  solicitud: "solicitud", solicitud_registrada: "solicitud",
  analisis: "analisis_rrhh", analisis_rrhh: "analisis_rrhh",
  cobertura: "gestion_bolsa", asignacion: "gestion_bolsa", asignacion_unidad: "gestion_bolsa",
  informe: "gestion_bolsa", informe_juridico: "gestion_bolsa", gestion_bolsa: "gestion_bolsa",
  fiscalizacion: "fiscalizacion", subsanacion_unidad: "fiscalizacion",
  llamamiento: "obtencion_candidato", obtencion_candidato: "obtencion_candidato",
  nombramiento: "nombramiento", incorporacion: "incorporacion",
  seguimiento: "seguimiento", cierre: "seguimiento",
});

export function faseRRHH(claveOrigen) {
  const clave = FASE_RRHH_DE_ORIGEN[claveOrigen];
  if (!clave) return null;
  return Object.freeze({ clave, orden: FASES_RRHH.indexOf(clave) + 1, total: FASES_RRHH.length });
}
