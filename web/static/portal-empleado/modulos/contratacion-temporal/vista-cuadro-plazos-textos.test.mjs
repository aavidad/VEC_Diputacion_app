import assert from "node:assert/strict";
import test from "node:test";
import { prepararTextosContratacionVista, crearTraductorCuadroCT } from "./i18n-vistas.js";
import { renderizarListaPeticiones } from "./vista-expedientes-lista.js";
import { filtroListaValido } from "./recuentos-peticiones.js";

const estados = ["pendiente", "en_curso", "espera", "incidencia", "completado", "cancelado"];
const plazos = [
  { estado: "vencido", dia: "2026-10-06", visible: "06/10/2026" },
  { estado: "vence_hoy", dia: "2026-10-07", visible: "07/10/2026" },
  { estado: "en_plazo", dia: "2026-10-08", visible: "08/10/2026" },
  { estado: "no_calculado", dia: "", visible: "" },
  { estado: "", dia: "", visible: "" },
];

function cuadroConPlazos(traducir) {
  return { cuadro: { generado_en: "2026-10-07T09:00:00Z", paginacion: { cursor_siguiente: "" },
    expedientes: estados.map((estado, indice) => ({
      expediente_ref: `expediente:ct:${indice + 1}`, numero_visible: `2026/CT-${indice + 1}`,
      centro: "centro:rrhh", categoria: "Auxiliar", fase_clave: "analisis",
      fase_actual: traducir("etiqueta_fase_analisis_rrhh"), estado_clave: estado,
      estado: traducir(`fase_${estado}`), plazo_estado: plazos[indice % plazos.length].estado,
      plazo_ultimo_dia: plazos[indice % plazos.length].dia,
      plazo: plazos[indice % plazos.length].visible, urgente: true,
    })) }, filtros: {} };
}

const ayudas = Object.freeze({ numeroVisible: (numero) => numero,
  centroVisible: (referencia) => ({ referencia, etiqueta: referencia }) });

for (const idioma of ["es", "en"]) {
  test(`lista ligera ${idioma}: plazos y seis estados con el catálogo que monta la pantalla`, async () => {
    const preparado = await prepararTextosContratacionVista("cuadro", { idioma });
    assert.equal(preparado.idioma, idioma);
    assert.deepEqual(Object.keys(preparado.secciones).sort(), [
      "contratacion-temporal-ficha-lista.general", "contratacion-temporal-lista-plazos.lista",
      "portal.fases_rrhh", "portal.general", "portal.textos",
    ].sort(), "la vista usa solo sus catálogos declarados");
    const traducir = crearTraductorCuadroCT(preparado);
    for (const estado of estados) assert.ok(traducir(`fase_${estado}`).trim(), estado);
    for (const estado of ["en_plazo", "vence_hoy", "vencido", "sin_calcular"]) {
      assert.ok(traducir(`plazo_fase_${estado}`).trim(), estado);
    }
    assert.ok(traducir("lista_sin_plazo").trim());
    assert.ok(traducir("marca_urgente").trim());
    const html = renderizarListaPeticiones(cuadroConPlazos(traducir), traducir,
      filtroListaValido({ mostrar: "todas" }), ayudas);
    assert.ok(html.includes(traducir("plazo_fase_vencido")));
    assert.ok(html.includes(traducir("plazo_fase_vence_hoy")));
    assert.ok(html.includes(traducir("marca_urgente")));
    assert.equal((html.match(/data-ct-exp-abrir=/gu) ?? []).length, estados.length);
    assert.equal((html.match(/class="ct-marca-urgente"/gu) ?? []).length, estados.length);

    const sinVencido = { ...preparado,
      secciones: { ...preparado.secciones,
        "contratacion-temporal-lista-plazos.lista": { ...preparado.secciones["contratacion-temporal-lista-plazos.lista"] } } };
    delete sinVencido.secciones["contratacion-temporal-lista-plazos.lista"].plazo_fase_vencido;
    const traductorRoto = crearTraductorCuadroCT(sinVencido);
    assert.throws(() => renderizarListaPeticiones(cuadroConPlazos(traductorRoto), traductorRoto,
      filtroListaValido({ mostrar: "todas" }), ayudas), /plazo_fase_vencido/u,
    "retirar la clave del catálogo activo rompe una fila vencida en el test");
  });
}
