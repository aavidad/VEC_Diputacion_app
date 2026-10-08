import { cargarTextos } from "../../comun/textos.js";
import { prepararTextosPersonal } from "../modulos/personal/i18n.js?v=20261008-alta-rpt-circular-v4";

const URL_ORGANIZACION = "./organizacion.js?v=20261007-pantallas-textos-final-v1";
const registrarFallo = (error) => globalThis.console?.error?.("organizacion.arranque.fallido", {
  codigo: error.codigo,
  causa: error.cause?.name,
});

function fallo(codigo, causa) {
  const error = new Error(codigo, { cause: causa });
  error.codigo = codigo;
  return error;
}

export function crearArranqueOrganizacion({
  documento = globalThis.document,
  preparar = prepararTextosPersonal,
  importar = () => import(URL_ORGANIZACION),
  cargarAviso = () => cargarTextos("portal"),
  registrarError = registrarFallo,
  recargar = () => globalThis.location?.reload?.(),
} = {}) {
  const raiz = documento?.getElementById?.("organizacion");
  if (!raiz || typeof preparar !== "function" || typeof importar !== "function"
    || typeof cargarAviso !== "function" || typeof registrarError !== "function") {
    throw new TypeError("arranque de Organización no disponible");
  }
  let ocupado = false;
  let iniciado = false;
  let tipoFallo = null;
  let aviso;
  let detalle;
  let boton;
  const originales = Array.from(raiz.children);
  const visibilidadOriginal = new Map(originales.map((nodo) => [nodo, nodo.hidden]));
  const ocultarOriginales = () => { for (const nodo of originales) nodo.hidden = true; };
  const restaurarOriginales = () => {
    for (const nodo of originales) nodo.hidden = visibilidadOriginal.get(nodo);
  };

  async function presentarError() {
    const textos = await cargarAviso();
    const t = (clave) => textos.traducir(`general.organizacion_arranque_${clave}`);
    documento.documentElement.lang = textos.idioma;
    documento.title = t("titulo");
    if (!aviso) {
      aviso = documento.createElement("section");
      aviso.className = "org-panel";
      aviso.dataset.organizacionArranqueError = "";
      const titulo = documento.createElement("h2");
      const cuerpo = documento.createElement("div");
      cuerpo.className = "org-state error";
      detalle = documento.createElement("p");
      boton = documento.createElement("button");
      boton.type = "button";
      boton.className = "org-secondary";
      boton.addEventListener("click", () => {
        if (ocupado || iniciado) return;
        if (tipoFallo === "importacion") recargar();
        else void iniciar().catch(registrarError);
      });
      cuerpo.append(detalle, boton);
      aviso.append(titulo, cuerpo);
      raiz.prepend(aviso);
    }
    aviso.setAttribute("role", "alert");
    aviso.querySelector("h2").textContent = t("titulo");
    detalle.textContent = t("error");
    boton.textContent = t("reintentar");
    boton.disabled = false;
    boton.focus?.();
  }

  async function iniciar() {
    if (ocupado || iniciado) return false;
    ocupado = true;
    raiz.setAttribute("aria-busy", "true");
    if (!aviso) raiz.inert = true;
    else {
      aviso.setAttribute("role", "status");
      boton.disabled = true;
      detalle.setAttribute("tabindex", "-1");
      detalle.focus?.();
    }
    let fase = "preparacion";
    try {
      const textos = await preparar();
      documento.documentElement.lang = textos.idioma;
      restaurarOriginales();
      fase = "importacion";
      await importar();
      iniciado = true;
      const devolverFoco = aviso && (documento.activeElement === detalle || documento.activeElement === boton);
      aviso?.remove();
      if (devolverFoco) raiz.focus?.();
      return true;
    } catch (causa) {
      tipoFallo = fase;
      const error = fallo(`organizacion.arranque.${fase}_fallida`, causa);
      registrarError(error);
      ocultarOriginales();
      raiz.inert = false;
      try { await presentarError(); }
      catch (causaAviso) {
        const completo = fallo("organizacion.arranque.aviso_no_disponible", new AggregateError([error, causaAviso]));
        registrarError(completo);
        throw completo;
      }
      return false;
    } finally {
      ocupado = false;
      raiz.inert = false;
      raiz.setAttribute("aria-busy", "false");
    }
  }

  return Object.freeze({ iniciar });
}

if (typeof document !== "undefined") {
  void crearArranqueOrganizacion().iniciar().catch(registrarFallo);
}
