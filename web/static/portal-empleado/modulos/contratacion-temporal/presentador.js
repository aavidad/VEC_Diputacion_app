import {
  CAPACIDAD_CREAR_SOLICITUD,
  ErrorValidacionAlta,
  LIMITES_ALTA_CONTRATACION,
  catalogosAltaOperables,
  clonarYCongelarAlta,
  crearBorradorAlta,
  crearComandoAlta,
  validarBorradorAlta,
  validarCatalogosAlta,
  validarReciboAlta,
} from "./contrato.js";

const FASE_EDICION = "edicion";
const FASE_REVISION = "revision";
const FASE_ENVIO = "envio";
const FASE_CANCELANDO = "cancelando";
const FASE_RECIBO = "recibo";

function errorPublico(codigo) {
  const error = new Error("La operación de alta no está disponible");
  error.codigo = codigo;
  return error;
}

function generarClaveSegura() {
  if (typeof globalThis.crypto?.randomUUID !== "function") {
    throw errorPublico("generador_no_disponible");
  }
  return globalThis.crypto.randomUUID();
}

function camposBorrador() {
  return Object.keys(crearBorradorAlta());
}

function copiarBorradorEntrada(entrada) {
  if (!entrada || typeof entrada !== "object" || Array.isArray(entrada)) {
    throw new ErrorValidacionAlta({ general: "contrato_cerrado" });
  }
  const campos = camposBorrador();
  const claves = Object.keys(entrada);
  if (claves.length !== campos.length || claves.some((clave) => !campos.includes(clave))
    || campos.some((campo) => !Object.hasOwn(entrada, campo))) {
    throw new ErrorValidacionAlta({ general: "contrato_cerrado" });
  }
  const resultado = {};
  for (const campo of campos) {
    const valor = entrada[campo];
    if (campo === "rc_existe") {
      resultado[campo] = valor;
      continue;
    }
    if (campo === "documentos_adjuntos") {
      if (!Array.isArray(valor)
        || valor.length > LIMITES_ALTA_CONTRATACION.adjuntos + 1) {
        throw new ErrorValidacionAlta({ documentos_adjuntos: "adjuntos" });
      }
      resultado[campo] = [...valor];
      continue;
    }
    if (typeof valor !== "string"
      || [...valor].length > LIMITES_ALTA_CONTRATACION.texto + 1) {
      throw new ErrorValidacionAlta({ [campo]: "texto_opcional" });
    }
    resultado[campo] = valor;
  }
  return clonarYCongelarAlta(resultado);
}

function limpiarDatosRC(borrador) {
  return {
    ...borrador,
    rc_numero: "",
    rc_fecha: "",
    rc_importe: "",
    rc_documento_ref: "",
  };
}

function solicitudesCanonicasIguales(primera, segunda) {
  return JSON.stringify(primera) === JSON.stringify(segunda);
}

function crearEstadoInicial(catalogos, disponible) {
  return clonarYCongelarAlta({
    fase: FASE_EDICION,
    disponible,
    ocupado: false,
    catalogos,
    borrador: crearBorradorAlta(),
    errores: {},
    mensaje_clave: disponible ? "estado_disponible" : "estado_no_disponible",
    tipo_mensaje: disponible ? "informacion" : "aviso",
    recibo: null,
  });
}

/**
 * Coordina el alta sin conocer HTTP ni obtener identidad del navegador.
 * `ejecutor` es el puerto neutral que O2-08 deberá inyectar.
 */
export function crearPresentadorAltaContratacionTemporal({
  catalogos: catalogosSinValidar,
  capacidad,
  ejecutor,
  generarClaveIdempotencia = generarClaveSegura,
} = {}) {
  const catalogos = validarCatalogosAlta(catalogosSinValidar);
  if (typeof generarClaveIdempotencia !== "function") {
    throw new TypeError("generador de clave de operación no válido");
  }
  const disponible = capacidad === CAPACIDAD_CREAR_SOLICITUD
    && typeof ejecutor === "function"
    && catalogosAltaOperables(catalogos);
  let estado = crearEstadoInicial(catalogos, disponible);
  let comandoActual = null;
  let envioActual = null;
  let controladorEnvio = null;
  let cancelacionSolicitada = false;

  function sustituirEstado(cambios) {
    estado = clonarYCongelarAlta({ ...estado, ...cambios });
  }

  function obtenerEstado() {
    return clonarYCongelarAlta(estado);
  }

  function actualizarBorrador(entrada) {
    if (estado.fase !== FASE_EDICION || estado.ocupado) {
      throw errorPublico("edicion_no_disponible");
    }
    let borrador;
    try {
      borrador = copiarBorradorEntrada(entrada);
    } catch (error) {
      const errores = error instanceof ErrorValidacionAlta
        ? error.errores
        : { general: "contrato_cerrado" };
      sustituirEstado({ errores, mensaje_clave: "errores_descripcion", tipo_mensaje: "error" });
      return obtenerEstado();
    }
    if (borrador.centro_ref !== estado.borrador.centro_ref) {
      borrador = clonarYCongelarAlta({ ...borrador, contacto_ref: "" });
    }
    if (borrador.categoria_ref !== estado.borrador.categoria_ref) {
      borrador = clonarYCongelarAlta({ ...borrador, grupo_subgrupo: "" });
    }
    if (borrador.rc_existe === false) borrador = clonarYCongelarAlta(limpiarDatosRC(borrador));
    sustituirEstado({
      borrador,
      errores: {},
      mensaje_clave: disponible ? "estado_disponible" : "estado_no_disponible",
      tipo_mensaje: disponible ? "informacion" : "aviso",
    });
    return obtenerEstado();
  }

  function prepararRevision(entrada = estado.borrador) {
    if (!disponible || estado.ocupado || estado.fase !== FASE_EDICION) {
      throw errorPublico("servicio_no_disponible");
    }
    let borrador;
    try {
      borrador = copiarBorradorEntrada(entrada);
    } catch (error) {
      const errores = error instanceof ErrorValidacionAlta
        ? error.errores
        : { general: "contrato_cerrado" };
      sustituirEstado({ errores, mensaje_clave: "errores_descripcion", tipo_mensaje: "error" });
      return false;
    }
    const validacion = validarBorradorAlta(borrador, catalogos);
    if (!validacion.valido) {
      sustituirEstado({
        borrador,
        errores: validacion.errores,
        mensaje_clave: "errores_descripcion",
        tipo_mensaje: "error",
      });
      return false;
    }
    try {
      if (comandoActual === null) {
        comandoActual = crearComandoAlta(borrador, catalogos, generarClaveIdempotencia());
      } else {
        const candidatoMismaClave = crearComandoAlta(
          borrador,
          catalogos,
          comandoActual.clave_idempotencia,
        );
        comandoActual = solicitudesCanonicasIguales(
          candidatoMismaClave.solicitud,
          comandoActual.solicitud,
        )
          ? candidatoMismaClave
          : crearComandoAlta(borrador, catalogos, generarClaveIdempotencia());
      }
    } catch (error) {
      const errores = error instanceof ErrorValidacionAlta
        ? error.errores
        : { general: "contrato_cerrado" };
      sustituirEstado({ borrador, errores, mensaje_clave: "errores_descripcion", tipo_mensaje: "error" });
      return false;
    }
    sustituirEstado({
      fase: FASE_REVISION,
      borrador,
      errores: {},
      mensaje_clave: "estado_disponible",
      tipo_mensaje: "informacion",
    });
    return true;
  }

  function volverAEdicion() {
    if (estado.ocupado || estado.fase !== FASE_REVISION) {
      throw errorPublico("edicion_no_disponible");
    }
    sustituirEstado({
      fase: FASE_EDICION,
      errores: {},
      mensaje_clave: "estado_disponible",
      tipo_mensaje: "informacion",
    });
  }

  function enviar() {
    if (envioActual) return envioActual;
    if (!disponible || estado.fase !== FASE_REVISION || comandoActual === null) {
      return Promise.reject(errorPublico("servicio_no_disponible"));
    }
    controladorEnvio = new AbortController();
    cancelacionSolicitada = false;
    sustituirEstado({
      fase: FASE_ENVIO,
      ocupado: true,
      mensaje_clave: "estado_enviando",
      tipo_mensaje: "informacion",
    });
    let respuestaRecibida = false;
    const comando = clonarYCongelarAlta(comandoActual);
    const tarea = (async () => {
      try {
        const respuesta = await ejecutor(
          comando,
          Object.freeze({ signal: controladorEnvio.signal }),
        );
        respuestaRecibida = true;
        const recibo = validarReciboAlta(respuesta);
        sustituirEstado({
          fase: FASE_RECIBO,
          ocupado: false,
          recibo,
          errores: {},
          mensaje_clave: "recibo_descripcion",
          tipo_mensaje: "exito",
        });
        comandoActual = null;
        return recibo;
      } catch (_errorPrivado) {
        const canceladaSinRespuesta = cancelacionSolicitada && !respuestaRecibida;
        sustituirEstado({
          fase: FASE_REVISION,
          ocupado: false,
          recibo: null,
          mensaje_clave: canceladaSinRespuesta
            ? "estado_cancelado"
            : (respuestaRecibida ? "estado_recibo_invalido" : "estado_error"),
          tipo_mensaje: canceladaSinRespuesta ? "aviso" : "error",
        });
        return null;
      } finally {
        controladorEnvio = null;
        cancelacionSolicitada = false;
        envioActual = null;
      }
    })();
    envioActual = tarea;
    return tarea;
  }

  function cancelarEnvio() {
    if (!controladorEnvio || !estado.ocupado
      || (estado.fase !== FASE_ENVIO && estado.fase !== FASE_CANCELANDO)) {
      return false;
    }
    cancelacionSolicitada = true;
    controladorEnvio.abort();
    sustituirEstado({
      fase: FASE_CANCELANDO,
      ocupado: true,
      mensaje_clave: "estado_cancelando",
      tipo_mensaje: "aviso",
    });
    return true;
  }

  function desmontar() {
    cancelarEnvio();
  }

  return Object.freeze({
    actualizarBorrador,
    cancelarEnvio,
    desmontar,
    enviar,
    obtenerEstado,
    prepararRevision,
    volverAEdicion,
  });
}
