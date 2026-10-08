import { formatearCantidadCronos, MENSAJES_CRONOS_SOLICITUDES } from "./i18n-solicitudes.js";
import { ErrorClienteSolicitudesCronos, crearClienteSolicitudesCronosHTTP, validarMovimientosPropiosCronos, validarEntradaCorreccionCronos } from "./cliente-solicitudes-http.js";
import { crearTraductorIncidenciasCronos } from "./i18n-incidencias.js?v=20261001-cronos-grafo-bandeja-v5";
import { LOCALIZACION_ACTUAL } from "../../../comun/idioma.js";
import { icono } from "../../../comun/iconos-vec.js?v=20260925-aspecto-v1";
export { hoyCivilCronos } from "./fecha-civil.js?v=20261007-pantallas-textos-final-v1";
import { hoyCivilCronos } from "./fecha-civil.js?v=20261007-pantallas-textos-final-v1";

const MOVIMIENTOS = ["entrada", "salida", "inicio_pausa", "fin_pausa"];
const ESTADOS_CORRECCION = ["pendiente_responsable", "pendiente_rrhh", "denegada_responsable", "denegada_rrhh", "pendiente_aplicacion", "aplicada"];
const ORDEN_TIPOS = ["festivo", "no_laborable", "ausencia", "olvido", "marcaje"];

function escaparHTML(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}

/** Encabezado de la parte: propio de página, o de tarjeta sin sobrelínea dentro de «Jornada». */
function encabezadoParte(sobrelinea, id, titulo, incrustada) {
  return incrustada ? `<h3 id="${id}">${escaparHTML(titulo)}</h3>`
    : `<p class="sobrelinea">${escaparHTML(sobrelinea)}</p><h2 id="${id}">${escaparHTML(titulo)}</h2>`;
}
function iso(anio, mes, dia) { return `${anio}-${String(mes).padStart(2, "0")}-${String(dia).padStart(2, "0")}`; }
function fechaVisible(fecha, locale, estilo = "medium") {
  const [a, m, d] = fecha.split("-").map(Number);
  return new Intl.DateTimeFormat(locale, { timeZone: "UTC", dateStyle: estilo }).format(new Date(Date.UTC(a, m - 1, d, 12)));
}
function instanteVisible(instante, locale, zona) {
  return new Intl.DateTimeFormat(locale, { timeZone: zona, dateStyle: "medium", timeStyle: "short" }).format(new Date(instante));
}
function claveNueva() { return globalThis.crypto.randomUUID(); }

/** Marca cada día con los tipos que acreditan los datos recibidos; nada se infiere. */
export function marcasPorDiaCronos(datos) {
  const marcas = new Map();
  const poner = (fecha, tipo) => { if (!marcas.has(fecha)) marcas.set(fecha, new Set()); marcas.get(fecha).add(tipo); };
  for (const d of datos.calendario.dias) poner(d.fecha, d.tipo);
  for (const d of datos.marcajes_por_dia) poner(d.fecha, "marcaje");
  for (const c of datos.correcciones) poner(c.fecha_civil, "olvido");
  for (const a of datos.absentismos) {
    const desde = a.desde < datos.periodo.desde ? datos.periodo.desde : a.desde;
    const hasta = a.hasta > datos.periodo.hasta ? datos.periodo.hasta : a.hasta;
    for (let f = new Date(`${desde}T12:00:00Z`); f <= new Date(`${hasta}T12:00:00Z`); f.setUTCDate(f.getUTCDate() + 1)) {
      const fecha = f.toISOString().slice(0, 10);
      if (fecha >= datos.periodo.desde && fecha <= datos.periodo.hasta) poner(fecha, "ausencia");
    }
  }
  return marcas;
}

function nombreTipo(tipo, t) { return t(`tipo_${tipo}`); }

function mesHTML(anio, mes, marcas, t, locale, seleccionada) {
  const dias = new Date(Date.UTC(anio, mes, 0)).getUTCDate();
  const huecos = (new Date(Date.UTC(anio, mes - 1, 1)).getUTCDay() + 6) % 7;
  const celdas = Array.from({ length: huecos }, () => '<td aria-hidden="true"></td>');
  for (let dia = 1; dia <= dias; dia++) {
    const fecha = iso(anio, mes, dia);
    const tipos = ORDEN_TIPOS.filter((tipo) => marcas.get(fecha)?.has(tipo));
    const etiqueta = tipos.length ? t("dia_con", { fecha: fechaVisible(fecha, locale, "full"), tipos: tipos.map((x) => nombreTipo(x, t)).join(", ") }) : fechaVisible(fecha, locale, "full");
    celdas.push(`<td><button type="button" class="cronos-dia" data-fecha="${fecha}"${tipos.length ? ` data-tipos="${tipos.join(" ")}"` : ""} data-cronos-cal-dia aria-pressed="${fecha === seleccionada}" aria-controls="cronos-movpropios-detalle" title="${escaparHTML(etiqueta)}"><span class="cronos-sr">${escaparHTML(etiqueta)}</span><span aria-hidden="true">${dia}</span></button></td>`);
  }
  while (celdas.length % 7) celdas.push('<td aria-hidden="true"></td>');
  const filas = [];
  for (let i = 0; i < celdas.length; i += 7) filas.push(`<tr>${celdas.slice(i, i + 7).join("")}</tr>`);
  const nombre = new Intl.DateTimeFormat(locale, { timeZone: "UTC", month: "long" }).format(new Date(Date.UTC(anio, mes - 1, 1)));
  const cabecera = Array.from({ length: 7 }, (_, i) => {
    const f = new Date(Date.UTC(2024, 0, 1 + i));
    return `<th scope="col"><abbr title="${escaparHTML(new Intl.DateTimeFormat(locale, { timeZone: "UTC", weekday: "long" }).format(f))}">${escaparHTML(new Intl.DateTimeFormat(locale, { timeZone: "UTC", weekday: "narrow" }).format(f))}</abbr></th>`;
  }).join("");
  return `<section class="cronos-mes" aria-label="${escaparHTML(t("calendario_mes", { mes: nombre, anio }))}"><h4>${escaparHTML(nombre)}</h4><table><thead><tr>${cabecera}</tr></thead><tbody>${filas.join("")}</tbody></table></section>`;
}

function fechaCivilValida(fecha) {
  if (typeof fecha !== "string" || !/^\d{4}-\d{2}-\d{2}$/u.test(fecha)) return false;
  const [anio, mes, dia] = fecha.split("-").map(Number);
  const f = new Date(Date.UTC(anio, mes - 1, dia));
  return anio >= 2000 && anio <= 2100 && f.getUTCFullYear() === anio && f.getUTCMonth() === mes - 1 && f.getUTCDate() === dia;
}

function fechaEnAnio(fecha, anio) {
  const [, mes, dia] = fecha.split("-").map(Number);
  return iso(anio, mes, Math.min(dia, new Date(Date.UTC(anio, mes, 0)).getUTCDate()));
}

function controlesCalendarioHTML(anio, vista, seleccionada, t, locale) {
  const mes = Number(seleccionada.slice(5, 7));
  const opciones = Array.from({ length: 12 }, (_, i) => {
    const nombre = new Intl.DateTimeFormat(locale, { timeZone: "UTC", month: "long" }).format(new Date(Date.UTC(anio, i, 1)));
    return `<option value="${i + 1}"${mes === i + 1 ? " selected" : ""}>${escaparHTML(nombre)}</option>`;
  }).join("");
  const meses = vista === "mes" ? `<nav class="cronos-solicitud-acciones" aria-label="${escaparHTML(t("calendario_mes", { mes: new Intl.DateTimeFormat(locale, { timeZone: "UTC", month: "long" }).format(new Date(Date.UTC(anio, mes - 1, 1))), anio }))}">
    <button type="button" class="boton-secundario" data-cronos-cal-mes="-1"${anio === 2000 && mes === 1 ? " disabled" : ""}>${escaparHTML(t("calendario_mes_anterior"))}</button>
    <button type="button" class="boton-secundario" data-cronos-cal-mes="1"${anio === 2100 && mes === 12 ? " disabled" : ""}>${escaparHTML(t("calendario_mes_siguiente"))}</button></nav>` : "";
  return `<div class="cronos-solicitud-formulario">
    <label>${escaparHTML(t("calendario_vista"))}<select class="control-formulario" data-cronos-cal-vista><option value="anio"${vista === "anio" ? " selected" : ""}>${escaparHTML(t("calendario_vista_anio"))}</option><option value="mes"${vista === "mes" ? " selected" : ""}>${escaparHTML(t("calendario_vista_mes"))}</option></select></label>
    <label>${escaparHTML(t("calendario_elegir_mes"))}<select class="control-formulario" data-cronos-cal-mes-elegido>${opciones}</select></label>
    <label>${escaparHTML(t("calendario_fecha"))}<input class="control-formulario" type="date" data-cronos-cal-fecha min="2000-01-01" max="2100-12-31" value="${seleccionada}"></label>
    <button type="button" class="boton-secundario" data-cronos-cal-hoy>${escaparHTML(t("calendario_hoy"))}</button>
  </div>${meses}`;
}

function detalleDiaHTML(datos, fecha, t, locale) {
  const detalles = [];
  const poner = (valor) => detalles.push(`<li>${escaparHTML(valor)}</li>`);
  for (const d of datos.calendario.dias.filter((d) => d.fecha === fecha)) poner(t("calendario_dato_dia", { tipo: nombreTipo(d.tipo, t), nombre: d.nombre }));
  const marcajes = datos.marcajes_por_dia.filter((d) => d.fecha === fecha).reduce((total, d) => total + d.marcajes, 0);
  if (marcajes) poner(t("calendario_marcajes_dia", { cantidad: new Intl.NumberFormat(locale).format(marcajes) }));
  for (const a of datos.absentismos.filter((a) => a.desde <= fecha && a.hasta >= fecha)) poner(t("calendario_dato_dia", { tipo: nombreTipo("ausencia", t), nombre: a.nombre }));
  for (const c of datos.correcciones.filter((c) => c.fecha_civil === fecha)) poner(t("calendario_olvido_dia", { tipo: nombreTipo("olvido", t), hora: c.hora_pretendida, movimiento: t(`movimiento_${c.movimiento}`), estado: t(`correccion_${c.estado}`) }));
  return `<section class="panel cronos-panel" id="cronos-movpropios-detalle" aria-labelledby="cronos-movpropios-dia"><div class="cabecera-panel"><h3 id="cronos-movpropios-dia">${escaparHTML(t("calendario_detalle_dia"))}</h3></div><div class="cuerpo-panel" role="status" aria-live="polite"><strong>${escaparHTML(fechaVisible(fecha, locale, "full"))}</strong>${detalles.length ? `<ul>${detalles.join("")}</ul>` : `<p>${escaparHTML(t("calendario_sin_datos_dia"))}</p>`}</div></section>`;
}

function tabla(cabeceras, filas, t) {
  const cuerpo = filas.length ? filas.join("") : `<tr><td colspan="${cabeceras.length}">${escaparHTML(t("sin_filas"))}</td></tr>`;
  return `<div class="cronos-tabla-contenedor"><table class="cronos-tabla"><thead><tr>${cabeceras.map((c) => `<th scope="col">${escaparHTML(t(c))}</th>`).join("")}</tr></thead><tbody>${cuerpo}</tbody></table></div>`;
}

function formularioOlvido(f, t, hoy) {
  if (!f?.abierto) return `<button type="button" class="boton-primario" data-cronos-olvido="abrir">${escaparHTML(t("olvido_solicitar"))}</button>`;
  const enviando = f.estado === "enviando";
  const opciones = MOVIMIENTOS.map((m) => `<option value="${m}"${f.movimiento === m ? " selected" : ""}>${escaparHTML(t(`movimiento_${m}`))}</option>`).join("");
  const aviso = f.mensaje ? `<p class="cronos-solicitud-aviso" data-tono="${f.estado === "hecho" ? "exito" : "error"}" role="${f.estado === "hecho" ? "status" : "alert"}">${escaparHTML(f.mensaje)}</p>` : "";
  return `<form class="cronos-solicitud-formulario" data-cronos-olvido-formulario tabindex="-1" aria-label="${escaparHTML(t("olvido_formulario"))}">
    <label>${escaparHTML(t("fecha"))}<input class="control-formulario" type="date" name="fecha_civil" required max="${hoy}" value="${escaparHTML(f.fecha ?? "")}"${enviando ? " disabled" : ""}></label>
    <label>${escaparHTML(t("hora"))}<input class="control-formulario" type="time" name="hora_pretendida" required value="${escaparHTML(f.hora ?? "")}"${enviando ? " disabled" : ""}></label>
    <label>${escaparHTML(t("olvido_movimiento"))}<select class="control-formulario" name="movimiento"${enviando ? " disabled" : ""}>${opciones}</select></label>
    <div class="cronos-solicitud-acciones"><button type="submit" class="boton-primario" data-cronos-olvido-enviar${enviando ? " disabled" : ""}>${escaparHTML(t(enviando ? "olvido_enviando" : "olvido_enviar"))}</button>
    <button type="button" class="boton-secundario" data-cronos-olvido="cerrar"${enviando ? " disabled" : ""}>${escaparHTML(t("cancelar"))}</button></div>${aviso}</form>`;
}

function validarTamanoPagina(tamano) {
  if (!Number.isSafeInteger(tamano) || tamano < 1 || tamano > 100) throw new RangeError("tamaño de página Cronos no válido");
}

/** Filtros locales: ausencia significa un dato recibido, nunca falta de marcaje. */
export function filtrarIncidenciasPropiasCronos(datos, filtros = {}) {
  const desde = filtros.desde || datos.periodo.desde;
  const hasta = filtros.hasta || datos.periodo.hasta;
  if (desde > hasta) return { correcciones: [], absentismos: [] };
  return {
    correcciones: datos.correcciones.filter((c) => c.fecha_civil >= desde && c.fecha_civil <= hasta && (!filtros.estado || c.estado === filtros.estado)),
    absentismos: datos.absentismos.filter((a) => a.hasta >= desde && a.desde <= hasta && (!filtros.justificante || (filtros.justificante === "pendiente") === a.pendiente_justificar)),
  };
}

function paginaHTML(filas, pagina, tamano, nombre, titulo, t, locale) {
  const actual = Math.max(1, Math.min(pagina, Math.max(1, Math.ceil(filas.length / tamano))));
  const inicio = (actual - 1) * tamano;
  const n = (valor) => new Intl.NumberFormat(locale).format(valor);
  return { filas: filas.slice(inicio, inicio + tamano), pie: `<nav class="cronos-solicitud-acciones cuerpo-panel" aria-label="${escaparHTML(t("paginacion", { tabla: t(titulo) }))}"><span role="status" aria-live="polite">${escaparHTML(t("mostrando", { desde: n(filas.length ? inicio + 1 : 0), hasta: n(Math.min(inicio + tamano, filas.length)), total: n(filas.length) }))}</span><button type="button" class="boton-secundario" data-cronos-pagina="${nombre}" data-pagina="${actual - 1}"${actual === 1 ? " disabled" : ""}>${escaparHTML(t("pagina_anterior"))}</button><button type="button" class="boton-secundario" data-cronos-pagina="${nombre}" data-pagina="${actual + 1}"${inicio + tamano >= filas.length ? " disabled" : ""}>${escaparHTML(t("pagina_siguiente"))}</button></nav>` };
}

function filtrosHTML(datos, filtros, t, locale) {
  const opcion = (valor, etiqueta, actual) => `<option value="${valor}"${actual === valor ? " selected" : ""}>${escaparHTML(t(etiqueta))}</option>`;
  const n = (valor) => new Intl.NumberFormat(locale).format(valor);
  const estados = ESTADOS_CORRECCION.map((e) => `<option value="${e}"${filtros.estado === e ? " selected" : ""}>${escaparHTML(t(`correccion_${e}`))} (${n(datos.correcciones.filter((c) => c.estado === e).length)})</option>`).join("");
  const resumen = [["correcciones_total", datos.correcciones.length, "correcciones", "reloj"], ["ausencias_total", datos.absentismos.length, "ausencias", "calendario"], ["justificaciones_total", datos.absentismos.filter((a) => a.pendiente_justificar).length, "pendiente", "documento"]]
    .map(([clave, cantidad, filtro, nombreIcono]) => `<button type="button" class="tarjeta-kpi" data-cronos-recuento="${filtro}"><span class="icono-kpi" aria-hidden="true">${icono(nombreIcono)}</span><span><strong class="valor-kpi">${n(cantidad)}</strong><span class="etiqueta-kpi">${escaparHTML(t(clave))}</span></span></button>`).join("");
  return `<section class="panel cronos-panel" aria-labelledby="cronos-incidencias-titulo"><div class="cabecera-panel"><h3 id="cronos-incidencias-titulo">${escaparHTML(t("incidencias"))}</h3></div><div class="cuerpo-panel"><div class="rejilla-kpi rejilla-kpi--compacta">${resumen}</div><div class="cronos-solicitud-formulario">
    <label>${escaparHTML(t("desde_filtro"))}<input class="control-formulario" type="date" data-cronos-filtro="desde" value="${escaparHTML(filtros.desde || "")}"></label>
    <label>${escaparHTML(t("hasta_filtro"))}<input class="control-formulario" type="date" data-cronos-filtro="hasta" value="${escaparHTML(filtros.hasta || "")}"></label>
    <label>${escaparHTML(t("filtrar_estado"))}<select class="control-formulario" data-cronos-filtro="estado">${opcion("", "todos_estados", filtros.estado || "")}${estados}</select></label>
    <label>${escaparHTML(t("filtrar_justificante"))}<select class="control-formulario" data-cronos-filtro="justificante">${opcion("", "todos_justificantes", filtros.justificante || "")}${opcion("pendiente", "justificante_pendiente", filtros.justificante)}${opcion("completo", "justificante_ok", filtros.justificante)}</select></label>
    <button type="button" class="boton-secundario" data-cronos-quitar-filtros>${escaparHTML(t("quitar_filtros"))}</button></div>${filtros.desde && filtros.hasta && filtros.desde > filtros.hasta ? `<p role="alert">${escaparHTML(t("periodo_invalido"))}</p>` : ""}</div></section>`;
}

// El cliente HTTP valida el sobre; los clientes inyectados entregan el DTO del recibo.
function validarReciboInyectado(recibo, entrada) {
  const claves = ["solicitud_ref", "actuacion_ref", "recibo_ref", "estado", "version", "instante_utc", "replay"];
  if (!recibo || claves.some((c) => !Object.hasOwn(recibo, c)) || Object.keys(recibo).some((c) => !claves.includes(c))
    || recibo.solicitud_ref !== `correccion:cronos:${entrada.clave_operacion}`
    || typeof recibo.recibo_ref !== "string" || !/^recibo:cronos:[A-Za-z0-9:._-]+$/u.test(recibo.recibo_ref) || recibo.recibo_ref.length > 200
    || recibo.estado !== "pendiente_responsable" || recibo.version !== 1 || typeof recibo.replay !== "boolean"
    || typeof recibo.instante_utc !== "string" || !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?(?:Z|[+-]\d{2}:\d{2})$/u.test(recibo.instante_utc) || !Number.isFinite(Date.parse(recibo.instante_utc))) {
    throw new ErrorClienteSolicitudesCronos("respuesta_incompatible", 200);
  }
  return recibo;
}

/** Calendario anual con los días marcados por tipo, ausencias y olvidos comunicados. */
export function renderizarMovimientosPropiosCronos({ estado = "cargando", anio, datos = null, formulario = null, mensajes = MENSAJES_CRONOS_SOLICITUDES,
  locale = LOCALIZACION_ACTUAL, zonaHoraria = "Europe/Madrid", hoy = hoyCivilCronos(zonaHoraria), incrustada = false, filtros = {}, paginas = {}, tamanoPagina = 10, vistaCalendario = "anio", fechaSeleccionada } = {}) {
  validarTamanoPagina(tamanoPagina);
  const t = crearTraductorIncidenciasCronos(mensajes);
  if (!Number.isInteger(anio) || anio < 2000 || anio > 2100) throw new RangeError("año de Cronos no válido");
  const seleccionada = fechaSeleccionada ?? fechaEnAnio(hoy, anio);
  if (!["anio", "mes"].includes(vistaCalendario) || !fechaCivilValida(seleccionada) || Number(seleccionada.slice(0, 4)) !== anio) throw new RangeError("vista de calendario Cronos no válida");
  if (estado === "listo") {
    try {
      validarMovimientosPropiosCronos(datos, { periodo: datos?.periodo?.tipo, desde: datos?.periodo?.desde, hasta: datos?.periodo?.hasta });
      if (datos.periodo.desde !== `${anio}-01-01` || datos.periodo.hasta !== `${anio}-12-31`) throw new ErrorClienteSolicitudesCronos("respuesta_incompatible", 200);
    }
    catch { estado = "error"; datos = null; }
  }
  const ayuda = t("abrir_ayuda", { asunto: t("calendario_titulo") });
  const cabecera = `<header class="cronos-encabezado"><div>${encabezadoParte(t("sobrelinea"), "cronos-movpropios-titulo", t("calendario_titulo"), incrustada)}</div>
    <button type="button" class="boton-secundario" data-cronos-actualizar${estado === "cargando" ? " disabled" : ""}>${escaparHTML(t(estado === "cargando" ? "actualizando" : "actualizar"))}</button><button type="button" class="cronos-boton-ayuda" data-accion="ayuda" aria-label="${escaparHTML(ayuda)}" title="${escaparHTML(ayuda)}"><span aria-hidden="true">?</span></button></header>`;
  const navegacion = `<nav class="cronos-anio-navegacion" aria-label="${escaparHTML(t("calendario_anual", { anio }))}"><button type="button" class="boton-secundario" data-cronos-anio="-1" aria-label="${escaparHTML(t("anio_anterior"))}"${anio <= 2000 ? " disabled" : ""}>‹</button><strong aria-live="polite">${anio}</strong><button type="button" class="boton-secundario" data-cronos-anio="1" aria-label="${escaparHTML(t("anio_siguiente"))}"${anio >= 2100 ? " disabled" : ""}>›</button></nav>`;
  let cuerpo;
  if (estado !== "listo") {
    const clave = { denegado: "denegado", sin_empleado: "sin_empleado", error: "error", no_disponible: "no_disponible" }[estado] ?? "cargando";
    cuerpo = `<section class="panel cronos-panel"><div class="cuerpo-panel"><p class="cronos-${["cargando", "no_disponible"].includes(estado) ? "vacio" : "acceso-denegado"}" data-cronos-movimientos-estado tabindex="-1" role="${estado === "error" ? "alert" : "status"}">${escaparHTML(t(clave))}</p></div></section>`;
  } else {
    const marcas = marcasPorDiaCronos(datos);
    const aviso = datos.calendario.disponible ? "" : `<p class="cronos-calendario-falta" role="status">${escaparHTML(t("calendario_sin_publicar", { anio }))}</p>`;
    const leyenda = `<ul class="cronos-leyenda" aria-label="${escaparHTML(t("leyenda"))}">${ORDEN_TIPOS.map((tipo) => `<li><span class="cronos-leyenda-muestra" data-tipos="${tipo}" aria-hidden="true"></span>${escaparHTML(nombreTipo(tipo, t))}</li>`).join("")}</ul>`;
    const indices = vistaCalendario === "mes" ? [Number(seleccionada.slice(5, 7))] : Array.from({ length: 12 }, (_, i) => i + 1);
    const meses = indices.map((mes) => mesHTML(anio, mes, marcas, t, locale, seleccionada)).join("");
    const calendarioEstado = datos.calendario.disponible ? "disponible" : "no_configurado";
    const indicador = `<span class="cronos-estado${datos.calendario.disponible ? "" : " cronos-estado-aviso"}" data-calendario-estado="${calendarioEstado}">${escaparHTML(t(`calendario_${calendarioEstado}`))}</span>`;
    const filtrados = filtrarIncidenciasPropiasCronos(datos, filtros);
    const ausencias = filtrados.absentismos.map((a) => `<tr><th scope="row">${escaparHTML(a.nombre)}</th><td>${escaparHTML(fechaVisible(a.desde, locale))}</td><td>${escaparHTML(fechaVisible(a.hasta, locale))}</td><td class="numero">${escaparHTML(formatearCantidadCronos(a.cantidad, a.unidad, t, locale))}</td><td>${a.pendiente_justificar ? `<span class="cronos-estado cronos-estado-aviso">${escaparHTML(t("justificante_pendiente"))}</span>` : escaparHTML(t("justificante_ok"))}</td></tr>`);
    const olvidos = filtrados.correcciones.map((c) => `<tr><th scope="row">${escaparHTML(fechaVisible(c.fecha_civil, locale))}</th><td>${escaparHTML(c.hora_pretendida)}</td><td>${escaparHTML(t(`movimiento_${c.movimiento}`))}</td><td><span class="cronos-estado" data-estado="${escaparHTML(c.estado)}">${escaparHTML(t(`correccion_${c.estado}`))}</span></td><td>${escaparHTML(t(`paso_${c.estado}`))}</td></tr>`);
    const paginaAusencias = paginaHTML(ausencias, paginas.ausencias || 1, tamanoPagina, "ausencias", "absentismos_titulo", t, locale);
    const paginaOlvidos = paginaHTML(olvidos, paginas.correcciones || 1, tamanoPagina, "correcciones", "olvidos_titulo", t, locale);
    const resultados = t("resultados", { correcciones: new Intl.NumberFormat(locale).format(filtrados.correcciones.length), ausencias: new Intl.NumberFormat(locale).format(filtrados.absentismos.length) });
    cuerpo = `${filtrosHTML(datos, filtros, t, locale)}<p class="cronos-sr" role="status" aria-live="polite">${escaparHTML(resultados)}</p><section class="panel cronos-panel" aria-labelledby="cronos-movpropios-cal"><div class="cabecera-panel"><h3 id="cronos-movpropios-cal">${escaparHTML(t("calendario_anual", { anio }))}</h3>${navegacion}</div><div class="cuerpo-panel">${indicador}${aviso}${controlesCalendarioHTML(anio, vistaCalendario, seleccionada, t, locale)}${leyenda}<div class="cronos-meses" data-calendario-vista="${vistaCalendario}">${meses}</div></div></section>${detalleDiaHTML(datos, seleccionada, t, locale)}
    <section class="panel cronos-panel" aria-labelledby="cronos-movpropios-aus"><div class="cabecera-panel"><h3 id="cronos-movpropios-aus">${escaparHTML(t("absentismos_titulo"))}</h3></div>${tabla(["permiso", "desde", "hasta", "duracion", "estado"], paginaAusencias.filas, t)}${paginaAusencias.pie}${datos.absentismos.some((a) => a.pendiente_justificar) ? `<div class="cuerpo-panel"><p>${escaparHTML(t("justificacion_no_conectada"))}</p></div>` : ""}</section>
    <section class="panel cronos-panel" aria-labelledby="cronos-movpropios-olv"><div class="cabecera-panel"><h3 id="cronos-movpropios-olv">${escaparHTML(t("olvidos_titulo"))}</h3></div>${tabla(["fecha", "hora", "olvido_movimiento", "estado", "siguiente_paso"], paginaOlvidos.filas, t)}${paginaOlvidos.pie}<div class="cuerpo-panel" data-cronos-olvido-zona>${formularioOlvido(formulario, t, hoy)}</div></section>`;
  }
  return `<section class="cronos-area cronos-movimientos-propios" aria-labelledby="cronos-movpropios-titulo" data-estado="${escaparHTML(estado)}">${cabecera}${estado === "listo" ? "" : `<section class="panel cronos-panel"><div class="cabecera-panel">${navegacion}</div></section>`}${cuerpo}</section>`;
}

function estadoError(error) {
  if (error instanceof ErrorClienteSolicitudesCronos) {
    if (error.codigo === "sin_empleado") return "sin_empleado";
    if (["acceso_denegado", "autenticacion_requerida"].includes(error.codigo)) return "denegado";
    if (error.estado === 404) return "no_disponible";
  }
  return "error";
}

/** Consulta el año y registra olvidos sin perder filtros ni el borrador en memoria. */
export function montarMovimientosPropiosCronos({ raiz, cliente = crearClienteSolicitudesCronosHTTP(), mensajes = MENSAJES_CRONOS_SOLICITUDES,
  anunciar = () => {}, registrarDesmontar, locale = LOCALIZACION_ACTUAL, zonaHoraria = "Europe/Madrid", anio, abrirOlvido = false, incrustada = false, tamanoPagina = 10, vistaCalendario = "anio", fechaSeleccionada } = {}) {
  validarTamanoPagina(tamanoPagina);
  if (!raiz?.append || !raiz.ownerDocument?.createElement || typeof cliente?.consultarMovimientos !== "function" || typeof cliente?.solicitarCorreccion !== "function"
    || typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) throw new TypeError("montaje de movimientos propios Cronos no disponible");
  const t = crearTraductorIncidenciasCronos(mensajes);
  const contenedor = raiz.ownerDocument.createElement("section"); contenedor.dataset.cronosMovimientosPropios = ""; raiz.append(contenedor);
  const hoy = hoyCivilCronos(zonaHoraria);
  let activa = true; let secuencia = 0; let controlador = null; let envio = null;
  let anioVisible = Number.isInteger(anio) ? anio : Number(hoy.slice(0, 4));
  let seleccionada = fechaSeleccionada ?? fechaEnAnio(hoy, anioVisible);
  let vista = vistaCalendario;
  let estado = "cargando"; let datos = null;
  let filtros = {}; let paginas = {}; let focoPendiente = null;
  let formulario = abrirOlvido ? { abierto: true, clave: claveNueva(), movimiento: "entrada", fecha: hoy } : null;
  const dibujar = () => {
    if (!activa) return;
    const foco = contenedor.querySelector?.(":focus");
    const documento = contenedor.ownerDocument ?? raiz.ownerDocument;
    const documentoActivo = documento.hasFocus?.() !== false;
    const selectorActual = !documentoActivo ? null
      : foco?.name && ["fecha_civil", "hora_pretendida", "movimiento"].includes(foco.name) ? `[name="${foco.name}"]`
      : foco?.hasAttribute?.("data-cronos-olvido-enviar") ? "[data-cronos-olvido-enviar]"
      : foco?.dataset?.cronosOlvido ? `[data-cronos-olvido="${foco.dataset.cronosOlvido}"]`
      : foco?.dataset?.cronosRecuento ? `[data-cronos-recuento="${foco.dataset.cronosRecuento}"]`
      : foco?.hasAttribute?.("data-cronos-quitar-filtros") ? "[data-cronos-quitar-filtros]"
      : foco?.getAttribute?.("data-accion") === "ayuda" ? '[data-accion="ayuda"]'
      : foco?.hasAttribute?.("data-cronos-cal-dia") && fechaCivilValida(foco.dataset?.fecha) ? `[data-cronos-cal-dia][data-fecha="${foco.dataset.fecha}"]`
      : foco?.hasAttribute?.("data-cronos-cal-vista") ? "[data-cronos-cal-vista]"
      : foco?.hasAttribute?.("data-cronos-cal-mes-elegido") ? "[data-cronos-cal-mes-elegido]"
      : foco?.hasAttribute?.("data-cronos-cal-fecha") ? "[data-cronos-cal-fecha]"
      : foco?.hasAttribute?.("data-cronos-cal-hoy") ? "[data-cronos-cal-hoy]"
      : foco?.dataset?.cronosCalMes ? `[data-cronos-cal-mes="${foco.dataset.cronosCalMes}"]`
      : foco?.dataset?.cronosFiltro ? `[data-cronos-filtro="${foco.dataset.cronosFiltro}"]`
      : foco?.hasAttribute?.("data-cronos-actualizar") ? "[data-cronos-actualizar]"
      : foco?.dataset?.cronosAnio ? `[data-cronos-anio="${foco.dataset.cronosAnio}"]`
      : foco?.dataset?.cronosPagina ? `[data-cronos-pagina="${foco.dataset.cronosPagina}"][data-pagina="${foco.dataset.pagina}"]` : null;
    const activoDocumento = documento.activeElement;
    if (!documentoActivo || (activoDocumento && activoDocumento !== documento.body && !contenedor.contains?.(activoDocumento))) focoPendiente = null;
    const selector = selectorActual || focoPendiente;
    focoPendiente = selector;
    contenedor.innerHTML = renderizarMovimientosPropiosCronos({ estado, anio: anioVisible, datos, formulario, mensajes, locale, zonaHoraria, hoy, incrustada, filtros, paginas, tamanoPagina, vistaCalendario: vista, fechaSeleccionada: seleccionada });
    if (selector) {
      const control = contenedor.querySelector?.(selector);
      const destino = control && !control.disabled ? control
        : control?.disabled && selector.startsWith("[data-cronos-cal-mes=") ? contenedor.querySelector?.("[data-cronos-cal-fecha]")
          : contenedor.querySelector?.("[data-cronos-movimientos-estado]")
            ?? contenedor.querySelector?.("[data-cronos-olvido-formulario]");
      destino?.focus?.();
      // El formulario y el mensaje son destinos provisionales mientras el
      // control está deshabilitado u oculto por la lectura.
      if (estado !== "cargando" && formulario?.estado !== "enviando" && destino === control && !control?.disabled) focoPendiente = null;
    }
  };
  const cargar = async () => {
    if (!activa) return;
    controlador?.abort(); const abortador = new AbortController(); controlador = abortador; const turno = ++secuencia;
    const anioConsulta = anioVisible;
    estado = "cargando"; datos = null; dibujar();
    const consulta = String(anioConsulta) === hoy.slice(0, 4) ? { periodo: "anio" } : { periodo: "rango", desde: `${anioConsulta}-01-01`, hasta: `${anioConsulta}-12-31` };
    try {
      const r = await cliente.consultarMovimientos(consulta, { signal: abortador.signal });
      if (!activa || turno !== secuencia || abortador.signal.aborted) return;
      validarMovimientosPropiosCronos(r, consulta);
      if (r.periodo.desde !== `${anioConsulta}-01-01` || r.periodo.hasta !== `${anioConsulta}-12-31`) throw new ErrorClienteSolicitudesCronos("respuesta_incompatible", 200);
      estado = "listo"; datos = r; dibujar(); anunciar(t("actualizado"));
    } catch (error) {
      if (!activa || turno !== secuencia || abortador.signal.aborted) return;
      estado = estadoError(error); dibujar(); anunciar(t({ denegado: "denegado", sin_empleado: "sin_empleado", no_disponible: "no_disponible" }[estado] ?? "error"));
    }
  };
  const seleccionarFecha = (fecha, mensual = true) => {
    if (!activa || !fechaCivilValida(fecha)) return;
    const nuevoAnio = Number(fecha.slice(0, 4));
    const cambiaAnio = nuevoAnio !== anioVisible;
    seleccionada = fecha; anioVisible = nuevoAnio;
    if (mensual) vista = "mes";
    if (cambiaAnio) { paginas = {}; void cargar(); }
    else dibujar();
  };
  const alPulsar = (evento) => {
    if (!activa) return;
    const dia = evento.target?.closest?.("[data-cronos-cal-dia]");
    if (dia) {
      const fecha = dia.dataset.fecha;
      if (!dia.disabled && estado === "listo" && fechaCivilValida(fecha) && fecha >= datos.periodo.desde && fecha <= datos.periodo.hasta) seleccionarFecha(fecha, false);
      return;
    }
    if (evento.target?.closest?.("[data-cronos-cal-hoy]")) { seleccionarFecha(hoy); return; }
    const mesBoton = evento.target?.closest?.("[data-cronos-cal-mes]");
    if (mesBoton && !mesBoton.disabled && vista === "mes") {
      const cambio = Number(mesBoton.dataset.cronosCalMes);
      if (![-1, 1].includes(cambio)) return;
      const [anio, mes, dia] = seleccionada.split("-").map(Number);
      const primero = new Date(Date.UTC(anio, mes - 1 + cambio, 1));
      const a = primero.getUTCFullYear(); const m = primero.getUTCMonth() + 1;
      seleccionarFecha(iso(a, m, Math.min(dia, new Date(Date.UTC(a, m, 0)).getUTCDate())));
      return;
    }
    if (evento.target?.closest?.("[data-cronos-actualizar]")) { void cargar(); return; }
    const limpiar = evento.target?.closest?.("[data-cronos-quitar-filtros]");
    const recuento = evento.target?.closest?.("[data-cronos-recuento]");
    if (limpiar || recuento) {
      filtros = {}; paginas = {};
      if (recuento?.dataset.cronosRecuento === "pendiente") filtros.justificante = "pendiente";
      dibujar();
      const destino = recuento ? (recuento.dataset.cronosRecuento === "correcciones" ? "[data-cronos-filtro=estado]" : "[data-cronos-filtro=justificante]") : "[data-cronos-filtro=desde]";
      contenedor.querySelector?.(destino)?.focus?.();
      return;
    }
    const pagina = evento.target?.closest?.("[data-cronos-pagina]");
    if (pagina && !pagina.disabled) {
      const nombre = pagina.dataset.cronosPagina; const siguiente = Number(pagina.dataset.pagina);
      if (["correcciones", "ausencias"].includes(nombre) && Number.isSafeInteger(siguiente) && siguiente >= 1 && siguiente <= 2000) {
        paginas[nombre] = siguiente; dibujar();
        contenedor.querySelector?.(`[data-cronos-pagina="${nombre}"]:not([disabled])`)?.focus?.();
      }
      return;
    }
    const anioBoton = evento.target?.closest?.("[data-cronos-anio]");
    if (anioBoton && !anioBoton.disabled) {
      const cambio = Number(anioBoton.dataset.cronosAnio);
      const siguiente = anioVisible + cambio;
      if ([-1, 1].includes(cambio) && siguiente >= 2000 && siguiente <= 2100) seleccionarFecha(fechaEnAnio(seleccionada, siguiente), false);
      return;
    }
    const olvido = evento.target?.closest?.("[data-cronos-olvido]");
    if (!olvido || formulario?.estado === "enviando") return;
    if (olvido.dataset.cronosOlvido === "abrir") {
      if (!formulario) formulario = { abierto: true, clave: claveNueva(), movimiento: "entrada", fecha: hoy };
    } else if (olvido.dataset.cronosOlvido === "cerrar") formulario = null;
    dibujar();
    contenedor.querySelector?.(formulario ? "[name=fecha_civil]" : "[data-cronos-olvido=abrir]")?.focus?.();
  };
  const alEditar = (evento) => {
    if (!activa) return;
    const campo = evento.target;
    if (evento.type === "change") {
      if (campo?.hasAttribute?.("data-cronos-cal-vista")) {
        if (["anio", "mes"].includes(campo.value)) { vista = campo.value; dibujar(); }
        return;
      }
      if (campo?.hasAttribute?.("data-cronos-cal-fecha")) { seleccionarFecha(campo.value); return; }
      if (campo?.hasAttribute?.("data-cronos-cal-mes-elegido")) {
        const mes = Number(campo.value); const dia = Number(seleccionada.slice(8, 10));
        if (Number.isInteger(mes) && mes >= 1 && mes <= 12) seleccionarFecha(iso(anioVisible, mes, Math.min(dia, new Date(Date.UTC(anioVisible, mes, 0)).getUTCDate())));
        return;
      }
    }
    if (formulario && formulario.estado !== "enviando" && ["fecha_civil", "hora_pretendida", "movimiento"].includes(campo?.name)) {
      const clave = { fecha_civil: "fecha", hora_pretendida: "hora", movimiento: "movimiento" }[campo.name];
      formulario[clave] = campo.value;
    }
    if (evento.type !== "change" || !["desde", "hasta", "estado", "justificante"].includes(campo?.dataset?.cronosFiltro)) return;
    const nombre = campo.dataset.cronosFiltro;
    if (nombre === "estado" && campo.value && !ESTADOS_CORRECCION.includes(campo.value)) return;
    if (nombre === "justificante" && !["", "pendiente", "completo"].includes(campo.value)) return;
    filtros[nombre] = campo.value; paginas = {}; dibujar();
  };
  const alEnviar = async (evento) => {
    if (!evento.target?.matches?.("[data-cronos-olvido-formulario]")) return;
    evento.preventDefault();
    if (!formulario || formulario.estado === "enviando" || envio) return;
    const valor = (n) => evento.target.elements?.namedItem?.(n)?.value ?? "";
    const entrada = { fecha: valor("fecha_civil"), hora: valor("hora_pretendida"), movimiento: valor("movimiento") };
    if (formulario.estado === "hecho" || (formulario.enviada && (formulario.enviada.fecha !== entrada.fecha || formulario.enviada.hora !== entrada.hora || formulario.enviada.movimiento !== entrada.movimiento))) formulario.clave = claveNueva();
    formulario = { ...formulario, ...entrada, estado: "enviando", mensaje: "", enviada: entrada };
    dibujar();
    const abortador = new AbortController(); envio = abortador;
    try {
      const cuerpo = validarEntradaCorreccionCronos({ clave_operacion: formulario.clave, movimiento: entrada.movimiento, fecha_civil: entrada.fecha, hora_pretendida: entrada.hora });
      const recibo = await cliente.solicitarCorreccion(cuerpo, { signal: abortador.signal });
      if (!activa || abortador.signal.aborted || envio !== abortador) return;
      validarReciboInyectado(recibo, cuerpo);
      formulario = { ...formulario, estado: "hecho", mensaje: t(recibo.replay ? "olvido_ya_registrado" : "olvido_registrado", { fecha: instanteVisible(recibo.instante_utc, locale, zonaHoraria) }), reciboRef: recibo.recibo_ref };
      anunciar(formulario.mensaje);
      await cargar();
    } catch (error) {
      if (!activa || abortador.signal.aborted || envio !== abortador) return;
      const codigo = error instanceof ErrorClienteSolicitudesCronos || error instanceof TypeError ? (error.codigo ?? "peticion_invalida") : "";
      const clave = codigo === "peticion_invalida" ? "olvido_invalido" : codigo === "conflicto" ? "olvido_conflicto" : "olvido_error";
      formulario = { ...formulario, estado: "error", mensaje: t(clave) };
      dibujar(); anunciar(formulario.mensaje);
    } finally { if (envio === abortador) envio = null; }
  };
  contenedor.addEventListener("click", alPulsar); contenedor.addEventListener("submit", alEnviar);
  contenedor.addEventListener("input", alEditar); contenedor.addEventListener("change", alEditar);
  void cargar();
  const desmontar = () => {
    if (!activa) return;
    activa = false; ++secuencia; controlador?.abort(); envio?.abort();
    contenedor.removeEventListener("click", alPulsar); contenedor.removeEventListener("submit", alEnviar);
    contenedor.removeEventListener("input", alEditar); contenedor.removeEventListener("change", alEditar); contenedor.remove?.();
  };
  const abrirFormularioOlvido = () => {
    if (!activa) return false;
    if (estado !== "listo") {
      const estadoVisible = contenedor.querySelector?.("[data-cronos-movimientos-estado]");
      estadoVisible?.scrollIntoView?.({ block: "center" }); estadoVisible?.focus?.();
      const clave = ["cargando", "denegado", "sin_empleado", "error", "no_disponible"].includes(estado) ? estado : "error";
      anunciar(t(clave));
      return false;
    }
    if (!formulario) { formulario = { abierto: true, clave: claveNueva(), movimiento: "entrada", fecha: hoy }; dibujar(); }
    const campo = contenedor.querySelector?.("[name=fecha_civil]");
    campo?.scrollIntoView?.({ block: "center" }); campo?.focus?.();
    return true;
  };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ desmontar, recargar: cargar, actualizar: cargar, abrirOlvido: abrirFormularioOlvido });
}
