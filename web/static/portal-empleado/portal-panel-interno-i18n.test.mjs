import test from "node:test";
import assert from "node:assert/strict";
import { traducirPortal } from "./portal-i18n.js";
import { crearPresentadorPanelInterno } from "./portal-panel-interno.js";

test("B7 traduce los cuatro pasos y distingue registro, recibo y entrega", () => {
  const bolsa = { bolsa_ref: "bolsa:01", categoria: "Auxiliar", tipo_lista: "ordinaria",
    vigente_desde: "2026-09-01", total: 1, por_estado: { disponible: 1 } };
  const flujo = { paso: 1, estados: ["disponible"], participaciones: ["participacion:01"],
    configuracion: null, error: "", recibo: "" };
  const presentador = crearPresentadorPanelInterno({
    claseEstado: () => "info",
    encabezadoVista: (_area, titulo, descripcion, acciones) => `<header><h2>${titulo}</h2><p>${descripcion}</p>${acciones}</header>`,
    escaparHTML: (valor) => String(valor ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;"),
    numero: (valor) => String(valor ?? 0),
    obtenerDatosPanel: () => ({ esquema: "vec.bolsa.panel.interno.v1" }),
    tituloVista: (valor) => valor,
    obtenerDatosCandidatosBolsa: () => ({ carga: "listo", datos: {
      bolsa, candidatos: [{ participacion_ref: "participacion:01", estado_clave: "disponible", orden: 1, nombre_visible: "Persona" }],
      contactos: [],
    } }),
    obtenerEstadoCandidatos: () => ({ nuevo_llamamiento: flujo }),
  });
  const renderizar = () => presentador.renderizarVista("bolsa-candidatos");

  assert.equal(traducirPortal("panel_b7_ayuda_limite_aria"), "Ayuda sobre el límite de selección");
  assert.match(renderizar(), /1\. Seleccionar bolsa/);
  flujo.paso = 2;
  const seleccion = renderizar();
  assert.match(seleccion, /<details><summary aria-label="Ayuda sobre el límite de selección">\?<\/summary>/);
  assert.doesNotMatch(seleccion, /<details open/);
  flujo.paso = 3;
  const configuracion = renderizar();
  assert.match(configuracion, /name="plazo" required minlength="2" maxlength="160" value=""/);
  assert.match(configuracion, /no presupone un plazo legal/);
  assert.match(configuracion, /El registro del llamamiento y su recibo no acreditan entrega/);
  assert.doesNotMatch(configuracion, /48 horas|relay de desarrollo|Recorrido real B7/);
  flujo.configuracion = { plazo: "Pendiente de definición por RRHH" };
  assert.match(renderizar(), /name="plazo" required minlength="2" maxlength="160" value=""/);
  flujo.paso = 4;
  flujo.configuracion = { referencia: "NEC-1", centro: "Centro", modalidad: "Sustitución", plazo: "Indicado por RRHH" };
  flujo.error = "Permiso denegado";
  flujo.recibo = "recibo:123";
  flujo.llamamiento_ref = "llamamiento:123";
  const confirmacion = renderizar();
  assert.match(confirmacion, /Permiso denegado/);
  assert.match(confirmacion, /recibo:123/);
  assert.match(confirmacion, /la entrega del correo debe comprobarse/);
  assert.doesNotMatch(confirmacion, /relay|PostgreSQL|Recorrido real B7/);
});
