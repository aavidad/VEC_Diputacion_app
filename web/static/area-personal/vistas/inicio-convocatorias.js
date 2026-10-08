import { encabezadoVista, enlaceRuta, panel } from "./comunes.js";
import { traducir } from "../i18n.js";

const t = (clave, variables) => traducir(`areaPersonal.vista.inicio.${clave}`, variables);

export function renderizarInicio() {
  return `${encabezadoVista(traducir("areaPersonal.rutas.inicio"), "")}
    ${panel(t("consultaBolsa.titulo"), "", enlaceRuta("llamamientos", t("consultaBolsa.abrir"), "boton-primario"))}`;
}
