import { referenciaCopiableTraducida } from "./portal-justificante.js";
import { traducirReferencia } from "./portal-referencias-i18n.js";
import { LOCALIZACION_PORTAL, textoPortal, traducirPortal, ZONA_HORARIA_PORTAL } from "./portal-i18n.js?v=20260926-i18n-v1";

const FASE_INICIAL = "inicial";
const FASE_CARGANDO = "cargando";
const FASE_ERROR = "error";

function instanteVisible(instante) {
  if (!instante) return traducirPortal("txt_sin_fecha");
  const fecha = new Date(instante);
  if (!Number.isFinite(fecha.getTime())) return instante;
  return new Intl.DateTimeFormat(LOCALIZACION_PORTAL, {
    dateStyle: "short",
    timeStyle: "short",
    timeZone: ZONA_HORARIA_PORTAL,
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

  // Referencias, huellas y etiquetas de concurrencia son internas: no se
  // muestran; la referencia que sirve para el seguimiento se ofrece para copiar.
  function copiable(referencia, claveAria, variables) {
    return referenciaCopiableTraducida(referencia, escaparHTML, traducirReferencia, traducirReferencia(claveAria, variables));
  }
  function configuracion(nombre, valor) {
    return `<div class="fila-resumen"><dt>${escaparHTML(nombre)}</dt><dd>${escaparHTML(traducirReferencia("borrador_version", { version: valor.version }))} ${copiable(valor.referencia, "borrador_configuracion_copiar_aria", { nombre: nombre.toLocaleLowerCase(LOCALIZACION_PORTAL) })}</dd></div>`;
  }

  function renderError(error, contexto) {
    if (!error) return "";
    const conflicto = error.tipoConflicto === "idempotencia"
      ? traducirPortal("txt_conflicto_de_idempotencia_http_409")
      : (error.tipoConflicto === "cas" ? traducirPortal("txt_conflicto_de_revision_cas_http_412") : contexto);
    return `
      <section class="borrador-error" role="alert" aria-labelledby="titulo-error-borrador">
        <div>
          <p class="sobrelinea">${escaparHTML(conflicto)}</p>
          <h3 id="titulo-error-borrador">${escaparHTML(error.mensaje)}</h3>
          ${error.conservarCambiosLocales ? "<p>" + textoPortal("txt_los_cambios_introducidos_continuan_en_este_edito") + "</p>" : ""}
        </div>
        <dl class="metadatos-error">
          <div><dt>${textoPortal("txt_codigo")}</dt><dd><code>${escaparHTML(error.codigo)}</code></dd></div>
          ${error.correlacion ? `<div><dt>${textoPortal("txt_correlacion")}</dt><dd><code>${escaparHTML(error.correlacion)}</code></dd></div>` : ""}
        </dl>
      </section>`;
  }

  function renderEstadoFuente() {
    if (estado.faseLista === FASE_CARGANDO || estado.faseLista === FASE_INICIAL) {
      return `
        <section class="panel borradores-cargando" role="status" aria-live="polite">
          <div class="cuerpo-panel"><strong>${textoPortal("txt_comprobando_acceso_y_cargando_borradores")}</strong><p>${textoPortal("txt_la_identidad_procede_exclusivamente_del_canal_in")}</p></div>
        </section>`;
    }
    if (estado.faseLista === FASE_ERROR) {
      return `
        ${renderError(estado.errorLista, traducirPortal("txt_servicio_de_borradores_no_disponible"))}
        <section class="panel"><div class="cuerpo-panel vacio-controlado">
          <p><strong>${textoPortal("txt_la_bandeja_no_puede_operar_sin_el_backend_autent")}</strong></p>
          <button type="button" class="boton-primario" data-borrador-accion="borradores-recargar">${textoPortal("txt_reintentar_conexion")}</button>
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
        <strong>${textoPortal("txt_no_se_pudo_actualizar_la_bandeja")}</strong>
        <span>${escaparHTML(estado.errorLista.mensaje)} · <code>${escaparHTML(estado.errorLista.codigo)}</code></span>
      </div>` : "";
    const filas = elementos.map((item) => {
      const seleccionada = item.referencia_estado.referencia === estado.referenciaSeleccionada;
      return `
        <tr aria-selected="${seleccionada}">
          <td><strong>${escaparHTML(item.titulo)}</strong><small>${escaparHTML(item.identificador_publico)}</small></td>
          <td><span class="estado-chip ${seleccionada ? "info" : "neutro"}">${textoPortal("txt_rev_n", { revision: item.referencia_estado.revision })}</span></td>
          <td><time datetime="${escaparHTML(item.actualizada_en)}">${escaparHTML(instanteVisible(item.actualizada_en))}</time></td>
          <td><button type="button" class="boton-terciario" data-borrador-accion="borradores-abrir" data-id="${escaparHTML(item.referencia_estado.referencia)}" ${estado.guardando ? "disabled" : ""}>${seleccionada ? traducirPortal("txt_abierto") : traducirPortal("txt_abrir")}</button></td>
        </tr>`;
    }).join("") || '<tr><td colspan="4" class="vacio-controlado">' + textoPortal("txt_no_hay_borradores_para_este_filtro_y_ambito") + '</td></tr>';
    return `
      <section class="panel bandeja-borradores" aria-labelledby="titulo-bandeja-borradores">
        <div class="cabecera-panel">
          <div><h3 id="titulo-bandeja-borradores">${textoPortal("txt_bandeja_de_borradores")}</h3><p>${textoPortal("txt_n_registros_ambito_autorizado", { total })}</p></div>
          <button type="button" class="boton-primario" data-borrador-accion="borradores-nuevo" ${estado.opciones.capacidades.crear && !estado.guardando ? "" : "disabled"}>${textoPortal("txt_nuevo_borrador")}</button>
        </div>
        ${avisoActualizacion}
        <form class="filtros-borradores" data-borrador-form="filtros" ${estado.guardando ? 'inert aria-busy="true"' : ""}>
          <label class="campo"><span>${textoPortal("txt_buscar_por_titulo")}</span><input type="search" name="texto" maxlength="180" value="${escaparHTML(estado.filtro.texto)}" autocomplete="off"></label>
          <label class="campo"><span>${textoPortal("txt_categoria")}</span><select name="categoria"><option value="">${textoPortal("txt_todas")}</option>${opcionesFiltroCategorias()}</select></label>
          <button type="submit" class="boton-secundario">${textoPortal("txt_aplicar_filtros")}</button>
          <button type="button" class="boton-terciario" data-borrador-accion="borradores-recargar">${textoPortal("txt_actualizar")}</button>
        </form>
        <div class="tabla-contenedor tabla-borradores">
          <table class="tabla-datos">
            <caption>${textoPortal("txt_borradores_editables_devueltos_por_la_api_intern")}</caption>
            <thead><tr><th scope="col">${textoPortal("txt_borrador")}</th><th scope="col">${textoPortal("txt_estado")}</th><th scope="col">${textoPortal("txt_actualizado")}</th><th scope="col">${textoPortal("txt_accion")}</th></tr></thead>
            <tbody>${filas}</tbody>
          </table>
        </div>
        <nav class="paginacion-borradores" aria-label="${textoPortal("txt_paginacion_de_borradores")}">
          <button type="button" class="boton-terciario" data-borrador-accion="borradores-pagina-anterior" ${estado.pagina === 0 || estado.guardando ? "disabled" : ""}>${textoPortal("txt_anterior_2")}</button>
          <span>${textoPortal("txt_pagina_visibles", { pagina: estado.pagina + 1, visibles: elementos.length })}</span>
          <button type="button" class="boton-terciario" data-borrador-accion="borradores-pagina-siguiente" ${estado.lista?.paginacion?.siguiente_cursor && !estado.guardando ? "" : "disabled"}>${textoPortal("txt_siguiente_2")}</button>
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
          <legend>${textoPortal("txt_identidad_gobernada_de_solo_lectura")}</legend>
          <dl class="resumen-expediente resumen-editor">
            <div class="fila-resumen"><dt>${textoPortal("txt_codigo_publico")}</dt><dd>${escaparHTML(detalle.identificador_publico)}</dd></div>
            <div class="fila-resumen"><dt>${textoPortal("txt_version_publica")}</dt><dd>${escaparHTML(detalle.codigo_version_publica)}</dd></div>
            <div class="fila-resumen"><dt>${textoPortal("txt_expediente")}</dt><dd>${escaparHTML(detalle.expediente_ref)}</dd></div>
            <div class="fila-resumen"><dt>${textoPortal("txt_referencia")}</dt><dd>${copiable(detalle.referencia_estado.referencia, "borrador_referencia_copiar_aria")}</dd></div>
          </dl>
        </fieldset>`;
    }
    return `
      <fieldset class="grupo-editor grupo-identidad">
        <legend>${textoPortal("txt_1_identidad_y_plantilla")}</legend>
        <div class="campos-editor-dos">
          <label class="campo campo-ancho"><span>${textoPortal("txt_plantilla_gobernada")}</span><select required data-borrador-ruta="plantilla_indice">${opcionesIndice(estado.opciones.plantillas, "nombre", estado.editor.plantilla_indice)}</select></label>
          <label class="campo"><span>${textoPortal("txt_identificador_publico")}</span><input required maxlength="80" pattern="[a-z0-9][a-z0-9-]{2,79}" data-borrador-ruta="identificador_publico" value="${escaparHTML(estado.editor.identificador_publico)}" aria-describedby="ayuda-identificador-publico"></label>
          <label class="campo"><span>${textoPortal("txt_codigo_de_version_publica")}</span><input required maxlength="80" pattern="[a-z0-9][a-z0-9._-]{0,79}" data-borrador-ruta="codigo_version_publica" value="${escaparHTML(estado.editor.codigo_version_publica)}"></label>
          <label class="campo campo-ancho"><span>${textoPortal("txt_referencia_de_expediente")}</span><input required maxlength="512" data-borrador-ruta="expediente_ref" value="${escaparHTML(estado.editor.expediente_ref)}"></label>
          <p id="ayuda-identificador-publico" class="ayuda-campo campo-ancho">${textoPortal("txt_use_una_referencia_estable_en_minusculas_la_iden")}</p>
        </div>
      </fieldset>`;
  }

  function renderCategorias() {
    const seleccionadas = new Set(estado.editor.contenido_editable.categorias);
    return `
      <fieldset class="selector-categorias campo-ancho">
        <legend>${textoPortal("txt_categorias")}</legend>
        <div>${estado.opciones.categorias.map((item) => `
          <label><input type="checkbox" data-borrador-ruta="contenido_editable.categorias" value="${escaparHTML(item.clave)}" ${seleccionadas.has(item.clave) ? "checked" : ""}> <span>${escaparHTML(item.etiqueta)}</span></label>`).join("")}</div>
      </fieldset>`;
  }

  function renderContenidoPrincipal() {
    const contenido = estado.editor.contenido_editable;
    const limites = estado.opciones.limites;
    return `
      <fieldset class="grupo-editor">
        <legend>${textoPortal("txt_2_contenido_de_la_convocatoria")}</legend>
        <div class="campos-editor-dos">
          <label class="campo"><span>${textoPortal("txt_tipo_de_convocatoria")}</span><select required data-borrador-ruta="contenido_editable.tipo">${estado.opciones.tipos.map((item) => `<option value="${escaparHTML(item.clave)}" ${contenido.tipo === item.clave ? "selected" : ""}>${escaparHTML(item.etiqueta)}</option>`).join("")}</select></label>
          ${renderCategorias()}
          <label class="campo campo-ancho"><span>${textoPortal("txt_titulo")}</span><input required maxlength="${limites.maximo_titulo}" data-borrador-ruta="contenido_editable.titulo" value="${escaparHTML(contenido.titulo)}"></label>
          <label class="campo campo-ancho"><span>${textoPortal("txt_resumen")}</span><textarea required maxlength="${limites.maximo_resumen}" data-borrador-ruta="contenido_editable.resumen">${escaparHTML(contenido.resumen)}</textarea></label>
          <label class="campo campo-ancho"><span>${textoPortal("txt_descripcion_completa")}</span><textarea class="texto-largo" required maxlength="${limites.maximo_descripcion}" data-borrador-ruta="contenido_editable.descripcion">${escaparHTML(contenido.descripcion)}</textarea></label>
        </div>
      </fieldset>`;
  }

  function renderPlazos() {
    const limites = estado.opciones.limites;
    const filas = estado.editor.contenido_editable.plazos.map((item, indice) => `
      <article class="elemento-editor" aria-labelledby="titulo-plazo-${indice}">
        <header><h4 id="titulo-plazo-${indice}">${textoPortal("txt_plazo_n", { numero: indice + 1 })}</h4><button type="button" class="boton-terciario peligro" data-borrador-accion="borradores-quitar" data-coleccion="plazos" data-indice="${indice}">${textoPortal("txt_quitar_plazo")}</button></header>
        <div class="campos-editor-dos">
          <label class="campo"><span>${textoPortal("txt_referencia")}</span><input required maxlength="160" data-borrador-ruta="contenido_editable.plazos.${indice}.referencia" value="${escaparHTML(item.referencia)}"></label>
          <label class="campo"><span>${textoPortal("txt_tipo_de_plazo")}</span><input required maxlength="80" pattern="[a-z0-9][a-z0-9._-]{0,79}" data-borrador-ruta="contenido_editable.plazos.${indice}.tipo" value="${escaparHTML(item.tipo)}"></label>
          <label class="campo campo-ancho"><span>${textoPortal("txt_titulo")}</span><input required maxlength="${limites.maximo_titulo_plazo}" data-borrador-ruta="contenido_editable.plazos.${indice}.titulo" value="${escaparHTML(item.titulo)}"></label>
          <label class="campo campo-ancho"><span>${textoPortal("txt_descripcion")}</span><textarea required maxlength="${limites.maximo_descripcion_plazo}" data-borrador-ruta="contenido_editable.plazos.${indice}.descripcion">${escaparHTML(item.descripcion)}</textarea></label>
          <label class="campo"><span>${textoPortal("txt_apertura_utc")}</span><input required inputmode="text" placeholder="2026-08-01T08:00:00Z" data-borrador-ruta="contenido_editable.plazos.${indice}.abre_en" value="${escaparHTML(item.abre_en)}"></label>
          <label class="campo"><span>${textoPortal("txt_cierre_utc")}</span><input required inputmode="text" placeholder="2026-08-15T12:00:00Z" data-borrador-ruta="contenido_editable.plazos.${indice}.cierra_en" value="${escaparHTML(item.cierra_en)}"></label>
        </div>
      </article>`).join("");
    return `
      <fieldset class="grupo-editor">
        <legend>${textoPortal("txt_3_plazos")}</legend>
        <p class="ayuda-campo">${textoPortal("txt_instantes_en_utc_iso_8601_el_cierre_debe_ser_pos")}</p>
        <div class="coleccion-editor">${filas}</div>
        <button type="button" class="boton-secundario" data-borrador-accion="borradores-agregar" data-coleccion="plazos" ${estado.editor.contenido_editable.plazos.length >= limites.maximo_plazos ? "disabled" : ""}>${textoPortal("txt_anadir_plazo")}</button>
      </fieldset>`;
  }

  function renderRequisitos() {
    const limites = estado.opciones.limites;
    const filas = estado.editor.contenido_editable.requisitos.map((item, indice) => `
      <article class="elemento-editor" aria-labelledby="titulo-requisito-${indice}">
        <header><h4 id="titulo-requisito-${indice}">${textoPortal("txt_requisito_n", { numero: indice + 1 })}</h4><button type="button" class="boton-terciario peligro" data-borrador-accion="borradores-quitar" data-coleccion="requisitos" data-indice="${indice}">${textoPortal("txt_quitar")}</button></header>
        <div class="campos-editor-dos">
          <label class="campo"><span>${textoPortal("txt_referencia")}</span><input required maxlength="160" data-borrador-ruta="contenido_editable.requisitos.${indice}.referencia" value="${escaparHTML(item.referencia)}"></label>
          <label class="campo"><span>${textoPortal("txt_orden")}</span><input required type="number" min="1" step="1" data-borrador-ruta="contenido_editable.requisitos.${indice}.orden" value="${escaparHTML(item.orden)}"></label>
          <label class="campo campo-ancho"><span>${textoPortal("txt_titulo")}</span><input required maxlength="${limites.maximo_titulo_requisito}" data-borrador-ruta="contenido_editable.requisitos.${indice}.titulo" value="${escaparHTML(item.titulo)}"></label>
          <label class="campo campo-ancho"><span>${textoPortal("txt_descripcion")}</span><textarea required maxlength="${limites.maximo_descripcion_requisito}" data-borrador-ruta="contenido_editable.requisitos.${indice}.descripcion">${escaparHTML(item.descripcion)}</textarea></label>
          <label class="casilla-editor campo-ancho"><input type="checkbox" data-borrador-ruta="contenido_editable.requisitos.${indice}.obligatorio" ${item.obligatorio ? "checked" : ""}> <span>${textoPortal("txt_requisito_obligatorio")}</span></label>
        </div>
      </article>`).join("");
    return `
      <fieldset class="grupo-editor">
        <legend>${textoPortal("txt_4_requisitos")}</legend>
        <div class="coleccion-editor">${filas || '<p class="vacio-coleccion">' + textoPortal("txt_no_se_han_anadido_requisitos") + '</p>'}</div>
        <button type="button" class="boton-secundario" data-borrador-accion="borradores-agregar" data-coleccion="requisitos" ${estado.editor.contenido_editable.requisitos.length >= limites.maximo_requisitos ? "disabled" : ""}>${textoPortal("txt_anadir_requisito")}</button>
      </fieldset>`;
  }

  function renderAyuda() {
    const limites = estado.opciones.limites;
    const filas = estado.editor.contenido_editable.ayuda.map((item, indice) => `
      <article class="elemento-editor" aria-labelledby="titulo-ayuda-${indice}">
        <header><h4 id="titulo-ayuda-${indice}">${textoPortal("txt_ayuda_n", { numero: indice + 1 })}</h4><button type="button" class="boton-terciario peligro" data-borrador-accion="borradores-quitar" data-coleccion="ayuda" data-indice="${indice}">${textoPortal("txt_quitar")}</button></header>
        <div class="campos-editor-dos">
          <label class="campo"><span>${textoPortal("txt_referencia")}</span><input required maxlength="160" data-borrador-ruta="contenido_editable.ayuda.${indice}.referencia" value="${escaparHTML(item.referencia)}"></label>
          <label class="campo"><span>${textoPortal("txt_categoria")}</span><input required maxlength="80" pattern="[a-z0-9][a-z0-9._-]{0,79}" data-borrador-ruta="contenido_editable.ayuda.${indice}.categoria" value="${escaparHTML(item.categoria)}"></label>
          <label class="campo"><span>${textoPortal("txt_orden")}</span><input required type="number" min="1" step="1" data-borrador-ruta="contenido_editable.ayuda.${indice}.orden" value="${escaparHTML(item.orden)}"></label>
          <label class="campo campo-ancho"><span>${textoPortal("txt_pregunta")}</span><input required maxlength="${limites.maximo_pregunta_ayuda}" data-borrador-ruta="contenido_editable.ayuda.${indice}.pregunta" value="${escaparHTML(item.pregunta)}"></label>
          <label class="campo campo-ancho"><span>${textoPortal("txt_respuesta")}</span><textarea required maxlength="${limites.maximo_respuesta_ayuda}" data-borrador-ruta="contenido_editable.ayuda.${indice}.respuesta">${escaparHTML(item.respuesta)}</textarea></label>
        </div>
      </article>`).join("");
    return `
      <fieldset class="grupo-editor">
        <legend>${textoPortal("txt_5_ayuda_publica")}</legend>
        <div class="coleccion-editor">${filas || '<p class="vacio-coleccion">' + textoPortal("txt_no_se_han_anadido_preguntas_de_ayuda") + '</p>'}</div>
        <button type="button" class="boton-secundario" data-borrador-accion="borradores-agregar" data-coleccion="ayuda" ${estado.editor.contenido_editable.ayuda.length >= limites.maximo_ayudas ? "disabled" : ""}>${textoPortal("txt_anadir_ayuda")}</button>
      </fieldset>`;
  }

  function renderRecibo() {
    if (!estado.recibo) return "";
    const recibo = estado.recibo;
    return `
      <section class="recibo-borrador" role="status" aria-labelledby="titulo-recibo-borrador">
        <div><p class="sobrelinea">${textoPortal("txt_guardado_confirmado")}</p><h3 id="titulo-recibo-borrador">${textoPortal("txt_recibo_administrativo_del_borrador")}</h3></div>
        <dl>
          <div><dt>${escaparHTML(traducirReferencia("borrador_justificante"))}</dt><dd>${copiable(recibo.transaccion_ref, "borrador_justificante_copiar_aria")}</dd></div>
          <div><dt>${textoPortal("txt_revision")}</dt><dd>${escaparHTML(recibo.referencia_estado.revision)}</dd></div>
          <div><dt>${textoPortal("txt_confirmado")}</dt><dd><time datetime="${escaparHTML(recibo.confirmada_en)}">${escaparHTML(instanteVisible(recibo.confirmada_en))}</time></dd></div>
        </dl>
      </section>`;
  }

  function renderResolucionConflicto() {
    const error = estado.errorEditor;
    if (!error?.tipoConflicto) return "";
    if (error.tipoConflicto === "idempotencia") {
      return `
        <section class="acciones-conflicto" aria-label="${textoPortal("txt_resolver_conflicto_de_idempotencia")}">
          <p>${textoPortal("txt_la_clave_anterior_fue_rechazada_para_esta_operac")}</p>
          <button type="button" class="boton-primario" data-borrador-accion="borradores-rotar-idempotencia">${textoPortal("txt_generar_nueva_clave_y_reintentar")}</button>
        </section>`;
    }
    if (!estado.conflictoRemoto) {
      return `
        <section class="acciones-conflicto" aria-label="${textoPortal("txt_resolver_conflicto_de_revision")}">
          <p>${textoPortal("txt_cargue_la_revision_vigente_para_compararla_la_co")}</p>
          <button type="button" class="boton-secundario" data-borrador-accion="borradores-cargar-vigente" ${estado.faseEditor === "comparando" ? "disabled" : ""}>${estado.faseEditor === "comparando" ? traducirPortal("txt_cargando_estado_vigente") : traducirPortal("txt_cargar_estado_vigente_para_comparar")}</button>
        </section>`;
    }
    const remoto = estado.conflictoRemoto;
    const puedeReaplicar = remoto.capacidades.actualizar === true;
    return `
      <section class="comparacion-cas" aria-labelledby="titulo-comparacion-cas">
        <h3 id="titulo-comparacion-cas">${textoPortal("txt_comparacion_antes_de_resolver")}</h3>
        <div class="tabla-contenedor"><table class="tabla-datos"><caption>${textoPortal("txt_cambios_locales_frente_al_estado_vigente")}</caption>
          <thead><tr><th scope="col">${textoPortal("txt_dato")}</th><th scope="col">${textoPortal("txt_copia_local")}</th><th scope="col">${textoPortal("txt_servidor")}</th></tr></thead>
          <tbody>
            <tr><th scope="row">${textoPortal("txt_revision_base")}</th><td>${escaparHTML(estado.detalle.referencia_estado.revision)}</td><td>${escaparHTML(remoto.referencia_estado.revision)}</td></tr>
            <tr><th scope="row">${textoPortal("txt_titulo")}</th><td>${escaparHTML(estado.editor.contenido_editable.titulo)}</td><td>${escaparHTML(remoto.contenido_editable.titulo)}</td></tr>
            <tr><th scope="row">${textoPortal("txt_resumen")}</th><td>${escaparHTML(estado.editor.contenido_editable.resumen)}</td><td>${escaparHTML(remoto.contenido_editable.resumen)}</td></tr>
            <tr><th scope="row">${textoPortal("txt_plazos")}</th><td>${estado.editor.contenido_editable.plazos.length}</td><td>${remoto.contenido_editable.plazos.length}</td></tr>
            <tr><th scope="row">${textoPortal("txt_requisitos")}</th><td>${estado.editor.contenido_editable.requisitos.length}</td><td>${remoto.contenido_editable.requisitos.length}</td></tr>
          </tbody>
        </table></div>
        ${puedeReaplicar ? "" : '<p class="ayuda-campo" role="status">' + textoPortal("txt_la_revision_vigente_es_de_solo_lectura_puede_des") + '</p>'}
        <label class="casilla-editor confirmacion-cas"><input type="checkbox" data-borrador-ruta="confirmar_reaplicacion" ${estado.confirmarReaplicacion ? "checked" : ""}> <span>${textoPortal("txt_confirmar_reaplicacion", { revision: remoto.referencia_estado.revision })}</span></label>
        <div class="acciones-vista">
          <button type="button" class="boton-secundario" data-borrador-accion="borradores-descartar-locales">${textoPortal("txt_descartar_locales_y_usar_servidor")}</button>
          <button type="button" class="boton-primario" data-borrador-accion="borradores-reaplicar-vigente" ${estado.confirmarReaplicacion && puedeReaplicar ? "" : "disabled"}>${textoPortal("txt_reaplicar_cambios_locales")}</button>
        </div>
      </section>`;
  }

  function renderMetadatos() {
    const detalle = estado.detalle;
    if (!detalle) {
      const plantilla = plantillaSeleccionada();
      return `
        <section class="panel"><div class="cabecera-panel"><h3>${textoPortal("txt_dependencias_de_alta")}</h3></div><div class="cuerpo-panel">
          <dl class="resumen-expediente">
            <div class="fila-resumen"><dt>${textoPortal("txt_plantilla")}</dt><dd>${escaparHTML(plantilla?.nombre || traducirPortal("txt_sin_seleccionar"))}</dd></div>
            <div class="fila-resumen"><dt>${textoPortal("txt_version")}</dt><dd>${escaparHTML(plantilla?.version || "—")}</dd></div>
          </dl>
        </div></section>`;
    }
    const documentos = detalle.configuracion_lectura.documentos;
    return `
      <section class="panel"><div class="cabecera-panel"><h3>${textoPortal("txt_control_de_concurrencia")}</h3><span class="estado-chip info">${textoPortal("txt_revision_n", { revision: detalle.referencia_estado.revision })}</span></div><div class="cuerpo-panel">
        <dl class="resumen-expediente">
          <div class="fila-resumen"><dt>${textoPortal("txt_actualizar")}</dt><dd>${detalle.capacidades.actualizar ? traducirPortal("txt_capacidad_concedida") : traducirPortal("txt_sin_capacidad")}</dd></div>
        </dl>
      </div></section>
      <section class="panel"><div class="cabecera-panel"><h3>${textoPortal("txt_configuracion_acreditada")}</h3><span class="estado-chip neutro">${textoPortal("txt_solo_lectura")}</span></div><div class="cuerpo-panel">
        <dl class="resumen-expediente">
          ${configuracion(traducirPortal("txt_catalogos"), detalle.configuracion_lectura.catalogos)}
          ${configuracion(traducirPortal("txt_calendario"), detalle.configuracion_lectura.calendario)}
          ${configuracion(traducirPortal("txt_baremacion"), detalle.configuracion_lectura.reglas_baremacion)}
          <div class="fila-resumen"><dt>${textoPortal("txt_documentos")}</dt><dd>${textoPortal("txt_n_gobernados", { numero: documentos.length })}</dd></div>
        </dl>
      </div></section>`;
  }

  function renderEditor() {
    if (estado.faseEditor === "cargando") {
      return '<section class="panel editor-borrador"><div class="cuerpo-panel vacio-controlado" role="status">' + textoPortal("txt_cargando_el_borrador_seleccionado") + '</div></section>';
    }
    if (!estado.editor) {
      return `
        <section class="panel editor-borrador"><div class="cuerpo-panel vacio-controlado">
          ${renderError(estado.errorEditor, traducirPortal("txt_detalle_posterior_no_disponible"))}
          ${renderRecibo()}
          <p><strong>${textoPortal("txt_seleccione_un_borrador_o_cree_uno_nuevo")}</strong></p>
          <p>${textoPortal("txt_el_contenido_solo_se_mantiene_en_memoria_durante")}</p>
        </div></section>`;
    }
    const capacidadGuardar = estado.modoEditor === "crear"
      ? estado.opciones.capacidades.crear
      : estado.detalle?.capacidades?.actualizar;
    const soloLectura = capacidadGuardar !== true;
    const estadoTexto = soloLectura
      ? traducirPortal("txt_solo_lectura_sin_capacidad")
      : (estado.guardando
      ? traducirPortal("txt_guardando_2")
      : (estado.sucio ? traducirPortal("txt_cambios_locales_sin_guardar") : traducirPortal("txt_sin_cambios_locales_pendientes")));
    const estadoClase = soloLectura ? "neutro" : (estado.guardando ? "info" : (estado.sucio ? "" : "exito"));
    return `
      <section class="editor-borrador" aria-labelledby="titulo-editor-borrador">
        <header class="cabecera-editor-borrador">
          <div><p class="sobrelinea">${estado.modoEditor === "crear" ? traducirPortal("txt_alta_de_borrador") : traducirPortal("txt_actualizacion_con_cas")}</p><h3 id="titulo-editor-borrador">${estado.modoEditor === "crear" ? traducirPortal("txt_nuevo_borrador_de_convocatoria") : escaparHTML(estado.editor.contenido_editable.titulo || traducirPortal("txt_borrador_sin_titulo"))}</h3></div>
          <span class="estado-chip ${estadoClase}" data-estado-editor aria-live="polite">${estadoTexto}</span>
        </header>
        ${renderError(estado.errorEditor, traducirPortal("txt_no_se_pudo_guardar_el_borrador"))}
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
              <legend>${textoPortal("txt_6_motivo_y_guardado")}</legend>
              <label class="campo"><span>${textoPortal("txt_motivo_gobernado")}</span><select required data-borrador-ruta="motivo_indice">${opcionesIndice(estado.opciones.motivos, "etiqueta", estado.editor.motivo_indice)}</select></label>
              <div class="barra-guardado-borrador">
                <span>${soloLectura ? traducirPortal("txt_la_sesion_puede_consultar_este_borrador_pero_no") : (estado.sucio ? traducirPortal("txt_la_copia_local_se_conservara_ante_cualquier_conf") : traducirPortal("txt_edite_algun_campo_para_preparar_un_guardado"))}</span>
                <div>
                  <button type="button" class="boton-secundario" data-borrador-accion="borradores-cancelar-edicion" ${estado.guardando ? "disabled" : ""}>${estado.modoEditor === "crear" ? traducirPortal("txt_cancelar_alta") : traducirPortal("txt_deshacer_cambios")}</button>
                  <button type="submit" class="boton-primario" data-borrador-guardar data-capacidad="${capacidadGuardar === true}" ${capacidadGuardar && !estado.guardando && estado.sucio ? "" : "disabled"}>${estado.guardando ? traducirPortal("txt_guardando") : (estado.modoEditor === "crear" ? traducirPortal("txt_crear_borrador") : traducirPortal("txt_guardar_con_cas"))}</button>
                </div>
              </div>
            </fieldset>
          </form>
          <aside class="contexto-editor-borrador" aria-label="${textoPortal("txt_contexto_y_evidencias_del_borrador")}">${renderMetadatos()}</aside>
        </div>
      </section>`;
  }

  function renderizar() {
    const cabecera = `
      <header class="encabezado-vista">
        <div><p class="sobrelinea">${textoPortal("txt_gestion_interna_de_bolsa")}</p><h2>${textoPortal("txt_borradores_de_convocatorias")}</h2><p>${textoPortal("txt_edicion_durable_con_catalogos_versionados_contro")}</p></div>
        <div class="acciones-vista"><button type="button" class="boton-secundario" data-vista="resumen">${textoPortal("txt_volver_al_cuadro_de_mando")}</button></div>
      </header>
      <section class="nota-seguridad" aria-label="${textoPortal("txt_tratamiento_del_borrador")}">${textoPortal("txt_la_identidad_procede_exclusivamente_del_canal_in_2")}</section>`;
    const fuente = renderEstadoFuente();
    if (fuente) return `${cabecera}${fuente}`;
    return `${cabecera}<div class="espacio-borradores">${renderLista()}${renderEditor()}</div>`;
  }

  return Object.freeze({ renderizar });
}
