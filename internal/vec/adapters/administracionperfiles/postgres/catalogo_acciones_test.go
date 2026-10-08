package postgres

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func catalogoAccionesPublicadoPrueba(t *testing.T) ([]byte, string) {
	t.Helper()
	desde := time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC)
	concesion := domain.ConcesionRol{Accion: "sintetico.consultar", ModuloID: "sintetico", TipoRecurso: "expediente",
		Finalidades: []string{"revision"}, GarantiaMinima: domain.AuthAssuranceHigh,
		CamposPermitidos: []string{"estado"}, Obligaciones: []string{"auditar"}}
	rol := domain.VersionRol{RolID: "revision_sintetica", Version: 1, Nombre: "revision_sintetica",
		Estado: domain.EstadoVersionRolPublicada, Concesiones: []domain.ConcesionRol{concesion},
		PublicadaPor: "actor:fuente", PublicadaEn: desde}
	c := domain.CatalogoAccionesAdministracionV1{Referencia: "catalogo:sintetico", Version: 1,
		FuenteRef: "fuente:sintetica", FuenteVersion: 1, FuenteHuellaSHA256: strings.Repeat("a", 64),
		VigenteDesde: desde, VigenteHasta: desde.Add(72 * time.Hour),
		Entradas: []domain.EntradaAccionAdministracionV1{{Referencia: "accion:sintetica", Version: 1,
			FuenteRef: "fuente:modulo", FuenteVersion: 2, FuenteHuellaSHA256: strings.Repeat("b", 64),
			Concesion: concesion, DimensionesAmbito: []string{"unidad"}, ClaseControl: "consulta_auditada",
			VigenteDesde: desde, VigenteHasta: desde.Add(48 * time.Hour)}},
		Perfiles: []domain.PerfilPublicadoAdministracionV1{{Rol: rol,
			TipoPerfil: domain.TipoPerfilAdministracionFijoSistemaV1,
			ControlVigencia: domain.ControlVigenciaVersionRol{VersionRolRef: rol.Referencia(), Revision: 1,
				Estado: domain.EstadoControlVigenciaVersionRolHabilitada, ActualizadoPor: "actor:fuente", ActualizadoEn: desde}}}}
	h, err := c.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	return b, h
}

func TestFuenteCatalogoAccionesRechazaRespuestaParcialOAlterada(t *testing.T) {
	canon, h := catalogoAccionesPublicadoPrueba(t)
	const ref = "catalogo:sintetico"
	if _, err := decodificarCatalogoAccionesPublicado(canon, ref, 1, h); err != nil {
		t.Fatal(err)
	}
	var c domain.CatalogoAccionesAdministracionV1
	if err := json.Unmarshal(canon, &c); err != nil {
		t.Fatal(err)
	}
	c.Perfiles = nil
	sinPerfiles, _ := json.Marshal(c)
	c.Perfiles = []domain.PerfilPublicadoAdministracionV1{}
	sinCenso, _ := json.Marshal(c)
	casos := []struct {
		nombre, ref, huella string
		version             int
		canon               []byte
	}{
		{"perfiles ausentes", ref, h, 1, sinPerfiles},
		{"censo vacio", ref, h, 1, sinCenso},
		{"otra huella", ref, strings.Repeat("f", 64), 1, canon},
		{"otra referencia", "catalogo:otro", h, 1, canon},
		{"otra version", ref, h, 2, canon},
		{"json desconocido", ref, h, 1, bytes.Replace(canon, []byte(`"version":1`), []byte(`"version":1,"autoridad":true`), 1)},
		{"json no canonico", ref, h, 1, append(append([]byte{}, canon...), '\n')},
		{"contenido excesivo", ref, h, 1, make([]byte, maximoBytesCatalogoAcciones+1)},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if _, err := decodificarCatalogoAccionesPublicado(caso.canon, caso.ref, caso.version, caso.huella); !errors.Is(err, ports.ErrAutoridadAdministracionPerfilesNoDisponible) {
				t.Fatalf("fuente parcial o distinta aceptada: %v", err)
			}
		})
	}
}
