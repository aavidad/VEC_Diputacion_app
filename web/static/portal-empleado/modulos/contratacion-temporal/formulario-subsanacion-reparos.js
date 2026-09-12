import { validarReciboSubsanacionReparos, validarSolicitudSubsanacionReparos } from "./cliente-http-subsanacion-reparos.js";

const REF = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
const escapar = (v) => String(v ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#039;");
function contextoValido(c) { return c && Object.getPrototypeOf(c) === Object.prototype && Object.keys(c).length === 2 && REF.test(c.expediente_ref) && Number.isSafeInteger(c.version_esperada) && c.version_esperada > 0; }
function indeterminado(error) { try { return error?.resultadoIndeterminado !== false; } catch { return true; } }

export function montarFormularioSubsanacionReparos({ raiz, cliente, contexto, reciboConfirmado = null, traducir, confirmarOperacion = () => false, generarClaveIdempotencia = () => globalThis.crypto?.randomUUID?.(), anunciar = () => {}, alConfirmar = () => {} } = {}) {
  if (!raiz || typeof raiz.addEventListener !== "function" || typeof raiz.removeEventListener !== "function" || typeof raiz.contains !== "function" || typeof raiz.querySelector !== "function" || typeof raiz.replaceChildren !== "function" || !contextoValido(contexto) || typeof cliente?.registrarSubsanacionReparos !== "function" || typeof traducir !== "function" || typeof confirmarOperacion !== "function" || typeof generarClaveIdempotencia !== "function" || typeof anunciar !== "function" || typeof alConfirmar !== "function") throw new TypeError("dependencias de subsanación no válidas");
  contexto = Object.freeze({ expediente_ref: contexto.expediente_ref, version_esperada: contexto.version_esperada });
  let reciboInicial = null;
  if (reciboConfirmado !== null) {
    if (!reciboConfirmado || Object.getPrototypeOf(reciboConfirmado) !== Object.prototype
      || !contextoValido(reciboConfirmado.contexto)) throw new TypeError("recibo confirmado de subsanación no válido");
    reciboInicial = validarReciboSubsanacionReparos(reciboConfirmado.recibo, reciboConfirmado.contexto);
    if (reciboInicial.expediente_ref !== contexto.expediente_ref
      || ![reciboConfirmado.contexto.version_esperada, reciboInicial.version_resultante].includes(contexto.version_esperada)) throw new TypeError("recibo ajeno al contexto de subsanación");
  }
  let montado = true, controlador = null, solicitud = null;
  const estado = { observaciones: "", recibo: reciboInicial, ocupado: false, bloqueado: reciboInicial !== null, mensaje: reciboInicial ? "subsanacion_confirmada" : "subsanacion_lista", tono: reciboInicial ? "exito" : "informacion" };
  function pintar() {
    if (!montado) return;
    const t = traducir;
    const contenido = estado.recibo ? `<section class="ct-recibo" data-ct-subsanacion-recibo role="status" tabindex="-1"><h3>${escapar(t("subsanacion_recibo"))}</h3><dl><div><dt>${escapar(t("subsanacion_expediente"))}</dt><dd><code>${escapar(estado.recibo.expediente_ref)}</code></dd></div><div><dt>${escapar(t("subsanacion_version"))}</dt><dd>${estado.recibo.version_resultante}</dd></div><div><dt>${escapar(t("subsanacion_circuito"))}</dt><dd>${escapar(t("subsanacion_fase_unidad"))} · ${escapar(t("subsanacion_estado_incidencia"))}</dd></div><div><dt>${escapar(t("subsanacion_referencia"))}</dt><dd><code>${escapar(estado.recibo.recibo_ref)}</code></dd></div><div><dt>${escapar(t("subsanacion_fecha"))}</dt><dd><time datetime="${escapar(estado.recibo.registrada_en)}">${escapar(estado.recibo.registrada_en)}</time></dd></div></dl></section>` : estado.bloqueado ? "" : `<form data-ct-subsanacion-form novalidate><label class="ct-campo" for="ct-subsanacion-observaciones"><span>${escapar(t("subsanacion_observaciones"))}</span><textarea id="ct-subsanacion-observaciones" name="observaciones" rows="5" maxlength="2000" required ${estado.ocupado ? "disabled" : ""}>${escapar(estado.observaciones)}</textarea><small>${escapar(t("subsanacion_ayuda"))}</small></label><button class="boton-primario" type="submit" ${estado.ocupado ? "disabled" : ""}>${escapar(t("subsanacion_confirmar"))}</button></form>`;
    raiz.innerHTML = `<section class="ct-alta" data-ct-subsanacion aria-labelledby="ct-subsanacion-titulo"><header class="ct-cabecera"><div><p class="sobrelinea">${escapar(t("subsanacion_sobrelinea"))}</p><h2 id="ct-subsanacion-titulo">${escapar(t("subsanacion_titulo"))}</h2><p>${escapar(t("subsanacion_descripcion"))}</p></div></header><dl class="ct-resumen"><div><dt>${escapar(t("subsanacion_expediente"))}</dt><dd><code>${escapar(contexto.expediente_ref)}</code></dd></div><div><dt>${escapar(t("subsanacion_version_actual"))}</dt><dd>${contexto.version_esperada}</dd></div></dl><p data-ct-subsanacion-estado role="${estado.tono === "error" ? "alert" : "status"}" aria-live="polite">${escapar(t(estado.mensaje))}</p>${contenido}</section>`;
  }
  async function enviar(evento) {
    const form = evento.target?.closest?.("[data-ct-subsanacion-form]"); if (!form || !raiz.contains(form) || estado.ocupado || estado.bloqueado || estado.recibo) return; evento.preventDefault();
    const observaciones = String(form.elements?.namedItem?.("observaciones")?.value ?? "").normalize("NFC").trim();
    estado.observaciones = observaciones;
    let q; try { q = solicitud ?? validarSolicitudSubsanacionReparos({ expediente_ref: contexto.expediente_ref, version_esperada: contexto.version_esperada, clave_idempotencia: generarClaveIdempotencia(), observaciones }); } catch { estado.mensaje="subsanacion_validacion"; estado.tono="error"; pintar(); return; }
    if (!confirmarOperacion({ titulo: traducir("subsanacion_titulo"), advertencia: traducir("subsanacion_confirmacion"), referencia: contexto.expediente_ref })) return;
    solicitud=q; estado.observaciones=observaciones; estado.ocupado=true; estado.mensaje="subsanacion_enviando"; estado.tono="informacion"; controlador=new AbortController(); pintar();
    try { const respuesta=await cliente.registrarSubsanacionReparos(q,{signal:controlador.signal}); if(!montado)return; estado.recibo=validarReciboSubsanacionReparos(respuesta,q); estado.mensaje="subsanacion_confirmada"; estado.tono="exito"; }
    catch(error) { if(!montado)return; estado.bloqueado=indeterminado(error); if (!estado.bloqueado) solicitud=null; estado.mensaje=estado.bloqueado?"subsanacion_indeterminada":error?.envelopeValido && error.estado === 403 ? "subsanacion_sin_permiso" : error?.envelopeValido && error.estado === 409 ? "subsanacion_conflicto" : error?.envelopeValido ? "subsanacion_rechazada" : "subsanacion_error"; estado.tono="error"; }
    finally { if(montado){estado.ocupado=false; controlador=null; pintar(); if (estado.recibo) { raiz.querySelector("[data-ct-subsanacion-recibo]")?.focus?.(); void Promise.resolve().then(() => { if (montado) return alConfirmar(estado.recibo, contexto); }).catch(() => {}); } anunciar(traducir(estado.mensaje),estado.tono);} }
  }
  raiz.addEventListener("submit",enviar); pintar();
  return () => { if(!montado)return; montado=false; controlador?.abort(); controlador=null; raiz.removeEventListener("submit",enviar); raiz.replaceChildren(); };
}
