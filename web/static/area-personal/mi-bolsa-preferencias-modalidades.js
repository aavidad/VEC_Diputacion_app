import { localizacionAreaPersonal, traducir } from "./i18n.js";
import { escaparHTML } from "./vistas/comunes.js";

export const RUTA_PREFERENCIAS_MODALIDADES = "/api/vec/bolsa/mi-bolsa/preferencias?tipo=modalidades";
const MAXIMO_BYTES = 64 * 1024;
const CLAVES = new Set(["sustitucion", "vacante", "acumulacion_tareas", "programa", "relevo"]);
const INSTANTE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{6}Z$/u;
const CAMPOS_ESTADO = ["version", "modalidades_guardadas", "modalidades_disponibles", "recibo_ref", "registrado_en"];
const CAMPOS_FUENTE = ["catalogo_id", "catalogo_version", "catalogo_huella_sha256", "entrada_clave"];
const CAMPOS_RECIBO = ["recibo_ref", "version", "modalidades", "registrado_en", "reutilizado"];
const t = (clave, variables) => traducir(`areaPersonal.miBolsa.preferenciasModalidades.${clave}`, variables);
const objeto = (v) => v !== null && typeof v === "object" && !Array.isArray(v);
const fallo = (codigo) => Object.assign(new Error(codigo), { codigo });
const mismasClaves = (v, claves) => objeto(v) && Object.keys(v).sort().join("|") === [...claves].sort().join("|");
const instante = (v) => typeof v === "string" && INSTANTE.test(v) && !Number.isNaN(Date.parse(v));
const listaClaves = (v) => Array.isArray(v) && v.length <= 5 && v.every((clave) => CLAVES.has(clave)) && new Set(v).size === v.length;
const iguales = (a, b) => a === null || b === null ? a === b : a.length === b.length && a.every((clave) => b.includes(clave));
const copiar = (v) => v === null ? null : [...v];

export function validarEstadoPreferenciasModalidades(v) {
  if (!mismasClaves(v, CAMPOS_ESTADO) || !Number.isSafeInteger(v.version) || v.version < 0 ||
    !(v.modalidades_guardadas === null || listaClaves(v.modalidades_guardadas)) ||
    !Array.isArray(v.modalidades_disponibles) || v.modalidades_disponibles.length > 5 ||
    (v.recibo_ref !== null && (typeof v.recibo_ref !== "string" || !v.recibo_ref)) ||
    (v.registrado_en !== null && !instante(v.registrado_en))) throw fallo("respuesta");
  const vistas = new Set();
  for (const disponible of v.modalidades_disponibles) {
    if (!mismasClaves(disponible, ["clave", "fuente"]) || !CLAVES.has(disponible.clave) || vistas.has(disponible.clave) ||
      !mismasClaves(disponible.fuente, CAMPOS_FUENTE) || disponible.fuente.entrada_clave !== disponible.clave ||
      typeof disponible.fuente.catalogo_id !== "string" || !disponible.fuente.catalogo_id ||
      !Number.isSafeInteger(disponible.fuente.catalogo_version) || disponible.fuente.catalogo_version < 1 ||
      typeof disponible.fuente.catalogo_huella_sha256 !== "string" || !/^[a-f0-9]{64}$/u.test(disponible.fuente.catalogo_huella_sha256) ||
      t(`opciones.${disponible.clave}`) === `areaPersonal.miBolsa.preferenciasModalidades.opciones.${disponible.clave}`) throw fallo("respuesta");
    vistas.add(disponible.clave);
  }
  if (v.modalidades_guardadas?.some((clave) => !vistas.has(clave))) throw fallo("respuesta");
  return structuredClone(v);
}

export function validarReciboPreferenciasModalidades(v, peticion) {
  if (!mismasClaves(v, CAMPOS_RECIBO) || typeof v.recibo_ref !== "string" || !v.recibo_ref ||
    !Number.isSafeInteger(v.version) || v.version !== peticion.version_esperada + 1 ||
    !(v.modalidades === null || listaClaves(v.modalidades)) || !iguales(v.modalidades, peticion.modalidades) ||
    !instante(v.registrado_en) || typeof v.reutilizado !== "boolean") throw fallo("respuesta");
  return structuredClone(v);
}

async function solicitar(api, metodo, cuerpo, signal) {
  let respuesta;
  try {
    respuesta = await api(RUTA_PREFERENCIAS_MODALIDADES, {
      method: metodo, credentials: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer",
      headers: { Accept: "application/json", ...(cuerpo ? { "Content-Type": "application/json" } : {}) },
      ...(cuerpo ? { body: JSON.stringify(cuerpo) } : {}), ...(signal ? { signal } : {}),
    });
  } catch { throw fallo(metodo === "PUT" ? "incierto" : "error"); }
  if (respuesta?.status !== 200 && !(metodo === "PUT" && respuesta?.status === 201)) {
    throw fallo(({ 401: "autenticacion", 403: "denegado", 409: "conflicto", 422: "invalido", 503: "servicio" })[respuesta?.status] || "error");
  }
  let texto;
  try { texto = await respuesta.text(); } catch { throw fallo(metodo === "PUT" ? "incierto" : "respuesta"); }
  if (new TextEncoder().encode(texto).byteLength > MAXIMO_BYTES) throw fallo(metodo === "PUT" ? "incierto" : "respuesta");
  try { return JSON.parse(texto); } catch { throw fallo(metodo === "PUT" ? "incierto" : "respuesta"); }
}

export async function cargarPreferenciasModalidades({ api = globalThis.fetch, signal } = {}) {
  return validarEstadoPreferenciasModalidades(await solicitar(api, "GET", null, signal));
}

export async function guardarPreferenciasModalidades(peticion, { api = globalThis.fetch, signal } = {}) {
  if (!mismasClaves(peticion, ["modalidades", "version_esperada", "clave_idempotencia"]) ||
    !(peticion.modalidades === null || listaClaves(peticion.modalidades)) ||
    !Number.isSafeInteger(peticion.version_esperada) || peticion.version_esperada < 0 ||
    typeof peticion.clave_idempotencia !== "string" || !/^[A-Za-z0-9._:-]{8,128}$/u.test(peticion.clave_idempotencia)) throw fallo("respuesta");
  const respuesta = await solicitar(api, "PUT", peticion, signal);
  try { return validarReciboPreferenciasModalidades(respuesta, peticion); }
  catch { throw fallo("incierto"); }
}

function fecha(valor, idioma) {
  return new Intl.DateTimeFormat(idioma(), { dateStyle: "medium", timeStyle: "short", timeZone: "Europe/Madrid" }).format(new Date(valor));
}

export function renderizarPreferenciasModalidades(datos, { estado = "cargando", borrador = null, abierto = false, recibo = null, pendiente = false, verTodasEstado = "", textos = t, idioma = localizacionAreaPersonal } = {}) {
  const tr = (clave, vars) => escaparHTML(textos(clave, vars));
  const valido = datos && (estado === "correcto" || estado === "guardado" || estado === "incierto" || estado === "servicio");
  const resumen = !valido ? tr(estado === "cargando" ? "cargando" : "sinConsulta") : borrador === null ? tr("todas") : tr("resumenElegidas", { cuenta: borrador.length });
  const accionVerTodas = `<button type="button" class="boton-secundario" data-modalidades="ver-todas" ${verTodasEstado === "cargando" ? "disabled" : ""}>${tr(verTodasEstado === "cargando" ? "verTodasCargando" : "verTodas")}</button>`;
  const mensajeVerTodas = verTodasEstado && verTodasEstado !== "cargando" ? `<p role="${verTodasEstado === "verTodasMostradas" ? "status" : "alert"}">${tr(verTodasEstado)}</p>` : "";
  let contenido = "";
  if (!valido) {
    const codigo = ["autenticacion", "denegado", "conflicto", "invalido", "respuesta", "servicio", "incierto"].includes(estado) ? estado : "error";
    contenido = estado === "cargando" || estado === "guardando" ? `<p role="status">${tr(estado)}</p>` :
      `<div class="estado-error" role="alert"><p>${tr(codigo)}</p>${codigo === "denegado" || codigo === "autenticacion" ? "" : `<div class="fila-acciones"><button type="button" class="boton-secundario" data-modalidades="recargar">${tr("recargar")}</button>${accionVerTodas}</div>${mensajeVerTodas}`}</div>`;
  } else {
    const opciones = datos.modalidades_disponibles.map(({ clave }) => `<label class="opcion-check"><input type="checkbox" data-modalidades-opcion="${clave}" ${borrador?.includes(clave) ? "checked" : ""} ${pendiente ? "disabled" : ""}><span>${tr(`opciones.${clave}`)}</span></label>`).join("");
    const cambios = !iguales(borrador, datos.modalidades_guardadas);
    const resultado = recibo || (datos.recibo_ref && datos.registrado_en ? { recibo_ref: datos.recibo_ref, registrado_en: datos.registrado_en } : null);
    const aviso = borrador !== null && borrador.length === 0 ? `<p class="preferencias-modalidades__aviso" role="status">${tr("avisoNinguna")}</p>` : "";
    const mensaje = estado === "incierto" || estado === "servicio" ? `<p role="alert">${tr(estado)}</p>` : "";
    contenido = `<p class="preferencias-modalidades__leyenda">${tr("leyenda")}</p>
      <fieldset class="preferencias-modalidades__grupo"><legend>${tr("elegirVista")}</legend>
        <label class="opcion-check"><input type="radio" name="preferencias-modalidades-modo" value="todas" data-modalidades-modo ${borrador === null ? "checked" : ""} ${pendiente ? "disabled" : ""}><span>${tr("todas")}</span></label>
        <label class="opcion-check"><input type="radio" name="preferencias-modalidades-modo" value="elegir" data-modalidades-modo ${borrador !== null ? "checked" : ""} ${pendiente ? "disabled" : ""}><span>${tr("elegir")}</span></label>
      </fieldset>${borrador !== null ? `<fieldset class="preferencias-modalidades__grupo"><legend>${tr("modalidades")}</legend>${opciones}</fieldset>` : ""}
      ${aviso}${mensaje}${mensajeVerTodas}<div class="fila-acciones"><button type="button" class="boton-primario" data-modalidades="${pendiente ? "reintentar" : "guardar"}" ${!pendiente && (!cambios || estado !== "correcto" && estado !== "guardado") ? "disabled" : ""}>${tr(pendiente ? "reintentarGuardado" : "guardar")}</button>${estado === "incierto" || estado === "servicio" ? `<button type="button" class="boton-secundario" data-modalidades="recargar">${tr("recargar")}</button>` : ""}${accionVerTodas}</div>
      ${resultado ? `<p class="preferencias-modalidades__recibo" role="status">${tr(estado === "guardado" ? "guardado" : "ultimoGuardado")} · ${escaparHTML(fecha(resultado.registrado_en, idioma))}</p>` : ""}`;
  }
  return `<section class="panel preferencias-modalidades"><details ${abierto || !valido && estado !== "cargando" ? "open" : ""} data-modalidades-detalles><summary><h3 class="preferencias-modalidades__titulo">${tr("titulo")}</h3><span class="preferencias-modalidades__resumen">${resumen}</span></summary><div class="panel-contenido">${contenido}</div></details></section>`;
}

export function crearControladorPreferenciasModalidades({ host, api = globalThis.fetch, textos = t, idioma = localizacionAreaPersonal, onVerTodas = () => {} } = {}) {
  let datos = null, borrador = null, recibo = null, estado = "cargando", abierto = false, pendiente = null;
  let lectura = null, escritura = null, verTodas = null, verTodasEstado = "", secuencia = 0, destruido = false, montado = false;
  const pintar = () => { if (!destruido && host) host.innerHTML = renderizarPreferenciasModalidades(datos, { estado, borrador, abierto, recibo, pendiente: Boolean(pendiente), verTodasEstado, textos, idioma }); };
  async function consultarTodas() {
    if (destruido || verTodas) return;
    const controller = new AbortController(); verTodas = controller; verTodasEstado = "cargando"; pintar();
    try {
      await onVerTodas({ signal: controller.signal });
      if (!destruido && !controller.signal.aborted) { verTodasEstado = "verTodasMostradas"; pintar(); }
    } catch (error) {
      if (!destruido && !controller.signal.aborted) {
        verTodasEstado = ["verTodasAutenticacion", "verTodasDenegado"].includes(error?.codigo) ? error.codigo : "verTodasError";
        abierto = true; pintar();
      }
    } finally { if (verTodas === controller) verTodas = null; }
  }
  async function cargar() {
    if (destruido || lectura || escritura) return;
    const controller = new AbortController(); lectura = controller;
    const propia = ++secuencia; estado = "cargando"; pintar();
    try {
      const nuevo = await cargarPreferenciasModalidades({ api, signal: controller.signal });
      if (destruido || propia !== secuencia) return;
      datos = nuevo; borrador = copiar(nuevo.modalidades_guardadas); recibo = null; estado = "correcto";
      if (pendiente) {
        if (nuevo.version > pendiente.version_esperada && iguales(nuevo.modalidades_guardadas, pendiente.modalidades) && nuevo.recibo_ref && nuevo.registrado_en) {
          recibo = { recibo_ref: nuevo.recibo_ref, registrado_en: nuevo.registrado_en }; pendiente = null; estado = "guardado";
        } else if (nuevo.version !== pendiente.version_esperada) { pendiente = null; estado = "conflicto"; }
        else estado = "incierto";
      }
      pintar();
    } catch (error) { if (!destruido && propia === secuencia && !controller.signal.aborted) { estado = error.codigo || "error"; pintar(); } }
    finally { if (lectura === controller) lectura = null; }
  }
  async function guardar() {
    if (destruido || escritura || lectura || !datos) return;
    if (!pendiente) {
      if (estado !== "correcto" && estado !== "guardado") return;
      if (iguales(borrador, datos.modalidades_guardadas) || (borrador && borrador.some((clave) => !datos.modalidades_disponibles.some((o) => o.clave === clave)))) return;
      if (typeof globalThis.crypto?.randomUUID !== "function") { estado = "error"; pintar(); return; }
      pendiente = { modalidades: copiar(borrador), version_esperada: datos.version, clave_idempotencia: globalThis.crypto.randomUUID() };
    }
    const peticion = pendiente, controller = new AbortController(); escritura = controller; estado = "guardando"; pintar();
    try {
      const nuevo = await guardarPreferenciasModalidades(peticion, { api, signal: controller.signal });
      if (destruido || escritura !== controller) return;
      pendiente = null; recibo = nuevo;
      datos = { ...datos, version: nuevo.version, modalidades_guardadas: copiar(nuevo.modalidades), recibo_ref: nuevo.recibo_ref, registrado_en: nuevo.registrado_en };
      borrador = copiar(nuevo.modalidades); estado = "guardado"; pintar();
    } catch (error) {
      if (["conflicto", "invalido", "denegado", "autenticacion"].includes(error.codigo)) pendiente = null;
      if (!destruido && !controller.signal.aborted) { estado = error.codigo || "incierto"; pintar(); }
    } finally { if (escritura === controller) escritura = null; }
  }
  function pulsar(evento) {
    const boton = evento.target?.closest?.("[data-modalidades]");
    if (!boton || destruido) return;
    const accion = boton.dataset.modalidades;
    if (accion === "ver-todas") { void consultarTodas(); return; }
    if (accion === "recargar") { abierto = true; void cargar(); }
    if (accion === "guardar" || accion === "reintentar") void guardar();
  }
  function cambiar(evento) {
    if (destruido || pendiente || escritura || !datos) return;
    const el = evento.target;
    if (el?.hasAttribute?.("data-modalidades-modo")) {
      borrador = el.value === "todas" ? null : el.value === "elegir" ? (borrador ?? []) : borrador;
    } else if (el?.hasAttribute?.("data-modalidades-opcion") && CLAVES.has(el.dataset.modalidadesOpcion)) {
      const elegidas = new Set(borrador ?? []);
      if (el.checked) elegidas.add(el.dataset.modalidadesOpcion); else elegidas.delete(el.dataset.modalidadesOpcion);
      borrador = [...elegidas];
    } else return;
    estado = "correcto"; abierto = true; pintar();
    const selector = el.hasAttribute("data-modalidades-modo") ? `[data-modalidades-modo][value="${el.value}"]` : `[data-modalidades-opcion="${el.dataset.modalidadesOpcion}"]`;
    host?.querySelector(selector)?.focus?.();
  }
  function alternar(evento) { if (evento.target?.matches?.("[data-modalidades-detalles]")) abierto = evento.target.open; }
  return {
    montar() {
      if (!host || destruido) return null;
      if (!montado) { host.addEventListener("click", pulsar); host.addEventListener("change", cambiar); host.addEventListener("toggle", alternar, true); montado = true; }
      pintar(); if (!datos && !lectura) void cargar(); return this;
    },
    destruir() {
      if (montado) { host.removeEventListener("click", pulsar); host.removeEventListener("change", cambiar); host.removeEventListener("toggle", alternar, true); }
      destruido = true; montado = false; secuencia++; lectura?.abort(); escritura?.abort(); verTodas?.abort(); lectura = null; escritura = null; verTodas = null; pendiente = null;
    },
  };
}
