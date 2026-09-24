import { crearTraductorPersonal } from "../modulos/personal/i18n.js";

export const API_ORGANIZACION_HISTORICA = "/api/vec/personal/organizacion-historica";
const tPersonal = crearTraductorPersonal();
const t = (clave, variables) => tPersonal(`organizacion_${clave}`, variables);
const SECCIONES = ["unidades", "puestos_tipo", "dotaciones", "plazas", "puestos_individuales", "vinculos"];
const COBERTURA = new Set(["completa", "parcial", "sin_datos"]);
const PARAMETROS = new Set(["vigente_en", "conocido_en", "unidad_clave", "version_rpt_ref", "version_plantilla_ref", "limite", "cursor"]);
const NOMBRES = Object.freeze({
  unidades: "historyUnits", puestos_tipo: "historyTypes", dotaciones: "historyAllocations",
  plazas: "historyPlazas", puestos_individuales: "historyPosts", vinculos: "historyLinks",
});
const esc = (valor) => String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
  .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");

export function validarPaginaHistorica(payload) {
  if (new TextEncoder().encode(JSON.stringify(payload)).byteLength > 1024 * 1024) {
    throw new Error("respuesta histórica demasiado grande");
  }
  const data = payload?.data, pagina = data?.pagina, evidencia = data?.evidencia;
  if (!pagina || !pagina.selector || !pagina.cobertura || !evidencia ||
    typeof pagina.selector.vigente_en !== "string" ||
    typeof pagina.selector.conocido_en !== "string" ||
    !SECCIONES.every((clave) => COBERTURA.has(pagina.cobertura[clave]) &&
      (Array.isArray(pagina[clave]) || pagina[clave] === null)) ||
    typeof evidencia.recibo_ref !== "string" || !evidencia.recibo_ref ||
    typeof evidencia.decision_ref !== "string" || !evidencia.decision_ref ||
    typeof evidencia.efecto_ref !== "string" || !evidencia.efecto_ref ||
    typeof evidencia.auditoria_ref !== "string" || !evidencia.auditoria_ref ||
    !/^[a-f0-9]{64}$/.test(evidencia.consumo_huella_sha256) ||
    typeof evidencia.consultada_en !== "string" || !Number.isFinite(Date.parse(evidencia.consultada_en)) ||
    (pagina.cursor_siguiente !== undefined && typeof pagina.cursor_siguiente !== "string")) {
    throw new Error("respuesta histórica no válida");
  }
  const colecciones = Object.fromEntries(SECCIONES.map((clave) => [clave, pagina[clave] ?? []]));
  if (SECCIONES.reduce((n, clave) => n + colecciones[clave].length, 0) > 100 ||
    !SECCIONES.every((clave) => colecciones[clave].every((fila) =>
      fila && typeof fila === "object" && fila.traza && typeof fila.traza.id === "string" &&
      Number.isInteger(fila.traza.version) && fila.traza.version > 0))) {
    throw new Error("filas históricas no válidas");
  }
  return { pagina: { ...pagina, ...colecciones }, evidencia };
}

export function crearClienteHistorico(fetchImpl = globalThis.fetch, timeoutMs = 10000) {
  return {
    async consultar(filtros, signal) {
      const params = new URLSearchParams();
      for (const [clave, valor] of Object.entries(filtros)) {
        if (!PARAMETROS.has(clave)) throw new Error("filtro histórico no permitido");
        if (valor !== "" && valor !== undefined) params.set(clave, String(valor));
      }
      const controller = new AbortController();
      const abortar = () => controller.abort();
      signal?.addEventListener("abort", abortar, { once: true });
      const timer = setTimeout(abortar, timeoutMs);
      try {
        const response = await fetchImpl(`${API_ORGANIZACION_HISTORICA}?${params}`, {
          credentials: "same-origin", redirect: "error", cache: "no-store", signal: controller.signal,
        });
        if (!response.ok) throw Object.assign(new Error(`HTTP ${response.status}`), { status: response.status });
        return validarPaginaHistorica(await response.json());
      } finally {
        clearTimeout(timer);
        signal?.removeEventListener("abort", abortar);
      }
    },
  };
}

export function filtrosHistoricos(campos) {
  const fecha = /^([0-3]\d)\/([0-1]\d)\/(\d{4})$/.exec(campos.vigente_en);
  const conocida = /^([0-3]\d)\/([0-1]\d)\/(\d{4}) ([0-2]\d):([0-5]\d)$/.exec(campos.conocido_en);
  if (!fecha || !conocida) {
    throw new Error("fechas históricas no válidas");
  }
  const vigente = `${fecha[3]}-${fecha[2]}-${fecha[1]}`;
  if (new Date(`${vigente}T12:00:00Z`).toISOString().slice(0, 10) !== vigente) {
    throw new Error("fecha de efectos no válida");
  }
  const local = `${conocida[3]}-${conocida[2]}-${conocida[1]}T${conocida[4]}:${conocida[5]}`;
  const instante = new Date(`${local}:00Z`);
  if (!Number.isFinite(instante.getTime()) ||
      instante.toISOString().slice(0, 16) !== local.slice(0, 16)) {
    throw new Error("instante histórico no válido");
  }
  const conocido = instante.toISOString().replace(/Z$/, "000Z");
  const filtro = {
    vigente_en: vigente,
    conocido_en: conocido,
    limite: 100,
  };
  if (campos.unidad_clave) filtro.unidad_clave = campos.unidad_clave;
  if (campos.version_rpt_ref) filtro.version_rpt_ref = campos.version_rpt_ref.trim();
  if (campos.version_plantilla_ref) filtro.version_plantilla_ref = campos.version_plantilla_ref.trim();
  return filtro;
}

const fechaUTC = (iso) => new Intl.DateTimeFormat("es-ES", {
  dateStyle: "medium", timeStyle: "short", timeZone: "UTC", hourCycle: "h23",
}).format(new Date(iso));
const fechaCivil = (iso) => new Intl.DateTimeFormat("es-ES", {
  dateStyle: "medium", timeZone: "UTC",
}).format(new Date(`${iso}T12:00:00Z`));
const numero = (n) => new Intl.NumberFormat("es-ES").format(n);
const dato = (v) => v === "" || v == null ? t("historyNotProvided") : String(v);
const tipoUnidad = (valor) => ({
  delegacion: t("delegacion"), centro: t("centro"), puesto_responsabilidad: t("puesto"),
})[valor] ?? dato(valor);
const columnas = {
  unidades: [["historyName", (f) => f.etiqueta], ["historyType", (f) => tipoUnidad(f.tipo)], ["historyParent", (f) => f.padre_id], ["historySource", (f) => f.traza.fuente_ref]],
  puestos_tipo: [["historyName", (f) => f.denominacion], ["historyCode", (f) => f.codigo_fuente], ["historyParent", (f) => f.unidad_id], ["historyClass", (f) => f.clasificacion_ref]],
  dotaciones: [["historyCode", (f) => f.puesto_tipo_id], ["historyQuantity", (f) => numero(f.cantidad)], ["historySource", (f) => f.traza.fuente_ref]],
  plazas: [["historyCode", (f) => f.codigo_fuente], ["historyClass", (f) => f.clasificacion_ref], ["historyParent", (f) => f.unidad_id], ["historyStructural", (f) => f.estado_estructural]],
  puestos_individuales: [["historyCode", (f) => f.codigo_fuente], ["historyType", (f) => f.puesto_tipo_id], ["historyParent", (f) => f.unidad_id], ["historyStructural", (f) => f.estado_estructural]],
  vinculos: [["historyPlazaCode", (f) => f.plaza_id], ["historyPostCode", (f) => f.puesto_id], ["historySource", (f) => f.traza.fuente_ref]],
};

export function iniciarHistorico(cliente = crearClienteHistorico()) {
  if (!document.getElementById?.("history-form")) return null;
  const buscar = (selector) => document.querySelector(selector);
  const form = buscar("#history-form"), state = buscar("#history-state"), results = buscar("#history-results");
  const unidad = buscar("#history-unit"), tipo = buscar("#history-kind"), more = buscar("#history-more");
  const hoy = new Date();
  const fechaHoy = `${String(hoy.getUTCDate()).padStart(2, "0")}/${String(hoy.getUTCMonth() + 1).padStart(2, "0")}/${hoy.getUTCFullYear()}`;
  buscar("#history-valid-date").value = fechaHoy;
  buscar("#history-known-at").value = `${fechaHoy} ${String(hoy.getUTCHours()).padStart(2, "0")}:${String(hoy.getUTCMinutes()).padStart(2, "0")}`;
  buscar("#history-search").disabled = false;
  state.textContent = t("historyReady");
  buscar("#history-authorization").textContent = t("historyPending");
  let filtros, datos, controlador, secuencia = 0, navegacionBloqueada = () => false;
  const seleccionar = (nombre, enfocar = false) => {
    if (navegacionBloqueada()) return;
    for (const clave of ["history", "catalog"]) {
      const activo = clave === nombre, tab = buscar(`#tab-${clave}`), panel = buscar(`#${clave === "history" ? "history" : "catalog"}-panel`);
      tab.setAttribute("aria-selected", String(activo));
      tab.tabIndex = activo ? 0 : -1;
      panel.hidden = !activo;
      if (activo && enfocar) tab.focus();
    }
    buscar("#editor").hidden = true;
  };
  for (const nombre of ["history", "catalog"]) buscar(`#tab-${nombre}`).onclick = () => seleccionar(nombre);
  buscar(".org-tabs").onkeydown = (evento) => {
    if (!["ArrowLeft", "ArrowRight", "Home", "End"].includes(evento.key)) return;
    evento.preventDefault();
    const actual = buscar("#tab-history").getAttribute("aria-selected") === "true" ? 0 : 1;
    const indice = evento.key === "Home" ? 0 : evento.key === "End" ? 1 : 1 - actual;
    seleccionar(["history", "catalog"][indice], true);
  };
  const renderizar = () => {
    if (!datos) return;
    const filas = datos.pagina[tipo.value], cabeceras = columnas[tipo.value];
    buscar("#history-count").textContent = t("historyCount", { total: numero(filas.length) });
    buscar("#history-caption").textContent = t(NOMBRES[tipo.value]);
    buscar("#history-head").innerHTML = `<tr>${cabeceras.map(([clave]) => `<th scope="col">${esc(t(clave))}</th>`).join("")}</tr>`;
    buscar("#history-rows").innerHTML = filas.length
      ? filas.map((fila) => `<tr>${cabeceras.map(([, valor]) => `<td>${esc(dato(valor(fila)))}</td>`).join("")}</tr>`).join("")
      : `<tr><td colspan="${cabeceras.length}">${esc(t("historyEmpty"))}</td></tr>`;
  };
  tipo.onchange = renderizar;
  const pintarResultado = () => {
    results.hidden = false;
    buscar("#history-panel").classList.toggle("has-data", SECCIONES.some((clave) => datos.pagina[clave].length > 0));
    buscar("#history-authorization").textContent = t("historyReadOnly");
    state.hidden = false;
    state.className = "solo-lectura";
    state.textContent = t("historyLoaded");
    buscar("#history-coverage").innerHTML = SECCIONES.map((clave) => {
      const cobertura = datos.pagina.cobertura[clave], estado = cobertura === "completa" ? "historyComplete" : cobertura === "parcial" ? "historyPartial" : "historyNoSource";
      return `<div class="org-history-kpi"><strong>${numero(datos.pagina[clave].length)}</strong><span>${esc(t(NOMBRES[clave]))}</span><small>${esc(t(estado))}</small></div>`;
    }).join("");
    const rpt = dato(datos.pagina.version_rpt_ref), plantilla = dato(datos.pagina.version_plantilla_ref);
    buscar("#history-meta").innerHTML = [
      t("historyEffective", { fecha: fechaCivil(filtros.vigente_en) }),
      t("historyKnown", { fecha: fechaUTC(filtros.conocido_en) }),
      t("historyRPT", { version: rpt }), t("historyPlantilla", { version: plantilla }),
      t("historyReceipt", { referencia: datos.evidencia.recibo_ref }),
    ].map((valor) => `<span>${esc(valor)}</span>`).join("");
    more.hidden = !datos.pagina.cursor_siguiente;
    renderizar();
  };
  const consultar = async (cursor = "") => {
    controlador?.abort();
    controlador = new AbortController();
    const actual = ++secuencia;
    state.hidden = false;
    state.className = "org-state";
    state.textContent = t("historyLoading");
    if (!cursor) results.hidden = true;
    if (!cursor) buscar("#history-panel").classList.remove("has-data");
    buscar("#history-search").disabled = true;
    more.disabled = true;
    try {
      const respuesta = await cliente.consultar({ ...filtros, ...(cursor ? { cursor } : {}) }, controlador.signal);
      if (actual !== secuencia) return;
      const selector = respuesta.pagina.selector;
      if (selector.vigente_en !== filtros.vigente_en ||
        new Date(selector.conocido_en).getTime() !== new Date(filtros.conocido_en).getTime() ||
        (filtros.unidad_clave && selector.unidad_clave !== filtros.unidad_clave) ||
        (filtros.version_rpt_ref && selector.version_rpt_ref !== filtros.version_rpt_ref) ||
        (filtros.version_plantilla_ref && selector.version_plantilla_ref !== filtros.version_plantilla_ref) ||
        (cursor && selector.cursor !== cursor)) throw new Error("corte histórico distinto");
      if (cursor && datos) {
        for (const clave of SECCIONES) respuesta.pagina[clave] = [...datos.pagina[clave], ...respuesta.pagina[clave]];
      }
      datos = respuesta;
      pintarResultado();
    } catch (error) {
      if (actual !== secuencia) return;
      results.hidden = true;
      buscar("#history-panel").classList.remove("has-data");
      buscar("#history-authorization").textContent = error.status === 401 || error.status === 403 ? t("historyDeniedPill") : t("historyPending");
      state.className = "org-state error";
      state.textContent = error.status === 401 || error.status === 403 ? t("historyDenied") : t("historyError");
    } finally {
      if (actual === secuencia) { buscar("#history-search").disabled = false; more.disabled = false; }
    }
  };
  form.onsubmit = (evento) => {
    evento.preventDefault();
    try {
      filtros = filtrosHistoricos({
        unidad_clave: unidad.value,
        vigente_en: buscar("#history-valid-date").value,
        conocido_en: buscar("#history-known-at").value,
        version_rpt_ref: buscar("#history-version-rpt").value,
        version_plantilla_ref: buscar("#history-version-plantilla").value,
      });
    } catch {
      results.hidden = true;
      buscar("#history-panel").classList.remove("has-data");
      buscar("#history-authorization").textContent = t("historyPending");
      state.hidden = false;
      state.className = "org-state error";
      state.textContent = t("historyInvalid");
      return;
    }
    void consultar();
  };
  more.onclick = () => { if (datos?.pagina.cursor_siguiente) void consultar(datos.pagina.cursor_siguiente); };
  return {
    establecerBloqueo(comprobar) { navegacionBloqueada = comprobar; },
    establecerUnidades(unidades) {
      const actual = unidad.value;
      unidad.innerHTML = `<option value="">${esc(t("historyChooseUnit"))}</option>` +
        unidades.map((u) => `<option value="${esc(u.clave)}">${esc(u.etiqueta)}</option>`).join("");
      unidad.value = unidades.some((u) => u.clave === actual) ? actual : "";
      unidad.disabled = false;
    },
    seleccionar,
  };
}
