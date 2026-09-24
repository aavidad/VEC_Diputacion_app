import assert from "node:assert/strict";
import { access, readFile } from "node:fs/promises";
import test from "node:test";

const version = "20260924-f2-shell-v1";
const versionTemaBase = "20260924-f2-tema-base-v2";
const versionCache = "20260924-f2-cache-v2";
const versionCachePersonal = "20260924-f2-cache-v3";
const versionPersonalInterno = "20260924-p1-personal-interno-v2";
const versionIntegracion = "20260924-integracion-b7-v1";
const raiz = new URL("./", import.meta.url);

function versionesDe(codigo, recurso) {
  const escaped = recurso.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  return [...codigo.matchAll(new RegExp(`${escaped}\\?v=([^"']+)`, "gu"))].map((match) => match[1]);
}

function versionDe(codigo, recurso) {
  return versionesDe(codigo, recurso)[0] || "";
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
    assert.equal(versionDe(html, `/portal-empleado/${recurso}`),
      recurso === "portal.js" ? versionIntegracion : recurso === "portal.css" ? versionTemaBase : version);
    assert.notEqual(versionDe(html, `/portal-empleado/${recurso}`), versionAntigua);
  }
  for (const recurso of ["portal-baremacion.css", "portal-contratos.css", "portal-convocatorias.css",
    "modulos/seleccion/inscripciones/inscripciones.css", "modulos/seleccion/pruebas/pruebas.css",
    "modulos/seleccion/comunicaciones/comunicaciones.css"]) {
    assert.equal(versionDe(html, `/portal-empleado/${recurso}`), version);
    await access(new URL(recurso, raiz));
  }
  assert.equal(versionDe(html, "/comun/tema-vec.css"), versionTemaBase);
});

test("la caché immutable del tema anterior descarga ambas hojas de estilo renovadas", async () => {
  const html = await readFile(new URL("index.html", raiz), "utf8");
  const recursos = [
    ["/portal-empleado/portal.css", new URL("portal.css", raiz)],
    ["/comun/tema-vec.css", new URL("../comun/tema-vec.css", raiz)],
  ];
  const cache = new Map(recursos.map(([ruta]) =>
    [`${ruta}?v=${version}`, `/* respuesta immutable anterior: ${ruta} */`]));
  const descargas = new Set();
  for (const [ruta, archivo] of recursos) {
    const versiones = versionesDe(html, ruta);
    assert.deepEqual(versiones, [versionTemaBase], `${ruta}: URL de tema única y renovada`);
    const urlAntigua = `${ruta}?v=${version}`;
    const urlNueva = `${ruta}?v=${versionTemaBase}`;
    assert.ok(cache.has(urlAntigua), `${ruta}: la caché antigua está precargada`);
    assert.ok(!html.includes(urlAntigua), `${ruta}: HTML no solicita la URL antigua`);
    if (!cache.has(urlNueva)) {
      cache.set(urlNueva, await readFile(archivo, "utf8"));
      descargas.add(urlNueva);
    }
  }
  assert.deepEqual(descargas, new Set(recursos.map(([ruta]) => `${ruta}?v=${versionTemaBase}`)),
    "las dos hojas se vuelven a solicitar con URL nueva");
});

test("el grafo JS propio llega desde HTML a los consumidores F2 con versiones nuevas", async () => {
  const [html, portal, coordinador, dietas] = await Promise.all([
    readFile(new URL("index.html", raiz), "utf8"),
    readFile(new URL("portal.js", raiz), "utf8"),
    readFile(new URL("portal-modulos-coordinador.js", raiz), "utf8"),
    readFile(new URL("modulos/dietas/vista-recorridos.js", raiz), "utf8"),
  ]);
  assert.equal(versionDe(html, "/portal-empleado/portal.js"), versionIntegracion);
  assert.equal(versionDe(portal, "./portal-modulos-coordinador.js"), versionPersonalInterno);
  for (const recurso of ["portal-menu-bolsa.js",
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

test("la caché immutable previa no retiene el catálogo i18n ni los consumidores F2", async () => {
  const versionesPrevias = new Map([
    ["portal.js", [version, versionCache, versionCachePersonal, "20260924-p1-personal-interno-v1"]],
    ["portal-modulos-coordinador.js", [version, versionCache, versionCachePersonal, "20260924-p1-personal-interno-v1"]],
    ["portal-catalogo-modulos.js", ["20260906-acceso-certificado-v1"]],
    ["portal-inicio.js", ["20260923-p4-reintento-v2"]],
    ["portal-eventos.js", ["20260721-acceso-real-v2"]],
    ["portal-borradores-ui.js", ["20260921-avisos-r5-v1"]],
    ["portal-borradores-acceso.js", ["20260721-acceso-real-v2"]],
    ["portal-i18n.js", ["20260721-acceso-real-v2", "20260923-p4-reintento-v2", version]],
    ["modulos/cronos/vista-recorridos.js", ["20260920-cronos-bandeja-v2"]],
    ["modulos/personal/vista.js", ["20260920-personal-catalogo-v1"]],
    ["modulos/personal/cliente-http-categorias.js", ["20260920-personal-catalogo-v1"]],
    ["modulos/personal/vista-estructura-organizativa-publica.js", ["20260920-personal-estructura-v1"]],
  ]);
  const cache = new Map();
  for (const [recurso, versiones] of versionesPrevias) {
    for (const previa of versiones) {
      const url = `/portal-empleado/${recurso}?v=${previa}`;
      cache.set(url, `/* respuesta immutable antigua: ${url} */`);
    }
  }
  const urlsPrevias = new Set(cache.keys());
  const hitsPrevios = [];
  const descargas = new Set();
  async function cargar(url) {
    if (cache.has(url)) {
      if (urlsPrevias.has(url)) hitsPrevios.push(url);
      return cache.get(url);
    }
    const ruta = url.split("?", 1)[0].replace(/^\/portal-empleado\//u, "");
    const codigo = await readFile(new URL(ruta, raiz), "utf8");
    cache.set(url, codigo);
    descargas.add(url);
    return codigo;
  }
  const aristas = new Map([
    ["index.html", ["portal.js"]],
    ["portal.js", ["portal-modulos-coordinador.js", "portal-inicio.js", "portal-eventos.js",
      "portal-borradores-ui.js", "portal-i18n.js"]],
    ["portal-modulos-coordinador.js", ["portal-catalogo-modulos.js", "portal-i18n.js",
      "modulos/cronos/vista-recorridos.js", "modulos/personal/vista.js",
      "modulos/personal/cliente-http-categorias.js",
      "modulos/personal/vista-estructura-organizativa-publica.js"]],
    ["portal-catalogo-modulos.js", ["portal-i18n.js"]],
    ["portal-inicio.js", ["portal-i18n.js"]],
    ["portal-eventos.js", ["portal-i18n.js"]],
    ["portal-borradores-ui.js", ["portal-borradores-acceso.js", "portal-i18n.js"]],
    ["portal-borradores-acceso.js", ["portal-i18n.js"]],
  ]);
  const html = await readFile(new URL("index.html", raiz), "utf8"); // HTML: no-store.
  const pendientes = [["index.html", html]];
  const visitados = new Set();
  while (pendientes.length > 0) {
    const [padre, codigo] = pendientes.shift();
    if (visitados.has(padre)) continue;
    visitados.add(padre);
    for (const hijo of aristas.get(padre) || []) {
      const ruta = padre === "index.html" ? `/portal-empleado/${hijo}` : `./${hijo}`;
      const versionesHijo = versionesDe(codigo, ruta);
      const personal = padre === "portal-modulos-coordinador.js" && hijo.startsWith("modulos/personal/");
      assert.equal(versionesHijo.length, ["modulos/personal/vista.js", "modulos/personal/cliente-http-categorias.js"].includes(hijo) ? 2 : 1,
        `${padre} → ${hijo}: número de aristas`);
      const versionEsperada = padre === "index.html" ? versionIntegracion : hijo === "portal-modulos-coordinador.js"
        || hijo === "modulos/personal/cliente-http-categorias.js"
        ? versionPersonalInterno : personal ? versionCachePersonal : versionCache;
      for (const versionHijo of versionesHijo) {
        assert.equal(versionHijo, versionEsperada, `${padre} → ${hijo}`);
        const url = `/portal-empleado/${hijo}?v=${versionHijo}`;
        pendientes.push([hijo, await cargar(url)]);
      }
    }
  }
  assert.deepEqual(hitsPrevios, [], "ninguna URL immutable antigua se recupera de caché");
  assert.ok(descargas.has(`/portal-empleado/modulos/personal/cliente-http-categorias.js?v=${versionPersonalInterno}`));
  assert.equal(descargas.size, versionesPrevias.size, "todos los recursos cambiados se descargan de nuevo");
});

test("la integración B7 renueva controlador y presentador desde la entrada HTML con caché caliente", async () => {
  const html = await readFile(new URL("index.html", raiz), "utf8");
  const portal = await readFile(new URL("portal.js", raiz), "utf8");
  const versionAnteriorB7 = "20260923-pweb13-b8-v1";
  const versionesAnteriores = new Map([
    ["portal.js", "20260924-p1-personal-interno-v2"],
    ["portal-bolsas-api.js", versionAnteriorB7],
    ["portal-panel-interno.js", "20260923-pweb17-v1"],
  ]);
  const cache = new Map([...versionesAnteriores].map(([recurso, previa]) => [`${recurso}?v=${previa}`, "bytes antiguos"]));
  const descargas = [];
  for (const [recurso, previa] of versionesAnteriores) {
    const codigo = recurso === "portal.js" ? html : portal;
    const prefijo = recurso === "portal.js" ? "/portal-empleado/" : "./";
    const nueva = versionDe(codigo, prefijo + recurso);
    assert.equal(nueva, "20260924-integracion-b7-v1", recurso);
    assert.notEqual(nueva, previa, recurso);
    assert.ok(!codigo.includes(`${prefijo}${recurso}?v=${previa}`), `${recurso}: no queda URL antigua`);
    const url = `${recurso}?v=${nueva}`;
    assert.ok(!cache.has(url), `${recurso} debe solicitar nuevos bytes`);
    cache.set(url, await readFile(new URL(recurso, raiz), "utf8"));
    descargas.push(recurso);
  }
  assert.deepEqual(descargas, [...versionesAnteriores.keys()]);
});
