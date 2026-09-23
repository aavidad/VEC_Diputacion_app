import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { crearTraductorComunicaciones, MENSAJES_COMUNICACIONES_ES } from "./i18n.js";
import { montarVistaComunicaciones, renderizarVistaComunicaciones } from "./vista.js";

const MUESTRAS_ANTERIORES = /Antonio López Fernández|COM-2026-|DIE-2026-|CRO-2026-|a•••••\.l••••@|Borrador de resolución disponible|preparaci[oó]n sint[eé]tica|informaci[oó]n sint[eé]tica/iu;

test("la bandeja arranca no configurada, vacía y sin datos de muestra", () => {
  const html = renderizarVistaComunicaciones();
  assert.match(html, /No configurado · ver límites y dependencias/u);
  assert.match(html, /0 visibles/u);
  assert.match(html, /No hay comunicaciones para mostrar: la fuente todavía no está conectada/u);
  assert.doesNotMatch(html, MUESTRAS_ANTERIORES);
  assert.doesNotMatch(html, /data-comunicaciones-seleccionar|<input/u);
});

test("transporte, entrega y lectura tienen columnas y estados separados sin recibo atribuido", () => {
  const html = renderizarVistaComunicaciones();
  for (const titulo of ["Aceptado por transporte", "Entregado al destinatario", "Leído por el destinatario"]) {
    assert.equal((html.match(new RegExp(titulo, "gu")) || []).length, 2);
  }
  assert.equal((html.match(/Sin fuente/g) || []).length, 3);
  assert.match(html, /La aceptación del transporte solo confirmaría la recepción por el canal técnico/u);
  assert.match(html, /<button[^>]*disabled aria-disabled="true"[^>]*>Enviar<\/button>/u);
  assert.match(html, /Envío no disponible: faltan un canal corporativo conectado/u);
});

test("las demás secciones tampoco muestran preferencias, plantillas ni campañas de muestra", async () => {
  for (const pestana of ["preferencias", "plantillas", "administrativas"]) {
    const html = renderizarVistaComunicaciones({ pestana });
    assert.match(html, /No configurado/u);
    assert.doesNotMatch(html, MUESTRAS_ANTERIORES);
    assert.match(html, /disabled aria-disabled="true"/u);
  }
  const fuente = await readFile(new URL("./vista.js", import.meta.url), "utf8");
  assert.doesNotMatch(fuente, /(?:fetch\(|localStorage|sessionStorage|document\.cookie|datos-presentacion)/u);
});

test("las pestañas responden al teclado, conservan foco y desmontan listeners", () => {
  const eventos = new Map(); const focos = []; const anuncios = [];
  const raiz = {
    innerHTML: "",
    addEventListener(tipo, fn) { eventos.set(tipo, fn); },
    removeEventListener(tipo, fn) { assert.equal(eventos.get(tipo), fn); eventos.delete(tipo); },
    replaceChildren() { this.innerHTML = ""; },
    querySelector(selector) { return { focus() { focos.push(selector); } }; },
  };
  const vista = montarVistaComunicaciones({ raiz, anunciar: (texto) => anuncios.push(texto) });
  let prevenido = false;
  eventos.get("keydown")({
    key: "ArrowRight", preventDefault() { prevenido = true; },
    target: { closest() { return { dataset: { comunicacionesPestana: "bandeja" } }; } },
  });
  assert.equal(prevenido, true);
  assert.match(raiz.innerHTML, /data-comunicaciones-pestana="preferencias" aria-selected="true"/u);
  assert.deepEqual(focos, ['[data-comunicaciones-pestana="preferencias"]']);
  assert.match(anuncios[0], /Preferencias/u);
  vista.desmontar();
  assert.equal(eventos.size, 0);
  assert.equal(raiz.innerHTML, "");
});

test("el catálogo español es cerrado y la vista obtiene de él sus etiquetas", async () => {
  const t = crearTraductorComunicaciones();
  assert.equal(t("visibles", { numero: 0 }), "0 visibles");
  assert.throws(() => t("clave_inexistente"), /desconocida/u);
  assert.throws(() => crearTraductorComunicaciones({ ...MENSAJES_COMUNICACIONES_ES, titulo: "" }), /incompleto/u);
  const fuente = await readFile(new URL("./vista.js", import.meta.url), "utf8");
  for (const literal of ["No configurado", "Enviar", "Aceptado por transporte", "Crear campaña"]) {
    assert.doesNotMatch(fuente, new RegExp(literal, "u"));
  }
});
