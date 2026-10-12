import { localizacionAreaPersonal, traducir } from "./i18n.js";
import { escaparHTML, panel } from "./vistas/comunes.js";

export const RUTA_PREFERENCIAS_BOLSAS = "/api/vec/bolsa/mi-bolsa/preferencias";
const MAXIMO_BYTES = 64 * 1024;
const CAMPOS_LECTURA = ["version", "orden_guardado", "bolsas_actuales", "bolsas_nuevas", "bolsas_retiradas", "revision_requerida", "recibo_ref", "registrado_en"];
const CAMPOS_RECIBO = ["recibo_ref", "version", "orden", "registrado_en", "reutilizado"];
const INSTANTE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{6}Z$/u;

const t = (clave, variables) => traducir(`areaPersonal.miBolsa.preferenciasBolsas.${clave}`, variables);
const objeto = (valor) => valor !== null && typeof valor === "object" && !Array.isArray(valor);
const fallo = (codigo) => Object.assign(new Error(codigo), { codigo });
function mismasClaves(valor, claves) {
  return objeto(valor) && Object.keys(valor).sort().join("|") === [...claves].sort().join("|");
}
function lista(valor) {
  return Array.isArray(valor) && valor.length <= 500 && valor.every((ref) => typeof ref === "string" && ref.length > 0 && ref.length <= 200) && new Set(valor).size === valor.length;
}
function instante(valor) {
  return typeof valor === "string" && INSTANTE.test(valor) && !Number.isNaN(Date.parse(valor));
}
function igual(a, b) { return a.length === b.length && a.every((item, i) => item === b[i]); }
function permutacion(orden, actuales) { return lista(orden) && orden.length === actuales.length && orden.every((ref) => actuales.includes(ref)); }

export function validarPreferenciasBolsas(valor) {
  if (!mismasClaves(valor, CAMPOS_LECTURA) || !Number.isSafeInteger(valor.version) || valor.version < 0 ||
    !lista(valor.orden_guardado) || !lista(valor.bolsas_actuales) || !lista(valor.bolsas_nuevas) || !lista(valor.bolsas_retiradas) ||
    typeof valor.revision_requerida !== "boolean" ||
    (valor.recibo_ref !== null && (typeof valor.recibo_ref !== "string" || !valor.recibo_ref)) ||
    (valor.registrado_en !== null && !instante(valor.registrado_en)) ||
    valor.bolsas_nuevas.some((ref) => !valor.bolsas_actuales.includes(ref)) ||
    valor.bolsas_retiradas.some((ref) => valor.bolsas_actuales.includes(ref)) ||
    valor.orden_guardado.some((ref) => !valor.bolsas_actuales.includes(ref) && !valor.bolsas_retiradas.includes(ref))) throw fallo("respuesta");
  return structuredClone(valor);
}

export function validarReciboPreferenciasBolsas(valor, peticion) {
  if (!mismasClaves(valor, CAMPOS_RECIBO) || typeof valor.recibo_ref !== "string" || !valor.recibo_ref ||
    !Number.isSafeInteger(valor.version) || valor.version !== peticion.version_esperada + 1 ||
    !lista(valor.orden) || !igual(valor.orden, peticion.orden) || !instante(valor.registrado_en) ||
    typeof valor.reutilizado !== "boolean") throw fallo("respuesta");
  return structuredClone(valor);
}

async function solicitar(fetchImpl, metodo, cuerpo, signal) {
  let respuesta;
  try {
    respuesta = await fetchImpl(RUTA_PREFERENCIAS_BOLSAS, {
      method: metodo, credentials: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer",
      headers: { Accept: "application/json", ...(cuerpo ? { "Content-Type": "application/json" } : {}) },
      ...(cuerpo ? { body: JSON.stringify(cuerpo) } : {}), ...(signal ? { signal } : {}),
    });
  } catch { throw fallo("incierto"); }
  if (respuesta?.status !== 200 && !(metodo === "PUT" && respuesta?.status === 201)) {
    const codigo = ({ 401: "autenticacion", 403: "denegado", 409: "conflicto", 503: "servicio" })[respuesta?.status] || "error";
    throw fallo(codigo);
  }
  let texto;
  try { texto = await respuesta.text(); } catch { throw fallo(metodo === "PUT" ? "incierto" : "error"); }
  if (new TextEncoder().encode(texto).byteLength > MAXIMO_BYTES) throw fallo("respuesta");
  try { return JSON.parse(texto); } catch { throw fallo(metodo === "PUT" ? "incierto" : "respuesta"); }
}

export async function cargarPreferenciasBolsas({ fetchImpl = globalThis.fetch, signal } = {}) {
  return validarPreferenciasBolsas(await solicitar(fetchImpl, "GET", null, signal));
}
export async function guardarPreferenciasBolsas(peticion, { fetchImpl = globalThis.fetch } = {}) {
  if (!mismasClaves(peticion, ["orden", "version_esperada", "clave_idempotencia"]) ||
    !lista(peticion.orden) || !Number.isSafeInteger(peticion.version_esperada) || peticion.version_esperada < 0 ||
    typeof peticion.clave_idempotencia !== "string" || !/^[A-Za-z0-9._:-]{8,128}$/u.test(peticion.clave_idempotencia)) throw fallo("respuesta");
  const resultado = await solicitar(fetchImpl, "PUT", peticion);
  try { return validarReciboPreferenciasBolsas(resultado, peticion); }
  catch { throw fallo("incierto"); }
}

function ordenInicial(datos) {
  return [...datos.orden_guardado.filter((ref) => datos.bolsas_actuales.includes(ref)),
    ...datos.bolsas_actuales.filter((ref) => !datos.orden_guardado.includes(ref))];
}
function nombre(ref, nombres) { return nombres.get(ref) || t("bolsaSinNombre"); }
function fecha(valor) { return new Intl.DateTimeFormat(localizacionAreaPersonal(), { dateStyle: "medium", timeStyle: "short", timeZone: "Europe/Madrid" }).format(new Date(valor)); }

export function renderizarPreferenciasBolsas(datos = null, { estado = "cargando", orden = null, nombres = new Map(), confirmado = false, recibo = null, pendiente = false } = {}) {
  let contenido;
  if (estado === "cargando" || estado === "guardando") contenido = `<p role="status">${escaparHTML(t(estado))}</p>`;
  else if (estado !== "correcto" && estado !== "guardado") {
    const clave = ["autenticacion", "denegado", "conflicto", "servicio", "incierto", "respuesta"].includes(estado) ? estado : "error";
    const reintento = clave === "denegado" || clave === "autenticacion" ? "" : `<button type="button" class="boton-secundario" data-preferencias-bolsas="${pendiente && (clave === "incierto" || clave === "servicio") ? "reintentar-guardado" : "recargar"}">${escaparHTML(t(pendiente && (clave === "incierto" || clave === "servicio") ? "reintentarGuardado" : "recargar"))}</button>`;
    contenido = `<div class="${clave === "denegado" || clave === "autenticacion" ? "estado-error" : "estado-vacio"}" role="alert"><p>${escaparHTML(t(clave))}</p>${reintento}</div>`;
  } else if (!datos?.bolsas_actuales.length) contenido = `<p class="estado-vacio">${escaparHTML(t("vacio"))}</p>`;
  else {
    const actual = orden ?? ordenInicial(datos);
    const novedades = datos.bolsas_nuevas.length || datos.bolsas_retiradas.length;
    const cambios = !igual(actual, datos.orden_guardado) || datos.revision_requerida;
    const items = actual.map((ref, i) => `<li class="portal-mi-bolsa__bolsa"><span>${escaparHTML(new Intl.NumberFormat(localizacionAreaPersonal()).format(i + 1))}. ${escaparHTML(nombre(ref, nombres))}${datos.bolsas_nuevas.includes(ref) ? ` · ${escaparHTML(t("nuevaBolsa"))}` : ""}</span><span class="fila-acciones"><button type="button" class="boton-secundario" data-preferencias-bolsas="subir" data-indice="${i}" ${i === 0 ? "disabled" : ""} aria-label="${escaparHTML(t("subirBolsa", { bolsa: nombre(ref, nombres) }))}">${escaparHTML(t("subir"))}</button><button type="button" class="boton-secundario" data-preferencias-bolsas="bajar" data-indice="${i}" ${i === actual.length - 1 ? "disabled" : ""} aria-label="${escaparHTML(t("bajarBolsa", { bolsa: nombre(ref, nombres) }))}">${escaparHTML(t("bajar"))}</button></span></li>`).join("");
    const avisos = datos.bolsas_retiradas.length ? `<p role="status">${escaparHTML(t("retiradaBolsa"))}</p>` : "";
    const confirmacion = novedades ? `<label class="opcion-check"><input type="checkbox" data-preferencias-bolsas="confirmar" ${confirmado ? "checked" : ""}><span>${escaparHTML(t("confirmarNuevoOrden"))}</span></label>` : "";
    const resultado = recibo || (datos.recibo_ref ? { recibo_ref: datos.recibo_ref, registrado_en: datos.registrado_en } : null);
    contenido = `${avisos}<ol class="linea-tiempo">${items}</ol>${confirmacion}<div class="fila-acciones"><button type="button" class="boton-primario" data-preferencias-bolsas="guardar" ${!cambios || novedades && !confirmado ? "disabled" : ""}>${escaparHTML(t("guardar"))}</button></div>${resultado ? `<p role="status">${escaparHTML(t(estado === "guardado" ? "guardado" : "ultimoGuardado"))}${resultado.registrado_en ? ` · ${escaparHTML(fecha(resultado.registrado_en))}` : ""}</p>` : ""}`;
  }
  return panel(t("titulo"), "", contenido, { clase: "preferencias-bolsas" });
}

export function crearControladorPreferenciasBolsas({ fetchImpl = globalThis.fetch } = {}) {
  let contenedor = null, nombres = new Map(), datos = null, orden = null;
  let confirmado = false, estado = "cargando", recibo = null, lectura = null;
  let secuencia = 0, guardando = false, destruido = false, cargado = false;
  let intencionPendiente = null;
  const pintar = () => { if (contenedor && !destruido) contenedor.innerHTML = renderizarPreferenciasBolsas(datos, { estado, orden, nombres, confirmado, recibo, pendiente: Boolean(intencionPendiente) }); };
  async function cargar() {
    if (destruido || guardando || lectura) return;
    const controller = new AbortController(); lectura = controller;
    const propia = ++secuencia; estado = "cargando"; pintar();
    try {
      const nuevo = await cargarPreferenciasBolsas({ fetchImpl, signal: controller.signal });
      if (destruido || propia !== secuencia) return;
      cargado = true; datos = nuevo; orden = ordenInicial(nuevo); confirmado = false; estado = "correcto";
      if (intencionPendiente && nuevo.version > intencionPendiente.version_esperada && igual(nuevo.orden_guardado, intencionPendiente.orden) && nuevo.recibo_ref && nuevo.registrado_en) {
        recibo = nuevo.recibo_ref ? { recibo_ref: nuevo.recibo_ref, registrado_en: nuevo.registrado_en } : null;
        intencionPendiente = null; estado = "guardado";
      } else if (intencionPendiente && nuevo.version !== intencionPendiente.version_esperada && !igual(nuevo.orden_guardado, intencionPendiente.orden)) {
        intencionPendiente = null; estado = "conflicto";
      } else if (intencionPendiente) estado = "incierto";
      pintar();
    } catch (error) { if (!destruido && propia === secuencia && !controller.signal.aborted) { estado = error.codigo || "error"; pintar(); } }
    finally { if (lectura === controller) lectura = null; }
  }
  async function guardar() {
    if (destruido || guardando || !datos) return;
    if (!intencionPendiente) {
      if (!permutacion(orden, datos.bolsas_actuales) || (datos.bolsas_nuevas.length || datos.bolsas_retiradas.length) && !confirmado) return;
      if (igual(orden, datos.orden_guardado) && !datos.revision_requerida) return;
      if (typeof globalThis.crypto?.randomUUID !== "function") { estado = "error"; pintar(); return; }
      intencionPendiente = { orden: [...orden], version_esperada: datos.version, clave_idempotencia: globalThis.crypto.randomUUID() };
    }
    const peticion = intencionPendiente;
    guardando = true; estado = "guardando"; pintar();
    try {
      const nuevoRecibo = await guardarPreferenciasBolsas(peticion, { fetchImpl });
      if (intencionPendiente === peticion) intencionPendiente = null;
      if (destruido) return;
      recibo = nuevoRecibo; datos = { ...datos, version: nuevoRecibo.version, orden_guardado: [...nuevoRecibo.orden], bolsas_nuevas: [], bolsas_retiradas: [], revision_requerida: false, recibo_ref: nuevoRecibo.recibo_ref, registrado_en: nuevoRecibo.registrado_en };
      orden = [...nuevoRecibo.orden]; confirmado = false; estado = "guardado"; pintar();
    } catch (error) {
      if (["conflicto", "denegado", "autenticacion"].includes(error.codigo)) intencionPendiente = null;
      if (!destruido) { estado = error.codigo || "incierto"; pintar(); }
    } finally { guardando = false; }
  }
  function pulsar(evento) {
    const boton = evento.target?.closest?.("[data-preferencias-bolsas]");
    const accion = boton?.dataset.preferenciasBolsas;
    if (!accion || guardando || destruido) return;
    if (accion === "recargar") { cargado = false; void cargar(); }
    else if (accion === "reintentar-guardado" || accion === "guardar") void guardar();
    else if ((accion === "subir" || accion === "bajar") && datos && !intencionPendiente) {
      const i = Number(boton.dataset.indice), j = i + (accion === "subir" ? -1 : 1);
      if (!Number.isInteger(i) || i < 0 || j < 0 || j >= orden.length) return;
      [orden[i], orden[j]] = [orden[j], orden[i]]; estado = "correcto"; pintar();
      const preferida = contenedor?.querySelector(`[data-preferencias-bolsas="${accion}"][data-indice="${j}"]`);
      const alternativa = contenedor?.querySelector(`[data-preferencias-bolsas="${accion === "subir" ? "bajar" : "subir"}"][data-indice="${j}"]`);
      (preferida?.disabled ? alternativa : preferida)?.focus?.();
    }
  }
  function cambiar(evento) {
    if (evento.target?.dataset?.preferenciasBolsas !== "confirmar" || intencionPendiente || guardando) return;
    confirmado = evento.target.checked === true; pintar();
    contenedor?.querySelector('[data-preferencias-bolsas="confirmar"]')?.focus?.();
  }
  return {
    montar({ contenedor: nuevo, participaciones = [] } = {}) {
      if (destruido || !nuevo) return null;
      if (contenedor) { contenedor.removeEventListener("click", pulsar); contenedor.removeEventListener("change", cambiar); }
      contenedor = nuevo;
      nombres = new Map((Array.isArray(participaciones) ? participaciones : []).filter((p) => p && typeof p.bolsa === "string" && typeof p.categoria === "string" && p.categoria.trim()).map((p) => [p.bolsa, p.categoria]));
      contenedor.addEventListener("click", pulsar); contenedor.addEventListener("change", cambiar);
      pintar();
      if (!cargado && !lectura && !guardando) void cargar();
      return this;
    },
    desmontar() {
      if (contenedor) { contenedor.removeEventListener("click", pulsar); contenedor.removeEventListener("change", cambiar); }
      contenedor = null;
    },
    destruir() {
      if (contenedor) { contenedor.removeEventListener("click", pulsar); contenedor.removeEventListener("change", cambiar); }
      contenedor = null; destruido = true; secuencia++; lectura?.abort(); lectura = null; intencionPendiente = null;
    },
  };
}

// Atajo para consumidores que no conservan el controlador entre renders.
export function montarPreferenciasBolsas({ contenedor, fetchImpl = globalThis.fetch, participaciones = [] } = {}) {
  return crearControladorPreferenciasBolsas({ fetchImpl }).montar({ contenedor, participaciones });
}
