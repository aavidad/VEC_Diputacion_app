import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { ErrorClienteNotificacionesCronos } from "./cliente-notificaciones-http.js";
import { montarBandejaNotificacionesCronos, renderizarBandejaNotificacionesCronos } from "./vista-bandeja-notificaciones.js";
import { cargarTextos } from "../../../comun/textos.js";

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
const diferida = () => { let resolver; let rechazar; const promesa = new Promise((resolve, reject) => { resolver = resolve; rechazar = reject; }); return { promesa, resolver, rechazar }; };

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

test("marcar atendida conserva la clave en el reintento y no vuelve a decidir una atención confirmada", async () => {
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
  nodo.eventos.click(pulsar("[data-cronos-atender]", { cronosAtender: REF }));
  await esperar();
  assert.equal(envios.length, 2, "el recibo confirmado impide otra decisión aunque llegue una lista anterior");
  nodo.eventos.click(pulsar("[data-cronos-filtro-notificaciones]", { cronosFiltroNotificaciones: "atendidas" }));
  assert.match(nodo.innerHTML, /aria-pressed="true">Atendidas/);
  vista.desmontar();
  assert.equal(nodo.eliminado, true);
});

test("POST confirmado y GET fallido conservan recibo y filtro; recuperar sólo consulta", async () => {
  const { nodo, raiz } = raizFalsa(); const envios = []; const lectura = diferida(); const anuncios = []; let consultas = 0;
  const recibo = { atencion_ref: "atencion:sintetica", notificacion_ref: REF, recibo_ref: "recibo:cronos:00000000-0000-4000-8000-000000000001", instante_utc: "2026-09-25T08:00:00Z", replay: false };
  const vista = montarBandejaNotificacionesCronos({ raiz, anunciar: (m) => anuncios.push(m), cliente: {
    consultarBandeja: async () => { consultas++; if (consultas === 2) throw new ErrorClienteNotificacionesCronos("servicio_no_disponible", 503); if (consultas === 3) return lectura.promesa; return bandeja(); },
    atender: async (entrada) => { envios.push(entrada); return recibo; },
  } });
  await esperar();
  nodo.eventos.click(pulsar("[data-cronos-atender]", { cronosAtender: REF })); await esperar();
  assert.match(nodo.innerHTML, /data-estado="error"/);
  assert.match(nodo.innerHTML, /Notificación marcada como atendida/);
  assert.match(nodo.innerHTML, /No se pudo actualizar la lista/);
  assert.match(nodo.innerHTML, /data-cronos-reintentar-bandeja/);
  assert.match(nodo.innerHTML, new RegExp(recibo.recibo_ref));
  assert.match(nodo.innerHTML, /25 sept 2026/u);
  assert.doesNotMatch(nodo.innerHTML, /data-cronos-atender=/);
  nodo.eventos.click(pulsar("[data-cronos-filtro-notificaciones]", { cronosFiltroNotificaciones: "atendidas" }));
  nodo.eventos.click(pulsar("[data-cronos-reintentar-bandeja]", {}));
  assert.match(nodo.innerHTML, /data-estado="cargando"/);
  assert.match(nodo.innerHTML, /Notificación marcada como atendida/);
  assert.match(nodo.innerHTML, new RegExp(recibo.recibo_ref));
  assert.doesNotMatch(nodo.innerHTML, /data-cronos-reintentar-bandeja/);
  lectura.resolver(bandeja()); await esperar();
  assert.match(nodo.innerHTML, /data-estado="listo"/);
  assert.match(nodo.innerHTML, /aria-pressed="true">Atendidas/);
  assert.match(nodo.innerHTML, new RegExp(recibo.recibo_ref));
  assert.equal(envios.length, 1);
  assert.equal(consultas, 3);
  assert.match(anuncios[0], /Notificación marcada como atendida/);
  assert.match(anuncios[1], /No se pudo actualizar la lista/);
  vista.desmontar();
});

test("el fallo inicial ofrece recuperación visible y una ya atendida conserva la confirmación sin inventar recibo", async () => {
  const { nodo, raiz } = raizFalsa(); let consultas = 0; let envios = 0;
  const vista = montarBandejaNotificacionesCronos({ raiz, cliente: {
    consultarBandeja: async () => { consultas++; if ([1, 3].includes(consultas)) throw new Error("sin conexión"); return bandeja(); },
    atender: async () => { envios++; throw new ErrorClienteNotificacionesCronos("estado_cambiado", 409); },
  } });
  await esperar();
  assert.match(nodo.innerHTML, /data-cronos-reintentar-bandeja/);
  nodo.eventos.click(pulsar("[data-cronos-reintentar-bandeja]", {})); await esperar();
  assert.match(nodo.innerHTML, /Persona sintética A/);
  assert.equal(envios, 0);
  nodo.eventos.click(pulsar("[data-cronos-atender]", { cronosAtender: REF })); await esperar();
  assert.match(nodo.innerHTML, /ya estaba atendida/);
  assert.match(nodo.innerHTML, /data-cronos-reintentar-bandeja/);
  assert.doesNotMatch(nodo.innerHTML, /Ver recibo de la atención/);
  nodo.eventos.click(pulsar("[data-cronos-reintentar-bandeja]", {})); await esperar();
  nodo.eventos.click(pulsar("[data-cronos-atender]", { cronosAtender: REF })); await esperar();
  assert.equal(envios, 1);
  vista.desmontar();
});

test("recargar cancela la lectura anterior y descarta su respuesta aunque el cliente ignore el aborto", async () => {
  const { nodo, raiz } = raizFalsa(); const primera = diferida(); const segunda = diferida(); const senales = []; const anuncios = [];
  const vista = montarBandejaNotificacionesCronos({ raiz, anunciar: (m) => anuncios.push(m), cliente: {
    consultarBandeja: ({ signal }) => { senales.push(signal); return senales.length === 1 ? primera.promesa : segunda.promesa; }, atender: async () => ({}),
  } });
  const nueva = vista.recargar();
  assert.equal(senales[0].aborted, true);
  segunda.resolver({ notificaciones: [] }); await nueva;
  const actual = nodo.innerHTML;
  primera.resolver(bandeja()); await esperar();
  assert.equal(nodo.innerHTML, actual);
  assert.deepEqual(anuncios, []);
  vista.desmontar();
  assert.equal(senales[1].aborted, true);
  await vista.recargar();
  assert.equal(senales.length, 2, "una vista desmontada no inicia consultas");
});

test("el reintento devuelve el foco a la acción de recuperar si falla o al filtro seleccionado si tiene éxito", async () => {
  const { nodo, raiz } = raizFalsa(); let consultas = 0; let envios = 0;
  nodo.querySelector = (selector) => ({ selector, focus() { raiz.ownerDocument.activeElement = this; } });
  const vista = montarBandejaNotificacionesCronos({ raiz, cliente: {
    consultarBandeja: async () => { consultas++; if (consultas < 3) throw new Error("sin conexión"); return bandeja(); },
    atender: async () => { envios++; return {}; },
  } });
  await esperar();
  nodo.eventos.click(pulsar("[data-cronos-filtro-notificaciones]", { cronosFiltroNotificaciones: "atendidas" }));
  nodo.eventos.click(pulsar("[data-cronos-reintentar-bandeja]", {})); await esperar();
  assert.equal(raiz.ownerDocument.activeElement.selector, "[data-cronos-reintentar-bandeja]");
  nodo.eventos.click(pulsar("[data-cronos-reintentar-bandeja]", {})); await esperar();
  assert.equal(raiz.ownerDocument.activeElement.selector, '[data-cronos-filtro-notificaciones="atendidas"]');
  assert.equal(envios, 0); assert.equal(consultas, 3);
  vista.desmontar();
});

test("desmontar aborta e ignora decisiones y lecturas tardías sin anunciar ni consultar otra vez", async () => {
  for (const fase of ["lectura", "decision"]) {
    const { nodo, raiz } = raizFalsa(); const pendiente = diferida(); const anuncios = []; let consultas = 0; let senal;
    const vista = montarBandejaNotificacionesCronos({ raiz, anunciar: (m) => anuncios.push(m), cliente: {
      consultarBandeja: ({ signal }) => { consultas++; if (fase === "lectura") { senal = signal; return pendiente.promesa; } return Promise.resolve(bandeja()); },
      atender: (_entrada, { signal }) => { senal = signal; return pendiente.promesa; },
    } });
    await esperar();
    if (fase === "decision") nodo.eventos.click(pulsar("[data-cronos-atender]", { cronosAtender: REF }));
    const html = nodo.innerHTML; vista.desmontar();
    assert.equal(senal.aborted, true);
    pendiente.resolver(fase === "lectura" ? bandeja() : { recibo_ref: "recibo:sintetico", instante_utc: "2026-09-25T08:00:00Z", replay: false });
    await esperar();
    assert.equal(nodo.innerHTML, html); assert.equal(consultas, 1); assert.deepEqual(anuncios, []);
  }
});

test("recuperación y recibo usan los catálogos reales en ambos idiomas y escapan sus datos", async () => {
  for (const idioma of ["es", "en"]) {
    const comunes = await cargarTextos("cronos", { idioma, porDefecto: "es" });
    const propios = await cargarTextos("cronos-bandeja-notificaciones", { idioma, porDefecto: "es" });
    assert.deepEqual(propios.faltantes, []);
    const mensajes = { ...comunes.seccion("solicitudes"), ...comunes.seccion("notificaciones"), ...propios.seccion("bandeja") };
    const html = renderizarBandejaNotificacionesCronos({ estado: "error", mensajes, recibo: { recibo_ref: '<script>alert("x")</script>', instante_utc: "2026-09-25T08:00:00Z" } });
    assert.match(html, /data-cronos-reintentar-bandeja/); assert.match(html, /&lt;script&gt;/);
    assert.ok(html.includes(propios.traducir("bandeja.error_actualizar_bandeja")));
    assert.ok(html.includes(comunes.traducir("notificaciones.notificaciones_reintentar_consulta")));
    assert.ok(propios.traducir("bandeja.error_actualizar_bandeja").includes(comunes.traducir("notificaciones.notificaciones_reintentar_consulta")), "el aviso nombra el botón que permite recuperar");
    assert.doesNotMatch(html, /<script>/);
  }
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
