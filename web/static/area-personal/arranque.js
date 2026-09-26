import { exigirParametrosConocidos, iniciarAreaPersonal } from "./aplicacion.js?v=20260926-convoca-f1-v1";
import { iniciarI18nAreaPersonal, traducir } from "./i18n.js";

await iniciarI18nAreaPersonal();

async function resolverCliente() {
  exigirParametrosConocidos(new URLSearchParams(window.location.search));
  const [{ crearClienteHTTPAreaPersonal }, { crearClienteSolicitudesPersona }] = await Promise.all([
    import("./cliente-http.js?v=20260925-sin-demo-v1"),
    import("./cliente-http-solicitudes.js?v=20260926-convoca-f1-v1"),
  ]);
  return { cliente: crearClienteHTTPAreaPersonal(), clienteSolicitudes: crearClienteSolicitudesPersona() };
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
