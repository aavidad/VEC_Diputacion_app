import { encabezadoVista, enlaceRuta, escaparHTML, panel } from "./comunes.js";
import { textoContactoPropio } from "../i18n-contacto-propio.js";
import { traducir } from "../i18n.js";

const p = (clave, variables) => traducir(`areaPersonal.vista.perfil.${clave}`, variables);
const h = (texto) => escaparHTML(texto);

function opcionCheck(nombre, marcado, titulo, detalle) {
  return `<label class="opcion-check"><input type="checkbox" name="${nombre}" ${marcado ? "checked" : ""}><span><strong>${h(titulo)}</strong><small>${h(detalle)}</small></span></label>`;
}

export function renderizarPerfil(datos) {
  const preferenciasAviso = datos.preferencias_notificacion || {};
  const preferencias = `<form id="formulario-notificaciones" data-operacion="actualizar_notificaciones"><fieldset><legend>${h(p("avisos.leyenda"))}</legend>${opcionCheck("correo", preferenciasAviso.correo, p("avisos.correo"), p("avisos.correoDetalle"))}${opcionCheck("telegram", preferenciasAviso.telegram, p("avisos.telegram"), p("avisos.telegramDetalle"))}${opcionCheck("interno", preferenciasAviso.interno, p("avisos.interno"), p("avisos.internoDetalle"))}</fieldset><p class="nota">${h(p("avisos.nota"))}</p><button type="submit" class="boton-secundario">${h(p("avisos.guardar"))}</button></form>`;

  return `${encabezadoVista(p("titulo"), p("descripcion"))}
    <div class="rejilla-principal perfil"><div>
      ${panel(traducir("areaPersonal.ficha.panel.titulo"), "", '<div id="ficha-aspirante"></div>')}
      ${panel(textoContactoPropio("titulo"), textoContactoPropio("subtitulo"), '<div id="contacto-propio"></div>')}
    </div><aside>
      ${panel(p("avisos.titulo"), p("avisos.subtitulo"), preferencias)}
      ${panel(p("privacidad.titulo"), p("privacidad.subtitulo"), `<p>${h(p("privacidad.texto"))}</p>${enlaceRuta("ayuda", p("privacidad.enlace"), "enlace-boton")}`)}
    </aside></div>`;
}
