package gobiernoreglasbaremo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	reglas "vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	"vec-diputacion-granada/internal/shared/baremacion"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

var instanteBasePrueba = time.Date(2026, 7, 17, 8, 30, 0, 123_456_000, time.UTC)

const actorPlanPrueba = "per_0123456789abcdef0123456789abcdef"

const (
	referenciaReglasPlanPrueba       = "rgl_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	referenciaConvocatoriaPlanPrueba = "con_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	referenciaExpedientePlanPrueba   = "exp_cccccccccccccccccccccccccccccccc"
)

type generadorCorrelacionPrueba struct{ valor string }

func (g generadorCorrelacionPrueba) NuevaReferenciaCorrelacionAutorizacionV2(
	context.Context,
) (string, error) {
	return g.valor, nil
}

func correlacionPrueba(t *testing.T) dominiovec.ReferenciaCorrelacionAutorizacionV2 {
	t.Helper()
	resultado, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(
		context.Background(),
		generadorCorrelacionPrueba{valor: "correlacion_0123456789abcdef0123456789abcdef"},
	)
	if err != nil {
		t.Fatal(err)
	}
	return resultado
}

func contextoActorPrueba(t *testing.T) dominiovec.ContextoActor {
	return contextoActorConPrincipalPrueba(t, actorPlanPrueba)
}

func contextoActorConPrincipalPrueba(
	t *testing.T,
	principalRef string,
) dominiovec.ContextoActor {
	t.Helper()
	cuenta := dominiovec.CuentaAutenticadaContextoActor{
		CuentaRef: "cta_0123456789abcdef0123456789abcdef",
		Metodo:    dominiovec.AuthMethodCertificate,
		Garantia:  dominiovec.AuthAssuranceHigh,
	}
	instantanea := dominiovec.InstantaneaContextoActor{
		VinculoRef:      "vca_0123456789abcdef0123456789abcdef",
		VinculoVersion:  5,
		CuentaRef:       cuenta.CuentaRef,
		PersonaRef:      principalRef,
		PersonaVersion:  3,
		PerfilActivoRef: "prf_0123456789abcdef0123456789abcdef",
		PerfilVersion:   4,
		Estado:          dominiovec.EstadoVinculoContextoActorActivo,
		VigenteDesde:    instanteBasePrueba.Add(-time.Hour),
		VigenteHasta:    instanteBasePrueba.Add(24 * time.Hour),
	}
	actor, err := dominiovec.NuevoContextoActor(cuenta, instantanea, instanteBasePrueba)
	if err != nil {
		t.Fatal(err)
	}
	return actor
}

func referenciaMotivoVersionPrueba(
	t *testing.T,
	version reglas.VersionGobernadaReglasBaremo,
) dominiovec.ReferenciaEntradaCatalogo {
	t.Helper()
	motivo, err := version.MotivoUltimaActuacion()
	if err != nil {
		t.Fatal(err)
	}
	catalogo := motivo.Catalogo()
	resultado := dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID:           catalogo.Referencia(),
		CatalogoVersion:      int(catalogo.Version()),
		CatalogoHuellaSHA256: catalogo.HuellaSHA256(),
		EntradaClave:         motivo.Clave(),
	}
	if !dominiovec.ReferenciaMotivoAutorizacionV2Valida(resultado) {
		t.Fatalf("motivo de prueba no es VEC V2: %#v", resultado)
	}
	return resultado
}

func referenciaPrueba(
	t *testing.T,
	referencia string,
	version uint64,
) reglas.ReferenciaVersionada {
	t.Helper()
	suma := sha256.Sum256([]byte(referencia))
	resultado, err := reglas.NuevaReferenciaVersionada(
		referencia,
		version,
		hex.EncodeToString(suma[:]),
	)
	if err != nil {
		t.Fatal(err)
	}
	return resultado
}

func referenciaTecnicaPrueba(
	t *testing.T,
	prefijo string,
	semilla string,
) reglas.ReferenciaVersionada {
	t.Helper()
	suma := sha256.Sum256([]byte(semilla))
	huella := hex.EncodeToString(suma[:])
	resultado, err := reglas.NuevaReferenciaVersionada(
		prefijo+huella,
		versionReferenciaV2,
		huella,
	)
	if err != nil {
		t.Fatal(err)
	}
	return resultado
}

func intencionPrueba(t *testing.T) IntencionGobiernoReglasBaremoV2 {
	t.Helper()
	resultado, err := NuevaIntencionGobiernoReglasBaremoV2(
		referenciaTecnicaPrueba(t, prefijoIntencionV2, "intencion-prueba"),
	)
	if err != nil {
		t.Fatal(err)
	}
	return resultado
}

func conjuntoPrueba(t *testing.T) reglas.ConjuntoReglasBaremo {
	t.Helper()
	identidad, err := reglas.NuevaIdentidadConjuntoReglasBaremo(
		referenciaReglasPlanPrueba,
		1,
		referenciaConvocatoriaPlanPrueba,
		referenciaExpedientePlanPrueba,
	)
	if err != nil {
		t.Fatal(err)
	}
	fecha, err := baremacion.NuevaFechaCivil(2026, 7, 17)
	if err != nil {
		t.Fatal(err)
	}
	cero, _ := baremacion.PuntosDesdeMicropuntos(0)
	diez, _ := baremacion.PuntosDesdeMicropuntos(10_000_000)
	seccion, err := reglas.NuevaSeccionBaremo(
		"experiencia",
		referenciaPrueba(t, "definicion:seccion:experiencia", 1),
		1,
		cero,
		diez,
	)
	if err != nil {
		t.Fatal(err)
	}
	criterio, err := reglas.NuevoCriterioExperiencia(
		"empleador",
		referenciaPrueba(t, "catalogo:empleadores", 1),
		[]string{"diputacion_granada"},
	)
	if err != nil {
		t.Fatal(err)
	}
	coincidencia, _ := reglas.NuevaPoliticaCoincidenciaReglas(
		reglas.CoincidenciaReglasRechazar,
	)
	solape, _ := reglas.NuevaPoliticaSolape(reglas.SolapeRechazar)
	grupo, err := reglas.NuevoGrupoConcurrenciaExperiencia(
		"grupo_experiencia",
		referenciaPrueba(t, "definicion:grupo:experiencia", 1),
		1,
		coincidencia,
		solape,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	uno, _ := baremacion.NuevoRacional(1, 1)
	temporal, _ := reglas.NuevaPoliticaUnidadTemporal(
		reglas.UnidadTemporalDia,
		reglas.UnidadTemporalDia,
		uno,
		reglas.ExtremoFinalInclusivo,
	)
	jornada, _ := reglas.NuevaPoliticaJornada(reglas.JornadaProporcional)
	restos, _ := reglas.NuevaPoliticaRestos(reglas.RestosConservarExactos)
	redondeo, _ := reglas.NuevaPoliticaRedondeo(
		reglas.RedondearPorRegla,
		baremacion.RedondeoTruncar,
	)
	puntos, _ := baremacion.PuntosDesdeMicropuntos(100_000)
	regla, err := reglas.NuevaReglaExperiencia(
		"servicios_diputacion",
		referenciaPrueba(t, "definicion:regla:servicios", 1),
		seccion.Clave(),
		1,
		[]reglas.CriterioExperiencia{criterio},
		grupo.Clave(),
		1,
		temporal,
		jornada,
		restos,
		redondeo,
		puntos,
		reglas.SinLimiteUnidades(),
		reglas.SinLimitePuntos(),
	)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := reglas.NuevoConjuntoReglasBaremo(
		identidad,
		referenciaPrueba(t, "documento:bases:test", 1),
		fecha,
		[]reglas.SeccionBaremo{seccion},
		[]reglas.GrupoConcurrenciaExperiencia{grupo},
		[]reglas.ReglaExperiencia{regla},
	)
	if err != nil {
		t.Fatal(err)
	}
	return resultado
}

func motivoPrueba(t *testing.T, clave string) reglas.MotivoCatalogadoReglasBaremo {
	t.Helper()
	suma := sha256.Sum256([]byte(clave))
	claveOpaca := "motivo_" + hex.EncodeToString(suma[:16])
	resultado, err := reglas.NuevoMotivoCatalogadoReglasBaremo(
		referenciaPrueba(t, "motivos_autorizacion", 1),
		claveOpaca,
	)
	if err != nil {
		t.Fatal(err)
	}
	return resultado
}

func borradorPrueba(t *testing.T) reglas.VersionGobernadaReglasBaremo {
	t.Helper()
	resultado, err := reglas.NuevaVersionGobernadaReglasBaremo(
		conjuntoPrueba(t),
		actorPlanPrueba,
		motivoPrueba(t, "creacion"),
		instanteBasePrueba,
	)
	if err != nil {
		t.Fatal(err)
	}
	return resultado
}

func publicadaPrueba(
	t *testing.T,
	borrador reglas.VersionGobernadaReglasBaremo,
) reglas.VersionGobernadaReglasBaremo {
	t.Helper()
	resultado, _ := publicadaConEvidenciaPrueba(t, borrador)
	return resultado
}

func publicadaConEvidenciaPrueba(
	t *testing.T,
	borrador reglas.VersionGobernadaReglasBaremo,
) (
	reglas.VersionGobernadaReglasBaremo,
	reglas.AtestacionAprobacionFirmadaReglasBaremo,
) {
	t.Helper()
	vinculo, _ := borrador.VinculoEstado()
	aprobacion, err := reglas.NuevaAtestacionAprobacionFirmadaReglasBaremo(
		reglas.DatosAtestacionAprobacionFirmadaReglasBaremo{
			Atestacion: referenciaTecnicaPrueba(
				t, prefijoAtestacionV2, "atestacion-aprobacion-test",
			),
			Vinculo:       vinculo,
			Firma:         referenciaPrueba(t, "firma:aprobacion:test", 1),
			PoliticaFirma: referenciaPrueba(t, "politica:firma:test", 1),
			Firmantes:     []string{"per_11111111111111111111111111111111"},
			FirmadaEn:     instanteBasePrueba.Add(time.Minute),
			VerificadaEn:  instanteBasePrueba.Add(2 * time.Minute),
			ValidaHasta:   instanteBasePrueba.Add(10 * time.Minute),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := borrador.Publicar(
		borrador.Revision(),
		actorPlanPrueba,
		motivoPrueba(t, "publicacion"),
		aprobacion,
		instanteBasePrueba.Add(3*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	return resultado, aprobacion
}

func activaPrueba(
	t *testing.T,
	publicada reglas.VersionGobernadaReglasBaremo,
) reglas.VersionGobernadaReglasBaremo {
	t.Helper()
	resultado, _ := activaConEvidenciaPrueba(t, publicada)
	return resultado
}

func activaConEvidenciaPrueba(
	t *testing.T,
	publicada reglas.VersionGobernadaReglasBaremo,
) (
	reglas.VersionGobernadaReglasBaremo,
	reglas.AtestacionDependenciasVigentesReglasBaremo,
) {
	t.Helper()
	vinculo, _ := publicada.VinculoEstado()
	dependencias, _ := publicada.DependenciasContenido()
	conjunto, _ := publicada.Conjunto()
	atestacion, err := reglas.NuevaAtestacionDependenciasVigentesReglasBaremo(
		reglas.DatosAtestacionDependenciasVigentesReglasBaremo{
			Atestacion: referenciaTecnicaPrueba(
				t, prefijoAtestacionV2, "atestacion-dependencias-test",
			),
			Vinculo:        vinculo,
			Convocatoria:   referenciaPrueba(t, conjunto.Identidad().ConvocatoriaRef(), 1),
			Bases:          conjunto.Bases(),
			Dependencias:   dependencias,
			VerificadorRef: "svc_0123456789abcdef0123456789abcdef",
			VerificadaEn:   instanteBasePrueba.Add(4 * time.Minute),
			ValidaHasta:    instanteBasePrueba.Add(12 * time.Minute),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := publicada.Activar(
		publicada.Revision(),
		actorPlanPrueba,
		motivoPrueba(t, "activacion"),
		atestacion,
		instanteBasePrueba.Add(5*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	return resultado, atestacion
}

func terminalPrueba(
	t *testing.T,
	version reglas.VersionGobernadaReglasBaremo,
	accion reglas.AccionGobiernoReglasBaremo,
) reglas.VersionGobernadaReglasBaremo {
	t.Helper()
	resultado, _ := terminalConEvidenciaPrueba(t, version, accion)
	return resultado
}

func terminalConEvidenciaPrueba(
	t *testing.T,
	version reglas.VersionGobernadaReglasBaremo,
	accion reglas.AccionGobiernoReglasBaremo,
) (
	reglas.VersionGobernadaReglasBaremo,
	reglas.AtestacionAutoridadReglasBaremo,
) {
	t.Helper()
	vinculo, _ := version.VinculoEstado()
	var relacionada *reglas.ReferenciaVersionada
	if accion == reglas.AccionSustituirReglasBaremo {
		valor := referenciaPrueba(t, "reglas:sucesoras:test", 1)
		relacionada = &valor
	}
	autoridad, err := reglas.NuevaAtestacionAutoridadReglasBaremo(
		reglas.DatosAtestacionAutoridadReglasBaremo{
			Atestacion: referenciaTecnicaPrueba(
				t, prefijoAtestacionV2, "atestacion-autoridad-"+string(accion),
			),
			Vinculo:      vinculo,
			Accion:       accion,
			PrincipalRef: actorPlanPrueba,
			Relacionada:  relacionada,
			EmitidaEn:    instanteBasePrueba.Add(6 * time.Minute),
			ValidaHasta:  instanteBasePrueba.Add(15 * time.Minute),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	instante := instanteBasePrueba.Add(7 * time.Minute)
	switch accion {
	case reglas.AccionSustituirReglasBaremo:
		resultado, err := version.Sustituir(
			version.Revision(),
			actorPlanPrueba,
			motivoPrueba(t, "sustitucion"),
			*relacionada,
			autoridad,
			instante,
		)
		if err != nil {
			t.Fatal(err)
		}
		return resultado, autoridad
	case reglas.AccionRetirarReglasBaremo:
		resultado, err := version.Retirar(
			version.Revision(),
			actorPlanPrueba,
			motivoPrueba(t, "retirada"),
			autoridad,
			instante,
		)
		if err != nil {
			t.Fatal(err)
		}
		return resultado, autoridad
	case reglas.AccionDescartarReglasBaremo:
		resultado, err := version.Descartar(
			version.Revision(),
			actorPlanPrueba,
			motivoPrueba(t, "descarte"),
			autoridad,
			instante,
		)
		if err != nil {
			t.Fatal(err)
		}
		return resultado, autoridad
	default:
		t.Fatalf("accion terminal inesperada: %q", accion)
		return reglas.VersionGobernadaReglasBaremo{}, reglas.AtestacionAutoridadReglasBaremo{}
	}
}

func vinculoPrueba(
	t *testing.T,
	version reglas.VersionGobernadaReglasBaremo,
) reglas.VinculoEstadoReglasBaremo {
	t.Helper()
	vinculo, err := version.VinculoEstado()
	if err != nil {
		t.Fatal(err)
	}
	return vinculo
}

func planAltaPrueba(t *testing.T) PlanCambioReglasBaremoV2 {
	t.Helper()
	borrador := borradorPrueba(t)
	resultado, err := NuevoPlanCambioReglasBaremoV2(DatosNuevoPlanCambioReglasBaremoV2{
		Operacion:          OperacionAltaBorrador,
		Intencion:          intencionPrueba(t),
		VersionResultado:   borrador,
		ContextoActor:      contextoActorPrueba(t),
		ReferenciaMotivo:   referenciaMotivoVersionPrueba(t, borrador),
		Correlacion:        correlacionPrueba(t),
		InstanteTransicion: instanteBasePrueba,
	})
	if err != nil {
		t.Fatal(err)
	}
	return resultado
}
