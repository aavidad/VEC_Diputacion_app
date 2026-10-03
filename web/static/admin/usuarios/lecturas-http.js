const BASE = "/api/admin/perfiles/v1";
const MAXIMO = 256 * 1024;
const REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9_:.-]{2,255}$/u;
function fallo(estado = 0) { return Object.assign(new Error("lectura_no_disponible"), { estado }); }
async function leer(respuesta) {
  if (Number(respuesta.headers?.get?.("Content-Length")) > MAXIMO) throw fallo(respuesta.status);
  if (!respuesta.body?.getReader) {
    const texto = await respuesta.text();
    if (new TextEncoder().encode(texto).byteLength > MAXIMO) throw fallo(respuesta.status);
    return JSON.parse(texto);
  }
  const lector = respuesta.body.getReader();
  const trozos = []; let total = 0;
  try {
    for (;;) {
      const { done, value } = await lector.read(); if (done) break;
      total += value.byteLength; if (total > MAXIMO) throw fallo(respuesta.status);
      trozos.push(value);
    }
  } catch (e) { await lector.cancel().catch(() => {}); throw e; }
  finally { lector.releaseLock(); }
  const bytes = new Uint8Array(total); let indice = 0;
  for (const trozo of trozos) { bytes.set(trozo, indice); indice += trozo.byteLength; }
  return JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(bytes));
}
/** Lecturas del contrato ADMIN central; no contiene operaciones de escritura. */
export function crearClienteLecturasUsuarios({ fetchImpl = globalThis.fetch, origen = globalThis.location?.origin } = {}) {
  if (typeof fetchImpl !== "function" || typeof origen !== "string" || new URL(origen).origin !== origen || !/^https?:/u.test(origen)) throw new TypeError("transporte_invalido");
  async function pedir(ruta, signal) {
    const url = new URL(BASE + ruta, origen);
    if (url.origin !== origen) throw new TypeError("destino_invalido");
    const control = new AbortController();
    const abortar = () => control.abort();
    if (signal?.aborted) abortar(); else signal?.addEventListener("abort", abortar, { once: true });
    const limite = setTimeout(abortar, 15000);
    try {
      const respuesta = await fetchImpl(url.href, { method: "GET", credentials: "same-origin", redirect: "error",
        referrerPolicy: "no-referrer", cache: "no-store", signal: control.signal, headers: { Accept: "application/json" } });
      if (respuesta?.redirected || !/^application\/json(?:\s*;|$)/iu.test(respuesta?.headers?.get?.("Content-Type") || "")) throw fallo(respuesta?.status);
      if (!respuesta.ok) throw fallo(respuesta.status);
      const datos = await leer(respuesta);
      if (!datos || typeof datos !== "object" || Array.isArray(datos)) throw fallo(respuesta.status);
      return datos;
    } catch (e) { if (e?.name === "AbortError" || Number.isInteger(e?.estado)) throw e; throw fallo(); }
    finally { clearTimeout(limite); signal?.removeEventListener("abort", abortar); }
  }
  return Object.freeze({
    capacidades: (signal) => pedir("/capacidades", signal),
    roles: (signal) => pedir("/roles", signal),
    persona: (ref, signal) => { if (typeof ref !== "string" || !REFERENCIA.test(ref)) throw new TypeError("referencia_invalida"); return pedir(`/personas/${encodeURIComponent(ref)}`, signal); },
    buscar: (filtros = {}, signal) => {
      const { busqueda = "", cursor = "", perfil_ref = "", unidad_ref = "", estado = "" } = filtros;
      if (typeof busqueda !== "string" || (busqueda && (busqueda.trim().length < 2 || busqueda.length > 132))
        || typeof cursor !== "string" || cursor.length > 256 || (typeof perfil_ref !== "string" || (perfil_ref && !REFERENCIA.test(perfil_ref)))
        || (typeof unidad_ref !== "string" || (unidad_ref && !REFERENCIA.test(unidad_ref))) || !["", "vigente", "caducado"].includes(estado)) throw new TypeError("filtros_invalidos");
      const parametros = new URLSearchParams();
      for (const [nombre, valor] of [["q", busqueda], ["cursor", cursor], ["perfil_ref", perfil_ref], ["unidad_ref", unidad_ref], ["estado", estado]]) if (valor) parametros.set(nombre, valor);
      return pedir(`/personas${parametros.size ? `?${parametros}` : ""}`, signal);
    },
  });
}
