package ports

import (
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func TestConfirmacionPublicacionLigaCatalogoEntradaI18NYParametros(t *testing.T) {
	inicio := instanteFuenteAnalisisPrueba()
	motivo := motivoFuenteAnalisisPrueba(t)
	base := SolicitudVerificarPublicacionMotivoFuenteAnalisis{
		Motivo:                motivo,
		HuellaRespuestaSHA256: strings.Repeat("d", 64),
		AutoridadRespuestaRef: "fuente_presupuesto_0123456789",
		GeneracionRespuesta:   7,
	}
	confirmacion, err := NuevaConfirmacionPublicacionMotivoFuenteAnalisis(
		base,
		"publicacion_catalogo_motivos_rc_012345",
		"recibo_verificacion_catalogo_012345",
		inicio,
	)
	if err != nil {
		t.Fatal(err)
	}
	vinculo, _ := motivo.Datos()
	casos := []struct {
		nombre string
		mutar  func(*VinculoMotivoFuenteAnalisis)
	}{
		{"referencia", func(v *VinculoMotivoFuenteAnalisis) {
			v.CatalogoRef = "otro_catalogo_motivos_0123456789"
		}},
		{"versión", func(v *VinculoMotivoFuenteAnalisis) { v.CatalogoVersion++ }},
		{"huella", func(v *VinculoMotivoFuenteAnalisis) {
			v.CatalogoHuella = strings.Repeat("e", 64)
		}},
		{"entrada", func(v *VinculoMotivoFuenteAnalisis) {
			v.EntradaClave = "otra_entrada_publicada"
		}},
		{"i18n", func(v *VinculoMotivoFuenteAnalisis) {
			v.ClaveMensajeI18N = "contratacion_temporal.rc.otro_mensaje"
		}},
		{"parámetro", func(v *VinculoMotivoFuenteAnalisis) {
			v.Parametros[0].Valor = domain.ClaveCatalogo("otro_valor_publicado")
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			mutado := vinculo
			mutado.Parametros = append(
				[]ParametroMotivoFuenteAnalisis(nil),
				vinculo.Parametros...,
			)
			caso.mutar(&mutado)
			solicitud := base
			solicitud.Motivo = MotivoFuenteAnalisis{datos: &mutado}
			if confirmacion.ValidarPara(solicitud, inicio) == nil {
				t.Fatal("confirmación de publicación reutilizada")
			}
		})
	}
}

func TestMotivoNoAdmiteTextoLibreDelProveedor(t *testing.T) {
	inicio := instanteFuenteAnalisisPrueba()
	solicitud := solicitudValidarRCPrueba(t, inicio)
	validacion := validacionRCNegativaPrueba(t, solicitud, inicio.Add(time.Second))
	validacion.Motivo = "El DNI 12345678Z no cumple."
	metadatos := metadatosRespuestaPrueba(
		validacion.FuenteRef,
		validacion.ReciboRef,
		inicio,
	)
	if _, err := NuevaPreimagenRespuestaValidacionRC(
		solicitud,
		validacion,
		motivoFuenteAnalisisPrueba(t),
		metadatos,
	); err == nil {
		t.Fatal("texto libre del proveedor aceptado")
	}
}
