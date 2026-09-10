import {
  validarSolicitudIncorporacionEjercicio, validarReciboIncorporacionEjercicio,
  validarPreparacionIncorporacionEjercicio,
} from "./contrato-incorporacion-ejercicio.js";

export const RUTA_INCORPORACION_EJERCICIO = "/api/vec/contratacion-temporal/incorporaciones-ejercicio";

export function crearIncorporacionEjercicioClienteHTTP({ ejecutar, validarOpciones } = {}) {
  if (typeof ejecutar !== "function" || typeof validarOpciones !== "function") {
    throw new TypeError("cliente de incorporación de ejercicio no disponible");
  }
  return Object.freeze({
    prepararIncorporacionEjercicio(expediente_ref, opciones) {
      if (typeof expediente_ref !== "string" || !/^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u.test(expediente_ref)) {
        throw new TypeError("expediente de preparación no válido");
      }
      const { signal } = validarOpciones(opciones);
      return ejecutar({ metodo: "GET", ruta: `${RUTA_INCORPORACION_EJERCICIO}?expediente_ref=${encodeURIComponent(expediente_ref)}`,
        signal, estadoEsperado: 200, maximoRespuesta: 12288, efecto: false,
        validarRespuesta: (respuesta) => validarPreparacionIncorporacionEjercicio(respuesta, expediente_ref) });
    },
    confirmarIncorporacionEjercicio(solicitud, opciones) {
      const entrada = validarSolicitudIncorporacionEjercicio(solicitud);
      const { signal } = validarOpciones(opciones);
      return ejecutar({ ruta: RUTA_INCORPORACION_EJERCICIO, entrada, signal, estadoEsperado: 200,
        maximoSolicitud: 8192, maximoRespuesta: 4096, efecto: true,
        validarRespuesta: (respuesta) => validarReciboIncorporacionEjercicio(respuesta, entrada),
        rechazoDeterminado: (error) => error?.envelopeValido === true && [400, 403, 422].includes(error.estado) });
    },
  });
}
