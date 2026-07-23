package ports

import (
	"context"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func TestCanonValidacionRCLigaCadaCoordenada(t *testing.T) {
	inicio := instanteFuenteAnalisisPrueba()
	base := preparacionValidarRCPrueba()
	selloBase := selloPreparacionRCPrueba(
		t,
		base,
		"pet_0123456789abcdefghijklmn",
		inicio,
	)
	casos := []struct {
		nombre string
		mutar  func(*PreparacionSolicitudValidarRC)
	}{
		{"organización", func(p *PreparacionSolicitudValidarRC) {
			p.OrganizacionRef = "organizacion_otra_0123456789"
		}},
		{"expediente", func(p *PreparacionSolicitudValidarRC) {
			p.ExpedienteRef = "expediente_otro_0123456789"
		}},
		{"versión", func(p *PreparacionSolicitudValidarRC) {
			p.VersionExpediente++
		}},
		{"entrada", func(p *PreparacionSolicitudValidarRC) {
			p.Entrada.Referencia = "entrada_rc_otra_0123456789"
		}},
		{"huella entrada", func(p *PreparacionSolicitudValidarRC) {
			p.Entrada.HuellaSHA256 = cadenaFuenteAnalisisPrueba("c")
		}},
		{"existencia", func(p *PreparacionSolicitudValidarRC) {
			p.Declaracion = domain.DeclaracionRC{}
		}},
		{"número", func(p *PreparacionSolicitudValidarRC) {
			p.Declaracion.Numero = "rc_2026_otro_0123456789"
		}},
		{"fecha", func(p *PreparacionSolicitudValidarRC) {
			p.Declaracion.Fecha = p.Declaracion.Fecha.AddDate(0, 0, 1)
		}},
		{"importe", func(p *PreparacionSolicitudValidarRC) {
			p.Declaracion.Importe.Centimos++
		}},
		{"documento", func(p *PreparacionSolicitudValidarRC) {
			p.Declaracion.DocumentoRef = "documento_rc_otro_0123456789"
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			mutado := base
			caso.mutar(&mutado)
			if obtenido := selloPreparacionRCPrueba(
				t,
				mutado,
				"pet_0123456789abcdefghijklmn",
				inicio,
			); obtenido == selloBase {
				t.Fatal("la coordenada no alteró el HMAC")
			}
		})
	}
	if selloPreparacionRCPrueba(
		t,
		base,
		"pet_otra234567890abcdefghijkl",
		inicio,
	) == selloBase {
		t.Fatal("PeticionRef no quedó ligada")
	}
	if selloPreparacionRCPrueba(
		t,
		base,
		"pet_0123456789abcdefghijklmn",
		inicio.Add(time.Microsecond),
	) == selloBase {
		t.Fatal("SolicitadaEn no quedó ligada")
	}

	datos := datosCanonicosRCPrueba(base, "pet_0123456789abcdefghijklmn", inicio)
	canonBase, err := canonPeticionValidacionRC(datos)
	if err != nil {
		t.Fatal(err)
	}
	datos.Declaracion.Existe = false
	canonMutado, err := canonPeticionValidacionRC(datos)
	assertCanonDistintoFuenteAnalisis(t, canonBase, canonMutado, err)
	datos = datosCanonicosRCPrueba(base, "pet_0123456789abcdefghijklmn", inicio)
	datos.Declaracion.Importe.Moneda = "USD"
	canonMutado, err = canonPeticionValidacionRC(datos)
	assertCanonDistintoFuenteAnalisis(t, canonBase, canonMutado, err)
}

func TestCanonCosteLigaCadaCoordenada(t *testing.T) {
	inicio := instanteFuenteAnalisisPrueba()
	base := preparacionCalcularCostePrueba()
	selloBase := selloPreparacionCostePrueba(
		t,
		base,
		"pet_abcdefghij0123456789klmn",
		inicio,
	)
	casos := []struct {
		nombre string
		mutar  func(*PreparacionSolicitudCalcularCoste)
	}{
		{"organización", func(p *PreparacionSolicitudCalcularCoste) {
			p.OrganizacionRef = "organizacion_otra_0123456789"
		}},
		{"expediente", func(p *PreparacionSolicitudCalcularCoste) {
			p.ExpedienteRef = "expediente_otro_0123456789"
		}},
		{"versión", func(p *PreparacionSolicitudCalcularCoste) {
			p.VersionExpediente++
		}},
		{"categoría", func(p *PreparacionSolicitudCalcularCoste) {
			p.CategoriaRef = "categoria_psicologia_012345"
		}},
		{"grupo", func(p *PreparacionSolicitudCalcularCoste) {
			p.GrupoSubgrupo = "C1"
		}},
		{"modalidad", func(p *PreparacionSolicitudCalcularCoste) {
			p.ModalidadClave = "programa_temporal"
		}},
		{"causa", func(p *PreparacionSolicitudCalcularCoste) {
			p.CausaClave = "vacante"
		}},
		{"inicio", func(p *PreparacionSolicitudCalcularCoste) {
			p.Periodo.Inicio = p.Periodo.Inicio.AddDate(0, 0, 1)
		}},
		{"fin", func(p *PreparacionSolicitudCalcularCoste) {
			p.Periodo.Fin = p.Periodo.Fin.AddDate(0, 0, 1)
		}},
		{"jornada", func(p *PreparacionSolicitudCalcularCoste) {
			p.Jornada = 5_000
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			mutado := base
			caso.mutar(&mutado)
			if obtenido := selloPreparacionCostePrueba(
				t,
				mutado,
				"pet_abcdefghij0123456789klmn",
				inicio,
			); obtenido == selloBase {
				t.Fatal("la coordenada no alteró el HMAC")
			}
		})
	}
	if selloPreparacionCostePrueba(
		t,
		base,
		"pet_otrodefghij0123456789klmn",
		inicio,
	) == selloBase {
		t.Fatal("PeticionRef no quedó ligada")
	}
	if selloPreparacionCostePrueba(
		t,
		base,
		"pet_abcdefghij0123456789klmn",
		inicio.Add(time.Microsecond),
	) == selloBase {
		t.Fatal("SolicitadaEn no quedó ligada")
	}
}

func TestFuentesAplicanLimitesExactosDeDominio(t *testing.T) {
	inicio := instanteFuenteAnalisisPrueba()
	preparacionCoste := preparacionCalcularCostePrueba()
	preparacionCoste.Periodo = domain.PeriodoPrevisto{
		Inicio: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Fin:    time.Date(2126, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	if _, err := crearSolicitudCosteDesdePreparacion(
		preparacionCoste,
		inicio,
	); err != nil {
		t.Fatalf("periodo exacto de 100 años rechazado: %v", err)
	}
	preparacionCoste.Periodo.Fin = preparacionCoste.Periodo.Fin.AddDate(0, 0, 1)
	if _, err := crearSolicitudCosteDesdePreparacion(
		preparacionCoste,
		inicio,
	); err == nil {
		t.Fatal("periodo superior a 100 años aceptado")
	}

	preparacionRC := preparacionValidarRCPrueba()
	preparacionRC.Declaracion.Importe.Centimos = maximoCentimosFuente
	solicitudRC, err := crearSolicitudRCDesdePreparacion(preparacionRC, inicio)
	if err != nil {
		t.Fatalf("importe máximo RC rechazado: %v", err)
	}
	validacion := validacionRCPrueba(t, solicitudRC, inicio.Add(time.Second))
	validacion.Importe.Centimos = maximoCentimosFuente
	if _, err := NuevoResultadoValidacionRC(
		solicitudRC,
		validacion,
		MotivoFuenteAnalisis{},
	); err != nil {
		t.Fatalf("resultado RC máximo rechazado: %v", err)
	}
	preparacionRC.Declaracion.Importe.Centimos++
	if _, err := crearSolicitudRCDesdePreparacion(preparacionRC, inicio); err == nil {
		t.Fatal("importe RC superior al máximo aceptado")
	}

	solicitudCoste := solicitudCalcularCostePrueba(t, inicio)
	if _, err := NuevoResultadoCalculoCoste(
		solicitudCoste,
		"tabla_retributiva_2026_v3",
		"recibo_coste_0123456789",
		domain.Importe{Centimos: maximoCentimosFuente, Moneda: "EUR"},
		inicio.Add(time.Second),
	); err != nil {
		t.Fatalf("coste máximo rechazado: %v", err)
	}
	if _, err := NuevoResultadoCalculoCoste(
		solicitudCoste,
		"tabla_retributiva_2026_v3",
		"recibo_coste_0123456789",
		domain.Importe{Centimos: maximoCentimosFuente + 1, Moneda: "EUR"},
		inicio.Add(time.Second),
	); err == nil {
		t.Fatal("coste superior al máximo aceptado")
	}
}

func selloPreparacionRCPrueba(
	t *testing.T,
	preparacion PreparacionSolicitudValidarRC,
	referencia string,
	instante time.Time,
) string {
	t.Helper()
	solicitud, err := NuevaSolicitudValidarRC(
		context.Background(),
		generadorFijoFuenteAnalisis(referencia),
		selladorHMACFuenteAnalisisPrueba(),
		relojFijoFuenteAnalisis(instante),
		preparacion,
	)
	if err != nil {
		t.Fatalf("crear solicitud RC: %v", err)
	}
	datos, _ := solicitud.Datos()
	return datos.HuellaPeticionHMAC
}

func selloPreparacionCostePrueba(
	t *testing.T,
	preparacion PreparacionSolicitudCalcularCoste,
	referencia string,
	instante time.Time,
) string {
	t.Helper()
	solicitud, err := crearSolicitudCosteDesdePreparacionConReferencia(
		preparacion,
		referencia,
		instante,
	)
	if err != nil {
		t.Fatalf("crear solicitud de coste: %v", err)
	}
	datos, _ := solicitud.Datos()
	return datos.HuellaPeticionHMAC
}

func crearSolicitudRCDesdePreparacion(
	preparacion PreparacionSolicitudValidarRC,
	instante time.Time,
) (SolicitudValidarRC, error) {
	return NuevaSolicitudValidarRC(
		context.Background(),
		generadorFijoFuenteAnalisis("pet_0123456789abcdefghijklmn"),
		selladorHMACFuenteAnalisisPrueba(),
		relojFijoFuenteAnalisis(instante),
		preparacion,
	)
}

func crearSolicitudCosteDesdePreparacion(
	preparacion PreparacionSolicitudCalcularCoste,
	instante time.Time,
) (SolicitudCalcularCoste, error) {
	return crearSolicitudCosteDesdePreparacionConReferencia(
		preparacion,
		"pet_abcdefghij0123456789klmn",
		instante,
	)
}

func crearSolicitudCosteDesdePreparacionConReferencia(
	preparacion PreparacionSolicitudCalcularCoste,
	referencia string,
	instante time.Time,
) (SolicitudCalcularCoste, error) {
	return NuevaSolicitudCalcularCoste(
		context.Background(),
		generadorFijoFuenteAnalisis(referencia),
		selladorHMACFuenteAnalisisPrueba(),
		relojFijoFuenteAnalisis(instante),
		preparacion,
	)
}

func cadenaFuenteAnalisisPrueba(caracter string) string {
	resultado := ""
	for range 64 {
		resultado += caracter
	}
	return resultado
}

func datosCanonicosRCPrueba(
	preparacion PreparacionSolicitudValidarRC,
	referencia string,
	instante time.Time,
) DatosSolicitudValidarRC {
	return DatosSolicitudValidarRC{
		PeticionRef:       referencia,
		OrganizacionRef:   preparacion.OrganizacionRef,
		ExpedienteRef:     preparacion.ExpedienteRef,
		VersionExpediente: preparacion.VersionExpediente,
		Entrada:           preparacion.Entrada,
		Declaracion:       preparacion.Declaracion,
		SolicitadaEn:      instante,
	}
}

func assertCanonDistintoFuenteAnalisis(
	t *testing.T,
	base []byte,
	obtenido []byte,
	err error,
) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	if string(base) == string(obtenido) {
		t.Fatal("la coordenada no alteró la preimagen canónica")
	}
}
