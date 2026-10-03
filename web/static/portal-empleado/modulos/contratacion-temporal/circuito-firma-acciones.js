/**
 * Vista del paso de firma. CT118 conserva su historia de prueba, pero esta
 * vista no ejecuta acciones hasta que el servidor acredite el preflight R5.
 */

import { escaparHTML } from "./componentes-expedientes.js?v=20261002-ct-fin-modalidad-v1";
import { PERFILES_BORRADOR_RRHH } from "./cliente-http-informe-definitivo.js?v=20261002-ct-fin-moad-v1";

/** Une el circuito del catálogo con el estado real registrado. */
export function fusionarEstadoFirmas(circuito, estado) {
  if (!circuito || !estado || estado.huella_sha256 !== circuito.huella_sha256) return null;
  const documentos = [];
  for (const documento of circuito.documentos) {
    const real = estado.documentos.find((d) => d.documento === documento.documento);
    if (!real || real.pasos.length !== documento.pasos.length) return null;
    documentos.push(Object.freeze({
      ...documento,
      paso_pendiente: real.paso_pendiente,
      pasos: Object.freeze(documento.pasos.map((paso, i) => Object.freeze({
        ...paso, estado: real.pasos[i].estado, motivo_devolucion: real.pasos[i].motivo_devolucion ?? "",
        registrada_en: real.pasos[i].registrada_en ?? "",
        documento_custodiado: real.pasos[i].documento_custodiado ?? null,
      }))),
    }));
  }
  return Object.freeze({ ...circuito, documentos: Object.freeze(documentos),
    registro: Object.freeze({ verificacion: estado.verificacion_disponible }) });
}

/**
 * Los controles de esta vista permanecen cerrados. El estado de CT118 y la
 * descarga del borrador no prueban custodia del original ni permiso nominal
 * para CT170. Tampoco prueban que Portafirmas esté conectado.
 */
export function renderizarAccionesPaso(circuito, documento, paso, t) {
  if (!circuito.registro || circuito.acciones === false || documento.paso_pendiente !== paso.orden
    || !Object.hasOwn(PERFILES_BORRADOR_RRHH, documento.documento)) return "";
  const avisoId = `ct-firma-resultado-${documento.documento}-${paso.orden}`;
  return `<div class="ct-circuito-acciones">
    <button type="button" class="boton-primario" disabled aria-describedby="${avisoId}">${escaparHTML(t("circuito_firma_firmar"))}</button>
    <button type="button" class="boton-secundario" disabled aria-describedby="${avisoId}">${escaparHTML(t("circuito_firma_devolver"))}</button>
    <p id="${avisoId}" class="ct-circuito-resultado" role="status">${escaparHTML(t("circuito_firma_vec_bloqueada"))}</p>
  </div>`;
}

/** Los clics sobre el circuito no tienen efecto hasta que exista preflight R5. */
export function crearAccionesFirma() {
  return Object.freeze({ manejarClic: async () => {} });
}
