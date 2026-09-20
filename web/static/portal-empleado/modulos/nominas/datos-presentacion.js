import { obtenerAtlasSinteticoRRHH, TEXTO_DATOS_FICTICIOS_RRHH } from "../../datos-sinteticos-rrhh.js";

// Información exclusivamente visual. No representa importes, documentos ni derechos reales.
const EJEMPLO = Object.freeze({
  persona: "Antonio López Fernández",
  aviso: TEXTO_DATOS_FICTICIOS_RRHH,
  estado: "visual_pendiente_backend",
  nominas: [
    { id: "ejemplo-2026-08", periodo: "2026-08", referencia: "EJEMPLO-AGOSTO-2026", estado: "Documento de ejemplo", bruto: "2.184,30 €", liquido: "1.734,90 €", conceptos: [["Retribución fija", "Devengo", "1.860,00 €"], ["Complemento de puesto", "Devengo", "324,30 €"], ["Deducciones mostradas", "Deducción", "449,40 €"]] },
    { id: "ejemplo-2026-07", periodo: "2026-07", referencia: "EJEMPLO-JULIO-2026", estado: "Documento de ejemplo", bruto: "2.164,30 €", liquido: "1.721,10 €", conceptos: [["Retribución fija", "Devengo", "1.860,00 €"], ["Complemento de puesto", "Devengo", "304,30 €"], ["Deducciones mostradas", "Deducción", "443,20 €"]] },
    { id: "ejemplo-2026-06", periodo: "2026-06", referencia: "EJEMPLO-JUNIO-2026", estado: "Documento de ejemplo", bruto: "2.164,30 €", liquido: "1.721,10 €", conceptos: [["Retribución fija", "Devengo", "1.860,00 €"], ["Complemento de puesto", "Devengo", "304,30 €"], ["Deducciones mostradas", "Deducción", "443,20 €"]] },
  ],
  certificados: [
    ["Certificado fiscal anual 2025", "Ejemplo no disponible para descarga"],
    ["Certificado de retribuciones 2026", "Ejemplo no disponible para descarga"],
  ],
  incidencias: [["INC-EJEMPLO-01", "Diferencia en concepto mostrado", "Pendiente de conexión"]],
});

export function obtenerDatosNominasPresentacion() {
  // El atlas común aporta sólo identidad visual sintética; no es una fuente de nómina.
  const atlas = obtenerAtlasSinteticoRRHH();
  return Object.freeze({ ...EJEMPLO, persona: atlas.persona_principal.nombre_visible, aviso: atlas.aviso_visible });
}
