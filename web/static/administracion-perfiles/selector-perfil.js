import { cargarTextos } from "../comun/textos.js";

const BASE = "/api/admin/seleccion-perfil/v1";
const MAX_JSON = 64 * 1024;
const REF_PERFIL = /^prf_[A-Za-z0-9_-]{22,128}$/u;
const REF_OPACA = /^[A-Za-z0-9][A-Za-z0-9:._-]{1,255}$/u;
// Referencia de la auditoría común (IS14/AD171) o la heredada del selector.
const REF_AUDITORIA = /^(?:aud_v3_p_[a-f0-9]{32}|auditoria_seleccion_admin:[a-f0-9]{64})$/u;

export class ErrorSelectorPerfil extends Error {
  constructor(estado = 0, codigo = "servicio_no_disponible") {
    super(codigo);
    this.name = "ErrorSelectorPerfil";
    this.estado = estado;
    this.codigo = codigo;
  }
}

function objeto(valor) { return valor !== null && typeof valor === "object" && !Array.isArray(valor); }
function perfilValido(valor) { return typeof valor === "string" && REF_PERFIL.test(valor); }
function rolValido(valor) { return typeof valor === "string" && REF_OPACA.test(valor); }
function textoSeguro(valor) { return typeof valor === "string" && valor.trim() !== "" && valor.length <= 256 && !/[\p{Cc}\p{Zl}\p{Zp}]/u.test(valor); }
function revisionValida(valor) { return Number.isSafeInteger(valor) && valor >= 0; }

export function validarPropios(datos) {
  if (!objeto(datos) || !revisionValida(datos.revision) || !Array.isArray(datos.perfiles) || datos.perfiles.length > 64) return null;
  const perfiles = [];
  const vistos = new Set();
  for (const perfil of datos.perfiles) {
    if (!objeto(perfil) || !perfilValido(perfil.perfil_ref) || !rolValido(perfil.rol_version_ref)
      || perfil.clave_i18n !== undefined && !textoSeguro(perfil.clave_i18n) || vistos.has(perfil.perfil_ref)) return null;
    vistos.add(perfil.perfil_ref);
    perfiles.push(Object.freeze({ perfil_ref: perfil.perfil_ref, rol_version_ref: perfil.rol_version_ref,
      clave_i18n: perfil.clave_i18n }));
  }
  const activo = datos.perfil_activo_ref ?? null;
  if (activo !== null && (!perfilValido(activo) || !vistos.has(activo))) return null;
  return Object.freeze({ revision: datos.revision, perfil_activo_ref: activo, perfiles: Object.freeze(perfiles) });
}

export function validarSeleccion(datos, solicitado, revisionAnterior) {
  return objeto(datos) && datos.perfil_activo_ref === solicitado && revisionValida(datos.revision)
    && datos.revision > revisionAnterior && textoSeguro(datos.seleccionada_en)
    && Number.isFinite(Date.parse(datos.seleccionada_en)) && typeof datos.auditoria_ref === "string" && REF_AUDITORIA.test(datos.auditoria_ref);
}

async function leerJSONAcotado(respuesta) {
  const longitud = Number(respuesta.headers?.get?.("Content-Length"));
  if (Number.isFinite(longitud) && longitud > MAX_JSON) throw new ErrorSelectorPerfil(respuesta.status);
  if (!respuesta.body?.getReader) {
    const texto = await respuesta.text();
    if (texto.length > MAX_JSON) throw new ErrorSelectorPerfil(respuesta.status);
    return JSON.parse(texto);
  }
  const lector = respuesta.body.getReader();
  const partes = [];
  let total = 0;
  try {
    for (;;) {
      const { done, value } = await lector.read();
      if (done) break;
      total += value.byteLength;
      if (total > MAX_JSON) throw new ErrorSelectorPerfil(respuesta.status);
      partes.push(value);
    }
  } finally { await lector.cancel().catch(() => {}); lector.releaseLock(); }
  const bytes = new Uint8Array(total);
  let offset = 0;
  for (const parte of partes) { bytes.set(parte, offset); offset += parte.byteLength; }
  return JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(bytes));
}

/** Transporte de origen fijo; la identidad la comprueba la frontera mTLS del servidor. */
export function crearClienteSelectorPerfil({ fetchImpl = globalThis.fetch, origen = globalThis.location?.origin } = {}) {
  if (typeof fetchImpl !== "function" || !origen || new URL(origen).origin !== origen) throw new TypeError("cliente no disponible");
  async function pedir(metodo, ruta, cuerpo, signal) {
    let respuesta;
    try {
      respuesta = await fetchImpl(new URL(BASE + ruta, origen).href, {
        method: metodo, credentials: "same-origin", redirect: "error", referrerPolicy: "no-referrer", cache: "no-store", signal,
        headers: { Accept: "application/json", ...(cuerpo === undefined ? {} : { "Content-Type": "application/json" }) },
        ...(cuerpo === undefined ? {} : { body: JSON.stringify(cuerpo) }),
      });
    } catch (error) { if (error?.name === "AbortError") throw error; throw new ErrorSelectorPerfil(); }
    if (!respuesta || respuesta.redirected || !String(respuesta.headers?.get?.("Content-Type") ?? "").toLowerCase().startsWith("application/json")) {
      throw new ErrorSelectorPerfil(respuesta?.status ?? 0);
    }
    let datos;
    try { datos = await leerJSONAcotado(respuesta); } catch { throw new ErrorSelectorPerfil(respuesta.status); }
    if (!objeto(datos)) throw new ErrorSelectorPerfil(respuesta.status);
    if (!respuesta.ok) throw new ErrorSelectorPerfil(respuesta.status, textoSeguro(datos.error?.codigo) ? datos.error.codigo : "servicio_no_disponible");
    return datos;
  }
  return Object.freeze({
    propios: (signal) => pedir("GET", "/propios", undefined, signal),
    seleccionar: (perfilRef, revisionEsperada, signal) => {
      if (!perfilValido(perfilRef) || !revisionValida(revisionEsperada)) throw new TypeError("selección inválida");
      return pedir("POST", "/seleccion", { perfil_ref: perfilRef, revision_esperada: revisionEsperada }, signal);
    },
  });
}

export function cargarTextosSelectorPerfil() { return cargarTextos("admin-selector"); }

function nodo(documento, nombre, clase, texto) {
  const elemento = documento.createElement(nombre);
  if (clase) elemento.className = clase;
  if (texto) elemento.textContent = texto;
  return elemento;
}

/** El montaje conserva solo una revisión de la lista propia en memoria de la página. */
export function montarSelectorPerfil({ contenedor, cliente, textos, limpiarEstado, cargarContexto, traducirPerfil } = {}) {
  if (!contenedor?.ownerDocument || !textos?.traducir || !textos?.seccion || typeof limpiarEstado !== "function" || typeof cargarContexto !== "function") {
    throw new TypeError("montaje incompleto");
  }
  const documento = contenedor.ownerDocument;
  documento.documentElement.lang = textos.idioma;
  const t = (clave, variables) => textos.traducir(`general.${clave}`, variables);
  const raiz = nodo(documento, "section", "panel selector-perfil");
  const cabecera = nodo(documento, "header", "cabecera-panel selector-perfil__cabecera");
  const titulo = nodo(documento, "h2", "", t("titulo"));
  titulo.id = "selector-perfil-titulo";
  raiz.setAttribute("aria-labelledby", titulo.id);
  cabecera.append(nodo(documento, "div", "", ""));
  cabecera.firstChild.append(titulo, nodo(documento, "p", "", t("resumen")));
  const cuerpo = nodo(documento, "div", "cuerpo-panel selector-perfil__cuerpo");
  const estado = nodo(documento, "p", "selector-perfil__estado");
  estado.setAttribute("role", "status");
  estado.setAttribute("aria-live", "polite");
  const formulario = nodo(documento, "form", "selector-perfil__formulario");
  const opciones = nodo(documento, "fieldset", "selector-perfil__opciones");
  const leyenda = nodo(documento, "legend", "", t("leyenda"));
  opciones.append(leyenda);
  const accion = nodo(documento, "button", "admin-boton admin-boton--primario", t("usar_perfil"));
  accion.type = "submit";
  accion.disabled = true;
  const recargar = nodo(documento, "button", "admin-boton admin-boton--secundario", t("actualizar"));
  recargar.type = "button";
  formulario.append(opciones, nodo(documento, "div", "selector-perfil__acciones"));
  formulario.lastChild.append(accion, recargar);
  cuerpo.append(estado, formulario);
  raiz.append(cabecera, cuerpo);
  contenedor.replaceChildren(raiz);

  let vivo = true;
  let secuencia = 0;
  let peticion = null;
  let propios = null;
  let fase = "cargando";
  function actualizarEstado(clave, tipo = "neutro") {
    estado.textContent = t(clave);
    estado.dataset.tipo = tipo;
  }
  function bloquear(clave, tipo = "neutro") {
    fase = "bloqueado";
    propios = null;
    opciones.replaceChildren(leyenda);
    accion.disabled = true;
    actualizarEstado(clave, tipo);
  }
  function etiqueta(perfil) {
    if (perfil.clave_i18n) {
      try { const traduccion = textos.traducir(`perfiles.${perfil.clave_i18n}`); if (textoSeguro(traduccion)) return traduccion; }
      catch { /* Una clave sin traducción conserva el nombre no disponible. */ }
    }
    if (perfil.clave_i18n && typeof traducirPerfil === "function") {
      try { const traduccion = traducirPerfil(perfil.clave_i18n, textos); if (textoSeguro(traduccion)) return traduccion; }
      catch { /* La falta de catálogo no altera autoridad. */ }
    }
    return t("perfil_sin_nombre", { referencia: perfil.perfil_ref });
  }
  function pintar(datos) {
    propios = datos;
    fase = "lista";
    opciones.replaceChildren(leyenda);
    if (!datos.perfiles.length) {
      actualizarEstado("sin_perfiles", "aviso");
      accion.disabled = true;
      return;
    }
    datos.perfiles.forEach((perfil, indice) => {
      const id = `selector-perfil-opcion-${indice}`;
      const fila = nodo(documento, "div", "selector-perfil__fila");
      const opcion = nodo(documento, "input");
      opcion.type = "radio";
      opcion.name = "selector-perfil";
      opcion.id = id;
      opcion.value = perfil.perfil_ref;
      opcion.checked = perfil.perfil_ref === datos.perfil_activo_ref;
      const etiquetaVisible = etiqueta(perfil);
      const label = nodo(documento, "label", "selector-perfil__etiqueta", etiquetaVisible);
      label.htmlFor = id;
      if (opcion.checked) label.append(nodo(documento, "span", "selector-perfil__activo", t("activo")));
      const detalle = nodo(documento, "details", "selector-perfil__detalle");
      detalle.append(nodo(documento, "summary", "", t("ver_referencia")),
        nodo(documento, "code", "", perfil.perfil_ref));
      fila.append(opcion, label, detalle);
      opciones.append(fila);
    });
    actualizarEstado(datos.perfil_activo_ref ? "perfil_en_uso" : "elegir_perfil", datos.perfil_activo_ref ? "exito" : "neutro");
    accion.disabled = true;
  }
  function seleccionActual() { return opciones.querySelector('input[name="selector-perfil"]:checked')?.value ?? ""; }
  function actualizarAccion() {
    accion.disabled = fase !== "lista" || !propios || !seleccionActual() || seleccionActual() === propios.perfil_activo_ref;
  }
  async function vaciar() {
    bloquear("cargando");
    await limpiarEstado();
  }
  async function leerPropios({ causa = "inicio", esperado = null, revision = null } = {}) {
    if (!vivo || !cliente?.propios) return;
    const turno = ++secuencia;
    peticion?.abort();
    const controlador = new AbortController();
    peticion = controlador;
    try {
      await vaciar();
      if (!vivo || turno !== secuencia) return;
      const datos = validarPropios(await cliente.propios(controlador.signal));
      if (!datos) throw new ErrorSelectorPerfil();
      if (!vivo || turno !== secuencia || controlador.signal.aborted) return;
      pintar(datos);
      fase = "contexto";
      if (datos.perfil_activo_ref) await cargarContexto({ perfil_ref: datos.perfil_activo_ref, revision: datos.revision, signal: controlador.signal });
      if (!vivo || turno !== secuencia) return;
      fase = "lista";
      if (causa === "conflicto") actualizarEstado("conflicto", "aviso");
      else if (causa === "ambiguo") actualizarEstado("estado_comprobado", "aviso");
      else if (causa === "confirmacion" && datos.perfil_activo_ref === esperado && datos.revision === revision) actualizarEstado("cambio_confirmado", "exito");
      else if (causa === "confirmacion") actualizarEstado("estado_cambiado", "aviso");
    } catch (error) {
      if (!vivo || turno !== secuencia || error?.name === "AbortError") return;
      await Promise.resolve().then(limpiarEstado).catch(() => {});
      bloquear(error?.estado === 401 || error?.estado === 403 ? "denegado" : causa === "ambiguo" ? "estado_no_comprobado" : "error_carga", "error");
    } finally { if (peticion === controlador) peticion = null; }
  }
  async function seleccionar(evento) {
    evento.preventDefault();
    if (!vivo || fase !== "lista" || !propios) return;
    const perfil = seleccionActual();
    if (!perfilValido(perfil) || !propios.perfiles.some((item) => item.perfil_ref === perfil) || perfil === propios.perfil_activo_ref) return;
    const anterior = propios.revision;
    const turno = ++secuencia;
    peticion?.abort();
    const controlador = new AbortController();
    peticion = controlador;
    try {
      await vaciar();
      if (!vivo || turno !== secuencia) return;
      actualizarEstado("cambiando");
      const respuesta = await cliente.seleccionar(perfil, anterior, controlador.signal);
      if (!vivo || turno !== secuencia || controlador.signal.aborted) return;
      if (!validarSeleccion(respuesta, perfil, anterior)) throw new ErrorSelectorPerfil();
      await leerPropios({ causa: "confirmacion", esperado: perfil, revision: respuesta.revision });
    } catch (error) {
      if (!vivo || turno !== secuencia || error?.name === "AbortError") return;
      if (error?.estado === 401 || error?.estado === 403) { bloquear("denegado", "error"); return; }
      await leerPropios({ causa: error?.estado === 409 ? "conflicto" : "ambiguo" });
    } finally { if (peticion === controlador) peticion = null; }
  }

  formulario.addEventListener("change", actualizarAccion);
  formulario.addEventListener("submit", seleccionar);
  recargar.addEventListener("click", () => { if (cliente?.propios) leerPropios(); });
  if (cliente?.propios && cliente?.seleccionar) leerPropios();
  else { bloquear("no_instalado", "aviso"); recargar.disabled = true; }
  return Object.freeze({
    recargar: () => cliente?.propios ? leerPropios() : Promise.resolve(),
    destruir() { vivo = false; ++secuencia; peticion?.abort(); contenedor.replaceChildren(); },
  });
}
