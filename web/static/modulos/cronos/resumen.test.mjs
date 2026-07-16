import assert from "node:assert/strict";
import test from "node:test";

import { renderizarResumenCronos } from "./resumen.js";

class ElementoPrueba {
  constructor(etiqueta) {
    this.tagName = etiqueta.toUpperCase();
    this.children = [];
    this.className = "";
    this.textContent = "";
    this.attributes = new Map();
    this.classList = {
      add: (clase) => {
        const clases = new Set(this.className.split(/\s+/).filter(Boolean));
        clases.add(clase);
        this.className = [...clases].join(" ");
      },
      contains: (clase) => this.className.split(/\s+/).includes(clase),
    };
  }

  setAttribute(nombre, valor) {
    this.attributes.set(nombre, String(valor));
  }

  replaceChildren(...hijos) {
    this.children = hijos;
  }
}

function documentoPrueba() {
  const destino = new ElementoPrueba("section");
  return {
    destino,
    querySelector: (selector) => selector === "#cronos-panel" ? destino : null,
    createElement: (etiqueta) => new ElementoPrueba(etiqueta),
  };
}

test("renderiza el resumen Cronos sin depender del estado global", () => {
  const documento = documentoPrueba();
  renderizarResumenCronos({
    workspace: {
      cronos_sections: ["Mi jornada", "Permisos"],
      cronos_daily_summary: {
        theoretical: "7:30", worked: "7:45", telework: "No", period_balance: "-0:20",
      },
      cronos_permission_balances: [
        { name: "Asuntos propios", request: true, max: "6", requested: "2", remaining: "4" },
      ],
    },
  }, documento);

  assert.equal(documento.destino.children.length, 3);
  const [navegacion, resumen, tabla] = documento.destino.children;
  assert.equal(navegacion.tagName, "UL");
  assert.equal(navegacion.attributes.get("aria-label"), "Secciones de Cronos");
  assert.deepEqual(navegacion.children.map((nodo) => nodo.textContent), ["Mi jornada", "Permisos"]);
  assert.deepEqual(resumen.children.map((nodo) => nodo.children[1].textContent), ["7:30", "7:45", "No", "-0:20"]);
  assert.equal(resumen.children[3].children[1].classList.contains("negative"), true);
  const tablaReal = tabla.children[0];
  assert.equal(tablaReal.attributes.get("aria-label"), "Saldos de permisos disponibles");
  assert.deepEqual(
    tablaReal.children[1].children[0].children.map((celda) => celda.textContent),
    ["Asuntos propios", "Sí", "6", "2", "4"],
  );
});

test("parte de estados vacios seguros y limita la tabla", () => {
  const documento = documentoPrueba();
  const permisos = Array.from({ length: 12 }, (_, indice) => ({ name: `Permiso ${indice + 1}` }));
  renderizarResumenCronos({ workspace: { cronos_permission_balances: permisos } }, documento);

  const resumen = documento.destino.children[1];
  const tabla = documento.destino.children[2].children[0];
  assert.deepEqual(resumen.children.map((nodo) => nodo.children[1].textContent), ["-", "-", "-", "-"]);
  assert.deepEqual(documento.destino.children[0].children.map((nodo) => nodo.textContent), ["Sin secciones disponibles"]);
  assert.equal(tabla.children[1].children.length, 8);
  assert.equal(tabla.children[1].children[0].children[1].textContent, "No");
  assert.equal(tabla.children[1].children[0].children[2].textContent, "-");
});

test("muestra un estado vacio explicito para permisos", () => {
  const documento = documentoPrueba();
  renderizarResumenCronos({ workspace: {} }, documento);

  const tabla = documento.destino.children[2].children[0];
  const celdaVacia = tabla.children[1].children[0].children[0];
  assert.equal(celdaVacia.textContent, "Sin saldos de permisos disponibles.");
  assert.equal(celdaVacia.attributes.get("colspan"), "5");
});
