import { referenciaLlamamientoValida } from "./contrato-llamamiento.js";

export const RUTA_CONSULTA_RECIBO_RESPUESTA = "/api/vec/contratacion-temporal/llamamientos/respuestas/recibo";
const CAMPOS = Object.freeze(["esquema", "organizacion_ref", "expediente_ref", "comunicacion_ref", "respuesta",
  "justificante_ref", "recibo_ref", "auditoria_ref", "registrada_en", "estado"]);
const INSTANTE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,6})?Z$/u;

export function validarConsultaReciboRespuesta(entrada) {
  if (entrada === null || typeof entrada !== "object" || Array.isArray(entrada)
    || Object.getPrototypeOf(entrada) !== Object.prototype
    || Object.getOwnPropertySymbols(entrada).length !== 0
    || Object.keys(entrada).length !== 3
    || !["organizacion_ref", "expediente_ref", "comunicacion_ref"].every((campo) => {
      const descriptor = Object.getOwnPropertyDescriptor(entrada, campo);
      return descriptor?.enumerable === true && Object.hasOwn(descriptor, "value")
        && referenciaLlamamientoValida(descriptor.value);
    })) throw new TypeError("consulta de recibo de respuesta no válida");
  return Object.freeze({ organizacion_ref: entrada.organizacion_ref, expediente_ref: entrada.expediente_ref,
    comunicacion_ref: entrada.comunicacion_ref });
}

export function validarReciboRespuestaConsultado(entrada, consulta) {
  const esperada = validarConsultaReciboRespuesta(consulta);
  if (entrada === null || typeof entrada !== "object" || Array.isArray(entrada)
    || Object.getPrototypeOf(entrada) !== Object.prototype
    || Object.getOwnPropertySymbols(entrada).length !== 0
    || Object.keys(entrada).length !== CAMPOS.length
    || !CAMPOS.every((campo) => {
      const descriptor = Object.getOwnPropertyDescriptor(entrada, campo);
      return descriptor?.enumerable === true && Object.hasOwn(descriptor, "value");
    })
    || entrada.esquema !== "vec.contratacion-temporal.recibo-respuesta-llamamiento.v1"
    || entrada.organizacion_ref !== esperada.organizacion_ref
    || entrada.expediente_ref !== esperada.expediente_ref
    || entrada.comunicacion_ref !== esperada.comunicacion_ref
    || !["aceptacion", "renuncia"].includes(entrada.respuesta)
    || !["justificante_ref", "recibo_ref", "auditoria_ref"].every(
      (campo) => referenciaLlamamientoValida(entrada[campo]))
    || entrada.estado !== "registrada_por_rrhh"
    || typeof entrada.registrada_en !== "string" || !INSTANTE.test(entrada.registrada_en)
    || !Number.isFinite(Date.parse(entrada.registrada_en))
    || new Date(entrada.registrada_en).toISOString().slice(0, 19) !== entrada.registrada_en.slice(0, 19)) {
    throw new TypeError("recibo de respuesta consultado no válido");
  }
  return Object.freeze(Object.fromEntries(CAMPOS.map((campo) => [campo, entrada[campo]])));
}

export function crearConsultaReciboRespuestaClienteHTTP({ ejecutar, validarOpciones } = {}) {
  if (typeof ejecutar !== "function" || typeof validarOpciones !== "function") {
    throw new TypeError("dependencias HTTP de consulta de respuesta no disponibles");
  }
  return Object.freeze({
    consultarReciboRespuesta(consulta, opciones) {
      const entrada = validarConsultaReciboRespuesta(consulta);
      const { signal } = validarOpciones(opciones);
      const query = new URLSearchParams(entrada);
      if (query.toString().length > 2048) throw new TypeError("consulta de recibo demasiado larga");
      return ejecutar({ metodo: "GET", ruta: `${RUTA_CONSULTA_RECIBO_RESPUESTA}?${query}`,
        signal, estadoEsperado: 200, maximoRespuesta: 4096, efecto: false,
        validarRespuesta: (respuesta) => validarReciboRespuestaConsultado(respuesta, entrada) });
    },
  });
}
