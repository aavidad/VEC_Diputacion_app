import { crearTraductorDietas, MENSAJES_DIETAS } from "./i18n.js?v=20260929-i18n-dietas-v1";

import { MENSAJES_CIRCUITO_DIETAS } from "./i18n-circuito.js?v=20260929-i18n-dietas-v1";
export { MENSAJES_CIRCUITO_DIETAS } from "./i18n-circuito.js?v=20260929-i18n-dietas-v1";
import { LOCALIZACION_PORTAL, ZONA_HORARIA_PORTAL } from "../../portal-i18n.js?v=20261001-ct-a-i18n-v1";
import { crearTraductorOtrosGastosDietas } from "./i18n-otros-gastos.js?v=20260929-i18n-dietas-v1";
import { describirOtroGasto } from "./formulario-otros-gastos.js?v=20260929-i18n-dietas-v1";
import { recortarBordes } from "./texto-dietas.js?v=20260925-d5d6-v1";

const codificador = new TextEncoder();
const errorMotivo = (motivo, decision) => {
  const bytes = codificador.encode(motivo).byteLength;
  if (bytes > 600 || /[\x00-\x1f\x7f]/u.test(motivo)) return "circuito_motivo_invalido";
  return decision === "devolver" && bytes < 3 ? "circuito_motivo_devolucion_incompleto" : "";
};

const ETAPAS = Object.freeze(["revision", "autorizacion", "liquidacion", "fiscalizacion"]);
const TONO_ESTADO = Object.freeze({ enviado_pendiente_revision: "info", pendiente_autorizacion: "info", pendiente_liquidacion: "violeta", pendiente_fiscalizacion: "violeta", fiscalizada: "exito", devuelta: "peligro" });
const nodo = (documento, etiqueta, texto = "") => { const resultado = documento.createElement(etiqueta); if (texto) resultado.textContent = texto; return resultado; };
const montada = (contenedor, raiz) => contenedor.querySelector?.("[data-dietas-bandeja-circuito]") === raiz;
// Las fechas civiles (AAAA-MM-DD) no tienen zona: se leen y pintan en UTC para no correr un día.
const fecha = (valor) => new Intl.DateTimeFormat(LOCALIZACION_PORTAL, { dateStyle: "medium", timeZone: "UTC" }).format(new Date(`${valor}T00:00:00Z`));
const instante = (valor) => new Intl.DateTimeFormat(LOCALIZACION_PORTAL, { dateStyle: "medium", timeStyle: "short", timeZone: ZONA_HORARIA_PORTAL }).format(new Date(valor));
const euros = (centimos) => new Intl.NumberFormat(LOCALIZACION_PORTAL, { style: "currency", currency: "EUR" }).format((Number(centimos) || 0) / 100);
const kilometros = (valor) => new Intl.NumberFormat(LOCALIZACION_PORTAL, { maximumFractionDigits: 1 }).format(Number(valor) || 0);
function crearTraductorCircuito(traducir) {
  return (clave, variables = {}) => {
    if (Object.hasOwn(MENSAJES_CIRCUITO_DIETAS, clave)) {
      try { const resultado = traducir(clave, variables); if (typeof resultado === "string" && resultado !== clave) return resultado; } catch { /* El montaje puede recibir un catálogo anterior al circuito. */ }
      return MENSAJES_CIRCUITO_DIETAS[clave].replace(/\{([a-z_]+)\}/gu, (_todo, nombre) => String(variables[nombre] ?? ""));
    }
    return traducir(clave, variables);
  };
}
function claveNueva(generar) {
  const valor = generar?.();
  if (typeof valor !== "string" || !/^[A-Za-z0-9_-]{16,128}$/u.test(valor)) throw new TypeError("clave de idempotencia no disponible");
  return valor;
}
function textoError(error, t, clavePorDefecto) {
  if (error?.codigo === "competencia_sin_fuente") return t("circuito_sin_fuente");
  if (error?.codigo === "acceso_denegado") return t("circuito_denegado");
  if (error?.codigo === "autenticacion_requerida") return t("circuito_autenticacion");
  if (error?.codigo === "conflicto_estado") return t("circuito_conflicto");
  if (error?.codigo === "no_encontrada") return t("circuito_no_encontrada");
  return t(clavePorDefecto);
}

/** Describe una línea del documento con textos de negocio, sin códigos internos. */
function describirLinea(linea, t) {
  if (linea.tipo === "dieta")
    return t(linea.concepto === "manutencion" ? "circuito_concepto_manutencion" : "circuito_concepto_alojamiento", { fecha: fecha(linea.fecha) });
  if (linea.tipo === "kilometraje") {
    const ruta = t("circuito_concepto_ruta", { indice: linea.ruta_indice, kilometros: kilometros(linea.kilometros) });
    return linea.motivo_ajuste ? `${ruta} (${t("circuito_concepto_ajuste", { motivo: linea.motivo_ajuste })})` : ruta;
  }
  return describirOtroGasto(linea, t, fecha);
}

/**
 * Bandeja de una etapa del circuito de Dietas. Con `control` añade la búsqueda
 * por etapa y fechas («Control de documentos»). Cada acción consulta el
 * servidor; el recibo solo se muestra tras una respuesta válida y, si la
 * fuente de competencia no existe, la bandeja lo dice sin ofrecer acciones.
 */
export function montarVistaBandejaCircuitoDietas(contenedor, {
  cliente,
  traducir = crearTraductorDietas(MENSAJES_DIETAS),
  anunciar = () => {},
  generarClaveIdempotencia = () => globalThis.crypto?.randomUUID?.(),
  registrarDesmontar,
  etapaInicial = "revision",
  etapas = [etapaInicial],
  control = false,
} = {}) {
  if (!contenedor?.ownerDocument || !cliente || typeof cliente.listar !== "function" || typeof cliente.decidir !== "function"
    || !Array.isArray(etapas) || !etapas.length || etapas.some((etapa) => !ETAPAS.includes(etapa)) || !etapas.includes(etapaInicial))
    throw new TypeError("bandeja del circuito de Dietas no disponible");
  const documento = contenedor.ownerDocument; const t = crearTraductorOtrosGastosDietas(crearTraductorCircuito(traducir));
  const raiz = nodo(documento, "section"); raiz.dataset.dietasBandejaCircuito = ""; raiz.className = "panel dietas-bandeja-circuito";
  const cabecera = nodo(documento, "header"); cabecera.className = "cabecera-panel";
  const titulo = nodo(documento, "h2"); cabecera.append(titulo);
  const filtros = nodo(documento, "form"); filtros.className = "dietas-bandeja-circuito-filtros"; filtros.dataset.dietasCircuitoFiltros = ""; filtros.hidden = !control;
  const selectorEtapa = nodo(documento, "select"); selectorEtapa.name = "etapa"; selectorEtapa.dataset.dietasCircuitoEtapa = "";
  etapas.forEach((valor) => { const opcion = nodo(documento, "option", t(`circuito_etapa_${valor}`)); opcion.value = valor; opcion.selected = valor === etapaInicial; selectorEtapa.append(opcion); }); selectorEtapa.value = etapaInicial;
  const desde = nodo(documento, "input"); desde.type = "date"; desde.name = "fecha_desde"; desde.dataset.dietasCircuitoDesde = "";
  const hasta = nodo(documento, "input"); hasta.type = "date"; hasta.name = "fecha_hasta"; hasta.dataset.dietasCircuitoHasta = "";
  const buscar = nodo(documento, "button", t("circuito_buscar")); buscar.type = "submit"; buscar.className = "boton-secundario";
  const campo = (clave, control) => { const etiqueta = nodo(documento, "label", t(clave)); etiqueta.append(control); return etiqueta; };
  filtros.append(campo("circuito_etapa", selectorEtapa), campo("circuito_fecha_desde", desde), campo("circuito_fecha_hasta", hasta), buscar);
  const estado = nodo(documento, "p"); estado.dataset.dietasCircuitoEstado = ""; estado.setAttribute("role", "status"); estado.setAttribute("aria-live", "polite"); estado.setAttribute("tabindex", "-1"); estado.className = "dietas-bandeja-circuito-estado";
  const contenido = nodo(documento, "div"); contenido.className = "cuerpo-panel"; contenido.dataset.dietasCircuitoContenido = "";
  const piePagina = nodo(documento, "div"); piePagina.className = "paginacion-marco";
  const cuentaPagina = nodo(documento, "p"); cuentaPagina.dataset.dietasCircuitoPagina = ""; cuentaPagina.setAttribute("role", "status"); cuentaPagina.setAttribute("aria-live", "polite"); cuentaPagina.setAttribute("aria-atomic", "true"); cuentaPagina.setAttribute("tabindex", "-1");
  const navegacion = nodo(documento, "nav"); navegacion.className = "paginacion-marco__paginas"; navegacion.setAttribute("aria-label", t("circuito_paginacion"));
  piePagina.append(cuentaPagina, navegacion);
  const detalle = nodo(documento, "div"); detalle.className = "dietas-circuito-detalle"; detalle.dataset.dietasCircuitoDetalle = ""; detalle.hidden = true;
  raiz.append(cabecera, filtros, estado, contenido, piePagina, detalle); contenedor.append(raiz);

  let activa = true; let controladorLectura; let controladorDocumento; let controladorDecision; let generacion = 0;
  let consulta = { etapa: etapaInicial, limit: 20 }; let pagina = { items: [], competencia: "acreditada" };
  // Solo cursores de la consulta activa: las páginas anteriores se consultan de nuevo.
  let cursores = [undefined]; let indicePagina = 0; let generacionDocumento = 0;
  let pendiente = null; let abierto = null; let mensaje = ""; let nivel = ""; let nivelDetalle = "";
  const publicar = (texto) => { if (activa) anunciar(texto); };
  const filtrada = () => Boolean(consulta.fecha_desde || consulta.fecha_hasta);
  const puedeRestaurarFoco = (disparador) => {
    const foco = documento.activeElement;
    return !foco || foco === documento.body || foco === disparador
      || [contenido, navegacion, detalle].some((region) => region.contains?.(foco));
  };
  const resumenPagina = () => t(pagina.items.length === 1 ? "circuito_pagina_una_fila" : "circuito_pagina_filas", {
    pagina: new Intl.NumberFormat(LOCALIZACION_PORTAL).format(indicePagina + 1),
    filas: new Intl.NumberFormat(LOCALIZACION_PORTAL).format(pagina.items.length),
  });

  function pintarPaginacion() {
    piePagina.hidden = Boolean(abierto) || nivel === "cargando" || pagina.competencia === "sin_fuente";
    cuentaPagina.textContent = nivel === "error" ? "" : resumenPagina();
    navegacion.replaceChildren();
    if (piePagina.hidden) return;
    const boton = (clave, atributo) => { const control = nodo(documento, "button", t(clave)); control.type = "button"; control.className = "boton-secundario"; control.dataset[atributo] = ""; navegacion.append(control); };
    if (indicePagina > 0) { boton("circuito_primera_pagina", "dietasCircuitoPrimera"); boton("circuito_anterior", "dietasCircuitoAnterior"); }
    if (pagina.siguiente_cursor) boton("circuito_siguiente_pagina", "dietasCircuitoSiguiente");
  }

  function pintarLista() {
    titulo.textContent = control ? t("circuito_control") : t(`circuito_bandeja_${consulta.etapa}`);
    contenido.replaceChildren(); contenido.hidden = Boolean(abierto);
    if (nivel === "cargando") { contenido.append(nodo(documento, "p", t("circuito_cargando"))); return; }
    if (nivel === "error" && !pendiente?.incierta && !pagina.items.length) {
      const reintentar = nodo(documento, "button", t("circuito_reintentar_consulta")); reintentar.type = "button"; reintentar.className = "boton-secundario"; reintentar.dataset.dietasCircuitoReintentar = ""; contenido.append(reintentar); return;
    }
    if (pagina.competencia === "sin_fuente") { const linea = nodo(documento, "p", t("circuito_sin_fuente")); linea.dataset.dietasCircuitoSinFuente = ""; linea.setAttribute("tabindex", "-1"); contenido.append(linea); return; }
    if (!pagina.items.length) { contenido.append(nodo(documento, "p", t(filtrada() ? "circuito_vacio_filtros" : "circuito_vacio"))); return; }
    const marco = nodo(documento, "div"); marco.className = "tabla-contenedor";
    const tabla = nodo(documento, "table"); tabla.className = "tabla-datos dietas-bandeja-circuito-tabla";
    const cabeza = nodo(documento, "thead"); const filaCabeza = nodo(documento, "tr");
    ["circuito_referencia", "circuito_periodo", "circuito_estado", "circuito_acciones"].forEach((clave) => { const th = nodo(documento, "th", t(clave)); th.setAttribute("scope", "col"); filaCabeza.append(th); }); cabeza.append(filaCabeza);
    const cuerpo = nodo(documento, "tbody");
    pagina.items.forEach((item) => {
      const fila = nodo(documento, "tr"); fila.dataset.dietasCircuitoComision = item.referencia;
      const chip = nodo(documento, "span", t(`circuito_estado_${item.estado}`)); chip.className = `estado-chip ${TONO_ESTADO[item.estado] || "neutro"}`;
      const celdaEstado = nodo(documento, "td"); celdaEstado.append(chip);
      const abrir = nodo(documento, "button", t("circuito_abrir")); abrir.type = "button"; abrir.className = "boton-secundario"; abrir.dataset.dietasCircuitoAbrir = item.referencia;
      abrir.setAttribute("aria-label", t("circuito_abrir_etiqueta", { fecha: fecha(item.fecha_inicio) }));
      const acciones = nodo(documento, "td"); acciones.append(abrir);
      fila.append(nodo(documento, "td", t("circuito_comision_fecha", { fecha: fecha(item.fecha_inicio) })), nodo(documento, "td", t("circuito_periodo_valor", { inicio: fecha(item.fecha_inicio), fin: fecha(item.fecha_fin) })), celdaEstado, acciones);
      cuerpo.append(fila);
    });
    tabla.append(cabeza, cuerpo); marco.append(tabla); contenido.append(marco);
  }

  function pintarDetalle() {
    detalle.replaceChildren(); detalle.hidden = !abierto;
    if (!abierto) return;
    const volver = nodo(documento, "button", t("circuito_volver")); volver.disabled = Boolean(pendiente); volver.type = "button"; volver.className = "boton-secundario"; volver.dataset.dietasCircuitoVolver = "";
    if (nivelDetalle === "cargando") { detalle.append(volver, nodo(documento, "p", t("circuito_documento_cargando"))); return; }
    if (!abierto.documento) { detalle.append(volver, nodo(documento, "p", t("circuito_documento_error"))); return; }
    const d = abierto.documento; const lineas = Array.isArray(d.documento?.lineas) ? d.documento.lineas : [];
    const encabezado = nodo(documento, "div"); encabezado.className = "dietas-circuito-detalle-cabecera";
    encabezado.append(nodo(documento, "h3", t("circuito_documento_titulo", { numero: d.numero_documento })), volver);
    const resumen = nodo(documento, "dl"); resumen.className = "dietas-circuito-resumen";
    // Un reenvío muestra la devolución anterior (etapa, fecha y motivo), nunca quién la hizo.
    const reenvio = d.devolucion ? [["circuito_documento_reenvio", t("circuito_documento_reenvio_valor", { etapa: t(`circuito_etapa_${d.devolucion.etapa}`), fecha: instante(d.devolucion.devuelta_en) })],
      ["circuito_documento_motivo_devolucion", d.devolucion.motivo]] : [];
    [["circuito_documento_periodo", t("circuito_documento_periodo_valor", { inicio: fecha(d.fecha_inicio), hora_inicio: d.hora_inicio, fin: fecha(d.fecha_fin), hora_fin: d.hora_fin })],
      ["circuito_documento_apertura", instante(d.fecha_apertura)], ["circuito_documento_motivo", d.motivo], ...reenvio].forEach(([clave, valor]) => {
      const grupo = nodo(documento, "div"); grupo.append(nodo(documento, "dt", t(clave)), nodo(documento, "dd", valor));
      if (reenvio.some(([otra]) => otra === clave)) grupo.dataset.dietasCircuitoReenvio = clave === "circuito_documento_reenvio" ? "etapa" : "motivo";
      resumen.append(grupo);
    });
    const marco = nodo(documento, "div"); marco.className = "tabla-contenedor";
    const tabla = nodo(documento, "table"); tabla.className = "tabla-datos dietas-circuito-lineas";
    const leyenda = nodo(documento, "caption", t("circuito_lineas")); tabla.append(leyenda);
    const cabeza = nodo(documento, "thead"); const filaCabeza = nodo(documento, "tr");
    ["circuito_linea_apartado", "circuito_linea_concepto", "circuito_linea_justificante", "circuito_linea_importe"].forEach((clave) => { const th = nodo(documento, "th", t(clave)); th.setAttribute("scope", "col"); if (clave === "circuito_linea_importe") th.className = "columna-numero"; filaCabeza.append(th); });
    cabeza.append(filaCabeza);
    const cuerpo = nodo(documento, "tbody");
    if (!lineas.length) { const fila = nodo(documento, "tr"); const celda = nodo(documento, "td", t("circuito_sin_lineas")); celda.setAttribute("colspan", "4"); fila.append(celda); cuerpo.append(fila); }
    lineas.forEach((linea) => {
      const fila = nodo(documento, "tr"); fila.dataset.dietasCircuitoLinea = String(linea.tipo ?? "");
      const justificante = ["otro_medio", "otro_gasto"].includes(linea.tipo)
        ? (linea.justificante_ref ? t("circuito_justificante_referencia", { referencia: linea.justificante_ref }) : t("circuito_sin_justificante")) : "";
      const importe = nodo(documento, "td", euros(linea.importe_centimos)); importe.className = "columna-numero";
      fila.append(nodo(documento, "td", t(`circuito_apartado_${linea.tipo}`)), nodo(documento, "td", describirLinea(linea, t)), nodo(documento, "td", justificante), importe);
      cuerpo.append(fila);
    });
    const pie = nodo(documento, "tfoot");
    const doc = d.documento || {};
    [["circuito_total_dietas", Number(doc.manutencion_centimos || 0) + Number(doc.alojamiento_tope_centimos || 0)], ["circuito_total_kilometraje", doc.kilometraje_centimos], ["circuito_total_otros", doc.otros_centimos], ["circuito_total", doc.total_orientativo_centimos]].forEach(([clave, valor]) => {
      const fila = nodo(documento, "tr"); if (clave === "circuito_total") fila.dataset.dietasCircuitoTotal = "";
      const th = nodo(documento, "th", t(clave)); th.setAttribute("scope", "row"); th.setAttribute("colspan", "3");
      const celda = nodo(documento, "td", euros(valor)); celda.className = "columna-numero"; fila.append(th, celda); pie.append(fila);
    });
    tabla.append(cabeza, cuerpo, pie); marco.append(tabla);
    const decision = nodo(documento, "div"); decision.className = "dietas-circuito-decision";
    const motivo = nodo(documento, "textarea"); motivo.maxLength = 600; motivo.rows = 3; motivo.dataset.dietasCircuitoMotivo = ""; motivo.id = `motivo-${abierto.referencia}`;
    motivo.value = abierto.motivo || "";
    motivo.setAttribute("aria-invalid", String(Boolean(abierto.errorMotivo)));
    const errorCampo = abierto.errorMotivo ? nodo(documento, "p", t(abierto.errorMotivo)) : null;
    if (errorCampo) {
      errorCampo.id = `${motivo.id}-error`; errorCampo.dataset.dietasCircuitoMotivoError = "";
      motivo.setAttribute("aria-describedby", errorCampo.id);
    }
    const etiqueta = nodo(documento, "label", t("circuito_motivo")); etiqueta.setAttribute("for", motivo.id);
    const botones = nodo(documento, "div"); botones.className = "dietas-circuito-botones";
    const aprobar = nodo(documento, "button", t(`circuito_aprobar_${consulta.etapa}`)); aprobar.type = "button"; aprobar.className = "boton-primario"; aprobar.dataset.dietasCircuitoDecision = "aprobar";
    const devolver = nodo(documento, "button", t("circuito_devolver")); devolver.type = "button"; devolver.className = "boton-peligro"; devolver.dataset.dietasCircuitoDecision = "devolver";
    const bloqueado = Boolean(controladorDecision || pendiente); aprobar.disabled = bloqueado; devolver.disabled = bloqueado; motivo.readOnly = bloqueado;
    botones.append(aprobar, devolver);
    if (pendiente?.incierta) { const reintento = nodo(documento, "button", t("circuito_reintentar_decision")); reintento.disabled = Boolean(controladorDecision); reintento.type = "button"; reintento.className = "boton-secundario"; reintento.dataset.dietasCircuitoReintento = ""; botones.append(reintento); }
    decision.append(etiqueta, motivo);
    if (errorCampo) decision.append(errorCampo);
    decision.append(botones);
    detalle.append(encabezado, resumen, marco, decision);
  }

  function pintar() {
    if (!activa || !montada(contenedor, raiz)) return;
    estado.textContent = mensaje; estado.dataset.nivel = nivel;
    pintarLista(); pintarPaginacion(); pintarDetalle(); pintarBloqueo();
  }

  function pintarBloqueo() {
    const bloqueado = Boolean(controladorDecision || pendiente);
    [selectorEtapa, desde, hasta, buscar].forEach((campo) => { campo.disabled = bloqueado; });
    detalle.setAttribute("aria-busy", String(Boolean(controladorDecision)));
  }

  async function cargar(indice = indicePagina, enfocar = false) {
    if (!activa || pendiente || !Number.isInteger(indice) || indice < 0 || indice >= cursores.length) return;
    const disparador = documento.activeElement;
    controladorLectura?.abort(); controladorLectura = new AbortController(); const actual = ++generacion;
    controladorDocumento?.abort(); ++generacionDocumento; abierto = null;
    const nuevaConsulta = { ...consulta, ...(cursores[indice] ? { cursor: cursores[indice] } : {}) };
    indicePagina = indice; pagina = { items: [], competencia: "acreditada" };
    mensaje = t("circuito_cargando"); nivel = "cargando"; contenido.setAttribute("aria-busy", "true"); pintar();
    try {
      const recibida = await cliente.listar(nuevaConsulta, { signal: controladorLectura.signal });
      if (!activa || actual !== generacion) return;
      pagina = recibida;
      cursores = cursores.slice(0, indice + 1);
      if (pagina.competencia === "sin_fuente") { pagina = { items: [], competencia: "sin_fuente" }; cursores = [undefined]; indicePagina = 0; }
      else if (pagina.siguiente_cursor) cursores.push(pagina.siguiente_cursor);
      mensaje = ""; nivel = "";
    } catch (error) {
      if (!activa || actual !== generacion || error?.codigo === "operacion_abortada") return;
      mensaje = textoError(error, t, "circuito_error"); nivel = "error";
      if (error?.codigo === "competencia_sin_fuente") { pagina = { items: [], competencia: "sin_fuente" }; cursores = [undefined]; indicePagina = 0; nivel = ""; }
      publicar(mensaje);
    } finally {
      if (activa && actual === generacion) {
        const restaurarFoco = enfocar && puedeRestaurarFoco(disparador);
        contenido.setAttribute("aria-busy", "false"); pintar();
        if (restaurarFoco) (pagina.competencia === "sin_fuente" ? contenido.querySelector?.("[data-dietas-circuito-sin-fuente]") : nivel === "error" ? estado : cuentaPagina)?.focus?.();
      }
    }
  }

  async function abrir(referencia) {
    if (!activa || controladorDecision || pendiente) return;
    const item = pagina.items.find((entrada) => entrada.referencia === referencia);
    if (!item || pagina.competencia === "sin_fuente" || typeof cliente.documento !== "function") return;
    const disparador = documento.activeElement;
    controladorDocumento?.abort(); controladorDocumento = new AbortController();
    const actual = ++generacionDocumento;
    abierto = { referencia, version: item.version, documento: null }; nivelDetalle = "cargando"; mensaje = ""; pintar();
    detalle.querySelector?.("[data-dietas-circuito-volver]")?.focus?.();
    try {
      const documentoLeido = await cliente.documento(referencia, consulta.etapa, { signal: controladorDocumento.signal });
      if (!activa || actual !== generacionDocumento || abierto?.referencia !== referencia) return;
      abierto = { ...abierto, documento: documentoLeido, version: documentoLeido.version };
    } catch (error) {
      if (!activa || actual !== generacionDocumento || error?.codigo === "operacion_abortada") return;
      mensaje = textoError(error, t, "circuito_documento_error"); nivel = "error"; publicar(mensaje);
    } finally {
      if (activa && actual === generacionDocumento) {
        const restaurarFoco = puedeRestaurarFoco(disparador);
        nivelDetalle = ""; pintar();
        if (restaurarFoco) detalle.querySelector?.("[data-dietas-circuito-volver]")?.focus?.();
      }
    }
  }

  function cerrar() { if (pendiente) return; controladorDocumento?.abort(); ++generacionDocumento; const referencia = abierto?.referencia; abierto = null; pintar(); contenido.querySelector?.(`[data-dietas-circuito-abrir="${referencia}"]`)?.focus?.(); }

  async function decidir(reintentar, boton) {
    if (controladorDecision || !activa || !abierto?.documento) return;
    if (!reintentar && pendiente) return;
    const decision = reintentar ? pendiente?.entrada.decision : boton?.dataset?.dietasCircuitoDecision;
    if (!["aprobar", "devolver"].includes(decision)) return;
    const campoMotivo = detalle.querySelector?.("[data-dietas-circuito-motivo]");
    // Recorta los mismos blancos de borde que Go rechaza (también U+0085).
    const motivo = reintentar ? pendiente.entrada.motivo : recortarBordes(campoMotivo?.value);
    const claveErrorMotivo = reintentar ? "" : errorMotivo(motivo, decision);
    if (claveErrorMotivo) {
      mensaje = t(claveErrorMotivo); nivel = "error"; abierto = { ...abierto, motivo: campoMotivo?.value || "", errorMotivo: claveErrorMotivo }; pintar(); publicar(mensaje);
      detalle.querySelector?.("[data-dietas-circuito-motivo]")?.focus?.(); return;
    }
    if (!reintentar) pendiente = { referencia: abierto.referencia, entrada: Object.freeze({ etapa: consulta.etapa, decision, motivo,
      clave_idempotencia: claveNueva(generarClaveIdempotencia), version_esperada: abierto.version }), incierta: false };
    const disparador = documento.activeElement;
    abierto = { ...abierto, motivo, errorMotivo: "" };
    controladorDecision = new AbortController(); mensaje = t("circuito_procesando"); nivel = "cargando";
    pintarDetalle(); pintarBloqueo(); estado.textContent = mensaje; estado.dataset.nivel = nivel;
    if (puedeRestaurarFoco(disparador)) estado.focus?.();
    try {
      const resultado = await cliente.decidir(pendiente.referencia, pendiente.entrada, { signal: controladorDecision.signal });
      if (!activa) return;
      mensaje = t(resultado.recibo.repeticion ? "circuito_recibo_repetido" : "circuito_recibo", { referencia: resultado.recibo.referencia, version: resultado.recibo.version, fecha: instante(resultado.recibo.registrado_en) });
      nivel = "exito"; publicar(mensaje);
      pagina = { ...pagina, items: pagina.items.filter((entrada) => entrada.referencia !== pendiente.referencia) };
      pendiente = null; abierto = null;
    } catch (error) {
      if (!activa || error?.codigo === "operacion_abortada") return;
      if (error?.resultadoIndeterminado) { pendiente = { ...pendiente, incierta: true }; mensaje = t("circuito_incierto"); }
      else {
        mensaje = textoError(error, t, "circuito_error_decision"); pendiente = null;
        if (error?.codigo === "competencia_sin_fuente") { pagina = { items: [], competencia: "sin_fuente" }; abierto = null; cursores = [undefined]; indicePagina = 0; }
      }
      nivel = pagina.competencia === "sin_fuente" ? "" : "error"; publicar(mensaje);
    } finally {
      controladorDecision = null;
      if (activa) {
        const restaurarFoco = puedeRestaurarFoco(disparador) || documento.activeElement === estado;
        pintar();
        if (restaurarFoco) (pendiente?.incierta ? detalle.querySelector?.("[data-dietas-circuito-reintento]") : abierto ? detalle.querySelector?.("[data-dietas-circuito-motivo]") : estado)?.focus?.();
      }
    }
  }

  async function clic(evento) {
    const objetivo = evento.target;
    const abrirBoton = objetivo?.closest?.("[data-dietas-circuito-abrir]");
    if (abrirBoton) { await abrir(abrirBoton.dataset.dietasCircuitoAbrir); return; }
    const decision = objetivo?.closest?.("[data-dietas-circuito-decision]");
    if (decision) { await decidir(false, decision); return; }
    if (objetivo?.closest?.("[data-dietas-circuito-reintento]")) { await decidir(true); return; }
    if (objetivo?.closest?.("[data-dietas-circuito-volver]")) { cerrar(); return; }
    if (objetivo?.closest?.("[data-dietas-circuito-reintentar]")) { await cargar(indicePagina, true); return; }
    if (objetivo?.closest?.("[data-dietas-circuito-primera]") && indicePagina > 0) { await cargar(0, true); return; }
    if (objetivo?.closest?.("[data-dietas-circuito-anterior]") && indicePagina > 0) { await cargar(indicePagina - 1, true); return; }
    if (objetivo?.closest?.("[data-dietas-circuito-siguiente]") && pagina.siguiente_cursor) await cargar(indicePagina + 1, true);
  }
  async function enviarFiltros(evento) {
    evento?.preventDefault?.();
    if (pendiente) return;
    abierto = null;
    consulta = { etapa: selectorEtapa.value, limit: 20, ...(desde.value ? { fecha_desde: desde.value } : {}), ...(hasta.value ? { fecha_hasta: hasta.value } : {}) };
    cursores = [undefined]; indicePagina = 0;
    await cargar(0, true);
  }
  function desmontar() {
    if (!activa) return; activa = false;
    controladorLectura?.abort(); controladorDocumento?.abort(); controladorDecision?.abort();
    pagina = { items: [], competencia: "acreditada" }; cursores = []; abierto = null; pendiente = null;
    raiz.removeEventListener("click", clic); filtros.removeEventListener("submit", enviarFiltros); raiz.remove();
  }
  raiz.addEventListener("click", clic); filtros.addEventListener("submit", enviarFiltros); registrarDesmontar?.(desmontar); cargar();
  return Object.freeze({ desmontar, recargar: () => cargar() });
}
