const RUTA = "/api/vec/bolsa/justificantes-renuncia/rrhh";
const MAX_PDF = 16 * 1024 * 1024;

function fallo(codigo, estado = 503) {
  return Object.assign(new Error(codigo), { codigo, estado });
}

function referencia(valor) {
  return typeof valor === "string" && valor.length > 0 && valor.length <= 255 && !/[\s?#%\\]/u.test(valor);
}

async function bytesLimitados(respuesta) {
  if (!respuesta.body?.getReader) {
    const bytes = new Uint8Array(await respuesta.arrayBuffer());
    if (!bytes.length || bytes.length > MAX_PDF) throw fallo("respuesta_invalida");
    return bytes;
  }
  const lector = respuesta.body.getReader();
  const partes = [];
  let longitud = 0;
  try {
    while (true) {
      const parte = await lector.read();
      if (parte.done) break;
      longitud += parte.value.byteLength;
      if (longitud > MAX_PDF) throw fallo("respuesta_invalida");
      partes.push(parte.value);
    }
  } finally { await lector.cancel().catch(() => {}); }
  if (!longitud) throw fallo("respuesta_invalida");
  const bytes = new Uint8Array(longitud);
  let posicion = 0;
  for (const parte of partes) { bytes.set(parte, posicion); posicion += parte.byteLength; }
  return bytes;
}

export function crearClienteJustificanteRenuncia({ fetchImpl = globalThis.fetch } = {}) {
  if (typeof fetchImpl !== "function") throw new TypeError("fetch_invalido");
  return Object.freeze({
    async descargar(bolsa, participacion, { signal } = {}) {
      if (!referencia(bolsa) || !referencia(participacion)) throw fallo("datos_no_validos", 400);
      const query = new URLSearchParams({ bolsa, participacion });
      let respuesta;
      try {
        respuesta = await fetchImpl(`${RUTA}?${query}`, { method: "GET", headers: { Accept: "application/pdf" },
          credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer", signal });
      } catch (error) {
        if (error?.name === "AbortError") throw error;
        throw fallo("servicio_no_disponible");
      }
      if (respuesta?.status === 401 || respuesta?.status === 403) throw fallo("acceso_denegado", respuesta.status);
      if (respuesta?.status === 404) throw fallo("justificante_no_disponible", 404);
      if (respuesta?.status === 413) throw fallo("archivo_demasiado_grande", 413);
      if (respuesta?.status !== 200) throw fallo("servicio_no_disponible", respuesta?.status);
      if (respuesta.headers?.get?.("Content-Type")?.split(";")[0]?.trim().toLowerCase() !== "application/pdf"
        || !/^attachment;\s*filename=(?:"justificante\.pdf"|justificante\.pdf)$/iu.test(respuesta.headers.get("Content-Disposition") || "")) {
        throw fallo("respuesta_invalida");
      }
      const bytes = await bytesLimitados(respuesta);
      if (bytes.length < 5 || String.fromCharCode(...bytes.slice(0, 5)) !== "%PDF-") throw fallo("respuesta_invalida");
      return { bytes, nombre: "justificante.pdf", tipo: "application/pdf" };
    },
  });
}
