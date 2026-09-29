import { IDIOMA_ACTUAL } from "../comun/idioma.js";

/** Solo los tipos de lista definidos por el contrato de Bolsa. */
export const TIPOS_LISTA_ES = Object.freeze({
  ordinaria: "Ordinaria",
  rotatoria: "Rotatoria",
});

export const TIPOS_LISTA_EN = Object.freeze({
  ordinaria: "Ordinary",
  rotatoria: "Rotating",
});

export function traducirTipoLista(clave, idioma = IDIOMA_ACTUAL) {
  const catalogo = idioma === "en" ? TIPOS_LISTA_EN : TIPOS_LISTA_ES;
  const valor = String(clave ?? "");
  return Object.hasOwn(catalogo, valor) ? catalogo[valor] : valor;
}
