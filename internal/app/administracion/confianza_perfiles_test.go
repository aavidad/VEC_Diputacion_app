package administracion

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type relojConfianzaPerfilesPrueba struct{ ahora time.Time }

func (r *relojConfianzaPerfilesPrueba) Ahora() time.Time { return r.ahora }

type firmanteConfianzaPerfilesPrueba struct{}

func (*firmanteConfianzaPerfilesPrueba) FirmarAtestacionAutorizacionV3(context.Context, ports.SolicitudFirmaAtestacionAutorizacionV3) (ports.ResultadoFirmaAtestacionAutorizacionV3, error) {
	panic("el constructor no debe firmar")
}

func escenarioConfianzaPerfilesPrueba(t *testing.T) (ConfiguracionConfianzaPerfilesV3, DependenciasConfianzaPerfilesV3) {
	t.Helper()
	ahora := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	publica := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{7}, ed25519.SeedSize)).Public().(ed25519.PublicKey)
	r := MaterialRaizPerfilesV3{ClaveID: "clave:admin:prueba", Audiencia: "vec:admin:prueba:atestacion:v3", Version: 1, Publica: publica, Estado: confianza.EstadoClaveAtestacionAutorizacionV3Activa, ValidaDesde: ahora.Add(-time.Hour), ValidaHasta: ahora.Add(time.Hour)}
	raiz, err := confianza.NuevaRaizPublicaAtestacionAutorizacionV3EdDSA(r.ClaveID, r.Version, r.Publica, r.Audiencia, r.Estado, r.ValidaDesde, r.ValidaHasta, r.RevocadaEn)
	if err != nil {
		t.Fatal(err)
	}
	g := GobiernoConfianzaPerfilesV3{Revision: "configuracion:admin:prueba", Secuencia: 1, PublicadaEn: ahora.Add(-time.Minute), ExpiraEn: ahora.Add(time.Hour)}
	c, err := confianza.NuevaConfiguracionConfianzaAtestacionAutorizacionV3(g.Revision, g.Secuencia, g.PublicadaEn, g.ExpiraEn, raiz)
	if err != nil {
		t.Fatal(err)
	}
	g.HuellaSHA256, err = c.HuellaSHA256ParaGobierno()
	if err != nil {
		t.Fatal(err)
	}
	cfg := ConfiguracionConfianzaPerfilesV3{Cabecera: domain.CabeceraAtestacionAutorizacionV3{FormatoVersion: domain.VersionFormatoAtestacionAutorizacionV3, Suite: confianza.SuiteAtestacionAutorizacionV3COSEEdDSA, ClaveID: r.ClaveID, Audiencia: r.Audiencia}, Raiz: r, Gobierno: g}
	for i, audiencia := range []string{AudienciaPerfilesOrdinarioV3, AudienciaPerfilesPropuestaV3, AudienciaPerfilesCierreV3, AudienciaPerfilesConsultaV3, AudienciaPerfilesCapacidadesV3, AudienciaPerfilesBuscarPersonasV3, AudienciaPerfilesPersonaV3, AudienciaPerfilesRolesV3, AudienciaPerfilesPropuestasV3, AudienciaPerfilesPropuestaLecturaV3, AudienciaPerfilesReciboV3} {
		cfg.EntradasCapacidad = append(cfg.EntradasCapacidad, MaterialCapacidadPerfilesV3{Audiencia: audiencia, ClaveID: fmt.Sprintf("clave:admin:capacidad:%d", i), EmisorID: "emisor:admin:prueba", HuellaGobierno: strings.Repeat("a", 64), Version: 1, RevisionGobierno: 1, Material: bytes.Repeat([]byte{byte(i + 1)}, 32), ValidaDesde: r.ValidaDesde, ValidaHasta: r.ValidaHasta, Estado: confianza.EstadoClaveHMACCapacidadAtestacionV3Emision})
	}
	// Pools sin abrir: si la construcción consulta o escribe, la prueba falla.
	deps := DependenciasConfianzaPerfilesV3{PoolFuente: new(pgxpool.Pool), PoolRegistro: new(pgxpool.Pool), PoolMotivos: new(pgxpool.Pool), CatalogoMotivosID: "motivos_admin_prueba", Firmante: new(firmanteConfianzaPerfilesPrueba), Reloj: &relojConfianzaPerfilesPrueba{ahora}, Generador: seguridad.GeneradorReferenciasCriptograficas{}, VigenciaDecision: time.Second}
	return cfg, deps
}

func TestConfianzaPerfilesV3ConstruyeCadenaRealSinEfectos(t *testing.T) {
	cfg, deps := escenarioConfianzaPerfilesPrueba(t)
	f, err := NuevaConfianzaPerfilesV3(cfg, deps)
	if err != nil || f.Fuente == nil || len(f.Emisores) != 11 {
		t.Fatalf("fabrica no disponible: %v", err)
	}
	for _, entrada := range cfg.EntradasCapacidad {
		if f.Emisores[entrada.Audiencia] == nil {
			t.Fatal("audiencia sin emisor")
		}
	}
}

func TestConfianzaPerfilesV3RechazaGobiernoAudienciasYPoolsAjenos(t *testing.T) {
	casos := map[string]func(*ConfiguracionConfianzaPerfilesV3, *DependenciasConfianzaPerfilesV3){
		"audiencia ajena": func(c *ConfiguracionConfianzaPerfilesV3, _ *DependenciasConfianzaPerfilesV3) {
			c.EntradasCapacidad[0].Audiencia = "vec:bolsa:ajena"
		},
		"audiencia repetida": func(c *ConfiguracionConfianzaPerfilesV3, _ *DependenciasConfianzaPerfilesV3) {
			c.EntradasCapacidad[0].Audiencia = c.EntradasCapacidad[1].Audiencia
		},
		"audiencia ausente": func(c *ConfiguracionConfianzaPerfilesV3, _ *DependenciasConfianzaPerfilesV3) {
			c.EntradasCapacidad = c.EntradasCapacidad[:10]
		},
		"material repetido": func(c *ConfiguracionConfianzaPerfilesV3, _ *DependenciasConfianzaPerfilesV3) {
			c.EntradasCapacidad[0].Material = c.EntradasCapacidad[1].Material
		},
		"clave de raiz": func(c *ConfiguracionConfianzaPerfilesV3, _ *DependenciasConfianzaPerfilesV3) {
			c.EntradasCapacidad[0].ClaveID = c.Raiz.ClaveID
		},
		"sin estado explicito": func(c *ConfiguracionConfianzaPerfilesV3, _ *DependenciasConfianzaPerfilesV3) {
			c.EntradasCapacidad[0].Estado = ""
		},
		"clave revocada": func(c *ConfiguracionConfianzaPerfilesV3, _ *DependenciasConfianzaPerfilesV3) {
			c.EntradasCapacidad[0].Estado = confianza.EstadoClaveHMACCapacidadAtestacionV3Revocada
		},
		"clave vencida": func(c *ConfiguracionConfianzaPerfilesV3, d *DependenciasConfianzaPerfilesV3) {
			c.EntradasCapacidad[0].ValidaHasta = d.Reloj.Ahora()
		},
		"cabecera y raiz": func(c *ConfiguracionConfianzaPerfilesV3, _ *DependenciasConfianzaPerfilesV3) {
			c.Cabecera.Audiencia = "vec:otro:despliegue"
		},
		"gobierno distinto": func(c *ConfiguracionConfianzaPerfilesV3, _ *DependenciasConfianzaPerfilesV3) {
			c.Gobierno.HuellaSHA256 = strings.Repeat("b", 64)
		},
		"gobierno vencido": func(c *ConfiguracionConfianzaPerfilesV3, d *DependenciasConfianzaPerfilesV3) {
			c.Gobierno.ExpiraEn = d.Reloj.Ahora()
		},
		"raiz revocada": func(c *ConfiguracionConfianzaPerfilesV3, _ *DependenciasConfianzaPerfilesV3) {
			c.Raiz.Estado = confianza.EstadoClaveAtestacionAutorizacionV3Revocada
		},
		"pool compartido": func(_ *ConfiguracionConfianzaPerfilesV3, d *DependenciasConfianzaPerfilesV3) {
			d.PoolMotivos = d.PoolRegistro
		},
		"sin TTL": func(_ *ConfiguracionConfianzaPerfilesV3, d *DependenciasConfianzaPerfilesV3) { d.VigenciaDecision = 0 },
		"firmante nulo tipado": func(_ *ConfiguracionConfianzaPerfilesV3, d *DependenciasConfianzaPerfilesV3) {
			d.Firmante = (*firmanteConfianzaPerfilesPrueba)(nil)
		},
		"reloj nulo tipado": func(_ *ConfiguracionConfianzaPerfilesV3, d *DependenciasConfianzaPerfilesV3) {
			d.Reloj = (*relojConfianzaPerfilesPrueba)(nil)
		},
	}
	for nombre, cambiar := range casos {
		t.Run(nombre, func(t *testing.T) {
			c, d := escenarioConfianzaPerfilesPrueba(t)
			cambiar(&c, &d)
			f, err := NuevaConfianzaPerfilesV3(c, d)
			if !errors.Is(err, ErrConfiguracion) || f.Fuente != nil || len(f.Emisores) != 0 {
				t.Fatal("configuracion insegura aceptada")
			}
		})
	}
}

func TestConfianzaPerfilesV3RedactaMaterialPrivado(t *testing.T) {
	cfg, _ := escenarioConfianzaPerfilesPrueba(t)
	marcador := "secreto-privado-admin-no-publicable"
	cfg.EntradasCapacidad[0].Material = []byte(marcador)
	for _, v := range []any{cfg, cfg.EntradasCapacidad[0]} {
		j, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		var registro bytes.Buffer
		slog.New(slog.NewJSONHandler(&registro, nil)).Info("prueba", "material", v)
		for _, salida := range []string{fmt.Sprintf("%+v", v), fmt.Sprintf("%#v", v), string(j), registro.String()} {
			if strings.Contains(salida, marcador) || strings.Contains(salida, "115 101 99 114") {
				t.Fatal("material expuesto")
			}
		}
	}
}
