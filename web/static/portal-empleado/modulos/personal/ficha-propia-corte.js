import { cargarTextos } from "../../../comun/textos.js";
import { LOCALIZACION_ACTUAL } from "../../../comun/idioma.js";

const textos = await cargarTextos("personal-corte-propio");
export const traducirCorteServicios = (clave, variables) => textos.traducir(`general.${clave}`, variables);

export function esFechaCorteServicios(valor) {
  if (typeof valor !== "string" || !/^\d{4}-\d{2}-\d{2}$/u.test(valor) || valor.startsWith("0000")) return false;
  const fecha = new Date(`${valor}T12:00:00Z`);
  return Number.isFinite(fecha.getTime()) && fecha.toISOString().slice(0, 10) === valor;
}

export function crearSelectorCorteServicios(d, referencia, consultar) {
  const t = traducirCorteServicios;
  const form = d.createElement("form"); form.className = "personal-ficha-corte";
  form.dataset.personalFichaCorte = "";
  const campo = d.createElement("div"); campo.className = "personal-ficha-corte-campo";
  const etiqueta = d.createElement("label"); etiqueta.textContent = t("fecha"); etiqueta.setAttribute("for", "personal-servicios-fecha");
  const fecha = d.createElement("input"); fecha.type = "date"; fecha.id = "personal-servicios-fecha";
  fecha.name = "fecha_referencia"; fecha.required = true; fecha.value = referencia; fecha.dataset.personalFichaFecha = "";
  fecha.setAttribute("aria-describedby", "personal-servicios-fecha-error");
  const error = d.createElement("p"); error.id = "personal-servicios-fecha-error"; error.setAttribute("role", "alert");
  const validar = () => {
    const valida = esFechaCorteServicios(fecha.value);
    fecha.setAttribute("aria-invalid", String(!valida)); error.textContent = valida ? "" : t("fecha_invalida");
    return valida;
  };
  fecha.addEventListener("blur", validar);
  fecha.addEventListener("input", () => { if (esFechaCorteServicios(fecha.value)) validar(); });
  form.addEventListener("submit", (evento) => { evento.preventDefault(); if (validar()) consultar(fecha.value); else fecha.focus(); });
  const aplicar = d.createElement("button"); aplicar.type = "submit"; aplicar.className = "boton-primario";
  aplicar.textContent = t("consultar"); aplicar.dataset.personalFichaConsultarCorte = "";
  const actuales = d.createElement("button"); actuales.type = "button"; actuales.className = "boton-secundario";
  actuales.textContent = t("actuales"); actuales.dataset.personalFichaCorteActual = "";
  actuales.addEventListener("click", () => consultar(""));
  campo.append(etiqueta, fecha, error); form.append(campo, aplicar, actuales); return form;
}

export function presentarFechaCorteServicios(iso) {
  return new Intl.DateTimeFormat(LOCALIZACION_ACTUAL, { dateStyle: "medium", timeZone: "UTC" }).format(new Date(`${iso}T12:00:00Z`));
}
