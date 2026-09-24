import { crearTraductorDietas, MENSAJES_DIETAS_ES } from "./i18n.js?v=20260925-dietas-montaje-v1";

import { MENSAJES_CIRCUITO_DIETAS_ES } from "./i18n-circuito.js?v=20260925-dietas-montaje-v1";
export { MENSAJES_CIRCUITO_DIETAS_ES } from "./i18n-circuito.js?v=20260925-dietas-montaje-v1";

const ETAPAS = Object.freeze(["revision", "autorizacion", "liquidacion", "fiscalizacion"]);
const nodo = (documento, etiqueta, texto = "") => { const resultado = documento.createElement(etiqueta); if (texto) resultado.textContent = texto; return resultado; };
const montada = (contenedor, raiz) => contenedor.querySelector?.("[data-dietas-bandeja-circuito]") === raiz;
const fecha = (valor) => new Intl.DateTimeFormat("es-ES", { dateStyle: "medium", timeZone: "UTC" }).format(new Date(`${valor}T00:00:00Z`));
const instante = (valor) => new Intl.DateTimeFormat("es-ES", { dateStyle: "medium", timeStyle: "medium", timeZone: "Europe/Madrid" }).format(new Date(valor));
function crearTraductorCircuito(traducir) {
  return (clave, variables = {}) => {
    if (Object.hasOwn(MENSAJES_CIRCUITO_DIETAS_ES, clave)) {
      try { const resultado = traducir(clave, variables); if (typeof resultado === "string" && resultado !== clave) return resultado; } catch { /* El montaje puede recibir el catálogo previo a D8. */ }
      return MENSAJES_CIRCUITO_DIETAS_ES[clave].replace(/\{([a-z_]+)\}/gu, (_todo, nombre) => String(variables[nombre] ?? ""));
    }
    return traducir(clave, variables);
  };
}
function claveNueva(generar) {
  const valor = generar?.();
  if (typeof valor !== "string" || !/^[A-Za-z0-9:_-]{16,128}$/u.test(valor)) throw new TypeError("clave de idempotencia no disponible");
  return valor;
}
function textoError(error, t, esDecision = false) {
  if (error?.codigo === "acceso_denegado") return t("circuito_denegado");
  if (error?.codigo === "autenticacion_requerida") return t("circuito_autenticacion");
  if (error?.codigo === "conflicto_estado") return t("circuito_conflicto");
  if (error?.codigo === "no_encontrada") return t("circuito_no_encontrada");
  return t(esDecision ? "circuito_error_decision" : "circuito_error");
}

/** Monta una bandeja conectada a D6/D8. El recibo se muestra sólo tras una respuesta válida. */
export function montarVistaBandejaCircuitoDietas(contenedor, {
  cliente,
  traducir = crearTraductorDietas(MENSAJES_DIETAS_ES),
  anunciar = () => {},
  generarClaveIdempotencia = () => globalThis.crypto?.randomUUID?.(),
  registrarDesmontar,
  etapaInicial = "revision",
} = {}) {
  if (!contenedor?.ownerDocument || !cliente || typeof cliente.listar !== "function" || typeof cliente.decidir !== "function" || !ETAPAS.includes(etapaInicial)) throw new TypeError("bandeja del circuito de Dietas no disponible");
  const documento = contenedor.ownerDocument; const t = crearTraductorCircuito(traducir);
  const raiz = nodo(documento, "section"); raiz.dataset.dietasBandejaCircuito = ""; raiz.className = "panel dietas-bandeja-circuito";
  const cabecera = nodo(documento, "header"); cabecera.className = "cabecera-panel";
  cabecera.append(nodo(documento, "h2", t("circuito_titulo")), nodo(documento, "p", t("circuito_subtitulo")));
  const filtros = nodo(documento, "form"); filtros.className = "dietas-bandeja-circuito-filtros"; filtros.dataset.dietasCircuitoFiltros = "";
  const etapa = nodo(documento, "select"); etapa.name = "etapa"; etapa.setAttribute("aria-label", t("circuito_etapa"));
  ETAPAS.forEach((valor) => { const opcion = nodo(documento, "option", t(`circuito_etapa_${valor}`)); opcion.value = valor; opcion.selected = valor === etapaInicial; etapa.append(opcion); }); etapa.value = etapaInicial;
  const desde = nodo(documento, "input"); desde.type = "date"; desde.name = "fecha_desde"; desde.setAttribute("aria-label", t("circuito_fecha_desde"));
  const hasta = nodo(documento, "input"); hasta.type = "date"; hasta.name = "fecha_hasta"; hasta.setAttribute("aria-label", t("circuito_fecha_hasta"));
  const buscar = nodo(documento, "button", t("circuito_buscar")); buscar.type = "submit";
  const campoFiltro = (clave, control) => { const etiqueta = nodo(documento, "label", t(clave)); etiqueta.append(control); return etiqueta; };
  filtros.append(campoFiltro("circuito_etapa", etapa), campoFiltro("circuito_fecha_desde", desde), campoFiltro("circuito_fecha_hasta", hasta), buscar);
  const estado = nodo(documento, "p"); estado.dataset.dietasCircuitoEstado = ""; estado.setAttribute("role", "status");
  const contenido = nodo(documento, "div"); contenido.className = "cuerpo-panel"; contenido.dataset.dietasCircuitoContenido = "";
  raiz.append(cabecera, filtros, estado, contenido); contenedor.append(raiz);
  let activa = true; let controladorLectura; let controladorDecision; let generacion = 0;
  let consulta = { etapa: etapaInicial, limit: 20 }; let pagina = { items: [] }; let pendiente = null; let mensaje = ""; let nivelMensaje = "";
  const publicar = (texto) => { if (activa) anunciar(texto); };
  function pintar() {
    if (!activa || !montada(contenedor, raiz)) return;
    contenido.replaceChildren(); estado.textContent = mensaje; estado.className = nivelMensaje ? `dietas-bandeja-circuito-${nivelMensaje}` : "";
    if (nivelMensaje === "cargando") { contenido.append(nodo(documento, "p", t("circuito_cargando"))); return; }
    if ((nivelMensaje === "error" || nivelMensaje === "denegado") && !pendiente?.incierta) { const reintentar = nodo(documento, "button", t("circuito_reintentar_consulta")); reintentar.type = "button"; reintentar.dataset.dietasCircuitoReintentar = ""; contenido.append(reintentar); return; }
    if (!pagina.items.length) { contenido.append(nodo(documento, "p", t("circuito_vacio"))); return; }
    const tabla = nodo(documento, "table"); tabla.className = "tabla-admin dietas-bandeja-circuito-tabla";
    const cabeza = nodo(documento, "thead"), filaCabeza = nodo(documento, "tr");
    ["circuito_referencia", "circuito_estado", "circuito_version", "circuito_periodo", "circuito_acciones"].forEach((clave) => filaCabeza.append(nodo(documento, "th", t(clave)))); cabeza.append(filaCabeza);
    const cuerpo = nodo(documento, "tbody");
    pagina.items.forEach((item) => {
      const fila = nodo(documento, "tr"); fila.dataset.dietasCircuitoComision = item.referencia;
      const referencia = nodo(documento, "td");
      referencia.append(nodo(documento, "strong", t("circuito_comision_fecha", { fecha: fecha(item.fecha_inicio) })),
        nodo(documento, "small", item.referencia));
      fila.append(referencia, nodo(documento, "td", t(`circuito_estado_${item.estado}`)), nodo(documento, "td", String(item.version)), nodo(documento, "td", t("circuito_periodo_valor", { inicio: fecha(item.fecha_inicio), fin: fecha(item.fecha_fin) })));
      const acciones = nodo(documento, "td");
      const motivo = nodo(documento, "textarea"); motivo.name = `motivo_${item.referencia}`; motivo.maxLength = 600; motivo.setAttribute("aria-label", t("circuito_motivo")); motivo.setAttribute("aria-describedby", `ayuda_${item.referencia}`);
      const ayuda = nodo(documento, "details"); ayuda.id = `ayuda_${item.referencia}`;
      const resumenAyuda = nodo(documento, "summary", "?"); resumenAyuda.setAttribute("aria-label", t("recorridos_abrir_ayuda"));
      ayuda.append(resumenAyuda, nodo(documento, "p", t("circuito_motivo_ayuda")));
      const aprobar = nodo(documento, "button", t("circuito_aprobar")); aprobar.type = "button"; aprobar.dataset.dietasCircuitoDecision = "aprobar"; aprobar.dataset.dietasCircuitoReferencia = item.referencia;
      const devolver = nodo(documento, "button", t("circuito_devolver")); devolver.type = "button"; devolver.dataset.dietasCircuitoDecision = "devolver"; devolver.dataset.dietasCircuitoReferencia = item.referencia;
      const deshabilitar = Boolean(pendiente && pendiente.referencia === item.referencia); aprobar.disabled = deshabilitar; devolver.disabled = deshabilitar; motivo.disabled = deshabilitar;
      acciones.append(motivo, ayuda, aprobar, devolver);
      if (pendiente?.referencia === item.referencia && pendiente.incierta) { const reintento = nodo(documento, "button", t("circuito_reintentar_decision")); reintento.type = "button"; reintento.dataset.dietasCircuitoReintento = ""; acciones.append(reintento); }
      fila.append(acciones); cuerpo.append(fila);
    }); tabla.append(cabeza, cuerpo); contenido.append(tabla);
    if (pagina.siguiente_cursor) { const siguiente = nodo(documento, "button", t("circuito_siguiente")); siguiente.type = "button"; siguiente.dataset.dietasCircuitoSiguiente = ""; contenido.append(siguiente); }
  }
  async function cargar(siguiente = false) {
    controladorLectura?.abort(); controladorLectura = new AbortController(); const actual = ++generacion;
    const nuevaConsulta = siguiente ? { ...consulta, cursor: pagina.siguiente_cursor } : { ...consulta }; mensaje = t("circuito_cargando"); nivelMensaje = "cargando"; pintar();
    try {
      const recibida = await cliente.listar(nuevaConsulta, { signal: controladorLectura.signal });
      if (!activa || actual !== generacion) return;
      pagina = siguiente ? { items: [...pagina.items, ...recibida.items], ...(recibida.siguiente_cursor ? { siguiente_cursor: recibida.siguiente_cursor } : {}) } : recibida;
      mensaje = ""; nivelMensaje = "";
    } catch (error) {
      if (!activa || actual !== generacion || error?.codigo === "operacion_abortada") return;
      mensaje = textoError(error, t); nivelMensaje = error?.codigo === "acceso_denegado" || error?.codigo === "autenticacion_requerida" ? "denegado" : "error"; publicar(mensaje);
    } finally { if (activa && actual === generacion) pintar(); }
  }
  async function decidir(reintentar = false, boton) {
    if (controladorDecision || !activa) return;
    const referencia = reintentar ? pendiente?.referencia : boton?.dataset?.dietasCircuitoReferencia;
    const decision = reintentar ? pendiente?.decision : boton?.dataset?.dietasCircuitoDecision;
    const item = pagina.items.find((entrada) => entrada.referencia === referencia); if (!item || !["aprobar", "devolver"].includes(decision)) return;
    const campo = raiz.querySelector?.(`[name="motivo_${referencia}"]`); const motivo = reintentar ? pendiente.motivo : String(campo?.value ?? "").trim();
    if (decision === "devolver" && (motivo.length < 3 || motivo.length > 600)) { mensaje = t("circuito_decision_invalida"); nivelMensaje = "error"; pintar(); publicar(mensaje); campo?.focus?.(); return; }
    pendiente = reintentar ? pendiente : { referencia, decision, motivo, clave: claveNueva(generarClaveIdempotencia), version: item.version, incierta: false };
    controladorDecision = new AbortController(); mensaje = t("circuito_procesando"); nivelMensaje = "cargando"; pintar();
    try {
      const resultado = await cliente.decidir(referencia, { etapa: consulta.etapa, decision, motivo, clave_idempotencia: pendiente.clave, version_esperada: pendiente.version }, { signal: controladorDecision.signal });
      if (!activa) return;
      mensaje = t(resultado.recibo.repeticion ? "circuito_recibo_repetido" : "circuito_recibo", { referencia: resultado.recibo.referencia, version: resultado.recibo.version, fecha: instante(resultado.recibo.registrado_en) }); nivelMensaje = "exito"; publicar(mensaje); pendiente = null;
      pagina = { items: pagina.items.filter((entrada) => entrada.referencia !== referencia), ...(pagina.siguiente_cursor ? { siguiente_cursor: pagina.siguiente_cursor } : {}) };
    } catch (error) {
      if (!activa || error?.codigo === "operacion_abortada") return;
      if (error?.resultadoIndeterminado) { pendiente = { ...pendiente, incierta: true }; mensaje = t("circuito_incierto"); nivelMensaje = "error"; }
      else { mensaje = textoError(error, t, true); nivelMensaje = "error"; pendiente = null; }
      publicar(mensaje);
    } finally { controladorDecision = null; if (activa) pintar(); }
  }
  async function evento(evento) {
    const objetivo = evento.target?.closest?.("[data-dietas-circuito-decision]");
    if (objetivo) { await decidir(false, objetivo); return; }
    if (evento.target?.closest?.("[data-dietas-circuito-reintento]")) { await decidir(true); return; }
    if (evento.target?.closest?.("[data-dietas-circuito-reintentar]")) { await cargar(); return; }
    if (evento.target?.closest?.("[data-dietas-circuito-siguiente]")) await cargar(true);
  }
  async function enviarFiltros(evento) {
    evento.preventDefault?.();
    consulta = { etapa: etapa.value, limit: 20, ...(desde.value ? { fecha_desde: desde.value } : {}), ...(hasta.value ? { fecha_hasta: hasta.value } : {}) };
    await cargar();
  }
  function desmontar() { if (!activa) return; activa = false; controladorLectura?.abort(); controladorDecision?.abort(); raiz.removeEventListener("click", evento); filtros.removeEventListener("submit", enviarFiltros); raiz.remove(); }
  raiz.addEventListener("click", evento); filtros.addEventListener("submit", enviarFiltros); registrarDesmontar?.(desmontar); cargar();
  return Object.freeze({ desmontar, recargar: () => cargar() });
}
