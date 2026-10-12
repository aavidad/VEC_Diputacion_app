const BASE = "/api/vec/bolsa/bolsas";
const ESQUEMA = "vec.bolsa.rrhh.documentos.v1";
const ESQUEMA_RECIBO = "vec.bolsa.rrhh.recibo_constitucion.v1";
const MAX_JSON = 1024 * 1024;

function referencia(valor) {
  return typeof valor === "string" && valor.length > 0 && valor.length <= 512
    && /^[A-Za-z0-9][A-Za-z0-9:._~-]*$/u.test(valor) && valor !== "." && valor !== "..";
}

function segmento(valor) {
  if (!referencia(valor)) throw new TypeError("referencia no válida");
  return encodeURIComponent(valor).replace(/%3A/giu, ":");
}

function rutaLista(bolsaRef) { return `${BASE}/${segmento(bolsaRef)}/documentos`; }
function rutaRecibo(bolsaRef, documentoRef) { return `${rutaLista(bolsaRef)}/${segmento(documentoRef)}/recibo`; }
const fechaValida = (v) => typeof v === "string" && v.length <= 64 && Number.isFinite(Date.parse(v));
const versionValida = (v) => Number.isSafeInteger(v) && v > 0;

function validarDocumento(item, bolsaRef) {
  if (!item || !referencia(item.documento_ref) || item.tipo !== "constitucion"
      || item.titulo_clave !== "bolsa_documento_constitucion" || !fechaValida(item.fecha)
      || !referencia(item.acta_ref) || item.documento_ref !== item.acta_ref
      || !versionValida(item.version_bolsa) || !referencia(item.instantanea_ref)
      || !versionValida(item.version_instantanea)
      || !Number.isSafeInteger(item.total_participaciones) || item.total_participaciones < 0
      || item.estado_firma !== "sin_firma_acreditada" || item.formato !== "application/json"
      || item.accion?.tipo !== "descargar_recibo" || item.accion.href !== rutaRecibo(bolsaRef, item.documento_ref)
      || item.ficha_candidatos_href !== `${BASE}/${segmento(bolsaRef)}/candidatos`) {
    throw new TypeError("documento no válido");
  }
  return Object.freeze({ ...item });
}

export function validarListaDocumentosBolsa(datos, bolsaRef) {
  if (datos?.esquema !== ESQUEMA || datos.bolsa_ref !== bolsaRef || !Array.isArray(datos.documentos)
      || datos.documentos.length > 100 || !datos.paginacion
      || !Number.isSafeInteger(datos.paginacion.limite) || datos.paginacion.limite < 1 || datos.paginacion.limite > 100
      || !Number.isSafeInteger(datos.paginacion.total) || datos.paginacion.total < datos.documentos.length
      || !(datos.paginacion.cursor_siguiente === null || typeof datos.paginacion.cursor_siguiente === "string"
        && datos.paginacion.cursor_siguiente.length > 0 && datos.paginacion.cursor_siguiente.length <= 512)) {
    throw new TypeError("lista de documentos no válida");
  }
  const documentos = datos.documentos.map((item) => validarDocumento(item, bolsaRef));
  if (new Set(documentos.map((item) => item.documento_ref)).size !== documentos.length) throw new TypeError("documento repetido");
  return Object.freeze({ documentos, paginacion: Object.freeze({ ...datos.paginacion }) });
}

function validarRecibo(datos, bolsaRef, documentoRef) {
  if (datos?.esquema !== ESQUEMA_RECIBO || datos.bolsa_ref !== bolsaRef || datos.documento_ref !== documentoRef
      || datos.acta_ref !== documentoRef || !versionValida(datos.version_bolsa)
      || !referencia(datos.instantanea_ref) || !versionValida(datos.version_instantanea)
      || !fechaValida(datos.confirmada_en) || !Number.isSafeInteger(datos.total_participaciones)
      || datos.total_participaciones < 0 || datos.estado_firma !== "sin_firma_acreditada") {
    throw new TypeError("recibo no válido");
  }
  return Object.freeze({ ...datos });
}

async function leerJSON(respuesta) {
  if (!/^application\/json(?:\s*;|$)/iu.test(respuesta.headers?.get("Content-Type") ?? "")) throw new TypeError("tipo de respuesta no válido");
  const longitud = Number(respuesta.headers?.get("Content-Length"));
  if (Number.isFinite(longitud) && longitud > MAX_JSON) throw new TypeError("respuesta demasiado grande");
  let bytes;
  if (respuesta.body?.getReader) {
    const lector = respuesta.body.getReader();
    const partes = [];
    let total = 0;
    try {
      while (true) {
        const parte = await lector.read();
        if (parte.done) break;
        total += parte.value.byteLength;
        if (total > MAX_JSON) throw new TypeError("respuesta demasiado grande");
        partes.push(parte.value);
      }
    } finally { await lector.cancel().catch(() => {}); }
    bytes = new Uint8Array(total);
    let posicion = 0;
    for (const parte of partes) { bytes.set(parte, posicion); posicion += parte.byteLength; }
  } else {
    bytes = new Uint8Array(await respuesta.arrayBuffer());
  }
  if (!bytes.length || bytes.length > MAX_JSON) throw new TypeError("respuesta no válida");
  const texto = new TextDecoder("utf-8", { fatal: true }).decode(bytes);
  return { bytes, datos: JSON.parse(texto)?.data };
}

async function consultar(url, { signal, fetchImpl }) {
  const respuesta = await fetchImpl(url, { method: "GET", credentials: "same-origin", mode: "same-origin",
    cache: "no-store", redirect: "error", referrerPolicy: "no-referrer", signal,
    headers: { Accept: "application/json" } });
  if (!respuesta.ok) throw Object.assign(new Error("consulta fallida"), { status: respuesta.status });
  return leerJSON(respuesta);
}

export function crearClienteDocumentosBolsa({ fetchImpl = globalThis.fetch } = {}) {
  if (typeof fetchImpl !== "function") throw new TypeError("transporte no válido");
  return Object.freeze({
    async listar({ bolsaRef, tipo = "", texto = "", cursor = "", signal } = {}) {
      const ruta = rutaLista(bolsaRef);
      if (tipo !== "" && tipo !== "constitucion" || typeof texto !== "string" || texto.length > 120
          || typeof cursor !== "string" || cursor.length > 512) throw new TypeError("filtro no válido");
      const filtros = new URLSearchParams({ limite: "50" });
      if (tipo) filtros.set("tipo", tipo);
      if (texto.trim()) filtros.set("texto", texto.trim());
      if (cursor) filtros.set("cursor", cursor);
      const { datos } = await consultar(`${ruta}?${filtros}`, { signal, fetchImpl });
      return validarListaDocumentosBolsa(datos, bolsaRef);
    },
    async recibir({ bolsaRef, documentoRef, signal } = {}) {
      const ruta = rutaRecibo(bolsaRef, documentoRef);
      const { bytes, datos } = await consultar(ruta, { signal, fetchImpl });
      return Object.freeze({ datos: validarRecibo(datos, bolsaRef, documentoRef), bytes });
    },
  });
}
