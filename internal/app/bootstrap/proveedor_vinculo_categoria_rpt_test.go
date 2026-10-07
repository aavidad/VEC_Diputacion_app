package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"

	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type sesionVinculoRPTPrueba struct {
	sesion   SesionVinculoCategoriaRPT
	err      error
	llamadas int
}

func (s *sesionVinculoRPTPrueba) ResolverSesionVinculoCategoriaRPT(context.Context) (SesionVinculoCategoriaRPT, error) {
	s.llamadas++
	return s.sesion, s.err
}

type contextosVinculoRPTPrueba struct {
	resolver func(ctports.SolicitudResolverContextoAutorizacionAltaV3) (ctports.ContextoAutorizacionAltaV3, error)
	perfiles []string
}

func (c *contextosVinculoRPTPrueba) ResolverContextoAutorizacionAltaV3(_ context.Context, s ctports.SolicitudResolverContextoAutorizacionAltaV3) (ctports.ContextoAutorizacionAltaV3, error) {
	c.perfiles = append(c.perfiles, s.PerfilRef)
	return c.resolver(s)
}

func configuracionVinculoRPTPrueba(perfilCT string) ConfiguracionProveedorVinculoCategoriaRPT {
	motivo := vecdomain.ReferenciaEntradaCatalogo{CatalogoID: "motivos_rpt_prueba", CatalogoVersion: 1,
		CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "motivo_0123456789abcdef0123456789abcdef"}
	return ConfiguracionProveedorVinculoCategoriaRPT{
		OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo,
		PerfilCT:        perfilCT, PerfilRPT: "prf_rpt_prueba_1234567890abcdef",
		CatalogoID: "categorias_rpt", ModuloID: "vec.module.puesto_trabajo",
		ConsumidorRPT: ctports.AudienciaConsultarPublicacionCategoriaRPT,
		FinalidadCT:   finalidadVinculoCategoriaRPTCT, FinalidadRPT: finalidadLecturaCategoriaRPT,
		MotivoConsultaCT: motivo, MotivoRegistroCT: motivo, MotivoLecturaRPT: motivo,
	}
}

func TestProveedorVinculoRPTConstructorExigeDescriptoresYAutoridades(t *testing.T) {
	_, err := NuevoProveedorVinculoCategoriaRPT(ConfiguracionProveedorVinculoCategoriaRPT{}, nil, nil, nil, nil, nil, nil, nil)
	if !errors.Is(err, ctports.ErrVinculoCategoriaRPTNoDisponible) {
		t.Fatalf("constructor sin autoridades: %v", err)
	}
	config := configuracionVinculoRPTPrueba("prf_ct_prueba_1234567890abcdef")
	if !config.valida() {
		t.Fatal("configuración nominal de prueba inválida")
	}
	casos := []struct {
		name    string
		cambiar func(*ConfiguracionProveedorVinculoCategoriaRPT)
	}{
		{"sin catálogo", func(c *ConfiguracionProveedorVinculoCategoriaRPT) { c.CatalogoID = "" }},
		{"sin perfil RPT", func(c *ConfiguracionProveedorVinculoCategoriaRPT) { c.PerfilRPT = "" }},
		{"consumidor ajeno", func(c *ConfiguracionProveedorVinculoCategoriaRPT) { c.ConsumidorRPT = "vec.otro" }},
		{"finalidad CT ajena", func(c *ConfiguracionProveedorVinculoCategoriaRPT) { c.FinalidadCT = "otra" }},
	}
	for _, caso := range casos {
		t.Run(caso.name, func(t *testing.T) {
			c := config
			caso.cambiar(&c)
			if c.valida() {
				t.Fatal("aceptó descriptor ajeno")
			}
		})
	}
}

func TestProveedorVinculoRPTConsultaCTNoRequierePerfilRPTYRevalidaSesion(t *testing.T) {
	alta, _, _ := escenarioConsultasRRHHDesarrolloPrueba(t)
	ct := alta.soporte.contexto
	v, err := ct.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	config := configuracionVinculoRPTPrueba(v.PerfilActivoRef)
	refs := ReferenciasSesionVinculoCategoriaRPT{v.AutenticacionRef, v.SesionRef}
	sesiones := &sesionVinculoRPTPrueba{sesion: SesionVinculoCategoriaRPT{CT: refs}}
	contextos := &contextosVinculoRPTPrueba{resolver: func(s ctports.SolicitudResolverContextoAutorizacionAltaV3) (ctports.ContextoAutorizacionAltaV3, error) {
		if s.PerfilRef != v.PerfilActivoRef {
			return ctports.ContextoAutorizacionAltaV3{}, vecdomain.ErrContextoActorNoResuelto
		}
		return ct, nil
	}}
	p := &ProveedorVinculoCategoriaRPT{config: config, sesiones: sesiones, contextos: contextos,
		consultaCT: &confianza.EmisorMaterialAutorizacionAtestadaV3{}, reloj: alta.soporte.reloj}
	if _, err := p.contextoCT(context.Background()); err != nil {
		t.Fatalf("contexto CT vigente: %v", err)
	}
	if sesiones.llamadas != 1 || len(contextos.perfiles) != 1 || contextos.perfiles[0] != config.PerfilCT {
		t.Fatalf("resolución de perfiles: %d %v", sesiones.llamadas, contextos.perfiles)
	}
	// La lectura CT sólo requiere su perfil. La segunda identidad RPT se exige
	// al preparar una lectura RPT o una escritura CT con doble capacidad.
	if _, _, err := p.contextosVigentes(context.Background(), true); !errors.Is(err, ctports.ErrVinculoCategoriaRPTDenegado) {
		t.Fatalf("perfil RPT ausente: %v", err)
	}
	sesiones.sesion.RPT = refs
	if _, _, err := p.contextosVigentes(context.Background(), true); !errors.Is(err, ctports.ErrVinculoCategoriaRPTDenegado) {
		t.Fatalf("perfil RPT revocado: %v", err)
	}
	contextos.resolver = func(s ctports.SolicitudResolverContextoAutorizacionAltaV3) (ctports.ContextoAutorizacionAltaV3, error) {
		if s.PerfilRef == config.PerfilRPT {
			return ctports.ContextoAutorizacionAltaV3{}, errors.New("fuente RPT caída")
		}
		return ct, nil
	}
	if _, _, err := p.contextosVigentes(context.Background(), true); !errors.Is(err, ctports.ErrVinculoCategoriaRPTNoDisponible) {
		t.Fatalf("fuente RPT caída: %v", err)
	}
	sesiones.err = errors.New("fuente de sesión caída")
	if _, err := p.contextoCT(context.Background()); !errors.Is(err, ctports.ErrVinculoCategoriaRPTNoDisponible) {
		t.Fatalf("dependencia caída: %v", err)
	}
	sesiones.err = vecdomain.ErrAutorizacionDenegada
	if _, err := p.contextoCT(context.Background()); !errors.Is(err, ctports.ErrVinculoCategoriaRPTDenegado) {
		t.Fatalf("sesión revocada: %v", err)
	}
}

func TestProveedorVinculoRPTPerfilesYResponsablesDistintosMismaIdentidad(t *testing.T) {
	soporte, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	ct := soporte.contexto
	rpt, err := nuevoContextoReincorporacionTitularDesarrollo(soporte, soporte.reloj.Ahora())
	if err != nil {
		t.Fatal(err)
	}
	vct, _ := ct.Vinculo.Datos()
	vrpt, _ := rpt.Vinculo.Datos()
	if vct.PerfilActivoRef == vrpt.PerfilActivoRef || vct.SesionRef == vrpt.SesionRef {
		t.Fatal("la prueba exige perfiles y sesiones diferentes")
	}
	config := configuracionVinculoRPTPrueba(vct.PerfilActivoRef)
	config.PerfilRPT = vrpt.PerfilActivoRef
	sesiones := &sesionVinculoRPTPrueba{sesion: SesionVinculoCategoriaRPT{
		CT:  ReferenciasSesionVinculoCategoriaRPT{vct.AutenticacionRef, vct.SesionRef},
		RPT: ReferenciasSesionVinculoCategoriaRPT{vrpt.AutenticacionRef, vrpt.SesionRef},
	}}
	contextos := &contextosVinculoRPTPrueba{resolver: func(s ctports.SolicitudResolverContextoAutorizacionAltaV3) (ctports.ContextoAutorizacionAltaV3, error) {
		switch s.PerfilRef {
		case config.PerfilCT:
			return ct, nil
		case config.PerfilRPT:
			return rpt, nil
		default:
			return ctports.ContextoAutorizacionAltaV3{}, vecdomain.ErrContextoActorNoResuelto
		}
	}}
	p := &ProveedorVinculoCategoriaRPT{config: config, sesiones: sesiones, contextos: contextos, reloj: soporte.reloj}
	primero, segundo, err := p.contextosVigentes(context.Background(), true)
	if err != nil || primero.Resultado.Contexto.PersonaRef != segundo.Resultado.Contexto.PersonaRef ||
		primero.Resultado.Contexto.Instantanea.CuentaRef != segundo.Resultado.Contexto.Instantanea.CuentaRef ||
		len(contextos.perfiles) != 2 {
		t.Fatalf("dos perfiles de la misma persona: %v, perfiles=%v", err, contextos.perfiles)
	}
}

func TestProveedorVinculoRPTMaterialPublicacionCincoClavesExactas(t *testing.T) {
	p := ctdomain.PublicacionCategoriaRPT{CatalogoID: "categorias_rpt", ModuloID: "vec.module.puesto_trabajo",
		CatalogoVersion: 5, CatalogoHuella: strings.Repeat("a", 64), CategoriaID: "categoria_01", CategoriaClave: "categoria_01"}
	correcto := []byte(`{"version": 5, "modulo_id": "vec.module.puesto_trabajo", "catalogo_id": "categorias_rpt", "categoria_id": "categoria_01", "huella_sha256": "` + strings.Repeat("a", 64) + `"}`)
	if !materialPublicacionRPTValido(correcto, p) {
		t.Fatal("rechazó las cinco claves exactas")
	}
	for _, material := range [][]byte{
		[]byte(`{"catalogo_id":"otro","modulo_id":"vec.module.puesto_trabajo","version":5,"categoria_id":"categoria_01","huella_sha256":"` + strings.Repeat("a", 64) + `"}`),
		[]byte(`{"catalogo_id":"categorias_rpt","modulo_id":"vec.module.puesto_trabajo","version":5,"categoria_id":"categoria_01","huella_sha256":"` + strings.Repeat("a", 64) + `","rol":"administrador"}`),
		[]byte(`{"catalogo_id":"categorias_rpt","modulo_id":"vec.module.puesto_trabajo","version":"5","categoria_id":"categoria_01","huella_sha256":"` + strings.Repeat("a", 64) + `"}`),
	} {
		if materialPublicacionRPTValido(material, p) {
			t.Fatalf("aceptó material ajeno: %s", material)
		}
	}
}
