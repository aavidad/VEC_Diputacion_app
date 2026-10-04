package capturacopias

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
	puertos "vec-diputacion-granada/internal/modules/administracion/ports/capturacopias"
)

type plataforma struct {
	pasos      []string
	fallar     string
	inventario copias.Inventario
	capturas   int
	checks     int
	cancel     context.CancelFunc
}

func (p *plataforma) paso(s string) error {
	p.pasos = append(p.pasos, s)
	if p.fallar == s {
		return errors.New("privado")
	}
	return nil
}
func (p *plataforma) Adquirir(context.Context, string) (func() error, error) {
	return func() error { return p.paso("liberar") }, p.paso("adquirir")
}
func (p *plataforma) CerrarAdmision(context.Context) error { return p.paso("cerrar") }
func (p *plataforma) Drenar(context.Context) error         { return p.paso("drenar") }
func (p *plataforma) ComprobarExclusion(context.Context) error {
	p.checks++
	return p.paso("comprobar")
}
func (p *plataforma) Reabrir(ctx context.Context) error {
	if ctx.Err() != nil {
		return errors.New("cancelado")
	}
	return p.paso("reabrir")
}
func (p *plataforma) Observar(context.Context) (copias.Inventario, error) {
	return p.inventario, p.paso("inventario")
}
func (p *plataforma) Capturar(context.Context, copias.Inventario) ([]copias.Artefacto, error) {
	p.capturas++
	if p.cancel != nil {
		p.cancel()
	}
	return []copias.Artefacto{{ID: "logico"}}, p.paso("capturar")
}

func ejemplo(t *testing.T) (*plataforma, Servicio, puertos.Peticion) {
	t.Helper()
	f, err := os.Open("../../../../../cmd/vec-copias-comprobar/testdata/compatible.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var v struct {
		Manifiesto copias.Manifiesto `json:"manifiesto"`
	}
	if json.NewDecoder(f).Decode(&v) != nil {
		t.Fatal("fixture")
	}
	p := &plataforma{inventario: v.Manifiesto.Inventario}
	now := time.Now().UTC()
	s := Servicio{Exclusor: p, Escritores: p, Inventario: p, Logico: p, Ahora: func() time.Time { return now }, TiempoLiberacion: time.Second}
	return p, s, puertos.Peticion{OrigenRef: v.Manifiesto.Inventario.PostgreSQL.ClusterRef, OperacionRef: "op:sintetica", Esperado: v.Manifiesto.Inventario, InicioVentana: now.Add(-time.Minute), FinVentana: now.Add(time.Minute)}
}

func TestCapturaMantieneVentanaHastaComponentesYNoDeclaraValidez(t *testing.T) {
	p, s, solicitud := ejemplo(t)
	s.Componentes = p
	r, err := s.Capturar(context.Background(), solicitud)
	if err != nil || r.Estado != "captura_parcial_pendiente" || r.Completa || r.Valida || r.Publicable || p.capturas != 2 {
		t.Fatalf("%+v %v", r, err)
	}
	esperado := []string{"adquirir", "cerrar", "drenar", "comprobar", "inventario", "comprobar", "inventario", "capturar", "comprobar", "capturar", "inventario", "comprobar", "reabrir", "liberar"}
	if !reflect.DeepEqual(p.pasos, esperado) {
		t.Fatalf("%v", p.pasos)
	}
}
func TestFallosNoPermitenCapturarYReabrenCierreParcial(t *testing.T) {
	for _, paso := range []string{"cerrar", "drenar", "comprobar", "inventario"} {
		t.Run(paso, func(t *testing.T) {
			p, s, solicitud := ejemplo(t)
			p.fallar = paso
			r, err := s.Capturar(context.Background(), solicitud)
			if err == nil || p.capturas != 0 || r.Estado != "captura_fallida" {
				t.Fatalf("%+v %v", r, err)
			}
			if !reflect.DeepEqual(p.pasos[len(p.pasos)-2:], []string{"reabrir", "liberar"}) {
				t.Fatal(p.pasos)
			}
		})
	}
}
func TestInventarioIncompatibleBloqueaVolcado(t *testing.T) {
	p, s, solicitud := ejemplo(t)
	p.inventario.Release.Commit = "ffffffffffffffffffffffffffffffffffffffff"
	_, err := s.Capturar(context.Background(), solicitud)
	if !errors.Is(err, ErrInventario) || p.capturas != 0 {
		t.Fatalf("%v", err)
	}
}
func TestCancelacionNoImpideReabrirYBloqueaContinuacion(t *testing.T) {
	p, s, solicitud := ejemplo(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p.cancel = cancel
	s.Componentes = p
	r, err := s.Capturar(ctx, solicitud)
	if err == nil || r.Estado != "captura_fallida" || p.capturas != 1 {
		t.Fatalf("%+v %v", r, err)
	}
	if !reflect.DeepEqual(p.pasos[len(p.pasos)-2:], []string{"reabrir", "liberar"}) {
		t.Fatal(p.pasos)
	}
}
func TestLiberacionFallidaNoDevuelveCapturaTerminada(t *testing.T) {
	p, s, solicitud := ejemplo(t)
	p.fallar = "reabrir"
	r, err := s.Capturar(context.Background(), solicitud)
	if !errors.Is(err, ErrLiberacion) || r.Estado != "captura_fallida" {
		t.Fatalf("%+v %v", r, err)
	}
}
func TestVentanaCerradaNoTieneEfectos(t *testing.T) {
	p, s, solicitud := ejemplo(t)
	solicitud.FinVentana = s.Ahora()
	_, err := s.Capturar(context.Background(), solicitud)
	if !errors.Is(err, ErrPrecondicion) || len(p.pasos) != 0 {
		t.Fatalf("%v %v", err, p.pasos)
	}
}
