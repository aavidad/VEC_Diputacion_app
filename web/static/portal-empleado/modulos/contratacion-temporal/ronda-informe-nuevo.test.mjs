import assert from "node:assert/strict";
import test from "node:test";

import { informeNuevoVigenteEnSubsanacion, inicioRondaInformeNuevo } from "./ronda-informe-nuevo.js?v=VEC_WEB_VERSION";
import { solicitudInformeDefinitivoDesdeEstado } from "./componentes-expedientes.js?v=VEC_WEB_VERSION";
import { crearGestorCircuitoFirma, validarCircuitoFirma } from "./circuito-firma.js?v=VEC_WEB_VERSION";
import { cargarTextos } from "../../../comun/textos.js";
import { IDIOMA_POR_DEFECTO } from "../../../comun/idioma.js";

const REF = "expediente:ct:ronda";
const INFORME = "contratacion_temporal.informe_juridico.generar";
const SUBSANACION = "contratacion_temporal.subsanacion_reparos.registrar";
const FISCALIZACION = "contratacion_temporal.fiscalizacion.registrar";

// El historial real de la ficha no trae la fase de destino de cada hito.
function hito(secuencia, accion_clave) {
  return { secuencia, version_expediente: secuencia, fecha: "20 oct 2026", fase: "Fase", accion: "Actuación",
    estado_clave: "incidencia", estado: "Incidencia", accion_clave };
}

function estadoTrasInformeNuevo({ ultimo = INFORME, fase = "subsanacion_unidad", estado = "incidencia" } = {}) {
  const historial = [hito(5, INFORME), hito(6, FISCALIZACION), hito(7, SUBSANACION)];
  if (ultimo) historial.push(hito(8, ultimo));
  const version = historial.at(-1).version_expediente;
  return { vista: "expediente", carga: "listo", expediente_ref: REF,
    expediente: { expediente_ref: REF, version, demostracion: false, historial },
    cuadro: { demostracion: false, expedientes: [{ expediente_ref: REF, version, fase_clave: fase, estado_clave: estado }] } };
}

test("el informe nuevo vigente se reconoce con el historial real, sin fase de destino en el hito", () => {
  assert.equal(informeNuevoVigenteEnSubsanacion(estadoTrasInformeNuevo()), true);
  assert.equal(informeNuevoVigenteEnSubsanacion(estadoTrasInformeNuevo({ ultimo: null })), false);
  assert.equal(informeNuevoVigenteEnSubsanacion(estadoTrasInformeNuevo({ fase: "fiscalizacion", estado: "en_curso" })), false);
  const desfasado = estadoTrasInformeNuevo();
  desfasado.cuadro.expedientes[0].version = 7;
  assert.equal(informeNuevoVigenteEnSubsanacion(desfasado), false);
});

test("la ronda empieza en la última emisión solo cuando sustituye a otra", () => {
  assert.equal(inicioRondaInformeNuevo(estadoTrasInformeNuevo()), 8);
  assert.equal(inicioRondaInformeNuevo(estadoTrasInformeNuevo({ ultimo: null })), 0);
  assert.equal(inicioRondaInformeNuevo({}), 0);
});

test("tras el informe nuevo se puede volver a firmar el informe definitivo y nada más", () => {
  const estado = estadoTrasInformeNuevo();
  assert.deepEqual(solicitudInformeDefinitivoDesdeEstado(estado, "informe_definitivo"),
    { expediente_ref: REF, version_observada: 8 });
  assert.equal(solicitudInformeDefinitivoDesdeEstado(estado), null);
  assert.equal(solicitudInformeDefinitivoDesdeEstado(estado, "resolucion"), null);
  assert.equal(solicitudInformeDefinitivoDesdeEstado(estadoTrasInformeNuevo({ ultimo: null }), "informe_definitivo"), null);
});

const textos = await cargarTextos("contratacion-temporal-firma", { idioma: IDIOMA_POR_DEFECTO });

function catalogo() {
  const paso = (orden) => ({ orden, cargo: `Cargo ${orden}`, perfil_ref: "perfil:ct:jefatura_servicio_rrhh",
    accion: "firma", condicion: orden === 1 ? "borrador_generado" : "firma_paso_anterior",
    habilita: orden === 2 ? "remision_intervencion" : "siguiente_paso", devolucion: "vuelve_a_redaccion",
    sustitucion: "suplente_designado", estado: orden === 1 ? "pendiente_firma" : "en_espera",
    referencia: `vec.contratacion_temporal.circuito_firma:1:informe_definitivo.p${orden}` });
  return validarCircuitoFirma({ esquema: "vec.contratacion_temporal.circuito_firma.v1",
    catalogo_ref: "vec.contratacion_temporal.circuito_firma:1", huella_sha256: "a".repeat(64), ejemplo: true,
    firma_eficaz: false, portafirmas: { conectado: false, motivo: "conexion_pendiente" },
    documentos: [{ documento: "informe_definitivo", etiqueta: "Informe definitivo", pasos: [paso(1), paso(2)] }] });
}

async function pintarConFirmas(versiones) {
  const estado = estadoTrasInformeNuevo();
  const circuito = catalogo();
  const insertados = [];
  const raiz = { querySelector: (s) => (s === ".ct-exp-progreso" ? { insertAdjacentHTML: (_, html) => insertados.push(html) } : null) };
  const gestor = crearGestorCircuitoFirma({ raiz, cargarTextos: async () => textos, obtenerEstado: () => estado,
    cliente: { obtenerCircuitoConEstado: async () => ({ estado: "disponible", circuito }) },
    resolverDisponibilidadFirma: (c) => ({ estado: "disponible", expediente_ref: c.ref, version_observada: c.version,
      documentos: ["informe_definitivo"] }),
    clienteFirmasV2: { consultar: async () => ({ expediente_ref: REF, version_expediente: 8, documento: "informe_definitivo",
      firmas: versiones.map((v, i) => ({ paso_orden: i + 1, secuencia: i + 1, version_expediente: v,
        catalogo_ref: circuito.catalogo_ref, catalogo_huella: circuito.huella_sha256, resultado: "firmado",
        via: "certificado_vec", revision_pdf: {}, documento_custodiado: {}, registrada_en: "2026-10-20T08:00:00Z" })) }) },
  });
  gestor.montarSiProcede(estado);
  await new Promise((resolver) => setTimeout(resolver, 5));
  gestor.retirar();
  return insertados.join("");
}

test("las firmas del informe sustituido no se presentan como firmas del informe nuevo", async () => {
  const anterior = await pintarConFirmas([5, 5]);
  assert.match(anterior, /data-ct-fase-firma-estado="borrador"/u);
  assert.match(anterior, /ct-circuito-paso--aviso" aria-current="step"/u);
  assert.doesNotMatch(anterior, /data-ct-fase-firma-estado="firmado"/u);
  const nueva = await pintarConFirmas([8, 8]);
  assert.match(nueva, /data-ct-fase-firma-estado="firmado"/u);
  assert.doesNotMatch(nueva, /aria-current="step"/u);
});
