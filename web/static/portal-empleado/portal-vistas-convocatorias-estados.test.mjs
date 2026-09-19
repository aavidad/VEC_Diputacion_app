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
