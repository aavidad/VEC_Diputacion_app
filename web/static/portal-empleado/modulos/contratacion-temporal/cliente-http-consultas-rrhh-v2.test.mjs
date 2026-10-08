import assert from "node:assert/strict";
import test from "node:test";
import { crearConsultasRRHHClienteHTTP } from "./cliente-http-consultas-rrhh.js";

const ESQUEMA_V2 = "vec.contratacion-temporal.cuadro-rrhh.v2";
const filtrosPlazo = Object.freeze({ texto: "", estado_clave: "", fase_clave: "", plazo_estado: "vencido" });
const solicitud = Object.freeze({ esquema: ESQUEMA_V2, filtros: filtrosPlazo,
  paginacion: { limite: 100, cursor: "" }, resumen: true });
const paginaV2 = Object.freeze({ esquema: "vec.contratacion-temporal.cuadro-rrhh.v2",
  generada_en: "2026-10-08T10:00:00Z", expedientes: [], hay_mas: false,
  totales: { total: 0, en_tramitacion: 0, con_incidencia: 0, en_llamamiento: 0 },
  resumen: { en_tramite: 0, con_incidencia: 0, vencidos: 0, vencen_hoy: 0,
    vencen_semana: 0, sin_calcular: 0, por_fase: {} } });

test("V2 envía el filtro de plazo por la ruta existente y exige una respuesta V2 íntegra", async () => {
  const llamadas = [];
  const cliente = crearConsultasRRHHClienteHTTP({ validarOpciones: () => ({ signal: undefined }),
    ejecutar: async (opciones) => {
      llamadas.push(opciones);
      return opciones.validarRespuesta(paginaV2);
    } });
  const respuesta = await cliente.consultarCuadroRRHHV2(solicitud);
  assert.equal(respuesta.totales.total, 0);
  assert.equal(llamadas.length, 1);
  assert.equal(llamadas[0].ruta, "/api/vec/contratacion-temporal/cuadro/consultas");
  assert.equal(llamadas[0].entrada.esquema, ESQUEMA_V2);
  assert.deepEqual(llamadas[0].entrada.filtros, filtrosPlazo);
  assert.equal(llamadas[0].efecto, false);
  assert.throws(() => llamadas[0].validarRespuesta({ ...paginaV2, esquema: "vec.contratacion-temporal.cuadro-rrhh.v1" }),
    /página de cuadro RRHH no válida/u);
  assert.throws(() => llamadas[0].validarRespuesta({ ...paginaV2, totales: { ...paginaV2.totales, total: -1 } }),
    /página de cuadro RRHH no válida/u);
});

test("el cliente V1 no admite plazo y el V2 rechaza vocabulario ajeno antes de consultar", async () => {
  let llamadas = 0;
  const cliente = crearConsultasRRHHClienteHTTP({ validarOpciones: () => ({ signal: undefined }),
    ejecutar: async () => { llamadas++; return paginaV2; } });
  assert.throws(() => cliente.consultarCuadroRRHH(solicitud), /solicitud de cuadro RRHH no válida/u);
  for (const plazo_estado of ["vencidos", "hoy", "", null]) {
    if (plazo_estado === "") continue;
    assert.throws(() => cliente.consultarCuadroRRHHV2({ ...solicitud,
      filtros: { ...filtrosPlazo, plazo_estado } }), /solicitud de cuadro RRHH V2 no válida/u);
  }
  assert.throws(() => cliente.consultarCuadroRRHHV2({ ...solicitud,
    filtros: { ...filtrosPlazo, estado_clave: "estado_ajeno" } }), /solicitud de cuadro RRHH V2 no válida/u);
  assert.throws(() => cliente.consultarCuadroRRHHV2({ ...solicitud,
    filtros: { ...filtrosPlazo, centro_ref: "centro:otro" } }), /solicitud de cuadro RRHH V2 no válida/u);
  assert.throws(() => cliente.consultarCuadroRRHHV2({ ...solicitud, esquema: "vec.contratacion-temporal.cuadro-rrhh.v1" }),
    /solicitud de cuadro RRHH V2 no válida/u);
  assert.equal(llamadas, 0);
});
