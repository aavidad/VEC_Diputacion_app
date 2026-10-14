const BASE = "/api/vec/dietas/comisiones";
const REF = /^dco_[A-Za-z0-9_-]{22,128}$/u;
const CLAVE = /^[A-Za-z0-9:_-]{16,128}$/u;
const SHA = /^[a-f0-9]{64}$/u;
const MAX_RESPUESTA = 256 * 1024;
const MAX_CUERPO = 16 * 1024;
const MAX_DOCUMENTO = 8 * 1024 * 1024;
const MIME_DOCUMENTO = Object.freeze({ pdf: "application/pdf", docx: "application/vnd.openxmlformats-officedocument.wordprocessingml.document" });
const entero = (n) => Number.isSafeInteger(n) && n >= 0;
const objeto = (v) => v !== null && typeof v === "object" && !Array.isArray(v);
const hex = (bytes) => [...bytes].map((byte) => byte.toString(16).padStart(2, "0")).join("");

export class ErrorClienteLiquidacionDietas extends Error {
  constructor(codigo, estado = 0, resultadoIndeterminado = false) {
    super(`liquidacion: ${codigo}`);
    this.codigo = codigo; this.estado = estado; this.resultadoIndeterminado = resultadoIndeterminado;
  }
}
const fallo = (codigo, estado = 0, incierto = false) => new ErrorClienteLiquidacionDietas(codigo, estado, incierto);
function validarOpciones(opciones) {
  if (!objeto(opciones) || Object.keys(opciones).some((k) => k !== "signal")) throw new TypeError("opciones no validas");
  if (opciones.signal !== undefined && (!(opciones.signal instanceof AbortSignal) || opciones.signal.aborted)) throw fallo("operacion_abortada");
  return opciones.signal;
}
function validarLineas(lineas, decisionPendiente) {
  return Array.isArray(lineas) && lineas.length > 0 && lineas.length <= 102 && lineas.every((l, i) => objeto(l)
    && l.indice === i && typeof l.regla_ref === "string" && l.regla_ref.length > 0
    && entero(l.original_centimos) && entero(l.calculado_publicado_centimos ?? 0)
    && Number.isSafeInteger(l.diferencia_centimos ?? 0)
    && (l.diferencia_centimos ?? 0) === (l.calculado_publicado_centimos ?? 0) - l.original_centimos
    && entero(l.reconocido_propuesto_centimos) && l.reconocido_propuesto_centimos <= (l.calculado_publicado_centimos ?? 0)
    && entero(l.rechazado_centimos) && l.reconocido_propuesto_centimos + l.rechazado_centimos === (l.calculado_publicado_centimos ?? 0)
    && typeof l.motivo_codigo === "string" && (l.motivo_codigo === "" || /^[a-z][a-z0-9_]{2,79}$/u.test(l.motivo_codigo))
    && (l.reconocido_propuesto_centimos !== l.original_centimos || l.motivo_codigo === "")
    && (decisionPendiente || l.reconocido_propuesto_centimos === l.original_centimos || l.motivo_codigo !== ""));
}
function validarPropuesta(p, ref, version) {
  if (!objeto(p) || p.esquema !== "vec_dietas_preparacion_liquidacion_publicada_v1"
    || p.comision_ref !== ref || p.comision_version !== version || !SHA.test(p.snapshot_sha256)
    || !objeto(p.catalogo) || p.catalogo.referencia !== p.catalogo_ref || p.catalogo.version !== p.catalogo_version
    || p.catalogo.huella_sha256 !== p.catalogo_sha256 || !SHA.test(p.catalogo_sha256)
    || typeof p.catalogo.publicacion_ref !== "string" || !p.catalogo.publicacion_ref
    || !/^\d{4}-\d{2}-\d{2}$/u.test(p.catalogo.fecha_publicacion)
    || !/^\d{4}-\d{2}-\d{2}$/u.test(p.catalogo.vigente_desde) || !/^\d{4}-\d{2}-\d{2}$/u.test(p.catalogo.vigente_hasta)
    || p.catalogo.vigente_hasta < p.catalogo.vigente_desde || typeof p.catalogo.version_tarifa_ref !== "string" || !p.catalogo.version_tarifa_ref
    || !Array.isArray(p.catalogo.normas) || !p.catalogo.normas.length || p.catalogo.normas.length > 8
    || p.catalogo.normas.some((n) => !objeto(n) || typeof n.referencia !== "string" || typeof n.articulo !== "string"
      || typeof n.url !== "string" || !/^https:\/\/[^\s\x00-\x1f]+$/iu.test(n.url))
    || !Array.isArray(p.catalogo.reglas) || !p.catalogo.reglas.length || p.catalogo.reglas.length > 32
    || p.catalogo.reglas.some((r) => !objeto(r) || typeof r.referencia !== "string" || !r.referencia
      || !["dieta", "kilometraje", "otro_medio", "otro_gasto"].includes(r.tipo) || typeof r.concepto !== "string")
    || typeof p.decision_pendiente !== "boolean" || !validarLineas(p.lineas, p.decision_pendiente) || !objeto(p.totales)
    || ![p.totales.original_centimos, p.totales.calculado_publicado_centimos ?? 0,
      p.totales.reconocido_propuesto_centimos, p.totales.rechazado_centimos].every(entero)
    || !Number.isSafeInteger(p.totales.diferencia_centimos ?? 0)
    || (p.totales.diferencia_centimos ?? 0) !== (p.totales.calculado_publicado_centimos ?? 0) - p.totales.original_centimos
    || p.lineas.reduce((n, l) => n + l.original_centimos, 0) !== p.totales.original_centimos
    || p.lineas.reduce((n, l) => n + (l.calculado_publicado_centimos ?? 0), 0) !== (p.totales.calculado_publicado_centimos ?? 0)
    || p.lineas.reduce((n, l) => n + l.reconocido_propuesto_centimos, 0) !== p.totales.reconocido_propuesto_centimos
    || p.lineas.reduce((n, l) => n + l.rechazado_centimos, 0) !== p.totales.rechazado_centimos) throw fallo("respuesta_incompatible", 200);
  return p;
}
function validarLiquidacion(l, ref, version = 0) {
  if (!objeto(l) || l.esquema !== "vec_dietas_liquidacion_economica_v1" || !SHA.test(l.snapshot_sha256)
    || !objeto(l.constancia) || !objeto(l.economia) || l.economia.comision_ref !== ref
    || !entero(l.constancia.version_entrada) || l.constancia.version_resultante !== l.constancia.version_entrada + 1
    || (version && l.constancia.version_resultante !== version) || l.constancia.version_entrada !== l.economia.comision_version
    || typeof l.constancia.recibo_ref !== "string" || !l.constancia.recibo_ref
    || typeof l.constancia.fecha !== "string" || !Number.isFinite(Date.parse(l.constancia.fecha))) throw fallo("respuesta_incompatible", 200);
  validarPropuesta(l.economia, ref, l.economia.comision_version);
  if (l.economia.decision_pendiente) throw fallo("respuesta_incompatible", 200);
  return l;
}
async function leer(respuesta, signal) {
  const longitud = respuesta.headers?.get?.("content-length");
  if (!/^application\/json(?:;\s*charset=utf-8)?$/iu.test(respuesta.headers?.get?.("content-type") || "")
    || respuesta.headers?.get?.("content-encoding") || !respuesta.body?.getReader
    || (longitud !== null && longitud !== undefined && (!/^\d+$/u.test(longitud) || Number(longitud) > MAX_RESPUESTA))) throw fallo("respuesta_incompatible", respuesta.status);
  const lector = respuesta.body.getReader(); const fragmentos = []; let total = 0;
  const abortar = () => { lector.cancel().catch(() => {}); };
  signal?.addEventListener("abort", abortar, { once: true });
  try {
    for (;;) {
      if (signal?.aborted) throw fallo("operacion_abortada", respuesta.status);
      const parte = await lector.read(); if (parte.done) break;
      if (!(parte.value instanceof Uint8Array) || (total += parte.value.byteLength) > MAX_RESPUESTA || fragmentos.length >= 256) throw fallo("respuesta_excesiva", respuesta.status);
      fragmentos.push(parte.value);
    }
    if (longitud !== null && longitud !== undefined && total !== Number(longitud)) throw fallo("respuesta_incompatible", respuesta.status);
    const bytes = new Uint8Array(total); let offset = 0;
    for (const parte of fragmentos) { bytes.set(parte, offset); offset += parte.byteLength; }
    return JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(bytes));
  } catch (error) { if (error instanceof ErrorClienteLiquidacionDietas) throw error; throw fallo("respuesta_incompatible", respuesta.status); }
  finally { signal?.removeEventListener("abort", abortar); try { lector.releaseLock(); } catch {} }
}
async function leerDocumento(respuesta, signal) {
  const longitud = respuesta.headers?.get?.("content-length");
  if (!longitud || !/^\d+$/u.test(longitud) || Number(longitud) < 1 || Number(longitud) > MAX_DOCUMENTO
    || !respuesta.body?.getReader) throw fallo("documento_incompatible", respuesta.status);
  const lector = respuesta.body.getReader(); const partes = []; let total = 0;
  const abortar = () => { Promise.resolve(lector.cancel()).catch(() => {}); };
  signal?.addEventListener("abort", abortar, { once: true });
  try {
    for (;;) {
      if (signal?.aborted) throw fallo("operacion_abortada", respuesta.status);
      const parte = await lector.read(); if (parte.done) break;
      if (!(parte.value instanceof Uint8Array) || (total += parte.value.byteLength) > MAX_DOCUMENTO || partes.length >= 1024) throw fallo("respuesta_excesiva", respuesta.status);
      partes.push(parte.value);
    }
    if (signal?.aborted) throw fallo("operacion_abortada", respuesta.status);
    if (total !== Number(longitud)) throw fallo("documento_incompatible", respuesta.status);
    const bytes = new Uint8Array(total); let offset = 0;
    for (const parte of partes) { bytes.set(parte, offset); offset += parte.byteLength; }
    return bytes;
  } catch (error) { if (error instanceof ErrorClienteLiquidacionDietas) throw error; throw fallo("documento_incompatible", respuesta.status); }
  finally { signal?.removeEventListener("abort", abortar); try { lector.releaseLock(); } catch {} }
}
const CODIGOS = { 400: ["peticion_invalida"], 403: ["acceso_denegado", "competencia_sin_fuente"], 404: ["no_encontrada"],
  409: ["conflicto_estado", "conflicto_version", "conflicto_idempotencia", "documento_no_disponible"],
  503: ["resultado_incierto", "no_disponible", "relacion_no_disponible"] };
async function ejecutar(fetchImpl, ruta, metodo, cuerpo, signal, escritura, validar) {
  let respuesta;
  try { respuesta = await fetchImpl(ruta, { method: metodo, headers: { Accept: "application/json", ...(cuerpo ? { "Content-Type": "application/json; charset=utf-8" } : {}) },
    ...(cuerpo ? { body: cuerpo } : {}), credentials: "omit", mode: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer", signal }); }
  catch { throw fallo(signal?.aborted ? "operacion_abortada" : "red_no_disponible", 0, escritura && !signal?.aborted); }
  if (!respuesta || respuesta.redirected) throw fallo("respuesta_rechazada", respuesta?.status || 0, escritura);
  let datos;
  try { datos = await leer(respuesta, signal); }
  catch (error) { if (escritura && !signal?.aborted && (respuesta.status >= 500 || respuesta.ok)) throw fallo(error.codigo || "respuesta_incompatible", respuesta.status, true); throw error; }
  if (!respuesta.ok || respuesta.status !== 200) {
    const recibido = typeof datos?.error === "string" ? datos.error.replace(/^dietas\.error\./u, "") : "";
    throw fallo(CODIGOS[respuesta.status]?.includes(recibido) ? recibido : "respuesta_rechazada", respuesta.status, escritura && respuesta.status >= 500);
  }
  try { return validar(datos); } catch { throw fallo("respuesta_incompatible", respuesta.status, escritura); }
}
function validarRevision(r) {
  return objeto(r) && Object.keys(r).every((k) => ["indice", "regla_ref", "reconocido_propuesto_centimos", "motivo_codigo"].includes(k))
    && entero(r.indice) && r.indice < 102 && typeof r.regla_ref === "string" && r.regla_ref.length > 0
    && entero(r.reconocido_propuesto_centimos) && typeof r.motivo_codigo === "string"
    && (r.motivo_codigo === "" || /^[a-z][a-z0-9_]{2,79}$/u.test(r.motivo_codigo));
}
export function crearClienteLiquidacionDietasHTTP({ fetchImpl = globalThis.fetch, cryptoImpl = globalThis.crypto } = {}) {
  if (typeof fetchImpl !== "function") throw new TypeError("fetch no disponible");
  return Object.freeze({
    propuesta: (ref, version, opciones = {}) => {
      if (!REF.test(ref) || !Number.isSafeInteger(version) || version < 1) throw new TypeError("consulta no valida");
      return ejecutar(fetchImpl, `${BASE}/circuito/${encodeURIComponent(ref)}/liquidaciones/propuesta?version_esperada=${version}`, "GET", null, validarOpciones(opciones), false,
        (datos) => { if (!objeto(datos) || Object.keys(datos).length !== 1) throw Error(); return validarPropuesta(datos.propuesta, ref, version); });
    },
    registrar: (ref, entrada, opciones = {}) => {
      if (!REF.test(ref) || !objeto(entrada) || Object.keys(entrada).some((k) => !["version_esperada", "clave_idempotencia", "revisiones"].includes(k))
        || !Number.isSafeInteger(entrada.version_esperada) || entrada.version_esperada < 1 || !CLAVE.test(entrada.clave_idempotencia)
        || !Array.isArray(entrada.revisiones) || !entrada.revisiones.length || entrada.revisiones.length > 102
        || entrada.revisiones.some((r, i) => !validarRevision(r) || r.indice !== i)) throw new TypeError("registro no valido");
      const cuerpo = JSON.stringify(entrada);
      if (new TextEncoder().encode(cuerpo).byteLength > MAX_CUERPO) throw new TypeError("registro excesivo");
      return ejecutar(fetchImpl, `${BASE}/circuito/${encodeURIComponent(ref)}/liquidaciones`, "POST", cuerpo, validarOpciones(opciones), true,
        (datos) => {
          if (!objeto(datos) || Object.keys(datos).length !== 1) throw Error();
          const l = validarLiquidacion(datos.liquidacion, ref);
          if (l.constancia.version_entrada !== entrada.version_esperada || l.economia.lineas.length !== entrada.revisiones.length
            || entrada.revisiones.some((r, i) => l.economia.lineas[i].indice !== r.indice || l.economia.lineas[i].regla_ref !== r.regla_ref
              || l.economia.lineas[i].reconocido_propuesto_centimos !== r.reconocido_propuesto_centimos
              || l.economia.lineas[i].motivo_codigo !== r.motivo_codigo)) throw Error();
          return l;
        });
    },
    leer: (ref, version = "actual", opciones = {}) => {
      if (!REF.test(ref) || (version !== "actual" && (!Number.isSafeInteger(version) || version < 2))) throw new TypeError("lectura no valida");
      return ejecutar(fetchImpl, `${BASE}/${encodeURIComponent(ref)}/liquidaciones/${version}`, "GET", null, validarOpciones(opciones), false,
        (datos) => { if (!objeto(datos) || Object.keys(datos).length !== 1) throw Error(); return validarLiquidacion(datos.liquidacion, ref, version === "actual" ? 0 : version); });
    },
    descargar: async (ref, version, numeroDocumento, formato, idioma, opciones = {}) => {
      if (!REF.test(ref) || !Number.isSafeInteger(version) || version < 2 || !/^VEC-D-\d{4}-\d{6,18}$/u.test(numeroDocumento)
        || !Object.hasOwn(MIME_DOCUMENTO, formato) || typeof idioma !== "string"
        || !/^[a-z]{2,3}(?:-[a-z0-9]{2,8})*$/u.test(idioma)) throw new TypeError("descarga no valida");
      const signal = validarOpciones(opciones); const mime = MIME_DOCUMENTO[formato];
      const ruta = `${BASE}/${encodeURIComponent(ref)}/liquidaciones/${version}/documento?formato=${formato}`;
      let respuesta;
      try { respuesta = await fetchImpl(ruta, { method: "GET", headers: { Accept: mime, "Accept-Language": idioma },
        credentials: "omit", mode: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer", signal }); }
      catch { throw fallo(signal?.aborted ? "operacion_abortada" : "red_no_disponible"); }
      if (!respuesta || respuesta.redirected) throw fallo("respuesta_rechazada", respuesta?.status || 0);
      if (!respuesta.ok || respuesta.status !== 200) {
        let datos; try { datos = await leer(respuesta, signal); } catch { throw fallo("respuesta_rechazada", respuesta.status); }
        const recibido = typeof datos?.error === "string" ? datos.error.replace(/^dietas\.error\./u, "") : "";
        throw fallo(CODIGOS[respuesta.status]?.includes(recibido) ? recibido : "respuesta_rechazada", respuesta.status);
      }
      const nombre = `${numeroDocumento}.${formato}`;
      if (respuesta.headers?.get?.("content-type") !== mime
        || respuesta.headers?.get?.("content-disposition") !== `attachment; filename="${nombre}"`
        || !/(?:^|,)\s*no-store\s*(?:,|$)/iu.test(respuesta.headers?.get?.("cache-control") || "")
        || respuesta.headers?.get?.("x-content-type-options") !== "nosniff"
        || !SHA.test(respuesta.headers?.get?.("x-content-sha256") || "")) throw fallo("documento_incompatible", respuesta.status);
      const bytes = await leerDocumento(respuesta, signal);
      if ((formato === "pdf" && new TextDecoder().decode(bytes.subarray(0, 5)) !== "%PDF-")
        || (formato === "docx" && !(bytes[0] === 0x50 && bytes[1] === 0x4b && bytes[2] === 0x03 && bytes[3] === 0x04))) throw fallo("documento_incompatible", respuesta.status);
      if (typeof cryptoImpl?.subtle?.digest !== "function") throw fallo("huella_no_verificable", respuesta.status);
      const huella = hex(new Uint8Array(await cryptoImpl.subtle.digest("SHA-256", bytes)));
      if (signal?.aborted) throw fallo("operacion_abortada", respuesta.status);
      if (huella !== respuesta.headers.get("x-content-sha256")) throw fallo("huella_no_coincide", respuesta.status);
      return Object.freeze({ nombre, mime, bytes, huella_sha256: huella });
    },
  });
}
