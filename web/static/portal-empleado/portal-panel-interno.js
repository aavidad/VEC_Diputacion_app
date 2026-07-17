/**
 * Presentador del contrato agregado del panel interno de Bolsa.
 *
 * Solo conoce `vec.bolsa.panel.interno.v1`: no adapta el juego sintético de
 * presentación ni completa con valores aparentes los datos que el contrato no
 * proporciona. Recibe las utilidades visuales para mantener este módulo puro y
 * comprobable sin acceder al DOM global.
 */

const ESQUEMA_PANEL_INTERNO = "vec.bolsa.panel.interno.v1";

export function crearPresentadorPanelInterno(dependencias) {
  const {
    claseEstado,
    encabezadoVista,
    escaparHTML,
    numero,
    obtenerDatosPanel,
    tituloVista,
  } = dependencias;
  if ([claseEstado, encabezadoVista, escaparHTML, numero, obtenerDatosPanel, tituloVista]
    .some((dependencia) => typeof dependencia !== "function")) {
    throw new Error("dependencias del presentador de panel interno no válidas");
  }

  function datosPanel() {
    return obtenerDatosPanel();
  }

  function esActivo() {
    return datosPanel()?.esquema === ESQUEMA_PANEL_INTERNO;
  }

  function etiquetaFuente() {
    return esActivo() ? "Panel interno agregado autorizado" : "";
  }

  function actualizarContextoSesion(elementos) {
    if (!esActivo()) return false;
    const datos = datosPanel();
    elementos.avatar.textContent = "INT";
    elementos.nombre.textContent = "Contexto interno autorizado";
    elementos.perfil.textContent = datos.selector.clase === "unidad_gestion"
      ? "Ámbito: unidad de gestión"
      : "Ámbito: organización";
    if (elementos.avisos) {
      elementos.avisos.textContent = "—";
      elementos.avisos.setAttribute("aria-label", "Avisos no incluidos en el contrato del panel interno");
    }
    return true;
  }

  function tarjetaKPI(sigla, valor, etiqueta) {
    return `
      <article class="tarjeta-kpi">
        <span class="icono-kpi" aria-hidden="true">${escaparHTML(sigla)}</span>
        <div><strong class="valor-kpi">${escaparHTML(valor)}</strong><span class="etiqueta-kpi">${escaparHTML(etiqueta)}</span></div>
      </article>`;
  }

  function etiquetaClave(clave) {
    const texto = String(clave || "").replaceAll(/[._-]+/g, " ").trim();
    return texto ? texto.charAt(0).toLocaleUpperCase("es-ES") + texto.slice(1) : "Sin clave";
  }

  function instanteVisible(instante) {
    if (!instante || String(instante).startsWith("0001-01-01")) return "Sin fecha límite";
    const fecha = new Date(instante);
    if (!Number.isFinite(fecha.getTime())) return "Fecha no disponible";
    return new Intl.DateTimeFormat("es-ES", {
      dateStyle: "short", timeStyle: "short", timeZone: "Europe/Madrid",
    }).format(fecha);
  }

  function filasConvocatorias(datos) {
    if (datos.convocatorias.length === 0) {
      return '<tr><td colspan="6" class="vacio-controlado">La fuente autorizada no ha devuelto convocatorias para este ámbito.</td></tr>';
    }
    return datos.convocatorias.map((item) => `
      <tr>
        <td><strong>${escaparHTML(item.convocatoria_ref)}</strong></td>
        <td>${escaparHTML(etiquetaClave(item.categoria_clave))}<br><small>${escaparHTML(item.categoria_clave)}</small></td>
        <td><span class="estado-chip ${claseEstado(item.estado_clave)}">${escaparHTML(etiquetaClave(item.estado_clave))}</span></td>
        <td>${item.plazo_cierra_en ? `<time datetime="${escaparHTML(item.plazo_cierra_en)}">${escaparHTML(instanteVisible(item.plazo_cierra_en))}</time>` : "Sin fecha límite"}</td>
        <td>${numero(item.numero_solicitudes)}</td><td>${numero(item.numero_pendientes)}</td>
      </tr>`).join("");
  }

  function filasActuaciones(datos) {
    if (datos.actuaciones_pendientes.length === 0) {
      return '<tr><td colspan="7" class="vacio-controlado">La fuente autorizada no ha devuelto actuaciones pendientes para este ámbito.</td></tr>';
    }
    return datos.actuaciones_pendientes.map((item) => `
      <tr>
        <td><strong>${escaparHTML(item.actuacion_ref)}</strong></td><td>${escaparHTML(item.recurso_ref)}</td>
        <td>${escaparHTML(etiquetaClave(item.tipo_clave))}<br><small>${escaparHTML(item.tipo_clave)}</small></td>
        <td><span class="estado-chip ${claseEstado(item.estado_clave)}">${escaparHTML(etiquetaClave(item.estado_clave))}</span></td>
        <td><span class="estado-chip ${claseEstado(item.prioridad_clave)}">${escaparHTML(etiquetaClave(item.prioridad_clave))}</span></td>
        <td>${item.fecha_limite ? `<time datetime="${escaparHTML(item.fecha_limite)}">${escaparHTML(instanteVisible(item.fecha_limite))}</time>` : "Sin fecha límite"}</td>
        <td>${numero(item.numero_elementos)}</td>
      </tr>`).join("");
  }

  function renderizarResumen(datos) {
    const i = datos.indicadores;
    return `
      ${encabezadoVista("Gestión interna de Bolsas", "Cuadro de mando", "Información agregada y autorizada del ámbito interno. El contrato no contiene datos personales ni habilita acciones administrativas.", '<button type="button" class="boton-secundario" data-accion="imprimir">Imprimir resumen</button>')}
      <section class="nota-seguridad" aria-label="Alcance del panel real">Vista de solo lectura. Los contadores, convocatorias y actuaciones proceden del contrato <code>${ESQUEMA_PANEL_INTERNO}</code>; no se completan con datos del modo de presentación.</section>
      <div class="rejilla-kpi" aria-label="Indicadores operativos de Bolsa">
        ${tarjetaKPI("BOR", numero(i.convocatorias_borrador), "Convocatorias en borrador")}
        ${tarjetaKPI("REV", numero(i.convocatorias_revision), "Convocatorias en revisión")}
        ${tarjetaKPI("FIR", numero(i.convocatorias_pendientes_firma), "Convocatorias pendientes de firma")}
        ${tarjetaKPI("PUB", numero(i.convocatorias_publicadas), "Convocatorias publicadas")}
        ${tarjetaKPI("BOL", numero(i.bolsas_activas), "Bolsas activas")}
        ${tarjetaKPI("SUS", numero(i.bolsas_suspendidas), "Bolsas suspendidas")}
        ${tarjetaKPI("AGO", numero(i.bolsas_agotadas), "Bolsas agotadas")}
        ${tarjetaKPI("LLA", numero(i.llamamientos_pendientes), "Llamamientos pendientes")}
        ${tarjetaKPI("CUR", numero(i.llamamientos_en_curso), "Llamamientos en curso")}
        ${tarjetaKPI("HOY", numero(i.llamamientos_vencen_hoy), "Llamamientos que vencen hoy")}
        ${tarjetaKPI("DOC", numero(i.documentos_pendientes_firma), "Documentos pendientes de firma")}
        ${tarjetaKPI("INC", numero(i.incidencias_abiertas), "Incidencias abiertas")}
      </div>
      <section class="panel"><div class="cabecera-panel"><h3>Convocatorias del ámbito autorizado</h3><span class="estado-chip info">${numero(datos.convocatorias.length)} registros</span></div><div class="tabla-contenedor"><table class="tabla-datos"><caption>Convocatorias agregadas devueltas por el panel interno</caption><thead><tr><th scope="col">Referencia</th><th scope="col">Categoría</th><th scope="col">Estado</th><th scope="col">Cierre de plazo</th><th scope="col">Solicitudes</th><th scope="col">Pendientes</th></tr></thead><tbody>${filasConvocatorias(datos)}</tbody></table></div></section>
      <section class="panel"><div class="cabecera-panel"><h3>Actuaciones pendientes</h3><span class="estado-chip info">${numero(datos.actuaciones_pendientes.length)} registros</span></div><div class="tabla-contenedor"><table class="tabla-datos"><caption>Trabajo administrativo pendiente sin identidad de personas interesadas</caption><thead><tr><th scope="col">Actuación</th><th scope="col">Recurso</th><th scope="col">Tipo</th><th scope="col">Estado</th><th scope="col">Prioridad</th><th scope="col">Fecha límite</th><th scope="col">Elementos</th></tr></thead><tbody>${filasActuaciones(datos)}</tbody></table></div></section>
      <section class="panel"><div class="cabecera-panel"><h3>Prueba de lectura</h3><span class="estado-chip exito">Lectura auditada</span></div><div class="cuerpo-panel"><dl class="resumen-expediente"><div class="fila-resumen"><dt>Ámbito</dt><dd>${escaparHTML(etiquetaClave(datos.selector.clase))}</dd></div><div class="fila-resumen"><dt>Revisión de fuente</dt><dd>${escaparHTML(datos.origen.revision)}</dd></div><div class="fila-resumen"><dt>Actualizada</dt><dd><time datetime="${escaparHTML(datos.origen.actualizada_en)}">${escaparHTML(instanteVisible(datos.origen.actualizada_en))}</time></dd></div><div class="fila-resumen"><dt>Lectura</dt><dd>${escaparHTML(datos.prueba_lectura.lectura_ref)}</dd></div><div class="fila-resumen"><dt>Auditoría</dt><dd>${escaparHTML(datos.prueba_lectura.auditoria_ref)} · secuencia ${numero(datos.prueba_lectura.auditoria_secuencia)}</dd></div><div class="fila-resumen"><dt>Confirmada</dt><dd><time datetime="${escaparHTML(datos.prueba_lectura.confirmada_en)}">${escaparHTML(instanteVisible(datos.prueba_lectura.confirmada_en))}</time></dd></div></dl></div></section>`;
  }

  function renderizarConvocatorias(datos) {
    const i = datos.indicadores;
    return `
      ${encabezadoVista("Proyección interna autorizada", "Convocatorias", "Consulta de convocatorias agregadas. La edición, las bases y la publicación todavía no están conectadas a esta superficie web.", '<button type="button" class="boton-secundario" data-vista="resumen">Volver al cuadro de mando</button>')}
      <section class="nota-pendiente">Modo de solo lectura: el contrato real no incluye expedientes de elaboración ni concede capacidad para modificarlos.</section>
      <div class="rejilla-kpi">${tarjetaKPI("BOR", numero(i.convocatorias_borrador), "Borrador")}${tarjetaKPI("REV", numero(i.convocatorias_revision), "En revisión")}${tarjetaKPI("FIR", numero(i.convocatorias_pendientes_firma), "Pendientes de firma")}${tarjetaKPI("PUB", numero(i.convocatorias_publicadas), "Publicadas")}</div>
      <section class="panel"><div class="cabecera-panel"><h3>Convocatorias del ámbito autorizado</h3><span class="estado-chip info">${numero(datos.convocatorias.length)} registros</span></div><div class="tabla-contenedor"><table class="tabla-datos"><caption>Convocatorias agregadas en modo de solo lectura</caption><thead><tr><th scope="col">Referencia</th><th scope="col">Categoría</th><th scope="col">Estado</th><th scope="col">Cierre de plazo</th><th scope="col">Solicitudes</th><th scope="col">Pendientes</th></tr></thead><tbody>${filasConvocatorias(datos)}</tbody></table></div></section>`;
  }

  function renderizarNoConectada(vista) {
    return `
      ${encabezadoVista("Contrato real de alcance mínimo", tituloVista(vista), "Esta sección no está disponible con el contrato interno actualmente conectado.", '<button type="button" class="boton-secundario" data-vista="resumen">Volver al cuadro de mando</button>')}
      <section class="panel"><div class="cuerpo-panel vacio-controlado"><p><strong>Funcionalidad no conectada</strong></p><p>El servidor solo ha acreditado indicadores agregados, convocatorias y actuaciones pendientes. No se muestran valores cero, tablas vacías ni controles aparentes para datos que no han sido proporcionados.</p></div></section>`;
  }

  function renderizarVista(vista) {
    if (!esActivo()) throw new Error("el presentador requiere un panel interno válido");
    const datos = datosPanel();
    if (vista === "resumen") return renderizarResumen(datos);
    if (vista === "elaboracion") return renderizarConvocatorias(datos);
    return renderizarNoConectada(vista);
  }

  return Object.freeze({ actualizarContextoSesion, esActivo, etiquetaFuente, renderizarVista });
}
