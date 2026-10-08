import assert from "node:assert/strict";
import test from "node:test";
import { prepararTextosPortal } from "./portal-i18n.js?v=20261007-pantallas-textos-final-v1";
await prepararTextosPortal("ayuda");
const { crearAyudanteTramites } = await import("./ayudante-tramites.js?v=20261007-pantallas-textos-final-v1");

test("las instrucciones de alternativas y motivo D4 se consultan en el ayudante existente", () => {
  let pulsar;
  let navegaciones = 0;
  const contenedor = {
    innerHTML: "",
    querySelector: () => null,
    addEventListener: (_tipo, accion) => { pulsar = accion; },
    removeEventListener: () => { pulsar = null; },
  };
  const ayudante = crearAyudanteTramites();
  assert.doesNotMatch(ayudante.contenido, /Puede ver otra ruta por carretera|Escriba entre 8 y 500/u);
  const desmontar = ayudante.instalar({ contenedor, documento: { querySelector: () => null }, navegar: () => { navegaciones += 1; } });
  const clic = (dataset, atributo = "") => pulsar({ target: { closest: () => ({ dataset, hasAttribute: (nombre) => nombre === atributo }) } });
  clic({ ayudanteTramite: "dietas-ruta" });
  assert.doesNotMatch(contenedor.innerHTML, /Puede ver otra ruta por carretera/u);
  clic({}, "data-ayudante-siguiente");
  assert.match(contenedor.innerHTML, /Puede ver otra ruta por carretera/u);
  assert.match(contenedor.innerHTML, /Escriba entre 8 y 500 caracteres/u);
  assert.match(contenedor.innerHTML, /no guarda la elección ni calcula un importe/u);
  assert.equal(navegaciones, 0, "consultar las instrucciones no navega ni ejecuta el trámite");
  desmontar();
  assert.equal(pulsar, null);
});
