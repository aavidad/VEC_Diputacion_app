import { encabezadoVista, escaparHTML, panel } from "./comunes.js";
import { textoContactoPropio } from "../i18n-contacto-propio.js";
import { traducir } from "../i18n.js";

const p = (clave, variables) => traducir(`areaPersonal.vista.perfil.${clave}`, variables);

export function renderizarPerfil() {
  return `${encabezadoVista(p("titulo"), p("descripcion"))}
    <div class="rejilla-principal perfil"><div>
      ${panel(traducir("areaPersonal.ficha.panel.titulo"), "", '<div id="ficha-aspirante"></div>')}
      ${panel(textoContactoPropio("titulo"), textoContactoPropio("subtitulo"), '<div id="contacto-propio"></div>')}
    </div><aside>
      ${panel(p("privacidad.titulo"), p("privacidad.subtitulo"), `<p>${escaparHTML(p("privacidad.texto"))}</p>`)}
    </aside></div>`;
}
