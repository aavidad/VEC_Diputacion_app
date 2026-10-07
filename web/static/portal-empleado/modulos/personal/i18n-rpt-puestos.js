import { cargarTextos } from "../../../comun/textos.js";
import { localizacionDe } from "../../../comun/idioma.js";

const TEXTOS_RPT_PUESTOS = await cargarTextos("personal");
const MENSAJES_RPT_PUESTOS = TEXTOS_RPT_PUESTOS.seccion("rpt_puestos");
export const IDIOMA_EFECTIVO_RPT_PUESTOS = TEXTOS_RPT_PUESTOS.idioma;
export const LOCALIZACION_EFECTIVA_RPT_PUESTOS = localizacionDe(TEXTOS_RPT_PUESTOS.idioma);
export const RESPALDO_RPT_PUESTOS = TEXTOS_RPT_PUESTOS.incidenciaCatalogo !== null;

export function crearTraductorRPTPuestos(catalogo = MENSAJES_RPT_PUESTOS) { if (!catalogo || typeof catalogo !== "object" || Object.keys(MENSAJES_RPT_PUESTOS).some((clave) => typeof catalogo[clave] !== "string" || catalogo[clave] === "")) throw new Error("catálogo RPT de puestos incompleto"); return (clave, variables = {}) => { if (!(clave in MENSAJES_RPT_PUESTOS)) throw new Error(`clave RPT desconocida: ${clave}`); return catalogo[clave].replace(/\{([a-z_]+)\}/g, (_m, nombre) => String(variables[nombre] ?? "")); }; }
export function formatearRecuentoRPTPuestos(total, vista, locale = LOCALIZACION_EFECTIVA_RPT_PUESTOS) { if (!Number.isSafeInteger(total) || total < 0 || !["categorias", "puestos", "centros"].includes(vista)) throw new TypeError("recuento RPT no válido"); const t = crearTraductorRPTPuestos(); const tipo = vista === "puestos" ? "puesto" : vista === "centros" ? "centro" : "categoria"; return t(`recuento_${tipo}_${total === 1 ? "uno" : "otro"}`, { total: new Intl.NumberFormat(locale, { useGrouping: "always" }).format(total) }); }
export function formatearResumenRPTPuestos(resumen, locale = LOCALIZACION_EFECTIVA_RPT_PUESTOS) { if (!resumen || typeof resumen !== "object" || ["puestos", "dotacion", "categorias", "centros"].some((campo) => !Number.isSafeInteger(resumen[campo]) || resumen[campo] < 0)) throw new TypeError("resumen RPT no válido"); const numero = new Intl.NumberFormat(locale, { useGrouping: "always" }); return crearTraductorRPTPuestos()("resumen", Object.fromEntries(Object.entries(resumen).map(([campo, valor]) => [campo, numero.format(valor)]))); }
export function formatearCentimosRPT(centimos, locale = LOCALIZACION_EFECTIVA_RPT_PUESTOS) { if (!Number.isSafeInteger(centimos) || centimos < 0) throw new TypeError("importe RPT no válido"); return new Intl.NumberFormat(locale, { style: "currency", currency: "EUR" }).format(centimos / 100); }

export function formatearEnlaceAgrupacionRPT(vista, nombre, puestos, dotacion, locale = LOCALIZACION_EFECTIVA_RPT_PUESTOS, catalogo = MENSAJES_RPT_PUESTOS) {
  if (!["categorias", "centros"].includes(vista) || typeof nombre !== "string" || nombre.trim() === ""
    || !Number.isSafeInteger(puestos) || puestos < 0 || !Number.isSafeInteger(dotacion) || dotacion < 0)
    throw new TypeError("agrupación RPT no válida");
  const t = crearTraductorRPTPuestos(catalogo);
  const numero = new Intl.NumberFormat(locale, { useGrouping: "always" });
  const plural = new Intl.PluralRules(locale);
  const recuento = (clave, total) => t(`recuento_${clave}_${plural.select(total) === "one" ? "uno" : "otro"}`,
    { total: numero.format(total) });
  return t(vista === "centros" ? "ver_puestos_centro_recuento" : "ver_puestos_categoria_recuento",
    { valor: nombre, puestos: recuento("puesto", puestos), dotacion: recuento("dotacion", dotacion) });
}
