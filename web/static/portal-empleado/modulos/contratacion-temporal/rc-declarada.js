/**
 * Retención de crédito declarada por el centro (regla provisional, duda 125).
 *
 * El documento RC lo expide Intervención en SICAL; el centro no lo custodia.
 * El centro solo escribe el número de operación de SICAL y la petición lo
 * envía como `documento_ref = "rc-declarado:<número>"`. Es un dato declarado
 * que comprueban RRHH o Intervención: nunca acredita el crédito.
 */
export const PREFIJO_RC_DECLARADA = "rc-declarado:";

// Con el prefijo, la referencia cumple el patrón del dominio (máximo 160).
export const MAXIMO_OPERACION_RC = 160 - PREFIJO_RC_DECLARADA.length;
const OPERACION = new RegExp(`^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,${MAXIMO_OPERACION_RC - 1}}$`, "u");
// Espejo de patronDocumentoIdentidadEnReferencia y patronEtiquetaDocumentoIdentidad
// del dominio de Bolsa: un DNI o NIE nunca viaja como número de operación.
const DOCUMENTO_IDENTIDAD = /((?:[0-9][._:/#-]?){8}|[XYZ][._:/#-]?(?:[0-9][._:/#-]?){7})[A-Z]/iu;
const ETIQUETA_DOCUMENTO_IDENTIDAD = /(^|[._:/#-])(dni|nie|nif|pasaporte|passport)([._:/#-]|$)/iu;

/** Indica si el número de operación de SICAL tecleado es admisible. */
export function operacionRCDeclaradaValida(valor) {
  return typeof valor === "string" && OPERACION.test(valor)
    && !DOCUMENTO_IDENTIDAD.test(valor) && !ETIQUETA_DOCUMENTO_IDENTIDAD.test(valor);
}

/** Referencia que envía la petición del centro para una RC declarada. */
export function referenciaRCDeclarada(numero) {
  return `${PREFIJO_RC_DECLARADA}${numero}`;
}

/** Indica si la RC de una petición procede de la declaración del centro. */
export function esRCDeclarada(rc) {
  return Boolean(rc?.existe) && typeof rc.documento_ref === "string"
    && rc.documento_ref.startsWith(PREFIJO_RC_DECLARADA);
}
