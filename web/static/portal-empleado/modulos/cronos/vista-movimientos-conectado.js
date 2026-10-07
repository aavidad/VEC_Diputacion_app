import { crearTraductorCronos, MENSAJES_CRONOS } from "./i18n.js?v=20260929-i18n-textos-v1";
import { crearClienteSaldoCronosHTTP, ErrorClienteSaldoCronos, validarConsultaSaldoCronos, validarResultadoSaldoCronos } from "./cliente-saldo-http.js";
import { crearTraductorConsultaCronos, MENSAJES_CONSULTA_CRONOS } from "./i18n-consulta.js?v=20261001-cronos-grafo-bandeja-v5";
import { LOCALIZACION_ACTUAL } from "../../../comun/idioma.js";

const PERIODOS = ["hoy", "semana", "mes", "anio", "rango"];
const MOVIMIENTOS = new Set(["entrada", "salida", "inicio_pausa", "fin_pausa"]);
const ORIGENES = new Set(["terminal", "remoto"]);
const ESTADOS = new Set(["disponible", "no_disponible", "incompleto"]);

function escaparHTML(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}

/** Encabezado de la parte: propio de página, o de tarjeta sin sobrelínea dentro de «Jornada». */
function encabezadoParte(sobrelinea, id, titulo, incrustada) {
  return incrustada ? `<h3 id="${id}">${escaparHTML(titulo)}</h3>`
    : `<p class="sobrelinea">${escaparHTML(sobrelinea)}</p><h2 id="${id}">${escaparHTML(titulo)}</h2>`;
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

// Pagina los días recibidos, manteniendo juntos sus marcajes y el periodo autorizado.
function paginarDetalle(detalle, pagina, tamanoPagina) {
  if (!Number.isSafeInteger(tamanoPagina) || tamanoPagina < 1 || tamanoPagina > 367
    || !Number.isSafeInteger(pagina) || pagina < 1) throw new RangeError("página de movimientos no válida");
  const paginas = Math.max(1, Math.ceil(detalle.length / tamanoPagina));
  const actual = Math.min(pagina, paginas); const inicio = (actual - 1) * tamanoPagina;
  return { actual, paginas, inicio, filas: detalle.slice(inicio, inicio + tamanoPagina) };
}
function recuentoPagina(paginacion, total, traducir, locale) {
  const numero = (valor) => new Intl.NumberFormat(locale).format(valor);
  return traducir("recuento", { desde: numero(paginacion.inicio + 1), hasta: numero(paginacion.inicio + paginacion.filas.length),
    total: numero(total), pagina: numero(paginacion.actual), paginas: numero(paginacion.paginas) });
}

/** Solo muestra hechos y estados recibidos del saldo propio; no infiere ausencias ni olvidos. */
export function renderizarVistaMovimientosCronos({ estado = "cargando", consulta = { periodo: "hoy" }, datos = null,
  mensajes = MENSAJES_CRONOS, locale = LOCALIZACION_ACTUAL, zonaHoraria = "Europe/Madrid", correccionDisponible = false, incrustada = false, pagina = 1, tamanoPagina = 31,
  mensajesConsulta = MENSAJES_CONSULTA_CRONOS } = {}) {
  const tc = crearTraductorConsultaCronos(mensajesConsulta);
  const t = crearTraductorCronos(mensajes);
  const seleccion = estado === "seleccion" && consulta?.periodo === "rango"
    ? { periodo: "rango", desde: consulta.desde ?? "", hasta: consulta.hasta ?? "" } : validarConsultaSaldoCronos(consulta);
  if (estado === "listo") validarResultadoSaldoCronos(datos, seleccion);
  const paginacion = paginarDetalle(estado === "listo" ? datos.detalle : [], pagina, tamanoPagina);
  const controles = estado === "listo" && datos.detalle.length ? `<nav class="cronos-navegacion" aria-label="${escaparHTML(tc("paginacion_movimientos"))}">
    <button type="button" class="boton-secundario" data-cronos-movimientos-pagina="anterior"${paginacion.actual === 1 ? " disabled" : ""}>${escaparHTML(tc("pagina_anterior"))}</button>
    <p data-cronos-movimientos-recuento tabindex="-1" role="status" aria-live="polite" aria-atomic="true">${escaparHTML(recuentoPagina(paginacion, datos.detalle.length, tc, locale))}</p>
    <button type="button" class="boton-secundario" data-cronos-movimientos-pagina="siguiente"${paginacion.actual === paginacion.paginas ? " disabled" : ""}>${escaparHTML(tc("pagina_siguiente"))}</button></nav>` : "";
  const actualizar = ["listo", "error", "cargando"].includes(estado) ? `<button type="button" class="boton-secundario" data-cronos-movimientos-actualizar${estado === "cargando" ? ' aria-disabled="true"' : ""}>${escaparHTML(tc(estado === "error" ? "reintentar" : "actualizar"))}</button>` : "";
  const rango = seleccion.periodo === "rango";
  const cargado = estado === "listo";
  const hayMarcajes = cargado && datos.detalle.some((dia) => dia.marcajes.length > 0);
  const claveEstado = estado === "denegado" ? "movimientos_denegado" : estado === "error" ? "movimientos_error"
    : estado === "no_disponible" ? "movimientos_no_disponible"
    : estado === "seleccion" ? "movimientos_seleccionar_rango" : "movimientos_cargando";
  const cuerpo = cargado ? `<p class="cronos-mensaje">${escaparHTML(tc("periodo_movimientos", { desde: fechaVisible(datos.periodo.desde, locale), hasta: fechaVisible(datos.periodo.hasta, locale) }))}</p>${hayMarcajes ? "" : `<p class="cronos-vacio" role="status">${escaparHTML(t("movimientos_vacio"))}</p>`}
    <div class="cronos-tabla-contenedor"><table class="cronos-tabla"><caption>${escaparHTML(t("movimientos_detalle"))}</caption>
      <thead><tr><th scope="col">${escaparHTML(t("movimientos_fecha"))}</th><th scope="col">${escaparHTML(t("movimientos_hora"))}</th><th scope="col">${escaparHTML(t("movimientos_tipo"))}</th><th scope="col">${escaparHTML(t("movimientos_origen"))}</th><th scope="col">${escaparHTML(t("movimientos_estado"))}</th></tr></thead>
      <tbody>${filasMovimientos(paginacion.filas, t, locale, zonaHoraria)}</tbody></table></div><div class="cuerpo-panel">${controles}</div>`
    : `<p class="cronos-${estado === "denegado" ? "acceso-denegado" : "vacio"}" data-cronos-movimientos-mensaje tabindex="-1" role="${estado === "error" ? "alert" : "status"}">${escaparHTML(t(claveEstado))}</p>`;
  return `<section class="cronos-area cronos-movimientos-conectado" aria-labelledby="cronos-movimientos-titulo" data-cronos-movimientos-estado="${escaparHTML(estado)}" aria-busy="${estado === "cargando"}">
    <header class="cronos-encabezado"><div>${encabezadoParte(t("sobrelinea"), "cronos-movimientos-titulo", t("movimientos_titulo"), incrustada)}</div>
      <button type="button" class="cronos-boton-ayuda" data-accion="ayuda" aria-label="${escaparHTML(t("abrir_ayuda", { asunto: t("movimientos_titulo") }))}" title="${escaparHTML(t("abrir_ayuda", { asunto: t("movimientos_titulo") }))}"><span aria-hidden="true">?</span></button></header>
    <section class="panel cronos-panel" aria-label="${escaparHTML(t("movimientos_periodos"))}"><div class="cabecera-panel"><h3>${escaparHTML(t("movimientos_periodos"))}</h3>${actualizar}</div><div class="cuerpo-panel">
      <nav class="cronos-navegacion" aria-label="${escaparHTML(t("movimientos_periodos"))}">${PERIODOS.map((periodo) => `<button type="button" data-cronos-movimientos-periodo="${periodo}" aria-pressed="${seleccion.periodo === periodo}">${escaparHTML(t(`saldo_${periodo}`))}</button>`).join("")}</nav>
      <form data-cronos-movimientos-rango ${rango ? "" : "hidden"}><label>${escaparHTML(t("saldo_desde"))}<input type="date" name="desde" value="${escaparHTML(rango ? seleccion.desde : "")}" ${rango ? "required" : ""}></label><label>${escaparHTML(t("saldo_hasta"))}<input type="date" name="hasta" value="${escaparHTML(rango ? seleccion.hasta : "")}" ${rango ? "required" : ""}></label><button type="submit" class="boton-primario" data-cronos-movimientos-consultar>${escaparHTML(tc("consultar_movimientos"))}</button><p data-cronos-movimientos-validacion role="alert" hidden></p></form>
    </div></section>
    <section class="panel cronos-panel" aria-labelledby="cronos-movimientos-detalle-titulo"><div class="cabecera-panel"><h3 id="cronos-movimientos-detalle-titulo">${escaparHTML(t("movimientos_detalle"))}</h3></div><div data-cronos-movimientos-contenido>${cuerpo}</div></section>
    ${correccionDisponible ? `<section class="panel cronos-panel" aria-labelledby="cronos-movimientos-correccion-titulo"><div class="cabecera-panel"><h3 id="cronos-movimientos-correccion-titulo">${escaparHTML(t("movimientos_correccion"))}</h3></div><div class="cuerpo-panel">
      <button type="button" class="boton-secundario" data-cronos-accion="solicitar-correccion">${escaparHTML(t("movimientos_correccion"))}</button></div></section>` : ""}
  </section>`;
}

/**
 * Inyecta el cliente propio y cancela peticiones al cambiar o desmontar. Solo
 * con `abrirCorreccion` se ofrece «olvido de marcaje», que abre la solicitud
 * de corrección; el marcaje registrado nunca se edita desde aquí.
 */
export function montarVistaMovimientosCronos({ raiz, cliente = crearClienteSaldoCronosHTTP(), mensajes = MENSAJES_CRONOS,
  anunciar = () => {}, registrarDesmontar, locale = LOCALIZACION_ACTUAL, zonaHoraria = "Europe/Madrid", abrirCorreccion, incrustada = false, tamanoPagina = 31,
  mensajesConsulta = MENSAJES_CONSULTA_CRONOS } = {}) {
  if (!raiz?.append || !raiz.ownerDocument?.createElement || typeof cliente?.consultar !== "function"
    || typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")
    || (abrirCorreccion !== undefined && typeof abrirCorreccion !== "function")) throw new TypeError("montaje de movimientos Cronos no disponible");
  const contenedor = raiz.ownerDocument.createElement("section"); contenedor.dataset.cronosMovimientos = ""; raiz.append(contenedor);
  const t = crearTraductorCronos(mensajes);
  let activa = true; let secuencia = 0; let controlador = null; let consulta = { periodo: "hoy" };
  let pagina = 1; let datos = null; let estado = "cargando"; let borradorRango = null;
  paginarDetalle([], pagina, tamanoPagina);
  const tc = crearTraductorConsultaCronos(mensajesConsulta);
  const selectorFoco = () => {
    const foco = raiz.ownerDocument.activeElement;
    if (!foco || !contenedor.contains?.(foco)) return null;
    for (const atributo of ["data-cronos-movimientos-periodo", "data-cronos-movimientos-pagina", "data-cronos-movimientos-actualizar", "data-cronos-movimientos-consultar", "data-cronos-movimientos-recuento", "data-cronos-movimientos-mensaje", "data-accion", "name"]) {
      const valor = foco.getAttribute?.(atributo);
      if (valor !== null && valor !== undefined && /^[a-z_]*$/u.test(valor)) return `[${atributo}="${valor}"]`;
    }
    return null;
  };
  const dibujar = () => {
    if (!activa) return;
    const selector = selectorFoco();
    contenedor.innerHTML = renderizarVistaMovimientosCronos({ estado, consulta: borradorRango ?? consulta, datos, mensajes, mensajesConsulta,
      locale, zonaHoraria, correccionDisponible: abrirCorreccion !== undefined, incrustada, pagina, tamanoPagina });
    if (selector) {
      const destino = contenedor.querySelector?.(selector);
      (destino && !destino.disabled ? destino : contenedor.querySelector?.("[data-cronos-movimientos-recuento]")
        ?? contenedor.querySelector?.("[data-cronos-movimientos-actualizar]")
        ?? contenedor.querySelector?.("[data-cronos-movimientos-mensaje]"))?.focus?.();
    }
  };
  const cargar = async (siguiente, conservarPagina = false) => {
    if (!activa) return;
    consulta = validarConsultaSaldoCronos(siguiente); borradorRango = null;
    if (!conservarPagina) pagina = 1;
    controlador?.abort(); controlador = new AbortController(); const turno = ++secuencia;
    estado = "cargando"; datos = null; dibujar();
    try {
      const respuesta = await cliente.consultar(consulta, { signal: controlador.signal });
      if (!activa || turno !== secuencia) return;
      datos = validarResultadoSaldoCronos(respuesta, consulta);
      pagina = paginarDetalle(datos.detalle, pagina, tamanoPagina).actual;
      estado = "listo"; dibujar(); anunciar(tc("consulta_movimientos_lista"));
    } catch (error) {
      if (!activa || turno !== secuencia || controlador.signal.aborted) return;
      datos = null;
      estado = !(error instanceof ErrorClienteSaldoCronos) ? "error" : error.codigo === "acceso_denegado" ? "denegado"
        : error.estado === 404 ? "no_disponible" : "error";
      dibujar(); anunciar(t(estado === "denegado" ? "movimientos_denegado" : estado === "no_disponible" ? "movimientos_no_disponible" : "movimientos_error"));
    }
  };
  // El rango sin enviar es un borrador: el refresco conjunto no lo consulta ni redibuja.
  // Un fichaje confirmado puede sustituir una lectura en curso del mismo periodo.
  const actualizar = () => ["listo", "error", "cargando"].includes(estado) ? cargar(consulta, true) : Promise.resolve();
  const alPulsar = (evento) => {
    if (abrirCorreccion && evento.target?.closest?.('[data-cronos-accion="solicitar-correccion"]')) { abrirCorreccion(); return; }
    if (evento.target?.closest?.("[data-cronos-movimientos-actualizar]")) {
      if (estado !== "cargando") void actualizar();
      return;
    }
    const cambio = evento.target?.closest?.("[data-cronos-movimientos-pagina]");
    if (cambio && estado === "listo" && !cambio.disabled) {
      const paso = cambio.dataset.cronosMovimientosPagina;
      if (!["anterior", "siguiente"].includes(paso)) return;
      pagina = paginarDetalle(datos.detalle, Math.max(1, pagina + (paso === "anterior" ? -1 : 1)), tamanoPagina).actual;
      dibujar(); anunciar(recuentoPagina(paginarDetalle(datos.detalle, pagina, tamanoPagina), datos.detalle.length, tc, locale)); return;
    }
    const boton = evento.target?.closest?.("[data-cronos-movimientos-periodo]");
    if (!boton) return;
    const periodo = boton.dataset.cronosMovimientosPeriodo;
    if (!PERIODOS.includes(periodo)) return;
    if (periodo === "rango") {
      if (borradorRango) return;
      controlador?.abort(); ++secuencia;
      borradorRango = { periodo: "rango", desde: "", hasta: "" }; estado = "seleccion"; datos = null;
      dibujar(); contenedor.querySelector?.("[name=desde]")?.focus?.(); return;
    }
    void cargar({ periodo });
  };
  const alEditar = (evento) => {
    const formulario = evento.target?.closest?.("[data-cronos-movimientos-rango]");
    if (!formulario || !["desde", "hasta"].includes(evento.target.name)) return;
    borradorRango = { periodo: "rango", desde: formulario.elements.namedItem("desde").value,
      hasta: formulario.elements.namedItem("hasta").value };
    controlador?.abort(); ++secuencia; estado = "seleccion"; datos = null;
    // Sustituir sólo el resultado evita interrumpir la edición y mantiene su foco.
    const contenido = contenedor.querySelector?.("[data-cronos-movimientos-contenido]");
    if (contenido) contenido.innerHTML = `<p class="cronos-vacio" role="status">${escaparHTML(t("movimientos_seleccionar_rango"))}</p>`;
    const accion = contenedor.querySelector?.("[data-cronos-movimientos-actualizar]");
    if (accion) accion.hidden = true;
    const region = contenedor.querySelector?.("[data-cronos-movimientos-estado]");
    region?.setAttribute?.("data-cronos-movimientos-estado", estado); region?.setAttribute?.("aria-busy", "false");
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
      anunciar(t("movimientos_rango_invalido")); return;
    }
    void cargar({ periodo: "rango", desde, hasta });
  };
  contenedor.addEventListener("click", alPulsar); contenedor.addEventListener("submit", alEnviar); contenedor.addEventListener("input", alEditar);
  void cargar(consulta);
  const desmontar = () => {
    if (!activa) return;
    activa = false; ++secuencia; controlador?.abort();
    contenedor.removeEventListener("click", alPulsar); contenedor.removeEventListener("submit", alEnviar); contenedor.removeEventListener("input", alEditar); contenedor.remove?.();
  };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ desmontar, consultar: cargar, actualizar });
}
