"use strict";

/**
 * Acceso a la inscripción desde la ficha pública. Solo se ofrece cuando la
 * convocatoria tiene un plazo de inscripción que el servidor declara abierto;
 * la solicitud se prepara en el área personal, tras identificarse. El destino
 * es el único enlace revisado de esta superficie hacia el área personal.
 */
((global) => {
  const DESTINO = "/area-personal/?vista=solicitud&id=";
  const PATRON_IDENTIFICADOR = /^[a-z0-9][a-z0-9-]{2,79}$/;

  function plazoInscripcionAbierto(plazos) {
    return Array.isArray(plazos) && plazos.some((plazo) => plazo?.tipo?.clave === "inscripcion" && plazo?.situacion === "abierto");
  }

  /** Dirección de la solicitud para la ficha recibida, o "" si no procede. */
  function enlace(datos) {
    const identificador = datos?.convocatoria?.identificador_publico;
    if (typeof identificador !== "string" || !PATRON_IDENTIFICADOR.test(identificador)) return "";
    return plazoInscripcionAbierto(datos?.plazos) ? `${DESTINO}${encodeURIComponent(identificador)}` : "";
  }

  global.VECBolsaInscripcion = Object.freeze({ enlace });
})(globalThis);
