// Datos de «Mi bolsa» que el catálogo de Bolsa (regla b28.campos_portal)
// puede mostrar u ocultar. El servidor ya retira los ocultos; la web solo
// deja de pintar su apartado. Sin lista en la respuesta se muestran todos.
export const CAMPOS_MI_BOLSA = Object.freeze([
  "bolsa", "posicion", "estado", "ultimo_llamamiento", "contratos", "fecha_disponible",
]);

export function validarCamposMiBolsa(lista) {
  if (lista === undefined) return [...CAMPOS_MI_BOLSA];
  if (!Array.isArray(lista) || lista.length > CAMPOS_MI_BOLSA.length || new Set(lista).size !== lista.length ||
      !lista.every((campo) => CAMPOS_MI_BOLSA.includes(campo)) || !lista.includes("bolsa")) {
    throw new TypeError("Los datos visibles de mi bolsa no son válidos.");
  }
  return [...lista];
}

export function campoVisibleMiBolsa(lista, campo) {
  return (Array.isArray(lista) ? lista : CAMPOS_MI_BOLSA).includes(campo);
}
