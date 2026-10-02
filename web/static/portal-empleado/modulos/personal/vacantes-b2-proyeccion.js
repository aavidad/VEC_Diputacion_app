/** Proyección de una página B2 ya autorizada y validada por su consumidor.
 * La ausencia de ocupación en una plaza no decide ocupación de puesto ni cobertura.
 * No interpreta actos, códigos de fuente ni versiones como decisiones jurídicas.
 */
function texto(valor, obligatorio = false) {
  if (!obligatorio && (valor === undefined || valor === "")) return "";
  if (typeof valor !== "string" || !valor || valor.length > 256 || /[\u0000-\u001f\u007f]/u.test(valor)) throw new TypeError("dato de vacantes incompatible");
  return valor;
}
// SQL10 admite 300 caracteres en denominaciones, no 300 unidades UTF-16.
function denominacion(valor) {
  if (valor === undefined || valor === "") return "";
  if (typeof valor !== "string" || [...valor].length > 300 || /[\u0000-\u001f\u007f]/u.test(valor)) throw new TypeError("denominación de vacantes incompatible");
  return valor;
}
function dato(codigoMensaje, valor, tipo = "texto") { return Object.freeze({ codigoMensaje, valor, tipo }); }

export function proyectarPaginaVacantesB2(pagina) {
  if (!pagina || pagina.cobertura !== "completa" || !pagina.corte ||
      !Array.isArray(pagina.vacantes) || pagina.vacantes.length > 100) throw new TypeError("cobertura de vacantes no acreditada");
  const corte = Object.freeze({ vigenteEn: texto(pagina.corte.vigente_en, true), conocidoEn: texto(pagina.corte.conocido_en, true) });
  const ambitoRef = texto(pagina.organismo_ref, true);
  const filas = pagina.vacantes.map((vacante) => {
    const traza = vacante?.traza;
    if (vacante?.estado_cobertura !== "vacante_sin_ocupacion" || !traza ||
        !Number.isSafeInteger(traza.revision_estructural) || traza.revision_estructural < 1 ||
        vacante.version_plantilla_ref !== traza.version_plantilla_ref ||
        typeof traza.fuente_huella_sha256 !== "string" || !/^[a-f0-9]{64}$/u.test(traza.fuente_huella_sha256)) throw new TypeError("origen de vacante incompatible");
    const plazaRef = texto(vacante.plaza_ref, true);
    const puestoRef = texto(vacante.puesto_ref);
    const visibles = Object.freeze([
      dato("desde", texto(traza.desde, true), "fecha"), dato("hasta", texto(traza.hasta), "fecha"),
      dato("registrada_en", texto(traza.registrada_en, true), "instante"), dato("revision_estructural", traza.revision_estructural, "numero"),
    ]);
    const tecnicos = Object.freeze([
      dato("plaza_ref", plazaRef), dato("puesto_ref", puestoRef), dato("unidad_ref", texto(vacante.unidad_ref, true)),
      dato("version_plantilla_ref", texto(vacante.version_plantilla_ref, true)), dato("version_rpt_ref", texto(vacante.version_rpt_ref)),
      dato("fuente_ref", texto(traza.fuente_ref, true)), dato("acto_ref", texto(traza.acto_ref, true)), dato("fuente_huella_sha256", traza.fuente_huella_sha256),
    ].filter((entrada) => entrada.valor !== ""));
    return Object.freeze({
      codigoPlaza: texto(vacante.codigo_plaza_fuente), unidad: denominacion(vacante.unidad_denominacion), puesto: puestoRef ? denominacion(vacante.puesto_denominacion) : "",
      dotacionMensaje: "sin_ocupacion", puestoMensaje: puestoRef ? "puesto_vinculado" : "puesto_no_consta", necesidadMensaje: "cubrible_pendiente",
      origenVisible: visibles, origenTecnico: tecnicos,
    });
  });
  return Object.freeze({ corte, ambitoRef, filas: Object.freeze(filas), hayPaginaSiguiente: Boolean(pagina.cursor_siguiente) });
}
