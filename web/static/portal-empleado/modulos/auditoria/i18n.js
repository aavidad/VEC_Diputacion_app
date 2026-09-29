// `portal.js` importa `vista.js` de forma estática y `vista.js` importa este
// fichero de forma estática también (fuera del grafo perezoso de módulos):
// la carga de `cargarTextos` se hace con `import()` para no incorporar
// `comun/textos.js` a la precarga estática de `index.html`, que no se toca
// en esta migración.
const { cargarTextos } = await import("../../../comun/textos.js");

/** Catálogo del módulo registrado bajo el i18n común del portal. */
export const MENSAJES_AUDITORIA = (await cargarTextos("auditoria")).seccion("general");

export function crearTraductorAuditoria(mensajes = MENSAJES_AUDITORIA) {
  return (clave, parametros = {}) => {
    if (!Object.hasOwn(mensajes, clave) || typeof mensajes[clave] !== "string") throw new RangeError(`mensaje de Auditoría no definido: ${clave}`);
    return mensajes[clave].replace(/\{([a-z_]+)\}/gu, (_todo, nombre) => String(parametros[nombre] ?? `{${nombre}}`));
  };
}
