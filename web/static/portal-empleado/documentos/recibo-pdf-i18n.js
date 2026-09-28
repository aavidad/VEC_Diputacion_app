/** Textos fijos del PDF local de demostración. Los datos del descriptor se conservan. */
export const MENSAJES_RECIBO_PDF_ES = Object.freeze({
  portal_demo: "Portal del Empleado - DEMO",
  dato: "DATO",
  valor: "VALOR",
  certifica: "CERTIFICA",
  firma_verificacion: "Firma y verificación",
  referencia: "Referencia: {referencia}",
  estado_demo: "Estado: documento de demostración sin validez administrativa",
  comprobar: "Comprobar documento",
  subtitulo_defecto: "Recibo emitido por el Portal del Empleado",
  nota_defecto: "El documento definitivo se generará y firmará en el servidor autorizado.",
  certificacion_defecto: "Se deja constancia de la actuación indicada y de su referencia de comprobación. El documento definitivo se emitirá desde el expediente administrativo autorizado.",
  rotulo_recibo: "RECIBO DE ACTUACIÓN",
  rotulo_certificado: "CERTIFICADO",
  titulo_descargador: "Recibo del Portal del Empleado",
  subtitulo_descargador: "Diputación de Granada · documento de demostración",
  marca_descargador: "Documento DEMO. En producción se emitirá desde el expediente firmado y custodiado.",
});

export const MENSAJES_RECIBO_PDF_EN = Object.freeze({
  portal_demo: "Employee Portal - DEMO",
  dato: "ITEM",
  valor: "VALUE",
  certifica: "CERTIFIES",
  firma_verificacion: "Signature and verification",
  referencia: "Reference: {referencia}",
  estado_demo: "Status: demonstration document with no administrative validity",
  comprobar: "Verify document",
  subtitulo_defecto: "Receipt issued by the Employee Portal",
  nota_defecto: "The final document will be generated and signed on the authorised server.",
  certificacion_defecto: "This records the stated action and its verification reference. The final document will be issued from the authorised administrative case file.",
  rotulo_recibo: "ACTION RECEIPT",
  rotulo_certificado: "CERTIFICATE",
  titulo_descargador: "Employee Portal receipt",
  subtitulo_descargador: "Diputación de Granada · demonstration document",
  marca_descargador: "DEMO document. In production it will be issued from the signed and safeguarded case file.",
});

const CLAVES = Object.keys(MENSAJES_RECIBO_PDF_ES);
export function crearTraductorReciboPDF(catalogo = MENSAJES_RECIBO_PDF_ES) {
  if (!catalogo || typeof catalogo !== "object" || Array.isArray(catalogo)
    || Object.keys(catalogo).length !== CLAVES.length
    || CLAVES.some((clave) => typeof catalogo[clave] !== "string" || !catalogo[clave].trim()
      || [...catalogo[clave].matchAll(/\{([a-z_]+)\}/gu)].map((m) => m[1]).sort().join(",")
        !== [...MENSAJES_RECIBO_PDF_ES[clave].matchAll(/\{([a-z_]+)\}/gu)].map((m) => m[1]).sort().join(","))) {
    throw new TypeError("catálogo del recibo PDF incompleto");
  }
  return (clave, variables = {}) => {
    if (!Object.hasOwn(MENSAJES_RECIBO_PDF_ES, clave)) throw new RangeError(`clave del recibo PDF desconocida: ${clave}`);
    return catalogo[clave].replace(/\{([a-z_]+)\}/gu, (_texto, nombre) => String(variables[nombre] ?? ""));
  };
}
