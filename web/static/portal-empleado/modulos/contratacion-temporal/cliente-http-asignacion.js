import {
  validarReciboAsignacion,
  validarSolicitudAsignacion,
} from "./contrato-asignacion.js";

export const RUTA_ASIGNACION_CONTRATACION_TEMPORAL =
  "/api/vec/contratacion-temporal/asignaciones";

const MAXIMO_ASIGNACION_BYTES = 8 * 1024;
const RECHAZOS_ASIGNACION_ANTERIORES_AL_EFECTO = new Set([
  "400:peticion_no_valida",
  "400:peticion_no_permitida",
  "401:autenticacion_requerida",
  "403:acceso_denegado",
  "404:recurso_no_encontrado",
  "405:metodo_no_permitido",
  "406:representacion_no_aceptable",
  "409:conflicto",
  "413:peticion_demasiado_grande",
  "415:tipo_contenido_no_admitido",
  "422:contenido_no_valido",
]);

function rechazoAsignacionAnteriorAlEfecto(error) {
  return error?.envelopeValido === true
    && RECHAZOS_ASIGNACION_ANTERIORES_AL_EFECTO.has(
      `${error.estado}:${error.codigo}`,
    );
}

export function crearAsignacionClienteHTTP({
  ejecutar,
  validarOpciones,
  serializarAcotado,
} = {}) {
  if (typeof ejecutar !== "function" || typeof validarOpciones !== "function"
    || typeof serializarAcotado !== "function") {
    throw new TypeError("dependencias HTTP de asignación no disponibles");
  }

  function asignarUnidad(solicitud, opciones) {
    const { signal } = validarOpciones(opciones);
    const entrada = validarSolicitudAsignacion(
      serializarAcotado(solicitud, MAXIMO_ASIGNACION_BYTES),
    );
    return ejecutar({
      ruta: RUTA_ASIGNACION_CONTRATACION_TEMPORAL,
      entrada,
      signal,
      estadoEsperado: 201,
      maximoSolicitud: MAXIMO_ASIGNACION_BYTES,
      maximoRespuesta: MAXIMO_ASIGNACION_BYTES,
      validarRespuesta: (respuesta) => {
        const recibo = validarReciboAsignacion(JSON.stringify(respuesta));
        if (recibo.operacion !== "asignar"
          || recibo.expediente_ref !== entrada.expediente_ref
          || recibo.version_resultante !== entrada.version_esperada + 1) {
          throw new TypeError("el recibo no corresponde a la asignación");
        }
        return recibo;
      },
      efecto: true,
      rechazoDeterminado: rechazoAsignacionAnteriorAlEfecto,
    });
  }

  return Object.freeze({ asignarUnidad });
}
