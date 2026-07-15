package almacen_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/almacen"
	"vec-diputacion-granada/internal/vec/adapters/memory"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
	pruebasvec "vec-diputacion-granada/internal/vec/pruebas"
)

type relojFijoAlmacen struct{ ahora time.Time }

func (r relojFijoAlmacen) Ahora() time.Time { return r.ahora }

// requisitosPrueba mantiene un perfil positivo y explicito compartido por
// las pruebas del registro. No interpreta capacidades omitidas como
// concedidas ni finge las garantias criptograficas de un conector productivo.
func requisitosPrueba() ports.RequisitosAlmacenObjetos {
	return ports.RequisitosAlmacenObjetos{
		EscrituraEnFlujo:       true,
		LecturaEnFlujo:         true,
		ReferenciasOpacas:      true,
		IntegridadSHA256:       true,
		Versionado:             true,
		Retencion:              true,
		BloqueoLegal:           true,
		PromocionAtomica:       true,
		PreservaObjetoOriginal: true,
		TamanoMinimoObjeto:     1024,
	}
}

func contextoContenidoPrueba(
	t *testing.T,
	sufijo string,
	accion string,
	objeto ports.ReferenciaObjetoAlmacen,
	instante time.Time,
) ports.ContextoOperacionAlmacen {
	t.Helper()
	accionNegocio := ports.AccionNegocioCustodiarDecisionBaremacion
	campos := []string{"documento_custodiado", "evidencia_custodia"}
	requiereObjeto := false
	constructor := func(
		decision domain.DecisionAutorizacion,
		recurso domain.RecursoAutorizable,
		vinculos ports.VinculosOperacionAlmacen,
	) (ports.ContextoOperacionAlmacen, error) {
		return ports.NuevoContextoCustodiarDecisionBaremacionAlmacen(decision, recurso, vinculos, instante)
	}
	if accion == ports.AccionAlmacenLeer {
		accionNegocio = ports.AccionNegocioAnalizarCargaDocumental
		campos = []string{"analisis_seguridad", "estado"}
		requiereObjeto = true
		constructor = func(
			decision domain.DecisionAutorizacion,
			recurso domain.RecursoAutorizable,
			vinculos ports.VinculosOperacionAlmacen,
		) (ports.ContextoOperacionAlmacen, error) {
			return ports.NuevoContextoAnalizarCargaDocumentalAlmacen(decision, recurso, vinculos, instante)
		}
	} else if accion != ports.AccionAlmacenEscribir {
		t.Fatalf("accion sin fabrica positiva: %s", accion)
	}
	if requiereObjeto && objeto.Validar() != nil {
		t.Fatal("la lectura exige referencia y version exactas")
	}
	vinculos := ports.VinculosOperacionAlmacen{
		OperacionRef: "operacion:contenido:" + sufijo, CargaRef: "carga:contenido:" + sufijo,
		Clasificacion:       "datos_personales_alta",
		SujetoSeudonimoHMAC: "hmac-sha256:sujeto_v1:" + strings.Repeat("a", 64),
		HuellaSolicitudHMAC: "hmac-sha256:solicitud_v1:" + strings.Repeat("b", 64),
		EfectoRef:           "efecto:contenido:" + sufijo,
		ObjetoVinculado:     objeto,
	}
	atributos := map[string]string{
		ports.AtributoAlmacenOperacionRef:        vinculos.OperacionRef,
		ports.AtributoAlmacenCargaRef:            vinculos.CargaRef,
		ports.AtributoAlmacenClasificacion:       vinculos.Clasificacion,
		ports.AtributoAlmacenSujetoSeudonimoHMAC: vinculos.SujetoSeudonimoHMAC,
		ports.AtributoAlmacenHuellaSolicitudHMAC: vinculos.HuellaSolicitudHMAC,
		ports.AtributoAlmacenEfectoRef:           vinculos.EfectoRef,
	}
	if requiereObjeto {
		atributos[ports.AtributoAlmacenObjetoRef] = objeto.Referencia
		atributos[ports.AtributoAlmacenObjetoVersion] = objeto.Version
	}
	recurso := domain.RecursoAutorizable{
		Referencia: "documento:" + sufijo, ModuloID: "bolsa", Tipo: "documento_bolsa",
		Ambitos: map[string]string{"organizacion": "diputacion_granada"}, Atributos: atributos,
	}
	huellaRecurso, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	_, vinculoActor, err := pruebasvec.NuevoContextoYVinculo(
		instante, "per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl",
		domain.AuthMethodCertificate, domain.AuthAssuranceHigh,
	)
	if err != nil {
		t.Fatal(err)
	}
	huellaCatalogo, err := domain.HuellaEvidenciasCatalogoPoliticasAutorizacion(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	decision := domain.DecisionAutorizacion{
		DecisionRef: "decision:contenido:" + sufijo, Concedida: true, Codigo: "concedida",
		PrincipalID: "per_0123456789abcdefghijkl", PerfilActivoRef: "prf_0123456789abcdefghijkl",
		Accion: accionNegocio, RecursoRef: recurso.Referencia, ModuloID: recurso.ModuloID,
		TipoRecurso: recurso.Tipo, ContextoRecursoHuellaSHA256: huellaRecurso,
		Finalidad: "custodia_documental", CorrelacionRef: "correlacion:contenido:" + sufijo,
		VinculoAutenticacionActor: vinculoActor,
		AsignacionRef:             "asignacion:contenido:v1", AsignacionHuellaSHA256: strings.Repeat("1", 64),
		VersionRolRef: "rol:contenido:v1", VersionRolHuellaSHA256: strings.Repeat("2", 64),
		ControlVigenciaVersionRolRef: "rol:contenido:v1", ControlVigenciaVersionRolRevision: 1,
		ControlVigenciaVersionRolHuellaSHA256: strings.Repeat("3", 64),
		RevisionCatalogoPoliticas:             1, CatalogoPoliticasHuellaSHA256: huellaCatalogo,
		PoliticasEvaluadasHuellasSHA256: map[string]string{}, GarantiaMinima: domain.AuthAssuranceHigh,
		CamposPermitidos: campos, EmitidaEn: instante.Add(-time.Minute), ValidaHasta: instante.Add(4 * time.Minute),
	}
	contexto, err := constructor(decision, recurso, vinculos)
	if err != nil {
		t.Fatalf("crear capacidad documental: %v", err)
	}
	return contexto
}

func solicitudGeneracionContenidoPrueba(
	t *testing.T,
	sufijo, referenciaLogica, claveIdempotencia string,
	formato domain.FormatoDocumento,
	zona ports.ZonaAlmacen,
	contenido []byte,
	instante time.Time,
) ports.SolicitudGuardarContenido {
	t.Helper()
	suma := sha256.Sum256(contenido)
	declaracion := ports.DeclaracionRepresentacionGeneracionDocumental{
		ReferenciaLogica: referenciaLogica, ClaveIdempotencia: claveIdempotencia,
		Formato: formato, Zona: zona, MIME: formato.MIME(), Tamano: int64(len(contenido)),
		HuellaSHA256: hex.EncodeToString(suma[:]),
	}
	manifiesto, contexto := contextoGeneracionContenidoPrueba(
		t, sufijo, []ports.DeclaracionRepresentacionGeneracionDocumental{declaracion}, instante,
	)
	proyeccion, err := manifiesto.Proyeccion()
	if err != nil || len(proyeccion.Pasos) != 1 {
		t.Fatalf("proyectar manifiesto simple: %#v, %v", proyeccion, err)
	}
	return ports.SolicitudGuardarContenido{
		Contexto: contexto, ClaveIdempotencia: declaracion.ClaveIdempotencia,
		DocumentoID: declaracion.ReferenciaLogica, Zona: declaracion.Zona, MIME: declaracion.MIME,
		HuellaSHA256: declaracion.HuellaSHA256, Tamano: declaracion.Tamano,
		Contenido: append([]byte(nil), contenido...),
	}
}

func contextoGeneracionContenidoPrueba(
	t *testing.T,
	sufijo string,
	declaraciones []ports.DeclaracionRepresentacionGeneracionDocumental,
	instante time.Time,
) (ports.ManifiestoGeneracionDocumental, ports.ContextoOperacionAlmacen) {
	t.Helper()
	formatos := make([]domain.FormatoDocumento, 0, len(declaraciones))
	vistos := map[domain.FormatoDocumento]bool{}
	for _, declaracion := range declaraciones {
		if !vistos[declaracion.Formato] {
			vistos[declaracion.Formato] = true
			formatos = append(formatos, declaracion.Formato)
		}
	}
	plantilla := domain.PlantillaDocumento{
		ID: "plantilla-contenido-" + sufijo, Version: 1, ModuloID: "bolsa", TipoDocumental: "documento_prueba",
		Nombre: "Plantilla de contenido", Titulo: "Documento {{numero}}",
		Parrafos: []string{"Contenido administrativo."},
		Campos:   []domain.CampoPlantillaDocumento{{Clave: "numero", Etiqueta: "Numero", Obligatorio: true}},
		Formatos: formatos, PermisoGenerar: "bolsa.documentos.generar",
		GarantiaMinima: domain.AuthAssuranceSubstantial, Estado: domain.EstadoPlantillaPublicada,
		CreadaPor: "tecnico:rrhh:contenido", CreadaEn: instante.Add(-2 * time.Hour),
		PublicadaPor: "jefatura:rrhh:contenido", PublicadaEn: instante.Add(-time.Hour),
		AprobacionRef: "aprobacion:contenido:" + sufijo, MotivoPublicacion: "prueba_contrato",
	}
	manifiesto, err := ports.NuevoManifiestoGeneracionDocumental(plantilla, declaraciones)
	if err != nil {
		t.Fatalf("crear manifiesto de contenido: %v", err)
	}
	vinculos := ports.VinculosOperacionAlmacen{
		OperacionRef: "operacion:generacion:" + sufijo, CargaRef: "generacion:contenido:" + sufijo,
		Clasificacion:       "datos_personales_alta",
		SujetoSeudonimoHMAC: "hmac-sha256:sujeto_documental_v1:" + strings.Repeat("a", 64),
		HuellaSolicitudHMAC: "hmac-sha256:solicitud_documental_v1:" + strings.Repeat("b", 64),
		EfectoRef:           "efecto:generacion:" + sufijo,
	}
	base := domain.RecursoAutorizable{
		Referencia: "recurso:generacion:" + sufijo, ModuloID: plantilla.ModuloID, Tipo: "documento_bolsa",
		Ambitos:   map[string]string{"organizacion": "diputacion_granada"},
		Atributos: map[string]string{"expediente_ref": "expediente:bolsa:" + sufijo},
	}
	recurso, err := ports.VincularRecursoGeneracionDocumental(base, manifiesto, vinculos)
	if err != nil {
		t.Fatalf("vincular recurso documental: %v", err)
	}
	_, vinculoActor, err := pruebasvec.NuevoContextoYVinculo(
		instante, "per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl",
		domain.AuthMethodCertificate, domain.AuthAssuranceHigh,
	)
	if err != nil {
		t.Fatal(err)
	}
	huellaCatalogo, err := domain.HuellaEvidenciasCatalogoPoliticasAutorizacion(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	huellaRecurso, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	decision := domain.DecisionAutorizacion{
		DecisionRef: "decision:generacion:" + sufijo, Concedida: true, Codigo: "concedida",
		PrincipalID: "per_0123456789abcdefghijkl", PerfilActivoRef: "prf_0123456789abcdefghijkl",
		Accion: plantilla.PermisoGenerar, RecursoRef: recurso.Referencia, ModuloID: recurso.ModuloID,
		TipoRecurso: recurso.Tipo, ContextoRecursoHuellaSHA256: huellaRecurso,
		Finalidad: "generar_documento", CorrelacionRef: "correlacion:generacion:" + sufijo,
		VinculoAutenticacionActor: vinculoActor,
		AsignacionRef:             "asignacion:generacion:v1", AsignacionHuellaSHA256: strings.Repeat("1", 64),
		VersionRolRef: "rol:generacion:v1", VersionRolHuellaSHA256: strings.Repeat("2", 64),
		ControlVigenciaVersionRolRef: "rol:generacion:v1", ControlVigenciaVersionRolRevision: 1,
		ControlVigenciaVersionRolHuellaSHA256: strings.Repeat("3", 64),
		RevisionCatalogoPoliticas:             1, CatalogoPoliticasHuellaSHA256: huellaCatalogo,
		PoliticasEvaluadasHuellasSHA256: map[string]string{}, GarantiaMinima: domain.AuthAssuranceSubstantial,
		EmitidaEn: instante.Add(-time.Minute), ValidaHasta: instante.Add(4 * time.Minute),
	}
	contexto, err := ports.NuevoContextoGeneracionDocumentalAlmacen(
		decision, recurso, manifiesto, vinculos, instante,
	)
	if err != nil {
		t.Fatalf("crear contexto de generacion: %v", err)
	}
	return manifiesto, contexto
}

func TestContenidoDocumentalFuncionaConConectoresIntercambiablesYCapacidadExacta(t *testing.T) {
	instante := time.Now().UTC().Truncate(time.Microsecond)
	for _, conectorID := range []string{"objetos-local", "objetos-s3-compatible"} {
		t.Run(conectorID, func(t *testing.T) {
			objetos, err := memory.NuevoAlmacenObjetosMemoria(
				conectorID, 2<<20, relojFijoAlmacen{ahora: instante},
			)
			if err != nil {
				t.Fatal(err)
			}
			adaptador, err := almacen.NuevoContenidoDocumental(
				context.Background(), objetos,
				ports.RequisitosAlmacenObjetos{EscrituraEnFlujo: true, LecturaEnFlujo: true, ReferenciasOpacas: true},
			)
			if err != nil {
				t.Fatal(err)
			}
			contenido := []byte("%PDF-1.7\ncontenido protegido\n%%EOF")
			solicitud := solicitudGeneracionContenidoPrueba(
				t, "escritura-"+conectorID, "representacion-"+conectorID,
				"idempotencia-"+conectorID, domain.FormatoDocumentoPDF,
				ports.ZonaAlmacenCuarentena, contenido, instante,
			)
			guardado, err := adaptador.GuardarContenido(context.Background(), solicitud)
			if err != nil || guardado.ConectorID != conectorID ||
				guardado.ValidarContra(solicitud) != nil ||
				guardado.Referencia != guardado.EvidenciaOperacion.Objeto.Referencia ||
				guardado.Version != guardado.EvidenciaOperacion.Objeto.Version ||
				guardado.EvidenciaOperacion.HuellaManifiestoSHA256 == "" ||
				guardado.EvidenciaOperacion.HuellaPasoSHA256 == "" {
				t.Fatalf("guardar: %+v, %v", guardado, err)
			}
			repetido, err := adaptador.GuardarContenido(context.Background(), solicitud)
			if err != nil || repetido.Referencia != guardado.Referencia ||
				!repetido.EvidenciaOperacion.ReintentoIdempotente || repetido.ValidarContra(solicitud) != nil {
				t.Fatalf("reintento exacto: %+v, %v", repetido, err)
			}
			contextoLectura := contextoContenidoPrueba(
				t, "lectura-"+conectorID, ports.AccionAlmacenLeer,
				guardado.EvidenciaOperacion.Objeto, instante,
			)
			leido, err := adaptador.LeerContenido(context.Background(), ports.SolicitudLeerContenido{
				Contexto: contextoLectura, Referencia: guardado.Referencia,
				Zona: ports.ZonaAlmacenCuarentena, Limite: int64(len(contenido)),
			})
			if err != nil || string(leido.Contenido) != string(contenido) || leido.HuellaSHA256 != guardado.HuellaSHA256 {
				t.Fatalf("leer: %+v, %v", leido, err)
			}
		})
	}
}

func TestContenidoDocumentalDeniegaValorCeroAccionCruzadaYReferenciaAlterada(t *testing.T) {
	instante := time.Now().UTC().Truncate(time.Microsecond)
	objetos, err := memory.NuevoAlmacenObjetosMemoria("objetos-prueba", 1024, relojFijoAlmacen{ahora: instante})
	if err != nil {
		t.Fatal(err)
	}
	adaptador, err := almacen.NuevoContenidoDocumental(context.Background(), objetos, ports.RequisitosAlmacenObjetos{})
	if err != nil {
		t.Fatal(err)
	}
	contenido := []byte("contenido")
	suma := sha256.Sum256(contenido)
	base := ports.SolicitudGuardarContenido{
		Contexto: ports.ContextoOperacionAlmacen{}, ClaveIdempotencia: "idempotencia-denegada",
		DocumentoID: "documento-denegado", Zona: ports.ZonaAlmacenCuarentena,
		MIME: "application/pdf", HuellaSHA256: hex.EncodeToString(suma[:]),
		Tamano: int64(len(contenido)), Contenido: contenido,
	}
	if _, err := adaptador.GuardarContenido(context.Background(), base); !errors.Is(err, ports.ErrSolicitudAlmacenInvalida) {
		t.Fatalf("valor cero aceptado: %v", err)
	}
	objetoFicticio := ports.ReferenciaObjetoAlmacen{Referencia: "objeto:ficticio", Version: "1"}
	base.Contexto = contextoContenidoPrueba(t, "accion-cruzada", ports.AccionAlmacenLeer, objetoFicticio, instante)
	if _, err := adaptador.GuardarContenido(context.Background(), base); !errors.Is(err, ports.ErrSolicitudAlmacenInvalida) {
		t.Fatalf("accion cruzada aceptada: %v", err)
	}
	contextoLectura := contextoContenidoPrueba(t, "referencia-alterada", ports.AccionAlmacenLeer, objetoFicticio, instante)
	if _, err := adaptador.LeerContenido(context.Background(), ports.SolicitudLeerContenido{
		Contexto: contextoLectura, Referencia: "almacen:v1:valor-no-canonico",
		Zona: ports.ZonaAlmacenCuarentena, Limite: 32,
	}); !errors.Is(err, ports.ErrSolicitudAlmacenInvalida) {
		t.Fatalf("referencia alterada aceptada: %v", err)
	}
}

func TestContenidoDocumentalEjecutaManifiestoMultipasoSinCruzarRepresentaciones(t *testing.T) {
	instante := time.Now().UTC().Truncate(time.Microsecond)
	docx := []byte("PK contenido docx compuesto")
	pdf := []byte("%PDF-1.7 contenido pdf compuesto")
	huellaDOCX := sha256.Sum256(docx)
	huellaPDF := sha256.Sum256(pdf)
	declaraciones := []ports.DeclaracionRepresentacionGeneracionDocumental{
		{
			ReferenciaLogica: "representacion:b:pdf", ClaveIdempotencia: "idempotencia:b:pdf",
			Formato: domain.FormatoDocumentoPDF, Zona: ports.ZonaAlmacenAdmitida,
			MIME: domain.FormatoDocumentoPDF.MIME(), Tamano: int64(len(pdf)),
			HuellaSHA256: hex.EncodeToString(huellaPDF[:]),
		},
		{
			ReferenciaLogica: "representacion:a:docx", ClaveIdempotencia: "idempotencia:a:docx",
			Formato: domain.FormatoDocumentoDOCX, Zona: ports.ZonaAlmacenAdmitida,
			MIME: domain.FormatoDocumentoDOCX.MIME(), Tamano: int64(len(docx)),
			HuellaSHA256: hex.EncodeToString(huellaDOCX[:]),
		},
	}
	manifiesto, contexto := contextoGeneracionContenidoPrueba(t, "multipaso", declaraciones, instante)
	proyeccion, err := manifiesto.Proyeccion()
	if err != nil || len(proyeccion.Pasos) != 2 {
		t.Fatalf("proyectar manifiesto compuesto: %#v, %v", proyeccion, err)
	}
	objetos, err := memory.NuevoAlmacenObjetosMemoria(
		"objetos-multipaso", 1<<20, relojFijoAlmacen{ahora: instante},
	)
	if err != nil {
		t.Fatal(err)
	}
	adaptador, err := almacen.NuevoContenidoDocumental(
		context.Background(), objetos,
		ports.RequisitosAlmacenObjetos{EscrituraEnFlujo: true, ReferenciasOpacas: true},
	)
	if err != nil {
		t.Fatal(err)
	}
	guardados := make([]ports.ContenidoDocumentoGuardado, 0, len(proyeccion.Pasos))
	for indice, paso := range proyeccion.Pasos {
		contextoPaso := contexto
		if indice > 0 {
			contextoPaso, err = contexto.DerivarPaso(paso.PasoRef)
			if err != nil {
				t.Fatalf("derivar paso %d: %v", indice, err)
			}
		}
		contenido := docx
		if paso.Formato == domain.FormatoDocumentoPDF {
			contenido = pdf
		}
		solicitud := ports.SolicitudGuardarContenido{
			Contexto: contextoPaso, ClaveIdempotencia: paso.ClaveIdempotencia,
			DocumentoID: paso.ReferenciaLogica, Zona: paso.Zona, MIME: paso.MIME,
			HuellaSHA256: paso.HuellaSHA256, Tamano: paso.Tamano,
			Contenido: append([]byte(nil), contenido...),
		}
		guardado, err := adaptador.GuardarContenido(context.Background(), solicitud)
		if err != nil || guardado.ValidarContra(solicitud) != nil ||
			guardado.EvidenciaOperacion.HuellaManifiestoSHA256 != proyeccion.HuellaManifiestoSHA256 ||
			guardado.EvidenciaOperacion.HuellaPasoSHA256 != paso.HuellaPasoSHA256 {
			t.Fatalf("guardar paso %d: %+v, %v", indice, guardado, err)
		}
		guardados = append(guardados, guardado)
	}
	if guardados[0].Referencia == guardados[1].Referencia ||
		guardados[0].EvidenciaOperacion.HuellaPasoSHA256 == guardados[1].EvidenciaOperacion.HuellaPasoSHA256 {
		t.Fatal("dos pasos distintos compartieron identidad fisica o huella de paso")
	}

	contextoSegundo, err := contexto.DerivarPaso(proyeccion.Pasos[1].PasoRef)
	if err != nil {
		t.Fatal(err)
	}
	cruzada := ports.SolicitudGuardarContenido{
		Contexto: contextoSegundo, ClaveIdempotencia: proyeccion.Pasos[0].ClaveIdempotencia,
		DocumentoID: proyeccion.Pasos[0].ReferenciaLogica, Zona: proyeccion.Pasos[0].Zona,
		MIME: proyeccion.Pasos[0].MIME, HuellaSHA256: proyeccion.Pasos[0].HuellaSHA256,
		Tamano: proyeccion.Pasos[0].Tamano, Contenido: append([]byte(nil), docx...),
	}
	if _, err := adaptador.GuardarContenido(context.Background(), cruzada); !errors.Is(err, ports.ErrSolicitudAlmacenInvalida) {
		t.Fatalf("se cruzaron metadatos entre pasos: %v", err)
	}
}

func TestContenidoDocumentalRevalidaCaducidadAntesDelEfecto(t *testing.T) {
	instanteDecision := time.Now().UTC().Add(-10 * time.Minute).Truncate(time.Microsecond)
	contenido := []byte("%PDF-1.7 decision caducada")
	solicitud := solicitudGeneracionContenidoPrueba(
		t, "caducada", "representacion:caducada", "idempotencia:caducada",
		domain.FormatoDocumentoPDF, ports.ZonaAlmacenAdmitida, contenido, instanteDecision,
	)
	objetos, err := memory.NuevoAlmacenObjetosMemoria(
		"objetos-caducidad", 1<<20, relojFijoAlmacen{ahora: time.Now().UTC()},
	)
	if err != nil {
		t.Fatal(err)
	}
	adaptador, err := almacen.NuevoContenidoDocumental(
		context.Background(), objetos, ports.RequisitosAlmacenObjetos{EscrituraEnFlujo: true},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := adaptador.GuardarContenido(context.Background(), solicitud); !errors.Is(err, ports.ErrSolicitudAlmacenInvalida) {
		t.Fatalf("decision caducada alcanzo el conector: %v", err)
	}
}

func TestContenidoDocumentalRechazaContextoYConectorTipadoNulos(t *testing.T) {
	var objetosNulos *memory.AlmacenObjetosMemoria
	if _, err := almacen.NuevoContenidoDocumental(
		context.Background(), objetosNulos, ports.RequisitosAlmacenObjetos{},
	); !errors.Is(err, ports.ErrSolicitudAlmacenInvalida) {
		t.Fatalf("conector tipado nulo aceptado: %v", err)
	}
	objetos, err := memory.NuevoAlmacenObjetosMemoria(
		"objetos-contexto", 1<<20, relojFijoAlmacen{ahora: time.Now().UTC()},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := almacen.NuevoContenidoDocumental(
		nil, objetos, ports.RequisitosAlmacenObjetos{},
	); !errors.Is(err, ports.ErrSolicitudAlmacenInvalida) {
		t.Fatalf("contexto nulo aceptado al componer: %v", err)
	}
}
