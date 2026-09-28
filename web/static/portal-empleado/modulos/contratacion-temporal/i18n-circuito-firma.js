/** Textos castellanos del circuito de firma de los borradores del expediente. */

export const MENSAJES_CIRCUITO_FIRMA_ES = Object.freeze({
  circuito_firma_titulo: "Circuito de firma",
  circuito_firma_ejemplo: "Circuito de ejemplo",
  circuito_firma_pasos: "Pasos de firma de {documento}",
  circuito_firma_estado_pendiente_firma: "Pendiente de firma por {cargo}",
  circuito_firma_estado_en_espera: "En espera del paso anterior",
  circuito_firma_estado_firmado: "Firmado por {cargo}",
  circuito_firma_estado_devuelto: "Devuelto por {cargo}",
  circuito_firma_accion_firma: "Firma",
  circuito_firma_accion_visto_bueno: "Visto bueno",
  circuito_firma_habilita_siguiente_paso: "Da paso al siguiente firmante",
  circuito_firma_habilita_remision_intervencion: "Permite remitir a Intervención",
  circuito_firma_habilita_envio_notificacion: "Permite enviar la notificación",
  circuito_firma_habilita_envio_comunicacion: "Permite enviar la comunicación",
  circuito_firma_habilita_cierre_circuito: "Cierra el circuito",
  circuito_firma_devolucion_vuelve_a_redaccion: "Si se devuelve, vuelve a redacción",
  circuito_firma_devolucion_vuelve_paso_anterior: "Si se devuelve, vuelve al paso anterior",
  circuito_firma_sustitucion_suplente_designado: "Admite suplente designado",
  circuito_firma_sustitucion_no_admitida: "Sin sustitución",
  circuito_firma_sin_eficacia: "Firma de prueba, sin eficacia administrativa",
  circuito_firma_motivo: "Motivo: {motivo}",
  circuito_firma_firmar: "Firmar",
  circuito_firma_devolver: "Devolver",
  circuito_firma_motivo_etiqueta: "Motivo de la devolución",
  circuito_firma_confirmar_devolucion: "Registrar la devolución",
  circuito_firma_cancelar: "Cancelar",
  circuito_firma_motivo_invalido: "Indique un motivo de entre 3 y 500 caracteres.",
  circuito_firma_preparando: "Preparando el borrador para firmar…",
  circuito_firma_abriendo_autofirma: "Abriendo AutoFirma: firme el documento en la ventana de AutoFirma.",
  circuito_firma_registrando: "Verificando y registrando…",
  circuito_firma_registrada: "Firma verificada y registrada con el recibo {recibo}. No tiene eficacia administrativa hasta el portafirmas corporativo.",
  circuito_firma_devuelta: "Devolución registrada con el recibo {recibo}.",
  circuito_firma_error_verificacion: "La verificación de firmas no está activada en este servidor, así que la firma no se ha registrado.",
  circuito_firma_error_no_verificada: "El validador no ha podido acreditar la firma ({motivo}); no se ha registrado.",
  circuito_firma_error_autofirma: "No se ha podido conectar con AutoFirma. Compruebe que está instalada y vuelva a intentarlo.",
  circuito_firma_error_cancelada: "Se ha cancelado la firma en AutoFirma.",
  circuito_firma_error_fallida: "AutoFirma no ha podido firmar el documento.",
  circuito_firma_error_cambiado: "El expediente o el circuito han cambiado. Vuelva a cargar el expediente.",
  circuito_firma_error_cadena: "El borrador no coincide con el que firmó el paso anterior; hace falta devolverlo a redacción.",
  circuito_firma_error_denegado: "No tiene permiso para firmar este documento.",
  circuito_firma_error_generico: "No se ha podido completar la operación. Vuelva a intentarlo.",
});

export const MENSAJES_CIRCUITO_FIRMA_EN = Object.freeze({
  circuito_firma_titulo: "Signing process",
  circuito_firma_ejemplo: "Example signing process",
  circuito_firma_pasos: "Signing steps for {documento}",
  circuito_firma_estado_pendiente_firma: "Awaiting signature by {cargo}",
  circuito_firma_estado_en_espera: "Waiting for the previous step",
  circuito_firma_estado_firmado: "Signed by {cargo}",
  circuito_firma_estado_devuelto: "Returned by {cargo}",
  circuito_firma_accion_firma: "Signature",
  circuito_firma_accion_visto_bueno: "Approval",
  circuito_firma_habilita_siguiente_paso: "Allows the next signatory to proceed",
  circuito_firma_habilita_remision_intervencion: "Allows referral to the Financial Control Office",
  circuito_firma_habilita_envio_notificacion: "Allows the notification to be sent",
  circuito_firma_habilita_envio_comunicacion: "Allows the communication to be sent",
  circuito_firma_habilita_cierre_circuito: "Completes the signing process",
  circuito_firma_devolucion_vuelve_a_redaccion: "If returned, it goes back for redrafting",
  circuito_firma_devolucion_vuelve_paso_anterior: "If returned, it goes back to the previous step",
  circuito_firma_sustitucion_suplente_designado: "A designated deputy is permitted",
  circuito_firma_sustitucion_no_admitida: "No substitution permitted",
  circuito_firma_sin_eficacia: "Test signature with no administrative effect",
  circuito_firma_motivo: "Reason: {motivo}",
  circuito_firma_firmar: "Sign",
  circuito_firma_devolver: "Return",
  circuito_firma_motivo_etiqueta: "Reason for return",
  circuito_firma_confirmar_devolucion: "Record the return",
  circuito_firma_cancelar: "Cancel",
  circuito_firma_motivo_invalido: "Enter a reason between 3 and 500 characters long.",
  circuito_firma_preparando: "Preparing the draft for signature…",
  circuito_firma_abriendo_autofirma: "Opening AutoFirma: sign the document in the AutoFirma window.",
  circuito_firma_registrando: "Verifying and recording…",
  circuito_firma_registrada: "Signature verified and recorded under receipt {recibo}. It has no administrative effect until it is processed through the corporate signing platform.",
  circuito_firma_devuelta: "Return recorded under receipt {recibo}.",
  circuito_firma_error_verificacion: "Signature verification is not enabled on this server, so the signature has not been recorded.",
  circuito_firma_error_no_verificada: "The validator could not verify the signature ({motivo}); it has not been recorded.",
  circuito_firma_error_autofirma: "AutoFirma could not be reached. Check that it is installed and try again.",
  circuito_firma_error_cancelada: "Signing was cancelled in AutoFirma.",
  circuito_firma_error_fallida: "AutoFirma could not sign the document.",
  circuito_firma_error_cambiado: "The case or signing process has changed. Reload the case.",
  circuito_firma_error_cadena: "The draft does not match the one signed at the previous step; it must be returned for redrafting.",
  circuito_firma_error_denegado: "You do not have permission to sign this document.",
  circuito_firma_error_generico: "The operation could not be completed. Please try again.",
});

export function crearTraductorCircuitoFirma(sobrescrituras = {}) {
  if (sobrescrituras === null || typeof sobrescrituras !== "object" || Array.isArray(sobrescrituras)) {
    throw new TypeError("mensajes del circuito de firma no válidos");
  }
  const mensajes = { ...MENSAJES_CIRCUITO_FIRMA_ES };
  for (const clave of Object.keys(MENSAJES_CIRCUITO_FIRMA_ES)) {
    const valor = sobrescrituras[clave];
    if (typeof valor === "string" && valor.trim() !== "") mensajes[clave] = valor;
  }
  return (clave, variables = {}) => {
    if (!Object.hasOwn(mensajes, clave)) throw new Error(`falta la traducción ${clave}`);
    return Object.entries(variables).reduce(
      (texto, [nombre, valor]) => texto.replaceAll(`{${nombre}}`, String(valor)),
      mensajes[clave],
    );
  };
}
