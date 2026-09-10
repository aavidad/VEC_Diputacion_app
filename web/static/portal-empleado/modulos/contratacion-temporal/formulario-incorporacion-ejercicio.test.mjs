import assert from "node:assert/strict";
import test from "node:test";
import { montarFormularioIncorporacionEjercicio } from "./formulario-incorporacion-ejercicio.js";

// Dobles de cliente y DOM, contrato real; no acredita HTTP, PostgreSQL ni montaje.
const expediente = "expediente:ejercicio:1", personal = "solicitud:personal:1";
const periodo = { desde: "2026-09-09T00:00:00Z", hasta: "2026-10-09T00:00:00Z" };
const recibo = { esquema: "vec.contratacion-temporal.incorporacion-ejercicio.recibo.v2",
  expediente_ref: expediente, solicitud_personal_ref: personal, relacion_ref: "relacion:personal:1",
  recibo_ref: "recibo:ct:original", actuacion_ref: "actuacion:ct:1", registrada_en: "2026-09-09T01:00:00Z",
  periodo_incorporacion: periodo, version_solicitud_personal: 7, version_actual_expediente: 8,
  seguimiento_ref: "seguimiento:ct:1", version_seguimiento_anterior: 0, version_seguimiento_resultante: 1,
  auditoria_ref: "auditoria:ct:1", outbox_ref: "outbox:ct:1", ejercicio_sintetico: true,
  firma_oficial: false, eficacia_administrativa: false };
const preparacion = { esquema: "vec.contratacion-temporal.incorporacion-ejercicio.preparacion.v2",
  expediente_ref: expediente, version_actual_expediente: 8, recibo: null,
  preparacion: { solicitud_personal_ref: personal, version_solicitud_personal: 7,
    version_seguimiento_esperada: 0, periodo_incorporacion: periodo,
    motivos: ["confirmar_incorporacion"], documentos_refs: ["documento:ejercicio:1"], disponible: true } };
const valores = { motivo_clave: "confirmar_incorporacion", confirma_revision_personal: true, confirma_ejercicio_sintetico: true };
const historia = () => ({ ...preparacion, version_actual_expediente: 11, preparacion: null, recibo });
function raizFormulario() {
  const eventos = new Map();
  return { innerHTML: "", addEventListener: (k, f) => eventos.set(k, f),
    removeEventListener: (k) => eventos.delete(k), replaceChildren() { this.innerHTML = ""; },
    enviar(v = valores) {
      const elements = Object.fromEntries(Object.entries(v).map(([k, x]) => [k, { value: x, checked: x }]));
      return eventos.get("submit")?.({ preventDefault() {}, target: { elements } });
    } };
}
function montar(cliente, proyeccion = preparacion, confirmarOperacion = () => true) {
  const raiz = raizFormulario();
  const desmontar = montarFormularioIncorporacionEjercicio({ raiz, cliente, preparacion: proyeccion, confirmarOperacion });
  return { raiz, desmontar };
}
test("nominal: intención exacta sin ids, un envío y recibo original", async () => {
  const posts = [];
  const x = montar({ prepararIncorporacionEjercicio() { assert.fail("GET inesperado"); },
    async confirmarIncorporacionEjercicio(s) { posts.push(s); return recibo; } });
  assert.doesNotMatch(x.raiz.innerHTML, / checked/u);
  const vuelo = x.raiz.enviar(); await x.raiz.enviar(); await vuelo;
  assert.deepEqual(posts, [{ expediente_ref: expediente, solicitud_personal_ref: personal,
    version_actual_expediente_observada: 8, documentos_refs: ["documento:ejercicio:1"], ...valores }]);
  assert.match(x.raiz.innerHTML, /recibo:ct:original/u);
  assert.doesNotMatch(x.raiz.innerHTML, /data-ct-incorporacion-ejercicio-form/u);
  x.desmontar();
});
test("dos revisiones previas y confirmación expresa obligatorias", async () => {
  let posts = 0, revisiones = 0;
  const cliente = { prepararIncorporacionEjercicio() { assert.fail(); },
    confirmarIncorporacionEjercicio() { posts++; return recibo; } };
  const x = montar(cliente, preparacion, () => { revisiones++; return false; });
  for (const campo of ["confirma_revision_personal", "confirma_ejercicio_sintetico"]) {
    await x.raiz.enviar({ ...valores, [campo]: false });
  }
  assert.equal(revisiones, 0); assert.equal(posts, 0);
  await x.raiz.enviar(); assert.equal(revisiones, 1); assert.equal(posts, 0);
  x.desmontar();
});
test("incertidumbre y conflicto: GET recupera historia original sin otro POST", async () => {
  for (const estado of [409, 503]) {
    const llamadas = [];
    const x = montar({ async confirmarIncorporacionEjercicio() {
      llamadas.push("POST"); throw Object.assign(new Error("detalle privado"), { estado });
    }, async prepararIncorporacionEjercicio(ref) {
      assert.equal(ref, expediente); llamadas.push("GET"); return historia();
    } });
    await x.raiz.enviar(); await x.raiz.enviar();
    assert.deepEqual(llamadas, ["POST", "GET"]);
    assert.match(x.raiz.innerHTML, /recibo:ct:original/u);
    assert.match(x.raiz.innerHTML, /Versión actual del expediente<\/dt><dd>11/u);
    assert.match(x.raiz.innerHTML, /Versión original del expediente<\/dt><dd>8/u);
    assert.doesNotMatch(x.raiz.innerHTML, /detalle privado|data-ct-incorporacion-ejercicio-form/u);
    x.desmontar();
  }
});
test("GET sin recibo mantiene intención inmutable para reintento", async () => {
  const posts = [];
  const x = montar({ async confirmarIncorporacionEjercicio(s) {
    posts.push(s); if (posts.length === 1) throw new Error(); return recibo;
  }, async prepararIncorporacionEjercicio() { return preparacion; } });
  await x.raiz.enviar(); assert.match(x.raiz.innerHTML, /<fieldset disabled/u);
  await x.raiz.enviar({ ...valores, motivo_clave: "otro" });
  assert.equal(posts.length, 2); assert.deepEqual(posts[0], posts[1]); x.desmontar();
});
test("historia inicial solo muestra recibo; sin efectos", async () => {
  const x = montar({ prepararIncorporacionEjercicio() { assert.fail(); },
    confirmarIncorporacionEjercicio() { assert.fail(); } }, historia());
  await x.raiz.enviar(); assert.match(x.raiz.innerHTML, /recibo:ct:original/u); x.desmontar();
});
test("desmontaje aborta POST o GET y descarta respuesta tardía", async () => {
  for (const fase of ["POST", "GET"]) {
    let resolver, signal;
    const pendiente = (_s, opciones) => { signal = opciones.signal; return new Promise((r) => { resolver = r; }); };
    const x = montar({ confirmarIncorporacionEjercicio: fase === "POST" ? pendiente : async () => { throw new Error(); },
      prepararIncorporacionEjercicio: pendiente });
    const vuelo = x.raiz.enviar(); await new Promise((r) => setImmediate(r));
    x.desmontar(); assert.equal(signal.aborted, true);
    resolver(fase === "POST" ? recibo : historia()); await vuelo;
    assert.equal(x.raiz.innerHTML, ""); assert.equal(x.raiz.enviar(), undefined);
  }
});
