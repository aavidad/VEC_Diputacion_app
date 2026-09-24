import assert from "node:assert/strict";
import test from "node:test";
import { crearUtilidadesVista } from "./portal-vistas-utilidades.js";
import { crearVistaReglas } from "./portal-vistas-reglas.js";

function utilidades(presentacion = false) {
  const escaparHTML = (valor) => String(valor ?? "").replace(/[&<>"]/g, (caracter) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" })[caracter]);
  return {
    escaparHTML, numero: (valor) => String(valor), fecha: (valor) => String(valor ?? ""),
    chip: (estado) => `<span>${escaparHTML(estado)}</span>`,
    tabla: ({ filas, vacio }) => filas.length ? `<table>${filas.flat().join("|")}</table>` : `<p>${escaparHTML(vacio)}</p>`,
    kpi: (_sigla, valor, etiqueta) => `${etiqueta}:${valor}`,
    encabezadoVista: (_miga, titulo, descripcion) => `<h1>${titulo}</h1><p>${descripcion}</p>`,
    avisoPresentacion: (texto) => `<aside>${escaparHTML(texto)}</aside>`,
    fuentePresentacion: () => "<span>Datos sintéticos · Memoria volátil</span>",
    esPresentacion: () => presentacion,
    botonOperacion: (_etiqueta, operacion, objetivo) => `<button data-operacion="${operacion}" data-objetivo="${objetivo}"></button>`,
    campo: (_etiqueta, control) => control,
  };
}
test("reglas conserva versión y procedencia sin habilitar edición", () => {
  const html = crearVistaReglas(utilidades()).renderizarReglas({
    reglas: [{ nombre: "Experiencia", ambito: "Bolsa demo", version: "v3", vigencia: "Desde hoy", estado: "Borrador", procedencia: "Escenario sintético" }], criterios_baremo: [],
  });
  assert.match(html, /v3/); assert.match(html, /Escenario sintético/);
  assert.match(html, /Datos sintéticos/); assert.match(html, /Edición de versiones no conectada/);
  assert.match(html, /data-comando="guardar-reglas-baremo"/);
  assert.match(html, /disabled aria-disabled="true"/);
});
test("reglas representa carga, error, denegación y vacío sin revelar datos", () => {
  const vista = crearVistaReglas(utilidades());
  assert.match(vista.renderizarReglas({}, { fase: "cargando" }), /Cargando versiones/);
  assert.match(vista.renderizarReglas({}, { fase: "error", detalle: "Error de red" }), /Error de red/);
  assert.match(vista.renderizarReglas({}, { fase: "denegado" }), /Acceso a reglas denegado/);
  const sinEstado = vista.renderizarReglas({ reglas: [], criterios_baremo: [] });
  assert.match(sinEstado, /Fuente real no conectada/);
  assert.doesNotMatch(sinEstado, /No hay versiones autorizadas|Versiones:0|Publicadas:0/);
  assert.match(vista.renderizarReglas({ reglas: [], criterios_baremo: [] }, { fase: "listo" }), /No hay versiones autorizadas que mostrar/);
});

test("sin selección explícita presenta todas las versiones sin atribuirles una selección", () => {
  const html = crearVistaReglas(utilidades()).renderizarReglas({
    reglas: [
      { nombre: "Regla A", ambito: "Ámbito A", version: "v2", vigencia: "2026", estado: "Publicada" },
      { nombre: "Regla B", ambito: "Ámbito B", version: "v3", vigencia: "2027", estado: "Borrador" },
    ],
    criterios_baremo: [
      { id: "Criterio-v2", bloque: "A", criterio: "A", formula: "A", maximo: "1", version: "v2", estado: "Publicado" },
      { id: "Criterio-v3", bloque: "B", criterio: "B", formula: "B", maximo: "2", version: "v3", estado: "Borrador" },
    ],
  });
  assert.match(html, /Criterio-v2/);
  assert.match(html, /Criterio-v3/);
  assert.match(html, /Configuración de criterios/);
  assert.doesNotMatch(html, /versión seleccionada|Ámbito A · v2|Ámbito A · v3/i);
});

test("una fase explícita de fallo prevalece sobre filas antiguas", () => {
  const html = crearVistaReglas(utilidades()).renderizarReglas({
    reglas: [{ nombre: "Fila anterior", estado: "Publicada" }],
  }, { fase: "error", detalle: "Consulta fallida" });
  assert.match(html, /Consulta fallida/);
  assert.doesNotMatch(html, /Fila anterior/);
});

test("el borrador DEMO solo aparece en presentación", () => {
  const html = crearVistaReglas(utilidades(true)).renderizarReglas({ reglas: [], criterios_baremo: [] }, { fase: "listo" });
  assert.match(html, /Modo presentación/);
  assert.match(html, /desaparece al recargar/);
  assert.match(html, /data-operacion="guardar-reglas-baremo"/);
  for (const nombre of ["unidad_tiempo", "puntos_unidad", "fraccion_jornada", "tope_bloque", "ambito_experiencia", "redondeo", "desempate_1", "desempate_2", "desempate_3", "ultimo_recurso"]) assert.match(html, new RegExp(`name="${nombre}"`));
});

test("las utilidades reales exponen el modo DEMO y no habilitan edición fuera de él", () => {
  const modo = { demo: true };
  const reales = crearUtilidadesVista({
    escaparHTML: (valor) => String(valor ?? "").replace(/[&<>\"]/g, (caracter) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" })[caracter]),
    numero: (valor) => String(valor ?? 0),
    claseEstado: () => "neutro",
    encabezadoVista: (_miga, titulo, descripcion) => `<h1>${titulo}</h1><p>${descripcion}</p>`,
    esPresentacion: () => modo.demo,
    operacionPermitida: () => true,
  });
  const vista = crearVistaReglas(reales);
  const demo = vista.renderizarReglas({ reglas: [], criterios_baremo: [] }, { fase: "listo" });
  assert.match(demo, /Borrador DEMO de reglas/);
  assert.match(demo, /Guardar borrador DEMO/);
  modo.demo = false;
  const interna = vista.renderizarReglas({ reglas: [], criterios_baremo: [] }, { fase: "listo" });
  assert.match(interna, /Edición de versiones no conectada/);
  assert.match(interna, /disabled aria-disabled="true"/);
  assert.doesNotMatch(interna, /Borrador DEMO de reglas/);
});
