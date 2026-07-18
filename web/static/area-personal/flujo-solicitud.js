/**
 * Reglas puras del asistente de solicitud.
 *
 * La interfaz y los adaptadores aplican además sus propias comprobaciones. Este
 * módulo evita que la navegación visual pueda saltarse requisitos previos y no
 * concede por sí mismo ninguna capacidad administrativa.
 */

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
    throw new TypeError("El paso de solicitud no es válido.");
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
    if (!convocatoriaId) throw new Error("Seleccione una convocatoria con plazo abierto.");
    if (!marcado(entrada.requisitos_confirmados)) {
      throw new Error("Debe confirmar que ha leído las bases y que cumple los requisitos declarados.");
    }
    return Object.freeze({
      ...crearProgresoSolicitud(convocatoriaId),
      requisitos_confirmados: true,
    });
  }

  if (!siguiente.convocatoria_id || !siguiente.requisitos_confirmados) {
    throw new Error("Complete primero la convocatoria y la declaración de requisitos.");
  }
  if (paso === 2) {
    if (!marcado(entrada.datos_confirmados)) {
      throw new Error("Debe confirmar que sus datos personales y de contacto son correctos.");
    }
    siguiente.datos_confirmados = true;
  }
  if (paso >= 3 && !siguiente.datos_confirmados) {
    throw new Error("Confirme sus datos antes de seleccionar méritos.");
  }
  if (paso === 3) {
    siguiente.meritos_ids = referencias(entrada.meritos);
    if (siguiente.meritos_ids.length === 0) {
      throw new Error("Seleccione al menos un mérito para continuar.");
    }
  }
  if (paso === 4) {
    if (siguiente.meritos_ids.length === 0) {
      throw new Error("Seleccione al menos un mérito antes de revisar la autobaremación.");
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
    throw new Error("El borrador no reúne convocatoria, requisitos, datos, méritos y autobaremación revisada.");
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
