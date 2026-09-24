/** Contrato de cartografía interna para Dietas. */
export const ESQUEMA_GEOMETRIA_RUTA_DIETAS = "vec.dietas.geometria-ruta.v1";
export const ESQUEMA_CATALOGO_RUTAS_DIETAS = "vec.dietas.catalogo-rutas.v1";
export const ESQUEMA_SOLICITUD_RUTA_DIETAS = "vec.dietas.solicitud-ruta.v1";
export const ESQUEMA_CALCULO_RUTA_DIETAS = "vec.dietas.calculo-ruta.v1";
export const PLANTILLA_TESELAS_OSM_INTERNA = "/tiles/osm/{z}/{x}/{y}.png";
// Los enlaces de licencia sólo navegan por acción expresa: no generan ninguna
// petición automática al montar el mapa ni al cargar las teselas internas.
export const ATRIBUCION_OSM_INTERNA = "© <a href=\"https://www.openstreetmap.org/copyright\" target=\"_blank\" rel=\"noopener noreferrer\">OpenStreetMap</a> contributors · © <a href=\"https://openmaptiles.org/\" target=\"_blank\" rel=\"noopener noreferrer\">OpenMapTiles</a> · servido en red interna";
export const CODIGO_ERROR_SERVICIO_RUTAS_DIETAS = "servicio_rutas_no_disponible";

export class ErrorServicioRutasDietas extends Error {
  constructor() {
    super("No se pudo calcular la ruta con el servicio interno.");
    this.name = "ErrorServicioRutasDietas";
    this.codigo = CODIGO_ERROR_SERVICIO_RUTAS_DIETAS;
    Object.freeze(this);
  }
}


export function copiarDietas(valor) {
  return structuredClone(valor);
}

function texto(valor, nombre, maximo = 200) {
  const resultado = String(valor ?? "").trim();
  if (!resultado || resultado.length > maximo || /[\u0000-\u0008\u000B\u000C\u000E-\u001F]/.test(resultado)) {
    throw new Error(`${nombre} no valido`);
  }
  return resultado;
}

function clavesExactas(valor, esperadas, nombre) {
  const recibidas = Object.keys(valor || {}).sort();
  const previstas = [...esperadas].sort();
  if (recibidas.length !== previstas.length || recibidas.some((clave, indice) => clave !== previstas[indice])) {
    throw new Error(`${nombre} no valido`);
  }
}

function numeroAcotado(valor, minimo, maximo, nombre) {
  const numero = Number(valor);
  if (!Number.isFinite(numero) || numero < minimo || numero > maximo) throw new Error(`${nombre} no valido`);
  return numero;
}

function enteroAcotado(valor, minimo, maximo, nombre) {
  if (!Number.isInteger(valor) || valor < minimo || valor > maximo) throw new Error(`${nombre} no valido`);
  return valor;
}

function codigoRuta(valor, nombre = "codigo de ruta") {
  const resultado = texto(valor, nombre, 64);
  if (!/^[A-Za-z0-9][A-Za-z0-9._:-]{1,63}$/.test(resultado)) throw new Error(`${nombre} no valido`);
  return resultado;
}

function congelarProfundo(valor) {
  if (!valor || typeof valor !== "object" || Object.isFrozen(valor)) return valor;
  Object.values(valor).forEach(congelarProfundo);
  return Object.freeze(valor);
}

export function validarGeometriaRutaDietas(geometria, rutaEsperada = []) {
  if (!geometria || typeof geometria !== "object" || Array.isArray(geometria)) {
    throw new Error("geometria de ruta de Dietas no valida");
  }
  clavesExactas(geometria, geometria?.tramos === undefined
    ? ["esquema", "origen", "liquidable", "paradas", "trazado"]
    : ["esquema", "origen", "liquidable", "paradas", "trazado", "tramos"], "geometria de ruta de Dietas");
  if (geometria.esquema !== ESQUEMA_GEOMETRIA_RUTA_DIETAS
    || geometria.origen !== "osrm_interno"
    || geometria.liquidable !== false || !Array.isArray(geometria.paradas)
    || !Array.isArray(geometria.trazado) || geometria.paradas.length < 2
    || geometria.paradas.length > 12 || geometria.trazado.length < 2
    || geometria.trazado.length > 2_000) {
    throw new Error("geometria de ruta de Dietas no valida");
  }
  if (rutaEsperada.length && (rutaEsperada.length !== geometria.paradas.length
    || rutaEsperada.some((parada, indice) => parada !== geometria.paradas[indice]?.etiqueta))) {
    throw new Error("la geometria no corresponde a la ruta de Dietas");
  }
  geometria.paradas.forEach((parada) => {
    if (!parada || typeof parada !== "object" || Array.isArray(parada)) throw new Error("parada de ruta no valida");
    clavesExactas(parada, ["etiqueta", "latitud", "longitud"], "parada de ruta");
    texto(parada.etiqueta, "etiqueta de parada", 80);
    numeroAcotado(parada.latitud, -90, 90, "latitud de parada");
    numeroAcotado(parada.longitud, -180, 180, "longitud de parada");
  });
  geometria.trazado.forEach((punto) => {
    if (!Array.isArray(punto) || punto.length !== 2) throw new Error("punto de trazado no valido");
    numeroAcotado(punto[0], -90, 90, "latitud de trazado");
    numeroAcotado(punto[1], -180, 180, "longitud de trazado");
  });
  // Las rutas históricas ya conservadas no llevaban segmentos. Se admiten
  // solo como lectura compatible; los proyectores OSRM actuales siempre los
  // incluyen y el visor nuevo los usa para el trazado multicolor.
  if (geometria.tramos === undefined) {
    return congelarProfundo(copiarDietas(geometria));
  }
  if (!Array.isArray(geometria.tramos) || geometria.tramos.length !== geometria.paradas.length - 1) {
    throw new Error("tramos geométricos de Dietas no válidos");
  }
  geometria.tramos.forEach((tramo, indice) => {
    if (!tramo || typeof tramo !== "object" || Array.isArray(tramo)) throw new Error("tramo geométrico de Dietas no válido");
    clavesExactas(tramo, ["indice", "trazado"], "tramo geométrico de Dietas");
    if (tramo.indice !== indice || !Array.isArray(tramo.trazado) || tramo.trazado.length < 2 || tramo.trazado.length > 2_000) {
      throw new Error("tramo geométrico de Dietas no válido");
    }
    tramo.trazado.forEach((punto) => {
      if (!Array.isArray(punto) || punto.length !== 2 || !Number.isFinite(punto[0]) || !Number.isFinite(punto[1])
        || punto[0] < -90 || punto[0] > 90 || punto[1] < -180 || punto[1] > 180) {
        throw new Error("coordenada de tramo de Dietas no válida");
      }
    });
  });
  return congelarProfundo(copiarDietas(geometria));
}

export function validarCatalogoRutasDietas(catalogo) {
  if (!catalogo || typeof catalogo !== "object" || Array.isArray(catalogo)) {
    throw new Error("catalogo provincial de rutas no valido");
  }
  clavesExactas(catalogo, ["esquema", "demostracion", "completo", "version", "puntos"], "catalogo provincial de rutas");
  if (catalogo.esquema !== ESQUEMA_CATALOGO_RUTAS_DIETAS || catalogo.demostracion !== false
    || typeof catalogo.completo !== "boolean" || !Array.isArray(catalogo.puntos)
    || catalogo.puntos.length < 2 || catalogo.puntos.length > 500) {
    throw new Error("catalogo provincial de rutas no valido");
  }
  texto(catalogo.version, "version del catalogo", 80);
  const codigos = new Set();
  catalogo.puntos.forEach((punto) => {
    if (!punto || typeof punto !== "object" || Array.isArray(punto)) throw new Error("punto del catalogo no valido");
    clavesExactas(punto, ["codigo", "nombre", "tipo", "municipio_codigo", "municipio_nombre"], "punto del catalogo");
    const codigo = codigoRuta(punto.codigo, "codigo del punto");
    if (codigos.has(codigo)) throw new Error("codigo del punto repetido");
    codigos.add(codigo);
    texto(punto.nombre, "nombre del punto", 100);
    texto(punto.tipo, "tipo del punto", 30);
    codigoRuta(punto.municipio_codigo, "codigo del municipio");
    texto(punto.municipio_nombre, "nombre del municipio", 100);
  });
  return congelarProfundo(copiarDietas(catalogo));
}

export function validarSolicitudRutaDietas(solicitud, catalogo = null) {
  if (!solicitud || typeof solicitud !== "object" || Array.isArray(solicitud)) {
    throw new Error("solicitud de calculo de ruta no valida");
  }
  clavesExactas(solicitud, ["esquema", "paradas", "alternativas"], "solicitud de calculo de ruta");
  if (solicitud.esquema !== ESQUEMA_SOLICITUD_RUTA_DIETAS || !Array.isArray(solicitud.paradas)
    || solicitud.paradas.length < 2 || solicitud.paradas.length > 12) {
    throw new Error("solicitud de calculo de ruta no valida");
  }
  enteroAcotado(solicitud.alternativas, 1, 3, "numero de alternativas");
  const permitidos = catalogo ? new Set(validarCatalogoRutasDietas(catalogo).puntos.map((punto) => punto.codigo)) : null;
  const paradas = solicitud.paradas.map((parada) => codigoRuta(parada, "codigo de parada"));
  paradas.forEach((parada, indice) => {
    if (permitidos && !permitidos.has(parada)) throw new Error("parada fuera del catalogo provincial");
    if (indice > 0 && parada === paradas[indice - 1]) throw new Error("la ruta contiene paradas consecutivas iguales");
  });
  return congelarProfundo({ esquema: solicitud.esquema, paradas, alternativas: solicitud.alternativas });
}

export function validarCalculoRutaDietas(calculo, solicitudEsperada = null) {
  if (!calculo || typeof calculo !== "object" || Array.isArray(calculo)) throw new Error("calculo de ruta no valido");
  clavesExactas(calculo, [
    "esquema", "referencia", "demostracion", "liquidable", "motor", "version_grafo", "alternativas",
  ], "calculo de ruta");
  if (calculo.esquema !== ESQUEMA_CALCULO_RUTA_DIETAS || calculo.demostracion !== false
    || calculo.liquidable !== false || calculo.motor !== "osrm_interno"
    || !Array.isArray(calculo.alternativas) || calculo.alternativas.length < 1 || calculo.alternativas.length > 3) {
    throw new Error("calculo de ruta no valido");
  }
  codigoRuta(calculo.referencia, "referencia del calculo");
  texto(calculo.version_grafo, "version del grafo", 100);
  const solicitud = solicitudEsperada ? validarSolicitudRutaDietas(solicitudEsperada) : null;
  const referencias = new Set();
  let recomendadas = 0;
  calculo.alternativas.forEach((alternativa) => {
    if (!alternativa || typeof alternativa !== "object" || Array.isArray(alternativa)) throw new Error("alternativa de ruta no valida");
    clavesExactas(alternativa, [
      "referencia", "recomendada", "etiqueta", "kilometros", "duracion_minutos", "tramos", "geometria",
    ], "alternativa de ruta");
    const referencia = codigoRuta(alternativa.referencia, "referencia de alternativa");
    if (referencias.has(referencia)) throw new Error("alternativa de ruta repetida");
    referencias.add(referencia);
    if (typeof alternativa.recomendada !== "boolean") throw new Error("alternativa de ruta no valida");
    if (alternativa.recomendada) recomendadas += 1;
    texto(alternativa.etiqueta, "etiqueta de alternativa", 80);
    numeroAcotado(alternativa.kilometros, 0.01, 10_000, "kilometros de alternativa");
    enteroAcotado(alternativa.duracion_minutos, 1, 20_000, "duracion de alternativa");
    if (!Array.isArray(alternativa.tramos) || alternativa.tramos.length < 1 || alternativa.tramos.length > 11
      || (solicitud && alternativa.tramos.length !== solicitud.paradas.length - 1)) {
      throw new Error("tramos de alternativa no validos");
    }
    alternativa.tramos.forEach((tramo, indice) => {
      if (!tramo || typeof tramo !== "object" || Array.isArray(tramo)) throw new Error("tramo de ruta no valido");
      clavesExactas(tramo, tramo.trazado === undefined
        ? ["indice", "origen_codigo", "origen_nombre", "destino_codigo", "destino_nombre", "kilometros", "duracion_minutos"]
        : ["indice", "origen_codigo", "origen_nombre", "destino_codigo", "destino_nombre", "kilometros", "duracion_minutos", "trazado"], "tramo de ruta");
      enteroAcotado(tramo.indice, 0, 10, "indice de tramo");
      if (tramo.indice !== indice) throw new Error("indices de tramo no consecutivos");
      codigoRuta(tramo.origen_codigo, "origen del tramo");
      codigoRuta(tramo.destino_codigo, "destino del tramo");
      texto(tramo.origen_nombre, "nombre de origen", 100);
      texto(tramo.destino_nombre, "nombre de destino", 100);
      numeroAcotado(tramo.kilometros, 0.01, 10_000, "kilometros del tramo");
      enteroAcotado(tramo.duracion_minutos, 1, 20_000, "duracion del tramo");
      if (tramo.trazado !== undefined && (!Array.isArray(tramo.trazado) || tramo.trazado.length < 2 || tramo.trazado.length > 2_000)) {
        throw new Error("trazado de tramo no válido");
      }
      if (solicitud && (tramo.origen_codigo !== solicitud.paradas[indice]
        || tramo.destino_codigo !== solicitud.paradas[indice + 1])) {
        throw new Error("el tramo no corresponde a la solicitud");
      }
    });
    const ruta = [alternativa.tramos[0].origen_nombre, ...alternativa.tramos.map((tramo) => tramo.destino_nombre)];
    const geometria = validarGeometriaRutaDietas(alternativa.geometria, ruta);
    if (geometria.origen !== "osrm_interno") {
      throw new Error("la geometria de ruta no corresponde al entorno del calculo");
    }
  });
  if (recomendadas !== 1) throw new Error("el calculo debe contener una unica ruta recomendada");
  return congelarProfundo(copiarDietas(calculo));
}
