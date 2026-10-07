export const RUTA_RPT_PUBLICA_V2 = "/api/vec/personal/rpt-publica/v2";
const MAXIMO_BYTES = 192 * 1024;
const MAXIMO_FRAGMENTOS = 256;
const SHA256 = /^[a-f0-9]{64}$/u;
const CLAVE = /^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$/u;
const CODIGO = /^[A-Z0-9][A-Z0-9-]{0,63}$/u;

export class ErrorClienteRPTPublicaV2 extends Error {
  constructor(codigo, estado = 0) {
    super(`cliente RPT publicada v2: ${codigo}`);
    this.name = "ErrorClienteRPTPublicaV2";
    this.codigo = codigo;
    this.estado = estado;
  }
}

function registro(v) { return v !== null && typeof v === "object" && !Array.isArray(v) && Object.getPrototypeOf(v) === Object.prototype; }
function exacto(v, claves) { return registro(v) && Object.keys(v).length === claves.length && claves.every((clave) => Object.hasOwn(v, clave)); }
function texto(v, maximo, vacio = false) { return typeof v === "string" && v === v.trim() && (vacio || v.length > 0) && [...v].length <= maximo && !/[\x00-\x1F\x7F-\x9F]/u.test(v); }
function entero(v, minimo = 0) { return Number.isSafeInteger(v) && v >= minimo; }
function listaTextos(v, maximo = 10000) { return Array.isArray(v) && v.length <= maximo && v.every((x) => texto(x, 512)); }
function error(codigo, estado = 0) { return new ErrorClienteRPTPublicaV2(codigo, estado); }

function validarConsulta(consulta) {
  if (!exacto(consulta, ["vista", "q", "limit", "offset", "categoria_clave", "centro_codigo"]) || !["categorias", "puestos"].includes(consulta.vista) ||
      !texto(consulta.q, 100, true) || !entero(consulta.limit, 1) || consulta.limit > 100 ||
      !entero(consulta.offset) || consulta.offset > 100000 ||
      (consulta.categoria_clave !== "" && !CLAVE.test(consulta.categoria_clave)) ||
      (consulta.centro_codigo !== "" && !/^[A-Za-z0-9-]{1,64}$/u.test(consulta.centro_codigo)) ||
      (consulta.vista !== "puestos" && (consulta.categoria_clave !== "" || consulta.centro_codigo !== ""))) throw new TypeError("consulta RPT v2 no válida");
  return Object.freeze({ ...consulta });
}

function validarCategoria(v) {
  return exacto(v, ["clave", "denominacion", "origen", "grupos", "escalas", "puestos", "dotacion", "nivel_destino_mediana", "complemento_especifico_anual_centimos_mediana"]) &&
    CLAVE.test(v.clave) && texto(v.denominacion, 512) && ["categoria", "denominacion"].includes(v.origen) &&
    listaTextos(v.grupos) && v.grupos.length > 0 && listaTextos(v.escalas) &&
    entero(v.puestos) && entero(v.dotacion) && entero(v.nivel_destino_mediana) && v.nivel_destino_mediana <= 99 &&
    entero(v.complemento_especifico_anual_centimos_mediana);
}

function validarPuesto(v) {
  return exacto(v, ["codigo", "denominacion", "centro_codigo", "centro", "delegacion", "grupos", "escala", "categoria_clave", "categorias_claves", "categorias_pendientes", "nivel_destino", "complemento_especifico_anual_centimos", "dotacion", "tipo", "provision"]) &&
    CODIGO.test(v.codigo) && texto(v.denominacion, 512) && texto(v.centro_codigo, 64) && texto(v.centro, 512) &&
    texto(v.delegacion, 512) && listaTextos(v.grupos) && texto(v.escala, 64, true) &&
    (v.categoria_clave === "" || CLAVE.test(v.categoria_clave)) && Array.isArray(v.categorias_claves) &&
    v.categorias_claves.length <= 100 && v.categorias_claves.every((clave) => CLAVE.test(clave)) &&
    new Set(v.categorias_claves).size === v.categorias_claves.length &&
    (v.categoria_clave === "" || v.categorias_claves.length === 1 && v.categoria_clave === v.categorias_claves[0]) &&
    Array.isArray(v.categorias_pendientes) && v.categorias_pendientes.length <= 100 &&
    v.categorias_pendientes.every((p) => exacto(p, ["denominacion", "origen"]) && texto(p.denominacion, 512) && ["categoria", "denominacion"].includes(p.origen)) &&
    entero(v.nivel_destino) && v.nivel_destino <= 99 && entero(v.complemento_especifico_anual_centimos) &&
    entero(v.dotacion) && texto(v.tipo, 64) && texto(v.provision, 128);
}

function validarFuente(v) { return exacto(v, ["documento", "importacion", "generado_en", "aviso"]) && texto(v.documento, 1024) && texto(v.importacion, 256) && texto(v.generado_en, 64) && texto(v.aviso, 4096); }
function validarResumen(v) { return exacto(v, ["puestos", "dotacion", "categorias", "centros"]) && ["puestos", "dotacion", "categorias", "centros"].every((k) => entero(v[k])); }
function validarEvidencia(v) { return exacto(v, ["recibo_ref", "decision_ref", "efecto_ref", "consumo_huella_sha256", "auditoria_ref", "consultada_en"]) && texto(v.recibo_ref, 160) && texto(v.decision_ref, 512) && texto(v.efecto_ref, 512) && SHA256.test(v.consumo_huella_sha256) && texto(v.auditoria_ref, 160) && texto(v.consultada_en, 64); }

function validarPagina(sobre, consulta) {
  const rpt = sobre?.data?.rpt;
  if (!exacto(sobre, ["data"]) || !exacto(sobre.data, ["rpt"]) ||
      !exacto(rpt, ["items", "total", "limit", "offset", "vista", "esquema", "estado", "publicacion_ref", "corte", "huella_sha256", "fuente", "resumen", "categorias_pendientes_grupo", "evidencia"]) ||
      rpt.esquema !== "vec.catalogo.rpt.candidato.v1" || rpt.estado !== "preparacion_no_autoritativa" ||
      !/^rpt-publicada:[a-z0-9][a-z0-9:-]{2,127}$/u.test(rpt.publicacion_ref) ||
      !/^\d{4}-\d{2}-\d{2}$/u.test(rpt.corte) || !SHA256.test(rpt.huella_sha256) ||
      !validarFuente(rpt.fuente) || !validarResumen(rpt.resumen) || !validarEvidencia(rpt.evidencia) ||
      rpt.evidencia.efecto_ref !== rpt.publicacion_ref || !listaTextos(rpt.categorias_pendientes_grupo, 1000) ||
      rpt.limit !== consulta.limit || rpt.offset !== consulta.offset || rpt.vista !== consulta.vista || !entero(rpt.total) ||
      !Array.isArray(rpt.items) || rpt.items.length !== Math.min(rpt.limit, Math.max(0, rpt.total - rpt.offset))) throw error("sobre_no_valido", 200);
  const verificar = consulta.vista === "puestos" ? validarPuesto : validarCategoria;
  const clave = consulta.vista === "puestos" ? "codigo" : "clave";
  const vistas = new Set();
  const items = rpt.items.map((item) => {
    if (!verificar(item) || vistas.has(item[clave])) throw error("fila_no_valida", 200);
    vistas.add(item[clave]);
    if (consulta.vista === "categorias") return Object.freeze({ ...item, grupos: Object.freeze([...item.grupos]), escalas: Object.freeze([...item.escalas]) });
    return Object.freeze({ ...item, grupos: Object.freeze([...item.grupos]), categorias_claves: Object.freeze([...item.categorias_claves]),
      categorias_pendientes: Object.freeze(item.categorias_pendientes.map((p) => Object.freeze({ ...p }))) });
  });
  return Object.freeze({ ...rpt, items: Object.freeze(items), fuente: Object.freeze({ ...rpt.fuente }), resumen: Object.freeze({ ...rpt.resumen }),
    evidencia: Object.freeze({ ...rpt.evidencia }), categorias_pendientes_grupo: Object.freeze([...rpt.categorias_pendientes_grupo]) });
}

async function leerAcotado(respuesta, signal) {
  const declarada = respuesta.headers?.get?.("content-length");
  if (declarada !== null && declarada !== undefined && (!/^(?:0|[1-9][0-9]*)$/u.test(declarada) || Number(declarada) > MAXIMO_BYTES)) throw error("respuesta_excesiva", respuesta.status);
  if (!respuesta.body || typeof respuesta.body.getReader !== "function") throw error("respuesta_incompatible", respuesta.status);
  const lector = respuesta.body.getReader();
  const trozos = [];
  let total = 0;
  try {
    while (true) {
      if (signal.aborted) throw error("operacion_abortada");
      const tramo = await lector.read();
      if (tramo.done) break;
      if (!(tramo.value instanceof Uint8Array) || tramo.value.byteLength === 0) throw error("respuesta_incompatible", respuesta.status);
      total += tramo.value.byteLength;
      if (total > MAXIMO_BYTES || trozos.length >= MAXIMO_FRAGMENTOS) throw error("respuesta_excesiva", respuesta.status);
      trozos.push(tramo.value);
    }
  } catch (causa) {
    await lector.cancel("respuesta descartada").catch(() => {});
    throw causa;
  } finally { try { lector.releaseLock(); } catch {} }
  const bytes = new Uint8Array(total);
  let posicion = 0;
  trozos.forEach((tramo) => { bytes.set(tramo, posicion); posicion += tramo.byteLength; });
  try { return JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(bytes)); }
  catch { throw error("json_no_valido", respuesta.status); }
}

export function crearClienteHTTPRPTPublicaV2({ fetchImpl = globalThis.fetch, plazoMs = 10_000 } = {}) {
  if (typeof fetchImpl !== "function" || !entero(plazoMs, 1) || plazoMs > 30_000) throw new TypeError("cliente RPT v2 no disponible");
  return Object.freeze({ async listar(entrada, { signal: externo } = {}) {
    const consulta = validarConsulta(entrada);
    if (externo !== undefined && (!externo || typeof externo.addEventListener !== "function" || typeof externo.removeEventListener !== "function" || typeof externo.aborted !== "boolean")) throw error("signal_no_valida");
    if (externo?.aborted) throw error("operacion_abortada");
    const params = new URLSearchParams({ vista: consulta.vista, q: consulta.q, limit: String(consulta.limit), offset: String(consulta.offset) });
    if (consulta.categoria_clave) params.set("categoria_clave", consulta.categoria_clave);
    if (consulta.centro_codigo) params.set("centro_codigo", consulta.centro_codigo);
    const ruta = `${RUTA_RPT_PUBLICA_V2}?${params}`;
    if (ruta.length - RUTA_RPT_PUBLICA_V2.length - 1 > 512) throw new TypeError("consulta RPT v2 demasiado larga");
    const controlador = new AbortController();
    const abortar = () => controlador.abort();
    externo?.addEventListener("abort", abortar, { once: true });
    let temporizador;
    try {
      const vencimiento = new Promise((_resolver, rechazar) => { temporizador = setTimeout(() => { controlador.abort(); rechazar(error("plazo_agotado")); }, plazoMs); });
      const operacion = (async () => {
        let respuesta;
        try { respuesta = await fetchImpl(ruta, { method: "GET", credentials: "omit", mode: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer", signal: controlador.signal }); }
        catch { throw error(externo?.aborted ? "operacion_abortada" : "red_no_disponible"); }
        if (!respuesta || respuesta.redirected || respuesta.ok !== true || respuesta.status !== 200) {
          await Promise.resolve(respuesta?.body?.cancel?.("respuesta descartada")).catch(() => {});
          throw error(respuesta?.status === 401 ? "autenticacion_requerida" : respuesta?.status === 403 ? "acceso_denegado" : "servicio_no_disponible", respuesta?.status || 0);
        }
        if (respuesta.headers?.get?.("content-type") !== "application/json; charset=utf-8") throw error("tipo_respuesta_no_valido", 200);
        return validarPagina(await leerAcotado(respuesta, controlador.signal), consulta);
      })();
      const pagina = await Promise.race([operacion, vencimiento]);
      if (externo?.aborted) throw error("operacion_abortada");
      if (controlador.signal.aborted) throw error("plazo_agotado");
      return pagina;
    } finally { clearTimeout(temporizador); externo?.removeEventListener("abort", abortar); controlador.abort(); }
  } });
}
