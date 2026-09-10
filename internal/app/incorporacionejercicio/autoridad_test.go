package incorporacionejercicio

// Fixtures nominales de ejercicio derivados de lecturaincorporacion/nominal_fixture_test.go.
// Identidad/contexto son dobles explícitos. No declaran commits ni gobierno real.
import (
	"context"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	pa "vec-diputacion-granada/internal/modules/personal/adapters/contrataciontemporal"
	pl "vec-diputacion-granada/internal/modules/personal/adapters/lecturaincorporacion"
	core "vec-diputacion-granada/internal/vec/domain"
)

type autoridadRevalidadorDoble struct {
	resultado core.AutenticacionRevalidadaV1
	fallo     error
	despues   func()
}

func (d autoridadRevalidadorDoble) RevalidarAutenticacionActorV1(context.Context, core.SolicitudRevalidacionAutenticacionActorV1) (core.AutenticacionRevalidadaV1, error) {
	if d.despues != nil {
		d.despues()
	}
	return d.resultado, d.fallo
}

type autoridadContextoDoble struct {
	resultado core.ResultadoContextoActorRegistradoV2
}

func (d autoridadContextoDoble) ResolverContextoActorRegistradoV2(context.Context, core.SolicitudContextoActor) (core.ResultadoContextoActorRegistradoV2, error) {
	return d.resultado.Clonar()
}

type autoridadRelojDoble struct {
	instante time.Time
	despues  func()
}

func (d autoridadRelojDoble) Ahora() time.Time {
	if d.despues != nil {
		d.despues()
	}
	return d.instante
}

type autoridadFuenteDoble struct {
	p       PeticionAutoridad
	fallo   error
	despues func()
}

func (d autoridadFuenteDoble) PeticionVerificada(context.Context) (PeticionAutoridad, error) {
	if d.despues != nil {
		d.despues()
	}
	return d.p, d.fallo
}
func autoridadFixtureContexto(
	t *testing.T,
	ahora time.Time,
	marcaActor string,
	marcaPerfil string,
) (ct.ContextoAutorizacionAltaV3, core.AutenticacionRevalidadaV1, core.SolicitudContextoActor) {
	t.Helper()
	cuenta := core.CuentaAutenticadaContextoActor{
		CuentaRef: "cta_0123456789abcdefghijkl",
		Metodo:    core.AuthMethodCertificate,
		Garantia:  core.AuthAssuranceHigh,
	}
	instantanea := core.InstantaneaContextoActor{
		VinculoRef:      "vca_0123456789abcdefghijkl" + marcaActor + marcaPerfil,
		VinculoVersion:  3,
		CuentaRef:       cuenta.CuentaRef,
		CuentaVersion:   4,
		PersonaRef:      "per_0123456789abcdefghijkl" + marcaActor,
		PersonaVersion:  2,
		PerfilActivoRef: "prf_0123456789abcdefghijkl" + marcaPerfil,
		PerfilVersion:   5,
		Estado:          core.EstadoVinculoContextoActorActivo,
		VigenteDesde:    ahora.Add(-time.Hour),
		VigenteHasta:    ahora.Add(time.Hour),
	}
	actor, err := core.NuevoContextoActor(cuenta, instantanea, ahora.Add(-2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	canon, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	huella, err := actor.HuellaSHA256VinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	acreditacion := core.AcreditacionProcedenciaComponenteContextoActorV1{
		ProcedenciaRef:          "prc_0123456789abcdefghijkl",
		ProcedenciaVersion:      1,
		ProcedenciaHuellaSHA256: strings.Repeat("4", 64),
		ProcedenciaAutoridad:    core.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
	}
	manifiesto := core.ManifiestoProcedenciaContextoActorV1{
		Esquema:           core.EsquemaManifiestoProcedenciaContextoActorV1,
		AutoridadEfectiva: core.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		Cuenta: core.ProcedenciaCuentaContextoActorV1{
			CuentaRef: instantanea.CuentaRef,
			Version:   instantanea.CuentaVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Persona: core.ProcedenciaPersonaContextoActorV1{
			PersonaRef: instantanea.PersonaRef,
			Version:    instantanea.PersonaVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Perfil: core.ProcedenciaPerfilContextoActorV1{
			PerfilRef: instantanea.PerfilActivoRef,
			Version:   instantanea.PerfilVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Contexto: core.ProcedenciaVinculoContextoActorV1{
			VinculoRef: instantanea.VinculoRef,
			Version:    instantanea.VinculoVersion,
			AcreditacionProcedenciaComponenteContextoActorV1: acreditacion,
		},
		Vinculos: []core.ProcedenciaVinculoReferenciaContextoActorV1{},
	}
	manifiestoCanon, err := manifiesto.RepresentacionCanonicaV1()
	if err != nil {
		t.Fatal(err)
	}
	manifiestoHuella, err := core.HuellaSHA256ManifiestoProcedenciaContextoActorV1(
		manifiestoCanon,
	)
	if err != nil {
		t.Fatal(err)
	}
	resultado := core.ResultadoContextoActorRegistradoV2{
		RegistroContextoRef: "rca_0123456789abcdefghijklmn" +
			marcaActor + marcaPerfil + strconv.FormatInt(ahora.UnixMicro(), 10),
		Contexto:                          actor,
		RepresentacionCanonica:            canon,
		HuellaSHA256:                      huella,
		ManifiestoProcedenciaCanonico:     manifiestoCanon,
		ManifiestoProcedenciaHuellaSHA256: manifiestoHuella,
		AutoridadEfectiva: core.
			AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		ResueltoEnAutoritativo: actor.ResueltoEn,
	}
	autenticacion := core.AutenticacionRevalidadaV1{
		AutenticacionRef:             "aut_0123456789abcdefghijkl",
		AutenticacionHuellaSHA256:    strings.Repeat("1", 64),
		AsercionRef:                  "ase_0123456789abcdefghijkl",
		SesionRef:                    "ses_0123456789abcdefghijkl",
		ControlSesionRef:             "cse_0123456789abcdefghijkl",
		ControlSesionRevision:        2,
		ControlSesionHuellaSHA256:    strings.Repeat("2", 64),
		CuentaRef:                    cuenta.CuentaRef,
		CuentaOrdinariaRef:           cuenta.CuentaRef,
		Superficie:                   core.SuperficieAutenticacionInternaCorporativaV1,
		MetodoObservado:              cuenta.Metodo,
		GarantiaObservada:            cuenta.Garantia,
		PoliticaGarantiaRef:          "pga_0123456789abcdefghijkl",
		PoliticaGarantiaHuellaSHA256: strings.Repeat("3", 64),
		AutenticacionVerificadaEn:    ahora.Add(-10 * time.Minute),
		SesionEmitidaEn:              ahora.Add(-9 * time.Minute),
		SesionValidaHasta:            ahora.Add(20 * time.Minute),
		SesionRevalidadaEn:           ahora.Add(-3 * time.Minute),
	}
	vinculo, err := core.CrearVinculoAutenticacionActorV2(
		context.Background(),
		autoridadRevalidadorDoble{resultado: autenticacion},
		core.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: autenticacion.AutenticacionRef,
			SesionRef:        autenticacion.SesionRef,
		},
		autoridadContextoDoble{resultado: resultado},
		core.SolicitudContextoActor{
			Cuenta:          cuenta,
			PerfilActivoRef: instantanea.PerfilActivoRef,
		},
		autoridadRelojDoble{instante: ahora},
	)
	if err != nil {
		t.Fatal(err)
	}
	return ct.ContextoAutorizacionAltaV3{Vinculo: vinculo, Resultado: resultado}, autenticacion, core.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: instantanea.PerfilActivoRef}
}

func TestAutoridadAplicacionIdentidadCadaCampo(t *testing.T) {
	ahora := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	c, r, _ := autoridadFixtureContexto(t, ahora, "a", "a")
	v, _ := c.Vinculo.Datos()
	if !autoridadAutenticacionExacta(r, v, ahora) {
		t.Fatal("nominal")
	}
	for i := 0; i < reflect.TypeOf(r).NumField(); i++ {
		t.Run(reflect.TypeOf(r).Field(i).Name, func(t *testing.T) {
			x := r
			f := reflect.ValueOf(&x).Elem().Field(i)
			switch f.Kind() {
			case reflect.String:
				f.SetString(f.String() + "x")
			case reflect.Bool:
				f.SetBool(!f.Bool())
			case reflect.Uint64:
				f.SetUint(f.Uint() + 1)
			case reflect.Struct:
				f.Set(reflect.ValueOf(ahora.Add(time.Hour)))
			}
			if autoridadAutenticacionExacta(x, v, ahora) {
				t.Fatal("cruce admitido")
			}
		})
	}
	r.SesionRevalidadaEn = ahora
	if !autoridadAutenticacionExacta(r, v, ahora) {
		t.Fatal("revalidacion fresca")
	}
}
func TestAutoridadAplicacionFronteras(t *testing.T) {
	for _, caso := range []string{"cancel_inicial", "fuente_cancel", "revalidacion_cancel", "ultimo_reloj", "retroceso", "utc", "submicro", "caducidad", "contexto_copia", "actor_material", "contexto_ajeno", "preparacion_ct"} {
		t.Run(caso, func(t *testing.T) {
			e := autoridadEntorno(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch caso {
			case "cancel_inicial":
				cancel()
				_, err := NuevaAutoridadAplicacion(ctx, nil, nil, nil, nil, nil, nil)
				if !errors.Is(err, context.Canceled) {
					t.Fatal("cancel")
				}
				return
			case "fuente_cancel":
				e.fuente.despues = cancel
				_, err := NuevaAutoridadAplicacion(ctx, e.fuente, e.reval, autoridadContextoDoble{e.a.contexto.Resultado}, e.a.cadena, e.gen, e.reloj)
				if !errors.Is(err, context.Canceled) {
					t.Fatal("fuente")
				}
				return
			case "revalidacion_cancel":
				e.a.revalidador = autoridadRevalidadorDoble{resultado: e.reval.resultado, despues: cancel}
			case "ultimo_reloj":
				e.a.reloj = autoridadRelojDoble{instante: e.ahora, despues: cancel}
			case "retroceso":
				e.a.reloj = autoridadRelojDoble{instante: e.ahora.Add(-time.Microsecond)}
			case "utc":
				e.a.reloj = autoridadRelojDoble{instante: e.ahora.In(time.FixedZone("otro", 3600))}
			case "submicro":
				e.a.reloj = autoridadRelojDoble{instante: e.ahora.Add(time.Nanosecond)}
			case "caducidad":
				e.a.reloj = autoridadRelojDoble{instante: e.ahora.Add(time.Hour)}
			case "contexto_copia":
				c, _ := e.a.ContextoAutoridad()
				c.Resultado.RepresentacionCanonica[0] ^= 1
				c.Resultado.Contexto.Instantanea.PersonaVersion++
				d, _ := e.a.ContextoAutoridad()
				if d.Resultado.Validar() != nil {
					t.Fatal("alias")
				}
				return
			case "actor_material":
				m := autoridadMaterialAlta(t, e)
				m.ActorRef = "actor:ajeno"
				x, err := e.a.AutorizarAlta(ctx, m)
				if err == nil || !reflect.DeepEqual(x, pa.AutorizacionAlta{}) || e.store.registros != 0 {
					t.Fatal("actor")
				}
				return
			case "contexto_ajeno":
				c, _, _ := autoridadFixtureContexto(t, e.ahora, "b", "b")
				m, err := pl.NuevoMaterialV2(autoridadSelector(e), e.fuente.p.PreparacionCT.UnidadRef, c, e.ahora)
				if err != nil {
					t.Fatal("fixture")
				}
				x, err := e.a.AutorizarLecturaIncorporacionV2(ctx, m)
				if err == nil || !reflect.DeepEqual(x, pl.AutorizacionV2{}) || e.store.registros != 0 {
					t.Fatal("contexto")
				}
				return
			case "preparacion_ct":
				d := autoridadMaterialCT(t, e)
				d.Preparacion.ActorRef = "actor:ajeno"
				m, err := ct.NuevoMaterialConfirmacionIncorporacionV2(d, e.ahora)
				if err != nil {
					t.Fatal("fixture")
				}
				x, err := e.a.AutorizarConfirmacionIncorporacion(ctx, m)
				if err == nil || !reflect.DeepEqual(x, ct.AutorizacionConfirmacionIncorporacionV2{}) || e.store.registros != 0 {
					t.Fatal("CT")
				}
				return
			}
			_, err := e.a.revalidar(ctx, e.ahora)
			if err == nil {
				t.Fatal("frontera admitida")
			}
			if (caso == "revalidacion_cancel" || caso == "ultimo_reloj") && !errors.Is(err, context.Canceled) {
				t.Fatal("cancel perdida")
			}
		})
	}
}
