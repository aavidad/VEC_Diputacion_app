import { crearTraductorOportunidades } from "./i18n.js";

const ESTADOS_CONSULTA = new Set(["cargando", "disponible", "vacio", "no_configurado", "denegado", "error"]);
const ESTADOS_REQUISITO = new Set(["cumple", "no_cumple", "pendiente"]);
const escapar = (valor) => String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
const texto = (valor) => typeof valor === "string" ? valor.trim() : "";

function evaluarRequisito(requisito, oportunidad, t) {
  const fuente = texto(requisito?.procedencia);
  const motivo = texto(requisito?.motivo);
  const estructurado = requisito?.estructurado === true;
  const version = texto(oportunidad?.version_bases);
  const origen = texto(oportunidad?.fuente_evaluacion);
  const estado = ESTADOS_REQUISITO.has(requisito?.estado) ? requisito.estado : "pendiente";
  if (!estructurado) return { estado: "pendiente", motivo: t("texto_libre"), procedencia: fuente };
  if (!version) return { estado: "pendiente", motivo: t("sin_version"), procedencia: fuente };
  if (!origen) return { estado: "pendiente", motivo: t("sin_fuente"), procedencia: fuente };
  if (!fuente) return { estado: "pendiente", motivo: t("sin_procedencia"), procedencia: "" };
  if (!motivo) return { estado: "pendiente", motivo: t("sin_motivo"), procedencia: fuente };
  return { estado, motivo, procedencia: fuente };
}

function situacionGlobal(oportunidad, requisitos, t) {
  if (oportunidad?.estado_solicitud === "presentada" && texto(oportunidad?.fuente_solicitud)) return { codigo: "ya_presentada", texto: t("ya_presentada") };
  if (oportunidad?.estado_plazo === "cerrado" && texto(oportunidad?.fuente_plazo)) return { codigo: "plazo_cerrado", texto: t("plazo_cerrado") };
  if (oportunidad?.estado_plazo !== "abierto" || !texto(oportunidad?.fuente_plazo)) return { codigo: "pendiente", texto: t("plazo_sin_fuente") };
  if (!requisitos.length) return { codigo: "pendiente", texto: t("sin_requisitos") };
  if (requisitos.some((r) => r.estado === "no_cumple")) return { codigo: "no_cumple", texto: t("incumplimiento") };
  if (requisitos.some((r) => r.estado === "pendiente")) return { codigo: "pendiente", texto: t("preparar") };
  return { codigo: "cumple", texto: t("puede_iniciar") };
}

function renderizarOportunidad(oportunidad, indice, t) {
  const requisitos = Array.isArray(oportunidad?.requisitos) ? oportunidad.requisitos.map((r) => evaluarRequisito(r, oportunidad, t)) : [];
  const global = situacionGlobal(oportunidad, requisitos, t);
  const identificador = texto(oportunidad?.identificador_publico);
  const href = identificador ? `/bolsa/?${new URLSearchParams({ convocatoria: identificador }).toString()}` : "";
  const plazo = texto(oportunidad?.fuente_plazo) ? texto(oportunidad?.plazo_etiqueta) : "";
  return `<article class="oportunidades-ficha" data-estado="${global.codigo}" aria-labelledby="oportunidades-titulo-${indice}">
    <div class="oportunidades-ficha-cabecera"><div><p class="oportunidades-categoria">${escapar(texto(oportunidad?.categoria) || t("sin_categoria"))}</p><h4 id="oportunidades-titulo-${indice}">${escapar(texto(oportunidad?.titulo) || t("sin_titulo"))}</h4></div><span class="oportunidades-estado oportunidades-estado--${global.codigo}">${escapar(global.texto)}</span></div>
    <dl class="oportunidades-metadatos"><div><dt>${escapar(t("plazo"))}</dt><dd>${escapar(plazo || t("plazo_no_disponible"))}</dd></div><div><dt>${escapar(t("bases"))}</dt><dd>${escapar(texto(oportunidad?.version_bases) || t("sin_version"))}</dd></div><div><dt>${escapar(t("fuente"))}</dt><dd>${escapar(texto(oportunidad?.fuente_evaluacion) || t("sin_fuente"))}</dd></div></dl>
    <details class="oportunidades-requisitos"><summary>${escapar(t("requisitos"))} <span>(${requisitos.length})</span></summary>${requisitos.length ? `<ul>${requisitos.map((requisito, numero) => `<li><div><strong>${escapar(texto(oportunidad.requisitos[numero]?.etiqueta) || t("requisitos"))}</strong><span class="oportunidades-estado oportunidades-estado--${requisito.estado}">${escapar(t(requisito.estado))}</span></div><p>${escapar(requisito.motivo)}</p><small>${escapar(requisito.procedencia || t("sin_procedencia"))}</small></li>`).join("")}</ul>` : `<p>${escapar(t("sin_requisitos"))}</p>`}</details>
    <div class="oportunidades-acciones">${href ? `<a href="${escapar(href)}">${escapar(t("ver_detalle"))}</a>` : `<span>${escapar(t("detalle_no_disponible"))}</span>`}<button type="button" disabled aria-disabled="true" title="${escapar(t("iniciar_pendiente"))}">${escapar(t("iniciar"))}</button></div>
  </article>`;
}

/** Vista pura. El consumidor inyecta una proyección autorizada; aquí no se calculan requisitos desde méritos. */
export function renderizarOportunidades({ estado = "no_configurado", oportunidades = [] } = {}, mensajes) {
  const t = crearTraductorOportunidades(mensajes);
  const estadoSeguro = ESTADOS_CONSULTA.has(estado) ? estado : "error";
  const lista = Array.isArray(oportunidades) ? oportunidades : [];
  const visible = estadoSeguro === "disponible" && lista.length ? "disponible" : estadoSeguro === "disponible" ? "vacio" : estadoSeguro;
  const mensaje = visible === "disponible" ? "" : `<div class="oportunidades-aviso" role="status"><p>${escapar(t(visible))}</p>${visible === "error" ? `<button type="button" data-oportunidades-reintentar>${escapar(t("reintentar"))}</button>` : ""}</div>`;
  return `<section class="oportunidades" data-oportunidades data-estado="${visible}" aria-labelledby="oportunidades-titulo"><header class="oportunidades-cabecera"><div><h2 id="oportunidades-titulo">${escapar(t("titulo"))}</h2><p>${escapar(t("descripcion"))}</p></div><details class="oportunidades-ayuda"><summary aria-label="${escapar(t("ayuda_etiqueta"))}" title="${escapar(t("ayuda_etiqueta"))}">?</summary><p>${escapar(t("ayuda"))}</p></details></header><section class="oportunidades-panel panel" aria-labelledby="oportunidades-apartado"><header class="cabecera-panel"><div><h3 id="oportunidades-apartado">${escapar(t("apartado"))}</h3><p>${escapar(t("apartado_ayuda"))}</p></div></header><div class="cuerpo-panel">${mensaje}${visible === "disponible" ? `<div class="oportunidades-lista">${lista.map((item, indice) => renderizarOportunidad(item, indice, t)).join("")}</div>` : ""}</div></section><p class="oportunidades-limite">${escapar(t("limite"))}</p></section>`;
}

/** Montaje sin red propia. cargar({signal}) debe devolver {oportunidades:[...]} o una lista. */
export function montarVistaOportunidades({ raiz, cargar, datos, anunciar = () => {}, registrarDesmontar, mensajes } = {}) {
  if (!raiz?.addEventListener || !raiz?.replaceChildren || (cargar !== undefined && typeof cargar !== "function") || typeof anunciar !== "function" || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) throw new TypeError("vista de oportunidades no disponible");
  const t = crearTraductorOportunidades(mensajes);
  let activa = true;
  let controlador;
  const pintar = (modelo) => { if (activa) raiz.innerHTML = renderizarOportunidades(modelo, mensajes); };
  const consultar = async () => {
    if (!activa) return;
    controlador?.abort();
    if (!cargar) { pintar({ estado: datos ? "disponible" : "no_configurado", oportunidades: datos?.oportunidades ?? datos }); return; }
    const actual = new AbortController(); controlador = actual;
    pintar({ estado: "cargando" });
    try {
      const respuesta = await cargar({ signal: actual.signal });
      if (!activa || actual.signal.aborted) return;
      if (["denegado", "no_configurado", "error", "vacio"].includes(respuesta?.estado)) { pintar({ estado: respuesta.estado }); return; }
      pintar({ estado: "disponible", oportunidades: respuesta?.oportunidades ?? respuesta });
    } catch (error) {
      if (!activa || actual.signal.aborted) return;
      pintar({ estado: error?.codigo === "denegado" ? "denegado" : "error" });
      anunciar(t("error"), "error");
    }
  };
  const alClick = (evento) => { if (evento.target?.closest?.("[data-oportunidades-reintentar]")) void consultar(); };
  raiz.addEventListener("click", alClick);
  void consultar();
  const desmontar = () => { if (!activa) return; activa = false; controlador?.abort(); raiz.removeEventListener("click", alClick); raiz.replaceChildren(); };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ desmontar, recargar: consultar });
}
