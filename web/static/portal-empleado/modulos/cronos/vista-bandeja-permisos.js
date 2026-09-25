import { crearTraductorResolucionCronos, periodoSolicitudCronos } from "./i18n-resolucion.js";
import { formatearCantidadCronos } from "./i18n-solicitudes.js";
import {
  ErrorClienteResolucionCronos, MAXIMO_MOTIVO_RESOLUCION_CRONOS, PASOS_RESOLUCION_CRONOS, crearClienteResolucionCronosHTTP, motivoResolucionValido,
} from "./cliente-resolucion-http.js";

const ERRORES = new Map([
  ["peticion_invalida", "error_motivo"], ["no_competente", "error_no_competente"], ["estado_cambiado", "error_estado_cambiado"],
  ["conflicto", "error_conflicto_resolucion"],
]);

function escaparHTML(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}
function instanteVisible(valor, locale, zonaHoraria) {
  return new Intl.DateTimeFormat(locale, { timeZone: zonaHoraria, dateStyle: "medium" }).format(new Date(valor));
}
function claveNueva() { return globalThis.crypto.randomUUID(); }

function persona(s, t) { return s.empleado_etiqueta || t("persona_sin_nombre"); }

function formulario(r, s, paso, t, locale) {
  if (!r || !s) return "";
  const enviando = r.estado === "enviando";
  const inactivo = enviando ? " disabled" : "";
  const titulo = t("resolver_solicitud", { permiso: s.nombre, persona: persona(s, t) });
  const opcion = (valor, etiqueta) => `<label class="cronos-resolucion-opcion"><input type="radio" name="decision" value="${valor}"${r.decision === valor ? " checked" : ""}${inactivo} required> ${escaparHTML(t(etiqueta))}</label>`;
  const aviso = r.mensaje ? `<p class="cronos-solicitud-aviso" data-tono="error" role="alert">${escaparHTML(r.mensaje)}</p>` : "";
  return `<section class="panel cronos-panel" aria-labelledby="cronos-resolucion-titulo"><div class="cabecera-panel"><h3 id="cronos-resolucion-titulo">${escaparHTML(titulo)}</h3>
    <span class="cronos-circuito">${escaparHTML(periodoSolicitudCronos(s, t, locale))} · ${escaparHTML(formatearCantidadCronos(s.cantidad, s.unidad, t, locale))}</span></div>
    <div class="cuerpo-panel"><form class="cronos-resolucion-formulario" data-cronos-resolucion-formulario aria-label="${escaparHTML(titulo)}">
    <fieldset class="cronos-resolucion-decision"><legend>${escaparHTML(t("decision"))}</legend>${opcion("aprobar", paso === "responsable" ? "decision_aprobar_responsable" : "decision_aprobar_administracion")}${opcion("denegar", "decision_denegar")}</fieldset>
    <label class="cronos-resolucion-motivo">${escaparHTML(t("motivo"))}<textarea name="motivo" rows="3" maxlength="${MAXIMO_MOTIVO_RESOLUCION_CRONOS}"${r.decision === "denegar" ? " required" : ""}${inactivo}>${escaparHTML(r.motivo ?? "")}</textarea></label>
    <div class="cronos-solicitud-acciones"><button type="submit" class="boton-primario"${inactivo}>${escaparHTML(t(enviando ? "resolucion_enviando" : "resolucion_enviar"))}</button>
    <button type="button" class="boton-secundario" data-cronos-resolucion-cerrar>${escaparHTML(t("cancelar"))}</button></div>${aviso}</form></div></section>`;
}

function estadoFila(s) {
  return s.estado === "pendiente_administracion" || s.circuito === "A" ? "estado_pendiente_administracion" : "estado_pendiente_jefatura";
}

/** Bandeja de un paso: solicitudes pendientes de quien resuelve y el formulario de resolución. */
export function renderizarBandejaPermisosCronos({ estado = "cargando", paso = "responsable", datos = null, resolucion = null, mensaje = "", tonoMensaje = "exito",
  mensajes, locale = "es-ES", zonaHoraria = "Europe/Madrid" } = {}) {
  const t = crearTraductorResolucionCronos(mensajes);
  if (!PASOS_RESOLUCION_CRONOS.includes(paso)) throw new RangeError("paso de resolución no válido");
  const ayuda = t("abrir_ayuda", { asunto: t("bandeja_titulo") });
  const cabecera = `<header class="cronos-encabezado"><div><p class="sobrelinea">${escaparHTML(t("sobrelinea"))}</p><h2 id="cronos-bandeja-titulo">${escaparHTML(t("bandeja_titulo"))}</h2></div>
    <button type="button" class="cronos-boton-ayuda" data-accion="ayuda" aria-label="${escaparHTML(ayuda)}" title="${escaparHTML(ayuda)}"><span aria-hidden="true">?</span></button></header>`;
  const selector = `<div class="cronos-selector-paso" role="group" aria-label="${escaparHTML(t("paso_selector"))}">${PASOS_RESOLUCION_CRONOS.map((p) =>
    `<button type="button" class="boton-secundario" data-cronos-paso="${p}" aria-pressed="${p === paso}">${escaparHTML(t(`paso_${p}`))}</button>`).join("")}</div>`;
  const cabeceraPanel = `<div class="cabecera-panel"><h3 id="cronos-bandeja-paso">${escaparHTML(t(`paso_${paso}`))}</h3>${selector}</div>`;
  const tono = tonoMensaje === "error" ? "error" : "exito";
  const aviso = mensaje ? `<p class="cronos-solicitud-aviso" data-tono="${tono}" role="${tono === "error" ? "alert" : "status"}">${escaparHTML(mensaje)}</p>` : "";
  if (estado !== "listo") {
    const clave = { denegado: "denegado", sin_empleado: "sin_empleado", error: "error" }[estado] ?? "cargando";
    return `<section class="cronos-area cronos-bandeja-permisos" aria-labelledby="cronos-bandeja-titulo" data-estado="${escaparHTML(estado)}">${cabecera}
      <section class="panel cronos-panel" aria-labelledby="cronos-bandeja-paso">${cabeceraPanel}<div class="cuerpo-panel"><p class="cronos-${estado === "cargando" ? "vacio" : "acceso-denegado"}" role="${estado === "error" ? "alert" : "status"}">${escaparHTML(t(clave))}</p></div></section></section>`;
  }
  const filas = datos.pendientes.map((s) => `<tr><th scope="row">${escaparHTML(persona(s, t))}</th><td>${escaparHTML(s.nombre)}${s.justificante_exigido ? ` <span class="cronos-estado cronos-estado-aviso">${escaparHTML(t("requiere_justificante"))}</span>` : ""}</td>
    <td>${escaparHTML(periodoSolicitudCronos(s, t, locale))}</td><td class="numero">${escaparHTML(formatearCantidadCronos(s.cantidad, s.unidad, t, locale))}</td>
    <td>${escaparHTML(instanteVisible(s.solicitada_en, locale, zonaHoraria))}</td><td><span class="cronos-estado" data-estado="${escaparHTML(s.estado)}">${escaparHTML(t(estadoFila(s)))}</span></td>
    <td><button type="button" class="boton-secundario" data-cronos-resolver="${escaparHTML(s.solicitud_ref)}" aria-label="${escaparHTML(t("resolver_solicitud", { permiso: s.nombre, persona: persona(s, t) }))}">${escaparHTML(t("resolver"))} ›</button></td></tr>`);
  const cabeceras = ["col_persona", "permiso", "periodo", "duracion", "col_solicitada", "estado", "col_accion"];
  const cuerpo = filas.length ? filas.join("") : `<tr><td colspan="${cabeceras.length}">${escaparHTML(t("bandeja_vacia"))}</td></tr>`;
  const tabla = `<div class="cronos-tabla-contenedor"><table class="cronos-tabla"><thead><tr>${cabeceras.map((c) => `<th scope="col"${c === "duracion" ? ' class="numero"' : ""}>${escaparHTML(t(c))}</th>`).join("")}</tr></thead><tbody>${cuerpo}</tbody></table></div>`;
  const elegida = resolucion ? datos.pendientes.find((s) => s.solicitud_ref === resolucion.solicitudRef) : null;
  return `<section class="cronos-area cronos-bandeja-permisos" aria-labelledby="cronos-bandeja-titulo" data-estado="listo">${cabecera}
    <section class="panel cronos-panel" aria-labelledby="cronos-bandeja-paso">${cabeceraPanel}${aviso ? `<div class="cuerpo-panel">${aviso}</div>` : ""}${tabla}</section>
    ${formulario(resolucion, elegida, paso, t, locale)}</section>`;
}

function estadoError(error) {
  if (error instanceof ErrorClienteResolucionCronos) {
    if (error.codigo === "sin_empleado") return "sin_empleado";
    if (["acceso_denegado", "autenticacion_requerida", "no_competente"].includes(error.codigo)) return "denegado";
  }
  return "error";
}

export function montarBandejaPermisosCronos({ raiz, cliente = crearClienteResolucionCronosHTTP(), mensajes, anunciar = () => {}, registrarDesmontar,
  locale = "es-ES", zonaHoraria = "Europe/Madrid", paso = "responsable" } = {}) {
  if (!raiz?.append || !raiz.ownerDocument?.createElement || typeof cliente?.consultarBandeja !== "function" || typeof cliente?.resolver !== "function"
    || typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")
    || !PASOS_RESOLUCION_CRONOS.includes(paso)) throw new TypeError("montaje de la bandeja de permisos Cronos no disponible");
  const t = crearTraductorResolucionCronos(mensajes);
  const contenedor = raiz.ownerDocument.createElement("section"); contenedor.dataset.cronosBandejaPermisos = ""; raiz.append(contenedor);
  let activa = true; let secuencia = 0; let controlador = null; let envio = null;
  let pasoVisible = paso; let estado = "cargando"; let datos = null; let resolucion = null; let mensaje = ""; let tonoMensaje = "exito";
  const dibujar = () => { if (activa) contenedor.innerHTML = renderizarBandejaPermisosCronos({ estado, paso: pasoVisible, datos, resolucion, mensaje, tonoMensaje, mensajes, locale, zonaHoraria }); };
  const cargar = async () => {
    controlador?.abort(); controlador = new AbortController(); const turno = ++secuencia;
    estado = "cargando"; datos = null; dibujar();
    try {
      const r = await cliente.consultarBandeja({ paso: pasoVisible }, { signal: controlador.signal });
      if (!activa || turno !== secuencia) return;
      estado = "listo"; datos = r;
      if (resolucion && !r.pendientes.some((s) => s.solicitud_ref === resolucion.solicitudRef)) resolucion = null;
      dibujar();
    } catch (error) {
      if (!activa || turno !== secuencia || controlador.signal.aborted) return;
      estado = estadoError(error); dibujar(); anunciar(t(estado === "sin_empleado" ? "sin_empleado" : estado === "denegado" ? "denegado" : "error"));
    }
  };
  const alPulsar = (evento) => {
    const botonPaso = evento.target?.closest?.("[data-cronos-paso]");
    if (botonPaso && PASOS_RESOLUCION_CRONOS.includes(botonPaso.dataset.cronosPaso) && botonPaso.dataset.cronosPaso !== pasoVisible) {
      envio?.abort(); resolucion = null; mensaje = ""; pasoVisible = botonPaso.dataset.cronosPaso; void cargar();
      return;
    }
    const abrir = evento.target?.closest?.("[data-cronos-resolver]");
    if (abrir && datos?.pendientes.some((s) => s.solicitud_ref === abrir.dataset.cronosResolver)) {
      envio?.abort(); mensaje = "";
      resolucion = { solicitudRef: abrir.dataset.cronosResolver, clave: claveNueva(), decision: "", motivo: "" };
      dibujar(); contenedor.querySelector?.("[data-cronos-resolucion-formulario] [name=decision]")?.focus?.();
      return;
    }
    if (evento.target?.closest?.("[data-cronos-resolucion-cerrar]")) { envio?.abort(); resolucion = null; dibujar(); }
  };
  const alCambiar = (evento) => {
    // Denegar exige motivo: se refleja en el formulario sin perder lo escrito.
    if (!resolucion || evento.target?.name !== "decision") return;
    const motivo = contenedor.querySelector?.("[data-cronos-resolucion-formulario] [name=motivo]");
    resolucion = { ...resolucion, decision: evento.target.value, motivo: motivo?.value ?? resolucion.motivo };
    if (motivo) motivo.required = resolucion.decision === "denegar";
  };
  const alEnviar = async (evento) => {
    if (!evento.target?.matches?.("[data-cronos-resolucion-formulario]") || !resolucion || resolucion.estado === "enviando") return;
    evento.preventDefault();
    const solicitud = datos?.pendientes.find((s) => s.solicitud_ref === resolucion.solicitudRef);
    if (!solicitud) return;
    const elementos = evento.target.elements;
    const decision = elementos?.namedItem?.("decision")?.value ?? resolucion.decision;
    const motivo = String(elementos?.namedItem?.("motivo")?.value ?? "").trim();
    if (!["aprobar", "denegar"].includes(decision) || (decision === "denegar" && !motivo) || !motivoResolucionValido(motivo)) {
      resolucion = { ...resolucion, decision, motivo, estado: "error", mensaje: t("error_motivo") }; dibujar(); anunciar(resolucion.mensaje);
      return;
    }
    // Un reintento de la misma resolución conserva la clave; otra, usa otra.
    const firma = JSON.stringify([decision, motivo]);
    if (resolucion.firma && resolucion.firma !== firma) resolucion.clave = claveNueva();
    resolucion = { ...resolucion, decision, motivo, firma, estado: "enviando", mensaje: "" };
    dibujar();
    envio = new AbortController();
    try {
      const recibo = await cliente.resolver({ clave_operacion: resolucion.clave, solicitud_ref: solicitud.solicitud_ref, paso: pasoVisible, decision,
        version_esperada: solicitud.version, ...(motivo ? { motivo } : {}) }, { signal: envio.signal });
      if (!activa) return;
      mensaje = t(recibo.replay ? "resolucion_ya_registrada" : `resolucion_${recibo.estado}`); tonoMensaje = "exito";
      resolucion = null; anunciar(mensaje);
      await cargar();
    } catch (error) {
      if (!activa || envio.signal.aborted) return;
      const codigo = error instanceof ErrorClienteResolucionCronos ? error.codigo : error instanceof TypeError ? "peticion_invalida" : "";
      const texto = t(ERRORES.get(codigo) ?? "error_resolucion");
      anunciar(texto);
      // Si otra resolución se adelantó, se cierra el formulario y se recarga.
      if (codigo === "estado_cambiado") { resolucion = null; mensaje = texto; tonoMensaje = "error"; await cargar(); return; }
      resolucion = { ...resolucion, estado: "error", mensaje: texto };
      dibujar();
    }
  };
  contenedor.addEventListener("click", alPulsar); contenedor.addEventListener("change", alCambiar); contenedor.addEventListener("submit", alEnviar);
  void cargar();
  const desmontar = () => {
    if (!activa) return;
    activa = false; ++secuencia; controlador?.abort(); envio?.abort();
    contenedor.removeEventListener("click", alPulsar); contenedor.removeEventListener("change", alCambiar); contenedor.removeEventListener("submit", alEnviar);
    contenedor.remove?.();
  };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ desmontar, recargar: cargar });
}
