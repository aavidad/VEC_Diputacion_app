/**
 * Panel de las comprobaciones automáticas de la vía de cobertura (fase 3).
 * Solo presenta propuestas; no decide ni envía nada. La procedencia de cada
 * regla se muestra únicamente en la ayuda que abre el botón «?».
 */

export const ACCION_AYUDA_AVISOS_VIA = "ayuda-avisos-via";
const ID_AYUDA = "ct-avisos-via-ayuda";

function escaparHTML(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#039;");
}

function formatearFecha(fecha, formateadorFechas) {
  if (!fecha) return "";
  const instante = new Date(`${fecha}T00:00:00Z`);
  return Number.isNaN(instante.getTime()) ? fecha : formateadorFechas.format(instante);
}

function detalleAviso(aviso, t, formateadorFechas) {
  const fecha = (valor) => formatearFecha(valor, formateadorFechas);
  if (aviso.clave === "bolsa_agotada_provisionalmente") {
    return [t("avisos_via_bolsa_agotada_provisionalmente_detalle", {
      disponibles: aviso.disponibles, integrantes: aviso.integrantes, umbral: aviso.umbral,
    })];
  }
  if (aviso.clave === "propuesta_oferta_sae") {
    const lineas = [t("avisos_via_propuesta_oferta_sae_detalle", {
      meses: aviso.duracion_maxima_meses, fin_maximo: fecha(aviso.fin_maximo),
    })];
    if (aviso.excede_duracion) {
      lineas.push(t("avisos_via_propuesta_oferta_sae_excede", { fin_previsto: fecha(aviso.fin_previsto) }));
    }
    return lineas;
  }
  return aviso.motivos.map((motivo) => t(`avisos_via_motivo_${motivo}`, {
    vigencia_hasta: fecha(aviso.vigencia_hasta), constituida_en: fecha(aviso.constituida_en),
  }));
}

function tonoAviso(aviso) {
  return aviso.clave === "propuesta_oferta_sae" ? "informacion" : "aviso";
}

function renderizarAviso(aviso, t, formateadorFechas) {
  const tono = tonoAviso(aviso);
  const estado = tono === "informacion" ? "avisos_via_estado_propuesta" : "avisos_via_estado_aviso";
  const lineas = detalleAviso(aviso, t, formateadorFechas);
  return `<li class="ct-aviso-via ct-tono-${tono}" data-ct-aviso-via="${escaparHTML(aviso.clave)}">
    <div class="ct-aviso-via-cabecera">
      <span class="ct-aviso-via-pastilla">${escaparHTML(t(estado))}</span>
      <strong>${escaparHTML(t(`avisos_via_${aviso.clave}_titulo`))}</strong>
    </div>
    ${lineas.map((linea) => `<p>${escaparHTML(linea)}</p>`).join("")}
  </li>`;
}

function renderizarProcedencia(avisos, t) {
  const vistas = new Map();
  for (const aviso of avisos) {
    for (const regla of aviso.reglas) vistas.set(regla.clave, regla);
  }
  if (vistas.size === 0) return `<p>${escaparHTML(t("avisos_via_ayuda_sin_reglas"))}</p>`;
  return `<dl class="ct-avisos-via-reglas">${[...vistas.values()].map((regla) => {
    const origen = regla.origen === "reglamento"
      ? t("avisos_via_regla_reglamento", { articulo: regla.articulo })
      : t("avisos_via_regla_ejemplo");
    const partes = [
      `<p class="ct-avisos-via-origen">${escaparHTML(origen)}</p>`,
      `<p>${escaparHTML(regla.descripcion)}</p>`,
      `<p>${escaparHTML(regla.norma)}</p>`,
    ];
    if (regla.parte_ejemplo) {
      partes.push(`<p>${escaparHTML(t("avisos_via_regla_parte_ejemplo", { texto: regla.parte_ejemplo }))}</p>`);
    }
    return `<div data-ct-aviso-via-regla="${escaparHTML(regla.clave)}"><dt>${escaparHTML(regla.etiqueta)}</dt><dd>${partes.join("")}</dd></div>`;
  }).join("")}</dl>`;
}

/**
 * Devuelve el HTML del panel o una cadena vacía si la propuesta no trae
 * comprobaciones (sin catálogo de reglas la pantalla queda como hoy).
 */
export function renderizarAvisosViaCobertura(avisosVia, t, { formateadorFechas, ayudaAbierta = false } = {}) {
  if (!avisosVia || !formateadorFechas) return "";
  let cuerpo;
  if (avisosVia.estado === "no_disponible") {
    cuerpo = `<p class="ct-avisos-via-vacio" role="status">${escaparHTML(t("avisos_via_no_disponible"))}</p>`;
  } else if (avisosVia.estado === "sin_bolsa") {
    cuerpo = `<p class="ct-avisos-via-vacio">${escaparHTML(t("avisos_via_sin_bolsa"))}</p>`;
  } else if (avisosVia.avisos.length === 0) {
    cuerpo = `<p class="ct-avisos-via-vacio">${escaparHTML(t("avisos_via_ninguno"))}</p>`;
  } else {
    cuerpo = `<ul class="ct-avisos-via-lista">${avisosVia.avisos
      .map((aviso) => renderizarAviso(aviso, t, formateadorFechas)).join("")}</ul>`;
  }
  const etiquetaAyuda = t(ayudaAbierta ? "avisos_via_ayuda_cerrar" : "avisos_via_ayuda_abrir");
  return `<section class="ct-avisos-via" data-ct-avisos-via aria-labelledby="ct-avisos-via-titulo">
    <header class="ct-avisos-via-titulo">
      <h3 id="ct-avisos-via-titulo">${escaparHTML(t("avisos_via_titulo"))}</h3>
      <button type="button" class="ct-avisos-via-boton-ayuda" data-ct-cobertura-accion="${ACCION_AYUDA_AVISOS_VIA}"
        aria-expanded="${ayudaAbierta ? "true" : "false"}" aria-controls="${ID_AYUDA}"
        aria-label="${escaparHTML(etiquetaAyuda)}" title="${escaparHTML(etiquetaAyuda)}"><span aria-hidden="true">?</span></button>
    </header>
    <div class="ct-avisos-via-ayuda" id="${ID_AYUDA}"${ayudaAbierta ? "" : " hidden"}>
      <p>${escaparHTML(t("avisos_via_ayuda_intro"))}</p>
      ${renderizarProcedencia(avisosVia.avisos, t)}
    </div>
    <div class="ct-avisos-via-cuerpo">${cuerpo}</div>
  </section>`;
}
