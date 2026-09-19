import assert from "node:assert/strict";
import test from "node:test";
import { obtenerDatosPresentacion } from "./datos-presentacion.js";
import { crearUtilidadesVista } from "./portal-vistas-utilidades.js";
import { crearVistasOperaciones } from "./portal-vistas-operaciones.js";

function utilidades() {
  return crearUtilidadesVista({
    escaparHTML: (valor) => String(valor),
    numero: (valor) => String(valor ?? 0),
    claseEstado: () => "neutro",
    encabezadoVista: (_sobrelinea, titulo, descripcion, acciones = "") => `<h2>${titulo}</h2><p>${descripcion}</p>${acciones}`,
    esPresentacion: () => true,
    operacionPermitida: () => true,
  });
}

test("las vistas operativas rotulan el recorrido DEMO sin alterar sus comandos", () => {
  const vistas = crearVistasOperaciones(utilidades());
  const datos = obtenerDatosPresentacion();
  const llamamientos = vistas.renderizarLlamamientos(datos);
  const contratos = vistas.renderizarContratos(datos);
  const documentos = vistas.renderizarDocumentos(datos);
  const comunicaciones = vistas.renderizarComunicaciones(datos);
  assert.match(llamamientos, /Llamamientos DEMO/);
  assert.match(llamamientos, /no fijan una regla ni un plazo operativo/);
  assert.match(contratos, /no acredita relación jurídica, cese ni incorporación/);
  assert.match(documentos, /La autenticación no firma documentos y ningún borrador es oficial/);
  assert.match(documentos, /Firmar DEMO · sin firma legal/);
  assert.match(comunicaciones, /Un aviso no acredita envío, entrega, notificación ni acuse/);
});
