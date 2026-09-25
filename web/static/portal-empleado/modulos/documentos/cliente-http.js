const RUTA_LISTA = "/api/vec/documentos/expedientes/consultas";
const RUTA_DESCARGA = "/api/vec/documentos/originales/descargas";
const MAX_JSON = 1024 * 1024;
const MAX_ORIGINAL = 20 * 1024 * 1024;
const LIMITE_MS = 15000;

function referencia(valor) {
  return typeof valor === "string" && (/^ref:[0-9a-f]{64}$/u.test(valor) && !/^ref:0{64}$/u.test(valor)
    || /^[a-z][a-z0-9_]{1,31}:[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/u.test(valor));
}

function fallo(codigo, estado) {
  return Object.assign(new Error(codigo), { codigo, estado });
}

async function leerLimitado(respuesta, maximo) {
  const lector = respuesta.body?.getReader();
  if (!lector) {
    const bytes = new Uint8Array(await respuesta.arrayBuffer());
    if (bytes.byteLength > maximo) throw fallo("respuesta_invalida");
    return bytes;
  }
  const partes = [];
  let longitud = 0;
  try {
    while (true) {
      const parte = await lector.read();
      if (parte.done) break;
      longitud += parte.value.byteLength;
      if (longitud > maximo) throw fallo("respuesta_invalida");
      partes.push(parte.value);
    }
  } finally {
    await lector.cancel().catch(() => {});
  }
  const bytes = new Uint8Array(longitud);
  let offset = 0;
  for (const parte of partes) { bytes.set(parte, offset); offset += parte.byteLength; }
  return bytes;
}

function nombreOriginal(cabecera, documento) {
  const coincidencia = /^attachment; filename="([A-Za-z0-9._-]{1,120})"$/u.exec(cabecera ?? "");
  if (coincidencia && !coincidencia[1].startsWith(".")) return coincidencia[1];
  const extension = documento?.mime === "application/pdf" ? ".pdf" : documento?.mime === "application/vnd.openxmlformats-officedocument.wordprocessingml.document" ? ".docx" : "";
  if (!extension) throw fallo("respuesta_invalida");
  return `documento-${documento.ref.replaceAll(/[^a-zA-Z0-9_-]/gu, "_")}${extension}`;
}

async function pedir(ruta, cuerpo, { signal, maximo, binario = false, fetchImpl = globalThis.fetch } = {}) {
  const controlador = new AbortController();
  const abortar = () => controlador.abort();
  signal?.addEventListener("abort", abortar, { once: true });
  const temporizador = setTimeout(abortar, LIMITE_MS);
  try {
    const respuesta = await fetchImpl(ruta, {
      method: "POST", body: JSON.stringify(cuerpo), signal: controlador.signal,
      headers: { "Content-Type": "application/json", Accept: binario ? "application/octet-stream" : "application/json" },
      credentials: "same-origin", mode: "same-origin", redirect: "error", referrerPolicy: "no-referrer", cache: "no-store",
    });
    if (respuesta.status === 401 || respuesta.status === 403 || respuesta.status === 404) throw fallo("denegado", respuesta.status);
    if (!respuesta.ok) throw fallo("consulta_fallida", respuesta.status);
    const bytes = await leerLimitado(respuesta, maximo);
    if (binario) {
      if (!bytes.byteLength) throw fallo("respuesta_invalida");
      return { bytes, cabeceras: respuesta.headers };
    }
    if (!/^(application\/json)(?:;|$)/iu.test(respuesta.headers.get("Content-Type") ?? "")) throw fallo("respuesta_invalida");
    let datos;
    try { datos = JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(bytes)); }
    catch { throw fallo("respuesta_invalida"); }
    return datos?.data;
  } catch (error) {
    if (signal?.aborted || controlador.signal.aborted) throw fallo("cancelado");
    throw error;
  } finally {
    clearTimeout(temporizador);
    signal?.removeEventListener("abort", abortar);
  }
}

// La referencia selecciona un recurso; el servidor resuelve actor, ámbito y V3 de su canal autenticado.
export function crearFuenteDocumentosHTTP({ expedienteRef = "", fetchImpl } = {}) {
  let expediente = expedienteRef;
  return Object.freeze({
    seleccionarExpediente(valor) {
      if (!referencia(valor)) throw fallo("referencia_invalida");
      expediente = valor;
    },
    async listar({ signal, cursor = "" } = {}) {
      if (!referencia(expediente)) throw fallo("referencia_invalida");
      if (cursor && (typeof cursor !== "string" || cursor.length > 512 || /[\s\\/?%*]/u.test(cursor))) throw fallo("referencia_invalida");
      return pedir(RUTA_LISTA, { expediente_ref: expediente, cursor, limite: 50 }, { signal, maximo: MAX_JSON, fetchImpl });
    },
    async descargar(ref, { version, signal, mime, huella } = {}) {
      if (!referencia(ref) || !Number.isSafeInteger(version) || version < 1 || version > 2147483647) throw fallo("referencia_invalida");
      const { bytes, cabeceras } = await pedir(RUTA_DESCARGA, { documento_ref: ref, version }, { signal, maximo: MAX_ORIGINAL, binario: true, fetchImpl });
      const tipo = cabeceras.get("Content-Type")?.split(";", 1)[0]?.trim();
      const huellaRespuesta = cabeceras.get("X-Content-SHA256");
      if (tipo !== mime || !/^[0-9a-f]{64}$/iu.test(huellaRespuesta ?? "") || huellaRespuesta.toLowerCase() !== huella) throw fallo("respuesta_invalida");
      if (!globalThis.crypto?.subtle) throw fallo("integridad_no_disponible");
      const calculada = Array.from(new Uint8Array(await globalThis.crypto.subtle.digest("SHA-256", bytes)), (b) => b.toString(16).padStart(2, "0")).join("");
      if (calculada !== huellaRespuesta.toLowerCase()) throw fallo("integridad_invalida");
      return { contenido: bytes, nombre: nombreOriginal(cabeceras.get("Content-Disposition"), { ref, mime }), tipo };
    },
  });
}
