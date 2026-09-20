import assert from "node:assert/strict";
import test from "node:test";
import { crearTraductorRPTPuestos, formatearCentimosRPT, formatearRecuentoRPTPuestos, formatearResumenRPTPuestos } from "./i18n-rpt-puestos.js";

test("i18n RPT localiza recuentos e importes sin tocar el catálogo común", () => {
  const t = crearTraductorRPTPuestos();
  assert.match(t("ayuda"), /No contiene ocupantes/);
  assert.equal(formatearResumenRPTPuestos({ puestos: 842, dotacion: 1714, categorias: 145, centros: 41 }), "842 puestos · 1.714 dotaciones · 145 categorías · 41 centros");
  assert.equal(formatearResumenRPTPuestos({ puestos: 2, dotacion: 3, categorias: 4, centros: 5 }), "2 puestos · 3 dotaciones · 4 categorías · 5 centros");
  assert.equal(formatearRecuentoRPTPuestos(1, "puestos"), "1 puesto");
  assert.equal(formatearRecuentoRPTPuestos(145, "categorias"), "145 categorías");
  assert.match(formatearCentimosRPT(1427496), /14[. ]274,96/);
  assert.throws(() => formatearRecuentoRPTPuestos(-1, "puestos"), /no válido/);
});
