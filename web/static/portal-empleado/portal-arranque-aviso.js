/* Aviso de arranque independiente del grafo de módulos del portal. */
import { IDIOMA_ACTUAL } from "../comun/idioma.js";
import { cargarTextos } from "../comun/textos.js";
(() => {
  const raiz = document.getElementById("espacio-trabajo");
  if (!raiz || raiz.childElementCount > 0) return;

  const aviso = document.createElement("section");
  aviso.className = "panel";
  aviso.setAttribute("role", "status");
  aviso.setAttribute("aria-live", "polite");
  aviso.dataset.portalArranqueAviso = "";
  const titulo = document.createElement("h2");
  const detalle = document.createElement("p");
  const reintentar = document.createElement("button");
  reintentar.type = "button";
  reintentar.className = "boton-primario";
  reintentar.hidden = true;
  reintentar.addEventListener("click", () => location.reload());
  aviso.append(titulo, detalle, reintentar);
  raiz.append(aviso);

  let fallo = false;
  let textos = null;
  let plazo;
  const observador = new MutationObserver(() => {
    if (raiz.contains(aviso)) return;
    clearTimeout(plazo);
    observador.disconnect();
  });
  observador.observe(raiz, { childList: true });

  function pintar() {
    if (!textos || !raiz.contains(aviso)) return;
    titulo.textContent = fallo ? textos.tituloError : textos.titulo;
    detalle.textContent = fallo ? textos.error : textos.cargando;
    reintentar.textContent = textos.reintentar;
    reintentar.hidden = !fallo;
  }

  function mostrarError(codigo) {
    if (fallo || !raiz.contains(aviso)) return;
    fallo = true;
    // Los fallos de importación pueden incluir URL o detalles de la respuesta.
    // Solo se registra un código fijo, nunca la excepción ni la sesión.
    console.error("portal.arranque.fallido", { codigo });
    pintar();
  }

  async function cargarTextosArranque() {
    const catalogo = await cargarTextos("portal", { idioma: IDIOMA_ACTUAL });
    const general = catalogo.seccion("general");
    const comunes = catalogo.seccion("textos");
    const valores = {
      titulo: comunes?.txt_portal_del_empleado,
      cargando: comunes?.txt_comprobando_acceso,
      tituloError: general?.vista_no_disponible_titulo,
      error: general?.portal_arranque_error,
      reintentar: comunes?.txt_reintentar,
    };
    if (Object.values(valores).some((valor) => typeof valor !== "string" || !valor)) throw new Error("textos incompletos");
    document.documentElement.lang = catalogo.idioma;
    textos = valores;
    pintar();
    // Si el grafo principal falla, el shell conserva sus rótulos estáticos.
    // El traductor común puede completarlos sin depender del coordinador.
    try {
      const idioma = await import("./portal-idioma.js?v=20261001-ct-a-i18n-v1");
      if (raiz.contains(aviso)) idioma.aplicarTextosPortal(document);
    } catch { /* El aviso del catálogo común ya queda visible. */ }
  }

  void cargarTextosArranque().catch(() => mostrarError("catalogo_arranque_no_disponible"));
  window.addEventListener("error", (evento) => {
    const script = evento.target;
    if (script?.tagName !== "SCRIPT" || script.type !== "module") return;
    if (new URL(script.src).pathname === "/portal-empleado/portal.js") mostrarError("importacion_portal_no_disponible");
  }, true);
  plazo = setTimeout(() => mostrarError("arranque_sin_vista"), 12_000);
})();
