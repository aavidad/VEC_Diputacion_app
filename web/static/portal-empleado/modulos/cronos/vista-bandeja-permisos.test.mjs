import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { ErrorClienteResolucionCronos } from "./cliente-resolucion-http.js";
import { montarBandejaPermisosCronos, renderizarBandejaPermisosCronos } from "./vista-bandeja-permisos.js";

function bandeja(paso = "responsable") {
  return {
    paso,
    pendientes: [
      { solicitud_ref: "permiso:cronos:solicitud:perm-va-00001", empleado_ref: "emp_AAAAAAAAAAAAAAAAAAAAAA", empleado_etiqueta: "Persona sintética A",
        permiso_ref: "permiso:cronos:vacaciones", nombre: "Vacaciones", circuito: "J-A", pendiente_asignacion: false, justificante_exigido: false, desde: "2026-10-05", hasta: "2026-10-07",
        cantidad: 3, unidad: "dia", estado: "solicitado", version: 1, solicitada_en: "2026-09-24T08:00:00Z" },
      { solicitud_ref: "permiso:cronos:solicitud:perm-hm-00001", empleado_ref: "emp_AAAAAAAAAAAAAAAAAAAAAA", empleado_etiqueta: "",
        permiso_ref: "permiso:cronos:horas-medico", nombre: "Horas de médico", circuito: "J-A", pendiente_asignacion: false, justificante_exigido: true, desde: "2026-10-08", hasta: "2026-10-08",
        hora_inicio: "09:00", hora_fin: "10:30", cantidad: 90, unidad: "hora", estado: "solicitado", version: 1, solicitada_en: "2026-09-24T09:00:00Z" },
    ],
  };
}
function raizFalsa() {
  const nodo = { dataset: {}, innerHTML: "", eventos: {}, addEventListener(tipo, fn) { this.eventos[tipo] = fn; },
    removeEventListener(tipo) { delete this.eventos[tipo]; }, remove() { this.eliminado = true; }, querySelector() { return null; } };
  return { nodo, raiz: { ownerDocument: { createElement: () => nodo }, append() {} } };
}
const esperar = async () => { for (let i = 0; i < 8; i++) await Promise.resolve(); };
const pulsar = (selector, dataset) => ({ target: { closest: (sel) => (sel === selector ? { dataset } : null) } });
const formulario = (valores) => ({ matches: () => true, elements: { namedItem: (n) => (n in valores ? { value: valores[n] } : null) } });

test("la bandeja muestra persona, permiso, periodo, duración y estado sin referencias internas", () => {
  const html = renderizarBandejaPermisosCronos({ estado: "listo", paso: "responsable", datos: bandeja() });
  assert.match(html, /Solicitudes por resolver/);
  assert.match(html, /Persona sintética A/);
  assert.match(html, /Sin nombre publicado/);
  assert.match(html, /3 días/);
  assert.match(html, /1 h 30 min/);
  assert.match(html, /Requiere justificante/);
  assert.match(html, /Pendiente de la jefatura/);
  assert.match(html, /aria-pressed="true">Jefatura/);
  assert.match(html, /data-accion="ayuda"/);
  assert.equal((html.match(/data-cronos-resolver=/g) || []).length, 2);
  assert.doesNotMatch(html, />[^<]*(emp_|catalogo:cronos|recibo:cronos|AD3|DEMO)[^<]*</u);
  assert.match(renderizarBandejaPermisosCronos({ estado: "listo", paso: "administracion", datos: { paso: "administracion", pendientes: [] } }), /No hay solicitudes pendientes/);
});

test("resolver exige motivo al denegar, conserva la clave en el reintento y recarga la bandeja", async () => {
  const { nodo, raiz } = raizFalsa(); const envios = []; let consultas = 0;
  let respuesta = new ErrorClienteResolucionCronos("servicio_no_disponible", 503);
  const cliente = {
    consultarBandeja: async ({ paso }) => { consultas++; return bandeja(paso); },
    resolver: async (e) => { envios.push(e); if (respuesta instanceof Error) throw respuesta; return respuesta; },
  };
  const vista = montarBandejaPermisosCronos({ raiz, cliente });
  await esperar();
  nodo.eventos.click(pulsar("[data-cronos-resolver]", { cronosResolver: "permiso:cronos:solicitud:perm-va-00001" }));
  assert.match(nodo.innerHTML, /Resolver Vacaciones de Persona sintética A/);
  assert.match(nodo.innerHTML, /Dar conformidad/);
  await nodo.eventos.submit({ target: formulario({ decision: "denegar", motivo: "   " }), preventDefault() {} });
  assert.match(nodo.innerHTML, /Indique el motivo de la denegación/);
  assert.equal(envios.length, 0, "sin motivo no se envía");
  await nodo.eventos.submit({ target: formulario({ decision: "denegar", motivo: " Coincide con el cierre " }), preventDefault() {} });
  assert.match(nodo.innerHTML, /No se pudo registrar la resolución/);
  assert.deepEqual(envios[0], { clave_operacion: envios[0].clave_operacion, solicitud_ref: "permiso:cronos:solicitud:perm-va-00001", paso: "responsable",
    decision: "denegar", version_esperada: 1, motivo: "Coincide con el cierre" });
  respuesta = { resolucion_ref: "x", solicitud_ref: "permiso:cronos:solicitud:perm-va-00001", recibo_ref: "recibo:cronos:1", estado: "denegado", version: 2, instante_utc: "2026-09-25T08:00:00Z", replay: false };
  await nodo.eventos.submit({ target: formulario({ decision: "denegar", motivo: "Coincide con el cierre" }), preventDefault() {} });
  await esperar();
  assert.equal(envios[1].clave_operacion, envios[0].clave_operacion, "un reintento conserva la clave");
  assert.match(nodo.innerHTML, /Permiso denegado\. La persona tiene el aviso en Cronos\./);
  assert.equal(consultas, 2, "tras resolver se recarga la bandeja");
  vista.desmontar();
  assert.equal(nodo.eliminado, true);
});

test("el paso de RRHH concede y otra resolución adelantada recarga con aviso", async () => {
  const { nodo, raiz } = raizFalsa(); const pasos = [];
  const cliente = {
    consultarBandeja: async ({ paso }) => { pasos.push(paso); return bandeja(paso); },
    resolver: async () => { throw new ErrorClienteResolucionCronos("estado_cambiado", 409); },
  };
  montarBandejaPermisosCronos({ raiz, cliente });
  await esperar();
  nodo.eventos.click(pulsar("[data-cronos-paso]", { cronosPaso: "administracion" }));
  await esperar();
  assert.deepEqual(pasos, ["responsable", "administracion"]);
  nodo.eventos.click(pulsar("[data-cronos-resolver]", { cronosResolver: "permiso:cronos:solicitud:perm-va-00001" }));
  assert.match(nodo.innerHTML, />\s*Conceder/);
  await nodo.eventos.submit({ target: formulario({ decision: "aprobar", motivo: "" }), preventDefault() {} });
  await esperar();
  assert.match(nodo.innerHTML, /La solicitud ha cambiado desde que la abrió/);
  assert.doesNotMatch(nodo.innerHTML, /data-cronos-resolucion-formulario/);
});

test("sin permiso o sin empleado no muestra solicitudes", async () => {
  for (const [codigo, texto] of [["no_competente", /No tiene permiso/], ["acceso_denegado", /No tiene permiso/], ["sin_empleado", /relación de empleo vigente/]]) {
    const { nodo, raiz } = raizFalsa();
    const vista = montarBandejaPermisosCronos({ raiz, cliente: { consultarBandeja: async () => { throw new ErrorClienteResolucionCronos(codigo, 403); }, resolver: async () => ({}) } });
    await esperar();
    assert.match(nodo.innerHTML, texto);
    assert.doesNotMatch(nodo.innerHTML, /data-cronos-resolver/);
    vista.desmontar();
  }
});

test("RRHH ve lo pendiente de asignar jefatura sin poder resolverlo y un 400 del servidor no culpa al motivo", async () => {
  const datos = bandeja("administracion");
  datos.pendientes[0].pendiente_asignacion = true;
  const html = renderizarBandejaPermisosCronos({ estado: "listo", paso: "administracion", datos });
  assert.match(html, /Pendiente de asignar jefatura/);
  assert.match(html, /Sin jefatura asignada: no se puede resolver hasta asignarla/);
  assert.equal((html.match(/data-cronos-resolver=/g) || []).length, 1, "sólo la otra fila se puede resolver");
  const { nodo, raiz } = raizFalsa();
  let respuesta = new ErrorClienteResolucionCronos("peticion_invalida", 400);
  const cliente = { consultarBandeja: async () => structuredClone(datos), resolver: async () => { throw respuesta; } };
  montarBandejaPermisosCronos({ raiz, cliente, paso: "administracion" });
  await esperar();
  nodo.eventos.click(pulsar("[data-cronos-resolver]", { cronosResolver: "permiso:cronos:solicitud:perm-va-00001" }));
  assert.doesNotMatch(nodo.innerHTML, /data-cronos-resolucion-formulario/, "no abre la resolución de lo pendiente de asignación");
  nodo.eventos.click(pulsar("[data-cronos-resolver]", { cronosResolver: "permiso:cronos:solicitud:perm-hm-00001" }));
  await nodo.eventos.submit({ target: formulario({ decision: "aprobar", motivo: "" }), preventDefault() {} });
  assert.match(nodo.innerHTML, /revise la decisión y el motivo/);
  assert.doesNotMatch(nodo.innerHTML, /Indique el motivo de la denegación/);
  respuesta = new ErrorClienteResolucionCronos("pendiente_asignacion", 409);
  await nodo.eventos.submit({ target: formulario({ decision: "aprobar", motivo: "" }), preventDefault() {} });
  await esperar();
  assert.match(nodo.innerHTML, /no tiene jefatura asignada/);
});

test("la vista no guarda nada en el navegador", async () => {
  const fuente = await readFile(new URL("./vista-bandeja-permisos.js", import.meta.url), "utf8");
  assert.doesNotMatch(fuente, /localStorage|sessionStorage|indexedDB|document\.cookie|Math\.random/u);
});
