/**
 * Fase de firma del expediente, siempre visible en la ficha: para cada
 * documento del circuito dice en qué estado está, quién debe firmar ahora y
 * desde cuándo. Los documentos, los pasos y quién firma vienen del catálogo
 * del circuito; el estado sale de la historia registrada. Si esa historia no
 * responde, se muestra el primer firmante del catálogo y el estado se declara
 * no disponible: nunca se deduce un estado del ejemplo.
 *
 * Firmadoc no está conectado: ningún documento puede estar «enviado a
 * Firmadoc» y ninguna firma de VEC cuenta como firma oficial. El estado
 * `enviado_portafirmas` existe en el catálogo de textos para cuando lo esté.
 * Los textos visibles salen del catálogo `contratacion-temporal-firma`.
 */

import { cargarTextos } from "../../../comun/textos.js";
import { escaparHTML } from "./componentes-expedientes.js";

export const MODULO_TEXTOS_FASE_FIRMA = "contratacion-temporal-firma";
const ZONA_HORARIA = "Europe/Madrid";

const TONO = Object.freeze({
  borrador: "neutro",
  pendiente_firma: "aviso",
  enviado_portafirmas: "info",
  firmado: "exito",
  devuelto: "peligro",
  no_disponible: "neutro",
});

/** Carga los textos de la fase; si el catálogo falla devuelve null. */
export function cargarTextosFaseFirma(cargar = cargarTextos) {
  return Promise.resolve().then(() => cargar(MODULO_TEXTOS_FASE_FIRMA)).catch(() => null);
}

function fechaValida(valor) {
  return typeof valor === "string" && valor !== "" && !Number.isNaN(Date.parse(valor));
}

/**
 * Estado de la fase de un documento. `documento` es la entrada del catálogo
 * y, si hay estado real, lleva `paso_pendiente` y el estado de cada paso con
 * `registrada_en`. Devuelve { estado, paso (el que debe firmar o null),
 * total, desde (ISO o "") }.
 */
export function calcularFaseDocumento(documento, conEstadoReal) {
  const pasos = documento.pasos;
  const total = pasos.length;
  if (!conEstadoReal || !Number.isSafeInteger(documento.paso_pendiente)) {
    return Object.freeze({ estado: "no_disponible", paso: pasos[0] ?? null, total, desde: "" });
  }
  if (documento.paso_pendiente === 0) {
    return Object.freeze({ estado: "firmado", paso: null, total, desde: pasos.at(-1)?.registrada_en ?? "" });
  }
  const paso = pasos[documento.paso_pendiente - 1] ?? null;
  const devuelto = pasos.find((p) => p.estado === "devuelto");
  if (devuelto) return Object.freeze({ estado: "devuelto", paso, total, desde: devuelto.registrada_en ?? "" });
  if (documento.paso_pendiente === 1) return Object.freeze({ estado: "borrador", paso, total, desde: "" });
  return Object.freeze({
    estado: "pendiente_firma", paso, total, desde: pasos[documento.paso_pendiente - 2]?.registrada_en ?? "",
  });
}

function textoDesde(fase, textos) {
  if (fase.estado === "borrador") return textos.traducir("fase.sin_fecha");
  if (!fechaValida(fase.desde)) return textos.traducir("fase.fecha_desconocida");
  return textos.fecha(fase.desde, { dateStyle: "medium", timeStyle: "short", timeZone: ZONA_HORARIA });
}

/**
 * HTML del apartado. `catalogo` es el circuito del catálogo (sin estado),
 * `real` el circuito fusionado con la historia o null. `nombrar(tipo, valor)`
 * traduce documento y cargo del catálogo.
 */
export function renderizarFaseFirma({ catalogo, real, textos, nombrar = (_tipo, valor) => valor }) {
  if (!textos || !catalogo?.documentos?.length) return "";
  const conEstado = Boolean(real);
  const documentos = conEstado ? real.documentos : catalogo.documentos;
  const filas = documentos.map((documento) => {
    const fase = calcularFaseDocumento(documento, conEstado);
    const tono = TONO[fase.estado];
    const cargo = fase.paso ? nombrar("cargo", fase.paso.cargo) : "";
    const quien = !fase.paso ? textos.traducir("fase.nadie")
      : fase.total > 1 ? `${cargo} (${textos.traducir("fase.paso", { actual: fase.paso.orden, total: fase.total })})` : cargo;
    return `<li class="ct-fase-firma-fila ct-fase-firma-fila--${tono}" data-ct-fase-firma-documento="${escaparHTML(documento.documento)}"
      data-ct-fase-firma-estado="${fase.estado}">
      <div class="ct-fase-firma-dato ct-fase-firma-documento"><span class="ct-fase-firma-rotulo">${escaparHTML(textos.traducir("fase.documento"))}</span>
        <strong>${escaparHTML(nombrar("documento", documento.etiqueta))}</strong></div>
      <div class="ct-fase-firma-dato"><span class="ct-fase-firma-rotulo">${escaparHTML(textos.traducir("fase.estado"))}</span>
        <span class="ct-fase-firma-estado ct-tono-${tono}">${escaparHTML(textos.traducir(`estado.${fase.estado}`))}</span></div>
      <div class="ct-fase-firma-dato"><span class="ct-fase-firma-rotulo">${escaparHTML(textos.traducir("fase.quien"))}</span>
        <span>${escaparHTML(quien)}</span></div>
      <div class="ct-fase-firma-dato"><span class="ct-fase-firma-rotulo">${escaparHTML(textos.traducir("fase.desde"))}</span>
        <span>${escaparHTML(textoDesde(fase, textos))}</span></div>
    </li>`;
  }).join("");
  return `<section class="ct-fase-firma" aria-labelledby="ct-fase-firma-titulo" data-ct-fase-firma>
    <h4 id="ct-fase-firma-titulo">${escaparHTML(textos.traducir("fase.titulo"))}</h4>
    <p class="ct-fase-firma-resumen">${escaparHTML(textos.traducir("fase.resumen"))}</p>
    ${conEstado ? "" : `<p class="ct-fase-firma-aviso" role="status">${escaparHTML(textos.traducir("fase.estado_no_disponible_aviso"))}</p>`}
    <ul class="ct-fase-firma-lista" aria-label="${escaparHTML(textos.traducir("fase.lista"))}">${filas}</ul>
  </section>`;
}
