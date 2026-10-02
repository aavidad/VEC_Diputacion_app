import {
  crearTraductorNotificacionesCronos, documentoNotificacionCronos, fechaCivilVisibleCronos, instanteVisibleCronos, numeroVisibleCronos,
} from "./i18n-notificaciones.js";
import {
  ErrorClienteNotificacionesCronos, MAXIMO_TEXTO_NOTIFICACION_CRONOS, adjuntoNotificacionValido, calcularHuellaDocumentoCronos,
  crearClienteNotificacionesCronosHTTP, textoNotificacionValido,
} from "./cliente-notificaciones-http.js";
import { crearTraductorNotificacionesHistorialCronos } from "./i18n-notificaciones-historial.js?v=20261001-cronos-c9-historial-v2";
import { LOCALIZACION_ACTUAL } from "../../../comun/idioma.js";

const ERRORES_ENVIO = new Map([
  ["peticion_invalida", "error_datos"], ["tipo_no_vigente", "error_tipo_no_vigente"], ["conflicto", "error_conflicto_notificacion"],
]);

function escaparHTML(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}
function claveNueva() { return globalThis.crypto.randomUUID(); }

/** Fecha civil de hoy en la zona de la persona (AAAA-MM-DD). */
export function hoyCivilCronos(ahora = new Date(), zonaHoraria = "Europe/Madrid") {
  const partes = Object.fromEntries(new Intl.DateTimeFormat(/* localización técnica */ "en-CA", { timeZone: zonaHoraria, year: "numeric", month: "2-digit", day: "2-digit" })
    .formatToParts(ahora).map((p) => [p.type, p.value]));
  return `${partes.year}-${partes.month}-${partes.day}`;
}

function formularioVacio(hoy) {
  return { tipo: "", fecha: hoy, texto: "", referencia: "", huella: "", documento: "" };
}

function formularioNotificacion(f, datos, envio, t, locale) {
  const enviando = envio?.enviando === true;
  const inactivo = enviando ? " disabled" : "";
  const opciones = datos.tipos.map((tipo) => `<option value="${escaparHTML(tipo.tipo_version_ref)}"${f.tipo === tipo.tipo_version_ref ? " selected" : ""}>${escaparHTML(tipo.nombre)}</option>`).join("");
  const usados = [...f.texto].length;
  const documento = f.documento
    ? `<p class="cronos-documento-estado" data-cronos-documento-estado data-tono="${f.documento === "documento_error" ? "error" : "exito"}" role="status">${escaparHTML(t(f.documento))}</p>`
    : `<p class="cronos-documento-estado" data-cronos-documento-estado role="status"></p>`;
  const aviso = avisoEnvio(envio?.mensaje);
  if (!datos.tipos.length) return `<p class="cronos-vacio" data-cronos-notificacion-sin-tipos role="status">${escaparHTML(t("sin_tipos"))}</p>`;
  return `<form class="cronos-notificacion-formulario" data-cronos-notificacion-formulario aria-labelledby="cronos-notificacion-nueva">
    <div class="cronos-notificacion-fila">
      <label class="cronos-campo">${escaparHTML(t("campo_tipo"))}<select name="tipo" required${inactivo}><option value="">${escaparHTML(t("campo_tipo_elegir"))}</option>${opciones}</select></label>
      <label class="cronos-campo">${escaparHTML(t("campo_fecha"))}<input type="date" name="fecha" value="${escaparHTML(f.fecha)}" min="2000-01-01" max="2100-12-31" required${inactivo}></label>
    </div>
    <label class="cronos-campo">${escaparHTML(t("campo_texto"))}<textarea name="texto" rows="5" maxlength="${MAXIMO_TEXTO_NOTIFICACION_CRONOS}" required aria-describedby="cronos-notificacion-cuenta"${inactivo}>${escaparHTML(f.texto)}</textarea>
      <span id="cronos-notificacion-cuenta" class="cronos-cuenta" data-cronos-cuenta aria-live="polite">${escaparHTML(t("campo_texto_cuenta", { usados: numeroVisibleCronos(usados, locale), maximo: numeroVisibleCronos(MAXIMO_TEXTO_NOTIFICACION_CRONOS, locale) }))}</span></label>
    <div class="cronos-notificacion-fila">
      <label class="cronos-campo">${escaparHTML(t("campo_documento_ref"))}<input type="text" name="referencia" value="${escaparHTML(f.referencia)}" maxlength="128" autocomplete="off"${inactivo}></label>
      <label class="cronos-campo">${escaparHTML(t("campo_documento"))}<input type="file" name="documento" data-cronos-documento${inactivo}></label>
    </div>
    ${documento}
    <div class="cronos-solicitud-acciones"><button type="submit" class="boton-primario" data-cronos-notificacion-enviar${inactivo}>${escaparHTML(t(enviando ? "enviando" : "enviar"))}</button></div>
    <div data-cronos-envio-aviso>${aviso}</div>
  </form>`;
}

function avisoEnvio(mensaje) {
  return mensaje ? `<p class="cronos-solicitud-aviso" data-tono="error" role="alert">${escaparHTML(mensaje)}</p>` : "";
}


function filaNotificacion(n, t, locale, zonaHoraria) {
  const estado = n.estado === "atendida" ? t("estado_atendida", { fecha: instanteVisibleCronos(n.atendida_en, locale, zonaHoraria) }) : t("estado_registrada");
  const documento = documentoNotificacionCronos(n, t);
  return `<tr><td>${escaparHTML(instanteVisibleCronos(n.registrada_en, locale, zonaHoraria))}</td><th scope="row">${escaparHTML(n.tipo_nombre)}</th>
    <td>${escaparHTML(fechaCivilVisibleCronos(n.fecha_referida, locale))}</td><td class="cronos-notificacion-texto">${escaparHTML(n.texto)}</td>
    <td>${documento}</td><td><span class="cronos-estado" data-estado="${escaparHTML(n.estado)}">${escaparHTML(estado)}</span></td></tr>`;
}

function paginaHistorial(notificaciones, historial, tamanoPagina) {
  if (!Number.isSafeInteger(tamanoPagina) || tamanoPagina < 1 || tamanoPagina > 100) throw new RangeError("tamaño de página Cronos no válido");
  const filtro = ["registrada", "atendida"].includes(historial.filtro) ? historial.filtro : "";
  const filtradas = notificaciones.filter((n) => !filtro || n.estado === filtro);
  const paginas = Math.max(1, Math.ceil(filtradas.length / tamanoPagina));
  const pagina = Math.max(1, Math.min(Number.isSafeInteger(historial.pagina) ? historial.pagina : 1, paginas));
  const inicio = (pagina - 1) * tamanoPagina;
  return { filtro, pagina, paginas, inicio, total: filtradas.length, filas: filtradas.slice(inicio, inicio + tamanoPagina) };
}

/** Filtra y pagina únicamente la respuesta ya consultada; no solicita otras filas. */
export function renderizarHistorialNotificacionesCronos({ datos, historial = {}, estado = "listo", mensajes, mensajesHistorial,
  tamanoPagina = 10, locale = LOCALIZACION_ACTUAL, zonaHoraria = "Europe/Madrid" } = {}) {
  const t = crearTraductorNotificacionesCronos(mensajes);
  const th = crearTraductorNotificacionesHistorialCronos(mensajesHistorial);
  const cargando = estado === "cargando";
  const cabecera = `<div class="cabecera-panel"><h3 id="cronos-mis-notificaciones">${escaparHTML(t("mis_notificaciones"))}</h3>
    <button type="button" class="boton-secundario" data-cronos-historial-actualizar${cargando ? " disabled" : ""}>${escaparHTML(th(cargando ? "actualizando" : "actualizar"))}</button></div>`;
  if (estado !== "listo") return `${cabecera}<div class="cuerpo-panel" aria-busy="${cargando}"><p data-cronos-historial-resumen tabindex="-1" role="${estado === "error" ? "alert" : "status"}">${escaparHTML(th(cargando ? "actualizando" : "error_consulta"))}</p></div>`;
  const p = paginaHistorial(datos.notificaciones, historial, tamanoPagina);
  const n = (v) => numeroVisibleCronos(v, locale);
  const opciones = [["", "todos"], ["registrada", "registradas"], ["atendida", "atendidas"]]
    .map(([valor, clave]) => `<option value="${valor}"${p.filtro === valor ? " selected" : ""}>${escaparHTML(th(clave))}</option>`).join("");
  const cabeceras = ["col_enviada", "col_tipo", "col_fecha", "col_mensaje", "col_documento", "col_estado"];
  const filas = p.filas.map((fila) => filaNotificacion(fila, t, locale, zonaHoraria)).join("");
  const resumen = th("mostrando", { desde: n(p.total ? p.inicio + 1 : 0), hasta: n(p.inicio + p.filas.length), total: n(p.total) });
  const filtro = th(p.filtro === "registrada" ? "registradas" : p.filtro === "atendida" ? "atendidas" : "todos");
  return `${cabecera}<div class="cuerpo-panel">
    <p>${escaparHTML(th("alcance"))}</p>
    <form class="cronos-notificacion-formulario" data-cronos-historial-filtros>
      <label class="cronos-campo">${escaparHTML(th("estado"))}<select class="control-formulario" name="estado_historial" data-cronos-historial-estado>${opciones}</select></label>
      <div class="cronos-solicitud-acciones"><button type="submit" class="boton-secundario">${escaparHTML(th("aplicar"))}</button>
        <button type="button" class="boton-secundario" data-cronos-historial-quitar${p.filtro ? "" : " disabled"}>${escaparHTML(th("quitar"))}</button></div>
    </form>
    <p data-cronos-historial-resumen tabindex="-1" role="status" aria-live="polite">${escaparHTML(th("filtro_activo", { estado: filtro }))} ${escaparHTML(resumen)}</p>
  </div><div class="cronos-tabla-contenedor"><table class="cronos-tabla" aria-labelledby="cronos-mis-notificaciones"><thead><tr>${cabeceras.map((c) => `<th scope="col">${escaparHTML(t(c))}</th>`).join("")}</tr></thead>
    <tbody>${filas || `<tr><td colspan="${cabeceras.length}">${escaparHTML(p.filtro && datos.notificaciones.length ? th("sin_resultados") : t("sin_notificaciones"))}</td></tr>`}</tbody></table></div>
    <nav class="cronos-solicitud-acciones cuerpo-panel" aria-label="${escaparHTML(th("paginacion"))}">
      <span>${escaparHTML(th("pagina", { actual: n(p.pagina), total: n(p.paginas) }))}</span>
      <button type="button" class="boton-secundario" data-cronos-historial-pagina="${p.pagina - 1}"${p.pagina === 1 ? " disabled" : ""}>${escaparHTML(th("anterior"))}</button>
      <button type="button" class="boton-secundario" data-cronos-historial-pagina="${p.pagina + 1}"${p.pagina === p.paginas ? " disabled" : ""}>${escaparHTML(th("siguiente"))}</button></nav>`;
}

/** Notificaciones propias a RRHH: formulario de envío y lista con su estado. */
export function renderizarNotificacionesPropiasCronos({ estado = "cargando", datos = null, formulario = null, envio = null, mensaje = "", tonoMensaje = "exito",
  mensajes, mensajesHistorial, historial = {}, estadoHistorial = "listo", tamanoPagina = 10, locale = LOCALIZACION_ACTUAL, zonaHoraria = "Europe/Madrid", hoy = hoyCivilCronos(new Date(), zonaHoraria) } = {}) {
  const t = crearTraductorNotificacionesCronos(mensajes);
  const ayuda = t("abrir_ayuda", { asunto: t("notificaciones_titulo") });
  const cabecera = `<header class="cronos-encabezado"><div><p class="sobrelinea">${escaparHTML(t("sobrelinea"))}</p><h2 id="cronos-notificaciones-propias-titulo">${escaparHTML(t("notificaciones_titulo"))}</h2></div>
    <button type="button" class="cronos-boton-ayuda" data-accion="ayuda" aria-label="${escaparHTML(ayuda)}" title="${escaparHTML(ayuda)}"><span aria-hidden="true">?</span></button></header>`;
  const tono = tonoMensaje === "error" ? "error" : "exito";
  const aviso = mensaje ? `<p class="cronos-solicitud-aviso" data-cronos-notificacion-mensaje data-tono="${tono}" role="${tono === "error" ? "alert" : "status"}" tabindex="-1">${escaparHTML(mensaje)}</p>` : "";
  if (estado !== "listo") {
    const clave = { denegado: "denegado", sin_empleado: "sin_empleado", error: "error" }[estado] ?? "cargando";
    const reintento = estado === "error" ? `<div class="cronos-solicitud-acciones"><button type="button" class="boton-secundario" data-cronos-notificacion-reintentar>${escaparHTML(t("notificaciones_reintentar_consulta"))}</button></div>` : "";
    return `<section class="cronos-area cronos-notificaciones-propias" aria-labelledby="cronos-notificaciones-propias-titulo" data-estado="${escaparHTML(estado)}">${cabecera}
      <section class="panel cronos-panel"><div class="cuerpo-panel">${aviso}<p class="cronos-${estado === "cargando" ? "vacio" : "acceso-denegado"}" role="${estado === "error" ? "alert" : "status"}">${escaparHTML(t(clave))}</p>${reintento}</div></section></section>`;
  }
  const f = formulario ?? formularioVacio(hoy);
  const contenidoHistorial = renderizarHistorialNotificacionesCronos({ datos, historial, estado: estadoHistorial, tamanoPagina, mensajes, mensajesHistorial, locale, zonaHoraria });
  return `<section class="cronos-area cronos-notificaciones-propias" aria-labelledby="cronos-notificaciones-propias-titulo" data-estado="listo">${cabecera}
    <section class="panel cronos-panel" aria-labelledby="cronos-notificacion-nueva"><div class="cabecera-panel"><h3 id="cronos-notificacion-nueva">${escaparHTML(t("nueva_titulo"))}</h3></div>
      <div class="cuerpo-panel">${aviso}${formularioNotificacion(f, datos, envio, t, locale)}</div></section>
    <section class="panel cronos-panel" data-cronos-historial aria-labelledby="cronos-mis-notificaciones">${contenidoHistorial}</section>
  </section>`;
}

function estadoError(error) {
  if (error instanceof ErrorClienteNotificacionesCronos) {
    if (error.codigo === "sin_empleado") return "sin_empleado";
    if (["acceso_denegado", "autenticacion_requerida", "no_competente"].includes(error.codigo)) return "denegado";
  }
  return "error";
}

export function montarNotificacionesPropiasCronos({ raiz, cliente = crearClienteNotificacionesCronosHTTP(), mensajes, mensajesHistorial, tamanoPagina = 10, anunciar = () => {}, registrarDesmontar,
  locale = LOCALIZACION_ACTUAL, zonaHoraria = "Europe/Madrid", cripto = globalThis.crypto, ahora = () => new Date() } = {}) {
  if (!raiz?.append || !raiz.ownerDocument?.createElement || typeof cliente?.consultarPropias !== "function" || typeof cliente?.enviar !== "function"
    || typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function") || typeof ahora !== "function") {
    throw new TypeError("montaje de las notificaciones de Cronos no disponible");
  }
  const t = crearTraductorNotificacionesCronos(mensajes);
  const th = crearTraductorNotificacionesHistorialCronos(mensajesHistorial);
  paginaHistorial([], {}, tamanoPagina);
  const contenedor = raiz.ownerDocument.createElement("section"); contenedor.dataset.cronosNotificacionesPropias = ""; raiz.append(contenedor);
  let activa = true; let secuencia = 0; let controlador = null; let peticion = null; let lecturaDocumento = 0;
  let estado = "cargando"; let datos = null; let mensaje = ""; let tonoMensaje = "exito";
  let formulario = formularioVacio(hoyCivilCronos(ahora(), zonaHoraria));
  // Un reintento del mismo contenido conserva la clave; otro contenido, otra.
  let envio = null;
  let historial = { filtro: "", pagina: 1 }; let estadoHistorial = "listo";
  const dibujar = () => {
    if (activa) contenedor.innerHTML = renderizarNotificacionesPropiasCronos({ estado, datos, formulario, envio, mensaje, tonoMensaje, mensajes, mensajesHistorial, historial, estadoHistorial, tamanoPagina, locale, zonaHoraria });
  };
  const dibujarHistorial = (foco = "") => {
    if (!activa) return;
    const region = contenedor.querySelector?.("[data-cronos-historial]");
    if (region) region.innerHTML = renderizarHistorialNotificacionesCronos({ datos, historial, estado: estadoHistorial, mensajes, mensajesHistorial, tamanoPagina, locale, zonaHoraria });
    else dibujar();
    if (foco) contenedor.querySelector?.(foco)?.focus?.();
  };
  const cargar = async ({ recuperarFoco = false, conservarFormulario = true } = {}) => {
    if (!activa) return;
    const parcial = conservarFormulario && estado === "listo";
    controlador?.abort(); controlador = new AbortController(); const signal = controlador.signal; const turno = ++secuencia;
    estadoHistorial = "cargando";
    if (parcial) dibujarHistorial(recuperarFoco ? "[data-cronos-historial-resumen]" : "");
    else { estado = "cargando"; datos = null; dibujar(); }
    const enfocarHistorial = () => {
      const documento = contenedor.ownerDocument;
      const region = contenedor.querySelector?.("[data-cronos-historial]");
      // Si la persona siguió escribiendo durante la consulta, conservar su foco.
      return recuperarFoco && (!documento?.activeElement || documento.activeElement === documento.body || region?.contains?.(documento.activeElement));
    };
    try {
      const r = await cliente.consultarPropias({ signal });
      if (!activa || turno !== secuencia || signal.aborted) return;
      const recuperarHistorial = parcial && enfocarHistorial();
      estado = "listo"; estadoHistorial = "listo"; datos = r;
      historial = { ...historial, pagina: paginaHistorial(r.notificaciones, historial, tamanoPagina).pagina };
      if (formulario.tipo && !r.tipos.some((tipo) => tipo.tipo_version_ref === formulario.tipo)) formulario = { ...formulario, tipo: "" };
      if (parcial) {
        // Sólo el catálogo del selector cambia; el fichero y su huella siguen juntos.
        const tipo = contenedor.querySelector?.('[data-cronos-notificacion-formulario] [name="tipo"]');
        if (tipo) tipo.innerHTML = `<option value="">${escaparHTML(t("campo_tipo_elegir"))}</option>` + r.tipos.map((item) => `<option value="${escaparHTML(item.tipo_version_ref)}"${formulario.tipo === item.tipo_version_ref ? " selected" : ""}>${escaparHTML(item.nombre)}</option>`).join("");
        if (!tipo && r.tipos.length) {
          // Si antes no había tipos, sustituir sólo ese aviso por el formulario.
          // Un formulario existente conserva sus nodos, el borrador y el fichero.
          const sinTipos = contenedor.querySelector?.("[data-cronos-notificacion-sin-tipos]");
          if (sinTipos) sinTipos.outerHTML = formularioNotificacion(formulario, datos, envio, t, locale);
        }
        dibujarHistorial(recuperarHistorial ? "[data-cronos-historial-actualizar]" : "");
        anunciar(th("actualizado"));
      } else {
        dibujar();
        if (recuperarFoco) contenedor.querySelector?.(mensaje ? "[data-cronos-notificacion-mensaje]" : "[data-cronos-notificacion-formulario] [name=tipo]")?.focus?.();
      }
    } catch (error) {
      if (!activa || turno !== secuencia || signal.aborted) return;
      const errorEstado = estadoError(error);
      if (parcial && errorEstado === "error") {
        const recuperarHistorial = enfocarHistorial();
        estadoHistorial = "error"; dibujarHistorial(recuperarHistorial ? "[data-cronos-historial-actualizar]" : ""); anunciar(th("error_consulta"));
      } else {
        estado = errorEstado; datos = null; dibujar(); anunciar(t(estado));
        if (recuperarFoco) contenedor.querySelector?.("[data-cronos-notificacion-reintentar]")?.focus?.();
      }
    }
  };
  const alConsultar = (evento) => {
    if (!activa) return;
    if (estado === "error" && evento.target?.closest?.("[data-cronos-notificacion-reintentar]")) return cargar({ recuperarFoco: true });
    if (estado !== "listo") return;
    if (evento.target?.closest?.("[data-cronos-historial-actualizar]") && estadoHistorial !== "cargando") return cargar({ recuperarFoco: true });
    if (estadoHistorial !== "listo") return;
    if (evento.target?.closest?.("[data-cronos-historial-quitar]")) {
      historial = { filtro: "", pagina: 1 }; dibujarHistorial("[data-cronos-historial-estado]"); return;
    }
    const boton = evento.target?.closest?.("[data-cronos-historial-pagina]");
    if (!boton || boton.disabled) return;
    const pagina = Number(boton.dataset?.cronosHistorialPagina);
    const p = paginaHistorial(datos.notificaciones, historial, tamanoPagina);
    if (!Number.isSafeInteger(pagina) || pagina < 1 || pagina > p.paginas) return;
    historial = { ...historial, pagina }; dibujarHistorial("[data-cronos-historial-resumen]");
  };
  const estadoDocumento = (clave) => {
    formulario = { ...formulario, documento: clave };
    const nodo = contenedor.querySelector?.("[data-cronos-documento-estado]");
    if (nodo) { nodo.textContent = clave ? t(clave) : ""; nodo.dataset.tono = clave === "documento_error" ? "error" : "exito"; }
  };
  // El fichero se lee una sola vez, en «change»; «input» también se dispara
  // en un input type=file y no debe volver a leerlo.
  const alCambiarDocumento = async (control) => {
    const lectura = ++lecturaDocumento;
    formulario = { ...formulario, huella: "" };
    const archivo = control.files?.length === 1 ? control.files[0] : null;
    if (!archivo) { estadoDocumento(""); return; }
    estadoDocumento("documento_calculando");
    try {
      const huella = await calcularHuellaDocumentoCronos(archivo, cripto);
      if (!activa || lectura !== lecturaDocumento) return;
      formulario = { ...formulario, huella };
      estadoDocumento("documento_listo");
    } catch {
      if (!activa || lectura !== lecturaDocumento) return;
      estadoDocumento("documento_error");
    }
  };
  // El envío se refleja sobre el formulario ya pintado, sin recrearlo: el
  // fichero elegido sigue en su campo junto a la huella que se conserva.
  // Tras un error, el foco vuelve al campo que lo causa o al botón.
  const reflejarEnvio = (foco = "") => {
    const form = contenedor.querySelector?.("[data-cronos-notificacion-formulario]");
    const aviso = form?.querySelector?.("[data-cronos-envio-aviso]");
    const boton = form?.querySelector?.("[data-cronos-notificacion-enviar]");
    if (!form?.elements || !aviso || !boton) { dibujar(); } else {
      const enviando = envio?.enviando === true;
      for (const control of Array.from(form.elements)) control.disabled = enviando;
      boton.textContent = t(enviando ? "enviando" : "enviar");
      aviso.innerHTML = avisoEnvio(envio?.mensaje);
    }
    if (!foco) return;
    const destino = foco === "enviar" ? "[data-cronos-notificacion-enviar]" : `[data-cronos-notificacion-formulario] [name="${foco}"]`;
    contenedor.querySelector?.(destino)?.focus?.();
  };
  const campoEnError = (clave) => {
    if (clave === "error_datos") return !formulario.tipo ? "tipo" : !/^\d{4}-\d{2}-\d{2}$/u.test(formulario.fecha) ? "fecha" : "texto";
    return formulario.referencia ? "documento" : "referencia";
  };
  const alCambiar = (evento) => {
    const control = evento.target;
    if (!control || typeof control.name !== "string") return;
    if (control.name === "documento") {
      return evento.type === "input" ? undefined : alCambiarDocumento(control);
    }
    if (["tipo", "fecha", "texto", "referencia"].includes(control.name)) {
      formulario = { ...formulario, [control.name]: String(control.value ?? "") };
      if (control.name === "texto") {
        const cuenta = contenedor.querySelector?.("[data-cronos-cuenta]");
        if (cuenta) cuenta.textContent = t("campo_texto_cuenta", { usados: numeroVisibleCronos([...formulario.texto].length, locale), maximo: numeroVisibleCronos(MAXIMO_TEXTO_NOTIFICACION_CRONOS, locale) });
      }
    }
  };
  const alEnviar = async (evento) => {
    if (activa && evento.target?.matches?.("[data-cronos-historial-filtros]")) {
      evento.preventDefault();
      if (estado !== "listo" || estadoHistorial !== "listo") return;
      const filtro = String(evento.target.elements?.namedItem?.("estado_historial")?.value ?? "");
      if (!["", "registrada", "atendida"].includes(filtro)) return;
      historial = { filtro, pagina: 1 }; dibujarHistorial("[data-cronos-historial-estado]"); return;
    }
    if (!activa || estado !== "listo" || !evento.target?.matches?.("[data-cronos-notificacion-formulario]") || envio?.enviando) return;
    evento.preventDefault();
    const elementos = evento.target.elements;
    const valor = (nombre, previo) => String(elementos?.namedItem?.(nombre)?.value ?? previo);
    formulario = { ...formulario, tipo: valor("tipo", formulario.tipo), fecha: valor("fecha", formulario.fecha), texto: valor("texto", formulario.texto),
      referencia: valor("referencia", formulario.referencia).trim() };
    const errorLocal = !formulario.tipo || !/^\d{4}-\d{2}-\d{2}$/u.test(formulario.fecha) || !textoNotificacionValido(formulario.texto) ? "error_datos"
      : formulario.documento === "documento_calculando" || !adjuntoNotificacionValido(formulario.referencia, formulario.huella) ? "error_documento" : "";
    if (errorLocal) {
      envio = { ...(envio ?? {}), enviando: false, mensaje: t(errorLocal) }; reflejarEnvio(campoEnError(errorLocal)); anunciar(envio.mensaje);
      return;
    }
    const entrada = { tipo_version_ref: formulario.tipo, fecha_referida: formulario.fecha, texto: formulario.texto,
      ...(formulario.referencia ? { adjunto_ref: formulario.referencia, adjunto_sha256: formulario.huella } : {}) };
    const firma = JSON.stringify(entrada);
    const clave = envio?.firma === firma && envio.clave ? envio.clave : claveNueva();
    envio = { clave, firma, enviando: true, mensaje: "" }; mensaje = "";
    contenedor.querySelector?.("[data-cronos-notificacion-mensaje]")?.remove?.();
    reflejarEnvio();
    peticion = new AbortController();
    try {
      const recibo = await cliente.enviar({ clave_operacion: clave, ...entrada }, { signal: peticion.signal });
      if (!activa) return;
      envio = null; formulario = formularioVacio(hoyCivilCronos(ahora(), zonaHoraria));
      mensaje = t(recibo.replay ? "ya_enviada" : "enviada"); tonoMensaje = "exito"; anunciar(mensaje);
      await cargar({ recuperarFoco: true, conservarFormulario: false });
    } catch (error) {
      if (!activa || peticion.signal.aborted) return;
      const codigo = error instanceof ErrorClienteNotificacionesCronos ? error.codigo : error instanceof TypeError ? "peticion_invalida" : "";
      const texto = t(ERRORES_ENVIO.get(codigo) ?? "error_envio");
      anunciar(texto);
      if (codigo === "tipo_no_vigente") {
        // La lista se repinta: el campo del fichero vuelve vacío, así que
        // tampoco se conserva su huella.
        envio = null; mensaje = texto; tonoMensaje = "error"; formulario = { ...formulario, tipo: "", huella: "", documento: "" };
        await cargar({ conservarFormulario: false }); contenedor.querySelector?.('[data-cronos-notificacion-formulario] [name="tipo"]')?.focus?.(); return;
      }
      envio = { ...envio, enviando: false, mensaje: texto };
      reflejarEnvio(codigo === "peticion_invalida" ? "tipo" : "enviar");
    }
  };
  contenedor.addEventListener("change", alCambiar); contenedor.addEventListener("input", alCambiar); contenedor.addEventListener("submit", alEnviar);
  contenedor.addEventListener("click", alConsultar);
  void cargar();
  const desmontar = () => {
    if (!activa) return;
    activa = false; ++secuencia; ++lecturaDocumento; controlador?.abort(); peticion?.abort();
    contenedor.removeEventListener("change", alCambiar); contenedor.removeEventListener("input", alCambiar); contenedor.removeEventListener("submit", alEnviar);
    contenedor.removeEventListener("click", alConsultar);
    contenedor.remove?.();
  };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ desmontar, recargar: cargar });
}
