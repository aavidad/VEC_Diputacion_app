import assert from "node:assert/strict";
import test from "node:test";
import { obtenerDatosPresentacion } from "./datos-presentacion.js";
import { crearUtilidadesVista } from "./portal-vistas-utilidades.js";
import { crearVistasGobierno } from "./portal-vistas-gobierno.js";

function utilidades() {
  return crearUtilidadesVista({
    escaparHTML: String,
    numero: String,
    claseEstado: () => "neutro",
    encabezadoVista: (_s, t, d, a = "") => "<h2>" + t + "</h2><p>" + d + "</p>" + a,
    esPresentacion: () => true,
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
