import {
  extraerDatosEnvelopeLlamamiento,
  validarConfirmacionPropuestaLlamamiento,
  validarReferenciaOpacaLlamamiento,
} from "./portal-llamamientos-contrato.js?v=20260718-llamamientos-v1";

const RUTA_PROPUESTAS_LLAMAMIENTO = "/api/vec/bolsa/propuestas-llamamiento";
const MAXIMO_RESPUESTA_BYTES = 8 * 1024;
const MAXIMO_FRAGMENTOS_RESPUESTA = 256;

function longitudDeclarada(respuesta) {
  const valor = respuesta.headers?.get?.("Content-Length");
  if (typeof valor !== "string" || !/^(?:0|[1-9][0-9]*)$/.test(valor)) {
    throw new Error("La respuesta no incluye un Content-Length canónico.");
  }
  const longitud = Number(valor);
  if (!Number.isSafeInteger(longitud) || longitud > MAXIMO_RESPUESTA_BYTES) {
    throw new Error("La respuesta de propuesta supera el límite de 8 KiB.");
  }
  if (longitud === 0) throw new Error("La respuesta JSON está vacía.");
  return longitud;
}

async function cancelarRespuesta(respuesta, lector = null, motivo = "respuesta rechazada") {
  const cancelar = lector !== null && typeof lector.cancel === "function"
    ? () => lector.cancel(motivo)
    : (respuesta?.body && typeof respuesta.body.cancel === "function"
      ? () => respuesta.body.cancel(motivo)
      : null);
  if (cancelar === null) return;
  try { await cancelar(); } catch { /* la cancelación nunca oculta el error causal */ }
}

async function leerTextoExactoAcotado(respuesta) {
  const longitud = longitudDeclarada(respuesta);
  if (!respuesta.body || typeof respuesta.body.getReader !== "function") {
    throw new Error("La respuesta no permite una lectura incremental acotada.");
  }
  const lector = respuesta.body.getReader();
  if (!lector || typeof lector.read !== "function" || typeof lector.cancel !== "function"
    || typeof lector.releaseLock !== "function") {
    throw new Error("El lector de respuesta no respeta el contrato Fetch requerido.");
  }

  const bytes = new Uint8Array(longitud);
  let total = 0;
  let fragmentos = 0;
  try {
    while (true) {
      const lectura = await lector.read();
      if (!lectura || typeof lectura.done !== "boolean") {
        throw new Error("El lector devolvió un estado no válido.");
      }
      if (lectura.done) break;
      fragmentos += 1;
      if (fragmentos > MAXIMO_FRAGMENTOS_RESPUESTA) {
        throw new Error("El flujo de respuesta contiene demasiados fragmentos.");
      }
      if (!(lectura.value instanceof Uint8Array)) {
        throw new Error("El flujo de respuesta no contiene bytes válidos.");
      }
      if (total + lectura.value.byteLength > longitud
        || total + lectura.value.byteLength > MAXIMO_RESPUESTA_BYTES) {
        throw new Error("El cuerpo recibido no coincide con Content-Length.");
      }
      bytes.set(lectura.value, total);
      total += lectura.value.byteLength;
    }
    if (total !== longitud) {
      throw new Error("El cuerpo recibido no coincide con Content-Length.");
    }
    try {
      return new TextDecoder("utf-8", { fatal: true }).decode(bytes);
    } catch (error) {
      throw new Error("La respuesta no contiene UTF-8 válido.", { cause: error });
    }
  } catch (error) {
    await cancelarRespuesta(respuesta, lector, "lectura rechazada");
    throw error;
  } finally {
    try { lector.releaseLock(); } catch { /* lector cancelado o no liberable */ }
  }
}

function analizarJSONExacto(texto) {
  let envelope;
  try {
    envelope = JSON.parse(texto);
  } catch (error) {
    throw new Error("La respuesta contiene JSON no válido.", { cause: error });
  }
  let recodificado;
  try {
    recodificado = JSON.stringify(envelope);
  } catch (error) {
    throw new Error("La respuesta contiene JSON no representable.", { cause: error });
  }
  if (recodificado !== texto) {
    throw new Error("La respuesta no contiene la representación JSON exacta esperada.");
  }
  return envelope;
}

export function crearClientePropuestasLlamamiento({ fetchImpl = globalThis.fetch } = {}) {
  async function solicitar({ necesidadId, capacidad }) {
    if (capacidad !== true) {
      return {
        ok: false,
        bloqueada: true,
        mensaje: "El servidor no ha concedido la capacidad para solicitar propuestas.",
      };
    }
    if (typeof fetchImpl !== "function") {
      return { ok: false, bloqueada: true, mensaje: "El servicio de propuestas no está disponible." };
    }

    let necesidad;
    try {
      necesidad = validarReferenciaOpacaLlamamiento(necesidadId, "referencia de necesidad");
    } catch (error) {
      return { ok: false, bloqueada: true, mensaje: error instanceof Error ? error.message : "Necesidad no válida." };
    }

    let respuesta;
    try {
      respuesta = await fetchImpl(RUTA_PROPUESTAS_LLAMAMIENTO, {
        method: "POST",
        credentials: "omit",
        headers: {
          Accept: "application/json",
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          data: {
            esquema: "vec.bolsa.propuesta-llamamiento.solicitud.v1",
            necesidad_id: necesidad,
          },
        }),
      });
      if (respuesta?.status !== 201) {
        const estado = Number.isInteger(respuesta?.status) ? ` (HTTP ${respuesta.status})` : "";
        throw new Error(`No se pudo obtener la confirmación de propuesta${estado}.`);
      }
      const tipo = respuesta.headers?.get?.("Content-Type") || "";
      if (!/^application\/json(?:;\s*charset=utf-8)?$/i.test(tipo)) {
        throw new Error("La respuesta de propuesta no es JSON canónico.");
      }
      const etag = respuesta.headers?.get?.("ETag");
      if (typeof etag !== "string" || etag === "") {
        throw new Error("La respuesta no incluye el ETag obligatorio.");
      }
      const texto = await leerTextoExactoAcotado(respuesta);
      const datos = extraerDatosEnvelopeLlamamiento(analizarJSONExacto(texto));
      const confirmacion = validarConfirmacionPropuestaLlamamiento(datos, etag);
      if (confirmacion.necesidad.referencia !== necesidad) {
        throw new Error("La confirmación no corresponde a la necesidad solicitada.");
      }
      return { ok: true, confirmacion, etag };
    } catch (error) {
      await cancelarRespuesta(respuesta);
      return {
        ok: false,
        bloqueada: true,
        mensaje: error instanceof Error ? error.message : "No se pudo obtener la confirmación de propuesta.",
      };
    }
  }

  return Object.freeze({ solicitar });
}
