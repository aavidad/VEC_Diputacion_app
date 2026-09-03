import {
  CAPACIDADES_CONTRATACION_TEMPORAL,
  validarCuadroContratacionTemporal,
  validarExpedienteContratacionTemporal,
} from "./contrato-expedientes.js";

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

function resumenVisual(entrada) {
  const estadoClave = estadoVisual(entrada.estado_clave);
  return {
    expediente_ref: entrada.expediente_ref,
    numero_visible: entrada.numero_visible,
    centro: entrada.centro_ref,
    categoria: entrada.categoria_ref,
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


function proyectarExpediente(detalle) {
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
    fases: [],
    tareas: [],
  });
}

export function crearAdaptadorHTTPExpedientesContratacionTemporal({ cliente } = {}) {
  if (typeof cliente?.consultarCuadroRRHH !== "function"
    || typeof cliente?.consultarDetalleRRHH !== "function") {
    throw new TypeError("cliente de expedientes de contratación temporal no disponible");
  }
  const versiones = new Map();
  const capacidadesConsultadas = new Set();
  const adaptador = {
    get capacidades() {
      return Object.freeze([...capacidadesConsultadas]);
    },
    async listar({ filtros = { texto: "", estado: "", fase: "" }, signal } = {}) {
      const pagina = await cliente.consultarCuadroRRHH({
        filtros: {
          texto: filtros.texto,
          estado_clave: estadoServidor(filtros.estado),
          fase_clave: filtros.fase,
        },
        paginacion: { limite: 100, cursor: "" },
      }, { signal });
      const cuadro = proyectarCuadro(pagina);
      versiones.clear();
      pagina.expedientes.forEach(({ expediente_ref: referencia, version }) => {
        versiones.set(referencia, version);
      });
      capacidadesConsultadas.add(CAPACIDADES_CONTRATACION_TEMPORAL.consultarCuadro);
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
      const expediente = proyectarExpediente(detalle);
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
