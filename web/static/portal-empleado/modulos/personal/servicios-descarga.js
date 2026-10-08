import { MAXIMO_EXPORTACION_SERVICIOS, MIME_EXPORTACION_SERVICIOS } from "./cliente-http-exportacion-servicios.js?v=20261004-personal-historia-v1";

/** Descarga los bytes comprobados del servidor, sin volver a crear el CSV. */
export function descargarResumenServicios(d, exportacion) {
  if (!(exportacion?.bytes instanceof Uint8Array) || exportacion.bytes.byteLength < 1 ||
      exportacion.bytes.byteLength > MAXIMO_EXPORTACION_SERVICIOS || exportacion.mime !== MIME_EXPORTACION_SERVICIOS ||
      !/^[a-zA-Z0-9_-]+\.csv$/u.test(exportacion.nombre || "") || !/^[0-9a-f]{64}$/u.test(exportacion.huella || "")) throw new TypeError("exportacion_no_valida");
  const ventana = d.defaultView;
  const blob = new ventana.Blob([exportacion.bytes], { type: exportacion.mime });
  const url = ventana.URL.createObjectURL(blob);
  const enlace = d.createElement("a");
  try {
    enlace.href = url; enlace.download = exportacion.nombre; enlace.hidden = true; d.body.append(enlace); enlace.click();
  } finally { enlace.remove(); ventana.URL.revokeObjectURL(url); }
}
