export const BASE_ADMIN_PERFILES = "/api/admin/perfiles/v1";
const MAXIMO_JSON = 256 * 1024;
const REFERENCIA_PERSONA = /^per_[A-Za-z0-9_-]{22,128}$/u;
const REFERENCIA_PROPUESTA = /^propuesta_admin:[a-f0-9]{32}$/u;
const REFERENCIA_RECIBO = /^recibo_admin:[a-f0-9]{32}$/u;

export class ErrorAdministracionPerfiles extends Error {
  constructor(estado = 0, codigo = "servicio_no_disponible") {
    super(codigo);
    this.name = "ErrorAdministracionPerfiles";
    this.estado = estado;
    this.codigo = codigo;
  }
}

async function leerAcotado(respuesta) {
  const longitud = Number(respuesta.headers?.get?.("Content-Length"));
  if (Number.isFinite(longitud) && longitud > MAXIMO_JSON) throw new ErrorAdministracionPerfiles(respuesta.status);
  if (!respuesta.body?.getReader) {
    const texto = await respuesta.text();
    if (texto.length > MAXIMO_JSON) throw new ErrorAdministracionPerfiles(respuesta.status);
    return JSON.parse(texto);
  }
  const lector = respuesta.body.getReader();
  const trozos = [];
  let total = 0;
  try {
    for (;;) {
      const { done, value } = await lector.read();
      if (done) break;
      total += value.byteLength;
      if (total > MAXIMO_JSON) throw new ErrorAdministracionPerfiles(respuesta.status);
      trozos.push(value);
    }
  } catch (error) {
    await lector.cancel().catch(() => {});
    throw error;
  } finally { lector.releaseLock(); }
  const bytes = new Uint8Array(total);
  let posicion = 0;
  for (const trozo of trozos) { bytes.set(trozo, posicion); posicion += trozo.byteLength; }
  return JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(bytes));
}

function objeto(valor) { return valor && typeof valor === "object" && !Array.isArray(valor); }

export function crearClienteAdministracion({ fetchImpl = globalThis.fetch, origen = globalThis.location?.origin } = {}) {
  if (typeof fetchImpl !== "function" || !origen || new URL(origen).origin !== origen) throw new TypeError("cliente no disponible");
  async function pedir(metodo, ruta, cuerpo, signal) {
    const url = new URL(BASE_ADMIN_PERFILES + ruta, origen);
    if (url.origin !== origen) throw new TypeError("ruta ajena");
    let respuesta;
    try {
      respuesta = await fetchImpl(url.href, {
        method: metodo,
        credentials: "same-origin",
        redirect: "error",
        referrerPolicy: "no-referrer",
        cache: "no-store",
        signal,
        headers: { Accept: "application/json", ...(cuerpo === undefined ? {} : { "Content-Type": "application/json" }) },
        ...(cuerpo === undefined ? {} : { body: JSON.stringify(cuerpo) }),
      });
    } catch (error) {
      if (error?.name === "AbortError") throw error;
      throw new ErrorAdministracionPerfiles();
    }
    if (!respuesta || respuesta.redirected || !String(respuesta.headers?.get?.("Content-Type") ?? "").toLowerCase().startsWith("application/json")) {
      throw new ErrorAdministracionPerfiles(respuesta?.status ?? 0);
    }
    let datos;
    try { datos = await leerAcotado(respuesta); }
    catch (error) { if (error instanceof ErrorAdministracionPerfiles) throw error; throw new ErrorAdministracionPerfiles(respuesta.status); }
    if (!objeto(datos)) throw new ErrorAdministracionPerfiles(respuesta.status);
    if (!respuesta.ok) {
      const codigo = typeof datos.error?.codigo === "string" ? datos.error.codigo : "servicio_no_disponible";
      throw new ErrorAdministracionPerfiles(respuesta.status, codigo);
    }
    return datos;
  }
  return Object.freeze({
    capacidades: (signal) => pedir("GET", "/capacidades", undefined, signal),
    buscar: (q, signal) => pedir("GET", `/personas?q=${encodeURIComponent(q)}`, undefined, signal),
    persona: (ref, signal) => { if (!REFERENCIA_PERSONA.test(ref)) throw new TypeError("referencia inválida"); return pedir("GET", `/personas/${encodeURIComponent(ref)}`, undefined, signal); },
    propuestas: (signal) => pedir("GET", "/propuestas?estado=pendiente", undefined, signal),
    roles: (signal) => pedir("GET", "/roles", undefined, signal),
    recibo: (ref, signal) => { if (!REFERENCIA_RECIBO.test(ref)) throw new TypeError("referencia inválida"); return pedir("GET", `/recibos/${encodeURIComponent(ref)}`, undefined, signal); },
    aplicar: (cuerpo, signal) => pedir("POST", "/actos-ordinarios", cuerpo, signal),
    proponer: (cuerpo, signal) => pedir("POST", "/propuestas", cuerpo, signal),
    cerrar: (ref, cuerpo, signal) => { if (!REFERENCIA_PROPUESTA.test(ref)) throw new TypeError("referencia inválida"); return pedir("POST", `/propuestas/${encodeURIComponent(ref)}/cierre`, cuerpo, signal); },
  });
}

export function nuevaOperacionRef(prefijo, cripto = globalThis.crypto) {
  if (!["acto_admin:", "propuesta_admin:", "cierre_admin:"].includes(prefijo) || !cripto?.getRandomValues) throw new TypeError("operación no disponible");
  const bytes = cripto.getRandomValues(new Uint8Array(16));
  return prefijo + Array.from(bytes, (byte) => byte.toString(16).padStart(2, "0")).join("");
}
