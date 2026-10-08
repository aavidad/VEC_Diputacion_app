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

function objeto(valor) {
  return valor !== null && typeof valor === "object" && !Array.isArray(valor);
}

function cadena(valor, maximo = 500) {
  return typeof valor === "string" && valor.length > 0 && valor.length <= maximo;
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
  return objeto(valor) && referencia(valor.convocatoria_ref) && cadena(valor.titulo, 200)
    && cadena(valor.categorias_resumen, 500)
    && Array.isArray(valor.categorias) && valor.categorias.length > 0 && valor.categorias.length <= 32
    && valor.categorias.every((c) => objeto(c) && referencia(c.categoria_ref) && cadena(c.categoria, 200))
    && new Set(valor.categorias.map((c) => c.categoria_ref)).size === valor.categorias.length
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

function solicitud(valor, { categoria = false } = {}) {
  return objeto(valor) && referencia(valor.solicitud_ref) && referencia(valor.convocatoria_ref)
    && (valor.bolsa_ref === undefined || valor.bolsa_ref === null || referencia(valor.bolsa_ref))
    && (!categoria || cadena(valor.categoria, 200))
    && ESTADOS.has(valor.estado) && Number.isSafeInteger(valor.version) && valor.version > 0
    && fecha(valor.registrada_en) && referencia(valor.recibo_ref);
}

function validar(tipo, entrada) {
  const datos = entrada?.data;
  if (!objeto(datos) || datos.esquema !== INSTANCIAS[tipo]) throw new TypeError("Respuesta de inscripción inválida");
  switch (tipo) {
    case "abiertas":
      if (!Array.isArray(datos.convocatorias) || datos.convocatorias.length > 100
        || datos.convocatorias.some((b) => !bolsaResumen(b))
        || !Number.isSafeInteger(datos.total) || datos.total < datos.convocatorias.length || !cursor(datos.cursor_siguiente))
        throw new TypeError("Relación de bolsas inválida");
      break;
    case "convocatoria":
      if (!bolsaResumen(datos.convocatoria)
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

async function peticion(ruta, { metodo = "GET", cuerpo, fetchImpl = globalThis.fetch, signal } = {}) {
  const respuesta = await fetchImpl(ruta, {
    method: metodo, credentials: "same-origin", mode: "same-origin", cache: "no-store",
    redirect: "error", referrerPolicy: "no-referrer", signal,
    headers: { Accept: "application/json", ...(cuerpo ? { "Content-Type": "application/json" } : {}) },
    ...(cuerpo ? { body: JSON.stringify(cuerpo) } : {}),
  });
  if (!respuesta?.ok) {
    const error = new Error("Error de inscripción");
    error.status = Number.isInteger(respuesta?.status) ? respuesta.status : 0;
    if ([409, 422].includes(error.status)) {
      try {
        const codigo = (await respuesta.json())?.error?.codigo;
        if (CODIGOS_ERROR.has(codigo)) error.codigo = codigo;
      } catch { /* La respuesta pública puede no traer cuerpo JSON. */ }
    }
    throw error;
  }
  return respuesta.json();
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
      const datos = await peticion(`${BASE}/inscripciones/convocatorias-abiertas/${segmento(convocatoriaRef)}?${idiomaConsulta}`, { fetchImpl, signal });
      const validada = validar("convocatoria", datos);
      if (validada.convocatoria.convocatoria_ref !== convocatoriaRef) throw new TypeError("Ficha de otra convocatoria");
      return validada;
    },
    async propias(opciones = {}) {
      const parametros = parametrosPagina(opciones);
      parametros.set("idioma", idioma);
      const datos = await peticion(`${BASE}/mi-bolsa/inscripciones?${parametros}`,
        { fetchImpl, signal: opciones.signal });
      return validar("propias", datos);
    },
    async detallePropio(solicitudRef, { signal } = {}) {
      const datos = await peticion(`${BASE}/mi-bolsa/inscripciones/${segmento(solicitudRef)}?${idiomaConsulta}`, { fetchImpl, signal });
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
      const datos = await peticion(`${BASE}/mi-bolsa/inscripciones`, {
        metodo: "POST", fetchImpl, signal,
        cuerpo: { convocatoria_ref: convocatoriaRef, categoria_ref: categoriaRef, clave_idempotencia: claveIdempotencia,
          catalogo_version: catalogoVersion, declaraciones },
      });
      const validado = validar("recibo", datos);
      if (validado.convocatoria_ref !== convocatoriaRef) throw new TypeError("Recibo de otra convocatoria");
      return validado;
    },
  });
}
