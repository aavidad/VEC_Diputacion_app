package seguridad

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	claveHuellaPrueba = "vec.contratacion-temporal.huella-peticion/v1"
	claveAmbitoPrueba = "vec.contratacion-temporal.ambito-idempotencia/v1"
)

type selladorPrueba struct {
	clave      []byte
	referencia string
	material   []byte
}

func (s *selladorPrueba) SellarDatos(
	ctx context.Context,
	material []byte,
) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	s.material = append([]byte(nil), material...)
	mac := hmac.New(sha256.New, s.clave)
	_, _ = mac.Write(material)
	return "hmac-sha256:" + s.referencia + ":" +
		hex.EncodeToString(mac.Sum(nil)), nil
}

func materialHuellaPrueba() ports.MaterialHuellaAlta {
	return ports.MaterialHuellaAlta{
		OrganizacionRef: "organizacion:diputacion-granada",
		ActorRef:        "actor:tecnica-rrhh-001",
		PerfilRef:       "perfil:tecnica-rrhh",
		Flujo: domain.ReferenciaFlujo{
			DefinicionRef: "flujo:contratacion-temporal-general",
			Version:       1,
			HuellaSHA256:  strings.Repeat("a", 64),
		},
		Solicitud: domain.SolicitudCentro{
			CentroRef:     "centro:residencia-001",
			ContactoRef:   "persona:responsable-centro-001",
			CategoriaRef:  "categoria:auxiliar-enfermeria",
			GrupoSubgrupo: "C2",
			MotivoClave:   "sustitucion.incapacidad_temporal",
			Detalle:       "Necesidad temporal para mantener el servicio.",
			Periodo: domain.PeriodoPrevisto{
				Inicio: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
				Fin:    time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC),
			},
			DocumentosAdjuntos: []string{"documento:informe-001"},
		},
	}
}

func TestDerivadorHuellaAltaHMACLigaMaterialCanonico(t *testing.T) {
	sellador := &selladorPrueba{
		clave:      []byte("clave-prueba-32-bytes-minimo-0001"),
		referencia: claveHuellaPrueba,
	}
	derivador, err := NuevoDerivadorHuellaAltaHMAC(
		claveHuellaPrueba,
		sellador,
	)
	if err != nil {
		t.Fatal(err)
	}
	material := materialHuellaPrueba()
	primero, err := derivador.DerivarHuellaAlta(context.Background(), material)
	if err != nil {
		t.Fatal(err)
	}
	segundo, err := derivador.DerivarHuellaAlta(context.Background(), material)
	if err != nil {
		t.Fatal(err)
	}
	if primero != segundo || !ports.SelloHMACSHA256Valido(primero) ||
		!strings.Contains(string(sellador.material), esquemaHuellaAltaV1) {
		t.Fatalf("sello no canónico: %q", primero)
	}

	alterado := material
	alterado.Solicitud.Detalle += " rectificada"
	tercero, err := derivador.DerivarHuellaAlta(context.Background(), alterado)
	if err != nil {
		t.Fatal(err)
	}
	if tercero == primero {
		t.Fatal("el sello no quedó ligado a la solicitud")
	}
}

func TestSelladorAmbitoNoExponeClaveEnResultado(t *testing.T) {
	sellador := &selladorPrueba{
		clave:      []byte("otra-clave-prueba-32-bytes-00002"),
		referencia: claveAmbitoPrueba,
	}
	adaptador, err := NuevoSelladorAmbitoIdempotenciaHMAC(
		claveAmbitoPrueba,
		sellador,
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitud := ports.SolicitudSellarAmbitoIdempotencia{
		ClaveIdempotencia: "01J2F8X4K4R9T2Y7W3M6Q8P1AB",
		OrganizacionRef:   "organizacion:diputacion-granada",
		ActorRef:          "actor:tecnica-rrhh-001",
		PerfilRef:         "perfil:tecnica-rrhh",
	}
	sello, err := adaptador.SellarAmbitoIdempotencia(
		context.Background(),
		solicitud,
	)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(sello, solicitud.ClaveIdempotencia) ||
		!strings.Contains(string(sellador.material), solicitud.ClaveIdempotencia) ||
		!strings.Contains(string(sellador.material), esquemaAmbitoAltaV1) {
		t.Fatalf("separación idempotente inválida: %q", sello)
	}
}

func TestSelladoresRechazanDominioOCancelacion(t *testing.T) {
	sellador := &selladorPrueba{
		clave:      []byte("clave-prueba-32-bytes-minimo-0001"),
		referencia: claveAmbitoPrueba,
	}
	if _, err := NuevoDerivadorHuellaAltaHMAC(
		claveAmbitoPrueba,
		sellador,
	); !errors.Is(err, ErrSelladoAltaNoDisponible) {
		t.Fatalf("se aceptó la clave del otro dominio: %v", err)
	}
	derivador, err := NuevoDerivadorHuellaAltaHMAC(
		claveHuellaPrueba,
		&selladorPrueba{
			clave:      []byte("clave-prueba-32-bytes-minimo-0001"),
			referencia: claveHuellaPrueba,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := derivador.DerivarHuellaAlta(
		ctx,
		materialHuellaPrueba(),
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelación perdida: %v", err)
	}
}

func TestDerivadorRechazaDependenciaNulaTipada(t *testing.T) {
	var sellador *selladorPrueba
	if _, err := NuevoDerivadorHuellaAltaHMAC(
		claveHuellaPrueba,
		sellador,
	); !errors.Is(err, ErrSelladoAltaNoDisponible) {
		t.Fatalf("dependencia nula aceptada: %v", err)
	}
}
