import assert from "node:assert/strict";
import test from "node:test";

import { montarModuloFiscalizacionContratacionTemporal } from "./vista-expedientes-fiscalizacion.js";

function prepararAcceso() {
  const eventos = new Map();
  const referencia = {
    name: "expediente_ref", value: "ref inválida", error: "",
    setCustomValidity(mensaje) { this.error = mensaje; },
  };
  const version = { name: "version_esperada", value: "5" };
  let informesMontados = 0;
  const formulario = {
    elements: { namedItem: (nombre) => nombre === "expediente_ref" ? referencia : version },
    closest(selector) { return selector === "[data-ct-fiscalizacion-acceso]" ? this : null; },
    checkValidity() {
      return referencia.error === "" && referencia.value !== ""
        && Number.isInteger(Number(version.value)) && Number(version.value) >= 1;
    },
    reportValidity() { this.informes = (this.informes ?? 0) + 1; },
  };
  const raiz = {
    innerHTML: "",
    addEventListener(tipo, manejador) { eventos.set(tipo, manejador); },
    removeEventListener(tipo, manejador) {
      if (eventos.get(tipo) === manejador) eventos.delete(tipo);
    },
    contains(elemento) { return elemento === formulario || elemento === referencia; },
    querySelector(selector) {
      if (selector === "[data-ct-exp-fiscalizacion]") {
        informesMontados += 1;
        return {
          innerHTML: "", addEventListener() {}, removeEventListener() {},
          querySelector() { return null; }, contains() { return false; },
          replaceChildren() {},
        };
      }
      return null;
    },
  };
  const modulo = montarModuloFiscalizacionContratacionTemporal({
    raiz,
    cliente: { registrarResultadoFiscalizacion() {} },
  });
  function enviar() {
    eventos.get("submit")({ target: formulario, preventDefault() {} });
  }
  function editarReferencia(valor) {
    referencia.value = valor;
    eventos.get("input")({ target: referencia });
  }
  return { modulo, raiz, referencia, version, formulario, enviar, editarReferencia,
    informesMontados: () => informesMontados, eventos };
}

test("la referencia corregida permite abrir fiscalización tras el rechazo", () => {
  const acceso = prepararAcceso();
  acceso.enviar();
  assert.match(acceso.referencia.error, /referencia íntegra/u);
  assert.equal(acceso.informesMontados(), 0);

  acceso.editarReferencia("expediente:ct:001");
  assert.equal(acceso.referencia.error, "");
  acceso.enviar();
  assert.equal(acceso.informesMontados(), 1);
  assert.match(acceso.raiz.innerHTML, /data-ct-exp-fiscalizacion/u);
  acceso.modulo.desmontar();
  assert.equal(acceso.eventos.size, 0);
});

test("la edición de una referencia que sigue siendo inválida no abre el formulario", () => {
  const acceso = prepararAcceso();
  acceso.enviar();
  acceso.editarReferencia("otra referencia inválida");
  acceso.enviar();
  assert.match(acceso.referencia.error, /referencia íntegra/u);
  assert.equal(acceso.informesMontados(), 0);
  assert.equal(acceso.formulario.informes, 2);
  acceso.modulo.desmontar();
});

test("corregir la referencia no omite la versión obligatoria", () => {
  const acceso = prepararAcceso();
  acceso.enviar();
  acceso.editarReferencia("expediente:ct:001");
  acceso.version.value = "0";
  acceso.enviar();
  assert.equal(acceso.informesMontados(), 0);
  acceso.modulo.desmontar();
});
