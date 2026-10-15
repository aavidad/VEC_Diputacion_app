import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { validarContenidoEditable, validarDetalleBorrador } from "./portal-borradores-contrato.js?v=20261016-modalidad-inscripcion-v1";
import { crearEstadoBorradores } from "./portal-borradores-estado.js";
import { editorDesdeDetalle, editorNuevo, asignarRutaEditor, errorSeguro } from "./portal-borradores-ui-soporte.js?v=20261016-modalidad-inscripcion-v1";
import { crearRenderizadorBorradores } from "./portal-borradores-vista.js?v=20261016-modalidad-inscripcion-v1";
import { contenido, detalle, limites, lista, opciones } from "./portal-borradores-fixtures.test-helper.mjs";

const escaparHTML = (valor) => String(valor ?? "").replaceAll("&", "&amp;")
  .replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;");

test("la modalidad empieza sin elegir y se conserva al reabrir el borrador", () => {
  const nuevo = editorNuevo(opciones());
  assert.equal(nuevo.contenido_editable.modalidad_inscripcion, "");
  const historico = validarDetalleBorrador(detalle(), limites());
  assert.equal(editorDesdeDetalle(historico).contenido_editable.modalidad_inscripcion, "");

  asignarRutaEditor(nuevo, "contenido_editable.modalidad_inscripcion", "conjunta");
  const guardado = validarContenidoEditable({ ...contenido(), modalidad_inscripcion: nuevo.contenido_editable.modalidad_inscripcion }, limites());
  assert.equal(editorDesdeDetalle({ contenido_editable: guardado }).contenido_editable.modalidad_inscripcion, "conjunta");
  for (const invalida of ["", "mixta", null]) {
    assert.throws(() => validarContenidoEditable({ ...contenido(), modalidad_inscripcion: invalida }, limites()), (error) => error.codigo === "modalidad_inscripcion_no_valida");
  }
});

test("el selector obligatorio muestra ambos valores y textos del catálogo", async () => {
  const estado = crearEstadoBorradores();
  estado.faseLista = "listo";
  estado.opciones = opciones();
  estado.lista = lista();
  estado.modoEditor = "crear";
  estado.faseEditor = "listo";
  estado.editor = editorNuevo(estado.opciones);
  const renderizar = crearRenderizadorBorradores({ escaparHTML, estado,
    motivoSeleccionado: () => null, plantillaSeleccionada: () => null }).renderizar;
  const html = renderizar();
  assert.match(html, /<select required data-borrador-ruta="contenido_editable\.modalidad_inscripcion">/);
  assert.match(html, /<option value="" selected disabled>Elija una opción<\/option>/);
  assert.ok(html.indexOf('data-borrador-ruta="contenido_editable.modalidad_inscripcion"') < html.indexOf('class="selector-categorias'));
  assert.match(html, /Una solicitud por categoría/);
  assert.match(html, /Una solicitud para las categorías elegidas/);
  estado.editor.contenido_editable.modalidad_inscripcion = "separada";
  assert.match(renderizar(), /<option value="separada" selected>/);

  const base = new URL("../textos/", import.meta.url);
  const [es, en] = await Promise.all(["es", "en"].map(async (idioma) =>
    JSON.parse(await readFile(new URL(`${idioma}/portal.json`, base), "utf8"))));
  for (const clave of ["etiqueta", "elegir", "separada", "conjunta", "no_valida"]) {
    const nombre = `borrador_modalidad_inscripcion_${clave}`;
    assert.ok(es.textos[nombre]);
    assert.ok(en.textos[nombre]);
  }
});

test("la validación de modalidad devuelve una indicación traducida", () => {
  const fallo = errorSeguro({ codigo: "modalidad_inscripcion_no_valida" }, "fallback");
  assert.equal(fallo.mensaje, "Elija cómo se presentarán las solicitudes.");
  assert.equal(fallo.codigo, "modalidad_inscripcion_no_valida");
});
