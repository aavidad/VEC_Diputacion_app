import assert from "node:assert/strict";
import test from "node:test";
import {
  crearCoordinadorModulosPortal,
  moduloDeVistaPortal,
  rutaDeVistaPortal,
} from "./portal-modulos-coordinador.js";

// Documentos (servicio común) se compone como los demás módulos conectados:
// solo si /api/vec/modules lo publica y su vista carga; nunca con la sección
// «documentos» de Bolsa.
test("Documentos se ofrece solo si el catálogo lo publica y su vista carga", async () => {
  const montajes = [];
  const peticiones = [];
  const entorno = { fetch: async (ruta, opciones) => { peticiones.push({ ruta, opciones }); return new Response("{}", { status: 403 }); } };
  const cargadorDocumentos = async () => Object.freeze({
    vista: { montarVistaDocumentos: (opciones) => { montajes.push(opciones); return Object.freeze({ desmontar() {} }); } },
    cliente: { crearFuenteDocumentosHTTP: ({ fetchImpl }) => Object.freeze({ fetchImpl }) },
  });
  const sinDocumentos = crearCoordinadorModulosPortal({
    escaparHTML: String, entorno,
    cargarCatalogoInterno: async () => Object.freeze([{ clave: "personal" }]),
    cargadoresInternos: { contratacion_temporal: async () => { throw new Error("sin CT"); }, documentos: cargadorDocumentos },
  });
  await sinDocumentos.cargarInterno();
  assert.equal(sinDocumentos.vistaDisponible("documentos-expediente"), false);
  assert.deepEqual(sinDocumentos.resolverAcceso("documentos"), { disponible: false, vista: "", estado: "no_disponible" });
  peticiones.length = 0;

  const conDocumentos = crearCoordinadorModulosPortal({
    escaparHTML: String, entorno,
    cargarCatalogoInterno: async () => Object.freeze([{ clave: "documentos" }]),
    cargadoresInternos: { contratacion_temporal: async () => { throw new Error("sin CT"); }, documentos: cargadorDocumentos },
  });
  await conDocumentos.cargarInterno();
  assert.equal(conDocumentos.vistaGestionada("documentos-expediente"), true);
  assert.equal(conDocumentos.vistaDisponible("documentos-expediente"), true);
  assert.deepEqual(conDocumentos.resolverAcceso("documentos"), { disponible: true, vista: "documentos-expediente" });
  assert.equal(moduloDeVistaPortal("documentos-expediente"), "documentos");
  assert.equal(rutaDeVistaPortal("documentos-expediente"), "#documentos-expediente");
  // La sección «documentos» de Bolsa no se confunde con el servicio común.
  assert.notEqual(moduloDeVistaPortal("documentos"), "documentos");
  const raiz = { replaceChildren() {}, innerHTML: "" };
  assert.equal(await conDocumentos.montarVista("documentos-expediente", raiz, { expedienteRef: "ref:" + "2".repeat(64) }), true);
  assert.equal(montajes.length, 1);
  assert.equal(montajes[0].expedienteRef, "ref:" + "2".repeat(64));
  assert.equal(typeof montajes[0].fuente.fetchImpl, "function");

  const fallida = crearCoordinadorModulosPortal({
    escaparHTML: String, entorno,
    cargarCatalogoInterno: async () => Object.freeze([{ clave: "documentos" }]),
    cargadoresInternos: { contratacion_temporal: async () => { throw new Error("sin CT"); },
      documentos: async () => { throw new Error("módulo roto"); } },
  });
  await fallida.cargarInterno();
  assert.equal(fallida.vistaDisponible("documentos-expediente"), false);
  assert.deepEqual(fallida.resolverAcceso("documentos"), { disponible: false, vista: "", estado: "no_disponible" });
  assert.equal(peticiones.length, 0, "cargar el módulo no consulta datos");
});
