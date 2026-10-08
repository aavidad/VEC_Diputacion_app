import { LOCALIZACION_PORTAL, textoPortal, traducirPortal } from "./portal-i18n.js?v=20261007-pantallas-textos-final-v1";

export const ESQUEMA_REINCORPORACIONES_TITULAR = "vec.bolsa.rrhh.reincorporaciones_titular.v1";
const BASE = "/api/vec/bolsa/bolsas";
const REFERENCIA_RUTA = /^[A-Za-z0-9:._-]{1,512}$/u;
const FECHA_CIVIL = /^(\d{4})-(\d{2})-(\d{2})$/u;
const HUELLA = /^[a-f0-9]{64}$/u;
const POR_PAGINA = 6;
export const LIMITES_REINCORPORACIONES_TITULAR = Object.freeze({ maximoBytes: 512 * 1024, tiempoMs: 15_000 });

function limiteSeguro(valor, maximo) {
  return Number.isSafeInteger(valor) && valor > 0 ? Math.min(valor, maximo) : maximo;
}

function cancelarCuerpo(respuesta, lector) {
  try {
    const cancelacion = lector?.cancel?.() ?? respuesta?.body?.cancel?.();
    Promise.resolve(cancelacion).catch(() => {});
  } catch { /* Una cancelación defectuosa no habilita la respuesta. */ }
}

async function leerJSONAcotado(respuesta, signal, abortada, limites, finTiempo) {
  const maximoBytes = limiteSeguro(limites?.maximoBytes, LIMITES_REINCORPORACIONES_TITULAR.maximoBytes);
  const declarada = respuesta.headers?.get?.("content-length");
  if (declarada !== null && declarada !== undefined && (!/^(?:0|[1-9][0-9]*)$/u.test(declarada) || Number(declarada) > maximoBytes)) {
    cancelarCuerpo(respuesta);
    throw new Error("respuesta_excesiva");
  }
  if (!respuesta.body || typeof respuesta.body.getReader !== "function") throw new Error("respuesta_no_incremental");
  let lector;
  try { lector = respuesta.body.getReader(); }
  catch { cancelarCuerpo(respuesta); throw new Error("respuesta_no_incremental"); }
  if (!lector || typeof lector.read !== "function" || typeof lector.cancel !== "function") {
    cancelarCuerpo(respuesta, lector);
    throw new Error("respuesta_no_incremental");
  }
  const descodificador = new TextDecoder("utf-8", { fatal: true });
  let texto = "";
  let total = 0;
  const cancelar = () => cancelarCuerpo(respuesta, lector);
  signal.addEventListener("abort", cancelar, { once: true });
  const leerCuerpo = async () => {
    for (;;) {
      if (signal.aborted) throw new DOMException("", "AbortError");
      if (performance.now() >= finTiempo) throw new Error("tiempo_agotado");
      const parte = await lector.read();
      // reader.cancel() puede resolver read() con done=true antes de que gane
      // la promesa de aborto; ese cierre nunca convierte un JSON parcial en éxito.
      if (signal.aborted) throw new DOMException("", "AbortError");
      if (performance.now() >= finTiempo) throw new Error("tiempo_agotado");
      if (!parte || typeof parte.done !== "boolean") throw new Error("respuesta_incompatible");
      if (parte.done) break;
      if (!(parte.value instanceof Uint8Array)) throw new Error("respuesta_incompatible");
      total += parte.value.byteLength;
      if (total > maximoBytes) throw new Error("respuesta_excesiva");
      if (parte.value.byteLength === 0) continue;
      try { texto += descodificador.decode(parte.value, { stream: true }); }
      catch { throw new Error("respuesta_incompatible"); }
    }
    try { texto += descodificador.decode(); }
    catch { throw new Error("respuesta_incompatible"); }
    return JSON.parse(texto);
  };
  try {
    return await Promise.race([leerCuerpo(), abortada]);
  } catch (error) {
    cancelar();
    throw error;
  } finally {
    signal.removeEventListener("abort", cancelar);
    try { lector.releaseLock?.(); } catch { /* El cuerpo ya se ha descartado. */ }
  }
}

function segmentoRutaValido(valor) {
  return typeof valor === "string" && REFERENCIA_RUTA.test(valor) && valor !== "." && valor !== "..";
}

export function rutaReincorporacionesTitular(bolsa, participacion) {
  if (!segmentoRutaValido(bolsa) || !segmentoRutaValido(participacion)) return null;
  return `${BASE}/${bolsa}/candidatos/${participacion}/reincorporaciones-titular`;
}

function fechaValida(valor) {
  const partes = typeof valor === "string" ? FECHA_CIVIL.exec(valor) : null;
  if (!partes) return false;
  const fecha = new Date(0);
  fecha.setUTCFullYear(Number(partes[1]), Number(partes[2]) - 1, Number(partes[3]));
  return fecha.getUTCFullYear() === Number(partes[1]) && fecha.getUTCMonth() === Number(partes[2]) - 1
    && fecha.getUTCDate() === Number(partes[3]);
}

function referenciaValida(valor) {
  return typeof valor === "string" && valor.length > 0 && valor.length <= 512 && !/[<>\u0000-\u001f]/u.test(valor);
}

function itemValido(item) {
  return item && typeof item === "object" && !Array.isArray(item)
    && ["evento_ref", "expediente_ref", "relacion_ref", "recibo_ct_ref", "cese_evento_ref"].every((campo) => referenciaValida(item[campo]))
    && fechaValida(item.fecha_efectiva)
    && item.estado === "cese_aplicado"
    && (item.disponible_desde === null || fechaValida(item.disponible_desde))
    && (item.regla_version === null || (Number.isSafeInteger(item.regla_version) && item.regla_version > 0))
    && (item.regla_huella_sha256 === "" || (typeof item.regla_huella_sha256 === "string" && HUELLA.test(item.regla_huella_sha256)));
}

export async function consultarReincorporacionesTitular(bolsa, participacion, { fetchImpl = fetch, signal, limites = LIMITES_REINCORPORACIONES_TITULAR } = {}) {
  const ruta = rutaReincorporacionesTitular(bolsa, participacion);
  if (!ruta) return { ok: false, status: 400, mensaje: traducirPortal("reincorporacion_error_referencia") };
  const controlador = new AbortController();
  let cancelacionExterna = false;
  let rechazoAbortar;
  const abortada = new Promise((_, reject) => { rechazoAbortar = reject; });
  void abortada.catch(() => {});
  const cancelar = (externa = false) => {
    if (controlador.signal.aborted) return;
    cancelacionExterna = externa;
    controlador.abort();
    rechazoAbortar(new DOMException("", "AbortError"));
  };
  const cancelarPorSignal = () => cancelar(true);
  signal?.addEventListener?.("abort", cancelarPorSignal, { once: true });
  const tiempoMs = limiteSeguro(limites?.tiempoMs, LIMITES_REINCORPORACIONES_TITULAR.tiempoMs);
  const finTiempo = performance.now() + tiempoMs;
  const temporizador = setTimeout(() => cancelar(), tiempoMs);
  try {
    if (signal?.aborted) cancelar(true);
    if (controlador.signal.aborted) return { ok: false, status: 0, abortada: true, mensaje: "" };
    const peticion = Promise.resolve(fetchImpl(ruta, {
      method: "GET", credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error",
      referrerPolicy: "no-referrer", signal: controlador.signal, headers: { Accept: "application/json" },
    }));
    void peticion.then((respuestaTardia) => {
      if (controlador.signal.aborted) cancelarCuerpo(respuestaTardia);
    }, () => {});
    const respuesta = await Promise.race([peticion, abortada]);
    if (!respuesta.ok) {
      cancelarCuerpo(respuesta);
      const clave = ({ 401: "reincorporacion_error_401", 403: "reincorporacion_error_403", 404: "reincorporacion_error_404", 503: "reincorporacion_error_503" })[respuesta.status];
      return { ok: false, status: respuesta.status, mensaje: traducirPortal(clave || "reincorporacion_error_http", { estado: respuesta.status }) };
    }
    const datos = (await leerJSONAcotado(respuesta, controlador.signal, abortada, limites, finTiempo))?.data;
    if (datos?.esquema !== ESQUEMA_REINCORPORACIONES_TITULAR || !Array.isArray(datos.items)
      || datos.items.length > 100 || !datos.items.every(itemValido)) {
      return { ok: false, status: 0, mensaje: traducirPortal("reincorporacion_error_contrato") };
    }
    return { ok: true, datos: datos.items };
  } catch (error) {
    if (error?.name === "AbortError" && cancelacionExterna) return { ok: false, status: 0, abortada: true, mensaje: "" };
    const clave = ["respuesta_excesiva", "respuesta_no_incremental", "respuesta_incompatible"].includes(error?.message)
      || error?.name === "SyntaxError" ? "reincorporacion_error_contrato" : "reincorporacion_error_red";
    return { ok: false, status: 0, mensaje: traducirPortal(clave) };
  } finally {
    clearTimeout(temporizador);
    signal?.removeEventListener?.("abort", cancelarPorSignal);
  }
}

export async function cargarReincorporacionesTitularFicha(modal, { estado, renderizar, consultar = consultarReincorporacionesTitular, disponibilidad, renderizarAlIniciar = true }) {
  if (!modal?.candidato?.participacion_ref) return;
  modal.controladorReincorporaciones?.abort();
  const registro = disponibilidad && disponibilidad.bolsa_ref === estado.bolsaSeleccionada
    && disponibilidad.participacion_ref === modal.candidato.participacion_ref
    && ["disponible", "no_autorizado", "sin_montaje", "indisponible"].includes(disponibilidad.estado)
    ? disponibilidad : disponibilidad ? { estado: "indisponible" } : null;
  if (registro && registro.estado !== "disponible") {
    modal.reincorporacionesTitular = { carga: registro.estado === "indisponible" ? "metadatos" : "omitida",
      items: [], pagina: 0 };
    if (renderizarAlIniciar) renderizar();
    return;
  }
  const controlador = new AbortController();
  modal.controladorReincorporaciones = controlador;
  modal.reincorporacionesTitular = { carga: "cargando", items: [], pagina: 0 };
  if (renderizarAlIniciar) renderizar();
  const respuesta = await consultar(estado.bolsaSeleccionada, modal.candidato.participacion_ref, { signal: controlador.signal });
  if (controlador.signal.aborted || estado.modalFicha !== modal) return;
  modal.reincorporacionesTitular = respuesta.ok
    ? { carga: "listo", items: respuesta.datos, pagina: 0 }
    : { carga: [401, 403].includes(respuesta.status) ? "denegado"
      : respuesta.status === 404 || respuesta.status === 503 ? "pendiente" : "error",
      error: respuesta.mensaje, items: [], pagina: 0 };
  renderizar();
}

function fechaVisible(valor) {
  const [anio, mes, dia] = valor.split("-").map(Number);
  const fecha = new Date(0);
  fecha.setUTCFullYear(anio, mes - 1, dia);
  return new Intl.DateTimeFormat(LOCALIZACION_PORTAL, { dateStyle: "short", timeZone: "UTC" }).format(fecha);
}

export function renderizarReincorporacionesTitular({ estado = {}, escaparHTML }) {
  const t = (clave, variables) => textoPortal(`reincorporacion_${clave}`, variables);
  const carga = estado.carga || "cargando";
  if (carga === "omitida") return "";
  let contenido;
  if (carga === "cargando") contenido = `<p class="vacio-controlado" role="status" aria-busy="true">${t("cargando")}</p>`;
  else if (carga === "metadatos") contenido = `<p class="${estado.metadatosCargando ? "vacio-controlado" : "mensaje-error"}" role="${estado.metadatosCargando ? "status" : "alert"}">${estado.metadatosCargando ? textoPortal("txt_comprobando_acceso") : t("error_503")}</p><button type="button" class="boton-secundario" data-reincorporacion-accion="reintentar" ${estado.metadatosCargando ? "disabled" : ""}>${t("reintentar")}</button>`;
  else if (carga === "denegado") contenido = `<p class="mensaje-error" role="alert">${escaparHTML(estado.error || traducirPortal("reincorporacion_error_403"))}</p>`;
  else if (carga === "pendiente" || carga === "error") contenido = `<p class="mensaje-error" role="alert">${escaparHTML(estado.error || traducirPortal("reincorporacion_error_red"))}</p><button type="button" class="boton-secundario" data-reincorporacion-accion="reintentar">${t("reintentar")}</button>`;
  else if (!estado.items?.length) contenido = `<p class="vacio-controlado" role="status">${t("vacio")}</p>`;
  else {
    const total = estado.items.length;
    const paginas = Math.ceil(total / POR_PAGINA);
    const pagina = Math.min(Math.max(0, Number(estado.pagina) || 0), paginas - 1);
    const filas = estado.items.slice(pagina * POR_PAGINA, (pagina + 1) * POR_PAGINA).map((item) => {
      const fecha = escaparHTML(fechaVisible(item.fecha_efectiva));
      const disponible = item.disponible_desde ? `<time datetime="${escaparHTML(item.disponible_desde)}">${escaparHTML(fechaVisible(item.disponible_desde))}</time>` : t("sin_fecha_disponible");
      return `<tr><td><span class="estado-chip exito">${t("estado_cese_aplicado")}</span></td><td><time datetime="${escaparHTML(item.fecha_efectiva)}">${fecha}</time></td><td>${disponible}</td><td><code>${escaparHTML(item.recibo_ct_ref)}</code></td></tr>`;
    }).join("");
    const desde = pagina * POR_PAGINA + 1;
    const hasta = Math.min((pagina + 1) * POR_PAGINA, total);
    const paginacion = paginas > 1 ? `<nav class="paginacion-bolsa" aria-label="${t("paginacion")}"><span>${t("mostrando", { desde, hasta, total })}</span><button type="button" class="boton-secundario" data-reincorporacion-accion="pagina" data-pagina="${pagina - 1}" ${pagina === 0 ? "disabled" : ""}>${t("anterior")}</button><button type="button" class="boton-secundario" data-reincorporacion-accion="pagina" data-pagina="${pagina + 1}" ${pagina + 1 === paginas ? "disabled" : ""}>${t("siguiente")}</button></nav>` : `<p>${t("mostrando", { desde, hasta, total })}</p>`;
    contenido = `<div class="tabla-contenedor" tabindex="0" role="region" aria-label="${t("tabla")}"><table class="tabla-datos"><caption>${t("tabla")}</caption><thead><tr><th scope="col">${t("col_hecho")}</th><th scope="col">${t("col_fecha")}</th><th scope="col">${t("col_disponibilidad")}</th><th scope="col">${t("col_recibo")}</th></tr></thead><tbody>${filas}</tbody></table></div>${paginacion}`;
  }
  return `<section class="panel panel-separado" data-reincorporacion-raiz="true" aria-labelledby="reincorporacion-titulo"><div class="cabecera-panel"><h4 id="reincorporacion-titulo" tabindex="-1">${t("titulo")}</h4></div><div class="cuerpo-panel">${contenido}</div></section>`;
}

export function manejarClickReincorporacionesTitular(evento, { estado, renderizar, consultar, reintentarMetadatos } = {}) {
  const control = evento.target?.closest?.("[data-reincorporacion-accion]");
  if (!control || !estado?.modalFicha) return false;
  evento.preventDefault();
  if (control.dataset.reincorporacionAccion === "reintentar") {
    if (estado.modalFicha.reincorporacionesTitular?.carga === "metadatos") {
      if (typeof reintentarMetadatos === "function") void reintentarMetadatos();
      return true;
    }
    if (estado.modalFicha.reincorporacionesTitular?.carga === "denegado") return true;
    void cargarReincorporacionesTitularFicha(estado.modalFicha, { estado, renderizar, ...(consultar ? { consultar } : {}) });
  } else if (control.dataset.reincorporacionAccion === "pagina" && estado.modalFicha.reincorporacionesTitular) {
    estado.modalFicha.reincorporacionesTitular = { ...estado.modalFicha.reincorporacionesTitular, pagina: Math.max(0, Number(control.dataset.pagina) || 0) };
    renderizar();
  }
  return true;
}
