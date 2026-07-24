/** Componentes HTML puros de la superficie de expedientes. */

export function escaparHTML(valor) {
  return String(valor ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

function estadoClave(estado) {
  return `ct-fase-${estado}`;
}

function textoEstado(clave, t) {
  return t(`fase_${clave}`);
}

export function renderizarEstadoCarga(estado, t) {
  const configuracion = {
    cargando: ["cargando_titulo", "cargando_detalle", "informacion"],
    error: ["error_titulo", "estado_error_carga", "peligro"],
    denegado: ["denegado_titulo", "estado_denegado", "peligro"],
    vacio: ["vacio_titulo", "vacio_detalle", "neutro"],
  }[estado.carga];
  if (!configuracion) return "";
  const [titulo, detallePredeterminado, tono] = configuracion;
  const detalle = estado.mensaje_clave || detallePredeterminado;
  return `<section class="ct-exp-estado-global ct-tono-${tono}" role="${tono === "peligro" ? "alert" : "status"}"
    ${estado.carga === "cargando" ? 'aria-busy="true"' : ""} tabindex="-1">
    <h3>${escaparHTML(t(titulo))}</h3>
    <p>${escaparHTML(t(detalle))}</p>
    ${estado.carga === "error"
    ? `<button type="button" class="boton-secundario" data-ct-exp-accion="reintentar">${escaparHTML(t("reintentar"))}</button>`
    : ""}
  </section>`;
}

function opcionFiltro(valor, etiqueta, seleccionado) {
  return `<option value="${escaparHTML(valor)}"${valor === seleccionado ? " selected" : ""}>${escaparHTML(etiqueta)}</option>`;
}

function renderizarTrabajoOperativo(cuadro, t) {
  const tareas = cuadro.expedientes
    .filter(({ estado_clave: estado }) => estado !== "completado")
    .slice(0, 3);
  const distribucion = [...new Set(cuadro.expedientes.map(({ fase_actual: fase }) => fase))]
    .map((fase) => ({
      fase,
      total: cuadro.expedientes.filter(({ fase_actual: actual }) => actual === fase).length,
    }));
  const primera = tareas[0] ?? cuadro.expedientes[0];
  return `<section class="ct-exp-operativo" aria-labelledby="ct-exp-operativo-titulo">
    <header>
      <p class="sobrelinea">${escaparHTML(t("trabajo_sobrelinea"))}</p>
      <h3 id="ct-exp-operativo-titulo">${escaparHTML(t("trabajo_titulo"))}</h3>
      <p>${escaparHTML(t("trabajo_descripcion"))}</p>
    </header>
    <article class="ct-exp-mis-tareas">
      <h4>${escaparHTML(t("mis_tareas"))}</h4>
      <ul>${tareas.map((expediente) => `<li>
        <span><strong>${escaparHTML(expediente.numero_visible)}</strong>
          <small>${escaparHTML(expediente.categoria)} · ${escaparHTML(expediente.fase_actual)}</small>
        </span>
        <button type="button" class="boton-terciario"
          data-ct-exp-abrir="${escaparHTML(expediente.expediente_ref)}">${escaparHTML(t("abrir"))}</button>
      </li>`).join("")}</ul>
    </article>
    <article class="ct-exp-distribucion">
      <h4>${escaparHTML(t("distribucion_fases"))}</h4>
      <dl>${distribucion.map(({ fase, total }) => `<div>
        <dt>${escaparHTML(fase)}</dt><dd>${total}</dd>
      </div>`).join("")}</dl>
    </article>
    <aside class="ct-exp-accesos">
      <h4>${escaparHTML(t("accesos_rapidos"))}</h4>
      <button type="button" class="boton-primario"
        data-ct-exp-vista="alta">${escaparHTML(t("crear_peticion"))}</button>
      ${primera ? `<button type="button" class="boton-secundario"
        data-ct-exp-abrir="${escaparHTML(primera.expediente_ref)}">${escaparHTML(t("continuar_tramitacion"))}</button>` : ""}
    </aside>
  </section>`;
}

export function renderizarCuadro(estado, t) {
  const cuadro = estado.cuadro;
  if (!cuadro) return renderizarEstadoCarga(estado, t);
  const estados = [
    ["", t("filtro_todos")],
    ["pendiente", t("fase_pendiente")],
    ["en_curso", t("fase_en_curso")],
    ["espera", t("fase_espera")],
    ["completado", t("fase_completado")],
    ["incidencia", t("fase_incidencia")],
    ["cancelado", t("fase_cancelado")],
  ];
  const fases = [...new Set(cuadro.expedientes.map(({ fase_actual: fase }) => fase))]
    .sort((a, b) => a.localeCompare(b, "es"));
  const indicadores = `<section class="ct-exp-indicadores" aria-label="${escaparHTML(t("indicadores"))}">
    ${cuadro.indicadores.map((indicador) => `<article class="ct-exp-indicador ct-tono-${escaparHTML(indicador.tono)}">
      <span>${escaparHTML(indicador.etiqueta)}</span>
      <strong>${escaparHTML(indicador.valor)}</strong>
    </article>`).join("")}
  </section>`;
  const filtros = `<form class="ct-exp-filtros" data-ct-exp-filtros aria-label="${escaparHTML(t("filtros"))}">
    <label>
      <span>${escaparHTML(t("filtro_texto"))}</span>
      <input type="search" name="texto" value="${escaparHTML(estado.filtros.texto)}"
        placeholder="${escaparHTML(t("filtro_texto_placeholder"))}">
    </label>
    <label>
      <span>${escaparHTML(t("filtro_estado"))}</span>
      <select name="estado">${estados.map(([valor, etiqueta]) => (
    opcionFiltro(valor, etiqueta, estado.filtros.estado)
  )).join("")}</select>
    </label>
    <label>
      <span>${escaparHTML(t("filtro_fase"))}</span>
      <select name="fase">
        ${opcionFiltro("", t("filtro_todos"), estado.filtros.fase)}
        ${fases.map((fase) => opcionFiltro(fase.toLocaleLowerCase("es-ES"), fase, estado.filtros.fase)).join("")}
      </select>
    </label>
    <div class="ct-exp-filtros-acciones">
      <button type="submit" class="boton-primario">${escaparHTML(t("aplicar_filtros"))}</button>
      <button type="button" class="boton-secundario" data-ct-exp-accion="limpiar-filtros">${escaparHTML(t("limpiar_filtros"))}</button>
    </div>
  </form>`;
  const filas = cuadro.expedientes.map((expediente) => `<tr>
    <th scope="row">${escaparHTML(expediente.numero_visible)}</th>
    <td>${escaparHTML(expediente.centro)}</td>
    <td>${escaparHTML(expediente.categoria)}</td>
    <td>${escaparHTML(expediente.modalidad)}</td>
    <td><span class="ct-exp-chip ${estadoClave(expediente.estado_clave)}">${escaparHTML(expediente.estado)}</span></td>
    <td>${escaparHTML(expediente.fase_actual)}</td>
    <td>${escaparHTML(expediente.plazo)}</td>
    <td><button type="button" class="boton-terciario" data-ct-exp-abrir="${escaparHTML(expediente.expediente_ref)}">${escaparHTML(t("abrir"))}</button></td>
  </tr>`).join("");
  const tabla = `<section class="panel ct-exp-listado">
    <div class="cabecera-panel">
      <h3>${escaparHTML(t("tabla_expedientes"))}</h3>
      <span class="estado-chip info">${escaparHTML(t("resultados", { total: cuadro.expedientes.length }))}</span>
    </div>
    <div class="tabla-contenedor tabla-contenedor--prioritaria" tabindex="0">
      <table class="tabla-datos tabla-datos--prioritaria">
        <caption>${escaparHTML(t("tabla_expedientes"))}</caption>
        <thead><tr>
          <th scope="col">${escaparHTML(t("columna_numero"))}</th>
          <th scope="col">${escaparHTML(t("columna_centro"))}</th>
          <th scope="col">${escaparHTML(t("columna_categoria"))}</th>
          <th scope="col">${escaparHTML(t("columna_modalidad"))}</th>
          <th scope="col">${escaparHTML(t("columna_estado"))}</th>
          <th scope="col">${escaparHTML(t("columna_fase"))}</th>
          <th scope="col">${escaparHTML(t("columna_plazo"))}</th>
          <th scope="col">${escaparHTML(t("columna_acciones"))}</th>
        </tr></thead>
        <tbody>${filas}</tbody>
      </table>
    </div>
  </section>`;
  return `${indicadores}${renderizarTrabajoOperativo(cuadro, t)}${filtros}${estado.carga === "vacio"
    ? renderizarEstadoCarga(estado, t) : tabla}`;
}

function renderizarFases(expediente, t) {
  return `<nav class="ct-exp-progreso" aria-label="${escaparHTML(t("fases_expediente"))}">
    <ol>${expediente.fases.map((fase) => `<li class="${estadoClave(fase.estado_clave)}"
      ${fase.estado_clave === "en_curso" ? 'aria-current="step"' : ""}>
      <span class="ct-exp-numero-fase" aria-hidden="true">${fase.orden}</span>
      <span>${escaparHTML(fase.etiqueta)}</span>
      <small>${escaparHTML(textoEstado(fase.estado_clave, t))}</small>
    </li>`).join("")}</ol>
  </nav>`;
}

function renderizarCabecera(expediente, t) {
  return `<section class="ct-exp-cabecera-expediente">
    <div>
      <p class="sobrelinea">${escaparHTML(t("expediente_etiqueta"))}</p>
      <h3>${escaparHTML(expediente.numero_visible)}</h3>
      <details class="ct-exp-detalle-tecnico">
        <summary>${escaparHTML(t("metadatos_tecnicos"))}</summary>
        <dl class="ct-exp-flujo">
          <div><dt>${escaparHTML(t("referencia_interna"))}</dt><dd><code>${escaparHTML(expediente.expediente_ref)}</code></dd></div>
          <div><dt>${escaparHTML(t("flujo_definicion"))}</dt><dd>${escaparHTML(expediente.flujo_ref)}</dd></div>
          <div><dt>${escaparHTML(t("flujo_version"))}</dt><dd>${expediente.flujo_version}</dd></div>
          <div><dt>${escaparHTML(t("flujo_huella"))}</dt><dd><code>${escaparHTML(expediente.flujo_huella)}</code></dd></div>
        </dl>
      </details>
    </div>
    <dl>${expediente.cabecera.map((campo) => `<div>
      <dt>${escaparHTML(campo.etiqueta)}</dt>
      <dd class="ct-tono-${escaparHTML(campo.tono)}">${escaparHTML(campo.valor)}</dd>
    </div>`).join("")}</dl>
  </section>`;
}

function renderizarTareas(expediente, tareaRef, t) {
  return `<nav class="ct-exp-tareas" aria-label="${escaparHTML(t("tareas_expediente"))}">
    <ol>${expediente.tareas.map((tarea) => `<li>
      <button type="button" data-ct-exp-tarea="${escaparHTML(tarea.tarea_ref)}"
        class="${estadoClave(tarea.estado_clave)}"
        ${tarea.tarea_ref === tareaRef ? 'aria-current="step"' : ""}>
        <span class="ct-exp-numero-tarea">${tarea.orden}</span>
        <span><strong>${escaparHTML(tarea.etiqueta)}</strong><small>${escaparHTML(tarea.estado)}</small></span>
      </button>
    </li>`).join("")}</ol>
  </nav>`;
}

function renderizarOpcionesSelect(campo) {
  return campo.opciones.map((opcion) => `<option value="${escaparHTML(opcion.clave)}"
    ${opcion.clave === campo.valor ? " selected" : ""}>${escaparHTML(opcion.etiqueta)}</option>`).join("");
}

function renderizarCampoEditable(campo, editable) {
  const id = `ct-exp-campo-${campo.clave}`;
  const requerido = campo.obligatorio ? " required aria-required=\"true\"" : "";
  const bloqueado = editable ? "" : " disabled";
  const etiqueta = `<span>${escaparHTML(campo.etiqueta)}${campo.obligatorio ? " *" : ""}</span>`;
  if (campo.control === "area") {
    return `<label class="ct-exp-campo ct-exp-campo-ancho" for="${id}">${etiqueta}
      <textarea id="${id}" name="${escaparHTML(campo.clave)}"${requerido}${bloqueado}>${escaparHTML(campo.valor)}</textarea>
    </label>`;
  }
  if (campo.control === "seleccion") {
    return `<label class="ct-exp-campo" for="${id}">${etiqueta}
      <select id="${id}" name="${escaparHTML(campo.clave)}"${requerido}${bloqueado}>${renderizarOpcionesSelect(campo)}</select>
    </label>`;
  }
  if (campo.control === "radio") {
    return `<fieldset class="ct-exp-radios"><legend>${etiqueta}</legend>
      ${campo.opciones.map((opcion) => `<label>
        <input type="radio" name="${escaparHTML(campo.clave)}" value="${escaparHTML(opcion.clave)}"
          ${opcion.clave === campo.valor ? " checked" : ""}${requerido}${bloqueado}>
        <span>${escaparHTML(opcion.etiqueta)}</span>
      </label>`).join("")}
    </fieldset>`;
  }
  const tipo = campo.control === "fecha" ? "date" : "text";
  const modo = campo.control === "importe" ? ' inputmode="decimal"' : "";
  return `<label class="ct-exp-campo" for="${id}">${etiqueta}
    <input id="${id}" name="${escaparHTML(campo.clave)}" type="${tipo}"${modo}
      value="${escaparHTML(campo.valor)}"${requerido}${bloqueado}>
  </label>`;
}

function renderizarCampo(campo, editable) {
  if (campo.control !== "solo_lectura") return renderizarCampoEditable(campo, editable);
  return `<div class="ct-exp-dato ct-tono-${escaparHTML(campo.tono)}">
    <dt>${escaparHTML(campo.etiqueta)}</dt>
    <dd>${escaparHTML(campo.valor)}</dd>
  </div>`;
}

function renderizarTablaPanel(panel, t) {
  if (panel.columnas.length === 0) return "";
  return `<div class="tabla-contenedor" tabindex="0">
    <table class="tabla-datos ct-exp-tabla-panel">
      <caption>${escaparHTML(t("tabla_panel", { titulo: panel.titulo }))}</caption>
      <thead><tr>${panel.columnas.map((columna) => `<th scope="col">${escaparHTML(columna.etiqueta)}</th>`).join("")}</tr></thead>
      <tbody>${panel.filas.map((fila) => `<tr>${fila.celdas.map((celda, indice) => (
    indice === 0
      ? `<th scope="row">${escaparHTML(celda)}</th>`
      : `<td>${escaparHTML(celda)}</td>`
  )).join("")}</tr>`).join("")}</tbody>
    </table>
  </div>`;
}

function renderizarPanel(panel, t, editable) {
  const panelTieneEdicion = panel.campos.some(({ control }) => control !== "solo_lectura");
  const contenidoCampos = panel.campos.length === 0
    ? "" : (panelTieneEdicion
      ? `<div class="ct-exp-campos">${panel.campos.map((campo) => (
        renderizarCampo(campo, editable)
      )).join("")}</div>`
      : `<dl class="ct-exp-datos">${panel.campos.map((campo) => (
        renderizarCampo(campo, editable)
      )).join("")}</dl>`);
  return `<section class="ct-exp-panel ct-exp-panel-${escaparHTML(panel.tipo)}">
    <header><h4>${escaparHTML(panel.titulo)}</h4>
      ${panel.descripcion ? `<p>${escaparHTML(panel.descripcion)}</p>` : ""}
    </header>
    ${contenidoCampos}
    ${renderizarTablaPanel(panel, t)}
    ${panel.campos.length === 0 && panel.columnas.length === 0
    ? `<p class="ct-exp-vacio">${escaparHTML(t("panel_sin_datos"))}</p>` : ""}
  </section>`;
}

function claseBoton(variante) {
  return {
    primaria: "boton-primario",
    secundaria: "boton-secundario",
    peligro: "boton-peligro",
  }[variante] || "boton-secundario";
}

function renderizarRecibo(recibo, t, locale, zonaHoraria) {
  if (!recibo) return "";
  const fecha = new Intl.DateTimeFormat(locale, {
    dateStyle: "medium",
    timeStyle: "medium",
    timeZone: zonaHoraria,
  }).format(new Date(recibo.registrada_en));
  return `<section class="ct-exp-recibo" role="status" aria-live="polite" tabindex="-1" data-ct-exp-recibo>
    <h4>${escaparHTML(t("recibo_titulo"))}</h4>
    <p>${escaparHTML(t("recibo_descripcion"))}</p>
    <dl>
      <div><dt>${escaparHTML(t("recibo_referencia"))}</dt><dd><code>${escaparHTML(recibo.recibo_ref)}</code></dd></div>
      <div><dt>${escaparHTML(t("recibo_expediente"))}</dt><dd>${escaparHTML(recibo.numero_visible)}</dd></div>
      <div><dt>${escaparHTML(t("recibo_version"))}</dt><dd>${escaparHTML(recibo.version)}</dd></div>
      <div><dt>${escaparHTML(t("recibo_actuacion"))}</dt><dd>${escaparHTML(recibo.actuacion)}</dd></div>
      <div><dt>${escaparHTML(t("recibo_estado"))}</dt><dd>${escaparHTML(recibo.estado_resultante)}</dd></div>
      <div><dt>${escaparHTML(t("recibo_fecha"))}</dt><dd>${escaparHTML(fecha)}</dd></div>
    </dl>
  </section>`;
}

function renderizarTarea(expediente, tareaRef, estado, t, locale, zonaHoraria) {
  const tarea = expediente.tareas.find(({ tarea_ref: referencia }) => referencia === tareaRef)
    ?? expediente.tareas[0];
  if (!tarea) return "";
  const editable = tarea.acciones.some((accion) => (
    accion.tipo === "efecto" && accion.disponible === true
  )) && !estado.ocupado && !estado.actualizacion_pendiente;
  return `<article class="ct-exp-tarea-actual" aria-labelledby="ct-exp-tarea-titulo">
    <header class="ct-exp-tarea-cabecera">
      <div><p class="sobrelinea">${escaparHTML(t("tarea_actual"))} · ${escaparHTML(t("posicion_tarea", {
    actual: tarea.orden,
    total: expediente.tareas.length,
  }))}</p>
        <h3 id="ct-exp-tarea-titulo" tabindex="-1">${escaparHTML(tarea.etiqueta)}</h3>
        <p>${escaparHTML(tarea.descripcion)}</p>
      </div>
      <span class="ct-exp-chip ${estadoClave(tarea.estado_clave)}">${escaparHTML(tarea.estado)}</span>
    </header>
    <dl class="ct-exp-metadata">
      <div><dt>${escaparHTML(t("unidad"))}</dt><dd>${escaparHTML(tarea.unidad)}</dd></div>
      <div><dt>${escaparHTML(t("responsable"))}</dt><dd>${escaparHTML(tarea.responsable)}</dd></div>
      <div><dt>${escaparHTML(t("entrada"))}</dt><dd>${escaparHTML(tarea.entrada)}</dd></div>
      <div><dt>${escaparHTML(t("salida"))}</dt><dd>${escaparHTML(tarea.salida || "—")}</dd></div>
      <div><dt>${escaparHTML(t("tiempo"))}</dt><dd>${escaparHTML(tarea.tiempo)}</dd></div>
      <div><dt>${escaparHTML(t("recibo"))}</dt><dd><code>${escaparHTML(tarea.recibo_ref || "—")}</code></dd></div>
      <div><dt>${escaparHTML(t("decision"))}</dt><dd><code>${escaparHTML(tarea.decision_ref || "—")}</code></dd></div>
    </dl>
    <form data-ct-exp-tarea-form aria-label="${escaparHTML(t("formulario_tarea", { tarea: tarea.etiqueta }))}">
      ${editable ? "" : `<p class="ct-exp-solo-lectura">${escaparHTML(t("tarea_solo_lectura"))}</p>`}
      ${tarea.paneles.some((panel) => panel.campos.some(({ obligatorio }) => obligatorio))
    ? `<p class="ct-exp-obligatorios">${escaparHTML(t("campos_obligatorios"))}</p>` : ""}
      <div class="ct-exp-paneles">${tarea.paneles.map((panel) => (
        renderizarPanel(panel, t, editable)
      )).join("")}</div>
      ${tarea.acciones.length ? `<div class="ct-exp-acciones">${tarea.acciones.map((accion, indice) => {
    const motivoId = `ct-exp-accion-motivo-${tarea.orden}-${indice}`;
    const bloqueada = accion.disponible !== true || estado.ocupado || estado.actualizacion_pendiente;
    return `<span class="ct-exp-accion">
        <button type="button" class="${claseBoton(accion.variante)}"
          data-ct-exp-efecto="${escaparHTML(accion.accion_ref)}"
          data-ct-exp-confirmacion="${escaparHTML(accion.confirmacion)}"
          ${bloqueada ? "disabled" : ""}
          ${accion.disponible ? "" : `aria-describedby="${motivoId}"`}>${escaparHTML(accion.etiqueta)}</button>
        ${accion.disponible ? "" : `<small id="${motivoId}">${escaparHTML(t("accion_no_disponible", {
    motivo: accion.motivo_no_disponible,
  }))}</small>`}
      </span>`;
  }).join("")}
        ${estado.ocupado ? `<button type="button" class="boton-secundario" data-ct-exp-accion="cancelar">${escaparHTML(t("cancelar_espera"))}</button>` : ""}
      </div>` : ""}
    </form>
    ${renderizarRecibo(estado.recibo, t, locale, zonaHoraria)}
  </article>`;
}

export function renderizarExpediente(estado, t, locale, zonaHoraria) {
  const expediente = estado.expediente;
  if (!expediente) return `<section class="ct-exp-estado-global"><p>${escaparHTML(t("expediente_sin_seleccionar"))}</p>
    <button type="button" class="boton-secundario" data-ct-exp-vista="cuadro">${escaparHTML(t("volver_cuadro"))}</button></section>`;
  return `${renderizarCabecera(expediente, t)}
    ${renderizarFases(expediente, t)}
    <div class="ct-exp-tramitacion">
      ${renderizarTareas(expediente, estado.tarea_ref, t)}
      ${renderizarTarea(expediente, estado.tarea_ref, estado, t, locale, zonaHoraria)}
    </div>`;
}

export function renderizarDocumentos(estado, t) {
  const expediente = estado.expediente;
  const indice = estado.documentos;
  if (!expediente || !indice) return renderizarExpediente(estado, t, "es-ES", "Europe/Madrid");
  return `${renderizarCabecera(expediente, t)}
    <header class="ct-exp-subcabecera"><h3>${escaparHTML(t("documentos_titulo"))}</h3><p>${escaparHTML(t("documentos_descripcion"))}</p></header>
    <div class="tabla-contenedor tabla-contenedor--prioritaria" tabindex="0">
      <table class="tabla-datos tabla-datos--prioritaria">
        <caption>${escaparHTML(t("documentos_tabla"))}</caption>
        <thead><tr>
          <th scope="col">${escaparHTML(t("documento"))}</th><th scope="col">${escaparHTML(t("tipo"))}</th>
          <th scope="col">${escaparHTML(t("version"))}</th><th scope="col">${escaparHTML(t("columna_estado"))}</th>
          <th scope="col">${escaparHTML(t("firma"))}</th><th scope="col">${escaparHTML(t("fecha"))}</th>
          <th scope="col">${escaparHTML(t("descarga"))}</th>
        </tr></thead>
        <tbody>${indice.documentos.map((documento) => `<tr>
          <th scope="row">${escaparHTML(documento.titulo)}<code>${escaparHTML(documento.documento_ref)}</code></th>
          <td>${escaparHTML(documento.tipo)}</td><td>${documento.version}</td>
          <td>${escaparHTML(documento.estado)}</td><td>${escaparHTML(documento.firma)}</td>
          <td>${escaparHTML(documento.fecha)}</td><td>${documento.descarga_disponible
    ? `<span class="ct-exp-descarga-pendiente">${escaparHTML(t("descarga_conector_pendiente"))}</span>`
    : escaparHTML(t("no_disponible"))}</td>
        </tr>`).join("")}</tbody>
      </table>
    </div>`;
}

export function renderizarAuditoria(estado, t) {
  const expediente = estado.expediente;
  const auditoria = estado.auditoria;
  if (!expediente || !auditoria) return renderizarExpediente(estado, t, "es-ES", "Europe/Madrid");
  return `${renderizarCabecera(expediente, t)}
    <header class="ct-exp-subcabecera"><h3>${escaparHTML(t("auditoria_titulo"))}</h3><p>${escaparHTML(t("auditoria_descripcion"))}</p></header>
    <div class="tabla-contenedor tabla-contenedor--prioritaria" tabindex="0">
      <table class="tabla-datos tabla-datos--prioritaria ct-exp-tabla-auditoria">
        <caption>${escaparHTML(t("auditoria_tabla"))}</caption>
        <thead><tr>
          <th scope="col">${escaparHTML(t("fecha"))}</th><th scope="col">${escaparHTML(t("columna_fase"))}</th>
          <th scope="col">${escaparHTML(t("actuacion"))}</th><th scope="col">${escaparHTML(t("actor"))}</th>
          <th scope="col">${escaparHTML(t("unidad"))}</th><th scope="col">${escaparHTML(t("columna_estado"))}</th>
          <th scope="col">${escaparHTML(t("observaciones"))}</th><th scope="col">${escaparHTML(t("documento_asociado"))}</th>
        </tr></thead>
        <tbody>${[...auditoria.actuaciones].reverse().map((actuacion) => `<tr>
          <th scope="row">${escaparHTML(actuacion.fecha)}</th><td>${escaparHTML(actuacion.fase)}</td>
          <td>${escaparHTML(actuacion.accion)}</td><td>${escaparHTML(actuacion.actor)}</td>
          <td>${escaparHTML(actuacion.unidad)}</td><td>${escaparHTML(actuacion.estado)}</td>
          <td>${escaparHTML(actuacion.observaciones)}</td><td><code>${escaparHTML(actuacion.documento_ref || "—")}</code></td>
        </tr>`).join("")}</tbody>
      </table>
    </div>`;
}
