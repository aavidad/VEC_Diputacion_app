import { referenciaLlamamientoValida } from "./contrato-llamamiento.js?v=20261018-f119-conjunto-v1";

export const RUTA_DERIVACION_NOMBRAMIENTO = "/api/vec/contratacion-temporal/nombramientos/derivaciones";
const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/u;
const CAMPOS = ["padre_ref", "version_padre", "oferta_ref", "numero_plaza", "secuencia_plaza",
  "participacion_ref", "recibo_adjudicacion_ref", "clave_idempotencia"];

export function validarSolicitudDerivacion(datos) {
  if (!datos || Object.keys(datos).length !== CAMPOS.length || !CAMPOS.every((campo) => Object.hasOwn(datos, campo))
    || !["padre_ref", "oferta_ref", "participacion_ref", "recibo_adjudicacion_ref"].every((campo) => referenciaLlamamientoValida(datos[campo]))
    || !Number.isSafeInteger(datos.version_padre) || datos.version_padre < 6
    || !Number.isSafeInteger(datos.numero_plaza) || datos.numero_plaza < 1
    || !Number.isSafeInteger(datos.secuencia_plaza) || datos.secuencia_plaza < 1
    || !UUID.test(datos.clave_idempotencia)) throw new TypeError("solicitud_derivacion_invalida");
  return Object.freeze(Object.fromEntries(CAMPOS.map((campo) => [campo, datos[campo]])));
}

export function validarReciboDerivacion(datos, solicitud) {
  if (!datos || datos.esquema !== "vec.ct.nombramiento.derivado.v1"
    || !["confirmado", "replay_confirmado"].includes(datos.estado)
    || datos.padre_ref !== solicitud.padre_ref || datos.numero_plaza !== solicitud.numero_plaza
    || !["hijo_ref", "aceptacion_ref", "recibo_ref"].every((campo) => referenciaLlamamientoValida(datos[campo]))
    || typeof datos.numero_visible !== "string" || datos.numero_visible.length > 50
    || !/^[0-9]{4}\/[A-Za-z0-9._-]{1,45}-P[1-9][0-9]*$/u.test(datos.numero_visible)
    || !Number.isSafeInteger(datos.total_personas) || datos.total_personas < datos.numero_plaza
    || datos.version !== 6 || typeof datos.confirmada_en !== "string"
    || !Number.isFinite(Date.parse(datos.confirmada_en))) throw new TypeError("recibo_derivacion_invalido");
  return Object.freeze({ ...datos });
}

export function crearClienteDerivacionHTTP({ ejecutar, validarOpciones } = {}) {
  if (typeof ejecutar !== "function" || typeof validarOpciones !== "function") throw new TypeError("cliente_derivacion_sin_transporte");
  return Object.freeze({
    derivarNombramiento(solicitud, opciones) {
      const entrada = validarSolicitudDerivacion(solicitud);
      const { signal } = validarOpciones(opciones);
      return ejecutar({ ruta: RUTA_DERIVACION_NOMBRAMIENTO, metodo: "POST", entrada, signal,
        estadoEsperado: [200, 201], maximoSolicitud: 4096, maximoRespuesta: 4096, efecto: true,
        validarRespuesta: (datos) => validarReciboDerivacion(datos, entrada),
        rechazoDeterminado: (error) => error?.envelopeValido === true && [400, 401, 403, 409, 422].includes(error.estado),
      });
    },
  });
}
