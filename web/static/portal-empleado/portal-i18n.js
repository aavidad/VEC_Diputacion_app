/** Catálogo común de los estados del shell y del acceso a Borradores. */
export const MENSAJES_PORTAL_ES = Object.freeze({
  acceso_borradores_disponible: "Borradores disponibles",
  acceso_borradores_denegado: "Sin permiso para gestionar borradores",
  acceso_borradores_error: "Servicio de borradores no disponible",
  acceso_borradores_cargando: "Comprobando acceso a borradores",
  error_capacidad_consultar_invalida: "La API no devolvió una capacidad de consulta válida.",
  error_capacidad_consultar_denegada: "La sesión no dispone de capacidad para consultar borradores.",
  error_sesion_borradores_denegada: "La sesión no dispone de acceso a los borradores.",
  error_servicio_borradores: "El servicio de borradores no está disponible.",
  anuncio_acceso_borradores_denegado: "Acceso a borradores no concedido",
  anuncio_servicio_borradores_error: "Servicio de borradores no disponible",
  anuncio_acceso_borradores_comprobado: "Acceso a borradores comprobado",
  anuncio_acceso_borradores_no_disponible: "El acceso a borradores continúa sin estar disponible",
  permiso_perfil_denegado: "Sin permiso para este perfil",
  estado_modulo_activo: "Activo",
  estado_modulo_comprobando: "Comprobando",
  estado_modulo_sin_permiso: "Sin permiso",
  estado_modulo_no_disponible: "No disponible",
  estado_modulo_no_habilitado: "No habilitado",
  estado_modulo_disponible_perfil: "Disponible para el perfil activo",
  accion_entrar: "Entrar",
  accion_reintentar: "Reintentar",
});

const CLAVES = Object.freeze(Object.keys(MENSAJES_PORTAL_ES));

export function crearTraductorPortal(catalogo = MENSAJES_PORTAL_ES) {
  if (!catalogo || typeof catalogo !== "object"
    || CLAVES.some((clave) => typeof catalogo[clave] !== "string" || catalogo[clave] === "")) {
    throw new Error("catálogo i18n del Portal del Empleado incompleto");
  }
  return (clave, variables = {}) => {
    if (!CLAVES.includes(clave)) throw new Error(`clave i18n del portal desconocida: ${clave}`);
    return catalogo[clave].replace(/\{([a-z_]+)\}/g,
      (_coincidencia, variable) => String(variables[variable] ?? ""));
  };
}

export const traducirPortal = crearTraductorPortal();
