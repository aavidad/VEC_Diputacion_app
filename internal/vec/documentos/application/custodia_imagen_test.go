package application

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
	"time"
	"vec-diputacion-granada/internal/vec/documentos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func TestCustodiaImagenSoloPNG256Completo(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 256, 256))
	img.Set(0, 0, color.RGBA{R: 8, A: 255})
	var salida bytes.Buffer
	if err := png.Encode(&salida, img); err != nil {
		t.Fatal(err)
	}
	if !bytesPNG256(salida.Bytes()) {
		t.Fatal("PNG 256 válido rechazado")
	}
	truncado := salida.Bytes()[:salida.Len()-8]
	if bytesPNG256(truncado) {
		t.Fatal("no se debe aceptar un PNG incompleto")
	}
	img2 := image.NewRGBA(image.Rect(0, 0, 257, 256))
	salida.Reset()
	if err := png.Encode(&salida, img2); err != nil {
		t.Fatal(err)
	}
	if bytesPNG256(salida.Bytes()) {
		t.Fatal("dimensiones distintas de 256")
	}
}

type referenciaImagenPrueba struct {
	activa   bool
	err      error
	llamadas int
}

func (r *referenciaImagenPrueba) ReferenciaActiva(_ context.Context, _ ports.OperacionImagen) (bool, error) {
	r.llamadas++
	return r.activa, r.err
}
func TestCustodiaImagenReferenciaRetiradaCierraLectura(t *testing.T) {
	fuente := &referenciaImagenPrueba{activa: false}
	s := ServicioCustodiaImagen{Usuarios: fuente}
	if err := s.referenciaActiva(context.Background(), ports.OperacionImagen{}); !errors.Is(err, ports.ErrImagenProhibida) {
		t.Fatalf("retirada: %v", err)
	}
	fuente.activa = true
	if err := s.referenciaActiva(context.Background(), ports.OperacionImagen{}); err != nil {
		t.Fatalf("activa: %v", err)
	}
	fuente.err = errors.New("usuarios indisponible")
	if err := s.referenciaActiva(context.Background(), ports.OperacionImagen{}); !errors.Is(err, ports.ErrImagenProhibida) {
		t.Fatalf("fallo de Usuarios debe cerrar: %v", err)
	}
	if fuente.llamadas != 3 {
		t.Fatalf("cada apertura debe consultar de nuevo: %d", fuente.llamadas)
	}
}

type autoridadRevocadaImagen struct{}

func (autoridadRevocadaImagen) AutorizarImagen(context.Context, ports.OperacionImagen) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrImagenProhibida
}

type registroImagenNoUsado struct{ ports.RegistroImagen }
type contextosImagenNoUsados struct{ ports.ContextosAlmacenImagen }
type almacenImagenNoUsado struct{ vecports.AlmacenObjetos }
type admisorImagenNoUsado struct{ ports.AdmisorImagen }

func actorImagenPrueba(t *testing.T, ahora time.Time) vecdomain.ContextoActor {
	t.Helper()
	z := strings.Repeat("a", 24)
	cuenta := vecdomain.CuentaAutenticadaContextoActor{CuentaRef: "cta_" + z, Metodo: vecdomain.AuthMethodCertificate, Garantia: vecdomain.AuthAssuranceHigh}
	instantanea := vecdomain.InstantaneaContextoActor{VinculoRef: "vca_" + z, VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, CuentaVersion: 1, PersonaRef: "per_" + z, PersonaVersion: 1, PerfilActivoRef: "prf_" + z, PerfilVersion: 1, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}
	actor, err := vecdomain.NuevoContextoActor(cuenta, instantanea, ahora)
	if err != nil {
		t.Fatal(err)
	}
	return actor
}
func TestCustodiaImagenRevocacionDocumentosDuranteLectura(t *testing.T) {
	ahora := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	actor := actorImagenPrueba(t, ahora)
	usuarios := &referenciaImagenPrueba{activa: true}
	s := ServicioCustodiaImagen{Registro: &registroImagenNoUsado{}, Autoridad: autoridadRevocadaImagen{}, Contextos: &contextosImagenNoUsados{}, Almacen: &almacenImagenNoUsado{}, Admisor: &admisorImagenNoUsado{}, Usuarios: usuarios, AhoraUTC: func() time.Time { return ahora }}
	op := ports.OperacionImagen{Actor: actor, TitularPersonaRef: actor.PersonaRef, Audiencia: ports.AudienciaImagenPersonal, Finalidad: ports.FinalidadImagenPropia, Accion: ports.AccionImagenAbrirPropia, DocumentoRef: "doc_1234567890123456"}
	// El buffer simula los bytes ya leídos del almacén. Usuarios sigue activo;
	// solo Documentos ha revocado el permiso mientras se recibían los bytes.
	buffer := []byte("bytes privados ya leídos")
	got, err := s.entregarTrasLectura(context.Background(), op, ports.ReservaImagen{}, buffer)
	if !errors.Is(err, ports.ErrImagenProhibida) || got != nil {
		t.Fatalf("se entregaron bytes tras revocación Documentos: %v, %q", err, got)
	}
	for _, b := range buffer {
		if b != 0 {
			t.Fatal("el buffer denegado no quedó limpio")
		}
	}
	if !usuarios.activa {
		t.Fatal("la prueba debe conservar Usuarios activo")
	}
}
