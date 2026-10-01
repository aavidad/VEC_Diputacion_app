/** Textos del circuito de firma de los borradores del expediente. */
import { IDIOMA_POR_DEFECTO, IDIOMAS_DISPONIBLES, localizacionDe } from "../../../comun/idioma.js";
import { cargarCatalogosContratacion } from "./i18n-catalogos.js?v=20261001-ct-a-i18n-v1";
import { cargarTextos } from "../../../comun/textos.js";

const catalogos = await cargarCatalogosContratacion("contratacion-temporal-circuito-firma");
export const MENSAJES_CIRCUITO_FIRMA_ES = catalogos.exportaciones.ES;
export const MENSAJES_CIRCUITO_FIRMA_EN = catalogos.exportaciones.EN;

export function crearTraductorCircuitoFirma(sobrescrituras = {}, locale = localizacionDe(IDIOMA_POR_DEFECTO)) {
  if (sobrescrituras === null || typeof sobrescrituras !== "object" || Array.isArray(sobrescrituras)) {
    throw new TypeError("mensajes del circuito de firma no válidos");
  }
  const idioma = IDIOMAS_DISPONIBLES.find(({ codigo, localizacion }) =>
    locale === codigo || locale === localizacion || locale.startsWith(`${codigo}-`))?.codigo ?? IDIOMA_POR_DEFECTO;
  const mensajes = { ...catalogos.porIdioma[idioma] };
  for (const clave of Object.keys(MENSAJES_CIRCUITO_FIRMA_ES)) {
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

const VALORES_CONTROLADOS = (await cargarTextos("contratacion-temporal-circuito-firma", {
  idioma: IDIOMA_POR_DEFECTO,
})).seccion("valores_controlados");

/** Traduce únicamente valores conocidos del catálogo; conserva otros como datos. */
export function traducirValorCircuitoFirma(tipo, valor, t) {
  const clave = Object.entries(VALORES_CONTROLADOS[tipo] ?? {})
    .find(([, origen]) => origen === valor)?.[0];
  return clave ? t(`circuito_firma_${tipo}_${clave}`) : valor;
}
