/**
 * Mensajes de la validación nativa de formularios en el idioma del portal.
 *
 * Los atributos `required`, `minlength`, `type="email"`… hacen que el navegador
 * muestre un bocadillo con un texto propio en el idioma de su interfaz (por
 * ejemplo, «Please fill out this field.»), ajeno al idioma elegido en VEC. Este
 * módulo escucha el evento `invalid` en fase de captura y sustituye ese texto
 * por el mensaje del catálogo del portal. Conserva el bloqueo nativo del envío
 * y respeta los mensajes que un módulo haya fijado con `setCustomValidity`.
 */
import { LOCALIZACION_PORTAL, traducirPortal } from "./portal-i18n.js?v=20260926-pulido-portal-v1";

const PROPIOS = new WeakMap();

/** Devuelve el mensaje traducido para el estado de validez de un control. */
export function mensajeValidacionPortal(control, traducir = traducirPortal) {
  const validez = control?.validity;
  if (!validez || validez.valid) return "";
  const tipo = String(control.type || "").toLowerCase();
  if (validez.valueMissing) {
    if (tipo === "checkbox") return traducir("validacion_casilla_obligatoria");
    if (tipo === "radio" || control.tagName === "SELECT") return traducir("validacion_opcion_obligatoria");
    if (tipo === "file") return traducir("validacion_archivo_obligatorio");
    return traducir("validacion_campo_obligatorio");
  }
  if (validez.badInput) return traducir(tipo === "number" ? "validacion_numero_invalido" : "validacion_valor_invalido");
  if (validez.typeMismatch) return traducir(tipo === "email" ? "validacion_correo_invalido" : tipo === "url" ? "validacion_url_invalida" : "validacion_valor_invalido");
  if (validez.tooShort) return traducir("validacion_demasiado_corto", { minimo: control.minLength });
  if (validez.tooLong) return traducir("validacion_demasiado_largo", { maximo: control.maxLength });
  if (validez.rangeUnderflow) return traducir("validacion_menor_minimo", { minimo: control.min });
  if (validez.rangeOverflow) return traducir("validacion_mayor_maximo", { maximo: control.max });
  if (validez.stepMismatch) return traducir("validacion_paso_invalido");
  if (validez.patternMismatch) return traducir("validacion_formato_invalido");
  return traducir("validacion_valor_invalido");
}

/**
 * Instala la traducción en un documento. Es idempotente y devuelve una función
 * para retirarla.
 */
export function instalarValidacionI18n(documento = globalThis.document, traducir = traducirPortal) {
  if (!documento?.addEventListener) return () => {};
  if (documento.__vecValidacionI18n) return documento.__vecValidacionI18n;
  const alInvalido = (evento) => {
    const control = evento.target;
    if (typeof control?.setCustomValidity !== "function") return;
    const propio = PROPIOS.get(control);
    if (control.validity?.customError && control.validationMessage !== propio) return;
    if (propio !== undefined) { control.setCustomValidity(""); PROPIOS.delete(control); }
    const mensaje = mensajeValidacionPortal(control, traducir);
    if (!mensaje) return;
    control.setCustomValidity(mensaje);
    PROPIOS.set(control, mensaje);
  };
  const alCambiar = (evento) => {
    const control = evento.target;
    const propio = PROPIOS.get(control);
    if (propio === undefined) return;
    PROPIOS.delete(control);
    if (control.validationMessage === propio) control.setCustomValidity("");
  };
  documento.addEventListener("invalid", alInvalido, true);
  documento.addEventListener("input", alCambiar, true);
  documento.addEventListener("change", alCambiar, true);
  const retirar = () => {
    documento.removeEventListener("invalid", alInvalido, true);
    documento.removeEventListener("input", alCambiar, true);
    documento.removeEventListener("change", alCambiar, true);
    delete documento.__vecValidacionI18n;
  };
  documento.__vecValidacionI18n = retirar;
  return retirar;
}

/**
 * Alinea `<html lang>` con la localización del portal, para que lectores de
 * pantalla, separación silábica y controles nativos usen el mismo idioma.
 */
export function aplicarIdiomaDocumento(documento = globalThis.document) {
  const raiz = documento?.documentElement;
  if (raiz) raiz.lang = String(LOCALIZACION_PORTAL).split("-")[0];
}

const ATRIBUTOS_TRADUCIBLES = Object.freeze([
  ["data-i18n-portal-aria-label", "aria-label"],
  ["data-i18n-portal-title", "title"],
  ["data-i18n-portal-placeholder", "placeholder"],
  ["data-i18n-portal-alt", "alt"],
]);

/**
 * Sustituye en un documento estático los textos y atributos marcados con
 * `data-i18n-portal` y `data-i18n-portal-<atributo>` por las claves del catálogo.
 */
export function aplicarTextosPortal(documento = globalThis.document, traducir = traducirPortal) {
  documento?.querySelectorAll?.("[data-i18n-portal]").forEach((elemento) => {
    elemento.textContent = traducir(elemento.getAttribute("data-i18n-portal"));
  });
  for (const [marca, atributo] of ATRIBUTOS_TRADUCIBLES) {
    documento?.querySelectorAll?.(`[${marca}]`).forEach((elemento) => {
      elemento.setAttribute(atributo, traducir(elemento.getAttribute(marca)));
    });
  }
}
