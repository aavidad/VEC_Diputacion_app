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
          credentials: "omit", redirect: "error", cache: "no-store", signal: controller.signal,
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
   buscar("#history-access-state").textContent = t("historyPending");
  let filtros, datos, controlador, secuencia = 0, navegacionBloqueada = () => false;
  const pestanas = ["history", "import", "catalog"];
  const seleccionar = (nombre, enfocar = false) => {
    if (navegacionBloqueada()) return;
    for (const clave of pestanas) {
      const activo = clave === nombre, tab = buscar(`#tab-${clave}`), panel = buscar(`#${clave}-panel`);
      tab.setAttribute("aria-selected", String(activo));
      tab.tabIndex = activo ? 0 : -1;
      panel.hidden = !activo;
      if (activo && enfocar) tab.focus();
    }
    buscar("#editor").hidden = true;
  };
  for (const nombre of pestanas) buscar(`#tab-${nombre}`).onclick = () => seleccionar(nombre);
  buscar(".org-tabs").onkeydown = (evento) => {
    if (!["ArrowLeft", "ArrowRight", "Home", "End"].includes(evento.key)) return;
    evento.preventDefault();
    const actual = pestanas.findIndex((clave) => buscar(`#tab-${clave}`).getAttribute("aria-selected") === "true");
    const indice = evento.key === "Home" ? 0 : evento.key === "End" ? pestanas.length - 1 :
      (actual + (evento.key === "ArrowRight" ? 1 : pestanas.length - 1)) % pestanas.length;
    seleccionar(pestanas[indice], true);
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
     buscar("#history-access-state").textContent = t("historyReadOnly");
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
       buscar("#history-access-state").textContent = error.status === 401 || error.status === 403 ? t("historyDeniedPill") : t("historyPending");
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
       buscar("#history-access-state").textContent = t("historyPending");
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

export const RUTAS_IMPORTACION = Object.freeze({
  preparar: `${API_ORGANIZACION_HISTORICA}/importaciones/preparar`,
  conciliar: `${API_ORGANIZACION_HISTORICA}/importaciones/conciliar`,
  publicar: `${API_ORGANIZACION_HISTORICA}/importaciones/publicar`,
});
const LIMITE_ARCHIVO_IMPORTACION = 8 * 1024 * 1024;
const HUELLA = /^[a-f0-9]{64}$/;
const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/;
const UUID_V4 = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;
const CLASES_HECHO = new Set(["nodo", "puesto_tipo", "dotacion", "plaza", "puesto_individual", "vinculo"]);
const CLASES_DECISION = new Set(["unidad", "clasificacion", "puesto_tipo", "dotacion", "plaza", "puesto_individual", "vinculo"]);
const CLAVES_MANIFIESTO = new Set(["tipo", "version_ref", "version_revision", "version_previa_ref", "fuente_ref", "fuente_version", "fuente_huella_sha256", "documento_ref", "custodia_ref", "diccionario_ref", "acto_ref", "aprobada_en", "publicada_en", "efectos_desde", "efectos_hasta", "catalogo_unidades", "catalogo_clasificaciones"]);
const CLAVES_CATALOGO = new Set(["id", "version", "revision", "huella_sha256"]);
const CLAVES_HECHO = new Set(["clase", "hecho_ref", "revision", "fila_fuente_ref", "pagina_fuente", "unidad_ref", "vigente_desde", "vigente_hasta", "codigo_fuente", "codigo_datos_reserva_fuente", "denominacion", "clasificacion_ref", "catalogo_entrada_clave", "tipo_unidad", "padre_ref", "tipo_ref", "tipo_revision", "plaza_ref", "plaza_revision", "puesto_ref", "puesto_revision", "cantidad", "reconciliacion", "regimen_ref", "forma_provision_ref", "nivel_destino", "estado_estructural", "dotacion_presupuestaria"]);
const CLAVES_DECISION = new Set(["fila_fuente_ref", "clase", "destino_ref", "resultado", "motivo", "evidencia_ref"]);
const objetoExacto = (valor, claves) => valor && typeof valor === "object" && !Array.isArray(valor) &&
  Object.keys(valor).every((clave) => claves.has(clave));
const cadena = (valor) => typeof valor === "string" && valor.length >= 3 && valor.length <= 256 &&
  valor === valor.trim() && !/[\u0000-\u001f\u007f]/.test(valor);
const entero = (valor) => Number.isSafeInteger(valor) && valor >= 1;
const fechaCivilValida = (valor) => typeof valor === "string" && /^\d{4}-\d{2}-\d{2}$/.test(valor) &&
  Number.isFinite(Date.parse(`${valor}T12:00:00Z`)) && new Date(`${valor}T12:00:00Z`).toISOString().slice(0, 10) === valor;
const catalogoImportacionValido = (c) => objetoExacto(c, CLAVES_CATALOGO) && cadena(c.id) &&
  entero(c.version) && entero(c.revision) && HUELLA.test(c.huella_sha256);

export function validarPaqueteImportacion(valor) {
  if (!objetoExacto(valor, new Set(["manifiesto", "hechos"])) ||
    !objetoExacto(valor.manifiesto, CLAVES_MANIFIESTO) ||
    !Array.isArray(valor.hechos) || valor.hechos.length < 1 || valor.hechos.length > 3000) {
    throw new Error("paquete de importación no válido");
  }
  const m = valor.manifiesto;
  if (!["rpt", "plantilla"].includes(m.tipo) || !UUID.test(m.version_ref) ||
    !entero(m.version_revision) || !cadena(m.fuente_ref) ||
    typeof m.fuente_version !== "string" || !m.fuente_version.trim() || m.fuente_version.length > 160 ||
    !HUELLA.test(m.fuente_huella_sha256) || !catalogoImportacionValido(m.catalogo_unidades) ||
    !catalogoImportacionValido(m.catalogo_clasificaciones) ||
    ["aprobada_en", "publicada_en", "efectos_desde", "efectos_hasta"].some((clave) => m[clave] && !fechaCivilValida(m[clave])) ||
    ["documento_ref", "custodia_ref", "diccionario_ref", "acto_ref"].some((clave) => m[clave] && !cadena(m[clave])) ||
    (m.version_previa_ref && !UUID.test(m.version_previa_ref))) {
    throw new Error("manifiesto de importación no válido");
  }
  const vistos = new Set();
  for (const hecho of valor.hechos) {
    if (!objetoExacto(hecho, CLAVES_HECHO) || !CLASES_HECHO.has(hecho.clase) ||
      !UUID.test(hecho.hecho_ref) || !entero(hecho.revision) ||
      !cadena(hecho.fila_fuente_ref) || !cadena(hecho.unidad_ref) ||
      !fechaCivilValida(hecho.vigente_desde) ||
      (hecho.vigente_hasta && !fechaCivilValida(hecho.vigente_hasta))) {
      throw new Error("hecho de importación no válido");
    }
    const clave = `${hecho.clase}\0${hecho.hecho_ref}`;
    if (vistos.has(clave)) throw new Error("hecho repetido");
    vistos.add(clave);
  }
  return structuredClone(valor);
}

export function validarDecisionesImportacion(valor) {
  if (!objetoExacto(valor, new Set(["decisiones"])) || !Array.isArray(valor.decisiones) ||
    valor.decisiones.length < 1 || valor.decisiones.length > 1000) {
    throw new Error("decisiones de importación no válidas");
  }
  const vistos = new Set();
  for (const d of valor.decisiones) {
    if (!objetoExacto(d, CLAVES_DECISION) || !CLASES_DECISION.has(d.clase) ||
      !cadena(d.fila_fuente_ref) || !cadena(d.evidencia_ref) ||
      typeof d.motivo !== "string" || !d.motivo.trim() || d.motivo.length > 2048 ||
      !["vinculada", "pendiente", "descartada"].includes(d.resultado) ||
      (d.resultado === "vinculada" ? !cadena(d.destino_ref) : d.destino_ref !== undefined && d.destino_ref !== "")) {
      throw new Error("decisión de conciliación no válida");
    }
    const clave = `${d.clase}\0${d.fila_fuente_ref}`;
    if (vistos.has(clave)) throw new Error("decisión repetida");
    vistos.add(clave);
  }
  return structuredClone(valor.decisiones);
}

export function validarReciboImportacion(payload, operacion) {
  const r = payload?.data, body = JSON.parse(operacion.cuerpo);
  if (!r || !cadena(r.recibo_ref) || !cadena(r.lote_ref) || r.fase !== operacion.fase ||
    r.clave_idempotencia !== operacion.clave ||
    r.revision_anterior !== body.revision_esperada || r.revision_nueva !== body.revision_esperada + 1 ||
    !HUELLA.test(r.material_huella_sha256) || r.fuente_huella_sha256 !== body.manifiesto.fuente_huella_sha256 ||
    !cadena(r.actor_ref) || !cadena(r.decision_ref) || !cadena(r.auditoria_ref) ||
    typeof r.registrado_en !== "string" || !Number.isFinite(Date.parse(r.registrado_en)) ||
    typeof r.replay !== "boolean" ||
    (operacion.fase !== "preparar" && r.lote_ref !== body.lote_ref) ||
    (operacion.fase === "preparar" && r.estado !== "preparacion_no_autoritativa") ||
    (operacion.fase === "conciliar" && !["conciliacion_pendiente", "conciliada"].includes(r.estado)) ||
    (operacion.fase === "publicar" && r.estado !== "publicada")) {
    throw new Error("recibo de importación no válido");
  }
  return r;
}

export function crearClienteImportacion(fetchImpl = globalThis.fetch, timeoutMs = 15000) {
  return {
    async enviar(operacion) {
      if (!RUTAS_IMPORTACION[operacion.fase] || !UUID_V4.test(operacion.clave) ||
        new TextEncoder().encode(operacion.cuerpo).byteLength > LIMITE_ARCHIVO_IMPORTACION) {
        throw Object.assign(new Error("operación inválida"), { rechazado: true });
      }
      const controller = new AbortController(), timer = setTimeout(() => controller.abort(), timeoutMs);
      let respuesta;
      try {
        respuesta = await fetchImpl(RUTAS_IMPORTACION[operacion.fase], {
          method: "POST", credentials: "omit", redirect: "error", cache: "no-store",
          headers: { "Content-Type": "application/json", "Idempotency-Key": operacion.clave },
          body: operacion.cuerpo, signal: controller.signal,
        });
        if ([400, 401, 403, 404, 405, 409].includes(respuesta.status)) {
          throw Object.assign(new Error(`HTTP ${respuesta.status}`), {
            conflicto: respuesta.status === 409,
            denegado: respuesta.status === 401 || respuesta.status === 403,
            rechazado: respuesta.status !== 409,
          });
        }
        if (!respuesta.ok) throw new Error(`HTTP ${respuesta.status}`);
        return validarReciboImportacion(await respuesta.json(), operacion);
      } catch (error) {
        if (error.conflicto || error.rechazado) throw error;
        throw Object.assign(new Error("respuesta de importación incierta"), { incierto: true });
      } finally {
        clearTimeout(timer);
      }
    },
  };
}

export const formatearFechaReciboImportacion = (iso) => new Intl.DateTimeFormat("es-ES", {
  dateStyle: "short", timeStyle: "medium", timeZone: "Europe/Madrid", hourCycle: "h23",
}).format(new Date(iso));
const claseTexto = Object.freeze({
  nodo: "importClassNode", puesto_tipo: "importClassType", dotacion: "importClassAllocation",
  plaza: "importClassPlaza", puesto_individual: "importClassPost", vinculo: "importClassLink",
});
const RECUENTOS_IMPORTACION = new Set(["importFacts", "importDecisions", ...Object.values(claseTexto)]);
export function formatearRecuentoImportacion(clave, total) {
  if (!RECUENTOS_IMPORTACION.has(clave) || !Number.isSafeInteger(total) || total < 0) {
    throw new TypeError("recuento de importación no válido");
  }
  const forma = new Intl.PluralRules("es-ES").select(total) === "one" ? "One" : "Many";
  return t(`${clave}${forma}`, { total: new Intl.NumberFormat("es-ES", { useGrouping: true }).format(total) });
}
export function renderizarResumenImportacion(m, hashPaquete, hechos) {
  const frases = [
    t("importSource", { fuente: m.fuente_ref }),
    t("importVersion", { version: `${m.tipo} · ${m.version_ref} · r${m.version_revision}` }),
    t("importSourceHash", { huella: m.fuente_huella_sha256 }),
    t("importPackageHash", { huella: hashPaquete }),
    t("importCatalogUnits", { ...m.catalogo_unidades, huella: m.catalogo_unidades.huella_sha256 }),
    t("importCatalogClasses", { ...m.catalogo_clasificaciones, huella: m.catalogo_clasificaciones.huella_sha256 }),
    formatearRecuentoImportacion("importFacts", hechos.length),
  ];
  const conteos = new Map();
  for (const h of hechos) conteos.set(h.clase, (conteos.get(h.clase) ?? 0) + 1);
  for (const [clase, total] of conteos) frases.push(formatearRecuentoImportacion(claseTexto[clase], total));
  return frases.map((frase) => `<span>${esc(frase)}</span>`).join("");
}
async function leerArchivoImportacion(archivo) {
  if (!archivo || archivo.size > LIMITE_ARCHIVO_IMPORTACION) throw new Error("archivo demasiado grande");
  const bytes = await archivo.arrayBuffer();
  const texto = new TextDecoder("utf-8", { fatal: true }).decode(bytes);
  const hash = await crypto.subtle.digest("SHA-256", bytes);
  return {
    valor: JSON.parse(texto),
    huella: [...new Uint8Array(hash)].map((v) => v.toString(16).padStart(2, "0")).join(""),
  };
}

export function iniciarImportacion(cliente = crearClienteImportacion()) {
  if (!document.getElementById?.("import-panel")) return null;
  const q = (selector) => document.querySelector(selector);
  const panel = q("#import-panel"), estado = q("#import-state"), revision = q("#import-review");
  const paqueteInput = q("#import-file"), decisionesInput = q("#decisions-file");
  let paquete, decisiones, recibo, pendiente, conflicto = false, secuenciaPaquete = 0, secuenciaDecisiones = 0;
  const bloqueado = () => pendiente?.faseEnvio === "enviando" || pendiente?.faseEnvio === "incierto";
  const estadoVisible = (clave, tipo = "") => {
    estado.className = `org-state ${tipo}`.trim();
    estado.textContent = t(clave);
  };
  const actualizar = () => {
    const enEnvio = bloqueado();
    q("#choose-import").disabled = Boolean(recibo) || enEnvio || conflicto;
    q("#import-prepare").disabled = !paquete || Boolean(recibo) || enEnvio || conflicto || Boolean(pendiente);
    const listo = Boolean(recibo) && recibo.fase === "preparar" ||
      Boolean(recibo) && recibo.fase === "conciliar" && recibo.estado === "conciliacion_pendiente";
    q("#choose-decisions").disabled = !listo || enEnvio || conflicto || Boolean(pendiente);
    q("#import-conciliate").disabled = !decisiones || !listo || enEnvio || conflicto || Boolean(pendiente);
    q("#import-publish").disabled = true;
    q("#import-stage").textContent = recibo?.fase === "conciliar" ? t("importStageConciliated") :
      recibo?.fase === "preparar" ? t("importStagePrepared") : t("importStageInitial");
  };
  const pintarRecibo = (r) => {
    const mensajes = [
      t("importSaved"), t("importReceipt", { recibo: r.recibo_ref }),
      t("importLot", { lote: r.lote_ref }),
      t("importRevision", { revision: numero(r.revision_nueva) }),
      t(r.estado === "preparacion_no_autoritativa" ? "importStatePreparation" :
        r.estado === "conciliacion_pendiente" ? "importStatePending" : "importStateConciliated"),
      t("importReceiptDate", { fecha: formatearFechaReciboImportacion(r.registrado_en) }),
    ];
    const div = document.createElement("div");
    div.textContent = mensajes.join(" · ");
    q("#import-receipts").append(div);
  };
  const mostrarRevision = (fase, cuerpo) => {
    if (bloqueado() || conflicto || pendiente) return;
    const texto = JSON.stringify(cuerpo);
    if (new TextEncoder().encode(texto).byteLength > LIMITE_ARCHIVO_IMPORTACION) {
      estadoVisible("importTooLarge", "error");
      return;
    }
    pendiente = { fase, clave: crypto.randomUUID(), cuerpo: texto, faseEnvio: "revision" };
    revision.hidden = false;
    revision.innerHTML = `<strong>${esc(t(fase === "preparar" ? "importReviewPrepare" : "importReviewConciliate"))}</strong>` +
      `<p>${esc(t("importOperation", { fase, clave: pendiente.clave }))}</p>` +
      `<div class="org-actions"><button type="button" id="import-confirm">${esc(t("importConfirm"))}</button>` +
      `<button type="button" id="import-cancel" class="org-secondary">${esc(t("importCancel"))}</button></div>`;
    q("#import-confirm").onclick = () => void enviar();
    q("#import-cancel").onclick = () => { pendiente = undefined; revision.hidden = true; actualizar(); };
    actualizar();
  };
  const enviar = async () => {
    if (!pendiente || pendiente.faseEnvio === "enviando") return;
    pendiente.faseEnvio = "enviando";
    revision.hidden = true;
    estadoVisible("importSending");
    actualizar();
    try {
      const nuevo = await cliente.enviar(pendiente);
      recibo = nuevo;
      pendiente = undefined;
      if (nuevo.fase === "conciliar") decisiones = undefined;
      pintarRecibo(nuevo);
      estadoVisible("importSaved", "success");
    } catch (error) {
      if (error.incierto) {
        pendiente.faseEnvio = "incierto";
        estadoVisible("importUncertain", "error");
        revision.hidden = false;
        revision.innerHTML = `<button type="button" id="import-retry">${esc(t("importRetryExact"))}</button>`;
        q("#import-retry").onclick = () => void enviar();
      } else {
        pendiente = undefined;
        conflicto = Boolean(error.conflicto);
        estadoVisible(error.conflicto ? "importConflict" : error.denegado ? "importDenied" : "importRejected", "error");
      }
    }
    actualizar();
  };
  q("#choose-import").onclick = () => paqueteInput.click();
  paqueteInput.onchange = async () => {
    if (recibo || bloqueado() || conflicto) return;
    const actual = ++secuenciaPaquete;
    paquete = undefined;
    actualizar();
    q("#import-preview").hidden = true;
    const archivo = paqueteInput.files?.[0];
    if (!archivo) return;
    q("#import-file-name").textContent = archivo.name;
    try {
      if (archivo.size > LIMITE_ARCHIVO_IMPORTACION) throw new Error("demasiado grande");
      const { valor, huella } = await leerArchivoImportacion(archivo);
      if (actual !== secuenciaPaquete) return;
      paquete = validarPaqueteImportacion(valor);
      q("#import-preview").innerHTML = renderizarResumenImportacion(paquete.manifiesto, huella, paquete.hechos);
      q("#import-preview").hidden = false;
      estadoVisible("importPackageReady");
    } catch (error) {
      if (actual !== secuenciaPaquete) return;
      estadoVisible(error.message === "demasiado grande" || error.message === "archivo demasiado grande" ? "importTooLarge" : "importBadPackage", "error");
    }
    actualizar();
  };
  q("#choose-decisions").onclick = () => decisionesInput.click();
  decisionesInput.onchange = async () => {
    if (!recibo || bloqueado() || conflicto) return;
    const actual = ++secuenciaDecisiones;
    decisiones = undefined;
    actualizar();
    q("#decisions-preview").hidden = true;
    const archivo = decisionesInput.files?.[0];
    if (!archivo) return;
    q("#decisions-file-name").textContent = archivo.name;
    try {
      if (archivo.size > LIMITE_ARCHIVO_IMPORTACION) throw new Error("demasiado grande");
      const { valor } = await leerArchivoImportacion(archivo);
      if (actual !== secuenciaDecisiones) return;
      decisiones = validarDecisionesImportacion(valor);
      q("#decisions-preview").innerHTML = `<span>${esc(formatearRecuentoImportacion("importDecisions", decisiones.length))}</span>`;
      q("#decisions-preview").hidden = false;
      estadoVisible("importStagePrepared");
    } catch (error) {
      if (actual !== secuenciaDecisiones) return;
      estadoVisible(error.message === "demasiado grande" || error.message === "archivo demasiado grande" ? "importTooLarge" : "importBadDecisions", "error");
    }
    actualizar();
  };
  q("#import-prepare").onclick = () => {
    if (!paquete || recibo || bloqueado() || conflicto) return;
    mostrarRevision("preparar", { revision_esperada: 0, manifiesto: paquete.manifiesto, hechos: paquete.hechos });
  };
  q("#import-conciliate").onclick = () => {
    if (!paquete || !decisiones || !recibo || bloqueado() || conflicto) return;
    mostrarRevision("conciliar", { lote_ref: recibo.lote_ref, revision_esperada: recibo.revision_nueva,
      manifiesto: paquete.manifiesto, decisiones });
  };
  q("#import-publish").disabled = true;
  estadoVisible("importReady");
  actualizar();
  return Object.freeze({ bloqueado });
}
