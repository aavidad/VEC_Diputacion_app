import { MENSAJES_RECTIFICACION_DIETAS_ES } from "./i18n-rectificacion-dietas.js?v=20260925-dietas-montaje-v1";

const CAMPOS = [
  ["centro_ref", "rectificacion_centro"], ["unidad_ref", "rectificacion_unidad"],
  ["administrativo_persona_ref", "rectificacion_administrativo"], ["responsable_persona_ref", "rectificacion_responsable"],
];
const texto = (mensajes, clave, valores = {}) => (mensajes[clave] || clave).replace(/\{([^}]+)\}/gu, (_m, nombre) => valores[nombre] ?? "");
const crear = (documento, etiqueta, contenido) => { const nodo = documento.createElement(etiqueta); if (contenido !== undefined) nodo.textContent = contenido; return nodo; };
const claveNueva = (generar) => {
  const clave = generar?.();
  if (typeof clave !== "string" || !/^[A-Za-z0-9:_-]{16,128}$/u.test(clave)) throw new TypeError("clave de idempotencia no disponible");
  return clave;
};
const errorTexto = (error, mensajes, escritura = false) => {
  if (error?.codigo === "acceso_denegado" || error?.codigo === "autenticacion_requerida") return texto(mensajes, "rectificacion_denegada");
  if (error?.codigo === "conflicto") return texto(mensajes, "rectificacion_conflicto");
  if (escritura && error?.resultadoIndeterminado) return texto(mensajes, "rectificacion_incierta");
  return texto(mensajes, escritura ? "rectificacion_error" : "rectificacion_error");
};
function validarAsignacion(asignacion) {
  if (!asignacion || asignacion.verificada !== true || typeof asignacion.relacion_ref !== "string" || typeof asignacion.unidad_ref !== "string"
    || !/^ads_[A-Za-z0-9_-]{22,128}$/u.test(asignacion.asignacion_ref) || !Number.isSafeInteger(asignacion.version) || asignacion.version < 1
    || typeof asignacion.fecha_referencia !== "string") throw new TypeError("asignación verificada requerida");
  return asignacion;
}

/** Monta el bloque opcional D7c dentro del panel ya existente de la asignación D7. */
export function montarVistaRectificacionDietas(contenedor, { cliente, asignacion, mensajes = MENSAJES_RECTIFICACION_DIETAS_ES, alConfirmar = () => {}, generarClaveIdempotencia = () => crypto.randomUUID().replaceAll("-", "") } = {}) {
  if (!contenedor?.ownerDocument || !cliente?.consultar || !cliente?.solicitar) throw new TypeError("vista de rectificación no disponible");
  const actual = validarAsignacion(asignacion); const documento = contenedor.ownerDocument;
  const raiz = crear(documento, "section"); raiz.dataset.dietasRectificacion = ""; raiz.className = "panel";
  const cabecera = crear(documento, "div"); cabecera.className = "cabecera-panel"; cabecera.append(crear(documento, "h3", texto(mensajes, "rectificacion_titulo")));
  const abrir = crear(documento, "button", texto(mensajes, "rectificacion_abrir")); abrir.type = "button"; abrir.dataset.dietasRectificacionAbrir = ""; cabecera.append(abrir);
  const estado = crear(documento, "p"); estado.dataset.dietasRectificacionEstado = ""; estado.setAttribute("role", "status");
  const formulario = crear(documento, "form"); formulario.dataset.dietasRectificacionForm = ""; formulario.hidden = true;
  const grupo = crear(documento, "fieldset"); grupo.append(crear(documento, "legend", texto(mensajes, "rectificacion_campos")));
  for (const [campo, clave] of CAMPOS) { const etiqueta = crear(documento, "label"); const caja = crear(documento, "input"); caja.type = "checkbox"; caja.name = "campos_a_revisar"; caja.value = campo; etiqueta.append(caja, crear(documento, "span", texto(mensajes, clave))); grupo.append(etiqueta); }
  const motivo = crear(documento, "textarea"); motivo.name = "motivo_revision"; motivo.required = true; motivo.maxLength = 500;
  const etiquetaMotivo = crear(documento, "label", texto(mensajes, "rectificacion_motivo")); etiquetaMotivo.append(motivo);
  const detalle = crear(documento, "textarea"); detalle.name = "detalle_solicitado"; detalle.maxLength = 500;
  const etiquetaDetalle = crear(documento, "label", texto(mensajes, "rectificacion_detalle")); etiquetaDetalle.append(detalle);
  const enviar = crear(documento, "button", texto(mensajes, "rectificacion_enviar")); enviar.type = "submit"; enviar.dataset.dietasRectificacionEnviar = "";
  const cerrar = crear(documento, "button", texto(mensajes, "rectificacion_cerrar")); cerrar.type = "button"; cerrar.dataset.dietasRectificacionCerrar = "";
  const ayuda = crear(documento, "details");
  const resumenAyuda = crear(documento, "summary", "?"); resumenAyuda.setAttribute("aria-label", texto(mensajes, "rectificacion_ayuda"));
  ayuda.append(resumenAyuda, crear(documento, "p", texto(mensajes, "rectificacion_intro")));
  formulario.append(grupo, etiquetaMotivo, etiquetaDetalle, enviar, cerrar, ayuda);
  raiz.append(cabecera, estado, formulario); contenedor.append(raiz);
  let activa = true; let lector; let escritor; let pendiente = null;
  const publicar = (mensaje, nivel = "") => { estado.textContent = mensaje; estado.dataset.estado = nivel; };
  const pintarResultado = (resultado) => publicar(`${texto(mensajes, resultado.estado === "replay_confirmado" ? "rectificacion_recibo_repetido" : "rectificacion_recibo", { recibo: resultado.recibo_ref, fecha: resultado.registrada_en })} ${texto(mensajes, "rectificacion_estado", { estado: resultado.estado })}`, "exito");
  async function consultar() {
    lector?.abort(); lector = new AbortController(); publicar(texto(mensajes, "rectificacion_cargando"), "cargando");
    try { const resultado = await cliente.consultar({ relacion_ref: actual.relacion_ref, unidad_ref: actual.unidad_ref, fecha_referencia: actual.fecha_referencia }, { signal: lector.signal }); if (activa) pintarResultado(resultado); }
    catch (error) { if (!activa || error?.codigo === "operacion_abortada") return; publicar(error?.codigo === "no_encontrada" ? texto(mensajes, "rectificacion_vacia") : errorTexto(error, mensajes), error?.codigo === "acceso_denegado" ? "denegado" : "error"); }
  }
  function abrirFormulario() { formulario.hidden = false; abrir.hidden = true; motivo.focus?.(); }
  function cerrarFormulario() { formulario.hidden = true; abrir.hidden = false; abrir.focus?.(); }
  async function solicitar(reintento = false) {
    if (!activa || escritor) return;
    const campos = [...formulario.querySelectorAll('[name="campos_a_revisar"]')].filter((nodo) => nodo.checked).map((nodo) => nodo.value);
    const datos = reintento ? pendiente : { relacion_ref: actual.relacion_ref, unidad_ref: actual.unidad_ref, asignacion_ref: actual.asignacion_ref, version_esperada: actual.version, fecha_referencia: actual.fecha_referencia, clave_idempotencia: claveNueva(generarClaveIdempotencia), campos_a_revisar: campos, motivo_revision: String(motivo.value || "").trim(), detalle_solicitado: String(detalle.value || "").trim() };
    if (!reintento && (campos.length === 0 || datos.motivo_revision.length < 3)) { publicar(texto(mensajes, "rectificacion_invalida"), "error"); return; }
    pendiente = datos; escritor = new AbortController(); enviar.disabled = true; publicar(texto(mensajes, "rectificacion_procesando"), "cargando");
    try { const resultado = await cliente.solicitar(datos, { signal: escritor.signal }); if (!activa) return; pintarResultado(resultado); pendiente = null; cerrarFormulario(); alConfirmar(resultado); }
    catch (error) { if (!activa || error?.codigo === "operacion_abortada") return; if (!error?.resultadoIndeterminado) pendiente = null; publicar(errorTexto(error, mensajes, true), "error"); }
    finally { escritor = null; enviar.disabled = false; if (activa && pendiente?.clave_idempotencia) { const reintentar = crear(documento, "button", texto(mensajes, "rectificacion_reintentar")); reintentar.type = "button"; reintentar.dataset.dietasRectificacionReintentar = ""; estado.append(" ", reintentar); } }
  }
  const click = (evento) => { const objetivo = evento.target?.closest?.("[data-dietas-rectificacion-abrir], [data-dietas-rectificacion-cerrar], [data-dietas-rectificacion-reintentar]"); if (!objetivo) return; if (objetivo.dataset.dietasRectificacionAbrir !== undefined) abrirFormulario(); else if (objetivo.dataset.dietasRectificacionCerrar !== undefined) cerrarFormulario(); else if (objetivo.dataset.dietasRectificacionReintentar !== undefined) void solicitar(true); };
  const submit = (evento) => { evento.preventDefault?.(); void solicitar(false); };
  raiz.addEventListener("click", click); formulario.addEventListener("submit", submit); consultar();
  return Object.freeze({ desmontar() { if (!activa) return; activa = false; lector?.abort(); escritor?.abort(); raiz.removeEventListener("click", click); formulario.removeEventListener("submit", submit); raiz.remove(); }, abrir: abrirFormulario, recargar: consultar });
}
