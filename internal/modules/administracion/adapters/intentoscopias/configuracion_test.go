package intentoscopias

import (
	"context"
	"testing"
	"time"

	http "vec-diputacion-granada/internal/modules/administracion/adapters/httpcopias"
	p "vec-diputacion-granada/internal/modules/administracion/ports/httpcopias"
	v "vec-diputacion-granada/internal/vec/ports"
)

func TestConfiguracionIncompletaAbiertaOPrivadaSeRechaza(t *testing.T) {
	f := fuenteFunc(func(context.Context, string) (Acreditacion, error) { return Acreditacion{}, nil })
	r := registradorFunc(func(context.Context, v.OrdenIntentoAuditoria) (v.AcuseIntentoAuditoria, error) {
		return v.AcuseIntentoAuditoria{}, nil
	})
	b := http.AuditorFrontera(func(context.Context, http.Denegacion) error { return nil })
	for _, mutar := range []func(*Configuracion){
		func(c *Configuracion) { c.Plazo = 0 },
		func(c *Configuracion) { c.Plazo = 31 * time.Second },
		func(c *Configuracion) { c.Canal = "interna_corporativa" },
		func(c *Configuracion) { c.Proceso = "/private/process" },
		func(c *Configuracion) { delete(c.Motivos, "servicio_no_disponible") },
		func(c *Configuracion) { c.Motivos["otro_codigo"] = c.Motivos["acceso_denegado"] },
		func(c *Configuracion) {
			m := c.Motivos["acceso_denegado"]
			m.CatalogoVersion = 0
			c.Motivos["acceso_denegado"] = m
		},
		func(c *Configuracion) { delete(c.Operaciones, p.Lanzar) },
		func(c *Configuracion) { c.Operaciones["accion_abierta"] = c.Operaciones[p.Lanzar] },
		func(c *Configuracion) {
			c.Operaciones[p.Lanzar] = Operacion{Accion: "administracion.copias.lanzar", FinalidadRef: "copias", RecursoFallback: "copias"}
		},
		func(c *Configuracion) { c.FronteraNominal = Operacion{} },
	} {
		c := configuracionPrueba()
		mutar(&c)
		if _, err := Nuevo(c, f, r, b); err != p.ErrNoDisponible {
			t.Fatal("unsafe configuration admitted", err)
		}
	}
	var fNil fuenteFunc
	var rNil registradorFunc
	for _, x := range []struct {
		f FuenteIntento
		r v.RegistradorIntentosAuditoria
		b http.AuditorFrontera
	}{{nil, r, b}, {fNil, r, b}, {f, nil, b}, {f, rNil, b}, {f, r, nil}} {
		if _, err := Nuevo(configuracionPrueba(), x.f, x.r, x.b); err != p.ErrNoDisponible {
			t.Fatal("nil dependency admitted", err)
		}
	}
}

func TestConfigServidorSeCopiaYElFalloConservaReferenciasGobernadas(t *testing.T) {
	in := intentoPrueba()
	in.Codigo = "servicio_no_disponible"
	in.RecursoRef = "copias"
	e := acreditacionPrueba(t, in.ActorPersonaRef, in.PerfilActivoRef, true)
	c := configuracionPrueba()
	expectedOp, expectedMotivo := c.Operaciones[p.Lanzar], c.Motivos[in.Codigo]
	appends := 0
	a, err := Nuevo(c, fuenteFunc(func(context.Context, string) (Acreditacion, error) { return e, nil }), registradorFunc(func(_ context.Context, o v.OrdenIntentoAuditoria) (v.AcuseIntentoAuditoria, error) {
		appends++
		data, err := o.Datos()
		if err != nil || data.Datos.Accion != expectedOp.Accion || data.Datos.FinalidadRef != expectedOp.FinalidadRef || data.Datos.RecursoRef != expectedOp.RecursoFallback || data.Datos.Motivo != expectedMotivo || data.Datos.Resultado != "error" {
			t.Fatal("configured references lost")
		}
		return acusePrueba(t, o), nil
	}), func(context.Context, http.Denegacion) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	delete(c.Motivos, in.Codigo)
	c.Operaciones[p.Lanzar] = Operacion{}
	if err = a.Registrar(context.Background(), in); err != nil || appends != 1 {
		t.Fatal(err, appends)
	}
}

func TestRutaDesconocidaNominalConservaIdentidadYConfiguracionFrontera(t *testing.T) {
	in := intentoPrueba()
	in.Accion, in.RecursoRef, in.Codigo = "", "", "recurso_no_encontrado"
	e := acreditacionPrueba(t, in.ActorPersonaRef, in.PerfilActivoRef, true)
	c := configuracionPrueba()
	appends := 0
	a, err := Nuevo(c, fuenteFunc(func(_ context.Context, correlation string) (Acreditacion, error) {
		if correlation != in.CorrelacionRef {
			t.Fatal("unknown route lost attempt")
		}
		return e, nil
	}), registradorFunc(func(_ context.Context, o v.OrdenIntentoAuditoria) (v.AcuseIntentoAuditoria, error) {
		appends++
		x, err := o.Datos()
		if err != nil || x.Datos.Accion != c.FronteraNominal.Accion || x.Datos.FinalidadRef != c.FronteraNominal.FinalidadRef || x.Datos.RecursoRef != c.FronteraNominal.RecursoFallback || x.ResultadoContexto.Contexto.PersonaRef != in.ActorPersonaRef {
			t.Fatal("unknown nominal route became anonymous")
		}
		return acusePrueba(t, o), nil
	}), func(context.Context, http.Denegacion) error { t.Fatal("nominal route downgraded"); return nil })
	if err != nil {
		t.Fatal(err)
	}
	if err = a.Registrar(context.Background(), in); err != nil || appends != 1 {
		t.Fatal(err, appends)
	}
}

func TestPlazoLimitaFuenteYNoAppendTrasAgotarlo(t *testing.T) {
	c := configuracionPrueba()
	c.Plazo = time.Millisecond
	writes := 0
	a, err := Nuevo(c, fuenteFunc(func(ctx context.Context, _ string) (Acreditacion, error) {
		<-ctx.Done()
		return Acreditacion{}, ctx.Err()
	}), registradorFunc(func(context.Context, v.OrdenIntentoAuditoria) (v.AcuseIntentoAuditoria, error) {
		writes++
		return v.AcuseIntentoAuditoria{}, nil
	}), func(context.Context, http.Denegacion) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if err = a.Registrar(context.Background(), intentoPrueba()); err != p.ErrNoDisponible || writes != 0 {
		t.Fatal(err, writes)
	}
}

func TestRecursoAjenoAlContratoComunSeMinimizaSinPerderIntento(t *testing.T) {
	in := intentoPrueba()
	e := acreditacionPrueba(t, in.ActorPersonaRef, in.PerfilActivoRef, true)
	for _, recurso := range []string{"Copia", "desconocida", "/private/path", "http://external.example", "copias", ""} {
		t.Run(recurso, func(t *testing.T) {
			in := in
			in.RecursoRef = recurso
			writes := 0
			c := configuracionPrueba()
			a, err := Nuevo(c, fuenteFunc(func(context.Context, string) (Acreditacion, error) { return e, nil }), registradorFunc(func(_ context.Context, o v.OrdenIntentoAuditoria) (v.AcuseIntentoAuditoria, error) {
				writes++
				x, err := o.Datos()
				if err != nil || x.Datos.RecursoRef != c.Operaciones[p.Lanzar].RecursoFallback || x.ResultadoContexto.Contexto.PersonaRef != in.ActorPersonaRef {
					t.Fatal("untrusted resource leaked or identity lost")
				}
				return acusePrueba(t, o), nil
			}), func(context.Context, http.Denegacion) error { t.Fatal("nominal downgrade"); return nil })
			if err != nil {
				t.Fatal(err)
			}
			if err = a.Registrar(context.Background(), in); err != nil || writes != 1 {
				t.Fatal(err, writes)
			}
		})
	}
}

func TestPlazoLimitaAppendYFronteraSeparadaSinReintentos(t *testing.T) {
	for _, nominal := range []bool{true, false} {
		c := configuracionPrueba()
		c.Plazo = 50 * time.Millisecond
		in := intentoPrueba()
		e := acreditacionPrueba(t, in.ActorPersonaRef, in.PerfilActivoRef, true)
		if !nominal {
			in.ActorPersonaRef, in.PerfilActivoRef, in.CorrelacionRef = "", "", ""
		}
		writes, boundaries := 0, 0
		a, err := Nuevo(c, fuenteFunc(func(context.Context, string) (Acreditacion, error) {
			if !nominal {
				t.Fatal("pre-session resolved a fabricated identity")
			}
			return e, nil
		}), registradorFunc(func(ctx context.Context, _ v.OrdenIntentoAuditoria) (v.AcuseIntentoAuditoria, error) {
			writes++
			<-ctx.Done()
			return v.AcuseIntentoAuditoria{}, ctx.Err()
		}), func(ctx context.Context, _ http.Denegacion) error {
			boundaries++
			<-ctx.Done()
			return ctx.Err()
		})
		if err != nil {
			t.Fatal(err)
		}
		if err = a.Registrar(context.Background(), in); err != p.ErrNoDisponible {
			t.Fatal("timeout leaked or ignored", err)
		}
		if nominal && (writes != 1 || boundaries != 0) || !nominal && (writes != 0 || boundaries != 1) {
			t.Fatal("timeout retried or crossed boundaries", writes, boundaries)
		}
	}
}
