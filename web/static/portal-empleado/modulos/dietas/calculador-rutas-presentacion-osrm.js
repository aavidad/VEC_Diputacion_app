/**
 * Adaptador de presentacion al OSRM interno real.
 *
 * La sesion sigue siendo DEMO y el resultado nunca es liquidable ni produce
 * efectos administrativos. Sustituye el cálculo sintético por distancias y
 * geometría del grafo on-premise. El navegador accede a un mediador
 * same-origin de ruta fija; no conoce ni puede elegir el host de OSRM ni
 * proyecta coordenadas en la URL o en registros de acceso del proxy.
 */

import {
  CAPACIDAD_CONSULTAR_RUTA,
  ErrorServicioRutasDietas,
  ESQUEMA_CALCULO_RUTA_DIETAS,
  ESQUEMA_GEOMETRIA_RUTA_DIETAS,
  exigirContextoActorDietas,
  tieneCapacidadDietas,
  validarCalculoRutaDietas,
  validarCapacidadesDietas,
  validarSolicitudRutaDietas,
} from "./contrato.js";
import {
  obtenerCatalogoRutasProvincial,
  resolverPuntosRutasProvincial,
} from "./catalogo-rutas-provincial.js";

const RUTA_MEDIADOR = "/api/presentacion/cartografia/rutas";
const TIEMPO_ESPERA_MS = 12_000;
const MAXIMO_RESPUESTA_BYTES = 8 * 1024 * 1024;
const MAXIMO_FRAGMENTOS_RESPUESTA = 8_192;
const MAXIMO_COORDENADAS_ENTRADA = 50_000;
const MAXIMO_PUNTOS_TRAZADO = 2_000;

const CAMPOS_OPCIONES = new Set([
  "contextoActor", "capacidades", "fetchImpl", "tiempoEsperaMs",
]);
const CAMPOS_OPCIONES_PETICION = new Set(["signal"]);

function esObjetoPlano(valor) {
  if (!valor || typeof valor !== "object" || Array.isArray(valor)) return false;
  const prototipo = Object.getPrototypeOf(valor);
  return prototipo === Object.prototype || prototipo === null;
}

function exigirOpciones(valor, campos, nombre) {
  if (!esObjetoPlano(valor) || Object.keys(valor).some((campo) => !campos.has(campo))) {
    throw new TypeError(`${nombre} no validas`);
  }
}

function textoCanonico(valor, nombre, maximo) {
  if (typeof valor !== "string" || valor.length === 0 || valor.length > maximo
    || valor !== valor.trim() || /[\u0000-\u001F\u007F-\u009F]/u.test(valor)) {
    throw new TypeError(`${nombre} no valido`);
  }
  return valor;
}

function versionGrafoGobernada(valor) {
  const version = textoCanonico(valor, "version gobernada del grafo", 100);
  if (!/^[A-Za-z0-9][A-Za-z0-9._:-]{7,99}$/u.test(version)) {
    throw new TypeError("version gobernada del grafo no valida");
  }
  return version;
}

function obtenerVersionGrafoRespuesta(respuesta) {
  const versionGrafo = respuesta.graph_version === undefined
    ? null : versionGrafoGobernada(respuesta.graph_version);
  const versionDatos = respuesta.data_version === undefined
    ? null : versionGrafoGobernada(respuesta.data_version);
  if (versionGrafo === null && versionDatos === null) {
    throw new TypeError("la respuesta de OSRM no acredita una version gobernada del grafo");
  }
  if (versionGrafo !== null && versionDatos !== null && versionGrafo !== versionDatos) {
    throw new TypeError("la respuesta de OSRM declara versiones de grafo contradictorias");
  }
  return versionGrafo ?? versionDatos;
}

function numeroAcotado(valor, minimo, maximo, nombre) {
  if (typeof valor !== "number" || !Number.isFinite(valor) || valor < minimo || valor > maximo) {
    throw new TypeError(`${nombre} no valido`);
  }
  return valor;
}

function enteroAcotado(valor, minimo, maximo, nombre) {
  if (!Number.isSafeInteger(valor) || valor < minimo || valor > maximo) {
    throw new TypeError(`${nombre} no valido`);
  }
  return valor;
}

function exigirAbortSignal(signal) {
  if (signal === undefined || signal === null) return null;
  if (typeof signal !== "object" || typeof signal.aborted !== "boolean"
    || typeof signal.addEventListener !== "function"
    || typeof signal.removeEventListener !== "function") {
    throw new TypeError("AbortSignal no valido");
  }
  return signal;
}

function opcionesPeticion(opciones = {}) {
  exigirOpciones(opciones, CAMPOS_OPCIONES_PETICION, "opciones de peticion");
  return exigirAbortSignal(opciones.signal);
}

async function leerJSONAcotado(respuesta) {
  const tipo = respuesta?.headers?.get?.("Content-Type") || "";
  if (!/^application\/json(?:\s*;\s*charset\s*=\s*utf-8)?$/iu.test(tipo)) {
    throw new Error("La respuesta cartografica interna no es JSON UTF-8.");
  }
  const declaradaTexto = respuesta.headers?.get?.("Content-Length");
  let declarada = null;
  if (declaradaTexto !== null && declaradaTexto !== undefined && declaradaTexto !== "") {
    if (!/^(?:0|[1-9][0-9]*)$/u.test(declaradaTexto)) {
      throw new Error("La respuesta cartografica declara una longitud no canonica.");
    }
    declarada = Number(declaradaTexto);
    if (!Number.isSafeInteger(declarada) || declarada < 1 || declarada > MAXIMO_RESPUESTA_BYTES) {
      throw new Error("La respuesta cartografica excede el limite autorizado.");
    }
  }
  if (!respuesta.body || typeof respuesta.body.getReader !== "function") {
    throw new Error("La respuesta cartografica no permite una lectura acotada.");
  }

  const lector = respuesta.body.getReader();
  const fragmentos = [];
  let total = 0;
  let cantidad = 0;
  try {
    while (true) {
      const lectura = await lector.read();
      if (!lectura || typeof lectura.done !== "boolean") {
        throw new Error("La respuesta cartografica contiene un flujo no valido.");
      }
      if (lectura.done) break;
      cantidad += 1;
      if (cantidad > MAXIMO_FRAGMENTOS_RESPUESTA
        || !(lectura.value instanceof Uint8Array)
        || total + lectura.value.byteLength > MAXIMO_RESPUESTA_BYTES) {
        throw new Error("La respuesta cartografica excede el limite autorizado.");
      }
      fragmentos.push(lectura.value);
      total += lectura.value.byteLength;
    }
  } catch (error) {
    try { await lector.cancel("respuesta rechazada"); } catch { /* conserva la causa */ }
    throw error;
  } finally {
    try { lector.releaseLock(); } catch { /* lector ya cancelado */ }
  }
  if (total === 0 || (declarada !== null && declarada !== total)) {
    throw new Error("La respuesta cartografica esta vacia o incompleta.");
  }
  const bytes = new Uint8Array(total);
  let desplazamiento = 0;
  fragmentos.forEach((fragmento) => {
    bytes.set(fragmento, desplazamiento);
    desplazamiento += fragmento.byteLength;
  });
  let contenido;
  try {
    contenido = new TextDecoder("utf-8", { fatal: true }).decode(bytes);
  } catch (error) {
    throw new Error("La respuesta cartografica no contiene UTF-8 valido.", { cause: error });
  }
  try {
    return JSON.parse(contenido);
  } catch (error) {
    throw new Error("La respuesta cartografica contiene JSON no valido.", { cause: error });
  }
}

async function solicitarOSRM(fetchImpl, cuerpo, signalExterno, tiempoEsperaMs) {
  const controlador = new AbortController();
  let motivoCorte = "";
  let rechazarCorte;
  const corte = new Promise((_, reject) => { rechazarCorte = reject; });
  const cancelarExterno = () => {
    motivoCorte = "cancelacion";
    controlador.abort(signalExterno?.reason);
    rechazarCorte(new Error("La consulta cartografica fue cancelada."));
  };
  if (signalExterno?.aborted) cancelarExterno();
  else signalExterno?.addEventListener("abort", cancelarExterno, { once: true });
  const temporizador = setTimeout(() => {
    motivoCorte = "tiempo";
    controlador.abort();
    rechazarCorte(new Error("El motor cartografico interno agoto su tiempo de respuesta."));
  }, tiempoEsperaMs);

  let respuesta;
  try {
    const operacion = (async () => {
      respuesta = await fetchImpl(RUTA_MEDIADOR, {
        method: "POST",
        headers: {
          Accept: "application/json",
          "Content-Type": "application/json",
        },
        body: JSON.stringify(cuerpo),
        credentials: "omit",
        mode: "same-origin",
        redirect: "error",
        cache: "no-store",
        referrer: "",
        referrerPolicy: "no-referrer",
        signal: controlador.signal,
      });
      if (!respuesta || respuesta.ok !== true || respuesta.redirected === true) {
        const estado = Number.isInteger(respuesta?.status) ? ` (HTTP ${respuesta.status})` : "";
        throw new Error(`No se pudo consultar el motor cartografico interno${estado}.`);
      }
      return leerJSONAcotado(respuesta);
    })();
    return await Promise.race([operacion, corte]);
  } catch (error) {
    if (motivoCorte === "tiempo") {
      throw new Error("El motor cartografico interno agoto su tiempo de respuesta.", { cause: error });
    }
    if (motivoCorte === "cancelacion" || signalExterno?.aborted) {
      throw new Error("La consulta cartografica fue cancelada.", { cause: error });
    }
    if (error instanceof Error) throw error;
    throw new Error("No se pudo consultar el motor cartografico interno.");
  } finally {
    clearTimeout(temporizador);
    signalExterno?.removeEventListener("abort", cancelarExterno);
  }
}

function redondear(valor, decimales = 2) {
  const factor = 10 ** decimales;
  return Math.round((valor + Number.EPSILON) * factor) / factor;
}

function construirCuerpoMediador(puntos, alternativas) {
  return Object.freeze({
    coordinates: Object.freeze(puntos.map((punto) => Object.freeze({
      lat: numeroAcotado(punto.latitud, -90, 90, "latitud provincial"),
      lon: numeroAcotado(punto.longitud, -180, 180, "longitud provincial"),
      name: textoCanonico(punto.nombre, "nombre provincial", 100),
    }))),
    alternatives: alternativas,
  });
}

function proyectarTrazado(geometria) {
  if (!esObjetoPlano(geometria) || geometria.type !== "LineString"
    || !Array.isArray(geometria.coordinates) || geometria.coordinates.length < 2
    || geometria.coordinates.length > MAXIMO_COORDENADAS_ENTRADA) {
    throw new TypeError("geometria GeoJSON de OSRM no valida");
  }
  const puntos = geometria.coordinates.map((coordenada) => {
    if (!Array.isArray(coordenada) || coordenada.length !== 2) {
      throw new TypeError("coordenada GeoJSON de OSRM no valida");
    }
    return Object.freeze([
      numeroAcotado(coordenada[1], -90, 90, "latitud GeoJSON"),
      numeroAcotado(coordenada[0], -180, 180, "longitud GeoJSON"),
    ]);
  });
  if (puntos.length <= MAXIMO_PUNTOS_TRAZADO) return Object.freeze(puntos);
  const reducidos = [];
  for (let indice = 0; indice < MAXIMO_PUNTOS_TRAZADO; indice += 1) {
    const posicion = Math.round(indice * (puntos.length - 1) / (MAXIMO_PUNTOS_TRAZADO - 1));
    reducidos.push(puntos[posicion]);
  }
  return Object.freeze(reducidos);
}

function proyectarTramos(ruta, solicitud, puntos) {
  if (!Array.isArray(ruta.legs) || ruta.legs.length !== puntos.length - 1) {
    throw new TypeError("tramos de OSRM no validos");
  }
  return ruta.legs.map((tramo, indice) => {
    if (!esObjetoPlano(tramo)) throw new TypeError("tramo de OSRM no valido");
    const origen = puntos[indice];
    const destino = puntos[indice + 1];
    return Object.freeze({
      indice,
      origen_codigo: solicitud.paradas[indice],
      origen_nombre: origen.nombre,
      destino_codigo: solicitud.paradas[indice + 1],
      destino_nombre: destino.nombre,
      kilometros: redondear(numeroAcotado(
        tramo.distance, 10, 10_000_000, "distancia de tramo OSRM",
      ) / 1_000),
      duracion_minutos: Math.max(1, Math.ceil(numeroAcotado(
        tramo.duration, 1, 1_200_000, "duracion de tramo OSRM",
      ) / 60)),
    });
  });
}

function proyectarRuta(ruta, solicitud, puntos, indice) {
  if (!esObjetoPlano(ruta)) throw new TypeError("ruta de OSRM no valida");
  const tramos = proyectarTramos(ruta, solicitud, puntos);
  const trazado = proyectarTrazado(ruta.geometry);
  return Object.freeze({
    recomendada: indice === 0,
    etiqueta: `ruta_alternativa_osrm_${indice + 1}`,
    kilometros: redondear(numeroAcotado(ruta.distance, 10, 10_000_000, "distancia OSRM") / 1_000),
    duracion_minutos: Math.max(1, Math.ceil(numeroAcotado(
      ruta.duration, 1, 1_200_000, "duracion OSRM",
    ) / 60)),
    tramos,
    geometria: {
      esquema: ESQUEMA_GEOMETRIA_RUTA_DIETAS,
      origen: "osrm_interno",
      liquidable: false,
      paradas: puntos.map((punto) => ({
        etiqueta: punto.nombre,
        latitud: punto.latitud,
        longitud: punto.longitud,
      })),
      trazado,
    },
  });
}

const ALFABETO_BASE32 = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567";

function base32(bytes) {
  let acumulador = 0;
  let bits = 0;
  let salida = "";
  for (const byte of bytes) {
    acumulador = (acumulador << 8) | byte;
    bits += 8;
    while (bits >= 5) {
      salida += ALFABETO_BASE32[(acumulador >>> (bits - 5)) & 31];
      bits -= 5;
    }
    acumulador &= (1 << bits) - 1;
  }
  if (bits > 0) salida += ALFABETO_BASE32[(acumulador << (5 - bits)) & 31];
  return salida;
}

async function crearReferencia(versionGrafo, solicitud, rutas, cripto) {
  if (!cripto?.subtle || typeof cripto.subtle.digest !== "function") {
    throw new Error("El entorno no dispone de SHA-256 para referenciar la ruta.");
  }
  // Estructura canonica: solo arrays y claves insertadas en orden fijo.
  const material = JSON.stringify({
    version_grafo: versionGrafo,
    paradas: solicitud.paradas,
    alternativas_solicitadas: solicitud.alternativas,
    rutas: rutas.map((ruta) => [
      ruta.kilometros,
      ruta.duracion_minutos,
      ruta.tramos.map((tramo) => [tramo.kilometros, tramo.duracion_minutos]),
      ruta.geometria.trazado,
    ]),
  });
  const digest = await cripto.subtle.digest("SHA-256", new TextEncoder().encode(material));
  return `OSRM-${base32(new Uint8Array(digest))}`;
}

async function proyectarCalculo(respuesta, solicitud, puntos, cripto) {
  if (!esObjetoPlano(respuesta) || respuesta.code !== "Ok"
    || respuesta.engine !== "osrm_on_premise" || !Array.isArray(respuesta.routes)
    || respuesta.routes.length < 1 || respuesta.routes.length > solicitud.alternativas
    || respuesta.routes.length > 3) {
    throw new TypeError("respuesta de OSRM no valida");
  }
  textoCanonico(respuesta.route_scope, "ambito del grafo OSRM", 160);
  const versionGrafo = obtenerVersionGrafoRespuesta(respuesta);
  const rutas = respuesta.routes.map((ruta, indice) => proyectarRuta(
    ruta, solicitud, puntos, indice,
  ));
  const referencia = await crearReferencia(versionGrafo, solicitud, rutas, cripto);
  return validarCalculoRutaDietas({
    esquema: ESQUEMA_CALCULO_RUTA_DIETAS,
    referencia,
    demostracion: true,
    liquidable: false,
    motor: "osrm_interno",
    version_grafo: versionGrafo,
    alternativas: rutas.map((ruta, indice) => ({
      referencia: `${referencia}-A${indice + 1}`,
      recomendada: ruta.recomendada,
      etiqueta: ruta.etiqueta,
      kilometros: ruta.kilometros,
      duracion_minutos: ruta.duracion_minutos,
      tramos: ruta.tramos,
      geometria: ruta.geometria,
    })),
  }, solicitud);
}

/** Compone el puerto DEMO con cartografia OSRM real y sin efectos. */
export function crearCalculadorRutasDietasPresentacionOSRM(opciones = {}) {
  exigirOpciones(opciones, CAMPOS_OPCIONES, "opciones del calculador OSRM de presentacion");
  const contexto = exigirContextoActorDietas(opciones.contextoActor);
  const capacidades = validarCapacidadesDietas(opciones.capacidades);
  if (contexto.demostracion !== true) {
    throw new Error("el calculador OSRM de presentacion exige un contexto DEMO");
  }
  if (!tieneCapacidadDietas(capacidades, CAPACIDAD_CONSULTAR_RUTA)) {
    throw new Error("falta capacidad para consultar rutas de Dietas");
  }
  if (typeof opciones.fetchImpl !== "function") {
    throw new TypeError("falta el cliente HTTP same-origin de presentacion");
  }
  const tiempoEsperaMs = opciones.tiempoEsperaMs ?? TIEMPO_ESPERA_MS;
  enteroAcotado(tiempoEsperaMs, 10, 30_000, "tiempo de espera HTTP");
  const catalogo = obtenerCatalogoRutasProvincial();
  const cripto = globalThis.crypto;

  return Object.freeze({
    obtenerCatalogo() {
      return catalogo;
    },
    async calcular(solicitudEntrada, opcionesEntrada = {}) {
      try {
        const signal = opcionesPeticion(opcionesEntrada);
        if (signal?.aborted) throw new Error("consulta cancelada");
        const solicitud = validarSolicitudRutaDietas(solicitudEntrada, catalogo);
        const puntos = resolverPuntosRutasProvincial(solicitud.paradas);
        const cuerpo = construirCuerpoMediador(puntos, solicitud.alternativas);
        const respuesta = await solicitarOSRM(
          opciones.fetchImpl, cuerpo, signal, tiempoEsperaMs,
        );
        return await proyectarCalculo(respuesta, solicitud, puntos, cripto);
      } catch {
        // La vista recibe únicamente un código cerrado y traducible. Nunca se
        // propaga texto del mediador, de Fetch ni de validadores internos.
        throw new ErrorServicioRutasDietas();
      }
    },
  });
}
