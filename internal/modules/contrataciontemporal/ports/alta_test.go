package ports

import (
	"reflect"
	"strconv"
	"strings"
	"testing"
)

const (
	dominioHuellaPrueba = "vec.contratacion-temporal.huella-peticion/v1"
	dominioAmbitoPrueba = "vec.contratacion-temporal.ambito-idempotencia/v1"
)

func identidadHMACPrueba(generacion uint32, digito string) DatosIdentidadHMACAlta {
	version := strconv.FormatUint(uint64(generacion), 10)
	return DatosIdentidadHMACAlta{
		Generacion: generacion,
		AmbitoIdempotenciaHMAC: selloPrueba(
			"vec.contratacion-temporal.ambito-idempotencia/v"+version,
			digito,
		),
		HuellaPeticionHMAC: selloPrueba(
			"vec.contratacion-temporal.huella-peticion/v"+version,
			digito,
		),
	}
}

func coleccionIdentidadesHMACPrueba(
	t *testing.T,
	activa DatosIdentidadHMACAlta,
	retenidas ...DatosIdentidadHMACAlta,
) ColeccionIdentidadesHMACAlta {
	t.Helper()
	coleccion, err := NuevaColeccionIdentidadesHMACAlta(activa, retenidas)
	if err != nil {
		t.Fatal(err)
	}
	return coleccion
}

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
	activa := identidadHMACPrueba(2, "b")
	retenida := identidadHMACPrueba(1, "a")
	solicitud := SolicitudPrepararAlta{
		ClaveIdempotencia: "018f3b2a-7c4d-4e5f-8a9b-0c1d2e3f4a5b",
		IdentidadesHMAC:   coleccionIdentidadesHMACPrueba(t, activa, retenida),
		OrganizacionRef:   "organizacion:diputacion-granada",
		ActorRef:          "actor:tecnica-rrhh-001",
		PerfilRef:         "perfil:tecnica-rrhh",
	}
	base := PreparacionAlta{
		ReservaRef: "reserva:ct-alta:001",
		Referencias: ReferenciasAlta{
			ExpedienteRef: "expediente:ct:001",
			NumeroVisible: "2026/CT-001",
			ReciboRef:     "recibo:ct-alta:001",
		},
		AmbitoIdempotenciaHMAC: activa.AmbitoIdempotenciaHMAC,
		HuellaPeticionHMAC:     activa.HuellaPeticionHMAC,
		OrganizacionRef:        solicitud.OrganizacionRef,
		ActorRef:               solicitud.ActorRef,
		PerfilRef:              solicitud.PerfilRef,
		Estado:                 PreparacionReservada,
	}
	if err := base.ValidarPara(solicitud); err != nil {
		t.Fatalf("preparación válida rechazada: %v", err)
	}
	repeticionRetenida := base
	repeticionRetenida.AmbitoIdempotenciaHMAC = retenida.AmbitoIdempotenciaHMAC
	repeticionRetenida.HuellaPeticionHMAC = retenida.HuellaPeticionHMAC
	if err := repeticionRetenida.ValidarPara(solicitud); err != nil {
		t.Fatalf("repetición de generación retenida rechazada: %v", err)
	}
	parMezclado := base
	parMezclado.AmbitoIdempotenciaHMAC = retenida.AmbitoIdempotenciaHMAC
	if err := parMezclado.ValidarPara(solicitud); err == nil {
		t.Fatal("se aceptó ámbito retenido con huella activa")
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
		"ambito": func(p *PreparacionAlta) {
			p.AmbitoIdempotenciaHMAC = selloPrueba(dominioAmbitoPrueba, "c")
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

func TestColeccionIdentidadesHMACAltaLigaGeneracionesYClonaRetenidas(t *testing.T) {
	base := coleccionIdentidadesHMACPrueba(
		t,
		identidadHMACPrueba(2, "b"),
		identidadHMACPrueba(1, "a"),
	)
	clon, err := base.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	datosClon, err := clon.Datos()
	if err != nil {
		t.Fatal(err)
	}
	datosClon.Retenidas[0] = DatosIdentidadHMACAlta{}
	datosBase, err := base.Datos()
	if err != nil || validarDatosIdentidadHMACAlta(datosBase.Retenidas[0]) != nil {
		t.Fatal("la copia modificó las generaciones retenidas")
	}

	mutacionGeneracion := identidadHMACPrueba(2, "b")
	mutacionGeneracion.Generacion = 1
	parMezclado := identidadHMACPrueba(2, "b")
	parMezclado.HuellaPeticionHMAC = identidadHMACPrueba(
		1,
		"a",
	).HuellaPeticionHMAC
	dominiosInvertidos := identidadHMACPrueba(2, "b")
	dominiosInvertidos.AmbitoIdempotenciaHMAC,
		dominiosInvertidos.HuellaPeticionHMAC =
		dominiosInvertidos.HuellaPeticionHMAC,
		dominiosInvertidos.AmbitoIdempotenciaHMAC
	casos := map[string]struct {
		activa    DatosIdentidadHMACAlta
		retenidas []DatosIdentidadHMACAlta
	}{
		"activa cero":                   {},
		"generacion declarada distinta": {activa: mutacionGeneracion},
		"generacion repetida": {
			activa: identidadHMACPrueba(2, "b"),
			retenidas: []DatosIdentidadHMACAlta{
				identidadHMACPrueba(2, "c"),
			},
		},
		"retenida superior": {
			activa: identidadHMACPrueba(2, "b"),
			retenidas: []DatosIdentidadHMACAlta{
				identidadHMACPrueba(3, "c"),
			},
		},
		"retenidas desordenadas": {
			activa: identidadHMACPrueba(4, "d"),
			retenidas: []DatosIdentidadHMACAlta{
				identidadHMACPrueba(2, "b"),
				identidadHMACPrueba(3, "c"),
			},
		},
		"par mezclado":        {activa: parMezclado},
		"dominios invertidos": {activa: dominiosInvertidos},
		"demasiadas": {
			activa: identidadHMACPrueba(5, "e"),
			retenidas: []DatosIdentidadHMACAlta{
				identidadHMACPrueba(4, "d"),
				identidadHMACPrueba(3, "c"),
				identidadHMACPrueba(2, "b"),
				identidadHMACPrueba(1, "a"),
			},
		},
	}
	maxima := coleccionIdentidadesHMACPrueba(
		t,
		identidadHMACPrueba(4, "d"),
		identidadHMACPrueba(3, "c"),
		identidadHMACPrueba(2, "b"),
		identidadHMACPrueba(1, "a"),
	)
	if maxima.Validar() != nil {
		t.Fatal("se rechazó el máximo de cuatro generaciones")
	}
	for nombre, candidata := range casos {
		t.Run(nombre, func(t *testing.T) {
			if _, err := NuevaColeccionIdentidadesHMACAlta(
				candidata.activa,
				candidata.retenidas,
			); err == nil {
				t.Fatal("colección HMAC inválida aceptada")
			}
		})
	}
}

func TestColeccionIdentidadesHMACAltaNoExponeCamposFabricables(t *testing.T) {
	tipo := reflect.TypeOf(ColeccionIdentidadesHMACAlta{})
	if tipo.NumField() != 1 || tipo.Field(0).IsExported() {
		t.Fatalf("la capacidad HMAC expone representación pública: %v", tipo)
	}
}

func TestSolicitudPrepararAltaRechazaColeccionHMACNula(t *testing.T) {
	solicitud := SolicitudPrepararAlta{
		ClaveIdempotencia: "018f3b2a-7c4d-4e5f-8a9b-0c1d2e3f4a5b",
		OrganizacionRef:   "organizacion:diputacion-granada",
		ActorRef:          "actor:tecnica-rrhh-001",
		PerfilRef:         "perfil:tecnica-rrhh",
	}
	if solicitud.Validar() == nil {
		t.Fatal("preparación con ámbito y huella de generaciones distintas aceptada")
	}
}

func TestSolicitudPrepararAltaExigeClaveIdempotenciaUUIDv4Canonica(t *testing.T) {
	base := SolicitudPrepararAlta{
		ClaveIdempotencia: "018f3b2a-7c4d-4e5f-8a9b-0c1d2e3f4a5b",
		IdentidadesHMAC: coleccionIdentidadesHMACPrueba(
			t,
			identidadHMACPrueba(1, "a"),
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
