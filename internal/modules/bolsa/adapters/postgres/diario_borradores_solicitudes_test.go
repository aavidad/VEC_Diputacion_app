package postgres

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	gobiernoconvocatorias "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoconvocatorias"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
	pruebasvec "vec-diputacion-granada/internal/vec/pruebas"
)

var errCapturaReservaDiarioPostgreSQL = errors.New("captura deliberada de reserva")

type relojCapturaReservaDiarioPostgreSQL struct{ instante time.Time }

func (r relojCapturaReservaDiarioPostgreSQL) Ahora() time.Time { return r.instante }

type catalogoCapturaReservaDiarioPostgreSQL struct {
	plantilla gobiernoconvocatorias.PlantillaBorradorResuelta
	ambito    dominiobolsa.AmbitoOrganizativoConvocatoria
}

func (c catalogoCapturaReservaDiarioPostgreSQL) ResolverPlantillaBorrador(
	_ context.Context, selector gobiernoconvocatorias.SelectorPlantillaBorrador, _ time.Time,
) (gobiernoconvocatorias.PlantillaBorradorResuelta, error) {
	if selector.ID != c.plantilla.Referencia.ID || selector.Version != c.plantilla.Referencia.Version ||
		selector.HuellaContenidoSHA256 != c.plantilla.Referencia.HuellaContenidoSHA256 {
		return gobiernoconvocatorias.PlantillaBorradorResuelta{}, errors.New("plantilla inesperada")
	}
	return c.plantilla, nil
}

func (c catalogoCapturaReservaDiarioPostgreSQL) PrepararAltaBorrador(
	_ context.Context, plantilla gobiernoconvocatorias.PlantillaBorradorResuelta,
	_, _ string, _ time.Time,
) (gobiernoconvocatorias.PreparacionAltaBorrador, error) {
	return gobiernoconvocatorias.PreparacionAltaBorrador{
		Plantilla: plantilla, ID: "proceso:bolsa:auxiliar-captura",
		InstanciaFlujoRef:  "instancia:flujo:convocatoria:captura",
		AmbitoOrganizativo: c.ambito,
	}, nil
}

func (catalogoCapturaReservaDiarioPostgreSQL) ResolverMotivoBorrador(
	_ context.Context, referencia dominiovec.ReferenciaEntradaCatalogo, _ time.Time,
) (dominiovec.ReferenciaEntradaCatalogo, error) {
	if referencia.Validar() != nil {
		return dominiovec.ReferenciaEntradaCatalogo{}, errors.New("motivo invalido")
	}
	return referencia, nil
}

type lectorCapturaReservaDiarioPostgreSQL struct{}

func (lectorCapturaReservaDiarioPostgreSQL) ObtenerBorradorExacto(
	context.Context, puertosbolsa.ReferenciaEstadoVersionConvocatoria,
) (dominiobolsa.VersionConvocatoriaGobernada, error) {
	return dominiobolsa.VersionConvocatoriaGobernada{}, errors.New("lector no invocable en alta")
}

type comprometedorCapturaReservaDiarioPostgreSQL struct{}

func (comprometedorCapturaReservaDiarioPostgreSQL) ComprometerMotivo(
	_ context.Context, solicitud puertosbolsa.SolicitudComprometerMotivoGobiernoConvocatoria,
) (puertosbolsa.HMACMotivoGobiernoConvocatoria, error) {
	return puertosbolsa.HMACMotivoGobiernoConvocatoria{
		DominioCriptografico: solicitud.DominioCriptografico, GeneracionClave: 3,
		ClaveHMACRef: "motivo-gobierno-v3", HuellaEntradaSHA256: solicitud.HuellaSolicitudSHA256,
		ValorHMACSHA256: strings.Repeat("a", 64),
	}, nil
}

type derivadorCapturaReservaDiarioPostgreSQL struct{}

func (derivadorCapturaReservaDiarioPostgreSQL) Derivar(
	_ context.Context, solicitud gobiernoconvocatorias.SolicitudDerivacionIdempotencia,
) (gobiernoconvocatorias.ConjuntoIdentidadesOperacion, error) {
	preimagenL, preimagenF, err := solicitud.MaterialParaConectorConfiable()
	if err != nil {
		return gobiernoconvocatorias.ConjuntoIdentidadesOperacion{}, err
	}
	identidades := make([]gobiernoconvocatorias.IdentidadOperacionDerivada, 0, 2)
	for _, generacion := range []uint32{2, 1} {
		macL := hmac.New(sha256.New, []byte(fmt.Sprintf("clave-localizador-captura-%02d", generacion)))
		_, _ = macL.Write(preimagenL)
		macF := hmac.New(sha256.New, []byte(fmt.Sprintf("clave-huella-captura-%02d", generacion)))
		_, _ = macF.Write(preimagenF)
		refL, errL := gobiernoconvocatorias.NuevaReferenciaClaveHMACLocalizador(
			fmt.Sprintf("clave:hmac:convocatorias:localizador:captura:v%d", generacion), generacion,
		)
		refF, errF := gobiernoconvocatorias.NuevaReferenciaClaveHMACHuellaSolicitud(
			fmt.Sprintf("clave:hmac:convocatorias:huella:captura:v%d", generacion), generacion,
		)
		localizador, errLocalizador := gobiernoconvocatorias.NuevoLocalizadorOperacion(
			2, refL, hex.EncodeToString(macL.Sum(nil)),
		)
		huella, errHuella := gobiernoconvocatorias.NuevaHuellaSolicitud(
			2, refF, hex.EncodeToString(macF.Sum(nil)),
		)
		identidad, errIdentidad := gobiernoconvocatorias.NuevaIdentidadOperacionDerivada(localizador, huella)
		if err := errors.Join(errL, errF, errLocalizador, errHuella, errIdentidad); err != nil {
			return gobiernoconvocatorias.ConjuntoIdentidadesOperacion{}, err
		}
		identidades = append(identidades, identidad)
	}
	return gobiernoconvocatorias.NuevoConjuntoIdentidadesOperacion(identidades...)
}

type autorizadorCapturaReservaDiarioPostgreSQL struct{}

func (autorizadorCapturaReservaDiarioPostgreSQL) EvaluarDecisionBorrador(
	_ context.Context, actor dominiovec.ContextoActor, vinculo dominiovec.VinculoAutenticacionActorV1,
	recurso dominiovec.RecursoAutorizable, correlacion string,
	_ dominiovec.ReferenciaEntradaCatalogo, _ gobiernoconvocatorias.IntencionBorradorCanonica,
	instante time.Time,
) (gobiernoconvocatorias.ResultadoEvaluacionPDPBorrador, error) {
	huellaContexto, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return gobiernoconvocatorias.ResultadoEvaluacionPDPBorrador{}, err
	}
	huellaCatalogo, err := dominiovec.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		return gobiernoconvocatorias.ResultadoEvaluacionPDPBorrador{}, err
	}
	decision := dominiovec.DecisionAutorizacion{
		DecisionRef: "decision:pdp:borrador:captura", Concedida: true, Codigo: "concedida",
		PrincipalID: actor.PersonaRef, PerfilActivoRef: actor.PerfilActivoRef,
		Accion: puertosbolsa.AccionCrearBorradorConvocatoria, RecursoRef: recurso.Referencia,
		ModuloID:                    puertosbolsa.ModuloGobiernoConvocatorias,
		TipoRecurso:                 puertosbolsa.TipoRecursoVersionConvocatoriaGobernada,
		ContextoRecursoHuellaSHA256: huellaContexto,
		Finalidad:                   puertosbolsa.FinalidadGobiernoConvocatorias, CorrelacionRef: correlacion,
		VinculoAutenticacionActor: vinculo, AsignacionRef: "asignacion:rrhh:captura:v1",
		AsignacionHuellaSHA256: strings.Repeat("1", 64), VersionRolRef: "rol:rrhh:captura:v1",
		VersionRolHuellaSHA256:                strings.Repeat("2", 64),
		ControlVigenciaVersionRolRef:          "rol:rrhh:captura:v1",
		ControlVigenciaVersionRolRevision:     1,
		ControlVigenciaVersionRolHuellaSHA256: strings.Repeat("3", 64),
		RevisionCatalogoPoliticas:             1, CatalogoPoliticasHuellaSHA256: huellaCatalogo,
		PoliticasEvaluadasHuellasSHA256: map[string]string{},
		GarantiaMinima:                  dominiovec.AuthAssuranceHigh,
		CamposPermitidos:                []string{"auditoria", "evento_outbox", "version_convocatoria"},
		EmitidaEn:                       instante, ValidaHasta: instante.Add(4 * time.Minute),
	}
	evidencia, err := puertosvec.NuevaEvidenciaUsoDecisionAutorizacion(decision, instante)
	if err != nil {
		return gobiernoconvocatorias.ResultadoEvaluacionPDPBorrador{}, err
	}
	return gobiernoconvocatorias.ResultadoEvaluacionPDPBorrador{
		Estado: gobiernoconvocatorias.EvaluacionPDPConcedida,
		Concesion: gobiernoconvocatorias.ConcesionBorradorDurable{
			Evidencia: evidencia,
			Atestacion: gobiernoconvocatorias.ProyeccionAtestacionPDP{
				DecisionRef: decision.DecisionRef, AtestacionRef: "atestacion:pdp:borrador:captura",
				VersionAtestacion: 1, EstadoAtestacion: "activa",
				HuellaAtestacionSHA256: strings.Repeat("4", 64),
				VerificadorRef:         "verificador:pdp:captura:v1", VerificadaEn: instante,
			},
		},
	}, nil
}

type diarioCapturaReservaPostgreSQL struct {
	reserva *gobiernoconvocatorias.SolicitudReservaDecisionBorrador
}

func (d *diarioCapturaReservaPostgreSQL) ConsultarIdentidades(
	context.Context, gobiernoconvocatorias.SolicitudConsultaIdentidadesBorrador,
) (gobiernoconvocatorias.ResultadoConsultaIdentidadesBorrador, error) {
	return gobiernoconvocatorias.ResultadoConsultaIdentidadesBorrador{}, nil
}

func (d *diarioCapturaReservaPostgreSQL) ReservarDecision(
	_ context.Context, solicitud gobiernoconvocatorias.SolicitudReservaDecisionBorrador,
) (gobiernoconvocatorias.ResultadoReservaDecisionBorrador, error) {
	copia := solicitud
	d.reserva = &copia
	return gobiernoconvocatorias.ResultadoReservaDecisionBorrador{}, errCapturaReservaDiarioPostgreSQL
}

func (*diarioCapturaReservaPostgreSQL) Reconciliar(
	context.Context, gobiernoconvocatorias.SolicitudReconciliacionBorrador,
) (gobiernoconvocatorias.ResultadoReconciliacionBorrador, error) {
	return gobiernoconvocatorias.ResultadoReconciliacionBorrador{}, errors.New("no invocable")
}

func (*diarioCapturaReservaPostgreSQL) ReclamarDecision(
	context.Context, gobiernoconvocatorias.SolicitudReclamacionDecisionBorrador,
) (gobiernoconvocatorias.ResultadoOperacionDiario, error) {
	return gobiernoconvocatorias.ResultadoOperacionDiario{}, errors.New("no invocable")
}

type selladorCapturaReservaPostgreSQL struct{}

func (selladorCapturaReservaPostgreSQL) VerificarYSellarMotivo(
	context.Context, gobiernoconvocatorias.SolicitudSelladoMotivoBorrador,
) (gobiernoconvocatorias.ProyeccionSelladoMotivoBorrador, error) {
	return gobiernoconvocatorias.ProyeccionSelladoMotivoBorrador{}, errors.New("no invocable")
}

type politicaCifradoCapturaReservaPostgreSQL struct{}

func (politicaCifradoCapturaReservaPostgreSQL) IdentidadAutoridadBorrador() gobiernoconvocatorias.IdentidadAutoridadBorrador {
	identidad, _ := gobiernoconvocatorias.NuevaIdentidadAutoridadBorrador(
		"politica-captura", "instancia-politica-captura", "credencial-politica-captura", "rol-politica-captura",
	)
	return identidad
}

func (politicaCifradoCapturaReservaPostgreSQL) SeleccionarPoliticaCifradoBorrador(
	context.Context, gobiernoconvocatorias.SolicitudSeleccionPoliticaCifradoBorrador,
) (gobiernoconvocatorias.PoliticaGobernadaCifradoBorrador, error) {
	return gobiernoconvocatorias.PoliticaGobernadaCifradoBorrador{}, errors.New("no invocable")
}

type perfilCifradoCapturaReservaPostgreSQL struct{}

func (perfilCifradoCapturaReservaPostgreSQL) IdentidadAutoridadBorrador() gobiernoconvocatorias.IdentidadAutoridadBorrador {
	identidad, _ := gobiernoconvocatorias.NuevaIdentidadAutoridadBorrador(
		"perfil-captura", "instancia-perfil-captura", "credencial-perfil-captura", "rol-perfil-captura",
	)
	return identidad
}

func (perfilCifradoCapturaReservaPostgreSQL) ResolverPerfilCifradoBorrador(
	context.Context, gobiernoconvocatorias.SolicitudResolucionPerfilCifradoBorrador,
) (gobiernoconvocatorias.ResolucionPerfilCifradoBorrador, error) {
	return gobiernoconvocatorias.ResolucionPerfilCifradoBorrador{}, errors.New("no invocable")
}

type cifradorCapturaReservaPostgreSQL struct{}

func (cifradorCapturaReservaPostgreSQL) CifrarBorrador(
	context.Context, gobiernoconvocatorias.SolicitudCifradoBorrador,
) (gobiernoconvocatorias.ResultadoCifradoBorrador, error) {
	return gobiernoconvocatorias.ResultadoCifradoBorrador{}, errors.New("no invocable")
}

type confirmadorCapturaReservaPostgreSQL struct{}

func (confirmadorCapturaReservaPostgreSQL) IdentidadAutoridadBorrador() gobiernoconvocatorias.IdentidadAutoridadBorrador {
	identidad, _ := gobiernoconvocatorias.NuevaIdentidadAutoridadBorrador(
		"confirmador-captura", "instancia-confirmador-captura", "credencial-confirmador-captura", "rol-confirmador-captura",
	)
	return identidad
}

func (confirmadorCapturaReservaPostgreSQL) ConfirmarBorrador(
	context.Context, gobiernoconvocatorias.SolicitudConfirmacionBorrador,
) (gobiernoconvocatorias.ResultadoConfirmacionAtomica, error) {
	return gobiernoconvocatorias.ResultadoConfirmacionAtomica{}, errors.New("no invocable")
}

type verificadorCapturaReservaPostgreSQL struct{}

func (verificadorCapturaReservaPostgreSQL) IdentidadAutoridadBorrador() gobiernoconvocatorias.IdentidadAutoridadBorrador {
	identidad, _ := gobiernoconvocatorias.NuevaIdentidadAutoridadBorrador(
		"verificador-captura", "instancia-verificador-captura", "credencial-verificador-captura", "rol-verificador-captura",
	)
	return identidad
}

func (verificadorCapturaReservaPostgreSQL) VerificarReciboBorrador(
	context.Context, gobiernoconvocatorias.ProyeccionReciboBorrador,
) error {
	return errors.New("no invocable")
}

func solicitudReservaCompletaDiarioPostgreSQLPrueba(
	t *testing.T,
) gobiernoconvocatorias.SolicitudReservaDecisionBorrador {
	t.Helper()
	actor, vinculo, err := pruebasvec.NuevoContextoYVinculo(
		instanteDiarioPostgreSQLPrueba, "per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl",
		dominiovec.AuthMethodCertificate, dominiovec.AuthAssuranceHigh,
	)
	if err != nil {
		t.Fatal(err)
	}
	contenido, configuracion, ambito := datosCapturaReservaDiarioPostgreSQLPrueba(t)
	plantilla := gobiernoconvocatorias.PlantillaBorradorResuelta{
		Referencia: dominiobolsa.ReferenciaConfiguracionConvocatoria{
			ID: "plantilla:bolsa:captura", Version: 2,
			HuellaContenidoSHA256: strings.Repeat("8", 64),
		},
		Configuracion: configuracion,
	}
	catalogo := catalogoCapturaReservaDiarioPostgreSQL{plantilla: plantilla, ambito: ambito}
	clave, err := gobiernoconvocatorias.NuevaClaveClienteIdempotenciaConvocatoria(
		"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
	)
	if err != nil {
		t.Fatal(err)
	}
	orden := gobiernoconvocatorias.OrdenCrearBorrador{
		ClaveCliente: clave, Actor: actor, VinculoAutenticacionActor: vinculo,
		Plantilla: gobiernoconvocatorias.SelectorPlantillaBorrador{
			ID: plantilla.Referencia.ID, Version: plantilla.Referencia.Version,
			HuellaContenidoSHA256: plantilla.Referencia.HuellaContenidoSHA256,
		},
		CodigoVersionPublica: "v1", Contenido: contenido,
		ExpedienteRef: "expediente:seleccion:captura",
		MotivoCatalogo: dominiovec.ReferenciaEntradaCatalogo{
			CatalogoID: "motivos_rrhh", CatalogoVersion: 1,
			CatalogoHuellaSHA256: strings.Repeat("9", 64), EntradaClave: "crear_borrador",
		},
		CorrelacionRef: "correlacion:convocatoria:captura",
	}
	diarioCaptura := &diarioCapturaReservaPostgreSQL{}
	procedencia, err := gobiernoconvocatorias.NuevaProcedenciaActoBorrador(
		"pruebas", gobiernoconvocatorias.AutoridadActoAutoritativa, "proveedor-pruebas", true,
	)
	if err != nil {
		t.Fatal(err)
	}
	servicio, err := gobiernoconvocatorias.NuevoServicioBorradores(
		relojCapturaReservaDiarioPostgreSQL{instanteDiarioPostgreSQLPrueba}, catalogo, catalogo,
		lectorCapturaReservaDiarioPostgreSQL{}, comprometedorCapturaReservaDiarioPostgreSQL{},
		derivadorCapturaReservaDiarioPostgreSQL{}, autorizadorCapturaReservaDiarioPostgreSQL{},
		diarioCaptura, selladorCapturaReservaPostgreSQL{}, politicaCifradoCapturaReservaPostgreSQL{},
		perfilCifradoCapturaReservaPostgreSQL{}, cifradorCapturaReservaPostgreSQL{},
		confirmadorCapturaReservaPostgreSQL{}, verificadorCapturaReservaPostgreSQL{}, procedencia,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = servicio.Crear(context.Background(), orden)
	if !errors.Is(err, errCapturaReservaDiarioPostgreSQL) || diarioCaptura.reserva == nil {
		t.Fatalf("no se capturo la reserva valida: %v", err)
	}
	if diarioCaptura.reserva.Validar() != nil {
		t.Fatal("el nucleo produjo una reserva invalida")
	}
	return *diarioCaptura.reserva
}

func datosCapturaReservaDiarioPostgreSQLPrueba(t *testing.T) (
	dominiobolsa.ContenidoPublicableConvocatoria,
	dominiobolsa.ConfiguracionFijadaConvocatoria,
	dominiobolsa.AmbitoOrganizativoConvocatoria,
) {
	t.Helper()
	ambito, err := dominiobolsa.NuevoAmbitoOrganizativoConvocatoria(
		"org_diputaciongranada", "uni_seleccionexterna",
	)
	if err != nil {
		t.Fatal(err)
	}
	contenido := dominiobolsa.ContenidoPublicableConvocatoria{
		IdentificadorPublico: "auxiliar-administrativo-captura", Tipo: "bolsa_temporal",
		CatalogoCategorias: dominiobolsa.ReferenciaCatalogoCategorias{
			CatalogoID: "categorias-profesionales", CatalogoVersion: 1,
			CatalogoHuellaSHA256: strings.Repeat("a", 64),
		},
		Categorias: []string{"auxiliar_administrativo"}, Titulo: "Bolsa temporal de auxiliares",
		Resumen: "Convocatoria publica para bolsa temporal.", Descripcion: "Proceso sujeto a bases.",
		Plazos: []dominiobolsa.PlazoConvocatoria{{
			Referencia: "plazo:inscripcion", Tipo: "inscripcion", Titulo: "Inscripcion",
			Descripcion: "Plazo de presentacion.", AbreEn: instanteDiarioPostgreSQLPrueba.Add(24 * time.Hour),
			CierraEn: instanteDiarioPostgreSQLPrueba.Add(30 * 24 * time.Hour),
		}},
		Requisitos: []dominiobolsa.RequisitoConvocatoria{{
			Referencia: "requisito:edad", Orden: 1, Titulo: "Edad",
			Descripcion: "Cumplir la edad exigida.", Obligatorio: true,
		}},
		Documentos: []dominiobolsa.DocumentoPublicableConvocatoria{{
			Referencia: "documento:bases", Tipo: "bases", Orden: 1, Titulo: "Bases",
			Descripcion: "Bases firmadas.", Formato: "pdf", URL: "/bolsa/documentos/bases.pdf",
		}},
	}
	referencia := func(id string, marca byte) dominiobolsa.ReferenciaConfiguracionConvocatoria {
		return dominiobolsa.ReferenciaConfiguracionConvocatoria{
			ID: id, Version: 1, HuellaContenidoSHA256: strings.Repeat(string(marca), 64),
		}
	}
	configuracion := dominiobolsa.ConfiguracionFijadaConvocatoria{
		Catalogos: referencia("catalogos:bolsa", '1'), Calendario: referencia("calendario:bolsa", '2'),
		ReglasBaremacion: referencia("baremo:bolsa", '3'),
		FlujoProceso:     referencia("convocatoria-bolsa", '4'),
		FlujoSolicitud:   referencia("solicitud-bolsa", '5'),
		Documentos: []dominiobolsa.ReferenciaDocumentoOficialConvocatoria{{
			Rol: "bases", PublicacionRef: "documento:bases", DocumentoRef: "documento:logico:bases:captura",
			VersionDocumento: 1, RepresentacionRef: "representacion:pdf:bases:captura",
			HuellaContenidoSHA256: strings.Repeat("b", 64),
			FirmaValidadaRef:      "firma:validada:bases:captura", ReciboCustodiaRef: "custodia:bases:captura",
		}},
	}
	return contenido, configuracion, ambito
}
