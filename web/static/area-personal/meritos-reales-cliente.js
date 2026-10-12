const RUTA_CARTERA = "/api/meritos/cartera";
const RUTA_RETIRADAS = "/api/meritos/cartera/retiradas";

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

export function crearClienteCartera({ fetchImpl = globalThis.fetch } = {}) {
  if (typeof fetchImpl !== "function") throw new TypeError("meritos.cartera.fetch_invalido");
  return Object.freeze({
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
