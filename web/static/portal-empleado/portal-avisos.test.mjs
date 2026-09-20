import assert from "node:assert/strict";
import test from "node:test";
import { obtenerDatosPresentacion } from "./datos-presentacion.js";
import { validarAvisosPortal } from "./portal-contrato.js";
import {
  instalarDestinosAvisos,
  renderizarAvisosNavegables,
} from "./portal-eventos.js";

const avisosDisponibles = obtenerDatosPresentacion().avisos;

function contenedorFalso() {
  let escuchar = null;
  return {
    addEventListener: (tipo, funcion) => { if (tipo === "click") escuchar = funcion; },
    removeEventListener: (tipo, funcion) => { if (tipo === "click" && escuchar === funcion) escuchar = null; },
    contains: () => true,
    click: (indice) => escuchar?.({
      target: { closest: () => ({ dataset: { avisoDestino: String(indice) } }) },
      preventDefault: () => {},
    }),
  };
}

test("los avisos solo admiten destinos internos registrados y referencias opacas", () => {
  const [primero] = validarAvisosPortal(avisosDisponibles);
  assert.equal(primero.destino.vista, "elaboracion");
  assert.equal(primero.destino.referencia, "DEMO-AVISO-BOL-014");
  assert.throws(() => validarAvisosPortal([{
    texto: "Destino malicioso",
    destino: { vista: "javascript:alert(1)", etiqueta: "No", estado: "disponible" },
  }]), /no registrada/);
  assert.throws(() => validarAvisosPortal([{
    texto: "URL libre",
    destino: { vista: "https://externo.example", etiqueta: "No", estado: "disponible" },
  }]), /no registrada/);
  assert.throws(() => validarAvisosPortal([{
    texto: "Referencia libre",
    destino: { vista: "dietas", etiqueta: "Dietas", estado: "disponible", referencia: "<img>" },
  }]), /referencia.*no válida/);
});

test("cada aviso disponible se presenta como botón accesible con contexto de destino", () => {
  const html = renderizarAvisosNavegables(avisosDisponibles, (valor) => String(valor).replaceAll("<", "&lt;"));
  assert.match(html, /<button type="button"[^>]*data-aviso-destino="0"[^>]*aria-label="Ir a Borradores de convocatorias">/);
  assert.match(html, />Ir a Borradores de convocatorias<\/button>/);
  // Un botón nativo recibe Enter y Espacio y emite click; no se añade un
  // atajo propio que pudiera ejecutarlo dos veces.
  assert.doesNotMatch(html, /role="link"|tabindex=/);
});

test("click, incluido el generado por teclado en el botón, cierra el diálogo y navega internamente", () => {
  const contenedor = contenedorFalso();
  const orden = [];
  instalarDestinosAvisos(contenedor, avisosDisponibles, {
    cerrar: () => orden.push("cerrar"),
    navegar: (vista, opciones) => orden.push(["navegar", vista, opciones]),
    anunciar: (texto) => orden.push(["anunciar", texto]),
  });
  contenedor.click(0);
  assert.deepEqual(orden, [
    "cerrar",
    ["navegar", "elaboracion", { referencia: "DEMO-AVISO-BOL-014" }],
    ["anunciar", "Aviso: Borradores de convocatorias"],
  ]);
});

test("un destino pendiente queda deshabilitado y no simula navegación", () => {
  const avisos = [{
    texto: "La consulta todavía no está conectada.",
    destino: { vista: "cronos", etiqueta: "Cronos", estado: "pendiente" },
  }];
  const html = renderizarAvisosNavegables(avisos, String);
  assert.match(html, /disabled aria-disabled="true"/);
  assert.match(html, /Pendiente de conexión/);
  const contenedor = contenedorFalso();
  let navegaciones = 0;
  instalarDestinosAvisos(contenedor, avisos, {
    cerrar: () => assert.fail("no debe cerrar"),
    navegar: () => { navegaciones += 1; },
    anunciar: () => assert.fail("no debe anunciar"),
  });
  contenedor.click(0);
  assert.equal(navegaciones, 0);
});
