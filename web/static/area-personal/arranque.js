import { exigirParametrosConocidos, iniciarAreaPersonal } from "./aplicacion.js?v=20260926-portal-candidato-v1";
import { iniciarI18nAreaPersonal, traducir } from "./i18n.js";

await iniciarI18nAreaPersonal();

async function resolverCliente() {
  exigirParametrosConocidos(new URLSearchParams(window.location.search));
  const { crearClienteHTTPAreaPersonal } = await import("./cliente-http.js?v=20260926-portal-candidato-v1");
  return { cliente: crearClienteHTTPAreaPersonal() };
}

try {
  const dependencias = await resolverCliente();
  await iniciarAreaPersonal(dependencias);
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
