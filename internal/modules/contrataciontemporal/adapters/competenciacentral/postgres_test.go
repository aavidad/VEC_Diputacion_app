package competenciacentral

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type revalidadorCentralPrueba struct {
	llamadas  int
	err       error
	evidencia vecdomain.EvidenciaAsignacionCompetencialV1
}

func (r *revalidadorCentralPrueba) RevalidarAsignacionCompetencialV1(_ context.Context, _ vecdomain.SolicitudAsignacionCompetencialV1, e vecdomain.EvidenciaAsignacionCompetencialV1) error {
	r.llamadas++
	r.evidencia = e
	return r.err
}

type revalidadorBindingPrueba struct {
	llamadas int
	err      error
	vinculo  VinculoCertificadoFirmante
}

type revalidadorRelacionPrueba struct {
	llamadas int
	err      error
	fallarEn int
	relacion RelacionRecursoCT
}

func (r *revalidadorRelacionPrueba) RevalidarRelacionRecursoCT(_ context.Context, v RelacionRecursoCT) error {
	r.llamadas++
	r.relacion = v
	if r.llamadas == r.fallarEn {
		return ctports.ErrCompetenciaFirmanteNoAcreditada
	}
	return r.err
}

func (r *revalidadorBindingPrueba) RevalidarVinculoCertificadoFirmante(_ context.Context, v VinculoCertificadoFirmante) error {
	r.llamadas++
	r.vinculo = v
	return r.err
}

func TestRevalidadorMantieneFuentesCompletasYNoAceptaProyeccion(t *testing.T) {
	f, i, _, _, q := datosPrueba(t)
	a, err := f.AcreditarCompetenciaCentral(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	central := &revalidadorCentralPrueba{}
	binding := &revalidadorBindingPrueba{}
	relacion := &revalidadorRelacionPrueba{}
	r, err := NuevoRevalidador(central, binding, relacion, f.reloj)
	if err != nil {
		t.Fatal(err)
	}
	// La misma acreditación se revalida de nuevo en replay con fuentes actuales.
	for j := 0; j < 2; j++ {
		if err = r.RevalidarCompetenciaCentral(context.Background(), a); err != nil {
			t.Fatal(err)
		}
	}
	if central.llamadas != 2 || binding.llamadas != 2 || relacion.llamadas != 4 || relacion.relacion != a.relacion || binding.vinculo != i.vinculo || central.evidencia.Cargo.Version != 2 || central.evidencia.EnlaceOcupante.Version != 4 {
		t.Fatal("perdió fuentes centrales")
	}
	if err = r.RevalidarCompetenciaCentral(context.Background(), Acreditacion{proyeccion: a.Proyeccion()}); err != ctports.ErrCompetenciaFirmanteNoAcreditada {
		t.Fatal("proyección aceptada sin fuentes")
	}
}

func TestRevalidadorRevocacionYCaducidadImpidenEfecto(t *testing.T) {
	f, _, _, _, q := datosPrueba(t)
	a, err := f.AcreditarCompetenciaCentral(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	central := &revalidadorCentralPrueba{}
	binding := &revalidadorBindingPrueba{err: ctports.ErrCompetenciaFirmanteNoAcreditada}
	relacion := &revalidadorRelacionPrueba{}
	r, _ := NuevoRevalidador(central, binding, relacion, f.reloj)
	if err = r.RevalidarCompetenciaCentral(context.Background(), a); err != ctports.ErrCompetenciaFirmanteNoAcreditada || central.llamadas != 0 {
		t.Fatal("revocación aceptada")
	}
	binding.err = nil
	central.err = errors.New("valor privado de la fuente")
	if err = r.RevalidarCompetenciaCentral(context.Background(), a); err != ctports.ErrCompetenciaFirmanteNoDisponible {
		t.Fatal("error privado expuesto")
	}
	central.err = nil
	r.reloj = &relojPrueba{a.evidencia.CargoVigenteHasta}
	if err = r.RevalidarCompetenciaCentral(context.Background(), a); err != ctports.ErrCompetenciaFirmanteNoAcreditada {
		t.Fatal("caducidad aceptada")
	}
	r.reloj = &relojPrueba{a.evidencia.ComprobadaEn.Add(time.Microsecond)}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err = r.RevalidarCompetenciaCentral(ctx, a); err != context.Canceled {
		t.Fatal("cancelación ignorada")
	}
}

func TestDescriptorExigeRolNominalExactoPublicadoSinListaLocal(t *testing.T) {
	f, _, a, _, q := datosPrueba(t)
	// La autoridad del conjunto es el descriptor ya publicado. Una evidencia
	// de otro RolID no se acepta aunque sea una versión de rol bien formada.
	a.evidencia.VersionRol.RolID = "ct_rol_nominal_otro"
	a.evidencia.Asignacion.VersionRolRef = a.evidencia.VersionRol.Referencia()
	a.evidencia.ControlVigencia.VersionRolRef = a.evidencia.VersionRol.Referencia()
	var err error
	a.evidencia.VersionRolHuellaSHA256, err = a.evidencia.VersionRol.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	a.evidencia.AsignacionHuellaSHA256, err = a.evidencia.Asignacion.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	a.evidencia.ControlVigenciaHuellaSHA256, err = a.evidencia.ControlVigencia.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.AcreditarCompetenciaCentral(context.Background(), q); err != ctports.ErrCompetenciaFirmanteNoAcreditada {
		t.Fatal("RolID no publicado acreditado")
	}
	if !rolDescriptorValido("ct_rol_nominal_configurado") {
		t.Fatal("inventó una lista local de roles")
	}
	for _, rol := range []string{"", "ct_cargo_*", "ct cargo", strings.Repeat("a", 129)} {
		if rolDescriptorValido(rol) {
			t.Fatal("RolID fuera de gramática permitido")
		}
	}
}

func TestRevalidadorExigeRelacionCTYLaRevalidaTambienTrasCentral(t *testing.T) {
	f, _, _, _, q := datosPrueba(t)
	a, err := f.AcreditarCompetenciaCentral(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	central := &revalidadorCentralPrueba{}
	binding := &revalidadorBindingPrueba{}
	if _, err := NuevoRevalidador(central, binding, nil, f.reloj); err != ctports.ErrCompetenciaFirmanteNoDisponible {
		t.Fatal("relación CT opcional")
	}
	relacion := &revalidadorRelacionPrueba{err: ctports.ErrCompetenciaFirmanteNoAcreditada}
	r, _ := NuevoRevalidador(central, binding, relacion, f.reloj)
	if err := r.RevalidarCompetenciaCentral(context.Background(), a); err != ctports.ErrCompetenciaFirmanteNoAcreditada || central.llamadas != 0 || binding.llamadas != 0 {
		t.Fatal("relación inválida alcanzó las fuentes centrales")
	}
	relacion.err = nil
	relacion.llamadas = 0
	relacion.fallarEn = 2
	if err := r.RevalidarCompetenciaCentral(context.Background(), a); err != ctports.ErrCompetenciaFirmanteNoAcreditada || relacion.llamadas != 2 || central.llamadas != 1 {
		t.Fatal("relación no revalidada después del lector central", err)
	}
}
