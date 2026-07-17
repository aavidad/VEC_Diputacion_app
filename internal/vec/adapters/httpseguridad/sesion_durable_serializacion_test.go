package httpseguridad

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestDatosDurablesNoFiltranNiCreanCapacidadSerializable(t *testing.T) {
	ahora := time.Date(2026, 7, 18, 9, 0, 0, 0, time.UTC)
	configuracion := configuracionInternaValida()
	verificador := &verificadorFalso{}
	registro := nuevoRegistroMemoria()
	servicio := debeServicio(t, configuracion, verificador, evaluadorValido(dominiovec.AuthAssuranceHigh), registro, &relojFijo{ahora: ahora})
	canal := debeCanalTLS(t, servicio, configuracion)
	verificador.fijarAsercion(asercionInternaValida(ahora, configuracion, canal))
	identidad, err := servicio.Resolver(context.Background(), debeCredencial(t, []byte("documento-confidencial"), canal))
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	_, contextoAuditoria, err := servicio.ProyectarPrincipal(context.Background(), identidad)
	if err != nil {
		t.Fatalf("proyectar: %v", err)
	}
	registro.mu.Lock()
	consulta := registro.ultimaConsulta
	registro.mu.Unlock()

	valores := []struct {
		nombre string
		valor  any
		json   func([]byte) error
		gob    func([]byte) error
	}{
		{"alta", identidad.confirmacion.AltaConfirmada,
			func(datos []byte) error { var v AltaSesionAtomica; return json.Unmarshal(datos, &v) },
			func(datos []byte) error {
				var v AltaSesionAtomica
				return gob.NewDecoder(bytes.NewReader(datos)).Decode(&v)
			}},
		{"confirmacion", identidad.confirmacion,
			func(datos []byte) error { var v ConfirmacionAltaSesion; return json.Unmarshal(datos, &v) },
			func(datos []byte) error {
				var v ConfirmacionAltaSesion
				return gob.NewDecoder(bytes.NewReader(datos)).Decode(&v)
			}},
		{"consulta", consulta,
			func(datos []byte) error { var v ConsultaSesionActiva; return json.Unmarshal(datos, &v) },
			func(datos []byte) error {
				var v ConsultaSesionActiva
				return gob.NewDecoder(bytes.NewReader(datos)).Decode(&v)
			}},
	}
	prohibidos := []string{
		"asercion-001", "sesion-001", "persona-001", "cuenta-tecnica",
		configuracion.EmisorIdentidad,
		identidad.confirmacion.AutenticacionRef, identidad.confirmacion.AsercionRef,
		identidad.confirmacion.SesionRef, identidad.confirmacion.ControlSesionRef,
		identidad.confirmacion.AltaConfirmada.AutenticacionHuellaSHA256,
	}
	for _, caso := range valores {
		t.Run(caso.nombre, func(t *testing.T) {
			jsonRedactado, err := json.Marshal(caso.valor)
			if err != nil {
				t.Fatalf("JSON: %v", err)
			}
			var gobRedactado bytes.Buffer
			if err := gob.NewEncoder(&gobRedactado).Encode(caso.valor); err != nil {
				t.Fatalf("gob: %v", err)
			}
			var log bytes.Buffer
			slog.New(slog.NewTextHandler(&log, nil)).Info("dato", "valor", caso.valor)
			representaciones := []string{
				fmt.Sprintf("%s %v %#v %+v", caso.valor, caso.valor, caso.valor, caso.valor),
				string(jsonRedactado), gobRedactado.String(), log.String(),
			}
			for _, representacion := range representaciones {
				for _, prohibido := range prohibidos {
					if strings.Contains(representacion, prohibido) {
						t.Fatalf("dato filtrado (%q): %q", prohibido, representacion)
					}
				}
			}
			if err := caso.json(jsonRedactado); !errors.Is(err, ErrDatosSesionNoSerializables) {
				t.Fatalf("JSON reconstruyo el dato: %v", err)
			}
			if err := caso.gob(gobRedactado.Bytes()); !errors.Is(err, ErrDatosSesionNoSerializables) {
				t.Fatalf("gob reconstruyo el dato: %v", err)
			}
		})
	}

	jsonContexto, err := json.Marshal(contextoAuditoria)
	if err != nil {
		t.Fatalf("JSON contexto: %v", err)
	}
	textoContexto, err := contextoAuditoria.MarshalText()
	if err != nil {
		t.Fatalf("texto contexto: %v", err)
	}
	binarioContexto, err := contextoAuditoria.MarshalBinary()
	if err != nil {
		t.Fatalf("binario contexto: %v", err)
	}
	var gobContexto bytes.Buffer
	if err := gob.NewEncoder(&gobContexto).Encode(contextoAuditoria); err != nil {
		t.Fatalf("gob contexto: %v", err)
	}
	var logContexto bytes.Buffer
	slog.New(slog.NewTextHandler(&logContexto, nil)).Info("contexto", "valor", contextoAuditoria)
	for _, representacion := range []string{
		fmt.Sprintf("%s %v %#v %+v", contextoAuditoria, contextoAuditoria, contextoAuditoria, contextoAuditoria),
		string(jsonContexto), string(textoContexto), string(binarioContexto), gobContexto.String(), logContexto.String(),
	} {
		for _, prohibido := range []string{
			"asercion-001", "sesion-001", "persona-001", "cuenta-tecnica",
			contextoAuditoria.AutenticacionRef(), contextoAuditoria.AsercionRef(),
			contextoAuditoria.SesionRef(), contextoAuditoria.ControlSesionRef(),
		} {
			if strings.Contains(representacion, prohibido) {
				t.Fatalf("contexto de auditoria filtrado (%q): %q", prohibido, representacion)
			}
		}
	}
	var reconstruido ContextoAuditoriaAutenticada
	if err := json.Unmarshal(jsonContexto, &reconstruido); !errors.Is(err, ErrContextoAuditoriaNoSerializable) {
		t.Fatalf("JSON reconstruyo contexto de auditoria: %v", err)
	}
	if err := gob.NewDecoder(bytes.NewReader(gobContexto.Bytes())).Decode(&reconstruido); !errors.Is(err, ErrContextoAuditoriaNoSerializable) {
		t.Fatalf("gob reconstruyo contexto de auditoria: %v", err)
	}
}
