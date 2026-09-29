import { ErrorAPIBorradorLlamamiento, crearClienteBorradorLlamamiento } from "./portal-borrador-llamamiento-api.js?v=20260928-rrhh-cache-unificada-v1";

import { traducirPortal } from "./portal-i18n.js?v=20260928-rrhh-i18n-unificada-v1";

const escapar = (valor) => String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#039;");
const traducirBorrador = (clave, variables = {}) => traducirPortal(`bl_${clave}`, variables);

export function crearSuperficieBorradorLlamamiento({ crearClienteImpl = crearClienteBorradorLlamamiento, alCambiar = () => {}, anunciar = () => {}, traducir = traducirBorrador } = {}) {
  let cliente = null; let controlador = null; let activa = true;
  const estado = { resumen: "", referencia: "", clave: null, enviando: false, error: "", errorOperacion: "", recibo: null, conectado: false };
  const cambiar = () => { if (activa) alCambiar(); };
  const obtenerCliente = () => (cliente ||= crearClienteImpl());
  async function ejecutar(accion, valor) {
    if (!activa || estado.enviando) return false;
    estado.error = ""; estado.errorOperacion = ""; estado.enviando = true; controlador?.abort(); controlador = new AbortController(); const signal = controlador.signal;
    try {
      let recibo;
      if (accion === "crear") { estado.resumen = String(valor ?? ""); estado.clave ||= obtenerCliente().generarClave(); cambiar(); recibo = await obtenerCliente().crear({ resumen: estado.resumen, claveIdempotencia: estado.clave, signal }); estado.clave = null; }
      else { estado.referencia = String(valor ?? ""); cambiar(); recibo = await obtenerCliente().consultar({ referencia: estado.referencia, signal }); }
      if (!activa || signal.aborted) return false;
      estado.recibo = recibo; estado.conectado = true; anunciar(traducir(accion === "crear" ? "registrado" : "recuperado")); return true;
    } catch (error) {
      if (!activa || signal.aborted || error?.name === "AbortError") return false;
      estado.error = error instanceof ErrorAPIBorradorLlamamiento ? error.message : traducir("error_generico"); estado.errorOperacion = accion; return false;
    } finally { if (activa && controlador?.signal === signal) { estado.enviando = false; cambiar(); } }
  }
  return Object.freeze({
    renderizar: () => `<section class="panel panel-separado" aria-labelledby="titulo-borrador-llamamiento"><div class="cabecera-panel"><div><h3 id="titulo-borrador-llamamiento">${escapar(traducir("titulo"))}</h3><p>${escapar(traducir("descripcion"))}</p></div><span class="estado-chip${estado.conectado ? " exito" : ""}">${escapar(traducir(estado.conectado ? "conectado" : "pendiente"))}</span></div><div class="cuerpo-panel"><form data-borrador-llamamiento-form data-accion="crear"><label for="resumen-borrador-llamamiento">${escapar(traducir("etiqueta_resumen"))}</label><textarea id="resumen-borrador-llamamiento" name="resumen" required rows="3"${estado.enviando ? " disabled" : ""}>${escapar(estado.resumen)}</textarea><p class="dato-secundario">${escapar(traducir("ayuda_resumen"))}</p><button type="submit" class="boton-primario"${estado.enviando ? " disabled" : ""}>${escapar(traducir(estado.enviando ? "guardando" : estado.clave ? "reintentar" : "guardar"))}</button></form><form data-borrador-llamamiento-form data-accion="consultar"><label for="referencia-borrador-llamamiento">${escapar(traducir("etiqueta_referencia"))}</label><input id="referencia-borrador-llamamiento" name="referencia" required maxlength="90" value="${escapar(estado.referencia)}"${estado.enviando ? " disabled" : ""}><button type="submit" class="boton-secundario"${estado.enviando ? " disabled" : ""}>${escapar(traducir(estado.enviando ? "recuperando" : "recuperar"))}</button></form>${estado.error ? `<p class="mensaje-error" role="alert">${escapar(estado.error)}${estado.errorOperacion === "crear" && estado.clave ? escapar(traducir("reintento_ayuda")) : ""}</p>` : ""}${estado.recibo ? `<p class="mensaje-exito" role="status">${escapar(traducir("confirmacion", { referencia: estado.recibo.borrador_ref, reintento: estado.recibo.reintento_idempotente ? traducir("reintento_confirmado") : "" }))}</p>` : ""}</div></section>`,
    manejarEnvio: ({ resumen }) => ejecutar("crear", resumen), manejarConsulta: ({ referencia }) => ejecutar("consultar", referencia),
    manejarFormulario: ({ accion, resumen, referencia }) => accion === "consultar" ? ejecutar("consultar", referencia) : ejecutar("crear", resumen),
    desmontar: () => { if (!activa) return; activa = false; controlador?.abort(); controlador = null; estado.enviando = false; }, activar: () => { activa = true; }, estado: () => Object.freeze({ ...estado }),
  });
}
