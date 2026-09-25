export const RUTA_REGISTRO_B2 = "/api/vec/personal";

export class ErrorRegistroB2 extends Error {
  constructor(codigo, estado = 0) {
    super(codigo);
    this.name = "ErrorRegistroB2";
    this.codigo = codigo;
    this.estado = estado;
  }
}
const UUID_V4 = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/u;
const CAMPOS_ALTA = new Set(["persona_ref", "organismo_ref", "unidad_ref", "regimen", "modalidad", "vigente_desde", "vigente_hasta", "acto_ref", "fuente_ref", "fuente_version", "fuente_huella_sha256"]);
const CAMPOS_HECHO = new Set(["tipo", "empleado_ref", "relacion_ref", "revision_esperada", "relacion_version_esperada", "unidad_ref", "regimen", "modalidad", "estado", "plaza_ref", "puesto_ref", "situacion", "clase_servicio", "clase_ocupacion", "version_plaza_ref", "version_puesto_ref", "periodo_desde", "periodo_hasta", "dias_reconocidos", "vigente_desde", "vigente_hasta", "acto_ref", "fuente_ref", "fuente_version", "fuente_huella_sha256"]);
function catalogoVersionado(valor) { return valor && typeof valor === "object" && !Array.isArray(valor) && Object.keys(valor).length === 2 && /^[a-z][a-z0-9_:-]{2,159}$/u.test(valor.ref) && Number.isSafeInteger(valor.version) && valor.version >= 1 && valor.version <= 2147483647; }
function validarCuerpo(cuerpo, campos, clave) {
  if (!cuerpo || typeof cuerpo !== "object" || Array.isArray(cuerpo) || Object.keys(cuerpo).some((campo) => !campos.has(campo)) ||
      !UUID_V4.test(clave) || !fechaCivil(cuerpo.vigente_desde) || (cuerpo.vigente_hasta !== undefined && !fechaCivil(cuerpo.vigente_hasta)) ||
      typeof cuerpo.acto_ref !== "string" || !cuerpo.acto_ref || typeof cuerpo.fuente_ref !== "string" || !cuerpo.fuente_ref ||
      !Number.isSafeInteger(cuerpo.fuente_version) || cuerpo.fuente_version < 1 || !/^[a-f0-9]{64}$/u.test(cuerpo.fuente_huella_sha256)) throw new TypeError("acto de Registro de Personal no válido");
  if (campos === CAMPOS_ALTA && (typeof cuerpo.persona_ref !== "string" || !/^per_[A-Za-z0-9_-]{22,128}$/u.test(cuerpo.persona_ref) ||
      !["organismo_ref", "unidad_ref"].every((campo) => typeof cuerpo[campo] === "string" && cuerpo[campo]) ||
      !catalogoVersionado(cuerpo.regimen) || !catalogoVersionado(cuerpo.modalidad))) throw new TypeError("alta de empleado no válida");
  const nuevaRelacion = cuerpo.relacion_version_esperada === 0 && (cuerpo.relacion_ref === undefined || cuerpo.relacion_ref === "");
  const revisionRelacion = cuerpo.relacion_version_esperada >= 1 && /^rel_[A-Za-z0-9_-]{22,128}$/u.test(cuerpo.relacion_ref);
  if (campos === CAMPOS_HECHO && (!["relacion", "ocupacion", "servicio", "situacion"].includes(cuerpo.tipo) ||
      !/^emp_[A-Za-z0-9_-]{22,128}$/u.test(cuerpo.empleado_ref) || !Number.isSafeInteger(cuerpo.revision_esperada) || cuerpo.revision_esperada < 1 ||
      !Number.isSafeInteger(cuerpo.relacion_version_esperada) || cuerpo.relacion_version_esperada < (cuerpo.tipo === "relacion" ? 0 : 1) ||
      (cuerpo.tipo !== "relacion" && !/^rel_[A-Za-z0-9_-]{22,128}$/u.test(cuerpo.relacion_ref)) ||
      (cuerpo.tipo === "relacion" && !nuevaRelacion && !revisionRelacion) ||
      (["relacion", "ocupacion"].includes(cuerpo.tipo) && !catalogoVersionado(cuerpo.modalidad)) ||
      (cuerpo.tipo === "relacion" && !catalogoVersionado(cuerpo.regimen)) ||
      (cuerpo.tipo === "situacion" && !catalogoVersionado(cuerpo.situacion)) ||
      (cuerpo.tipo === "servicio" && !catalogoVersionado(cuerpo.clase_servicio)) ||
      (cuerpo.tipo === "ocupacion" && !["titular", "provisional", "temporal", "reserva"].includes(cuerpo.clase_ocupacion)))) throw new TypeError("hecho de empleado no válido");
  return JSON.stringify(cuerpo);
}
function fechaCivil(valor) { if (typeof valor !== "string" || !/^\d{4}-\d{2}-\d{2}$/u.test(valor)) return false; const fecha = new Date(`${valor}T12:00:00Z`); return Number.isFinite(fecha.getTime()) && fecha.toISOString().slice(0, 10) === valor; }
function validarRecibo(respuesta, estado) {
  const r = respuesta?.data?.recibo;
  const acceso = respuesta?.data?.acceso_actual;
  if (!r || typeof r !== "object" || !/^perrec_[0-9a-f]{32}$/u.test(r.recibo_ref) ||
      !/^emp_[A-Za-z0-9_-]{22,128}$/u.test(r.empleado_ref) || !/^rel_[A-Za-z0-9_-]{22,128}$/u.test(r.relacion_ref) ||
      !Number.isSafeInteger(r.version) || r.version < 1 || !["alta", "relacion", "ocupacion", "servicio", "situacion"].includes(r.tipo) ||
      !Number.isFinite(Date.parse(r.registrado_en)) ||
      !["decision_ref", "efecto_ref", "auditoria_ref"].every((campo) => typeof r[campo] === "string" && r[campo]) ||
      !/^[a-f0-9]{64}$/u.test(r.consumo_huella_sha256) || r.eficacia_administrativa !== false || r.firma_oficial !== false ||
      (r.tipo === "alta" && !/^pep_[A-Za-z0-9_-]{22,128}$/u.test(r.proyeccion_ref)) ||
      (r.tipo !== "alta" && !/^(?:rel|ocu|srv|sit)_[A-Za-z0-9_-]{22,128}$/u.test(r.hecho_ref)) ||
      !acceso || typeof acceso !== "object" || !["registrado", "replay"].includes(acceso.estado_replay) ||
      !["decision_ref", "efecto_ref", "auditoria_ref"].every((campo) => typeof acceso[campo] === "string" && acceso[campo]) ||
      !/^[a-f0-9]{64}$/u.test(acceso.consumo_huella_sha256) || !Number.isFinite(Date.parse(acceso.consultada_en)) ||
      (estado === 201 && acceso.estado_replay !== "registrado") || (estado === 200 && acceso.estado_replay !== "replay")) throw new ErrorRegistroB2("recibo_incompatible", estado);
  return Object.freeze({ recibo: Object.freeze({ ...r }), accesoActual: Object.freeze({ ...acceso }) });
}

function consultaValida({ vigenteEn, conocidoEn }) {
  const fecha = new Date(`${vigenteEn}T12:00:00Z`);
  if (typeof vigenteEn !== "string" || !/^\d{4}-\d{2}-\d{2}$/u.test(vigenteEn) ||
      !Number.isFinite(fecha.getTime()) || fecha.toISOString().slice(0, 10) !== vigenteEn ||
      typeof conocidoEn !== "string" || !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{6}Z$/u.test(conocidoEn) ||
      !Number.isFinite(Date.parse(conocidoEn))) throw new TypeError("corte de Registro de Personal no válido");
}

async function leerJSON(respuesta, signal) {
  const longitud = respuesta.headers?.get?.("content-length");
  if (longitud !== null && longitud !== undefined && (!/^\d+$/u.test(longitud) || Number(longitud) > 512 * 1024)) {
    await respuesta.body?.cancel?.();
    throw new ErrorRegistroB2("respuesta_excesiva", respuesta.status);
  }
  if (!respuesta.body?.getReader) throw new ErrorRegistroB2("respuesta_no_incremental", respuesta.status);
  const lector = respuesta.body.getReader();
  const trozos = [];
  let total = 0;
  try {
    while (true) {
      if (signal.aborted) throw new ErrorRegistroB2("operacion_abortada");
      const siguiente = await lector.read();
      if (siguiente.done) break;
      if (!(siguiente.value instanceof Uint8Array) || siguiente.value.byteLength === 0) throw new ErrorRegistroB2("respuesta_incompatible", respuesta.status);
      total += siguiente.value.byteLength;
      if (total > 512 * 1024 || trozos.length >= 256) throw new ErrorRegistroB2("respuesta_excesiva", respuesta.status);
      trozos.push(siguiente.value);
    }
  } catch (error) {
    await lector.cancel().catch(() => {});
    throw error;
  } finally {
    lector.releaseLock?.();
  }
  const bytes = new Uint8Array(total);
  let posicion = 0;
  for (const trozo of trozos) { bytes.set(trozo, posicion); posicion += trozo.byteLength; }
  try { return JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(bytes)); }
  catch { throw new ErrorRegistroB2("json_invalido", respuesta.status); }
}

export function crearClienteRegistroB2({ fetchImpl = globalThis.fetch, plazoMs = 10_000 } = {}) {
  if (typeof fetchImpl !== "function" || !Number.isSafeInteger(plazoMs) || plazoMs < 1 || plazoMs > 30_000) throw new TypeError("cliente de Registro de Personal no disponible");
  async function get(ruta, parametros, signal) {
    if (signal !== undefined && (!signal || typeof signal.addEventListener !== "function" || typeof signal.aborted !== "boolean")) throw new TypeError("señal de cancelación no válida");
    if (signal?.aborted) throw new ErrorRegistroB2("operacion_abortada");
    const controlador = new AbortController();
    const abortar = () => controlador.abort();
    signal?.addEventListener("abort", abortar, { once: true });
    const temporizador = setTimeout(abortar, plazoMs);
    try {
      let respuesta;
      try {
        respuesta = await fetchImpl(`${ruta}?${parametros}`, {
          method: "GET", credentials: "same-origin", mode: "same-origin", cache: "no-store",
          redirect: "error", referrerPolicy: "no-referrer", signal: controlador.signal,
        });
      } catch {
        throw new ErrorRegistroB2(controlador.signal.aborted ? "operacion_abortada" : "red_no_disponible");
      }
      if (!respuesta || respuesta.redirected || respuesta.status !== 200 || respuesta.ok !== true) {
        let codigo = "estado_no_valido";
        if (respuesta?.status === 503 && respuesta.headers?.get?.("content-type") === "application/json; charset=utf-8") {
          try { const cuerpo = await leerJSON(respuesta, controlador.signal); if (cuerpo?.error?.codigo === "cobertura_no_acreditada") codigo = "cobertura_no_acreditada"; }
          catch { /* Un error sin sobre válido sigue siendo fallo general. */ }
        } else await respuesta?.body?.cancel?.().catch(() => {});
        throw new ErrorRegistroB2(codigo, respuesta?.status || 0);
      }
      if (respuesta.headers?.get?.("content-type") !== "application/json; charset=utf-8") {
        await respuesta.body?.cancel?.().catch(() => {});
        throw new ErrorRegistroB2("tipo_respuesta_no_valido", respuesta.status);
      }
      return await leerJSON(respuesta, controlador.signal);
    } finally {
      clearTimeout(temporizador);
      signal?.removeEventListener("abort", abortar);
    }
  }
  async function post(ruta, cuerpo, campos, claveIdempotencia) {
    const bytes = validarCuerpo(cuerpo, campos, claveIdempotencia);
    const controlador = new AbortController();
    const temporizador = setTimeout(() => controlador.abort(), plazoMs);
    try {
      let respuesta;
      try {
        respuesta = await fetchImpl(ruta, {
          method: "POST", credentials: "same-origin", mode: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer",
          headers: { "content-type": "application/json", "Idempotency-Key": claveIdempotencia }, body: bytes, signal: controlador.signal,
        });
      } catch { throw new ErrorRegistroB2("resultado_incierto"); }
      if (!respuesta || respuesta.redirected) throw new ErrorRegistroB2("resultado_incierto");
      if (respuesta.status !== 200 && respuesta.status !== 201) {
        if ([400, 401, 403, 404, 409].includes(respuesta.status)) { await respuesta.body?.cancel?.().catch(() => {}); throw new ErrorRegistroB2("acto_rechazado", respuesta.status); }
        await respuesta.body?.cancel?.().catch(() => {});
        throw new ErrorRegistroB2("resultado_incierto", respuesta.status);
      }
      if (respuesta.headers?.get?.("content-type") !== "application/json; charset=utf-8") throw new ErrorRegistroB2("resultado_incierto", respuesta.status);
      let datos;
      try { datos = await leerJSON(respuesta, controlador.signal); }
      catch { throw new ErrorRegistroB2("resultado_incierto", respuesta.status); }
      try { return validarRecibo(datos, respuesta.status); }
      catch { throw new ErrorRegistroB2("resultado_incierto", respuesta.status); }
    } finally { clearTimeout(temporizador); }
  }
  return Object.freeze({
    async consultarFicha({ empleadoRef, vigenteEn, conocidoEn, signal }) {
      if (typeof empleadoRef !== "string" || !/^emp_[A-Za-z0-9_-]{22,128}$/u.test(empleadoRef)) throw new TypeError("referencia de empleado no válida");
      consultaValida({ vigenteEn, conocidoEn });
      const query = new URLSearchParams({ vigente_en: vigenteEn, conocido_en: conocidoEn });
      const respuesta = await get(`${RUTA_REGISTRO_B2}/empleados/${encodeURIComponent(empleadoRef)}`, query, signal);
      if (!respuesta?.data?.ficha || !respuesta.data.evidencia) throw new ErrorRegistroB2("sobre_no_valido", 200);
      return respuesta.data;
    },
    async listarVacantes({ vigenteEn, conocidoEn, limite = 25, cursor = "", signal }) {
      consultaValida({ vigenteEn, conocidoEn });
      if (!Number.isSafeInteger(limite) || limite < 1 || limite > 100 || typeof cursor !== "string" || cursor.length > 2048) throw new TypeError("paginación de vacantes no válida");
      const query = new URLSearchParams({ vigente_en: vigenteEn, conocido_en: conocidoEn, limite: String(limite) });
      if (cursor) query.set("cursor", cursor);
      const respuesta = await get(`${RUTA_REGISTRO_B2}/vacantes`, query, signal);
      if (!respuesta?.data?.pagina || !respuesta.data.evidencia) throw new ErrorRegistroB2("sobre_no_valido", 200);
      return respuesta.data;
    },
    async listarEmpleados({ vigenteEn, conocidoEn, limite = 25, cursor = "", signal }) {
      consultaValida({ vigenteEn, conocidoEn });
      if (!Number.isSafeInteger(limite) || limite < 1 || limite > 100 || typeof cursor !== "string" || cursor.length > 256) throw new TypeError("paginación de empleados no válida");
      const query = new URLSearchParams({ vigente_en: vigenteEn, conocido_en: conocidoEn, limite: String(limite) });
      if (cursor) query.set("cursor", cursor);
      const respuesta = await get(`${RUTA_REGISTRO_B2}/empleados-organismo`, query, signal);
      if (!respuesta?.data?.pagina || !respuesta.data.evidencia) throw new ErrorRegistroB2("sobre_no_valido", 200);
      return respuesta.data;
    },
    registrarAlta(cuerpo, { claveIdempotencia } = {}) { return post(`${RUTA_REGISTRO_B2}/empleados`, cuerpo, CAMPOS_ALTA, claveIdempotencia); },
    registrarHecho(cuerpo, { claveIdempotencia } = {}) { return post(`${RUTA_REGISTRO_B2}/hechos`, cuerpo, CAMPOS_HECHO, claveIdempotencia); },
  });
}
