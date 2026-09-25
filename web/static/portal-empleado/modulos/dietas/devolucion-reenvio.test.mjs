// D6: la persona ve su documento devuelto con el motivo, lo corrige y lo
// reenvía a revisión del administrativo; nunca lo elimina.
import assert from "node:assert/strict";
import test from "node:test";

import { montarVistaBorradoresPropios } from "./vista-borradores-propios.js";
import { crearClienteBorradoresDietasHTTP } from "./cliente-borradores-http.js";

// DOM mínimo: hijos, atributos, data-* y búsqueda por etiqueta o data-*.
const claveDatos = (atributo) => atributo.slice(5).replace(/-([a-z])/g, (_m, letra) => letra.toUpperCase());
class Nodo {
  constructor(documento, etiqueta = "div") {
    Object.assign(this, { ownerDocument: documento, tagName: etiqueta, children: [], dataset: {}, attrs: {}, listeners: {},
      parent: null, textContent: "", disabled: false, value: "" });
  }
  append(...nodos) { this.children.push(...nodos); nodos.forEach((nodo) => { nodo.parent = this; }); }
  replaceChildren(...nodos) { this.children = []; this.append(...nodos); }
  removeChild(nodo) { this.children = this.children.filter((hijo) => hijo !== nodo); nodo.parent = null; }
  remove() { this.parent?.removeChild(this); }
  addEventListener(tipo, listener) { this.listeners[tipo] = listener; }
  removeEventListener(tipo) { delete this.listeners[tipo]; }
  setAttribute(nombre, valor) { this.attrs[nombre] = String(valor); }
  focus() { this.ownerDocument.activeElement = this; }
  matches(selector) {
    if (!selector.startsWith("[")) return this.tagName === selector;
    const [, atributo, esperado] = selector.match(/^\[([^=\]]+)(?:="([^"]*)")?\]$/u);
    const valor = this.dataset[claveDatos(atributo)];
    return valor !== undefined && (esperado === undefined || valor === esperado);
  }
  closest(selector) { for (let actual = this; actual; actual = actual.parent) if (actual.matches(selector)) return actual; return null; }
  querySelector(selector) { return this.querySelectorAll(selector)[0] || null; }
  querySelectorAll(selector) {
    const salida = [];
    const visitar = (actual) => { if (actual.matches(selector)) salida.push(actual); actual.children.forEach(visitar); };
    visitar(this);
    return salida;
  }
}
const raiz = () => { const documento = { createElement: (etiqueta) => new Nodo(documento, etiqueta) }; return new Nodo(documento, "root"); };
const textoVisible = (nodo) => [nodo.textContent, ...nodo.children.map(textoVisible)].join(" ");

const relacion = "rel_1234567890123456789012";
const tarifa = "provisional:rd462:20260923";
const calculo = { procedencia: "sin_vehiculo_propio", motor: "no_aplica", version_grafo: "no_aplica", version_tarifa: tarifa,
  rotulo: "PROVISIONAL · pendiente de confirmación por RRHH", hora_inicio: "09:00", hora_fin: "18:00",
  kilometros: "0.0000", eur_por_km: "0.2600", importe_kilometraje_centimos: 0, tramos_ruta: [],
  opciones_dieta: [1, 2, 3].map((grupo) => ({ grupo, calculo: { total_maximo_orientativo_centimos: 0, tramos: [] } })) };
const documentoComision = { vehiculo_propio: false, grupo_dieta: 2, version_tarifa_aceptada: tarifa, tramos_aceptados: [],
  lineas: [], manutencion_centimos: 0, alojamiento_tope_centimos: 0, kilometraje_centimos: 0, otros_centimos: 0,
  total_orientativo_centimos: 0 };
const devolucion = Object.freeze({ etapa: "autorizacion", motivo: "Falta el justificante del taxi", version: 4,
  devuelta_en: "2026-09-23T09:00:00.123456Z" });
const devuelta = Object.freeze({
  comision: { referencia: "dco_1234567890123456789012", numero_documento: "VEC-D-2026-000001",
    fecha_apertura: "2026-09-20T08:00:00.000000Z", version: 4, estado: "devuelta", fecha_inicio: "2026-09-20",
    fecha_fin: "2026-09-20", motivo: "Reunión", codigos_ruta: ["18087", "18003"], relacion_ref: relacion,
    centro_ref: "centro:uno", unidad_ref: "unidad:uno", calculo, vehiculo_propio: false, rutas: [],
    documento: documentoComision, devolucion },
  recibo: { referencia: "rcd_1234567890123456789012", version: 4, registrado_en: "2026-09-23T09:00:00.123456Z", repeticion: true },
});
const asignacion = { verificada: true, asignacion_ref: "ads_1234567890123456789012", relacion_ref: relacion,
  unidad_ref: "unidad:uno", fecha_referencia: "2026-09-24", centro_ref: "centro:uno",
  administrativo_persona_ref: "per_aaaaaaaaaaaaaaaaaaaaaa", responsable_persona_ref: "per_bbbbbbbbbbbbbbbbbbbbbb",
  grupo_dieta: 2, version: 1 };
const respuesta = (cuerpo, estado = 200) => new Response(JSON.stringify(cuerpo), {
  status: estado, headers: { "Content-Type": "application/json; charset=utf-8" } });

async function abrir(item, opciones = {}) {
  const contenedor = raiz();
  const vista = montarVistaBorradoresPropios(contenedor, {
    relacionesAutorizadas: [{ relacion_ref: relacion, unidad_ref: "unidad:uno" }],
    fechaReferenciaPersonal: "2026-09-24",
    clienteAsignacion: { obtener: async () => asignacion },
    cliente: { listar: async () => ({ items: [item] }), obtener: async () => item, crear: async () => item,
      editar: async () => item, borrar: async () => item, enviar: async () => item },
    ...opciones,
  });
  await Promise.resolve(); await Promise.resolve();
  const panel = contenedor.querySelector("[data-dietas-borradores-propios]");
  await panel.listeners.click({ target: contenedor.querySelector("[data-dietas-borrador-detalle]") });
  await Promise.resolve(); await Promise.resolve();
  return { contenedor, vista, panel };
}

test("el cliente acepta la devolución vigente con centro y unidad del documento v2", async () => {
  const cliente = crearClienteBorradoresDietasHTTP({ fetchImpl: async () => respuesta(devuelta) });
  const leido = await cliente.obtener(devuelta.comision.referencia);
  assert.deepEqual(leido.comision.devolucion, devolucion);
  assert.ok(Object.isFrozen(leido.comision.devolucion));
  assert.equal(leido.comision.centro_ref, "centro:uno");
});

test("el cliente rechaza una devolución incoherente con el estado o con forma inesperada", async () => {
  for (const comision of [
    { ...devuelta.comision, estado: "pendiente_autorizacion" },
    { ...devuelta.comision, devolucion: { ...devolucion, etapa: "otra" } },
    { ...devuelta.comision, devolucion: { ...devolucion, motivo: "no" } },
    { ...devuelta.comision, devolucion: { ...devolucion, version: 5 } },
    { ...devuelta.comision, devolucion: { ...devolucion, devuelta_en: "2026-09-23T09:00:00Z" } },
    { ...devuelta.comision, devolucion: { ...devolucion, actor_ref: "per_aaaaaaaaaaaaaaaaaaaaaa" } },
  ]) {
    const cliente = crearClienteBorradoresDietasHTTP({ fetchImpl: async () => respuesta({ ...devuelta, comision }) });
    await assert.rejects(() => cliente.obtener(devuelta.comision.referencia), /devolución de Dietas incompatible/u);
  }
});

test("el documento devuelto muestra etapa, fecha y motivo y ofrece corregir y reenviar, no eliminar", async () => {
  const { contenedor, vista, panel } = await abrir(devuelta);
  try {
    const bloque = contenedor.querySelector("[data-dietas-comision-devolucion]");
    assert.ok(bloque);
    const texto = textoVisible(bloque);
    assert.match(texto, /Devuelta para corregir/u);
    assert.match(texto, /Autorización del responsable/u);
    assert.match(texto, /Falta el justificante del taxi/u);
    assert.equal(contenedor.querySelector("[data-dietas-borrador-eliminar]"), null);
    const editar = contenedor.querySelector("[data-dietas-borrador-editar]");
    const enviar = contenedor.querySelector("[data-dietas-borrador-enviar]");
    assert.equal(editar.textContent, "Corregir");
    assert.equal(editar.disabled, false);
    assert.equal(enviar.textContent, "Reenviar a revisión del administrativo");
    assert.equal(enviar.disabled, false);
    // Corregir abre la edición con la declaración devuelta.
    await panel.listeners.click({ target: editar });
    const form = contenedor.querySelector("[data-dietas-borrador-form]");
    assert.equal(form.hidden, false);
    assert.equal(form.querySelectorAll("input").find((campo) => campo.name === "motivo")?.value, "Reunión");
  } finally { vista.desmontar(); }
});

test("reenviar pide confirmación, envía la versión devuelta con su clave y confirma la vuelta a revisión", async () => {
  const solicitudes = []; const confirmaciones = [];
  const reenviado = { ...devuelta, comision: { ...devuelta.comision, estado: "enviado_pendiente_revision", version: 6,
    devolucion: undefined }, recibo: { ...devuelta.recibo, version: 6, repeticion: false } };
  delete reenviado.comision.devolucion;
  let listados = 0;
  const { contenedor, vista, panel } = await abrir(devuelta, {
    cliente: { listar: async () => ({ items: [listados++ === 0 ? devuelta : reenviado] }), obtener: async () => devuelta,
      crear: async () => reenviado, editar: async () => reenviado, borrar: async () => reenviado,
      enviar: async (referencia, entrada) => { solicitudes.push([referencia, entrada]); return reenviado; } },
    confirmarOperacion: (texto) => { confirmaciones.push(texto); return true; },
    generarClaveIdempotencia: () => "reenviar-devuelta-20260924",
  });
  try {
    await panel.listeners.click({ target: contenedor.querySelector("[data-dietas-borrador-enviar]") });
    await Promise.resolve(); await Promise.resolve();
    assert.deepEqual(confirmaciones, ["¿Reenviar el documento a revisión del administrativo?"]);
    assert.deepEqual(solicitudes, [[devuelta.comision.referencia,
      { clave_idempotencia: "reenviar-devuelta-20260924", version_esperada: 4, relacion_ref: relacion }]]);
    assert.match(textoVisible(contenedor), /Reenvío registrado con recibo\. El documento vuelve a revisión del administrativo\./u);
    assert.equal(contenedor.querySelector("[data-dietas-comision-devolucion]"), null);
  } finally { vista.desmontar(); }
});

test("un documento en corrección se rotula como tal y los ya reenviados no admiten acciones", async () => {
  const enCorreccion = { ...devuelta, comision: { ...devuelta.comision, estado: "borrador", version: 5 } };
  const primera = await abrir(enCorreccion);
  try {
    const chip = primera.contenedor.querySelector("[data-dietas-estado-comision]");
    assert.equal(chip.textContent, "En corrección");
    assert.equal(primera.contenedor.querySelector("[data-dietas-borrador-eliminar]"), null);
    assert.equal(primera.contenedor.querySelector("[data-dietas-borrador-enviar]").textContent, "Reenviar a revisión del administrativo");
  } finally { primera.vista.desmontar(); }
  const pendiente = { ...devuelta, comision: { ...devuelta.comision, estado: "pendiente_autorizacion" } };
  delete pendiente.comision.devolucion;
  const segunda = await abrir(pendiente);
  try {
    assert.equal(segunda.contenedor.querySelector("[data-dietas-comision-devolucion]"), null);
    for (const selector of ["[data-dietas-borrador-editar]", "[data-dietas-borrador-eliminar]", "[data-dietas-borrador-enviar]"])
      assert.equal(segunda.contenedor.querySelector(selector).disabled, true);
  } finally { segunda.vista.desmontar(); }
});
