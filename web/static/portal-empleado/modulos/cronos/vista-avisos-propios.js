import { crearTraductorResolucionCronos, periodoSolicitudCronos } from "./i18n-resolucion.js";
import { formatearCantidadCronos } from "./i18n-solicitudes.js";
import { ErrorClienteResolucionCronos, crearClienteResolucionCronosHTTP } from "./cliente-resolucion-http.js";

const FILTROS = Object.freeze(["recibidos", "archivados"]);

function escaparHTML(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}
function instanteVisible(valor, locale, zonaHoraria) {
  return new Intl.DateTimeFormat(locale, { timeZone: zonaHoraria, dateStyle: "medium" }).format(new Date(valor));
}
function claveNueva() { return globalThis.crypto.randomUUID(); }

function aviso(a, filtro, archivando, t, locale, zonaHoraria) {
  const texto = t(a.estado === "concedido" ? "aviso_concedido" : "aviso_denegado", {
    permiso: a.nombre, periodo: periodoSolicitudCronos(a, t, locale), cantidad: formatearCantidadCronos(a.cantidad, a.unidad, t, locale) });
  const motivo = a.motivo ? `<p class="cronos-aviso-motivo">${escaparHTML(t("aviso_motivo", { motivo: a.motivo }))}</p>` : "";
  const fecha = a.archivado ? t("aviso_archivado_el", { fecha: instanteVisible(a.archivado_en, locale, zonaHoraria) })
    : t("aviso_resuelto", { fecha: instanteVisible(a.resuelto_en, locale, zonaHoraria) });
  const boton = filtro === "recibidos"
    ? `<button type="button" class="boton-secundario" data-cronos-archivar="${escaparHTML(a.aviso_ref)}" aria-label="${escaparHTML(t("archivar_aviso", { permiso: a.nombre }))}"${archivando ? " disabled" : ""}>${escaparHTML(t("archivar"))}</button>` : "";
  return `<li class="cronos-aviso" data-estado="${escaparHTML(a.estado)}"><span class="cronos-aviso-icono" aria-hidden="true"></span>
    <div class="cronos-aviso-texto"><p>${escaparHTML(texto)}</p>${motivo}<p class="cronos-aviso-fecha">${escaparHTML(fecha)}</p></div>${boton}</li>`;
}

/** Avisos de resolución de los permisos propios, recibidos o archivados. */
export function renderizarAvisosPropiosCronos({ estado = "cargando", filtro = "recibidos", datos = null, mensaje = "", tonoMensaje = "exito", archivando = "",
  mensajes, locale = "es-ES", zonaHoraria = "Europe/Madrid" } = {}) {
  const t = crearTraductorResolucionCronos(mensajes);
  if (!FILTROS.includes(filtro)) throw new RangeError("filtro de avisos no válido");
  const ayuda = t("abrir_ayuda", { asunto: t("avisos_titulo") });
  const cabecera = `<header class="cronos-encabezado"><div><p class="sobrelinea">${escaparHTML(t("sobrelinea"))}</p><h2 id="cronos-avisos-titulo">${escaparHTML(t("avisos_titulo"))}</h2></div>
    <button type="button" class="cronos-boton-ayuda" data-accion="ayuda" aria-label="${escaparHTML(ayuda)}" title="${escaparHTML(ayuda)}"><span aria-hidden="true">?</span></button></header>`;
  const selector = `<div class="cronos-selector-paso" role="group" aria-label="${escaparHTML(t("avisos_titulo"))}">${FILTROS.map((f) =>
    `<button type="button" class="boton-secundario" data-cronos-filtro="${f}" aria-pressed="${f === filtro}">${escaparHTML(t(`avisos_${f}`))}</button>`).join("")}</div>`;
  const cabeceraPanel = `<div class="cabecera-panel"><h3 id="cronos-avisos-filtro">${escaparHTML(t(`avisos_${filtro}`))}</h3>${selector}</div>`;
  if (estado !== "listo") {
    const clave = { denegado: "denegado", sin_empleado: "sin_empleado", error: "error" }[estado] ?? "cargando";
    return `<section class="cronos-area cronos-avisos-propios" aria-labelledby="cronos-avisos-titulo" data-estado="${escaparHTML(estado)}">${cabecera}
      <section class="panel cronos-panel" aria-labelledby="cronos-avisos-filtro">${cabeceraPanel}<div class="cuerpo-panel"><p class="cronos-${estado === "cargando" ? "vacio" : "acceso-denegado"}" role="${estado === "error" ? "alert" : "status"}">${escaparHTML(t(clave))}</p></div></section></section>`;
  }
  const tono = tonoMensaje === "error" ? "error" : "exito";
  const avisoMensaje = mensaje ? `<p class="cronos-solicitud-aviso" data-tono="${tono}" role="${tono === "error" ? "alert" : "status"}">${escaparHTML(mensaje)}</p>` : "";
  const visibles = datos.avisos.filter((a) => a.archivado === (filtro === "archivados"));
  const lista = visibles.length ? `<ul class="cronos-avisos-lista">${visibles.map((a) => aviso(a, filtro, archivando === a.aviso_ref, t, locale, zonaHoraria)).join("")}</ul>`
    : `<p class="cronos-vacio" role="status">${escaparHTML(t("avisos_vacio"))}</p>`;
  return `<section class="cronos-area cronos-avisos-propios" aria-labelledby="cronos-avisos-titulo" data-estado="listo">${cabecera}
    <section class="panel cronos-panel" aria-labelledby="cronos-avisos-filtro">${cabeceraPanel}<div class="cuerpo-panel">${avisoMensaje}${lista}</div></section></section>`;
}

function estadoError(error) {
  if (error instanceof ErrorClienteResolucionCronos) {
    if (error.codigo === "sin_empleado") return "sin_empleado";
    if (["acceso_denegado", "autenticacion_requerida"].includes(error.codigo)) return "denegado";
  }
  return "error";
}

export function montarAvisosPropiosCronos({ raiz, cliente = crearClienteResolucionCronosHTTP(), mensajes, anunciar = () => {}, registrarDesmontar,
  locale = "es-ES", zonaHoraria = "Europe/Madrid" } = {}) {
  if (!raiz?.append || !raiz.ownerDocument?.createElement || typeof cliente?.consultarAvisos !== "function" || typeof cliente?.archivarAviso !== "function"
    || typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) throw new TypeError("montaje de avisos Cronos no disponible");
  const t = crearTraductorResolucionCronos(mensajes);
  const contenedor = raiz.ownerDocument.createElement("section"); contenedor.dataset.cronosAvisosPropios = ""; raiz.append(contenedor);
  let activa = true; let secuencia = 0; let controlador = null; let envio = null;
  let filtro = "recibidos"; let estado = "cargando"; let datos = null; let mensaje = ""; let tonoMensaje = "exito";
  // Una sola operación de archivo a la vez; su clave se conserva en el reintento.
  let archivo = null;
  const dibujar = () => {
    if (activa) contenedor.innerHTML = renderizarAvisosPropiosCronos({ estado, filtro, datos, mensaje, tonoMensaje, archivando: archivo?.enCurso ? archivo.avisoRef : "", mensajes, locale, zonaHoraria });
  };
  const cargar = async () => {
    controlador?.abort(); controlador = new AbortController(); const turno = ++secuencia;
    estado = "cargando"; datos = null; dibujar();
    try {
      const r = await cliente.consultarAvisos({ signal: controlador.signal });
      if (!activa || turno !== secuencia) return;
      estado = "listo"; datos = r; dibujar();
    } catch (error) {
      if (!activa || turno !== secuencia || controlador.signal.aborted) return;
      estado = estadoError(error); dibujar(); anunciar(t(estado));
    }
  };
  const archivar = async (avisoRef) => {
    if (archivo?.enCurso || !datos?.avisos.some((a) => a.aviso_ref === avisoRef && !a.archivado)) return;
    if (!archivo || archivo.avisoRef !== avisoRef) archivo = { avisoRef, clave: claveNueva() };
    archivo.enCurso = true; mensaje = ""; dibujar();
    envio = new AbortController();
    try {
      const recibo = await cliente.archivarAviso({ clave_operacion: archivo.clave, aviso_ref: avisoRef }, { signal: envio.signal });
      if (!activa) return;
      archivo = null; mensaje = t(recibo.replay ? "aviso_ya_archivado" : "aviso_archivado"); tonoMensaje = "exito"; anunciar(mensaje);
      await cargar();
    } catch (error) {
      if (!activa || envio.signal.aborted) return;
      const codigo = error instanceof ErrorClienteResolucionCronos ? error.codigo : "";
      archivo.enCurso = false;
      if (codigo === "estado_cambiado") { archivo = null; mensaje = t("aviso_ya_archivado"); tonoMensaje = "exito"; anunciar(mensaje); await cargar(); return; }
      mensaje = t("error_archivar"); tonoMensaje = "error"; anunciar(mensaje); dibujar();
    }
  };
  const alPulsar = (evento) => {
    const botonFiltro = evento.target?.closest?.("[data-cronos-filtro]");
    if (botonFiltro && FILTROS.includes(botonFiltro.dataset.cronosFiltro)) { filtro = botonFiltro.dataset.cronosFiltro; mensaje = ""; dibujar(); return; }
    const botonArchivar = evento.target?.closest?.("[data-cronos-archivar]");
    if (botonArchivar && !botonArchivar.disabled) void archivar(botonArchivar.dataset.cronosArchivar);
  };
  contenedor.addEventListener("click", alPulsar);
  void cargar();
  const desmontar = () => {
    if (!activa) return;
    activa = false; ++secuencia; controlador?.abort(); envio?.abort();
    contenedor.removeEventListener("click", alPulsar); contenedor.remove?.();
  };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ desmontar, recargar: cargar });
}
