/**
 * Marcas de una participación en el cuadro, la ficha y la selección de un
 * llamamiento (Bolsa 000041): ya presta servicios (b16), en revisión (duda 18)
 * y encadenamiento (b17). Las calcula el servidor con los parámetros del
 * catálogo; aquí solo se validan y se rotulan. La explicación vive en la
 * ayuda «?».
 */

export const MENSAJES_MARCAS_BOLSA_ES = Object.freeze({
  presta_servicios_aviso: "Ya presta servicios",
  presta_servicios_excluir: "Ya presta servicios: no se puede llamar",
  en_revision: "En revisión",
  revision_renuncia_pendiente: "Renuncia comunicada pendiente de confirmar",
  revision_solicitud_pendiente: "Solicitud del portal pendiente de validar",
  revision_baja_propuesta: "Baja propuesta por intentos sin contacto",
  encadenamiento: "Encadenamiento",
  encadenamiento_detalle: "{dias} días con contrato en los últimos {ventana} meses (umbral: {umbral} meses)",
  ficha_presta_servicios: "Otras participaciones",
  ficha_revision: "Revisión pendiente",
  ficha_encadenamiento: "Encadenamiento de contratos",
  no_seleccionable_aria: "No seleccionable: ya presta servicios",
});

const CLAVES = Object.freeze(Object.keys(MENSAJES_MARCAS_BOLSA_ES));
const MODOS = Object.freeze(["aviso", "excluir"]);
const REVISIONES = Object.freeze(["renuncia_pendiente", "solicitud_pendiente", "baja_propuesta"]);
const FORMATO_NUMERO = new Intl.NumberFormat("es-ES");

export function traducirMarcasBolsa(clave, variables = {}) {
  if (!CLAVES.includes(clave)) throw new Error(`clave i18n de marcas de Bolsa desconocida: ${clave}`);
  return MENSAJES_MARCAS_BOLSA_ES[clave].replace(/\{([a-z_]+)\}/g, (_coincidencia, variable) => String(variables[variable] ?? ""));
}

function enteroPositivo(valor) {
  return Number.isSafeInteger(valor) && valor > 0;
}

/** Valida el bloque «marcas» del contrato de candidatos (cerrado). */
export function validarMarcasCandidato(marcas) {
  if (!marcas || typeof marcas !== "object" || Array.isArray(marcas)) throw new Error("marcas no es un objeto válido");
  const campos = Object.keys(marcas);
  if (campos.length !== 3 || !["presta_servicios", "en_revision", "encadenamiento"].every((campo) => Object.hasOwn(marcas, campo))) {
    throw new Error("marcas no respeta el contrato cerrado");
  }
  if (marcas.presta_servicios !== null && !MODOS.includes(marcas.presta_servicios)) throw new Error("presta_servicios no reconocido");
  if (marcas.en_revision !== null && !REVISIONES.includes(marcas.en_revision)) throw new Error("en_revision no reconocido");
  let encadenamiento = null;
  if (marcas.encadenamiento !== null) {
    const e = marcas.encadenamiento;
    if (!e || typeof e !== "object" || Object.keys(e).length !== 3 || !enteroPositivo(e.dias_acumulados) || !enteroPositivo(e.umbral_meses) || !enteroPositivo(e.ventana_meses)) {
      throw new Error("encadenamiento no válido");
    }
    encadenamiento = Object.freeze({ dias_acumulados: e.dias_acumulados, umbral_meses: e.umbral_meses, ventana_meses: e.ventana_meses });
  }
  return Object.freeze({ presta_servicios: marcas.presta_servicios, en_revision: marcas.en_revision, encadenamiento });
}

/** Una persona que ya presta servicios con el catálogo en «excluir» no se puede llamar. */
export function seleccionableEnLlamamiento(candidato) {
  return candidato?.marcas?.presta_servicios !== "excluir";
}

function detalleEncadenamiento(e) {
  return traducirMarcasBolsa("encadenamiento_detalle", { dias: FORMATO_NUMERO.format(e.dias_acumulados), ventana: FORMATO_NUMERO.format(e.ventana_meses), umbral: FORMATO_NUMERO.format(e.umbral_meses) });
}

/** Rótulos junto a la situación en el cuadro y en la selección del llamamiento. */
export function renderizarChipsMarcas(candidato, escaparHTML) {
  const marcas = candidato?.marcas;
  if (!marcas) return "";
  const chips = [];
  if (marcas.en_revision) {
    chips.push(`<span class="estado-chip advertencia" title="${escaparHTML(traducirMarcasBolsa(`revision_${marcas.en_revision}`))}">${escaparHTML(traducirMarcasBolsa("en_revision"))}</span>`);
  }
  if (marcas.presta_servicios) {
    chips.push(`<span class="estado-chip ${marcas.presta_servicios === "excluir" ? "peligro" : "info"}">${escaparHTML(traducirMarcasBolsa(`presta_servicios_${marcas.presta_servicios}`))}</span>`);
  }
  if (marcas.encadenamiento) {
    chips.push(`<span class="estado-chip advertencia" title="${escaparHTML(detalleEncadenamiento(marcas.encadenamiento))}">${escaparHTML(traducirMarcasBolsa("encadenamiento"))}</span>`);
  }
  return chips.length ? ` ${chips.join(" ")}` : "";
}

/** Filas de la ficha de participación con el detalle de cada marca. */
export function renderizarMarcasFicha(candidato, escaparHTML) {
  const marcas = candidato?.marcas;
  if (!marcas) return "";
  const fila = (titulo, texto) => `<div class="fila-resumen" data-bolsa-marca><dt>${escaparHTML(traducirMarcasBolsa(titulo))}</dt><dd>${escaparHTML(texto)}</dd></div>`;
  return [
    marcas.en_revision ? fila("ficha_revision", traducirMarcasBolsa(`revision_${marcas.en_revision}`)) : "",
    marcas.presta_servicios ? fila("ficha_presta_servicios", traducirMarcasBolsa(`presta_servicios_${marcas.presta_servicios}`)) : "",
    marcas.encadenamiento ? fila("ficha_encadenamiento", detalleEncadenamiento(marcas.encadenamiento)) : "",
  ].join("");
}
