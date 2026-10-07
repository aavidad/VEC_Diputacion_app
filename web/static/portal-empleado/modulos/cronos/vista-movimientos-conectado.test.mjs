import "./test-preparar-textos.mjs";
import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { ErrorClienteSaldoCronos } from "./cliente-saldo-http.js";
import { renderizarVistaMovimientosCronos, montarVistaMovimientosCronos } from "./vista-movimientos-conectado.js";

function respuesta(tipo = "hoy", origen = "remoto") {
  return {
    periodo: { tipo, desde: "2026-09-20", hasta: "2026-09-20" },
    resumen: { previstos_minutos: null, trabajados_minutos: 0, saldo_minutos: null, estado: "incompleto" },
    detalle: [{ fecha: "2026-09-20", previstos_minutos: null, trabajados_minutos: 0, pausas_minutos: 0,
      saldo_minutos: null, estado: "incompleto", marcajes: [
        { instante_utc: "2026-09-20T09:10:00Z", movimiento: "entrada", origen },
      ] }],
  };
}
function diferido() { let resolver; const promesa = new Promise((si) => { resolver = si; }); return { promesa, resolver }; }
function raizFalsa() {
  const nodo = { dataset: {}, innerHTML: "", eventos: {}, addEventListener(tipo, fn) { this.eventos[tipo] = fn; },
    removeEventListener(tipo) { delete this.eventos[tipo]; }, remove() { this.eliminado = true; },
    querySelector() { return null; } };
  return { nodo, raiz: { ownerDocument: { createElement: () => nodo }, append() {} } };
}

test("muestra movimiento remoto acreditado y estado incompleto sin inferir ausencia u olvido", () => {
  const html = renderizarVistaMovimientosCronos({ estado: "listo", datos: respuesta() });
  assert.match(html, /Entrada/);
  assert.match(html, /Remoto/);
  assert.match(html, /Incompleto/);
  assert.doesNotMatch(html, /solicitar-correccion|Solicitar corrección/u, "sin circuito de corrección no se ofrece");
  assert.match(renderizarVistaMovimientosCronos({ estado: "listo", datos: respuesta(), correccionDisponible: true }),
    /data-cronos-accion="solicitar-correccion">/u);
  assert.match(html, /data-accion="ayuda"/);
  assert.doesNotMatch(html, /absentismo|olvido|DEMO|datos sintéticos/i);
  const sinOrigen = respuesta(); sinOrigen.detalle[0].marcajes[0].origen = null;
  assert.match(renderizarVistaMovimientosCronos({ estado: "listo", datos: sinOrigen }), /Origen pendiente de verificación/);
});

test("sin movimientos presenta vacío y conserva estado diario acreditado", () => {
  const sinMovimientos = respuesta(); sinMovimientos.detalle[0].marcajes = [];
  const html = renderizarVistaMovimientosCronos({ estado: "listo", datos: sinMovimientos });
  assert.match(html, /No hay movimientos en este periodo/);
  assert.match(html, /Sin movimientos registrados/);
  assert.match(html, /Incompleto/);
  const rango = renderizarVistaMovimientosCronos({ estado: "seleccion", consulta: { periodo: "rango", desde: "", hasta: "" } });
  assert.match(rango, /name="desde" value=""/);
  assert.match(rango, /Seleccione las dos fechas/);
});

test("403 y 503 muestran mensajes distintos sin reutilizar hechos anteriores", async () => {
  for (const [codigo, estadoHTTP, texto] of [["acceso_denegado", 403, "No tiene permiso"], ["servicio_no_disponible", 503, "No se pudieron consultar"]]) {
    const { nodo, raiz } = raizFalsa();
    const vista = montarVistaMovimientosCronos({ raiz, cliente: { consultar: async () => { throw new ErrorClienteSaldoCronos(codigo, estadoHTTP); } } });
    await Promise.resolve(); await Promise.resolve();
    assert.match(nodo.innerHTML, new RegExp(texto));
    assert.doesNotMatch(nodo.innerHTML, /09:10|Entrada|Remoto/);
    vista.desmontar();
  }
});

test("cambio de periodo y desmontaje abortan; una respuesta antigua no sustituye la nueva", async () => {
  const primera = diferido(); const segunda = diferido(); const señales = [];
  const { nodo, raiz } = raizFalsa();
  const vista = montarVistaMovimientosCronos({ raiz, cliente: { consultar(_consulta, { signal }) {
    señales.push(signal); return señales.length === 1 ? primera.promesa : segunda.promesa;
  } } });
  assert.match(nodo.innerHTML, /Consultando movimientos/);
  const nueva = vista.consultar({ periodo: "semana" });
  assert.equal(señales[0].aborted, true);
  primera.resolver(respuesta("hoy", "terminal"));
  await Promise.resolve(); await Promise.resolve();
  assert.doesNotMatch(nodo.innerHTML, /Terminal/);
  segunda.resolver(respuesta("semana", "remoto")); await nueva;
  assert.match(nodo.innerHTML, /Remoto/);
  vista.desmontar();
  assert.equal(señales[1].aborted, true);
  assert.equal(nodo.eliminado, true);
});

test("elegir rango deja fechas vacías y consulta solo al enviar fechas válidas", async () => {
  const primera = diferido(); const llamadas = []; const { nodo, raiz } = raizFalsa();
  const vista = montarVistaMovimientosCronos({ raiz, cliente: { consultar(consulta, { signal }) {
    llamadas.push({ consulta, signal });
    return llamadas.length === 1 ? primera.promesa : Promise.resolve(respuesta("rango"));
  } } });
  const boton = { dataset: { cronosMovimientosPeriodo: "rango" } };
  nodo.eventos.click({ target: { closest: (selector) => selector === "[data-cronos-movimientos-periodo]" ? boton : null } });
  assert.equal(llamadas[0].signal.aborted, true);
  assert.equal(llamadas.length, 1);
  assert.match(nodo.innerHTML, /name="desde" value=""/);
  const formulario = { matches: () => true, elements: { namedItem: (nombre) => ({ value: nombre === "desde" ? "2026-09-20" : "2026-09-21" }) } };
  nodo.eventos.submit({ target: formulario, preventDefault() {} });
  await Promise.resolve(); await Promise.resolve();
  assert.deepEqual(llamadas[1].consulta, { periodo: "rango", desde: "2026-09-20", hasta: "2026-09-21" });
  vista.desmontar();
});

test("la vista no usa almacenamiento web ni crea otra fuente de movimientos", async () => {
  const fuente = await readFile(new URL("./vista-movimientos-conectado.js", import.meta.url), "utf8");
  assert.doesNotMatch(fuente, /localStorage|sessionStorage|indexedDB|document\.cookie|navigator\.geolocation|datos-sinteticos|Math\.random/u);
});

test("404 de la API: «no disponible» neutro, sin alerta; incrustada sin sobrelínea", async () => {
  const { nodo, raiz } = raizFalsa();
  const vista = montarVistaMovimientosCronos({ raiz, incrustada: true,
    cliente: { consultar: async () => { throw new ErrorClienteSaldoCronos("servicio_no_disponible", 404); } } });
  await Promise.resolve(); await Promise.resolve();
  assert.match(nodo.innerHTML, /data-cronos-movimientos-estado="no_disponible"/u);
  assert.match(nodo.innerHTML, /<p class="cronos-vacio"[^>]*role="status">La consulta de movimientos no está disponible\.<\/p>/u);
  assert.doesNotMatch(nodo.innerHTML, /role="alert"[^>]*>(?!<\/p>)|No se pudieron consultar/u);
  assert.doesNotMatch(nodo.innerHTML, /sobrelinea|<h2/u);
  assert.match(nodo.innerHTML, /<h3 id="cronos-movimientos-titulo">/u);
  vista.desmontar();
});

function pulsar(nodo, atributo, valor = "") {
  const boton = { dataset: { [atributo]: valor }, disabled: false };
  const selector = `[${atributo.replace(/[A-Z]/gu, (letra) => `-${letra.toLowerCase()}`).replace(/^cronos/u, "data-cronos")}]`;
  nodo.eventos.click({ target: { closest: (buscado) => buscado === selector ? boton : null } });
}
function anual(tipo = "anio") {
  const base = respuesta(tipo);
  base.periodo = { tipo, desde: "2026-01-01", hasta: "2026-12-31" };
  base.detalle = Array.from({ length: 365 }, (_, indice) => {
    const fecha = new Date(Date.UTC(2026, 0, indice + 1)).toISOString().slice(0, 10);
    return { ...base.detalle[0], fecha, marcajes: [
      { instante_utc: `${fecha}T07:00:00Z`, movimiento: "entrada", origen: "remoto" },
      { instante_utc: `${fecha}T14:00:00Z`, movimiento: "salida", origen: "terminal" },
    ] };
  });
  return base;
}
const completar = async () => { await Promise.resolve(); await Promise.resolve(); };

test("rango en borrador no rechaza el refresco conjunto, no consulta ni pierde formulario o foco", async () => {
  const llamadas = []; const { nodo, raiz } = raizFalsa();
  const vista = montarVistaMovimientosCronos({ raiz, cliente: { consultar: async (consulta) => { llamadas.push(consulta); return respuesta(); } } });
  await completar();
  pulsar(nodo, "cronosMovimientosPeriodo", "rango");
  const desde = { value: "2026-09-20" }; const hasta = { value: "" };
  nodo.querySelector = (selector) => selector === "[name=desde]" ? desde : selector === "[name=hasta]" ? hasta : null;
  // El formulario real mantiene las entradas en el DOM: actualizar no lo sustituye.
  const html = nodo.innerHTML; raiz.ownerDocument.activeElement = hasta;
  await assert.doesNotReject(Promise.all([vista.actualizar(), Promise.resolve("recibo confirmado")]));
  assert.equal(llamadas.length, 1); assert.equal(nodo.innerHTML, html);
  assert.equal(desde.value, "2026-09-20"); assert.equal(hasta.value, "");
  assert.equal(raiz.ownerDocument.activeElement, hasta);
  const formulario = { matches: () => true, elements: { namedItem: (nombre) => nombre === "desde" ? desde : hasta } };
  nodo.eventos.submit({ target: formulario, preventDefault() {} });
  await assert.doesNotReject(vista.actualizar()); assert.equal(llamadas.length, 1);
  vista.desmontar(); await assert.doesNotReject(vista.actualizar()); assert.equal(llamadas.length, 1);
});

test("año paginado conserva todos los días y sus marcajes, sin nuevas consultas ni totales fabricados", () => {
  const datos = anual(); const fechas = [];
  for (let pagina = 1; pagina <= 12; pagina++) {
    const html = renderizarVistaMovimientosCronos({ estado: "listo", consulta: { periodo: "anio" }, datos, pagina });
    const filas = [...html.matchAll(/<th scope="row"><time datetime="([^"]+)"/gu)].map((fila) => fila[1]);
    assert.equal(filas.length, pagina === 12 ? 48 : 62);
    for (let indice = 0; indice < filas.length; indice += 2) {
      assert.equal(filas[indice], filas[indice + 1]); fechas.push(filas[indice]);
    }
    assert.match(html, /de 365\. Página/u);
  }
  assert.deepEqual(fechas, datos.detalle.map((dia) => dia.fecha));
});

test("503 y reintento conservan consulta y página; cambiar periodo reinicia y 403/404 ocultan los datos", async () => {
  const { nodo, raiz } = raizFalsa(); const llamadas = []; let fallo = null;
  const vista = montarVistaMovimientosCronos({ raiz, cliente: { consultar: async (consulta) => {
    llamadas.push(consulta); if (fallo) throw fallo;
    return consulta.periodo === "hoy" ? respuesta() : anual(consulta.periodo);
  } } });
  await completar();
  await vista.consultar({ periodo: "rango", desde: "2026-01-01", hasta: "2026-12-31" });
  pulsar(nodo, "cronosMovimientosPagina", "siguiente"); assert.match(nodo.innerHTML, /Página 2 de 12/u);
  assert.equal(llamadas.length, 2);
  fallo = new ErrorClienteSaldoCronos("servicio_no_disponible", 503); await vista.actualizar();
  assert.match(nodo.innerHTML, /Reintentar/u); assert.doesNotMatch(nodo.innerHTML, /2026-02-01T07/u);
  fallo = null; pulsar(nodo, "cronosMovimientosActualizar"); await completar();
  assert.match(nodo.innerHTML, /Página 2 de 12/u);
  assert.deepEqual(llamadas.slice(-2), [{ periodo: "rango", desde: "2026-01-01", hasta: "2026-12-31" }, { periodo: "rango", desde: "2026-01-01", hasta: "2026-12-31" }]);
  await vista.consultar({ periodo: "anio" }); assert.match(nodo.innerHTML, /Página 1 de 12/u);
  for (const estado of [403, 404]) {
    fallo = new ErrorClienteSaldoCronos(estado === 403 ? "acceso_denegado" : "servicio_no_disponible", estado);
    await vista.actualizar(); assert.doesNotMatch(nodo.innerHTML, /2026-01-01T07|Reintentar/u);
    const numero = llamadas.length; await vista.actualizar(); assert.equal(llamadas.length, numero);
    fallo = null; await vista.consultar({ periodo: "anio" });
  }
  vista.desmontar();
});

test("refresco tras fichaje sustituye lectura en curso y descarta respuesta vieja", async () => {
  const { nodo, raiz } = raizFalsa(); const viejo = diferido(); const nuevo = diferido(); const llamadas = [];
  const vista = montarVistaMovimientosCronos({ raiz, cliente: { consultar(consulta, { signal }) {
    llamadas.push({ consulta, signal }); return llamadas.length === 1 ? viejo.promesa : nuevo.promesa;
  } } });
  const refresco = vista.actualizar(); assert.equal(llamadas[0].signal.aborted, true);
  nuevo.resolver(respuesta("hoy", "remoto")); await refresco;
  viejo.resolver(respuesta("hoy", "terminal")); await completar();
  assert.match(nodo.innerHTML, /Remoto/u); assert.doesNotMatch(nodo.innerHTML, /Terminal/u);
  assert.deepEqual(llamadas.map(({ consulta }) => consulta), [{ periodo: "hoy" }, { periodo: "hoy" }]);
  vista.desmontar();
});

test("texto y fechas usan los catálogos ingleses reales; el borrador escapa valores", async () => {
  const { cargarTextos } = await import("../../../comun/textos.js");
  const mensajes = (await cargarTextos("cronos", { idioma: "en" })).seccion("general");
  const mensajesConsulta = (await cargarTextos("cronos-consulta", { idioma: "en" })).seccion("consulta");
  const html = renderizarVistaMovimientosCronos({ estado: "listo", datos: respuesta(), mensajes, mensajesConsulta, locale: "en-GB" });
  assert.match(html, /Refresh|Previous days/u); assert.match(html, /Movements from 20\/09\/2026 to 20\/09\/2026/u);
  const borrador = renderizarVistaMovimientosCronos({ estado: "seleccion", consulta: { periodo: "rango", desde: '\"><script>alert(1)</script>', hasta: "" }, mensajes, mensajesConsulta });
  assert.doesNotMatch(borrador, /<script>/u); assert.match(borrador, /View movements/u);
});

test("una respuesta incompatible del cliente inyectado queda en error sin mostrar movimientos", async () => {
  const { nodo, raiz } = raizFalsa(); const datos = respuesta(); datos.periodo.tipo = "anio";
  const vista = montarVistaMovimientosCronos({ raiz, cliente: { consultar: async () => datos } });
  await completar(); assert.match(nodo.innerHTML, /data-cronos-movimientos-estado="error"/u);
  assert.doesNotMatch(nodo.innerHTML, /09:10|Entrada|Remoto/u); vista.desmontar();
});

test("editar un rango ya confirmado bloquea refresco y paginación hasta enviarlo, sin sustituir campos", async () => {
  const { nodo, raiz } = raizFalsa(); const llamadas = [];
  const vista = montarVistaMovimientosCronos({ raiz, cliente: { consultar: async (consulta) => { llamadas.push(consulta); return consulta.periodo === "hoy" ? respuesta() : anual("rango"); } } });
  await completar(); await vista.consultar({ periodo: "rango", desde: "2026-01-01", hasta: "2026-12-31" });
  pulsar(nodo, "cronosMovimientosPagina", "siguiente");
  const desde = { name: "desde", value: "2026-02-01" }; const hasta = { name: "hasta", value: "2026-12-31" };
  const formulario = { elements: { namedItem: (nombre) => nombre === "desde" ? desde : hasta } };
  desde.closest = () => formulario; raiz.ownerDocument.activeElement = desde;
  const html = nodo.innerHTML; nodo.eventos.input({ target: desde });
  await assert.doesNotReject(Promise.all([vista.actualizar(), Promise.resolve("fichaje confirmado")]));
  pulsar(nodo, "cronosMovimientosPagina", "siguiente");
  assert.equal(llamadas.length, 2); assert.equal(nodo.innerHTML, html);
  assert.equal(desde.value, "2026-02-01"); assert.equal(raiz.ownerDocument.activeElement, desde);
  vista.desmontar(); assert.equal(nodo.eventos.input, undefined);
});
