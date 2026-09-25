import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { ErrorClienteNotificacionesCronos } from "./cliente-notificaciones-http.js";
import { montarBandejaNotificacionesCronos, renderizarBandejaNotificacionesCronos } from "./vista-bandeja-notificaciones.js";

const REF = "notificacion:cronos:0f0e0d0c-0b0a-4000-8000-000000000001";
function bandeja() {
  return { notificaciones: [
    { notificacion_ref: REF, empleado_ref: "emp_AAAAAAAAAAAAAAAAAAAAAA", empleado_etiqueta: "Persona sintética A", tipo_ref: "notificacion:cronos:tipo:otra-comunicacion",
      tipo_nombre: "Otra comunicación a RRHH", fecha_referida: "2026-09-24", texto: "Mensaje\nen dos líneas", adjunto_ref: "registro:sintetico:0001", adjunto_sha256: "a".repeat(64),
      registrada_en: "2026-09-24T08:00:00Z", atendida: false },
    { notificacion_ref: "notificacion:cronos:0f0e0d0c-0b0a-4000-8000-000000000002", empleado_ref: "emp_CCCCCCCCCCCCCCCCCCCCCC", empleado_etiqueta: "",
      tipo_ref: "notificacion:cronos:tipo:otra-comunicacion", tipo_nombre: "Otra comunicación a RRHH", fecha_referida: "2026-09-20", texto: "Ya atendida",
      registrada_en: "2026-09-20T08:00:00Z", atendida: true, atendida_en: "2026-09-21T08:00:00Z" },
  ] };
}
function raizFalsa() {
  const nodo = { dataset: {}, innerHTML: "", eventos: {}, addEventListener(tipo, fn) { this.eventos[tipo] = fn; },
    removeEventListener(tipo) { delete this.eventos[tipo]; }, remove() { this.eliminado = true; }, querySelector() { return null; } };
  return { nodo, raiz: { ownerDocument: { createElement: () => nodo }, append() {} } };
}
const esperar = async () => { for (let i = 0; i < 8; i++) await Promise.resolve(); };
const pulsar = (selector, dataset) => ({ target: { closest: (sel) => (sel === selector ? { dataset, disabled: false } : null) } });

test("la bandeja separa pendientes y atendidas y no muestra referencias internas", () => {
  const pendientes = renderizarBandejaNotificacionesCronos({ estado: "listo", datos: bandeja() });
  assert.match(pendientes, /Notificaciones recibidas/);
  assert.match(pendientes, /Persona sintética A/);
  assert.match(pendientes, /Marcar atendida/);
  assert.match(pendientes, /registro:sintetico:0001/);
  assert.doesNotMatch(pendientes, /Ya atendida/);
  assert.match(pendientes, /data-accion="ayuda"/);
  assert.doesNotMatch(pendientes, />[^<]*(emp_|notificacion:cronos:[0-9a-f]|recibo:cronos|AD3)[^<]*</u);
  const atendidas = renderizarBandejaNotificacionesCronos({ estado: "listo", filtro: "atendidas", datos: bandeja() });
  assert.match(atendidas, /Sin nombre publicado/);
  assert.match(atendidas, /Atendida el/);
  assert.doesNotMatch(atendidas, /data-cronos-atender=/);
  assert.match(renderizarBandejaNotificacionesCronos({ estado: "listo", datos: { notificaciones: [] } }), /No hay notificaciones pendientes de atender/);
});

test("marcar atendida conserva la clave en el reintento, recarga y trata la ya atendida como hecha", async () => {
  const { nodo, raiz } = raizFalsa(); const envios = []; let consultas = 0;
  let respuesta = new ErrorClienteNotificacionesCronos("servicio_no_disponible", 503);
  const cliente = {
    consultarBandeja: async () => { consultas++; return bandeja(); },
    atender: async (e) => { envios.push(e); if (respuesta instanceof Error) throw respuesta; return respuesta; },
  };
  const vista = montarBandejaNotificacionesCronos({ raiz, cliente });
  await esperar();
  nodo.eventos.click(pulsar("[data-cronos-atender]", { cronosAtender: REF }));
  await esperar();
  assert.match(nodo.innerHTML, /No se pudo marcar como atendida/);
  respuesta = { atencion_ref: "x", notificacion_ref: REF, recibo_ref: "recibo:cronos:1", instante_utc: "2026-09-25T08:00:00Z", replay: false };
  nodo.eventos.click(pulsar("[data-cronos-atender]", { cronosAtender: REF }));
  await esperar();
  assert.equal(envios[1].clave_operacion, envios[0].clave_operacion, "un reintento conserva la clave");
  assert.match(nodo.innerHTML, /Notificación marcada como atendida/);
  assert.equal(consultas, 2);
  respuesta = new ErrorClienteNotificacionesCronos("estado_cambiado", 409);
  nodo.eventos.click(pulsar("[data-cronos-atender]", { cronosAtender: REF }));
  await esperar();
  assert.match(nodo.innerHTML, /ya estaba atendida/);
  nodo.eventos.click(pulsar("[data-cronos-filtro-notificaciones]", { cronosFiltroNotificaciones: "atendidas" }));
  assert.match(nodo.innerHTML, /aria-pressed="true">Atendidas/);
  vista.desmontar();
  assert.equal(nodo.eliminado, true);
});

test("sin permiso o sin empleado no muestra notificaciones", async () => {
  for (const [codigo, texto] of [["no_competente", /No tiene permiso/], ["sin_empleado", /relación de empleo vigente/]]) {
    const { nodo, raiz } = raizFalsa();
    montarBandejaNotificacionesCronos({ raiz, cliente: { consultarBandeja: async () => { throw new ErrorClienteNotificacionesCronos(codigo, 403); }, atender: async () => ({}) } });
    await esperar();
    assert.match(nodo.innerHTML, texto);
    assert.doesNotMatch(nodo.innerHTML, /data-cronos-atender/);
  }
});

test("una bandeja de más de 500 se avisa sin códigos y sin mostrar una lista recortada", async () => {
  const { nodo, raiz } = raizFalsa(); const anuncios = [];
  montarBandejaNotificacionesCronos({ raiz, anunciar: (m) => anuncios.push(m),
    cliente: { consultarBandeja: async () => { throw new ErrorClienteNotificacionesCronos("bandeja_demasiado_grande", 409); }, atender: async () => ({}) } });
  await esperar();
  assert.match(nodo.innerHTML, /Hay más de 500 notificaciones/);
  assert.match(nodo.innerHTML, /role="alert"/);
  assert.doesNotMatch(nodo.innerHTML, /PC013|bandeja_demasiado_grande|409|data-cronos-atender/u);
  assert.match(anuncios[0], /Hay más de 500 notificaciones/);
});

test("la huella del documento se ofrece en un detalle desplegable, no sólo en un title", () => {
  const html = renderizarBandejaNotificacionesCronos({ estado: "listo", datos: bandeja() });
  assert.match(html, /<details class="cronos-huella"><summary>Huella<\/summary><span class="cronos-huella-valor">Huella SHA-256: a{64}<\/span><\/details>/u);
  assert.doesNotMatch(html, /title="Huella/u);
});

test("la vista no guarda nada en el navegador", async () => {
  const fuente = await readFile(new URL("./vista-bandeja-notificaciones.js", import.meta.url), "utf8");
  assert.doesNotMatch(fuente, /localStorage|sessionStorage|indexedDB|document\.cookie|Math\.random|querySelectorAll/u);
});
