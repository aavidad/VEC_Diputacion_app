// Blancos de borde de un texto libre de Dietas. El conjunto es la unión de lo
// que recorta String.prototype.trim (\s, que incluye U+FEFF) y lo que recorta
// strings.TrimSpace de Go (además U+0085). Go rechaza un texto con cualquiera
// de ellos en un extremo (domain.TextoSinBordes); el cliente recorta y valida
// ese mismo conjunto, para que un motivo aceptado sea el que ve la titular.
const BORDES = /^[\s\u0085]+|[\s\u0085]+$/gu;

export const recortarBordes = (valor) => String(valor ?? "").replace(BORDES, "");
export const sinBordes = (valor) => typeof valor === "string" && recortarBordes(valor) === valor;
// Referencia opaca de Personal (centro o unidad): no vacía, acotada, sin
// blancos de borde ni caracteres de control.
export const textoRef = (valor, maximo) => typeof valor === "string" && valor.length >= 1 &&
  valor.length <= maximo && sinBordes(valor) && !/[\x00-\x1f\x7f]/u.test(valor);
