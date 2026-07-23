package ports

import (
	"strings"
	"testing"
)

const (
	dominioHuellaPrueba = "vec.contratacion-temporal.huella-peticion/v1"
	dominioAmbitoPrueba = "vec.contratacion-temporal.ambito-idempotencia/v1"
)

func TestSelloHMACSHA256ValidoExigeDominioVersionadoYValorNoNulo(t *testing.T) {
	valido := "hmac-sha256:" + dominioHuellaPrueba + ":" + strings.Repeat("a", 64)
	casos := map[string]struct {
		valor string
		vale  bool
	}{
		"versionado":       {valor: valido, vale: true},
		"huella desnuda":   {valor: strings.Repeat("a", 64), vale: false},
		"digest nulo":      {valor: "hmac-sha256:" + dominioHuellaPrueba + ":" + strings.Repeat("0", 64), vale: false},
		"dominio vacío":    {valor: "hmac-sha256::" + strings.Repeat("a", 64), vale: false},
		"algoritmo ajeno":  {valor: "sha256:" + dominioHuellaPrueba + ":" + strings.Repeat("a", 64), vale: false},
		"hexadecimal alto": {valor: "hmac-sha256:" + dominioHuellaPrueba + ":" + strings.Repeat("A", 64), vale: false},
	}
	for nombre, caso := range casos {
		t.Run(nombre, func(t *testing.T) {
			if obtenido := SelloHMACSHA256Valido(caso.valor); obtenido != caso.vale {
				t.Fatalf("validez=%v; se esperaba %v para %q", obtenido, caso.vale, caso.valor)
			}
		})
	}
}

func TestPreparacionAltaQuedaLigadaATenantActorPerfilYPeticion(t *testing.T) {
	huellas := coleccionSelloPrueba(t, selloPrueba(dominioHuellaPrueba, "a"))
	solicitud := SolicitudPrepararAlta{
		ClaveIdempotencia:   "018f3b2a-7c4d-4e5f-8a9b-0c1d2e3f4a5b",
		HuellasPeticionHMAC: huellas,
		OrganizacionRef:     "organizacion:diputacion-granada",
		ActorRef:            "actor:tecnica-rrhh-001",
		PerfilRef:           "perfil:tecnica-rrhh",
	}
	base := PreparacionAlta{
		ReservaRef: "reserva:ct-alta:001",
		Referencias: ReferenciasAlta{
			ExpedienteRef: "expediente:ct:001",
			NumeroVisible: "2026/CT-001",
			ReciboRef:     "recibo:ct-alta:001",
		},
		AmbitoIdempotenciaHMAC: selloPrueba(dominioAmbitoPrueba, "b"),
		HuellaPeticionHMAC:     huellas.datos.Activo.Valor,
		OrganizacionRef:        solicitud.OrganizacionRef,
		ActorRef:               solicitud.ActorRef,
		PerfilRef:              solicitud.PerfilRef,
		Estado:                 PreparacionReservada,
	}
	if err := base.ValidarPara(solicitud); err != nil {
		t.Fatalf("preparación válida rechazada: %v", err)
	}

	casos := map[string]func(*PreparacionAlta){
		"tenant": func(p *PreparacionAlta) {
			p.OrganizacionRef = "organizacion:ajena"
		},
		"actor": func(p *PreparacionAlta) {
			p.ActorRef = "actor:ajeno"
		},
		"perfil": func(p *PreparacionAlta) {
			p.PerfilRef = "perfil:ajeno"
		},
		"peticion": func(p *PreparacionAlta) {
			p.HuellaPeticionHMAC = selloPrueba(dominioHuellaPrueba, "c")
		},
	}
	for nombre, adulterar := range casos {
		t.Run(nombre, func(t *testing.T) {
			preparacion := base
			adulterar(&preparacion)
			if err := preparacion.ValidarPara(solicitud); err == nil {
				t.Fatal("se aceptó una preparación adulterada")
			}
		})
	}
}

func TestSolicitudPrepararAltaExigeClaveIdempotenciaUUIDv4Canonica(t *testing.T) {
	base := SolicitudPrepararAlta{
		ClaveIdempotencia: "018f3b2a-7c4d-4e5f-8a9b-0c1d2e3f4a5b",
		HuellasPeticionHMAC: coleccionSelloPrueba(
			t,
			selloPrueba(dominioHuellaPrueba, "a"),
		),
		OrganizacionRef: "organizacion:diputacion-granada",
		ActorRef:        "actor:tecnica-rrhh-001",
		PerfilRef:       "perfil:tecnica-rrhh",
	}
	if err := base.Validar(); err != nil {
		t.Fatalf("UUIDv4 canónico rechazado: %v", err)
	}
	for _, clave := range []string{
		"aaaaaaaaaaaaaaaaaaaaaa",
		"018f3b2a-7c4d-1e5f-8a9b-0c1d2e3f4a5b",
		"018F3B2A-7C4D-4E5F-8A9B-0C1D2E3F4A5B",
		"00000000-0000-4000-8000-000000000000",
	} {
		solicitud := base
		solicitud.ClaveIdempotencia = clave
		if err := solicitud.Validar(); err == nil {
			t.Fatalf("clave trivial/no canónica aceptada: %q", clave)
		}
	}
}

func TestReferenciasAltaRechazaNumeroVisibleFueraDeContrato(t *testing.T) {
	base := ReferenciasAlta{
		ExpedienteRef: "expediente:ct:001",
		NumeroVisible: "2026/CT-001",
		ReciboRef:     "recibo:ct-alta:001",
	}
	if err := base.Validar(); err != nil {
		t.Fatalf("referencias válidas rechazadas: %v", err)
	}
	for _, numero := range []string{
		"CT-001", "20/CT-001", "2026/", "2026/CT 001", "2026/CT/001",
		"2026/" + strings.Repeat("a", 41),
	} {
		referencias := base
		referencias.NumeroVisible = numero
		if err := referencias.Validar(); err == nil {
			t.Fatalf("número visible inválido aceptado: %q", numero)
		}
	}
}

func selloPrueba(dominio, caracter string) string {
	return "hmac-sha256:" + dominio + ":" + strings.Repeat(caracter, 64)
}

func coleccionSelloPrueba(t *testing.T, activo string) ColeccionSellosHMAC {
	t.Helper()
	coleccion, err := NuevaColeccionSellosHMAC(activo, nil)
	if err != nil {
		t.Fatal(err)
	}
	return coleccion
}

func TestColeccionesHMACAltaExigenGeneracionesAlineadasYParInseparable(t *testing.T) {
	ambitoV2 := selloPrueba(
		"vec.contratacion-temporal.ambito-idempotencia/v2",
		"b",
	)
	ambitoV1 := selloPrueba(dominioAmbitoPrueba, "a")
	huellaV2 := selloPrueba(
		"vec.contratacion-temporal.huella-peticion/v2",
		"b",
	)
	huellaV1 := selloPrueba(dominioHuellaPrueba, "a")
	ambitos, err := NuevaColeccionSellosHMAC(ambitoV2, []string{ambitoV1})
	if err != nil {
		t.Fatal(err)
	}
	huellas, err := NuevaColeccionSellosHMAC(huellaV2, []string{huellaV1})
	if err != nil {
		t.Fatal(err)
	}
	activoAmbito, activaHuella, err := ParActivoColeccionesHMACAlta(
		ambitos,
		huellas,
	)
	if err != nil || activoAmbito != ambitoV2 || activaHuella != huellaV2 {
		t.Fatalf("par activo incorrecto: %q %q %v", activoAmbito, activaHuella, err)
	}
	if !ColeccionesHMACAltaContienenPar(
		ambitos,
		huellas,
		ambitoV1,
		huellaV1,
	) || ColeccionesHMACAltaContienenPar(
		ambitos,
		huellas,
		ambitoV1,
		huellaV2,
	) {
		t.Fatal("se separó el par HMAC generacional")
	}
	huellasSoloActiva, err := NuevaColeccionSellosHMAC(huellaV2, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := ParActivoColeccionesHMACAlta(
		ambitos,
		huellasSoloActiva,
	); err == nil {
		t.Fatal("colecciones con historias generacionales distintas aceptadas")
	}
}
