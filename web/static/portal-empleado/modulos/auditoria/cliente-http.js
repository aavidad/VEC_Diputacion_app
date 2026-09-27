const RUTA = "/api/vec/auditoria/consultas";
const MAX_RESPUESTA = 256 * 1024;
const LIMITE_MS = 15000;

const referencia = (valor, maximo = 512) => typeof valor === "string" && valor.length > 0 && valor.length <= maximo
  && !/[\x00-\x20\x7f*?%\\/]/u.test(valor) && !valor.includes("..");
const instante = (valor) => typeof valor === "string" && /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?Z$/u.test(valor)
  && Number.isFinite(Date.parse(valor));
const fallo = (codigo, estado = 0) => Object.assign(new Error(codigo), { codigo, estado });

function validarConsulta(entrada) {
  if (!entrada || !referencia(entrada.expediente_ref) || !instante(entrada.desde) || !instante(entrada.hasta)
    || Date.parse(entrada.hasta) <= Date.parse(entrada.desde)
    || Date.parse(entrada.hasta) - Date.parse(entrada.desde) > 31 * 86400000
    || !referencia(entrada.finalidad, 128) || !referencia(entrada.motivo, 128)
    || (entrada.actor_ref && !referencia(entrada.actor_ref))
    || (entrada.cursor && !referencia(entrada.cursor, 512))) throw fallo("consulta_invalida");
  return {
    expediente_ref: entrada.expediente_ref, desde: entrada.desde, hasta: entrada.hasta,
    ...(entrada.actor_ref ? { actor_ref: entrada.actor_ref } : {}), finalidad: entrada.finalidad,
    motivo: entrada.motivo, limite: 50, ...(entrada.cursor ? { cursor: entrada.cursor } : {}),
  };
}

async function leerLimitado(respuesta) {
  const longitud = respuesta.headers?.get?.("content-length");
  if (longitud && (!/^\d+$/u.test(longitud) || Number(longitud) > MAX_RESPUESTA)) throw fallo("respuesta_invalida", respuesta.status);
  if (!/^application\/json(?:;|$)/iu.test(respuesta.headers?.get?.("content-type") || "")) throw fallo("respuesta_invalida", respuesta.status);
  const lector = respuesta.body?.getReader?.();
  if (!lector) throw fallo("respuesta_invalida", respuesta.status);
  const partes = []; let total = 0;
  try {
    while (true) {
      const tramo = await lector.read();
      if (tramo.done) break;
      if (!(tramo.value instanceof Uint8Array) || (total += tramo.value.byteLength) > MAX_RESPUESTA) throw fallo("respuesta_invalida", respuesta.status);
      partes.push(tramo.value);
    }
  } finally { try { lector.releaseLock?.(); } catch {} }
  const bytes = new Uint8Array(total); let posicion = 0;
  for (const parte of partes) { bytes.set(parte, posicion); posicion += parte.byteLength; }
  try { return JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(bytes)); }
  catch { throw fallo("respuesta_invalida", respuesta.status); }
}

export function crearFuenteAuditoriaHTTP({ fetchImpl = globalThis.fetch } = {}) {
  if (typeof fetchImpl !== "function") throw new TypeError("cliente de Auditoría no disponible");
  return Object.freeze({
    async consultar(entrada, { signal } = {}) {
      const cuerpo = JSON.stringify(validarConsulta(entrada));
      const controlador = new AbortController();
      const abortar = () => controlador.abort();
      signal?.addEventListener("abort", abortar, { once: true });
      const temporizador = setTimeout(abortar, LIMITE_MS);
      try {
        const respuesta = await fetchImpl(RUTA, {
          method: "POST", headers: { Accept: "application/json", "Content-Type": "application/json" },
          body: cuerpo, signal: controlador.signal, credentials: "same-origin", mode: "same-origin",
          cache: "no-store", redirect: "error", referrerPolicy: "no-referrer",
        });
        if (respuesta?.redirected || !respuesta) throw fallo("respuesta_invalida");
        if ([401, 403, 404].includes(respuesta.status)) throw fallo("denegado", respuesta.status);
        if (!respuesta.ok || respuesta.status !== 200) throw fallo("consulta_fallida", respuesta.status);
        return await leerLimitado(respuesta);
      } catch (error) {
        if (signal?.aborted || controlador.signal.aborted) throw fallo("cancelado");
        throw error;
      } finally { clearTimeout(temporizador); signal?.removeEventListener("abort", abortar); }
    },
  });
}
