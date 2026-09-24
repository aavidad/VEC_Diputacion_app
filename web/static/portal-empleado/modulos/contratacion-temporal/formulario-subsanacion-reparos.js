import { MAXIMO_ARCHIVO_RECUPERACION_SUBSANACION, serializarDatosRecuperacionSubsanacion, validarDatosRecuperacionSubsanacion, validarReciboSubsanacionReparos, validarSolicitudSubsanacionReparos, validarVersionesRecuperacionSubsanacion } from "./cliente-http-subsanacion-reparos.js";

const REF = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
const escapar = (v) => String(v ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#039;");
function contextoValido(c) { return c && Object.getPrototypeOf(c) === Object.prototype && Object.keys(c).length === 2 && REF.test(c.expediente_ref) && Number.isSafeInteger(c.version_esperada) && c.version_esperada > 0; }
function indeterminado(error) { try { return error?.resultadoIndeterminado !== false; } catch { return true; } }
function intencionValida(valor, contexto, versionesRecuperacionAnteriores) {
  if (!valor || Object.getPrototypeOf(valor) !== Object.prototype
    || Object.keys(valor).length !== 2 || !Object.hasOwn(valor, "solicitud")
    || !Object.hasOwn(valor, "incierta") || typeof valor.incierta !== "boolean") throw new TypeError("intención de subsanación no válida");
  const solicitud = validarSolicitudSubsanacionReparos(valor.solicitud);
  if (solicitud.expediente_ref !== contexto.expediente_ref
    || (solicitud.version_esperada !== contexto.version_esperada
      && (!valor.incierta || !versionesRecuperacionAnteriores.includes(solicitud.version_esperada)))) throw new TypeError("intención ajena al expediente de subsanación");
  return Object.freeze({ solicitud, incierta: valor.incierta });
}

export function montarFormularioSubsanacionReparos({ raiz, cliente, contexto, reciboConfirmado = null, intencionInicial = null, soloImportar = false, versionesRecuperacionAnteriores = [], traducir, confirmarOperacion = () => false, generarClaveIdempotencia = () => globalThis.crypto?.randomUUID?.(), anunciar = () => {}, alConfirmar = () => {}, alCambiarIntencion, alDenegacion } = {}) {
  if (!raiz || typeof raiz.addEventListener !== "function" || typeof raiz.removeEventListener !== "function" || typeof raiz.contains !== "function" || typeof raiz.querySelector !== "function" || typeof raiz.replaceChildren !== "function" || !contextoValido(contexto) || typeof soloImportar !== "boolean" || typeof cliente?.registrarSubsanacionReparos !== "function" || typeof traducir !== "function" || typeof confirmarOperacion !== "function" || typeof generarClaveIdempotencia !== "function" || typeof anunciar !== "function" || typeof alConfirmar !== "function" || typeof alCambiarIntencion !== "function" || typeof alDenegacion !== "function") throw new TypeError("dependencias de subsanación no válidas");
  contexto = Object.freeze({ expediente_ref: contexto.expediente_ref, version_esperada: contexto.version_esperada });
  versionesRecuperacionAnteriores = validarVersionesRecuperacionSubsanacion(contexto, versionesRecuperacionAnteriores);
  const intencion = intencionInicial === null ? null : intencionValida(intencionInicial, contexto, versionesRecuperacionAnteriores);
  let reciboInicial = null;
  if (reciboConfirmado !== null) {
    if (!reciboConfirmado || Object.getPrototypeOf(reciboConfirmado) !== Object.prototype
      || !contextoValido(reciboConfirmado.contexto)) throw new TypeError("recibo confirmado de subsanación no válido");
    reciboInicial = validarReciboSubsanacionReparos(reciboConfirmado.recibo, reciboConfirmado.contexto);
    if (reciboInicial.expediente_ref !== contexto.expediente_ref
      || (![reciboConfirmado.contexto.version_esperada, reciboInicial.version_resultante].includes(contexto.version_esperada)
        && !versionesRecuperacionAnteriores.includes(reciboConfirmado.contexto.version_esperada))) throw new TypeError("recibo ajeno al contexto de subsanación");
  }
  if (reciboInicial && intencion) throw new TypeError("recibo e intención pendientes simultáneos");
  let montado = true, controlador = null, solicitud = intencion?.solicitud ?? null;
  const urls = new Set();
  const estado = { observaciones: "", recibo: reciboInicial, ocupado: false, incierta: intencion?.incierta ?? false, denegado: false, descargaIniciada: false, mensaje: reciboInicial ? "subsanacion_confirmada" : intencion?.incierta ? "subsanacion_indeterminada" : intencion ? "subsanacion_preparada" : soloImportar ? "subsanacion_importacion_lista" : "subsanacion_lista", tono: reciboInicial ? "exito" : intencion?.incierta ? "error" : "informacion" };
  function contenidoPendiente(t) {
    return `<section class="ct-alcance" data-ct-subsanacion-indeterminada aria-labelledby="ct-subsanacion-recuperacion-titulo"><h3 id="ct-subsanacion-recuperacion-titulo">${escapar(t(estado.incierta ? "subsanacion_recuperacion_titulo" : "subsanacion_preparacion_titulo"))}</h3><p>${escapar(t(estado.incierta ? "subsanacion_recuperacion_ayuda" : "subsanacion_preparacion_ayuda"))}</p><p><strong>${escapar(t("subsanacion_contenido_original"))}</strong></p><blockquote>${escapar(solicitud.observaciones)}</blockquote>${solicitud.version_esperada !== contexto.version_esperada ? `<p class="ct-ayuda">${escapar(t("subsanacion_version_original"))}: ${solicitud.version_esperada}</p>` : ""}<p class="ct-ayuda">${escapar(t("subsanacion_archivo_advertencia"))}</p><div class="ct-acciones"><button class="boton-secundario" type="button" data-ct-subsanacion-guardar ${estado.ocupado ? "disabled" : ""}>${escapar(t("subsanacion_guardar_archivo"))}</button><button class="boton-primario" type="button" ${estado.incierta ? "data-ct-subsanacion-recuperar" : "data-ct-subsanacion-enviar"} ${estado.ocupado || (!estado.incierta && !estado.descargaIniciada) ? "disabled" : ""}>${escapar(t(estado.incierta ? "subsanacion_recuperar" : "subsanacion_enviar_preparada"))}</button></div></section>`;
  }
  function contenidoInicial(t) {
    const alta = soloImportar ? "" : `<form data-ct-subsanacion-form novalidate><label class="ct-campo" for="ct-subsanacion-observaciones"><span>${escapar(t("subsanacion_observaciones"))}</span><textarea id="ct-subsanacion-observaciones" name="observaciones" rows="5" maxlength="2000" required ${estado.ocupado ? "disabled" : ""}>${escapar(estado.observaciones)}</textarea><small>${escapar(t("subsanacion_ayuda"))}</small></label><button class="boton-primario" type="submit" ${estado.ocupado ? "disabled" : ""}>${escapar(t("subsanacion_confirmar"))}</button></form>`;
    return `${alta}<section class="ct-alcance" data-ct-subsanacion-importacion aria-labelledby="ct-subsanacion-importacion-titulo"><h3 id="ct-subsanacion-importacion-titulo">${escapar(t("subsanacion_importacion_titulo"))}</h3><p>${escapar(t("subsanacion_importacion_ayuda"))}</p><label for="ct-subsanacion-archivo">${escapar(t("subsanacion_archivo"))}</label><input id="ct-subsanacion-archivo" data-ct-subsanacion-archivo type="file" accept="application/json,.json" ${estado.ocupado ? "disabled" : ""}></section>`;
  }
  function pintar() {
    if (!montado) return;
    const t = traducir;
    if (estado.denegado) {
      raiz.innerHTML = `<section class="ct-alta" data-ct-subsanacion-denegada><h2>${escapar(t("subsanacion_denegada_titulo"))}</h2><p role="alert">${escapar(t("subsanacion_sin_permiso"))}</p></section>`;
      return;
    }
    const contenido = estado.recibo ? `<section class="ct-recibo" data-ct-subsanacion-recibo role="status" tabindex="-1"><h3>${escapar(t("subsanacion_recibo"))}</h3><dl><div><dt>${escapar(t("subsanacion_expediente"))}</dt><dd><code>${escapar(estado.recibo.expediente_ref)}</code></dd></div><div><dt>${escapar(t("subsanacion_version"))}</dt><dd>${estado.recibo.version_resultante}</dd></div><div><dt>${escapar(t("subsanacion_circuito"))}</dt><dd>${escapar(t("subsanacion_fase_unidad"))} · ${escapar(t("subsanacion_estado_incidencia"))}</dd></div><div><dt>${escapar(t("subsanacion_referencia"))}</dt><dd><code>${escapar(estado.recibo.recibo_ref)}</code></dd></div><div><dt>${escapar(t("subsanacion_fecha"))}</dt><dd><time datetime="${escapar(estado.recibo.registrada_en)}">${escapar(estado.recibo.registrada_en)}</time></dd></div></dl></section>` : solicitud ? contenidoPendiente(t) : contenidoInicial(t);
    raiz.innerHTML = `<section class="ct-alta" data-ct-subsanacion aria-labelledby="ct-subsanacion-titulo"><header class="ct-cabecera"><div><h2 id="ct-subsanacion-titulo">${escapar(t("subsanacion_titulo"))}</h2></div></header><dl class="ct-resumen"><div><dt>${escapar(t("subsanacion_expediente"))}</dt><dd><code>${escapar(contexto.expediente_ref)}</code></dd></div><div><dt>${escapar(t("subsanacion_version_actual"))}</dt><dd>${contexto.version_esperada}</dd></div></dl><p data-ct-subsanacion-estado role="${estado.tono === "error" ? "alert" : "status"}" aria-live="polite">${escapar(t(estado.mensaje))}</p>${contenido}</section>`;
  }
  function destinoFocoDisponible() {
    const documento = raiz.ownerDocument;
    const activo = documento?.activeElement;
    // Si la persona cambió a otro control durante la petición, respetar su foco.
    return !documento || !activo || activo === documento.body;
  }
  function conservarIntencion(incierta) {
    try { return alCambiarIntencion(Object.freeze({ solicitud, incierta })) === true; }
    catch { return false; }
  }
  function descargarArchivo() {
    if (!montado || estado.denegado || estado.ocupado || !solicitud || estado.recibo) return;
    let url = null;
    try {
      const texto = serializarDatosRecuperacionSubsanacion(solicitud);
      const documento = raiz.ownerDocument;
      const enlace = documento?.createElement?.("a");
      if (typeof Blob !== "function" || !globalThis.URL?.createObjectURL || !enlace || !documento.body?.append) throw new Error("descarga no disponible");
      url = URL.createObjectURL(new Blob([texto], { type: "application/json" }));
      urls.add(url);
      enlace.href = url;
      enlace.download = "recuperacion-subsanacion-v1.json";
      enlace.hidden = true;
      documento.body.append(enlace);
      try { enlace.click(); } finally { enlace.remove(); }
      estado.descargaIniciada = true;
      estado.mensaje = "subsanacion_descarga_iniciada";
      estado.tono = "informacion";
    } catch {
      estado.mensaje = "subsanacion_descarga_error";
      estado.tono = "error";
    } finally {
      if (url) setTimeout(() => { if (urls.delete(url)) URL.revokeObjectURL(url); }, 0);
      pintar();
    }
  }
  async function importarArchivo(archivo) {
    if (!montado || estado.denegado || estado.ocupado || solicitud || estado.recibo) return;
    if (!archivo || !Number.isSafeInteger(archivo.size) || archivo.size < 1
      || archivo.size > MAXIMO_ARCHIVO_RECUPERACION_SUBSANACION || typeof archivo.text !== "function") {
      estado.mensaje = "subsanacion_archivo_invalido"; estado.tono = "error"; pintar(); return;
    }
    estado.ocupado = true; estado.mensaje = "subsanacion_archivo_validando"; estado.tono = "informacion"; pintar();
    try {
      const texto = await archivo.text();
      if (!montado) return;
      if (new TextEncoder().encode(texto).byteLength > MAXIMO_ARCHIVO_RECUPERACION_SUBSANACION) throw new TypeError("archivo demasiado grande");
      const cargada = validarDatosRecuperacionSubsanacion(JSON.parse(texto), contexto, versionesRecuperacionAnteriores).solicitud;
      if (solicitud) throw new TypeError("intención pendiente");
      solicitud = cargada;
      if (!conservarIntencion(true)) { solicitud = null; throw new TypeError("intención no conservada"); }
      estado.incierta = true;
      estado.mensaje = "subsanacion_archivo_importado"; estado.tono = "informacion";
    } catch {
      if (!montado) return;
      estado.mensaje = "subsanacion_archivo_invalido"; estado.tono = "error";
    } finally {
      if (montado) {
        const enfocar = solicitud && destinoFocoDisponible();
        estado.ocupado = false; pintar();
        if (enfocar) raiz.querySelector("[data-ct-subsanacion-recuperar]")?.focus?.();
      }
    }
  }
  async function ejecutar(solicitudOriginal, recuperacion = false) {
    estado.ocupado=true; estado.mensaje=recuperacion ? "subsanacion_recuperando" : "subsanacion_enviando"; estado.tono="informacion"; controlador=new AbortController(); pintar();
    try {
      const respuesta = await cliente.registrarSubsanacionReparos(solicitudOriginal, { signal: controlador.signal });
      if (!montado) return;
      estado.recibo = validarReciboSubsanacionReparos(respuesta, solicitudOriginal);
      estado.mensaje = "subsanacion_confirmada"; estado.tono = "exito";
      try {
        const contextoOriginal = Object.freeze({ expediente_ref: solicitudOriginal.expediente_ref, version_esperada: solicitudOriginal.version_esperada });
        const actualizacion = alConfirmar(estado.recibo, contextoOriginal);
        if (actualizacion && typeof actualizacion.then === "function") void actualizacion.catch(() => {});
        if (alCambiarIntencion(null) !== true) throw new TypeError("intención confirmada no liberada");
      } catch { estado.mensaje = "subsanacion_actualizacion_pendiente"; estado.tono = "error"; }
    } catch (error) {
      if (error?.estado === 401 || error?.estado === 403) {
        try { alDenegacion(); } catch {}
        solicitud = null; estado.observaciones = "";
        estado.incierta = false; estado.denegado = true;
        estado.mensaje = "subsanacion_sin_permiso"; estado.tono = "error";
        return;
      }
      if (!montado) return;
      estado.incierta = true;
      estado.mensaje = !indeterminado(error) ? "subsanacion_recuperacion_no_confirmada" : "subsanacion_indeterminada";
      estado.tono = "error";
    }
    finally { if(montado){
      const enfocar = destinoFocoDisponible();
      estado.ocupado=false; controlador=null; pintar();
      if (enfocar) raiz.querySelector(estado.recibo ? "[data-ct-subsanacion-recibo]" : "[data-ct-subsanacion-recuperar]")?.focus?.();
      anunciar(traducir(estado.mensaje),estado.tono);
    } }
  }
  function preparar(evento) {
    const form = evento.target?.closest?.("[data-ct-subsanacion-form]"); if (!form || !raiz.contains(form) || soloImportar || estado.denegado || estado.ocupado || solicitud || estado.recibo) return; evento.preventDefault();
    const observaciones = String(form.elements?.namedItem?.("observaciones")?.value ?? "").normalize("NFC").trim();
    estado.observaciones = observaciones;
    try {
      solicitud = validarSolicitudSubsanacionReparos({ expediente_ref: contexto.expediente_ref, version_esperada: contexto.version_esperada, clave_idempotencia: generarClaveIdempotencia(), observaciones });
    } catch {
      solicitud = null; estado.mensaje = "subsanacion_validacion"; estado.tono = "error"; pintar(); return;
    }
    if (!conservarIntencion(false)) {
      solicitud = null; estado.mensaje = "subsanacion_preparacion_error"; estado.tono = "error"; pintar(); return;
    }
    estado.mensaje = "subsanacion_preparada"; estado.tono = "informacion";
    pintar();
  }
  async function enviarPreparada(evento) {
    const control = evento.target?.closest?.("[data-ct-subsanacion-enviar]");
    if (!control || !raiz.contains(control) || !montado || estado.denegado || estado.ocupado || !solicitud || estado.incierta || !estado.descargaIniciada || estado.recibo) return;
    evento.preventDefault();
    let confirmada = false;
    try { confirmada = confirmarOperacion({ titulo: traducir("subsanacion_titulo"), advertencia: traducir("subsanacion_confirmacion"), referencia: contexto.expediente_ref }) === true; } catch {}
    if (!confirmada) return;
    if (!conservarIntencion(true)) { estado.mensaje = "subsanacion_preparacion_error"; estado.tono = "error"; pintar(); return; }
    estado.incierta = true;
    await ejecutar(solicitud);
  }
  async function recuperar(evento) {
    const control = evento.target?.closest?.("[data-ct-subsanacion-recuperar]");
    if (!control || !raiz.contains(control) || !montado || estado.denegado || !estado.incierta || !solicitud || estado.ocupado || estado.recibo) return;
    evento.preventDefault();
    let confirmada = false;
    try { confirmada = confirmarOperacion({ titulo: traducir("subsanacion_recuperacion_titulo"), advertencia: traducir("subsanacion_recuperacion_confirmacion"), referencia: contexto.expediente_ref }) === true; } catch {}
    if (!confirmada) return;
    await ejecutar(solicitud, true);
  }
  function pulsar(evento) {
    if (evento.target?.closest?.("[data-ct-subsanacion-guardar]")) { const control = evento.target.closest("[data-ct-subsanacion-guardar]"); if (raiz.contains(control)) { evento.preventDefault(); descargarArchivo(); } return; }
    if (evento.target?.closest?.("[data-ct-subsanacion-enviar]")) return enviarPreparada(evento);
    return recuperar(evento);
  }
  function cambiar(evento) {
    const control = evento.target?.closest?.("[data-ct-subsanacion-archivo]");
    if (!control || !raiz.contains(control)) return;
    return importarArchivo(control.files?.[0]);
  }
  raiz.addEventListener("submit", preparar); raiz.addEventListener("click", pulsar); raiz.addEventListener("change", cambiar); pintar();
  return () => { if(!montado)return; montado=false; controlador?.abort(); controlador=null; for (const url of urls) URL.revokeObjectURL?.(url); urls.clear(); raiz.removeEventListener("submit",preparar); raiz.removeEventListener("click",pulsar); raiz.removeEventListener("change",cambiar); raiz.replaceChildren(); };
}
