import {
  CAPACIDAD_CREAR_SOLICITUD,
  ErrorValidacionAlta,
  LIMITES_ALTA_CONTRATACION,
  catalogosAltaOperables,
  clonarYCongelarAlta,
  crearBorradorAlta,
  crearComandoAlta,
  ESQUEMA_CATALOGOS_NECESIDADES,
  numeroExpedienteMOADValido,
  validarBorradorAlta,
  validarCatalogosAlta,
  validarReciboAlta,
} from "./contrato.js?v=20261009-centro-campos-cohorte-v4";

const FASE_EDICION = "edicion";
const FASE_REVISION = "revision";
const FASE_ENVIO = "envio";
const FASE_CANCELANDO = "cancelando";
const FASE_RECIBO = "recibo";
const FASE_PENDIENTE = "pendiente";

function errorPublico(codigo) {
  const error = new Error("La operación de alta no está disponible");
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

function esNumeroMOADRechazado(error) {
  try {
    return error instanceof Error && error.envelopeValido === true
      && error.estado === 422 && error.codigo === "contenido_no_valido"
      && error.campo === "numero_expediente_moad";
  } catch {
    return false;
  }
}

function generarClaveSegura() {
  if (typeof globalThis.crypto?.randomUUID !== "function") {
    throw errorPublico("generador_no_disponible");
  }
  return globalThis.crypto.randomUUID();
}

function camposBorrador(catalogos) {
  return Object.keys(crearBorradorAlta({ conNumeroMOAD: true,
    conNecesidad: catalogos.esquema === ESQUEMA_CATALOGOS_NECESIDADES,
    jornadaReferenciaMinutos: catalogos.necesidades?.jornada_referencia_minutos }));
}

function copiarBorradorEntrada(entrada, catalogos) {
  if (!entrada || typeof entrada !== "object" || Array.isArray(entrada)) {
    throw new ErrorValidacionAlta({ general: "contrato_cerrado" });
  }
  const campos = camposBorrador(catalogos);
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
    borrador: crearBorradorAlta({ conNumeroMOAD: true,
      conNecesidad: catalogos.esquema === ESQUEMA_CATALOGOS_NECESIDADES,
      jornadaReferenciaMinutos: catalogos.necesidades?.jornada_referencia_minutos }),
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
  let catalogos = validarCatalogosAlta(catalogosSinValidar);
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
  let controladorRefresco = null;
  let cancelacionSolicitada = false;
  let numeroMOADRechazado = null;
  let refrescoCatalogosPendiente = false;

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
      borrador = copiarBorradorEntrada(entrada, catalogos);
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
    if (catalogos.esquema === ESQUEMA_CATALOGOS_NECESIDADES
      && borrador.motivo_clave !== estado.borrador.motivo_clave) {
      const causa = catalogos.necesidades.causas.find((dato) => dato.clave === borrador.motivo_clave);
      borrador = clonarYCongelarAlta(Object.fromEntries(Object.entries(borrador).map(([campo, valor]) =>
        [campo, campo === "jornada_minutos" || !estado.borrador[campo]
          || !catalogos.necesidades.causas.some((dato) => dato.campos_permitidos.includes(campo))
          || causa?.campos_permitidos.includes(campo) ? valor : ""])));
    }
    if (borrador.rc_existe === false) borrador = clonarYCongelarAlta(limpiarDatosRC(borrador));
    if (numeroMOADRechazado !== null
      && borrador.numero_expediente_moad !== estado.borrador.numero_expediente_moad) {
      refrescoCatalogosPendiente = true;
    }
    const numeroSinCorregir = numeroMOADRechazado !== null
      && borrador.numero_expediente_moad === numeroMOADRechazado;
    const numeroConFormatoInvalido = numeroMOADRechazado !== null
      && !numeroExpedienteMOADValido(borrador.numero_expediente_moad);
    const errorNumero = numeroSinCorregir || numeroConFormatoInvalido;
    sustituirEstado({
      borrador,
      errores: errorNumero ? { numero_expediente_moad: "numero_moad_formato" } : {},
      mensaje_clave: errorNumero ? "estado_numero_moad_no_valido"
        : refrescoCatalogosPendiente ? "estado_catalogo_numero_pendiente"
          : disponible ? "estado_disponible" : "estado_no_disponible",
      tipo_mensaje: errorNumero ? "error" : disponible ? "informacion" : "aviso",
    });
    return obtenerEstado();
  }

  function prepararRevision(entrada = estado.borrador) {
    if (!disponible || estado.ocupado || estado.fase !== FASE_EDICION) {
      throw errorPublico("servicio_no_disponible");
    }
    let borrador;
    try {
      borrador = copiarBorradorEntrada(entrada, catalogos);
    } catch (error) {
      const errores = error instanceof ErrorValidacionAlta
        ? error.errores
        : { general: "contrato_cerrado" };
      sustituirEstado({ errores, mensaje_clave: "errores_descripcion", tipo_mensaje: "error" });
      return false;
    }
    if (numeroMOADRechazado !== null && borrador.numero_expediente_moad === numeroMOADRechazado) {
      sustituirEstado({ borrador, errores: { numero_expediente_moad: "numero_moad_formato" },
        mensaje_clave: "estado_numero_moad_no_valido", tipo_mensaje: "error" });
      return false;
    }
    if (numeroMOADRechazado !== null && !numeroExpedienteMOADValido(borrador.numero_expediente_moad)) {
      sustituirEstado({ borrador, errores: { numero_expediente_moad: "numero_moad_formato" },
        mensaje_clave: "estado_numero_moad_no_valido", tipo_mensaje: "error" });
      return false;
    }
    if (refrescoCatalogosPendiente) {
      sustituirEstado({ borrador, errores: { general: "catalogo_no_actualizado" },
        mensaje_clave: "estado_catalogo_numero_pendiente", tipo_mensaje: "aviso" });
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
        comandoActual = solicitudesCanonicasIguales(candidatoMismaClave, comandoActual)
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
      errores: estado.errores,
      mensaje_clave: Object.keys(estado.errores).length ? "errores_descripcion" : "estado_disponible",
      tipo_mensaje: Object.keys(estado.errores).length ? "error" : "informacion",
    });
  }

  function necesitaRefrescoCatalogos() {
    return refrescoCatalogosPendiente && numeroMOADRechazado !== null
      && estado.borrador.numero_expediente_moad !== numeroMOADRechazado
      && numeroExpedienteMOADValido(estado.borrador.numero_expediente_moad);
  }

  async function refrescarCatalogos(obtenerActuales) {
    if (estado.fase !== FASE_EDICION || estado.ocupado || !necesitaRefrescoCatalogos()) {
      return false;
    }
    const controlador = new AbortController();
    controladorRefresco = controlador;
    sustituirEstado({ ocupado: true, errores: {},
      mensaje_clave: "estado_catalogo_numero_cargando", tipo_mensaje: "informacion" });
    try {
      if (typeof obtenerActuales !== "function") throw new TypeError("catálogo no disponible");
      const nuevos = validarCatalogosAlta(await obtenerActuales({ signal: controlador.signal }));
      if (controlador.signal.aborted || controladorRefresco !== controlador) return false;
      if (nuevos.esquema !== catalogos.esquema || !catalogosAltaOperables(nuevos)) {
        throw new TypeError("catálogo incompatible");
      }
      catalogos = nuevos;
      refrescoCatalogosPendiente = false;
      sustituirEstado({ ocupado: false, catalogos: nuevos, errores: {},
        mensaje_clave: "estado_disponible", tipo_mensaje: "informacion" });
      return true;
    } catch {
      if (controlador.signal.aborted || controladorRefresco !== controlador) return false;
      sustituirEstado({ ocupado: false, errores: { general: "catalogo_no_actualizado" },
        mensaje_clave: "estado_catalogo_numero_error", tipo_mensaje: "error" });
      return false;
    } finally {
      if (controladorRefresco === controlador) controladorRefresco = null;
    }
  }

  function enviar() {
    if (envioActual) return envioActual;
    if (!disponible || estado.fase !== FASE_REVISION || comandoActual === null
      || Object.keys(estado.errores).length !== 0 || refrescoCatalogosPendiente
      || comandoActual.numero_expediente_moad === numeroMOADRechazado) {
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
        if (recibo.numero_visible !== comando.numero_expediente_moad) {
          throw errorPublico("recibo_numero_no_coincide");
        }
        sustituirEstado({
          fase: FASE_RECIBO,
          ocupado: false,
          recibo,
          errores: {},
          mensaje_clave: "recibo_descripcion",
          tipo_mensaje: "exito",
        });
        comandoActual = null;
        numeroMOADRechazado = null;
        refrescoCatalogosPendiente = false;
        return recibo;
      } catch (_errorPrivado) {
        const resultadoIndeterminado =
          esResultadoIndeterminado(_errorPrivado);
        const numeroMOADInvalido = !resultadoIndeterminado
          && esNumeroMOADRechazado(_errorPrivado);
        if (numeroMOADInvalido) {
          numeroMOADRechazado = comando.numero_expediente_moad;
          refrescoCatalogosPendiente = true;
        }
        const canceladaSinRespuesta = cancelacionSolicitada && !respuestaRecibida;
        sustituirEstado({
          fase: resultadoIndeterminado ? FASE_PENDIENTE : FASE_REVISION,
          ocupado: false,
          recibo: null,
          errores: numeroMOADInvalido ? { numero_expediente_moad: "numero_moad_formato" } : {},
          mensaje_clave: resultadoIndeterminado
            ? "estado_operacion_pendiente"
            : canceladaSinRespuesta
            ? "estado_cancelado"
            : (respuestaRecibida ? "estado_recibo_invalido"
              : numeroMOADInvalido ? "estado_numero_moad_no_valido" : "estado_error"),
          tipo_mensaje: resultadoIndeterminado || canceladaSinRespuesta
            ? "aviso"
            : "error",
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
    controladorRefresco?.abort();
  }

  return Object.freeze({
    actualizarBorrador,
    cancelarEnvio,
    desmontar,
    enviar,
    necesitaRefrescoCatalogos,
    obtenerEstado,
    prepararRevision,
    refrescarCatalogos,
    tieneRechazoNumero: () => numeroMOADRechazado !== null,
    volverAEdicion,
  });
}
