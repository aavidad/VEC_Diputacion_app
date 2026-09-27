import { crearTraductorAuditoria } from "./i18n.js?v=20260928-rrhh-auditoria";
import { LOCALIZACION_PORTAL, ZONA_HORARIA_PORTAL, formatearNumeroPortal } from "../../portal-i18n.js?v=20260926-huecos-rrhh-v2";

const escapar = (v) => String(v ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
  .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
const texto = (v, n = 512) => typeof v === "string" && v.length <= n && !/[\x00-\x1f\x7f]/u.test(v);
const fecha = (v) => typeof v === "string" && /^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(?:\.\d+)?Z$/u.test(v) && Number.isFinite(Date.parse(v));
const digest = (v) => v == null || v === "" || typeof v === "string" && /^[a-f0-9]{64}$/iu.test(v);
const formatoFecha = new Intl.DateTimeFormat(LOCALIZACION_PORTAL, { dateStyle: "medium", timeStyle: "medium", timeZone: ZONA_HORARIA_PORTAL });

function proyeccion(v) {
  if (!v || typeof v !== "object" || Array.isArray(v) || Object.keys(v).length > 20
    || Object.entries(v).some(([k, s]) => !/^[a-z][a-z0-9_.-]{0,79}$/u.test(k) || !texto(s, 500))) throw new TypeError("proyección inválida");
  return Object.freeze({ ...v });
}

/** Valida toda la página antes de representar cualquier registro. */
export function validarRespuestaAuditoria(v) {
  if (!v || !Array.isArray(v.registros) || v.registros.length > 100 || typeof v.siguiente_cursor !== "string"
    || v.siguiente_cursor.length > 512 || /[\s\\/?%*]/u.test(v.siguiente_cursor)) throw new TypeError("respuesta de Auditoría inválida");
  const ids = new Set();
  const registros = v.registros.map((r) => {
    if (!r || !texto(r.id, 128) || !r.id || ids.has(r.id) || !texto(r.modulo_id, 128)
      || !texto(r.accion, 128) || !texto(r.actor_ref) || !fecha(r.ocurrido_en)
      || !texto(r.resultado, 80) || !texto(r.expediente_ref) || !texto(r.recibo_ref)
      || !digest(r.antes_sha256) || !digest(r.despues_sha256) || !texto(r.motivo, 500)
      || !texto(r.fuente, 128) || typeof r.datos_disponibles !== "boolean") throw new TypeError("registro de Auditoría inválido");
    const antes = proyeccion(r.antes), despues = proyeccion(r.despues);
    if (!r.datos_disponibles && (Object.keys(antes).length || Object.keys(despues).length)) throw new TypeError("proyección contradictoria");
    ids.add(r.id);
    return Object.freeze({ ...r, antes, despues });
  });
  return Object.freeze({ registros: Object.freeze(registros), siguienteCursor: v.siguiente_cursor || "" });
}

function bloqueValores(t, clave, valores, huella) {
  const filas = Object.entries(valores).map(([k, v]) => `<div><dt>${escapar(k)}</dt><dd>${escapar(v)}</dd></div>`).join("");
  return `<section class="auditoria-proyeccion"><h5>${escapar(t(clave))}</h5>
    ${filas ? `<dl>${filas}</dl>` : `<p>${escapar(t("sin_valores"))}</p>`}
    ${huella ? `<p class="auditoria-huella">${escapar(t("huella"))} <code>${escapar(huella)}</code></p>` : ""}</section>`;
}

function fila(t, r) {
  return `<tr><td><time datetime="${escapar(r.ocurrido_en)}">${escapar(formatoFecha.format(new Date(r.ocurrido_en)))}</time></td>
    <td>${escapar(r.actor_ref)}</td><td>${escapar(r.accion)}</td><td><span class="auditoria-resultado">${escapar(r.resultado)}</span></td>
    <td><details><summary>${escapar(t("ver_cambio"))}</summary><div class="auditoria-detalle">
      <dl class="auditoria-metadatos">
        ${[["campo_modulo", r.modulo_id], ["campo_expediente", r.expediente_ref], ["campo_recibo", r.recibo_ref],
          ["campo_fuente", r.fuente], ["campo_motivo", r.motivo]].map(([k, v]) =>
            `<div><dt>${escapar(t(k))}</dt><dd>${escapar(v || t("sin_dato"))}</dd></div>`).join("")}
      </dl><div class="auditoria-comparacion">${bloqueValores(t, "antes", r.antes, r.antes_sha256)}
        ${bloqueValores(t, "despues", r.despues, r.despues_sha256)}</div>
      ${r.datos_disponibles ? "" : `<p class="auditoria-dato-ausente">${escapar(t("valores_no_disponibles"))}</p>`}
    </div></details></td></tr>`;
}

/** Marcado puro para verificar estados y escape sin consultar datos. */
export function renderizarVistaAuditoria({ estado = "no_configurado", ayudaAbierta = false, habilitada = false,
  expedienteRef = "", ejemplo = false, filtros = {}, registros = [], pagina = 1, siguienteCursor = "", puedeAnterior = false } = {}) {
  const t = crearTraductorAuditoria();
  const mensaje = t(`estado_${["no_configurado", "cargando_opciones", "esperando", "cargando", "disponible", "vacio", "denegado", "error", "invalido"].includes(estado) ? estado : "error"}`);
  const bloqueada = !habilitada;
  return `<section class="modulo-auditoria-rrhh" data-auditoria-vista data-estado="${escapar(estado)}">
    <header class="auditoria-cabecera"><h2>${escapar(t("titulo"))}</h2>
      <button type="button" class="boton-secundario auditoria-ayuda-boton" data-auditoria-ayuda aria-controls="auditoria-ayuda"
        aria-expanded="${ayudaAbierta}" aria-label="${escapar(t("ayuda_aria"))}">?</button></header>
    <section id="auditoria-ayuda" class="panel auditoria-ayuda" ${ayudaAbierta ? "" : "hidden"}>
      <div class="cabecera-panel"><h3>${escapar(t("ayuda_titulo"))}</h3></div>
      <div class="cuerpo-panel"><p>${escapar(t("ayuda_alcance"))}</p><p>${escapar(t("ayuda_lectura"))}</p></div></section>
    <section class="panel auditoria-filtro-panel" aria-labelledby="auditoria-filtros-titulo">
      <div class="cabecera-panel"><h3 id="auditoria-filtros-titulo">${escapar(t("filtros_titulo"))}</h3></div>
      <div class="cuerpo-panel">
        <p class="auditoria-expediente"><strong>${escapar(t("expediente"))}:</strong> ${escapar(expedienteRef || t("sin_expediente"))}</p>
        ${ejemplo ? `<p class="auditoria-ejemplo">${escapar(t("configuracion_ejemplo"))}</p>` : ""}
        <form data-auditoria-filtros class="auditoria-filtros">
        ${[["desde", "desde", "datetime-local", "required"], ["hasta", "hasta", "datetime-local", "required"]].map(([name, label, type, attrs]) =>
          `<label>${escapar(t(label))}<input name="${name}" type="${type}" ${attrs} value="${escapar(filtros[name])}" ${bloqueada ? "disabled" : ""}></label>`).join("")}
        <button type="submit" class="boton-primario" ${bloqueada || estado === "cargando" ? "disabled" : ""}>${escapar(t("consultar"))}</button>
      </form></div></section>
    <section class="panel auditoria-resultados" aria-labelledby="auditoria-resultados-titulo">
      <div class="cabecera-panel"><h3 id="auditoria-resultados-titulo">${escapar(t("resultados_titulo"))}</h3>
        <span class="estado-chip ${["denegado", "error"].includes(estado) ? "peligro" : estado === "disponible" ? "exito" : "aviso"}">${escapar(mensaje)}</span></div>
      <div class="cuerpo-panel"><p class="auditoria-estado-texto" role="status" aria-live="polite">${escapar(mensaje)}</p>
      ${estado === "error" && expedienteRef ? `<button type="button" class="boton-secundario auditoria-reintentar" data-auditoria-reintentar>${escapar(t("reintentar"))}</button>` : ""}
      ${estado === "disponible" ? `<div class="auditoria-tabla" role="region" tabindex="0" aria-label="${escapar(t("tabla_aria"))}">
        <table><thead><tr>${["fecha", "actor", "accion", "resultado", "detalle"].map((k) => `<th scope="col">${escapar(t(k))}</th>`).join("")}</tr></thead>
        <tbody>${registros.map((r) => fila(t, r)).join("")}</tbody></table></div>` : ""}
      ${["disponible", "vacio"].includes(estado) ? `<nav class="auditoria-paginacion" aria-label="${escapar(t("paginacion"))}">
        <button type="button" class="boton-secundario" data-auditoria-anterior ${puedeAnterior ? "" : "disabled"}>${escapar(t("anterior"))}</button>
        <span>${escapar(t("pagina", { numero: formatearNumeroPortal(pagina) }))}</span>
        <button type="button" class="boton-secundario" data-auditoria-siguiente ${siguienteCursor ? "" : "disabled"}>${escapar(t("siguiente"))}</button>
      </nav>` : ""}</div></section></section>`;
}

/** Expediente procede de navegación ya autorizada; opciones del GET autenticado. */
export function montarVistaAuditoria({ raiz, fuente, expedienteRef = "", anunciar = () => {}, registrarDesmontar } = {}) {
  if (!raiz?.replaceChildren || typeof anunciar !== "function" ||
    (fuente !== undefined && (typeof fuente.consultar !== "function" || typeof fuente.obtenerOpciones !== "function"))
    || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) throw new TypeError("vista de Auditoría no disponible");
  const t = crearTraductorAuditoria();
  const expedienteValido = typeof expedienteRef === "string" && expedienteRef.length > 0 && expedienteRef.length <= 512
    && !/[\x00-\x20\x7f*?%\\/]/u.test(expedienteRef) && !expedienteRef.includes("..");
  let habilitada = false, opciones = null;
  let activa = true, ayudaAbierta = false, estado = fuente && expedienteValido ? "cargando_opciones" : "no_configurado";
  let filtros = {}, registros = [], siguienteCursor = "", cursores = [""], controlador, secuencia = 0;
  const pintar = () => { if (activa) raiz.innerHTML = renderizarVistaAuditoria({
    estado, ayudaAbierta, habilitada, expedienteRef: expedienteValido ? expedienteRef : "",
    ejemplo: opciones?.es_ejemplo === true, filtros, registros, pagina: cursores.length,
    siguienteCursor, puedeAnterior: cursores.length > 1,
  }); };
  const cancelar = () => { ++secuencia; controlador?.abort(); controlador = undefined; };
  async function cargarOpciones() {
    cancelar(); controlador = new AbortController(); const signal = controlador.signal, actual = secuencia;
    try {
      const recibidas = await fuente.obtenerOpciones({ signal });
      if (!activa || signal.aborted || secuencia !== actual) return;
      if (!recibidas || typeof recibidas.finalidad_ref !== "string" || !recibidas.finalidad_ref
        || typeof recibidas.motivo_ref !== "string" || !recibidas.motivo_ref
        || typeof recibidas.es_ejemplo !== "boolean") throw new TypeError("opciones incompatibles");
      opciones = recibidas; habilitada = true; estado = "esperando"; pintar();
    } catch (error) {
      if (!activa || signal.aborted || secuencia !== actual) return;
      opciones = null; habilitada = false; registros = []; estado = error?.codigo === "denegado" ? "denegado" : "error"; pintar();
    }
  }
  async function consultar() {
    if (!habilitada) return;
    cancelar(); controlador = new AbortController(); const signal = controlador.signal, actual = secuencia;
    estado = "cargando"; registros = []; siguienteCursor = ""; pintar();
    try {
      const respuesta = validarRespuestaAuditoria(await fuente.consultar({
        expediente_ref: expedienteRef, actor_ref: "",
        desde: new Date(filtros.desde).toISOString(), hasta: new Date(filtros.hasta).toISOString(),
        finalidad_ref: opciones.finalidad_ref, motivo_ref: opciones.motivo_ref, cursor: cursores.at(-1),
      }, { signal }));
      if (!activa || signal.aborted || secuencia !== actual) return;
      registros = respuesta.registros; siguienteCursor = respuesta.siguienteCursor;
      estado = registros.length ? "disponible" : "vacio"; pintar(); anunciar(t(`estado_${estado}`), "info");
    } catch (error) {
      if (!activa || signal.aborted || secuencia !== actual) return;
      registros = []; siguienteCursor = ""; estado = error?.codigo === "denegado" ? "denegado" : "error";
      pintar(); anunciar(t(`estado_${estado}`), "error");
    }
  }
  const pulsar = (evento) => {
    if (evento.target.closest?.("[data-auditoria-ayuda]")) {
      ayudaAbierta = !ayudaAbierta; pintar(); raiz.querySelector("[data-auditoria-ayuda]")?.focus();
    } else if (evento.target.closest?.("[data-auditoria-siguiente]") && siguienteCursor) {
      cursores.push(siguienteCursor); void consultar();
    } else if (evento.target.closest?.("[data-auditoria-anterior]") && cursores.length > 1) {
      cursores.pop(); void consultar();
    } else if (evento.target.closest?.("[data-auditoria-reintentar]") && fuente && expedienteValido) {
      estado = "cargando_opciones"; pintar(); void cargarOpciones();
    }
  };
  const enviar = (evento) => {
    if (!evento.target.matches?.("[data-auditoria-filtros]")) return;
    evento.preventDefault(); if (!habilitada) return;
    const datos = new FormData(evento.target);
    filtros = Object.fromEntries(["desde", "hasta"].map((k) => [k, String(datos.get(k) || "").trim()]));
    cursores = [""];
    const inicio = Date.parse(filtros.desde), fin = Date.parse(filtros.hasta);
    if (!Number.isFinite(inicio) || !Number.isFinite(fin) || fin <= inicio || fin - inicio > 31 * 86400000) {
      cancelar(); registros = []; estado = "invalido"; pintar(); return;
    }
    void consultar();
  };
  const cambiar = (evento) => {
    if (!evento.target.matches?.("[data-auditoria-filtros] input") || !habilitada) return;
    cancelar(); registros = []; siguienteCursor = ""; cursores = [""]; estado = "esperando";
    const campo = evento.target.name;
    if (["desde", "hasta"].includes(campo)) filtros = { ...filtros, [campo]: evento.target.value };
    const inicio = evento.target.selectionStart, fin = evento.target.selectionEnd;
    pintar();
    const nuevo = raiz.querySelector?.(`[data-auditoria-filtros] [name="${campo}"]`);
    nuevo?.focus?.();
    try { if (inicio !== null && fin !== null) nuevo?.setSelectionRange?.(inicio, fin); } catch {}
  };
  raiz.addEventListener("click", pulsar); raiz.addEventListener("submit", enviar);
  raiz.addEventListener("input", cambiar); pintar();
  if (fuente && expedienteValido) void cargarOpciones();
  const desmontar = () => { if (!activa) return; activa = false; cancelar(); raiz.removeEventListener("click", pulsar);
    raiz.removeEventListener("submit", enviar); raiz.removeEventListener("input", cambiar); raiz.replaceChildren(); };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ desmontar });
}
