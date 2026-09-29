import { cargarTextos } from "../comun/textos.js";

/** Textos de convocatorias, bases y calendario: `textos/<idioma>/portal-bolsa.json`. */
export const MENSAJES_CONVOCATORIAS_S1 = (await cargarTextos("portal-bolsa")).seccion("convocatorias");

const CLAVES = Object.freeze(Object.keys(MENSAJES_CONVOCATORIAS_S1));

export function crearTraductorConvocatoriasS1(catalogo = MENSAJES_CONVOCATORIAS_S1) {
  if (!catalogo || CLAVES.some((clave) => typeof catalogo[clave] !== "string" || catalogo[clave] === "")) {
    throw new Error("Catálogo de convocatorias S1 incompleto");
  }
  return (clave, variables = {}) => {
    if (!CLAVES.includes(clave)) throw new Error(`Clave de convocatorias S1 desconocida: ${clave}`);
    return catalogo[clave].replace(/\{([a-z_]+)\}/g, (_coincidencia, variable) => String(variables[variable] ?? ""));
  };
}

export const traducirConvocatoriasS1 = crearTraductorConvocatoriasS1();
