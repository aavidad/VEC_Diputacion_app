import { crearTraductorSolicitudesCronos, formatearCantidadCronos, MENSAJES_CRONOS_SOLICITUDES_ES } from "./i18n-solicitudes.js";
import { ErrorClienteSolicitudesCronos, crearClienteSolicitudesCronosHTTP } from "./cliente-solicitudes-http.js";

const MOVIMIENTOS = ["entrada", "salida", "inicio_pausa", "fin_pausa"];
const ORDEN_TIPOS = ["festivo", "no_laborable", "ausencia", "olvido", "marcaje"];

function escaparHTML(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}
function iso(anio, mes, dia) { return `${anio}-${String(mes).padStart(2, "0")}-${String(dia).padStart(2, "0")}`; }
function fechaVisible(fecha, locale, estilo = "medium") {
  const [a, m, d] = fecha.split("-").map(Number);
  return new Intl.DateTimeFormat(locale, { timeZone: "UTC", dateStyle: estilo }).format(new Date(Date.UTC(a, m - 1, d, 12)));
}
function instanteVisible(instante, locale, zona) {
  return new Intl.DateTimeFormat(locale, { timeZone: zona, dateStyle: "medium", timeStyle: "short" }).format(new Date(instante));
}
export function hoyCivilCronos(zona = "Europe/Madrid", ahora = new Date()) {
  const p = new Intl.DateTimeFormat("en-CA", { timeZone: zona, year: "numeric", month: "2-digit", day: "2-digit" }).formatToParts(ahora);
  const v = (t) => p.find((x) => x.type === t)?.value;
  return `${v("year")}-${v("month")}-${v("day")}`;
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
    for (let f = new Date(`${a.desde}T12:00:00Z`); f <= new Date(`${a.hasta}T12:00:00Z`); f.setUTCDate(f.getUTCDate() + 1)) {
      const fecha = f.toISOString().slice(0, 10);
      if (fecha >= datos.periodo.desde && fecha <= datos.periodo.hasta) poner(fecha, "ausencia");
    }
  }
  return marcas;
}

function nombreTipo(tipo, t) { return t(`tipo_${tipo}`); }

function mesHTML(anio, mes, marcas, t, locale) {
  const dias = new Date(Date.UTC(anio, mes, 0)).getUTCDate();
  const huecos = (new Date(Date.UTC(anio, mes - 1, 1)).getUTCDay() + 6) % 7;
  const celdas = Array.from({ length: huecos }, () => '<td aria-hidden="true"></td>');
  for (let dia = 1; dia <= dias; dia++) {
    const fecha = iso(anio, mes, dia);
    const tipos = ORDEN_TIPOS.filter((tipo) => marcas.get(fecha)?.has(tipo));
    const etiqueta = tipos.length ? t("dia_con", { fecha: fechaVisible(fecha, locale, "full"), tipos: tipos.map((x) => nombreTipo(x, t)).join(", ") }) : fechaVisible(fecha, locale, "full");
    celdas.push(`<td><span class="cronos-dia" data-fecha="${fecha}"${tipos.length ? ` data-tipos="${tipos.join(" ")}"` : ""} title="${escaparHTML(etiqueta)}"><span class="cronos-sr">${escaparHTML(etiqueta)}</span><span aria-hidden="true">${dia}</span></span></td>`);
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

function tabla(cabeceras, filas, t) {
  const cuerpo = filas.length ? filas.join("") : `<tr><td colspan="${cabeceras.length}">${escaparHTML(t("sin_filas"))}</td></tr>`;
  return `<div class="cronos-tabla-contenedor"><table class="cronos-tabla"><thead><tr>${cabeceras.map((c) => `<th scope="col">${escaparHTML(t(c))}</th>`).join("")}</tr></thead><tbody>${cuerpo}</tbody></table></div>`;
}

function formularioOlvido(f, t, hoy) {
  if (!f?.abierto) return `<button type="button" class="boton-primario" data-cronos-olvido="abrir">${escaparHTML(t("olvido_solicitar"))}</button>`;
  const enviando = f.estado === "enviando";
  const opciones = MOVIMIENTOS.map((m) => `<option value="${m}"${f.movimiento === m ? " selected" : ""}>${escaparHTML(t(`movimiento_${m}`))}</option>`).join("");
  const aviso = f.mensaje ? `<p class="cronos-solicitud-aviso" data-tono="${f.estado === "hecho" ? "exito" : "error"}" role="${f.estado === "hecho" ? "status" : "alert"}">${escaparHTML(f.mensaje)}</p>` : "";
  return `<form class="cronos-solicitud-formulario" data-cronos-olvido-formulario aria-label="${escaparHTML(t("olvido_formulario"))}">
    <label>${escaparHTML(t("fecha"))}<input type="date" name="fecha_civil" required max="${hoy}" value="${escaparHTML(f.fecha ?? "")}"${enviando ? " disabled" : ""}></label>
    <label>${escaparHTML(t("hora"))}<input type="time" name="hora_pretendida" required value="${escaparHTML(f.hora ?? "")}"${enviando ? " disabled" : ""}></label>
    <label>${escaparHTML(t("olvido_movimiento"))}<select name="movimiento"${enviando ? " disabled" : ""}>${opciones}</select></label>
    <div class="cronos-solicitud-acciones"><button type="submit" class="boton-primario"${enviando ? " disabled" : ""}>${escaparHTML(t(enviando ? "olvido_enviando" : "olvido_enviar"))}</button>
    <button type="button" class="boton-secundario" data-cronos-olvido="cerrar">${escaparHTML(t("cancelar"))}</button></div>${aviso}</form>`;
}

/** Calendario anual con los días marcados por tipo, ausencias y olvidos comunicados. */
export function renderizarMovimientosPropiosCronos({ estado = "cargando", anio, datos = null, formulario = null, mensajes = MENSAJES_CRONOS_SOLICITUDES_ES,
  locale = "es-ES", zonaHoraria = "Europe/Madrid", hoy = hoyCivilCronos(zonaHoraria) } = {}) {
  const t = crearTraductorSolicitudesCronos(mensajes);
  if (!Number.isInteger(anio) || anio < 2000 || anio > 2100) throw new RangeError("año de Cronos no válido");
  const ayuda = t("abrir_ayuda", { asunto: t("calendario_titulo") });
  const cabecera = `<header class="cronos-encabezado"><div><p class="sobrelinea">${escaparHTML(t("sobrelinea"))}</p><h2 id="cronos-movpropios-titulo">${escaparHTML(t("calendario_titulo"))}</h2></div>
    <button type="button" class="cronos-boton-ayuda" data-accion="ayuda" aria-label="${escaparHTML(ayuda)}" title="${escaparHTML(ayuda)}"><span aria-hidden="true">?</span></button></header>`;
  const navegacion = `<nav class="cronos-anio-navegacion" aria-label="${escaparHTML(t("calendario_anual", { anio }))}"><button type="button" class="boton-secundario" data-cronos-anio="-1" aria-label="${escaparHTML(t("anio_anterior"))}"${anio <= 2000 ? " disabled" : ""}>‹</button><strong aria-live="polite">${anio}</strong><button type="button" class="boton-secundario" data-cronos-anio="1" aria-label="${escaparHTML(t("anio_siguiente"))}"${anio >= 2100 ? " disabled" : ""}>›</button></nav>`;
  let cuerpo;
  if (estado !== "listo") {
    const clave = { denegado: "denegado", sin_empleado: "sin_empleado", error: "error" }[estado] ?? "cargando";
    cuerpo = `<section class="panel cronos-panel"><div class="cuerpo-panel"><p class="cronos-${estado === "cargando" ? "vacio" : "acceso-denegado"}" role="${estado === "error" ? "alert" : "status"}">${escaparHTML(t(clave))}</p></div></section>`;
  } else {
    const marcas = marcasPorDiaCronos(datos);
    const aviso = datos.calendario.disponible ? "" : `<p class="cronos-calendario-falta" role="status">${escaparHTML(t("calendario_sin_publicar", { anio }))}</p>`;
    const leyenda = `<ul class="cronos-leyenda" aria-label="${escaparHTML(t("leyenda"))}">${ORDEN_TIPOS.map((tipo) => `<li><span class="cronos-leyenda-muestra" data-tipos="${tipo}" aria-hidden="true"></span>${escaparHTML(nombreTipo(tipo, t))}</li>`).join("")}</ul>`;
    const meses = Array.from({ length: 12 }, (_, i) => mesHTML(anio, i + 1, marcas, t, locale)).join("");
    const ausencias = datos.absentismos.map((a) => `<tr><th scope="row">${escaparHTML(a.nombre)}</th><td>${escaparHTML(fechaVisible(a.desde, locale))}</td><td>${escaparHTML(fechaVisible(a.hasta, locale))}</td><td class="numero">${escaparHTML(formatearCantidadCronos(a.cantidad, a.unidad, t, locale))}</td><td>${a.pendiente_justificar ? `<span class="cronos-estado cronos-estado-aviso">${escaparHTML(t("justificante_pendiente"))}</span>` : escaparHTML(t("justificante_ok"))}</td></tr>`);
    const olvidos = datos.correcciones.map((c) => `<tr><th scope="row">${escaparHTML(fechaVisible(c.fecha_civil, locale))}</th><td>${escaparHTML(c.hora_pretendida)}</td><td>${escaparHTML(t(`movimiento_${c.movimiento}`))}</td><td><span class="cronos-estado" data-estado="${escaparHTML(c.estado)}">${escaparHTML(t(`correccion_${c.estado}`))}</span></td></tr>`);
    cuerpo = `<section class="panel cronos-panel" aria-labelledby="cronos-movpropios-cal"><div class="cabecera-panel"><h3 id="cronos-movpropios-cal">${escaparHTML(t("calendario_anual", { anio }))}</h3>${navegacion}</div><div class="cuerpo-panel">${aviso}${leyenda}<div class="cronos-meses">${meses}</div></div></section>
    <section class="panel cronos-panel" aria-labelledby="cronos-movpropios-aus"><div class="cabecera-panel"><h3 id="cronos-movpropios-aus">${escaparHTML(t("absentismos_titulo"))}</h3></div>${tabla(["permiso", "desde", "hasta", "duracion", "estado"], ausencias, t)}</section>
    <section class="panel cronos-panel" aria-labelledby="cronos-movpropios-olv"><div class="cabecera-panel"><h3 id="cronos-movpropios-olv">${escaparHTML(t("olvidos_titulo"))}</h3></div>${tabla(["fecha", "hora", "olvido_movimiento", "estado"], olvidos, t)}<div class="cuerpo-panel" data-cronos-olvido-zona>${formularioOlvido(formulario, t, hoy)}</div></section>`;
  }
  return `<section class="cronos-area cronos-movimientos-propios" aria-labelledby="cronos-movpropios-titulo" data-estado="${escaparHTML(estado)}">${cabecera}${estado === "listo" ? "" : `<section class="panel cronos-panel"><div class="cabecera-panel">${navegacion}</div></section>`}${cuerpo}</section>`;
}

function estadoError(error) {
  if (error instanceof ErrorClienteSolicitudesCronos) {
    if (error.codigo === "sin_empleado") return "sin_empleado";
    if (["acceso_denegado", "autenticacion_requerida"].includes(error.codigo)) return "denegado";
  }
  return "error";
}

/** Consulta el año con el cliente propio y registra olvidos; aborta al cambiar o desmontar. */
export function montarMovimientosPropiosCronos({ raiz, cliente = crearClienteSolicitudesCronosHTTP(), mensajes = MENSAJES_CRONOS_SOLICITUDES_ES,
  anunciar = () => {}, registrarDesmontar, locale = "es-ES", zonaHoraria = "Europe/Madrid", anio, abrirOlvido = false } = {}) {
  if (!raiz?.append || !raiz.ownerDocument?.createElement || typeof cliente?.consultarMovimientos !== "function" || typeof cliente?.solicitarCorreccion !== "function"
    || typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) throw new TypeError("montaje de movimientos propios Cronos no disponible");
  const t = crearTraductorSolicitudesCronos(mensajes);
  const contenedor = raiz.ownerDocument.createElement("section"); contenedor.dataset.cronosMovimientosPropios = ""; raiz.append(contenedor);
  const hoy = hoyCivilCronos(zonaHoraria);
  let activa = true; let secuencia = 0; let controlador = null; let envio = null;
  let anioVisible = Number.isInteger(anio) ? anio : Number(hoy.slice(0, 4));
  let estado = "cargando"; let datos = null;
  let formulario = abrirOlvido ? { abierto: true, clave: claveNueva(), movimiento: "entrada", fecha: hoy } : null;
  const dibujar = () => { if (activa) contenedor.innerHTML = renderizarMovimientosPropiosCronos({ estado, anio: anioVisible, datos, formulario, mensajes, locale, zonaHoraria, hoy }); };
  const cargar = async () => {
    controlador?.abort(); controlador = new AbortController(); const turno = ++secuencia;
    estado = "cargando"; datos = null; dibujar();
    const consulta = String(anioVisible) === hoy.slice(0, 4) ? { periodo: "anio" } : { periodo: "rango", desde: `${anioVisible}-01-01`, hasta: `${anioVisible}-12-31` };
    try {
      const r = await cliente.consultarMovimientos(consulta, { signal: controlador.signal });
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
      if (siguiente >= 2000 && siguiente <= 2100) { anioVisible = siguiente; void cargar(); }
      return;
    }
    const olvido = evento.target?.closest?.("[data-cronos-olvido]");
    if (!olvido) return;
    if (olvido.dataset.cronosOlvido === "abrir") formulario = { abierto: true, clave: claveNueva(), movimiento: "entrada", fecha: hoy };
    else { envio?.abort(); formulario = null; }
    dibujar();
    if (formulario) contenedor.querySelector?.("[name=fecha_civil]")?.focus?.();
  };
  const alEnviar = async (evento) => {
    if (!evento.target?.matches?.("[data-cronos-olvido-formulario]") || !formulario || formulario.estado === "enviando") return;
    evento.preventDefault();
    const valor = (n) => evento.target.elements?.namedItem?.(n)?.value ?? "";
    const entrada = { fecha: valor("fecha_civil"), hora: valor("hora_pretendida"), movimiento: valor("movimiento") };
    // Otra declaración distinta necesita otra clave; un reintento de la misma la conserva.
    if (formulario.estado === "hecho" || (formulario.enviada && (formulario.enviada.fecha !== entrada.fecha || formulario.enviada.hora !== entrada.hora || formulario.enviada.movimiento !== entrada.movimiento))) formulario.clave = claveNueva();
    formulario = { ...formulario, ...entrada, estado: "enviando", mensaje: "", enviada: entrada };
    dibujar();
    envio = new AbortController();
    try {
      const recibo = await cliente.solicitarCorreccion({ clave_operacion: formulario.clave, movimiento: entrada.movimiento, fecha_civil: entrada.fecha, hora_pretendida: entrada.hora }, { signal: envio.signal });
      if (!activa) return;
      formulario = { ...formulario, estado: "hecho", mensaje: t(recibo.replay ? "olvido_ya_registrado" : "olvido_registrado", { fecha: instanteVisible(recibo.instante_utc, locale, zonaHoraria) }), reciboRef: recibo.recibo_ref };
      anunciar(formulario.mensaje);
      await cargar();
    } catch (error) {
      if (!activa || envio.signal.aborted) return;
      const codigo = error instanceof ErrorClienteSolicitudesCronos || error instanceof TypeError ? (error.codigo ?? "peticion_invalida") : "";
      const clave = codigo === "peticion_invalida" ? "olvido_invalido" : codigo === "conflicto" ? "olvido_conflicto" : "olvido_error";
      formulario = { ...formulario, estado: "error", mensaje: t(clave) };
      dibujar(); anunciar(formulario.mensaje);
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
