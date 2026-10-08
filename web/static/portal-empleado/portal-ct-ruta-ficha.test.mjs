import assert from "node:assert/strict";
import test from "node:test";
import { enlaceCTSinVersionFueraDePagina, leerFichaCTDeRuta, rutaConFichaCT } from "./portal-ct-ruta-ficha.js";

test("la ficha CT conserva filtro e idioma sin introducir actor en la URL", () => {
  const ruta = rutaConFichaCT({ pathname: "/portal-empleado/", search: "?lang=en&ct_estado=incidencia",
    hash: "#contratacion-temporal" }, { expedienteRef: "expediente:ct:opaco-127", version: 7 });
  const enlace = new URL(ruta, "https://vec.example");
  assert.equal(enlace.searchParams.get("ct_estado"), "incidencia");
  assert.equal(enlace.searchParams.get("lang"), "en");
  assert.deepEqual(leerFichaCTDeRuta(enlace.search), { expedienteRef: "expediente:ct:opaco-127", version: 7 });
  assert.equal(enlace.searchParams.has("actor"), false);
  assert.equal(rutaConFichaCT({ pathname: enlace.pathname, search: enlace.search, hash: "#portal" }),
    "/portal-empleado/?lang=en&ct_estado=incidencia#portal");
});

test("la ruta rechaza referencias, versiones y parámetros ambiguos", () => {
  for (const busqueda of ["?expediente=%2Fapi%2Fvec", "?expediente=ref%3A1&expediente=ref%3A2",
    "?expediente=ref%3A1&expediente_version=0", "?expediente=ref%3A1&expediente_version=1000000001",
    "?expediente=ref%3A1&expediente_version=1&expediente_version=2"]) {
    assert.equal(leerFichaCTDeRuta(busqueda), null, busqueda);
  }
  assert.throws(() => rutaConFichaCT({ pathname: "//evil.example/", search: "", hash: "#contratacion-temporal" }));
});

test("un enlace antiguo sin versión fuera de la primera página exige un aviso recuperable", () => {
  const ficha = leerFichaCTDeRuta("?expediente=expediente%3Act%3A127");
  const cuadro = { demostracion: false, expedientes: [{ expediente_ref: "expediente:ct:001" }] };
  assert.equal(enlaceCTSinVersionFueraDePagina(ficha, cuadro), true);
  assert.equal(enlaceCTSinVersionFueraDePagina(ficha, { ...cuadro,
    expedientes: [{ expediente_ref: "expediente:ct:127" }] }), false);
  assert.equal(enlaceCTSinVersionFueraDePagina({ ...ficha, version: 7 }, cuadro), false);
  assert.equal(enlaceCTSinVersionFueraDePagina(ficha, null), false);
});
