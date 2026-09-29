package application

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type conectorPortafirmasDoble struct {
	estado    ports.EstadoConexionPortafirmas
	errEstado error
	recibo    ports.ReciboEnvioPortafirmas
	errEnvio  error
	envios    int
}

func (c *conectorPortafirmasDoble) EstadoConexion(context.Context) (ports.EstadoConexionPortafirmas, error) {
	return c.estado, c.errEstado
}

func (c *conectorPortafirmasDoble) Enviar(context.Context, ports.SolicitudEnvioPortafirmas) (ports.ReciboEnvioPortafirmas, error) {
	c.envios++
	return c.recibo, c.errEnvio
}

func solicitudPortafirmas() ports.SolicitudEnvioPortafirmas {
	original := []byte("%PDF-1.7 resolucion")
	return ports.SolicitudEnvioPortafirmas{
		OrganizacionRef: "org:diputacion", ExpedienteRef: "expediente:ct:001", VersionExpediente: 7,
		Documento: "resolucion", OriginalHuella: huella(original), Original: original,
		ClaveIdempotencia: "envio-portafirmas-0001",
	}
}

func TestEstadoPortafirmasFallaCerrado(t *testing.T) {
	ctx := context.Background()
	casos := map[string]ports.ConectorPortafirmas{
		"nulo":         nil,
		"puntero nulo": (*conectorPortafirmasDoble)(nil),
		"error":        &conectorPortafirmasDoble{errEstado: errors.New("caído")},
		"incoherente":  &conectorPortafirmasDoble{estado: ports.EstadoConexionPortafirmas{Conectado: true, Motivo: "x_y_z"}},
		"sin motivo":   &conectorPortafirmasDoble{estado: ports.EstadoConexionPortafirmas{}},
		"no conectado": &conectorPortafirmasDoble{estado: ports.EstadoConexionPortafirmas{Motivo: ports.MotivoPortafirmasConexionPendiente}},
	}
	for nombre, conector := range casos {
		if e := EstadoPortafirmas(ctx, conector); e.Conectado || e.Motivo != ports.MotivoPortafirmasConexionPendiente {
			t.Errorf("%s: %+v", nombre, e)
		}
	}
	if e := EstadoPortafirmas(ctx, &conectorPortafirmasDoble{estado: ports.EstadoConexionPortafirmas{Conectado: true}}); !e.Conectado {
		t.Errorf("un conector coherente y conectado se respeta: %+v", e)
	}
}

func TestEnviarAPortafirmasApagadoNoDejaConstancia(t *testing.T) {
	apagado := &conectorPortafirmasDoble{estado: ports.EstadoConexionPortafirmas{Motivo: ports.MotivoPortafirmasConexionPendiente},
		errEnvio: ports.ErrPortafirmasNoDisponible}
	recibo, err := EnviarAPortafirmas(context.Background(), apagado, solicitudPortafirmas())
	if !errors.Is(err, ports.ErrPortafirmasNoDisponible) || recibo != (ports.ReciboEnvioPortafirmas{}) {
		t.Fatalf("apagado: %+v, %v", recibo, err)
	}
}

func TestEnviarAPortafirmasExigeReciboDelMismoDocumento(t *testing.T) {
	sol := solicitudPortafirmas()
	conectado := ports.EstadoConexionPortafirmas{Conectado: true}
	desconectado := &conectorPortafirmasDoble{estado: ports.EstadoConexionPortafirmas{Motivo: "conexion_pendiente"}}
	if _, err := EnviarAPortafirmas(context.Background(), desconectado, sol); !errors.Is(err, ports.ErrPortafirmasNoDisponible) || desconectado.envios != 0 {
		t.Fatalf("sin conexión no se intenta el envío: %v, %d", err, desconectado.envios)
	}
	for nombre, doble := range map[string]*conectorPortafirmasDoble{
		"error":          {estado: conectado, errEnvio: errors.New("503")},
		"otra huella":    {estado: conectado, recibo: ports.ReciboEnvioPortafirmas{EnvioRef: "envio:1", OriginalHuella: huella([]byte("otro"))}},
		"sin referencia": {estado: conectado, recibo: ports.ReciboEnvioPortafirmas{OriginalHuella: sol.OriginalHuella}},
	} {
		if r, err := EnviarAPortafirmas(context.Background(), doble, sol); !errors.Is(err, ports.ErrPortafirmasNoDisponible) || r != (ports.ReciboEnvioPortafirmas{}) {
			t.Errorf("%s: %+v, %v", nombre, r, err)
		}
	}
	bueno := &conectorPortafirmasDoble{estado: conectado, recibo: ports.ReciboEnvioPortafirmas{EnvioRef: "envio:1", OriginalHuella: sol.OriginalHuella}}
	if r, err := EnviarAPortafirmas(context.Background(), bueno, sol); err != nil || r.EnvioRef != "envio:1" {
		t.Fatalf("envío aceptado: %+v, %v", r, err)
	}
	mala := sol
	mala.OriginalHuella = huella([]byte("otro"))
	if _, err := EnviarAPortafirmas(context.Background(), bueno, mala); !errors.Is(err, ports.ErrSolicitudFirmaDocumentoInvalida) {
		t.Fatalf("huella que no es la del original: %v", err)
	}
}
