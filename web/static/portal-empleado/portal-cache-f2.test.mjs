import assert from "node:assert/strict";
import { access, readFile } from "node:fs/promises";
import test from "node:test";
import { exigirVersiones, posterior } from "./versiones-cache.test-helper.mjs";

// Versiones publicadas en la serie F2 y sucesivas. Los recursos que cambiaron
// después no fijan su literal vigente: se exige `posterior(...)`, es decir, una
// URL única y distinta de la publicada, que una caché immutable no retiene.
const version = "20260924-f2-shell-v1";
const versionTemaBase = "20260924-f2-tema-base-v2";
const versionSaltoMovil = "20260924-f2-salto-movil-v3";
const versionSaltoMovilAnterior = "20260924-f2-salto-movil-v2";
const versionCache = "20260924-f2-cache-v2";
const versionCachePersonal = "20260924-f2-cache-v3";
const versionPersonalInterno = "20260924-p1-personal-interno-v2";
const versionPersonalEstados = "20260924-f2-personal-estados-v4";
const versionCronosPermisos = "20260924-f2-cronos-permisos-v2";
const versionCronosAyuda = "20260924-cronos-integrado-v1";
const versionCronosVista = "20260924-web-paradas-periodos-v1";
const versionDietasShell = "20260924-web-paradas-periodos-v1";
const versionEntradaAyuda = "20260924-rescate-web-v4";
const versionDietasIcono = "20260924-dietas-ayuda-icono-v1";
const versionDietasVista = "20260924-web-paradas-periodos-v1";
const versionVistasC = "20260924-web-c-v1";
const versionDietasRecuperacion = "20260924-dietas-recuperacion-v3";
const versionIntegracion = "20260924-web-paradas-periodos-v1";
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
  const esperadas = new Map([
    ["portal.js", posterior(versionEntradaAyuda)],
    ["portal.css", posterior(versionSaltoMovil)],
    ["portal-componentes.css", posterior(version)],
    ["modulos/cronos/cronos.css", posterior(versionCronosVista)],
    ["modulos/dietas/dietas.css", posterior(versionDietasIcono)],
    ["modulos/personal/ficha-integral.css", version],
    ["modulos/contratacion-temporal/expedientes-operativo.css", posterior(version)],
  ]);
  for (const [recurso, versionAntigua] of previo) {
    const vigente = exigirVersiones(html, `/portal-empleado/${recurso}`, esperadas.get(recurso));
    assert.notEqual(vigente, versionAntigua);
  }
  for (const recurso of ["portal-contratos.css"]) {
    exigirVersiones(html, `/portal-empleado/${recurso}`, posterior(version));
    await access(new URL(recurso, raiz));
  }
  exigirVersiones(html, "/comun/tema-vec.css", posterior(versionTemaBase));
});

test("la caché immutable del tema anterior descarga ambas hojas de estilo renovadas", async () => {
  const html = await readFile(new URL("index.html", raiz), "utf8");
  const recursos = [
    ["/portal-empleado/portal.css", new URL("portal.css", raiz)],
    ["/comun/tema-vec.css", new URL("../comun/tema-vec.css", raiz)],
  ];
  const anteriores = (ruta) => ruta.endsWith("/portal.css")
    ? [versionSaltoMovilAnterior, versionSaltoMovil] : [version, versionTemaBase];
  const cache = new Map(recursos.flatMap(([ruta]) => anteriores(ruta).map((versionAnterior) =>
    [`${ruta}?v=${versionAnterior}`, `/* respuesta immutable anterior: ${ruta} */`])));
  const descargas = new Set();
  const nuevas = new Set();
  for (const [ruta, archivo] of recursos) {
    const [anteriorInmediata] = anteriores(ruta).slice(-1);
    const versionEsperada = exigirVersiones(html, ruta, posterior(anteriorInmediata));
    const urlNueva = `${ruta}?v=${versionEsperada}`;
    nuevas.add(urlNueva);
    for (const versionAnterior of anteriores(ruta)) {
      const urlAntigua = `${ruta}?v=${versionAnterior}`;
      assert.ok(cache.has(urlAntigua), `${ruta}: la caché antigua está precargada`);
      assert.ok(!html.includes(urlAntigua), `${ruta}: HTML no solicita la URL antigua`);
    }
    if (!cache.has(urlNueva)) {
      cache.set(urlNueva, await readFile(archivo, "utf8"));
      descargas.add(urlNueva);
    }
  }
  assert.deepEqual(descargas, nuevas, "las dos hojas se vuelven a solicitar con URL nueva");
  assert.ok(!html.includes(`/portal-empleado/portal.css?v=${versionTemaBase}`),
    "el HTML deja de solicitar la versión anterior de portal.css");
  assert.ok(!html.includes("/portal-empleado/portal.css?v=20260924-f2-salto-movil-v1"),
    "el HTML deja de solicitar la primera versión del salto móvil");
  assert.ok(!html.includes(`/portal-empleado/portal.css?v=${versionSaltoMovilAnterior}`),
    "el HTML deja de solicitar la versión de salto móvil que movía el menú");
});

test("el grafo JS propio llega desde HTML a los consumidores F2 con versiones nuevas", async () => {
  const [html, portal, coordinador, dietas] = await Promise.all([
    readFile(new URL("index.html", raiz), "utf8"),
    readFile(new URL("portal.js", raiz), "utf8"),
    readFile(new URL("portal-modulos-coordinador.js", raiz), "utf8"),
    readFile(new URL("modulos/dietas/vista-recorridos.js", raiz), "utf8"),
  ]);
  exigirVersiones(html, "/portal-empleado/portal.js", posterior(versionEntradaAyuda));
  exigirVersiones(portal, "./portal-modulos-coordinador.js", posterior(versionDietasShell));
  assert.notEqual(versionDe(html, "/portal-empleado/portal.js"), "20260924-f2-cronos-permisos-v1");
  assert.notEqual(versionDe(portal, "./portal-modulos-coordinador.js"), "20260924-f2-cronos-permisos-v1");
  assert.notEqual(versionDe(html, "/portal-empleado/portal.js"), versionPersonalEstados);
  assert.notEqual(versionDe(portal, "./portal-modulos-coordinador.js"), versionPersonalEstados);
  for (const recurso of ["portal-vistas-operaciones.js"]) {
    assert.equal(versionDe(portal, `./${recurso}`), version, recurso);
    await access(new URL(recurso, raiz));
  }
  // El menú de Bolsa cambió después de F2: sus dos importadores piden la misma
  // URL nueva para no cargar dos instancias del módulo.
  exigirVersiones(portal, "./portal-menu-bolsa.js", posterior(version));
  assert.deepEqual(versionesDe(coordinador, "./portal-menu-bolsa.js"), versionesDe(portal, "./portal-menu-bolsa.js"));
  await access(new URL("portal-menu-bolsa.js", raiz));
  // Cronos interno ya no importa la jornada: solo quedan las vistas conectadas.
  assert.deepEqual(versionesDe(coordinador, "./modulos/cronos/vista.js"), []);
  for (const recurso of [
    "modulos/dietas/cliente-borradores-http.js", "modulos/personal/vista-ficha-integral.js"]) {
    exigirVersiones(coordinador, `./${recurso}`, recurso === "modulos/dietas/cliente-borradores-http.js" ? posterior(version)
      : recurso === "modulos/personal/vista-ficha-integral.js" ? posterior(versionVistasC) : version);
    await access(new URL(recurso, raiz));
  }
  for (const recurso of ["nominas", "solicitudes", "meritos", "comunicaciones", "aprobaciones", "auditoria", "administracion"]) {
    assert.equal(versionesDe(coordinador, `./modulos/${recurso}/vista.js`).length, 0,
      `${recurso}: sin cargador productivo`);
  }
  exigirVersiones(coordinador, "./modulos/dietas/vista-recorridos.js", posterior(versionDietasVista));
  assert.doesNotMatch(coordinador, /modulos\/dietas\/vista-itinerario\.js/u);
  exigirVersiones(dietas, "./vista-borradores-propios.js", posterior(versionDietasVista));
});

test("la caché immutable previa no retiene el catálogo i18n ni los consumidores F2", async () => {
  const versionesPrevias = new Map([
    ["portal.js", ["20260924-web-integrada-v1", version, versionCache, versionCachePersonal, "20260924-p1-personal-interno-v1", versionPersonalInterno, versionPersonalEstados, "20260924-f2-cronos-permisos-v1", versionCronosPermisos, "20260924-f2-dietas-consulta-v2", "20260924-web-c-v3", "20260924-web-c-ayuda-v4", "20260924-web-c-ayuda-v5", versionEntradaAyuda]],
    ["portal-modulos-coordinador.js", ["20260924-web-integrada-v1", version, versionCache, versionCachePersonal, "20260924-p1-personal-interno-v1", versionPersonalInterno, versionPersonalEstados, "20260924-f2-cronos-permisos-v1", versionCronosPermisos, "20260924-f2-dietas-consulta-v2", "20260924-web-c-v3", "20260924-web-c-ayuda-v4", "20260924-web-c-ayuda-v5", versionDietasShell]],
    ["portal-catalogo-modulos.js", ["20260906-acceso-certificado-v1", versionCache, versionCronosPermisos]],
    ["portal-inicio.js", ["20260923-p4-reintento-v2", versionCache, versionCronosPermisos]],
    ["portal-eventos.js", ["20260721-acceso-real-v2", versionCache, versionCronosPermisos, versionEntradaAyuda]],
    ["portal-borradores-ui.js", ["20260921-avisos-r5-v1", versionCache, versionCronosPermisos]],
    ["portal-borradores-acceso.js", ["20260721-acceso-real-v2", versionCache, versionCronosPermisos]],
    ["portal-i18n.js", ["20260721-acceso-real-v2", "20260923-p4-reintento-v2", version, versionCache, versionCronosPermisos, versionEntradaAyuda]],
    ["modulos/cronos/vista-saldo-conectado.js", ["20260925-tanda-v1"]],
    ["modulos/dietas/vista-recorridos.js", [version, "20260924-f2-dietas-consulta-v2", versionDietasRecuperacion, versionDietasIcono, "20260924-dietas-ayuda-sin-guia-v1", versionDietasVista]],
    ["modulos/personal/vista.js", ["20260920-personal-catalogo-v1", versionCachePersonal, versionPersonalEstados]],
    ["modulos/personal/cliente-http-categorias.js", ["20260920-personal-catalogo-v1"]],
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
    ["portal-modulos-coordinador.js", ["portal-catalogo-modulos.js", "portal-inicio.js", "portal-i18n.js",
      "modulos/cronos/vista-saldo-conectado.js", "modulos/dietas/vista-recorridos.js", "modulos/personal/vista.js",
      "modulos/personal/cliente-http-categorias.js"]],
    ["portal-catalogo-modulos.js", ["portal-i18n.js"]],
    ["portal-inicio.js", ["portal-i18n.js"]],
    ["portal-eventos.js", ["portal-i18n.js"]],
    ["portal-borradores-ui.js", ["portal-borradores-acceso.js", "portal-i18n.js"]],
    ["portal-borradores-acceso.js", ["portal-i18n.js"]],
  ]);
  const html = await readFile(new URL("index.html", raiz), "utf8"); // HTML: no-store.
  const pendientes = [["index.html", html]];
  const visitados = new Set();
  const vigentes = new Map();
  while (pendientes.length > 0) {
    const [padre, codigo] = pendientes.shift();
    if (visitados.has(padre)) continue;
    visitados.add(padre);
    for (const hijo of aristas.get(padre) || []) {
      const ruta = padre === "index.html" ? `/portal-empleado/${hijo}` : `./${hijo}`;
      const versionesHijo = versionesDe(codigo, ruta);
      const personal = padre === "portal-modulos-coordinador.js" && hijo.startsWith("modulos/personal/");
      assert.equal(versionesHijo.length, 1,
        `${padre} → ${hijo}: número de aristas`);
      const versionEsperada = padre === "index.html" ? posterior(versionEntradaAyuda) : hijo === "portal-modulos-coordinador.js"
        ? posterior(versionDietasShell) : padre === "portal.js" && ["portal-i18n.js", "portal-eventos.js"].includes(hijo) ? posterior(versionEntradaAyuda)
        : hijo === "modulos/cronos/vista-saldo-conectado.js" ? posterior("20260925-tanda-v1") : hijo === "modulos/dietas/vista-recorridos.js" ? posterior(versionDietasVista) : [
        "portal-catalogo-modulos.js", "portal-inicio.js", "portal-eventos.js",
        "portal-borradores-ui.js", "portal-borradores-acceso.js", "portal-i18n.js"].includes(hijo)
        ? posterior(versionCronosPermisos)
        : hijo === "modulos/personal/vista.js" ? posterior(versionPersonalEstados)
        : hijo === "modulos/personal/cliente-http-categorias.js" ? versionPersonalInterno
        : personal ? versionCachePersonal : versionCache;
      const versionHijoVigente = exigirVersiones(codigo, ruta, versionEsperada, versionesHijo.length);
      assert.ok(!vigentes.has(hijo) || vigentes.get(hijo) === versionHijoVigente,
        `${hijo}: todos sus importadores piden la misma URL`);
      vigentes.set(hijo, versionHijoVigente);
      for (const versionHijo of versionesHijo) {
        const url = `/portal-empleado/${hijo}?v=${versionHijo}`;
        pendientes.push([hijo, await cargar(url)]);
      }
    }
  }
  assert.deepEqual(hitsPrevios, [], "ninguna URL immutable antigua se recupera de caché");
  assert.ok(descargas.has(`/portal-empleado/modulos/personal/cliente-http-categorias.js?v=${versionPersonalInterno}`));
  assert.ok(descargas.has(`/portal-empleado/modulos/dietas/vista-recorridos.js?v=${vigentes.get("modulos/dietas/vista-recorridos.js")}`));
  assert.ok(descargas.has(`/portal-empleado/portal-i18n.js?v=${vigentes.get("portal-i18n.js")}`),
    "el portal carga el catálogo de ayuda y B7 renovado");
  assert.equal(descargas.size, versionesPrevias.size,
    "cada consumidor F2 descarga sus bytes una sola vez: ningún recurso se pide con dos URL");
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
    // portal-bolsas-api.js no ha cambiado desde B7 y conserva su URL; entrada y
    // panel interno sí, y piden una posterior.
    const nueva = exigirVersiones(codigo, prefijo + recurso,
      recurso === "portal-bolsas-api.js" ? versionEntradaAyuda : posterior(versionEntradaAyuda));
    assert.notEqual(nueva, previa, recurso);
    assert.ok(!codigo.includes(`${prefijo}${recurso}?v=${previa}`), `${recurso}: no queda URL antigua`);
    const url = `${recurso}?v=${nueva}`;
    assert.ok(!cache.has(url), `${recurso} debe solicitar nuevos bytes`);
    cache.set(url, await readFile(new URL(recurso, raiz), "utf8"));
    descargas.push(recurso);
  }
  assert.deepEqual(descargas, [...versionesAnteriores.keys()]);
});

test("la recuperación de subsanación renueva toda la cadena immutable y ambas entradas al módulo", async () => {
  const nueva = "20260924-web-subsanacion-v1";
  const html = await readFile(new URL("index.html", raiz), "utf8");
  const portal = await readFile(new URL("portal.js", raiz), "utf8");
  const coordinador = await readFile(new URL("portal-modulos-coordinador.js", raiz), "utf8");
  const pasos = [
    [html, "/portal-empleado/portal.js", "20260924-integracion-b7-v1", 1],
    [portal, "./portal-modulos-coordinador.js", "20260924-p1-personal-interno-v2", 1],
    [coordinador, "./modulos/contratacion-temporal/vista-expedientes.js", "20260923-pweb17-v1", 1],
  ];
  const cache = new Map(pasos.map(([, ruta, previa]) => [`${ruta}?v=${previa}`, "módulo anterior"]));
  for (const [codigo, ruta, previa, cantidad] of pasos) {
    const vigente = exigirVersiones(codigo, ruta, ruta.endsWith("vista-expedientes.js") ? nueva : ruta.endsWith("/portal.js") ? posterior(versionEntradaAyuda) : posterior(versionIntegracion), cantidad);
    assert.ok(!codigo.includes(`${ruta}?v=${previa}`));
    assert.ok(!cache.has(`${ruta}?v=${vigente}`), "la vista anterior no sustituye los bytes nuevos");
  }
});

test("Cronos renueva la importación interna de permisos y su hoja de estilo", async () => {
  const html = await readFile(new URL("index.html", raiz), "utf8");
  const coordinador = await readFile(new URL("portal-modulos-coordinador.js", raiz), "utf8");
  const portal = await readFile(new URL("portal.js", raiz), "utf8");
  const cacheAnterior = new Map([
    ["portal.js?v=20260924-web-integrada-v1", "entrada anterior"],
    ["portal-modulos-coordinador.js?v=20260924-web-integrada-v1", "coordinador anterior"],
  ]);
  const entrada = exigirVersiones(html, "/portal-empleado/portal.js", posterior(versionEntradaAyuda));
  const coordinadorVigente = exigirVersiones(portal, "./portal-modulos-coordinador.js", posterior(versionIntegracion));
  assert.ok(!html.includes("portal.js?v=20260924-web-integrada-v1"));
  assert.ok(!portal.includes("portal-modulos-coordinador.js?v=20260924-web-integrada-v1"));
  assert.ok(!cacheAnterior.has(`portal.js?v=${entrada}`));
  assert.ok(!cacheAnterior.has(`portal-modulos-coordinador.js?v=${coordinadorVigente}`));
  exigirVersiones(coordinador, "./modulos/cronos/vista-permisos-propios.js", posterior("20260925-tanda-v1"), 1);
  exigirVersiones(html, "/portal-empleado/modulos/cronos/permisos.css", posterior(versionCronosAyuda));
  assert.ok(!coordinador.includes("cronos/vista-recorridos.js?v=20260924-f2-cache-v2"));
  assert.ok(!html.includes("cronos/permisos.css?v=20260924-f2-shell-v1"));
});
