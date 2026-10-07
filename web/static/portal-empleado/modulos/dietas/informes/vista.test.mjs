import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { cargarTextos } from "../../../../comun/textos.js";
import { montarInformesDietas, resumirInformesDietas } from "./vista.js";

const datos = JSON.parse(await readFile(new URL("../../../../../../data/demo/dietas/informes.json", import.meta.url), "utf8"));
const configuracion = JSON.parse(await readFile(new URL("../../../../../../data/catalogos/dietas/informes-ejemplo-v1.json", import.meta.url), "utf8"));
const catalogo = await cargarTextos("dietas-informes");
const t = (clave, variables) => catalogo.traducir(`general.${clave}`, variables);
const cargarConfiguracion = async () => configuracion;

class Nodo {
  constructor(documento, etiqueta) {
    this.ownerDocument = documento; this.tagName = etiqueta; this.children = []; this.parent = null;
    this.dataset = {}; this.attrs = {}; this.listeners = {}; this.textContent = ""; this.value = "";
    this.hidden = false; this.disabled = false;
  }
  append(...nodos) { this.children.push(...nodos); nodos.forEach((nodo) => { nodo.parent = this; }); }
  replaceChildren(...nodos) { this.children.forEach((nodo) => { nodo.parent = null; }); this.children = []; this.append(...nodos); }
  remove() { if (this.parent) this.parent.children = this.parent.children.filter((hijo) => hijo !== this); this.parent = null; }
  setAttribute(clave, valor) { this.attrs[clave] = String(valor); }
  removeAttribute(clave) { delete this.attrs[clave]; }
  addEventListener(clave, funcion) { this.listeners[clave] = funcion; }
  removeEventListener(clave) { delete this.listeners[clave]; }
  focus() { this.ownerDocument.activeElement = this; }
  querySelector(selector) { return this.querySelectorAll(selector)[0] ?? null; }
  querySelectorAll(selector) {
    const atributo = selector.match(/^\[data-([a-z-]+)\]$/u)?.[1];
    const propiedad = atributo?.replace(/-([a-z])/gu, (_todo, letra) => letra.toUpperCase());
    const salida = []; const visitar = (nodo) => {
      if ((propiedad && Object.hasOwn(nodo.dataset, propiedad)) || (!propiedad && nodo.tagName === selector)) salida.push(nodo);
      nodo.children.forEach(visitar);
    };
    visitar(this); return salida;
  }
}
function raiz() { const documento = { createElement: (etiqueta) => new Nodo(documento, etiqueta) }; return new Nodo(documento, "root"); }
function texto(nodo) { return [nodo.textContent, ...nodo.children.map(texto)].join(" "); }
const esperar = () => new Promise((resolver) => setImmediate(resolver));

test("el informe suma céntimos conservados y combina persona, unidad y período", () => {
  const todos = resumirInformesDietas(datos.registros, configuracion.criterio);
  assert.equal(todos.registros.length, 8);
  assert.equal(todos.total_centimos, 41420);
  assert.deepEqual(todos.conceptos_centimos, { manutencion: 25500, kilometraje: 11720, otros_gastos: 4200 });
  const filtrado = resumirInformesDietas(datos.registros, configuracion.criterio, {
    persona: "persona-demo-01", unidad: "unidad-demo-01", desde: "2026-09-10", hasta: "2026-09-30",
  });
  assert.deepEqual(filtrado.registros.map((fila) => fila.referencia), ["DI-004"]);
  assert.equal(filtrado.total_centimos, 6090);
  assert.equal(resumirInformesDietas(datos.registros, configuracion.criterio,
    { persona: "persona-demo-01", unidad: "unidad-demo-02" }).registros.length, 0);
});

test("el montaje consulta el JSON inyectado, pagina y mantiene exportar e imprimir cerrados", async () => {
  const contenedor = raiz(); let lecturas = 0;
  const vista = montarInformesDietas(contenedor, { cargarDatos: async () => { lecturas += 1; return datos; },
    cargarConfiguracion, traducir: t, localizacion: catalogo.localizacion });
  await esperar();
  assert.equal(lecturas, 1);
  assert.equal(contenedor.querySelectorAll("button").filter((boton) => [t("exportar"), t("imprimir")].includes(boton.textContent)).every((boton) => boton.disabled), true);
  assert.match(texto(contenedor), /Datos de ejemplo/u);
  const nota = contenedor.querySelectorAll("p").find((parrafo) => parrafo.textContent === t("origen_sintetico"));
  assert.equal(nota.className, "nota-integracion");
  assert.equal(nota.attrs.role, "note");
  assert.equal(contenedor.querySelectorAll("tbody")[0].children.length, 6);
  const listado = contenedor.querySelector("[data-dietas-informes-listado]");
  listado.listeners.click({ target: listado.querySelectorAll("[data-dietas-informes-pagina]")[1] });
  assert.equal(contenedor.querySelectorAll("tbody")[0].children.length, 2);
  assert.match(texto(contenedor.querySelector("[data-dietas-informes-cuenta]")), /7 a 8 de 8/u);
  vista.desmontar(); assert.equal(contenedor.querySelector("[data-dietas-informes]"), null);
});

test("período invertido muestra error y el filtro vacío se puede limpiar", async () => {
  const contenedor = raiz(); const vista = montarInformesDietas(contenedor, {
    cargarDatos: async () => datos, cargarConfiguracion, traducir: t, localizacion: catalogo.localizacion,
  });
  await esperar();
  const form = contenedor.querySelector("[data-dietas-informes-filtros]");
  const fechas = form.querySelectorAll("input"); fechas[0].value = "2026-09-30"; fechas[1].value = "2026-09-01";
  form.listeners.submit({ preventDefault() {} });
  assert.equal(fechas[0].attrs["aria-invalid"], "true");
  assert.equal(fechas[1].attrs["aria-invalid"], "true");
  assert.match(contenedor.querySelector("[data-dietas-informes-estado]").textContent, /fecha inicial/u);
  fechas[0].value = "2027-01-01"; fechas[1].value = "2027-01-31";
  form.listeners.submit({ preventDefault() {} });
  assert.match(texto(contenedor), /No hay informes/u);
  const limpiar = form.querySelectorAll("button")[1]; limpiar.listeners.click();
  assert.equal(contenedor.querySelectorAll("tbody")[0].children.length, 6);
  vista.desmontar();
});

test("el resumen distingue los filtros aplicados de las ediciones aún sin aplicar en ambos idiomas", async () => {
  for (const idioma of ["es", "en"]) {
    const textos = await cargarTextos("dietas-informes", { idioma });
    assert.deepEqual(textos.faltantes, []);
    const traducir = (clave, variables) => textos.traducir(`general.${clave}`, variables);
    const contenedor = raiz();
    const vista = montarInformesDietas(contenedor, { cargarDatos: async () => datos,
      cargarConfiguracion, traducir, localizacion: textos.localizacion });
    await esperar();
    const form = contenedor.querySelector("[data-dietas-informes-filtros]");
    const aplicados = contenedor.querySelector("[data-dietas-informes-aplicados]");
    const [persona, unidad] = form.querySelectorAll("select");
    const [desde, hasta] = form.querySelectorAll("input");
    const aviso = aplicados.children.at(-1);
    assert.equal(aplicados.attrs["aria-label"], traducir("filtros_aplicados"));
    assert.equal(aplicados.children[1].textContent, traducir("sin_filtros"));
    assert.equal(aviso.hidden, true);

    persona.value = "persona-demo-01"; unidad.value = "unidad-demo-01";
    desde.value = "2026-09-10"; hasta.value = "2026-09-30";
    desde.focus(); form.listeners.input();
    assert.equal(aviso.hidden, false);
    assert.equal(contenedor.ownerDocument.activeElement, desde);
    assert.equal(contenedor.querySelectorAll("tbody")[0].children.length, 6);
    form.listeners.submit({ preventDefault() {} });
    const fecha = new Intl.DateTimeFormat(textos.localizacion, { dateStyle: "medium", timeZone: "UTC" });
    assert.equal(aviso.hidden, true);
    assert.match(texto(aplicados), /Ana Molina/u);
    assert.match(texto(aplicados), /Servicios Generales/u);
    assert.ok(texto(aplicados).includes(fecha.format(new Date("2026-09-10T00:00:00Z"))));
    assert.ok(texto(aplicados).includes(fecha.format(new Date("2026-09-30T00:00:00Z"))));
    assert.deepEqual(contenedor.querySelectorAll("tbody")[0].children.map((fila) => fila.children[0].textContent), ["DI-004"]);

    persona.value = "persona-demo-02"; form.listeners.change();
    assert.equal(aviso.hidden, false);
    assert.match(texto(aplicados), /Ana Molina/u);
    assert.doesNotMatch(texto(aplicados), /persona-demo-02/u);
    await vista.recargar();
    assert.equal(persona.value, "persona-demo-01");
    assert.equal(aviso.hidden, true);
    assert.match(texto(aplicados), /Ana Molina/u);
    form.querySelectorAll("button")[1].listeners.click();
    assert.equal(aplicados.children[1].textContent, traducir("sin_filtros"));
    assert.equal(aviso.hidden, true);
    vista.desmontar();
  }
});

test("cada fecha inválida se marca por separado, con error traducido y filtros confirmados conservados", async () => {
  for (const idioma of ["es", "en"]) {
    const textos = await cargarTextos("dietas-informes", { idioma });
    assert.deepEqual(textos.faltantes, []);
    const traducir = (clave, variables) => textos.traducir(`general.${clave}`, variables);
    const contenedor = raiz();
    const vista = montarInformesDietas(contenedor, { cargarDatos: async () => datos, cargarConfiguracion,
      traducir, localizacion: textos.localizacion });
    await esperar();
    const form = contenedor.querySelector("[data-dietas-informes-filtros]");
    const [persona, unidad] = form.querySelectorAll("select");
    const [desde, hasta] = form.querySelectorAll("input");
    const estado = contenedor.querySelector("[data-dietas-informes-estado]");
    persona.value = "persona-demo-01"; unidad.value = "unidad-demo-01";
    desde.value = "2026-09-10"; hasta.value = "2026-09-30";
    form.listeners.submit({ preventDefault() {} });

    for (const [inicio, fin, invalidas] of [
      ["2026-02-30", "2026-09-30", [true, false]],
      ["2026-09-10", "10000-01-01", [false, true]],
      ["2026-02-30", "10000-01-01", [true, true]],
      ["", "2026-02-30", [false, true]],
    ]) {
      desde.value = inicio; hasta.value = fin;
      form.listeners.submit({ preventDefault() {} });
      assert.deepEqual([desde, hasta].map((campo) => campo.attrs["aria-invalid"] === "true"), invalidas);
      assert.equal(estado.textContent, traducir("fecha_error"));
      assert.equal(estado.attrs["aria-live"], "polite");
      assert.equal(contenedor.ownerDocument.activeElement, estado);
      assert.deepEqual(contenedor.querySelectorAll("tbody")[0].children.map((fila) => fila.children[0].textContent), ["DI-004"]);
    }

    await vista.recargar();
    assert.deepEqual([persona.value, unidad.value, desde.value, hasta.value],
      ["persona-demo-01", "unidad-demo-01", "2026-09-10", "2026-09-30"]);
    assert.deepEqual([desde, hasta].map((campo) => campo.attrs["aria-invalid"]), [undefined, undefined]);
    assert.deepEqual(contenedor.querySelectorAll("tbody")[0].children.map((fila) => fila.children[0].textContent), ["DI-004"]);
    desde.value = "2026-02-30";
    form.listeners.submit({ preventDefault() {} });
    form.querySelectorAll("button")[1].listeners.click();
    assert.deepEqual([desde, hasta].map((campo) => campo.attrs["aria-invalid"]), [undefined, undefined]);
    assert.deepEqual([desde.value, hasta.value], ["", ""]);
    assert.equal(contenedor.querySelectorAll("tbody")[0].children.length, 6);
    vista.desmontar();
  }
});

test("recargar conserva solo los filtros aplicados y descarta un período editado inválido", async () => {
  const contenedor = raiz();
  const vista = montarInformesDietas(contenedor, {
    cargarDatos: async () => datos, cargarConfiguracion, traducir: t, localizacion: catalogo.localizacion,
  });
  await esperar();
  const form = contenedor.querySelector("[data-dietas-informes-filtros]");
  const [persona, unidad] = form.querySelectorAll("select");
  const [desde, hasta] = form.querySelectorAll("input");
  persona.value = "persona-demo-01"; unidad.value = "unidad-demo-01";
  desde.value = "2026-09-10"; hasta.value = "2026-09-30";
  form.listeners.submit({ preventDefault() {} });
  assert.deepEqual(contenedor.querySelectorAll("tbody")[0].children.map((fila) => fila.children[0].textContent), ["DI-004"]);

  persona.value = "persona-demo-02"; unidad.value = "unidad-demo-02";
  desde.value = "2026-09-30"; hasta.value = "2026-09-01";
  form.listeners.submit({ preventDefault() {} });
  assert.equal(desde.attrs["aria-invalid"], "true");
  await vista.recargar();
  assert.deepEqual([persona.value, unidad.value, desde.value, hasta.value],
    ["persona-demo-01", "unidad-demo-01", "2026-09-10", "2026-09-30"]);
  assert.equal(desde.attrs["aria-invalid"], undefined);
  assert.deepEqual(contenedor.querySelectorAll("tbody")[0].children.map((fila) => fila.children[0].textContent), ["DI-004"]);
  assert.match(contenedor.querySelector("[data-dietas-informes-estado]").textContent, /1/u);
  vista.desmontar();
});

test("recargar conserva una persona y unidad aplicadas aunque desaparezcan de la fuente", async () => {
  const contenedor = raiz(); let lecturas = 0;
  const vista = montarInformesDietas(contenedor, {
    cargarDatos: async () => {
      lecturas += 1;
      return lecturas === 1 ? datos : { ...datos, registros: datos.registros.filter((fila) => fila.persona_ref !== "persona-demo-04") };
    }, cargarConfiguracion, traducir: t, localizacion: catalogo.localizacion,
  });
  await esperar();
  const form = contenedor.querySelector("[data-dietas-informes-filtros]");
  const [persona, unidad] = form.querySelectorAll("select");
  persona.value = "persona-demo-04"; unidad.value = "unidad-demo-03";
  form.listeners.submit({ preventDefault() {} });
  assert.equal(contenedor.querySelectorAll("tbody")[0].children.length, 2);
  await vista.recargar();
  assert.deepEqual([persona.value, unidad.value], ["persona-demo-04", "unidad-demo-03"]);
  assert.equal(persona.children.find((opcion) => opcion.value === persona.value).textContent, "Eva Torres");
  assert.equal(unidad.children.find((opcion) => opcion.value === unidad.value).textContent, "Medio Ambiente");
  const aplicados = contenedor.querySelector("[data-dietas-informes-aplicados]");
  assert.match(texto(aplicados), /Eva Torres/u);
  assert.match(texto(aplicados), /Medio Ambiente/u);
  assert.equal(contenedor.querySelectorAll("tbody").length, 0);
  assert.match(texto(contenedor), /No hay informes/u);
  form.querySelectorAll("button")[1].listeners.click();
  assert.equal(contenedor.querySelectorAll("tbody")[0].children.length, 6);
  vista.desmontar();
});

test("datos alterados no producen cifras y una nueva carga válida permite recuperarse", async () => {
  const contenedor = raiz(); let intento = 0;
  const vista = montarInformesDietas(contenedor, { cargarDatos: async () => {
    intento += 1;
    return intento === 1 ? { ...datos, registros: [{ ...datos.registros[0], total_centimos: 1 }] } : datos;
  }, cargarConfiguracion, traducir: t, localizacion: catalogo.localizacion });
  await esperar();
  assert.equal(contenedor.querySelectorAll("tbody").length, 0);
  assert.match(contenedor.querySelector("[data-dietas-informes-estado]").textContent, /datos incorrectos/u);
  await vista.recargar();
  assert.equal(contenedor.querySelectorAll("tbody")[0].children.length, 6);
  vista.desmontar();
});

test("una recarga fallida oculta el informe anterior y cierra los filtros hasta recuperar", async () => {
  const contenedor = raiz(); let intento = 0;
  const vista = montarInformesDietas(contenedor, { cargarDatos: async () => {
    intento += 1; if (intento === 2) throw new Error("fuente caída"); return datos;
  }, cargarConfiguracion, traducir: t, localizacion: catalogo.localizacion });
  await esperar(); assert.equal(contenedor.querySelectorAll("tbody")[0].children.length, 6);
  await vista.recargar();
  const form = contenedor.querySelector("[data-dietas-informes-filtros]");
  assert.equal(form.hidden, true); assert.equal(contenedor.querySelectorAll("tbody").length, 0);
  form.listeners.submit({ preventDefault() {} });
  assert.equal(contenedor.querySelectorAll("tbody").length, 0);
  assert.match(contenedor.querySelector("[data-dietas-informes-estado]").textContent, /No se han podido cargar/u);
  await vista.recargar();
  assert.equal(form.hidden, false); assert.equal(contenedor.querySelectorAll("tbody")[0].children.length, 6);
  vista.desmontar();
});

test("el máximo entero seguro conserva sus dos céntimos finales en tarjeta, desglose y las dos columnas de tabla", async () => {
  const maximo = Number.MAX_SAFE_INTEGER;
  const fuente = { ...datos, registros: [{ ...datos.registros[0], total_centimos: maximo,
    conceptos_centimos: { manutencion: maximo, kilometraje: 0, otros_gastos: 0 } }] };
  const contenedor = raiz();
  const vista = montarInformesDietas(contenedor, {
    cargarDatos: async () => fuente, cargarConfiguracion, traducir: t, localizacion: catalogo.localizacion,
  });
  await esperar();
  const exacto = "90.071.992.547.409,91";
  assert.equal(texto(contenedor).split(exacto).length - 1, 4);
  assert.doesNotMatch(texto(contenedor), /90\.071\.992\.547\.409,90/u);
  vista.desmontar();
});

test("la moneda procede de la fuente y nunca mezcla divisas de distintas filas", async () => {
  const maximo = Number.MAX_SAFE_INTEGER;
  const fuente = { ...datos, moneda: "USD", registros: [{ ...datos.registros[0],
    total_centimos: maximo, conceptos_centimos: { manutencion: maximo, kilometraje: 0, otros_gastos: 0 } }] };
  const contenedor = raiz();
  const vista = montarInformesDietas(contenedor, {
    cargarDatos: async () => fuente, cargarConfiguracion, traducir: t, localizacion: catalogo.localizacion,
  });
  await esperar();
  const signo = new Intl.NumberFormat(catalogo.localizacion, { style: "currency", currency: fuente.moneda })
    .formatToParts(0).find((parte) => parte.type === "currency").value;
  assert.match(texto(contenedor), /90\.071\.992\.547\.409,91/u);
  assert.ok(texto(contenedor).includes(signo));
  assert.doesNotMatch(texto(contenedor), /€/u);
  vista.desmontar();

  const mezcla = raiz();
  const vistaMezcla = montarInformesDietas(mezcla, { cargarDatos: async () => ({ ...fuente,
    registros: [{ ...fuente.registros[0], moneda: "EUR" }] }), cargarConfiguracion, traducir: t, localizacion: catalogo.localizacion });
  await esperar();
  assert.equal(mezcla.querySelectorAll("tbody").length, 0);
  assert.match(mezcla.querySelector("[data-dietas-informes-estado]").textContent, /datos incorrectos/u);
  vistaMezcla.desmontar();

  const sinCentimos = raiz();
  const vistaSinCentimos = montarInformesDietas(sinCentimos, { cargarDatos: async () => ({ ...fuente, moneda: "JPY" }),
    cargarConfiguracion, traducir: t, localizacion: catalogo.localizacion });
  await esperar();
  assert.match(sinCentimos.querySelector("[data-dietas-informes-estado]").textContent, /datos incorrectos/u);
  vistaSinCentimos.desmontar();

  const desconocida = raiz();
  const vistaDesconocida = montarInformesDietas(desconocida, { cargarDatos: async () => ({ ...fuente, moneda: "ZZZ" }),
    cargarConfiguracion, traducir: t, localizacion: catalogo.localizacion });
  await esperar();
  assert.equal(desconocida.querySelectorAll("tbody").length, 0);
  assert.match(desconocida.querySelector("[data-dietas-informes-estado]").textContent, /datos incorrectos/u);
  vistaDesconocida.desmontar();
});

test("Reintentar lleva el foco del botón oculto al resumen visible al recuperar", async () => {
  const contenedor = raiz(); let resolver; let intento = 0;
  const vista = montarInformesDietas(contenedor, { cargarDatos: () => {
    intento += 1;
    return intento === 1 ? Promise.reject(new Error("fuente caída"))
      : new Promise((resolve) => { resolver = resolve; });
  }, cargarConfiguracion, traducir: t, localizacion: catalogo.localizacion });
  await esperar();
  const reintentar = contenedor.querySelectorAll("button").find((boton) => boton.textContent === t("reintentar"));
  const estado = contenedor.querySelector("[data-dietas-informes-estado]");
  const resumen = contenedor.querySelector("[data-dietas-informes-resumen]");
  reintentar.focus();
  const carga = reintentar.listeners.click();
  assert.equal(reintentar.hidden, true);
  assert.equal(contenedor.ownerDocument.activeElement, estado);
  resolver(datos); await carga;
  assert.equal(resumen.hidden, false);
  assert.equal(resumen.attrs.tabindex, "-1");
  assert.equal(contenedor.ownerDocument.activeElement, resumen);
  vista.desmontar();
});

test("un catálogo versionado elige fecha, estados y conceptos sin recalcular la comisión", async () => {
  const variante = { ...configuracion, criterio: { campo_fecha: "fecha_liquidacion",
    estados_incluidos: ["liquidado"], conceptos_incluidos: ["manutencion"] } };
  const contenedor = raiz();
  const vista = montarInformesDietas(contenedor, { cargarDatos: async () => datos,
    cargarConfiguracion: async () => variante, traducir: t, localizacion: catalogo.localizacion });
  await esperar();
  const form = contenedor.querySelector("[data-dietas-informes-filtros]");
  const fechas = form.querySelectorAll("input"); fechas[0].value = "2026-09-05"; fechas[1].value = "2026-09-20";
  form.listeners.submit({ preventDefault() {} });
  const filas = contenedor.querySelectorAll("tbody")[0].children;
  assert.deepEqual(filas.map((fila) => fila.children[0].textContent), ["DI-002", "DI-006"]);
  assert.equal(contenedor.querySelectorAll("th").length, 8);
  assert.equal(contenedor.querySelector("[data-dietas-informes-resumen]").querySelectorAll("dd").length, 1);
  assert.match(texto(contenedor.querySelector("[data-dietas-informes-resumen]")), /42,50\s*€/u);
  assert.match(texto(contenedor.querySelector("[data-dietas-informes-ayuda]")), /fecha de liquidación/u);
  assert.equal(contenedor.querySelectorAll("th")[3].textContent, "Liquidación");
  assert.match(filas[0].children.at(-1).textContent, /42,50\s*€/u);
  assert.match(filas[1].children.at(-1).textContent, /0,00\s*€/u);
  vista.desmontar();
});

test("referencia e historia incoherentes cierran la carga del catálogo", async () => {
  for (const mutacion of [
    { referencia: "propuesta:otra" },
    { historia: [{ ...configuracion.historia[0], version: 2 }] },
    { historia: [{ ...configuracion.historia[0], fecha: "2026-02-30T00:00:00Z" }] },
    { historia: [{ ...configuracion.historia[0], actor_nombre: "" }] },
  ]) {
    const contenedor = raiz();
    const vista = montarInformesDietas(contenedor, { cargarDatos: async () => datos,
      cargarConfiguracion: async () => ({ ...configuracion, ...mutacion }),
      traducir: t, localizacion: catalogo.localizacion });
    await esperar();
    assert.equal(contenedor.querySelectorAll("tbody").length, 0);
    assert.match(contenedor.querySelector("[data-dietas-informes-estado]").textContent, /datos incorrectos/u);
    vista.desmontar();
  }
});

test("Ayuda muestra configuración e historia sintética sin abrir referencias", async () => {
  const contenedor = raiz();
  const vista = montarInformesDietas(contenedor, { cargarDatos: async () => datos,
    cargarConfiguracion, traducir: t, localizacion: catalogo.localizacion });
  await esperar();
  const boton = contenedor.querySelectorAll("button").find((n) => n.attrs["aria-label"] === t("ayuda"));
  const ayuda = contenedor.querySelector("[data-dietas-informes-ayuda]");
  assert.equal(ayuda.hidden, true);
  boton.listeners.click();
  assert.equal(boton.attrs["aria-expanded"], "true");
  assert.equal(ayuda.hidden, false);
  assert.match(texto(ayuda), /Configuración de este ejemplo/u);
  assert.match(texto(ayuda), /Marina Torres/u);
  assert.match(texto(ayuda), /Preparación de un informe sintético/u);
  assert.match(texto(ayuda), /No acredita aprobación administrativa/u);
  const motivo = ayuda.querySelectorAll("dd").find((n) => n.textContent === configuracion.historia[0].motivo);
  assert.equal(motivo.attrs.lang, configuracion.historia[0].idioma_motivo);
  const referencias = ayuda.querySelector("details");
  assert.equal(referencias.querySelector("summary").textContent, t("detalle_referencias"));
  assert.notEqual(referencias.open, true);
  boton.listeners.click();
  assert.equal(ayuda.hidden, true);
  assert.equal(boton.attrs["aria-expanded"], "false");
  vista.desmontar();
});

test("una fecha obligatoria ausente cierra la carga sin publicar un total parcial", async () => {
  const casos = [
    { fuente: { ...datos, registros: [{ ...datos.registros[0], fecha_inicio: null }] },
      catalogo: configuracion },
    { fuente: { ...datos, registros: [{ ...datos.registros[1], fecha_liquidacion: null }] },
      catalogo: { ...configuracion, criterio: { ...configuracion.criterio,
        campo_fecha: "fecha_liquidacion", estados_incluidos: ["liquidado"] } } },
  ];
  for (const caso of casos) {
    const contenedor = raiz();
    const vista = montarInformesDietas(contenedor, { cargarDatos: async () => caso.fuente,
      cargarConfiguracion: async () => caso.catalogo, traducir: t, localizacion: catalogo.localizacion });
    await esperar();
    assert.equal(contenedor.querySelectorAll("tbody").length, 0);
    assert.equal(contenedor.querySelector("[data-dietas-informes-resumen]").hidden, true);
    assert.match(contenedor.querySelector("[data-dietas-informes-estado]").textContent, /datos incorrectos/u);
    vista.desmontar();
  }
});

test("filtrar por situación mantiene la intersección del catálogo y sus importes conservados", () => {
  for (const [situacion, cuenta, total] of [["orientativo", 3, 16015], ["liquidado", 3, 13600], ["fiscalizado", 2, 11805]]) {
    const resumen = resumirInformesDietas(datos.registros, configuracion.criterio, { situacion });
    assert.equal(resumen.registros.length, cuenta);
    assert.equal(resumen.total_centimos, total);
    assert.ok(resumen.registros.every((fila) => fila.situacion === situacion));
    assert.equal(Object.values(resumen.conceptos_centimos).reduce((a, b) => a + b, 0), total);
  }
  const limitado = { ...configuracion.criterio, estados_incluidos: ["liquidado"] };
  assert.equal(resumirInformesDietas(datos.registros, limitado, { situacion: "orientativo" }).registros.length, 0);
  assert.equal(resumirInformesDietas(datos.registros, limitado, { situacion: "inventado" }).total_centimos, 0);
  const conjunto = resumirInformesDietas(datos.registros, configuracion.criterio, {
    situacion: "fiscalizado", persona: "persona-demo-01", unidad: "unidad-demo-01", desde: "2026-09-10", hasta: "2026-09-30",
  });
  assert.deepEqual(conjunto.registros.map((fila) => fila.referencia), ["DI-004"]);
  assert.equal(conjunto.total_centimos, 6090);
});

test("situaciones configuradas: aplicar, recargar sin ampliar el filtro y limpiar en ambos idiomas", async () => {
  for (const idioma of ["es", "en"]) {
    const textos = await cargarTextos("dietas-informes", { idioma });
    assert.deepEqual(textos.faltantes, []);
    const traducir = (clave, variables) => textos.traducir(`general.${clave}`, variables);
    let criterio = configuracion;
    const contenedor = raiz();
    const vista = montarInformesDietas(contenedor, { cargarDatos: async () => datos,
      cargarConfiguracion: async () => criterio, traducir, localizacion: textos.localizacion });
    await esperar();
    const form = contenedor.querySelector("[data-dietas-informes-filtros]");
    const situacion = form.querySelectorAll("select")[2];
    const aplicados = contenedor.querySelector("[data-dietas-informes-aplicados]");
    const aviso = aplicados.children.at(-1);
    assert.equal(situacion.parent.textContent, traducir("situacion"));
    assert.equal(situacion.children[0].textContent, traducir("todas_situaciones"));
    assert.deepEqual(new Set(situacion.children.map((opcion) => opcion.value)), new Set(["", ...criterio.criterio.estados_incluidos]));
    const siguiente = contenedor.querySelectorAll("button").find((boton) => boton.dataset.dietasInformesPagina === "1");
    contenedor.querySelector("[data-dietas-informes-listado]").listeners.click({ target: siguiente });
    situacion.value = "liquidado"; situacion.focus(); form.listeners.change();
    assert.equal(aviso.hidden, false);
    assert.equal(contenedor.querySelectorAll("tbody")[0].children.length, 2);
    form.listeners.submit({ preventDefault() {} });
    assert.equal(aviso.hidden, true);
    assert.equal(contenedor.ownerDocument.activeElement, situacion);
    assert.ok(texto(aplicados).includes(traducir("situacion_liquidado")));
    assert.deepEqual(contenedor.querySelectorAll("tbody")[0].children.map((fila) => fila.children[0].textContent), ["DI-002", "DI-006", "DI-007"]);
    assert.equal(contenedor.querySelectorAll("button").find((boton) => boton.dataset.dietasInformesPagina === "-1").disabled, true);
    situacion.value = "orientativo"; form.listeners.change();
    await vista.recargar();
    assert.equal(situacion.value, "liquidado");
    assert.equal(aviso.hidden, true);
    criterio = { ...configuracion, criterio: { ...configuracion.criterio, estados_incluidos: ["orientativo"] } };
    await vista.recargar();
    assert.equal(situacion.value, "liquidado");
    assert.equal(situacion.children.find((opcion) => opcion.value === "liquidado").textContent, traducir("situacion_liquidado"));
    assert.ok(texto(aplicados).includes(traducir("situacion_liquidado")));
    assert.equal(contenedor.querySelectorAll("tbody").length, 0);
    assert.ok(texto(contenedor).includes(traducir("vacio")));
    assert.equal(contenedor.querySelector("[data-dietas-informes-estado]").textContent, traducir("resultado_filtro", { cuenta: "0" }));
    form.querySelectorAll("button")[1].listeners.click();
    assert.equal(situacion.value, "");
    assert.equal(contenedor.querySelectorAll("tbody")[0].children.length, 3);
    assert.equal(aplicados.children[1].textContent, traducir("sin_filtros"));
    for (const clave of ["exportar", "imprimir"]) assert.equal(contenedor.querySelectorAll("button").find((boton) => boton.textContent === traducir(clave)).disabled, true);
    vista.desmontar();
  }
});
