import assert from "node:assert/strict";
import test from "node:test";
import { cargarTextos } from "../../../comun/textos.js";
import { crearTextosConsultaMerito } from "./consulta-i18n.js";
import { renderizarConsultaMerito } from "./consulta-vista.js";
import { resultadoPrueba } from "./consulta-prueba.test-helper.mjs";

test("la carga predeterminada usa el catálogo castellano real", () => {
  assert.equal(crearTextosConsultaMerito().idioma, "es");
  assert.equal(crearTextosConsultaMerito().t("titulo"), "Detalle del mérito");
});

for (const [idioma, titulo, fecha, cantidad, plural] of [
  ["es", "Detalle del mérito", "2 oct 2026, 10:40", "12.345", "2 referencias de evidencia"],
  ["en", "Merit details", "2 Oct 2026, 10:40", "12,345", "2 evidence references"],
]) {
  test(`traductor común ${idioma}: ficha, plural, números y fecha Madrid sin claves pendientes`, async () => {
    const comun = await cargarTextos("meritos-consulta", { idioma, porDefecto: "es" });
    assert.deepEqual(comun.faltantes, []);
    const textos = crearTextosConsultaMerito(comun);
    assert.equal(textos.t("titulo"), titulo);
    assert.equal(textos.numero(12345), cantidad);
    assert.equal(textos.instante("2026-10-02T08:40:00Z"), fecha);
    assert.equal(textos.t("evidencias_cuenta", { cuenta: 2 }), plural);
    const datos = resultadoPrueba(); datos.hecho_actual.evidencias.push({ id: "documento:2", version: 2 });
    const html = renderizarConsultaMerito({ estado: "resultado", resultado: datos, hechoRef: "hecho:propio-a" }, textos);
    assert.ok(html.includes(titulo)); assert.ok(html.includes(plural)); assert.ok(html.includes(fecha));
    assert.doesNotMatch(html, /consulta\.(?:titulo|procedencia|evidencias)/u);
    assert.throws(() => textos.t("no_existe"));
  });
}
