/**
 * Adaptador sintético y descartable de la presentación RRHH.
 *
 * Solo se carga dinámicamente con `?presentacion=rrhh`. No usa red,
 * persistencia del navegador ni sistemas externos. La composición productiva
 * debe sustituir este único adaptador por los puertos reales.
 */

import { exigirContextoParaModulo } from "../../identidad/contexto-actor.js";
import {
  CAPACIDADES_CONTRATACION_TEMPORAL as CAP,
  validarAuditoriaContratacionTemporal,
  validarComandoActuacion,
  validarCuadroContratacionTemporal,
  validarDocumentosContratacionTemporal,
  validarExpedienteContratacionTemporal,
  validarReciboActuacion,
} from "./contrato-expedientes.js";
import { validarComandoAlta, validarReciboAlta } from "./contrato.js";
import {
  crearAuditoriaContratacionTemporalPresentacion,
  crearCatalogosAltaContratacionTemporalPresentacion,
  crearCuadroContratacionTemporalPresentacion,
  crearDocumentosContratacionTemporalPresentacion,
  crearExpedienteContratacionTemporalPresentacion,
} from "./datos-presentacion.js";

const CAPACIDADES_ADMIN = Object.freeze(Object.values(CAP));
const CAPACIDADES_TECNICO = Object.freeze([
  CAP.consultarCuadro,
  CAP.consultarExpediente,
  CAP.consultarDocumentos,
  CAP.crearSolicitud,
  CAP.enviarAnalisis,
  CAP.analizar,
  CAP.decidirCobertura,
  CAP.asignarUnidad,
  CAP.prepararInforme,
  CAP.solicitarFiscalizacion,
  CAP.registrarSubsanacion,
  CAP.prepararLlamamiento,
  CAP.seleccionarCandidatura,
  CAP.registrarResultadoLlamamiento,
  CAP.prepararFormalizacion,
  CAP.confirmarIncorporacion,
  CAP.exportarGinpix,
  CAP.registrarSeguimiento,
  CAP.consultarAuditoria,
]);

function capacidadesDelContexto(contexto) {
  if (contexto.rol.clave === "administrador_funcional_bolsa") return CAPACIDADES_ADMIN;
  if (contexto.rol.clave === "tecnico_revisor_rrhh") return CAPACIDADES_TECNICO;
  throw new Error("el perfil sintético no dispone de contratación temporal");
}

function copiar(valor) {
  return structuredClone(valor);
}

function abortarSiProcede(signal) {
  if (signal?.aborted) throw new DOMException("Operación cancelada", "AbortError");
}

function textoFiltro(valor) {
  return String(valor ?? "").normalize("NFC").trim().toLocaleLowerCase("es-ES");
}

function filtrarCuadro(cuadro, filtros) {
  const texto = textoFiltro(filtros?.texto);
  const estado = textoFiltro(filtros?.estado);
  const fase = textoFiltro(filtros?.fase);
  const expedientes = cuadro.expedientes.filter((expediente) => {
    const contenido = [
      expediente.numero_visible,
      expediente.centro,
      expediente.categoria,
      expediente.modalidad,
      expediente.responsable,
    ].join(" ").toLocaleLowerCase("es-ES");
    return (texto === "" || contenido.includes(texto))
      && (estado === "" || expediente.estado_clave === estado)
      && (fase === "" || expediente.fase_actual.toLocaleLowerCase("es-ES") === fase);
  });
  return validarCuadroContratacionTemporal({ ...copiar(cuadro), expedientes });
}

function expedienteDerivado(base, resumen) {
  if (resumen.expediente_ref === base.expediente_ref) return base;
  const cabecera = copiar(base.cabecera).map((campo) => {
    const valores = {
      centro: resumen.centro,
      categoria: resumen.categoria,
      modalidad: resumen.modalidad,
      responsable: resumen.responsable,
      estado: resumen.estado,
    };
    return Object.hasOwn(valores, campo.clave) ? { ...campo, valor: valores[campo.clave] } : campo;
  });
  const indiceActual = {
    fiscalización: 6,
    subsanación: 7,
    llamamiento: 9,
    seguimiento: 16,
  }[resumen.fase_actual.toLocaleLowerCase("es-ES")] ?? 0;
  const finalizado = resumen.estado_clave === "completado";
  const tareas = copiar(base.tareas).map((tarea, indice) => {
    const completada = finalizado || indice < indiceActual;
    const actual = !finalizado && indice === indiceActual;
    let estadoClave = "pendiente";
    if (completada) estadoClave = "completado";
    else if (actual) estadoClave = resumen.estado_clave;
    const accionesDisponibles = actual && !["espera", "cancelado"].includes(estadoClave);
    return {
      ...tarea,
      estado_clave: estadoClave,
      estado: completada ? "Completado" : (actual ? resumen.estado : "Pendiente"),
      entrada: completada || actual ? tarea.entrada : "Pendiente",
      salida: completada ? (tarea.salida || "22/07/2026 12:00") : "",
      tiempo: completada ? (tarea.tiempo || "Completado") : (actual ? resumen.plazo : "Sin iniciar"),
      recibo_ref: completada ? (tarea.recibo_ref || `rec-demo-${resumen.version}-${tarea.orden}`) : "",
      decision_ref: completada
        ? (tarea.decision_ref || `dec-demo-${resumen.version}-${tarea.orden}`) : "",
      acciones: tarea.acciones.map((accion) => ({
        ...accion,
        disponible: accionesDisponibles,
        motivo_no_disponible: accionesDisponibles
          ? "" : (completada
            ? "La tarea forma parte del histórico cerrado."
            : "La actuación aún no está disponible en esta fase."),
      })),
    };
  });
  return {
    ...copiar(base),
    expediente_ref: resumen.expediente_ref,
    numero_visible: resumen.numero_visible,
    version: resumen.version,
    cabecera,
    fases: fasesDesdeTareas(base.fases, tareas),
    tareas,
  };
}

function proyeccionDerivada(base, resumen, coleccion) {
  const limite = {
    fiscalización: coleccion === "documentos" ? 4 : 5,
    subsanación: coleccion === "documentos" ? 4 : 6,
    llamamiento: coleccion === "documentos" ? 5 : 7,
  }[resumen.fase_actual.toLocaleLowerCase("es-ES")] ?? base[coleccion].length;
  return {
    ...copiar(base),
    expediente_ref: resumen.expediente_ref,
    version: resumen.version,
    [coleccion]: copiar(base[coleccion].slice(0, limite)),
  };
}

function fasesDesdeTareas(fases, tareas) {
  return fases.map((fase) => {
    const estados = tareas
      .filter(({ fase_ref: referencia }) => referencia === fase.fase_ref)
      .map(({ estado_clave: estado }) => estado);
    let estadoClave = "pendiente";
    if (estados.includes("incidencia")) estadoClave = "incidencia";
    else if (estados.includes("en_curso")) estadoClave = "en_curso";
    else if (estados.includes("espera")) estadoClave = "espera";
    else if (estados.length > 0 && estados.every((estado) => estado === "completado")) {
      estadoClave = "completado";
    } else if (estados.includes("cancelado")) estadoClave = "cancelado";
    return { ...copiar(fase), estado_clave: estadoClave };
  });
}

function aplicarEfectoEnDetalle(detalle, comando, reciboRef, decisionRef) {
  const indiceTarea = detalle.tareas.findIndex(
    ({ tarea_ref: referencia }) => referencia === comando.tarea_ref,
  );
  const tareas = detalle.tareas.map((tarea, indice) => {
    if (indice !== indiceTarea) return copiar(tarea);
    const acciones = tarea.acciones.map((accion) => (
      accion.accion_ref === comando.accion_ref
        ? {
          ...copiar(accion),
          disponible: false,
          motivo_no_disponible: "La actuación ya quedó registrada en esta versión.",
        }
        : copiar(accion)
    ));
    const mantieneAccion = acciones.some(({ disponible }) => disponible);
    return {
      ...copiar(tarea),
      estado_clave: mantieneAccion ? "en_curso" : "completado",
      estado: mantieneAccion ? "Actuación registrada; continúan tareas" : "Completado",
      salida: mantieneAccion ? "" : "23/07/2026 11:35",
      tiempo: mantieneAccion ? "En curso" : "Completado en presentación",
      recibo_ref: reciboRef,
      decision_ref: decisionRef,
      acciones,
    };
  });
  if (tareas[indiceTarea]?.estado_clave === "completado" && tareas[indiceTarea + 1]) {
    const siguiente = tareas[indiceTarea + 1];
    tareas[indiceTarea + 1] = {
      ...copiar(siguiente),
      estado_clave: "en_curso",
      estado: "En tramitación",
      entrada: "23/07/2026 11:35",
      tiempo: "En curso",
      acciones: siguiente.acciones.map((accion) => ({
        ...copiar(accion),
        disponible: true,
        motivo_no_disponible: "",
      })),
    };
  }
  return {
    ...copiar(detalle),
    version: comando.version_esperada + 1,
    fases: fasesDesdeTareas(detalle.fases, tareas),
    tareas,
  };
}

function documentosTrasEfecto(indice, accionRef, version) {
  let aplicado = false;
  return {
    ...copiar(indice),
    version,
    documentos: indice.documentos.map((documento) => {
      if (aplicado) return copiar(documento);
      const generar = accionRef.includes("generar") && documento.estado === "Pendiente";
      const firmar = accionRef.includes("firma") && documento.estado.includes("Pendiente de firma");
      if (!generar && !firmar) return copiar(documento);
      aplicado = true;
      return {
        ...copiar(documento),
        version: documento.version + 1,
        estado: generar ? "Generado" : "En firma",
        firma: generar ? "Pendiente de firma" : "Enviada al portafirmas DEMO",
        fecha: "23/07/2026",
        descarga_disponible: generar,
      };
    }),
  };
}

export function crearAdaptadorContratacionTemporalPresentacion({ contextoActor } = {}) {
  const contexto = exigirContextoParaModulo(contextoActor, "contratacion_temporal");
  if (contexto.demostracion !== true) {
    throw new TypeError("el adaptador de presentación exige un contexto sintético");
  }

  let cuadro = validarCuadroContratacionTemporal(
    crearCuadroContratacionTemporalPresentacion(),
  );
  const expedienteBase = validarExpedienteContratacionTemporal(
    crearExpedienteContratacionTemporalPresentacion(),
  );
  const documentosBase = validarDocumentosContratacionTemporal(
    crearDocumentosContratacionTemporalPresentacion(),
  );
  const auditoriaBase = validarAuditoriaContratacionTemporal(
    crearAuditoriaContratacionTemporalPresentacion(),
  );
  const expedientes = new Map();
  const documentos = new Map();
  const auditorias = new Map();
  for (const resumen of cuadro.expedientes) {
    expedientes.set(
      resumen.expediente_ref,
      validarExpedienteContratacionTemporal(expedienteDerivado(expedienteBase, resumen)),
    );
    documentos.set(
      resumen.expediente_ref,
      validarDocumentosContratacionTemporal(
        proyeccionDerivada(documentosBase, resumen, "documentos"),
      ),
    );
    auditorias.set(
      resumen.expediente_ref,
      validarAuditoriaContratacionTemporal(
        proyeccionDerivada(auditoriaBase, resumen, "actuaciones"),
      ),
    );
  }
  let secuenciaRecibos = 1;
  let secuenciaAltas = 5488;
  const capacidades = capacidadesDelContexto(contexto);

  async function listar({ filtros = {}, signal } = {}) {
    if (!capacidades.includes(CAP.consultarCuadro)) throw new Error("Acceso denegado.");
    abortarSiProcede(signal);
    await Promise.resolve();
    abortarSiProcede(signal);
    return filtrarCuadro(cuadro, filtros);
  }

  async function obtener(expedienteRef, { signal } = {}) {
    if (!capacidades.includes(CAP.consultarExpediente)) throw new Error("Acceso denegado.");
    abortarSiProcede(signal);
    const expediente = expedientes.get(expedienteRef);
    if (!expediente) throw new Error("El expediente solicitado no está disponible.");
    await Promise.resolve();
    abortarSiProcede(signal);
    return validarExpedienteContratacionTemporal(copiar(expediente));
  }

  async function obtenerDocumentos(expedienteRef, { signal } = {}) {
    if (!capacidades.includes(CAP.consultarDocumentos)) throw new Error("Acceso denegado.");
    abortarSiProcede(signal);
    const indice = documentos.get(expedienteRef);
    if (!indice) throw new Error("El índice documental no está disponible.");
    await Promise.resolve();
    abortarSiProcede(signal);
    return validarDocumentosContratacionTemporal(copiar(indice));
  }

  async function obtenerAuditoria(expedienteRef, { signal } = {}) {
    if (!capacidades.includes(CAP.consultarAuditoria)) throw new Error("Acceso denegado.");
    abortarSiProcede(signal);
    const auditoria = auditorias.get(expedienteRef);
    if (!auditoria) throw new Error("La auditoría no está disponible.");
    await Promise.resolve();
    abortarSiProcede(signal);
    return validarAuditoriaContratacionTemporal(copiar(auditoria));
  }

  async function ejecutar(entrada, { signal } = {}) {
    abortarSiProcede(signal);
    const comando = validarComandoActuacion(entrada);
    const resumen = cuadro.expedientes.find(
      (expediente) => expediente.expediente_ref === comando.expediente_ref,
    );
    if (!resumen) throw new Error("El expediente solicitado no está disponible.");
    if (resumen.version !== comando.version_esperada) {
      throw new Error("El expediente cambió. Recargue antes de continuar.");
    }
    const detalle = expedientes.get(resumen.expediente_ref);
    const tarea = detalle.tareas.find(({ tarea_ref: referencia }) => referencia === comando.tarea_ref);
    const actuacion = tarea?.acciones.find(({ accion_ref: referencia }) => referencia === comando.accion_ref);
    if (!actuacion || actuacion.tipo !== "efecto" || actuacion.disponible !== true) {
      throw new Error("La actuación no está disponible en esta tarea.");
    }
    if (!capacidades.includes(actuacion.capacidad)) throw new Error("Acceso denegado.");
    if (actuacion.accion_ref === "cerrar_expediente") {
      throw new Error("El expediente mantiene tareas pendientes.");
    }
    await Promise.resolve();
    abortarSiProcede(signal);

    const version = resumen.version + 1;
    const secuencia = String(secuenciaRecibos).padStart(6, "0");
    const reciboRef = `rec-demo-actuacion-${secuencia}`;
    const decisionRef = `dec-demo-actuacion-${secuencia}`;
    const detalleActualizado = validarExpedienteContratacionTemporal(
      aplicarEfectoEnDetalle(detalle, comando, reciboRef, decisionRef),
    );
    const tareaActual = detalleActualizado.tareas.find(
      ({ estado_clave: estado }) => estado === "en_curso",
    ) ?? detalleActualizado.tareas.at(-1);
    cuadro = validarCuadroContratacionTemporal({
      ...copiar(cuadro),
      generado_en: "2026-07-23T09:35:00Z",
      expedientes: cuadro.expedientes.map((elemento) => (
        elemento.expediente_ref === resumen.expediente_ref
          ? {
            ...copiar(elemento),
            version,
            estado: "Actuación DEMO registrada",
            fase_actual: tareaActual?.etiqueta ?? elemento.fase_actual,
          }
          : copiar(elemento)
      )),
    });
    expedientes.set(resumen.expediente_ref, detalleActualizado);
    const indiceAnterior = documentos.get(resumen.expediente_ref);
    documentos.set(
      resumen.expediente_ref,
      validarDocumentosContratacionTemporal(
        documentosTrasEfecto(indiceAnterior, actuacion.accion_ref, version),
      ),
    );
    const auditoriaAnterior = auditorias.get(resumen.expediente_ref);
    auditorias.set(resumen.expediente_ref, validarAuditoriaContratacionTemporal({
      ...copiar(auditoriaAnterior),
      version,
      actuaciones: [
        ...copiar(auditoriaAnterior.actuaciones),
        {
          actuacion_ref: `act-demo-nueva-${String(secuenciaRecibos).padStart(3, "0")}`,
          fecha: "23/07/2026 11:35",
          fase: tarea.etiqueta,
          accion: actuacion.etiqueta,
          actor: contexto.actor.nombre_visible,
          unidad: tarea.unidad,
          estado: "Registrado",
          observaciones: "Actuación sintética sin efectos administrativos.",
          documento_ref: "",
        },
      ],
    }));
    const recibo = validarReciboActuacion({
      esquema: "vec.contratacion_temporal.recibo-actuacion.v1",
      recibo_ref: reciboRef,
      expediente_ref: resumen.expediente_ref,
      numero_visible: resumen.numero_visible,
      version,
      actuacion: actuacion.etiqueta,
      estado_resultante: "Actuación DEMO registrada sin efectos reales",
      registrada_en: "2026-07-23T09:35:00Z",
    });
    secuenciaRecibos += 1;
    return recibo;
  }

  async function registrarSolicitud(entrada, { signal } = {}) {
    if (!capacidades.includes(CAP.crearSolicitud)) throw new Error("Acceso denegado.");
    abortarSiProcede(signal);
    const comando = validarComandoAlta(entrada);
    await Promise.resolve();
    abortarSiProcede(signal);
    const numero = String(secuenciaAltas).padStart(5, "0");
    secuenciaAltas += 1;
    const expedienteRef = `exp-demo-contratacion-${numero}`;
    const numeroVisible = `2026/CT-${numero}`;
    const catalogos = crearCatalogosAltaContratacionTemporalPresentacion();
    const solicitud = comando.solicitud;
    const centro = catalogos.centros.find(({ referencia }) => referencia === solicitud.centro_ref);
    const categoria = catalogos.categorias.find(
      ({ referencia }) => referencia === solicitud.categoria_ref,
    );
    const nuevoResumen = {
      expediente_ref: expedienteRef,
      numero_visible: numeroVisible,
      centro: centro?.etiqueta ?? solicitud.centro_ref,
      categoria: categoria?.etiqueta ?? solicitud.categoria_ref,
      modalidad: "Pendiente de análisis",
      estado_clave: "en_curso",
      estado: "Solicitud registrada",
      fase_actual: "Solicitud",
      fecha_solicitud: "23/07/2026",
      responsable: "Dirección de RRHH",
      plazo: "Pendiente de asignación",
      version: 1,
    };
    cuadro = validarCuadroContratacionTemporal({
      ...copiar(cuadro),
      generado_en: "2026-07-23T09:36:00Z",
      indicadores: cuadro.indicadores.map((indicador) => (
        indicador.clave === "tramitacion"
          ? { ...copiar(indicador), valor: String(Number(indicador.valor) + 1) }
          : copiar(indicador)
      )),
      expedientes: [nuevoResumen, ...copiar(cuadro.expedientes)],
    });
    const nuevoDetalle = validarExpedienteContratacionTemporal({
      esquema: "vec.contratacion_temporal.expediente.v1",
      demostracion: true,
      expediente_ref: expedienteRef,
      numero_visible: numeroVisible,
      version: 1,
      flujo_ref: expedienteBase.flujo_ref,
      flujo_version: expedienteBase.flujo_version,
      flujo_huella: expedienteBase.flujo_huella,
      cabecera: [
        ["centro", "Servicio solicitante", nuevoResumen.centro, "neutro"],
        ["categoria", "Categoría", nuevoResumen.categoria, "neutro"],
        ["modalidad", "Modalidad", "Pendiente de análisis", "aviso"],
        ["procedimiento", "Procedimiento", "Pendiente de decisión", "aviso"],
        ["periodo", "Periodo previsto", `${solicitud.periodo.inicio.slice(0, 10)} — ${solicitud.periodo.fin.slice(0, 10)}`, "neutro"],
        ["coste", "Coste estimado", "Pendiente de cálculo autorizado", "aviso"],
        ["responsable", "Unidad responsable", "Dirección de RRHH", "neutro"],
        ["estado", "Estado actual", "Solicitud registrada", "informacion"],
      ].map(([clave, etiqueta, valor, tono]) => ({
        clave, etiqueta, valor, tono, control: "solo_lectura", obligatorio: false, opciones: [],
      })),
      fases: expedienteBase.fases.map((fase, indice) => ({
        ...copiar(fase),
        estado_clave: indice === 0 ? "en_curso" : "pendiente",
      })),
      tareas: expedienteBase.tareas.map((tarea, indice) => ({
        tarea_ref: tarea.tarea_ref,
        orden: tarea.orden,
        fase_ref: tarea.fase_ref,
        etiqueta: tarea.etiqueta,
        descripcion: tarea.descripcion,
        estado_clave: indice === 0 ? "en_curso" : "pendiente",
        estado: indice === 0 ? "Solicitud registrada" : "Pendiente",
        unidad: indice === 0 ? "Centro solicitante DEMO" : tarea.unidad,
        responsable: indice === 0 ? contexto.actor.nombre_visible : tarea.responsable,
        entrada: indice === 0 ? "23/07/2026 11:36" : "Pendiente",
        salida: "",
        tiempo: indice === 0 ? "En curso" : "Sin iniciar",
        recibo_ref: "",
        decision_ref: "",
        paneles: indice === 0 ? [{
          panel_ref: "panel-solicitud-nueva",
          tipo: "datos",
          titulo: "Solicitud recién registrada",
          descripcion: "Proyección mínima del comando confirmado; las fases posteriores aún no tienen datos.",
          campos: [
            ["detalle", "Petición", solicitud.detalle, "neutro"],
            ["periodo", "Periodo", `${solicitud.periodo.inicio.slice(0, 10)} — ${solicitud.periodo.fin.slice(0, 10)}`, "neutro"],
            ["rc", "Retención de crédito", solicitud.rc.existe ? "Declarada; pendiente de validar" : "No declarada", solicitud.rc.existe ? "aviso" : "neutro"],
            ["documentos", "Referencias documentales", String(solicitud.documentos_adjuntos.length), "neutro"],
          ].map(([clave, etiqueta, valor, tono]) => ({
            clave, etiqueta, valor, tono, control: "solo_lectura", obligatorio: false, opciones: [],
          })),
          columnas: [],
          filas: [],
        }] : [],
        acciones: indice === 0
          ? tarea.acciones.map((accion) => ({
            ...copiar(accion),
            disponible: true,
            motivo_no_disponible: "",
          }))
          : [],
      })),
    });
    expedientes.set(expedienteRef, nuevoDetalle);
    documentos.set(expedienteRef, validarDocumentosContratacionTemporal({
      esquema: "vec.contratacion_temporal.documentos.v1",
      demostracion: true,
      expediente_ref: expedienteRef,
      version: 1,
      documentos: [],
    }));
    auditorias.set(expedienteRef, validarAuditoriaContratacionTemporal({
      esquema: "vec.contratacion_temporal.auditoria.v1",
      demostracion: true,
      expediente_ref: expedienteRef,
      version: 1,
      actuaciones: [{
        actuacion_ref: `act-demo-alta-${numero}`,
        fecha: "23/07/2026 11:36",
        fase: "Solicitud",
        accion: "Registro de solicitud",
        actor: contexto.actor.nombre_visible,
        unidad: "Centro solicitante DEMO",
        estado: "Registrado",
        observaciones: "Solicitud sintética registrada sin efectos administrativos.",
        documento_ref: "",
      }],
    }));
    return validarReciboAlta({
      expediente_ref: `exp-demo-contratacion-${numero}`,
      numero_visible: numeroVisible,
      version: 1,
      recibo_ref: `rec-demo-alta-${numero}`,
      confirmada_en: "2026-07-23T09:36:00Z",
    });
  }

  return Object.freeze({
    capacidades,
    listar,
    obtener,
    obtenerDocumentos,
    obtenerAuditoria,
    ejecutar,
    registrarSolicitud,
    obtenerCatalogosAlta() {
      return copiar(crearCatalogosAltaContratacionTemporalPresentacion());
    },
  });
}
