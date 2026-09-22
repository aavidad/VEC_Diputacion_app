import {
  botonOperacion, chip, encabezadoVista, enlaceRuta, escaparAtributo, escaparHTML,
  formatoPuntos, listaDatos, panel, tabla,
} from "./comunes.js";

export function renderizarSeguimiento(datos, estado) {
  const solicitud = datos.solicitudes.find((item) => item.id === estado.expedienteSeleccionado) || datos.solicitudes[0];
  if (!solicitud) {
    const cabecera = encabezadoVista(
      "Mis expedientes y seguimiento",
      "Estado, puntuación, posición, documentos y próximos pasos.",
      enlaceRuta("certificados", "Certificados y descargas", "boton-secundario"),
    );
    const vacio = panel(
      "Sin expedientes en el ámbito autorizado",
      "No se ha seleccionado ningún expediente",
      "<p>No hay datos de expediente para consultar o descargar.</p>",
    );
    const acciones = panel(
      "Acciones disponibles",
      "Solo para el expediente seleccionado",
      "<p>No hay acciones disponibles hasta que el servicio facilite un expediente autorizado.</p>",
    );
    return `${cabecera}${vacio}${acciones}`;
  }
  const filas = datos.solicitudes.map((item) => [
    `<strong>${escaparHTML(item.titulo)}</strong><small>${escaparHTML(item.referencia)}</small>`,
    chip(item.estado),
    `<strong>${formatoPuntos(item.puntuacion)} puntos</strong><small>${escaparHTML(item.posicion)}</small>`,
    escaparHTML(item.actualizado),
    `<div class="acciones-tabla"><button type="button" class="boton-secundario" data-accion="abrir-expediente" data-id="${escaparAtributo(item.id)}">Seleccionar</button></div>`,
  ]);
  const timeline = datos.actividad.map((item) => `<li><strong>${escaparHTML(item.titulo)}</strong><span>${escaparHTML(item.detalle)}</span><small>${escaparHTML(item.fecha)} · ${escaparHTML(item.actor)} · ${escaparHTML(item.recibo)}</small></li>`).join("");
  const panelPosicion = datos.posicion
    ? panel("Mi posición", "Situación y orden de prelación en la bolsa adscrita", `<div class="posicion-destacada"><output>#${escaparHTML(String(datos.posicion.orden))}</output><span><strong>${escaparHTML(datos.posicion.categoria)}</strong><small>${escaparHTML(datos.posicion.bolsa)}</small></span></div>${listaDatos([["Bolsa", escaparHTML(datos.posicion.bolsa)], ["Categoría", escaparHTML(datos.posicion.categoria)], ["Orden", `#${escaparHTML(String(datos.posicion.orden))} de ${escaparHTML(String(datos.posicion.total))}`], ["Puntuación", `${formatoPuntos(datos.posicion.puntuacion)} puntos`], ["Vigente desde", escaparHTML(datos.posicion.vigente_desde)]])}`, { estado: `#${datos.posicion.orden}` })
    : panel("Posición provisional", "La posición puede cambiar tras revisión y alegaciones", `<div class="posicion-destacada"><output>${escaparHTML((solicitud.posicion.match(/^\d+/) || ["—"])[0])}</output><span><strong>${escaparHTML(solicitud.posicion)}</strong><small>Orden provisional y sujeto a las bases.</small></span></div>`, { estado: "Provisional" });
  return `${encabezadoVista("Mis expedientes y seguimiento", "Estado, puntuación, posición, documentos y próximos pasos.", enlaceRuta("certificados", "Certificados y descargas", "boton-secundario"))}
    ${panel("Solicitudes en curso", "Seleccione un expediente para consultar el detalle", tabla({ descripcion: "Solicitudes de la persona autenticada", columnas: ["Proceso", "Estado", "Puntuación y posición", "Actualización", "Acción"], filas }))}
    <div class="rejilla-principal"><div>
      ${panel(solicitud.titulo, solicitud.referencia, `${listaDatos([["Estado", chip(solicitud.estado)], ["Puntuación provisional", `${formatoPuntos(solicitud.puntuacion)} puntos`], ["Posición", escaparHTML(solicitud.posicion)], ["Tasa", escaparHTML(solicitud.pago)], ["Firma", escaparHTML(solicitud.firma)], ["Última actualización", escaparHTML(solicitud.actualizado)]])}<p class="nota aviso"><strong>Siguiente actuación:</strong> ${escaparHTML(solicitud.siguiente)}</p>`, { estado: solicitud.estado })}
      ${panel("Historial del expediente", "Cada cambio muestra fecha, actor y referencia", `<ol class="linea-tiempo">${timeline}</ol>`)}
    </div><aside>
      ${panelPosicion}
      ${panel("Acciones disponibles", "Solo para el expediente seleccionado", `<div class="fila-acciones">${enlaceRuta("subsanaciones", "Subsanar", "enlace-boton")}${enlaceRuta("alegaciones", "Alegar", "enlace-boton")}${botonOperacion("solicitar_descarga", "Descargar expediente", { id: solicitud.id, clase: "boton-secundario", descripcion: "Preparar una copia descargable del expediente" })}</div>`)}
    </aside></div>`;
}

export function renderizarLlamamientos(datos, estado = {}) {
  const participaciones = Array.isArray(estado.participaciones) && estado.participaciones.length
    ? estado.participaciones : datos.posicion ? [{ bolsa: datos.posicion.bolsa, categoria: datos.posicion.categoria, orden_inicial: datos.posicion.orden, total_instantanea: datos.posicion.total, version: "Pendiente de integración", estado_bolsa: "Pendiente de integración", vigente_desde: datos.posicion.vigente_desde, vigente_hasta: null }] : [];
  const pagina = Math.max(1, Number(estado.paginaParticipaciones || 1));
  const porPagina = 6;
  const totalPaginas = Math.max(1, Math.ceil(participaciones.length / porPagina));
  const paginaActual = Math.min(pagina, totalPaginas);
  const visibles = participaciones.slice((paginaActual - 1) * porPagina, paginaActual * porPagina);
  const tarjetasParticipacion = visibles.map((item) => panel("Mi participación", item.categoria, `${listaDatos([["Estado de la bolsa", chip(item.estado_bolsa)], ["Mi número de orden", `${escaparHTML(String(item.orden_inicial))} de ${escaparHTML(String(item.total_instantanea))}`], ["Versión de la bolsa", escaparHTML(String(item.version))], ["Vigencia", `${escaparHTML(item.vigente_desde)}${item.vigente_hasta ? ` · hasta ${escaparHTML(item.vigente_hasta)}` : " · vigente"}`]])}`, { estado: item.estado_bolsa, clase: "participacion-propia" })).join("");
  const paginacion = participaciones.length > porPagina ? `<nav class="paginacion-participaciones" aria-label="Paginación de participaciones"><span>Mostrando ${(paginaActual - 1) * porPagina + 1} a ${Math.min(paginaActual * porPagina, participaciones.length)} de ${participaciones.length}</span><button type="button" class="boton-secundario" data-accion="pagina-participaciones" data-pagina="${paginaActual - 1}" ${paginaActual === 1 ? "disabled" : ""}>Anterior</button><button type="button" class="boton-secundario" data-accion="pagina-participaciones" data-pagina="${paginaActual + 1}" ${paginaActual === totalPaginas ? "disabled" : ""}>Siguiente</button></nav>` : "";
  const avisoFuente = estado.fuenteBolsa === "ejemplo" ? `<section class="aviso-fuente-ejemplo" role="status"><strong>Datos de ejemplo.</strong> El acceso del candidato con DNIe o certificado está pendiente de desarrollo en VEC.${estado.causaBolsa === "autenticacion_requerida" ? " Identifíquese con certificado cuando la frontera esté disponible." : ""}</section>` : "";
  const fichaParticipaciones = participaciones.length ? `<section class="marco-participaciones" aria-label="Mis participaciones en bolsa">${tarjetasParticipacion}${paginacion}</section>` : panel("Mis participaciones", "Sin participaciones activas", "<p>No constan participaciones en bolsa para la identidad actual.</p>");
  const disponibles = datos.disponibilidad;
  const sufijoDemo = datos.meta.presentacion ? " DEMO" : "";
  const tarjetas = datos.llamamientos.map((item, indice) => {
    const camposLlamamiento = [
      ["Puesto y destino", escaparHTML(item.puesto)],
      item.jornada ? ["Jornada", escaparHTML(item.jornada)] : null,
      item.duracion ? ["Duración", escaparHTML(item.duracion)] : null,
      ["Plazo", escaparHTML(item.plazo)],
      item.posicion ? ["Prelación", escaparHTML(item.posicion)] : null,
      item.canal ? ["Canal", escaparHTML(item.canal)] : null,
      item.comunicado_en ? ["Comunicado el", escaparHTML(item.comunicado_en)] : null,
      item.resultado_clave ? ["Resultado", chip(item.resultado_clave)] : null,
    ].filter(Boolean);
    const titulo = indice === 0 ? "Último llamamiento" : "Llamamiento anterior";
    return `<article class="panel"><header><div><p>${titulo}</p><h3>${escaparHTML(item.bolsa)}</h3><p>${escaparHTML(item.id)}</p></div>${chip(item.estado)}</header><div class="panel-contenido">${listaDatos(camposLlamamiento)}${item.estado === "Pendiente de respuesta" ? `<p class="nota aviso">Canal, plazo y efectos pendientes de confirmación por RRHH. Esta acción solo cambia la memoria de presentación.</p><div class="fila-acciones">${botonOperacion("responder_llamamiento", `Aceptar llamamiento${sufijoDemo}`, { id: item.id, descripcion: "Registrar una respuesta efímera al llamamiento mostrado" })}${botonOperacion("responder_llamamiento", `Rechazar llamamiento${sufijoDemo}`, { id: `${item.id}|rechazar`, clase: "boton-peligro", descripcion: "Registrar una respuesta efímera al llamamiento mostrado" })}</div>` : `<p class="nota">La respuesta mostrada es sintética y no acredita un efecto administrativo.</p>`}</div></article>`;
  }).join("");
  const llamamientos = panel("Llamamientos", "Información propia", "<p class=\"nota aviso\"><strong>Pendiente de integración.</strong> Esta información no la devuelve todavía Mi bolsa.</p>");
  const contratos = Array.isArray(datos.contratos) && datos.contratos.length > 0
    ? listaDatos(datos.contratos.map((item) => [escaparHTML(item.id), escaparHTML(item.estado)]))
    : `<p>No consta ningún contrato propio en los datos disponibles.</p>`;

  const pendiente = "<p class=\"nota aviso\"><strong>Pendiente de integración.</strong> Esta información no la devuelve todavía Mi bolsa.</p>";
  return `${encabezadoVista("Mi bolsa", "Consulte su posición y vigencia. Identificarse con certificado no firma documentos.")}
    ${avisoFuente}${fichaParticipaciones}
    <div class="rejilla-principal"><div>${llamamientos}</div><aside>
      ${panel("Disponibilidad", "Pausar o reactivar", pendiente)}
      ${panel("Último llamamiento", "Información propia", pendiente)}
      ${panel("Contratos", "Información propia", pendiente)}
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
