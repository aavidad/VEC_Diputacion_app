import assert from "node:assert/strict";
import test from "node:test";
import { RUTAS_CANCELACIONES_CENTRO, crearClienteCancelacionesCentro, montarCancelacionesCentro } from "./cancelaciones-centro.js";

const fila = (extra = {}) => ({ peticion_ref: "peticion:centro:1", expediente_ref: "expediente:ct:1", numero_visible: "2026/CT-000124", version: 3,
  fase: "solicitud", estado: "en_curso", modalidad_clave: "", categoria_ref: "cat:1", periodo: { inicio: "2026-10-01", fin: "2026-12-31" },
  confirmacion: null, documento_exigido: "", ...extra });
const opciones = (expedienteRef = "expediente:ct:1") => ({ esquema: "vec.contratacion-temporal.cancelacion-expediente.v1", expediente_ref: expedienteRef,
  fases_admitidas: ["solicitud", "asignacion_unidad"], motivos: [{ clave: "necesidad_desaparecida", etiqueta: "Ha desaparecido la necesidad", clave_i18n: "" }],
  cancelacion: null });
const recibo = { esquema: "vec.contratacion-temporal.recibo-cancelacion.v1", operacion: "cancelar_expediente", expediente_ref: "expediente:ct:1",
  version_anterior: 3, version_resultante: 4, fase_resultante: "solicitud", estado_resultante: "cancelado", motivo_clave: "necesidad_desaparecida",
  recibo_ref: "recibo:cancelacion:1", auditoria_ref: "auditoria:1", evento_ref: "evento:1", registrada_en: "2026-09-26T08:00:00Z" };
const esperar = () => new Promise((r) => setImmediate(r));

function contenedorFalso() {
  const eventos = new Map();
  return { innerHTML: "", hidden: true, eventos, ownerDocument: { addEventListener() {} },
    addEventListener: (n, f) => eventos.set(n, f), removeEventListener: (n) => eventos.delete(n), querySelector: () => null };
}

test("el cliente pide solo las rutas fijas del centro, sin caché, redirecciones ni referente", async () => {
  const llamadas = [];
  const fetchFalso = async (ruta, o) => {
    llamadas.push({ ruta, o });
    const data = ruta === RUTAS_CANCELACIONES_CENTRO.consulta ? opciones() : recibo;
    return { ok: true, status: 200, text: async () => JSON.stringify({ data }) };
  };
  const cliente = crearClienteCancelacionesCentro(fetchFalso);
  assert.equal((await cliente.consultar("expediente:ct:1")).motivos.length, 1);
  const r = await cliente.cancelar({ expediente_ref: "expediente:ct:1", version_esperada: 3, clave_idempotencia: "7c9e6679-7425-40de-944b-e07fc1f90ae7",
    motivo_clave: "necesidad_desaparecida", observaciones: "" });
  assert.equal(r.recibo_ref, "recibo:cancelacion:1");
  assert.deepEqual(llamadas.map((l) => l.ruta), [RUTAS_CANCELACIONES_CENTRO.consulta, RUTAS_CANCELACIONES_CENTRO.cancelaciones]);
  for (const { o } of llamadas) {
    assert.equal(o.method, "POST");
    assert.equal(o.credentials, "same-origin");
    assert.equal(o.mode, "same-origin");
    assert.equal(o.cache, "no-store");
    assert.equal(o.redirect, "error");
    assert.equal(o.referrerPolicy, "no-referrer");
  }
  await assert.rejects(() => crearClienteCancelacionesCentro(async () => ({ ok: true, status: 201, text: async () => JSON.stringify({ data: { ...recibo, estado_resultante: "en_curso" } }) }))
    .cancelar({ expediente_ref: "expediente:ct:1", version_esperada: 3, clave_idempotencia: "7c9e6679-7425-40de-944b-e07fc1f90ae7", motivo_clave: "necesidad_desaparecida", observaciones: "" }),
  (e) => e.indeterminado === true, "un recibo incoherente no se da por bueno");
});

test("solo ofrece cancelar en las fases que admite el catálogo y se oculta si el perfil no cancela", async () => {
  const oculto = contenedorFalso();
  montarCancelacionesCentro({ contenedor: oculto, bandeja: { bandeja: async () => ({ expedientes: [fila()] }) },
    cliente: { consultar: async () => { throw Object.assign(new Error("x"), { estado: 403 }); } } });
  await esperar(); await esperar();
  assert.equal(oculto.hidden, true);
  const c = contenedorFalso();
  const desmontar = montarCancelacionesCentro({ contenedor: c, bandeja: { bandeja: async () => ({ expedientes: [fila(),
    fila({ expediente_ref: "expediente:ct:2", numero_visible: "2026/CT-000125", fase: "nombramiento" }),
    fila({ expediente_ref: "expediente:ct:3", numero_visible: "2026/CT-000126", estado: "cancelado" })] }) },
  cliente: { consultar: async (ref) => opciones(ref) } });
  await esperar(); await esperar();
  assert.equal(c.hidden, false);
  assert.match(c.innerHTML, /data-cc-abrir="expediente:ct:1"/u);
  assert.doesNotMatch(c.innerHTML, /2026\/CT-000125/u, "fase fuera del catálogo");
  assert.match(c.innerHTML, /2026\/CT-000126[\s\S]*Cancelado/u);
  c.eventos.get("click")({ target: { closest: () => ({ matches: () => false, dataset: { ccAbrir: "expediente:ct:1" } }) } });
  assert.match(c.innerHTML, /data-cc-form="expediente:ct:1"/u);
  assert.match(c.innerHTML, /<option value="necesidad_desaparecida">Ha desaparecido la necesidad<\/option>/u);
  assert.doesNotMatch(c.innerHTML, /peticion:centro|expediente:ct:1<|recibo:/u, "sin referencias internas visibles");
  desmontar();
  assert.equal(c.eventos.size, 0);
});
