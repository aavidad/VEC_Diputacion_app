import assert from "node:assert/strict";
import test from "node:test";
import { obtenerDatosPresentacion } from "./datos-presentacion.js";
import { crearUtilidadesVista } from "./portal-vistas-utilidades.js";
import { crearVistasGobierno } from "./portal-vistas-gobierno.js";

function utilidades(modoPresentacion = true) {
  return crearUtilidadesVista({
    escaparHTML: String,
    numero: String,
    claseEstado: () => "neutro",
    encabezadoVista: (_s, t, d, a = "") => "<h2>" + t + "</h2><p>" + d + "</p>" + a,
    esPresentacion: () => modoPresentacion,
    operacionPermitida: () => true,
  });
}

test("gobierno conserva simulaciones DEMO y solo bloquea la activación oficial", () => {
  const vistas = crearVistasGobierno(utilidades());
  const datos = obtenerDatosPresentacion();
  const estadisticas = vistas.renderizarEstadisticas(datos);
  const auditoria = vistas.renderizarAuditoria(datos);
  const configuracion = vistas.renderizarConfiguracion(datos);
  for (const salida of [estadisticas, auditoria, configuracion]) assert.match(salida, /data-accion="operacion-presentacion"/);
  assert.match(estadisticas, /La acción genera un recibo, no un archivo/);
  assert.match(auditoria, /Muestra de actuaciones DEMO/);
  assert.match(configuracion, /No se modifica ninguna identidad ni autorización/);
  assert.match(configuracion, /data-accion="bloqueo-presentacion"/);
  assert.match(configuracion, /Activar rol en directorio/);
});

test("auditoría muestra hora UTC legible y cuenta los eventos sin recibo que existen", () => {
  const datos = structuredClone(obtenerDatosPresentacion());
  datos.auditoria_eventos.push({ referencia: " ", instante: "fecha no válida", actor: "", operacion: "consulta", objetivo: "DEMO-OBJ-1", resultado: "Pendiente", efectos_reales: false });
  const salida = crearVistasGobierno(utilidades()).renderizarAuditoria(datos);
  assert.match(salida, /<time datetime="2026-07-17T08:15:00\.000Z">17\/7\/26, 8:15 UTC<\/time>/);
  assert.doesNotMatch(salida, />2026-07-17T08:15:00Z</);
  assert.doesNotMatch(salida, /fecha no válida/);
  assert.match(salida, /Sin fecha/);
  assert.match(salida, /<strong class="valor-kpi">1<\/strong><span class="etiqueta-kpi">Eventos sin recibo/);
  assert.match(salida, /<span class="estado-chip aviso">Sin recibo<\/span>/);
});

test("la hora UTC no cambia en el salto al horario de verano de Madrid", () => {
  const datos = structuredClone(obtenerDatosPresentacion());
  datos.auditoria_eventos[0].instante = "2026-03-29T02:30:00Z";
  const salida = crearVistasGobierno(utilidades()).renderizarAuditoria(datos);
  assert.match(salida, /<time datetime="2026-03-29T02:30:00\.000Z">29\/3\/26, 2:30 UTC<\/time>/);
  assert.doesNotMatch(salida, />29\/3\/26, 3:30 UTC</);
});

test("una fecha imposible no se normaliza a otro día", () => {
  const datos = structuredClone(obtenerDatosPresentacion());
  datos.auditoria_eventos[0].instante = "2026-02-31T10:00:00Z";
  const salida = crearVistasGobierno(utilidades()).renderizarAuditoria(datos);
  assert.match(salida, /<td data-columna="fecha">Sin fecha<\/td>/);
  assert.doesNotMatch(salida, /2026-02-31|3\/3\/26, 10:00 UTC/);
});

test("tablas vacías de gobierno siguen visibles, enfocables y con columnas de estado", () => {
  const datos = structuredClone(obtenerDatosPresentacion());
  datos.bolsas = [];
  datos.indicadores = {};
  datos.auditoria_eventos = [];
  datos.roles_demo = [];
  datos.configuraciones_demo = [];
  const vistas = crearVistasGobierno(utilidades());
  const salidas = [vistas.renderizarEstadisticas(datos), vistas.renderizarAuditoria(datos), vistas.renderizarConfiguracion(datos)];
  assert.equal(salidas.reduce((n, salida) => n + (salida.match(/class="vacio-controlado"/g) || []).length, 0), 4);
  assert.equal(salidas.reduce((n, salida) => n + (salida.match(/tabindex="0" role="region"/g) || []).length, 0), 4);
  assert.equal(salidas.reduce((n, salida) => n + (salida.match(/data-columna="estado"/g) || []).length, 0), 4);
  assert.match(salidas[0], /<strong class="valor-kpi">No disponible<\/strong><span class="etiqueta-kpi">Cobertura media/);
  assert.doesNotMatch(salidas[0], /NaN|undefined|>0 %</);
  assert.match(salidas[1], /<strong class="valor-kpi">0<\/strong><span class="etiqueta-kpi">Eventos sin recibo/);
  assert.match(crearVistasGobierno(utilidades(false)).renderizarAuditoria(datos), /<strong class="valor-kpi">No disponible<\/strong><span class="etiqueta-kpi">Efectos reales/);
});
