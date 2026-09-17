/**
 * Presentador del contrato agregado del panel interno de Bolsa y vistas B12 y B5.
 *
 * Conoce `vec.bolsa.panel.interno.v1`, `vec.bolsa.rrhh.bolsas.v1` y `vec.bolsa.rrhh.candidatos.v1`.
 * No adapta el juego sintético de presentación ni completa con valores aparentes los datos que el
 * contrato no proporciona. Recibe las utilidades visuales para mantener este módulo puro y comprobable
 * sin acceder al DOM global.
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
    obtenerDatosBolsas,
    obtenerDatosCandidatosBolsa,
    obtenerEstadoCandidatos,
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

  function renderizarCuadroB12() {
    const estadoBolsas = typeof obtenerDatosBolsas === "function" ? obtenerDatosBolsas() : null;
    if (!estadoBolsas) return "";

    if (estadoBolsas.carga === "cargando") {
      return `
        <section class="panel" aria-labelledby="titulo-cuadro-b12">
          <div class="cabecera-panel">
            <h3 id="titulo-cuadro-b12">Bolsas de trabajo (Cuadro B12)</h3>
            <span class="estado-chip neutro">Consultando…</span>
          </div>
          <div class="cuerpo-panel vacio-controlado" role="status" aria-busy="true">
            <p><strong>Cargando bolsas de trabajo…</strong></p>
            <p>Consultando la disponibilidad y el desglose de aspirantes en el ámbito autorizado.</p>
          </div>
        </section>`;
    }

    if (estadoBolsas.carga === "error") {
      return `
        <section class="panel" aria-labelledby="titulo-cuadro-b12">
          <div class="cabecera-panel">
            <h3 id="titulo-cuadro-b12">Bolsas de trabajo (Cuadro B12)</h3>
            <span class="estado-chip peligro">Error de carga</span>
          </div>
          <div class="cuerpo-panel vacio-controlado" role="alert">
            <p><strong>No se pudieron cargar las bolsas de trabajo</strong></p>
            <p>${escaparHTML(estadoBolsas.error || "Se ha producido un error al consultar las bolsas.")}</p>
            <div class="acciones-vista">
              <button type="button" class="boton-secundario" data-bolsa-accion="reintentar-bolsas">Reintentar</button>
            </div>
          </div>
        </section>`;
    }

    if (estadoBolsas.carga === "denegado") {
      return `
        <section class="panel" aria-labelledby="titulo-cuadro-b12">
          <div class="cabecera-panel">
            <h3 id="titulo-cuadro-b12">Bolsas de trabajo (Cuadro B12)</h3>
            <span class="estado-chip peligro">Acceso denegado</span>
          </div>
          <div class="cuerpo-panel vacio-controlado" role="alert">
            <p><strong>Acceso denegado a la consulta de bolsas</strong></p>
            <p>La sesión actual no dispone de permisos suficientes para consultar el cuadro de bolsas de trabajo.</p>
          </div>
        </section>`;
    }

    const bolsas = estadoBolsas.datos?.bolsas || [];
    if (bolsas.length === 0) {
      return `
        <section class="panel" aria-labelledby="titulo-cuadro-b12">
          <div class="cabecera-panel">
            <h3 id="titulo-cuadro-b12">Bolsas de trabajo (Cuadro B12)</h3>
            <span class="estado-chip neutro">0 registros</span>
          </div>
          <div class="cuerpo-panel vacio-controlado" role="status">
            <p><strong>No hay bolsas de trabajo activas</strong></p>
            <p>El servicio no ha devuelto bolsas de trabajo registradas para este ámbito.</p>
            <div class="acciones-vista">
              <button type="button" class="boton-secundario" data-bolsa-accion="reintentar-bolsas">Reintentar</button>
            </div>
          </div>
        </section>`;
    }

    const filas = bolsas.map((b) => `
      <tr data-bolsa-ref="${escaparHTML(b.bolsa_ref)}">
        <td><strong>${escaparHTML(b.categoria)}</strong><br><small>${escaparHTML(b.categoria_clave)}</small></td>
        <td><span class="estado-chip neutro">${escaparHTML(etiquetaClave(b.tipo_lista))}</span></td>
        <td><small>${escaparHTML(b.vigente_desde)} ${b.vigente_hasta ? `— ${escaparHTML(b.vigente_hasta)}` : "(vigente)"}</small></td>
        <td><strong>${numero(b.total)}</strong></td>
        <td><span class="estado-chip exito">${numero(b.por_estado?.disponible)}</span></td>
        <td><span class="estado-chip neutro">${numero(b.por_estado?.ocupado)}</span></td>
        <td><span class="estado-chip peligro">${numero(b.por_estado?.no_disponible)}</span></td>
        <td><span class="estado-chip peligro">${numero(b.por_estado?.excluido)}</span></td>
        <td><span class="estado-chip">${numero(b.por_estado?.renuncia_pendiente)}</span></td>
        <td>
          <button type="button" class="boton-secundario" data-accion="ver-bolsa" data-bolsa-ref="${escaparHTML(b.bolsa_ref)}">Ver candidatos</button>
        </td>
      </tr>
    `).join("");

    return `
      <section class="panel" aria-labelledby="titulo-cuadro-b12">
        <div class="cabecera-panel">
          <h3 id="titulo-cuadro-b12">Bolsas de trabajo activas (Cuadro B12)</h3>
          <span class="estado-chip info">${numero(bolsas.length)} bolsas</span>
        </div>
        <div class="tabla-contenedor">
          <table class="tabla-datos">
            <caption>Cuadro B12: Bolsas de trabajo y distribución de aspirantes por situación</caption>
            <thead>
              <tr>
                <th scope="col">Bolsa / Categoría</th>
                <th scope="col">Tipo de lista</th>
                <th scope="col">Vigencia</th>
                <th scope="col">Total</th>
                <th scope="col">Disponibles</th>
                <th scope="col">Ocupados</th>
                <th scope="col">No disp.</th>
                <th scope="col">Excluidos</th>
                <th scope="col">Renuncia pend.</th>
                <th scope="col">Acciones</th>
              </tr>
            </thead>
            <tbody>
              ${filas}
            </tbody>
          </table>
        </div>
      </section>`;
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
      ${renderizarCuadroB12()}
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

  function renderizarCandidatosBolsa() {
    const estadoCandidatos = typeof obtenerDatosCandidatosBolsa === "function"
      ? obtenerDatosCandidatosBolsa()
      : null;
    const filtrosActuales = typeof obtenerEstadoCandidatos === "function"
      ? obtenerEstadoCandidatos()
      : { estado: "", texto: "" };

    const accionesEncabezado = '<button type="button" class="boton-secundario" data-vista="resumen">Volver al cuadro</button>';

    if (!estadoCandidatos || estadoCandidatos.carga === "cargando") {
      return `
        ${encabezadoVista("Gestión interna de Bolsas", "Candidatos de la bolsa", "Consulta ordenada de aspirantes y situación de disponibilidad.", accionesEncabezado)}
        <section class="panel">
          <div class="cuerpo-panel vacio-controlado" role="status" aria-busy="true">
            <p><strong>Cargando lista de candidatos…</strong></p>
            <p>Obteniendo las candidaturas y llamamientos de la bolsa seleccionada.</p>
          </div>
        </section>`;
    }

    if (estadoCandidatos.carga === "error") {
      return `
        ${encabezadoVista("Gestión interna de Bolsas", "Candidatos de la bolsa", "Consulta ordenada de aspirantes y situación de disponibilidad.", accionesEncabezado)}
        <section class="panel">
          <div class="cuerpo-panel vacio-controlado" role="alert">
            <p><strong>Error al consultar candidatos</strong></p>
            <p>${escaparHTML(estadoCandidatos.error || "No se pudo cargar la relación de aspirantes.")}</p>
            <div class="acciones-vista">
              <button type="button" class="boton-secundario" data-bolsa-accion="reintentar-candidatos">Reintentar</button>
              <button type="button" class="boton-secundario" data-vista="resumen">Volver al cuadro</button>
            </div>
          </div>
        </section>`;
    }

    if (estadoCandidatos.carga === "denegado") {
      return `
        ${encabezadoVista("Gestión interna de Bolsas", "Candidatos de la bolsa", "Consulta ordenada de aspirantes y situación de disponibilidad.", accionesEncabezado)}
        <section class="panel">
          <div class="cuerpo-panel vacio-controlado" role="alert">
            <p><strong>Acceso denegado</strong></p>
            <p>La sesión no dispone de permisos para consultar los candidatos de esta bolsa.</p>
            <div class="acciones-vista">
              <button type="button" class="boton-secundario" data-vista="resumen">Volver al cuadro</button>
            </div>
          </div>
        </section>`;
    }

    const bolsa = estadoCandidatos.datos?.bolsa;
    const candidatos = estadoCandidatos.datos?.candidatos || [];
    const hayMas = estadoCandidatos.datos?.hay_mas === true;
    const cursorSiguiente = estadoCandidatos.datos?.cursor_siguiente || "";

    const tituloBolsa = bolsa ? `Candidatos: ${bolsa.categoria}` : "Candidatos de la bolsa";
    const descripcionBolsa = bolsa
      ? `${bolsa.categoria_clave} · Lista ${bolsa.tipo_lista} · Total: ${numero(bolsa.total)} aspirantes`
      : "Consulta ordenada de aspirantes y situación de disponibilidad.";

    const opcionesEstado = [
      ["", "Todos los estados"],
      ["disponible", "Disponible"],
      ["ocupado", "Ocupado"],
      ["no_disponible", "No disponible"],
      ["excluido", "Excluido"],
      ["renuncia_pendiente", "Renuncia pendiente"],
    ].map(([valor, etiqueta]) => `
      <option value="${escaparHTML(valor)}"${valor === filtrosActuales.estado ? " selected" : ""}>${escaparHTML(etiqueta)}</option>
    `).join("");

    const formularioFiltros = `
      <form class="barra-filtros-bolsa" data-bolsa-form="filtros" role="search" aria-label="Filtros de candidatos">
        <div class="campo-filtro">
          <label for="filtro-bolsa-estado">Situación:</label>
          <select id="filtro-bolsa-estado" name="estado">${opcionesEstado}</select>
        </div>
        <div class="campo-filtro">
          <label for="filtro-bolsa-texto">Buscar:</label>
          <input type="search" id="filtro-bolsa-texto" name="texto" value="${escaparHTML(filtrosActuales.texto || "")}" placeholder="Nombre o documento…">
        </div>
        <div class="acciones-filtro">
          <button type="submit" class="boton-primario">Filtrar</button>
          <button type="button" class="boton-secundario" data-bolsa-accion="limpiar-filtros">Limpiar</button>
        </div>
      </form>`;

    let cuerpoTabla = "";
    if (candidatos.length === 0) {
      cuerpoTabla = `<tr><td colspan="7" class="vacio-controlado">No se han encontrado aspirantes que coincidan con los criterios seleccionados.</td></tr>`;
    } else {
      cuerpoTabla = candidatos.map((c) => {
        let detalleLlamamiento = '<small class="texto-atenuado">Sin llamamientos</small>';
        if (c.ultimo_llamamiento) {
          const l = c.ultimo_llamamiento;
          detalleLlamamiento = `<span>${escaparHTML(etiquetaClave(l.canal))} · ${escaparHTML(etiquetaClave(l.resultado))}<br><small><time datetime="${escaparHTML(l.comunicado_en)}">${escaparHTML(instanteVisible(l.comunicado_en))}</time></small></span>`;
        }
        return `
          <tr data-participacion-ref="${escaparHTML(c.participacion_ref)}">
            <td><strong>#${numero(c.orden)}</strong></td>
            <td><strong>${escaparHTML(c.nombre_visible)}</strong></td>
            <td><code>${escaparHTML(c.documento_enmascarado)}</code></td>
            <td><span class="estado-chip ${claseEstado(c.estado_clave)}">${escaparHTML(etiquetaClave(c.estado_clave))}</span></td>
            <td><small>${escaparHTML(instanteVisible(c.estado_desde))}</small></td>
            <td><small>${c.disponible_desde ? escaparHTML(instanteVisible(c.disponible_desde)) : "—"}</small></td>
            <td>${detalleLlamamiento}</td>
          </tr>`;
      }).join("");
    }

    const paginacion = hayMas && cursorSiguiente
      ? `<div class="paginacion-bolsa">
           <button type="button" class="boton-secundario" data-bolsa-accion="pagina-siguiente" data-cursor="${escaparHTML(cursorSiguiente)}">Cargar siguientes aspirantes</button>
         </div>`
      : "";

    return `
      ${encabezadoVista("Gestión interna de Bolsas", tituloBolsa, descripcionBolsa, accionesEncabezado)}
      <section class="panel">
        <div class="cabecera-panel">
          <h3>Filtros y ordenación de aspirantes (Vista B5)</h3>
          <span class="estado-chip info">${numero(candidatos.length)} en esta página</span>
        </div>
        <div class="cuerpo-panel">
          ${formularioFiltros}
        </div>
      </section>
      <section class="panel">
        <div class="cabecera-panel">
          <h3>Relación ordenada de candidatos</h3>
        </div>
        <div class="tabla-contenedor">
          <table class="tabla-datos">
            <caption>Aspirantes ordenados por mérito y situación en bolsa</caption>
            <thead>
              <tr>
                <th scope="col">Orden</th>
                <th scope="col">Aspirante</th>
                <th scope="col">Documento</th>
                <th scope="col">Situación</th>
                <th scope="col">Desde</th>
                <th scope="col">Disponible desde</th>
                <th scope="col">Último llamamiento</th>
              </tr>
            </thead>
            <tbody>
              ${cuerpoTabla}
            </tbody>
          </table>
        </div>
        ${paginacion}
      </section>`;
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
    if (vista === "bolsa-candidatos") return renderizarCandidatosBolsa();
    return renderizarNoConectada(vista);
  }

  return Object.freeze({ actualizarContextoSesion, esActivo, etiquetaFuente, renderizarVista });
}
