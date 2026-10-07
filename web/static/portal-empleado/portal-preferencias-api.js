import { consultarJSON, ErrorConsultaJSON } from "../comun/http.js?v=20261007-p7-http-v1";
import { iniciarRegistroErrores } from "../comun/registro-errores.js?v=20261007-p7-http-v1";

const RUTA = "/api/vec/usuarios/mis-preferencias";
const MAX_BYTES = 65536;
const LIMITE_MS = 10000;
const CAMPOS = Object.freeze(["idioma", "tamano_texto", "alto_contraste", "tema", "inicio", "filas", "aviso_correo_tareas", "aviso_correo_plazos"]);
const TEMAS_V1 = Object.freeze(["sistema", "claro", "oscuro"]);
const TEMAS_V2 = Object.freeze([...TEMAS_V1, "diputacion_granada", "arena", "salvia", "lavanda", "azul_sereno", "noche_suave"]);
const TEMAS_POR_VERSION = Object.freeze({ "usuarios-preferencias-v1": TEMAS_V1, "usuarios-preferencias-v2": TEMAS_V2 });

export class ErrorPreferencias extends Error {
  constructor(estado = 0, codigo = "") {
    super(`preferencias HTTP ${estado}`);
    this.estado = estado;
    this.codigo = codigo;
    this.clave_i18n = codigo ? `api.usuarios.preferencias.error.${codigo}` : "";
  }
}

function objeto(valor) { return valor && typeof valor === "object" && !Array.isArray(valor); }
function validarValores(valores, temasPermitidos = TEMAS_V2) {
  if (!objeto(valores) || Object.keys(valores).length !== CAMPOS.length || !CAMPOS.every((campo) => Object.hasOwn(valores, campo))) throw new TypeError("valores de preferencias inválidos");
  if (!["navegador", "es", "en"].includes(valores.idioma)
    || !["normal", "grande", "muy_grande"].includes(valores.tamano_texto)
    || typeof valores.alto_contraste !== "boolean"
    || !temasPermitidos.includes(valores.tema)
    || !["cuadro", "peticiones", "bolsas"].includes(valores.inicio)
    || ![20, 50, 100].includes(valores.filas)
    || typeof valores.aviso_correo_tareas !== "boolean"
    || typeof valores.aviso_correo_plazos !== "boolean") throw new TypeError("valores de preferencias inválidos");
  return Object.freeze({ ...valores });
}

function validarCatalogo(catalogo) {
  if (!objeto(catalogo) || !Object.hasOwn(TEMAS_POR_VERSION, catalogo.version_ref)) throw new TypeError("catálogo de preferencias inválido");
  for (const [nombre, permitidos] of [["idiomas", ["navegador", "es", "en"]], ["tamanos_texto", ["normal", "grande", "muy_grande"]], ["temas", TEMAS_POR_VERSION[catalogo.version_ref]], ["inicios", ["cuadro", "peticiones", "bolsas"]]]) {
    const opciones = catalogo[nombre];
    if (!Array.isArray(opciones) || opciones.length === 0 || opciones.some((opcion) => !objeto(opcion) || !permitidos.includes(opcion.codigo) || typeof opcion.nombre_key !== "string")) throw new TypeError("opciones de preferencias inválidas");
  }
  if (!Array.isArray(catalogo.filas) || catalogo.filas.length === 0 || catalogo.filas.some((filas) => ![20, 50, 100].includes(filas))) throw new TypeError("opciones de filas inválidas");
  validarValores(catalogo.predeterminados, TEMAS_POR_VERSION[catalogo.version_ref]);
  return catalogo;
}

async function contenidoJSON(respuesta) {
  const tipo = respuesta.headers?.get?.("content-type") || "";
  if (!tipo.toLowerCase().includes("application/json")) throw new TypeError("respuesta de preferencias no JSON");
  const longitud = Number(respuesta.headers?.get?.("content-length") || 0);
  if (longitud > MAX_BYTES) throw new TypeError("respuesta de preferencias demasiado grande");
  if (!respuesta.body?.getReader) return respuesta.json();
  const lector = respuesta.body.getReader();
  const partes = [];
  let total = 0;
  try {
    while (true) {
      const { done, value } = await lector.read();
      if (done) break;
      total += value.byteLength;
      if (total > MAX_BYTES) throw new TypeError("respuesta de preferencias demasiado grande");
      partes.push(value);
    }
  } finally { await lector.cancel().catch(() => {}); }
  const bytes = new Uint8Array(total);
  let offset = 0;
  for (const parte of partes) { bytes.set(parte, offset); offset += parte.byteLength; }
  return JSON.parse(new TextDecoder().decode(bytes));
}

export function crearClientePreferencias({ fetchImpl = globalThis.fetch } = {}) {
  if (typeof fetchImpl !== "function") throw new TypeError("cliente HTTP no disponible");
  const detenerRegistroErrores = iniciarRegistroErrores();
  async function solicitar(metodo, cuerpo, signal) {
    if (metodo === "GET") {
      try {
        return await consultarJSON(RUTA, { fetchImpl, signal, limiteBytes: MAX_BYTES, plazoMs: LIMITE_MS });
      } catch (fallo) {
        if (signal?.aborted) throw fallo;
        if (fallo instanceof ErrorConsultaJSON) {
          const codigos = { 401: "no_autenticado", 403: "prohibido", 409: "conflicto", 422: "peticion_invalida", 503: "no_disponible" };
          throw new ErrorPreferencias(fallo.estado, codigos[fallo.estado] ?? "");
        }
        throw new ErrorPreferencias(0);
      }
    }
    const controlador = new AbortController();
    const abortar = () => controlador.abort();
    if (signal?.aborted) abortar();
    else signal?.addEventListener?.("abort", abortar, { once: true });
    const temporizador = setTimeout(abortar, LIMITE_MS);
    let respuesta;
    try {
      respuesta = await fetchImpl(RUTA, {
        method: metodo, credentials: "same-origin", mode: "same-origin", redirect: "error",
        cache: "no-store", referrerPolicy: "no-referrer", signal: controlador.signal,
        headers: { Accept: "application/json", ...(cuerpo ? { "Content-Type": "application/json" } : {}) },
        ...(cuerpo ? { body: JSON.stringify(cuerpo) } : {}),
      });
      if (!respuesta.ok) {
        let codigo = "";
        try {
          const error = (await contenidoJSON(respuesta))?.error;
          if (error?.clave_i18n === `api.usuarios.preferencias.error.${error.codigo}`
            && ["no_autenticado", "prohibido", "conflicto", "peticion_invalida", "no_disponible"].includes(error.codigo)) codigo = error.codigo;
        } catch { /* El estado HTTP basta para dar una respuesta segura. */ }
        throw new ErrorPreferencias(respuesta.status, codigo);
      }
      return await contenidoJSON(respuesta);
    } catch (error) {
      if (signal?.aborted) throw error;
      if (error instanceof ErrorPreferencias) throw error;
      throw new ErrorPreferencias(0);
    } finally {
      clearTimeout(temporizador);
      signal?.removeEventListener?.("abort", abortar);
    }
  }
  return Object.freeze({
    detenerRegistroErrores,
    async consultar({ signal } = {}) {
      const cuerpo = await solicitar("GET", null, signal);
      const datos = cuerpo?.data;
      const catalogo = validarCatalogo(datos?.catalogo);
      const estado = datos?.estado;
      if (!objeto(estado) || !Number.isSafeInteger(estado.version) || estado.version < 0
        || !Object.hasOwn(TEMAS_POR_VERSION, estado.catalogo_version_ref)
        || (catalogo.version_ref === "usuarios-preferencias-v1" && estado.catalogo_version_ref !== catalogo.version_ref)) {
        throw new TypeError("estado de preferencias inválido");
      }
      const valores = validarValores(estado.valores, TEMAS_POR_VERSION[estado.catalogo_version_ref]);
      if (!catalogo.temas.some((opcion) => opcion.codigo === valores.tema)) throw new TypeError("estado de preferencias inválido");
      return Object.freeze({ catalogo, estado: Object.freeze({ version: estado.version, valores }) });
    },
    async guardar({ version, catalogoVersion, clave, valores, signal }) {
      if (!Number.isSafeInteger(version) || version < 0 || !Object.hasOwn(TEMAS_POR_VERSION, catalogoVersion)
        || typeof clave !== "string" || !/^[A-Za-z0-9:_.-]{16,128}$/u.test(clave)) throw new TypeError("petición de preferencias inválida");
      const cuerpo = await solicitar("PUT", {
        version_esperada: version, catalogo_version_ref: catalogoVersion,
        clave_operacion: clave, valores: validarValores(valores, TEMAS_POR_VERSION[catalogoVersion]),
      }, signal);
      const recibo = cuerpo?.data;
      if (!objeto(recibo) || typeof recibo.recibo_ref !== "string" || recibo.recibo_ref.length === 0
        || !Number.isSafeInteger(recibo.version) || recibo.version < 1
        || recibo.catalogo_version_ref !== catalogoVersion || typeof recibo.fecha_utc !== "string") throw new TypeError("recibo de preferencias inválido");
      return Object.freeze({ recibo_ref: recibo.recibo_ref, version: recibo.version,
        fecha_utc: recibo.fecha_utc, replay: recibo.replay === true, valores: validarValores(recibo.valores, TEMAS_POR_VERSION[catalogoVersion]) });
    },
  });
}
