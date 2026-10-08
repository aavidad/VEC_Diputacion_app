import test from "node:test";
import assert from "node:assert/strict";
import { crearClienteHTTPContratacionTemporal } from "./cliente-http.js?v=20261008-alta-circular-v3";
import { crearAdaptadorHTTPExpedientesContratacionTemporal } from "./adaptador-http-expedientes.js?v=20261008-alta-circular-v3";
import { crearPresentadorExpedientesContratacionTemporal } from "./presentador-expedientes.js?v=20261008-alta-circular-v3";
import { montarModuloContratacionTemporal } from "./vista-expedientes.js?v=20261008-alta-rpt-circular-v4";

// La ficha real coloca la lista común de documentos (consulta y descarga
// autorizadas por el servidor) dentro de su panel «Documentos».
const EXPEDIENTE = "expediente:ct:documentos";

function resumen() {
  return {
    expediente_ref: EXPEDIENTE, numero_visible: "2026/CT-0002", version: 3,
    flujo_ref: "flujo:ct:general", flujo_version: 1, flujo_huella_sha256: "a".repeat(64),
    fase_clave: "solicitud", estado_clave: "en_curso", centro_ref: "centro:001",
    categoria_ref: "categoria:auxiliar", creado_en: "2026-09-03T08:00:00Z", actualizado_en: "2026-09-03T09:00:00Z",
  };
}

function detalle() {
  return {
    esquema: "vec.contratacion-temporal.detalle-rrhh.v1", resumen: resumen(),
    solicitud: { grupo_subgrupo: "C2", motivo_clave: "sustitucion", periodo_inicio: "2026-10-01T00:00:00Z", periodo_fin: "2026-12-31T00:00:00Z" },
    hitos: [{ secuencia: 1, version_expediente: 1, accion_clave: "registrar_solicitud",
      realizada_en: "2026-09-03T09:00:00Z", fase_destino: "solicitud", estado_origen: "en_curso", estado_destino: "en_curso" }],
  };
}

function raizFalsa() {
  const eventos = new Map(), vigentes = new Set();
  let html = "", zona = null;
  const documento = { createElement: (tag) => ({ tag, hijos: [], append(...n) { this.hijos.push(...n); } }) };
  const raiz = {
    ownerDocument: documento,
    get innerHTML() { return html; },
    set innerHTML(valor) {
      html = valor; vigentes.clear(); zona = null;
      if (valor.includes("data-ct-exp-documentos-comun")) {
        zona = { contenido: null, replaceChildren(nodo) { this.contenido = nodo ?? null; } };
      }
    },
    contains: (nodo) => vigentes.has(nodo),
    querySelector: (selector) => (selector === "[data-ct-exp-documentos-comun]" ? zona : null),
    querySelectorAll: () => [],
    addEventListener: (tipo, fn) => eventos.set(tipo, fn),
    removeEventListener: (tipo) => eventos.delete(tipo),
  };
  return {
    raiz,
    zona: () => zona,
    async abrir() {
      const control = { dataset: { ctExpAbrir: EXPEDIENTE }, closest: (s) => (s === "[data-ct-exp-abrir]" ? control : null) };
      vigentes.add(control);
      await eventos.get("click")({ target: control, preventDefault() {} });
    },
  };
}

async function montar(documentosComun) {
  const dom = raizFalsa();
  const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async (ruta) => {
    const data = ruta.endsWith("/cuadro/consultas")
      ? { esquema: "vec.contratacion-temporal.cuadro-rrhh.v1", generada_en: "2026-09-03T09:05:00Z", expedientes: [resumen()], hay_mas: false }
      : detalle();
    return new Response(JSON.stringify({ data }), { headers: { "content-type": "application/json; charset=utf-8" } });
  } });
  const fuente = crearAdaptadorHTTPExpedientesContratacionTemporal({ cliente });
  await fuente.listar();
  const presentador = crearPresentadorExpedientesContratacionTemporal({ fuente, capacidades: fuente.capacidades });
  const modulo = await montarModuloContratacionTemporal({ raiz: dom.raiz, presentador, documentosComun });
  await dom.abrir();
  await new Promise((r) => setImmediate(r));
  return { dom, modulo };
}

test("la ficha monta la lista común de documentos del expediente abierto y la retira al desmontar", async () => {
  const llamadas = [];
  let retiradas = 0;
  const { dom, modulo } = await montar({
    montar(opciones) {
      llamadas.push(opciones);
      return { desmontar: () => { retiradas += 1; } };
    },
  });
  assert.ok(llamadas.length >= 1, "monta la lista común al abrir la ficha");
  const ultima = llamadas.at(-1);
  assert.equal(ultima.expedienteRef, EXPEDIENTE);
  assert.equal(typeof ultima.anunciar, "function");
  assert.equal(dom.zona().contenido, ultima.raiz, "la lista sustituye al aviso dentro del panel");
  const antes = retiradas;
  modulo.desmontar();
  assert.equal(retiradas, antes + 1);
});

test("sin módulo de documentos publicado, la ficha conserva el aviso y no falla", async () => {
  const { dom, modulo } = await montar({ montar: () => null });
  assert.equal(dom.zona().contenido, null);
  assert.match(dom.raiz.innerHTML, /data-ct-exp-documentos-comun/u);
  modulo.desmontar();
});
