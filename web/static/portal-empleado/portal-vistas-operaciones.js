/** Vistas compartidas de importación, llamamientos, relaciones, documentos y comunicaciones. */
import { traducirContratos } from "./portal-i18n-contratos.js";

export function crearVistasOperaciones(u) {
  const { escaparHTML: e, numero, fecha, chip, tabla, kpi, encabezadoVista,
    avisoPresentacion, botonOperacion, botonBloqueado, campo, fuentePresentacion } = u;

  function renderizarImportacion(datos) {
    const filas = datos.importaciones.map((item) => [e(item.id), e(item.origen), e(item.lote), `<code>${e(item.huella)}</code>`, numero(item.validas), numero(item.incidencias), e(item.autoridad), chip(item.estado), `<div class="acciones-fila">${botonOperacion("Validar lote", "validar-importacion", item.id, "boton-terciario")}${botonOperacion("Descartar", "descartar-importacion", item.id, "boton-terciario")}</div>`]);
    const detalles = datos.importaciones.flatMap((item) => (item.incidencias_detalle || []).map((incidencia) => [e(item.id), e(incidencia.fila), e(incidencia.campo), e(incidencia.motivo)]));
    const controles = datos.importaciones.flatMap((item) => (item.controles_formato || []).map((control) => [e(item.id), e(control.control), e(control.resultado)]));
    return `
      ${encabezadoVista("Carga gobernada", "Importación desde Convoca", "Recorrido visual de staging, validación y conciliación. La fuente nunca se considera autoritativa por sí sola.", botonOperacion("Simular nueva lectura", "validar-importacion", "DEMO-IMP-NUEVA", "boton-primario"))}
      ${avisoPresentacion("No se selecciona ni procesa ningún archivo del equipo. Los lotes son fixtures sintéticos ya incluidos en la demostración.")}
      <div class="rejilla-kpi">${kpi("LOT", numero(datos.importaciones.length), "Lotes")}${kpi("VAL", numero(datos.importaciones.reduce((s, x) => s + x.validas, 0)), "Filas válidas")}${kpi("INC", numero(datos.importaciones.reduce((s, x) => s + x.incidencias, 0)), "Incidencias")}${kpi("AUT", "0", "Altas automáticas")}</div>
      <section class="panel"><div class="cabecera-panel"><div><h3>Lotes en staging</h3><p>El original se identifica por huella y cada fila conserva procedencia.</p></div>${fuentePresentacion()}</div>${tabla({
        titulo: "Importaciones Convoca",
        cabeceras: ["Lote", "Origen", "Contenido", "Huella", "Válidas", "Incidencias", "Autoridad", "Estado", "Acción"],
        clavesColumnas: ["referencia", "origen", "contenido", "huella", "validas", "incidencias", "autoridad", "estado", "acciones"],
        prioridadColumnas: "estado-acciones",
        filas,
      })}</section>
      <div class="rejilla-dos-columnas panel-separado">
        <section class="panel"><div class="cabecera-panel"><div><h3>Incidencias detectadas</h3><p>Se revisan antes de cualquier validación DEMO; no se corrige ni se da de alta ninguna persona.</p></div><span class="estado-chip aviso">${numero(detalles.length)} pendientes</span></div>${tabla({
          titulo: "Incidencias por lote Convoca",
          cabeceras: ["Lote", "Fila", "Campo", "Motivo"],
          clavesColumnas: ["referencia", "fila", "campo", "motivo"],
          filas: detalles.length > 0 ? detalles : [["—", "—", "—", "Sin incidencias sintéticas registradas"]],
        })}</section>
        <section class="panel"><div class="cabecera-panel"><div><h3>Controles de formato</h3><p>Resultado visible por lote, sin leer archivos del equipo.</p></div><span class="estado-chip info">Solo DEMO</span></div>${tabla({
          titulo: "Controles de los lotes Convoca",
          cabeceras: ["Lote", "Control", "Resultado"],
          clavesColumnas: ["referencia", "control", "resultado"],
          filas: controles,
        })}</section>
      </div>
      <div class="rejilla-dos-columnas panel-separado"><section class="panel"><div class="cabecera-panel"><h3>Controles antes de conciliar</h3><span class="estado-chip violeta">Controles del formato</span></div><ul class="lista-comprobacion"><li>Formato y cabeceras exactos</li><li>Límites de tamaño, filas, columnas y celdas</li><li>Fórmulas y contenido activo rechazados</li><li>Normalización sin ocultar valores originales</li><li>Duplicados e incoherencias señalados</li><li>Acta con huella, actor y resultado</li><li class="pendiente">Conciliación corporativa pendiente de conector</li></ul></section><aside class="nota-pendiente"><strong>Bloqueo explícito.</strong> La importación no crea personas, contratos ni posiciones de Bolsa. ${botonBloqueado("Conciliar con datos corporativos", "Falta el conector corporativo autorizado, cifrado de identificadores y política de retención aprobada.")}</aside></div>`;
  }

  function renderizarLlamamientos(datos) {
    const filas = datos.llamamientos_demo.map((item) => [e(item.id), e(item.necesidad), e(item.bolsa), e(item.orden), numero(item.incluidos), e(item.plazo), e(item.canal), chip(item.estado), `<div class="acciones-fila">${botonOperacion("Preparar", "emitir-llamamiento", item.id, "boton-terciario")}${botonOperacion("Registrar respuesta", "registrar-respuesta", item.id, "boton-terciario")}</div>`]);
    return `
      ${encabezadoVista("Prelación y disponibilidad", "Llamamientos DEMO", "Propuesta, preparación y respuesta sintéticas, volátiles y sin efectos sobre Bolsa ni contactos.", botonOperacion("Nuevo llamamiento DEMO", "emitir-llamamiento", "DEMO-LLA-NUEVO", "boton-primario"))}
      ${avisoPresentacion("No se identifica ni contacta a personas reales. Orden, plazo, canal y respuesta son datos sintéticos; no fijan una regla ni un plazo operativo.")}
      <div class="rejilla-kpi">${kpi("PEN", numero(datos.llamamientos_demo.filter((x) => /pendiente/i.test(x.estado)).length), "Pendientes DEMO")}${kpi("PRE", numero(datos.llamamientos_demo.filter((x) => /preparado/i.test(x.estado)).length), "Preparados DEMO")}${kpi("RES", numero(datos.llamamientos_demo.filter((x) => /respuesta/i.test(x.estado)).length), "Respuestas DEMO")}${kpi("SIN", "0", "Envíos acreditados")}</div>
      <section class="panel"><div class="cabecera-panel"><div><h3>Expedientes de llamamiento</h3><p>La persona concreta solo se resolverá en el comando autorizado del servidor.</p></div>${fuentePresentacion()}</div>${tabla({ titulo: "Llamamientos", cabeceras: ["Llamamiento", "Necesidad", "Bolsa", "Regla", "Incluidos", "Plazo", "Canal", "Estado", "Acción"], filas })}</section>
      <section class="panel panel-separado"><div class="cabecera-panel"><div><h3>Configurar llamamiento</h3><p>Recorrido completo sin ejecutar conectores.</p></div><span class="estado-chip info">DEMO-LLA-NUEVO</span></div><form class="cuerpo-panel formulario-gobernado" data-comando="emitir-llamamiento"><fieldset><legend>Necesidad de cobertura</legend><div class="rejilla-formulario">${campo("Bolsa", '<select name="bolsa"><option>Auxiliar Administrativo</option><option>Trabajador Social</option></select>')}${campo("Destino", '<input name="destino" value="Centro DEMO 01" readonly>')}${campo("Jornada", '<select name="jornada"><option>Completa</option><option>Parcial 50 %</option><option>Parcial 33 %</option></select>')}${campo("Duración prevista", '<input name="duracion" value="3 meses">')}</div></fieldset><fieldset><legend>Contacto y respuesta</legend><div class="rejilla-formulario">${campo("Regla de orden", '<select name="regla"><option>Prelación estricta · v3</option></select>')}${campo("Plazo de respuesta", '<select name="plazo_respuesta"><option>24 horas</option><option>48 horas</option></select>')}${campo("Canales", '<select name="canales"><option>Correo + aviso interno</option><option>Notificación fehaciente</option></select>')}${campo("Plantilla", '<select name="plantilla"><option>DEMO-PLT-LLA-v3</option></select>')}</div></fieldset>${botonOperacion("Revisar y preparar", "emitir-llamamiento", "DEMO-LLA-NUEVO", "boton-primario")}</form></section>`;
  }

  function renderizarContratos(datos) {
    const t = traducirContratos;
    // Solo este campo explícito admite lecturas autorizadas; `datos.contratos`
    // pertenece al antiguo panel de presentación y nunca alimenta esta vista.
    const fuente = datos?.contratos_fuente;
    const estados = ["cargando", "disponible", "vacio", "no_configurado", "denegado", "error"];
    let estado = estados.includes(fuente?.estado) ? fuente.estado : "no_configurado";
    if (estado === "disponible" && !Array.isArray(fuente.registros)) estado = "error";
    const contratos = estado === "disponible" ? fuente.registros : [];
    if (estado === "disponible" && contratos.length === 0) estado = "vacio";
    const clasesEstado = { cargando: "info", disponible: "exito", vacio: "neutro", no_configurado: "aviso", denegado: "peligro", error: "peligro" };
    const filas = contratos.map((item) => [
      `<strong class="contratos-referencia">${e(item.expediente)}</strong>`,
      e(item.acto), e(item.bolsa),
      `<span class="contratos-fecha">${e(fecha(item.inicio))}</span><span class="contratos-fecha contratos-fecha-fin">${e(fecha(item.fin))}</span>`,
      chip(item.estado),
    ]);
    const pasos = [
      ["paso_bolsa", "paso_bolsa_descripcion"],
      ["paso_formalizacion", "paso_formalizacion_descripcion"],
      ["paso_personal", "paso_personal_descripcion"],
      ["paso_ginpix", "paso_ginpix_descripcion"],
      ["paso_reincorporacion", "paso_reincorporacion_descripcion"],
    ];
    const pasoHTML = pasos.map(([titulo, descripcion], indice) => `
      <li class="contratos-paso"><span class="contratos-paso-numero" aria-hidden="true">${indice + 1}</span>
        <div><strong>${e(t(titulo))}</strong><small>${e(t(descripcion))}</small></div></li>`).join("");
    const accionPendiente = (etiqueta, motivo, atributoOperacion) => `
      <div class="contratos-accion"><button class="boton-secundario" type="button" ${atributoOperacion} disabled aria-disabled="true" title="${e(t(motivo))}">${e(t(etiqueta))}</button>
        <small>${e(t(motivo))}</small></div>`;
    return `<div class="contratos-vista">
      ${encabezadoVista(t("sobrelinea"), t("titulo"), t("descripcion"))}
      <section class="nota-pendiente" role="status" aria-live="polite">${e(t(`detalle_${estado}`))}</section>
      <section class="panel contratos-panel-circuito" aria-labelledby="contratos-circuito-titulo">
        <div class="cabecera-panel"><div><h3 id="contratos-circuito-titulo">${e(t("circuito_titulo"))}</h3><p>${e(t("circuito_subtitulo"))}</p></div></div>
        <ol class="contratos-pasos">${pasoHTML}</ol>
      </section>
      <section class="panel contratos-panel-registros" aria-labelledby="contratos-registros-titulo">
        <div class="cabecera-panel"><div><h3 id="contratos-registros-titulo">${e(t("registros_titulo"))}</h3><p>${e(t("registros_subtitulo"))}</p></div><span class="estado-chip ${clasesEstado[estado]}">${e(t(`estado_${estado}`))}</span></div>
        ${tabla({ titulo: t("tabla_titulo"), cabeceras: [t("columna_expediente"), t("columna_acto"), t("columna_bolsa"), t("columna_fechas"), t("columna_estado")], clavesColumnas: ["referencia", "acto", "bolsa", "fechas", "estado"], prioridadColumnas: "estado", filas, vacio: t("vacio") })}
      </section>
      <section class="panel contratos-panel-acciones" aria-labelledby="contratos-acciones-titulo">
        <div class="cabecera-panel"><div><h3 id="contratos-acciones-titulo">${e(t("acciones_titulo"))}</h3><p>${e(t("acciones_subtitulo"))}</p></div></div>
        <div class="contratos-acciones">${accionPendiente("accion_contrato", "motivo_contrato", 'data-operacion="registrar-contrato"')}${accionPendiente("accion_cese", "motivo_cese", 'data-operacion="registrar-cese"')}${accionPendiente("accion_reincorporar", "motivo_reincorporar", 'data-operacion="reincorporar-bolsa"')}</div>
      </section>
      <details class="contratos-ayuda"><summary>${e(t("ayuda"))}</summary><p>${e(t("ayuda_contenido"))}</p></details>
    </div>`;
  }

  function renderizarDocumentos(datos) {
    const filas = datos.documentos.map((item) => [e(item.referencia), e(item.plantilla), e(item.formatos), e(item.version), chip(item.estado), `<div class="acciones-fila">${botonOperacion("Generar DEMO", "generar-documento", item.referencia, "boton-terciario")}${botonOperacion("Enviar a firma DEMO", "enviar-firma-documento", item.referencia, "boton-terciario")}${botonOperacion("Firmar DEMO · sin firma legal", "firmar-documento", item.referencia, "boton-terciario")}</div>`]);
    return `
      ${encabezadoVista("Expediente documental", "Documentos y circuito de firma DEMO", "Previsualización sintética de plantillas, formatos y circuito. La autenticación no firma documentos y ningún borrador es oficial.", botonOperacion("Generar previsualización DEMO", "generar-documento", "DEMO-DOC-NUEVO", "boton-primario"))}
      ${avisoPresentacion("No se crea ni descarga un fichero binario y no se invoca Autofirma. El recibo volátil acredita solo la simulación visual; no hay firma, sello, custodia ni CSV verificable.")}
      <section class="panel"><div class="cabecera-panel"><div><h3>Plantillas y documentos</h3><p>Una plantilla puede producir varios formatos mediante conectores de salida.</p></div>${fuentePresentacion()}</div>${tabla({ titulo: "Documentos", cabeceras: ["Referencia", "Plantilla", "Formatos", "Versión", "Estado", "Acciones"], filas })}</section>
      <div class="rejilla-dos-columnas panel-separado"><section class="panel"><div class="cabecera-panel"><h3>Preparar documento</h3><span class="estado-chip violeta">Metadatos no sensibles</span></div><form class="cuerpo-panel formulario-gobernado" data-comando="generar-documento"><fieldset><legend>Salida</legend><div class="rejilla-formulario">${campo("Plantilla", '<select name="plantilla"><option>Resolución de llamamiento v2</option><option>Listado provisional v5</option></select>')}${campo("Formato", '<select name="formato"><option>PDF/A</option><option>ODT</option><option>DOCX</option><option>CSV</option><option>JSON</option><option>TXT</option></select>')}${campo("Circuito", '<select name="circuito"><option>Jefatura → Secretaría → Delegación</option></select>')}${campo("Cotejo", '<select name="cotejo"><option>CSV + QR verificable</option><option>CSV visible</option></select>')}</div></fieldset>${botonOperacion("Generar previsualización", "generar-documento", "DEMO-DOC-NUEVO", "boton-primario")}</form></section><aside class="nota-seguridad"><strong>Custodia prevista.</strong> Cifrado, control de acceso por finalidad, versiones inmutables, antivirus intercambiable y registro de cada lectura o descarga.</aside></div>`;
  }

  function renderizarComunicaciones(datos) {
    const filas = datos.comunicaciones_demo.map((item) => [e(item.id), e(item.expediente), e(item.plantilla), e(item.canal), e(item.destinatario), e(item.acuse), chip(item.estado), `<div class="acciones-fila">${botonOperacion("Preparar DEMO", "preparar-comunicacion", item.id, "boton-terciario")}${botonOperacion("Simular envío DEMO · sin entrega", "enviar-comunicacion", item.id, "boton-terciario")}</div>`]);
    return `
      ${encabezadoVista("Conectores de comunicación", "Comunicaciones y notificaciones DEMO", "Contenido, canal y acuse sintéticos. Un aviso no acredita envío, entrega, notificación ni acuse.", botonOperacion("Nueva comunicación DEMO", "preparar-comunicacion", "DEMO-COM-NUEVA", "boton-primario"))}
      ${avisoPresentacion("Correo, Telegram y notificación permanecen desconectados. Simular envío solo cambia estado en memoria: no hay intento externo, entrega ni acuse acreditado.")}
      <section class="panel"><div class="cabecera-panel"><div><h3>Bandeja de comunicaciones</h3><p>Identificadores sintéticos; no hay direcciones, teléfonos ni usuarios reales.</p></div>${fuentePresentacion()}</div>${tabla({ titulo: "Comunicaciones", cabeceras: ["Referencia", "Expediente", "Plantilla", "Canal", "Destinatario", "Acuse", "Estado", "Acción"], filas })}</section>
      <section class="panel panel-separado"><div class="cabecera-panel"><h3>Componer comunicación DEMO</h3><span class="estado-chip info">Sin envío externo</span></div><form class="cuerpo-panel formulario-gobernado" data-comando="preparar-comunicacion"><fieldset><legend>Mensaje sintético</legend><div class="rejilla-formulario">${campo("Expediente", '<select name="expediente"><option>DEMO-LLA-045</option><option>DEMO-ALE-001</option></select>')}${campo("Plantilla", '<select name="plantilla"><option>Aviso de llamamiento v3</option><option>Resolución de alegación v2</option></select>')}${campo("Canales DEMO", '<select name="canales"><option>Correo + aviso interno</option><option>Notificación fehaciente</option></select>')}${campo("Plazo DEMO", '<input name="plazo" value="10 días hábiles">')}${campo("Asunto", '<input name="asunto" value="Comunicación DEMO del expediente">')}${campo("Contenido sintético", '<textarea name="contenido">Contenido sintético generado desde una plantilla versionada.</textarea>')}</div></fieldset>${botonOperacion("Preparar DEMO sin enviar", "preparar-comunicacion", "DEMO-COM-NUEVA", "boton-primario")}</form></section>`;
  }

  return Object.freeze({
    renderizarComunicaciones,
    renderizarContratos,
    renderizarDocumentos,
    renderizarImportacion,
    renderizarLlamamientos,
  });
}
