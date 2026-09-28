import { referenciaLlamamientoValida } from "./contrato-llamamiento.js";

export const RUTA_CONSULTA_COMUNICACIONES_EXPEDIENTE = "/api/vec/contratacion-temporal/expedientes/comunicaciones";
export const LIMITE_PAGINA_COMUNICACIONES = 10;
export const LIMITE_TOTAL_COMUNICACIONES = 100;
const CURSOR = /^[A-Za-z0-9][A-Za-z0-9._:/-]{2,94}#[0-9a-f]{64}$/u;
const INSTANTE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,6})?Z$/u;
const CAMPOS_FILA = Object.freeze(["organizacion_ref", "expediente_ref", "llamamiento_ref",
  "comunicacion_ref", "version", "estado", "registrada_en", "recibo_comunicacion_ref",
  "antecedente_tipo", "recibo_antecedente_ref"]);

function registro(valor, campos, opcionales = []) {
  if (valor === null || typeof valor !== "object" || Array.isArray(valor)
    || Object.getPrototypeOf(valor) !== Object.prototype
    || Object.getOwnPropertySymbols(valor).length !== 0
    || !campos.every((campo) => Object.hasOwn(valor, campo))
    || !Object.keys(valor).every((campo) => campos.includes(campo) || opcionales.includes(campo))
    || !Object.values(Object.getOwnPropertyDescriptors(valor)).every((d) => d.enumerable && Object.hasOwn(d, "value"))) {
    throw new TypeError("representación de comunicaciones no válida");
  }
  return valor;
}

function instante(valor) {
  return typeof valor === "string" && INSTANTE.test(valor)
    && !valor.startsWith("0000-")
    && Number.isFinite(Date.parse(valor))
    && new Date(valor).toISOString().slice(0, 19) === valor.slice(0, 19);
}
export function instanteOrdenComunicacion(valor) {
  if (!instante(valor)) throw new TypeError("fecha de comunicación no válida");
  return `${valor.slice(0, 19)}.${(valor.match(/\.(\d+)Z$/u)?.[1] ?? "").padEnd(6, "0")}Z`;
}

export function validarConsultaComunicacionesExpediente(entrada) {
  const valor = registro(entrada, ["expediente_ref"], ["cursor"]);
  if (!referenciaLlamamientoValida(valor.expediente_ref)
    || (valor.cursor !== undefined && (typeof valor.cursor !== "string" || !CURSOR.test(valor.cursor)))) {
    throw new TypeError("consulta de comunicaciones no válida");
  }
  return Object.freeze({ expediente_ref: valor.expediente_ref, ...(valor.cursor ? { cursor: valor.cursor } : {}) });
}

export function validarPaginaComunicacionesExpediente(entrada, consulta) {
  const esperada = validarConsultaComunicacionesExpediente(consulta);
  const pagina = registro(entrada, ["expediente_ref", "comunicaciones"], ["siguiente_cursor"]);
  if (pagina.expediente_ref !== esperada.expediente_ref || !Array.isArray(pagina.comunicaciones)
    || pagina.comunicaciones.length > LIMITE_PAGINA_COMUNICACIONES
    || !Array.from({ length: pagina.comunicaciones.length }, (_, i) => Object.hasOwn(pagina.comunicaciones, i))
      .every(Boolean)) throw new TypeError("página de comunicaciones no válida");
  let anterior = null, organizacion = null;
  const filas = pagina.comunicaciones.map((entradaFila) => {
    const fila = registro(entradaFila, CAMPOS_FILA);
    if (!["organizacion_ref", "llamamiento_ref", "comunicacion_ref", "recibo_comunicacion_ref", "recibo_antecedente_ref"]
      .every((campo) => referenciaLlamamientoValida(fila[campo]))
      || fila.expediente_ref !== esperada.expediente_ref || fila.version !== 2
      || fila.estado !== "registrada_localmente" || !instante(fila.registrada_en)
      || !["seleccion_confirmada", "continuacion_confirmada"].includes(fila.antecedente_tipo)
      || (organizacion !== null && organizacion !== fila.organizacion_ref)
      || (anterior && (instanteOrdenComunicacion(fila.registrada_en) < instanteOrdenComunicacion(anterior.registrada_en)
        || (instanteOrdenComunicacion(fila.registrada_en) === instanteOrdenComunicacion(anterior.registrada_en)
          && fila.comunicacion_ref <= anterior.comunicacion_ref)))) {
      throw new TypeError("fila de comunicación no válida");
    }
    organizacion = fila.organizacion_ref;
    anterior = fila;
    return Object.freeze(Object.fromEntries(CAMPOS_FILA.map((campo) => [campo, fila[campo]])));
  });
  const cursor = pagina.siguiente_cursor;
  if (cursor !== undefined && (typeof cursor !== "string" || !CURSOR.test(cursor)
    || filas.length !== LIMITE_PAGINA_COMUNICACIONES
    || !cursor.startsWith(`${filas.at(-1).comunicacion_ref}#`))) {
    throw new TypeError("cursor de comunicaciones no válido");
  }
  return Object.freeze({ expediente_ref: esperada.expediente_ref, comunicaciones: Object.freeze(filas),
    ...(cursor ? { siguiente_cursor: cursor } : {}) });
}

export function crearConsultaComunicacionesExpedienteClienteHTTP({ ejecutar, validarOpciones } = {}) {
  if (typeof ejecutar !== "function" || typeof validarOpciones !== "function") {
    throw new TypeError("dependencias HTTP de comunicaciones no disponibles");
  }
  return Object.freeze({
    consultarComunicacionesExpediente(consulta, opciones) {
      const entrada = validarConsultaComunicacionesExpediente(consulta);
      const { signal } = validarOpciones(opciones);
      const query = new URLSearchParams({ expediente_ref: entrada.expediente_ref,
        limite: String(LIMITE_PAGINA_COMUNICACIONES), ...(entrada.cursor ? { cursor: entrada.cursor } : {}) });
      return ejecutar({ metodo: "GET", ruta: `${RUTA_CONSULTA_COMUNICACIONES_EXPEDIENTE}?${query}`,
        signal, estadoEsperado: 200, maximoRespuesta: 16 * 1024, efecto: false,
        validarRespuesta: (respuesta) => validarPaginaComunicacionesExpediente(respuesta, entrada) });
    },
  });
}
