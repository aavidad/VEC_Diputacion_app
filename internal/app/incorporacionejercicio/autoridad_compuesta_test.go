package incorporacionejercicio

import (
	"context"
	"errors"
	"testing"

	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	pa "vec-diputacion-granada/internal/modules/personal/adapters/contrataciontemporal"
	pl "vec-diputacion-granada/internal/modules/personal/adapters/lecturaincorporacion"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type revalidadorCompuestoPrueba struct {
	ct, alta                 core.AutenticacionRevalidadaV1
	falloAlta                bool
	lecturasCT, lecturasAlta int
}

func (r *revalidadorCompuestoPrueba) RevalidarAutenticacionActorV1(_ context.Context, s core.SolicitudRevalidacionAutenticacionActorV1) (core.AutenticacionRevalidadaV1, error) {
	if s.AutenticacionRef == r.ct.AutenticacionRef && s.SesionRef == r.ct.SesionRef {
		r.lecturasCT++
		return r.ct, nil
	}
	if s.AutenticacionRef == r.alta.AutenticacionRef && s.SesionRef == r.alta.SesionRef {
		r.lecturasAlta++
		if !r.falloAlta {
			return r.alta, nil
		}
	}
	return core.AutenticacionRevalidadaV1{}, ErrAutoridadAplicacion
}

type resolutorCompuestoPrueba struct {
	ct, alta core.ResultadoContextoActorRegistradoV2
}

func (r resolutorCompuestoPrueba) ResolverContextoActorRegistradoV2(_ context.Context, s core.SolicitudContextoActor) (core.ResultadoContextoActorRegistradoV2, error) {
	if s.PerfilActivoRef == r.ct.Contexto.PerfilActivoRef {
		return r.ct.Clonar()
	}
	if s.PerfilActivoRef == r.alta.Contexto.PerfilActivoRef {
		return r.alta.Clonar()
	}
	return core.ResultadoContextoActorRegistradoV2{}, ErrAutoridadAplicacion
}

type pdpCompuestoPrueba struct {
	ct, alta vp.AutorizadorSolicitudLigadaV3
}

func (p pdpCompuestoPrueba) ExigirSolicitudLigadaV3(ctx context.Context, s core.SolicitudAutorizacionLigadaV3, r core.ResultadoContextoActorRegistradoV2) (core.DecisionAutorizacionLigadaV3, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
	d, err := s.Datos()
	if err == nil && d.Accion == pa.AccionAltaEjercicio {
		return p.alta.ExigirSolicitudLigadaV3(ctx, s, r)
	}
	return p.ct.ExigirSolicitudLigadaV3(ctx, s, r)
}

func casoAutoridadCompuesta(t *testing.T) (*autoridadEscenario, *autoridadEscenario, *revalidadorCompuestoPrueba, resolutorCompuestoPrueba, PeticionAutoridad) {
	t.Helper()
	c, a := autoridadEntorno(t), autoridadEntorno(t, autoridadAlta)
	autAlta := a.reval.resultado
	autAlta.AutenticacionRef += "alta"
	autAlta.SesionRef += "alta"
	autAlta.ControlSesionRef += "alta"
	autAlta.AsercionRef += "alta"
	reval := &revalidadorCompuestoPrueba{ct: c.reval.resultado, alta: autAlta}
	res := resolutorCompuestoPrueba{ct: c.a.contexto.Resultado, alta: a.a.contexto.Resultado}
	p := c.fuente.p
	p.AltaNominal = &PeticionAutoridadAlta{Autenticacion: core.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: autAlta.AutenticacionRef, SesionRef: autAlta.SesionRef}, Contexto: a.fuente.p.Contexto}
	p.PerfilesNominales = PerfilesAutoridadAplicacion{AltaRef: a.fuente.p.Contexto.PerfilActivoRef, ConsultaConfirmacionRef: p.Contexto.PerfilActivoRef}
	c.a.cadena.servicio = pdpCompuestoPrueba{ct: c.a.cadena.servicio, alta: a.a.cadena.servicio}
	return c, a, reval, res, p
}

func TestAutoridadCompuestaSeparaAltaYConservaLecturaCT(t *testing.T) {
	c, alta, reval, res, p := casoAutoridadCompuesta(t)
	a, err := NuevaAutoridadAplicacion(context.Background(), autoridadFuenteDoble{p: p}, reval, res, c.a.cadena, c.gen, c.reloj)
	autoridadExigir(t, err)
	// La captura externa no puede cambiar el perfil/sesión ya fijados.
	p.AltaNominal.Contexto.PerfilActivoRef = p.Contexto.PerfilActivoRef
	m := autoridadMaterialAlta(t, alta)
	antesCT, antesAlta := reval.lecturasCT, reval.lecturasAlta
	r, err := (proveedorAltaLigado{base: a, p: ct.PreparacionIncorporacionAplicacionV2{SolicitudPersonal: m.Preparacion.Solicitud, Preparacion: c.fuente.p.PreparacionCT, Contexto: a.contexto}}).ResolverAutoridad(context.Background(), m.Preparacion)
	autoridadExigir(t, err)
	vAlta, _ := r.Contexto.Vinculo.Datos()
	vCT, _ := a.contexto.Vinculo.Datos()
	if vAlta.PerfilActivoRef != a.peticion.PerfilesNominales.AltaRef || vCT.PerfilActivoRef != a.peticion.PerfilesNominales.ConsultaConfirmacionRef || vAlta.SesionRef == vCT.SesionRef || reval.lecturasCT <= antesCT || reval.lecturasAlta <= antesAlta {
		t.Fatal("el alta no usó su perfil/sesión nominales revalidados")
	}
	autAlta, err := a.AutorizarAlta(context.Background(), m)
	autoridadExigir(t, err)
	sAlta, err := autAlta.Solicitud.Datos()
	autoridadExigir(t, err)
	if !sAlta.VinculoAutenticacionActor.CoincideExactamenteCon(r.Contexto.Vinculo) {
		t.Fatal("concesión de alta ligada a CT")
	}
	lectura, err := pl.NuevoMaterialV2(autoridadSelector(c), c.fuente.p.PreparacionCT.UnidadRef, a.contexto, c.ahora)
	autoridadExigir(t, err)
	autLectura, err := a.AutorizarLecturaIncorporacionV2(context.Background(), lectura)
	autoridadExigir(t, err)
	sLectura, _ := autLectura.Solicitud.Datos()
	if !sLectura.VinculoAutenticacionActor.CoincideExactamenteCon(a.contexto.Vinculo) {
		t.Fatal("lectura Personal separada de CT")
	}
	c.a = a
	materialCT, err := ct.NuevoMaterialConfirmacionIncorporacionV2(autoridadMaterialCT(t, c, m), c.ahora)
	autoridadExigir(t, err)
	autCT, err := a.AutorizarConfirmacionIncorporacion(context.Background(), materialCT)
	autoridadExigir(t, err)
	sCT, _ := autCT.Solicitud.Datos()
	if !sCT.VinculoAutenticacionActor.CoincideExactamenteCon(sLectura.VinculoAutenticacionActor) {
		t.Fatal("confirmación CT separada de la lectura Personal")
	}
	cruzado := proveedorAltaLigado{base: a, p: ct.PreparacionIncorporacionAplicacionV2{SolicitudPersonal: m.Preparacion.Solicitud, Preparacion: c.fuente.p.PreparacionCT, Contexto: r.Contexto}}
	if _, err := cruzado.ResolverAutoridad(context.Background(), m.Preparacion); !errors.Is(err, ct.ErrComposicionIncorporacionAplicacion) {
		t.Fatal("el perfil de alta sustituyó el contexto de confirmación", err)
	}
	reval.falloAlta = true
	if _, err := a.ResolverAutoridad(context.Background(), m.Preparacion); !errors.Is(err, ErrAutoridadAplicacion) {
		t.Fatal("sesión alta revocada admitida", err)
	}
	// La lectura CT conserva su propia sesión; la caída del alta no concede
	// efectos ni sustituye la lectura por el perfil de Personal.
	_, err = a.AutorizarLecturaIncorporacionV2(context.Background(), lectura)
	autoridadExigir(t, err)
}

func TestAutoridadCompuestaRechazaCapturasCruzadas(t *testing.T) {
	for _, caso := range []string{"sin_alta", "sin_perfiles", "perfil_alta", "perfil_ct", "sesion_compartida", "persona", "cuenta"} {
		t.Run(caso, func(t *testing.T) {
			c, _, reval, res, p := casoAutoridadCompuesta(t)
			switch caso {
			case "sin_alta":
				p.AltaNominal = nil
			case "sin_perfiles":
				p.PerfilesNominales = PerfilesAutoridadAplicacion{}
			case "perfil_alta":
				p.PerfilesNominales.AltaRef = p.Contexto.PerfilActivoRef
			case "perfil_ct":
				p.PerfilesNominales.ConsultaConfirmacionRef = p.PerfilesNominales.AltaRef
			case "sesion_compartida":
				p.AltaNominal.Autenticacion.SesionRef = p.Autenticacion.SesionRef
			case "persona":
				otro, _, sc := autoridadFixtureContexto(t, c.ahora, "b", "b")
				res.alta, p.AltaNominal.Contexto = otro.Resultado, sc
			case "cuenta":
				reval.alta.CuentaRef += "otra"
				reval.alta.CuentaOrdinariaRef = reval.alta.CuentaRef
			}
			if a, err := NuevaAutoridadAplicacion(context.Background(), autoridadFuenteDoble{p: p}, reval, res, c.a.cadena, c.gen, c.reloj); a != nil || err == nil {
				t.Fatalf("captura cruzada admitida: %v", err)
			}
		})
	}
}
