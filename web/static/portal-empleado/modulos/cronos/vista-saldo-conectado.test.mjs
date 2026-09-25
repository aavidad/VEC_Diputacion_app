import assert from "node:assert/strict";
import test from "node:test";
import { MENSAJES_CRONOS_ES } from "./i18n.js";
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
  const malicioso = renderizarVistaSaldoCronos({ estado: "cargando", mensajes: { ...MENSAJES_CRONOS_ES, saldo_titulo: '<img src=x onerror="x()">' } });
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
  assert.match(nodo.innerHTML, /<p class="cronos-vacio" role="status">El servicio de saldo no está disponible/u);
  assert.doesNotMatch(nodo.innerHTML, /No se pudo consultar/u);
  assert.doesNotMatch(nodo.innerHTML, /sobrelinea|<h2/u);
  assert.match(nodo.innerHTML, /<h3 id="cronos-saldo-titulo">/u);
  assert.match(renderizarVistaSaldoCronos({ estado: "cargando" }), /<p class="sobrelinea">[^<]+<\/p><h2 id="cronos-saldo-titulo">/u, "suelta conserva su encabezado de página");
  vista.desmontar();
});
