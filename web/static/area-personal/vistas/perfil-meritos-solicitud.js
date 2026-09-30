import {
  barraProgreso, botonOperacion, chip, cifraResumen, encabezadoVista, enlaceRuta, escaparAtributo,
  escaparHTML, formatoPuntos, listaDatos, panel, tabla,
} from "./comunes.js";
import { estadoActosSolicitud, localizarSolicitudEdicion } from "../flujo-solicitud.js";
import { calcularAutobaremo } from "../calculo-autobaremo.js";
import { textoContactoPropio } from "../i18n-contacto-propio.js";
import { traducir } from "../i18n.js";

const p = (clave, variables) => traducir(`areaPersonal.vista.perfil.${clave}`, variables);
const m = (clave, variables) => traducir(`areaPersonal.vista.meritos.${clave}`, variables);
const s = (clave, variables) => traducir(`areaPersonal.vista.solicitud.${clave}`, variables);
const a = (clave, variables) => traducir(`areaPersonal.vista.autobaremo.${clave}`, variables);
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
  const formulario = `<form id="formulario-merito" data-operacion="incorporar_merito"><div class="formulario-rejilla"><div class="campo"><label for="merito-tipo">${h(m("formulario.tipo"))}</label><select id="merito-tipo" name="tipo" required><option value="">${h(m("formulario.seleccione"))}</option>${opciones(TIPOS_MERITO, "tipos")}</select></div><div class="campo"><label for="merito-titulo">${h(m("formulario.denominacion"))}</label><input id="merito-titulo" name="titulo" required maxlength="180"></div><div class="campo"><label for="merito-jornada">${h(m("formulario.jornada"))}</label><select id="merito-jornada" name="jornada">${opciones(JORNADAS_MERITO, "jornadas")}</select></div><div class="campo"><label for="merito-documento">${h(m("formulario.documento"))}</label><input id="merito-documento" name="documento" type="file" accept=".pdf,.odt,.docx,.jpg,.png"><small>${h(m("formulario.documentoAyuda"))}</small></div></div><p class="nota aviso">${h(m("formulario.nota"))}</p><button type="submit" class="boton-primario">${h(m("formulario.revisar"))}</button></form>`;

  return `${encabezadoVista(m("titulo"), m("descripcion"), `<button type="button" class="boton-primario" data-accion="enfocar-nuevo-merito">${h(m("anadir"))}</button>`)}
    <section class="resumen-cifras">${cifraResumen(datos.meritos.length, m("cifras.inventariados"), m("cifras.inventariadosAyuda"), { clase: "merito", nombreIcono: "expediente" })}${cifraResumen(datos.meritos.filter((item) => item.estado === "Validado").length, m("cifras.validados"), m("cifras.validadosAyuda"), { clase: "exito", nombreIcono: "correcto" })}${cifraResumen(datos.meritos.filter((item) => item.estado !== "Validado").length, m("cifras.pendientes"), m("cifras.pendientesAyuda"), { clase: "aviso", nombreIcono: "pendiente" })}${cifraResumen(datos.documentos.length, m("cifras.documentos"), m("cifras.documentosAyuda"), { nombreIcono: "documento" })}</section>
    ${panel(m("inventario.titulo"), m("inventario.subtitulo"), tabla({ descripcion: m("inventario.tabla"), columnas: [m("columnas.merito"), m("columnas.tipo"), m("columnas.detalle"), m("columnas.estado"), m("columnas.evidencia")], filas: filasMeritos }))}
    <div class="rejilla-dos"><div>${panel(m("incorporar.titulo"), m("incorporar.subtitulo"), formulario, { clase: "panel-nuevo-merito" })}</div><div>${panel(m("documentos.titulo"), m("documentos.subtitulo"), tabla({ descripcion: m("documentos.tabla"), columnas: [m("columnas.documento"), m("columnas.tipo"), m("columnas.fecha"), m("columnas.estado"), m("columnas.accion")], filas: filasDocumentos }))}</div></div>`;
}

function pasosSolicitud(paso) {
  const etiquetas = ["convocatoria", "datos", "meritos", "autobaremo", "registro"].map((clave) => s(`pasos.${clave}`));
  return `<ol class="pasos" aria-label="${escaparAtributo(s("pasos.titulo"))}">${etiquetas.map((etiqueta, indice) => `<li class="${paso === indice + 1 ? "activo" : paso > indice + 1 ? "completo" : ""}" ${paso === indice + 1 ? 'aria-current="step"' : ""}><span>${h(s("pasos.numero", { numero: indice + 1 }))}</span>${escaparHTML(etiqueta)}</li>`).join("")}</ol>`;
}

function accionesPaso(paso) {
  const etiqueta = paso === 4 ? s("guardarBorradorContinuar") : s("guardarContinuar");
  return `<div class="fila-acciones">${paso > 1 ? `<button type="button" class="boton-secundario" data-accion="paso-anterior">${h(s("anterior"))}</button>` : ""}${paso < 5 ? `<button type="submit" class="boton-primario">${h(etiqueta)}</button>` : ""}</div>`;
}

function meritosAutobaremacion(datos, estado, convocatoriaId) {
  if (datos.resultado_autobaremo?.convocatoria_id === convocatoriaId
    && Array.isArray(datos.resultado_autobaremo.meritos_ids)) {
    return datos.resultado_autobaremo.meritos_ids;
  }
  if (estado.progresoSolicitud?.convocatoria_id === convocatoriaId
    && estado.progresoSolicitud.meritos_ids?.length) {
    return estado.progresoSolicitud.meritos_ids;
  }
  const borrador = localizarSolicitudEdicion(datos, {
    solicitudId: estado.solicitudEdicionId,
    convocatoriaId,
  });
  if (borrador?.meritos_ids?.length) return borrador.meritos_ids;
  return datos.meritos.map((item) => item.id);
}

function declaracion(nombre, marcado, titulo, detalle) {
  return `<label class="opcion-check"><input type="checkbox" name="${nombre}" value="true" required ${marcado ? "checked" : ""}><span><strong>${h(titulo)}</strong><small>${h(detalle)}</small></span></label>`;
}

function contenidoPaso(datos, estado, convocatoria) {
  const paso = estado.pasoSolicitud;
  if (paso === 1) {
    const disponibles = datos.convocatorias.filter((item) => item.estado === "Plazo abierto");
    const requisitosConfirmados = estado.progresoSolicitud?.requisitos_confirmados === true;
    return `<fieldset><legend>${h(s("paso1.leyenda"))}</legend>${disponibles.map((item) => `<label class="opcion-check"><input type="radio" name="convocatoria" value="${escaparAtributo(item.id)}" ${item.id === convocatoria.id ? "checked" : ""} required data-accion="seleccionar-convocatoria"><span><strong>${escaparHTML(item.titulo)}</strong><small>${escaparHTML(item.referencia)} · ${h(s("paso1.comprobacion"))}</small></span></label>`).join("")}</fieldset>${declaracion("requisitos_confirmados", requisitosConfirmados, s("paso1.declaracion"), s("paso1.declaracionDetalle"))}`;
  }
  if (paso === 2) {
    return `${listaDatos([[s("paso2.identidad"), escaparHTML(datos.perfil.nombre_visible)], [s("paso2.identificador"), escaparHTML(datos.perfil.identificador_visible)], [s("paso2.correo"), escaparHTML(datos.perfil.correo)], [s("paso2.telefono"), escaparHTML(datos.perfil.telefono)], [s("paso2.verificacion"), chip(datos.perfil.estado_verificacion)]])}${declaracion("datos_confirmados", estado.progresoSolicitud?.datos_confirmados === true, s("paso2.declaracion"), s("paso2.declaracionDetalle"))}`;
  }
  if (paso === 3) {
    const seleccionados = new Set(estado.progresoSolicitud?.meritos_ids || []);
    return `<fieldset><legend>${h(s("paso3.leyenda"))}</legend>${datos.meritos.map((item) => `<label class="opcion-check"><input type="checkbox" name="meritos" value="${escaparAtributo(item.id)}" ${seleccionados.has(item.id) ? "checked" : ""}><span><strong>${escaparHTML(item.titulo)}</strong><small>${escaparHTML(item.estado)} · ${h(m("puntosEstimados", { puntos: formatoPuntos(item.puntos_estimados) }))}</small></span></label>`).join("")}</fieldset><p class="nota aviso">${h(s("paso3.nota"))}</p>`;
  }
  if (paso === 4) {
    const calculo = calcularAutobaremo(datos, estado.progresoSolicitud?.meritos_ids);
    return `<div class="rejilla-dos"><div>${calculo.criterios.map((item) => `<div class="criterio-baremo"><span><strong>${escaparHTML(item.nombre)}</strong><small>${escaparHTML(item.detalle)}</small></span>${barraProgreso(item.puntos, item.maximo)}<output>${formatoPuntos(item.puntos)}</output></div>`).join("")}</div><aside class="puntuacion-total"><span>${h(s("paso4.total"))}</span><output>${formatoPuntos(calculo.total)}</output><span>${h(s("paso4.provisionales"))}</span></aside></div><p class="nota aviso">${h(s("paso4.nota"))}</p>`;
  }
  const solicitud = localizarSolicitudEdicion(datos, {
    solicitudId: estado.solicitudEdicionId,
    convocatoriaId: convocatoria.id,
  });
  if (!solicitud) return `<p class="nota error" role="alert"><strong>${h(s("paso5.sinBorrador"))}</strong> ${h(s("paso5.sinBorradorDetalle"))}</p>`;
  const actos = estadoActosSolicitud(solicitud);
  const pago = actos.pagoConfirmado
    ? chip(s("paso5.pagoConfirmado"))
    : botonOperacion("iniciar_pago", s("paso5.pagar"), { id: solicitud.id, descripcion: s("paso5.pagarDescripcion") });
  const firma = actos.firmaConfirmada
    ? chip(s("paso5.firmaConfirmada"))
    : actos.pagoConfirmado
      ? botonOperacion("firmar_solicitud", s("paso5.firmar"), { id: solicitud.id, descripcion: s("paso5.firmarDescripcion") })
      : `<button type="button" class="boton-secundario" disabled aria-disabled="true" title="${escaparAtributo(s("paso5.firmaBloqueadaAyuda"))}">${h(s("paso5.firmaBloqueada"))}</button>`;
  let registro = `<p class="nota"><strong>${h(s("paso5.registrada"))}</strong> ${h(s("paso5.registradaDetalle"))}</p>`;
  if (!actos.registrada) {
    registro = actos.pagoConfirmado && actos.firmaConfirmada
      ? `<form id="formulario-registro-solicitud" data-operacion="registrar_solicitud" data-id="${escaparAtributo(solicitud.id)}"><label class="opcion-check"><input type="checkbox" name="declaracion_final" value="true" required><span><strong>${h(s("paso5.declaracion"))}</strong><small>${h(s("paso5.declaracionDetalle"))}</small></span></label><button type="submit" class="boton-primario">${h(s("paso5.registrar"))}</button></form>`
      : `<p class="nota aviso"><strong>${h(s("paso5.registroBloqueado"))}</strong> ${h(s("paso5.registroBloqueadoDetalle"))}</p>`;
  }
  return `<p class="nota"><strong>${h(s("paso5.borrador"))}</strong> ${escaparHTML(solicitud.id)}</p><div class="rejilla-dos"><section><h3>${h(s("paso5.tasa"))}</h3><p>${h(s("paso5.importe"))} <strong>${escaparHTML(convocatoria.tasa)}</strong></p>${pago}</section><section><h3>${h(s("paso5.firma"))}</h3><p>${h(s("paso5.firmaDetalle"))}</p>${firma}</section></div><section class="panel separacion-superior"><div class="panel-contenido"><h3>${h(s("paso5.registro"))}</h3><p>${h(s("paso5.registroDetalle"))}</p>${registro}</div></section>`;
}

export function renderizarSolicitud(datos, estado) {
  const convocatoria = datos.convocatorias.find((item) => item.id === estado.convocatoriaSolicitud)
    || datos.convocatorias.find((item) => item.estado === "Plazo abierto")
    || datos.convocatorias[0];
  const solicitud = localizarSolicitudEdicion(datos, {
    solicitudId: estado.solicitudEdicionId,
    convocatoriaId: convocatoria.id,
  });
  const error = estado.errorPasoSolicitud
    ? `<p class="nota error" role="alert"><strong>${h(s("error"))}</strong> ${escaparHTML(estado.errorPasoSolicitud)}</p>`
    : "";
  const contenido = `${pasosSolicitud(estado.pasoSolicitud)}${error}<section class="panel"><header><div><h3>${h(s("pasoDe", { paso: estado.pasoSolicitud, total: 5 }))}</h3></div>${chip(estado.pasoSolicitud === 5 ? s("revisionFinal") : s("enPreparacion"))}</header><div class="panel-contenido">${contenidoPaso(datos, estado, convocatoria)}</div></section>`;
  const asistente = estado.pasoSolicitud < 5
    ? `<form id="formulario-solicitud-paso" data-paso="${estado.pasoSolicitud}">${contenido}${accionesPaso(estado.pasoSolicitud)}</form>`
    : contenido;
  return `${encabezadoVista(s("titulo"), `${convocatoria.titulo} · ${convocatoria.referencia}`, chip(solicitud?.estado || s("sinGuardar")))}${asistente}`;
}

export function renderizarAutobaremacion(datos, estado = {}) {
  const convocatoria = datos.convocatorias.find((item) => item.id === estado.convocatoriaSolicitud)
    || datos.convocatorias.find((item) => item.estado === "Plazo abierto")
    || datos.convocatorias[0];
  const meritosIds = meritosAutobaremacion(datos, estado, convocatoria?.id || "");
  const calculo = calcularAutobaremo(datos, meritosIds);
  const criterios = calculo.criterios.map((criterio) => `<article class="criterio-baremo"><span><strong>${escaparHTML(criterio.nombre)}</strong><small>${escaparHTML(criterio.detalle)} · ${escaparHTML(criterio.estado)}</small></span>${barraProgreso(criterio.puntos, criterio.maximo)}<output>${formatoPuntos(criterio.puntos)}</output></article>`).join("");
  const recalculado = datos.resultado_autobaremo?.convocatoria_id === convocatoria?.id
    ? `<p class="nota"><strong>${h(a("recalculado"))}</strong> ${escaparHTML(datos.resultado_autobaremo.calculado_en)} · ${h(a("meritos", { cuenta: meritosIds.length }))}</p>`
    : "";
  return `${encabezadoVista(a("titulo"), a("descripcion"), botonOperacion("calcular_autobaremo", a("recalcular"), { id: convocatoria?.id || "", descripcion: a("recalcularDescripcion") }))}
    ${recalculado}<div class="rejilla-principal"><div>${panel(a("criterios"), a("criteriosSubtitulo", { convocatoria: convocatoria?.titulo || a("convocatoriaSeleccionada") }), `<div class="desglose-baremo">${criterios}</div>`)}</div><aside>${panel(a("resultado"), a("resultadoSubtitulo", { cuenta: meritosIds.length }), `<div class="puntuacion-total"><span>${h(a("estimada"))}</span><output>${formatoPuntos(calculo.total)}</output><span>${h(a("deMaximo", { maximo: formatoPuntos(calculo.maximo) }))}</span></div><p>${barraProgreso(calculo.total, calculo.maximo)}</p><p class="nota aviso">${h(a("nota"))}</p>`, { estado: a("provisional") })}${panel(a("reglas"), a("reglasSubtitulo"), `<ul><li>${h(a("regla1"))}</li><li>${h(a("regla2"))}</li><li>${h(a("regla3"))}</li><li>${h(a("regla4"))}</li><li>${h(a("regla5"))}</li></ul>`)}</aside></div>`;
}
