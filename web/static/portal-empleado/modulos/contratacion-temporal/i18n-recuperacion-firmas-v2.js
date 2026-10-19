const CLAVES = Object.freeze([
  "demo_titulo", "demo_aviso", "titulo", "consultar",
  "cancelar", "cargando", "vacio", "cancelada", "denegada", "no_disponible",
  "no_confiable", "firma", "firmado", "devuelto", "recibo_con_valor", "fecha",
  "historicas_titulo", "historicas_consultar", "historicas_consultar_firma",
  "historicas_paso", "historicas_version", "historicas_documento",
  "historicas_documento_informe_definitivo", "historicas_documento_resolucion",
  "historicas_documento_diligencia", "historicas_documento_toma_posesion",
  "historicas_documento_notificacion", "historicas_documento_comunicacion_centro",
]);

// Recibe la sección completa del catálogo común ya cargado. No decide idioma
// ni sustituye mensajes ausentes por textos dentro del código.
export function crearTraductorRecuperacionFirmasV2(mensajes) {
  if (mensajes === null || typeof mensajes !== "object" || Array.isArray(mensajes)
    || Object.keys(mensajes).length !== CLAVES.length
    || CLAVES.some((clave) => typeof mensajes[clave] !== "string" || mensajes[clave].trim() === "")) {
    throw new TypeError("catalogo_recuperacion_firmas_invalido");
  }
  return (clave, variables = {}) => {
    if (!Object.hasOwn(mensajes, clave)) throw new TypeError("clave_recuperacion_firmas_desconocida");
    return mensajes[clave].replace(/\{([a-z_]+)\}/gu, (_texto, nombre) => String(variables[nombre] ?? ""));
  };
}
