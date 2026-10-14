import { cargarTextos } from "../../../comun/textos.js";

export const MENSAJES_ORIGEN_SALDO = (await cargarTextos("cronos-saldo-origen")).seccion("saldo");

export function crearTraductorOrigenSaldo(mensajes = MENSAJES_ORIGEN_SALDO) {
  const claves = Object.keys(MENSAJES_ORIGEN_SALDO);
  if (claves.some((clave) => typeof mensajes?.[clave] !== "string" || !mensajes[clave].trim())) {
    throw new TypeError("catálogo de origen de saldo incompleto");
  }
  return (clave) => {
    if (!claves.includes(clave)) throw new TypeError("clave de origen de saldo desconocida");
    return mensajes[clave];
  };
}
