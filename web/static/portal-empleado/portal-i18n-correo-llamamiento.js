import { cargarTextos } from "../comun/textos.js";

/** Textos del correo personalizado del nuevo llamamiento (B7): `textos/<idioma>/portal-bolsa.json`. */
export const MENSAJES_CORREO_LLAMAMIENTO = (await cargarTextos("portal-bolsa")).seccion("correo_llamamiento");

const CLAVES = Object.freeze(Object.keys(MENSAJES_CORREO_LLAMAMIENTO));

export function crearTraductorCorreoLlamamiento(catalogo = MENSAJES_CORREO_LLAMAMIENTO) {
  if (!catalogo || typeof catalogo !== "object" || CLAVES.some((clave) => typeof catalogo[clave] !== "string" || !catalogo[clave])) {
    throw new TypeError("catálogo i18n del correo de llamamiento incompleto");
  }
  return (clave, variables = {}) => {
    if (!CLAVES.includes(clave)) throw new TypeError(`clave i18n del correo de llamamiento desconocida: ${clave}`);
    return catalogo[clave].replace(/\{([a-z_]+)\}/g, (_c, variable) => String(variables[variable] ?? ""));
  };
}

export const traducirCorreoLlamamiento = crearTraductorCorreoLlamamiento();
