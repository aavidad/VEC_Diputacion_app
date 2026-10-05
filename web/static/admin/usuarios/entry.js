import { cargarTextos } from "../../comun/textos.js";
import { montarUsuarios } from "./vista.js?v=20261005-admin-lote-pantalla-v1";
import { crearClienteLecturasUsuarios, crearClienteActosUsuarios } from "./lecturas-http.js?v=20261005-admin-lote-pantalla-v1";
import { cargarTextosSelectorPerfil, crearClienteSelectorPerfil, montarSelectorPerfil } from "/administracion-perfiles/selector-perfil.js?v=20261005-admin-selector-auditoria-v3";
const root = document.getElementById("usuarios-contenido");
const panelSelector = document.getElementById("usuarios-selector-panel");
let montaje, selector;
try {
  const [textos, textosSelector] = await Promise.all([cargarTextos("admin-usuarios"), cargarTextosSelectorPerfil()]);
  document.documentElement.lang = textos.idioma;
  document.title = textos.traducir("general.titulo");
  for (const nodo of document.querySelectorAll("[data-texto]")) nodo.textContent = textos.traducir(nodo.dataset.texto);
  for (const nodo of document.querySelectorAll("[data-etiqueta]")) nodo.setAttribute("aria-label", textos.traducir(nodo.dataset.etiqueta));
  document.getElementById("usuarios-selector-resumen").textContent = textosSelector.traducir("general.titulo");
  selector = montarSelectorPerfil({ contenedor: document.getElementById("usuarios-selector"), textos: textosSelector,
    cliente: crearClienteSelectorPerfil(), limpiarEstado() {
      montaje?.desmontar(); montaje = null; root.hidden = true; root.replaceChildren(); panelSelector.open = true;
    }, async cargarContexto({ signal }) {
      if (signal.aborted) return;
      root.hidden = false;
      // Lecturas de usuarios y, si el servidor monta el lote, preparar y aplicar.
      const lecturas = crearClienteLecturasUsuarios({ proyeccion: "metadatos_v1" });
      const cliente = Object.freeze({ ...lecturas, aplicarLote: crearClienteActosUsuarios().aplicarLote });
      const actual = montarUsuarios(root, { textos, cliente }); montaje = actual;
      signal.addEventListener("abort", () => actual.desmontar(), { once: true });
      await actual.listo;
      if (!signal.aborted && montaje === actual) { panelSelector.open = false; root.focus(); }
    } });
  for (const boton of document.getElementById("usuarios-selector").querySelectorAll("button")) boton.classList.add(boton.type === "submit" ? "boton-primario" : "boton-secundario");
} catch {
  // Sin catálogo no se inventa idioma ni se abre ningún puerto administrativo.
  montaje?.desmontar(); selector?.destruir(); root?.replaceChildren();
}
window.addEventListener("pagehide", () => { montaje?.desmontar(); selector?.destruir(); });
