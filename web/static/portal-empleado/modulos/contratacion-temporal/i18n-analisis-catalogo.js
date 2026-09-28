/** Textos del análisis que dependen del catálogo de reglas: duración máxima y urgencia. */
export const MENSAJES_ANALISIS_CATALOGO_ES = Object.freeze({
  analisis_aviso_duracion_maxima:
    "El periodo supera la duración máxima de esta modalidad ({duracion}). Revíselo antes de registrar.",
  analisis_error_duracion_maxima: "El periodo supera la duración máxima de esta modalidad.",
  analisis_duracion_meses_uno: "{cantidad} mes",
  analisis_duracion_meses_otros: "{cantidad} meses",
  analisis_duracion_anios_uno: "{cantidad} año",
  analisis_duracion_anios_otros: "{cantidad} años",
  analisis_duracion_dias_naturales_uno: "{cantidad} día natural",
  analisis_duracion_dias_naturales_otros: "{cantidad} días naturales",
  analisis_urgencia_leyenda: "Tramitación urgente",
  analisis_urgencia_marcar: "Declarar urgente este expediente",
  analisis_urgencia_motivo: "Motivo de la urgencia",
  analisis_urgencia_declarada: "Urgente",
  analisis_error_urgencia_motivo: "Indique el motivo de la urgencia (hasta 1.000 caracteres, sin caracteres no admitidos).",
});

export const MENSAJES_ANALISIS_CATALOGO_EN = Object.freeze({
  analisis_aviso_duracion_maxima:
    "The period exceeds the maximum duration for this type ({duracion}). Please review it before recording.",
  analisis_error_duracion_maxima: "The period exceeds the maximum duration for this type.",
  analisis_duracion_meses_uno: "{cantidad} month",
  analisis_duracion_meses_otros: "{cantidad} months",
  analisis_duracion_anios_uno: "{cantidad} year",
  analisis_duracion_anios_otros: "{cantidad} years",
  analisis_duracion_dias_naturales_uno: "{cantidad} calendar day",
  analisis_duracion_dias_naturales_otros: "{cantidad} calendar days",
  analisis_urgencia_leyenda: "Urgent processing",
  analisis_urgencia_marcar: "Mark this case for urgent processing",
  analisis_urgencia_motivo: "Reason for urgency",
  analisis_urgencia_declarada: "Urgent",
  analisis_error_urgencia_motivo: "Enter the reason for urgency (up to 1,000 characters, without unsupported characters).",
});
