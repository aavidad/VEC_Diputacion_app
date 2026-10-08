import { cargarTextos } from "../../comun/textos.js";
import { aplicarIdiomaDocumento, aplicarTextosPortal, instalarValidacionI18n } from "../portal-idioma.js?v=20261007-pantallas-textos-final-v1";
import { traducirPortal } from "../portal-i18n.js?v=20261007-pantallas-textos-final-v1";
import { ERROR_TEXTOS_PETICIONES_CENTRO, prepararTextosPeticionesCentro } from "./i18n-peticiones-centro.js?v=20261007-pc-recuperacion-v1";

const raiz = document.querySelector("#aplicacion");
const ayuda = document.querySelector("#pc-ayuda-abrir");
let promesaArranque = null;
let montado = false;

function escapar(valor) {
  return String(valor).replaceAll("&", "&amp;").replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#039;");
}

function traductorPortal(catalogo) {
  const secciones = [catalogo.seccion("general"), catalogo.seccion("textos"), catalogo.seccion("panel_interno")];
  return (clave, variables = {}) => {
    const plantilla = secciones.find((seccion) => Object.hasOwn(seccion, clave))?.[clave];
    if (typeof plantilla !== "string") throw new Error(`clave del portal desconocida: ${clave}`);
    return plantilla.replace(/\{([a-z_]+)\}/gu, (_coincidencia, nombre) => String(variables[nombre] ?? ""));
  };
}

function puedeDevolverFoco(origen) {
  return origen && (document.activeElement === origen || document.activeElement === document.body
    || document.activeElement === document.documentElement);
}

function mostrarError(origen = null) {
  ayuda.disabled = true;
  raiz.innerHTML = `<section class="pc-panel pc-detalle" role="alert" lang="${escapar(document.documentElement.lang)}">
    <h1 tabindex="-1">${escapar(traducirPortal("txt_error_de_carga"))}</h1>
    <p>${escapar(traducirPortal("peticiones_centro_arranque_error"))}</p>
    <div class="pc-acciones"><button type="button" class="boton-secundario" data-pc-reintentar>
      ${escapar(traducirPortal("accion_reintentar"))}</button></div>
  </section>`;
  if (puedeDevolverFoco(origen)) raiz.querySelector("[data-pc-reintentar]")?.focus();
}

async function arrancar(origen = null) {
  if (montado) return;
  if (promesaArranque) return promesaArranque;
  raiz.setAttribute("aria-busy", "true");
  promesaArranque = (async () => {
    try {
      const pc = await prepararTextosPeticionesCentro();
      const portal = await cargarTextos("portal", { idioma: pc.idioma });
      if (portal.idioma !== pc.idioma) throw new Error("catálogo del portal en otro idioma");
      const traducir = traductorPortal(portal);
      aplicarTextosPortal(document, traducir);
      document.documentElement.lang = pc.idioma;
      instalarValidacionI18n(document, traducir);
      const peticiones = await import("./peticiones-centro.js?v=20261008-alta-etiquetas-ayuda-v2");
      peticiones.instalarAyudaPeticionCentro(document);
      await Promise.all([
        import("./incorporaciones-centro.js?v=20261007-pc-recuperacion-v1"),
        import("./cancelaciones-centro.js?v=20261007-pc-recuperacion-v1"),
      ]);
      ayuda.disabled = false;
      await peticiones.iniciarPeticionCentro();
      montado = true;
      if (puedeDevolverFoco(origen)) {
        const titulo = raiz.querySelector("h1, h2");
        titulo?.setAttribute("tabindex", "-1");
        titulo?.focus();
      }
    } catch (error) {
      console.error("pc.arranque.fallido", { tipo: error?.name });
      mostrarError(origen);
    }
  })().finally(() => { promesaArranque = null; raiz.setAttribute("aria-busy", "false"); });
  return promesaArranque;
}

if (raiz) {
  ayuda.disabled = true;
  aplicarTextosPortal(document);
  aplicarIdiomaDocumento(document);
  raiz.addEventListener("click", (evento) => {
    const boton = evento.target.closest?.("[data-pc-reintentar]");
    if (!boton || promesaArranque) return;
    boton.disabled = true;
    arrancar(boton);
  });
  if (ERROR_TEXTOS_PETICIONES_CENTRO) {
    console.error("pc.textos.no_disponibles", { tipo: ERROR_TEXTOS_PETICIONES_CENTRO.name });
    mostrarError();
  }
  else arrancar();
}
