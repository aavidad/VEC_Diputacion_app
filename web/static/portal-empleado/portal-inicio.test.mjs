import assert from "node:assert/strict";
import test from "node:test";
import { crearVistaInicioPortal } from "./portal-inicio.js";

const moduloBolsa = Object.freeze({
  clave: "bolsa",
  sigla: "BOL",
  titulo: "Bolsas de trabajo",
  texto: "Gobierno de convocatorias y llamamientos.",
});

const escaparHTML = (valor) => String(valor)
  .replaceAll("&", "&amp;")
  .replaceAll("<", "&lt;")
  .replaceAll(">", "&gt;")
  .replaceAll('"', "&quot;")
  .replaceAll("'", "&#039;");

function renderizar(acceso) {
  return crearVistaInicioPortal({
    encabezadoVista: () => "<header>Portal</header>",
    escaparHTML,
    obtenerCatalogo: () => [moduloBolsa],
    resolverAcceso: () => acceso,
  })();
}

test("la tarjeta anuncia la comprobación sin ofrecer una ruta prematura", () => {
  const html = renderizar({
    disponible: false,
    vista: "",
    estado: "cargando",
    etiqueta: "Comprobando acceso a borradores",
  });
  assert.match(html, /data-modulo-catalogo="bolsa" tabindex="-1" aria-busy="true"/);
  assert.match(html, /role="status" aria-live="polite">Comprobando acceso a borradores/);
  assert.match(html, /<button[^>]+disabled>Comprobando<\/button>/);
  assert.doesNotMatch(html, /data-vista=/);
});

test("la tarjeta diferencia denegación de error técnico y solo este permite reintentar", () => {
  const denegado = renderizar({
    disponible: false,
    vista: "",
    estado: "denegado",
    etiqueta: "Sin permiso para gestionar borradores",
  });
  assert.match(denegado, /Sin permiso para gestionar borradores/);
  assert.match(denegado, /<button[^>]+disabled>Sin permiso<\/button>/);
  assert.doesNotMatch(denegado, /reintentar-borradores/);

  const error = renderizar({
    disponible: false,
    vista: "",
    estado: "error",
    etiqueta: "Servicio de borradores no disponible",
    reintentar: true,
  });
  assert.match(error, /Servicio de borradores no disponible/);
  assert.match(error, /data-accion="reintentar-borradores">Reintentar<\/button>/);
  assert.doesNotMatch(error, /data-vista=/);
});

test("la capacidad propia abre Elaboración aunque el panel agregado no participe", () => {
  const html = renderizar({
    disponible: true,
    vista: "elaboracion",
    estado: "disponible",
    etiqueta: "Borradores disponibles",
  });
  assert.match(html, /Disponible para el perfil activo/);
  assert.match(html, /data-vista="elaboracion">Entrar<\/button>/);
  assert.doesNotMatch(html, /panel|resumen/u);
});
