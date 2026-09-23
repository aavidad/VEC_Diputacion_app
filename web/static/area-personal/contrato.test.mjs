import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { crearAdaptadorPresentacion } from "./adaptador-presentacion.js";
import {
  CONTRATO_AREA_PERSONAL,
  MOTIVOS_PAUSA_DISPONIBILIDAD,
  RESULTADOS_LLAMAMIENTO,
  SITUACIONES_PARTICIPACION_BOLSA,
  esModoPresentacion,
  validarDatosAreaPersonal,
  validarRespuestaMiBolsa,
  validarPayloadCambiarDisponibilidad,
  validarRecibo,
} from "./contrato.js";

test("mi bolsa valida la situación actual sin exigirla a la respuesta antigua", () => {
  const base = { data: { esquema: "vec.bolsa.mi-bolsa.v1", consultada_en: "2026-09-23T10:00:00.000Z", participaciones: [
    { bolsa: "bolsa:prueba:01", categoria: "Auxiliar", version: 3, orden_inicial: 12, total_instantanea: 87, estado_bolsa: "vigente", vigente_desde: "2026-09-01T00:00:00.000Z", vigente_hasta: null },
  ] } };
  assert.equal(validarRespuestaMiBolsa(base).participaciones[0].situacion_actual, undefined);
  base.data.participaciones[0].situacion_actual = { estado: "disponible_desde", desde: "2026-09-20T10:00:00.000Z", hasta: null, fecha_disponible: "2026-10-01T10:00:00.000Z" };
  assert.equal(validarRespuestaMiBolsa(base).participaciones[0].situacion_actual.estado, "disponible_desde");
  base.data.participaciones[0].situacion_actual.estado = "inventado";
  assert.throws(() => validarRespuestaMiBolsa(base), /situación actual/u);
  base.data.participaciones[0].situacion_actual.estado = "disponible";
  assert.throws(() => validarRespuestaMiBolsa(base), /fecha de disponibilidad/u);
});

test("el selector de presentación es único y explícito", () => {
  assert.equal(esModoPresentacion(new URLSearchParams("presentacion=rrhh")), true);
  assert.equal(esModoPresentacion(new URLSearchParams("presentacion=aspirante")), false);
  assert.equal(esModoPresentacion(new URLSearchParams()), false);
  assert.equal(esModoPresentacion(new URLSearchParams("presentacion=RRHH")), false);
  assert.throws(
    () => esModoPresentacion(new URLSearchParams("presentacion=rrhh&presentacion=otro")),
    /selector de presentación es ambiguo/iu,
  );
});

test("el contrato acepta el juego mixto de presentación y lo congela", async () => {
  const datos = await crearAdaptadorPresentacion().cargar();
  assert.equal(datos.meta.presentacion, true);
  assert.equal(datos.meta.esquema, "vec.bolsa.area-personal.v1");
  assert.equal(Object.isFrozen(datos), true);
  assert.equal(Object.isFrozen(datos.convocatorias), true);
  assert.ok(datos.sesion.persona_ref.startsWith("DEMO-"));
  assert.ok(datos.perfil.referencia.startsWith("DEMO-"));
});

test("la presentación rechaza DNI o NIE formalmente válidos", async () => {
  const datos = structuredClone(await crearAdaptadorPresentacion().cargar());
  datos.perfil.identificador_visible = "12345678Z";
  assert.throws(
    () => validarDatosAreaPersonal(datos, { presentacionEsperada: true }),
    /no admite DNI o NIE/,
  );
});

test("la presentación solo acepta correos reservados .test", async () => {
  const datos = structuredClone(await crearAdaptadorPresentacion().cargar());
  datos.perfil.correo = "persona@example.com";
  assert.throws(
    () => validarDatosAreaPersonal(datos, { presentacionEsperada: true }),
    /dominio \.test/,
  );
});

test("un origen sintético nunca pasa por el contrato productivo", async () => {
  const datos = await crearAdaptadorPresentacion().cargar();
  assert.throws(
    () => validarDatosAreaPersonal(datos, { presentacionEsperada: false }),
    /origen no coincide/,
  );
});

test("el recibo DEMO es inequívoco y no pasa como recibo real", () => {
  const entrada = {
    esquema: "vec.bolsa.area-personal.recibo-demo.v1",
    presentacion: true,
    referencia: "DEMO-REC-0001",
    accion: "guardar_borrador",
    objetivo: "DEMO-CONV-001",
    resultado: "Simulación completada sin efectos administrativos",
    actor: "Persona Aspirante de Demostración",
    fecha: "2026-07-18T09:00:00Z",
    advertencia: "RECIBO DEMO · Sin validez administrativa.",
  };
  assert.equal(validarRecibo(entrada, { presentacionEsperada: true }).presentacion, true);
  assert.throws(() => validarRecibo(entrada, { presentacionEsperada: false }), /esquema del recibo no es compatible/);
});

test("el contrato valida los campos ampliados de disponibilidad (B11)", async () => {
  const base = structuredClone(await crearAdaptadorPresentacion().cargar());
  base.disponibilidad.estado_clave = "ocupado";
  base.disponibilidad.estado_desde = "2026-07-01T08:00:00Z";
  base.disponibilidad.disponible_desde = "2026-10-01";
  base.disponibilidad.motivo_visible = "Incorporación temporal";
  const validado = validarDatosAreaPersonal(base, { presentacionEsperada: true });
  assert.equal(validado.disponibilidad.estado_clave, "ocupado");
  assert.equal(validado.disponibilidad.disponible_desde, "2026-10-01");
  assert.equal(validado.disponibilidad.motivo_visible, "Incorporación temporal");

  // Rechaza estado_clave fuera de catálogo
  base.disponibilidad.estado_clave = "invalido";
  assert.throws(
    () => validarDatosAreaPersonal(base, { presentacionEsperada: true }),
    /estado_clave no reconocido en el catálogo/u,
  );
});

test("el contrato valida canal, comunicado_en y resultado_clave en llamamientos", async () => {
  const base = structuredClone(await crearAdaptadorPresentacion().cargar());
  base.llamamientos[0].canal = "Sede electrónica";
  base.llamamientos[0].comunicado_en = "2026-07-17T10:30:00Z";
  base.llamamientos[0].resultado_clave = "aceptado";
  const validado = validarDatosAreaPersonal(base, { presentacionEsperada: true });
  assert.equal(validado.llamamientos[0].resultado_clave, "aceptado");

  // Rechaza resultado_clave desconocido
  base.llamamientos[0].resultado_clave = "rechazo_desconocido";
  assert.throws(
    () => validarDatosAreaPersonal(base, { presentacionEsperada: true }),
    /resultado_clave no reconocido en el catálogo/u,
  );
});

test("el contrato valida la sección Mi posición (B11)", async () => {
  const base = structuredClone(await crearAdaptadorPresentacion().cargar());
  base.posicion = {
    bolsa: "Bolsa de empleo de Operario",
    categoria: "Operario/a",
    orden: 3,
    total: 120,
    puntuacion: 22.5,
    vigente_desde: "2026-06-01",
  };
  const validado = validarDatosAreaPersonal(base, { presentacionEsperada: true });
  assert.equal(validado.posicion.orden, 3);
  assert.equal(validado.posicion.total, 120);

  // Rechaza campos faltantes o inválidos en posición
  const invalido = structuredClone(base);
  delete invalido.posicion.orden;
  assert.throws(
    () => validarDatosAreaPersonal(invalido, { presentacionEsperada: true }),
    /posicion\.orden debe ser un número/u,
  );
});

test("el recibo de cambiar_disponibilidad devuelve estado_clave y disponible_desde (B8)", () => {
  const reciboValido = {
    esquema: "vec.bolsa.area-personal.recibo-demo.v1",
    presentacion: true,
    referencia: "DEMO-REC-0010",
    accion: "cambiar_disponibilidad",
    objetivo: "DEMO-PER-0001",
    resultado: "Simulación completada sin efectos administrativos",
    actor: "Persona Aspirante de Demostración",
    fecha: "2026-07-18T09:00:00Z",
    advertencia: "RECIBO DEMO · Sin validez administrativa.",
    estado_clave: "no_disponible",
    disponible_desde: "2026-11-01",
  };
  const validado = validarRecibo(reciboValido, { presentacionEsperada: true });
  assert.equal(validado.estado_clave, "no_disponible");
  assert.equal(validado.disponible_desde, "2026-11-01");

  const reciboInvalido = structuredClone(reciboValido);
  reciboInvalido.estado_clave = "estado_inexistente";
  assert.throws(
    () => validarRecibo(reciboInvalido, { presentacionEsperada: true }),
    /estado_clave no reconocido en el catálogo/u,
  );
});

test("validarPayloadCambiarDisponibilidad valida esquemas de pausa y reactivación (B8)", () => {
  // Reactivación
  const reactivacion = validarPayloadCambiarDisponibilidad({ disponible: true });
  assert.equal(reactivacion.disponible, true);

  // Pausa completa
  const pausa = validarPayloadCambiarDisponibilidad({
    disponible: false,
    motivo_clave: "enfermedad",
    motivo_texto: "Incapacidad temporal",
    hasta: "2026-12-15",
  });
  assert.equal(pausa.disponible, false);
  assert.equal(pausa.motivo_clave, "enfermedad");
  assert.equal(pausa.hasta, "2026-12-15");

  // Rechaza motivo_clave no permitido
  assert.throws(
    () => validarPayloadCambiarDisponibilidad({ disponible: false, motivo_clave: "invalido" }),
    /motivo_clave no reconocido/u,
  );

  // Rechaza formato de fecha erróneo
  assert.throws(
    () => validarPayloadCambiarDisponibilidad({ disponible: false, motivo_clave: "pausa_voluntaria", hasta: "15/12/2026" }),
    /hasta debe tener formato AAAA-MM-DD/u,
  );
});

test("el contrato procesa fixtures derivadas del dataset sintético con mapeo de estados G12", async () => {
  const rutaDataset = new URL("../../../data/demo/bolsa/v1.bolsas-demo.json", import.meta.url);
  const raw = JSON.parse(await readFile(rutaDataset, "utf8"));

  function mapearSituacion(estadoClave) {
    switch (estadoClave) {
      case "trabajando":
      case "pendiente_incorporacion":
        return "ocupado";
      case "disponible_desde":
      case "no_disponible":
        return "no_disponible";
      case "renuncia":
        return "renuncia_pendiente";
      case "disponible":
        return "disponible";
      case "excluido":
        return "excluido";
      default:
        return "no_disponible";
    }
  }

  function mapearResultadoLlamamiento(resultado) {
    switch (resultado) {
      case "aceptado":
        return "aceptado";
      case "renuncia":
        return "renuncia";
      case "sin_respuesta":
        return "sin_respuesta";
      default:
        return "pendiente";
    }
  }

  assert.ok(raw.candidaturas?.length > 0);
  assert.ok(raw.llamamientos?.length > 0);

  const base = structuredClone(await crearAdaptadorPresentacion().cargar());

  // Tomamos una muestra de candidaturas del dataset
  for (const c of raw.candidaturas.slice(0, 10)) {
    const estadoMapeado = mapearSituacion(c.estado_clave);
    assert.ok(SITUACIONES_PARTICIPACION_BOLSA.includes(estadoMapeado));

    base.disponibilidad.estado_clave = estadoMapeado;
    base.disponibilidad.disponible = estadoMapeado === "disponible";
    base.disponibilidad.estado = c.estado || estadoMapeado;
    base.disponibilidad.estado_desde = c.estado_desde ? new Date(c.estado_desde).toISOString() : "2026-01-01T00:00:00Z";
    base.disponibilidad.disponible_desde = c.disponible_desde || null;
    base.disponibilidad.motivo_visible = c.estado || null;

    const bolsa = raw.bolsas.find((b) => b.bolsa_ref === c.bolsa_ref);
    base.posicion = {
      bolsa: bolsa?.categoria || c.bolsa_ref,
      categoria: bolsa?.categoria || "General",
      orden: c.orden,
      total: bolsa?.candidaturas || 50,
      puntuacion: c.puntuacion,
      vigente_desde: bolsa?.vigente_desde || "2025-01-01",
    };

    const llamamientosCandidatura = raw.llamamientos.filter((l) => l.candidatura_ref === c.candidatura_ref);
    if (llamamientosCandidatura.length > 0) {
      base.llamamientos = llamamientosCandidatura.map((l, i) => ({
        id: `DEMO-LLA-${String(i + 1).padStart(4, "0")}`,
        bolsa: base.posicion.bolsa,
        puesto: l.puesto || "Puesto de bolsa",
        plazo: l.plazo_respuesta_hasta || "48 horas",
        estado: l.resultado === "aceptado" ? "Aceptado" : "Pendiente de respuesta",
        canal: l.canal || "correo",
        comunicado_en: l.comunicado_en ? new Date(l.comunicado_en).toISOString() : "2026-01-01T00:00:00Z",
        resultado_clave: mapearResultadoLlamamiento(l.resultado),
      }));
    }

    const resultado = validarDatosAreaPersonal(base, { presentacionEsperada: true });
    assert.equal(resultado.disponibilidad.estado_clave, estadoMapeado);
    assert.equal(resultado.posicion.orden, c.orden);
  }
});
