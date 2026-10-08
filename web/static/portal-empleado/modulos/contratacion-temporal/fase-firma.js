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
import { escaparHTML } from "./componentes-expedientes.js?v=20261008-ct-sin-bolsa-v1";

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

/** Último paso firmado cuyo PDF guarda Documentos, o null. */
function ultimoCustodiado(pasos) {
  for (let i = pasos.length - 1; i >= 0; i -= 1) {
    if (pasos[i]?.estado === "firmado" && pasos[i].documento_custodiado) return pasos[i];
  }
  return null;
}

/**
 * Estado de la fase de un documento. `documento` es la entrada del catálogo
 * y, si hay estado real, lleva `paso_pendiente` y el estado de cada paso con
 * `registrada_en`. Devuelve { estado, paso (el que debe firmar o null),
 * total, desde (ISO o ""), devueltoPor (paso que devolvió, solo si devuelto),
 * custodiado (paso cuyo PDF firmado guarda Documentos, o null) }.
 */
export function calcularFaseDocumento(documento, conEstadoReal) {
  const pasos = documento.pasos;
  const total = pasos.length;
  if (!conEstadoReal || !Number.isSafeInteger(documento.paso_pendiente)) {
    return Object.freeze({ estado: "no_disponible", paso: pasos[0] ?? null, total, desde: "", custodiado: null });
  }
  const custodiado = ultimoCustodiado(pasos);
  if (documento.paso_pendiente === 0) {
    return Object.freeze({ estado: "firmado", paso: null, total, desde: pasos.at(-1)?.registrada_en ?? "", custodiado });
  }
  const paso = pasos[documento.paso_pendiente - 1] ?? null;
  const devuelto = pasos.find((p) => p.estado === "devuelto");
  if (devuelto) return Object.freeze({ estado: "devuelto", paso, total, desde: devuelto.registrada_en ?? "", devueltoPor: devuelto, custodiado });
  if (documento.paso_pendiente === 1) return Object.freeze({ estado: "borrador", paso, total, desde: "", custodiado });
  return Object.freeze({
    estado: "pendiente_firma", paso, total, desde: pasos[documento.paso_pendiente - 2]?.registrada_en ?? "", custodiado,
  });
}

function textoDesde(fase, textos) {
  if (fase.estado === "borrador") return textos.traducir("fase.sin_fecha");
  if (!fechaValida(fase.desde)) return textos.traducir("fase.fecha_desconocida");
  return textos.fecha(fase.desde, { dateStyle: "medium", timeStyle: "short", timeZone: ZONA_HORARIA });
}

/** Rótulo y texto de la columna de personas: quién firma o quién devolvió. */
function persona(fase, textos, nombrar) {
  if (fase.devueltoPor) {
    const cargo = nombrar("cargo", fase.devueltoPor.cargo);
    const motivo = fase.devueltoPor.motivo_devolucion;
    return [textos.traducir("fase.devuelto_por"),
      motivo ? textos.traducir("fase.devuelto_motivo", { cargo, motivo }) : cargo];
  }
  const rotulo = textos.traducir("fase.quien");
  if (!fase.paso) return [rotulo, textos.traducir("fase.nadie")];
  const cargo = nombrar("cargo", fase.paso.cargo);
  return [rotulo, fase.total > 1 ? textos.traducir("fase.paso", { cargo, actual: fase.paso.orden, total: fase.total }) : cargo];
}

/**
 * Botón para descargar el PDF firmado que guarda Documentos. Los datos del
 * botón son los que la descarga liga en el servidor: expediente, documento,
 * versión y huella esperada.
 */
function botonFirmado(fase, etiqueta, textos) {
  const c = fase.custodiado?.documento_custodiado;
  if (!c) return "";
  const pasoTexto = fase.total > 1 ? ` ${textos.traducir("fase.firmado_paso", { actual: fase.custodiado.orden, total: fase.total })}` : "";
  return `<div class="ct-fase-firma-firmado">
        <button type="button" class="boton-secundario" data-ct-descargar-firmado
          data-ct-firmado-expediente="${escaparHTML(c.expediente_ref)}" data-ct-firmado-documento="${escaparHTML(c.documento_ref)}"
          data-ct-firmado-version="${c.version}" data-ct-firmado-huella="${escaparHTML(c.huella_sha256)}"
          aria-label="${escaparHTML(textos.traducir("fase.descargar_firmado_de", { documento: etiqueta }))}">${escaparHTML(textos.traducir("fase.descargar_firmado"))}</button>
        <span class="ct-fase-firma-rotulo">${escaparHTML(textos.traducir("fase.firmado_prueba"))}${escaparHTML(pasoTexto)}</span>
      </div>`;
}

/**
 * HTML del apartado. `catalogo` es el circuito del catálogo (sin estado),
 * `real` el circuito fusionado con la historia o null. `nombrar(tipo, valor)`
 * traduce documento y cargo del catálogo. `aviso` false omite el aviso de
 * estado no disponible cuando el bloque ya explica otro motivo (permiso).
 */
export function renderizarFaseFirma({ catalogo, real, textos, nombrar = (_tipo, valor) => valor, aviso = true }) {
  if (!textos || !catalogo?.documentos?.length) return "";
  const conEstado = Boolean(real);
  const documentos = conEstado ? real.documentos : catalogo.documentos;
  const filas = documentos.map((documento) => {
    const fase = calcularFaseDocumento(documento, conEstado);
    const tono = TONO[fase.estado];
    const [rotuloPersona, textoPersona] = persona(fase, textos, nombrar);
    const etiqueta = nombrar("documento", documento.etiqueta);
    return `<li class="ct-fase-firma-fila ct-fase-firma-fila--${tono}" data-ct-fase-firma-documento="${escaparHTML(documento.documento)}"
      data-ct-fase-firma-estado="${fase.estado}">
      <div class="ct-fase-firma-dato ct-fase-firma-documento"><span class="ct-fase-firma-rotulo">${escaparHTML(textos.traducir("fase.documento"))}</span>
        <strong>${escaparHTML(etiqueta)}</strong></div>
      <div class="ct-fase-firma-dato"><span class="ct-fase-firma-rotulo">${escaparHTML(textos.traducir("fase.estado"))}</span>
        <span class="ct-fase-firma-estado ct-tono-${tono}">${escaparHTML(textos.traducir(`estado.${fase.estado}`))}</span></div>
      <div class="ct-fase-firma-dato"><span class="ct-fase-firma-rotulo">${escaparHTML(rotuloPersona)}</span>
        <span>${escaparHTML(textoPersona)}</span></div>
      <div class="ct-fase-firma-dato"><span class="ct-fase-firma-rotulo">${escaparHTML(textos.traducir("fase.desde"))}</span>
        <span>${escaparHTML(textoDesde(fase, textos))}</span></div>
      ${botonFirmado(fase, etiqueta, textos)}
    </li>`;
  }).join("");
  return `<section class="ct-fase-firma" aria-labelledby="ct-fase-firma-titulo" data-ct-fase-firma>
    <h4 id="ct-fase-firma-titulo">${escaparHTML(textos.traducir("fase.titulo"))}</h4>
    ${conEstado || !aviso ? "" : `<p class="ct-fase-firma-aviso" role="status">${escaparHTML(textos.traducir("fase.estado_no_disponible_aviso"))}</p>`}
    <ul class="ct-fase-firma-lista" aria-label="${escaparHTML(textos.traducir("fase.lista"))}">${filas}</ul>
  </section>`;
}
