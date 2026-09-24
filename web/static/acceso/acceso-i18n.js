const CATALOGOS_EMPAQUETADOS = Object.freeze({
  es: "/acceso/locales/es.json",
});

const ATRIBUTOS_ADMITIDOS = new Set(["content", "aria-label"]);

function idiomaBase(valor) {
  return String(valor ?? "").trim().toLowerCase().split("-", 1)[0];
}

export function seleccionarIdiomaAcceso(preferidos = []) {
  for (const preferido of preferidos) {
    const idioma = idiomaBase(preferido);
    if (Object.hasOwn(CATALOGOS_EMPAQUETADOS, idioma)) return idioma;
  }
  return "es";
}

export function rutaCatalogoAcceso(preferidos = []) {
  return CATALOGOS_EMPAQUETADOS[seleccionarIdiomaAcceso(preferidos)];
}

function esCatalogo(valor) {
  return valor !== null && typeof valor === "object" && !Array.isArray(valor);
}

export function aplicarCatalogoAcceso(documento, catalogo) {
  if (!documento?.querySelectorAll || !esCatalogo(catalogo)) return;
  documento.querySelectorAll("[data-i18n]").forEach((elemento) => {
    const clave = elemento.getAttribute("data-i18n");
    if (typeof catalogo[clave] === "string") elemento.textContent = catalogo[clave];
  });
  documento.querySelectorAll("[data-i18n-atributo]").forEach((elemento) => {
    const [atributo, clave, extra] = String(elemento.getAttribute("data-i18n-atributo")).split(":");
    if (extra || !ATRIBUTOS_ADMITIDOS.has(atributo) || typeof catalogo[clave] !== "string") return;
    elemento.setAttribute(atributo, catalogo[clave]);
  });
}

export async function iniciarI18nAcceso(documento = document, fetcher = fetch, idiomasNavegador = navigator.languages) {
  const idioma = seleccionarIdiomaAcceso([documento?.documentElement?.lang, ...(idiomasNavegador ?? [])]);
  try {
    const respuesta = await fetcher(rutaCatalogoAcceso([idioma]), { credentials: "omit" });
    if (!respuesta?.ok) return idioma;
    const catalogo = await respuesta.json();
    aplicarCatalogoAcceso(documento, catalogo);
  } catch {
    // El HTML empaquetado ya contiene el español de respaldo.
  }
  return idioma;
}

export function montarAyudaAcceso(documento = globalThis.document) {
  const boton = documento?.getElementById?.("boton-ayuda-acceso");
  const contenido = documento?.getElementById?.("acceso-autorizacion");
  if (!boton || !contenido || boton.getAttribute("aria-controls") !== contenido.id) return false;

  contenido.hidden = true;
  boton.setAttribute("aria-expanded", "false");
  boton.addEventListener("click", () => {
    const abrir = contenido.hidden;
    contenido.hidden = !abrir;
    boton.setAttribute("aria-expanded", String(abrir));
    if (abrir) contenido.focus();
    else boton.focus();
  });
  documento.addEventListener("keydown", (evento) => {
    if (evento.key !== "Escape" || contenido.hidden) return;
    evento.preventDefault();
    contenido.hidden = true;
    boton.setAttribute("aria-expanded", "false");
    boton.focus();
  });
  return true;
}

if (typeof document !== "undefined") {
  montarAyudaAcceso();
  if (typeof fetch === "function") void iniciarI18nAcceso();
}
