import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { runInNewContext } from "node:vm";
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

test("F5 entrega también el callback que mantiene la URL al editar el filtro", async () => {
  const portal = await readFile(new URL("./portal.js", import.meta.url), "utf8");
  const inicio = portal.indexOf("function opcionesDesdeEnlace(vista)");
  const fin = portal.indexOf("function vistaDesdeHash()", inicio);
  assert.ok(inicio > 0 && fin > inicio);
  const location = { pathname: "/portal-empleado/", search: "?lang=en&ct_estado=incidencia", hash: "#contratacion-temporal" };
  const callback = () => {};
  const leer = runInNewContext(`${portal.slice(inicio, fin)}; opcionesDesdeEnlace`, {
    window: { location }, URLSearchParams, leerFiltroCTDeRuta,
    alCambiarFiltroListaCT: callback, history: { replaceState() {} },
  });
  const opciones = leer("contratacion-temporal");
  assert.deepEqual({ ...opciones.filtroServidorRuta }, FILTRO_INCIDENCIA_CT);
  assert.equal(opciones.alCambiarFiltroLista, callback);
  location.search = "?lang=en";
  assert.equal(leer("contratacion-temporal").alCambiarFiltroLista, callback,
    "la entrada CT sin filtro también sincroniza cambios posteriores");
  assert.equal(leer("portal"), null);
});
