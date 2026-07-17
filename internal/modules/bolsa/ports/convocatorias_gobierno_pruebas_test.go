package ports

import (
	"context"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
	pruebasvec "vec-diputacion-granada/internal/vec/pruebas"
)

func versionGobernadaPuertosPrueba(t *testing.T) dominiobolsa.VersionConvocatoriaGobernada {
	t.Helper()
	ambito, err := dominiobolsa.NuevoAmbitoOrganizativoConvocatoria(
		"org_diputaciongranada", "uni_seleccionexterna",
	)
	if err != nil {
		t.Fatalf("crear ambito organizativo de prueba: %v", err)
	}
	contenido := dominiobolsa.ContenidoPublicableConvocatoria{
		IdentificadorPublico: "auxiliar-administrativo-2026", Tipo: "bolsa_temporal",
		CatalogoCategorias: dominiobolsa.ReferenciaCatalogoCategorias{
			CatalogoID: "categorias-profesionales", CatalogoVersion: 1,
			CatalogoHuellaSHA256: huellaConvocatoriaPrueba('a'),
		},
		Categorias: []string{"auxiliar_administrativo"}, Titulo: "Bolsa temporal de auxiliares",
		Resumen:     "Convocatoria publica para la bolsa temporal.",
		Descripcion: "Proceso selectivo sujeto a bases firmadas y publicadas.",
		Plazos: []dominiobolsa.PlazoConvocatoria{{
			Referencia: "plazo:inscripcion", Tipo: "inscripcion", Titulo: "Inscripcion",
			Descripcion: "Plazo de presentacion.", AbreEn: instanteGobiernoConvocatoriaPrueba.Add(24 * time.Hour),
			CierraEn: instanteGobiernoConvocatoriaPrueba.Add(30 * 24 * time.Hour),
		}},
		Requisitos: []dominiobolsa.RequisitoConvocatoria{{
			Referencia: "requisito:edad", Orden: 1, Titulo: "Edad",
			Descripcion: "Cumplir la edad exigida.", Obligatorio: true,
		}},
		Documentos: []dominiobolsa.DocumentoPublicableConvocatoria{{
			Referencia: "documento:bases", Tipo: "bases", Orden: 1, Titulo: "Bases",
			Descripcion: "Bases de la convocatoria.", Formato: "pdf", URL: "/bolsa/documentos/bases.pdf",
		}},
		Ayuda: []dominiobolsa.AyudaConvocatoria{{
			Referencia: "ayuda:inscripcion", Categoria: "inscripcion", Orden: 1,
			Pregunta: "¿Como presentar la solicitud?", Respuesta: "Acceda a su area personal.",
		}},
	}
	referencia := func(id string, version int, marca byte) dominiobolsa.ReferenciaConfiguracionConvocatoria {
		return dominiobolsa.ReferenciaConfiguracionConvocatoria{
			ID: id, Version: version, HuellaContenidoSHA256: huellaConvocatoriaPrueba(marca),
		}
	}
	configuracion := dominiobolsa.ConfiguracionFijadaConvocatoria{
		Catalogos:        referencia("catalogos:bolsa", 3, '1'),
		Calendario:       referencia("calendario:auxiliar", 2, '2'),
		ReglasBaremacion: referencia("baremo:auxiliar", 5, '3'),
		FlujoProceso:     referencia("convocatoria-bolsa", 4, '4'),
		FlujoSolicitud:   referencia("solicitud-bolsa", 7, '5'),
		Documentos: []dominiobolsa.ReferenciaDocumentoOficialConvocatoria{{
			Rol: "bases", PublicacionRef: "documento:bases", DocumentoRef: "documento:logico:bases:001",
			VersionDocumento: 2, RepresentacionRef: "representacion:pdf:bases:002",
			HuellaContenidoSHA256: huellaConvocatoriaPrueba('6'),
			FirmaValidadaRef:      "firma:validada:bases:002", ReciboCustodiaRef: "custodia:bases:002",
		}},
	}
	version, err := dominiobolsa.NuevaVersionConvocatoriaGobernada(
		dominiobolsa.DatosNuevaVersionConvocatoriaGobernada{
			ID: "proceso:bolsa:auxiliar-2026", CodigoVersionPublica: "v1",
			InstanciaFlujoRef:  "instancia:flujo:convocatoria:001",
			AmbitoOrganizativo: ambito,
			Contenido:          contenido, Configuracion: configuracion,
			ExpedienteRef: "expediente:seleccion:2026-001", Motivo: "Preparacion administrativa.",
			ActorID: "persona:tecnica:001", Instante: instanteGobiernoConvocatoriaPrueba.Add(-time.Hour),
		},
	)
	if err != nil {
		t.Fatalf("crear version de prueba: %v", err)
	}
	return version
}

func actualizarYRestaurarContenidoConvocatoriaPrueba(
	t *testing.T,
	original dominiobolsa.VersionConvocatoriaGobernada,
) dominiobolsa.VersionConvocatoriaGobernada {
	t.Helper()
	configuracionDistinta := original.Configuracion
	configuracionDistinta.ReglasBaremacion.Version++
	configuracionDistinta.ReglasBaremacion.HuellaContenidoSHA256 = huellaConvocatoriaPrueba('c')
	segunda, err := original.ActualizarBorrador(
		original.Revision, original.Contenido, configuracionDistinta,
		"persona:tecnica:002", "Cambio temporal de baremo.", original.CreadaEn.Add(time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	tercera, err := segunda.ActualizarBorrador(
		segunda.Revision, original.Contenido, original.Configuracion,
		"persona:tecnica:003", "Restauracion del baremo inicial.", original.CreadaEn.Add(2*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	return tercera
}

func publicarInicialConvocatoriaPuertosPrueba(
	t *testing.T,
	borrador dominiobolsa.VersionConvocatoriaGobernada,
	publicadaEn time.Time,
) dominiobolsa.VersionConvocatoriaGobernada {
	t.Helper()
	huellaContenido, err := borrador.HuellaContenidoSHA256()
	if err != nil {
		t.Fatal(err)
	}
	huellaEstado, err := borrador.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	aprobacion := dominiobolsa.EvidenciaAprobacionConvocatoria{
		Accion: "publicar", Referencia: "aprobacion:publicacion:001",
		HuellaEvidenciaSHA256: huellaConvocatoriaPrueba('a'), ConvocatoriaRef: borrador.Referencia(),
		Revision: borrador.Revision, HuellaContenidoSHA256: huellaContenido, HuellaEstadoSHA256: huellaEstado,
		AprobadaPor: "persona:supervisora:001", AprobadaEn: publicadaEn.Add(-2 * time.Minute),
	}
	dependencias := dominiobolsa.EvidenciaDependenciasConvocatoria{
		Referencia: "dependencias:publicacion:001", HuellaEvidenciaSHA256: huellaConvocatoriaPrueba('b'),
		ConvocatoriaRef: borrador.Referencia(), Revision: borrador.Revision,
		HuellaContenidoSHA256: huellaContenido, HuellaEstadoSHA256: huellaEstado,
		VerificadaEn: publicadaEn.Add(-time.Minute),
	}
	publicada, err := borrador.PublicarInicial(
		"persona:gestora:001", aprobacion, dependencias, "Publicacion inicial.", publicadaEn,
	)
	if err != nil {
		t.Fatalf("publicar version inicial: %v", err)
	}
	return publicada
}

func instanciaFlujoConvocatoriaPuertosPrueba(
	version dominiobolsa.VersionConvocatoriaGobernada,
) dominiovec.InstanciaFlujo {
	creadaEn := version.CreadaEn
	if !version.PublicadaEn.IsZero() {
		creadaEn = version.PublicadaEn
	}
	return dominiovec.InstanciaFlujo{
		ID: version.InstanciaFlujoRef, TipoEntidad: dominiobolsa.TipoEntidadFlujoConvocatoriaBolsa,
		EntidadRef: version.ID, DefinicionRef: version.Configuracion.FlujoProceso.ReferenciaVersionada(),
		DefinicionContenidoHuellaSHA256: version.Configuracion.FlujoProceso.HuellaContenidoSHA256,
		EstadoActual:                    "inscripcion", Revision: 1, CreadaPor: "sistema:bolsa", CreadaEn: creadaEn,
	}
}

func autorizacionMutacionConvocatoriaPrueba(
	t *testing.T,
	material MaterialIntencionGobiernoConvocatoria,
	version dominiobolsa.VersionConvocatoriaGobernada,
) puertosvec.EvidenciaUsoDecisionAutorizacion {
	t.Helper()
	recurso, err := RecursoAutorizableMutacionConvocatoria(material, version)
	if err != nil {
		t.Fatal(err)
	}
	return evidenciaAutorizacionConvocatoriaPrueba(t, material.Accion, recurso, instanteGobiernoConvocatoriaPrueba)
}

func evidenciaAutorizacionConvocatoriaPrueba(
	t *testing.T,
	accion string,
	recurso dominiovec.RecursoAutorizable,
	instante time.Time,
) puertosvec.EvidenciaUsoDecisionAutorizacion {
	t.Helper()
	vinculo, err := pruebasvec.NuevoVinculoGenerico(instante)
	if err != nil {
		t.Fatal(err)
	}
	return evidenciaAutorizacionConvocatoriaConVinculoPrueba(t, accion, recurso, instante, vinculo)
}

func evidenciaAutorizacionConvocatoriaConVinculoPrueba(
	t *testing.T,
	accion string,
	recurso dominiovec.RecursoAutorizable,
	instante time.Time,
	vinculo dominiovec.VinculoAutenticacionActorV1,
) puertosvec.EvidenciaUsoDecisionAutorizacion {
	return evidenciaAutorizacionConvocatoriaConVinculoYCamposPrueba(
		t, accion, recurso, instante, vinculo, nil,
	)
}

func evidenciaAutorizacionConvocatoriaConVinculoYCamposPrueba(
	t *testing.T,
	accion string,
	recurso dominiovec.RecursoAutorizable,
	instante time.Time,
	vinculo dominiovec.VinculoAutenticacionActorV1,
	campos []string,
) puertosvec.EvidenciaUsoDecisionAutorizacion {
	t.Helper()
	especificacion, existe := especificacionesAutorizacionConvocatoria[accion]
	if !existe {
		t.Fatalf("accion de autorizacion desconocida: %s", accion)
	}
	datosVinculo, err := vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	huellaContexto, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	huellaCatalogo, err := dominiovec.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		t.Fatal(err)
	}
	camposDecision := append([]string(nil), especificacion.campos...)
	if campos != nil {
		camposDecision = append([]string(nil), campos...)
	}
	decision := dominiovec.DecisionAutorizacion{
		DecisionRef: "autorizacion:" + stringsSeguroAccionConvocatoria(accion), Concedida: true,
		Codigo: "concedida", PrincipalID: datosVinculo.PrincipalID, PerfilActivoRef: datosVinculo.PerfilActivoRef,
		Accion: accion, RecursoRef: recurso.Referencia, Finalidad: especificacion.finalidad,
		ModuloID: ModuloGobiernoConvocatorias, TipoRecurso: TipoRecursoVersionConvocatoriaGobernada,
		ContextoRecursoHuellaSHA256: huellaContexto, CorrelacionRef: "correlacion:convocatoria:001",
		AsignacionRef: "asignacion:rrhh:v1", AsignacionHuellaSHA256: huellaConvocatoriaPrueba('7'),
		VersionRolRef: "rol:rrhh:v1", VersionRolHuellaSHA256: huellaConvocatoriaPrueba('8'),
		GarantiaMinima: dominiovec.AuthAssuranceHigh, VinculoAutenticacionActor: vinculo,
		ControlVigenciaVersionRolRef: "rol:rrhh:v1", ControlVigenciaVersionRolRevision: 1,
		ControlVigenciaVersionRolHuellaSHA256: huellaConvocatoriaPrueba('9'),
		RevisionCatalogoPoliticas:             1, CatalogoPoliticasHuellaSHA256: huellaCatalogo,
		PoliticasEvaluadasHuellasSHA256: map[string]string{},
		CamposPermitidos:                camposDecision,
		EmitidaEn:                       instante.Add(-time.Minute), ValidaHasta: instante.Add(4 * time.Minute),
	}
	evidencia, err := puertosvec.NuevaEvidenciaUsoDecisionAutorizacion(decision, instante)
	if err != nil {
		t.Fatalf("crear evidencia de autorizacion: %v", err)
	}
	return evidencia
}

type revalidadorSuperficieConvocatoriaPrueba struct {
	resultado dominiovec.AutenticacionRevalidadaV1
}

func (r revalidadorSuperficieConvocatoriaPrueba) RevalidarAutenticacionActorV1(
	context.Context,
	dominiovec.SolicitudRevalidacionAutenticacionActorV1,
) (dominiovec.AutenticacionRevalidadaV1, error) {
	return r.resultado, nil
}

func vinculoExternoPersonalConvocatoriaPrueba(
	t *testing.T,
	instante time.Time,
) dominiovec.VinculoAutenticacionActorV1 {
	t.Helper()
	cuenta := dominiovec.CuentaAutenticadaContextoActor{
		CuentaRef: "cta_0123456789abcdefghijkl", Metodo: dominiovec.AuthMethodCertificate,
		Garantia: dominiovec.AuthAssuranceHigh,
	}
	instantanea := dominiovec.InstantaneaContextoActor{
		VinculoRef: "vca_0123456789abcdefghijkl", VinculoVersion: 5,
		CuentaRef: cuenta.CuentaRef, PersonaRef: "per_0123456789abcdefghijkl", PersonaVersion: 3,
		PerfilActivoRef: "prf_0123456789abcdefghijkl", PerfilVersion: 4,
		Estado:       dominiovec.EstadoVinculoContextoActorActivo,
		VigenteDesde: instante.Add(-time.Hour), VigenteHasta: instante.Add(30 * time.Minute),
	}
	actor, err := dominiovec.NuevoContextoActor(cuenta, instantanea, instante.Add(-2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	autenticacion := dominiovec.AutenticacionRevalidadaV1{
		AutenticacionRef: "aut_0123456789abcdefghijkl", AutenticacionHuellaSHA256: huellaConvocatoriaPrueba('1'),
		AsercionRef: "ase_0123456789abcdefghijkl", SesionRef: "ses_0123456789abcdefghijkl",
		ControlSesionRef: "cse_0123456789abcdefghijkl", ControlSesionRevision: 7,
		ControlSesionHuellaSHA256: huellaConvocatoriaPrueba('2'), CuentaRef: cuenta.CuentaRef,
		CuentaOrdinariaRef: cuenta.CuentaRef, Superficie: dominiovec.SuperficieAutenticacionExternaPersonalV1,
		MetodoObservado: cuenta.Metodo, GarantiaObservada: cuenta.Garantia,
		PoliticaGarantiaRef:          "pga_0123456789abcdefghijkl",
		PoliticaGarantiaHuellaSHA256: huellaConvocatoriaPrueba('3'),
		AutenticacionVerificadaEn:    instante.Add(-5 * time.Minute), SesionEmitidaEn: instante.Add(-4 * time.Minute),
		SesionRevalidadaEn: instante.Add(-3 * time.Minute), SesionValidaHasta: instante.Add(10 * time.Minute),
	}
	vinculo, err := dominiovec.CrearVinculoAutenticacionActorV1(
		context.Background(), revalidadorSuperficieConvocatoriaPrueba{resultado: autenticacion},
		dominiovec.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: autenticacion.AutenticacionRef, SesionRef: autenticacion.SesionRef,
		}, actor, instante,
	)
	if err != nil {
		t.Fatal(err)
	}
	return vinculo
}

func testimonioIdempotenciaConvocatoriaPrueba(
	t *testing.T,
	material MaterialIntencionGobiernoConvocatoria,
	autorizacion puertosvec.EvidenciaUsoDecisionAutorizacion,
) TestimonioIdempotenciaConvocatoria {
	t.Helper()
	huella, err := material.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	datosAutorizacion, err := autorizacion.Datos()
	if err != nil {
		t.Fatal(err)
	}
	testimonio, err := NuevoTestimonioIdempotenciaConvocatoria(DatosTestimonioIdempotenciaConvocatoria{
		Version: VersionTestimonioIdempotenciaConvocatoriaV1, GeneracionClave: 1,
		ClaveHMACRef: "clave:idempotencia:v1", ProtectorRef: "protector:idempotencia:v1",
		AtestacionRef: "atestacion:idempotencia:001", HuellaAtestacionSHA256: huellaConvocatoriaPrueba('c'),
		IndiceOperacionHMACSHA256: hmacConvocatoriaPrueba("indice", 'b'),
		PrincipalRef:              datosAutorizacion.Decision.PrincipalID, HuellaIntencionSHA256: huella,
		EmitidoEn:   instanteGobiernoConvocatoriaPrueba.Add(-time.Minute),
		ValidoHasta: instanteGobiernoConvocatoriaPrueba.Add(5 * time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	return testimonio
}

func principalAutorizacionConvocatoria(
	t *testing.T,
	autorizacion puertosvec.EvidenciaUsoDecisionAutorizacion,
) string {
	t.Helper()
	datos, err := autorizacion.Datos()
	if err != nil {
		t.Fatal(err)
	}
	return datos.Decision.PrincipalID
}

func huellaConvocatoriaPrueba(marca byte) string {
	bytes := make([]byte, 64)
	for indice := range bytes {
		bytes[indice] = marca
	}
	return string(bytes)
}

func hmacConvocatoriaPrueba(clave string, marca byte) string {
	return "hmac-sha256:" + clave + ":" + huellaConvocatoriaPrueba(marca)
}

func hmacMotivoConvocatoriaPrueba(marca byte) HMACMotivoGobiernoConvocatoria {
	return HMACMotivoGobiernoConvocatoria{
		DominioCriptografico: DominioCriptograficoMotivoGobiernoConvocatoriaV1,
		GeneracionClave:      3, ClaveHMACRef: "motivo-gobierno-v3", ValorHMACSHA256: huellaConvocatoriaPrueba(marca),
	}
}

func atestacionMotivoConvocatoriaPrueba(
	t *testing.T,
	accion, convocatoriaRef string,
	marca byte,
) AtestacionSelladoMotivoConvocatoria {
	t.Helper()
	solicitud := SolicitudSellarMotivoGobiernoConvocatoria{
		DominioCriptografico: DominioCriptograficoMotivoGobiernoConvocatoriaV1,
		Accion:               accion, ConvocatoriaRef: convocatoriaRef,
		PrincipalRef:   "per_0123456789abcdefghijkl",
		CorrelacionRef: "correlacion:convocatoria:001",
		Motivo:         "Motivo administrativo exacto de la operacion.",
		SolicitadaEn:   instanteGobiernoConvocatoriaPrueba.Add(-2 * time.Minute),
	}
	huella, err := solicitud.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	atestacion, err := NuevaAtestacionSelladoMotivoConvocatoria(
		solicitud,
		DatosAtestacionSelladoMotivoConvocatoria{
			HMAC: hmacMotivoConvocatoriaPrueba(marca), Accion: accion,
			ConvocatoriaRef: convocatoriaRef, PrincipalRef: solicitud.PrincipalRef,
			CorrelacionRef: solicitud.CorrelacionRef, HuellaSolicitudSHA256: huella,
			SelladorRef: "sellador:motivo:v3", AtestacionRef: "atestacion:motivo:" + string(marca),
			HuellaAtestacionSHA256: huellaConvocatoriaPrueba(marca),
			TokenConsumoRef:        "consumo:motivo:" + string(marca),
			AtestacionEmitidaEn:    instanteGobiernoConvocatoriaPrueba.Add(-time.Minute),
			AtestacionValidaHasta:  instanteGobiernoConvocatoriaPrueba.Add(3 * time.Minute),
		},
	)
	if err != nil {
		t.Fatalf("crear atestacion de motivo: %v", err)
	}
	return atestacion
}

func stringsSeguroAccionConvocatoria(accion string) string {
	resultado := make([]byte, len(accion))
	copy(resultado, accion)
	for indice, caracter := range resultado {
		if caracter == '.' {
			resultado[indice] = '-'
		}
	}
	return string(resultado)
}
