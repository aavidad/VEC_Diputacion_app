import { crearTraductorCronos, MENSAJES_CRONOS_ES } from "./i18n.js?v=20260924-cronos-integrado-v1";
import { MENSAJES_CRONOS_C5_ES } from "./i18n-c5.js?v=20260924-c5-web1";

const ESTADOS = new Set(["no_configurado", "cargando", "vacio", "disponible", "error", "denegado"]);
const MOVIMIENTOS = new Set(["entrada", "salida", "inicio_pausa", "fin_pausa"]);

function escaparHTML(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}

function traductor(mensajes) {
  const general = crearTraductorCronos({ ...MENSAJES_CRONOS_ES, ...mensajes });
  return (clave) => {
    if (!Object.hasOwn(MENSAJES_CRONOS_C5_ES, clave)) return general(clave);
    const texto = mensajes?.[clave] ?? MENSAJES_CRONOS_C5_ES[clave];
    if (typeof texto !== "string" || texto.length === 0) throw new TypeError(`texto C5 no válido: ${clave}`);
    return texto;
  };
}

function fechaLocal(fecha) {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(fecha)) return "";
  const instante = new Date(`${fecha}T12:00:00Z`);
  if (!Number.isFinite(instante.getTime()) || instante.toISOString().slice(0, 10) !== fecha) return "";
  return new Intl.DateTimeFormat("es-ES", { day: "2-digit", month: "2-digit", year: "numeric", timeZone: "UTC" }).format(instante);
}

function movimientosVisibles(fichajes, estado) {
  if (estado !== "disponible") return [];
  if (!Array.isArray(fichajes) || fichajes.length > 62) throw new TypeError("movimientos C5 no válidos");
  const vistos = new Set();
  return fichajes.map((fichaje) => {
    if (!fichaje || typeof fichaje !== "object"
      || typeof fichaje.id !== "string" || !/^[A-Za-z0-9][A-Za-z0-9:._-]{5,127}$/.test(fichaje.id)
      || vistos.has(fichaje.id) || !MOVIMIENTOS.has(fichaje.tipo_clave)
      || typeof fichaje.instante !== "string"
      || !/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,6})?Z$/.test(fichaje.instante)
      || !Number.isFinite(Date.parse(fichaje.instante))) throw new TypeError("movimiento C5 no válido");
    vistos.add(fichaje.id);
    return fichaje;
  });
}

/** Recibe exclusivamente la proyección propia ya autorizada por el consumidor; no consulta ni concede acceso. */
export function renderizarCorreccionesCronos({ fichajes = [], estado = "no_configurado", mensajes = {} } = {}) {
  if (!ESTADOS.has(estado)) throw new TypeError("estado C5 no válido");
  const t = traductor(mensajes);
  const e = (clave) => escaparHTML(t(clave));
  const visibles = movimientosVisibles(fichajes, estado);
  const formato = new Intl.DateTimeFormat("es-ES", { dateStyle: "short", timeStyle: "short", timeZone: "Europe/Madrid" });
  const opciones = visibles.map((item) => `<option value="${escaparHTML(item.id)}">${escaparHTML(formato.format(new Date(item.instante)))} · ${e(`movimiento_${item.tipo_clave}`)}</option>`).join("");
  const estadoOriginal = estado === "disponible" && visibles.length === 0 ? "c5_original_vacio"
    : estado === "disponible" ? "" : `c5_original_${estado === "no_configurado" ? "no_disponible" : estado}`;
  return `<section class="cronos-c5 panel" data-cronos-c5-estado="${estado}" aria-labelledby="cronos-c5-titulo">
    <header class="cabecera-panel"><div><h3 id="cronos-c5-titulo">${e("c5_titulo")}</h3><p>${e("c5_descripcion")}</p></div><span class="cronos-c5-estado" role="status">${e("c5_estado_local")}</span></header>
    <div class="cuerpo-panel"><p class="cronos-c5-limite" role="status">${e("c5_limite")}</p>
      <div class="cronos-c5-rejilla"><form class="cronos-c5-formulario" data-cronos-c5-formulario novalidate>
        <label>${e("c5_fecha")}<input name="fecha" type="date" autocomplete="off"></label>
        <label>${e("c5_hora")}<input name="hora" type="time" autocomplete="off"></label>
        <label class="cronos-c5-ancho">${e("c5_tipo")}<select name="tipo"><option value="">${e("c5_revision_vacio")}</option>${[...MOVIMIENTOS].map((tipo) => `<option value="${tipo}">${e(`movimiento_${tipo}`)}</option>`).join("")}</select></label>
        <label class="cronos-c5-ancho">${e("c5_original")}<select name="original"><option value="">${e("c5_original_ninguno")}</option>${opciones}</select></label>
        ${estadoOriginal ? `<p class="cronos-c5-ayuda cronos-c5-ancho" role="status">${e(estadoOriginal)}</p>` : ""}
        <label class="cronos-c5-ancho">${e("c5_motivo")}<textarea name="motivo" maxlength="500" rows="3" autocomplete="off"></textarea></label>
        <p class="cronos-c5-ayuda cronos-c5-ancho">${e("c5_motivo_ayuda")}</p>
      </form><aside class="cronos-c5-revision" aria-labelledby="cronos-c5-revision-titulo"><h4 id="cronos-c5-revision-titulo">${e("c5_revision")}</h4>
        <dl><div><dt>${e("c5_revision_fecha")}</dt><dd data-cronos-c5-resumen="fecha">${e("c5_revision_vacio")}</dd></div>
          <div><dt>${e("c5_revision_tipo")}</dt><dd data-cronos-c5-resumen="tipo">${e("c5_revision_vacio")}</dd></div>
          <div><dt>${e("c5_revision_hora")}</dt><dd data-cronos-c5-resumen="hora">${e("c5_revision_vacio")}</dd></div>
          <div><dt>${e("c5_revision_original")}</dt><dd data-cronos-c5-resumen="original">${e("c5_original_ninguno")}</dd></div>
          <div><dt>${e("c5_revision_motivo")}</dt><dd data-cronos-c5-resumen="motivo">${e("c5_revision_vacio")}</dd></div></dl>
        <p class="cronos-c5-ayuda">${e("c5_ayuda")}</p><button type="button" class="boton-primario" disabled aria-disabled="true" title="${e("c5_sin_envio")}">${e("c5_enviar")}</button>
        <p class="cronos-c5-ayuda">${e("c5_sin_envio")}</p></aside></div>
    </div></section>`;
}

/** Mantiene los datos solo en los controles de la vista; al desmontar los elimina. */
export function montarVistaCorreccionesCronos({ raiz, registrarDesmontar, ...proyeccion } = {}) {
  if (!raiz?.append || !raiz.ownerDocument?.createElement
    || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) throw new TypeError("vista C5 no disponible");
  const contenedor = raiz.ownerDocument.createElement("section");
  contenedor.dataset.cronosC5 = "";
  contenedor.innerHTML = renderizarCorreccionesCronos(proyeccion);
  raiz.append(contenedor);
  const t = traductor(proyeccion.mensajes || {});
  const form = contenedor.querySelector?.("[data-cronos-c5-formulario]");
  const resumir = () => {
    if (!form) return;
    const campo = (nombre) => form.querySelector?.(`[name="${nombre}"]`);
    const fecha = fechaLocal(campo("fecha")?.value || "");
    const hora = /^([01]\d|2[0-3]):[0-5]\d$/.test(campo("hora")?.value || "") ? campo("hora").value : "";
    const tipo = MOVIMIENTOS.has(campo("tipo")?.value) ? t(`movimiento_${campo("tipo").value}`) : "";
    const original = campo("original");
    const valorOriginal = original?.selectedOptions?.[0]?.textContent || t("c5_original_ninguno");
    const motivo = (campo("motivo")?.value || "").trim();
    for (const [clave, valor] of Object.entries({ fecha, tipo, hora, original: valorOriginal, motivo })) {
      const nodo = contenedor.querySelector?.(`[data-cronos-c5-resumen="${clave}"]`);
      if (nodo) nodo.textContent = valor || t("c5_revision_vacio");
    }
  };
  const impedirEnvio = (evento) => { if (evento.target === form) evento.preventDefault(); };
  form?.addEventListener?.("input", resumir);
  form?.addEventListener?.("change", resumir);
  form?.addEventListener?.("submit", impedirEnvio);
  let activa = true;
  const desmontar = () => {
    if (!activa) return;
    activa = false;
    form?.removeEventListener?.("input", resumir);
    form?.removeEventListener?.("change", resumir);
    form?.removeEventListener?.("submit", impedirEnvio);
    form?.reset?.();
    contenedor.innerHTML = "";
    contenedor.remove?.();
  };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ desmontar });
}
