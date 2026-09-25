import {
  CAPACIDADES_CONTRATACION_TEMPORAL,
  validarCuadroContratacionTemporal,
  validarExpedienteContratacionTemporal,
} from "./contrato-expedientes.js";
import { minutosJornadaCompletaValidos } from "./contrato-analisis.js";
import { validarCatalogosAlta } from "./contrato.js";
import { crearTraductorContratacionTemporal } from "./i18n.js";
import { crearTraductorExpedientesContratacion } from "./i18n-expedientes.js";

const ESTADOS_SERVIDOR_A_VISUAL = new Map([
  ["pendiente", "pendiente"],
  ["en_curso", "en_curso"],
  ["espera_externa", "espera"],
  ["completado", "completado"],
  ["incidencia", "incidencia"],
  ["cancelado", "cancelado"],
]);
const ESTADOS_VISUAL_A_SERVIDOR = new Map(
  [...ESTADOS_SERVIDOR_A_VISUAL].map(([servidor, visual]) => [visual, servidor]),
);
const PATRON_INSTANTE_CIVIL = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2})(?:\.(\d{1,9}))?Z$/u;

const CLAVES_ETIQUETAS_CONOCIDAS = new Map([
  ["pendiente", "fase_pendiente"],
  ["en_curso", "fase_en_curso"],
  ["espera_externa", "etiqueta_estado_espera_externa"],
  ["completado", "fase_completado"],
  ["incidencia", "fase_incidencia"],
  ["cancelado", "fase_cancelado"],
  ["solicitud", "etiqueta_fase_solicitud"],
  ["solicitud_registrada", "etiqueta_fase_solicitud_registrada"],
  ["analisis", "etiqueta_fase_analisis"],
  ["analisis_rrhh", "etiqueta_fase_analisis_rrhh"],
  ["gestion_bolsa", "etiqueta_fase_gestion_bolsa"],
  ["asignacion_unidad", "etiqueta_fase_asignacion_unidad"],
  ["fiscalizacion", "etiqueta_fase_fiscalizacion"],
  ["informe_juridico", "etiqueta_fase_informe_juridico"],
  ["subsanacion_unidad", "etiqueta_fase_subsanacion_unidad"],
  ["obtencion_candidato", "etiqueta_fase_obtencion_candidato"],
  ["llamamiento", "etiqueta_fase_llamamiento"],
  ["nombramiento", "etiqueta_fase_nombramiento"],
  ["incorporacion", "etiqueta_fase_incorporacion"],
  ["seguimiento", "etiqueta_fase_seguimiento"],
  ["bolsa", "etiqueta_modalidad_bolsa"],
  ["sustitucion", "etiqueta_modalidad_sustitucion"],
  ["vacante", "etiqueta_modalidad_vacante"],
  ["programa", "etiqueta_modalidad_programa"],
  ["acumulacion_tareas", "etiqueta_modalidad_acumulacion_tareas"],
  ["interinidad", "etiqueta_modalidad_interinidad"],
  ["relevo", "etiqueta_modalidad_relevo"],
  ["existe_bolsa_vigente", "comprobacion_bolsa_vigente"],
  ["hay_candidaturas_disponibles", "comprobacion_candidaturas_disponibles"],
  ["oferta_sae_disponible", "comprobacion_oferta_sae"],
  ["requiere_nueva_convocatoria", "comprobacion_nueva_convocatoria"],
  ["afirmativa", "comprobacion_resultado_afirmativa"],
  ["negativa", "comprobacion_resultado_negativa"],
  ["no_consta", "comprobacion_resultado_no_consta"],
]);

// La jornada se guarda en diezmilésimas; se muestra en horas y minutos de media
// semanal con su porcentaje. La jornada completa de referencia la sirve el
// servidor (regla c07); si no se conoce, solo se muestra el porcentaje.
function jornadaVisible(diezmilesimas, locale, t, minutosCompleta) {
  const porcentaje = new Intl.NumberFormat(locale, { style: "percent", maximumFractionDigits: 2 })
    .format(diezmilesimas / 10_000);
  if (!minutosJornadaCompletaValidos(minutosCompleta)) return t("cabecera_jornada_porcentaje", { porcentaje });
  const minutos = Math.round(diezmilesimas * minutosCompleta / 10_000);
  if (minutos < 1) return t("cabecera_jornada_menos_minuto", { porcentaje });
  return t("cabecera_jornada_valor", {
    horas: String(Math.floor(minutos / 60)), minutos: String(minutos % 60), porcentaje,
  });
}

function etiqueta(clave, t, alternativa = t("etiqueta_no_consta")) {
  if (typeof clave !== "string" || clave === "") return alternativa;
  const mensaje = CLAVES_ETIQUETAS_CONOCIDAS.get(clave);
  if (mensaje) return t(mensaje);
  const texto = clave.replaceAll("_", " ").replaceAll("-", " ");
  return texto.charAt(0).toLocaleUpperCase("es-ES") + texto.slice(1);
}

function estadoVisual(clave) {
  const visual = ESTADOS_SERVIDOR_A_VISUAL.get(clave);
  if (visual === undefined) throw new TypeError("estado operativo del servidor no válido");
  return visual;
}

function estadoServidor(clave) {
  if (clave === "") return "";
  const servidor = ESTADOS_VISUAL_A_SERVIDOR.get(clave);
  if (servidor === undefined) throw new TypeError("filtro de estado visual no válido");
  return servidor;
}

function campo(clave, titulo, valor) {
  return {
    clave,
    etiqueta: titulo,
    valor: String(valor ?? ""),
    tono: "neutro",
    control: "solo_lectura",
    obligatorio: false,
    opciones: [],
  };
}

function etiquetasCatalogos(obtenerCatalogos) {
  try {
    const { centros, categorias } = validarCatalogosAlta(obtenerCatalogos());
    return {
      centros: new Map(centros.map(({ referencia, etiqueta: texto }) => [referencia, texto])),
      categorias: new Map(categorias.map(({ referencia, etiqueta: texto }) => [referencia, texto])),
    };
  } catch {
    return null;
  }
}

function referenciaVisible(catalogos, tipo, referencia) {
  return catalogos?.[tipo].get(referencia) ?? referencia;
}

// Sin plazo_fase (sin catálogo de reglas o fase sin plazo) la columna queda
// en «—» como antes; la procedencia de la regla no se pinta en la tabla.
function plazoVisual(entrada, locale, t) {
  const plazo = entrada.plazo_fase;
  if (!plazo) return { plazo: "—" };
  if (plazo.estado === "no_calculado") return { plazo: t("plazo_fase_sin_calcular"), plazo_estado: plazo.estado };
  return { plazo: fechaCivil(`${plazo.ultimo_dia}T00:00:00Z`, locale), plazo_estado: plazo.estado };
}

function resumenVisual(entrada, catalogos, t, locale) {
  const estadoClave = estadoVisual(entrada.estado_clave);
  return {
    expediente_ref: entrada.expediente_ref,
    numero_visible: entrada.numero_visible,
    centro: referenciaVisible(catalogos, "centros", entrada.centro_ref),
    categoria: referenciaVisible(catalogos, "categorias", entrada.categoria_ref),
    modalidad: etiqueta(entrada.modalidad_clave, t, "—"),
    estado_clave: estadoClave,
    estado: etiqueta(entrada.estado_clave, t),
    fase_clave: entrada.fase_clave,
    fase_actual: etiqueta(entrada.fase_clave, t),
    fecha_solicitud: entrada.creado_en,
    responsable: "—",
    ...plazoVisual(entrada, locale, t),
    version: entrada.version,
  };
}

function indicadores(expedientes, t) {
  const contar = (clave) => expedientes.filter(
    ({ estado_clave: actual }) => actual === clave,
  ).length;
  return [
    ["total", "indicador_total", expedientes.length, "informacion"],
    ["pendientes", "indicador_pendientes", contar("pendiente"), "aviso"],
    ["en_curso", "indicador_en_curso", contar("en_curso"), "informacion"],
    ["incidencias", "indicador_incidencias", contar("incidencia"), "peligro"],
  ].map(([clave, etiquetaClave, valor, tono]) => ({
    clave, etiqueta: t(etiquetaClave), valor: String(valor), tono,
  }));
}

function proyectarCuadro(pagina, { cursor, numeroPagina, catalogos, t, locale }) {
  const expedientes = pagina.expedientes.map((entrada) => resumenVisual(entrada, catalogos, t, locale));
  return validarCuadroContratacionTemporal({
    esquema: "vec.contratacion_temporal.cuadro.v1",
    demostracion: false,
    generado_en: pagina.generada_en,
    indicadores: indicadores(expedientes, t),
    expedientes,
    paginacion: {
      pagina: numeroPagina,
      cursor_actual: cursor,
      cursor_siguiente: pagina.hay_mas ? pagina.cursor_siguiente : "",
    },
  });
}

function fechaCivil(instante, locale, incluirHora = false) {
  const coincidencia = typeof instante === "string" && PATRON_INSTANTE_CIVIL.exec(instante);
  if (!coincidencia) return instante;
  const [ano, mes, dia, hora, minuto, segundo] = coincidencia.slice(1, 7).map(Number);
  if (hora > 23 || minuto > 59 || segundo > 59) return instante;
  const fraccion = (coincidencia[7] ?? "").padEnd(3, "0").slice(0, 3);
  const fecha = new Date(0);
  fecha.setUTCFullYear(ano, mes - 1, dia);
  fecha.setUTCHours(hora, minuto, segundo, Number(fraccion));
  if (fecha.getUTCFullYear() !== ano || fecha.getUTCMonth() !== mes - 1
    || fecha.getUTCDate() !== dia || fecha.getUTCHours() !== hora
    || fecha.getUTCMinutes() !== minuto || fecha.getUTCSeconds() !== segundo) return instante;
  return new Intl.DateTimeFormat(locale, {
    dateStyle: "medium",
    ...(incluirHora ? { timeStyle: "medium", timeZone: "Europe/Madrid" } : { timeZone: "UTC" }),
  }).format(fecha);
}

const MENSAJES_ACCIONES_HISTORIAL = new Map([
  ["contratacion_temporal.solicitud.crear", "hito_solicitud"],
  ["contratacion_temporal.analisis.registrar", "hito_analisis"],
  ["contratacion_temporal.analisis.rectificar", "hito_rectificacion_analisis"],
  ["contratacion_temporal.cobertura.decidir", "hito_cobertura"],
  ["contratacion_temporal.cobertura.rectificar", "hito_rectificacion_cobertura"],
  ["contratacion_temporal.unidad.asignar", "hito_asignacion"],
  ["contratacion_temporal.unidad.reasignar", "hito_reasignacion"],
  ["contratacion_temporal.informe_juridico.generar", "hito_informe_juridico"],
  ["contratacion_temporal.fiscalizacion.registrar", "hito_fiscalizacion"],
  ["contratacion_temporal.subsanacion_reparos.registrar", "hito_subsanacion_reparo"],
  ["contratacion_temporal.anotacion_administrativa.registrar", "hito_anotacion"],
  ["contratacion_temporal.incorporacion.confirmar", "hito_incorporacion"],
  ["contratacion_temporal.seguimiento.cerrar", "hito_cierre"],
  ["contratacion_temporal.seguimiento.cesar", "hito_cese"],
  ["contratacion_temporal.expediente.cerrar", "hito_cierre_expediente"],
  ["contratacion_temporal.expediente.modificar_tras_nombramiento", "hito_modificacion_nombramiento"],
  ["registrar_solicitud", "hito_solicitud"],
  ["registrar_analisis", "hito_analisis"],
  ["registrar_cobertura", "hito_cobertura"],
  ["registrar_asignacion", "hito_asignacion"],
  ["registrar_informe_juridico", "hito_informe_juridico"],
  ["registrar_fiscalizacion", "hito_fiscalizacion"],
]);

function etiquetaAccionHito(clave, t) {
  const mensaje = MENSAJES_ACCIONES_HISTORIAL.get(clave);
  return mensaje ? t(mensaje) : etiqueta(clave, t);
}

// Etiquetas compartidas por los historiales de expediente e informe.
export function presentarEtiquetasHitoRRHH(hito, mensajes = {}) {
  const t = crearTraductorExpedientesContratacion(mensajes);
  return {
    accion: etiquetaAccionHito(hito.accion_clave, t),
    faseOrigen: etiqueta(hito.fase_origen, t, "—"),
    faseDestino: etiqueta(hito.fase_destino, t),
    estadoOrigen: etiqueta(hito.estado_origen, t),
    estadoDestino: etiqueta(hito.estado_destino, t),
  };
}

// El detalle RRHH ya llega autorizado y validado por el cliente HTTP. Los
// hitos son historia, no fases del flujo de presentación. Conservamos sólo la
// identidad canónica y la versión ya autorizadas por el detalle para enlazar
// acciones que el servidor volverá a autorizar; nunca observaciones, actores
// ni referencias de retorno.
function historialDesdeHitos(hitos, locale, t) {
  return hitos.map((hito) => ({
    secuencia: hito.secuencia,
    fecha: fechaCivil(hito.realizada_en, locale, true),
    fase: etiqueta(hito.fase_destino, t),
    accion: etiquetaAccionHito(hito.accion_clave, t),
    estado_clave: estadoVisual(hito.estado_destino),
    estado: etiqueta(hito.estado_destino, t),
    accion_clave: hito.accion_clave,
    version_expediente: hito.version_expediente,
  }));
}

// El coste estimado se muestra con su origen cuando el análisis lo registró;
// si la fuente no lo calculó, se dice, en vez de inventar una cifra.
function costeEstimadoVisible(analisis, locale, t) {
  const coste = analisis.coste_previsto;
  if (!coste || !Number.isSafeInteger(coste.centimos) || coste.centimos <= 0) return t("coste_sin_calcular");
  const importe = new Intl.NumberFormat(locale, { style: "currency", currency: coste.moneda || "EUR" })
    .format(coste.centimos / 100);
  return analisis.fuente_coste_ref ? t("coste_con_fuente", { importe }) : importe;
}

function cabeceraDetalle(detalle, locale, catalogos, t, minutosCompleta) {
  const { resumen, solicitud } = detalle;
  const campos = [
    campo("centro", t("cabecera_centro"), referenciaVisible(catalogos, "centros", resumen.centro_ref)),
    campo("categoria", t("cabecera_categoria"), referenciaVisible(catalogos, "categorias", resumen.categoria_ref)),
    campo("modalidad", t("cabecera_modalidad"), etiqueta(resumen.modalidad_clave, t)),
    campo("fase", t("cabecera_fase_actual"), etiqueta(resumen.fase_clave, t)),
    campo("estado", t("cabecera_estado"), etiqueta(resumen.estado_clave, t)),
    campo("grupo_subgrupo", t("cabecera_grupo_subgrupo"), solicitud.grupo_subgrupo),
    campo("motivo", t("cabecera_motivo"), etiqueta(solicitud.motivo_clave, t)),
    campo("periodo", t("cabecera_periodo_solicitado"), `${fechaCivil(solicitud.periodo_inicio, locale)} — ${fechaCivil(solicitud.periodo_fin, locale)}`),
  ];
  if (detalle.analisis) {
    campos.push(
      campo("periodo_analizado", t("cabecera_periodo_analizado"), `${fechaCivil(detalle.analisis.periodo_inicio, locale)} — ${fechaCivil(detalle.analisis.periodo_fin, locale)}`),
      campo("causa", t("cabecera_causa_analizada"), etiqueta(detalle.analisis.causa_clave, t)),
      campo("jornada", t("cabecera_jornada"), jornadaVisible(detalle.analisis.porcentaje_jornada, locale, t, minutosCompleta)),
      campo("resultado_rc", t("cabecera_resultado_rc"), etiqueta(detalle.analisis.resultado_rc, t)),
      campo("coste_estimado", t("cabecera_coste_estimado"), costeEstimadoVisible(detalle.analisis, locale, t)),
    );
    if (detalle.analisis.observaciones) {
      campos.push(campo("observaciones", t("observaciones", "Observaciones"), detalle.analisis.observaciones));
    }
  }
  if (detalle.cobertura) {
    campos.push(
      campo("via_cobertura", t("cabecera_via_cobertura"), etiqueta(detalle.cobertura.via_clave, t)),
      campo("decision_gobernada", t("cabecera_decision_gobernada"), detalle.cobertura.decision_gobernada ? t("respuesta_si") : t("respuesta_no")),
    );
    for (const comprobacion of detalle.cobertura.comprobaciones || []) {
      campos.push(campo(
        `comprobacion_${comprobacion.clave}`,
        t("cabecera_comprobacion_bolsa"),
        `${etiqueta(comprobacion.clave, t)}: ${etiqueta(comprobacion.resultado, t)}`,
      ));
    }
  }
  if (detalle.asignacion) {
    campos.push(campo("unidad", t("cabecera_unidad_asignada"), detalle.asignacion.unidad_ref));
  }
  return campos;
}


function versionPropuestaHistorica(detalle) {
  const { resumen, hitos } = detalle;
  if (resumen.version < 8 || resumen.fase_clave !== "nombramiento"
    || resumen.estado_clave !== "en_curso" || !Array.isArray(hitos) || hitos.length !== resumen.version
    || !hitos.every((hito, indice) => hito?.secuencia === indice + 1
      && hito.version_expediente === indice + 1)) return null;
  const tieneAnotacion = hitos.at(-1)?.accion_clave === "contratacion_temporal.anotacion_administrativa.registrar";
  const indicePropuesta = hitos.length - (tieneAnotacion ? 3 : 2);
  if (indicePropuesta < 6) return null;
  const propuesta = hitos[indicePropuesta], resolucion = hitos[indicePropuesta + 1];
  const predecesores = propuesta.accion_clave === "registrar_propuesta_formalizacion"
    && propuesta.fase_destino === "nombramiento" && propuesta.estado_destino === "en_curso"
    && resolucion.accion_clave === "registrar_resolucion_formalizacion"
    && resolucion.fase_origen === "nombramiento" && resolucion.fase_destino === "nombramiento"
    && resolucion.estado_origen === "en_curso" && resolucion.estado_destino === "en_curso";
  if (!predecesores) return null;
  if (!tieneAnotacion) return propuesta.version_expediente;
  const anotacion = hitos.at(-1);
  return anotacion.accion_clave === "contratacion_temporal.anotacion_administrativa.registrar"
    && anotacion.fase_origen === "nombramiento" && anotacion.fase_destino === "nombramiento"
    && anotacion.estado_origen === "en_curso" && anotacion.estado_destino === "en_curso"
    ? propuesta.version_expediente : null;
}

// La propuesta actual se deriva de un hito autorizado, nunca de la versión sola.
function versionPropuestaDocumental(detalle) {
  const historica = versionPropuestaHistorica(detalle);
  if (historica !== null) return historica;
  const { resumen, hitos } = detalle;
  if (resumen.version <= 7 || resumen.fase_clave !== "nombramiento"
    || resumen.estado_clave !== "en_curso" || !Array.isArray(hitos)
    || hitos.length !== resumen.version
    || !hitos.every((h, i) => h.secuencia === i + 1 && h.version_expediente === i + 1)) return null;
  const propuesta = hitos.at(-1);
  return propuesta.accion_clave === "registrar_propuesta_formalizacion"
    && propuesta.fase_destino === "nombramiento" && propuesta.estado_destino === "en_curso"
    ? propuesta.version_expediente : null;
}

// Estados visuales de una fase del raíl (claves del catálogo, no valores secretos).
const ESTADO_FASE_PENDIENTE = "pendiente";
const ESTADO_FASE_COMPLETADO = "completado";

// Equivalencias visuales del procedimiento RRHH; no cambian el flujo administrativo.
const FASE_VISUAL = Object.freeze({
  solicitud: "solicitud", analisis: "analisis_rrhh",
  asignacion: "gestion_bolsa", asignacion_unidad: "gestion_bolsa",
  informe_juridico: "gestion_bolsa", fiscalizacion: "fiscalizacion",
  subsanacion_unidad: "fiscalizacion", llamamiento: "obtencion_candidato",
  nombramiento: "nombramiento", incorporacion: "incorporacion", seguimiento: "seguimiento",
});

// Acciones que cumplen una fase del procedimiento de RRHH sin que el expediente cambie de fase
// administrativa: el análisis se registra dentro de la fase de solicitud.
const ACCIONES_FASES_VISUALES_COMPLETADAS = Object.freeze({
  "contratacion_temporal.analisis.registrar": ["solicitud", "analisis_rrhh"],
  "contratacion_temporal.analisis.rectificar": ["solicitud", "analisis_rrhh"],
  // La propuesta sólo se registra tras la aceptación que el caso de uso ya
  // revalida. Es evidencia de que se obtuvo candidato; no acredita firma,
  // envío, incorporación ni el cumplimiento de las demás fases.
  "registrar_propuesta_formalizacion": ["obtencion_candidato"],
});

function fasesDesdeHitos(detalle, traducir) {
  const presentacion = detalle.presentacion_flujo;
  if (!presentacion) return [];
  const fases = presentacion.fases.map((fase) => ({
    fase_ref: `presentacion:${presentacion.referencia}:${fase.clave}`,
    orden: fase.orden, etiqueta: traducir(fase.clave_i18n), estado_clave: ESTADO_FASE_PENDIENTE,
  }));
  const indice = (clave) => presentacion.fases.findIndex((f) => f.clave === clave);
  for (const hito of detalle.hitos) {
    const origen = indice(FASE_VISUAL[hito.fase_origen]);
    const destino = indice(FASE_VISUAL[hito.fase_destino]);
    if (destino < 0) continue;
    if (origen >= 0 && origen !== destino) {
      if (fases[destino].orden < fases[origen].orden) {
        // Un retorno reabre el recorrido: las fases posteriores ya no están completadas.
        for (const fase of fases) {
          if (fase.orden >= fases[destino].orden) fase.estado_clave = ESTADO_FASE_PENDIENTE;
        }
      } else {
        fases[origen].estado_clave = ESTADO_FASE_COMPLETADO;
      }
    }
    fases[destino].estado_clave = estadoVisual(hito.estado_destino);
    for (const clave of ACCIONES_FASES_VISUALES_COMPLETADAS[hito.accion_clave] ?? []) {
      const cumplida = indice(clave);
      if (cumplida >= 0) fases[cumplida].estado_clave = ESTADO_FASE_COMPLETADO;
    }
  }
  const actual = indice(presentacion.fase_actual || FASE_VISUAL[detalle.resumen.fase_clave]);
  // La fase administrativa actual manda, salvo que una acción ya la haya cumplido (análisis registrado).
  if (actual >= 0 && fases[actual].estado_clave !== ESTADO_FASE_COMPLETADO) {
    fases[actual].estado_clave = estadoVisual(detalle.resumen.estado_clave);
  }
  return fases;
}

function proyectarExpediente(detalle, locale, catalogos, t, mensajes, minutosCompleta) {
  const traducir = crearTraductorContratacionTemporal(mensajes);
  const versionPropuesta = versionPropuestaDocumental(detalle);
  return validarExpedienteContratacionTemporal({
    esquema: "vec.contratacion_temporal.expediente.v1",
    demostracion: false,
    expediente_ref: detalle.resumen.expediente_ref,
    numero_visible: detalle.resumen.numero_visible,
    version: detalle.resumen.version,
    flujo_ref: detalle.resumen.flujo_ref,
    flujo_version: detalle.resumen.flujo_version,
    flujo_huella: detalle.resumen.flujo_huella_sha256,
    cabecera: cabeceraDetalle(detalle, locale, catalogos, t, minutosCompleta),
    ...(detalle.analisis ? { analisis_previo: {
      modalidad_clave: detalle.analisis.modalidad_clave,
      categoria_ref: detalle.analisis.categoria_ref,
      causa_clave: detalle.analisis.causa_clave,
      periodo: { inicio: detalle.analisis.periodo_inicio, fin: detalle.analisis.periodo_fin },
      porcentaje_jornada: detalle.analisis.porcentaje_jornada,
      ...(detalle.analisis.observaciones ? { observaciones: detalle.analisis.observaciones } : {}),
    } } : {}),
    fases: fasesDesdeHitos(detalle, traducir),
    historial: historialDesdeHitos(detalle.hitos, locale, t),
    ...(detalle.fiscalizacion ? { fiscalizacion: {
      resultado_clave: detalle.fiscalizacion.resultado_clave,
      resultado: etiqueta(detalle.fiscalizacion.resultado_clave, t),
      reparos: detalle.fiscalizacion.reparos.map((r) => ({ clave: r.clave, texto: r.texto })),
      registrada_en: fechaCivil(detalle.fiscalizacion.registrada_en, locale),
      ...(detalle.fiscalizacion.subsanacion ? { subsanacion: {
        registrada_en: fechaCivil(detalle.fiscalizacion.subsanacion.registrada_en, locale),
        texto: detalle.fiscalizacion.subsanacion.texto,
      } } : {}),
    } } : {}),
    tareas: [],
    // Sólo selección documental histórica; cada descarga exige autorización vigente.
    ...(versionPropuesta !== null ? { version_propuesta_documental: versionPropuesta } : {}),
  });
}

export function crearAdaptadorHTTPExpedientesContratacionTemporal({
  cliente, locale = "es-ES", obtenerCatalogos = () => null, mensajes = {},
  obtenerJornadaCompleta = () => null,
} = {}) {
  if (typeof cliente?.consultarCuadroRRHH !== "function"
    || typeof cliente?.consultarDetalleRRHH !== "function") {
    throw new TypeError("cliente de expedientes de contratación temporal no disponible");
  }
  if (typeof locale !== "string" || locale.trim() === "") {
    throw new TypeError("locale de expedientes no válido");
  }
  if (typeof obtenerCatalogos !== "function") {
    throw new TypeError("obtener catálogos de expedientes no válido");
  }
  if (typeof obtenerJornadaCompleta !== "function") {
    throw new TypeError("obtener jornada completa de expedientes no válido");
  }
  const t = crearTraductorExpedientesContratacion(mensajes);
  const versiones = new Map();
  const capacidadesConsultadas = new Set();
  let secuenciaCuadro = 0;
  let catalogosCargados = null;
  let promesaCatalogos = null;

  async function resolverCatalogos() {
    const directo = obtenerCatalogos();
    if (directo !== null && directo !== undefined) {
      return etiquetasCatalogos(() => directo);
    }
    if (catalogosCargados !== null) {
      return catalogosCargados;
    }
    if (promesaCatalogos === null) {
      if (typeof cliente?.catalogosAlta !== "function") {
        return null;
      }
      promesaCatalogos = cliente.catalogosAlta()
        .then((respuesta) => {
          catalogosCargados = etiquetasCatalogos(() => respuesta);
          return catalogosCargados;
        })
        .catch(() => {
          catalogosCargados = null;
          return null;
        });
    }
    return promesaCatalogos;
  }

  const adaptador = {
    get capacidades() {
      return Object.freeze([...capacidadesConsultadas]);
    },
    async listar({ filtros = { texto: "", estado: "", fase: "" }, cursor = "", numeroPagina = 1, signal } = {}) {
      if (typeof cursor !== "string" || (cursor !== "" && !/^[A-Za-z0-9_-]{42}[AEIMQUYcgkosw048]$/u.test(cursor))
        || !Number.isSafeInteger(numeroPagina) || numeroPagina < 1
        || (numeroPagina === 1) !== (cursor === "")) throw new TypeError("paginación de cuadro no válida");
      const operacion = ++secuenciaCuadro;
      const [pagina, catalogos] = await Promise.all([
        cliente.consultarCuadroRRHH({
          filtros: {
            texto: filtros.texto,
            estado_clave: estadoServidor(filtros.estado),
            fase_clave: filtros.fase,
          },
          paginacion: { limite: 100, cursor },
        }, { signal }),
        resolverCatalogos(),
      ]);
      const cuadro = proyectarCuadro(pagina, {
        cursor, numeroPagina, catalogos, t, locale,
      });
      if (operacion === secuenciaCuadro && !signal?.aborted) {
        versiones.clear();
        pagina.expedientes.forEach(({ expediente_ref: referencia, version }) => versiones.set(referencia, version));
        capacidadesConsultadas.add(CAPACIDADES_CONTRATACION_TEMPORAL.consultarCuadro);
      }
      return cuadro;
    },
    async obtener(expedienteRef, { signal } = {}) {
      const version = versiones.get(expedienteRef);
      if (!Number.isSafeInteger(version) || version < 1) {
        throw new TypeError("expediente fuera del cuadro consultado");
      }
      const [detalle, catalogos] = await Promise.all([
        cliente.consultarDetalleRRHH({
          expediente_ref: expedienteRef,
          version_observada: version,
        }, { signal }),
        resolverCatalogos(),
      ]);
      const expediente = proyectarExpediente(
        detalle, locale, catalogos, t, mensajes, obtenerJornadaCompleta(),
      );
      capacidadesConsultadas.add(CAPACIDADES_CONTRATACION_TEMPORAL.consultarExpediente);
      return expediente;
    },
    async ejecutar() {
      const error = new Error("Las actuaciones todavía no están conectadas");
      error.codigo = "actuacion_no_disponible";
      throw error;
    },
  };
  return Object.freeze(adaptador);
}
