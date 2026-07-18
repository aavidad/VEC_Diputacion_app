import {
  botonOperacion, chip, encabezadoVista, enlaceRuta, escaparAtributo, escaparHTML,
  formatoPuntos, listaDatos, panel, tabla,
} from "./comunes.js";

export function renderizarSeguimiento(datos, estado) {
  const solicitud = datos.solicitudes.find((item) => item.id === estado.expedienteSeleccionado) || datos.solicitudes[0];
  const filas = datos.solicitudes.map((item) => [
    `<strong>${escaparHTML(item.titulo)}</strong><small>${escaparHTML(item.referencia)}</small>`,
    chip(item.estado),
    `<strong>${formatoPuntos(item.puntuacion)} puntos</strong><small>${escaparHTML(item.posicion)}</small>`,
    escaparHTML(item.actualizado),
    `<div class="acciones-tabla"><button type="button" class="boton-secundario" data-accion="abrir-expediente" data-id="${escaparAtributo(item.id)}">Seleccionar</button></div>`,
  ]);
  const timeline = datos.actividad.map((item) => `<li><strong>${escaparHTML(item.titulo)}</strong><span>${escaparHTML(item.detalle)}</span><small>${escaparHTML(item.fecha)} · ${escaparHTML(item.actor)} · ${escaparHTML(item.recibo)}</small></li>`).join("");
  return `${encabezadoVista("Mis expedientes y seguimiento", "Estado, puntuación, posición, documentos y próximos pasos.", enlaceRuta("certificados", "Certificados y descargas", "boton-secundario"))}
    ${panel("Solicitudes en curso", "Seleccione un expediente para consultar el detalle", tabla({ descripcion: "Solicitudes de la persona autenticada", columnas: ["Proceso", "Estado", "Puntuación y posición", "Actualización", "Acción"], filas }))}
    <div class="rejilla-principal"><div>
      ${panel(solicitud.titulo, solicitud.referencia, `${listaDatos([["Estado", chip(solicitud.estado)], ["Puntuación provisional", `${formatoPuntos(solicitud.puntuacion)} puntos`], ["Posición", escaparHTML(solicitud.posicion)], ["Tasa", escaparHTML(solicitud.pago)], ["Firma", escaparHTML(solicitud.firma)], ["Última actualización", escaparHTML(solicitud.actualizado)]])}<p class="nota aviso"><strong>Siguiente actuación:</strong> ${escaparHTML(solicitud.siguiente)}</p>`, { estado: solicitud.estado })}
      ${panel("Historial del expediente", "Cada cambio muestra fecha, actor y referencia", `<ol class="linea-tiempo">${timeline}</ol>`)}
    </div><aside>
      ${panel("Posición provisional", "La posición puede cambiar tras revisión y alegaciones", `<div class="posicion-destacada"><output>${escaparHTML((solicitud.posicion.match(/^\d+/) || ["—"])[0])}</output><span><strong>${escaparHTML(solicitud.posicion)}</strong><small>Orden provisional y sujeto a las bases.</small></span></div>`, { estado: "Provisional" })}
      ${panel("Acciones disponibles", "Solo para el expediente seleccionado", `<div class="fila-acciones">${enlaceRuta("subsanaciones", "Subsanar", "enlace-boton")}${enlaceRuta("alegaciones", "Alegar", "enlace-boton")}${botonOperacion("solicitar_descarga", "Descargar expediente", { id: solicitud.id, clase: "boton-secundario", descripcion: "Preparar una copia descargable del expediente" })}</div>`)}
    </aside></div>`;
}

export function renderizarLlamamientos(datos) {
  const disponibles = datos.disponibilidad;
  const sufijoDemo = datos.meta.presentacion ? " DEMO" : "";
  const tarjetas = datos.llamamientos.map((item) => `<article class="panel"><header><div><h3>${escaparHTML(item.bolsa)}</h3><p>${escaparHTML(item.id)}</p></div>${chip(item.estado)}</header><div class="panel-contenido">${listaDatos([["Puesto y destino", escaparHTML(item.puesto)], ["Jornada", escaparHTML(item.jornada)], ["Duración", escaparHTML(item.duracion)], ["Plazo", escaparHTML(item.plazo)], ["Prelación", escaparHTML(item.posicion)]])}${item.estado === "Pendiente de respuesta" ? `<p class="nota aviso">Revise cuidadosamente destino, jornada, duración y plazo antes de responder.</p><div class="fila-acciones">${botonOperacion("responder_llamamiento", `Aceptar llamamiento${sufijoDemo}`, { id: item.id, descripcion: "Aceptar el llamamiento mostrado" })}${botonOperacion("responder_llamamiento", `Rechazar llamamiento${sufijoDemo}`, { id: `${item.id}|rechazar`, clase: "boton-peligro", descripcion: "Rechazar el llamamiento mostrado" })}</div>` : `<p class="nota">La respuesta ya consta como ${escaparHTML(item.estado)}.</p>`}</div></article>`).join("");
  const accionDisponibilidad = disponibles.disponible
    ? botonOperacion("cambiar_disponibilidad", `Declarar no disponibilidad${sufijoDemo}`, { id: "false", clase: "boton-peligro", descripcion: "Declarar una situación temporal de no disponibilidad" })
    : botonOperacion("cambiar_disponibilidad", `Declarar disponibilidad${sufijoDemo}`, { id: "true", descripcion: "Volver a estar disponible para llamamientos" });
  return `${encabezadoVista("Disponibilidad y llamamientos", "Controle su situación y responda únicamente a sus propios llamamientos.")}
    <div class="rejilla-principal"><div>${tarjetas}</div><aside>
      ${panel("Situación actual", "Aplicada a las bolsas en las que figura", `${listaDatos([["Estado", chip(disponibles.estado)], ["Desde", escaparHTML(disponibles.desde)], ["Próxima revisión", escaparHTML(disponibles.proxima_revision)], ["Bolsas", disponibles.bolsas.map(escaparHTML).join("<br>")]])}<div class="fila-acciones">${accionDisponibilidad}</div>`, { estado: disponibles.estado })}
      ${panel("Garantías del llamamiento", "Reglas que el servidor debe comprobar", `<ul><li>Orden vigente y causas de exclusión.</li><li>Disponibilidad en la fecha de corte.</li><li>Intentos, canales y plazo de respuesta.</li><li>Versión de bolsa y reglas aplicadas.</li><li>Recibo de cada actuación.</li></ul>`)}
    </aside></div>`;
}

export function renderizarSubsanaciones(datos) {
  const demo = datos.meta.presentacion;
  const formularios = datos.subsanaciones.map((item) => `<article class="panel"><header><div><h3>${escaparHTML(item.motivo)}</h3><p>${escaparHTML(item.id)} · ${escaparHTML(item.solicitud_ref)}</p></div>${chip(item.estado)}</header><div class="panel-contenido">${listaDatos([["Documento solicitado", escaparHTML(item.documento_solicitado)], ["Plazo", escaparHTML(item.plazo)], ["Estado", chip(item.estado)]])}${item.estado === "Pendiente" ? `<form data-operacion="presentar_subsanacion" data-id="${escaparAtributo(item.id)}"><div class="formulario-rejilla"><div class="campo ancho-completo"><label for="subsanacion-${escaparAtributo(item.id)}">Explicación</label><textarea id="subsanacion-${escaparAtributo(item.id)}" name="explicacion" required maxlength="1000">Se aporta documentación para completar la información solicitada.</textarea></div><div class="campo ancho-completo"><label for="fichero-${escaparAtributo(item.id)}">Documento</label><input id="fichero-${escaparAtributo(item.id)}" name="documento" type="file" accept=".pdf,.odt,.docx,.jpg,.png" required><small>${demo ? "En demostración no se lee ni envía el contenido." : "El fichero se custodiará solo si el servicio confirma la carga."}</small></div></div><label class="opcion-check"><input type="checkbox" name="declaracion" required><span><strong>Declaro que la documentación corresponde al requerimiento</strong><small>${demo ? "La presentación real requerirá firma y registro." : "La operación requerirá firma y devolverá un recibo de registro."}</small></span></label><button type="submit" class="boton-primario">Revisar, firmar y presentar${demo ? " DEMO" : ""}</button></form>` : `<p class="nota">La subsanación ya no requiere actuación en este recorrido.</p>`}</div></article>`).join("");
  return `${encabezadoVista("Subsanaciones", "Responda a requerimientos dentro de plazo y conserve el recibo de presentación.")}${formularios || panel("Sin subsanaciones", "No hay requerimientos pendientes", `<p>Cuando exista un requerimiento aparecerá aquí con su plazo y documentación solicitada.</p>`)}`;
}

export function renderizarAlegaciones(datos) {
  const demo = datos.meta.presentacion;
  const tarjetas = datos.alegaciones.map((item) => `<article class="panel"><header><div><h3>${escaparHTML(item.asunto)}</h3><p>${escaparHTML(item.id)} · ${escaparHTML(item.solicitud_ref)}</p></div>${chip(item.estado)}</header><div class="panel-contenido">${listaDatos([["Fecha", escaparHTML(item.fecha)], ["Estado", chip(item.estado)]])}${item.estado === "Borrador" ? `<form data-operacion="presentar_alegacion" data-id="${escaparAtributo(item.id)}"><div class="campo"><label for="alegacion-${escaparAtributo(item.id)}">Fundamento de la alegación</label><textarea id="alegacion-${escaparAtributo(item.id)}" name="fundamento" required maxlength="2000">Solicito la revisión del mérito señalado conforme al criterio de las bases.</textarea><small>Identifique el concepto discutido y la evidencia que lo respalda.</small></div><div class="campo"><label for="evidencia-${escaparAtributo(item.id)}">Evidencia adicional, si procede</label><input id="evidencia-${escaparAtributo(item.id)}" name="documento" type="file" accept=".pdf,.odt,.docx,.jpg,.png"></div><label class="opcion-check"><input type="checkbox" name="declaracion" required><span><strong>Confirmo el contenido de la alegación</strong><small>${demo ? "La presentación real se firmará y registrará." : "La operación se firmará y registrará."}</small></span></label><button type="submit" class="boton-primario">Revisar, firmar y presentar${demo ? " DEMO" : ""}</button></form>` : `<p class="nota">La alegación consta como ${escaparHTML(item.estado)}.</p>`}</div></article>`).join("");
  return `${encabezadoVista("Alegaciones y revisión", "Discuta una puntuación o decisión provisional con fundamento y evidencia.", enlaceRuta("autobaremacion", "Ver puntuación desglosada", "boton-secundario"))}${tarjetas || panel("Sin alegaciones", "No existen alegaciones asociadas", `<p>Podrá iniciar una cuando el procedimiento y el plazo lo permitan.</p>`)}`;
}
