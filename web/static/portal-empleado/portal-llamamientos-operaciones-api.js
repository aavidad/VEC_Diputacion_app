import {
  construirEnvelopeAccionBolsa,
  validarPayloadCrearLlamamiento,
  validarPayloadResultadoLlamamiento,
} from "./portal-bolsas-contrato.js";
import { validarEmisionLlamamiento } from "./portal-llamamientos-contrato.js?v=20260718-llamamientos-v1";

export const RUTA_EMISIONES_LLAMAMIENTO = "/api/vec/bolsa/llamamientos/emisiones";

function segmentoRuta(referencia) {
  return encodeURIComponent(String(referencia ?? "").trim()).replace(/%3A/gi, ":");
}

export async function emitirLlamamiento(payload, { fetchImpl = fetch } = {}) {
  if (!payload?.bolsa_ref || !Array.isArray(payload.participaciones) || payload.participaciones.length === 0 || !payload.configuracion || !payload.clave_idempotencia) {
    return { ok: false, status: 400, codigo: "solicitud_invalida", mensaje: "Faltan datos obligatorios del llamamiento." };
  }
  try {
    const respuesta = await fetchImpl(RUTA_EMISIONES_LLAMAMIENTO, {
      method: "POST", credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error",
      headers: { Accept: "application/json", "Content-Type": "application/json", "Idempotency-Key": payload.clave_idempotencia },
      body: JSON.stringify({ bolsa_ref: payload.bolsa_ref, participaciones: payload.participaciones, configuracion: payload.configuracion }),
    });
    const cuerpo = await respuesta.json().catch(() => ({}));
    if (respuesta.status === 201 || respuesta.status === 200) return { ok: true, datos: validarEmisionLlamamiento(cuerpo?.data) };
    const mensajes = { 403: "La sesión no dispone de permiso para emitir llamamientos.", 409: "La clave ya corresponde a otro llamamiento.", 422: "La selección ya no respeta la bolsa o la disponibilidad vigente." };
    return { ok: false, status: respuesta.status, codigo: cuerpo?.error?.codigo || "error_servidor", mensaje: mensajes[respuesta.status] || "No se pudo emitir el llamamiento." };
  } catch (error) {
    return { ok: false, status: 0, codigo: "error_red", mensaje: error instanceof Error ? error.message : "Error de comunicación al emitir." };
  }
}

export async function crearLlamamientoCandidato(participacionRef, payload, { fetchImpl = fetch } = {}) {
  if (typeof participacionRef !== "string" || participacionRef.trim() === "") return { ok: false, status: 400, codigo: "referencia_invalida", mensaje: "Referencia de candidato no válida." };
  let envelopeAccion;
  try {
    envelopeAccion = construirEnvelopeAccionBolsa("crear_llamamiento", validarPayloadCrearLlamamiento(payload), { confirmacion: true });
  } catch (error) {
    return { ok: false, status: 400, codigo: "solicitud_invalida", mensaje: error instanceof Error ? error.message : "Datos de llamamiento no válidos." };
  }
  try {
    const respuesta = await fetchImpl(`/api/vec/bolsa/candidatos/${segmentoRuta(participacionRef)}/llamamientos`, { method: "POST", credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error", headers: { "Content-Type": "application/json", Accept: "application/json" }, body: JSON.stringify(envelopeAccion) });
    if (!respuesta.ok) {
      const mensajes = { 400: "Solicitud de llamamiento rechazada por el servidor.", 401: "Se requiere una sesión interna autenticada.", 403: "Sin permiso para registrar llamamientos.", 404: "Candidato no encontrado para el llamamiento.", 409: "El aspirante no está disponible o ya tiene un llamamiento en curso.", 422: "Los datos del llamamiento no cumplen las reglas de negocio." };
      return { ok: false, status: respuesta.status, codigo: respuesta.status === 401 ? "no_autenticado" : respuesta.status === 403 ? "acceso_denegado" : respuesta.status === 404 ? "no_encontrado" : respuesta.status === 409 ? "conflicto" : respuesta.status === 422 ? "no_procesable" : respuesta.status === 400 ? "solicitud_invalida" : "error_servidor", mensaje: mensajes[respuesta.status] || `No se pudo registrar el llamamiento (HTTP ${respuesta.status}).` };
    }
    const envelope = await respuesta.json();
    return { ok: true, datos: envelope.data || envelope };
  } catch (error) {
    return { ok: false, status: 0, codigo: "error_red_o_contrato", mensaje: error instanceof Error ? error.message : "Error de comunicación al registrar llamamiento." };
  }
}

export async function registrarResultadoLlamamiento(llamamientoRef, payload, { fetchImpl = fetch } = {}) {
  if (typeof llamamientoRef !== "string" || llamamientoRef.trim() === "") return { ok: false, status: 400, codigo: "referencia_invalida", mensaje: "Referencia de llamamiento no válida." };
  let envelopeAccion;
  try {
    envelopeAccion = construirEnvelopeAccionBolsa("registrar_resultado", validarPayloadResultadoLlamamiento(payload), { confirmacion: true });
  } catch (error) {
    return { ok: false, status: 400, codigo: "solicitud_invalida", mensaje: error instanceof Error ? error.message : "Datos de resultado no válidos." };
  }
  try {
    const respuesta = await fetchImpl(`/api/vec/bolsa/llamamientos/${segmentoRuta(llamamientoRef)}/resultado`, { method: "POST", credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error", headers: { "Content-Type": "application/json", Accept: "application/json" }, body: JSON.stringify(envelopeAccion) });
    if (!respuesta.ok) {
      const mensajes = { 400: "Solicitud de resultado rechazada por el servidor.", 401: "Se requiere una sesión interna autenticada.", 403: "Sin permiso para registrar resultado de llamamiento.", 404: "Llamamiento no encontrado.", 409: "El llamamiento ya tiene resultado o su estado no permite registrarlo.", 422: "El resultado no es procesable según las reglas de bolsa." };
      return { ok: false, status: respuesta.status, codigo: respuesta.status === 401 ? "no_autenticado" : respuesta.status === 403 ? "acceso_denegado" : respuesta.status === 404 ? "no_encontrado" : respuesta.status === 409 ? "conflicto" : respuesta.status === 422 ? "no_procesable" : respuesta.status === 400 ? "solicitud_invalida" : "error_servidor", mensaje: mensajes[respuesta.status] || `No se pudo registrar el resultado (HTTP ${respuesta.status}).` };
    }
    const envelope = await respuesta.json();
    return { ok: true, datos: envelope.data || envelope };
  } catch (error) {
    return { ok: false, status: 0, codigo: "error_red_o_contrato", mensaje: error instanceof Error ? error.message : "Error de comunicación al registrar resultado." };
  }
}
