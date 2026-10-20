/**
 * Nombres de las personas a las que se dirigió cada llamamiento de Bolsa que
 * se puede vincular a un expediente. Usa la lectura autorizada de candidatos
 * de la pantalla de Bolsas filtrada por llamamiento, de modo que solo llegan
 * las personas de ese llamamiento y no la bolsa entera. No guarda nada fuera
 * de memoria. Si la lectura falla, el selector sigue sin nombres.
 */
const RUTA_BOLSAS = "/api/vec/bolsa/bolsas";
const REFERENCIA = /^[A-Za-z][A-Za-z0-9:_-]{0,255}$/u;
const LLAMAMIENTO = /^llamamiento:[A-Za-z0-9:_-]{8,200}$/u;
// Un expediente vincula pocos llamamientos; por encima no se piden nombres.
const MAXIMO_LLAMAMIENTOS = 20;

const segmento = (referencia) => encodeURIComponent(referencia).replace(/%3A/giu, ":");

function texto(valor, maximo) {
  return typeof valor === "string" && valor.trim() !== "" && valor.length <= maximo;
}

/**
 * Valida la respuesta filtrada por un llamamiento y devuelve sus nombres en el
 * orden de la Bolsa. Cada persona debe constar contactada en ese llamamiento.
 */
export function validarPaginaPersonasBolsa(sobre, bolsaRef, llamamientoRef) {
  const datos = sobre?.data;
  if (!datos || datos.bolsa?.bolsa_ref !== bolsaRef || !Array.isArray(datos.candidatos) || datos.candidatos.length > 100
    || !Array.isArray(datos.contactos) || datos.hay_mas !== false) throw new TypeError("personas_bolsa_contrato");
  const contactadas = new Set(datos.contactos.filter((c) => c?.llamamiento_ref === llamamientoRef).map((c) => c.participacion_ref));
  return datos.candidatos.map((c) => {
    if (!REFERENCIA.test(c?.participacion_ref ?? "") || !texto(c.nombre_visible, 200)) throw new TypeError("personas_bolsa_candidato");
    if (!contactadas.has(c.participacion_ref)) throw new TypeError("personas_bolsa_sin_contacto");
    return c.nombre_visible.trim();
  });
}

/**
 * Devuelve un Map llamamiento_ref → nombres visibles. Pide cada llamamiento
 * por separado; el que falla o no trae a nadie se queda sin nombre.
 */
export async function consultarPersonasLlamamientos(emisiones, { fetchImpl = globalThis.fetch, signal } = {}) {
  const pares = new Map();
  for (const emision of emisiones ?? []) {
    if (REFERENCIA.test(emision?.bolsa_ref ?? "") && LLAMAMIENTO.test(emision?.llamamiento_ref ?? "")) {
      pares.set(emision.llamamiento_ref, emision.bolsa_ref);
    }
  }
  if (pares.size > MAXIMO_LLAMAMIENTOS) return new Map();
  const resultado = new Map();
  await Promise.allSettled([...pares].map(async ([llamamientoRef, bolsaRef]) => {
    const consulta = new URLSearchParams({ limite: "100", llamamiento: llamamientoRef });
    const respuesta = await fetchImpl(`${RUTA_BOLSAS}/${segmento(bolsaRef)}/candidatos?${consulta}`, {
      method: "GET", credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error",
      referrerPolicy: "no-referrer", signal, headers: { Accept: "application/json" } });
    if (!respuesta.ok) throw Object.assign(new Error("personas_bolsa_no_disponibles"), { estado: respuesta.status });
    const nombres = validarPaginaPersonasBolsa(await respuesta.json(), bolsaRef, llamamientoRef);
    if (nombres.length) resultado.set(llamamientoRef, nombres);
  }));
  if (signal?.aborted) throw new DOMException("cancelada", "AbortError");
  return resultado;
}

/** Texto corto de las personas de un llamamiento: un nombre, o el primero y cuántas más. */
export function textoPersonasLlamamiento(nombres, t) {
  if (!Array.isArray(nombres) || !nombres.length) return "";
  return nombres.length === 1 ? nombres[0]
    : t("resultado_bolsa_personas_varias", { nombre: nombres[0], resto: nombres.length - 1 });
}
