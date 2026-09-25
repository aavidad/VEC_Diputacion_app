/** Textos del correo personalizado del nuevo llamamiento (B7). */
export const MENSAJES_CORREO_LLAMAMIENTO_ES = Object.freeze({
  marcadores_grupo: "Datos de cada persona que se pueden insertar",
  marcador_insertar: "Insertar el dato {marcador} en el texto",
  vista_previa_titulo: "Vista previa del correo",
  vista_previa_destinatario: "Destinatario",
  vista_previa_persona: "Persona {numero}",
  vista_previa_ver: "Ver vista previa",
  vista_previa_cargando: "Preparando la vista previa…",
  vista_previa_asunto: "Asunto",
  vista_previa_cuerpo: "Texto",
  vista_previa_longitud: "{caracteres} de {limite} caracteres",
  error_plantilla_invalida: "El texto usa un dato que no existe. Revise los datos entre llaves.",
  error_datos_incompletos: "Falta algún dato de esta persona para completar el correo.",
  error_correo_excede_limite: "Con los datos de esta persona el correo supera el tamaño máximo. Acorte el texto.",
  error_acceso_denegado: "La sesión no dispone de permiso para ver este correo.",
  error_servicio: "La vista previa no está disponible ahora. Puede reintentarlo.",
});

const CLAVES = Object.freeze(Object.keys(MENSAJES_CORREO_LLAMAMIENTO_ES));

export function crearTraductorCorreoLlamamiento(catalogo = MENSAJES_CORREO_LLAMAMIENTO_ES) {
  if (!catalogo || typeof catalogo !== "object" || CLAVES.some((clave) => typeof catalogo[clave] !== "string" || !catalogo[clave])) {
    throw new TypeError("catálogo i18n del correo de llamamiento incompleto");
  }
  return (clave, variables = {}) => {
    if (!CLAVES.includes(clave)) throw new TypeError(`clave i18n del correo de llamamiento desconocida: ${clave}`);
    return catalogo[clave].replace(/\{([a-z_]+)\}/g, (_c, variable) => String(variables[variable] ?? ""));
  };
}

export const traducirCorreoLlamamiento = crearTraductorCorreoLlamamiento();
