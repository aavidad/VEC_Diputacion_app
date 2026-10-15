const RUTA_CARTERA = "/api/vec/meritos/cartera";
const RUTA_RETIRADAS = "/api/vec/meritos/cartera/retiradas";
const RUTA_DECLARACIONES = "/api/vec/meritos/cartera/declaraciones";
const RUTA_COMPROBANTES = "/api/vec/meritos/cartera/comprobantes";
const RUTA_ESTADO_COMPROBANTE = `${RUTA_COMPROBANTES}/estado`;

export class ErrorCartera extends Error {
  constructor(codigo, estado, causa) {
    super(codigo, { cause: causa });
    this.name = "ErrorCartera";
    this.codigo = codigo;
    this.estado = estado;
  }
}

function exigirRespuesta(respuesta) {
  if (!respuesta || typeof respuesta.status !== "number" || typeof respuesta.json !== "function") {
    throw new ErrorCartera("respuesta_invalida", 503);
  }
  return respuesta;
}

async function leerJSON(respuesta) {
  try { return await respuesta.json(); }
  catch (causa) { throw new ErrorCartera("respuesta_invalida", 503, causa); }
}

function errorHTTP(respuesta, contenido) {
  const codigo = typeof contenido?.error?.codigo === "string" ? contenido.error.codigo : "servicio_no_disponible";
  return new ErrorCartera(codigo, respuesta.status);
}

const REFERENCIA_COMPROBANTE = /^ref:[A-Za-z0-9:_-]+$/u;
const FECHA_ISO = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})$/u;

function validarEstadoComprobante(data) {
  if (!data || typeof data.carga_id !== "string" || !REFERENCIA_COMPROBANTE.test(data.carga_id)
    || (data.recibo_ref !== undefined && (typeof data.recibo_ref !== "string" || !data.recibo_ref))
    || !["reservada", "recibido", "analizando", "admitido", "rechazado"].includes(data.estado)) {
    throw new ErrorCartera("respuesta_invalida", 503);
  }
  if (data.estado === "admitido") {
    if (typeof data.comprobante_ref !== "string" || !REFERENCIA_COMPROBANTE.test(data.comprobante_ref)
      || typeof data.registrado_en !== "string" || !FECHA_ISO.test(data.registrado_en)
      || !Number.isFinite(Date.parse(data.registrado_en)) || data.codigo_rechazo) {
      throw new ErrorCartera("respuesta_invalida", 503);
    }
  } else if (data.comprobante_ref || data.registrado_en
    || (data.estado === "rechazado" ? typeof data.codigo_rechazo !== "string" || !data.codigo_rechazo : !!data.codigo_rechazo)) {
    throw new ErrorCartera("respuesta_invalida", 503);
  }
  return data;
}

export function crearClienteCartera({ fetchImpl = globalThis.fetch } = {}) {
  if (typeof fetchImpl !== "function") throw new TypeError("meritos.cartera.fetch_invalido");
  return Object.freeze({
    async subirComprobante(clave, documento, { signal } = {}) {
      if (typeof clave !== "string" || !clave || !(documento instanceof Blob)
        || typeof documento.name !== "string" || !documento.name) {
        throw new TypeError("meritos.cartera.comprobante_invalido");
      }
      const cuerpo = new FormData();
      cuerpo.append("clave_idempotencia", clave);
      cuerpo.append("documento", documento, documento.name);
      let respuesta;
      try {
        respuesta = exigirRespuesta(await fetchImpl(RUTA_COMPROBANTES, {
          method: "POST", headers: { Accept: "application/json" }, body: cuerpo,
          mode: "same-origin", credentials: "same-origin", cache: "no-store",
          redirect: "error", referrerPolicy: "no-referrer", signal,
        }));
      } catch (causa) {
        if (causa?.name === "AbortError" || causa instanceof ErrorCartera) throw causa;
        throw new ErrorCartera("resultado_indeterminado", 503, causa);
      }
      let contenido;
      try { contenido = await leerJSON(respuesta); }
      catch (causa) { throw new ErrorCartera("resultado_indeterminado", 503, causa); }
      if (![200, 202, 422].includes(respuesta.status) || (respuesta.status === 422 && !contenido?.data)) {
        throw errorHTTP(respuesta, contenido);
      }
      const recibo = contenido?.data;
      const estado = validarEstadoComprobante(recibo);
      if ((respuesta.status === 200 && estado.estado !== "admitido")
        || (respuesta.status === 202 && !["reservada", "recibido", "analizando"].includes(estado.estado))
        || (respuesta.status === 422 && estado.estado !== "rechazado")) {
        throw new ErrorCartera("resultado_indeterminado", 503);
      }
      return estado;
    },
    async consultarComprobante(clave, { signal } = {}) {
      if (typeof clave !== "string" || !clave) throw new TypeError("meritos.cartera.clave_invalida");
      let respuesta;
      try {
        respuesta = exigirRespuesta(await fetchImpl(`${RUTA_ESTADO_COMPROBANTE}?clave_idempotencia=${encodeURIComponent(clave)}`, {
          method: "GET", headers: { Accept: "application/json" }, mode: "same-origin",
          credentials: "same-origin", cache: "no-store", redirect: "error",
          referrerPolicy: "no-referrer", signal,
        }));
      } catch (causa) {
        if (causa?.name === "AbortError" || causa instanceof ErrorCartera) throw causa;
        throw new ErrorCartera("servicio_no_disponible", 503, causa);
      }
      const contenido = await leerJSON(respuesta);
      if (!respuesta.ok) throw errorHTTP(respuesta, contenido);
      if (respuesta.status !== 200) throw new ErrorCartera("respuesta_invalida", 503);
      return validarEstadoComprobante(contenido?.data);
    },
    async descargarComprobante(ref, { signal } = {}) {
      if (typeof ref !== "string" || !/^ref:[A-Za-z0-9:_-]+$/u.test(ref)) {
        throw new TypeError("meritos.cartera.referencia_invalida");
      }
      let respuesta;
      try {
        respuesta = await fetchImpl(`${RUTA_COMPROBANTES}/${encodeURIComponent(ref)}/original`, {
          method: "GET", headers: { Accept: "application/pdf" }, mode: "same-origin",
          credentials: "same-origin", cache: "no-store", redirect: "error",
          referrerPolicy: "no-referrer", signal,
        });
      } catch (causa) {
        if (causa?.name === "AbortError") throw causa;
        throw new ErrorCartera("servicio_no_disponible", 503, causa);
      }
      if (!respuesta?.ok) {
        let contenido;
        try { contenido = await respuesta?.json?.(); } catch { /* El mensaje visible sigue siendo genérico. */ }
        throw respuesta ? errorHTTP(respuesta, contenido) : new ErrorCartera("servicio_no_disponible", 503);
      }
      const tipo = respuesta.headers?.get?.("Content-Type")?.split(";")[0]?.trim().toLowerCase();
      if (tipo !== "application/pdf" || typeof respuesta.blob !== "function") {
        throw new ErrorCartera("respuesta_invalida", 503);
      }
      const archivo = await respuesta.blob();
      if (!archivo.size) throw new ErrorCartera("respuesta_invalida", 503);
      return archivo;
    },
    async declarar(orden, { signal } = {}) {
      if (!orden || typeof orden !== "object") throw new TypeError("meritos.cartera.orden_invalida");
      let respuesta;
      try {
        respuesta = exigirRespuesta(await fetchImpl(RUTA_DECLARACIONES, {
          method: "POST", headers: { Accept: "application/json", "Content-Type": "application/json" },
          body: JSON.stringify(orden), mode: "same-origin", credentials: "omit", cache: "no-store",
          redirect: "error", referrerPolicy: "no-referrer", signal,
        }));
      } catch (causa) {
        if (causa?.name === "AbortError" || causa instanceof ErrorCartera) throw causa;
        throw new ErrorCartera("resultado_indeterminado", 503, causa);
      }
      let contenido;
      try { contenido = await leerJSON(respuesta); }
      catch (causa) { throw new ErrorCartera("resultado_indeterminado", 503, causa); }
      if (!respuesta.ok) throw errorHTTP(respuesta, contenido);
      const recibo = contenido?.data;
      if (respuesta.status !== 200 || !recibo?.referencia || !recibo.auditoria_ref
        || recibo.accion !== "meritos.hecho.declarar" || recibo.clave_idempotencia !== orden.clave_idempotencia
        || recibo.registro?.hecho?.estado !== "declarado" || recibo.registro.hecho.version !== 1
        || recibo.registro.hecho.denominacion !== orden.denominacion) {
        throw new ErrorCartera("resultado_indeterminado", 503);
      }
      return recibo;
    },
    async consultar({ filtro = "todos", idioma = "es", limite = 25, cursor = "", signal } = {}) {
      if (!["es", "en"].includes(idioma)) throw new TypeError("meritos.cartera.idioma_invalido");
      const parametros = new URLSearchParams({ filtro, limite: String(limite), lang: idioma });
      if (cursor) parametros.set("cursor", cursor);
      let respuesta;
      try {
        respuesta = exigirRespuesta(await fetchImpl(`${RUTA_CARTERA}?${parametros}`, {
          method: "GET", headers: { Accept: "application/json" },
          credentials: "omit", cache: "no-store", redirect: "error", signal,
        }));
      } catch (causa) {
        if (causa?.name === "AbortError" || causa instanceof ErrorCartera) throw causa;
        throw new ErrorCartera("servicio_no_disponible", 503, causa);
      }
      const contenido = await leerJSON(respuesta);
      if (!respuesta.ok) throw errorHTTP(respuesta, contenido);
      const pagina = contenido?.data;
      const siguiente = pagina?.cursor_siguiente ?? "";
      if (!pagina || !Array.isArray(pagina.hechos) || !Array.isArray(pagina.motivos_retirada)
        || pagina.motivos_retirada.length < 1 || pagina.motivos_retirada.length > 128
        || pagina.motivos_retirada.some((m) => typeof m?.etiqueta !== "string" || !m.etiqueta.trim()
          || typeof m?.motivo?.entrada_clave !== "string")
        || !Number.isSafeInteger(pagina.total)
        || pagina.total < 0 || pagina.filtro !== filtro || typeof siguiente !== "string"
        || typeof pagina.corte_en !== "string" || !pagina.recibo_acceso?.auditoria_ref) {
        throw new ErrorCartera("respuesta_invalida", 503);
      }
      return { ...pagina, cursor_siguiente: siguiente };
    },
    async retirar(orden, { signal } = {}) {
      if (!orden || typeof orden !== "object") throw new TypeError("meritos.cartera.orden_invalida");
      let respuesta;
      try {
        respuesta = exigirRespuesta(await fetchImpl(RUTA_RETIRADAS, {
          method: "POST", headers: { Accept: "application/json", "Content-Type": "application/json" },
          body: JSON.stringify(orden), credentials: "omit", cache: "no-store", redirect: "error", signal,
        }));
      } catch (causa) {
        if (causa?.name === "AbortError" || causa instanceof ErrorCartera) throw causa;
        throw new ErrorCartera("resultado_indeterminado", 503, causa);
      }
      let contenido;
      try { contenido = await leerJSON(respuesta); }
      catch (causa) { throw new ErrorCartera("resultado_indeterminado", 503, causa); }
      if (!respuesta.ok) throw errorHTTP(respuesta, contenido);
      const confirmacion = contenido?.data;
      if (![200, 201].includes(respuesta.status) || !confirmacion?.recibo?.referencia
        || typeof confirmacion.repetida !== "boolean" || !confirmacion.auditoria_acceso_ref
        || (respuesta.status === 200) !== confirmacion.repetida) {
        throw new ErrorCartera("resultado_indeterminado", 503);
      }
      return confirmacion;
    },
  });
}
