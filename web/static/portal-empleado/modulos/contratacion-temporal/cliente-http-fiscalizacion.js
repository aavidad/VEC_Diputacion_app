import {
  validarReciboResultadoFiscalizacion,
  validarSolicitudResultadoFiscalizacion,
} from "./contrato-fiscalizacion.js";

export const RUTA_RESULTADOS_FISCALIZACION =
  "/api/vec/contratacion-temporal/fiscalizaciones/resultados";

const MAXIMO_SOLICITUD_BYTES = 16 * 1024;
const MAXIMO_RESPUESTA_BYTES = 16 * 1024;
const RECHAZOS_ANTERIORES_AL_EFECTO = new Set([
  "400:peticion_no_valida", "400:peticion_no_permitida",
  "401:autenticacion_requerida", "403:acceso_denegado",
  "404:recurso_no_encontrado", "405:metodo_no_permitido",
  "406:representacion_no_aceptable", "409:conflicto",
  "413:peticion_demasiado_grande", "415:tipo_contenido_no_admitido",
  "422:contenido_no_valido",
]);

function rechazoAnteriorAlEfecto(error) {
  return error?.envelopeValido === true
    && RECHAZOS_ANTERIORES_AL_EFECTO.has(`${error.estado}:${error.codigo}`);
}

export function crearFiscalizacionClienteHTTP({
  ejecutar,
  validarOpciones,
  serializarAcotado,
} = {}) {
  if (typeof ejecutar !== "function" || typeof validarOpciones !== "function"
    || typeof serializarAcotado !== "function") {
    throw new TypeError("dependencias HTTP de fiscalización no disponibles");
  }

  function registrarResultadoFiscalizacion(solicitud, opciones) {
    const { signal } = validarOpciones(opciones);
    const entrada = validarSolicitudResultadoFiscalizacion(
      serializarAcotado(solicitud, MAXIMO_SOLICITUD_BYTES),
    );
    return ejecutar({
      ruta: RUTA_RESULTADOS_FISCALIZACION,
      entrada,
      signal,
      estadoEsperado: 201,
      maximoSolicitud: MAXIMO_SOLICITUD_BYTES,
      maximoRespuesta: MAXIMO_RESPUESTA_BYTES,
      validarRespuesta: (respuesta) => {
        const recibo = validarReciboResultadoFiscalizacion(JSON.stringify(respuesta));
        if (recibo.expediente_ref !== entrada.expediente_ref
          || recibo.version_resultante !== entrada.version_esperada + 1
          || recibo.resultado !== entrada.resultado) {
          throw new TypeError("el recibo no corresponde al resultado de fiscalización");
        }
        return recibo;
      },
      efecto: true,
      rechazoDeterminado: rechazoAnteriorAlEfecto,
    });
  }

  return Object.freeze({ registrarResultadoFiscalizacion });
}
