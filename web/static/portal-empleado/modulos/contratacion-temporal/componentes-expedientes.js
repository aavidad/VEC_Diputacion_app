/** Componentes HTML puros de la superficie de expedientes. */

import "./atajos-incidencia.js";
import "./fases-expediente.js";
import { CAPACIDADES_CONTRATACION_TEMPORAL, versionPropuestaDocumentalValida } from "./contrato-expedientes.js";

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
  const esExpediente = estado.vista === "expediente" || Boolean(estado.expediente_ref);
  const tieneAcciones = estado.carga === "error" || estado.carga === "vacio" || esExpediente;
  return `<section class="ct-exp-estado-global ct-tono-${tono}" role="${tono === "peligro" ? "alert" : "status"}"
    ${estado.carga === "cargando" ? 'aria-busy="true"' : ""} tabindex="-1">
    <h3>${escaparHTML(t(titulo))}</h3>
    <p>${escaparHTML(t(detalle))}</p>
    ${tieneAcciones ? `<div class="ct-exp-acciones-estado">
      ${(estado.carga === "error" || estado.carga === "vacio")
    ? `<button type="button" class="boton-secundario" data-ct-exp-accion="reintentar">${escaparHTML(t("reintentar"))}</button>`
    : ""}
      ${estado.carga === "vacio" && !esExpediente
    ? `<button type="button" class="boton-secundario" data-ct-exp-accion="limpiar-filtros">${escaparHTML(t("limpiar_filtros"))}</button>`
    : ""}
      ${esExpediente
    ? `<button type="button" class="boton-secundario" data-ct-exp-vista="cuadro">${escaparHTML(t("volver_cuadro"))}</button>`
    : ""}
    </div>` : ""}
  </section>`;
}

function opcionFiltro(valor, etiqueta, seleccionado) {
  return `<option value="${escaparHTML(valor)}"${valor === seleccionado ? " selected" : ""}>${escaparHTML(etiqueta)}</option>`;
}

function renderizarTrabajoOperativo(cuadro, t) {
  const esDemostracion = cuadro.demostracion === true;
  // Sin expedientes no hay bandeja ni distribución que mostrar.
  if (!esDemostracion && cuadro.expedientes.length === 0) return "";
  const expedientesNoCompletados = cuadro.expedientes
    .filter(({ estado_clave: estado }) => estado !== "completado")
    .slice(0, 3);
  const distribucion = [...new Set(cuadro.expedientes.map(({ fase_actual: fase }) => fase))]
    .map((fase) => ({
      fase,
      total: cuadro.expedientes.filter(({ fase_actual: actual }) => actual === fase).length,
    }));
  const primero = expedientesNoCompletados[0]
    ?? (esDemostracion ? cuadro.expedientes[0] : undefined);
  const titulo = esDemostracion ? t("trabajo_titulo") : t("bandeja_titulo");
  const descripcion = esDemostracion ? t("trabajo_descripcion") : t("bandeja_descripcion");
  const tituloExpedientes = esDemostracion ? t("mis_tareas") : t("bandeja_expedientes");
  const tituloDistribucion = esDemostracion
    ? t("distribucion_fases") : t("bandeja_distribucion_fases");
  return `<section class="ct-exp-operativo" aria-labelledby="ct-exp-operativo-titulo">
    <header>
      <p class="sobrelinea">${escaparHTML(t("trabajo_sobrelinea"))}</p>
      <h3 id="ct-exp-operativo-titulo">${escaparHTML(titulo)}</h3>
      <p>${escaparHTML(descripcion)}</p>
    </header>
    <article class="ct-exp-mis-tareas">
      <h4>${escaparHTML(tituloExpedientes)}</h4>
      <ul>${expedientesNoCompletados.map((expediente) => `<li>
        <span><strong>${escaparHTML(expediente.numero_visible)}</strong>
          <small>${escaparHTML(expediente.categoria)} · ${escaparHTML(expediente.fase_actual)} · ${escaparHTML(expediente.estado)}</small>
        </span>
        <button type="button" class="boton-terciario"
          data-ct-exp-abrir="${escaparHTML(expediente.expediente_ref)}">${escaparHTML(t("abrir"))}</button>
      </li>`).join("") || `<li class="ct-exp-vacio">${escaparHTML(t("bandeja_sin_expedientes"))}</li>`}</ul>
    </article>
    <article class="ct-exp-distribucion">
      <h4>${escaparHTML(tituloDistribucion)}</h4>
      <dl>${distribucion.map(({ fase, total }) => `<div>
        <dt>${escaparHTML(fase)}</dt><dd>${total}</dd>
      </div>`).join("")}</dl>
    </article>
    <aside class="ct-exp-accesos">
      <h4>${escaparHTML(t("accesos_rapidos"))}</h4>
      <button type="button" class="boton-primario"
        data-ct-exp-vista="alta">${escaparHTML(t("crear_peticion"))}</button>
      ${primero ? `<button type="button" class="boton-secundario"
        data-ct-exp-abrir="${escaparHTML(primero.expediente_ref)}">${escaparHTML(esDemostracion ? t("continuar_tramitacion") : t("bandeja_abrir_primero"))}</button>` : ""}
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
  const fases = [...new Map(cuadro.expedientes.map((expediente) => [
    expediente.fase_clave ?? expediente.fase_actual.toLocaleLowerCase("es-ES"),
    expediente.fase_actual,
  ])).entries()].map(([clave, etiqueta]) => ({ clave, etiqueta }))
    .sort((a, b) => a.etiqueta.localeCompare(b.etiqueta, "es"));
  const indicadores = `<section class="ct-exp-indicadores" aria-label="${escaparHTML(t(cuadro.paginacion ? "indicadores_pagina" : "indicadores"))}">
    ${cuadro.indicadores.map((indicador) => `<article class="ct-exp-indicador ct-tono-${escaparHTML(indicador.tono)}">
      <span>${escaparHTML(cuadro.paginacion ? t("indicador_ambito_pagina", { indicador: indicador.etiqueta }) : indicador.etiqueta)}</span>
      <strong>${escaparHTML(indicador.valor)}</strong>
    </article>`).join("")}
  </section>`;
  const filtros = `<form class="ct-exp-filtros" data-ct-exp-filtros aria-label="${escaparHTML(t("filtros"))}">
    <label>
      <span>${escaparHTML(t("filtro_texto"))}</span>
      <input type="search" name="texto" value="${escaparHTML(estado.filtros.texto)}"
        maxlength="80" placeholder="${escaparHTML(t("filtro_texto_placeholder"))}">
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
        ${fases.map(({ clave, etiqueta }) => opcionFiltro(clave, etiqueta, estado.filtros.fase)).join("")}
      </select>
    </label>
    <div class="ct-exp-filtros-acciones">
      <button type="submit" class="boton-primario">${escaparHTML(t("aplicar_filtros"))}</button>
      <button type="button" class="boton-secundario" data-ct-exp-accion="limpiar-filtros">${escaparHTML(t("limpiar_filtros"))}</button>
    </div>
  </form>`;
  const filas = cuadro.expedientes.map((expediente) => {
    const resumenId = `ct-exp-resumen-${expediente.expediente_ref}`;
    const controlId = `ct-exp-resumen-control-${expediente.expediente_ref}`;
    return `<tr>
    <th scope="row"><button type="button" class="enlace-tabla" id="${escaparHTML(controlId)}"
      data-ct-exp-resumen aria-controls="${escaparHTML(resumenId)}" aria-expanded="false"
      aria-label="${escaparHTML(t("resumen_fila", { expediente: expediente.numero_visible }))}">${escaparHTML(expediente.numero_visible)}</button></th>
    <td>${escaparHTML(expediente.centro)}</td>
    <td>${escaparHTML(expediente.categoria)}</td>
    <td>${escaparHTML(expediente.modalidad)}</td>
    <td><span class="ct-exp-chip ${estadoClave(expediente.estado_clave)}">${escaparHTML(expediente.estado)}</span></td>
    <td>${escaparHTML(expediente.fase_actual)}</td>
    <td>${escaparHTML(expediente.plazo)}</td>
    <td><button type="button" class="boton-terciario" data-ct-exp-abrir="${escaparHTML(expediente.expediente_ref)}">${escaparHTML(t("abrir"))}</button></td>
  </tr>
  <tr class="ct-exp-fila-resumen" id="${escaparHTML(resumenId)}" data-ct-exp-resumen-fila
    aria-labelledby="${escaparHTML(controlId)}" hidden>
    <td colspan="8">
      <section aria-label="${escaparHTML(t("resumen_fila", { expediente: expediente.numero_visible }))}">
        <dl>
          <div><dt>${escaparHTML(t("columna_centro"))}</dt><dd>${escaparHTML(expediente.centro)}</dd></div>
          <div><dt>${escaparHTML(t("columna_categoria"))}</dt><dd>${escaparHTML(expediente.categoria)}</dd></div>
          <div><dt>${escaparHTML(t("columna_modalidad"))}</dt><dd>${escaparHTML(expediente.modalidad)}</dd></div>
          <div><dt>${escaparHTML(t("columna_estado"))}</dt><dd><span class="ct-exp-chip ${estadoClave(expediente.estado_clave)}">${escaparHTML(expediente.estado)}</span></dd></div>
          <div><dt>${escaparHTML(t("columna_fase"))}</dt><dd>${escaparHTML(expediente.fase_actual)}</dd></div>
          <div><dt>${escaparHTML(t("columna_plazo"))}</dt><dd>${escaparHTML(expediente.plazo)}</dd></div>
        </dl>
        <button type="button" class="boton-terciario" data-ct-exp-abrir="${escaparHTML(expediente.expediente_ref)}">${escaparHTML(t("resumen_abrir_expediente"))}</button>
      </section>
    </td>
  </tr>`;
  }).join("");
  const tabla = `<section class="panel ct-exp-listado">
    <div class="cabecera-panel">
      <h3>${escaparHTML(t("tabla_expedientes"))}</h3>
      <span class="estado-chip info">${escaparHTML(t(cuadro.paginacion ? "resultados_pagina" : "resultados", { total: cuadro.expedientes.length }))}</span>
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
  const paginacion = cuadro.paginacion ? `<nav class="ct-exp-paginacion" aria-label="${escaparHTML(t("paginacion"))}">
    <span>${escaparHTML(t("pagina_actual", { pagina: cuadro.paginacion.pagina }))}</span>
    <button type="button" class="boton-secundario" data-ct-exp-pagina="primera"
      ${cuadro.paginacion.pagina === 1 && !estado.paginacion_requiere_reinicio && estado.carga !== "error" ? "disabled" : ""}>${escaparHTML(t("pagina_primera"))}</button>
    <button type="button" class="boton-secundario" data-ct-exp-pagina="siguiente"
      ${cuadro.paginacion.cursor_siguiente && !estado.paginacion_requiere_reinicio && estado.carga !== "error" ? "" : "disabled"}>${escaparHTML(t("pagina_siguiente"))}</button>
  </nav>` : "";
  const trabajoOperativo = renderizarTrabajoOperativo(cuadro, t);
  const organizacion = `<p><a class="boton-secundario" href="/portal-empleado/organizacion/" target="_blank" rel="noopener">${escaparHTML(t("organizacion_referencia"))}</a> <a class="boton-secundario" href="/portal-empleado/peticiones-centro/?vista=rrhh" target="_blank" rel="noopener">${escaparHTML(t("peticiones_centros_rrhh"))}</a></p>`;
  return `${indicadores}${organizacion}${filtros}${estado.carga === "vacio"
    ? renderizarEstadoCarga(estado, t) : `${tabla}${paginacion}`}${trabajoOperativo}`;
}

// La incidencia se explica con lo que el detalle ya trae: la fase marcada, el
// hito que la originó y lo ocurrido después. Los atajos llevan a donde se
// resuelve o se consulta; ninguno ejecuta una acción por sí mismo.
const ACCION_SUBSANACION = "contratacion_temporal.subsanacion_reparos.registrar";

function renderizarIncidencia(expediente, t, navegacion) {
  const fase = (expediente.fases || []).find((f) => f.estado_clave === "incidencia");
  if (!fase) return "";
  const historial = expediente.historial || [];
  const origen = historial.find((hito, i) => hito.estado_clave === "incidencia"
    && (i === 0 || historial[i - 1].estado_clave !== "incidencia"));
  const posteriores = origen ? historial.filter((hito) => hito.secuencia > origen.secuencia) : [];
  const subsanacion = posteriores.find((hito) => hito.accion_clave === ACCION_SUBSANACION);
  const situacion = subsanacion
    ? t("incidencia_subsanada", { fecha: subsanacion.fecha })
    : t("incidencia_pendiente_subsanacion");
  const fiscalizacion = expediente.fiscalizacion;
  const reparos = (fiscalizacion?.reparos ?? []).map((r) => `<blockquote class="ct-exp-incidencia-reparo">${escaparHTML(r.texto)}</blockquote>`).join("");
  const subsanacionTexto = fiscalizacion?.subsanacion
    ? `<p class="ct-exp-incidencia-subsanacion"><strong>${escaparHTML(t("incidencia_subsanacion_texto"))}</strong> ${escaparHTML(fiscalizacion.subsanacion.texto)}</p>`
    : "";
  return `<section class="ct-exp-incidencia" role="alert" aria-labelledby="ct-exp-incidencia-titulo">
    <div class="ct-exp-incidencia-texto">
      <h3 id="ct-exp-incidencia-titulo">${escaparHTML(t("incidencia_titulo", { fase: fase.etiqueta }))}</h3>
      ${origen ? `<p>${escaparHTML(t("incidencia_origen", { accion: origen.accion, fecha: origen.fecha, secuencia: origen.secuencia }))}</p>` : ""}
      ${reparos ? `<p><strong>${escaparHTML(t("incidencia_reparos"))}</strong></p>${reparos}` : ""}
      <p>${escaparHTML(situacion)}</p>
      ${subsanacionTexto}
    </div>
    <div class="ct-exp-incidencia-atajos">
      <button type="button" class="boton-primario" data-ct-exp-accion="abrir-historial">${escaparHTML(t("incidencia_atajo_historial"))}</button>
      ${navegacion?.documentos === false ? "" : `<button type="button" class="boton-secundario" data-ct-exp-vista="documentos">${escaparHTML(t("incidencia_atajo_documentos"))}</button>`}
      ${navegacion?.auditoria === false ? "" : `<button type="button" class="boton-secundario" data-ct-exp-vista="auditoria">${escaparHTML(t("incidencia_atajo_auditoria"))}</button>`}
    </div>
  </section>`;
}

function renderizarFases(expediente, t) {
  if (expediente.fases.length === 0) return "";
  return `<nav class="ct-exp-progreso" aria-label="${escaparHTML(t("fases_expediente"))}">
    <ol>${expediente.fases.map((fase) => `<li class="${estadoClave(fase.estado_clave)}"
      ${fase.estado_clave === "en_curso" ? 'aria-current="step"' : ""}>
      <button type="button" class="ct-exp-fase-boton" data-ct-exp-fase-ver="${escaparHTML(claveDeFase(fase))}" aria-pressed="false"
        aria-label="${escaparHTML(t("fase_ver_pantalla", { fase: fase.etiqueta }))}">
        <span class="ct-exp-numero-fase" aria-hidden="true">${fase.orden}</span>
        <span>${escaparHTML(fase.etiqueta)}</span>
        <small>${escaparHTML(textoEstado(fase.estado_clave, t))}</small>
      </button>
    </li>`).join("")}</ol>
  </nav>`;
}

function renderizarHistorialHitos(expediente, t) {
  if (!Array.isArray(expediente.historial) || expediente.historial.length === 0) return "";
  return `<details class="ct-exp-detalle-tecnico ct-exp-historial">
    <summary>${escaparHTML(t("historial_hitos_titulo"))} (${expediente.historial.length})</summary>
    <p>${escaparHTML(t("historial_hitos_descripcion"))}</p>
    <div class="tabla-contenedor" tabindex="0" role="region" aria-label="${escaparHTML(t("historial_hitos_titulo"))}">
      <table class="tabla-datos ct-exp-tabla-panel">
        <thead><tr><th scope="col">${escaparHTML(t("historial_hito_secuencia"))}</th>
          <th scope="col">${escaparHTML(t("historial_hito_fecha"))}</th>
          <th scope="col">${escaparHTML(t("historial_hito_accion"))}</th>
          <th scope="col">${escaparHTML(t("historial_hito_fase"))}</th>
          <th scope="col">${escaparHTML(t("historial_hito_estado"))}</th></tr></thead>
        <tbody>${expediente.historial.map((hito) => `<tr data-ct-exp-hito-fase="${escaparHTML(hito.fase)}" data-ct-exp-hito-accion="${escaparHTML(hito.accion_clave ?? "")}">
          <td>${hito.secuencia}</td><td>${escaparHTML(hito.fecha)}</td>
          <td>${escaparHTML(hito.accion)}</td><td>${escaparHTML(hito.fase)}</td>
          <td>${escaparHTML(hito.estado)}</td></tr>`).join("")}</tbody>
      </table>
    </div>
  </details>`;
}

const BORRADORES_FORMALIZACION = Object.freeze([
  ["informe_definitivo", "informe-definitivo"], ["resolucion", "resolucion"],
  ["diligencia", "diligencia"], ["toma_posesion", "toma-posesion"],
  ["notificacion", "notificacion"], ["comunicacion_centro", "comunicacion-centro"],
]);

function renderizarBorradoresFormalizacion(t) {
  return `<section class="ct-exp-borradores" aria-labelledby="ct-exp-borradores-titulo">
    <div class="ct-exp-borradores-cabecera"><div>
      <h4 id="ct-exp-borradores-titulo">${escaparHTML(t("borradores_titulo"))}</h4>
      <p>${escaparHTML(t("borradores_descripcion"))}</p>
    </div><button type="button" class="boton-terciario" data-ct-exp-accion="cancelar-descarga" disabled>${escaparHTML(t("cancelar_descarga"))}</button></div>
    <ul>${BORRADORES_FORMALIZACION.map(([clave, accion]) => `<li>
      <h5>${escaparHTML(t(`${clave}_titulo`))}</h5>
      <p>${escaparHTML(t("borrador_sin_firma"))}</p>
      <div class="ct-exp-borradores-acciones">
        <button type="button" class="boton-secundario" data-ct-exp-accion="descargar-${accion}">${escaparHTML(t(`${clave}_descargar`))}</button>
        <button type="button" class="boton-secundario" data-ct-exp-accion="descargar-docx-${accion}">${escaparHTML(t(`${clave}_descargar_docx`))}</button>
      </div>
      <p data-ct-exp-resultado-descarga="${accion}" role="status" aria-live="polite">${escaparHTML(t("descarga_sin_solicitar"))}</p>
      <button type="button" class="boton-terciario" data-ct-exp-accion="reintentar-descarga-${accion}" disabled hidden>${escaparHTML(t("reintentar_descarga"))}</button>
    </li>`).join("")}</ul>
  </section>`;
}

export function solicitudInformeDefinitivoDesdeEstado(estado) {
  const expediente = estado.expediente;
  const indiceActual = estado.vista !== "documentos" || (estado.documentos?.demostracion === false
    && estado.documentos.expediente_ref === expediente?.expediente_ref
    && estado.documentos.version === expediente?.version);
  if (!(["expediente", "documentos"].includes(estado.vista)) || !indiceActual || estado.carga !== "listo" || estado.ocupado
    || estado.actualizacion_pendiente || estado.resultado_indeterminado
    || expediente?.demostracion !== false || !Number.isSafeInteger(expediente.version) || expediente.version < 7
    || (expediente.version >= 8 && !versionPropuestaDocumentalValida(expediente))
    || estado.expediente_ref !== expediente.expediente_ref
    || estado.cuadro?.demostracion !== false) return null;
  const resumen = estado.cuadro.expedientes.find(({ expediente_ref }) => expediente_ref === expediente.expediente_ref);
  if (resumen?.version !== expediente.version || resumen.fase_clave !== "nombramiento"
    || resumen.estado_clave !== "en_curso") return null;
  return Object.freeze({ expediente_ref: expediente.expediente_ref, version_observada: expediente.version });
}

// Cada dato de la cabecera pertenece a una fase del procedimiento de RRHH; el
// raíl permite ver la pantalla de cada fase con sus datos y sus actuaciones.
const FASE_DE_CAMPO = Object.freeze({
  centro: "solicitud", categoria: "solicitud", modalidad: "solicitud", grupo_subgrupo: "solicitud",
  motivo: "solicitud", periodo: "solicitud",
  periodo_analizado: "analisis_rrhh", causa: "analisis_rrhh", jornada: "analisis_rrhh",
  resultado_rc: "analisis_rrhh", coste_estimado: "analisis_rrhh", observaciones: "analisis_rrhh",
  via_cobertura: "gestion_bolsa", decision_gobernada: "gestion_bolsa", unidad: "gestion_bolsa",
});

function faseDeCampo(clave) {
  if (clave.startsWith("comprobacion_")) return "gestion_bolsa";
  return FASE_DE_CAMPO[clave] ?? "general";
}

function claveDeFase(fase) {
  return String(fase.fase_ref ?? "").split(":").at(-1) ?? "";
}

function renderizarCabecera(expediente, t, informeDisponible = false) {
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
    <dl>${expediente.cabecera.map((campo) => `<div data-ct-exp-campo-fase="${escaparHTML(faseDeCampo(campo.clave))}">
      <dt>${escaparHTML(campo.etiqueta)}</dt>
      <dd class="ct-tono-${escaparHTML(campo.tono)}">${escaparHTML(campo.valor)}</dd>
    </div>`).join("")}</dl>
    ${informeDisponible ? renderizarBorradoresFormalizacion(t) : ""}
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

function renderizarTarea(
  expediente,
  tareaRef,
  estado,
  t,
  locale,
  zonaHoraria,
  analisisDisponible,
) {
  const tarea = expediente.tareas.find(({ tarea_ref: referencia }) => referencia === tareaRef)
    ?? expediente.tareas[0];
  if (!tarea) {
    const resumen = estado.cuadro?.expedientes?.find(({ expediente_ref: referencia }) => (
      referencia === expediente.expediente_ref
    ));
    const solicitudV1EnCurso = estado.carga === "listo"
      && expediente.version === 1
      && resumen?.fase_clave === "solicitud"
      && resumen.version === expediente.version
      && resumen.estado_clave === "en_curso";
    const montarAnalisis = analisisDisponible === true
      && solicitudV1EnCurso
      && !estado.ocupado
      && !estado.actualizacion_pendiente
      && estado.resultado_indeterminado !== true;
    return montarAnalisis
      ? '<div data-ct-exp-analisis></div>'
      : "";
  }
  const montarAnalisis = analisisDisponible === true
    && !estado.ocupado
    && !estado.actualizacion_pendiente
    && estado.resultado_indeterminado !== true
    && tarea.acciones.some((accion) => (
      accion.tipo === "efecto"
        && accion.disponible === true
        && accion.capacidad === CAPACIDADES_CONTRATACION_TEMPORAL.analizar
    ));
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
    ${montarAnalisis ? '<div data-ct-exp-analisis></div>' : `<form data-ct-exp-tarea-form aria-label="${escaparHTML(t("formulario_tarea", { tarea: tarea.etiqueta }))}">
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
    </form>`}
    ${montarAnalisis ? "" : renderizarRecibo(estado.recibo, t, locale, zonaHoraria)}
  </article>`;
}

export function renderizarExpediente(estado, t, locale, zonaHoraria, analisisDisponible = false) {
  const expediente = estado.expediente;
  if (!expediente) {
    const esError = estado.carga === "error";
    const tono = esError ? "peligro" : "neutro";
    return `<section class="ct-exp-estado-global ct-tono-${tono}" role="${esError ? "alert" : "status"}" tabindex="-1">
      ${esError ? `<h3>${escaparHTML(t("error_titulo"))}</h3>` : ""}
      <p>${escaparHTML(t(esError ? "expediente_error_carga" : "expediente_sin_seleccionar"))}</p>
      <div class="ct-exp-acciones-estado">
        ${esError ? `<button type="button" class="boton-secundario" data-ct-exp-accion="reintentar">${escaparHTML(t("reintentar"))}</button>` : ""}
        <button type="button" class="boton-secundario" data-ct-exp-vista="cuadro">${escaparHTML(t("volver_cuadro"))}</button>
      </div>
    </section>`;
  }
  const tramitacion = expediente.tareas.length === 0
    ? renderizarTarea(expediente, estado.tarea_ref, estado, t, locale, zonaHoraria, analisisDisponible)
    : `<div class="ct-exp-tramitacion">
      ${renderizarTareas(expediente, estado.tarea_ref, t)}
      ${renderizarTarea(
    expediente,
    estado.tarea_ref,
    estado,
    t,
    locale,
    zonaHoraria,
    analisisDisponible,
  )}
    </div>`;
  return `${renderizarIncidencia(expediente, t, estado.navegacion)}
    ${renderizarCabecera(expediente, t, solicitudInformeDefinitivoDesdeEstado(estado) !== null)}
    ${renderizarFases(expediente, t)}
    ${tramitacion}
    ${renderizarHistorialHitos(expediente, t)}`;
}

export function renderizarDocumentos(estado, t) {
  const expediente = estado.expediente;
  const indice = estado.documentos;
  if (!expediente || !indice) return renderizarExpediente(estado, t, "es-ES", "Europe/Madrid");
  return `${renderizarCabecera(expediente, t)}
    <header class="ct-exp-subcabecera"><h3>${escaparHTML(t("documentos_titulo"))}</h3><p>${escaparHTML(t("documentos_descripcion"))}</p></header>
    ${solicitudInformeDefinitivoDesdeEstado(estado) ? renderizarBorradoresFormalizacion(t) : ""}
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
