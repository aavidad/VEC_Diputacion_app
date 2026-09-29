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

/** Filtros de la lista que se aplican en pantalla sobre la consulta ya cargada. */
export const FILTRO_LISTA_INICIAL = Object.freeze({ texto: "", fase: "", centro: "", categoria: "", mostrar: "en_tramite" });
export const OPCIONES_MOSTRAR = Object.freeze(["en_tramite", "atencion", "vencen_semana", "espera", "terminadas", "todas"]);

/** Normaliza un filtro recibido (de la portada o del formulario) sin aceptar claves ajenas. */
export function filtroListaValido(entrada = {}) {
  const filtro = { ...FILTRO_LISTA_INICIAL };
  for (const clave of Object.keys(filtro)) {
    const valor = entrada?.[clave];
    if (typeof valor === "string" && valor.length <= 120) filtro[clave] = valor.trim();
  }
  if (filtro.fase && !FASES_RRHH.includes(filtro.fase)) filtro.fase = "";
  if (!OPCIONES_MOSTRAR.includes(filtro.mostrar)) filtro.mostrar = FILTRO_LISTA_INICIAL.mostrar;
  return Object.freeze(filtro);
}

function normalizar(texto) {
  return String(texto ?? "").normalize("NFD").replace(/\p{Diacritic}/gu, "").toLocaleLowerCase("es-ES");
}

function cumpleMostrar(expediente, mostrar, hoy) {
  if (mostrar === "todas") return true;
  if (mostrar === "terminadas") return !enTramite(expediente);
  if (!enTramite(expediente)) return false;
  if (mostrar === "atencion") return requiereAtencion(expediente);
  if (mostrar === "espera") return expediente.estado_clave === "espera";
  if (mostrar === "vencen_semana") {
    const dia = diaPlazo(expediente);
    if (!dia || !hoy) return false;
    const dias = diasEntre(hoy, dia);
    return dias >= 0 && dias <= 6;
  }
  return true;
}

/** Aplica los filtros de pantalla y ordena por plazo (lo más urgente, arriba). */
export function filtrarPeticiones(expedientes, filtro = FILTRO_LISTA_INICIAL, generadoEn = "") {
  const hoy = /^\d{4}-\d{2}-\d{2}/u.exec(String(generadoEn ?? ""))?.[0] ?? "";
  const texto = normalizar(filtro.texto);
  return (Array.isArray(expedientes) ? expedientes : []).filter((expediente) => (
    cumpleMostrar(expediente, filtro.mostrar, hoy)
    && (!filtro.fase || faseRRHH(expediente.fase_clave)?.clave === filtro.fase)
    && (!filtro.centro || expediente.centro === filtro.centro)
    && (!filtro.categoria || expediente.categoria === filtro.categoria)
    && (!texto || [expediente.numero_visible, expediente.centro, expediente.categoria]
      .some((valor) => normalizar(valor).includes(texto)))
  )).sort(compararPorPlazo);
}
