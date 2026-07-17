package ports

import (
	"errors"
	"strings"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
	pruebasvec "vec-diputacion-granada/internal/vec/pruebas"
)

var instanteComandoLlamamientoPrueba = time.Date(2026, time.July, 17, 12, 0, 0, 123_456_000, time.UTC)

func TestComandoGuardarPropuestaLlamamientoConservaInstantaneaCompletaYCopiasDefensivas(t *testing.T) {
	instantanea, propuesta, evidencia := materialesComandoLlamamientoPrueba(t, "principal")
	comando, err := NuevoComandoGuardarPropuestaLlamamiento(instantanea, propuesta, evidencia)
	if err != nil {
		t.Fatalf("construir comando: %v", err)
	}
	if err := comando.ValidarEn(instanteComandoLlamamientoPrueba.Add(time.Second)); err != nil {
		t.Fatalf("comando vigente rechazado: %v", err)
	}

	// La propuesta termina en el primer elegible; la instantanea debe conservar
	// tambien las posiciones posteriores que no se evaluaron.
	obtenida, propuestaObtenida, _, err := comando.Datos()
	if err != nil || len(obtenida.Entradas) != 2 || len(propuestaObtenida.Evaluaciones) != 1 {
		t.Fatalf("el comando perdio el orden completo: entradas=%d evaluaciones=%d error=%v",
			len(obtenida.Entradas), len(propuestaObtenida.Evaluaciones), err)
	}

	instantanea.Entradas[0].Participacion.Situaciones[0].EstadoClave = "mutado_origen"
	propuesta.Evaluaciones[0].Motivos[0].Clave = "mutado_origen"
	obtenida.Entradas[0].Participacion.Situaciones[0].EstadoClave = "mutado_salida"
	propuestaObtenida.Evaluaciones[0].Motivos[0].Clave = "mutado_salida"

	segundaInstantanea, segundaPropuesta, _, err := comando.Datos()
	if err != nil || segundaInstantanea.Entradas[0].Participacion.Situaciones[0].EstadoClave == "mutado_origen" ||
		segundaInstantanea.Entradas[0].Participacion.Situaciones[0].EstadoClave == "mutado_salida" ||
		segundaPropuesta.Evaluaciones[0].Motivos[0].Clave == "mutado_origen" ||
		segundaPropuesta.Evaluaciones[0].Motivos[0].Clave == "mutado_salida" {
		t.Fatalf("el comando compartio memoria mutable: %v", err)
	}
}

func TestComandoGuardarPropuestaLlamamientoDeniegaCrucesAlteracionYCaducidad(t *testing.T) {
	instantanea, propuesta, evidencia := materialesComandoLlamamientoPrueba(t, "uno")
	instantaneaAjena, _, _ := materialesComandoLlamamientoPrueba(t, "dos")
	if _, err := NuevoComandoGuardarPropuestaLlamamiento(instantaneaAjena, propuesta, evidencia); !errors.Is(
		err, ErrComandoGuardarPropuestaLlamamientoInvalido,
	) {
		t.Fatalf("instantanea ajena admitida: %v", err)
	}
	if _, err := NuevoComandoGuardarPropuestaLlamamiento(
		instantanea, propuesta, puertosvec.EvidenciaUsoDecisionAutorizacion{},
	); !errors.Is(err, puertosvec.ErrEvidenciaUsoDecisionAutorizacionInvalida) {
		t.Fatalf("evidencia cero admitida: %v", err)
	}
	comando, err := NuevoComandoGuardarPropuestaLlamamiento(instantanea, propuesta, evidencia)
	if err != nil {
		t.Fatal(err)
	}
	if err := comando.ValidarEn(instanteComandoLlamamientoPrueba.Add(3 * time.Minute)); !errors.Is(err, puertosvec.ErrEvidenciaUsoDecisionAutorizacionInvalida) {
		t.Fatalf("capacidad caducada admitida: %v", err)
	}
	comando.datos.instantanea.Entradas[0].Participacion.Situaciones[0].EstadoClave = "alterado"
	if _, _, _, err := comando.Datos(); !errors.Is(err, ErrComandoGuardarPropuestaLlamamientoInvalido) {
		t.Fatalf("alteracion interna no detectada: %v", err)
	}
	if _, _, _, err := (ComandoGuardarPropuestaLlamamiento{}).Datos(); !errors.Is(
		err, ErrComandoGuardarPropuestaLlamamientoInvalido,
	) {
		t.Fatalf("valor cero admitido: %v", err)
	}
}

func materialesComandoLlamamientoPrueba(
	t *testing.T,
	sufijo string,
) (
	dominiobolsa.InstantaneaOrdenBolsa,
	dominiobolsa.PropuestaLlamamiento,
	puertosvec.EvidenciaUsoDecisionAutorizacion,
) {
	t.Helper()
	instante := instanteComandoLlamamientoPrueba
	huella := func(caracter byte) string { return strings.Repeat(string(caracter), 64) }
	bolsa, err := dominiobolsa.NuevaBolsaConstituida(dominiobolsa.AltaBolsaConstituida{
		BolsaRef: "bolsa:comando:" + sufijo, Version: 1, ProcesoRef: "proceso:comando:" + sufijo,
		CategoriaRef: "categoria:comando", ListadoDefinitivoRef: "listado:comando:" + sufijo,
		VersionListado: 1, HuellaListadoSHA256: huella('a'),
		ResolucionConstitucionRef: "resolucion:comando:" + sufijo, HuellaResolucionSHA256: huella('b'),
		ConstituidaEn: instante.Add(-48 * time.Hour), VigenteDesde: instante.Add(-24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	huellaBolsa, _ := bolsa.HuellaCanonicaSHA256()
	necesidad, err := dominiobolsa.NuevaNecesidadCobertura(dominiobolsa.AltaNecesidadCobertura{
		NecesidadRef: "necesidad:comando", Version: 1, BolsaRef: bolsa.BolsaRef,
		VersionBolsa: bolsa.Version, HuellaBolsaSHA256: huellaBolsa, CategoriaRef: bolsa.CategoriaRef,
		PuestoRef: "puesto:comando", UnidadRef: "unidad:comando", TipoCoberturaRef: "tipo:comando",
		NumeroPuestos: 1, InicioPrevisto: instante.Add(time.Hour), FinPrevisto: instante.Add(30 * 24 * time.Hour),
		CreadaEn: instante.Add(-time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	politica, err := dominiobolsa.NuevaReferenciaPoliticaLlamamiento(dominiobolsa.ReferenciaPoliticaLlamamiento{
		PoliticaRef: "politica:comando", Clave: "politica_comando_gobernada", Version: 1,
		HuellaSHA256: huella('c'), PublicadaEn: instante.Add(-48 * time.Hour), VigenteDesde: instante.Add(-24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	entradas := make([]dominiobolsa.EntradaOrdenBolsa, 2)
	for indice := range entradas {
		orden := indice + 1
		participacion, err := dominiobolsa.NuevaParticipacionBolsa(dominiobolsa.AltaParticipacionBolsa{
			ParticipacionRef: "participacion:comando:" + sufijo + ":" + string(rune('1'+indice)),
			BolsaRef:         bolsa.BolsaRef, SujetoRef: "sujeto:comando:" + sufijo + ":" + string(rune('1'+indice)),
			Version: 1, AltaEn: instante.Add(-12 * time.Hour),
			Situaciones: []dominiobolsa.SituacionParticipacionBolsa{{
				Secuencia: 1, EstadoClave: "estado_comando_gobernado", EstadoVersion: 1,
				HuellaEstadoSHA256: huella('d'), CausaClave: "causa_comando_gobernada", CausaVersion: 1,
				HuellaCausaSHA256: huella('e'), DecisionRef: "decision:situacion:comando:" + sufijo + ":" + string(rune('1'+indice)),
				HuellaDecisionSHA256: huella('f'), Desde: instante.Add(-12 * time.Hour),
			}},
		})
		if err != nil {
			t.Fatal(err)
		}
		entradas[indice] = dominiobolsa.EntradaOrdenBolsa{Orden: uint64(orden), Participacion: participacion}
	}
	instantanea, err := dominiobolsa.NuevaInstantaneaOrdenBolsa(dominiobolsa.AltaInstantaneaOrdenBolsa{
		InstantaneaRef: "instantanea:comando:" + sufijo, Version: 1, Bolsa: bolsa,
		ReferidaEn: instante, GeneradaEn: instante, Entradas: entradas,
	})
	if err != nil {
		t.Fatal(err)
	}
	situacion, _ := entradas[0].Participacion.SituacionVigenteEn(instante)
	huellaNecesidad, _ := necesidad.HuellaCanonicaSHA256()
	evaluacion := dominiobolsa.EvaluacionParticipacionLlamamiento{
		ParticipacionRef: entradas[0].Participacion.ParticipacionRef, SujetoRef: entradas[0].Participacion.SujetoRef,
		Orden: 1, SituacionSecuencia: situacion.Secuencia, EstadoClave: situacion.EstadoClave,
		EstadoVersion: situacion.EstadoVersion, HuellaEstadoSHA256: situacion.HuellaEstadoSHA256,
		NecesidadRef: necesidad.NecesidadRef, VersionNecesidad: necesidad.Version, HuellaNecesidadSHA256: huellaNecesidad,
		InstantaneaRef: instantanea.InstantaneaRef, VersionInstantanea: instantanea.Version,
		HuellaInstantaneaSHA256: instantanea.HuellaContenidoSHA256,
		PoliticaRef:             politica.PoliticaRef, VersionPolitica: politica.Version, HuellaPoliticaSHA256: politica.HuellaSHA256,
		Resultado: dominiobolsa.ResultadoElegible,
		Motivos: []dominiobolsa.MotivoEvaluacionLlamamiento{{
			Clave: "resultado_comando", ReglaRef: "regla:comando", VersionRegla: 1, HuellaReglaSHA256: huella('1'),
		}},
		EntradaEvaluacionRef: "recibo:entrada:comando:" + sufijo, HuellaEntradaSHA256: huella('2'),
		ResultadoEvaluacionRef: "recibo:resultado:comando:" + sufijo, HuellaResultadoSHA256: huella('3'),
		EvaluadaEn: instante,
	}
	propuesta, err := dominiobolsa.ProponerPrimerLlamamiento(dominiobolsa.OrdenProponerPrimerLlamamiento{
		PropuestaRef: "propuesta:comando:" + sufijo, Bolsa: bolsa, Necesidad: necesidad,
		Instantanea: instantanea, Politica: politica,
		Evaluaciones: []dominiobolsa.EvaluacionParticipacionLlamamiento{evaluacion}, GeneradaEn: instante,
	})
	if err != nil {
		t.Fatal(err)
	}
	vinculo, err := pruebasvec.NuevoVinculoGenerico(instante)
	if err != nil {
		t.Fatal(err)
	}
	datosVinculo, _ := vinculo.Datos()
	recurso := dominiovec.RecursoAutorizable{
		Referencia: propuesta.NecesidadRef, ModuloID: ModuloLlamamientos, Tipo: TipoRecursoNecesidad,
		Ambitos: map[string]string{"categoria_ref": bolsa.CategoriaRef, "unidad_ref": necesidad.UnidadRef},
	}
	huellaRecurso, _ := recurso.HuellaContextoAutorizacionSHA256()
	politicaSeguridad := "politica:seguridad:comando"
	huellas := map[string]string{politicaSeguridad: huella('4')}
	huellaCatalogo, _ := dominiovec.HuellaEvidenciasCatalogoPoliticasAutorizacion([]string{politicaSeguridad}, huellas)
	decision := dominiovec.DecisionAutorizacion{
		DecisionRef: "decision:comando:" + sufijo, Concedida: true, Codigo: "concedida",
		PrincipalID: datosVinculo.PrincipalID, PerfilActivoRef: datosVinculo.PerfilActivoRef,
		Accion: AccionProponerLlamamiento, RecursoRef: propuesta.NecesidadRef,
		ModuloID: ModuloLlamamientos, TipoRecurso: TipoRecursoNecesidad,
		ContextoRecursoHuellaSHA256: huellaRecurso, Finalidad: FinalidadProponerLlamamiento,
		CorrelacionRef: "correlacion:comando:" + sufijo, VinculoAutenticacionActor: vinculo,
		AsignacionRef: "asignacion:comando", AsignacionHuellaSHA256: huella('5'),
		VersionRolRef: "rol:comando", VersionRolHuellaSHA256: huella('6'),
		ControlVigenciaVersionRolRef: "rol:comando", ControlVigenciaVersionRolRevision: 1,
		ControlVigenciaVersionRolHuellaSHA256: huella('7'), RevisionCatalogoPoliticas: 1,
		CatalogoPoliticasHuellaSHA256: huellaCatalogo, PoliticasEvaluadasRefs: []string{politicaSeguridad},
		PoliticasEvaluadasHuellasSHA256: huellas, PoliticasRefs: []string{politicaSeguridad}, PoliticasHuellasSHA256: huellas,
		GarantiaMinima: dominiovec.AuthAssuranceHigh, EmitidaEn: instante.Add(-time.Second),
		ValidaHasta: instante.Add(2 * time.Minute),
	}
	evidencia, err := puertosvec.NuevaEvidenciaUsoDecisionAutorizacion(decision, instante)
	if err != nil {
		t.Fatal(err)
	}
	return instantanea, propuesta, evidencia
}
