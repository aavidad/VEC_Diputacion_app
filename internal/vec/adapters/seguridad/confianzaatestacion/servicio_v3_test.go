package confianzaatestacion

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func TestServicioConfianzaAtestacionV3VerificaPerfilExacto(t *testing.T) {
	escenario := nuevoEscenarioConfianzaAtestacionV3Prueba(t)
	prueba, err := escenario.servicio.Verificar(
		context.Background(),
		escenario.solicitud,
		escenario.decision,
		escenario.motivo,
		escenario.resultado,
		escenario.atestacion,
	)
	if err != nil || prueba.Validar() != nil ||
		prueba.ValidarPara(
			escenario.solicitud,
			escenario.decision,
			escenario.motivo,
			escenario.resultado,
			escenario.atestacion,
		) != nil {
		t.Fatalf("verificacion V3 valida rechazada: prueba=%v err=%v", prueba, err)
	}
	datos, err := prueba.Datos()
	if err != nil || datos.Validar() != nil ||
		datos.Suite != SuiteAtestacionAutorizacionV3COSEEdDSA ||
		datos.AudienciaDespliegue != audienciaConfianzaAtestacionV3Prueba ||
		datos.ReferenciaContexto != escenario.resultado.RegistroContextoRef ||
		datos.HuellaContextoSHA256 != escenario.resultado.HuellaSHA256 ||
		datos.RaizVersion != escenario.raiz.version ||
		datos.SecuenciaConfiguracion != escenario.configuracion.secuencia ||
		datos.EstadoClave != EstadoClaveAtestacionAutorizacionV3Activa ||
		!datos.VerificadaEn.Equal(escenario.ahora) {
		t.Fatalf("prueba V3 incompleta: %+v, %v", datos, err)
	}
}

func TestServicioConfianzaAtestacionV3RechazaMatrizCriptografica(t *testing.T) {
	escenario := nuevoEscenarioConfianzaAtestacionV3Prueba(t)
	solicitud, _ := escenario.atestacion.Solicitud()
	mensaje, _ := solicitud.Mensaje()
	aad, _ := AADExternoAtestacionAutorizacionV3(
		audienciaConfianzaAtestacionV3Prueba,
	)
	sobreOtraClave := firmarSobreConfianzaAtestacionV3Prueba(
		t,
		ed25519.NewKeyFromSeed([]byte(strings.Repeat("x", ed25519.SeedSize))),
		[]byte(escenario.raiz.claveID),
		mensaje,
		aad,
	)
	resultado, _ := escenario.atestacion.Resultado()
	sobreAlterado, _ := resultado.Firma()
	sobreAlterado[len(sobreAlterado)-1] ^= 1

	cabeceraOtraAudiencia := domain.CabeceraAtestacionAutorizacionV3{
		FormatoVersion: domain.VersionFormatoAtestacionAutorizacionV3,
		Suite:          SuiteAtestacionAutorizacionV3COSEEdDSA,
		ClaveID:        escenario.raiz.claveID,
		Audiencia:      "vec-diputacion/pruebas/otra-audiencia",
	}
	otraAudiencia := atestacionConfianzaAtestacionV3Prueba(
		t,
		cabeceraOtraAudiencia,
		escenario.decision,
		escenario.motivo,
		escenario.resultado,
		escenario.privada,
		escenario.ahora,
	)
	cabeceraOtraSuite := domain.CabeceraAtestacionAutorizacionV3{
		FormatoVersion: domain.VersionFormatoAtestacionAutorizacionV3,
		Suite:          "VEC-AD-3-COSE-OTRA-1",
		ClaveID:        escenario.raiz.claveID,
		Audiencia:      audienciaConfianzaAtestacionV3Prueba,
	}
	otraSuite := atestacionConfianzaAtestacionV3Prueba(
		t,
		cabeceraOtraSuite,
		escenario.decision,
		escenario.motivo,
		escenario.resultado,
		escenario.privada,
		escenario.ahora,
	)
	casos := map[string]ports.AtestacionAutorizacionV3{
		"otra_clave": reemplazarSobreConfianzaAtestacionV3Prueba(
			t, escenario.atestacion, sobreOtraClave,
		),
		"firma_alterada": reemplazarSobreConfianzaAtestacionV3Prueba(
			t, escenario.atestacion, sobreAlterado,
		),
		"otra_audiencia": otraAudiencia,
		"otra_suite":     otraSuite,
		"valor_cero":     {},
	}
	for nombre, atestacion := range casos {
		t.Run(nombre, func(t *testing.T) {
			if _, err := escenario.servicio.Verificar(
				context.Background(),
				escenario.solicitud,
				escenario.decision,
				escenario.motivo,
				escenario.resultado,
				atestacion,
			); !errors.Is(err, ErrVerificacionConfianzaAtestacionV3Fallida) {
				t.Fatalf("cruce criptografico aceptado: %v", err)
			}
		})
	}
}

func TestServicioConfianzaAtestacionV3RechazaRevocacionTiempoYCruces(t *testing.T) {
	escenario := nuevoEscenarioConfianzaAtestacionV3Prueba(t)
	motivoAjeno := escenario.motivo
	motivoAjeno.EntradaClave = "motivo_22222222222222222222222222222222"
	if _, err := escenario.servicio.Verificar(
		context.Background(),
		escenario.solicitud,
		escenario.decision,
		motivoAjeno,
		escenario.resultado,
		escenario.atestacion,
	); !errors.Is(err, ErrVerificacionConfianzaAtestacionV3Fallida) {
		t.Fatalf("motivo cruzado aceptado: %v", err)
	}
	escenario.reloj.ahora = escenario.configuracion.expiraEn
	if _, err := escenario.servicio.Verificar(
		context.Background(),
		escenario.solicitud,
		escenario.decision,
		escenario.motivo,
		escenario.resultado,
		escenario.atestacion,
	); !errors.Is(err, ErrVerificacionConfianzaAtestacionV3Fallida) {
		t.Fatalf("configuracion expirada aceptada: %v", err)
	}

	revocada, err := NuevaRaizPublicaAtestacionAutorizacionV3EdDSA(
		escenario.raiz.claveID,
		escenario.raiz.version,
		escenario.raiz.clavePublica,
		escenario.raiz.audienciaDespliegue,
		EstadoClaveAtestacionAutorizacionV3Revocada,
		escenario.raiz.validaDesde,
		escenario.raiz.validaHasta,
		escenario.ahora.Add(-2*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	configuracion, err := NuevaConfiguracionConfianzaAtestacionAutorizacionV3(
		"confianza:atestacion:v3:revocada",
		escenario.configuracion.secuencia+1,
		escenario.ahora.Add(-time.Minute),
		escenario.ahora.Add(time.Hour),
		revocada,
	)
	if err != nil {
		t.Fatal(err)
	}
	servicio, err := NuevoServicioConfianzaAtestacionAutorizacionV3(
		configuracion,
		&relojConfianzaAtestacionV3Prueba{ahora: escenario.ahora},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := servicio.Verificar(
		context.Background(),
		escenario.solicitud,
		escenario.decision,
		escenario.motivo,
		escenario.resultado,
		escenario.atestacion,
	); !errors.Is(err, ErrVerificacionConfianzaAtestacionV3Fallida) {
		t.Fatalf("raiz revocada aceptada: %v", err)
	}
}

func TestServicioConfianzaAtestacionV3RotaRaicesSinAmbiguedad(
	t *testing.T,
) {
	escenario := nuevoEscenarioConfianzaAtestacionV3Prueba(t)
	privadaRotada := ed25519.NewKeyFromSeed(
		[]byte(strings.Repeat("r", ed25519.SeedSize)),
	)
	raizRotada, err := NuevaRaizPublicaAtestacionAutorizacionV3EdDSA(
		"clave:atestacion:v3:rotada:2026-08",
		2,
		privadaRotada.Public().(ed25519.PublicKey),
		audienciaConfianzaAtestacionV3Prueba,
		EstadoClaveAtestacionAutorizacionV3Activa,
		escenario.ahora.Add(-time.Minute),
		escenario.ahora.Add(time.Hour),
		time.Time{},
	)
	if err != nil {
		t.Fatal(err)
	}
	configuracion, err := NuevaConfiguracionConfianzaAtestacionAutorizacionV3(
		"confianza:atestacion:v3:rotacion:2026-08",
		escenario.configuracion.secuencia+1,
		escenario.ahora.Add(-time.Minute),
		escenario.ahora.Add(time.Hour),
		escenario.raiz,
		raizRotada,
	)
	if err != nil {
		t.Fatal(err)
	}
	servicio, err := NuevoServicioConfianzaAtestacionAutorizacionV3(
		configuracion,
		&relojConfianzaAtestacionV3Prueba{ahora: escenario.ahora},
	)
	if err != nil {
		t.Fatal(err)
	}
	atestacionRotada := atestacionConfianzaAtestacionV3Prueba(
		t,
		domain.CabeceraAtestacionAutorizacionV3{
			FormatoVersion: domain.VersionFormatoAtestacionAutorizacionV3,
			Suite:          SuiteAtestacionAutorizacionV3COSEEdDSA,
			ClaveID:        raizRotada.claveID,
			Audiencia:      audienciaConfianzaAtestacionV3Prueba,
		},
		escenario.decision,
		escenario.motivo,
		escenario.resultado,
		privadaRotada,
		escenario.ahora,
	)
	prueba, err := servicio.Verificar(
		context.Background(),
		escenario.solicitud,
		escenario.decision,
		escenario.motivo,
		escenario.resultado,
		atestacionRotada,
	)
	if err != nil {
		t.Fatalf("segunda raiz vigente rechazada: %v", err)
	}
	datos, err := prueba.Datos()
	if err != nil ||
		datos.ClaveID != raizRotada.claveID ||
		datos.RaizVersion != raizRotada.version ||
		datos.HuellaClaveSPKISHA256 != raizRotada.huellaClaveSPKISHA256 ||
		datos.SecuenciaConfiguracion != configuracion.secuencia {
		t.Fatalf("identidad de raiz rotada incompleta: %+v, %v", datos, err)
	}

	mismoKidOtraClave, err :=
		NuevaRaizPublicaAtestacionAutorizacionV3EdDSA(
			escenario.raiz.claveID,
			escenario.raiz.version+1,
			privadaRotada.Public().(ed25519.PublicKey),
			audienciaConfianzaAtestacionV3Prueba,
			EstadoClaveAtestacionAutorizacionV3Activa,
			escenario.ahora.Add(-time.Minute),
			escenario.ahora.Add(time.Hour),
			time.Time{},
		)
	if err != nil {
		t.Fatal(err)
	}
	otroKidMismaClave, err :=
		NuevaRaizPublicaAtestacionAutorizacionV3EdDSA(
			"clave:atestacion:v3:alias-no-permitido",
			1,
			escenario.raiz.clavePublica,
			audienciaConfianzaAtestacionV3Prueba,
			EstadoClaveAtestacionAutorizacionV3Activa,
			escenario.ahora.Add(-time.Minute),
			escenario.ahora.Add(time.Hour),
			time.Time{},
		)
	if err != nil {
		t.Fatal(err)
	}
	for nombre, segunda := range map[string]RaizPublicaAtestacionAutorizacionV3{
		"kid_duplicado":  mismoKidOtraClave,
		"spki_duplicado": otroKidMismaClave,
	} {
		t.Run(nombre, func(t *testing.T) {
			if _, err := NuevaConfiguracionConfianzaAtestacionAutorizacionV3(
				"confianza:atestacion:v3:ambigua:"+nombre,
				configuracion.secuencia+1,
				escenario.ahora.Add(-time.Minute),
				escenario.ahora.Add(time.Hour),
				escenario.raiz,
				segunda,
			); !errors.Is(
				err,
				ErrConfiguracionConfianzaAtestacionV3Invalida,
			) {
				t.Fatalf("configuracion ambigua aceptada: %v", err)
			}
		})
	}
	if _, err := NuevaRaizPublicaAtestacionAutorizacionV3EdDSA(
		"clave:atestacion:v3:version-nula",
		0,
		privadaRotada.Public().(ed25519.PublicKey),
		audienciaConfianzaAtestacionV3Prueba,
		EstadoClaveAtestacionAutorizacionV3Activa,
		escenario.ahora.Add(-time.Minute),
		escenario.ahora.Add(time.Hour),
		time.Time{},
	); !errors.Is(err, ErrConfiguracionConfianzaAtestacionV3Invalida) {
		t.Fatalf("version nula de raiz aceptada: %v", err)
	}
	if _, err := NuevaConfiguracionConfianzaAtestacionAutorizacionV3(
		"confianza:atestacion:v3:secuencia-nula",
		0,
		escenario.ahora.Add(-time.Minute),
		escenario.ahora.Add(time.Hour),
		escenario.raiz,
	); !errors.Is(err, ErrConfiguracionConfianzaAtestacionV3Invalida) {
		t.Fatalf("secuencia nula de configuracion aceptada: %v", err)
	}
}

func TestServicioConfianzaAtestacionV3RespetaCancelacionAntesYDespuesDelReloj(
	t *testing.T,
) {
	escenario := nuevoEscenarioConfianzaAtestacionV3Prueba(t)
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := escenario.servicio.Verificar(
		ctx,
		escenario.solicitud,
		escenario.decision,
		escenario.motivo,
		escenario.resultado,
		escenario.atestacion,
	); !errors.Is(err, context.Canceled) || escenario.reloj.invocaciones != 0 {
		t.Fatalf("cancelacion previa alcanzo reloj: %v", err)
	}

	ctx, cancelar = context.WithCancel(context.Background())
	escenario.reloj.cancelar = cancelar
	if _, err := escenario.servicio.Verificar(
		ctx,
		escenario.solicitud,
		escenario.decision,
		escenario.motivo,
		escenario.resultado,
		escenario.atestacion,
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelacion tras reloj ignorada: %v", err)
	}
}

func TestConfianzaAtestacionV3BloqueaCodecsYRedacta(t *testing.T) {
	escenario := nuevoEscenarioConfianzaAtestacionV3Prueba(t)
	prueba, err := escenario.servicio.Verificar(
		context.Background(),
		escenario.solicitud,
		escenario.decision,
		escenario.motivo,
		escenario.resultado,
		escenario.atestacion,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := json.Marshal(prueba); !errors.Is(
		err,
		ErrSerializacionConfianzaAtestacionV3Prohibida,
	) {
		t.Fatalf("JSON permitido: %v", err)
	}
	const marca = "[CONFIANZA-ATESTACION-AUTORIZACION-V3-REDACTADA]"
	for _, texto := range []string{
		fmt.Sprint(prueba), fmt.Sprintf("%+v", prueba), fmt.Sprintf("%#v", prueba),
	} {
		if !strings.Contains(texto, marca) ||
			strings.Contains(texto, escenario.raiz.claveID) {
			t.Fatalf("formato filtro datos: %q", texto)
		}
	}
	var bitacora bytes.Buffer
	slog.New(slog.NewTextHandler(&bitacora, nil)).Info("prueba", "valor", prueba)
	if !strings.Contains(bitacora.String(), marca) ||
		strings.Contains(bitacora.String(), escenario.raiz.claveID) {
		t.Fatalf("slog filtro datos: %s", bitacora.String())
	}
}

func reemplazarSobreConfianzaAtestacionV3Prueba(
	t *testing.T,
	atestacion ports.AtestacionAutorizacionV3,
	sobre []byte,
) ports.AtestacionAutorizacionV3 {
	t.Helper()
	solicitud, err := atestacion.Solicitud()
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := ports.NuevoResultadoFirmaAtestacionAutorizacionV3(
		solicitud,
		sobre,
		"evidencia:firma:confianza:atestacion:v3:reemplazada",
		time.Date(2026, 7, 23, 9, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}
	nueva, err := ports.NuevaAtestacionAutorizacionV3(solicitud, resultado)
	if err != nil {
		t.Fatal(err)
	}
	return nueva
}
