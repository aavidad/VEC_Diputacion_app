import assert from "node:assert/strict";
import test from "node:test";
import { montarFormularioCierreAdministrativo } from "./formulario-cierre-administrativo.js";

const preparacion = {
  expediente_ref: "expediente:1", seguimiento_ref: "seguimiento:1",
  version_esperada: 3, motivos: ["fin_ejercicio_sintetico"],
};
const idempotenciaSintetica = "11111111-1111-4111-8111-111111111111";
const recibo = { recibo_ref: "recibo:1", version_seguimiento: 4 };
const lecturaCierre = (cambios = {}) => ({
  expediente_ref: preparacion.expediente_ref, seguimiento_ref: preparacion.seguimiento_ref,
  version_actual: recibo.version_seguimiento, estado_actual: "cerrado_administrativamente",
  preparada_en: "2026-09-12T19:16:48.462825Z", preparacion: null, ...cambios,
});

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
    releer() { return eventos.click({ type: "click", target: this.querySelector?.("[data-ct-cierre-reintentar-lectura]") ?? {
      matches: selector => selector === "[data-ct-cierre-reintentar-lectura]",
    } }); },
  };
  return raiz;
}

function crearRaizConFoco() {
  const raiz = crearRaiz();
  const documento = { body: {}, activeElement: null };
  documento.activeElement = documento.body;
  const nodos = new Map();
  let html = "", focos = 0;
  raiz.ownerDocument = documento;
  raiz.querySelector = selector => nodos.get(selector) ?? null;
  raiz.contains = nodo => [...nodos.values()].includes(nodo);
  Object.defineProperty(raiz, "innerHTML", {
    get() { return html; },
    set(valor) {
      if (raiz.contains(documento.activeElement)) documento.activeElement = documento.body;
      for (const nodo of nodos.values()) nodo.isConnected = false;
      nodos.clear();
      html = valor;
      for (const atributo of ["reintentar-lectura", "estado", "recibo", "guardar-recuperacion"]) {
        const selector = `[data-ct-cierre-${atributo}]`;
        if (!html.includes(`data-ct-cierre-${atributo}`)) continue;
        const nodo = {
          isConnected: true,
          matches: consulta => consulta === selector,
          focus() { documento.activeElement = this; focos++; },
        };
        nodos.set(selector, nodo);
      }
    },
  });
  raiz.replaceChildren = () => { raiz.innerHTML = ""; };
  return { raiz, documento, focos: () => focos };
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
      return lecturaCierre();
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

test("GET vigente tras recibo v2 sigue pendiente y permite relectura", async () => {
  const raiz = crearRaiz();
  const preparacionV1 = { ...preparacion, version_esperada: 1 };
  const reciboV2 = { ...recibo, version_seguimiento: 2 };
  let cierres = 0, lecturas = 0;
  montarFormularioCierreAdministrativo({ raiz, preparacion: preparacionV1,
    cliente: { async cerrar() { cierres++; return reciboV2; } },
    confirmarOperacion: () => true, generarClaveIdempotencia: () => idempotenciaSintetica,
    alConfirmar: async () => { lecturas++; return lecturaCierre({ estado_actual: "vigente", version_actual: 1, preparacion: preparacionV1 }); },
  });
  await raiz.preparar(); await raiz.continuar(); await raiz.releer();
  assert.equal(cierres, 1);
  assert.equal(lecturas, 2);
  assert.match(raiz.innerHTML, /recibo:1/u);
  assert.match(raiz.innerHTML, /actualización del estado sigue pendiente/u);
  assert.match(raiz.innerHTML, /data-ct-cierre-reintentar-lectura/u);
  assert.doesNotMatch(raiz.innerHTML, /Estado actual del seguimiento: Vigente|data-ct-cierre-estado>Recibo verificado\.<\/p>/u);
});

test("GET cerrado de otro contexto o versión anterior no confirma el cierre", async () => {
  for (const respuesta of [
    lecturaCierre({ expediente_ref: "expediente:ajeno" }),
    lecturaCierre({ seguimiento_ref: "seguimiento:ajeno" }),
    lecturaCierre({ version_actual: recibo.version_seguimiento - 1 }),
  ]) {
    const raiz = crearRaiz();
    let cierres = 0;
    montarFormularioCierreAdministrativo({ raiz, preparacion,
      cliente: { async cerrar() { cierres++; return recibo; } },
      confirmarOperacion: () => true, generarClaveIdempotencia: () => idempotenciaSintetica,
      alConfirmar: async () => respuesta,
    });
    await raiz.preparar(); await raiz.continuar();
    assert.equal(cierres, 1);
    assert.match(raiz.innerHTML, /actualización del estado sigue pendiente/u);
    assert.match(raiz.innerHTML, /data-ct-cierre-reintentar-lectura/u);
    assert.doesNotMatch(raiz.innerHTML, /Estado actual del seguimiento: Cerrado/u);
  }
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
  resolver({ ...lecturaCierre(), dato_ajeno: "no confiable" });
  await pendiente;
  assert.equal(cierres, 1);
  assert.match(raiz.innerHTML, /actualización del estado sigue pendiente/u);
  assert.doesNotMatch(raiz.innerHTML, /Estado actual del seguimiento: Cerrado/u);
  assert.match(raiz.innerHTML, /data-ct-cierre-reintentar-lectura/u);
});

test("foco de relectura pasa por estado, vuelve al botón en error y termina en recibo", async () => {
  const { raiz, documento } = crearRaizConFoco();
  let cierres = 0, lecturas = 0, resolver;
  montarFormularioCierreAdministrativo({ raiz, preparacion,
    cliente: { async cerrar() { cierres++; return recibo; } },
    confirmarOperacion: () => true, generarClaveIdempotencia: () => idempotenciaSintetica,
    alConfirmar: () => ++lecturas === 1 ? false : new Promise(resolve => { resolver = resolve; }),
  });
  await raiz.preparar(); await raiz.continuar();
  raiz.querySelector("[data-ct-cierre-reintentar-lectura]").focus();
  const primera = raiz.releer();
  assert.equal(documento.activeElement, raiz.querySelector("[data-ct-cierre-estado]"));
  resolver(false); await primera;
  assert.equal(documento.activeElement, raiz.querySelector("[data-ct-cierre-reintentar-lectura]"));
  const segunda = raiz.releer();
  assert.equal(documento.activeElement, raiz.querySelector("[data-ct-cierre-estado]"));
  resolver(lecturaCierre()); await segunda;
  assert.equal(documento.activeElement, raiz.querySelector("[data-ct-cierre-recibo]"));
  assert.equal(cierres, 1);
});

test("relectura no roba foco movido ni enfoca tras desmontar", async () => {
  const { raiz, documento, focos } = crearRaizConFoco();
  let lecturas = 0, resolver;
  const desmontar = montarFormularioCierreAdministrativo({ raiz, preparacion,
    cliente: { async cerrar() { return recibo; } },
    confirmarOperacion: () => true, generarClaveIdempotencia: () => idempotenciaSintetica,
    alConfirmar: () => ++lecturas === 1 ? false : new Promise(resolve => { resolver = resolve; }),
  });
  await raiz.preparar(); await raiz.continuar();
  raiz.querySelector("[data-ct-cierre-reintentar-lectura]").focus();
  const pendiente = raiz.releer();
  const focoAjeno = {};
  documento.activeElement = focoAjeno;
  resolver(false); await pendiente;
  assert.equal(documento.activeElement, focoAjeno);
  raiz.querySelector("[data-ct-cierre-reintentar-lectura]").focus();
  const segunda = raiz.releer();
  raiz.querySelector("[data-ct-cierre-guardar-recuperacion]").focus();
  resolver(false); await segunda;
  assert.equal(documento.activeElement, raiz.querySelector("[data-ct-cierre-guardar-recuperacion]"));
  raiz.querySelector("[data-ct-cierre-reintentar-lectura]").focus();
  const posterior = raiz.releer();
  desmontar();
  const focosAntes = focos();
  resolver({ estadoActual: "cerrado" }); await posterior;
  assert.equal(focos(), focosAntes);
  assert.equal(raiz.innerHTML, "");
});
