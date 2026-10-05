package mibolsa

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type historialLecturaDoble struct {
	error     error
	terminado bool
}

func (d *historialLecturaDoble) ConsultarHistorial(context.Context, Orden, int) (bolsa.PaginaHistorialMiBolsa, error) {
	d.terminado = true
	return bolsa.PaginaHistorialMiBolsa{Items: []bolsa.HechoHistorialMiBolsa{{Bolsa: "bolsa:privada"}}}, d.error
}

type registradorLecturasDoble struct {
	t             *testing.T
	ordenes       []ports.OrdenIntentoAuditoria
	antes         func()
	err           error
	acuseInvalido bool
}

func (d *registradorLecturasDoble) AppendIntentoAuditoria(_ context.Context, orden ports.OrdenIntentoAuditoria) (ports.AcuseIntentoAuditoria, error) {
	d.t.Helper()
	if d.antes != nil {
		d.antes()
	}
	d.ordenes = append(d.ordenes, orden)
	datos, err := orden.Datos()
	if err != nil {
		d.t.Fatal(err)
	}
	if d.err != nil {
		return ports.AcuseIntentoAuditoria{}, d.err
	}
	if d.acuseInvalido {
		return ports.AcuseIntentoAuditoria{}, nil
	}
	return ports.AcuseIntentoAuditoria{AuditoriaRef: "aud_lectura_sintetica", Secuencia: 1,
		HuellaSHA256: strings.Repeat("a", 64), CorrelacionRef: datos.Datos.CorrelacionRef,
		RegistradaEn: time.Date(2026, 10, 4, 1, 0, 0, 0, time.UTC)}, nil
}

func TestLecturasPropiasAuditadasConservanAccesoConfirmado(t *testing.T) {
	e := nuevoEntorno(t)
	r := &registradorLecturasDoble{t: t}
	s, err := NuevasLecturasAuditadas(e.servicio, &historialLecturaDoble{}, r, "vec-bolsa-prueba")
	exigir(t, err)
	resultado, err := s.Consultar(context.Background(), e.orden)
	exigir(t, err)
	if len(resultado.Participaciones) != 1 || e.repositorio.llamadas != 1 || len(r.ordenes) != 0 {
		t.Fatal("el acceso confirmado cambió o recibió un intento fallido")
	}
}

func TestLecturasPropiasAuditanDenegacionYFalloTrasRetorno(t *testing.T) {
	for _, caso := range []struct {
		nombre    string
		cambiar   func(*entornoMiBolsa)
		resultado domain.ResultadoIntentoAuditoria
	}{
		{"sin concesion", func(e *entornoMiBolsa) {
			e.fuente.instantanea.VersionRol.Concesiones[0].Accion = "bolsa.otra.consultar"
		}, domain.ResultadoIntentoAuditoriaDenegado},
		{"fuente indisponible", func(e *entornoMiBolsa) { e.fuente.err = ports.ErrFuenteAutorizacionNoDisponible }, domain.ResultadoIntentoAuditoriaError},
		{"repositorio indisponible", func(e *entornoMiBolsa) { e.repositorio.err = bolsa.ErrMaterialMiBolsaNoDisponible }, domain.ResultadoIntentoAuditoriaError},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			e := nuevoEntorno(t)
			caso.cambiar(e)
			r := &registradorLecturasDoble{t: t}
			s, err := NuevasLecturasAuditadas(e.servicio, &historialLecturaDoble{}, r, "vec-bolsa-prueba")
			exigir(t, err)
			resultado, err := s.Consultar(context.Background(), e.orden)
			if err == nil || len(resultado.Participaciones) != 0 || len(r.ordenes) != 1 {
				t.Fatal("fallo sin registro o con datos")
			}
			if _, ok := AcuseLecturaFallida(err); !ok {
				t.Fatal("no conserva acuse confirmado")
			}
			datos, err := r.ordenes[0].Datos()
			exigir(t, err)
			if datos.Datos.Resultado != caso.resultado || datos.Datos.Accion != bolsa.AccionConsultarMiBolsa ||
				datos.Datos.RecursoRef != recursoIntentoMiBolsa(referenciaServicioContextoActorPrueba("can_", "c")) ||
				datos.Datos.Motivo != e.orden.Motivo || datos.Datos.Canal != "externa_personal" ||
				datos.ResultadoContexto.RegistroContextoRef != e.orden.ResultadoContexto.RegistroContextoRef {
				t.Fatal("sobre nominal sustituido")
			}
		})
	}
}

func TestHistorialAuditaTrasCerrarServicioYNoDevuelveFilasFallidas(t *testing.T) {
	e := nuevoEntorno(t)
	h := &historialLecturaDoble{error: domain.ErrAutorizacionDenegada}
	r := &registradorLecturasDoble{t: t, antes: func() {
		if !h.terminado {
			t.Fatal("auditoría antes de retorno del servicio")
		}
	}}
	s, err := NuevasLecturasAuditadas(e.servicio, h, r, "vec-bolsa-prueba")
	exigir(t, err)
	resultado, err := s.ConsultarHistorial(context.Background(), e.orden, 1)
	if !errors.Is(err, domain.ErrAutorizacionDenegada) || len(resultado.Items) != 0 || len(r.ordenes) != 1 {
		t.Fatal("denegación de historial sin registro o con datos")
	}
	datos, err := r.ordenes[0].Datos()
	exigir(t, err)
	if datos.Datos.Accion != bolsa.AccionConsultarHistorialPropio || datos.Datos.FinalidadRef != bolsa.FinalidadHistorialMiBolsa {
		t.Fatal("acción de historial inexacta")
	}
}

func TestFalloRegistroOAcuseInvalidoCierraSinDatosNiAcuse(t *testing.T) {
	for _, fallo := range []bool{false, true} {
		e := nuevoEntorno(t)
		e.repositorio.err = domain.ErrAutorizacionDenegada
		h := &historialLecturaDoble{error: domain.ErrAutorizacionDenegada}
		r := &registradorLecturasDoble{t: t, acuseInvalido: fallo}
		if !fallo {
			r.err = errors.New("fallo privado del adaptador")
		}
		s, err := NuevasLecturasAuditadas(e.servicio, h, r, "vec-bolsa-prueba")
		exigir(t, err)
		resultado, err := s.Consultar(context.Background(), e.orden)
		if !errors.Is(err, bolsa.ErrMaterialMiBolsaNoDisponible) || !errors.Is(err, ports.ErrIntentoAuditoriaNoDisponible) ||
			errors.Is(err, domain.ErrAutorizacionDenegada) || strings.Contains(err.Error(), "privado") || len(resultado.Participaciones) != 0 {
			t.Fatal("fallo auditoría abierto o causa cruda")
		}
		if _, ok := AcuseLecturaFallida(err); ok {
			t.Fatal("acuse falso")
		}
		p, err := s.ConsultarHistorial(context.Background(), e.orden, 1)
		if !errors.Is(err, bolsa.ErrHistorialMiBolsaNoDisponible) || len(p.Items) != 0 {
			t.Fatal("historial abierto")
		}
	}
}

func TestLecturaCanceladaAuditaIdentidadHistoricaSinInventarla(t *testing.T) {
	e := nuevoEntorno(t)
	r := &registradorLecturasDoble{t: t}
	s, err := NuevasLecturasAuditadas(e.servicio, &historialLecturaDoble{}, r, "vec-bolsa-prueba")
	exigir(t, err)
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	_, err = s.Consultar(ctx, e.orden)
	if !errors.Is(err, context.Canceled) || len(r.ordenes) != 1 {
		t.Fatal("cancelación perdió auditoría nominal")
	}
	e.orden.ResultadoContexto.HuellaSHA256 = "alterada"
	_, err = s.Consultar(ctx, e.orden)
	if !errors.Is(err, ports.ErrIntentoAuditoriaNoDisponible) || len(r.ordenes) != 1 {
		t.Fatal("se fabricó identidad o acuse")
	}
}

// Las referencias reales de candidato son base64url con mayúsculas; la
// auditoría común (Go y SQL de AD169) sólo admite minúsculas.
func TestRecursoIntentoMiBolsaAdmiteReferenciasRealesDeCandidato(t *testing.T) {
	candidato := "can_5RckYrhrUeIrAYLjLTAkqnoyUdzdmqMtL0YX-41TK9c"
	ref := recursoIntentoMiBolsa(candidato)
	if !regexp.MustCompile(`^[a-z0-9][a-z0-9._:-]{0,199}$`).MatchString(ref) || ref != recursoIntentoMiBolsa(candidato) ||
		ref == recursoIntentoMiBolsa("can_5rckyrhrueirayljltakqnoyudzdmqmtl0yx-41tk9c") || strings.Contains(ref, candidato) {
		t.Fatalf("referencia de recurso no admitida: %q", ref)
	}
	datos := domain.DatosIntentoAuditoria{Accion: bolsa.AccionConsultarMiBolsa, ModuloID: bolsa.ModuloMiBolsa, RecursoRef: ref,
		FinalidadRef: bolsa.FinalidadMiBolsa, Resultado: domain.ResultadoIntentoAuditoriaDenegado, Motivo: nuevoEntorno(t).orden.Motivo,
		Proceso: "vec-portal-personal", Canal: "externa_personal", CorrelacionRef: "correlacion_00000000000000000000000000000000"}
	if err := datos.Validar(); err != nil {
		t.Fatal(err)
	}
}
