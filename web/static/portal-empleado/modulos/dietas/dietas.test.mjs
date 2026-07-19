import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import {
  ESQUEMA_CONTEXTO_ACTOR_FRONTEND,
  validarYCongelarContextoActor,
} from "../../identidad/contexto-actor.js";
import { crearAdaptadorDietasPresentacion } from "./adaptador-presentacion.js";
import {
  CAPACIDAD_CONSULTAR_AUDITORIA,
  CAPACIDAD_CONSULTAR_GASTO,
  CAPACIDAD_CONSULTAR_RUTA,
  CAPACIDAD_GESTIONAR_GASTO,
  CAPACIDAD_GESTIONAR_RUTA,
} from "./contrato.js";
import { crearDatosDietasPresentacion } from "./datos-presentacion.js";
import { crearTraductorDietas, MENSAJES_DIETAS_ES } from "./i18n.js";
import { crearPresentadorDietas } from "./presentador.js";
import { renderizarDietas } from "./vista.js";

const CONTEXTO_COMPARTIDO = validarYCongelarContextoActor({
  esquema: ESQUEMA_CONTEXTO_ACTOR_FRONTEND,
  revision: 1,
  demostracion: true,
  persona_ref: "per_demo_persona_interna_dietas_000001",
  cuenta_ref: "cta_demo_cuenta_interna_dietas_000001",
  perfil_ref: "prf_demo_perfil_interno_dietas_000001",
  actor: {
    actor_ref: "DEMO-PERFIL-INTERNO-COMPARTIDO-01",
    iniciales: "AI",
    nombre_visible: "Agente interno DEMO",
  },
  rol: {
    clave: "empleado_publico",
    etiqueta: "Personal de la Diputación · escenario DEMO",
  },
  ambito: {
    clase: "personal_interno",
    organizacion_ref: "org_demo_diputacion_granada_000001",
    unidad_ref: "uni_demo_unidad_interna_dietas_000001",
    modulos: ["bolsa", "cronos", "dietas"],
  },
  autenticacion: {
    sesion_ref: "ses_demo_sesion_interna_dietas_000001",
    metodo: "demo",
    garantia: "bajo",
  },
  resuelto_en: "2026-07-19T00:00:00.000Z",
});

const CONTEXTO_PRODUCTIVO = validarYCongelarContextoActor({
  esquema: ESQUEMA_CONTEXTO_ACTOR_FRONTEND,
  revision: 2,
  demostracion: false,
  persona_ref: "per_persona_interna_dietas_000001",
  cuenta_ref: "cta_cuenta_interna_dietas_000001",
  perfil_ref: "prf_perfil_interno_dietas_000001",
  actor: {
    actor_ref: "prf_perfil_interno_dietas_000001",
    iniciales: "PI",
    nombre_visible: "Persona interna",
  },
  rol: {
    clave: "empleado_publico",
    etiqueta: "Personal de la Diputación",
  },
  ambito: {
    clase: "personal_interno",
    organizacion_ref: "org_diputacion_granada_interna_000001",
    unidad_ref: "uni_unidad_interna_dietas_000001",
    modulos: ["bolsa", "cronos", "dietas"],
  },
  autenticacion: {
    sesion_ref: "ses_sesion_interna_dietas_000001",
    metodo: "kerberos_ad",
    garantia: "alto",
  },
  resuelto_en: "2026-07-19T08:00:00.000Z",
});

const CAPACIDADES_EMPLEADO = Object.freeze([
  CAPACIDAD_CONSULTAR_GASTO,
  CAPACIDAD_GESTIONAR_GASTO,
  CAPACIDAD_CONSULTAR_RUTA,
  CAPACIDAD_GESTIONAR_RUTA,
]);
const CAPACIDADES_COMPLETAS = Object.freeze([...CAPACIDADES_EMPLEADO, CAPACIDAD_CONSULTAR_AUDITORIA]);
const t = crearTraductorDietas();

function presentador(capacidades = CAPACIDADES_EMPLEADO) {
  return crearPresentadorDietas({
    datos: adaptador(capacidades).obtenerDatos(),
    contextoActor: CONTEXTO_COMPARTIDO,
    capacidades,
    origenComprobacion: "https://vec.demo.dipgra.es",
  });
}

function adaptador(capacidades = CAPACIDADES_EMPLEADO) {
  return crearAdaptadorDietasPresentacion({
    contextoActor: CONTEXTO_COMPARTIDO,
    capacidades,
    reloj: () => new Date("2026-07-19T10:15:00Z"),
    crearReferencia: () => "DEMO-DIE-NUEVA-999",
  });
}

function datosProductivos() {
  const datos = crearDatosDietasPresentacion(CONTEXTO_COMPARTIDO);
  datos.origen = { demostracion: false, efectos_reales: true, adaptador: "http_interno" };
  datos.comisiones.forEach((comision, indiceComision) => {
    comision.titular_ref = CONTEXTO_PRODUCTIVO.actor.actor_ref;
    comision.historial.forEach((evento, indiceEvento) => {
      evento.recibo = `recibo:dietas:${indiceComision}:${indiceEvento}:2026`;
    });
  });
  return datos;
}

test("conserva por referencia el ContextoActor común y rechaza expedientes ajenos", () => {
  assert.throws(() => crearDatosDietasPresentacion(), /contexto debe estar validado/);
  const datos = crearDatosDietasPresentacion(CONTEXTO_COMPARTIDO);
  const modelo = crearPresentadorDietas({
    datos, contextoActor: CONTEXTO_COMPARTIDO, capacidades: CAPACIDADES_EMPLEADO,
  }).obtenerModelo();
  assert.strictEqual(modelo.identidad, CONTEXTO_COMPARTIDO);
  assert.ok(datos.comisiones.every((item) => item.titular_ref === CONTEXTO_COMPARTIDO.actor.actor_ref));

  const contaminados = structuredClone(datos);
  contaminados.comisiones[0].titular_ref = "DEMO-PERFIL-INTERNO-AJENO-01";
  assert.throws(
    () => crearPresentadorDietas({
      datos: contaminados, contextoActor: CONTEXTO_COMPARTIDO, capacidades: CAPACIDADES_EMPLEADO,
    }),
    /expediente ajeno/,
  );
});

test("deniega por defecto y no proyecta referencias, importes ni rutas", () => {
  const modelo = presentador([]).obtenerModelo();
  assert.equal(modelo.capacidades.consultarGastos, false);
  assert.deepEqual(modelo.comisiones, []);
  assert.equal(modelo.seleccionada, null);
  const html = renderizarDietas(modelo);
  assert.match(html, /Acceso a Dietas no autorizado/);
  assert.doesNotMatch(html, /DEMO-DIE-2026|Granada →|267,79/);
  assert.doesNotMatch(html, /data-dietas-formulario|data-dietas-enviar/);
});

test("separa capacidad de gastos y rutas sin filtrar localizaciones", () => {
  const modelo = presentador([CAPACIDAD_CONSULTAR_GASTO]).obtenerModelo();
  assert.equal(modelo.comisiones.length, 5);
  assert.deepEqual(modelo.comisiones[0].ruta, []);
  assert.equal(modelo.comisiones[0].kilometros, null);
  assert.equal(modelo.resumen.kilometros, null);
  const html = renderizarDietas(modelo);
  assert.match(html, /Sin capacidad/);
  assert.doesNotMatch(html, /Albolote|Motril|Guadix|Loja|Baza/);
  assert.doesNotMatch(html, /data-dietas-formulario/);
});

test("resume comisiones, kilometraje, importes y pagos con capacidades explícitas", () => {
  const modelo = presentador().obtenerModelo();
  assert.deepEqual(modelo.resumen, {
    expedientes: 5, pendientes: 3, kilometros: 599, total_euros: 267.79, pagado_euros: 154.24,
  });
  assert.equal(modelo.historialMensual.length, 4);
  assert.equal(modelo.comisiones[0].ruta.join(" → "), "Granada → Albolote → Granada");
  assert.equal(modelo.seleccionada.historial.length, 3);
});

test("filtra mediante códigos canónicos independientes del idioma", () => {
  const modulo = presentador();
  assert.equal(modulo.filtrar({ estado: "pagada", texto: "Baza" }).comisiones.length, 1);
  assert.equal(modulo.filtrar({ estado: "todos", texto: "sin coincidencia" }).comisiones.length, 0);
  assert.throws(() => modulo.filtrar({ estado: "Pagada" }), /no permitido/);
});

test("el adaptador DEMO crea un borrador volátil con recibo del actor común", () => {
  const puerto = adaptador();
  const datos = puerto.ejecutar({
    tipo: "crear_borrador",
    campos: {
      fecha: "2026-07-20", motivo: "Visita técnica", origen: "Granada", destino: "Motril",
      kilometros: "140.8", manutencion_euros: "20", alojamiento_euros: "0", otros_gastos_euros: "5",
    },
  });
  const modelo = crearPresentadorDietas({
    datos, contextoActor: CONTEXTO_COMPARTIDO, capacidades: CAPACIDADES_EMPLEADO,
  }).obtenerModelo();
  assert.equal(modelo.resumen.expedientes, 6);
  assert.equal(modelo.comisiones[0].referencia, "DEMO-DIE-NUEVA-999");
  assert.equal(modelo.comisiones[0].kilometraje_euros, 36.61);
  assert.equal(modelo.comisiones[0].total_euros, 61.61);
  assert.deepEqual(modelo.ultimoRecibo, {
    referencia: "DEMO-REC-DIE-VOL-0001",
    operacion: "crear_borrador",
    objetivo: "DEMO-DIE-NUEVA-999",
    resultado: "borrador_creado_demo",
    actor_ref: CONTEXTO_COMPARTIDO.actor.actor_ref,
    instante: "2026-07-19T10:15:00.000Z",
    efectos_reales: false,
    persistencia: "memoria_volatil",
  });
  assert.throws(() => adaptador([]).ejecutar({ tipo: "enviar_validacion", referencia: "DEMO-DIE-2026-0091" }), /no tiene capacidad/);
});

test("el envío cambia códigos canónicos y conserva la trazabilidad", () => {
  const puerto = adaptador();
  const datos = puerto.ejecutar({ tipo: "enviar_validacion", referencia: "DEMO-DIE-2026-0091" });
  const item = datos.comisiones.find((comision) => comision.referencia === "DEMO-DIE-2026-0091");
  assert.equal(item.estado, "pendiente_jefatura");
  assert.equal(item.etapa_actual, 1);
  assert.equal(item.historial.at(-1).recibo, datos.ultimo_recibo.referencia);
  assert.equal(datos.ultimo_recibo.efectos_reales, false);
});

test("prepara PDF DEMO con logo y QR resoluble sin datos personales", () => {
  const descriptor = presentador().prepararDescriptorRecibo("DEMO-REC-DIE-0084-03", t);
  assert.equal(descriptor.formato, "pdf");
  assert.equal(descriptor.identidad_visual.logo_src, "/portal-empleado/assets/logo-diputacion-granada.svg");
  assert.equal(descriptor.comprobacion.qr_contenido, "https://vec.demo.dipgra.es/verificar/?ref=DEMO-REC-DIE-0084-03&presentacion=rrhh");
  assert.equal(descriptor.comprobacion.metodo, "consulta_estatica_demo");
  assert.equal(descriptor.comprobacion.contiene_datos_personales, false);
  assert.doesNotMatch(descriptor.comprobacion.qr_contenido, /Agente|DNI|nombre/i);
  assert.match(descriptor.marca, /SIN EFECTOS ADMINISTRATIVOS/);
});

test("acepta referencias opacas productivas, rechaza DEMO y prepara cotejo POST", () => {
  const modulo = crearPresentadorDietas({
    datos: datosProductivos(), contextoActor: CONTEXTO_PRODUCTIVO,
    capacidades: CAPACIDADES_EMPLEADO, origenComprobacion: "https://vec.dipgra.es",
  });
  const referencia = "recibo:dietas:0:2:2026";
  const descriptor = modulo.prepararDescriptorRecibo(referencia, t);
  assert.equal(descriptor.marca, "");
  assert.equal(descriptor.comprobacion.metodo, "post_servicio_cotejo");
  assert.equal(descriptor.comprobacion.qr_contenido, "https://vec.dipgra.es/verificar/?ref=recibo%3Adietas%3A0%3A2%3A2026");

  const datosConDemo = datosProductivos();
  datosConDemo.comisiones[0].historial[0].recibo = "DEMO-REC-DIE-PROHIBIDO-01";
  const moduloConDemo = crearPresentadorDietas({
    datos: datosConDemo, contextoActor: CONTEXTO_PRODUCTIVO, capacidades: CAPACIDADES_EMPLEADO,
  });
  assert.throws(() => moduloConDemo.prepararDescriptorRecibo("DEMO-REC-DIE-PROHIBIDO-01", t), /no encontrado/);
});

test("renderiza un espacio administrativo accesible, denso y traducido", () => {
  const html = renderizarDietas(presentador(CAPACIDADES_COMPLETAS).obtenerModelo(), {
    descargaDisponible: true, confirmacionDisponible: true,
  });
  assert.match(html, /Portal del Empleado → Dietas/);
  assert.match(html, /Agente interno DEMO/);
  assert.match(html, /Circuito de aprobación/);
  assert.match(html, /Desglose de gastos/);
  assert.match(html, /Historial y trazabilidad/);
  assert.match(html, /<caption>/);
  assert.match(html, /<th scope="col">/);
  assert.match(html, /data-dietas-descargar-recibo="DEMO-REC-DIE-0084-03"/);
  assert.match(html, /aria-current="true"/);
  assert.doesNotMatch(html, /aria-selected=/);
  assert.doesNotMatch(html, /onclick=|javascript:/i);
});

test("permite sustituir todo el catálogo de interfaz sin cambiar estados", () => {
  const alternativo = { ...MENSAJES_DIETAS_ES, titulo: "Expense workspace", buscar: "Search" };
  const html = renderizarDietas(presentador().obtenerModelo(), { mensajes: alternativo });
  assert.match(html, /Expense workspace/);
  assert.match(html, />Search<input/);
  assert.match(html, /value="pagada"/);
  assert.doesNotMatch(html, />Mis dietas y comisiones de servicio</);
  assert.throws(() => crearTraductorDietas({ titulo: "incompleto" }), /incompleto/);
});

test("la vista final no importa fixtures y gobierna concurrencia e interfaces por puertos", async () => {
  const fuentes = await Promise.all([
    "contrato.js", "datos-presentacion.js", "adaptador-presentacion.js", "presentador.js", "vista.js", "i18n.js",
  ].map((archivo) => readFile(new URL(archivo, import.meta.url), "utf8")));
  const codigo = fuentes.join("\n");
  assert.doesNotMatch(codigo, /\bfetch\s*\(|XMLHttpRequest|WebSocket|\.cookie\b|localStorage|sessionStorage|indexedDB/);
  assert.doesNotMatch(codigo, /leaflet|openstreetmap|google\s*maps|mapbox/i);
  const vista = fuentes[4];
  assert.doesNotMatch(vista, /datos-presentacion|adaptador-presentacion/);
  assert.doesNotMatch(vista, /Visita técnica|2026-07-20|value="Granada"|value="Motril"|value="140\.8"/);
  assert.match(vista, /let ocupado = false/);
  assert.match(vista, /conBloqueo/);
  assert.match(vista, /aria-busy/);
  assert.match(vista, /confirmarOperacion/);
  assert.match(vista, /descargarRecibo/);
});

test("la hoja de estilos conserva densidad, foco visible y adaptación móvil", async () => {
  const css = await readFile(new URL("dietas.css", import.meta.url), "utf8");
  assert.match(css, /:focus-visible/);
  assert.match(css, /@media \(max-width: 1180px\)/);
  assert.match(css, /@media \(max-width: 720px\)/);
  assert.match(css, /@media \(max-width: 420px\)/);
  assert.match(css, /overflow-x|tabla-contenedor/);
  assert.match(css, /prefers-reduced-motion/);
  assert.doesNotMatch(css, /font-family\s*:/);
});
