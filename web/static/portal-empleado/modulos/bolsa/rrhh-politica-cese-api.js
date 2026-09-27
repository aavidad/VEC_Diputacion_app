/** Consulta RRHH de la política B45 efectiva; sin credenciales persistidas. */
export const RUTA_POLITICA_CESE = "/api/vec/bolsa/politica-cese";
const ESQUEMA = "vec.bolsa.rrhh.politica_cese.v1";
const HUELLA = /^[a-f0-9]{64}$/u;
const CLAVE = /^[a-z][a-z0-9._-]{1,79}(\|[a-z][a-z0-9._-]{1,79})?$/u;
const FECHA = /^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(?:\.\d+)?Z$/u;
const MAXIMO = 64 * 1024;
const CLASES = new Set(["general", "acumulacion_tareas"]);
const bytesUTF8 = (valor) => new TextEncoder().encode(valor).byteLength;
const referenciaCatalogoValida = (valor) => typeof valor === "string"
  && valor === valor.replace(/^ +| +$/gu, "") && bytesUTF8(valor) >= 1 && bytesUTF8(valor) <= 512;

export function validarPoliticaCeseRRHH(sobre) {
  const data = sobre?.data;
  const p = data?.politica_cese;
  if (data?.esquema !== ESQUEMA || !p || typeof p !== "object" || Array.isArray(p)
    || !Number.isSafeInteger(p.version) || p.version < 1
    || !referenciaCatalogoValida(p.catalogo_ref)
    || typeof p.catalogo_sha256 !== "string" || !HUELLA.test(p.catalogo_sha256)
    || !Number.isSafeInteger(p.meses_general) || p.meses_general < 0 || p.meses_general > 120
    || !Number.isSafeInteger(p.meses_acumulacion) || p.meses_acumulacion < 0 || p.meses_acumulacion > 120
    || p.computo !== "fecha_cese_meses_calendario_ajuste_fin_mes"
    || p.estado !== "ejemplo_sintetico" || typeof p.publicada_en !== "string"
    || !FECHA.test(p.publicada_en) || !Number.isFinite(Date.parse(p.publicada_en))
    || !p.mapeo || typeof p.mapeo !== "object" || Array.isArray(p.mapeo)) {
    throw new TypeError("política de cese incompatible");
  }
  const entradas = Object.entries(p.mapeo);
  if (entradas.length < 1 || entradas.length > 100
    || bytesUTF8(JSON.stringify(p.mapeo)) > 16 * 1024
    || entradas.some(([clave, clase]) => !CLAVE.test(clave) || !CLASES.has(clase))) {
    throw new TypeError("mapeo de política de cese incompatible");
  }
  return Object.freeze({ ...p, mapeo: Object.freeze(Object.fromEntries(entradas)) });
}

async function leerJSONAcotado(respuesta) {
  if (!/^application\/json(?:;|$)/iu.test(respuesta.headers?.get?.("content-type") || "")) {
    throw new TypeError("respuesta de política de cese incompatible");
  }
  const longitud = respuesta.headers?.get?.("content-length");
  if (longitud && (!/^\d+$/u.test(longitud) || Number(longitud) > MAXIMO)) throw new TypeError("respuesta excesiva");
  const lector = respuesta.body?.getReader?.();
  if (!lector) throw new TypeError("respuesta de política de cese incompatible");
  const partes = [];
  let total = 0;
  try {
    for (;;) {
      const { done, value } = await lector.read();
      if (done) break;
      if (!(value instanceof Uint8Array) || (total += value.byteLength) > MAXIMO) throw new TypeError("respuesta excesiva");
      partes.push(value);
    }
  } catch (error) {
    await lector.cancel().catch(() => {});
    throw error;
  } finally { try { lector.releaseLock(); } catch {} }
  const bytes = new Uint8Array(total);
  let posicion = 0;
  for (const parte of partes) { bytes.set(parte, posicion); posicion += parte.byteLength; }
  return JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(bytes));
}

export function crearClientePoliticaCeseRRHH({ fetchImpl = globalThis.fetch } = {}) {
  if (typeof fetchImpl !== "function") throw new TypeError("cliente de política de cese no disponible");
  return Object.freeze({
    async consultar({ signal } = {}) {
      const controlador = new AbortController();
      const abortar = () => controlador.abort();
      if (signal?.aborted) abortar();
      else signal?.addEventListener?.("abort", abortar, { once: true });
      const temporizador = setTimeout(abortar, 15_000);
      try {
        // El mismo origen conserva el certificado TLS cliente; JS no añade
        // Authorization ni Cookie y el servidor vuelve a evaluar V3.
        const respuesta = await fetchImpl(RUTA_POLITICA_CESE, { method: "GET", signal: controlador.signal,
          credentials: "same-origin", mode: "same-origin", cache: "no-store",
          redirect: "error", referrerPolicy: "no-referrer", headers: { Accept: "application/json" } });
        if (!respuesta || respuesta.redirected || !Number.isInteger(respuesta.status)) throw new TypeError("respuesta incompatible");
        if (!respuesta.ok) throw Object.assign(new Error("consulta de política de cese denegada"), { estado: respuesta.status });
        if (respuesta.status !== 200) throw new TypeError("estado de política de cese incompatible");
        return validarPoliticaCeseRRHH(await leerJSONAcotado(respuesta));
      } finally {
        clearTimeout(temporizador);
        signal?.removeEventListener?.("abort", abortar);
      }
    },
  });
}
