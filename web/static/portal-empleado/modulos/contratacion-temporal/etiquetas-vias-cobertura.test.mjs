import assert from "node:assert/strict";
import test from "node:test";

import { cargarEtiquetasViasCobertura, etiquetasViasDesdeReglas } from "./etiquetas-vias-cobertura.js";
import { montarFormularioCobertura } from "./formulario-cobertura.js";

const HUELLA = "a".repeat(64);
const reglas = (reglasCT) => ({ esquema: "vec.reglas.vigentes.v1", catalogos: [
  { modulo: "contratacion_temporal", catalogo_id: "vec.contratacion_temporal.reglas", estado: "disponible", paquete_ejemplo: true, reglas: reglasCT },
  { modulo: "bolsa", catalogo_id: "vec.bolsa.reglas", estado: "disponible", paquete_ejemplo: true,
    reglas: [{ clave: "c17.via_cobertura.ajena", etiqueta: "De otro catálogo" }] },
] });

test("solo toma las vías del catálogo de Contratación temporal", () => {
  const mapa = etiquetasViasDesdeReglas(reglas([
    { clave: "c17.via_cobertura.oferta_sae", etiqueta: "Oferta del SAE" },
    { clave: "c17.via_cobertura.mejora_empleo", etiqueta: "Mejora de empleo" },
    { clave: "c16.numeracion", etiqueta: "Numeración" },
    { clave: "c17.via_cobertura.", etiqueta: "Sin clave" },
  ]));
  assert.deepEqual([...mapa], [["oferta_sae", "Oferta del SAE"], ["mejora_empleo", "Mejora de empleo"]]);
  assert.equal(etiquetasViasDesdeReglas(null).size, 0);
});

test("un fallo al consultar el catálogo no bloquea y deja reintentar", async () => {
  let llamadas = 0;
  const fallido = { reglas: async () => { llamadas += 1; throw new Error("sin servicio"); } };
  assert.equal((await cargarEtiquetasViasCobertura(fallido)).size, 0);
  const bueno = { reglas: async () => { llamadas += 1; return reglas([{ clave: "c17.via_cobertura.x_y", etiqueta: "Vía X" }]); } };
  assert.equal((await cargarEtiquetasViasCobertura(bueno)).get("x_y"), "Vía X");
  assert.equal(llamadas, 2);
});

function raizFalsa() {
  const eventos = new Map();
  return { innerHTML: "", eventos, addEventListener(tipo, f) { eventos.set(tipo, f); }, removeEventListener(tipo) { eventos.delete(tipo); },
    contains() { return true; }, querySelector() { return { focus() {}, scrollIntoView() {} }; }, replaceChildren() { this.innerHTML = ""; } };
}

test("la propuesta nombra con el catálogo una vía que no es de las de siempre", async () => {
  const raiz = raizFalsa();
  const evaluacion = (via_clave, prioridad) => ({ via_clave, prioridad, estado: "viable", resultados_omitidos: [], ausencias_bloqueantes: [],
    ausencias_admitidas: [], no_habilitantes: [], conflictos: [] });
  const desmontar = montarFormularioCobertura({
    raiz,
    cliente: {
      async proponerCobertura() {
        return { esquema: "vec.contratacion-temporal.propuesta-cobertura.v1", estado: "viable", via_recomendada: "mejora_empleo",
          evaluaciones: [evaluacion("mejora_empleo", 1), evaluacion("bolsa_vigente", 2)],
          identidad_semantica: { referencia: `propuesta-cobertura-semantica:sha256:${HUELLA}`, huella_sha256: HUELLA,
            canon: { dominio: "vec.dipgra.contratacion-temporal.propuesta-decision-cobertura-semantica", version_esquema: 1, algoritmo: "sha-256" } } };
      },
      async decidirCobertura() { throw new Error("no"); },
      async consultarResultadoCobertura() { throw new Error("no"); },
    },
    contexto: { expediente_ref: "expediente:ct:prueba:cobertura:001", version_esperada: 2 },
    etiquetasVias: async () => new Map([["mejora_empleo", "Mejora de empleo"]]),
  });
  for (let i = 0; i < 10; i += 1) await new Promise((r) => setImmediate(r));
  assert.match(raiz.innerHTML, /<strong>Mejora de empleo<\/strong>/u);
  assert.match(raiz.innerHTML, /data-ct-cobertura-evaluacion="mejora_empleo"[\s\S]*?<span>Mejora de empleo<\/span>/u);
  assert.match(raiz.innerHTML, /data-ct-cobertura-evaluacion="bolsa_vigente"[\s\S]*?<span>Bolsa vigente<\/span>/u, "las de siempre conservan su texto");
  desmontar();
});
