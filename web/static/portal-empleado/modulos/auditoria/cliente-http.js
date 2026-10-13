const RUTA = "/api/vec/auditoria/consultas";
const RUTA_OPCIONES = "/api/vec/auditoria/opciones";
const MAX_RESPUESTA = 256 * 1024;
const LIMITE_MS = 15000;

const referencia = (valor, maximo = 512) => typeof valor === "string" && valor.length > 0 && valor.length <= maximo
  && !/[\x00-\x20\x7f*?%\\/]/u.test(valor) && !valor.includes("..");
const instante = (valor) => typeof valor === "string" && /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?Z$/u.test(valor)
  && Number.isFinite(Date.parse(valor));
const fallo = (codigo, estado = 0) => Object.assign(new Error(codigo), { codigo, estado });

function validarConsulta(entrada) {
  if (!entrada || !["ct", "bolsa", "persona"].includes(entrada.fuente) || !referencia(entrada.expediente_ref) || !instante(entrada.desde) || !instante(entrada.hasta)
    || Date.parse(entrada.hasta) <= Date.parse(entrada.desde)
    || Date.parse(entrada.hasta) - Date.parse(entrada.desde) > 31 * 86400000
    || !referencia(entrada.finalidad_ref, 128) || !referencia(entrada.motivo_ref, 128)
    || (entrada.actor_ref && !referencia(entrada.actor_ref))
    || (entrada.cursor && !referencia(entrada.cursor, 512))) throw fallo("consulta_invalida");
  return {
    fuente: entrada.fuente, expediente_ref: entrada.expediente_ref, actor_ref: entrada.actor_ref || "",
    desde: entrada.desde, hasta: entrada.hasta, limite: 50, cursor: entrada.cursor || "",
    finalidad_ref: entrada.finalidad_ref, motivo_ref: entrada.motivo_ref,
    ...(entrada.motivo_consulta !== undefined ? { motivo_consulta: validarMotivo(entrada.motivo_consulta) } : {}),
  };
}

function validarMotivo(valor) {
  if (typeof valor !== "string" || valor.trim() !== valor || valor.length < 3 || valor.length > 500
    || /[\x00-\x1f\x7f]/u.test(valor)) throw fallo("consulta_invalida");
  return valor;
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
  async function pedir(ruta, metodo, entrada, { signal } = {}) {
      const controlador = new AbortController();
      const abortar = () => controlador.abort();
      signal?.addEventListener("abort", abortar, { once: true });
      if (signal?.aborted) abortar();
      const temporizador = setTimeout(abortar, LIMITE_MS);
      try {
        const respuesta = await fetchImpl(ruta, {
          method: metodo, headers: { Accept: "application/json", ...(metodo === "POST" ? { "Content-Type": "application/json" } : {}) },
          ...(metodo === "POST" ? { body: JSON.stringify(entrada) } : {}),
          signal: controlador.signal, credentials: "same-origin", mode: "same-origin",
          cache: "no-store", redirect: "error", referrerPolicy: "no-referrer",
        });
        if (respuesta?.redirected || !respuesta) throw fallo("respuesta_invalida");
        if ([401, 403].includes(respuesta.status)) throw fallo("denegado", respuesta.status);
        if (respuesta.status === 404) throw fallo("no_disponible", respuesta.status);
        if (!respuesta.ok || respuesta.status !== 200) throw fallo("consulta_fallida", respuesta.status);
        return await leerLimitado(respuesta);
      } catch (error) {
        if (signal?.aborted || controlador.signal.aborted) throw fallo("cancelado");
        throw error;
      } finally { clearTimeout(temporizador); signal?.removeEventListener("abort", abortar); }
  }
  return Object.freeze({
    async obtenerOpciones({ signal } = {}) {
      const respuesta = await pedir(RUTA_OPCIONES, "GET", undefined, { signal });
      if (!respuesta || !referencia(respuesta.finalidad_ref, 128) || !referencia(respuesta.motivo_ref, 128)
        || !Array.isArray(respuesta.fuentes) || respuesta.fuentes.length < 1 || respuesta.fuentes.length > 3
        || new Set(respuesta.fuentes).size !== respuesta.fuentes.length || respuesta.fuentes.some((valor) => !["ct", "bolsa", "persona"].includes(valor))
        || !referencia(respuesta.permiso_requerido, 128) || typeof respuesta.es_ejemplo !== "boolean") throw fallo("respuesta_invalida");
      return Object.freeze({ finalidad_ref: respuesta.finalidad_ref, motivo_ref: respuesta.motivo_ref,
        permiso_requerido: respuesta.permiso_requerido, es_ejemplo: respuesta.es_ejemplo,
        seleccion_objetivo: respuesta.seleccion_objetivo === true,
        fuentes: Object.freeze([...respuesta.fuentes]) });
    },
    async buscarObjetivos(entrada, { signal } = {}) {
      if (!entrada || !["ct", "bolsa", "persona"].includes(entrada.fuente)
        || !referencia(entrada.finalidad_ref, 128) || !referencia(entrada.motivo_ref, 128)
        || typeof entrada.busqueda !== "string" || entrada.busqueda.length > 80
        || /[\x00-\x1f\x7f]/u.test(entrada.busqueda)
        || (entrada.despues_ref && !referencia(entrada.despues_ref))) throw fallo("consulta_invalida");
      const respuesta = await pedir(RUTA, "POST", { operacion: "buscar", fuente: entrada.fuente,
        busqueda: entrada.busqueda.trim(), despues_ref: entrada.despues_ref || "", limite: 20,
        finalidad_ref: entrada.finalidad_ref, motivo_ref: entrada.motivo_ref,
        motivo_consulta: validarMotivo(entrada.motivo_consulta) }, { signal });
      if (!Array.isArray(respuesta?.objetivos) || respuesta.objetivos.length > 20
        || typeof respuesta.despues_ref !== "string" || (respuesta.despues_ref && !referencia(respuesta.despues_ref))) throw fallo("respuesta_invalida");
      const refs = new Set();
      for (const o of respuesta.objetivos) {
        if (!referencia(o?.referencia) || refs.has(o.referencia) || typeof o.etiqueta !== "string"
          || !o.etiqueta || o.etiqueta.length > 160 || /[\x00-\x1f\x7f]/u.test(o.etiqueta)) throw fallo("respuesta_invalida");
        refs.add(o.referencia);
      }
      return Object.freeze({ objetivos: Object.freeze(respuesta.objetivos.map((o) => Object.freeze({ ...o }))), despues_ref: respuesta.despues_ref });
    },
    async consultar(entrada, { signal } = {}) {
      return pedir(RUTA, "POST", validarConsulta(entrada), { signal });
    },
  });
}
