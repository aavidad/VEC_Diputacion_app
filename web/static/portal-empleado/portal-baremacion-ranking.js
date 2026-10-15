import { cargarTextos } from "../comun/textos.js";
import { rutaCandidatosBolsaCompartible } from "./portal-bolsas-ruta-filtros.js?v=20261015-r1-criterios-v1";

export function rutaRankingBolsa(search, bolsaRef = "", cursor = "") {
  const base = bolsaRef ? rutaCandidatosBolsaCompartible(search, bolsaRef) : rutaCandidatosBolsaCompartible(search, "bolsa:selector");
  const parametros = new URLSearchParams(base.split("#")[0]);
  if (!bolsaRef) parametros.delete("bolsa_ref");
  if (cursor) parametros.set("cursor", cursor);
  return `?${parametros}#bolsa/baremacion`;
}

export function renderizarRankingBolsa({ bolsas, referencia, pagina, carga, ayuda, cursor }, { textos, escaparHTML: e, search = "" }) {
  const t = (clave, variables) => e(textos.traducir(clave, variables));
  const numero = (valor) => textos.numero(valor);
  // Se conserva el decimal original: no se redondea ni pasa por Number.
  const puntos = (valor) => valor == null ? t("sin_puntos") : e(valor.replace(".", textos.numero(1.1).replace(/[0-9]/gu, "")));
  const seleccionada = bolsas.find((b) => b.bolsa_ref === referencia);
  const opciones = bolsas.map((b) => `<option value="${e(b.bolsa_ref)}"${b.bolsa_ref === referencia ? " selected" : ""}>${e(b.categoria)}</option>`).join("");
  const aviso = carga === "cargando" ? `<p role="status" aria-busy="true">${t("cargando")}</p>`
    : carga === "denegado" ? `<p role="alert">${t("denegado")}</p>`
      : carga === "error" ? `<p role="alert">${t("error")}</p><button type="button" class="boton-secundario" data-ranking-reintentar>${t("reintentar")}</button>`
        : bolsas.length === 0 ? `<p role="status">${t("sin_bolsas")}</p>` : "";
  const filas = pagina?.candidatos.map((c) => `<tr><td>${e(numero(c.orden_acta))}</td><th scope="row">${e(c.nombre_visible)}</th><td>${puntos(c.puntuacion)}</td><td>${c.orden == null ? t("sin_turno") : e(numero(c.orden))}</td></tr>`).join("") || "";
  const enlaceLista = seleccionada ? rutaCandidatosBolsaCompartible(search, referencia) : "";
  return `<section class="panel" aria-labelledby="ranking-titulo"><header class="cabecera-panel"><h2 id="ranking-titulo">${t("titulo")}</h2><button type="button" class="boton-secundario" data-ranking-ayuda aria-label="${t("ayuda_boton")}" aria-expanded="${ayuda}" aria-controls="ranking-ayuda">?</button></header><div class="cuerpo-panel"><div id="ranking-ayuda" ${ayuda ? "" : "hidden"}>${t("ayuda")}</div>${bolsas.length ? `<form data-ranking-selector class="barra-filtros"><label class="campo" for="ranking-bolsa"><span>${t("bolsa")}</span><select id="ranking-bolsa" name="bolsa"><option value="">${t("elegir")}</option>${opciones}</select></label><button type="submit" class="boton-primario">${t("consultar")}</button></form>` : ""}${aviso}</div></section>${seleccionada && carga === "listo" && pagina ? `<section class="panel"><header class="cabecera-panel"><h3>${e(seleccionada.categoria)}</h3><a class="enlace-tabla" href="${e(enlaceLista)}">${t("total", { cantidad: numero(pagina.bolsa.total) })}</a></header><div class="tabla-contenedor" tabindex="0" role="region" aria-label="${t("tabla")}"><table class="tabla-datos"><thead><tr>${["posicion", "persona", "puntos", "turno"].map((clave) => `<th scope="col">${t(clave)}</th>`).join("")}</tr></thead><tbody>${filas || `<tr><td colspan="4" class="vacio-controlado">${t("vacio")}</td></tr>`}</tbody></table></div><div class="cuerpo-panel acciones-vista paginacion-bolsa">${cursor ? `<a class="boton-secundario" href="${e(rutaRankingBolsa(search, referencia))}" data-ranking-cursor="">${t("primera")}</a>` : ""}${pagina.hay_mas ? `<a class="boton-secundario" href="${e(rutaRankingBolsa(search, referencia, pagina.cursor_siguiente))}" data-ranking-cursor="${e(pagina.cursor_siguiente)}">${t("siguiente")}</a>` : ""}<a class="boton-secundario" href="${e(enlaceLista)}">${t("ver_bolsa")}</a></div></section>` : ""}`;
}

export async function montarRankingBolsa({ raiz, bolsas, consultar, signal, escaparHTML, localizacion = globalThis.location, historial = globalThis.history }) {
  const textos = await cargarTextos("bolsa-ranking");
  if (signal.aborted) return;
  let referencia = "", cursor = "", pagina = null, carga = "listo", ayuda = false, peticion = null, turno = 0;
  const cancelar = () => { turno++; peticion?.abort(); };
  signal.addEventListener("abort", cancelar, { once: true });
  const pintar = () => { if (!signal.aborted) raiz.innerHTML = renderizarRankingBolsa({ bolsas, referencia, pagina, carga, ayuda, cursor }, { textos, escaparHTML, search: localizacion.search }); };
  function leerRuta() {
    const parametros = new URLSearchParams(localizacion.search);
    const propuestas = parametros.getAll("bolsa_ref"), cursores = parametros.getAll("cursor");
    referencia = propuestas.length === 1 && bolsas.some((b) => b.bolsa_ref === propuestas[0]) ? propuestas[0] : "";
    cursor = cursores.length === 1 && cursores[0].length <= 512 ? cursores[0] : "";
  }
  async function cargar() {
    cancelar(); pagina = null;
    if (!referencia) { carga = "listo"; pintar(); return; }
    peticion = new AbortController(); const actual = turno;
    carga = "cargando"; pintar();
    try {
      const respuesta = await consultar(referencia, { orden: "acta", limite: 50, cursor }, { signal: peticion.signal });
      if (actual !== turno || signal.aborted) return;
      if (!respuesta.ok || respuesta.datos.bolsa.bolsa_ref !== referencia) {
        carga = [401, 403].includes(respuesta.status) ? "denegado" : "error";
      } else { pagina = respuesta.datos; carga = "listo"; }
    } catch (error) { if (actual !== turno || signal.aborted || error.name === "AbortError") return; carga = "error"; }
    pintar();
  }
  const ir = (ref, siguiente = "") => { referencia = ref; cursor = siguiente; historial.pushState(null, "", rutaRankingBolsa(localizacion.search, referencia, cursor)); void cargar(); };
  const click = (evento) => {
    const boton = evento.target.closest?.("[data-ranking-ayuda], [data-ranking-cursor], [data-ranking-reintentar]");
    if (!boton || !raiz.contains(boton)) return;
    evento.preventDefault();
    if (boton.hasAttribute("data-ranking-ayuda")) { ayuda = !ayuda; pintar(); raiz.querySelector("[data-ranking-ayuda]")?.focus(); }
    else if (boton.hasAttribute("data-ranking-cursor")) ir(referencia, boton.dataset.rankingCursor);
    else void cargar();
  };
  const submit = (evento) => { if (!evento.target.matches("[data-ranking-selector]")) return; evento.preventDefault(); ir(new FormData(evento.target).get("bolsa")); };
  raiz.addEventListener("click", click); raiz.addEventListener("submit", submit);
  leerRuta(); void cargar();
  return { actualizarRuta() { const anterior = `${referencia}\0${cursor}`; leerRuta(); if (anterior !== `${referencia}\0${cursor}`) void cargar(); }, desmontar() { cancelar(); signal.removeEventListener("abort", cancelar); raiz.removeEventListener("click", click); raiz.removeEventListener("submit", submit); } };
}
