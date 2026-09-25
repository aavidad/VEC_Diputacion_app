package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"
	cronosapp "vec-diputacion-granada/internal/modules/cronos/application"

	"github.com/jackc/pgx/v5"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	postgrescontratacion "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	ctapplication "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	puertosct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	altapersonal "vec-diputacion-granada/internal/modules/personal/adapters/contrataciontemporal"
	lecturapersonal "vec-diputacion-granada/internal/modules/personal/adapters/lecturaincorporacion"
)

// El doble comprueba el contrato de selección del gobierno; no suplanta
// PostgreSQL ni acredita la transacción o la instalación de AD3-30/31.
type txGobiernoContinuidadPrueba struct {
	pgx.Tx
	audienciaActual string
}

type filaGobiernoContinuidadPrueba func(...any) error

func (f filaGobiernoContinuidadPrueba) Scan(destinos ...any) error { return f(destinos...) }

func (tx *txGobiernoContinuidadPrueba) QueryRow(
	_ context.Context, sql string, args ...any,
) pgx.Row {
	return filaGobiernoContinuidadPrueba(func(destinos ...any) error {
		switch {
		case strings.Contains(sql, "count(*) FROM vec_autorizacion_atestada_v3.puntero_clave_emision"):
			*destinos[0].(*int64) = 1
			*destinos[1].(*int64) = 1
			return nil
		case strings.Contains(sql, "c.audiencia_consumo IN"):
			if len(args) != 55 {
				return errors.New("numero de audiencias de gobierno inesperado")
			}
			admitida := false
			for _, indice := range []int{0, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 54} {
				if args[indice] == tx.audienciaActual {
					admitida = true
				}
			}
			*destinos[0].(*bool) = admitida
			return nil
		default:
			return errors.New("consulta de gobierno fuera del contrato")
		}
	})
}

func TestGobiernoPostgreSQLContinuidadNominalAD330YAD331(t *testing.T) {
	audienciasPropias := []string{
		audienciaConsumoAltaContratacionTemporal,
		ports.AudienciaIntegracionLlamamientoDesarrollo,
		puertosct.AudienciaConsumoConsultaCuadroRRHHV3,
		puertosct.AudienciaConsumoConsultaDetalleRRHHV3,
		altapersonal.AudienciaAltaEjercicio,
		lecturapersonal.AudienciaV2,
		puertosct.AudienciaConfirmacionIncorporacionV2,
		postgrescontratacion.AudienciaAnotacionAdministrativaV1,
		postgrescontratacion.AudienciaCierreAdministrativoSinCese,
		ctapplication.AudienciaDespachoCorreoLlamamientoV3,
		ctapplication.AudienciaResultadoCorreoLlamamientoV3,
		ports.AudienciaCrearBorradorLlamamientoInterno,
		ports.AudienciaConsultarBorradorLlamamientoInterno,
		ports.AudienciaCambiarSituacionParticipacion,
		ports.AudienciaRegistrarContactoParticipacion,
		ports.AudienciaConsultarContactoParticipacion,
		ports.AudienciaRegistrarDatosContactoParticipacion,
		ports.AudienciaEmitirLlamamiento,
		audienciaConsumoPersonalDietasDesarrollo,
		audienciaConsumoCrearDietasDesarrollo,
		audienciaConsumoConsultarDietasDesarrollo,
		audienciaConsumoEditarDietasDesarrollo,
		audienciaConsumoBorrarDietasDesarrollo,
		audienciaConsumoEnviarDietasDesarrollo,
		audienciaConsumoDocumentoDietasDesarrollo,
		audienciaConsumoConsultarAsignacionDietas,
		audienciaConsumoRegistrarAsignacionDietas,
		audienciaConsumoCorregirAsignacionDietas,
		audienciaConsumoCorregirGrupoDietas,
		audienciaConsumoRevisarDietas,
		audienciaConsumoAutorizarDietas,
		audienciaConsumoLiquidarDietas,
		audienciaConsumoFiscalizarDietas,
		audienciaConsumoBandejaRevisionDietas,
		audienciaConsumoBandejaAutorizacionDietas,
		audienciaConsumoBandejaLiquidacionDietas,
		audienciaConsumoBandejaFiscalizacionDietas,
		audienciaConsumoRevisorDocumentoDietas,
		cronosapp.AudienciaMarcajePropio,
		cronosapp.AudienciaDisponibilidadMarcajeRemoto,
		cronosapp.AudienciaRecuperacionMarcajeRemoto,
		cronosapp.AudienciaConsultaSaldoPropio,
		cronosapp.AudienciaConsultaMovimientosPropios,
		cronosapp.AudienciaSolicitudCorreccionPropia,
		cronosapp.AudienciaConsultaPermisosPropios,
		cronosapp.AudienciaSolicitudPermisoPropio,
	}
	// Tras publicar B2, su última audiencia es la del puntero vigente: el
	// siguiente arranque debe reconocer el gobierno como propio.
	for _, d := range DescriptoresCapacidadPersonalB2V3Desarrollo() {
		audienciasPropias = append(audienciasPropias, d.Audiencia)
	}
	for _, audiencia := range audienciasPropias {
		t.Run(audiencia, func(t *testing.T) {
			propio, err := gobiernoActualPostgreSQLContratacionTemporalDesarrolloEsPropio(
				context.Background(), &txGobiernoContinuidadPrueba{audienciaActual: audiencia},
			)
			if err != nil || !propio {
				t.Fatalf("audiencia propia rechazada: propio=%t err=%v", propio, err)
			}
			if !audienciaConsumoGobiernoPostgreSQLContratacionTemporalDesarrolloEsPropia(audiencia) {
				t.Fatal("publicador no reconoce una audiencia propia")
			}
		})
	}

	const audienciaAjena = "vec_contratacion_temporal.ajena.v1"
	propio, err := gobiernoActualPostgreSQLContratacionTemporalDesarrolloEsPropio(
		context.Background(), &txGobiernoContinuidadPrueba{audienciaActual: audienciaAjena},
	)
	if err != nil || propio {
		t.Fatalf("audiencia ajena aceptada: propio=%t err=%v", propio, err)
	}
	if audienciaConsumoGobiernoPostgreSQLContratacionTemporalDesarrolloEsPropia(audienciaAjena) {
		t.Fatal("publicador acepta una audiencia ajena")
	}
}

// Toda audiencia que se añade al catálogo común debe poder publicarla el
// gobierno de CT; si no, el arranque con ese módulo activo cae entero.
func TestAudienciasDietasPublicablesPorElGobiernoCT(t *testing.T) {
	for _, d := range descriptoresMaterialDietasDesarrollo() {
		if !audienciaConsumoGobiernoPostgreSQLContratacionTemporalDesarrolloEsPropia(d.Audiencia) {
			t.Fatalf("audiencia de Dietas no publicable por CT: %s", d.Audiencia)
		}
	}
}

func TestAudienciasCronosPublicablesPorElGobiernoCT(t *testing.T) {
	descriptores := descriptoresMaterialCronosDesarrollo()
	audiencias := audienciasCronosEmpleadoDesarrollo()
	if len(descriptores) != len(audiencias) {
		t.Fatal("descriptores y audiencias de Cronos divergen")
	}
	for i, d := range descriptores {
		if d.Audiencia != audiencias[i] || !audienciaConsumoGobiernoPostgreSQLContratacionTemporalDesarrolloEsPropia(d.Audiencia) {
			t.Fatalf("audiencia de Cronos no publicable por CT: %s", d.Audiencia)
		}
	}
	if _, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(append(descriptoresMaterialDietasDesarrollo(), descriptores...)); err != nil {
		t.Fatal("Cronos colisiona con Dietas en el catálogo común", err)
	}
}
