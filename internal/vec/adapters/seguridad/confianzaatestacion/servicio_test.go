package confianzaatestacion

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func TestServicioConfianzaAtestacionV2VerificaPerfilSinConcederEfecto(t *testing.T) {
	escenario := nuevoEscenarioConfianzaAtestacionV2Prueba(t)
	prueba, err := escenario.servicio.Verificar(
		context.Background(),
		escenario.decision,
		escenario.motivo,
		escenario.atestacion,
	)
	if err != nil || prueba.Validar() != nil ||
		prueba.ValidarPara(escenario.decision, escenario.motivo, escenario.atestacion) != nil {
		t.Fatalf("verificar perfil V2: prueba=%v err=%v", prueba, err)
	}
	datos, err := prueba.Datos()
	if err != nil || datos.Validar() != nil {
		t.Fatalf("datos de prueba invalidos: %v", err)
	}
	if datos.ReferenciaDecision != escenario.decision.DecisionRef ||
		datos.HuellaSolicitudLigadaSHA256 != escenario.decision.SolicitudHuellaSHA256 ||
		datos.HuellaMotivoCatalogoSHA256 != escenario.decision.MotivoHuellaSHA256 ||
		datos.ClaveID != escenario.raiz.claveID ||
		datos.HuellaClaveSPKISHA256 != escenario.raiz.huellaClaveSPKISHA256 ||
		datos.AlgoritmoCOSE != AlgoritmoCOSEAtestacionAutorizacionV2EdDSA ||
		datos.Suite != SuiteAtestacionAutorizacionV2COSEEdDSA ||
		datos.AudienciaDespliegue != audienciaConfianzaAtestacionV2Prueba ||
		datos.RevisionConfiguracion != escenario.configuracion.revision ||
		datos.HuellaConfiguracionSHA256 != escenario.configuracion.huellaSHA256 ||
		!datos.VerificadaEn.Equal(escenario.ahora) {
		t.Fatalf("compromisos de confianza incompletos: %+v", datos)
	}

	solicitud, _ := escenario.atestacion.Solicitud()
	mensaje, _ := solicitud.Mensaje()
	resultado, _ := escenario.atestacion.Resultado()
	sobre, _ := resultado.Firma()
	if len(sobre) >= len(mensaje) || bytes.Contains(sobre, mensaje) {
		t.Fatalf("el sobre duplico el payload: sobre=%d mensaje=%d", len(sobre), len(mensaje))
	}

	// firmada_en procede del proveedor y no esta dentro de la firma. Cambiar
	// solo ese metadato no puede convertirlo en reloj de seguridad.
	resultadoAntiguo, err := ports.NuevoResultadoFirmaAtestacionAutorizacionV2(
		solicitud,
		sobre,
		"evidencia:firma:fecha-informativa",
		escenario.ahora.Add(-365*24*time.Hour),
	)
	if err != nil {
		t.Fatal(err)
	}
	atestacionFechaAntigua, err := ports.NuevaAtestacionAutorizacionV2(solicitud, resultadoAntiguo)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := escenario.servicio.Verificar(
		context.Background(),
		escenario.decision,
		escenario.motivo,
		atestacionFechaAntigua,
	); err != nil {
		t.Fatalf("se confio indebidamente en la fecha informativa: %v", err)
	}
}

func TestServicioConfianzaAtestacionV2RechazaCrucesCriptograficos(t *testing.T) {
	escenario := nuevoEscenarioConfianzaAtestacionV2Prueba(t)
	solicitud, _ := escenario.atestacion.Solicitud()
	mensaje, _ := solicitud.Mensaje()
	aad, _ := AADExternoAtestacionAutorizacionV2(audienciaConfianzaAtestacionV2Prueba)
	claveID := []byte(escenario.raiz.claveID)

	firmar := func(
		privadaSemilla byte,
		kid []byte,
		payload []byte,
		aadFirma []byte,
		separado bool,
	) ports.AtestacionAutorizacionV2 {
		sobre := firmarSobreConfianzaAtestacionV2Prueba(
			t,
			clavePrivadaConfianzaAtestacionV2Prueba(privadaSemilla),
			kid,
			payload,
			aadFirma,
			separado,
		)
		return reemplazarSobreConfianzaAtestacionV2Prueba(t, escenario.atestacion, sobre)
	}
	tamper := func() ports.AtestacionAutorizacionV2 {
		resultado, _ := escenario.atestacion.Resultado()
		sobre, _ := resultado.Firma()
		sobre[len(sobre)-1] ^= 1
		return reemplazarSobreConfianzaAtestacionV2Prueba(t, escenario.atestacion, sobre)
	}
	casos := []struct {
		nombre     string
		atestacion ports.AtestacionAutorizacionV2
	}{
		{"otra_clave", firmar(81, claveID, mensaje, aad, true)},
		{"otro_kid", firmar(80, []byte("clave:atestacion:v2:otra"), mensaje, aad, true)},
		{"otro_payload", firmar(80, claveID, []byte("payload-ajeno"), aad, true)},
		{"otro_aad", firmar(80, claveID, mensaje, []byte("aad-ajeno"), true)},
		{"payload_incrustado", firmar(80, claveID, mensaje, aad, false)},
		{"firma_alterada", tamper()},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if _, err := escenario.servicio.Verificar(
				context.Background(),
				escenario.decision,
				escenario.motivo,
				caso.atestacion,
			); !errors.Is(err, ErrVerificacionConfianzaAtestacionV2Fallida) {
				t.Fatalf("cruce criptografico aceptado: %v", err)
			}
		})
	}
}

func TestServicioConfianzaAtestacionV2RechazaCrucesSemanticos(t *testing.T) {
	escenario := nuevoEscenarioConfianzaAtestacionV2Prueba(t)
	motivoAjeno := escenario.motivo
	motivoAjeno.EntradaClave = "motivo_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	decisionAjena := escenario.decision
	decisionAjena.DecisionRef += ":otra"

	cabeceraOtraAudiencia := domain.CabeceraAtestacionAutorizacionV2{
		FormatoVersion: domain.VersionFormatoAtestacionAutorizacionV2,
		Suite:          SuiteAtestacionAutorizacionV2COSEEdDSA,
		ClaveID:        escenario.raiz.claveID,
		Audiencia:      "vec-diputacion/otro/vec/autorizacion-v2",
	}
	atestacionOtraAudiencia := atestacionConfianzaAtestacionV2Prueba(
		t, cabeceraOtraAudiencia, escenario.decision, escenario.motivo,
		escenario.privada, []byte(escenario.raiz.claveID), nil, nil, true, escenario.ahora,
	)
	cabeceraOtraSuite := domain.CabeceraAtestacionAutorizacionV2{
		FormatoVersion: domain.VersionFormatoAtestacionAutorizacionV2,
		Suite:          "VEC-AD-2-COSE-OTRA-1",
		ClaveID:        escenario.raiz.claveID,
		Audiencia:      audienciaConfianzaAtestacionV2Prueba,
	}
	atestacionOtraSuite := atestacionConfianzaAtestacionV2Prueba(
		t, cabeceraOtraSuite, escenario.decision, escenario.motivo,
		escenario.privada, []byte(escenario.raiz.claveID), nil, nil, true, escenario.ahora,
	)

	casos := []struct {
		nombre     string
		decision   domain.DecisionAutorizacion
		motivo     domain.ReferenciaEntradaCatalogo
		atestacion ports.AtestacionAutorizacionV2
	}{
		{"decision_ajena", decisionAjena, escenario.motivo, escenario.atestacion},
		{"motivo_ajeno", escenario.decision, motivoAjeno, escenario.atestacion},
		{"audiencia_ajena", escenario.decision, escenario.motivo, atestacionOtraAudiencia},
		{"suite_ajena", escenario.decision, escenario.motivo, atestacionOtraSuite},
		{"atestacion_cero", escenario.decision, escenario.motivo, ports.AtestacionAutorizacionV2{}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if _, err := escenario.servicio.Verificar(
				context.Background(), caso.decision, caso.motivo, caso.atestacion,
			); !errors.Is(err, ErrVerificacionConfianzaAtestacionV2Fallida) {
				t.Fatalf("cruce semantico aceptado: %v", err)
			}
		})
	}
}

func TestServicioConfianzaAtestacionV2FallaConTiempoORevocacion(t *testing.T) {
	escenario := nuevoEscenarioConfianzaAtestacionV2Prueba(t)
	for nombre, instante := range map[string]time.Time{
		"antes_de_publicar":      escenario.configuracion.publicadaEn.Add(-time.Microsecond),
		"configuracion_expirada": escenario.configuracion.expiraEn,
		"decision_expirada":      escenario.decision.ValidaHasta,
	} {
		t.Run(nombre, func(t *testing.T) {
			escenario.reloj.ahora = instante
			if _, err := escenario.servicio.Verificar(
				context.Background(), escenario.decision, escenario.motivo, escenario.atestacion,
			); !errors.Is(err, ErrVerificacionConfianzaAtestacionV2Fallida) {
				t.Fatalf("tiempo invalido aceptado: %v", err)
			}
		})
	}

	revocada := nuevaRaizConfianzaAtestacionV2Prueba(
		t, escenario.raiz.claveID, escenario.raiz.clavePublica,
		escenario.raiz.audienciaDespliegue,
		EstadoClaveAtestacionAutorizacionV2Revocada,
		escenario.raiz.validaDesde, escenario.raiz.validaHasta,
		escenario.ahora.Add(-time.Microsecond),
	)
	configuracionRevocada, err := NuevaConfiguracionConfianzaAtestacionAutorizacionV2(
		"confianza:atestacion:v2:revocada",
		escenario.ahora,
		escenario.ahora.Add(time.Hour),
		revocada,
	)
	if err != nil {
		t.Fatal(err)
	}
	servicioRevocado, err := NuevoServicioConfianzaAtestacionAutorizacionV2(
		configuracionRevocada,
		&relojConfianzaAtestacionV2Prueba{ahora: escenario.ahora},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := servicioRevocado.Verificar(
		context.Background(), escenario.decision, escenario.motivo, escenario.atestacion,
	); !errors.Is(err, ErrVerificacionConfianzaAtestacionV2Fallida) {
		t.Fatalf("raiz revocada aceptada: %v", err)
	}
}

func TestServicioConfianzaAtestacionV2RespetaContextoYNulos(t *testing.T) {
	escenario := nuevoEscenarioConfianzaAtestacionV2Prueba(t)
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := escenario.servicio.Verificar(
		ctx, escenario.decision, escenario.motivo, escenario.atestacion,
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelacion no conservada: %v", err)
	}
	if _, err := escenario.servicio.Verificar(
		nil, escenario.decision, escenario.motivo, escenario.atestacion,
	); !errors.Is(err, ErrVerificacionConfianzaAtestacionV2Fallida) {
		t.Fatalf("contexto nulo aceptado: %v", err)
	}
	var servicio *ServicioConfianzaAtestacionAutorizacionV2
	if _, err := servicio.Verificar(
		context.Background(), escenario.decision, escenario.motivo, escenario.atestacion,
	); !errors.Is(err, ErrVerificacionConfianzaAtestacionV2Fallida) {
		t.Fatalf("servicio nulo aceptado: %v", err)
	}
	var reloj *relojConfianzaAtestacionV2Prueba
	creado, err := NuevoServicioConfianzaAtestacionAutorizacionV2(escenario.configuracion, reloj)
	if creado != nil || !errors.Is(err, ErrConfiguracionConfianzaAtestacionV2Invalida) {
		t.Fatalf("reloj nulo tipado aceptado: servicio=%v err=%v", creado, err)
	}
}

func TestServicioConfianzaAtestacionV2EsSeguroEnLecturasConcurrentes(t *testing.T) {
	escenario := nuevoEscenarioConfianzaAtestacionV2Prueba(t)
	const lectores = 24
	var grupo sync.WaitGroup
	errores := make(chan error, lectores)
	for indice := 0; indice < lectores; indice++ {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			prueba, err := escenario.servicio.Verificar(
				context.Background(), escenario.decision, escenario.motivo, escenario.atestacion,
			)
			if err == nil {
				err = prueba.ValidarPara(escenario.decision, escenario.motivo, escenario.atestacion)
			}
			if err != nil {
				errores <- err
			}
		}()
	}
	grupo.Wait()
	close(errores)
	for err := range errores {
		t.Fatalf("lectura concurrente: %v", err)
	}
}

func TestPruebaConfianzaAtestacionV2RedactaBloqueaYClonaDatos(t *testing.T) {
	escenario := nuevoEscenarioConfianzaAtestacionV2Prueba(t)
	prueba, err := escenario.servicio.Verificar(
		context.Background(), escenario.decision, escenario.motivo, escenario.atestacion,
	)
	if err != nil {
		t.Fatal(err)
	}
	datos, err := prueba.Datos()
	if err != nil {
		t.Fatal(err)
	}
	datos.ReferenciaDecision = "decision:alterada"
	segunda, _ := prueba.Datos()
	if segunda.ReferenciaDecision != escenario.decision.DecisionRef {
		t.Fatal("Datos compartio estado con la prueba")
	}

	for _, valor := range []any{escenario.servicio, prueba, segunda} {
		if _, err := json.Marshal(valor); !errors.Is(err, ErrSerializacionConfianzaAtestacionV2Prohibida) {
			t.Fatalf("JSON no bloqueado para %T: %v", valor, err)
		}
		if _, err := xml.Marshal(valor); !errors.Is(err, ErrSerializacionConfianzaAtestacionV2Prohibida) {
			t.Fatalf("XML no bloqueado para %T: %v", valor, err)
		}
		texto := fmt.Sprintf("%v|%+v|%#v", valor, valor, valor)
		if strings.Contains(texto, escenario.decision.DecisionRef) ||
			strings.Contains(texto, escenario.raiz.claveID) ||
			!strings.Contains(texto, "REDACTADA") {
			t.Fatalf("formato no redactado para %T: %s", valor, texto)
		}
		registro := slog.AnyValue(valor).Resolve().String()
		if strings.Contains(registro, escenario.decision.DecisionRef) ||
			!strings.Contains(registro, "REDACTADA") {
			t.Fatalf("slog no redactado para %T: %s", valor, registro)
		}
	}
	if (PruebaConfianzaAtestacionAutorizacionV2{}).Validar() == nil ||
		(DatosPruebaConfianzaAtestacionAutorizacionV2{}).Validar() == nil {
		t.Fatal("un valor cero de prueba fue aceptado")
	}
}
