export const RUTA_CATALOGOS_REGISTRO_B2 = "/api/vec/personal/catalogos-registro-empleado";
const TIPOS = new Set(["regimen", "modalidad", "situacion", "clase_servicio"]);
const REF = /^[a-z][a-z0-9_:-]{2,159}$/u;
const UUID_V4 = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/u;
const HUELLA = /^[a-f0-9]{64}$/u;
const MAX_BYTES = 512 * 1024;
const CAMPOS_CAMBIO = new Set(["operacion", "tipo", "ref", "version", "revision", "denominacion", "huella_sha256", "vigente_desde", "vigente_hasta", "acto_ref"]);

export class ErrorCatalogosRegistroB2 extends Error {
  constructor(codigo, estado = 0) { super(codigo); this.name = "ErrorCatalogosRegistroB2"; this.codigo = codigo; this.estado = estado; }
}

function fechaCivil(valor) { const fecha = new Date(`${valor}T12:00:00Z`); return typeof valor === "string" && /^\d{4}-\d{2}-\d{2}$/u.test(valor) && Number.isFinite(fecha.getTime()) && fecha.toISOString().slice(0, 10) === valor; }
function entero(valor) { return Number.isSafeInteger(valor) && valor >= 1 && valor <= 2147483647; }
function nombre(valor) { return typeof valor === "string" && valor !== "" && valor === valor.trim() && new TextEncoder().encode(valor).length <= 256 && !/[\p{Cc}\p{Cf}]/u.test(valor); }
function entradaValida(e, organismo, tipo, estado = "") {
  return e && typeof e === "object" && e.organismo_ref === organismo && e.tipo === tipo && REF.test(e.ref) && entero(e.version) && entero(e.revision) &&
    nombre(e.denominacion) && HUELLA.test(e.huella_sha256) && fechaCivil(e.vigente_desde) && (e.vigente_hasta === "" || fechaCivil(e.vigente_hasta)) &&
    ["publicada", "retirada"].includes(e.estado) && (!estado || e.estado === estado);
}
function evidenciaValida(e, efecto, campoFecha) { return e && typeof e === "object" && REF.test(e.decision_ref) && REF.test(e.auditoria_ref) && HUELLA.test(e.consumo_huella_sha256) && (efecto === undefined || e.efecto_ref === efecto) && typeof e[campoFecha] === "string" && Number.isFinite(Date.parse(e[campoFecha])); }
async function leerJSON(respuesta, signal) {
  if (respuesta.headers?.get?.("content-type") !== "application/json; charset=utf-8") throw new ErrorCatalogosRegistroB2("tipo_respuesta_no_valido", respuesta.status);
  const declarado = respuesta.headers?.get?.("content-length");
  if (declarado !== null && declarado !== undefined && (!/^\d+$/u.test(declarado) || Number(declarado) > MAX_BYTES)) throw new ErrorCatalogosRegistroB2("respuesta_excesiva", respuesta.status);
  if (!respuesta.body?.getReader) throw new ErrorCatalogosRegistroB2("respuesta_no_incremental", respuesta.status);
  const lector = respuesta.body.getReader(); const trozos = []; let total = 0;
  try {
    while (true) {
      if (signal.aborted) throw new ErrorCatalogosRegistroB2("operacion_abortada");
      const parte = await lector.read(); if (parte.done) break;
      if (!(parte.value instanceof Uint8Array) || !parte.value.byteLength) throw new ErrorCatalogosRegistroB2("respuesta_incompatible", respuesta.status);
      total += parte.value.byteLength; if (total > MAX_BYTES || trozos.length >= 256) throw new ErrorCatalogosRegistroB2("respuesta_excesiva", respuesta.status);
      trozos.push(parte.value);
    }
  } catch (error) { await lector.cancel().catch(() => {}); throw error; }
  finally { lector.releaseLock?.(); }
  const bytes = new Uint8Array(total); let posicion = 0; for (const trozo of trozos) { bytes.set(trozo, posicion); posicion += trozo.byteLength; }
  try { return JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(bytes)); }
  catch { throw new ErrorCatalogosRegistroB2("json_invalido", respuesta.status); }
}

export async function calcularHuellaPublicacionCatalogoB2({ organismoRef, tipo, ref, version, revision, denominacion, vigenteDesde, vigenteHasta = "", actoRef } = {}) {
  if (!REF.test(organismoRef) || organismoRef.length > 127 || !TIPOS.has(tipo) || !REF.test(ref) || !entero(version) || revision !== 1 ||
      !nombre(denominacion) || !fechaCivil(vigenteDesde) || (vigenteHasta && (!fechaCivil(vigenteHasta) || vigenteHasta <= vigenteDesde)) || !REF.test(actoRef) ||
      typeof globalThis.crypto?.subtle?.digest !== "function") throw new TypeError("publicación de catálogo no válida");
  const partes = ["vec.personal.catalogo-registro-empleado.entrada.v1", organismoRef, tipo, ref, String(version), String(revision), denominacion, vigenteDesde, vigenteHasta, actoRef];
  const bytes = new TextEncoder().encode(partes.join("\n"));
  const huella = new Uint8Array(await globalThis.crypto.subtle.digest("SHA-256", bytes));
  return [...huella].map((valor) => valor.toString(16).padStart(2, "0")).join("");
}

export function crearClienteCatalogosRegistroB2({ fetchImpl = globalThis.fetch, plazoMs = 10_000 } = {}) {
  if (typeof fetchImpl !== "function" || !Number.isSafeInteger(plazoMs) || plazoMs < 1 || plazoMs > 30_000) throw new TypeError("cliente de catálogos no disponible");
  async function solicitar(url, opciones, externo, escritura) {
    const controlador = new AbortController(); const abortar = () => controlador.abort();
    externo?.addEventListener("abort", abortar, { once: true }); const temporizador = setTimeout(abortar, plazoMs);
    try {
      let respuesta;
      try { respuesta = await fetchImpl(url, { ...opciones, credentials: "omit", mode: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer", signal: controlador.signal }); }
      catch { throw new ErrorCatalogosRegistroB2(escritura ? "resultado_incierto" : "red_no_disponible"); }
      if (!respuesta || respuesta.redirected) throw new ErrorCatalogosRegistroB2(escritura ? "resultado_incierto" : "respuesta_incompatible");
      if (respuesta.status !== 200 && (!escritura || respuesta.status !== 201)) {
        let codigo = "estado_no_valido";
        if (respuesta.headers?.get?.("content-type") === "application/json; charset=utf-8") {
          try { const error = await leerJSON(respuesta, controlador.signal); if (typeof error?.error?.codigo === "string") codigo = error.error.codigo; } catch { /* Rechazo sin sobre válido. */ }
        } else await respuesta.body?.cancel?.().catch(() => {});
        if (escritura && ![400, 401, 403, 404, 409].includes(respuesta.status)) codigo = "resultado_incierto";
        throw new ErrorCatalogosRegistroB2(codigo, respuesta.status);
      }
      try { return { estado: respuesta.status, datos: await leerJSON(respuesta, controlador.signal) }; }
      catch (error) { if (escritura) throw new ErrorCatalogosRegistroB2("resultado_incierto", respuesta.status); throw error; }
    } finally { clearTimeout(temporizador); externo?.removeEventListener("abort", abortar); }
  }
  return Object.freeze({
    async listar({ tipo, estado = "", cursorRef = "", cursorVersion = 0, limite = 50, signal } = {}) {
      if (!TIPOS.has(tipo) || !["", "publicada", "retirada"].includes(estado) || !Number.isSafeInteger(limite) || limite < 1 || limite > 100 ||
          ((cursorRef === "") !== (cursorVersion === 0)) || (cursorRef && (!REF.test(cursorRef) || !entero(cursorVersion))) ||
          (signal !== undefined && (!signal || typeof signal.addEventListener !== "function" || typeof signal.aborted !== "boolean"))) throw new TypeError("consulta de catálogo no válida");
      if (signal?.aborted) throw new ErrorCatalogosRegistroB2("operacion_abortada");
      const query = new URLSearchParams({ tipo, limite: String(limite) }); if (estado) query.set("estado", estado);
      if (cursorRef) { query.set("cursor_ref", cursorRef); query.set("cursor_version", String(cursorVersion)); }
      const { datos } = await solicitar(`${RUTA_CATALOGOS_REGISTRO_B2}?${query}`, { method: "GET" }, signal, false);
      const p = datos?.data;
      if (!p || !REF.test(p.organismo_ref) || p.organismo_ref.length > 127 || !Array.isArray(p.entradas) || p.entradas.length > limite ||
          !p.entradas.every((e) => entradaValida(e, p.organismo_ref, tipo, estado)) ||
          (p.cursor_siguiente !== null && p.cursor_siguiente !== undefined && (!REF.test(p.cursor_siguiente.ref) || !entero(p.cursor_siguiente.version))) ||
          !evidenciaValida(p.evidencia, `${p.organismo_ref}:${tipo}`, "consultada_en")) throw new ErrorCatalogosRegistroB2("respuesta_incompatible", 200);
      return Object.freeze({ organismoRef: p.organismo_ref, entradas: Object.freeze(p.entradas.map((e) => Object.freeze({ ...e }))), cursorSiguiente: p.cursor_siguiente ? Object.freeze({ ref: p.cursor_siguiente.ref, version: p.cursor_siguiente.version }) : null, evidencia: Object.freeze({ ...p.evidencia }) });
    },
    async cambiar(cuerpo, { claveIdempotencia } = {}) {
      if (!cuerpo || typeof cuerpo !== "object" || Array.isArray(cuerpo) || Object.keys(cuerpo).some((clave) => !CAMPOS_CAMBIO.has(clave)) ||
          !["publicar", "retirar"].includes(cuerpo.operacion) || !TIPOS.has(cuerpo.tipo) || !REF.test(cuerpo.ref) || !entero(cuerpo.version) || !entero(cuerpo.revision) ||
          (cuerpo.operacion === "publicar" && cuerpo.revision !== 1) || (cuerpo.operacion === "retirar" && cuerpo.revision < 2) ||
          !nombre(cuerpo.denominacion) || !HUELLA.test(cuerpo.huella_sha256) || !fechaCivil(cuerpo.vigente_desde) ||
          (cuerpo.vigente_hasta !== undefined && cuerpo.vigente_hasta !== "" && !fechaCivil(cuerpo.vigente_hasta)) || !REF.test(cuerpo.acto_ref) || !UUID_V4.test(claveIdempotencia)) throw new TypeError("cambio de catálogo no válido");
      const { estado, datos } = await solicitar(RUTA_CATALOGOS_REGISTRO_B2, { method: "POST", headers: { "content-type": "application/json", "Idempotency-Key": claveIdempotencia }, body: JSON.stringify(cuerpo) }, undefined, true);
      const d = datos?.data; const e = d?.entrada; const acceso = d?.acceso_actual;
      if (!e || !REF.test(e.organismo_ref) || !entradaValida(e, e.organismo_ref, cuerpo.tipo) || e.ref !== cuerpo.ref || e.version !== cuerpo.version || e.revision !== cuerpo.revision ||
          e.denominacion !== cuerpo.denominacion || e.huella_sha256 !== cuerpo.huella_sha256 || e.vigente_desde !== cuerpo.vigente_desde || e.vigente_hasta !== (cuerpo.vigente_hasta || "") || e.estado !== (cuerpo.operacion === "publicar" ? "publicada" : "retirada") ||
          !evidenciaValida(d.recibo, undefined, "registrado_en") || !evidenciaValida(acceso, undefined, "registrado_en") ||
          !["registrado", "replay"].includes(acceso.estado_replay) || (estado === 201 && acceso.estado_replay !== "registrado") || (estado === 200 && acceso.estado_replay !== "replay")) throw new ErrorCatalogosRegistroB2("resultado_incierto", estado);
      return Object.freeze({ entrada: Object.freeze({ ...e }), recibo: Object.freeze({ ...d.recibo }), accesoActual: Object.freeze({ ...acceso }) });
    },
  });
}
