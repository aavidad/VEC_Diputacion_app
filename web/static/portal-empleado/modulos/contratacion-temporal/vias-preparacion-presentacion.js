/** Relación informativa de la publicación V2; no inicia trámites ni decide cobertura. */
const CLAVES_ESTRUCTURA = Object.freeze([
  "cobertura_preparacion_titulo", "cobertura_preparacion_bolsa",
  "cobertura_preparacion_sae", "cobertura_preparacion_documentos",
  "cobertura_preparacion_datos", "cobertura_preparacion_ejemplo",
  "cobertura_preparacion_no_disponible", "cobertura_preparacion_sae_solo_lista",
  "cobertura_preparacion_sin_elementos",
]);

function escapar(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#039;");
}

export function traduccionesPreparacionDisponibles(propuesta, t) {
  try {
    for (const clave of CLAVES_ESTRUCTURA) t(clave);
    for (const via of propuesta?.catalogo?.vias ?? []) {
      for (const elemento of [...via.documentos, ...via.datos]) t(elemento.clave_i18n);
    }
    return true;
  } catch {
    return false;
  }
}

function listaElementos(elementos, t) {
  return elementos.length === 0
    ? `<p class="ct-exp-vacio">${escapar(t("cobertura_preparacion_sin_elementos"))}</p>`
    : `<ul>${elementos.map(({ clave_i18n }) => `<li>${escapar(t(clave_i18n))}</li>`).join("")}</ul>`;
}

function viaPresentada(via, clave, t) {
  const sae = clave === "oferta_sae";
  const titulo = sae ? "cobertura_preparacion_sae" : "cobertura_preparacion_bolsa";
  return `<details class="panel ct-bloque ct-via-preparacion" data-ct-preparacion-via="${sae ? "oferta_sae" : "bolsa_vigente"}"${sae ? "" : " open"}>
    <summary>${escapar(t(titulo))}</summary>
    <div class="ct-exp-paneles">
      <section><h4>${escapar(t("cobertura_preparacion_documentos"))}</h4>${listaElementos(via.documentos, t)}</section>
      <section><h4>${escapar(t("cobertura_preparacion_datos"))}</h4>${listaElementos(via.datos, t)}</section>
    </div>
    ${sae ? `<p>${escapar(t("cobertura_preparacion_sae_solo_lista"))}</p>` : ""}
  </details>`;
}

export function renderizarViasPreparacion(propuesta, t) {
  const catalogo = propuesta?.catalogo;
  const bolsa = catalogo?.vias.find(({ clave }) => clave === "bolsa_vigente");
  const sae = catalogo?.vias.find(({ clave }) => clave === "oferta_sae");
  if (!bolsa || !sae || !traduccionesPreparacionDisponibles(propuesta, t)) {
    return `<section class="panel ct-bloque" role="status" data-ct-preparacion-no-disponible>
      <p>${escapar(t("cobertura_preparacion_no_disponible"))}</p></section>`;
  }
  return `<section class="ct-preparacion-vias" aria-labelledby="ct-preparacion-vias-titulo" data-ct-preparacion-vias>
    <h3 id="ct-preparacion-vias-titulo">${escapar(t("cobertura_preparacion_titulo"))}</h3>
    ${catalogo?.es_ejemplo ? `<p role="status">${escapar(t("cobertura_preparacion_ejemplo"))}</p>` : ""}
    <div class="ct-exp-paneles">${viaPresentada(bolsa, "bolsa_vigente", t)}${viaPresentada(sae, "oferta_sae", t)}</div>
  </section>`;
}
