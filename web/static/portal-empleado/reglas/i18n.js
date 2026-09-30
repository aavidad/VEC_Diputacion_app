/**
 * Textos de la pantalla de reglas vigentes. Viven en los catálogos
 * `textos/<idioma>/reglas.json`; aquí solo se leen.
 *
 * Este catálogo pertenece únicamente a la pantalla de reglas. El enlace del
 * portal usa el catálogo común y no depende de esta carga opcional.
 */
const { cargarTextos } = await import("../../comun/textos.js");
const { IDIOMA_ACTUAL, IDIOMA_POR_DEFECTO, LOCALIZACION_ACTUAL } = await import("../../comun/idioma.js");

/** Idioma de la interfaz, para el atributo `lang` de la página. */
export const IDIOMA_REGLAS = IDIOMA_ACTUAL;

/** Idioma en que el catálogo de reglas escribe sus textos: el idioma por defecto. */
export const IDIOMA_DATOS_REGLAS = IDIOMA_POR_DEFECTO;

// La forma mínima de la pantalla se comprueba antes de crear el traductor: un
// JSON válido con una sección parcial tampoco puede dejar la página a medias.
const CLAVES = Object.freeze(`
  documentTitle miga titulo volver ayudaAbrir ayudaTitulo ayudaCerrar ayudaQue ayudaOrigen ayudaVersion
  filtros filtroModulo filtroOrigen filtroTexto todos origen_reglamento origen_ejemplo cargando sinResultados
  kpiTotal kpiReglamento kpiEjemplo modulo_bolsa modulo_contratacion_temporal catalogoVersion reintentar
  paqueteEjemplo estado_sin_catalogo estado_no_disponible contadorReglas colRegla colValor colUnidad colOrigen
  colDuda colVersion origenArticulo origenEjemplo parteEjemplo sinValor computo_administrativo computo_civil
  unidad_dias_habiles unidad_dias_naturales unidad_meses unidad_anios unidad_horas unidad_minutos_semanales
  unidad_intentos unidad_procesos unidad_franja_horaria unidad_lista unidad_ninguna error_solicitud_invalida
  error_servicio_no_disponible error_autenticacion_requerida error_acceso_denegado error_respuesta ayudaDetalle
  ayudaResolver detalleQue detalleNorma detalleOrigen detalleDuda detalleSinDescripcion
  ajusteRevision ajusteRevisionDetalle detalleEstado
  ajustesTitulo ajustesCargando ajustesNoDisponible ajustesRespuestaInvalida ajustesSinPermiso
  ajustesConflicto ajustesValorInvalido ajustesSinReglas ajustesBaseSinPublicar ajustesBaseInactiva
  ajustesVersion ajustesCambiar
  ajustesSinMotivos ajustesPersonaGenerica ajustesHistorial ajustesSinHistoria ajustesMotivoNoIdentificado
  ajustesGuardado ajustesMasHistoria ajustesCampo_cantidad ajustesCampo_cantidad_urgente
  ajustesCampo_unidad ajustesCampo_computo ajustesMotivo ajustesElegirMotivo
  ajustesMotivo_respuesta_rrhh_duda ajustesMotivo_acuerdo_instruccion
  ajustesMotivo_cambio_normativo ajustesMotivo_correccion_error
  ajustesReferencia ajustesNota ajustesSinDatosPersonales ajustesRevisar ajustesEfecto
  ajustesCancelar ajustesGuardar ajustesGuardando ajustesSinCambios ajustesMotivoRequerido
  ajustesReferenciaHistoria ajustesAuditoria
  ajustesReglaHistorica
  ajustesCorregir
  ajustesSoloLectura
  ajustesConflictoSinLectura
  resultado
`.trim().split(/\s+/u));

const catalogoCompleto = (catalogo) => catalogo && typeof catalogo === "object"
  && CLAVES.every((clave) => typeof catalogo[clave] === "string" && catalogo[clave] !== "")
  && catalogo.parteEjemplo.includes("{texto}");

const TEXTOS_REGLAS = await cargarTextos("reglas").catch(() => null);
export const MENSAJES_REGLAS = (() => {
  try {
    const catalogo = TEXTOS_REGLAS?.seccion("general");
    return catalogoCompleto(catalogo) ? catalogo : null;
  } catch {
    return null;
  }
})();

export function crearTraductorReglas(catalogo = MENSAJES_REGLAS) {
  if (!catalogoCompleto(catalogo)) {
    throw new Error("catálogo i18n de reglas incompleto");
  }
  return (clave, variables = {}) => {
    if (!Object.hasOwn(catalogo, clave)) throw new Error(`clave i18n de reglas desconocida: ${clave}`);
    return catalogo[clave].replace(/\{([a-z_]+)\}/g, (_c, v) => String(variables[v] ?? ""));
  };
}

/** ¿Existe la clave? Para textos que dependen de un valor del servidor. */
export const existeClaveReglas = (clave) => MENSAJES_REGLAS !== null && Object.hasOwn(MENSAJES_REGLAS, clave);

export function formatearNumero(n) {
  return new Intl.NumberFormat(LOCALIZACION_ACTUAL).format(n);
}

/** Minúsculas según el idioma de la interfaz, para buscar sin distinguir mayúsculas. */
export const minusculas = (texto) => String(texto ?? "").toLocaleLowerCase(LOCALIZACION_ACTUAL);

/** Traduce una explicación solo cuando coincide con la fuente catalogada. */
export function textoPresentacionRegla(modulo, regla, campo, {
  presentacion = TEXTOS_REGLAS?.mensajes.presentacion,
  idioma = TEXTOS_REGLAS?.idioma ?? IDIOMA_DATOS_REGLAS,
  faltantes = TEXTOS_REGLAS?.faltantes ?? [],
} = {}) {
  const original = regla?.[campo];
  const reglas = Object.hasOwn(presentacion ?? {}, modulo) ? presentacion[modulo] : null;
  let campos = reglas;
  for (const parte of String(regla?.clave ?? "").split(".")) {
    if (["__proto__", "constructor", "prototype"].includes(parte) || !Object.hasOwn(campos ?? {}, parte)) {
      campos = null;
      break;
    }
    campos = campos[parte];
  }
  const entrada = Object.hasOwn(campos ?? {}, campo) ? campos[campo] : null;
  const rutaTexto = `presentacion.${modulo}.${regla?.clave}.${campo}.texto`;
  if (faltantes.includes(rutaTexto)) return { texto: original, idioma: IDIOMA_DATOS_REGLAS };
  if (typeof original === "string" && entrada?.original === original
    && typeof entrada.texto === "string" && entrada.texto !== "") {
    return { texto: entrada.texto, idioma };
  }
  return { texto: original, idioma: IDIOMA_DATOS_REGLAS };
}
