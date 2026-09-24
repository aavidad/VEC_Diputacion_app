/**
 * Cliente HTTP seguro para la consulta de estadísticas de contratación temporal (C18 / G16).
 *
 * Sigue la política DEC-053:
 * - credentials: "same-origin".
 * - Accept: "application/json".
 * - Validación exhaustiva con contrato-estadisticas.js.
 */

import {
  PERIODOS_ESTADISTICAS,
  validarRespuestaEstadisticas,
} from "./contrato-estadisticas.js";

export const RUTA_ESTADISTICAS = "/api/vec/contratacion-temporal/estadisticas";

export function construirUrlEstadisticas({ periodo = "mensual", desde = "", hasta = "" } = {}) {
  const parametros = new URLSearchParams();
  if (periodo && PERIODOS_ESTADISTICAS.includes(periodo)) {
    parametros.set("periodo", periodo);
  }
  if (desde && desde.trim() !== "") {
    parametros.set("desde", desde.trim());
  }
  if (hasta && hasta.trim() !== "") {
    parametros.set("hasta", hasta.trim());
  }

  const query = parametros.toString();
  return query ? `${RUTA_ESTADISTICAS}?${query}` : RUTA_ESTADISTICAS;
}

export async function consultarEstadisticas(filtros = {}, { fetchImpl = fetch, signal } = {}) {
  const url = construirUrlEstadisticas(filtros);

  let respuesta;
  try {
    respuesta = await fetchImpl(url, {
      method: "GET",
      credentials: "same-origin",
      headers: { Accept: "application/json" },
      signal,
    });
  } catch (error) {
    if (signal?.aborted || error?.name === "AbortError") {
      return { ok: false, status: 0, codigo: "consulta_cancelada", mensaje: "Consulta de estadísticas cancelada." };
    }
    return { ok: false, status: 0, codigo: "error_red", mensaje: "No se pudo conectar con el servicio de estadísticas." };
  }

  if (!respuesta.ok) {
    if (respuesta.status === 400) {
      return { ok: false, status: 400, codigo: "solicitud_invalida", mensaje: "Parámetros de consulta de estadísticas no válidos." };
    }
    if (respuesta.status === 401) {
      return { ok: false, status: 401, codigo: "no_autenticado", mensaje: "Se requiere una sesión autenticada para consultar estadísticas." };
    }
    if (respuesta.status === 403) {
      return { ok: false, status: 403, codigo: "acceso_denegado", mensaje: "La sesión no dispone de permisos para consultar estadísticas." };
    }
    if (respuesta.status === 404) {
      return { ok: false, status: 404, codigo: "no_encontrado", mensaje: "El servicio de estadísticas no está disponible." };
    }
    if (respuesta.status === 422) {
      return { ok: false, status: 422, codigo: "no_procesable", mensaje: "Rango de fechas o periodo no procesable." };
    }
    return {
      ok: false,
      status: respuesta.status,
      codigo: "error_servidor",
      mensaje: "No se pudieron consultar las estadísticas. Inténtelo de nuevo.",
    };
  }

  try {
    const envelope = await respuesta.json();
    const datos = validarRespuestaEstadisticas(envelope);
    return { ok: true, datos };
  } catch (error) {
    if (signal?.aborted || error?.name === "AbortError") {
      return { ok: false, status: 0, codigo: "consulta_cancelada", mensaje: "Consulta de estadísticas cancelada." };
    }
    return {
      ok: false,
      status: respuesta.status,
      codigo: "respuesta_invalida",
      mensaje: "La respuesta del servicio de estadísticas no es válida.",
    };
  }
}
