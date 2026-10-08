import { cargarTextos, reintentarTextos } from "../../../../comun/textos.js";
import { crearClienteCategorias } from "../../../categorias-rpt/cliente.js?v=20261008-w-ct-borradores-main-v2";
import { crearClienteCargaConvoca } from "./cliente.js?v=20261008-u-b1-recibo-v2";
import { montarVistaCargaConvoca } from "./vista.js?v=20261008-u-b1-recibo-v2";

async function iniciar(reintento = false) {
  const error = document.getElementById("arranque-error");
  const boton = document.getElementById("arranque-reintentar");
  boton.disabled = true;
  try {
    const textos = reintento ? await reintentarTextos("bolsa-carga-convoca") : await cargarTextos("bolsa-carga-convoca");
    const vista = montarVistaCargaConvoca({
      doc: document,
      cliente: crearClienteCargaConvoca(),
      categorias: crearClienteCategorias(),
      textos,
    });
    error.hidden = true;
    document.getElementById("carga-cabecera").hidden = false;
    document.getElementById("carga-contenido").hidden = false;
    await vista.iniciar();
  } catch {
    try {
      const respaldo = await cargarTextos("portal-arranque");
      const t = (clave) => respaldo.traducir(clave);
      document.getElementById("arranque-error-titulo").textContent = t("titulo_error");
      document.getElementById("arranque-error-detalle").textContent = t("error");
      boton.textContent = t("reintentar");
      error.hidden = false;
      error.focus();
    } catch {
      try {
        const respaldo = await cargarTextos("bolsa-carga-convoca-recuperacion");
        document.getElementById("arranque-error-titulo").textContent = respaldo.traducir("general.titulo");
        document.getElementById("arranque-error-detalle").textContent = respaldo.traducir("general.detalle");
        boton.textContent = respaldo.traducir("general.reintentar");
        error.hidden = false;
        error.focus();
      } catch {
        // Sin ningún catálogo disponible, el control queda visible para recuperar la carga.
        error.hidden = false;
        error.focus();
      }
    }
  } finally {
    boton.disabled = false;
  }
}

document.getElementById("arranque-reintentar").addEventListener("click", () => void iniciar(true));
void iniciar();
