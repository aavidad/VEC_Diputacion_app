import {
  validarSolicitudSeleccionLlamamiento, validarSolicitudComunicacionLlamamiento,
  validarReciboSeleccionLlamamiento, validarReciboComunicacionLlamamiento,
  validarSolicitudRespuestaRecibida, validarReciboRespuestaRecibida,
  validarSolicitudResolucionLlamamiento, validarReciboResolucionLlamamiento,
  validarSolicitudContinuacionLlamamiento, validarReciboContinuacionLlamamiento,
  validarSolicitudPropuestaFormalizacion, validarReciboPropuestaFormalizacion,
  snapshotsFormalizacionDesarrollo,
} from "./contrato-llamamiento.js";

export const RUTAS_LLAMAMIENTO = Object.freeze({
  seleccionLlamamiento: "/api/vec/contratacion-temporal/llamamientos/seleccion",
  comunicacionLlamamiento: "/api/vec/contratacion-temporal/llamamientos/comunicaciones",
  respuestaRecibida: "/api/vec/contratacion-temporal/llamamientos/respuestas/registro",
  resolucionLlamamiento: "/api/vec/contratacion-temporal/llamamientos/resoluciones",
  continuacionLlamamiento: "/api/vec/contratacion-temporal/llamamientos/siguientes",
  propuestaFormalizacion: "/api/vec/contratacion-temporal/formalizacion/propuestas",
});
export const RUTA_PUBLICACIONES_FORMALIZACION = "/portal-empleado/modulos/contratacion-temporal/formalizacion-desarrollo.json";
// Asset público, no endpoint de autoridad ni material remitido por el usuario.
export async function cargarPublicacionesFormalizacionDesarrollo({
  fetchImpl = globalThis.fetch, criptografia = globalThis.crypto, signal,
} = {}) {
  let lector;
  try {
    if (typeof fetchImpl !== "function" || signal?.aborted) throw new TypeError();
    const r = await fetchImpl(RUTA_PUBLICACIONES_FORMALIZACION, {
      method: "GET", credentials: "omit", mode: "same-origin", cache: "no-store",
      redirect: "error", referrerPolicy: "no-referrer", headers: { Accept: "application/json" }, signal,
    });
    lector = r?.body?.getReader();
    if (r?.status !== 200 || r.redirected || !lector
      || !/^application\/json(?:;\s*charset=utf-8)?$/iu.test(r.headers.get("content-type") ?? "")
      || Number(r.headers.get("content-length")) > 16384) throw new TypeError();
    const bytes = new Uint8Array(16384); let longitud = 0;
    for (;;) {
      const { value, done } = await lector.read();
      if (signal?.aborted) throw new TypeError();
      if (done) break;
      if (!(value instanceof Uint8Array) || longitud + value.length > bytes.length) throw new TypeError();
      bytes.set(value, longitud); longitud += value.length;
    }
    const asset = JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(bytes.subarray(0, longitud)));
    const snapshots = await snapshotsFormalizacionDesarrollo(asset, criptografia);
    if (signal?.aborted) throw new TypeError();
    return snapshots;
  } catch {
    throw new TypeError("publicaciones de formalización no disponibles");
  } finally {
    try { await lector?.cancel(); } catch { /* Cierre de la lectura pública. */ }
    lector?.releaseLock();
  }
}
export function prefijoErrorLlamamiento(ruta) {
  if (ruta === RUTAS_LLAMAMIENTO.propuestaFormalizacion) return "api.contratacion_temporal.propuesta_formalizacion.error.";
  if (ruta === RUTAS_LLAMAMIENTO.seleccionLlamamiento) {
    return "api.contratacion_temporal.seleccion_llamamiento.error.";
  }
  if (ruta === RUTAS_LLAMAMIENTO.comunicacionLlamamiento
    || ruta === RUTAS_LLAMAMIENTO.resolucionLlamamiento
    || ruta === RUTAS_LLAMAMIENTO.continuacionLlamamiento) {
    return "api.contratacion_temporal.comunicacion_llamamiento.error.";
  }
  if (ruta === RUTAS_LLAMAMIENTO.respuestaRecibida) {
    return "api.contratacion_temporal.respuesta_recibida.error.";
  }
  return null;
}
export function conflictoLlamamientoValido(ruta, codigo) {
  if (ruta === RUTAS_LLAMAMIENTO.propuestaFormalizacion)
    return ["version_en_conflicto", "clave_idempotencia_reutilizada", "resolucion_no_aceptada"].includes(codigo);
  if (ruta === RUTAS_LLAMAMIENTO.continuacionLlamamiento) return codigo === "clave_idempotencia_reutilizada";
  if (ruta === RUTAS_LLAMAMIENTO.resolucionLlamamiento
    && codigo === "validacion_respuesta_pendiente") return true;
  return (ruta === RUTAS_LLAMAMIENTO.seleccionLlamamiento
    ? ["conflicto_no_reintentable", "seleccion_no_disponible"]
    : ["version_en_conflicto", "clave_idempotencia_reutilizada"]).includes(codigo);
}
const RECHAZOS_PREVIOS = new Set([
  "400:peticion_no_valida", "400:peticion_no_permitida",
  "401:autenticacion_requerida", "403:acceso_denegado", "404:recurso_no_encontrado",
  "405:metodo_no_permitido", "406:representacion_no_aceptable",
  "413:peticion_demasiado_grande", "415:tipo_contenido_no_admitido",
  "422:contenido_no_valido",
]);
export function esValidacionRespuestaPendiente(error) {
  return error?.envelopeValido === true && error.estado === 409
    && error.codigo === "validacion_respuesta_pendiente";
}
export function crearLlamamientoClienteHTTP({ ejecutar, validarOpciones } = {}) {
  if (typeof ejecutar !== "function" || typeof validarOpciones !== "function") {
    throw new TypeError("dependencias HTTP de llamamiento no disponibles");
  }
  function enviar(ruta, entrada, opciones, validarRespuesta) {
    const { signal } = validarOpciones(opciones);
    return ejecutar({
      ruta, entrada, signal, estadoEsperado: [200, 201],
      maximoSolicitud: 4096, maximoRespuesta: 4096, validarRespuesta, efecto: true,
      rechazoDeterminado: (error) => error?.envelopeValido === true
        && (RECHAZOS_PREVIOS.has(`${error.estado}:${error.codigo}`)
          || (ruta === RUTAS_LLAMAMIENTO.resolucionLlamamiento
            && esValidacionRespuestaPendiente(error))),
    });
  }
  return Object.freeze({
    seleccionarLlamamiento(solicitud, opciones) {
      return enviar(RUTAS_LLAMAMIENTO.seleccionLlamamiento,
        validarSolicitudSeleccionLlamamiento(solicitud), opciones,
        validarReciboSeleccionLlamamiento);
    },
    registrarComunicacionLlamamiento(solicitud, opciones) {
      const entrada = validarSolicitudComunicacionLlamamiento(solicitud);
      return enviar(RUTAS_LLAMAMIENTO.comunicacionLlamamiento, entrada, opciones,
        (respuesta) => validarReciboComunicacionLlamamiento(respuesta, entrada));
    },
    registrarRespuestaRecibida(solicitud, opciones) {
      const entrada = validarSolicitudRespuestaRecibida(solicitud);
      return enviar(RUTAS_LLAMAMIENTO.respuestaRecibida, entrada, opciones,
        (respuesta) => validarReciboRespuestaRecibida(respuesta, entrada));
    },
    resolverLlamamiento(solicitud, opciones) {
      const entrada = validarSolicitudResolucionLlamamiento(solicitud);
      return enviar(RUTAS_LLAMAMIENTO.resolucionLlamamiento, entrada, opciones,
        (respuesta) => validarReciboResolucionLlamamiento(respuesta, entrada));
    },
    continuarLlamamiento(solicitud, opciones) {
      const entrada = validarSolicitudContinuacionLlamamiento(solicitud);
      return enviar(RUTAS_LLAMAMIENTO.continuacionLlamamiento, entrada, opciones,
        (respuesta) => validarReciboContinuacionLlamamiento(respuesta, entrada));
    },
    prepararPropuestaFormalizacion(solicitud, opciones) {
      const entrada = validarSolicitudPropuestaFormalizacion(solicitud);
      return enviar(RUTAS_LLAMAMIENTO.propuestaFormalizacion, entrada, opciones,
        (respuesta, status) => {
          const recibo = validarReciboPropuestaFormalizacion(respuesta, entrada);
          if (recibo.estado_local !== (status === 201 ? "confirmado" : "replay_confirmado")) throw new TypeError();
          return recibo;
        });
    },
  });
}
