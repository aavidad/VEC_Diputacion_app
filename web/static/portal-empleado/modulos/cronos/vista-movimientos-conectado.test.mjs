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
  assert.match(html, /data-cronos-accion="solicitar-correccion" disabled/);
  assert.match(html, /title="La solicitud de corrección no está disponible/);
  assert.doesNotMatch(html, /<p[^>]*>La solicitud de corrección/u);
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
  nodo.eventos.click({ target: { closest: () => boton } });
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
