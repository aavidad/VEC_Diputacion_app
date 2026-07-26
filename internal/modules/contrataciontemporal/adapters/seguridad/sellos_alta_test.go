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

	materialHuellaAltaV1Dorado = `{"esquema":"vec.contratacion-temporal.huella-alta.v1","organizacion_ref":"organizacion:diputacion-granada","actor_ref":"actor:tecnica-rrhh-001","perfil_ref":"perfil:tecnica-rrhh","flujo":{"definicion_ref":"flujo:contratacion-temporal-general","version":1,"huella_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},"solicitud":{"centro_ref":"centro:residencia-001","contacto_ref":"persona:responsable-centro-001","categoria_ref":"categoria:auxiliar-enfermeria","grupo_subgrupo":"C2","motivo_clave":"sustitucion.incapacidad_temporal","detalle":"Necesidad temporal para mantener el servicio.","periodo":{"inicio":"2026-08-01T00:00:00Z","fin":"2026-08-31T00:00:00Z"},"rc":{"existe":false,"fecha":"0001-01-01T00:00:00Z","importe":{"centimos":0,"moneda":""}},"documentos_adjuntos":["documento:informe-001"]}}`
	selloHuellaAltaV1Dorado    = "hmac-sha256:vec.contratacion-temporal.huella-peticion/v1:63ff3ba5d29c6008455989ee543d430fe3dfd9363a4981023bbe596fff63e95a"

	materialAmbitoAltaV1Dorado = `{"esquema":"vec.contratacion-temporal.ambito-idempotencia.v1","clave_idempotencia":"018f3b2a-7c4d-4e5f-8a9b-0c1d2e3f4a5b","organizacion_ref":"organizacion:diputacion-granada","actor_ref":"actor:tecnica-rrhh-001","perfil_ref":"perfil:tecnica-rrhh"}`
	selloAmbitoAltaV1Dorado    = "hmac-sha256:vec.contratacion-temporal.ambito-idempotencia/v1:e5d33266c2e25b77c838790830645c3a6de7694cd50340709a3f37b7c81fedfc"
)

type selladorPrueba struct {
	clave      []byte
	referencia string
	material   []byte
	err        error
}

func (s *selladorPrueba) SellarDatos(
	ctx context.Context,
	material []byte,
) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if s.err != nil {
		return "", s.err
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
	datosPrimero, err := primero.Datos()
	if err != nil {
		t.Fatal(err)
	}
	datosSegundo, err := segundo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	if datosPrimero.Activo != datosSegundo.Activo ||
		!ports.SelloHMACSHA256Valido(datosPrimero.Activo.Valor) ||
		!strings.Contains(string(sellador.material), esquemaHuellaAltaV1) {
		t.Fatalf("sello no canónico: %#v", datosPrimero)
	}
	if string(sellador.material) != materialHuellaAltaV1Dorado ||
		datosPrimero.Activo.Valor != selloHuellaAltaV1Dorado {
		t.Fatalf(
			"vector V1 alterado sin elevar esquema: material=%q sello=%#v",
			sellador.material,
			datosPrimero,
		)
	}

	alterado := material
	alterado.Solicitud.Detalle += " rectificada"
	tercero, err := derivador.DerivarHuellaAlta(context.Background(), alterado)
	if err != nil {
		t.Fatal(err)
	}
	if tercero.Contiene(datosPrimero.Activo.Valor) {
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
		ClaveIdempotencia: "018f3b2a-7c4d-4e5f-8a9b-0c1d2e3f4a5b",
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
	datosSello, err := sello.Datos()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(datosSello.Activo.Valor, solicitud.ClaveIdempotencia) ||
		!strings.Contains(string(sellador.material), solicitud.ClaveIdempotencia) ||
		!strings.Contains(string(sellador.material), esquemaAmbitoAltaV1) {
		t.Fatalf("separación idempotente inválida: %#v", datosSello)
	}
	if string(sellador.material) != materialAmbitoAltaV1Dorado ||
		datosSello.Activo.Valor != selloAmbitoAltaV1Dorado {
		t.Fatalf(
			"vector V1 alterado sin elevar esquema: material=%q sello=%q",
			sellador.material,
			datosSello,
		)
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

func TestLlaveroHMACRotaV2ConV1RetenidaSinExponerClaves(t *testing.T) {
	const (
		referenciaV2 = "vec.contratacion-temporal.huella-peticion/v2"
		referenciaV1 = "vec.contratacion-temporal.huella-peticion/v1"
	)
	activo := &selladorPrueba{
		clave:      []byte("material-sintetico-activo-prueba"),
		referencia: referenciaV2,
	}
	retenido := &selladorPrueba{
		clave:      []byte("material-sintetico-retenido-prueba"),
		referencia: referenciaV1,
	}
	configuracionActiva, err := NuevaConfiguracionSelladorHMAC(
		referenciaV2,
		activo,
	)
	if err != nil {
		t.Fatal(err)
	}
	configuracionRetenida, err := NuevaConfiguracionSelladorHMAC(
		referenciaV1,
		retenido,
	)
	if err != nil {
		t.Fatal(err)
	}
	derivador, err := NuevoDerivadorHuellaAltaHMACRotable(
		configuracionActiva,
		[]ConfiguracionSelladorHMAC{configuracionRetenida},
	)
	if err != nil {
		t.Fatal(err)
	}
	coleccion, err := derivador.DerivarHuellaAlta(
		context.Background(),
		materialHuellaPrueba(),
	)
	if err != nil {
		t.Fatal(err)
	}
	datos, err := coleccion.Datos()
	if err != nil {
		t.Fatal(err)
	}
	if datos.Activo.Generacion != 2 || len(datos.Retenidos) != 1 ||
		datos.Retenidos[0].Generacion != 1 ||
		!hmac.Equal(activo.material, retenido.material) ||
		strings.Contains(datos.Activo.Valor, string(activo.clave)) ||
		strings.Contains(datos.Retenidos[0].Valor, string(retenido.clave)) {
		t.Fatalf("llavero de rotación incoherente: %#v", datos)
	}
}

func TestLlaveroHMACFallaCerradoSiFaltaGeneracionRetenida(t *testing.T) {
	activa, err := NuevaConfiguracionSelladorHMAC(
		"vec.contratacion-temporal.huella-peticion/v2",
		&selladorPrueba{
			clave:      []byte("material-sintetico-activo-prueba"),
			referencia: "vec.contratacion-temporal.huella-peticion/v2",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	retenida, err := NuevaConfiguracionSelladorHMAC(
		"vec.contratacion-temporal.huella-peticion/v1",
		&selladorPrueba{
			referencia: "vec.contratacion-temporal.huella-peticion/v1",
			err:        errors.New("generación histórica no disponible"),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	derivador, err := NuevoDerivadorHuellaAltaHMACRotable(
		activa,
		[]ConfiguracionSelladorHMAC{retenida},
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = derivador.DerivarHuellaAlta(
		context.Background(),
		materialHuellaPrueba(),
	)
	if !errors.Is(err, ErrSelladoAltaNoDisponible) {
		t.Fatalf("ausencia histórica no cerrada: %v", err)
	}
}
