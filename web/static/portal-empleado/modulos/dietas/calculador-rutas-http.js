/**
 * Adaptador HTTP productivo del puerto de rutas de Dietas.
 *
 * Las coordenadas del catalogo gobernado permanecen privadas en este cierre y
 * solo se envian al endpoint same-origin que media con el OSRM on-premise. El
 * ContextoActor sirve para fijar el ambito de composicion, nunca como fuente de
 * autorizacion: el servidor debe autenticar y autorizar de nuevo cada peticion.
 * La política HTTP usa `credentials: omit`: esta superficie no envía cookies,
 * certificados cliente ni credenciales HTTP del navegador. Por ello no admite
 * `globalThis.fetch` de forma implícita: exige un cliente inyectado por el
 * conector de identidad nativo o una mediación corporativa autenticada sin
 * cookies. Hasta disponer de ese conector, la composición productiva falla
 * cerrada; no se presupone que Kerberos/SPNEGO o mTLS atraviesen Fetch.
 *
 * Dependencia de integracion: GET /api/vec/workspace debe proyectar
 * province_route_points y province_route_matrix para el actor autorizado. A
 * fecha de este adaptador la ruta productiva falla cerrada hasta disponer de
 * esa proyeccion PDP; no se emplean datos alternativos ni semillas locales.
 */

import {
  CAPACIDAD_CONSULTAR_RUTA,
  ESQUEMA_CALCULO_RUTA_DIETAS,
  ESQUEMA_CATALOGO_RUTAS_DIETAS,
  ESQUEMA_GEOMETRIA_RUTA_DIETAS,
  exigirContextoActorDietas,
  tieneCapacidadDietas,
  validarCalculoRutaDietas,
  validarCapacidadesDietas,
  validarCatalogoRutasDietas,
  validarSolicitudRutaDietas,
} from "./contrato.js";

const RUTA_CATALOGO = "/api/vec/workspace";
const RUTA_CALCULO = "/api/vec/dietas/road-route";
const TIEMPO_ESPERA_MS = 12_000;
const MAXIMO_RESPUESTA_CATALOGO = 2 * 1024 * 1024;
const MAXIMO_RESPUESTA_RUTA = 20 * 1024 * 1024;
const MAXIMO_FRAGMENTOS_RESPUESTA = 8_192;
const MAXIMO_COORDENADAS_OSRM = 50_000;
const MAXIMO_PUNTOS_TRAZADO = 2_000;

const CAMPOS_OPCIONES = new Set([
  "contextoActor", "capacidades", "fetchImpl", "tiempoEsperaMs",
]);
const CAMPOS_OPCIONES_PETICION = new Set(["signal"]);
const CAMPOS_PUNTO_BACKEND = new Set([
  "code", "name", "kind", "municipality_code", "municipality_name",
  "lat", "lon", "source", "state",
]);

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

function textoCanonico(valor, nombre, maximo = 200) {
  if (typeof valor !== "string" || valor.length === 0 || valor.length > maximo
    || valor !== valor.trim() || /[\u0000-\u001F\u007F-\u009F]/u.test(valor)) {
    throw new TypeError(`${nombre} no valido`);
  }
  return valor;
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

function mensajeHTTP(operacion, respuesta) {
  const estado = Number.isInteger(respuesta?.status) ? ` (HTTP ${respuesta.status})` : "";
  return `No se pudo ${operacion}${estado}.`;
}

async function leerJSONAcotado(respuesta, maximoBytes) {
  const tipo = respuesta?.headers?.get?.("Content-Type") || "";
  if (!/^application\/json(?:\s*;\s*charset\s*=\s*utf-8)?$/iu.test(tipo)) {
    throw new Error("La respuesta interna no es JSON UTF-8.");
  }

  const longitudTexto = respuesta.headers?.get?.("Content-Length");
  let longitudDeclarada = null;
  if (longitudTexto !== null && longitudTexto !== undefined && longitudTexto !== "") {
    if (!/^(?:0|[1-9][0-9]*)$/u.test(longitudTexto)) {
      throw new Error("La respuesta interna declara una longitud no canonica.");
    }
    longitudDeclarada = Number(longitudTexto);
    if (!Number.isSafeInteger(longitudDeclarada) || longitudDeclarada === 0
      || longitudDeclarada > maximoBytes) {
      throw new Error("La respuesta interna excede el limite autorizado.");
    }
  }

  if (!respuesta.body || typeof respuesta.body.getReader !== "function") {
    throw new Error("La respuesta interna no permite una lectura acotada.");
  }
  const lector = respuesta.body.getReader();
  const fragmentos = [];
  let total = 0;
  let cantidadFragmentos = 0;
  try {
    while (true) {
      const lectura = await lector.read();
      if (!lectura || typeof lectura.done !== "boolean") {
        throw new Error("La respuesta interna contiene un flujo no valido.");
      }
      if (lectura.done) break;
      cantidadFragmentos += 1;
      if (cantidadFragmentos > MAXIMO_FRAGMENTOS_RESPUESTA
        || !(lectura.value instanceof Uint8Array)
        || total + lectura.value.byteLength > maximoBytes) {
        throw new Error("La respuesta interna excede el limite autorizado.");
      }
      fragmentos.push(lectura.value);
      total += lectura.value.byteLength;
    }
  } catch (error) {
    try { await lector.cancel("respuesta rechazada"); } catch { /* no oculta la causa */ }
    throw error;
  } finally {
    try { lector.releaseLock(); } catch { /* lector cancelado */ }
  }
  if (total === 0) throw new Error("La respuesta interna esta vacia.");
  if (longitudDeclarada !== null && total !== longitudDeclarada) {
    throw new Error("La respuesta interna no coincide con su longitud declarada.");
  }

  const bytes = new Uint8Array(total);
  let desplazamiento = 0;
  fragmentos.forEach((fragmento) => {
    bytes.set(fragmento, desplazamiento);
    desplazamiento += fragmento.byteLength;
  });
  let texto;
  try {
    texto = new TextDecoder("utf-8", { fatal: true }).decode(bytes);
  } catch (error) {
    throw new Error("La respuesta interna no contiene UTF-8 valido.", { cause: error });
  }
  try {
    return JSON.parse(texto);
  } catch (error) {
    throw new Error("La respuesta interna contiene JSON no valido.", { cause: error });
  }
}

async function solicitarJSON(fetchImpl, ruta, configuracion, signalExterno, tiempoEsperaMs, maximoBytes) {
  const controlador = new AbortController();
  let agotado = false;
  let rechazarCorte;
  const corte = new Promise((_, reject) => {
    rechazarCorte = reject;
  });
  const cancelarDesdeExterior = () => {
    controlador.abort(signalExterno?.reason);
    rechazarCorte(new Error("La consulta de Dietas fue cancelada."));
  };
  if (signalExterno?.aborted) cancelarDesdeExterior();
  else signalExterno?.addEventListener("abort", cancelarDesdeExterior, { once: true });
  const temporizador = setTimeout(() => {
    agotado = true;
    controlador.abort();
    rechazarCorte(new Error("El servicio interno de Dietas agoto su tiempo de respuesta."));
  }, tiempoEsperaMs);

  let respuesta;
  try {
    const operacion = (async () => {
      respuesta = await fetchImpl(ruta, {
        ...configuracion,
        credentials: "omit",
        mode: "same-origin",
        redirect: "error",
        cache: "no-store",
        referrer: "",
        referrerPolicy: "no-referrer",
        signal: controlador.signal,
      });
      if (!respuesta || respuesta.ok !== true) {
        throw new Error(mensajeHTTP("consultar el servicio interno de Dietas", respuesta));
      }
      return leerJSONAcotado(respuesta, maximoBytes);
    })();
    return await Promise.race([operacion, corte]);
  } catch (error) {
    if (agotado) throw new Error("El servicio interno de Dietas agoto su tiempo de respuesta.", { cause: error });
    if (signalExterno?.aborted) throw new Error("La consulta de Dietas fue cancelada.", { cause: error });
    if (error instanceof Error) throw error;
    throw new Error("No se pudo consultar el servicio interno de Dietas.");
  } finally {
    clearTimeout(temporizador);
    signalExterno?.removeEventListener("abort", cancelarDesdeExterior);
    if (respuesta && (respuesta.ok !== true || agotado || signalExterno?.aborted)
      && respuesta.body?.cancel) {
      try { await respuesta.body.cancel("respuesta HTTP rechazada"); } catch { /* no oculta la causa */ }
    }
  }
}

function validarPuntoBackend(entrada) {
  if (!esObjetoPlano(entrada) || Object.keys(entrada).length !== CAMPOS_PUNTO_BACKEND.size
    || Object.keys(entrada).some((campo) => !CAMPOS_PUNTO_BACKEND.has(campo))) {
    throw new TypeError("punto del catalogo gobernado no valido");
  }
  const codigo = textoCanonico(entrada.code, "codigo de punto", 64);
  const nombre = textoCanonico(entrada.name, "nombre de punto", 100);
  const tipo = textoCanonico(entrada.kind, "tipo de punto", 30);
  const municipioCodigo = textoCanonico(entrada.municipality_code, "codigo de municipio", 64);
  const municipioNombre = textoCanonico(entrada.municipality_name, "nombre de municipio", 100);
  textoCanonico(entrada.source, "fuente de punto", 500);
  textoCanonico(entrada.state, "estado de punto", 100);
  const latitud = numeroAcotado(entrada.lat, -90, 90, "latitud de punto");
  const longitud = numeroAcotado(entrada.lon, -180, 180, "longitud de punto");
  return Object.freeze({
    publico: Object.freeze({
      codigo,
      nombre,
      tipo,
      municipio_codigo: municipioCodigo,
      municipio_nombre: municipioNombre,
    }),
    coordenada: Object.freeze({ latitud, longitud }),
  });
}

function proyectarCatalogo(respuesta) {
  if (!esObjetoPlano(respuesta) || !Array.isArray(respuesta.province_route_points)
    || !esObjetoPlano(respuesta.province_route_matrix)) {
    throw new TypeError("la proyeccion productiva del catalogo de Dietas no esta disponible");
  }
  const matriz = respuesta.province_route_matrix;
  const version = textoCanonico(matriz.matrix_version, "version del catalogo", 80);
  enteroAcotado(matriz.route_points_loaded, 2, 500, "cantidad de puntos cargados");
  if (typeof matriz.import_required_before_liquidation !== "boolean") {
    throw new TypeError("estado de homologacion del catalogo no valido");
  }
  if (respuesta.province_route_points.length !== matriz.route_points_loaded) {
    throw new TypeError("el catalogo y su manifiesto de cobertura no coinciden");
  }

  const puntosInternos = respuesta.province_route_points.map(validarPuntoBackend);
  const catalogo = validarCatalogoRutasDietas({
    esquema: ESQUEMA_CATALOGO_RUTAS_DIETAS,
    demostracion: false,
    completo: matriz.import_required_before_liquidation === false,
    version,
    puntos: puntosInternos.map((punto) => punto.publico),
  });
  const coordenadasPorCodigo = new Map();
  puntosInternos.forEach((punto) => {
    if (coordenadasPorCodigo.has(punto.publico.codigo)) {
      throw new TypeError("el catalogo gobernado contiene codigos repetidos");
    }
    coordenadasPorCodigo.set(punto.publico.codigo, Object.freeze({
      codigo: punto.publico.codigo,
      nombre: punto.publico.nombre,
      latitud: punto.coordenada.latitud,
      longitud: punto.coordenada.longitud,
    }));
  });
  return Object.freeze({ catalogo, coordenadasPorCodigo });
}

function redondear(valor, decimales = 2) {
  const factor = 10 ** decimales;
  return Math.round((valor + Number.EPSILON) * factor) / factor;
}

function coordenadasTrazado(geometria) {
  if (!esObjetoPlano(geometria) || geometria.type !== "LineString"
    || !Array.isArray(geometria.coordinates) || geometria.coordinates.length < 2
    || geometria.coordinates.length > MAXIMO_COORDENADAS_OSRM) {
    throw new TypeError("geometria OSRM no valida");
  }
  const puntos = geometria.coordinates.map((coordenada) => {
    if (!Array.isArray(coordenada) || coordenada.length !== 2) {
      throw new TypeError("coordenada OSRM no valida");
    }
    const longitud = numeroAcotado(coordenada[0], -180, 180, "longitud OSRM");
    const latitud = numeroAcotado(coordenada[1], -90, 90, "latitud OSRM");
    return Object.freeze([latitud, longitud]);
  });
  if (puntos.length <= MAXIMO_PUNTOS_TRAZADO) return puntos;

  // La geometria es solo una proyeccion cartografica no liquidable. Conserva
  // extremos y muestrea de forma uniforme sin exponer el payload OSRM crudo.
  const reducidos = [];
  for (let indice = 0; indice < MAXIMO_PUNTOS_TRAZADO; indice += 1) {
    const posicion = Math.round(indice * (puntos.length - 1) / (MAXIMO_PUNTOS_TRAZADO - 1));
    reducidos.push(puntos[posicion]);
  }
  return reducidos;
}

function proyectarTramos(rutaOSRM, solicitud, puntosPorCodigo) {
  let tramosOSRM = rutaOSRM.legs;
  if (tramosOSRM === undefined && solicitud.paradas.length === 2) {
    tramosOSRM = [{ distance: rutaOSRM.distance, duration: rutaOSRM.duration }];
  }
  if (!Array.isArray(tramosOSRM) || tramosOSRM.length !== solicitud.paradas.length - 1) {
    throw new TypeError("tramos OSRM no validos");
  }
  return tramosOSRM.map((tramo, indice) => {
    if (!esObjetoPlano(tramo)) throw new TypeError("tramo OSRM no valido");
    const distanciaMetros = numeroAcotado(tramo.distance, 10, 10_000_000, "distancia de tramo OSRM");
    const duracionSegundos = numeroAcotado(tramo.duration, 1, 1_200_000, "duracion de tramo OSRM");
    const origen = puntosPorCodigo.get(solicitud.paradas[indice]);
    const destino = puntosPorCodigo.get(solicitud.paradas[indice + 1]);
    if (!origen || !destino) throw new TypeError("parada sin coordenada gobernada");
    return Object.freeze({
      indice,
      origen_codigo: origen.codigo,
      origen_nombre: origen.nombre,
      destino_codigo: destino.codigo,
      destino_nombre: destino.nombre,
      kilometros: redondear(distanciaMetros / 1_000),
      duracion_minutos: Math.max(1, Math.ceil(duracionSegundos / 60)),
    });
  });
}

function proyectarRutaOSRM(rutaOSRM, solicitud, puntosPorCodigo, indice) {
  if (!esObjetoPlano(rutaOSRM)) throw new TypeError("ruta OSRM no valida");
  const distanciaMetros = numeroAcotado(rutaOSRM.distance, 10, 10_000_000, "distancia OSRM");
  const duracionSegundos = numeroAcotado(rutaOSRM.duration, 1, 1_200_000, "duracion OSRM");
  const tramos = proyectarTramos(rutaOSRM, solicitud, puntosPorCodigo);
  const paradas = solicitud.paradas.map((codigo) => {
    const punto = puntosPorCodigo.get(codigo);
    if (!punto) throw new TypeError("parada sin coordenada gobernada");
    return {
      etiqueta: punto.nombre,
      latitud: punto.latitud,
      longitud: punto.longitud,
    };
  });
  return Object.freeze({
    recomendada: indice === 0,
    // Etiqueta tecnica estable; la vista traduce el estado `recomendada`.
    etiqueta: `OSRM · A${indice + 1}`,
    kilometros: redondear(distanciaMetros / 1_000),
    duracion_minutos: Math.max(1, Math.ceil(duracionSegundos / 60)),
    tramos,
    geometria: {
      esquema: ESQUEMA_GEOMETRIA_RUTA_DIETAS,
      origen: "osrm_interno",
      liquidable: false,
      paradas,
      trazado: coordenadasTrazado(rutaOSRM.geometry),
    },
  });
}

async function referenciaCalculo(version, solicitud, rutas, cripto) {
  if (!cripto?.subtle || typeof cripto.subtle.digest !== "function") {
    throw new Error("El entorno no dispone de SHA-256 para referenciar el calculo.");
  }
  const material = JSON.stringify({
    version,
    paradas: solicitud.paradas,
    alternativas: rutas.map((ruta) => ({
      kilometros: ruta.kilometros,
      duracion_minutos: ruta.duracion_minutos,
      tramos: ruta.tramos.map((tramo) => [tramo.kilometros, tramo.duracion_minutos]),
      trazado: ruta.geometria.trazado,
    })),
  });
  const huella = new Uint8Array(await cripto.subtle.digest("SHA-256", new TextEncoder().encode(material)));
  const hexadecimal = [...huella].slice(0, 16)
    .map((byte) => byte.toString(16).padStart(2, "0")).join("").toUpperCase();
  return `RUTA-OSRM-${hexadecimal}`;
}

async function proyectarCalculo(respuesta, solicitud, puntosPorCodigo, cripto) {
  if (!esObjetoPlano(respuesta) || respuesta.code !== "Ok"
    || respuesta.engine !== "osrm_on_premise") {
    throw new TypeError("respuesta del mediador OSRM no valida");
  }
  textoCanonico(respuesta.route_scope, "ambito de ruta", 160);
  const version = textoCanonico(respuesta.data_version, "version del grafo", 100);
  if (!Array.isArray(respuesta.routes) || respuesta.routes.length < 1
    || respuesta.routes.length > solicitud.alternativas || respuesta.routes.length > 3) {
    throw new TypeError("alternativas OSRM no validas");
  }
  const rutas = respuesta.routes.map((ruta, indice) => proyectarRutaOSRM(
    ruta, solicitud, puntosPorCodigo, indice,
  ));
  const referencia = await referenciaCalculo(version, solicitud, rutas, cripto);
  return validarCalculoRutaDietas({
    esquema: ESQUEMA_CALCULO_RUTA_DIETAS,
    referencia,
    demostracion: false,
    liquidable: false,
    motor: "osrm_interno",
    version_grafo: version,
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

/** Compone el adaptador productivo; no acepta ContextoActor de demostracion. */
export function crearCalculadorRutasDietasHTTP(opciones = {}) {
  exigirOpciones(opciones, CAMPOS_OPCIONES, "opciones del calculador HTTP");
  const contexto = exigirContextoActorDietas(opciones.contextoActor);
  const capacidades = validarCapacidadesDietas(opciones.capacidades);
  if (contexto.demostracion !== false) {
    throw new Error("el calculador HTTP exige un ContextoActor productivo");
  }
  if (!tieneCapacidadDietas(capacidades, CAPACIDAD_CONSULTAR_RUTA)) {
    throw new Error("falta capacidad para consultar rutas de Dietas");
  }
  const fetchImpl = opciones.fetchImpl;
  if (typeof fetchImpl !== "function") {
    throw new TypeError("falta el cliente HTTP del conector de identidad sin cookies");
  }
  const tiempoEsperaMs = opciones.tiempoEsperaMs ?? TIEMPO_ESPERA_MS;
  enteroAcotado(tiempoEsperaMs, 10, 30_000, "tiempo de espera HTTP");
  const cripto = globalThis.crypto;
  let proyeccionCatalogo = null;

  async function obtenerCatalogo(opcionesEntrada = {}) {
    const signal = opcionesPeticion(opcionesEntrada);
    if (signal?.aborted) throw new Error("La consulta de Dietas fue cancelada.");
    const respuesta = await solicitarJSON(fetchImpl, RUTA_CATALOGO, {
      method: "GET",
      headers: { Accept: "application/json" },
    }, signal, tiempoEsperaMs, MAXIMO_RESPUESTA_CATALOGO);
    const proyectada = proyectarCatalogo(respuesta);
    proyeccionCatalogo = proyectada;
    return proyectada.catalogo;
  }

  async function calcular(solicitudEntrada, opcionesEntrada = {}) {
    const signal = opcionesPeticion(opcionesEntrada);
    if (signal?.aborted) throw new Error("La consulta de Dietas fue cancelada.");
    if (proyeccionCatalogo === null) await obtenerCatalogo({ signal });
    // Una actualizacion concurrente del catalogo no puede mezclar las
    // coordenadas enviadas con nombres o puntos de otra version.
    const proyeccionCalculo = proyeccionCatalogo;
    const solicitud = validarSolicitudRutaDietas(
      solicitudEntrada, proyeccionCalculo.catalogo,
    );
    const coordenadas = solicitud.paradas.map((codigo) => {
      const punto = proyeccionCalculo.coordenadasPorCodigo.get(codigo);
      if (!punto) throw new Error("parada sin coordenada gobernada");
      return { lat: punto.latitud, lon: punto.longitud };
    });
    const respuesta = await solicitarJSON(fetchImpl, RUTA_CALCULO, {
      method: "POST",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ coordinates: coordenadas, alternatives: solicitud.alternativas }),
    }, signal, tiempoEsperaMs, MAXIMO_RESPUESTA_RUTA);
    return proyectarCalculo(
      respuesta, solicitud, proyeccionCalculo.coordenadasPorCodigo, cripto,
    );
  }

  return Object.freeze({ obtenerCatalogo, calcular });
}
