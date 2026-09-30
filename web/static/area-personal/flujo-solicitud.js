/**
 * Reglas puras del asistente de solicitud.
 *
 * La interfaz y los adaptadores aplican además sus propias comprobaciones. Este
 * módulo evita que la navegación visual pueda saltarse requisitos previos y no
 * concede por sí mismo ninguna capacidad administrativa.
 */
import { traducir } from "./i18n.js";

const mensaje = (clave) => traducir(`areaPersonal.flujo.${clave}`);

function marcado(valor) {
  return valor === true || valor === "true" || valor === "on";
}

function referencias(valor) {
  const valores = Array.isArray(valor) ? valor : valor ? [valor] : [];
  return [...new Set(valores
    .filter((item) => typeof item === "string" && item.trim() !== "")
    .map((item) => item.trim().slice(0, 100)))];
}

function exigirPaso(paso) {
  if (!Number.isInteger(paso) || paso < 1 || paso > 4) {
    throw new TypeError(mensaje("pasoNoValido"));
  }
}

export function crearProgresoSolicitud(convocatoriaId = "") {
  return Object.freeze({
    convocatoria_id: String(convocatoriaId || "").slice(0, 100),
    requisitos_confirmados: false,
    datos_confirmados: false,
    meritos_ids: Object.freeze([]),
    autobaremo_revisado: false,
  });
}

export function aplicarPasoSolicitud(progreso, paso, entrada = {}) {
  exigirPaso(paso);
  const actual = progreso && typeof progreso === "object"
    ? progreso
    : crearProgresoSolicitud();
  const siguiente = {
    convocatoria_id: String(actual.convocatoria_id || "").slice(0, 100),
    requisitos_confirmados: actual.requisitos_confirmados === true,
    datos_confirmados: actual.datos_confirmados === true,
    meritos_ids: referencias(actual.meritos_ids),
    autobaremo_revisado: actual.autobaremo_revisado === true,
  };

  if (paso === 1) {
    const convocatoriaId = String(entrada.convocatoria || "").trim().slice(0, 100);
    if (!convocatoriaId) throw new Error(mensaje("sinConvocatoria"));
    if (!marcado(entrada.requisitos_confirmados)) {
      throw new Error(mensaje("sinRequisitos"));
    }
    return Object.freeze({
      ...crearProgresoSolicitud(convocatoriaId),
      requisitos_confirmados: true,
    });
  }

  if (!siguiente.convocatoria_id || !siguiente.requisitos_confirmados) {
    throw new Error(mensaje("completarPaso1"));
  }
  if (paso === 2) {
    if (!marcado(entrada.datos_confirmados)) {
      throw new Error(mensaje("sinDatos"));
    }
    siguiente.datos_confirmados = true;
  }
  if (paso >= 3 && !siguiente.datos_confirmados) {
    throw new Error(mensaje("completarPaso2"));
  }
  if (paso === 3) {
    siguiente.meritos_ids = referencias(entrada.meritos);
    if (siguiente.meritos_ids.length === 0) {
      throw new Error(mensaje("sinMeritos"));
    }
  }
  if (paso === 4) {
    if (siguiente.meritos_ids.length === 0) {
      throw new Error(mensaje("sinMeritosAutobaremo"));
    }
    siguiente.autobaremo_revisado = true;
  }
  siguiente.meritos_ids = Object.freeze([...siguiente.meritos_ids]);
  return Object.freeze(siguiente);
}

export function crearPayloadBorrador(progreso, solicitudId = "") {
  const actual = progreso && typeof progreso === "object" ? progreso : {};
  if (!actual.convocatoria_id || actual.requisitos_confirmados !== true
    || actual.datos_confirmados !== true || referencias(actual.meritos_ids).length === 0
    || actual.autobaremo_revisado !== true) {
    throw new Error(mensaje("borradorIncompleto"));
  }
  return Object.freeze({
    id: String(solicitudId || "").slice(0, 100),
    convocatoria_id: String(actual.convocatoria_id).slice(0, 100),
    requisitos_confirmados: true,
    datos_confirmados: true,
    meritos_ids: Object.freeze(referencias(actual.meritos_ids)),
    autobaremo_revisado: true,
  });
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
