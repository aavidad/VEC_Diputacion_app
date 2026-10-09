package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/adapters/httpapi"
	"vec-diputacion-granada/internal/vec/adapters/postgres"
	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type autorizadorNuloGobiernoRPT struct{}

func (*autorizadorNuloGobiernoRPT) EmitirMaterialAutorizacionAtestadaV3(context.Context, domain.SolicitudAutorizacionLigadaV3, domain.ResultadoContextoActorRegistradoV2) (domain.DecisionAutorizacionLigadaV3, ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, ports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	return domain.DecisionAutorizacionLigadaV3{}, ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, errors.New("sin autoridad")
}

type sesionNulaGobiernoRPT struct{}

func (*sesionNulaGobiernoRPT) ResolverCredenciales(context.Context) (application.CredencialesGobiernoCategoriaRPT, error) {
	return application.CredencialesGobiernoCategoriaRPT{}, errors.New("sin sesion")
}

type auditorNuloGobiernoRPT struct{}

func (*auditorNuloGobiernoRPT) RegistrarRechazoGobiernoCategoriaRPT(context.Context, httpapi.RechazoGobiernoCategoriaRPTInterno) error {
	return errors.New("sin auditoria")
}

type relojNuloGobiernoRPT struct{}

func (*relojNuloGobiernoRPT) Ahora() time.Time { return time.Time{} }

func dependenciasGobiernoRPTPrueba() DependenciasGobiernoCategoriaRPTInterno {
	bytes := []byte("fuente de prueba")
	suma := sha256.Sum256(bytes)
	desde := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	hasta := desde.Add(24 * time.Hour)
	fuente := postgres.FuenteGobiernoCategoriaRPT{
		Bytes: bytes, SHA256: hex.EncodeToString(suma[:]), FuenteRef: "fuente:ejercicio:rpt",
		Clase: "ejercicio", ProcedenciaRef: "procedencia:prueba", CustodiaRef: "custodia:prueba",
		OrganizacionRef: "organizacion:prueba", VigenteDesde: desde, VigenteHasta: hasta,
	}
	return DependenciasGobiernoCategoriaRPTInterno{
		Pool: &pgxpool.Pool{}, Descriptor: ports.DescriptorCatalogoRPT{CatalogoID: "rpt", ModuloID: "rpt"},
		OrganizacionRef: fuente.OrganizacionRef, Fuente: fuente,
		FuenteAdmitida: FuenteAdmitidaGobiernoCategoriaRPT{
			CatalogoID: "rpt", ModuloID: "rpt",
			SHA256: fuente.SHA256, FuenteRef: fuente.FuenteRef, Clase: fuente.Clase,
			ProcedenciaRef: fuente.ProcedenciaRef, CustodiaRef: fuente.CustodiaRef,
			OrganizacionRef: fuente.OrganizacionRef, VigenteDesde: desde, VigenteHasta: hasta,
		},
		Autorizador: &autorizadorNuloGobiernoRPT{}, AutoridadSesion: &sesionNulaGobiernoRPT{},
		AuditorRechazos: &auditorNuloGobiernoRPT{}, Reloj: &relojNuloGobiernoRPT{},
		VersionesRol: application.VersionesRolGobiernoCategoriaRPT{Preparacion: "rol:preparacion:v1", Revision: "rol:revision:v1"},
	}
}

func TestComponerGobiernoCategoriaRPTInternoDeniegaDependenciasAusentes(t *testing.T) {
	casos := map[string]func(*DependenciasGobiernoCategoriaRPTInterno){
		"pool":        func(d *DependenciasGobiernoCategoriaRPTInterno) { d.Pool = nil },
		"autorizador": func(d *DependenciasGobiernoCategoriaRPTInterno) { d.Autorizador = (*autorizadorNuloGobiernoRPT)(nil) },
		"sesion":      func(d *DependenciasGobiernoCategoriaRPTInterno) { d.AutoridadSesion = nil },
		"auditor":     func(d *DependenciasGobiernoCategoriaRPTInterno) { d.AuditorRechazos = nil },
		"reloj":       func(d *DependenciasGobiernoCategoriaRPTInterno) { d.Reloj = nil },
		"roles_iguales": func(d *DependenciasGobiernoCategoriaRPTInterno) {
			d.VersionesRol.Revision = d.VersionesRol.Preparacion
		},
	}
	for nombre, cambiar := range casos {
		t.Run(nombre, func(t *testing.T) {
			d := dependenciasGobiernoRPTPrueba()
			cambiar(&d)
			h, err := ComponerGobiernoCategoriaRPTInterno(d)
			if h != nil || !errors.Is(err, ports.ErrGobiernoCategoriaRPTNoDisponible) {
				t.Fatalf("handler=%v error=%v", h, err)
			}
		})
	}
}

func TestComponerGobiernoCategoriaRPTInternoRechazaFuenteYAlcanceIncoherentes(t *testing.T) {
	casos := map[string]func(*DependenciasGobiernoCategoriaRPTInterno){
		"catalogo":      func(d *DependenciasGobiernoCategoriaRPTInterno) { d.Descriptor.CatalogoID = "" },
		"otro_catalogo": func(d *DependenciasGobiernoCategoriaRPTInterno) { d.FuenteAdmitida.CatalogoID = "otro" },
		"organizacion":  func(d *DependenciasGobiernoCategoriaRPTInterno) { d.Fuente.OrganizacionRef = "otra-organizacion" },
		"clase":         func(d *DependenciasGobiernoCategoriaRPTInterno) { d.FuenteAdmitida.Clase = "rpt_legal" },
		"custodia":      func(d *DependenciasGobiernoCategoriaRPTInterno) { d.Fuente.CustodiaRef = "otra-custodia" },
		"vigencia":      func(d *DependenciasGobiernoCategoriaRPTInterno) { d.Fuente.VigenteHasta = d.Fuente.VigenteDesde },
		"huella": func(d *DependenciasGobiernoCategoriaRPTInterno) {
			d.Fuente.SHA256 = "invalida"
			d.FuenteAdmitida.SHA256 = "invalida"
		},
	}
	for nombre, cambiar := range casos {
		t.Run(nombre, func(t *testing.T) {
			d := dependenciasGobiernoRPTPrueba()
			cambiar(&d)
			h, err := ComponerGobiernoCategoriaRPTInterno(d)
			if h != nil || err == nil {
				t.Fatalf("handler=%v error=%v", h, err)
			}
		})
	}
}
