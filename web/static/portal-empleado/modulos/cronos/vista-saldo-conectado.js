import { crearTraductorCronos, MENSAJES_CRONOS_ES } from "./i18n.js?v=20260925-cronos-p2-v1";
import { ErrorClienteSaldoCronos, validarConsultaSaldoCronos, validarResultadoSaldoCronos } from "./cliente-saldo-http.js";

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
function tablaDetalle(datos, t, locale, zonaHoraria) {
  if (!datos.detalle.length) return `<p class="cronos-vacio" role="status">${escaparHTML(t("saldo_vacio"))}</p>`;
  return `<div class="cronos-tabla-contenedor"><table class="cronos-tabla">
    <caption>${escaparHTML(t("saldo_detalle"))}</caption>
    <thead><tr><th scope="col">${escaparHTML(t("saldo_fecha"))}</th><th scope="col">${escaparHTML(t("saldo_previsto"))}</th><th scope="col">${escaparHTML(t("saldo_trabajado"))}</th><th scope="col">${escaparHTML(t("saldo_pausas"))}</th><th scope="col">${escaparHTML(t("saldo_diferencia"))}</th><th scope="col">${escaparHTML(t("saldo_estado"))}</th><th scope="col">${escaparHTML(t("saldo_marcajes"))}</th></tr></thead>
    <tbody>${datos.detalle.map((dia) => `<tr><th scope="row"><time datetime="${escaparHTML(dia.fecha)}">${escaparHTML(fechaVisible(dia.fecha, locale))}</time></th>
      <td>${escaparHTML(minutosVisibles(dia.previstos_minutos, t))}</td><td>${escaparHTML(minutosVisibles(dia.trabajados_minutos, t))}</td><td>${escaparHTML(minutosVisibles(dia.pausas_minutos, t))}</td>
      <td><strong>${escaparHTML(minutosVisibles(dia.saldo_minutos, t))}</strong></td>
      <td><span class="cronos-estado cronos-estado-${claseEstado(dia.estado)}">${escaparHTML(estadoVisible(dia.estado, t))}</span></td>
      <td>${marcajesVisibles(dia.marcajes, t, locale, zonaHoraria)}</td></tr>`).join("")}</tbody></table></div>`;
}

/** Render puro: los importes de tiempo proceden exclusivamente de la respuesta validada. */
export function renderizarVistaSaldoCronos({ estado = "cargando", consulta = { periodo: "hoy" }, datos = null,
  mensajes = MENSAJES_CRONOS_ES, locale = "es-ES", zonaHoraria = "Europe/Madrid", incrustada = false } = {}) {
  const t = crearTraductorCronos(mensajes);
  const seleccion = estado === "seleccion" && consulta?.periodo === "rango"
    ? { periodo: "rango", desde: "", hasta: "" } : validarConsultaSaldoCronos(consulta);
  if (estado === "listo") validarResultadoSaldoCronos(datos, seleccion);
  const rango = seleccion.periodo === "rango";
  const estadoClave = estado === "denegado" ? "saldo_denegado" : estado === "error" ? "saldo_error"
    : estado === "no_disponible" ? "saldo_no_disponible"
    : estado === "cargando" ? "saldo_cargando" : estado === "seleccion" ? "saldo_seleccionar_rango"
      : estado === "rango_invalido" ? "saldo_rango_invalido" : "saldo_vacio";
  const resumen = estado === "listo" && datos ? `<div class="rejilla-kpi cronos-indicadores" aria-label="${escaparHTML(t("saldo_titulo"))}">
    ${[["saldo_previsto", datos.resumen.previstos_minutos, "◷"], ["saldo_trabajado", datos.resumen.trabajados_minutos, "◴"], ["saldo_diferencia", datos.resumen.saldo_minutos, "±"]].map(([clave, valor, icono]) => `<article class="tarjeta-kpi"><span class="icono-kpi" aria-hidden="true">${icono}</span><div><span class="etiqueta-kpi">${escaparHTML(t(clave))}</span><strong class="valor-kpi">${escaparHTML(minutosVisibles(valor, t))}</strong></div></article>`).join("")}
  </div><p class="cronos-mensaje" role="status">${escaparHTML(estadoVisible(datos.resumen.estado, t))}</p>` : "";
  const contenido = estado === "listo" && datos ? `${resumen}<section class="panel cronos-panel" aria-labelledby="cronos-saldo-detalle-titulo"><div class="cabecera-panel"><h3 id="cronos-saldo-detalle-titulo">${escaparHTML(t("saldo_detalle"))}</h3></div>${tablaDetalle(datos, t, locale, zonaHoraria)}</section>`
    : `<section class="panel cronos-panel"><div class="cabecera-panel"><h3>${escaparHTML(t("saldo_detalle"))}</h3></div><p class="cronos-${estado === "denegado" ? "acceso-denegado" : "vacio"}" role="${estado === "error" ? "alert" : "status"}">${escaparHTML(t(estadoClave))}</p></section>`;
  return `<section class="cronos-area cronos-saldo-conectado" aria-labelledby="cronos-saldo-titulo" data-cronos-saldo-estado="${escaparHTML(estado)}">
    <header class="cronos-encabezado"><div>${encabezadoParte(t("sobrelinea"), "cronos-saldo-titulo", t("saldo_titulo"), incrustada)}</div>
      <button type="button" class="cronos-boton-ayuda" data-accion="ayuda" aria-label="${escaparHTML(t("abrir_ayuda", { asunto: t("saldo_titulo") }))}" title="${escaparHTML(t("abrir_ayuda", { asunto: t("saldo_titulo") }))}"><span aria-hidden="true">?</span></button></header>
    <section class="panel cronos-panel" aria-label="${escaparHTML(t("saldo_periodos"))}"><div class="cabecera-panel"><h3>${escaparHTML(t("saldo_periodos"))}</h3></div><div class="cuerpo-panel">
      <nav class="cronos-navegacion" aria-label="${escaparHTML(t("saldo_periodos"))}">${PERIODOS.map((periodo) => `<button type="button" data-cronos-saldo-periodo="${periodo}" aria-pressed="${seleccion.periodo === periodo}">${escaparHTML(t(`saldo_${periodo}`))}</button>`).join("")}</nav>
      <form data-cronos-saldo-rango ${rango ? "" : "hidden"}><label>${escaparHTML(t("saldo_desde"))}<input type="date" name="desde" value="${escaparHTML(rango ? seleccion.desde : "")}" ${rango ? "required" : ""}></label><label>${escaparHTML(t("saldo_hasta"))}<input type="date" name="hasta" value="${escaparHTML(rango ? seleccion.hasta : "")}" ${rango ? "required" : ""}></label><button type="submit" class="boton-primario">${escaparHTML(t("saldo_consultar"))}</button><p data-cronos-saldo-validacion role="alert" hidden></p></form>
    </div></section>${contenido}</section>`;
}

/** Montaje aislado; aborta cada consulta anterior y descarta sus respuestas tardías. */
export function montarVistaSaldoCronos({ raiz, cliente, mensajes = MENSAJES_CRONOS_ES, anunciar = () => {}, registrarDesmontar,
  locale = "es-ES", zonaHoraria = "Europe/Madrid", incrustada = false } = {}) {
  if (!raiz?.append || !raiz.ownerDocument?.createElement || typeof cliente?.consultar !== "function"
    || typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) throw new TypeError("montaje de saldo Cronos no disponible");
  const contenedor = raiz.ownerDocument.createElement("section");
  contenedor.dataset.cronosSaldo = ""; raiz.append(contenedor);
  const t = crearTraductorCronos(mensajes);
  let activa = true; let secuencia = 0; let controlador = null; let consulta = { periodo: "hoy" };
  const dibujar = (estado, datos) => {
    if (activa) contenedor.innerHTML = renderizarVistaSaldoCronos({ estado, consulta, datos, mensajes, locale, zonaHoraria, incrustada });
  };
  const cargar = async (siguiente) => {
    consulta = validarConsultaSaldoCronos(siguiente);
    controlador?.abort(); controlador = new AbortController();
    const turno = ++secuencia; dibujar("cargando");
    try {
      const datos = await cliente.consultar(consulta, { signal: controlador.signal });
      if (!activa || turno !== secuencia) return;
      dibujar("listo", datos);
    } catch (error) {
      if (!activa || turno !== secuencia || controlador.signal.aborted) return;
      const estado = !(error instanceof ErrorClienteSaldoCronos) ? "error" : error.codigo === "acceso_denegado" ? "denegado"
        : error.estado === 404 ? "no_disponible" : "error";
      dibujar(estado); anunciar(estado);
    }
  };
  const alPulsar = (evento) => {
    const boton = evento.target?.closest?.("[data-cronos-saldo-periodo]");
    if (!boton) return;
    const periodo = boton.dataset.cronosSaldoPeriodo;
    if (!PERIODOS.includes(periodo)) return;
    if (periodo === "rango") {
      controlador?.abort(); ++secuencia;
      consulta = { periodo: "rango", desde: "", hasta: "" };
      dibujar("seleccion");
      contenedor.querySelector?.("[name=desde]")?.focus?.();
      return;
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
      anunciar("rango_invalido"); return;
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
  return Object.freeze({ desmontar, consultar: cargar });
}
