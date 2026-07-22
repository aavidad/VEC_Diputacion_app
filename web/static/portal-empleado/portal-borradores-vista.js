const FASE_INICIAL = "inicial";
const FASE_CARGANDO = "cargando";
const FASE_ERROR = "error";

function instanteVisible(instante) {
  if (!instante) return "Sin fecha";
  const fecha = new Date(instante);
  if (!Number.isFinite(fecha.getTime())) return instante;
  return new Intl.DateTimeFormat("es-ES", {
    dateStyle: "short",
    timeStyle: "short",
    timeZone: "Europe/Madrid",
  }).format(fecha);
}

export function crearRenderizadorBorradores({
  escaparHTML,
  estado,
  motivoSeleccionado,
  plantillaSeleccionada,
} = {}) {
  if (typeof escaparHTML !== "function" || !estado
    || typeof motivoSeleccionado !== "function"
    || typeof plantillaSeleccionada !== "function") {
    throw new TypeError("dependencias de presentación de borradores no válidas");
  }

  function renderError(error, contexto) {
    if (!error) return "";
    const conflicto = error.tipoConflicto === "idempotencia"
      ? "Conflicto de idempotencia (HTTP 409)"
      : (error.tipoConflicto === "cas" ? "Conflicto de revisión CAS (HTTP 412)" : contexto);
    return `
      <section class="borrador-error" role="alert" aria-labelledby="titulo-error-borrador">
        <div>
          <p class="sobrelinea">${escaparHTML(conflicto)}</p>
          <h3 id="titulo-error-borrador">${escaparHTML(error.mensaje)}</h3>
          ${error.conservarCambiosLocales ? "<p>Los cambios introducidos continúan en este editor y no se han descartado.</p>" : ""}
        </div>
        <dl class="metadatos-error">
          <div><dt>Código</dt><dd><code>${escaparHTML(error.codigo)}</code></dd></div>
          ${error.correlacion ? `<div><dt>Correlación</dt><dd><code>${escaparHTML(error.correlacion)}</code></dd></div>` : ""}
        </dl>
      </section>`;
  }

  function renderEstadoFuente() {
    if (estado.faseLista === FASE_CARGANDO || estado.faseLista === FASE_INICIAL) {
      return `
        <section class="panel borradores-cargando" role="status" aria-live="polite">
          <div class="cuerpo-panel"><strong>Comprobando acceso y cargando borradores…</strong><p>La identidad procede exclusivamente del canal interno autenticado. No se usan cookies, tokens ni almacenamiento local.</p></div>
        </section>`;
    }
    if (estado.faseLista === FASE_ERROR) {
      return `
        ${renderError(estado.errorLista, "Servicio de borradores no disponible")}
        <section class="panel"><div class="cuerpo-panel vacio-controlado">
          <p><strong>La bandeja no puede operar sin el backend autenticado.</strong></p>
          <p>No se muestran borradores ficticios ni una copia local alternativa.</p>
          <button type="button" class="boton-primario" data-borrador-accion="borradores-recargar">Reintentar conexión</button>
        </div></section>`;
    }
    return "";
  }

  function opcionesFiltroCategorias() {
    return (estado.opciones?.categorias || []).map((item) => `
      <option value="${escaparHTML(item.clave)}" ${estado.filtro.categoria === item.clave ? "selected" : ""}>${escaparHTML(item.etiqueta)}</option>`).join("");
  }

  function renderLista() {
    const elementos = estado.lista?.elementos || [];
    const total = estado.lista?.paginacion?.total ?? 0;
    const avisoActualizacion = estado.errorLista ? `
      <div class="borrador-aviso" role="status">
        <strong>No se pudo actualizar la bandeja.</strong>
        <span>${escaparHTML(estado.errorLista.mensaje)} · <code>${escaparHTML(estado.errorLista.codigo)}</code></span>
      </div>` : "";
    const filas = elementos.map((item) => {
      const seleccionada = item.referencia_estado.referencia === estado.referenciaSeleccionada;
      return `
        <tr aria-selected="${seleccionada}">
          <td><strong>${escaparHTML(item.titulo)}</strong><small>${escaparHTML(item.identificador_publico)}</small></td>
          <td><span class="estado-chip ${seleccionada ? "info" : "neutro"}">Rev. ${escaparHTML(item.referencia_estado.revision)}</span></td>
          <td><time datetime="${escaparHTML(item.actualizada_en)}">${escaparHTML(instanteVisible(item.actualizada_en))}</time></td>
          <td><button type="button" class="boton-terciario" data-borrador-accion="borradores-abrir" data-id="${escaparHTML(item.referencia_estado.referencia)}" ${estado.guardando ? "disabled" : ""}>${seleccionada ? "Abierto" : "Abrir"}</button></td>
        </tr>`;
    }).join("") || '<tr><td colspan="4" class="vacio-controlado">No hay borradores para este filtro y ámbito.</td></tr>';
    return `
      <section class="panel bandeja-borradores" aria-labelledby="titulo-bandeja-borradores">
        <div class="cabecera-panel">
          <div><h3 id="titulo-bandeja-borradores">Bandeja de borradores</h3><p>${escaparHTML(total)} registros en el ámbito autorizado</p></div>
          <button type="button" class="boton-primario" data-borrador-accion="borradores-nuevo" ${estado.opciones.capacidades.crear && !estado.guardando ? "" : "disabled"}>Nuevo borrador</button>
        </div>
        ${avisoActualizacion}
        <form class="filtros-borradores" data-borrador-form="filtros" ${estado.guardando ? 'inert aria-busy="true"' : ""}>
          <label class="campo"><span>Buscar por título</span><input type="search" name="texto" maxlength="180" value="${escaparHTML(estado.filtro.texto)}" autocomplete="off"></label>
          <label class="campo"><span>Categoría</span><select name="categoria"><option value="">Todas</option>${opcionesFiltroCategorias()}</select></label>
          <button type="submit" class="boton-secundario">Aplicar filtros</button>
          <button type="button" class="boton-terciario" data-borrador-accion="borradores-recargar">Actualizar</button>
        </form>
        <div class="tabla-contenedor tabla-borradores">
          <table class="tabla-datos">
            <caption>Borradores editables devueltos por la API interna</caption>
            <thead><tr><th scope="col">Borrador</th><th scope="col">Estado</th><th scope="col">Actualizado</th><th scope="col">Acción</th></tr></thead>
            <tbody>${filas}</tbody>
          </table>
        </div>
        <nav class="paginacion-borradores" aria-label="Paginación de borradores">
          <button type="button" class="boton-terciario" data-borrador-accion="borradores-pagina-anterior" ${estado.pagina === 0 || estado.guardando ? "disabled" : ""}>← Anterior</button>
          <span>Página ${estado.pagina + 1} · ${elementos.length} visibles</span>
          <button type="button" class="boton-terciario" data-borrador-accion="borradores-pagina-siguiente" ${estado.lista?.paginacion?.siguiente_cursor && !estado.guardando ? "" : "disabled"}>Siguiente →</button>
        </nav>
      </section>`;
  }

  function opcionesIndice(items, campoEtiqueta, seleccionado) {
    return items.map((item, indice) => `
      <option value="${indice}" ${Number(seleccionado) === indice ? "selected" : ""}>${escaparHTML(item[campoEtiqueta])} · v${escaparHTML(item.version)}</option>`).join("");
  }

  function renderIdentidad() {
    if (estado.modoEditor === "actualizar") {
      const detalle = estado.detalle;
      return `
        <fieldset class="grupo-editor grupo-identidad">
          <legend>Identidad gobernada de solo lectura</legend>
          <dl class="resumen-expediente resumen-editor">
            <div class="fila-resumen"><dt>Código público</dt><dd>${escaparHTML(detalle.identificador_publico)}</dd></div>
            <div class="fila-resumen"><dt>Versión pública</dt><dd>${escaparHTML(detalle.codigo_version_publica)}</dd></div>
            <div class="fila-resumen"><dt>Expediente</dt><dd>${escaparHTML(detalle.expediente_ref)}</dd></div>
            <div class="fila-resumen"><dt>Referencia</dt><dd><code>${escaparHTML(detalle.referencia_estado.referencia)}</code></dd></div>
          </dl>
        </fieldset>`;
    }
    return `
      <fieldset class="grupo-editor grupo-identidad">
        <legend>1. Identidad y plantilla</legend>
        <div class="campos-editor-dos">
          <label class="campo campo-ancho"><span>Plantilla gobernada</span><select required data-borrador-ruta="plantilla_indice">${opcionesIndice(estado.opciones.plantillas, "nombre", estado.editor.plantilla_indice)}</select></label>
          <label class="campo"><span>Identificador público</span><input required maxlength="80" pattern="[a-z0-9][a-z0-9-]{2,79}" data-borrador-ruta="identificador_publico" value="${escaparHTML(estado.editor.identificador_publico)}" aria-describedby="ayuda-identificador-publico"></label>
          <label class="campo"><span>Código de versión pública</span><input required maxlength="80" pattern="[a-z0-9][a-z0-9._-]{0,79}" data-borrador-ruta="codigo_version_publica" value="${escaparHTML(estado.editor.codigo_version_publica)}"></label>
          <label class="campo campo-ancho"><span>Referencia de expediente</span><input required maxlength="512" data-borrador-ruta="expediente_ref" value="${escaparHTML(estado.editor.expediente_ref)}"></label>
          <p id="ayuda-identificador-publico" class="ayuda-campo campo-ancho">Use una referencia estable en minúsculas; la identidad no podrá cambiarse al actualizar.</p>
        </div>
      </fieldset>`;
  }

  function renderCategorias() {
    const seleccionadas = new Set(estado.editor.contenido_editable.categorias);
    return `
      <fieldset class="selector-categorias campo-ancho">
        <legend>Categorías</legend>
        <div>${estado.opciones.categorias.map((item) => `
          <label><input type="checkbox" data-borrador-ruta="contenido_editable.categorias" value="${escaparHTML(item.clave)}" ${seleccionadas.has(item.clave) ? "checked" : ""}> <span>${escaparHTML(item.etiqueta)}</span></label>`).join("")}</div>
      </fieldset>`;
  }

  function renderContenidoPrincipal() {
    const contenido = estado.editor.contenido_editable;
    const limites = estado.opciones.limites;
    return `
      <fieldset class="grupo-editor">
        <legend>2. Contenido de la convocatoria</legend>
        <div class="campos-editor-dos">
          <label class="campo"><span>Tipo de convocatoria</span><select required data-borrador-ruta="contenido_editable.tipo">${estado.opciones.tipos.map((item) => `<option value="${escaparHTML(item.clave)}" ${contenido.tipo === item.clave ? "selected" : ""}>${escaparHTML(item.etiqueta)}</option>`).join("")}</select></label>
          ${renderCategorias()}
          <label class="campo campo-ancho"><span>Título</span><input required maxlength="${limites.maximo_titulo}" data-borrador-ruta="contenido_editable.titulo" value="${escaparHTML(contenido.titulo)}"></label>
          <label class="campo campo-ancho"><span>Resumen</span><textarea required maxlength="${limites.maximo_resumen}" data-borrador-ruta="contenido_editable.resumen">${escaparHTML(contenido.resumen)}</textarea></label>
          <label class="campo campo-ancho"><span>Descripción completa</span><textarea class="texto-largo" required maxlength="${limites.maximo_descripcion}" data-borrador-ruta="contenido_editable.descripcion">${escaparHTML(contenido.descripcion)}</textarea></label>
        </div>
      </fieldset>`;
  }

  function renderPlazos() {
    const limites = estado.opciones.limites;
    const filas = estado.editor.contenido_editable.plazos.map((item, indice) => `
      <article class="elemento-editor" aria-labelledby="titulo-plazo-${indice}">
        <header><h4 id="titulo-plazo-${indice}">Plazo ${indice + 1}</h4><button type="button" class="boton-terciario peligro" data-borrador-accion="borradores-quitar" data-coleccion="plazos" data-indice="${indice}">Quitar plazo</button></header>
        <div class="campos-editor-dos">
          <label class="campo"><span>Referencia</span><input required maxlength="160" data-borrador-ruta="contenido_editable.plazos.${indice}.referencia" value="${escaparHTML(item.referencia)}"></label>
          <label class="campo"><span>Tipo de plazo</span><input required maxlength="80" pattern="[a-z0-9][a-z0-9._-]{0,79}" data-borrador-ruta="contenido_editable.plazos.${indice}.tipo" value="${escaparHTML(item.tipo)}"></label>
          <label class="campo campo-ancho"><span>Título</span><input required maxlength="${limites.maximo_titulo_plazo}" data-borrador-ruta="contenido_editable.plazos.${indice}.titulo" value="${escaparHTML(item.titulo)}"></label>
          <label class="campo campo-ancho"><span>Descripción</span><textarea required maxlength="${limites.maximo_descripcion_plazo}" data-borrador-ruta="contenido_editable.plazos.${indice}.descripcion">${escaparHTML(item.descripcion)}</textarea></label>
          <label class="campo"><span>Apertura UTC</span><input required inputmode="text" placeholder="2026-08-01T08:00:00Z" data-borrador-ruta="contenido_editable.plazos.${indice}.abre_en" value="${escaparHTML(item.abre_en)}"></label>
          <label class="campo"><span>Cierre UTC</span><input required inputmode="text" placeholder="2026-08-15T12:00:00Z" data-borrador-ruta="contenido_editable.plazos.${indice}.cierra_en" value="${escaparHTML(item.cierra_en)}"></label>
        </div>
      </article>`).join("");
    return `
      <fieldset class="grupo-editor">
        <legend>3. Plazos</legend>
        <p class="ayuda-campo">Instantes en UTC ISO 8601. El cierre debe ser posterior a la apertura.</p>
        <div class="coleccion-editor">${filas}</div>
        <button type="button" class="boton-secundario" data-borrador-accion="borradores-agregar" data-coleccion="plazos" ${estado.editor.contenido_editable.plazos.length >= limites.maximo_plazos ? "disabled" : ""}>Añadir plazo</button>
      </fieldset>`;
  }

  function renderRequisitos() {
    const limites = estado.opciones.limites;
    const filas = estado.editor.contenido_editable.requisitos.map((item, indice) => `
      <article class="elemento-editor" aria-labelledby="titulo-requisito-${indice}">
        <header><h4 id="titulo-requisito-${indice}">Requisito ${indice + 1}</h4><button type="button" class="boton-terciario peligro" data-borrador-accion="borradores-quitar" data-coleccion="requisitos" data-indice="${indice}">Quitar</button></header>
        <div class="campos-editor-dos">
          <label class="campo"><span>Referencia</span><input required maxlength="160" data-borrador-ruta="contenido_editable.requisitos.${indice}.referencia" value="${escaparHTML(item.referencia)}"></label>
          <label class="campo"><span>Orden</span><input required type="number" min="1" step="1" data-borrador-ruta="contenido_editable.requisitos.${indice}.orden" value="${escaparHTML(item.orden)}"></label>
          <label class="campo campo-ancho"><span>Título</span><input required maxlength="${limites.maximo_titulo_requisito}" data-borrador-ruta="contenido_editable.requisitos.${indice}.titulo" value="${escaparHTML(item.titulo)}"></label>
          <label class="campo campo-ancho"><span>Descripción</span><textarea required maxlength="${limites.maximo_descripcion_requisito}" data-borrador-ruta="contenido_editable.requisitos.${indice}.descripcion">${escaparHTML(item.descripcion)}</textarea></label>
          <label class="casilla-editor campo-ancho"><input type="checkbox" data-borrador-ruta="contenido_editable.requisitos.${indice}.obligatorio" ${item.obligatorio ? "checked" : ""}> <span>Requisito obligatorio</span></label>
        </div>
      </article>`).join("");
    return `
      <fieldset class="grupo-editor">
        <legend>4. Requisitos</legend>
        <div class="coleccion-editor">${filas || '<p class="vacio-coleccion">No se han añadido requisitos.</p>'}</div>
        <button type="button" class="boton-secundario" data-borrador-accion="borradores-agregar" data-coleccion="requisitos" ${estado.editor.contenido_editable.requisitos.length >= limites.maximo_requisitos ? "disabled" : ""}>Añadir requisito</button>
      </fieldset>`;
  }

  function renderAyuda() {
    const limites = estado.opciones.limites;
    const filas = estado.editor.contenido_editable.ayuda.map((item, indice) => `
      <article class="elemento-editor" aria-labelledby="titulo-ayuda-${indice}">
        <header><h4 id="titulo-ayuda-${indice}">Ayuda ${indice + 1}</h4><button type="button" class="boton-terciario peligro" data-borrador-accion="borradores-quitar" data-coleccion="ayuda" data-indice="${indice}">Quitar</button></header>
        <div class="campos-editor-dos">
          <label class="campo"><span>Referencia</span><input required maxlength="160" data-borrador-ruta="contenido_editable.ayuda.${indice}.referencia" value="${escaparHTML(item.referencia)}"></label>
          <label class="campo"><span>Categoría</span><input required maxlength="80" pattern="[a-z0-9][a-z0-9._-]{0,79}" data-borrador-ruta="contenido_editable.ayuda.${indice}.categoria" value="${escaparHTML(item.categoria)}"></label>
          <label class="campo"><span>Orden</span><input required type="number" min="1" step="1" data-borrador-ruta="contenido_editable.ayuda.${indice}.orden" value="${escaparHTML(item.orden)}"></label>
          <label class="campo campo-ancho"><span>Pregunta</span><input required maxlength="${limites.maximo_pregunta_ayuda}" data-borrador-ruta="contenido_editable.ayuda.${indice}.pregunta" value="${escaparHTML(item.pregunta)}"></label>
          <label class="campo campo-ancho"><span>Respuesta</span><textarea required maxlength="${limites.maximo_respuesta_ayuda}" data-borrador-ruta="contenido_editable.ayuda.${indice}.respuesta">${escaparHTML(item.respuesta)}</textarea></label>
        </div>
      </article>`).join("");
    return `
      <fieldset class="grupo-editor">
        <legend>5. Ayuda pública</legend>
        <div class="coleccion-editor">${filas || '<p class="vacio-coleccion">No se han añadido preguntas de ayuda.</p>'}</div>
        <button type="button" class="boton-secundario" data-borrador-accion="borradores-agregar" data-coleccion="ayuda" ${estado.editor.contenido_editable.ayuda.length >= limites.maximo_ayudas ? "disabled" : ""}>Añadir ayuda</button>
      </fieldset>`;
  }

  function renderRecibo() {
    if (!estado.recibo) return "";
    const recibo = estado.recibo;
    return `
      <section class="recibo-borrador" role="status" aria-labelledby="titulo-recibo-borrador">
        <div><p class="sobrelinea">Guardado confirmado</p><h3 id="titulo-recibo-borrador">Recibo administrativo del borrador</h3></div>
        <dl>
          <div><dt>Transacción</dt><dd><code>${escaparHTML(recibo.transaccion_ref)}</code></dd></div>
          <div><dt>Revisión</dt><dd>${escaparHTML(recibo.referencia_estado.revision)}</dd></div>
          <div><dt>Auditoría</dt><dd><code>${escaparHTML(recibo.auditoria_ref)}</code></dd></div>
          <div><dt>Evento</dt><dd><code>${escaparHTML(recibo.evento_outbox_ref)}</code></dd></div>
          <div><dt>Confirmado</dt><dd><time datetime="${escaparHTML(recibo.confirmada_en)}">${escaparHTML(instanteVisible(recibo.confirmada_en))}</time></dd></div>
        </dl>
      </section>`;
  }

  function renderResolucionConflicto() {
    const error = estado.errorEditor;
    if (!error?.tipoConflicto) return "";
    if (error.tipoConflicto === "idempotencia") {
      return `
        <section class="acciones-conflicto" aria-label="Resolver conflicto de idempotencia">
          <p>La clave anterior fue rechazada para esta operación. Revise el contenido y reintente con una clave nueva.</p>
          <button type="button" class="boton-primario" data-borrador-accion="borradores-rotar-idempotencia">Generar nueva clave y reintentar</button>
        </section>`;
    }
    if (!estado.conflictoRemoto) {
      return `
        <section class="acciones-conflicto" aria-label="Resolver conflicto de revisión">
          <p>Cargue la revisión vigente para compararla. La copia local permanecerá intacta.</p>
          <button type="button" class="boton-secundario" data-borrador-accion="borradores-cargar-vigente" ${estado.faseEditor === "comparando" ? "disabled" : ""}>${estado.faseEditor === "comparando" ? "Cargando estado vigente…" : "Cargar estado vigente para comparar"}</button>
        </section>`;
    }
    const remoto = estado.conflictoRemoto;
    const puedeReaplicar = remoto.capacidades.actualizar === true;
    return `
      <section class="comparacion-cas" aria-labelledby="titulo-comparacion-cas">
        <h3 id="titulo-comparacion-cas">Comparación antes de resolver</h3>
        <div class="tabla-contenedor"><table class="tabla-datos"><caption>Cambios locales frente al estado vigente</caption>
          <thead><tr><th scope="col">Dato</th><th scope="col">Copia local</th><th scope="col">Servidor</th></tr></thead>
          <tbody>
            <tr><th scope="row">Revisión base</th><td>${escaparHTML(estado.detalle.referencia_estado.revision)}</td><td>${escaparHTML(remoto.referencia_estado.revision)}</td></tr>
            <tr><th scope="row">Título</th><td>${escaparHTML(estado.editor.contenido_editable.titulo)}</td><td>${escaparHTML(remoto.contenido_editable.titulo)}</td></tr>
            <tr><th scope="row">Resumen</th><td>${escaparHTML(estado.editor.contenido_editable.resumen)}</td><td>${escaparHTML(remoto.contenido_editable.resumen)}</td></tr>
            <tr><th scope="row">Plazos</th><td>${estado.editor.contenido_editable.plazos.length}</td><td>${remoto.contenido_editable.plazos.length}</td></tr>
            <tr><th scope="row">Requisitos</th><td>${estado.editor.contenido_editable.requisitos.length}</td><td>${remoto.contenido_editable.requisitos.length}</td></tr>
          </tbody>
        </table></div>
        ${puedeReaplicar ? "" : '<p class="ayuda-campo" role="status">La revisión vigente es de solo lectura; puede descartar la copia local, pero no reaplicarla.</p>'}
        <label class="casilla-editor confirmacion-cas"><input type="checkbox" data-borrador-ruta="confirmar_reaplicacion" ${estado.confirmarReaplicacion ? "checked" : ""}> <span>He revisado la comparación y deseo reaplicar conscientemente los cambios locales sobre la revisión ${escaparHTML(remoto.referencia_estado.revision)}.</span></label>
        <div class="acciones-vista">
          <button type="button" class="boton-secundario" data-borrador-accion="borradores-descartar-locales">Descartar locales y usar servidor</button>
          <button type="button" class="boton-primario" data-borrador-accion="borradores-reaplicar-vigente" ${estado.confirmarReaplicacion && puedeReaplicar ? "" : "disabled"}>Reaplicar cambios locales</button>
        </div>
      </section>`;
  }

  function renderMetadatos() {
    const detalle = estado.detalle;
    if (!detalle) {
      const plantilla = plantillaSeleccionada();
      return `
        <section class="panel"><div class="cabecera-panel"><h3>Dependencias de alta</h3></div><div class="cuerpo-panel">
          <dl class="resumen-expediente">
            <div class="fila-resumen"><dt>Plantilla</dt><dd>${escaparHTML(plantilla?.nombre || "Sin seleccionar")}</dd></div>
            <div class="fila-resumen"><dt>Versión</dt><dd>${escaparHTML(plantilla?.version || "—")}</dd></div>
            <div class="fila-resumen"><dt>Huella</dt><dd><code>${escaparHTML(plantilla?.huella_sha256 || "—")}</code></dd></div>
          </dl>
        </div></section>`;
    }
    const documentos = detalle.configuracion_lectura.documentos;
    return `
      <section class="panel"><div class="cabecera-panel"><h3>Control de concurrencia</h3><span class="estado-chip info">Revisión ${escaparHTML(detalle.referencia_estado.revision)}</span></div><div class="cuerpo-panel">
        <dl class="resumen-expediente">
          <div class="fila-resumen"><dt>ETag fuerte</dt><dd><code>${escaparHTML(detalle.etag)}</code></dd></div>
          <div class="fila-resumen"><dt>Huella de estado</dt><dd><code>${escaparHTML(detalle.referencia_estado.huella_estado_sha256)}</code></dd></div>
          <div class="fila-resumen"><dt>Actualizar</dt><dd>${detalle.capacidades.actualizar ? "Capacidad concedida" : "Sin capacidad"}</dd></div>
        </dl>
      </div></section>
      <section class="panel"><div class="cabecera-panel"><h3>Configuración acreditada</h3><span class="estado-chip neutro">Solo lectura</span></div><div class="cuerpo-panel">
        <dl class="resumen-expediente">
          <div class="fila-resumen"><dt>Catálogos</dt><dd>${escaparHTML(detalle.configuracion_lectura.catalogos.referencia)} · v${escaparHTML(detalle.configuracion_lectura.catalogos.version)}</dd></div>
          <div class="fila-resumen"><dt>Calendario</dt><dd>${escaparHTML(detalle.configuracion_lectura.calendario.referencia)} · v${escaparHTML(detalle.configuracion_lectura.calendario.version)}</dd></div>
          <div class="fila-resumen"><dt>Baremación</dt><dd>${escaparHTML(detalle.configuracion_lectura.reglas_baremacion.referencia)} · v${escaparHTML(detalle.configuracion_lectura.reglas_baremacion.version)}</dd></div>
          <div class="fila-resumen"><dt>Documentos</dt><dd>${documentos.length} gobernados</dd></div>
        </dl>
      </div></section>`;
  }

  function renderEditor() {
    if (estado.faseEditor === "cargando") {
      return '<section class="panel editor-borrador"><div class="cuerpo-panel vacio-controlado" role="status">Cargando el borrador seleccionado…</div></section>';
    }
    if (!estado.editor) {
      return `
        <section class="panel editor-borrador"><div class="cuerpo-panel vacio-controlado">
          ${renderError(estado.errorEditor, "Detalle posterior no disponible")}
          ${renderRecibo()}
          <p><strong>Seleccione un borrador o cree uno nuevo.</strong></p>
          <p>El contenido solo se mantiene en memoria durante esta sesión de página.</p>
        </div></section>`;
    }
    const motivo = motivoSeleccionado();
    const capacidadGuardar = estado.modoEditor === "crear"
      ? estado.opciones.capacidades.crear
      : estado.detalle?.capacidades?.actualizar;
    const soloLectura = capacidadGuardar !== true;
    const estadoTexto = soloLectura
      ? "Solo lectura: sin capacidad"
      : (estado.guardando
      ? "Guardando"
      : (estado.sucio ? "Cambios locales sin guardar" : "Sin cambios locales pendientes"));
    const estadoClase = soloLectura ? "neutro" : (estado.guardando ? "info" : (estado.sucio ? "" : "exito"));
    return `
      <section class="editor-borrador" aria-labelledby="titulo-editor-borrador">
        <header class="cabecera-editor-borrador">
          <div><p class="sobrelinea">${estado.modoEditor === "crear" ? "Alta de borrador" : "Actualización con CAS"}</p><h3 id="titulo-editor-borrador">${estado.modoEditor === "crear" ? "Nuevo borrador de convocatoria" : escaparHTML(estado.editor.contenido_editable.titulo || "Borrador sin título")}</h3></div>
          <span class="estado-chip ${estadoClase}" data-estado-editor aria-live="polite">${estadoTexto}</span>
        </header>
        ${renderError(estado.errorEditor, "No se pudo guardar el borrador")}
        ${renderResolucionConflicto()}
        ${renderRecibo()}
        <div class="distribucion-editor-borrador">
          <form class="formulario-borrador" data-borrador-form="editor" ${estado.guardando || soloLectura ? `inert aria-busy="${estado.guardando}" aria-disabled="${soloLectura}"` : ""}>
            ${renderIdentidad()}
            ${renderContenidoPrincipal()}
            ${renderPlazos()}
            ${renderRequisitos()}
            ${renderAyuda()}
            <fieldset class="grupo-editor">
              <legend>6. Motivo y guardado</legend>
              <label class="campo"><span>Motivo gobernado</span><select required data-borrador-ruta="motivo_indice">${opcionesIndice(estado.opciones.motivos, "etiqueta", estado.editor.motivo_indice)}</select></label>
              <p class="ayuda-campo">Referencia: <code>${escaparHTML(motivo?.motivo_ref || "—")}</code> · huella <code>${escaparHTML(motivo?.huella_sha256 || "—")}</code></p>
              <div class="barra-guardado-borrador">
                <span>${soloLectura ? "La sesión puede consultar este borrador, pero no modificarlo." : (estado.sucio ? "La copia local se conservará ante cualquier conflicto." : "Edite algún campo para preparar un guardado.")}</span>
                <div>
                  <button type="button" class="boton-secundario" data-borrador-accion="borradores-cancelar-edicion" ${estado.guardando ? "disabled" : ""}>${estado.modoEditor === "crear" ? "Cancelar alta" : "Deshacer cambios"}</button>
                  <button type="submit" class="boton-primario" data-borrador-guardar data-capacidad="${capacidadGuardar === true}" ${capacidadGuardar && !estado.guardando && estado.sucio ? "" : "disabled"}>${estado.guardando ? "Guardando…" : (estado.modoEditor === "crear" ? "Crear borrador" : "Guardar con CAS")}</button>
                </div>
              </div>
            </fieldset>
          </form>
          <aside class="contexto-editor-borrador" aria-label="Contexto y evidencias del borrador">${renderMetadatos()}</aside>
        </div>
      </section>`;
  }

  function renderizar() {
    const cabecera = `
      <header class="encabezado-vista">
        <div><p class="sobrelinea">Gestión interna de Bolsa</p><h2>Borradores de convocatorias</h2><p>Edición durable con catálogos versionados, control CAS, idempotencia y recibo de auditoría.</p></div>
        <div class="acciones-vista"><button type="button" class="boton-secundario" data-vista="resumen">Volver al cuadro de mando</button></div>
      </header>
      <section class="nota-seguridad" aria-label="Tratamiento del borrador">La identidad procede exclusivamente del canal interno autenticado. No se usan cookies, tokens, almacenamiento local ni datos de presentación para esta bandeja.</section>`;
    const fuente = renderEstadoFuente();
    if (fuente) return `${cabecera}${fuente}`;
    return `${cabecera}<div class="espacio-borradores">${renderLista()}${renderEditor()}</div>`;
  }

  return Object.freeze({ renderizar });
}
