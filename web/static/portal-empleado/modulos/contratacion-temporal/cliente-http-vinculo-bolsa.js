/** El vínculo usa el transporte CT existente y su recuperación de actos inciertos. */
import { referenciaLlamamientoValida } from "./contrato-llamamiento.js";

export const RUTA_VINCULO_BOLSA = "/api/vec/contratacion-temporal/expedientes/vinculos-bolsa";
const REFERENCIAS = ["expediente_ref", "bolsa_ref", "llamamiento_ref", "recibo_emision_ref"];

export function validarSolicitudVinculoBolsa(entrada) {
  if (!entrada || !REFERENCIAS.every((clave) => referenciaLlamamientoValida(entrada[clave]))
    || !Number.isSafeInteger(entrada.version_esperada) || entrada.version_esperada < 1
    || typeof entrada.clave_idempotencia !== "string"
    || !/^[A-Za-z0-9][A-Za-z0-9._:-]{7,159}$/u.test(entrada.clave_idempotencia)) {
    throw new TypeError("vinculo_bolsa_solicitud_invalida");
  }
  return Object.freeze(Object.fromEntries([...REFERENCIAS, "version_esperada", "clave_idempotencia"].map(
    (clave) => [clave, entrada[clave]],
  )));
}

export function validarReciboVinculoBolsa(recibo, entrada) {
  if (!recibo || !REFERENCIAS.every((clave) => recibo[clave] === entrada[clave])
    || !referenciaLlamamientoValida(recibo.recibo_vinculo_ref)
    || typeof recibo.reutilizado !== "boolean"
    || typeof recibo.vinculado_en !== "string" || !recibo.vinculado_en.endsWith("Z")
    || !Number.isFinite(Date.parse(recibo.vinculado_en))) throw new TypeError("vinculo_bolsa_recibo_invalido");
  return Object.freeze({ ...recibo });
}

export function crearVinculoBolsaClienteHTTP({ ejecutar, validarOpciones } = {}) {
  if (typeof ejecutar !== "function" || typeof validarOpciones !== "function") throw new TypeError("vinculo_bolsa_dependencias_invalidas");
  return Object.freeze({
    vincularLlamamientoBolsa(solicitud, opciones) {
      const entrada = validarSolicitudVinculoBolsa(solicitud);
      const { signal } = validarOpciones(opciones);
      return ejecutar({ ruta: RUTA_VINCULO_BOLSA, entrada, signal, estadoEsperado: [200, 201],
        maximoSolicitud: 8192, maximoRespuesta: 8192, efecto: true,
        validarRespuesta: (recibo) => validarReciboVinculoBolsa(recibo, entrada),
        rechazoDeterminado: (error) => error?.envelopeValido === true
          && ["400:peticion_no_valida", "400:peticion_no_permitida", "401:autenticacion_requerida", "403:acceso_denegado", "409:conflicto", "409:vinculo_en_conflicto", "422:contenido_no_valido"].includes(`${error.estado}:${error.codigo}`),
      });
    },
  });
}
