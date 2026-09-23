import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { ATLAS_COMUNICACIONES, ESTADO_COMUNICACIONES } from "./datos-presentacion.js";
import { crearTraductorComunicaciones, MENSAJES_COMUNICACIONES_ES } from "./i18n.js";
import { montarVistaComunicaciones, renderizarVistaComunicaciones } from "./vista.js";

test("Comunicaciones presenta datos sintéticos minimizados y límites de conexión", () => {
  const html = renderizarVistaComunicaciones();
  assert.match(html, /Bandeja de Antonio López Fernández/u);
  assert.match(html, /Qué falta para terminarlo/u);
  assert.match(html, /No hay acuse, documento adjunto, despacho, entrega ni notificación fehaciente acreditados/u);
  assert.match(html, /disabled aria-disabled="true"/u);
  assert.match(JSON.stringify(ATLAS_COMUNICACIONES), /a•••••\.l••••@d••••\.es/u);
  assert.doesNotMatch(JSON.stringify(ATLAS_COMUNICACIONES), /(?:@dipgra\.es|@gmail\.com|\b\d{8}[A-Z]\b)/iu);
  assert.equal(ESTADO_COMUNICACIONES.estado, "visual_pendiente_backend");
});

test("filtra y cambia la superficie local sin red o almacenamiento", async () => {
  const filtrada = renderizarVistaComunicaciones({ filtro: "SMS" });
  const plantillas = renderizarVistaComunicaciones({ pestana: "plantillas" });
  assert.match(filtrada, /Solicitud de ausencia pendiente de validación/u);
  assert.doesNotMatch(filtrada, /Actualización de documentación de Dietas/u);
  assert.match(plantillas, /Plantillas/u);
  const fuente = await readFile(new URL("./vista.js", import.meta.url), "utf8");
  assert.doesNotMatch(fuente, /(?:fetch\(|localStorage|sessionStorage|document\.cookie)/u);
});

test("distingue transporte, entrega y lectura sin acreditar ninguno ni mostrar correo ficticio", () => {
  const html = renderizarVistaComunicaciones();
  for (const titulo of ["Aceptado por transporte", "Entregado al destinatario", "Leído por el destinatario"]) {
    assert.match(html, new RegExp(titulo, "u"));
  }
  assert.match(html, /La aceptación del transporte solo confirmaría la recepción por el canal técnico/u);
  assert.match(html, /Envío no disponible: falta un canal corporativo conectado/u);
  assert.match(html, /Enviar<\/button>/u);
  assert.doesNotMatch(html, /a•••••\.l••••@d••••\.es|Pendiente de lectura/u);
  assert.equal((html.match(/Sin evidencia/g) || []).length, 12);
});

test("el filtro sin coincidencias no muestra detalle ajeno y escapa la entrada", () => {
  const html = renderizarVistaComunicaciones({ filtro: "<script>alert(1)</script>" });
  assert.match(html, /No hay comunicaciones que coincidan con el filtro/u);
  assert.match(html, /&lt;script&gt;alert\(1\)&lt;\/script&gt;/u);
  assert.doesNotMatch(html, /<script>|Actualización de documentación de Dietas/u);
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

test("el catálogo español es cerrado, completo y la vista usa sus etiquetas", async () => {
  const clavesEsperadas = [
    "detalle_sobrelinea", "limite_detalle", "acusar_recepcion", "bandeja_de", "caption_bandeja",
    "preferencias_titulo", "plantillas_titulo", "administrativas_titulo", "titulo", "navegacion",
    "pestana_bandeja", "filtrar", "aplicar_filtro", "filtro_aplicado",
  ];
  for (const clave of clavesEsperadas) assert.equal(typeof MENSAJES_COMUNICACIONES_ES[clave], "string");
  const t = crearTraductorComunicaciones();
  assert.equal(t("bandeja_de", { persona: "Antonio López Fernández" }), "Bandeja de Antonio López Fernández");
  assert.throws(() => t("clave_inexistente"), /desconocida/u);
  assert.throws(() => crearTraductorComunicaciones({ ...MENSAJES_COMUNICACIONES_ES, titulo: "" }), /incompleto/u);
  const fuente = await readFile(new URL("./vista.js", import.meta.url), "utf8");
  for (const literal of ["Detalle seleccionado", "Acusar recepción", "Bandeja de", "Filtrar avisos", "Crear campaña"]) {
    assert.doesNotMatch(fuente, new RegExp(literal, "u"));
  }
});
