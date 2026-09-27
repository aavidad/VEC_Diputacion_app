export const MENSAJES_REINCORPORACION_RRHH_ES = Object.freeze({
  rrhh_reincorporacion_titulo: "Registrar la reincorporación del titular",
  rrhh_reincorporacion_subtitulo: "Actuación de RRHH en el expediente de contratación temporal",
  rrhh_reincorporacion_pasos: "Secuencia de seguimiento",
  rrhh_reincorporacion_paso_cese: "Cese",
  rrhh_reincorporacion_paso_reincorporacion: "Reincorporación",
  rrhh_reincorporacion_paso_cierre: "Cierre",
  rrhh_reincorporacion_ayuda_boton: "Ayuda sobre la reincorporación",
  rrhh_reincorporacion_ayuda: "Esta actuación requiere un cese registrado por fin de sustitución, con la misma fecha efectiva, la misma referencia y huella documental, y la relación original acreditada. Se registra después del cese y antes de cerrar el expediente. La operación conserva un recibo en Contratación temporal; Bolsa recibe el evento y aplica su propia regla de disponibilidad cuando corresponda. La referencia y la huella del documento no sustituyen su custodia.",
  rrhh_reincorporacion_datos: "Datos de la reincorporación",
  rrhh_reincorporacion_expediente: "Expediente",
  rrhh_reincorporacion_version: "Versión observada",
  rrhh_reincorporacion_relacion: "Referencia de la relación del titular",
  rrhh_reincorporacion_fecha: "Fecha efectiva de reincorporación",
  rrhh_reincorporacion_documento: "Referencia del documento acreditativo",
  rrhh_reincorporacion_huella: "Huella SHA-256 del documento",
  rrhh_reincorporacion_registrar: "Registrar reincorporación",
  rrhh_reincorporacion_reintentar: "Comprobar con la misma operación",
  rrhh_reincorporacion_confirmar: "Se registrará la reincorporación del titular en este expediente. ¿Confirma los datos?",
  rrhh_reincorporacion_lista: "Preparada para registrar.",
  rrhh_reincorporacion_validacion: "Revise las referencias, la fecha y la huella del documento.",
  rrhh_reincorporacion_cancelada: "La operación no se ha enviado.",
  rrhh_reincorporacion_enviando: "Registrando la reincorporación. Espere la respuesta.",
  rrhh_reincorporacion_denegada: "No tiene permiso para registrar esta reincorporación. Se han retirado los datos del formulario.",
  rrhh_reincorporacion_conflicto: "El expediente cambió o los datos entran en conflicto. Actualice el detalle antes de continuar.",
  rrhh_reincorporacion_rechazada: "La operación se rechazó. Revise los datos antes de enviarlos otra vez.",
  rrhh_reincorporacion_incierta: "No se puede confirmar el resultado. Compruebe el registro solo con la misma operación y los mismos datos.",
  rrhh_reincorporacion_incierta_clave: "Clave de la operación original",
  rrhh_reincorporacion_confirmada: "Reincorporación registrada en Contratación temporal.",
  rrhh_reincorporacion_recibo: "Recibo de la actuación",
  rrhh_reincorporacion_recibo_ref: "Justificante",
  rrhh_reincorporacion_cese_recibo_ref: "Cese asociado",
  rrhh_reincorporacion_evento_ref: "Evento",
  rrhh_reincorporacion_registrada_en: "Registrada el",
  rrhh_reincorporacion_bolsa: "Reflejo en Bolsa",
  rrhh_reincorporacion_bolsa_pendiente: "Pendiente de confirmación por Bolsa",
});

export function crearTraductorReincorporacionRRHH(mensajes = MENSAJES_REINCORPORACION_RRHH_ES) {
  return (clave) => {
    if (!Object.hasOwn(mensajes, clave) || typeof mensajes[clave] !== "string") {
      throw new Error(`falta la traducción ${clave}`);
    }
    return mensajes[clave];
  };
}
