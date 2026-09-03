/** Estado y coordinación neutral de la superficie de expedientes. */

import {
  CAPACIDADES_CONTRATACION_TEMPORAL,
  validarAuditoriaContratacionTemporal,
  validarComandoActuacion,
  validarCuadroContratacionTemporal,
  validarDocumentosContratacionTemporal,
  validarExpedienteContratacionTemporal,
  validarReciboActuacion,
} from "./contrato-expedientes.js";

const VISTAS = new Set(["cuadro", "alta", "expediente", "documentos", "auditoria"]);
const ESTADOS_CARGA = new Set([
  "inicial", "cargando", "listo", "vacio", "error", "denegado",
]);

const PATRON_TEXTO_CUADRO = /^[0-9A-Za-zÁÉÍÓÚÜÑáéíóúüñ/._ -]{0,80}$/u;
const PATRON_CLAVE_FASE = /^[a-z][a-z0-9._-]{1,79}$/u;
const ESTADOS_FILTRO = new Set([
  "", "pendiente", "en_curso", "espera", "completado", "incidencia", "cancelado",
]);
function congelar(valor) {
  if (valor && typeof valor === "object" && !Object.isFrozen(valor)) {
    Object.values(valor).forEach(congelar);
    Object.freeze(valor);
  }
  return valor;
}

function copiar(valor) {
  return valor === null || valor === undefined ? valor : structuredClone(valor);
}

function normalizarCapacidades(capacidades) {
  if (!Array.isArray(capacidades) || capacidades.length > 128
    || capacidades.some((capacidad) => (
      typeof capacidad !== "string"
      || !/^[a-z][a-z0-9._-]{1,159}$/u.test(capacidad)
    ))) {
    throw new TypeError("capacidades de presentación no válidas");
  }
  return new Set(capacidades);
}

function proyectarAutorizacionVisual(expediente, concesiones) {
  const proyeccion = copiar(expediente);
  proyeccion.tareas = proyeccion.tareas.map((tarea) => ({
    ...tarea,
    acciones: tarea.acciones.map((accion) => {
      if (accion.tipo !== "efecto" || concesiones.has(accion.capacidad)) return accion;
      return {
        ...accion,
        disponible: false,
        motivo_no_disponible: "El perfil activo no tiene concedida esta actuación.",
      };
    }),
  }));
  return validarExpedienteContratacionTemporal(proyeccion);
}

function fuenteValida(fuente) {
  return fuente !== null && typeof fuente === "object"
    && ["listar", "obtener", "ejecutar"].every((metodo) => typeof fuente[metodo] === "function");
}

function filtrosIniciales() {
  return { texto: "", estado: "", fase: "" };
}

function estadoInicial(disponible) {
  return {
    vista: "cuadro",
    carga: disponible ? "inicial" : "denegado",
    cuadro: null,
    expediente: null,
    documentos: null,
    auditoria: null,
    expediente_ref: "",
    tarea_ref: "",
    filtros: filtrosIniciales(),
    ocupado: false,
    actualizacion_pendiente: false,
    resultado_indeterminado: false,
    recibo: null,
    mensaje_clave: disponible ? "estado_inicial" : "estado_denegado",
    tipo_mensaje: disponible ? "informacion" : "error",
  };
}

function errorPublico(codigo) {
  const error = new Error("La operación no está disponible");
  error.codigo = codigo;
  return error;
}

function esResultadoIndeterminado(error) {
  try {
    return error instanceof Error
      && error.resultadoIndeterminado === true
      && error.reintentoPermitido === false;
  } catch {
    return false;
  }
}

function filtrosValidos(entrada) {
  const resultado = filtrosIniciales();
  if (!entrada || typeof entrada !== "object" || Array.isArray(entrada)
    || Object.keys(entrada).length !== Object.keys(resultado).length
    || Object.keys(entrada).some((campo) => !Object.hasOwn(resultado, campo))) {
    throw new TypeError("filtros no válidos");
  }
  for (const campo of Object.keys(resultado)) {
    const valor = entrada[campo];
    if (typeof valor !== "string" || valor !== valor.trim()
      || valor.normalize("NFC") !== valor) {
      throw new TypeError("filtros no válidos");
    }
    resultado[campo] = valor;
  }
  if (!PATRON_TEXTO_CUADRO.test(resultado.texto)
    || !ESTADOS_FILTRO.has(resultado.estado)
    || (resultado.fase !== "" && !PATRON_CLAVE_FASE.test(resultado.fase))) {
    throw new TypeError("filtros no válidos");
  }
  return resultado;
}

export function crearPresentadorExpedientesContratacionTemporal({
  fuente,
  capacidades = [],
} = {}) {
  const concesionesVisuales = normalizarCapacidades(capacidades);
  const puedeConsultarCuadro = concesionesVisuales.has(
    CAPACIDADES_CONTRATACION_TEMPORAL.consultarCuadro,
  );
  const puedeConsultarExpediente = concesionesVisuales.has(
    CAPACIDADES_CONTRATACION_TEMPORAL.consultarExpediente,
  );
  const puedeConsultarDocumentos = concesionesVisuales.has(
    CAPACIDADES_CONTRATACION_TEMPORAL.consultarDocumentos,
  );
  const puedeConsultarAuditoria = concesionesVisuales.has(
    CAPACIDADES_CONTRATACION_TEMPORAL.consultarAuditoria,
  );
  const disponible = fuenteValida(fuente) && puedeConsultarCuadro;
  let estado = congelar(estadoInicial(disponible));
  let controlador = null;
  let secuencia = 0;
  let desmontado = false;

  function reemplazar(cambios) {
    if (desmontado) return;
    const siguiente = { ...copiar(estado), ...copiar(cambios) };
    if (!VISTAS.has(siguiente.vista) || !ESTADOS_CARGA.has(siguiente.carga)) {
      throw new TypeError("estado de expedientes no válido");
    }
    estado = congelar(siguiente);
  }

  function cancelarEnCurso() {
    secuencia += 1;
    controlador?.abort();
    controlador = null;
  }

  function exigirSinEfectoEnCurso() {
    if (estado.ocupado) throw errorPublico("actuacion_en_curso");
  }

  async function cargar(filtros = estado.filtros) {
    exigirSinEfectoEnCurso();
    if (!disponible || desmontado) {
      reemplazar({
        carga: "denegado",
        mensaje_clave: "estado_denegado",
        tipo_mensaje: "error",
      });
      return estado;
    }
    const filtrosCerrados = filtrosValidos(filtros);
    cancelarEnCurso();
    const operacion = secuencia;
    controlador = new AbortController();
    reemplazar({
      carga: "cargando",
      filtros: filtrosCerrados,
      recibo: null,
      mensaje_clave: "estado_cargando",
      tipo_mensaje: "informacion",
    });
    try {
      const cuadro = validarCuadroContratacionTemporal(await fuente.listar({
        filtros: filtrosCerrados,
        signal: controlador.signal,
      }));
      if (desmontado || operacion !== secuencia) return estado;
      const seleccionVisible = cuadro.expedientes.some(
        ({ expediente_ref: referencia }) => referencia === estado.expediente_ref,
      );
      reemplazar({
        carga: cuadro.expedientes.length === 0 ? "vacio" : "listo",
        vista: seleccionVisible ? estado.vista : "cuadro",
        cuadro,
        expediente: seleccionVisible ? estado.expediente : null,
        documentos: seleccionVisible ? estado.documentos : null,
        auditoria: seleccionVisible ? estado.auditoria : null,
        expediente_ref: seleccionVisible ? estado.expediente_ref : "",
        tarea_ref: seleccionVisible ? estado.tarea_ref : "",
        mensaje_clave: estado.resultado_indeterminado
          ? "estado_resultado_indeterminado"
          : (cuadro.expedientes.length === 0 ? "estado_vacio" : "estado_listo"),
        tipo_mensaje: estado.resultado_indeterminado
          ? "aviso"
          : "informacion",
      });
    } catch (error) {
      if (desmontado || operacion !== secuencia || error?.name === "AbortError") return estado;
      reemplazar({
        carga: "error",
        cuadro: null,
        mensaje_clave: "estado_error_carga",
        tipo_mensaje: "error",
      });
    } finally {
      if (operacion === secuencia) controlador = null;
    }
    return estado;
  }

  async function seleccionarExpediente(expedienteRef, vista = "expediente") {
    exigirSinEfectoEnCurso();
    const vistaDenegada = vista === "documentos" && (
      !puedeConsultarDocumentos || typeof fuente?.obtenerDocumentos !== "function"
    ) || vista === "auditoria" && (
      !puedeConsultarAuditoria || typeof fuente?.obtenerAuditoria !== "function"
    );
    const consultaRealDelegada = estado.cuadro?.demostracion === false;
    if (!disponible || (!puedeConsultarExpediente && !consultaRealDelegada)
      || vistaDenegada || desmontado) {
      reemplazar({
        carga: "denegado",
        mensaje_clave: "estado_denegado_expediente",
        tipo_mensaje: "error",
      });
      return estado;
    }
    if (!VISTAS.has(vista) || !["expediente", "documentos", "auditoria"].includes(vista)
      || typeof expedienteRef !== "string") {
      throw new TypeError("selección de expediente no válida");
    }
    const existe = estado.cuadro?.expedientes.some(
      ({ expediente_ref: referencia }) => referencia === expedienteRef,
    );
    if (!existe) throw new TypeError("expediente fuera del cuadro actual");
    cancelarEnCurso();
    const operacion = secuencia;
    controlador = new AbortController();
    reemplazar({
      vista,
      carga: "cargando",
      expediente: null,
      documentos: null,
      auditoria: null,
      expediente_ref: expedienteRef,
      tarea_ref: "",
      recibo: null,
      mensaje_clave: "estado_cargando_expediente",
      tipo_mensaje: "informacion",
    });
    try {
      const expediente = proyectarAutorizacionVisual(
        validarExpedienteContratacionTemporal(
          await fuente.obtener(expedienteRef, { signal: controlador.signal }),
        ),
        concesionesVisuales,
      );
      if (expediente.expediente_ref !== expedienteRef) {
        throw new TypeError("la proyección no corresponde al expediente solicitado");
      }
      let documentos = null;
      let auditoria = null;
      if (vista === "documentos") {
        documentos = validarDocumentosContratacionTemporal(
          await fuente.obtenerDocumentos(expedienteRef, { signal: controlador.signal }),
        );
        if (documentos.expediente_ref !== expedienteRef
          || documentos.version !== expediente.version) {
          throw new TypeError("el índice documental no corresponde al expediente");
        }
      }
      if (vista === "auditoria") {
        auditoria = validarAuditoriaContratacionTemporal(
          await fuente.obtenerAuditoria(expedienteRef, { signal: controlador.signal }),
        );
        if (auditoria.expediente_ref !== expedienteRef
          || auditoria.version !== expediente.version) {
          throw new TypeError("la auditoría no corresponde al expediente");
        }
      }
      if (desmontado || operacion !== secuencia) return estado;
      const actual = expediente.tareas.find(
        ({ estado_clave: clave }) => ["en_curso", "espera", "incidencia"].includes(clave),
      ) ?? expediente.tareas.at(-1);
      reemplazar({
        carga: "listo",
        expediente,
        documentos,
        auditoria,
        expediente_ref: expediente.expediente_ref,
        tarea_ref: actual?.tarea_ref ?? "",
        actualizacion_pendiente: estado.resultado_indeterminado,
        mensaje_clave: estado.resultado_indeterminado
          ? "estado_resultado_indeterminado"
          : "estado_expediente_listo",
        tipo_mensaje: estado.resultado_indeterminado
          ? "aviso"
          : "informacion",
      });
    } catch (error) {
      if (desmontado || operacion !== secuencia || error?.name === "AbortError") return estado;
      reemplazar({
        carga: "error",
        expediente: null,
        mensaje_clave: "estado_error_expediente",
        tipo_mensaje: "error",
      });
    } finally {
      if (operacion === secuencia) controlador = null;
    }
    return estado;
  }

  function cambiarVista(vista) {
    exigirSinEfectoEnCurso();
    if (!VISTAS.has(vista)) throw new TypeError("vista de contratación temporal no válida");
    if (["expediente", "documentos", "auditoria"].includes(vista) && estado.expediente === null) {
      throw errorPublico("expediente_no_seleccionado");
    }
    cancelarEnCurso();
    reemplazar({
      vista,
      carga: estado.cuadro?.expedientes.length === 0 ? "vacio" : "listo",
      recibo: null,
      mensaje_clave: estado.resultado_indeterminado
        ? "estado_resultado_indeterminado"
        : "estado_listo",
      tipo_mensaje: estado.resultado_indeterminado
        ? "aviso"
        : "informacion",
    });
    return estado;
  }

  function seleccionarTarea(tareaRef) {
    exigirSinEfectoEnCurso();
    if (typeof tareaRef !== "string"
      || !estado.expediente?.tareas.some(({ tarea_ref: referencia }) => referencia === tareaRef)) {
      throw new TypeError("tarea no válida");
    }
    reemplazar({ vista: "expediente", tarea_ref: tareaRef, recibo: null });
    return estado;
  }

  async function ejecutarActuacion({ accionRef, datos = {} } = {}) {
    if (estado.ocupado) return estado;
    if (estado.actualizacion_pendiente) {
      reemplazar({
        mensaje_clave: estado.resultado_indeterminado
          ? "estado_resultado_indeterminado"
          : "estado_actualizacion_pendiente",
        tipo_mensaje: "aviso",
      });
      return estado;
    }
    const expediente = estado.expediente;
    const tarea = expediente?.tareas.find(
      ({ tarea_ref: referencia }) => referencia === estado.tarea_ref,
    );
    const accion = tarea?.acciones.find(
      ({ accion_ref: referencia }) => referencia === accionRef,
    );
    if (!accion || accion.tipo !== "efecto" || accion.disponible !== true
      || !concesionesVisuales.has(accion.capacidad)) {
      reemplazar({
        mensaje_clave: "estado_accion_denegada",
        tipo_mensaje: "error",
      });
      return estado;
    }
    const comando = validarComandoActuacion({
      esquema: "vec.contratacion_temporal.actuacion.v1",
      expediente_ref: expediente.expediente_ref,
      version_esperada: expediente.version,
      tarea_ref: tarea.tarea_ref,
      accion_ref: accion.accion_ref,
      datos,
    });
    cancelarEnCurso();
    const operacion = secuencia;
    controlador = new AbortController();
    reemplazar({
      ocupado: true,
      recibo: null,
      actualizacion_pendiente: false,
      resultado_indeterminado: false,
      mensaje_clave: "estado_registrando_actuacion",
      tipo_mensaje: "informacion",
    });
    try {
      const recibo = validarReciboActuacion(
        await fuente.ejecutar(comando, { signal: controlador.signal }),
      );
      if (desmontado || operacion !== secuencia) return estado;
      if (recibo.expediente_ref !== comando.expediente_ref
        || recibo.numero_visible !== expediente.numero_visible
        || recibo.version !== comando.version_esperada + 1
        || recibo.actuacion !== accion.etiqueta) {
        throw new TypeError("el recibo no corresponde a la actuación solicitada");
      }
      reemplazar({
        ocupado: false,
        recibo,
        actualizacion_pendiente: true,
        mensaje_clave: "estado_confirmada_actualizacion_pendiente",
        tipo_mensaje: "aviso",
      });
      try {
        const actualizado = proyectarAutorizacionVisual(
          validarExpedienteContratacionTemporal(
            await fuente.obtener(expediente.expediente_ref, { signal: controlador.signal }),
          ),
          concesionesVisuales,
        );
        if (actualizado.expediente_ref !== comando.expediente_ref
          || actualizado.numero_visible !== recibo.numero_visible
          || actualizado.version !== recibo.version) {
          throw new TypeError("la actualización no corresponde al recibo confirmado");
        }
        if (desmontado || operacion !== secuencia) return estado;
        reemplazar({
          expediente: actualizado,
          actualizacion_pendiente: false,
          resultado_indeterminado: false,
          mensaje_clave: "estado_actuacion_registrada",
          tipo_mensaje: "exito",
        });
      } catch (error) {
        if (error?.name === "AbortError") throw error;
      }
    } catch (error) {
      if (desmontado || operacion !== secuencia) return estado;
      if (esResultadoIndeterminado(error)) {
        reemplazar({
          ocupado: false,
          recibo: null,
          actualizacion_pendiente: true,
          resultado_indeterminado: true,
          mensaje_clave: "estado_resultado_indeterminado",
          tipo_mensaje: "aviso",
        });
        return estado;
      }
      if (error?.name === "AbortError") return estado;
      reemplazar({
        ocupado: false,
        recibo: null,
        actualizacion_pendiente: false,
        resultado_indeterminado: false,
        mensaje_clave: "estado_error_actuacion",
        tipo_mensaje: "error",
      });
    } finally {
      if (operacion === secuencia) controlador = null;
    }
    return estado;
  }

  function cancelar() {
    if (!estado.ocupado && estado.carga !== "cargando") return estado;
    const efectoEnCurso = estado.ocupado;
    cancelarEnCurso();
    reemplazar({
      ocupado: false,
      carga: estado.cuadro ? "listo" : "inicial",
      actualizacion_pendiente: efectoEnCurso
        ? true
        : estado.actualizacion_pendiente,
      resultado_indeterminado: efectoEnCurso
        ? true
        : estado.resultado_indeterminado,
      mensaje_clave: efectoEnCurso
        ? "estado_cancelado"
        : (estado.resultado_indeterminado
          ? "estado_resultado_indeterminado"
          : "estado_lectura_cancelada"),
      tipo_mensaje: "aviso",
    });
    return estado;
  }

  function desmontar() {
    cancelarEnCurso();
    desmontado = true;
  }

  return Object.freeze({
    cargar,
    seleccionarExpediente,
    cambiarVista,
    seleccionarTarea,
    ejecutarActuacion,
    cancelar,
    desmontar,
    obtenerEstado() {
      return estado;
    },
  });
}
