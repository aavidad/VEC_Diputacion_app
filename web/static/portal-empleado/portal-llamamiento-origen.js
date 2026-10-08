/**
 * Origen de un llamamiento de Bolsa abierto desde una petición de
 * Contratación temporal. Viaja por la URL, así que se trata como entrada no
 * fiable: solo se aceptan referencias y textos cortos con caracteres simples.
 * Una referencia o un número inválidos anulan el origen; un centro o una fecha
 * inválidos solo se descartan.
 */
const PATRON_EXPEDIENTE = /^[A-Za-z0-9][A-Za-z0-9:._-]{0,159}$/u;
const PATRON_TEXTO = /^[\p{L}\p{N}][\p{L}\p{M}\p{N} .,:;()/ºª'’"«»&+_-]*$/u;
const PATRON_FECHA = /^(\d{4})-(\d{2})-(\d{2})$/u;
export const LIMITE_REFERENCIA_ORIGEN = 160;
export const LIMITE_CENTRO_ORIGEN = 200;

function textoValido(valor, maximo) {
  return typeof valor === "string" && valor.length >= 2 && valor.length <= maximo
    && valor === valor.trim() && PATRON_TEXTO.test(valor);
}

function fechaValida(valor) {
  const partes = typeof valor === "string" ? PATRON_FECHA.exec(valor) : null;
  if (!partes) return false;
  const [anio, mes, dia] = partes.slice(1).map(Number);
  const fecha = new Date(Date.UTC(anio, mes - 1, dia));
  return anio >= 2000 && fecha.getUTCFullYear() === anio && fecha.getUTCMonth() === mes - 1 && fecha.getUTCDate() === dia;
}

/** Devuelve el origen normalizado y congelado, o null si no es utilizable. */
export function origenLlamamientoValido(origen) {
  if (!origen || typeof origen !== "object") return null;
  const { expediente_ref: expediente, referencia, centro, fecha_inicio: inicio } = origen;
  if (typeof expediente !== "string" || !PATRON_EXPEDIENTE.test(expediente)
    || !textoValido(referencia, LIMITE_REFERENCIA_ORIGEN)) return null;
  return Object.freeze({
    expediente_ref: expediente,
    referencia,
    ...(textoValido(centro, LIMITE_CENTRO_ORIGEN) ? { centro } : {}),
    ...(fechaValida(inicio) ? { fecha_inicio: inicio } : {}),
  });
}
