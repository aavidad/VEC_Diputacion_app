import assert from "node:assert/strict";
import test from "node:test";
import { esFiltroIncidenciaCT, leerFiltroCTDeRuta, limpiarFiltroCTDeBusqueda,
  rutaPortalConFiltroCT } from "./portal-ct-ruta-filtro.js";

test("la URL compartida solo representa incidencia y conserva idioma y expediente", () => {
  const ubicacion = { pathname: "/portal-empleado/", search: "?lang=en&expediente=expediente%3Act%3A1" };
  const ruta = rutaPortalConFiltroCT(ubicacion, "#contratacion-temporal", "incidencia");
  assert.equal(ruta, "/portal-empleado/?lang=en&expediente=expediente%3Act%3A1&ct_mostrar=incidencia#contratacion-temporal");
  assert.deepEqual(leerFiltroCTDeRuta(new URL(ruta, "https://vec.example").search), { mostrar: "incidencia" });
  assert.equal(rutaPortalConFiltroCT({ ...ubicacion, search: "?lang=en&ct_mostrar=incidencia&expediente=expediente%3Act%3A1" },
    "#portal"), "/portal-empleado/?lang=en&expediente=expediente%3Act%3A1#portal");
  assert.equal(limpiarFiltroCTDeBusqueda("?lang=es&ct_mostrar=incidencia&vista=ct"), "?lang=es&vista=ct");
});

test("duplicados y filtros V1 sin predicado completo no se convierten en incidencia", () => {
  assert.equal(leerFiltroCTDeRuta("?lang=es"), null);
  for (const busqueda of ["?ct_mostrar=sin_plazo", "?ct_mostrar=vencidos", "?ct_mostrar=incidencia&ct_mostrar=incidencia"]) {
    assert.equal(leerFiltroCTDeRuta(busqueda).mostrar, "__filtro_no_soportado__");
  }
  assert.equal(esFiltroIncidenciaCT({ mostrar: "incidencia", fase: "" }), true);
  assert.equal(esFiltroIncidenciaCT({ mostrar: "incidencia", fase: "nombramiento" }), false);
  assert.equal(esFiltroIncidenciaCT({ mostrar: "incidencia", texto: "otro" }), false);
  assert.equal(esFiltroIncidenciaCT({ mostrar: "vencidos" }), false);
  assert.throws(() => rutaPortalConFiltroCT({ pathname: "/portal-empleado/", search: "" }, "#contratacion-temporal", "vencidos"));
  assert.throws(() => rutaPortalConFiltroCT({ pathname: "//otro.example/", search: "" }, "#portal"));
});
