import {
  chip, encabezadoVista, escaparAtributo, escaparHTML, listaDatos, panel,
} from "./comunes.js";
import { localizacionAreaPersonal, traducir } from "../i18n.js";
import { campoVisibleMiBolsa, nombreCategoria } from "../mi-bolsa-campos.js";
import { renderizarPortalMiBolsa, textoPortal } from "../mi-bolsa-portal.js?v=20261002-rrhh17-v1";
import { renderizarOfertasMiBolsa, textoOfertas } from "../mi-bolsa-ofertas.js";
import { renderizarContactoMiBolsa, textoContacto } from "../mi-bolsa-contacto.js";
import { renderizarHistorialMiBolsa } from "../mi-bolsa-historial.js";

const b = (clave, variables) => traducir(`areaPersonal.vista.miBolsa.${clave}`, variables);
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
