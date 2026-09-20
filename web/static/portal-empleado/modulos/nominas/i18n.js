export const MENSAJES_NOMINAS_ES = Object.freeze({
  titulo: "Nóminas y retribuciones",
  sobrelinea: "Portal del Empleado · gestión económica personal",
  aviso: "Presentación visual · ejemplo ficticio, no nómina emitida",
  descripcion: "Consulta orientativa de documentos retributivos, certificados e incidencias personales.",
  falta: "Qué falta para activar este recorrido: fuente de nómina, documentos originales, identidad y autorización, trazabilidad y recibos.",
  periodo: "Periodo", filtrar: "Aplicar filtro", todos: "Todos los periodos",
  bruta: "Devengo mostrado", liquido: "Resultado mostrado", documentos: "Documentos de ejemplo", incidencias: "Incidencias abiertas",
  listado: "Nóminas del periodo", detalle: "Detalle del documento", certificados: "Certificados fiscales", incidencias_titulo: "Incidencias retributivas", aclaracion: "Solicitar aclaración",
  concepto: "Concepto", tipo: "Tipo", importe: "Importe de ejemplo", estado: "Estado", acciones: "Acciones", fecha: "Fecha", referencia: "Referencia",
  seleccionar: "Ver detalle", sin_documentos: "No hay documentos de ejemplo para el filtro seleccionado.",
  devengos: "Devengos de ejemplo", deducciones: "Deducciones de ejemplo", resultado: "Resultado orientativo", estado_visual: "Visual · pendiente de backend",
  certificado_irpf: "Certificado fiscal anual", certificado_retribuciones: "Certificado de retribuciones", descargar: "Descargar", registrar: "Registrar incidencia", enviar: "Enviar solicitud",
  bloqueado: "Pendiente de conexión con backend; esta acción no registra, descarga ni comunica nada.",
  incidencia_descripcion: "Describa la aclaración sin incluir datos innecesarios.", asunto: "Asunto", detalle_incidencia: "Detalle", cancelar: "Cancelar", preparado: "Formulario preparado localmente; no se ha enviado ninguna solicitud.",
  persona: "Antonio López Fernández", estado_entrega: "Estado de entrega: visual_pendiente_backend",
});

export function crearTraductorNominas(mensajes = MENSAJES_NOMINAS_ES) {
  return (clave) => mensajes[clave] || clave;
}
