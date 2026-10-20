/**
 * Nombres de las personas a las que se dirigió cada llamamiento de Bolsa que
 * se puede vincular a un expediente. Usa la misma lectura autorizada de
 * candidatos que la pantalla de Bolsas: sus contactos dicen qué participación
 * recibió cada llamamiento y sus filas, el nombre visible del acta. No guarda
 * nada fuera de memoria. Si la lectura falla, el selector sigue sin nombres.
 */
const RUTA_BOLSAS = "/api/vec/bolsa/bolsas";
const REFERENCIA = /^[A-Za-z][A-Za-z0-9:_-]{0,255}$/u;
const MAXIMO_PAGINAS = 10;

const segmento = (referencia) => encodeURIComponent(referencia).replace(/%3A/giu, ":");

function texto(valor, maximo) {
  return typeof valor === "string" && valor.trim() !== "" && valor.length <= maximo;
}

/** Valida lo justo de una página de candidatos para cruzar nombres y llamamientos. */
export function validarPaginaPersonasBolsa(sobre, bolsaRef) {
  const datos = sobre?.data;
  if (!datos || datos.bolsa?.bolsa_ref !== bolsaRef || !Array.isArray(datos.candidatos) || datos.candidatos.length > 100
    || !Array.isArray(datos.contactos) || typeof datos.hay_mas !== "boolean"
    || (datos.hay_mas && !texto(datos.cursor_siguiente, 256))) throw new TypeError("personas_bolsa_contrato");
  const candidatos = datos.candidatos.map((c) => {
    if (!REFERENCIA.test(c?.participacion_ref ?? "") || !texto(c.nombre_visible, 200)) throw new TypeError("personas_bolsa_candidato");
    return { participacion_ref: c.participacion_ref, nombre_visible: c.nombre_visible.trim() };
  });
  const contactos = datos.contactos.filter((c) => c?.llamamiento_ref !== null && c?.llamamiento_ref !== undefined).map((c) => {
    if (!REFERENCIA.test(c.participacion_ref ?? "") || !REFERENCIA.test(c.llamamiento_ref ?? "")) throw new TypeError("personas_bolsa_contacto");
    return { participacion_ref: c.participacion_ref, llamamiento_ref: c.llamamiento_ref };
  });
  return { candidatos, contactos, cursor: datos.hay_mas ? datos.cursor_siguiente : null };
}

/**
 * Devuelve un Map llamamiento_ref → nombres visibles, en el orden en que la
 * Bolsa registró a cada persona. Solo pide más páginas mientras falte algún
 * nombre de los llamamientos buscados.
 */
export async function consultarPersonasLlamamientos(emisiones, { fetchImpl = globalThis.fetch, signal } = {}) {
  const resultado = new Map();
  const porBolsa = new Map();
  for (const emision of emisiones ?? []) {
    if (!REFERENCIA.test(emision?.bolsa_ref ?? "") || !REFERENCIA.test(emision?.llamamiento_ref ?? "")) continue;
    if (!porBolsa.has(emision.bolsa_ref)) porBolsa.set(emision.bolsa_ref, new Set());
    porBolsa.get(emision.bolsa_ref).add(emision.llamamiento_ref);
  }
  for (const [bolsaRef, llamamientos] of porBolsa) {
    const nombres = new Map();
    let participaciones = null;
    let cursor = "";
    for (let pagina = 0; pagina < MAXIMO_PAGINAS; pagina += 1) {
      const consulta = new URLSearchParams({ limite: "100" });
      if (cursor) consulta.set("cursor", cursor);
      const respuesta = await fetchImpl(`${RUTA_BOLSAS}/${segmento(bolsaRef)}/candidatos?${consulta}`, {
        method: "GET", credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error",
        referrerPolicy: "no-referrer", signal, headers: { Accept: "application/json" } });
      if (!respuesta.ok) throw Object.assign(new Error("personas_bolsa_no_disponibles"), { estado: respuesta.status });
      const datos = validarPaginaPersonasBolsa(await respuesta.json(), bolsaRef);
      if (participaciones === null) {
        participaciones = new Map([...llamamientos].map((ref) => [ref, []]));
        for (const contacto of datos.contactos) {
          const lista = participaciones.get(contacto.llamamiento_ref);
          if (lista && !lista.includes(contacto.participacion_ref)) lista.push(contacto.participacion_ref);
        }
      }
      for (const candidato of datos.candidatos) nombres.set(candidato.participacion_ref, candidato.nombre_visible);
      const faltan = [...participaciones.values()].some((lista) => lista.some((ref) => !nombres.has(ref)));
      if (!faltan || !datos.cursor) break;
      cursor = datos.cursor;
    }
    for (const [llamamiento, lista] of participaciones ?? []) {
      const visibles = lista.map((ref) => nombres.get(ref)).filter(Boolean);
      if (visibles.length && visibles.length === lista.length) resultado.set(llamamiento, visibles);
    }
  }
  return resultado;
}

/** Texto corto de las personas de un llamamiento: un nombre, o el primero y cuántas más. */
export function textoPersonasLlamamiento(nombres, t) {
  if (!Array.isArray(nombres) || !nombres.length) return "";
  return nombres.length === 1 ? nombres[0]
    : t("resultado_bolsa_personas_varias", { nombre: nombres[0], resto: nombres.length - 1 });
}
