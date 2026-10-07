import { IDIOMAS_DISPONIBLES, cambiarIdioma } from "../../../comun/idioma.js";
import { referenciaConsultaValida } from "./consulta-contrato.js";
import { crearLectorConsultaMeritoPropio, ErrorConsultaMerito } from "./consulta-cliente-http.js";
import { crearTextosConsultaMerito } from "./consulta-i18n.js";
import { montarConsultaMeritoPropio, renderizarConsultaMerito } from "./consulta-vista.js";

const RUTA_CONTEXTO = "/api/meritos/consulta-contexto";
const MAX_CONTEXTO_BYTES = 1024;

/** Solo el contexto privado aporta la referencia conocida y la naturaleza del ejercicio. */
export async function obtenerContextoConsultaMerito({ fetchImpl = globalThis.fetch, signal } = {}) {
  if (typeof fetchImpl !== "function" || typeof signal?.aborted !== "boolean"
    || typeof signal.addEventListener !== "function" || typeof signal.removeEventListener !== "function") throw new TypeError("meritos.consulta.contexto_invalido");
  const controlador = new AbortController();
  const abortar = () => controlador.abort();
  if (signal.aborted) throw new ErrorConsultaMerito("abortada");
  signal.addEventListener("abort", abortar, { once: true });
  const timer = setTimeout(abortar, 10_000);
  let reader;
  try {
    const respuesta = await fetchImpl(RUTA_CONTEXTO, {
      method: "GET", mode: "same-origin", credentials: "omit", cache: "no-store", redirect: "error", referrerPolicy: "no-referrer",
      headers: { Accept: "application/json" }, signal: controlador.signal,
    });
    if (controlador.signal.aborted) throw new ErrorConsultaMerito("abortada");
    if (respuesta?.status !== 200 || respuesta.ok !== true || respuesta.redirected === true) {
      try { await respuesta?.body?.cancel?.(); } catch {}
      throw new ErrorConsultaMerito([401, 403].includes(respuesta?.status) ? "denegada" : "error");
    }
    const tipo = respuesta.headers?.get?.("content-type") ?? "";
    const longitud = respuesta.headers?.get?.("content-length");
    if (!/^application\/json(?:;\s*charset=utf-8)?$/iu.test(tipo)
      || (longitud !== null && longitud !== undefined && (!/^\d+$/u.test(longitud) || Number(longitud) > MAX_CONTEXTO_BYTES))) {
      try { await respuesta.body?.cancel?.(); } catch {}
      throw new ErrorConsultaMerito("error");
    }
    reader = respuesta.body?.getReader?.();
    if (!reader) throw new ErrorConsultaMerito("error");
    const alAborto = () => { Promise.resolve(reader.cancel()).catch(() => {}); };
    controlador.signal.addEventListener("abort", alAborto, { once: true });
    let texto = ""; let bytes = 0; let fragmentos = 0;
    const decoder = new TextDecoder("utf-8", { fatal: true });
    try {
      while (true) {
        const { done, value } = await reader.read();
        if (controlador.signal.aborted) throw new ErrorConsultaMerito("abortada");
        if (done) break;
        if (!(value instanceof Uint8Array) || ++fragmentos > 64 || (bytes += value.byteLength) > MAX_CONTEXTO_BYTES) throw new ErrorConsultaMerito("error");
        texto += decoder.decode(value, { stream: true });
      }
      texto += decoder.decode();
    } finally { controlador.signal.removeEventListener("abort", alAborto); }
    const contexto = JSON.parse(texto);
    if (!contexto || typeof contexto !== "object" || Array.isArray(contexto)
      || Object.keys(contexto).length !== 2 || !Object.hasOwn(contexto, "hecho_ref") || !Object.hasOwn(contexto, "sintetico")
      || !referenciaConsultaValida(contexto.hecho_ref) || contexto.sintetico !== true) throw new ErrorConsultaMerito("error");
    return Object.freeze({ hecho_ref: contexto.hecho_ref, sintetico: true });
  } catch (causa) {
    try { await reader?.cancel?.(); } catch {}
    if (signal.aborted) throw new ErrorConsultaMerito("abortada");
    if (causa instanceof ErrorConsultaMerito && causa.codigo === "denegada") throw causa;
    throw new ErrorConsultaMerito("error");
  } finally {
    reader?.releaseLock?.(); clearTimeout(timer); signal.removeEventListener("abort", abortar);
  }
}

/** Consumidor privado de lectura. No pertenece a los manifiestos institucionales. */
export function montarConsumidorConsultaMerito({ documento = globalThis.document, ventana = globalThis.window, fetchImpl = globalThis.fetch, textos = crearTextosConsultaMerito() } = {}) {
  const raiz = documento?.getElementById?.("consulta-espacio");
  const aviso = documento?.getElementById?.("consulta-aviso-sintetico");
  const anuncio = documento?.getElementById?.("consulta-anuncio");
  const idiomas = documento?.getElementById?.("consulta-idiomas");
  if (!raiz || !aviso || !anuncio || !idiomas || !documento.documentElement) throw new TypeError("meritos.consulta.documento_invalido");
  const { t } = textos;
  documento.documentElement.lang = textos.idioma; documento.title = t("titulo");
  for (const elemento of documento.querySelectorAll("[data-consulta-texto]")) elemento.textContent = t(elemento.dataset.consultaTexto);
  for (const elemento of documento.querySelectorAll("[data-consulta-etiqueta]")) elemento.setAttribute("aria-label", t(elemento.dataset.consultaEtiqueta));
  if (ventana?.location?.href && typeof ventana.history?.replaceState === "function") {
    const destino = new URL(ventana.location.href);
    if (destino.searchParams.has("lang") && destino.searchParams.get("lang") !== textos.idioma) {
      destino.searchParams.set("lang", textos.idioma);
      ventana.history.replaceState(ventana.history.state, "", destino);
    }
  }
  const enlace = documento.querySelector("[data-consulta-enlace-actual]");
  if (enlace && ventana?.location?.href) enlace.href = ventana.location.href;
  const controlador = new AbortController();
  let activa = true; let ficha; let revision = 0; let recuperada;
  const anunciar = (mensaje) => { if (activa) anuncio.textContent = mensaje; };
  const botones = IDIOMAS_DISPONIBLES.map((idioma) => {
    const boton = documento.createElement("button");
    boton.type = "button"; boton.className = "boton-secundario"; boton.textContent = idioma.nombre; boton.lang = idioma.codigo;
    boton.setAttribute("aria-pressed", String(idioma.codigo === textos.idioma));
    const cambiar = () => cambiarIdioma(idioma.codigo, ventana?.location);
    boton.addEventListener("click", cambiar); idiomas.append(boton);
    return { boton, cambiar };
  });
  const abrir = async (recuperarFoco = false) => {
    if (!activa) return;
    const vigente = ++revision; ficha?.desmontar(); aviso.hidden = true;
    raiz.innerHTML = renderizarConsultaMerito({ estado: "cargando" }, textos); anunciar(t("cargando"));
    if (recuperarFoco) { raiz.tabIndex = -1; raiz.focus?.({ preventScroll: true }); }
    try {
      const contexto = await obtenerContextoConsultaMerito({ fetchImpl, signal: controlador.signal });
      if (!activa || vigente !== revision) return;
      aviso.hidden = !contexto.sintetico;
      raiz.innerHTML = "";
      ficha = montarConsultaMeritoPropio({ raiz, lector: crearLectorConsultaMeritoPropio({ fetchImpl }), hechoRef: contexto.hecho_ref, textos, anunciar });
      await ficha.preparada;
    } catch (causa) {
      if (!activa || controlador.signal.aborted || vigente !== revision) return;
      const estado = causa?.codigo === "denegada" ? "denegada" : "error";
      raiz.innerHTML = renderizarConsultaMerito({ estado }, textos);
      if (estado === "error") {
        const boton = documento.createElement("button"); boton.type = "button"; boton.className = "boton-secundario";
        boton.textContent = t("reintentar"); boton.dataset.consultaContextoReintentar = ""; raiz.append(boton);
      }
      anunciar(t(estado));
    }
  };
  const alClick = (evento) => { const boton = evento.target?.closest?.("[data-consulta-contexto-reintentar]"); if (boton && raiz.contains(boton)) void abrir(true); };
  const alRecuperar = (evento) => {
    if (!evento?.persisted) return;
    ventana?.removeEventListener?.("pageshow", alRecuperar);
    recuperada = montarConsumidorConsultaMerito({ documento, ventana, fetchImpl, textos });
  };
  const alOcultar = (evento) => {
    desmontar();
    if (evento?.persisted) ventana?.addEventListener?.("pageshow", alRecuperar);
  };
  const desmontar = () => {
    ventana?.removeEventListener?.("pageshow", alRecuperar);
    recuperada?.desmontar(); recuperada = undefined;
    if (!activa) return;
    activa = false; revision++; controlador.abort(); ficha?.desmontar(); raiz.innerHTML = ""; anuncio.textContent = ""; aviso.hidden = true;
    raiz.removeEventListener("click", alClick); ventana?.removeEventListener?.("pagehide", alOcultar);
    for (const { boton, cambiar } of botones) { boton.removeEventListener("click", cambiar); boton.remove(); }
  };
  raiz.addEventListener("click", alClick); ventana?.addEventListener?.("pagehide", alOcultar);
  const preparada = abrir();
  return Object.freeze({ get preparada() { return recuperada?.preparada ?? preparada; }, desmontar });
}

if (globalThis.document?.getElementById?.("consulta-espacio")) montarConsumidorConsultaMerito();
