import assert from "node:assert/strict";
import test from "node:test";
import { descargarJustificanteRenuncia, renderizarAccionJustificanteRenuncia, renderizarSeguimientoLlamamiento } from "./portal-bolsas-seguimiento.js";

const bolsa = "bolsa:administrativo:01";
const participacion = "participacion:antonio:01";
const llamamiento = "llamamiento:01";
const candidato = { participacion_ref: participacion, nombre_visible: "Antonio Reyes Álvarez", orden: 1,
  estado_clave: "renuncia" };
const telefono = { canal: "telefono", seguimiento: true, resultados: ["acepta", "rechaza"], resultados_cierre: ["acepta", "rechaza"] };
const html = (texto) => String(texto).replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;");

function vista(respuesta, persona = candidato) {
  return renderizarSeguimientoLlamamiento({
    estadoCandidatos: { carga: "listo", datos: { bolsa: { categoria: "Administrativo" }, candidatos: [persona],
      contactos: [], canales_llamamiento: [telefono], respuestas_portal: {
        llamamiento_ref: llamamiento, respuestas: [{ participacion_ref: participacion, respuesta }],
      } } },
    seguimiento: { bolsa_ref: bolsa, llamamiento_ref: llamamiento },
    encabezadoVista: (_miga, titulo) => `<header>${html(titulo)}</header>`, escaparHTML: html,
    numero: String, instanteVisible: String, renderizarFicha: () => "", modalFicha: null,
  });
}

test("solo la respuesta de renuncia justificada ofrece el PDF, sin pedirlo al renderizar", () => {
  const justificada = vista("renuncia_justificada");
  assert.match(justificada, /data-bolsa-accion="descargar-justificante-renuncia"/u);
  assert.match(justificada, /data-bolsa-ref="bolsa:administrativo:01"/u);
  assert.match(justificada, /data-participacion-ref="participacion:antonio:01"/u);
  assert.match(justificada, /aria-describedby="justificante-/u);
  assert.match(justificada, /aria-label="Descargar el justificante de Antonio Reyes Álvarez"/u);
  assert.match(justificada, /Descargar justificante/u);
  assert.doesNotMatch(vista("renuncia"), /descargar-justificante-renuncia/u);
  const escapada = vista("renuncia_justificada", { ...candidato, nombre_visible: 'Ana "Contreras" <RRHH> & familia' });
  assert.match(escapada, /aria-label="Descargar el justificante de Ana &quot;Contreras&quot; &lt;RRHH&gt; &amp; familia"/u);
  assert.doesNotMatch(escapada, /aria-label="[^"]*<RRHH>/u);
});

function control() {
  const estado = { textContent: "" };
  return { dataset: { bolsaRef: bolsa, participacionRef: participacion }, disabled: false,
    closest: () => ({ querySelector: () => estado }), estado };
}

test("un error de permiso queda junto al botón y permite reintentar", async () => {
  const boton = control();
  assert.equal(await descargarJustificanteRenuncia(boton, { fetchImpl: async () => ({ status: 403 }) }), false);
  assert.match(boton.estado.textContent, /No tiene permiso/u);
  assert.equal(boton.disabled, false);
});

test("un fallo temporal conserva la fila y explica cómo reintentar", async () => {
  const boton = control();
  assert.equal(await descargarJustificanteRenuncia(boton, { fetchImpl: async () => ({ status: 503 }) }), false);
  assert.match(boton.estado.textContent, /Inténtelo de nuevo/u);
  assert.equal(boton.disabled, false);
});

test("la ficha ofrece consulta diferida y explica si el justificante no existe", async () => {
  const ficha = renderizarAccionJustificanteRenuncia({ bolsaRef: bolsa, participacionRef: participacion,
    nombre: candidato.nombre_visible, escaparHTML: html, ficha: true });
  assert.match(ficha, /Ver justificante de renuncia/u);
  assert.match(ficha, /data-bolsa-accion="descargar-justificante-renuncia"/u);
  const boton = control();
  assert.equal(await descargarJustificanteRenuncia(boton, { fetchImpl: async () => ({ status: 404 }) }), false);
  assert.match(boton.estado.textContent, /No hay un justificante/u);
});

test("descarga los bytes originales al pulsar y revoca la URL", async () => {
  const boton = control();
  const eventos = [];
  let blob;
  const enlace = { click() { eventos.push("click"); }, remove() { eventos.push("remove"); } };
  const documento = { body: { append() { eventos.push("append"); } }, createElement: () => enlace,
    defaultView: { Blob, URL: { createObjectURL(archivo) { blob = archivo; return "blob:pdf"; }, revokeObjectURL() {} } } };
  const bytes = new TextEncoder().encode("%PDF-1.7\noriginal");
  let ruta;
  assert.equal(await descargarJustificanteRenuncia(boton, { documento, fetchImpl: async (url, opciones) => {
    ruta = url;
    assert.equal(opciones.credentials, "same-origin");
    return new Response(bytes, { status: 200, headers: { "Content-Type": "application/pdf",
      "Content-Disposition": 'attachment; filename="justificante.pdf"' } });
  } }), true);
  assert.match(ruta, /bolsa=bolsa%3Aadministrativo%3A01&participacion=participacion%3Aantonio%3A01/u);
  assert.deepEqual(eventos, ["append", "click", "remove"]);
  assert.equal(enlace.download, "justificante.pdf");
  assert.deepEqual(await blob.arrayBuffer(), bytes.buffer);
  assert.equal(boton.estado.textContent, "");
});
