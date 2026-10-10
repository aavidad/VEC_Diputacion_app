/** Cliente de inscripción de la persona identificada por la sesión externa. */
import { IDIOMA_ACTUAL, INDICE_IDIOMAS } from "../comun/idioma.js";

const BASE = "/api/vec/bolsa";
const REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9:._-]{0,199}$/u;
const CLAVE = /^[A-Za-z0-9][A-Za-z0-9:._-]{0,127}$/u;
const CODIGOS_ERROR = new Set(["plazo_cerrado", "catalogo_cambiado", "requisito_invalido",
  "declaracion_invalida", "solicitud_existente", "clave_en_conflicto"]);
const ESTADOS_REQUISITO = new Set(["cumple", "no_cumple", "pendiente"]);
const INSTANCIAS = Object.freeze({
  abiertas: "vec.bolsa.inscripciones.convocatorias_abiertas.v1",
  convocatoria: "vec.bolsa.inscripcion.convocatoria.v1",
  propias: "vec.bolsa.inscripciones.propias.v1",
  recibo: "vec.bolsa.inscripcion.recibo.v1",
  detallePropio: "vec.bolsa.inscripcion.propias.detalle.v1",
});
const ESTADOS = new Set(["pendiente", "admitida_a_convocatoria", "incorporada", "rechazada"]);
const LIMITE_LISTA = 256 * 1024;
const LIMITE_DETALLE_CONVOCATORIA = 1024 * 1024;
const LIMITE_RECIBO = 64 * 1024;
const LIMITE_ERROR = 8 * 1024;
const PLAZO_PETICION_MS = 15_000;
const PLAZO_CANCELACION_MS = 100;
const CODIFICADOR = new TextEncoder();

function objeto(valor) {
  return valor !== null && typeof valor === "object" && !Array.isArray(valor);
}

function cadena(valor, maximo = 500) {
  return typeof valor === "string" && valor.length > 0 && valor.length <= maximo;
}

function titulo(valor) {
  return typeof valor === "string" && valor.length > 0 && Array.from(valor).length <= 180;
}

function etiquetaCategoria(valor) {
  return typeof valor === "string" && valor.length > 0 && CODIFICADOR.encode(valor).byteLength <= 2048;
}

function referencia(valor) {
  return typeof valor === "string" && REFERENCIA.test(valor);
}

function fecha(valor) {
  return cadena(valor, 40) && !Number.isNaN(Date.parse(valor));
}

function cursor(valor) {
  return valor === null || (cadena(valor, 1024) && !/[\u0000-\u001f\u007f]/u.test(valor));
}

function bolsaResumen(valor) {
  return objeto(valor) && referencia(valor.convocatoria_ref) && titulo(valor.titulo)
    && Number.isSafeInteger(valor.numero_categorias) && valor.numero_categorias >= 1
    && valor.numero_categorias <= 128
    && fecha(valor.plazo_inicio) && fecha(valor.plazo_fin)
    && Date.parse(valor.plazo_inicio) <= Date.parse(valor.plazo_fin)
    && cadena(valor.requisitos_resumen, 2000) && Number.isSafeInteger(valor.catalogo_version)
    && valor.catalogo_version > 0
    && typeof valor.puede_iniciar === "boolean"
    && (valor.puede_iniciar || cadena(valor.impedimento_etiqueta, 500))
    && (valor.estado_solicitud_propia === undefined || valor.estado_solicitud_propia === null
      || ESTADOS.has(valor.estado_solicitud_propia))
    && (valor.solicitud_ref === undefined || valor.solicitud_ref === null || referencia(valor.solicitud_ref))
    && Boolean(valor.estado_solicitud_propia) === Boolean(valor.solicitud_ref);
}

function bolsaListado(valor) {
  return bolsaResumen(valor) && valor.categorias === undefined && valor.categorias_resumen === undefined;
}

function bolsaDetalle(valor) {
  return bolsaResumen(valor) && valor.categorias_resumen === undefined
    && Array.isArray(valor.categorias) && valor.categorias.length === valor.numero_categorias
    && valor.categorias.every((c) => objeto(c) && referencia(c.categoria_ref) && etiquetaCategoria(c.categoria))
    && new Set(valor.categorias.map((c) => c.categoria_ref)).size === valor.categorias.length;
}

function solicitud(valor, { categoria = false } = {}) {
  return objeto(valor) && referencia(valor.solicitud_ref) && referencia(valor.convocatoria_ref)
    && (valor.bolsa_ref === undefined || valor.bolsa_ref === null || referencia(valor.bolsa_ref))
    && (!categoria || etiquetaCategoria(valor.categoria))
    && ESTADOS.has(valor.estado) && Number.isSafeInteger(valor.version) && valor.version > 0
    && fecha(valor.registrada_en) && referencia(valor.recibo_ref);
}

function validar(tipo, entrada) {
  const datos = entrada?.data;
  if (!objeto(datos) || datos.esquema !== INSTANCIAS[tipo]) throw new TypeError("Respuesta de inscripción inválida");
  switch (tipo) {
    case "abiertas":
      if (!Array.isArray(datos.convocatorias) || datos.convocatorias.length > 100
        || datos.convocatorias.some((b) => !bolsaListado(b))
        || !Number.isSafeInteger(datos.total) || datos.total < datos.convocatorias.length || !cursor(datos.cursor_siguiente))
        throw new TypeError("Relación de bolsas inválida");
      break;
    case "convocatoria":
      if (!bolsaDetalle(datos.convocatoria)
        || !Array.isArray(datos.convocatoria.requisitos)
        || datos.convocatoria.requisitos.length > 32 || datos.convocatoria.requisitos.some((r) =>
          !objeto(r) || !cadena(r.codigo, 100) || !cadena(r.descripcion, 2000)
          || typeof r.obligatorio !== "boolean" || !ESTADOS_REQUISITO.has(r.estado)
          || !cadena(r.motivo_codigo, 100) || !cadena(r.motivo_etiqueta, 500)
          || (r.procedencia_ref !== undefined && r.procedencia_ref !== null && !referencia(r.procedencia_ref))
          || (r.hito_cumplimiento !== undefined && r.hito_cumplimiento !== null && !cadena(r.hito_cumplimiento, 100))
          || (r.hito_etiqueta !== undefined && r.hito_etiqueta !== null && !cadena(r.hito_etiqueta, 200))
          || Boolean(r.hito_cumplimiento) !== Boolean(r.hito_etiqueta)
          || (r.hito_fecha !== undefined && r.hito_fecha !== null && !fecha(r.hito_fecha)))
        || new Set(datos.convocatoria.requisitos.map((r) => r.codigo)).size !== datos.convocatoria.requisitos.length)
        throw new TypeError("Ficha de convocatoria inválida");
      break;
    case "propias":
      if (!Array.isArray(datos.solicitudes) || datos.solicitudes.length > 100
        || datos.solicitudes.some((s) => !solicitud(s, { categoria: true })) || !cursor(datos.cursor_siguiente))
        throw new TypeError("Relación de solicitudes inválida");
      break;
    case "recibo":
      if (!solicitud(datos) || typeof datos.repetida !== "boolean") throw new TypeError("Recibo inválido");
      break;
    case "detallePropio":
      if (!solicitud(datos.solicitud, { categoria: true })
        || (datos.solicitud.decidida_en !== undefined && datos.solicitud.decidida_en !== null && !fecha(datos.solicitud.decidida_en))
        || (datos.solicitud.motivo_codigo !== undefined && datos.solicitud.motivo_codigo !== null
          && !cadena(datos.solicitud.motivo_codigo, 100))
        || (datos.solicitud.estado === "rechazada" && !cadena(datos.solicitud.motivo_etiqueta, 500))
        || (datos.solicitud.decision_ref !== undefined && datos.solicitud.decision_ref !== null
          && !referencia(datos.solicitud.decision_ref))) throw new TypeError("Solicitud propia inválida");
      break;
    default: throw new TypeError("Operación de inscripción inválida");
  }
  return datos;
}

function segmento(valor) {
  if (!referencia(valor)) throw new TypeError("Referencia inválida");
  return encodeURIComponent(valor).replace(/%3A/giu, ":");
}

async function cancelarAcotado(fuente) {
  if (typeof fuente?.cancel !== "function") return null;
  let temporizador;
  try {
    return await Promise.race([
      Promise.resolve().then(() => fuente.cancel()).then(() => null, () => "cancelacion_fallida"),
      new Promise((resolver) => { temporizador = setTimeout(() => resolver("cancelacion_sin_acuse"), PLAZO_CANCELACION_MS); }),
    ]);
  } finally { clearTimeout(temporizador); }
}

async function leerJSONAcotado(respuesta, limite, signal, abortada) {
  const tipo = respuesta.headers?.get?.("Content-Type") ?? "";
  if (tipo.split(";")[0].trim().toLowerCase() !== "application/json") {
    const error = new TypeError("Respuesta de inscripción no JSON");
    error.causa_limpieza = await cancelarAcotado(respuesta.body);
    throw error;
  }
  const declarada = respuesta.headers?.get?.("Content-Length");
  if (declarada !== null && declarada !== undefined
    && (!/^(?:0|[1-9][0-9]*)$/u.test(declarada) || Number(declarada) > limite)) {
    const error = new TypeError("Respuesta de inscripción demasiado grande");
    error.causa_limpieza = await cancelarAcotado(respuesta.body);
    throw error;
  }
  const lector = respuesta.body?.getReader?.();
  if (!lector) throw new TypeError("Respuesta de inscripción sin flujo");
  const decodificador = new TextDecoder("utf-8", { fatal: true });
  let bytes = 0;
  let texto = "";
  try {
    for (;;) {
      if (signal.aborted) throw signal.reason;
      const { done, value } = await Promise.race([lector.read(), abortada]);
      if (done) break;
      bytes += value.byteLength;
      if (bytes > limite) throw new TypeError("Respuesta de inscripción demasiado grande");
      texto += decodificador.decode(value, { stream: true });
    }
    texto += decodificador.decode();
    return JSON.parse(texto);
  } catch (error) {
    error.causa_limpieza = await cancelarAcotado(lector);
    throw error;
  } finally { lector.releaseLock(); }
}

async function peticion(ruta, { metodo = "GET", cuerpo, fetchImpl = globalThis.fetch,
  signal, limite = LIMITE_LISTA } = {}) {
  const controlador = new AbortController();
  const abortarExterior = () => controlador.abort(signal.reason instanceof Error
    ? signal.reason : new DOMException("Petición cancelada", "AbortError"));
  if (signal?.aborted) abortarExterior();
  else signal?.addEventListener?.("abort", abortarExterior, { once: true });
  let rechazarAbortada;
  const abortada = new Promise((_resolver, rechazar) => { rechazarAbortada = rechazar; });
  const alAbortar = () => rechazarAbortada(controlador.signal.reason ?? new DOMException("Petición cancelada", "AbortError"));
  controlador.signal.addEventListener("abort", alAbortar, { once: true });
  if (controlador.signal.aborted) alAbortar();
  const temporizador = setTimeout(() => controlador.abort(new DOMException("Tiempo agotado", "TimeoutError")), PLAZO_PETICION_MS);
  try {
    const respuesta = await Promise.race([
      Promise.resolve().then(() => {
        if (controlador.signal.aborted) throw controlador.signal.reason;
        return fetchImpl(ruta, {
          method: metodo, credentials: "same-origin", mode: "same-origin", cache: "no-store",
          redirect: "error", referrerPolicy: "no-referrer", signal: controlador.signal,
          headers: { Accept: "application/json", ...(cuerpo ? { "Content-Type": "application/json" } : {}) },
          ...(cuerpo ? { body: JSON.stringify(cuerpo) } : {}),
        });
      }), abortada,
    ]);
    if (!respuesta?.ok) {
      const error = new Error("Error de inscripción");
      error.status = Number.isInteger(respuesta?.status) ? respuesta.status : 0;
      if ([409, 422].includes(error.status)) {
        try {
          const codigo = (await leerJSONAcotado(respuesta, LIMITE_ERROR, controlador.signal, abortada))?.error?.codigo;
          if (CODIGOS_ERROR.has(codigo)) error.codigo = codigo;
        } catch (causa) {
          if (signal?.aborted) throw causa;
          error.causa_lectura = causa?.name ?? "Error";
        }
      } else error.causa_limpieza = await cancelarAcotado(respuesta.body);
      throw error;
    }
    return await leerJSONAcotado(respuesta, limite, controlador.signal, abortada);
  } finally {
    clearTimeout(temporizador);
    signal?.removeEventListener?.("abort", abortarExterior);
    controlador.signal.removeEventListener("abort", alAbortar);
  }
}

function parametrosPagina({ limite = 20, cursor: siguiente = "" } = {}) {
  if (!Number.isInteger(limite) || limite < 20 || limite > 100
    || (siguiente && !cursor(siguiente))) throw new TypeError("Página inválida");
  const parametros = new URLSearchParams({ limite: String(limite) });
  if (siguiente) parametros.set("cursor", siguiente);
  return parametros;
}

export function crearClienteInscripcionBolsa({ fetchImpl = globalThis.fetch, idioma = IDIOMA_ACTUAL } = {}) {
  if (!INDICE_IDIOMAS.idiomas.some(({ codigo }) => codigo === idioma)) throw new TypeError("Idioma no disponible");
  const idiomaConsulta = new URLSearchParams({ idioma });
  return Object.freeze({
    async abiertas(opciones = {}) {
      const parametros = parametrosPagina(opciones);
      parametros.set("idioma", idioma);
      const datos = await peticion(`${BASE}/inscripciones/convocatorias-abiertas?${parametros}`,
        { fetchImpl, signal: opciones.signal });
      return validar("abiertas", datos);
    },
    async convocatoria(convocatoriaRef, { signal } = {}) {
      const datos = await peticion(`${BASE}/inscripciones/convocatorias-abiertas/${segmento(convocatoriaRef)}?${idiomaConsulta}`,
        { fetchImpl, signal, limite: LIMITE_DETALLE_CONVOCATORIA });
      const validada = validar("convocatoria", datos);
      if (validada.convocatoria.convocatoria_ref !== convocatoriaRef) throw new TypeError("Ficha de otra convocatoria");
      return validada;
    },
    async propias(opciones = {}) {
      const parametros = parametrosPagina(opciones);
      parametros.set("idioma", idioma);
      const datos = await peticion(`${BASE}/inscripciones/propias?${parametros}`,
        { fetchImpl, signal: opciones.signal });
      return validar("propias", datos);
    },
    async detallePropio(solicitudRef, { signal } = {}) {
      const datos = await peticion(`${BASE}/inscripciones/propias/${segmento(solicitudRef)}?${idiomaConsulta}`, { fetchImpl, signal });
      const validado = validar("detallePropio", datos);
      if (validado.solicitud.solicitud_ref !== solicitudRef) throw new TypeError("Solicitud propia distinta");
      return validado;
    },
    async inscribir({ convocatoriaRef, categoriaRef, claveIdempotencia, catalogoVersion, declaraciones = [], signal } = {}) {
      if (!referencia(convocatoriaRef) || !referencia(categoriaRef) || !CLAVE.test(claveIdempotencia ?? "")
        || !Number.isSafeInteger(catalogoVersion) || catalogoVersion < 1
        || !Array.isArray(declaraciones) || declaraciones.length > 32
        || new Set(declaraciones.map((d) => d?.requisito_codigo)).size !== declaraciones.length
        || declaraciones.some((d) => !objeto(d) || !cadena(d.requisito_codigo, 100)
          || (d.evidencia_ref !== undefined && !referencia(d.evidencia_ref)))) throw new TypeError("Solicitud inválida");
      const datos = await peticion(`${BASE}/inscripciones/propias`, {
        metodo: "POST", fetchImpl, signal, limite: LIMITE_RECIBO,
        cuerpo: { convocatoria_ref: convocatoriaRef, categoria_ref: categoriaRef, clave_idempotencia: claveIdempotencia,
          catalogo_version: catalogoVersion, declaraciones },
      });
      const validado = validar("recibo", datos);
      if (validado.convocatoria_ref !== convocatoriaRef) throw new TypeError("Recibo de otra convocatoria");
      return validado;
    },
  });
}
