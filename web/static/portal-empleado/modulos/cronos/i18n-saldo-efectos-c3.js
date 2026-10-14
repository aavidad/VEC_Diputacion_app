import { cargarTextos } from "../../../comun/textos.js";

export const MENSAJES_SALDO_EFECTOS_C3 = (await cargarTextos("cronos-saldo-efectos-c3")).seccion("saldo");

export function crearTraductorSaldoEfectosC3(mensajes = MENSAJES_SALDO_EFECTOS_C3) {
  const claves = Object.keys(MENSAJES_SALDO_EFECTOS_C3);
  if (claves.some((clave) => typeof mensajes?.[clave] !== "string" || !mensajes[clave].trim())) {
    throw new TypeError("catálogo de efectos del saldo incompleto");
  }
  return (clave) => {
    if (!claves.includes(clave)) throw new TypeError("clave de efecto del saldo desconocida");
    return mensajes[clave];
  };
}
