import { crearTraductorSolicitudesCronos, formatearCantidadCronos, MENSAJES_CRONOS_SOLICITUDES_ES } from "./i18n-solicitudes.js";
import { ErrorClienteSolicitudesCronos, crearClienteSolicitudesCronosHTTP } from "./cliente-solicitudes-http.js";
import { hoyCivilCronos } from "./vista-movimientos-propios.js?v=20260925-tanda2-v1";

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

/** Estado de una solicitud viva según el circuito de su permiso; la concesión es de otro corte. */
function estadoSolicitud(s, circuito) {
  if (s.estado === "concedido") return "estado_concedido";
  if (s.estado === "pendiente_administracion" || circuito === "A") return "estado_pendiente_administracion";
  return "estado_pendiente_jefatura";
}

function tabla(cabeceras, filas, t) {
  const cuerpo = filas.length ? filas.join("") : `<tr><td colspan="${cabeceras.length}">${escaparHTML(t("sin_filas"))}</td></tr>`;
  return `<div class="cronos-tabla-contenedor"><table class="cronos-tabla"><thead><tr>${cabeceras.map((c) => `<th scope="col"${c.startsWith("col_m") || ["col_solicitado", "col_concedido", "col_resta", "duracion"].includes(c) ? ' class="numero"' : ""}>${escaparHTML(t(c))}</th>`).join("")}</tr></thead><tbody>${cuerpo}</tbody></table></div>`;
}

function formularioSolicitud(f, permiso, t) {
  if (!f || !permiso) return "";
  const enviando = f.estado === "enviando";
  const campos = permiso.unidad === "hora"
    ? `<label>${escaparHTML(t("fecha"))}<input type="date" name="desde" required value="${escaparHTML(f.desde ?? "")}"${enviando ? " disabled" : ""}></label>
       <label>${escaparHTML(t("hora_inicio"))}<input type="time" name="hora_inicio" required value="${escaparHTML(f.hora_inicio ?? "")}"${enviando ? " disabled" : ""}></label>
       <label>${escaparHTML(t("hora_fin"))}<input type="time" name="hora_fin" required value="${escaparHTML(f.hora_fin ?? "")}"${enviando ? " disabled" : ""}></label>`
    : `<label>${escaparHTML(t("desde"))}<input type="date" name="desde" required value="${escaparHTML(f.desde ?? "")}"${enviando ? " disabled" : ""}></label>
       <label>${escaparHTML(t("hasta"))}<input type="date" name="hasta" required value="${escaparHTML(f.hasta ?? "")}"${enviando ? " disabled" : ""}></label>`;
  const aviso = f.mensaje ? `<p class="cronos-solicitud-aviso" data-tono="${f.estado === "hecho" ? "exito" : "error"}" role="${f.estado === "hecho" ? "status" : "alert"}">${escaparHTML(f.mensaje)}</p>` : "";
  const titulo = t("solicitar_permiso", { permiso: permiso.nombre });
  return `<section class="panel cronos-panel" aria-labelledby="cronos-permiso-form-titulo"><div class="cabecera-panel"><h3 id="cronos-permiso-form-titulo">${escaparHTML(titulo)}</h3><span class="cronos-circuito">${escaparHTML(t(`circuito_${permiso.circuito}`))}</span></div>
    <div class="cuerpo-panel"><form class="cronos-solicitud-formulario" data-cronos-permiso-formulario aria-label="${escaparHTML(titulo)}">${campos}
    <div class="cronos-solicitud-acciones"><button type="submit" class="boton-primario"${enviando ? " disabled" : ""}>${escaparHTML(t(enviando ? "solicitar_enviando" : "solicitar_enviar"))}</button>
    <button type="button" class="boton-secundario" data-cronos-permiso-cerrar>${escaparHTML(t("cancelar"))}</button></div>${aviso}</form></div></section>`;
}

/** Listado anual del catálogo versionado, solicitud y pendientes de conceder y de justificar. */
export function renderizarPermisosPropiosCronos({ estado = "cargando", anio, datos = null, solicitud = null, mensajes = MENSAJES_CRONOS_SOLICITUDES_ES, locale = "es-ES", hoy = null } = {}) {
  const t = crearTraductorSolicitudesCronos(mensajes);
  if (!Number.isInteger(anio) || anio < 2000 || anio > 2100) throw new RangeError("año de Cronos no válido");
  const ayuda = t("abrir_ayuda", { asunto: t("permisos_titulo") });
  const navegacion = `<nav class="cronos-anio-navegacion" aria-label="${escaparHTML(t("permisos_anio", { anio }))}"><button type="button" class="boton-secundario" data-cronos-anio="-1" aria-label="${escaparHTML(t("anio_anterior"))}"${anio <= 2000 ? " disabled" : ""}>‹</button><strong aria-live="polite">${anio}</strong><button type="button" class="boton-secundario" data-cronos-anio="1" aria-label="${escaparHTML(t("anio_siguiente"))}"${anio >= 2100 ? " disabled" : ""}>›</button></nav>`;
  const cabecera = `<header class="cronos-encabezado"><div><p class="sobrelinea">${escaparHTML(t("sobrelinea"))}</p><h2 id="cronos-permisos-propios-titulo">${escaparHTML(t("permisos_titulo"))}</h2></div>
    <button type="button" class="cronos-boton-ayuda" data-accion="ayuda" aria-label="${escaparHTML(ayuda)}" title="${escaparHTML(ayuda)}"><span aria-hidden="true">?</span></button></header>`;
  if (estado !== "listo") {
    const clave = { denegado: "denegado", sin_empleado: "sin_empleado", error: "error" }[estado] ?? "cargando";
    return `<section class="cronos-area cronos-permisos-propios" aria-labelledby="cronos-permisos-propios-titulo" data-estado="${escaparHTML(estado)}">${cabecera}
      <section class="panel cronos-panel"><div class="cabecera-panel">${navegacion}</div><div class="cuerpo-panel"><p class="cronos-${estado === "cargando" ? "vacio" : "acceso-denegado"}" role="${estado === "error" ? "alert" : "status"}">${escaparHTML(t(clave))}</p></div></section></section>`;
  }
  const porRef = new Map(datos.permisos.map((p) => [p.permiso_ref, p]));
  const pendientes = datos.solicitudes.filter((s) => s.estado === "solicitado" || s.estado === "pendiente_administracion");
  const justificar = datos.solicitudes.filter((s) => s.estado === "concedido" && s.pendiente_justificar);
  const concedidos = datos.solicitudes.filter((s) => s.estado === "concedido");
  const kpi = (clave, valor, tono) => `<article class="tarjeta-kpi" data-tono="${tono}"><span class="icono-kpi" aria-hidden="true"></span><div><p class="valor-kpi">${escaparHTML(new Intl.NumberFormat(locale).format(valor))}</p><p class="etiqueta-kpi">${escaparHTML(t(clave))}</p></div></article>`;
  const sintetico = datos.permisos.some((p) => p.sintetico);
  const filas = datos.permisos.map((p) => `<tr><th scope="row">${escaparHTML(p.nombre)}</th><td>${escaparHTML(t(`circuito_${p.circuito}`))}</td>
    <td class="numero">${escaparHTML(maximo(p, t, locale))}</td><td class="numero">${escaparHTML(formatearCantidadCronos(p.minimo, p.unidad, t, locale))}</td>
    <td class="numero">${escaparHTML(formatearCantidadCronos(p.solicitado, p.unidad, t, locale))}</td><td class="numero">${escaparHTML(formatearCantidadCronos(p.concedido, p.unidad, t, locale))}</td>
    <td class="numero">${resta(p, datos.solicitudes, anio, hoy, t, locale)}</td>
    <td>${p.solicitable ? `<button type="button" class="boton-secundario" data-cronos-solicitar="${escaparHTML(p.permiso_ref)}" aria-label="${escaparHTML(t("solicitar_permiso", { permiso: p.nombre }))}">${escaparHTML(t("solicitar"))} ›</button>` : ""}</td></tr>`);
  const fila = (s) => {
    const p = porRef.get(s.permiso_ref);
    return `<tr><th scope="row">${escaparHTML(p?.nombre ?? "")}</th><td>${escaparHTML(periodo(s, t, locale))}</td><td class="numero">${escaparHTML(formatearCantidadCronos(s.cantidad, s.unidad, t, locale))}</td><td><span class="cronos-estado" data-estado="${escaparHTML(s.estado)}">${escaparHTML(t(estadoSolicitud(s, p?.circuito)))}</span></td></tr>`;
  };
  const elegido = solicitud ? porRef.get(solicitud.permisoRef) : null;
  return `<section class="cronos-area cronos-permisos-propios" aria-labelledby="cronos-permisos-propios-titulo" data-estado="listo">${cabecera}
    <div class="rejilla-kpi">${kpi("kpi_pendientes_conceder", pendientes.length, "naranja")}${kpi("kpi_pendientes_justificar", justificar.length, "violeta")}${kpi("kpi_concedidos", concedidos.length, "verde")}</div>
    <section class="panel cronos-panel" aria-labelledby="cronos-permisos-anio"><div class="cabecera-panel"><h3 id="cronos-permisos-anio">${escaparHTML(t("permisos_anio", { anio }))}</h3>${sintetico ? `<span class="cronos-estado cronos-estado-aviso">${escaparHTML(t("permisos_a_confirmar"))}</span>` : ""}${navegacion}</div>
      ${tabla(["permiso", "col_concede", "col_maximo", "col_minimo", "col_solicitado", "col_concedido", "col_resta", "col_accion"], filas, t)}</section>
    ${formularioSolicitud(solicitud, elegido, t)}
    <section class="panel cronos-panel" aria-labelledby="cronos-permisos-pend"><div class="cabecera-panel"><h3 id="cronos-permisos-pend">${escaparHTML(t("pendientes_conceder_titulo"))}</h3></div>${tabla(["permiso", "periodo", "duracion", "estado"], pendientes.map(fila), t)}</section>
    <section class="panel cronos-panel" aria-labelledby="cronos-permisos-just"><div class="cabecera-panel"><h3 id="cronos-permisos-just">${escaparHTML(t("pendientes_justificar_titulo"))}</h3></div>${tabla(["permiso", "periodo", "duracion", "estado"], justificar.map(fila), t)}</section>
    <section class="panel cronos-panel" aria-labelledby="cronos-permisos-conc"><div class="cabecera-panel"><h3 id="cronos-permisos-conc">${escaparHTML(t("concedidos_titulo"))}</h3></div>${tabla(["permiso", "periodo", "duracion", "estado"], concedidos.map(fila), t)}</section>
  </section>`;
}

function estadoError(error) {
  if (error instanceof ErrorClienteSolicitudesCronos) {
    if (error.codigo === "sin_empleado") return "sin_empleado";
    if (["acceso_denegado", "autenticacion_requerida"].includes(error.codigo)) return "denegado";
  }
  return "error";
}

export function montarPermisosPropiosCronos({ raiz, cliente = crearClienteSolicitudesCronosHTTP(), mensajes = MENSAJES_CRONOS_SOLICITUDES_ES,
  anunciar = () => {}, registrarDesmontar, locale = "es-ES", zonaHoraria = "Europe/Madrid", anio } = {}) {
  if (!raiz?.append || !raiz.ownerDocument?.createElement || typeof cliente?.consultarPermisos !== "function" || typeof cliente?.solicitarPermiso !== "function"
    || typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) throw new TypeError("montaje de permisos propios Cronos no disponible");
  const t = crearTraductorSolicitudesCronos(mensajes);
  const contenedor = raiz.ownerDocument.createElement("section"); contenedor.dataset.cronosPermisosPropios = ""; raiz.append(contenedor);
  let anioVisible = Number.isInteger(anio) ? anio : Number(hoyCivilCronos(zonaHoraria).slice(0, 4));
  let activa = true; let secuencia = 0; let controlador = null; let envio = null;
  let estado = "cargando"; let datos = null; let solicitud = null;
  const dibujar = () => { if (activa) contenedor.innerHTML = renderizarPermisosPropiosCronos({ estado, anio: anioVisible, datos, solicitud, mensajes, locale, hoy: hoyCivilCronos(zonaHoraria) }); };
  const cargar = async () => {
    controlador?.abort(); controlador = new AbortController(); const turno = ++secuencia;
    estado = "cargando"; datos = null; dibujar();
    try {
      const r = await cliente.consultarPermisos({ anio: anioVisible }, { signal: controlador.signal });
      if (!activa || turno !== secuencia) return;
      estado = "listo"; datos = r; dibujar();
    } catch (error) {
      if (!activa || turno !== secuencia || controlador.signal.aborted) return;
      estado = estadoError(error); dibujar(); anunciar(estado);
    }
  };
  const alPulsar = (evento) => {
    const anioBoton = evento.target?.closest?.("[data-cronos-anio]");
    if (anioBoton && !anioBoton.disabled) {
      const siguiente = anioVisible + Number(anioBoton.dataset.cronosAnio);
      if (siguiente >= 2000 && siguiente <= 2100) { anioVisible = siguiente; envio?.abort(); solicitud = null; void cargar(); }
      return;
    }
    const pedir = evento.target?.closest?.("[data-cronos-solicitar]");
    if (pedir && datos?.permisos.some((p) => p.permiso_ref === pedir.dataset.cronosSolicitar && p.solicitable)) {
      envio?.abort();
      solicitud = { permisoRef: pedir.dataset.cronosSolicitar, clave: claveNueva() };
      dibujar(); contenedor.querySelector?.("[data-cronos-permiso-formulario] [name=desde]")?.focus?.();
      return;
    }
    if (evento.target?.closest?.("[data-cronos-permiso-cerrar]")) { envio?.abort(); solicitud = null; dibujar(); }
  };
  const alEnviar = async (evento) => {
    if (!evento.target?.matches?.("[data-cronos-permiso-formulario]") || !solicitud || solicitud.estado === "enviando") return;
    evento.preventDefault();
    const permiso = datos?.permisos.find((p) => p.permiso_ref === solicitud.permisoRef);
    if (!permiso) return;
    const valor = (n) => evento.target.elements?.namedItem?.(n)?.value ?? "";
    const horas = permiso.unidad === "hora";
    const campos = horas ? { desde: valor("desde"), hasta: valor("desde"), hora_inicio: valor("hora_inicio"), hora_fin: valor("hora_fin") } : { desde: valor("desde"), hasta: valor("hasta") };
    // Un reintento de la misma petición conserva la clave; otra petición usa otra.
    const firma = JSON.stringify(campos);
    if (solicitud.estado === "hecho" || (solicitud.firma && solicitud.firma !== firma)) solicitud.clave = claveNueva();
    solicitud = { ...solicitud, ...campos, estado: "enviando", mensaje: "", firma };
    dibujar();
    envio = new AbortController();
    try {
      const recibo = await cliente.solicitarPermiso({ clave_operacion: solicitud.clave, permiso_ref: permiso.permiso_ref, ...campos }, { signal: envio.signal });
      if (!activa) return;
      const cantidad = formatearCantidadCronos(recibo.cantidad, recibo.unidad, t, locale);
      solicitud = { ...solicitud, estado: "hecho", mensaje: t(recibo.replay ? "solicitud_ya_registrada" : "solicitud_registrada", { cantidad, circuito: t(`circuito_${permiso.circuito}`) }) };
      anunciar(solicitud.mensaje);
      await cargar();
    } catch (error) {
      if (!activa || envio.signal.aborted) return;
      const codigo = error instanceof ErrorClienteSolicitudesCronos ? error.codigo : error instanceof TypeError ? "peticion_invalida" : "";
      solicitud = { ...solicitud, estado: "error", mensaje: t(ERRORES.get(codigo) ?? "error_solicitud") };
      dibujar(); anunciar(solicitud.mensaje);
    }
  };
  contenedor.addEventListener("click", alPulsar); contenedor.addEventListener("submit", alEnviar);
  void cargar();
  const desmontar = () => {
    if (!activa) return;
    activa = false; ++secuencia; controlador?.abort(); envio?.abort();
    contenedor.removeEventListener("click", alPulsar); contenedor.removeEventListener("submit", alEnviar); contenedor.remove?.();
  };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ desmontar, recargar: cargar });
}
