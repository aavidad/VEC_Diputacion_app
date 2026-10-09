/** Componentes HTML puros de la superficie de expedientes. */

import "./atajos-incidencia.js";
import { renderizarResultadoBolsa, continuidadBolsaDisponible } from "./resultado-bolsa.js?v=20261009-ct-resultado-bolsa-v1";
import { CAPACIDADES_CONTRATACION_TEMPORAL, versionPropuestaDocumentalValida } from "./contrato-expedientes.js?v=20261002-ct-fin-modalidad-v1";
import { renderizarCambiosExpediente } from "./vista-expedientes-cambios.js?v=20261008-w-ct-borradores-main-v2";
import { crearTraductorExpedientesContratacion } from "./i18n-expedientes.js?v=20261007-pantallas-textos-final-v1";
import { justificanteTraducido } from "../../portal-justificante.js";
import { traducirPortal } from "../../portal-i18n.js?v=20261007-pantallas-textos-final-v1";
import { origenLlamamientoValido } from "../../portal-llamamiento-origen.js";
import { FILTRO_LISTA_INICIAL, filtroListaValido } from "./recuentos-peticiones.js?v=20261007-pantallas-textos-final-v1";
import { renderizarListaPeticiones, renderizarResultadosLista } from "./vista-expedientes-lista.js?v=20261008-ct-inicio-v1";
import {
  renderizarCabeceraFicha, renderizarDatosPeticion, renderizarDocumentosFicha, renderizarHistorialFicha,
  renderizarLineaFases, renderizarSiguientePasoFicha,
} from "./vista-expedientes-ficha.js?v=20261008-r-fichas-idioma-nav-v1";

const traductorPorOmision = crearTraductorExpedientesContratacion();

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


function centroVisible(centro, t = traductorPorOmision) {
  const referencia = String(centro ?? "");
  const coincidencia = /^centro:([^:]+):(\d+)$/u.exec(referencia);
  if (!coincidencia) return { etiqueta: referencia, referencia: "" };
  const [, ambito, numero] = coincidencia;
  return { etiqueta: t("centro_visible", { ambito: ambito.replaceAll(/[-_]+/g, " "), numero }), referencia };
}


// Los expedientes dados de alta antes de la numeración anual conservan en su
// historia el identificador técnico («2026/CT-8c17ba0b…»): figuran sin numerar.
export function numeroExpedienteVisible(numero, t = traductorPorOmision) {
  const texto = String(numero ?? "");
  return /^\d{4}\/CT-[0-9a-f]{12,}$/iu.test(texto) ? t("numero_expediente_sin_asignar") : texto;
}

// Número y centro legibles que la lista necesita (evita dependencias circulares).
// Con el traductor del idioma activo, para que el centro sin nombre también
// se diga en ese idioma.
function ayudasLista(t) {
  return Object.freeze({
    numeroVisible: (numero) => numeroExpedienteVisible(numero, t),
    centroVisible: (centro) => centroVisible(centro, t),
  });
}


export function renderizarCuadro(estado, t, filtroLista = FILTRO_LISTA_INICIAL) {
  const cuadro = estado.cuadro;
  if (!cuadro) return renderizarEstadoCarga(estado, t);
  // La paginación del servidor solo aparece cuando hay otra página o hay que
  // reiniciar la consulta; entonces va junto a la tabla.
  const paginacionRemota = cuadro.paginacion && (cuadro.paginacion.pagina > 1
    || Boolean(cuadro.paginacion.cursor_siguiente)
    || estado.paginacion_requiere_reinicio === true || estado.carga === "error");
  const paginacion = paginacionRemota ? `<nav class="ct-exp-paginacion" aria-label="${escaparHTML(t("paginacion"))}">
    <span>${escaparHTML(t("pagina_actual", { pagina: cuadro.paginacion.pagina }))}</span>
    <button type="button" class="boton-secundario" data-ct-exp-pagina="primera"
      ${cuadro.paginacion.pagina === 1 && !estado.paginacion_requiere_reinicio && estado.carga !== "error" ? "disabled" : ""}>${escaparHTML(t("pagina_primera"))}</button>
    <button type="button" class="boton-secundario" data-ct-exp-pagina="siguiente"
      ${cuadro.paginacion.cursor_siguiente && !estado.paginacion_requiere_reinicio && estado.carga !== "error" ? "" : "disabled"}>${escaparHTML(t("pagina_siguiente"))}</button>
  </nav>` : "";
  const filtrosServidor = Object.values(estado.filtros ?? {}).some((valor) => valor !== "" && valor != null);
  if (estado.carga === "vacio" && filtrosServidor) return renderizarEstadoCarga(estado, t);
  return renderizarListaPeticiones(estado, t, filtroListaValido(filtroLista), ayudasLista(t), paginacion);
}

/** Solo la parte de resultados, para repintar al escribir sin perder el foco. */
export function renderizarResultadosCuadro(estado, t, filtroLista = FILTRO_LISTA_INICIAL) {
  return estado.cuadro ? renderizarResultadosLista(estado, t, filtroListaValido(filtroLista), ayudasLista(t)) : "";
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




const BORRADORES_FORMALIZACION = Object.freeze([
  ["informe_definitivo", "informe-definitivo"], ["resolucion", "resolucion"],
  ["diligencia", "diligencia"], ["toma_posesion", "toma-posesion"],
  ["notificacion", "notificacion"], ["comunicacion_centro", "comunicacion-centro"],
  ["contrato_laboral", "contrato-laboral"], ["nombramiento", "nombramiento"],
  ["cese", "cese"], ["modificacion_nombramiento", "modificacion-nombramiento"],
]);

function renderizarBorradoresFormalizacion(t) {
  return `<section class="ct-exp-borradores" aria-labelledby="ct-exp-borradores-titulo">
    <div class="ct-exp-borradores-cabecera"><div>
      <h4 id="ct-exp-borradores-titulo">${escaparHTML(t("borradores_titulo"))}</h4>
    </div><button type="button" class="boton-terciario" data-ct-exp-accion="cancelar-descarga" disabled>${escaparHTML(t("cancelar_descarga"))}</button></div>
    <ul>${BORRADORES_FORMALIZACION.map(([clave, accion]) => `<li>
      <h5>${escaparHTML(t(`${clave}_titulo`))} <span class="ct-exp-chip">${escaparHTML(t("ficha_borrador_sin_firmar"))}</span></h5>
      <div class="ct-exp-borradores-acciones">
        <button type="button" class="boton-secundario" data-ct-exp-accion="descargar-${accion}" aria-label="${escaparHTML(t(`${clave}_descargar`))}">${escaparHTML(t("ficha_descargar_pdf"))}</button>
        <button type="button" class="boton-secundario" data-ct-exp-accion="descargar-docx-${accion}" aria-label="${escaparHTML(t(`${clave}_descargar_docx`))}">${escaparHTML(t("ficha_descargar_word"))}</button>
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
  bolsa_cobertura: "gestion_bolsa",
});

function faseDeCampo(clave) {
  if (clave.startsWith("comprobacion_")) return "gestion_bolsa";
  return FASE_DE_CAMPO[clave] ?? "general";
}

function claveDeFase(fase) {
  return String(fase.fase_ref ?? "").split(":").at(-1) ?? "";
}

// La bolsa de la cobertura enlaza con su histórico de llamamientos en Bolsa
// (el enlace lo atiende el controlador de Bolsa del portal) solo si el perfil
// ve esa bolsa; su referencia opaca no se muestra nunca como texto.
function valorCampoCabecera(campo, t, resolverBolsa) {
  if (campo.clave !== "bolsa_cobertura") return escaparHTML(campo.valor);
  const bolsa = typeof resolverBolsa === "function" ? resolverBolsa(campo.valor) : null;
  if (!bolsa?.categoria) return null;
  return `<button type="button" class="enlace-tabla" data-accion="ver-bolsa" data-bolsa-ref="${escaparHTML(campo.valor)}" data-pestana="historico" aria-label="${escaparHTML(t("enlace_bolsa_historico_aria", { bolsa: bolsa.categoria }))}">${escaparHTML(bolsa.categoria)}</button>`;
}

// «Abrir llamamiento en Bolsa»: la bolsa elegida para cubrir la petición o, si
// no hay, la vigente de su categoría. Bolsa abre el asistente con la referencia,
// el centro y la fecha de inicio de la petición ya puestos.
export function renderizarAbrirLlamamiento(expediente, resolverBolsa, t) {
  if (typeof resolverBolsa !== "function") return "";
  const valor = (clave) => expediente.cabecera?.find((campo) => campo.clave === clave)?.valor;
  const cobertura = valor("bolsa_cobertura");
  let bolsaRef = typeof cobertura === "string" && resolverBolsa(cobertura)?.categoria ? cobertura : "";
  let sinBolsaConfirmada = false;
  let incidenciaBolsa = "";
  if (!bolsaRef) {
    const categoriaRef = expediente.analisis_previo?.categoria_ref ?? expediente.datos_peticion?.categoria_ref;
    const bolsaCategoria = typeof categoriaRef === "string" ? resolverBolsa("", { categoriaRef }) : null;
    bolsaRef = bolsaCategoria?.bolsa_ref || "";
    sinBolsaConfirmada = !cobertura && bolsaCategoria?.estado === "sin_bolsa";
    incidenciaBolsa = ["error", "denegado"].includes(bolsaCategoria?.estado) ? bolsaCategoria.estado : "";
  }
  // El detalle CT conserva fechas civiles a medianoche UTC; el origen de Bolsa
  // transporta solo el día y valida de nuevo su calendario.
  const inicioCT = expediente.analisis_previo?.periodo?.inicio ?? expediente.datos_peticion?.periodo?.inicio;
  const inicioBolsa = typeof inicioCT === "string" && /^\d{4}-\d{2}-\d{2}T00:00:00Z$/u.test(inicioCT)
    ? inicioCT.slice(0, 10) : inicioCT;
  const origen = bolsaRef ? origenLlamamientoValido({
    expediente_ref: expediente.expediente_ref, referencia: expediente.numero_visible, centro: valor("centro"),
    fecha_inicio: inicioBolsa,
  }) : null;
  if (!origen) {
    if (sinBolsaConfirmada) return `<section class="panel" role="status"><div class="cuerpo-panel"><p>${escaparHTML(t("ficha_llamamiento_sin_bolsa"))}</p></div></section>`;
    if (incidenciaBolsa) return `<section class="panel" role="status"><div class="cuerpo-panel"><p>${escaparHTML(t(`ficha_llamamiento_bolsa_${incidenciaBolsa}`))}</p>${incidenciaBolsa === "error" ? `<button type="button" class="boton-secundario" data-ct-bolsa-reintentar>${escaparHTML(t("ficha_llamamiento_bolsa_reintentar"))}</button>` : ""}</div></section>`;
    return "";
  }
  const atributo = (nombre, dato) => (dato ? ` data-origen-${nombre}="${escaparHTML(dato)}"` : "");
  const etiqueta = continuidadBolsaDisponible(expediente) ? t("resultado_bolsa_continuar") : traducirPortal("panel_ct_abrir_llamamiento");
  return `<div class="acciones-vista"><button type="button" class="boton-primario" data-accion="ver-bolsa" data-bolsa-ref="${escaparHTML(bolsaRef)}"${atributo("expediente", origen.expediente_ref)}${atributo("referencia", origen.referencia)}${atributo("centro", origen.centro)}${atributo("inicio", origen.fecha_inicio)}>${escaparHTML(etiqueta)}</button></div>`;
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
      <div><dt>${escaparHTML(t("recibo_referencia"))}</dt><dd>${justificanteTraducido(recibo.recibo_ref, escaparHTML, t)}</dd></div>
      <div><dt>${escaparHTML(t("recibo_expediente"))}</dt><dd>${escaparHTML(recibo.numero_visible)}</dd></div>
      <div><dt>${escaparHTML(t("recibo_actuacion"))}</dt><dd>${escaparHTML(recibo.actuacion)}</dd></div>
      <div><dt>${escaparHTML(t("recibo_estado"))}</dt><dd>${escaparHTML(recibo.estado_resultante)}</dd></div>
      <div><dt>${escaparHTML(t("recibo_fecha"))}</dt><dd>${escaparHTML(fecha)}</dd></div>
    </dl>
    <details class="detalle-tecnico-plegado"><summary>${escaparHTML(t("ficha_detalle_tecnico"))}</summary>
      <dl><div><dt>${escaparHTML(t("recibo_version"))}</dt><dd>${escaparHTML(recibo.version)}</dd></div></dl>
    </details>
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
      <div><dt>${escaparHTML(t("tiempo"))}</dt><dd>${escaparHTML(tarea.tiempo)}</dd></div>
    </dl>
    <details class="detalle-tecnico-plegado"><summary>${escaparHTML(t("ficha_detalle_tecnico"))}</summary>
      <dl class="ct-exp-metadata">
        <div><dt>${escaparHTML(t("entrada"))}</dt><dd>${escaparHTML(tarea.entrada)}</dd></div>
        <div><dt>${escaparHTML(t("salida"))}</dt><dd>${escaparHTML(tarea.salida || "—")}</dd></div>
        <div><dt>${escaparHTML(t("recibo"))}</dt><dd>${justificanteTraducido(tarea.recibo_ref, escaparHTML, t)}</dd></div>
      </dl>
    </details>
    ${montarAnalisis ? '<div data-ct-exp-analisis></div>' : `<form data-ct-exp-tarea-form aria-label="${escaparHTML(t("formulario_tarea", { tarea: tarea.etiqueta }))}">
      ${editable ? "" : `<p class="ct-exp-solo-lectura">${escaparHTML(t("tarea_solo_lectura"))}</p>`}
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

export function renderizarExpediente(estado, t, locale, zonaHoraria, analisisDisponible = false, resolverBolsa = null, coberturaPendiente = false) {
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
  // Orden de la ficha: qué toca, en qué fase está y qué hay; los trámites de
  // la fase se montan después, a partir de la marca «ct-exp-tramite».
  return `${renderizarCabeceraFicha(expediente, estado, t)}
    ${renderizarSiguientePasoFicha(expediente, estado, t, coberturaPendiente)}
    ${renderizarIncidencia(expediente, t, estado.navegacion)}
    ${renderizarLineaFases(expediente, t)}
    <div class="rejilla-principal ct-exp-ficha-rejilla">
      <div class="pila">
        ${renderizarDocumentosFicha(estado, t, solicitudInformeDefinitivoDesdeEstado(estado) ? renderizarBorradoresFormalizacion(t) : "")}
        ${renderizarHistorialFicha(expediente, t, faseDeCampo)}
        ${renderizarCambiosExpediente(expediente)}
      </div>
      <div class="pila">
        ${renderizarDatosPeticion(expediente, t, { valorCampo: (campo) => valorCampoCabecera(campo, t, resolverBolsa), faseDeCampo })}
        <div data-ct-resultado-bolsa-zona>${renderizarResultadoBolsa(expediente, t, locale, zonaHoraria)}</div>
        ${typeof resolverBolsa === "function" ? `<div data-ct-bolsa-ficha aria-live="polite">${renderizarAbrirLlamamiento(expediente, resolverBolsa, t)}</div>` : ""}
      </div>
    </div>
    ${tramitacion}
    ${renderizarContinuidadDesdeExpediente(estado, t)}
    <div class="ct-exp-tramite" id="ct-exp-tramite" tabindex="-1" data-ct-exp-tramite></div>`;
}

// La versión solo decide si mostrar orientación; el recibo y las consultas
// autorizadas deciden si se montan la ficha o el seguimiento reales.
function versionConPosibleIncorporacion(expediente) {
  return expediente?.demostracion === false
    && Number.isSafeInteger(expediente.version) && expediente.version >= 8;
}

function renderizarContinuidadDesdeExpediente(estado, t) {
  if (!versionConPosibleIncorporacion(estado.expediente)
    || estado.navegacion?.documentos !== true) return "";
  return `<section class="ct-exp-continuidad" aria-labelledby="ct-exp-continuidad-titulo">
    <div><h3 id="ct-exp-continuidad-titulo">${escaparHTML(t("continuidad_expediente_titulo"))}</h3>
</div>
    <button type="button" class="boton-secundario" data-ct-exp-vista="documentos">${escaparHTML(t("continuidad_expediente_documentos"))}</button>
  </section>`;
}

function renderizarContinuidadDesdeDocumentos(expediente, t) {
  if (!versionConPosibleIncorporacion(expediente)) return "";
  return `<section class="ct-exp-continuidad ct-exp-continuidad-documentos" aria-labelledby="ct-exp-continuidad-documentos-titulo">
    <h3 id="ct-exp-continuidad-documentos-titulo">${escaparHTML(t("continuidad_documentos_titulo"))}</h3>
    <div class="ct-exp-continuidad-pasos">
      <article><h4>${escaparHTML(t("continuidad_documentos_ficha"))}</h4>
        <p>${escaparHTML(t("continuidad_documentos_ficha_estado"))}</p></article>
      <article><h4>${escaparHTML(t("continuidad_documentos_seguimiento"))}</h4>
        <p>${escaparHTML(t("continuidad_documentos_seguimiento_estado"))}</p></article>
    </div>
  </section>`;
}

export function renderizarDocumentos(estado, t) {
  const expediente = estado.expediente;
  const indice = estado.documentos;
  if (!expediente || !indice) return renderizarExpediente(estado, t, "es-ES", "Europe/Madrid");
  return `${renderizarCabeceraFicha(expediente, estado, t)}
    <button type="button" class="boton-secundario" data-ct-exp-vista="expediente">${escaparHTML(t("nav_expediente"))}</button>
    ${renderizarDocumentosFicha(estado, t, solicitudInformeDefinitivoDesdeEstado(estado) ? renderizarBorradoresFormalizacion(t) : "")}
    ${renderizarContinuidadDesdeDocumentos(expediente, t)}`;
}

export function renderizarAuditoria(estado, t) {
  const expediente = estado.expediente;
  const auditoria = estado.auditoria;
  if (!expediente || !auditoria) return renderizarExpediente(estado, t, "es-ES", "Europe/Madrid");
  return `${renderizarCabeceraFicha(expediente, estado, t)}
    <header class="ct-exp-subcabecera"><h3>${escaparHTML(t("auditoria_titulo"))}</h3></header>
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
          <td>${escaparHTML(actuacion.observaciones)}</td><td>${justificanteTraducido(actuacion.documento_ref, escaparHTML, t)}</td>
        </tr>`).join("")}</tbody>
      </table>
    </div>`;
}
