/** Lógica de composición y cercado de cliente para Análisis en contratación temporal. */

import { validarReciboAnalisis } from "./contrato-analisis.js";

export const CAMPOS_COMPOSICION_ANALISIS = Object.freeze(["cliente", "catalogos", "contexto", "analisisInicial", "rectificacion"]);
export const CAMPOS_COMPOSICION_ANALISIS_OBLIGATORIOS = Object.freeze(["cliente", "catalogos", "contexto", "analisisInicial"]);
export const CAMPOS_CONTEXTO_ANALISIS = Object.freeze(["operacion", "artefacto_ref"]);
export const CAMPOS_RECTIFICACION_ANALISIS = Object.freeze(["operacion", "artefacto_ref", "analisisInicial"]);
export const PATRON_REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;

export function descriptoresCerrados(entrada, campos, nombre, obligatorios = campos) {
  if (entrada === null || typeof entrada !== "object" || Array.isArray(entrada)
    || Object.getPrototypeOf(entrada) !== Object.prototype) {
    throw new TypeError(`${nombre} no válida`);
  }
  const descriptores = Object.getOwnPropertyDescriptors(entrada);
  const claves = Object.keys(descriptores);
  if (Object.getOwnPropertySymbols(entrada).length !== 0
    || claves.some((clave) => !campos.includes(clave))
    || claves.some((clave) => !Object.hasOwn(descriptores[clave], "value")
      || descriptores[clave].enumerable !== true)) {
    throw new TypeError(`${nombre} no válida`);
  }
  if (obligatorios.some((campo) => !Object.hasOwn(descriptores, campo))) return null;
  return descriptores;
}

export function prepararComposicionAnalisis(entrada) {
  if (entrada === null || entrada === undefined) return null;
  const descriptores = descriptoresCerrados(
    entrada,
    CAMPOS_COMPOSICION_ANALISIS,
    "composición del análisis",
    CAMPOS_COMPOSICION_ANALISIS_OBLIGATORIOS,
  );
  if (descriptores === null) return null;
  const contextoEntrada = descriptores.contexto.value;
  if (contextoEntrada === null || typeof contextoEntrada !== "object"
    || Array.isArray(contextoEntrada)) return null;
  const contextoDescriptores = descriptoresCerrados(
    contextoEntrada,
    CAMPOS_CONTEXTO_ANALISIS,
    "contexto de composición del análisis",
  );
  if (contextoDescriptores === null) return null;
  const operacion = contextoDescriptores.operacion.value;
  const artefactoRef = contextoDescriptores.artefacto_ref.value;
  const cliente = descriptores.cliente.value;
  const catalogos = descriptores.catalogos.value;
  let rectificacion = null;
  if (Object.hasOwn(descriptores, "rectificacion")) {
    const entradaRectificacion = descriptores.rectificacion.value;
    const descriptoresRectificacion = descriptoresCerrados(
      entradaRectificacion,
      CAMPOS_RECTIFICACION_ANALISIS,
      "composición de rectificación del análisis",
    );
    if (descriptoresRectificacion === null
      || descriptoresRectificacion.operacion.value !== "rectificar"
      || descriptoresRectificacion.artefacto_ref.value !== artefactoRef
      || descriptoresRectificacion.analisisInicial.value !== null) return null;
    let metodoRectificacion;
    try { metodoRectificacion = cliente?.rectificarAnalisis; } catch { return null; }
    if (typeof metodoRectificacion !== "function") return null;
    rectificacion = Object.freeze({
      operacion: "rectificar",
      artefacto_ref: artefactoRef,
      analisisInicial: null,
      metodoCliente: metodoRectificacion,
    });
  }
  const metodo = operacion === "rectificar" ? "rectificarAnalisis" : "registrarAnalisis";
  let metodoCliente;
  try { metodoCliente = cliente?.[metodo]; } catch { return null; }
  if (!(["registrar", "rectificar"].includes(operacion))
    || typeof artefactoRef !== "string" || !PATRON_REFERENCIA.test(artefactoRef)
    || cliente === null || typeof cliente !== "object" || typeof metodoCliente !== "function"
    || catalogos === null || typeof catalogos !== "object" || Array.isArray(catalogos)) {
    return null;
  }
  return Object.freeze({
    cliente,
    metodoCliente,
    nombreMetodo: metodo,
    catalogos,
    contexto: Object.freeze({ operacion, artefacto_ref: artefactoRef }),
    analisisInicial: descriptores.analisisInicial.value,
    rectificacion,
  });
}

export function errorIndeterminadoAnalisis() {
  const error = new Error("resultado de análisis indeterminado");
  Object.defineProperties(error, {
    codigo: { value: "resultado_indeterminado", enumerable: true },
    resultadoIndeterminado: { value: true, enumerable: true },
  });
  return error;
}

export function clasificarErrorAnalisis(error, signal) {
  let indeterminado;
  let abortado = false;
  try {
    indeterminado = error?.resultadoIndeterminado;
    abortado = signal?.aborted === true;
  } catch {
    return Object.freeze({ error: errorIndeterminadoAnalisis(), etapa: "indeterminado" });
  }
  if (indeterminado === false && !abortado) {
    return Object.freeze({ error, etapa: "reintentable" });
  }
  return Object.freeze({
    error: indeterminado === true && !abortado ? error : errorIndeterminadoAnalisis(),
    etapa: "indeterminado",
  });
}

export function crearClienteAnalisisCercado(
  composicion,
  contexto,
  cambiarEtapa,
  alConfirmar,
  alErrorConfirmado = () => {},
) {
  const nombreMetodo = contexto.operacion === "rectificar"
    ? "rectificarAnalisis" : "registrarAnalisis";
  const metodoCliente = contexto.operacion === composicion.contexto.operacion
    ? composicion.metodoCliente
    : composicion.rectificacion?.metodoCliente;
  if (typeof metodoCliente !== "function") {
    throw new TypeError("método de análisis no disponible");
  }
  const invocar = function invocarAnalisis(solicitud, opciones) {
    const vuelo = Object.freeze({});
    cambiarEtapa("transmitiendo", vuelo);
    let resultado;
    try {
      resultado = Reflect.apply(
        metodoCliente,
        composicion.cliente,
        [solicitud, opciones],
      );
    } catch (error) {
      const clasificado = clasificarErrorAnalisis(error, opciones?.signal);
      cambiarEtapa(clasificado.etapa, vuelo);
      throw clasificado.error;
    }
    return Promise.resolve(resultado).then((respuesta) => {
      let recibo;
      try {
        recibo = validarReciboAnalisis(respuesta);
        if (recibo.operacion !== contexto.operacion
          || recibo.expediente_ref !== contexto.expediente_ref
          || recibo.version_resultante !== contexto.version_esperada + 1) {
          throw new TypeError("recibo no ligado");
        }
      } catch {
        cambiarEtapa("indeterminado", vuelo);
        throw errorIndeterminadoAnalisis();
      }
      cambiarEtapa("confirmado", vuelo);
      try { alConfirmar(recibo); } catch {
        // El recibo de Análisis prevalece si el siguiente paso no puede montarse.
        alErrorConfirmado(recibo);
      }
      return recibo;
    }, (error) => {
      const clasificado = clasificarErrorAnalisis(error, opciones?.signal);
      cambiarEtapa(clasificado.etapa, vuelo);
      throw clasificado.error;
    });
  };
  return Object.freeze({ [nombreMetodo]: invocar });
}
