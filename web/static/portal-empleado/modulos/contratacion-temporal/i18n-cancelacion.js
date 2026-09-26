/** Textos de la cancelación del expediente antes de la fiscalización. */

export const MENSAJES_CANCELACION = Object.freeze({
  titulo: "Cancelación del expediente",
  abrir: "Cancelar expediente",
  ayuda_boton: "Ayuda sobre la cancelación del expediente",
  ayuda: "Mientras el expediente no se haya fiscalizado, Recursos Humanos puede cancelarlo eligiendo un motivo del catálogo y, si quiere, una observación. Las fases en las que se admite y los motivos los fija el catálogo de reglas. El expediente cancelado queda terminado, conserva toda su historia y no admite más actuaciones. Antes de la fiscalización todavía no hay llamamiento, así que Bolsa no se ve afectada.",
  cerrar_formulario: "Volver sin cancelar",
  motivo: "Motivo de la cancelación",
  observaciones: "Observaciones (opcional)",
  enviar: "Cancelar el expediente",
  confirmar_titulo: "Cancelar el expediente",
  confirmar: "El expediente quedará cancelado y no admitirá más actuaciones. ¿Confirma la cancelación?",
  cargando: "Consultando si el expediente se puede cancelar.",
  no_disponible: "No se ha podido consultar la cancelación de este expediente.",
  reintentar: "Reintentar",
  estado_cancelado: "Cancelado",
  cancelado_resumen: "Expediente cancelado por {quien}: {motivo}.",
  cancelado_fecha: "Fecha de la cancelación",
  cancelado_fase: "Fase en la que se canceló",
  cancelado_observaciones: "Observaciones",
  quien_rrhh: "Recursos Humanos",
  quien_centro: "el centro solicitante",
  justificante_registrado: "Justificante de la cancelación",
  justificante_copiar: "Copiar referencia",
  justificante_copiado: "Referencia copiada",
  enviando: "Cancelando; espere la respuesta.",
  recibo: "Expediente cancelado.",
  error_contenido_no_valido: "Revise los datos: elija un motivo y revise las observaciones.",
  error_acceso_denegado: "No tiene permiso para cancelar este expediente.",
  error_version_en_conflicto: "El expediente ha cambiado. Actualice el detalle antes de continuar.",
  error_clave_reutilizada: "Esta cancelación ya se registró con otros datos.",
  error_fase_no_admitida: "El expediente ya no está en una fase en la que se pueda cancelar.",
  error_tras_fiscalizacion: "El expediente ya pasó por fiscalización y no se puede cancelar.",
  error_cancelacion_existente: "El expediente ya estaba cancelado.",
  error_indeterminado: "No se ha podido confirmar el resultado. Reintente: se usará la misma operación y no se duplicará.",
  error_general: "No se ha podido cancelar el expediente. Inténtelo de nuevo más tarde.",
  fase_solicitud: "Solicitud",
  fase_asignacion_unidad: "Asignación de unidad",
  fase_informe_juridico: "Informe jurídico",
});

export function crearTraductorCancelacion(mensajes = {}) {
  return (clave, valores = {}) => {
    const plantilla = typeof mensajes?.[`cancelacion.${clave}`] === "string"
      ? mensajes[`cancelacion.${clave}`]
      : MENSAJES_CANCELACION[clave] ?? clave;
    return plantilla.replace(/\{([a-z_]+)\}/gu, (_, nombre) => (Object.hasOwn(valores, nombre) ? String(valores[nombre]) : `{${nombre}}`));
  };
}
