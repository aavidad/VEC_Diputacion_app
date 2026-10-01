import { TEXTOS_COPIAS, traducirCopias as t, traducirCodigo as tc } from "./i18n.js?v=20261001-cs09-copias-ux-v2";

export function soporte(documento) {
  const crear = (tag, texto = "", clase = "") => {
    const n = documento.createElement(tag);
    if (texto) n.textContent = texto;
    if (clase) n.className = clase;
    return n;
  };
  const boton = (clave, accion, { clase = "boton-secundario", disabled = false } = {}) => {
    const b = crear("button", t(clave), clase); b.type = "button"; b.disabled = disabled; b.dataset.copiasFoco = clave;
    b.addEventListener("click", accion); return b;
  };
  const panel = (clave) => {
    const p = crear("section", "", "panel"), h = crear("header", "", "cabecera-panel"), c = crear("div", "", "cuerpo-panel");
    const titulo = crear("h3", t(clave)); titulo.dataset.copiasFoco = "titulo:" + clave; h.append(titulo); p.append(h, c); return { p, h, c };
  };
  const dato = (dl, clave, valor) => { const grupo = crear("div"); grupo.append(crear("dt", t(clave)), crear("dd", valor ?? t("no_dato"))); dl.append(grupo); };
  const chip = (estado) => crear("span", tc("estados", estado), "estado-chip " + ({ valida: "exito", compatible: "exito", no_valida: "peligro", incompatible: "peligro", no_comprobable: "aviso", verificando: "info", completo: "exito", verificada_declarada: "aviso", no_valida_declarada: "peligro" }[estado] ?? "aviso"));
  const fecha = valor => valor ? TEXTOS_COPIAS.fecha(valor, { dateStyle: "short", timeStyle: "short", timeZone: TEXTOS_COPIAS.seccion("formato").zona_horaria }) : t("no_dato");
  const numero = valor => TEXTOS_COPIAS.numero(valor);
  const campo = (clave, tipo, valor, { min, max, required = true } = {}) => {
    const label = crear("label", t(clave), "copias-campo"), input = crear("input");
    input.type = tipo; input.name = clave; input.value = valor ?? ""; input.required = required;
    if (min !== undefined) input.min = min;
    if (max !== undefined) input.max = max;
    label.append(input); return { label, input };
  };
  return { crear, boton, panel, dato, chip, fecha, numero, campo };
}
