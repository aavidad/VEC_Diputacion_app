/**
 * Utilidades puras de las operaciones heredadas del expediente (selección de
 * la solicitud en edición y actos ya confirmados). El asistente de solicitud
 * de participación vive en solicitud-convocatoria.js: allí los méritos son
 * opcionales y plazo, requisitos y puntuación los decide el servidor.
 */

function marcado(valor) {
  return valor === true || valor === "true" || valor === "on";
}

export function localizarSolicitudEdicion(datos, { solicitudId = "", convocatoriaId = "" } = {}) {
  const solicitudes = Array.isArray(datos?.solicitudes) ? datos.solicitudes : [];
  const exacta = solicitudId
    ? solicitudes.find((item) => item?.id === solicitudId)
    : null;
  if (exacta) return exacta;
  return solicitudes.find((item) => item?.convocatoria_id === convocatoriaId
    && /borrador/i.test(String(item?.estado || ""))) || null;
}

export function estadoActosSolicitud(solicitud) {
  const pago = String(solicitud?.pago || "");
  const firma = String(solicitud?.firma || "");
  const estado = String(solicitud?.estado || "");
  return Object.freeze({
    pagoConfirmado: !/pendiente/i.test(pago) && /confirmad|abonad|exent/i.test(pago),
    firmaConfirmada: !/pendiente/i.test(firma) && /confirmad|firmad|válid/i.test(firma),
    registrada: /registr/i.test(estado) && !/borrador/i.test(estado),
  });
}

export function declaracionFinalConfirmada(valor) {
  return marcado(valor);
}
