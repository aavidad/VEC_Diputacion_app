
export let MENSAJES_CONSULTA_PERMISOS_CRONOS;
export function instalarMENSAJES_CONSULTA_PERMISOS_CRONOS(mensajes) { MENSAJES_CONSULTA_PERMISOS_CRONOS = mensajes; }

export function crearTraductorConsultaPermisosCronos(mensajes = MENSAJES_CONSULTA_PERMISOS_CRONOS) {
  if (!MENSAJES_CONSULTA_PERMISOS_CRONOS) throw new TypeError("catálogo de consulta de permisos Cronos sin preparar");
  const claves = Object.keys(MENSAJES_CONSULTA_PERMISOS_CRONOS);
  if (!mensajes || claves.some((clave) => typeof mensajes[clave] !== "string" || !mensajes[clave])) {
    throw new TypeError("catálogo de consulta de permisos Cronos incompleto");
  }
  return (clave) => {
    if (!claves.includes(clave)) throw new TypeError("clave de consulta de permisos Cronos desconocida");
    return mensajes[clave];
  };
}
