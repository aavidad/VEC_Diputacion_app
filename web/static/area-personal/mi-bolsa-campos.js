import { localizacionAreaPersonal, traducir } from "./i18n.js";

// Datos de «Mi bolsa» que el catálogo de Bolsa (regla b29.campos_mi_bolsa)
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

// Nombre legible de la categoría de una participación u oferta. Si el dato
// trae etiqueta (categoria_nombre o etiqueta) se usa. Si solo trae un código
// del tipo «categoria:rpt:administrativo», se toma el último tramo tras «:»,
// se cambian «_» y «-» por espacios y se pone mayúscula inicial
// («Administrativo»). Un texto sin «:» ya es un nombre y se deja igual.
// Sin nada utilizable: «Categoría sin nombre» (catálogo i18n).
export function nombreCategoria(item) {
  const etiqueta = [item?.categoria_nombre, item?.etiqueta].find((v) => typeof v === "string" && v.trim());
  if (etiqueta) return etiqueta.trim();
  const codigo = typeof item?.categoria === "string" ? item.categoria.trim() : "";
  if (codigo && !codigo.includes(":")) return codigo;
  const tramo = codigo.split(":").pop().replace(/[_-]+/gu, " ").trim();
  if (!tramo) return traducir("areaPersonal.miBolsa.categoriaSinNombre");
  return tramo.charAt(0).toLocaleUpperCase(localizacionAreaPersonal()) + tramo.slice(1);
}
