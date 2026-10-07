export const RUTA_POLITICA_OFERTAS = "/api/vec/bolsa/politica-ofertas";
export const RUTA_CAPACIDAD_POLITICA_OFERTAS = `${RUTA_POLITICA_OFERTAS}/capacidad`;
export const ESQUEMA_POLITICA_OFERTAS = "vec.bolsa.rrhh.politica-ofertas.v1";

const REFERENCIA = /^[A-Za-z0-9][A-Za-z0-9:._/-]{0,255}$/u;
const MUNICIPIO = /^[0-9]{5}$/u;
const SHA256 = /^[a-f0-9]{64}$/u;
const UNIDADES_DIAS = new Set(["dias_habiles", "dias_naturales"]);
// Apartado «plazas» (duda 75): solo los valores que ejecuta Bolsa.
export const LLAMADAS_PLAZAS = Object.freeze(["simultanea", "sucesiva"]);
export const TRAS_RENUNCIA_PLAZAS = Object.freeze(["siguiente_en_orden", "llamamiento_directo"]);
export const MAXIMO_HORAS_RESPUESTA = 720;
// Entradas del paquete de reglas de ejemplo con las que se rellena el
// apartado cuando la política de la bolsa aún no lo tiene.
const REGLAS_EJEMPLO_PLAZAS = Object.freeze({ llamada: "b30.plazas_llamada", respuesta: "b30.plazas_plazo_respuesta", tras: "b30.plazas_tras_renuncia" });

/**
 * Lector de reglas vigentes que comparte la lectura en curso: las propuestas
 * del formulario (plazo, plazas y confirmación) se piden a la vez al abrir el
 * llamamiento y reciben una sola respuesta. Al terminar se olvida, de modo
 * que volver a abrir o recargar lee otra vez lo vigente.
 */
export function crearLectorReglasCompartido(obtenerCliente) {
  let enCurso = null;
  return () => {
    enCurso ??= Promise.resolve().then(obtenerCliente).then((lector) => lector.reglas())
      .finally(() => { enCurso = null; });
    return enCurso;
  };
}
// Carga diferida: el cliente de reglas no entra en la precarga del portal.
const leerReglasCompartidas = crearLectorReglasCompartido(async () =>
  (await import("../../reglas/reglas.js?v=20261007-p5-solicitudes-reglas-v1")).crearCliente());

/** Reglas vigentes: con `cliente` (pruebas) se lee de él; si no, la lectura compartida. */
export function leerReglasVigentes({ cliente } = {}) {
  return cliente ? cliente.reglas() : leerReglasCompartidas();
}

/** La confirmación nueva procede del catálogo; sin regla se conserva el modo vigente. */
export async function cargarConfirmacionAdjudicacion({ cliente } = {}) {
  try {
    const datos = await leerReglasVigentes({ cliente });
    const reglas = datos.catalogos.filter((c) => c.modulo === "bolsa" && c.estado === "disponible")
      .flatMap((c) => c.reglas).filter((r) => r.clave === "b30.confirmacion_adjudicacion");
    return reglas.length === 1 && reglas[0].valor === "aceptacion_previa" ? reglas[0].valor : null;
  } catch {
    return null;
  }
}

/** Apartado de plazas completo y con valores admitidos. */
export function plazasCompletas(plazas) {
  return Boolean(plazas) && Object.keys(plazas).length === 3 && LLAMADAS_PLAZAS.includes(plazas.llamada)
    && Number.isSafeInteger(plazas.respuesta_horas) && plazas.respuesta_horas >= 1 && plazas.respuesta_horas <= MAXIMO_HORAS_RESPUESTA
    && TRAS_RENUNCIA_PLAZAS.includes(plazas.tras_renuncia);
}

/**
 * Propuesta de ejemplo para el apartado de plazas, leída de las reglas
 * vigentes (paquete retirable). Sin paquete o con valores ajenos devuelve null
 * y el formulario queda sin rellenar.
 */
export async function cargarEjemploPlazas({ cliente } = {}) {
  try {
    const datos = await leerReglasVigentes({ cliente });
    const reglas = new Map(datos.catalogos.filter((c) => c.modulo === "bolsa" && c.estado === "disponible")
      .flatMap((c) => c.reglas).map((r) => [r.clave, r]));
    const ejemplo = { llamada: reglas.get(REGLAS_EJEMPLO_PLAZAS.llamada)?.valor,
      respuesta_horas: reglas.get(REGLAS_EJEMPLO_PLAZAS.respuesta)?.cantidad,
      tras_renuncia: reglas.get(REGLAS_EJEMPLO_PLAZAS.tras)?.valor };
    return plazasCompletas(ejemplo) ? ejemplo : null;
  } catch {
    return null;
  }
}

export function validarPoliticaEditable(politica, { permitirLegada = false } = {}) {
  const plazo = politica?.plazo;
  const horas = plazo?.unidad === "horas_naturales";
  if (!plazo || (!horas && !UNIDADES_DIAS.has(plazo.unidad)) || !Number.isSafeInteger(plazo.cantidad)
    || plazo.cantidad < 1 || plazo.cantidad > (horas ? 720 : 30)
    || (plazo.inicio !== "notificacion" && !(permitirLegada && plazo.inicio === undefined))
    || plazo.computo !== (horas ? "continuo_utc" : "administrativo")
    || typeof plazo.municipio_sede !== "string" || !MUNICIPIO.test(plazo.municipio_sede)
    || politica?.adjudicacion?.criterio !== "orden_vigente"
    || politica.adjudicacion.elegibilidad !== "disposicion_en_plazo"
    || (politica.adjudicacion.confirmacion !== undefined && politica.adjudicacion.confirmacion !== "aceptacion_previa")
    || politica?.no_cubierta?.accion !== "llamamiento_directo"
    || politica.no_cubierta.condicion !== "sin_disposiciones_elegibles"
    || (politica.plazas !== undefined && politica.plazas !== null && !plazasCompletas(politica.plazas))) {
    throw new TypeError("política de ofertas no válida");
  }
  const copia = structuredClone(politica);
  if (copia.plazas === null) delete copia.plazas;
  return copia;
}

export function validarPoliticaRecibida(sobre) {
  const p = sobre?.data ?? sobre;
  if (!p || p.esquema !== ESQUEMA_POLITICA_OFERTAS
    || !REFERENCIA.test(p.bolsa_ref ?? "") || !Number.isSafeInteger(p.version)
    || p.version < 0 || typeof p.configurada !== "boolean"
    || (p.huella_sha256 !== undefined && !SHA256.test(p.huella_sha256))
    || p.ejemplo !== true || (p.recibo_ref !== undefined && !REFERENCIA.test(p.recibo_ref))
    || (p.publicada_en !== undefined && Number.isNaN(Date.parse(p.publicada_en)))
    || (p.configurada && (p.version < 1 || !p.huella_sha256 || !p.publicada_en))
    || (!p.configurada && (p.version !== 0 || p.politica !== null))
    || (Object.hasOwn(p, "puede_publicar") && typeof p.puede_publicar !== "boolean")) {
    throw new TypeError("respuesta de política de ofertas no válida");
  }
  // Las versiones anteriores conservan sus cuatro campos: lectura y replay,
  // sin inventar un inicio ni permitir que se publique otra versión desde ellas.
  const politica = { ...p, politica: p.configurada ? validarPoliticaEditable(p.politica, { permitirLegada: true }) : null };
  delete politica.puede_publicar;
  return politica;
}

function validarBolsa(bolsaRef) {
  if (typeof bolsaRef !== "string" || !REFERENCIA.test(bolsaRef)) throw new TypeError("bolsa no válida");
  return bolsaRef;
}

async function codigoError(respuesta) {
  try { return (await respuesta.json())?.error?.codigo ?? ""; } catch { return ""; }
}

export function crearClientePoliticaOfertas({ fetchImpl = fetch } = {}) {
  const opciones = (signal) => ({ credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error",
    referrerPolicy: "no-referrer", signal, headers: { Accept: "application/json" } });
  return Object.freeze({
    async consultar(bolsaRef, { signal } = {}) {
      const bolsa = validarBolsa(bolsaRef);
      const ruta = `${RUTA_POLITICA_OFERTAS}?${new URLSearchParams({ bolsa_ref: bolsa })}`;
      const respuesta = await fetchImpl(ruta, { ...opciones(signal), method: "GET" });
      if (!respuesta.ok) return { ok: false, status: respuesta.status, codigo: await codigoError(respuesta) };
      const politica = validarPoliticaRecibida(await respuesta.json());
      if (politica.bolsa_ref !== bolsa) throw new TypeError("política de otra bolsa");
      return { ok: true, politica };
    },
    async consultarCapacidad(bolsaRef, { signal } = {}) {
      const bolsa = validarBolsa(bolsaRef);
      const respuesta = await fetchImpl(RUTA_CAPACIDAD_POLITICA_OFERTAS, { ...opciones(signal), method: "POST",
        headers: { Accept: "application/json", "Content-Type": "application/json" },
        body: JSON.stringify({ bolsa_ref: bolsa }) });
      if (!respuesta.ok) return { ok: false, status: respuesta.status, codigo: await codigoError(respuesta) };
      const capacidad = await respuesta.json();
      if (typeof capacidad?.puede_publicar !== "boolean") throw new TypeError("capacidad de política no válida");
      return { ok: true, puede_publicar: capacidad.puede_publicar };
    },
    async publicar({ bolsa_ref, version_esperada, clave_idempotencia, politica }, { signal } = {}) {
      validarBolsa(bolsa_ref);
      if (!Number.isSafeInteger(version_esperada) || version_esperada < 0
        || typeof clave_idempotencia !== "string" || !REFERENCIA.test(clave_idempotencia)) {
        throw new TypeError("comando de política no válido");
      }
      const cuerpo = { bolsa_ref, version_esperada, clave_idempotencia, politica: validarPoliticaEditable(politica) };
      const respuesta = await fetchImpl(RUTA_POLITICA_OFERTAS, { ...opciones(signal), method: "POST",
        headers: { Accept: "application/json", "Content-Type": "application/json", "Idempotency-Key": clave_idempotencia },
        body: JSON.stringify(cuerpo) });
      if (!respuesta.ok) return { ok: false, status: respuesta.status, codigo: await codigoError(respuesta) };
      const recibida = validarPoliticaRecibida(await respuesta.json());
      if (recibida.bolsa_ref !== bolsa_ref || !recibida.configurada
        || recibida.version !== version_esperada + 1 || !recibida.recibo_ref
        || recibida.politica.plazo.inicio !== "notificacion") {
        throw new TypeError("recibo de política incoherente");
      }
      return { ok: true, status: respuesta.status, politica: recibida };
    },
  });
}
