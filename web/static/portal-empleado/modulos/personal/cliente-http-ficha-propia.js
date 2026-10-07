import { crearClienteHistoriaRelacionesPropia } from "./cliente-http-historia-relaciones-propia.js?v=20261004-personal-relaciones-v1";
import { crearClienteHistoriaServiciosPropia } from "./cliente-http-historia-servicios-propia.js?v=20261004-personal-historia-v1";
/**
 * Ficha propia de la persona empleada («mis datos» de Personal).
 *
 * El servidor deriva persona, perfil y empleado del certificado de la
 * petición y autoriza cada consulta; el cliente no envía referencias
 * de identidad. La fecha civil opcional se usa solo para Servicios.
 * Una consulta actual alimenta los apartados «Relaciones y
 * puestos» y «Servicios» con denominaciones legibles; nada se guarda fuera
 * de la memoria de la vista montada.
 */
import { crearTraductorFichaPropia, formatearDiasFichaPropia } from "./i18n-ficha-propia.js?v=20260929-i18n-personal-v1";

import { crearClienteExportacionServicios } from "./cliente-http-exportacion-servicios.js?v=20261004-personal-historia-v1";

export const RUTA_FICHA_PROPIA = "/api/interna/personal/mi-ficha";
export const PREFER_HISTORIA_SERVICIOS = "vec-personal-historia-servicios-v1";
export const PREFER_HISTORIAS_PROPIAS = PREFER_HISTORIA_SERVICIOS + ", vec-personal-historia-relaciones-v1";
export const ACCEPT_FICHA_PROPIA_EXPORTACION = 'application/json; profile="urn:vec:personal:ficha-propia:exportacion:v1"';

const MAXIMO_RESPUESTA_BYTES = 256 * 1024;
const MAXIMO_FILAS = 200;
/**
 * Longitud máxima de cada texto de la ficha (denominaciones de 000020 y
 * 000010). Es el mismo límite que admite la vista para cualquier celda, de
 * modo que un texto válido para el servidor nunca invalida la tabla.
 */
export const LIMITE_TEXTO_FICHA_PROPIA = 300;
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
  if (typeof valor !== "string" || !FECHA.test(valor) || valor.startsWith("0000")) return false;
  const d = new Date(`${valor}T12:00:00Z`);
  return Number.isFinite(d.getTime()) && d.toISOString().slice(0, 10) === valor;
}
function texto(valor) { return typeof valor === "string" && valor.length <= LIMITE_TEXTO_FICHA_PROPIA && !/[\u0000-\u001f\u007f]/u.test(valor); }
/** Recorta a LIMITE_TEXTO_FICHA_PROPIA con «…», sin partir un par sustituto. */
function recortar(valor) {
  if (valor.length <= LIMITE_TEXTO_FICHA_PROPIA) return valor;
  let corte = valor.slice(0, LIMITE_TEXTO_FICHA_PROPIA - 1);
  if (/[\ud800-\udbff]$/u.test(corte)) corte = corte.slice(0, -1);
  return `${corte}…`;
}

function validarSobre(sobre) {
  const datos = sobre?.data;
  const ficha = datos?.ficha;
  if (!claves(sobre, ["data"]) ||
      !(registro(datos) && ["ficha", "recibo_ref", "consultada_en"].every((k) => Object.hasOwn(datos, k)) && Object.keys(datos).every((k) => ["ficha", "recibo_ref", "consultada_en", "exportacion_servicios_disponible", "historia_servicios_disponible", "historia_relaciones_disponible"].includes(k))) ||
      (Object.hasOwn(datos, "exportacion_servicios_disponible") && typeof datos.exportacion_servicios_disponible !== "boolean") ||
      (Object.hasOwn(datos, "historia_servicios_disponible") && typeof datos.historia_servicios_disponible !== "boolean") ||
      (Object.hasOwn(datos, "historia_relaciones_disponible") && typeof datos.historia_relaciones_disponible !== "boolean") ||
      typeof datos.recibo_ref !== "string" || !/^fichapropia:[0-9a-f-]{36}$/u.test(datos.recibo_ref) ||
      typeof datos.consultada_en !== "string" || !INSTANTE.test(datos.consultada_en) || !Number.isFinite(Date.parse(datos.consultada_en)) ||
      !claves(ficha, ["corte", "relaciones", "servicios"]) || !claves(ficha.corte, ["vigente_en", "conocido_en"]) || !fecha(ficha.corte.vigente_en) ||
      typeof ficha.corte.conocido_en !== "string" || !INSTANTE.test(ficha.corte.conocido_en) || !Number.isFinite(Date.parse(ficha.corte.conocido_en)) ||
      !Array.isArray(ficha.relaciones) || !Array.isArray(ficha.servicios)) throw error("sobre_no_valido", 200);
  // Más filas de las que la vista pinta: estado propio, no desaparición.
  if (ficha.relaciones.length > MAXIMO_FILAS || ficha.servicios.length > MAXIMO_FILAS) return Object.freeze({ excedeLimite: true });
  for (const r of ficha.relaciones) {
    if (!claves(r, ["inicio", "fin", "estado", "regimen", "modalidad", "unidad", "puesto", "situacion"]) ||
        !fecha(r.inicio) || !fecha(r.fin, true) || !ESTADOS_RELACION.has(r.estado) ||
        ![r.regimen, r.modalidad, r.unidad, r.puesto, r.situacion].every(texto)) throw error("sobre_no_valido", 200);
  }
  for (const s of ficha.servicios) {
    if (!claves(s, ["inicio", "fin", "clase", "dias", "estado"]) || !fecha(s.inicio) || !fecha(s.fin) ||
        !texto(s.clase) || !Number.isSafeInteger(s.dias) || s.dias < 0 || !ESTADOS_SERVICIO.has(s.estado)) throw error("sobre_no_valido", 200);
  }
  return Object.freeze({ ficha, consultadaEn: datos.consultada_en, reciboRef: datos.recibo_ref, exportacionServiciosDisponible: datos.exportacion_servicios_disponible === true, historiaServiciosDisponible: datos.historia_servicios_disponible === true, historiaRelacionesDisponible: datos.historia_relaciones_disponible === true });
}

async function consultar(fetchImpl, plazoMs, externo, fechaReferencia = "") {
  const controlador = new AbortController();
  const abortar = () => controlador.abort();
  if (externo?.aborted) throw error("operacion_abortada");
  externo?.addEventListener?.("abort", abortar, { once: true });
  const temporizador = setTimeout(abortar, plazoMs);
  try {
    let respuesta;
    try {
      respuesta = await fetchImpl(fechaReferencia ? `${RUTA_FICHA_PROPIA}?fecha_referencia=${fechaReferencia}` : RUTA_FICHA_PROPIA, {
        method: "GET", credentials: "same-origin", mode: "same-origin", cache: "no-store",
        redirect: "error", referrerPolicy: "no-referrer", headers: { Accept: ACCEPT_FICHA_PROPIA_EXPORTACION, Prefer: PREFER_HISTORIAS_PROPIAS },
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
    // La ficha existe pero supera las filas que se pueden mostrar (SQL 54000):
    // los apartados se ofrecen con ese estado propio.
    if (estado === 422) {
      let cuerpo = "";
      try { cuerpo = await respuesta.text(); } catch {}
      if (cuerpo === '{"error":"excede_limite"}') return Object.freeze({ excedeLimite: true });
      throw error("estado_no_valido", estado);
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
    const validado = validarSobre(sobre);
    if (fechaReferencia && validado.ficha && validado.ficha.corte.vigente_en !== fechaReferencia) throw error("corte_no_valido", estado);
    return validado;
  } finally {
    clearTimeout(temporizador);
    externo?.removeEventListener?.("abort", abortar);
  }
}

/**
 * Una relación abierta llega con fin vacío y se muestra «Actualidad». La
 * columna Estado da la situación administrativa de una relación vigente y,
 * si la relación no está vigente, su propio estado (suspendida, finalizada):
 * la última situación de una relación terminada no la describe.
 */
function presentarRelaciones(ficha, t) {
  return ficha.relaciones.map((r) => ({
    desde: r.inicio,
    hasta: r.fin || t("relacion_abierta"),
    regimen: recortar([r.regimen, r.modalidad].filter(Boolean).join(" · ")),
    puesto: r.puesto,
    unidad: r.unidad,
    estado: r.estado === "vigente" ? (r.situacion || t("estado_relacion_vigente")) : t(`estado_relacion_${r.estado}`),
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
 * `preparar({signal})` hace una consulta cancelable. Solo 401, 403 y 404
 * retiran los apartados; un fallo temporal se presenta en ambos para que la
 * persona pueda reintentar sin confundirlo con ausencia de fuente.
 */
export function crearFuentesFichaPropia({ fetchImpl = globalThis.fetch, traducir = crearTraductorFichaPropia(), plazoMs = PLAZO_POR_DEFECTO_MS } = {}) {
  if (typeof fetchImpl !== "function" || typeof traducir !== "function" ||
      !Number.isSafeInteger(plazoMs) || plazoMs < 1 || plazoMs > 30_000) throw new TypeError("fuentes de la ficha propia no disponibles");
  const resultados = new Map();
  let revision = 0;
  let fechaActual = "";
  let sesionCaducada = false;
  const recibosSinExportacion = new Set();
  const obtener = async (signal, fechaReferencia = "") => {
    if (fechaReferencia !== "" && !fecha(fechaReferencia)) throw error("fecha_no_valida");
    if (signal?.aborted) throw error("operacion_abortada");
    if (sesionCaducada) return Object.freeze({ sinFuente: true, estado: 401 });
    if (resultados.has(fechaReferencia)) return resultados.get(fechaReferencia);
    const vigente = revision;
    try {
      const consulta = await consultar(fetchImpl, plazoMs, signal, fechaReferencia);
      if (signal?.aborted) throw error("operacion_abortada");
      if (vigente === revision) {
        // Solo la fecha pedida sin parámetros pertenece a Relaciones y puestos.
        resultados.set(fechaReferencia, consulta);
        if (!fechaReferencia && consulta.ficha) fechaActual = consulta.ficha.corte.vigente_en;
      }
      return consulta;
    } catch (causa) {
      if (signal?.aborted || causa?.codigo === "operacion_abortada") throw error("operacion_abortada");
      const fallo = Object.freeze({ error: true });
      if (vigente === revision) resultados.set(fechaReferencia, fallo);
      return fallo;
    }
  };
  const grupoActualizacion = Object.freeze({});
  const bloque = (presentar, admiteFecha = false) => Object.freeze({
    grupoActualizacion,
    get estadoInicial() { return resultados.get("")?.error ? "error" : "sin_consulta"; },
    get fechaReferencia() { return admiteFecha ? fechaActual : ""; },
    seleccionarFecha: admiteFecha,
    async consultarPropios({ signal, fechaReferencia = "" } = {}) {
      if (!admiteFecha && fechaReferencia !== "") throw error("fecha_no_admitida");
      const consulta = await obtener(signal, fechaReferencia);
      if (consulta.error) return { estado: "error" };
      if (consulta.excedeLimite) return { estado: "excede_limite" };
      if (consulta.sinFuente) return { estado: consulta.estado === 404 ? "no_configurado" : "denegado", ...(sesionCaducada ? { aviso_exportacion: "sesion_caducada" } : {}) };
      const items = presentar(consulta.ficha, traducir);
      return { estado: items.length ? "disponible" : "vacio", fuente: traducir("fuente_registro"), actualizado_en: consulta.consultadaEn, items, ...(!admiteFecha ? { historia_relaciones_disponible: consulta.historiaRelacionesDisponible } : {}), ...(admiteFecha ? { fecha_referencia: consulta.ficha.corte.vigente_en, exportacion_servicios_disponible: consulta.exportacionServiciosDisponible && !recibosSinExportacion.has(consulta.reciboRef), historia_servicios_disponible: consulta.historiaServiciosDisponible, recibo_ref: consulta.reciboRef, corte: Object.freeze({ ...consulta.ficha.corte }) } : {}) };
    },
    ...({ clienteHistoria: Object.freeze({ async consultar(entrada) {
      if (sesionCaducada) throw error("sesion_caducada", 401);
      const vigente = revision;
      try { return await (admiteFecha ? crearClienteHistoriaServiciosPropia : crearClienteHistoriaRelacionesPropia)({ fetchImpl, plazoMs }).consultar(entrada); }
      catch (causa) {
        if (vigente === revision && causa?.estado === 401) {
          revision += 1; resultados.clear(); recibosSinExportacion.clear(); sesionCaducada = true;
        }
        throw causa;
      }
    } }) }),
    ...(admiteFecha ? { async exportarPropios(entrada) {
      if (sesionCaducada || recibosSinExportacion.has(entrada?.reciboRef)) throw error("denegado", sesionCaducada ? 401 : 403);
      const vigente = revision;
      try { return await crearClienteExportacionServicios({ fetchImpl, plazoMs }).exportar(entrada); }
      catch (causa) {
        if (vigente === revision && causa?.estado === 401) {
          revision += 1; resultados.clear(); recibosSinExportacion.clear(); sesionCaducada = true;
        } else if (vigente === revision && [403, 404].includes(causa?.estado)) {
          recibosSinExportacion.add(entrada.reciboRef);
        }
        throw causa;
      }
    } } : {}),
    actualizar() { revision += 1; resultados.clear(); recibosSinExportacion.clear(); sesionCaducada = false; },
  });
  const fuentes = Object.freeze({ relaciones: bloque(presentarRelaciones), servicios: bloque(presentarServicios, true) });
  return Object.freeze({
    async preparar({ signal } = {}) {
      const consulta = await obtener(signal);
      return consulta.sinFuente ? {} : fuentes;
    },
  });
}
