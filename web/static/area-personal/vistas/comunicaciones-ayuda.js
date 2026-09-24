import {
  botonOperacion, chip, encabezadoVista, enlaceRuta, escaparAtributo, escaparHTML,
  estadoVacio, panel, tabla,
} from "./comunes.js";
import { traducir } from "../i18n.js";

export function renderizarMensajes(datos) {
  const preferenciasAviso = datos.preferencias_notificacion || {};
  const listaMensajes = Array.isArray(datos.mensajes) ? datos.mensajes : null;
  const mensajes = listaMensajes === null
    ? '<p class="nota error" role="alert"><strong>No se puede mostrar la bandeja.</strong> No se ha realizado ninguna operación ni se ha enviado aviso alguno.</p>'
    : listaMensajes.map((item) => `<li><span class="fecha-bloque" aria-hidden="true">${item.estado === "No leído" ? "!" : "·"}</span><span><strong>${escaparHTML(item.asunto)}</strong><small>${escaparHTML(item.resumen)}</small><small>${escaparHTML(item.fecha)} · ${escaparHTML(item.tipo)}</small></span><div class="fila-acciones">${item.estado === "No leído" ? botonOperacion("marcar_mensaje", "Marcar como leído", { id: item.id, clase: "boton-secundario", descripcion: `Marcar como leído: ${item.asunto}` }) : chip(item.estado)}${enlaceRuta(item.ruta, "Abrir asunto", "enlace-boton")}</div></li>`).join("");
  const bandeja = listaMensajes === null ? mensajes : listaMensajes.length ? `<ul class="lista-mensajes">${mensajes}</ul>` : estadoVacio("No hay mensajes", "Cuando exista una comunicación accesible para usted aparecerá aquí.");
  const preferencias = `<p class="nota aviso" role="status">Los canales de aviso no están conectados en esta superficie. Guardar una preferencia no configura un envío, una entrega ni una notificación administrativa.</p><form data-operacion="actualizar_notificaciones"><fieldset><legend>Avisos que desea recibir cuando el canal autorizado esté disponible</legend><label class="opcion-check"><input type="checkbox" name="convocatorias" ${preferenciasAviso.convocatorias ? "checked" : ""}><span><strong>Nuevas convocatorias de categorías compatibles</strong><small>Según titulaciones validadas y sus preferencias.</small></span></label><label class="opcion-check"><input type="checkbox" name="plazos" ${preferenciasAviso.plazos ? "checked" : ""}><span><strong>Plazos de mis expedientes</strong><small>Subsanaciones, alegaciones y publicaciones.</small></span></label><label class="opcion-check"><input type="checkbox" name="llamamientos" ${preferenciasAviso.llamamientos ? "checked" : ""}><span><strong>Llamamientos personales</strong><small>Aviso complementario sujeto al canal autorizado.</small></span></label><label class="opcion-check"><input type="checkbox" name="noticias" ${preferenciasAviso.noticias ? "checked" : ""}><span><strong>Noticias generales de empleo público</strong><small>Información sin efectos administrativos.</small></span></label></fieldset><button type="submit" class="boton-secundario">Guardar preferencias</button></form>`;
  return `${encabezadoVista("Mensajes, avisos y noticias", "Bandeja personal con el proceso relacionado, plazo y canal.")}
    <div class="rejilla-principal"><div>${panel("Bandeja de entrada", "Las notificaciones administrativas se identificarán expresamente", bandeja, { estado: `${datos.resumen?.mensajes_no_leidos || 0} no leídos` })}</div><aside aria-label="Preferencias y alcance de los avisos">${panel("Preferencias", "Configure avisos complementarios", preferencias)}${panel("Diferencia importante", "Aviso frente a notificación", `<p class="nota aviso">Un aviso no acredita una notificación administrativa. Cuando esta sea exigible, el portal mostrará expresamente su carácter, acceso, fecha y efectos.</p>`)}</aside></div>`;
}

export function renderizarCertificados(datos) {
  const listaCertificados = Array.isArray(datos.certificados) ? datos.certificados : null;
  const certificados = (listaCertificados || []).map((item) => {
    const formatos = item.formatos.split(/, | o /);
    return `<article class="panel"><header><div><h3>${escaparHTML(item.tipo)}</h3><p><small>Referencia técnica: ${escaparHTML(item.id)}</small></p></div>${chip(item.estado)}</header><div class="panel-contenido"><p>${escaparHTML(item.descripcion)}</p><form class="fila-acciones" data-operacion="solicitar_certificado" data-id="${escaparAtributo(item.id)}"><div class="campo"><label for="formato-${escaparAtributo(item.id)}">Formato</label><select id="formato-${escaparAtributo(item.id)}" name="formato">${formatos.map((formato) => `<option>${escaparHTML(formato)}</option>`).join("")}</select></div><button type="submit" class="boton-primario">Solicitar certificado</button></form></div></article>`;
  }).join("");
  const filas = (Array.isArray(datos.documentos) ? datos.documentos : []).map((item) => [
    `<strong>${escaparHTML(item.nombre)}</strong><small>${escaparHTML(item.id)}</small>`, escaparHTML(item.tipo),
    escaparHTML(item.fecha), chip(item.estado),
    `<div class="acciones-tabla">${botonOperacion("solicitar_descarga", "Descargar", { id: item.id, clase: "boton-secundario", descripcion: `Preparar descarga de ${item.nombre}` })}</div>`,
  ]);
  return `${encabezadoVista("Certificados y descargas", "Obtenga documentos en formatos configurados y consulte su procedencia.")}
    <p class="nota aviso">Un certificado solo podrá presentarse como oficial cuando incluya firma o sello, CSV/QR verificable, versión y, cuando corresponda, vigencia o revocación.</p>
    <div class="rejilla-dos">${listaCertificados === null ? '<p class="nota error" role="alert"><strong>No se pueden mostrar los certificados.</strong> Inténtelo de nuevo cuando el servicio autorizado esté disponible.</p>' : certificados || estadoVacio("No hay certificados disponibles", "No se ha preparado ningún certificado para esta identidad.")}</div>
    ${panel("Mis documentos", "Descargas autorizadas asociadas a la identidad", tabla({ descripcion: "Documentación disponible para la persona interesada", columnas: ["Documento", "Tipo", "Fecha", "Estado", "Acción"], filas }))}`;
}

export function renderizarAyuda(datos, estado = {}) {
  const termino = (estado.consultaAyuda || "").trim().toLowerCase();
  const ayuda = Array.isArray(datos.ayuda) ? datos.ayuda : null;
  const preguntas = (ayuda || []).filter((item) => !termino || `${item.pregunta} ${item.respuesta}`.toLowerCase().includes(termino));
  const faq = ayuda === null ? '<p class="nota error" role="alert"><strong>La ayuda no está disponible.</strong> No se ha enviado información a ningún servicio externo.</p>' : preguntas.map((item) => `<details><summary>${escaparHTML(item.pregunta)}</summary><p>${escaparHTML(item.respuesta)}</p></details>`).join("") || estadoVacio("Sin coincidencias", "Pruebe con otras palabras o borre la búsqueda.");
  const transcripcion = `${traducir("areaPersonal.ayuda.guia.texto")} ${traducir("areaPersonal.ayuda.guia.limiteServicio")}`;
  return `${encabezadoVista("Ayuda y accesibilidad", traducir("areaPersonal.ayuda.descripcion"), `<button type="button" class="boton-primario" data-accion="leer-pantalla">Leer esta página</button>`)}
    <div class="rejilla-principal"><div>
      ${panel("Buscar en la ayuda", "Respuestas sobre inscripción, méritos, baremo y llamamientos", `<form id="busqueda-ayuda" data-accion="buscar-ayuda"><div class="campo"><label for="consulta-ayuda">¿Qué necesita saber?</label><input id="consulta-ayuda" name="consulta" type="search" value="${escaparAtributo(estado.consultaAyuda || "")}" placeholder="Ejemplo: presentar una subsanación"></div><button type="submit" class="boton-primario">Buscar</button></form><div class="resultado-ayuda">${faq}</div>`)}
      ${panel(traducir("areaPersonal.ayuda.guia.titulo"), traducir("areaPersonal.ayuda.guia.subtitulo"), `<section aria-labelledby="titulo-guia-ayuda"><h4 id="titulo-guia-ayuda">${escaparHTML(traducir("areaPersonal.ayuda.guia.encabezado"))}</h4><p>${escaparHTML(transcripcion)}</p></section>`)}
    </div><aside aria-label="Opciones y canales de ayuda">
      ${panel("Opciones de visualización", "Se aplican solo durante esta visita", `<div class="fila-acciones"><button type="button" class="boton-secundario" data-accion="alternar-texto">Aumentar texto</button><button type="button" class="boton-secundario" data-accion="alternar-contraste">Alto contraste</button><button type="button" class="boton-secundario" data-accion="leer-pantalla">Lectura por voz</button></div><p>La interfaz admite teclado, ampliación del navegador, lectura de pantalla y reducción de movimiento.</p>`)}
      ${panel("Canales de soporte", "Ayuda sin exponer datos personales", `<dl class="dato-lista"><dt>Asistente</dt><dd>Consultas públicas sobre plazos, requisitos y uso del portal.</dd><dt>Soporte técnico</dt><dd>Canal autorizado indicado por el servicio</dd><dt>Protección de datos</dt><dd>Información de derechos y tratamiento en el aviso aplicable.</dd></dl><p class="nota">No incluya documentación personal en consultas generales de ayuda.</p>`)}
      ${panel("Navegación rápida", "Recorridos habituales", `<div class="fila-acciones">${enlaceRuta("convocatorias", "Inscribirme", "enlace-boton")}${enlaceRuta("meritos", "Aportar méritos", "enlace-boton")}${enlaceRuta("llamamientos", "Responder llamamiento", "enlace-boton")}</div>`)}
    </aside></div>`;
}
