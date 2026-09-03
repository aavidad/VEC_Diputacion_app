package bootstrap

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestSelladorHMACCoberturaGeneraParejaRotadaExacta(t *testing.T) {
	derivador := nuevoDerivadorIdempotenciaPrueba(t, 2, 1)
	sellador, err := nuevoSelladorHMACCoberturaDesarrollo(derivador)
	if err != nil {
		t.Fatal(err)
	}
	preimagenes, _ := preimagenesCoberturaDesarrolloPrueba(t, "a")
	ambito, _ := preimagenes.BytesAmbito()
	semantica, _ := preimagenes.BytesSemantica()

	sellos, err := sellador.SellarOperacionDecisionCobertura(
		context.Background(),
		preimagenes,
	)
	if err != nil || sellos.Validar() != nil {
		t.Fatalf("sellar cobertura: %v", err)
	}
	datosAmbito, _ := sellos.AmbitosIdempotenciaHMAC.Datos()
	datosSemantica, _ := sellos.HuellasSemanticasHMAC.Datos()
	if datosAmbito.Activo.Generacion != 2 || datosSemantica.Activo.Generacion != 2 ||
		len(datosAmbito.Retenidos) != 1 || len(datosSemantica.Retenidos) != 1 ||
		datosAmbito.Retenidos[0].Generacion != 1 ||
		datosSemantica.Retenidos[0].Generacion != 1 {
		t.Fatalf("rotacion desalineada: %+v %+v", datosAmbito, datosSemantica)
	}
	esperadoAmbito := selloHMACCoberturaDesarrolloPrueba(
		derivador.generaciones[0].localizador.material[:],
		ambito,
		dominioHMACAmbitoCoberturaDesarrollo,
		2,
	)
	esperadoSemantica := selloHMACCoberturaDesarrolloPrueba(
		derivador.generaciones[0].huellaSolicitud.material[:],
		semantica,
		dominioHMACSemanticaCoberturaDesarrollo,
		2,
	)
	if datosAmbito.Activo.Valor != esperadoAmbito ||
		datosSemantica.Activo.Valor != esperadoSemantica {
		t.Fatalf(
			"sellos activos inesperados:\n%s\n%s",
			datosAmbito.Activo.Valor,
			datosSemantica.Activo.Valor,
		)
	}
}

func TestSelladorHMACRecuperacionReproduceAmbitoDecision(t *testing.T) {
	derivador := nuevoDerivadorIdempotenciaPrueba(t, 3, 2)
	sellador, err := nuevoSelladorHMACCoberturaDesarrollo(derivador)
	if err != nil {
		t.Fatal(err)
	}
	preimagenes, recuperacion := preimagenesCoberturaDesarrolloPrueba(t, "a")
	decision, err := sellador.SellarOperacionDecisionCobertura(
		context.Background(),
		preimagenes,
	)
	if err != nil {
		t.Fatal(err)
	}
	ambitosRecuperacion, err := sellador.SellarAmbitoOperacionDecisionCobertura(
		context.Background(),
		recuperacion,
	)
	if err != nil {
		t.Fatal(err)
	}
	datosDecision, _ := decision.AmbitosIdempotenciaHMAC.Datos()
	datosRecuperacion, _ := ambitosRecuperacion.Datos()
	if !reflect.DeepEqual(datosDecision, datosRecuperacion) {
		t.Fatalf("recuperacion no reprodujo ambito: %+v %+v", datosDecision, datosRecuperacion)
	}

	otrasPreimagenes, _ := preimagenesCoberturaDesarrolloPrueba(t, "b")
	otraDecision, err := sellador.SellarOperacionDecisionCobertura(
		context.Background(),
		otrasPreimagenes,
	)
	if err != nil {
		t.Fatal(err)
	}
	otroAmbito, _ := otraDecision.AmbitosIdempotenciaHMAC.Datos()
	semantica, _ := decision.HuellasSemanticasHMAC.Datos()
	otraSemantica, _ := otraDecision.HuellasSemanticasHMAC.Datos()
	if !reflect.DeepEqual(datosDecision, otroAmbito) ||
		semantica.Activo.Valor == otraSemantica.Activo.Valor {
		t.Fatal("cambiar semantica altero el ambito o mantuvo su huella")
	}
}

func TestSelladorHMACCoberturaEsEstableYFallaCerrado(t *testing.T) {
	primero := nuevoDerivadorIdempotenciaPrueba(t, 2, 1)
	segundo := nuevoDerivadorIdempotenciaPrueba(t, 2, 1)
	preimagenes, recuperacion := preimagenesCoberturaDesarrolloPrueba(t, "a")
	selladorPrimero, _ := nuevoSelladorHMACCoberturaDesarrollo(primero)
	selladorSegundo, _ := nuevoSelladorHMACCoberturaDesarrollo(segundo)
	sellosPrimero, err := selladorPrimero.SellarOperacionDecisionCobertura(
		context.Background(),
		preimagenes,
	)
	if err != nil {
		t.Fatal(err)
	}
	sellosSegundo, err := selladorSegundo.SellarOperacionDecisionCobertura(
		context.Background(),
		preimagenes,
	)
	if err != nil || !sellosCoberturaDesarrolloIguales(sellosPrimero, sellosSegundo) {
		t.Fatalf("material reconstruido no fue estable: %v", err)
	}

	ctxCancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	for nombre, ctx := range map[string]context.Context{
		"nulo":        nil,
		"nulo tipado": (*contextoNuloPrueba)(nil),
		"cancelado":   ctxCancelado,
	} {
		t.Run(nombre, func(t *testing.T) {
			if _, err := selladorPrimero.SellarOperacionDecisionCobertura(
				ctx,
				preimagenes,
			); !errors.Is(err, cobertura.ErrOperacionDecisionCoberturaIdempotenteInvalida) {
				t.Fatalf("decision aceptada: %v", err)
			}
			if _, err := selladorPrimero.SellarAmbitoOperacionDecisionCobertura(
				ctx,
				recuperacion,
			); !errors.Is(err, cobertura.ErrOperacionDecisionCoberturaIdempotenteInvalida) {
				t.Fatalf("recuperacion aceptada: %v", err)
			}
		})
	}
	var selladorNulo *selladorHMACCoberturaDesarrollo
	if _, err := selladorNulo.SellarOperacionDecisionCobertura(
		context.Background(),
		preimagenes,
	); !errors.Is(err, cobertura.ErrOperacionDecisionCoberturaIdempotenteInvalida) {
		t.Fatalf("receptor nulo aceptado: %v", err)
	}
	if _, err := nuevoSelladorHMACCoberturaDesarrollo(nil); !errors.Is(
		err,
		cobertura.ErrOperacionDecisionCoberturaIdempotenteInvalida,
	) {
		t.Fatalf("derivador nulo aceptado: %v", err)
	}
	if _, err := selladorPrimero.SellarOperacionDecisionCobertura(
		context.Background(),
		cobertura.PreimagenesOperacionDecisionCobertura{},
	); !errors.Is(err, cobertura.ErrOperacionDecisionCoberturaIdempotenteInvalida) {
		t.Fatalf("preimagenes vacias aceptadas: %v", err)
	}
}

func preimagenesCoberturaDesarrolloPrueba(
	t *testing.T,
	semantica string,
) (
	cobertura.PreimagenesOperacionDecisionCobertura,
	cobertura.PreimagenAmbitoRecuperacionOperacionDecisionCobertura,
) {
	t.Helper()
	base := time.Date(2026, 9, 4, 8, 0, 0, 0, time.UTC)
	principal := dominiovec.Principal{
		ID:            "certificado_rrhh_cobertura_desarrollo",
		Roles:         []string{"tecnico_rrhh"},
		AuthMethod:    dominiovec.AuthMethodCertificate,
		AuthAssurance: dominiovec.AuthAssuranceHigh,
		Attributes: map[string]string{
			"autoridad":          AutoridadNoAutoritativa,
			"perfil_ejecucion":   config.ExecutionProfileDevelopment,
			"certificate_sha256": strings.Repeat("c", 64),
		},
	}
	contexto, err := nuevoContextoAltaContratacionTemporalDesarrollo(principal, base)
	if err != nil {
		t.Fatal(err)
	}
	vinculo, err := contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	solicitudContexto := ports.SolicitudResolverContextoAutorizacionAltaV3{
		AutenticacionRef: vinculo.AutenticacionRef,
		SesionRef:        vinculo.SesionRef,
		PerfilRef:        vinculo.PerfilActivoRef,
	}
	huellaSemantica := strings.Repeat(semantica, 64)
	identidad, err := cobertura.NuevaIdentidadOperacionDecisionCobertura(
		"018f3b2a-7c4d-4e5f-8a9b-0c1d2e3f4a5b",
		domain.DecisionCoberturaInicial,
		organizacionAltaContratacionTemporalDesarrollo,
		"expediente_temporal_desarrollo_0001",
		1,
		contexto,
		solicitudContexto,
		base,
		domain.AccionDecidirCoberturaGobernada,
		"bolsa_vigente",
		domain.IdentidadSemanticaPropuestaDecisionCobertura{
			Referencia:   "propuesta-cobertura-semantica:sha256:" + huellaSemantica,
			HuellaSHA256: huellaSemantica,
			Canon:        domain.CanonHuellaSemanticaPropuestaDecisionCoberturaV1(),
		},
		domain.MotivoGobernadoDecisionCobertura{},
		"",
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
	preimagenes, err := cobertura.NuevasPreimagenesOperacionDecisionCobertura(identidad)
	if err != nil {
		t.Fatal(err)
	}
	contextoRecuperacion, err := ports.NuevoContextoRecuperacionResultadoCobertura(
		solicitudContexto,
		contexto,
		organizacionAltaContratacionTemporalDesarrollo,
		base,
	)
	if err != nil {
		t.Fatal(err)
	}
	recuperacion, err := cobertura.NuevaPreimagenAmbitoRecuperacionOperacionDecisionCobertura(
		"018f3b2a-7c4d-4e5f-8a9b-0c1d2e3f4a5b",
		"expediente_temporal_desarrollo_0001",
		contextoRecuperacion,
		base,
	)
	if err != nil {
		t.Fatal(err)
	}
	return preimagenes, recuperacion
}

func selloHMACCoberturaDesarrolloPrueba(
	clave []byte,
	preimagen []byte,
	dominio string,
	generacion uint32,
) string {
	mac := hmac.New(sha256.New, clave)
	_, _ = mac.Write(preimagen)
	return fmt.Sprintf(
		"hmac-sha256:%s/v%d:%s",
		dominio,
		generacion,
		hex.EncodeToString(mac.Sum(nil)),
	)
}

func sellosCoberturaDesarrolloIguales(
	primero cobertura.SellosOperacionDecisionCobertura,
	segundo cobertura.SellosOperacionDecisionCobertura,
) bool {
	ambitosPrimero, errAP := primero.AmbitosIdempotenciaHMAC.Datos()
	ambitosSegundo, errAS := segundo.AmbitosIdempotenciaHMAC.Datos()
	semanticasPrimero, errSP := primero.HuellasSemanticasHMAC.Datos()
	semanticasSegundo, errSS := segundo.HuellasSemanticasHMAC.Datos()
	return errors.Join(errAP, errAS, errSP, errSS) == nil &&
		reflect.DeepEqual(ambitosPrimero, ambitosSegundo) &&
		reflect.DeepEqual(semanticasPrimero, semanticasSegundo)
}
