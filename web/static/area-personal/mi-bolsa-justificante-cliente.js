const RUTA = "/api/vec/bolsa/mi-bolsa/justificantes";
export const MAX_PDF_JUSTIFICANTE = 16 * 1024 * 1024;
const REF = /^ref:[0-9a-f]{64}$/u;
const FECHA = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})$/u;

export class ErrorJustificante extends Error {
  constructor(codigo, estado = 503, causa) {
    super(codigo, { cause: causa });
    this.codigo = codigo;
    this.estado = estado;
  }
}

function validar(data) {
  if (!data || typeof data !== "object" || Array.isArray(data) || typeof data.carga_id !== "string" || !REF.test(data.carga_id)
    || !["reservada", "recibido", "analizando", "admitido", "rechazado"].includes(data.estado)
    || (data.recibo_ref !== undefined && (typeof data.recibo_ref !== "string" || !data.recibo_ref))) {
    throw new ErrorJustificante("respuesta_invalida");
  }
  if (data.estado === "admitido") {
    if (typeof data.comprobante_ref !== "string" || !REF.test(data.comprobante_ref)
      || !Number.isSafeInteger(data.documento_version) || data.documento_version < 1
      || typeof data.recibo_ref !== "string" || !data.recibo_ref
      || typeof data.registrado_en !== "string" || !FECHA.test(data.registrado_en)
      || Number.isNaN(Date.parse(data.registrado_en)) || data.codigo_rechazo !== undefined) {
      throw new ErrorJustificante("respuesta_invalida");
    }
  } else if (data.comprobante_ref !== undefined || data.registrado_en !== undefined || data.documento_version !== undefined
    || (data.estado === "rechazado" ? typeof data.codigo_rechazo !== "string" || !data.codigo_rechazo : data.codigo_rechazo !== undefined)) {
    throw new ErrorJustificante("respuesta_invalida");
  }
  return data;
}

async function pedir(fetchImpl, url, opciones) {
  let respuesta;
  try { respuesta = await fetchImpl(url, opciones); }
  catch (causa) {
    if (causa?.name === "AbortError") throw causa;
    throw new ErrorJustificante("servicio_no_disponible", 503, causa);
  }
  if (!respuesta || typeof respuesta.status !== "number" || (opciones.method !== "GET" || respuesta.status !== 404)
    && typeof respuesta.json !== "function") {
    throw new ErrorJustificante("respuesta_invalida");
  }
  if (respuesta.status === 413) throw new ErrorJustificante("archivo_demasiado_grande", 413);
  if (opciones.method === "GET" && respuesta.status === 404) return { respuesta, contenido: null };
  let contenido;
  try { contenido = await respuesta.json(); }
  catch (causa) { throw new ErrorJustificante("respuesta_invalida", 503, causa); }
  return { respuesta, contenido };
}

export function crearClienteJustificante({ fetchImpl = globalThis.fetch } = {}) {
  if (typeof fetchImpl !== "function") throw new TypeError("fetch_invalido");
  const comunes = { headers: { Accept: "application/json" }, mode: "same-origin", credentials: "same-origin",
    cache: "no-store", redirect: "error", referrerPolicy: "no-referrer" };
  return Object.freeze({
    async consultar(bolsa, clave, { signal } = {}) {
      const query = new URLSearchParams({ bolsa, clave_idempotencia: clave });
      const { respuesta, contenido } = await pedir(fetchImpl, `${RUTA}?${query}`, { ...comunes, method: "GET", signal });
      if (respuesta.status === 404) return null;
      if (![200, 202, 422].includes(respuesta.status) || !contenido?.data) {
        throw new ErrorJustificante(contenido?.error?.codigo || "servicio_no_disponible", respuesta.status);
      }
      const estado = validar(contenido.data);
      if ((respuesta.status === 200 && estado.estado !== "admitido")
        || (respuesta.status === 202 && !["reservada", "recibido", "analizando"].includes(estado.estado))
        || (respuesta.status === 422 && estado.estado !== "rechazado")) throw new ErrorJustificante("respuesta_invalida");
      return estado;
    },
    async subir(bolsa, clave, documento, { signal } = {}) {
      if (typeof bolsa !== "string" || !bolsa || typeof clave !== "string" || !clave || !(documento instanceof Blob)
        || !documento.size || !documento.name) throw new TypeError("justificante_invalido");
      if (documento.size > MAX_PDF_JUSTIFICANTE) throw new ErrorJustificante("archivo_demasiado_grande", 413);
      const cuerpo = new FormData();
      cuerpo.append("bolsa", bolsa);
      cuerpo.append("clave_idempotencia", clave);
      cuerpo.append("documento", documento, documento.name);
      const { respuesta, contenido } = await pedir(fetchImpl, RUTA, { ...comunes, method: "POST", body: cuerpo, signal });
      if (![200, 201, 202, 422].includes(respuesta.status) || !contenido?.data) {
        throw new ErrorJustificante(contenido?.error?.codigo || "servicio_no_disponible", respuesta.status);
      }
      const estado = validar(contenido?.data);
      if (([200, 201].includes(respuesta.status) && estado.estado !== "admitido")
        || (respuesta.status === 202 && !["reservada", "recibido", "analizando"].includes(estado.estado))
        || (respuesta.status === 422 && estado.estado !== "rechazado")) throw new ErrorJustificante("respuesta_invalida");
      return estado;
    },
  });
}
