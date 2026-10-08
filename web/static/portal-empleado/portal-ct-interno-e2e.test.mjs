import assert from "node:assert/strict";
import test from "node:test";

import { crearCatalogoModulosDesdeManifiestos } from "./portal-catalogo-modulos.js?v=20261001-ct-a-i18n-v1";
import { crearCoordinadorModulosPortal } from "./portal-modulos-coordinador.js?v=20261008-alta-rpt-circular-v5";

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
  const rutas = [];
  const fetchImpl = async (ruta, opciones) => {
    rutas.push(ruta);
    if (ruta === "/api/vec/contratacion-temporal/catalogos-alta") {
      assert.equal(opciones.method, "GET");
      return new Response(JSON.stringify({ data: {
        esquema: "vec.contratacion_temporal.catalogos_alta.v1",
        numero_expediente_moad: { referencia: "catalogo:ct:numero-expediente-moad", version: 1,
          patron: "^[0-9]{4}/[1-9][0-9]{0,9}$", ejemplo: "2026/5487" },
        centros: [{ referencia: "centro:001", etiqueta: "Centro 001", contactos: [{ referencia: "con:001", etiqueta: "Contacto 001" }] }],
        categorias: [{ referencia: "categoria:auxiliar", etiqueta: "categoria:auxiliar", grupos_subgrupos: [{ clave: "C2", etiqueta: "Grupo C2" }] }],
        motivos: [{ clave: "sustitucion", etiqueta: "Sustitución" }],
        documentos: [{ referencia: "doc:001", etiqueta: "Documento 001" }],
      } }), {
        status: 200,
        headers: { "Content-Type": "application/json; charset=utf-8" },
      });
    }
    assert.equal(ruta, "/api/vec/contratacion-temporal/cuadro/consultas");
    assert.equal(opciones.method, "POST");
    assert.equal(opciones.headers.get("content-type"), "application/json");
    const solicitud = JSON.parse(opciones.body);
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
      ...(solicitud.resumen ? { resumen: { en_tramite: 1, con_incidencia: 0,
        vencidos: 0, vencen_hoy: 0, vencen_semana: 0, sin_calcular: 0, por_fase: { solicitud: 1 } } } : {}),
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
  const acceso = coordinador.resolverAcceso("contratacion_temporal");
  assert.equal(acceso.disponible, true, `acceso=${JSON.stringify(acceso)}; consultas=${consultas}; rutas=${rutas.join(",")}`);
  assert.equal(consultas, 0, "el cargador CT no consulta el cuadro antes de abrir Inicio o la lista");
  const resumen = await coordinador.prepararResumenInicio();
  assert.equal(resumen.resumen.en_tramite, 1);
  assert.equal(consultas, 1, "Inicio consulta su resumen al necesitarlo");

  const raiz = raizFalsa();
  const montada = await coordinador.montarVista("contratacion-temporal", raiz);
  assert.equal(montada, true, `vista disponible=${coordinador.vistaDisponible("contratacion-temporal")}; consultas=${consultas}; contenido=${raiz.innerHTML.slice(0, 160)}`);
  assert.equal(consultas, 2, "la lista hace una lectura paginada adicional al resumen de Inicio");
  assert.match(raiz.innerHTML, /2026\/CT-0001/);
  assert.match(raiz.innerHTML, /categoria:auxiliar/);
  assert.match(raiz.innerHTML, /option value="solicitud"/);
  assert.doesNotMatch(raiz.innerHTML, /option value="Solicitud"/);
  assert.doesNotMatch(raiz.innerHTML, /mis tareas/i);
  assert.doesNotMatch(raiz.innerHTML, /DEMO|demostraci[oó]n/i);
});
