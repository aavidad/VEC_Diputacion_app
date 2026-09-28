/** Textos del correo personalizado del nuevo llamamiento (B7). */
import { IDIOMA_ACTUAL } from "../comun/idioma.js";
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

export const MENSAJES_CORREO_LLAMAMIENTO_EN = Object.freeze({
  marcadores_grupo: "Details for each person that can be inserted",
  marcador_insertar: "Insert {marcador} in the message",
  vista_previa_titulo: "Email preview",
  vista_previa_destinatario: "Recipient",
  vista_previa_persona: "Person {numero}",
  vista_previa_ver: "Preview",
  vista_previa_cargando: "Preparing preview…",
  vista_previa_asunto: "Subject",
  vista_previa_cuerpo: "Message",
  vista_previa_longitud: "{caracteres} of {limite} characters",
  error_plantilla_invalida: "The message uses a field that does not exist. Check the fields in braces.",
  error_datos_incompletos: "Some details for this person are missing, so the email cannot be completed.",
  error_correo_excede_limite: "With this person's details, the email exceeds the maximum length. Shorten the message.",
  error_acceso_denegado: "This session does not have permission to view this email.",
  error_servicio: "The preview is currently unavailable. You can try again.",
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

export const traducirCorreoLlamamiento = crearTraductorCorreoLlamamiento(
  IDIOMA_ACTUAL === "en" ? MENSAJES_CORREO_LLAMAMIENTO_EN : MENSAJES_CORREO_LLAMAMIENTO_ES,
);
