import test from "node:test";
import assert from "node:assert/strict";
import { crearClienteConsultaSeguimientoInterno, montarConsultaSeguimientoInterno } from "./consulta-seguimiento.js";

const expedienteRef = "expediente:ct:consulta";
function vista(ref = expedienteRef) {
  return {
    esquema: "vec.contratacion-temporal.seguimiento-incorporacion.v2",
    alcance: "original_incorporacion", expediente_ref: ref, version_expediente: 8,
    recibo_incorporacion_ref: "recibo:incorporacion:1", seguimiento_ref: "seguimiento:1",
    version_seguimiento: 1, estado_clave: "vigente",
    periodo: { desde: "2027-01-01T00:00:00Z", hasta: "2027-03-31T00:00:00Z" },
    registrado_en: "2026-09-10T13:07:06Z",
    actuaciones: [{ actuacion_ref: "actuacion:1", transicion_clave: "confirmar_incorporacion",
      estado_origen: "pendiente_incorporacion", estado_destino: "vigente",
      efectivo_en: "2027-01-01T00:00:00Z", registrada_en: "2026-09-10T13:07:06Z",
      documentos: [{ tipo_clave: "justificante", referencia: "documento:1" }] }],
    ejercicio_sintetico: true, firma_oficial: false, eficacia_administrativa: false,
  };
}

function documentoPrueba() {
  const eventos = new Map();
  const formulario = { addEventListener: (n, f) => eventos.set(`form:${n}`, f),
    removeEventListener: (n) => eventos.delete(`form:${n}`) };
  const entrada = { value: "", focus() { this.enfocada = true; },
    addEventListener: (n, f) => eventos.set(`input:${n}`, f),
    removeEventListener: (n) => eventos.delete(`input:${n}`) };
  const estado = { textContent: "" };
  const panel = { hidden: false };
  const resultado = { innerHTML: "", replaceChildren() { this.innerHTML = ""; } };
  const nodos = new Map([
    ["[data-ct-consulta-form]", formulario], ["#ct-consulta-expediente", entrada],
    ["[data-ct-consulta-estado]", estado], ["[data-ct-consulta-resultado-panel]", panel],
    ["[data-ct-consulta-resultado]", resultado],
  ]);
  const documento = { title: "", querySelector: (s) => nodos.get(s), querySelectorAll: () => [] };
  return { documento, entrada, estado, panel, resultado,
    enviar: () => eventos.get("form:submit")({ preventDefault() {} }),
    cambiar: () => eventos.get("input:input")(),
  };
}

test("consulta interna omite credenciales web y no lee cookies ni almacenamiento", async () => {
  const ui = documentoPrueba();
  Object.defineProperty(ui.documento, "cookie", { get() { throw new Error("cookie leída"); } });
  const anteriores = new Map();
  for (const nombre of ["localStorage", "sessionStorage", "indexedDB"]) {
    anteriores.set(nombre, Object.getOwnPropertyDescriptor(globalThis, nombre));
    Object.defineProperty(globalThis, nombre, { configurable: true, get() { throw new Error(`${nombre} leído`); } });
  }
  const efectivas = [];
  try {
    const cliente = crearClienteConsultaSeguimientoInterno(async (ruta, opciones) => {
      efectivas.push({ ruta, opciones });
      if (opciones.credentials !== "omit" || opciones.headers.has("cookie")
        || opciones.headers.has("authorization")) throw new Error("credencial web enviada");
      return new Response(JSON.stringify({ data: vista() }), {
        headers: { "content-type": "application/json; charset=utf-8" },
      });
    });
    const destruir = montarConsultaSeguimientoInterno({ documento: ui.documento, cliente });
    ui.entrada.value = expedienteRef;
    await ui.enviar();
    assert.equal(efectivas.length, 1);
    assert.equal(efectivas[0].ruta,
      `/api/vec/contratacion-temporal/incorporaciones-ejercicio/seguimiento?expediente_ref=${encodeURIComponent(expedienteRef)}`);
    assert.equal(efectivas[0].opciones.method, "GET");
    assert.equal(efectivas[0].opciones.body, undefined);
    assert.equal(efectivas[0].opciones.cache, "no-store");
    assert.deepEqual([...efectivas[0].opciones.headers.keys()], ["accept"]);
    assert.equal(ui.panel.hidden, false);
    destruir();
  } finally {
    for (const [nombre, descriptor] of anteriores) {
      if (descriptor) Object.defineProperty(globalThis, nombre, descriptor);
      else delete globalThis[nombre];
    }
  }
});

test("consulta interna valida referencia, pinta el GET y borra datos al cambiar expediente", async () => {
  const ui = documentoPrueba();
  const llamadas = [];
  const destruir = montarConsultaSeguimientoInterno({ documento: ui.documento,
    cliente: { async consultar(ref, opciones) {
      llamadas.push({ ref, signal: opciones.signal });
      return vista(ref);
    } },
  });
  assert.equal(ui.panel.hidden, true);
  assert.match(ui.estado.textContent, /Sin expediente seleccionado/u);
  ui.entrada.value = "no válido";
  await ui.enviar();
  assert.equal(llamadas.length, 0);
  assert.equal(ui.entrada.enfocada, true);
  ui.entrada.value = expedienteRef;
  await ui.enviar();
  assert.deepEqual(llamadas.map((l) => l.ref), [expedienteRef]);
  assert.equal(ui.panel.hidden, false);
  assert.match(ui.resultado.innerHTML, /Vigente/u);
  assert.match(ui.resultado.innerHTML, /<details class="ct-seguimiento-trazabilidad">/u);
  ui.entrada.value = "expediente:ct:otro";
  ui.cambiar();
  assert.equal(ui.panel.hidden, true);
  assert.equal(ui.resultado.innerHTML, "");
  destruir();
});

test("consulta interna descarta respuesta tardía y denegación sin exponer datos", async () => {
  const ui = documentoPrueba();
  let resolver;
  const destruir = montarConsultaSeguimientoInterno({ documento: ui.documento,
    cliente: { consultar(ref) {
      if (ref === expedienteRef) return new Promise((resolve) => { resolver = resolve; });
      return Promise.reject({ estado: 403 });
    } },
  });
  ui.entrada.value = expedienteRef;
  const primera = ui.enviar();
  ui.entrada.value = "expediente:ct:otro";
  ui.cambiar();
  await ui.enviar();
  resolver(vista());
  await primera;
  assert.equal(ui.panel.hidden, true);
  assert.equal(ui.resultado.innerHTML, "");
  assert.match(ui.estado.textContent, /No hay seguimiento disponible/u);
  destruir();
});
