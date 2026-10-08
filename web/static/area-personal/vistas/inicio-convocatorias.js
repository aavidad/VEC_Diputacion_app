import { encabezadoVista, enlaceRuta, escaparHTML, panel } from "./comunes.js";
import { traducir } from "../i18n.js";

const t = (clave, variables) => traducir(`areaPersonal.vista.inicio.${clave}`, variables);

export function renderizarInicio() {
  return `${encabezadoVista(traducir("areaPersonal.rutas.inicio"), t("consultaBolsa.descripcion"))}
    ${panel(t("consultaBolsa.titulo"), t("consultaBolsa.subtitulo"),
      `<p>${escaparHTML(t("consultaBolsa.detalle"))}</p>${enlaceRuta("llamamientos", t("consultaBolsa.abrir"), "boton-primario")}`)}`;
}
