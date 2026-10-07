import assert from "node:assert/strict";
import test from "node:test";
import { cargarTextos } from "../../comun/textos.js";
import { IDIOMAS_DISPONIBLES } from "../../comun/idioma.js";
import { montarModulos, prepararSolicitud, validarConsulta } from "./vista.js";

const fuente = () => ({ version: 1, huella_sha256: "a".repeat(64), escritura_disponible: false, modulos: [
  { modulo_id: "vec.module.cronos", nombre_key: "ui.vec.module.cronos.name", habilitado: true, gobernado: true },
  { modulo_id: "vec.module.administracion", nombre_key: "ui.vec.module.administracion.name", habilitado: true, gobernado: false },
] });
const diferida = () => {
  let resolver;
  const promesa = new Promise((r) => { resolver = r; });
  return { promesa, resolver };
};

// DOM mínimo para comprobar eventos, limpieza y respuestas tardías, sin otro runtime.
class Nodo {
  constructor() { this.attrs = {}; this.listeners = {}; this.value = ""; this.hidden = false; this.disabled = false; this.textContent = ""; }
  setAttribute(k, v) { this.attrs[k] = v; }
  removeAttribute(k) { delete this.attrs[k]; }
  addEventListener(k, fn) { (this.listeners[k] ??= []).push(fn); }
  removeEventListener(k, fn) { this.listeners[k] = this.listeners[k].filter((f) => f !== fn); }
  async emitir(k) { await Promise.all((this.listeners[k] ?? []).map((fn) => fn({ preventDefault() {} }))); }
  focus() { this.enfocado = true; }
  replaceChildren() { this.innerHTML = ""; }
}
class Raiz extends Nodo {
  set innerHTML(html) {
    this.html = html;
    this.nodos = new Map([...html.matchAll(/id="([^"]+)"/gu)].map((m) => [m[1], new Nodo()]));
  }
  get innerHTML() { return this.html; }
  querySelector(selector) { return this.nodos.get(selector.slice(1)); }
  campo(nombre) { return [...this.nodos.entries()].find(([k]) => k.endsWith(`-${nombre}`))?.[1]; }
}
const textos = await cargarTextos("administracion-modulos");
const t = textos.traducir;

test("prepara la revisión con la versión y huella consultadas sin cambiar la fuente", () => {
  const original = fuente();
  const copia = structuredClone(original);
  assert.deepEqual(prepararSolicitud(original, "vec.module.cronos", false, "  Revisión programada  "), {
    modulo_id: "vec.module.cronos", habilitado: false, version_esperada: 1,
    huella_esperada: "a".repeat(64), motivo: "Revisión programada",
  });
  assert.deepEqual(original, copia);
  assert.equal(prepararSolicitud(original, "vec.module.cronos", false, "Revisión\nprogramada").motivo, "Revisión\nprogramada");
  assert.ok(Object.isFrozen(validarConsulta(original).modulos[0]));
  for (const args of [["vec.module.administracion", false, "motivo"], ["vec.module.cronos", true, "motivo"],
    ["vec.module.cronos", false, "  "], ["vec.module.cronos", false, "a".repeat(501)], ["vec.module.otro", false, "motivo"]]) {
    assert.throws(() => prepararSolicitud(original, ...args));
  }
});

test("rechaza catálogo inválido o que anuncie capacidad de escritura", () => {
  for (const parcial of [{ escritura_disponible: true }, { version: 0 }, { version: "1" },
    { huella_sha256: "huella" }, { modulos: [fuente().modulos[0], fuente().modulos[0]] },
    { modulos: [{ ...fuente().modulos[0], habilitado: "true" }] },
    { modulos: [{ ...fuente().modulos[0], modulo_id: "../cronos" }] }]) {
    assert.throws(() => validarConsulta({ ...fuente(), ...parcial }));
  }
});

test("admite los límites canónicos y muestra nombre pendiente cuando falta una traducción", async () => {
  const modulos = Array.from({ length: 128 }, (_, n) => ({ ...fuente().modulos[0],
    modulo_id: `vec.module.m${n}`, nombre_key: "a".repeat(256), gobernado: false }));
  assert.equal(validarConsulta({ ...fuente(), modulos }).modulos.length, 128);
  const root = new Raiz();
  const vista = montarModulos(root, { t, consultar: async () => ({ ...fuente(), modulos: [modulos[0]] }) });
  await vista.listo;
  assert.match(root.campo("filas").innerHTML, new RegExp(t("consulta.nombre_pendiente"), "u"));
  assert.doesNotMatch(root.campo("filas").innerHTML, /a{256}/u);
});

test("sin consulta inyectada no inventa estados ni invoca preparación", async () => {
  const root = new Raiz();
  let preparaciones = 0;
  const vista = montarModulos(root, { t, preparar: () => { preparaciones++; } });
  await vista.listo;
  assert.equal(root.campo("estado").textContent, t("consulta.pendiente"));
  assert.equal(root.campo("editor").hidden, true);
  assert.equal(root.campo("tabla").hidden, true);
  await root.campo("preparar").emitir("click");
  assert.equal(preparaciones, 0);
  vista.desmontar();
});

test("revisar no llama al puerto; preparar una vez conserva la configuración actual", async () => {
  const root = new Raiz();
  const llamadas = [];
  const vista = montarModulos(root, { t, consultar: async () => fuente(), preparar: async (cambio) => {
    llamadas.push(cambio);
    return { cambio, publicado: false, version_siguiente: 2 };
  } });
  await vista.listo;
  assert.doesNotMatch(root.campo("modulo").innerHTML, /value="vec\.module\.administracion"/u);
  root.campo("modulo").value = "vec.module.cronos";
  root.campo("habilitado").value = "false";
  root.campo("motivo").value = "Revisión programada";
  await root.campo("formulario").emitir("submit");
  assert.equal(llamadas.length, 0);
  assert.equal(root.campo("revision").hidden, false);
  const filas = root.campo("filas").innerHTML;
  await root.campo("preparar").emitir("click");
  await root.campo("preparar").emitir("click");
  assert.equal(llamadas.length, 1);
  assert.equal(root.campo("resultado").textContent, t("revision.preparada"));
  assert.equal(root.campo("filas").innerHTML, filas);
});

test("errores y preparación no válida no se anuncian como éxito", async () => {
  for (const respuesta of [undefined, { publicado: true }, { publicado: false, cambio: {}, version_siguiente: 2 }]) {
    const root = new Raiz();
    const vista = montarModulos(root, { t, consultar: async () => fuente(), preparar: async () => respuesta });
    await vista.listo;
    await root.campo("formulario").emitir("submit");
    assert.equal(root.campo("errores").hidden, false);
    root.campo("modulo").value = "vec.module.cronos";
    root.campo("habilitado").value = "false";
    root.campo("motivo").value = "Motivo";
    await root.campo("formulario").emitir("submit");
    await root.campo("preparar").emitir("click");
    assert.equal(root.campo("resultado").textContent, t("errores.preparacion"));
    vista.desmontar();
  }
});

test("ignora consulta tardía tras recarga y desmontaje; aborta el puerto", async () => {
  const root = new Raiz();
  const pendientes = [];
  const vista = montarModulos(root, { t, consultar: ({ signal }) => {
    const d = diferida(); pendientes.push({ ...d, signal }); return d.promesa;
  } });
  const recarga = vista.cargar();
  assert.equal(pendientes[0].signal.aborted, true);
  pendientes[1].resolver({ ...fuente(), modulos: [] });
  await recarga;
  pendientes[0].resolver(fuente());
  await vista.listo;
  assert.equal(root.campo("estado").textContent, t("consulta.vacia"));
  const ultima = vista.cargar();
  vista.desmontar();
  pendientes[2].resolver(fuente());
  await ultima;
  assert.equal(pendientes[2].signal.aborted, true);
  assert.equal(root.innerHTML, "");
});

test("preparación tardía no reconstruye la pantalla desmontada", async () => {
  const root = new Raiz();
  const d = diferida();
  let señal;
  const vista = montarModulos(root, { t, consultar: async () => fuente(), preparar: (_, { signal }) => { señal = signal; return d.promesa; } });
  await vista.listo;
  root.campo("modulo").value = "vec.module.cronos";
  root.campo("habilitado").value = "false";
  root.campo("motivo").value = "Motivo";
  await root.campo("formulario").emitir("submit");
  const pendiente = root.campo("preparar").emitir("click");
  vista.desmontar();
  d.resolver();
  await pendiente;
  assert.equal(señal.aborted, true);
  assert.equal(root.innerHTML, "");
});

test("los dos catálogos resuelven textos y nombres canónicos sin faltantes", async () => {
  const claves = (objeto, prefijo = "") => Object.entries(objeto).flatMap(([k, v]) =>
    typeof v === "string" ? [prefijo + k] : claves(v, `${prefijo}${k}.`));
  const esperadas = claves(textos.mensajes);
  for (const idioma of IDIOMAS_DISPONIBLES) {
    const traducido = await cargarTextos("administracion-modulos", { idioma: idioma.codigo });
    assert.deepEqual(traducido.faltantes, []);
    for (const clave of esperadas) assert.ok(traducido.traducir(clave));
  }
});
