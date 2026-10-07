package bootstrap

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestPreparacionBasesSesionClasificaSoloFronterasNominales(t *testing.T) {
	for i := range 2 {
		broker, ctx := brokerPreparacionBasesPrueba(t, i)
		p := broker.sesiones[i]
		for _, caso := range []struct{ causa, esperada error }{
			{ctports.ErrConsultaRRHHNoDisponible, ctports.ErrConsultaRRHHNoDisponible},
			{errors.New("dependencia sintética"), ctports.ErrConsultaRRHHNoDisponible},
			{ctports.ErrAutorizacionDenegada, ErrSeguridadComunDesarrolloDenegada},
		} {
			if err := p.errorSesionConsultaComunicacionesExpediente(ctx, caso.causa); !errors.Is(err, caso.esperada) {
				t.Fatalf("ruta=%s clasificación=%v esperada=%v", broker.perfiles[i].ruta, err, caso.esperada)
			}
		}
		cancelado, cancelar := context.WithCancel(ctx)
		cancelar()
		if err := p.errorSesionConsultaComunicacionesExpediente(cancelado, context.Canceled); !errors.Is(err, context.Canceled) {
			t.Fatal("cancelación nominal perdió la causa técnica")
		}
		f := ctx.Value(claveFronteraSeguridadComunDesarrollo{}).(fronteraSeguridadComunDesarrollo)
		ds, err := fronterasPreparacionBasesHTTPV3(broker.perfiles[0].perfilRef(), broker.perfiles[1].perfilRef())
		if err != nil {
			t.Fatal(err)
		}
		f.catalogo, err = nuevoCatalogoFronterasComunDesarrollo(ds)
		if err != nil {
			t.Fatal(err)
		}
		ajeno := context.WithValue(ctx, claveFronteraSeguridadComunDesarrollo{}, f)
		if err := p.errorSesionConsultaComunicacionesExpediente(ajeno, ctports.ErrConsultaRRHHNoDisponible); !errors.Is(err, ErrSeguridadComunDesarrolloDenegada) {
			t.Fatal("otro catálogo accedió al clasificador nominal")
		}
	}
}

func TestPreparacionBasesRaizEmiteHolderSoloConCapacidadNominal(t *testing.T) {
	// Estado TLS unitario del middleware existente; no acredita handshake,
	// registro de sesión ni PostgreSQL.
	for i := range 2 {
		broker, ctx := brokerPreparacionBasesPrueba(t, i)
		canal := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
		raw := []byte("certificado_sintetico_preparacion_bases")
		h := sha256.Sum256(raw)
		canal.principal.Attributes["certificate_sha256"] = hex.EncodeToString(h[:])
		broker.perfiles[i].soporte.certificadoSHA256 = hex.EncodeToString(h[:])
		resolvedor, err := nuevoResolvedorIdentidadDesarrollo(identidadCertificadoDesarrollo{huella: h, principal: canal.principal})
		if err != nil {
			t.Fatal(err)
		}
		var anterior *contextoOperacionCTDesarrollo
		for _, vigente := range []bool{true, true, false} {
			ahora := time.Now().UTC().Truncate(time.Microsecond)
			certificado := &x509.Certificate{Raw: raw, NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour)}
			if !vigente {
				certificado.NotAfter = ahora.Add(-time.Second)
			}
			r := httptest.NewRequest(http.MethodPost, broker.perfiles[i].ruta, nil)
			r.RemoteAddr = "127.0.0.1:41000"
			r.TLS = &tls.ConnectionState{HandshakeComplete: true, Version: tls.VersionTLS13,
				PeerCertificates: []*x509.Certificate{certificado}, VerifiedChains: [][]*x509.Certificate{{certificado, {}}}}
			llamadas := 0
			m := &revalidadorConsultasContratacionTemporalDesarrollo{fronteras: broker.sesiones[i].fronteras,
				autoridad: &autoridadConsultasContratacionTemporalDesarrollo{sello: canal.sello, resolvedor: resolvedor},
				siguiente: http.HandlerFunc(func(_ http.ResponseWriter, peticion *http.Request) {
					llamadas++
					c, existe := peticion.Context().Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
					if !vigente {
						if existe && c.contextoOperacion != nil {
							t.Fatal("certificado caducado emitió holder S2")
						}
						return
					}
					if !existe || c.contextoOperacion == nil || c.contextoOperacion == anterior || c.contextoOperacion.soporte != nil {
						t.Fatal("la raíz no creó un holder nuevo para la petición S2")
					}
					if indice, err := broker.indice(peticion.Context()); err != nil || indice != i {
						t.Fatal("capacidad emitida por raíz no alcanzó broker nominal")
					}
					anterior = c.contextoOperacion
				})}
			m.ServeHTTP(httptest.NewRecorder(), r)
			if llamadas != 1 {
				t.Fatal("middleware no entregó la petición una vez")
			}
		}
	}
}
