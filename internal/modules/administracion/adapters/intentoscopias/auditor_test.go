package intentoscopias

import (
	"context"
	"errors"
	"testing"
	"time"

	http "vec-diputacion-granada/internal/modules/administracion/adapters/httpcopias"
	p "vec-diputacion-granada/internal/modules/administracion/ports/httpcopias"
	d "vec-diputacion-granada/internal/vec/domain"
	v "vec-diputacion-granada/internal/vec/ports"
)

func TestIntentoExactoCanceladoConservaUnAppendYAcreditacionOriginal(t *testing.T) {
	intento := intentoPrueba()
	evidencia := acreditacionPrueba(t, intento.ActorPersonaRef, intento.PerfilActivoRef, true)
	// The original pair has expired now. Failed-attempt audit must not ask for
	// fresh permission or reactivate the session to preserve revocation evidence.
	if evidencia.Vinculo.VigenteEn(time.Now().UTC().Truncate(time.Microsecond), evidencia.Resultado) {
		t.Fatal("fixture must be historic")
	}
	fuenteCalls, appendCalls := 0, 0
	fuente := fuenteFunc(func(ctx context.Context, correlacion string) (Acreditacion, error) {
		fuenteCalls++
		if ctx.Err() != nil || correlacion != correlacionPrueba {
			t.Fatal("lost original attempt or independent context")
		}
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) <= 0 || time.Until(deadline) > time.Second {
			t.Fatal("missing bounded deadline")
		}
		return evidencia, nil
	})
	registro := registradorFunc(func(ctx context.Context, o v.OrdenIntentoAuditoria) (v.AcuseIntentoAuditoria, error) {
		appendCalls++
		if ctx.Err() != nil {
			t.Fatal("client cancellation leaked into append")
		}
		data, err := o.Datos()
		if err != nil || data.Datos.CorrelacionRef != correlacionPrueba || data.Datos.Canal != "administracion_privilegiada" || data.Datos.Proceso != "vec-admin" || data.Datos.RecursoRef != intento.RecursoRef || data.Datos.Resultado != d.ResultadoIntentoAuditoriaDenegado || !data.Vinculo.CoincideExactamenteCon(evidencia.Vinculo) {
			t.Fatal("crossed nominal evidence")
		}
		return acusePrueba(t, o), nil
	})
	a, err := Nuevo(configuracionPrueba(), fuente, registro, func(context.Context, http.Denegacion) error { t.Fatal("nominal attempt downgraded"); return nil })
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var callback http.AuditorFrontera = a.Registrar
	if err = callback(ctx, intento); err != nil || appendCalls != 1 || fuenteCalls != 1 {
		t.Fatal(err, appendCalls, fuenteCalls)
	}
}

func TestFuentesCruzadasCorrelacionAjenaYCanalAjenoNoEscriben(t *testing.T) {
	intento := intentoPrueba()
	for _, caso := range []string{"persona", "perfil", "par_cruzado", "correlacion", "corporativo", "resultado_invalido", "error_fuente", "accion_ajena", "codigo_ajeno", "identidad_parcial"} {
		t.Run(caso, func(t *testing.T) {
			in := intento
			e := acreditacionPrueba(t, in.ActorPersonaRef, in.PerfilActivoRef, true)
			switch caso {
			case "persona":
				e = acreditacionPrueba(t, "per_1123456789abcdefghijkl", in.PerfilActivoRef, true)
			case "perfil":
				e = acreditacionPrueba(t, in.ActorPersonaRef, "prf_1123456789abcdefghijkl", true)
			case "par_cruzado":
				otra := acreditacionPrueba(t, "per_1123456789abcdefghijkl", in.PerfilActivoRef, true)
				e.Vinculo = otra.Vinculo
			case "correlacion":
				e.CorrelacionRef = "correlacion_22222222222222222222222222222222"
			case "corporativo":
				e = acreditacionPrueba(t, in.ActorPersonaRef, in.PerfilActivoRef, false)
			case "resultado_invalido":
				e.Resultado.HuellaSHA256 = "privado"
			case "accion_ajena":
				in.Accion = "copias_administrar_todo"
			case "codigo_ajeno":
				in.Codigo = "causa_privada"
			case "identidad_parcial":
				in.PerfilActivoRef = ""
			}
			writes := 0
			a, err := Nuevo(configuracionPrueba(), fuenteFunc(func(context.Context, string) (Acreditacion, error) {
				if caso == "error_fuente" {
					return e, errors.New("causa_privada_del_proveedor")
				}
				return e, nil
			}), registradorFunc(func(context.Context, v.OrdenIntentoAuditoria) (v.AcuseIntentoAuditoria, error) {
				writes++
				return v.AcuseIntentoAuditoria{}, nil
			}), func(context.Context, http.Denegacion) error { t.Fatal("unsafe fallback"); return nil })
			if err != nil {
				t.Fatal(err)
			}
			if err = a.Registrar(context.Background(), in); err != p.ErrNoDisponible || writes != 0 {
				t.Fatal("unsafe write or leaked provider error", err, writes)
			}
		})
	}
}

func TestFalloOAcuseNoConfirmadoNoSeReintentan(t *testing.T) {
	in := intentoPrueba()
	e := acreditacionPrueba(t, in.ActorPersonaRef, in.PerfilActivoRef, true)
	for _, caso := range []string{"error_append", "acuse_vacio", "acuse_ajeno"} {
		t.Run(caso, func(t *testing.T) {
			writes := 0
			a, err := Nuevo(configuracionPrueba(), fuenteFunc(func(context.Context, string) (Acreditacion, error) { return e, nil }), registradorFunc(func(_ context.Context, o v.OrdenIntentoAuditoria) (v.AcuseIntentoAuditoria, error) {
				writes++
				switch caso {
				case "error_append":
					return v.AcuseIntentoAuditoria{}, errors.New("causa_privada_sql")
				case "acuse_vacio":
					return v.AcuseIntentoAuditoria{}, nil
				default:
					a := acusePrueba(t, o)
					a.CorrelacionRef = "correlacion_22222222222222222222222222222222"
					return a, nil
				}
			}), func(context.Context, http.Denegacion) error { return nil })
			if err != nil {
				t.Fatal(err)
			}
			if err = a.Registrar(context.Background(), in); err != p.ErrNoDisponible || writes != 1 {
				t.Fatal("retry or false commit", err, writes)
			}
		})
	}
}

func TestFronteraPreSesionEsSeparada(t *testing.T) {
	for _, accion := range []string{string(p.Consultar), ""} {
		in := http.Denegacion{Codigo: "autenticacion_requerida", Accion: accion}
		for _, falla := range []bool{false, true} {
			calls := 0
			a, err := Nuevo(configuracionPrueba(), fuenteFunc(func(context.Context, string) (Acreditacion, error) {
				t.Fatal("fabricated nominal identity")
				return Acreditacion{}, nil
			}), registradorFunc(func(context.Context, v.OrdenIntentoAuditoria) (v.AcuseIntentoAuditoria, error) {
				t.Fatal("nominal append before session")
				return v.AcuseIntentoAuditoria{}, nil
			}), func(ctx context.Context, received http.Denegacion) error {
				calls++
				if received != in || ctx.Err() != nil {
					t.Fatal("modified pre-session attempt")
				}
				if falla {
					return errors.New("private_boundary_error")
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			err = a.Registrar(ctx, in)
			if calls != 1 || falla && err != p.ErrNoDisponible || !falla && err != nil {
				t.Fatal(err, calls)
			}
		}
	}
}
