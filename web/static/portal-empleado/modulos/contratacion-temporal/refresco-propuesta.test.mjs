import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import vm from "node:vm";
import test from "node:test";

// Ejecuta el coordinador real con consultas controladas, sin servidor ni POST.
const fuente = readFileSync(new URL("./vista-expedientes.js", import.meta.url), "utf8");
const inicio = fuente.indexOf("  async function refrescarDetalleTrasPropuesta(");
const fin = fuente.indexOf("\n  function montarLlamamiento(", inicio);
assert.ok(inicio >= 0 && fin > inicio);

test("reintenta GET tras cuadro avanzado y detalle fallido, sin perder el recibo", async () => {
 const panel = {}, recibo = { propuesta_ref: "propuesta:sintetica", version_resultante: 9 };
 const solicitud = { expediente_ref: "expediente:sintetico" };
 let estado = { vista: "expediente", expediente: { expediente_ref: solicitud.expediente_ref, version: 8 } };
 let lecturas = 0, repintados = 0, panelActual = panel;
 const contexto = vm.createContext({ montada: true, reciboPropuestaConfirmado: { recibo, solicitud, panel },
  raiz: { querySelector: () => panelActual }, mensajes: {}, anunciar() {},
  crearTraductorContratacionTemporal: () => (clave) => clave,
  repintar() { repintados++; },
  presentador: {
   obtenerEstado: () => estado,
   async cargar() { estado = { vista: "expediente", expediente: null,
    cuadro: { expedientes: [{ expediente_ref: solicitud.expediente_ref, version: 9 }] } }; },
   async seleccionarExpediente() { lecturas++; if (lecturas === 1) throw new Error("lectura interrumpida");
    estado.expediente = { expediente_ref: solicitud.expediente_ref, version: 9 }; },
  },
 });
 vm.runInContext(fuente.slice(inicio, fin), contexto);
 const refrescar = contexto.refrescarDetalleTrasPropuesta;
 assert.equal(await refrescar(recibo, solicitud), false);
 assert.equal(estado.expediente, null);
 assert.equal(await refrescar(recibo, solicitud), true);
 assert.equal(lecturas, 2);
 assert.equal(repintados, 1);
 assert.equal(contexto.reciboPropuestaConfirmado.recibo, recibo);
 panelActual = {};
 assert.equal(await refrescar(recibo, solicitud), false);
 assert.equal(lecturas, 2);
});
