import assert from "node:assert/strict";
import test from "node:test";
import { crearVistasConvocatorias } from "./portal-vistas-convocatorias.js";

function utilidades() {
  return {
    escaparHTML: (valor) => String(valor).replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll('"', "&quot;"),
    numero: (valor) => String(valor ?? 0),
    fecha: (valor) => String(valor ?? ""),
    chip: (valor) => `<span>${String(valor)}</span>`,
    tabla: ({ filas, vacio = "No hay registros para los filtros aplicados." }) => filas.length ? `<table>${filas.flat().join("")}</table>` : `<p>${vacio}</p>`,
    kpi: (_sigla, valor, etiqueta) => `<span>${valor} ${etiqueta}</span>`,
    encabezadoVista: (_sobrelinea, titulo, descripcion, acciones = "") => `<header><h2>${titulo}</h2><p>${descripcion}</p>${acciones}</header>`,
    avisoPresentacion: (texto) => `<aside>${texto}</aside>`,
    botonOperacion: (etiqueta, operacion) => `<button disabled aria-disabled="true" title="Capacidad de servidor no conectada" data-operacion="${operacion}">${etiqueta}</button>`,
    campo: (etiqueta, control) => `<label>${etiqueta}${control}</label>`,
    fuentePresentacion: () => "<span>Datos sintéticos</span>",
  };
}

const datos = Object.freeze({ elaboraciones: [], solicitudes: [] });
const vistas = crearVistasConvocatorias(utilidades());

test("convocatorias y admisión expresan carga, denegación, error y vacío sin habilitar efectos", () => {
  const carga = vistas.renderizarConvocatorias(datos, { fuenteLista: false });
  assert.match(carga, /Cargando convocatorias/);
  assert.match(carga, /role="status"/);
  assert.doesNotMatch(carga, /Circuitos pendientes/);
  const convocatoriasVacias = vistas.renderizarConvocatorias(datos, { fuenteLista: true });
  assert.match(convocatoriasVacias, /0 convocatorias encontradas/);
  assert.match(convocatoriasVacias, /— Circuitos pendientes · Fuente real no conectada/);

  const denegado = vistas.renderizarSolicitudes(datos, { fuenteLista: true, datosBolsas: { carga: "denegado" } });
  assert.match(denegado, /Acceso denegado/);
  assert.match(denegado, /no dispone de permiso/);
  const denegadoConFuentePendiente = vistas.renderizarConvocatorias(datos, { fuenteLista: false, datosBolsas: { carga: "denegado" } });
  assert.match(denegadoConFuentePendiente, /Acceso denegado/);
  assert.doesNotMatch(denegadoConFuentePendiente, /Cargando convocatorias/);

  const error = vistas.renderizarConvocatorias(datos, { fuenteLista: true, errorFuente: "La API interna no responde." });
  assert.match(error, /Error al cargar convocatorias/);
  assert.match(error, /La API interna no responde/);

  const vacio = vistas.renderizarSolicitudes(datos, { fuenteLista: true });
  assert.match(vacio, /0 solicitudes encontradas/);
  assert.match(vacio, /No hay registros para los filtros aplicados/);
  assert.match(vacio, /disabled aria-disabled="true"/);
  assert.match(vacio, /Capacidad de servidor no conectada/);
  assert.match(vacio, /data-operacion="publicar-lista-provisional"/);
  const conSolicitud = vistas.renderizarSolicitudes({ ...datos, solicitudes: [{ id: "SOL-1", persona_ref: "PER-1", convocatoria: "CNV-1", registrada: "hoy", requisitos: "1/1", subsanacion: "No", estado: "Pendiente de revisión" }] }, { fuenteLista: true });
  assert.match(conSolicitud, /data-operacion="admitir-solicitud"/);
});

test("la vista de elaboración no deduce circuitos de firmas generales ni de los expedientes", () => {
  const conExpedientes = {
    ...datos,
    indicadores: { firmas_pendientes: 73 },
    elaboraciones: [
      { id: "DEMO-1", nombre: "Convocatoria A", expediente: "EXP-1", fase: "Revisión", reglas: "v1", estado: "En revisión", responsable: "Unidad DEMO" },
      { id: "DEMO-2", nombre: "Convocatoria B", expediente: "EXP-2", fase: "Borrador", reglas: "v1", estado: "Borrador", responsable: "Unidad DEMO" },
    ],
  };
  const salidaPresentacion = vistas.renderizarConvocatorias(conExpedientes, { fuenteLista: true, modoPresentacion: true });
  assert.match(salidaPresentacion, /1 Borradores/);
  assert.match(salidaPresentacion, /1 En revisión/);
  assert.match(salidaPresentacion, /— Circuitos pendientes · Fuente real no conectada/);
  assert.doesNotMatch(salidaPresentacion, /2 Circuitos pendientes|73 Circuitos pendientes/);

  const salidaSinPresentacion = vistas.renderizarConvocatorias(conExpedientes, { fuenteLista: true, modoPresentacion: false });
  assert.match(salidaSinPresentacion, /— Circuitos pendientes · Fuente real no conectada/);
});
