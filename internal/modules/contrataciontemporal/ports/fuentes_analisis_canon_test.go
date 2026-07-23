package ports

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func TestCanonPeticionRCLigaCadaCoordenada(t *testing.T) {
	inicio := instanteFuenteAnalisisPrueba()
	base := datosCanonicosRCPrueba(
		preparacionValidarRCPrueba(),
		"pet_0123456789abcdefghijklmn",
		inicio,
	)
	canonBase, err := canonPeticionValidacionRC(base)
	if err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		nombre string
		mutar  func(*DatosSolicitudValidarRC)
	}{
		{"petición", func(d *DatosSolicitudValidarRC) {
			d.PeticionRef = "pet_otra234567890abcdefghijkl"
		}},
		{"organización", func(d *DatosSolicitudValidarRC) {
			d.OrganizacionRef = "organizacion_otra_0123456789"
		}},
		{"expediente", func(d *DatosSolicitudValidarRC) {
			d.ExpedienteRef = "expediente_otro_0123456789"
		}},
		{"versión", func(d *DatosSolicitudValidarRC) { d.VersionExpediente++ }},
		{"entrada", func(d *DatosSolicitudValidarRC) {
			d.Entrada.Referencia = "entrada_rc_otra_0123456789"
		}},
		{"huella entrada", func(d *DatosSolicitudValidarRC) {
			d.Entrada.HuellaSHA256 = cadenaFuenteAnalisisPrueba("c")
		}},
		{"existencia", func(d *DatosSolicitudValidarRC) {
			d.Declaracion.Existe = false
		}},
		{"número", func(d *DatosSolicitudValidarRC) {
			d.Declaracion.Numero = "rc_2026_otro_0123456789"
		}},
		{"fecha", func(d *DatosSolicitudValidarRC) {
			d.Declaracion.Fecha = d.Declaracion.Fecha.AddDate(0, 0, 1)
		}},
		{"importe", func(d *DatosSolicitudValidarRC) {
			d.Declaracion.Importe.Centimos++
		}},
		{"moneda", func(d *DatosSolicitudValidarRC) {
			d.Declaracion.Importe.Moneda = "USD"
		}},
		{"documento", func(d *DatosSolicitudValidarRC) {
			d.Declaracion.DocumentoRef = "documento_rc_otro_0123456789"
		}},
		{"instante", func(d *DatosSolicitudValidarRC) {
			d.SolicitadaEn = d.SolicitadaEn.Add(time.Microsecond)
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			mutada := base
			caso.mutar(&mutada)
			canon, err := canonPeticionValidacionRC(mutada)
			if err != nil || bytes.Equal(canonBase, canon) {
				t.Fatalf("coordenada no ligada: %v", err)
			}
		})
	}
}

func TestCanonPeticionCosteLigaCadaCoordenada(t *testing.T) {
	inicio := instanteFuenteAnalisisPrueba()
	base := datosCanonicosCostePrueba(
		preparacionCalcularCostePrueba(),
		"pet_abcdefghij0123456789klmn",
		inicio,
	)
	canonBase, err := canonPeticionCalculoCoste(base)
	if err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		nombre string
		mutar  func(*DatosSolicitudCalcularCoste)
	}{
		{"petición", func(d *DatosSolicitudCalcularCoste) {
			d.PeticionRef = "pet_otrodefghij0123456789klmn"
		}},
		{"organización", func(d *DatosSolicitudCalcularCoste) {
			d.OrganizacionRef = "organizacion_otra_0123456789"
		}},
		{"expediente", func(d *DatosSolicitudCalcularCoste) {
			d.ExpedienteRef = "expediente_otro_0123456789"
		}},
		{"versión", func(d *DatosSolicitudCalcularCoste) { d.VersionExpediente++ }},
		{"categoría", func(d *DatosSolicitudCalcularCoste) {
			d.CategoriaRef = "categoria_psicologia_012345"
		}},
		{"grupo", func(d *DatosSolicitudCalcularCoste) { d.GrupoSubgrupo = "C1" }},
		{"modalidad", func(d *DatosSolicitudCalcularCoste) {
			d.ModalidadClave = "programa_temporal"
		}},
		{"causa", func(d *DatosSolicitudCalcularCoste) { d.CausaClave = "vacante" }},
		{"inicio", func(d *DatosSolicitudCalcularCoste) {
			d.Periodo.Inicio = d.Periodo.Inicio.AddDate(0, 0, 1)
		}},
		{"fin", func(d *DatosSolicitudCalcularCoste) {
			d.Periodo.Fin = d.Periodo.Fin.AddDate(0, 0, 1)
		}},
		{"jornada", func(d *DatosSolicitudCalcularCoste) { d.Jornada = 5_000 }},
		{"instante", func(d *DatosSolicitudCalcularCoste) {
			d.SolicitadaEn = d.SolicitadaEn.Add(time.Microsecond)
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			mutada := base
			caso.mutar(&mutada)
			canon, err := canonPeticionCalculoCoste(mutada)
			if err != nil || bytes.Equal(canonBase, canon) {
				t.Fatalf("coordenada no ligada: %v", err)
			}
		})
	}
}

func TestCanonRespuestaLigaSalidasAutoridadGeneracionYVentana(t *testing.T) {
	inicio := instanteFuenteAnalisisPrueba()
	solicitud := solicitudCalcularCostePrueba(t, inicio)
	metadatos := metadatosRespuestaPrueba(
		"tabla_retributiva_2026_v3",
		"recibo_coste_0123456789",
		inicio,
	)
	importe := domain.Importe{Centimos: 3_148_025, Moneda: "EUR"}
	calculadoEn := inicio.Add(time.Second)
	base, err := canonRespuestaCalculoCoste(
		solicitud,
		metadatos.AutoridadRef,
		metadatos.ReciboRef,
		importe,
		calculadoEn,
		metadatos,
	)
	if err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		nombre    string
		fuente    string
		recibo    string
		importe   domain.Importe
		calculado time.Time
		meta      MetadatosAtestacionRespuestaFuenteAnalisis
	}{
		{"fuente", "otra_fuente_0123456789", metadatos.ReciboRef, importe, calculadoEn, metadatos},
		{"recibo", metadatos.AutoridadRef, "otro_recibo_0123456789", importe, calculadoEn, metadatos},
		{"importe", metadatos.AutoridadRef, metadatos.ReciboRef, domain.Importe{Centimos: importe.Centimos + 1, Moneda: "EUR"}, calculadoEn, metadatos},
		{"instante salida", metadatos.AutoridadRef, metadatos.ReciboRef, importe, calculadoEn.Add(time.Microsecond), metadatos},
	}
	metaAutoridad := metadatos
	metaAutoridad.AutoridadRef = "otra_fuente_0123456789"
	metaGeneracion := metadatos
	metaGeneracion.Generacion++
	metaVentana := metadatos
	metaVentana.ValidaHasta = metaVentana.ValidaHasta.Add(-time.Microsecond)
	casos = append(casos,
		struct {
			nombre    string
			fuente    string
			recibo    string
			importe   domain.Importe
			calculado time.Time
			meta      MetadatosAtestacionRespuestaFuenteAnalisis
		}{"autoridad", metadatos.AutoridadRef, metadatos.ReciboRef, importe, calculadoEn, metaAutoridad},
		struct {
			nombre    string
			fuente    string
			recibo    string
			importe   domain.Importe
			calculado time.Time
			meta      MetadatosAtestacionRespuestaFuenteAnalisis
		}{"generación", metadatos.AutoridadRef, metadatos.ReciboRef, importe, calculadoEn, metaGeneracion},
		struct {
			nombre    string
			fuente    string
			recibo    string
			importe   domain.Importe
			calculado time.Time
			meta      MetadatosAtestacionRespuestaFuenteAnalisis
		}{"ventana", metadatos.AutoridadRef, metadatos.ReciboRef, importe, calculadoEn, metaVentana},
	)
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			canon, err := canonRespuestaCalculoCoste(
				solicitud, caso.fuente, caso.recibo, caso.importe,
				caso.calculado, caso.meta,
			)
			if err != nil || bytes.Equal(base, canon) {
				t.Fatalf("salida no ligada: %v", err)
			}
		})
	}
}

func TestCanonRespuestaRCLigaCadaSalidaYMotivoPublicado(t *testing.T) {
	inicio := instanteFuenteAnalisisPrueba()
	solicitud := solicitudValidarRCPrueba(t, inicio)
	validacion := validacionRCPrueba(t, solicitud, inicio.Add(time.Second))
	metadatos := metadatosRespuestaPrueba(
		validacion.FuenteRef,
		validacion.ReciboRef,
		inicio,
	)
	base, err := canonRespuestaValidacionRC(
		solicitud,
		validacion,
		MotivoFuenteAnalisis{},
		metadatos,
	)
	if err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		nombre string
		mutar  func(*domain.ValidacionRC)
	}{
		{"resultado", func(v *domain.ValidacionRC) { v.Resultado = domain.RCRechazada }},
		{"entrada", func(v *domain.ValidacionRC) { v.EntradaRef = "otra_entrada_0123456789" }},
		{"huella entrada", func(v *domain.ValidacionRC) {
			v.HuellaEntradaSHA256 = strings.Repeat("d", 64)
		}},
		{"fuente", func(v *domain.ValidacionRC) { v.FuenteRef = "otra_fuente_0123456789" }},
		{"recibo", func(v *domain.ValidacionRC) { v.ReciboRef = "otro_recibo_0123456789" }},
		{"instante", func(v *domain.ValidacionRC) { v.ValidadaEn = v.ValidadaEn.Add(time.Microsecond) }},
		{"fecha", func(v *domain.ValidacionRC) {
			fecha := v.FechaRC.AddDate(0, 0, 1)
			v.FechaRC = &fecha
		}},
		{"número", func(v *domain.ValidacionRC) { v.Numero = "rc_2026_otro_0123456789" }},
		{"importe", func(v *domain.ValidacionRC) { v.Importe.Centimos++ }},
		{"moneda", func(v *domain.ValidacionRC) { v.Importe.Moneda = "USD" }},
		{"documento", func(v *domain.ValidacionRC) {
			v.DocumentoRef = "otro_documento_rc_0123456789"
		}},
		{"motivo crudo", func(v *domain.ValidacionRC) { v.Motivo = "texto no permitido" }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			mutada := clonarValidacionRC(validacion)
			caso.mutar(&mutada)
			canon, err := canonRespuestaValidacionRC(
				solicitud,
				mutada,
				MotivoFuenteAnalisis{},
				metadatos,
			)
			if err != nil || bytes.Equal(base, canon) {
				t.Fatalf("salida RC no ligada: %v", err)
			}
		})
	}

	negativa := validacionRCNegativaPrueba(t, solicitud, inicio.Add(time.Second))
	motivo := motivoFuenteAnalisisPrueba(t)
	baseMotivo, err := canonRespuestaValidacionRC(
		solicitud,
		negativa,
		motivo,
		metadatos,
	)
	if err != nil {
		t.Fatal(err)
	}
	vinculo, _ := motivo.Datos()
	vinculo.CatalogoVersion++
	motivoMutado := MotivoFuenteAnalisis{datos: &vinculo}
	canonMutado, err := canonRespuestaValidacionRC(
		solicitud,
		negativa,
		motivoMutado,
		metadatos,
	)
	if err != nil || bytes.Equal(baseMotivo, canonMutado) {
		t.Fatalf("versión de catálogo no ligada: %v", err)
	}
}

func TestFuentesAplicanLimitesExactosDeDominio(t *testing.T) {
	inicio := instanteFuenteAnalisisPrueba()
	preparacion := preparacionCalcularCostePrueba()
	preparacion.Periodo = domain.PeriodoPrevisto{
		Inicio: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Fin:    time.Date(2126, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	if _, err := NuevaSolicitudCalcularCoste(
		context.Background(),
		generadorFijoFuenteAnalisis("pet_abcdefghij0123456789klmn"),
		selladorHMACFuenteAnalisisPrueba(),
		relojFijoFuenteAnalisis(inicio),
		preparacion,
	); err != nil {
		t.Fatalf("100 años rechazados: %v", err)
	}
	preparacion.Periodo.Fin = preparacion.Periodo.Fin.AddDate(0, 0, 1)
	if _, err := NuevaSolicitudCalcularCoste(
		context.Background(),
		generadorFijoFuenteAnalisis("pet_abcdefghij0123456789klmn"),
		selladorHMACFuenteAnalisisPrueba(),
		relojFijoFuenteAnalisis(inicio),
		preparacion,
	); err == nil {
		t.Fatal("periodo superior a 100 años aceptado")
	}
}

func datosCanonicosRCPrueba(
	preparacion PreparacionSolicitudValidarRC,
	referencia string,
	instante time.Time,
) DatosSolicitudValidarRC {
	return DatosSolicitudValidarRC{
		PeticionRef: referencia, OrganizacionRef: preparacion.OrganizacionRef,
		ExpedienteRef:     preparacion.ExpedienteRef,
		VersionExpediente: preparacion.VersionExpediente,
		Entrada:           preparacion.Entrada, Declaracion: preparacion.Declaracion,
		SolicitadaEn: instante,
	}
}

func datosCanonicosCostePrueba(
	preparacion PreparacionSolicitudCalcularCoste,
	referencia string,
	instante time.Time,
) DatosSolicitudCalcularCoste {
	return DatosSolicitudCalcularCoste{
		PeticionRef: referencia, OrganizacionRef: preparacion.OrganizacionRef,
		ExpedienteRef:     preparacion.ExpedienteRef,
		VersionExpediente: preparacion.VersionExpediente,
		CategoriaRef:      preparacion.CategoriaRef, GrupoSubgrupo: preparacion.GrupoSubgrupo,
		ModalidadClave: preparacion.ModalidadClave, CausaClave: preparacion.CausaClave,
		Periodo: preparacion.Periodo, Jornada: preparacion.Jornada,
		SolicitadaEn: instante,
	}
}

func cadenaFuenteAnalisisPrueba(caracter string) string {
	return strings.Repeat(caracter, 64)
}
