import assert from "node:assert/strict";
import test from "node:test";
import { readFileSync } from "node:fs";
import { webcrypto } from "node:crypto";
import { iniciarPreparacionLocal, iniciarPestanasOrganizacion, MONTAJE_ORGANIZACION_HISTORICA,
  PREPARACION_LOCAL_ORGANIZACION } from "./historico.js";
import { crearTraductorPersonal } from "../modulos/personal/i18n.js";

if (!globalThis.crypto) globalThis.crypto = webcrypto;
const traducir = (clave, variables) => crearTraductorPersonal()(`organizacion_${clave}`, variables);
const limite = 8 * 1024 * 1024;
const paquete = () => ({
  manifiesto: {
    tipo: "rpt", version_ref: "11111111-1111-4111-8111-111111111111", version_revision: 1,
    fuente_ref: "archivo_local_ejemplo", fuente_version: "Ejemplo 2026", fuente_huella_sha256: "a".repeat(64),
    catalogo_unidades: { id: "unidades", version: 1, revision: 1, huella_sha256: "b".repeat(64) },
    catalogo_clasificaciones: { id: "clases", version: 1, revision: 1, huella_sha256: "c".repeat(64) },
  },
  hechos: [{ clase: "nodo", hecho_ref: "22222222-2222-4222-8222-222222222222", revision: 1,
    fila_fuente_ref: "fila_1", unidad_ref: "centro_ejemplo", vigente_desde: "2026-09-25",
    catalogo_entrada_clave: "centro_ejemplo", denominacion: "Centro de Servicios Municipales", tipo_unidad: "centro" }],
});
const fichero = (valor, nombre = "organizacion-ejemplo.json") => {
  const bytes = typeof valor === "object" && !(valor instanceof Uint8Array)
    ? new TextEncoder().encode(JSON.stringify(valor))
    : typeof valor === "string" ? new TextEncoder().encode(valor) : valor;
  return { name: nombre, size: bytes.byteLength, arrayBuffer: async () => bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength) };
};
function documento() {
  const nodos = new Map();
  let activo = null;
  const obtener = (selector) => {
    if (!nodos.has(selector)) {
      const atributos = new Map();
      nodos.set(selector, { hidden: true, disabled: false, open: false, value: "", textContent: "", innerHTML: "",
        files: [], onclick: null, clicks: 0,
        click() { this.clicks += 1; return this.onclick?.(); },
        focus() { activo = this; }, setAttribute(k, v) { atributos.set(k, String(v)); }, getAttribute(k) { return atributos.get(k); } });
    }
    return nodos.get(selector);
  };
  return { querySelector: obtener, querySelectorAll: () => [], getElementById: (id) => obtener(`#${id}`), get activeElement() { return activo; } };
}
function preparar(t) {
  const doc = documento(), llamadas = [];
  const anterior = globalThis.fetch;
  globalThis.fetch = (...args) => { llamadas.push(args); throw new Error("HTTP inesperado"); };
  t.after(() => { globalThis.fetch = anterior; assert.deepEqual(llamadas, []); });
  const control = iniciarPreparacionLocal(PREPARACION_LOCAL_ORGANIZACION, doc);
  const q = doc.querySelector;
  const elegir = (archivo) => { q("#local-file").files = archivo ? [archivo] : []; return q("#local-file").onchange(); };
  return { doc, q, control, elegir };
}

test("la revisión local ofrece su pestaña sin activar rutas ni acciones de importación", async (t) => {
  const { doc, q, elegir } = preparar(t);
  const anterior = globalThis.document;
  globalThis.document = doc;
  t.after(() => { globalThis.document = anterior; });
  const pestanas = iniciarPestanasOrganizacion(undefined, PREPARACION_LOCAL_ORGANIZACION.habilitada);
  assert.deepEqual(MONTAJE_ORGANIZACION_HISTORICA, { consulta: false, importacion: false });
  assert.deepEqual(pestanas.pestanas, ["catalog", "local"]);
  pestanas.seleccionar("local");
  assert.equal(q("#local-panel").hidden, false);
  assert.equal(q("#history-panel").hidden, true);
  assert.equal(q("#import-panel").hidden, true);
  await elegir(fichero(paquete()));
  for (const id of ["import-prepare", "import-conciliate", "import-publish", "import-confirm", "import-retry"]) {
    q(`#${id}`).disabled = false;
    q(`#${id}`).click();
    assert.equal(q(`#${id}`).onclick, null);
  }
  const montaje = readFileSync(new URL("./organizacion.js", import.meta.url), "utf8");
  assert.doesNotMatch(montaje, /iniciarImportacion\(|crearClienteImportacion\(/);
});

test("el archivo válido muestra recuentos y detalle plegado, escapando sus referencias", async (t) => {
  const { q, elegir } = preparar(t);
  const ejemplo = paquete();
  ejemplo.manifiesto.fuente_ref = '<img src=x onerror="alert(1)">';
  await elegir(fichero(ejemplo, "<archivo>.json"));
  assert.equal(q("#local-state").textContent, traducir("localCompleted"));
  assert.equal(q("#local-results").hidden, false);
  assert.equal(q("#local-details").open, false);
  assert.equal(q("#local-file-name").textContent, "<archivo>.json");
  assert.match(q("#local-summary").innerHTML, /1/);
  assert.doesNotMatch(q("#local-summary").innerHTML, /archivo_local|onerror|SHA|Centro de/);
  assert.doesNotMatch(q("#local-technical").innerHTML, /<img/);
  assert.match(q("#local-technical").innerHTML, /&lt;img/);
  assert.match(q("#local-technical").innerHTML, /[a-f0-9]{64}/);
});

test("los errores de intervalo identifican manifiesto o fila y limpian el resultado anterior", async (t) => {
  const { q, elegir } = preparar(t);
  for (const hasta of ["2026-09-24", "2026-09-25"]) {
    await elegir(fichero(paquete()));
    const manifiesto = paquete();
    Object.assign(manifiesto.manifiesto, { efectos_desde: "2026-09-25", efectos_hasta: hasta });
    await elegir(fichero(manifiesto));
    assert.equal(q("#local-state").textContent, traducir("localManifestInterval"));
    const hecho = paquete();
    hecho.hechos[0].vigente_hasta = hasta;
    await elegir(fichero(hecho));
    assert.equal(q("#local-state").textContent, traducir("localRowInterval", { fila: 1 }));
    assert.equal(q("#local-file-name").textContent, "");
    assert.equal(q("#local-file").value, "");
    assert.equal(q("#local-results").hidden, true);
    assert.equal(q("#local-summary").innerHTML, "");
    assert.equal(q("#local-technical").innerHTML, "");
  }
});

test("JSON malformado, UTF-8 inválido y más de 3000 filas no dejan revisión ni envío", async (t) => {
  const { q, elegir } = preparar(t);
  const excesivo = paquete();
  excesivo.hechos = Array.from({ length: 3001 }, () => structuredClone(excesivo.hechos[0]));
  for (const datos of ["{", new Uint8Array([0xff, 0xfe]), excesivo]) {
    await elegir(fichero(paquete()));
    await elegir(fichero(datos));
    assert.equal(q("#local-state").textContent, traducir("localBadPackage"));
    assert.equal(q("#local-results").hidden, true);
    assert.equal(q("#local-file-name").textContent, "");
  }
});

test("las fechas inexistentes o mal formadas indican el manifiesto o la fila", async (t) => {
  const { q, elegir } = preparar(t);
  for (const fecha of ["2026-02-30", "25/09/2026", null, 0]) {
    const manifiesto = paquete();
    manifiesto.manifiesto.efectos_desde = fecha;
    await elegir(fichero(manifiesto));
    assert.equal(q("#local-state").textContent, traducir("localManifestDate"));
    const hecho = paquete();
    hecho.hechos[0].vigente_desde = fecha;
    await elegir(fichero(hecho));
    assert.equal(q("#local-state").textContent, traducir("localRowDate", { fila: 1 }));
  }
});

test("3000 hechos con identidad distinta se cuentan sin expandirlos ni modificar el archivo", async (t) => {
  const { q, elegir } = preparar(t);
  const ejemplo = paquete(), original = ejemplo.hechos[0];
  ejemplo.hechos = Array.from({ length: 3000 }, (_, indice) => ({ ...original,
    hecho_ref: `${indice.toString(16).padStart(8, "0")}-2222-4222-8222-222222222222`, fila_fuente_ref: `fila_${indice + 1}` }));
  await elegir(fichero(ejemplo));
  assert.equal(q("#local-state").textContent, traducir("localCompleted"));
  assert.match(q("#local-summary").innerHTML, /3\.000/);
  assert.equal(ejemplo.hechos.length, 3000);
});

test("el límite de 8 MiB se comprueba antes y después de leer los bytes", async (t) => {
  const { q, elegir } = preparar(t);
  let leido = false;
  await elegir({ name: "grande.json", size: limite + 1, arrayBuffer: async () => { leido = true; } });
  assert.equal(leido, false);
  assert.equal(q("#local-state").textContent, traducir("localTooLarge"));
  await elegir({ name: "tamano-incorrecto.json", size: 1, arrayBuffer: async () => new ArrayBuffer(limite + 1) });
  assert.equal(q("#local-state").textContent, traducir("localTooLarge"));
  const json = JSON.stringify(paquete());
  await elegir(fichero(json + " ".repeat(limite - json.length)));
  assert.equal(q("#local-state").textContent, traducir("localCompleted"));
});

test("cambiar o vaciar el archivo descarta lecturas antiguas y espera su turno", async (t) => {
  const { doc, q, elegir, control } = preparar(t);
  let terminar, iniciadas = 0;
  const viejo = fichero(paquete(), "anterior.json");
  viejo.arrayBuffer = () => { iniciadas += 1; return new Promise((resolve) => { terminar = resolve; }); };
  const pendiente = elegir(viejo);
  await Promise.resolve();
  const nuevo = fichero(paquete(), "actual.json"), leerNuevo = nuevo.arrayBuffer;
  nuevo.arrayBuffer = () => { iniciadas += 1; return leerNuevo(); };
  const actual = elegir(nuevo);
  assert.equal(iniciadas, 1);
  assert.equal(q("#local-file-name").textContent, "");
  terminar(await fichero(paquete()).arrayBuffer());
  await Promise.all([pendiente, actual]);
  assert.equal(iniciadas, 2);
  assert.equal(q("#local-file-name").textContent, "actual.json");
  control.limpiar();
  assert.equal(q("#local-state").textContent, traducir("localReady"));
  const porVaciar = elegir(viejo);
  await Promise.resolve();
  q("#local-clear").focus();
  q("#local-clear").click();
  assert.equal(q("#local-clear").disabled, true);
  assert.equal(doc.activeElement, q("#local-choose"));
  terminar(await fichero(paquete()).arrayBuffer());
  await porVaciar;
  assert.equal(q("#local-results").hidden, true);
  assert.equal(q("#local-file-name").textContent, "");
  await elegir();
  assert.equal(q("#local-state").textContent, traducir("localReady"));
});
