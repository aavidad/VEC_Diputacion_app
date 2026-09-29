import { cargarTextos } from "../comun/textos.js";

/** Textos de contratos, ceses y reincorporaciones: `textos/<idioma>/portal-bolsa.json`. */
export const MENSAJES_CONTRATOS = (await cargarTextos("portal-bolsa")).seccion("contratos");

export function crearTraductorContratos(catalogo = MENSAJES_CONTRATOS) {
  const claves = Object.keys(MENSAJES_CONTRATOS);
  if (!catalogo || typeof catalogo !== "object" || claves.some((clave) => typeof catalogo[clave] !== "string" || !catalogo[clave])) {
    throw new TypeError("catálogo i18n de contratos incompleto");
  }
  return (clave) => {
    if (!claves.includes(clave)) throw new TypeError(`clave i18n de contratos desconocida: ${clave}`);
    return catalogo[clave];
  };
}

export const traducirContratos = crearTraductorContratos();
