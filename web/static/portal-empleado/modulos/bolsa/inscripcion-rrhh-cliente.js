import { consultarJSON, ErrorConsultaJSON } from "../../../comun/http.js?v=20261010-http-carga-v1";

const BASE = "/api/vec/bolsa/rrhh/inscripciones";
const SEGMENTO = /^[A-Za-z0-9][A-Za-z0-9:._-]{0,511}$/u;
const ESTADOS = new Set(["pendiente", "admitida_a_convocatoria", "incorporada", "rechazada"]);
const DECISIONES = new Set(["admitir", "rechazar"]);
const REF = /^[^/\u0000-\u001f\u007f-\u009f]{1,512}$/u;
const CLAVE = /^[a-zA-Z0-9._:-]{8,128}$/u;
const IDIOMA = /^[a-z]{2,3}(?:-[a-z0-9]{2,8})*$/u;
const ESQUEMAS = Object.freeze({ lista: "vec.bolsa.inscripciones.rrhh.v1",
  convocatorias: "vec.bolsa.inscripciones.rrhh.convocatorias.v1",
  detalle: "vec.bolsa.inscripcion.rrhh.v1", motivos: "vec.bolsa.inscripcion.motivos.v1",
  recibo: "vec.bolsa.inscripcion.decision.recibo.v1" });
const MAXIMO_RESPUESTA = 256 * 1024;
const bytesUTF8 = (valor) => new TextEncoder().encode(valor).byteLength;

function referencia(valor) {
  if (typeof valor !== "string" || !REF.test(valor) || valor.trim() !== valor) throw new TypeError("referencia incompatible");
  return valor;
}

function solicitudValida(item) {
  return item && typeof item === "object" && referencia(item.solicitud_ref) && referencia(item.recibo_ref)
    && referencia(item.convocatoria_ref) && referencia(item.categoria_ref) && referencia(item.declaracion_ref)
    && (item.bolsa_ref == null || referencia(item.bolsa_ref))
    && typeof item.categoria === "string" && item.categoria.trim().length > 0
    && bytesUTF8(item.categoria) <= 2048
    && typeof item.convocatoria_titulo === "string" && item.convocatoria_titulo.trim().length > 0
    && Array.from(item.convocatoria_titulo).length <= 180
    && (item.persona_resumen == null
      || typeof item.persona_resumen === "string" && item.persona_resumen.length <= 240)
    && ESTADOS.has(item.estado)
    && Number.isSafeInteger(item.version) && item.version > 0
    && typeof item.registrada_en === "string" && Number.isFinite(Date.parse(item.registrada_en))
    && (item.motivo_codigo == null || referencia(item.motivo_codigo))
    && (item.motivo_etiqueta == null || typeof item.motivo_etiqueta === "string" && item.motivo_etiqueta.length <= 240);
}
function detalleValido(item) {
  return solicitudValida(item) && referencia(item.bases_ref)
    && Number.isSafeInteger(item.catalogo_version) && item.catalogo_version > 0
    && typeof item.plazo_inicio === "string" && Number.isFinite(Date.parse(item.plazo_inicio))
    && typeof item.plazo_fin === "string" && Number.isFinite(Date.parse(item.plazo_fin))
    && Array.isArray(item.requisitos) && item.requisitos.length <= 100
    && item.requisitos.every((r) => r && referencia(r.codigo) && typeof r.descripcion === "string"
      && r.descripcion.length > 0 && r.descripcion.length <= 500 && typeof r.obligatorio === "boolean"
      && ["cumple", "no_cumple", "pendiente"].includes(r.estado)
      && (r.motivo_etiqueta == null || typeof r.motivo_etiqueta === "string" && r.motivo_etiqueta.length <= 500)
      && (r.hito_etiqueta == null || typeof r.hito_etiqueta === "string" && r.hito_etiqueta.length <= 240)
      && (r.fuente_ref == null || referencia(r.fuente_ref))
      && (r.evidencia_ref == null || referencia(r.evidencia_ref)));
}

// Ruta interna sin codificar: la comprobación común no admite «%» en el camino.
function segmento(valor) {
  referencia(valor);
  if (!SEGMENTO.test(valor)) throw new TypeError("referencia incompatible");
  return valor;
}

// Transporte común (mTLS del mismo origen, límites, plazo y reintento de GET).
// Los errores conservan el estado HTTP y el código de negocio de 409/422.
async function pedir(fetchImpl, ruta, { method = "GET", body, signal } = {}) {
  try {
    return { data: (await consultarJSON(ruta, { metodo: method, cuerpo: body, signal, fetchImpl, limiteBytes: MAXIMO_RESPUESTA }))?.data };
  } catch (fallo) {
    if (!(fallo instanceof ErrorConsultaJSON)) throw fallo;
    throw Object.assign(new Error("operación no completada"), { estado: fallo.estado,
      ...(fallo.codigoServidor ? { codigo: fallo.codigoServidor } : {}) });
  }
}

export function crearClienteInscripcionesRRHH({ fetchImpl = globalThis.fetch } = {}) {
  if (typeof fetchImpl !== "function") throw new TypeError("transporte no disponible");
  return Object.freeze({
    async convocatorias({ cursor = "", limite = 20, idioma = "", signal } = {}) {
      if ((cursor && !referencia(cursor)) || !Number.isSafeInteger(limite) || limite < 1 || limite > 100
        || (idioma && !IDIOMA.test(idioma))) throw new TypeError("filtro de convocatorias incompatible");
      const parametros = new URLSearchParams({ limite: String(limite) });
      if (cursor) parametros.set("cursor", cursor);
      if (idioma) parametros.set("idioma", idioma);
      const { data } = await pedir(fetchImpl, `${BASE}/convocatorias?${parametros}`, { signal });
      if (data?.esquema !== ESQUEMAS.convocatorias || !Array.isArray(data.convocatorias)
        || data.convocatorias.length > limite || !Number.isSafeInteger(data.total)
        || data.total < data.convocatorias.length
        || (data.cursor_siguiente !== null && !referencia(data.cursor_siguiente))
        || !data.convocatorias.every((item) => item && referencia(item.convocatoria_ref)
          && typeof item.titulo === "string" && item.titulo.trim().length > 0 && item.titulo.length <= 180
          && typeof item.categorias_resumen === "string" && item.categorias_resumen.trim().length > 0
          && bytesUTF8(item.categorias_resumen) <= 2048
          && typeof item.plazo_fin === "string" && Number.isFinite(Date.parse(item.plazo_fin))
          && Number.isSafeInteger(item.pendientes) && item.pendientes >= 0
          && ["publicada", "sustituida", "retirada"].includes(item.estado_publicacion))) {
        throw new TypeError("convocatorias incompatibles");
      }
      return data;
    },
    async listar({ estado = "pendiente", convocatoria = "", cursor = "", limite = 50, idioma = "", signal } = {}) {
      if (!ESTADOS.has(estado) || !referencia(convocatoria) || (cursor && !referencia(cursor))
        || !Number.isSafeInteger(limite) || limite < 20 || limite > 100
        || (idioma && !IDIOMA.test(idioma))) throw new TypeError("filtro incompatible");
      const parametros = new URLSearchParams({ estado, limite: String(limite) });
      if (convocatoria) parametros.set("convocatoria_ref", convocatoria);
      if (cursor) parametros.set("cursor", cursor);
      if (idioma) parametros.set("idioma", idioma);
      const { data } = await pedir(fetchImpl, `${BASE}?${parametros}`, { signal });
      if (data?.esquema !== ESQUEMAS.lista
        || typeof data.convocatoria_titulo !== "string" || !data.convocatoria_titulo.trim()
        || data.convocatoria_titulo.length > 180 || !Array.isArray(data.solicitudes)
        || data.solicitudes.length > limite || !data.solicitudes.every(solicitudValida)
        || !Number.isSafeInteger(data.total) || data.total < data.solicitudes.length
        || (data.cursor_siguiente !== null && !referencia(data.cursor_siguiente))) throw new TypeError("lista incompatible");
      return data;
    },
    async detalle(solicitudRef, { signal, idioma = "" } = {}) {
      if (idioma && !IDIOMA.test(idioma)) throw new TypeError("idioma incompatible");
      const ruta = `${BASE}/${segmento(solicitudRef)}${idioma ? `?idioma=${encodeURIComponent(idioma)}` : ""}`;
      const { data } = await pedir(fetchImpl, ruta, { signal });
      if (data?.esquema !== ESQUEMAS.detalle || !detalleValido(data.solicitud)
        || data.solicitud.solicitud_ref !== solicitudRef) throw new TypeError("detalle incompatible");
      return data.solicitud;
    },
    async motivos(decision, { signal, idioma = "" } = {}) {
      if (!DECISIONES.has(decision) || (idioma && !IDIOMA.test(idioma))) throw new TypeError("decisión incompatible");
      const parametros = new URLSearchParams({ decision });
      if (idioma) parametros.set("idioma", idioma);
      const { data } = await pedir(fetchImpl, `${BASE}/motivos?${parametros}`, { signal });
      if (data?.esquema !== ESQUEMAS.motivos
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
      const { data } = await pedir(fetchImpl, `${BASE}/${segmento(solicitudRef)}/decisiones`,
        { method: "POST", body, signal });
      if (data?.esquema !== ESQUEMAS.recibo
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
      const { data } = await pedir(fetchImpl, `${BASE}/${segmento(solicitudRef)}/incorporaciones`,
        { method: "POST", body: { evidencia_ref: evidenciaRef, version_esperada: versionEsperada,
          clave_idempotencia: claveIdempotencia }, signal });
      if (data?.esquema !== "vec.bolsa.inscripcion.incorporacion.recibo.v1"
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
