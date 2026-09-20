import assert from "node:assert/strict";
import test from "node:test";
import { crearTraductorRPTPuestos, formatearCentimosRPT, formatearRecuentoRPTPuestos } from "./i18n-rpt-puestos.js";

test("i18n RPT localiza recuentos e importes sin tocar el catálogo común", () => {
  const t = crearTraductorRPTPuestos();
  assert.match(t("ayuda"), /No contiene ocupantes/);
  assert.match(t("resumen"), /842 puestos/);
  assert.equal(formatearRecuentoRPTPuestos(1, "puestos"), "1 puesto");
  assert.equal(formatearRecuentoRPTPuestos(145, "categorias"), "145 categorías");
  assert.match(formatearCentimosRPT(1427496), /14[. ]274,96/);
  assert.throws(() => formatearRecuentoRPTPuestos(-1, "puestos"), /no válido/);
});
