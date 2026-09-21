import { ErrorAPIBorradorLlamamiento, crearClienteBorradorLlamamiento } from "./portal-borrador-llamamiento-api.js";

export const MENSAJES_BORRADOR_LLAMAMIENTO_ES = Object.freeze({
  titulo: "Preparar borrador interno", descripcion: "Registre un resumen para continuar su revisión. No selecciona personas ni realiza contactos.",
  conectado: "Conectado", pendiente: "Pendiente de conexión", etiqueta_resumen: "Resumen de la preparación",
  ayuda_resumen: "No incluya datos personales, candidatos, teléfonos ni correos. El resumen admite entre 3 y 2.000 bytes UTF-8.", guardar: "Guardar borrador", guardando: "Guardando…",
  reintentar: "Reintentar guardado", etiqueta_referencia: "Recuperar borrador por referencia", recuperar: "Recuperar borrador",
  recuperando: "Recuperando…", recuperado: "Borrador recuperado", registrado: "Borrador interno de llamamiento registrado",
  error_generico: "No se pudo preparar el borrador.", reintento_ayuda: " Puede reintentar el mismo contenido.",
  confirmacion: "Borrador registrado: {referencia}{reintento}.", reintento_confirmado: " (reintento confirmado)",
});
const CLAVES = Object.freeze(Object.keys(MENSAJES_BORRADOR_LLAMAMIENTO_ES));
const escapar = (valor) => String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#039;");

export function crearTraductorBorradorLlamamiento(catalogo = MENSAJES_BORRADOR_LLAMAMIENTO_ES) {
  if (!catalogo || typeof catalogo !== "object" || CLAVES.some((clave) => typeof catalogo[clave] !== "string" || catalogo[clave] === "")) throw new TypeError("catálogo de borrador de llamamiento incompleto");
  return (clave, variables = {}) => {
    if (!CLAVES.includes(clave)) throw new Error(`clave i18n de borrador de llamamiento desconocida: ${clave}`);
    return catalogo[clave].replace(/\{([a-z_]+)\}/g, (_texto, nombre) => String(variables[nombre] ?? ""));
  };
}

export function crearSuperficieBorradorLlamamiento({ crearClienteImpl = crearClienteBorradorLlamamiento, alCambiar = () => {}, anunciar = () => {}, traducir = crearTraductorBorradorLlamamiento() } = {}) {
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
