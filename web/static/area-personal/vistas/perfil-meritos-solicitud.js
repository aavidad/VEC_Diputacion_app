import {
  barraProgreso, botonOperacion, chip, encabezadoVista, enlaceRuta, escaparAtributo,
  escaparHTML, formatoPuntos, listaDatos, notaDemostracion, panel, tabla,
} from "./comunes.js";
import { estadoActosSolicitud, localizarSolicitudEdicion } from "../flujo-solicitud.js";
import { calcularAutobaremo } from "../calculo-autobaremo.js";

export function renderizarPerfil(datos) {
  const perfil = datos.perfil;
  const preferenciasAviso = datos.preferencias_notificacion || {};
  const formularioContacto = `<form id="formulario-contacto" data-operacion="actualizar_contacto">
    <div class="formulario-rejilla">
      <div class="campo"><label for="perfil-correo">Correo de avisos</label><input id="perfil-correo" name="correo" type="email" autocomplete="email" value="${escaparAtributo(perfil.correo)}" required><small>Los actos que requieran notificación fehaciente usarán el canal administrativo configurado.</small></div>
      <div class="campo"><label for="perfil-telefono">Teléfono de contacto</label><input id="perfil-telefono" name="telefono" type="tel" autocomplete="tel" value="${escaparAtributo(perfil.telefono)}" required></div>
      <div class="campo ancho-completo"><label for="perfil-domicilio">Domicilio a efectos de contacto</label><textarea id="perfil-domicilio" name="domicilio" autocomplete="street-address" required>${escaparHTML(perfil.domicilio)}</textarea></div>
    </div><div class="fila-acciones"><button type="submit" class="boton-primario">Revisar y guardar cambios</button></div></form>`;
  const preferencias = `<form id="formulario-notificaciones" data-operacion="actualizar_notificaciones"><fieldset><legend>Canales de aviso voluntarios</legend><label class="opcion-check"><input type="checkbox" name="correo" ${preferenciasAviso.correo ? "checked" : ""}><span><strong>Correo electrónico</strong><small>Avisos de plazos, cambios y llamamientos.</small></span></label><label class="opcion-check"><input type="checkbox" name="telegram" ${preferenciasAviso.telegram ? "checked" : ""}><span><strong>Telegram</strong><small>Se activará cuando el conector y el consentimiento estén disponibles.</small></span></label><label class="opcion-check"><input type="checkbox" name="interno" ${preferenciasAviso.interno ? "checked" : ""}><span><strong>Bandeja interna</strong><small>Siempre disponible dentro del área personal.</small></span></label></fieldset><p class="nota">Los avisos complementan, pero no sustituyen, una notificación administrativa cuando esta sea exigible.</p><button type="submit" class="boton-secundario">Guardar preferencias</button></form>`;

  return `${encabezadoVista("Perfil, identidad y contacto", "Una identidad común para solicitudes, bolsas, certificados y futuros módulos.")}
    ${datos.meta.presentacion ? notaDemostracion() : ""}
    <div class="rejilla-principal"><div>
      ${panel("Datos de identidad", "La identidad principal procede del sistema de autenticación", listaDatos([
        ["Nombre visible", escaparHTML(perfil.nombre_visible)], ["Identificador", escaparHTML(perfil.identificador_visible)],
        ["Referencia interna", escaparHTML(perfil.referencia)], ["Verificación", chip(perfil.estado_verificacion)],
        ["Provincia", escaparHTML(perfil.provincia)], ["Idioma", escaparHTML(perfil.idioma)],
      ]), { estado: perfil.estado_verificacion })}
      ${panel("Datos de contacto", "Puede proponer cambios; el servidor aplicará las validaciones y permisos", formularioContacto)}
    </div><aside>
      ${panel("Preferencias de comunicación", "Seleccione los avisos que desea recibir", preferencias)}
      ${panel("Privacidad y trazabilidad", "Control sobre el uso de sus datos", `<p>Los accesos y cambios relevantes quedan sujetos a registro de auditoría. Desde esta área podrá consultar el origen de los datos, rectificarlos cuando proceda y conocer qué documentos se reutilizan.</p>${enlaceRuta("ayuda", "Consultar protección de datos", "enlace-boton")}`)}
    </aside></div>`;
}

export function renderizarMeritos(datos) {
  const modoDemo = datos.meta.presentacion;
  const filasMeritos = datos.meritos.map((merito) => [
    `<strong>${escaparHTML(merito.titulo)}</strong><small>${escaparHTML(merito.id)}</small>`,
    escaparHTML(merito.tipo),
    `<span>${escaparHTML(merito.detalle)}</span><small>${formatoPuntos(merito.puntos_estimados)} puntos estimados</small>`,
    chip(merito.estado),
    `<div class="acciones-tabla"><button type="button" class="boton-secundario" data-accion="abrir-documento" data-id="${escaparAtributo(merito.documento_ref)}">Ver evidencia</button></div>`,
  ]);
  const filasDocumentos = datos.documentos.map((documento) => [
    `<strong>${escaparHTML(documento.nombre)}</strong><small>${escaparHTML(documento.id)}</small>`,
    escaparHTML(documento.tipo), escaparHTML(documento.fecha), chip(documento.estado),
    `<div class="acciones-tabla">${botonOperacion("solicitar_descarga", "Descargar", { id: documento.id, clase: "boton-secundario", descripcion: `Preparar descarga de ${documento.nombre}` })}</div>`,
  ]);
  const formulario = `<form id="formulario-merito" data-operacion="incorporar_merito"><div class="formulario-rejilla"><div class="campo"><label for="merito-tipo">Tipo de mérito</label><select id="merito-tipo" name="tipo" required><option value="">Seleccione</option><option>Titulación</option><option>Experiencia</option><option>Formación</option><option>Ejercicio superado</option><option>Otro mérito previsto en bases</option></select></div><div class="campo"><label for="merito-titulo">Denominación</label><input id="merito-titulo" name="titulo" required maxlength="180"></div><div class="campo"><label for="merito-jornada">Jornada, si corresponde</label><select id="merito-jornada" name="jornada"><option>No corresponde</option><option>Completa</option><option>Parcial 50 %</option><option>Parcial 33 %</option><option>Porcentaje distinto</option></select></div><div class="campo"><label for="merito-documento">Documento acreditativo</label><input id="merito-documento" name="documento" type="file" accept=".pdf,.odt,.docx,.jpg,.png"><small>${modoDemo ? "En demostración solo se usa el nombre y nunca se lee ni envía el fichero." : "El documento solo se incorporará cuando el servicio seguro confirme su custodia."}</small></div></div><p class="nota aviso">La aceptación y puntuación dependen de las bases de cada convocatoria. Aportar un mérito al inventario no implica su validación.</p><button type="submit" class="boton-primario">Revisar incorporación</button></form>`;

  return `${encabezadoVista("Méritos, títulos y documentos", "Inventario reutilizable con estado, evidencia y contribución a cada proceso.", `<button type="button" class="boton-primario" data-accion="enfocar-nuevo-merito">Añadir mérito</button>`)}
    <section class="resumen-cifras"><article class="cifra-resumen merito"><span>Méritos inventariados</span><strong>${datos.meritos.length}</strong><small>Todas las categorías</small></article><article class="cifra-resumen exito"><span>Validados</span><strong>${datos.meritos.filter((item) => item.estado === "Validado").length}</strong><small>Confirmados por personal técnico</small></article><article class="cifra-resumen aviso"><span>Pendientes o subsanables</span><strong>${datos.meritos.filter((item) => item.estado !== "Validado").length}</strong><small>Requieren revisión</small></article><article class="cifra-resumen"><span>Documentos</span><strong>${datos.documentos.length}</strong><small>Con versión y trazabilidad</small></article></section>
    ${panel("Inventario de méritos", "La puntuación se recalcula para las bases de cada proceso", tabla({ descripcion: "Méritos asociados al expediente personal", columnas: ["Mérito", "Tipo", "Detalle y estimación", "Estado", "Evidencia"], filas: filasMeritos }))}
    <div class="rejilla-dos"><div>${panel("Incorporar un mérito", "Aporte los datos y la evidencia disponible", formulario, { clase: "panel-nuevo-merito" })}</div><div>${panel("Documentos del expediente", "Descarga y estado de validación", tabla({ descripcion: "Documentos aportados u obtenidos de oficio", columnas: ["Documento", "Tipo", "Fecha", "Estado", "Acción"], filas: filasDocumentos }))}</div></div>`;
}

function pasosSolicitud(paso) {
  const etiquetas = ["Convocatoria", "Datos", "Méritos", "Autobaremo", "Pago, firma y registro"];
  return `<ol class="pasos" aria-label="Pasos de la solicitud">${etiquetas.map((etiqueta, indice) => `<li class="${paso === indice + 1 ? "activo" : paso > indice + 1 ? "completo" : ""}" ${paso === indice + 1 ? 'aria-current="step"' : ""}><span>Paso ${indice + 1}</span>${escaparHTML(etiqueta)}</li>`).join("")}</ol>`;
}

function accionesPaso(paso) {
  const etiqueta = paso === 4 ? "Guardar borrador y continuar" : "Guardar y continuar";
  return `<div class="fila-acciones">${paso > 1 ? '<button type="button" class="boton-secundario" data-accion="paso-anterior">Anterior</button>' : ""}${paso < 5 ? `<button type="submit" class="boton-primario">${etiqueta}</button>` : ""}</div>`;
}

function meritosAutobaremacion(datos, estado, convocatoriaId) {
  if (datos.resultado_autobaremo?.convocatoria_id === convocatoriaId
    && Array.isArray(datos.resultado_autobaremo.meritos_ids)) {
    return datos.resultado_autobaremo.meritos_ids;
  }
  if (estado.progresoSolicitud?.convocatoria_id === convocatoriaId
    && estado.progresoSolicitud.meritos_ids?.length) {
    return estado.progresoSolicitud.meritos_ids;
  }
  const borrador = localizarSolicitudEdicion(datos, {
    solicitudId: estado.solicitudEdicionId,
    convocatoriaId,
  });
  if (borrador?.meritos_ids?.length) return borrador.meritos_ids;
  return datos.meritos.map((item) => item.id);
}

function contenidoPaso(datos, estado, convocatoria) {
  const paso = estado.pasoSolicitud;
  if (paso === 1) {
    const disponibles = datos.convocatorias.filter((item) => item.estado === "Plazo abierto" || (datos.meta.presentacion && item.recorrido_demo === true));
    const requisitosConfirmados = estado.progresoSolicitud?.requisitos_confirmados === true;
    const leyenda = datos.meta.presentacion ? "Referencia pública para simular la solicitud" : "Convocatoria con plazo abierto";
    const aviso = datos.meta.presentacion ? "Referencia BOP real · inscripción y plazo exclusivamente DEMO" : "El servicio comprobará el plazo vigente";
    return `<fieldset><legend>${escaparHTML(leyenda)}</legend>${disponibles.map((item) => `<label class="opcion-check"><input type="radio" name="convocatoria" value="${escaparAtributo(item.id)}" ${item.id === convocatoria.id ? "checked" : ""} required data-accion="seleccionar-convocatoria"><span><strong>${escaparHTML(item.titulo)}</strong><small>${escaparHTML(item.referencia)} · ${escaparHTML(aviso)}</small></span></label>`).join("")}</fieldset><label class="opcion-check"><input type="checkbox" name="requisitos_confirmados" value="true" required ${requisitosConfirmados ? "checked" : ""}><span><strong>He leído las bases y declaro cumplir los requisitos</strong><small>${datos.meta.presentacion ? "Esta declaración y todo el expediente son sintéticos y no presentan una solicitud real." : "El servicio volverá a comprobar plazo, requisitos y causas de exclusión antes del registro."}</small></span></label>`;
  }
  if (paso === 2) {
    return `${listaDatos([["Identidad", escaparHTML(datos.perfil.nombre_visible)], ["Identificador", escaparHTML(datos.perfil.identificador_visible)], ["Correo", escaparHTML(datos.perfil.correo)], ["Teléfono", escaparHTML(datos.perfil.telefono)], ["Verificación", chip(datos.perfil.estado_verificacion)]])}<label class="opcion-check"><input type="checkbox" name="datos_confirmados" value="true" required ${estado.progresoSolicitud?.datos_confirmados === true ? "checked" : ""}><span><strong>Confirmo que mis datos personales y de contacto son correctos</strong><small>Puede modificarlos desde Perfil y contacto antes de continuar.</small></span></label>`;
  }
  if (paso === 3) {
    const seleccionados = new Set(estado.progresoSolicitud?.meritos_ids || []);
    return `<fieldset><legend>Méritos que desea asociar</legend>${datos.meritos.map((item) => `<label class="opcion-check"><input type="checkbox" name="meritos" value="${escaparAtributo(item.id)}" ${seleccionados.has(item.id) ? "checked" : ""}><span><strong>${escaparHTML(item.titulo)}</strong><small>${escaparHTML(item.estado)} · ${formatoPuntos(item.puntos_estimados)} puntos estimados</small></span></label>`).join("")}</fieldset><p class="nota aviso">Seleccione al menos un mérito. Los documentos se reutilizan sin duplicar el fichero y la solicitud conserva la referencia y versión exactas.</p>`;
  }
  if (paso === 4) {
    const calculo = calcularAutobaremo(datos, estado.progresoSolicitud?.meritos_ids);
    return `<div class="rejilla-dos"><div>${calculo.criterios.map((item) => `<div class="criterio-baremo"><span><strong>${escaparHTML(item.nombre)}</strong><small>${escaparHTML(item.detalle)}</small></span>${barraProgreso(item.puntos, item.maximo)}<output>${formatoPuntos(item.puntos)}</output></div>`).join("")}</div><aside class="puntuacion-total"><span>Total autobaremado para los méritos seleccionados y datos de oficio</span><output>${formatoPuntos(calculo.total)}</output><span>puntos provisionales</span></aside></div><p class="nota aviso">El cálculo usa los méritos elegidos y los conceptos obtenidos de oficio para esta convocatoria. No vincula a RRHH y cada concepto será revisado conforme a las bases.</p>`;
  }
  const demo = datos.meta.presentacion;
  const solicitud = localizarSolicitudEdicion(datos, {
    solicitudId: estado.solicitudEdicionId,
    convocatoriaId: convocatoria.id,
  });
  if (!solicitud) return '<p class="nota error" role="alert"><strong>Borrador no disponible.</strong> Vuelva al paso anterior y guárdelo antes de iniciar pago, firma o registro.</p>';
  const actos = estadoActosSolicitud(solicitud);
  const pago = actos.pagoConfirmado
    ? chip(demo ? "Pago o exención DEMO confirmado" : "Pago o exención confirmado")
    : botonOperacion("iniciar_pago", demo ? "Simular pago o exención" : "Pagar o acreditar exención", { id: solicitud.id, descripcion: demo ? "Confirmar la simulación de pago o exención" : "Confirmar el pago o la acreditación de exención" });
  const firma = actos.firmaConfirmada
    ? chip(demo ? "Firma DEMO confirmada" : "Firma confirmada")
    : actos.pagoConfirmado
      ? botonOperacion("firmar_solicitud", demo ? "Simular firma" : "Firmar solicitud", { id: solicitud.id, descripcion: demo ? "Confirmar la simulación de firma electrónica" : "Firmar electrónicamente la solicitud" })
      : '<button type="button" class="boton-secundario" disabled aria-disabled="true" title="Confirme antes el pago o la exención">Firma bloqueada hasta confirmar pago o exención</button>';
  let registro = `<p class="nota ${demo ? "demo" : ""}"><strong>${demo ? "Registro DEMO completado." : "Solicitud registrada."}</strong> ${demo ? "Consulte el recibo mostrado; no existe asiento administrativo real." : "Conserve el recibo y el asiento devueltos por el servicio."}</p>`;
  if (!actos.registrada) {
    registro = actos.pagoConfirmado && actos.firmaConfirmada
      ? `<form id="formulario-registro-solicitud" data-operacion="registrar_solicitud" data-id="${escaparAtributo(solicitud.id)}"><label class="opcion-check"><input type="checkbox" name="declaracion_final" value="true" required><span><strong>Confirmo la solicitud completa que se va a presentar</strong><small>He revisado convocatoria, requisitos, datos, méritos, autobaremación, tasa o exención y firma.</small></span></label><button type="submit" class="boton-primario">${demo ? "Registrar solicitud DEMO" : "Registrar solicitud"}</button></form>`
      : '<p class="nota aviso"><strong>Registro bloqueado.</strong> Debe confirmar primero el pago o la exención y la firma.</p>';
  }
  return `<p class="nota"><strong>Borrador seleccionado:</strong> ${escaparHTML(solicitud.id)}</p><div class="rejilla-dos"><section><h3>1. Tasa o exención</h3><p>Importe mostrado: <strong>${escaparHTML(convocatoria.tasa)}</strong></p>${pago}</section><section><h3>2. Firma electrónica</h3><p>Se firmará la representación exacta de la solicitud.</p>${firma}</section></div><section class="panel separacion-superior"><div class="panel-contenido"><h3>3. Registro</h3><p>El registro es el acto que presenta la solicitud y solo se habilita tras pago o exención y firma.</p>${registro}</div></section>`;
}

export function renderizarSolicitud(datos, estado) {
  const convocatoria = datos.convocatorias.find((item) => item.id === estado.convocatoriaSolicitud) || datos.convocatorias.find((item) => item.estado === "Plazo abierto") || datos.convocatorias[0];
  const solicitud = localizarSolicitudEdicion(datos, {
    solicitudId: estado.solicitudEdicionId,
    convocatoriaId: convocatoria.id,
  });
  const error = estado.errorPasoSolicitud
    ? `<p class="nota error" role="alert"><strong>No se puede continuar.</strong> ${escaparHTML(estado.errorPasoSolicitud)}</p>`
    : "";
  const contenido = `${pasosSolicitud(estado.pasoSolicitud)}${error}<section class="panel"><header><div><h3>Paso ${estado.pasoSolicitud} de 5</h3><p>Los controles definitivos se reutilizan con el cliente productivo</p></div>${chip(estado.pasoSolicitud === 5 ? "Revisión final" : "En preparación")}</header><div class="panel-contenido">${contenidoPaso(datos, estado, convocatoria)}</div></section>`;
  const asistente = estado.pasoSolicitud < 5
    ? `<form id="formulario-solicitud-paso" data-paso="${estado.pasoSolicitud}">${contenido}${accionesPaso(estado.pasoSolicitud)}</form>`
    : contenido;
  return `${encabezadoVista("Nueva solicitud", `${convocatoria.titulo} · ${convocatoria.referencia}`, chip(solicitud?.estado || "Borrador sin guardar"))}${asistente}`;
}

export function renderizarAutobaremacion(datos, estado = {}) {
  const convocatoria = datos.convocatorias.find((item) => item.id === estado.convocatoriaSolicitud)
    || datos.convocatorias.find((item) => item.estado === "Plazo abierto")
    || datos.convocatorias[0];
  const meritosIds = meritosAutobaremacion(datos, estado, convocatoria?.id || "");
  const calculo = calcularAutobaremo(datos, meritosIds);
  const criterios = calculo.criterios.map((criterio) => `<article class="criterio-baremo"><span><strong>${escaparHTML(criterio.nombre)}</strong><small>${escaparHTML(criterio.detalle)} · ${escaparHTML(criterio.estado)}</small></span>${barraProgreso(criterio.puntos, criterio.maximo)}<output>${formatoPuntos(criterio.puntos)}</output></article>`).join("");
  const recalculado = datos.resultado_autobaremo?.convocatoria_id === convocatoria?.id
    ? `<p class="nota ${datos.meta.presentacion ? "demo" : ""}"><strong>Resultado recalculado.</strong> ${escaparHTML(datos.resultado_autobaremo.calculado_en)} · ${meritosIds.length} méritos.</p>`
    : "";
  return `${encabezadoVista("Autobaremación desglosada", "Estimación trazable aplicada a la versión de bases de la convocatoria.", botonOperacion("calcular_autobaremo", datos.meta.presentacion ? "Recalcular autobaremo DEMO" : "Recalcular autobaremo", { id: convocatoria?.id || "", descripcion: "Recalcular la autobaremación con los méritos seleccionados" }))}
    ${recalculado}<div class="rejilla-principal"><div>${panel("Criterios aplicados", `${convocatoria?.titulo || "Convocatoria seleccionada"} · ${datos.meta.presentacion ? "referencia pública real; reglas y puntuaciones DEMO" : "versión vigente de bases"}`, `<div class="desglose-baremo">${criterios}</div>`)}</div><aside>${panel("Resultado provisional", `${meritosIds.length} méritos seleccionados más conceptos de oficio`, `<div class="puntuacion-total"><span>Puntuación estimada</span><output>${formatoPuntos(calculo.total)}</output><span>de ${formatoPuntos(calculo.maximo)} posibles</span></div><p>${barraProgreso(calculo.total, calculo.maximo)}</p><p class="nota aviso">No constituye puntuación oficial. RRHH aceptará, rechazará o ajustará cada concepto con motivación y trazabilidad.</p>`, { estado: "Provisional" })}${panel("Qué se tendrá en cuenta", "Reglas parametrizadas desde las bases", `<ul><li>Periodos exactos y solapamientos.</li><li>Porcentaje de jornada y reducciones.</li><li>Administración, categoría y rama.</li><li>Topes por bloque y puntuación máxima.</li><li>Documentos de oficio y aportados.</li></ul>`)}</aside></div>`;
}
