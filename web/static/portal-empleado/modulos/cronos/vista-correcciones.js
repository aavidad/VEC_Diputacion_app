import { MENSAJES_CRONOS_C5_ES } from "./i18n-c5.js?v=20260924-c5-web1";

const ESTADOS = new Set(["no_configurado", "cargando", "vacio", "disponible", "error", "denegado"]);

function escaparHTML(valor) {
  return String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}

function traducir(clave, mensajes) {
  const valor = mensajes?.[clave] ?? MENSAJES_CRONOS_C5_ES[clave];
  if (typeof valor !== "string" || !valor) throw new TypeError(`texto C5 no válido: ${clave}`);
  return escaparHTML(valor);
}

/** La corrección no se presenta como trámite hasta disponer de su servicio autorizado. */
export function renderizarCorreccionesCronos({ estado = "no_configurado", mensajes = {} } = {}) {
  if (!ESTADOS.has(estado)) throw new TypeError("estado C5 no válido");
  const t = (clave) => traducir(clave, mensajes);
  const estadoConsulta = estado === "disponible" ? "c5_original_disponible"
    : estado === "no_configurado" ? "c5_original_no_disponible" : `c5_original_${estado}`;
  return `<section class="cronos-c5 panel" data-cronos-c5-estado="${estado}" aria-labelledby="cronos-c5-titulo">
    <header class="cabecera-panel"><h3 id="cronos-c5-titulo">${t("c5_titulo")}</h3><span class="cronos-c5-estado" role="status">${t("c5_estado_no_configurado")}</span></header>
    <div class="cuerpo-panel"><p class="cronos-c5-limite" role="status">${t(estadoConsulta)}</p>
      <button type="button" class="boton-primario" disabled aria-disabled="true" aria-describedby="cronos-c5-sin-envio">${t("c5_enviar")}</button>
      <p id="cronos-c5-sin-envio" class="cronos-c5-motivo">${t("c5_sin_envio")}</p>
    </div>
  </section>`;
}

export function montarVistaCorreccionesCronos({ raiz, registrarDesmontar, ...proyeccion } = {}) {
  if (!raiz?.append || !raiz.ownerDocument?.createElement
    || (registrarDesmontar !== undefined && typeof registrarDesmontar !== "function")) throw new TypeError("vista C5 no disponible");
  const contenedor = raiz.ownerDocument.createElement("section");
  contenedor.dataset.cronosC5 = "";
  contenedor.innerHTML = renderizarCorreccionesCronos(proyeccion);
  raiz.append(contenedor);
  let activa = true;
  const desmontar = () => {
    if (!activa) return;
    activa = false;
    contenedor.innerHTML = "";
    contenedor.remove?.();
  };
  registrarDesmontar?.(desmontar);
  return Object.freeze({ desmontar });
}
