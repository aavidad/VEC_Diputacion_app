package ports

import (
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const claveRespuestaFuenteAnalisisPrueba = "clave-tcb-respuesta-fuente-analisis"

const (
	organizacionAutoridadPrueba = "organizacion_diputacion_granada"
	audienciaAutoridadPrueba    = "audiencia_fuentes_analisis_interna"
	raizAutoridadPruebaID       = "raiz_institucional_fuentes_012345"
)

func claveEd25519Prueba(etiqueta string) ed25519.PrivateKey {
	semilla := sha256.Sum256([]byte("VEC-CT-PRUEBA:" + etiqueta))
	return ed25519.NewKeyFromSeed(semilla[:])
}

func datosCredencialAutoridadPrueba(
	rol RolAutoridadFuenteAnalisis,
	autoridadRef string,
	backendRef string,
) DatosCredencialAutoridadFuenteAnalisis {
	clave := claveEd25519Prueba(string(rol) + ":" + autoridadRef)
	return DatosCredencialAutoridadFuenteAnalisis{
		RaizClaveID: raizAutoridadPruebaID, AutoridadRef: autoridadRef,
		BackendRef: backendRef, OrganizacionRef: organizacionAutoridadPrueba,
		Audiencia: audienciaAutoridadPrueba, Rol: rol,
		Serie: 1, Generacion: 1,
		ClavePruebaEd25519: clave.Public().(ed25519.PublicKey),
		EmitidaEn:          time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		ValidaHasta:        time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func presentacionAutoridadPrueba(
	rol RolAutoridadFuenteAnalisis,
	autoridadRef string,
	backendRef string,
	desafio DesafioAutoridadFuenteAnalisis,
) (PresentacionAutoridadFuenteAnalisis, error) {
	datos := datosCredencialAutoridadPrueba(rol, autoridadRef, backendRef)
	documento, err := canonCredencialAutoridadFuenteAnalisis(datos)
	if err != nil {
		return PresentacionAutoridadFuenteAnalisis{}, err
	}
	firmaRaiz := ed25519.Sign(
		claveEd25519Prueba("raiz-institucional"),
		documento,
	)
	credencial, err := NuevaCredencialAutoridadFuenteAnalisis(datos, firmaRaiz)
	if err != nil {
		return PresentacionAutoridadFuenteAnalisis{}, err
	}
	material, err := desafio.Bytes()
	if err != nil {
		return PresentacionAutoridadFuenteAnalisis{}, err
	}
	prueba := ed25519.Sign(
		claveEd25519Prueba(string(rol)+":"+autoridadRef),
		material,
	)
	return NuevaPresentacionAutoridadFuenteAnalisis(credencial, prueba)
}

func confianzaAutoridadesPrueba(t *testing.T) ConfianzaAutoridadesFuenteAnalisis {
	t.Helper()
	raizPrivada := claveEd25519Prueba("raiz-institucional")
	confianza, err := NuevaConfianzaAutoridadesFuenteAnalisis(
		organizacionAutoridadPrueba,
		audienciaAutoridadPrueba,
		[]RaizConfianzaAutoridadFuenteAnalisis{{
			ClaveID:             raizAutoridadPruebaID,
			ClavePublicaEd25519: raizPrivada.Public().(ed25519.PublicKey),
			Estado:              RaizAutoridadActiva,
			ValidaDesde:         time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			ValidaHasta:         time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC),
			UltimaEmisionPermitida: time.Date(
				2030, 1, 1, 0, 0, 0, 0, time.UTC,
			),
		}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	return confianza
}

func preparacionValidarRCPrueba() PreparacionSolicitudValidarRC {
	fechaRC := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	return PreparacionSolicitudValidarRC{
		OrganizacionRef:   "organizacion_diputacion_granada",
		ExpedienteRef:     "expediente_temporal_0123456789",
		VersionExpediente: 2,
		Entrada: domain.VinculoEntradaRC{
			Referencia:   "entrada_rc_0123456789",
			HuellaSHA256: strings.Repeat("a", 64),
		},
		Declaracion: domain.DeclaracionRC{
			Existe: true, Numero: "rc_2026_0123456789", Fecha: fechaRC,
			Importe:      domain.Importe{Centimos: 3_245_000, Moneda: "EUR"},
			DocumentoRef: "documento_rc_0123456789",
		},
	}
}

func preparacionCalcularCostePrueba() PreparacionSolicitudCalcularCoste {
	return PreparacionSolicitudCalcularCoste{
		OrganizacionRef:   "organizacion_diputacion_granada",
		ExpedienteRef:     "expediente_temporal_0123456789",
		VersionExpediente: 2,
		CategoriaRef:      "categoria_trabajo_social",
		GrupoSubgrupo:     "A2",
		ModalidadClave:    "sustitucion",
		CausaClave:        "incapacidad_temporal",
		Periodo: domain.PeriodoPrevisto{
			Inicio: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
			Fin:    time.Date(2027, 3, 31, 0, 0, 0, 0, time.UTC),
		},
		Jornada: domain.JornadaCompletaDiezmilesimas,
	}
}

func validacionRCPrueba(
	t *testing.T,
	solicitud SolicitudValidarRC,
	validadaEn time.Time,
) domain.ValidacionRC {
	t.Helper()
	datos, err := solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	fechaRC := datos.Declaracion.Fecha
	importe := datos.Declaracion.Importe
	return domain.ValidacionRC{
		Resultado:           domain.RCValidada,
		EntradaRef:          datos.Entrada.Referencia,
		HuellaEntradaSHA256: datos.Entrada.HuellaSHA256,
		FuenteRef:           "fuente_presupuesto_0123456789",
		ReciboRef:           "recibo_presupuesto_0123456789",
		ValidadaEn:          validadaEn,
		FechaRC:             &fechaRC,
		Numero:              datos.Declaracion.Numero,
		Importe:             &importe,
		DocumentoRef:        datos.Declaracion.DocumentoRef,
	}
}

func validacionRCNegativaPrueba(
	t *testing.T,
	solicitud SolicitudValidarRC,
	validadaEn time.Time,
) domain.ValidacionRC {
	t.Helper()
	datos, err := solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	return domain.ValidacionRC{
		Resultado:           domain.RCNoRequerida,
		EntradaRef:          datos.Entrada.Referencia,
		HuellaEntradaSHA256: datos.Entrada.HuellaSHA256,
		FuenteRef:           "fuente_presupuesto_0123456789",
		ReciboRef:           "recibo_presupuesto_0123456789",
		ValidadaEn:          validadaEn,
	}
}

func motivoFuenteAnalisisPrueba(t *testing.T) MotivoFuenteAnalisis {
	t.Helper()
	motivo, err := NuevoMotivoFuenteAnalisis(
		"catalogo_motivos_rc_0123456789",
		7,
		strings.Repeat("b", 64),
		"rc_no_requerida",
		"contratacion_temporal.rc.no_requerida",
		[]ParametroMotivoFuenteAnalisis{
			{Clave: "causa_configurada", Valor: "no_consta_rc"},
			{Clave: "resultado_configurado", Valor: "no_requerida"},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return motivo
}

func metadatosRespuestaPrueba(
	autoridadRef string,
	reciboRef string,
	inicio time.Time,
) MetadatosAtestacionRespuestaFuenteAnalisis {
	return MetadatosAtestacionRespuestaFuenteAnalisis{
		AutoridadRef: autoridadRef,
		Generacion:   7,
		ReciboRef:    reciboRef,
		EmitidaEn:    inicio.Add(2 * time.Second),
		ValidaHasta:  inicio.Add(5 * time.Second),
	}
}

func atestacionRespuestaPrueba(
	t *testing.T,
	preimagen PreimagenRespuestaFuenteAnalisis,
	metadatos MetadatosAtestacionRespuestaFuenteAnalisis,
) AtestacionRespuestaFuenteAnalisis {
	t.Helper()
	contenido, err := preimagen.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	mac := hmac.New(sha256.New, []byte(claveRespuestaFuenteAnalisisPrueba))
	_, _ = mac.Write(contenido)
	sello := "hmac-sha256:" + dominioSelloRespuestaFuenteAnalisis +
		"7:" + hex.EncodeToString(mac.Sum(nil))
	atestacion, err := NuevaAtestacionRespuestaFuenteAnalisis(metadatos, sello)
	if err != nil {
		t.Fatal(err)
	}
	return atestacion
}

func resultadoRCFirmadoPrueba(
	t *testing.T,
	solicitud SolicitudValidarRC,
	validacion domain.ValidacionRC,
	motivo MotivoFuenteAnalisis,
	metadatos MetadatosAtestacionRespuestaFuenteAnalisis,
) ResultadoValidacionRC {
	t.Helper()
	preimagen, err := NuevaPreimagenRespuestaValidacionRC(
		solicitud,
		validacion,
		motivo,
		metadatos,
	)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := NuevoResultadoValidacionRC(
		solicitud,
		validacion,
		motivo,
		atestacionRespuestaPrueba(t, preimagen, metadatos),
	)
	if err != nil {
		t.Fatal(err)
	}
	return resultado
}

func resultadoCosteFirmadoPrueba(
	t *testing.T,
	solicitud SolicitudCalcularCoste,
	metadatos MetadatosAtestacionRespuestaFuenteAnalisis,
) ResultadoCalculoCoste {
	t.Helper()
	importe := domain.Importe{Centimos: 3_148_025, Moneda: "EUR"}
	calculadoEn := metadatos.EmitidaEn.Add(-time.Second)
	preimagen, err := NuevaPreimagenRespuestaCalculoCoste(
		solicitud,
		metadatos.AutoridadRef,
		"recibo_coste_0123456789",
		importe,
		calculadoEn,
		metadatos,
	)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := NuevoResultadoCalculoCoste(
		solicitud,
		metadatos.AutoridadRef,
		"recibo_coste_0123456789",
		importe,
		calculadoEn,
		atestacionRespuestaPrueba(t, preimagen, metadatos),
	)
	if err != nil {
		t.Fatal(err)
	}
	return resultado
}

func verificadorRespuestaHMACPrueba(
	verificadaEn time.Time,
) verificadorRespuestaDoble {
	return func(
		_ context.Context,
		solicitud SolicitudVerificarRespuestaFuenteAnalisis,
	) (ConfirmacionRespuestaFuenteAnalisis, error) {
		preimagen, atestacion, err := solicitud.Material()
		if err != nil {
			return ConfirmacionRespuestaFuenteAnalisis{}, err
		}
		contenido, _ := preimagen.Bytes()
		mac := hmac.New(sha256.New, []byte(claveRespuestaFuenteAnalisisPrueba))
		_, _ = mac.Write(contenido)
		esperado := "hmac-sha256:" + dominioSelloRespuestaFuenteAnalisis +
			"7:" + hex.EncodeToString(mac.Sum(nil))
		if !hmac.Equal([]byte(esperado), []byte(atestacion.SelloHMAC)) {
			return ConfirmacionRespuestaFuenteAnalisis{},
				ErrResultadoFuenteAnalisisNoConfiable
		}
		return NuevaConfirmacionRespuestaFuenteAnalisis(
			solicitud,
			"verificador_tcb_presupuestario_012345",
			verificadaEn,
		)
	}
}

func verificadorPublicacionPrueba(
	verificadaEn time.Time,
) verificadorPublicacionMotivoDoble {
	return func(
		_ context.Context,
		solicitud SolicitudVerificarPublicacionMotivoFuenteAnalisis,
	) (ConfirmacionPublicacionMotivoFuenteAnalisis, error) {
		return NuevaConfirmacionPublicacionMotivoFuenteAnalisis(
			solicitud,
			"publicador_catalogo_motivos_012345",
			"publicacion_catalogo_motivos_rc_012345",
			"recibo_verificacion_catalogo_012345",
			verificadaEn,
		)
	}
}

func verificadorPublicacionNoInvocablePrueba(
	t *testing.T,
) verificadorPublicacionMotivoDoble {
	t.Helper()
	return func(
		context.Context,
		SolicitudVerificarPublicacionMotivoFuenteAnalisis,
	) (ConfirmacionPublicacionMotivoFuenteAnalisis, error) {
		t.Fatal("el resultado positivo no debe verificar un motivo ausente")
		return ConfirmacionPublicacionMotivoFuenteAnalisis{}, nil
	}
}

func consumidorRespuestaPrueba(
	consumidaEn time.Time,
) consumidorRespuestaDoble {
	return func(
		_ context.Context,
		orden OrdenConsumoRespuestaFuenteAnalisis,
	) (ReciboConsumoRespuestaFuenteAnalisis, error) {
		return NuevoReciboConsumoRespuestaFuenteAnalisis(
			orden,
			"consumo_respuesta_fuente_0123456789",
			consumidaEn,
		)
	}
}
