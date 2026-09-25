/**
 * Ficha propia de la persona empleada («mis datos» de Personal).
 *
 * El servidor deriva persona, perfil y empleado del certificado de la
 * petición y autoriza cada consulta; el cliente no envía referencias ni
 * parámetros. Una sola consulta alimenta los apartados «Relaciones y
 * puestos» y «Servicios» con denominaciones legibles; nada se guarda fuera
 * de la memoria de la vista montada.
 */
import { crearTraductorFichaPropia, formatearDiasFichaPropia } from "./i18n-ficha-propia.js?v=20260925-personal-mis-datos-v1";

export const RUTA_FICHA_PROPIA = "/api/interna/personal/mi-ficha";

const MAXIMO_RESPUESTA_BYTES = 256 * 1024;
const MAXIMO_FILAS = 200;
const PLAZO_POR_DEFECTO_MS = 10_000;
const ESTADOS_RELACION = new Set(["vigente", "suspendida", "finalizada"]);
const ESTADOS_SERVICIO = new Set(["declarado", "comprobado", "reconocido"]);
const FECHA = /^\d{4}-\d{2}-\d{2}$/u;
const INSTANTE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{6}Z$/u;

export class ErrorClienteFichaPropia extends Error {
  constructor(codigo, estado = 0) {
    super(`ficha propia de Personal: ${codigo}`);
    this.name = "ErrorClienteFichaPropia";
    this.codigo = codigo;
    this.estado = estado;
    Object.freeze(this);
  }
}

function error(codigo, estado = 0) { return new ErrorClienteFichaPropia(codigo, estado); }
function registro(valor) { return valor !== null && typeof valor === "object" && !Array.isArray(valor); }
function claves(valor, esperadas) {
  return registro(valor) && Object.keys(valor).length === esperadas.length && esperadas.every((clave) => Object.hasOwn(valor, clave));
}
function fecha(valor, vacia = false) {
  if (vacia && valor === "") return true;
  if (typeof valor !== "string" || !FECHA.test(valor)) return false;
  const d = new Date(`${valor}T12:00:00Z`);
  return Number.isFinite(d.getTime()) && d.toISOString().slice(0, 10) === valor;
}
function texto(valor) { return typeof valor === "string" && valor.length <= 300 && !/[\u0000-\u001f\u007f]/u.test(valor); }

function validarSobre(sobre) {
  const datos = sobre?.data;
  const ficha = datos?.ficha;
  if (!claves(sobre, ["data"]) || !claves(datos, ["ficha", "recibo_ref", "consultada_en"]) ||
      typeof datos.recibo_ref !== "string" || !/^fichapropia:[0-9a-f-]{36}$/u.test(datos.recibo_ref) ||
      typeof datos.consultada_en !== "string" || !INSTANTE.test(datos.consultada_en) || !Number.isFinite(Date.parse(datos.consultada_en)) ||
      !claves(ficha, ["corte", "relaciones", "servicios"]) || !claves(ficha.corte, ["vigente_en", "conocido_en"]) || !fecha(ficha.corte.vigente_en) ||
      !Array.isArray(ficha.relaciones) || !Array.isArray(ficha.servicios) ||
      ficha.relaciones.length > MAXIMO_FILAS || ficha.servicios.length > MAXIMO_FILAS) throw error("sobre_no_valido", 200);
  for (const r of ficha.relaciones) {
    if (!claves(r, ["inicio", "fin", "estado", "regimen", "modalidad", "unidad", "puesto", "situacion"]) ||
        !fecha(r.inicio) || !fecha(r.fin, true) || !ESTADOS_RELACION.has(r.estado) ||
        ![r.regimen, r.modalidad, r.unidad, r.puesto, r.situacion].every(texto)) throw error("sobre_no_valido", 200);
  }
  for (const s of ficha.servicios) {
    if (!claves(s, ["inicio", "fin", "clase", "dias", "estado"]) || !fecha(s.inicio) || !fecha(s.fin) ||
        !texto(s.clase) || !Number.isSafeInteger(s.dias) || s.dias < 0 || !ESTADOS_SERVICIO.has(s.estado)) throw error("sobre_no_valido", 200);
  }
  return Object.freeze({ ficha, consultadaEn: datos.consultada_en });
}

async function consultar(fetchImpl, plazoMs, externo) {
  const controlador = new AbortController();
  const abortar = () => controlador.abort();
  if (externo?.aborted) throw error("operacion_abortada");
  externo?.addEventListener?.("abort", abortar, { once: true });
  const temporizador = setTimeout(abortar, plazoMs);
  try {
    let respuesta;
    try {
      respuesta = await fetchImpl(RUTA_FICHA_PROPIA, {
        method: "GET", credentials: "same-origin", mode: "same-origin", cache: "no-store",
        redirect: "error", referrerPolicy: "no-referrer", headers: { Accept: "application/json" },
        signal: controlador.signal,
      });
    } catch {
      throw error(externo?.aborted ? "operacion_abortada" : "red_no_disponible");
    }
    const estado = respuesta?.status || 0;
    // Ruta no servida por esta superficie o persona sin permiso o sin empleado:
    // el apartado no tiene fuente para ella.
    if (estado === 404 || estado === 401 || estado === 403) {
      try { await respuesta.body?.cancel?.(); } catch {}
      return Object.freeze({ sinFuente: true, estado });
    }
    if (estado !== 200 || respuesta.ok !== true || respuesta.redirected === true) throw error("estado_no_valido", estado);
    const tipo = respuesta.headers?.get?.("content-type");
    const longitud = respuesta.headers?.get?.("content-length");
    if (tipo !== "application/json; charset=utf-8" ||
        (longitud !== null && longitud !== undefined && (!/^\d+$/u.test(longitud) || Number(longitud) > MAXIMO_RESPUESTA_BYTES))) {
      throw error("respuesta_no_valida", estado);
    }
    const cuerpo = await respuesta.text();
    if (typeof cuerpo !== "string" || cuerpo.length > MAXIMO_RESPUESTA_BYTES) throw error("respuesta_no_valida", estado);
    let sobre;
    try { sobre = JSON.parse(cuerpo); } catch { throw error("json_no_valido", estado); }
    return validarSobre(sobre);
  } finally {
    clearTimeout(temporizador);
    externo?.removeEventListener?.("abort", abortar);
  }
}

function presentarRelaciones(ficha, t) {
  return ficha.relaciones.map((r) => ({
    desde: r.inicio,
    hasta: r.fin,
    regimen: [r.regimen, r.modalidad].filter(Boolean).join(" · "),
    puesto: r.puesto,
    unidad: r.unidad,
    estado: r.situacion || t(`estado_relacion_${r.estado}`),
  }));
}

function presentarServicios(ficha, t) {
  return ficha.servicios.map((s) => ({
    desde: s.inicio,
    hasta: s.fin,
    procedencia: s.clase,
    reconocimiento: formatearDiasFichaPropia(s.dias, t),
    estado: t(`estado_servicio_${s.estado}`),
  }));
}

/**
 * Crea las fuentes de los apartados de la ficha para una vista montada.
 * `preparar()` hace la consulta una vez: con una ficha válida devuelve los
 * apartados, que se pintan desde esa misma respuesta; si la superficie no la
 * sirve, la persona no tiene acceso o la consulta falla, devuelve `{}` y los
 * apartados no se ofrecen.
 */
export function crearFuentesFichaPropia({ fetchImpl = globalThis.fetch, traducir = crearTraductorFichaPropia(), plazoMs = PLAZO_POR_DEFECTO_MS } = {}) {
  if (typeof fetchImpl !== "function" || typeof traducir !== "function" ||
      !Number.isSafeInteger(plazoMs) || plazoMs < 1 || plazoMs > 30_000) throw new TypeError("fuentes de la ficha propia no disponibles");
  let resultado;
  const obtener = async (signal) => {
    if (resultado) return resultado;
    const consulta = await consultar(fetchImpl, plazoMs, signal);
    if (!consulta.sinFuente) resultado = consulta;
    return consulta;
  };
  const bloque = (presentar) => Object.freeze({
    async consultarPropios({ signal } = {}) {
      const consulta = await obtener(signal);
      if (consulta.sinFuente) return { estado: consulta.estado === 404 ? "no_configurado" : "denegado" };
      const items = presentar(consulta.ficha, traducir);
      return { estado: items.length ? "disponible" : "vacio", fuente: traducir("fuente_registro"), actualizado_en: consulta.consultadaEn, items };
    },
  });
  const fuentes = Object.freeze({ relaciones: bloque(presentarRelaciones), servicios: bloque(presentarServicios) });
  return Object.freeze({
    async preparar({ signal } = {}) {
      try {
        const consulta = await obtener(signal);
        return consulta.sinFuente ? {} : fuentes;
      } catch (causa) {
        // Sin una respuesta válida no hay fuente acreditada para esta vista:
        // los apartados no se ofrecen y se volverá a consultar al entrar.
        if (causa?.codigo === "operacion_abortada") throw causa;
        return {};
      }
    },
  });
}
