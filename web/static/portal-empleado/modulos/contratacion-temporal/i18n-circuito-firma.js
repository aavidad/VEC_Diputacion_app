/** Textos del circuito de firma de los borradores del expediente. */
import { IDIOMA_ACTUAL, IDIOMAS_DISPONIBLES, localizacionDe } from "../../../comun/idioma.js";
import { cargarCatalogosContratacion } from "./i18n-catalogos.js?v=20261007-pantallas-textos-final-v1";

const catalogos = await cargarCatalogosContratacion("contratacion-temporal-circuito-firma");
export const MENSAJES_CIRCUITO_FIRMA_ES = catalogos.exportaciones.ES;
export const MENSAJES_CIRCUITO_FIRMA_EN = catalogos.exportaciones.EN;

export function crearTraductorCircuitoFirma(sobrescrituras = {}, locale = localizacionDe(IDIOMA_ACTUAL)) {
  if (sobrescrituras === null || typeof sobrescrituras !== "object" || Array.isArray(sobrescrituras)) {
    throw new TypeError("mensajes del circuito de firma no válidos");
  }
  const idioma = IDIOMAS_DISPONIBLES.find(({ codigo, localizacion }) =>
    locale === codigo || locale === localizacion || locale.startsWith(`${codigo}-`))?.codigo ?? IDIOMA_ACTUAL;
  if (idioma !== catalogos.idioma && Object.keys(catalogos.actual).some((clave) =>
    typeof sobrescrituras[clave] !== "string" || sobrescrituras[clave].trim() === "")) {
    throw new RangeError("catálogo del idioma solicitado no cargado");
  }
  const mensajes = { ...catalogos.actual };
  for (const clave of Object.keys(catalogos.actual)) {
    const valor = sobrescrituras[clave];
    if (typeof valor === "string" && valor.trim() !== "") mensajes[clave] = valor;
  }
  return (clave, variables = {}) => {
    if (!Object.hasOwn(mensajes, clave)) throw new Error(`falta la traducción ${clave}`);
    return Object.entries(variables).reduce(
      (texto, [nombre, valor]) => texto.replaceAll(`{${nombre}}`, String(valor)),
      mensajes[clave],
    );
  };
}

const VALORES_CONTROLADOS = (await cargarCatalogosContratacion(
  "contratacion-temporal-circuito-firma", "valores_controlados",
)).actual;

/** Traduce únicamente valores conocidos del catálogo; conserva otros como datos. */
export function traducirValorCircuitoFirma(tipo, valor, t) {
  const clave = Object.entries(VALORES_CONTROLADOS[tipo] ?? {})
    .find(([, origen]) => origen === valor)?.[0];
  return clave ? t(`circuito_firma_${tipo}_${clave}`) : valor;
}
