import { TRAMITES_AYUDANTE_PORTAL } from "./ayuda-contenido.js?v=20260924-rescate-web-v2";
import { traducirPortal } from "./portal-i18n.js?v=20260924-rescate-web-v2";

const TEXTO = Object.freeze({
  titulo: traducirPortal("ayuda_contenido_302"),
  introduccion: traducirPortal("ayuda_contenido_303"),
  elegir: traducirPortal("ayuda_contenido_304"),
  pasos: traducirPortal("ayuda_contenido_305"),
  anterior: traducirPortal("ayuda_contenido_306"),
  siguiente: traducirPortal("ayuda_contenido_307"),
  ir: traducirPortal("ayuda_contenido_308"),
  volver: traducirPortal("ayuda_contenido_309"),
  detalle: traducirPortal("ayuda_contenido_310"),
  ocultar: traducirPortal("ayuda_contenido_311"),
  objetivo: traducirPortal("ayuda_contenido_312"),
  preparacion: traducirPortal("ayuda_contenido_313"),
  resultado: traducirPortal("ayuda_contenido_314"),
  actor: traducirPortal("ayuda_contenido_315"),
  limite: traducirPortal("ayuda_contenido_316"),
  aviso: traducirPortal("ayuda_contenido_317"),
  pendiente: traducirPortal("ayuda_contenido_318"),
});

function escaparHTML(valor) {
  return String(valor).replaceAll("&", "&amp;").replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}

function tramitePorId(tramites, id) {
  return tramites.find((tramite) => tramite.id === id) || null;
}

function validarVista(vista, etiqueta) {
  if (typeof vista !== "string" || !/^[a-z][a-z0-9-]{0,63}$/u.test(vista)) {
    throw new TypeError(`${etiqueta} no válida`);
  }
  return vista;
}

function validarTramites(tramites) {
  if (!Array.isArray(tramites) || tramites.length === 0 || tramites.length > 32) {
    throw new TypeError("catálogo del ayudante no disponible");
  }
  const ids = new Set();
  for (const tramite of tramites) {
    if (!tramite || typeof tramite !== "object" || !/^[a-z][a-z0-9-]{0,63}$/u.test(tramite.id || "")
      || ids.has(tramite.id) || !Array.isArray(tramite.pasos) || tramite.pasos.length === 0) {
      throw new TypeError("trámite del ayudante no válido");
    }
    ids.add(tramite.id);
    validarVista(tramite.vista, "vista del trámite");
    for (const paso of tramite.pasos) {
      if (!paso || typeof paso !== "object") throw new TypeError("paso del ayudante no válido");
      if (paso.vista !== undefined) validarVista(paso.vista, "vista del paso");
    }
  }
  return tramites;
}

function resolverPaso(tramite, indice) {
  const paso = tramite.pasos[indice];
  return Object.freeze({
    ...paso,
    selector: paso.selector || tramite.selector,
    activar: paso.activar || tramite.activar,
    vista: paso.vista || tramite.vista,
    bloqueado: paso.bloqueado ?? true,
  });
}

function renderizarLista(tramites, escapar) {
  return `<section class="ayudante-tramites" data-ayudante-tramites aria-labelledby="ayudante-tramites-titulo">
    <p class="ayudante-tramites-introduccion">${escapar(TEXTO.introduccion)}</p>
    <h3 id="ayudante-tramites-titulo">${escapar(TEXTO.elegir)}</h3>
    <ul class="ayudante-tramites-lista">${tramites.map((tramite) => `<li><button type="button" class="ayudante-tramite" data-ayudante-tramite="${escapar(tramite.id)}"><span>${escapar(tramite.titulo)}</span><small>${escapar(tramite.modulo)} · ${escapar(tramite.resumen)}</small></button></li>`).join("")}</ul>
    <p class="ayudante-tramites-extension">${escapar(traducirPortal("ayuda_extension"))}</p>
    <p class="ayudante-tramites-aviso">${escapar(TEXTO.aviso)}</p>
  </section>`;
}

function renderizarPaso(tramite, indice, escapar, expandido = false) {
  const paso = resolverPaso(tramite, indice);
  const detalleId = `ayudante-detalle-${tramite.id}-${indice}`;
  return `<section class="ayudante-tramites" data-ayudante-tramites aria-labelledby="ayudante-paso-titulo">
    <p class="sobrelinea">${escapar(tramite.modulo)} · ${escapar(traducirPortal("ayuda_paso_de", { actual: indice + 1, total: tramite.pasos.length }))}</p>
    <h3 id="ayudante-paso-titulo">${escapar(paso.titulo)}</h3>
    <p class="ayudante-tramites-instruccion">${escapar(paso.instruccion)}</p>
    <div class="ayudante-tramites-acciones">
      <button type="button" class="boton-secundario" data-ayudante-anterior ${indice === 0 ? "disabled aria-disabled=\"true\"" : ""}>${escapar(TEXTO.anterior)}</button>
      <button type="button" class="boton-primario" data-ayudante-ir>${escapar(TEXTO.ir)}</button>
      <button type="button" class="boton-secundario" data-ayudante-siguiente ${indice === tramite.pasos.length - 1 ? "disabled aria-disabled=\"true\"" : ""}>${escapar(TEXTO.siguiente)}</button>
    </div>
    <button type="button" class="ayudante-tramites-detalle" data-ayudante-detalle aria-expanded="${String(expandido)}" aria-controls="${detalleId}"><span aria-hidden="true">?</span> ${escapar(expandido ? TEXTO.ocultar : TEXTO.detalle)}</button>
    <dl id="${detalleId}" class="ayudante-tramites-explicacion" ${expandido ? "" : "hidden"}>
      <div><dt>${escapar(TEXTO.objetivo)}</dt><dd>${escapar(paso.objetivo)}</dd></div>
      <div><dt>${escapar(TEXTO.preparacion)}</dt><dd>${escapar(paso.preparacion)}</dd></div>
      <div><dt>${escapar(TEXTO.resultado)}</dt><dd>${escapar(paso.resultado)}</dd></div>
      <div><dt>${escapar(TEXTO.actor)}</dt><dd>${escapar(paso.actor)}</dd></div>
      <div><dt>${escapar(TEXTO.limite)}</dt><dd>${escapar(paso.limite)}</dd></div>
    </dl>
    ${paso.bloqueado ? `<p class="ayudante-tramites-pendiente">${escapar(TEXTO.pendiente)}</p>` : ""}
    <button type="button" class="enlace-tabla" data-ayudante-volver>${escapar(TEXTO.volver)}</button>
  </section>`;
}

function enfocarDestino(documento, selector, activar) {
  let intentos = 0;
  const buscar = () => {
    const activador = activar ? documento.querySelector(activar) : null;
    if (activador && !activador.closest("[hidden]")) activador.click?.();
    const destino = selector ? documento.querySelector(selector) : null;
    if (destino) {
      if (!destino.hasAttribute("tabindex")) destino.setAttribute("tabindex", "-1");
      destino.focus?.({ preventScroll: true });
      destino.scrollIntoView?.({ behavior: "smooth", block: "start" });
      return;
    }
    intentos += 1;
    if (intentos < 20) globalThis.setTimeout?.(buscar, 25);
  };
  buscar();
}

/**
 * Ayudante puramente local: enlaza una orientación con superficies existentes,
 * sin consultas, almacenamiento ni acciones de negocio.
 */
export function crearAyudanteTramites({ escapar = escaparHTML, tramites = TRAMITES_AYUDANTE_PORTAL } = {}) {
  if (typeof escapar !== "function") throw new TypeError("escapador del ayudante no disponible");
  const catalogo = validarTramites(tramites);
  return {
    titulo: TEXTO.titulo,
    contenido: renderizarLista(catalogo, escapar),
    instalar({ contenedor, documento = globalThis.document, navegar = () => {}, anunciar = () => {}, cerrar = () => {} } = {}) {
      if (!contenedor?.addEventListener || !documento?.querySelector || typeof navegar !== "function") return () => {};
      let tramite = null;
      let paso = 0;
      let detalle = false;
      const pintar = (selectorFoco = "[data-ayudante-tramite], [data-ayudante-ir]") => {
        contenedor.innerHTML = tramite ? renderizarPaso(tramite, paso, escapar, detalle) : renderizarLista(catalogo, escapar);
        contenedor.querySelector(selectorFoco)?.focus?.({ preventScroll: true });
      };
      const alPulsar = (evento) => {
        const boton = evento.target?.closest?.("[data-ayudante-tramite], [data-ayudante-anterior], [data-ayudante-siguiente], [data-ayudante-detalle], [data-ayudante-volver], [data-ayudante-ir]");
        if (!boton || boton.disabled) return;
        if (boton.dataset.ayudanteTramite) {
          tramite = tramitePorId(catalogo, boton.dataset.ayudanteTramite); paso = 0; detalle = false; pintar(); return;
        }
        if (!tramite) return;
        if (boton.hasAttribute("data-ayudante-anterior")) { paso -= 1; detalle = false; pintar(); return; }
        if (boton.hasAttribute("data-ayudante-siguiente")) { paso += 1; detalle = false; pintar(); return; }
        if (boton.hasAttribute("data-ayudante-detalle")) { detalle = !detalle; pintar("[data-ayudante-detalle]"); return; }
        if (boton.hasAttribute("data-ayudante-volver")) { tramite = null; paso = 0; detalle = false; pintar(); return; }
        if (boton.hasAttribute("data-ayudante-ir")) {
          const pasoActual = resolverPaso(tramite, paso);
          cerrar();
          navegar(pasoActual.vista, { enfocar: false });
          globalThis.setTimeout?.(() => enfocarDestino(documento, pasoActual.selector, pasoActual.activar), 0);
          anunciar(`Guía abierta: ${pasoActual.titulo}. ${pasoActual.limite}`);
        }
      };
      contenedor.addEventListener("click", alPulsar);
      contenedor.querySelector("[data-ayudante-tramite]")?.focus?.({ preventScroll: true });
      return () => contenedor.removeEventListener("click", alPulsar);
    },
  };
}

export { TEXTO as MENSAJES_AYUDANTE_TRAMITES_ES };
