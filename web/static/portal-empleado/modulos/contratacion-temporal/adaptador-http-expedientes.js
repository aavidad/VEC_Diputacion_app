import {
  CAPACIDADES_CONTRATACION_TEMPORAL,
  validarCuadroContratacionTemporal,
  validarExpedienteContratacionTemporal,
} from "./contrato-expedientes.js";
import { validarCatalogosAlta } from "./contrato.js";
import { crearTraductorContratacionTemporal } from "./i18n.js";

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

function etiqueta(clave, alternativa = "No consta") {
  if (typeof clave !== "string" || clave === "") return alternativa;
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

function resumenVisual(entrada, catalogos) {
  const estadoClave = estadoVisual(entrada.estado_clave);
  return {
    expediente_ref: entrada.expediente_ref,
    numero_visible: entrada.numero_visible,
    centro: referenciaVisible(catalogos, "centros", entrada.centro_ref),
    categoria: referenciaVisible(catalogos, "categorias", entrada.categoria_ref),
    modalidad: etiqueta(entrada.modalidad_clave, "—"),
    estado_clave: estadoClave,
    estado: etiqueta(entrada.estado_clave),
    fase_clave: entrada.fase_clave,
    fase_actual: etiqueta(entrada.fase_clave),
    fecha_solicitud: entrada.creado_en,
    responsable: "—",
    plazo: "—",
    version: entrada.version,
  };
}

function indicadores(expedientes) {
  const contar = (clave) => expedientes.filter(
    ({ estado_clave: actual }) => actual === clave,
  ).length;
  return [
    ["total", "Expedientes", expedientes.length, "informacion"],
    ["pendientes", "Pendientes", contar("pendiente"), "aviso"],
    ["en_curso", "En curso", contar("en_curso"), "informacion"],
    ["incidencias", "Incidencias", contar("incidencia"), "peligro"],
  ].map(([clave, titulo, valor, tono]) => ({
    clave, etiqueta: titulo, valor: String(valor), tono,
  }));
}

function proyectarCuadro(pagina, { cursor, numeroPagina, catalogos }) {
  const expedientes = pagina.expedientes.map((entrada) => resumenVisual(entrada, catalogos));
  return validarCuadroContratacionTemporal({
    esquema: "vec.contratacion_temporal.cuadro.v1",
    demostracion: false,
    generado_en: pagina.generada_en,
    indicadores: indicadores(expedientes),
    expedientes,
    paginacion: {
      pagina: numeroPagina,
      cursor_actual: cursor,
      cursor_siguiente: pagina.hay_mas ? pagina.cursor_siguiente : "",
    },
  });
}

function fechaCivil(instante, locale) {
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
    dateStyle: "medium", timeZone: "UTC",
  }).format(fecha);
}

function cabeceraDetalle(detalle, locale, catalogos) {
  const { resumen, solicitud } = detalle;
  const campos = [
    campo("centro", "Centro", referenciaVisible(catalogos, "centros", resumen.centro_ref)),
    campo("categoria", "Categoría", referenciaVisible(catalogos, "categorias", resumen.categoria_ref)),
    campo("modalidad", "Modalidad", etiqueta(resumen.modalidad_clave)),
    campo("fase", "Fase actual", etiqueta(resumen.fase_clave)),
    campo("estado", "Estado", etiqueta(resumen.estado_clave)),
    campo("grupo_subgrupo", "Grupo/Subgrupo", solicitud.grupo_subgrupo),
    campo("motivo", "Motivo", etiqueta(solicitud.motivo_clave)),
    campo("periodo", "Periodo previsto", `${fechaCivil(solicitud.periodo_inicio, locale)} — ${fechaCivil(solicitud.periodo_fin, locale)}`),
  ];
  if (detalle.analisis) {
    campos.push(
      campo("causa", "Causa analizada", etiqueta(detalle.analisis.causa_clave)),
      campo("jornada", "Jornada", new Intl.NumberFormat(locale, {
        style: "percent", maximumFractionDigits: 2,
      }).format(detalle.analisis.porcentaje_jornada / 10_000)),
      campo("resultado_rc", "Resultado RC", etiqueta(detalle.analisis.resultado_rc)),
    );
  }
  if (detalle.cobertura) {
    campos.push(
      campo("via_cobertura", "Vía de cobertura", etiqueta(detalle.cobertura.via_clave)),
      campo("decision_gobernada", "Decisión gobernada", detalle.cobertura.decision_gobernada ? "Sí" : "No"),
    );
  }
  if (detalle.asignacion) {
    campos.push(campo("unidad", "Unidad asignada", detalle.asignacion.unidad_ref));
  }
  return campos;
}


function resolucionConPropuestaHistorica(detalle) {
  const { resumen, hitos } = detalle;
  if (![8, 9].includes(resumen.version) || resumen.fase_clave !== "nombramiento"
    || resumen.estado_clave !== "en_curso" || !Array.isArray(hitos) || hitos.length !== resumen.version
    || !hitos.every((hito, indice) => hito?.secuencia === indice + 1
      && hito.version_expediente === indice + 1)) return false;
  const propuesta = hitos[6], resolucion = hitos[7];
  const predecesores = propuesta.accion_clave === "registrar_propuesta_formalizacion"
    && propuesta.fase_destino === "nombramiento" && propuesta.estado_destino === "en_curso"
    && resolucion.accion_clave === "registrar_resolucion_formalizacion"
    && resolucion.fase_origen === "nombramiento" && resolucion.fase_destino === "nombramiento"
    && resolucion.estado_origen === "en_curso" && resolucion.estado_destino === "en_curso";
  if (!predecesores) return false;
  if (resumen.version === 8) return true;
  const anotacion = hitos[8];
  return anotacion.accion_clave === "contratacion_temporal.anotacion_administrativa.registrar"
    && anotacion.fase_origen === "nombramiento" && anotacion.fase_destino === "nombramiento"
    && anotacion.estado_origen === "en_curso" && anotacion.estado_destino === "en_curso";
}

function proyectarExpediente(detalle, locale, catalogos) {
  const traducir = crearTraductorContratacionTemporal();
  return validarExpedienteContratacionTemporal({
    esquema: "vec.contratacion_temporal.expediente.v1",
    demostracion: false,
    expediente_ref: detalle.resumen.expediente_ref,
    numero_visible: detalle.resumen.numero_visible,
    version: detalle.resumen.version,
    flujo_ref: detalle.resumen.flujo_ref,
    flujo_version: detalle.resumen.flujo_version,
    flujo_huella: detalle.resumen.flujo_huella_sha256,
    cabecera: cabeceraDetalle(detalle, locale, catalogos),
    fases: (detalle.presentacion_flujo?.fases ?? []).map((fase) => ({
      fase_ref: `presentacion:${detalle.presentacion_flujo.referencia}:${fase.clave}`,
      orden: fase.orden,
      etiqueta: traducir(fase.clave_i18n),
      estado_clave: detalle.resumen.estado_clave === "en_curso"
        && detalle.presentacion_flujo.fase_actual === fase.clave
        ? "en_curso" : "sin_confirmar",
    })),
    tareas: [],
    // Sólo selección documental histórica; cada descarga exige autorización vigente.
    ...(resolucionConPropuestaHistorica(detalle) ? { version_propuesta_documental: 7 } : {}),
  });
}

export function crearAdaptadorHTTPExpedientesContratacionTemporal({
  cliente, locale = "es-ES", obtenerCatalogos = () => null,
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
  const versiones = new Map();
  const capacidadesConsultadas = new Set();
  let secuenciaCuadro = 0;
  const adaptador = {
    get capacidades() {
      return Object.freeze([...capacidadesConsultadas]);
    },
    async listar({ filtros = { texto: "", estado: "", fase: "" }, cursor = "", numeroPagina = 1, signal } = {}) {
      if (typeof cursor !== "string" || (cursor !== "" && !/^[A-Za-z0-9_-]{42}[AEIMQUYcgkosw048]$/u.test(cursor))
        || !Number.isSafeInteger(numeroPagina) || numeroPagina < 1
        || (numeroPagina === 1) !== (cursor === "")) throw new TypeError("paginación de cuadro no válida");
      const operacion = ++secuenciaCuadro;
      const pagina = await cliente.consultarCuadroRRHH({
        filtros: {
          texto: filtros.texto,
          estado_clave: estadoServidor(filtros.estado),
          fase_clave: filtros.fase,
        },
        paginacion: { limite: 100, cursor },
      }, { signal });
      const cuadro = proyectarCuadro(pagina, {
        cursor, numeroPagina, catalogos: etiquetasCatalogos(obtenerCatalogos),
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
      const detalle = await cliente.consultarDetalleRRHH({
        expediente_ref: expedienteRef,
        version_observada: version,
      }, { signal });
      const expediente = proyectarExpediente(detalle, locale, etiquetasCatalogos(obtenerCatalogos));
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
