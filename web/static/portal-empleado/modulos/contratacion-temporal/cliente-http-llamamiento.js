import {
  validarSolicitudSeleccionLlamamiento, validarSolicitudComunicacionLlamamiento,
  validarReciboSeleccionLlamamiento, validarReciboComunicacionLlamamiento,
  validarSolicitudRespuestaRecibida, validarReciboRespuestaRecibida,
} from "./contrato-llamamiento.js";

export const RUTAS_LLAMAMIENTO = Object.freeze({
  seleccionLlamamiento: "/api/vec/contratacion-temporal/llamamientos/seleccion",
  comunicacionLlamamiento: "/api/vec/contratacion-temporal/llamamientos/comunicaciones",
  respuestaRecibida: "/api/vec/contratacion-temporal/llamamientos/respuestas/registro",
});
export function prefijoErrorLlamamiento(ruta) {
  if (ruta === RUTAS_LLAMAMIENTO.seleccionLlamamiento) {
    return "api.contratacion_temporal.seleccion_llamamiento.error.";
  }
  if (ruta === RUTAS_LLAMAMIENTO.comunicacionLlamamiento) {
    return "api.contratacion_temporal.comunicacion_llamamiento.error.";
  }
  if (ruta === RUTAS_LLAMAMIENTO.respuestaRecibida) {
    return "api.contratacion_temporal.respuesta_recibida.error.";
  }
  return null;
}
export function conflictoLlamamientoValido(ruta, codigo) {
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
        && RECHAZOS_PREVIOS.has(`${error.estado}:${error.codigo}`),
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
  });
}
