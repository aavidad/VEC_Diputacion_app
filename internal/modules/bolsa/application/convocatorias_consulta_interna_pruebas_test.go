package application

import (
	"context"
	"strings"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type exigidorConsultaConvocatoriaPrueba struct {
	instante     time.Time
	incluirFlujo bool
	error        error
	llamadas     int
	observar     func(dominiovec.RecursoAutorizable, string)
}

func (e *exigidorConsultaConvocatoriaPrueba) ExigirEvidencia(
	ctx context.Context,
	actor dominiovec.ContextoActor,
	vinculo dominiovec.VinculoAutenticacionActorV1,
	recurso dominiovec.RecursoAutorizable,
	correlacionRef string,
	motivo string,
	_ aplicacionvec.PoliticaUsoDecisionAutorizacion,
) (puertosvec.EvidenciaUsoDecisionAutorizacion, error) {
	e.llamadas++
	if e.observar != nil {
		e.observar(recurso, motivo)
	}
	if e.error != nil {
		return puertosvec.EvidenciaUsoDecisionAutorizacion{}, e.error
	}
	if err := ctx.Err(); err != nil {
		return puertosvec.EvidenciaUsoDecisionAutorizacion{}, err
	}
	accion := puertosbolsa.AccionConsultarVersionConvocatoria
	campos := []string{"version_convocatoria"}
	if e.incluirFlujo {
		accion = puertosbolsa.AccionConsultarVersionConFlujoConvocatoria
		campos = []string{"instancia_flujo", "version_convocatoria"}
	}
	huellaRecurso, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return puertosvec.EvidenciaUsoDecisionAutorizacion{}, err
	}
	huellaCatalogo, err := dominiovec.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		return puertosvec.EvidenciaUsoDecisionAutorizacion{}, err
	}
	decision := dominiovec.DecisionAutorizacion{
		DecisionRef: "autorizacion:consulta:convocatoria:aplicacion:001",
		Concedida:   true, Codigo: "concedida", PrincipalID: actor.Principal.ID,
		PerfilActivoRef: actor.PerfilActivoRef, Accion: accion, RecursoRef: recurso.Referencia,
		ModuloID: recurso.ModuloID, TipoRecurso: recurso.Tipo,
		ContextoRecursoHuellaSHA256: huellaRecurso,
		Finalidad:                   puertosbolsa.FinalidadConsultaInternaConvocatorias,
		CorrelacionRef:              correlacionRef, VinculoAutenticacionActor: vinculo,
		AsignacionRef: "asignacion:rrhh:consulta:v1", AsignacionHuellaSHA256: strings.Repeat("7", 64),
		VersionRolRef: "rol:rrhh:consulta:v1", VersionRolHuellaSHA256: strings.Repeat("8", 64),
		ControlVigenciaVersionRolRef: "rol:rrhh:consulta:v1", ControlVigenciaVersionRolRevision: 1,
		ControlVigenciaVersionRolHuellaSHA256: strings.Repeat("9", 64),
		RevisionCatalogoPoliticas:             1, CatalogoPoliticasHuellaSHA256: huellaCatalogo,
		PoliticasEvaluadasHuellasSHA256: map[string]string{},
		GarantiaMinima:                  dominiovec.AuthAssuranceHigh, CamposPermitidos: campos,
		EmitidaEn: e.instante.Add(-time.Minute), ValidaHasta: e.instante.Add(4 * time.Minute),
	}
	return puertosvec.NuevaEvidenciaUsoDecisionAutorizacion(decision, e.instante)
}

type consultaVersionConvocatoriaPrueba struct {
	version   dominiobolsa.VersionConvocatoriaGobernada
	error     error
	llamadas  int
	antes     func(puertosbolsa.SolicitudConsultaVersionConvocatoriaAutorizada)
	manipular func(*puertosbolsa.ResultadoConsultaVersionConvocatoria)
}

func (c *consultaVersionConvocatoriaPrueba) ObtenerVersionExacta(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudConsultaVersionConvocatoriaAutorizada,
) (puertosbolsa.ResultadoConsultaVersionConvocatoria, error) {
	c.llamadas++
	if c.antes != nil {
		c.antes(solicitud)
	}
	if c.error != nil {
		return puertosbolsa.ResultadoConsultaVersionConvocatoria{}, c.error
	}
	if err := ctx.Err(); err != nil {
		return puertosbolsa.ResultadoConsultaVersionConvocatoria{}, err
	}
	datos, err := solicitud.Autorizacion.Datos()
	if err != nil {
		return puertosbolsa.ResultadoConsultaVersionConvocatoria{}, err
	}
	huellaVersion, err := c.version.HuellaSHA256()
	if err != nil {
		return puertosbolsa.ResultadoConsultaVersionConvocatoria{}, err
	}
	resultado := puertosbolsa.ResultadoConsultaVersionConvocatoria{
		Version: c.version, HuellaVersionSHA256: huellaVersion,
		AutorizacionRef:                    datos.Decision.DecisionRef,
		HuellaAutorizacionSHA256:           datos.HuellaDecisionSHA256,
		AtestacionAutorizacionRef:          "atestacion:pdp:consulta:aplicacion:001",
		HuellaAtestacionAutorizacionSHA256: strings.Repeat("3", 64),
		ConsumoAutorizacionRef:             "consumo:autorizacion:consulta:aplicacion:001",
		AuditoriaRef:                       "auditoria:consulta:aplicacion:001",
		HuellaAuditoriaSHA256:              strings.Repeat("4", 64), ConsultadaEn: solicitud.ConsultadaEn,
	}
	if c.manipular != nil {
		c.manipular(&resultado)
	}
	return resultado, nil
}

func nuevaVersionConsultaConvocatoriaAplicacionPrueba(
	t *testing.T,
) dominiobolsa.VersionConvocatoriaGobernada {
	t.Helper()
	huella := func(marca string) string { return strings.Repeat(marca, 64) }
	contenido := dominiobolsa.ContenidoPublicableConvocatoria{
		IdentificadorPublico: "auxiliar-administrativo-2026", Tipo: "bolsa_temporal",
		CatalogoCategorias: dominiobolsa.ReferenciaCatalogoCategorias{
			CatalogoID: "categorias-profesionales", CatalogoVersion: 1,
			CatalogoHuellaSHA256: huella("a"),
		},
		Categorias: []string{"auxiliar_administrativo"}, Titulo: "Bolsa temporal de auxiliares",
		Resumen:     "Convocatoria publica para la bolsa temporal.",
		Descripcion: "Proceso selectivo sujeto a bases firmadas y publicadas.",
		Plazos: []dominiobolsa.PlazoConvocatoria{{
			Referencia: "plazo:inscripcion", Tipo: "inscripcion", Titulo: "Inscripcion",
			Descripcion: "Plazo de presentacion.", AbreEn: instantePanelInternoPrueba.Add(time.Hour),
			CierraEn: instantePanelInternoPrueba.Add(30 * 24 * time.Hour),
		}},
		Documentos: []dominiobolsa.DocumentoPublicableConvocatoria{{
			Referencia: "documento:bases", Tipo: "bases", Orden: 1, Titulo: "Bases",
			Descripcion: "Bases de la convocatoria.", Formato: "pdf", URL: "/bolsa/documentos/bases.pdf",
		}},
	}
	referencia := func(id string, marca string) dominiobolsa.ReferenciaConfiguracionConvocatoria {
		return dominiobolsa.ReferenciaConfiguracionConvocatoria{
			ID: id, Version: 1, HuellaContenidoSHA256: huella(marca),
		}
	}
	configuracion := dominiobolsa.ConfiguracionFijadaConvocatoria{
		Catalogos: referencia("catalogos:bolsa", "1"), Calendario: referencia("calendario:auxiliar", "2"),
		ReglasBaremacion: referencia("baremo:auxiliar", "3"),
		FlujoProceso:     referencia("convocatoria-bolsa", "4"), FlujoSolicitud: referencia("solicitud-bolsa", "5"),
		Plantilla: referencia("plantilla:bolsa:consulta", "8"),
		Documentos: []dominiobolsa.ReferenciaDocumentoOficialConvocatoria{{
			Rol: "bases", PublicacionRef: "documento:bases", DocumentoRef: "documento:logico:bases:001",
			VersionDocumento: 1, RepresentacionRef: "representacion:pdf:bases:001",
			HuellaContenidoSHA256: huella("6"), FirmaValidadaRef: "firma:validada:bases:001",
			ReciboCustodiaRef: "custodia:bases:001",
		}},
	}
	ambito, err := dominiobolsa.NuevoAmbitoOrganizativoConvocatoria(
		"org_diputaciongranada", "uni_seleccionexterna",
	)
	if err != nil {
		t.Fatal(err)
	}
	version, err := dominiobolsa.NuevaVersionConvocatoriaGobernada(
		dominiobolsa.DatosNuevaVersionConvocatoriaGobernada{
			ID: "proceso:bolsa:auxiliar-2026", CodigoVersionPublica: "v1",
			InstanciaFlujoRef: "instancia:flujo:convocatoria:001", AmbitoOrganizativo: ambito,
			Contenido:     contenido,
			Configuracion: configuracion, ExpedienteRef: "expediente:seleccion:2026-001",
			Motivo: "Preparacion administrativa.", ActorID: "persona:tecnica:001",
			Instante: instantePanelInternoPrueba.Add(-time.Hour),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return version
}

func nuevaOrdenConsultaConvocatoriaAplicacionPrueba(
	t *testing.T,
	version dominiobolsa.VersionConvocatoriaGobernada,
) OrdenConsultaVersionConvocatoria {
	t.Helper()
	actor, vinculo := nuevoContextoYVinculoPanelPrueba(
		t, dominiovec.AuthMethodCertificate, dominiovec.AuthAssuranceHigh,
		dominiovec.SuperficieAutenticacionInternaCorporativaV1,
	)
	return OrdenConsultaVersionConvocatoria{
		ContextoActor: actor, VinculoAutenticacionActor: vinculo,
		Selector: puertosbolsa.SelectorVersionConvocatoriaExacta{
			ID: version.ID, Secuencia: version.Secuencia,
		},
		CorrelacionRef: "correlacion:consulta:convocatoria:aplicacion:001",
	}
}
