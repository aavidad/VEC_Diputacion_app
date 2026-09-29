import { cargarTextos } from "../../../comun/textos.js";
import { LOCALIZACION_ACTUAL } from "../../../comun/idioma.js";

/** Textos de los apartados de la ficha propia servidos por Personal. */
const MENSAJES_FICHA_PROPIA = (await cargarTextos("personal")).seccion("ficha_propia");

const CLAVES = Object.freeze(Object.keys(MENSAJES_FICHA_PROPIA));

export function crearTraductorFichaPropia(catalogo = MENSAJES_FICHA_PROPIA) {
  if (!catalogo || typeof catalogo !== "object"
    || CLAVES.some((clave) => typeof catalogo[clave] !== "string" || catalogo[clave] === "")) {
    throw new Error("catálogo i18n de la ficha propia incompleto");
  }
  return (clave, variables = {}) => {
    if (!CLAVES.includes(clave)) throw new Error(`clave i18n de la ficha propia desconocida: ${clave}`);
    return catalogo[clave].replace(/\{([a-z_]+)\}/g, (_coincidencia, variable) => String(variables[variable] ?? ""));
  };
}

/** Días reconocidos con el plural y el formato numérico de la interfaz. */
export function formatearDiasFichaPropia(total, t = crearTraductorFichaPropia(), locale = LOCALIZACION_ACTUAL) {
  if (!Number.isSafeInteger(total) || total < 0) throw new TypeError("días reconocidos no válidos");
  return t(total === 1 ? "dias_uno" : "dias_otro", { total: new Intl.NumberFormat(locale).format(total) });
}
