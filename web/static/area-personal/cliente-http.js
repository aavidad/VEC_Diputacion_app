import { validarRecibo, validarRespuestaMiBolsa } from "./contrato.js";

const RUTA_MI_BOLSA = "/api/vec/bolsa/mi-bolsa";
const RUTA_CONTACTO_PROPIO = "/api/vec/usuarios/contacto-propio";
const RUTA_RECIBO_CONTACTO_PROPIO = `${RUTA_CONTACTO_PROPIO}/recibo`;
export const RUTAS_OPERACIONES_CONTACTO = Object.freeze(Object.fromEntries(
  ["preparar", "confirmar", "cancelar", "consultas", "detalle"]
    .map((accion) => [accion, `${RUTA_CONTACTO_PROPIO}/operaciones/${accion}`]),
));
const MAXIMO_JSON_BYTES = 512 * 1024;
const MAXIMO_SOLICITUD_BYTES = 64 * 1024;
const DENEGACIONES_CONTACTO = Object.freeze({ 401: "autenticacion_requerida", 403: "acceso_denegado", 404: "no_encontrada" });
const ACCIONES = Object.freeze({
  actualizar_contacto: ["PUT", "/api/vec/personas/mi-perfil/contacto"],
  incorporar_merito: ["POST", "/api/vec/bolsa/mi-expediente/meritos"],
  guardar_borrador: ["PUT", "/api/vec/bolsa/mis-solicitudes/borrador"],
  calcular_autobaremo: ["POST", "/api/vec/bolsa/mis-solicitudes/autobaremo"],
  iniciar_pago: ["POST", "/api/vec/bolsa/mis-solicitudes/pago"],
  firmar_solicitud: ["POST", "/api/vec/bolsa/mis-solicitudes/firma"],
  registrar_solicitud: ["POST", "/api/vec/bolsa/mis-solicitudes/registro"],
  cambiar_disponibilidad: ["POST", "/api/vec/bolsa/mi-disponibilidad"],
  responder_llamamiento: ["POST", "/api/vec/bolsa/mis-llamamientos/respuesta"],
  presentar_subsanacion: ["POST", "/api/vec/bolsa/mis-subsanaciones"],
  presentar_alegacion: ["POST", "/api/vec/bolsa/mis-alegaciones"],
  marcar_mensaje: ["POST", "/api/vec/bolsa/mis-mensajes/lectura"],
  actualizar_notificaciones: ["PUT", "/api/vec/personas/mis-preferencias/notificaciones"],
  solicitar_certificado: ["POST", "/api/vec/bolsa/mis-certificados"],
  solicitar_descarga: ["POST", "/api/vec/bolsa/mis-documentos/descarga"],
});

export class ErrorClienteAreaPersonal extends Error {
  constructor(codigo, mensaje, causa) {
    super(mensaje, causa ? { cause: causa } : undefined);
    this.name = "ErrorClienteAreaPersonal";
    this.codigo = codigo;
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
        method: "POST", credentials: "omit", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer",
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

function exigirEnvelope(valor, nombre) {
  if (!valor || typeof valor !== "object" || Array.isArray(valor)
    || !valor.data || typeof valor.data !== "object" || Array.isArray(valor.data)) {
    throw new ErrorClienteAreaPersonal("respuesta_incompatible", `${nombre} no contiene el envelope data esperado.`);
  }
  return valor.data;
}

async function leerJSONAcotado(respuesta) {
  const tipo = respuesta.headers?.get?.("Content-Type") || "";
  if (!/^application\/json(?:\s*;\s*charset=utf-8)?$/i.test(tipo)) {
    throw new ErrorClienteAreaPersonal("tipo_respuesta", "El servicio no devolvió JSON UTF-8.");
  }
  const declarada = respuesta.headers?.get?.("Content-Length");
  if (declarada !== null && declarada !== undefined && declarada !== "") {
    if (!/^(?:0|[1-9][0-9]*)$/.test(declarada) || Number(declarada) > MAXIMO_JSON_BYTES) {
      throw new ErrorClienteAreaPersonal("respuesta_excesiva", "La respuesta supera el límite permitido.");
    }
  }
  const texto = await respuesta.text();
  if (new TextEncoder().encode(texto).byteLength > MAXIMO_JSON_BYTES) {
    throw new ErrorClienteAreaPersonal("respuesta_excesiva", "La respuesta supera el límite permitido.");
  }
  try {
    return JSON.parse(texto);
  } catch (error) {
    throw new ErrorClienteAreaPersonal("json_invalido", "El servicio devolvió JSON no válido.", error);
  }
}

function nuevaIdempotencia() {
  if (typeof globalThis.crypto?.randomUUID !== "function") {
    throw new ErrorClienteAreaPersonal("idempotencia_no_disponible", "No se puede garantizar la idempotencia de la operación.");
  }
  return `WEB-${globalThis.crypto.randomUUID()}`;
}

function contieneDescriptorFichero(valor) {
  if (!valor || typeof valor !== "object") return false;
  if (Array.isArray(valor)) return valor.some(contieneDescriptorFichero);
  if (typeof valor.nombre === "string" && typeof valor.tipo === "string" && Number.isFinite(valor.tamano)) return true;
  return Object.values(valor).some(contieneDescriptorFichero);
}

function serializarSolicitudAcotada(valor) {
  const texto = JSON.stringify(valor);
  if (new TextEncoder().encode(texto).byteLength > MAXIMO_SOLICITUD_BYTES) {
    throw new ErrorClienteAreaPersonal("solicitud_excesiva", "La solicitud supera el límite permitido.");
  }
  return texto;
}

function mensajeHTTP(estado) {
  if (estado === 401) return "La identificación ha caducado o no está disponible.";
  if (estado === 403) return "La sesión no dispone de permiso para esta operación.";
  if (estado === 409) return "El expediente cambió. Recargue la información antes de continuar.";
  if (estado === 422) return "La operación no supera las validaciones del expediente.";
  if (estado === 429) return "El servicio está ocupado. Espere antes de reintentar.";
  return `El servicio no pudo completar la operación (HTTP ${estado}).`;
}

export function crearClienteHTTPAreaPersonal({ fetchImpl = globalThis.fetch } = {}) {
  if (typeof fetchImpl !== "function") {
    return Object.freeze({
      modo: "http",
      cargar: async () => { throw new ErrorClienteAreaPersonal("transporte_no_disponible", "El cliente HTTP no está disponible."); },
      ejecutar: async () => { throw new ErrorClienteAreaPersonal("transporte_no_disponible", "El cliente HTTP no está disponible."); },
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
        credentials: "omit",
      });
    } catch (error) {
      throw new ErrorClienteAreaPersonal("servicio_no_disponible", "No se pudo establecer una conexión segura con el servicio.", error);
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
      throw new ErrorClienteAreaPersonal("respuesta_incompatible", "El estado del contacto no es válido.");
    }
    let recibo = null;
    if (estado.encontrado) {
      const recibido = await solicitar(RUTA_RECIBO_CONTACTO_PROPIO, {
        method: "POST", headers: { Accept: "application/json", "Content-Type": "application/json" },
        body: JSON.stringify({ version: estado.version }), signal,
      }, [200]);
      if (!recibido || typeof recibido.recibo_ref !== "string" || !recibido.recibo_ref
        || recibido.version !== estado.version) throw new ErrorClienteAreaPersonal("respuesta_incompatible", "El recibo del contacto no coincide.");
      recibo = Object.freeze({ reciboRef: recibido.recibo_ref, version: recibido.version });
    }
    return Object.freeze({ autorizacion: Object.freeze({ capacidad: true, version: estado.version, consultarRecibo: estado.encontrado }), recibo });
  }
  async function cargar() {
    const envelope = await solicitar(RUTA_MI_BOLSA, {
      method: "GET",
      credentials: "omit",
      headers: { Accept: "application/json" },
    }, [200]);
    return Object.freeze({ fuente: "real", consulta: validarRespuestaMiBolsa(envelope) });
  }

  async function ejecutar({ accion, payload = {}, confirmacion = false, capacidad = false } = {}) {
    if (capacidad !== true) {
      throw new ErrorClienteAreaPersonal("capacidad_denegada", "El servidor no ha concedido capacidad para esta acción.");
    }
    if (confirmacion !== true) {
      throw new ErrorClienteAreaPersonal("confirmacion_ausente", "La acción requiere confirmación explícita.");
    }
    const definicion = ACCIONES[accion];
    if (!definicion) throw new ErrorClienteAreaPersonal("accion_no_admitida", "La acción solicitada no está admitida por este cliente.");
    if (contieneDescriptorFichero(payload)) {
      throw new ErrorClienteAreaPersonal("carga_documental_no_compuesta", "La carga documental segura todavía no está conectada. No se ha enviado el fichero.");
    }
    const [metodo, ruta] = definicion;
    const envelope = await solicitar(ruta, {
      method: metodo,
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
        "X-Idempotency-Key": nuevaIdempotencia(),
      },
      body: serializarSolicitudAcotada({
        data: {
          esquema: "vec.bolsa.area-personal.accion.v1",
          accion,
          confirmacion: true,
          payload,
        },
      }),
    }, [200, 201]);
    const datos = exigirEnvelope(envelope, "La confirmación de la operación");
    return Object.freeze({
      recibo: validarRecibo(datos.recibo, { presentacionEsperada: false }),
      datos: datos.resultado && typeof datos.resultado === "object" ? structuredClone(datos.resultado) : null,
    });
  }

  return Object.freeze({ modo: "http", cargar, cargarContactoPropio, ejecutar });
}

export const RUTAS_AREA_PERSONAL = Object.freeze({ miBolsa: RUTA_MI_BOLSA, acciones: ACCIONES });
