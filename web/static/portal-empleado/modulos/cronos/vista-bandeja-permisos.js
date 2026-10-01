import { crearTraductorResolucionCronos, periodoSolicitudCronos } from "./i18n-resolucion.js?v=20261001-cronos-grafo-bandeja-v5";
import { formatearCantidadCronos } from "./i18n-solicitudes.js";
import {
  ErrorClienteResolucionCronos, MAXIMO_MOTIVO_RESOLUCION_CRONOS, PASOS_RESOLUCION_CRONOS, crearClienteResolucionCronosHTTP, motivoResolucionValido,
} from "./cliente-resolucion-http.js";
import { icono } from "../../../comun/iconos-vec.js?v=20260925-aspecto-v1";
import { LOCALIZACION_ACTUAL } from "../../../comun/idioma.js";

// El motivo se comprueba antes de enviar; un 400 del servidor es genérico.
const ERRORES = new Map([
  ["peticion_invalida", "error_peticion_resolucion"], ["no_competente", "error_no_competente"], ["estado_cambiado", "error_estado_cambiado"],
  ["conflicto", "error_conflicto_resolucion"], ["pendiente_asignacion", "error_pendiente_asignacion"],
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
    <div class="cuerpo-panel"><form class="cronos-resolucion-formulario" data-cronos-resolucion-formulario tabindex="-1" aria-label="${escaparHTML(titulo)}">
    <fieldset class="cronos-resolucion-decision"><legend>${escaparHTML(t("decision"))}</legend>${opcion("aprobar", paso === "responsable" ? "decision_aprobar_responsable" : "decision_aprobar_administracion")}${opcion("denegar", "decision_denegar")}</fieldset>
    <label class="cronos-resolucion-motivo">${escaparHTML(t("motivo"))}<textarea name="motivo" rows="3" maxlength="${MAXIMO_MOTIVO_RESOLUCION_CRONOS}"${r.decision === "denegar" ? " required" : ""}${inactivo}>${escaparHTML(r.motivo ?? "")}</textarea></label>
    <div class="cronos-solicitud-acciones"><button type="submit" class="boton-primario" data-cronos-resolucion-enviar${inactivo}>${escaparHTML(t(enviando ? "resolucion_enviando" : "resolucion_enviar"))}</button>
    <button type="button" class="boton-secundario" data-cronos-resolucion-cerrar>${escaparHTML(t("cancelar"))}</button></div>${aviso}</form></div></section>`;
}

function estadoFila(s) {
  if (s.pendiente_asignacion) return "estado_pendiente_asignacion";
  return s.estado === "pendiente_administracion" || s.circuito === "A" ? "estado_pendiente_administracion" : "estado_pendiente_jefatura";
}

/** Lo pendiente de asignar jefatura no se resuelve: se explica en la fila. */
function accionFila(s, t) {
  if (s.pendiente_asignacion) return `<span class="cronos-sin-jefatura">${escaparHTML(t("sin_jefatura_asignada"))}</span>`;
  return `<button type="button" class="boton-secundario" data-cronos-resolver="${escaparHTML(s.solicitud_ref)}" aria-label="${escaparHTML(t("resolver_solicitud", { permiso: s.nombre, persona: persona(s, t) }))}">${escaparHTML(t("resolver"))} ›</button>`;
}

const TAMANIO_PAGINA = 20;
const FILTROS_ESTADO = Object.freeze(["todos", "jefatura", "rrhh", "asignacion"]);
function estadoFiltro(s) {
  if (s.pendiente_asignacion) return "asignacion";
  return s.estado === "pendiente_administracion" || s.circuito === "A" ? "rrhh" : "jefatura";
}
function textoBusqueda(valor) { return String(valor ?? "").normalize("NFKD").replace(/\p{M}/gu, "").toLowerCase().trim(); }
function seleccionarPendientes(datos, filtros, pagina, t) {
  const busqueda = textoBusqueda(filtros.busqueda);
  const filas = datos.pendientes.filter((s) => (!busqueda || textoBusqueda(persona(s, t)).includes(busqueda))
    && (filtros.estado === "todos" || estadoFiltro(s) === filtros.estado));
  const paginas = Math.max(1, Math.ceil(filas.length / TAMANIO_PAGINA));
  const actual = Math.min(paginas, Math.max(1, Number.isSafeInteger(pagina) ? pagina : 1));
  return { filas, paginas, actual, visibles: filas.slice((actual - 1) * TAMANIO_PAGINA, actual * TAMANIO_PAGINA) };
}
function controlesBandeja(datos, seleccion, filtros, paso, t, locale, enviando) {
  const numero = (n) => new Intl.NumberFormat(locale).format(n);
  const deshabilitado = enviando ? " disabled" : "";
  const resumen = [["resumen_total", datos.pendientes.length, "documento"], ["resumen_resolubles", datos.pendientes.filter((s) => !s.pendiente_asignacion).length, "pendiente"],
    ["resumen_asignacion", datos.pendientes.filter((s) => s.pendiente_asignacion).length, "alerta"]];
  return `<div class="cuerpo-panel"><p>${escaparHTML(t("resumen_alcance", { paso: t(`paso_${paso}`) }))}</p>
    <div class="rejilla-kpi">${resumen.map(([clave, cantidad, nombreIcono]) => `<article class="tarjeta-kpi"><span class="icono-kpi">${icono(nombreIcono)}</span><div><span class="etiqueta-kpi">${escaparHTML(t(clave))}</span><strong class="valor-kpi">${numero(cantidad)}</strong></div></article>`).join("")}</div>
    <form data-cronos-bandeja-filtros class="cronos-jornada-periodo-formulario">
      <label>${escaparHTML(t("buscar_persona"))}<input type="search" name="busqueda" maxlength="120" value="${escaparHTML(filtros.busqueda)}"${deshabilitado}></label>
      <label>${escaparHTML(t("filtrar_estado"))}<select name="estado"${deshabilitado}>${FILTROS_ESTADO.map((e) => `<option value="${e}"${filtros.estado === e ? " selected" : ""}>${escaparHTML(t(`filtro_${e}`))}</option>`).join("")}</select></label>
      <button type="submit" class="boton-secundario" data-cronos-bandeja-aplicar${deshabilitado}>${escaparHTML(t("aplicar_filtros"))}</button>
      <button type="button" class="boton-secundario" data-cronos-bandeja-limpiar${deshabilitado}>${escaparHTML(t("limpiar_filtros"))}</button>
    </form>
    <p role="status" aria-live="polite">${escaparHTML(t("resultados_pagina", { desde: numero(seleccion.filas.length ? (seleccion.actual - 1) * TAMANIO_PAGINA + 1 : 0),
      hasta: numero(Math.min(seleccion.actual * TAMANIO_PAGINA, seleccion.filas.length)), total: numero(seleccion.filas.length), bandeja: numero(datos.pendientes.length) }))}</p></div>`;
}
function paginasBandeja(seleccion, t, locale, enviando) {
  if (seleccion.paginas === 1) return "";
  const numero = (n) => new Intl.NumberFormat(locale).format(n);
  return `<nav class="cuerpo-panel cronos-solicitud-acciones" aria-label="${escaparHTML(t("paginas_etiqueta"))}">
    <button type="button" class="boton-secundario" data-cronos-bandeja-pagina="anterior"${seleccion.actual === 1 || enviando ? " disabled" : ""}>${escaparHTML(t("pagina_anterior"))}</button>
    <span data-cronos-bandeja-paginacion tabindex="-1">${escaparHTML(t("pagina_indicador", { pagina: numero(seleccion.actual), paginas: numero(seleccion.paginas) }))}</span>
    <button type="button" class="boton-secundario" data-cronos-bandeja-pagina="siguiente"${seleccion.actual === seleccion.paginas || enviando ? " disabled" : ""}>${escaparHTML(t("pagina_siguiente"))}</button></nav>`;
}

/** Bandeja de un paso: solicitudes pendientes de quien resuelve y el formulario de resolución. */
export function renderizarBandejaPermisosCronos({ estado = "cargando", paso = "responsable", datos = null, resolucion = null, mensaje = "", tonoMensaje = "exito",
  mensajes, locale = LOCALIZACION_ACTUAL, zonaHoraria = "Europe/Madrid", filtros = { busqueda: "", estado: "todos" }, pagina = 1 } = {}) {
  const t = crearTraductorResolucionCronos(mensajes);
  if (!PASOS_RESOLUCION_CRONOS.includes(paso)) throw new RangeError("paso de resolución no válido");
  const ayuda = t("abrir_ayuda", { asunto: t("bandeja_titulo") });
  const cabecera = `<header class="cronos-encabezado"><div><p class="sobrelinea">${escaparHTML(t("sobrelinea"))}</p><h2 id="cronos-bandeja-titulo">${escaparHTML(t("bandeja_titulo"))}</h2></div>
    <button type="button" class="cronos-boton-ayuda" data-accion="ayuda" aria-label="${escaparHTML(ayuda)}" title="${escaparHTML(ayuda)}"><span aria-hidden="true">?</span></button></header>`;
  const selector = `<div class="cronos-selector-paso" role="group" aria-label="${escaparHTML(t("paso_selector"))}">${PASOS_RESOLUCION_CRONOS.map((p) =>
    `<button type="button" class="boton-secundario" data-cronos-paso="${p}" aria-pressed="${p === paso}">${escaparHTML(t(`paso_${p}`))}</button>`).join("")}</div>`;
  const cabeceraPanel = `<div class="cabecera-panel"><h3 id="cronos-bandeja-paso">${escaparHTML(t(`paso_${paso}`))}</h3>${selector}</div>`;
  const tono = tonoMensaje === "error" ? "error" : "exito";
  const aviso = mensaje ? `<p class="cronos-solicitud-aviso" data-cronos-bandeja-resultado tabindex="-1" data-tono="${tono}" role="${tono === "error" ? "alert" : "status"}">${escaparHTML(mensaje)}</p>` : "";
  if (estado !== "listo") {
    const clave = { denegado: "denegado", sin_empleado: "sin_empleado", error: "error" }[estado] ?? "cargando";
    return `<section class="cronos-area cronos-bandeja-permisos" aria-labelledby="cronos-bandeja-titulo" data-estado="${escaparHTML(estado)}">${cabecera}
      <section class="panel cronos-panel" aria-labelledby="cronos-bandeja-paso">${cabeceraPanel}<div class="cuerpo-panel"><p class="cronos-${estado === "cargando" ? "vacio" : "acceso-denegado"}" data-cronos-bandeja-estado tabindex="-1" role="${estado === "error" ? "alert" : "status"}">${escaparHTML(t(clave))}</p></div></section></section>`;
  }
  const seleccion = seleccionarPendientes(datos, filtros, pagina, t);
  const enviando = resolucion?.estado === "enviando";
  const controles = controlesBandeja(datos, seleccion, filtros, paso, t, locale, enviando);
  const filas = seleccion.visibles.map((s) => `<tr><th scope="row">${escaparHTML(persona(s, t))}</th><td>${escaparHTML(s.nombre)}${s.justificante_exigido ? ` <span class="cronos-estado cronos-estado-aviso">${escaparHTML(t("requiere_justificante"))}</span>` : ""}</td>
    <td>${escaparHTML(periodoSolicitudCronos(s, t, locale))}</td><td class="numero">${escaparHTML(formatearCantidadCronos(s.cantidad, s.unidad, t, locale))}</td>
    <td>${escaparHTML(instanteVisible(s.solicitada_en, locale, zonaHoraria))}</td><td><span class="cronos-estado" data-estado="${escaparHTML(s.estado)}">${escaparHTML(t(estadoFila(s)))}</span></td>
    <td>${accionFila(s, t)}</td></tr>`);
  const cabeceras = ["col_persona", "permiso", "periodo", "duracion", "col_solicitada", "estado", "col_accion"];
  const cuerpo = filas.length ? filas.join("") : `<tr><td colspan="${cabeceras.length}">${escaparHTML(t(datos.pendientes.length ? "sin_resultados_filtro" : "bandeja_vacia"))}</td></tr>`;
  const tabla = `<div class="cronos-tabla-contenedor"><table class="cronos-tabla"><thead><tr>${cabeceras.map((c) => `<th scope="col"${c === "duracion" ? ' class="numero"' : ""}>${escaparHTML(t(c))}</th>`).join("")}</tr></thead><tbody>${cuerpo}</tbody></table></div>`;
  const elegida = resolucion ? datos.pendientes.find((s) => s.solicitud_ref === resolucion.solicitudRef) : null;
  return `<section class="cronos-area cronos-bandeja-permisos" aria-labelledby="cronos-bandeja-titulo" data-estado="listo">${cabecera}
    <section class="panel cronos-panel" aria-labelledby="cronos-bandeja-paso">${cabeceraPanel}${controles}${aviso ? `<div class="cuerpo-panel">${aviso}</div>` : ""}${tabla}${paginasBandeja(seleccion, t, locale, enviando)}</section>
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
  locale = LOCALIZACION_ACTUAL, zonaHoraria = "Europe/Madrid", paso = "responsable" } = {}) {
  if (!raiz?.append || !raiz.ownerDocument?.createElement || typeof cliente?.consultarBandeja !== "function" || typeof cliente?.resolver !== "function"
    || typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")
    || !PASOS_RESOLUCION_CRONOS.includes(paso)) throw new TypeError("montaje de la bandeja de permisos Cronos no disponible");
  const t = crearTraductorResolucionCronos(mensajes);
  const contenedor = raiz.ownerDocument.createElement("section"); contenedor.dataset.cronosBandejaPermisos = ""; raiz.append(contenedor);
  let activa = true; let secuencia = 0; let controlador = null; let envio = null;
  let pasoVisible = paso; let estado = "cargando"; let datos = null; let resolucion = null; let mensaje = ""; let tonoMensaje = "exito";
  let filtros = { busqueda: "", estado: "todos" }; let pagina = 1;
  let focoPendiente = null;
  const selectorFoco = (foco) => {
    const paso = foco?.getAttribute?.("data-cronos-paso");
    if (PASOS_RESOLUCION_CRONOS.includes(paso)) return `[data-cronos-paso="${paso}"]`;
    const nombre = foco?.getAttribute?.("name");
    if (nombre === "decision" && ["aprobar", "denegar"].includes(foco.value)) return `[name="decision"][value="${foco.value}"]`;
    if (["busqueda", "estado", "motivo"].includes(nombre)) return `[name="${nombre}"]`;
    for (const atributo of ["data-cronos-resolucion-enviar", "data-cronos-resolucion-cerrar", "data-cronos-bandeja-aplicar", "data-cronos-bandeja-limpiar", "data-cronos-bandeja-paginacion"]) {
      if (foco?.hasAttribute?.(atributo)) return `[${atributo}]`;
    }
    const paginaFoco = foco?.getAttribute?.("data-cronos-bandeja-pagina");
    if (["anterior", "siguiente"].includes(paginaFoco)) return `[data-cronos-bandeja-pagina="${paginaFoco}"]`;
    const solicitud = foco?.getAttribute?.("data-cronos-resolver");
    if (typeof solicitud === "string" && /^[A-Za-z0-9:._-]{1,200}$/u.test(solicitud)) return `[data-cronos-resolver="${solicitud}"]`;
    return foco?.getAttribute?.("data-accion") === "ayuda" ? '[data-accion="ayuda"]' : null;
  };
  const dibujar = () => {
    if (!activa) return;
    const documento = raiz.ownerDocument;
    const foco = documento.activeElement;
    const dentro = contenedor.contains?.(foco) === true;
    const documentoActivo = documento.hasFocus?.() !== false;
    // Un destino provisional conserva el control hasta terminar la lectura o
    // el envío. Salir del módulo cancela esa intención antes de responder.
    if ((foco && foco !== documento.body && !dentro) || !documentoActivo) focoPendiente = null;
    const selectorActual = dentro && documentoActivo ? selectorFoco(foco) : null;
    if (selectorActual) focoPendiente = selectorActual;
    const restaurar = documentoActivo && (dentro || (foco === documento.body && focoPendiente));
    contenedor.innerHTML = renderizarBandejaPermisosCronos({ estado, paso: pasoVisible, datos, resolucion, mensaje, tonoMensaje, mensajes, locale, zonaHoraria, filtros, pagina });
    if (!restaurar || !focoPendiente) return;
    const control = contenedor.querySelector?.(focoPendiente);
    const destino = control && !control.disabled ? control
      : contenedor.querySelector?.("[data-cronos-bandeja-estado]")
        ?? contenedor.querySelector?.("[data-cronos-bandeja-resultado]")
        ?? contenedor.querySelector?.("[data-cronos-resolucion-formulario]");
    destino?.focus?.();
    if (destino === control && !control.disabled) focoPendiente = null;
  };
  const cargar = async () => {
    conservarBorrador();
    controlador?.abort(); controlador = new AbortController(); const turno = ++secuencia;
    estado = "cargando"; datos = null; dibujar();
    try {
      const r = await cliente.consultarBandeja({ paso: pasoVisible }, { signal: controlador.signal });
      if (!activa || turno !== secuencia) return;
      estado = "listo"; datos = r; pagina = seleccionarPendientes(datos, filtros, pagina, t).actual;
      if (resolucion && !r.pendientes.some((s) => s.solicitud_ref === resolucion.solicitudRef)) resolucion = null;
      dibujar();
    } catch (error) {
      if (!activa || turno !== secuencia || controlador.signal.aborted) return;
      estado = estadoError(error); dibujar(); anunciar(t(estado === "sin_empleado" ? "sin_empleado" : estado === "denegado" ? "denegado" : "error"));
    }
  };
  const conservarBorrador = () => {
    if (!resolucion || resolucion.estado === "enviando") return;
    const motivo = contenedor.querySelector?.("[data-cronos-resolucion-formulario] [name=motivo]");
    if (motivo) resolucion = { ...resolucion, motivo: motivo.value };
  };
  const alPulsar = (evento) => {
    const limpiar = evento.target?.closest?.("[data-cronos-bandeja-limpiar]");
    const botonPagina = evento.target?.closest?.("[data-cronos-bandeja-pagina]");
    if ((limpiar || botonPagina) && estado === "listo" && resolucion?.estado !== "enviando") {
      conservarBorrador();
      if (limpiar) { filtros = { busqueda: "", estado: "todos" }; pagina = 1; }
      else {
        const seleccion = seleccionarPendientes(datos, filtros, pagina, t);
        pagina = Math.min(seleccion.paginas, Math.max(1, seleccion.actual + (botonPagina.dataset.cronosBandejaPagina === "siguiente" ? 1 : -1)));
      }
      dibujar();
      contenedor.querySelector?.(limpiar ? "[name=busqueda]" : "[data-cronos-bandeja-paginacion]")?.focus?.();
      return;
    }
    const botonPaso = evento.target?.closest?.("[data-cronos-paso]");
    if (botonPaso && PASOS_RESOLUCION_CRONOS.includes(botonPaso.dataset.cronosPaso) && botonPaso.dataset.cronosPaso !== pasoVisible) {
      envio?.abort(); resolucion = null; mensaje = ""; filtros = { busqueda: "", estado: "todos" }; pagina = 1; pasoVisible = botonPaso.dataset.cronosPaso; void cargar();
      return;
    }
    const abrir = evento.target?.closest?.("[data-cronos-resolver]");
    if (abrir && datos?.pendientes.some((s) => s.solicitud_ref === abrir.dataset.cronosResolver && !s.pendiente_asignacion)) {
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
    if (evento.target?.matches?.("[data-cronos-bandeja-filtros]")) {
      evento.preventDefault();
      if (estado !== "listo" || resolucion?.estado === "enviando") return;
      const elementos = evento.target.elements;
      const busqueda = String(elementos?.namedItem?.("busqueda")?.value ?? "").slice(0, 120);
      const filtro = elementos?.namedItem?.("estado")?.value;
      if (!FILTROS_ESTADO.includes(filtro)) return;
      conservarBorrador(); filtros = { busqueda, estado: filtro }; pagina = 1; dibujar();
      contenedor.querySelector?.("[data-cronos-bandeja-aplicar]")?.focus?.();
      return;
    }
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
    const envioActual = new AbortController(); envio = envioActual;
    const claveOperacion = resolucion.clave; const pasoOperacion = pasoVisible;
    const envioVigente = () => activa && !envioActual.signal.aborted && envio === envioActual
      && resolucion?.clave === claveOperacion && pasoVisible === pasoOperacion;
    try {
      const recibo = await cliente.resolver({ clave_operacion: resolucion.clave, solicitud_ref: solicitud.solicitud_ref, paso: pasoVisible, decision,
        version_esperada: solicitud.version, ...(motivo ? { motivo } : {}) }, { signal: envioActual.signal });
      if (!envioVigente()) return;
      mensaje = t(recibo.replay ? "resolucion_ya_registrada" : `resolucion_${recibo.estado}`); tonoMensaje = "exito";
      resolucion = null; anunciar(mensaje);
      await cargar();
    } catch (error) {
      if (!envioVigente()) return;
      const codigo = error instanceof ErrorClienteResolucionCronos ? error.codigo : error instanceof TypeError ? "peticion_invalida" : "";
      const texto = t(ERRORES.get(codigo) ?? "error_resolucion");
      anunciar(texto);
      // Si otra resolución se adelantó, se cierra el formulario y se recarga.
      if (codigo === "estado_cambiado" || codigo === "pendiente_asignacion") { resolucion = null; mensaje = texto; tonoMensaje = "error"; await cargar(); return; }
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
