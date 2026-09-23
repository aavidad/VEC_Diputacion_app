import assert from "node:assert/strict";
import { access, readFile } from "node:fs/promises";
import test from "node:test";

const version = "20260924-f2-shell-v1";
const raiz = new URL("./", import.meta.url);

function versionDe(codigo, recurso) {
  const escaped = recurso.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  return codigo.match(new RegExp(`${escaped}\\?v=([^"']+)`, "u"))?.[1] || "";
}

test("una carga con caché caliente solicita CSS F2 y entrada JS con URL nueva", async () => {
  const html = await readFile(new URL("index.html", raiz), "utf8");
  const previo = new Map([
    ["portal.js", "20260923-p4-reintento-v2"],
    ["portal.css", "20260923-pweb17-v1"],
    ["portal-componentes.css", "20260923-pweb17-v1"],
    ["modulos/cronos/cronos.css", "20260920-cronos-bandeja-v2"],
    ["modulos/dietas/dietas.css", "20260923-dietas-r1"],
    ["modulos/personal/ficha-integral.css", "20260920-recorridos-visibles-v1"],
    ["modulos/contratacion-temporal/expedientes-operativo.css", "20260918-botones-v1"],
  ]);
  for (const [recurso, versionAntigua] of previo) {
    assert.equal(versionDe(html, `/portal-empleado/${recurso}`), version);
    assert.notEqual(versionDe(html, `/portal-empleado/${recurso}`), versionAntigua);
  }
  for (const recurso of ["portal-baremacion.css", "portal-contratos.css", "portal-convocatorias.css",
    "modulos/seleccion/inscripciones/inscripciones.css", "modulos/seleccion/pruebas/pruebas.css",
    "modulos/seleccion/comunicaciones/comunicaciones.css"]) {
    assert.equal(versionDe(html, `/portal-empleado/${recurso}`), version);
    await access(new URL(recurso, raiz));
  }
  assert.equal(versionDe(html, "/comun/tema-vec.css"), version);
});

test("el grafo JS propio llega desde HTML a los consumidores F2 con versiones nuevas", async () => {
  const [html, portal, coordinador, dietas] = await Promise.all([
    readFile(new URL("index.html", raiz), "utf8"),
    readFile(new URL("portal.js", raiz), "utf8"),
    readFile(new URL("portal-modulos-coordinador.js", raiz), "utf8"),
    readFile(new URL("modulos/dietas/vista-recorridos.js", raiz), "utf8"),
  ]);
  assert.equal(versionDe(html, "/portal-empleado/portal.js"), version);
  for (const recurso of ["portal-modulos-coordinador.js", "portal-menu-bolsa.js",
    "portal-vistas-baremacion.js", "portal-vistas-convocatorias.js", "portal-vistas-operaciones.js",
    "modulos/seleccion/inscripciones/vista.js", "modulos/seleccion/pruebas/vista.js",
    "modulos/seleccion/comunicaciones/vista.js"]) {
    assert.equal(versionDe(portal, `./${recurso}`), version, recurso);
    await access(new URL(recurso, raiz));
  }
  for (const recurso of ["modulos/cronos/vista.js", "modulos/dietas/vista-recorridos.js",
    "modulos/dietas/cliente-borradores-http.js", "modulos/personal/vista-ficha-integral.js",
    "modulos/nominas/vista.js", "modulos/solicitudes/vista.js", "modulos/meritos/vista.js",
    "modulos/comunicaciones/vista.js", "modulos/documentos/vista.js", "modulos/aprobaciones/vista.js"]) {
    assert.equal(versionDe(coordinador, `./${recurso}`), version, recurso);
    await access(new URL(recurso, raiz));
  }
  assert.equal(versionDe(dietas, "./vista-borradores-propios.js"), version);
});
