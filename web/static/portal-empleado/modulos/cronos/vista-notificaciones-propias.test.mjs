import "./test-preparar-textos.mjs";
import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { crearClienteNotificacionesCronosHTTP, ErrorClienteNotificacionesCronos } from "./cliente-notificaciones-http.js";
import { hoyCivilCronos, montarNotificacionesPropiasCronos, renderizarNotificacionesPropiasCronos, renderizarHistorialNotificacionesCronos } from "./vista-notificaciones-propias.js";

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
const formulario = (valores) => ({ matches: (selector) => selector === "[data-cronos-notificacion-formulario]", elements: { namedItem: (n) => (n in valores ? { value: valores[n] } : null) } });

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

test("el documento se lee una sola vez: «input» no lo vuelve a leer, «change» sí", async () => {
  const { nodo, nodos, raiz } = raizFalsa(); let lecturas = 0;
  montarNotificacionesPropiasCronos({ raiz, cliente: { consultarPropias: async () => datos(), enviar: async () => ({}) },
    cripto: { subtle: { digest: async () => new Uint8Array(32).fill(0xab).buffer } } });
  await esperar();
  nodos["[data-cronos-documento-estado]"] = { textContent: "", dataset: {} };
  const control = { name: "documento", files: [{ size: 3, arrayBuffer: async () => { lecturas++; return new Uint8Array([1, 2, 3]).buffer; } }] };
  await nodo.eventos.input({ type: "input", target: control });
  assert.equal(lecturas, 0, "input no lee el fichero");
  await nodo.eventos.change({ type: "change", target: control });
  assert.equal(lecturas, 1);
  assert.equal(nodos["[data-cronos-documento-estado]"].textContent, "Huella del documento calculada.");
});

test("un error de envío no repinta el formulario: el fichero sigue con su huella y el foco vuelve al campo o al botón", async () => {
  const { nodo, nodos, raiz } = raizFalsa(); const envios = [];
  const cliente = { consultarPropias: async () => datos(), enviar: async (e) => { envios.push(e); throw new ErrorClienteNotificacionesCronos("servicio_no_disponible", 503); } };
  montarNotificacionesPropiasCronos({ raiz, cliente, cripto: { subtle: { digest: async () => new Uint8Array(32).fill(0xab).buffer } } });
  await esperar();
  const foco = [];
  const control = (name) => ({ name, disabled: false, focus() { foco.push(name); } });
  const campos = ["tipo", "fecha", "texto", "referencia", "documento"].map(control);
  const boton = { ...control("enviar"), textContent: "Enviar a RRHH" };
  const aviso = { innerHTML: "" };
  const elementos = Object.assign([...campos, boton], { namedItem: () => null });
  const form = { elements: elementos, querySelector: (sel) => ({ "[data-cronos-envio-aviso]": aviso, "[data-cronos-notificacion-enviar]": boton })[sel] ?? null };
  nodos["[data-cronos-notificacion-formulario]"] = form;
  nodos["[data-cronos-notificacion-enviar]"] = boton;
  for (const c of campos) nodos[`[data-cronos-notificacion-formulario] [name="${c.name}"]`] = c;
  nodos["[data-cronos-documento-estado]"] = { textContent: "", dataset: {} };
  await nodo.eventos.change({ type: "change", target: { name: "documento", files: [{ size: 3, arrayBuffer: async () => new Uint8Array([1, 2, 3]).buffer }] } });
  const pintado = nodo.innerHTML;
  await nodo.eventos.submit({ target: formulario({ tipo: TIPO, fecha: "2026-09-24", texto: "   ", referencia: "registro:sintetico:0001" }), preventDefault() {} });
  assert.equal(nodo.innerHTML, pintado, "un error local no repinta");
  assert.match(aviso.innerHTML, /Revise el tipo, la fecha y el mensaje/);
  assert.deepEqual(foco, ["texto"], "el foco va al campo en error");
  await nodo.eventos.submit({ target: formulario({ tipo: TIPO, fecha: "2026-09-24", texto: "Olvidé fichar", referencia: "registro:sintetico:0001" }), preventDefault() {} });
  assert.equal(nodo.innerHTML, pintado, "el envío fallido no recrea el campo del fichero");
  assert.equal(envios[0].adjunto_sha256, "ab".repeat(32), "se envía la huella del fichero que sigue elegido");
  assert.match(aviso.innerHTML, /No se pudo enviar la notificación/);
  assert.equal(boton.textContent, "Enviar a RRHH");
  assert.ok([...campos, boton].every((c) => c.disabled === false), "los campos vuelven a estar activos");
  assert.deepEqual(foco, ["texto", "enviar"], "tras un error del servidor el foco vuelve al botón");
});

test("la huella del documento se ofrece en un detalle desplegable, no sólo en un title", () => {
  const html = renderizarNotificacionesPropiasCronos({ estado: "listo", datos: datos(), hoy: "2026-09-25" });
  assert.match(html, /<details class="cronos-huella"><summary>Huella<\/summary><span class="cronos-huella-valor">Huella SHA-256: a{64}<\/span><\/details>/u);
  assert.doesNotMatch(html, /title="Huella/u);
});

function diferida() {
  let resolver; let rechazar;
  const promesa = new Promise((resolve, reject) => { resolver = resolve; rechazar = reject; });
  return { promesa, resolver, rechazar };
}
const clicConsultar = () => ({ target: { closest: (selector) => selector === "[data-cronos-notificacion-reintentar]" ? {} : null } });

for (const [status, replay, confirmacion] of [[201, false, /Notificación enviada a RRHH/], [200, true, /Esta notificación ya estaba registrada/]]) {
  test(`registro ${status}, consulta 503 y recuperación GET mantienen la confirmación sin otro POST`, async () => {
    const { nodo, nodos, raiz } = raizFalsa(); const llamadas = []; const pendiente = diferida(); const foco = [];
    let consultas = 0;
    const cliente = crearClienteNotificacionesCronosHTTP({ fetchImpl: async (_url, opciones) => {
      llamadas.push(opciones.method);
      if (opciones.method === "POST") return new Response(JSON.stringify({ recibo: {
        notificacion_ref: "notificacion:cronos:0f0e0d0c-0b0a-4000-8000-000000000003",
        recibo_ref: "recibo:cronos:0b9f3c2e-1d4a-4c6b-9e8f-0a1b2c3d4e5f", instante_utc: "2026-09-25T08:00:00Z", replay,
      } }), { status, headers: { "content-type": "application/json" } });
      consultas++;
      if (consultas === 2) return pendiente.promesa;
      return new Response(JSON.stringify(datos()), { status: 200, headers: { "content-type": "application/json" } });
    } });
    nodos["[data-cronos-notificacion-reintentar]"] = { focus: () => foco.push("consulta") };
    nodos["[data-cronos-notificacion-mensaje]"] = { focus: () => foco.push("confirmacion") };
    const vista = montarNotificacionesPropiasCronos({ raiz, cliente });
    await esperar(); await esperar();
    const registro = nodo.eventos.submit({ target: formulario({ tipo: TIPO, fecha: "2026-09-24", texto: "Texto registrado", referencia: "" }), preventDefault() {} });
    await esperar(); await esperar();
    assert.match(nodo.innerHTML, /data-estado="cargando"/);
    assert.match(nodo.innerHTML, confirmacion);
    assert.doesNotMatch(nodo.innerHTML, /<table|data-cronos-notificacion-formulario/);
    pendiente.resolver(new Response(JSON.stringify({ error: "no_disponible" }), { status: 503, headers: { "content-type": "application/json" } }));
    await registro;
    assert.match(nodo.innerHTML, /data-estado="error"/);
    assert.match(nodo.innerHTML, confirmacion);
    assert.match(nodo.innerHTML, /data-cronos-notificacion-reintentar/);
    assert.doesNotMatch(nodo.innerHTML, /<table|data-cronos-notificacion-formulario|No pude fichar/);
    assert.deepEqual(foco, ["consulta"]);
    await nodo.eventos.click(clicConsultar());
    assert.match(nodo.innerHTML, /data-estado="listo"/);
    assert.match(nodo.innerHTML, confirmacion);
    assert.match(nodo.innerHTML, /<table/);
    assert.match(nodo.innerHTML, /<textarea[^>]*><\/textarea>/);
    assert.deepEqual(llamadas, ["GET", "POST", "GET", "GET"]);
    assert.deepEqual(foco, ["consulta", "confirmacion"]);
    vista.desmontar();
  });
}

test("consulta antigua resuelta o rechazada no sustituye la recuperación vigente ni mueve el foco", async () => {
  for (const rechazada of [false, true]) {
    const { nodo, nodos, raiz } = raizFalsa(); const antigua = diferida(); const vigente = diferida(); const signals = []; const anuncios = []; let consulta = 0;
    const cliente = { consultarPropias: ({ signal }) => {
      signals.push(signal); consulta++;
      return consulta === 1 ? Promise.reject(new ErrorClienteNotificacionesCronos("servicio_no_disponible", 503))
        : consulta === 2 ? antigua.promesa : vigente.promesa;
    }, enviar: async () => { assert.fail("consultar no debe registrar"); } };
    let focos = 0;
    nodos["[data-cronos-notificacion-reintentar]"] = { focus: () => { focos++; } };
    const vista = montarNotificacionesPropiasCronos({ raiz, cliente, anunciar: (texto) => anuncios.push(texto) });
    await esperar();
    const reintento = nodo.eventos.click(clicConsultar());
    const actual = vista.recargar();
    assert.equal(signals[1].aborted, true);
    if (rechazada) antigua.rechazar(new ErrorClienteNotificacionesCronos("servicio_no_disponible", 503)); else antigua.resolver(datos());
    await reintento;
    assert.match(nodo.innerHTML, /data-estado="cargando"/);
    assert.equal(focos, 0); assert.equal(anuncios.length, 1);
    vigente.resolver({ tipos: datos().tipos, notificaciones: [] }); await actual;
    assert.match(nodo.innerHTML, /data-estado="listo"/);
    assert.doesNotMatch(nodo.innerHTML, /No pude fichar/);
    vista.desmontar();
  }
});

test("desmontar durante recuperación cancela el GET y descarta su respuesta y su foco", async () => {
  for (const rechazada of [false, true]) {
    const { nodo, nodos, raiz } = raizFalsa(); const pendiente = diferida(); let consulta = 0; let signal; let focos = 0;
    const cliente = { consultarPropias: (opciones) => {
      consulta++; signal = opciones.signal;
      return consulta === 1 ? Promise.reject(new ErrorClienteNotificacionesCronos("servicio_no_disponible", 503)) : pendiente.promesa;
    }, enviar: async () => { assert.fail("consultar no debe registrar"); } };
    nodos["[data-cronos-notificacion-reintentar]"] = { focus: () => { focos++; } };
    const vista = montarNotificacionesPropiasCronos({ raiz, cliente }); await esperar();
    const alConsultar = nodo.eventos.click;
    const reintento = alConsultar(clicConsultar()); const html = nodo.innerHTML;
    vista.desmontar(); assert.equal(signal.aborted, true); assert.equal(nodo.eventos.click, undefined);
    if (rechazada) pendiente.rechazar(new ErrorClienteNotificacionesCronos("servicio_no_disponible", 503)); else pendiente.resolver(datos());
    await reintento; await vista.recargar(); await alConsultar(clicConsultar());
    assert.equal(nodo.innerHTML, html); assert.equal(focos, 0); assert.equal(consulta, 2);
  }
});

test("el recibo tardío tras desmontar no muestra confirmación ni inicia otra consulta", async () => {
  const { nodo, raiz } = raizFalsa(); const pendiente = diferida(); let consultas = 0; let signal; const anuncios = [];
  const vista = montarNotificacionesPropiasCronos({ raiz, anunciar: (texto) => anuncios.push(texto), cliente: {
    consultarPropias: async () => { consultas++; return datos(); },
    enviar: (_entrada, opciones) => { signal = opciones.signal; return pendiente.promesa; },
  } });
  await esperar();
  const registro = nodo.eventos.submit({ target: formulario({ tipo: TIPO, fecha: "2026-09-24", texto: "Texto", referencia: "" }), preventDefault() {} });
  const html = nodo.innerHTML; vista.desmontar(); assert.equal(signal.aborted, true);
  pendiente.resolver({ replay: false }); await registro;
  assert.equal(nodo.innerHTML, html); assert.equal(consultas, 1); assert.equal(anuncios.length, 0);
});

test("tipo retirado 409 y consulta 503 conservan el aviso sin afirmar una actualización ni repetir POST", async () => {
  const { nodo, raiz } = raizFalsa(); const llamadas = []; let consultas = 0;
  const cliente = crearClienteNotificacionesCronosHTTP({ fetchImpl: async (_url, opciones) => {
    llamadas.push(opciones.method);
    if (opciones.method === "POST") return new Response(JSON.stringify({ error: "tipo_no_vigente" }), {
      status: 409, headers: { "content-type": "application/json" },
    });
    consultas++;
    return new Response(JSON.stringify(consultas === 2 ? { error: "no_disponible" } : datos()), {
      status: consultas === 2 ? 503 : 200, headers: { "content-type": "application/json" },
    });
  } });
  const vista = montarNotificacionesPropiasCronos({ raiz, cliente });
  await esperar(); await esperar();
  await nodo.eventos.submit({ target: formulario({ tipo: TIPO, fecha: "2026-09-24", texto: "Texto", referencia: "" }), preventDefault() {} });
  assert.match(nodo.innerHTML, /data-estado="error"/);
  assert.match(nodo.innerHTML, /Ese tipo ya no está disponible\. Elija otro tipo\./);
  assert.match(nodo.innerHTML, /data-cronos-notificacion-reintentar/);
  assert.doesNotMatch(nodo.innerHTML, /La lista se ha actualizado|<table|data-cronos-notificacion-formulario|data-tono="exito"/);
  await nodo.eventos.click(clicConsultar());
  assert.match(nodo.innerHTML, /data-estado="listo"/);
  assert.match(nodo.innerHTML, /Ese tipo ya no está disponible\. Elija otro tipo\./);
  assert.doesNotMatch(nodo.innerHTML, /<option value="notificacion:cronos:tipo:[^"]*" selected/);
  assert.deepEqual(llamadas, ["GET", "POST", "GET", "GET"]);
  vista.desmontar();
});

function datosHistorial(cantidad = 23) {
  const base = datos();
  return { ...base, notificaciones: Array.from({ length: cantidad }, (_, i) => ({ ...base.notificaciones[i % 2], texto: `Mensaje ${i + 1}` })) };
}
const filtroHistorial = (valor) => ({ target: {
  matches: (selector) => selector === "[data-cronos-historial-filtros]",
  elements: { namedItem: (nombre) => nombre === "estado_historial" ? { value: valor } : null },
}, preventDefault() {} });
const clicHistorial = (selector, dataset = {}) => ({ target: { closest: (s) => s === selector ? { dataset } : null } });

function historialFalso() {
  const base = raizFalsa(); const focos = [];
  const historial = { innerHTML: "" };
  base.nodos["[data-cronos-historial]"] = historial;
  for (const [selector, nombre] of [["[data-cronos-historial-resumen]", "resumen"], ["[data-cronos-historial-estado]", "estado"], ["[data-cronos-historial-actualizar]", "actualizar"]]) {
    base.nodos[selector] = { focus: () => focos.push(nombre) };
  }
  return { ...base, historial, focos };
}

test("el historial pagina las filas consultadas y cuenta sólo los resultados del filtro", () => {
  const base = datosHistorial();
  const primera = renderizarHistorialNotificacionesCronos({ datos: base });
  assert.match(primera, /Mostrando 1–10 de 23 notificaciones consultadas/);
  assert.match(primera, /Página 1 de 3/);
  assert.doesNotMatch(primera, /Mensaje 11</);
  const ultima = renderizarHistorialNotificacionesCronos({ datos: base, historial: { pagina: 99 } });
  assert.match(ultima, /Mostrando 21–23 de 23/);
  assert.match(ultima, /data-cronos-historial-pagina="4" disabled/);
  const atendidas = renderizarHistorialNotificacionesCronos({ datos: base, historial: { filtro: "atendida", pagina: 2 } });
  assert.match(atendidas, /Mostrando 11–12 de 12/);
  assert.doesNotMatch(atendidas, /data-estado="registrada"/);
  const registradas = renderizarHistorialNotificacionesCronos({ datos: base, historial: { filtro: "registrada" } });
  assert.match(registradas, /Mostrando 1–10 de 11/);
  assert.doesNotMatch(registradas, /data-estado="atendida"/);
  assert.match(registradas, /como máximo 500 notificaciones/);
  assert.match(renderizarHistorialNotificacionesCronos({ datos: datosHistorial(1), historial: { filtro: "registrada" } }), /No hay notificaciones consultadas con este estado/);
  assert.match(renderizarHistorialNotificacionesCronos({ datos: datosHistorial(0) }), /Mostrando 0–0 de 0/);
});

test("filtros, páginas y refresco conservan el formulario, el documento y la clave de un envío fallido", async () => {
  const { nodo, nodos, raiz, historial, focos } = historialFalso(); const envios = []; let consultas = 0;
  const vista = montarNotificacionesPropiasCronos({ raiz, cliente: {
    consultarPropias: async () => { consultas++; return datosHistorial(); },
    enviar: async (entrada) => { envios.push(entrada); throw new ErrorClienteNotificacionesCronos("servicio_no_disponible", 503); },
  }, cripto: { subtle: { digest: async () => new Uint8Array(32).fill(0xab).buffer } } });
  await esperar();
  const archivo = { size: 3, arrayBuffer: async () => new Uint8Array([1, 2, 3]).buffer };
  const controlDocumento = { name: "documento", files: [archivo] };
  const boton = { textContent: "", focus() {} }; const aviso = { innerHTML: "" };
  const form = { elements: [controlDocumento], querySelector: (s) => s === "[data-cronos-envio-aviso]" ? aviso : s === "[data-cronos-notificacion-enviar]" ? boton : null };
  nodos["[data-cronos-notificacion-formulario]"] = form;
  await nodo.eventos.change({ type: "change", target: controlDocumento });
  const borrador = { tipo: TIPO, fecha: "2026-09-24", texto: "Olvidé fichar", referencia: "registro:sintetico:0001" };
  await nodo.eventos.submit({ target: formulario(borrador), preventDefault() {} });
  const html = nodo.innerHTML;
  await nodo.eventos.submit(filtroHistorial("atendida"));
  await nodo.eventos.click(clicHistorial("[data-cronos-historial-pagina]", { cronosHistorialPagina: "2" }));
  assert.match(historial.innerHTML, /Mostrando 11–12 de 12/);
  assert.equal(consultas, 1, "filtrar y paginar no consultan ni registran");
  assert.equal(envios.length, 1);
  await nodo.eventos.click(clicHistorial("[data-cronos-historial-actualizar]"));
  assert.match(historial.innerHTML, /value="atendida" selected/);
  assert.match(historial.innerHTML, /Mostrando 11–12 de 12/);
  assert.equal(consultas, 2);
  assert.equal(nodo.innerHTML, html, "el contenedor del formulario no se repinta");
  assert.equal(form.elements[0].files[0], archivo);
  assert.deepEqual(focos, ["estado", "resumen", "resumen", "actualizar"]);
  await nodo.eventos.submit({ target: formulario(borrador), preventDefault() {} });
  assert.deepEqual(envios[1], envios[0], "el reintento conserva clave, referencia y huella");
  await nodo.eventos.click(clicHistorial("[data-cronos-historial-quitar]"));
  assert.match(historial.innerHTML, /Mostrando 1–10 de 23/);
  assert.match(historial.innerHTML, /value="" selected/);
  assert.equal(consultas, 2); assert.equal(envios.length, 2);
  vista.desmontar();
});

test("una nueva consulta ajusta la página al último resultado y conserva sólo el filtro confirmado", async () => {
  const { nodo, raiz, historial } = historialFalso(); let cantidad = 23;
  const vista = montarNotificacionesPropiasCronos({ raiz, cliente: { consultarPropias: async () => datosHistorial(cantidad), enviar: async () => assert.fail("no debe enviar") } });
  await esperar();
  await nodo.eventos.submit(filtroHistorial("atendida"));
  await nodo.eventos.click(clicHistorial("[data-cronos-historial-pagina]", { cronosHistorialPagina: "2" }));
  // Cambiar el selector sin confirmar no altera el filtro aplicado.
  await nodo.eventos.change({ target: { name: "estado_historial", value: "registrada" } });
  cantidad = 3; await vista.recargar();
  assert.match(historial.innerHTML, /value="atendida" selected/);
  assert.match(historial.innerHTML, /Página 1 de 1/);
  assert.match(historial.innerHTML, /Mostrando 1–2 de 2/);
  vista.desmontar();
});

test("503 de refresco conserva borrador y filtro; recuperación sólo GET y denegación oculta datos", async () => {
  const { nodo, raiz, historial } = historialFalso(); let fallo; let consultas = 0;
  const vista = montarNotificacionesPropiasCronos({ raiz, cliente: {
    consultarPropias: async () => { consultas++; if (fallo) throw fallo; return datosHistorial(); },
    enviar: async () => assert.fail("no debe enviar"),
  } });
  await esperar(); await nodo.eventos.submit(filtroHistorial("registrada"));
  const html = nodo.innerHTML;
  fallo = new ErrorClienteNotificacionesCronos("servicio_no_disponible", 503);
  await vista.recargar();
  assert.equal(nodo.innerHTML, html);
  assert.match(historial.innerHTML, /El borrador sigue en el formulario/);
  assert.doesNotMatch(historial.innerHTML, /<table/);
  fallo = null; await nodo.eventos.click(clicHistorial("[data-cronos-historial-actualizar]"));
  assert.match(historial.innerHTML, /value="registrada" selected/);
  assert.equal(consultas, 3);
  fallo = new ErrorClienteNotificacionesCronos("acceso_denegado", 403); await vista.recargar();
  assert.match(nodo.innerHTML, /data-estado="denegado"/);
  assert.doesNotMatch(nodo.innerHTML, /<table|data-cronos-notificacion-formulario/);
  vista.desmontar();
});

test("los catálogos completos funcionan con el traductor real y escapan sus variables", async () => {
  const { cargarTextos } = await import("../../../comun/textos.js");
  const { crearTraductorNotificacionesHistorialCronos } = await import("./i18n-notificaciones-historial.js?v=20261001-cronos-c9-historial-v2");
  for (const idioma of ["es", "en"]) {
    const textos = await cargarTextos("cronos-notificaciones-historial", { idioma });
    assert.deepEqual(textos.faltantes, []);
    const mensajes = textos.seccion("historial");
    const th = crearTraductorNotificacionesHistorialCronos(mensajes);
    assert.throws(() => th("desconocida"));
    assert.throws(() => crearTraductorNotificacionesHistorialCronos({}));
    const html = renderizarHistorialNotificacionesCronos({ datos: datosHistorial(0), mensajesHistorial: { ...mensajes, filtro_activo: "<script>{estado}</script>" } });
    assert.match(html, /&lt;script&gt;/);
    assert.doesNotMatch(html, /<script/);
  }
});

test("el límite recibido de 500 filas sólo produce páginas locales, sin un total supuesto", () => {
  const html = renderizarHistorialNotificacionesCronos({ datos: datosHistorial(500), historial: { pagina: 50 } });
  assert.match(html, /Mostrando 491–500 de 500 notificaciones consultadas/);
  assert.match(html, /Página 50 de 50/);
  assert.equal((html.match(/<tr>/gu) ?? []).length, 11, "diez filas y una cabecera");
  assert.match(html, /data-cronos-historial-pagina="51" disabled/);
});

test("una respuesta antigua de refresco no reemplaza el historial vigente tras recargar o desmontar", async () => {
  const { nodo, raiz, historial } = historialFalso(); const antigua = diferida(); let consulta = 0;
  const signals = [];
  const vista = montarNotificacionesPropiasCronos({ raiz, cliente: {
    consultarPropias: ({ signal }) => { signals.push(signal); consulta++; return consulta === 2 ? antigua.promesa : Promise.resolve(datosHistorial(consulta === 1 ? 23 : 3)); },
    enviar: async () => assert.fail("no debe enviar"),
  } });
  await esperar(); await nodo.eventos.submit(filtroHistorial("atendida"));
  const refresco = vista.recargar(); await vista.recargar();
  assert.equal(signals[1].aborted, true);
  const vigente = historial.innerHTML;
  antigua.resolver(datosHistorial(99)); await refresco;
  assert.equal(historial.innerHTML, vigente);
  assert.match(vigente, /Mostrando 1–2 de 2/);
  vista.desmontar(); await vista.recargar();
  assert.equal(consulta, 3);
});


test("tipos inicialmente vacíos: refrescar crea el formulario y permite registrar con el tipo recibido", async () => {
  const { nodo, nodos, raiz, historial } = historialFalso(); const envios = []; let consultas = 0;
  const vista = montarNotificacionesPropiasCronos({ raiz, cliente: {
    consultarPropias: async () => { consultas++; return consultas === 1 ? { ...datos(), tipos: [] } : datos(); },
    enviar: async (entrada) => { envios.push(entrada); return { replay: false }; },
  } });
  await esperar();
  assert.match(nodo.innerHTML, /data-cronos-notificacion-sin-tipos/);
  assert.doesNotMatch(nodo.innerHTML, /<form[^>]*data-cronos-notificacion-formulario/);
  const aviso = { outerHTML: "" };
  nodos["[data-cronos-notificacion-sin-tipos]"] = aviso;
  await nodo.eventos.click(clicHistorial("[data-cronos-historial-actualizar]"));
  assert.match(aviso.outerHTML, /<form[^>]*data-cronos-notificacion-formulario/);
  assert.match(aviso.outerHTML, /<option value="notificacion:cronos:tipo:incidencia-marcaje:sintetico-1">/);
  assert.doesNotMatch(aviso.outerHTML, /data-cronos-notificacion-sin-tipos|No hay tipos de notificación disponibles/);
  assert.match(historial.innerHTML, /Actualizar historial/);
  await nodo.eventos.submit({ target: formulario({ tipo: TIPO, fecha: "2026-09-24", texto: "Olvidé fichar al salir.", referencia: "" }), preventDefault() {} });
  assert.equal(envios.length, 1);
  assert.equal(envios[0].tipo_version_ref, TIPO);
  assert.equal(consultas, 3, "primera consulta, refresco y consulta tras el registro");
  assert.match(nodo.innerHTML, /Notificación enviada a RRHH/);
  vista.desmontar();
});
