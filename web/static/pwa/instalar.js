import { cargarTextos, urlCatalogo } from "../comun/textos.js";
import { IDIOMA_ACTUAL } from "../comun/idioma.js";

const VERSION = "20261003-pwa-ci-v5";
const VERSION_MANIFIESTO = "20261002-pwa-v1";

export async function iniciarPWA({ documento = globalThis.document, ventana = globalThis.window, navegador = globalThis.navigator, idioma = IDIOMA_ACTUAL } = {}) {
  const navegacion = documento?.querySelector?.("[data-pwa-navegacion]");
  const manifiesto = documento?.querySelector?.("link[data-pwa-manifest]");
  if (!navegacion || !manifiesto || !ventana) return;

  const scope = navegacion.dataset.pwaScope;
  const portal = manifiesto.dataset.pwaManifest;
  if (!scope || !/^\/[a-z0-9-]+(?:\/[a-z0-9-]+)?\/$/u.test(scope) ||
      !/^pwa-[a-z0-9-]+$/u.test(portal) || !ventana.location.pathname.startsWith(scope)) return;

  const textos = await cargarTextos("pwa", { idioma });
  const urlManifiesto = urlCatalogo(textos.idioma, portal, new URL("/textos/", ventana.location.origin));
  urlManifiesto.searchParams.set("v", VERSION_MANIFIESTO);
  manifiesto.href = urlManifiesto.pathname + urlManifiesto.search;
  const inicio = `${scope}?lang=${encodeURIComponent(textos.idioma)}`;

  const instalada = ventana.matchMedia?.("(display-mode: standalone)");
  const actualizarVisibilidad = () => {
    navegacion.hidden = !instalada?.matches;
    documento.body?.classList?.toggle("pwa-instalada", Boolean(instalada?.matches));
  };
  actualizarVisibilidad();
  instalada?.addEventListener?.("change", actualizarVisibilidad);
  navegacion.setAttribute("aria-label", textos.traducir("general.navegacion_instalada"));
  for (const boton of navegacion.querySelectorAll("button[data-pwa-accion]")) {
    const accion = boton.dataset.pwaAccion;
    if (!["volver", "inicio", "recargar"].includes(accion)) continue;
    const etiqueta = textos.traducir(`general.${accion}`);
    boton.textContent = etiqueta;
    boton.setAttribute("aria-label", etiqueta);
    boton.addEventListener("click", () => {
      if (accion === "volver") {
        if (ventana.history.length > 1) ventana.history.back();
        else ventana.location.assign(inicio);
      } else if (accion === "inicio") ventana.location.assign(inicio);
      else ventana.location.reload();
    });
  }

  if (navegador?.serviceWorker && ventana.isSecureContext) {
    try {
      await navegador.serviceWorker.register(`${scope}sw.js?v=${VERSION}`, {
        scope, updateViaCache: "none",
      });
    } catch {
      // La página sigue operativa en red si el navegador impide instalar el worker.
    }
  }
}

if (globalThis.window?.document) void iniciarPWA().catch(() => {});
