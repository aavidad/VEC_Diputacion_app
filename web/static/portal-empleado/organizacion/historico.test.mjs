import assert from "node:assert/strict";
import test from "node:test";
import { readFileSync } from "node:fs";
import { API_ORGANIZACION_HISTORICA, crearClienteHistorico, filtrosHistoricos, validarPaginaHistorica } from "./historico.js";
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
  assert.equal(filtros.conocido_en, "2026-09-25T12:30:00.000000Z");
  await cliente.consultar(filtros);
  const url = new URL(llamada.url, "https://vec.local");
  assert.equal(url.pathname, API_ORGANIZACION_HISTORICA);
  assert.equal(url.searchParams.get("vigente_en"), "2024-12-31");
  assert.match(url.searchParams.get("conocido_en"), /^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d\.\d{6}Z$/);
  assert.equal(url.searchParams.get("unidad_clave"), "centro_1");
  assert.equal(url.searchParams.get("version_rpt_ref"), "rpt_2024");
  assert.equal(url.searchParams.has("organismo_ref"), false);
  assert.equal(llamada.options.credentials, "same-origin");
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
