/**
 * Recuentos de peticiones de personal temporal que comparten la portada y la
 * lista, para que las dos pantallas digan siempre los mismos números.
 *
 * Trabaja sobre los resúmenes que ya devolvió la consulta autorizada del
 * cuadro (fase, estado y plazo de la fase calculado por el servidor). No
 * deduce responsables ni tareas: un expediente «pendiente» es el que tiene el
 * plazo de su fase vencido o que vence hoy, o una incidencia abierta.
 */
import { FASES_RRHH, FASE_RRHH_DE_ORIGEN, faseRRHH } from "./fases-rrhh-datos.js?v=20261007-pantallas-textos-final-v1";

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

/** Solo plazos vencidos de expedientes que siguen en trámite. */
export function tienePlazoVencido(expediente) {
  return enTramite(expediente) && expediente.plazo_estado === "vencido";
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
export function diasEntre(a, b) {
  return Math.round((Date.parse(`${b}T00:00:00Z`) - Date.parse(`${a}T00:00:00Z`)) / 86_400_000);
}

/** Día civil de la lectura en la sede; evita contar el UTC anterior tras la medianoche. */
export function diaConsulta(generadoEn) {
  const fecha = new Date(generadoEn ?? "");
  if (!Number.isFinite(fecha.getTime())) return "";
  const partes = Object.fromEntries(new Intl.DateTimeFormat(undefined, {
    timeZone: "Europe/Madrid", calendar: "gregory", numberingSystem: "latn",
    year: "numeric", month: "2-digit", day: "2-digit",
  }).formatToParts(fecha).map(({ type, value }) => [type, value]));
  return `${partes.year}-${partes.month}-${partes.day}`;
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
  const hoy = diaConsulta(generadoEn);
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
    vencidos: vivos.filter(tienePlazoVencido).length,
    atencion: Object.freeze(vivos.filter(requiereAtencion).sort(compararPorPlazo)),
    vencenSemana,
    porFase: Object.freeze(Object.fromEntries(porFase)),
  });
}

/** Filtros de la lista que se aplican en pantalla sobre la consulta ya cargada. */
export const FILTRO_LISTA_INICIAL = Object.freeze({ texto: "", fase: "", centro: "", categoria: "", mostrar: "en_tramite" });
export const OPCIONES_MOSTRAR = Object.freeze(["en_tramite", "vencidos", "vence_hoy", "incidencia", "sin_plazo", "atencion",
  "vencen_semana", "espera", "terminadas", "todas"]);

/** Normaliza un filtro recibido (de la portada o del formulario) sin aceptar claves ajenas. */
export function filtroListaValido(entrada = {}) {
  const filtro = { ...FILTRO_LISTA_INICIAL };
  for (const clave of Object.keys(filtro)) {
    const valor = entrada?.[clave];
    const maximo = clave === "texto" ? 80 : (["centro", "categoria"].includes(clave) ? 160 : 120);
    if (typeof valor === "string" && valor.length <= maximo) filtro[clave] = valor.trim();
  }
  if (filtro.fase && !FASES_RRHH.includes(filtro.fase)) filtro.fase = "";
  if (!OPCIONES_MOSTRAR.includes(filtro.mostrar)) filtro.mostrar = FILTRO_LISTA_INICIAL.mostrar;
  return Object.freeze(filtro);
}

const CAMPOS_URL_CT = Object.freeze({
  texto: "ct_texto", fase: "ct_fase", centro: "ct_centro",
  categoria: "ct_categoria", mostrar: "ct_mostrar",
});
const ESTADOS_VIVOS = Object.freeze(["pendiente", "en_curso", "espera_externa", "incidencia"]);
const PATRON_REFERENCIA_FILTRO = /^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$/u;
const PATRON_TEXTO_FILTRO = /^[0-9A-Za-zÁÉÍÓÚÜÑáéíóúüñ/._ -]{0,80}$/u;
const PATRON_FASE_ADMINISTRATIVA = /^[a-z][a-z0-9._-]{1,79}$/u;

function fasesExactasV2(faseVisual, fases) {
  if (!Array.isArray(fases) || fases.length > 32 || new Set(fases).size !== fases.length
    || fases.some((clave) => typeof clave !== "string" || !PATRON_FASE_ADMINISTRATIVA.test(clave)
      || FASE_RRHH_DE_ORIGEN[clave] !== faseVisual)
    || Boolean(faseVisual) !== Boolean(fases.length)) {
    throw new TypeError("fases administrativas CT no válidas");
  }
  return Object.freeze([...fases].sort());
}

function filtroV2Estricto(entrada) {
  if (!entrada || typeof entrada !== "object" || Array.isArray(entrada)
    || Object.keys(entrada).some((clave) => !Object.hasOwn(FILTRO_LISTA_INICIAL, clave))) {
    throw new TypeError("filtro de lista CT no válido");
  }
  const validado = filtroListaValido(entrada);
  if (Object.entries(entrada).some(([clave, valor]) => typeof valor !== "string"
    || valor !== validado[clave]) || !PATRON_TEXTO_FILTRO.test(validado.texto)
    || (validado.centro && !PATRON_REFERENCIA_FILTRO.test(validado.centro))
    || (validado.categoria && !PATRON_REFERENCIA_FILTRO.test(validado.categoria))) {
    throw new TypeError("filtro de lista CT no válido");
  }
  return validado;
}

/** Solo parámetros de la lista; conserva los del portal y rechaza duplicados. */
export function leerFiltroListaV2DesdeURL(parametros) {
  if (!(parametros instanceof URLSearchParams)) throw new TypeError("URL de lista CT no válida");
  if ([...parametros.keys()].some((clave) => clave.startsWith("ct_")
    && !Object.values(CAMPOS_URL_CT).includes(clave) && clave !== "ct_fases")) {
    throw new TypeError("URL de lista CT no válida");
  }
  const entrada = {};
  for (const [campo, clave] of Object.entries(CAMPOS_URL_CT)) {
    const valores = parametros.getAll(clave);
    if (valores.length > 1) throw new TypeError("URL de lista CT duplicada");
    if (valores.length === 1) entrada[campo] = valores[0];
  }
  const fasesClave = parametros.getAll("ct_fases");
  if (!Object.keys(entrada).length && !fasesClave.length) return null;
  const filtro = filtroV2Estricto(entrada);
  return Object.freeze({ filtro, fasesClave: fasesExactasV2(filtro.fase, fasesClave) });
}

/** Produce la URL canónica que el shell puede guardar en history. */
export function escribirFiltroListaV2EnURL(parametros, entrada, fasesClave = []) {
  if (!(parametros instanceof URLSearchParams)) throw new TypeError("URL de lista CT no válida");
  const filtro = filtroV2Estricto(entrada);
  const fases = fasesExactasV2(filtro.fase, fasesClave);
  const salida = new URLSearchParams(parametros);
  for (const clave of Object.values(CAMPOS_URL_CT)) salida.delete(clave);
  salida.delete("ct_fases");
  for (const [campo, clave] of Object.entries(CAMPOS_URL_CT)) {
    if (filtro[campo] !== FILTRO_LISTA_INICIAL[campo]) salida.set(clave, filtro[campo]);
  }
  for (const fase of fases) salida.append("ct_fases", fase);
  return salida;
}

/** Traduce los controles visibles al contrato de consulta v2, sin filtrar una página. */
export function filtrosCuadroRRHHV2(entrada, porFaseAdministrativa = {}, fasesClave = null) {
  const filtro = filtroV2Estricto(entrada);
  if (!porFaseAdministrativa || typeof porFaseAdministrativa !== "object"
    || Array.isArray(porFaseAdministrativa)) throw new TypeError("fases CT no disponibles");
  const estados = {
    en_tramite: ESTADOS_VIVOS, incidencia: ["incidencia"], espera: ["espera_externa"],
    terminadas: ["completado", "cancelado"], todas: [],
  }[filtro.mostrar];
  if (!estados) {
    throw Object.assign(new Error("filtro CT sin fuente completa"),
      { codigo: "filtro_servidor_no_disponible" });
  }
  if (filtro.fase && ["terminadas", "todas"].includes(filtro.mostrar)) {
    throw Object.assign(new Error("fase sin definición completa para todos los estados"),
      { codigo: "filtro_servidor_no_disponible" });
  }
  const fases = fasesClave !== null ? fasesExactasV2(filtro.fase, fasesClave) : filtro.fase
    ? Object.keys(porFaseAdministrativa).filter((clave) => Number.isSafeInteger(porFaseAdministrativa[clave])
      && porFaseAdministrativa[clave] > 0 && FASE_RRHH_DE_ORIGEN[clave] === filtro.fase).sort()
    : [];
  if (fases.length > 32) throw new TypeError("demasiadas fases CT para la consulta");
  return Object.freeze({
    sinCoincidenciasDeFase: Boolean(filtro.fase && fases.length === 0),
    filtros: Object.freeze({ texto: filtro.texto, centro_ref: filtro.centro,
      categoria_ref: filtro.categoria, estados_clave: Object.freeze([...estados]),
      fases_clave: Object.freeze(fases) }),
  });
}

function normalizar(texto) {
  return String(texto ?? "").normalize("NFD").replace(/\p{Diacritic}/gu, "").toLocaleLowerCase("es-ES");
}

function cumpleMostrar(expediente, mostrar, hoy) {
  if (mostrar === "todas") return true;
  if (mostrar === "terminadas") return !enTramite(expediente);
  if (!enTramite(expediente)) return false;
  if (mostrar === "vencidos") return tienePlazoVencido(expediente);
  // Las mismas cifras que la portada: vencen hoy, incidencia abierta y plazo
  // que no se pudo calcular.
  if (mostrar === "vence_hoy") return expediente.plazo_estado === "vence_hoy";
  if (mostrar === "incidencia") return expediente.estado_clave === "incidencia";
  if (mostrar === "sin_plazo") return expediente.plazo_estado === "no_calculado";
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

/** Aplica los filtros y ordena por plazo; la vista puede aportar etiquetas buscables. */
export function filtrarPeticiones(expedientes, filtro = FILTRO_LISTA_INICIAL, generadoEn = "", valoresBusqueda = () => []) {
  const hoy = diaConsulta(generadoEn);
  const texto = normalizar(filtro.texto);
  return (Array.isArray(expedientes) ? expedientes : []).filter((expediente) => (
    cumpleMostrar(expediente, filtro.mostrar, hoy)
    && (!filtro.fase || faseRRHH(expediente.fase_clave)?.clave === filtro.fase)
    && (!filtro.centro || expediente.centro === filtro.centro)
    && (!filtro.categoria || expediente.categoria === filtro.categoria)
    && (!texto || [expediente.numero_visible, expediente.centro, expediente.categoria, ...valoresBusqueda(expediente)]
      .some((valor) => normalizar(valor).includes(texto)))
  )).sort(compararPorPlazo);
}
