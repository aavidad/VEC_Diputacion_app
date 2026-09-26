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
import { crearTraductorContratacionTemporal } from "./i18n.js";

const traducirCT = crearTraductorContratacionTemporal();

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
      return { ok: false, status: 0, codigo: "consulta_cancelada", mensaje: traducirCT("ct_txt_consulta_de_estadisticas_cancelada") };
    }
    return { ok: false, status: 0, codigo: "error_red", mensaje: traducirCT("ct_txt_no_se_pudo_conectar_con_el_servicio_de_estadisti") };
  }

  if (!respuesta.ok) {
    if (respuesta.status === 400) {
      return { ok: false, status: 400, codigo: "solicitud_invalida", mensaje: traducirCT("ct_txt_parametros_de_consulta_de_estadisticas_no_valido") };
    }
    if (respuesta.status === 401) {
      return { ok: false, status: 401, codigo: "no_autenticado", mensaje: traducirCT("ct_txt_se_requiere_una_sesion_autenticada_para_consulta") };
    }
    if (respuesta.status === 403) {
      return { ok: false, status: 403, codigo: "acceso_denegado", mensaje: traducirCT("ct_txt_la_sesion_no_dispone_de_permisos_para_consultar") };
    }
    if (respuesta.status === 404) {
      return { ok: false, status: 404, codigo: "no_encontrado", mensaje: traducirCT("ct_txt_el_servicio_de_estadisticas_no_esta_disponible") };
    }
    if (respuesta.status === 422) {
      return { ok: false, status: 422, codigo: "no_procesable", mensaje: traducirCT("ct_txt_rango_de_fechas_o_periodo_no_procesable") };
    }
    return {
      ok: false,
      status: respuesta.status,
      codigo: "error_servidor",
      mensaje: traducirCT("ct_txt_no_se_pudieron_consultar_las_estadisticas_intent"),
    };
  }

  try {
    const envelope = await respuesta.json();
    const datos = validarRespuestaEstadisticas(envelope);
    return { ok: true, datos };
  } catch (error) {
    if (signal?.aborted || error?.name === "AbortError") {
      return { ok: false, status: 0, codigo: "consulta_cancelada", mensaje: traducirCT("ct_txt_consulta_de_estadisticas_cancelada") };
    }
    return {
      ok: false,
      status: respuesta.status,
      codigo: "respuesta_invalida",
      mensaje: traducirCT("ct_txt_la_respuesta_del_servicio_de_estadisticas_no_es"),
    };
  }
}
