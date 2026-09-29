import { cargarTextos } from "../../../comun/textos.js";

const MENSAJES_ADMINISTRACION = (await cargarTextos("administracion")).seccion("general");

export function crearTraductorAdministracion(mensajes = MENSAJES_ADMINISTRACION) {
  return (clave, variables = {}) => {
    if (!Object.hasOwn(mensajes, clave)) throw new Error("clave i18n de Administración ausente: " + clave);
    return mensajes[clave].replace(/\{([a-z_]+)\}/gu, (_, nombre) => String(variables[nombre] ?? ""));
  };
}

export const CLAVES_I18N_ADMINISTRACION = Object.freeze(Object.keys(MENSAJES_ADMINISTRACION));
