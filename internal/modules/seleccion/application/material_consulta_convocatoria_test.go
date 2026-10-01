package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/seleccion/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func TestConsultaExactaTieneCoberturaNominalSinCubrirOtraVersion(t *testing.T) {
	s := solicitudConvocatoriaPrueba(t)
	p, err := PrepararConsultaConvocatoria(s)
	if err != nil {
		t.Fatal(err)
	}
	now := s.Actor.ResueltoEn
	a := vecdomain.AsignacionPerfil{AsignacionID: "fixture_s1", Version: 1, PerfilActivoRef: s.Actor.PerfilActivoRef,
		PrincipalID: s.Actor.PersonaRef, VersionRolRef: "rol:fixture_s1:v1", Estado: vecdomain.EstadoAsignacionPerfilActiva,
		Ambitos:      []vecdomain.AmbitoPerfil{{Clave: "convocatoria_id", Valores: []string{s.Selector.ID}}, {Clave: "secuencia", Valores: []string{"2"}}},
		VigenteDesde: now, VigenteHasta: now.Add(time.Hour), EmitidaPor: "fixture:s1:owner", EmitidaEn: now}
	if a.Validar() != nil || !a.Cubre(p.Recurso) {
		t.Fatal("una asignación exacta válida no cubre el recurso")
	}
	previous := p.Recurso
	previous.Ambitos = nil
	if a.Cubre(previous) {
		t.Fatal("el recurso sin ámbitos obtuvo cobertura")
	}
	other := s
	other.Selector.Secuencia++
	p2, err := PrepararConsultaConvocatoria(other)
	if err != nil || a.Cubre(p2.Recurso) {
		t.Fatal("la concesión cubre otra versión")
	}
	other = s
	other.Selector.ID = "convocatoria:ajena"
	p2, err = PrepararConsultaConvocatoria(other)
	if err != nil || a.Cubre(p2.Recurso) {
		t.Fatal("la concesión cubre otra convocatoria")
	}
}

func TestMaterialConsultaLigaSelectorContextoYCorrelacion(t *testing.T) {
	s := solicitudConvocatoriaPrueba(t)
	p, err := PrepararConsultaConvocatoria(s)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(p.Canonico, &m); err != nil {
		t.Fatal(err)
	}
	if len(m) != 10 || m["convocatoria_id"] != s.Selector.ID || m["actor_ref"] != s.Actor.PersonaRef || m["contexto_version"] != float64(s.Actor.Instantanea.VinculoVersion) || m["correlacion_ref"] != correlacionConvocatoria(s) {
		t.Fatal("material no ligado a la operación nominal")
	}
	h := sha256.Sum256(p.Canonico)
	if p.Recurso.Referencia != s.Selector.Referencia() || p.Recurso.Atributos["material_sha256"] != hex.EncodeToString(h[:]) {
		t.Fatal("recurso no compromete material")
	}
	otro := s
	otro.Selector.Secuencia++
	p2, err := PrepararConsultaConvocatoria(otro)
	if err != nil || p.Recurso.Atributos["material_sha256"] == p2.Recurso.Atributos["material_sha256"] {
		t.Fatal("dos versiones comparten material")
	}
}

type autorizadorConvocatoriaV3Prueba struct {
	llamadas int
	err      error
}

func (a *autorizadorConvocatoriaV3Prueba) AutorizarConsultaConvocatoria(context.Context, ports.SolicitudConsultaConvocatoria) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	a.llamadas++
	return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, a.err
}

type repositorioConvocatoriaV3Prueba struct{ llamadas int }

func (r *repositorioConvocatoriaV3Prueba) ObtenerVersionExactaV3(context.Context, ports.OrdenConsultaConvocatoria) (ports.ResultadoVersionConvocatoria, error) {
	r.llamadas++
	return ports.ResultadoVersionConvocatoria{}, nil
}

func TestLectorV3NoConsultaSinConcesionNominal(t *testing.T) {
	s := solicitudConvocatoriaPrueba(t)
	for _, causa := range []error{nil, ports.ErrConsultaConvocatoriaDenegada, ports.ErrConvocatoriaNoDisponible} {
		a, r := &autorizadorConvocatoriaV3Prueba{err: causa}, &repositorioConvocatoriaV3Prueba{}
		l, err := NuevoLectorConvocatoriaV3(a, r)
		if err != nil {
			t.Fatal(err)
		}
		resultado, err := l.ConsultarExacta(context.Background(), s)
		esperado := causa
		if esperado == nil {
			esperado = ports.ErrConvocatoriaNoDisponible
		}
		if !errors.Is(err, esperado) || r.llamadas != 0 || a.llamadas != 1 || resultado.Ficha.ConvocatoriaID != "" {
			t.Fatal("se consultó sin concesión o se dio éxito parcial")
		}
	}
}
