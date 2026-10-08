import { encabezadoVista, enlaceRuta, escaparAtributo, escaparHTML, panel } from "./comunes.js";
import { traducir } from "../i18n.js";

const y = (clave, variables) => traducir(`areaPersonal.vista.ayuda.${clave}`, variables);
const h = (texto) => escaparHTML(texto);

export function renderizarAyuda() {
  const transcripcion = `${traducir("areaPersonal.ayuda.guia.texto")} ${traducir("areaPersonal.ayuda.guia.limiteServicio")}`;
  return `${encabezadoVista(y("titulo"), traducir("areaPersonal.ayuda.descripcion"))}
    <div class="rejilla-principal"><div>
      ${panel(traducir("areaPersonal.ayuda.guia.titulo"), traducir("areaPersonal.ayuda.guia.subtitulo"), `<section aria-labelledby="titulo-guia-ayuda"><h3 id="titulo-guia-ayuda">${h(traducir("areaPersonal.ayuda.guia.encabezado"))}</h3><p>${h(transcripcion)}</p></section>`)}
    </div><aside aria-label="${escaparAtributo(y("lateral"))}">
      ${panel(y("visualizacion.titulo"), y("visualizacion.subtitulo"), `<div class="fila-acciones"><button type="button" class="boton-secundario" data-accion="alternar-texto">${h(y("visualizacion.texto"))}</button><button type="button" class="boton-secundario" data-accion="alternar-contraste">${h(y("visualizacion.contraste"))}</button></div><p>${h(y("visualizacion.detalle"))}</p>`)}
      ${panel(y("soporte.titulo"), y("soporte.subtitulo"), `<dl class="dato-lista"><dt>${h(y("soporte.tecnico"))}</dt><dd>${h(y("soporte.tecnicoDetalle"))}</dd><dt>${h(y("soporte.datos"))}</dt><dd>${h(y("soporte.datosDetalle"))}</dd></dl><p class="nota">${h(y("soporte.nota"))}</p>`)}
      ${panel(y("rapida.titulo"), y("rapida.subtitulo"), enlaceRuta("llamamientos", y("rapida.llamamiento"), "enlace-boton"))}
    </aside></div>`;
}
