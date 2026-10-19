import { consultarJSON, ErrorConsultaJSON } from "../../../comun/http.js?v=VEC_WEB_VERSION";
import { cargarTextos, reintentarTextos } from "../../../comun/textos.js";

const RUTA = "/api/vec/meritos/rrhh/revision";
const ESTADOS = new Set(["declarado", "pendiente", "acreditado", "rechazado"]);
const TIPOS = new Set(["titulacion", "curso_asistencia", "curso_superacion", "experiencia", "idioma", "otro"]);
const REF = /^[A-Za-z0-9:._-]{1,256}$/u;

export function crearClienteRevisionMeritosRRHH({ fetchImpl = globalThis.fetch } = {}) {
  return Object.freeze({
    async consultar({ solicitudRef, cursor = "", signal } = {}) {
      if (!/^solicitud_inscripcion_[0-9a-f]{64}$/u.test(solicitudRef) || (cursor && !REF.test(cursor))) {
        throw new TypeError("meritos.revision.selector_invalido");
      }
      const q = new URLSearchParams({ solicitud_ref: solicitudRef, limite: "25" });
      if (cursor) q.set("cursor", cursor);
      const { data } = await consultarJSON(`${RUTA}?${q}`, { fetchImpl, signal, limiteBytes: 262144 });
      if (data?.esquema !== "vec.meritos.revision-rrhh.v1" || !Array.isArray(data.hechos)
        || data.hechos.length > 25 || !Number.isSafeInteger(data.total) || data.total < data.hechos.length
        || typeof data.cursor_siguiente !== "string" || (data.cursor_siguiente && !REF.test(data.cursor_siguiente))
        || !data.recibo_acceso?.auditoria_ref || !Number.isFinite(Date.parse(data.recibo_acceso.registrada_en))
        || !data.hechos.every((h) => h && REF.test(h.hecho_ref) && Number.isSafeInteger(h.version) && h.version > 0
          && TIPOS.has(h.tipo) && ESTADOS.has(h.estado) && typeof h.denominacion === "string"
          && h.denominacion.length > 0 && h.denominacion.length <= 512 && h.vigencia
          && /^\d{4}-\d{2}-\d{2}$/u.test(h.vigencia.desde) && Array.isArray(h.evidencias)
          && h.evidencias.length <= 32)) throw new TypeError("meritos.revision.respuesta_invalida");
      return data;
    },
  });
}

/** La misma solicitud autoriza el candidato y permite volver a su revisión. */
export async function montarRevisionMeritosRRHH({ raiz, solicitudRef, escaparHTML: e,
  alVolver, nombre = "", signal, cliente = crearClienteRevisionMeritosRRHH() } = {}) {
  if (!raiz || typeof e !== "function" || typeof alVolver !== "function") throw new TypeError("meritos.revision.montaje_invalido");
  const textos = await cargarTextos("meritos-revision-rrhh").catch(() => reintentarTextos("meritos-revision-rrhh"));
  if (signal?.aborted) return Object.freeze({ desmontar() {} });
  let activa = !signal?.aborted;
  let controlador;
  let revision = 0;
  let estado = "cargando";
  let datos = null;
  let cursor = new URLSearchParams(globalThis.location?.search || "").get("meritos_cursor") || "";
  const t = (clave, vars) => e(textos.traducir(clave, vars));
  const fecha = (valor) => e(textos.fecha(valor, { dateStyle: "short", timeZone: "Europe/Madrid" }));

  function pintar() {
    if (!activa) return;
    const mensaje = estado === "listo" ? "" : `<div class="cuerpo-panel" role="${estado === "error" || estado === "denegado" ? "alert" : "status"}"${estado === "cargando" ? ' aria-busy="true"' : ""}>
      <p>${t(`estado_${estado}`)}</p>${estado === "error" ? `<button type="button" class="boton-secundario" data-meritos-revision-reintentar>${t("reintentar")}</button>` : ""}</div>`;
    const filas = datos?.hechos.map((h) => `<article class="panel">
      <header class="cabecera-panel"><h3>${e(h.denominacion)}</h3><span class="estado-chip ${h.estado === "acreditado" ? "exito" : h.estado === "rechazado" ? "peligro" : "advertencia"}">${t(`estado_hecho_${h.estado}`)}</span></header>
      <div class="cuerpo-panel"><dl class="datos-clave"><div><dt>${t("tipo")}</dt><dd>${t(`tipo_${h.tipo}`)}</dd></div>
        <div><dt>${t("fecha_desde")}</dt><dd>${fecha(h.vigencia.desde)}</dd></div>${h.vigencia.hasta ? `<div><dt>${t("fecha_hasta")}</dt><dd>${fecha(h.vigencia.hasta)}</dd></div>` : ""}
        ${Number.isSafeInteger(h.horas) ? `<div><dt>${t("horas")}</dt><dd>${e(textos.numero(h.horas))}</dd></div>` : ""}</dl>
        ${h.evidencias.length ? `<details><summary>${t("documentos", { cuenta: textos.numero(h.evidencias.length) })}</summary><ul>${h.evidencias.map((_doc, i) => `<li>${t("documento", { numero: textos.numero(i + 1) })}</li>`).join("")}</ul></details>` : `<p class="dato-secundario">${t("sin_documentos")}</p>`}</div></article>`).join("") || "";
    raiz.innerHTML = `<section class="panel"><header class="cabecera-panel"><div><h2 tabindex="-1" data-meritos-revision-titulo>${t("titulo")}</h2>${nombre ? `<p>${e(nombre)}</p>` : ""}</div>
      <div class="acciones-vista"><button type="button" class="boton-secundario" data-meritos-revision-volver>${t("volver")}</button><details><summary aria-label="${t("ayuda_boton")}">?</summary><p>${t("ayuda")}</p></details></div></header>${mensaje}</section>
      ${estado === "listo" ? `<div>${filas}</div><nav class="paginacion-bolsa" aria-label="${t("paginacion")}"><button type="button" class="boton-secundario" data-meritos-revision-primera>${t("total", { cuenta: datos.total })}</button>${datos.cursor_siguiente ? `<button type="button" class="boton-secundario" data-meritos-revision-siguiente>${t("siguiente")}</button>` : ""}</nav>` : ""}`;
  }

  function conservarCursor() {
    if (!globalThis.location?.href) return;
    const url = new URL(globalThis.location.href);
    if (cursor) url.searchParams.set("meritos_cursor", cursor); else url.searchParams.delete("meritos_cursor");
    globalThis.history?.replaceState(null, "", url.pathname + url.search + url.hash);
  }
  async function consultar() {
    controlador?.abort(); controlador = new AbortController(); const actual = ++revision;
    datos = null; estado = "cargando"; pintar();
    try {
      const pagina = await cliente.consultar({ solicitudRef, cursor, signal: controlador.signal });
      if (!activa || actual !== revision) return;
      datos = pagina; estado = pagina.hechos.length ? "listo" : "vacio";
    } catch (error) {
      if (!activa || actual !== revision) return;
      datos = null; estado = error instanceof ErrorConsultaJSON && [401, 403].includes(error.estado) ? "denegado" : "error";
      console.error("meritos_revision_consulta", { tipo: error?.name });
    }
    pintar(); raiz.querySelector("[data-meritos-revision-titulo]")?.focus({ preventScroll: true });
  }
  const click = (evento) => {
    const boton = evento.target.closest("button");
    if (!boton || !raiz.contains(boton)) return;
    if (boton.hasAttribute("data-meritos-revision-volver")) alVolver();
    else if (boton.hasAttribute("data-meritos-revision-reintentar")) void consultar();
    else if (boton.hasAttribute("data-meritos-revision-primera")) { cursor = ""; conservarCursor(); void consultar(); }
    else if (boton.hasAttribute("data-meritos-revision-siguiente") && datos?.cursor_siguiente) { cursor = datos.cursor_siguiente; conservarCursor(); void consultar(); }
  };
  const desmontar = () => { activa = false; revision++; controlador?.abort(); raiz.removeEventListener("click", click); signal?.removeEventListener("abort", desmontar); raiz.replaceChildren(); };
  raiz.addEventListener("click", click); signal?.addEventListener("abort", desmontar, { once: true });
  if (activa) void consultar();
  return Object.freeze({ desmontar });
}
