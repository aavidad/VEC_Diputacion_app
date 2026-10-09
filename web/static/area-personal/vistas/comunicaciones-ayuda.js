import {
  botonOperacion, chip, encabezadoVista, enlaceRuta, escaparAtributo, escaparHTML,
  estadoVacio, panel, tabla,
} from "./comunes.js";
import { traducir } from "../i18n.js";

const n = (clave, variables) => traducir(`areaPersonal.vista.mensajes.${clave}`, variables);
const c = (clave, variables) => traducir(`areaPersonal.vista.certificados.${clave}`, variables);
const y = (clave, variables) => traducir(`areaPersonal.vista.ayuda.${clave}`, variables);
const h = (texto) => escaparHTML(texto);

function opcionAviso(nombre, marcado, titulo, detalle) {
  return `<label class="opcion-check"><input type="checkbox" name="${nombre}" ${marcado ? "checked" : ""}><span><strong>${h(titulo)}</strong><small>${h(detalle)}</small></span></label>`;
}

export function renderizarMensajes(datos) {
  const preferenciasAviso = datos.preferencias_notificacion || {};
  const listaMensajes = Array.isArray(datos.mensajes) ? datos.mensajes : null;
  const mensajes = listaMensajes === null
    ? `<p class="nota error" role="alert"><strong>${h(n("errorBandeja"))}</strong> ${h(n("errorBandejaDetalle"))}</p>`
    : listaMensajes.map((item) => `<li><span class="fecha-bloque" aria-hidden="true">${item.estado === "No leído" ? "!" : "·"}</span><span><strong>${escaparHTML(item.asunto)}</strong><small>${escaparHTML(item.resumen)}</small><small>${escaparHTML(item.fecha)} · ${escaparHTML(item.tipo)}</small></span><div class="fila-acciones">${item.estado === "No leído" ? botonOperacion("marcar_mensaje", n("marcarLeido"), { id: item.id, clase: "boton-secundario", descripcion: n("marcarLeidoDescripcion", { asunto: item.asunto }) }) : chip(item.estado)}${enlaceRuta(item.ruta, n("abrirAsunto"), "enlace-boton")}</div></li>`).join("");
  const bandeja = listaMensajes === null ? mensajes : listaMensajes.length ? `<ul class="lista-mensajes">${mensajes}</ul>` : estadoVacio(n("vacio"), n("vacioDetalle"));
  const preferencias = `<p class="nota aviso" role="status">${h(n("preferencias.aviso"))}</p><form data-operacion="actualizar_notificaciones"><fieldset><legend>${h(n("preferencias.leyenda"))}</legend>${opcionAviso("convocatorias", preferenciasAviso.convocatorias, n("preferencias.convocatorias"), n("preferencias.convocatoriasDetalle"))}${opcionAviso("plazos", preferenciasAviso.plazos, n("preferencias.plazos"), n("preferencias.plazosDetalle"))}${opcionAviso("llamamientos", preferenciasAviso.llamamientos, n("preferencias.llamamientos"), n("preferencias.llamamientosDetalle"))}${opcionAviso("noticias", preferenciasAviso.noticias, n("preferencias.noticias"), n("preferencias.noticiasDetalle"))}</fieldset><button type="submit" class="boton-secundario">${h(n("preferencias.guardar"))}</button></form>`;
  return `${encabezadoVista(n("titulo"), n("descripcion"))}
    <div class="rejilla-principal"><div>${panel(n("bandeja"), n("bandejaSubtitulo"), bandeja, { estado: n("noLeidos", { cuenta: datos.resumen?.mensajes_no_leidos || 0 }) })}</div><aside aria-label="${escaparAtributo(n("lateral"))}">${panel(n("preferencias.titulo"), n("preferencias.subtitulo"), preferencias)}${panel(n("diferencia.titulo"), n("diferencia.subtitulo"), `<p class="nota aviso">${h(n("diferencia.texto"))}</p>`)}</aside></div>`;
}

export function renderizarCertificados(datos) {
  const listaCertificados = Array.isArray(datos.certificados) ? datos.certificados : null;
  const certificados = (listaCertificados || []).map((item) => {
    const formatos = item.formatos.split(/, | o /);
    return `<article class="panel"><header><div><h3>${escaparHTML(item.tipo)}</h3><p><small>${h(c("referencia", { id: item.id }))}</small></p></div>${chip(item.estado)}</header><div class="panel-contenido"><p>${escaparHTML(item.descripcion)}</p><form class="fila-acciones" data-operacion="solicitar_certificado" data-id="${escaparAtributo(item.id)}"><div class="campo"><label for="formato-${escaparAtributo(item.id)}">${h(c("formato"))}</label><select id="formato-${escaparAtributo(item.id)}" name="formato">${formatos.map((formato) => `<option>${escaparHTML(formato)}</option>`).join("")}</select></div><button type="submit" class="boton-primario">${h(c("solicitar"))}</button></form></div></article>`;
  }).join("");
  const filas = (Array.isArray(datos.documentos) ? datos.documentos : []).map((item) => [
    `<strong>${escaparHTML(item.nombre)}</strong><small>${escaparHTML(item.id)}</small>`, escaparHTML(item.tipo),
    escaparHTML(item.fecha), chip(item.estado),
    `<div class="acciones-tabla">${botonOperacion("solicitar_descarga", c("descargar"), { id: item.id, clase: "boton-secundario", descripcion: c("prepararDescarga", { nombre: item.nombre }) })}</div>`,
  ]);
  return `${encabezadoVista(c("titulo"), c("descripcion"))}
    <p class="nota aviso">${h(c("nota"))}</p>
    <div class="rejilla-dos">${listaCertificados === null ? `<p class="nota error" role="alert"><strong>${h(c("error"))}</strong> ${h(c("errorDetalle"))}</p>` : certificados || estadoVacio(c("vacio"), c("vacioDetalle"))}</div>
    ${panel(c("documentos.titulo"), c("documentos.subtitulo"), tabla({ descripcion: c("documentos.tabla"), columnas: [c("columnas.documento"), c("columnas.tipo"), c("columnas.fecha"), c("columnas.estado"), c("columnas.accion")], filas }))}`;
}

export function renderizarAyuda(datos, estado = {}) {
  const termino = (estado.consultaAyuda || "").trim().toLowerCase();
  const ayuda = Array.isArray(datos.ayuda) ? datos.ayuda : null;
  const preguntas = (ayuda || []).filter((item) => !termino || `${item.pregunta} ${item.respuesta}`.toLowerCase().includes(termino));
  const faq = ayuda === null ? `<p class="nota error" role="alert"><strong>${h(y("error"))}</strong> ${h(y("errorDetalle"))}</p>` : preguntas.map((item) => `<details><summary>${escaparHTML(item.pregunta)}</summary><p>${escaparHTML(item.respuesta)}</p></details>`).join("") || estadoVacio(y("sinCoincidencias"), y("sinCoincidenciasDetalle"));
  const transcripcion = `${traducir("areaPersonal.ayuda.guia.texto")} ${traducir("areaPersonal.ayuda.guia.limiteServicio")}`;
  return `${encabezadoVista(y("titulo"), traducir("areaPersonal.ayuda.descripcion"), `<button type="button" class="boton-primario" data-accion="leer-pantalla">${h(y("leerPagina"))}</button>`)}
    <div class="rejilla-principal"><div>
      ${panel(y("buscar.titulo"), y("buscar.subtitulo"), `<form id="busqueda-ayuda" data-accion="buscar-ayuda"><div class="campo"><label for="consulta-ayuda">${h(y("buscar.pregunta"))}</label><input id="consulta-ayuda" name="consulta" type="search" value="${escaparAtributo(estado.consultaAyuda || "")}" placeholder="${escaparAtributo(y("buscar.ejemplo"))}"></div><button type="submit" class="boton-primario">${h(y("buscar.boton"))}</button></form><div class="resultado-ayuda">${faq}</div>`)}
      ${panel(traducir("areaPersonal.ayuda.guia.titulo"), traducir("areaPersonal.ayuda.guia.subtitulo"), `<section aria-labelledby="titulo-guia-ayuda"><h4 id="titulo-guia-ayuda">${escaparHTML(traducir("areaPersonal.ayuda.guia.encabezado"))}</h4><p>${escaparHTML(transcripcion)}</p></section>`)}
    </div><aside aria-label="${escaparAtributo(y("lateral"))}">
      ${panel(y("visualizacion.titulo"), y("visualizacion.subtitulo"), `<div class="fila-acciones"><button type="button" class="boton-secundario" data-accion="alternar-texto">${h(y("visualizacion.texto"))}</button><button type="button" class="boton-secundario" data-accion="alternar-contraste">${h(y("visualizacion.contraste"))}</button><button type="button" class="boton-secundario" data-accion="leer-pantalla">${h(y("visualizacion.voz"))}</button></div><p>${h(y("visualizacion.detalle"))}</p>`)}
      ${panel(y("soporte.titulo"), y("soporte.subtitulo"), `<dl class="dato-lista"><dt>${h(y("soporte.asistente"))}</dt><dd>${h(y("soporte.asistenteDetalle"))}</dd><dt>${h(y("soporte.tecnico"))}</dt><dd>${h(y("soporte.tecnicoDetalle"))}</dd><dt>${h(y("soporte.datos"))}</dt><dd>${h(y("soporte.datosDetalle"))}</dd></dl><p class="nota">${h(y("soporte.nota"))}</p>`)}
    </aside></div>`;
}
