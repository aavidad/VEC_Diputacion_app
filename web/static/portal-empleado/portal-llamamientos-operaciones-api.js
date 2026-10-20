import {
  construirEnvelopeAccionBolsa,
  validarPayloadCrearLlamamiento,
  validarPayloadResultadoLlamamiento,
} from "./portal-bolsas-contrato.js?v=VEC_WEB_VERSION";
import { validarEmisionLlamamiento } from "./portal-llamamientos-contrato.js?v=VEC_WEB_VERSION";
import { formatearNumeroPortal, traducirBolsaInterna, traducirPortal } from "./portal-i18n.js?v=VEC_WEB_VERSION";

export const RUTA_EMISIONES_LLAMAMIENTO = "/api/vec/bolsa/llamamientos/emisiones";

function segmentoRuta(referencia) {
  return encodeURIComponent(String(referencia ?? "").trim()).replace(/%3A/gi, ":");
}

export async function emitirLlamamiento(payload, { fetchImpl = fetch } = {}) {
  if (!payload?.bolsa_ref || !Array.isArray(payload.participaciones) || payload.participaciones.length === 0 || !payload.configuracion || !payload.clave_idempotencia) {
    return { ok: false, status: 400, codigo: "solicitud_invalida", mensaje: traducirPortal("txt_faltan_datos_obligatorios_del_llamamiento") };
  }
  try {
    const respuesta = await fetchImpl(RUTA_EMISIONES_LLAMAMIENTO, {
      method: "POST", credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error",
      headers: { Accept: "application/json", "Content-Type": "application/json", "Idempotency-Key": payload.clave_idempotencia },
      body: JSON.stringify({ bolsa_ref: payload.bolsa_ref, participaciones: payload.participaciones, configuracion: payload.configuracion }),
    });
    const cuerpo = await respuesta.json().catch(() => ({}));
    if (respuesta.status === 201 || respuesta.status === 200) return { ok: true, datos: validarEmisionLlamamiento(cuerpo?.data) };
    if (respuesta.status === 409 && cuerpo?.error?.codigo === "llamamiento_fuera_de_orden") {
      const antes = Number.isSafeInteger(cuerpo.error.orden_vigente) && cuerpo.error.orden_vigente > 0 && typeof cuerpo.error.participacion_ref === "string"
        ? { participacion_ref: cuerpo.error.participacion_ref, orden_vigente: cuerpo.error.orden_vigente } : null;
      return { ok: false, status: 409, codigo: "llamamiento_fuera_de_orden", antes, mensaje: mensajeFueraDeOrden(antes) };
    }
    const mensajes = { 403: traducirPortal("txt_la_sesion_no_dispone_de_permiso_para_emitir_llam"), 409: traducirPortal("txt_la_clave_ya_corresponde_a_otro_llamamiento"), 422: traducirPortal("txt_la_seleccion_ya_no_respeta_la_bolsa_o_la_disponi") };
    return { ok: false, status: respuesta.status, codigo: cuerpo?.error?.codigo || "error_servidor", mensaje: mensajes[respuesta.status] || traducirPortal("txt_no_se_pudo_emitir_el_llamamiento") };
  } catch (error) {
    return { ok: false, status: 0, codigo: "error_red", mensaje: error instanceof Error ? error.message : traducirPortal("txt_error_de_comunicacion_al_emitir") };
  }
}

// Regla b01: dice quién va antes. El nombre sale de la lista que la pantalla
// ya tiene cargada; si no está, basta el número de orden.
export function mensajeFueraDeOrden(antes, candidatos = []) {
  if (!antes) return traducirBolsaInterna("b7_fuera_de_orden");
  const orden = formatearNumeroPortal(antes.orden_vigente);
  const nombre = (Array.isArray(candidatos) ? candidatos : []).find((c) => c?.participacion_ref === antes.participacion_ref)?.nombre_visible;
  return typeof nombre === "string" && nombre.trim()
    ? traducirBolsaInterna("b7_fuera_de_orden_persona", { nombre: nombre.trim(), orden })
    : traducirBolsaInterna("b7_fuera_de_orden_numero", { orden });
}

export async function crearLlamamientoCandidato(participacionRef, payload, { fetchImpl = fetch } = {}) {
  if (typeof participacionRef !== "string" || participacionRef.trim() === "") return { ok: false, status: 400, codigo: "referencia_invalida", mensaje: traducirPortal("txt_referencia_de_candidato_no_valida") };
  let envelopeAccion;
  try {
    envelopeAccion = construirEnvelopeAccionBolsa("crear_llamamiento", validarPayloadCrearLlamamiento(payload), { confirmacion: true });
  } catch (error) {
    return { ok: false, status: 400, codigo: "solicitud_invalida", mensaje: error instanceof Error ? error.message : traducirPortal("txt_datos_de_llamamiento_no_validos") };
  }
  try {
    const respuesta = await fetchImpl(`/api/vec/bolsa/candidatos/${segmentoRuta(participacionRef)}/llamamientos`, { method: "POST", credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error", headers: { "Content-Type": "application/json", Accept: "application/json" }, body: JSON.stringify(envelopeAccion) });
    if (!respuesta.ok) {
      const mensajes = { 400: traducirPortal("txt_solicitud_de_llamamiento_rechazada_por_el_servid"), 401: traducirPortal("txt_se_requiere_una_sesion_interna_autenticada"), 403: traducirPortal("txt_sin_permiso_para_registrar_llamamientos"), 404: traducirPortal("txt_candidato_no_encontrado_para_el_llamamiento"), 409: traducirPortal("txt_el_aspirante_no_esta_disponible_o_ya_tiene_un_ll"), 422: traducirPortal("txt_los_datos_del_llamamiento_no_cumplen_las_reglas") };
      return { ok: false, status: respuesta.status, codigo: respuesta.status === 401 ? "no_autenticado" : respuesta.status === 403 ? "acceso_denegado" : respuesta.status === 404 ? "no_encontrado" : respuesta.status === 409 ? "conflicto" : respuesta.status === 422 ? "no_procesable" : respuesta.status === 400 ? "solicitud_invalida" : "error_servidor", mensaje: mensajes[respuesta.status] || traducirPortal("txt_no_se_pudo_registrar_el_llamamiento_http", { estado: respuesta.status }) };
    }
    const envelope = await respuesta.json();
    return { ok: true, datos: envelope.data || envelope };
  } catch (error) {
    return { ok: false, status: 0, codigo: "error_red_o_contrato", mensaje: error instanceof Error ? error.message : traducirPortal("txt_error_de_comunicacion_al_registrar_llamamiento") };
  }
}

export async function registrarResultadoLlamamiento(llamamientoRef, payload, { fetchImpl = fetch } = {}) {
  if (typeof llamamientoRef !== "string" || llamamientoRef.trim() === "") return { ok: false, status: 400, codigo: "referencia_invalida", mensaje: traducirPortal("txt_referencia_de_llamamiento_no_valida") };
  let envelopeAccion;
  try {
    envelopeAccion = construirEnvelopeAccionBolsa("registrar_resultado", validarPayloadResultadoLlamamiento(payload), { confirmacion: true });
  } catch (error) {
    return { ok: false, status: 400, codigo: "solicitud_invalida", mensaje: error instanceof Error ? error.message : traducirPortal("txt_datos_de_resultado_no_validos") };
  }
  try {
    const respuesta = await fetchImpl(`/api/vec/bolsa/llamamientos/${segmentoRuta(llamamientoRef)}/resultado`, { method: "POST", credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error", headers: { "Content-Type": "application/json", Accept: "application/json" }, body: JSON.stringify(envelopeAccion) });
    if (!respuesta.ok) {
      const mensajes = { 400: traducirPortal("txt_solicitud_de_resultado_rechazada_por_el_servidor"), 401: traducirPortal("txt_se_requiere_una_sesion_interna_autenticada"), 403: traducirPortal("txt_sin_permiso_para_registrar_resultado_de_llamamie"), 404: traducirPortal("txt_llamamiento_no_encontrado"), 409: traducirPortal("txt_el_llamamiento_ya_tiene_resultado_o_su_estado_no"), 422: traducirPortal("txt_el_resultado_no_es_procesable_segun_las_reglas_d") };
      return { ok: false, status: respuesta.status, codigo: respuesta.status === 401 ? "no_autenticado" : respuesta.status === 403 ? "acceso_denegado" : respuesta.status === 404 ? "no_encontrado" : respuesta.status === 409 ? "conflicto" : respuesta.status === 422 ? "no_procesable" : respuesta.status === 400 ? "solicitud_invalida" : "error_servidor", mensaje: mensajes[respuesta.status] || traducirPortal("txt_no_se_pudo_registrar_el_resultado_http", { estado: respuesta.status }) };
    }
    const envelope = await respuesta.json();
    return { ok: true, datos: envelope.data || envelope };
  } catch (error) {
    return { ok: false, status: 0, codigo: "error_red_o_contrato", mensaje: error instanceof Error ? error.message : traducirPortal("txt_error_de_comunicacion_al_registrar_resultado") };
  }
}
