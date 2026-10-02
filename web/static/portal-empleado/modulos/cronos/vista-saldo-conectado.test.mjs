import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { MENSAJES_CRONOS } from "./i18n.js";
import { ErrorClienteSaldoCronos } from "./cliente-saldo-http.js";
import { montarVistaSaldoCronos, renderizarVistaSaldoCronos } from "./vista-saldo-conectado.js";

function datos(tipo = "hoy", trabajados = 65) {
  return {
    periodo: { tipo, desde: "2026-09-20", hasta: "2026-09-20" },
    resumen: { previstos_minutos: null, trabajados_minutos: trabajados, saldo_minutos: null, estado: "no_disponible" },
    detalle: [{ fecha: "2026-09-20", previstos_minutos: null, trabajados_minutos: trabajados, pausas_minutos: 5, saldo_minutos: null, estado: "no_disponible", marcajes: [
      { instante_utc: "2026-09-20T09:10:00Z", movimiento: "entrada", origen: null },
    ] }],
  };
}
function diferido() { let resolver; let rechazar; const promesa = new Promise((si, no) => { resolver = si; rechazar = no; }); return { promesa, resolver, rechazar }; }

test("detalle del cálculo distingue cero, jornada ausente e incompleto sin inferir salidas", () => {
  const resultado = datos();
  Object.assign(resultado.detalle[0], { previstos_minutos: 0, trabajados_minutos: 0, pausas_minutos: 0, saldo_minutos: 123, estado: "disponible" });
  const render = () => renderizarVistaSaldoCronos({ estado: "listo", datos: resultado });
  let html = render();
  assert.match(html, /<details><summary>Detalle del cálculo<\/summary><dl>\s*<dt>Tiempo previsto<\/dt><dd>00:00<\/dd>\s*<dt>Tiempo computado<\/dt><dd>00:00<\/dd>\s*<dt>Pausas<\/dt><dd>00:00<\/dd>/u);
  assert.match(html, /02:03/u, "el saldo de la consulta no se recalcula con los valores del detalle");
  assert.doesNotMatch(html, /No consta jornada prevista|Cálculo incompleto/u);
  resultado.detalle[0].previstos_minutos = null;
  html = render();
  assert.match(html, /No consta jornada prevista para este día\./u);
  assert.doesNotMatch(html, /Cálculo incompleto/u);
  resultado.detalle[0].estado = "incompleto";
  html = render();
  assert.match(html, /Cálculo incompleto\./u);
  assert.match(html, /No consta jornada prevista/u);
  assert.doesNotMatch(html, /falta.*salida|salida.*pendiente/iu);
  resultado.detalle[0].previstos_minutos = 420;
  assert.doesNotMatch(render(), /No consta jornada prevista/u);
});

test("referencias opcionales sólo en detalle técnico plegado, escapadas y sin enlaces", () => {
  const resultado = datos();
  const render = () => renderizarVistaSaldoCronos({ estado: "listo", datos: resultado });
  assert.doesNotMatch(render(), /Detalle técnico|Referencia del turno|Versión de la política/u);
  resultado.detalle[0].turno_ref = '<img src=x onerror="x()">&\'';
  resultado.detalle[0].politica_version_ref = 'https://example.invalid/<script>"';
  let html = render();
  assert.match(html, /<details><summary>Detalle técnico<\/summary><dl><dt>Referencia del turno<\/dt><dd>&lt;img src=x onerror=&quot;x\(\)&quot;&gt;&amp;&#39;<\/dd>/u);
  assert.match(html, /https:\/\/example.invalid\/&lt;script&gt;&quot;/u);
  assert.doesNotMatch(html, /<img|<script|<a\b|<details[^>]*\bopen\b/u);
  delete resultado.detalle[0].turno_ref;
  html = render();
  assert.doesNotMatch(html, /Referencia del turno/u);
  assert.match(html, /Versión de la política de cálculo/u);
  resultado.detalle[0].politica_version_ref = "";
  assert.doesNotMatch(render(), /Detalle técnico/u);
});

test("detalle del cálculo usa el catálogo inglés real y escapa las nuevas etiquetas", async () => {
  const mensajes = JSON.parse(await readFile(new URL("../../../textos/en/cronos.json", import.meta.url), "utf8")).general;
  const resultado = datos(); resultado.detalle[0].estado = "incompleto";
  const html = renderizarVistaSaldoCronos({ estado: "listo", datos: resultado, mensajes, locale: "en-GB" });
  assert.match(html, /Calculation details/u);
  assert.match(html, /Incomplete calculation\./u);
  assert.match(html, /No scheduled working hours are recorded for this day\./u);
  const escapado = renderizarVistaSaldoCronos({ estado: "listo", datos: resultado,
    mensajes: { ...MENSAJES_CRONOS, saldo_calculo_detalle: "<script>" } });
  assert.match(escapado, /<summary>&lt;script&gt;<\/summary>/u);
  assert.doesNotMatch(escapado, /<script>/u);
});

test("saldo conectado muestra minutos reales, nulidad y detalle sin revelar códigos privados", () => {
  const html = renderizarVistaSaldoCronos({ estado: "listo", consulta: { periodo: "hoy" }, datos: datos() });
  assert.match(html, /01:05/);
  assert.match(html, /00:05/);
  assert.match(html, /No disponible/);
  assert.match(html, /Origen pendiente de verificación/);
  assert.match(html, /Detalle por día/);
  assert.equal((html.match(/class="tarjeta-kpi"/g) || []).length, 3);
  assert.equal((html.match(/class="icono-kpi" aria-hidden="true"/g) || []).length, 3);
  assert.doesNotMatch(html, /Exceso semanal/);
  assert.match(html, /data-accion="ayuda"/);
  assert.doesNotMatch(html, /origen_ref|canal/);
  assert.doesNotMatch(html, /datos sintéticos|DEMO/i);
  const remoto = datos(); remoto.detalle[0].marcajes[0].origen = "remoto";
  assert.match(renderizarVistaSaldoCronos({ estado: "listo", datos: remoto }), /Remoto/);
  const malicioso = renderizarVistaSaldoCronos({ estado: "cargando", mensajes: { ...MENSAJES_CRONOS, saldo_titulo: '<img src=x onerror="x()">' } });
  assert.match(malicioso, /&lt;img/);
  assert.doesNotMatch(malicioso, /<img/);
});

test("carga, denegación y vacío no presentan números de una consulta anterior", () => {
  for (const estado of ["cargando", "denegado", "error"]) {
    const html = renderizarVistaSaldoCronos({ estado, datos: datos() });
    assert.doesNotMatch(html, /01:05/);
    assert.match(html, new RegExp(`data-cronos-saldo-estado="${estado}"`));
  }
  const vacio = renderizarVistaSaldoCronos({ estado: "listo", datos: { ...datos(), detalle: [] } });
  assert.match(vacio, /No hay datos de saldo para este periodo/);
  const rango = renderizarVistaSaldoCronos({ estado: "seleccion", consulta: { periodo: "rango", desde: "", hasta: "" } });
  assert.match(rango, /name="desde" value=""/);
  assert.match(rango, /Seleccione las dos fechas/);
});

test("cambiar de periodo cancela y descarta una respuesta tardía; desmontar cancela", async () => {
  const primero = diferido(); const segundo = diferido(); const señales = [];
  const nodo = { dataset: {}, innerHTML: "", eventos: {}, addEventListener(tipo, fn) { this.eventos[tipo] = fn; }, removeEventListener(tipo) { delete this.eventos[tipo]; }, remove() { this.eliminado = true; } };
  const raiz = { ownerDocument: { createElement: () => nodo }, append() {} };
  const vista = montarVistaSaldoCronos({ raiz, cliente: { consultar(_consulta, { signal }) { señales.push(signal); return señales.length === 1 ? primero.promesa : segundo.promesa; } } });
  assert.match(nodo.innerHTML, /Consultando saldo/);
  const consultaSegunda = vista.consultar({ periodo: "semana" });
  assert.equal(señales[0].aborted, true);
  primero.resolver(datos("hoy", 999));
  await Promise.resolve(); await Promise.resolve();
  assert.doesNotMatch(nodo.innerHTML, /16:39/);
  segundo.resolver(datos("semana", 65)); await consultaSegunda;
  assert.match(nodo.innerHTML, /01:05/);
  vista.desmontar();
  assert.equal(señales[1].aborted, true);
  assert.equal(nodo.eliminado, true);
});

test("una denegación queda distinguida de una caída de servicio", async () => {
  const nodo = { dataset: {}, innerHTML: "", addEventListener() {}, removeEventListener() {}, remove() {} };
  const raiz = { ownerDocument: { createElement: () => nodo }, append() {} };
  const vista = montarVistaSaldoCronos({ raiz, cliente: { consultar: async () => { throw new ErrorClienteSaldoCronos("acceso_denegado", 403); } } });
  await Promise.resolve(); await Promise.resolve();
  assert.match(nodo.innerHTML, /No tiene permiso/);
  assert.doesNotMatch(nodo.innerHTML, /No se pudo consultar/);
  vista.desmontar();
});

test("404 de la API: «no disponible» neutro, sin alerta; incrustada sin sobrelínea", async () => {
  const nodo = { dataset: {}, innerHTML: "", addEventListener() {}, removeEventListener() {}, remove() {} };
  const raiz = { ownerDocument: { createElement: () => nodo }, append() {} };
  const vista = montarVistaSaldoCronos({ raiz, incrustada: true, cliente: { consultar: async () => { throw new ErrorClienteSaldoCronos("servicio_no_disponible", 404); } } });
  await Promise.resolve(); await Promise.resolve();
  assert.match(nodo.innerHTML, /<p class="cronos-vacio"[^>]*role="status">El servicio de saldo no está disponible/u);
  assert.doesNotMatch(nodo.innerHTML, /No se pudo consultar/u);
  assert.doesNotMatch(nodo.innerHTML, /sobrelinea|<h2/u);
  assert.match(nodo.innerHTML, /<h3 id="cronos-saldo-titulo">/u);
  assert.match(renderizarVistaSaldoCronos({ estado: "cargando" }), /<p class="sobrelinea">[^<]+<\/p><h2 id="cronos-saldo-titulo">/u, "suelta conserva su encabezado de página");
  vista.desmontar();
});

function datosLargos(tipo = "anio", cantidad = 65) {
  const resultado = datos(tipo, 600);
  resultado.periodo = { tipo, desde: "2026-01-01", hasta: "2026-12-31" };
  resultado.detalle = Array.from({ length: cantidad }, (_, i) => ({ ...datos().detalle[0],
    fecha: new Date(Date.UTC(2026, 0, i + 1)).toISOString().slice(0, 10), marcajes: [] }));
  return resultado;
}
function montajePrueba(cliente, opciones = {}) {
  const documento = { activeElement: null }; const nodos = new Map();
  const nodo = { dataset: {}, innerHTML: "", eventos: {}, addEventListener(tipo, fn) { this.eventos[tipo] = fn; },
    removeEventListener(tipo) { delete this.eventos[tipo]; }, remove() {}, contains: (elemento) => [...nodos.values()].includes(elemento),
    querySelector(selector) {
      if (!nodos.has(selector)) nodos.set(selector, { disabled: false,
        getAttribute: (atributo) => selector.startsWith(`[${atributo}=`) ? selector.split('"')[1] : null,
        focus() { documento.activeElement = this; } });
      return nodos.get(selector);
    } };
  const raiz = { ownerDocument: Object.assign(documento, { createElement: () => nodo }), append() {} };
  const vista = montarVistaSaldoCronos({ raiz, cliente, ...opciones });
  const pulsar = (selector, dataset) => nodo.eventos.click({ target: { closest: (buscado) => buscado === selector ? { dataset } : null } });
  return { vista, nodo, documento, pulsar };
}
async function asentarse() { await Promise.resolve(); await Promise.resolve(); }

test("pagina el detalle completo sin alterar el resumen ni hacer peticiones por página", async () => {
  const recibidas = [];
  const m = montajePrueba({ consultar: async (consulta) => { recibidas.push(consulta); return datosLargos(consulta.periodo); } });
  await asentarse();
  assert.equal((m.nodo.innerHTML.match(/<th scope="row">/gu) || []).length, 31);
  assert.match(m.nodo.innerHTML, /Días 1–31 de 65. Página 1 de 3/);
  assert.match(m.nodo.innerHTML, /10:00/);
  m.pulsar("[data-cronos-saldo-pagina]", { cronosSaldoPagina: "siguiente" });
  assert.match(m.nodo.innerHTML, /Días 32–62 de 65. Página 2 de 3/);
  assert.equal((m.nodo.innerHTML.match(/<summary>Detalle del cálculo<\/summary>/gu) || []).length, 31);
  assert.match(m.nodo.innerHTML, /No consta jornada prevista para este día/u);
  assert.match(m.nodo.innerHTML, /datetime="2026-02-01"/);
  assert.match(m.nodo.innerHTML, /10:00/);
  assert.equal(recibidas.length, 1);
  m.pulsar("[data-cronos-saldo-pagina]", { cronosSaldoPagina: "siguiente" });
  assert.equal((m.nodo.innerHTML.match(/<th scope="row">/gu) || []).length, 3);
  assert.match(m.nodo.innerHTML, /Días 63–65 de 65/);
  m.vista.desmontar();
});

test("actualizar y reintentar conservan rango, página y foco sin dejar totales anteriores", async () => {
  const consultas = []; let fallar = false; let cantidad = 65; const anuncios = [];
  const consulta = { periodo: "rango", desde: "2026-01-01", hasta: "2026-12-31" };
  const m = montajePrueba({ consultar: async (c) => {
    consultas.push(c);
    if (fallar) throw new ErrorClienteSaldoCronos("red_no_disponible");
    return datosLargos(c.periodo, cantidad);
  } }, { anunciar: (texto) => anuncios.push(texto) });
  await asentarse(); await m.vista.consultar(consulta);
  m.pulsar("[data-cronos-saldo-pagina]", { cronosSaldoPagina: "siguiente" });
  const control = m.nodo.querySelector('[data-cronos-saldo-actualizar=""]'); control.focus();
  fallar = true; await m.vista.actualizar();
  assert.match(m.nodo.innerHTML, /Reintentar/);
  assert.doesNotMatch(m.nodo.innerHTML, /10:00/);
  assert.match(m.nodo.innerHTML, /name="desde" value="2026-01-01"/);
  assert.equal(m.documento.activeElement, control);
  fallar = false; await m.vista.actualizar();
  assert.match(m.nodo.innerHTML, /Página 2 de 3/);
  assert.equal(m.documento.activeElement, control);
  assert.deepEqual(consultas.slice(1), [consulta, consulta, consulta]);
  cantidad = 2; await m.vista.actualizar();
  assert.match(m.nodo.innerHTML, /Página 1 de 1/);
  assert.ok(anuncios.includes("Saldo actualizado."));
  assert.ok(!anuncios.includes("error"));
  m.vista.desmontar();
});

test("no ofrece recuperación para 403 o 404 y valida respuestas de clientes inyectados", async () => {
  for (const error of [new ErrorClienteSaldoCronos("acceso_denegado", 403), new ErrorClienteSaldoCronos("servicio_no_disponible", 404)]) {
    let llamadas = 0;
    const m = montajePrueba({ consultar: async () => { llamadas++; throw error; } });
    await asentarse(); await m.vista.actualizar();
    assert.equal(llamadas, 1);
    assert.doesNotMatch(m.nodo.innerHTML, /data-cronos-saldo-actualizar/);
    m.vista.desmontar();
  }
  const m = montajePrueba({ consultar: async () => ({ ...datos(), extra: "no admitido" }) });
  await asentarse();
  assert.match(m.nodo.innerHTML, /data-cronos-saldo-estado="error"/);
  assert.doesNotMatch(m.nodo.innerHTML, /01:05/);
  assert.match(m.nodo.innerHTML, /Reintentar/);
  m.vista.desmontar();
});

test("cambiar periodo reinicia página y finalizar una actualización no roba foco externo", async () => {
  const pendiente = diferido(); let esperar = false;
  const m = montajePrueba({ consultar: async (c) => esperar ? pendiente.promesa : datosLargos(c.periodo) });
  await asentarse();
  m.pulsar("[data-cronos-saldo-pagina]", { cronosSaldoPagina: "siguiente" });
  await m.vista.consultar({ periodo: "mes" });
  assert.match(m.nodo.innerHTML, /Página 1 de 3/);
  m.nodo.querySelector('[data-cronos-saldo-actualizar=""]').focus();
  esperar = true; const actualizar = m.vista.actualizar();
  const externo = {}; m.documento.activeElement = externo;
  pendiente.resolver(datosLargos("mes")); await actualizar;
  assert.equal(m.documento.activeElement, externo);
  m.vista.desmontar();
  await m.vista.actualizar();
});

test("tamaño de presentación configurable, acotado y datos vacíos sin paginación ficticia", () => {
  const html = renderizarVistaSaldoCronos({ estado: "listo", datos: datosLargos("hoy", 5), tamanoPagina: 2 });
  assert.equal((html.match(/<th scope="row">/gu) || []).length, 2);
  assert.match(html, /Página 1 de 3/);
  const vacio = renderizarVistaSaldoCronos({ estado: "listo", datos: { ...datos(), detalle: [] } });
  assert.doesNotMatch(vacio, /data-cronos-saldo-pagina/);
  for (const tamanoPagina of [0, -1, 1.5, 368, Infinity]) {
    assert.throws(() => renderizarVistaSaldoCronos({ tamanoPagina }), RangeError);
  }
});

test("un fichaje confirmado sustituye una lectura anterior pendiente del mismo periodo", async () => {
  const anterior = diferido();
  const posterior = diferido();
  const llamadas = [];
  const { nodo, vista } = montajePrueba({ consultar(consulta, { signal }) {
    llamadas.push({ consulta, signal });
    return llamadas.length === 1 ? anterior.promesa : posterior.promesa;
  } });
  nodo.eventos.click({ target: { closest(selector) {
    return selector === "[data-cronos-saldo-actualizar]" ? {} : null;
  } } });
  assert.equal(llamadas.length, 1, "el control de actualización no actúa mientras carga");
  const refresco = vista.actualizar();
  assert.equal(llamadas.length, 2);
  assert.equal(llamadas[0].signal.aborted, true);
  assert.deepEqual(llamadas[1].consulta, llamadas[0].consulta);
  posterior.resolver(datos("hoy", 65));
  await refresco;
  anterior.resolver(datos("hoy", 999));
  await Promise.resolve(); await Promise.resolve();
  assert.match(nodo.innerHTML, /01:05/);
  assert.doesNotMatch(nodo.innerHTML, /16:39/);
  vista.desmontar();
});
