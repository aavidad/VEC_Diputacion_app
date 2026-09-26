import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import test from "node:test";
import {
  calcularHuellaArchivo, comprobarHuellasFormulario, instalarHuellaArchivo, LIMITE_HUELLA_ARCHIVO_BYTES,
  renderizarCampoHuellaArchivo, traducirHuellaArchivo,
} from "./portal-huella-archivo.js";

const CONTENIDO = "Resolución sintética de prueba\n";
const HUELLA = createHash("sha256").update(CONTENIDO).digest("hex");

test("calcula la huella SHA-256 de un contenido conocido y limpia los bytes", async () => {
  const resultado = await calcularHuellaArchivo(new Blob([CONTENIDO]));
  assert.deepEqual(resultado, { ok: true, huella: HUELLA });
  assert.equal(HUELLA, createHash("sha256").update(Buffer.from(CONTENIDO, "utf8")).digest("hex"));
});

test("sin archivo, vacío o sin criptografía no hay huella", async () => {
  assert.deepEqual(await calcularHuellaArchivo(null), { ok: false, motivo: "sin_archivo" });
  assert.deepEqual(await calcularHuellaArchivo(new Blob([])), { ok: false, motivo: "vacio" });
  assert.deepEqual(await calcularHuellaArchivo(new Blob(["x"]), { criptografia: {} }), { ok: false, motivo: "no_disponible" });
});

test("el límite de tamaño se aplica antes de leer el archivo", async () => {
  assert.equal(LIMITE_HUELLA_ARCHIVO_BYTES, 20 * 1024 * 1024);
  let leido = false;
  const grande = { size: LIMITE_HUELLA_ARCHIVO_BYTES + 1, arrayBuffer: async () => { leido = true; return new ArrayBuffer(0); } };
  assert.deepEqual(await calcularHuellaArchivo(grande), { ok: false, motivo: "demasiado_grande" });
  assert.equal(leido, false);
  assert.deepEqual(await calcularHuellaArchivo(new Blob(["12345"]), { limite: 4 }), { ok: false, motivo: "demasiado_grande" });
  assert.match(traducirHuellaArchivo("demasiado_grande", { limite: "20" }), /20 MB/u);
});

test("el campo es accesible, conserva una huella válida y no muestra la huella", () => {
  const html = renderizarCampoHuellaArchivo({ id: "prueba-archivo", nombre: "justificante_sha256", huella: HUELLA });
  assert.match(html, /<label class="campo" for="prueba-archivo"><span>Archivo del justificante<\/span><input id="prueba-archivo"/u);
  assert.match(html, /type="file"[^>]*aria-describedby="prueba-archivo-estado"[^>]*aria-required="true"/u);
  assert.match(html, new RegExp(`type="hidden" name="justificante_sha256" value="${HUELLA}"`, "u"));
  assert.match(html, /id="prueba-archivo-estado"[^>]*role="status"[^>]*>Documento comprobado en este equipo/u);
  assert.equal(html.split(HUELLA).length - 1, 1, "la huella solo viaja en el campo oculto");
  const invalida = renderizarCampoHuellaArchivo({ id: "p2", huella: "no-es-huella", obligatorio: false });
  assert.match(invalida, /type="hidden" name="sha256" value=""/u);
  assert.doesNotMatch(invalida, /aria-required/u);
  assert.throws(() => renderizarCampoHuellaArchivo({ id: "1 malo" }), TypeError);
});

// Dobles mínimos del DOM: un formulario con un campo de huella.
function formularioFalso({ huella = "", obligatorio = true, calculando = false, deshabilitado = false } = {}) {
  const oculto = { value: huella };
  const estado = { textContent: "" };
  const atributos = new Map(obligatorio ? [["data-huella-obligatoria", ""]] : []);
  const campo = { querySelector: (sel) => (sel === "[data-huella-valor]" ? oculto : sel === "[data-huella-estado]" ? estado : null) };
  const control = {
    disabled: deshabilitado, value: "x", files: [], enfocado: false,
    dataset: calculando ? { huellaCalculando: "true" } : {},
    hasAttribute: (n) => atributos.has(n), setAttribute: (n, v) => atributos.set(n, v), removeAttribute: (n) => atributos.delete(n),
    getAttribute: (n) => atributos.get(n) ?? null,
    closest: (sel) => (sel === "[data-huella-archivo]" ? control : sel === "[data-huella-campo]" ? campo : null),
    focus() { this.enfocado = true; },
  };
  const formulario = { querySelectorAll: () => [control] };
  return { formulario, control, oculto, estado };
}

test("sin archivo el envío se bloquea con mensaje legible y foco en el control", () => {
  const { formulario, control, estado } = formularioFalso();
  const mensaje = comprobarHuellasFormulario(formulario);
  assert.equal(mensaje, traducirHuellaArchivo("sin_archivo"));
  assert.equal(estado.textContent, mensaje);
  assert.equal(control.getAttribute("aria-invalid"), "true");
  assert.equal(control.enfocado, true);
  assert.equal(comprobarHuellasFormulario(formularioFalso({ huella: HUELLA }).formulario), "");
  assert.equal(comprobarHuellasFormulario(formularioFalso({ obligatorio: false }).formulario), "");
  assert.equal(comprobarHuellasFormulario(formularioFalso({ deshabilitado: true }).formulario), "");
  assert.equal(comprobarHuellasFormulario(formularioFalso({ huella: HUELLA, calculando: true }).formulario), traducirHuellaArchivo("esperando"));
});

test("al elegir archivo rellena el campo oculto; el envío sin archivo se detiene en captura", async () => {
  const oyentes = {};
  const raiz = { addEventListener: (tipo, f, captura) => { oyentes[tipo] = { f, captura }; }, removeEventListener: (tipo) => { delete oyentes[tipo]; } };
  const retirar = instalarHuellaArchivo(raiz);
  assert.equal(oyentes.submit.captura, true);
  assert.deepEqual(instalarHuellaArchivo(raiz)(), undefined, "una sola instalación por raíz");
  const { formulario, control, oculto, estado } = formularioFalso();
  control.files = [new Blob([CONTENIDO])];
  await oyentes.change.f({ target: control });
  assert.equal(oculto.value, HUELLA);
  assert.equal(estado.textContent, traducirHuellaArchivo("comprobado"));
  const vacio = formularioFalso();
  let detenido = false;
  let prevenido = false;
  oyentes.submit.f({ target: vacio.formulario, preventDefault: () => { prevenido = true; }, stopImmediatePropagation: () => { detenido = true; } });
  assert.equal(prevenido && detenido, true);
  control.files = [{ size: LIMITE_HUELLA_ARCHIVO_BYTES + 1, arrayBuffer: async () => new ArrayBuffer(0) }];
  await oyentes.change.f({ target: control });
  assert.equal(oculto.value, "");
  assert.match(estado.textContent, /20 MB/u);
  assert.equal(control.getAttribute("aria-invalid"), "true");
  retirar();
  assert.equal(oyentes.change, undefined);
});
