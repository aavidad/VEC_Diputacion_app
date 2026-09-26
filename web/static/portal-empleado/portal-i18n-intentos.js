/** Textos del control de intentos de contacto de la ficha de Bolsa. */
export const MENSAJES_INTENTOS_ES = Object.freeze({
  titulo: "Intentos de contacto del llamamiento",
  sin_llamamiento: "Sin llamamiento en curso para esta participación.",
  cargando: "Cargando intentos de contacto…",
  error_carga: "No se pudo consultar el control de intentos.",
  reintentar: "Reintentar",
  sin_catalogo: "Sin reglas de intentos en el catálogo: se registra sin control.",
  historico_incompleto: "Histórico parcial: el recuento puede no incluir los contactos más antiguos.",
  estado_contactado: "Contactado",
  estado_baja_propuesta: "Baja propuesta",
  estado_proceso: "Proceso {proceso} de {procesos} · intento {intento} de {intentos}",
  dato_sin_contacto: "Intentos sin contacto",
  dato_valor_sin_contacto: "{sin} de {maximo}",
  dato_ultimo: "Último intento",
  dato_siguiente: "Siguiente intento desde",
  dato_franja: "Franja de llamadas",
  franja_habiles: "{valor}, días hábiles",
  sin_valor: "Sin registrar",
  aviso_antes_de_separacion: "Antes de la separación mínima",
  aviso_fuera_de_franja: "Fuera de la franja",
  aviso_dia_no_habil: "Día no hábil",
  reglas: "Reglas aplicadas",
  regla_ejemplo: "Regla de ejemplo",
  regla_reglamento: "Reglamento",
  formulario_intento: "Registrar intento telefónico",
  formulario_rebote: "Registrar correo no entregado",
  campo_resultado: "Resultado",
  campo_instante: "Fecha y hora",
  campo_anotacion: "Anotación",
  resultado_contactado: "Contactado",
  resultado_no_contesta: "No contesta",
  resultado_numero_erroneo: "Número erróneo",
  boton_intento: "Registrar intento",
  boton_rebote: "Registrar rebote",
  boton_enviando: "Registrando…",
  boton_proponer_baja: "Proponer baja",
  registrado: "Contacto registrado.",
  baja_propuesta_texto: "Agotados los procesos sin contacto. La baja la confirma RRHH con la operación de exclusión.",
  error_intento_antes_de_separacion: "No ha pasado la separación mínima desde el último intento.",
  error_intento_fuera_de_franja: "El intento cae fuera de la franja de llamadas.",
  error_intentos_agotados: "Los intentos del llamamiento están agotados.",
  error_acceso_denegado: "La sesión no tiene permiso para registrar contactos.",
  error_contacto_en_conflicto: "El contacto no es válido o la clave ya se usó con otros datos.",
  error_solicitud_invalida: "Revise los campos del formulario.",
  error_servicio: "No se pudo registrar el contacto. Puede reintentar.",
});

const CLAVES = Object.freeze(Object.keys(MENSAJES_INTENTOS_ES));
export function crearTraductorIntentos(catalogo = MENSAJES_INTENTOS_ES) {
  if (!catalogo || typeof catalogo !== "object" || CLAVES.some((clave) => typeof catalogo[clave] !== "string" || catalogo[clave] === "")) {
    throw new Error("catálogo i18n de intentos de contacto incompleto");
  }
  return (clave, variables = {}) => {
    if (!CLAVES.includes(clave)) throw new Error(`clave i18n de intentos desconocida: ${clave}`);
    return catalogo[clave].replace(/\{([a-z_]+)\}/g, (_c, variable) => String(variables[variable] ?? ""));
  };
}
export const traducirIntentos = crearTraductorIntentos();
