import { cargarTextos, reintentarTextos } from "../../../../comun/textos.js";
import { crearClienteCategorias } from "../../../categorias-rpt/cliente.js?v=20261002-ct-fin-moad-v1";
import { crearClienteCargaConvoca } from "./cliente.js?v=20261008-b1-correctivo-v1";
import { montarVistaCargaConvoca } from "./vista.js?v=20261008-b1-correctivo-v1";

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
      // El aviso común del portal tampoco se pudo leer; conserva la pantalla sin datos.
    }
  } finally {
    boton.disabled = false;
  }
}

document.getElementById("arranque-reintentar").addEventListener("click", () => void iniciar(true));
void iniciar();
