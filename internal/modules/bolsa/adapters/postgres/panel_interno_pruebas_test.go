package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
	pruebasvec "vec-diputacion-granada/internal/vec/pruebas"
)

var instantePanelPostgreSQLPrueba = time.Date(
	2026, time.July, 17, 14, 0, 0, 123_456_000, time.UTC,
)

type generadorCorrelacionPanelPostgreSQLPrueba struct{ valor string }

func (g generadorCorrelacionPanelPostgreSQLPrueba) NuevaReferenciaCorrelacionAutorizacionV2(
	context.Context,
) (string, error) {
	return g.valor, nil
}

type iniciadorPanelPostgreSQLPrueba struct {
	tx       pgx.Tx
	opciones []pgx.TxOptions
}

func (i *iniciadorPanelPostgreSQLPrueba) BeginTx(
	_ context.Context,
	opciones pgx.TxOptions,
) (pgx.Tx, error) {
	i.opciones = append(i.opciones, opciones)
	return i.tx, nil
}

type transaccionPanelPostgreSQLPrueba struct {
	pgx.Tx
	fila            pgx.Row
	consulta        string
	argumentos      []any
	configuraciones int
	confirmaciones  int
	reversiones     int
	errorCommit     error
}

func (t *transaccionPanelPostgreSQLPrueba) Exec(
	context.Context,
	string,
	...any,
) (pgconn.CommandTag, error) {
	t.configuraciones++
	return pgconn.NewCommandTag("SELECT 1"), nil
}

func (t *transaccionPanelPostgreSQLPrueba) QueryRow(
	_ context.Context,
	consulta string,
	argumentos ...any,
) pgx.Row {
	t.consulta = consulta
	t.argumentos = make([]any, len(argumentos))
	for indice, argumento := range argumentos {
		if contenido, esBinario := argumento.([]byte); esBinario {
			t.argumentos[indice] = append([]byte(nil), contenido...)
			continue
		}
		t.argumentos[indice] = argumento
	}
	return t.fila
}

func (t *transaccionPanelPostgreSQLPrueba) Commit(context.Context) error {
	t.confirmaciones++
	return t.errorCommit
}

func (t *transaccionPanelPostgreSQLPrueba) Rollback(context.Context) error {
	t.reversiones++
	return nil
}

type filaPanelPostgreSQLPrueba struct {
	contenido []byte
	error     error
}

func (f filaPanelPostgreSQLPrueba) Scan(destinos ...any) error {
	if f.error != nil {
		return f.error
	}
	if len(destinos) != 1 {
		return errors.New("numero de columnas del panel inesperado")
	}
	destino, valido := destinos[0].(*[]byte)
	if !valido {
		return errors.New("tipo de columna del panel inesperado")
	}
	*destino = append([]byte(nil), f.contenido...)
	return nil
}

func solicitudPanelPostgreSQLPrueba(
	t *testing.T,
) puertosbolsa.SolicitudConsultaPanelInterno {
	t.Helper()
	selector := selectorPanelPostgreSQLPrueba()
	motivo := motivoPanelPostgreSQLPrueba()
	correlacion := correlacionPanelPostgreSQLPrueba(t)
	actor, vinculo, err := pruebasvec.NuevoContextoYVinculo(
		instantePanelPostgreSQLPrueba,
		"per_0123456789abcdefghijkl",
		"prf_0123456789abcdefghijkl",
		dominiovec.AuthMethodCertificate,
		dominiovec.AuthAssuranceHigh,
	)
	if err != nil {
		t.Fatal(err)
	}
	recurso, err := puertosbolsa.RecursoAutorizablePanelInterno(selector, motivo)
	if err != nil {
		t.Fatal(err)
	}
	huellaContexto, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	correlacionRef, err := correlacion.ValorCanonico()
	if err != nil {
		t.Fatal(err)
	}
	huellaCatalogo, err := dominiovec.HuellaEvidenciasCatalogoPoliticasAutorizacion(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	solicitudPDP, err := dominiovec.NuevaSolicitudAutorizacionLigadaV2(
		dominiovec.DatosSolicitudAutorizacionLigadaV2{
			ContextoActor: actor, VinculoAutenticacionActor: vinculo,
			ReferenciaMotivo: motivo, Accion: puertosbolsa.AccionConsultarPanelInterno,
			Recurso: recurso, Finalidad: puertosbolsa.FinalidadPanelInternoBolsa,
			Correlacion: correlacion,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	huellaSolicitud, err := dominiovec.HuellaSHA256SolicitudAutorizacionV2(solicitudPDP)
	if err != nil {
		t.Fatal(err)
	}
	huellaMotivo, err := dominiovec.HuellaSHA256MotivoAutorizacionV2(motivo)
	if err != nil {
		t.Fatal(err)
	}
	decision := dominiovec.DecisionAutorizacion{
		DecisionRef: "decision:panel:postgresql:00000001", Concedida: true, Codigo: "concedida",
		PrincipalID: actor.Principal.ID, PerfilActivoRef: actor.PerfilActivoRef,
		Accion: puertosbolsa.AccionConsultarPanelInterno, RecursoRef: recurso.Referencia,
		ModuloID: recurso.ModuloID, TipoRecurso: recurso.Tipo,
		ContextoRecursoHuellaSHA256: huellaContexto,
		Finalidad:                   puertosbolsa.FinalidadPanelInternoBolsa, CorrelacionRef: correlacionRef,
		EsquemaHuellaSolicitud:    dominiovec.EsquemaHuellaSolicitudAutorizacionV2,
		SolicitudHuellaSHA256:     huellaSolicitud,
		EsquemaHuellaMotivo:       dominiovec.EsquemaHuellaMotivoAutorizacionV2,
		MotivoHuellaSHA256:        huellaMotivo,
		VinculoAutenticacionActor: vinculo,
		AsignacionRef:             "asignacion:panel:postgresql:v1",
		AsignacionHuellaSHA256:    strings.Repeat("a", 64),
		VersionRolRef:             "rol:panel_rrhh:v1", VersionRolHuellaSHA256: strings.Repeat("b", 64),
		ControlVigenciaVersionRolRef:          "rol:panel_rrhh:v1",
		ControlVigenciaVersionRolRevision:     1,
		ControlVigenciaVersionRolHuellaSHA256: strings.Repeat("c", 64),
		RevisionCatalogoPoliticas:             1, CatalogoPoliticasHuellaSHA256: huellaCatalogo,
		GarantiaMinima:   dominiovec.AuthAssuranceHigh,
		CamposPermitidos: []string{puertosbolsa.CampoPanelInternoAgregado},
		EmitidaEn:        instantePanelPostgreSQLPrueba.Add(-time.Second),
		ValidaHasta:      instantePanelPostgreSQLPrueba.Add(time.Minute),
	}
	evidencia, err := puertosvec.NuevaEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2(
		decision,
		instantePanelPostgreSQLPrueba,
	)
	if err != nil {
		t.Fatalf("crear evidencia V2 del panel: %v", err)
	}
	solicitud, err := puertosbolsa.NuevaSolicitudConsultaPanelInterno(
		selector,
		evidencia,
		motivo,
		correlacion,
		instantePanelPostgreSQLPrueba,
	)
	if err != nil {
		t.Fatalf("crear solicitud durable del panel: %v", err)
	}
	return solicitud
}

func instantaneaPanelPostgreSQLPrueba(
	t *testing.T,
	solicitud puertosbolsa.SolicitudConsultaPanelInterno,
) puertosbolsa.InstantaneaPanelInterno {
	t.Helper()
	selector, err := solicitud.Selector()
	autorizacion, errAutorizacion := solicitud.Autorizacion()
	datos, errDatos := autorizacion.Datos()
	correlacion, errCorrelacion := solicitud.Correlacion()
	correlacionRef, errCorrelacionRef := correlacion.ValorCanonico()
	if err != nil || errAutorizacion != nil || errDatos != nil ||
		errCorrelacion != nil || errCorrelacionRef != nil {
		t.Fatalf("extraer solicitud del panel: %v",
			errors.Join(err, errAutorizacion, errDatos, errCorrelacion, errCorrelacionRef))
	}
	return puertosbolsa.InstantaneaPanelInterno{
		Esquema:  puertosbolsa.EsquemaPanelInternoBolsaV1,
		Selector: selector,
		Origen: puertosbolsa.OrigenPanelInterno{
			Revision:      "rev_0123456789abcdef",
			ActualizadaEn: instantePanelPostgreSQLPrueba.Add(-time.Minute),
		},
		PruebaLectura: puertosbolsa.PruebaLecturaPanelInterno{
			LecturaRef: "lec_0123456789abcdef", AuditoriaRef: "aud_0123456789abcdef",
			AuditoriaSecuencia: 27, DecisionRef: datos.Decision.DecisionRef,
			HuellaDecisionSHA256: datos.HuellaDecisionSHA256, CorrelacionRef: correlacionRef,
			ConfirmadaEn: instantePanelPostgreSQLPrueba.Add(time.Microsecond),
		},
		Indicadores: puertosbolsa.IndicadoresPanelInterno{
			ConvocatoriasBorrador: 2, ConvocatoriasRevision: 1,
			ConvocatoriasPendientesFirma: 1, ConvocatoriasPublicadas: 5,
			BolsasActivas: 4, BolsasSuspendidas: 1, LlamamientosPendientes: 3,
			LlamamientosEnCurso: 2, DocumentosPendientesFirma: 2, IncidenciasAbiertas: 1,
		},
		Convocatorias: []puertosbolsa.ResumenConvocatoriaPanelInterno{{
			ConvocatoriaRef: "cnv_0123456789abcdef", CategoriaClave: "auxiliar_administrativo",
			EstadoClave: "revision", PlazoCierraEn: instantePanelPostgreSQLPrueba.Add(48 * time.Hour),
			NumeroSolicitudes: 80, NumeroPendientes: 6,
		}},
		ActuacionesPendientes: []puertosbolsa.ActuacionPendientePanelInterno{{
			ActuacionRef: "act_0123456789abcdef", RecursoRef: "cnv_0123456789abcdef",
			TipoClave: "revisar_bases", EstadoClave: "pendiente", PrioridadClave: "alta",
			FechaLimite: instantePanelPostgreSQLPrueba.Add(24 * time.Hour), NumeroElementos: 1,
		}},
	}
}

func respuestaPanelPostgreSQLPrueba(
	t *testing.T,
	solicitud puertosbolsa.SolicitudConsultaPanelInterno,
) []byte {
	t.Helper()
	contenido, err := json.Marshal(instantaneaPanelPostgreSQLPrueba(t, solicitud))
	if err != nil {
		t.Fatal(err)
	}
	return contenido
}

func selectorPanelPostgreSQLPrueba() puertosbolsa.SelectorPanelInterno {
	return puertosbolsa.SelectorPanelInterno{
		Clase:            puertosbolsa.AmbitoPanelUnidad,
		OrganizacionRef:  "org_0123456789abcdef",
		UnidadGestionRef: "uni_fedcba9876543210",
	}
}

func motivoPanelPostgreSQLPrueba() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID: "catalogo_motivos_panel_bolsa", CatalogoVersion: 1,
		CatalogoHuellaSHA256: strings.Repeat("d", 64),
		EntradaClave:         "motivo_0123456789abcdef0123456789abcdef",
	}
}

func correlacionPanelPostgreSQLPrueba(
	t *testing.T,
) dominiovec.ReferenciaCorrelacionAutorizacionV2 {
	t.Helper()
	referencia, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(
		context.Background(),
		generadorCorrelacionPanelPostgreSQLPrueba{
			valor: "correlacion_0123456789abcdef0123456789abcdef",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return referencia
}
