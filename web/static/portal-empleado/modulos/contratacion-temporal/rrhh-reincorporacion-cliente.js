import { validarReciboReincorporacionRRHH, validarSolicitudReincorporacionRRHH } from "./rrhh-reincorporacion-contrato.js";

export const RUTA_REINCORPORACION_RRHH = "/api/vec/contratacion-temporal/reincorporaciones-titular";
export const RUTA_CAPACIDAD_REINCORPORACION_RRHH = `${RUTA_REINCORPORACION_RRHH}/capacidad`;
const ESQUEMA_CAPACIDAD = "vec.contratacion-temporal.capacidad-reincorporacion-titular.v1";
const REF = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;

export function validarCapacidadReincorporacionRRHH(respuesta) {
  if (!respuesta || typeof respuesta !== "object" || Array.isArray(respuesta)
    || respuesta.esquema !== ESQUEMA_CAPACIDAD
    || typeof respuesta.puede_registrar_reincorporacion_titular !== "boolean") {
    throw new TypeError("capacidad de reincorporación RRHH incompatible");
  }
  return respuesta.puede_registrar_reincorporacion_titular;
}

export function crearClienteReincorporacionRRHHHTTP({ ejecutar, validarOpciones } = {}) {
  if (typeof ejecutar !== "function" || typeof validarOpciones !== "function") {
    throw new TypeError("cliente de reincorporación RRHH no disponible");
  }
  return Object.freeze({
    consultarCapacidadReincorporacion({ expediente_ref, version_esperada }, opciones) {
      if (typeof expediente_ref !== "string" || !REF.test(expediente_ref)
        || !Number.isSafeInteger(version_esperada) || version_esperada < 1) {
        throw new TypeError("contexto de reincorporación RRHH no válido");
      }
      const { signal } = validarOpciones(opciones);
      return ejecutar({ ruta: RUTA_CAPACIDAD_REINCORPORACION_RRHH,
        metodo: "POST", entrada: { expediente_ref, version_esperada }, signal,
        estadoEsperado: 200, maximoSolicitud: 8192, maximoRespuesta: 4096, efecto: false,
        validarRespuesta: validarCapacidadReincorporacionRRHH });
    },
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
