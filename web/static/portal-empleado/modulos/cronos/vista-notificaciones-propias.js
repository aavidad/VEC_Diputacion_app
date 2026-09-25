import {
  crearTraductorNotificacionesCronos, fechaCivilVisibleCronos, instanteVisibleCronos, numeroVisibleCronos,
} from "./i18n-notificaciones.js";
import {
  ErrorClienteNotificacionesCronos, MAXIMO_TEXTO_NOTIFICACION_CRONOS, adjuntoNotificacionValido, calcularHuellaDocumentoCronos,
  crearClienteNotificacionesCronosHTTP, textoNotificacionValido,
} from "./cliente-notificaciones-http.js";

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
  const partes = Object.fromEntries(new Intl.DateTimeFormat("en-CA", { timeZone: zonaHoraria, year: "numeric", month: "2-digit", day: "2-digit" })
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
  const aviso = envio?.mensaje ? `<p class="cronos-solicitud-aviso" data-tono="error" role="alert">${escaparHTML(envio.mensaje)}</p>` : "";
  if (!datos.tipos.length) return `<p class="cronos-vacio" role="status">${escaparHTML(t("sin_tipos"))}</p>`;
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
    <div class="cronos-solicitud-acciones"><button type="submit" class="boton-primario"${inactivo}>${escaparHTML(t(enviando ? "enviando" : "enviar"))}</button></div>${aviso}
  </form>`;
}

function filaNotificacion(n, t, locale, zonaHoraria) {
  const estado = n.estado === "atendida" ? t("estado_atendida", { fecha: instanteVisibleCronos(n.atendida_en, locale, zonaHoraria) }) : t("estado_registrada");
  const documento = n.adjunto_ref
    ? `<span title="${escaparHTML(t("huella_documento", { huella: n.adjunto_sha256 }))}">${escaparHTML(n.adjunto_ref)}</span>` : escaparHTML(t("sin_documento"));
  return `<tr><td>${escaparHTML(instanteVisibleCronos(n.registrada_en, locale, zonaHoraria))}</td><th scope="row">${escaparHTML(n.tipo_nombre)}</th>
    <td>${escaparHTML(fechaCivilVisibleCronos(n.fecha_referida, locale))}</td><td class="cronos-notificacion-texto">${escaparHTML(n.texto)}</td>
    <td>${documento}</td><td><span class="cronos-estado" data-estado="${escaparHTML(n.estado)}">${escaparHTML(estado)}</span></td></tr>`;
}

/** Notificaciones propias a RRHH: formulario de envío y lista con su estado. */
export function renderizarNotificacionesPropiasCronos({ estado = "cargando", datos = null, formulario = null, envio = null, mensaje = "", tonoMensaje = "exito",
  mensajes, locale = "es-ES", zonaHoraria = "Europe/Madrid", hoy = hoyCivilCronos(new Date(), zonaHoraria) } = {}) {
  const t = crearTraductorNotificacionesCronos(mensajes);
  const ayuda = t("abrir_ayuda", { asunto: t("notificaciones_titulo") });
  const cabecera = `<header class="cronos-encabezado"><div><p class="sobrelinea">${escaparHTML(t("sobrelinea"))}</p><h2 id="cronos-notificaciones-propias-titulo">${escaparHTML(t("notificaciones_titulo"))}</h2></div>
    <button type="button" class="cronos-boton-ayuda" data-accion="ayuda" aria-label="${escaparHTML(ayuda)}" title="${escaparHTML(ayuda)}"><span aria-hidden="true">?</span></button></header>`;
  if (estado !== "listo") {
    const clave = { denegado: "denegado", sin_empleado: "sin_empleado", error: "error" }[estado] ?? "cargando";
    return `<section class="cronos-area cronos-notificaciones-propias" aria-labelledby="cronos-notificaciones-propias-titulo" data-estado="${escaparHTML(estado)}">${cabecera}
      <section class="panel cronos-panel"><div class="cuerpo-panel"><p class="cronos-${estado === "cargando" ? "vacio" : "acceso-denegado"}" role="${estado === "error" ? "alert" : "status"}">${escaparHTML(t(clave))}</p></div></section></section>`;
  }
  const tono = tonoMensaje === "error" ? "error" : "exito";
  const aviso = mensaje ? `<p class="cronos-solicitud-aviso" data-tono="${tono}" role="${tono === "error" ? "alert" : "status"}">${escaparHTML(mensaje)}</p>` : "";
  const f = formulario ?? formularioVacio(hoy);
  const cabeceras = ["col_enviada", "col_tipo", "col_fecha", "col_mensaje", "col_documento", "col_estado"];
  const filas = datos.notificaciones.map((n) => filaNotificacion(n, t, locale, zonaHoraria)).join("");
  const cuerpo = filas || `<tr><td colspan="${cabeceras.length}">${escaparHTML(t("sin_notificaciones"))}</td></tr>`;
  return `<section class="cronos-area cronos-notificaciones-propias" aria-labelledby="cronos-notificaciones-propias-titulo" data-estado="listo">${cabecera}
    <section class="panel cronos-panel" aria-labelledby="cronos-notificacion-nueva"><div class="cabecera-panel"><h3 id="cronos-notificacion-nueva">${escaparHTML(t("nueva_titulo"))}</h3></div>
      <div class="cuerpo-panel">${aviso}${formularioNotificacion(f, datos, envio, t, locale)}</div></section>
    <section class="panel cronos-panel" aria-labelledby="cronos-mis-notificaciones"><div class="cabecera-panel"><h3 id="cronos-mis-notificaciones">${escaparHTML(t("mis_notificaciones"))}</h3></div>
      <div class="cronos-tabla-contenedor"><table class="cronos-tabla"><thead><tr>${cabeceras.map((c) => `<th scope="col">${escaparHTML(t(c))}</th>`).join("")}</tr></thead><tbody>${cuerpo}</tbody></table></div></section>
  </section>`;
}

function estadoError(error) {
  if (error instanceof ErrorClienteNotificacionesCronos) {
    if (error.codigo === "sin_empleado") return "sin_empleado";
    if (["acceso_denegado", "autenticacion_requerida", "no_competente"].includes(error.codigo)) return "denegado";
  }
  return "error";
}

export function montarNotificacionesPropiasCronos({ raiz, cliente = crearClienteNotificacionesCronosHTTP(), mensajes, anunciar = () => {}, registrarDesmontar,
  locale = "es-ES", zonaHoraria = "Europe/Madrid", cripto = globalThis.crypto, ahora = () => new Date() } = {}) {
  if (!raiz?.append || !raiz.ownerDocument?.createElement || typeof cliente?.consultarPropias !== "function" || typeof cliente?.enviar !== "function"
    || typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function") || typeof ahora !== "function") {
    throw new TypeError("montaje de las notificaciones de Cronos no disponible");
  }
  const t = crearTraductorNotificacionesCronos(mensajes);
  const contenedor = raiz.ownerDocument.createElement("section"); contenedor.dataset.cronosNotificacionesPropias = ""; raiz.append(contenedor);
  let activa = true; let secuencia = 0; let controlador = null; let peticion = null; let lecturaDocumento = 0;
  let estado = "cargando"; let datos = null; let mensaje = ""; let tonoMensaje = "exito";
  let formulario = formularioVacio(hoyCivilCronos(ahora(), zonaHoraria));
  // Un reintento del mismo contenido conserva la clave; otro contenido, otra.
  let envio = null;
  const dibujar = () => {
    if (activa) contenedor.innerHTML = renderizarNotificacionesPropiasCronos({ estado, datos, formulario, envio, mensaje, tonoMensaje, mensajes, locale, zonaHoraria });
  };
  const cargar = async () => {
    controlador?.abort(); controlador = new AbortController(); const turno = ++secuencia;
    estado = "cargando"; datos = null; dibujar();
    try {
      const r = await cliente.consultarPropias({ signal: controlador.signal });
      if (!activa || turno !== secuencia) return;
      estado = "listo"; datos = r;
      if (formulario.tipo && !r.tipos.some((tipo) => tipo.tipo_version_ref === formulario.tipo)) formulario = { ...formulario, tipo: "" };
      dibujar();
    } catch (error) {
      if (!activa || turno !== secuencia || controlador.signal.aborted) return;
      estado = estadoError(error); dibujar(); anunciar(t(estado));
    }
  };
  const estadoDocumento = (clave) => {
    formulario = { ...formulario, documento: clave };
    const nodo = contenedor.querySelector?.("[data-cronos-documento-estado]");
    if (nodo) { nodo.textContent = clave ? t(clave) : ""; nodo.dataset.tono = clave === "documento_error" ? "error" : "exito"; }
  };
  const alCambiar = async (evento) => {
    const control = evento.target;
    if (!control || typeof control.name !== "string") return;
    if (control.name === "documento") {
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
      return;
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
    if (!evento.target?.matches?.("[data-cronos-notificacion-formulario]") || envio?.enviando) return;
    evento.preventDefault();
    const elementos = evento.target.elements;
    const valor = (nombre, previo) => String(elementos?.namedItem?.(nombre)?.value ?? previo);
    formulario = { ...formulario, tipo: valor("tipo", formulario.tipo), fecha: valor("fecha", formulario.fecha), texto: valor("texto", formulario.texto),
      referencia: valor("referencia", formulario.referencia).trim() };
    const errorLocal = !formulario.tipo || !/^\d{4}-\d{2}-\d{2}$/u.test(formulario.fecha) || !textoNotificacionValido(formulario.texto) ? "error_datos"
      : formulario.documento === "documento_calculando" || !adjuntoNotificacionValido(formulario.referencia, formulario.huella) ? "error_documento" : "";
    if (errorLocal) {
      envio = { ...(envio ?? {}), enviando: false, mensaje: t(errorLocal) }; dibujar(); anunciar(envio.mensaje);
      return;
    }
    const entrada = { tipo_version_ref: formulario.tipo, fecha_referida: formulario.fecha, texto: formulario.texto,
      ...(formulario.referencia ? { adjunto_ref: formulario.referencia, adjunto_sha256: formulario.huella } : {}) };
    const firma = JSON.stringify(entrada);
    const clave = envio?.firma === firma && envio.clave ? envio.clave : claveNueva();
    envio = { clave, firma, enviando: true, mensaje: "" }; mensaje = ""; dibujar();
    peticion = new AbortController();
    try {
      const recibo = await cliente.enviar({ clave_operacion: clave, ...entrada }, { signal: peticion.signal });
      if (!activa) return;
      envio = null; formulario = formularioVacio(hoyCivilCronos(ahora(), zonaHoraria));
      mensaje = t(recibo.replay ? "ya_enviada" : "enviada"); tonoMensaje = "exito"; anunciar(mensaje);
      await cargar();
    } catch (error) {
      if (!activa || peticion.signal.aborted) return;
      const codigo = error instanceof ErrorClienteNotificacionesCronos ? error.codigo : error instanceof TypeError ? "peticion_invalida" : "";
      const texto = t(ERRORES_ENVIO.get(codigo) ?? "error_envio");
      anunciar(texto);
      if (codigo === "tipo_no_vigente") { envio = null; mensaje = texto; tonoMensaje = "error"; await cargar(); return; }
      envio = { ...envio, enviando: false, mensaje: texto };
      dibujar();
    }
  };
  contenedor.addEventListener("change", alCambiar); contenedor.addEventListener("input", alCambiar); contenedor.addEventListener("submit", alEnviar);
  void cargar();
  const desmontar = () => {
    if (!activa) return;
    activa = false; ++secuencia; ++lecturaDocumento; controlador?.abort(); peticion?.abort();
    contenedor.removeEventListener("change", alCambiar); contenedor.removeEventListener("input", alCambiar); contenedor.removeEventListener("submit", alEnviar);
    contenedor.remove?.();
  };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ desmontar, recargar: cargar });
}
