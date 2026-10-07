import assert from "node:assert/strict";
import test from "node:test";
import { cargarTextos } from "../comun/textos.js";
import { crearTraductorAccesosEmpleado, crearTraductorResumenAccesosEmpleado, renderizarAccesosEmpleado } from "./portal-accesos-empleado.js";
import { resumenAccesosModulos } from "./portal-menu-bolsa.js";
import { crearTraductorPortal, cargarMensajesPortal } from "./portal-i18n.js";

const escaparHTML = (valor) => String(valor ?? "").replaceAll("&", "&amp;")
  .replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#39;");
const renderizar = (accesos, traducir) => renderizarAccesosEmpleado({ accesos, escaparHTML, traducir });

test("ofrece solo los tres destinos propios enumerados, disponibles o diferidos", () => {
  const html = renderizar({
    personal: { estado: "diferido", vista: "rrhh", href: "https://fuera.invalid" },
    cronos: { estado: "disponible" }, dietas: { estado: "diferido" }, rrhh: { estado: "disponible" },
  });
  assert.match(html, /<h3[^>]*>Mi espacio<\/h3>/u);
  assert.equal((html.match(/<li\b/gu) ?? []).length, 3);
  // La fila común reserva 10 px al marcador; el contenido ocupa su segunda columna.
  assert.equal((html.match(/<li class="elemento-actividad"><span class="marca-actividad" aria-hidden="true"><\/span><div class="acciones-fila">/gu) ?? []).length, 3);
  for (const [vista, etiqueta] of Object.entries({ personal: "Mi carpeta personal", cronos: "Mi jornada", dietas: "Mis dietas" })) {
    assert.match(html, new RegExp(`<a[^>]+href="#${vista}" data-vista="${vista}"[^>]*>[\\s\\S]*?${etiqueta}</a>`, "u"));
  }
  assert.doesNotMatch(html, /https:|rrhh|aria-busy|autorizad|permiso/iu);
});

test("la carga conserva el enlace nativo y anuncia el estado sin afirmar autorización", () => {
  const html = renderizar({ cronos: { estado: "cargando" } });
  assert.match(html, /href="#cronos" data-vista="cronos" aria-busy="true"/u);
  assert.match(html, /role="status">Cargando la vista…<\/span>/u);
  assert.doesNotMatch(html, /disabled|autorizad|permiso/iu);
});

test("el error muestra el destino y recuperación sin ofrecer un enlace", () => {
  const html = renderizar({ dietas: { estado: "error", error: "detalle privado" } });
  assert.match(html, /Mis dietas/u);
  assert.match(html, /role="alert">No se pudo cargar la vista\. Actualice la página para volver a intentarlo\./u);
  assert.match(html, /Mis dietas<\/strong> <span role="alert">/u);
  assert.doesNotMatch(html, /<a\b|data-vista|detalle privado/u);
});

test("omite ausentes, denegados, no disponibles, heredados y estados desconocidos", () => {
  for (const accesos of [undefined, null, {}, { personal: { estado: "denegado" } },
    { cronos: { estado: "no_disponible" } }, { dietas: { estado: "otro" } },
    Object.create({ personal: { estado: "disponible" } })]) {
    assert.equal(renderizar(accesos), "");
  }
  const html = renderizar({ personal: { estado: "denegado" }, cronos: { estado: "disponible" }, dietas: { estado: "no_disponible" } });
  assert.equal((html.match(/<li\b/gu) ?? []).length, 1);
  assert.doesNotMatch(html, /#personal|#dietas|Mi carpeta personal|Mis dietas/u);
});

test("los catálogos completos se traducen con el lector común en ambos idiomas", async () => {
  const textosES = await cargarTextos("accesos-empleado", { idioma: "es", avisar: assert.fail });
  const textosEN = await cargarTextos("accesos-empleado", { idioma: "en", avisar: assert.fail });
  assert.deepEqual(Object.keys(textosES.seccion("accesos")), Object.keys(textosEN.seccion("accesos")));
  for (const textos of [textosES, textosEN]) {
    const traducir = crearTraductorAccesosEmpleado(textos);
    for (const [clave, mensaje] of Object.entries(textos.seccion("accesos"))) assert.equal(traducir(clave), mensaje);
    assert.throws(() => traducir("inexistente"));
  }
  const html = renderizar({ personal: { estado: "disponible" }, cronos: { estado: "cargando" }, dietas: { estado: "error" } }, crearTraductorAccesosEmpleado(textosEN));
  assert.match(html, /My space[\s\S]*My personnel file[\s\S]*My working hours[\s\S]*Loading the view…[\s\S]*My travel expenses[\s\S]*Refresh the page to try again\./u);
  assert.doesNotMatch(html, /Mi espacio|Mi jornada|Cargando/u);
});

test("escapa los textos traducidos en títulos, enlaces y estados", () => {
  const html = renderizar({ personal: { estado: "disponible" }, cronos: { estado: "cargando" }, dietas: { estado: "error" } }, () => '<img src=x onerror="alert(1)"> & \'');
  assert.match(html, /&lt;img src=x onerror=&quot;alert\(1\)&quot;&gt; &amp; &#39;/u);
  assert.doesNotMatch(html, /<img\b|onerror="/u);
});

test("renderizar no consulta datos ni ejecuta transportes de la composición", () => {
  const anterior = globalThis.fetch;
  globalThis.fetch = () => assert.fail("el renderer no debe hacer consultas");
  try {
    const acceso = { estado: "disponible", consultar: () => assert.fail("no debe consultar el destino") };
    Object.defineProperty(acceso, "datos", { get: () => assert.fail("no debe leer datos personales") });
    assert.match(renderizar({ personal: acceso, cronos: acceso, dietas: acceso }), /Mi espacio/u);
  } finally {
    globalThis.fetch = anterior;
  }
});

test("el resumen sin menú remite a Mi espacio y conserva carga, recuento y denegación", async () => {
  for (const idioma of ["es", "en"]) {
    const textos = await cargarTextos("accesos-empleado", { idioma });
    const traducir = crearTraductorPortal(await cargarMensajesPortal(idioma));
    const propio = crearTraductorResumenAccesosEmpleado({ accesos: { personal: { estado: "diferido" } }, traducir, textos });
    assert.equal(resumenAccesosModulos([], false, propio), textos.traducir("accesos.consultar_accesos"));
    for (const [accesos, carga] of [[[], true], [[{ estado: "cargando" }], false], [[{ disponible: true }], false]]) {
      assert.equal(resumenAccesosModulos(accesos, carga, propio), resumenAccesosModulos(accesos, carga, traducir));
    }
    for (const accesos of [{}, { personal: { estado: "denegado" } }, { personal: { estado: "no_disponible" } }]) {
      const sinPropio = crearTraductorResumenAccesosEmpleado({ accesos, traducir, textos });
      assert.equal(resumenAccesosModulos([], false, sinPropio), resumenAccesosModulos([], false, traducir));
    }
  }
});
