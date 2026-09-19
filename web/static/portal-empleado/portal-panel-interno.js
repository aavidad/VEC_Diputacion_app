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
    obtenerModalContactos,
    obtenerModalFicha,
    obtenerModalLlamar,
    obtenerModalResultado,
    esLecturaPresentacion = () => false,
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
    const lecturaPresentacion = esLecturaPresentacion() === true;
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
      cuerpoTabla = `<tr><td colspan="8" class="vacio-controlado">No se han encontrado aspirantes que coincidan con los criterios seleccionados.</td></tr>`;
    } else {
      cuerpoTabla = candidatos.map((c) => {
        let detalleLlamamiento = '<small class="texto-atenuado">Sin llamamientos</small>';
        if (c.ultimo_llamamiento) {
          const l = c.ultimo_llamamiento;
          detalleLlamamiento = `<span>${escaparHTML(etiquetaClave(l.canal))} · ${escaparHTML(etiquetaClave(l.resultado))}<br><small><time datetime="${escaparHTML(l.comunicado_en)}">${escaparHTML(instanteVisible(l.comunicado_en))}</time></small></span>`;
        }

        const acciones = [];
        acciones.push(`<button type="button" class="boton-secundario" data-bolsa-accion="abrir-ficha" data-participacion-ref="${escaparHTML(c.participacion_ref)}">Ver ficha</button>`);
        acciones.push(`<button type="button" class="boton-secundario" data-bolsa-accion="abrir-contactos" data-participacion-ref="${escaparHTML(c.participacion_ref)}" data-nombre-visible="${escaparHTML(c.nombre_visible)}">Contactos</button>`);
        if (c.estado_clave === "disponible" && !lecturaPresentacion) {
          acciones.push(`<button type="button" class="boton-primario" data-bolsa-accion="abrir-llamar" data-participacion-ref="${escaparHTML(c.participacion_ref)}" data-nombre-visible="${escaparHTML(c.nombre_visible)}" data-orden="${numero(c.orden)}">Llamar</button>`);
        }
        if (c.ultimo_llamamiento?.llamamiento_ref && !lecturaPresentacion) {
          acciones.push(`<button type="button" class="boton-secundario" data-bolsa-accion="abrir-resultado" data-llamamiento-ref="${escaparHTML(c.ultimo_llamamiento.llamamiento_ref)}" data-participacion-ref="${escaparHTML(c.participacion_ref)}" data-nombre-visible="${escaparHTML(c.nombre_visible)}" data-orden="${numero(c.orden)}">Resultado</button>`);
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
            <td class="acciones-candidato">${acciones.join(" ")}</td>
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
      ${lecturaPresentacion ? '<section class="nota-pendiente" role="note"><strong>Presentación sintética de solo lectura.</strong> El historial no acredita contacto, envío ni entrega. Llamar y Resultado permanecen deshabilitados hasta disponer de reglas, canal y plazo aprobados.</section>' : ""}
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
          <table class="tabla-datos tabla-datos--candidatos">
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
                <th scope="col">Acciones</th>
              </tr>
            </thead>
            <tbody>
              ${cuerpoTabla}
            </tbody>
          </table>
        </div>
        ${paginacion}
      </section>
      ${renderizarModalFicha(typeof obtenerModalFicha === "function" ? obtenerModalFicha() : null)}
      ${renderizarModalContactos(typeof obtenerModalContactos === "function" ? obtenerModalContactos() : null)}
      ${renderizarModalLlamar(typeof obtenerModalLlamar === "function" ? obtenerModalLlamar() : null)}
      ${renderizarModalResultado(typeof obtenerModalResultado === "function" ? obtenerModalResultado() : null)}`;
  }

  function renderizarModalFicha(modal) {
    if (!modal || !modal.abierto) return "";
    const candidato = modal.candidato;
    const bolsa = modal.bolsa;
    if (!candidato || !bolsa) return "";
    const vigencia = bolsa.vigente_hasta
      ? `${instanteVisible(bolsa.vigente_desde)} — ${instanteVisible(bolsa.vigente_hasta)}`
      : `${instanteVisible(bolsa.vigente_desde)} — vigente`;
    const disponibilidad = candidato.disponible_desde
      ? `<div class="fila-resumen"><dt>Disponible desde</dt><dd>${escaparHTML(instanteVisible(candidato.disponible_desde))}</dd></div>`
      : "";
    const ultimoLlamamiento = candidato.ultimo_llamamiento
      ? `<div class="fila-resumen"><dt>Último llamamiento</dt><dd>${escaparHTML(etiquetaClave(candidato.ultimo_llamamiento.canal))} · ${escaparHTML(etiquetaClave(candidato.ultimo_llamamiento.resultado))}<br><small><time datetime="${escaparHTML(candidato.ultimo_llamamiento.comunicado_en)}">${escaparHTML(instanteVisible(candidato.ultimo_llamamiento.comunicado_en))}</time> · <code>${escaparHTML(candidato.ultimo_llamamiento.llamamiento_ref)}</code></small></dd></div>`
      : `<div class="fila-resumen"><dt>Último llamamiento</dt><dd>Sin llamamientos registrados</dd></div>`;

    return `
      <div class="modal-fondo" role="dialog" aria-modal="true" aria-labelledby="titulo-modal-ficha">
        <div class="modal-contenido" tabindex="-1">
          <div class="cabecera-panel">
            <h3 id="titulo-modal-ficha">Ficha de participación</h3>
            <button type="button" class="boton-cerrar" data-bolsa-accion="cerrar-ficha" aria-label="Cerrar ficha de participación">×</button>
          </div>
          <div class="cuerpo-panel">
            <p><strong>${escaparHTML(candidato.nombre_visible)}</strong> · <code>${escaparHTML(candidato.documento_enmascarado)}</code></p>
            <dl class="resumen-expediente">
              <div class="fila-resumen"><dt>Bolsa</dt><dd>${escaparHTML(bolsa.categoria)}<br><small>${escaparHTML(bolsa.categoria_clave)} · ${escaparHTML(etiquetaClave(bolsa.tipo_lista))}</small></dd></div>
              <div class="fila-resumen"><dt>Vigencia</dt><dd>${escaparHTML(vigencia)}</dd></div>
              <div class="fila-resumen"><dt>Orden</dt><dd>#${numero(candidato.orden)}</dd></div>
              <div class="fila-resumen"><dt>Situación</dt><dd><span class="estado-chip ${claseEstado(candidato.estado_clave)}">${escaparHTML(etiquetaClave(candidato.estado_clave))}</span> desde ${escaparHTML(instanteVisible(candidato.estado_desde))}</dd></div>
              ${disponibilidad}
              <div class="fila-resumen"><dt>Referencia de participación</dt><dd><code>${escaparHTML(candidato.participacion_ref)}</code></dd></div>
              ${ultimoLlamamiento}
            </dl>
          </div>
          <div class="acciones-vista">
            <button type="button" class="boton-secundario" data-bolsa-accion="cerrar-ficha">Cerrar</button>
          </div>
        </div>
      </div>`;
  }

  function renderizarModalContactos(modal) {
    if (!modal || !modal.abierto) return "";
    let contenido = "";
    if (modal.carga === "cargando") {
      contenido = '<p class="vacio-controlado" role="status" aria-busy="true">Cargando historial de contactos…</p>';
    } else if (modal.carga === "error") {
      contenido = `<p class="mensaje-error" role="alert">${escaparHTML(modal.error || "No se pudieron consultar los contactos.")}</p>`;
    } else if (!modal.contactos || modal.contactos.length === 0) {
      contenido = '<p class="vacio-controlado" role="status">No hay contactos previos registrados para este aspirante.</p>';
    } else {
      const filas = modal.contactos.map((ct) => `
        <tr data-contacto-ref="${escaparHTML(ct.contacto_ref)}">
          <td><span class="estado-chip neutro">${escaparHTML(etiquetaClave(ct.canal))}</span></td>
          <td><time datetime="${escaparHTML(ct.realizado_en)}">${escaparHTML(instanteVisible(ct.realizado_en))}</time></td>
          <td><span class="estado-chip ${claseEstado(ct.resultado_clave)}">${escaparHTML(etiquetaClave(ct.resultado_clave))}</span></td>
          <td><small>${escaparHTML(ct.anotacion || "—")}</small></td>
        </tr>
      `).join("");
      contenido = `
        <div class="tabla-contenedor">
          <table class="tabla-datos">
            <caption>Historial de comunicaciones y respuestas del aspirante</caption>
            <thead>
              <tr>
                <th scope="col">Canal</th>
                <th scope="col">Fecha y hora</th>
                <th scope="col">Resultado</th>
                <th scope="col">Anotación</th>
              </tr>
            </thead>
            <tbody>
              ${filas}
            </tbody>
          </table>
        </div>`;
    }

    return `
      <div class="modal-fondo" role="dialog" aria-modal="true" aria-labelledby="titulo-modal-contactos">
        <div class="modal-contenido">
          <div class="cabecera-panel">
            <h3 id="titulo-modal-contactos">Historial de contactos: ${escaparHTML(modal.nombreVisible || modal.participacionRef)}</h3>
            <button type="button" class="boton-cerrar" data-bolsa-accion="cerrar-contactos" aria-label="Cerrar">×</button>
          </div>
          <div class="cuerpo-panel">
            ${esLecturaPresentacion() === true ? '<p class="nota-pendiente">Historial sintético de solo lectura; no acredita contacto, envío ni entrega.</p>' : ""}
            ${contenido}
          </div>
          <div class="acciones-vista">
            <button type="button" class="boton-secundario" data-bolsa-accion="cerrar-contactos">Cerrar</button>
          </div>
        </div>
      </div>`;
  }

  function renderizarModalLlamar(modal) {
    if (!modal || !modal.abierto) return "";
    const errorHtml = modal.error
      ? `<div class="mensaje-error" role="alert"><p><strong>Error:</strong> ${escaparHTML(modal.error)}</p></div>`
      : "";
    const enviando = modal.carga === "enviando";

    return `
      <div class="modal-fondo" role="dialog" aria-modal="true" aria-labelledby="titulo-modal-llamar">
        <div class="modal-contenido">
          <div class="cabecera-panel">
            <h3 id="titulo-modal-llamar">Nuevo llamamiento (B7): ${escaparHTML(modal.nombreVisible || modal.participacionRef)}</h3>
            <button type="button" class="boton-cerrar" data-bolsa-accion="cerrar-llamar" aria-label="Cerrar">×</button>
          </div>
          <div class="cuerpo-panel">
            ${errorHtml}
            <p>Aspirante en orden <strong>#${numero(modal.orden)}</strong> de la bolsa. Se registrará la comunicación y el plazo de respuesta.</p>
            <form data-bolsa-form="llamar" data-participacion-ref="${escaparHTML(modal.participacionRef)}">
              <div class="campo-formulario">
                <label for="llamar-canal">Canal de comunicación *</label>
                <select id="llamar-canal" name="canal" required>
                  <option value="correo">Correo electrónico</option>
                  <option value="telefono">Teléfono</option>
                  <option value="sede">Sede electrónica</option>
                </select>
              </div>
              <div class="campo-formulario">
                <label for="llamar-comunicado-en">Fecha y hora de comunicación *</label>
                <input type="datetime-local" id="llamar-comunicado-en" name="comunicado_en" required>
              </div>
              <div class="campo-formulario">
                <label for="llamar-plazo-hasta">Plazo límite de respuesta *</label>
                <input type="datetime-local" id="llamar-plazo-hasta" name="plazo_respuesta_hasta" required>
              </div>
              <div class="campo-formulario">
                <label for="llamar-anotacion">Anotación u observaciones (opcional)</label>
                <textarea id="llamar-anotacion" name="anotacion" rows="3" maxlength="1024" placeholder="Detalle del intento de localización o notas del llamamiento…"></textarea>
              </div>
              <div class="campo-confirmacion">
                <label for="llamar-confirmacion">
                  <input type="checkbox" id="llamar-confirmacion" name="confirmacion" value="true" required>
                  Confirmo el llamamiento formal al aspirante conforme a las normas de gestión de bolsa.
                </label>
              </div>
              <div class="acciones-formulario">
                <button type="submit" class="boton-primario"${enviando ? " disabled" : ""}>${enviando ? "Registrando…" : "Registrar llamamiento"}</button>
                <button type="button" class="boton-secundario" data-bolsa-accion="cerrar-llamar">Cancelar</button>
              </div>
            </form>
          </div>
        </div>
      </div>`;
  }

  function renderizarModalResultado(modal) {
    if (!modal || !modal.abierto) return "";
    const errorHtml = modal.error
      ? `<div class="mensaje-error" role="alert"><p><strong>Error:</strong> ${escaparHTML(modal.error)}</p></div>`
      : "";
    const enviando = modal.carga === "enviando";

    return `
      <div class="modal-fondo" role="dialog" aria-modal="true" aria-labelledby="titulo-modal-resultado">
        <div class="modal-contenido">
          <div class="cabecera-panel">
            <h3 id="titulo-modal-resultado">Registrar resultado de llamamiento (B3)</h3>
            <button type="button" class="boton-cerrar" data-bolsa-accion="cerrar-resultado" aria-label="Cerrar">×</button>
          </div>
          <div class="cuerpo-panel">
            ${errorHtml}
            <p>Aspirante: <strong>${escaparHTML(modal.nombreVisible || modal.participacionRef)}</strong></p>
            <form data-bolsa-form="resultado" data-llamamiento-ref="${escaparHTML(modal.llamamientoRef)}">
              <div class="campo-formulario">
                <label for="resultado-clave">Resultado del llamamiento *</label>
                <select id="resultado-clave" name="resultado_clave" required>
                  <option value="">Seleccione un resultado…</option>
                  <option value="aceptado">Aceptado (pasa a situación Ocupado)</option>
                  <option value="renuncia">Renuncia (pasa a Renuncia pendiente)</option>
                  <option value="sin_respuesta">Sin respuesta (continúa Disponible tras salto)</option>
                </select>
              </div>
              <div class="campo-formulario">
                <label for="resultado-anotacion">Anotación administrativa (opcional)</label>
                <textarea id="resultado-anotacion" name="anotacion" rows="3" maxlength="1024" placeholder="Observaciones sobre la respuesta o justificante aportado…"></textarea>
              </div>
              <div class="campo-confirmacion">
                <label for="resultado-confirmacion">
                  <input type="checkbox" id="resultado-confirmacion" name="confirmacion" value="true" required>
                  Confirmo el resultado del llamamiento y los efectos sobre la posición en bolsa.
                </label>
              </div>
              <div class="acciones-formulario">
                <button type="submit" class="boton-primario"${enviando ? " disabled" : ""}>${enviando ? "Guardando…" : "Guardar resultado"}</button>
                <button type="button" class="boton-secundario" data-bolsa-accion="cerrar-resultado">Cancelar</button>
              </div>
            </form>
          </div>
        </div>
      </div>`;
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

  // Sin panel agregado compuesto, las bolsas constituidas (B12/B5) se muestran
  // igualmente: tienen su propia API y no dependen de los indicadores.
  function renderizarSoloBolsas(vista) {
    if (vista === "bolsa-candidatos") return renderizarCandidatosBolsa();
    return `
      ${encabezadoVista("Gestión interna de Bolsas", "Cuadro de mando", "Bolsas constituidas y su desglose por situación. Los indicadores agregados del panel interno no están compuestos todavía.")}
      ${renderizarCuadroB12()}`;
  }

  return Object.freeze({ actualizarContextoSesion, esActivo, etiquetaFuente, renderizarSoloBolsas, renderizarVista });
}
