import { chip, encabezadoVista, escaparAtributo, escaparHTML, listaDatos, panel } from "./comunes.js";
import { localizacionAreaPersonal, traducir } from "../i18n.js";
import { campoVisibleMiBolsa, nombreCategoria } from "../mi-bolsa-campos.js";
import { renderizarPortalMiBolsa, textoPortal } from "../mi-bolsa-portal.js?v=20261009-retoques-textos-v1";
import { renderizarOfertasMiBolsa, textoOfertas } from "../mi-bolsa-ofertas.js";
import { renderizarContactoMiBolsa, textoContacto } from "../mi-bolsa-contacto.js";
import { renderizarHistorialMiBolsa } from "../mi-bolsa-historial.js?v=20261009-retoques-textos-v1";

const b = (clave, variables) => traducir(`areaPersonal.vista.miBolsa.${clave}`, variables);
const u = (clave, variables) => traducir(`areaPersonal.vista.subsanaciones.${clave}`, variables);
const g = (clave, variables) => traducir(`areaPersonal.vista.alegaciones.${clave}`, variables);
const h = (texto) => escaparHTML(texto);

const CLASES_SITUACION = Object.freeze({
  disponible: "exito", no_disponible: "aviso", trabajando: "info",
  pendiente_incorporacion: "aviso", renuncia: "aviso", excluido: "error",
  disponible_desde: "aviso", en_revision: "merito",
});

function fechaSituacion(valor) {
  return new Intl.DateTimeFormat(localizacionAreaPersonal(), { dateStyle: "medium", timeStyle: "short", timeZone: "Europe/Madrid" }).format(new Date(valor));
}

function fichaSituacionActual(actual) {
  if (!actual || !Object.hasOwn(CLASES_SITUACION, actual.estado)) {
    return `<p class="nota aviso">${escaparHTML(traducir("areaPersonal.miBolsa.situacion.sinDato"))}</p>`;
  }
  const etiqueta = traducir(`areaPersonal.miBolsa.situacion.${actual.estado}`);
  const filas = [
    [traducir("areaPersonal.miBolsa.situacion.estado"), `<span class="estado-chip ${CLASES_SITUACION[actual.estado]}">${escaparHTML(etiqueta)}</span>`],
    [traducir("areaPersonal.miBolsa.situacion.desde"), escaparHTML(fechaSituacion(actual.desde))],
    [traducir("areaPersonal.miBolsa.situacion.hasta"), actual.hasta ? escaparHTML(fechaSituacion(actual.hasta)) : escaparHTML(traducir("areaPersonal.miBolsa.situacion.sinFin"))],
  ];
  if (actual.fecha_disponible) filas.push([traducir("areaPersonal.miBolsa.situacion.fechaDisponible"), escaparHTML(fechaSituacion(actual.fecha_disponible))]);
  filas.push([traducir("areaPersonal.miBolsa.situacion.explicacion"), escaparHTML(traducir(`areaPersonal.miBolsa.situacion.explicacion.${actual.estado}`))]);
  return listaDatos(filas);
}

export function renderizarLlamamientos(datos, estado = {}) {
  const participaciones = Array.isArray(estado.participaciones) && estado.participaciones.length
    ? estado.participaciones : datos.posicion ? [{ bolsa: datos.posicion.bolsa, categoria: datos.posicion.categoria, orden_inicial: datos.posicion.orden, total_instantanea: datos.posicion.total, version: "—", estado_bolsa: b("sinDatos"), vigente_desde: datos.posicion.vigente_desde, vigente_hasta: null }] : [];
  const pagina = Math.max(1, Number(estado.paginaParticipaciones || 1));
  const porPagina = [20, 50, 100].includes(estado.filasPreferidas) ? estado.filasPreferidas : 20;
  const totalPaginas = Math.max(1, Math.ceil(participaciones.length / porPagina));
  const paginaActual = Math.min(pagina, totalPaginas);
  const visibles = participaciones.slice((paginaActual - 1) * porPagina, paginaActual * porPagina);
  const ver = (campo) => campoVisibleMiBolsa(estado.camposMiBolsa, campo);
  const tarjetasParticipacion = visibles.map((item) => panel(b("participacion"), nombreCategoria(item), `${listaDatos([
    ...(ver("estado") ? [[b("estadoBolsa"), chip(item.estado_bolsa)]] : []),
    ...(ver("posicion") ? [[traducir("areaPersonal.miBolsa.ordenInicial"), h(b("ordenDe", { orden: String(item.orden_inicial), total: String(item.total_instantanea) }))]] : []),
    [traducir("areaPersonal.miBolsa.vigenciaBolsa"), h(item.vigente_hasta ? b("vigenciaHasta", { desde: fechaSituacion(item.vigente_desde), hasta: fechaSituacion(item.vigente_hasta) }) : b("vigenciaAbierta", { desde: fechaSituacion(item.vigente_desde) }))],
    ...(!ver("estado") && ver("fecha_disponible") && item.situacion_actual?.fecha_disponible ? [[traducir("areaPersonal.miBolsa.situacion.fechaDisponible"), escaparHTML(fechaSituacion(item.situacion_actual.fecha_disponible))]] : []),
  ])}${ver("estado") ? `<h4>${escaparHTML(traducir("areaPersonal.miBolsa.situacion.titulo"))}</h4>${fichaSituacionActual(item.situacion_actual)}` : ""}`, { estado: ver("estado") ? item.estado_bolsa : "", clase: "participacion-propia" })).join("");
  const paginacion = participaciones.length > porPagina ? `<nav class="paginacion-participaciones" aria-label="${escaparAtributo(b("paginacion"))}"><span>${h(b("mostrando", { desde: (paginaActual - 1) * porPagina + 1, hasta: Math.min(paginaActual * porPagina, participaciones.length), total: participaciones.length }))}</span><button type="button" class="boton-secundario" data-accion="pagina-participaciones" data-pagina="${paginaActual - 1}" ${paginaActual === 1 ? "disabled" : ""}>${h(b("anterior"))}</button><button type="button" class="boton-secundario" data-accion="pagina-participaciones" data-pagina="${paginaActual + 1}" ${paginaActual === totalPaginas ? "disabled" : ""}>${h(b("siguiente"))}</button></nav>` : "";
  const fichaParticipaciones = participaciones.length ? `<section class="marco-participaciones" aria-label="${escaparAtributo(b("participacionesEtiqueta"))}">${tarjetasParticipacion}${paginacion}</section>` : panel(b("participaciones"), b("sinParticipaciones"), `<p>${h(b("sinParticipacionesDetalle"))}</p>`);
  const propios = participaciones.filter((item) => item.ultimo_llamamiento);
  propios.sort((a, b) => b.ultimo_llamamiento.emitido_en.localeCompare(a.ultimo_llamamiento.emitido_en) || a.categoria.localeCompare(b.categoria) || a.bolsa.localeCompare(b.bolsa));
  const ultimo = propios[0];
  const resultado = ultimo?.ultimo_llamamiento.resultado;
  const detalle = ultimo ? `${listaDatos([
    [traducir("areaPersonal.miBolsa.llamamiento.bolsa"), escaparHTML(ultimo.bolsa)],
    [traducir("areaPersonal.miBolsa.llamamiento.categoria"), escaparHTML(nombreCategoria(ultimo))],
    [traducir("areaPersonal.miBolsa.llamamiento.fecha"), escaparHTML(fechaSituacion(ultimo.ultimo_llamamiento.emitido_en))],
    [traducir("areaPersonal.miBolsa.llamamiento.canal"), escaparHTML(traducir("areaPersonal.miBolsa.llamamiento.correo"))],
    [traducir("areaPersonal.miBolsa.llamamiento.resultado"), `<span class="estado-chip ${resultado === "enviado" ? "info" : "aviso"}">${escaparHTML(traducir(`areaPersonal.miBolsa.llamamiento.${resultado}`))}</span>`],
  ])}<p class="nota aviso">${escaparHTML(traducir("areaPersonal.miBolsa.llamamiento.limite"))}</p>`
    : `<p class="nota aviso">${escaparHTML(traducir("areaPersonal.miBolsa.llamamiento.sinDato"))}</p>`;
  const llamamientos = panel(traducir("areaPersonal.miBolsa.llamamiento.titulo"), traducir("areaPersonal.miBolsa.llamamiento.subtitulo"), detalle);

  return `${encabezadoVista(b("titulo"), "")}
    ${fichaParticipaciones}
    <div id="historial-mi-bolsa" aria-live="polite">${renderizarHistorialMiBolsa()}</div>
    <div class="rejilla-principal"><div>${ver("ultimo_llamamiento") ? llamamientos : ""}${estado.ofertasMiBolsa?.length ? panel(textoOfertas("titulo"), textoOfertas("subtitulo"), renderizarOfertasMiBolsa(estado.ofertasMiBolsa)) : ""}</div><aside>
      ${estado.accionesPortal
    ? panel(textoPortal("titulo"), textoPortal("subtitulo"), renderizarPortalMiBolsa(participaciones, estado.portalMiBolsa, estado.accionesPortal))
    : panel(traducir("areaPersonal.miBolsa.disponibilidad.titulo"), traducir("areaPersonal.miBolsa.disponibilidad.subtitulo"), `<p class="nota aviso">${escaparHTML(traducir("areaPersonal.miBolsa.disponibilidad.detalle"))}</p>`)}
      ${estado.contactosMiBolsa?.length ? panel(textoContacto("titulo"), textoContacto("subtitulo"), renderizarContactoMiBolsa(participaciones, estado.contactosMiBolsa)) : ""}
    </aside></div>`;
}

export function renderizarSubsanaciones(datos) {
  const formularios = datos.subsanaciones.map((item) => `<article class="panel"><header><div><h3>${escaparHTML(item.motivo)}</h3><p>${escaparHTML(item.id)} · ${escaparHTML(item.solicitud_ref)}</p></div>${chip(item.estado)}</header><div class="panel-contenido">${listaDatos([[u("documentoSolicitado"), escaparHTML(item.documento_solicitado)], [u("plazo"), escaparHTML(item.plazo)], [u("estado"), chip(item.estado)]])}${item.estado === "Pendiente" ? `<form data-operacion="presentar_subsanacion" data-id="${escaparAtributo(item.id)}"><div class="formulario-rejilla"><div class="campo ancho-completo"><label for="subsanacion-${escaparAtributo(item.id)}">${h(u("explicacion"))}</label><textarea id="subsanacion-${escaparAtributo(item.id)}" name="explicacion" required maxlength="1000">${h(u("explicacionInicial"))}</textarea></div><div class="campo ancho-completo"><label for="fichero-${escaparAtributo(item.id)}">${h(u("documento"))}</label><input id="fichero-${escaparAtributo(item.id)}" name="documento" type="file" accept=".pdf,.odt,.docx,.jpg,.png" required><small>${h(u("documentoAyuda"))}</small></div></div><label class="opcion-check"><input type="checkbox" name="declaracion" required><span><strong>${h(u("declaracion"))}</strong><small>${h(u("declaracionDetalle"))}</small></span></label><button type="submit" class="boton-primario">${h(u("presentar"))}</button></form>` : `<p class="nota">${h(u("sinActuacion"))}</p>`}</div></article>`).join("");
  return `${encabezadoVista(u("titulo"), u("descripcion"))}${formularios || panel(u("vacio.titulo"), u("vacio.subtitulo"), `<p>${h(u("vacio.detalle"))}</p>`)}`;
}

export function renderizarAlegaciones(datos) {
  const tarjetas = datos.alegaciones.map((item) => `<article class="panel"><header><div><h3>${escaparHTML(item.asunto)}</h3><p>${escaparHTML(item.id)} · ${escaparHTML(item.solicitud_ref)}</p></div>${chip(item.estado)}</header><div class="panel-contenido">${listaDatos([[g("fecha"), escaparHTML(item.fecha)], [g("estado"), chip(item.estado)]])}${item.estado === "Borrador" ? `<form data-operacion="presentar_alegacion" data-id="${escaparAtributo(item.id)}"><div class="campo"><label for="alegacion-${escaparAtributo(item.id)}">${h(g("fundamento"))}</label><textarea id="alegacion-${escaparAtributo(item.id)}" name="fundamento" required maxlength="2000">${h(g("fundamentoInicial"))}</textarea><small>${h(g("fundamentoAyuda"))}</small></div><div class="campo"><label for="evidencia-${escaparAtributo(item.id)}">${h(g("evidencia"))}</label><input id="evidencia-${escaparAtributo(item.id)}" name="documento" type="file" accept=".pdf,.odt,.docx,.jpg,.png"></div><label class="opcion-check"><input type="checkbox" name="declaracion" required><span><strong>${h(g("declaracion"))}</strong><small>${h(g("declaracionDetalle"))}</small></span></label><button type="submit" class="boton-primario">${h(g("presentar"))}</button></form>` : `<p class="nota">${h(g("consta", { estado: item.estado }))}</p>`}</div></article>`).join("");
  return `${encabezadoVista(g("titulo"), g("descripcion"))}${tarjetas || panel(g("vacio.titulo"), g("vacio.subtitulo"), `<p>${h(g("vacio.detalle"))}</p>`)}`;
}
