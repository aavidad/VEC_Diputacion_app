package ports

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func TestResultadosRechazanCrucesCampoACampo(t *testing.T) {
	inicio := instanteFuenteAnalisisPrueba()
	solicitudRC := solicitudValidarRCPrueba(t, inicio)
	baseRC, err := NuevoResultadoValidacionRC(
		solicitudRC,
		validacionRCPrueba(t, solicitudRC, inicio.Add(time.Second)),
		MotivoFuenteAnalisis{},
	)
	if err != nil {
		t.Fatal(err)
	}
	casosRC := []struct {
		nombre string
		mutar  func(*DatosResultadoValidacionRC)
	}{
		{"petición", func(d *DatosResultadoValidacionRC) {
			d.PeticionRef = "pet_otra234567890abcdefghijkl"
		}},
		{"HMAC", func(d *DatosResultadoValidacionRC) {
			d.HuellaPeticionHMAC = dominioSelloPeticionAnalisis + strings.Repeat("c", 64)
		}},
		{"organización", func(d *DatosResultadoValidacionRC) {
			d.OrganizacionRef = "organizacion_otra_0123456789"
		}},
		{"expediente", func(d *DatosResultadoValidacionRC) {
			d.ExpedienteRef = "expediente_otro_0123456789"
		}},
		{"versión", func(d *DatosResultadoValidacionRC) {
			d.VersionExpediente++
		}},
		{"entrada", func(d *DatosResultadoValidacionRC) {
			d.Validacion.EntradaRef = "entrada_rc_otra_0123456789"
		}},
		{"huella entrada", func(d *DatosResultadoValidacionRC) {
			d.Validacion.HuellaEntradaSHA256 = strings.Repeat("d", 64)
		}},
	}
	for _, caso := range casosRC {
		t.Run("RC "+caso.nombre, func(t *testing.T) {
			copia := copiarResultadoRCPrueba(baseRC)
			caso.mutar(copia.datos)
			if copia.ValidarPara(solicitudRC, inicio.Add(2*time.Second)) == nil {
				t.Fatal("resultado RC cruzado aceptado")
			}
		})
	}

	solicitudCoste := solicitudCalcularCostePrueba(t, inicio)
	baseCoste, err := NuevoResultadoCalculoCoste(
		solicitudCoste,
		"tabla_retributiva_2026_v3",
		"recibo_coste_0123456789",
		domain.Importe{Centimos: 3_148_025, Moneda: "EUR"},
		inicio.Add(time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	casosCoste := []struct {
		nombre string
		mutar  func(*DatosResultadoCalculoCoste)
	}{
		{"petición", func(d *DatosResultadoCalculoCoste) {
			d.PeticionRef = "pet_otra234567890abcdefghijkl"
		}},
		{"HMAC", func(d *DatosResultadoCalculoCoste) {
			d.HuellaPeticionHMAC = dominioSelloPeticionAnalisis + strings.Repeat("c", 64)
		}},
		{"organización", func(d *DatosResultadoCalculoCoste) {
			d.OrganizacionRef = "organizacion_otra_0123456789"
		}},
		{"expediente", func(d *DatosResultadoCalculoCoste) {
			d.ExpedienteRef = "expediente_otro_0123456789"
		}},
		{"versión", func(d *DatosResultadoCalculoCoste) {
			d.VersionExpediente++
		}},
	}
	for _, caso := range casosCoste {
		t.Run("coste "+caso.nombre, func(t *testing.T) {
			copia := baseCoste.clonar()
			caso.mutar(copia.datos)
			if copia.ValidarPara(solicitudCoste, inicio.Add(2*time.Second)) == nil {
				t.Fatal("resultado de coste cruzado aceptado")
			}
		})
	}
}

func TestFuentesRechazanResultadosAnterioresFuturosYPosterioresAlFin(
	t *testing.T,
) {
	inicio := instanteFuenteAnalisisPrueba()
	solicitud := solicitudCalcularCostePrueba(t, inicio)
	for _, caso := range []struct {
		nombre       string
		resultadoEn  time.Time
		finalizadaEn time.Time
	}{
		{"anterior", inicio.Add(-time.Microsecond), inicio.Add(time.Second)},
		{"futuro", inicio.Add(3 * time.Second), inicio.Add(2 * time.Second)},
		{"posterior al fin", inicio.Add(2*time.Second + time.Microsecond), inicio.Add(2 * time.Second)},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			resultado := resultadoCosteCrudoPrueba(
				t,
				solicitud,
				caso.resultadoEn,
			)
			if resultado.ValidarPara(solicitud, caso.finalizadaEn) == nil {
				t.Fatal("instante no confiable aceptado")
			}
		})
	}

	solicitudRC := solicitudValidarRCPrueba(t, inicio)
	validacion := validacionRCPrueba(t, solicitudRC, inicio.Add(3*time.Second))
	resultadoRC := resultadoRCCrudoPrueba(
		t,
		solicitudRC,
		validacion,
		MotivoFuenteAnalisis{},
	)
	if resultadoRC.ValidarPara(solicitudRC, inicio.Add(2*time.Second)) == nil {
		t.Fatal("validación RC posterior al fin aceptada")
	}
}

func TestContextoSeCompruebaInmediatamenteTrasCadaDependencia(t *testing.T) {
	inicio := instanteFuenteAnalisisPrueba()
	t.Run("reloj de preparación", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		generadorLlamado := false
		reloj := relojFuenteAnalisisDoble(func() time.Time {
			cancelar()
			return inicio
		})
		generador := generadorPeticionAnalisisDoble(func(
			context.Context,
			TipoPeticionFuenteAnalisis,
		) (string, error) {
			generadorLlamado = true
			return "pet_0123456789abcdefghijklmn", nil
		})
		_, err := NuevaSolicitudValidarRC(
			ctx,
			generador,
			selladorHMACFuenteAnalisisPrueba(),
			reloj,
			preparacionValidarRCPrueba(),
		)
		if !errors.Is(err, context.Canceled) || generadorLlamado {
			t.Fatalf("no comprobó ctx tras reloj: %v", err)
		}
	})

	t.Run("generador", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		selladorLlamado := false
		generador := generadorPeticionAnalisisDoble(func(
			context.Context,
			TipoPeticionFuenteAnalisis,
		) (string, error) {
			cancelar()
			return "pet_0123456789abcdefghijklmn", nil
		})
		sellador := selladorPeticionAnalisisDoble(func(
			context.Context,
			PreimagenPeticionFuenteAnalisis,
		) (string, error) {
			selladorLlamado = true
			return dominioSelloPeticionAnalisis + strings.Repeat("a", 64), nil
		})
		_, err := NuevaSolicitudValidarRC(
			ctx,
			generador,
			sellador,
			relojFijoFuenteAnalisis(inicio),
			preparacionValidarRCPrueba(),
		)
		if !errors.Is(err, context.Canceled) || selladorLlamado {
			t.Fatalf("no comprobó ctx tras generador: %v", err)
		}
	})

	t.Run("fuente", func(t *testing.T) {
		solicitud := solicitudValidarRCPrueba(t, inicio)
		ctx, cancelar := context.WithCancel(context.Background())
		relojLlamado := false
		fuente := fuentePresupuestariaDoble(func(
			context.Context,
			SolicitudValidarRC,
		) (ResultadoValidacionRC, error) {
			cancelar()
			return ResultadoValidacionRC{}, nil
		})
		_, err := ValidarRCConFuente(
			ctx,
			fuente,
			relojFuenteAnalisisDoble(func() time.Time {
				relojLlamado = true
				return inicio
			}),
			solicitud,
		)
		if !errors.Is(err, context.Canceled) || relojLlamado {
			t.Fatalf("no comprobó ctx tras fuente: %v", err)
		}
	})
}

func TestFuenteRecibeTimeoutMaximoPropio(t *testing.T) {
	inicio := instanteFuenteAnalisisPrueba()
	solicitud := solicitudCalcularCostePrueba(t, inicio)
	calculador := calculadorCosteDoble(func(
		ctx context.Context,
		recibida SolicitudCalcularCoste,
	) (ResultadoCalculoCoste, error) {
		limite, existe := ctx.Deadline()
		restante := time.Until(limite)
		if !existe || restante <= 0 ||
			restante > TiempoMaximoFuenteAnalisis+time.Second {
			t.Fatalf("timeout propio ausente: %s", restante)
		}
		return NuevoResultadoCalculoCoste(
			recibida,
			"tabla_retributiva_2026_v3",
			"recibo_coste_0123456789",
			domain.Importe{Centimos: 3_148_025, Moneda: "EUR"},
			inicio.Add(time.Second),
		)
	})
	if _, err := CalcularCosteConFuente(
		context.Background(),
		calculador,
		relojFijoFuenteAnalisis(inicio.Add(2*time.Second)),
		solicitud,
	); err != nil {
		t.Fatalf("cálculo acotado: %v", err)
	}
}

func TestMotivoCatalogadoRechazaPIITextoLibreYFiltraciones(t *testing.T) {
	invalidos := [][]ParametroMotivoFuenteAnalisis{
		{{Clave: ParametroMotivoCausa, Valor: "dni_12345678z"}},
		{{Clave: "persona", Valor: "no_consta_rc"}},
		{
			{Clave: ParametroMotivoResultado, Valor: "no_requerida"},
			{Clave: ParametroMotivoCausa, Valor: "no_consta_rc"},
		},
	}
	for _, parametros := range invalidos {
		if _, err := NuevoMotivoFuenteAnalisis(
			"catalogo_motivos_rc_0123456789",
			7,
			strings.Repeat("b", 64),
			"rc_no_requerida",
			"contratacion_temporal.rc.no_requerida",
			parametros,
		); err == nil {
			t.Fatalf("parámetros libres aceptados: %#v", parametros)
		}
	}

	inicio := instanteFuenteAnalisisPrueba()
	solicitud := solicitudValidarRCPrueba(t, inicio)
	conTexto := validacionRCNegativaPrueba(t, solicitud, inicio.Add(time.Second))
	conTexto.Motivo = "El DNI 12345678Z no cumple la regla."
	if _, err := NuevoResultadoValidacionRC(
		solicitud,
		conTexto,
		motivoFuenteAnalisisPrueba(t),
	); err == nil {
		t.Fatal("texto libre del proveedor aceptado")
	}

	motivo := motivoFuenteAnalisisPrueba(t)
	const marca = "[MOTIVO-FUENTE-ANALISIS-CATALOGADO-REDACTADO]"
	for _, texto := range []string{
		fmt.Sprint(motivo),
		fmt.Sprintf("%+v", motivo),
		fmt.Sprintf("%#v", motivo),
	} {
		if texto != marca || strings.Contains(texto, "catalogo_motivos") {
			t.Fatalf("formato no redactado: %q", texto)
		}
	}
	var registro bytes.Buffer
	slog.New(slog.NewTextHandler(&registro, nil)).Info("prueba", "motivo", motivo)
	if !strings.Contains(registro.String(), marca) ||
		strings.Contains(registro.String(), "catalogo_motivos") {
		t.Fatalf("slog filtró el motivo: %s", registro.String())
	}
}

func copiarResultadoRCPrueba(
	resultado ResultadoValidacionRC,
) ResultadoValidacionRC {
	if resultado.datos == nil {
		return ResultadoValidacionRC{}
	}
	datos := *resultado.datos
	datos.Validacion = clonarValidacionRC(datos.Validacion)
	datos.Motivo = datos.Motivo.clonar()
	return ResultadoValidacionRC{datos: &datos}
}

func resultadoCosteCrudoPrueba(
	t *testing.T,
	solicitud SolicitudCalcularCoste,
	calculadoEn time.Time,
) ResultadoCalculoCoste {
	t.Helper()
	datos, err := solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	return ResultadoCalculoCoste{datos: &DatosResultadoCalculoCoste{
		PeticionRef:        datos.PeticionRef,
		HuellaPeticionHMAC: datos.HuellaPeticionHMAC,
		OrganizacionRef:    datos.OrganizacionRef,
		ExpedienteRef:      datos.ExpedienteRef,
		VersionExpediente:  datos.VersionExpediente,
		FuenteRef:          "tabla_retributiva_2026_v3",
		ReciboRef:          "recibo_coste_0123456789",
		Importe:            domain.Importe{Centimos: 3_148_025, Moneda: "EUR"},
		CalculadoEn:        calculadoEn,
	}}
}

func resultadoRCCrudoPrueba(
	t *testing.T,
	solicitud SolicitudValidarRC,
	validacion domain.ValidacionRC,
	motivo MotivoFuenteAnalisis,
) ResultadoValidacionRC {
	t.Helper()
	datos, err := solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	return ResultadoValidacionRC{datos: &DatosResultadoValidacionRC{
		PeticionRef:        datos.PeticionRef,
		HuellaPeticionHMAC: datos.HuellaPeticionHMAC,
		OrganizacionRef:    datos.OrganizacionRef,
		ExpedienteRef:      datos.ExpedienteRef,
		VersionExpediente:  datos.VersionExpediente,
		Validacion:         validacion,
		Motivo:             motivo,
	}}
}
