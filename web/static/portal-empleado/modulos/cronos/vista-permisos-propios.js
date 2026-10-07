import { crearTraductorSolicitudesCronos, formatearCantidadCronos, MENSAJES_CRONOS_SOLICITUDES } from "./i18n-solicitudes.js";
import { ErrorClienteSolicitudesCronos, crearClienteSolicitudesCronosHTTP, validarPermisosPropiosCronos, validarEntradaPermisoCronos } from "./cliente-solicitudes-http.js";
import { hoyCivilCronos } from "./fecha-civil.js";
import { crearTraductorHistorialCronos, MENSAJES_HISTORIAL_CRONOS } from "./i18n-historial.js?v=20261001-cronos-historial-v1";
import { LOCALIZACION_ACTUAL } from "../../../comun/idioma.js";
import { crearTraductorJustificacionCronos, MENSAJES_JUSTIFICACION_CRONOS } from "./i18n-permisos.js?v=20261001-cronos-grafo-bandeja-v5";

import { crearTraductorConsultaPermisosCronos, MENSAJES_CONSULTA_PERMISOS_CRONOS } from "./i18n-permisos-consulta.js?v=20261001-cronos-c7-consulta-v2";

const ERRORES = new Map([
  ["peticion_invalida", "error_peticion_invalida"], ["conflicto", "error_conflicto"],
  ["permiso_no_solicitable", "error_permiso_no_solicitable"], ["calendario_no_publicado", "error_calendario_no_publicado"],
  ["fuera_de_limites", "error_fuera_de_limites"], ["solapado", "error_solapado"],
]);

function escaparHTML(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}
function fechaVisible(fecha, locale) {
  const [a, m, d] = fecha.split("-").map(Number);
  return new Intl.DateTimeFormat(locale, { timeZone: "UTC", dateStyle: "medium" }).format(new Date(Date.UTC(a, m - 1, d, 12)));
}
function claveNueva() { return globalThis.crypto.randomUUID(); }

function maximo(p, t, locale) {
  if (p.maximo_anual !== null) return formatearCantidadCronos(p.maximo_anual, p.unidad, t, locale);
  if (p.maximo_mensual !== null) return t("por_mes", { valor: formatearCantidadCronos(p.maximo_mensual, p.unidad, t, locale) });
  if (p.maximo_solicitud !== null) return t("por_solicitud", { valor: formatearCantidadCronos(p.maximo_solicitud, p.unidad, t, locale) });
  return t("sin_limite");
}

const VIVAS = new Set(["solicitado", "pendiente_administracion", "concedido"]);

/** Resta anual; con solo cupo mensual, lo que queda del mes en curso (si se mira el año en curso). */
function resta(p, solicitudes, anio, hoy, t, locale) {
  if (p.sin_conciliar) return `<span class="cronos-estado cronos-estado-aviso">${escaparHTML(t("resta_sin_conciliar"))}</span>`;
  if (p.resta !== null) return escaparHTML(formatearCantidadCronos(p.resta, p.unidad, t, locale));
  if (p.maximo_mensual === null || typeof hoy !== "string" || !hoy.startsWith(`${anio}-`)) return escaparHTML(t("sin_limite"));
  const mes = hoy.slice(0, 7);
  const usado = solicitudes.filter((s) => s.permiso_ref === p.permiso_ref && s.unidad === p.unidad && s.desde.startsWith(mes) && VIVAS.has(s.estado))
    .reduce((suma, s) => suma + s.cantidad, 0);
  return escaparHTML(t("resta_mes", { valor: formatearCantidadCronos(Math.max(0, p.maximo_mensual - usado), p.unidad, t, locale) }));
}

function periodo(s, t, locale) {
  return s.hora_inicio ? t("periodo_horas", { fecha: fechaVisible(s.desde, locale), inicio: s.hora_inicio, fin: s.hora_fin })
    : t("periodo_dias", { desde: fechaVisible(s.desde, locale), hasta: fechaVisible(s.hasta, locale) });
}

/**
 * Punto del circuito en que está la solicitud. El circuito lo aplica el
 * servidor (jefatura y después RRHH salvo marca directa), nunca el catálogo;
 * si el servidor no lo indica, sólo se dice que está pendiente de resolver.
 */
export function estadoSolicitudPermisoCronos(s) {
  switch (s.estado) {
    case "concedido": return "estado_concedido";
    case "denegado": return "estado_permiso_denegado";
    case "cancelado": return "estado_permiso_cancelado";
    case "pendiente_administracion": return "estado_permiso_pendiente_rrhh";
    case "solicitado":
      if (s.pendiente_asignacion === true) return "estado_permiso_pendiente_asignacion";
      if (s.circuito === "A") return "estado_permiso_pendiente_rrhh";
      if (s.circuito === "J-A") return "estado_permiso_pendiente_jefatura";
      return "estado_permiso_pendiente";
    default: return "estado_permiso_pendiente";
  }
}

function tabla(cabeceras, filas, t, vacio = t("sin_filas")) {
  const cuerpo = filas.length ? filas.join("") : `<tr><td colspan="${cabeceras.length}">${escaparHTML(vacio)}</td></tr>`;
  return `<div class="cronos-tabla-contenedor"><table class="cronos-tabla"><thead><tr>${cabeceras.map((c) => `<th scope="col"${c.startsWith("col_m") || ["col_solicitado", "col_concedido", "col_resta", "duracion"].includes(c) ? ' class="numero"' : ""}>${escaparHTML(t(c))}</th>`).join("")}</tr></thead><tbody>${cuerpo}</tbody></table></div>`;
}

const FILTROS_HISTORIAL = Object.freeze(["todas", "pendientes", "concedidos", "denegados", "cancelados"]);
function filtrarSolicitudes(solicitudes, filtro) {
  if (!FILTROS_HISTORIAL.includes(filtro)) throw new RangeError("filtro de historial no válido");
  return solicitudes.filter((s) => filtro === "todas" || (filtro === "pendientes"
    ? ["solicitado", "pendiente_administracion"].includes(s.estado) : s.estado === filtro.slice(0, -1)));
}
function paginarHistorial(solicitudes, filtro, pagina, tamanoPagina) {
  if (!Number.isSafeInteger(pagina) || pagina < 1 || !Number.isSafeInteger(tamanoPagina)
    || tamanoPagina < 1 || tamanoPagina > 5000) throw new RangeError("página de historial no válida");
  const filtradas = filtrarSolicitudes(solicitudes, filtro);
  const paginas = Math.max(1, Math.ceil(filtradas.length / tamanoPagina));
  const actual = Math.min(pagina, paginas); const inicio = (actual - 1) * tamanoPagina;
  return { total: filtradas.length, actual, paginas, inicio, filas: filtradas.slice(inicio, inicio + tamanoPagina) };
}
function recuentoHistorial(pagina, t, locale) {
  const n = (valor) => new Intl.NumberFormat(locale).format(valor);
  return t("recuento", { desde: n(pagina.total ? pagina.inicio + 1 : 0), hasta: n(pagina.inicio + pagina.filas.length),
    total: n(pagina.total), pagina: n(pagina.actual), paginas: n(pagina.paginas) });
}
// Revalida el recibo al usar un cliente inyectado: no basta una promesa resuelta.
function validarReciboSolicitud(recibo, entrada, unidadEsperada) {
  const claves = ["solicitud_ref", "recibo_ref", "catalogo_version_ref", "version", "estado", "cantidad", "unidad", "instante_utc", "replay"];
  const ref = (valor, prefijo) => typeof valor === "string" && valor.startsWith(prefijo) && /^[A-Za-z0-9:._-]{1,200}$/u.test(valor);
  if (!recibo || Object.keys(recibo).length !== claves.length || claves.some((clave) => !Object.hasOwn(recibo, clave))
    || recibo.solicitud_ref !== `permiso:cronos:solicitud:${entrada.clave_operacion}` || !ref(recibo.recibo_ref, "recibo:cronos:")
    || !ref(recibo.catalogo_version_ref, "catalogo:cronos:") || recibo.version !== 1 || recibo.estado !== "solicitado"
    || !Number.isSafeInteger(recibo.cantidad) || recibo.cantidad < 1 || recibo.unidad !== unidadEsperada
    || typeof recibo.instante_utc !== "string" || !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?(?:Z|[+-]\d{2}:\d{2})$/u.test(recibo.instante_utc)
    || !Number.isFinite(Date.parse(recibo.instante_utc)) || typeof recibo.replay !== "boolean") throw new ErrorClienteSolicitudesCronos("respuesta_incompatible", 200);
  return Object.freeze({ ...recibo });
}
function panelRecibo(recibo, t, locale, zonaHoraria) {
  if (!recibo) return "";
  const fecha = new Intl.DateTimeFormat(locale, { timeZone: zonaHoraria, dateStyle: "medium", timeStyle: "short" }).format(new Date(recibo.instante_utc));
  return `<section class="panel cronos-panel" aria-labelledby="cronos-ultimo-recibo"><div class="cabecera-panel"><h3 id="cronos-ultimo-recibo">${escaparHTML(t("recibo_titulo"))}</h3></div>
    <div class="cuerpo-panel"><details><summary>${escaparHTML(t("recibo_detalle"))}</summary><dl><dt>${escaparHTML(t("recibo_referencia"))}</dt><dd>${escaparHTML(recibo.recibo_ref)}</dd>
    <dt>${escaparHTML(t("recibo_fecha"))}</dt><dd><time datetime="${escaparHTML(recibo.instante_utc)}">${escaparHTML(fecha)}</time></dd></dl></details></div></section>`;
}

function formularioSolicitud(f, permiso, t) {
  if (!f || !permiso) return "";
  const enviando = f.estado === "enviando";
  const campos = permiso.unidad === "hora"
    ? `<label>${escaparHTML(t("fecha"))}<input class="control-formulario" type="date" name="desde" required value="${escaparHTML(f.desde ?? "")}"${enviando ? " disabled" : ""}></label>
       <label>${escaparHTML(t("hora_inicio"))}<input class="control-formulario" type="time" name="hora_inicio" required value="${escaparHTML(f.hora_inicio ?? "")}"${enviando ? " disabled" : ""}></label>
       <label>${escaparHTML(t("hora_fin"))}<input class="control-formulario" type="time" name="hora_fin" required value="${escaparHTML(f.hora_fin ?? "")}"${enviando ? " disabled" : ""}></label>`
    : `<label>${escaparHTML(t("desde"))}<input class="control-formulario" type="date" name="desde" required value="${escaparHTML(f.desde ?? "")}"${enviando ? " disabled" : ""}></label>
       <label>${escaparHTML(t("hasta"))}<input class="control-formulario" type="date" name="hasta" required value="${escaparHTML(f.hasta ?? "")}"${enviando ? " disabled" : ""}></label>`;
  const aviso = f.mensaje ? `<p class="cronos-solicitud-aviso" data-tono="${f.estado === "hecho" ? "exito" : "error"}" role="${f.estado === "hecho" ? "status" : "alert"}">${escaparHTML(f.mensaje)}</p>` : "";
  const titulo = t("solicitar_permiso", { permiso: permiso.nombre });
  return `<section class="panel cronos-panel" aria-labelledby="cronos-permiso-form-titulo"><div class="cabecera-panel"><h3 id="cronos-permiso-form-titulo">${escaparHTML(titulo)}</h3></div>
    <div class="cuerpo-panel"><form class="cronos-solicitud-formulario" data-cronos-permiso-formulario aria-label="${escaparHTML(titulo)}">${campos}
    <div class="cronos-solicitud-acciones"><button type="submit" class="boton-primario"${enviando ? " disabled" : ""}>${escaparHTML(t(enviando ? "solicitar_enviando" : "solicitar_enviar"))}</button>
    <button type="button" class="boton-secundario" data-cronos-permiso-cerrar>${escaparHTML(t("cancelar"))}</button></div>${aviso}</form></div></section>`;
}

/** Listado anual del catálogo versionado, solicitud y pendientes de conceder y de justificar. */
export function renderizarPermisosPropiosCronos({ estado = "cargando", anio, datos = null, solicitud = null, mensajes = MENSAJES_CRONOS_SOLICITUDES, locale = LOCALIZACION_ACTUAL, hoy = null, filtro = "todas", pagina = 1, tamanoPagina = 20,
  reciboConfirmado = null, mensajesHistorial = MENSAJES_HISTORIAL_CRONOS, zonaHoraria = "Europe/Madrid",
  mensajesJustificacion = MENSAJES_JUSTIFICACION_CRONOS, ayudaJustificacion = false, mensajesConsulta = MENSAJES_CONSULTA_PERMISOS_CRONOS } = {}) {
  const t = crearTraductorSolicitudesCronos(mensajes);
  const th = crearTraductorHistorialCronos(mensajesHistorial);
  const tj = crearTraductorJustificacionCronos(mensajesJustificacion);
  const tc = crearTraductorConsultaPermisosCronos(mensajesConsulta);
  if (estado === "listo") validarPermisosPropiosCronos(datos, anio);
  const historial = paginarHistorial(estado === "listo" ? datos.solicitudes : [], filtro, pagina, tamanoPagina);
  const recibo = panelRecibo(reciboConfirmado, th, locale, zonaHoraria);
  if (!Number.isInteger(anio) || anio < 2000 || anio > 2100) throw new RangeError("año de Cronos no válido");
  const ayuda = t("abrir_ayuda", { asunto: t("permisos_titulo") });
  const navegacion = `<nav class="cronos-anio-navegacion" aria-label="${escaparHTML(t("permisos_anio", { anio }))}"><button type="button" class="boton-secundario" data-cronos-anio="-1" aria-label="${escaparHTML(t("anio_anterior"))}"${anio <= 2000 ? " disabled" : ""}>‹</button><strong aria-live="polite">${anio}</strong><button type="button" class="boton-secundario" data-cronos-anio="1" aria-label="${escaparHTML(t("anio_siguiente"))}"${anio >= 2100 ? " disabled" : ""}>›</button></nav>`;
  const actualizar = `<button type="button" class="boton-secundario" data-cronos-permisos-actualizar=""${estado === "cargando" ? ' aria-disabled="true"' : ""}${solicitud?.estado === "enviando" ? " disabled" : ""}>${escaparHTML(tc(estado === "listo" ? "actualizar" : estado === "cargando" ? "cargando" : "reintentar"))}</button>`;
  const cabecera = `<header class="cronos-encabezado"><div><p class="sobrelinea">${escaparHTML(t("sobrelinea"))}</p><h2 id="cronos-permisos-propios-titulo">${escaparHTML(t("permisos_titulo"))}</h2></div>
    <div class="cronos-solicitud-acciones">${actualizar}<button type="button" class="cronos-boton-ayuda" data-accion="ayuda" aria-label="${escaparHTML(ayuda)}" title="${escaparHTML(ayuda)}"><span aria-hidden="true">?</span></button></div></header>`;
  if (estado !== "listo") {
    const clave = { denegado: "denegado", sin_empleado: "sin_empleado", error: "error" }[estado] ?? "cargando";
    return `<section class="cronos-area cronos-permisos-propios" aria-labelledby="cronos-permisos-propios-titulo" data-estado="${escaparHTML(estado)}">${cabecera}${recibo}
      <section class="panel cronos-panel"><div class="cabecera-panel">${navegacion}</div><div class="cuerpo-panel"><p class="cronos-${estado === "cargando" ? "vacio" : "acceso-denegado"}" role="${estado === "error" ? "alert" : "status"}">${escaparHTML(estado === "error" ? tc("error") : t(clave))}</p></div></section></section>`;
  }
  const porRef = new Map(datos.permisos.map((p) => [p.permiso_ref, p]));
  const pendientes = datos.solicitudes.filter((s) => s.estado === "solicitado" || s.estado === "pendiente_administracion");
  const justificar = datos.solicitudes.filter((s) => s.estado === "concedido" && s.pendiente_justificar);
  const concedidos = datos.solicitudes.filter((s) => s.estado === "concedido");
  const kpi = (clave, valor, tono) => `<article class="tarjeta-kpi" data-tono="${tono}"><span class="icono-kpi" aria-hidden="true"></span><div><p class="valor-kpi">${escaparHTML(new Intl.NumberFormat(locale).format(valor))}</p><p class="etiqueta-kpi">${escaparHTML(t(clave))}</p>${clave === "kpi_pendientes_justificar" ? `<button type="button" class="boton-secundario" data-cronos-ver-justificacion aria-controls="cronos-permisos-just-panel">${escaparHTML(tj("ver_pendientes"))}</button>` : ""}</div></article>`;
  const sintetico = datos.permisos.some((p) => p.sintetico);
  const filas = datos.permisos.map((p) => `<tr><th scope="row">${escaparHTML(p.nombre)}</th>
    <td class="numero">${escaparHTML(maximo(p, t, locale))}</td><td class="numero">${escaparHTML(formatearCantidadCronos(p.minimo, p.unidad, t, locale))}</td>
    <td class="numero">${escaparHTML(formatearCantidadCronos(p.solicitado, p.unidad, t, locale))}</td><td class="numero">${escaparHTML(formatearCantidadCronos(p.concedido, p.unidad, t, locale))}</td>
    <td class="numero">${resta(p, datos.solicitudes, anio, hoy, t, locale)}</td>
    <td>${p.solicitable ? `<button type="button" class="boton-secundario" data-cronos-solicitar="${escaparHTML(p.permiso_ref)}" aria-label="${escaparHTML(t("solicitar_permiso", { permiso: p.nombre }))}">${escaparHTML(t("solicitar"))} ›</button>` : ""}</td></tr>`);
  const fila = (s, pendienteJustificacion = false) => {
    const p = porRef.get(s.permiso_ref);
    return `<tr><th scope="row">${escaparHTML(p?.nombre ?? th("permiso_sin_nombre"))}</th><td>${escaparHTML(periodo(s, t, locale))}</td><td class="numero">${escaparHTML(formatearCantidadCronos(s.cantidad, s.unidad, t, locale))}</td><td><span class="cronos-estado${pendienteJustificacion ? " cronos-estado-aviso" : ""}" data-estado="${escaparHTML(s.estado)}">${escaparHTML(pendienteJustificacion ? tj("estado_pendiente") : t(estadoSolicitudPermisoCronos(s)))}</span></td></tr>`;
  };
  const filtros = `<label for="cronos-historial-filtro">${escaparHTML(th("filtro"))}</label><select class="control-formulario" id="cronos-historial-filtro" data-cronos-historial-filtro>${FILTROS_HISTORIAL.map((f) =>
    `<option value="${f}"${f === filtro ? " selected" : ""}>${escaparHTML(th(f))} (${new Intl.NumberFormat(locale).format(filtrarSolicitudes(datos.solicitudes, f).length)})</option>`).join("")}</select>`;
  const paginacion = `<nav class="cronos-navegacion" aria-label="${escaparHTML(th("titulo"))}"><button type="button" class="boton-secundario" data-cronos-historial-pagina="anterior"${historial.actual === 1 ? " disabled" : ""}>${escaparHTML(th("anterior"))}</button>
    <p data-cronos-historial-recuento tabindex="-1" role="status" aria-live="polite" aria-atomic="true">${escaparHTML(recuentoHistorial(historial, th, locale))}</p>
    <button type="button" class="boton-secundario" data-cronos-historial-pagina="siguiente"${historial.actual === historial.paginas ? " disabled" : ""}>${escaparHTML(th("siguiente"))}</button></nav>`;
  const elegido = solicitud ? porRef.get(solicitud.permisoRef) : null;
  return `<section class="cronos-area cronos-permisos-propios" aria-labelledby="cronos-permisos-propios-titulo" data-estado="listo">${cabecera}${recibo}
    <div class="rejilla-kpi">${kpi("kpi_pendientes_conceder", pendientes.length, "naranja")}${kpi("kpi_pendientes_justificar", justificar.length, "violeta")}${kpi("kpi_concedidos", concedidos.length, "verde")}</div>
    <section class="panel cronos-panel" aria-labelledby="cronos-permisos-anio"><div class="cabecera-panel"><h3 id="cronos-permisos-anio">${escaparHTML(t("permisos_anio", { anio }))}</h3>${sintetico ? `<span class="cronos-estado cronos-estado-aviso">${escaparHTML(t("permisos_a_confirmar"))}</span>` : ""}${navegacion}</div>
      ${tabla(["permiso", "col_maximo", "col_minimo", "col_solicitado", "col_concedido", "col_resta", "col_accion"], filas, t)}</section>
    ${solicitud && !elegido?.solicitable ? `<section class="panel cronos-panel"><div class="cuerpo-panel"><p role="status">${escaparHTML(tc("no_solicitable"))}</p><button type="button" class="boton-secundario" data-cronos-permiso-cerrar>${escaparHTML(t("cancelar"))}</button></div></section>` : formularioSolicitud(solicitud, elegido, t)}
    <section class="panel cronos-panel" aria-labelledby="cronos-historial-titulo"><div class="cabecera-panel"><h3 id="cronos-historial-titulo">${escaparHTML(th("titulo"))}</h3></div>
      <div class="cuerpo-panel">${filtros}</div>${tabla(["permiso", "periodo", "duracion", "estado"], historial.filas.map((s) => fila(s)), t, th("sin_resultados"))}<div class="cuerpo-panel">${paginacion}</div></section>
    <section class="panel cronos-panel" id="cronos-permisos-just-panel" aria-labelledby="cronos-permisos-just"><div class="cabecera-panel"><h3 id="cronos-permisos-just" tabindex="-1">${escaparHTML(t("pendientes_justificar_titulo"))}</h3>
      <button type="button" class="cronos-boton-ayuda" data-cronos-justificacion-ayuda aria-label="${escaparHTML(tj("ayuda_titulo"))}" aria-expanded="${ayudaJustificacion}" aria-controls="cronos-justificacion-ayuda"><span aria-hidden="true">?</span></button></div>
      <div class="cuerpo-panel" id="cronos-justificacion-ayuda"${ayudaJustificacion ? "" : " hidden"}><p>${escaparHTML(tj("ayuda_origen"))}</p><p>${escaparHTML(tj("ayuda_custodia"))}</p></div>
      ${justificar.length ? `<div class="cuerpo-panel"><p class="cronos-mensaje">${escaparHTML(tj("registro_pendiente"))}</p></div>` : ""}
      ${tabla(["permiso", "periodo", "duracion", "estado"], justificar.map((s) => fila(s, true)), t, tj("sin_pendientes"))}</section>
  </section>`;
}

function estadoError(error) {
  if (error instanceof ErrorClienteSolicitudesCronos) {
    if (error.codigo === "sin_empleado") return "sin_empleado";
    if (["acceso_denegado", "autenticacion_requerida"].includes(error.codigo)) return "denegado";
  }
  return "error";
}

export function montarPermisosPropiosCronos({ raiz, cliente = crearClienteSolicitudesCronosHTTP(), mensajes = MENSAJES_CRONOS_SOLICITUDES,
  anunciar = () => {}, registrarDesmontar, locale = LOCALIZACION_ACTUAL, zonaHoraria = "Europe/Madrid", anio, tamanoPagina = 20, mensajesHistorial = MENSAJES_HISTORIAL_CRONOS,
  mensajesJustificacion = MENSAJES_JUSTIFICACION_CRONOS, mensajesConsulta = MENSAJES_CONSULTA_PERMISOS_CRONOS } = {}) {
  if (!raiz?.append || !raiz.ownerDocument?.createElement || typeof cliente?.consultarPermisos !== "function" || typeof cliente?.solicitarPermiso !== "function"
    || typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) throw new TypeError("montaje de permisos propios Cronos no disponible");
  const t = crearTraductorSolicitudesCronos(mensajes);
  const th = crearTraductorHistorialCronos(mensajesHistorial);
  const tc = crearTraductorConsultaPermisosCronos(mensajesConsulta);
  paginarHistorial([], "todas", 1, tamanoPagina);
  const contenedor = raiz.ownerDocument.createElement("section"); contenedor.dataset.cronosPermisosPropios = ""; raiz.append(contenedor);
  let anioVisible = Number.isInteger(anio) ? anio : Number(hoyCivilCronos(zonaHoraria).slice(0, 4));
  let activa = true; let secuencia = 0; let controlador = null; let envio = null;
  let estado = "cargando"; let datos = null; let solicitud = null;
  let filtro = "todas"; let pagina = 1; let reciboConfirmado = null;
  let ayudaJustificacion = false;
  const dibujar = () => {
    if (!activa) return;
    const foco = raiz.ownerDocument.activeElement;
    let selector = null; let direccionAnio = null;
    if (foco && contenedor.contains?.(foco)) {
      for (const atributo of ["data-cronos-permisos-actualizar", "data-cronos-anio", "data-cronos-historial-filtro", "data-cronos-historial-pagina", "data-cronos-historial-recuento", "data-cronos-ver-justificacion", "data-cronos-justificacion-ayuda", "name"]) {
        const valor = foco.getAttribute?.(atributo);
        const valido = atributo === "data-cronos-anio" ? ["-1", "1"].includes(valor) : valor !== null && valor !== undefined && /^[a-z_]*$/u.test(valor);
        if (valido) { selector = `[${atributo}="${valor}"]`; if (atributo === "data-cronos-anio") direccionAnio = valor; break; }
      }
    }
    contenedor.innerHTML = renderizarPermisosPropiosCronos({ estado, anio: anioVisible, datos, solicitud, mensajes, locale,
      hoy: hoyCivilCronos(zonaHoraria), filtro, pagina, tamanoPagina, reciboConfirmado, mensajesHistorial, zonaHoraria, mensajesJustificacion, ayudaJustificacion, mensajesConsulta });
    if (selector) {
      let destino = contenedor.querySelector?.(selector);
      if (!destino || destino.disabled) destino = contenedor.querySelector?.(direccionAnio
        ? `[data-cronos-anio="${direccionAnio === "-1" ? "1" : "-1"}"]` : "[data-cronos-historial-recuento]");
      if (!destino?.disabled) destino?.focus?.();
    }
  };
  const cargar = async () => {
    if (!activa) return;
    controlador?.abort(); const consulta = new AbortController(); controlador = consulta; const turno = ++secuencia;
    estado = "cargando"; datos = null; dibujar();
    try {
      const r = await cliente.consultarPermisos({ anio: anioVisible }, { signal: consulta.signal });
      if (!activa || turno !== secuencia) return;
      datos = validarPermisosPropiosCronos(r, anioVisible);
      pagina = paginarHistorial(datos.solicitudes, filtro, pagina, tamanoPagina).actual;
      estado = "listo"; dibujar();
    } catch (error) {
      if (!activa || turno !== secuencia || consulta.signal.aborted) return;
      datos = null; estado = estadoError(error); dibujar(); anunciar(estado === "error" ? tc("error") : t(estado));
    }
  };
  const alPulsar = (evento) => {
    const actualizar = evento.target?.closest?.("[data-cronos-permisos-actualizar]");
    if (actualizar) {
      if (!actualizar.disabled && estado !== "cargando" && solicitud?.estado !== "enviando") void cargar();
      return;
    }
    if (estado === "listo" && evento.target?.closest?.("[data-cronos-ver-justificacion]")) {
      const destino = contenedor.querySelector?.("#cronos-permisos-just");
      destino?.focus?.(); destino?.scrollIntoView?.({ block: "nearest" }); return;
    }
    if (estado === "listo" && evento.target?.closest?.("[data-cronos-justificacion-ayuda]")) {
      ayudaJustificacion = !ayudaJustificacion; dibujar();
      contenedor.querySelector?.("[data-cronos-justificacion-ayuda]")?.focus?.(); return;
    }
    const cambio = evento.target?.closest?.("[data-cronos-historial-pagina]");
    if (cambio && !cambio.disabled && estado === "listo") {
      const paso = cambio.dataset.cronosHistorialPagina;
      if (!["anterior", "siguiente"].includes(paso)) return;
      pagina = paginarHistorial(datos.solicitudes, filtro, Math.max(1, pagina + (paso === "anterior" ? -1 : 1)), tamanoPagina).actual;
      dibujar(); anunciar(recuentoHistorial(paginarHistorial(datos.solicitudes, filtro, pagina, tamanoPagina), th, locale)); return;
    }
    const anioBoton = evento.target?.closest?.("[data-cronos-anio]");
    if (anioBoton && !anioBoton.disabled) {
      const siguiente = anioVisible + Number(anioBoton.dataset.cronosAnio);
      if (siguiente >= 2000 && siguiente <= 2100) { anioVisible = siguiente; pagina = 1; envio?.abort(); solicitud = null; void cargar(); }
      return;
    }
    const pedir = evento.target?.closest?.("[data-cronos-solicitar]");
    if (estado === "listo" && pedir && datos?.permisos.some((p) => p.permiso_ref === pedir.dataset.cronosSolicitar && p.solicitable)) {
      envio?.abort();
      solicitud = { permisoRef: pedir.dataset.cronosSolicitar, clave: claveNueva() };
      dibujar(); contenedor.querySelector?.("[data-cronos-permiso-formulario] [name=desde]")?.focus?.();
      return;
    }
    if (evento.target?.closest?.("[data-cronos-permiso-cerrar]")) {
      const permisoRef = solicitud?.permisoRef;
      envio?.abort(); solicitud = null; dibujar();
      if (permisoRef) Array.from(contenedor.querySelectorAll?.("[data-cronos-solicitar]") ?? [])
        .find((boton) => boton.dataset.cronosSolicitar === permisoRef)?.focus?.();
    }
  };
  const alEnviar = async (evento) => {
    if (!evento.target?.matches?.("[data-cronos-permiso-formulario]") || !solicitud || solicitud.estado === "enviando" || estado !== "listo") return;
    evento.preventDefault();
    const permiso = datos?.permisos.find((p) => p.permiso_ref === solicitud.permisoRef);
    if (!permiso?.solicitable) return;
    const valor = (n) => evento.target.elements?.namedItem?.(n)?.value ?? "";
    const horas = permiso.unidad === "hora";
    const campos = horas ? { desde: valor("desde"), hasta: valor("desde"), hora_inicio: valor("hora_inicio"), hora_fin: valor("hora_fin") } : { desde: valor("desde"), hasta: valor("hasta") };
    // Un reintento de la misma petición conserva la clave; otra petición usa otra.
    const firma = JSON.stringify(campos);
    if (solicitud.estado === "hecho" || (solicitud.firma && solicitud.firma !== firma)) solicitud.clave = claveNueva();
    solicitud = { ...solicitud, ...campos, estado: "enviando", mensaje: "", firma };
    dibujar();
    const controladorEnvio = new AbortController(); envio = controladorEnvio;
    const entrada = { clave_operacion: solicitud.clave, permiso_ref: permiso.permiso_ref, ...campos };
    try {
      validarEntradaPermisoCronos(entrada);
      const respuesta = await cliente.solicitarPermiso(entrada, { signal: controladorEnvio.signal });
      if (!activa || controladorEnvio.signal.aborted || solicitud?.clave !== entrada.clave_operacion) return;
      const recibo = validarReciboSolicitud(respuesta, entrada, permiso.unidad);
      reciboConfirmado = recibo;
      const cantidad = formatearCantidadCronos(recibo.cantidad, recibo.unidad, t, locale);
      solicitud = { ...solicitud, estado: "hecho", mensaje: t(recibo.replay ? "solicitud_ya_registrada" : "solicitud_registrada", { cantidad }) };
      anunciar(solicitud.mensaje);
      await cargar();
    } catch (error) {
      if (!activa || controladorEnvio.signal.aborted || solicitud?.clave !== entrada.clave_operacion) return;
      const codigo = error instanceof ErrorClienteSolicitudesCronos ? error.codigo : error instanceof TypeError ? "peticion_invalida" : "";
      solicitud = { ...solicitud, estado: "error", mensaje: t(ERRORES.get(codigo) ?? "error_solicitud") };
      dibujar(); anunciar(solicitud.mensaje);
    }
  };
  const alCambiar = (evento) => {
    if (!evento.target?.matches?.("[data-cronos-historial-filtro]") || estado !== "listo" || !FILTROS_HISTORIAL.includes(evento.target.value)) return;
    filtro = evento.target.value; pagina = 1; dibujar();
    anunciar(recuentoHistorial(paginarHistorial(datos.solicitudes, filtro, pagina, tamanoPagina), th, locale));
  };
  const alEditar = (evento) => {
    if (!solicitud || solicitud.estado === "enviando" || !evento.target?.closest?.("[data-cronos-permiso-formulario]")) return;
    if (["desde", "hasta", "hora_inicio", "hora_fin"].includes(evento.target.name)) solicitud = { ...solicitud, [evento.target.name]: evento.target.value };
  };
  contenedor.addEventListener("change", alCambiar); contenedor.addEventListener("input", alEditar);
  contenedor.addEventListener("click", alPulsar); contenedor.addEventListener("submit", alEnviar);
  void cargar();
  const desmontar = () => {
    if (!activa) return;
    activa = false; ++secuencia; controlador?.abort(); envio?.abort();
    contenedor.removeEventListener("change", alCambiar); contenedor.removeEventListener("input", alEditar);
    contenedor.removeEventListener("click", alPulsar); contenedor.removeEventListener("submit", alEnviar); contenedor.remove?.();
  };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ desmontar, recargar: cargar });
}
