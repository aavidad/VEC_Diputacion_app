import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { readFileSync } from "node:fs";
import test from "node:test";
import { exigirRenovado, versionDe } from "../versiones-cache.test-helper.mjs";
import {
  API_REGLAS, ErrorReglas, crearCliente, detalleRegla, filtrar, idRegla, iniciar, mensajeError, origenRegla, renderizarCatalogo,
  renderizarResumen, validarReglas, valorRegla,
} from "./reglas.js";
import { MENSAJES_REGLAS, crearTraductorReglas } from "./i18n.js";

const leer = (nombre) => readFileSync(new URL(nombre, import.meta.url), "utf8");

function regla(extra = {}) {
  return { clave: "b05.plazo_respuesta", etiqueta: "Plazo de respuesta", descripcion: "Un día hábil.", unidad: "dias_habiles",
    cantidad: 1, computo: "administrativo", inicio: "contacto_efectivo", origen: "ejemplo", norma: "Supuesto de trabajo.",
    duda: "Duda 1 de RRHH.", version: 1, referencia: "vec.bolsa.reglas:1:b05.plazo_respuesta", paquete_ejemplo: true, ...extra };
}

function respuesta() {
  return { data: { esquema: "vec.reglas.vigentes.v1", catalogos: [
    { modulo: "bolsa", catalogo_id: "vec.bolsa.reglas", estado: "disponible", version: 1, huella_sha256: "a".repeat(64), paquete_ejemplo: true,
      reglas: [regla(), regla({ clave: "b04.franja_llamadas", etiqueta: "Franja <b>", unidad: "franja_horaria", cantidad: undefined, computo: undefined,
        valor: "09:00-14:00", origen: "reglamento", articulo: "art. 8.2.a", duda: "Sin duda abierta." })] },
    { modulo: "contratacion_temporal", catalogo_id: "vec.contratacion_temporal.reglas", estado: "sin_catalogo", paquete_ejemplo: false, reglas: [] },
  ] } };
}

test("el catálogo i18n está completo y toda clave de la página existe", () => {
  const t = crearTraductorReglas();
  assert.throws(() => crearTraductorReglas({ titulo: "x" }), /incompleto/u);
  assert.throws(() => crearTraductorReglas({ ...MENSAJES_REGLAS, ayudaResolver: "" }), /incompleto/u);
  assert.throws(() => crearTraductorReglas({ ...MENSAJES_REGLAS, parteEjemplo: "Sin dato" }), /incompleto/u);
  assert.throws(() => t("desconocida"), /desconocida/u);
  const html = leer("./index.html");
  for (const [, clave] of html.matchAll(/data-i18n(?:-label)?="([^"]+)"/gu)) assert.ok(Object.hasOwn(MENSAJES_REGLAS, clave), clave);
  const js = leer("./reglas.js");
  for (const [, id] of js.matchAll(/\$\("([a-z-]+)"\)/gu)) assert.match(html, new RegExp(`id="${id}"`, "u"), id);
  assert.ok(!/style=|<script>/u.test(html), "sin estilos ni guiones en línea");
  assert.match(html, /id="rg-ayuda-abrir"[^>]*>\?</u, "la ayuda solo se abre con «?»");
});

test("el catálogo renovado usa una URL única en la pantalla y en sus consumidores de Bolsa y CT", () => {
  const html = leer("./index.html");
  const reglas = leer("./reglas.js");
  const version = exigirRenovado([html, reglas], "i18n.js", "20260930-reglas-detalle-v3");
  assert.equal(version, "20261007-p5-solicitudes-reglas-v1");
  const bolsa = leer("../modulos/bolsa/rrhh-plazos-api.js");
  const etiquetas = leer("../modulos/contratacion-temporal/etiquetas-vias-cobertura.js");
  assert.equal(exigirRenovado([html, bolsa, etiquetas], "reglas.js", "20260930-reglas-detalle-v3"), version);
  // La pantalla de plazos lee las reglas a través de rrhh-plazos-api.js (una sola lectura compartida).
  assert.doesNotMatch(leer("../modulos/bolsa/rrhh-plazos-ui.js"), /reglas\/reglas\.js/u);
  const formulario = leer("../modulos/contratacion-temporal/formulario-cobertura.js");
  assert.equal(exigirRenovado(formulario, "etiquetas-vias-cobertura.js", "20260930-reglas-detalle-v3"), version);
  // El enlace renueva su URL con el catálogo común del portal.
  const render = leer("../modulos/contratacion-temporal/vista-expedientes-render.js");
  assert.equal(exigirRenovado(render, "enlace.js", "20260930-reglas-detalle-v1"), "20261001-ct-a-i18n-v1");
});

test("las versiones en caché se renuevan juntas y la pantalla está en el manifiesto interno", () => {
  const html = leer("./index.html");
  const js = leer("./reglas.js");
  assert.equal(versionDe(html, "reglas.js"), versionDe(html, "reglas.css"));
  assert.equal(versionDe(js, "i18n.js"), versionDe(html, "i18n.js"));
  assert.equal(versionDe(js, "i18n.js"), versionDe(html, "reglas.js"));
  for (const nombre of ["interno.manifest", "produccion.manifest"]) {
    const manifiesto = readFileSync(new URL(`../../../${nombre}`, import.meta.url), "utf8");
    for (const f of ["index.html", "reglas.js", "reglas.css", "i18n.js"]) assert.match(manifiesto, new RegExp(`static/portal-empleado/reglas/${f.replace(".", "\\.")}`, "u"), `${nombre}: ${f}`);
  }
});

test("valida el contrato y rechaza lo que no encaja", () => {
  const d = validarReglas(respuesta());
  assert.equal(d.catalogos.length, 2);
  assert.throws(() => validarReglas({ data: { esquema: "otro", catalogos: [] } }), ErrorReglas);
  const sinArticulo = respuesta();
  sinArticulo.data.catalogos[0].reglas[1].articulo = undefined;
  assert.throws(() => validarReglas(sinArticulo), ErrorReglas);
  const ejemploConArticulo = respuesta();
  ejemploConArticulo.data.catalogos[0].reglas[0].articulo = "art. 1";
  assert.throws(() => validarReglas(ejemploConArticulo), ErrorReglas);
  const sinCatalogoConReglas = respuesta();
  sinCatalogoConReglas.data.catalogos[1].reglas = [regla()];
  assert.throws(() => validarReglas(sinCatalogoConReglas), ErrorReglas);
});

test("pinta valor, unidad, origen, duda y versión, escapando el contenido", () => {
  const d = validarReglas(respuesta());
  const html = renderizarCatalogo(d.catalogos[0]);
  assert.match(html, /Reglamento, art\. 8\.2\.a/u);
  assert.match(html, /09:00–14:00/u);
  assert.match(html, /Días hábiles/u);
  assert.match(html, /Cómputo administrativo/u);
  assert.match(html, /Duda 1 de RRHH\./u);
  assert.match(html, /Paquete de ejemplo/u);
  assert.match(html, /Franja &lt;b&gt;/u);
  assert.ok(!html.includes("<b>"));
  assert.match(renderizarCatalogo(d.catalogos[1]), /Sin catálogo de reglas cargado/u);
  assert.equal(valorRegla({ unidad: "ninguna" }), "No aplica");
  assert.equal(valorRegla({ unidad: "lista", valor: "a,b" }), "a, b");
  assert.equal(origenRegla({ origen: "ejemplo" }), "Ejemplo");
  const resumen = renderizarResumen(d.catalogos);
  assert.match(resumen, /Reglas vigentes<\/span><strong>2</u);
  assert.match(resumen, /Del Reglamento<\/span><strong>1</u);
});

test("filtra por módulo, origen y texto", () => {
  const d = validarReglas(respuesta());
  assert.equal(filtrar(d.catalogos, { modulo: "bolsa" }).length, 1);
  assert.equal(filtrar(d.catalogos, { origen: "reglamento" })[0].reglas.length, 1);
  assert.equal(filtrar(d.catalogos, { texto: "DUDA 1" })[0].reglas.length, 1);
});

test("el cliente pide solo lectura, sin identidad propia, y traduce los errores", async () => {
  let pedido;
  const ok = crearCliente(async (url, opciones) => {
    pedido = { url, opciones };
    return new Response(JSON.stringify(respuesta()), { status: 200 });
  });
  await ok.reglas();
  assert.equal(pedido.url, API_REGLAS);
  assert.equal(pedido.opciones.method, "GET");
  assert.deepEqual(Object.keys(pedido.opciones.headers), ["Accept"]);
  for (const [estado, codigo] of [[401, "error_autenticacion_requerida"], [403, "error_acceso_denegado"], [503, "error_servicio_no_disponible"]]) {
    const cliente = crearCliente(async () => new Response("{}", { status: estado }));
    await assert.rejects(cliente.reglas(), (e) => e.codigo === codigo && mensajeError(e).length > 0);
  }
  const roto = crearCliente(async () => new Response("no json", { status: 200 }));
  await assert.rejects(roto.reglas(), (e) => e.codigo === "error_respuesta");
  const caido = crearCliente(async () => { throw new TypeError("red"); });
  await assert.rejects(caido.reglas(), (e) => e.codigo === "error_servicio_no_disponible");
});

test("cada regla se abre para leerla entera: descripción, origen, norma y duda, sin códigos internos", () => {
  const d = validarReglas(respuesta());
  const larga = "Pregunta larga de RRHH ".repeat(30).trim();
  const lista = regla({ clave: "b30.documentos", unidad: "lista", cantidad: undefined, valor: "dni, titulo <x>,carnet", duda: larga });
  const html = renderizarCatalogo({ ...d.catalogos[0], reglas: [lista] });
  const id = idRegla("bolsa", "b30.documentos");
  assert.match(id, /^rg-regla-[a-z0-9-]+$/u);
  assert.match(html, new RegExp(`<button type="button" class="rg-regla-abrir" aria-expanded="false" aria-controls="${id}">`, "u"));
  assert.match(html, new RegExp(`<tr class="rg-detalle rg-fila--ejemplo" id="${id}" hidden>`, "u"));
  const detalle = detalleRegla(lista);
  for (const texto of ["Qué establece", "Un día hábil.", "Norma en que se basa", "Supuesto de trabajo.", "Duda de RRHH", larga]) {
    assert.ok(detalle.includes(texto), texto);
  }
  for (const codigo of ["vec.bolsa.reglas:1", "b30.documentos", "dni", "carnet"]) assert.ok(!detalle.includes(codigo), `sin código interno: ${codigo}`);
  assert.match(detalleRegla(regla({ descripcion: "" })), /El catálogo no describe esta regla/u);
  const abierta = renderizarCatalogo({ ...d.catalogos[0], reglas: [lista] }, new Set([id]));
  assert.match(abierta, /aria-expanded="true"/u);
  assert.ok(!abierta.includes(`id="${id}" hidden`));
});

test("claves válidas distintas conservan identificadores únicos y los datos no se traducen", () => {
  const claves = ["b04.franja", "b04-franja", "b04_franja", "b04:franja"];
  const ids = claves.map((clave) => idRegla("bolsa", clave));
  assert.equal(new Set(ids).size, claves.length);
  assert.notEqual(idRegla("bolsa-a", "b"), idRegla("bolsa", "a-b"));
  const html = renderizarCatalogo({ ...validarReglas(respuesta()).catalogos[0], reglas: [
    regla({ clave: claves[0], ejemplo_parcial: "Solo personal fijo" }),
    regla({ clave: claves[1], ejemplo_parcial: "Solo temporal" }),
  ] });
  for (const id of ids.slice(0, 2)) {
    assert.match(html, new RegExp(`aria-controls="${id}"`, "u"));
    assert.match(html, new RegExp(`id="${id}"`, "u"));
  }
  assert.ok(!html.includes(">b04.franja<"), "la clave interna no se ve en pantalla");
  assert.match(html, /Parte de ejemplo: <span>Solo personal fijo<\/span>/u);
});

test("la vista inglesa marca como españoles solo los datos recibidos", () => {
  const modulo = new URL("./reglas.js", import.meta.url).href;
  const idioma = new URL("../../comun/idioma.js", import.meta.url).href;
  const codigo = `globalThis.location = { href: "http://localhost/portal-empleado/reglas/?lang=en" };
    await (await import(${JSON.stringify(idioma)})).prepararIdiomas();
    const { renderizarCatalogo } = await import(${JSON.stringify(modulo)});
    const regla = ${JSON.stringify(regla({ ejemplo_parcial: "Solo personal fijo" }))};
    process.stdout.write(renderizarCatalogo({ modulo: "bolsa", catalogo_id: "vec.bolsa.reglas",
      estado: "disponible", paquete_ejemplo: true, reglas: [regla] }));`;
  const html = execFileSync(process.execPath, ["--input-type=module", "-e", codigo], { encoding: "utf8" });
  assert.match(html, /<span class="rg-regla-nombre" lang="es">Plazo de respuesta<\/span>/u);
  assert.match(html, /Example part: <span lang="es">Solo personal fijo<\/span>/u);
  assert.ok(!html.includes(">b05.plazo_respuesta<"), "la clave interna no se ve en pantalla");
  assert.doesNotMatch(html, /lang="es">Example part:/u);
});

test("los textos de la pantalla vienen de los catálogos es y en con las mismas claves", () => {
  const es = JSON.parse(readFileSync(new URL("../../textos/es/reglas.json", import.meta.url), "utf8")).general;
  const en = JSON.parse(readFileSync(new URL("../../textos/en/reglas.json", import.meta.url), "utf8")).general;
  assert.deepEqual(Object.keys(en).sort(), Object.keys(es).sort());
  assert.deepEqual(MENSAJES_REGLAS, es);
  assert.ok(!/"[a-z_]+":\s*"/u.test(leer("./i18n.js")), "sin diccionario en el código");
  for (const [clave, texto] of Object.entries(es)) {
    const vars = (v) => [...v.matchAll(/\{([a-z_]+)\}/gu)].map((m) => m[1]).sort().join();
    assert.equal(vars(en[clave]), vars(texto), clave);
  }
});

/** DOM mínimo para recorrer la pantalla sin navegador: suficiente para abrir, filtrar y recargar. */
function documentoFalso(hash = "") {
  const html = leer("./index.html");
  const nodos = new Map();
  let doc;
  const crearNodo = (id) => {
    const oyentes = {};
    const nodo = { id, hidden: id === "rg-resultado", value: "", innerHTML: "", classList: { toggle() {} },
      hijos: [], atributos: {},
      addEventListener: (tipo, f) => { oyentes[tipo] = f; }, oyentes,
      setAttribute: (clave, valor) => { nodo.atributos[clave] = valor; },
      append: (...hijos) => nodo.hijos.push(...hijos),
      focus: () => { doc.activeElement = nodo; },
    };
    let contenido = "";
    Object.defineProperty(nodo, "textContent", {
      get: () => contenido,
      set: (valor) => {
        if (nodo.hijos.includes(doc.activeElement)) doc.activeElement = doc.body;
        contenido = valor;
        nodo.hijos = [];
      },
    });
    return nodo;
  };
  for (const [, id] of html.matchAll(/id="([a-z-]+)"/gu)) {
    nodos.set(id, crearNodo(id));
  }
  const estado = { hash, reemplazos: [], recargas: 0 };
  const vista = { location: { get hash() { return estado.hash; }, pathname: "/portal-empleado/reglas/", search: "", reload: () => { estado.recargas++; } },
    history: { replaceState: (_e, _t, url) => { estado.reemplazos.push(url); estado.hash = url.startsWith("#") ? url : ""; } } };
  doc = {
    body: {}, createElement: (tag) => crearNodo(tag),
    title: "", documentElement: { lang: "" }, defaultView: vista,
    getElementById: (id) => nodos.get(id) ?? detalles.get(id) ?? null,
    querySelectorAll: () => [], getSelection: () => ({ toString: () => "" }),
  };
  doc.activeElement = doc.body;
  const detalles = new Map();
  return { doc, nodos, estado, detalles };
}

function filaFalsa(id, detalles) {
  const atributos = {};
  const clases = new Set();
  const detalle = { hidden: true, previousElementSibling: { scrollIntoView() {} } };
  detalles.set(id, detalle);
  const boton = { setAttribute: (k, v) => { atributos[k] = v; }, closest: (s) => (s === ".rg-regla-abrir" ? boton : s === "tr.rg-fila" ? fila : null) };
  const fila = { dataset: { regla: id }, classList: { toggle: (c, v) => (v ? clases.add(c) : clases.delete(c)) }, querySelector: () => boton };
  return { fila, boton, detalle, atributos, clases };
}

test("pulsar el nombre abre y cierra el detalle, lo anota en el ancla y la recarga lo conserva", async () => {
  const cliente = { reglas: async () => validarReglas(respuesta()) };
  const { doc, nodos, estado, detalles } = documentoFalso();
  await iniciar(doc, cliente);
  assert.equal(nodos.get("rg-resultado").hidden, false);
  const id = idRegla("bolsa", "b05.plazo_respuesta");
  assert.match(nodos.get("rg-catalogos").innerHTML, new RegExp(`aria-controls="${id}"`, "u"));
  const { boton, detalle, atributos, clases } = filaFalsa(id, detalles);
  nodos.get("rg-catalogos").oyentes.click({ target: boton });
  assert.equal(detalle.hidden, false);
  assert.equal(atributos["aria-expanded"], "true");
  assert.ok(clases.has("rg-fila--abierta"));
  assert.equal(estado.hash, `#${id}`);
  // Filtrar vuelve a pintar y la regla sigue abierta.
  nodos.get("rg-texto").value = "plazo";
  nodos.get("rg-texto").oyentes.input();
  assert.match(nodos.get("rg-catalogos").innerHTML, new RegExp(`aria-controls="${id}"`, "u"));
  assert.ok(!nodos.get("rg-catalogos").innerHTML.includes(`id="${id}" hidden`));
  // Recargar con el ancla: la regla aparece ya abierta.
  const recarga = documentoFalso(`#${id}`);
  await iniciar(recarga.doc, cliente);
  assert.ok(recarga.nodos.get("rg-catalogos").innerHTML.includes(`aria-expanded="true" aria-controls="${id}"`));
  // Volver a pulsar la cierra y limpia el ancla.
  nodos.get("rg-catalogos").oyentes.click({ target: boton });
  assert.equal(detalle.hidden, true);
  assert.equal(estado.hash, "");
  // Un ancla ajena no abre nada ni se inyecta.
  const ajena = documentoFalso("#<img src=x>");
  await iniciar(ajena.doc, cliente);
  assert.ok(!ajena.nodos.get("rg-catalogos").innerHTML.includes('aria-expanded="true"'));
  const malformada = documentoFalso("#%");
  await iniciar(malformada.doc, cliente);
  assert.equal(malformada.nodos.get("rg-resultado").hidden, false);
  assert.match(malformada.nodos.get("rg-catalogos").innerHTML, /aria-expanded="false"/u);
});

test("la cabecera del catálogo muestra la versión sin identificadores ni huellas", () => {
  const html = renderizarCatalogo(validarReglas(respuesta()).catalogos[0]);
  assert.match(html, /<p class="rg-meta">[^<]*1[^<]*<\/p>/u);
  for (const codigo of ["vec.bolsa.reglas", "aaaaaaaaaaaa"]) assert.ok(!html.includes(codigo), `sin código interno: ${codigo}`);
});


test("Reintentar conserva el foco con un error persistente y consulta sin recargar la página", async () => {
  const { doc, nodos, estado } = documentoFalso();
  let llamadas = 0;
  let terminar;
  const cliente = { reglas: async () => {
    llamadas++;
    if (llamadas > 1) await new Promise((resolver) => { terminar = resolver; });
    throw new ErrorReglas("error_servicio_no_disponible");
  } };
  await iniciar(doc, cliente);
  const aviso = nodos.get("rg-estado");
  assert.match(leer("./index.html"), /id="rg-estado"[^>]*tabindex="-1"[^>]*role="status"[^>]*aria-live="polite"/u);
  assert.equal(doc.activeElement, doc.body, "la carga inicial no mueve el foco");
  for (let intento = 0; intento < 2; intento++) {
    const boton = aviso.hijos.at(-1);
    assert.equal(boton.textContent, MENSAJES_REGLAS.reintentar);
    boton.focus();
    const consulta = boton.oyentes.click();
    assert.equal(doc.activeElement, aviso, "el aviso conserva el foco mientras espera");
    assert.equal(aviso.textContent, MENSAJES_REGLAS.cargando);
    terminar();
    await consulta;
    assert.equal(doc.activeElement, aviso.hijos.at(-1), "el nuevo botón recibe el foco");
    assert.notEqual(aviso.hijos.at(-1), boton);
    assert.equal(nodos.get("rg-resultado").hidden, true);
    assert.equal(aviso.hidden, false);
    assert.equal(aviso.textContent, MENSAJES_REGLAS.error_servicio_no_disponible);
  }
  assert.equal(llamadas, 3);
  assert.equal(estado.recargas, 0);
});

test("Reintentar recupera las reglas y continúa el teclado en el primer filtro", async () => {
  const { doc, nodos, estado } = documentoFalso();
  let llamadas = 0;
  const cliente = { reglas: async () => {
    if (++llamadas === 1) throw new ErrorReglas("error_servicio_no_disponible");
    return validarReglas(respuesta());
  } };
  await iniciar(doc, cliente);
  const boton = nodos.get("rg-estado").hijos.at(-1);
  boton.focus();
  await boton.oyentes.click();
  assert.equal(nodos.get("rg-resultado").hidden, false);
  assert.equal(nodos.get("rg-estado").hidden, true);
  assert.equal(doc.activeElement, nodos.get("rg-modulo"));
  assert.match(nodos.get("rg-catalogos").innerHTML, /Plazo de respuesta/u);
  assert.equal(llamadas, 2);
  assert.equal(estado.recargas, 0);
});

test("el reintento respeta el foco si la persona cambia de control durante la espera", async () => {
  for (const recuperado of [false, true]) {
    const { doc, nodos } = documentoFalso();
    let terminar;
    let llamadas = 0;
    const cliente = { reglas: async () => {
      if (++llamadas > 1) await new Promise((resolver) => { terminar = resolver; });
      if (recuperado && llamadas > 1) return validarReglas(respuesta());
      throw new ErrorReglas("error_servicio_no_disponible");
    } };
    await iniciar(doc, cliente);
    const boton = nodos.get("rg-estado").hijos.at(-1);
    boton.focus();
    const consulta = boton.oyentes.click();
    const ayuda = nodos.get("rg-ayuda-abrir");
    ayuda.focus();
    terminar();
    await consulta;
    assert.equal(doc.activeElement, ayuda);
  }
});

function catalogosDemo() {
  return ["bolsa", "ct"].map((nombre) => {
    const { catalogo } = JSON.parse(leer(`../../../../data/demo/reglas/${nombre}_reglas.ejemplo.demo.json`));
    return { modulo: catalogo.modulo_id, catalogo_id: catalogo.id, estado: "disponible", version: catalogo.version,
      paquete_ejemplo: true, reglas: catalogo.entradas.map((entrada) => regla({
        clave: entrada.clave, etiqueta: entrada.etiqueta, descripcion: entrada.descripcion,
        ...entrada.atributos, cantidad: entrada.atributos.cantidad ? Number(entrada.atributos.cantidad) : undefined,
      })) };
  });
}

test("las 23 reglas se presentan y se buscan en ES/EN sin modificar la definición recibida", () => {
  const modulo = new URL("./reglas.js", import.meta.url).href;
  const moduloIdioma = new URL("../../comun/idioma.js", import.meta.url).href;
  const catalogos = catalogosDemo();
  for (const idioma of ["es", "en"]) {
    const presentacion = JSON.parse(leer(`../../textos/${idioma}/reglas.json`)).presentacion;
    const codigo = `
      import assert from "node:assert/strict";
      globalThis.location = { href: "http://localhost/portal-empleado/reglas/?lang=${idioma}" };
      await (await import(${JSON.stringify(moduloIdioma)})).prepararIdiomas();
      const { filtrar, renderizarCatalogo } = await import(${JSON.stringify(modulo)});
      const catalogos = ${JSON.stringify(catalogos)};
      const presentacion = ${JSON.stringify(presentacion)};
      const preimagen = JSON.stringify(catalogos);
      for (const catalogo of catalogos) {
        catalogo.reglas.forEach(Object.freeze);
        Object.freeze(catalogo.reglas);
        Object.freeze(catalogo);
      }
      Object.freeze(catalogos);
      const escapar = (texto) => texto.replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;")
        .replaceAll('"', "&quot;").replaceAll("'", "&#39;");
      let comprobadas = 0;
      for (const catalogo of catalogos) {
        const html = renderizarCatalogo(catalogo);
        for (const regla of catalogo.reglas) {
          const clave = regla.clave;
          const campos = clave.split(".").reduce((nodo, parte) => nodo?.[parte], presentacion[catalogo.modulo]);
          if (!campos) continue;
          for (const [campo, entrada] of Object.entries(campos)) {
            assert.equal(regla[campo], entrada.original, clave + ": " + campo);
            assert.ok(html.includes("<span>" + escapar(entrada.texto) + "</span>"), clave + ": " + campo);
            const filtrado = filtrar(catalogos, { texto: entrada.texto }).find((c) => c.modulo === catalogo.modulo);
            assert.ok(filtrado.reglas.includes(regla), clave + ": búsqueda por texto presentado");
          }
          comprobadas++;
        }
      }
      assert.equal(JSON.stringify(catalogos), preimagen);
      process.stdout.write(String(comprobadas));`;
    assert.equal(execFileSync(process.execPath, ["--input-type=module", "-e", codigo], { encoding: "utf8" }), "23");
  }
});

test("la presentación de una regla no cambia otro campo ni una definición nueva", () => {
  const modulo = new URL("./reglas.js", import.meta.url).href;
  const idioma = new URL("../../comun/idioma.js", import.meta.url).href;
  const fuente = catalogosDemo().find((c) => c.modulo === "contratacion_temporal");
  const original = fuente.reglas.find((r) => r.clave === "c20.cancelacion_expediente");
  const en = JSON.parse(leer("../../textos/en/reglas.json")).presentacion.contratacion_temporal.c20.cancelacion_expediente;
  const codigo = `
    import assert from "node:assert/strict";
    globalThis.location = { href: "http://localhost/portal-empleado/reglas/?lang=en" };
    await (await import(${JSON.stringify(idioma)})).prepararIdiomas();
    const { detalleRegla, filtrar } = await import(${JSON.stringify(modulo)});
    const regla = ${JSON.stringify(original)};
    const html = detalleRegla(regla, "contratacion_temporal");
    assert.ok(html.includes("<span>" + ${JSON.stringify(en.descripcion.texto)} + "</span>"));
    assert.ok(html.includes('<span lang="es">' + regla.norma + '</span>'));
    const nueva = Object.freeze({ ...regla, descripcion: "Cambio de fuente <img src=x>" });
    const sinSustituir = detalleRegla(nueva, "contratacion_temporal");
    assert.ok(sinSustituir.includes('<span lang="es">Cambio de fuente &lt;img src=x&gt;</span>'));
    assert.ok(!sinSustituir.includes(${JSON.stringify(en.descripcion.texto)}));
    const catalogos = [{ modulo: "contratacion_temporal", reglas: [nueva] }];
    assert.equal(filtrar(catalogos, { texto: "Cambio de fuente" })[0].reglas[0], nueva);
    assert.equal(filtrar(catalogos, { texto: ${JSON.stringify(en.descripcion.texto)} })[0].reglas.length, 0);`;
  execFileSync(process.execPath, ["--input-type=module", "-e", codigo]);
});
