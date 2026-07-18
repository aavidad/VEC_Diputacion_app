import { esModoPresentacion } from "./contrato.js";
import { iniciarAreaPersonal } from "./aplicacion.js";

async function resolverCliente() {
  const presentacion = esModoPresentacion(new URLSearchParams(window.location.search));
  if (presentacion) {
    const { crearAdaptadorPresentacion } = await import("./adaptador-presentacion.js?v=20260718-1");
    return { cliente: crearAdaptadorPresentacion(), presentacionSolicitada: true };
  }
  const { crearClienteHTTPAreaPersonal } = await import("./cliente-http.js?v=20260718-1");
  return { cliente: crearClienteHTTPAreaPersonal(), presentacionSolicitada: false };
}

try {
  const dependencias = await resolverCliente();
  await iniciarAreaPersonal(dependencias);
} catch (error) {
  const carga = document.getElementById("estado-carga");
  if (carga) {
    carga.className = "estado-error";
    const titulo = document.createElement("h2");
    titulo.textContent = "No se pudo iniciar el área personal";
    const detalle = document.createElement("p");
    detalle.textContent = error instanceof Error ? error.message : "Error de inicialización.";
    const garantia = document.createElement("p");
    garantia.textContent = "No se ha realizado ninguna operación.";
    carga.replaceChildren(titulo, detalle, garantia);
  }
}
