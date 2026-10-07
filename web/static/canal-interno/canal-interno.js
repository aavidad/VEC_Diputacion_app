import { IDIOMA_ACTUAL, montarSelectorIdioma, leerRecursoJSON } from "/comun/idioma.js";
import { cargarTextos } from "/comun/textos.js";

const RUTA_FUENTES = new URL("./destinos.json", import.meta.url);
const DESTINOS = Object.freeze({
  canal: "https://centinela.lefebvre.es/lp/canal-denuncias/diputacion-granada",
  institucional: "https://www.dipgra.es/e-administracion/administracion-electronica/sistema-interno-de-informacion/",
  estrategia: "https://bop.dipgra.es/export/sites/bop/.galleries/Documentos-Anuncios-en-PDF/firmado-1723590017898-final-7e73c306-2.pdf?p=1779215866944",
  reglamento: "https://bop.dipgra.es/export/sites/bop/.galleries/Documentos-Anuncios-en-PDF/firmado-1751497250139-final-40f10e71-1.pdf",
  ley: "https://www.boe.es/eli/es/l/2023/02/20/2/con",
});

export function validarFuentes(datos) {
  if (!datos || datos.version !== 1 || !datos.destinos ||
      Object.keys(datos.destinos).length !== Object.keys(DESTINOS).length) throw new TypeError("fuentes no válidas");
  for (const [clave, esperado] of Object.entries(DESTINOS)) {
    const registro = datos.destinos[clave];
    if (registro?.url !== esperado || !registro.organo || !registro.fuente ||
        !registro.publicacion || !registro.vigencia) throw new TypeError("destino no admitido");
  }
  return datos.destinos;
}

const porId = (id) => document.getElementById(id);
let textos;

function ponerTextos() {
  porId("ayuda-boton").hidden = false;
  document.documentElement.lang = textos.idioma;
  document.title = textos.traducir("documento");
  document.querySelectorAll("[data-t]").forEach((elemento) => {
    elemento.textContent = textos.traducir(elemento.dataset.t);
  });
  porId("ayuda-boton").setAttribute("aria-label", textos.traducir("ayuda_boton"));
  porId("ayuda-boton").title = textos.traducir("ayuda_boton");
}

function mostrarError() {
  porId("pagina").hidden = true;
  const estado = porId("estado");
  estado.replaceChildren();
  if (!textos) return;
  for (const clave of ["saltar", "marca", "idioma", "pie"]) {
    document.querySelector(`[data-t="${clave}"]`).textContent = textos.traducir(clave);
  }
  porId("ayuda-boton").hidden = true;
  porId("ayuda").hidden = true;
  porId("ayuda-boton").setAttribute("aria-expanded", "false");
  const mensaje = document.createElement("p");
  mensaje.textContent = textos.traducir("error");
  const reintentar = document.createElement("button");
  reintentar.type = "button";
  reintentar.textContent = textos.traducir("reintentar");
  reintentar.addEventListener("click", cargar, { once: true });
  estado.append(mensaje, reintentar);
}

async function cargar() {
  porId("estado").replaceChildren();
  porId("pagina").hidden = true;
  try {
    textos = await cargarTextos("canal-interno", { leer: leerRecursoJSON });
    if (textos.idioma !== IDIOMA_ACTUAL || textos.faltantes.length > 0) throw new TypeError("catálogo incompleto");
    ponerTextos();
    const fuentes = validarFuentes(await leerRecursoJSON(RUTA_FUENTES));
    Object.entries(fuentes).forEach(([clave, registro]) => { porId(clave).href = registro.url; });
    porId("pagina").hidden = false;
  } catch {
    try {
      textos = await cargarTextos("canal-interno-error", { leer: leerRecursoJSON });
      if (textos.idioma !== IDIOMA_ACTUAL || textos.faltantes.length > 0) throw new TypeError("catálogo de error incompleto");
      document.documentElement.lang = textos.idioma;
      document.title = textos.traducir("documento");
    } catch { textos = null; }
    mostrarError();
  }
}

montarSelectorIdioma(porId("idioma"));
porId("ayuda-boton").addEventListener("click", () => {
  const ayuda = porId("ayuda");
  ayuda.hidden = !ayuda.hidden;
  porId("ayuda-boton").setAttribute("aria-expanded", String(!ayuda.hidden));
  if (!ayuda.hidden) ayuda.querySelector("h2").focus();
});
cargar();
