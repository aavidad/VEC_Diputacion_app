import {
  descomponerReferenciaEstadoBorrador,
  extraerDatosEnvelopeBorradores,
  extraerErrorEnvelopeBorradores,
  validarDetalleBorrador,
  validarETagBorrador,
  validarListaBorradores,
  validarOpcionesBorradores,
  validarReciboGuardadoBorrador,
  validarSelectorListaBorradores,
  validarSolicitudActualizarBorrador,
  validarSolicitudCrearBorrador,
} from "./portal-borradores-contrato.js";
import { traducirPortal } from "./portal-i18n.js?v=20260926-pulido-portal-v1";

export const RUTAS_API_BORRADORES = Object.freeze({
  opciones: "/api/vec/bolsa/convocatorias/borradores/opciones",
  lista: "/api/vec/bolsa/convocatorias/borradores",
  detalle: "/api/vec/bolsa/convocatorias/borradores",
});

const MAXIMO_RESPUESTA_BYTES = 4 * 1024 * 1024;
const MAXIMO_FRAGMENTOS_RESPUESTA = 65_536;
const MAXIMO_ERROR_BYTES = 16 * 1024;
const MAXIMO_FRAGMENTOS_ERROR = 256;
const PATRON_CLAVE_IDEMPOTENCIA = /^[A-Za-z0-9_-]{43}$/;
const ALFABETO_BASE64URL = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_";

/**
 * Matriz mínima soportada por este puerto: Chromium 105+, Firefox 102+ y
 * Safari 16+, siempre con Fetch, AbortSignal, Response.body.getReader(),
 * cancelación de ReadableStream y TextDecoder fatal. La comprobación efectiva
 * es por capacidad y falla cerrada; deliberadamente no existe un fallback que
 * materialice el cuerpo completo.
 */
export const MATRIZ_FETCH_BORRADORES = Object.freeze({
  chromium: ">=105",
  firefox: ">=102",
  safari: ">=16",
  requiere: Object.freeze([
    "Fetch", "AbortSignal", "Response.body.getReader", "ReadableStream.cancel", "TextDecoder.fatal",
  ]),
  fallbackMaterializador: false,
});

export class ErrorAPIBorradores extends Error {
  constructor(mensaje, estado = 0, causa = undefined, detalle = {}) {
    super(mensaje, causa === undefined ? undefined : { cause: causa });
    this.name = "ErrorAPIBorradores";
    this.estado = estado;
    this.codigo = detalle.codigo || "error_cliente_borradores";
    this.correlacion = detalle.correlacion || null;
    this.correlacionRef = this.correlacion;
    this.envelopeValido = detalle.envelopeValido === true;
    this.tipoConflicto = estado === 409 ? "idempotencia" : (estado === 412 ? "cas" : null);
    this.esConflictoIdempotencia = estado === 409;
    this.esConflictoCAS = estado === 412;
    this.esConflicto = this.tipoConflicto !== null;
    this.conservarCambiosLocales = this.esConflicto;
  }
}

function codificarBase64URL(bytes) {
  let salida = "";
  for (let indice = 0; indice < bytes.length; indice += 3) {
    const a = bytes[indice];
    const existeB = indice + 1 < bytes.length;
    const existeC = indice + 2 < bytes.length;
    const b = existeB ? bytes[indice + 1] : 0;
    const c = existeC ? bytes[indice + 2] : 0;
    const bloque = (a << 16) | (b << 8) | c;
    salida += ALFABETO_BASE64URL[(bloque >>> 18) & 63];
    salida += ALFABETO_BASE64URL[(bloque >>> 12) & 63];
    if (existeB) salida += ALFABETO_BASE64URL[(bloque >>> 6) & 63];
    if (existeC) salida += ALFABETO_BASE64URL[bloque & 63];
  }
  return salida;
}

function decodificarBase64URLEstricto(valor) {
  if (typeof valor !== "string" || !PATRON_CLAVE_IDEMPOTENCIA.test(valor)) return null;
  const salida = new Uint8Array(32);
  let acumulador = 0;
  let bits = 0;
  let posicion = 0;
  for (const caracter of valor) {
    const sexteto = ALFABETO_BASE64URL.indexOf(caracter);
    if (sexteto < 0) return null;
    acumulador = (acumulador << 6) | sexteto;
    bits += 6;
    if (bits >= 8) {
      bits -= 8;
      if (posicion >= salida.length) return null;
      salida[posicion] = (acumulador >>> bits) & 0xFF;
      posicion += 1;
      acumulador &= (1 << bits) - 1;
    }
  }
  if (posicion !== salida.length || acumulador !== 0) return null;
  return salida;
}

export function generarClaveIdempotencia(criptografia = globalThis.crypto) {
  if (!criptografia || typeof criptografia.getRandomValues !== "function") {
    throw new ErrorAPIBorradores(traducirPortal("txt_no_esta_disponible_un_generador_criptografico_se"));
  }
  const bytes = new Uint8Array(32);
  criptografia.getRandomValues(bytes);
  const clave = codificarBase64URL(bytes);
  if (decodificarBase64URLEstricto(clave) === null) {
    throw new ErrorAPIBorradores(traducirPortal("txt_no_se_pudo_generar_una_clave_de_idempotencia_can"));
  }
  return clave;
}

export function validarClaveIdempotencia(clave) {
  const bytes = decodificarBase64URLEstricto(clave);
  if (bytes === null || bytes.byteLength !== 32 || codificarBase64URL(bytes) !== clave) {
    throw new ErrorAPIBorradores(traducirPortal("txt_clave_de_idempotencia_no_valida"));
  }
  return clave;
}

function rutaVersionBorrador(referencia) {
  let partes;
  try {
    partes = descomponerReferenciaEstadoBorrador(referencia);
  } catch (error) {
    throw new ErrorAPIBorradores(traducirPortal("txt_referencia_de_borrador_no_valida"), 0, error);
  }
  return `${RUTAS_API_BORRADORES.detalle}/${encodeURIComponent(partes.id)}/versiones/${partes.secuencia}`;
}

function mensajeEstado(estado) {
  if (estado === 401) return traducirPortal("txt_se_requiere_autenticacion_interna");
  if (estado === 403) return traducirPortal("txt_la_sesion_no_dispone_de_autorizacion_para_esta_o");
  if (estado === 404) return traducirPortal("txt_el_borrador_solicitado_no_existe_o_no_es_visible");
  if (estado === 409) return traducirPortal("txt_la_clave_de_idempotencia_ya_se_utilizo_para_otra");
  if (estado === 412) return traducirPortal("txt_el_borrador_cambio_en_el_servidor_se_han_conserv");
  if (estado === 413) return traducirPortal("txt_la_solicitud_o_la_respuesta_supera_el_limite_adm");
  if (estado === 422) return traducirPortal("txt_el_servidor_rechazo_el_contenido_del_borrador");
  if (estado >= 500) return traducirPortal("txt_el_servicio_de_borradores_no_esta_disponible_tem");
  return traducirPortal("txt_la_api_de_borradores_rechazo_la_operacion_http", { estado: estado });
}

function longitudDeclarada(respuesta) {
  const valor = respuesta.headers?.get?.("content-length");
  if (valor === null || valor === undefined || valor === "") return null;
  if (!/^(?:0|[1-9][0-9]*)$/.test(valor)) {
    throw new ErrorAPIBorradores(traducirPortal("txt_content_length_no_canonico"), respuesta.status);
  }
  const longitud = Number(valor);
  if (!Number.isSafeInteger(longitud) || longitud > MAXIMO_RESPUESTA_BYTES) {
    throw new ErrorAPIBorradores(traducirPortal("txt_la_respuesta_de_la_api_supera_el_limite_admitido"), 413);
  }
  return longitud;
}

function validarSignal(signal) {
  if (signal === undefined || signal === null) return undefined;
  if (typeof signal !== "object" || typeof signal.aborted !== "boolean"
    || typeof signal.addEventListener !== "function"
    || typeof signal.removeEventListener !== "function") {
    throw new ErrorAPIBorradores(traducirPortal("txt_abortsignal_no_valido"));
  }
  return signal;
}

function errorAborto(signal) {
  let causa;
  try {
    causa = signal?.reason;
  } catch {
    causa = undefined;
  }
  if (!(causa instanceof Error)) {
    causa = typeof DOMException === "function"
      ? new DOMException(traducirPortal("txt_la_operacion_fue_cancelada"), "AbortError")
      : new Error("La operación fue cancelada.");
  }
  return new ErrorAPIBorradores(
    traducirPortal("txt_la_operacion_de_borradores_fue_cancelada"), 0, causa, { codigo: "operacion_abortada" },
  );
}

async function ejecutarAbortable(
  iniciar, signal, alAbortar = null, alResolverTardio = null,
) {
  if (signal === undefined) return iniciar();
  return new Promise((resolve, reject) => {
    let finalizada = false;
    const limpiar = () => signal.removeEventListener("abort", abortar);
    const abortar = () => {
      if (finalizada) return;
      finalizada = true;
      limpiar();
      if (alAbortar !== null) {
        try { Promise.resolve(alAbortar()).catch(() => {}); } catch { /* mejor esfuerzo */ }
      }
      reject(errorAborto(signal));
    };
    signal.addEventListener("abort", abortar, { once: true });
    if (signal.aborted) {
      abortar();
      return;
    }
    let operacion;
    try {
      operacion = iniciar();
    } catch (error) {
      finalizada = true;
      limpiar();
      reject(error);
      return;
    }
    Promise.resolve(operacion).then(
      (valor) => {
        if (finalizada) {
          if (alResolverTardio !== null) {
            try { Promise.resolve(alResolverTardio(valor)).catch(() => {}); } catch { /* mejor esfuerzo */ }
          }
          return;
        }
        finalizada = true;
        limpiar();
        resolve(valor);
      },
      (error) => {
        if (finalizada) return;
        finalizada = true;
        limpiar();
        reject(error);
      },
    );
  });
}

async function cancelarRespuesta(respuesta, lector = null, motivo = traducirPortal("txt_respuesta_rechazada")) {
  const cancelar = lector !== null && typeof lector.cancel === "function"
    ? () => lector.cancel(motivo)
    : (respuesta?.body && typeof respuesta.body.cancel === "function"
      ? () => respuesta.body.cancel(motivo)
      : null);
  if (cancelar === null) return;
  try { await cancelar(); } catch { /* nunca sustituye al error causal */ }
}

async function leerTextoAcotado(
  respuesta,
  signal,
  maximoBytes = MAXIMO_RESPUESTA_BYTES,
  maximoFragmentos = MAXIMO_FRAGMENTOS_RESPUESTA,
) {
  const declarada = longitudDeclarada(respuesta);
  if (declarada !== null && declarada > maximoBytes) {
    throw new ErrorAPIBorradores(traducirPortal("txt_la_respuesta_de_la_api_supera_el_limite_admitido"), 413);
  }
  if (!respuesta.body || typeof respuesta.body.getReader !== "function") {
    // Fail-closed: la matriz soportada exige streaming; no se materializa el cuerpo.
    throw new ErrorAPIBorradores(traducirPortal("txt_la_respuesta_no_permite_una_lectura_incremental"), respuesta.status);
  }
  const lector = respuesta.body.getReader();
  if (!lector || typeof lector.read !== "function" || typeof lector.cancel !== "function"
    || typeof lector.releaseLock !== "function") {
    throw new ErrorAPIBorradores(traducirPortal("txt_el_lector_de_respuesta_no_respeta_la_matriz_fetc"), respuesta.status);
  }

  const fragmentos = [];
  let total = 0;
  let numeroFragmentos = 0;
  try {
    while (true) {
      const lectura = await ejecutarAbortable(
        () => lector.read(), signal, () => lector.cancel(traducirPortal("txt_operacion_cancelada")),
      );
      if (!lectura || typeof lectura.done !== "boolean") {
        throw new ErrorAPIBorradores(traducirPortal("txt_el_lector_devolvio_un_estado_no_valido"), respuesta.status);
      }
      if (lectura.done) break;
      numeroFragmentos += 1;
      if (numeroFragmentos > maximoFragmentos) {
        throw new ErrorAPIBorradores(traducirPortal("txt_el_flujo_de_respuesta_contiene_demasiados_fragme"), 413);
      }
      if (!(lectura.value instanceof Uint8Array)) {
        throw new ErrorAPIBorradores(traducirPortal("txt_el_flujo_de_respuesta_no_contiene_bytes_validos"), respuesta.status);
      }
      if (total + lectura.value.byteLength > maximoBytes) {
        throw new ErrorAPIBorradores(traducirPortal("txt_la_respuesta_de_la_api_supera_el_limite_admitido"), 413);
      }
      total += lectura.value.byteLength;
      fragmentos.push(lectura.value);
    }
    if (total === 0) throw new ErrorAPIBorradores(traducirPortal("txt_la_respuesta_json_esta_vacia"), respuesta.status);
    const bytes = new Uint8Array(total);
    let desplazamiento = 0;
    for (const fragmento of fragmentos) {
      bytes.set(fragmento, desplazamiento);
      desplazamiento += fragmento.byteLength;
    }
    return new TextDecoder("utf-8", { fatal: true }).decode(bytes);
  } catch (error) {
    await cancelarRespuesta(respuesta, lector, traducirPortal("txt_lectura_rechazada"));
    if (error instanceof ErrorAPIBorradores) throw error;
    const mensaje = error instanceof TypeError && total > 0
      ? traducirPortal("txt_la_respuesta_no_contiene_utf_8_valido")
      : traducirPortal("txt_no_se_pudo_leer_el_flujo_de_respuesta");
    throw new ErrorAPIBorradores(mensaje, respuesta.status, error);
  } finally {
    try { lector.releaseLock(); } catch { /* cancelado o lectura aún liquidándose */ }
  }
}

async function leerJSONCerrado(
  respuesta, signal, maximoBytes = MAXIMO_RESPUESTA_BYTES,
  maximoFragmentos = MAXIMO_FRAGMENTOS_RESPUESTA,
) {
  const tipo = respuesta.headers?.get?.("content-type") || "";
  if (!/^application\/(?:[a-z0-9.+-]*\+)?json(?:\s*;|$)/i.test(tipo)) {
    throw new ErrorAPIBorradores(traducirPortal("txt_la_api_no_respondio_con_json"), respuesta.status);
  }
  const texto = await leerTextoAcotado(respuesta, signal, maximoBytes, maximoFragmentos);
  try {
    return JSON.parse(texto);
  } catch (error) {
    throw new ErrorAPIBorradores(traducirPortal("txt_la_api_devolvio_un_json_no_valido"), respuesta.status, error);
  }
}

function comprobarETagRespuesta(respuesta, resultado) {
  let etagCabecera;
  try {
    etagCabecera = respuesta.headers?.get?.("etag");
  } catch (error) {
    throw new ErrorAPIBorradores(traducirPortal("txt_no_se_pudo_leer_la_cabecera_etag"), respuesta.status, error);
  }
  if (etagCabecera === null || etagCabecera === undefined || etagCabecera === "") {
    throw new ErrorAPIBorradores(traducirPortal("txt_la_respuesta_no_incluye_el_etag_obligatorio"), respuesta.status);
  }
  let validado;
  try {
    validado = validarETagBorrador(etagCabecera, resultado.referencia_estado);
  } catch (error) {
    throw new ErrorAPIBorradores(traducirPortal("txt_la_cabecera_etag_no_es_canonica"), respuesta.status, error);
  }
  if (validado !== resultado.etag) {
    throw new ErrorAPIBorradores(traducirPortal("txt_el_etag_de_cabecera_no_coincide_con_el_contrato"), respuesta.status);
  }
}

async function construirErrorRespuesta(respuesta, signal) {
  let detalle;
  try {
    detalle = extraerErrorEnvelopeBorradores(await leerJSONCerrado(
      respuesta, signal, MAXIMO_ERROR_BYTES, MAXIMO_FRAGMENTOS_ERROR,
    ));
  } catch (error) {
    if (error instanceof ErrorAPIBorradores && error.codigo === "operacion_abortada") throw error;
    throw new ErrorAPIBorradores(
      traducirPortal("txt_la_respuesta_de_error_no_respeta_el_contrato_cer"),
      respuesta.status,
      error,
      { codigo: "respuesta_error_no_valida" },
    );
  }
  return new ErrorAPIBorradores(
    mensajeEstado(respuesta.status),
    respuesta.status,
    undefined,
    {
      codigo: detalle.codigo,
      correlacion: detalle.correlacion_ref,
      envelopeValido: true,
    },
  );
}

function validarEntrada(validar, mensaje) {
  try {
    return validar();
  } catch (error) {
    throw new ErrorAPIBorradores(mensaje, 0, error);
  }
}

function validarOpcionesCliente(valor, campos, nombre) {
  const opciones = valor === undefined ? {} : valor;
  if (opciones === null || typeof opciones !== "object" || Array.isArray(opciones)
    || Object.keys(opciones).some((campo) => !campos.includes(campo))) {
    throw new ErrorAPIBorradores(`${nombre} no respeta el contrato del cliente.`);
  }
  return opciones;
}

function comprobarLocationCreacion(respuesta, resultado) {
  let location;
  try {
    location = respuesta.headers?.get?.("location");
  } catch (error) {
    throw new ErrorAPIBorradores(traducirPortal("txt_no_se_pudo_leer_la_cabecera_location"), respuesta.status, error);
  }
  const esperada = rutaVersionBorrador(resultado.referencia_estado.referencia);
  if (location !== esperada) {
    throw new ErrorAPIBorradores(traducirPortal("txt_la_cabecera_location_no_corresponde_al_borrador"), respuesta.status);
  }
}

/**
 * Crea el cliente del puerto HTTP interno.
 *
 * La identidad procede exclusivamente del canal interno autenticado. El
 * contrato de configuración es cerrado y no admite proveedores de tokens,
 * cookies ni cabeceras de identidad construidas por el navegador.
 */
export function crearClienteBorradores(configuracion = {}) {
  if (configuracion === null || typeof configuracion !== "object" || Array.isArray(configuracion)
    || Object.keys(configuracion).some((campo) => !["fetchImpl", "HeadersImpl"].includes(campo))) {
    throw new TypeError("configuración del cliente de borradores no válida");
  }
  const {
    fetchImpl = globalThis.fetch,
    HeadersImpl = globalThis.Headers,
  } = configuracion;
  if (typeof fetchImpl !== "function") throw new TypeError("fetchImpl es obligatorio");
  if (typeof HeadersImpl !== "function") throw new TypeError("HeadersImpl es obligatorio");

  async function ejecutar(
    ruta, opciones, estadosEsperados, validar, exigeETag = false, exigeLocationCreacion = false,
  ) {
    const signal = validarSignal(opciones.signal);
    let headers;
    try {
      headers = new HeadersImpl(opciones.headers || {});
      headers.set("Accept", "application/json");
    } catch (error) {
      if (error instanceof ErrorAPIBorradores) throw error;
      if (signal?.aborted) throw errorAborto(signal);
      throw new ErrorAPIBorradores(traducirPortal("txt_no_se_pudieron_construir_las_cabeceras_seguras"), 0, error);
    }
    let respuesta;
    try {
      respuesta = await ejecutarAbortable(
        () => fetchImpl(ruta, {
          ...opciones,
          signal,
          headers,
          credentials: "same-origin",
          mode: "same-origin",
          cache: "no-store",
          redirect: "error",
          referrerPolicy: "no-referrer",
        }),
        signal,
        null,
        (respuestaTardia) => cancelarRespuesta(respuestaTardia, null, traducirPortal("txt_fetch_cancelado")),
      );
    } catch (error) {
      if (error instanceof ErrorAPIBorradores) throw error;
      if (signal?.aborted) throw errorAborto(signal);
      throw new ErrorAPIBorradores(traducirPortal("txt_no_se_pudo_contactar_con_la_api_de_borradores"), 0, error);
    }
    try {
      if (!respuesta || !Number.isInteger(respuesta.status)
        || respuesta.status < 200 || respuesta.status > 599) {
        throw new ErrorAPIBorradores(traducirPortal("txt_fetch_devolvio_una_respuesta_no_valida"));
      }
      if (!estadosEsperados.includes(respuesta.status)) {
        if (respuesta.status >= 400) throw await construirErrorRespuesta(respuesta, signal);
        throw new ErrorAPIBorradores(mensajeEstado(respuesta.status), respuesta.status);
      }
      let validado;
      try {
        const datos = extraerDatosEnvelopeBorradores(await leerJSONCerrado(respuesta, signal));
        validado = validar(datos);
      } catch (error) {
        if (error instanceof ErrorAPIBorradores) throw error;
        throw new ErrorAPIBorradores(
          traducirPortal("txt_la_respuesta_no_respeta_el_contrato_de_borradore"), respuesta.status, error,
        );
      }
      if (exigeETag) comprobarETagRespuesta(respuesta, validado);
      if (exigeLocationCreacion) comprobarLocationCreacion(respuesta, validado);
      return validado;
    } catch (error) {
      await cancelarRespuesta(respuesta, null, traducirPortal("txt_salida_rechazada"));
      if (error instanceof ErrorAPIBorradores) throw error;
      throw new ErrorAPIBorradores(
        traducirPortal("txt_la_respuesta_no_respeta_el_contrato_de_borradore"), respuesta?.status || 0, error,
      );
    }
  }

  async function obtenerOpciones(opcionesCliente) {
    const { signal } = validarOpcionesCliente(opcionesCliente, ["signal"], "Opciones de consulta");
    return ejecutar(RUTAS_API_BORRADORES.opciones, { method: "GET", signal }, [200], validarOpcionesBorradores);
  }

  async function listar(opcionesCliente) {
    const {
      limite = 40, cursor, texto, categoria, signal,
    } = validarOpcionesCliente(
      opcionesCliente, ["limite", "cursor", "texto", "categoria", "signal"], "Opciones de listado",
    );
    const entradaSelector = { limite };
    if (cursor !== undefined) entradaSelector.cursor = cursor;
    if (texto !== undefined) entradaSelector.texto = texto;
    if (categoria !== undefined) entradaSelector.categoria = categoria;
    const selectorSolicitado = validarEntrada(
      () => validarSelectorListaBorradores(entradaSelector),
      "Selector de listado no válido.",
    );
    const parametros = new URLSearchParams({ limite: String(selectorSolicitado.limite) });
    if (selectorSolicitado.cursor !== undefined) parametros.set("cursor", selectorSolicitado.cursor);
    if (selectorSolicitado.texto !== undefined) parametros.set("texto", selectorSolicitado.texto);
    if (selectorSolicitado.categoria !== undefined) parametros.set("categoria", selectorSolicitado.categoria);
    return ejecutar(
      `${RUTAS_API_BORRADORES.lista}?${parametros}`,
      { method: "GET", signal },
      [200],
      (datos) => {
        const resultado = validarListaBorradores(datos);
        if (Object.keys(resultado.selector).length !== Object.keys(selectorSolicitado).length
          || Object.entries(selectorSolicitado).some(([campo, valor]) => resultado.selector[campo] !== valor)) {
          throw new Error("la lista no corresponde al selector solicitado");
        }
        return resultado;
      },
    );
  }

  async function obtenerDetalle(referencia, limites, opcionesCliente) {
    const { signal } = validarOpcionesCliente(opcionesCliente, ["signal"], "Opciones de detalle");
    const ruta = rutaVersionBorrador(referencia);
    return ejecutar(
      ruta, { method: "GET", signal }, [200],
      (datos) => {
        const resultado = validarDetalleBorrador(datos, limites);
        if (resultado.referencia_estado.referencia !== referencia) {
          throw new Error("el detalle no corresponde al borrador solicitado");
        }
        return resultado;
      }, true,
    );
  }

  async function crear(solicitud, limites, opcionesCliente) {
    const { claveIdempotencia, signal } = validarOpcionesCliente(
      opcionesCliente, ["claveIdempotencia", "signal"], "Opciones de alta",
    );
    const cuerpo = validarEntrada(
      () => validarSolicitudCrearBorrador(solicitud, limites),
      "La solicitud de alta no respeta el contrato.",
    );
    const clave = validarEntrada(
      () => validarClaveIdempotencia(claveIdempotencia),
      "El alta requiere una clave de idempotencia explícita.",
    );
    return ejecutar(
      RUTAS_API_BORRADORES.lista,
      {
        method: "POST",
        signal,
        headers: {
          "Content-Type": "application/json",
          "Idempotency-Key": clave,
        },
        body: JSON.stringify({ data: cuerpo }),
      },
      [201],
      (datos) => {
        const resultado = validarReciboGuardadoBorrador(datos);
        if (resultado.accion !== "crear") throw new Error("el recibo no corresponde al alta solicitada");
        return resultado;
      },
      true,
      true,
    );
  }

  async function actualizar(referencia, solicitud, limites, opcionesCliente) {
    const { etag, claveIdempotencia, signal } = validarOpcionesCliente(
      opcionesCliente, ["etag", "claveIdempotencia", "signal"], "Opciones de actualización",
    );
    const ruta = rutaVersionBorrador(referencia);
    const cuerpo = validarEntrada(
      () => validarSolicitudActualizarBorrador(solicitud, limites),
      "La actualización no respeta el contrato.",
    );
    const etagValidado = validarEntrada(
      () => validarETagBorrador(etag),
      "La actualización requiere un ETag canónico.",
    );
    const clave = validarEntrada(
      () => validarClaveIdempotencia(claveIdempotencia),
      "La actualización requiere una clave de idempotencia explícita.",
    );
    return ejecutar(
      ruta,
      {
        method: "PUT",
        signal,
        headers: {
          "Content-Type": "application/json",
          "Idempotency-Key": clave,
          "If-Match": etagValidado,
        },
        body: JSON.stringify({ data: cuerpo }),
      },
      [200],
      (datos) => {
        const resultado = validarReciboGuardadoBorrador(datos);
        if (resultado.accion !== "actualizar"
          || resultado.referencia_estado.referencia !== referencia) {
          throw new Error("el recibo no corresponde a la actualización solicitada");
        }
        return resultado;
      },
      true,
    );
  }

  return Object.freeze({ actualizar, crear, listar, obtenerDetalle, obtenerOpciones });
}
