import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { montarVistaAccesoPapelesDietas } from "./vista-acceso-papeles.js";
import { crearTraductorD1Dietas, MENSAJES_D1_DIETAS_ES } from "./i18n-d1.js";

class Nodo {
  constructor(documento, tagName) {
    this.ownerDocument = documento;
    this.tagName = tagName;
    this.children = [];
    this.dataset = {};
    this.attrs = {};
    this.parentNode = null;
    this.textContent = "";
  }
  append(...hijos) {
    for (const hijo of hijos) {
      hijo.parentNode = this;
      this.children.push(hijo);
    }
  }
  replaceChildren(...hijos) {
    for (const hijo of this.children) hijo.parentNode = null;
    this.children = [];
    this.append(...hijos);
  }
  removeChild(hijo) {
    this.children = this.children.filter((actual) => actual !== hijo);
    hijo.parentNode = null;
  }
  setAttribute(clave, valor) { this.attrs[clave] = String(valor); }
  querySelectorAll(selector) {
    const encontrados = [];
    const visitar = (nodo) => {
      if (nodo.tagName === selector) encontrados.push(nodo);
      nodo.children.forEach(visitar);
    };
    visitar(this);
    return encontrados;
  }
}

function crearContenedor() {
  const documento = {};
  documento.createElement = (etiqueta) => new Nodo(documento, etiqueta);
  return new Nodo(documento, "div");
}

test("D1 muestra los cinco papeles exactos, sus tareas y estado cerrado por defecto", () => {
  const contenedor = crearContenedor();
  const { desmontar } = montarVistaAccesoPapelesDietas(contenedor);
  const tarjetas = contenedor.querySelectorAll("article");
  assert.deepEqual(tarjetas.map((tarjeta) => tarjeta.dataset.dietasPapel), [
    "empleado", "administrativo_servicio", "responsable_centro", "rrhh", "intervencion",
  ]);
  assert.deepEqual(tarjetas.map((tarjeta) => tarjeta.querySelectorAll("h3")[0].textContent), [
    "Empleado", "Administrativo del servicio", "Responsable de centro", "RRHH", "Intervención",
  ]);
  assert.ok(tarjetas.every((tarjeta) => tarjeta.querySelectorAll("p")[0].textContent.length > 0));
  assert.ok(tarjetas.every((tarjeta) => tarjeta.querySelectorAll("span")[0].dataset.dietasEstadoPapel === "no_configurado"));
  assert.deepEqual(contenedor.querySelectorAll("summary").map((resumen) => resumen.textContent), ["?"]);
  assert.equal(contenedor.querySelectorAll("button").length, 0);
  assert.equal(contenedor.querySelectorAll("select").length, 0);
  assert.equal(contenedor.querySelectorAll("input").length, 0);
  desmontar();
  assert.equal(contenedor.children.length, 0);
});

test("un estado proyectado solo cambia el indicador, sin añadir operaciones ni identidad", () => {
  const contenedor = crearContenedor();
  const falsosHeredados = Object.create({ rrhh: "disponible" });
  falsosHeredados.empleado = "disponible";
  const vista = montarVistaAccesoPapelesDietas(contenedor, { estadosVerificados: falsosHeredados });
  const estados = Object.fromEntries(contenedor.querySelectorAll("article").map((tarjeta) => [
    tarjeta.dataset.dietasPapel, tarjeta.querySelectorAll("span")[0].dataset.dietasEstadoPapel,
  ]));
  assert.equal(estados.empleado, "disponible");
  assert.equal(estados.rrhh, "no_configurado");
  assert.equal(Object.values(estados).filter((estado) => estado === "disponible").length, 1);
  assert.equal(contenedor.querySelectorAll("button").length, 0);
  vista.desmontar();
  vista.desmontar();
});

test("la extensión i18n usa el catálogo común y permite traducción futura", () => {
  const t = crearTraductorD1Dietas((clave) => clave === "d1_titulo" ? "Accesos traducidos" : clave);
  assert.equal(t("d1_titulo"), "Accesos traducidos");
  assert.equal(t("d1_intervencion"), MENSAJES_D1_DIETAS_ES.d1_intervencion);
  assert.equal(crearTraductorD1Dietas()("estado_borrador"), "Borrador");
  assert.throws(() => crearTraductorD1Dietas(null), TypeError);
});

test("la vista no usa almacenamiento, credenciales ni HTML interpolado", async () => {
  const fuente = await readFile(new URL("vista-acceso-papeles.js", import.meta.url), "utf8");
  assert.doesNotMatch(fuente, /localStorage|sessionStorage|indexedDB|document\.cookie|innerHTML|fetch\(/u);
  assert.doesNotMatch(fuente, /password|contrasena|contraseña/u);
});
