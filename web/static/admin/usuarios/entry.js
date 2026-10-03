import { cargarTextos } from "../../comun/textos.js";
import { montarUsuarios } from "./vista.js?v=20261003-admin-usuarios-v2";
import { crearClienteLecturasUsuarios } from "./lecturas-http.js?v=20261003-admin-usuarios-v2";
const root = document.getElementById("usuarios-contenido");
let montaje;
try {
  const textos = await cargarTextos("admin-usuarios");
  document.documentElement.lang = textos.idioma;
  document.title = textos.traducir("general.titulo");
  for (const nodo of document.querySelectorAll("[data-texto]")) nodo.textContent = textos.traducir(nodo.dataset.texto);
  for (const nodo of document.querySelectorAll("[data-etiqueta]")) nodo.setAttribute("aria-label", textos.traducir(nodo.dataset.etiqueta));
  montaje = montarUsuarios(root, { textos, cliente: crearClienteLecturasUsuarios() });
} catch {
  // Sin catálogo no se inventa idioma ni se abre ningún puerto administrativo.
  root?.replaceChildren();
}
window.addEventListener("pagehide", () => montaje?.desmontar(), { once: true });
