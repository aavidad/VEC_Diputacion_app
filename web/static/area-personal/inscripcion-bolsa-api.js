/** Cliente de inscripción de la persona identificada por la sesión externa. */
const BASE = "/api/vec/bolsa";
const REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9:._-]{0,199}$/u;
const CLAVE = /^[A-Za-z0-9][A-Za-z0-9:._-]{0,127}$/u;
const CODIGOS_ERROR = new Set(["plazo_cerrado", "catalogo_cambiado", "requisito_invalido",
  "declaracion_invalida", "solicitud_existente", "clave_en_conflicto"]);
const INSTANCIAS = Object.freeze({
  abiertas: "vec.bolsa.inscripciones.abiertas.v1",
  bolsa: "vec.bolsa.inscripcion.bolsa.v1",
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
  return objeto(valor) && referencia(valor.bolsa_ref) && cadena(valor.categoria, 200)
    && fecha(valor.plazo_inicio) && fecha(valor.plazo_fin)
    && cadena(valor.requisitos_resumen, 2000) && Number.isSafeInteger(valor.catalogo_version)
    && valor.catalogo_version > 0
    && (valor.estado_solicitud_propia === undefined || valor.estado_solicitud_propia === null
      || ESTADOS.has(valor.estado_solicitud_propia))
    && (valor.solicitud_ref === undefined || valor.solicitud_ref === null || referencia(valor.solicitud_ref))
    && Boolean(valor.estado_solicitud_propia) === Boolean(valor.solicitud_ref);
}

function solicitud(valor, { categoria = false } = {}) {
  return objeto(valor) && referencia(valor.solicitud_ref) && referencia(valor.bolsa_ref)
    && (!categoria || cadena(valor.categoria, 200))
    && ESTADOS.has(valor.estado) && Number.isSafeInteger(valor.version) && valor.version > 0
    && fecha(valor.registrada_en) && referencia(valor.recibo_ref);
}

function validar(tipo, entrada) {
  const datos = entrada?.data;
  if (!objeto(datos) || datos.esquema !== INSTANCIAS[tipo]) throw new TypeError("Respuesta de inscripción inválida");
  switch (tipo) {
    case "abiertas":
      if (!Array.isArray(datos.bolsas) || datos.bolsas.length > 100 || datos.bolsas.some((b) => !bolsaResumen(b))
        || !Number.isSafeInteger(datos.total) || datos.total < datos.bolsas.length || !cursor(datos.cursor_siguiente))
        throw new TypeError("Relación de bolsas inválida");
      break;
    case "bolsa":
      if (!objeto(datos.bolsa) || !referencia(datos.bolsa.bolsa_ref)
        || !cadena(datos.bolsa.categoria, 200) || !fecha(datos.bolsa.plazo_inicio)
        || !fecha(datos.bolsa.plazo_fin) || !Number.isSafeInteger(datos.bolsa.catalogo_version)
        || datos.bolsa.catalogo_version < 1 || !Array.isArray(datos.bolsa.requisitos)
        || datos.bolsa.requisitos.length > 32 || datos.bolsa.requisitos.some((r) =>
          !objeto(r) || !cadena(r.codigo, 100) || !cadena(r.descripcion, 2000)
          || typeof r.obligatorio !== "boolean")) throw new TypeError("Ficha de bolsa inválida");
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

export function crearClienteInscripcionBolsa({ fetchImpl = globalThis.fetch } = {}) {
  return Object.freeze({
    async abiertas(opciones = {}) {
      const datos = await peticion(`${BASE}/inscripciones/bolsas-abiertas?${parametrosPagina(opciones)}`,
        { fetchImpl, signal: opciones.signal });
      return validar("abiertas", datos);
    },
    async bolsa(bolsaRef, { signal } = {}) {
      const datos = await peticion(`${BASE}/inscripciones/bolsas-abiertas/${segmento(bolsaRef)}`, { fetchImpl, signal });
      const validada = validar("bolsa", datos);
      if (validada.bolsa.bolsa_ref !== bolsaRef) throw new TypeError("Ficha de otra bolsa");
      return validada;
    },
    async propias(opciones = {}) {
      const datos = await peticion(`${BASE}/mi-bolsa/inscripciones?${parametrosPagina(opciones)}`,
        { fetchImpl, signal: opciones.signal });
      return validar("propias", datos);
    },
    async detallePropio(solicitudRef, { signal } = {}) {
      const datos = await peticion(`${BASE}/mi-bolsa/inscripciones/${segmento(solicitudRef)}`, { fetchImpl, signal });
      const validado = validar("detallePropio", datos);
      if (validado.solicitud.solicitud_ref !== solicitudRef) throw new TypeError("Solicitud propia distinta");
      return validado;
    },
    async inscribir({ bolsaRef, claveIdempotencia, catalogoVersion, declaraciones = [], signal } = {}) {
      if (!referencia(bolsaRef) || !CLAVE.test(claveIdempotencia ?? "")
        || !Number.isSafeInteger(catalogoVersion) || catalogoVersion < 1
        || !Array.isArray(declaraciones) || declaraciones.length > 32
        || new Set(declaraciones.map((d) => d?.requisito_codigo)).size !== declaraciones.length
        || declaraciones.some((d) => !objeto(d) || !cadena(d.requisito_codigo, 100)
          || (d.evidencia_ref !== undefined && !referencia(d.evidencia_ref)))) throw new TypeError("Solicitud inválida");
      const datos = await peticion(`${BASE}/mi-bolsa/inscripciones`, {
        metodo: "POST", fetchImpl, signal,
        cuerpo: { bolsa_ref: bolsaRef, clave_idempotencia: claveIdempotencia,
          catalogo_version: catalogoVersion, declaraciones },
      });
      const validado = validar("recibo", datos);
      if (validado.bolsa_ref !== bolsaRef) throw new TypeError("Recibo de otra bolsa");
      return validado;
    },
  });
}
