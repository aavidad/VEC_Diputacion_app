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
const formulario = (valores) => ({ matches: (sel) => sel === "[data-cronos-resolucion-formulario]", elements: { namedItem: (n) => (n in valores ? { value: valores[n] } : null) } });

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

function variasSolicitudes(cantidad, paso = "responsable") {
  const datos = bandeja(paso);
  datos.pendientes = Array.from({ length: cantidad }, (_, i) => ({ ...datos.pendientes[0],
    solicitud_ref: `permiso:cronos:solicitud:solicitud-${String(i).padStart(5, "0")}`,
    empleado_etiqueta: `Persona ${String(i).padStart(3, "0")}` }));
  return datos;
}
function formularioFiltros(busqueda, estado = "todos") {
  return { matches: (sel) => sel === "[data-cronos-bandeja-filtros]",
    elements: { namedItem: (n) => ({ value: n === "busqueda" ? busqueda : estado }) } };
}

test("el resumen y las páginas cuentan solo el conjunto de la bandeja autorizada", () => {
  const datos = variasSolicitudes(41, "administracion");
  datos.pendientes[0].pendiente_asignacion = true;
  datos.pendientes[1].estado = "pendiente_administracion";
  const primera = renderizarBandejaPermisosCronos({ estado: "listo", paso: "administracion", datos });
  assert.match(primera, /solo las solicitudes de esta bandeja para RRHH/);
  assert.match(primera, /Mostrando 1–20 de 41 solicitudes/);
  assert.match(primera, /Página 1 de 3/);
  assert.equal((primera.match(/data-cronos-resolver=/g) || []).length, 19, "lo no resoluble permanece deshabilitado");
  assert.doesNotMatch(primera, /Persona 020/);
  const ultima = renderizarBandejaPermisosCronos({ estado: "listo", paso: "administracion", datos, pagina: 99 });
  assert.match(ultima, /Mostrando 41–41 de 41/);
  assert.match(ultima, /Persona 040/);
  assert.equal((ultima.match(/data-cronos-resolver=/g) || []).length, 1);
  assert.match(ultima, /data-cronos-bandeja-pagina="siguiente" disabled/);
});

test("la búsqueda ignora acentos, filtra estados y nunca busca referencias internas", () => {
  const datos = bandeja("administracion");
  datos.pendientes[0].empleado_etiqueta = "Ángela Sintética";
  datos.pendientes[0].pendiente_asignacion = true;
  datos.pendientes[1].estado = "pendiente_administracion";
  const filtrar = (busqueda, estado = "todos") => renderizarBandejaPermisosCronos({ estado: "listo", paso: "administracion", datos, filtros: { busqueda, estado } });
  assert.match(filtrar("ANGELA", "asignacion"), /Mostrando 1–1 de 1 solicitudes que coinciden; 2 en esta bandeja/);
  assert.match(filtrar("", "rrhh"), /Mostrando 1–1 de 1 solicitudes/);
  assert.match(filtrar("emp_A"), /Ninguna solicitud coincide/);
  assert.match(filtrar("ANGELA", "rrhh"), /Ninguna solicitud coincide/);
  assert.match(filtrar('"<img onerror=x>'), /value="&quot;&lt;img onerror=x&gt;"/);
});

test("filtrar y paginar conservan el borrador, el foco y las operaciones del cliente", async () => {
  const { nodo, raiz } = raizFalsa();
  const focos = []; const consultas = []; const envios = [];
  let motivo = "Motivo que estoy preparando";
  nodo.querySelector = (selector) => selector.endsWith("[name=motivo]") ? { value: motivo }
    : { focus() { focos.push(selector); } };
  const vista = montarBandejaPermisosCronos({ raiz, cliente: {
    consultarBandeja: async ({ paso }) => { consultas.push(paso); return variasSolicitudes(41, paso); },
    resolver: async (peticion) => { envios.push(peticion); throw new ErrorClienteResolucionCronos("servicio_no_disponible", 503); },
  } });
  await esperar();
  nodo.eventos.click(pulsar("[data-cronos-resolver]", { cronosResolver: "permiso:cronos:solicitud:solicitud-00000" }));
  nodo.eventos.change({ target: { name: "decision", value: "denegar" } });
  nodo.eventos.click(pulsar("[data-cronos-bandeja-pagina]", { cronosBandejaPagina: "siguiente" }));
  assert.match(nodo.innerHTML, /Página 2 de 3/);
  assert.match(nodo.innerHTML, /Motivo que estoy preparando/);
  assert.match(nodo.innerHTML, /value="denegar" checked/);
  assert.equal(focos.at(-1), "[data-cronos-bandeja-paginacion]");
  await nodo.eventos.submit({ target: formularioFiltros("Persona 040"), preventDefault() {} });
  assert.match(nodo.innerHTML, /Mostrando 1–1 de 1/);
  assert.match(nodo.innerHTML, /Motivo que estoy preparando/);
  assert.equal(focos.at(-1), "[data-cronos-bandeja-aplicar]");
  assert.deepEqual(consultas, ["responsable"], "los filtros no consultan a otras personas");
  assert.equal(envios.length, 0, "no resuelven automáticamente");
  nodo.eventos.click(pulsar("[data-cronos-bandeja-limpiar]", {}));
  assert.match(nodo.innerHTML, /Página 1 de 3/);
  assert.equal(focos.at(-1), "[name=busqueda]");
  await nodo.eventos.submit({ target: formulario({ decision: "denegar", motivo }), preventDefault() {} });
  assert.equal(envios[0].solicitud_ref, "permiso:cronos:solicitud:solicitud-00000");
  assert.equal(envios[0].version_esperada, 1);
  assert.equal(envios[0].motivo, motivo);
  nodo.eventos.click(pulsar("[data-cronos-paso]", { cronosPaso: "administracion" }));
  await esperar();
  assert.match(nodo.innerHTML, /Mostrando 1–20 de 41/);
  assert.doesNotMatch(nodo.innerHTML, /data-cronos-resolucion-formulario/);
  assert.deepEqual(consultas, ["responsable", "administracion"]);
  vista.desmontar();
});

test("una denegación de la recarga retira filas, resumen y filtros previos", async () => {
  const { nodo, raiz } = raizFalsa(); let denegado = false;
  const vista = montarBandejaPermisosCronos({ raiz, cliente: {
    consultarBandeja: async () => { if (denegado) throw new ErrorClienteResolucionCronos("acceso_denegado", 403); return variasSolicitudes(22); },
    resolver: async () => ({}),
  } });
  await esperar(); assert.match(nodo.innerHTML, /Página 1 de 2/);
  denegado = true; await vista.recargar();
  assert.doesNotMatch(nodo.innerHTML, /Persona 000|data-cronos-bandeja-filtros|Solicitudes en esta bandeja|data-cronos-resolver/);
  vista.desmontar();
});


test("el nuevo catálogo traduce el alcance, filtros y páginas en inglés", async () => {
  const base = JSON.parse(await readFile(new URL("../../../textos/en/cronos.json", import.meta.url), "utf8"));
  const extra = JSON.parse(await readFile(new URL("../../../textos/en/cronos-resolucion.json", import.meta.url), "utf8"));
  const html = renderizarBandejaPermisosCronos({ estado: "listo", datos: variasSolicitudes(21),
    mensajes: { ...base.solicitudes, ...base.resolucion, ...extra.bandeja }, locale: "en-GB" });
  assert.match(html, /The summary counts only requests in this inbox/);
  assert.match(html, /Search by person/);
  assert.match(html, /Showing 1–20 of 21 matching requests/);
  assert.match(html, /Page 1 of 2/);
  assert.match(html, /<svg[^>]*aria-hidden="true"/);
});


test("una resolución tardía no altera el paso nuevo, el cierre ni el desmontaje", async () => {
  for (const accion of ["paso", "cerrar", "desmontar"]) {
    for (const resultado of ["exito", "fallo"]) {
      const { nodo, raiz } = raizFalsa(); const anuncios = []; let completar; let signal;
      const vista = montarBandejaPermisosCronos({ raiz, anunciar: (mensaje) => anuncios.push(mensaje), cliente: {
        consultarBandeja: async ({ paso }) => bandeja(paso),
        resolver: (_peticion, opciones) => { signal = opciones.signal; return new Promise((resolve, reject) => {
          completar = () => resultado === "fallo" ? reject(new ErrorClienteResolucionCronos("servicio_no_disponible", 503))
            : resolve({ estado: "concedido", replay: false });
        }); },
      } });
      await esperar();
      nodo.eventos.click(pulsar("[data-cronos-resolver]", { cronosResolver: "permiso:cronos:solicitud:perm-va-00001" }));
      const envio = nodo.eventos.submit({ target: formulario({ decision: "aprobar", motivo: "" }), preventDefault() {} });
      if (accion === "paso") nodo.eventos.click(pulsar("[data-cronos-paso]", { cronosPaso: "administracion" }));
      else if (accion === "cerrar") nodo.eventos.click(pulsar("[data-cronos-resolucion-cerrar]", {}));
      else vista.desmontar();
      await esperar(); assert.equal(signal.aborted, true);
      const htmlAntes = nodo.innerHTML; const anunciosAntes = [...anuncios];
      completar(); await envio; await esperar();
      assert.equal(nodo.innerHTML, htmlAntes, `${accion}/${resultado}: conserva la vista actual`);
      assert.deepEqual(anuncios, anunciosAntes, `${accion}/${resultado}: no anuncia otra resolución`);
      vista.desmontar();
    }
  }
});
