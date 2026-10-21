package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"testing"

	bp "vec-diputacion-granada/internal/modules/bolsa/ports"
	httpct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type escenarioSesionAnclajePropuestaPrueba struct {
	autoridad *autoridadIncorporacionPersonalB2
	perfil    *perfilFijoCTDesarrollo
	sesion    *sesionNominalIncorporacion
	entorno   *entornoSesionConsultaPrueba
	lector    *lectorAsignacionB2ContadoPrueba
	contexto  context.Context
	selector  bp.SelectorAceptacionNativaCT
}

func nuevaSesionAnclajePropuestaPrueba(t *testing.T) escenarioSesionAnclajePropuestaPrueba {
	t.Helper()
	e := nuevaSesionConsultaPrueba(t)
	v, err := e.soporte.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	refs := ReferenciasCTIncorporacionDesarrollo{
		PrincipalV3Ref: v.PrincipalID, PerfilV3Ref: v.PerfilActivoRef,
		OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo,
		UnidadRef:       "unidad:desarrollo:rrhh", ActorRef: v.PrincipalID,
	}
	perfiles := &perfilesNominalesIncorporacion{soporte: e.soporte}
	config := &archivoIncorporacionPersonalB2{
		Protocolo: "personal_b2_v1", OrganismoRef: "organismo:prueba",
		CatalogoRPTID: "categorias_rpt", ModuloRPTID: "personal",
	}
	if err := extenderPerfilesNominalesB2(perfiles, refs, config, e.reloj.Ahora()); err != nil {
		t.Fatal(err)
	}
	perfil := perfiles.b2[bp.AccionConsultaAnclajeAceptacionNativaCT]
	if perfil == nil {
		t.Fatal("falta perfil nominal de anclaje")
	}
	perfil.contextoEsperadoRegistrado = perfil.contexto.Resultado
	e.resolutor.base = perfil.contexto.Resultado
	fija, err := nuevaSesionReincorporacionTitularDesarrollo(e.p, perfil.contexto.Resultado)
	if err != nil {
		t.Fatal(err)
	}
	proveedor := fija.(*proveedorSesionConsultaRRHHDesarrollo)
	// En el montaje real lo fija configurarSesionesNominalesIncorporacion.
	proveedor.anclajePropuesta = perfiles
	sesion := &sesionNominalIncorporacion{proveedor: proveedor}
	perfil.sesionOperativa = sesion
	publicador := &publicadorPermisoIncorporacionPrueba{publicadas: map[string]instantaneaPublicadaDesarrollo{
		perfil.perfilRef(): {instantanea: perfil.plantilla, actoAsignacion: actoAsignacionPerfilFijoCTDesarrollo},
	}}
	lector := &lectorAsignacionB2ContadoPrueba{publicadorPermisoIncorporacionPrueba: publicador, base: publicador}
	e.soporte.autoridadAsignaciones = lector

	propuesta := nuevaPropuestaCoordinadorPrueba(t)
	preparacion, err := prepararReferenciasLlamamientoDesarrollo(
		propuesta.lector.expediente, propuesta.registro.antecedente.SeleccionClave)
	if err != nil {
		t.Fatal(err)
	}
	x := bp.SelectorAceptacionNativaCT{
		OrganizacionRef: preparacion.expediente.Fiscalizado.OrganizacionRef,
		ExpedienteRef:   preparacion.expediente.Fiscalizado.Referencia,
		ClaveCT:         preparacion.clave, UnidadRef: preparacion.unidad,
		CategoriaRef: preparacion.categoria, NecesidadRef: preparacion.necesidad,
		BolsaRef: "bolsa:una", LlamamientoRef: "llamamiento:uno",
		ReciboEmisionRef: "recibo:emision:uno", ReciboRespuestaRef: "recibo:respuesta:uno",
	}
	ctx := contextoRutaCoberturaDesarrolloPrueba(e.soporte, e.principal, httpct.RutaPropuestaFormalizacion)
	descriptor, ok := e.p.fronteras.resolver(http.MethodPost, httpct.RutaPropuestaFormalizacion)
	if !ok || descriptor.admitePerfil(perfil.perfilRef()) {
		t.Fatal("el descriptor público CT debe conservar solo su perfil declarado")
	}
	ctx = context.WithValue(ctx, claveFronteraSeguridadComunDesarrollo{}, fronteraSeguridadComunDesarrollo{
		metodo: http.MethodPost, ruta: httpct.RutaPropuestaFormalizacion,
		superficie: superficieInternaSeguridadComunDesarrollo,
		catalogo:   e.p.fronteras, descriptor: descriptor,
	})
	ctx = context.WithValue(ctx, claveIncorporacionV2Desarrollo{}, e.soporte.sello)
	ctx = context.WithValue(ctx, claveContextosNominalesIncorporacion{}, &capturaContextosNominalesIncorporacion{})
	ctx = context.WithValue(ctx, clavePreparacionLlamamientoDesarrollo{}, preparacion)
	ctx = context.WithValue(ctx, claveAnclajeNativoPropuestaCT{}, x)
	return escenarioSesionAnclajePropuestaPrueba{
		autoridad: &autoridadIncorporacionPersonalB2{perfiles: perfiles, reloj: e.reloj},
		perfil:    perfil, sesion: sesion, entorno: e, lector: lector, contexto: ctx, selector: x,
	}
}

func TestAnclajePropuestaSesionaPerfilBolsaSinAmpliarDescriptorCT(t *testing.T) {
	s := nuevaSesionAnclajePropuestaPrueba(t)
	if !s.autoridad.anclajeNativoPropuestaPermitido(s.contexto, bp.AccionConsultaAnclajeAceptacionNativaCT) {
		t.Fatal("la autoridad B113 no reconoció la propuesta exacta")
	}
	if _, err := s.sesion.ResolverContexto(s.contexto); !errors.Is(err, ct.ErrAutorizacionDenegada) {
		t.Fatalf("sesión directa sin ligadura privada: %v", err)
	}
	if len(s.entorno.registro.altas) != 0 {
		t.Fatal("la sesión directa creó una sesión")
	}
	actual, err := s.autoridad.contexto(s.contexto, bp.AccionConsultaAnclajeAceptacionNativaCT)
	if err != nil {
		t.Fatalf("autoridad B113 admitida, sesión nominal denegada: %v", err)
	}
	if actual.Resultado.Contexto.PerfilActivoRef != s.perfil.perfilRef() ||
		len(s.entorno.registro.altas) != 1 || s.entorno.revalidador.llamadas != 1 ||
		s.entorno.resolutor.llamadas != 1 || s.lector.lecturas != 1 {
		t.Fatal("anclaje no consumió sesión y asignación nominal exactas")
	}
}

func TestAnclajePropuestaSesionDeniegaFronterasAjenas(t *testing.T) {
	for _, caso := range []string{"otra_accion", "otro_perfil", "otra_ruta", "otra_unidad", "otra_clave", "otro_expediente", "otro_sello", "otro_propietario", "sin_asignacion"} {
		t.Run(caso, func(t *testing.T) {
			s := nuevaSesionAnclajePropuestaPrueba(t)
			ctx := s.contexto
			accion := bp.AccionConsultaAnclajeAceptacionNativaCT
			x := s.selector
			switch caso {
			case "otra_accion":
				accion = bp.AccionConsultaPersonaAceptacionNativaCT
			case "otro_perfil":
				s.sesion.proveedor.base = s.entorno.p.base
			case "otra_ruta":
				capacidad := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
				capacidad.ruta = httpct.RutaPropuestaCobertura
				ctx = context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad)
			case "otra_unidad":
				x.UnidadRef = "unidad:ajena"
			case "otra_clave":
				x.ClaveCT = "clave:ajena"
			case "otro_expediente":
				x.ExpedienteRef = "expediente:ajeno"
			case "otro_sello":
				ctx = context.WithValue(ctx, claveIncorporacionV2Desarrollo{}, new(struct{}))
			case "otro_propietario":
				holder := ctx.Value(claveContextosNominalesIncorporacion{}).(*capturaContextosNominalesIncorporacion)
				holder.propietario = &perfilesNominalesIncorporacion{}
			case "sin_asignacion":
				delete(s.lector.base.publicadas, s.perfil.perfilRef())
			}
			ctx = context.WithValue(ctx, claveAnclajeNativoPropuestaCT{}, x)
			if _, err := s.autoridad.contexto(ctx, accion); !errors.Is(err, ct.ErrAutorizacionDenegada) {
				t.Fatalf("frontera ajena admitida: %v", err)
			}
			if caso != "sin_asignacion" && len(s.entorno.registro.altas) != 0 {
				t.Fatal("frontera ajena creó una sesión")
			}
		})
	}
}
