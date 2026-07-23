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
	solicitud := SolicitudPrepararAlta{
		ClaveIdempotencia:  "01J2F8X4K4R9T2Y7W3M6Q8P1AB",
		HuellaPeticionHMAC: selloPrueba(dominioHuellaPrueba, "a"),
		OrganizacionRef:    "organizacion:diputacion-granada",
		ActorRef:           "actor:tecnica-rrhh-001",
		PerfilRef:          "perfil:tecnica-rrhh",
	}
	base := PreparacionAlta{
		ReservaRef: "reserva:ct-alta:001",
		Referencias: ReferenciasAlta{
			ExpedienteRef: "expediente:ct:001",
			NumeroVisible: "2026/CT-001",
			ReciboRef:     "recibo:ct-alta:001",
		},
		AmbitoIdempotenciaHMAC: selloPrueba(dominioAmbitoPrueba, "b"),
		HuellaPeticionHMAC:     solicitud.HuellaPeticionHMAC,
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
