import assert from "node:assert/strict";
import test from "node:test";

import {
  CAPACIDAD_CONSULTAR_FICHAJES,
  CAPACIDAD_CONSULTAR_HORARIO,
  validarDatosCronos,
} from "./contrato.js";
import { renderizarJornadaCronos, validarSeleccionPeriodoCronos } from "./vista.js";
import { MENSAJES_CRONOS_ES } from "./i18n.js";
import { ESQUEMA_CONTEXTO_ACTOR_FRONTEND, validarYCongelarContextoActor } from "../../identidad/contexto-actor.js";

<<<<<<< HEAD
const directorio = new URL("./", import.meta.url);
const [css, presentadorFuente, vistaFuente, datosFuente, adaptadorFuente] = await Promise.all([
  readFile(new URL("cronos.css", directorio), "utf8"),
  readFile(new URL("presentador.js", directorio), "utf8"),
  readFile(new URL("vista.js", directorio), "utf8"),
  readFile(new URL("datos-presentacion.js", directorio), "utf8"),
  readFile(new URL("adaptador-presentacion.js", directorio), "utf8"),
]);

const TODAS_LAS_CAPACIDADES = [
  CAPACIDAD_CONSULTAR_FICHAJES,
  CAPACIDAD_REGISTRAR_FICHAJE,
  CAPACIDAD_CONSULTAR_HORARIO,
  CAPACIDAD_CONSULTAR_PERMISOS,
  CAPACIDAD_SOLICITAR_PERMISO,
];

test("Jornada sin servicio muestra estados honestos y no expone cifras ni acciones", () => {
  for (const estado of ["cargando", "vacio", "no_configurado", "denegado", "error"]) {
    const html = renderizarJornadaCronos({ estado });
    assert.match(html, new RegExp(`data-estado="${estado}"`));
    assert.doesNotMatch(html, /07:36|DEMO-REC-FIC-1900|tabla-cronos-jornada/);
    assert.ok((html.match(/disabled aria-disabled="true"/g) || []).length === 2);
  }
  assert.match(renderizarJornadaCronos({ estado: "cargando" }), /aria-busy="true"/);
  assert.throws(() => renderizarJornadaCronos({ estado: "inventado" }), /estado de jornada/);
});

test("Jornada permite elegir escala y rango, y distingue un periodo sin proyección", () => {
  const inicial = renderizarJornadaCronos();
  for (const tipo of ["dia", "semana", "mes", "anio", "periodo"]) {
    assert.match(inicial, new RegExp(`value="${tipo}"`));
  }
  assert.match(inicial, /data-cronos-form-periodo/);
  assert.match(inicial, /Fecha de referencia/);
  const contexto = contextoReal();
  const seleccion = validarSeleccionPeriodoCronos({ tipo: "semana", desde: "2026-09-24" });
  const html = renderizarJornadaCronos({
    estado: "disponible", contextoActor: contexto, datos: datosRealesPara(contexto),
    capacidades: [CAPACIDAD_CONSULTAR_HORARIO, CAPACIDAD_CONSULTAR_FICHAJES], seleccion,
  });
  assert.match(html, /Semana del 21\/09\/2026 al 27\/09\/2026/);
  assert.match(html, /No hay datos de jornada para el periodo seleccionado/);
  assert.match(html, /Jornada teórica.*Tiempo trabajado.*Permisos.*Saldo/s);
  assert.doesNotMatch(html, /tabla-cronos-jornada|Jornada diaria|08:01|DEMO-REC-FIC-1900/);
  for (const [tipo, esperado] of [
    ["dia", "Día 24/09/2026"], ["mes", "Mes de septiembre de 2026"], ["anio", "Año 2026"],
  ]) {
    assert.match(renderizarJornadaCronos({ seleccion: validarSeleccionPeriodoCronos({ tipo, desde: "2026-09-24" }) }),
      new RegExp(esperado));
  }
  const denegado = renderizarJornadaCronos({ estado: "denegado", seleccion });
  assert.match(denegado, /Acceso denegado/);
  assert.doesNotMatch(denegado, /No hay datos de jornada para el periodo|data-cronos-form-periodo/);
});

test("La selección rechaza fechas imposibles y rangos invertidos", () => {
  assert.throws(() => validarSeleccionPeriodoCronos({ tipo: "dia", desde: "2026-02-30" }), /fecha/);
  assert.throws(() => validarSeleccionPeriodoCronos({ tipo: "periodo", desde: "2026-09-25", hasta: "2026-09-24" }), /rango/);
  assert.deepEqual(validarSeleccionPeriodoCronos({ tipo: "periodo", desde: "2026-09-24", hasta: "2026-09-24" }),
    { tipo: "periodo", desde: "2026-09-24", hasta: "2026-09-24" });
});

test("El formulario aplica la selección y retira sus oyentes al desmontar", () => {
  let contenedor;
  const raiz = {
    ownerDocument: { createElement: () => {
      contenedor = { dataset: {}, innerHTML: "", oyentes: new Map(),
        addEventListener(tipo, fn) { this.oyentes.set(tipo, fn); },
        removeEventListener(tipo) { this.oyentes.delete(tipo); },
        remove() {},
      };
      return contenedor;
    } },
    append() {},
  };
  const anuncios = [];
  const vista = montarJornadaCronos({ raiz, anunciar: (texto) => anuncios.push(texto) });
  let focoHasta = false;
  const controles = new Map(Object.entries({ tipo: { value: "periodo" }, desde: { value: "2026-09-24" },
    hasta: { value: "2026-09-23", focus() { focoHasta = true; } } }));
  let prevenido = false;
  const error = { hidden: true, textContent: "" };
  const formulario = { matches: () => true, elements: { namedItem: (nombre) => controles.get(nombre) },
    querySelector: () => error };
  contenedor.oyentes.get("submit")({ target: formulario, preventDefault() { prevenido = true; } });
  assert.equal(focoHasta, true);
  assert.equal(error.hidden, false);
  assert.match(error.textContent, /Revise las fechas/);
  controles.get("hasta").value = "2026-09-25";
  contenedor.oyentes.get("submit")({ target: formulario, preventDefault() { prevenido = true; } });
  assert.equal(prevenido, true);
  assert.match(contenedor.innerHTML, /Periodo del 24\/09\/2026 al 25\/09\/2026/);
  assert.match(anuncios.at(-1), /Periodo del 24\/09\/2026 al 25\/09\/2026/);
  controles.get("hasta").value = "2026-09-26";
  contenedor.oyentes.get("submit")({ target: formulario, preventDefault() {} });
  assert.match(contenedor.innerHTML, /Periodo del 24\/09\/2026 al 26\/09\/2026/);
  vista.desmontar();
  assert.equal(contenedor.oyentes.size, 0);
});

test("La selección conserva la denegación de una proyección real sin capacidades", () => {
  const contexto = contextoReal();
  const datos = datosRealesPara(contexto);
  const seleccion = validarSeleccionPeriodoCronos({ tipo: "semana", desde: "2026-09-24" });
  const proyeccion = { estado: "disponible", contextoActor: contexto, datos, capacidades: [] };
  const html = renderizarJornadaCronos({ ...proyeccion, seleccion });
  assert.match(html, /data-estado="denegado"/);
  assert.match(html, /Acceso denegado/);
  assert.match(html, /data-cronos-jornada-resultado tabindex="-1"/);
  assert.doesNotMatch(html, /Servicio pendiente|No hay datos de jornada para el periodo|data-cronos-form-periodo/);

  let contenedor;
  let focos = 0;
  let focoDenegado = 0;
  const documento = { activeElement: {}, createElement: () => {
    contenedor = { dataset: {}, innerHTML: "", oyentes: new Map(),
      addEventListener(tipo, fn) { this.oyentes.set(tipo, fn); },
      removeEventListener(tipo) { this.oyentes.delete(tipo); },
      contains: () => true,
      querySelector(selector) {
        if (selector.includes("data-cronos-periodo-resultado") && this.innerHTML.includes("data-cronos-periodo-resultado")) {
          return { focus() { focos += 1; } };
        }
        if (selector.includes("data-cronos-jornada-resultado") && this.innerHTML.includes('data-estado="denegado"')) {
          return { focus() { focoDenegado += 1; } };
        }
        return null;
      },
      remove() {},
    };
    return contenedor;
  } };
  const raiz = { ownerDocument: documento, append() {} };
  const anuncios = [];
  const vista = montarJornadaCronos({ raiz, anunciar: (mensaje) => anuncios.push(mensaje) });
  const controles = new Map(Object.entries({ tipo: { value: "semana" }, desde: { value: "2026-09-24" }, hasta: { value: "" } }));
  const formulario = { matches: () => true, elements: { namedItem: (nombre) => controles.get(nombre) } };
  contenedor.oyentes.get("submit")({ target: formulario, preventDefault() {} });
  assert.ok(focos >= 1);
  vista.actualizar(proyeccion);
  assert.match(contenedor.innerHTML, /data-estado="denegado"/);
  assert.equal(focoDenegado, 1);
  assert.equal(anuncios.at(-1), "Acceso denegado");
  vista.desmontar();
});

test("Jornada disponible exige proyección propia y oculta bloques sin capacidad", () => {
  const contexto = contextoReal();
  const datos = datosRealesPara(contexto);
  const contextoDemo = contextoPresentacion();
  assert.match(renderizarJornadaCronos({ estado: "disponible", contextoActor: contexto, capacidades: [CAPACIDAD_CONSULTAR_FICHAJES] }), /data-estado="no_configurado"/);
  assert.match(renderizarJornadaCronos({
    estado: "disponible", contextoActor: contextoDemo,
    datos: crearDatosCronosPresentacion(contextoDemo), capacidades: [CAPACIDAD_CONSULTAR_FICHAJES],
  }), /data-estado="no_configurado"/);
  const soloHorario = renderizarJornadaCronos({
    estado: "disponible", contextoActor: contexto, datos,
    capacidades: [CAPACIDAD_CONSULTAR_HORARIO],
  });
  assert.match(soloHorario, /Datos de la consulta recibida/);
  assert.match(soloHorario, /La sesión no tiene capacidad para consultar fichajes/);
  assert.doesNotMatch(soloHorario, /DEMO-REC-FIC-1900|08:01|tabla-cronos-jornada/);
  const todo = renderizarJornadaCronos({
    estado: "disponible", contextoActor: contexto, datos,
    capacidades: [CAPACIDAD_CONSULTAR_HORARIO, CAPACIDAD_CONSULTAR_FICHAJES],
  });
  assert.match(todo, /tabla-cronos-jornada/);
  assert.match(todo, /Jornada diaria/);
  assert.match(todo, /Calendario laboral/);
  assert.match(todo, /Sin calendario autorizado para el centro y el periodo/);
  assert.doesNotMatch(todo, /cronos-mini-calendario|data-cronos-control|data-cronos-detalle/);
  assert.match(todo, /data-accion="ayuda"/);
  assert.doesNotMatch(todo, /data-cronos-accion="registrar-fichaje"/);
  const inyectado = renderizarJornadaCronos({
    estado: "disponible", contextoActor: contexto,
    datos: { ...datos, perfil_jornada: { ...datos.perfil_jornada, nombre: "<script>alert(1)</script>" } },
    capacidades: [CAPACIDAD_CONSULTAR_HORARIO],
  });
  assert.match(inyectado, /&lt;script&gt;alert\(1\)&lt;\/script&gt;/);
  assert.doesNotMatch(inyectado, /<script>/);
  assert.throws(() => renderizarJornadaCronos({
    estado: "disponible", contextoActor: contexto, datos: { ...datos, actor_ref: "REAL-OTRO-ACTOR" },
    capacidades: [CAPACIDAD_CONSULTAR_FICHAJES],
  }), /no pertenecen/);
});

test("Jornada se monta y desmonta sin listeners ni efectos", () => {
  const montajes = [];
  const raiz = {
    ownerDocument: { createElement: () => ({ dataset: {}, innerHTML: "", remove() { this.eliminado = true; } }) },
    append: (elemento) => montajes.push(elemento),
  };
  let desmontarRegistrado;
  const anuncios = [];
  const vista = montarJornadaCronos({ raiz, anunciar: (texto) => anuncios.push(texto), registrarDesmontar: (desmontar) => { desmontarRegistrado = desmontar; } });
  assert.equal(montajes.length, 1);
  assert.match(montajes[0].innerHTML, /Servicio pendiente/);
  vista.actualizar({ estado: "error" });
  assert.match(montajes[0].innerHTML, /Consulta no disponible/);
  vista.actualizar({ estado: "disponible" });
  assert.match(montajes[0].innerHTML, /data-estado="no_configurado"/);
  assert.equal(anuncios.at(-1), "Servicio pendiente");
  desmontarRegistrado();
  assert.equal(montajes[0].eliminado, true);
  assert.throws(() => vista.actualizar({ estado: "cargando" }), /desmontada/);
});

test("Jornada inserta el calendario civil y retira sus oyentes al actualizar y salir", () => {
  const creados = [];
  const destino = { ownerDocument: null, append(nodo) { this.ultimo = nodo; } };
  const documento = { createElement() {
    const nodo = { dataset: {}, innerHTML: "", oyentes: new Map(), addEventListener(tipo, fn) { this.oyentes.set(tipo, fn); },
      removeEventListener(tipo) { this.oyentes.delete(tipo); }, remove() { this.eliminado = true; },
      querySelector(selector) { return selector === "[data-cronos-calendario-raiz]" ? destino : null; } };
    creados.push(nodo);
    return nodo;
  } };
  destino.ownerDocument = documento;
  const vista = montarJornadaCronos({ raiz: { ownerDocument: documento, append() {} } });
  assert.match(creados[1].innerHTML, /Calendario civil/u);
  assert.match(creados[1].innerHTML, /Calendario laboral no configurado/u);
  assert.equal(creados[1].oyentes.size, 2);
  vista.actualizar({ estado: "error" });
  assert.equal(creados[1].eliminado, true);
  assert.equal(creados[1].oyentes.size, 0);
  assert.match(creados[2].innerHTML, /Calendario civil/u);
  vista.desmontar();
  assert.equal(creados[2].eliminado, true);
  assert.equal(creados[2].oyentes.size, 0);
});

function contextoPresentacion() {
  return crearContextoActorPresentacionDesdeSesion({
    actor_ref: "DEMO-PERFIL-FUNCIONARIO-01",
    iniciales: "FU",
    nombre: "Funcionario DEMO 01",
    perfil: "Funcionario · autoservicio interno DEMO",
  });
}

function contextoReal() {
=======
function contexto() {
>>>>>>> b6ae6c62 (Cronos: retira presentación volátil y catálogo compilado)
  return validarYCongelarContextoActor({
    esquema: ESQUEMA_CONTEXTO_ACTOR_FRONTEND,
    revision: 1,
    demostracion: false,
    persona_ref: "per_persona_sintetica_000001",
    cuenta_ref: "cta_cuenta_sintetica_000001",
    perfil_ref: "prf_perfil_sintetico_000001",
    actor: { actor_ref: "act_actor_sintetico_000001", nombre_visible: "Persona sintética", iniciales: "PS" },
    rol: { clave: "personal_interno", etiqueta: "Personal interno" },
    ambito: {
      clase: "personal_interno",
      organizacion_ref: "org_organizacion_sintetica_000001",
      unidad_ref: "uni_unidad_sintetica_000001",
      modulos: ["cronos"],
    },
    autenticacion: { sesion_ref: "ses_sesion_sintetica_000001", metodo: "kerberos_ad", garantia: "alto" },
    resuelto_en: "2026-09-24T09:00:00.000Z",
  });
}

function datos(actor) {
  return {
    esquema: "vec.cronos.area-personal.v1",
    actor_ref: actor.actor.actor_ref,
    periodo: "Septiembre de 2026",
    actualizado_en: "2026-09-24T09:00:00.000Z",
    perfil_jornada: { nombre: "Jornada ordinaria", jornada_diaria: "07:30", ventana_entrada: "07:00–09:00", tramo_obligatorio: "09:00–14:00" },
    resumen: { teoricas_hoy: "07:30", trabajadas_hoy: "02:00", saldo_hoy: "-05:30", saldo_periodo: "+01:00" },
    fichajes: [{ id: "fic_marcaje_sintetico_000001", actor_ref: actor.actor.actor_ref,
      instante: "2026-09-24T07:00:00.000Z", tipo_clave: "entrada", canal: "Terminal",
      modalidad: "Presencial", estado_clave: "registrado", recibo_ref: "rec_marcaje_sintetico_000001" }],
    saldos: [], solicitudes: [], historial: [],
  };
}

test("Jornada sin fuente muestra estados honestos y no ofrece fichaje genérico", () => {
  for (const estado of ["cargando", "vacio", "no_configurado", "denegado", "error"]) {
    const html = renderizarJornadaCronos({ estado });
    assert.match(html, new RegExp(`data-estado="${estado}"`));
    assert.doesNotMatch(html, /tabla-cronos-jornada|02:00/);
    assert.doesNotMatch(html, /data-cronos-fichar|jornada-acciones/);
  }
  assert.match(renderizarJornadaCronos({ estado: "disponible" }), /data-estado="no_configurado"/);
  assert.throws(() => renderizarJornadaCronos({ estado: "inventado" }), /estado de jornada/);
});

test("la consulta propia exige actor y capacidad explícita", () => {
  const actor = contexto();
  const proyeccion = datos(actor);
  const html = renderizarJornadaCronos({ estado: "disponible", contextoActor: actor,
    capacidades: [CAPACIDAD_CONSULTAR_FICHAJES, CAPACIDAD_CONSULTAR_HORARIO], datos: proyeccion });
  assert.match(html, /data-estado="disponible"/);
  assert.match(html, /tabla-cronos-jornada/);
  assert.match(html, /Terminal/);
  assert.match(html, /aria-label="(Abrir ayuda sobre [^"]+)" title="\1"><span aria-hidden="true">\?<\/span><\/button>/);
  assert.doesNotMatch(html, /cronos-boton-ayuda-texto/);
  for (const clave of ["jornada_descripcion", "jornada_periodo_descripcion",
    "jornada_movimientos_detalle", "horario_descripcion", "jornada_calendario_fuente"]) {
    assert.equal(html.includes(MENSAJES_CRONOS_ES[clave]), false, clave);
  }
  assert.match(renderizarJornadaCronos({ estado: "disponible", contextoActor: actor,
    capacidades: [], datos: proyeccion }), /data-estado="denegado"/);
  assert.throws(() => validarDatosCronos({ ...proyeccion, actor_ref: "act_otro_actor_sintetico_000001" }, actor), /no pertenecen/);
  assert.throws(() => validarDatosCronos({ ...proyeccion, fichajes: [
    { ...proyeccion.fichajes[0], actor_ref: "act_otro_actor_sintetico_000001" },
  ] }, actor), /registros ajenos/);
  assert.throws(() => validarDatosCronos({ ...proyeccion, saldos: [{ id: "sal_otro_000001", estado_clave: "estado_desconocido" }] }, actor), /contrato cerrado/);
});

test("la selección civil rechaza fechas imposibles y no finge un saldo", () => {
  assert.throws(() => validarSeleccionPeriodoCronos({ tipo: "dia", desde: "2026-02-30" }), /fecha/);
  assert.throws(() => validarSeleccionPeriodoCronos({ tipo: "periodo", desde: "2026-09-25", hasta: "2026-09-24" }), /rango/);
  const seleccion = validarSeleccionPeriodoCronos({ tipo: "semana", desde: "2026-09-24" });
  const actor = contexto();
  const html = renderizarJornadaCronos({ estado: "disponible", contextoActor: actor,
    capacidades: [CAPACIDAD_CONSULTAR_HORARIO], datos: datos(actor), seleccion });
  assert.match(html, /Semana del 21\/09\/2026 al 27\/09\/2026/);
  assert.match(html, /Falta una proyección autorizada para el periodo seleccionado/);
  assert.doesNotMatch(html, /tabla-cronos-jornada/);
});
