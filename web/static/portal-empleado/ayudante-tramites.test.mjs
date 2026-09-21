import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { TRAMITES_AYUDANTE_PORTAL } from "./ayuda-contenido.js";
import { crearAyudanteTramites, MENSAJES_AYUDANTE_TRAMITES_ES } from "./ayudante-tramites.js";

test("el ayudante cubre trámites comunes de Dietas, Cronos y Personal", () => {
  const ids = new Set(TRAMITES_AYUDANTE_PORTAL.map((tramite) => tramite.id));
  const idsComunes = [
    "dietas-crear-borrador", "dietas-consultar-borrador", "dietas-ruta",
    "cronos-corregir-marcaje", "cronos-solicitar-permiso", "cronos-consultar-saldo",
    "personal-consultar-ficha", "personal-relaciones", "personal-catalogos",
  ];
  idsComunes.forEach((id) => assert.ok(ids.has(id), `falta el trámite ${id}`));
  for (const id of idsComunes) {
    const tramite = TRAMITES_AYUDANTE_PORTAL.find((candidato) => candidato.id === id);
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

test("las guías de Bolsa recorren B12 y B5 sin automatizar decisiones", () => {
  const consulta = TRAMITES_AYUDANTE_PORTAL.find(({ id }) => id === "bolsa-consultar-candidatos");
  assert.ok(consulta);
  assert.equal(consulta.vista, "resumen");
  assert.deepEqual(consulta.pasos.map(({ vista, selector, bloqueado }) => ({ vista, selector, bloqueado })), [
    { vista: "resumen", selector: "#titulo-cuadro-b12", bloqueado: false },
    { vista: "resumen", selector: '[data-accion="ver-bolsa"][data-bolsa-ref]', bloqueado: false },
    { vista: "bolsa-candidatos", selector: '[data-bolsa-form="filtros"]', bloqueado: false },
    { vista: "bolsa-candidatos", selector: '[data-bolsa-accion="abrir-ficha"]', bloqueado: false },
  ]);
  assert.equal(consulta.pasos.some((paso) => "activar" in paso), false);
  assert.match(consulta.pasos[1].instruccion, /manualmente/u);
});

test("la guía de llamamiento conserva el límite C23 y la política pendiente de RRHH", () => {
  const llamamiento = TRAMITES_AYUDANTE_PORTAL.find(({ id }) => id === "bolsa-gestionar-llamamiento");
  assert.ok(llamamiento);
  const ultimo = llamamiento.pasos.at(-1);
  assert.equal(ultimo.vista, "bolsa-candidatos");
  assert.equal(ultimo.selector, "[data-bolsa-c23-pendiente]");
  assert.equal(ultimo.bloqueado, true);
  assert.match(ultimo.limite, /C23/u);
  assert.match(ultimo.limite, /contacto, la apertura ni el resultado/u);
  assert.match(ultimo.limite, /política de RRHH/u);
  assert.equal(llamamiento.pasos.some((paso) => "activar" in paso), false);
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

test("un paso puede navegar a otra vista válida antes de enfocar", () => {
  const tramite = {
    id: "guia-compuesta", titulo: "Guía compuesta", modulo: "Módulo", vista: "inicio",
    selector: "#inicio", resumen: "Dos vistas de un mismo recorrido.",
    pasos: [
      { titulo: "Primer paso", instruccion: "Abra la primera vista.", objetivo: "Consultar.", preparacion: "Acceso.", resultado: "Vista abierta.", actor: "Persona.", limite: "Sin efectos.", vista: "primera", selector: "#primera" },
      { titulo: "Segundo paso", instruccion: "Abra la segunda vista.", objetivo: "Consultar.", preparacion: "Acceso.", resultado: "Vista abierta.", actor: "Persona.", limite: "Sin efectos.", vista: "segunda", selector: "#segunda" },
    ],
  };
  let pulsador;
  const navegaciones = [];
  const contenedor = { innerHTML: "", addEventListener(_tipo, fn) { pulsador = fn; }, removeEventListener() {}, querySelector() { return null; } };
  const ayudante = crearAyudanteTramites({ tramites: [tramite] });
  ayudante.instalar({ contenedor, documento: { querySelector() { return null; } }, navegar: (vista) => navegaciones.push(vista) });
  const boton = (dataset = {}, atributos = []) => ({ dataset, disabled: false, hasAttribute: (nombre) => atributos.includes(nombre) });
  pulsador({ target: { closest: () => boton({ ayudanteTramite: "guia-compuesta" }) } });
  pulsador({ target: { closest: () => boton({}, ["data-ayudante-ir"]) } });
  pulsador({ target: { closest: () => boton({}, ["data-ayudante-siguiente"]) } });
  pulsador({ target: { closest: () => boton({}, ["data-ayudante-ir"]) } });
  assert.deepEqual(navegaciones, ["primera", "segunda"]);
});

test("las vistas inválidas del trámite o del paso se rechazan antes de montar", () => {
  const base = { id: "seguro", titulo: "Seguro", modulo: "Módulo", vista: "inicio", resumen: "Resumen", pasos: [{ titulo: "Paso", instruccion: "Instrucción", objetivo: "Objetivo", preparacion: "Preparación", resultado: "Resultado", actor: "Actor", limite: "Límite" }] };
  assert.throws(() => crearAyudanteTramites({ tramites: [{ ...base, vista: "#invalida" }] }), /vista del trámite/u);
  assert.throws(() => crearAyudanteTramites({ tramites: [{ ...base, pasos: [{ ...base.pasos[0], vista: "no válida" }] }] }), /vista del paso/u);
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
