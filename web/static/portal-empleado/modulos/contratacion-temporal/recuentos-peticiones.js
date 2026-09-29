/**
 * Recuentos de peticiones de personal temporal que comparten la portada y la
 * lista, para que las dos pantallas digan siempre los mismos números.
 *
 * Trabaja sobre los resúmenes que ya devolvió la consulta autorizada del
 * cuadro (fase, estado y plazo de la fase calculado por el servidor). No
 * deduce responsables ni tareas: un expediente «pendiente» es el que tiene el
 * plazo de su fase vencido o que vence hoy, o una incidencia abierta.
 */
import { FASES_RRHH, faseRRHH } from "./i18n-fases-rrhh.js";

const TERMINADOS = new Set(["completado", "cancelado"]);
const PATRON_DIA = /^\d{4}-\d{2}-\d{2}$/u;

/** El expediente sigue en trámite (ni terminado ni cancelado). */
export function enTramite(expediente) {
  return !TERMINADOS.has(expediente?.estado_clave);
}

/** Plazo de la fase vencido o que vence hoy, o incidencia abierta. */
export function requiereAtencion(expediente) {
  return enTramite(expediente) && (expediente.plazo_estado === "vencido"
    || expediente.plazo_estado === "vence_hoy" || expediente.estado_clave === "incidencia");
}

function diaPlazo(expediente) {
  return PATRON_DIA.test(expediente?.plazo_ultimo_dia ?? "") ? expediente.plazo_ultimo_dia : "";
}

/** Plazo más próximo primero; sin plazo, al final; a igualdad, por número. */
export function compararPorPlazo(a, b) {
  const [pa, pb] = [diaPlazo(a), diaPlazo(b)];
  if (pa !== pb) {
    if (!pa) return 1;
    if (!pb) return -1;
    return pa < pb ? -1 : 1;
  }
  return String(a?.numero_visible ?? "").localeCompare(String(b?.numero_visible ?? ""), "es");
}

/** Días entre dos fechas civiles AAAA-MM-DD (b − a). */
function diasEntre(a, b) {
  return Math.round((Date.parse(`${b}T00:00:00Z`) - Date.parse(`${a}T00:00:00Z`)) / 86_400_000);
}

/**
 * Resumen único de la consulta: cuántas siguen en trámite, cuáles requieren
 * atención (ordenadas por plazo), cuántos plazos vencen en los próximos siete
 * días desde la fecha de la consulta y cuántas hay en cada una de las ocho
 * fases. `parcial` indica que la consulta tiene más páginas sin cargar.
 */
export function resumirPeticiones({ expedientes = [], parcial = false, generadoEn = "" } = {}) {
  const lista = Array.isArray(expedientes) ? expedientes : [];
  const vivos = lista.filter(enTramite);
  const hoy = /^\d{4}-\d{2}-\d{2}/u.exec(String(generadoEn ?? ""))?.[0] ?? "";
  const porFase = new Map(FASES_RRHH.map((fase) => [fase, 0]));
  for (const expediente of vivos) {
    const fase = faseRRHH(expediente.fase_clave);
    if (fase) porFase.set(fase.clave, porFase.get(fase.clave) + 1);
  }
  const vencenSemana = hoy === "" ? null : vivos.filter((expediente) => {
    const dia = diaPlazo(expediente);
    if (!dia) return false;
    const dias = diasEntre(hoy, dia);
    return dias >= 0 && dias <= 6;
  }).length;
  return Object.freeze({
    parcial: parcial === true,
    total: lista.length,
    enTramite: vivos.length,
    atencion: Object.freeze(vivos.filter(requiereAtencion).sort(compararPorPlazo)),
    vencenSemana,
    porFase: Object.freeze(Object.fromEntries(porFase)),
  });
}
