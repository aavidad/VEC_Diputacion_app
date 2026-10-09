/**
 * Circuito de firma de los borradores del expediente. Consulta el catálogo
 * vigente y, si el registro de firmas está compuesto, el estado real de cada
 * paso. Mantiene visible la fase oficial aunque el catálogo o el registro no
 * respondan. Las acciones siguen cerradas hasta el preflight nominal R5.
 * Una firma de prueba registrada no acredita envío ni firma en Firmadoc.
 * Encima de todo va la fase de firma de cada documento (fase-firma.js).
 */

import { escaparHTML, solicitudInformeDefinitivoDesdeEstado } from "./componentes-expedientes.js?v=20261009-ct-bolsa-cohorte-v7";
import { crearAccionesFirma, fusionarEstadoFirmas, renderizarAccionesPaso } from "./circuito-firma-acciones.js?v=20261009-ct-bolsa-cohorte-v7";
import { crearClienteFirmaDocumento } from "./firma-documento-cliente.js?v=20260930-custodia-506-e3-v3";
import { cargarTextosFaseFirma, renderizarFaseFirma, renderizarFirmadosEnDocumentos } from "./fase-firma.js?v=20261009-ct-bolsa-cohorte-v7";
import { crearTraductorCircuitoFirma, traducirValorCircuitoFirma } from "./i18n-circuito-firma.js?v=20261007-pantallas-textos-final-v1";
import { crearFuenteDocumentosHTTP } from "../documentos/cliente-http.js?v=20261007-pantallas-textos-final-v1";

export const RUTA_CIRCUITO_FIRMA = "/api/vec/contratacion-temporal/circuito-firma";
const ESQUEMA_V1 = "vec.contratacion_temporal.circuito_firma.v1";
const ESQUEMA_V2 = "vec.contratacion_temporal.circuito_firma.v2";
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
const NO_CONECTADO = Object.freeze({ conectado: false, motivo: "conexion_pendiente" });
const MOTIVO_PORTAFIRMAS = /^[a-z][a-z0-9_]{2,63}$/u;
const CAMPOS_DOCUMENTO = ["documento", "etiqueta", "pasos"];
const CAMPOS_PASO = ["orden", "cargo", "perfil_ref", "accion", "condicion", "habilita", "devolucion", "sustitucion", "estado", "referencia"];
const CAMPO_ALTERNATIVAS = "perfiles_ref_alternativos";

function camposExactos(valor, campos) {
  return valor !== null && typeof valor === "object" && !Array.isArray(valor)
    && Object.getPrototypeOf(valor) === Object.prototype
    && Object.keys(valor).length === campos.length && campos.every((campo) => Object.hasOwn(valor, campo));
}

function textoAcotado(valor, maximo = 512) {
  return typeof valor === "string" && valor.trim() !== "" && valor.length <= maximo;
}

function validarPaso(paso, indice, total, esquema) {
  const conAlternativas = paso !== null && typeof paso === "object" && Object.hasOwn(paso, CAMPO_ALTERNATIVAS);
  if (!camposExactos(paso, conAlternativas && esquema === ESQUEMA_V2 ? [...CAMPOS_PASO, CAMPO_ALTERNATIVAS] : CAMPOS_PASO)
    || paso.orden !== indice + 1
    || !textoAcotado(paso.cargo) || !PERFIL.test(paso.perfil_ref) || !textoAcotado(paso.referencia)
    || Object.entries(VOCABULARIO).some(([campo, valores]) => !valores.includes(paso[campo]))
    || (indice === total - 1) === (paso.habilita === "siguiente_paso")) return null;
  if (!conAlternativas) return Object.freeze({ ...paso });
  const alternativas = paso.perfiles_ref_alternativos;
  if (!Array.isArray(alternativas) || alternativas.length < 1 || alternativas.length > MAXIMO_PASOS
    || alternativas.some((ref) => typeof ref !== "string" || !PERFIL.test(ref))
    || new Set([paso.perfil_ref, ...alternativas]).size !== alternativas.length + 1) return null;
  return Object.freeze({ ...paso, perfiles_ref_alternativos: Object.freeze([...alternativas]) });
}

/**
 * Estado de Firmadoc que da el servidor: conectado o no, con motivo cerrado.
 * Nunca trae envíos ni firmas; con cualquier otra forma no es válido.
 */
function validarPortafirmas(valor) {
  if (!camposExactos(valor, valor?.conectado === false ? ["conectado", "motivo"] : ["conectado"])
    || typeof valor.conectado !== "boolean"
    || (!valor.conectado && !MOTIVO_PORTAFIRMAS.test(valor.motivo))) return null;
  return Object.freeze({ ...valor });
}

/**
 * Valida la respuesta completa; cualquier desviación la invalida entera.
 * `portafirmas` es opcional: si falta, Firmadoc cuenta como no conectado.
 */
export function validarCircuitoFirma(datos) {
  const conPortafirmas = datos !== null && typeof datos === "object" && Object.hasOwn(datos, "portafirmas");
  if (!camposExactos(datos, conPortafirmas ? [...CAMPOS_CIRCUITO, "portafirmas"] : CAMPOS_CIRCUITO)
    || (datos.esquema !== ESQUEMA_V1 && datos.esquema !== ESQUEMA_V2)
    || !textoAcotado(datos.catalogo_ref) || !HUELLA.test(datos.huella_sha256)
    || typeof datos.ejemplo !== "boolean" || datos.firma_eficaz !== false
    || (conPortafirmas && !validarPortafirmas(datos.portafirmas))
    || !Array.isArray(datos.documentos) || datos.documentos.length < 1
    || datos.documentos.length > MAXIMO_DOCUMENTOS) return null;
  const documentos = [];
  let conAlternativas = false;
  for (const documento of datos.documentos) {
    if (!camposExactos(documento, CAMPOS_DOCUMENTO) || !CLAVE.test(documento.documento)
      || !textoAcotado(documento.etiqueta) || !Array.isArray(documento.pasos)
      || documento.pasos.length < 1 || documento.pasos.length > MAXIMO_PASOS) return null;
    const pasos = documento.pasos.map((paso, indice) => validarPaso(paso, indice, documento.pasos.length, datos.esquema));
    if (pasos.includes(null)) return null;
    conAlternativas ||= pasos.some((paso) => Object.hasOwn(paso, CAMPO_ALTERNATIVAS));
    documentos.push(Object.freeze({ documento: documento.documento, etiqueta: documento.etiqueta, pasos: Object.freeze(pasos) }));
  }
  if ((datos.esquema === ESQUEMA_V2) !== conAlternativas) return null;
  return Object.freeze({
    catalogo_ref: datos.catalogo_ref, huella_sha256: datos.huella_sha256,
    ejemplo: datos.ejemplo, documentos: Object.freeze(documentos), portafirmas: conPortafirmas ? validarPortafirmas(datos.portafirmas) : NO_CONECTADO,
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

/** Cliente de solo lectura; la consulta tipada distingue denegación de fallo. */
export function crearClienteHTTPCircuitoFirma({ fetchImpl = globalThis.fetch } = {}) {
  async function obtenerCircuitoConEstado({ signal } = {}) {
    if (typeof fetchImpl !== "function") return Object.freeze({ estado: "no_disponible" });
    try {
      const respuesta = await fetchImpl(RUTA_CIRCUITO_FIRMA, {
        method: "GET", headers: { Accept: "application/json" }, signal,
        mode: "same-origin", credentials: "same-origin", cache: "no-store",
        redirect: "error", referrerPolicy: "no-referrer",
      });
      if (respuesta.redirected || respuesta.status !== 200
        || !/^application\/json(?:;\s*charset=utf-8)?$/iu.test(respuesta.headers.get("Content-Type") ?? "")) {
        void respuesta.body?.cancel?.().catch(() => {});
        return Object.freeze({ estado: respuesta.status === 401 || respuesta.status === 403 ? "denegado" : "no_disponible" });
      }
      const envoltorio = JSON.parse(await leerAcotado(respuesta));
      const circuito = camposExactos(envoltorio, ["data"]) ? validarCircuitoFirma(envoltorio.data) : null;
      return circuito ? Object.freeze({ estado: "disponible", circuito }) : Object.freeze({ estado: "no_disponible" });
    } catch {
      return Object.freeze({ estado: "no_disponible" });
    }
  }
  return Object.freeze({
    obtenerCircuitoConEstado,
    /** Conserva el contrato anterior para consumidores de solo lectura. */
    async obtenerCircuito({ signal } = {}) {
      const resultado = await obtenerCircuitoConEstado({ signal });
      return resultado.estado === "disponible" ? resultado.circuito : null;
    },
  });
}

function renderizarPaso(paso, t, circuito, documento) {
  const tono = circuito.estado_no_acreditado ? "neutro" : TONO_ESTADO[paso.estado];
  const cargo = traducirValorCircuitoFirma("cargo", paso.cargo, t);
  const detalles = [
    t(`circuito_firma_accion_${paso.accion}`),
    t(`circuito_firma_habilita_${paso.habilita}`),
    t(`circuito_firma_devolucion_${paso.devolucion}`),
    t(`circuito_firma_sustitucion_${paso.sustitucion}`),
  ];
  return `<li class="ct-circuito-paso ct-circuito-paso--${tono}"${!circuito.estado_no_acreditado && paso.estado === "pendiente_firma" ? ' aria-current="step"' : ""}>
    <span class="ct-circuito-orden" aria-hidden="true">${paso.orden}</span>
    <div class="ct-circuito-paso-cuerpo">
      <p class="ct-circuito-cargo">${escaparHTML(cargo)}</p>
      <p class="ct-circuito-detalle">${detalles.map(escaparHTML).join(" · ")}</p>
      ${paso.estado === "devuelto" && paso.motivo_devolucion ? `<p class="ct-circuito-motivo">${escaparHTML(t("circuito_firma_motivo", { motivo: paso.motivo_devolucion }))}</p>` : ""}
      ${circuito.preflight_compuesto ? "" : renderizarAccionesPaso(circuito, documento, paso, t)}
    </div>
    <span class="ct-circuito-estado ct-tono-${tono}">${escaparHTML(t(circuito.estado_no_acreditado ? "circuito_firma_estado_pendiente_consulta" : `circuito_firma_estado_${paso.estado}`, { cargo }))}</span>
  </li>`;
}

/**
 * HTML del bloque; los textos visibles salen del traductor o del catálogo.
 * `fase` ({ catalogo, textos }) añade la fase de firma siempre visible.
 */
export function renderizarCircuitoFirma(circuito, t, estadoConsulta = circuito ? "disponible" : "no_disponible", fase = null) {
  const claveFallo = estadoConsulta === "denegado" ? "circuito_firma_consulta_denegada"
    : estadoConsulta === "no_disponible" ? "circuito_firma_consulta_no_disponible"
      : "circuito_firma_portafirmas_estado_no_disponible";
  return `<section class="ct-circuito-firma" aria-labelledby="ct-circuito-firma-titulo" data-ct-circuito-firma>
    <header class="ct-circuito-cabecera">
      <h3 id="ct-circuito-firma-titulo">${escaparHTML(t("circuito_firma_titulo"))}</h3>
      ${circuito?.registro ? `<span class="ct-circuito-marca">${escaparHTML(t("circuito_firma_sin_eficacia"))}</span>` : ""}
    </header>
    <p id="ct-firma-aviso" class="ct-circuito-aviso" role="status" aria-live="polite" tabindex="-1" data-ct-firma-aviso></p>
    ${fase ? renderizarFaseFirma({
    catalogo: fase.catalogo, real: circuito?.registro ? circuito : null, textos: fase.textos,
    nombrar: (tipo, valor) => traducirValorCircuitoFirma(tipo, valor, t), aviso: estadoConsulta !== "denegado",
    mostrarDescargaFirmado: false,
  }) : ""}
    <section class="ct-circuito-portafirmas" aria-labelledby="ct-circuito-portafirmas-titulo">
      <h4 id="ct-circuito-portafirmas-titulo">${escaparHTML(t("circuito_firma_envio_corporativo_titulo"))}</h4>
      <span class="ct-circuito-estado ct-tono-aviso">${escaparHTML(t("circuito_firma_envio_corporativo_pendiente"))}</span>
      <div class="ct-circuito-envio">
        <button type="button" class="boton-primario" disabled aria-describedby="ct-circuito-envio-motivo">${escaparHTML(t("circuito_firma_envio_corporativo_accion"))}</button>
        <p id="ct-circuito-envio-motivo">${escaparHTML(t("circuito_firma_envio_corporativo_bloqueado"))}</p>
      </div>
      <details class="ct-circuito-limite"><summary>${escaparHTML(t("circuito_firma_portafirmas_detalle"))}</summary>
        <p>${escaparHTML(t("circuito_firma_portafirmas_sin_envio"))}</p>
      </details>
    </section>
    ${circuito ? `<details class="ct-circuito-prueba" data-ct-firma-detalles>
      <summary>${escaparHTML(t("circuito_firma_autofirma_prueba"))} · ${escaparHTML(t("circuito_firma_sin_eficacia"))} · ${escaparHTML(t("circuito_firma_ver_pasos"))}</summary>
      <div class="ct-circuito-documentos">${circuito.documentos.map((documento) => {
    const etiqueta = traducirValorCircuitoFirma("documento", documento.etiqueta, t);
    return `<article class="ct-circuito-documento" data-ct-circuito-documento="${escaparHTML(documento.documento)}">
      <h5>${escaparHTML(etiqueta)}</h5>
      <ol aria-label="${escaparHTML(t("circuito_firma_pasos", { documento: etiqueta }))}">${documento.pasos.map((paso) => renderizarPaso(paso, t, circuito, documento)).join("")}</ol>
      ${circuito.preflight_compuesto ? renderizarAccionesPaso(circuito, documento, documento.pasos[0], t) : ""}
    </article>`;
  }).join("")}</div></details>` : fase && estadoConsulta === "no_disponible" ? "" : `<p class="ct-circuito-indisponible" role="${estadoConsulta === "denegado" ? "alert" : "status"}">${escaparHTML(t(claveFallo))}</p>`}
  </section>`;
}

/**
 * Monta el bloque tras la cabecera del expediente. La consulta se hace una
 * vez por montaje; si falla el catálogo, la fase oficial sigue visible y
 * el estado de los pasos se declara no disponible.
 */
export function crearGestorCircuitoFirma({
  raiz, obtenerEstado, cliente = crearClienteHTTPCircuitoFirma(), mensajes = {}, esMontada = () => true,
  clienteFirma = crearClienteFirmaDocumento(), dependenciasAcciones = {}, cargarTextos,
  locale = globalThis.document?.documentElement?.lang || "es-ES",
  crearDocumentos = crearFuenteDocumentosHTTP, entornoDescarga = globalThis,
} = {}) {
  if (typeof obtenerEstado !== "function") throw new TypeError("estado del circuito de firma no disponible");
  const t = crearTraductorCircuitoFirma(mensajes, locale);
  const controlador = new AbortController();
  let consulta = null;
  let circuitoActual = null;
  let secuenciaMontaje = 0;
  const consultaFirmaCompuesta = typeof dependenciasAcciones.clientePreflight?.consultar === "function"
    && typeof dependenciasAcciones.obtenerVinculoOriginal === "function"
    && typeof dependenciasAcciones.obtenerOriginal === "function";
  const textosFase = cargarTextosFaseFirma(cargarTextos);
  const fase = async (resultado) => {
    const textos = await textosFase;
    const catalogo = resultado?.catalogo ?? resultado?.circuito ?? null;
    return textos && catalogo ? { catalogo, textos } : null;
  };
  function expedienteConsultable(estado) {
    const expediente = estado?.expediente;
    if (!["expediente", "documentos"].includes(estado?.vista) || estado.carga !== "listo"
      || expediente?.demostracion !== false || !expediente.expediente_ref
      || estado.expediente_ref !== expediente.expediente_ref
      || !Number.isSafeInteger(expediente.version) || expediente.version < 1) return null;
    if (estado.vista === "documentos" && (estado.documentos?.demostracion !== false
      || estado.documentos.expediente_ref !== expediente.expediente_ref
      || estado.documentos.version !== expediente.version)) return null;
    return { ref: expediente.expediente_ref, version: expediente.version, vista: estado.vista };
  }
  function pintarFirmados(resultado, datosFase) {
    const zona = raiz.querySelector?.("[data-ct-exp-firmados]");
    if (!zona) return;
    zona.innerHTML = renderizarFirmadosEnDocumentos(resultado?.circuito, datosFase?.textos,
      (tipo, valor) => traducirValorCircuitoFirma(tipo, valor, t));
  }

  async function obtenerCatalogo() {
    if (typeof cliente?.obtenerCircuitoConEstado === "function") {
      const resultado = await cliente.obtenerCircuitoConEstado({ signal: controlador.signal });
      if (resultado?.estado === "disponible" && resultado.circuito) return resultado;
      return { estado: resultado?.estado === "denegado" ? "denegado" : "no_disponible" };
    }
    const circuito = await cliente.obtenerCircuito({ signal: controlador.signal });
    return { estado: circuito ? "disponible" : "no_disponible", circuito };
  }

  // El estado real se consulta con cualquier expediente real abierto, para
  // que la fase de firma se vea en todas sus fases; firmar o devolver solo se
  // ofrece cuando sus borradores se pueden descargar (fase de nombramiento).
  // Sin registro compuesto el bloque sigue siendo informativo.
  async function conEstadoReal(resultado, contextoEsperado = null) {
    const circuito = resultado?.circuito;
    const estado = obtenerEstado();
    const contextoActual = expedienteConsultable(estado);
    if (contextoEsperado && (contextoActual?.ref !== contextoEsperado.ref
      || contextoActual.version !== contextoEsperado.version || contextoActual.vista !== contextoEsperado.vista)) return resultado;
    const solicitud = circuito ? solicitudInformeDefinitivoDesdeEstado(estado) : null;
    const expedienteRef = circuito ? contextoActual?.ref : null;
    if (!expedienteRef) return resultado;
    let respuesta;
    if (typeof clienteFirma?.consultarConEstado === "function") {
      respuesta = await clienteFirma.consultarConEstado(expedienteRef, { signal: controlador.signal })
        .catch(() => ({ estado: "no_disponible" }));
    } else if (typeof clienteFirma?.consultar === "function") {
      const datos = await clienteFirma.consultar(expedienteRef, { signal: controlador.signal }).catch(() => null);
      respuesta = { estado: datos ? "disponible" : "no_disponible", datos };
    } else {
      respuesta = { estado: "no_disponible" };
    }
    // El catálogo describe los pasos, pero no acredita su estado. Si CT118 no
    // responde o no coincide, no se muestran estados derivados del ejemplo.
    const fusionado = respuesta?.estado === "disponible" ? fusionarEstadoFirmas(circuito, respuesta.datos) : null;
    const permitirAcciones = estado.vista === "expediente" && Boolean(solicitud);
    const real = fusionado ? Object.freeze({ ...fusionado,
      acciones: permitirAcciones, preflight_compuesto: consultaFirmaCompuesta && permitirAcciones,
    }) : consultaFirmaCompuesta && permitirAcciones ? Object.freeze({ ...circuito,
      acciones: true, preflight_compuesto: true, estado_no_acreditado: true,
    }) : null;
    return { estado: real ? "disponible" : respuesta?.estado === "denegado" ? "denegado" : "no_disponible", circuito: real, catalogo: circuito };
  }

  const acciones = crearAccionesFirma({
    ...dependenciasAcciones, obtenerEstado, obtenerCircuito: () => circuitoActual, t,
    async alCambiar(aviso) {
      const nuevo = await conEstadoReal(await consulta);
      const datosFase = await fase(nuevo);
      const actual = raiz.querySelector?.("[data-ct-circuito-firma]");
      if (!actual || !esMontada()) return;
      circuitoActual = nuevo.circuito;
      const detallesAbiertos = Boolean(actual.querySelector?.("[data-ct-firma-detalles]")?.open);
      const limiteAbierto = Boolean(actual.querySelector?.(".ct-circuito-limite")?.open);
      actual.outerHTML = renderizarCircuitoFirma(nuevo.circuito, t, nuevo.estado, datosFase);
      pintarFirmados(nuevo, datosFase);
      const repintado = raiz.querySelector?.("[data-ct-circuito-firma]");
      repintado?.addEventListener?.("click", manejar);
      const detalles = repintado?.querySelector?.("[data-ct-firma-detalles]");
      if (detalles) detalles.open = detallesAbiertos;
      const limite = repintado?.querySelector?.(".ct-circuito-limite");
      if (limite) limite.open = limiteAbierto;
      const aviso_ = repintado?.querySelector?.("[data-ct-firma-aviso]");
      if (aviso_) {
        aviso_.textContent = aviso;
        aviso_.focus?.({ preventScroll: true });
        aviso_.scrollIntoView?.({ block: "nearest", inline: "nearest" });
      }
    },
  });
  // Descarga el PDF firmado que guarda Documentos: la ruta de Documentos
  // autoriza la terna expediente, documento y versión, y el cliente coteja la
  // huella antes de entregar el fichero.
  let descargando = false;
  async function descargarFirmado(boton) {
    if (descargando) return;
    descargando = true;
    boton.setAttribute("aria-disabled", "true");
    boton.setAttribute("aria-describedby", boton.closest?.("[data-ct-exp-firmados]")
      ? "ct-firma-descarga-aviso" : "ct-firma-aviso");
    const textos = await textosFase;
    const aviso = boton.closest?.("[data-ct-exp-firmados]")?.querySelector?.("#ct-firma-descarga-aviso")
      ?? boton.closest?.("[data-ct-circuito-firma]")?.querySelector?.("[data-ct-firma-aviso]");
    const decir = (clave, valores) => { if (aviso && textos) aviso.textContent = textos.traducir(clave, valores); };
    const documento = boton.closest?.(".ct-fase-firma-fila")?.querySelector?.(".ct-fase-firma-documento strong")?.textContent?.trim();
    decir("fase.descargando");
    let url = "";
    try {
      const d = boton.dataset;
      const fuente = crearDocumentos({ expedienteRef: d.ctFirmadoExpediente });
      const archivo = await fuente.descargar(d.ctFirmadoDocumento, {
        version: Number(d.ctFirmadoVersion), mime: "application/pdf", huella: d.ctFirmadoHuella, signal: controlador.signal,
      });
      const pagina = entornoDescarga.document;
      if (!pagina?.body || typeof entornoDescarga.URL?.createObjectURL !== "function" || typeof entornoDescarga.Blob !== "function") {
        throw new TypeError("descarga no disponible");
      }
      url = entornoDescarga.URL.createObjectURL(new entornoDescarga.Blob([archivo.contenido], { type: archivo.tipo }));
      const enlace = pagina.createElement("a");
      try {
        enlace.href = url; enlace.download = archivo.nombre; enlace.hidden = true;
        pagina.body.append(enlace); enlace.click();
      } finally { enlace.remove(); }
      decir(documento ? "fase.descargado" : "fase.descargado_sin_nombre", { documento });
    } catch (error) {
      if (controlador.signal.aborted) return;
      decir(error?.codigo === "denegado" ? "fase.descarga_denegada" : "fase.descarga_error");
    } finally {
      if (url) setTimeout(() => entornoDescarga.URL.revokeObjectURL?.(url), 0);
      boton.removeAttribute("aria-disabled");
      descargando = false;
    }
  }

  function manejar(evento) {
    const boton = evento?.target?.closest?.("[data-ct-descargar-firmado]");
    if (boton?.dataset?.ctDescargarFirmado !== undefined) { void descargarFirmado(boton); return; }
    void acciones.manejarClic(evento);
  }
  function manejarFirmadoEnDocumentos(evento) {
    const boton = evento?.target?.closest?.("[data-ct-exp-firmados] [data-ct-descargar-firmado]");
    if (boton && raiz.contains?.(boton)) { evento.preventDefault?.(); void descargarFirmado(boton); }
  }
  raiz.addEventListener?.("click", manejarFirmadoEnDocumentos);

  function insertar(resultado, datosFase) {
    if (!esMontada() || raiz.querySelector?.("[data-ct-circuito-firma]")) return;
    // En la ficha, la firma va con los documentos; sin esa marca, tras el siguiente paso.
    const ancla = raiz.querySelector?.("[data-ct-exp-ancla-firma]")
      ?? raiz.querySelector?.(".ct-exp-siguiente-paso") ?? raiz.querySelector?.(".ct-exp-progreso");
    if (typeof ancla?.insertAdjacentHTML !== "function") return;
    circuitoActual = resultado.circuito;
    ancla.insertAdjacentHTML("afterend", renderizarCircuitoFirma(resultado.circuito, t, resultado.estado, datosFase));
    pintarFirmados(resultado, datosFase);
    raiz.querySelector?.("[data-ct-circuito-firma]")?.addEventListener?.("click", manejar);
  }

  function montarSiProcede(estado) {
    acciones.actualizar();
    const contexto = expedienteConsultable(estado);
    if (!contexto ||
      (typeof cliente?.obtenerCircuito !== "function" && typeof cliente?.obtenerCircuitoConEstado !== "function")) return;
    consulta ??= Promise.resolve(obtenerCatalogo()).catch(() => ({ estado: "no_disponible" }));
    const actualMontaje = ++secuenciaMontaje;
    void consulta.then((resultado) => conEstadoReal(resultado, contexto)).then(async (resultado) => {
      const datosFase = await fase(resultado);
      const actual = expedienteConsultable(obtenerEstado());
      if (!esMontada() || controlador.signal.aborted || actualMontaje !== secuenciaMontaje
        || actual?.ref !== contexto.ref || actual.version !== contexto.version || actual.vista !== contexto.vista) return;
      if (contexto.vista === "documentos") pintarFirmados(resultado, datosFase);
      else insertar(resultado, datosFase);
    });
  }

  return Object.freeze({
    montarSiProcede,
    retirar() { ++secuenciaMontaje; circuitoActual = null; acciones.retirar(); controlador.abort(); raiz.removeEventListener?.("click", manejarFirmadoEnDocumentos); },
  });
}
