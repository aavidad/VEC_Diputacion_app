export const RUTA_CAUSAS_PARTICIPACION = "/api/vec/bolsa/causas-participacion";
export const RUTA_PROPUESTAS_CAUSAS = `${RUTA_CAUSAS_PARTICIPACION}/propuestas`;

const CODIGO = /^[a-z][a-z0-9_]{2,63}$/u;
const SHA256 = /^[a-f0-9]{64}$/u;
const RECIBO = /^recibo:causa:[a-f0-9]{64}$/u;
const PROPUESTA_REF = /^propuesta:causa:[a-f0-9]{64}$/u;
const RECIBO_PROPUESTA = /^recibo:propuesta:causa:[a-f0-9]{64}$/u;
const LIMITE_RESPUESTA = 262144;
const LIMITE_ENTRADAS = 500;
const LIMITE_ESPERA_MS = 8000;
const bytes = (texto) => new TextEncoder().encode(texto).length;

function objeto(valor) {
  return valor !== null && typeof valor === "object" && !Array.isArray(valor);
}

function etiquetaValida(valor) {
  return typeof valor === "string" && valor === valor.trim() && bytes(valor) >= 1
    && bytes(valor) <= 120 && !/[\p{Cc}\p{Cf}]/u.test(valor);
}

export function validarCausaEditable(valor) {
  if (!objeto(valor) || !CODIGO.test(valor.codigo ?? "")
    || !Number.isSafeInteger(valor.version) || valor.version < 1
    || !etiquetaValida(valor.etiqueta)
    || typeof valor.aplica_situacion !== "boolean" || typeof valor.aplica_contacto !== "boolean"
    || (!valor.aplica_situacion && !valor.aplica_contacto)
    || typeof valor.publicable !== "boolean" || typeof valor.activa !== "boolean") {
    throw new TypeError("causa de participación no válida");
  }
  return {
    codigo: valor.codigo, version: valor.version, etiqueta: valor.etiqueta,
    aplica_situacion: valor.aplica_situacion, aplica_contacto: valor.aplica_contacto,
    publicable: valor.publicable, activa: valor.activa,
  };
}

function validarCausaRecibida(valor, { lectura = false } = {}) {
  if (!objeto(valor) || !CODIGO.test(valor.codigo ?? "")
    || !Number.isSafeInteger(valor.version) || valor.version < 1
    || !SHA256.test(valor.huella_sha256 ?? "") || !etiquetaValida(valor.etiqueta)
    || typeof valor.aplica_situacion !== "boolean" || typeof valor.aplica_contacto !== "boolean"
    || (!valor.aplica_situacion && !valor.aplica_contacto)
    || (!lectura && (typeof valor.publicable !== "boolean" || typeof valor.activa !== "boolean"))) {
    throw new TypeError("catálogo causal recibido no válido");
  }
  return lectura ? {
    codigo: valor.codigo, version: valor.version, huella_sha256: valor.huella_sha256,
    etiqueta: valor.etiqueta, aplica_situacion: valor.aplica_situacion,
    aplica_contacto: valor.aplica_contacto,
  } : {
    ...validarCausaEditable(valor), huella_sha256: valor.huella_sha256,
  };
}

export function validarCatalogoCausasRecibido(sobre) {
  if (!objeto(sobre) || !Array.isArray(sobre.data) || sobre.data.length > LIMITE_ENTRADAS) {
    throw new TypeError("catálogo causal recibido no válido");
  }
  const codigos = new Set();
  return sobre.data.map((valor) => {
    const causa = validarCausaRecibida(valor, { lectura: true });
    if (codigos.has(causa.codigo)) throw new TypeError("catálogo causal duplicado");
    codigos.add(causa.codigo);
    return causa;
  });
}

async function sha256Hex(texto) {
  if (!globalThis.crypto?.subtle?.digest) throw new TypeError("verificación SHA256 no disponible");
  const huella = new Uint8Array(await globalThis.crypto.subtle.digest("SHA-256", new TextEncoder().encode(texto)));
  return Array.from(huella, (byte) => byte.toString(16).padStart(2, "0")).join("");
}

export async function calcularHuellaCausa(entrada) {
  const e = validarCausaEditable(entrada);
  return sha256Hex(["bolsa.causa_participacion.v1", e.codigo, String(e.version), e.etiqueta,
    String(e.aplica_situacion), String(e.aplica_contacto), String(e.publicable), String(e.activa)].join("\n"));
}

export async function validarReciboCausa(sobre, enviada) {
  if (!objeto(sobre) || !objeto(sobre.data) || !RECIBO.test(sobre.data.recibo ?? "")) {
    throw new TypeError("recibo causal recibido no válido");
  }
  const catalogo = validarCausaRecibida(sobre.data.catalogo);
  for (const campo of Object.keys(enviada)) {
    if (catalogo[campo] !== enviada[campo]) throw new TypeError("recibo causal incoherente");
  }
  const huella = await calcularHuellaCausa(enviada);
  const recibo = `recibo:causa:${await sha256Hex(`bolsa.causa_participacion.recibo.v1\n${huella}`)}`;
  if (catalogo.huella_sha256 !== huella || sobre.data.recibo !== recibo) {
    throw new TypeError("huella o recibo causal incoherente");
  }
  return { catalogo, recibo: sobre.data.recibo };
}

export async function validarReciboPropuesta(sobre, enviada) {
  if (!objeto(sobre) || !objeto(sobre.data) || !objeto(sobre.data.propuesta)
    || !PROPUESTA_REF.test(sobre.data.propuesta.propuesta_ref ?? "")
    || !RECIBO_PROPUESTA.test(sobre.data.recibo ?? "")) {
    throw new TypeError("recibo de propuesta causal no válido");
  }
  const propuesta = validarCausaRecibida(sobre.data.propuesta);
  for (const campo of Object.keys(enviada)) {
    if (propuesta[campo] !== enviada[campo]) throw new TypeError("propuesta causal incoherente");
  }
  if (propuesta.huella_sha256 !== await calcularHuellaCausa(enviada)) {
    throw new TypeError("huella de propuesta causal incoherente");
  }
  const propuestaRef = sobre.data.propuesta.propuesta_ref;
  const recibo = `recibo:propuesta:causa:${await sha256Hex(`bolsa.causa_participacion.propuesta.recibo.v1\n${propuestaRef}`)}`;
  if (sobre.data.recibo !== recibo) throw new TypeError("recibo de propuesta causal incoherente");
  return { propuesta: { ...propuesta, propuesta_ref: propuestaRef }, recibo };
}

export async function validarConsultaPropuesta(sobre, propuestaRef) {
  if (!objeto(sobre) || !objeto(sobre.data)
    || !["pendiente", "publicada", "superada"].includes(sobre.data.estado)
    || sobre.data.propuesta?.propuesta_ref !== propuestaRef) {
    throw new TypeError("consulta de propuesta causal no válida");
  }
  const entrada = validarCausaEditable(sobre.data.propuesta);
  const recibida = await validarReciboPropuesta(sobre, entrada);
  return { ...recibida, estado: sobre.data.estado };
}

function combinarSenales(externa, interna) {
  if (!externa) return interna;
  if (typeof AbortSignal.any === "function") return AbortSignal.any([externa, interna]);
  const combinada = new AbortController();
  const abortar = () => combinada.abort();
  externa.addEventListener("abort", abortar, { once: true });
  interna.addEventListener("abort", abortar, { once: true });
  if (externa.aborted || interna.aborted) combinada.abort();
  return combinada.signal;
}

async function leerJSONLimitado(respuesta) {
  if (!/^application\/json(?:\s*;|\s*$)/iu.test(respuesta.headers.get("content-type") ?? "")) {
    throw new TypeError("respuesta causal no JSON");
  }
  if (Number(respuesta.headers.get("content-length")) > LIMITE_RESPUESTA) {
    throw new TypeError("respuesta causal demasiado grande");
  }
  const lector = respuesta.body?.getReader?.();
  if (!lector) throw new TypeError("respuesta causal sin cuerpo");
  const fragmentos = [];
  let longitud = 0;
  try {
    for (;;) {
      const { done, value } = await lector.read();
      if (done) break;
      longitud += value.byteLength;
      if (longitud > LIMITE_RESPUESTA) throw new TypeError("respuesta causal demasiado grande");
      fragmentos.push(value);
    }
  } finally { lector.releaseLock(); }
  const completo = new Uint8Array(longitud);
  let posicion = 0;
  for (const fragmento of fragmentos) { completo.set(fragmento, posicion); posicion += fragmento.byteLength; }
  return JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(completo));
}

export function crearClienteCausasParticipacion({ fetchImpl = fetch, tiempoMs = LIMITE_ESPERA_MS } = {}) {
  if (typeof fetchImpl !== "function" || !Number.isSafeInteger(tiempoMs) || tiempoMs < 1 || tiempoMs > 30000) {
    throw new TypeError("cliente causal no válido");
  }
  async function solicitar(ruta, method, cuerpo, signal) {
    const tiempo = new AbortController();
    const temporizador = setTimeout(() => tiempo.abort(), tiempoMs);
    try {
      const respuesta = await fetchImpl(ruta, {
        method, credentials: "same-origin", mode: "same-origin", cache: "no-store",
        redirect: "error", referrerPolicy: "no-referrer", signal: combinarSenales(signal, tiempo.signal),
        headers: method === "POST" ? { Accept: "application/json", "Content-Type": "application/json" }
          : { Accept: "application/json" },
        ...(cuerpo === undefined ? {} : { body: cuerpo }),
      });
      if (!respuesta.ok) return { ok: false, status: respuesta.status };
      const data = await leerJSONLimitado(respuesta);
      return { ok: true, status: respuesta.status, data };
    } finally { clearTimeout(temporizador); }
  }
  return Object.freeze({
    async consultar({ signal } = {}) {
      const resultado = await solicitar(RUTA_CAUSAS_PARTICIPACION, "GET", undefined, signal);
      if (!resultado.ok) return resultado;
      if (resultado.status !== 200) throw new TypeError("estado de consulta causal no válido");
      return { ok: true, causas: validarCatalogoCausasRecibido(resultado.data) };
    },
    async proponer(entrada, { signal } = {}) {
      const enviada = validarCausaEditable(entrada);
      const cuerpo = JSON.stringify(enviada);
      if (bytes(cuerpo) > 4096) throw new TypeError("comando causal demasiado grande");
      const resultado = await solicitar(RUTA_PROPUESTAS_CAUSAS, "POST", cuerpo, signal);
      if (!resultado.ok) return resultado;
      if (resultado.status !== 201) throw new TypeError("estado de propuesta causal no válido");
      return { ok: true, ...await validarReciboPropuesta(resultado.data, enviada) };
    },
    async consultarPropuesta(propuestaRef, { signal } = {}) {
      if (typeof propuestaRef !== "string" || !PROPUESTA_REF.test(propuestaRef)) {
        throw new TypeError("referencia de propuesta causal no válida");
      }
      const ruta = `${RUTA_PROPUESTAS_CAUSAS}/${propuestaRef}`;
      const resultado = await solicitar(ruta, "GET", undefined, signal);
      if (!resultado.ok) return resultado;
      if (resultado.status !== 200) throw new TypeError("estado de consulta de propuesta causal no válido");
      return { ok: true, ...await validarConsultaPropuesta(resultado.data, propuestaRef) };
    },
    async publicar(propuestaRef, entrada, { signal } = {}) {
      if (typeof propuestaRef !== "string" || !PROPUESTA_REF.test(propuestaRef)) {
        throw new TypeError("referencia de propuesta causal no válida");
      }
      const enviada = validarCausaEditable(entrada);
      const cuerpo = JSON.stringify(enviada);
      if (bytes(cuerpo) > 4096) throw new TypeError("comando causal demasiado grande");
      const resultado = await solicitar(`${RUTA_PROPUESTAS_CAUSAS}/${propuestaRef}/publicar`, "POST", cuerpo, signal);
      if (!resultado.ok) return resultado;
      if (resultado.status !== 201) throw new TypeError("estado de publicación causal no válido");
      return { ok: true, ...await validarReciboCausa(resultado.data, enviada) };
    },
  });
}
