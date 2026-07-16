package memory

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestRepositorioFlujoFirmaRechazaArrendamientoConCercadoObsoleto(t *testing.T) {
	ctx := context.Background()
	reloj := &relojMemoriaPrueba{instante: instanteMemoriaPrueba}
	sellador := selladorEstadoFlujoMemoriaPrueba{}
	repositorio, err := NuevoRepositorioFlujosFirmaBaremacion(reloj, sellador)
	if err != nil {
		t.Fatal(err)
	}
	protector, err := NuevoProtectorEstadoFlujoFirmaBaremacion(
		"clave-estado-flujo-firma-v1",
		[]byte("0123456789abcdef0123456789abcdef"),
	)
	if err != nil {
		t.Fatal(err)
	}
	carga, err := puertosbolsa.NuevaCargaProtegida([]byte("estado-inicial-sin-capacidades"))
	if err != nil {
		t.Fatal(err)
	}
	estado, err := protector.ProtegerEstadoFlujoFirmaBaremacion(ctx, carga)
	if err != nil {
		t.Fatal(err)
	}
	expediente := sellarExpedienteFlujoMemoriaPrueba(t, sellador, puertosbolsa.ExpedienteFlujoFirmaBaremacion{
		FlujoRef: "flujo-firma-cercado-001", Version: 1,
		IndiceIdempotenciaHMAC: hmacEstadoFlujoMemoriaPrueba("indice"),
		HuellaSolicitudHMAC:    hmacEstadoFlujoMemoriaPrueba("solicitud"),
		VinculoActorHMAC:       hmacEstadoFlujoMemoriaPrueba("actor"),
		PerfilActorClave:       perfilBaremacionMemoriaPrueba,
		ProcesoRef:             "proceso-001",
		SolicitudRef:           "solicitud-001",
		BaremacionMeritoRef:    "baremacion-001",
		DecisionRef:            "decision-001",
		Estado:                 puertosbolsa.EstadoExpedienteFirmaPreparando,
		EstadoProtegido:        estado,
		CreadoEn:               instanteMemoriaPrueba,
		ActualizadoEn:          instanteMemoriaPrueba,
	})
	if _, err := repositorio.CrearORecuperarFlujoFirmaBaremacion(
		ctx,
		puertosbolsa.SolicitudCrearORecuperarFlujoFirmaBaremacion{Expediente: expediente},
	); err != nil {
		t.Fatal(err)
	}
	consulta := puertosbolsa.SolicitudObtenerFlujoFirmaBaremacion{
		FlujoRef: expediente.FlujoRef, IndiceIdempotenciaHMAC: expediente.IndiceIdempotenciaHMAC,
		VinculoActorHMAC: expediente.VinculoActorHMAC,
	}
	primero, err := repositorio.AdquirirArrendamientoFlujoFirmaBaremacion(
		ctx,
		puertosbolsa.SolicitudAdquirirArrendamientoFlujoFirmaBaremacion{
			Consulta: consulta, VersionEsperada: 1,
			PropietarioRef: "propietario-antiguo", Duracion: time.Second,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	reloj.fijar(instanteMemoriaPrueba.Add(2 * time.Second))
	segundo, err := repositorio.AdquirirArrendamientoFlujoFirmaBaremacion(
		ctx,
		puertosbolsa.SolicitudAdquirirArrendamientoFlujoFirmaBaremacion{
			Consulta: consulta, VersionEsperada: 1,
			PropietarioRef: "propietario-vigente", Duracion: time.Second,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if segundo.Arrendamiento.SecuenciaCercado <= primero.Arrendamiento.SecuenciaCercado {
		t.Fatalf("cercado no monotono: primero=%d segundo=%d", primero.Arrendamiento.SecuenciaCercado, segundo.Arrendamiento.SecuenciaCercado)
	}

	siguiente, err := expediente.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	siguiente.Version = 2
	siguiente.ActualizadoEn = reloj.Ahora()
	siguiente.PuntosControl = append(siguiente.PuntosControl, puertosbolsa.PuntoControlFirmaBaremacion{
		Paso: puertosbolsa.PasoPrepararFirmaBaremacion, Estado: puertosbolsa.EstadoPuntoControlFirmaDeclarado,
		EfectoRef: "efecto-preparacion-001", ClaveIdempotenciaHMAC: hmacEstadoFlujoMemoriaPrueba("efecto"),
		DeclaradoEn: reloj.Ahora(),
	})
	siguiente.SelloEstadoHMAC = ""
	siguiente = sellarExpedienteFlujoMemoriaPrueba(t, sellador, siguiente)

	_, err = repositorio.GuardarFlujoFirmaBaremacion(ctx, puertosbolsa.SolicitudGuardarFlujoFirmaBaremacion{
		VersionEsperada: 1, Arrendamiento: primero.Arrendamiento, Siguiente: siguiente,
	})
	if !errors.Is(err, puertosbolsa.ErrArrendamientoFlujoFirmaInvalido) {
		t.Fatalf("Guardar() con cercado obsoleto error = %v", err)
	}
	guardado, err := repositorio.GuardarFlujoFirmaBaremacion(ctx, puertosbolsa.SolicitudGuardarFlujoFirmaBaremacion{
		VersionEsperada: 1, Arrendamiento: segundo.Arrendamiento, Siguiente: siguiente,
	})
	if err != nil {
		t.Fatalf("Guardar() con cercado vigente error = %v", err)
	}
	if guardado.Version != 2 || len(guardado.PuntosControl) != 1 {
		t.Fatalf("estado guardado inesperado: version=%d puntos=%d", guardado.Version, len(guardado.PuntosControl))
	}
}

type selladorEstadoFlujoMemoriaPrueba struct{}

func (selladorEstadoFlujoMemoriaPrueba) SellarSolicitudBaremacion(
	ctx context.Context,
	carga puertosbolsa.CargaProtegida,
) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if carga.Validar() != nil {
		return "", puertosbolsa.ErrCargaProtegidaInvalida
	}
	return hmacEstadoFlujoMemoriaPrueba(string(carga.Revelar())), nil
}

func (s selladorEstadoFlujoMemoriaPrueba) VerificarEstadoFlujoFirmaBaremacion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudVerificarEstadoFlujoFirmaBaremacion,
) error {
	if solicitud.Validar() != nil {
		return puertosbolsa.ErrEstadoFlujoFirmaAlterado
	}
	esperado, err := s.SellarSolicitudBaremacion(ctx, solicitud.RepresentacionCanonica)
	if err != nil || !hmac.Equal([]byte(esperado), []byte(solicitud.SelloHMAC)) {
		return puertosbolsa.ErrEstadoFlujoFirmaAlterado
	}
	return nil
}

func sellarExpedienteFlujoMemoriaPrueba(
	t *testing.T,
	sellador selladorEstadoFlujoMemoriaPrueba,
	expediente puertosbolsa.ExpedienteFlujoFirmaBaremacion,
) puertosbolsa.ExpedienteFlujoFirmaBaremacion {
	t.Helper()
	preparado, canonica, err := expediente.PrepararSellado()
	if err != nil {
		t.Fatal(err)
	}
	sello, err := sellador.SellarSolicitudBaremacion(context.Background(), canonica)
	if err != nil {
		t.Fatal(err)
	}
	sellado, err := preparado.IncorporarSello(sello)
	if err != nil {
		t.Fatal(err)
	}
	return sellado
}

func hmacEstadoFlujoMemoriaPrueba(valor string) string {
	mac := hmac.New(sha256.New, claveHMACMemoriaPrueba)
	_, _ = mac.Write([]byte(valor))
	return "hmac-sha256:flujo_firma_memoria_v1:" + hex.EncodeToString(mac.Sum(nil))
}
