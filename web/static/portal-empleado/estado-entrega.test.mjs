import assert from "node:assert/strict";
import test from "node:test";
import {
  ESTADOS_ENTREGA,
  escaparTextoEstadoEntrega,
  renderizarEstadoEntrega,
  validarEstadoEntrega,
} from "./estado-entrega.js";
import {
  CLAVES_ESTADO_ENTREGA,
  crearTraductorEstadoEntrega,
  MENSAJES_ESTADO_ENTREGA_ES,
} from "./estado-entrega-i18n.js";

const pendiente = {
  estado: "visual_pendiente_backend",
  resumen: "La pantalla está lista para la revisión de RRHH.",
  pendientes: ["Conectar el caso de uso real", "Acreditar el recorrido con datos sintéticos"],
  fuente: { etiqueta: "Inventario funcional VEC" },
  conexion: "API interna del módulo",
};

test("acepta los tres estados cerrados y comunica el límite visible", () => {
  assert.deepEqual(ESTADOS_ENTREGA, ["conectado", "visual_pendiente_backend", "bloqueado_dependencia"]);
  for (const estado of ESTADOS_ENTREGA) {
    const entrada = { ...pendiente, estado, pendientes: estado === "conectado" ? [] : pendiente.pendientes };
    const html = renderizarEstadoEntrega(entrada);
    assert.match(html, /Qué falta para terminarlo/u);
    assert.match(html, /estado-entrega--/u);
  }
});

test("el catálogo español es completo, cerrado y sirve todas las etiquetas de interfaz", () => {
  assert.deepEqual(CLAVES_ESTADO_ENTREGA, [
    "estado_conectado_etiqueta", "estado_conectado_encabezado",
    "estado_pendiente_etiqueta", "estado_pendiente_encabezado",
    "estado_bloqueado_etiqueta", "estado_bloqueado_encabezado",
    "aria_estado", "pendientes_vacios", "pendientes_titulo",
    "fuente_etiqueta", "conexion_etiqueta",
  ]);
  const traducir = crearTraductorEstadoEntrega();
  for (const clave of CLAVES_ESTADO_ENTREGA.filter((clave) => clave !== "aria_estado")) {
    assert.equal(traducir(clave), MENSAJES_ESTADO_ENTREGA_ES[clave]);
  }
  assert.equal(traducir("aria_estado", { estado: "Conectado" }), "Estado de entrega: Conectado");
  assert.throws(() => traducir("texto_inventado"), /clave de estado de entrega desconocida/u);
  assert.throws(() => crearTraductorEstadoEntrega({ ...MENSAJES_ESTADO_ENTREGA_ES, extra: "no" }), /catálogo/u);
  assert.throws(() => crearTraductorEstadoEntrega({ ...MENSAJES_ESTADO_ENTREGA_ES, pendientes_titulo: "" }), /catálogo/u);
});

test("escapa todo texto interpolado", () => {
  const html = renderizarEstadoEntrega({
    ...pendiente,
    resumen: '<img src=x onerror="alert(1)">',
    pendientes: ["<script>robar()</script>"],
    fuente: { etiqueta: "A & B ' C" },
  });
  assert.doesNotMatch(html, /<script>|<img /u);
  assert.match(html, /&lt;script&gt;robar\(\)&lt;\/script&gt;/u);
  assert.match(html, /A &amp; B &#39; C/u);
  assert.equal(escaparTextoEstadoEntrega('"<&>'), "&quot;&lt;&amp;&gt;");
});

test("rechaza campos, URLs y estados que ocultan pendientes", () => {
  assert.throws(() => validarEstadoEntrega({ ...pendiente, url: "https://externo.test" }), /estado\.url/u);
  assert.throws(() => validarEstadoEntrega({ ...pendiente, fuente: { etiqueta: "x", url: "/interna" } }), /fuente\.url/u);
  assert.throws(() => validarEstadoEntrega({ ...pendiente, pendientes: [] }), /pendientes vacíos/u);
  assert.throws(() => validarEstadoEntrega({ ...pendiente, estado: "listo" }), /estado/u);
  assert.throws(() => validarEstadoEntrega({ ...pendiente, pendientes: [" "] }), /pendientes\.0/u);
});
