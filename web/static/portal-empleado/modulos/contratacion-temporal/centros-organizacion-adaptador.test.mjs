import assert from "node:assert/strict";
import test from "node:test";
import { crearAdaptadorHTTPExpedientesContratacionTemporal } from "./adaptador-http-expedientes.js?v=20261008-alta-rpt-circular-v5";

// Sin catálogo del alta, el adaptador nombra el centro con Organización.
const resumen = Object.freeze({
  expediente_ref: "expediente:ct:001", numero_visible: "2026/CT-0001", version: 1,
  flujo_ref: "flujo:ct:general", flujo_version: 1, flujo_huella_sha256: "a".repeat(64),
  fase_clave: "solicitud", estado_clave: "en_curso", centro_ref: "centro:rpt:600",
  categoria_ref: "categoria:auxiliar", creado_en: "2026-09-03T08:00:00Z", actualizado_en: "2026-09-03T09:00:00Z",
});
const cliente = Object.freeze({
  consultarCuadroRRHH: async () => ({ esquema: "vec.contratacion-temporal.cuadro-rrhh.v1",
    generada_en: "2026-09-03T09:05:00Z", expedientes: [resumen], hay_mas: false }),
  consultarDetalleRRHH: async () => { throw new Error("no se usa"); },
});

test("sin catálogo del alta, el centro se nombra con Organización", async () => {
  const adaptador = crearAdaptadorHTTPExpedientesContratacionTemporal({
    cliente, obtenerCentrosOrganizacion: async () => new Map([["centro:rpt:600", "DEPORTES"]]),
  });
  assert.equal((await adaptador.listar()).expedientes[0].centro, "DEPORTES");
});

test("si Organización falla, la lista sigue con la referencia", async () => {
  const adaptador = crearAdaptadorHTTPExpedientesContratacionTemporal({
    cliente, obtenerCentrosOrganizacion: async () => { throw new Error("caída"); },
  });
  assert.equal((await adaptador.listar()).expedientes[0].centro, "centro:rpt:600");
});
