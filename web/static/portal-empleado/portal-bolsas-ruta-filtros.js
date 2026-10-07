/** Enlaces compartibles de la lista autorizada de candidaturas de una bolsa. */
import { SITUACIONES_PARTICIPACION_BOLSA } from "./portal-bolsas-contrato.js?v=20261007-pantallas-textos-final-v1";

const CLAVE_BOLSA = "bolsa_ref";
const CLAVE_ESTADO = "estado";
const HASH_CANDIDATOS = "#bolsa/bolsa-candidatos";
const HASH_RESUMEN = "#bolsa/resumen";

function parametrosDe(search) {
  if (typeof search !== "string" || search.length > 4_096) throw new TypeError("URL de Bolsa no válida");
  return new URLSearchParams(search);
}

function bolsaRefValida(referencia) {
  return typeof referencia === "string" && referencia.length > 0 && referencia.length <= 512
    && referencia === referencia.trim() && !referencia.includes("/")
    && !/[\u0000-\u001f\u007f-\u009f]/u.test(referencia);
}

function estadoValido(estado) {
  return estado === "" || SITUACIONES_PARTICIPACION_BOLSA.includes(estado);
}

export function rutaCandidatosBolsaCompartible(search, bolsaRef, estado = "") {
  if (!bolsaRefValida(bolsaRef) || !estadoValido(estado)) throw new TypeError("filtro de Bolsa no válido");
  const parametros = parametrosDe(search);
  parametros.delete(CLAVE_BOLSA);
  parametros.delete(CLAVE_ESTADO);
  parametros.set(CLAVE_BOLSA, bolsaRef);
  if (estado) parametros.set(CLAVE_ESTADO, estado);
  return `?${parametros}${HASH_CANDIDATOS}`;
}

export function rutaResumenBolsasCompartible(search) {
  const parametros = parametrosDe(search);
  parametros.delete(CLAVE_BOLSA);
  parametros.delete(CLAVE_ESTADO);
  return `${parametros.size ? `?${parametros}` : ""}${HASH_RESUMEN}`;
}

/** La URL selecciona un filtro, nunca acredita acceso a una bolsa. */
export function leerCandidatosBolsaCompartible(search, bolsasAutorizadas) {
  const parametros = parametrosDe(search);
  const referencias = parametros.getAll(CLAVE_BOLSA);
  const estados = parametros.getAll(CLAVE_ESTADO);
  if (referencias.length > 1 || estados.length > 1 || (estados.length && !referencias.length)) {
    throw new TypeError("filtro de Bolsa duplicado o incompleto");
  }
  if (!referencias.length) return null;
  const bolsaRef = referencias[0];
  const estado = estados[0] ?? "";
  if (!bolsaRefValida(bolsaRef) || !estadoValido(estado)
    || !Array.isArray(bolsasAutorizadas)
    || !bolsasAutorizadas.some((bolsa) => bolsa?.bolsa_ref === bolsaRef)) {
    throw new RangeError("bolsa o filtro no disponible para este ámbito");
  }
  return Object.freeze({ bolsaRef, estado });
}
