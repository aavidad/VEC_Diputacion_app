/**
 * Cliente HTTP para la consulta pública de bolsas y lista de aspirantes (B10).
 * Consulta anónima sin autenticación y sin persistencia en navegador.
 */

import {
  validarRespuestaBolsasPublicas,
  validarRespuestaListaPublica,
  PATRON_DOCUMENTO_ENMASCARADO,
} from "./contrato-publico-bolsas.js";

// El enrutador del servidor solo acepta rutas canónicas (sin secuencias
// porcentuales): las referencias llevan ":" y "-", legales en un segmento de
// ruta, así que se envían sin escapar y solo se escapa lo que no es legal.
function segmentoRuta(referencia) {
  return encodeURIComponent(String(referencia ?? "").trim()).replace(/%3A/gi, ":");
}

const RUTA_BOLSAS_PUBLICAS = "/api/publico/bolsa/bolsas";

/**
 * Consulta la relación de bolsas públicas activas.
 * GET /api/publico/bolsa/bolsas
 */
export async function consultarBolsasPublicas({ fetchImpl = globalThis.fetch } = {}) {
  const respuesta = await fetchImpl(RUTA_BOLSAS_PUBLICAS, {
    method: "GET",
    credentials: "omit",
    headers: {
      Accept: "application/json",
    },
  });

  if (!respuesta.ok) {
    let mensaje = `Error HTTP ${respuesta.status}`;
    try {
      const cuerpoError = await respuesta.json();
      if (cuerpoError && cuerpoError.error && cuerpoError.error.mensaje) {
        mensaje = cuerpoError.error.mensaje;
      }
    } catch {
      // Ignorar fallo de decodificación JSON de error
    }
    const error = new Error(mensaje);
    error.status = respuesta.status;
    throw error;
  }

  const json = await respuesta.json();
  return validarRespuestaBolsasPublicas(json);
}

/**
 * Consulta la lista ordenada de aspirantes en una bolsa pública.
 * GET /api/publico/bolsa/bolsas/{bolsa_ref}/lista?cursor=&limite=&documento=
 */
export async function consultarListaBolsaPublica({
  bolsa_ref,
  cursor = "",
  limite = 50,
  documento = "",
  fetchImpl = globalThis.fetch,
} = {}) {
  if (!bolsa_ref || typeof bolsa_ref !== "string" || bolsa_ref.trim() === "") {
    throw new Error("Se requiere bolsa_ref para consultar la lista pública");
  }

  const limiteSeguro = Math.min(Math.max(Number(limite) || 50, 1), 100);
  const parametros = new URLSearchParams();
  parametros.set("limite", String(limiteSeguro));

  if (cursor && typeof cursor === "string" && cursor.trim() !== "") {
    parametros.set("cursor", cursor.trim());
  }

  if (documento && typeof documento === "string") {
    const docLimpio = documento.trim();
    if (PATRON_DOCUMENTO_ENMASCARADO.test(docLimpio)) {
      parametros.set("documento", docLimpio);
    }
  }

  const ruta = `${RUTA_BOLSAS_PUBLICAS}/${segmentoRuta(bolsa_ref)}/lista?${parametros.toString()}`;

  const respuesta = await fetchImpl(ruta, {
    method: "GET",
    credentials: "omit",
    headers: {
      Accept: "application/json",
    },
  });

  if (!respuesta.ok) {
    let mensaje = `Error HTTP ${respuesta.status}`;
    try {
      const cuerpoError = await respuesta.json();
      if (cuerpoError && cuerpoError.error && cuerpoError.error.mensaje) {
        mensaje = cuerpoError.error.mensaje;
      }
    } catch {
      // Ignorar fallo de decodificación JSON de error
    }
    const error = new Error(mensaje);
    error.status = respuesta.status;
    throw error;
  }

  const json = await respuesta.json();
  return validarRespuestaListaPublica(json);
}

export const VECPublicoBolsasAPI = Object.freeze({
  consultarBolsasPublicas,
  consultarListaBolsaPublica,
});

if (typeof globalThis !== "undefined") {
  globalThis.VECPublicoBolsasAPI = VECPublicoBolsasAPI;
}
