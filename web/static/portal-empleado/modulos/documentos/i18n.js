// `portal.js` importa este fichero de forma estática (fuera del grafo
// perezoso de módulos): la carga de `cargarTextos` se hace con `import()`
// para no incorporar `comun/textos.js` a su precarga estática, que es de
// `index.html` y no se toca en esta migración.
const { cargarTextos } = await import("../../../comun/textos.js");

export const MENSAJES_DOCUMENTOS = (await cargarTextos("documentos")).seccion("general");

export function crearTraductorDocumentos(mensajes = MENSAJES_DOCUMENTOS) {
  return (clave, variables = {}) => {
    const texto = mensajes[clave];
    if (typeof texto !== "string") return clave;
    return texto.replace(/\{([a-z_]+)\}/gu, (_coincidencia, nombre) => String(variables[nombre] ?? ""));
  };
}
