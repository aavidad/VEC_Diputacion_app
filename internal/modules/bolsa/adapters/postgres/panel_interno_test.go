package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestPanelInternoPostgreSQLConsultaFuncionEstrechaYConfirmaTrasValidar(t *testing.T) {
	solicitud := solicitudPanelPostgreSQLPrueba(t)
	respuesta := respuestaPanelPostgreSQLPrueba(t, solicitud)
	tx := &transaccionPanelPostgreSQLPrueba{
		fila: filaPanelPostgreSQLPrueba{contenido: respuesta},
	}
	iniciador := &iniciadorPanelPostgreSQLPrueba{tx: tx}
	adaptador, err := nuevaConsultaPanelInternoPostgreSQL(iniciador)
	if err != nil {
		t.Fatal(err)
	}

	resultado, err := adaptador.ConsultarPanel(context.Background(), solicitud)
	if err != nil {
		t.Fatalf("consultar panel durable: %v", err)
	}
	if _, err := resultado.ClonarValidadaPara(solicitud); err != nil {
		t.Fatalf("resultado no ligado a la solicitud: %v", err)
	}
	if len(iniciador.opciones) != 1 || iniciador.opciones[0].IsoLevel != pgx.Serializable ||
		iniciador.opciones[0].AccessMode != pgx.ReadWrite || tx.configuraciones != 1 ||
		tx.confirmaciones != 1 || tx.reversiones != 1 {
		t.Fatalf("frontera transaccional incorrecta: opciones=%+v config=%d commit=%d rollback=%d",
			iniciador.opciones, tx.configuraciones, tx.confirmaciones, tx.reversiones)
	}
	if !strings.Contains(tx.consulta, funcionConsultarPanelInternoV1) ||
		!strings.Contains(tx.consulta, "$5::text") || len(tx.argumentos) != 5 {
		t.Fatalf("contrato SQL no es estrecho: consulta=%q argumentos=%d", tx.consulta, len(tx.argumentos))
	}
	comprobarArgumentosCanonicosPanelPostgreSQL(t, solicitud, tx.argumentos)
}

func TestPanelInternoPostgreSQLRechazaJSONAmbiguoAntesDeCommit(t *testing.T) {
	solicitud := solicitudPanelPostgreSQLPrueba(t)
	valida := respuestaPanelPostgreSQLPrueba(t, solicitud)
	duplicada := append([]byte(`{"esquema":"otra",`), valida[1:]...)
	desconocida := append([]byte(`{"dni":"dato_no_permitido",`), valida[1:]...)
	candidatosGlobales := append([]byte(`{"candidatos":[],`), valida[1:]...)
	conContenidoPosterior := append(append([]byte(nil), valida...), []byte(` {}`)...)

	for nombre, contenido := range map[string][]byte{
		"clave duplicada":     duplicada,
		"campo desconocido":   desconocida,
		"listado global":      candidatosGlobales,
		"contenido posterior": conContenidoPosterior,
	} {
		t.Run(nombre, func(t *testing.T) {
			tx := &transaccionPanelPostgreSQLPrueba{
				fila: filaPanelPostgreSQLPrueba{contenido: contenido},
			}
			adaptador, err := nuevaConsultaPanelInternoPostgreSQL(
				&iniciadorPanelPostgreSQLPrueba{tx: tx},
			)
			if err != nil {
				t.Fatal(err)
			}
			_, err = adaptador.ConsultarPanel(context.Background(), solicitud)
			if !errors.Is(err, puertosbolsa.ErrResultadoPanelInternoInvalido) ||
				tx.confirmaciones != 0 || tx.reversiones != 1 {
				t.Fatalf("JSON ambiguo confirmado: error=%v commit=%d rollback=%d",
					err, tx.confirmaciones, tx.reversiones)
			}
		})
	}
}

func TestPanelInternoPostgreSQLRechazaRespuestaNoLigadaAntesDeCommit(t *testing.T) {
	solicitud := solicitudPanelPostgreSQLPrueba(t)
	base := instantaneaPanelPostgreSQLPrueba(t, solicitud)
	casos := map[string]func(*puertosbolsa.InstantaneaPanelInterno){
		"selector distinto": func(i *puertosbolsa.InstantaneaPanelInterno) {
			i.Selector.UnidadGestionRef = "uni_1111111111111111"
		},
		"decision distinta": func(i *puertosbolsa.InstantaneaPanelInterno) {
			i.PruebaLectura.DecisionRef = "decision:panel:postgresql:otra"
		},
		"correlacion distinta": func(i *puertosbolsa.InstantaneaPanelInterno) {
			i.PruebaLectura.CorrelacionRef = "correlacion_ffffffffffffffffffffffffffffffff"
		},
		"origen demo": func(i *puertosbolsa.InstantaneaPanelInterno) {
			i.Origen.Demostracion = true
		},
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			alterada := base
			alterada.Convocatorias = append([]puertosbolsa.ResumenConvocatoriaPanelInterno(nil), base.Convocatorias...)
			alterada.ActuacionesPendientes = append(
				[]puertosbolsa.ActuacionPendientePanelInterno(nil), base.ActuacionesPendientes...,
			)
			mutar(&alterada)
			contenido, err := json.Marshal(alterada)
			if err != nil {
				t.Fatal(err)
			}
			tx := &transaccionPanelPostgreSQLPrueba{
				fila: filaPanelPostgreSQLPrueba{contenido: contenido},
			}
			adaptador, err := nuevaConsultaPanelInternoPostgreSQL(
				&iniciadorPanelPostgreSQLPrueba{tx: tx},
			)
			if err != nil {
				t.Fatal(err)
			}
			_, err = adaptador.ConsultarPanel(context.Background(), solicitud)
			if !errors.Is(err, puertosbolsa.ErrResultadoPanelInternoInvalido) ||
				tx.confirmaciones != 0 || tx.reversiones != 1 {
				t.Fatalf("respuesta no ligada confirmada: error=%v commit=%d rollback=%d",
					err, tx.confirmaciones, tx.reversiones)
			}
		})
	}
}

func TestPanelInternoPostgreSQLFallaCerradoSinFuenteOSolicitud(t *testing.T) {
	if _, err := nuevaConsultaPanelInternoPostgreSQL(nil); !errors.Is(
		err,
		ErrFuentePanelInternoPostgreSQLNoDisponible,
	) {
		t.Fatalf("constructor sin fuente: %v", err)
	}
	tx := &transaccionPanelPostgreSQLPrueba{}
	adaptador, err := nuevaConsultaPanelInternoPostgreSQL(
		&iniciadorPanelPostgreSQLPrueba{tx: tx},
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = adaptador.ConsultarPanel(
		context.Background(),
		puertosbolsa.SolicitudConsultaPanelInterno{},
	)
	if !errors.Is(err, puertosbolsa.ErrConsultaPanelInternoInvalida) || tx.consulta != "" {
		t.Fatalf("solicitud invalida alcanzo PostgreSQL: %v", err)
	}
}

func comprobarArgumentosCanonicosPanelPostgreSQL(
	t *testing.T,
	solicitud puertosbolsa.SolicitudConsultaPanelInterno,
	argumentos []any,
) {
	t.Helper()
	autorizacion, _ := solicitud.Autorizacion()
	datos, _ := autorizacion.Datos()
	decisionEsperada, err := datos.RepresentacionCanonica()
	if err != nil {
		t.Fatal(err)
	}
	motivo, _ := solicitud.Motivo()
	motivoEsperado, err := dominiovec.RepresentacionCanonicaMotivoAutorizacionV2(motivo)
	if err != nil {
		t.Fatal(err)
	}
	correlacion, _ := solicitud.Correlacion()
	correlacionEsperada, _ := correlacion.ValorCanonico()
	operacion, operacionValida := argumentos[0].([]byte)
	prueba, pruebaValida := argumentos[1].([]byte)
	decision, decisionValida := argumentos[2].([]byte)
	motivoCanonico, motivoValido := argumentos[3].([]byte)
	correlacionRef, correlacionValida := argumentos[4].(string)
	if !operacionValida || !pruebaValida || !decisionValida || !motivoValido ||
		!correlacionValida || !json.Valid(operacion) || !json.Valid(prueba) ||
		!bytes.Equal(decision, decisionEsperada) || !bytes.Equal(motivoCanonico, motivoEsperado) ||
		correlacionRef != correlacionEsperada {
		t.Fatal("la funcion no recibio los canones V2 y referencias opacas exactos")
	}
	for _, prohibido := range [][]byte{
		[]byte(`"dni"`), []byte(`"nombre"`), []byte(`"correo"`),
		[]byte(`"candidato"`), []byte(`"principal_ref"`),
	} {
		if bytes.Contains(operacion, prohibido) || bytes.Contains(prueba, prohibido) {
			t.Fatalf("PII o identidad innecesaria en argumentos minimizados: %q", prohibido)
		}
	}
}
