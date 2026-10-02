import { MENSAJES_RECTIFICACION_DIETAS } from "./i18n-rectificacion-dietas.js?v=20260929-i18n-dietas-v1";
import { LOCALIZACION_PORTAL, ZONA_HORARIA_PORTAL } from "../../portal-i18n.js?v=20261001-ct-a-i18n-v1";

const CAMPOS = [
  ["centro_ref", "rectificacion_centro"], ["unidad_ref", "rectificacion_unidad"],
  ["administrativo_persona_ref", "rectificacion_administrativo"], ["responsable_persona_ref", "rectificacion_responsable"],
];
const ESTADOS = new Set(["pendiente", "confirmada", "rechazada", "replay_confirmado"]);
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
export function montarVistaRectificacionDietas(contenedor, { cliente, asignacion, mensajes = MENSAJES_RECTIFICACION_DIETAS, alConfirmar = () => {}, generarClaveIdempotencia = () => crypto.randomUUID().replaceAll("-", "") } = {}) {
  if (!contenedor?.ownerDocument || !cliente?.consultar || !cliente?.solicitar) throw new TypeError("vista de rectificación no disponible");
  const actual = validarAsignacion(asignacion); const documento = contenedor.ownerDocument;
  const raiz = crear(documento, "section"); raiz.dataset.dietasRectificacion = ""; raiz.className = "panel";
  const cabecera = crear(documento, "div"); cabecera.className = "cabecera-panel"; cabecera.append(crear(documento, "h3", texto(mensajes, "rectificacion_titulo")));
  const abrir = crear(documento, "button", texto(mensajes, "rectificacion_abrir")); abrir.type = "button"; abrir.className = "boton-secundario"; abrir.dataset.dietasRectificacionAbrir = ""; cabecera.append(abrir);
  const estado = crear(documento, "p"); estado.dataset.dietasRectificacionEstado = ""; estado.setAttribute("role", "status"); estado.setAttribute("aria-live", "polite");
  const formulario = crear(documento, "form"); formulario.dataset.dietasRectificacionForm = ""; formulario.hidden = true;
  const grupo = crear(documento, "fieldset"); grupo.append(crear(documento, "legend", texto(mensajes, "rectificacion_campos")));
  for (const [campo, clave] of CAMPOS) { const fila = crear(documento, "div"); fila.className = "casilla-confirmacion"; const caja = crear(documento, "input"); caja.type = "checkbox"; caja.name = "campos_a_revisar"; caja.value = campo; caja.id = `dietas-rectificacion-${actual.asignacion_ref}-${campo}`; const etiqueta = crear(documento, "label", texto(mensajes, clave)); etiqueta.htmlFor = caja.id; fila.append(caja, etiqueta); grupo.append(fila); }
  const motivo = crear(documento, "textarea"); motivo.name = "motivo_revision"; motivo.required = true; motivo.maxLength = 500;
  const etiquetaMotivo = crear(documento, "label", texto(mensajes, "rectificacion_motivo")); etiquetaMotivo.append(motivo);
  const detalle = crear(documento, "textarea"); detalle.name = "detalle_solicitado"; detalle.maxLength = 500;
  const etiquetaDetalle = crear(documento, "label", texto(mensajes, "rectificacion_detalle")); etiquetaDetalle.append(detalle);
  const enviar = crear(documento, "button", texto(mensajes, "rectificacion_enviar")); enviar.type = "submit"; enviar.className = "boton-primario"; enviar.dataset.dietasRectificacionEnviar = "";
  const cerrar = crear(documento, "button", texto(mensajes, "rectificacion_cerrar")); cerrar.type = "button"; cerrar.className = "boton-secundario"; cerrar.dataset.dietasRectificacionCerrar = "";
  const ayuda = crear(documento, "details");
  const resumenAyuda = crear(documento, "summary", "?"); resumenAyuda.setAttribute("aria-label", texto(mensajes, "rectificacion_ayuda"));
  ayuda.append(resumenAyuda, crear(documento, "p", texto(mensajes, "rectificacion_intro")));
  formulario.append(grupo, etiquetaMotivo, etiquetaDetalle, enviar, cerrar, ayuda);
  raiz.append(cabecera, estado, formulario); contenedor.append(raiz);
  let activa = true; let lector; let escritor; let pendiente = null;
  const publicar = (mensaje, nivel = "") => { estado.textContent = mensaje; estado.dataset.estado = nivel; };
  const pintarResultado = (resultado, consulta = false) => {
    const fecha = new Intl.DateTimeFormat(LOCALIZACION_PORTAL, { dateStyle: "medium", timeStyle: "short", timeZone: ZONA_HORARIA_PORTAL }).format(new Date(resultado.registrada_en));
    const claveEstado = ESTADOS.has(resultado.estado) ? `rectificacion_estado_${resultado.estado}` : "rectificacion_estado_desconocido";
    const claveRecibo = consulta ? "rectificacion_consulta" : resultado.estado === "replay_confirmado" ? "rectificacion_recibo_repetido" : "rectificacion_recibo";
    publicar(`${texto(mensajes, claveRecibo, { recibo: resultado.recibo_ref, fecha })} ${texto(mensajes, "rectificacion_estado", { estado: texto(mensajes, claveEstado) })}`, "exito");
  };
  async function consultar() {
    if (pendiente || escritor) return;
    lector?.abort(); lector = new AbortController(); publicar(texto(mensajes, "rectificacion_cargando"), "cargando");
    try { const resultado = await cliente.consultar({ relacion_ref: actual.relacion_ref, unidad_ref: actual.unidad_ref, fecha_referencia: actual.fecha_referencia }, { signal: lector.signal }); if (activa && !pendiente && !escritor) pintarResultado(resultado, true); }
    catch (error) { if (!activa || pendiente || escritor || error?.codigo === "operacion_abortada") return; publicar(error?.codigo === "no_encontrada" ? texto(mensajes, "rectificacion_vacia") : errorTexto(error, mensajes), error?.codigo === "acceso_denegado" ? "denegado" : "error"); }
  }
  function abrirFormulario() { formulario.hidden = false; abrir.hidden = true; motivo.focus?.(); }
  function cerrarFormulario() { if (pendiente) return; formulario.hidden = true; abrir.hidden = false; abrir.focus?.(); }
  const bloquearFormulario = (bloqueado) => { grupo.disabled = bloqueado; motivo.disabled = bloqueado; detalle.disabled = bloqueado; enviar.disabled = bloqueado; cerrar.disabled = bloqueado; };
  async function solicitar() {
    if (!activa || escritor) return;
    const campos = [...formulario.querySelectorAll('[name="campos_a_revisar"]')].filter((nodo) => nodo.checked).map((nodo) => nodo.value);
    const datos = pendiente || { relacion_ref: actual.relacion_ref, unidad_ref: actual.unidad_ref, asignacion_ref: actual.asignacion_ref, version_esperada: actual.version, fecha_referencia: actual.fecha_referencia, clave_idempotencia: claveNueva(generarClaveIdempotencia), campos_a_revisar: campos, motivo_revision: String(motivo.value || "").trim(), detalle_solicitado: String(detalle.value || "").trim() };
    if (!pendiente && (campos.length === 0 || datos.motivo_revision.length < 3)) { publicar(texto(mensajes, "rectificacion_invalida"), "error"); return; }
    lector?.abort(); pendiente = datos; escritor = new AbortController(); bloquearFormulario(true); publicar(texto(mensajes, "rectificacion_procesando"), "cargando");
    try { const resultado = await cliente.solicitar(datos, { signal: escritor.signal }); if (!activa) return; pintarResultado(resultado); pendiente = null; cerrarFormulario(); alConfirmar(resultado); }
    catch (error) { if (!activa || error?.codigo === "operacion_abortada") return; if (!error?.resultadoIndeterminado) pendiente = null; publicar(errorTexto(error, mensajes, true), "error"); }
    finally { escritor = null; bloquearFormulario(Boolean(pendiente)); if (activa && pendiente?.clave_idempotencia) { const reintentar = crear(documento, "button", texto(mensajes, "rectificacion_reintentar")); reintentar.type = "button"; reintentar.className = "boton-secundario"; reintentar.dataset.dietasRectificacionReintentar = ""; estado.append(" ", reintentar); } }
  }
  const click = (evento) => { const objetivo = evento.target?.closest?.("[data-dietas-rectificacion-abrir], [data-dietas-rectificacion-cerrar], [data-dietas-rectificacion-reintentar]"); if (!objetivo) return; if (objetivo.dataset.dietasRectificacionAbrir !== undefined) abrirFormulario(); else if (objetivo.dataset.dietasRectificacionCerrar !== undefined) cerrarFormulario(); else if (objetivo.dataset.dietasRectificacionReintentar !== undefined) void solicitar(); };
  const submit = (evento) => { evento.preventDefault?.(); void solicitar(); };
  raiz.addEventListener("click", click); formulario.addEventListener("submit", submit); consultar();
  return Object.freeze({ desmontar() { if (!activa) return; activa = false; lector?.abort(); escritor?.abort(); raiz.removeEventListener("click", click); formulario.removeEventListener("submit", submit); raiz.remove(); }, abrir: abrirFormulario, recargar: consultar });
}
