import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { TRAMITES_AYUDANTE_PORTAL } from "./ayuda-contenido.js";
import { crearAyudanteTramites, MENSAJES_AYUDANTE_TRAMITES_ES } from "./ayudante-tramites.js";

test("el ayudante cubre trámites comunes de Dietas, Cronos y Personal", () => {
  const ids = new Set(TRAMITES_AYUDANTE_PORTAL.map((tramite) => tramite.id));
  [
    "dietas-crear-borrador", "dietas-consultar-borrador", "dietas-ruta",
    "cronos-corregir-marcaje", "cronos-solicitar-permiso", "cronos-consultar-saldo",
    "personal-consultar-ficha", "personal-relaciones", "personal-catalogos",
  ].forEach((id) => assert.ok(ids.has(id), `falta el trámite ${id}`));
  for (const tramite of TRAMITES_AYUDANTE_PORTAL) {
    assert.ok(["dietas", "cronos", "personal"].includes(tramite.vista));
    assert.equal(typeof tramite.selector, "string");
    assert.ok(tramite.pasos.length >= 2);
    for (const paso of tramite.pasos) {
      ["objetivo", "preparacion", "resultado", "actor", "limite"].forEach((campo) => {
        assert.equal(typeof paso[campo], "string", `${tramite.id}: falta ${campo}`);
        assert.ok(paso[campo].length > 12, `${tramite.id}: ${campo} demasiado corto`);
      });
    }
  }
});

test("la guía presenta pasos, detalle accesible y límites sin interpolar HTML", async () => {
  const ayudante = crearAyudanteTramites();
  assert.equal(ayudante.titulo, MENSAJES_AYUDANTE_TRAMITES_ES.titulo);
  assert.match(ayudante.contenido, /data-ayudante-tramite=/u);
  assert.match(ayudante.contenido, /Bolsa de trabajo y Contratación temporal/u);
  assert.doesNotMatch(ayudante.contenido, /<script/u);
  assert.match(ayudante.contenido, /no guarda datos/u);
  const paso = await renderizarPrimerPaso();
  assert.match(paso, /aria-expanded="false"/u);
  assert.match(paso, /data-ayudante-detalle/u);
  assert.match(paso, /Límite o dependencia/u);
  assert.match(paso, /queda detenido/u);
});

test("los pasos usan sus selectores concretos y solo avisan de detención cuando procede", async () => {
  const codigo = await readFile(new URL("./ayudante-tramites.js", import.meta.url), "utf8");
  assert.match(codigo, /pasoActual\.selector/u);
  assert.match(codigo, /pasoActual\.activar/u);
  assert.match(codigo, /paso\.bloqueado \? `<p class="ayudante-tramites-pendiente"/u);
  const ruta = TRAMITES_AYUDANTE_PORTAL.find(({ id }) => id === "dietas-ruta");
  const catalogos = TRAMITES_AYUDANTE_PORTAL.find(({ id }) => id === "personal-catalogos");
  assert.equal(ruta.pasos[0].selector, "[data-dietas-area-itinerario]");
  assert.equal(ruta.pasos[0].bloqueado, false);
  assert.equal(catalogos.pasos[0].selector, '[data-personal-ficha-tab="catalogos"]');
  assert.equal(catalogos.pasos[0].bloqueado, false);
});

test("acepta guías externas sin modificar el motor", () => {
  const tramite = Object.freeze({
    id: "modulo-externo", titulo: "Consulta externa", modulo: "Módulo", vista: "externo",
    selector: "#externo", resumen: "Guía registrada por otro módulo.",
    pasos: Object.freeze([Object.freeze({
      titulo: "Abrir", instruccion: "Abra la vista.", objetivo: "Consultar la vista autorizada.",
      preparacion: "Tener acceso concedido.", resultado: "La vista queda enfocada.",
      actor: "Persona autorizada.", limite: "No realiza ninguna operación.", bloqueado: false,
    })]),
  });
  const ayudante = crearAyudanteTramites({ tramites: [tramite] });
  assert.match(ayudante.contenido, /Consulta externa/u);
  assert.doesNotMatch(ayudante.contenido, /Crear un borrador de dieta/u);
});

async function renderizarPrimerPaso() {
  let pulsador;
  const contenedor = {
    innerHTML: "",
    addEventListener(_tipo, listener) { pulsador = listener; },
    removeEventListener() {},
    querySelector() { return null; },
  };
  const ayudante = crearAyudanteTramites();
  ayudante.instalar({ contenedor, documento: { querySelector() { return null; } } });
  pulsador({ target: { closest() { return { dataset: { ayudanteTramite: "dietas-crear-borrador" } }; } } });
  return contenedor.innerHTML;
}

test("el ayudante no introduce red ni almacenamiento persistente", async () => {
  const codigo = await readFile(new URL("./ayudante-tramites.js", import.meta.url), "utf8");
  const eventos = await readFile(new URL("./portal-eventos.js", import.meta.url), "utf8");
  assert.doesNotMatch(codigo, /\b(fetch|XMLHttpRequest|localStorage|sessionStorage|indexedDB)\b/u);
  assert.match(codigo, /removeEventListener/u);
  assert.match(codigo, /aria-controls/u);
  assert.match(eventos, /addEventListener\?\.\("close", limpiarContenidoDialogo\)/u);
  assert.match(eventos, /addEventListener\?\.\("cancel", limpiarContenidoDialogo\)/u);
});
