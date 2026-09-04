import {
  validarCatalogosAlta,
  validarComandoAlta,
  validarReciboAlta,
} from "./contrato.js";

const MAXIMO_SOLICITUD_ALTA_BYTES = 256 * 1024;
const MAXIMO_RESPUESTA_ALTA_BYTES = 16 * 1024;
const MAXIMO_RESPUESTA_CATALOGOS_BYTES = 64 * 1024;

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
    const entrada = validarComandoAlta(comando);
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

  return Object.freeze({ registrarSolicitud, obtenerCatalogosAlta });
}
