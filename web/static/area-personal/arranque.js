import { esModoPresentacion } from "./contrato.js";
import { iniciarAreaPersonal } from "./aplicacion.js?v=20260924-f2-b15-area-v4";
import { iniciarI18nAreaPersonal, traducir } from "./i18n.js";

await iniciarI18nAreaPersonal();

async function resolverCliente() {
  const presentacion = esModoPresentacion(new URLSearchParams(window.location.search));
  if (presentacion) {
    const { crearAdaptadorPresentacion } = await import("./adaptador-presentacion.js?v=20260720-pulido-escritorio-v2");
    const { crearDescargadorRecibosPresentacion } = await import("../portal-empleado/documentos/descarga-recibos-presentacion.js?v=20260720-pulido-escritorio-v2");
    return {
      cliente: crearAdaptadorPresentacion(),
      descargarReciboPDF: crearDescargadorRecibosPresentacion(window),
      presentacionSolicitada: true,
    };
  }
  const { crearClienteHTTPAreaPersonal } = await import("./cliente-http.js?v=20260924-f2-b11-v2");
  return { cliente: crearClienteHTTPAreaPersonal(), presentacionSolicitada: false };
}

try {
  const dependencias = await resolverCliente();
  await iniciarAreaPersonal(dependencias);
  if (dependencias.presentacionSolicitada) {
    const selector = await import("../presentacion/selector-perfiles.js?v=20260720-selector-perfiles-v1");
    selector.instalarSelectorPerfilesPresentacion({
      disparador: document.querySelector(".sesion-usuario"),
      perfilActivo: "usuario_externo",
    });
  }
} catch {
  const carga = document.getElementById("estado-carga");
  if (carga) {
    carga.className = "estado-error";
    const titulo = document.createElement("h2");
    titulo.textContent = traducir("areaPersonal.estado.error.titulo");
    const detalle = document.createElement("p");
    detalle.textContent = traducir("areaPersonal.estado.error.detalle");
    const garantia = document.createElement("p");
    garantia.textContent = traducir("areaPersonal.estado.error.carga.garantia");
    carga.replaceChildren(titulo, detalle, garantia);
  }
}
