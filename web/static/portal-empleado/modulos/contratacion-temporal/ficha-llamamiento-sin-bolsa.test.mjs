import assert from "node:assert/strict";
import test from "node:test";
import { renderizarExpediente } from "./componentes-expedientes.js";
import { crearTraductorExpedientesContratacion, cargarMensajesExpedientesContratacionEnIdioma } from "./i18n-expedientes.js";

const expediente = {
  expediente_ref: "expediente:sintetico:42", numero_visible: "2026/CT-00042", version: 3,
  fases: [], tareas: [], hitos: [], cabecera: [],
  datos_peticion: { categoria_ref: "categoria:rpt:auxiliar", periodo: { inicio: "2026-10-20" } },
};
const estado = (cambios = {}) => ({ carga: "listo", tarea_ref: "", expediente: { ...expediente, ...cambios } });
const t = crearTraductorExpedientesContratacion();
const renderizar = (ficha, resolver, traducir = t) => renderizarExpediente(
  ficha, traducir, "es-ES", "Europe/Madrid", false, resolver,
);

test("ausencia confirmada por Bolsa explica cómo continuar sin abrir un llamamiento", () => {
  const llamadas = [];
  const html = renderizar(estado(), (ref, opciones) => {
    llamadas.push([ref, opciones]);
    return { estado: "sin_bolsa" };
  });
  assert.deepEqual(llamadas, [["", { categoriaRef: "categoria:rpt:auxiliar" }]]);
  assert.match(html, /role="status"[^>]*><div class="cuerpo-panel"><p>No hay bolsa vigente para esta categoría\. Revise la vía de cobertura del expediente\./u);
  assert.doesNotMatch(html, /data-accion="ver-bolsa"|data-origen-/u);
});

test("carga fallida, permiso ausente o categoría desconocida no se confunden con ausencia", () => {
  for (const resolver of [null, () => null]) {
    const html = renderizar(estado(), resolver);
    assert.doesNotMatch(html, /No hay bolsa vigente|data-accion="ver-bolsa"/u);
  }
  const sinCategoria = renderizar(estado({ datos_peticion: {} }), () => ({ estado: "sin_bolsa" }));
  assert.doesNotMatch(sinCategoria, /No hay bolsa vigente/u);
  const bolsaNoVisible = renderizar(estado({ cabecera: [{ clave: "bolsa_cobertura", valor: "bolsa:oculta" }] }),
    (ref) => ref ? null : { estado: "sin_bolsa" });
  assert.doesNotMatch(bolsaNoVisible, /No hay bolsa vigente|data-accion="ver-bolsa"/u);
});

test("la traducción inglesa del aviso procede del catálogo de la ficha", async () => {
  const mensajes = await cargarMensajesExpedientesContratacionEnIdioma("en");
  const html = renderizar(estado(), () => ({ estado: "sin_bolsa" }),
    crearTraductorExpedientesContratacion(mensajes));
  assert.match(html, /There is no active job pool for this category\. Review the case file&#039;s recruitment route\./u);
});

test("error y denegación de Bolsa no afirman ausencia; solo el error permite reintentar", () => {
  const error = renderizar(estado(), () => ({ estado: "error" }));
  assert.match(error, /No se pudo comprobar si hay una bolsa vigente para esta categoría/u);
  assert.match(error, /data-ct-bolsa-reintentar/u);
  assert.doesNotMatch(error, /No hay bolsa vigente|data-accion="ver-bolsa"/u);
  const denegado = renderizar(estado(), () => ({ estado: "denegado" }));
  assert.match(denegado, /No tiene permiso para consultar las bolsas de esta categoría/u);
  assert.doesNotMatch(denegado, /No hay bolsa vigente|data-ct-bolsa-reintentar|data-accion="ver-bolsa"/u);
  const cobertura = renderizar(estado({ cabecera: [{ clave: "bolsa_cobertura", valor: "bolsa:oculta" }] }),
    () => ({ estado: "error" }));
  assert.doesNotMatch(cobertura, /data-bolsa-ref="bolsa:oculta"/u);
});

test("la fecha civil del detalle CT llega al asistente de Bolsa sin perderse", () => {
  const ficha = estado({ datos_peticion: { categoria_ref: "categoria:rpt:auxiliar",
    periodo: { inicio: "2026-10-20T00:00:00Z" } } });
  const html = renderizar(ficha, () => ({ categoria: "Auxiliar", bolsa_ref: "bolsa:sintetica:1" }));
  assert.match(html, /data-origen-inicio="2026-10-20"/u);
});
