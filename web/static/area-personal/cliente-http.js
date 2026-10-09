import { validarRespuestaMiBolsa } from "./contrato.js?v=20261009-nombre-propio-v2";
import { traducir } from "./i18n.js";
import { IDIOMAS_DISPONIBLES } from "../comun/idioma.js";

const mensaje = (clave, variables) => traducir(`areaPersonal.cliente.${clave}`, variables);

const RUTA_MI_BOLSA = "/api/vec/bolsa/mi-bolsa";
export const RUTA_MIS_PREFERENCIAS = "/api/vec/usuarios/area-personal/mis-preferencias";
const CAMPOS_MIS_PREFERENCIAS = Object.freeze(["idioma", "tamano_texto", "alto_contraste", "tema", "inicio", "filas", "aviso_correo_tareas", "aviso_correo_plazos"]);
const TEMAS_PREFERENCIAS_V1 = Object.freeze(["sistema", "claro", "oscuro"]);
const TEMAS_PREFERENCIAS = Object.freeze({
  "usuarios-preferencias-v1": TEMAS_PREFERENCIAS_V1,
  "usuarios-preferencias-v2": Object.freeze([...TEMAS_PREFERENCIAS_V1, "diputacion_granada", "arena", "salvia", "lavanda", "azul_sereno", "noche_suave"]),
});
const RUTA_CONTACTO_PROPIO = "/api/vec/usuarios/contacto-propio";
const RUTA_RECIBO_CONTACTO_PROPIO = `${RUTA_CONTACTO_PROPIO}/recibo`;
export const RUTAS_OPERACIONES_CONTACTO = Object.freeze(Object.fromEntries(
  ["preparar", "confirmar", "cancelar", "consultas", "detalle"]
    .map((accion) => [accion, `${RUTA_CONTACTO_PROPIO}/operaciones/${accion}`]),
));
const MAXIMO_JSON_BYTES = 512 * 1024;
const DENEGACIONES_CONTACTO = Object.freeze({ 401: "autenticacion_requerida", 403: "acceso_denegado", 404: "no_encontrada" });
export class ErrorClienteAreaPersonal extends Error {
  constructor(codigo, mensaje, causa) {
    super(mensaje, causa ? { cause: causa } : undefined);
    this.name = "ErrorClienteAreaPersonal";
    this.codigo = codigo;
  }
}

export class ErrorPreferencias extends Error {
  constructor(codigo, estado = 0) {
    super(codigo);
    this.name = "ErrorPreferencias";
    this.codigo = codigo;
    this.estado = estado;
  }
}

const CODIGOS_PREFERENCIAS = Object.freeze({
  401: "autenticacion", 403: "denegado", 409: "conflicto",
  422: "validacion", 503: "servicio",
});

function validarValoresPreferencias(valores, catalogoVersion) {
  return valores && typeof valores === "object" && !Array.isArray(valores)
    && Object.hasOwn(TEMAS_PREFERENCIAS, catalogoVersion)
    && (valores.idioma === "navegador" || IDIOMAS_DISPONIBLES.some(({ codigo }) => codigo === valores.idioma))
    && ["normal", "grande", "muy_grande"].includes(valores.tamano_texto)
    && typeof valores.alto_contraste === "boolean"
    && TEMAS_PREFERENCIAS[catalogoVersion].includes(valores.tema)
    && ["cuadro", "peticiones", "bolsas"].includes(valores.inicio)
    && [20, 50, 100].includes(valores.filas)
    && typeof valores.aviso_correo_tareas === "boolean"
    && typeof valores.aviso_correo_plazos === "boolean"
    && Object.keys(valores).length === CAMPOS_MIS_PREFERENCIAS.length;
}

function validarEstadoPreferencias(estado) {
  if (!estado || typeof estado !== "object" || Array.isArray(estado)
    || typeof estado.persona_ref !== "string" || !estado.persona_ref
    || !Number.isSafeInteger(estado.version) || estado.version < 0
    || !Object.hasOwn(TEMAS_PREFERENCIAS, estado.catalogo_version_ref)
    || !validarValoresPreferencias(estado.valores, estado.catalogo_version_ref)) throw new ErrorPreferencias("respuesta");
  return estado;
}

function validarCatalogoPreferencias(catalogo) {
  if (!catalogo || typeof catalogo !== "object" || Array.isArray(catalogo)
    || !Object.hasOwn(TEMAS_PREFERENCIAS, catalogo.version_ref)
    || !["idiomas", "tamanos_texto", "temas", "inicios"].every((campo) =>
      Array.isArray(catalogo[campo]) && catalogo[campo].length > 0
      && catalogo[campo].every((opcion) => typeof opcion?.codigo === "string"
        && (campo !== "temas" || TEMAS_PREFERENCIAS[catalogo.version_ref].includes(opcion.codigo))
        && typeof opcion?.nombre_key === "string"))
    || !Array.isArray(catalogo.filas) || catalogo.filas.some((valor) => ![20, 50, 100].includes(valor))
    || !validarValoresPreferencias(catalogo.predeterminados, catalogo.version_ref)) throw new ErrorPreferencias("respuesta");
  return catalogo;
}

export function crearClientePreferencias({ fetchImpl = globalThis.fetch } = {}) {
  async function solicitar(metodo, cuerpo, { signal } = {}) {
    if (typeof fetchImpl !== "function") throw new ErrorPreferencias("servicio");
    let respuesta;
    try {
      respuesta = await fetchImpl(RUTA_MIS_PREFERENCIAS, {
        method: metodo, credentials: "same-origin", cache: "no-store", redirect: "error",
        referrerPolicy: "no-referrer", signal,
        headers: { Accept: "application/json", ...(cuerpo ? { "Content-Type": "application/json" } : {}) },
        ...(cuerpo ? { body: JSON.stringify(cuerpo) } : {}),
      });
    } catch (error) {
      if (signal?.aborted) throw error;
      throw new ErrorPreferencias("servicio");
    }
    if (respuesta?.status !== 200 && !(metodo === "PUT" && respuesta?.status === 201)) {
      throw new ErrorPreferencias(CODIGOS_PREFERENCIAS[respuesta?.status] || "respuesta", respuesta?.status || 0);
    }
    try {
      const envoltura = await leerJSONAcotado(respuesta);
      if (!envoltura || typeof envoltura !== "object" || Array.isArray(envoltura)
        || !envoltura.data || typeof envoltura.data !== "object" || Array.isArray(envoltura.data)) {
        throw new ErrorPreferencias("respuesta", respuesta.status);
      }
      return { valor: envoltura.data, estado: respuesta.status };
    }
    catch { throw new ErrorPreferencias("respuesta", 200); }
  }
  return Object.freeze({
    async cargar(opciones) {
      const { valor: dato } = await solicitar("GET", null, opciones);
      const catalogo = validarCatalogoPreferencias(dato?.catalogo);
      const estado = validarEstadoPreferencias(dato?.estado);
      if (catalogo.version_ref === "usuarios-preferencias-v1" && estado.catalogo_version_ref !== catalogo.version_ref) {
        throw new ErrorPreferencias("respuesta", 200);
      }
      if (!catalogo.temas.some((opcion) => opcion.codigo === estado.valores.tema)) throw new ErrorPreferencias("respuesta", 200);
      return Object.freeze({ catalogo, estado });
    },
    async guardar(cuerpo, opciones) {
      if (!cuerpo || Object.keys(cuerpo).length !== 4
        || !Object.keys(cuerpo).every((campo) => ["version_esperada", "catalogo_version_ref", "clave_operacion", "valores"].includes(campo))
        || !Number.isSafeInteger(cuerpo.version_esperada) || cuerpo.version_esperada < 0
        || !Object.hasOwn(TEMAS_PREFERENCIAS, cuerpo.catalogo_version_ref)
        || typeof cuerpo.clave_operacion !== "string" || !/^[A-Za-z0-9_.:-]{16,128}$/u.test(cuerpo.clave_operacion)
        || !validarValoresPreferencias(cuerpo.valores, cuerpo.catalogo_version_ref)) throw new ErrorPreferencias("validacion");
      const { valor: resultado, estado } = await solicitar("PUT", cuerpo, opciones);
      validarEstadoPreferencias(resultado);
      if (resultado.version !== cuerpo.version_esperada + 1
        || resultado.catalogo_version_ref !== cuerpo.catalogo_version_ref
        || typeof resultado.recibo_ref !== "string" || !resultado.recibo_ref
        || typeof resultado.fecha_utc !== "string" || !resultado.fecha_utc
        || typeof resultado.replay !== "boolean"
        || resultado.replay !== (estado === 200)
        || CAMPOS_MIS_PREFERENCIAS.some((campo) => resultado.valores[campo] !== cuerpo.valores[campo])) throw new ErrorPreferencias("respuesta", estado);
      return Object.freeze(resultado);
    },
  });
}

// El arranque del área personal continúa aunque esta consulta no responda.
export async function cargarPreferenciasIniciales(cliente, { tiempoMaximoMs = 8000 } = {}) {
  const controlador = new AbortController();
  let temporizador;
  const limite = new Promise((_, rechazar) => {
    temporizador = setTimeout(() => {
      controlador.abort();
      rechazar(new ErrorPreferencias("servicio"));
    }, tiempoMaximoMs);
  });
  try {
    return await Promise.race([cliente.cargar({ signal: controlador.signal }), limite]);
  } finally {
    clearTimeout(temporizador);
  }
}

export class ErrorOperacionContacto extends Error {
  constructor(codigo, estado = 0, operacionRef = "") {
    super(codigo);
    this.name = "ErrorOperacionContacto";
    this.codigo = codigo;
    this.estado = estado;
    this.operacionRef = operacionRef;
  }
}

const REFERENCIA_OPERACION_CONTACTO = /^opr_[A-Za-z0-9_-]{22,128}$/u;
export function referenciaOperacionContactoValida(ref) {
  return typeof ref === "string" && REFERENCIA_OPERACION_CONTACTO.test(ref);
}

function validarDTOOperacionContacto(valor) {
  if (!valor || typeof valor !== "object" || Array.isArray(valor)
    || !referenciaOperacionContactoValida(valor.operacion_ref)
    || !Number.isSafeInteger(valor.version_esperada) || valor.version_esperada < 0
    || valor.version_esperada >= Number.MAX_SAFE_INTEGER) {
    throw new ErrorOperacionContacto("respuesta_incompatible");
  }
  if (valor.estado === "confirmada") {
    if (valor.version !== valor.version_esperada + 1 || typeof valor.recibo_ref !== "string" || !valor.recibo_ref) {
      throw new ErrorOperacionContacto("respuesta_incompatible");
    }
  } else if (!["preparada", "cancelada"].includes(valor.estado)
    || valor.version !== undefined || valor.recibo_ref !== undefined) {
    throw new ErrorOperacionContacto("respuesta_incompatible");
  }
  const claves = valor.estado === "confirmada"
    ? ["operacion_ref", "estado", "version_esperada", "version", "recibo_ref"]
    : ["operacion_ref", "estado", "version_esperada"];
  if (Object.keys(valor).length !== claves.length || Object.keys(valor).some((clave) => !claves.includes(clave))) {
    throw new ErrorOperacionContacto("respuesta_incompatible");
  }
  return Object.freeze({ operacion_ref: valor.operacion_ref, estado: valor.estado,
    version_esperada: valor.version_esperada,
    ...(valor.estado === "confirmada" ? { version: valor.version, recibo_ref: valor.recibo_ref } : {}) });
}

export function crearClienteOperacionesContactoPropio({ fetchImpl = globalThis.fetch } = {}) {
  async function pedir(accion, cuerpo, { signal } = {}) {
    if (typeof fetchImpl !== "function" || !RUTAS_OPERACIONES_CONTACTO[accion]) {
      throw new ErrorOperacionContacto("servicio_no_disponible");
    }
    let respuesta;
    try {
      respuesta = await fetchImpl(RUTAS_OPERACIONES_CONTACTO[accion], {
        method: "POST", credentials: "same-origin", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer",
        headers: { Accept: "application/json", "Content-Type": "application/json" },
        body: JSON.stringify(cuerpo), signal,
      });
    } catch (error) {
      if (signal?.aborted) throw error;
      throw new ErrorOperacionContacto("servicio_no_disponible");
    }
    const estado = respuesta?.status;
    if (Object.hasOwn(DENEGACIONES_CONTACTO, estado)) {
      throw new ErrorOperacionContacto(DENEGACIONES_CONTACTO[estado], estado);
    }
    if (![200, 201, 400, 409, 503].includes(estado)) {
      throw new ErrorOperacionContacto("servicio_no_disponible", estado);
    }
    let dato;
    try { dato = await leerJSONAcotado(respuesta); }
    catch { throw new ErrorOperacionContacto("respuesta_incompatible", estado); }
    if (![200, 201].includes(estado)) {
      const admitidos = {
        400: ["peticion_invalida"],
        409: ["conflicto", "operacion_preparada"], 503: ["confirmacion_incierta", "servicio_no_disponible"],
      };
      const codigo = admitidos[estado]?.includes(dato?.codigo) ? dato.codigo : "respuesta_incompatible";
      const ref = ["operacion_preparada", "confirmacion_incierta"].includes(codigo)
        && referenciaOperacionContactoValida(dato?.operacion_ref) ? dato.operacion_ref : "";
      throw new ErrorOperacionContacto(codigo, estado, ref);
    }
    if (accion === "consultas") {
      if (estado !== 200 || !Array.isArray(dato?.operaciones) || dato.operaciones.length > cuerpo.limite
        || dato.siguiente_desde !== undefined && (!referenciaOperacionContactoValida(dato.siguiente_desde)
          || dato.siguiente_desde !== dato.operaciones.at(-1)?.operacion_ref)) {
        throw new ErrorOperacionContacto("respuesta_incompatible", estado);
      }
      return Object.freeze({ operaciones: Object.freeze(dato.operaciones.map(validarDTOOperacionContacto)),
        siguiente_desde: dato.siguiente_desde || "" });
    }
    const operacion = validarDTOOperacionContacto(dato);
    if (accion === "preparar" && !(operacion.estado === "preparada" || estado === 200 && operacion.estado === "confirmada")
      || accion === "confirmar" && operacion.estado !== "confirmada"
      || accion === "cancelar" && operacion.estado !== "cancelada"
      || accion !== "preparar" && accion !== "consultas" && operacion.operacion_ref !== cuerpo.operacion_ref
      || ["preparar", "confirmar"].includes(accion) && operacion.version_esperada !== cuerpo.version_esperada
      || ["preparar", "confirmar", "detalle"].includes(accion) && ![200, 201].includes(estado)
      || ["cancelar", "detalle"].includes(accion) && estado !== 200) {
      throw new ErrorOperacionContacto("respuesta_incompatible", estado);
    }
    return operacion;
  }
  return Object.freeze({
    preparar: (correo, versionEsperada, opciones) => pedir("preparar", { correo, version_esperada: versionEsperada }, opciones),
    confirmar: (ref, correo, versionEsperada, opciones) => pedir("confirmar", { operacion_ref: ref, correo, version_esperada: versionEsperada }, opciones),
    cancelar: (ref, opciones) => pedir("cancelar", { operacion_ref: ref }, opciones),
    listar: (limite = 20, despuesDe = "", opciones) => pedir("consultas", { limite, ...(despuesDe ? { despues_de: despuesDe } : {}) }, opciones),
    detalle: (ref, opciones) => pedir("detalle", { operacion_ref: ref }, opciones),
  });
}

async function leerJSONAcotado(respuesta) {
  const tipo = respuesta.headers?.get?.("Content-Type") || "";
  if (!/^application\/json(?:\s*;\s*charset=utf-8)?$/i.test(tipo)) {
    throw new ErrorClienteAreaPersonal("tipo_respuesta", mensaje("tipoRespuesta"));
  }
  const declarada = respuesta.headers?.get?.("Content-Length");
  if (declarada !== null && declarada !== undefined && declarada !== "") {
    if (!/^(?:0|[1-9][0-9]*)$/.test(declarada) || Number(declarada) > MAXIMO_JSON_BYTES) {
      throw new ErrorClienteAreaPersonal("respuesta_excesiva", mensaje("respuestaExcesiva"));
    }
  }
  const texto = await respuesta.text();
  if (new TextEncoder().encode(texto).byteLength > MAXIMO_JSON_BYTES) {
    throw new ErrorClienteAreaPersonal("respuesta_excesiva", mensaje("respuestaExcesiva"));
  }
  try {
    return JSON.parse(texto);
  } catch (error) {
    throw new ErrorClienteAreaPersonal("json_invalido", mensaje("jsonInvalido"), error);
  }
}

function mensajeHTTP(estado) {
  if (estado === 401) return mensaje("http401");
  if (estado === 403) return mensaje("http403");
  if (estado === 409) return mensaje("http409");
  if (estado === 422) return mensaje("http422");
  if (estado === 429) return mensaje("http429");
  return mensaje("httpOtro", { estado });
}

export function crearClienteHTTPAreaPersonal({ fetchImpl = globalThis.fetch } = {}) {
  if (typeof fetchImpl !== "function") {
    return Object.freeze({
      modo: "http",
      cargar: async () => { throw new ErrorClienteAreaPersonal("transporte_no_disponible", mensaje("sinTransporte")); },
    });
  }

  async function solicitar(ruta, opciones, estadosValidos) {
    let respuesta;
    try {
      respuesta = await fetchImpl(ruta, {
        cache: "no-store",
        redirect: "error",
        referrerPolicy: "no-referrer",
        ...opciones,
        credentials: "same-origin",
      });
    } catch (error) {
      throw new ErrorClienteAreaPersonal("servicio_no_disponible", mensaje("sinConexion"), error);
    }
    if (!estadosValidos.includes(respuesta?.status)) {
      const codigo = respuesta?.status === 401 ? "autenticacion_requerida"
        : respuesta?.status === 403 ? "acceso_denegado"
          : respuesta?.status === 404 ? "recurso_no_encontrado"
            : respuesta?.status === 503 ? "servicio_no_disponible" : "operacion_rechazada";
      throw new ErrorClienteAreaPersonal(codigo, mensajeHTTP(respuesta?.status));
    }
    return leerJSONAcotado(respuesta);
  }

  async function cargarContactoPropio({ signal } = {}) {
    const estado = await solicitar(RUTA_CONTACTO_PROPIO, {
      method: "GET", headers: { Accept: "application/json" }, signal,
    }, [200]);
    if (estado?.capacidad !== true) return null;
    if (!estado || typeof estado !== "object" || Array.isArray(estado)
      || typeof estado.encontrado !== "boolean" || !Number.isSafeInteger(estado.version)
      || estado.version < 0 || estado.encontrado !== (estado.version > 0)) {
      throw new ErrorClienteAreaPersonal("respuesta_incompatible", mensaje("contactoNoValido"));
    }
    let recibo = null;
    if (estado.encontrado) {
      const recibido = await solicitar(RUTA_RECIBO_CONTACTO_PROPIO, {
        method: "POST", headers: { Accept: "application/json", "Content-Type": "application/json" },
        body: JSON.stringify({ version: estado.version }), signal,
      }, [200]);
      if (!recibido || typeof recibido.recibo_ref !== "string" || !recibido.recibo_ref
        || recibido.version !== estado.version) throw new ErrorClienteAreaPersonal("respuesta_incompatible", mensaje("reciboContacto"));
      recibo = Object.freeze({ reciboRef: recibido.recibo_ref, version: recibido.version });
    }
    return Object.freeze({ autorizacion: Object.freeze({ capacidad: true, version: estado.version, consultarRecibo: estado.encontrado }), recibo });
  }
  async function cargar() {
    const envelope = await solicitar(RUTA_MI_BOLSA, {
      method: "GET",
      credentials: "same-origin",
      headers: { Accept: "application/json" },
    }, [200]);
    return Object.freeze({ fuente: "real", consulta: validarRespuestaMiBolsa(envelope) });
  }

  return Object.freeze({ modo: "http", cargar, cargarContactoPropio });
}
