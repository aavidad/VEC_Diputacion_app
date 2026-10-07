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
function crearTransporte({ fetchImpl = globalThis.fetch, origen = globalThis.location?.origin } = {}) {
  if (typeof fetchImpl !== "function" || typeof origen !== "string" || new URL(origen).origin !== origen || !/^https?:/u.test(origen)) throw new TypeError("transporte_invalido");
  async function pedir(ruta, signal, cuerpo, maximoCuerpo = 16384) {
    const url = new URL(BASE + ruta, origen);
    if (url.origin !== origen) throw new TypeError("destino_invalido");
    const control = new AbortController();
    const abortar = () => control.abort();
    if (signal?.aborted) abortar(); else signal?.addEventListener("abort", abortar, { once: true });
    const limite = setTimeout(abortar, 15000);
    try {
      const escritura = cuerpo !== undefined;
      if (escritura && !origen.startsWith("https://")) throw new TypeError("canal_invalido");
      const body = escritura ? JSON.stringify(cuerpo) : undefined;
      if (body && new TextEncoder().encode(body).byteLength > maximoCuerpo) throw new TypeError("solicitud_excesiva");
      const respuesta = await fetchImpl(url.href, { method: escritura ? "POST" : "GET", credentials: "same-origin", redirect: "error",
        referrerPolicy: "no-referrer", cache: "no-store", signal: control.signal, body,
        headers: { Accept: "application/json", ...(escritura ? { "Content-Type": "application/json" } : {}) } });
      if (respuesta?.redirected || !/^application\/json(?:\s*;|$)/iu.test(respuesta?.headers?.get?.("Content-Type") || "")) throw fallo(respuesta?.status);
      if (!respuesta.ok) throw fallo(respuesta.status);
      const datos = await leer(respuesta);
      if (!datos || typeof datos !== "object" || Array.isArray(datos)) throw fallo(respuesta.status);
      return datos;
    } catch (e) { if (e?.name === "AbortError" || Number.isInteger(e?.estado)) throw e; throw fallo(); }
    finally { clearTimeout(limite); signal?.removeEventListener("abort", abortar); }
  }
  return pedir;
}
/** Transporte de solo lectura, también cuando el servidor anuncie cambios. */
export function crearClienteLecturasUsuarios(opciones = {}) {
  if (opciones.proyeccion !== undefined && opciones.proyeccion !== "metadatos_v1") throw new TypeError("proyeccion_invalida");
  const pedir = crearTransporte(opciones);
  return Object.freeze({
    ...(opciones.proyeccion ? { proyeccion: opciones.proyeccion } : {}),
    capacidades: (signal) => pedir("/capacidades", signal),
    roles: (signal) => pedir("/roles", signal),
    propuestas: (signal) => pedir("/propuestas?estado=pendiente", signal),
    persona: (ref, signal) => { if (typeof ref !== "string" || !REFERENCIA.test(ref)) throw new TypeError("referencia_invalida"); return pedir(`/personas/${encodeURIComponent(ref)}`, signal); },
    buscar: (filtros = {}, signal) => {
      const { busqueda = "", cursor = "", perfil_ref = "", unidad_ref = "", estado = "" } = filtros;
      if (opciones.proyeccion === "metadatos_v1" && busqueda !== "") throw new TypeError("filtros_invalidos");
      if (typeof busqueda !== "string" || (busqueda && (busqueda.trim().length < 2 || busqueda.length > 132))
        || typeof cursor !== "string" || cursor.length > 256 || (typeof perfil_ref !== "string" || (perfil_ref && !REFERENCIA.test(perfil_ref)))
        || (typeof unidad_ref !== "string" || (unidad_ref && !REFERENCIA.test(unidad_ref))) || !["", "vigente", "caducado"].includes(estado)) throw new TypeError("filtros_invalidos");
      const parametros = new URLSearchParams();
      for (const [nombre, valor] of [["q", busqueda], ["cursor", cursor], ["perfil_ref", perfil_ref], ["unidad_ref", unidad_ref], ["estado", estado]]) if (valor) parametros.set(nombre, valor);
      return pedir(`/personas${parametros.size ? `?${parametros}` : ""}`, signal);
    },
  });
}
/** Inyectable solo cuando la composición disponga de una autoridad durable. */
export function crearClienteActosUsuarios(opciones = {}) {
  const pedir = crearTransporte(opciones);
  const singular = (ruta, cuerpo, signal) => {
    if (!cuerpo || typeof cuerpo !== "object" || Array.isArray(cuerpo) || cuerpo.solicitudes
      || Object.keys(cuerpo).some((k) => !["operacion_ref", "operacion", "rol_version_ref", "objetivo", "motivo", "referencia_acto"].includes(k))) throw new TypeError("solicitud_invalida");
    return pedir(ruta, signal, cuerpo);
  };
  return Object.freeze({
    aplicar: (cuerpo, signal) => singular("/actos-ordinarios", cuerpo, signal),
    proponer: (cuerpo, signal) => singular("/propuestas", cuerpo, signal),
    aplicarLote: (cuerpo, signal) => {
      if (!cuerpo || !Array.isArray(cuerpo.cambios) || cuerpo.cambios.length < 1 || cuerpo.cambios.length > 32
        || Object.keys(cuerpo).some((k) => !["operacion_ref", "cambios", "motivo", "referencia_acto"].includes(k))) throw new TypeError("solicitud_invalida");
      return pedir("/lotes-ordinarios", signal, cuerpo, 65536);
    },
    cerrarPropuesta: (ref, cuerpo, signal) => {
      if (!/^propuesta_admin:[a-f0-9]{32}$/u.test(ref) || !cuerpo || typeof cuerpo !== "object"
        || Object.keys(cuerpo).some((k) => !["operacion_ref", "propuesta_huella_sha256", "decision", "motivo"].includes(k))) throw new TypeError("solicitud_invalida");
      return pedir(`/propuestas/${encodeURIComponent(ref)}/cierre`, signal, cuerpo);
    },
  });
}
