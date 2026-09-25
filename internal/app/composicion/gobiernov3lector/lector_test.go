package gobiernov3lector

import (
	"context"
	"crypto/ed25519"
	"errors"
	"testing"
	"time"

	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
)

type relojPrueba struct{ ahora time.Time }

func (r *relojPrueba) Ahora() time.Time { return r.ahora }

func publicacionPrueba(t *testing.T, raiz confianza.RaizPublicaAtestacionAutorizacionV3, dia time.Time, secuencia uint64) Publicacion {
	t.Helper()
	ref := "configuracion:prueba:" + dia.Format("2006-01-02")
	config, err := confianza.NuevaConfiguracionConfianzaAtestacionAutorizacionV3(ref, secuencia, dia, dia.Add(24*time.Hour), raiz)
	if err != nil {
		t.Fatal(err)
	}
	huella, err := config.HuellaSHA256ParaGobierno()
	if err != nil {
		t.Fatal(err)
	}
	return Publicacion{Revision: ref, Secuencia: secuencia, HuellaSHA256: huella, PublicadaEn: dia, ExpiraEn: dia.Add(24 * time.Hour)}
}

func escenarioPrueba(t *testing.T) (confianza.RaizPublicaAtestacionAutorizacionV3, Publicacion, Publicacion, *relojPrueba) {
	t.Helper()
	dia := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	semilla := make([]byte, ed25519.SeedSize)
	clave := ed25519.NewKeyFromSeed(semilla)
	raiz, err := confianza.NuevaRaizPublicaAtestacionAutorizacionV3EdDSA(
		"clave:prueba:v3", 1, clave.Public().(ed25519.PublicKey), "vec:desarrollo:contratacion-temporal:atestacion:v3",
		confianza.EstadoClaveAtestacionAutorizacionV3Activa, dia.Add(-time.Hour), dia.Add(72*time.Hour), time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	return raiz, publicacionPrueba(t, raiz, dia, 20260925), publicacionPrueba(t, raiz, dia.Add(24*time.Hour), 20260926), &relojPrueba{ahora: dia.Add(time.Hour)}
}

func TestLectorReleeCadaOperacionYAdoptaCambioDeDia(t *testing.T) {
	raiz, primera, segunda, reloj := escenarioPrueba(t)
	actual := primera
	lecturas := 0
	lector, err := Nuevo(primera, raiz, reloj, func(_ context.Context, _ Publicacion) (Publicacion, error) {
		lecturas++
		return actual, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lector.Instantanea(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := lector.Instantanea(context.Background()); err != nil || lecturas != 2 {
		t.Fatalf("no releyó la revocación/puntero: %d %v", lecturas, err)
	}
	actual = segunda
	reloj.ahora = segunda.PublicadaEn.Add(time.Minute)
	if _, err := lector.Instantanea(context.Background()); err != nil || lector.anterior.Secuencia != segunda.Secuencia {
		t.Fatalf("cambio de día rechazado: %v", err)
	}
	actual = primera
	if _, err := lector.Instantanea(context.Background()); !errors.Is(err, ErrGobiernoNoDisponible) {
		t.Fatalf("retroceso admitido: %v", err)
	}
}

func TestLectorFallaCerradoAnteCambioIncoherenteRevocacionYCaducidad(t *testing.T) {
	raiz, primera, _, reloj := escenarioPrueba(t)
	respuesta := primera
	var fallo error
	lector, err := Nuevo(primera, raiz, reloj, func(context.Context, Publicacion) (Publicacion, error) { return respuesta, fallo })
	if err != nil {
		t.Fatal(err)
	}
	respuesta.HuellaSHA256 = "0000000000000000000000000000000000000000000000000000000000000000"
	if _, err := lector.Instantanea(context.Background()); !errors.Is(err, ErrGobiernoNoDisponible) {
		t.Fatalf("huella alterada admitida: %v", err)
	}
	respuesta = primera
	fallo = errors.New("revocación publicada")
	if _, err := lector.Instantanea(context.Background()); !errors.Is(err, ErrGobiernoNoDisponible) {
		t.Fatalf("revocación ignorada: %v", err)
	}
	fallo = nil
	reloj.ahora = primera.ExpiraEn
	if _, err := lector.Instantanea(context.Background()); !errors.Is(err, ErrGobiernoNoDisponible) {
		t.Fatalf("configuración vencida admitida: %v", err)
	}
}

// PostgreSQL devuelve las fechas de la publicación con desplazamiento +00:00;
// al decodificarlas llegan con una Location distinta de time.UTC.
func TestLectorAdmiteFechasConDesplazamientoCero(t *testing.T) {
	raiz, primera, _, reloj := escenarioPrueba(t)
	cero := time.FixedZone("", 0)
	leida := primera
	leida.PublicadaEn = primera.PublicadaEn.In(cero)
	leida.ExpiraEn = primera.ExpiraEn.In(cero)
	anterior := primera
	anterior.PublicadaEn = primera.PublicadaEn.In(time.Local)
	lector, err := Nuevo(anterior, raiz, reloj, func(_ context.Context, _ Publicacion) (Publicacion, error) {
		return leida, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, p, err := lector.Leer(context.Background()); err != nil || p.PublicadaEn.Location() != time.UTC {
		t.Fatalf("publicación con desplazamiento cero rechazada: %v", err)
	}
}
