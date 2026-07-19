/** Puente exclusivo de presentación entre el puerto documental y el PDF local. */
import { descargarReciboPDFPresentacion } from "./recibo-pdf-presentacion.js";

export function crearDescargadorRecibosPresentacion(entorno = globalThis) {
  return async function descargarRecibo(descriptor) {
    if (!descriptor || typeof descriptor !== "object" || Array.isArray(descriptor)) {
      throw new TypeError("descriptor de recibo no válido");
    }
    const filas = normalizarFilas(descriptor.filas);
    const urlVerificacion = descriptor.comprobacion?.qr_contenido || descriptor.urlVerificacion;
    const referencia = String(descriptor.referencia || "").trim();
    const nombreArchivo = descriptor.nombre_archivo || `recibo-${referencia.toLowerCase()}.pdf`;
    return descargarReciboPDFPresentacion({
      referencia,
      titulo: descriptor.titulo || "Recibo del Portal del Empleado",
      subtitulo: descriptor.subtitulo || "Diputación de Granada · documento de demostración",
      filas,
      urlVerificacion,
      nombreArchivo,
      nota: descriptor.marca || "Documento DEMO. En producción se emitirá desde el expediente firmado y custodiado.",
      textoCertificacion: descriptor.texto_certificacion,
    }, entorno);
  };
}

function normalizarFilas(filas) {
  if (!Array.isArray(filas)) throw new TypeError("filas de recibo no válidas");
  return filas.map((fila) => {
    if (Array.isArray(fila) && fila.length === 2) return [String(fila[0]), String(fila[1])];
    if (fila && typeof fila === "object" && !Array.isArray(fila)
      && Object.keys(fila).length === 2 && Object.hasOwn(fila, "etiqueta") && Object.hasOwn(fila, "valor")) {
      return [String(fila.etiqueta), String(fila.valor)];
    }
    throw new TypeError("fila de recibo no válida");
  });
}
