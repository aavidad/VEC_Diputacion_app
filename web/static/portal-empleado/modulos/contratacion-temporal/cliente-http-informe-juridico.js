import {
  validarReciboInformeJuridico,
  validarSolicitudInformeJuridico,
} from "./contrato-informe-juridico.js";

export const RUTA_PREPARACION_INFORME_JURIDICO =
  "/api/vec/contratacion-temporal/informes-juridicos/preparaciones";

const MAXIMO_SOLICITUD_BYTES = 4 * 1024;
const MAXIMO_RESPUESTA_BYTES = 288 * 1024;
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

export function crearInformeJuridicoClienteHTTP({
  ejecutar,
  validarOpciones,
  serializarAcotado,
} = {}) {
  if (typeof ejecutar !== "function" || typeof validarOpciones !== "function"
    || typeof serializarAcotado !== "function") {
    throw new TypeError("dependencias HTTP de informe jurídico no disponibles");
  }

  function prepararInformeJuridico(solicitud, opciones) {
    const { signal } = validarOpciones(opciones);
    const entrada = validarSolicitudInformeJuridico(
      serializarAcotado(solicitud, MAXIMO_SOLICITUD_BYTES),
    );
    return ejecutar({
      ruta: RUTA_PREPARACION_INFORME_JURIDICO,
      entrada,
      signal,
      estadoEsperado: 201,
      maximoSolicitud: MAXIMO_SOLICITUD_BYTES,
      maximoRespuesta: MAXIMO_RESPUESTA_BYTES,
      validarRespuesta: (respuesta) => {
        const recibo = validarReciboInformeJuridico(JSON.stringify(respuesta));
        if (recibo.expediente_ref !== entrada.expediente_ref
          || recibo.version_resultante !== entrada.version_esperada + 1) {
          throw new TypeError("el recibo no corresponde al informe jurídico");
        }
        return recibo;
      },
      efecto: true,
      rechazoDeterminado: rechazoAnteriorAlEfecto,
    });
  }

  return Object.freeze({ prepararInformeJuridico });
}
