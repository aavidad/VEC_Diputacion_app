/** Dos entradas de consulta de la vía de cobertura dentro del expediente RRHH. */

const DATOS_COMUNES = Object.freeze([
  ["centro", "cabecera_centro"],
  ["categoria", "cabecera_categoria"],
  ["grupo_subgrupo", "cabecera_grupo_subgrupo"],
  ["modalidad", "cabecera_modalidad"],
  ["motivo", "cabecera_motivo"],
  ["periodo", "cabecera_periodo_solicitado"],
  ["resultado_rc", "cabecera_resultado_rc"],
  ["via_cobertura", "cabecera_via_cobertura"],
  ["unidad", "cabecera_unidad_asignada"],
]);

const DOCUMENTOS_BOLSA = Object.freeze([
  "informe_definitivo_titulo", "resolucion_titulo", "diligencia_titulo",
  "toma_posesion_titulo", "notificacion_titulo", "comunicacion_centro_titulo",
]);

const DOCUMENTOS_SAE = Object.freeze([
  "vias_sae_doc_solicitud", "vias_sae_doc_acta", "vias_sae_doc_resolucion",
]);

const DATOS_SAE = Object.freeze([
  "vias_sae_plazas", "vias_sae_requisitos", "vias_sae_numero_oferta",
  "vias_sae_fecha_envio", "vias_sae_remitidos", "vias_sae_valoracion",
  "vias_sae_estado",
]);

function e(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#039;");
}

function valorCabecera(cabecera, clave) {
  const campo = cabecera.find((entrada) => entrada?.clave === clave);
  const valor = campo?.valor;
  return typeof valor === "string" && valor.trim() !== "" && valor.trim() !== "—"
    ? valor : "";
}

function filaDato(etiqueta, valor, t) {
  const registrado = valor !== "";
  return `<div class="ct-via-dato"><dt>${e(etiqueta)}</dt><dd>${registrado
    ? e(valor) : `<span class="ct-exp-chip ct-via-pendiente">${e(t("vias_estado_no_consta"))}</span>`}</dd></div>`;
}

function datosComunes(cabecera, t) {
  return DATOS_COMUNES.map(([clave, etiqueta]) => filaDato(t(etiqueta), valorCabecera(cabecera, clave), t)).join("");
}

function documentos(claves, t) {
  return `<ul class="ct-via-documentos">${claves.map((clave) => `<li>${e(t(clave))}</li>`).join("")}</ul>`;
}

function panelVia({ clave, titulo, mensajeEstado, ayuda, datosAdicionales, documentosClaves, cabecera, puedeVerDocumentos, t }) {
  return `<details class="panel ct-via-panel" data-ct-via="${clave}">
    <summary class="cabecera-panel ct-via-resumen"><span>${e(t(titulo))}</span><span class="ct-via-indicador" aria-hidden="true">⌄</span></summary>
    <div class="cuerpo-panel ct-via-cuerpo">
      <div class="ct-via-encabezado"><p class="ct-via-estado" role="status">${e(t(mensajeEstado))}</p>
        <details class="ct-via-ayuda"><summary aria-label="${e(t("vias_ayuda_aria"))}">?</summary><p>${e(t(ayuda))}</p></details></div>
      <section class="ct-via-bloque" aria-labelledby="ct-via-${clave}-datos">
        <h4 id="ct-via-${clave}-datos">${e(t("vias_datos"))}</h4>
        <dl class="ct-via-datos">${datosComunes(cabecera, t)}${datosAdicionales}</dl>
      </section>
      <section class="ct-via-bloque" aria-labelledby="ct-via-${clave}-documentos">
        <h4 id="ct-via-${clave}-documentos">${e(t("vias_documentos"))}</h4>
        ${documentos(documentosClaves, t)}
        <p class="ct-via-nota">${e(t("vias_documentos_orientativos"))}</p>
        ${puedeVerDocumentos
    ? `<button type="button" class="boton-secundario" data-ct-exp-vista="documentos">${e(t("vias_documentos_abrir"))}</button>`
    : `<p class="ct-via-nota" role="status">${e(t("vias_documentos_denegado"))}</p>`}
      </section>
    </div>
  </details>`;
}

// Sólo se usa sobre un detalle real leído con autorización vigente. La entrada
// SAE es una preparación visible: ningún dato local constituye un alta en Bolsa.
export function renderizarViasExpediente(estado, t) {
  const expediente = estado?.expediente;
  if (estado?.vista !== "expediente" || estado?.carga !== "listo"
    || expediente?.demostracion !== false || !Array.isArray(expediente.cabecera)
    || typeof t !== "function") return "";

  const cabecera = expediente.cabecera;
  const puedeVerDocumentos = estado.navegacion?.documentos === true;
  const bolsaRef = valorCabecera(cabecera, "bolsa_cobertura");
  const datosBolsa = filaDato(t("cabecera_bolsa_cobertura"),
    bolsaRef ? t("vias_estado_registrado") : "", t);
  const datosSAE = DATOS_SAE.map((clave) => filaDato(t(clave), "", t)).join("");

  return `<section class="ct-vias-expediente" aria-labelledby="ct-vias-titulo">
    <header class="ct-vias-cabecera"><h3 id="ct-vias-titulo">${e(t("vias_titulo"))}</h3>
      <p>${e(t("vias_intro"))}</p></header>
    <div class="ct-vias-entradas">
      ${panelVia({ clave: "bolsa", titulo: "vias_bolsa", mensajeEstado: "vias_estado_bolsa_sin_accion",
    ayuda: "vias_ayuda_bolsa", datosAdicionales: datosBolsa, documentosClaves: DOCUMENTOS_BOLSA,
    cabecera, puedeVerDocumentos, t })}
      ${panelVia({ clave: "sae", titulo: "vias_sae", mensajeEstado: "vias_estado_sae_sin_tramite",
    ayuda: "vias_ayuda_sae", datosAdicionales: datosSAE, documentosClaves: DOCUMENTOS_SAE,
    cabecera, puedeVerDocumentos, t })}
    </div>
  </section>`;
}
