import assert from "node:assert/strict";
import test from "node:test";
import { montarFormularioCierreAdministrativo } from "./formulario-cierre-administrativo.js";

const preparacion = {
  expediente_ref: "expediente:1", seguimiento_ref: "seguimiento:1",
  version_esperada: 3, motivos: ["fin_ejercicio_sintetico"],
};
const idempotenciaSintetica = "11111111-1111-4111-8111-111111111111";
const recibo = { recibo_ref: "recibo:1", version_seguimiento: 4 };

function crearRaiz() {
  const eventos = {};
  const raiz = {
    innerHTML: "",
    addEventListener(tipo, fn) { eventos[tipo] = fn; },
    removeEventListener(tipo) { delete eventos[tipo]; },
    replaceChildren() { this.innerHTML = ""; },
    preparar() { return eventos.submit({ type: "submit", preventDefault() {}, target: {
      matches: selector => selector === "[data-ct-cierre-administrativo-form]",
      elements: { motivo_clave: { value: "fin_ejercicio_sintetico" } },
    } }); },
    continuar() { return eventos.submit({ type: "submit", preventDefault() {}, target: {
      matches: selector => selector === "[data-ct-cierre-continuar-form]",
    } }); },
    releer() { return eventos.click({ type: "click", target: {
      matches: selector => selector === "[data-ct-cierre-reintentar-lectura]",
    } }); },
  };
  return raiz;
}

test("GET 503 después del recibo ofrece relectura y recupera estado sin repetir POST", async () => {
  const raiz = crearRaiz();
  let cierres = 0;
  const lecturas = [];
  montarFormularioCierreAdministrativo({ raiz, preparacion, estadoActual: "vigente",
    cliente: { async cerrar() { cierres++; return recibo; } },
    confirmarOperacion: () => true, generarClaveIdempotencia: () => idempotenciaSintetica,
    alConfirmar: async datos => {
      lecturas.push(datos);
      if (lecturas.length === 1) return false; // GET 503 del consumidor.
      return { estadoActual: "cerrado_administrativamente" };
    },
  });
  await raiz.preparar(); await raiz.continuar();
  assert.equal(cierres, 1);
  assert.match(raiz.innerHTML, /data-ct-cierre-reintentar-lectura/u);
  assert.match(raiz.innerHTML, /Reintentar la lectura de cierre/u);
  assert.match(raiz.innerHTML, /recibo:1/u);
  assert.match(raiz.innerHTML, new RegExp(idempotenciaSintetica, "u"));
  assert.doesNotMatch(raiz.innerHTML, /Estado actual del seguimiento: Vigente/u);
  await raiz.releer();
  assert.equal(cierres, 1);
  assert.equal(lecturas.length, 2);
  assert.deepEqual(lecturas[1], lecturas[0]);
  assert.equal(lecturas[1].recibo.recibo_ref, recibo.recibo_ref);
  assert.match(raiz.innerHTML, /Estado actual del seguimiento: Cerrado/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-cierre-reintentar-lectura/u);
  assert.match(raiz.innerHTML, /recibo:1/u);
});

test("excepción y denegación del GET conservan recibo sin filtrar error ni inventar estado", async () => {
  const raiz = crearRaiz();
  let cierres = 0, lecturas = 0;
  montarFormularioCierreAdministrativo({ raiz, preparacion,
    cliente: { async cerrar() { cierres++; return recibo; } },
    confirmarOperacion: () => true, generarClaveIdempotencia: () => idempotenciaSintetica,
    alConfirmar: async () => {
      lecturas++;
      throw Object.assign(new Error("DNI-SINTETICO-NO-MOSTRAR"), { estado: lecturas === 1 ? 503 : 403 });
    },
  });
  await raiz.preparar(); await raiz.continuar(); await raiz.releer();
  assert.equal(cierres, 1);
  assert.equal(lecturas, 2);
  assert.match(raiz.innerHTML, /recibo:1/u);
  assert.match(raiz.innerHTML, /actualización del estado sigue pendiente/u);
  assert.match(raiz.innerHTML, /data-ct-cierre-reintentar-lectura/u);
  assert.doesNotMatch(raiz.innerHTML, /DNI-SINTETICO-NO-MOSTRAR|Estado actual del seguimiento:/u);
});

test("una relectura en curso impide otra y un resultado ajeno no confirma estado", async () => {
  const raiz = crearRaiz();
  let cierres = 0, lecturas = 0, resolver;
  montarFormularioCierreAdministrativo({ raiz, preparacion,
    cliente: { async cerrar() { cierres++; return recibo; } },
    confirmarOperacion: () => true, generarClaveIdempotencia: () => idempotenciaSintetica,
    alConfirmar: () => ++lecturas === 1 ? false : new Promise(resolve => { resolver = resolve; }),
  });
  await raiz.preparar(); await raiz.continuar();
  const pendiente = raiz.releer();
  assert.match(raiz.innerHTML, /data-ct-cierre-reintentar-lectura disabled/u);
  await raiz.releer();
  assert.equal(lecturas, 2);
  resolver({ estadoActual: "cerrado", dato_ajeno: "no confiable" });
  await pendiente;
  assert.equal(cierres, 1);
  assert.match(raiz.innerHTML, /actualización del estado sigue pendiente/u);
  assert.doesNotMatch(raiz.innerHTML, /Estado actual del seguimiento: Cerrado/u);
  assert.match(raiz.innerHTML, /data-ct-cierre-reintentar-lectura/u);
});
