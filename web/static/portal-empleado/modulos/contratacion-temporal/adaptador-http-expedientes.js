import {
  CAPACIDADES_CONTRATACION_TEMPORAL,
  validarCuadroContratacionTemporal,
  validarExpedienteContratacionTemporal,
} from "./contrato-expedientes.js";

const ESTADOS = new Set([
  "pendiente", "en_curso", "espera", "completado", "incidencia", "cancelado",
]);

function etiqueta(clave, alternativa = "No consta") {
  if (typeof clave !== "string" || clave === "") return alternativa;
  const texto = clave.replaceAll("_", " ").replaceAll("-", " ");
  return texto.charAt(0).toLocaleUpperCase("es-ES") + texto.slice(1);
}

function estado(clave) {
  return ESTADOS.has(clave) ? clave : "pendiente";
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

function resumenVisual(entrada) {
  return {
    expediente_ref: entrada.expediente_ref,
    numero_visible: entrada.numero_visible,
    centro: entrada.centro_ref,
    categoria: entrada.categoria_ref,
    modalidad: etiqueta(entrada.modalidad_clave, "Pendiente de análisis"),
    estado_clave: estado(entrada.estado_clave),
    estado: etiqueta(entrada.estado_clave),
    fase_actual: etiqueta(entrada.fase_clave),
    fecha_solicitud: entrada.creado_en,
    responsable: entrada.unidad_ref || "Sin unidad asignada",
    plazo: entrada.actualizado_en,
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

function proyectarCuadro(pagina) {
  const expedientes = pagina.expedientes.map(resumenVisual);
  return validarCuadroContratacionTemporal({
    esquema: "vec.contratacion_temporal.cuadro.v1",
    demostracion: false,
    generado_en: pagina.generada_en,
    indicadores: indicadores(expedientes),
    expedientes,
  });
}

function cabeceraDetalle(detalle) {
  const { resumen, solicitud } = detalle;
  const campos = [
    campo("centro", "Centro", resumen.centro_ref),
    campo("categoria", "Categoría", resumen.categoria_ref),
    campo("modalidad", "Modalidad", etiqueta(resumen.modalidad_clave)),
    campo("fase", "Fase actual", etiqueta(resumen.fase_clave)),
    campo("estado", "Estado", etiqueta(resumen.estado_clave)),
    campo("grupo_subgrupo", "Grupo/Subgrupo", solicitud.grupo_subgrupo),
    campo("motivo", "Motivo", etiqueta(solicitud.motivo_clave)),
    campo("periodo", "Periodo previsto", `${solicitud.periodo_inicio} — ${solicitud.periodo_fin}`),
  ];
  if (detalle.analisis) {
    campos.push(
      campo("causa", "Causa analizada", etiqueta(detalle.analisis.causa_clave)),
      campo("jornada", "Jornada (diezmilésimas)", detalle.analisis.porcentaje_jornada),
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

function normalizarHitos(detalle) {
  if (detalle.hitos.length > 0) return detalle.hitos;
  return [{
    secuencia: 1,
    version_expediente: detalle.resumen.version,
    accion_clave: "estado_actual",
    realizada_en: detalle.resumen.actualizado_en,
    fase_destino: detalle.resumen.fase_clave,
    estado_origen: detalle.resumen.estado_clave,
    estado_destino: detalle.resumen.estado_clave,
  }];
}

function proyectarExpediente(detalle) {
  const hitos = normalizarHitos(detalle);
  const clavesFases = [...new Set(hitos.map(({ fase_destino: clave }) => clave))];
  const fases = clavesFases.map((clave, indice) => ({
    fase_ref: `fase:${clave}`,
    orden: indice + 1,
    etiqueta: etiqueta(clave),
    estado_clave: indice === clavesFases.length - 1
      ? estado(detalle.resumen.estado_clave) : "completado",
  }));
  const tareas = hitos.map((hito, indice) => {
    const ultima = indice === hitos.length - 1;
    const estadoTarea = ultima ? estado(detalle.resumen.estado_clave) : "completado";
    return {
      tarea_ref: `tarea:hito:${hito.secuencia}`,
      orden: indice + 1,
      fase_ref: `fase:${hito.fase_destino}`,
      etiqueta: etiqueta(hito.accion_clave),
      descripcion: "Actuación registrada en la cronología operativa del expediente.",
      estado_clave: estadoTarea,
      estado: etiqueta(estadoTarea),
      unidad: detalle.resumen.unidad_ref || "Sin unidad asignada",
      responsable: "Identidad no publicada",
      entrada: hito.realizada_en,
      salida: ultima && estadoTarea !== "completado" ? "" : hito.realizada_en,
      tiempo: ultima ? "Estado actual" : "Registrada",
      recibo_ref: "",
      decision_ref: "",
      paneles: [{
        panel_ref: `panel:hito:${hito.secuencia}`,
        tipo: "datos",
        titulo: "Traza operativa",
        descripcion: "Proyección minimizada publicada por el servidor.",
        campos: [
          campo("accion", "Actuación", etiqueta(hito.accion_clave)),
          campo("realizada_en", "Realizada", hito.realizada_en),
          campo("transicion", "Transición", `${etiqueta(hito.estado_origen)} → ${etiqueta(hito.estado_destino)}`),
        ],
        columnas: [],
        filas: [],
      }],
      acciones: [],
    };
  });
  return validarExpedienteContratacionTemporal({
    esquema: "vec.contratacion_temporal.expediente.v1",
    demostracion: false,
    expediente_ref: detalle.resumen.expediente_ref,
    numero_visible: detalle.resumen.numero_visible,
    version: detalle.resumen.version,
    flujo_ref: detalle.resumen.flujo_ref,
    flujo_version: detalle.resumen.flujo_version,
    flujo_huella: detalle.resumen.flujo_huella_sha256,
    cabecera: cabeceraDetalle(detalle),
    fases,
    tareas,
  });
}

export function crearAdaptadorHTTPExpedientesContratacionTemporal({ cliente } = {}) {
  if (typeof cliente?.consultarCuadroRRHH !== "function"
    || typeof cliente?.consultarDetalleRRHH !== "function") {
    throw new TypeError("cliente de expedientes de contratación temporal no disponible");
  }
  const versiones = new Map();
  return Object.freeze({
    capacidades: Object.freeze([
      CAPACIDADES_CONTRATACION_TEMPORAL.consultarCuadro,
      CAPACIDADES_CONTRATACION_TEMPORAL.consultarExpediente,
    ]),
    async listar({ filtros = { texto: "", estado: "", fase: "" }, signal } = {}) {
      const pagina = await cliente.consultarCuadroRRHH({
        filtros: {
          texto: filtros.texto,
          estado_clave: filtros.estado,
          fase_clave: filtros.fase,
        },
        paginacion: { limite: 100, cursor: "" },
      }, { signal });
      versiones.clear();
      pagina.expedientes.forEach(({ expediente_ref: referencia, version }) => {
        versiones.set(referencia, version);
      });
      return proyectarCuadro(pagina);
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
      return proyectarExpediente(detalle);
    },
    async ejecutar() {
      const error = new Error("Las actuaciones todavía no están conectadas");
      error.codigo = "actuacion_no_disponible";
      throw error;
    },
  });
}
