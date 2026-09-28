/** Textos del circuito de firma de los borradores del expediente. */

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
  circuito_firma_portafirmas_titulo: "Firma oficial en Firmadoc",
  circuito_firma_portafirmas_pendiente: "Conexión pendiente",
  circuito_firma_portafirmas_sin_envio: "Sin constancia de envío ni firma oficial en VEC.",
  circuito_firma_portafirmas_estado_no_disponible: "No se puede consultar el circuito de firma.",
  circuito_firma_consulta_denegada: "No dispone de permiso para consultar las firmas de este expediente.",
  circuito_firma_consulta_no_disponible: "El estado de las firmas no está disponible. Vuelva a intentarlo.",
  circuito_firma_autofirma_prueba: "Firmas de prueba con AutoFirma",
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
  circuito_firma_titulo: "Signing workflow",
  circuito_firma_ejemplo: "Example workflow",
  circuito_firma_pasos: "Signing steps for {documento}",
  circuito_firma_estado_pendiente_firma: "Awaiting signature by {cargo}",
  circuito_firma_estado_en_espera: "Waiting for the previous step",
  circuito_firma_estado_firmado: "Signed by {cargo}",
  circuito_firma_estado_devuelto: "Returned by {cargo}",
  circuito_firma_accion_firma: "Signature",
  circuito_firma_accion_visto_bueno: "Approval",
  circuito_firma_habilita_siguiente_paso: "Allows the next signer to proceed",
  circuito_firma_habilita_remision_intervencion: "Allows referral to Financial Control",
  circuito_firma_habilita_envio_notificacion: "Allows the notification to be sent",
  circuito_firma_habilita_envio_comunicacion: "Allows the communication to be sent",
  circuito_firma_habilita_cierre_circuito: "Completes the workflow",
  circuito_firma_devolucion_vuelve_a_redaccion: "If returned, goes back to drafting",
  circuito_firma_devolucion_vuelve_paso_anterior: "If returned, goes back to the previous step",
  circuito_firma_sustitucion_suplente_designado: "A designated substitute may sign",
  circuito_firma_sustitucion_no_admitida: "No substitution allowed",
  circuito_firma_sin_eficacia: "Test signature with no administrative effect",
  circuito_firma_portafirmas_titulo: "Official signing in Firmadoc",
  circuito_firma_portafirmas_pendiente: "Connection pending",
  circuito_firma_portafirmas_sin_envio: "No recorded submission or official signature in VEC.",
  circuito_firma_portafirmas_estado_no_disponible: "The signing workflow cannot be retrieved.",
  circuito_firma_consulta_denegada: "You are not authorised to view the signatures for this case.",
  circuito_firma_consulta_no_disponible: "The signature status is unavailable. Please try again.",
  circuito_firma_autofirma_prueba: "Test signatures with AutoFirma",
  circuito_firma_motivo: "Reason: {motivo}",
  circuito_firma_firmar: "Sign",
  circuito_firma_devolver: "Return",
  circuito_firma_motivo_etiqueta: "Reason for returning",
  circuito_firma_confirmar_devolucion: "Record the return",
  circuito_firma_cancelar: "Cancel",
  circuito_firma_motivo_invalido: "Enter a reason of 3 to 500 characters.",
  circuito_firma_preparando: "Preparing the draft for signing…",
  circuito_firma_abriendo_autofirma: "Opening AutoFirma: sign the document in the AutoFirma window.",
  circuito_firma_registrando: "Verifying and recording…",
  circuito_firma_registrada: "Signature verified and recorded under receipt {recibo}. It has no administrative effect until processed by the corporate signing platform.",
  circuito_firma_devuelta: "Return recorded under receipt {recibo}.",
  circuito_firma_error_verificacion: "Signature verification is not enabled on this server, so the signature was not recorded.",
  circuito_firma_error_no_verificada: "The verifier could not validate the signature ({motivo}); it was not recorded.",
  circuito_firma_error_autofirma: "Could not connect to AutoFirma. Check that it is installed and try again.",
  circuito_firma_error_cancelada: "Signing was cancelled in AutoFirma.",
  circuito_firma_error_fallida: "AutoFirma could not sign the document.",
  circuito_firma_error_cambiado: "The case or signing workflow has changed. Reload the case.",
  circuito_firma_error_cadena: "The draft does not match the one signed in the previous step; it must be returned for redrafting.",
  circuito_firma_error_denegado: "You are not authorised to sign this document.",
  circuito_firma_error_generico: "The operation could not be completed. Try again.",
});

export function crearTraductorCircuitoFirma(sobrescrituras = {}, locale = "es-ES") {
  if (sobrescrituras === null || typeof sobrescrituras !== "object" || Array.isArray(sobrescrituras)) {
    throw new TypeError("mensajes del circuito de firma no válidos");
  }
  const mensajes = { ...(locale.startsWith("en") ? MENSAJES_CIRCUITO_FIRMA_EN : MENSAJES_CIRCUITO_FIRMA_ES) };
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
