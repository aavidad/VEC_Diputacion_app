package postgres

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func TestResolutorCandidatoExternoUsaFachadaCincoArgumentos(t *testing.T) {
	solicitud, fila := solicitudYFilaContextoActorV2(t)
	tx := &txContextoActorDoble{filas: []pgx.Row{fila}}
	pool := &poolContextoActorDoble{transacciones: []*txContextoActorDoble{tx}}
	adaptador, err := nuevoResolutorRegistroContextoActorExternoPostgreSQLV1(
		pool, bytes.NewReader(bytes.Repeat([]byte{0x55}, bytesAleatoriosReferenciaContextoActorV2)),
	)
	if err != nil {
		t.Fatal(err)
	}
	confirmacion, err := adaptador.ResolverYRegistrarContextoActorV2(context.Background(), solicitud)
	if err != nil || confirmacion.ValidarParaProductiva(solicitud) != nil {
		t.Fatalf("confirmacion candidata denegada: %v", err)
	}
	if pool.llamadas != 1 || pool.opciones[0].IsoLevel != pgx.Serializable || tx.commits != 1 ||
		len(tx.consultas) != 1 || !strings.Contains(tx.consultas[0], "resolver_contexto_candidato_externo_v1") ||
		len(tx.argumentos[0]) != 5 || tx.argumentos[0][0] != solicitud.OperacionRef ||
		tx.argumentos[0][2] != solicitud.Contexto.Cuenta.CuentaRef ||
		tx.argumentos[0][3] != solicitud.Contexto.PerfilActivoRef ||
		tx.argumentos[0][4] != solicitud.SolicitadoEn {
		t.Fatal("la fachada candidata no recibio exactamente los cinco argumentos")
	}
}

func TestResolutorCandidatoExternoRechazaEntradasFueraDeContratoAntesDeSQL(t *testing.T) {
	solicitud, _ := solicitudYFilaContextoActorV2(t)
	casos := []struct {
		nombre string
		mutar  func(*ports.SolicitudResolucionRegistroContextoActorV2)
	}{
		{"metodo", func(s *ports.SolicitudResolucionRegistroContextoActorV2) {
			s.Contexto.Cuenta.Metodo = domain.AuthMethodDNIe
		}},
		{"garantia", func(s *ports.SolicitudResolucionRegistroContextoActorV2) {
			s.Contexto.Cuenta.Garantia = domain.AuthAssuranceSubstantial
		}},
		{"proyeccion", func(s *ports.SolicitudResolucionRegistroContextoActorV2) {
			var err error
			s.Proyecciones, err = domain.NuevoAlcanceProyeccionesContextoActor(domain.ProyeccionContextoActorEmpleado)
			if err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			peticion := solicitud
			caso.mutar(&peticion)
			pool := &poolContextoActorDoble{}
			adaptador, _ := nuevoResolutorRegistroContextoActorExternoPostgreSQLV1(pool, bytes.NewReader(nil))
			_, err := adaptador.ResolverYRegistrarContextoActorV2(context.Background(), peticion)
			if !errors.Is(err, ports.ErrResolutorRegistroContextoActorNoDisponible) || pool.llamadas != 0 {
				t.Fatalf("entrada ajena aceptada o consultada: %v", err)
			}
		})
	}
}

func TestResolutorCandidatoExternoReconciliaMismaFila(t *testing.T) {
	solicitud, fila := solicitudYFilaContextoActorV2(t)
	primera := &txContextoActorDoble{filas: []pgx.Row{fila}, errCommit: errors.New("commit incierto")}
	segunda := &txContextoActorDoble{filas: []pgx.Row{filaContextoActorDoble{valores: append([]any(nil), fila.valores...)}}}
	pool := &poolContextoActorDoble{transacciones: []*txContextoActorDoble{primera, segunda}}
	adaptador, _ := nuevoResolutorRegistroContextoActorExternoPostgreSQLV1(
		pool, bytes.NewReader(bytes.Repeat([]byte{0x55}, bytesAleatoriosReferenciaContextoActorV2)),
	)
	if _, err := adaptador.ResolverYRegistrarContextoActorV2(context.Background(), solicitud); err != nil {
		t.Fatal(err)
	}
	if pool.llamadas != 2 || pool.opciones[1].IsoLevel != pgx.ReadCommitted ||
		!strings.Contains(segunda.consultas[0], "reconciliar_contexto_candidato_externo_v1") ||
		len(segunda.argumentos[0]) != 5 || primera.argumentos[0][1] != segunda.argumentos[0][1] {
		t.Fatal("la reconciliacion no conservo recibo y cinco argumentos")
	}
}

func TestResolutorCandidatoExternoDeniegaCanonSinUnicoVinculoCandidato(t *testing.T) {
	solicitud, fila := solicitudYFilaContextoActorV2Alcance(t, true)
	respuesta := respuestaContextoActorPostgreSQL{
		operacionRef: fila.valores[0].(string), reciboRef: fila.valores[1].(string),
		representacion: fila.valores[2].([]byte), huella: fila.valores[3].(string),
		manifiesto: fila.valores[4].([]byte), huellaManifiesto: fila.valores[5].(string),
		autoridadEfectiva: fila.valores[6].(string), resueltoEn: fila.valores[7].(time.Time),
	}
	if _, err := confirmarRespuestaContextoActor(solicitud, respuesta); err != nil {
		t.Fatalf("la fila de prueba V2 debe ser valida antes del filtro externo: %v", err)
	}
	_, err := confirmarRespuestaCandidatoExterno(solicitud, respuesta)
	if !errors.Is(err, ports.ErrResolutorRegistroContextoActorNoDisponible) {
		t.Fatalf("canon con empleado aceptado: %v", err)
	}
}
