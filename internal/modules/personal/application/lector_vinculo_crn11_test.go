package application

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type autorizadorVinculoCRN11Prueba func(context.Context, domain.MaterialVinculoPropioCRN11) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)

func (f autorizadorVinculoCRN11Prueba) AutorizarVinculoPropioCRN11(ctx context.Context, m domain.MaterialVinculoPropioCRN11) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return f(ctx, m)
}

type repositorioVinculoCRN11Prueba func(context.Context, ports.OrdenVinculoPropioCRN11) (ports.ResultadoVinculoPropioCRN11, error)

func (f repositorioVinculoCRN11Prueba) ConsultarVinculoPropioCRN11(ctx context.Context, o ports.OrdenVinculoPropioCRN11) (ports.ResultadoVinculoPropioCRN11, error) {
	return f(ctx, o)
}

func solicitudVinculoCRN11Prueba(t *testing.T, prefijo string) domain.SolicitudVinculoPropioCRN11 {
	t.Helper()
	s := solicitudFichaPropiaPrueba(t, prefijo)
	return domain.SolicitudVinculoPropioCRN11{Actor: s.Actor, EmpleadoRef: "emp_" + strings.Repeat("a", 24)}
}

type varianteAtestacionCRN11 struct {
	accion, audiencia, efecto, huella string
	canon                             []byte
	personaVersion, perfilVersion     uint64
}

func atestacionVinculoCRN11Prueba(t *testing.T, m domain.MaterialVinculoPropioCRN11, v varianteAtestacionCRN11) vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	actor := m.Actor()
	canon, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	if v.canon != nil {
		canon = v.canon
	}
	h, err := m.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	if v.huella != "" {
		h = v.huella
	}
	if v.accion == "" {
		v.accion = domain.AccionVinculoPropioCRN11
	}
	if v.audiencia == "" {
		v.audiencia = domain.AudienciaVinculoPropioCRN11
	}
	if v.efecto == "" {
		v.efecto = m.EmpleadoRef()
	}
	if v.personaVersion == 0 {
		v.personaVersion = actor.Instantanea.PersonaVersion
	}
	if v.perfilVersion == 0 {
		v.perfilVersion = actor.Instantanea.PerfilVersion
	}
	n, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_crn11", strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_crn11", strings.Repeat("c", 64), v.accion, v.efecto, h, v.audiencia, actor.ResueltoEn, actor.ResueltoEn.Add(3*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	raiz, err := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	if err != nil {
		t.Fatal(err)
	}
	a, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), n, []byte("d"), []byte("m"), canon, v.personaVersion, v.perfilVersion, []byte("p"), []byte("s"), []byte("e"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func resultadoVinculoCRN11Prueba(o ports.OrdenVinculoPropioCRN11) ports.ResultadoVinculoPropioCRN11 {
	m := o.Material
	x := o.Autorizacion.ResumenCapacidad()
	return ports.ResultadoVinculoPropioCRN11{
		Vinculo:   domain.VinculoHistoricoCRN11{PersonaRef: m.Actor().PersonaRef, EmpleadoRef: m.EmpleadoRef(), VinculoRef: m.VinculoRef(), FuenteRef: "prc_" + strings.Repeat("a", 24), Version: m.VinculoVersion()},
		Evidencia: ports.EvidenciaRegistroEmpleadoB2{ReciboRef: "recibo:personal:crn11", DecisionRef: x.DecisionRef(), EfectoRef: x.EfectoRef(), ConsumoHuellaSHA256: strings.Repeat("d", 64), AuditoriaRef: "auditoria:personal:crn11", ConsultadaEn: x.EmitidaEn().Add(time.Microsecond)},
	}
}

func TestVinculoCRN11ConsultaPropiaConAutoridadNominal(t *testing.T) {
	in := solicitudVinculoCRN11Prueba(t, "pep_")
	ahora := in.Actor.ResueltoEn.Add(time.Millisecond)
	consultasAutorizacion, consultasFuente := 0, 0
	a := autorizadorVinculoCRN11Prueba(func(_ context.Context, m domain.MaterialVinculoPropioCRN11) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
		consultasAutorizacion++
		if m.EmpleadoRef() != in.EmpleadoRef || m.VinculoRef() != "pep_"+strings.Repeat("a", 24) || m.VinculoVersion() != 1 {
			t.Fatal("material sin proyeccion exacta")
		}
		return atestacionVinculoCRN11Prueba(t, m, varianteAtestacionCRN11{}), nil
	})
	r := repositorioVinculoCRN11Prueba(func(_ context.Context, o ports.OrdenVinculoPropioCRN11) (ports.ResultadoVinculoPropioCRN11, error) {
		consultasFuente++
		return resultadoVinculoCRN11Prueba(o), nil
	})
	s, err := NuevoServicioVinculoPropioCRN11(a, r, func() time.Time { return ahora })
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.ConsultarVinculoPropioCRN11(context.Background(), in)
	if err != nil || got.Vinculo.PersonaRef != in.Actor.PersonaRef || got.Vinculo.EmpleadoRef != in.EmpleadoRef || got.Vinculo.Version != 1 || consultasAutorizacion != 1 || consultasFuente != 1 {
		t.Fatalf("lectura propia: resultado=%+v error=%v autorizaciones=%d fuente=%d", got, err, consultasAutorizacion, consultasFuente)
	}
}

func TestVinculoCRN11RechazaSelectorAjenoYOrigenLegadoAntesDeAutorizar(t *testing.T) {
	for _, nombre := range []string{"ajeno", "vin_", "sin_empleado"} {
		t.Run(nombre, func(t *testing.T) {
			in := solicitudVinculoCRN11Prueba(t, "pep_")
			switch nombre {
			case "ajeno":
				in.EmpleadoRef = "emp_" + strings.Repeat("b", 24)
			case "vin_":
				in = solicitudVinculoCRN11Prueba(t, "vin_")
			case "sin_empleado":
				in.Actor.Instantanea.Vinculos = nil
			}
			llamadas := 0
			a := autorizadorVinculoCRN11Prueba(func(context.Context, domain.MaterialVinculoPropioCRN11) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
				llamadas++
				return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, nil
			})
			r := repositorioVinculoCRN11Prueba(func(context.Context, ports.OrdenVinculoPropioCRN11) (ports.ResultadoVinculoPropioCRN11, error) {
				llamadas++
				return ports.ResultadoVinculoPropioCRN11{}, nil
			})
			s, _ := NuevoServicioVinculoPropioCRN11(a, r, func() time.Time { return in.Actor.ResueltoEn })
			got, err := s.ConsultarVinculoPropioCRN11(context.Background(), in)
			if !errors.Is(err, domain.ErrVinculoCRN11Denegado) || got != (ports.ResultadoVinculoPropioCRN11{}) || llamadas != 0 {
				t.Fatalf("selector admitido: %v, llamadas=%d", err, llamadas)
			}
		})
	}
}

func TestVinculoCRN11RechazaAtestacionDeOtroUso(t *testing.T) {
	for _, caso := range []struct {
		name    string
		alterar func(*varianteAtestacionCRN11)
	}{
		{"accion", func(v *varianteAtestacionCRN11) { v.accion = domain.AccionFichaPropia }},
		{"audiencia", func(v *varianteAtestacionCRN11) { v.audiencia = domain.AudienciaFichaPropia }},
		{"efecto", func(v *varianteAtestacionCRN11) { v.efecto = "emp_" + strings.Repeat("b", 24) }},
		{"huella", func(v *varianteAtestacionCRN11) { v.huella = strings.Repeat("f", 64) }},
		{"contexto", func(v *varianteAtestacionCRN11) { v.canon = []byte("otro contexto") }},
		{"persona_version", func(v *varianteAtestacionCRN11) { v.personaVersion = 2 }},
		{"perfil_version", func(v *varianteAtestacionCRN11) { v.perfilVersion = 2 }},
	} {
		t.Run(caso.name, func(t *testing.T) {
			in := solicitudVinculoCRN11Prueba(t, "pep_")
			fuente := 0
			a := autorizadorVinculoCRN11Prueba(func(_ context.Context, m domain.MaterialVinculoPropioCRN11) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
				v := varianteAtestacionCRN11{}
				caso.alterar(&v)
				return atestacionVinculoCRN11Prueba(t, m, v), nil
			})
			r := repositorioVinculoCRN11Prueba(func(context.Context, ports.OrdenVinculoPropioCRN11) (ports.ResultadoVinculoPropioCRN11, error) {
				fuente++
				return ports.ResultadoVinculoPropioCRN11{}, nil
			})
			s, _ := NuevoServicioVinculoPropioCRN11(a, r, func() time.Time { return in.Actor.ResueltoEn.Add(time.Millisecond) })
			got, err := s.ConsultarVinculoPropioCRN11(context.Background(), in)
			if !errors.Is(err, domain.ErrVinculoCRN11NoDisponible) || got != (ports.ResultadoVinculoPropioCRN11{}) || fuente != 0 {
				t.Fatalf("autoridad cruzada admitida: %v, fuente=%d", err, fuente)
			}
		})
	}
}

func TestVinculoCRN11SinConcesionNoConsultaFuente(t *testing.T) {
	for _, caso := range []struct {
		nombre, detalle string
		fallo, esperado error
	}{
		{"ausente", "", domain.ErrVinculoCRN11Denegado, domain.ErrVinculoCRN11Denegado},
		{"autoridad_caida", "detalle sensible del PDP", errors.New("detalle sensible del PDP"), domain.ErrVinculoCRN11NoDisponible},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			in := solicitudVinculoCRN11Prueba(t, "pep_")
			fuente := 0
			a := autorizadorVinculoCRN11Prueba(func(context.Context, domain.MaterialVinculoPropioCRN11) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
				return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, caso.fallo
			})
			r := repositorioVinculoCRN11Prueba(func(context.Context, ports.OrdenVinculoPropioCRN11) (ports.ResultadoVinculoPropioCRN11, error) {
				fuente++
				return ports.ResultadoVinculoPropioCRN11{}, nil
			})
			s, _ := NuevoServicioVinculoPropioCRN11(a, r, func() time.Time { return in.Actor.ResueltoEn })
			got, err := s.ConsultarVinculoPropioCRN11(context.Background(), in)
			if !errors.Is(err, caso.esperado) || got != (ports.ResultadoVinculoPropioCRN11{}) || fuente != 0 || strings.Contains(err.Error(), caso.detalle) && caso.detalle != "" {
				t.Fatalf("concesion ausente llego a fuente o filtro detalle: %v, llamadas=%d", err, fuente)
			}
		})
	}
}

func TestVinculoCRN11RechazaFuenteCruzadaOIncompleta(t *testing.T) {
	for _, caso := range []struct {
		name    string
		alterar func(*ports.ResultadoVinculoPropioCRN11)
	}{
		{"persona", func(r *ports.ResultadoVinculoPropioCRN11) { r.Vinculo.PersonaRef = "per_" + strings.Repeat("b", 24) }},
		{"empleado", func(r *ports.ResultadoVinculoPropioCRN11) { r.Vinculo.EmpleadoRef = "emp_" + strings.Repeat("b", 24) }},
		{"proyeccion", func(r *ports.ResultadoVinculoPropioCRN11) { r.Vinculo.VinculoRef = "pep_" + strings.Repeat("b", 24) }},
		{"version", func(r *ports.ResultadoVinculoPropioCRN11) { r.Vinculo.Version++ }},
		{"sin_version", func(r *ports.ResultadoVinculoPropioCRN11) { r.Vinculo.Version = 0 }},
		{"sin_procedencia", func(r *ports.ResultadoVinculoPropioCRN11) { r.Vinculo.FuenteRef = "" }},
		{"procedencia_ajena", func(r *ports.ResultadoVinculoPropioCRN11) { r.Vinculo.FuenteRef = "fuente:otra" }},
		{"decision", func(r *ports.ResultadoVinculoPropioCRN11) { r.Evidencia.DecisionRef = "dec_otra" }},
		{"recibo", func(r *ports.ResultadoVinculoPropioCRN11) { r.Evidencia.ReciboRef = "" }},
		{"auditoria", func(r *ports.ResultadoVinculoPropioCRN11) { r.Evidencia.AuditoriaRef = "" }},
		{"fecha_futura", func(r *ports.ResultadoVinculoPropioCRN11) {
			r.Evidencia.ConsultadaEn = r.Evidencia.ConsultadaEn.Add(time.Second)
		}},
	} {
		t.Run(caso.name, func(t *testing.T) {
			in := solicitudVinculoCRN11Prueba(t, "pep_")
			a := autorizadorVinculoCRN11Prueba(func(_ context.Context, m domain.MaterialVinculoPropioCRN11) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
				return atestacionVinculoCRN11Prueba(t, m, varianteAtestacionCRN11{}), nil
			})
			r := repositorioVinculoCRN11Prueba(func(_ context.Context, o ports.OrdenVinculoPropioCRN11) (ports.ResultadoVinculoPropioCRN11, error) {
				got := resultadoVinculoCRN11Prueba(o)
				caso.alterar(&got)
				return got, nil
			})
			s, _ := NuevoServicioVinculoPropioCRN11(a, r, func() time.Time { return in.Actor.ResueltoEn.Add(time.Millisecond) })
			got, err := s.ConsultarVinculoPropioCRN11(context.Background(), in)
			if !errors.Is(err, domain.ErrVinculoCRN11NoDisponible) || got != (ports.ResultadoVinculoPropioCRN11{}) {
				t.Fatalf("fuente admitida: %+v, %v", got, err)
			}
		})
	}
}

func TestVinculoCRN11RevalidaCaducidadYNoFiltraErrores(t *testing.T) {
	in := solicitudVinculoCRN11Prueba(t, "pep_")
	for _, caso := range []struct {
		name     string
		fallo    error
		caducar  bool
		esperado error
	}{
		{"permiso_retirado", domain.ErrVinculoCRN11Denegado, false, domain.ErrVinculoCRN11Denegado},
		{"fuente_caida", errors.New("detalle sensible de base"), false, domain.ErrVinculoCRN11NoDisponible},
		{"caduca_durante_lectura", nil, true, domain.ErrVinculoCRN11Denegado},
	} {
		t.Run(caso.name, func(t *testing.T) {
			ahora := in.Actor.ResueltoEn.Add(time.Millisecond)
			a := autorizadorVinculoCRN11Prueba(func(_ context.Context, m domain.MaterialVinculoPropioCRN11) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
				return atestacionVinculoCRN11Prueba(t, m, varianteAtestacionCRN11{}), nil
			})
			r := repositorioVinculoCRN11Prueba(func(_ context.Context, o ports.OrdenVinculoPropioCRN11) (ports.ResultadoVinculoPropioCRN11, error) {
				if caso.caducar {
					ahora = in.Actor.ResueltoEn.Add(3 * time.Second)
				}
				return resultadoVinculoCRN11Prueba(o), caso.fallo
			})
			s, _ := NuevoServicioVinculoPropioCRN11(a, r, func() time.Time { return ahora })
			got, err := s.ConsultarVinculoPropioCRN11(context.Background(), in)
			if !errors.Is(err, caso.esperado) || got != (ports.ResultadoVinculoPropioCRN11{}) || strings.Contains(err.Error(), "sensible") {
				t.Fatalf("fallo filtrado: %+v, %v", got, err)
			}
		})
	}
}

func TestVinculoCRN11ConstructorRechazaDependenciasNulas(t *testing.T) {
	var aNula *autorizadorVinculoCRN11Prueba
	var rNulo *repositorioVinculoCRN11Prueba
	a := autorizadorVinculoCRN11Prueba(func(context.Context, domain.MaterialVinculoPropioCRN11) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, nil
	})
	r := repositorioVinculoCRN11Prueba(func(context.Context, ports.OrdenVinculoPropioCRN11) (ports.ResultadoVinculoPropioCRN11, error) {
		return ports.ResultadoVinculoPropioCRN11{}, nil
	})
	for _, deps := range []struct {
		a     ports.ProveedorAutorizacionVinculoCRN11
		r     ports.RepositorioVinculoPropioCRN11
		reloj func() time.Time
	}{
		{nil, r, time.Now}, {aNula, r, time.Now}, {a, nil, time.Now}, {a, rNulo, time.Now}, {a, r, nil},
	} {
		s, err := NuevoServicioVinculoPropioCRN11(deps.a, deps.r, deps.reloj)
		if s != nil || !errors.Is(err, domain.ErrVinculoCRN11NoDisponible) {
			t.Fatalf("dependencia nula: %v", err)
		}
	}
}
