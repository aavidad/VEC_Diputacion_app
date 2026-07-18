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
		{"eco de politica alterado", func(c *ConfirmacionAltaSesion, _ AltaSesionAtomica) {
			c.AltaConfirmada.PoliticaGarantiaRef = "pga_otra23456789abcdefghijkl"
		}},
		{"eco de espacio de identidad alterado", func(c *ConfirmacionAltaSesion, _ AltaSesionAtomica) {
			c.AltaConfirmada.EspacioIdentidad = "https://otro-idp.example.test"
		}},
		{"eco de tiempo alterado", func(c *ConfirmacionAltaSesion, _ AltaSesionAtomica) {
			c.AltaConfirmada.AsercionExpiraEn = c.AltaConfirmada.AsercionExpiraEn.Add(time.Second)
		}},
		{"revision cero", func(c *ConfirmacionAltaSesion, _ AltaSesionAtomica) { c.ControlSesionRevision = 0 }},
		{"revision de alta distinta de uno", func(c *ConfirmacionAltaSesion, _ AltaSesionAtomica) {
			c.ControlSesionRevision = 2
		}},
		{"control revocado", func(c *ConfirmacionAltaSesion, _ AltaSesionAtomica) {
			c.ControlSesionEstado = EstadoControlSesionRevocada
		}},
		{"huella de control no canonica", func(c *ConfirmacionAltaSesion, _ AltaSesionAtomica) {
			c.ControlSesionHuellaSHA256 = strings.Repeat("A", 64)
		}},
		{"revalidacion sin microsegundos", func(c *ConfirmacionAltaSesion, _ AltaSesionAtomica) {
			c.SesionRevalidadaEn = c.SesionRevalidadaEn.Add(time.Nanosecond)
		}},
		{"vigencia amplia la asercion", func(c *ConfirmacionAltaSesion, alta AltaSesionAtomica) {
			c.SesionValidaHasta = alta.AsercionExpiraEn.Add(time.Microsecond)
		}},
		{"vigencia no posterior", func(c *ConfirmacionAltaSesion, _ AltaSesionAtomica) {
			c.SesionValidaHasta = c.SesionRevalidadaEn
		}},
		{"revalidacion futura", func(c *ConfirmacionAltaSesion, alta AltaSesionAtomica) {
			c.SesionRevalidadaEn = alta.SesionEmitidaEn.Add(90 * time.Second)
			c.SesionValidaHasta = alta.AsercionExpiraEn
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

	cuenta, auditoria, err := servicio.ProyectarCuentaAutenticada(context.Background(), identidad)
	if err != nil {
		t.Fatalf("proyectar: %v", err)
	}
	confirmacion := identidad.confirmacion
	if cuenta.CuentaRef != confirmacion.CuentaRef || cuenta.Metodo != confirmacion.AltaConfirmada.MetodoObservado ||
		cuenta.Garantia != confirmacion.AltaConfirmada.GarantiaObservada ||
		auditoria.AutenticacionRef() != confirmacion.AutenticacionRef ||
		auditoria.AsercionRef() != confirmacion.AsercionRef || auditoria.SesionRef() != confirmacion.SesionRef ||
		auditoria.ControlSesionRef() != confirmacion.ControlSesionRef || auditoria.CuentaRef() != confirmacion.CuentaRef ||
		auditoria.CuentaOrdinariaRef() != confirmacion.CuentaOrdinariaRef ||
		auditoria.CuentaRef() != auditoria.CuentaOrdinariaRef() || auditoria.CuentaRef() != cuenta.CuentaRef {
		t.Fatalf("proyeccion durable incompleta: cuenta=%#v auditoria=%#v", cuenta, auditoria)
	}

	registro.mu.Lock()
	consulta := registro.ultimaConsulta
	registro.mu.Unlock()
	if consulta.AutenticacionRef != confirmacion.AutenticacionRef || consulta.AsercionRef != confirmacion.AsercionRef ||
		consulta.SesionRef != confirmacion.SesionRef || consulta.ControlSesionRef != confirmacion.ControlSesionRef ||
		consulta.CuentaRef != confirmacion.CuentaRef || consulta.CuentaOrdinariaRef != confirmacion.CuentaOrdinariaRef ||
		consulta.AutenticacionHuellaSHA256 != confirmacion.AltaConfirmada.AutenticacionHuellaSHA256 ||
		consulta.CuentaPrivilegiada != confirmacion.AltaConfirmada.CuentaPrivilegiada ||
		consulta.Superficie != confirmacion.AltaConfirmada.Superficie ||
		consulta.MetodoObservado != confirmacion.AltaConfirmada.MetodoObservado ||
		consulta.GarantiaObservada != confirmacion.AltaConfirmada.GarantiaObservada ||
		consulta.PoliticaGarantiaRef != confirmacion.AltaConfirmada.PoliticaGarantiaRef ||
		consulta.PoliticaGarantiaHuellaSHA256 != confirmacion.AltaConfirmada.PoliticaGarantiaHuellaSHA256 ||
		!consulta.AutenticacionVerificadaEn.Equal(confirmacion.AltaConfirmada.AutenticacionVerificadaEn) ||
		!consulta.SesionEmitidaEn.Equal(confirmacion.AltaConfirmada.SesionEmitidaEn) ||
		consulta.ControlSesionRevision != confirmacion.ControlSesionRevision ||
		consulta.ControlSesionEstado != confirmacion.ControlSesionEstado ||
		consulta.ControlSesionHuellaSHA256 != confirmacion.ControlSesionHuellaSHA256 ||
		!consulta.SesionRevalidadaEn.Equal(confirmacion.SesionRevalidadaEn) ||
		!consulta.SesionValidaHasta.Equal(confirmacion.SesionValidaHasta) {
		t.Fatalf("consulta de revalidacion no reproduce la confirmacion: %#v", consulta)
	}

	identidadConReciboAlterado := identidad
	identidadConReciboAlterado.confirmacion.ControlSesionHuellaSHA256 = strings.Repeat("b", 64)
	if _, _, err := servicio.ProyectarCuentaAutenticada(context.Background(), identidadConReciboAlterado); !errors.Is(err, ErrSesionNoValida) {
		t.Fatalf("un recibo durable alterado fue aceptado: %v", err)
	}

	registro.mu.Lock()
	alterada := registro.sesiones[confirmacion.SesionRef]
	alterada.ControlSesionRef = "cse_alterado567890abcdefghijkl"
	registro.sesiones[confirmacion.SesionRef] = alterada
	registro.mu.Unlock()
	if _, _, err := servicio.ProyectarCuentaAutenticada(context.Background(), identidad); !errors.Is(err, ErrSesionNoValida) {
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
	cuenta, auditoria, err := servicio.ProyectarCuentaAutenticada(context.Background(), identidad)
	if err != nil || auditoria.CuentaRef() == auditoria.CuentaOrdinariaRef() ||
		cuenta.CuentaRef != identidad.confirmacion.CuentaRef || cuenta.CuentaRef != auditoria.CuentaRef() ||
		cuenta.CuentaRef == auditoria.CuentaOrdinariaRef() {
		t.Fatalf("referencias de cuenta privilegiada no separadas: cuenta=%#v auditoria=%#v %v", cuenta, auditoria, err)
	}
	resultado := fmt.Sprintf("%#v %#v %s %s", cuenta, auditoria, debeJSON(t, cuenta), debeJSON(t, auditoria))
	for _, identificadorIdP := range []string{asercion.Cuenta.ID, asercion.Cuenta.CuentaOrdinariaID, asercion.SujetoID} {
		if strings.Contains(resultado, identificadorIdP) {
			t.Fatalf("la cuenta privilegiada filtro el identificador del IdP %q: %q", identificadorIdP, resultado)
		}
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
	if _, _, err := servicio.ProyectarCuentaAutenticada(context.Background(), reconstruida); !errors.Is(err, ErrSesionNoValida) {
		t.Fatalf("identidad reconstruida aceptada: %v", err)
	}
}
