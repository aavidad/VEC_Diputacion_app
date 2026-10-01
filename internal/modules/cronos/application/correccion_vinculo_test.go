package application

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

type lectorVinculoCRN11Prueba func(context.Context, ports.InputConsultaVinculoPropioCRN11) (ports.VinculoPropioHistoricoCRN11, error)

func (f lectorVinculoCRN11Prueba) ConsultarVinculoPropioCRN11(ctx context.Context, in ports.InputConsultaVinculoPropioCRN11) (ports.VinculoPropioHistoricoCRN11, error) {
	return f(ctx, in)
}

type repositorioVinculoCRN11Prueba struct {
	repositorioCorreccionPrueba
	orden ports.OrdenConsumoCorreccion
}

func (r *repositorioVinculoCRN11Prueba) RecuperarRecibo(ctx context.Context, clave ports.ClaveRecuperacionCorreccion, orden ports.OrdenConsumoCorreccion) (ports.ReciboCorreccion, error) {
	r.orden = orden
	return r.repositorioCorreccionPrueba.RecuperarRecibo(ctx, clave, orden)
}

func claveInicialCRN11() ports.ClaveRecuperacionCorreccion {
	return ports.ClaveRecuperacionCorreccion{SolicitudRef: "correccion:cronos:olvido_0001", ClaveOperacion: "olvido_0001", Paso: domain.PasoSolicitudCorreccion}
}

func vinculoCRN11Prueba(in ports.InputConsultaVinculoPropioCRN11, instante time.Time) ports.VinculoPropioHistoricoCRN11 {
	return ports.VinculoPropioHistoricoCRN11{
		PersonaRef: in.Actor.PersonaRef, EmpleadoRef: in.EmpleadoRef,
		VinculoRef: "vinculo:personal:historico:1", FuenteRef: "fuente:personal:1", Version: 2,
		Evidencia: ports.EvidenciaLecturaVinculoCRN11{ReciboRef: "recibo:personal:1", DecisionRef: "decision:personal:1", AuditoriaRef: "auditoria:personal:1", ConsultadaEn: instante},
	}
}

func TestCRN11RecuperaOriginalConLecturaNuevaYTransportaPruebaPorValor(t *testing.T) {
	orden := ordenCorreccionPrueba(t)
	actor, _ := orden.ContextoActor()
	instante := actor.ResueltoEn
	original := ports.ReciboCorreccion{SolicitudRef: claveInicialCRN11().SolicitudRef, ActuacionRef: "correccion:actuacion:1", ReciboRef: "recibo:cronos:1", Estado: domain.CorreccionPendienteResponsable, Version: 1, InstanteUTC: instante.Add(-time.Hour), Replay: true}
	repo := &repositorioVinculoCRN11Prueba{repositorioCorreccionPrueba: repositorioCorreccionPrueba{recuperado: &original}}
	llamadas := 0
	lector := lectorVinculoCRN11Prueba(func(_ context.Context, in ports.InputConsultaVinculoPropioCRN11) (ports.VinculoPropioHistoricoCRN11, error) {
		llamadas++
		if in.Actor.PersonaRef != actor.PersonaRef || in.EmpleadoRef != "emp_0123456789abcdefghijkl" {
			t.Fatal("consulta ajena al contexto propio")
		}
		v := vinculoCRN11Prueba(in, instante)
		v.Evidencia.DecisionRef = fmt.Sprintf("decision:personal:%d", llamadas)
		v.Evidencia.ReciboRef = fmt.Sprintf("recibo:personal:%d", llamadas)
		v.Evidencia.AuditoriaRef = fmt.Sprintf("auditoria:personal:%d", llamadas)
		// Un proveedor no puede alterar la orden retenida por el consumidor.
		in.Actor.Instantanea.Vinculos[0].Referencia = "emp_abcdefghijkl0123456789"
		return v, nil
	})
	s, err := NuevoServicioCorreccionesConVinculoHistorico(repo, relojMarcajePrueba{instante}, lector)
	if err != nil {
		t.Fatal(err)
	}
	var anterior string
	for i := 0; i < 2; i++ {
		recibo, err := s.RecuperarRecibo(context.Background(), orden, claveInicialCRN11())
		if err != nil || recibo != original || repo.recuperaciones != i+1 || llamadas != i+1 {
			t.Fatalf("recuperacion %d: recibo=%+v error=%v", i, recibo, err)
		}
		prueba, ok := repo.orden.VinculoPropioHistoricoCRN11()
		if !ok || prueba.PersonaRef != actor.PersonaRef || prueba.EmpleadoRef != actor.Instantanea.Vinculos[0].Referencia || prueba.Version != 2 || prueba.Evidencia.DecisionRef == anterior {
			t.Fatal("prueba ausente, cruzada o reutilizada")
		}
		anterior = prueba.Evidencia.DecisionRef
		prueba.EmpleadoRef = "otro"
		copia, _ := repo.orden.VinculoPropioHistoricoCRN11()
		if copia.EmpleadoRef != actor.Instantanea.Vinculos[0].Referencia {
			t.Fatal("getter comparte prueba mutable")
		}
	}
	if _, ok := orden.VinculoPropioHistoricoCRN11(); ok {
		t.Fatal("se modifico la orden original")
	}
	conservado, _ := orden.ContextoActor()
	if conservado.Instantanea.Vinculos[0].Referencia != actor.Instantanea.Vinculos[0].Referencia || repo.solicitudes != 0 || repo.actuaciones != 0 {
		t.Fatal("se modifico contexto o negocio")
	}
}

func TestCRN11RechazaFuenteIncompletaCruzadaOCaidaAntesDelRepositorio(t *testing.T) {
	casos := []struct {
		nombre  string
		alterar func(*ports.VinculoPropioHistoricoCRN11)
		err     error
	}{
		{"otra_persona", func(v *ports.VinculoPropioHistoricoCRN11) { v.PersonaRef = "per_abcdefghijkl0123456789" }, nil},
		{"otro_empleado", func(v *ports.VinculoPropioHistoricoCRN11) { v.EmpleadoRef = "emp_abcdefghijkl0123456789" }, nil},
		{"version_ausente", func(v *ports.VinculoPropioHistoricoCRN11) { v.Version = 0 }, nil},
		{"vinculo_ausente", func(v *ports.VinculoPropioHistoricoCRN11) { v.VinculoRef = " " }, nil},
		{"procedencia_ausente", func(v *ports.VinculoPropioHistoricoCRN11) { v.FuenteRef = "" }, nil},
		{"recibo_ausente", func(v *ports.VinculoPropioHistoricoCRN11) { v.Evidencia.ReciboRef = "" }, nil},
		{"decision_ausente", func(v *ports.VinculoPropioHistoricoCRN11) { v.Evidencia.DecisionRef = "" }, nil},
		{"auditoria_ausente", func(v *ports.VinculoPropioHistoricoCRN11) { v.Evidencia.AuditoriaRef = "" }, nil},
		{"instante_ausente", func(v *ports.VinculoPropioHistoricoCRN11) { v.Evidencia.ConsultadaEn = time.Time{} }, nil},
		{"instante_futuro", func(v *ports.VinculoPropioHistoricoCRN11) {
			v.Evidencia.ConsultadaEn = v.Evidencia.ConsultadaEn.Add(time.Hour)
		}, nil},
		{"denegacion", nil, ports.ErrCorreccionNoAutorizada},
		{"caida", nil, ports.ErrDependenciaNoDisponible},
		{"cancelacion", nil, context.Canceled},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			orden := ordenCorreccionPrueba(t)
			actor, _ := orden.ContextoActor()
			repo := &repositorioVinculoCRN11Prueba{}
			llamadas := 0
			lector := lectorVinculoCRN11Prueba(func(_ context.Context, in ports.InputConsultaVinculoPropioCRN11) (ports.VinculoPropioHistoricoCRN11, error) {
				llamadas++
				v := vinculoCRN11Prueba(in, actor.ResueltoEn)
				if caso.alterar != nil {
					caso.alterar(&v)
				}
				return v, caso.err
			})
			s, _ := NuevoServicioCorreccionesConVinculoHistorico(repo, relojMarcajePrueba{actor.ResueltoEn}, lector)
			recibo, err := s.RecuperarRecibo(context.Background(), orden, claveInicialCRN11())
			esperado := caso.err
			if esperado == nil {
				esperado = ports.ErrDependenciaNoDisponible
			}
			if !errors.Is(err, esperado) || recibo != (ports.ReciboCorreccion{}) || llamadas != 1 || repo.recuperaciones != 0 {
				t.Fatalf("fuente invalida llega al repo: llamadas=%d repo=%d error=%v", llamadas, repo.recuperaciones, err)
			}
		})
	}
}

func TestCRN11GuardasPreviasNoConsultanFuenteNiRepositorio(t *testing.T) {
	for _, nombre := range []string{"sin_empleado", "dos_empleados", "expirado", "clave_cruzada", "clave_invalida", "cancelado", "sin_lector"} {
		t.Run(nombre, func(t *testing.T) {
			orden := ordenCorreccionPrueba(t)
			actor, _ := orden.ContextoActor()
			instante := actor.ResueltoEn
			clave := claveInicialCRN11()
			ctx, cancelar := context.WithCancel(context.Background())
			defer cancelar()
			switch nombre {
			case "sin_empleado":
				actor.Instantanea.Vinculos = nil
				var err error
				orden, err = ports.NuevaOrdenConsumoCorreccion(actor, proveedorCorreccionVacio{})
				if err != nil {
					t.Fatal(err)
				}
			case "dos_empleados":
				segundo := actor.Instantanea.Vinculos[0]
				segundo.VinculoRef = "vin_abcdefghijkl0123456789"
				segundo.Referencia = "emp_abcdefghijkl0123456789"
				actor.Instantanea.Vinculos = append(actor.Instantanea.Vinculos, segundo)
				var err error
				orden, err = ports.NuevaOrdenConsumoCorreccion(actor, proveedorCorreccionVacio{})
				if err == nil {
					t.Fatal("contexto ambiguo admitido")
				}
			case "expirado":
				instante = actor.Instantanea.VigenteHasta
			case "clave_cruzada":
				clave.SolicitudRef = "correccion:cronos:olvido_0002"
			case "clave_invalida":
				clave.ClaveOperacion = ""
			case "cancelado":
				cancelar()
			}
			llamadas := 0
			repo := &repositorioVinculoCRN11Prueba{}
			lector := lectorVinculoCRN11Prueba(func(_ context.Context, in ports.InputConsultaVinculoPropioCRN11) (ports.VinculoPropioHistoricoCRN11, error) {
				llamadas++
				return vinculoCRN11Prueba(in, instante), nil
			})
			s, _ := NuevoServicioCorreccionesConVinculoHistorico(repo, relojMarcajePrueba{instante}, lector)
			if nombre == "sin_lector" {
				s, _ = NuevoServicioCorrecciones(repo, relojMarcajePrueba{instante})
			}
			recibo, err := s.RecuperarRecibo(ctx, orden, clave)
			if err == nil || recibo != (ports.ReciboCorreccion{}) || llamadas != 0 || repo.recuperaciones != 0 {
				t.Fatal("entrada invalida consulto una dependencia", err)
			}
		})
	}
}

func TestCRN11RevalidaVigenciaYCancelacionTrasFuenteLenta(t *testing.T) {
	for _, cancelarDurante := range []bool{false, true} {
		t.Run(fmt.Sprint(cancelarDurante), func(t *testing.T) {
			orden := ordenCorreccionPrueba(t)
			actor, _ := orden.ContextoActor()
			reloj := &relojMarcajePrueba{actor.ResueltoEn}
			repo := &repositorioVinculoCRN11Prueba{}
			ctx, cancelar := context.WithCancel(context.Background())
			defer cancelar()
			lector := lectorVinculoCRN11Prueba(func(_ context.Context, in ports.InputConsultaVinculoPropioCRN11) (ports.VinculoPropioHistoricoCRN11, error) {
				if cancelarDurante {
					cancelar()
				} else {
					reloj.instante = actor.Instantanea.VigenteHasta
				}
				return vinculoCRN11Prueba(in, actor.ResueltoEn), nil
			})
			s, _ := NuevoServicioCorreccionesConVinculoHistorico(repo, reloj, lector)
			recibo, err := s.RecuperarRecibo(ctx, orden, claveInicialCRN11())
			if !errors.Is(err, ports.ErrCorreccionNoAutorizada) || recibo != (ports.ReciboCorreccion{}) || repo.recuperaciones != 0 {
				t.Fatal("contexto vencido durante lectura alcanzo el repo", err)
			}
		})
	}
}

func TestCRN11ConstructorExigeDependencias(t *testing.T) {
	lector := lectorVinculoCRN11Prueba(func(context.Context, ports.InputConsultaVinculoPropioCRN11) (ports.VinculoPropioHistoricoCRN11, error) {
		return ports.VinculoPropioHistoricoCRN11{}, nil
	})
	for _, deps := range []struct {
		repo   ports.RepositorioCorrecciones
		reloj  ports.Reloj
		lector ports.LectorVinculoPropioHistoricoCRN11
	}{
		{nil, relojMarcajePrueba{}, lector}, {&repositorioCorreccionPrueba{}, nil, lector}, {&repositorioCorreccionPrueba{}, relojMarcajePrueba{}, nil},
	} {
		if s, err := NuevoServicioCorreccionesConVinculoHistorico(deps.repo, deps.reloj, deps.lector); s != nil || !errors.Is(err, ports.ErrDependenciaNoDisponible) {
			t.Fatal("dependencia ausente aceptada", err)
		}
	}
}

func TestCRN11VinculoEmpleadoCaducaAntesQueInstantanea(t *testing.T) {
	for _, duranteLectura := range []bool{false, true} {
		t.Run(fmt.Sprint(duranteLectura), func(t *testing.T) {
			orden := ordenCorreccionPrueba(t)
			actor, _ := orden.ContextoActor()
			caducidad := actor.ResueltoEn.Add(time.Second)
			actor.Instantanea.Vinculos[0].VigenteHasta = caducidad
			orden, err := ports.NuevaOrdenConsumoCorreccion(actor, proveedorCorreccionVacio{})
			if err != nil {
				t.Fatal(err)
			}
			reloj := &relojMarcajePrueba{actor.ResueltoEn}
			if !duranteLectura {
				reloj.instante = caducidad
			}
			if !actor.Instantanea.VigenteEn(caducidad) {
				t.Fatal("fixture no separa las vigencias")
			}
			llamadas := 0
			repo := &repositorioVinculoCRN11Prueba{}
			lector := lectorVinculoCRN11Prueba(func(_ context.Context, in ports.InputConsultaVinculoPropioCRN11) (ports.VinculoPropioHistoricoCRN11, error) {
				llamadas++
				reloj.instante = caducidad
				return vinculoCRN11Prueba(in, actor.ResueltoEn), nil
			})
			s, _ := NuevoServicioCorreccionesConVinculoHistorico(repo, reloj, lector)
			recibo, err := s.RecuperarRecibo(context.Background(), orden, claveInicialCRN11())
			esperadas := 0
			if duranteLectura {
				esperadas = 1
			}
			if !errors.Is(err, ports.ErrCorreccionNoAutorizada) || recibo != (ports.ReciboCorreccion{}) || llamadas != esperadas || repo.recuperaciones != 0 {
				t.Fatal("vinculo expirado admitido con contexto general vigente", err)
			}
		})
	}
}

var _ ports.LectorVinculoPropioHistoricoCRN11 = lectorVinculoCRN11Prueba(nil)
