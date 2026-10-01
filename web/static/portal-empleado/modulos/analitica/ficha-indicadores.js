import { cargarTextos } from "../../../comun/textos.js";
import { leerRecursoJSON } from "../../../comun/idioma.js";

const URL_CATALOGO = new URL("./catalogo-indicadores.json?v=20261001-ana001-v1", import.meta.url);
const CLAVE = /^[a-z][a-z0-9_]*$/u;
const ID = /^[A-Za-z][A-Za-z0-9_-]*$/u;

function congelar(datos) {
  if (datos && typeof datos === "object") {
    for (const valor of Object.values(datos)) congelar(valor);
    Object.freeze(datos);
  }
  return datos;
}

// Definiciones descriptivas; esta hoja no calcula indicadores ni concede acceso.
export function normalizarCatalogoIndicadores(datos) {
  if (datos?.esquema !== "vec.analitica.definiciones.v1" || !Number.isSafeInteger(datos.version)
    || datos.version < 1 || datos.responsable !== null || datos.aprobacion !== "pendiente"
    || !datos.fuente || !Object.values(datos.fuente).every((v) => typeof v === "string" && v.length > 0 && v.length <= 240)
    || !Array.isArray(datos.indicadores) || datos.indicadores.length !== 5
    || new Set(datos.indicadores.map((i) => i?.id)).size !== 5
    || !datos.indicadores.every((i) => CLAVE.test(i?.id) && i.nombre === `indicadores.${i.id}.nombre`
      && i.definicion === `indicadores.${i.id}.definicion`)
    || !Array.isArray(datos.notas) || datos.notas.length > 10 || !datos.notas.every((n) => CLAVE.test(n))) {
    throw new TypeError("catalogo_indicadores_invalido");
  }
  return congelar(structuredClone(datos));
}

/** Carga solo datos estáticos del mismo origen y textos con el lector común. */
export async function cargarFichaIndicadores({ opcionesTextos = {}, leer = leerRecursoJSON } = {}) {
  const [datos, textos] = await Promise.all([leer(URL_CATALOGO), cargarTextos("analitica", opcionesTextos)]);
  return Object.freeze({ catalogo: normalizarCatalogoIndicadores(datos), textos });
}

function escapar(valor) {
  return String(valor).replaceAll("&", "&amp;").replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#039;");
}

/** HTML para el contenedor de Ayuda existente; no introduce otro diálogo. */
export function renderizarFichaIndicadores({ catalogo, textos, id = "ayuda-indicadores", respuesta = null, conCierre = true }) {
  if (!ID.test(id)) throw new TypeError("id_ficha_invalido");
  const datos = normalizarCatalogoIndicadores(catalogo);
  const t = (clave, variables) => escapar(textos.traducir(clave, variables));
  const compatible = respuesta?.esquema === datos.fuente.esquema_respuesta
    && Number.isSafeInteger(respuesta.corte_global) && respuesta.corte_global >= 0;
  const consulta = compatible
    ? `<p>${t("corte_consultado", { corte: textos.numero(respuesta.corte_global) })}</p>`
    : `<p role="status">${t(respuesta === null ? "consulta_pendiente" : "consulta_incompatible")}</p>`;
  const formato = compatible ? `<p>${t("esquema_consultado", { esquema: respuesta.esquema })}</p>` : "";
  const definiciones = datos.indicadores.map((i) => `<dt><strong>${t(i.nombre)}</strong></dt><dd>${t(i.definicion)}</dd>`).join("");
  const origen = ["funcion", "proyeccion", "corte", "endpoint", "migracion"].map((clave) =>
    `<dt>${t(clave === "corte" ? "corte_tecnico" : clave)}</dt><dd><code style="overflow-wrap:anywhere">${escapar(datos.fuente[clave])}</code></dd>`).join("");
  return `<section class="panel" aria-labelledby="${id}-titulo">
    <div class="cabecera-panel"><h2 id="${id}-titulo">${t("titulo")}</h2>
      ${conCierre ? `<button type="button" class="boton-secundario" data-analitica-cerrar>${t("cerrar")}</button>` : ""}</div>
    <div class="cuerpo-panel">${consulta}<dl>${definiciones}</dl>
      ${datos.notas.map((nota) => `<p>${t(nota)}</p>`).join("")}
      <p>${t("version", { version: textos.numero(datos.version) })}</p>
      <details><summary>${t("fuente")}</summary>${formato}<dl>${origen}</dl></details>
    </div></section>`;
}

/**
 * Recibe un contenedor exclusivo dentro de Ayuda. El anfitrión conserva el
 * diálogo y su gestión modal; abrir/cerrar solo gobierna esta hoja y su foco.
 * Cargar la ficha antes de montar evita cambios de foco por respuestas tardías.
 */
export function montarFichaIndicadores({ raiz, catalogo, textos, id, alCerrar = () => {} }) {
  const html = renderizarFichaIndicadores({ catalogo, textos, id });
  let montada = true;
  let origen;
  raiz.hidden = true;
  function cerrar() {
    if (!montada || raiz.hidden) return;
    raiz.hidden = true;
    alCerrar();
    if (origen?.isConnected !== false) origen?.focus?.();
    origen = null;
  }
  function click(evento) {
    if (evento.target?.closest?.("[data-analitica-cerrar]")) cerrar();
  }
  function tecla(evento) {
    if (evento.key === "Escape" && !raiz.hidden) {
      evento.preventDefault();
      evento.stopPropagation();
      cerrar();
    }
  }
  raiz.addEventListener("click", click);
  raiz.addEventListener("keydown", tecla);
  return Object.freeze({
    abrir(botonOrigen = raiz.ownerDocument?.activeElement) {
      if (!montada || !raiz.hidden) return;
      origen = botonOrigen;
      raiz.innerHTML = html;
      raiz.hidden = false;
      raiz.querySelector("[data-analitica-cerrar]")?.focus();
    },
    cerrar,
    desmontar() {
      if (!montada) return;
      cerrar();
      montada = false;
      raiz.removeEventListener("click", click);
      raiz.removeEventListener("keydown", tecla);
      raiz.innerHTML = "";
    },
  });
}
