package gobiernoconvocatorias

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestCrashTrasReservaLeaseVivoYReclamacionExpirada(t *testing.T) {
	e := nuevoEscenario(t, confirmarBien, 2, 1)
	e.diario.fallarTrasReserva = true
	_, err := e.servicio.Crear(context.Background(), e.orden)
	if !errors.Is(err, ErrOperacionBorradorIndeterminada) || e.diario.reservas != 1 ||
		e.confirmador.llamadas != 0 {
		t.Fatalf("caida tras reserva no quedo recuperable: %v", err)
	}
	_, err = e.servicio.Crear(context.Background(), e.orden)
	if !errors.Is(err, ErrOperacionBorradorEnCurso) || e.diario.reclamos != 0 ||
		e.confirmador.llamadas != 0 {
		t.Fatalf("lease vivo fue reclamado: %v", err)
	}
	e.reloj.avanzar(3 * time.Minute)
	recibo, err := e.servicio.Crear(context.Background(), e.orden)
	if err != nil || !reciboProyectadoValido(recibo, recibo.Identidad) {
		t.Fatalf("lease expirado no se recupero: %v", err)
	}
	if e.diario.reclamos != 1 || e.confirmador.efectos != 1 ||
		recibo.CercadoConfirmado < 3 || recibo.RevisionConfirmada < 4 {
		t.Fatalf("reclamacion no elevo controles: reclamos=%d rev=%d fence=%d efectos=%d",
			e.diario.reclamos, recibo.RevisionConfirmada, recibo.CercadoConfirmado,
			e.confirmador.efectos)
	}
}

func TestIndeterminadoSeReconciliaACommitSinRepetir(t *testing.T) {
	e := nuevoEscenario(t, confirmarIndeterminadoCommit, 2, 1)
	_, err := e.servicio.Crear(context.Background(), e.orden)
	if !errors.Is(err, ErrOperacionBorradorIndeterminada) || e.confirmador.efectos != 1 {
		t.Fatalf("commit sin respuesta no quedo indeterminado: %v", err)
	}
	recibo, err := e.servicio.Crear(context.Background(), e.orden)
	if err != nil || !reciboProyectadoValido(recibo, recibo.Identidad) ||
		e.confirmador.llamadas != 1 || e.confirmador.efectos != 1 || e.diario.reclamos != 0 {
		t.Fatalf("reconciliacion repitio el commit: %v", err)
	}
	if recibo.RevisionConfirmada <= 1 || recibo.CercadoConfirmado != 1 {
		t.Fatalf("commit reconciliado no conservo epoch: rev=%d fence=%d",
			recibo.RevisionConfirmada, recibo.CercadoConfirmado)
	}
}

func TestIndeterminadoRollbackSoloReclamaTrasLease(t *testing.T) {
	e := nuevoEscenario(t, confirmarIndeterminadoRollback, 2, 1)
	_, err := e.servicio.Crear(context.Background(), e.orden)
	if !errors.Is(err, ErrOperacionBorradorIndeterminada) || e.confirmador.efectos != 0 {
		t.Fatalf("rollback desconocido no quedo indeterminado: %v", err)
	}
	_, err = e.servicio.Crear(context.Background(), e.orden)
	if !errors.Is(err, ErrOperacionBorradorEnCurso) || e.diario.reclamos != 0 {
		t.Fatalf("rollback probado con lease vivo fue reclamado: %v", err)
	}
	e.reloj.avanzar(3 * time.Minute)
	e.confirmador.cambiarModo(confirmarBien)
	if _, err = e.servicio.Crear(context.Background(), e.orden); err != nil {
		t.Fatal(err)
	}
	if e.diario.reclamos != 1 || e.confirmador.efectos != 1 {
		t.Fatalf("rollback probado no produjo una unica reejecucion: reclamos=%d efectos=%d",
			e.diario.reclamos, e.confirmador.efectos)
	}
}

func TestConfirmacionObsoletaRechazaRevisionYCercado(t *testing.T) {
	e := nuevoEscenario(t, confirmarBien, 2, 1)
	if _, err := e.servicio.Crear(context.Background(), e.orden); err != nil {
		t.Fatal(err)
	}
	if e.confirmador.ultima == nil {
		t.Fatal("no se conservo orden de prueba")
	}
	obsoleta := *e.confirmador.ultima
	resultado, err := e.confirmador.ConfirmarBorrador(context.Background(), obsoleta)
	if err == nil || resultado.Estado == ResultadoDiarioConfirmado || e.confirmador.efectos != 1 {
		t.Fatalf("orden obsoleta repitio efecto: estado=%s err=%v", resultado.Estado, err)
	}
}

func TestReciboInmediatoYReplayAplicanMismoVeredictoTemporal(t *testing.T) {
	e := nuevoEscenario(t, confirmarBien, 1)
	inmediato, err := e.servicio.Crear(context.Background(), e.orden)
	if err != nil {
		t.Fatal(err)
	}
	e.reloj.avanzar(6 * time.Minute)
	replay, err := e.reiniciar(t, 2, 1).Crear(context.Background(), e.orden)
	if err != nil || !reflect.DeepEqual(inmediato, replay) {
		t.Fatalf("replay temporal divergio del inmediato: %v", err)
	}
	if !reciboProyectadoValido(replay, replay.Identidad) ||
		!replay.ConfirmadaEn.Before(replay.ArrendamientoVenceEn) ||
		!replay.ConfirmadaEn.Before(replay.Decision.ValidaHasta) ||
		!replay.ConfirmadaEn.Before(replay.SelladoMotivo.AtestacionValidaHasta) {
		t.Fatal("recibo no conserva los limites temporales del commit")
	}
}

func TestReconciliacionRechazaControlNoCreciente(t *testing.T) {
	e := nuevoEscenario(t, confirmarBien, 2, 1)
	e.diario.fallarTrasReserva = true
	_, _ = e.servicio.Crear(context.Background(), e.orden)
	consulta, err := solicitudConsultaEscenario(t, e)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := e.diario.ConsultarIdentidades(context.Background(), consulta)
	if err != nil || len(resultado.Coincidencias) != 1 {
		t.Fatal("reserva de prueba ausente")
	}
	control := resultado.Coincidencias[0]
	solicitud := SolicitudReconciliacionBorrador{
		Identidad: control.Identidad, Control: control.Resultado,
		SolicitadaEn: control.Resultado.ArrendamientoIniciaEn.Add(time.Second),
	}
	mal := ResultadoReconciliacionBorrador{
		Resultado: ResultadoOperacionDiario{
			Estado: ResultadoDiarioNoAplicado, Revision: control.Resultado.Revision + 1,
			Cercado:               control.Resultado.Cercado,
			ArrendamientoIniciaEn: control.Resultado.ArrendamientoIniciaEn,
			ArrendamientoVenceEn:  control.Resultado.ArrendamientoVenceEn,
		},
		ComprobadaEn: solicitud.SolicitadaEn, PruebaDesenlaceRef: "prueba:rollback:001",
		HuellaPruebaSHA256: huellaHexPrueba('e'),
	}
	if mal.ValidarPara(solicitud) == nil {
		t.Fatal("cierre no_aplicado sin aumentar fence fue aceptado")
	}
}

func solicitudConsultaEscenario(
	t *testing.T,
	e escenarioPrueba,
) (SolicitudConsultaIdentidadesBorrador, error) {
	t.Helper()
	intencion, err := nuevaIntencionAltaBorradorCanonica(
		e.catalogo.plantilla.Referencia, e.orden.CodigoVersionPublica, e.orden.ExpedienteRef,
		e.orden.Contenido, e.orden.MotivoCatalogo,
	)
	if err != nil {
		return SolicitudConsultaIdentidadesBorrador{}, err
	}
	solicitud, err := nuevaSolicitudDerivacionIdempotencia(e.orden.ClaveCliente, intencion, e.orden.Actor)
	if err != nil {
		return SolicitudConsultaIdentidadesBorrador{}, err
	}
	conjunto, err := e.derivador.Derivar(context.Background(), solicitud)
	if err != nil {
		return SolicitudConsultaIdentidadesBorrador{}, err
	}
	return nuevaSolicitudConsultaIdentidadesBorrador(conjunto)
}

func TestValoresNuevosBloqueanCodecsYFormato(t *testing.T) {
	secreto := "clave:hmac:convocatorias:localizador:no-filtrar"
	valores := []any{
		OrdenCrearBorrador{}, OrdenActualizarBorrador{}, PlantillaBorradorResuelta{},
		PreparacionAltaBorrador{}, IntencionBorradorCanonica{}, SolicitudDerivacionIdempotencia{},
		IdentidadOperacionDerivada{}, ConjuntoIdentidadesOperacion{},
		SolicitudConsultaIdentidadesBorrador{}, ResultadoConsultaIdentidadesBorrador{},
		ResultadoReservaDecisionBorrador{}, SolicitudReservaDecisionBorrador{},
		SolicitudReconciliacionBorrador{}, ResultadoReconciliacionBorrador{},
		SolicitudReclamacionDecisionBorrador{}, ProyeccionReciboBorrador{ReciboRef: secreto},
		SolicitudSelladoMotivoBorrador{}, SolicitudConfirmacionBorrador{}, ResultadoConfirmacionAtomica{},
	}
	for _, valor := range valores {
		if _, err := json.Marshal(valor); !errors.Is(err, ErrSerializacionDiarioProhibida) {
			t.Fatalf("%T permitio JSON: %v", valor, err)
		}
		for _, formato := range []string{"%v", "%+v", "%#v"} {
			if salida := fmt.Sprintf(formato, valor); strings.Contains(salida, secreto) {
				t.Fatalf("%T filtro un seudonimo con %s", valor, formato)
			}
		}
	}
}
