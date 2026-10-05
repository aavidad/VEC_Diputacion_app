import { cargarTextos } from "../../comun/textos.js";

export const TEXTOS_COPIAS = await cargarTextos("copias-admin");
export const traducirCopias = (clave, variables) => TEXTOS_COPIAS.traducir("general." + clave, variables);
export function traducirCodigo(grupo, codigo) {
  return Object.hasOwn(TEXTOS_COPIAS.seccion(grupo), codigo)
    ? TEXTOS_COPIAS.traducir(grupo + "." + codigo)
    : traducirCopias("desconocido");
}
