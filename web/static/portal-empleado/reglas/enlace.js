/** Acceso a la pantalla de reglas vigentes desde Bolsa y Contratación temporal (no va en el menú principal). */
import { crearTraductorReglas } from "./i18n.js?v=20260926-integracion-bolsa-ct-v1";

export const RUTA_PANTALLA_REGLAS = "/portal-empleado/reglas/";

const t = crearTraductorReglas();

/** Enlace secundario que abre la pantalla en otra pestaña; `clase` admite solo nombres simples. */
export function enlaceReglasVigentes(clase = "boton-secundario") {
  const segura = /^[a-z][a-z0-9 -]*$/u.test(clase) ? clase : "boton-secundario";
  return `<a class="${segura}" href="${RUTA_PANTALLA_REGLAS}" target="_blank" rel="noopener">${t("titulo")}</a>`;
}
