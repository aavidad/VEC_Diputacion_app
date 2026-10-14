import { crearClienteCorreccionesCronosHTTP, ErrorClienteCorreccionesCronos, motivoCorreccionValido, PASOS_BANDEJA_CORRECCIONES, validarBandejaCorreccionesCronos, validarHistoriaCorreccionCronos } from "./cliente-correcciones-http.js";
import { crearClienteSolicitudesCronosHTTP, ErrorClienteSolicitudesCronos, validarMovimientosPropiosCronos } from "./cliente-solicitudes-http.js";
import { crearTraductorCorreccionesC5B } from "./i18n-correcciones-c5b.js";
import { LOCALIZACION_ACTUAL } from "../../../comun/idioma.js";

const escapar = (v) => String(v ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
const fechaVisible = (v, locale) => {
  const [a, m, d] = v.split("-").map(Number);
  return new Intl.DateTimeFormat(locale, { timeZone: "UTC", dateStyle: "medium" }).format(new Date(Date.UTC(a, m - 1, d, 12)));
};
const instanteVisible = (v, locale, zona) => new Intl.DateTimeFormat(locale, { timeZone: zona, dateStyle: "medium", timeStyle: "short" }).format(new Date(v));
const claveNueva = () => globalThis.crypto.randomUUID();
const mensajeError = (e, t) => {
  if (!(e instanceof ErrorClienteCorreccionesCronos)) return t("error");
  if (["no_competente", "acceso_denegado", "autenticacion_requerida", "sin_empleado"].includes(e.codigo)) return t("denegado");
  if (["conflicto", "estado_cambiado"].includes(e.codigo)) return t("error_conflicto");
  if (e.codigo === "peticion_invalida") return t("error_peticion");
  return t("error");
};
const estadoConsulta = (e) => (e instanceof ErrorClienteCorreccionesCronos || e instanceof ErrorClienteSolicitudesCronos)
  && ["no_competente", "acceso_denegado", "autenticacion_requerida", "sin_empleado"].includes(e.codigo) ? "denegado" : "error";

function historiaHTML(h, t, locale, zona) {
  const titulo = t("historia_solicitud", { fecha: fechaVisible(h.fecha_civil, locale), movimiento: t(`movimiento_${h.movimiento}`), hora: h.hora_pretendida });
  const actos = h.actuaciones.map((a) => `<li><strong>${escapar(t(`paso_${a.paso}`))}</strong> · ${escapar(instanteVisible(a.instante_utc, locale, zona))} · ${escapar(t(`estado_${a.estado}`))}
    ${a.motivo ? `<p>${escapar(t("historia_motivo", { motivo: a.motivo }))}</p>` : ""}<details><summary>${escapar(t("recibo_detalle"))}</summary><p>${escapar(t("referencia_recibo"))}: ${escapar(a.recibo_ref)}</p></details></li>`).join("");
  const asientos = h.asientos.map((a) => `<li>${escapar(t(`asiento_${a.tipo}`))}${a.instante_utc ? ` · ${escapar(instanteVisible(a.instante_utc, locale, zona))}` : ""}
    <details><summary>${escapar(t("historia_fuente"))}</summary><p>${escapar(a.fuente_version_ref)}</p></details></li>`).join("");
  return `<section class="panel cronos-panel" data-cronos-c5b-historia aria-label="${escapar(t("historia_actuaciones"))}"><div class="cabecera-panel"><h3>${escapar(titulo)}</h3>
    <button type="button" class="boton-secundario" data-c5b-volver>${escapar(t("volver"))}</button></div><div class="cuerpo-panel"><p role="status">${escapar(t("historia_actual", { estado: t(`estado_${h.estado}`) }))}</p>
    <h4>${escapar(t("historia_actuaciones"))}</h4><ol>${actos}</ol><h4>${escapar(t("historia_asientos"))}</h4>${asientos ? `<ol>${asientos}</ol>` : `<p>${escapar(t("sin_asientos"))}</p>`}</div></section>`;
}

function reciboHTML(r, t, locale, zona) {
  return `<div class="cronos-solicitud-recibo" role="status" aria-live="polite"><p>${escapar(t("recibo"))} · ${escapar(instanteVisible(r.instante_utc, locale, zona))}</p>
    <details><summary>${escapar(t("recibo_detalle"))}</summary><p>${escapar(t("referencia_recibo"))}: ${escapar(r.recibo_ref)}</p></details></div>`;
}

function formularioHTML(s, accion, t, locale, zona) {
  if (!s || !accion) return "";
  const ocupado = ["revisar", "enviando", "incierto", "recuperando", "hecho"].includes(accion.estado);
  const aplicar = s.estado === "pendiente_aplicacion";
  const aviso = accion.mensaje ? `<p class="cronos-solicitud-aviso" data-c5b-resultado tabindex="-1" role="${accion.estado === "hecho" ? "status" : "alert"}" aria-live="polite" data-tono="${accion.estado === "hecho" ? "exito" : "error"}">${escapar(accion.mensaje)}</p>` : "";
  const resumen = ["revisar", "enviando", "incierto", "recuperando", "hecho"].includes(accion.estado) ? `<section class="panel cronos-panel"><div class="cabecera-panel"><h4>${escapar(t("resumen"))}</h4></div><div class="cuerpo-panel"><p>${escapar(t("resumen_resultado", { resultado: t(aplicar ? "aplicar" : accion.resultado) }))}</p>
    <p>${escapar(t("resumen_motivo", { motivo: accion.motivo }))}</p></div></section>` : "";
  return `<section class="panel cronos-panel" data-c5b-formulario aria-labelledby="c5b-form-titulo"><div class="cabecera-panel"><h3 id="c5b-form-titulo">${escapar(t("decidir"))}: ${escapar(s.empleado_etiqueta || t("persona_sin_nombre"))}</h3></div><div class="cuerpo-panel">
    <p>${escapar(fechaVisible(s.fecha_civil, locale))} · ${escapar(s.hora_pretendida)} · ${escapar(t(`movimiento_${s.movimiento}`))}</p>
    <p>${escapar(t(s.marcaje_original_ref ? "con_marcaje_original" : "sin_marcaje_original"))}</p>
    <form data-c5b-form><fieldset${ocupado || aplicar ? " disabled" : ""}><legend>${escapar(t("decision"))}</legend>
      ${aplicar ? `<p>${escapar(t("aplicar"))}</p>` : `<label><input type="radio" name="resultado" value="favorable"${accion.resultado === "favorable" ? " checked" : ""}>${escapar(t("favorable"))}</label>
      <label><input type="radio" name="resultado" value="desfavorable"${accion.resultado === "desfavorable" ? " checked" : ""}>${escapar(t("desfavorable"))}</label>`}</fieldset>
      <label>${escapar(t("motivo"))}<textarea name="motivo" rows="3" maxlength="500" required${ocupado ? " disabled" : ""}>${escapar(accion.motivo ?? "")}</textarea></label>
      ${resumen}${aviso}${accion.recibo ? reciboHTML(accion.recibo, t, locale, zona) : ""}
      <div class="cronos-solicitud-acciones">${accion.estado === "editar" ? `<button type="submit" class="boton-primario">${escapar(t("continuar"))}</button>` : ""}
      ${accion.estado === "revisar" ? `<button type="button" class="boton-primario" data-c5b-confirmar>${escapar(t("confirmar"))}</button>` : ""}
      ${accion.estado === "revisar" ? `<button type="button" class="boton-secundario" data-c5b-editar>${escapar(t("editar"))}</button>` : ""}
      ${accion.estado === "enviando" ? `<button type="button" class="boton-primario" disabled>${escapar(t("enviando"))}</button>` : ""}
      ${accion.estado === "incierto" ? `<button type="button" class="boton-primario" data-c5b-recuperar>${escapar(t("recuperar"))}</button>
        <button type="button" class="boton-secundario" data-c5b-reintentar>${escapar(t("reintentar_igual"))}</button>` : ""}
      <button type="button" class="boton-secundario" data-c5b-cerrar${["enviando", "incierto", "recuperando"].includes(accion.estado) ? " disabled" : ""}>${escapar(t("cancelar"))}</button></div></form></div></section>`;
}

export function renderizarBandejaCorreccionesCronos({ estado = "cargando", paso = "responsable", datos = null, elegida = null, accion = null, historia = null, mensaje = "", mensajes,
  locale = LOCALIZACION_ACTUAL, zonaHoraria = "Europe/Madrid" } = {}) {
  const t = crearTraductorCorreccionesC5B(mensajes);
  if (!PASOS_BANDEJA_CORRECCIONES.includes(paso)) throw new TypeError("paso de bandeja no válido");
  const cabecera = `<header class="cronos-encabezado"><div><p class="sobrelinea">${escapar(t("sobrelinea"))}</p><h2>${escapar(t("bandeja_titulo"))}</h2></div>
    <button type="button" class="boton-secundario" data-c5b-actualizar${accion && ["enviando", "incierto", "recuperando"].includes(accion.estado) ? " disabled" : ""}>${escapar(t("actualizar"))}</button></header>`;
  const pasos = `<nav class="cronos-solicitud-acciones" aria-label="${escapar(t("bandeja_titulo"))}">${PASOS_BANDEJA_CORRECCIONES.map((p) => `<button type="button" class="boton-secundario" data-c5b-paso="${p}"${paso === p ? ' aria-current="page"' : ""}${accion && ["enviando", "incierto", "recuperando"].includes(accion.estado) ? " disabled" : ""}>${escapar(t(`paso_${p}`))}</button>`).join("")}</nav>`;
  if (estado !== "listo") return `<section class="cronos-area" data-c5b-estado="${estado}">${cabecera}${pasos}<section class="panel cronos-panel"><div class="cuerpo-panel"><p role="${estado === "error" ? "alert" : "status"}" aria-live="polite">${escapar(t(estado === "denegado" ? "denegado" : estado === "error" ? "error" : "cargando"))}</p></div></section></section>`;
  validarBandejaCorreccionesCronos(datos, paso);
  const filas = datos.pendientes.map((s) => `<tr><th scope="row">${escapar(s.empleado_etiqueta || t("persona_sin_nombre"))}</th><td>${escapar(fechaVisible(s.fecha_civil, locale))} · ${escapar(s.hora_pretendida)}</td>
    <td>${escapar(t(`movimiento_${s.movimiento}`))}</td><td><span class="cronos-estado">${escapar(t(`estado_${s.estado}`))}</span></td>
    <td><button type="button" class="boton-secundario" data-c5b-revisar="${escapar(s.solicitud_ref)}"${accion && ["enviando", "incierto", "recuperando"].includes(accion.estado) ? " disabled" : ""}>${escapar(t("revisar"))}</button>
    <button type="button" class="boton-secundario" data-c5b-historia="${escapar(s.solicitud_ref)}">${escapar(t("ver_historia"))}</button></td></tr>`).join("");
  const cabeceras = ["col_persona", "col_fecha", "col_marcaje", "col_estado", "col_accion"];
  const tabla = `<div class="cronos-tabla-contenedor"><table class="cronos-tabla"><thead><tr>${cabeceras.map((c) => `<th scope="col">${escapar(t(c))}</th>`).join("")}</tr></thead><tbody>${filas || `<tr><td colspan="5">${escapar(t("vacio"))}</td></tr>`}</tbody></table></div>`;
  return `<section class="cronos-area" data-c5b-estado="listo">${cabecera}${pasos}<section class="panel cronos-panel">${tabla}</section>
    ${mensaje ? `<p class="cronos-solicitud-aviso" role="status" aria-live="polite">${escapar(mensaje)}</p>` : ""}
    ${historia ? historiaHTML(historia, t, locale, zonaHoraria) : ""}${formularioHTML(elegida, accion, t, locale, zonaHoraria)}</section>`;
}

export function montarBandejaCorreccionesCronos({ raiz, cliente = crearClienteCorreccionesCronosHTTP(), anunciar = () => {}, registrarDesmontar, mensajes,
  locale = LOCALIZACION_ACTUAL, zonaHoraria = "Europe/Madrid" } = {}) {
  if (!raiz?.append || !raiz.ownerDocument?.createElement || typeof cliente?.consultarBandeja !== "function" || typeof cliente?.registrarActuacion !== "function"
    || typeof cliente?.consultarHistoria !== "function" || typeof anunciar !== "function") throw new TypeError("bandeja de correcciones no disponible");
  const t = crearTraductorCorreccionesC5B(mensajes);
  const nodo = raiz.ownerDocument.createElement("section"); nodo.dataset.cronosCorreccionesBandeja = ""; raiz.append(nodo);
  let activa = true; let seq = 0; let consulta = null; let envio = null; let paso = "responsable"; let estado = "cargando"; let datos = null;
  let elegida = null; let accion = null; let historia = null; let mensaje = "";
  const dibujar = (foco) => {
    if (!activa) return;
    nodo.innerHTML = renderizarBandejaCorreccionesCronos({ estado, paso, datos, elegida, accion, historia, mensaje, mensajes, locale, zonaHoraria });
    if (foco) nodo.querySelector?.(foco)?.focus?.();
  };
  const cargar = async () => {
    if (!activa || accion && ["enviando", "incierto", "recuperando"].includes(accion.estado)) return;
    consulta?.abort(); const controlador = new AbortController(); consulta = controlador; const turno = ++seq;
    estado = "cargando"; datos = null; elegida = null; accion = null; historia = null; dibujar();
    try {
      const valor = validarBandejaCorreccionesCronos(await cliente.consultarBandeja(paso, { signal: controlador.signal }), paso);
      if (!activa || controlador.signal.aborted || turno !== seq) return;
      datos = valor; estado = "listo"; dibujar();
    } catch (e) { if (!activa || controlador.signal.aborted || turno !== seq) return; estado = estadoConsulta(e); dibujar(); anunciar(mensajeError(e, t)); }
  };
  const abrirHistoria = async (ref, { recuperar = false } = {}) => {
    if (!datos?.pendientes.some((s) => s.solicitud_ref === ref)) return;
    consulta?.abort(); const controlador = new AbortController(); consulta = controlador; const turno = ++seq;
    try {
      const h = validarHistoriaCorreccionCronos(await cliente.consultarHistoria(ref, { vista: "equipo", paso, signal: controlador.signal }), ref, "equipo");
      if (!activa || controlador.signal.aborted || turno !== seq) return;
      historia = h;
      if (recuperar && accion?.estado === "recuperando") {
        const a = h.actuaciones.find((x) => x.version === Number(accion.entrada.version_esperada) + 1);
        const coincide = a && a.resultado === accion.entrada.resultado && a.motivo === accion.entrada.motivo;
        accion = { ...accion, estado: "incierto", mensaje: t(a ? coincide ? "recuperacion_coincide" : "recuperacion_otro" : "recuperacion_ausente") };
      }
      dibujar(recuperar ? '[data-c5b-resultado]' : '[data-c5b-volver]');
    } catch (e) {
      if (!activa || controlador.signal.aborted || turno !== seq) return;
      const texto = e instanceof ErrorClienteCorreccionesCronos && ["acceso_denegado", "no_competente", "sin_empleado", "autenticacion_requerida"].includes(e.codigo) ? t("recuperacion_denegada") : t("recuperacion_error");
      if (recuperar && accion) accion = { ...accion, estado: "incierto", mensaje: texto };
      else mensaje = texto;
      dibujar(); anunciar(texto);
    }
  };
  const enviar = async () => {
    if (!accion || !["revisar", "incierto"].includes(accion.estado) || envio) return;
    const entrada = accion.entrada; const controlador = new AbortController(); envio = controlador;
    accion = { ...accion, estado: "enviando", mensaje: "" }; dibujar();
    try {
      const recibo = await cliente.registrarActuacion(entrada, { signal: controlador.signal });
      if (!activa || controlador.signal.aborted || envio !== controlador) return;
      accion = { ...accion, estado: "hecho", recibo, mensaje: t(recibo.replay ? "ya_registrada" : "registrada", { fecha: instanteVisible(recibo.instante_utc, locale, zonaHoraria) }) };
      dibujar(); anunciar(accion.mensaje);
    } catch (e) {
      if (!activa || controlador.signal.aborted || envio !== controlador) return;
      const definitivo = e instanceof ErrorClienteCorreccionesCronos && ["no_competente", "acceso_denegado", "autenticacion_requerida", "sin_empleado", "peticion_invalida", "conflicto", "estado_cambiado"].includes(e.codigo);
      accion = { ...accion, estado: definitivo ? "editar" : "incierto", mensaje: definitivo ? mensajeError(e, t) : t("resultado_incierto") };
      dibujar(); anunciar(accion.mensaje);
    } finally { if (envio === controlador) envio = null; }
  };
  const alClick = (evento) => {
    if (!activa) return;
    const boton = evento.target?.closest?.("button"); if (!boton || boton.disabled) return;
    if (boton.hasAttribute("data-c5b-actualizar")) { void cargar(); return; }
    if (boton.dataset.c5bPaso && PASOS_BANDEJA_CORRECCIONES.includes(boton.dataset.c5bPaso)) {
      if (accion && ["enviando", "incierto", "recuperando"].includes(accion.estado)) return;
      paso = boton.dataset.c5bPaso; void cargar(); return;
    }
    const ref = boton.dataset.c5bRevisar;
    if (ref && datos?.pendientes.some((s) => s.solicitud_ref === ref) && !["incierto", "enviando", "recuperando"].includes(accion?.estado)) {
      elegida = datos.pendientes.find((s) => s.solicitud_ref === ref);
      const pasoActo = elegida.estado === "pendiente_aplicacion" ? "aplicar" : paso === "responsable" ? "responsable" : "rrhh";
      accion = { estado: "editar", clave: claveNueva(), resultado: pasoActo === "aplicar" ? "" : "favorable", motivo: "", paso: pasoActo, mensaje: "" };
      historia = null; dibujar('[name="motivo"]'); return;
    }
    if (boton.dataset.c5bHistoria && datos?.pendientes.some((s) => s.solicitud_ref === boton.dataset.c5bHistoria)) { void abrirHistoria(boton.dataset.c5bHistoria); return; }
    if (boton.hasAttribute("data-c5b-volver")) { historia = null; dibujar(); return; }
    if (boton.hasAttribute("data-c5b-cerrar") && !["incierto", "enviando", "recuperando"].includes(accion?.estado)) { accion = null; elegida = null; dibujar(); return; }
    if (boton.hasAttribute("data-c5b-editar") && accion?.estado === "revisar") { accion = { ...accion, estado: "editar" }; dibujar('[name="motivo"]'); return; }
    if (boton.hasAttribute("data-c5b-confirmar")) { void enviar(); return; }
    if (boton.hasAttribute("data-c5b-reintentar") && accion?.estado === "incierto") { void enviar(); return; }
    if (boton.hasAttribute("data-c5b-recuperar") && accion?.estado === "incierto") { accion = { ...accion, estado: "recuperando" }; dibujar(); void abrirHistoria(accion.entrada.solicitud_ref, { recuperar: true }); }
  };
  const alEnviar = (evento) => {
    if (!evento.target?.matches?.("[data-c5b-form]")) return;
    evento.preventDefault(); if (!accion || accion.estado !== "editar" || !elegida) return;
    const resultado = accion.paso === "aplicar" ? "" : evento.target.elements?.namedItem?.("resultado")?.value ?? "";
    const motivo = evento.target.elements?.namedItem?.("motivo")?.value ?? "";
    if (!motivoCorreccionValido(motivo) || (accion.paso !== "aplicar" && !["favorable", "desfavorable"].includes(resultado))) {
      accion = { ...accion, resultado, motivo, mensaje: t("error_motivo") }; dibujar('[name="motivo"]'); return;
    }
    accion = { ...accion, estado: "revisar", resultado, motivo, mensaje: "", entrada: {
      solicitud_ref: elegida.solicitud_ref, clave_operacion: accion.clave, paso: accion.paso, resultado, motivo, version_esperada: String(elegida.version),
    } }; dibujar('[data-c5b-confirmar]');
  };
  nodo.addEventListener("click", alClick); nodo.addEventListener("submit", alEnviar);
  void cargar();
  const desmontar = () => { if (!activa) return; activa = false; ++seq; consulta?.abort(); envio?.abort(); nodo.removeEventListener("click", alClick); nodo.removeEventListener("submit", alEnviar); nodo.remove?.(); };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ desmontar, actualizar: cargar });
}

export function renderizarHistoriaCorreccionesPropiasCronos({ estado = "cargando", anio, datos = null, historia = null, mensaje = "", mensajes,
  locale = LOCALIZACION_ACTUAL, zonaHoraria = "Europe/Madrid" } = {}) {
  const t = crearTraductorCorreccionesC5B(mensajes);
  const cabecera = `<header class="cronos-encabezado"><div><p class="sobrelinea">${escapar(t("sobrelinea"))}</p><h2>${escapar(t("historia_titulo"))}</h2></div></header>
    <form data-c5b-anio class="cronos-solicitud-formulario"><label>${escapar(t("anio"))}<input name="anio" type="number" min="2000" max="2100" value="${anio}"></label><button type="submit" class="boton-secundario">${escapar(t("consultar"))}</button></form>`;
  if (estado !== "listo") return `<section class="cronos-area" data-c5b-propia-estado="${estado}">${cabecera}<p role="${estado === "error" ? "alert" : "status"}" aria-live="polite">${escapar(t(estado === "denegado" ? "denegado" : estado === "error" ? "error" : "cargando"))}</p></section>`;
  const filas = datos.correcciones.map((c) => `<tr><th scope="row">${escapar(fechaVisible(c.fecha_civil, locale))} · ${escapar(c.hora_pretendida)}</th><td>${escapar(t(`movimiento_${c.movimiento}`))}</td>
    <td><span class="cronos-estado">${escapar(t(`estado_${c.estado}`))}</span></td><td><button type="button" class="boton-secundario" data-c5b-propia-ref="${escapar(c.solicitud_ref)}">${escapar(t("ver_historia"))}</button></td></tr>`).join("");
  return `<section class="cronos-area" data-c5b-propia-estado="listo">${cabecera}${mensaje ? `<p role="alert" aria-live="polite">${escapar(mensaje)}</p>` : ""}
    <section class="panel cronos-panel"><div class="cronos-tabla-contenedor"><table class="cronos-tabla"><thead><tr>${["col_fecha", "col_marcaje", "col_estado", "col_accion"].map((c) => `<th scope="col">${escapar(t(c))}</th>`).join("")}</tr></thead><tbody>${filas || `<tr><td colspan="4">${escapar(t("sin_historia"))}</td></tr>`}</tbody></table></div></section>
    ${historia ? historiaHTML(historia, t, locale, zonaHoraria) : ""}</section>`;
}

export function montarHistoriaCorreccionesPropiasCronos({ raiz, cliente = crearClienteCorreccionesCronosHTTP(), clienteSolicitudes = crearClienteSolicitudesCronosHTTP(),
  anunciar = () => {}, registrarDesmontar, mensajes, locale = LOCALIZACION_ACTUAL, zonaHoraria = "Europe/Madrid" } = {}) {
  if (!raiz?.append || !raiz.ownerDocument?.createElement || typeof cliente?.consultarHistoria !== "function" || typeof clienteSolicitudes?.consultarMovimientos !== "function") throw new TypeError("historia propia de correcciones no disponible");
  const t = crearTraductorCorreccionesC5B(mensajes);
  const nodo = raiz.ownerDocument.createElement("section"); nodo.dataset.cronosCorreccionesPropias = ""; raiz.append(nodo);
  let activa = true; let seq = 0; let controlador = null; let anio = Number(new Intl.DateTimeFormat("en", { timeZone: zonaHoraria, year: "numeric" }).format(new Date()));
  let estado = "cargando"; let datos = null; let historia = null; let mensaje = "";
  const dibujar = () => { if (activa) nodo.innerHTML = renderizarHistoriaCorreccionesPropiasCronos({ estado, anio, datos, historia, mensaje, mensajes, locale, zonaHoraria }); };
  const cargar = async () => {
    controlador?.abort(); const cancelacion = new AbortController(); controlador = cancelacion; const turno = ++seq;
    estado = "cargando"; historia = null; datos = null; dibujar();
    const actual = Number(new Intl.DateTimeFormat("en", { timeZone: zonaHoraria, year: "numeric" }).format(new Date()));
    const consulta = anio === actual ? { periodo: "anio" } : { periodo: "rango", desde: `${anio}-01-01`, hasta: `${anio}-12-31` };
    try {
      const v = validarMovimientosPropiosCronos(await clienteSolicitudes.consultarMovimientos(consulta, { signal: cancelacion.signal }), consulta);
      if (!activa || cancelacion.signal.aborted || turno !== seq) return;
      datos = v; estado = "listo"; dibujar();
    } catch (e) { if (!activa || cancelacion.signal.aborted || turno !== seq) return; estado = estadoConsulta(e); dibujar(); anunciar(t(estado)); }
  };
  const alEnviar = (evento) => { if (!evento.target?.matches?.("[data-c5b-anio]")) return; evento.preventDefault(); const nuevo = Number(evento.target.elements?.namedItem?.("anio")?.value);
    if (Number.isSafeInteger(nuevo) && nuevo >= 2000 && nuevo <= 2100) { anio = nuevo; void cargar(); } };
  const alClick = async (evento) => {
    const boton = evento.target?.closest?.("button"); if (!boton) return;
    if (boton.hasAttribute("data-c5b-volver")) { historia = null; dibujar(); return; }
    const ref = boton.dataset.c5bPropiaRef;
    if (!datos?.correcciones.some((c) => c.solicitud_ref === ref)) return;
    controlador?.abort(); const cancelacion = new AbortController(); controlador = cancelacion; const turno = ++seq;
    try {
      const h = validarHistoriaCorreccionCronos(await cliente.consultarHistoria(ref, { vista: "propia", signal: cancelacion.signal }), ref, "propia");
      if (!activa || cancelacion.signal.aborted || turno !== seq) return;
      historia = h; mensaje = ""; dibujar(); nodo.querySelector?.("[data-c5b-volver]")?.focus?.();
    } catch (e) { if (!activa || cancelacion.signal.aborted || turno !== seq) return; mensaje = mensajeError(e, t); dibujar(); anunciar(mensaje); }
  };
  nodo.addEventListener("submit", alEnviar); nodo.addEventListener("click", alClick); void cargar();
  const desmontar = () => { if (!activa) return; activa = false; ++seq; controlador?.abort(); nodo.removeEventListener("submit", alEnviar); nodo.removeEventListener("click", alClick); nodo.remove?.(); };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ desmontar, actualizar: cargar });
}
