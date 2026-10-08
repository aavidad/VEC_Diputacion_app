/** Enlaces compartibles de la lista autorizada de candidaturas de una bolsa. */
import { SITUACIONES_PARTICIPACION_BOLSA } from "./portal-bolsas-contrato.js?v=20261008-canal-telefono-v1";
import { origenLlamamientoValido } from "./portal-llamamiento-origen.js";

const CLAVE_BOLSA = "bolsa_ref";
const CLAVE_ESTADO = "estado";
// Seguimiento por teléfono de un llamamiento ya emitido.
const CLAVE_SEGUIMIENTO = "seguimiento";
// Origen de un llamamiento nuevo abierto desde una petición de personal.
const CLAVES_ORIGEN = Object.freeze({
  expediente_ref: "origen_expediente",
  referencia: "origen_referencia",
  centro: "origen_centro",
  fecha_inicio: "origen_inicio",
});
const HASH_CANDIDATOS = "#bolsa/bolsa-candidatos";
const HASH_RESUMEN = "#bolsa/resumen";

function parametrosDe(search) {
  if (typeof search !== "string" || search.length > 4_096) throw new TypeError("URL de Bolsa no válida");
  return new URLSearchParams(search);
}

function referenciaValida(referencia) {
  return typeof referencia === "string" && referencia.length > 0 && referencia.length <= 512
    && referencia === referencia.trim() && !referencia.includes("/")
    && !/[\u0000-\u001f\u007f-\u009f]/u.test(referencia);
}

function estadoValido(estado) {
  return estado === "" || SITUACIONES_PARTICIPACION_BOLSA.includes(estado);
}

function quitarFiltros(parametros) {
  for (const clave of [CLAVE_BOLSA, CLAVE_ESTADO, CLAVE_SEGUIMIENTO, "cursor", ...Object.values(CLAVES_ORIGEN)]) {
    parametros.delete(clave);
  }
}

export function rutaCandidatosBolsaCompartible(search, bolsaRef, estado = "", { seguimiento = "", origen = null } = {}) {
  if (!referenciaValida(bolsaRef) || !estadoValido(estado)) throw new TypeError("filtro de Bolsa no válido");
  if (seguimiento && (!referenciaValida(seguimiento) || estado || origen)) throw new TypeError("seguimiento de Bolsa no válido");
  const origenValido = origen ? origenLlamamientoValido(origen) : null;
  if (origen && (!origenValido || estado)) throw new TypeError("origen del llamamiento no válido");
  const parametros = parametrosDe(search);
  quitarFiltros(parametros);
  parametros.set(CLAVE_BOLSA, bolsaRef);
  if (estado) parametros.set(CLAVE_ESTADO, estado);
  if (seguimiento) parametros.set(CLAVE_SEGUIMIENTO, seguimiento);
  if (origenValido) {
    for (const [campo, clave] of Object.entries(CLAVES_ORIGEN)) {
      if (origenValido[campo]) parametros.set(clave, origenValido[campo]);
    }
  }
  return `?${parametros}${HASH_CANDIDATOS}`;
}

export function rutaResumenBolsasCompartible(search) {
  const parametros = parametrosDe(search);
  quitarFiltros(parametros);
  return `${parametros.size ? `?${parametros}` : ""}${HASH_RESUMEN}`;
}

function unico(parametros, clave) {
  const valores = parametros.getAll(clave);
  if (valores.length > 1) throw new TypeError("filtro de Bolsa duplicado o incompleto");
  return valores[0];
}

/** La URL selecciona un filtro, nunca acredita acceso a una bolsa. */
export function leerCandidatosBolsaCompartible(search, bolsasAutorizadas) {
  const parametros = parametrosDe(search);
  const bolsaRef = unico(parametros, CLAVE_BOLSA);
  const estado = unico(parametros, CLAVE_ESTADO) ?? "";
  const seguimiento = unico(parametros, CLAVE_SEGUIMIENTO) ?? "";
  const crudo = Object.fromEntries(Object.entries(CLAVES_ORIGEN).map(([campo, clave]) => [campo, unico(parametros, clave)]));
  const hayOrigen = Object.values(crudo).some((valor) => valor !== undefined);
  if (parametros.has("cursor") || ((estado || seguimiento || hayOrigen) && bolsaRef === undefined)
    || (seguimiento && (estado || hayOrigen)) || (hayOrigen && estado)) {
    throw new TypeError("filtro de Bolsa duplicado o incompleto");
  }
  if (bolsaRef === undefined) return null;
  if (!referenciaValida(bolsaRef) || !estadoValido(estado)
    || (seguimiento && !referenciaValida(seguimiento))
    || !Array.isArray(bolsasAutorizadas)
    || !bolsasAutorizadas.some((bolsa) => bolsa?.bolsa_ref === bolsaRef)) {
    throw new RangeError("bolsa o filtro no disponible para este ámbito");
  }
  // Un origen ilegible no impide abrir la bolsa: solo se descarta.
  const origen = hayOrigen ? origenLlamamientoValido(crudo) : null;
  return Object.freeze({ bolsaRef, estado, ...(seguimiento ? { seguimiento } : {}), ...(origen ? { origen } : {}) });
}
