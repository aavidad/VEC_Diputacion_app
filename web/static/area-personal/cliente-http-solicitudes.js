/**
 * Adaptador HTTP de la solicitud de participación de la persona (Selección).
 *
 * Traduce el contrato «mis solicitudes» (revisión 2) a llamadas de mismo
 * origen: el navegador presenta el certificado de la persona (mTLS) y el
 * servidor deriva de él la identidad; ningún cuerpo lleva identidad. No guarda
 * nada en el navegador: la clave de idempotencia vive en memoria mientras dura
 * el intento. Los errores salen como códigos cerrados que la vista traduce.
 */

const BASE = "/api/vec/seleccion/mis-solicitudes";
export const RUTAS_SOLICITUDES_PERSONA = Object.freeze({
  lista: BASE,
  convocatorias: `${BASE}/convocatorias`,
  convocatoria: `${BASE}/convocatoria`,
  borrador: `${BASE}/borrador`,
  presentacion: `${BASE}/presentacion`,
});

const MAXIMO_RESPUESTA = 256 * 1024;
const MAXIMO_CUERPO = 64 * 1024;
const PATRON_CONVOCATORIA = /^[A-Za-z0-9][A-Za-z0-9:._-]{0,159}$/u;
const PATRON_REFERENCIA = PATRON_CONVOCATORIA;
const PATRON_DECIMAL = /^-?\d{1,9}(?:\.\d{1,6})?$/u;
// Códigos del contrato que la vista sabe explicar; cualquier otro se trata
// como respuesta no admitida, sin mostrar texto del servidor.
export const CODIGOS_ERROR_SOLICITUD = Object.freeze(new Set([
  "sin_borrador", "version_obsoleta", "clave_reutilizada", "fuera_de_plazo", "ya_presentada",
  "convocatoria_no_disponible", "convocatoria_actualizada", "requisito_no_cumplido",
  "declaracion_requerida", "datos_no_validos", "peticion_no_permitida", "recurso_no_encontrado",
  "metodo_no_permitido", "autenticacion_requerida", "acceso_denegado", "servicio_no_disponible",
]));
const CODIGOS_ESTADO = Object.freeze({
  400: "datos_no_validos", 401: "autenticacion_requerida", 403: "acceso_denegado", 404: "no_disponible",
  405: "no_disponible", 409: "conflicto", 422: "datos_no_validos", 429: "servicio_ocupado", 503: "servicio_no_disponible",
});

export class ErrorSolicitudPersona extends Error {
  constructor(codigo, estado = 0) {
    super(codigo);
    this.name = "ErrorSolicitudPersona";
    this.codigo = codigo;
    this.estado = estado;
  }
}

const incompatible = () => new ErrorSolicitudPersona("respuesta_incompatible");
function objeto(valor) {
  if (!valor || typeof valor !== "object" || Array.isArray(valor)) throw incompatible();
  return valor;
}
const cadena = (valor, maximo = 400) => typeof valor === "string" ? valor.trim().slice(0, maximo) : "";
const instante = (valor) => typeof valor === "string" && !Number.isNaN(Date.parse(valor)) ? valor : null;
const version = (valor) => Number.isSafeInteger(valor) && valor >= 0;
/** Puntuación del servidor: cadena decimal o null si no viene. */
const decimal = (valor) => typeof valor === "string" && PATRON_DECIMAL.test(valor) ? valor : null;

async function leerJSON(respuesta) {
  const tipo = respuesta?.headers?.get?.("Content-Type") || "";
  if (!/^application\/json(?:\s*;\s*charset=utf-8)?$/iu.test(tipo)) return null;
  const texto = await respuesta.text();
  if (new TextEncoder().encode(texto).byteLength > MAXIMO_RESPUESTA) throw incompatible();
  try { return JSON.parse(texto); } catch { return null; }
}

function codigoDeError(cuerpo) {
  const codigo = cuerpo?.error?.codigo ?? cuerpo?.codigo;
  return typeof codigo === "string" && CODIGOS_ERROR_SOLICITUD.has(codigo) ? codigo : "";
}

function nuevaClaveIdempotencia() {
  if (typeof globalThis.crypto?.randomUUID !== "function") throw new ErrorSolicitudPersona("idempotencia_no_disponible");
  return `sol-${globalThis.crypto.randomUUID()}`;
}

function validarResumen(item) {
  objeto(item);
  if (!PATRON_REFERENCIA.test(item.solicitud_ref || "") || typeof item.convocatoria_ref !== "string"
    || typeof item.estado !== "string" || !version(item.version)) throw incompatible();
  return Object.freeze({
    solicitud_ref: item.solicitud_ref,
    convocatoria_ref: item.convocatoria_ref,
    convocatoria_titulo: cadena(item.convocatoria_titulo, 300),
    estado: cadena(item.estado, 60),
    version: item.version,
    presentada_en: instante(item.presentada_en),
    numero_justificante: cadena(item.numero_justificante, 40) || null,
    puntuacion_autobaremo: decimal(item.puntuacion_autobaremo),
  });
}

function validarConvocatoriaResumen(item) {
  objeto(item);
  if (!PATRON_CONVOCATORIA.test(item.convocatoria_ref || "") || !cadena(item.titulo)) throw incompatible();
  return Object.freeze({
    convocatoria_ref: item.convocatoria_ref, titulo: cadena(item.titulo, 300),
    abre_en: instante(item.abre_en), cierra_en: instante(item.cierra_en), abierta: item.abierta === true,
  });
}

function validarBorrador(dato) {
  objeto(dato);
  if (!PATRON_REFERENCIA.test(dato.solicitud_ref || "") || !version(dato.version) || dato.version < 1) throw incompatible();
  return Object.freeze(structuredClone(dato));
}

function validarPresentacion(dato, solicitudRef) {
  objeto(dato);
  if (dato.solicitud_ref !== solicitudRef || !cadena(dato.numero_justificante) || !instante(dato.presentada_en)
    || !cadena(dato.recibo_ref)) throw incompatible();
  const servicios = dato.servicios && typeof dato.servicios === "object" && !Array.isArray(dato.servicios)
    ? Object.fromEntries(Object.entries(dato.servicios).filter(([clave, valor]) => /^[a-z_]{2,40}$/u.test(clave) && typeof valor === "string").slice(0, 10))
    : {};
  return Object.freeze({
    solicitud_ref: dato.solicitud_ref,
    estado: cadena(dato.estado, 60),
    numero_justificante: cadena(dato.numero_justificante, 40),
    presentada_en: dato.presentada_en,
    recibo_ref: cadena(dato.recibo_ref, 200),
    puntuacion_autobaremo: decimal(dato.puntuacion_autobaremo),
    repetida: dato.repetida === true,
    servicios: Object.freeze(servicios),
  });
}

/**
 * @param {{ fetchImpl?: typeof fetch }} opciones
 */
export function crearClienteSolicitudesPersona({ fetchImpl = globalThis.fetch } = {}) {
  async function pedir(ruta, { metodo = "GET", cuerpo, clave = "", estados = [200] } = {}) {
    if (typeof fetchImpl !== "function") throw new ErrorSolicitudPersona("servicio_no_disponible");
    const cabeceras = { Accept: "application/json" };
    let serializado;
    if (cuerpo !== undefined) {
      serializado = JSON.stringify(cuerpo);
      if (new TextEncoder().encode(serializado).byteLength > MAXIMO_CUERPO) throw new ErrorSolicitudPersona("solicitud_excesiva");
      cabeceras["Content-Type"] = "application/json";
    }
    if (clave) cabeceras["Idempotency-Key"] = clave;
    let respuesta;
    try {
      respuesta = await fetchImpl(ruta, {
        method: metodo, credentials: "same-origin", cache: "no-store", redirect: "error",
        referrerPolicy: "no-referrer", headers: cabeceras, ...(serializado ? { body: serializado } : {}),
      });
    } catch {
      throw new ErrorSolicitudPersona("servicio_no_disponible");
    }
    const estado = respuesta?.status;
    const datos = await leerJSON(respuesta).catch(() => null);
    if (!estados.includes(estado)) {
      throw new ErrorSolicitudPersona(codigoDeError(datos) || CODIGOS_ESTADO[estado] || "respuesta_incompatible", estado);
    }
    if (datos === null) throw new ErrorSolicitudPersona("respuesta_incompatible", estado);
    return objeto(objeto(datos).data);
  }
  const conConvocatoria = (ruta, ref) => {
    if (!PATRON_CONVOCATORIA.test(ref || "")) throw new ErrorSolicitudPersona("convocatoria_no_disponible");
    return `${ruta}?convocatoria_ref=${encodeURIComponent(ref)}`;
  };

  return Object.freeze({
    nuevaClaveIdempotencia,

    /** Convocatorias con plazo publicadas para solicitar. */
    async convocatorias() {
      const lista = (await pedir(RUTAS_SOLICITUDES_PERSONA.convocatorias)).convocatorias;
      if (!Array.isArray(lista) || lista.length > 500) throw incompatible();
      return Object.freeze(lista.map(validarConvocatoriaResumen));
    },

    /** Reglas publicadas de una convocatoria: plazo, turnos, requisitos y baremo. */
    async convocatoria(convocatoriaRef) {
      const dato = await pedir(conConvocatoria(RUTAS_SOLICITUDES_PERSONA.convocatoria, convocatoriaRef));
      if (dato.convocatoria_ref !== convocatoriaRef || !cadena(dato.titulo) || !Array.isArray(dato.requisitos)) throw incompatible();
      return Object.freeze(structuredClone(dato));
    },

    async listar() {
      const lista = (await pedir(RUTAS_SOLICITUDES_PERSONA.lista)).solicitudes;
      if (!Array.isArray(lista) || lista.length > 500) throw incompatible();
      return Object.freeze(lista.map(validarResumen));
    },

    /** Última versión de la solicitud (borrador o presentada) o null si no hay. */
    async obtenerBorrador(convocatoriaRef) {
      try {
        return validarBorrador(await pedir(conConvocatoria(RUTAS_SOLICITUDES_PERSONA.borrador, convocatoriaRef)));
      } catch (error) {
        if (error?.codigo === "sin_borrador") return null;
        throw error;
      }
    },

    async guardarBorrador(cuerpo, clave) {
      if (!clave) throw new ErrorSolicitudPersona("idempotencia_no_disponible");
      const dato = await pedir(RUTAS_SOLICITUDES_PERSONA.borrador, { metodo: "PUT", cuerpo, clave, estados: [200, 201] });
      if (!PATRON_REFERENCIA.test(dato.solicitud_ref || "") || !version(dato.version) || dato.version <= cuerpo.version_esperada) throw incompatible();
      return Object.freeze({
        solicitud_ref: dato.solicitud_ref, version: dato.version, estado: cadena(dato.estado, 60),
        puntuacion_autobaremo: decimal(dato.puntuacion_autobaremo), repetida: dato.repetida === true,
      });
    },

    async presentar({ solicitudRef, versionEsperada }, clave) {
      if (!clave) throw new ErrorSolicitudPersona("idempotencia_no_disponible");
      const dato = await pedir(RUTAS_SOLICITUDES_PERSONA.presentacion, {
        metodo: "POST", clave, estados: [200, 201],
        cuerpo: { solicitud_ref: solicitudRef, version_esperada: versionEsperada, declaracion_responsable: true },
      });
      return validarPresentacion(dato, solicitudRef);
    },
  });
}
