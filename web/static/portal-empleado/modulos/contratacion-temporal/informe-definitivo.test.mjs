import assert from "node:assert/strict";
import test from "node:test";
import { montarModuloContratacionTemporal, renderizarModuloContratacionTemporal } from "./vista-expedientes.js";
import { solicitudInformeDefinitivoDesdeEstado } from "./componentes-expedientes.js";
import { crearClienteHTTPContratacionTemporal, RUTAS_HTTP_CONTRATACION_TEMPORAL } from "./cliente-http.js";
import { crearAdaptadorHTTPExpedientesContratacionTemporal } from "./adaptador-http-expedientes.js";
import { crearPresentadorExpedientesContratacionTemporal } from "./presentador-expedientes.js";
import { validarExpedienteContratacionTemporal } from "./contrato-expedientes.js";
import { crearClienteHTTPBorradorRRHH } from "./cliente-http-informe-definitivo.js";

function estadoReal() {
  const expediente = {
    demostracion: false, expediente_ref: "expediente:ct:sintetico-009", version: 7,
    numero_visible: "2026/CT-009", flujo_ref: "flujo:ct:desarrollo", flujo_version: 1,
    flujo_huella: "a".repeat(64), cabecera: [], fases: [], tareas: [],
  };
  return {
    vista: "expediente", carga: "listo", expediente, expediente_ref: expediente.expediente_ref,
    cuadro: { demostracion: false, indicadores: [], expedientes: [{
      ...expediente, fase_clave: "nombramiento", estado_clave: "en_curso",
    }] },
    tarea_ref: "", filtros: { texto: "", estado: "", fase: "" },
    ocupado: false, actualizacion_pendiente: false, resultado_indeterminado: false,
    mensaje_clave: "estado_expediente_listo", tipo_mensaje: "informacion",
  };
}

const perfiles = [
  { tipo: "informe_definitivo", accion: "descargar-informe-definitivo", nombre: "informe-definitivo-borrador.pdf" },
  { tipo: "resolucion", accion: "descargar-resolucion", nombre: "resolucion-borrador.pdf" },
  { tipo: "diligencia", accion: "descargar-diligencia", nombre: "diligencia-borrador.pdf" },
  { tipo: "toma_posesion", accion: "descargar-toma-posesion", nombre: "toma-posesion-borrador.pdf" },
  { tipo: "notificacion", accion: "descargar-notificacion", nombre: "notificacion-borrador.pdf" },
  { tipo: "comunicacion_centro", accion: "descargar-comunicacion-centro", nombre: "comunicacion-centro-borrador.pdf" },
];

async function montar(descargar, perfil = perfiles[0], estado = estadoReal()) {
  const eventos = new Map();
  const mensaje = { textContent: "", setAttribute() {} };
  const descargas = [], creados = [], revocados = [];
  const botones = perfiles.map(({ accion }) => ({ disabled: false, dataset: {
    ctExpAccion: accion, expedienteRef: "expediente:ajeno", version: 99,
  } }));
  const boton = botones.find((control) => control.dataset.ctExpAccion === perfil.accion);
  const raiz = {
    innerHTML: "", contains: () => true,
    addEventListener: (tipo, fn) => eventos.set(tipo, fn),
    removeEventListener: (tipo) => eventos.delete(tipo),
    querySelector: (selector) => selector === "[data-ct-exp-mensaje]" ? mensaje : null,
    querySelectorAll: (selector) => selector.includes("data-ct-exp-accion") ? botones : [],
  };
  const montaje = await montarModuloContratacionTemporal({
    raiz, presentador: {
      obtenerEstado: () => estado, cargar() { throw new Error("no recargar ni ejecutar"); },
      cambiarVista(vista) { estado.vista = vista; },
    },
    clienteBorradorRRHH: { descargarBorrador: descargar },
    entornoDescarga: {
      URL: {
        createObjectURL(blob) { creados.push(blob); return "blob:sintetico-009"; },
        revokeObjectURL(url) { revocados.push(url); },
      },
      document: { body: { append() {} }, createElement() { return {
        click() { descargas.push({ href: this.href, download: this.download, hidden: this.hidden }); }, remove() {},
      }; } },
    },
  });
  const click = (selector = "[data-ct-exp-accion]", control = boton) => eventos.get("click")({
    target: { closest: (buscado) => buscado === selector ? control : null }, preventDefault() {},
  });
  return { estado, raiz, boton, botones, mensaje, montaje, click, descargas, creados, revocados };
}

test("seis botones de cabecera v7 real sin tareas, nunca fase/versión/consulta pendiente ajenas", () => {
  const estado = estadoReal();
  const html = renderizarModuloContratacionTemporal(estado);
  assert.match(html, /<section class="ct-exp-cabecera-expediente">[^]*data-ct-exp-accion="descargar-informe-definitivo"[^]*<\/section>/u);
  assert.match(html, /Descargar informe · borrador de desarrollo/u);
  assert.match(html, /<section class="ct-exp-cabecera-expediente">[^]*data-ct-exp-accion="descargar-resolucion"[^]*<\/section>/u);
  assert.match(html, /Descargar resolución · borrador de desarrollo/u);
  assert.match(html, /<section class="ct-exp-cabecera-expediente">[^]*data-ct-exp-accion="descargar-diligencia"[^]*<\/section>/u);
  assert.match(html, /Descargar diligencia · borrador de desarrollo/u);
  assert.match(html, /<section class="ct-exp-cabecera-expediente">[^]*data-ct-exp-accion="descargar-toma-posesion"[^]*<\/section>/u);
  assert.match(html, /Descargar toma de posesión · borrador de desarrollo/u);
  assert.match(html, /<section class="ct-exp-cabecera-expediente">[^]*data-ct-exp-accion="descargar-notificacion"[^]*<\/section>/u);
  assert.match(html, /Descargar notificación · borrador de desarrollo/u);
  assert.match(html, /<section class="ct-exp-cabecera-expediente">[^]*data-ct-exp-accion="descargar-comunicacion-centro"[^]*<\/section>/u);
  assert.match(html, /Descargar comunicación al centro · borrador de desarrollo/u);
  assert.deepEqual(solicitudInformeDefinitivoDesdeEstado(estado), {
    expediente_ref: estado.expediente_ref, version_observada: 7,
  });
  for (const modificar of [
    (e) => { e.expediente.demostracion = true; }, (e) => { e.cuadro.demostracion = true; },
    (e) => { e.expediente.version = 6; }, (e) => { e.cuadro.expedientes[0].version = 6; },
    (e) => { Object.assign(e.cuadro.expedientes[0], { fase_clave: "fiscalizacion" }); },
    (e) => { Object.assign(e.cuadro.expedientes[0], { estado_clave: "completado" }); },
    (e) => { e.carga = "cargando"; }, (e) => { e.ocupado = true; },
    (e) => { e.actualizacion_pendiente = true; }, (e) => { e.resultado_indeterminado = true; },
    (e) => { e.expediente_ref = "expediente:otro"; }, (e) => { e.vista = "documentos"; },
  ]) {
    const otro = estadoReal(); modificar(otro);
    assert.equal(solicitudInformeDefinitivoDesdeEstado(otro), null);
    assert.doesNotMatch(renderizarModuloContratacionTemporal(otro), /data-ct-exp-accion="descargar-informe-definitivo"/u);
    assert.doesNotMatch(renderizarModuloContratacionTemporal(otro), /data-ct-exp-accion="descargar-resolucion"/u);
    assert.doesNotMatch(renderizarModuloContratacionTemporal(otro), /data-ct-exp-accion="descargar-diligencia"/u);
    assert.doesNotMatch(renderizarModuloContratacionTemporal(otro), /data-ct-exp-accion="descargar-toma-posesion"/u);
    assert.doesNotMatch(renderizarModuloContratacionTemporal(otro), /data-ct-exp-accion="descargar-notificacion"/u);
    assert.doesNotMatch(renderizarModuloContratacionTemporal(otro), /data-ct-exp-accion="descargar-comunicacion-centro"/u);
  }
});

async function estadoResolucionDesdeHTTP(modificar = () => {}) {
  const resumen = {
    expediente_ref: "expediente:ct:pdf-historico", numero_visible: "2026/CT-009", version: 8,
    flujo_ref: "flujo:ct:desarrollo", flujo_version: 1, flujo_huella_sha256: "a".repeat(64),
    fase_clave: "nombramiento", estado_clave: "en_curso", centro_ref: "centro:prueba",
    categoria_ref: "categoria:prueba", creado_en: "2026-09-03T08:00:00Z",
    actualizado_en: "2026-09-09T22:27:12Z",
  };
  const acciones = ["registrar_solicitud", "registrar_analisis", "registrar_cobertura",
    "registrar_asignacion", "registrar_informe_juridico", "registrar_fiscalizacion",
    "registrar_propuesta_formalizacion", "registrar_resolucion_formalizacion"];
  const detalle = {
    esquema: "vec.contratacion-temporal.detalle-rrhh.v1", resumen,
    solicitud: { grupo_subgrupo: "A2", motivo_clave: "sustitucion",
      periodo_inicio: "2026-09-04T00:00:00Z", periodo_fin: "2026-12-31T00:00:00Z" },
    hitos: acciones.map((accion_clave, indice) => ({
      secuencia: indice + 1, version_expediente: indice + 1, accion_clave,
      realizada_en: "2026-09-09T22:27:12Z", fase_origen: "nombramiento",
      fase_destino: "nombramiento", estado_origen: "en_curso", estado_destino: "en_curso",
    })),
  };
  modificar(detalle);
  const cliente = crearClienteHTTPContratacionTemporal({ fetchImpl: async (ruta) => {
    assert.ok([RUTAS_HTTP_CONTRATACION_TEMPORAL.cuadroRRHH,
      RUTAS_HTTP_CONTRATACION_TEMPORAL.detalleRRHH].includes(ruta));
    const data = ruta === RUTAS_HTTP_CONTRATACION_TEMPORAL.cuadroRRHH
      ? { esquema: "vec.contratacion-temporal.cuadro-rrhh.v1", generada_en: resumen.actualizado_en,
        expedientes: [resumen], hay_mas: false } : detalle;
    return new Response(JSON.stringify({ data }), {
      status: 200, headers: { "Content-Type": "application/json; charset=utf-8" },
    });
  } });
  const fuente = crearAdaptadorHTTPExpedientesContratacionTemporal({ cliente });
  await fuente.listar();
  const presentador = crearPresentadorExpedientesContratacionTemporal({ fuente, capacidades: fuente.capacidades });
  await presentador.cargar();
  await presentador.seleccionarExpediente(resumen.expediente_ref);
  const estado = presentador.obtenerEstado();
  assert.equal(estado.carga, "listo");
  return estado;
}

test("resolución v8 proyectada desde HTTP conserva los seis PDF de la propuesta v7", async () => {
  const estado = await estadoResolucionDesdeHTTP();
  assert.deepEqual(solicitudInformeDefinitivoDesdeEstado(estado), {
    expediente_ref: estado.expediente_ref, version_observada: 8,
  });
  const html = renderizarModuloContratacionTemporal(estado);
  for (const perfil of perfiles) assert.ok(html.includes(`data-ct-exp-accion="${perfil.accion}"`));
});

test("anotación v9 conserva los seis borradores de la propuesta v7", async () => {
  const estado = await estadoResolucionDesdeHTTP((detalle) => {
    detalle.resumen.version = 9;
    detalle.hitos.push({
      secuencia: 9, version_expediente: 9,
      accion_clave: "contratacion_temporal.anotacion_administrativa.registrar",
      realizada_en: "2026-09-12T10:00:00Z", fase_origen: "nombramiento",
      fase_destino: "nombramiento", estado_origen: "en_curso", estado_destino: "en_curso",
    });
  });
  assert.deepEqual(solicitudInformeDefinitivoDesdeEstado(estado), {
    expediente_ref: estado.expediente_ref, version_observada: 9,
  });
  const html = renderizarModuloContratacionTemporal(estado);
  for (const perfil of perfiles) assert.ok(html.includes(`data-ct-exp-accion="${perfil.accion}"`));
});

test("una v8 sin propuesta/resolución histórica exacta no muestra descargas", async () => {
  for (const modificar of [
    (d) => { d.hitos = []; },
    (d) => { d.hitos.pop(); },
    (d) => { d.hitos[6].accion_clave = "otra_propuesta"; },
    (d) => { d.hitos[7].accion_clave = "otra_resolucion"; },
    (d) => { d.hitos[6].version_expediente = 6; },
    (d) => { d.hitos[7].version_expediente = 9; },
    (d) => { d.hitos[7].secuencia = 9; },
    (d) => { d.hitos[0].secuencia = 2; },
    (d) => { d.hitos[6].fase_destino = "fiscalizacion"; },
    (d) => { d.hitos[6].estado_destino = "completado"; },
    (d) => { d.hitos[7].fase_origen = "fiscalizacion"; },
    (d) => { d.hitos[7].fase_destino = "fiscalizacion"; },
    (d) => { d.hitos[7].estado_origen = "pendiente"; },
    (d) => { d.hitos[7].estado_destino = "completado"; },
    (d) => { d.hitos.push({ ...d.hitos[7] }); },
    (d) => { d.resumen.fase_clave = "fiscalizacion"; },
    (d) => { d.resumen.estado_clave = "completado"; },
    (d) => { d.resumen.version = 9; },
  ]) {
    const estado = await estadoResolucionDesdeHTTP(modificar);
    assert.equal(solicitudInformeDefinitivoDesdeEstado(estado), null);
    const html = renderizarModuloContratacionTemporal(estado);
    for (const perfil of perfiles) assert.ok(!html.includes(`data-ct-exp-accion="${perfil.accion}"`));
  }
});

test("indicador documental cerrado a7 en v8 real, sin conceder descarga con estado cruzado", async () => {
  const estado = await estadoResolucionDesdeHTTP();
  assert.equal(estado.expediente.version_propuesta_documental, 7);
  assert.deepEqual(validarExpedienteContratacionTemporal(estado.expediente), estado.expediente);
  assert.ok(Object.isFrozen(estado.expediente));
  for (const cambio of [
    ...[null, true, "7", 6, 8].map(version_propuesta_documental => ({ version_propuesta_documental })),
    { version: 7 }, { version: 10 }, { demostracion: true }, { campo_desconocido: 7 },
  ]) assert.throws(() => validarExpedienteContratacionTemporal({ ...estado.expediente, ...cambio }), TypeError);
  for (const modificar of [
    (e) => { delete e.expediente.version_propuesta_documental; },
    (e) => { e.cuadro.expedientes[0].version = 7; },
    (e) => { e.expediente_ref = "expediente:otro"; },
    (e) => { e.expediente.demostracion = true; },
    (e) => { e.cuadro.demostracion = true; },
    (e) => { e.actualizacion_pendiente = true; },
  ]) {
    const otro = structuredClone(estado); modificar(otro);
    assert.equal(solicitudInformeDefinitivoDesdeEstado(otro), null);
  }
});

test("un click en resolución v8 usa el cliente PDF existente y solicita la versión actual", async () => {
  const estado = await estadoResolucionDesdeHTTP(), llamadas = [];
  const pdf = "%PDF-1.7\nPrueba aislada de transporte\n%%EOF";
  const http = crearClienteHTTPBorradorRRHH({ fetchImpl: async (ruta, opciones) => {
    llamadas.push({ ruta, opciones });
    return new Response(pdf, { headers: { "Content-Type": "application/pdf",
      "Content-Disposition": 'attachment; filename="resolucion-borrador.pdf"' } });
  } });
  const vista = await montar(http.descargarBorrador, perfiles[1], estado);
  try {
    assert.equal(llamadas.length, 0);
    await vista.click();
    assert.equal(llamadas.length, 1);
    assert.equal(llamadas[0].ruta, RUTAS_HTTP_CONTRATACION_TEMPORAL.detalleRRHH);
    assert.equal(llamadas[0].opciones.method, "POST");
    assert.deepEqual(JSON.parse(llamadas[0].opciones.body), {
      expediente_ref: estado.expediente_ref, version_observada: 8,
    });
    assert.equal(llamadas[0].opciones.headers.Accept, "application/pdf; documento=resolucion-desarrollo");
    assert.equal(vista.creados.length, 1);
    assert.equal(await vista.creados[0].text(), pdf);
    await http.descargarBorrador({ expediente_ref: estado.expediente_ref, version_observada: 8 }, { tipo: "resolucion" });
    assert.equal(llamadas.length, 2);
  } finally { vista.montaje.desmontar(); }
});

test("solo click explícito, payload desde estado no DOM; Blob nominal y revocación al desmontar", async () => {
  for (const perfil of perfiles) {
    const llamadas = [];
    const vista = await montar(async (solicitud, opciones) => {
      llamadas.push({ solicitud, opciones });
      return new Blob(["%PDF-1.7"], { type: "application/pdf" });
    }, perfil);
    assert.equal(llamadas.length, 0);
    const detalle = vista.raiz.innerHTML;
    await vista.click();
    assert.deepEqual(llamadas[0].solicitud, { expediente_ref: vista.estado.expediente_ref, version_observada: vista.estado.expediente.version });
    assert.equal(llamadas.length, 1);
    assert.equal(llamadas[0].opciones.tipo, perfil.tipo);
    assert.equal(vista.raiz.innerHTML, detalle);
    assert.deepEqual(vista.descargas, [{ href: "blob:sintetico-009", download: perfil.nombre, hidden: true }]);
    assert.ok(vista.botones.every((control) => control.disabled === false));
    assert.match(vista.mensaje.textContent, /Sin firma ni nombramiento eficaz/u);
    vista.montaje.desmontar();
    assert.deepEqual(vista.revocados, ["blob:sintetico-009"]);
  }
});

test("errores conservan detalle y permiten únicamente nueva descarga explícita", async () => {
  for (const perfil of perfiles) {
    let llamadas = 0;
    const vista = await montar(async () => {
      llamadas += 1;
      throw { codigo: "documento_no_disponible", envelopeValido: true };
    }, perfil);
    const detalle = vista.raiz.innerHTML;
    await vista.click();
    assert.equal(vista.raiz.innerHTML, detalle);
    assert.match(vista.mensaje.textContent, /no está disponible/u);
    assert.equal(vista.boton.disabled, false);
    assert.ok(vista.botones.every((control) => control.disabled === false));
    assert.equal(llamadas, 1);
    assert.deepEqual(vista.creados, []);
    await vista.click();
    assert.equal(llamadas, 2);
    vista.montaje.desmontar();
  }
});

test("navegar o desmontar cancela, descarta respuesta tardía y evita doble envío", async () => {
  for (const perfil of perfiles) {
    for (const desmontar of [false, true]) {
      let completar, signal;
      let llamadas = 0;
      const vista = await montar((_, opciones) => {
        llamadas += 1; signal = opciones.signal;
        return new Promise((resolve) => { completar = resolve; });
      }, perfil);
      const pendiente = vista.click();
      assert.equal(vista.boton.disabled, true);
      assert.ok(vista.botones.every((control) => control.disabled === true));
      await vista.click();
      for (const control of vista.botones.filter((otro) => otro !== vista.boton)) {
        await vista.click("[data-ct-exp-accion]", control);
      }
      assert.equal(llamadas, 1);
      if (desmontar) vista.montaje.desmontar();
      else await vista.click("[data-ct-exp-vista]", { dataset: { ctExpVista: "alta" } });
      assert.equal(signal.aborted, true);
      completar(new Blob(["%PDF-1.7"], { type: "application/pdf" }));
      await pendiente;
      assert.deepEqual(vista.creados, []);
      assert.deepEqual(vista.descargas, []);
      vista.montaje.desmontar();
    }
  }
});
