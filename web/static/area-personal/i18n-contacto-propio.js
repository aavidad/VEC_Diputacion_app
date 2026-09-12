export const textosContactoPropio = Object.freeze({
  titulo: "Datos de contacto",
  subtitulo: "El correo se guarda de forma independiente",
  ayuda: "Este formulario guarda únicamente el correo de contacto. No guarda teléfono ni domicilio.",
  etiquetaCorreo: "Correo de contacto",
  guardar: "Guardar correo",
  guardarTelefonoDomicilio: "Revisar y guardar teléfono y domicilio",
  preparando: "Guardando correo…",
  sinAutorizacion: "El correo no se puede actualizar porque el servicio no ha aportado permiso expreso y versión vigente.",
  correcto: "Correo de contacto guardado. Referencia de recibo: {recibo}.",
  correctoAnterior: "Correo de contacto guardado. Referencia de recibo: {recibo}.",
  errorEntrada: "Revise el correo de contacto antes de enviarlo.",
  errorPermiso: "No dispone de permiso para actualizar el correo de contacto.",
  errorServicio: "No se pudo confirmar el guardado del correo de contacto. No se ha repetido la operación.",
  consultarRecibo: "Consultar recibo",
  consultando: "Consultando recibo…",
  consultaNoDisponible: "La consulta del recibo no está disponible para este intento.",
  consultaSinConfirmacion: "No se ha podido recuperar el recibo. El guardado anterior podría haberse completado; esta consulta no lo confirma ni lo descarta.",
  reciboConsultado: "Recibo de la versión {version}: {recibo}. Esta consulta no confirma que el correo del último intento coincida con el guardado. El formulario no se ha actualizado.",
});

export function textoContactoPropio(clave, valores = {}) {
  const plantilla = textosContactoPropio[clave];
  if (typeof plantilla !== "string") throw new TypeError("La clave de traducción del correo no existe.");
  return plantilla.replace(/\{([a-z_]+)\}/gu, (_, nombre) => String(valores[nombre] ?? ""));
}
