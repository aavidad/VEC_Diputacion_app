import { validarReciboReincorporacionRRHH, validarSolicitudReincorporacionRRHH } from "./rrhh-reincorporacion-contrato.js";

export const RUTA_REINCORPORACION_RRHH = "/api/vec/contratacion-temporal/reincorporaciones-titular";

export function crearClienteReincorporacionRRHHHTTP({ ejecutar, validarOpciones } = {}) {
  if (typeof ejecutar !== "function" || typeof validarOpciones !== "function") {
    throw new TypeError("cliente de reincorporación RRHH no disponible");
  }
  return Object.freeze({
    registrarReincorporacion(solicitud, opciones) {
      const entrada = validarSolicitudReincorporacionRRHH(solicitud);
      const { signal } = validarOpciones(opciones);
      return ejecutar({
        ruta: RUTA_REINCORPORACION_RRHH,
        metodo: "POST",
        entrada,
        signal,
        estadoEsperado: [200, 201],
        maximoSolicitud: 8192,
        maximoRespuesta: 8192,
        efecto: true,
        validarRespuesta: (respuesta) => validarReciboReincorporacionRRHH(respuesta, entrada),
        rechazoDeterminado: (error) => error?.envelopeValido === true
          && [400, 401, 403, 409, 422].includes(error.estado),
      });
    },
  });
}
