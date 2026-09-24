/** Mensajes de la comprobación pública. No guarda idioma ni referencias. */
export const MENSAJES_VERIFICAR_ES = Object.freeze({
  tiempo_espera: "El servicio tarda demasiado en responder. Puede volver a comprobar la referencia.",
});

export function traducirVerificar(clave) {
  if (!Object.hasOwn(MENSAJES_VERIFICAR_ES, clave)) throw new Error(`clave i18n de cotejo desconocida: ${clave}`);
  return MENSAJES_VERIFICAR_ES[clave];
}
