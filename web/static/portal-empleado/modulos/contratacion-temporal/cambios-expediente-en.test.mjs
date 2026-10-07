import assert from "node:assert/strict";
import test from "node:test";

globalThis.location = { href: "https://example.invalid/portal-empleado/?lang=en" };

const idioma = await import("../../../comun/idioma.js");
const {
  crearTraductorCambiosExpediente,
  renderizarCambiosExpediente,
  renderizarTablaCambios,
} = await import("./vista-expedientes-cambios.js");

test("Cambios de datos sigue el idioma común sin traducir valores recibidos", () => {
  assert.equal(idioma.LOCALIZACION_ACTUAL, "en-GB");
  const t = crearTraductorCambiosExpediente();
  const apartado = renderizarCambiosExpediente({
    demostracion: false, expediente_ref: "expediente:ct:sintetico:001", version: 2,
  });
  assert.match(apartado, /<summary>Data changes<\/summary>/u);
  assert.match(apartado, />View changes<\/button>/u);

  const instante = "2026-09-24T11:00:00Z";
  const tabla = renderizarTablaCambios({ cambios: [{
    version_expediente: 2, registrada_en: instante, ruta: "solicitud.grupo_subgrupo",
    valor_anterior: "Centro sintético", valor_nuevo: "Categoría sintética",
  }], recortado: false }, t);
  const fecha = new Intl.DateTimeFormat("en-GB", {
    dateStyle: "short", timeStyle: "short", timeZone: "Europe/Madrid",
  }).format(new Date(instante));
  assert.match(tabla, /<th scope="col">Date and time<\/th>/u);
  assert.match(tabla, new RegExp(fecha, "u"));
  assert.match(tabla, /Request · Group\/Subgroup/u);
  assert.match(tabla, /Centro sintético|Categoría sintética/u);
  assert.doesNotMatch(tabla, /Fecha y hora|Solicitud · Grupo\/Subgrupo/u);
});
