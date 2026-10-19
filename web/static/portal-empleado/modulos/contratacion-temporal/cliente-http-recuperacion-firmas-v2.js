import {
  validarSolicitudRecuperacionFirmasV2, validarRespuestaRecuperacionFirmasV2, validarRespuestaConsultaFirmasV2,
} from "./contrato-recuperacion-firmas-v2.js?v=20261016-firma-historica-v5";

export const RUTA_RECUPERACION_FIRMAS_V2 =
  "/api/vec/contratacion-temporal/firmas-documento/recuperaciones-v2";
export const RUTA_CONSULTA_FIRMAS_V2 =
  "/api/vec/contratacion-temporal/firmas-documento/consultas-v2";

function aborto() {
  const error = new Error("operacion_abortada");
  error.name = "AbortError";
  return error;
}

// ejecutar es el transporte interno ya compuesto con mTLS, límites y
// credenciales de mismo origen. Esta hoja no lee actor ni certificado.
export function crearClienteRecuperacionFirmasV2({ ejecutar, validarOpciones, cryptoImpl = globalThis.crypto } = {}) {
  if (typeof ejecutar !== "function" || typeof validarOpciones !== "function"
    || typeof cryptoImpl?.subtle?.digest !== "function") {
    throw new TypeError("cliente_recuperacion_firmas_no_disponible");
  }
  return Object.freeze({
    async consultar(solicitud, opciones) {
      const q = validarSolicitudRecuperacionFirmasV2(solicitud);
      const { signal } = validarOpciones(opciones);
      if (signal?.aborted) throw aborto();
      const data = await ejecutar({
        ruta: RUTA_CONSULTA_FIRMAS_V2, metodo: "POST", entrada: q, signal,
        tipoContenido: "application/json",
        estadoEsperado: 200, maximoSolicitud: 4 * 1024, maximoRespuesta: 512 * 1024,
        efecto: false, validarRespuesta: (respuesta) => validarRespuestaConsultaFirmasV2(respuesta, q),
      });
      if (signal?.aborted) throw aborto();
      return data;
    },
    async recuperar(solicitud, opciones) {
      const q = validarSolicitudRecuperacionFirmasV2(solicitud);
      const { signal } = validarOpciones(opciones);
      if (signal?.aborted) throw aborto();
      const data = await ejecutar({
        ruta: RUTA_RECUPERACION_FIRMAS_V2, metodo: "POST", entrada: q, signal,
        tipoContenido: "application/json",
        estadoEsperado: 200, maximoSolicitud: 4 * 1024,
        maximoRespuesta: 8 * 1024 * 1024, efecto: false,
        validarRespuesta: (respuesta) => respuesta,
      });
      if (signal?.aborted) throw aborto();
      const resultado = await validarRespuestaRecuperacionFirmasV2(data, q, cryptoImpl);
      if (signal?.aborted) throw aborto();
      return resultado;
    },
  });
}
