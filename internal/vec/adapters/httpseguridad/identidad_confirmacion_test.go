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

type registroConfirmacionManipulada struct {
	base      *registroMemoria
	modificar func(*ConfirmacionAltaSesion, AltaSesionAtomica)
}

func (r *registroConfirmacionManipulada) ConsumirAsercionYRegistrar(
	ctx context.Context,
	alta AltaSesionAtomica,
) (ConfirmacionAltaSesion, error) {
	confirmacion, err := r.base.ConsumirAsercionYRegistrar(ctx, alta)
	if err == nil && r.modificar != nil {
		r.modificar(&confirmacion, alta)
	}
	return confirmacion, err
}

func (r *registroConfirmacionManipulada) ComprobarSesionYCuentaActivas(
	ctx context.Context,
	consulta ConsultaSesionActiva,
) error {
	return r.base.ComprobarSesionYCuentaActivas(ctx, consulta)
}

func TestResolverRechazaConfirmacionDuraderaNoLigadaAlAlta(t *testing.T) {
	pruebas := []struct {
		nombre    string
		modificar func(*ConfirmacionAltaSesion, AltaSesionAtomica)
	}{
		{"resultado vacio", func(c *ConfirmacionAltaSesion, _ AltaSesionAtomica) { *c = ConfirmacionAltaSesion{} }},
		{"autenticacion sin prefijo", func(c *ConfirmacionAltaSesion, _ AltaSesionAtomica) {
			c.AutenticacionRef = "xxx_0123456789abcdefghijkl"
		}},
		{"asercion sin prefijo", func(c *ConfirmacionAltaSesion, _ AltaSesionAtomica) { c.AsercionRef = "xxx_0123456789abcdefghijkl" }},
		{"sesion demasiado corta", func(c *ConfirmacionAltaSesion, _ AltaSesionAtomica) { c.SesionRef = "ses_corta" }},
		{"control con caracter invalido", func(c *ConfirmacionAltaSesion, _ AltaSesionAtomica) {
			c.ControlSesionRef = "cse_0123456789abcdefghijk!"
		}},
		{"cuenta no canonica", func(c *ConfirmacionAltaSesion, _ AltaSesionAtomica) { c.CuentaRef = "cuenta-tecnica" }},
		{"cuenta ordinaria distinta sin privilegio", func(c *ConfirmacionAltaSesion, _ AltaSesionAtomica) {
			c.CuentaOrdinariaRef = "cta_otra23456789abcdefghijkl"
		}},
		{"eco de cuenta alterado", func(c *ConfirmacionAltaSesion, _ AltaSesionAtomica) { c.AltaConfirmada.CuentaID = "otra-cuenta" }},
		{"eco de politica alterado", func(c *ConfirmacionAltaSesion, _ AltaSesionAtomica) { c.AltaConfirmada.PoliticaRef = "otra-politica" }},
		{"eco de tiempo alterado", func(c *ConfirmacionAltaSesion, _ AltaSesionAtomica) {
			c.AltaConfirmada.ExpiraEn = c.AltaConfirmada.ExpiraEn.Add(time.Second)
		}},
	}

	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			ahora := time.Date(2026, 7, 18, 8, 0, 0, 0, time.UTC)
			configuracion := configuracionInternaValida()
			verificador := &verificadorFalso{}
			registro := &registroConfirmacionManipulada{base: nuevoRegistroMemoria(), modificar: prueba.modificar}
			servicio := debeServicio(t, configuracion, verificador, evaluadorValido(dominiovec.AuthAssuranceHigh), registro, &relojFijo{ahora: ahora})
			canal := debeCanalTLS(t, servicio, configuracion)
			verificador.fijarAsercion(asercionInternaValida(ahora, configuracion, canal))

			if _, err := servicio.Resolver(context.Background(), debeCredencial(t, []byte("opaca"), canal)); !errors.Is(err, ErrSesionNoValida) {
				t.Fatalf("confirmacion manipulada aceptada: %v", err)
			}
		})
	}
}

func TestResolverNoAceptaReferenciasProcedentesDeLaAsercion(t *testing.T) {
	ahora := time.Date(2026, 7, 18, 8, 0, 0, 0, time.UTC)
	configuracion := configuracionInternaValida()
	verificador := &verificadorFalso{}
	registro := &registroConfirmacionManipulada{
		base: nuevoRegistroMemoria(),
		modificar: func(c *ConfirmacionAltaSesion, alta AltaSesionAtomica) {
			c.SesionRef = alta.SesionID
		},
	}
	servicio := debeServicio(t, configuracion, verificador, evaluadorValido(dominiovec.AuthAssuranceHigh), registro, &relojFijo{ahora: ahora})
	canal := debeCanalTLS(t, servicio, configuracion)
	asercion := asercionInternaValida(ahora, configuracion, canal)
	asercion.SesionID = "ses_entrada_cliente_0123456789"
	verificador.fijarAsercion(asercion)

	if _, err := servicio.Resolver(context.Background(), debeCredencial(t, []byte("opaca"), canal)); !errors.Is(err, ErrSesionNoValida) {
		t.Fatalf("la sesion declarada por la asercion no puede convertirse en referencia autoritativa: %v", err)
	}
}

func TestProyeccionConservaYRevalidaConfirmacionExacta(t *testing.T) {
	ahora := time.Date(2026, 7, 18, 8, 0, 0, 0, time.UTC)
	configuracion := configuracionInternaValida()
	verificador := &verificadorFalso{}
	registro := nuevoRegistroMemoria()
	servicio := debeServicio(t, configuracion, verificador, evaluadorValido(dominiovec.AuthAssuranceHigh), registro, &relojFijo{ahora: ahora})
	canal := debeCanalTLS(t, servicio, configuracion)
	verificador.fijarAsercion(asercionInternaValida(ahora, configuracion, canal))
	identidad, err := servicio.Resolver(context.Background(), debeCredencial(t, []byte("opaca"), canal))
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}

	principal, auditoria, err := servicio.ProyectarPrincipal(context.Background(), identidad)
	if err != nil {
		t.Fatalf("proyectar: %v", err)
	}
	confirmacion := identidad.confirmacion
	if principal.ID != "cuenta-tecnica" || auditoria.AutenticacionRef() != confirmacion.AutenticacionRef ||
		auditoria.AsercionRef() != confirmacion.AsercionRef || auditoria.SesionRef() != confirmacion.SesionRef ||
		auditoria.ControlSesionRef() != confirmacion.ControlSesionRef || auditoria.CuentaRef() != confirmacion.CuentaRef ||
		auditoria.CuentaOrdinariaRef() != confirmacion.CuentaOrdinariaRef ||
		auditoria.CuentaRef() != auditoria.CuentaOrdinariaRef() || auditoria.CuentaRef() == principal.ID {
		t.Fatalf("proyeccion durable incompleta: principal=%#v auditoria=%#v", principal, auditoria)
	}

	registro.mu.Lock()
	consulta := registro.ultimaConsulta
	registro.mu.Unlock()
	if consulta.AutenticacionRef != confirmacion.AutenticacionRef || consulta.AsercionRef != confirmacion.AsercionRef ||
		consulta.SesionRef != confirmacion.SesionRef || consulta.ControlSesionRef != confirmacion.ControlSesionRef ||
		consulta.CuentaRef != confirmacion.CuentaRef || consulta.CuentaOrdinariaRef != confirmacion.CuentaOrdinariaRef ||
		!altasSesionCoinciden(confirmacion.AltaConfirmada, AltaSesionAtomica{
			AsercionID: consulta.AsercionID, SesionID: consulta.SesionID, SujetoID: consulta.SujetoID,
			CuentaID: consulta.CuentaID, CuentaOrdinariaID: consulta.CuentaOrdinariaID,
			CuentaPrivilegiada: consulta.CuentaPrivilegiada, Superficie: consulta.Superficie,
			EmitidaEn: consulta.EmitidaEn, ExpiraEn: consulta.ExpiraEn,
			PoliticaRef: consulta.PoliticaRef, HuellaPolitica: consulta.HuellaPolitica,
		}) {
		t.Fatalf("consulta de revalidacion no reproduce la confirmacion: %#v", consulta)
	}

	registro.mu.Lock()
	alterada := registro.sesiones["sesion-001"]
	alterada.ControlSesionRef = "cse_alterado567890abcdefghijkl"
	registro.sesiones["sesion-001"] = alterada
	registro.mu.Unlock()
	if _, _, err := servicio.ProyectarPrincipal(context.Background(), identidad); !errors.Is(err, ErrSesionNoValida) {
		t.Fatalf("el registro alterado durante la revalidacion fue aceptado: %v", err)
	}
}

func TestCuentaPrivilegiadaConservaReferenciasDeCuentaSeparadas(t *testing.T) {
	ahora := time.Date(2026, 7, 18, 8, 0, 0, 0, time.UTC)
	configuracion := configuracionAdministracionValida()
	verificador := &verificadorFalso{}
	registro := nuevoRegistroMemoria()
	servicio := debeServicio(t, configuracion, verificador, evaluadorValido(dominiovec.AuthAssuranceHigh), registro, &relojFijo{ahora: ahora})
	canal := debeCanalTLS(t, servicio, configuracion)
	asercion := asercionInternaValida(ahora, configuracion, canal)
	asercion.Cuenta = CuentaAcceso{ID: "adm-cuenta-tecnica", SujetoVinculadoID: asercion.SujetoID, CuentaOrdinariaID: "cuenta-tecnica", Privilegiada: true}
	verificador.fijarAsercion(asercion)
	identidad, err := servicio.Resolver(context.Background(), debeCredencial(t, []byte("opaca"), canal))
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	_, auditoria, err := servicio.ProyectarPrincipal(context.Background(), identidad)
	if err != nil || auditoria.CuentaRef() == auditoria.CuentaOrdinariaRef() ||
		auditoria.CuentaRef() == auditoria.CuentaID() || auditoria.CuentaOrdinariaRef() == auditoria.CuentaOrdinariaID() {
		t.Fatalf("referencias de cuenta privilegiada no separadas: %#v %v", auditoria, err)
	}
}

func TestIdentidadSesionNoFiltraNiSeReconstruyePorSerializacion(t *testing.T) {
	ahora := time.Date(2026, 7, 18, 8, 0, 0, 0, time.UTC)
	configuracion := configuracionInternaValida()
	verificador := &verificadorFalso{}
	servicio := debeServicio(t, configuracion, verificador, evaluadorValido(dominiovec.AuthAssuranceHigh), nuevoRegistroMemoria(), &relojFijo{ahora: ahora})
	canal := debeCanalTLS(t, servicio, configuracion)
	verificador.fijarAsercion(asercionInternaValida(ahora, configuracion, canal))
	identidad, err := servicio.Resolver(context.Background(), debeCredencial(t, []byte("asercion-protegida-secreta"), canal))
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}

	jsonRedactado, err := json.Marshal(identidad)
	if err != nil {
		t.Fatalf("redactar JSON: %v", err)
	}
	textoRedactado, err := identidad.MarshalText()
	if err != nil {
		t.Fatalf("redactar texto: %v", err)
	}
	binarioRedactado, err := identidad.MarshalBinary()
	if err != nil {
		t.Fatalf("redactar binario: %v", err)
	}
	var log bytes.Buffer
	slog.New(slog.NewTextHandler(&log, nil)).Info("identidad", "valor", identidad)
	var gobRedactado bytes.Buffer
	if err := gob.NewEncoder(&gobRedactado).Encode(identidad); err != nil {
		t.Fatalf("redactar gob: %v", err)
	}
	representaciones := []string{
		fmt.Sprintf("%s %v %#v %+v", identidad, identidad, identidad, identidad),
		string(jsonRedactado), string(textoRedactado), string(binarioRedactado), log.String(), gobRedactado.String(),
	}
	prohibidos := []string{
		identidad.confirmacion.AutenticacionRef, identidad.confirmacion.AsercionRef,
		identidad.confirmacion.SesionRef, identidad.confirmacion.ControlSesionRef,
		identidad.confirmacion.CuentaRef, "persona-001", "cuenta-tecnica", "sesion-001",
	}
	for _, representacion := range representaciones {
		for _, prohibido := range prohibidos {
			if strings.Contains(representacion, prohibido) {
				t.Fatalf("identidad filtrada (%q): %q", prohibido, representacion)
			}
		}
	}

	var reconstruida IdentidadSesion
	if err := json.Unmarshal(jsonRedactado, &reconstruida); !errors.Is(err, ErrIdentidadNoSerializable) {
		t.Fatalf("JSON reconstruyo una identidad: %v", err)
	}
	if err := reconstruida.UnmarshalText(textoRedactado); !errors.Is(err, ErrIdentidadNoSerializable) {
		t.Fatalf("texto reconstruyo una identidad: %v", err)
	}
	if err := reconstruida.UnmarshalBinary(binarioRedactado); !errors.Is(err, ErrIdentidadNoSerializable) {
		t.Fatalf("binario reconstruyo una identidad: %v", err)
	}
	if err := gob.NewDecoder(bytes.NewReader(gobRedactado.Bytes())).Decode(&reconstruida); !errors.Is(err, ErrIdentidadNoSerializable) {
		t.Fatalf("gob reconstruyo una identidad: %v", err)
	}
	if _, _, err := servicio.ProyectarPrincipal(context.Background(), reconstruida); !errors.Is(err, ErrSesionNoValida) {
		t.Fatalf("identidad reconstruida aceptada: %v", err)
	}
}
