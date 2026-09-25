import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { ErrorClienteSolicitudesCronos } from "./cliente-solicitudes-http.js";
import { montarPermisosPropiosCronos, renderizarPermisosPropiosCronos } from "./vista-permisos-propios.js";

function permiso(ref, nombre, extra = {}) {
  return { permiso_ref: `permiso:cronos:${ref}`, version_ref: `catalogo:cronos:${ref}:v1`, nombre, unidad: "dia", computo: "laborables", circuito: "A",
    minimo: 1, maximo_solicitud: null, maximo_mensual: null, maximo_anual: 6, justificante_exigido: false, solicitable: true, sintetico: true,
    solicitado: 0, concedido: 0, pendiente_justificar: 0, resta: 6, ...extra };
}
function datos() {
  return {
    anio: 2026,
    permisos: [
      permiso("asuntos-propios", "Asuntos propios", { solicitado: 1, concedido: 2, resta: 3 }),
      permiso("horas-medico", "Horas de médico", { unidad: "hora", circuito: "J-A", minimo: 15, maximo_anual: null, maximo_solicitud: 180, resta: null }),
      permiso("horas-sindicales", "Horas sindicales", { unidad: "hora", minimo: 15, maximo_anual: null, maximo_mensual: 3600, resta: null }),
      permiso("nacimiento", "Nacimiento", { computo: "naturales", solicitable: false, maximo_anual: 7, resta: 7 }),
    ],
    solicitudes: [
      { solicitud_ref: "permiso:cronos:solicitud:a-1", catalogo_version_ref: "catalogo:cronos:asuntos-propios:v1", permiso_ref: "permiso:cronos:asuntos-propios", desde: "2026-03-02", hasta: "2026-03-03", cantidad: 2, unidad: "dia", estado: "concedido", version: 2, pendiente_justificar: true, solicitada_en: "2026-02-20T08:00:00Z" },
      { solicitud_ref: "permiso:cronos:solicitud:a-2", catalogo_version_ref: "catalogo:cronos:asuntos-propios:v1", permiso_ref: "permiso:cronos:asuntos-propios", desde: "2026-10-05", hasta: "2026-10-05", cantidad: 1, unidad: "dia", estado: "solicitado", version: 1, pendiente_justificar: false, solicitada_en: "2026-09-25T08:00:00Z" },
      { solicitud_ref: "permiso:cronos:solicitud:m-1", catalogo_version_ref: "catalogo:cronos:horas-medico:v1", permiso_ref: "permiso:cronos:horas-medico", desde: "2026-10-07", hasta: "2026-10-07", hora_inicio: "09:00", hora_fin: "11:30", cantidad: 150, unidad: "hora", estado: "solicitado", version: 1, pendiente_justificar: false, solicitada_en: "2026-09-25T08:00:00Z" },
    ],
  };
}
function raizFalsa() {
  const nodo = { dataset: {}, innerHTML: "", eventos: {}, addEventListener(tipo, fn) { this.eventos[tipo] = fn; },
    removeEventListener(tipo) { delete this.eventos[tipo]; }, remove() { this.eliminado = true; }, querySelector() { return null; } };
  return { nodo, raiz: { ownerDocument: { createElement: () => nodo }, append() {} } };
}
const esperar = async () => { for (let i = 0; i < 6; i++) await Promise.resolve(); };

test("listado anual con máximo, mínimo, solicitado, concedido y resta en días u horas y minutos", () => {
  const html = renderizarPermisosPropiosCronos({ estado: "listo", anio: 2026, datos: datos() });
  assert.match(html, /Asuntos propios/);
  assert.match(html, /6 días/);
  assert.match(html, /3 días/);
  assert.match(html, /3 h por solicitud/);
  assert.match(html, /60 h al mes/);
  assert.match(html, /15 min/);
  assert.match(html, /2 h 30 min/);
  assert.match(html, /Jefatura y administración/);
  assert.match(html, /Cuantías pendientes de confirmar por RRHH/);
  assert.equal((html.match(/data-cronos-solicitar=/g) || []).length, 3, "el permiso no solicitable no ofrece Solicitar");
  assert.match(html, /Pendiente de la jefatura/);
  assert.match(html, /Pendiente de administración/);
  assert.match(html, /Pendientes de justificar/);
  assert.doesNotMatch(html, /catalogo:cronos|solicitud:a-|Conceder|Denegar|DEMO/u);
  const reales = datos(); reales.permisos.forEach((p) => { p.sintetico = false; });
  assert.doesNotMatch(renderizarPermisosPropiosCronos({ estado: "listo", anio: 2026, datos: reales }), /pendientes de confirmar/);
});

test("solicitar un permiso por horas envía un solo día con su tramo y muestra el rechazo nominal", async () => {
  const { nodo, raiz } = raizFalsa(); const envios = []; let respuesta = new ErrorClienteSolicitudesCronos("fuera_de_limites", 422);
  const cliente = {
    consultarPermisos: async () => datos(),
    solicitarPermiso: async (e) => { envios.push(e); if (respuesta instanceof Error) throw respuesta; return respuesta; },
  };
  const vista = montarPermisosPropiosCronos({ raiz, cliente, anio: 2026 });
  await esperar();
  nodo.eventos.click({ target: { closest: (sel) => (sel === "[data-cronos-solicitar]" ? { dataset: { cronosSolicitar: "permiso:cronos:horas-medico" } } : null) } });
  assert.match(nodo.innerHTML, /name="hora_inicio"/);
  assert.doesNotMatch(nodo.innerHTML, /name="hasta"/);
  const formulario = { matches: () => true, elements: { namedItem: (n) => ({ value: { desde: "2026-10-08", hora_inicio: "09:00", hora_fin: "10:00" }[n] }) } };
  await nodo.eventos.submit({ target: formulario, preventDefault() {} });
  assert.match(nodo.innerHTML, /supera el máximo/);
  assert.deepEqual(envios[0], { clave_operacion: envios[0].clave_operacion, permiso_ref: "permiso:cronos:horas-medico", desde: "2026-10-08", hasta: "2026-10-08", hora_inicio: "09:00", hora_fin: "10:00" });
  respuesta = { solicitud_ref: "x", recibo_ref: "recibo:cronos:1", estado: "solicitado", version: 1, cantidad: 60, unidad: "hora", instante_utc: "2026-09-25T08:00:00Z", replay: false };
  await nodo.eventos.submit({ target: formulario, preventDefault() {} });
  await esperar();
  assert.equal(envios[1].clave_operacion, envios[0].clave_operacion, "un reintento conserva la clave");
  assert.match(nodo.innerHTML, /Solicitud registrada: 1 h\. Queda pendiente de conceder \(Jefatura y administración\)/);
  vista.desmontar();
  assert.equal(nodo.eliminado, true);
});

test("sin empleado o sin servicio no muestra cifras", async () => {
  for (const [codigo, texto] of [["sin_empleado", /relación de empleo vigente/], ["servicio_no_disponible", /No se pudieron consultar/]]) {
    const { nodo, raiz } = raizFalsa();
    const vista = montarPermisosPropiosCronos({ raiz, anio: 2026, cliente: { consultarPermisos: async () => { throw new ErrorClienteSolicitudesCronos(codigo, 503); }, solicitarPermiso: async () => ({}) } });
    await esperar();
    assert.match(nodo.innerHTML, texto);
    assert.doesNotMatch(nodo.innerHTML, /días|tarjeta-kpi/);
    vista.desmontar();
  }
});

test("la vista no guarda nada en el navegador ni concede permisos", async () => {
  const fuente = await readFile(new URL("./vista-permisos-propios.js", import.meta.url), "utf8");
  assert.doesNotMatch(fuente, /localStorage|sessionStorage|indexedDB|document\.cookie|Math\.random|decidir|conceder\(/u);
});
