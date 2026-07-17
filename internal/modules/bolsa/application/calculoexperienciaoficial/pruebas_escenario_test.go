package calculoexperienciaoficial

import (
	"strings"
	"testing"
	"time"

	calculo "vec-diputacion-granada/internal/modules/bolsa/domain/calculoexperiencia"
	oficial "vec-diputacion-granada/internal/modules/bolsa/domain/calculoexperienciaoficial"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type escenarioServicioPrueba struct {
	ahora       time.Time
	servicio    *Servicio
	orden       OrdenCalculoExperienciaOficial
	datosOrden  DatosOrdenConfiable
	fuente      *fuentePrueba
	exigidor    *exigidorPrueba
	confirmador *confirmadorPrueba
}

func nuevoEscenarioServicioPrueba(
	t *testing.T,
	perfil perfilServicio,
	bloqueado bool,
) escenarioServicioPrueba {
	t.Helper()
	ahora := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	version, convocatoria := versionActivaPrueba(t, ahora)
	entrada := entradaPrueba(t, bloqueado)
	estado := debePrueba(version.VinculoEstado())
	sujeto := referenciaPrueba(
		t, "hmac-sha256:seudonimo_oficial_v1:"+strings.Repeat("0", 64), 1,
	)
	selector := puertosbolsa.SelectorFuenteExactaCalculoReglasBaremo{
		EstadoReglas: estado, InstantaneaEntrada: entrada.Instantanea(),
		SujetoPseudonimo: sujeto, Convocatoria: convocatoria,
	}
	huellaEntrada := debePrueba(entrada.HuellaSHA256())
	fuente := puertosbolsa.FuenteExactaCalculoReglasBaremo{
		Version: version, Entrada: entrada,
		Prueba: puertosbolsa.PruebaFuenteExactaCalculoReglasBaremo{
			Evidencia: referenciaPrueba(
				t, tokenPrueba("evidencia:fuente:", "evidencia-fuente-oficial"), 1,
			),
			Verificador: referenciaPrueba(
				t, tokenPrueba("verificador:fuente:", "verificador-fuente-oficial"), 1,
			),
			EstadoReglas: estado, InstantaneaEntrada: entrada.Instantanea(),
			HuellaEntradaSHA256: huellaEntrada, SujetoPseudonimo: sujeto,
			Convocatoria: convocatoria, EmitidaEn: ahora.Add(-time.Minute),
			ValidaHasta: ahora.Add(10 * time.Minute),
		},
		Auditoria: referenciaPrueba(
			t, tokenPrueba("auditoria:fuente:", "auditoria-fuente-oficial"), 1,
		),
		ConsumoPrueba: referenciaPrueba(
			t, tokenPrueba("consumo:prueba:", "consumo-prueba-oficial"), 1,
		),
		ObtenidaEn: ahora,
	}
	plan := debePrueba(calculo.Compilar(debePrueba(version.Conjunto())))
	resultado := debePrueba(calculo.CalcularExperienciaV1(plan, entrada))
	motorResultado := resultado.Vinculos().Motor()
	motor := oficial.VinculoMotorV1{
		Contrato: motorResultado.Contrato(), Version: motorResultado.Version(),
		HuellaContratoSHA256: motorResultado.HuellaContratoSHA256(),
	}
	superficie := dominiovec.SuperficieAutenticacionExternaPersonalV1
	garantia := dominiovec.AuthAssuranceSubstantial
	if perfil == perfilInternoAlto {
		superficie = dominiovec.SuperficieAutenticacionInternaCorporativaV1
		garantia = dominiovec.AuthAssuranceHigh
	}
	actor, vinculo := sesionPrueba(t, ahora, superficie, garantia)
	motivo := dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID: "catalogo_motivos_calculo", CatalogoVersion: 1,
		CatalogoHuellaSHA256: strings.Repeat("d", 64),
		EntradaClave:         "motivo_0123456789abcdef0123456789abcdef",
	}
	datosOrden := DatosOrdenConfiable{
		ContextoActor: actor, VinculoAutenticacionActor: vinculo, Selector: selector,
		Motivo: motivo, CorrelacionLectura: correlacionPrueba(t, "1"),
		CorrelacionEscritura: correlacionPrueba(t, "2"),
		Causa: oficial.CausaGobernadaV1{
			Catalogo: oficial.ReferenciaExactaV1{
				Referencia: motivo.CatalogoID, Version: uint64(motivo.CatalogoVersion),
				HuellaSHA256: motivo.CatalogoHuellaSHA256,
			},
			Clave: motivo.EntradaClave,
		},
		MotorEsperado: motor, TipoEfecto: oficial.EfectoCalculoInicial,
	}
	orden := debePrueba(NuevaOrdenConfiable(datosOrden))
	dobleFuente := &fuentePrueba{resultado: fuente}
	exigidor := &exigidorPrueba{ahora: ahora}
	confirmador := &confirmadorPrueba{}
	var servicio *Servicio
	var err error
	if perfil == perfilExternoOrdinario {
		servicio, err = NuevoServicioExternoOrdinario(
			dobleFuente, exigidor, confirmador, confirmador, relojPrueba{ahora},
		)
	} else {
		servicio, err = NuevoServicioInternoAlto(
			dobleFuente, exigidor, confirmador, confirmador, relojPrueba{ahora},
		)
	}
	if err != nil {
		t.Fatal(err)
	}
	return escenarioServicioPrueba{
		ahora: ahora, servicio: servicio, orden: orden, datosOrden: datosOrden,
		fuente: dobleFuente, exigidor: exigidor, confirmador: confirmador,
	}
}
