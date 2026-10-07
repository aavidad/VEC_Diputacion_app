import assert from "node:assert/strict";
import test from "node:test";
import { readFile } from "node:fs/promises";
import { ErrorClienteResolucionCronos } from "./cliente-resolucion-http.js";
import { montarAvisosPropiosCronos, renderizarAvisosPropiosCronos } from "./vista-avisos-propios.js";

const AVISO_DENEGADO = "aviso:cronos:00000000-0000-4000-8000-000000000001";
function avisos() {
  return { avisos: [
    { aviso_ref: AVISO_DENEGADO, solicitud_ref: "permiso:cronos:solicitud:perm-ap-00001", estado: "denegado", motivo: "Coincide con el cierre del servicio",
      resuelto_en: "2026-09-25T08:00:00Z", permiso_ref: "permiso:cronos:asuntos-propios", nombre: "Asuntos propios", desde: "2026-10-05", hasta: "2026-10-06",
      cantidad: 2, unidad: "dia", archivado: false },
    { aviso_ref: "aviso:cronos:00000000-0000-4000-8000-000000000002", solicitud_ref: "permiso:cronos:solicitud:perm-hm-00001", estado: "concedido",
      resuelto_en: "2026-09-24T08:00:00Z", permiso_ref: "permiso:cronos:horas-medico", nombre: "Horas de médico", desde: "2026-10-08", hasta: "2026-10-08",
      hora_inicio: "09:00", hora_fin: "10:00", cantidad: 60, unidad: "hora", archivado: true, archivado_en: "2026-09-25T10:00:00Z" },
  ] };
}
function raizFalsa() {
  const nodo = { dataset: {}, innerHTML: "", eventos: {}, addEventListener(tipo, fn) { this.eventos[tipo] = fn; },
    removeEventListener(tipo) { delete this.eventos[tipo]; }, remove() { this.eliminado = true; },
    querySelector(selector) { return { focus: () => { this.foco = selector; } }; } };
  return { nodo, raiz: { ownerDocument: { createElement: () => nodo }, append() {} } };
}
const esperar = async () => { for (let i = 0; i < 8; i++) await Promise.resolve(); };
const pulsar = (selector, dataset) => ({ target: { closest: (sel) => (sel === selector ? { dataset } : null) } });

test("los avisos recibidos dicen qué se resolvió y por qué, sin referencias internas", () => {
  const html = renderizarAvisosPropiosCronos({ estado: "listo", datos: avisos() });
  assert.match(html, /Se deniega el permiso Asuntos propios: .*\(2 días\)\./u);
  assert.match(html, /Motivo: Coincide con el cierre del servicio/);
  assert.match(html, /data-cronos-archivar=/);
  assert.doesNotMatch(html, /Horas de médico/, "el archivado no está entre los recibidos");
  assert.doesNotMatch(html, />[^<]*(aviso:cronos|permiso:cronos|recibo:cronos|DEMO)[^<]*</u);
  const archivados = renderizarAvisosPropiosCronos({ estado: "listo", filtro: "archivados", datos: avisos() });
  assert.match(archivados, /Se concede el permiso Horas de médico: .*09:00–10:00 \(1 h\)\./u);
  assert.match(archivados, /Archivado el/);
  assert.doesNotMatch(archivados, /data-cronos-archivar=/);
});

test("archivar conserva la clave en el reintento, recarga y trata el archivo previo como hecho", async () => {
  const { nodo, raiz } = raizFalsa(); const envios = []; let consultas = 0;
  let respuesta = new ErrorClienteResolucionCronos("servicio_no_disponible", 503);
  const cliente = {
    consultarAvisos: async () => { consultas++; return avisos(); },
    archivarAviso: async (e) => { envios.push(e); if (respuesta instanceof Error) throw respuesta; return respuesta; },
  };
  const vista = montarAvisosPropiosCronos({ raiz, cliente });
  await esperar();
  await nodo.eventos.click(pulsar("[data-cronos-archivar]", { cronosArchivar: AVISO_DENEGADO }));
  await esperar();
  assert.match(nodo.innerHTML, /No se pudo archivar el aviso/);
  respuesta = { archivo_ref: "x", aviso_ref: AVISO_DENEGADO, recibo_ref: "recibo:cronos:3", instante_utc: "2026-09-25T09:00:00Z", replay: false };
  nodo.eventos.click(pulsar("[data-cronos-archivar]", { cronosArchivar: AVISO_DENEGADO }));
  await esperar();
  assert.equal(envios[1].clave_operacion, envios[0].clave_operacion, "un reintento conserva la clave");
  assert.match(nodo.innerHTML, /Aviso archivado\./);
  assert.equal(consultas, 2);
  respuesta = new ErrorClienteResolucionCronos("estado_cambiado", 409);
  nodo.eventos.click(pulsar("[data-cronos-archivar]", { cronosArchivar: AVISO_DENEGADO }));
  await esperar();
  assert.match(nodo.innerHTML, /Este aviso ya estaba archivado\./);
  nodo.eventos.click(pulsar("[data-cronos-filtro]", { cronosFiltro: "archivados" }));
  assert.match(nodo.innerHTML, /Horas de médico/);
  vista.desmontar();
  assert.equal(nodo.eliminado, true);
});

test("sin empleado o sin servicio no muestra avisos", async () => {
  for (const [codigo, texto] of [["sin_empleado", /relación de empleo vigente/], ["servicio_no_disponible", /No se pudieron consultar/]]) {
    const { nodo, raiz } = raizFalsa();
    const vista = montarAvisosPropiosCronos({ raiz, cliente: { consultarAvisos: async () => { throw new ErrorClienteResolucionCronos(codigo, 503); }, archivarAviso: async () => ({}) } });
    await esperar();
    assert.match(nodo.innerHTML, texto);
    vista.desmontar();
  }
});

test("el archivo confirmado conserva el resultado si falla la lectura y reintenta sólo la consulta", async () => {
  for (const resultado of ["nuevo", "replay", "estado_cambiado"]) {
    const { nodo, raiz } = raizFalsa(); const anuncios = []; let consultas = 0; let envios = 0;
    const confirmacion = resultado === "nuevo" ? /Aviso archivado\./u : /Este aviso ya estaba archivado\./u;
    const actualizados = avisos();
    actualizados.avisos[0] = { ...actualizados.avisos[0], archivado: true, archivado_en: "2026-09-25T09:00:00Z" };
    const cliente = {
      consultarAvisos: async () => {
        consultas++;
        if (consultas === 2 || consultas === 3) throw new ErrorClienteResolucionCronos("servicio_no_disponible", 503);
        return consultas === 1 ? avisos() : actualizados;
      },
      archivarAviso: async () => {
        envios++;
        if (resultado === "estado_cambiado") throw new ErrorClienteResolucionCronos("estado_cambiado", 409);
        return { replay: resultado === "replay" };
      },
    };
    const vista = montarAvisosPropiosCronos({ raiz, cliente, anunciar: (mensaje) => anuncios.push(mensaje) });
    await esperar();
    nodo.eventos.click(pulsar("[data-cronos-archivar]", { cronosArchivar: AVISO_DENEGADO }));
    await esperar();
    assert.match(nodo.innerHTML, confirmacion);
    assert.match(nodo.innerHTML, /data-estado="error"/u);
    assert.match(nodo.innerHTML, /data-cronos-reintentar-consulta/u);
    assert.doesNotMatch(nodo.innerHTML, /data-cronos-archivar=|cronos-avisos-lista/u, "la lista anterior no se muestra como vigente");
    assert.equal(anuncios.some((mensaje) => confirmacion.test(mensaje)), true);
    nodo.eventos.click(pulsar("[data-cronos-filtro]", { cronosFiltro: "archivados" }));
    assert.match(nodo.innerHTML, confirmacion, "cambiar de filtro durante el error conserva el resultado");
    nodo.eventos.click(pulsar("[data-cronos-reintentar-consulta]", {}));
    await esperar();
    assert.match(nodo.innerHTML, confirmacion);
    assert.equal(nodo.foco, "[data-cronos-reintentar-consulta]");
    nodo.eventos.click(pulsar("[data-cronos-reintentar-consulta]", {}));
    await esperar();
    assert.equal(consultas, 4);
    assert.equal(envios, 1, "recuperar la lectura nunca repite el archivo confirmado");
    assert.match(nodo.innerHTML, /data-estado="listo"/u);
    assert.match(nodo.innerHTML, confirmacion);
    assert.match(nodo.innerHTML, /Asuntos propios/u, "la consulta recuperada utiliza el filtro elegido");
    assert.doesNotMatch(nodo.innerHTML, /data-cronos-reintentar-consulta|data-cronos-archivar=/u);
    assert.equal(nodo.foco, '[data-cronos-filtro="archivados"]');
    vista.desmontar();
  }
});

test("una consulta sustituida o desmontada no cambia el estado ni mueve el foco", async () => {
  const { nodo, raiz } = raizFalsa(); const lecturas = []; const anuncios = [];
  const cliente = {
    consultarAvisos: ({ signal }) => new Promise((resolve, reject) => lecturas.push({ signal, resolve, reject })),
    archivarAviso: async () => ({}),
  };
  const vista = montarAvisosPropiosCronos({ raiz, cliente, anunciar: (mensaje) => anuncios.push(mensaje) });
  lecturas[0].reject(new ErrorClienteResolucionCronos("servicio_no_disponible", 503));
  await esperar();
  nodo.eventos.click(pulsar("[data-cronos-reintentar-consulta]", {}));
  const vigente = vista.recargar();
  assert.equal(lecturas[1].signal.aborted, true);
  lecturas[2].resolve(avisos()); await vigente;
  const html = nodo.innerHTML;
  lecturas[1].reject(new ErrorClienteResolucionCronos("servicio_no_disponible", 503));
  await esperar();
  assert.equal(nodo.innerHTML, html);
  assert.equal(nodo.foco, undefined);
  assert.equal(anuncios.length, 1);
  const pendiente = vista.recargar();
  const desmontado = nodo.innerHTML;
  vista.desmontar();
  assert.equal(lecturas[3].signal.aborted, true);
  lecturas[3].resolve(avisos()); await pendiente;
  assert.equal(nodo.innerHTML, desmontado);
  assert.equal(nodo.foco, undefined);
  await vista.recargar();
  assert.equal(lecturas.length, 4, "una vista desmontada no consulta de nuevo");
});

test("la vista no guarda nada en el navegador", async () => {
  const fuente = await readFile(new URL("./vista-avisos-propios.js", import.meta.url), "utf8");
  assert.doesNotMatch(fuente, /localStorage|sessionStorage|indexedDB|document\.cookie|Math\.random/u);
});


test("volver a consultar usa el catálogo efectivo de resolución en ambos idiomas", async () => {
  for (const [idioma, etiqueta] of [["es", "Volver a consultar"], ["en", "Refresh notices"]]) {
    const raiz = new URL(`../../../textos/${idioma}/`, import.meta.url);
    const [cronos, resolucion] = await Promise.all(["cronos.json", "cronos-resolucion.json"].map(async (nombre) =>
      JSON.parse(await readFile(new URL(nombre, raiz), "utf8"))));
    assert.equal(cronos.resolucion.avisos_reintentar_consulta, undefined);
    const mensajes = { ...cronos.solicitudes, ...cronos.resolucion, ...resolucion.bandeja };
    const html = renderizarAvisosPropiosCronos({ estado: "error", mensajes });
    assert.ok(html.includes(`data-cronos-reintentar-consulta>${etiqueta}</button>`));
  }
});
