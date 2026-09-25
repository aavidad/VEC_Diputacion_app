/**
 * Consulta de la documentación exigida para formalizar y de su estado.
 * Los plazos y la lista los calcula Contratación temporal desde el catálogo de
 * reglas; qué documentos constan anotados lo conserva Documentos. Nada se
 * guarda en el navegador.
 */
import { crearFuenteDocumentosHTTP } from "../documentos/cliente-http.js?v=20260926-integracion-bolsa-ct-v1";

export const RUTA_DOCUMENTACION_FORMALIZACION = "/api/vec/contratacion-temporal/formalizacion/documentacion";
const ESQUEMA = "vec.ct.formalizacion.documentacion.v1";
const MAX_JSON = 256 * 1024;
const LIMITE_MS = 15000;
const MAX_PAGINAS = 10;
const ESTADOS = new Set(["en_curso", "ultimo_dia", "vencido"]);
const CLAVE = /^[a-z][a-z0-9_]{1,63}$/u;
const TIPO = /^[a-z][a-z0-9_.]{2,127}$/u;
const FECHA = /^\d{4}-\d{2}-\d{2}$/u;
const INSTANTE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?Z$/u;
const EXPEDIENTE_CT = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
const AGRUPACION = /^ref:[0-9a-f]{64}$/u;

function fallo(codigo, estado) {
  return Object.assign(new Error(codigo), { codigo, estado });
}

function exigir(condicion) {
  if (!condicion) throw fallo("respuesta_invalida");
}

function texto(valor, maximo = 512) {
  return typeof valor === "string" && valor.length <= maximo;
}

function regla(valor) {
  exigir(valor && typeof valor === "object" && CLAVE.test(valor.clave?.replaceAll(".", "_") ?? "")
    && texto(valor.unidad, 64) && ["reglamento", "ejemplo"].includes(valor.origen) && typeof valor.ejemplo === "boolean"
    && texto(valor.norma) && texto(valor.duda) && texto(valor.referencia) && /^[0-9a-f]{64}$/u.test(valor.huella_catalogo ?? "")
    && (valor.cantidad === undefined || (Number.isSafeInteger(valor.cantidad) && valor.cantidad > 0))
    && (valor.articulo === undefined || texto(valor.articulo)) && (valor.parte_ejemplo === undefined || texto(valor.parte_ejemplo)));
  return Object.freeze({ ...valor });
}

function plazo(valor) {
  exigir(valor && typeof valor === "object" && FECHA.test(valor.ultimo_dia ?? "") && INSTANTE.test(valor.vence_antes_de ?? "")
    && typeof valor.prorrogado === "boolean" && ESTADOS.has(valor.estado));
  return Object.freeze({ ...valor, regla: regla(valor.regla) });
}

/** Valida la respuesta cerrada de la consulta de documentación. */
export function validarDocumentacionFormalizacion(datos) {
  exigir(datos && typeof datos === "object" && datos.esquema === ESQUEMA && INSTANTE.test(datos.aceptada_en ?? "")
    && AGRUPACION.test(datos.expediente_documental_ref ?? "") && !/^ref:0{64}$/u.test(datos.expediente_documental_ref)
    && INSTANTE.test(datos.consultada_en ?? "") && typeof datos.por_modalidad === "boolean"
    && Array.isArray(datos.documentos) && datos.documentos.length > 0 && datos.documentos.length <= 64);
  const vistos = new Set();
  const documentos = datos.documentos.map((documento) => {
    exigir(documento && CLAVE.test(documento.clave ?? "") && TIPO.test(documento.tipo_documental ?? "")
      && typeof documento.registrable === "boolean" && !vistos.has(documento.clave));
    vistos.add(documento.clave);
    return Object.freeze({ clave: documento.clave, tipo_documental: documento.tipo_documental, registrable: documento.registrable });
  });
  return Object.freeze({
    expediente_documental_ref: datos.expediente_documental_ref, aceptada_en: datos.aceptada_en, consultada_en: datos.consultada_en, por_modalidad: datos.por_modalidad,
    documentos: Object.freeze(documentos), regla_documentos: regla(datos.regla_documentos),
    plazo_documentacion: plazo(datos.plazo_documentacion),
    plazo_incorporacion: datos.plazo_incorporacion === undefined ? null : plazo(datos.plazo_incorporacion),
  });
}

async function consultarDocumentacion({ expedienteRef, aceptadaEn, signal, fetchImpl }) {
  if (typeof aceptadaEn !== "string" || !INSTANTE.test(aceptadaEn)
    || typeof expedienteRef !== "string" || !EXPEDIENTE_CT.test(expedienteRef)) throw fallo("referencia_invalida");
  const controlador = new AbortController();
  const abortar = () => controlador.abort();
  signal?.addEventListener("abort", abortar, { once: true });
  const temporizador = setTimeout(abortar, LIMITE_MS);
  try {
    const consulta = `expediente_ref=${encodeURIComponent(expedienteRef)}&aceptada_en=${encodeURIComponent(aceptadaEn)}`;
    const respuesta = await fetchImpl(`${RUTA_DOCUMENTACION_FORMALIZACION}?${consulta}`, {
      method: "GET", signal: controlador.signal, headers: { Accept: "application/json" },
      credentials: "same-origin", mode: "same-origin", redirect: "error", referrerPolicy: "no-referrer", cache: "no-store",
    });
    const bruto = await respuesta.text();
    if (bruto.length > MAX_JSON) throw fallo("respuesta_invalida");
    let cuerpo = null;
    try { cuerpo = JSON.parse(bruto); } catch { cuerpo = null; }
    if (respuesta.status === 401 || respuesta.status === 403) throw fallo("denegado", respuesta.status);
    if (respuesta.status === 503 && cuerpo?.error?.codigo === "reglas_no_configuradas") throw fallo("sin_reglas", 503);
    if (!respuesta.ok) throw fallo("no_disponible", respuesta.status);
    return validarDocumentacionFormalizacion(cuerpo?.data);
  } catch (error) {
    if (signal?.aborted) throw fallo("cancelado");
    if (controlador.signal.aborted) throw fallo("no_disponible");
    throw error;
  } finally {
    clearTimeout(temporizador);
    signal?.removeEventListener("abort", abortar);
  }
}

/**
 * Fuente neutral del panel. `documentos` es la fuente de Documentos para un
 * expediente; la composición puede sustituir ambas en pruebas.
 */
export function crearFuenteDocumentacionFormalizacionHTTP({ fetchImpl = globalThis.fetch, crearDocumentos = crearFuenteDocumentosHTTP } = {}) {
  if (typeof fetchImpl !== "function" || typeof crearDocumentos !== "function") {
    throw new TypeError("dependencias de documentación de formalización no válidas");
  }
  return Object.freeze({
    consultar: ({ expedienteRef, aceptadaEn, signal } = {}) => consultarDocumentacion({ expedienteRef, aceptadaEn, signal, fetchImpl }),
    /**
     * Devuelve, por tipo documental, la anotación más reciente de la
     * agrupación documental que devolvió la consulta (expediente_documental_ref).
     */
    async anotados({ expedienteRef, signal } = {}) {
      const fuente = crearDocumentos({ expedienteRef, fetchImpl });
      const porTipo = new Map();
      let cursor = "";
      for (let pagina = 0; pagina < MAX_PAGINAS; pagina += 1) {
        const datos = await fuente.listar({ signal, cursor });
        exigir(datos && Array.isArray(datos.documentos) && typeof datos.siguiente_cursor === "string");
        for (const documento of datos.documentos) {
          if (!TIPO.test(documento?.tipo ?? "") || documento.custodia !== "externa") continue;
          porTipo.set(documento.tipo, Object.freeze({ numero_vec: String(documento.numero_vec), huella: String(documento.huella) }));
        }
        if (!datos.siguiente_cursor) return porTipo;
        cursor = datos.siguiente_cursor;
      }
      throw fallo("respuesta_invalida");
    },
    async registrar({ expedienteRef, tipo, referencia, huella, claveIdempotencia, signal } = {}) {
      const datos = await crearDocumentos({ expedienteRef, fetchImpl })
        .registrarExterno({ tipo, referencia, huella, claveIdempotencia, signal });
      exigir(datos?.estado === "registrado" && datos.documento?.tipo === tipo && datos.documento.huella === huella
        && /^VEC-\d{4}-\d{1,12}$/u.test(datos.documento.numero_vec ?? ""));
      return Object.freeze({ numero_vec: datos.documento.numero_vec, huella: datos.documento.huella });
    },
  });
}
