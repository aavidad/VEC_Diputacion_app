const BASE = "/api/vec/bolsa/rrhh/inscripciones";
const ESTADOS = new Set(["pendiente", "admitida_a_convocatoria", "incorporada", "rechazada"]);
const DECISIONES = new Set(["admitir", "rechazar"]);
const REF = /^[^/\u0000-\u001f\u007f-\u009f]{1,512}$/u;
const CLAVE = /^[a-zA-Z0-9._:-]{8,128}$/u;
const IDIOMA = /^[a-z]{2,3}(?:-[a-z0-9]{2,8})*$/u;
const ESQUEMAS = Object.freeze({ lista: "vec.bolsa.inscripciones.rrhh.v1",
  detalle: "vec.bolsa.inscripcion.rrhh.v1", motivos: "vec.bolsa.inscripcion.motivos.v1",
  recibo: "vec.bolsa.inscripcion.decision.recibo.v1" });
const MAXIMO_RESPUESTA = 256 * 1024;

function referencia(valor) {
  if (typeof valor !== "string" || !REF.test(valor) || valor.trim() !== valor) throw new TypeError("referencia incompatible");
  return valor;
}

function solicitudValida(item) {
  return item && typeof item === "object" && referencia(item.solicitud_ref)
    && referencia(item.convocatoria_ref) && (item.bolsa_ref == null || referencia(item.bolsa_ref))
    && typeof item.categoria === "string" && item.categoria.trim().length > 0 && item.categoria.length <= 240
    && (item.persona_resumen == null
      || typeof item.persona_resumen === "string" && item.persona_resumen.length <= 240)
    && ESTADOS.has(item.estado)
    && Number.isSafeInteger(item.version) && item.version > 0
    && typeof item.registrada_en === "string" && Number.isFinite(Date.parse(item.registrada_en))
    && (item.motivo_codigo == null || referencia(item.motivo_codigo));
}
function detalleValido(item) {
  return solicitudValida(item) && referencia(item.bases_ref)
    && Number.isSafeInteger(item.catalogo_version) && item.catalogo_version > 0
    && typeof item.plazo_inicio === "string" && Number.isFinite(Date.parse(item.plazo_inicio))
    && typeof item.plazo_fin === "string" && Number.isFinite(Date.parse(item.plazo_fin))
    && referencia(item.declaracion_ref) && Array.isArray(item.requisitos) && item.requisitos.length <= 100
    && item.requisitos.every((r) => r && referencia(r.codigo) && typeof r.descripcion === "string"
      && r.descripcion.length > 0 && r.descripcion.length <= 500 && typeof r.obligatorio === "boolean"
      && ["cumple", "no_cumple", "pendiente"].includes(r.estado)
      && (r.fuente_ref == null || referencia(r.fuente_ref))
      && (r.evidencia_ref == null || referencia(r.evidencia_ref)));
}

async function pedir(fetchImpl, ruta, { method = "GET", body, signal } = {}) {
  const respuesta = await fetchImpl(ruta, { method, credentials: "same-origin", mode: "same-origin",
    cache: "no-store", redirect: "error", referrerPolicy: "no-referrer", signal,
    headers: { Accept: "application/json", ...(body ? { "Content-Type": "application/json" } : {}) },
    ...(body ? { body: JSON.stringify(body) } : {}) });
  if (!respuesta || respuesta.redirected || !Number.isInteger(respuesta.status)) throw new TypeError("respuesta incompatible");
  if (!respuesta.ok) throw Object.assign(new Error("operación no completada"), { estado: respuesta.status });
  if (!/^application\/json(?:;|$)/iu.test(respuesta.headers?.get?.("content-type") || "")) throw new TypeError("respuesta incompatible");
  const longitud = respuesta.headers?.get?.("content-length");
  if (longitud && (!/^\d+$/u.test(longitud) || Number(longitud) > MAXIMO_RESPUESTA)) throw new TypeError("respuesta excesiva");
  const lector = respuesta.body?.getReader?.();
  let texto;
  if (lector) {
    let total = 0;
    const partes = [];
    try {
      for (;;) {
        const { done, value } = await lector.read();
        if (done) break;
        if (!(value instanceof Uint8Array) || (total += value.byteLength) > MAXIMO_RESPUESTA) {
          await lector.cancel(); throw new TypeError("respuesta excesiva");
        }
        partes.push(value);
      }
    } finally { lector.releaseLock(); }
    const bytes = new Uint8Array(total);
    let posicion = 0;
    for (const parte of partes) { bytes.set(parte, posicion); posicion += parte.byteLength; }
    texto = new TextDecoder("utf-8", { fatal: true }).decode(bytes);
  } else {
    texto = await respuesta.text();
    if (new TextEncoder().encode(texto).byteLength > MAXIMO_RESPUESTA) throw new TypeError("respuesta excesiva");
  }
  return { estado: respuesta.status, data: JSON.parse(texto)?.data };
}

export function crearClienteInscripcionesRRHH({ fetchImpl = globalThis.fetch } = {}) {
  if (typeof fetchImpl !== "function") throw new TypeError("transporte no disponible");
  return Object.freeze({
    async listar({ estado = "pendiente", convocatoria = "", cursor = "", limite = 50, idioma = "", signal } = {}) {
      if (!ESTADOS.has(estado) || (convocatoria && !referencia(convocatoria)) || (cursor && !referencia(cursor))
        || !Number.isSafeInteger(limite) || limite < 20 || limite > 100
        || (idioma && !IDIOMA.test(idioma))) throw new TypeError("filtro incompatible");
      const parametros = new URLSearchParams({ estado, limite: String(limite) });
      if (convocatoria) parametros.set("convocatoria_ref", convocatoria);
      if (cursor) parametros.set("cursor", cursor);
      if (idioma) parametros.set("idioma", idioma);
      const { estado: http, data } = await pedir(fetchImpl, `${BASE}?${parametros}`, { signal });
      if (http !== 200 || data?.esquema !== ESQUEMAS.lista || !Array.isArray(data.solicitudes)
        || data.solicitudes.length > limite || !data.solicitudes.every(solicitudValida)
        || !Number.isSafeInteger(data.total) || data.total < data.solicitudes.length
        || (data.cursor_siguiente !== null && !referencia(data.cursor_siguiente))) throw new TypeError("lista incompatible");
      return data;
    },
    async detalle(solicitudRef, { signal, idioma = "" } = {}) {
      if (idioma && !IDIOMA.test(idioma)) throw new TypeError("idioma incompatible");
      const ruta = `${BASE}/${encodeURIComponent(referencia(solicitudRef))}${idioma ? `?idioma=${encodeURIComponent(idioma)}` : ""}`;
      const { estado, data } = await pedir(fetchImpl, ruta, { signal });
      if (estado !== 200 || data?.esquema !== ESQUEMAS.detalle || !detalleValido(data.solicitud)
        || data.solicitud.solicitud_ref !== solicitudRef) throw new TypeError("detalle incompatible");
      return data.solicitud;
    },
    async motivos(decision, { signal, idioma = "" } = {}) {
      if (!DECISIONES.has(decision) || (idioma && !IDIOMA.test(idioma))) throw new TypeError("decisión incompatible");
      const parametros = new URLSearchParams({ decision });
      if (idioma) parametros.set("idioma", idioma);
      const { estado, data } = await pedir(fetchImpl, `${BASE}/motivos?${parametros}`, { signal });
      if (estado !== 200 || data?.esquema !== ESQUEMAS.motivos
        || !Number.isSafeInteger(data.catalogo_version) || data.catalogo_version < 1
        || !Array.isArray(data.motivos) || data.motivos.length > 100
        || !data.motivos.every((m) => m && referencia(m.codigo) && typeof m.etiqueta === "string"
          && m.etiqueta.length > 0 && m.etiqueta.length <= 240 && typeof m.obligatorio === "boolean")) {
        throw new TypeError("motivos incompatibles");
      }
      return data;
    },
    async decidir({ solicitudRef, decision, motivoCodigo = "", versionEsperada, claveIdempotencia, signal }) {
      if (!DECISIONES.has(decision) || (motivoCodigo && !referencia(motivoCodigo))
        || !Number.isSafeInteger(versionEsperada) || versionEsperada < 1
        || typeof claveIdempotencia !== "string" || !CLAVE.test(claveIdempotencia)) throw new TypeError("decisión incompatible");
      const body = { decision, version_esperada: versionEsperada, clave_idempotencia: claveIdempotencia };
      if (motivoCodigo) body.motivo_codigo = motivoCodigo;
      const { estado, data } = await pedir(fetchImpl, `${BASE}/${encodeURIComponent(referencia(solicitudRef))}/decisiones`,
        { method: "POST", body, signal });
      if (![200, 201].includes(estado) || data?.esquema !== ESQUEMAS.recibo
        || data.solicitud_ref !== solicitudRef || !referencia(data.recibo_ref)
        || !(decision === "admitir" ? ["admitida_a_convocatoria", "incorporada"].includes(data.estado) : data.estado === "rechazada")
        || !Number.isSafeInteger(data.version) || data.version <= versionEsperada
        || typeof data.decidida_en !== "string" || !Number.isFinite(Date.parse(data.decidida_en))
        || typeof data.repetida !== "boolean" || (data.participacion_ref != null && !referencia(data.participacion_ref))
        || (data.estado !== "incorporada" && data.participacion_ref != null)
        || (data.estado === "incorporada" && !data.participacion_ref)) {
        throw new TypeError("recibo incompatible");
      }
      return data;
    },
    async incorporar({ solicitudRef, evidenciaRef, versionEsperada, claveIdempotencia, signal }) {
      referencia(evidenciaRef);
      if (!Number.isSafeInteger(versionEsperada) || versionEsperada < 1
        || typeof claveIdempotencia !== "string" || !CLAVE.test(claveIdempotencia)) throw new TypeError("incorporación incompatible");
      const { estado, data } = await pedir(fetchImpl, `${BASE}/${encodeURIComponent(referencia(solicitudRef))}/incorporaciones`,
        { method: "POST", body: { evidencia_ref: evidenciaRef, version_esperada: versionEsperada,
          clave_idempotencia: claveIdempotencia }, signal });
      if (![200, 201].includes(estado) || data?.esquema !== "vec.bolsa.inscripcion.incorporacion.recibo.v1"
        || data.solicitud_ref !== solicitudRef || !referencia(data.recibo_ref)
        || data.estado !== "incorporada" || !Number.isSafeInteger(data.version) || data.version <= versionEsperada
        || !referencia(data.participacion_ref) || typeof data.incorporada_en !== "string"
        || !Number.isFinite(Date.parse(data.incorporada_en)) || typeof data.repetida !== "boolean") {
        throw new TypeError("recibo de incorporación incompatible");
      }
      return data;
    },
  });
}
