import test from "node:test";
import assert from "node:assert/strict";
import {
  CLAVE, seleccion, recibo, comunicacionRegistrada, declaracion, justificante,
  archivoCorreo, montar, raizPrueba,
} from "./formulario-llamamiento-pruebas.js?v=20261008-alta-rpt-circular-v5";

const CLAVE_CONTACTO = "123e4567-e89b-42d3-a456-426614174011";
const PLAZO = {
  respuesta_hasta: "2026-09-06T22:00:00Z", ultimo_dia: "2026-09-06",
  politica_ref: "vec.bolsa.reglas:1:b05.plazo_respuesta", tratamiento_fuera_de_plazo: "exige_causa_justificada",
  confirmacion_expiracion: "rrhh", criterio_respuesta_ref: "vec.bolsa.reglas:1:b07.fuera_de_plazo",
  criterio_expiracion_ref: "vec.bolsa.reglas:1:b08.sin_respuesta_baja", regla_ejemplo: true,
};

// Amplía el doble existente: reemplazar HTML crea controles nuevos y pierde
// el foco, como el DOM. Los valores se leen del renderizador real.
function raizConControles() {
  const raiz = raizPrueba(), formas = new Map(), nodos = new Map();
  const exterior = { id: "fuera-del-formulario" };
  const documento = { activeElement: exterior, getElementById: (id) => nodos.get(id) ?? null };
  let html = "", alPintar = () => {};
  const atributos = (texto) => Object.fromEntries(Array.from(
    texto.matchAll(/([\w-]+)(?:="([^"]*)")?/gu), ([, nombre, valor]) => [nombre, valor ?? ""],
  ));
  const nodo = (attrs, formulario = null, deshabilitado = false) => ({
    id: attrs.id ?? "", type: attrs.type, value: attrs.value ?? "", checked: Object.hasOwn(attrs, "checked"),
    disabled: deshabilitado || Object.hasOwn(attrs, "disabled"),
    selectionStart: attrs.type === "text" ? 0 : null, selectionEnd: attrs.type === "text" ? 0 : null,
    selectionDirection: "none", getAttribute: (nombre) => attrs[nombre] ?? null,
    closest(selector) {
      return selector === "[data-ct-llamamiento-correo]" && Object.hasOwn(attrs, "data-ct-llamamiento-correo")
        ? this : formulario;
    }, matches() { return this.disabled; },
    focus() { if (!this.disabled && attrs.type !== "hidden") documento.activeElement = this; },
    setAttribute(nombre, valor) { attrs[nombre] = valor; }, scrollIntoView() {},
    setSelectionRange(inicio, fin, direccion) {
      this.selectionStart = inicio; this.selectionEnd = fin; this.selectionDirection = direccion;
    },
  });
  Object.defineProperty(raiz, "innerHTML", {
    get: () => html,
    set(valor) {
      html = valor; formas.clear(); nodos.clear();
      if (documento.activeElement !== exterior) documento.activeElement = exterior;
      for (const [, tipo, cuerpo] of html.matchAll(/<form data-ct-llamamiento-form="([^"]+)"[^>]*>([\s\S]*?)<\/form>/gu)) {
        const formulario = { dataset: { ctLlamamientoForm: tipo }, elements: [], closest() { return this; } };
        for (const [, etiqueta, texto] of cuerpo.matchAll(/<(input|button|select)\b([^>]*)>/gu)) {
          const attrs = atributos(texto);
          const control = nodo(attrs, formulario, etiqueta !== "button" && /<fieldset disabled/u.test(cuerpo));
          control.name = attrs.name ?? "";
          formulario.elements.push(control);
          if (control.id) nodos.set(control.id, control);
        }
        formulario.elements.namedItem = (nombre) => {
          const grupo = formulario.elements.filter((control) => control.name === nombre);
          return grupo.length <= 1 ? grupo[0] : {
            get value() { return grupo.find((control) => control.checked)?.value ?? ""; },
            set value(valor) { grupo.forEach((control) => { control.checked = control.value === valor; }); },
          };
        };
        formas.set(tipo, formulario);
      }
      for (const [, texto] of html.matchAll(/<(?:h[1-6]|section|div|p)\b([^>]*)>/gu)) {
        const attrs = atributos(texto), control = nodo(attrs);
        if (attrs.id) nodos.set(attrs.id, control);
        for (const tipo of ["estado", "recibo"]) {
          if (attrs[`data-ct-llamamiento-${tipo}`]) nodos.set(`[data-ct-llamamiento-${tipo}="${attrs[`data-ct-llamamiento-${tipo}`]}"]`, control);
        }
      }
      alPintar();
    },
  });
  raiz.ownerDocument = documento;
  raiz.contains = (control) => Array.from(nodos.values()).includes(control)
    || Array.from(formas.values()).some((formulario) => formulario === control || formulario.elements.includes(control));
  raiz.querySelector = (selector) => {
    const tipo = selector.match(/^\[data-ct-llamamiento-form="([^"]+)"\]$/u)?.[1];
    if (tipo) return formas.get(tipo) ?? null;
    if (selector === "[data-ct-llamamiento-comunicacion]") return { open: true };
    return nodos.get(selector.replace(/^#/u, "")) ?? null;
  };
  raiz.preparar = (tipo, valores) => {
    const formulario = formas.get(tipo);
    assert.ok(formulario, `formulario ${tipo} visible`);
    for (const [nombre, valor] of Object.entries(valores)) {
      const control = formulario.elements.namedItem(nombre);
      if (!control) continue;
      if (typeof valor === "boolean") control.checked = valor;
      else control.value = valor;
    }
    return formulario;
  };
  raiz.archivo = (archivo, operacion = "respuesta") => {
    const control = formas.get(operacion).elements.find((control) => control.type === "file");
    control.files = [archivo];
    return raiz.eventos.get("change")({ target: control });
  };
  return { raiz, documento, exterior, alPintar(fn) { alPintar = fn; } };
}

async function escenario(t) {
  const dom = raizConControles(), efectos = [], confirmaciones = [], anuncios = [];
  let ahora = Date.parse(PLAZO.respuesta_hasta) - 100;
  const cerrar = montar(dom.raiz, {
    seleccionarLlamamiento: async () => { efectos.push("seleccion"); return recibo; },
    registrarComunicacionLlamamiento: async () => { efectos.push("comunicacion"); return comunicacionRegistrada; },
    registrarRespuestaRecibida: async (s) => { efectos.push("respuesta"); return justificante(s); },
    registrarEventoPlazoLlamamiento: async (s) => {
      efectos.push("contacto");
      return { ...s, esquema: "vec.contratacion-temporal.evento-plazo-llamamiento.v1",
        evento_ref: "evento:contacto", recibo_ref: "recibo:contacto", auditoria_ref: "auditoria:contacto",
        registrado_en: "2026-09-05T08:30:01Z", estado: "registrado", plazo: PLAZO };
    },
    resolverLlamamiento: () => assert.fail("vencer no resuelve"),
    continuarLlamamiento: () => assert.fail("vencer no continúa"),
  }, { reloj: () => ahora, confirmarOperacion: (dato) => { confirmaciones.push(dato); return true; },
    anunciar: (...args) => anuncios.push(args) });
  t.after(cerrar);
  await dom.raiz.enviar("seleccion", seleccion());
  await dom.raiz.enviar("comunicacion", { clave_idempotencia: CLAVE });
  t.mock.timers.enable({ apis: ["setTimeout"] });
  await dom.raiz.enviar("contacto", { clave_idempotencia: CLAVE_CONTACTO,
    instante_en: "2026-09-05T08:30", prueba_ref: "prueba:llamada" });
  return { ...dom, efectos, confirmar: () => confirmaciones.length, anuncios: () => anuncios.length,
    vencer() { ahora = Date.parse(PLAZO.respuesta_hasta) + 1000; t.mock.timers.tick(1100); }, cerrar };
}

for (const campo of ["correo_ref", "recibida_en", "respuesta"])
test(`el temporizador conserva la edición y el foco en ${campo} sin efectuar operaciones`, async (t) => {
  const e = await escenario(t);
  const formulario = e.raiz.preparar("respuesta", { ...declaracion(), respuesta: "renuncia", correo_ref: "correo:en-edicion" });
  const control = campo === "respuesta" ? e.documento.getElementById("ct-llamamiento-respuesta-respuesta-renuncia")
    : formulario.elements.namedItem(campo);
  control.focus();
  if (campo === "correo_ref") control.setSelectionRange(7, 12, "backward");
  const confirmaciones = e.confirmar(), anuncios = e.anuncios(), efectos = [...e.efectos];
  e.vencer();
  const nuevo = e.raiz.querySelector('[data-ct-llamamiento-form="respuesta"]');
  assert.equal(nuevo.elements.namedItem("correo_ref").value, "correo:en-edicion");
  assert.equal(nuevo.elements.namedItem("recibida_en").value, declaracion().recibida_en);
  assert.equal(nuevo.elements.namedItem("respuesta").value, "renuncia");
  assert.notEqual(e.documento.activeElement, control, "el control anterior fue reemplazado");
  assert.equal(e.documento.activeElement.id, control.id);
  if (campo === "correo_ref") {
    assert.deepEqual([e.documento.activeElement.selectionStart, e.documento.activeElement.selectionEnd,
      e.documento.activeElement.selectionDirection], [7, 12, "backward"]);
  }
  assert.match(e.raiz.innerHTML, /data-ct-llamamiento-plazo-situacion="vencido"/u);
  assert.match(e.raiz.innerHTML, /data-ct-llamamiento-propuesta-expiracion/u);
  assert.deepEqual(e.efectos, efectos); assert.equal(e.confirmar(), confirmaciones); assert.equal(e.anuncios(), anuncios);
});

test("el temporizador conserva el foco en un botón sin id", async (t) => {
  const e = await escenario(t);
  const boton = e.raiz.querySelector('[data-ct-llamamiento-form="respuesta"]').elements.find((control) => !control.name);
  boton.focus(); e.vencer();
  assert.notEqual(e.documento.activeElement, boton);
  assert.equal(e.documento.activeElement, e.raiz.querySelector('[data-ct-llamamiento-form="respuesta"]').elements.find((control) => !control.name));
});

test("la revisión pública del plazo conserva el borrador y la selección sin esperar al temporizador", async (t) => {
  const e = await escenario(t);
  const control = e.raiz.preparar("respuesta", { ...declaracion(), correo_ref: "correo:revision" }).elements.namedItem("correo_ref");
  control.focus(); control.setSelectionRange(3, 6, "forward");
  e.cerrar.revisarPlazo();
  assert.equal(e.documento.activeElement.value, "correo:revision");
  assert.deepEqual([e.documento.activeElement.selectionStart, e.documento.activeElement.selectionEnd], [3, 6]);
  assert.deepEqual(e.efectos, ["seleccion", "comunicacion", "contacto"]);
});

test("un botón distinto en la misma posición no recibe el foco anterior", async (t) => {
  const e = await escenario(t);
  e.raiz.querySelector('[data-ct-llamamiento-form="respuesta"]').elements.find((control) => !control.name).focus();
  e.alPintar(() => {
    const formulario = e.raiz.querySelector('[data-ct-llamamiento-form="respuesta"]');
    if (formulario) formulario.elements.find((control) => !control.name).type = "reset";
  });
  e.vencer();
  assert.equal(e.documento.activeElement.id, "ct-llamamiento-plazo-titulo");
});

for (const desaparece of [true, false])
test(`control ${desaparece ? "ausente" : "deshabilitado"} tras repintar dirige el foco al título del plazo`, async (t) => {
  const e = await escenario(t);
  e.raiz.querySelector('[data-ct-llamamiento-form="respuesta"]').elements.namedItem("correo_ref").focus();
  e.alPintar(() => {
    const control = e.documento.getElementById("ct-llamamiento-respuesta-correo_ref");
    if (!control) return;
    if (desaparece) e.raiz.contains = (nodo) => nodo !== control;
    else control.disabled = true;
  });
  e.vencer();
  assert.equal(e.documento.activeElement.id, "ct-llamamiento-plazo-titulo");
  assert.equal(e.documento.activeElement.getAttribute("tabindex"), "-1");
  assert.deepEqual(e.efectos, ["seleccion", "comunicacion", "contacto"]);
});

test("el temporizador no roba el foco exterior y deja intacto el recibo confirmado", async (t) => {
  const e = await escenario(t);
  await e.raiz.archivo(archivoCorreo());
  await e.raiz.enviar("respuesta", declaracion());
  assert.equal(e.documento.activeElement.getAttribute("data-ct-llamamiento-recibo"), "respuesta");
  const formulario = e.raiz.innerHTML.match(/<form data-ct-llamamiento-form="respuesta"[^>]*>[\s\S]*?<\/form>/u)[0];
  e.documento.activeElement = e.exterior;
  e.vencer();
  assert.equal(e.documento.activeElement, e.exterior);
  assert.equal(e.raiz.innerHTML.match(/<form data-ct-llamamiento-form="respuesta"[^>]*>[\s\S]*?<\/form>/u)[0], formulario);
  assert.ok(e.raiz.querySelector('[data-ct-llamamiento-recibo="respuesta"]'));
  assert.deepEqual(e.efectos, ["seleccion", "comunicacion", "contacto", "respuesta"]);
});

test("el temporizador conserva el foco del recibo y no actúa después de desmontar", async (t) => {
  const e = await escenario(t);
  assert.equal(e.documento.activeElement.getAttribute("data-ct-llamamiento-recibo"), "contacto");
  e.vencer();
  assert.equal(e.documento.activeElement.getAttribute("data-ct-llamamiento-recibo"), "contacto");
  e.cerrar(); e.vencer();
  assert.equal(e.raiz.innerHTML, "");
  assert.deepEqual(e.efectos, ["seleccion", "comunicacion", "contacto"]);
});
