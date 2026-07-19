/** Contrato estable entre la superficie Dietas y cualquier adaptador. */

import { exigirContextoParaModulo } from "../../identidad/contexto-actor.js";

export const ESQUEMA_PANEL_DIETAS = "vec.dietas.portal.v1";
export const ESQUEMA_RECIBO_DIETAS = "vec.documentos.recibo.dietas.v1";
export const ESQUEMA_RESUMEN_ANUAL_DIETAS = "vec.documentos.resumen-anual.dietas.v1";
export const ESQUEMA_GEOMETRIA_RUTA_DIETAS = "vec.dietas.geometria-ruta.v1";
export const ESQUEMA_CATALOGO_RUTAS_DIETAS = "vec.dietas.catalogo-rutas.v1";
export const ESQUEMA_SOLICITUD_RUTA_DIETAS = "vec.dietas.solicitud-ruta.v1";
export const ESQUEMA_CALCULO_RUTA_DIETAS = "vec.dietas.calculo-ruta.v1";
export const PLANTILLA_TESELAS_OSM_INTERNA = "/tiles/osm/{z}/{x}/{y}.png";
// Los enlaces de licencia sólo navegan por acción expresa: no generan ninguna
// petición automática al montar el mapa ni al cargar las teselas internas.
export const ATRIBUCION_OSM_INTERNA = "© <a href=\"https://www.openstreetmap.org/copyright\" target=\"_blank\" rel=\"noopener noreferrer\">OpenStreetMap</a> contributors · © <a href=\"https://openmaptiles.org/\" target=\"_blank\" rel=\"noopener noreferrer\">OpenMapTiles</a> · servido en red interna";
export const CAPACIDAD_CONSULTAR_GASTO = "dietas.gasto.read";
export const CAPACIDAD_GESTIONAR_GASTO = "dietas.gasto.manage";
export const CAPACIDAD_CONSULTAR_RUTA = "dietas.ruta.read";
export const CAPACIDAD_GESTIONAR_RUTA = "dietas.ruta.manage";
export const CAPACIDAD_GESTIONAR_APROBACION = "dietas.aprobacion.manage";
export const CAPACIDAD_CONSULTAR_AUDITORIA = "dietas.audit.read";

const CAPACIDADES_DIETAS = new Set([
  CAPACIDAD_CONSULTAR_GASTO,
  CAPACIDAD_GESTIONAR_GASTO,
  CAPACIDAD_CONSULTAR_RUTA,
  CAPACIDAD_GESTIONAR_RUTA,
  CAPACIDAD_GESTIONAR_APROBACION,
  CAPACIDAD_CONSULTAR_AUDITORIA,
]);
const ESTADOS_DIETAS = new Set([
  "borrador", "pendiente_jefatura", "aprobada", "enviada_rrhh", "enviada_nomina", "pagada",
]);
const ETAPAS_DIETAS = new Set(["borrador", "jefatura", "aprobada", "rrhh", "nomina", "pagada"]);
const SIGUIENTES_ACTUACIONES = new Set([
  "remision_rrhh", "completar_enviar_validacion", "expediente_finalizado", "inclusion_nomina", "revision_jefatura",
]);

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
  clavesExactas(geometria, ["esquema", "origen", "liquidable", "paradas", "trazado"], "geometria de ruta de Dietas");
  if (geometria.esquema !== ESQUEMA_GEOMETRIA_RUTA_DIETAS
    || !["sintetica_demo", "osrm_interno"].includes(geometria.origen)
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
  return congelarProfundo(copiarDietas(geometria));
}

export function validarCatalogoRutasDietas(catalogo) {
  if (!catalogo || typeof catalogo !== "object" || Array.isArray(catalogo)) {
    throw new Error("catalogo provincial de rutas no valido");
  }
  clavesExactas(catalogo, ["esquema", "demostracion", "completo", "version", "puntos"], "catalogo provincial de rutas");
  if (catalogo.esquema !== ESQUEMA_CATALOGO_RUTAS_DIETAS || typeof catalogo.demostracion !== "boolean"
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
  if (calculo.esquema !== ESQUEMA_CALCULO_RUTA_DIETAS || typeof calculo.demostracion !== "boolean"
    || calculo.liquidable !== false || !["simulacion_osrm_demo", "osrm_interno"].includes(calculo.motor)
    || !Array.isArray(calculo.alternativas) || calculo.alternativas.length < 1 || calculo.alternativas.length > 3) {
    throw new Error("calculo de ruta no valido");
  }
  const motorEsperado = calculo.demostracion ? "simulacion_osrm_demo" : "osrm_interno";
  const origenGeometriaEsperado = calculo.demostracion ? "sintetica_demo" : "osrm_interno";
  if (calculo.motor !== motorEsperado) {
    throw new Error("el motor de ruta no corresponde al entorno del calculo");
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
      clavesExactas(tramo, [
        "indice", "origen_codigo", "origen_nombre", "destino_codigo", "destino_nombre", "kilometros", "duracion_minutos",
      ], "tramo de ruta");
      enteroAcotado(tramo.indice, 0, 10, "indice de tramo");
      if (tramo.indice !== indice) throw new Error("indices de tramo no consecutivos");
      codigoRuta(tramo.origen_codigo, "origen del tramo");
      codigoRuta(tramo.destino_codigo, "destino del tramo");
      texto(tramo.origen_nombre, "nombre de origen", 100);
      texto(tramo.destino_nombre, "nombre de destino", 100);
      numeroAcotado(tramo.kilometros, 0.01, 10_000, "kilometros del tramo");
      enteroAcotado(tramo.duracion_minutos, 1, 20_000, "duracion del tramo");
      if (solicitud && (tramo.origen_codigo !== solicitud.paradas[indice]
        || tramo.destino_codigo !== solicitud.paradas[indice + 1])) {
        throw new Error("el tramo no corresponde a la solicitud");
      }
    });
    const ruta = [alternativa.tramos[0].origen_nombre, ...alternativa.tramos.map((tramo) => tramo.destino_nombre)];
    const geometria = validarGeometriaRutaDietas(alternativa.geometria, ruta);
    if (geometria.origen !== origenGeometriaEsperado) {
      throw new Error("la geometria de ruta no corresponde al entorno del calculo");
    }
  });
  if (recomendadas !== 1) throw new Error("el calculo debe contener una unica ruta recomendada");
  return congelarProfundo(copiarDietas(calculo));
}

export function exigirContextoActorDietas(contextoActor) {
  return exigirContextoParaModulo(contextoActor, "dietas");
}

export function validarCapacidadesDietas(capacidades = []) {
  if (!Array.isArray(capacidades) || capacidades.length > CAPACIDADES_DIETAS.size) {
    throw new Error("capacidades de Dietas no validas");
  }
  const unicas = new Set();
  for (const capacidad of capacidades) {
    if (!CAPACIDADES_DIETAS.has(capacidad) || unicas.has(capacidad)) {
      throw new Error("capacidad de Dietas no valida o repetida");
    }
    unicas.add(capacidad);
  }
  return Object.freeze([...unicas]);
}

export function tieneCapacidadDietas(capacidades, capacidad) {
  return Array.isArray(capacidades) && capacidades.includes(capacidad);
}

export function validarPanelDietas(datos, titularEsperado = "", capacidadesEsperadas = null) {
  if (!datos || typeof datos !== "object" || Array.isArray(datos)
    || datos.esquema !== ESQUEMA_PANEL_DIETAS || !datos.origen || typeof datos.origen !== "object"
    || !Array.isArray(datos.etapas) || !Array.isArray(datos.comisiones) || !datos.politica) {
    throw new Error("panel de Dietas no valido");
  }
  if (datos.origen.demostracion === true && datos.origen.efectos_reales !== false) {
    throw new Error("un panel demostrativo de Dietas no puede declarar efectos reales");
  }
  if (datos.borrador_inicial !== undefined
    && (!datos.borrador_inicial || typeof datos.borrador_inicial !== "object" || Array.isArray(datos.borrador_inicial))) {
    throw new Error("borrador inicial de Dietas no valido");
  }
  if (datos.etapas.length !== ETAPAS_DIETAS.size || datos.etapas.some((etapa) => !ETAPAS_DIETAS.has(etapa))) {
    throw new Error("etapas de Dietas no validas");
  }
  const capacidades = capacidadesEsperadas === null ? null : validarCapacidadesDietas(capacidadesEsperadas);
  const puedeConsultar = capacidades === null || tieneCapacidadDietas(capacidades, CAPACIDAD_CONSULTAR_GASTO);
  const puedeConsultarRutas = capacidades === null || tieneCapacidadDietas(capacidades, CAPACIDAD_CONSULTAR_RUTA);
  if (!puedeConsultar && datos.comisiones.length) {
    throw new Error("el panel contiene expedientes sin capacidad de consulta");
  }
  const referencias = new Set();
  for (const item of datos.comisiones) {
    const referencia = texto(item?.referencia, "referencia de comision", 100);
    if (referencias.has(referencia)) throw new Error("referencia de comision repetida");
    referencias.add(referencia);
    if (!Array.isArray(item.ruta) || !Array.isArray(item.historial)
      || !ESTADOS_DIETAS.has(item.estado) || !SIGUIENTES_ACTUACIONES.has(item.siguiente_actuacion)) {
      throw new Error("comision de Dietas incompleta");
    }
    if (titularEsperado && item.titular_ref !== titularEsperado) {
      throw new Error("el panel de Dietas contiene un expediente ajeno al contexto de actor");
    }
    if (!puedeConsultarRutas && (item.ruta.length || item.kilometros !== null || item.kilometraje_euros !== null)) {
      throw new Error("el panel contiene rutas sin capacidad de consulta");
    }
    if (puedeConsultarRutas && item.geometria_ruta !== null && item.geometria_ruta !== undefined) {
      validarGeometriaRutaDietas(item.geometria_ruta, item.ruta);
    }
    if (!puedeConsultarRutas && item.geometria_ruta !== null) {
      throw new Error("el panel contiene coordenadas sin capacidad de consulta");
    }
    if (item.historial.some((evento) => !ESTADOS_DIETAS.has(evento?.estado))) {
      throw new Error("historial de Dietas no valido");
    }
  }
  return copiarDietas(datos);
}

export function validarComandoDietas(comando) {
  if (!comando || typeof comando !== "object" || Array.isArray(comando)) throw new Error("comando de Dietas no valido");
  const tipo = texto(comando.tipo, "tipo de comando", 40);
  if (tipo === "crear_borrador") {
    if (!comando.campos || typeof comando.campos !== "object" || Array.isArray(comando.campos)) {
      throw new Error("campos de comision no validos");
    }
    return Object.freeze({ tipo, campos: Object.freeze({ ...comando.campos }) });
  }
  if (tipo === "enviar_validacion") {
    return Object.freeze({ tipo, referencia: texto(comando.referencia, "referencia", 100) });
  }
  throw new Error("comando de Dietas no permitido");
}
