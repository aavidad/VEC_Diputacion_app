import assert from "node:assert/strict";
import test from "node:test";
import { ErrorClienteCorreccionesCronos } from "./cliente-correcciones-http.js";
import { montarBandejaCorreccionesCronos, montarHistoriaCorreccionesPropiasCronos } from "./vista-correcciones-c5b.js";

const CLAVE = "123e4567-e89b-42d3-a456-426614174000";
const REF = `correccion:cronos:${CLAVE}`;
function raiz() {
  const nodo = { dataset: {}, innerHTML: "", eventos: {}, addEventListener(tipo, fn) { this.eventos[tipo] = fn; }, removeEventListener(tipo) { delete this.eventos[tipo]; }, remove() { this.retirado = true; }, querySelector() { return null; } };
  return { nodo, raiz: { ownerDocument: { createElement: () => nodo }, append() {} } };
}
const esperar = async () => { for (let i = 0; i < 8; i++) await Promise.resolve(); };
function pulsar(nodo, nombre, valor) {
  const boton = { disabled: false, dataset: valor ? { [nombre]: valor } : {}, hasAttribute: (v) => v === nombre };
  nodo.eventos.click({ target: { closest: (v) => v === "button" ? boton : null } });
}
function bandeja() { return { paso: "responsable", pendientes: [{ solicitud_ref: REF, empleado_ref: "emp_0123456789abcdefghijkl", empleado_etiqueta: "Persona ficticia", marcaje_original_ref: "", hueco_declarado: true,
  movimiento: "entrada", fecha_civil: "2026-10-14", hora_pretendida: "08:00", motivo_codigo: "olvido_marcaje", estado: "pendiente_responsable", version: 1, solicitada_en: "2026-10-14T08:00:00Z" }] }; }
function historia() { return { solicitud_ref: REF, empleado_ref: "emp_0123456789abcdefghijkl", estado: "pendiente_rrhh", version: 2, fecha_civil: "2026-10-14", hora_pretendida: "08:00", movimiento: "entrada", motivo_codigo: "olvido_marcaje",
  actuaciones: [
    { paso: "solicitud", resultado: "", motivo: "", actor_ref: "per_0123456789abcdefghijkl", perfil_ref: "prf_0123456789abcdefghijkl", estado: "pendiente_responsable", version: 1, recibo_ref: "recibo:cronos:inicial", instante_utc: "2026-10-14T08:00:00Z" },
    { paso: "decision_responsable", resultado: "favorable", motivo: "Olvido revisado", actor_ref: "per_0123456789abcdefghijkl", perfil_ref: "prf_0123456789abcdefghijkl", estado: "pendiente_rrhh", version: 2, recibo_ref: "recibo:cronos:decision", instante_utc: "2026-10-14T09:00:00Z" },
  ], asientos: [] }; }

test("jefatura revisa y confirma con motivo; un timeout bloquea cambios y reintenta material idéntico", async () => {
  const { nodo, raiz: root } = raiz(); const envios = []; let fallar = true;
  const vista = montarBandejaCorreccionesCronos({ raiz: root, cliente: {
    consultarBandeja: async () => bandeja(), consultarHistoria: async () => historia(),
    registrarActuacion: async (entrada) => { envios.push(entrada); if (fallar) { fallar = false; throw new ErrorClienteCorreccionesCronos("plazo_agotado"); }
      return { solicitud_ref: REF, actuacion_ref: "correccion:actuacion:1", recibo_ref: "recibo:cronos:decision", estado: "pendiente_rrhh", version: 2, instante_utc: "2026-10-14T09:00:00Z", replay: true }; },
  } });
  await esperar();
  pulsar(nodo, "c5bRevisar", REF);
  assert.match(nodo.innerHTML, /Persona ficticia/);
  nodo.eventos.submit({ target: { matches: () => true, elements: { namedItem: (n) => ({ value: n === "resultado" ? "favorable" : "Olvido revisado" }) } }, preventDefault() {} });
  assert.match(nodo.innerHTML, /Compruebe la actuación/);
  pulsar(nodo, "data-c5b-confirmar"); await esperar();
  assert.match(nodo.innerHTML, /No se pudo confirmar esta actuación/);
  assert.match(nodo.innerHTML, /name="motivo"[^>]*disabled/);
  pulsar(nodo, "data-c5b-cerrar");
  assert.match(nodo.innerHTML, /Reintentar la misma actuación/);
  pulsar(nodo, "data-c5b-reintentar"); await esperar();
  assert.equal(envios.length, 2); assert.deepEqual(envios[0], envios[1]);
  assert.match(nodo.innerHTML, /Esta actuación ya estaba registrada/);
  assert.match(nodo.innerHTML, /recibo:cronos:decision/);
  vista.desmontar();
});

test("historia propia sale de la lista propia y no ofrece selector de empleado", async () => {
  const { nodo, raiz: root } = raiz(); let consultaRef;
  const propia = historia(); propia.actuaciones = propia.actuaciones.map(({ actor_ref: _a, perfil_ref: _p, ...a }) => ({ ...a, actor_ref: "", perfil_ref: "" }));
  const vista = montarHistoriaCorreccionesPropiasCronos({ raiz: root, clienteSolicitudes: { consultarMovimientos: async (consulta) => ({ periodo: { tipo: consulta.periodo, desde: consulta.desde ?? "2026-01-01", hasta: consulta.hasta ?? "2026-12-31" }, calendario: { disponible: false, dias: [] }, marcajes_por_dia: [], absentismos: [], correcciones: [{ solicitud_ref: REF, fecha_civil: "2026-10-14", hora_pretendida: "08:00", movimiento: "entrada", estado: "pendiente_rrhh", version: 2, solicitada_en: "2026-10-14T08:00:00Z" }] }) },
    cliente: { consultarHistoria: async (ref, opciones) => { consultaRef = [ref, opciones.vista]; return propia; } } });
  await esperar(); assert.match(nodo.innerHTML, /Mis correcciones/);
  assert.doesNotMatch(nodo.innerHTML, /name="empleado/);
  pulsar(nodo, "c5bPropiaRef", REF); await esperar();
  assert.deepEqual(consultaRef, [REF, "propia"]);
  assert.match(nodo.innerHTML, /Actuaciones registradas/);
  assert.doesNotMatch(nodo.innerHTML, /per_0123456789abcdefghijkl/);
  vista.desmontar();
});

test("desmontar cancela una actuación y descarta la respuesta tardía", async () => {
  let resolver; const pendiente = new Promise((aceptar) => { resolver = aceptar; });
  const { nodo, raiz: root } = raiz(); let senal; let avisos = 0;
  const vista = montarBandejaCorreccionesCronos({ raiz: root, anunciar: () => { avisos++; }, cliente: {
    consultarBandeja: async () => bandeja(), consultarHistoria: async () => historia(),
    registrarActuacion: async (_entrada, { signal }) => { senal = signal; return pendiente; },
  } });
  await esperar(); pulsar(nodo, "c5bRevisar", REF);
  nodo.eventos.submit({ target: { matches: () => true, elements: { namedItem: (n) => ({ value: n === "resultado" ? "favorable" : "Olvido revisado" }) } }, preventDefault() {} });
  pulsar(nodo, "data-c5b-confirmar"); await esperar();
  const html = nodo.innerHTML; const previos = avisos; vista.desmontar();
  assert.equal(senal.aborted, true);
  resolver({ solicitud_ref: REF, actuacion_ref: "correccion:actuacion:1", recibo_ref: "recibo:cronos:1", estado: "pendiente_rrhh", version: 2, instante_utc: "2026-10-14T09:00:00Z", replay: false });
  await esperar(); assert.equal(nodo.innerHTML, html); assert.equal(avisos, previos);
});

test("tras perder el POST consulta la historia sin atribuirse una actuación por coincidencia", async () => {
  const { nodo, raiz: root } = raiz(); let envios = 0; let lecturas = 0;
  const vista = montarBandejaCorreccionesCronos({ raiz: root, cliente: {
    consultarBandeja: async () => bandeja(),
    registrarActuacion: async () => { envios++; throw new ErrorClienteCorreccionesCronos("red_no_disponible"); },
    consultarHistoria: async () => { lecturas++; return historia(); },
  } });
  await esperar(); pulsar(nodo, "c5bRevisar", REF);
  nodo.eventos.submit({ target: { matches: () => true, elements: { namedItem: (n) => ({ value: n === "resultado" ? "favorable" : "Olvido revisado" }) } }, preventDefault() {} });
  pulsar(nodo, "data-c5b-confirmar"); await esperar();
  pulsar(nodo, "data-c5b-recuperar"); await esperar();
  assert.equal(envios, 1); assert.equal(lecturas, 1);
  assert.match(nodo.innerHTML, /La historia muestra una actuación con estos datos/);
  assert.match(nodo.innerHTML, /Reintentar la misma actuación/);
  assert.doesNotMatch(nodo.innerHTML, /Actuación registrada el/);
  vista.desmontar();
});
