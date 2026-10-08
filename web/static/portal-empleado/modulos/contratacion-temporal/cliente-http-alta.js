import {
  validarCatalogosAlta,
  validarComandoAlta,
  validarComandoAltaNecesidad,
  ESQUEMA_ALTA_NECESIDAD,
  validarReciboAlta,
} from "./contrato.js?v=20261008-alta-rechazo-v2";

const MAXIMO_SOLICITUD_ALTA_BYTES = 256 * 1024;
const MAXIMO_RESPUESTA_ALTA_BYTES = 16 * 1024;
// Los catálogos del alta traen, además de centros y categorías de la RPT
// (unos 30 KiB), la relación de documentos y datos por vía de cobertura, que el
// catálogo de vías limita a 512 elementos (menos de 64 KiB).
export const MAXIMO_RESPUESTA_CATALOGOS_BYTES = 128 * 1024;

export const RUTAS_ALTA_CONTRATACION_TEMPORAL = Object.freeze({
  alta: "/api/vec/contratacion-temporal/solicitudes",
  catalogosAlta: "/api/vec/contratacion-temporal/catalogos-alta",
});

export function crearAltaClienteHTTP({ ejecutar, validarOpciones } = {}) {
  if (typeof ejecutar !== "function" || typeof validarOpciones !== "function") {
    throw new TypeError("dependencias HTTP del alta no disponibles");
  }

  async function registrarSolicitud(comando, opciones) {
    const { signal } = validarOpciones(opciones);
    const entrada = comando?.esquema === ESQUEMA_ALTA_NECESIDAD
      ? validarComandoAltaNecesidad(comando) : validarComandoAlta(comando);
    return ejecutar({
      metodo: "POST",
      ruta: RUTAS_ALTA_CONTRATACION_TEMPORAL.alta,
      entrada,
      signal,
      estadoEsperado: 201,
      maximoSolicitud: MAXIMO_SOLICITUD_ALTA_BYTES,
      maximoRespuesta: MAXIMO_RESPUESTA_ALTA_BYTES,
      validarRespuesta: validarReciboAlta,
      efecto: true,
    });
  }

  async function obtenerCatalogosAlta(opciones) {
    const { signal } = validarOpciones(opciones);
    return ejecutar({
      metodo: "GET",
      ruta: RUTAS_ALTA_CONTRATACION_TEMPORAL.catalogosAlta,
      signal,
      estadoEsperado: 200,
      maximoRespuesta: MAXIMO_RESPUESTA_CATALOGOS_BYTES,
      validarRespuesta: validarCatalogosAlta,
      efecto: false,
    });
  }

  async function obtenerCatalogosNecesidadesAlta(opciones) {
    const { signal } = validarOpciones(opciones);
    return ejecutar({
      metodo: "GET",
      ruta: `${RUTAS_ALTA_CONTRATACION_TEMPORAL.catalogosAlta}?version=2`,
      signal,
      estadoEsperado: 200,
      maximoRespuesta: MAXIMO_RESPUESTA_CATALOGOS_BYTES,
      validarRespuesta: validarCatalogosAlta,
      efecto: false,
    });
  }

  return Object.freeze({ registrarSolicitud, obtenerCatalogosAlta,
    obtenerCatalogosNecesidadesAlta });
}
