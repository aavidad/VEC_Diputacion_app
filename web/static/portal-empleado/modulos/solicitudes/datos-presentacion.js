/** Datos estrictamente sintéticos para el recorrido visual de RRHH. */
export const DATOS_SOLICITUDES_PRESENTACION = Object.freeze({
  persona: "Antonio López Fernández",
  actualizado: "20 de septiembre de 2026 · 09:30",
  tramites: Object.freeze([
    Object.freeze({ id: "SOL-2026-00184", tipo: "Reconocimiento de servicios previos", categoria: "Personal", fecha: "17/09/2026", estado: "En revisión", unidad: "Servicio de Personal", paso: "Comprobación documental", descripcion: "Solicitud para incorporar servicios prestados en otra administración." }),
    Object.freeze({ id: "SOL-2026-00167", tipo: "Permiso por asistencia a consulta médica", categoria: "Cronos", fecha: "12/09/2026", estado: "Pendiente de subsanación", unidad: "Jefatura de Área", paso: "Aportar justificante", descripcion: "Falta el justificante de asistencia indicado por la unidad tramitadora." }),
    Object.freeze({ id: "SOL-2026-00121", tipo: "Solicitud de ayuda de acción social", categoria: "Retribuciones", fecha: "03/09/2026", estado: "Registrada", unidad: "Sección de Nóminas", paso: "Entrada registrada", descripcion: "Solicitud incluida en la convocatoria anual de ayudas sociales." }),
    Object.freeze({ id: "SOL-2026-00098", tipo: "Certificado de servicios prestados", categoria: "Certificados", fecha: "28/08/2026", estado: "Finalizada", unidad: "Servicio de Personal", paso: "Documento disponible", descripcion: "Certificado pendiente de conexión con el repositorio documental." }),
  ]),
  catalogo: Object.freeze([
    Object.freeze({ id: "servicios", titulo: "Servicios previos", texto: "Solicita el reconocimiento de servicios prestados en otra administración.", etiqueta: "Personal" }),
    Object.freeze({ id: "permiso", titulo: "Permisos y ausencias", texto: "Prepara una solicitud de permiso, ausencia o justificación.", etiqueta: "Cronos" }),
    Object.freeze({ id: "accion-social", titulo: "Acción social", texto: "Consulta y prepara solicitudes de las convocatorias de ayudas.", etiqueta: "Retribuciones" }),
    Object.freeze({ id: "compatibilidad", titulo: "Compatibilidad", texto: "Inicia la solicitud de compatibilidad para segunda actividad.", etiqueta: "Personal" }),
  ]),
  certificados: Object.freeze([
    Object.freeze({ tipo: "Servicios prestados", alcance: "Relación de servicios y periodos", estado: "Disponible tras conexión", referencia: "CERT-2026-041" }),
    Object.freeze({ tipo: "Situación administrativa", alcance: "Situación vigente a fecha de emisión", estado: "Pendiente de validación", referencia: "CERT-2026-039" }),
    Object.freeze({ tipo: "Retenciones e ingresos", alcance: "Ejercicio 2025", estado: "Disponible tras conexión", referencia: "CERT-2026-031" }),
  ]),
});
