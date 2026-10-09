/** Lectura y descarga de tipos publicados para un expediente CT ya consultado. */
export const RUTA_BORRADORES_PUBLICADOS = "/api/vec/contratacion-temporal/expedientes/borradores";
export const RUTA_BORRADORES_DISPONIBLES = `${RUTA_BORRADORES_PUBLICADOS}/disponibles`;
const ESQUEMA = "vec.contratacion-temporal.borradores-disponibles.v1";
const REF = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
const CLAVE = /^[a-z][a-z0-9._-]{1,79}$/u;
const HUELLA = /^[a-f0-9]{64}$/u;
const RECIBO_PUBLICACION = /^recibo:[0-9a-f]{8}-(?:[0-9a-f]{4}-){3}[0-9a-f]{12}$/u;
const MIME = Object.freeze({ pdf: "application/pdf",
  docx: "application/vnd.openxmlformats-officedocument.wordprocessingml.document" });
const MAXIMO_CATALOGO = 64 * 1024;
const MAXIMO_DOCUMENTO = 2 * 1024 * 1024;
const MAXIMO_ERROR = 4 * 1024;

const fallo = (codigo, estado = 0) => Object.assign(new Error(codigo), { codigo, estado });
function contextoValido(contexto) {
  if (!contexto || typeof contexto.expediente_ref !== "string" || !REF.test(contexto.expediente_ref)
    || !Number.isSafeInteger(contexto.version_observada) || contexto.version_observada < 1) {
    throw fallo("contexto_invalido");
  }
  return Object.freeze({ expediente_ref: contexto.expediente_ref, version_observada: contexto.version_observada });
}

export function validarBorradoresDisponibles(respuesta) {
  if (!respuesta || typeof respuesta !== "object" || Array.isArray(respuesta)
    || Object.keys(respuesta).length !== 5
    || !["esquema", "catalogo_ref", "catalogo_huella_sha256", "procedencia_ref", "tipos"].every((clave) => Object.hasOwn(respuesta, clave))
    || respuesta.esquema !== ESQUEMA || typeof respuesta.catalogo_ref !== "string"
    || !REF.test(respuesta.catalogo_ref) || typeof respuesta.catalogo_huella_sha256 !== "string"
    || !HUELLA.test(respuesta.catalogo_huella_sha256)
    || typeof respuesta.procedencia_ref !== "string" || !RECIBO_PUBLICACION.test(respuesta.procedencia_ref)
    || !Array.isArray(respuesta.tipos)
    || respuesta.tipos.length > 64) throw fallo("catalogo_incompatible");
  const claves = new Set();
  const tipos = respuesta.tipos.map((tipo) => {
    // `disponible` es opcional: un servidor anterior no lo envía y entonces
    // todos los tipos listados se consideran preparables.
    const conDisponible = Object.hasOwn(tipo ?? {}, "disponible");
    if (!tipo || typeof tipo !== "object" || Array.isArray(tipo) || Object.keys(tipo).length !== (conDisponible ? 4 : 3)
      || !["clave", "etiqueta", "formatos"].every((clave) => Object.hasOwn(tipo, clave))
      || (conDisponible && typeof tipo.disponible !== "boolean")
      || typeof tipo.clave !== "string" || !CLAVE.test(tipo.clave) || tipo.clave === "etiquetas" || claves.has(tipo.clave)
      || typeof tipo.etiqueta !== "string" || tipo.etiqueta !== tipo.etiqueta.trim()
      || !tipo.etiqueta || tipo.etiqueta.length > 256 || /[\u0000-\u001f\u007f]/u.test(tipo.etiqueta)
      || !Array.isArray(tipo.formatos) || tipo.formatos.length < 1 || tipo.formatos.length > 2
      || tipo.formatos.some((formato) => !Object.hasOwn(MIME, formato))
      || new Set(tipo.formatos).size !== tipo.formatos.length) throw fallo("catalogo_incompatible");
    claves.add(tipo.clave);
    return Object.freeze({ clave: tipo.clave, etiqueta: tipo.etiqueta, formatos: Object.freeze([...tipo.formatos]),
      disponible: conDisponible ? tipo.disponible : true });
  });
  return Object.freeze({ catalogo_ref: respuesta.catalogo_ref,
    catalogo_huella_sha256: respuesta.catalogo_huella_sha256,
    procedencia_ref: respuesta.procedencia_ref, tipos: Object.freeze(tipos) });
}

async function leerAcotado(respuesta, maximo) {
  const longitud = respuesta.headers?.get?.("content-length");
  if (longitud && (!/^\d+$/u.test(longitud) || Number(longitud) > maximo)) throw fallo("respuesta_excesiva");
  const lector = respuesta.body?.getReader?.();
  if (!lector) throw fallo("respuesta_incompatible");
  const partes = [];
  let total = 0;
  try {
    for (;;) {
      const { done, value } = await lector.read();
      if (done) break;
      if (!(value instanceof Uint8Array) || (total += value.byteLength) > maximo) throw fallo("respuesta_excesiva");
      partes.push(value);
    }
  } catch (error) {
    await lector.cancel().catch(() => {});
    throw error;
  } finally { try { lector.releaseLock(); } catch {} }
  const bytes = new Uint8Array(total);
  let posicion = 0;
  for (const parte of partes) { bytes.set(parte, posicion); posicion += parte.byteLength; }
  return bytes;
}

// Un 409 puede ser «el expediente cambió» o «este documento aún no se puede
// preparar»; solo el código del sobre de error los distingue.
async function codigoConflicto(respuesta) {
  try {
    if (!/^application\/json(?:;|$)/iu.test(respuesta.headers?.get?.("content-type") || "")) {
      void respuesta.body?.cancel?.().catch(() => {});
      return "";
    }
    const sobre = JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(await leerAcotado(respuesta, MAXIMO_ERROR)));
    return typeof sobre?.error?.codigo === "string" ? sobre.error.codigo : "";
  } catch {
    void respuesta.body?.cancel?.()?.catch?.(() => {});
    return "";
  }
}

async function ejecutar(fetchImpl, ruta, entrada, accept, maximo, signal, limiteMs) {
  const controlador = new AbortController();
  const abortar = () => controlador.abort();
  if (signal?.aborted) abortar();
  else signal?.addEventListener?.("abort", abortar, { once: true });
  const temporizador = setTimeout(abortar, limiteMs);
  try {
    if (controlador.signal.aborted) throw fallo("cancelado");
    const respuesta = await fetchImpl(ruta, { method: "POST", signal: controlador.signal,
      headers: { Accept: accept, "Content-Type": "application/json" }, body: JSON.stringify(entrada),
      credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer" });
    if (!respuesta || respuesta.redirected || !Number.isInteger(respuesta.status)) throw fallo("respuesta_incompatible");
    if (!respuesta.ok) {
      const codigo = respuesta.status === 409 ? await codigoConflicto(respuesta) : "";
      if (respuesta.status !== 409) void respuesta.body?.cancel?.().catch(() => {});
      throw fallo([401, 403].includes(respuesta.status) ? "denegado"
        : codigo === "documento_no_disponible" ? codigo
          : respuesta.status === 409 ? "conflicto" : "servicio_no_disponible", respuesta.status);
    }
    if (respuesta.status !== 200) throw fallo("respuesta_incompatible", respuesta.status);
    const bytes = await leerAcotado(respuesta, maximo);
    if (controlador.signal.aborted) throw fallo("cancelado");
    return { respuesta, bytes };
  } finally { clearTimeout(temporizador); signal?.removeEventListener?.("abort", abortar); }
}

function hex(bytes) { return [...bytes].map((byte) => byte.toString(16).padStart(2, "0")).join(""); }

export function crearClienteBorradoresPublicados({ fetchImpl = globalThis.fetch, cryptoImpl = globalThis.crypto } = {}) {
  if (typeof fetchImpl !== "function") throw new TypeError("cliente de borradores publicados no disponible");
  return Object.freeze({
    async consultarDisponibles(contexto, { signal } = {}) {
      const { respuesta, bytes } = await ejecutar(fetchImpl, RUTA_BORRADORES_DISPONIBLES,
        contextoValido(contexto), "application/json", MAXIMO_CATALOGO, signal, 15_000);
      if (!/^application\/json(?:;|$)/iu.test(respuesta.headers.get("content-type") || "")) throw fallo("catalogo_incompatible");
      let datos;
      try { datos = JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(bytes)); }
      catch { throw fallo("catalogo_incompatible"); }
      return validarBorradoresDisponibles(datos);
    },
    async descargar(contexto, catalogo, tipo, formato, { signal } = {}) {
      const entrada = contextoValido(contexto);
      const valido = validarBorradoresDisponibles({ esquema: ESQUEMA, ...catalogo });
      if (typeof tipo !== "string" || !CLAVE.test(tipo) || !Object.hasOwn(MIME, formato)
        || !valido.tipos.some((item) => item.clave === tipo && item.formatos.includes(formato))) throw fallo("tipo_no_publicado");
      if (!valido.tipos.some((item) => item.clave === tipo && item.disponible)) throw fallo("documento_no_disponible");
      const { respuesta, bytes } = await ejecutar(fetchImpl, RUTA_BORRADORES_PUBLICADOS,
        { ...entrada, tipo, formato }, MIME[formato], MAXIMO_DOCUMENTO, signal, 30_000);
      const nombre = `${tipo}-borrador.${formato}`;
      if (respuesta.headers.get("content-type") !== MIME[formato]
        || respuesta.headers.get("content-disposition") !== `attachment; filename="${nombre}"`
        || respuesta.headers.get("x-vec-catalogo-ref") !== valido.catalogo_ref
        || respuesta.headers.get("x-vec-catalogo-huella-sha256") !== valido.catalogo_huella_sha256
        || respuesta.headers.get("x-vec-plantilla-procedencia-ref") !== valido.procedencia_ref
        || !HUELLA.test(respuesta.headers.get("x-vec-documento-sha256") || "")
        || bytes.length === 0 || (formato === "pdf" && new TextDecoder().decode(bytes.subarray(0, 5)) !== "%PDF-")
        || (formato === "docx" && !(bytes[0] === 0x50 && bytes[1] === 0x4b && bytes[2] === 0x03 && bytes[3] === 0x04))) {
        throw fallo("documento_incompatible");
      }
      if (typeof cryptoImpl?.subtle?.digest !== "function") throw fallo("huella_no_verificable");
      const huella = hex(new Uint8Array(await cryptoImpl.subtle.digest("SHA-256", bytes)));
      if (huella !== respuesta.headers.get("x-vec-documento-sha256")) throw fallo("huella_no_coincide");
      return Object.freeze({ nombre, mime: MIME[formato], huella_sha256: huella,
        catalogo_ref: valido.catalogo_ref, catalogo_huella_sha256: valido.catalogo_huella_sha256,
        procedencia_ref: valido.procedencia_ref,
        bytes });
    },
  });
}
