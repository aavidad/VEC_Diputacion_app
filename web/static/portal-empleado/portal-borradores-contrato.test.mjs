import assert from "node:assert/strict";
import test from "node:test";
import {
  generarClaveIdempotencia,
  validarClaveIdempotencia,
} from "./portal-borradores-api.js";
import {
  ESQUEMAS_BORRADORES,
  derivarETagBorrador,
  extraerDatosEnvelopeBorradores,
  extraerErrorEnvelopeBorradores,
  validarContenidoEditable,
  validarDetalleBorrador,
  validarListaBorradores,
  validarOpcionesBorradores,
  validarReciboGuardadoBorrador,
  validarSolicitudActualizarBorrador,
  validarSolicitudCrearBorrador,
} from "./portal-borradores-contrato.js";
import {
  CLAVE_IDEMPOTENCIA_A,
  CLAVE_IDEMPOTENCIA_B,
  HUELLA_A,
  HUELLA_B,
  configuracionLectura,
  contenido,
  crearRespuestaControlada,
  detalle,
  etagEstado,
  fila,
  limites,
  lista,
  opciones,
  recibo,
  referenciaEstado,
  respuestaError,
  respuestaJSON,
  solicitudActualizar,
  solicitudCrear,
} from "./portal-borradores-fixtures.test-helper.mjs";

test("opciones versionadas y límites proceden íntegramente de la API", () => {
  const validado = validarOpcionesBorradores(opciones());
  assert.equal(validado.categorias[0].version, 7);
  assert.equal(validado.tipos[0].clave, "bolsa_temporal");
  assert.equal(validado.plantillas[0].huella_sha256, HUELLA_A);
  assert.equal(validado.limites.maximo_categorias, 1024);
  const conUsoCodificado = structuredClone(opciones());
  conUsoCodificado.tipos[0].uso = "convocatoria";
  assert.throws(() => validarOpcionesBorradores(conUsoCodificado), /contrato cerrado/);
  const mayuscula = structuredClone(opciones());
  mayuscula.tipos[0].clave = "Bolsa_Temporal";
  assert.throws(() => validarOpcionesBorradores(mayuscula), /no válida/);
  const claveAmbigua = structuredClone(opciones());
  claveAmbigua.categorias.push({
    ...claveAmbigua.categorias[0], categoria_ref: "catalogo:categorias:otro",
    version: 8, huella_sha256: HUELLA_B, etiqueta: "Etiqueta contradictoria",
  });
  assert.throws(() => validarOpcionesBorradores(claveAmbigua), /claves visibles ambiguas/);

  const variasClavesCoherentes = structuredClone(opciones());
  variasClavesCoherentes.categorias.push({
    ...variasClavesCoherentes.categorias[0],
    clave: "auxiliar_administrativo_turno_libre",
    etiqueta: "Auxiliar administrativo — turno libre",
  });
  assert.equal(
    validarOpcionesBorradores(variasClavesCoherentes).categorias.length,
    2,
    "una identidad versionada puede proyectar varias claves si conserva una única huella",
  );
  const huellaContradictoria = structuredClone(variasClavesCoherentes);
  huellaContradictoria.categorias[1].huella_sha256 = HUELLA_B;
  assert.throws(
    () => validarOpcionesBorradores(huellaContradictoria),
    /huellas contradictorias para catálogo y versión/,
  );
  const tipoContradictorio = structuredClone(opciones());
  tipoContradictorio.tipos.push({
    ...tipoContradictorio.tipos[0],
    clave: "bolsa_temporal_especifica",
    etiqueta: "Bolsa temporal específica",
    huella_sha256: HUELLA_A,
  });
  assert.throws(
    () => validarOpcionesBorradores(tipoContradictorio),
    /huellas contradictorias para catálogo y versión/,
  );
  const absolutos = {
    maximo_categorias: 1024, maximo_plazos: 64, maximo_requisitos: 256,
    maximo_documentos: 256, maximo_ayudas: 128, maximo_titulo: 180,
    maximo_resumen: 500, maximo_descripcion: 12_000, maximo_titulo_plazo: 180,
    maximo_descripcion_plazo: 1_000, maximo_titulo_requisito: 180,
    maximo_descripcion_requisito: 3_000, maximo_pregunta_ayuda: 300,
    maximo_respuesta_ayuda: 5_000,
  };
  for (const [campo, maximo] of Object.entries(absolutos)) {
    const desbordada = structuredClone(opciones());
    desbordada.limites[campo] = maximo + 1;
    assert.throws(() => validarOpcionesBorradores(desbordada), new RegExp(campo));
  }
  const sustitutoAislado = structuredClone(opciones());
  sustitutoAislado.categorias[0].etiqueta = `Categoría\uD800`;
  assert.throws(() => validarOpcionesBorradores(sustitutoAislado), /no válido/);
});

test("lista por cursor exige referencia de estado completa y máximo cincuenta", () => {
  const datos = lista();
  datos.selector = { limite: 40, cursor: "cursor:opaco:1", texto: "temporal", categoria: "auxiliar_administrativo" };
  datos.paginacion.total = 2;
  datos.paginacion.siguiente_cursor = "cursor:opaco:2";
  const validado = validarListaBorradores(datos);
  assert.equal(validado.elementos[0].referencia_estado.revision, 3);
  assert.equal(validado.paginacion.siguiente_cursor, "cursor:opaco:2");
  const sinHuella = lista();
  delete sinHuella.elementos[0].referencia_estado.huella_estado_sha256;
  assert.throws(() => validarListaBorradores(sinHuella), /contrato cerrado/);
  const porPagina = lista();
  porPagina.selector = { pagina: 1, tamano: 40 };
  assert.throws(() => validarListaBorradores(porPagina), /contrato cerrado/);
  const exceso = lista();
  exceso.selector.limite = 51;
  exceso.paginacion.limite = 51;
  assert.throws(() => validarListaBorradores(exceso), /límite no válido/);

  const vacia = lista();
  vacia.elementos = [];
  vacia.paginacion.total = 0;
  assert.equal(validarListaBorradores(vacia).elementos.length, 0);
  vacia.paginacion.siguiente_cursor = "cursor:imposible:1";
  assert.throws(() => validarListaBorradores(vacia), /siguiente cursor incoherente/);

  const sinElementosConCursor = lista();
  sinElementosConCursor.elementos = [];
  sinElementosConCursor.paginacion.siguiente_cursor = "cursor:imposible:1";
  assert.throws(() => validarListaBorradores(sinElementosConCursor), /vacío|siguiente cursor incoherente/);

  const fechasInvertidas = lista();
  fechasInvertidas.elementos[0].creada_en = "2026-07-18T10:00:00Z";
  fechasInvertidas.elementos[0].actualizada_en = "2026-07-18T09:00:00Z";
  assert.throws(() => validarListaBorradores(fechasInvertidas), /fechas incoherentes/);
  const fechasMicro = lista();
  fechasMicro.elementos[0].creada_en = "2026-07-18T09:00:00.123455Z";
  fechasMicro.elementos[0].actualizada_en = "2026-07-18T09:00:00.123456Z";
  assert.doesNotThrow(() => validarListaBorradores(fechasMicro));
  fechasMicro.elementos[0].creada_en = "2026-07-18T09:00:00.123457Z";
  assert.throws(() => validarListaBorradores(fechasMicro), /fechas incoherentes/);

  const demasiadosDocumentos = lista();
  demasiadosDocumentos.elementos[0].numero_documentos = 257;
  assert.throws(() => validarListaBorradores(demasiadosDocumentos), /numero_documentos no válido/);
});

test("contenido completo valida plazos, requisitos y ayuda con límites recibidos", () => {
  const validado = validarContenidoEditable(contenido(), limites());
  assert.equal(validado.plazos[0].tipo, "presentacion_solicitudes");
  assert.equal(validado.requisitos[0].obligatorio, true);
  const intervalo = contenido();
  intervalo.plazos[0].cierra_en = intervalo.plazos[0].abre_en;
  assert.throws(() => validarContenidoEditable(intervalo, limites()), /intervalo válido/);
  const limiteAPI = limites();
  limiteAPI.maximo_requisitos = 1;
  const demasiados = contenido();
  demasiados.requisitos.push({ ...demasiados.requisitos[0], referencia: "requisito:otro:2", orden: 2 });
  assert.throws(() => validarContenidoEditable(demasiados, limiteAPI), /requisitos no válida/);
  const ordenRepetido = contenido();
  ordenRepetido.requisitos.push({
    ...ordenRepetido.requisitos[0], referencia: "requisito:otro:2",
  });
  assert.throws(() => validarContenidoEditable(ordenRepetido, limites()), /órdenes repetidos/);
  const referenciaRepetida = contenido();
  referenciaRepetida.requisitos.push({ ...referenciaRepetida.requisitos[0], orden: 2 });
  assert.throws(() => validarContenidoEditable(referenciaRepetida, limites()), /elementos repetidos/);
  const ordenAyudaRepetido = contenido();
  ordenAyudaRepetido.ayuda.push({
    ...ordenAyudaRepetido.ayuda[0], referencia: "ayuda:otra:2",
  });
  assert.throws(() => validarContenidoEditable(ordenAyudaRepetido, limites()), /órdenes repetidos/);
  const referenciaAyudaRepetida = contenido();
  referenciaAyudaRepetida.ayuda.push({ ...referenciaAyudaRepetida.ayuda[0], orden: 2 });
  assert.throws(() => validarContenidoEditable(referenciaAyudaRepetida, limites()), /elementos repetidos/);

  for (const imposible of [
    "2026-02-30T08:00:00Z", "2026-13-01T08:00:00Z", "2026-00-01T08:00:00Z",
    "2026-01-32T08:00:00Z", "2026-01-01T24:00:00Z", "2026-01-01T08:60:00Z",
    "2026-01-01T08:00:60Z", "2026-01-01T08:00:00.1234567Z", "0000-01-01T08:00:00Z",
  ]) {
    const fechaInvalida = contenido();
    fechaInvalida.plazos[0].abre_en = imposible;
    assert.throws(() => validarContenidoEditable(fechaInvalida, limites()), /no válido/);
  }
  const microsegundos = contenido();
  microsegundos.plazos[0].abre_en = "2028-02-29T08:00:00.123456Z";
  microsegundos.plazos[0].cierra_en = "2028-02-29T09:00:00.654321Z";
  const microsegundosValidados = validarContenidoEditable(microsegundos, limites());
  assert.equal(microsegundosValidados.plazos[0].abre_en, "2028-02-29T08:00:00.123456Z");
  const intervaloDeUnMicrosegundo = contenido();
  intervaloDeUnMicrosegundo.plazos[0].abre_en = "2028-02-29T08:00:00.123455Z";
  intervaloDeUnMicrosegundo.plazos[0].cierra_en = "2028-02-29T08:00:00.123456Z";
  assert.doesNotThrow(() => validarContenidoEditable(intervaloDeUnMicrosegundo, limites()));
  [
    intervaloDeUnMicrosegundo.plazos[0].abre_en,
    intervaloDeUnMicrosegundo.plazos[0].cierra_en,
  ] = [
    intervaloDeUnMicrosegundo.plazos[0].cierra_en,
    intervaloDeUnMicrosegundo.plazos[0].abre_en,
  ];
  assert.throws(() => validarContenidoEditable(intervaloDeUnMicrosegundo, limites()), /intervalo válido/);

  const sustitutoAislado = contenido();
  sustitutoAislado.titulo = `Título\uD800`;
  assert.throws(() => validarContenidoEditable(sustitutoAislado, limites()), /no válido/);
});

test("todo contenido válido bajo límites de dominio cabe en la proyección de lista", () => {
  const contenidoLimite = contenido();
  contenidoLimite.titulo = "t".repeat(limites().maximo_titulo);
  assert.equal(validarContenidoEditable(contenidoLimite, limites()).titulo.length, 180);
  const datosLista = lista();
  datosLista.elementos[0].titulo = contenidoLimite.titulo;
  assert.equal(validarListaBorradores(datosLista).elementos[0].titulo.length, 180);
  datosLista.elementos[0].titulo += "x";
  assert.throws(() => validarListaBorradores(datosLista), /no válido/);
});

test("alta fija identidad y dependencias; actualización no puede cambiarlas", () => {
  const alta = validarSolicitudCrearBorrador(solicitudCrear(), limites());
  assert.equal(alta.identificador_publico, "bolsa-auxiliares-2026");
  assert.equal(alta.plantilla_huella_sha256, HUELLA_A);
  assert.equal(alta.motivo_huella_sha256, HUELLA_B);
  const actualizacion = validarSolicitudActualizarBorrador(solicitudActualizar(), limites());
  assert.equal(actualizacion.contenido_editable.titulo, contenido().titulo);
  for (const campoProhibido of [
    "actor", "rol", "ambito", "estado", "firma", "configuracion",
    "identificador_publico", "codigo_version_publica", "expediente_ref", "plantilla_ref",
  ]) {
    const intento = solicitudActualizar();
    intento[campoProhibido] = "declarado-por-cliente";
    assert.throws(() => validarSolicitudActualizarBorrador(intento, limites()), /contrato cerrado/);
  }
});

test("detalle separa contenido editable de ámbito, identidad y configuración de lectura", () => {
  const validado = validarDetalleBorrador(detalle(), limites());
  assert.equal(validado.identificador_publico, "bolsa-auxiliares-2026");
  assert.equal(validado.configuracion_lectura.documentos[0].firma_validada_ref, "firma:bases:1");
  assert.equal(Object.hasOwn(validado.contenido_editable, "documentos"), false);
  const documentoEditable = detalle();
  documentoEditable.contenido_editable.documentos = [];
  assert.throws(() => validarDetalleBorrador(documentoEditable, limites()), /contrato cerrado/);

  const base = configuracionLectura().documentos[0];
  const segundo = {
    ...base,
    publicacion_ref: "publicacion:bases:2",
    documento_ref: "documento:bases:2",
    version_documento: 2,
    representacion_ref: "representacion:bases:pdf:2",
    huella_contenido_sha256: HUELLA_A,
    firma_validada_ref: "firma:bases:2",
    recibo_custodia_ref: "custodia:bases:2",
  };
  const ejesDuplicados = [
    ["publicacion_ref", /elementos repetidos/],
    ["documento_version", /documento y versión/],
    ["representacion_ref", /representación/],
    ["firma_validada_ref", /firma/],
    ["recibo_custodia_ref", /custodia/],
  ];
  for (const [eje, patron] of ejesDuplicados) {
    const cruzado = detalle();
    const copiaSegundo = { ...segundo };
    if (eje === "documento_version") {
      copiaSegundo.documento_ref = base.documento_ref;
      copiaSegundo.version_documento = base.version_documento;
    } else {
      copiaSegundo[eje] = base[eje];
    }
    cruzado.configuracion_lectura.documentos.push(copiaSegundo);
    assert.throws(() => validarDetalleBorrador(cruzado, limites()), patron);
  }

  const otraVersionMismoDocumento = detalle();
  otraVersionMismoDocumento.configuracion_lectura.documentos.push({
    ...segundo, documento_ref: base.documento_ref,
  });
  assert.equal(
    validarDetalleBorrador(otraVersionMismoDocumento, limites()).configuracion_lectura.documentos.length,
    2,
  );
});

test("envelope y recibo son cerrados y conservan evidencias", () => {
  assert.deepEqual(extraerDatosEnvelopeBorradores({ data: lista() }), lista());
  assert.throws(() => extraerDatosEnvelopeBorradores(lista()), /envelope no respeta/);
  assert.throws(() => extraerDatosEnvelopeBorradores({ data: lista(), meta: {} }), /contrato cerrado/);
  const validado = validarReciboGuardadoBorrador(recibo("actualizar"));
  assert.equal(validado.referencia_estado.revision, 4);
  assert.equal(validado.auditoria_ref, "auditoria:borrador:2026:17");
  const incompleto = recibo();
  delete incompleto.evento_outbox_ref;
  assert.throws(() => validarReciboGuardadoBorrador(incompleto), /contrato cerrado/);

  const errorValidado = extraerErrorEnvelopeBorradores({
    error: { codigo: "conflicto_cas", correlacion_ref: "correlacion:borradores:001" },
  });
  assert.equal(errorValidado.codigo, "conflicto_cas");
  assert.throws(
    () => extraerErrorEnvelopeBorradores({
      error: {
        codigo: "error_interno",
        correlacion_ref: "correlacion:borradores:001",
        mensaje: "detalle arbitrario que nunca debe mostrarse",
      },
    }),
    /contrato cerrado/,
  );
});

test("ETag fuerte representa exactamente revisión y huella de estado", () => {
  const etag = etagEstado(3);
  assert.equal(etag, `"vec-borrador-v1.r3.sha256-${HUELLA_A}"`);
  assert.equal(validarDetalleBorrador(detalle(), limites()).etag, etag);

  for (const construir of [lista, detalle, () => recibo("actualizar")]) {
    const valor = construir();
    const objetivo = Object.hasOwn(valor, "elementos") ? valor.elementos[0] : valor;
    objetivo.etag = etagEstado(objetivo.referencia_estado.revision, HUELLA_B);
    assert.throws(
      () => (Object.hasOwn(valor, "elementos")
        ? validarListaBorradores(valor)
        : (valor.esquema === ESQUEMAS_BORRADORES.detalle
          ? validarDetalleBorrador(valor, limites())
          : validarReciboGuardadoBorrador(valor))),
      /ETag no corresponde a revisión y huella/,
    );
  }
  const opaco = detalle();
  opaco.etag = '"revision-3-huella-a"';
  assert.throws(() => validarDetalleBorrador(opaco, limites()), /ETag fuerte no válido/);
});

test("idempotencia usa exactamente 32 bytes CSPRNG y base64url sin padding", () => {
  let longitud = 0;
  const ceroSeguro = {
    getRandomValues(bytes) {
      longitud = bytes.length;
      bytes.fill(0);
      return bytes;
    },
  };
  const clave = generarClaveIdempotencia(ceroSeguro);
  assert.equal(longitud, 32);
  assert.equal(clave, CLAVE_IDEMPOTENCIA_A, "solo se valida la forma; el cliente no infiere entropía");
  assert.equal(validarClaveIdempotencia(clave), clave);
  assert.throws(() => validarClaveIdempotencia(`${clave}=`), /no válida/);
  assert.throws(() => validarClaveIdempotencia("a".repeat(42)), /no válida/);
  assert.throws(
    () => validarClaveIdempotencia(`${"A".repeat(42)}B`),
    /no válida/,
    "los bits de relleno no nulos no forman Base64URL canónico",
  );
  assert.equal(validarClaveIdempotencia(CLAVE_IDEMPOTENCIA_B), CLAVE_IDEMPOTENCIA_B);
  assert.throws(() => generarClaveIdempotencia({}), /generador criptográfico/);
});
