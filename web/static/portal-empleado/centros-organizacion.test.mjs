import assert from "node:assert/strict";
import test from "node:test";
import { centrosDeOrganizacion } from "./portal-modulos-coordinador.js?v=20261008-alta-rpt-circular-v6";

test("los centros de Organización se nombran con su clave y con la forma de Contratación", () => {
  const centros = centrosDeOrganizacion([
    { clave: "centro-600", etiqueta: "DEPORTES", tipo: "centro", adscripcion_clave: "delegacion-04" },
    { clave: "delegacion-04", etiqueta: "DELEGACIÓN", tipo: "delegacion" },
    { clave: "centro-a1b", etiqueta: "OTRO", tipo: "centro" },
  ]);
  assert.equal(centros.get("centro-600"), "DEPORTES");
  assert.equal(centros.get("centro:rpt:600"), "DEPORTES");
  assert.equal(centros.get("centro:rpt:A1B"), "OTRO");
  assert.equal(centros.has("delegacion-04"), false);
  assert.equal(centrosDeOrganizacion(null).size, 0);
});
