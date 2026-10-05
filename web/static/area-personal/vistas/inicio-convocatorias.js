import {
  botonOperacion, chip, cifraResumen, encabezadoVista, enlaceRuta, escaparAtributo, escaparHTML,
  estadoVacio, formatoPuntos, listaDatos, panel, tabla,
} from "./comunes.js";
import { traducir } from "../i18n.js";

const t = (clave, variables) => traducir(`areaPersonal.vista.inicio.${clave}`, variables);
const c = (clave, variables) => traducir(`areaPersonal.vista.convocatorias.${clave}`, variables);
const soloMiBolsa = (datos) => datos.meta?.origen === "GET /api/vec/bolsa/mi-bolsa";

export function renderizarInicio(datos) {
  if (soloMiBolsa(datos)) {
    return `${encabezadoVista(traducir("areaPersonal.rutas.inicio"), t("consultaBolsa.descripcion"))}
      ${panel(t("consultaBolsa.titulo"), t("consultaBolsa.subtitulo"),
        `<p>${escaparHTML(t("consultaBolsa.detalle"))}</p>${enlaceRuta("llamamientos", t("consultaBolsa.abrir"), "boton-primario")}`)}`;
  }
  const cifras = [
    [datos.resumen.acciones_pendientes, t("cifras.pendientes"), t("cifras.pendientesAyuda"), "aviso", "pendiente"],
    [datos.resumen.convocatorias_abiertas, t("cifras.convocatorias"), t("cifras.convocatoriasAyuda"), "", "documento"],
    [datos.resumen.solicitudes_activas, t("cifras.solicitudes"), t("cifras.solicitudesAyuda"), "exito", "expediente"],
    [formatoPuntos(datos.resumen.puntuacion_provisional), t("cifras.puntos"), t("cifras.puntosAyuda"), "merito", "grafico"],
  ].map(([valor, etiqueta, ayuda, clase, nombreIcono]) => cifraResumen(valor, etiqueta, ayuda, { clase, nombreIcono })).join("");

  const plazos = datos.plazos.map((plazo) => `<li><span class="fecha-bloque">${escaparHTML(plazo.dia)}<small>${escaparHTML(plazo.mes)}</small></span><span><strong>${escaparHTML(plazo.titulo)}</strong><small>${escaparHTML(plazo.detalle)}</small></span>${enlaceRuta(plazo.ruta, t("abrir"), "enlace-boton")}</li>`).join("");
  const acciones = datos.mensajes.filter((mensaje) => mensaje.estado === "No leído").slice(0, 3).map((mensaje) => `<li><span class="fecha-bloque" aria-hidden="true">!</span><span><strong>${escaparHTML(mensaje.asunto)}</strong><small>${escaparHTML(mensaje.resumen)}</small></span>${enlaceRuta(mensaje.ruta, t("atender"), "enlace-boton")}</li>`).join("");
  const solicitudes = datos.solicitudes.map((solicitud) => [
    `<strong>${escaparHTML(solicitud.titulo)}</strong><small>${escaparHTML(solicitud.referencia)}</small>`,
    chip(solicitud.estado),
    `<strong>${escaparHTML(t("puntos", { puntos: formatoPuntos(solicitud.puntuacion) }))}</strong><small>${escaparHTML(solicitud.posicion)}</small>`,
    `<div class="acciones-tabla"><button type="button" class="boton-secundario" data-accion="abrir-expediente" data-id="${escaparAtributo(solicitud.id)}">${escaparHTML(t("verSeguimiento"))}</button></div>`,
  ]);
  const actividad = datos.actividad.slice(0, 4).map((item) => `<li><span class="fecha-bloque" aria-hidden="true">·</span><span><strong>${escaparHTML(item.titulo)}</strong><small>${escaparHTML(item.detalle)}</small><small>${escaparHTML(item.actor)}</small></span><small>${escaparHTML(item.fecha)}</small></li>`).join("");

  return `${encabezadoVista(t("titulo"), t("descripcion"), enlaceRuta("convocatorias", t("verConvocatorias"), "boton-primario"))}
    <section class="resumen-cifras" aria-label="${escaparAtributo(t("resumen"))}">${cifras}</section>
    <div class="rejilla-principal"><div>
      ${panel(t("plazos.titulo"), t("plazos.subtitulo"), plazos
        ? `<ul class="lista-plazos">${plazos}</ul>`
        : estadoVacio(t("plazos.vacioTitulo"), t("plazos.vacioDetalle")), { estado: t("plazos.cuenta", { cuenta: datos.plazos.length }) })}
      ${panel(t("solicitudes.titulo"), t("solicitudes.subtitulo"), tabla({ descripcion: t("solicitudes.tabla"), columnas: [t("columnas.proceso"), t("columnas.estado"), t("columnas.puntuacion"), t("columnas.accion")], filas: solicitudes }))}
    </div><aside>
      ${panel(t("atencion.titulo"), t("atencion.subtitulo"), acciones ? `<ul class="lista-mensajes">${acciones}</ul>` : `<p>${escaparHTML(t("atencion.vacio"))}</p>`, { estado: t("atencion.cuenta", { cuenta: datos.resumen.acciones_pendientes }) })}
      ${panel(t("actividad.titulo"), t("actividad.subtitulo"), actividad
        ? `<ul class="lista-actividad">${actividad}</ul>`
        : estadoVacio(t("actividad.vacioTitulo"), t("actividad.vacioDetalle")))}
    </aside></div>`;
}

export function renderizarConvocatorias(datos, estado) {
  if (soloMiBolsa(datos)) {
    return `${encabezadoVista(c("titulo"), c("consultaBolsa.descripcion"))}
      ${panel(c("consultaBolsa.titulo"), c("consultaBolsa.subtitulo"),
        `<p>${escaparHTML(c("consultaBolsa.detalle"))}</p>${enlaceRuta("llamamientos", c("consultaBolsa.abrir"), "boton-primario")}`)}`;
  }
  // «Todas» es a la vez la opción visible y el valor del filtro: se toma del catálogo.
  const todas = c("todas");
  const termino = estado.filtros?.termino?.toLowerCase() || "";
  const filtroEstado = estado.filtros?.estado || todas;
  const filtroCategoria = estado.filtros?.categoria || todas;
  const categorias = [...new Set(datos.convocatorias.map((item) => item.categoria))];
  const resultados = datos.convocatorias.filter((item) => {
    const coincideTexto = !termino || `${item.titulo} ${item.referencia} ${item.categoria}`.toLowerCase().includes(termino);
    return coincideTexto && (filtroEstado === todas || item.estado === filtroEstado)
      && (filtroCategoria === todas || item.categoria === filtroCategoria);
  });
  const tarjetas = resultados.map((convocatoria) => {
    return `<article class="tarjeta-convocatoria"><div><div class="metadatos"><span>${escaparHTML(convocatoria.cve_bop || convocatoria.referencia)}</span>${chip(convocatoria.estado)}<span>${escaparHTML(convocatoria.categoria)}</span></div><h3>${escaparHTML(convocatoria.titulo)}</h3><p>${escaparHTML(convocatoria.descripcion)}</p><div class="metadatos"><strong>${escaparHTML(convocatoria.plazo)}</strong>${convocatoria.publicada_en ? `<span>${escaparHTML(c("publicada", { fecha: convocatoria.publicada_en }))}</span>` : ""}</div></div><div class="fila-acciones"><button type="button" class="boton-secundario" data-accion="abrir-convocatoria" data-id="${escaparAtributo(convocatoria.id)}">${escaparHTML(c("verDetalle"))}</button></div></article>`;
  }).join("");

  return `${encabezadoVista(c("titulo"), c("descripcion"))}
    <section class="panel"><form class="filtros" id="filtros-convocatorias" data-accion="filtrar-convocatorias"><div class="campo"><label for="filtro-texto">${escaparHTML(c("filtros.buscar"))}</label><input id="filtro-texto" name="termino" type="search" value="${escaparAtributo(estado.filtros?.termino || "")}" placeholder="${escaparAtributo(c("filtros.buscarEjemplo"))}"></div><div class="campo"><label for="filtro-estado">${escaparHTML(c("filtros.estado"))}</label><select id="filtro-estado" name="estado">${[todas, ...new Set(datos.convocatorias.map((item) => item.estado))].map((valor) => `<option${valor === filtroEstado ? " selected" : ""}>${escaparHTML(valor)}</option>`).join("")}</select></div><div class="campo"><label for="filtro-categoria">${escaparHTML(c("filtros.categoria"))}</label><select id="filtro-categoria" name="categoria">${[todas, ...categorias].map((valor) => `<option${valor === filtroCategoria ? " selected" : ""}>${escaparHTML(valor)}</option>`).join("")}</select></div><button type="submit" class="boton-primario">${escaparHTML(c("filtros.aplicar"))}</button></form><div class="panel-contenido">${tarjetas || estadoVacio(datos.convocatorias.length ? c("sinResultados") : c("sinConvocatorias"), datos.convocatorias.length ? c("sinResultadosDetalle") : c("sinConvocatoriasDetalle"))}</div></section>`;
}

export function renderizarDetalleConvocatoria(datos, estado) {
  const convocatoria = datos.convocatorias.find((item) => item.id === estado.convocatoriaSeleccionada);
  if (!convocatoria) {
    const consultaBolsa = soloMiBolsa(datos);
    return `${encabezadoVista(c("detalle.noEncontradaTitulo"), c("detalle.noEncontradaDetalle"))}
      ${panel(c("detalle.noEncontradaTitulo"), "", estadoVacio(c("detalle.noEncontradaEstado"),
        c(consultaBolsa ? "detalle.noEncontradaAyudaBolsa" : "detalle.noEncontradaAyuda"),
        enlaceRuta(consultaBolsa ? "llamamientos" : "convocatorias",
          c(consultaBolsa ? "consultaBolsa.abrir" : "detalle.volver"), "boton-secundario")))}`;
  }
  const requisitos = convocatoria.requisitos.map((requisito) => `<li>${escaparHTML(requisito)}</li>`).join("");
  const documentos = convocatoria.documentos.map((documento, indice) => {
    const descriptor = typeof documento === "object" && documento !== null
      ? documento
      : { titulo: String(documento), aviso: c("detalle.documentoPublico"), url: "" };
    const accion = descriptor.url
      ? `<a class="boton-secundario" href="${escaparAtributo(descriptor.url)}" target="_blank" rel="noopener">${escaparHTML(c("detalle.abrir", { formato: descriptor.formato || c("detalle.documento") }))}</a>`
      : botonOperacion("solicitar_descarga", c("detalle.descargar"), { id: convocatoria.id, clase: "boton-secundario", descripcion: c("detalle.prepararDescarga", { titulo: descriptor.titulo }) });
    return `<li><span class="fecha-bloque" aria-hidden="true">${indice + 1}</span><span><strong>${escaparHTML(descriptor.titulo)}</strong><small>${escaparHTML(descriptor.aviso || c("detalle.documentoPublico"))}</small></span>${accion}</li>`;
  }).join("");
  const acciones = `<button type="button" class="boton-secundario" data-accion="volver-convocatorias">${escaparHTML(c("detalle.volver"))}</button>`;
  return `${encabezadoVista(convocatoria.titulo, convocatoria.referencia, acciones)}
    <div class="rejilla-principal"><div>
      ${panel(c("detalle.resumen"), convocatoria.descripcion, listaDatos([
        [c("detalle.estado"), chip(convocatoria.estado)], [c("detalle.categoria"), escaparHTML(convocatoria.categoria)], [c("detalle.publicacion"), escaparHTML(convocatoria.publicada_en || c("detalle.noDisponible"))],
        [c("detalle.cve"), escaparHTML(convocatoria.cve_bop || convocatoria.referencia)], [c("detalle.plazo"), escaparHTML(convocatoria.plazo)],
        [c("detalle.presentacionHasta"), escaparHTML(convocatoria.presentacion_hasta)], [c("detalle.plazas"), escaparHTML(convocatoria.plazas)], [c("detalle.tasa"), escaparHTML(convocatoria.tasa)],
      ]), { estado: convocatoria.estado })}
      ${panel(c("detalle.requisitos"), c("detalle.requisitosSubtitulo"), `<ul>${requisitos}</ul><p class="nota aviso">${escaparHTML(c("detalle.requisitosNota"))}</p>`)}
    </div><aside>
      ${panel(c("detalle.documentacion"), c("detalle.documentacionSubtitulo"), `<ul class="lista-documentos">${documentos}</ul>`)}
      ${panel(c("detalle.antes"), c("detalle.antesSubtitulo"), `<ol><li>${escaparHTML(c("detalle.paso1"))}</li><li>${escaparHTML(c("detalle.paso2"))}</li><li>${escaparHTML(c("detalle.paso3"))}</li><li>${escaparHTML(c("detalle.paso4"))}</li></ol>`)}
    </aside></div>`;
}
