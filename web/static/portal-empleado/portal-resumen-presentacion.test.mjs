import assert from "node:assert/strict";
import test from "node:test";
import { crearVistaResumenPresentacion } from "./portal-resumen-presentacion.js";

const escaparHTML = (valor) => String(valor)
  .replaceAll("&", "&amp;")
  .replaceAll("<", "&lt;")
  .replaceAll(">", "&gt;")
  .replaceAll('"', "&quot;");
const numero = (valor, decimales = 0) => Number(valor).toFixed(decimales);
const porcentajeSeguro = (valor) => Math.max(0, Math.min(100, Number(valor) || 0));

function datosPresentacion() {
  return {
    indicadores: {
      bolsas_activas: 3,
      candidatos_disponibles: 20,
      llamamientos_pendientes: 2,
      contratos_activos: 8,
      cobertura_media: 75.5,
    },
    distribucion_global: {
      disponibles: 20,
      en_llamamiento: 5,
      contratados: 8,
      no_disponibles: 2,
    },
    series: { contratos_mes: [1, 4, 2], periodo_contratos: "Enero–marzo" },
    bolsas: [{
      id: 'DEMO-1"><script>',
      nombre: "Auxiliares <prueba>",
      categoria: "Administración",
      integrantes: 10,
      llamamiento: 2,
      cobertura: 75,
    }],
    proximos: [{
      dia: "22", mes: "JUL", bolsa: "Auxiliares", numero: "3", fecha: "22/07/2026", estado: "Previsto",
    }],
    actividad: [{
      accion: "Revisión", objeto: "DEMO-1", actor: "Técnico RRHH", fecha: "21/07/2026",
    }],
  };
}

test("el resumen de presentación exige todas sus dependencias", () => {
  assert.throws(() => crearVistaResumenPresentacion(), /dependencias/);
});

test("el resumen conserva semántica accesible y escapa los datos sintéticos", () => {
  const renderizar = crearVistaResumenPresentacion({
    escaparHTML,
    encabezadoVista: (_sobrelinea, titulo) => `<header><h2>${escaparHTML(titulo)}</h2></header>`,
    etiquetaFuentePanel: () => "Presentación RRHH",
    numero,
    obtenerDatosPanel: datosPresentacion,
    porcentajeSeguro,
  });
  const html = renderizar();
  assert.match(html, /<h2>Cuadro de mando<\/h2>/);
  assert.match(html, /role="meter"[^>]+aria-valuenow="75"/);
  assert.match(html, /role="img" aria-label="20 disponibles/);
  assert.match(html, /Auxiliares &lt;prueba&gt;/);
  assert.doesNotMatch(html, /<script>/);
  assert.match(html, /data-id="DEMO-1&quot;&gt;&lt;script&gt;"/);
  assert.match(html, /Presentación RRHH/);
});
