import { crearTraductorCronos, MENSAJES_CRONOS_ES } from "./i18n.js";
import { crearClienteSaldoCronosHTTP, ErrorClienteSaldoCronos, validarConsultaSaldoCronos, validarResultadoSaldoCronos } from "./cliente-saldo-http.js";

const PERIODOS = ["hoy", "semana", "mes", "anio", "rango"];
const MOVIMIENTOS = new Set(["entrada", "salida", "inicio_pausa", "fin_pausa"]);
const ORIGENES = new Set(["terminal", "remoto"]);
const ESTADOS = new Set(["disponible", "no_disponible", "incompleto"]);

function escaparHTML(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}
function fechaVisible(fecha, locale) {
  const [anio, mes, dia] = fecha.split("-").map(Number);
  return new Intl.DateTimeFormat(locale, { day: "2-digit", month: "2-digit", year: "numeric", timeZone: "UTC" })
    .format(new Date(Date.UTC(anio, mes - 1, dia, 12)));
}
function horaVisible(instante, locale, zonaHoraria) {
  return new Intl.DateTimeFormat(locale, { hour: "2-digit", minute: "2-digit", timeZone: zonaHoraria })
    .format(new Date(instante));
}
function estadoVisible(estado, t) {
  return t(ESTADOS.has(estado) ? `saldo_estado_${estado}` : "saldo_estado_no_disponible");
}
function chipEstado(estado, t) {
  const clase = estado === "disponible" ? "exito" : estado === "incompleto" ? "aviso" : "neutro";
  return `<span class="cronos-estado cronos-estado-${clase}">${escaparHTML(estadoVisible(estado, t))}</span>`;
}
function filasMovimientos(detalle, t, locale, zonaHoraria) {
  return detalle.map((dia) => {
    const fecha = `<time datetime="${escaparHTML(dia.fecha)}">${escaparHTML(fechaVisible(dia.fecha, locale))}</time>`;
    if (!dia.marcajes.length) return `<tr><th scope="row">${fecha}</th><td colspan="3">${escaparHTML(t("movimientos_sin_marcajes"))}</td><td>${chipEstado(dia.estado, t)}</td></tr>`;
    return dia.marcajes.map((marcaje) => {
      const movimiento = t(MOVIMIENTOS.has(marcaje.movimiento) ? `saldo_movimiento_${marcaje.movimiento}` : "saldo_estado_no_disponible");
      const origen = t(ORIGENES.has(marcaje.origen) ? `saldo_origen_${marcaje.origen}` : "saldo_origen_sin_verificar");
      return `<tr><th scope="row">${fecha}</th><td><time datetime="${escaparHTML(marcaje.instante_utc)}">${escaparHTML(horaVisible(marcaje.instante_utc, locale, zonaHoraria))}</time></td><td>${escaparHTML(movimiento)}</td><td>${escaparHTML(origen)}</td><td>${chipEstado(dia.estado, t)}</td></tr>`;
    }).join("");
  }).join("");
}

/** Solo muestra hechos y estados recibidos del saldo propio; no infiere ausencias ni olvidos. */
export function renderizarVistaMovimientosCronos({ estado = "cargando", consulta = { periodo: "hoy" }, datos = null,
  mensajes = MENSAJES_CRONOS_ES, locale = "es-ES", zonaHoraria = "Europe/Madrid", correccionDisponible = false } = {}) {
  const t = crearTraductorCronos(mensajes);
  const seleccion = estado === "seleccion" && consulta?.periodo === "rango"
    ? { periodo: "rango", desde: "", hasta: "" } : validarConsultaSaldoCronos(consulta);
  if (estado === "listo") validarResultadoSaldoCronos(datos, seleccion);
  const rango = seleccion.periodo === "rango";
  const cargado = estado === "listo";
  const hayMarcajes = cargado && datos.detalle.some((dia) => dia.marcajes.length > 0);
  const claveEstado = estado === "denegado" ? "movimientos_denegado" : estado === "error" ? "movimientos_error"
    : estado === "seleccion" ? "movimientos_seleccionar_rango" : "movimientos_cargando";
  const cuerpo = cargado ? `${hayMarcajes ? "" : `<p class="cronos-vacio" role="status">${escaparHTML(t("movimientos_vacio"))}</p>`}
    <div class="cronos-tabla-contenedor"><table class="cronos-tabla"><caption>${escaparHTML(t("movimientos_detalle"))}</caption>
      <thead><tr><th scope="col">${escaparHTML(t("movimientos_fecha"))}</th><th scope="col">${escaparHTML(t("movimientos_hora"))}</th><th scope="col">${escaparHTML(t("movimientos_tipo"))}</th><th scope="col">${escaparHTML(t("movimientos_origen"))}</th><th scope="col">${escaparHTML(t("movimientos_estado"))}</th></tr></thead>
      <tbody>${filasMovimientos(datos.detalle, t, locale, zonaHoraria)}</tbody></table></div>`
    : `<p class="cronos-${estado === "denegado" ? "acceso-denegado" : "vacio"}" role="${estado === "error" ? "alert" : "status"}">${escaparHTML(t(claveEstado))}</p>`;
  return `<section class="cronos-area cronos-movimientos-conectado" aria-labelledby="cronos-movimientos-titulo" data-cronos-movimientos-estado="${escaparHTML(estado)}">
    <header class="cronos-encabezado"><div><p class="sobrelinea">${escaparHTML(t("sobrelinea"))}</p><h2 id="cronos-movimientos-titulo">${escaparHTML(t("movimientos_titulo"))}</h2></div>
      <button type="button" class="cronos-boton-ayuda" data-accion="ayuda" aria-label="${escaparHTML(t("abrir_ayuda", { asunto: t("movimientos_titulo") }))}" title="${escaparHTML(t("abrir_ayuda", { asunto: t("movimientos_titulo") }))}"><span aria-hidden="true">?</span></button></header>
    <section class="panel cronos-panel" aria-label="${escaparHTML(t("movimientos_periodos"))}"><div class="cabecera-panel"><h3>${escaparHTML(t("movimientos_periodos"))}</h3></div><div class="cuerpo-panel">
      <nav class="cronos-navegacion" aria-label="${escaparHTML(t("movimientos_periodos"))}">${PERIODOS.map((periodo) => `<button type="button" data-cronos-movimientos-periodo="${periodo}" aria-pressed="${seleccion.periodo === periodo}">${escaparHTML(t(`saldo_${periodo}`))}</button>`).join("")}</nav>
      <form data-cronos-movimientos-rango ${rango ? "" : "hidden"}><label>${escaparHTML(t("saldo_desde"))}<input type="date" name="desde" value="${escaparHTML(rango ? seleccion.desde : "")}" ${rango ? "required" : ""}></label><label>${escaparHTML(t("saldo_hasta"))}<input type="date" name="hasta" value="${escaparHTML(rango ? seleccion.hasta : "")}" ${rango ? "required" : ""}></label><button type="submit" class="boton-primario">${escaparHTML(t("saldo_consultar"))}</button><p data-cronos-movimientos-validacion role="alert" hidden></p></form>
    </div></section>
    <section class="panel cronos-panel" aria-labelledby="cronos-movimientos-detalle-titulo"><div class="cabecera-panel"><h3 id="cronos-movimientos-detalle-titulo">${escaparHTML(t("movimientos_detalle"))}</h3></div>${cuerpo}</section>
    <section class="panel cronos-panel" aria-labelledby="cronos-movimientos-correccion-titulo"><div class="cabecera-panel"><h3 id="cronos-movimientos-correccion-titulo">${escaparHTML(t("movimientos_correccion"))}</h3></div><div class="cuerpo-panel">${correccionDisponible
      ? `<button type="button" class="boton-secundario" data-cronos-accion="solicitar-correccion">${escaparHTML(t("movimientos_correccion"))}</button>`
      : `<button type="button" class="boton-secundario" data-cronos-accion="solicitar-correccion" disabled aria-disabled="true" title="${escaparHTML(t("movimientos_correccion_pendiente"))}" aria-label="${escaparHTML(`${t("movimientos_correccion")}. ${t("movimientos_correccion_pendiente")}`)}">${escaparHTML(t("movimientos_correccion"))}</button>`}</div></section>
  </section>`;
}

/**
 * Inyecta el cliente propio y cancela peticiones al cambiar o desmontar. Con
 * `abrirCorreccion`, «olvido de marcaje» abre la solicitud de corrección; el
 * marcaje registrado nunca se edita desde aquí.
 */
export function montarVistaMovimientosCronos({ raiz, cliente = crearClienteSaldoCronosHTTP(), mensajes = MENSAJES_CRONOS_ES,
  anunciar = () => {}, registrarDesmontar, locale = "es-ES", zonaHoraria = "Europe/Madrid", abrirCorreccion } = {}) {
  if (!raiz?.append || !raiz.ownerDocument?.createElement || typeof cliente?.consultar !== "function"
    || typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")
    || (abrirCorreccion !== undefined && typeof abrirCorreccion !== "function")) throw new TypeError("montaje de movimientos Cronos no disponible");
  const contenedor = raiz.ownerDocument.createElement("section"); contenedor.dataset.cronosMovimientos = ""; raiz.append(contenedor);
  const t = crearTraductorCronos(mensajes);
  let activa = true; let secuencia = 0; let controlador = null; let consulta = { periodo: "hoy" };
  const dibujar = (estado, datos) => {
    if (activa) contenedor.innerHTML = renderizarVistaMovimientosCronos({ estado, consulta, datos, mensajes, locale, zonaHoraria, correccionDisponible: abrirCorreccion !== undefined });
  };
  const cargar = async (siguiente) => {
    consulta = validarConsultaSaldoCronos(siguiente);
    controlador?.abort(); controlador = new AbortController(); const turno = ++secuencia;
    dibujar("cargando");
    try {
      const datos = await cliente.consultar(consulta, { signal: controlador.signal });
      if (!activa || turno !== secuencia) return;
      dibujar("listo", datos);
    } catch (error) {
      if (!activa || turno !== secuencia || controlador.signal.aborted) return;
      const estado = error instanceof ErrorClienteSaldoCronos && error.codigo === "acceso_denegado" ? "denegado" : "error";
      dibujar(estado); anunciar(estado);
    }
  };
  const alPulsar = (evento) => {
    if (abrirCorreccion && evento.target?.closest?.('[data-cronos-accion="solicitar-correccion"]')) { abrirCorreccion(); return; }
    const boton = evento.target?.closest?.("[data-cronos-movimientos-periodo]");
    if (!boton) return;
    const periodo = boton.dataset.cronosMovimientosPeriodo;
    if (!PERIODOS.includes(periodo)) return;
    if (periodo === "rango") {
      controlador?.abort(); ++secuencia; consulta = { periodo: "rango", desde: "", hasta: "" };
      dibujar("seleccion"); contenedor.querySelector?.("[name=desde]")?.focus?.(); return;
    }
    void cargar({ periodo });
  };
  const alEnviar = (evento) => {
    if (!evento.target?.matches?.("[data-cronos-movimientos-rango]")) return;
    evento.preventDefault();
    const desde = evento.target.elements?.namedItem?.("desde")?.value;
    const hasta = evento.target.elements?.namedItem?.("hasta")?.value;
    try { validarConsultaSaldoCronos({ periodo: "rango", desde, hasta }); }
    catch {
      const validacion = contenedor.querySelector?.("[data-cronos-movimientos-validacion]");
      if (validacion) { validacion.textContent = t("movimientos_rango_invalido"); validacion.hidden = false; }
      anunciar("rango_invalido"); return;
    }
    void cargar({ periodo: "rango", desde, hasta });
  };
  contenedor.addEventListener("click", alPulsar); contenedor.addEventListener("submit", alEnviar);
  void cargar(consulta);
  const desmontar = () => {
    if (!activa) return;
    activa = false; ++secuencia; controlador?.abort();
    contenedor.removeEventListener("click", alPulsar); contenedor.removeEventListener("submit", alEnviar); contenedor.remove?.();
  };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ desmontar, consultar: cargar });
}
