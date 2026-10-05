package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type operadorDatosContactoAuditadoPrueba struct {
	leidos    ports.DatosContactoParticipacionLeidos
	err       error
	cerrada   bool
	consultas int
	registros int
	cancelar  func()
}

func (o *operadorDatosContactoAuditadoPrueba) Registrar(context.Context, ports.SolicitudRegistrarDatosContactoParticipacion) (ports.RegistroDatosContactoParticipacion, error) {
	o.registros++
	return ports.RegistroDatosContactoParticipacion{ReciboRef: "recibo:datos-contacto:prueba"}, nil
}

func (o *operadorDatosContactoAuditadoPrueba) Consultar(context.Context, ports.SolicitudConsultarDatosContactoParticipacion) (ports.DatosContactoParticipacionLeidos, error) {
	o.consultas++
	o.cerrada = true
	if o.cancelar != nil {
		o.cancelar()
	}
	return o.leidos, o.err
}

type registradorDatosContactoAuditadoPrueba struct {
	t        *testing.T
	operador *operadorDatosContactoAuditadoPrueba
	ordenes  []vecports.DatosOrdenIntentoAuditoria
	err      error
}

func (r *registradorDatosContactoAuditadoPrueba) AppendIntentoAuditoria(ctx context.Context, o vecports.OrdenIntentoAuditoria) (vecports.AcuseIntentoAuditoria, error) {
	r.t.Helper()
	if !r.operador.cerrada || ctx.Err() != nil {
		r.t.Fatal("registro antes del cierre o cancelado por HTTP")
	}
	d, err := o.Datos()
	if err != nil {
		r.t.Fatal(err)
	}
	r.ordenes = append(r.ordenes, d)
	if r.err != nil {
		return vecports.AcuseIntentoAuditoria{}, r.err
	}
	return vecports.AcuseIntentoAuditoria{AuditoriaRef: "aud_datos_contacto_sintetica", Secuencia: 1, HuellaSHA256: strings.Repeat("b", 64),
		CorrelacionRef: d.Datos.CorrelacionRef, RegistradaEn: time.Date(2026, 10, 5, 1, 0, 0, 0, time.UTC)}, nil
}

func leidosCompletosAuditadosPrueba() ports.DatosContactoParticipacionLeidos {
	datos := dominiobolsa.DatosContactoParticipacion{ParticipacionRef: "participacion:b4", Correo: correoContactoAuditadoPrueba, Telefono1: "600123456"}
	return ports.DatosContactoParticipacionLeidos{ParticipacionRef: "participacion:b4", Version: 1, Datos: datos, Enmascarados: datos.Enmascarados(),
		AuditoriaRef: "aud_v3_0123456789abcdef0123456789abcdef", DecisionRef: "decision:prueba"}
}

func TestDatosContactoAuditadosConservanPositivoYEnmascaradoSinRegistro(t *testing.T) {
	ahora := time.Date(2026, 10, 5, 1, 0, 0, 0, time.UTC)
	o := &operadorDatosContactoAuditadoPrueba{leidos: leidosCompletosAuditadosPrueba()}
	r := &registradorDatosContactoAuditadoPrueba{t: t, operador: o}
	s, err := NuevosDatosContactoRRHHAuditados(o, r, "vec-bolsa-prueba")
	if err != nil {
		t.Fatal(err)
	}
	leidos, err := s.Consultar(context.Background(), consultaCompletaDatosContactoPrueba(t, ahora))
	if err != nil || leidos.Datos.Correo != correoContactoAuditadoPrueba || len(r.ordenes) != 0 {
		t.Fatalf("positivo alterado o registrado: %+v err=%v ordenes=%d", leidos, err, len(r.ordenes))
	}
	// La enmascarada y el registro se delegan aunque fallen: sin acción propia.
	o.err = core.ErrAutorizacionDenegada
	enmascarada := consultaCompletaDatosContactoPrueba(t, ahora)
	enmascarada.Completo = false
	if _, err := s.Consultar(context.Background(), enmascarada); !errors.Is(err, core.ErrAutorizacionDenegada) || len(r.ordenes) != 0 {
		t.Fatalf("enmascarada registró intento: err=%v ordenes=%d", err, len(r.ordenes))
	}
	if _, err := s.Registrar(context.Background(), ports.SolicitudRegistrarDatosContactoParticipacion{}); err != nil || o.registros != 1 || len(r.ordenes) != 0 {
		t.Fatalf("registro alterado: err=%v", err)
	}
}

func TestDatosContactoAuditadosRegistranDenegacionYErrorSinDatos(t *testing.T) {
	ahora := time.Date(2026, 10, 5, 1, 0, 0, 0, time.UTC)
	casos := []struct {
		nombre    string
		err       error
		leidos    ports.DatosContactoParticipacionLeidos
		resultado core.ResultadoIntentoAuditoria
	}{
		{"denegada", core.ErrAutorizacionDenegada, ports.DatosContactoParticipacionLeidos{}, core.ResultadoIntentoAuditoriaDenegado},
		{"denegada envuelta", fmt.Errorf("pdp: %w", core.ErrPermissionDenied), ports.DatosContactoParticipacionLeidos{}, core.ResultadoIntentoAuditoriaDenegado},
		{"indisponible", ErrRegistroDatosContactoParticipacionNoDisponible, ports.DatosContactoParticipacionLeidos{}, core.ResultadoIntentoAuditoriaError},
		{"sin datos", ports.ErrDatosContactoParticipacionNoEncontrados, ports.DatosContactoParticipacionLeidos{}, core.ResultadoIntentoAuditoriaError},
		{"positivo sin acuse", nil, func() ports.DatosContactoParticipacionLeidos {
			l := leidosCompletosAuditadosPrueba()
			l.AuditoriaRef = ""
			return l
		}(), core.ResultadoIntentoAuditoriaError},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			o := &operadorDatosContactoAuditadoPrueba{err: caso.err, leidos: caso.leidos}
			r := &registradorDatosContactoAuditadoPrueba{t: t, operador: o}
			s, err := NuevosDatosContactoRRHHAuditados(o, r, "vec-bolsa-prueba")
			if err != nil {
				t.Fatal(err)
			}
			q := consultaCompletaDatosContactoPrueba(t, ahora)
			leidos, err := s.Consultar(context.Background(), q)
			acuse, confirmado := AcuseConsultaDatosContactoFallida(err)
			if err == nil || !confirmado || acuse.AuditoriaRef != "aud_datos_contacto_sintetica" || leidos.Datos.Correo != "" || len(r.ordenes) != 1 {
				t.Fatalf("fallo no auditado: err=%v acuse=%+v leidos=%+v", err, acuse, leidos)
			}
			if caso.err != nil && !errors.Is(err, caso.err) {
				t.Fatalf("la causa se perdió: %v", err)
			}
			d := r.ordenes[0].Datos
			correlacion, _ := q.Correlacion.ValorCanonico()
			if d.Accion != ports.AccionConsultarDatosContactoParticipacion || d.FinalidadRef != ports.FinalidadConsultarDatosContactoParticipacion ||
				d.RecursoRef != "participacion:b4" || d.ModuloID != ports.ModuloSituacionParticipacion || d.Resultado != caso.resultado ||
				d.Proceso != "vec-bolsa-prueba" || d.Canal != string(core.SuperficieAutenticacionInternaCorporativaV1) || d.CorrelacionRef != correlacion {
				t.Fatalf("orden AD169 inexacta: %+v", d)
			}
			if strings.Contains(fmt.Sprintf("%+v", r.ordenes[0]), correoContactoAuditadoPrueba) || strings.Contains(fmt.Sprintf("%+v", r.ordenes[0]), "600123456") {
				t.Fatal("la orden AD169 contiene datos de contacto")
			}
		})
	}
}

func TestDatosContactoAuditadosCierranSinAcuseYConCancelacion(t *testing.T) {
	ahora := time.Date(2026, 10, 5, 1, 0, 0, 0, time.UTC)
	// Sin registrador disponible el fallo no se presenta como auditado.
	o := &operadorDatosContactoAuditadoPrueba{err: core.ErrAutorizacionDenegada}
	r := &registradorDatosContactoAuditadoPrueba{t: t, operador: o, err: errors.New("registrador caído")}
	s, _ := NuevosDatosContactoRRHHAuditados(o, r, "vec-bolsa-prueba")
	_, err := s.Consultar(context.Background(), consultaCompletaDatosContactoPrueba(t, ahora))
	if _, confirmado := AcuseConsultaDatosContactoFallida(err); confirmado || !errors.Is(err, vecports.ErrIntentoAuditoriaNoDisponible) {
		t.Fatalf("fallo de registro presentado como auditado: %v", err)
	}
	// Un positivo cuyo contexto se cancela durante la lectura no se entrega y
	// el intento se registra fuera del plazo HTTP.
	ctx, cancelar := context.WithCancel(context.Background())
	o = &operadorDatosContactoAuditadoPrueba{leidos: leidosCompletosAuditadosPrueba(), cancelar: cancelar}
	r = &registradorDatosContactoAuditadoPrueba{t: t, operador: o}
	s, _ = NuevosDatosContactoRRHHAuditados(o, r, "vec-bolsa-prueba")
	leidos, err := s.Consultar(ctx, consultaCompletaDatosContactoPrueba(t, ahora))
	if !errors.Is(err, context.Canceled) || leidos.Datos.Correo != "" || len(r.ordenes) != 1 || r.ordenes[0].Datos.Resultado != core.ResultadoIntentoAuditoriaError {
		t.Fatalf("cancelación: err=%v leidos=%+v ordenes=%d", err, leidos, len(r.ordenes))
	}
	// Consulta completa incoherente: ni lectura ni registro.
	o = &operadorDatosContactoAuditadoPrueba{}
	r = &registradorDatosContactoAuditadoPrueba{t: t, operador: o}
	s, _ = NuevosDatosContactoRRHHAuditados(o, r, "vec-bolsa-prueba")
	q := consultaCompletaDatosContactoPrueba(t, ahora)
	q.MotivoAutorizacion = core.ReferenciaEntradaCatalogo{}
	if _, err := s.Consultar(context.Background(), q); err == nil || o.consultas != 0 || len(r.ordenes) != 0 {
		t.Fatalf("consulta incoherente: err=%v consultas=%d", err, o.consultas)
	}
	if _, err := NuevosDatosContactoRRHHAuditados(o, r, "Proceso Inválido"); err == nil {
		t.Fatal("proceso inválido admitido")
	}
}
