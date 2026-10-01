import { cargarTextos } from "../../comun/textos.js";
import { resolverIdiomaNavegacion } from "../../comun/idioma.js";

/** El enlace nace exclusivamente de una ficha pública validada por el consumidor. */
export function actualizarEntrada(enlace, textos, idioma) {
  const identificador = enlace.dataset.convocatoria ?? "";
  if (enlace.dataset.demostracion !== "false" || !/^[a-z0-9][a-z0-9-]{2,79}$/u.test(identificador)) {
    enlace.hidden = true;
    enlace.removeAttribute("href");
    return;
  }
  const parametros = new URLSearchParams({ convocatoria: identificador, lang: idioma });
  enlace.href = `/bolsa/preparacion/?${parametros}`;
  enlace.textContent = textos.traducir("pagina.titulo_pantalla");
  enlace.hidden = false;
}

if (typeof document !== "undefined") {
  const enlace = document.getElementById("preparar-solicitud");
  if (enlace) {
    const idioma = resolverIdiomaNavegacion();
    cargarTextos("convoca-preparacion", { idioma }).then((textos) => {
      const actualizar = () => actualizarEntrada(enlace, textos, idioma);
      actualizar();
      const observador = new MutationObserver(actualizar);
      const observar = () => observador.observe(enlace, {
        attributes: true, attributeFilter: ["data-convocatoria", "data-demostracion"],
      });
      observar();
      window.addEventListener("pagehide", () => observador.disconnect());
      window.addEventListener("pageshow", (evento) => {
        if (evento.persisted) {
          actualizar();
          observar();
        }
      });
    }).catch((error) => {
      enlace.hidden = true;
      console.error(error);
    });
  }
}
