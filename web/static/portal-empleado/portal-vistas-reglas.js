/** Vista de consulta de gobierno y versionado del motor de reglas. */
import { traducirBolsaInterna } from "./portal-i18n.js";

export function crearVistaReglas(u) {
  const { escaparHTML: e, numero, fecha, chip, tabla, kpi, encabezadoVista,
    avisoPresentacion, fuentePresentacion, esPresentacion, botonOperacion, campo } = u;
  function normalizarEstado(estado = {}, datos = {}) {
    const fase = estado.fase || estado.carga;
    if (!fase) {
      // El consumidor todavía no transmite el resultado de consulta. Las filas
      // recibidas sí se pueden presentar, pero un objeto vacío no acredita vacío.
      return (Array.isArray(datos.reglas) && datos.reglas.length > 0)
        || (Array.isArray(datos.criterios_baremo) && datos.criterios_baremo.length > 0)
        ? "listo" : "sin_evidencia";
    }
    return ["cargando", "error", "denegado", "listo"].includes(fase) ? fase : "error";
  }
  function renderizarEstadoNoDisponible(fase, detalle) {
    const contenido = {
      cargando: ["Cargando versiones de reglas…", "La consulta interna sigue pendiente de respuesta."],
      error: ["No se pudieron consultar las reglas", detalle || "Revise la conexión autorizada e inténtelo de nuevo."],
      denegado: ["Acceso a reglas denegado", "No se han mostrado versiones ni criterios porque falta una concesión vigente y exacta."],
      sin_evidencia: [traducirBolsaInterna("fuente_real_no_conectada"), traducirBolsaInterna("detalle_funcionalidad_no_conectada")],
    }[fase];
    return `${encabezadoVista("Gobierno del motor", "Reglas, versiones y configuración", "Consulta interna de versiones gobernadas.")}
      <section class="panel"><div class="cuerpo-panel vacio-controlado" role="status" aria-live="polite">
        <p><strong>${e(contenido[0])}</strong></p><p>${e(contenido[1])}</p>
      </div></section>`;
  }
  function renderizarEdicionNoConectada(objetivo) {
    return `<form class="cuerpo-panel formulario-gobernado vacio-controlado" aria-label="Edición de reglas no disponible" data-comando="guardar-reglas-baremo">
      <p><strong>Edición de versiones no conectada.</strong></p>
      <p>La creación, validación, aprobación, publicación y activación requieren el circuito autorizado de reglas. Esta pantalla no genera borradores ni modifica versiones.</p>
      <button type="button" class="boton-primario" data-accion="operacion-presentacion" data-comando="guardar-reglas-baremo" data-operacion="guardar-reglas-baremo" data-objetivo="${e(objetivo)}" disabled aria-disabled="true" title="Capacidad de servidor no conectada">Guardar borrador de versión</button>
    </form>`;
  }
  function renderizarEdicionPresentacion(objetivo) {
    return `<form class="cuerpo-panel formulario-gobernado" aria-label="Borrador DEMO de reglas" data-comando="guardar-reglas-baremo">
      <p class="nota-seguridad"><strong>Modo presentación.</strong> Guarda un borrador DEMO en memoria y desaparece al recargar. No valida, aprueba, publica ni activa reglas.</p>
      <fieldset><legend>Parámetros del borrador DEMO</legend><div class="rejilla-formulario">${campo("Unidad de tiempo", '<select name="unidad_tiempo"><option value="mes">Mes</option></select>')}${campo("Puntos por unidad", '<input name="puntos_unidad" value="0">')}${campo("Fracción de jornada", '<select name="fraccion_jornada"><option value="proporcional">Proporcional</option></select>')}${campo("Tope del bloque", '<input name="tope_bloque" value="0">')}${campo("Ámbito de experiencia", '<select name="ambito_experiencia"><option value="bolsa_demo">Bolsa DEMO</option></select>')}${campo("Redondeo", '<select name="redondeo"><option value="dos_decimales">Dos decimales</option></select>')}${campo("Primer desempate", '<select name="desempate_1"><option value="experiencia">Experiencia</option></select>')}${campo("Segundo desempate", '<select name="desempate_2"><option value="formacion">Formación</option></select>')}${campo("Tercer desempate", '<select name="desempate_3"><option value="solicitud">Solicitud</option></select>')}${campo("Último recurso", '<select name="ultimo_recurso"><option value="revision_manual">Revisión manual</option></select>')}</div></fieldset>${botonOperacion("Guardar borrador DEMO", "guardar-reglas-baremo", objetivo, "boton-primario")}
    </form>`;
  }
  function renderizarReglas(datos = {}, estado = {}) {
    const fase = normalizarEstado(estado, datos);
    if (fase !== "listo") return renderizarEstadoNoDisponible(fase, estado.detalle || estado.error);
    const reglas = Array.isArray(datos.reglas) ? datos.reglas : [];
    const criterios = Array.isArray(datos.criterios_baremo) ? datos.criterios_baremo : [];
    const publicadas = reglas.filter((item) => /publicada/i.test(item.estado)).length;
    const enValidacion = reglas.filter((item) => /validación|revisión/i.test(item.estado)).length;
    const objetivoBorrador = criterios[0]?.id || reglas[0]?.version || "reglas-no-conectadas";
    const versiones = reglas.map((item) => [
      `<strong>${e(item.nombre)}</strong>`, e(item.ambito), e(item.version), e(fecha(item.vigencia)),
      chip(item.estado), e(item.procedencia || "Procedencia no aportada"),
    ]);
    const configuracion = criterios.map((item) => [
      `<strong>${e(item.id)}</strong>`, e(item.bloque), e(item.criterio), e(item.formula),
      e(item.maximo), e(item.version), chip(item.estado),
    ]);
    return `
      ${encabezadoVista("Gobierno del motor", "Reglas, versiones y configuración", "Consulta de versiones, vigencia y procedencia; no acredita aprobación, firma ni activación.")}
      ${avisoPresentacion("Las versiones y criterios mostrados pueden ser sintéticos. La pantalla solo los presenta: no crea, valida, publica ni activa reglas.")}
      <div class="rejilla-kpi">${kpi("VER", numero(reglas.length), "Versiones")}${kpi("PUB", numero(publicadas), "Publicadas")}${kpi("VAL", numero(enValidacion), "En validación")}${kpi("CRI", numero(criterios.length), "Criterios configurados")}</div>
      <div class="rejilla-dos-columnas rejilla-dos-columnas--tabla-densa">
        <section class="panel">
          <div class="cabecera-panel"><div><h3>Registro de versiones</h3><p>La fuente, la versión y la vigencia se muestran tal como se recibieron; una procedencia ausente se señala expresamente.</p></div>${fuentePresentacion()}</div>
          ${tabla({
            titulo: "Versiones gobernadas del motor de reglas",
            cabeceras: ["Regla", "Ámbito", "Versión", "Vigencia", "Estado", "Procedencia"],
            clavesColumnas: ["regla", "ambito", "version", "vigencia", "estado", "procedencia"],
            prioridadColumnas: "estado-acciones", filas: versiones,
            vacio: "No hay versiones autorizadas que mostrar.",
          })}
        </section>
        <aside class="resumen-lateral">
          <section class="panel"><div class="cabecera-panel"><h3>Ciclo de gobierno</h3><span class="estado-chip violeta">Pendiente de conexión</span></div><ol class="lista-comprobacion"><li>Borrador vinculado a bases y fuente jurídica</li><li>Pruebas con casos ordinarios y límite</li><li>Validación técnica y jurídica</li><li>Aprobación, firma y fecha de vigencia</li><li>Sustitución sin borrar el historial</li></ol></section>
          <section class="nota-seguridad"><strong>Inmutabilidad funcional.</strong> Los expedientes y cálculos deben conservar la versión exacta aplicada; una corrección requiere una versión sucesora.</section>
        </aside>
      </div>
      <section class="panel panel-separado">
        <div class="cabecera-panel"><div><h3>Configuración de criterios</h3><p>Vista de consulta previa a cualquier cálculo de baremación.</p></div></div>
        ${tabla({
          titulo: "Configuración de criterios",
          cabeceras: ["Criterio", "Bloque", "Descripción", "Fórmula", "Máximo", "Versión", "Estado"],
          clavesColumnas: ["referencia", "bloque", "descripcion", "formula", "maximo", "version", "estado"],
          prioridadColumnas: "estado", filas: configuracion,
          vacio: "No hay criterios autorizados que mostrar.",
        })}
      </section>
      <section class="panel panel-separado" aria-label="Edición de reglas no disponible">
        <div class="cabecera-panel"><div><h3>Preparar la siguiente versión</h3><p>Disponible cuando exista el circuito HTTP autorizado y compuesto.</p></div>${fuentePresentacion()}</div>
        ${typeof esPresentacion === "function" && esPresentacion() ? renderizarEdicionPresentacion(objetivoBorrador) : renderizarEdicionNoConectada(objetivoBorrador)}
      </section>`;
  }
  return Object.freeze({ renderizarReglas });
}
