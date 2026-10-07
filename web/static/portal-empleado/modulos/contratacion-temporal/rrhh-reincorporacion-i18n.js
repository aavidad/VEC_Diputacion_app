import { IDIOMA_ACTUAL, IDIOMA_POR_DEFECTO } from "../../../comun/idioma.js";

export const MENSAJES_REINCORPORACION_RRHH_ES = Object.freeze({
  rrhh_reincorporacion_titulo: "Registrar la reincorporación del titular",
  rrhh_reincorporacion_subtitulo: "Actuación de RRHH en el expediente de petición de personal temporal",
  rrhh_reincorporacion_pasos: "Secuencia de seguimiento",
  rrhh_reincorporacion_paso_cese: "Cese",
  rrhh_reincorporacion_paso_reincorporacion: "Reincorporación",
  rrhh_reincorporacion_paso_cierre: "Cierre",
  rrhh_reincorporacion_ayuda_boton: "Ayuda sobre la reincorporación",
  rrhh_reincorporacion_ayuda: "Esta actuación requiere un cese registrado por fin de sustitución. La relación afectada identifica la del sustituto cesado, no una relación laboral del titular. Se registra después del cese y antes de cerrar el expediente. La operación conserva un recibo en Peticiones de personal temporal; Bolsa recibe el evento y aplica su propia regla de disponibilidad cuando corresponda. La referencia y la huella del documento no sustituyen su custodia.",
  rrhh_reincorporacion_datos: "Datos de la reincorporación",
  rrhh_reincorporacion_expediente: "Expediente",
  rrhh_reincorporacion_version: "Versión observada",
  rrhh_reincorporacion_relacion: "Relación afectada del sustituto cesado",
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
  rrhh_reincorporacion_confirmada: "Reincorporación registrada en Peticiones de personal temporal.",
  rrhh_reincorporacion_recibo: "Recibo de la actuación",
  rrhh_reincorporacion_recibo_ref: "Justificante",
  rrhh_reincorporacion_cese_recibo_ref: "Cese asociado",
  rrhh_reincorporacion_evento_ref: "Evento",
  rrhh_reincorporacion_registrada_en: "Registrada el",
  rrhh_reincorporacion_bolsa: "Reflejo en Bolsa",
  rrhh_reincorporacion_bolsa_pendiente: "Pendiente de confirmación por Bolsa",
});

export const MENSAJES_REINCORPORACION_RRHH_EN = Object.freeze({
  rrhh_reincorporacion_titulo: "Record the substantive postholder’s return",
  rrhh_reincorporacion_subtitulo: "Human Resources action in the temporary staff requests case",
  rrhh_reincorporacion_pasos: "Follow-up sequence",
  rrhh_reincorporacion_paso_cese: "End of service",
  rrhh_reincorporacion_paso_reincorporacion: "Return to post",
  rrhh_reincorporacion_paso_cierre: "Closure",
  rrhh_reincorporacion_ayuda_boton: "Help with return to post",
  rrhh_reincorporacion_ayuda: "This action requires a recorded end of service due to the end of the substitution. The affected employment relationship belongs to the substitute whose service ended, not to the substantive postholder. This is recorded after the end of service and before the case is closed. The operation retains a receipt in Temporary Staff Requests; the recruitment pool receives the event and applies its own availability rule where applicable. The document reference and digest do not replace custody of the document.",
  rrhh_reincorporacion_datos: "Return to post details",
  rrhh_reincorporacion_expediente: "Case",
  rrhh_reincorporacion_version: "Version viewed",
  rrhh_reincorporacion_relacion: "Affected relationship of the substitute whose service ended",
  rrhh_reincorporacion_fecha: "Effective date of return",
  rrhh_reincorporacion_documento: "Supporting document reference",
  rrhh_reincorporacion_huella: "Document SHA-256 digest",
  rrhh_reincorporacion_registrar: "Record return to post",
  rrhh_reincorporacion_reintentar: "Check using the same operation",
  rrhh_reincorporacion_confirmar: "The substantive postholder’s return will be recorded in this case. Confirm the details?",
  rrhh_reincorporacion_lista: "Ready to record.",
  rrhh_reincorporacion_validacion: "Check the references, date and document digest.",
  rrhh_reincorporacion_cancelada: "The operation has not been submitted.",
  rrhh_reincorporacion_enviando: "Recording the return to post. Wait for a response.",
  rrhh_reincorporacion_denegada: "You do not have permission to record this return to post. The form data has been cleared.",
  rrhh_reincorporacion_conflicto: "The case has changed or the data conflicts. Refresh the details before proceeding.",
  rrhh_reincorporacion_rechazada: "The operation was rejected. Check the details before submitting again.",
  rrhh_reincorporacion_incierta: "The outcome cannot be confirmed. Check the record using only the same operation and data.",
  rrhh_reincorporacion_incierta_clave: "Original operation key",
  rrhh_reincorporacion_confirmada: "Return to post recorded in Temporary Staff Requests.",
  rrhh_reincorporacion_recibo: "Action receipt",
  rrhh_reincorporacion_recibo_ref: "Receipt",
  rrhh_reincorporacion_cese_recibo_ref: "Associated end of service",
  rrhh_reincorporacion_evento_ref: "Event",
  rrhh_reincorporacion_registrada_en: "Recorded on",
  rrhh_reincorporacion_bolsa: "Recruitment pool update",
  rrhh_reincorporacion_bolsa_pendiente: "Awaiting confirmation from the recruitment pool",
});

export function crearTraductorReincorporacionRRHH(mensajes = IDIOMA_ACTUAL === IDIOMA_POR_DEFECTO
  ? MENSAJES_REINCORPORACION_RRHH_ES : MENSAJES_REINCORPORACION_RRHH_EN) {
  return (clave) => {
    if (!Object.hasOwn(mensajes, clave) || typeof mensajes[clave] !== "string") {
      throw new Error(`falta la traducción ${clave}`);
    }
    return mensajes[clave];
  };
}
