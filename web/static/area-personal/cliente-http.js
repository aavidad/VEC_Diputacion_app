import { validarDatosAreaPersonal, validarRecibo } from "./contrato.js";

const RUTA_PANEL = "/api/vec/bolsa/area-personal";
const MAXIMO_JSON_BYTES = 512 * 1024;
const MAXIMO_SOLICITUD_BYTES = 64 * 1024;
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
        credentials: "omit",
        cache: "no-store",
        redirect: "error",
        referrerPolicy: "no-referrer",
        ...opciones,
      });
    } catch (error) {
      throw new ErrorClienteAreaPersonal("servicio_no_disponible", "No se pudo establecer una conexión segura con el servicio.", error);
    }
    if (!estadosValidos.includes(respuesta?.status)) {
      throw new ErrorClienteAreaPersonal("operacion_rechazada", mensajeHTTP(respuesta?.status));
    }
    return leerJSONAcotado(respuesta);
  }

  async function cargar() {
    const envelope = await solicitar(RUTA_PANEL, {
      method: "GET",
      headers: { Accept: "application/json" },
    }, [200]);
    return validarDatosAreaPersonal(exigirEnvelope(envelope, "La respuesta del área personal"), {
      presentacionEsperada: false,
    });
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

  return Object.freeze({ modo: "http", cargar, ejecutar });
}

export const RUTAS_AREA_PERSONAL = Object.freeze({ panel: RUTA_PANEL, acciones: ACCIONES });
