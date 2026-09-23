/** Presentación S3–S7. No adopta decisiones sin conector autorizado. */
import { traducirBaremacion as t } from "./portal-i18n-baremacion.js";

export function crearVistasBaremacion(u) {
  const { escaparHTML: e, numero, fecha, chip, tabla, kpi, encabezadoVista,
    avisoPresentacion, campo } = u;
  const lista = (datos, nombre) => Array.isArray(datos?.[nombre]) ? datos[nombre] : [];
  const valor = (dato, respaldo) => String(dato ?? "").trim() || respaldo;
  const filtro = (estado, nombre, defecto = "") => String(estado?.filtros?.meritos?.[nombre] ?? defecto);
  const contiene = (dato, busca) => String(dato ?? "").toLocaleLowerCase("es").includes(String(busca).trim().toLocaleLowerCase("es"));
  const lectura = (estado, vista) => estado?.vistasBaremacion?.[vista] || { carga: "listo" };
  const opcion = (dato, seleccionado) => `<option value="${e(dato)}"${dato === seleccionado ? " selected" : ""}>${e(dato)}</option>`;

  function ayuda(titulo, contenido) {
    return `<details class="baremacion-ayuda"><summary aria-label="${e(t("ayuda"))}: ${e(t(titulo))}" title="${e(t("ayuda"))}">?</summary><p>${e(t(contenido))}</p></details>`;
  }
  function recorrido() {
    return `<ol class="baremacion-recorrido" aria-label="${e(t("baremacion_titulo"))}">${["fase_dato", "fase_acceso", "fase_merito", "fase_lista"]
      .map((clave, indice) => `<li><span aria-hidden="true">${indice + 1}</span>${e(t(clave))}</li>`).join("")}</ol>`;
  }
  function accionBloqueada(etiqueta, operacion, objetivo, clase = "boton-secundario") {
    return `<button type="button" class="${e(clase)}" data-comando="${e(operacion)}" data-objetivo="${e(objetivo)}" disabled aria-disabled="true" title="${e(t("accion_bloqueada"))}">${e(t(etiqueta))}</button>`;
  }
  function estadoNoDisponible(titulo, estado, vacio) {
    const estados = {
      cargando: ["estado_cargando", "estado_cargando_detalle", "estado_bloqueo_ayuda", "status"],
      denegado: ["estado_denegado", "estado_denegado_detalle", "estado_bloqueo_ayuda", "alert"],
      error: ["estado_error", "estado_error_detalle", "estado_error_ayuda", "alert"],
      no_configurado: ["estado_no_configurado", "estado_no_configurado_detalle", "estado_bloqueo_ayuda", "status"],
    };
    const claves = estados[estado?.carga] || (vacio ? ["estado_vacio", "estado_vacio_detalle", "estado_vacio_ayuda", "status"] : null);
    if (!claves) return "";
    const [chipClave, detalle, ayudaClave, rol] = claves;
    return `<section class="panel baremacion-estado"><div class="cabecera-panel"><h3>${e(t(titulo))}</h3><span class="estado-chip ${rol === "alert" ? "peligro" : "neutro"}">${e(t(chipClave))}</span></div><div class="cuerpo-panel vacio-controlado" role="${rol}"${estado?.carga === "cargando" ? ' aria-busy="true"' : ""}><p><strong>${e(t(detalle))}</strong></p><p>${e(t(ayudaClave))}</p></div></section>`;
  }
  function cabecera(titulo, contenido, subtitulo) {
    return `<div class="cabecera-panel"><div><h3>${e(t(titulo))}</h3><p>${e(t(subtitulo))}</p></div>${ayuda(titulo, contenido)}</div>`;
  }
  function estadoDato(item) {
    const clave = {
      declarado: "meritos_dato_declarado", pendiente: "meritos_dato_pendiente",
      acreditado: "meritos_dato_acreditado", rechazado: "meritos_dato_rechazado",
    }[item.estado_dato] || (item.declarado ? "meritos_dato_declarado" : "meritos_dato_pendiente");
    if ((clave === "meritos_dato_acreditado" || clave === "meritos_dato_rechazado")
      && (!item.fuente || !item.evidencia || !item.vigencia)) return t("meritos_dato_pendiente");
    return t(clave);
  }
  function acceso(item) {
    const evaluacion = item.evaluacion_acceso;
    if (!evaluacion || typeof evaluacion !== "object"
      || !evaluacion.convocatoria_ref || !evaluacion.requisito_ref
      || !evaluacion.fuente || !evaluacion.version_bases || !evaluacion.motivo
      || (evaluacion.resultado === "cumplimiento_previsto"
        && (!evaluacion.hito || evaluacion.tipo_proceso !== "ope"
          || evaluacion.permite_inscripcion_pendiente !== true))) {
      return `<span class="baremacion-pendiente">${e(t("meritos_acceso_pendiente"))}</span>`;
    }
    const clave = { cumple: "meritos_acceso_cumple", no_cumple: "meritos_acceso_no_cumple",
      cumplimiento_previsto: "meritos_acceso_condicional" }[evaluacion.resultado] || "meritos_acceso_pendiente";
    return `<strong>${e(t(clave))}</strong><small>${e(evaluacion.motivo)} · ${e(evaluacion.fuente)} · ${e(evaluacion.version_bases)}</small>`;
  }
  function valoracion(item, demo) {
    if (demo && item.puntos != null) return e(t("meritos_valoracion_demo", { puntos: item.puntos }));
    const v = item.valoracion;
    if (!v || !v.solicitud_ref || !v.convocatoria_ref || !v.version_reglas || !v.fuente_calculo || v.puntos == null) {
      return `<span class="baremacion-pendiente">${e(t("meritos_valoracion_pendiente"))}</span>`;
    }
    return `<strong>${e(v.puntos)}</strong><small>${e(v.convocatoria_ref)} · ${e(v.version_reglas)}</small>`;
  }
  function revision(item, demo) {
    if (demo) return e(t("meritos_revision_demo", { estado: valor(item.estado, t("meritos_revision_pendiente")) }));
    const r = item.revision;
    if (!r || !r.estado || !r.fuente || !r.motivo || !r.version_criterio) {
      return `<span class="baremacion-pendiente">${e(t("meritos_revision_pendiente"))}</span>`;
    }
    return `<strong>${e(r.estado)}</strong><small>${e(r.motivo)} · ${e(r.fuente)} · ${e(r.version_criterio)}</small>`;
  }

  function renderizarMeritos(datos, estado = {}) {
    const fuente = lectura(estado, "meritos");
    const todos = lista(datos, "meritos_revision");
    const bloqueado = estadoNoDisponible("meritos_titulo", fuente, todos.length === 0);
    if (bloqueado) return bloqueado;
    const referencia = filtro(estado, "referencia");
    const tipo = filtro(estado, "tipo", t("meritos_todos"));
    const estadoSeleccionado = filtro(estado, "estado", t("meritos_todos"));
    const meritos = todos.filter((item) => (!referencia || [item.id, item.persona_ref, item.evidencia].some((x) => contiene(x, referencia)))
      && (tipo === t("meritos_todos") || item.tipo === tipo)
      && (estadoSeleccionado === t("meritos_todos") || item.estado === estadoSeleccionado));
    const demo = fuente.origen !== "api_autorizada";
    const filas = meritos.map((item) => [
      `<strong>${e(item.id)}</strong><small>${e(item.tipo)}</small>`,
      `<strong>${e(valor(item.declarado, t("meritos_sin_dato")))}</strong><small>${e(estadoDato(item))}</small>`,
      `${e(valor(item.fuente, t("meritos_sin_fuente")))}<small>${e(valor(item.evidencia, t("meritos_sin_evidencia")))} · ${e(valor(item.vigencia, t("meritos_sin_vigencia")))}</small>`,
      acceso(item), valoracion(item, demo),
      revision(item, demo),
    ]);
    const objetivo = todos[0]?.id || "DEMO-MER-001";
    return `<div class="baremacion-vista">
      ${encabezadoVista(t("meritos_seccion"), t("meritos_titulo"), t("meritos_descripcion"), accionBloqueada("meritos_informe", "exportar-informe", "DEMO-REV-MERITOS"))}
      ${avisoPresentacion(t("meritos_decision_bloqueada"))}${recorrido()}
      <div class="rejilla-kpi baremacion-kpi">${kpi("PEN", numero(todos.filter((x) => x.estado === "Pendiente").length), t("meritos_pendientes"))}${kpi("DEC", numero(todos.filter((x) => !!x.declarado).length), t("meritos_declarados"))}${kpi("REV", numero(todos.filter((x) => /Aceptado|Rechazado/.test(x.estado || "")).length), t("meritos_revisados"))}</div>
      <section class="panel">${cabecera("meritos_panel", "ayuda_meritos", "meritos_panel_ayuda")}
        <form class="barra-filtros" aria-label="${e(t("meritos_panel"))}" data-filtro="meritos">
          ${campo(t("meritos_filtro_ref"), `<input type="search" name="referencia" value="${e(referencia)}" placeholder="${e(t("meritos_referencia_ejemplo"))}">`)}
          ${campo(t("meritos_filtro_tipo"), `<select name="tipo">${["meritos_todos", "meritos_tipo_experiencia", "meritos_tipo_formacion", "meritos_tipo_titulacion"].map((x) => opcion(t(x), tipo)).join("")}</select>`)}
          ${campo(t("meritos_filtro_estado"), `<select name="estado">${["meritos_todos", "meritos_estado_pendiente", "meritos_estado_aceptado", "meritos_estado_rechazado"].map((x) => opcion(t(x), estadoSeleccionado)).join("")}</select>`)}
          <button type="submit" class="boton-secundario">${e(t("meritos_aplicar"))}</button>
        </form>
        <p class="resultado-filtro" role="status" data-total-filtro="meritos" data-total="${meritos.length}">${e(t("meritos_encontrados", { numero: numero(meritos.length) }))}</p>
        ${tabla({ titulo: t("meritos_tabla"), cabeceras: ["meritos_id", "meritos_dato", "meritos_fuente", "meritos_acceso", "meritos_puntos", "meritos_revision"].map(t), filas,
          clavesColumnas: ["referencia", "dato", "fuente", "acceso", "puntos", "estado"], prioridadColumnas: "estado" })}
      </section>
      <section class="panel panel-separado">${cabecera("meritos_decision_panel", "ayuda_meritos", "meritos_decision_ayuda")}
        <details class="baremacion-decision"><summary>${e(t("meritos_decision_ver"))}</summary>
        <form class="cuerpo-panel formulario-gobernado" aria-label="${e(t("meritos_formulario"))}" data-comando="aceptar-merito">
          <fieldset disabled aria-disabled="true"><legend>${e(t("meritos_fundamento"))}</legend><div class="rejilla-formulario">
            ${campo(t("meritos_criterio"), `<select name="criterio"><option>${e(t("meritos_sin_criterio"))}</option></select>`)}
            ${campo(t("meritos_motivo"), `<select name="motivo_tipificado"><option>${e(t("meritos_sin_motivo"))}</option></select>`)}
            ${campo(t("meritos_observacion"), '<textarea name="observacion"></textarea>')}
          </div></fieldset><p class="nota-pendiente">${e(t("meritos_decision_bloqueada"))}</p>
          <div class="acciones-formulario">${accionBloqueada("meritos_aceptar", "aceptar-merito", objetivo, "boton-primario")}${accionBloqueada("meritos_rechazar", "rechazar-merito", objetivo)}${accionBloqueada("meritos_revocar", "revocar-merito", objetivo)}${accionBloqueada("meritos_rehabilitar", "rehabilitar-merito", objetivo)}</div>
        </form>
        </details>
      </section><p class="nota-seguridad">${e(t("meritos_rectificacion"))}</p>
    </div>`;
  }

  function renderizarBaremacion(datos, estado = {}) {
    const fuente = lectura(estado, "baremacion");
    const criterios = lista(datos, "criterios_baremo");
    const entradas = lista(datos, "ranking");
    const bloqueado = estadoNoDisponible("baremacion_titulo", fuente, criterios.length === 0 && entradas.length === 0);
    if (bloqueado) return bloqueado;
    const version = valor(criterios[0]?.version, t("baremacion_sin_version"));
    const referencia = valor(datos?.convocatoria_ref, "DEMO-BOL-014");
    const filas = entradas.map((item) => [e(item.posicion), e(item.persona_ref), e(item.experiencia), e(item.formacion), e(item.otros), `<strong>${e(item.total)}</strong>`, e(item.desempate), chip(item.estado)]);
    return `<div class="baremacion-vista">
      ${encabezadoVista(t("baremacion_seccion"), t("baremacion_titulo"), t("baremacion_descripcion"), accionBloqueada("baremacion_calcular", "calcular-baremo", referencia, "boton-primario"))}
      ${avisoPresentacion(t("baremacion_aviso"))}${recorrido()}
      <div class="rejilla-kpi baremacion-kpi">${kpi("VER", version, t("baremacion_version"))}${kpi("CRI", numero(criterios.length), t("baremacion_criterios"))}${kpi("BLQ", numero(new Set(criterios.map((x) => x.bloque)).size), t("baremacion_bloques"))}${kpi("ASP", numero(entradas.length), t("baremacion_personas"))}</div>
      <section class="panel">${cabecera("baremacion_contexto", "ayuda_ranking", "baremacion_contexto_ayuda")}
        <div class="cuerpo-panel"><dl class="resumen-expediente">
          <div class="fila-resumen"><dt>${e(t("baremacion_convocatoria"))}</dt><dd>${e(referencia)} · ${e(t("baremacion_demo_ref"))}</dd></div>
          <div class="fila-resumen"><dt>${e(t("baremacion_entrada"))}</dt><dd>${e(t("baremacion_demo_entrada"))}</dd></div>
          <div class="fila-resumen"><dt>${e(t("baremacion_reglas"))}</dt><dd>${e(version)}</dd></div>
          <div class="fila-resumen"><dt>${e(t("baremacion_salida"))}</dt><dd>${e(t("baremacion_demo_salida"))}</dd></div>
        </dl></div>
      </section>
      <section class="panel panel-separado">${cabecera("baremacion_ranking_panel", "ayuda_ranking", "baremacion_ranking_ayuda")}
        <div class="baremacion-acciones"><span class="estado-chip info">${e(t("fuente_demo"))}</span>${accionBloqueada("baremacion_publicar", "publicar-lista-provisional", referencia, "boton-primario")}${accionBloqueada("baremacion_exportar", "exportar-informe", "DEMO-RAN-BOL-014")}</div>
        ${tabla({ titulo: t("baremacion_ranking_tabla"), cabeceras: ["baremacion_posicion", "baremacion_persona", "baremacion_experiencia", "baremacion_formacion", "baremacion_otros", "baremacion_total", "baremacion_desempate", "baremacion_estado"].map(t), filas })}
      </section><p class="nota-pendiente">${e(t("baremacion_accion_bloqueada"))}</p>
    </div>`;
  }

  function renderizarAlegaciones(datos, estado = {}) {
    const fuente = lectura(estado, "alegaciones");
    const alegaciones = lista(datos, "alegaciones");
    const bloqueado = estadoNoDisponible("alegaciones_titulo", fuente, alegaciones.length === 0);
    if (bloqueado) return bloqueado;
    const filas = alegaciones.map((item) => [
      `<strong>${e(item.id)}</strong>`, e(item.persona_ref), e(item.objeto), e(fecha(item.registrada)),
      e(t("alegaciones_sin_plazo")), e(item.evidencia), chip(item.estado),
      `<div class="acciones-fila">${accionBloqueada("alegaciones_estimar", "resolver-alegacion", item.id, "boton-terciario")}${accionBloqueada("alegaciones_desestimar", "desestimar-alegacion", item.id, "boton-terciario")}</div>`,
    ]);
    return `<div class="baremacion-vista">
      ${encabezadoVista(t("alegaciones_seccion"), t("alegaciones_titulo"), t("alegaciones_descripcion"), accionBloqueada("alegaciones_preparar", "generar-documento", "DEMO-ALE-LOTE-01", "boton-primario"))}
      ${avisoPresentacion(t("alegaciones_aviso"))}${recorrido()}
      <section class="panel">${cabecera("alegaciones_panel", "ayuda_alegaciones", "alegaciones_panel_ayuda")}
        ${tabla({ titulo: t("alegaciones_tabla"), cabeceras: ["alegaciones_id", "alegaciones_persona", "alegaciones_objeto", "alegaciones_registro", "alegaciones_plazo", "alegaciones_evidencia", "alegaciones_estado", "alegaciones_resolucion"].map(t),
          clavesColumnas: ["referencia", "persona", "objeto", "registro", "plazo", "evidencia", "estado", "acciones"], prioridadColumnas: "estado-acciones", filas })}
      </section><p class="nota-pendiente">${e(t("alegaciones_bloqueada"))} ${e(t("alegaciones_rectificacion"))}</p>
    </div>`;
  }
  return Object.freeze({ renderizarAlegaciones, renderizarBaremacion, renderizarMeritos });
}
