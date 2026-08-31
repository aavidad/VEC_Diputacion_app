package ports

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
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

func TestMaterialHuellaAsignacionDistingueAltaYReasignacion(t *testing.T) {
	base := materialHuellaAsignacionPrueba()
	if err := base.Validar(); err != nil {
		t.Fatalf("material de alta válido rechazado: %v", err)
	}

	reasignacion := base
	reasignacion.Operacion = OperacionRegistrarReasignacion
	reasignacion.MotivoReasignacionClave = "necesidad_servicio"
	reasignacion.Observaciones = "Cambio motivado de unidad responsable."
	if err := reasignacion.Validar(); err != nil {
		t.Fatalf("material de reasignación válido rechazado: %v", err)
	}

	casos := map[string]func(*MaterialHuellaAsignacion){
		"alta con observaciones": func(m *MaterialHuellaAsignacion) {
			m.Observaciones = "No procede en el alta."
		},
		"reasignación sin motivo": func(m *MaterialHuellaAsignacion) {
			m.Operacion = OperacionRegistrarReasignacion
			m.Observaciones = "Falta el motivo catalogado."
		},
		"texto no canónico": func(m *MaterialHuellaAsignacion) {
			m.Operacion = OperacionRegistrarReasignacion
			m.MotivoReasignacionClave = "necesidad_servicio"
			m.Observaciones = " texto con espacio"
		},
		"versión no incrementable": func(m *MaterialHuellaAsignacion) {
			m.VersionExpediente = MaximoEnteroSeguroOperacionAnalisis
		},
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			material := base
			mutar(&material)
			if err := material.Validar(); !errors.Is(
				err,
				ErrPreparacionAsignacionInvalida,
			) {
				t.Fatalf("se esperaba rechazo cerrado, recibido %v", err)
			}
		})
	}
}

func TestSolicitudPrepararAsignacionExigeSellosYCoordenadas(t *testing.T) {
	solicitud := solicitudPrepararAsignacionPrueba(t)
	if err := solicitud.Validar(); err != nil {
		t.Fatalf("solicitud válida rechazada: %v", err)
	}

	otraHuella := coleccionSelloPrueba(
		t,
		selloPrueba("vec.otro-dominio/v1", "c"),
	)
	solicitud.HuellasPeticionHMAC = otraHuella
	if err := solicitud.Validar(); !errors.Is(
		err,
		ErrPreparacionAsignacionInvalida,
	) {
		t.Fatalf("se esperaba dominio HMAC rechazado, recibido %v", err)
	}
}

func TestConsultaAsignacionIdempotenteUsaSoloElParHMACActivo(t *testing.T) {
	solicitud := solicitudPrepararAsignacionPrueba(t)
	ambitoV1 := selloPrueba(
		DominioAmbitoIdempotenciaAsignacion+"/v1",
		"1",
	)
	huellaV1 := selloPrueba(
		DominioHuellaPeticionAsignacion+"/v1",
		"2",
	)
	ambitoV2 := selloPrueba(
		DominioAmbitoIdempotenciaAsignacion+"/v2",
		"3",
	)
	huellaV2 := selloPrueba(
		DominioHuellaPeticionAsignacion+"/v2",
		"4",
	)
	var err error
	solicitud.AmbitosHMAC, err = NuevaColeccionSellosHMAC(
		ambitoV2,
		[]string{ambitoV1},
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitud.HuellasPeticionHMAC, err = NuevaColeccionSellosHMAC(
		huellaV2,
		[]string{huellaV1},
	)
	if err != nil {
		t.Fatal(err)
	}

	consulta, err := NuevaSolicitudConsultarAsignacionIdempotente(solicitud)
	if err != nil {
		t.Fatal(err)
	}
	if consulta.AmbitoIdempotenciaHMACActivo != ambitoV2 ||
		consulta.HuellaPeticionHMACActiva != huellaV2 ||
		consulta.Validar() != nil {
		t.Fatalf("la consulta no quedó ligada al par activo: %#v", consulta)
	}
	consulta.HuellaPeticionHMACActiva = huellaV1
	if consulta.Validar() == nil {
		t.Fatal("se aceptó un par HMAC de generaciones distintas")
	}
}

func TestEstadoCandidatoAsignacionNoSeSerializa(t *testing.T) {
	estado := EstadoCandidatoAsignacionIdempotente{}
	if _, err := json.Marshal(estado); !errors.Is(
		err,
		ErrResultadoAsignacionNoConfiable,
	) {
		t.Fatalf("estado opaco serializable: %v", err)
	}
	if _, err := estado.MarshalText(); !errors.Is(
		err,
		ErrResultadoAsignacionNoConfiable,
	) {
		t.Fatalf("estado opaco convertible a texto: %v", err)
	}
}

func TestReciboAsignacionQuedaLigadoALaPreparacion(t *testing.T) {
	solicitud := solicitudPrepararAsignacionPrueba(t)
	ambito, huella, err := ParActivoColeccionesHMAC(
		solicitud.AmbitosHMAC,
		DominioAmbitoIdempotenciaAsignacion,
		solicitud.HuellasPeticionHMAC,
		DominioHuellaPeticionAsignacion,
	)
	if err != nil {
		t.Fatal(err)
	}
	preparacion := PreparacionAsignacion{
		Expediente: domain.Expediente{
			Referencia:      solicitud.ExpedienteRef,
			OrganizacionRef: solicitud.OrganizacionRef,
			Version:         solicitud.VersionExpediente,
		},
		Referencias: ReferenciasEfectoAsignacion{
			ReservaRef:      "reserva:asignacion:0123456789abcdef",
			ReciboRef:       "recibo:asignacion:0123456789abcdef",
			NotificacionRef: "notificacion:asignacion:0123456789abcdef",
			BandejaRef:      "bandeja:asignacion:0123456789abcdef",
			AuditoriaRef:    "auditoria:asignacion:0123456789abcdef",
			EventoRef:       "evento:asignacion:0123456789abcdef",
		},
		AmbitoIdempotenciaHMAC: ambito,
		HuellaPeticionHMAC:     huella,
		Operacion:              solicitud.Operacion,
		OrganizacionRef:        solicitud.OrganizacionRef,
		ActorRef:               solicitud.ActorRef,
		PerfilRef:              solicitud.PerfilRef,
		UnidadRef:              solicitud.UnidadRef,
		ResponsableRef:         solicitud.ResponsableRef,
		Estado:                 PreparacionAsignacionReservada,
	}
	recibo := reciboAsignacionPrueba(preparacion)
	if err := recibo.ValidarParaPreparacion(preparacion); err != nil {
		t.Fatalf("recibo válido rechazado: %v", err)
	}

	recibo.BandejaRef = "bandeja:otra:0123456789abcdef"
	if err := recibo.ValidarParaPreparacion(preparacion); !errors.Is(
		err,
		ErrResultadoAsignacionNoConfiable,
	) {
		t.Fatalf("se esperaba recibo adulterado rechazado, recibido %v", err)
	}
}

func materialHuellaAsignacionPrueba() MaterialHuellaAsignacion {
	return MaterialHuellaAsignacion{
		Operacion:         OperacionRegistrarAsignacion,
		OrganizacionRef:   "organizacion:dipgra:0123456789abcdef",
		ExpedienteRef:     "expediente:contratacion:0123456789abcdef",
		VersionExpediente: 3,
		ActorRef:          "persona:tecnica:0123456789abcdef",
		PerfilRef:         "perfil:tecnico:0123456789abcdef",
		UnidadRef:         "unidad:seleccion:0123456789abcdef",
		ResponsableRef:    "persona:responsable:0123456789abcdef",
	}
}

func solicitudPrepararAsignacionPrueba(
	t *testing.T,
) SolicitudPrepararAsignacion {
	t.Helper()
	material := materialHuellaAsignacionPrueba()
	return SolicitudPrepararAsignacion{
		ClaveIdempotencia: "018f3b2a-7c4d-4e5f-8a9b-0c1d2e3f4a5b",
		AmbitosHMAC: coleccionSelloPrueba(
			t,
			selloPrueba(DominioAmbitoIdempotenciaAsignacion+"/v1", "a"),
		),
		HuellasPeticionHMAC: coleccionSelloPrueba(
			t,
			selloPrueba(DominioHuellaPeticionAsignacion+"/v1", "b"),
		),
		Operacion:         material.Operacion,
		OrganizacionRef:   material.OrganizacionRef,
		ExpedienteRef:     material.ExpedienteRef,
		VersionExpediente: material.VersionExpediente,
		ActorRef:          material.ActorRef,
		PerfilRef:         material.PerfilRef,
		UnidadRef:         material.UnidadRef,
		ResponsableRef:    material.ResponsableRef,
	}
}

func reciboAsignacionPrueba(
	preparacion PreparacionAsignacion,
) ReciboAsignacion {
	return ReciboAsignacion{
		Operacion:              preparacion.Operacion,
		OrganizacionRef:        preparacion.OrganizacionRef,
		ExpedienteRef:          preparacion.Expediente.Referencia,
		VersionAnterior:        preparacion.Expediente.Version,
		VersionResultante:      preparacion.Expediente.Version + 1,
		UnidadRef:              preparacion.UnidadRef,
		ResponsableRef:         preparacion.ResponsableRef,
		ReciboRef:              preparacion.Referencias.ReciboRef,
		NotificacionRef:        preparacion.Referencias.NotificacionRef,
		BandejaRef:             preparacion.Referencias.BandejaRef,
		AuditoriaRef:           preparacion.Referencias.AuditoriaRef,
		EventoRef:              preparacion.Referencias.EventoRef,
		ConcesionV3DecisionRef: "decision:asignacion:0123456789abcdef",
		AmbitoIdempotenciaHMAC: preparacion.AmbitoIdempotenciaHMAC,
		HuellaPeticionHMAC:     preparacion.HuellaPeticionHMAC,
		ConfirmadaEn:           time.Date(2026, 7, 24, 10, 0, 0, 0, time.UTC),
	}
}
