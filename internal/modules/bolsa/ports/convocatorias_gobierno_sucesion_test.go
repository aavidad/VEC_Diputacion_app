package ports

import (
	"errors"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
)

func TestConsultaPublicadaExigeFlujoEstableYAutorizacionExacta(t *testing.T) {
	borrador := versionGobernadaPuertosPrueba(t)
	publicada := publicarInicialConvocatoriaPuertosPrueba(
		t, borrador, instanteGobiernoConvocatoriaPrueba.Add(-10*time.Minute),
	)
	selector := SelectorVersionConvocatoriaExacta{ID: publicada.ID, Secuencia: publicada.Secuencia}
	recurso, err := RecursoAutorizableConsultaVersionConvocatoria(selector)
	if err != nil {
		t.Fatal(err)
	}
	autorizacion := evidenciaAutorizacionConvocatoriaPrueba(
		t, AccionConsultarVersionConFlujoConvocatoria, recurso,
		instanteGobiernoConvocatoriaPrueba,
	)
	solicitud := SolicitudConsultaVersionConvocatoriaAutorizada{
		Selector: selector, IncluirInstanciaFlujo: true,
		Autorizacion: autorizacion, ConsultadaEn: instanteGobiernoConvocatoriaPrueba,
	}
	datos, err := autorizacion.Datos()
	if err != nil {
		t.Fatal(err)
	}
	instancia := instanciaFlujoConvocatoriaPuertosPrueba(publicada)
	huellaVersion, err := publicada.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	huellaInstancia, err := instancia.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	resultado := ResultadoConsultaVersionConvocatoria{
		Version: publicada, InstanciaFlujo: &instancia,
		HuellaVersionSHA256: huellaVersion, HuellaInstanciaFlujoSHA256: huellaInstancia,
		AutorizacionRef:                    datos.Decision.DecisionRef,
		HuellaAutorizacionSHA256:           datos.HuellaDecisionSHA256,
		AtestacionAutorizacionRef:          "atestacion:pdp:consulta-flujo:001",
		HuellaAtestacionAutorizacionSHA256: huellaConvocatoriaPrueba('a'),
		ConsumoAutorizacionRef:             "consumo:autorizacion:consulta-flujo:001",
		AuditoriaRef:                       "auditoria:consulta-flujo:001",
		HuellaAuditoriaSHA256:              huellaConvocatoriaPrueba('b'),
		ConsultadaEn:                       instanteGobiernoConvocatoriaPrueba,
	}
	if err := resultado.ValidarPara(solicitud); err != nil {
		t.Fatalf("consulta publicada con flujo estable rechazada: %v", err)
	}
	sinHuellaFlujo := resultado
	sinHuellaFlujo.HuellaInstanciaFlujoSHA256 = huellaConvocatoriaPrueba('0')
	if err := sinHuellaFlujo.ValidarPara(solicitud); !errors.Is(err, ErrConsultaGobiernoConvocatoriaInvalida) {
		t.Fatalf("auditoria acepto una huella distinta del snapshot devuelto: %v", err)
	}
	flujoGemelo := instancia
	flujoGemelo.ID = "instancia:flujo:convocatoria:gemela"
	resultado.InstanciaFlujo = &flujoGemelo
	if err := resultado.ValidarPara(solicitud); !errors.Is(err, ErrConsultaGobiernoConvocatoriaInvalida) {
		t.Fatalf("se acepto otra instancia con igual entidad y definicion: %v", err)
	}

	flujoPorVersion := instancia
	flujoPorVersion.EntidadRef = publicada.Referencia()
	resultado.InstanciaFlujo = &flujoPorVersion
	if err := resultado.ValidarPara(solicitud); !errors.Is(err, ErrConsultaGobiernoConvocatoriaInvalida) {
		t.Fatalf("se acepto un flujo ligado a la version en vez de al proceso estable: %v", err)
	}

	resultado.InstanciaFlujo = &instancia
	sinPermisoFlujo := solicitud
	sinPermisoFlujo.IncluirInstanciaFlujo = false
	if err := resultado.ValidarPara(sinPermisoFlujo); !errors.Is(err, ErrConsultaGobiernoConvocatoriaInvalida) {
		t.Fatalf("se devolvio el flujo con una decision que no autoriza ese campo: %v", err)
	}
}

func TestPublicacionSucesoraMaterializaCASAtomicoDeAmbasVersiones(t *testing.T) {
	publicadaV1 := publicarInicialConvocatoriaPuertosPrueba(
		t, versionGobernadaPuertosPrueba(t),
		instanteGobiernoConvocatoriaPrueba.Add(-40*time.Minute),
	)
	borradorV2, err := publicadaV1.NuevaVersion(
		"v2", publicadaV1.Contenido, publicadaV1.Configuracion,
		"expediente:seleccion:2026-001", "persona:tecnica:segunda",
		"Preparacion de la segunda version.",
		instanteGobiernoConvocatoriaPrueba.Add(-30*time.Minute),
	)
	if err != nil {
		t.Fatalf("crear sucesora: %v", err)
	}
	huellaContenido, err := borradorV2.HuellaContenidoSHA256()
	if err != nil {
		t.Fatal(err)
	}
	huellaEstado, err := borradorV2.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	aprobacion := dominiobolsa.EvidenciaAprobacionConvocatoria{
		Accion: "publicar", Referencia: "aprobacion:publicacion:002",
		HuellaEvidenciaSHA256: huellaConvocatoriaPrueba('c'),
		ConvocatoriaRef:       borradorV2.Referencia(), Revision: borradorV2.Revision,
		HuellaContenidoSHA256: huellaContenido, HuellaEstadoSHA256: huellaEstado,
		AprobadaPor: "persona:supervisora:segunda",
		AprobadaEn:  instanteGobiernoConvocatoriaPrueba.Add(-20 * time.Minute),
	}
	dependencias := dominiobolsa.EvidenciaDependenciasConvocatoria{
		Referencia:            "dependencias:publicacion:002",
		HuellaEvidenciaSHA256: huellaConvocatoriaPrueba('d'),
		ConvocatoriaRef:       borradorV2.Referencia(), Revision: borradorV2.Revision,
		HuellaContenidoSHA256: huellaContenido, HuellaEstadoSHA256: huellaEstado,
		VerificadaEn: instanteGobiernoConvocatoriaPrueba.Add(-19 * time.Minute),
	}
	resultado, err := borradorV2.PublicarSucesora(
		publicadaV1, "persona:gestora:segunda", aprobacion, dependencias,
		"Publicacion y sustitucion atomicas.",
		instanteGobiernoConvocatoriaPrueba.Add(-10*time.Minute),
	)
	if err != nil {
		t.Fatalf("publicar sucesora: %v", err)
	}
	esperadaV2, err := EstadoVersionConvocatoria(borradorV2)
	if err != nil {
		t.Fatal(err)
	}
	esperadaV1, err := EstadoVersionConvocatoria(publicadaV1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := MaterialAltaBorradorConvocatoria(
		borradorV2, &esperadaV1, &publicadaV1, compromisoMotivoConvocatoriaPrueba(
			t, AccionCrearBorradorConvocatoria, borradorV2,
			borradorV2.CreadaPor, borradorV2.MotivoCreacion, 'f',
		),
	); !errors.Is(err, ErrMaterialIntencionConvocatoriaInvalido) {
		t.Fatalf("material V2 de alta acepto estado relacionado esperado: %v", err)
	}
	borradorConFlujoGemelo := borradorV2
	borradorConFlujoGemelo.InstanciaFlujoRef = "instancia:flujo:convocatoria:gemela"
	if borradorConFlujoGemelo.Validar() != nil {
		t.Fatal("el ataque debe usar un agregado individualmente valido")
	}
	if _, err := MaterialAltaBorradorConvocatoria(
		borradorConFlujoGemelo, &esperadaV1, &publicadaV1,
		compromisoMotivoConvocatoriaPrueba(
			t, AccionCrearBorradorConvocatoria, borradorConFlujoGemelo,
			borradorConFlujoGemelo.CreadaPor, borradorConFlujoGemelo.MotivoCreacion, 'f',
		),
	); !errors.Is(err, ErrMaterialIntencionConvocatoriaInvalido) {
		t.Fatalf("alta acepto una instancia de flujo distinta de la predecesora: %v", err)
	}

	if _, err := MaterialPublicacionConvocatoria(
		esperadaV2, resultado.Publicada, nil, nil, compromisoMotivoConvocatoriaPrueba(
			t, AccionPublicarYSustituirConvocatoria, resultado.Publicada,
			resultado.Publicada.PublicadaPor, resultado.Publicada.MotivoPublicacion, 'e',
		),
	); !errors.Is(err, ErrMaterialIntencionConvocatoriaInvalido) {
		t.Fatalf("publicacion sucesora sin CAS de predecesora aceptada: %v", err)
	}
	material, err := MaterialPublicacionConvocatoria(
		esperadaV2, resultado.Publicada, &esperadaV1, &resultado.Predecesora,
		compromisoMotivoConvocatoriaPrueba(
			t, AccionPublicarYSustituirConvocatoria, resultado.Publicada,
			resultado.Publicada.PublicadaPor, resultado.Publicada.MotivoPublicacion, 'e',
		),
	)
	if err != nil {
		t.Fatalf("material atomico valido rechazado: %v", err)
	}
	if material.Accion != AccionPublicarYSustituirConvocatoria ||
		material.EstadoRelacionadoEsperado == nil || material.EstadoRelacionadoNuevo == nil ||
		*material.EstadoRelacionadoEsperado != esperadaV1 ||
		material.EstadoRelacionadoNuevo.Referencia != esperadaV1.Referencia ||
		material.EstadoRelacionadoNuevo.HuellaEstadoSHA256 == esperadaV1.HuellaEstadoSHA256 {
		t.Fatal("la intencion no liga preestado y resultado de la predecesora")
	}
	parejaFabricada := resultado.Predecesora
	parejaFabricada.SustituidaPor = "persona:gestora:ajena"
	if parejaFabricada.Validar() != nil {
		t.Fatal("el ataque debe usar una predecesora individualmente valida")
	}
	if _, err := MaterialPublicacionConvocatoria(
		esperadaV2, resultado.Publicada, &esperadaV1, &parejaFabricada,
		compromisoMotivoConvocatoriaPrueba(
			t, AccionPublicarYSustituirConvocatoria, resultado.Publicada,
			resultado.Publicada.PublicadaPor, resultado.Publicada.MotivoPublicacion, 'e',
		),
	); !errors.Is(err, ErrMaterialIntencionConvocatoriaInvalido) {
		t.Fatalf("se aceptaron instantaneas que no forman la misma transicion: %v", err)
	}

	predecesoraAjena := esperadaV1
	predecesoraAjena.Referencia = "proceso:bolsa:ajeno#1"
	if _, err := MaterialPublicacionConvocatoria(
		esperadaV2, resultado.Publicada, &predecesoraAjena, &resultado.Predecesora,
		compromisoMotivoConvocatoriaPrueba(
			t, AccionPublicarYSustituirConvocatoria, resultado.Publicada,
			resultado.Publicada.PublicadaPor, resultado.Publicada.MotivoPublicacion, 'e',
		),
	); !errors.Is(err, ErrMaterialIntencionConvocatoriaInvalido) {
		t.Fatalf("se acepto una predecesora ajena: %v", err)
	}
}
