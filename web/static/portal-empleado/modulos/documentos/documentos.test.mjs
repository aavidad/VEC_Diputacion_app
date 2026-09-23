import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { MENSAJES_DOCUMENTOS_ES, crearTraductorDocumentos } from "./i18n.js";

const directorio = new URL("./", import.meta.url);
const [vista, css] = await Promise.all([
  readFile(new URL("vista.js", directorio), "utf8"),
  readFile(new URL("documentos.css", directorio), "utf8"),
]);

test("sin fuente el repositorio monta una lista vacía y se desmonta", () => {
  assert.match(vista, /export function montarVistaDocumentos/u);
  assert.match(vista, /dataset\.estado = "no_configurado"/u);
  assert.match(vista, /registrarDesmontar\?\.\(desmontar\)/u);
  assert.match(vista, /"lista_vacia"/u);
  assert.doesNotMatch(vista, /datos-presentacion|datos-sinteticos|<tbody|createElement\("tr"\)/iu);
  assert.doesNotMatch(vista, /fetch\(|XMLHttpRequest|document\.cookie|localStorage|sessionStorage|indexedDB/iu);
});

test("borrador, firma, descarga y custodia mantienen significados distintos", () => {
  const t = crearTraductorDocumentos();
  assert.equal(t("titulo"), "Repositorio documental");
  assert.match(t("sin_fuente"), /no ha consultado documentos/u);
  assert.match(t("nota_borrador"), /no equivale a firma/u);
  assert.match(t("nota_firmado"), /firma validada/u);
  assert.match(t("nota_descarga"), /original y permiso/u);
  assert.match(t("nota_custodia"), /recibo de conservación/u);
  assert.match(t("lista_vacia"), /No se muestran documentos/u);
  assert.match(vista, /boton\.disabled = true/u);
  assert.match(vista, /firma_sin_evidencia/u);
  assert.match(vista, /huella_sin_fuente/u);
  assert.match(vista, /custodia_sin_fuente/u);
});

test("los tipos previstos solo figuran en la ayuda, sin filas de repositorio", () => {
  assert.match(vista, /"details"/u);
  assert.match(vista, /"summary"/u);
  assert.match(vista, /"tipos_previstos"/u);
  assert.match(vista, /tipo_informe", "tipo_resolucion", "tipo_rc", "tipo_diligencia/u);
  assert.match(MENSAJES_DOCUMENTOS_ES.tipos_previstos, /sin documentos asociados/u);
  assert.doesNotMatch(`${vista}\n${css}\n${JSON.stringify(MENSAJES_DOCUMENTOS_ES)}`, /\bDEMO\b|Antonio López|CT-2026-0148|recibo:[0-9a-f-]+/iu);
});

test("i18n, componentes comunes y respuesta móvil", () => {
  const claves = [...vista.matchAll(/t\("([a-z0-9_]+)"/gu)].map((resultado) => resultado[1]);
  assert.ok(claves.length > 25);
  for (const clave of claves) assert.equal(typeof MENSAJES_DOCUMENTOS_ES[clave], "string", `falta ${clave}`);
  assert.match(vista, /"cabecera-panel"/u);
  assert.match(vista, /"cuerpo-panel"/u);
  assert.match(vista, /renderizarEstadoEntrega/u);
  assert.match(css, /@media \(max-width: 1150px\)/u);
  assert.match(css, /@media \(max-width: 850px\)/u);
  assert.match(css, /@media \(max-width: 480px\)/u);
  assert.doesNotMatch(css, /#[0-9a-f]{3,8}\b/iu);
});
