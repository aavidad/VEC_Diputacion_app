import {
  botonOperacion, chip, cifraResumen, encabezadoVista, enlaceRuta, escaparAtributo,
  escaparHTML, formatoPuntos, panel, tabla,
} from "./comunes.js";
import { textoContactoPropio } from "../i18n-contacto-propio.js";
import { traducir } from "../i18n.js";

const p = (clave, variables) => traducir(`areaPersonal.vista.perfil.${clave}`, variables);
const m = (clave, variables) => traducir(`areaPersonal.vista.meritos.${clave}`, variables);
const h = (texto) => escaparHTML(texto);

// Códigos de las opciones del formulario de méritos; su nombre visible está en el catálogo.
const TIPOS_MERITO = Object.freeze(["titulacion", "experiencia", "formacion", "ejercicio", "otro"]);
const JORNADAS_MERITO = Object.freeze(["no_corresponde", "completa", "parcial_50", "parcial_33", "otra"]);

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

export function renderizarMeritos(datos) {
  const filasMeritos = datos.meritos.map((merito) => [
    `<strong>${escaparHTML(merito.titulo)}</strong><small>${escaparHTML(merito.id)}</small>`,
    escaparHTML(merito.tipo),
    `<span>${escaparHTML(merito.detalle)}</span><small>${h(m("puntosEstimados", { puntos: formatoPuntos(merito.puntos_estimados) }))}</small>`,
    chip(merito.estado),
    `<div class="acciones-tabla"><button type="button" class="boton-secundario" data-accion="abrir-documento" data-id="${escaparAtributo(merito.documento_ref)}">${h(m("verEvidencia"))}</button></div>`,
  ]);
  const filasDocumentos = datos.documentos.map((documento) => [
    `<strong>${escaparHTML(documento.nombre)}</strong><small>${escaparHTML(documento.id)}</small>`,
    escaparHTML(documento.tipo), escaparHTML(documento.fecha), chip(documento.estado),
    `<div class="acciones-tabla">${botonOperacion("solicitar_descarga", m("descargar"), { id: documento.id, clase: "boton-secundario", descripcion: m("prepararDescarga", { nombre: documento.nombre }) })}</div>`,
  ]);
  const opciones = (codigos, grupo) => codigos.map((codigo) => `<option value="${codigo}">${h(m(`${grupo}.${codigo}`))}</option>`).join("");
  const formulario = `<form id="formulario-merito" data-operacion="incorporar_merito"><div class="formulario-rejilla"><div class="campo"><label for="merito-tipo">${h(m("formulario.tipo"))} <span>${h(traducir("areaPersonal.vista.comun.campoObligatorio"))}</span></label><select id="merito-tipo" name="tipo" required><option value="">${h(m("formulario.seleccione"))}</option>${opciones(TIPOS_MERITO, "tipos")}</select></div><div class="campo"><label for="merito-titulo">${h(m("formulario.denominacion"))} <span>${h(traducir("areaPersonal.vista.comun.campoObligatorio"))}</span></label><input id="merito-titulo" name="titulo" required maxlength="180"></div><div class="campo"><label for="merito-jornada">${h(m("formulario.jornada"))} <span>${h(traducir("areaPersonal.vista.comun.campoOpcional"))}</span></label><select id="merito-jornada" name="jornada">${opciones(JORNADAS_MERITO, "jornadas")}</select></div><div class="campo"><label for="merito-documento">${h(m("formulario.documento"))} <span>${h(traducir("areaPersonal.vista.comun.campoOpcional"))}</span></label><input id="merito-documento" name="documento" type="file" accept=".pdf,.odt,.docx,.jpg,.png" aria-describedby="merito-documento-ayuda"><small id="merito-documento-ayuda">${h(m("formulario.documentoAyuda"))}</small></div></div><p class="nota aviso">${h(m("formulario.nota"))}</p><button type="submit" class="boton-primario">${h(m("formulario.revisar"))}</button></form>`;

  return `${encabezadoVista(m("titulo"), m("descripcion"), `<button type="button" class="boton-primario" data-accion="enfocar-nuevo-merito">${h(m("anadir"))}</button>`)}
    <section class="resumen-cifras">${cifraResumen(datos.meritos.length, m("cifras.inventariados"), m("cifras.inventariadosAyuda"), { clase: "merito", nombreIcono: "expediente" })}${cifraResumen(datos.meritos.filter((item) => item.estado === "Validado").length, m("cifras.validados"), m("cifras.validadosAyuda"), { clase: "exito", nombreIcono: "correcto" })}${cifraResumen(datos.meritos.filter((item) => item.estado !== "Validado").length, m("cifras.pendientes"), m("cifras.pendientesAyuda"), { clase: "aviso", nombreIcono: "pendiente" })}${cifraResumen(datos.documentos.length, m("cifras.documentos"), m("cifras.documentosAyuda"), { nombreIcono: "documento" })}</section>
    ${panel(m("inventario.titulo"), m("inventario.subtitulo"), tabla({ descripcion: m("inventario.tabla"), columnas: [m("columnas.merito"), m("columnas.tipo"), m("columnas.detalle"), m("columnas.estado"), m("columnas.evidencia")], filas: filasMeritos }))}
    <div class="rejilla-dos"><div>${panel(m("incorporar.titulo"), m("incorporar.subtitulo"), formulario, { clase: "panel-nuevo-merito" })}</div><div>${panel(m("documentos.titulo"), m("documentos.subtitulo"), tabla({ descripcion: m("documentos.tabla"), columnas: [m("columnas.documento"), m("columnas.tipo"), m("columnas.fecha"), m("columnas.estado"), m("columnas.accion")], filas: filasDocumentos }))}</div></div>`;
}
