import { validarConsultaCircuitoRRHH, validarCircuitoRRHH } from "./contrato-circuito-rrhh.js?v=20261002-rrhh-circuito-v2";

export const RUTA_CONSULTA_CIRCUITO_RRHH = "/api/vec/contratacion-temporal/circuito/consulta";
const MAXIMO_BYTES = 128 * 1024;

async function leerRespuesta(respuesta) {
  const longitud = respuesta.headers.get("Content-Length");
  if (longitud !== null && (!/^(0|[1-9][0-9]*)$/u.test(longitud) || Number(longitud) > MAXIMO_BYTES)) {
    throw new TypeError("circuito_rrhh.respuesta_no_valida");
  }
  const lector = respuesta.body?.getReader();
  if (!lector) throw new TypeError("circuito_rrhh.respuesta_no_valida");
  const fragmentos = [];
  let total = 0;
  try {
    for (;;) {
      const { done, value } = await lector.read();
      if (done) break;
      total += value.byteLength;
      if (total > MAXIMO_BYTES || fragmentos.length >= 4096) throw new TypeError("circuito_rrhh.respuesta_no_valida");
      fragmentos.push(value);
    }
  } finally { void lector.cancel().catch(() => {}); }
  const contenido = new Uint8Array(total);
  let posicion = 0;
  for (const fragmento of fragmentos) { contenido.set(fragmento, posicion); posicion += fragmento.byteLength; }
  return JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(contenido));
}

/** El contexto efectivo de identidad y competencia procede del servidor. */
export function crearClienteCircuitoRRHH({ fetchImpl = globalThis.fetch } = {}) {
  return Object.freeze({
    async consultar(consulta, { signal } = {}) {
      const entrada = validarConsultaCircuitoRRHH(consulta);
      try {
        if (typeof fetchImpl !== "function") return Object.freeze({ estado: "no_disponible" });
        const respuesta = await fetchImpl(RUTA_CONSULTA_CIRCUITO_RRHH, {
          method: "POST", headers: { Accept: "application/json", "Content-Type": "application/json" },
          body: JSON.stringify(entrada), signal, mode: "same-origin", credentials: "same-origin",
          cache: "no-store", redirect: "error", referrerPolicy: "no-referrer",
        });
        if (respuesta.redirected || respuesta.status !== 200
          || !/^application\/json(?:;\s*charset=utf-8)?$/iu.test(respuesta.headers.get("Content-Type") ?? "")) {
          void respuesta.body?.cancel?.().catch(() => {});
          return Object.freeze({ estado: [401, 403].includes(respuesta.status) ? "denegado" : "no_disponible" });
        }
        const envoltorio = await leerRespuesta(respuesta);
        if (!envoltorio || Object.keys(envoltorio).length !== 1 || !Object.hasOwn(envoltorio, "data")) {
          return Object.freeze({ estado: "no_disponible" });
        }
        return Object.freeze({ estado: "disponible", datos: validarCircuitoRRHH(envoltorio.data, entrada) });
      } catch { return Object.freeze({ estado: signal?.aborted ? "cancelado" : "no_disponible" }); }
    },
  });
}
