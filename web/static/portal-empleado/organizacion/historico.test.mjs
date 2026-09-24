import assert from "node:assert/strict";
import test from "node:test";
import { readFileSync } from "node:fs";
import { API_ORGANIZACION_HISTORICA, RUTAS_IMPORTACION, crearClienteHistorico, crearClienteImportacion,
  filtrosHistoricos, validarPaginaHistorica, validarPaqueteImportacion, validarDecisionesImportacion,
  renderizarResumenImportacion, formatearRecuentoImportacion, formatearFechaReciboImportacion,
  formatearConocidoEn, instanteDesdeHoraMadrid } from "./historico.js";
import { MENSAJES_PERSONAL_ES, crearTraductorPersonal } from "../modulos/personal/i18n.js";

const respuesta = () => ({
  data: {
    pagina: {
      selector: { organismo_ref: "org_123", vigente_en: "2024-12-31", conocido_en: "2026-09-25T10:30:00.000000Z", limite: 100 },
      version_rpt_ref: "", version_plantilla_ref: "",
      cobertura: { unidades: "parcial", puestos_tipo: "sin_datos", dotaciones: "sin_datos", plazas: "sin_datos", puestos_individuales: "sin_datos", vinculos: "sin_datos" },
      unidades: [{ traza: { id: "unidad_1", version: 1 }, etiqueta: "Centro Uno", clave_catalogo: "centro_1" }],
      puestos_tipo: [], dotaciones: [], plazas: [], puestos_individuales: [], vinculos: [],
      cursor_siguiente: "",
    },
    evidencia: { recibo_ref: "recibo:uno", decision_ref: "decision:uno", efecto_ref: "efecto:uno", auditoria_ref: "auditoria:uno", consumo_huella_sha256: "a".repeat(64), consultada_en: "2026-09-25T10:30:01Z" },
  },
});

test("B3 conserva vacío y cobertura por fuente sin deducir vacantes", () => {
  const resultado = validarPaginaHistorica(respuesta());
  assert.equal(resultado.pagina.cobertura.plazas, "sin_datos");
  assert.deepEqual(resultado.pagina.plazas, []);
  assert.equal(resultado.pagina.unidades[0].etiqueta, "Centro Uno");
  const sinCobertura = respuesta();
  delete sinCobertura.data.pagina.cobertura.plazas;
  assert.throws(() => validarPaginaHistorica(sinCobertura), /no válida/);
  const demasiadas = respuesta();
  demasiadas.data.pagina.unidades = Array.from({ length: 101 }, (_, i) => ({ traza: { id: `unidad_${i}`, version: 1 } }));
  assert.throws(() => validarPaginaHistorica(demasiadas), /no válidas/);
});

test("B3 envía GET autorizado sin organismo ni almacenamiento y conserva los dos cortes", async () => {
  let llamada;
  const cliente = crearClienteHistorico(async (url, options) => {
    llamada = { url, options };
    return { ok: true, json: async () => respuesta() };
  });
  const filtros = filtrosHistoricos({
    vigente_en: "31/12/2024", conocido_en: "25/09/2026 12:30",
    unidad_clave: "centro_1", version_rpt_ref: "rpt_2024", version_plantilla_ref: "",
  });
  // 12:30 en Madrid (CEST, UTC+2) son las 10:30 UTC.
  assert.equal(filtros.conocido_en, "2026-09-25T10:30:00.000000Z");
  await cliente.consultar(filtros);
  const url = new URL(llamada.url, "https://vec.local");
  assert.equal(url.pathname, API_ORGANIZACION_HISTORICA);
  assert.equal(url.searchParams.get("vigente_en"), "2024-12-31");
  assert.match(url.searchParams.get("conocido_en"), /^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d\.\d{6}Z$/);
  assert.equal(url.searchParams.get("unidad_clave"), "centro_1");
  assert.equal(url.searchParams.get("version_rpt_ref"), "rpt_2024");
  assert.equal(url.searchParams.has("organismo_ref"), false);
  assert.equal(llamada.options.credentials, "omit");
  assert.equal(llamada.options.cache, "no-store");
  assert.equal(llamada.options.redirect, "error");
  await assert.rejects(cliente.consultar({ ...filtros, organismo_ref: "inventado" }), /no permitido/);
});

test("B3 rechaza fechas imposibles, denegación y catálogo i18n incompleto", async () => {
  assert.throws(() => filtrosHistoricos({ vigente_en: "31/02/2024", conocido_en: "25/09/2026 12:30" }), /no válida/);
  const cliente = crearClienteHistorico(async () => ({ ok: false, status: 403 }));
  await assert.rejects(cliente.consultar({ vigente_en: "2024-12-31", conocido_en: "2026-09-25T10:30:00.000000Z", limite: 100 }), (e) => e.status === 403);
  const t = crearTraductorPersonal();
  assert.match(t("organizacion_historyNoSource"), /Sin datos/);
  assert.match(t("organizacion_historyDenied"), /permiso/);
  assert.ok(Object.hasOwn(MENSAJES_PERSONAL_ES, "organizacion_historyReceipt"));
});

test("la ayuda extensa queda detrás de ? y todas las etiquetas usan i18n común", () => {
  const html = readFileSync(new URL("./index.html", import.meta.url), "utf8");
  const ayuda = html.match(/<details class="org-help">([\s\S]*?)<\/details>/)?.[1];
  assert.ok(ayuda);
  for (const clave of ["intro", "notice", "note", "editorHelper"]) {
    assert.match(ayuda, new RegExp(`data-i18n="${clave}"`));
    assert.equal((html.match(new RegExp(`data-i18n="${clave}"`, "g")) ?? []).length, 1);
  }
  for (const [, clave] of html.matchAll(/data-i18n(?:-label|-placeholder)?="([A-Za-z]+)"/g)) {
    assert.equal(typeof MENSAJES_PERSONAL_ES[`organizacion_${clave}`], "string", clave);
  }
});

const importacion = () => ({
  manifiesto: {
    tipo: "rpt", version_ref: "11111111-1111-4111-8111-111111111111", version_revision: 1,
    fuente_ref: "fuente_rpt", fuente_version: "2026", fuente_huella_sha256: "a".repeat(64),
    catalogo_unidades: { id: "unidades", version: 1, revision: 2, huella_sha256: "b".repeat(64) },
    catalogo_clasificaciones: { id: "clases", version: 1, revision: 1, huella_sha256: "c".repeat(64) },
  },
  hechos: [{ clase: "nodo", hecho_ref: "22222222-2222-4222-8222-222222222222", revision: 1,
    fila_fuente_ref: "fila_1", unidad_ref: "centro-520", vigente_desde: "2026-09-25",
    catalogo_entrada_clave: "centro-520", denominacion: "Centro", tipo_unidad: "centro" }],
});

test("B3 prepara solo manifiesto y hechos tipados, sin aceptar ámbito o actor del navegador", () => {
  const paquete = validarPaqueteImportacion(importacion());
  assert.equal(paquete.hechos[0].clase, "nodo");
  assert.notEqual(paquete, importacion());
  const conOrganismo = importacion();
  conOrganismo.manifiesto.organismo_ref = "org_inventada";
  assert.throws(() => validarPaqueteImportacion(conOrganismo), /no válido/);
  const conActor = importacion();
  conActor.hechos[0].actor = "actor_inventado";
  assert.throws(() => validarPaqueteImportacion(conActor), /no válido/);
  const duplicado = importacion();
  duplicado.hechos.push(structuredClone(duplicado.hechos[0]));
  assert.throws(() => validarPaqueteImportacion(duplicado), /repetido/);
});

test("el resumen de importación escapa fuente hostil y no muestra denominaciones del archivo", () => {
  const paquete = importacion();
  paquete.manifiesto.fuente_ref = "<img onerror=alert(1)>";
  paquete.hechos[0].denominacion = "<img onerror=alert(2)>";
  const validado = validarPaqueteImportacion(paquete);
  const html = renderizarResumenImportacion(validado.manifiesto, "d".repeat(64), validado.hechos);
  assert.doesNotMatch(html, /<img/);
  assert.match(html, /&lt;img onerror=alert\(1\)&gt;/);
  assert.doesNotMatch(html, /alert\(2\)/);
});

test("los hechos, decisiones y seis clases usan singular, plural y número es-ES", () => {
  const pares = [
    ["importFacts", "hecho tipado", "hechos tipados"],
    ["importDecisions", "decisión", "decisiones"],
    ["importClassNode", "unidad", "unidades"],
    ["importClassType", "puesto tipo", "puestos tipo"],
    ["importClassAllocation", "dotación", "dotaciones"],
    ["importClassPlaza", "plaza", "plazas"],
    ["importClassPost", "puesto individual", "puestos individuales"],
    ["importClassLink", "vínculo", "vínculos"],
  ];
  for (const [clave, singular, plural] of pares) {
    assert.equal(formatearRecuentoImportacion(clave, 1), `1 ${singular}`);
    assert.equal(formatearRecuentoImportacion(clave, 0), `0 ${plural}`);
    assert.equal(formatearRecuentoImportacion(clave, 2), `2 ${plural}`);
  }
  assert.equal(formatearRecuentoImportacion("importFacts", 1000), "1.000 hechos tipados");
  assert.throws(() => formatearRecuentoImportacion("importFacts", -1), /no válido/);
});

test("el recibo convierte UTC a hora Europe/Madrid antes de rotularlo", () => {
  assert.match(formatearFechaReciboImportacion("2026-09-25T10:30:00Z"), /12:30:00/);
  assert.match(formatearFechaReciboImportacion("2026-01-25T10:30:00Z"), /11:30:00/);
  assert.match(crearTraductorPersonal()("organizacion_importReceiptDate", { fecha: "12:30" }), /Europe\/Madrid/);
});

test("B3 exige decisiones con evidencia y destino solo cuando hay vínculo", () => {
  const base = { decisiones: [{ clase: "unidad", fila_fuente_ref: "fila_1", resultado: "vinculada",
    destino_ref: "centro-520", motivo: "Cotejo de fuente", evidencia_ref: "acta_2026" }] };
  assert.equal(validarDecisionesImportacion(base).length, 1);
  const sinDestino = structuredClone(base);
  delete sinDestino.decisiones[0].destino_ref;
  assert.throws(() => validarDecisionesImportacion(sinDestino), /no válida/);
  const sinEvidencia = structuredClone(base);
  sinEvidencia.decisiones[0].evidencia_ref = "";
  assert.throws(() => validarDecisionesImportacion(sinEvidencia), /no válida/);
});

test("B3 reintenta POST incierto con los mismos bytes y clave; valida recibo y 409", async () => {
  const identificadorOperacion = "12345678-1234-4123-8123-123456789abc";
  const cuerpo = JSON.stringify({ revision_esperada: 0, ...importacion() });
  const operacion = { fase: "preparar", clave: identificadorOperacion, cuerpo };
  const llamadas = [];
  let intento = 0;
  const cliente = crearClienteImportacion(async (url, options) => {
    llamadas.push({ url, body: options.body, key: options.headers["Idempotency-Key"], contentType: options.headers["Content-Type"], credentials: options.credentials });
    if (++intento === 1) throw new Error("red incierta");
    return { ok: true, status: 201, json: async () => ({ data: {
      recibo_ref: "recibo:preparar", lote_ref: "lote:uno", fase: "preparar", estado: "preparacion_no_autoritativa",
      revision_anterior: 0, revision_nueva: 1, clave_idempotencia: identificadorOperacion,
      material_huella_sha256: "d".repeat(64), fuente_huella_sha256: "a".repeat(64), actor_ref: "actor:uno",
      decision_ref: "decision:uno", auditoria_ref: "auditoria:uno", registrado_en: "2026-09-25T10:30:00Z", replay: false,
    } }) };
  });
  await assert.rejects(cliente.enviar(operacion), (e) => e.incierto === true);
  const recibo = await cliente.enviar(operacion);
  assert.equal(recibo.revision_nueva, 1);
  assert.deepEqual(llamadas.map((l) => [l.url, l.body, l.key, l.contentType]), [
    [RUTAS_IMPORTACION.preparar, cuerpo, identificadorOperacion, "application/json"],
    [RUTAS_IMPORTACION.preparar, cuerpo, identificadorOperacion, "application/json"],
  ]);
  assert.ok(llamadas.every((l) => l.credentials === "omit"));
  await assert.rejects(crearClienteImportacion(async () => ({ ok: false, status: 409 })).enviar(operacion), (e) => e.conflicto === true);
});

test("B3 mantiene publicar inhabilitado sin acreditación observable", () => {
  const html = readFileSync(new URL("./index.html", import.meta.url), "utf8");
  assert.match(html, /id="import-publish" disabled aria-describedby="import-publish-reason"/);
  assert.match(html, /id="import-publish-reason"[^>]+data-i18n="importPublishBlocked"/);
});

test("los textos explicativos de publicación y alcance viven en la ayuda tras ?", () => {
  const html = readFileSync(new URL("./index.html", import.meta.url), "utf8");
  const ayuda = html.match(/<details class="org-help">([\s\S]*?)<\/details>/)?.[1];
  for (const clave of ["importPublishBlocked", "historyPageScope", "importHelp"]) {
    assert.match(ayuda, new RegExp(`data-i18n="${clave}"`), clave);
    assert.equal((html.match(new RegExp(`data-i18n="${clave}"`, "g")) ?? []).length, 1, clave);
  }
});

test("«conocido en» se escribe y se muestra en hora de Madrid, no en UTC", () => {
  // Invierno (CET, UTC+1) y verano (CEST, UTC+2).
  assert.equal(filtrosHistoricos({ vigente_en: "15/01/2026", conocido_en: "15/01/2026 09:05" }).conocido_en,
    "2026-01-15T08:05:00.000000Z");
  assert.equal(filtrosHistoricos({ vigente_en: "15/07/2026", conocido_en: "15/07/2026 00:30" }).conocido_en,
    "2026-07-14T22:30:00.000000Z");
  // Hora inexistente del cambio de marzo y primera aparición de la repetida de octubre.
  assert.equal(instanteDesdeHoraMadrid("2026-03-29T02:30"), null);
  assert.throws(() => filtrosHistoricos({ vigente_en: "29/03/2026", conocido_en: "29/03/2026 02:30" }), /no válido/);
  assert.equal(new Date(instanteDesdeHoraMadrid("2026-10-25T02:30")).toISOString(), "2026-10-25T00:30:00.000Z");
  assert.equal(instanteDesdeHoraMadrid("2026-02-30T10:00"), null);
  assert.match(formatearConocidoEn("2026-09-25T10:30:00.000000Z"), /12:30/);
  assert.match(formatearConocidoEn("2026-01-15T08:05:00.000000Z"), /9:05/);
  const t = crearTraductorPersonal();
  assert.doesNotMatch(t("organizacion_historyKnownAt"), /UTC/);
  assert.match(t("organizacion_historyKnown", { fecha: "x" }), /Europe\/Madrid/);
});
