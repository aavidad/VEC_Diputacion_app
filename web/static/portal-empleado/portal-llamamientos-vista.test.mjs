import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { obtenerDatosPresentacion, obtenerPropuestaPresentacion } from "./datos-presentacion.js";
import {
  actualizarConfiguracionLlamamiento,
  crearAsistenteLlamamientos,
  crearConfiguracionInicialLlamamiento,
  prepararOperacionLlamamiento,
  renderizarConfirmacionCompacta,
  renderizarDetalleLlamamientoBloqueado,
  renderizarPasosLlamamiento,
  validarConfiguracionLlamamiento,
} from "./portal-llamamientos-vista.js";

const escaparHTML = (valor) => String(valor ?? "")
  .replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;");

function utilidades({ permitido = true } = {}) {
  return {
    avisoPresentacion: (texto) => `<section>${escaparHTML(texto)}</section>`,
    encabezadoVista: (_sobrelinea, titulo, descripcion, acciones = "") => `<header><h2>${escaparHTML(titulo)}</h2><p>${escaparHTML(descripcion)}</p>${acciones}</header>`,
    escaparHTML,
    fuentePresentacion: () => '<span class="estado-chip">Datos sintéticos</span>',
    numero: (valor) => String(valor ?? 0),
    operacionPermitida: (operacion) => permitido && operacion === "emitir-llamamiento",
  };
}

function estadoDemo(paso = 1) {
  return {
    modoPresentacion: true,
    pasoLlamamiento: paso,
    necesidadSeleccionada: "DEMO-NEC-0045",
    propuestaLlamamiento: paso >= 2 ? obtenerPropuestaPresentacion("DEMO-NEC-0045") : null,
    confirmacionPropuestaLlamamiento: null,
    configuracionLlamamiento: null,
    erroresConfiguracionLlamamiento: [],
    reciboLlamamiento: null,
  };
}

function confirmacionReal() {
  return {
    propuesta_ref: "prp_0123456789abcdef",
    necesidad: { referencia: "nec_0123456789abcdef", version: "1" },
    bolsa: { referencia: "bol_0123456789abcdef", version: "3" },
    instantanea: { referencia: "ins_0123456789abcdef", version: "7" },
    politica: { referencia: "pol_0123456789abcdef", version: "2" },
    generada_en: "2026-07-19T08:00:00Z",
    total_evaluaciones: "2",
    orden_seleccionado: "2",
  };
}

test("los pasos futuros quedan deshabilitados y la presentación identifica su carácter DEMO", () => {
  const html = renderizarPasosLlamamiento({ modoPresentacion: true, pasoActual: 3, pasoMaximo: 3 });
  assert.equal(html.match(/ disabled/g)?.length, 1);
  assert.match(html, /Seleccionar bolsa/);
  assert.match(html, /Seleccionar candidatos · DEMO automático/);
  assert.match(html, /Configurar llamamiento/);
  assert.match(html, /Revisar y preparar DEMO/);
  assert.match(html, /data-paso="3" aria-current="step"/);
});

test("la ruta real solo habilita solicitar con capacidad y nunca abre configuración", () => {
  const datos = obtenerDatosPresentacion();
  const asistente = crearAsistenteLlamamientos(utilidades({ permitido: false }));
  const estado = { ...estadoDemo(1), modoPresentacion: false };
  datos.capacidades.solicitar_propuesta_llamamiento = false;
  const cerrado = asistente.renderizar(datos, estado);
  assert.match(cerrado, /data-accion="solicitar-propuesta" disabled/);
  assert.match(cerrado, /El servidor no concede la capacidad/);
  assert.match(cerrado, /<strong>MODO REAL<\/strong>/);
  assert.doesNotMatch(cerrado, /MODO DEMO PERSISTENTE/);

  datos.capacidades.solicitar_propuesta_llamamiento = true;
  const abierto = asistente.renderizar(datos, estado);
  const boton = abierto.match(/<button[^>]+data-accion="solicitar-propuesta"[^>]*>/)?.[0] || "";
  assert.doesNotMatch(boton, /disabled/);
  assert.match(abierto, /Solicitar confirmación al servidor/);

  estado.pasoLlamamiento = 2;
  estado.confirmacionPropuestaLlamamiento = confirmacionReal();
  const confirmada = asistente.renderizar(datos, estado);
  assert.match(confirmada, /Confirmación del servidor/);
  assert.match(confirmada, /Detalle no disponible/);
  assert.doesNotMatch(confirmada, /data-llamamiento-campo/);
  assert.equal(renderizarPasosLlamamiento({ modoPresentacion: false, pasoActual: 2, pasoMaximo: 2 })
    .match(/ disabled/g)?.length, 2, "configuración y preparación deben permanecer cerradas");
});

test("la selección DEMO solo muestra orden, elegibilidad y motivos minimizados", () => {
  const datos = obtenerDatosPresentacion();
  const salida = crearAsistenteLlamamientos(utilidades()).renderizar(datos, estadoDemo(2));
  for (const texto of ["Evaluaciones sintéticas sin identidad ni contacto", "Posición 1", "Elegible", "R4 · Indisponibilidad", "Incluida por el motor"]) {
    assert.match(salida, new RegExp(texto));
  }
  assert.doesNotMatch(salida, /\bDNI\b|nombre y apellidos|data-candidato|type="checkbox"/i);
  assert.doesNotMatch(salida, /DEMO-PER-/);
  assert.deepEqual(Object.keys(obtenerPropuestaPresentacion("DEMO-NEC-0045").evaluaciones[0]), ["orden", "resultado", "motivos"]);
});

test("la configuración es cerrada, validada y no acepta campos ambientales", () => {
  const datos = obtenerDatosPresentacion();
  const estado = estadoDemo(3);
  estado.configuracionLlamamiento = crearConfiguracionInicialLlamamiento(datos, estado);
  assert.equal(validarConfiguracionLlamamiento(datos, estado).length, 0);
  estado.configuracionLlamamiento = actualizarConfiguracionLlamamiento(
    estado.configuracionLlamamiento, "jornada", "Jornada inventada",
  );
  assert.match(validarConfiguracionLlamamiento(datos, estado).map((error) => error.mensaje).join(" "), /catálogo vigente/);
  assert.throws(
    () => actualizarConfiguracionLlamamiento(estado.configuracionLlamamiento, "actor_ref", "DEMO-PERFIL-ADMIN"),
    /no válido/,
  );
  assert.throws(
    () => actualizarConfiguracionLlamamiento(estado.configuracionLlamamiento, "destino", "x\u0000y"),
    /no válido/,
  );
  const salida = crearAsistenteLlamamientos(utilidades()).renderizar(datos, estado);
  assert.match(salida, /data-formulario-llamamiento aria-describedby="marca-modo-llamamiento"/);
  assert.match(salida, /MODO DEMO PERSISTENTE/);
});

test("el avance valida cada transición y el reinicio borra configuración y recibo", () => {
  const datos = obtenerDatosPresentacion();
  const asistente = crearAsistenteLlamamientos(utilidades());
  const estado = estadoDemo(2);
  assert.equal(asistente.avanzar(datos, estado).ok, true);
  assert.equal(estado.pasoLlamamiento, 3);
  estado.configuracionLlamamiento = actualizarConfiguracionLlamamiento(
    estado.configuracionLlamamiento, "destino", "",
  );
  assert.equal(asistente.avanzar(datos, estado).ok, false);
  assert.equal(estado.pasoLlamamiento, 3);
  assert.ok(estado.erroresConfiguracionLlamamiento.length > 0);
  estado.reciboLlamamiento = { referencia: "DEMO-REC-000001" };
  asistente.reiniciar(estado, "DEMO-NEC-0038");
  assert.deepEqual({
    paso: estado.pasoLlamamiento,
    necesidad: estado.necesidadSeleccionada,
    propuesta: estado.propuestaLlamamiento,
    configuracion: estado.configuracionLlamamiento,
    errores: estado.erroresConfiguracionLlamamiento,
    recibo: estado.reciboLlamamiento,
  }, {
    paso: 1, necesidad: "DEMO-NEC-0038", propuesta: null, configuracion: null, errores: [], recibo: null,
  });
});

test("la preparación usa el comando existente, campos cerrados y objetivo del expediente", () => {
  const datos = obtenerDatosPresentacion();
  const estado = estadoDemo(3);
  estado.configuracionLlamamiento = crearConfiguracionInicialLlamamiento(datos, estado);
  const preparacion = prepararOperacionLlamamiento(datos, estado);
  assert.equal(preparacion.ok, true);
  assert.equal(preparacion.objetivo, "DEMO-LLA-045");
  assert.deepEqual(Object.keys(preparacion.campos), [
    "bolsa", "destino", "jornada", "duracion", "regla", "plazo_respuesta", "canales", "plantilla",
  ]);
  assert.doesNotMatch(JSON.stringify(preparacion), /actor|dni|nombre|contacto/i);
});

test("el último paso diferencia permiso, simulación y recibo sin efectos reales", () => {
  const datos = obtenerDatosPresentacion();
  const estado = estadoDemo(4);
  estado.configuracionLlamamiento = crearConfiguracionInicialLlamamiento(datos, estado);
  const bloqueada = crearAsistenteLlamamientos(utilidades({ permitido: false })).renderizar(datos, estado);
  assert.match(bloqueada, /data-accion="preparar-llamamiento-demo" disabled/);

  estado.reciboLlamamiento = {
    referencia: "DEMO-REC-000001",
    actor: "DEMO-PERFIL-ADMIN-FUNCIONAL-BOLSA-01",
    instante: "2026-07-19T08:00:00.000Z",
    objetivo: "DEMO-LLA-045",
    resultado: "Preparado",
  };
  const salida = crearAsistenteLlamamientos(utilidades()).renderizar(datos, estado);
  assert.match(salida, /Preparar DEMO · sin enviar/);
  assert.match(salida, /Recibo volátil del recorrido/);
  assert.match(salida, /DEMO-REC-000001/);
  assert.match(salida, /Efectos reales[\s\S]*?<strong>No<\/strong>/);
  assert.match(salida, /No se ha enviado ninguna comunicación/);
  assert.match(salida, /MODO DEMO PERSISTENTE/);
});

test("la confirmación compacta escapa referencias y declara el bloqueo", () => {
  const html = renderizarConfirmacionCompacta({ ...confirmacionReal(), propuesta_ref: "propuesta:<script>" });
  assert.doesNotMatch(html, /<script>/);
  assert.match(html, /propuesta:&lt;script&gt;/);
  assert.match(html, /Detalle no disponible/);
  assert.match(renderizarDetalleLlamamientoBloqueado(), /configuración del llamamiento/);
});

test("los estilos están aislados, cacheados y cubren 1040, 780 y móvil", async () => {
  const [html, estilos, vista, eventos, fuenteUtilidades] = await Promise.all([
    readFile(new URL("index.html", import.meta.url), "utf8"),
    readFile(new URL("portal-llamamientos.css", import.meta.url), "utf8"),
    readFile(new URL("portal-llamamientos-vista.js", import.meta.url), "utf8"),
    readFile(new URL("portal-eventos.js", import.meta.url), "utf8"),
    readFile(new URL("portal-vistas-utilidades.js", import.meta.url), "utf8"),
  ]);
  assert.match(html, /portal-llamamientos\.css\?v=20260719-asistente-llamamientos-v2/);
  assert.match(html, /portal\.js\?v=20260725-aislamiento-modular-v2/);
  assert.match(estilos, /@media \(max-width: 1040px\)/);
  assert.match(estilos, /@media \(max-width: 780px\)/);
  assert.match(estilos, /@media \(max-width: 520px\)/);
  assert.match(estilos, /@media \(forced-colors: active\)/);
  assert.match(estilos, /\.estado-asistente\s*\{[\s\S]{0,100}position: sticky/);
  assert.doesNotMatch(html, /Presentación funcional/);
  assert.doesNotMatch(`${vista}\n${eventos}`, /localStorage|sessionStorage|document\.cookie/);
  assert.match(eventos, /estado\.pasoLlamamiento !== 4[\s\S]{0,100}operacionPermitida\("emitir-llamamiento"\)/);
  assert.match(eventos, /ejecutarOperacionPresentacion\([\s\S]{0,100}"emitir-llamamiento"/);
  assert.match(fuenteUtilidades, /operacionPermitida,[\s\S]{0,80}tabla/);
});
