import { crearTraductorCronos, MENSAJES_CRONOS } from "./i18n.js?v=20260929-i18n-textos-v1";
import { ErrorClienteSaldoCronos, validarConsultaSaldoCronos, validarResultadoSaldoCronos } from "./cliente-saldo-http.js";
import { crearTraductorConsultaCronos, MENSAJES_CONSULTA_CRONOS } from "./i18n-consulta.js?v=20261001-cronos-grafo-bandeja-v5";
import { LOCALIZACION_ACTUAL } from "../../../comun/idioma.js";

const PERIODOS = ["hoy", "semana", "mes", "anio", "rango"];
const ESTADOS = new Set(["disponible", "incompleto", "no_disponible"]);
const MOVIMIENTOS = new Set(["entrada", "salida", "inicio_pausa", "fin_pausa"]);
const ORIGENES = new Set(["remoto", "terminal"]);

function escaparHTML(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}

/** Encabezado de la parte: propio de página, o de tarjeta sin sobrelínea dentro de «Jornada». */
function encabezadoParte(sobrelinea, id, titulo, incrustada) {
  return incrustada ? `<h3 id="${id}">${escaparHTML(titulo)}</h3>`
    : `<p class="sobrelinea">${escaparHTML(sobrelinea)}</p><h2 id="${id}">${escaparHTML(titulo)}</h2>`;
}
function fechaVisible(valor, locale) {
  const [anio, mes, dia] = valor.split("-").map(Number);
  return new Intl.DateTimeFormat(locale, { day: "2-digit", month: "2-digit", year: "numeric", timeZone: "UTC" })
    .format(new Date(Date.UTC(anio, mes - 1, dia, 12)));
}
function minutosVisibles(valor, t) {
  if (valor === null) return t("saldo_estado_no_disponible");
  const signo = valor < 0 ? "−" : "";
  const absoluto = Math.abs(valor);
  return `${signo}${String(Math.floor(absoluto / 60)).padStart(2, "0")}:${String(absoluto % 60).padStart(2, "0")}`;
}
function estadoVisible(codigo, t) { return t(ESTADOS.has(codigo) ? `saldo_estado_${codigo}` : "saldo_estado_no_disponible"); }
function claseEstado(codigo) { return codigo === "disponible" ? "exito" : codigo === "incompleto" ? "aviso" : "neutro"; }
function marcajesVisibles(marcajes, t, locale, zonaHoraria) {
  if (!marcajes.length) return escaparHTML(t("saldo_sin_marcajes"));
  return `<details><summary>${escaparHTML(t("saldo_marcajes"))} (${marcajes.length})</summary><ul>${marcajes.map((marcaje) => {
    const hora = new Intl.DateTimeFormat(locale, { hour: "2-digit", minute: "2-digit", timeZone: zonaHoraria })
      .format(new Date(marcaje.instante_utc));
    const movimiento = t(MOVIMIENTOS.has(marcaje.movimiento) ? `saldo_movimiento_${marcaje.movimiento}` : "saldo_estado_no_disponible");
    const origen = t(ORIGENES.has(marcaje.origen) ? `saldo_origen_${marcaje.origen}` : "saldo_origen_sin_verificar");
    return `<li><time datetime="${escaparHTML(marcaje.instante_utc)}">${escaparHTML(hora)}</time> · ${escaparHTML(movimiento)} · ${escaparHTML(origen)}</li>`;
  }).join("")}</ul></details>`;
}
function calculoVisible(dia, t) {
  const previstos = dia.previstos_minutos === null ? t("saldo_sin_jornada_prevista") : minutosVisibles(dia.previstos_minutos, t);
  const incompleto = dia.estado === "incompleto" ? `<p class="cronos-mensaje">${escaparHTML(t("saldo_calculo_incompleto"))}</p>` : "";
  const referencias = [["saldo_turno_ref", dia.turno_ref], ["saldo_politica_version_ref", dia.politica_version_ref]]
    .filter(([, valor]) => typeof valor === "string" && valor.length > 0);
  const tecnico = referencias.length ? `<details><summary>${escaparHTML(t("saldo_detalle_tecnico"))}</summary><dl>${referencias.map(([clave, valor]) =>
    `<dt>${escaparHTML(t(clave))}</dt><dd>${escaparHTML(valor)}</dd>`).join("")}</dl></details>` : "";
  return `<details><summary>${escaparHTML(t("saldo_calculo_detalle"))}</summary>${incompleto}<dl>
    <dt>${escaparHTML(t("saldo_previsto"))}</dt><dd>${escaparHTML(previstos)}</dd>
    <dt>${escaparHTML(t("saldo_trabajado"))}</dt><dd>${escaparHTML(minutosVisibles(dia.trabajados_minutos, t))}</dd>
    <dt>${escaparHTML(t("saldo_pausas"))}</dt><dd>${escaparHTML(minutosVisibles(dia.pausas_minutos, t))}</dd>
    </dl>${tecnico}</details>`;
}
// Tamaño de presentación; no cambia el periodo autorizado ni sus totales.
function paginarDetalle(detalle, pagina, tamanoPagina) {
  if (!Number.isSafeInteger(tamanoPagina) || tamanoPagina < 1 || tamanoPagina > 367
    || !Number.isSafeInteger(pagina) || pagina < 1) throw new RangeError("página de saldo no válida");
  const paginas = Math.max(1, Math.ceil(detalle.length / tamanoPagina));
  const actual = Math.min(pagina, paginas); const inicio = (actual - 1) * tamanoPagina;
  return { actual, paginas, inicio, filas: detalle.slice(inicio, inicio + tamanoPagina) };
}
function recuentoPagina(paginacion, total, traducir, locale) {
  const numero = (valor) => new Intl.NumberFormat(locale).format(valor);
  return traducir("recuento", { desde: numero(paginacion.inicio + 1), hasta: numero(paginacion.inicio + paginacion.filas.length),
    total: numero(total), pagina: numero(paginacion.actual), paginas: numero(paginacion.paginas) });
}
function tablaDetalle(datos, t, locale, zonaHoraria, filas) {
  if (!datos.detalle.length) return `<p class="cronos-vacio" role="status">${escaparHTML(t("saldo_vacio"))}</p>`;
  return `<div class="cronos-tabla-contenedor"><table class="cronos-tabla">
    <caption>${escaparHTML(t("saldo_detalle"))}</caption>
    <thead><tr><th scope="col">${escaparHTML(t("saldo_fecha"))}</th><th scope="col">${escaparHTML(t("saldo_previsto"))}</th><th scope="col">${escaparHTML(t("saldo_trabajado"))}</th><th scope="col">${escaparHTML(t("saldo_pausas"))}</th><th scope="col">${escaparHTML(t("saldo_diferencia"))}</th><th scope="col">${escaparHTML(t("saldo_estado"))}</th><th scope="col">${escaparHTML(t("saldo_fichajes_calculo"))}</th></tr></thead>
    <tbody>${filas.map((dia) => `<tr><th scope="row"><time datetime="${escaparHTML(dia.fecha)}">${escaparHTML(fechaVisible(dia.fecha, locale))}</time></th>
      <td>${escaparHTML(minutosVisibles(dia.previstos_minutos, t))}</td><td>${escaparHTML(minutosVisibles(dia.trabajados_minutos, t))}</td><td>${escaparHTML(minutosVisibles(dia.pausas_minutos, t))}</td>
      <td><strong>${escaparHTML(minutosVisibles(dia.saldo_minutos, t))}</strong></td>
      <td><span class="cronos-estado cronos-estado-${claseEstado(dia.estado)}">${escaparHTML(estadoVisible(dia.estado, t))}</span></td>
      <td>${marcajesVisibles(dia.marcajes, t, locale, zonaHoraria)}${calculoVisible(dia, t)}</td></tr>`).join("")}</tbody></table></div>`;
}

/** Render puro: los importes de tiempo proceden exclusivamente de la respuesta validada. */
export function renderizarVistaSaldoCronos({ estado = "cargando", consulta = { periodo: "hoy" }, datos = null,
  mensajes = MENSAJES_CRONOS, locale = LOCALIZACION_ACTUAL, zonaHoraria = "Europe/Madrid", incrustada = false, pagina = 1, tamanoPagina = 31,
  mensajesConsulta = MENSAJES_CONSULTA_CRONOS } = {}) {
  const tc = crearTraductorConsultaCronos(mensajesConsulta);
  const t = crearTraductorCronos(mensajes);
  const seleccion = estado === "seleccion" && consulta?.periodo === "rango"
    ? { periodo: "rango", desde: "", hasta: "" } : validarConsultaSaldoCronos(consulta);
  if (estado === "listo") validarResultadoSaldoCronos(datos, seleccion);
  const paginacion = paginarDetalle(estado === "listo" ? datos.detalle : [], pagina, tamanoPagina);
  const controles = estado === "listo" && datos.detalle.length ? `<nav class="cronos-navegacion" aria-label="${escaparHTML(tc("paginacion"))}">
    <button type="button" class="boton-secundario" data-cronos-saldo-pagina="anterior"${paginacion.actual === 1 ? " disabled" : ""}>${escaparHTML(tc("pagina_anterior"))}</button>
    <p data-cronos-saldo-recuento tabindex="-1" role="status" aria-live="polite" aria-atomic="true">${escaparHTML(recuentoPagina(paginacion, datos.detalle.length, tc, locale))}</p>
    <button type="button" class="boton-secundario" data-cronos-saldo-pagina="siguiente"${paginacion.actual === paginacion.paginas ? " disabled" : ""}>${escaparHTML(tc("pagina_siguiente"))}</button></nav>` : "";
  const actualizar = ["listo", "error", "cargando"].includes(estado) ? `<button type="button" class="boton-secundario" data-cronos-saldo-actualizar${estado === "cargando" ? ' aria-disabled="true"' : ""}>${escaparHTML(tc(estado === "error" ? "reintentar" : "actualizar"))}</button>` : "";
  const rango = seleccion.periodo === "rango";
  const estadoClave = estado === "denegado" ? "saldo_denegado" : estado === "error" ? "saldo_error"
    : estado === "no_disponible" ? "saldo_no_disponible"
    : estado === "cargando" ? "saldo_cargando" : estado === "seleccion" ? "saldo_seleccionar_rango"
      : estado === "rango_invalido" ? "saldo_rango_invalido" : "saldo_vacio";
  const resumen = estado === "listo" && datos ? `<div class="rejilla-kpi cronos-indicadores" aria-label="${escaparHTML(t("saldo_titulo"))}">
    ${[["saldo_previsto", datos.resumen.previstos_minutos, "◷"], ["saldo_trabajado", datos.resumen.trabajados_minutos, "◴"], ["saldo_diferencia", datos.resumen.saldo_minutos, "±"]].map(([clave, valor, icono]) => `<article class="tarjeta-kpi"><span class="icono-kpi" aria-hidden="true">${icono}</span><div><span class="etiqueta-kpi">${escaparHTML(t(clave))}</span><strong class="valor-kpi">${escaparHTML(minutosVisibles(valor, t))}</strong></div></article>`).join("")}
  </div><p class="cronos-mensaje" role="status">${escaparHTML(estadoVisible(datos.resumen.estado, t))}</p>` : "";
  const contenido = estado === "listo" && datos ? `${resumen}<p class="cronos-mensaje">${escaparHTML(tc("periodo_completo", { desde: fechaVisible(datos.periodo.desde, locale), hasta: fechaVisible(datos.periodo.hasta, locale) }))}</p><section class="panel cronos-panel" aria-labelledby="cronos-saldo-detalle-titulo"><div class="cabecera-panel"><h3 id="cronos-saldo-detalle-titulo">${escaparHTML(t("saldo_detalle"))}</h3></div>${tablaDetalle(datos, t, locale, zonaHoraria, paginacion.filas)}<div class="cuerpo-panel">${controles}</div></section>`
    : `<section class="panel cronos-panel"><div class="cabecera-panel"><h3>${escaparHTML(t("saldo_detalle"))}</h3></div><p class="cronos-${estado === "denegado" ? "acceso-denegado" : "vacio"}" data-cronos-saldo-mensaje tabindex="-1" role="${estado === "error" ? "alert" : "status"}">${escaparHTML(t(estadoClave))}</p></section>`;
  return `<section class="cronos-area cronos-saldo-conectado" aria-labelledby="cronos-saldo-titulo" data-cronos-saldo-estado="${escaparHTML(estado)}" aria-busy="${estado === "cargando"}">
    <header class="cronos-encabezado"><div>${encabezadoParte(t("sobrelinea"), "cronos-saldo-titulo", t("saldo_titulo"), incrustada)}</div>
      <button type="button" class="cronos-boton-ayuda" data-accion="ayuda" aria-label="${escaparHTML(t("abrir_ayuda", { asunto: t("saldo_titulo") }))}" title="${escaparHTML(t("abrir_ayuda", { asunto: t("saldo_titulo") }))}"><span aria-hidden="true">?</span></button></header>
    <section class="panel cronos-panel" aria-label="${escaparHTML(t("saldo_periodos"))}"><div class="cabecera-panel"><h3>${escaparHTML(t("saldo_periodos"))}</h3>${actualizar}</div><div class="cuerpo-panel">
      <nav class="cronos-navegacion" aria-label="${escaparHTML(t("saldo_periodos"))}">${PERIODOS.map((periodo) => `<button type="button" data-cronos-saldo-periodo="${periodo}" aria-pressed="${seleccion.periodo === periodo}">${escaparHTML(t(`saldo_${periodo}`))}</button>`).join("")}</nav>
      <form data-cronos-saldo-rango ${rango ? "" : "hidden"}><label>${escaparHTML(t("saldo_desde"))}<input type="date" name="desde" value="${escaparHTML(rango ? seleccion.desde : "")}" ${rango ? "required" : ""}></label><label>${escaparHTML(t("saldo_hasta"))}<input type="date" name="hasta" value="${escaparHTML(rango ? seleccion.hasta : "")}" ${rango ? "required" : ""}></label><button type="submit" class="boton-primario">${escaparHTML(t("saldo_consultar"))}</button><p data-cronos-saldo-validacion role="alert" hidden></p></form>
    </div></section>${contenido}</section>`;
}

/** Montaje aislado; aborta cada consulta anterior y descarta sus respuestas tardías. */
export function montarVistaSaldoCronos({ raiz, cliente, mensajes = MENSAJES_CRONOS, anunciar = () => {}, registrarDesmontar,
  locale = LOCALIZACION_ACTUAL, zonaHoraria = "Europe/Madrid", incrustada = false, tamanoPagina = 31,
  mensajesConsulta = MENSAJES_CONSULTA_CRONOS } = {}) {
  paginarDetalle([], 1, tamanoPagina);
  const tc = crearTraductorConsultaCronos(mensajesConsulta);
  if (!raiz?.append || !raiz.ownerDocument?.createElement || typeof cliente?.consultar !== "function"
    || typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) throw new TypeError("montaje de saldo Cronos no disponible");
  const contenedor = raiz.ownerDocument.createElement("section");
  contenedor.dataset.cronosSaldo = ""; raiz.append(contenedor);
  const t = crearTraductorCronos(mensajes);
  let activa = true; let secuencia = 0; let controlador = null; let consulta = { periodo: "hoy" };
  let pagina = 1; let datos = null; let estado = "cargando";
  const selectorFoco = () => {
    const foco = raiz.ownerDocument.activeElement;
    if (!foco || !contenedor.contains?.(foco)) return null;
    for (const atributo of ["data-cronos-saldo-periodo", "data-cronos-saldo-pagina", "data-cronos-saldo-actualizar", "data-cronos-saldo-recuento", "data-cronos-saldo-mensaje", "data-accion", "name"]) {
      const valor = foco.getAttribute?.(atributo);
      if (valor !== null && valor !== undefined && /^[a-z_]*$/u.test(valor)) return `[${atributo}="${valor}"]`;
    }
    return null;
  };
  const dibujar = () => {
    if (!activa) return;
    const selector = selectorFoco();
    contenedor.innerHTML = renderizarVistaSaldoCronos({ estado, consulta, datos, mensajes, mensajesConsulta,
      locale, zonaHoraria, incrustada, pagina, tamanoPagina });
    if (selector) {
      const destino = contenedor.querySelector?.(selector);
      (destino && !destino.disabled ? destino : contenedor.querySelector?.("[data-cronos-saldo-recuento]")
        ?? contenedor.querySelector?.("[data-cronos-saldo-actualizar]")
        ?? contenedor.querySelector?.("[data-cronos-saldo-mensaje]"))?.focus?.();
    }
  };
  const cargar = async (siguiente, conservarPagina = false) => {
    if (!activa) return;
    consulta = validarConsultaSaldoCronos(siguiente);
    if (!conservarPagina) pagina = 1;
    controlador?.abort(); controlador = new AbortController();
    const turno = ++secuencia; estado = "cargando"; datos = null; dibujar();
    try {
      const respuesta = await cliente.consultar(consulta, { signal: controlador.signal });
      if (!activa || turno !== secuencia) return;
      datos = validarResultadoSaldoCronos(respuesta, consulta);
      pagina = paginarDetalle(datos.detalle, pagina, tamanoPagina).actual;
      estado = "listo"; dibujar(); anunciar(tc("consulta_lista"));
    } catch (error) {
      if (!activa || turno !== secuencia || controlador.signal.aborted) return;
      datos = null;
      estado = !(error instanceof ErrorClienteSaldoCronos) ? "error" : error.codigo === "acceso_denegado" ? "denegado"
        : error.estado === 404 ? "no_disponible" : "error";
      dibujar(); anunciar(t(estado === "denegado" ? "saldo_denegado" : estado === "no_disponible" ? "saldo_no_disponible" : "saldo_error"));
    }
  };
  // Un fichaje confirmado puede llegar mientras se lee: sustituye esa
  // lectura anterior, sin cambiar su periodo ni la página seleccionada.
  const actualizar = () => ["listo", "error", "cargando"].includes(estado) ? cargar(consulta, true) : Promise.resolve();
  const alPulsar = (evento) => {
    const accion = evento.target?.closest?.("[data-cronos-saldo-actualizar]");
    if (accion) {
      if (estado !== "cargando") void actualizar();
      return;
    }
    const cambio = evento.target?.closest?.("[data-cronos-saldo-pagina]");
    if (cambio && estado === "listo" && !cambio.disabled) {
      const paso = cambio.dataset.cronosSaldoPagina;
      if (!["anterior", "siguiente"].includes(paso)) return;
      pagina = paginarDetalle(datos.detalle, Math.max(1, pagina + (paso === "anterior" ? -1 : 1)), tamanoPagina).actual;
      dibujar(); anunciar(recuentoPagina(paginarDetalle(datos.detalle, pagina, tamanoPagina), datos.detalle.length, tc, locale)); return;
    }
    const boton = evento.target?.closest?.("[data-cronos-saldo-periodo]");
    if (!boton) return;
    const periodo = boton.dataset.cronosSaldoPeriodo;
    if (!PERIODOS.includes(periodo)) return;
    if (periodo === "rango") {
      controlador?.abort(); ++secuencia; pagina = 1; datos = null;
      consulta = { periodo: "rango", desde: "", hasta: "" }; estado = "seleccion";
      dibujar(); contenedor.querySelector?.("[name=desde]")?.focus?.(); return;
    }
    void cargar({ periodo });
  };
  const alEnviar = (evento) => {
    if (!evento.target?.matches?.("[data-cronos-saldo-rango]")) return;
    evento.preventDefault();
    const desde = evento.target.elements?.namedItem?.("desde")?.value;
    const hasta = evento.target.elements?.namedItem?.("hasta")?.value;
    try { validarConsultaSaldoCronos({ periodo: "rango", desde, hasta }); }
    catch {
      const validacion = contenedor.querySelector?.("[data-cronos-saldo-validacion]");
      if (validacion) { validacion.textContent = t("saldo_rango_invalido"); validacion.hidden = false; }
      anunciar(t("saldo_rango_invalido")); return;
    }
    void cargar({ periodo: "rango", desde, hasta });
  };
  contenedor.addEventListener("click", alPulsar);
  contenedor.addEventListener("submit", alEnviar);
  void cargar(consulta);
  const desmontar = () => {
    if (!activa) return;
    activa = false; ++secuencia; controlador?.abort();
    contenedor.removeEventListener("click", alPulsar); contenedor.removeEventListener("submit", alEnviar);
    contenedor.remove?.();
  };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ desmontar, consultar: cargar, actualizar });
}
