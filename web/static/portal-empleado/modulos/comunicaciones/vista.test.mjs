import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { ATLAS_COMUNICACIONES, ESTADO_COMUNICACIONES } from "./datos-presentacion.js";
import { crearTraductorComunicaciones, MENSAJES_COMUNICACIONES_ES } from "./i18n.js";
import { renderizarVistaComunicaciones } from "./vista.js";

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
