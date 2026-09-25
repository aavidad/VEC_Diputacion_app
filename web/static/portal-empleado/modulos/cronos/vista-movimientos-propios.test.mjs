import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { ErrorClienteSolicitudesCronos } from "./cliente-solicitudes-http.js";
import { marcasPorDiaCronos, montarMovimientosPropiosCronos, renderizarMovimientosPropiosCronos } from "./vista-movimientos-propios.js";

function datos(disponible = true) {
  return {
    periodo: { tipo: "rango", desde: "2026-01-01", hasta: "2026-12-31" },
    calendario: { disponible, dias: disponible ? [{ fecha: "2026-01-06", tipo: "festivo", nombre: "Epifanía" }] : [] },
    marcajes_por_dia: [{ fecha: "2026-09-21", marcajes: 4 }],
    absentismos: [{ solicitud_ref: "permiso:cronos:solicitud:c-0000001", permiso_ref: "permiso:cronos:traslado", nombre: "Traslado de domicilio", desde: "2026-09-22", hasta: "2026-09-23", cantidad: 2, unidad: "dia", pendiente_justificar: true }],
    correcciones: [{ solicitud_ref: "correccion:cronos:corr-0000001", fecha_civil: "2026-09-24", hora_pretendida: "08:00", movimiento: "entrada", estado: "pendiente_responsable", version: 1, solicitada_en: "2026-09-25T07:00:00Z" }],
  };
}
function raizFalsa() {
  const nodo = { dataset: {}, innerHTML: "", eventos: {}, addEventListener(tipo, fn) { this.eventos[tipo] = fn; },
    removeEventListener(tipo) { delete this.eventos[tipo]; }, remove() { this.eliminado = true; }, querySelector() { return null; } };
  return { nodo, raiz: { ownerDocument: { createElement: () => nodo }, append() {} } };
}
const esperar = async () => { for (let i = 0; i < 6; i++) await Promise.resolve(); };

test("marca cada día por tipo con los datos recibidos y dice en una línea si falta el calendario", () => {
  const marcas = marcasPorDiaCronos(datos());
  assert.deepEqual([...marcas.get("2026-09-22")], ["ausencia"]);
  assert.ok(marcas.get("2026-09-23").has("ausencia"));
  assert.ok(marcas.get("2026-09-21").has("marcaje"));
  assert.ok(marcas.get("2026-09-24").has("olvido"));
  assert.ok(marcas.get("2026-01-06").has("festivo"));
  const html = renderizarMovimientosPropiosCronos({ estado: "listo", anio: 2026, datos: datos(), hoy: "2026-09-25" });
  assert.match(html, /data-fecha="2026-01-06" data-tipos="festivo"/);
  assert.match(html, /Traslado de domicilio/);
  assert.match(html, /2 días/);
  assert.match(html, /Pendiente de justificar/);
  assert.match(html, /Pendiente de la jefatura/);
  assert.match(html, /data-accion="ayuda"/);
  assert.doesNotMatch(html, /no está publicado/);
  assert.doesNotMatch(html, /permiso:cronos|correccion:cronos|recibo:cronos|DEMO|sintétic/iu);
  const sin = renderizarMovimientosPropiosCronos({ estado: "listo", anio: 2026, datos: datos(false), hoy: "2026-09-25" });
  assert.equal((sin.match(/no está publicado/g) || []).length, 1);
  assert.doesNotMatch(sin, /data-fecha="2026-01-06" data-tipos/);
});

test("el olvido abre la solicitud, conserva la clave al reintentar y recarga el año tras registrarla", async () => {
  const { nodo, raiz } = raizFalsa();
  const envios = []; let fallar = true; let consultas = 0;
  const cliente = {
    consultarMovimientos: async () => { consultas++; return datos(); },
    solicitarCorreccion: async (entrada) => {
      envios.push(entrada);
      if (fallar) { fallar = false; throw new ErrorClienteSolicitudesCronos("servicio_no_disponible", 503); }
      return { solicitud_ref: `correccion:cronos:${entrada.clave_operacion}`, recibo_ref: "recibo:cronos:1", estado: "pendiente_responsable", version: 1, instante_utc: "2026-09-25T07:00:00Z", replay: false };
    },
  };
  const vista = montarMovimientosPropiosCronos({ raiz, cliente, anio: 2026 });
  await esperar();
  assert.match(nodo.innerHTML, /Comunicar un olvido/);
  nodo.eventos.click({ target: { closest: (sel) => (sel === "[data-cronos-olvido]" ? { dataset: { cronosOlvido: "abrir" } } : null) } });
  assert.match(nodo.innerHTML, /name="hora_pretendida"/);
  const formulario = { matches: () => true, elements: { namedItem: (n) => ({ value: { fecha_civil: "2026-09-24", hora_pretendida: "08:00", movimiento: "entrada" }[n] }) } };
  await nodo.eventos.submit({ target: formulario, preventDefault() {} });
  assert.match(nodo.innerHTML, /Puede reintentarla sin duplicarla/);
  await nodo.eventos.submit({ target: formulario, preventDefault() {} });
  await esperar();
  assert.equal(envios.length, 2);
  assert.equal(envios[0].clave_operacion, envios[1].clave_operacion);
  assert.deepEqual(Object.keys(envios[1]).sort(), ["clave_operacion", "fecha_civil", "hora_pretendida", "movimiento"]);
  assert.match(nodo.innerHTML, /Queda pendiente de la jefatura/);
  assert.ok(consultas >= 2);
  vista.desmontar();
  assert.equal(nodo.eliminado, true);
});

test("sin empleado y denegación muestran su motivo sin hechos anteriores", async () => {
  for (const [codigo, texto] of [["sin_empleado", /relación de empleo vigente/], ["acceso_denegado", /No tiene permiso/], ["servicio_no_disponible", /No se pudieron consultar/]]) {
    const { nodo, raiz } = raizFalsa();
    const vista = montarMovimientosPropiosCronos({ raiz, anio: 2026, cliente: { consultarMovimientos: async () => { throw new ErrorClienteSolicitudesCronos(codigo, 403); }, solicitarCorreccion: async () => ({}) } });
    await esperar();
    assert.match(nodo.innerHTML, texto);
    assert.doesNotMatch(nodo.innerHTML, /Traslado/);
    vista.desmontar();
  }
});

test("la vista no guarda nada en el navegador ni edita marcajes", async () => {
  const fuente = await readFile(new URL("./vista-movimientos-propios.js", import.meta.url), "utf8");
  assert.doesNotMatch(fuente, /localStorage|sessionStorage|indexedDB|document\.cookie|geolocation|Math\.random|marcajes\/remoto/u);
});

test("404 de la API: «no disponible» neutro, sin alerta; incrustada sin sobrelínea", async () => {
  const { nodo, raiz } = raizFalsa();
  const vista = montarMovimientosPropiosCronos({ raiz, anio: 2026, incrustada: true, cliente: {
    consultarMovimientos: async () => { throw new ErrorClienteSolicitudesCronos("servicio_no_disponible", 404); }, solicitarCorreccion: async () => ({}) } });
  await esperar();
  assert.match(nodo.innerHTML, /<p class="cronos-vacio" role="status">Esta consulta no está disponible\.<\/p>/u);
  assert.doesNotMatch(nodo.innerHTML, /role="alert"|No se pudieron consultar|sobrelinea|<h2/u);
  assert.match(nodo.innerHTML, /<h3 id="cronos-movpropios-titulo">/u);
  vista.desmontar();
});
