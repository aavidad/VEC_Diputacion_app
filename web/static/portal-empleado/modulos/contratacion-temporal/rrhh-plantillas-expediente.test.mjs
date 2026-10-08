import assert from "node:assert/strict";
import test from "node:test";
import { contextoPlantillasPublicadasDesdeEstado } from "./vista-expedientes.js?v=20261008-alta-rpt-circular-v6";

const expediente = Object.freeze({ expediente_ref: "expediente:ct:uno", version: 3, demostracion: false });
const cuadro = Object.freeze({ demostracion: false, expedientes: [{ expediente_ref: expediente.expediente_ref,
  version: expediente.version, fase_clave: "analisis_rrhh", estado_clave: "en_curso" }] });
const estado = Object.freeze({ vista: "expediente", carga: "listo", cuadro, expediente,
  expediente_ref: expediente.expediente_ref, ocupado: false,
  actualizacion_pendiente: false, resultado_indeterminado: false });

test("la consulta de tipos publicados usa la versión autorizada también antes de formalización", () => {
  assert.deepEqual(contextoPlantillasPublicadasDesdeEstado(estado), {
    expediente_ref: expediente.expediente_ref, version_observada: 3,
  });
  assert.deepEqual(contextoPlantillasPublicadasDesdeEstado({ ...estado, vista: "documentos",
    documentos: { demostracion: false, expediente_ref: expediente.expediente_ref, version: 3 } }), {
    expediente_ref: expediente.expediente_ref, version_observada: 3,
  });
});

test("no solicita catálogo con datos de muestra, detalle desfasado ni resultado indeterminado", () => {
  for (const alterado of [
    { expediente: { ...expediente, demostracion: true } },
    { cuadro: { ...cuadro, expedientes: [{ ...cuadro.expedientes[0], version: 4 }] } },
    { resultado_indeterminado: true },
    { vista: "documentos", documentos: { demostracion: false,
      expediente_ref: expediente.expediente_ref, version: 2 } },
  ]) assert.equal(contextoPlantillasPublicadasDesdeEstado({ ...estado, ...alterado }), null);
});
