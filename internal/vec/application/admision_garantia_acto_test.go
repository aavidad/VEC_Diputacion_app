package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	core "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/pruebas"
)

var ahoraAdmisionPrueba = time.Date(2026, 10, 7, 10, 0, 0, 0, time.UTC)

const (
	personaAdmisionPrueba = "per_0123456789abcdefghijkl"
	perfilAdmisionPrueba  = "prf_0123456789abcdefghijkl"
)

type relojAdmisionPrueba struct{ ahora time.Time }

func (r *relojAdmisionPrueba) Ahora() time.Time { return r.ahora }

type entornoAdmisionPrueba struct {
	vinculo    core.VinculoAutenticacionActorV2
	resultado  core.ResultadoContextoActorRegistradoV2
	snapshot   core.InstantaneaAutorizacion
	config     ConfiguracionAdmisionGarantiaFirmaVecDesarrollo
	descriptor DescriptorAdmisionGarantiaActo
	reloj      *relojAdmisionPrueba
}

func nuevoEntornoAdmisionPrueba(t *testing.T) *entornoAdmisionPrueba {
	t.Helper()
	reloj := &relojAdmisionPrueba{ahora: ahoraAdmisionPrueba}
	resultado, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(ahoraAdmisionPrueba.Add(time.Minute),
		personaAdmisionPrueba, perfilAdmisionPrueba, core.AuthMethodCertificate, core.AuthAssuranceSubstantial)
	if err != nil {
		t.Fatal(err)
	}
	datosVinculo, err := vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	rol := core.VersionRol{RolID: "firma_vec_desarrollo", Version: 1, Nombre: "Firma VEC desarrollo",
		Estado: core.EstadoVersionRolPublicada, PublicadaPor: "seguridad",
		PublicadaEn: ahoraAdmisionPrueba.Add(-2 * time.Hour),
		Concesiones: []core.ConcesionRol{{Accion: accionAdmisionFirmaVec,
			ModuloID: moduloAdmisionFirmaVec, TipoRecurso: tipoAdmisionFirmaVec,
			Finalidades: []string{finalidadAdmisionFirmaVec}, GarantiaMinima: core.AuthAssuranceSubstantial}},
	}
	asignacion := core.AsignacionPerfil{AsignacionID: "asignacion_firma_desarrollo", Version: 1,
		PerfilActivoRef: perfilAdmisionPrueba, PrincipalID: personaAdmisionPrueba,
		VersionRolRef: rol.Referencia(), Estado: core.EstadoAsignacionPerfilActiva,
		Ambitos:    []core.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{"org_0123456789abcdefghijkl"}}},
		EmitidaPor: "administracion", EmitidaEn: ahoraAdmisionPrueba.Add(-90 * time.Minute),
		VigenteDesde: ahoraAdmisionPrueba.Add(-time.Hour), VigenteHasta: ahoraAdmisionPrueba.Add(time.Hour)}
	control := core.ControlVigenciaVersionRol{VersionRolRef: rol.Referencia(), Revision: 1,
		Estado: core.EstadoControlVigenciaVersionRolHabilitada, ActualizadoPor: "administracion",
		ActualizadoEn: ahoraAdmisionPrueba.Add(-time.Hour)}
	catalogoSHA, err := core.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := core.InstantaneaAutorizacion{AsignacionPerfil: asignacion, VersionRol: rol,
		ControlVigenciaVersionRol: control, RevisionCatalogoPoliticas: 1,
		CatalogoPoliticasHuellaSHA256: catalogoSHA}
	if err := snapshot.Validar(); err != nil {
		t.Fatalf("instantánea sintética inválida: %v", err)
	}
	rolSHA, err := rol.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	controlSHA, err := control.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	politica := PoliticaPrivadaAdmisionFirmaDesarrollo{
		Referencia: "politica:vec:firma:desarrollo-un-factor:v1", Version: 1,
		HuellaSHA256: strings.Repeat("a", 64), RetiradaEn: ahoraAdmisionPrueba.Add(30 * time.Minute),
		PoliticaAutenticacionRef:    datosVinculo.PoliticaGarantiaRef,
		PoliticaAutenticacionSHA256: datosVinculo.PoliticaGarantiaHuellaSHA256,
		RolVersionRef:               rol.Referencia(), RolSHA256: rolSHA,
		ControlRevision: control.Revision, ControlSHA256: controlSHA,
	}
	return &entornoAdmisionPrueba{vinculo: vinculo, resultado: resultado, snapshot: snapshot,
		config: ConfiguracionAdmisionGarantiaFirmaVecDesarrollo{
			EntornoConfiable: EntornoAdmisionFirmaDesarrollo, Politica: politica, Reloj: reloj},
		descriptor: DescriptorAdmisionGarantiaActo{
			Accion: accionAdmisionFirmaVec, Audiencia: audienciaAdmisionFirmaVec,
			ModuloID: moduloAdmisionFirmaVec, TipoRecurso: tipoAdmisionFirmaVec,
			Finalidad:            finalidadAdmisionFirmaVec,
			RecursoRef:           "operacion-firma-vec-ct:0123456789abcdefghijkl",
			MaterialHuellaSHA256: strings.Repeat("b", 64),
		}, reloj: reloj}
}

func TestAdmisionGarantiaFirmaDevConservaGarantiaRealYActoExacto(t *testing.T) {
	e := nuevoEntornoAdmisionPrueba(t)
	gate, err := NuevaAdmisionGarantiaFirmaVecDesarrollo(e.config)
	if err != nil {
		t.Fatal(err)
	}
	evidencia, err := gate.Admitir(context.Background(), e.vinculo, e.resultado, e.snapshot, e.descriptor)
	if err != nil {
		t.Fatal(err)
	}
	resumen, err := evidencia.Resumen()
	if err != nil || resumen.GarantiaReal != core.AuthAssuranceSubstantial ||
		resumen.PoliticaRef != e.config.Politica.Referencia || resumen.PoliticaVersion != 1 ||
		resumen.PoliticaSHA256 != e.config.Politica.HuellaSHA256 ||
		resumen.RolVersionRef != e.snapshot.VersionRol.Referencia() ||
		resumen.ControlRevision != e.snapshot.ControlVigenciaVersionRol.Revision ||
		resumen.RecursoRef != e.descriptor.RecursoRef ||
		resumen.MaterialHuellaSHA256 != e.descriptor.MaterialHuellaSHA256 ||
		!resumen.VigenteHasta.After(ahoraAdmisionPrueba) {
		t.Fatalf("evidencia de garantía incompleta: %v", err)
	}
	if core.AuthAssuranceSubstantial.Cumple(core.AuthAssuranceHigh) {
		t.Fatal("la admisión temporal elevó la garantía observada")
	}
	if err := gate.ValidarEvidencia(context.Background(), evidencia, e.vinculo, e.resultado, e.snapshot, e.descriptor); err != nil {
		t.Fatalf("la evidencia original no se cotejó: %v", err)
	}
	e.reloj.ahora = ahoraAdmisionPrueba.Add(time.Second)
	if err := gate.ValidarEvidencia(context.Background(), evidencia, e.vinculo, e.resultado, e.snapshot, e.descriptor); err != nil {
		t.Fatalf("el reloj posterior rechazó la misma evidencia vigente: %v", err)
	}
	e.reloj.ahora = ahoraAdmisionPrueba.Add(-time.Second)
	if err := gate.ValidarEvidencia(context.Background(), evidencia, e.vinculo, e.resultado, e.snapshot, e.descriptor); !errors.Is(err, ErrAdmisionGarantiaActoDenegada) {
		t.Fatalf("el reloj anterior a la admisión reutilizó la evidencia: %v", err)
	}
	e.reloj.ahora = ahoraAdmisionPrueba.Add(time.Second)
	if err := evidencia.ExigirVentanaDecisionV3(ahoraAdmisionPrueba, ahoraAdmisionPrueba.Add(time.Second)); err != nil {
		t.Fatalf("ventana V3 dentro de la excepción rechazada: %v", err)
	}
	if err := evidencia.ExigirVentanaDecisionV3(ahoraAdmisionPrueba, resumen.VigenteHasta.Add(time.Microsecond)); !errors.Is(err, ErrAdmisionGarantiaActoDenegada) {
		t.Fatalf("decisión V3 más larga que la excepción: %v", err)
	}
	for _, cambiar := range []func(*DescriptorAdmisionGarantiaActo){
		func(d *DescriptorAdmisionGarantiaActo) {
			d.RecursoRef = "operacion-firma-vec-ct:otra0123456789abcdefghijkl"
		},
		func(d *DescriptorAdmisionGarantiaActo) { d.MaterialHuellaSHA256 = strings.Repeat("c", 64) },
	} {
		otro := e.descriptor
		cambiar(&otro)
		if err := gate.ValidarEvidencia(context.Background(), evidencia, e.vinculo, e.resultado, e.snapshot, otro); !errors.Is(err, ErrAdmisionGarantiaActoDenegada) {
			t.Fatalf("evidencia reutilizada en otro material o recurso: %v", err)
		}
	}
	if _, err := json.Marshal(evidencia); !errors.Is(err, ErrAdmisionGarantiaActoDenegada) ||
		strings.Contains(fmt.Sprintf("%+v", evidencia), personaAdmisionPrueba) {
		t.Fatalf("la evidencia salió de la frontera privada: %v", err)
	}
}

func TestAdmisionGarantiaFirmaDevFallaCerradoSinPoliticaOEnProduccion(t *testing.T) {
	e := nuevoEntornoAdmisionPrueba(t)
	for _, caso := range []struct {
		nombre string
		mutar  func(*ConfiguracionAdmisionGarantiaFirmaVecDesarrollo)
	}{
		{"produccion", func(c *ConfiguracionAdmisionGarantiaFirmaVecDesarrollo) { c.EntornoConfiable = "produccion" }},
		{"sin politica", func(c *ConfiguracionAdmisionGarantiaFirmaVecDesarrollo) {
			c.Politica = PoliticaPrivadaAdmisionFirmaDesarrollo{}
		}},
		{"politica retirada", func(c *ConfiguracionAdmisionGarantiaFirmaVecDesarrollo) { c.Politica.RetiradaEn = ahoraAdmisionPrueba }},
		{"huella privada vacia", func(c *ConfiguracionAdmisionGarantiaFirmaVecDesarrollo) { c.Politica.HuellaSHA256 = "" }},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			config := e.config
			caso.mutar(&config)
			if _, err := NuevaAdmisionGarantiaFirmaVecDesarrollo(config); !errors.Is(err, ErrAdmisionGarantiaActoDenegada) {
				t.Fatalf("modo sin aprobación explícita admitido: %v", err)
			}
		})
	}
}

func TestAdmisionGarantiaFirmaDevDeniegaRolHighYDesviaciones(t *testing.T) {
	for _, caso := range []struct {
		nombre string
		mutar  func(*entornoAdmisionPrueba)
	}{
		{"rol exige HIGH", func(e *entornoAdmisionPrueba) {
			e.snapshot.VersionRol.Concesiones[0].GarantiaMinima = core.AuthAssuranceHigh
			e.config.Politica.RolSHA256, _ = e.snapshot.VersionRol.HuellaSHA256()
		}},
		{"control retirado", func(e *entornoAdmisionPrueba) {
			c := &e.snapshot.ControlVigenciaVersionRol
			c.Estado, c.ActoRef, c.MotivoCodigo = core.EstadoControlVigenciaVersionRolRetirada, "acto:retirada", "baja"
			e.config.Politica.ControlSHA256, _ = c.HuellaSHA256()
		}},
		{"asignacion vencida", func(e *entornoAdmisionPrueba) {
			e.snapshot.AsignacionPerfil.VigenteHasta = ahoraAdmisionPrueba
		}},
		{"perfil ajeno", func(e *entornoAdmisionPrueba) {
			e.snapshot.AsignacionPerfil.PerfilActivoRef = "prf_ajeno0123456789abcdefghijkl"
		}},
		{"politica autenticacion ajena", func(e *entornoAdmisionPrueba) {
			e.config.Politica.PoliticaAutenticacionSHA256 = strings.Repeat("d", 64)
		}},
		{"lectura", func(e *entornoAdmisionPrueba) {
			e.descriptor.Accion = "contratacion_temporal.documento.firmas_r5_v2.consultar"
		}},
		{"firma externa", func(e *entornoAdmisionPrueba) {
			e.descriptor.Audiencia = "vec_contratacion_temporal.firma_externa.v2"
		}},
		{"material sin huella", func(e *entornoAdmisionPrueba) {
			e.descriptor.MaterialHuellaSHA256 = ""
		}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			e := nuevoEntornoAdmisionPrueba(t)
			caso.mutar(e)
			gate, err := NuevaAdmisionGarantiaFirmaVecDesarrollo(e.config)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := gate.Admitir(context.Background(), e.vinculo, e.resultado, e.snapshot, e.descriptor); !errors.Is(err, ErrAdmisionGarantiaActoDenegada) {
				t.Fatalf("la desviación fue admitida: %v", err)
			}
		})
	}
}

func TestAdmisionGarantiaFirmaDevRetiradaYCancelacionNoFiltran(t *testing.T) {
	e := nuevoEntornoAdmisionPrueba(t)
	gate, err := NuevaAdmisionGarantiaFirmaVecDesarrollo(e.config)
	if err != nil {
		t.Fatal(err)
	}
	e.reloj.ahora = e.config.Politica.RetiradaEn
	if _, err := gate.Admitir(context.Background(), e.vinculo, e.resultado, e.snapshot, e.descriptor); !errors.Is(err, ErrAdmisionGarantiaActoDenegada) {
		t.Fatalf("retirada no bloqueó admisión: %v", err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := gate.Admitir(ctx, e.vinculo, e.resultado, e.snapshot, e.descriptor); !errors.Is(err, context.Canceled) || err.Error() != ErrAdmisionGarantiaActoDenegada.Error() {
		t.Fatalf("cancelación no conservada o mensaje no opaco: %v", err)
	}
}

func TestAdmisionGarantiaFirmaDevDeniegaGarantiaBajaReal(t *testing.T) {
	e := nuevoEntornoAdmisionPrueba(t)
	resultado, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(ahoraAdmisionPrueba.Add(time.Minute),
		personaAdmisionPrueba, perfilAdmisionPrueba, core.AuthMethodCertificate, core.AuthAssuranceLow)
	if err != nil {
		t.Fatal(err)
	}
	datos, err := vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	e.config.Politica.PoliticaAutenticacionRef = datos.PoliticaGarantiaRef
	e.config.Politica.PoliticaAutenticacionSHA256 = datos.PoliticaGarantiaHuellaSHA256
	gate, err := NuevaAdmisionGarantiaFirmaVecDesarrollo(e.config)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := gate.Admitir(context.Background(), vinculo, resultado, e.snapshot, e.descriptor); !errors.Is(err, ErrAdmisionGarantiaActoDenegada) {
		t.Fatalf("la garantía baja se elevó por la política DEV: %v", err)
	}
}
