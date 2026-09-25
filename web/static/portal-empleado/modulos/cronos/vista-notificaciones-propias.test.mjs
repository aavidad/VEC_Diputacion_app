import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { ErrorClienteNotificacionesCronos } from "./cliente-notificaciones-http.js";
import { hoyCivilCronos, montarNotificacionesPropiasCronos, renderizarNotificacionesPropiasCronos } from "./vista-notificaciones-propias.js";

const TIPO = "notificacion:cronos:tipo:incidencia-marcaje:sintetico-1";
function datos() {
  return {
    tipos: [{ tipo_version_ref: TIPO, tipo_ref: "notificacion:cronos:tipo:incidencia-marcaje", nombre: "Incidencia en el marcaje" }],
    notificaciones: [
      { notificacion_ref: "notificacion:cronos:0f0e0d0c-0b0a-4000-8000-000000000001", tipo_ref: "notificacion:cronos:tipo:incidencia-marcaje", tipo_nombre: "Incidencia en el marcaje",
        fecha_referida: "2026-09-24", texto: "No pude fichar <la salida>.", adjunto_ref: "registro:sintetico:0001", adjunto_sha256: "a".repeat(64),
        registrada_en: "2026-09-24T08:00:00Z", estado: "atendida", atendida_en: "2026-09-25T08:00:00Z" },
      { notificacion_ref: "notificacion:cronos:0f0e0d0c-0b0a-4000-8000-000000000002", tipo_ref: "notificacion:cronos:tipo:incidencia-marcaje", tipo_nombre: "Incidencia en el marcaje",
        fecha_referida: "2026-09-23", texto: "Otra", registrada_en: "2026-09-23T08:00:00Z", estado: "registrada" },
    ],
  };
}
function raizFalsa() {
  const nodos = {};
  const nodo = { dataset: {}, innerHTML: "", eventos: {}, addEventListener(tipo, fn) { this.eventos[tipo] = fn; },
    removeEventListener(tipo) { delete this.eventos[tipo]; }, remove() { this.eliminado = true; }, querySelector: (sel) => nodos[sel] ?? null };
  return { nodo, nodos, raiz: { ownerDocument: { createElement: () => nodo }, append() {} } };
}
const esperar = async () => { for (let i = 0; i < 10; i++) await Promise.resolve(); };
const formulario = (valores) => ({ matches: () => true, elements: { namedItem: (n) => (n in valores ? { value: valores[n] } : null) } });

test("la vista muestra el formulario y las notificaciones con su estado, escapadas y sin referencias internas", () => {
  const html = renderizarNotificacionesPropiasCronos({ estado: "listo", datos: datos(), hoy: "2026-09-25" });
  assert.match(html, /Notificaciones a RRHH/);
  assert.match(html, /Nueva notificación/);
  assert.match(html, /<option value="notificacion:cronos:tipo:incidencia-marcaje:sintetico-1">Incidencia en el marcaje<\/option>/);
  assert.match(html, /value="2026-09-25"/);
  assert.match(html, /0 de 512 caracteres/);
  assert.match(html, /Atendida el/);
  assert.match(html, /Pendiente de atender/);
  assert.match(html, /No pude fichar &lt;la salida&gt;\./);
  assert.match(html, /registro:sintetico:0001/);
  assert.match(html, /data-accion="ayuda"/);
  assert.doesNotMatch(html, />[^<]*(emp_|notificacion:cronos:[0-9a-f]|recibo:cronos|AD3|DEMO)[^<]*</u);
  assert.match(renderizarNotificacionesPropiasCronos({ estado: "listo", datos: { tipos: [], notificaciones: [] } }), /No hay tipos de notificación disponibles/);
});

test("enviar valida antes, calcula la huella, conserva la clave en el reintento y recarga", async () => {
  const { nodo, nodos, raiz } = raizFalsa(); const envios = []; let consultas = 0;
  let respuesta = new ErrorClienteNotificacionesCronos("servicio_no_disponible", 503);
  const cliente = {
    consultarPropias: async () => { consultas++; return datos(); },
    enviar: async (e) => { envios.push(e); if (respuesta instanceof Error) throw respuesta; return respuesta; },
  };
  const cripto = { subtle: { digest: async () => new Uint8Array(32).fill(0xab).buffer } };
  montarNotificacionesPropiasCronos({ raiz, cliente, cripto, ahora: () => new Date("2026-09-25T10:00:00Z") });
  await esperar();
  await nodo.eventos.submit({ target: formulario({ tipo: TIPO, fecha: "2026-09-24", texto: "   ", referencia: "" }), preventDefault() {} });
  assert.match(nodo.innerHTML, /Revise el tipo, la fecha y el mensaje/);
  await nodo.eventos.submit({ target: formulario({ tipo: TIPO, fecha: "2026-09-24", texto: "Olvidé fichar", referencia: "registro:sintetico:0001" }), preventDefault() {} });
  assert.match(nodo.innerHTML, /Indique la referencia y elija el documento/);
  assert.equal(envios.length, 0, "sin datos válidos no se envía");
  nodos["[data-cronos-documento-estado]"] = { textContent: "", dataset: {} };
  await nodo.eventos.change({ target: { name: "documento", files: [{ size: 3, arrayBuffer: async () => new Uint8Array([1, 2, 3]).buffer }] } });
  assert.equal(nodos["[data-cronos-documento-estado]"].textContent, "Huella del documento calculada.");
  await nodo.eventos.submit({ target: formulario({ tipo: TIPO, fecha: "2026-09-24", texto: "Olvidé fichar", referencia: "registro:sintetico:0001" }), preventDefault() {} });
  assert.match(nodo.innerHTML, /No se pudo enviar la notificación/);
  assert.deepEqual(envios[0], { clave_operacion: envios[0].clave_operacion, tipo_version_ref: TIPO, fecha_referida: "2026-09-24", texto: "Olvidé fichar",
    adjunto_ref: "registro:sintetico:0001", adjunto_sha256: "ab".repeat(32) });
  respuesta = { notificacion_ref: "notificacion:cronos:0f0e0d0c-0b0a-4000-8000-000000000003", recibo_ref: "recibo:cronos:1", instante_utc: "2026-09-25T08:00:00Z", replay: false };
  await nodo.eventos.submit({ target: formulario({ tipo: TIPO, fecha: "2026-09-24", texto: "Olvidé fichar", referencia: "registro:sintetico:0001" }), preventDefault() {} });
  await esperar();
  assert.equal(envios[1].clave_operacion, envios[0].clave_operacion, "un reintento conserva la clave");
  assert.match(nodo.innerHTML, /Notificación enviada a RRHH/);
  assert.equal(consultas, 2, "tras enviar se recarga la lista");
  await nodo.eventos.submit({ target: formulario({ tipo: TIPO, fecha: "2026-09-24", texto: "Otra cosa", referencia: "" }), preventDefault() {} });
  assert.notEqual(envios[2].clave_operacion, envios[0].clave_operacion, "otro envío usa otra clave");
});

test("un tipo que ya no está vigente recarga la lista con aviso", async () => {
  const { nodo, raiz } = raizFalsa(); let consultas = 0;
  const cliente = { consultarPropias: async () => { consultas++; return datos(); }, enviar: async () => { throw new ErrorClienteNotificacionesCronos("tipo_no_vigente", 409); } };
  montarNotificacionesPropiasCronos({ raiz, cliente });
  await esperar();
  await nodo.eventos.submit({ target: formulario({ tipo: TIPO, fecha: "2026-09-24", texto: "Texto", referencia: "" }), preventDefault() {} });
  await esperar();
  assert.equal(consultas, 2);
  assert.match(nodo.innerHTML, /Ese tipo ya no está disponible/);
});

test("sin permiso o sin empleado no muestra datos", async () => {
  for (const [codigo, texto] of [["acceso_denegado", /No tiene permiso/], ["sin_empleado", /relación de empleo vigente/]]) {
    const { nodo, raiz } = raizFalsa();
    const vista = montarNotificacionesPropiasCronos({ raiz, cliente: { consultarPropias: async () => { throw new ErrorClienteNotificacionesCronos(codigo, 403); }, enviar: async () => ({}) } });
    await esperar();
    assert.match(nodo.innerHTML, texto);
    assert.doesNotMatch(nodo.innerHTML, /data-cronos-notificacion-formulario/);
    vista.desmontar();
    assert.equal(nodo.eliminado, true);
  }
});

test("la fecha de hoy es la civil de Madrid y la vista no guarda nada en el navegador", async () => {
  assert.equal(hoyCivilCronos(new Date("2026-09-24T22:30:00Z"), "Europe/Madrid"), "2026-09-25");
  const fuente = await readFile(new URL("./vista-notificaciones-propias.js", import.meta.url), "utf8");
  assert.doesNotMatch(fuente, /localStorage|sessionStorage|indexedDB|document\.cookie|Math\.random|FormData|querySelectorAll/u);
});
