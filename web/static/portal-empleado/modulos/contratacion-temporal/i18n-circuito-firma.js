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
