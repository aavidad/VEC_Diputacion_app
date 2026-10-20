import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";

import { presentarEtiquetasHitoRRHH } from "./adaptador-http-expedientes.js?v=VEC_WEB_VERSION";
import { montarFormularioInformeJuridico, renderizarContenidoInforme } from "./formulario-informe-juridico.js?v=VEC_WEB_VERSION";
import { renderizarModuloContratacionTemporal } from "./vista-expedientes.js?v=VEC_WEB_VERSION";

const EXPEDIENTE = "expediente:ct:sintetico:informe-001";
const CLAVE = "123e4567-e89b-42d3-a456-426614174000";

test("presenta las referencias del paquete jurídico sin exponer JSON ni ejecutar texto", async () => {
  for (const idioma of ["es", "en"]) {
    const catalogo = JSON.parse(await readFile(new URL(`../../../textos/${idioma}/contratacion-temporal-informe-juridico.json`, import.meta.url), "utf8"));
    const t = (clave, variables = {}) => Object.entries(variables).reduce(
      (valor, [nombre, dato]) => valor.replaceAll(`{${nombre}}`, dato), catalogo.general[clave]);
    const paquete = {
      esquema: "vec.dipgra.contratacion-temporal.informe-juridico.paquete-datos",
      version_esquema: 1,
      version_expediente: 4,
      plantilla: { plantilla_ref: "plantilla:<script>", version: 3, huella_sha256: "a".repeat(64) },
      referencias_normativas: [{ norma_ref: "norma:<img src=x onerror=alert(1)>",
        version: 2, huella_sha256: "b".repeat(64) }],
      anexos: [{ documento_ref: "documento:ct:anexo-1", version_documento: 1,
        huella_sha256: "c".repeat(64) }],
    };
    const contenido = `DOCUMENTO DE DESARROLLO — SIN FIRMA NI VALIDEZ JURIDICA\n\nDatos juridicos canónicos:\n${JSON.stringify(paquete)}\n`;
    const html = renderizarContenidoInforme(contenido, t);
    const visible = html.split('<details class="ct-informe-original">')[0];
    assert.match(visible, /ct-informe-secciones/u);
    assert.match(visible, new RegExp(catalogo.general.normas, "u"));
    assert.match(visible, /&lt;img src=x onerror=alert\(1\)&gt;/u);
    assert.doesNotMatch(visible, /<img|<script|"referencias_normativas"/u);
    assert.match(html, /ct-informe-original/u);
  }
});

test("un documento histórico sin paquete válido sigue disponible sin ensanchar la vista", async () => {
  const catalogo = JSON.parse(await readFile(new URL("../../../textos/es/contratacion-temporal-informe-juridico.json", import.meta.url), "utf8"));
  const contenido = "DOCUMENTO DE DESARROLLO — SIN FIRMA NI VALIDEZ JURIDICA\n" + "x".repeat(9000);
  const html = renderizarContenidoInforme(contenido, (clave) => catalogo.general[clave]);
  assert.match(html, /No se puede mostrar este documento por apartados/u);
  assert.match(html, /<details class="ct-informe-original">/u);
  assert.match(html, /x{9000}/u);
  assert.doesNotMatch(html.split("<details")[0], /x{100}/u);
});

function recibo() {
  return {
    esquema: "vec.contratacion-temporal.recibo-informe-juridico.v1",
    operacion: "preparar",
    expediente_ref: EXPEDIENTE,
    version_resultante: 5,
    informe_ref: "informe:ct:sintetico-001",
    documento_ref: "documento:ct:sintetico-001",
    version_documento: 1,
    formato: "text/plain; charset=utf-8",
    nombre: "informe-juridico-desarrollo.txt",
    huella_documento_sha256: "a".repeat(64),
    recibo_ref: "recibo:ct:informe:sintetico-001",
    auditoria_ref: "auditoria:ct:informe:sintetico-001",
    evento_ref: "evento:ct:informe:sintetico-001",
    contenido_desarrollo:
      "DOCUMENTO DE DESARROLLO — SIN FIRMA NI VALIDEZ JURIDICA\n",
    confirmada_en: "2026-09-04T18:00:00Z",
  };
}

function raizFalsa() {
  const eventos = new Map();
  return {
    innerHTML: "",
    addEventListener(tipo, manejador) { eventos.set(tipo, manejador); },
    removeEventListener(tipo, manejador) {
      if (eventos.get(tipo) === manejador) eventos.delete(tipo);
    },
    contains() { return true; },
    querySelector() { return { focus() {}, scrollIntoView() {} }; },
    replaceChildren() { this.innerHTML = ""; },
    enviar() {
      const formulario = {
        closest(selector) {
          return selector === "[data-ct-informe-form]" ? this : null;
        },
        checkValidity() { return true; },
        reportValidity() {},
      };
      return eventos.get("submit")({ target: formulario, preventDefault() {} });
    },
  };
}

test("conserva documento y recibo si el historial no corresponde", async () => {
  const raiz = raizFalsa();
  const solicitudes = [];
  const confirmados = [];
  const referenciasConfirmacion = [];
  montarFormularioInformeJuridico({
    raiz,
    cliente: {
      async prepararInformeJuridico(solicitud) {
        solicitudes.push(solicitud);
        return recibo();
      },
      async consultarDetalleRRHH() {
        return {
          resumen: { expediente_ref: EXPEDIENTE, version: 4 },
          hitos: [{
            secuencia: 4,
            version_expediente: 4,
            accion_clave: "contratacion_temporal.unidad.asignar",
            fase_destino: "asignacion_unidad",
          }],
        };
      },
    },
    contexto: { expediente_ref: EXPEDIENTE, version_esperada: 4 },
    generarClaveIdempotencia: () => CLAVE,
    confirmarOperacion({ referencia }) { referenciasConfirmacion.push(referencia); return true; },
    alConfirmar: (resultado) => confirmados.push(resultado),
  });

  assert.match(raiz.innerHTML, /Expediente seleccionado/u);
  assert.doesNotMatch(raiz.innerHTML, /expediente:ct:|versión asignada|\{expediente\}|\{version\}/u);
  await raiz.enviar();
  assert.deepEqual(referenciasConfirmacion, ["Expediente seleccionado."]);
  assert.deepEqual(solicitudes, [{
    expediente_ref: EXPEDIENTE,
    version_esperada: 4,
    clave_idempotencia: CLAVE,
  }]);
  assert.match(raiz.innerHTML, /data-ct-informe-recibo/u);
  assert.match(raiz.innerHTML, /data-ct-informe-documento/u);
  assert.match(raiz.innerHTML, /data-ct-informe-historial-error/u);
  assert.deepEqual(confirmados, [recibo()]);
});

test("el informe nombra el expediente visible sin versión ni referencia interna", async () => {
  const raiz = raizFalsa();
  const referenciasConfirmacion = [];
  const solicitudes = [];
  montarFormularioInformeJuridico({
    raiz,
    cliente: {
      async prepararInformeJuridico(solicitud) { solicitudes.push(solicitud); return recibo(); },
      async consultarDetalleRRHH() { return { resumen: { expediente_ref: EXPEDIENTE, version: 5 }, hitos: [] }; },
    },
    contexto: { expediente_ref: EXPEDIENTE, version_esperada: 4, numero_visible: " 2026/95101 " },
    generarClaveIdempotencia: () => CLAVE,
    confirmarOperacion({ referencia }) { referenciasConfirmacion.push(referencia); return true; },
  });
  assert.match(raiz.innerHTML, /Expediente 2026\/95101/u);
  assert.doesNotMatch(raiz.innerHTML, /expediente:ct:|versión 4/u);
  await raiz.enviar();
  assert.deepEqual(referenciasConfirmacion, ["2026/95101"]);
  assert.deepEqual(solicitudes, [{ expediente_ref: EXPEDIENTE, version_esperada: 4, clave_idempotencia: CLAVE }]);
});

test("un catálogo incompleto usa el idioma de respaldo y no impide recuperar el historial", async () => {
  const raiz = raizFalsa();
  const idiomas = [];
  const catalogo = JSON.parse(await readFile(new URL(
    "../../../textos/es/contratacion-temporal-informe-juridico.json", import.meta.url,
  ), "utf8"));
  const leerCatalogoInforme = async (_modulo, opciones) => {
    idiomas.push(opciones ?? null);
    const mensajes = opciones ? catalogo.general : { sin_estructura: "Incomplete" };
    return {
      idioma: opciones ? opciones.idioma : "en",
      traducir(ruta) {
        const texto = mensajes[ruta.split(".")[1]];
        if (!texto) throw new Error("clave ausente");
        return texto;
      },
    };
  };
  const desmontar = montarFormularioInformeJuridico({
    raiz,
    cliente: {
      async prepararInformeJuridico() { return recibo(); },
      async consultarDetalleRRHH() {
        return { resumen: { expediente_ref: EXPEDIENTE, version: 5 }, hitos: [{
          secuencia: 5, version_expediente: 5,
          accion_clave: "contratacion_temporal.informe_juridico.generar",
          fase_destino: "informe_juridico", realizada_en: recibo().confirmada_en,
        }] };
      },
    },
    contexto: { expediente_ref: EXPEDIENTE, version_esperada: 4 },
    generarClaveIdempotencia: () => CLAVE, confirmarOperacion: () => true,
    leerCatalogoInforme,
  });
  await raiz.enviar();
  await new Promise((resolver) => setImmediate(resolver));
  assert.equal(idiomas.length, 2);
  assert.equal(idiomas[1].idioma, idiomas[1].porDefecto);
  assert.match(raiz.innerHTML, /data-ct-informe-recibo/u);
  assert.match(raiz.innerHTML, /data-ct-informe-historial/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-informe-historial-error/u);
  assert.match(raiz.innerHTML, /No se puede mostrar este documento por apartados/u);
  desmontar();
});

test("ofrece el informe al reabrir un expediente asignado", () => {
  const expediente = {
    expediente_ref: EXPEDIENTE,
    numero_visible: "2026/CT-001",
    version: 4,
    flujo_ref: "flujo:ct:sintetico",
    flujo_version: 1,
    flujo_huella: "b".repeat(64),
    cabecera: [{ clave: "unidad", valor: "unidad:seleccion" }],
    fases: [],
    tareas: [],
  };
  const html = renderizarModuloContratacionTemporal({
    vista: "expediente",
    carga: "listo",
    cuadro: { expedientes: [{
      expediente_ref: EXPEDIENTE,
      version: 4,
      fase_clave: "asignacion_unidad",
      estado_clave: "en_curso",
    }] },
    expediente,
    tarea_ref: "",
    mensaje_clave: "estado_expediente_listo",
    tipo_mensaje: "informacion",
  }, { informeJuridicoDisponible: true });

  assert.match(html, /data-ct-exp-informe-juridico/u);
});

test("el detalle base deja un solo lugar para el circuito de firma", () => {
  const expediente = {
    expediente_ref: EXPEDIENTE, numero_visible: "2026/CT-001", version: 5,
    flujo_ref: "flujo:ct:sintetico", flujo_version: 1, flujo_huella: "b".repeat(64),
    cabecera: [], fases: [], tareas: [], historial: [{
      accion_clave: "contratacion_temporal.informe_juridico.generar", accion: "Informe jurídico generado",
    }],
  };
  const html = renderizarModuloContratacionTemporal({
    vista: "expediente", carga: "listo", cuadro: { expedientes: [] }, expediente,
    tarea_ref: "", mensaje_clave: "", tipo_mensaje: "informacion",
  });
  assert.match(html, /ct-exp-siguiente-paso/u);
  assert.doesNotMatch(html, /ct-exp-firma-pendiente|ct-exp-enviar-firma-motivo/u);
  assert.doesNotMatch(html, /Firma confirmada|Documento firmado|recibo.*confirmado/iu);
});

test("no mantiene la firma pendiente cuando la historia ya avanzó", () => {
  const expediente = {
    expediente_ref: EXPEDIENTE, numero_visible: "2026/CT-001", version: 6,
    flujo_ref: "flujo:ct:sintetico", flujo_version: 1, flujo_huella: "b".repeat(64),
    cabecera: [], fases: [], tareas: [], historial: [
      { accion_clave: "contratacion_temporal.informe_juridico.generar", accion: "Informe jurídico generado" },
      { accion_clave: "contratacion_temporal.fiscalizacion.registrar", accion: "Fiscalización registrada" },
    ],
  };
  const html = renderizarModuloContratacionTemporal({
    vista: "expediente", carga: "listo", cuadro: { expedientes: [] }, expediente,
    tarea_ref: "", mensaje_clave: "", tipo_mensaje: "informacion",
  });
  assert.doesNotMatch(html, /Firma y remisión a Intervención/u);
});

test("el historial del informe reutiliza etiquetas legibles y traducciones", () => {
  const etiquetas = presentarEtiquetasHitoRRHH({
    accion_clave: "contratacion_temporal.analisis.registrar",
    fase_origen: null, fase_destino: "solicitud", estado_origen: "pendiente", estado_destino: "en_curso",
  }, { hito_analisis: "Análisis revisado" });
  assert.equal(etiquetas.accion, "Análisis revisado");
  assert.equal(etiquetas.faseOrigen, "—");
  assert.equal(etiquetas.faseDestino, "Solicitud");
  assert.equal(etiquetas.estadoDestino, "En trámite");
});

test("presenta el historial autorizado tras confirmar y conserva traducciones", async () => {
  const raiz = raizFalsa();
  const desmontar = montarFormularioInformeJuridico({
    raiz,
    cliente: {
      async prepararInformeJuridico() { return recibo(); },
      async consultarDetalleRRHH() {
        return { resumen: { expediente_ref: EXPEDIENTE, version: 5 }, hitos: [{
          secuencia: 5, version_expediente: 5,
          accion_clave: "contratacion_temporal.informe_juridico.generar",
          fase_origen: "asignacion_unidad", fase_destino: "informe_juridico",
          estado_origen: "en_curso", estado_destino: "en_curso",
          realizada_en: recibo().confirmada_en,
        }] };
      },
    },
    contexto: { expediente_ref: EXPEDIENTE, version_esperada: 4 },
    generarClaveIdempotencia: () => CLAVE, confirmarOperacion: () => true,
    mensajes: { hito_informe_juridico: "Informe preparado para revisión" },
  });
  await raiz.enviar();
  assert.match(raiz.innerHTML, /Informe preparado para revisión/u);
  assert.match(raiz.innerHTML, /data-ct-informe-recibo/u);
  assert.doesNotMatch(raiz.innerHTML, /data-ct-informe-historial-error/u);
  desmontar();
});
