import test from "node:test";
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { renderizarMeritosPropios } from "./meritos-reales-vista.js";

const catalogo = JSON.parse(await readFile(new URL("../textos/es/meritos-reales.json", import.meta.url), "utf8"));
const textos = { traducir(ruta, variables = {}) {
  const valor = ruta.split(".").reduce((parte, clave) => parte[clave], catalogo);
  const plantilla = typeof valor === "string" ? valor : valor[variables.cuenta === 1 ? "one" : "other"];
  return plantilla.replace(/\{([a-z]+)\}/gu, (_todo, clave) => String(variables[clave] ?? ""));
} };

const hecho = (ref, estado, denominacion) => ({ hecho_ref: ref, version_hecho: 2,
  tipo: "curso_superacion", denominacion, disponibilidad: { estado, version: estado === "retirado" ? 1 : 0 },
  retirada: estado === "retirado" ? { recibo_ref: "recibo:anterior", fecha: "2026-10-11T15:00:00Z" } : null });

test("cartera vacía no fabrica puntos, alta ni acción sin motivos publicados", () => {
  const html = renderizarMeritosPropios({ estado: "vacio", filtro: "todos", hechos: [] }, textos);
  assert.match(html, /No hay méritos en este filtro/u);
  assert.doesNotMatch(html, /puntos|Puntuación|<input type="file"|data-meritos-retirar/u);
  assert.match(html, /data-meritos-filtro="todos"/u);
});

test("filtros y total abren la misma lista; solo disponible ofrece retirada y escapa nombres", () => {
  const html = renderizarMeritosPropios({ estado: "listo", filtro: "todos", total: 2,
    hechos: [hecho("hecho:1", "disponible", "Curso <script>alert(1)</script>"),
      hecho("hecho:2", "retirado", "Curso anterior")], motivos: [{ etiqueta: "A petición propia" }],
    cursor_siguiente: "cursor-opaco" }, textos);
  assert.match(html, /2 méritos en este filtro/u);
  assert.match(html, /data-meritos-filtro="todos" class="meritos-reales-total"/u);
  assert.match(html, /data-meritos-retirar="hecho:1"/u);
  assert.doesNotMatch(html, /data-meritos-retirar="hecho:2"|<script>/u);
  assert.match(html, /&lt;script&gt;/u);
  assert.match(html, /data-meritos-mas/u);
});

test("revisión muestra efecto y motivo recibido; recibo confirmado conserva replay", () => {
  const html = renderizarMeritosPropios({ estado: "listo", filtro: "todos", total: 1,
    hechos: [hecho("hecho:1", "disponible", "Curso de archivos")],
    motivos: [{ etiqueta: "Motivo <privado>" }], seleccion: { hecho_ref: "hecho:1" },
    confirmacion: { repetida: true, recibo: { referencia: "recibo:original" } } }, textos);
  assert.match(html, /Revise la retirada/u);
  assert.match(html, /Las anteriores conservarán lo presentado/u);
  assert.match(html, /value="0">Motivo &lt;privado&gt;/u);
  assert.match(html, /La retirada ya estaba registrada/u);
  assert.match(html, /Recibo recibo:original/u);
  assert.doesNotMatch(html, /<privado>|puntos/u);
});
