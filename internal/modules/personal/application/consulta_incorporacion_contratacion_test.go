package application

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
)

type fuenteIncorporacionCTPrueba struct {
	resultado ports.ResultadoFichaEmpleadoB2
	err       error
	llamadas  int
	ultima    domain.SolicitudFichaEmpleadoB2
	antes     func(context.Context)
}

func (f *fuenteIncorporacionCTPrueba) ConsultarFicha(ctx context.Context, s domain.SolicitudFichaEmpleadoB2) (ports.ResultadoFichaEmpleadoB2, error) {
	f.llamadas++
	f.ultima = s
	if f.antes != nil {
		f.antes(ctx)
	}
	return f.resultado, f.err
}

func incorporacionCTPrueba(t *testing.T) (ports.SolicitudHechosIncorporacionCT, *fuenteIncorporacionCTPrueba) {
	t.Helper()
	actor := solicitudP(t).Actor
	corte := domain.CorteEmpleadoB2{VigenteEn: "2026-09-25", ConocidoEn: time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)}
	x := ports.SeleccionHechosIncorporacionCT{OrganismoRef: "organismo:dipgra", PersonaRef: "per_" + strings.Repeat("p", 24), EmpleadoRef: "emp_" + strings.Repeat("e", 24), RelacionRef: "rel_" + strings.Repeat("r", 24), OcupacionRef: "ocupacion:uno", VersionEmpleado: 3, VersionRelacion: 2, VersionOcupacion: 1, UnidadRef: "unidad:uno", PuestoRef: "puesto:uno", PlazaRef: "plaza:uno", Corte: corte}
	traza := domain.TrazaEmpleadoB2{Desde: "2026-09-01", Hasta: "2026-10-01", RegistradaEn: corte.ConocidoEn, Version: 2, ActoRef: "acto:uno", FuenteRef: "fuente:personal", FuenteVersion: 1}
	snapshot := func(tipo, ref string) *domain.SnapshotEntradaCatalogoEmpleadoB2 {
		return &domain.SnapshotEntradaCatalogoEmpleadoB2{OrganismoRef: x.OrganismoRef, Tipo: tipo, Ref: ref, Version: 1, Revision: 1, Denominacion: "Entrada sintética", HuellaSHA256: strings.Repeat("a", 64), VigenteDesde: "2026-01-01", Estado: "publicada"}
	}
	relacion := domain.RelacionRegistroEmpleadoB2{RelacionRef: x.RelacionRef, OrganismoRef: x.OrganismoRef, UnidadRef: x.UnidadRef, RegimenRef: "regimen:uno", ModalidadRef: "modalidad:uno", Estado: "vigente", Traza: traza, CatalogoSnapshot: domain.SnapshotCatalogoEmpleadoB2{Regimen: snapshot("regimen", "regimen:uno"), Modalidad: snapshot("modalidad", "modalidad:uno")}}
	traza.Version = 1
	ocupacion := domain.OcupacionEmpleadoB2{OcupacionRef: x.OcupacionRef, RelacionRef: x.RelacionRef, UnidadRef: x.UnidadRef, PlazaRef: x.PlazaRef, PuestoRef: x.PuestoRef, ModalidadRef: "modalidad:uno", Clase: "temporal", Estado: "vigente", Traza: traza, CatalogoSnapshot: domain.SnapshotCatalogoEmpleadoB2{Modalidad: snapshot("modalidad", "modalidad:uno")}}
	f := domain.FichaEmpleadoB2{EmpleadoRef: x.EmpleadoRef, PersonaRef: x.PersonaRef, OrganismoRef: x.OrganismoRef, Corte: corte, Version: x.VersionEmpleado, Relaciones: []domain.RelacionRegistroEmpleadoB2{relacion}, Ocupaciones: []domain.OcupacionEmpleadoB2{ocupacion}}
	e := ports.EvidenciaRegistroEmpleadoB2{ReciboRef: "recibo:lectura", DecisionRef: "decision:lectura", AuditoriaRef: "auditoria:lectura", EfectoRef: x.EmpleadoRef, ConsumoHuellaSHA256: strings.Repeat("a", 64), ConsultadaEn: corte.ConocidoEn}
	return ports.SolicitudHechosIncorporacionCT{Seleccion: x, Actor: actor}, &fuenteIncorporacionCTPrueba{resultado: ports.ResultadoFichaEmpleadoB2{Ficha: f, Evidencia: e}}
}

func TestConsultaIncorporacionCTHechosExactosYLecturaEnCadaRecuperacion(t *testing.T) {
	s, f := incorporacionCTPrueba(t)
	c, err := NuevoServicioConsultaIncorporacionCT(f)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		r, err := c.ConsultarHechosIncorporacionCT(context.Background(), s)
		if err != nil {
			t.Fatal(err)
		}
		if r.Esquema != EsquemaHechosIncorporacionCT || r.Seleccion != s.Seleccion || r.Ocupacion != f.resultado.Ficha.Ocupaciones[0].Traza || r.Relacion != f.resultado.Ficha.Relaciones[0].Traza || r.Evidencia != f.resultado.Evidencia || r.FirmaOficial || r.EficaciaAdministrativa {
			t.Fatal("proyección alterada o afirmación de eficacia")
		}
	}
	if f.llamadas != 2 || f.ultima.EmpleadoRef != s.Seleccion.EmpleadoRef || f.ultima.OrganismoRef != s.Seleccion.OrganismoRef || f.ultima.Corte != s.Seleccion.Corte || f.ultima.Actor.PersonaRef != s.Actor.PersonaRef {
		t.Fatal("lectura almacenada o selector cambiado")
	}
}

func TestConsultaIncorporacionCTFinAbiertoSinInventarFecha(t *testing.T) {
	s, f := incorporacionCTPrueba(t)
	f.resultado.Ficha.Relaciones[0].Traza.Hasta = ""
	f.resultado.Ficha.Ocupaciones[0].Traza.Hasta = ""
	c, _ := NuevoServicioConsultaIncorporacionCT(f)
	r, err := c.ConsultarHechosIncorporacionCT(context.Background(), s)
	if err != nil || r.Relacion.Hasta != "" || r.Ocupacion.Hasta != "" {
		t.Fatalf("fin abierto: %v", err)
	}
}

func TestConsultaIncorporacionCTNoAceptaCrucesNiVersionesDivergentes(t *testing.T) {
	casos := map[string]func(*ports.SolicitudHechosIncorporacionCT, *fuenteIncorporacionCTPrueba){
		"persona": func(_ *ports.SolicitudHechosIncorporacionCT, f *fuenteIncorporacionCTPrueba) {
			f.resultado.Ficha.PersonaRef = "per_" + strings.Repeat("q", 24)
		},
		"organismo": func(_ *ports.SolicitudHechosIncorporacionCT, f *fuenteIncorporacionCTPrueba) {
			f.resultado.Ficha.OrganismoRef = "organismo:otro"
		},
		"empleado": func(_ *ports.SolicitudHechosIncorporacionCT, f *fuenteIncorporacionCTPrueba) {
			f.resultado.Ficha.EmpleadoRef = "emp_" + strings.Repeat("q", 24)
		},
		"version ficha": func(_ *ports.SolicitudHechosIncorporacionCT, f *fuenteIncorporacionCTPrueba) {
			f.resultado.Ficha.Version++
		},
		"version relacion": func(s *ports.SolicitudHechosIncorporacionCT, _ *fuenteIncorporacionCTPrueba) {
			s.Seleccion.VersionRelacion++
		},
		"version ocupacion": func(s *ports.SolicitudHechosIncorporacionCT, _ *fuenteIncorporacionCTPrueba) {
			s.Seleccion.VersionOcupacion++
		},
		"vinculo relacion": func(_ *ports.SolicitudHechosIncorporacionCT, f *fuenteIncorporacionCTPrueba) {
			f.resultado.Ficha.Ocupaciones[0].RelacionRef = "rel_" + strings.Repeat("q", 24)
		},
		"puesto": func(s *ports.SolicitudHechosIncorporacionCT, _ *fuenteIncorporacionCTPrueba) {
			s.Seleccion.PuestoRef = "puesto:otro"
		},
		"plaza": func(s *ports.SolicitudHechosIncorporacionCT, _ *fuenteIncorporacionCTPrueba) {
			s.Seleccion.PlazaRef = "plaza:otra"
		},
		"unidad": func(s *ports.SolicitudHechosIncorporacionCT, _ *fuenteIncorporacionCTPrueba) {
			s.Seleccion.UnidadRef = "unidad:otra"
		},
		"reserva": func(_ *ports.SolicitudHechosIncorporacionCT, f *fuenteIncorporacionCTPrueba) {
			f.resultado.Ficha.Ocupaciones[0].Clase = "reserva"
		},
		"suspendida": func(_ *ports.SolicitudHechosIncorporacionCT, f *fuenteIncorporacionCTPrueba) {
			f.resultado.Ficha.Relaciones[0].Estado = "suspendida"
		},
		"borde hasta": func(s *ports.SolicitudHechosIncorporacionCT, f *fuenteIncorporacionCTPrueba) {
			s.Seleccion.Corte.VigenteEn = "2026-10-01"
			f.resultado.Ficha.Corte = s.Seleccion.Corte
		},
		"antes desde": func(s *ports.SolicitudHechosIncorporacionCT, f *fuenteIncorporacionCTPrueba) {
			s.Seleccion.Corte.VigenteEn = "2026-08-31"
			f.resultado.Ficha.Corte = s.Seleccion.Corte
		},
		"relacion desplazada": func(_ *ports.SolicitudHechosIncorporacionCT, f *fuenteIncorporacionCTPrueba) {
			v := f.resultado.Ficha.Relaciones[0]
			v.Traza.Version++
			v.Estado = "suspendida"
			f.resultado.Ficha.Relaciones = append(f.resultado.Ficha.Relaciones, v)
		},
		"ocupacion desplazada": func(_ *ports.SolicitudHechosIncorporacionCT, f *fuenteIncorporacionCTPrueba) {
			v := f.resultado.Ficha.Ocupaciones[0]
			v.Traza.Version++
			v.Estado = "finalizada"
			f.resultado.Ficha.Ocupaciones = append(f.resultado.Ficha.Ocupaciones, v)
		},
		"ocupacion duplicada": func(_ *ports.SolicitudHechosIncorporacionCT, f *fuenteIncorporacionCTPrueba) {
			f.resultado.Ficha.Ocupaciones = append(f.resultado.Ficha.Ocupaciones, f.resultado.Ficha.Ocupaciones[0])
		},
		"evidencia ajena": func(_ *ports.SolicitudHechosIncorporacionCT, f *fuenteIncorporacionCTPrueba) {
			f.resultado.Evidencia.EfectoRef = "empleado:otro"
		},
		"evidencia sin huella": func(_ *ports.SolicitudHechosIncorporacionCT, f *fuenteIncorporacionCTPrueba) {
			f.resultado.Evidencia.ConsumoHuellaSHA256 = ""
		},
		"firma": func(_ *ports.SolicitudHechosIncorporacionCT, f *fuenteIncorporacionCTPrueba) {
			f.resultado.Ficha.FirmaOficial = true
		},
		"eficacia": func(_ *ports.SolicitudHechosIncorporacionCT, f *fuenteIncorporacionCTPrueba) {
			f.resultado.Ficha.EficaciaAdministrativa = true
		},
	}
	for nombre, cambio := range casos {
		t.Run(nombre, func(t *testing.T) {
			s, f := incorporacionCTPrueba(t)
			cambio(&s, f)
			c, _ := NuevoServicioConsultaIncorporacionCT(f)
			r, err := c.ConsultarHechosIncorporacionCT(context.Background(), s)
			if err == nil || !reflect.DeepEqual(r, ports.HechosIncorporacionCT{}) {
				t.Fatal("aceptó cruce o devolvió datos parciales")
			}
		})
	}
}

func TestConsultaIncorporacionCTRevisionRelacionNoReescribePeriodoOcupacion(t *testing.T) {
	s, f := incorporacionCTPrueba(t)
	f.resultado.Ficha.Relaciones[0].Traza.Desde = "2026-09-20"
	f.resultado.Ficha.Ocupaciones[0].Traza.Hasta = ""
	c, _ := NuevoServicioConsultaIncorporacionCT(f)
	r, err := c.ConsultarHechosIncorporacionCT(context.Background(), s)
	if err != nil || r.Ocupacion.Desde != "2026-09-01" || r.Ocupacion.Hasta != "" || r.Relacion.Desde != "2026-09-20" {
		t.Fatalf("confundió tramo de revisión con periodo de ocupación: %v", err)
	}
}

func TestConsultaIncorporacionCTDenegacionCancelacionYNulos(t *testing.T) {
	var fNula *fuenteIncorporacionCTPrueba
	if _, err := NuevoServicioConsultaIncorporacionCT(fNula); !errors.Is(err, domain.ErrRegistroEmpleadoB2NoDisponible) {
		t.Fatal(err)
	}
	for _, fallo := range []error{domain.ErrRegistroEmpleadoB2Denegado, context.Canceled, context.DeadlineExceeded, errors.New("detalle privado")} {
		s, f := incorporacionCTPrueba(t)
		f.err = fallo
		c, _ := NuevoServicioConsultaIncorporacionCT(f)
		r, err := c.ConsultarHechosIncorporacionCT(context.Background(), s)
		esperado := fallo
		if fallo.Error() == "detalle privado" {
			esperado = domain.ErrRegistroEmpleadoB2NoDisponible
		}
		if !errors.Is(err, esperado) || !reflect.DeepEqual(r, ports.HechosIncorporacionCT{}) {
			t.Fatalf("error no opaco: %v", err)
		}
	}
	s, f := incorporacionCTPrueba(t)
	c, _ := NuevoServicioConsultaIncorporacionCT(f)
	ctx, cancel := context.WithCancel(context.Background())
	f.antes = func(context.Context) { cancel() }
	if _, err := c.ConsultarHechosIncorporacionCT(ctx, s); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	s.Seleccion.PersonaRef = "persona:ejercicio:uno"
	f.llamadas = 0
	if _, err := c.ConsultarHechosIncorporacionCT(context.Background(), s); !errors.Is(err, domain.ErrRegistroEmpleadoB2Invalido) || f.llamadas != 0 {
		t.Fatal("aceptó persona de ejercicio o leyó selector inválido")
	}
}
