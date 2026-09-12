import { traducirPortal } from "../portal-empleado/portal-i18n.js";
import { crearClienteCorreo, montarConfiguracionCorreo } from "./configuracion-correo.js";

document.title = traducirPortal("admin_correo_titulo");
const contenedor = document.getElementById("administracion");
if (contenedor) {
  const vista = montarConfiguracionCorreo(contenedor, { cliente: crearClienteCorreo(), traducir: traducirPortal });
  void vista.cargar();
  window.addEventListener("pagehide", () => vista.desmontar(), { once: true });
}
