import assert from "node:assert/strict";
import test from "node:test";
import { crearPresentadorExpedientesContratacionTemporal } from "./presentador-expedientes.js";
import { crearCuadroContratacionTemporalPresentacion, crearExpedienteContratacionTemporalPresentacion } from "./datos-presentacion.js";
import { montarModuloContratacionTemporal } from "./vista-expedientes.js";

const REF = "expediente:ct:sintetico:001";
const cuadro = crearCuadroContratacionTemporalPresentacion();
cuadro.demostracion = false;
cuadro.expedientes[0].expediente_ref = REF;
const expediente = crearExpedienteContratacionTemporalPresentacion();
expediente.demostracion = false;
expediente.expediente_ref = REF;

test("la vista completa prepara Bolsa una vez al seleccionar detalle; cuadro y repintados no la repiten", async () => {
  const fuente = {
    capacidades: ["contratacion_temporal.cuadro.consultar", "contratacion_temporal.expediente.consultar"],
    async listar() { return cuadro; }, async obtener() { return expediente; },
    async ejecutar() { assert.fail("sin efectos"); },
  };
  const presentador = crearPresentadorExpedientesContratacionTemporal({ fuente, capacidades: fuente.capacidades });
  const eventos = new Map();
  const raiz = { innerHTML: "", addEventListener: (tipo, fn) => eventos.set(tipo, fn),
    removeEventListener: (tipo) => eventos.delete(tipo), querySelector: () => null,
    contains: () => true };
  const lecturas = [];
  let preparada = "";
  const modulo = await montarModuloContratacionTemporal({ raiz, presentador,
    resolverBolsa: () => ({ estado: "error" }),
    prepararFichaBolsa: (estado, actualizar, reintentar = false) => {
      if (estado.vista !== "expediente" || estado.carga !== "listo") return;
      if (preparada !== estado.expediente_ref || reintentar) {
        lecturas.push(estado.expediente_ref);
        preparada = estado.expediente_ref;
        actualizar();
      }
    },
  });
  try {
    assert.equal(lecturas.length, 0);
    const control = { dataset: { ctExpAbrir: REF }, closest: (selector) => selector === "[data-ct-exp-abrir]" ? control : null };
    await eventos.get("click")({ target: control, preventDefault() {} });
    assert.deepEqual(lecturas, [REF]);
    assert.equal(presentador.obtenerEstado().vista, "expediente");
    assert.equal(presentador.obtenerEstado().carga, "listo");
    const reintentar = { closest: (selector) => selector === "[data-ct-bolsa-reintentar]" ? reintentar : null };
    await eventos.get("click")({ target: reintentar, preventDefault() {} });
    assert.deepEqual(lecturas, [REF, REF], "el reintento de Bolsa no repite la consulta CT");
  } finally { modulo.desmontar(); }
});
