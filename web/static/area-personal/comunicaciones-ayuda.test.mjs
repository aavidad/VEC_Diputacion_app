import assert from "node:assert/strict";
import test from "node:test";

import {
  renderizarAyuda, renderizarCertificados, renderizarMensajes,
} from "./vistas/comunicaciones-ayuda.js";

function datosBase() {
  return {
    meta: { presentacion: true },
    resumen: { mensajes_no_leidos: 0 },
    preferencias_notificacion: {},
    mensajes: [],
    certificados: [],
    documentos: [],
    ayuda: [],
  };
}

test("mensajes distingue bandeja vacía, error de datos y canales no conectados", () => {
  const vacia = renderizarMensajes(datosBase());
  assert.match(vacia, /No hay mensajes/u);
  assert.match(vacia, /canales de aviso no están conectados/u);
  assert.match(vacia, /no configura un envío, una entrega ni una notificación administrativa/u);
  assert.doesNotMatch(vacia, /Telegram|SMTP/iu);

  const error = renderizarMensajes({ ...datosBase(), mensajes: null, resumen: null });
  assert.match(error, /role="alert"/u);
  assert.match(error, /No se puede mostrar la bandeja/u);
});

test("certificados vacíos o no disponibles no se presentan como oficiales", () => {
  const vacia = renderizarCertificados(datosBase());
  assert.match(vacia, /No hay certificados disponibles/u);
  assert.match(vacia, /No contiene firma o sello oficial/u);
  assert.match(vacia, /ni acredita entrega/u);

  const error = renderizarCertificados({ ...datosBase(), certificados: null });
  assert.match(error, /No se pueden mostrar los certificados/u);

  const disponible = renderizarCertificados({
    ...datosBase(),
    certificados: [{ id: "DEMO-CER-001", tipo: "Certificado", descripcion: "Documento sintético", estado: "Disponible", formatos: "PDF, ODT" }],
  });
  assert.match(disponible, /Preparar certificado DEMO/u);
  assert.match(disponible, /Referencia técnica: DEMO-CER-001/u);
  assert.doesNotMatch(disponible, /Generar certificado/u);
});

test("ayuda conserva la guía textual y jerarquía accesible sin audio ajeno", () => {
  const ayuda = renderizarAyuda(datosBase(), { consultaAyuda: "sin resultado" });
  assert.match(ayuda, /Sin coincidencias/u);
  assert.match(ayuda, /<section aria-labelledby="titulo-guia-ayuda"><h4 id="titulo-guia-ayuda">Leer esta ayuda<\/h4>/u);
  assert.match(ayuda, /Esta guía explica el área personal de Bolsa/u);
  assert.match(ayuda, /<aside aria-label="Opciones y canales de ayuda">/u);
  assert.doesNotMatch(ayuda, /<audio\b|ayuda-llamamiento-bolsa\.mp3/u);
  assert.doesNotMatch(ayuda, /https?:\/\//u);

  const error = renderizarAyuda({ ...datosBase(), ayuda: null });
  assert.match(error, /La ayuda no está disponible/u);
  assert.match(error, /No se ha enviado información a ningún servicio externo/u);
});
