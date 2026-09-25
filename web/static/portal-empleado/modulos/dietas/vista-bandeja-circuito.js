import { crearTraductorDietas, MENSAJES_DIETAS_ES } from "./i18n.js?v=20260925-tanda2-v1";

import { MENSAJES_CIRCUITO_DIETAS_ES } from "./i18n-circuito.js?v=20260925-tanda2-v1";
export { MENSAJES_CIRCUITO_DIETAS_ES } from "./i18n-circuito.js?v=20260925-tanda2-v1";
import { LOCALIZACION_PORTAL, ZONA_HORARIA_PORTAL } from "../../portal-i18n.js?v=20260925-tanda2-v1";

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
    if (Object.hasOwn(MENSAJES_CIRCUITO_DIETAS_ES, clave)) {
      try { const resultado = traducir(clave, variables); if (typeof resultado === "string" && resultado !== clave) return resultado; } catch { /* El montaje puede recibir un catálogo anterior al circuito. */ }
      return MENSAJES_CIRCUITO_DIETAS_ES[clave].replace(/\{([a-z_]+)\}/gu, (_todo, nombre) => String(variables[nombre] ?? ""));
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
  return String(linea.concepto ?? "");
}

/**
 * Bandeja de una etapa del circuito de Dietas. Con `control` añade la búsqueda
 * por etapa y fechas («Control de documentos»). Cada acción consulta el
 * servidor; el recibo solo se muestra tras una respuesta válida y, si la
 * fuente de competencia no existe, la bandeja lo dice sin ofrecer acciones.
 */
export function montarVistaBandejaCircuitoDietas(contenedor, {
  cliente,
  traducir = crearTraductorDietas(MENSAJES_DIETAS_ES),
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
  const documento = contenedor.ownerDocument; const t = crearTraductorCircuito(traducir);
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
  const estado = nodo(documento, "p"); estado.dataset.dietasCircuitoEstado = ""; estado.setAttribute("role", "status"); estado.className = "dietas-bandeja-circuito-estado";
  const contenido = nodo(documento, "div"); contenido.className = "cuerpo-panel"; contenido.dataset.dietasCircuitoContenido = "";
  const detalle = nodo(documento, "div"); detalle.className = "dietas-circuito-detalle"; detalle.dataset.dietasCircuitoDetalle = ""; detalle.hidden = true;
  raiz.append(cabecera, filtros, estado, contenido, detalle); contenedor.append(raiz);

  let activa = true; let controladorLectura; let controladorDocumento; let controladorDecision; let generacion = 0;
  let consulta = { etapa: etapaInicial, limit: 20 }; let pagina = { items: [], competencia: "acreditada" };
  let pendiente = null; let abierto = null; let mensaje = ""; let nivel = ""; let nivelDetalle = "";
  const publicar = (texto) => { if (activa) anunciar(texto); };
  const filtrada = () => Boolean(consulta.fecha_desde || consulta.fecha_hasta);

  function pintarLista() {
    titulo.textContent = control ? t("circuito_control") : t(`circuito_bandeja_${consulta.etapa}`);
    contenido.replaceChildren(); contenido.hidden = Boolean(abierto);
    if (nivel === "cargando") { contenido.append(nodo(documento, "p", t("circuito_cargando"))); return; }
    if (nivel === "error" && !pendiente?.incierta && !pagina.items.length) {
      const reintentar = nodo(documento, "button", t("circuito_reintentar_consulta")); reintentar.type = "button"; reintentar.className = "boton-secundario"; reintentar.dataset.dietasCircuitoReintentar = ""; contenido.append(reintentar); return;
    }
    if (pagina.competencia === "sin_fuente") { const linea = nodo(documento, "p", t("circuito_sin_fuente")); linea.dataset.dietasCircuitoSinFuente = ""; contenido.append(linea); return; }
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
    if (pagina.siguiente_cursor) { const siguiente = nodo(documento, "button", t("circuito_siguiente")); siguiente.type = "button"; siguiente.className = "boton-secundario"; siguiente.dataset.dietasCircuitoSiguiente = ""; contenido.append(siguiente); }
  }

  function pintarDetalle() {
    detalle.replaceChildren(); detalle.hidden = !abierto;
    if (!abierto) return;
    const volver = nodo(documento, "button", t("circuito_volver")); volver.type = "button"; volver.className = "boton-secundario"; volver.dataset.dietasCircuitoVolver = "";
    if (nivelDetalle === "cargando") { detalle.append(volver, nodo(documento, "p", t("circuito_documento_cargando"))); return; }
    if (!abierto.documento) { detalle.append(volver, nodo(documento, "p", t("circuito_documento_error"))); return; }
    const d = abierto.documento; const lineas = Array.isArray(d.documento?.lineas) ? d.documento.lineas : [];
    const encabezado = nodo(documento, "div"); encabezado.className = "dietas-circuito-detalle-cabecera";
    encabezado.append(nodo(documento, "h3", t("circuito_documento_titulo", { numero: d.numero_documento })), volver);
    const resumen = nodo(documento, "dl"); resumen.className = "dietas-circuito-resumen";
    [["circuito_documento_periodo", t("circuito_documento_periodo_valor", { inicio: fecha(d.fecha_inicio), hora_inicio: d.hora_inicio, fin: fecha(d.fecha_fin), hora_fin: d.hora_fin })],
      ["circuito_documento_apertura", instante(d.fecha_apertura)], ["circuito_documento_motivo", d.motivo]].forEach(([clave, valor]) => {
      const grupo = nodo(documento, "div"); grupo.append(nodo(documento, "dt", t(clave)), nodo(documento, "dd", valor)); resumen.append(grupo);
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
    const etiqueta = nodo(documento, "label", t("circuito_motivo")); etiqueta.setAttribute("for", motivo.id);
    const botones = nodo(documento, "div"); botones.className = "dietas-circuito-botones";
    const aprobar = nodo(documento, "button", t(`circuito_aprobar_${consulta.etapa}`)); aprobar.type = "button"; aprobar.className = "boton-primario"; aprobar.dataset.dietasCircuitoDecision = "aprobar";
    const devolver = nodo(documento, "button", t("circuito_devolver")); devolver.type = "button"; devolver.className = "boton-peligro"; devolver.dataset.dietasCircuitoDecision = "devolver";
    const bloqueado = Boolean(controladorDecision || pendiente); aprobar.disabled = bloqueado; devolver.disabled = bloqueado; motivo.disabled = bloqueado;
    botones.append(aprobar, devolver);
    if (pendiente?.incierta) { const reintento = nodo(documento, "button", t("circuito_reintentar_decision")); reintento.type = "button"; reintento.className = "boton-secundario"; reintento.dataset.dietasCircuitoReintento = ""; botones.append(reintento); }
    decision.append(etiqueta, motivo, botones);
    detalle.append(encabezado, resumen, marco, decision);
  }

  function pintar() {
    if (!activa || !montada(contenedor, raiz)) return;
    estado.textContent = mensaje; estado.dataset.nivel = nivel;
    pintarLista(); pintarDetalle();
  }

  async function cargar(siguiente = false) {
    controladorLectura?.abort(); controladorLectura = new AbortController(); const actual = ++generacion;
    const nuevaConsulta = siguiente ? { ...consulta, cursor: pagina.siguiente_cursor } : { ...consulta };
    if (!siguiente) { mensaje = ""; nivel = "cargando"; pintar(); }
    try {
      const recibida = await cliente.listar(nuevaConsulta, { signal: controladorLectura.signal });
      if (!activa || actual !== generacion) return;
      pagina = siguiente ? { ...recibida, items: [...pagina.items, ...recibida.items] } : recibida;
      if (nivel === "cargando") nivel = "";
    } catch (error) {
      if (!activa || actual !== generacion || error?.codigo === "operacion_abortada") return;
      mensaje = textoError(error, t, "circuito_error"); nivel = "error"; publicar(mensaje);
    } finally { if (activa && actual === generacion) pintar(); }
  }

  async function abrir(referencia) {
    const item = pagina.items.find((entrada) => entrada.referencia === referencia);
    if (!item || typeof cliente.documento !== "function") return;
    controladorDocumento?.abort(); controladorDocumento = new AbortController();
    abierto = { referencia, version: item.version, documento: null }; nivelDetalle = "cargando"; mensaje = ""; pintar();
    try {
      const documentoLeido = await cliente.documento(referencia, consulta.etapa, { signal: controladorDocumento.signal });
      if (!activa || abierto?.referencia !== referencia) return;
      abierto = { ...abierto, documento: documentoLeido, version: documentoLeido.version };
    } catch (error) {
      if (!activa || error?.codigo === "operacion_abortada") return;
      mensaje = textoError(error, t, "circuito_documento_error"); nivel = "error"; publicar(mensaje);
    } finally { nivelDetalle = ""; if (activa) pintar(); }
  }

  function cerrar() { controladorDocumento?.abort(); if (pendiente) return; abierto = null; pintar(); }

  async function decidir(reintentar, boton) {
    if (controladorDecision || !activa || !abierto?.documento) return;
    if (!reintentar && pendiente) return;
    const decision = reintentar ? pendiente?.decision : boton?.dataset?.dietasCircuitoDecision;
    if (!["aprobar", "devolver"].includes(decision)) return;
    const campoMotivo = detalle.querySelector?.("[data-dietas-circuito-motivo]");
    const motivo = reintentar ? pendiente.motivo : String(campoMotivo?.value ?? "").trim();
    if (!reintentar && decision === "devolver" && (motivo.length < 3 || motivo.length > 600)) {
      mensaje = t("circuito_decision_invalida"); nivel = "error"; abierto = { ...abierto, motivo }; pintar(); publicar(mensaje);
      detalle.querySelector?.("[data-dietas-circuito-motivo]")?.focus?.(); return;
    }
    if (!reintentar) pendiente = { referencia: abierto.referencia, decision, motivo, clave: claveNueva(generarClaveIdempotencia), version: abierto.version, incierta: false };
    abierto = { ...abierto, motivo };
    controladorDecision = new AbortController(); mensaje = t("circuito_procesando"); nivel = "cargando"; pintarDetalle(); estado.textContent = mensaje;
    try {
      const resultado = await cliente.decidir(pendiente.referencia, { etapa: consulta.etapa, decision: pendiente.decision, motivo: pendiente.motivo, clave_idempotencia: pendiente.clave, version_esperada: pendiente.version }, { signal: controladorDecision.signal });
      if (!activa) return;
      mensaje = t(resultado.recibo.repeticion ? "circuito_recibo_repetido" : "circuito_recibo", { referencia: resultado.recibo.referencia, version: resultado.recibo.version, fecha: instante(resultado.recibo.registrado_en) });
      nivel = "exito"; publicar(mensaje);
      pagina = { ...pagina, items: pagina.items.filter((entrada) => entrada.referencia !== pendiente.referencia) };
      pendiente = null; abierto = null;
    } catch (error) {
      if (!activa || error?.codigo === "operacion_abortada") return;
      if (error?.resultadoIndeterminado) { pendiente = { ...pendiente, incierta: true }; mensaje = t("circuito_incierto"); }
      else { mensaje = textoError(error, t, "circuito_error_decision"); pendiente = null; }
      nivel = "error"; publicar(mensaje);
    } finally { controladorDecision = null; if (activa) pintar(); }
  }

  async function clic(evento) {
    const objetivo = evento.target;
    const abrirBoton = objetivo?.closest?.("[data-dietas-circuito-abrir]");
    if (abrirBoton) { await abrir(abrirBoton.dataset.dietasCircuitoAbrir); return; }
    const decision = objetivo?.closest?.("[data-dietas-circuito-decision]");
    if (decision) { await decidir(false, decision); return; }
    if (objetivo?.closest?.("[data-dietas-circuito-reintento]")) { await decidir(true); return; }
    if (objetivo?.closest?.("[data-dietas-circuito-volver]")) { cerrar(); return; }
    if (objetivo?.closest?.("[data-dietas-circuito-reintentar]")) { await cargar(); return; }
    if (objetivo?.closest?.("[data-dietas-circuito-siguiente]")) await cargar(true);
  }
  async function enviarFiltros(evento) {
    evento?.preventDefault?.();
    if (pendiente) return;
    abierto = null;
    consulta = { etapa: selectorEtapa.value, limit: 20, ...(desde.value ? { fecha_desde: desde.value } : {}), ...(hasta.value ? { fecha_hasta: hasta.value } : {}) };
    await cargar();
  }
  function desmontar() {
    if (!activa) return; activa = false;
    controladorLectura?.abort(); controladorDocumento?.abort(); controladorDecision?.abort();
    raiz.removeEventListener("click", clic); filtros.removeEventListener("submit", enviarFiltros); raiz.remove();
  }
  raiz.addEventListener("click", clic); filtros.addEventListener("submit", enviarFiltros); registrarDesmontar?.(desmontar); cargar();
  return Object.freeze({ desmontar, recargar: () => cargar() });
}
