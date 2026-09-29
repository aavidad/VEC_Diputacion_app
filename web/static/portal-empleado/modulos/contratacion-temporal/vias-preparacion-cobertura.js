/**
 * Relación de documentos y datos que pide cada vía de cobertura, en dos
 * pestañas: «Por bolsa de trabajo» y «Por oferta al SAE».
 *
 * Es informativa: no inicia trámites, no decide la cobertura y no guarda ni
 * envía nada. La vía SAE solo enumera lo que se pediría (duda 73). El
 * catálogo llega ya validado (`validarCatalogoPreparacion`) desde los
 * catálogos del alta o desde la propuesta de cobertura del expediente, y los
 * textos, incluidos los nombres de cada documento y dato por su `clave_i18n`,
 * salen del catálogo de textos `contratacion-temporal-cobertura`.
 */
import { cargarTextos } from "../../../comun/textos.js";

const textos = await cargarTextos("contratacion-temporal-cobertura");

export const VIAS_PREPARACION = Object.freeze(["bolsa_vigente", "oferta_sae"]);
const PREFIJO_CLAVES_ELEMENTO = "contratacion_temporal.cobertura.";
const PATRON_PREFIJO = /^[a-z][a-z0-9-]{0,39}$/u;

function escapar(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#039;");
}

/** Texto de la pantalla de preparación (sección `preparacion` del catálogo). */
export function textoPreparacion(clave) {
  return textos.traducir(`preparacion.${clave}`);
}

function nombreElemento(claveI18n) {
  if (typeof claveI18n !== "string" || !claveI18n.startsWith(PREFIJO_CLAVES_ELEMENTO)) {
    throw new TypeError("clave de texto de preparación no admitida");
  }
  return textos.traducir(claveI18n);
}

function viasPresentables(catalogo) {
  const vias = VIAS_PREPARACION.map((clave) => catalogo?.vias?.find((via) => via.clave === clave));
  return vias.every(Boolean) ? vias : null;
}

/**
 * Cierto si el catálogo trae las dos vías y hay texto publicado para cada
 * documento y dato. Una clave sin texto no se muestra como código: la
 * relación entera pasa a «no disponible».
 */
export function preparacionPresentable(catalogo) {
  const vias = viasPresentables(catalogo);
  if (!vias) return false;
  try {
    for (const via of vias) {
      for (const { clave_i18n: claveI18n } of [...via.documentos, ...via.datos]) nombreElemento(claveI18n);
    }
    return true;
  } catch {
    return false;
  }
}

function listaElementos(elementos) {
  return elementos.length === 0
    ? `<p class="ct-preparacion-vacio">${escapar(textoPreparacion("sin_elementos"))}</p>`
    : `<ul class="ct-preparacion-lista">${elementos.map(({ clave_i18n: claveI18n }) =>
      `<li>${escapar(nombreElemento(claveI18n))}</li>`).join("")}</ul>`;
}

function panelVia(via, prefijo, seleccionada) {
  const sae = via.clave === "oferta_sae";
  return `<div class="ct-preparacion-panel" role="tabpanel" id="${prefijo}-panel-${via.clave}"
      aria-labelledby="${prefijo}-pestana-${via.clave}" tabindex="0"
      data-ct-preparacion-via="${via.clave}"${seleccionada ? "" : " hidden"}>
    ${sae ? `<p class="ct-preparacion-nota" role="note">${escapar(textoPreparacion("sae_solo_informativa"))}</p>` : ""}
    <div class="ct-exp-paneles">
      <section aria-labelledby="${prefijo}-documentos-${via.clave}">
        <h4 id="${prefijo}-documentos-${via.clave}">${escapar(textoPreparacion("documentos"))}</h4>
        ${listaElementos(via.documentos)}
      </section>
      <section aria-labelledby="${prefijo}-datos-${via.clave}">
        <h4 id="${prefijo}-datos-${via.clave}">${escapar(textoPreparacion("datos"))}</h4>
        ${listaElementos(via.datos)}
      </section>
    </div>
  </div>`;
}

/**
 * HTML de la relación en dos pestañas. `prefijo` distingue los identificadores
 * de cada pantalla; `seleccionada` es la vía cuya pestaña está abierta.
 */
export function renderizarViasPreparacion(catalogo, { prefijo = "ct-preparacion", seleccionada } = {}) {
  if (!PATRON_PREFIJO.test(prefijo)) throw new TypeError("prefijo de preparación no válido");
  if (!preparacionPresentable(catalogo)) {
    return `<section class="panel ct-bloque ct-preparacion-vias" role="status" data-ct-preparacion-no-disponible>
      <p>${escapar(textoPreparacion("no_disponible"))}</p></section>`;
  }
  const abierta = VIAS_PREPARACION.includes(seleccionada) ? seleccionada : VIAS_PREPARACION[0];
  const vias = viasPresentables(catalogo);
  const pestanas = vias.map((via) => {
    const activa = via.clave === abierta;
    return `<button type="button" role="tab" id="${prefijo}-pestana-${via.clave}"
      aria-controls="${prefijo}-panel-${via.clave}" aria-selected="${activa}" tabindex="${activa ? "0" : "-1"}"
      data-ct-preparacion-pestana="${via.clave}">${escapar(textoPreparacion(via.clave === "oferta_sae" ? "via_sae" : "via_bolsa"))}</button>`;
  }).join("");
  return `<section class="panel ct-bloque ct-preparacion-vias" aria-labelledby="${prefijo}-titulo" data-ct-preparacion-vias>
    <header class="ct-preparacion-cabecera"><h3 id="${prefijo}-titulo">${escapar(textoPreparacion("titulo"))}</h3></header>
    ${catalogo.es_ejemplo ? `<p class="ct-preparacion-ejemplo" role="note">${escapar(textoPreparacion("ejemplo"))}</p>` : ""}
    <nav class="acciones-vista ct-preparacion-pestanas" role="tablist" aria-label="${escapar(textoPreparacion("pestanas"))}">${pestanas}</nav>
    ${vias.map((via) => panelVia(via, prefijo, via.clave === abierta)).join("")}
  </section>`;
}

/**
 * Vía que pide abrir un evento sobre las pestañas: un clic en una pestaña o,
 * con el foco en ella, flechas izquierda/derecha, Inicio y Fin. Devuelve null
 * si el evento no es de las pestañas; quien llama repinta y enfoca la pestaña.
 */
export function viaPreparacionDeEvento(evento) {
  const pestana = evento?.target?.closest?.("[data-ct-preparacion-pestana]");
  const actual = pestana?.dataset?.ctPreparacionPestana;
  if (!VIAS_PREPARACION.includes(actual)) return null;
  if (evento.type === "click") return actual;
  if (evento.type !== "keydown") return null;
  const indice = VIAS_PREPARACION.indexOf(actual);
  const total = VIAS_PREPARACION.length;
  const destino = {
    ArrowRight: (indice + 1) % total,
    ArrowLeft: (indice - 1 + total) % total,
    Home: 0,
    End: total - 1,
  }[evento.key];
  return destino === undefined ? null : VIAS_PREPARACION[destino];
}

/** Selector de la pestaña de una vía, para devolverle el foco tras repintar. */
export function selectorPestanaPreparacion(via) {
  return VIAS_PREPARACION.includes(via) ? `[data-ct-preparacion-pestana="${via}"]` : "";
}

/**
 * Monta la relación en `zona` y atiende sus pestañas (clic y teclado),
 * conservando la vía abierta entre repintados. Devuelve la función que
 * retira las escuchas y vacía la zona. Es solo informativa: no condiciona
 * ningún formulario.
 */
export function montarPestanasPreparacion(zona, catalogo, prefijo) {
  let seleccionada;
  const pintar = (selectorFoco = "") => {
    zona.innerHTML = renderizarViasPreparacion(catalogo, { prefijo, seleccionada });
    if (selectorFoco) zona.querySelector(selectorFoco)?.focus?.();
  };
  const alEvento = (evento) => {
    const via = viaPreparacionDeEvento(evento);
    if (via === null) return;
    evento.preventDefault?.();
    seleccionada = via;
    pintar(selectorPestanaPreparacion(via));
  };
  pintar();
  zona.addEventListener("click", alEvento);
  zona.addEventListener("keydown", alEvento);
  return () => {
    zona.removeEventListener("click", alEvento);
    zona.removeEventListener("keydown", alEvento);
    zona.replaceChildren();
  };
}
