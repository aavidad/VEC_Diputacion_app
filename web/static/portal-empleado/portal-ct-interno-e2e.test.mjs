import assert from "node:assert/strict";
import test from "node:test";

import { crearCatalogoModulosDesdeManifiestos } from "./portal-catalogo-modulos.js";
import { crearCoordinadorModulosPortal } from "./portal-modulos-coordinador.js";

function raizFalsa() {
  const eventos = new Map();
  return {
    innerHTML: "",
    addEventListener(tipo, manejador) { eventos.set(tipo, manejador); },
    removeEventListener(tipo, manejador) {
      if (eventos.get(tipo) === manejador) eventos.delete(tipo);
    },
    replaceChildren() { this.innerHTML = ""; },
    contains() { return true; },
    querySelector() { return null; },
    querySelectorAll() { return []; },
    setAttribute() {},
    removeAttribute() {},
  };
}

test("el portal interno recorre cliente, adaptador y vista reales de contratación temporal", async () => {
  const manifiesto = {
    id: "vec.module.contratacion_temporal",
    name_key: "ui.vec.module.contratacion_temporal.name",
    description_key: "ui.vec.module.contratacion_temporal.description",
    version: "v0.2.0",
    group: "recursos_humanos",
    base_path: "/modules/contratacion-temporal",
    permissions: [{
      key: "contratacion_temporal.cuadro.consultar",
      label_key: "ui.permission.contratacion_temporal.cuadro",
    }],
    menu: [{
      id: "contratacion_temporal.cuadro",
      module_id: "vec.module.contratacion_temporal",
      label_key: "ui.vec.menu.contratacion_temporal.cuadro",
      path: "/modules/contratacion-temporal/cuadro",
      icon: "layout-dashboard",
      group: "modulo_contratacion_temporal",
      order: 100,
      required_permissions: ["contratacion_temporal.cuadro.consultar"],
    }],
  };
  const catalogo = crearCatalogoModulosDesdeManifiestos([manifiesto], {
    "ui.vec.module.contratacion_temporal.name": "Contratación temporal",
    "ui.vec.module.contratacion_temporal.description": "Expedientes temporales",
  });
  let consultas = 0;
  const fetchImpl = async (ruta, opciones) => {
    assert.equal(ruta, "/api/vec/contratacion-temporal/cuadro/consultas");
    assert.equal(opciones.method, "POST");
    consultas += 1;
    return new Response(JSON.stringify({ data: {
      esquema: "vec.contratacion-temporal.cuadro-rrhh.v1",
      generada_en: "2026-09-03T09:05:00Z",
      expedientes: [{
        expediente_ref: "expediente:ct:001",
        numero_visible: "2026/CT-0001",
        version: 1,
        flujo_ref: "flujo:ct:general",
        flujo_version: 1,
        flujo_huella_sha256: "a".repeat(64),
        fase_clave: "solicitud",
        estado_clave: "pendiente",
        centro_ref: "centro:001",
        categoria_ref: "categoria:auxiliar",
        creado_en: "2026-09-03T08:00:00Z",
        actualizado_en: "2026-09-03T09:00:00Z",
      }],
      hay_mas: false,
    } }), {
      status: 200,
      headers: { "Content-Type": "application/json; charset=utf-8" },
    });
  };
  const coordinador = crearCoordinadorModulosPortal({
    escaparHTML: String,
    cargarCatalogoInterno: async () => catalogo,
    entorno: { fetch: fetchImpl, Headers },
  });
  await coordinador.cargarInterno();
  assert.equal(coordinador.resolverAcceso("contratacion_temporal").disponible, true);

  const raiz = raizFalsa();
  assert.equal(await coordinador.montarVista("contratacion-temporal", raiz), true);
  assert.equal(consultas, 2);
  assert.match(raiz.innerHTML, /2026\/CT-0001/);
  assert.match(raiz.innerHTML, /categoria:auxiliar/);
  assert.doesNotMatch(raiz.innerHTML, /DEMO|demostraci[oó]n/i);
});
