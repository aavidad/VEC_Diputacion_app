import assert from "node:assert/strict";
import test from "node:test";
import { montarModuloContratacionTemporal, renderizarModuloContratacionTemporal } from "./vista-expedientes.js";
import { solicitudInformeDefinitivoDesdeEstado } from "./componentes-expedientes.js";

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
];

async function montar(descargar, perfil = perfiles[0]) {
  const estado = estadoReal();
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

test("tres botones de cabecera v7 real sin tareas, nunca fase/versión/consulta pendiente ajenas", () => {
  const estado = estadoReal();
  const html = renderizarModuloContratacionTemporal(estado);
  assert.match(html, /<section class="ct-exp-cabecera-expediente">[^]*data-ct-exp-accion="descargar-informe-definitivo"[^]*<\/section>/u);
  assert.match(html, /Descargar informe · borrador de desarrollo/u);
  assert.match(html, /<section class="ct-exp-cabecera-expediente">[^]*data-ct-exp-accion="descargar-resolucion"[^]*<\/section>/u);
  assert.match(html, /Descargar resolución · borrador de desarrollo/u);
  assert.match(html, /<section class="ct-exp-cabecera-expediente">[^]*data-ct-exp-accion="descargar-diligencia"[^]*<\/section>/u);
  assert.match(html, /Descargar diligencia · borrador de desarrollo/u);
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
  }
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
    assert.deepEqual(llamadas[0].solicitud, { expediente_ref: vista.estado.expediente_ref, version_observada: 7 });
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
