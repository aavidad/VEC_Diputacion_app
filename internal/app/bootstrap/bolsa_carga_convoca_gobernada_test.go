package bootstrap

import (
	"context"
	"errors"
	"testing"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type fuenteCargaConvocaGobernadaPrueba struct {
	instantanea dominiovec.InstantaneaAutorizacion
	consultas   int
}

func (f *fuenteCargaConvocaGobernadaPrueba) ObtenerInstantaneaAutorizacion(
	_ context.Context, _, _ string,
) (dominiovec.InstantaneaAutorizacion, error) {
	f.consultas++
	return f.instantanea, nil
}

func concesionCargaConvocaPrueba() dominiovec.ConcesionRol {
	return dominiovec.ConcesionRol{
		Accion:         puertosbolsa.AccionConfirmarCargaConvoca,
		ModuloID:       puertosbolsa.ModuloCargaConvoca,
		TipoRecurso:    puertosbolsa.TipoRecursoCargaConvoca,
		Finalidades:    []string{puertosbolsa.FinalidadConfirmarCargaConvoca},
		GarantiaMinima: dominiovec.AuthAssuranceHigh,
	}
}

func instantaneaCargaConvocaPublicadaPrueba(t *testing.T) (*politicaBorradorLlamamientoBolsaDesarrollo,
	*autoridadProvisionBolsaPrueba, dominiovec.InstantaneaAutorizacion) {
	t.Helper()
	p, autoridad, datos := politicaProvisionBolsaPrueba(t, 16)
	i := autoridad.leida.instantanea
	i.VersionRol.Version = 23
	i.VersionRol.PublicadaPor = "seguridad:administracion:acto-b1"
	i.VersionRol.Concesiones = append(i.VersionRol.Concesiones, concesionCargaConvocaPrueba())
	i.AsignacionPerfil.VersionRolRef = i.VersionRol.Referencia()
	i.ControlVigenciaVersionRol.VersionRolRef = i.VersionRol.Referencia()
	i.ControlVigenciaVersionRol.ActualizadoPor = i.VersionRol.PublicadaPor
	if err := i.Validar(); err != nil {
		t.Fatal(err)
	}
	if !instantaneaBolsaCargaConvocaCompatible(i, datos, p.soporte, p.reloj.Ahora(), 5) {
		t.Fatal("la concesión gobernada compatible fue rechazada")
	}
	return p, autoridad, i
}

func TestCargaConvocaConsumeVersionGobernadaSinPublicarSemilla(t *testing.T) {
	p, autoridad, i := instantaneaCargaConvocaPublicadaPrueba(t)
	autoridad.leida.instantanea = i
	if err := p.PublicarInicial(context.Background()); err != nil {
		t.Fatal(err)
	}
	if autoridad.publicadas != 0 || autoridad.preparadas != 0 || !p.permiteCargaConvoca() {
		t.Fatalf("el arranque cambió la autoridad o no consumió B1: %+v", autoridad)
	}
	if p.instantanea.VersionRol.Version != 23 || p.instantanea.VersionRol.PublicadaPor != i.VersionRol.PublicadaPor {
		t.Fatal("se sustituyó el acto publicado por la semilla")
	}
	registro := &registroBorradorBolsaPrueba{}
	fuente := &fuenteCargaConvocaGobernadaPrueba{instantanea: i}
	politica, activa, err := politicaCargaConvocaGobernada(context.Background(), fuente, registro, registro,
		validadorMotivoBorradorPoliticaPrueba{}, i, p.reloj.Ahora())
	if err != nil || !activa || !politica.valida() || fuente.consultas != 1 {
		t.Fatalf("B1 no usa fuente central inyectada: activa=%t err=%v", activa, err)
	}
	if _, err := politica.fuente.ObtenerInstantaneaAutorizacion(context.Background(), i.AsignacionPerfil.PrincipalID,
		i.AsignacionPerfil.PerfilActivoRef); err != nil || fuente.consultas != 2 {
		t.Fatalf("la política no reconsulta la autoridad: %v", err)
	}
	fuente.instantanea = clonarInstantaneaAutorizacionPostgreSQLDesarrollo(i)
	fuente.instantanea.VersionRol.Concesiones = fuente.instantanea.VersionRol.Concesiones[:len(fuente.instantanea.VersionRol.Concesiones)-1]
	if _, err := politica.fuente.ObtenerInstantaneaAutorizacion(context.Background(), i.AsignacionPerfil.PrincipalID,
		i.AsignacionPerfil.PerfilActivoRef); !errors.Is(err, puertosvec.ErrFuenteAutorizacionNoDisponible) {
		t.Fatalf("revocación B1 no cerró fuente: %v", err)
	}
	fuente.instantanea = clonarInstantaneaAutorizacionPostgreSQLDesarrollo(i)
	fuente.instantanea.AsignacionPerfil.Ambitos[0].Valores = []string{"unidad_ajena"}
	if _, err := politica.fuente.ObtenerInstantaneaAutorizacion(context.Background(), i.AsignacionPerfil.PrincipalID,
		i.AsignacionPerfil.PerfilActivoRef); !errors.Is(err, puertosvec.ErrFuenteAutorizacionNoDisponible) {
		t.Fatalf("cambio de ámbito no cerró fuente: %v", err)
	}
}

func TestCargaConvocaNiegaConcesionAusenteAlteradaOAsignacionAjena(t *testing.T) {
	p, _, valida := instantaneaCargaConvocaPublicadaPrueba(t)
	datos, err := p.soporte.soporteCanal.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		nombre string
		mutar  func(*dominiovec.InstantaneaAutorizacion)
	}{
		{"sin concesión", func(i *dominiovec.InstantaneaAutorizacion) {
			i.VersionRol.Concesiones = i.VersionRol.Concesiones[:len(i.VersionRol.Concesiones)-1]
		}},
		{"finalidad ajena", func(i *dominiovec.InstantaneaAutorizacion) {
			i.VersionRol.Concesiones[len(i.VersionRol.Concesiones)-1].Finalidades = []string{puertosbolsa.FinalidadCrearBorradorLlamamientoInterno}
		}},
		{"campos añadidos", func(i *dominiovec.InstantaneaAutorizacion) {
			i.VersionRol.Concesiones[len(i.VersionRol.Concesiones)-1].CamposPermitidos = []string{"nombre"}
		}},
		{"perfil ajeno", func(i *dominiovec.InstantaneaAutorizacion) { i.AsignacionPerfil.PerfilActivoRef = "perfil_ajeno" }},
		{"ámbito distinto", func(i *dominiovec.InstantaneaAutorizacion) {
			i.AsignacionPerfil.Ambitos[0].Valores = []string{"unidad_ajena"}
		}},
		{"control retirado", func(i *dominiovec.InstantaneaAutorizacion) {
			i.ControlVigenciaVersionRol.Estado = dominiovec.EstadoControlVigenciaVersionRolRetirada
		}},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			i := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(valida)
			tc.mutar(&i)
			if instantaneaBolsaCargaConvocaCompatible(i, datos, p.soporte, p.reloj.Ahora(), 5) {
				t.Fatal("asignación incompatible admitida")
			}
		})
	}
	registro := &registroBorradorBolsaPrueba{}
	fuente := &fuenteCargaConvocaGobernadaPrueba{instantanea: valida}
	if _, activa, _ := politicaCargaConvocaGobernada(context.Background(), fuente, registro, registro,
		validadorMotivoBorradorPoliticaPrueba{}, valida, valida.AsignacionPerfil.VigenteHasta.Add(time.Second)); activa {
		t.Fatal("asignación caducada admitida")
	}
}
