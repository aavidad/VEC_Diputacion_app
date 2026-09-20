import { TRAMITES_AYUDANTE_PORTAL } from "./ayuda-contenido.js";

const TEXTO = Object.freeze({
  titulo: "Ayudante de trámites",
  introduccion: "Elija un trámite frecuente. La guía le lleva a la sección correspondiente y explica cada paso; no realiza trámites ni sustituye una decisión administrativa.",
  elegir: "Trámites frecuentes",
  pasos: "Pasos del trámite",
  anterior: "Anterior",
  siguiente: "Siguiente",
  ir: "Ir al paso en la aplicación",
  volver: "Volver a trámites",
  detalle: "Explicar este paso",
  ocultar: "Ocultar explicación",
  objetivo: "Objetivo",
  preparacion: "Antes de empezar",
  resultado: "Resultado esperado",
  actor: "Responsable",
  limite: "Límite o dependencia",
  aviso: "La guía no guarda datos, no solicita documentos y no produce efectos administrativos.",
  pendiente: "Este paso queda detenido hasta que la dependencia indicada esté conectada y autorizada.",
});

function escaparHTML(valor) {
  return String(valor).replaceAll("&", "&amp;").replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
}

function tramitePorId(tramites, id) {
  return tramites.find((tramite) => tramite.id === id) || null;
}

function resolverPaso(tramite, indice) {
  const paso = tramite.pasos[indice];
  return Object.freeze({
    ...paso,
    selector: paso.selector || tramite.selector,
    activar: paso.activar || tramite.activar,
    bloqueado: paso.bloqueado ?? true,
  });
}

function renderizarLista(tramites, escapar) {
  return `<section class="ayudante-tramites" data-ayudante-tramites aria-labelledby="ayudante-tramites-titulo">
    <p class="ayudante-tramites-introduccion">${escapar(TEXTO.introduccion)}</p>
    <h3 id="ayudante-tramites-titulo">${escapar(TEXTO.elegir)}</h3>
    <ul class="ayudante-tramites-lista">${tramites.map((tramite) => `<li><button type="button" class="ayudante-tramite" data-ayudante-tramite="${escapar(tramite.id)}"><span>${escapar(tramite.titulo)}</span><small>${escapar(tramite.modulo)} · ${escapar(tramite.resumen)}</small></button></li>`).join("")}</ul>
    <p class="ayudante-tramites-extension">Bolsa de trabajo y Contratación temporal mantienen su ayuda propia; su ampliación corresponde al recorrido de sus módulos.</p>
    <p class="ayudante-tramites-aviso">${escapar(TEXTO.aviso)}</p>
  </section>`;
}

function renderizarPaso(tramite, indice, escapar, expandido = false) {
  const paso = resolverPaso(tramite, indice);
  const detalleId = `ayudante-detalle-${tramite.id}-${indice}`;
  return `<section class="ayudante-tramites" data-ayudante-tramites aria-labelledby="ayudante-paso-titulo">
    <p class="sobrelinea">${escapar(tramite.modulo)} · ${escapar(`Paso ${indice + 1} de ${tramite.pasos.length}`)}</p>
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
  if (!Array.isArray(tramites) || tramites.length === 0) throw new TypeError("catálogo del ayudante no disponible");
  return {
    titulo: TEXTO.titulo,
    contenido: renderizarLista(tramites, escapar),
    instalar({ contenedor, documento = globalThis.document, navegar = () => {}, anunciar = () => {}, cerrar = () => {} } = {}) {
      if (!contenedor?.addEventListener || !documento?.querySelector || typeof navegar !== "function") return () => {};
      let tramite = null;
      let paso = 0;
      let detalle = false;
      const pintar = () => {
        contenedor.innerHTML = tramite ? renderizarPaso(tramite, paso, escapar, detalle) : renderizarLista(tramites, escapar);
        contenedor.querySelector("[data-ayudante-tramite], [data-ayudante-ir]")?.focus?.({ preventScroll: true });
      };
      const alPulsar = (evento) => {
        const boton = evento.target?.closest?.("[data-ayudante-tramite], [data-ayudante-anterior], [data-ayudante-siguiente], [data-ayudante-detalle], [data-ayudante-volver], [data-ayudante-ir]");
        if (!boton || boton.disabled) return;
        if (boton.dataset.ayudanteTramite) {
          tramite = tramitePorId(tramites, boton.dataset.ayudanteTramite); paso = 0; detalle = false; pintar(); return;
        }
        if (!tramite) return;
        if (boton.hasAttribute("data-ayudante-anterior")) { paso -= 1; detalle = false; pintar(); return; }
        if (boton.hasAttribute("data-ayudante-siguiente")) { paso += 1; detalle = false; pintar(); return; }
        if (boton.hasAttribute("data-ayudante-detalle")) { detalle = !detalle; pintar(); return; }
        if (boton.hasAttribute("data-ayudante-volver")) { tramite = null; paso = 0; detalle = false; pintar(); return; }
        if (boton.hasAttribute("data-ayudante-ir")) {
          const pasoActual = resolverPaso(tramite, paso);
          cerrar();
          navegar(tramite.vista, { enfocar: false });
          globalThis.setTimeout?.(() => enfocarDestino(documento, pasoActual.selector, pasoActual.activar), 0);
          anunciar(`Guía abierta: ${pasoActual.titulo}. ${pasoActual.limite}`);
        }
      };
      contenedor.addEventListener("click", alPulsar);
      return () => contenedor.removeEventListener("click", alPulsar);
    },
  };
}

export { TEXTO as MENSAJES_AYUDANTE_TRAMITES_ES };
