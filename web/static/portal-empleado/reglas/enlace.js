/** Acceso a la pantalla de reglas vigentes desde Bolsa y Contratación temporal (no va en el menú principal). */
import { textoPortal } from "../portal-i18n.js?v=20261007-pantallas-textos-final-v1";

export const RUTA_PANTALLA_REGLAS = "/portal-empleado/reglas/";

/** Enlace secundario que abre la pantalla en otra pestaña; `clase` admite solo nombres simples. */
export function enlaceReglasVigentes(clase = "boton-secundario") {
  const segura = /^[a-z][a-z0-9 -]*$/u.test(clase) ? clase : "boton-secundario";
  return `<a class="${segura}" href="${RUTA_PANTALLA_REGLAS}" target="_blank" rel="noopener">${textoPortal("txt_reglas_vigentes")}</a>`;
}
