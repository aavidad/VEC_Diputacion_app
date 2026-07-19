/**
 * Adaptador de cálculo de rutas exclusivamente demostrativo.
 *
 * Contiene una instantánea sintética y versionada del catálogo provincial para
 * que la presentación sea reproducible y aislada. No es fuente de verdad, no
 * consulta red ni conserva datos. La simulación nunca es prueba de distancia ni
 * permite liquidar una comisión de servicio. En producción se sustituye por el
 * puerto de rutas que autoriza el servidor y consulta el OSRM interno versionado.
 */

import {
  CAPACIDAD_CONSULTAR_RUTA,
  ESQUEMA_CALCULO_RUTA_DIETAS,
  ESQUEMA_GEOMETRIA_RUTA_DIETAS,
  exigirContextoActorDietas,
  tieneCapacidadDietas,
  validarCalculoRutaDietas,
  validarCapacidadesDietas,
  validarGeometriaRutaDietas,
  validarSolicitudRutaDietas,
} from "./contrato.js";
import {
  obtenerCatalogoRutasProvincial,
  resolverPuntosRutasProvincial,
} from "./catalogo-rutas-provincial.js";

const VERSION_GRAFO = "pendiente-osrm-ngmep-2026";
const MOTOR_DEMO = "simulacion_osrm_demo";
const SEMILLAS_TRAMOS = new Map([
  ["18087>18003", Object.freeze({ kilometros: 13.4, duracion_minutos: 18 })],
  ["18087>NGMEP-MECINA-BOMBARON", Object.freeze({ kilometros: 98.4, duracion_minutos: 93 })],
  ["NGMEP-MECINA-BOMBARON>18087", Object.freeze({ kilometros: 98.4, duracion_minutos: 93 })],
  ["18087>18116", Object.freeze({ kilometros: 45.9, duracion_minutos: 41 })],
  ["18116>NGMEP-MECINA-BOMBARON", Object.freeze({ kilometros: 49.5, duracion_minutos: 61 })],
  ["18087>18184", Object.freeze({ kilometros: 54.7, duracion_minutos: 47 })],
  ["18184>NGMEP-MECINA-BOMBARON", Object.freeze({ kilometros: 52, duracion_minutos: 59 })],
  ["18003>NGMEP-MECINA-BOMBARON", Object.freeze({ kilometros: 124.6, duracion_minutos: 122 })],
  ["NGMEP-MECINA-BOMBARON>18140", Object.freeze({ kilometros: 83.2, duracion_minutos: 96 })],
  ["18140>18087", Object.freeze({ kilometros: 70.4, duracion_minutos: 55 })],
  ["18087>18140", Object.freeze({ kilometros: 70.4, duracion_minutos: 55 })],
  ["18087>18122", Object.freeze({ kilometros: 54, duracion_minutos: 45 })],
  ["18087>18089", Object.freeze({ kilometros: 60.5, duracion_minutos: 50 })],
  ["18087>18023", Object.freeze({ kilometros: 107.2, duracion_minutos: 80 })],
  ["18087>18017", Object.freeze({ kilometros: 80.5, duracion_minutos: 65 })],
  ["18087>18147", Object.freeze({ kilometros: 55.8, duracion_minutos: 58 })],
]);

const PERFILES_ALTERNATIVA = Object.freeze([
  Object.freeze({
    etiqueta: "Simulación principal (DEMO)",
    factor_kilometros: 1,
    factor_duracion: 1,
    curvatura: 0.009,
  }),
  Object.freeze({
    etiqueta: "Alternativa 2 (DEMO)",
    factor_kilometros: 1.07,
    factor_duracion: 0.92,
    curvatura: -0.014,
  }),
  Object.freeze({
    etiqueta: "Alternativa 3 (DEMO)",
    factor_kilometros: 0.95,
    factor_duracion: 1.16,
    curvatura: 0.018,
  }),
]);

function esObjetoPlano(valor) {
  if (!valor || typeof valor !== "object" || Array.isArray(valor)) return false;
  const prototipo = Object.getPrototypeOf(valor);
  return prototipo === Object.prototype || prototipo === null;
}

function exigirClavesExactas(valor, esperadas, nombre) {
  if (!esObjetoPlano(valor)) throw new TypeError(nombre + " no valida");
  const recibidas = Object.keys(valor).sort();
  const previstas = [...esperadas].sort();
  if (recibidas.length !== previstas.length
    || recibidas.some((clave, indice) => clave !== previstas[indice])) {
    throw new TypeError(nombre + " no respeta el contrato cerrado");
  }
}

function validarSolicitud(solicitud) {
  return validarSolicitudRutaDietas(solicitud, obtenerCatalogoRutasProvincial());
}

function redondear(valor, decimales = 1) {
  const factor = 10 ** decimales;
  return Math.round((valor + Number.EPSILON) * factor) / factor;
}

function distanciaGeodesica(origen, destino) {
  const aRadianes = (grados) => grados * Math.PI / 180;
  const diferenciaLatitud = aRadianes(destino.latitud - origen.latitud);
  const diferenciaLongitud = aRadianes(destino.longitud - origen.longitud);
  const latitudOrigen = aRadianes(origen.latitud);
  const latitudDestino = aRadianes(destino.latitud);
  const senoLatitud = Math.sin(diferenciaLatitud / 2);
  const senoLongitud = Math.sin(diferenciaLongitud / 2);
  const termino = senoLatitud ** 2
    + Math.cos(latitudOrigen) * Math.cos(latitudDestino) * senoLongitud ** 2;
  return 6_371 * 2 * Math.atan2(Math.sqrt(termino), Math.sqrt(1 - termino));
}

function tramoBase(origen, destino) {
  const directo = SEMILLAS_TRAMOS.get(origen.codigo + ">" + destino.codigo);
  const inverso = SEMILLAS_TRAMOS.get(destino.codigo + ">" + origen.codigo);
  if (directo || inverso) return directo || inverso;
  const kilometros = Math.max(1, redondear(distanciaGeodesica(origen, destino) * 1.22));
  return Object.freeze({
    kilometros,
    duracion_minutos: Math.max(3, Math.round(kilometros / 68 * 60 + 5)),
  });
}

function crearGeometria(puntos, perfil, indiceAlternativa) {
  const paradas = puntos.map((punto) => ({
    etiqueta: punto.nombre,
    latitud: punto.latitud,
    longitud: punto.longitud,
  }));
  const trazado = [];
  for (let indice = 0; indice < puntos.length - 1; indice += 1) {
    const origen = puntos[indice];
    const destino = puntos[indice + 1];
    if (indice === 0) trazado.push([origen.latitud, origen.longitud]);
    const deltaLatitud = destino.latitud - origen.latitud;
    const deltaLongitud = destino.longitud - origen.longitud;
    const norma = Math.max(Math.hypot(deltaLatitud, deltaLongitud), 0.001);
    const signo = (indice + indiceAlternativa) % 2 === 0 ? 1 : -1;
    const desplazamiento = perfil.curvatura * signo;
    trazado.push([
      origen.latitud + deltaLatitud / 3 - deltaLongitud / norma * desplazamiento,
      origen.longitud + deltaLongitud / 3 + deltaLatitud / norma * desplazamiento,
    ]);
    trazado.push([
      origen.latitud + deltaLatitud * 2 / 3 - deltaLongitud / norma * desplazamiento,
      origen.longitud + deltaLongitud * 2 / 3 + deltaLatitud / norma * desplazamiento,
    ]);
    trazado.push([destino.latitud, destino.longitud]);
  }
  return validarGeometriaRutaDietas({
    esquema: ESQUEMA_GEOMETRIA_RUTA_DIETAS,
    origen: "sintetica_demo",
    liquidable: false,
    paradas,
    trazado,
  }, puntos.map((punto) => punto.nombre));
}

function crearReferencia(codigos, alternativas) {
  const semilla = VERSION_GRAFO + "|" + codigos.join(">") + "|A" + alternativas;
  let huella = 2_166_136_261;
  for (let indice = 0; indice < semilla.length; indice += 1) {
    huella ^= semilla.charCodeAt(indice);
    huella = Math.imul(huella, 16_777_619);
  }
  return "DEMO-RUTA-" + (huella >>> 0).toString(16).toUpperCase().padStart(8, "0");
}

function calcularAlternativa(puntos, referenciaCalculo, perfil, indiceAlternativa) {
  const tramos = [];
  for (let indice = 0; indice < puntos.length - 1; indice += 1) {
    const origen = puntos[indice];
    const destino = puntos[indice + 1];
    const base = tramoBase(origen, destino);
    tramos.push({
      indice,
      origen_codigo: origen.codigo,
      origen_nombre: origen.nombre,
      destino_codigo: destino.codigo,
      destino_nombre: destino.nombre,
      kilometros: redondear(base.kilometros * perfil.factor_kilometros),
      duracion_minutos: Math.max(1, Math.round(base.duracion_minutos * perfil.factor_duracion)),
    });
  }
  return {
    referencia: referenciaCalculo + "-A" + (indiceAlternativa + 1),
    recomendada: indiceAlternativa === 0,
    etiqueta: perfil.etiqueta,
    kilometros: redondear(tramos.reduce((total, tramo) => total + tramo.kilometros, 0)),
    duracion_minutos: tramos.reduce((total, tramo) => total + tramo.duracion_minutos, 0),
    tramos,
    geometria: crearGeometria(puntos, perfil, indiceAlternativa),
  };
}

/**
 * Compone el puerto DEMO de rutas. La capacidad solo proyecta la interfaz:
 * toda autorización productiva debe repetirse en el servidor.
 */
export function crearCalculadorRutasDietasPresentacion(opciones = {}) {
  exigirClavesExactas(opciones, ["contextoActor", "capacidades"], "opciones del calculador");
  const contexto = exigirContextoActorDietas(opciones.contextoActor);
  const capacidades = validarCapacidadesDietas(opciones.capacidades);
  if (contexto.demostracion !== true) {
    throw new Error("el calculador de presentación exige un contexto DEMO");
  }
  if (!tieneCapacidadDietas(capacidades, CAPACIDAD_CONSULTAR_RUTA)) {
    throw new Error("falta capacidad para consultar rutas de Dietas");
  }

  return Object.freeze({
    obtenerCatalogo() {
      return obtenerCatalogoRutasProvincial();
    },
    calcular(solicitud) {
      const validada = validarSolicitud(solicitud);
      const puntos = resolverPuntosRutasProvincial(validada.paradas);
      const referencia = crearReferencia(validada.paradas, validada.alternativas);
      return validarCalculoRutaDietas({
        esquema: ESQUEMA_CALCULO_RUTA_DIETAS,
        referencia,
        demostracion: true,
        liquidable: false,
        motor: MOTOR_DEMO,
        version_grafo: VERSION_GRAFO,
        alternativas: PERFILES_ALTERNATIVA
          .slice(0, validada.alternativas)
          .map((perfil, indice) => calcularAlternativa(puntos, referencia, perfil, indice)),
      }, validada);
    },
  });
}
