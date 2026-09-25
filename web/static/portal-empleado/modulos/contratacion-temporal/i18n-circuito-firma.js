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
