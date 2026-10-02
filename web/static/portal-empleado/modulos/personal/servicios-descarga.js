import { cargarTextos } from "../../../comun/textos.js";
import { esFechaCorteServicios } from "./ficha-propia-corte.js?v=20261002-personal-servicios-csv-v1";

const textos = await cargarTextos("personal-servicios-descarga");
export const traducirDescargaServicios = (clave) => textos.traducir(`general.${clave}`);
const CAMPOS = Object.freeze(["desde", "hasta", "procedencia", "reconocimiento", "estado"]);

// Todas las celdas se entrecomillan. El apóstrofo impide que Excel interprete
// fórmulas tras espacios o controles, incluidos caracteres invisibles Unicode.
function celda(valor) {
  const original = String(valor);
  const seguro = original.replace(/[\p{Cc}\p{Cf}]/gu, " ");
  const texto = /^[\s\p{Cc}\p{Cf}=+@-]/u.test(original) ? `'${seguro}` : seguro;
  return `"${texto.replaceAll('"', '""')}"`;
}

/** Recibe únicamente la proyección de servicios ya validada por la vista. */
export function crearResumenServiciosCSV(resultado, catalogo = textos) {
  if (!resultado || !["disponible", "vacio"].includes(resultado.estado) || !Array.isArray(resultado.items)) throw new TypeError("resumen de servicios no disponible");
  const t = (clave) => catalogo.traducir(`general.${clave}`);
  // Mismo corte civil que la ficha: mediodía UTC y formato del catálogo activo.
  const civil = (valor) => esFechaCorteServicios(valor)
    ? catalogo.fecha(`${valor}T12:00:00Z`, { dateStyle: "medium", timeZone: "UTC" }) : valor;
  const filas = [
    [t("titulo")], [t("alcance")],
    [t("corte"), resultado.fecha_referencia ? civil(resultado.fecha_referencia) : t("corte_no_indicado")],
    [t("fuente"), resultado.fuente], [t("actualizado"), catalogo.fecha(resultado.actualizado_en, { dateStyle: "medium", timeStyle: "short", timeZone: "Europe/Madrid" })],
    [], CAMPOS.map((campo) => t(campo)),
    ...resultado.items.map((item) => CAMPOS.map((campo) => (["desde", "hasta"].includes(campo) ? civil(item[campo]) : item[campo]) || t("no_consta"))),
  ];
  if (resultado.estado === "vacio") filas.push([t("vacio")]);
  return `\uFEFF${filas.map((fila) => fila.map(celda).join(";")).join("\r\n")}\r\n`;
}

export function descargarResumenServicios(d, resultado) {
  const ventana = d.defaultView;
  const blob = new ventana.Blob([crearResumenServiciosCSV(resultado)], { type: "text/csv;charset=utf-8" });
  const url = ventana.URL.createObjectURL(blob);
  const enlace = d.createElement("a");
  try {
    enlace.href = url; enlace.download = traducirDescargaServicios("archivo");
    enlace.hidden = true; d.body.append(enlace); enlace.click();
  } finally {
    enlace.remove(); ventana.URL.revokeObjectURL(url);
  }
}
