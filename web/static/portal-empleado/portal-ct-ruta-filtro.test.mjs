import assert from "node:assert/strict";
import test from "node:test";
import { FILTRO_CT_NO_SOPORTADO, FILTRO_INCIDENCIA_CT, filtroServidorCTValido,
  leerFiltroCTDeRuta, limpiarFiltroCTDeBusqueda, rutaPortalConFiltroCT } from "./portal-ct-ruta-filtro.js";

test("la URL CT V1 conserva idioma y expediente y solo codifica el predicado aplicado", () => {
  const ubicacion = { pathname: "/portal-empleado/", search: "?lang=en&expediente=expediente%3Act%3A1" };
  const ruta = rutaPortalConFiltroCT(ubicacion, "#contratacion-temporal", FILTRO_INCIDENCIA_CT);
  assert.equal(ruta, "/portal-empleado/?lang=en&expediente=expediente%3Act%3A1&ct_estado=incidencia#contratacion-temporal");
  assert.deepEqual(leerFiltroCTDeRuta(new URL(ruta, "https://vec.example").search), FILTRO_INCIDENCIA_CT);
  const combinado = { texto: "2026/CT", estado_clave: "espera_externa", fase_clave: "analisis" };
  const enlace = rutaPortalConFiltroCT({ ...ubicacion, search: "?lang=es&ct_estado=incidencia" },
    "#contratacion-temporal", combinado);
  assert.equal(enlace, "/portal-empleado/?lang=es&ct_texto=2026%2FCT&ct_estado=espera_externa&ct_fase=analisis#contratacion-temporal");
  assert.deepEqual(leerFiltroCTDeRuta(new URL(enlace, "https://vec.example").search), combinado);
  assert.equal(rutaPortalConFiltroCT({ ...ubicacion, search: "?lang=en&ct_estado=incidencia&expediente=expediente%3Act%3A1" },
    "#portal"), "/portal-empleado/?lang=en&expediente=expediente%3Act%3A1#portal");
});

test("el lector deniega filtros duplicados, plazos y claves ajenas sin convertirlos en incidencia", () => {
  assert.equal(leerFiltroCTDeRuta("?lang=es"), null);
  for (const busqueda of ["?ct_mostrar=incidencia", "?ct_estado=sin_plazo", "?ct_estado=incidencia&ct_estado=incidencia",
    "?ct_fase=gestion_bolsa", "?ct_texto=%40", "?ct_cursor=opaco"]) {
    assert.equal(leerFiltroCTDeRuta(busqueda), FILTRO_CT_NO_SOPORTADO, busqueda);
  }
  assert.equal(filtroServidorCTValido({ texto: "", estado_clave: "incidencia", fase_clave: "cierre" }), true);
  assert.equal(filtroServidorCTValido({ texto: "", estado_clave: "incidencia", fase_clave: "gestion_bolsa" }), false);
  assert.equal(limpiarFiltroCTDeBusqueda("?lang=es&ct_estado=incidencia&ct_fase=cierre&vista=ct"), "?lang=es&vista=ct");
  assert.throws(() => rutaPortalConFiltroCT({ pathname: "//otro.example/", search: "" }, "#portal"));
});
