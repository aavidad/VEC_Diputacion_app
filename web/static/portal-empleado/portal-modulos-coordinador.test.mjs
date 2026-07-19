import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { obtenerDatosPresentacion } from "./datos-presentacion.js";
import { crearAdaptadorPresentacion } from "./portal-presentacion-adaptador.js";
import {
  crearCoordinadorModulosPortal,
  moduloDeVistaPortal,
  rutaDeVistaPortal,
} from "./portal-modulos-coordinador.js";

function raizFalsa() {
  const eventos = new Map();
  return {
    innerHTML: "",
    addEventListener(tipo, manejador) { eventos.set(tipo, manejador); },
    removeEventListener(tipo, manejador) {
      if (eventos.get(tipo) === manejador) eventos.delete(tipo);
    },
    replaceChildren() { this.innerHTML = ""; },
    contains() { return true; },
    querySelector() { return null; },
    querySelectorAll() { return []; },
    setAttribute() {},
    removeAttribute() {},
  };
}

function crearCoordinador() {
  return crearCoordinadorModulosPortal({
    escaparHTML: (valor) => String(valor).replaceAll("&", "&amp;").replaceAll("<", "&lt;"),
    anunciar: () => {},
    confirmarOperacion: () => true,
    entorno: { location: { origin: "http://127.0.0.2:8081" } },
  });
}

test("Bolsa, Cronos y Dietas comparten exactamente el ContextoActor del arranque", async () => {
  const datosBolsa = obtenerDatosPresentacion("administrador");
  const coordinador = crearCoordinador();
  const contextoBolsa = await coordinador.cargarPresentacion(datosBolsa.sesion);
  const adaptadorBolsa = crearAdaptadorPresentacion({ datosIniciales: datosBolsa, contextoActor: contextoBolsa });
  assert.equal(adaptadorBolsa.identidad, contextoBolsa);
  assert.equal(adaptadorBolsa.actor, contextoBolsa.actor.actor_ref);
  assert.equal(coordinador.obtenerContextoBolsa(), contextoBolsa);
  assert.equal(coordinador.resolverAcceso("bolsa", true).disponible, true);
  assert.equal(coordinador.resolverAcceso("cronos", true).disponible, true);
  assert.equal(coordinador.resolverAcceso("dietas", true).disponible, true);
});

test("el portal muestra el catálogo completo y dentro de cada módulo solo el acceso activo", async () => {
  const coordinador = crearCoordinador();
  await coordinador.cargarPresentacion(obtenerDatosPresentacion("tecnico").sesion);
  const portal = coordinador.renderizarNavegacion(true, "portal");
  assert.equal((portal.match(/data-modulo-portal=/g) || []).length, 12);
  assert.equal((portal.match(/modulo-habilitado/g) || []).length, 3);

  const bolsa = coordinador.renderizarNavegacion(true, "bolsa");
  assert.equal((bolsa.match(/data-modulo-portal=/g) || []).length, 1);
  assert.match(bolsa, /data-modulo-portal="bolsa"/);
  assert.doesNotMatch(bolsa, /data-modulo-portal="cronos"/);

  const cronos = coordinador.renderizarNavegacion(true, "cronos");
  assert.equal((cronos.match(/data-modulo-portal=/g) || []).length, 1);
  assert.match(cronos, /data-modulo-portal="cronos"/);
  assert.doesNotMatch(cronos, /data-modulo-portal="bolsa"/);
});

test("las rutas estables no mezclan el submenú de Bolsa con los módulos personales", () => {
  assert.equal(rutaDeVistaPortal("portal"), "#portal");
  assert.equal(rutaDeVistaPortal("resumen"), "#bolsa/resumen");
  assert.equal(rutaDeVistaPortal("cronos"), "#cronos");
  assert.equal(rutaDeVistaPortal("dietas"), "#dietas");
  assert.equal(moduloDeVistaPortal("convocatorias"), "bolsa");
  assert.equal(moduloDeVistaPortal("cronos"), "cronos");
});

test("Cronos y Dietas montan contenido administrativo y nunca dejan el área en blanco", async () => {
  const coordinador = crearCoordinador();
  await coordinador.cargarPresentacion(obtenerDatosPresentacion("administrador").sesion);
  const raiz = raizFalsa();
  assert.equal(await coordinador.montarVista("cronos", raiz), true);
  assert.match(raiz.innerHTML, /class="cronos-area"/);
  assert.match(raiz.innerHTML, /Descargar recibo/);
  assert.equal(await coordinador.montarVista("dietas", raiz), true);
  assert.match(raiz.innerHTML, /data-modulo="dietas"/);
  assert.match(raiz.innerHTML, /comisiones de servicio/i);
  coordinador.desmontarVistaActual();
});

test("el coordinador no autentica ni conserva estado en el navegador", async () => {
  const [fuente, estilos] = await Promise.all([
    readFile(new URL("portal-modulos-coordinador.js", import.meta.url), "utf8"),
    readFile(new URL("portal-modulos.css", import.meta.url), "utf8"),
  ]);
  assert.doesNotMatch(fuente, /document\.cookie|localStorage|sessionStorage/);
  assert.match(fuente, /CAPACIDADES_AUTOSERVICIO_CRONOS/);
  assert.match(fuente, /CAPACIDADES_AUTOSERVICIO_DIETAS/);
  assert.match(fuente, /import\("\.\/modulos\/cronos\/datos-presentacion\.js/);
  assert.match(fuente, /import\("\.\/modulos\/dietas\/adaptador-presentacion\.js/);
  assert.match(estilos, /data-modulo-catalogo="bolsa"/);
  assert.match(estilos, /data-modulo-catalogo="cronos"/);
  assert.match(estilos, /data-modulo-catalogo="dietas"/);
  assert.match(estilos, /data-modulo-portal="cronos"/);
  assert.match(estilos, /forced-colors: active/);
  assert.doesNotMatch(estilos, /\.tarjeta-modulo-bloqueada/);
});
