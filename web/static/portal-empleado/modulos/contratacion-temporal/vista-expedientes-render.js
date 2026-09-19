/** Funciones de renderizado y extracción de contextos para expedientes de contratación temporal. */

import {
  escaparHTML, renderizarAuditoria, renderizarCuadro, renderizarDocumentos,
  renderizarEstadoCarga, renderizarExpediente,
} from "./componentes-expedientes.js";
import { crearTraductorExpedientesContratacion } from "./i18n-expedientes.js";
import { crearTraductorContratacionTemporal } from "./i18n.js";
import { PATRON_REFERENCIA } from "./vista-expedientes-analisis.js";

export function renderizarNavegacion(estado, t) {
  const opciones = [
    ["cuadro", "nav_cuadro"],
    ["alta", "nav_alta"],
    ["expediente", "nav_expediente"],
    ["documentos", "nav_documentos"],
    ["auditoria", "nav_auditoria"],
    ["estadisticas", "nav_estadisticas"],
  ];
  return `<nav class="ct-exp-navegacion" aria-label="${escaparHTML(t("navegacion"))}">
    ${opciones.map(([vista, clave]) => {
    const requiereExpediente = ["expediente", "documentos", "auditoria"].includes(vista);
    return `<button type="button" data-ct-exp-vista="${vista}"
      ${estado.vista === vista ? 'aria-current="page"' : ""}
      ${requiereExpediente && !estado.expediente && !estado.cuadro?.expedientes.length ? "disabled" : ""}>
      ${escaparHTML(t(clave))}
    </button>`;
  }).join("")}
  </nav>`;
}

export function renderizarCabeceraModulo(estado, t) {
  const demostracion = estado.cuadro?.demostracion === true
    || estado.expediente?.demostracion === true;
  return `<header class="ct-exp-cabecera-modulo">
    <div>
      <p class="sobrelinea">${escaparHTML(t("sobrelinea"))}</p>
      <h2>${escaparHTML(t("titulo"))}</h2>
      <p>${escaparHTML(t("descripcion"))}</p>
    </div>
    ${demostracion ? `<p class="ct-exp-aviso-presentacion" role="note">${escaparHTML(t("presentacion"))}</p>` : ""}
  </header>`;
}

export function renderizarAlta(
  t,
  disponible,
  analisisDisponible,
  coberturaDisponible,
  asignacionDisponible,
  informeJuridicoDisponible,
  fiscalizacionDisponible,
  catalogoDisponible = true,
) {
  if (!disponible || !catalogoDisponible) {
    const esErrorCatalogo = !catalogoDisponible;
    const titulo = esErrorCatalogo ? t("catalogo_no_disponible_titulo") : t("denegado_titulo");
    const detalle = esErrorCatalogo ? t("catalogo_no_disponible_detalle") : t("estado_denegado");
    return `<section class="ct-exp-estado-global ct-tono-peligro" role="alert" tabindex="-1">
      <h3>${escaparHTML(titulo)}</h3>
      <p>${escaparHTML(detalle)}</p>
      <div class="ct-exp-acciones-estado">
        <button type="button" class="boton-secundario" data-ct-exp-accion="reintentar">${escaparHTML(t("reintentar"))}</button>
        <button type="button" class="boton-secundario" data-ct-exp-vista="cuadro">${escaparHTML(t("volver_cuadro"))}</button>
      </div>
    </section>`;
  }
  return `<div data-ct-exp-alta></div>
    ${analisisDisponible ? '<div data-ct-exp-analisis></div>' : ""}
    ${coberturaDisponible ? '<div data-ct-exp-cobertura></div>' : ""}
    ${asignacionDisponible ? '<div data-ct-exp-asignacion></div>' : ""}
    ${informeJuridicoDisponible ? '<div data-ct-exp-informe-juridico></div>' : ""}
    ${fiscalizacionDisponible ? '<div data-ct-exp-fiscalizacion></div>' : ""}`;
}

export function contextoLlamamientoDesdeEstado(estado) {
  const expediente = estado?.expediente;
  if (estado?.vista !== "expediente" || estado.carga !== "listo"
    || estado.ocupado || estado.actualizacion_pendiente || estado.resultado_indeterminado
    || expediente?.demostracion !== false || estado.cuadro?.demostracion !== false
    || estado.expediente_ref !== expediente.expediente_ref
    || !Array.isArray(estado.cuadro.expedientes)) return null;
  const resumen = estado.cuadro.expedientes.find(({ expediente_ref: referencia }) => (
    referencia === expediente.expediente_ref
  ));
  if (resumen?.fase_clave !== "fiscalizacion"
    || resumen.version !== expediente.version) return null;
  // Es contexto del formulario; el servidor decide vigencia y permisos al enviar.
  return Object.freeze({
    expediente_ref: expediente.expediente_ref,
    version_esperada: expediente.version,
  });
}

export function contextoFiscalizacionDesdeEstado(estado) {
  if (estado?.vista !== "expediente" || estado.expediente === null
    || estado.cuadro === null || !Array.isArray(estado.cuadro.expedientes)) return null;
  const resumen = estado.cuadro.expedientes.find(({ expediente_ref: referencia }) => (
    referencia === estado.expediente.expediente_ref
  ));
  if (resumen?.version !== estado.expediente.version) return null;
  const esInformeInicial = resumen.fase_clave === "informe_juridico";
  const ultimoHito = estado.expediente.historial?.at?.(-1);
  const esSubsanacionAutorizada = resumen.fase_clave === "subsanacion_unidad"
    && resumen.estado_clave === "incidencia"
    && ultimoHito?.accion_clave === "contratacion_temporal.subsanacion_reparos.registrar"
    && ultimoHito.version_expediente === estado.expediente.version;
  if (!esInformeInicial && !esSubsanacionAutorizada) return null;
  const informe = estado.expediente.cabecera?.find(
    ({ clave }) => clave === "informe_ref",
  )?.valor;
  return Object.freeze({
    expediente_ref: estado.expediente.expediente_ref,
    version_esperada: estado.expediente.version,
    fase_clave: esSubsanacionAutorizada ? "subsanacion_unidad" : resumen.fase_clave,
    informe_ref: !esSubsanacionAutorizada
      && typeof informe === "string" && PATRON_REFERENCIA.test(informe)
      ? informe : "",
  });
}

export function asignacionConfirmadaEnDetalle(expediente) {
  return Array.isArray(expediente?.cabecera) && expediente.cabecera.some(
    ({ clave, valor }) => clave === "unidad"
      && typeof valor === "string" && PATRON_REFERENCIA.test(valor),
  );
}

function renderizarFirmaPendiente(expediente, t) {
  const informe = expediente?.historial?.at?.(-1);
  if (!(informe?.accion_clave === "contratacion_temporal.informe_juridico.generar"
      || informe?.accion_clave === "registrar_informe_juridico")) return "";
  return `<section class="ct-exp-firma-pendiente" role="note" aria-labelledby="ct-exp-firma-pendiente-titulo">
    <p class="sobrelinea">${escaparHTML(t("firma_pendiente_sobrelinea"))}</p>
    <h3 id="ct-exp-firma-pendiente-titulo">${escaparHTML(t("firma_pendiente_titulo"))}</h3>
    <p>${escaparHTML(t("firma_pendiente_estado"))}</p>
    <dl><div><dt>${escaparHTML(t("firma_pendiente_documento"))}</dt><dd>${escaparHTML(informe.accion)}</dd></div>
      <div><dt>${escaparHTML(t("firma_pendiente_destino"))}</dt><dd>${escaparHTML(t("firma_pendiente_destino_valor"))}</dd></div></dl>
    <p>${escaparHTML(t("firma_pendiente_limite"))}</p>
  </section>`;
}

// La fase y el reparo proyectado solo acotan el contenedor. La disponibilidad
// efectiva llega como dependencia de composición y el servidor la revalida al
// registrar; la vista nunca la deduce de este estado.
export function contextoSubsanacionDesdeEstado(estado) {
  if (estado?.vista !== "expediente" || estado.carga !== "listo"
    || estado.expediente?.demostracion !== false || estado.cuadro?.demostracion !== false
    || !Array.isArray(estado.cuadro?.expedientes)) return null;
  const resumen = estado.cuadro.expedientes.find(({ expediente_ref: referencia }) => (
    referencia === estado.expediente.expediente_ref
  ));
  if (resumen?.fase_clave !== "subsanacion_unidad" || resumen.estado_clave !== "incidencia"
    || resumen.version !== estado.expediente.version) return null;
  return Object.freeze({ expediente_ref: estado.expediente.expediente_ref,
    version_esperada: estado.expediente.version });
}

export function renderizarReciboAsignacionConfirmada(recibo, contextoInforme, t, locale, zonaHoraria) {
  if (recibo?.expediente_ref !== contextoInforme?.expediente_ref
    || recibo?.version_resultante !== contextoInforme?.version_esperada
    || typeof recibo.recibo_ref !== "string" || typeof recibo.confirmada_en !== "string") return "";
  const fecha = new Date(recibo.confirmada_en);
  if (!Number.isFinite(fecha.getTime())) return "";
  const formateador = new Intl.DateTimeFormat(locale, {
    dateStyle: "long", timeStyle: "medium", timeZone: zonaHoraria,
  });
  return `<section class="ct-recibo" data-ct-asignacion-confirmada role="status"
    aria-live="polite" aria-atomic="true" tabindex="-1">
    <h3>${escaparHTML(t("asignacion_confirmada_titulo"))}</h3>
    <p>${escaparHTML(t("asignacion_confirmada_descripcion"))}</p>
    <dl><div><dt>${escaparHTML(t("asignacion_confirmada_recibo"))}</dt><dd><code>${escaparHTML(recibo.recibo_ref)}</code></dd></div>
    <div><dt>${escaparHTML(t("asignacion_confirmada_version"))}</dt><dd>${recibo.version_resultante}</dd></div>
    <div><dt>${escaparHTML(t("asignacion_confirmada_fecha"))}</dt><dd><time datetime="${escaparHTML(recibo.confirmada_en)}">${escaparHTML(formateador.format(fecha))}</time></dd></div></dl>
  </section>`;
}

export function renderizarReciboFiscalizacionConfirmada(recibo, expediente, t, locale, zonaHoraria, mensajes) {
  if (!recibo || !expediente) return "";
  if (recibo.expediente_ref !== expediente?.expediente_ref
    || recibo?.version_resultante !== expediente?.version
    || !["favorable", "favorable_con_observaciones", "desfavorable"].includes(recibo.resultado)
    || typeof recibo.recibo_ref !== "string" || typeof recibo.registrada_en !== "string") return "";
  const fecha = new Date(recibo.registrada_en);
  if (!Number.isFinite(fecha.getTime())) return "";
  const formateador = new Intl.DateTimeFormat(locale, {
    dateStyle: "long", timeStyle: "medium", timeZone: zonaHoraria,
  });
  const tFiscalizacion = crearTraductorContratacionTemporal(mensajes);
  return `<section class="ct-recibo" data-ct-fiscalizacion-confirmada role="status"
    aria-live="polite" aria-atomic="true" tabindex="-1">
    <h3>${escaparHTML(t("fiscalizacion_confirmada_titulo"))}</h3>
    <p>${escaparHTML(t("fiscalizacion_confirmada_descripcion"))}</p>
    <dl><div><dt>${escaparHTML(t("fiscalizacion_confirmada_resultado"))}</dt><dd>${escaparHTML(
      tFiscalizacion(`fiscalizacion_resultado_${recibo.resultado}`),
    )}</dd></div>
    <div><dt>${escaparHTML(t("fiscalizacion_confirmada_recibo"))}</dt><dd><code>${escaparHTML(recibo.recibo_ref)}</code></dd></div>
    <div><dt>${escaparHTML(t("fiscalizacion_confirmada_version"))}</dt><dd>${recibo.version_resultante}</dd></div>
    <div><dt>${escaparHTML(t("fiscalizacion_confirmada_fecha"))}</dt><dd><time datetime="${escaparHTML(recibo.registrada_en)}">${escaparHTML(formateador.format(fecha))}</time></dd></div></dl>
  </section>`;
}

export function contextoInformeJuridicoDesdeEstado(estado) {
  if (estado?.vista !== "expediente" || estado.expediente === null
    || estado.cuadro === null || !Array.isArray(estado.cuadro.expedientes)) return null;
  const resumen = estado.cuadro.expedientes.find(({ expediente_ref: referencia }) => (
    referencia === estado.expediente.expediente_ref
  ));
  if (resumen?.fase_clave !== "asignacion_unidad"
    || resumen.estado_clave !== "en_curso"
    || resumen.version !== estado.expediente.version
    || !asignacionConfirmadaEnDetalle(estado.expediente)) return null;
  return Object.freeze({
    expediente_ref: estado.expediente.expediente_ref,
    version_esperada: estado.expediente.version,
  });
}

export function contextoAsignacionDesdeEstado(estado) {
  if (estado?.vista !== "expediente" || estado.carga !== "listo" || estado.expediente == null
    || estado.cuadro == null || !Array.isArray(estado.cuadro.expedientes)) return null;
  const resumen = estado.cuadro.expedientes.find(({ expediente_ref: referencia }) => (
    referencia === estado.expediente.expediente_ref
  ));
  if (resumen?.fase_clave !== "asignacion_unidad" || resumen.estado_clave !== "en_curso"
    || resumen.version !== estado.expediente.version
    || asignacionConfirmadaEnDetalle(estado.expediente)) return null;
  return Object.freeze({
    expediente_ref: estado.expediente.expediente_ref,
    version_esperada: estado.expediente.version,
  });
}

export function contextoCoberturaDesdeEstado(estado) {
  if (estado?.vista !== "expediente" || estado.carga !== "listo" || estado.expediente == null
    || estado.cuadro?.demostracion !== false || estado.expediente.demostracion !== false
    || !Array.isArray(estado.cuadro.expedientes) || !Array.isArray(estado.expediente.cabecera)) return null;
  const resumen = estado.cuadro.expedientes.find(({ expediente_ref: referencia }) => (
    referencia === estado.expediente.expediente_ref
  ));
  const analisisConfirmado = estado.expediente.cabecera.some(({ clave, valor }) => (
    clave === "resultado_rc" && typeof valor === "string" && valor !== ""
  ));
  const coberturaOAsignacionExistente = estado.expediente.cabecera.some(({ clave }) => (
    clave === "via_cobertura" || clave === "decision_gobernada" || clave === "unidad"
  ));
  if (resumen?.fase_clave !== "solicitud" || resumen.estado_clave !== "en_curso"
    || resumen.version !== estado.expediente.version || resumen.version < 2
    || !analisisConfirmado || coberturaOAsignacionExistente) return null;
  return Object.freeze({
    expediente_ref: estado.expediente.expediente_ref,
    version_esperada: estado.expediente.version,
  });
}

export function contextoRectificacionAnalisisDesdeEstado(estado) {
  if (estado?.vista !== "expediente" || estado.carga !== "listo" || estado.expediente == null
    || estado.cuadro?.demostracion !== false || !Array.isArray(estado.cuadro.expedientes)
    || estado.expediente.demostracion !== false || estado.expediente.version < 2
    || !Array.isArray(estado.expediente.cabecera)) return null;
  const resumen = estado.cuadro.expedientes.find(({ expediente_ref: referencia }) => (
    referencia === estado.expediente.expediente_ref
  ));
  const analisisConfirmado = estado.expediente.cabecera.some(({ clave, valor }) => (
    clave === "resultado_rc" && typeof valor === "string" && valor !== ""
  ));
  if (resumen?.fase_clave !== "solicitud" || resumen.estado_clave !== "en_curso"
    || resumen.version !== estado.expediente.version || !analisisConfirmado) {
    return null;
  }
  const claves = new Set(estado.expediente.cabecera.map(({ clave }) => clave));
  if (!claves.has("resultado_rc") || claves.has("via_cobertura")
    || claves.has("decision_gobernada") || claves.has("unidad")) return null;
  return Object.freeze({
    operacion: "rectificar",
    expediente_ref: estado.expediente.expediente_ref,
    version_esperada: estado.expediente.version,
  });
}

export function renderizarModuloContratacionTemporal(estado, {
  mensajes = {},
  locale = "es-ES",
  zonaHoraria = "Europe/Madrid",
  altaDisponible = false,
  analisisDisponible = false,
  coberturaDisponible = false,
  asignacionDisponible = false,
  informeJuridicoDisponible = false,
  fiscalizacionDisponible = false,
  subsanacionDisponible = false,
  reciboSubsanacionConfirmado = null,
  reciboFiscalizacionConfirmado = null,
  llamamientoDisponible = false,
  resolucionFormalizacionDisponible = false,
  incorporacionEjercicioDisponible = false,
  reciboAsignacionConfirmado = null,
  catalogoDisponible = true,
} = {}) {
  const t = crearTraductorExpedientesContratacion(mensajes);
  let contenido;
  const errorPaginadoRecuperable = estado.vista === "cuadro" && estado.carga === "error"
    && estado.cuadro?.paginacion && estado.paginacion_requiere_reinicio === true;
  if (["cargando", "error", "denegado"].includes(estado.carga)
    && !errorPaginadoRecuperable
    && estado.vista !== "alta") {
    contenido = renderizarEstadoCarga(estado, t);
  } else if (estado.vista === "alta") {
    contenido = renderizarAlta(
      t,
      altaDisponible,
      analisisDisponible,
      coberturaDisponible,
      asignacionDisponible,
      informeJuridicoDisponible,
      fiscalizacionDisponible,
      catalogoDisponible,
    );
  } else if (estado.vista === "expediente") {
    const detalle = renderizarExpediente(
      estado,
      t,
      locale,
      zonaHoraria,
      analisisDisponible,
    );
    const contextoInforme = informeJuridicoDisponible
      ? contextoInformeJuridicoDesdeEstado(estado)
      : null;
    const reciboAsignacionPendiente = Boolean(reciboAsignacionConfirmado && estado.expediente)
      && reciboAsignacionConfirmado.expediente_ref
      === estado.expediente?.expediente_ref
      && Number.isSafeInteger(reciboAsignacionConfirmado.version_resultante)
      && reciboAsignacionConfirmado.version_resultante > estado.expediente?.version;
    const contextoReciboAsignacion = contextoInforme ?? (reciboAsignacionPendiente
      ? Object.freeze({
        expediente_ref: estado.expediente.expediente_ref,
        version_esperada: reciboAsignacionConfirmado.version_resultante,
      })
      : null);
    const reciboAsignacion = contextoReciboAsignacion === null ? ""
      : renderizarReciboAsignacionConfirmada(
        reciboAsignacionConfirmado, contextoReciboAsignacion, t, locale, zonaHoraria,
      );
    const reciboFiscalizacion = renderizarReciboFiscalizacionConfirmada(
      reciboFiscalizacionConfirmado, estado.expediente, t, locale, zonaHoraria, mensajes,
    );
    const contextoAsignacion = asignacionDisponible && !reciboAsignacionPendiente
      ? contextoAsignacionDesdeEstado(estado)
      : null;
    const contextoCobertura = coberturaDisponible
      ? contextoCoberturaDesdeEstado(estado)
      : null;
    const contextoRectificacion = analisisDisponible
      ? contextoRectificacionAnalisisDesdeEstado(estado)
      : null;
    const contextoFiscalizacion = fiscalizacionDisponible
      ? contextoFiscalizacionDesdeEstado(estado)
      : null;
    const contextoSubsanacion = subsanacionDisponible
      ? contextoSubsanacionDesdeEstado(estado)
      : null;
    const ultimoHito = estado.expediente?.historial?.at(-1);
    const conservarReciboSubsanacion = contextoSubsanacion !== null
      && reciboSubsanacionConfirmado?.recibo?.expediente_ref === contextoSubsanacion.expediente_ref
      && reciboSubsanacionConfirmado.recibo.version_resultante === contextoSubsanacion.version_esperada;
    const subsanacionRegistrada = contextoSubsanacion !== null && !conservarReciboSubsanacion
      && ultimoHito?.accion_clave === "contratacion_temporal.subsanacion_reparos.registrar"
      && ultimoHito.version_expediente === contextoSubsanacion.version_esperada
      && ultimoHito.secuencia === contextoSubsanacion.version_esperada;
    contenido = `${detalle}${renderizarFirmaPendiente(estado.expediente, t)}${reciboAsignacion}${reciboFiscalizacion}${contextoRectificacion
      ? '<div data-ct-exp-rectificacion></div>'
      : ""}${contextoCobertura
      ? '<div data-ct-exp-cobertura></div>'
      : ""}${contextoAsignacion
      ? '<div data-ct-exp-asignacion></div>'
      : ""}${contextoInforme
      ? '<div data-ct-exp-informe-juridico></div>'
      : ""}${fiscalizacionDisponible && (contextoInforme || contextoFiscalizacion)
      ? '<div data-ct-exp-fiscalizacion></div>'
      : ""}${subsanacionRegistrada
      ? `<p class="ct-exp-mensaje ct-tono-informacion" role="status">${escaparHTML(t("subsanacion_registrada_pendiente_fiscalizacion"))}</p>`
      : contextoSubsanacion ? '<div data-ct-exp-subsanacion></div>' : ""}${resolucionFormalizacionDisponible ? '<div data-ct-exp-resolucion-formalizacion></div>' : ""}
      ${incorporacionEjercicioDisponible ? '<div data-ct-exp-incorporacion-ejercicio></div>' : ""}`;
  } else if (estado.vista === "documentos") {
    contenido = renderizarDocumentos(estado, t);
  } else if (estado.vista === "auditoria") {
    contenido = renderizarAuditoria(estado, t);
  } else if (estado.vista === "estadisticas") {
    contenido = '<div data-ct-exp-estadisticas></div>';
  } else {
    contenido = renderizarCuadro(estado, t);
  }
  return `<section class="ct-expedientes" data-modulo="contratacion-temporal"
    aria-labelledby="ct-exp-titulo">
    ${renderizarCabeceraModulo(estado, t)
    .replace("<h2>", '<h2 id="ct-exp-titulo">')}
    ${renderizarNavegacion(estado, t)}
    <div class="ct-exp-mensaje ct-tono-${escaparHTML(estado.tipo_mensaje || "info")}"
      data-ct-exp-mensaje role="${estado.tipo_mensaje === "error" ? "alert" : "status"}"
      aria-live="polite">${escaparHTML(estado.mensaje_clave ? t(estado.mensaje_clave) : "")}</div>
    <div class="ct-exp-contenido">${contenido}${llamamientoDisponible
      && estado.vista === "expediente" && estado.carga === "listo"
      && estado.expediente !== null
      ? '<div data-ct-exp-llamamiento></div>' : ""}</div>
  </section>`;
}
