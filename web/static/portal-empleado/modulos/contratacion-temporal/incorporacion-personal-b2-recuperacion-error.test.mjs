import assert from "node:assert/strict";
import test from "node:test";
import { cargarTextos } from "../../../comun/textos.js";
import { montarIncorporacionPersonalB2 } from "./incorporacion-personal-b2.js?v=20261001-ct-a-i18n-v1";
import { ESQUEMA_CONSULTA_B2, ESQUEMA_RECIBO_B2 } from "./contrato-incorporacion-personal-b2.js";

const expediente = "expediente:b2:recuperacion";
const idempotenciaFija = "99000000-0000-4000-8000-000000000001";
const sha = "a".repeat(64);
const consulta = {
  esquema: ESQUEMA_CONSULTA_B2, expediente_ref: expediente, version_expediente_actual: 7,
  estado: "sin_plan", prerrequisitos: [{ clave_i18n: "previo.aceptacion", cumplido: true }],
  opciones: {
    vacantes: [{ plaza_ref: "plaza:1", puesto_ref: "puesto:1", version_plantilla_ref: "plantilla:1",
      version_rpt_ref: "rpt:1", unidad_ref: "unidad:1", categoria_ref: "categoria:1",
      plaza_etiqueta: "Plaza uno", puesto_etiqueta: "Puesto uno" }],
    regimenes: [{ ref: "regimen:1", version: 1, denominacion: "Personal laboral" }],
    modalidades: [{ ref: "modalidad:1", version: 1, denominacion: "Sustitución" }],
    catalogo_clases_ocupacion: { ref: "catalogo:clases:1", version: 1, huella_sha256: sha },
    clases_ocupacion: [{ valor: "temporal", texto_clave: "personal.ocupacion.clase.temporal" }],
    motivos: ["incorporar"],
    documentos: [{ documento_ref: "documento:1", documento_sha256: sha, etiqueta_clave_i18n: "documento.resolucion" }],
    periodo: { desde: "2026-10-01", hasta: "", fuente_ref: "periodo:1" },
  }, plan: null, recibo: null,
};

function crearRaiz() {
  const eventos = new Map();
  return {
    innerHTML: "", isConnected: true,
    addEventListener(tipo, fn) { eventos.set(tipo, fn); },
    removeEventListener(tipo) { eventos.delete(tipo); },
    replaceChildren() { this.innerHTML = ""; },
    revisar() {
      const valores = { desde: "2026-10-01", hasta: "", clase_ocupacion: "0" };
      eventos.get("submit")({ preventDefault() {}, target: { matches: () => true,
        elements: Object.fromEntries(Object.entries(valores).map(([clave, value]) => [clave, { value }])) } });
    },
    click(accion) { return eventos.get("click")({ target: { closest: () => ({ getAttribute: () => accion }) } }); },
  };
}

test("un GET 503 tras revisar invalida los datos y bloquea el POST hasta recuperar la consulta", async () => {
  const textos = await cargarTextos("contratacion-temporal-incorporacion-personal-b2");
  const raiz = crearRaiz();
  let lecturas = 0; const planes = []; let confirmaciones = 0;
  const cliente = {
    consultar: async () => {
      if (++lecturas === 2) throw Object.assign(new Error("fallo privado"), { estado: 503, envelopeValido: true });
      return consulta;
    },
    preparar: async (intencion) => {
      planes.push(intencion);
      return { ...consulta, estado: "plan_preparado", plan: {
        plan_ref: "plan:b2:recuperacion", version: 1, sha256: sha, intencion,
      } };
    },
    confirmar: async (solicitud) => {
      confirmaciones++;
      return { esquema: ESQUEMA_RECIBO_B2, expediente_ref: expediente, plan_ref: solicitud.plan_ref,
        plan_version: solicitud.version_plan, recibo_ref: "recibo:b2:original",
        registrada_en: "2026-10-01T10:00:00.123456Z", empleado_ref: "empleado:1",
        relacion_ref: "relacion:1", ocupacion_ref: "ocupacion:1", firma_oficial: false,
        eficacia_administrativa: false };
    },
  };
  const desmontar = montarIncorporacionPersonalB2({ raiz, cliente, expedienteRef: expediente,
    versionEsperada: 7, textos, generarClave: () => idempotenciaFija,
    resolverEtiqueta: (clave) => ({ "previo.aceptacion": "Aceptación confirmada", "motivo.incorporar": "Incorporación",
      "documento.resolucion": "Resolución" })[clave] });
  try {
    await new Promise((resolver) => setImmediate(resolver));
    raiz.revisar();
    assert.match(raiz.innerHTML, /data-b2-accion="registrar"/u);
    await raiz.click("consultar");
    assert.equal(lecturas, 2);
    assert.doesNotMatch(raiz.innerHTML, /fallo privado/u);
    assert.match(raiz.innerHTML, /data-b2-accion="consultar"/u);
    assert.doesNotMatch(raiz.innerHTML, /data-b2-accion="registrar"|data-b2-accion="retomar"/u);
    await raiz.click("registrar");
    await raiz.click("retomar");
    assert.equal(planes.length, 0);
    await raiz.click("consultar");
    assert.equal(lecturas, 3);
    assert.match(raiz.innerHTML, /data-b2-accion="registrar"/u);
    assert.doesNotMatch(raiz.innerHTML, /data-b2-accion="retomar"/u);
    await raiz.click("registrar");
    assert.equal(planes.length, 1);
    assert.equal(planes[0].clave_idempotencia, idempotenciaFija);
    assert.equal(confirmaciones, 1);
    assert.match(raiz.innerHTML, /recibo:b2:original/u);
  } finally { desmontar(); }
});
