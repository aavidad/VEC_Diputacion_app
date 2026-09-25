/**
 * Circuito de firma de los borradores del expediente. Consulta el catálogo
 * vigente y, si el registro de firmas está compuesto, el estado real de cada
 * paso. En el paso pendiente ofrece «Firmar» (AutoFirma) y «Devolver». No
 * decide estados: los recibe del servidor. Una firma de prueba registrada no
 * tiene eficacia administrativa hasta el portafirmas corporativo.
 */

import { escaparHTML, solicitudInformeDefinitivoDesdeEstado } from "./componentes-expedientes.js";
import { crearAccionesFirma, fusionarEstadoFirmas, renderizarAccionesPaso } from "./circuito-firma-acciones.js";
import { crearClienteFirmaDocumento } from "./firma-documento-cliente.js";
import { crearTraductorCircuitoFirma } from "./i18n-circuito-firma.js";

export const RUTA_CIRCUITO_FIRMA = "/api/vec/contratacion-temporal/circuito-firma";
const ESQUEMA = "vec.contratacion_temporal.circuito_firma.v1";
const MAXIMO_RESPUESTA = 64 * 1024;
const MAXIMO_DOCUMENTOS = 32;
const MAXIMO_PASOS = 16;
const CLAVE = /^[a-z][a-z0-9._-]{0,127}$/u;
const PERFIL = /^[a-z][a-z0-9._:-]{2,127}$/u;
const HUELLA = /^[0-9a-f]{64}$/u;
const VOCABULARIO = Object.freeze({
  accion: ["firma", "visto_bueno"],
  condicion: ["borrador_generado", "firma_paso_anterior"],
  habilita: ["siguiente_paso", "remision_intervencion", "envio_notificacion", "envio_comunicacion", "cierre_circuito"],
  devolucion: ["vuelve_a_redaccion", "vuelve_paso_anterior"],
  sustitucion: ["suplente_designado", "no_admitida"],
  estado: ["pendiente_firma", "en_espera", "firmado", "devuelto"],
});
const TONO_ESTADO = Object.freeze({
  pendiente_firma: "aviso", en_espera: "neutro", firmado: "exito", devuelto: "peligro",
});
const CAMPOS_CIRCUITO = ["esquema", "catalogo_ref", "huella_sha256", "ejemplo", "firma_eficaz", "documentos"];
const CAMPOS_DOCUMENTO = ["documento", "etiqueta", "pasos"];
const CAMPOS_PASO = ["orden", "cargo", "perfil_ref", "accion", "condicion", "habilita", "devolucion", "sustitucion", "estado", "referencia"];

function camposExactos(valor, campos) {
  return valor !== null && typeof valor === "object" && !Array.isArray(valor)
    && Object.getPrototypeOf(valor) === Object.prototype
    && Object.keys(valor).length === campos.length && campos.every((campo) => Object.hasOwn(valor, campo));
}

function textoAcotado(valor, maximo = 512) {
  return typeof valor === "string" && valor.trim() !== "" && valor.length <= maximo;
}

function validarPaso(paso, indice, total) {
  if (!camposExactos(paso, CAMPOS_PASO) || paso.orden !== indice + 1
    || !textoAcotado(paso.cargo) || !PERFIL.test(paso.perfil_ref) || !textoAcotado(paso.referencia)
    || Object.entries(VOCABULARIO).some(([campo, valores]) => !valores.includes(paso[campo]))
    || (indice === total - 1) === (paso.habilita === "siguiente_paso")) return null;
  return Object.freeze({ ...paso });
}

/** Valida la respuesta completa; cualquier desviación la invalida entera. */
export function validarCircuitoFirma(datos) {
  if (!camposExactos(datos, CAMPOS_CIRCUITO) || datos.esquema !== ESQUEMA
    || !textoAcotado(datos.catalogo_ref) || !HUELLA.test(datos.huella_sha256)
    || typeof datos.ejemplo !== "boolean" || datos.firma_eficaz !== false
    || !Array.isArray(datos.documentos) || datos.documentos.length < 1
    || datos.documentos.length > MAXIMO_DOCUMENTOS) return null;
  const documentos = [];
  for (const documento of datos.documentos) {
    if (!camposExactos(documento, CAMPOS_DOCUMENTO) || !CLAVE.test(documento.documento)
      || !textoAcotado(documento.etiqueta) || !Array.isArray(documento.pasos)
      || documento.pasos.length < 1 || documento.pasos.length > MAXIMO_PASOS) return null;
    const pasos = documento.pasos.map((paso, indice) => validarPaso(paso, indice, documento.pasos.length));
    if (pasos.includes(null)) return null;
    documentos.push(Object.freeze({ documento: documento.documento, etiqueta: documento.etiqueta, pasos: Object.freeze(pasos) }));
  }
  return Object.freeze({
    catalogo_ref: datos.catalogo_ref, huella_sha256: datos.huella_sha256,
    ejemplo: datos.ejemplo, documentos: Object.freeze(documentos),
  });
}

async function leerAcotado(respuesta) {
  const longitud = respuesta.headers.get("Content-Length");
  if (longitud !== null && (!/^(0|[1-9][0-9]*)$/u.test(longitud) || Number(longitud) > MAXIMO_RESPUESTA)) {
    throw new TypeError("respuesta demasiado grande");
  }
  const lector = respuesta.body?.getReader();
  if (!lector) throw new TypeError("respuesta sin cuerpo");
  const fragmentos = [];
  let total = 0;
  try {
    for (;;) {
      const { value, done } = await lector.read();
      if (done) break;
      total += value.byteLength;
      if (total > MAXIMO_RESPUESTA) throw new TypeError("respuesta demasiado grande");
      fragmentos.push(value);
    }
  } finally {
    void lector.cancel().catch(() => {});
  }
  const bytes = new Uint8Array(total);
  let posicion = 0;
  for (const fragmento of fragmentos) {
    bytes.set(fragmento, posicion);
    posicion += fragmento.byteLength;
  }
  return new TextDecoder("utf-8", { fatal: true }).decode(bytes);
}

/** Cliente de solo lectura. Devuelve null si el circuito no está disponible. */
export function crearClienteHTTPCircuitoFirma({ fetchImpl = globalThis.fetch } = {}) {
  return Object.freeze({
    async obtenerCircuito({ signal } = {}) {
      if (typeof fetchImpl !== "function") return null;
      try {
        const respuesta = await fetchImpl(RUTA_CIRCUITO_FIRMA, {
          method: "GET", headers: { Accept: "application/json" }, signal,
          mode: "same-origin", credentials: "same-origin", cache: "no-store",
          redirect: "error", referrerPolicy: "no-referrer",
        });
        if (respuesta.redirected || respuesta.status !== 200
          || !/^application\/json(?:;\s*charset=utf-8)?$/iu.test(respuesta.headers.get("Content-Type") ?? "")) {
          void respuesta.body?.cancel?.().catch(() => {});
          return null;
        }
        const envoltorio = JSON.parse(await leerAcotado(respuesta));
        return camposExactos(envoltorio, ["data"]) ? validarCircuitoFirma(envoltorio.data) : null;
      } catch {
        return null;
      }
    },
  });
}

function renderizarPaso(paso, t, circuito, documento) {
  const tono = TONO_ESTADO[paso.estado];
  const detalles = [
    t(`circuito_firma_accion_${paso.accion}`),
    t(`circuito_firma_habilita_${paso.habilita}`),
    t(`circuito_firma_devolucion_${paso.devolucion}`),
    t(`circuito_firma_sustitucion_${paso.sustitucion}`),
  ];
  return `<li class="ct-circuito-paso ct-circuito-paso--${tono}"${paso.estado === "pendiente_firma" ? ' aria-current="step"' : ""}>
    <span class="ct-circuito-orden" aria-hidden="true">${paso.orden}</span>
    <div class="ct-circuito-paso-cuerpo">
      <p class="ct-circuito-cargo">${escaparHTML(paso.cargo)}</p>
      <p class="ct-circuito-detalle">${detalles.map(escaparHTML).join(" · ")}</p>
      ${paso.estado === "devuelto" && paso.motivo_devolucion ? `<p class="ct-circuito-motivo">${escaparHTML(t("circuito_firma_motivo", { motivo: paso.motivo_devolucion }))}</p>` : ""}
      ${renderizarAccionesPaso(circuito, documento, paso, t)}
    </div>
    <span class="ct-circuito-estado ct-tono-${tono}">${escaparHTML(t(`circuito_firma_estado_${paso.estado}`, { cargo: paso.cargo }))}</span>
  </li>`;
}

/** HTML del bloque; los textos visibles salen del traductor o del catálogo. */
export function renderizarCircuitoFirma(circuito, t) {
  return `<section class="ct-circuito-firma" aria-labelledby="ct-circuito-firma-titulo" data-ct-circuito-firma>
    <header class="ct-circuito-cabecera">
      <h3 id="ct-circuito-firma-titulo">${escaparHTML(t("circuito_firma_titulo"))}</h3>
      ${circuito.ejemplo ? `<span class="ct-circuito-marca">${escaparHTML(t("circuito_firma_ejemplo"))}</span>` : ""}
      ${circuito.registro ? `<span class="ct-circuito-marca">${escaparHTML(t("circuito_firma_sin_eficacia"))}</span>` : ""}
    </header>
    <p class="ct-circuito-aviso" role="status" aria-live="polite" data-ct-firma-aviso></p>
    <div class="ct-circuito-documentos">${circuito.documentos.map((documento) => `<article class="ct-circuito-documento" data-ct-circuito-documento="${escaparHTML(documento.documento)}">
      <h4>${escaparHTML(documento.etiqueta)}</h4>
      <ol aria-label="${escaparHTML(t("circuito_firma_pasos", { documento: documento.etiqueta }))}">${documento.pasos.map((paso) => renderizarPaso(paso, t, circuito, documento)).join("")}</ol>
    </article>`).join("")}</div>
  </section>`;
}

/**
 * Monta el bloque tras la cabecera del expediente. La consulta se hace una
 * vez por montaje; si falla o no hay catálogo, el bloque no aparece.
 */
export function crearGestorCircuitoFirma({
  raiz, obtenerEstado, cliente = crearClienteHTTPCircuitoFirma(), mensajes = {}, esMontada = () => true,
  clienteFirma = crearClienteFirmaDocumento(), dependenciasAcciones = {},
} = {}) {
  if (typeof obtenerEstado !== "function") throw new TypeError("estado del circuito de firma no disponible");
  const t = crearTraductorCircuitoFirma(mensajes);
  const controlador = new AbortController();
  let consulta = null;

  // El estado real solo se consulta con un expediente cuyos borradores
  // existen; sin registro compuesto el bloque sigue siendo informativo.
  async function conEstadoReal(circuito) {
    const solicitud = circuito ? solicitudInformeDefinitivoDesdeEstado(obtenerEstado()) : null;
    if (!solicitud || typeof clienteFirma?.consultar !== "function") return circuito;
    const estado = await clienteFirma.consultar(solicitud.expediente_ref, { signal: controlador.signal }).catch(() => null);
    return fusionarEstadoFirmas(circuito, estado) ?? circuito;
  }

  const acciones = crearAccionesFirma({
    obtenerEstado, t, clienteFirma, ...dependenciasAcciones,
    async alCambiar(aviso) {
      const circuito = await consulta;
      const nuevo = await conEstadoReal(circuito);
      const actual = raiz.querySelector?.("[data-ct-circuito-firma]");
      if (!nuevo || !actual || !esMontada()) return;
      actual.outerHTML = renderizarCircuitoFirma(nuevo, t);
      const repintado = raiz.querySelector?.("[data-ct-circuito-firma]");
      repintado?.addEventListener?.("click", manejar);
      const aviso_ = repintado?.querySelector?.("[data-ct-firma-aviso]");
      if (aviso_) aviso_.textContent = aviso;
    },
  });
  function manejar(evento) { void acciones.manejarClic(evento); }

  function insertar(circuito) {
    if (!circuito || !esMontada() || raiz.querySelector?.("[data-ct-circuito-firma]")) return;
    const cabecera = raiz.querySelector?.(".ct-exp-cabecera-expediente");
    if (typeof cabecera?.insertAdjacentHTML !== "function") return;
    cabecera.insertAdjacentHTML("afterend", renderizarCircuitoFirma(circuito, t));
    raiz.querySelector?.("[data-ct-circuito-firma]")?.addEventListener?.("click", manejar);
  }

  function montarSiProcede(estado) {
    if (estado?.vista !== "expediente" || !estado.expediente || typeof cliente?.obtenerCircuito !== "function") return;
    consulta ??= Promise.resolve(cliente.obtenerCircuito({ signal: controlador.signal })).catch(() => null);
    const expedienteRef = estado.expediente.expediente_ref;
    void consulta.then(conEstadoReal).then((circuito) => {
      const actual = obtenerEstado();
      if (actual?.vista === "expediente" && actual.expediente?.expediente_ref === expedienteRef) insertar(circuito);
    });
  }

  return Object.freeze({
    montarSiProcede,
    retirar() { controlador.abort(); },
  });
}
