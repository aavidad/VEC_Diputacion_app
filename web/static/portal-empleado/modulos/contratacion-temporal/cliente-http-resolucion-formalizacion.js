import { validarSolicitudResolucionFormalizacion, validarReciboResolucionFormalizacion, validarPreparacionResolucionFormalizacion } from "./contrato-resolucion-formalizacion.js";
export const RUTA_RESOLUCION_FORMALIZACION = "/api/vec/contratacion-temporal/resoluciones-formalizacion";
export function crearResolucionFormalizacionClienteHTTP({ ejecutar, validarOpciones } = {}) {
  if (typeof ejecutar !== "function" || typeof validarOpciones !== "function") throw new TypeError("cliente de resolución de formalización no disponible");
  return Object.freeze({
    prepararResolucionFormalizacion(expediente_ref, opciones) {
      if (typeof expediente_ref !== "string") throw new TypeError("expediente de preparación no válido");
      const { signal } = validarOpciones(opciones);
      return ejecutar({ metodo: "GET", ruta: `${RUTA_RESOLUCION_FORMALIZACION}?expediente_ref=${encodeURIComponent(expediente_ref)}`,
        signal, estadoEsperado: 200, maximoRespuesta: 4096, efecto: false,
        validarRespuesta: (respuesta) => validarPreparacionResolucionFormalizacion(respuesta, expediente_ref) });
    },
    registrarResolucionFormalizacion(solicitud, opciones) {
      const entrada = validarSolicitudResolucionFormalizacion(solicitud);
      const { signal } = validarOpciones(opciones);
      return ejecutar({ ruta: RUTA_RESOLUCION_FORMALIZACION, entrada, signal, estadoEsperado: [200, 201], maximoSolicitud: 4096, maximoRespuesta: 4096, efecto: true,
        validarRespuesta: (respuesta, estado) => {
          const recibo = validarReciboResolucionFormalizacion(respuesta, entrada);
          if (recibo.estado !== (estado === 201 ? "registrada" : "replay_registrada")) throw new TypeError();
          return recibo;
        },
        rechazoDeterminado: (error) => error?.envelopeValido === true && [400, 403, 422].includes(error.estado),
      });
    },
  });
}
