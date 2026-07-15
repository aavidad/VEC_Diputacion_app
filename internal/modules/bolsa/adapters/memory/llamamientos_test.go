package memory

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var instanteMemoriaLlamamientoPrueba = time.Date(2026, time.July, 15, 10, 30, 0, 123_456_000, time.UTC)

type relojMemoriaLlamamiento struct{ instante time.Time }

func (r *relojMemoriaLlamamiento) Ahora() time.Time { return r.instante }

type revalidadorActorMemoriaFunc func(context.Context, dominiovec.SolicitudRevalidacionAutenticacionActorV1) (dominiovec.AutenticacionRevalidadaV1, error)

func (f revalidadorActorMemoriaFunc) RevalidarAutenticacionActorV1(
	ctx context.Context,
	s dominiovec.SolicitudRevalidacionAutenticacionActorV1,
) (dominiovec.AutenticacionRevalidadaV1, error) {
	return f(ctx, s)
}

func TestRegistroPropuestasLlamamientoIdempotenciaExactaYCopiasDefensivas(t *testing.T) {
	reloj := &relojMemoriaLlamamiento{instante: instanteMemoriaLlamamientoPrueba.Add(time.Second)}
	registro, err := NuevoRegistroPropuestasLlamamiento(reloj, PerfilRegistroPropuestasSoloPruebas())
	if err != nil {
		t.Fatal(err)
	}
	propuesta, evidencia := propuestaYEvidenciaMemoriaPrueba(t, "propuesta:memoria:0001", "instantanea:memoria:0001", "A", "decision:memoria:0001")
	if err := registro.GuardarPropuestaLlamamiento(context.Background(), propuesta, evidencia); err != nil {
		t.Fatalf("primer guardado: %v", err)
	}
	if err := registro.GuardarPropuestaLlamamiento(context.Background(), propuesta, evidencia); err != nil {
		t.Fatalf("reintento exacto: %v", err)
	}
	if registro.NumeroPropuestasParaPruebas() != 1 {
		t.Fatalf("el reintento duplico efecto: %d", registro.NumeroPropuestasParaPruebas())
	}
	propuesta.Evaluaciones[0].Motivos[0].Clave = "mutado_fuera"
	recuperada, err := registro.ObtenerPropuestaParaPruebas(context.Background(), "propuesta:memoria:0001")
	if err != nil || recuperada.Evaluaciones[0].Motivos[0].Clave == "mutado_fuera" {
		t.Fatalf("el registro compartio memoria de entrada: %+v / %v", recuperada, err)
	}
	recuperada.Evaluaciones[0].Motivos[0].Clave = "mutado_salida"
	segunda, _ := registro.ObtenerPropuestaParaPruebas(context.Background(), "propuesta:memoria:0001")
	if segunda.Evaluaciones[0].Motivos[0].Clave == "mutado_salida" {
		t.Fatal("la consulta compartio memoria interna")
	}
}

func TestRegistroPropuestasLlamamientoDeniegaSegundoEfectoParaMismaVersionDeNecesidad(t *testing.T) {
	reloj := &relojMemoriaLlamamiento{instante: instanteMemoriaLlamamientoPrueba.Add(time.Second)}
	registro, err := NuevoRegistroPropuestasLlamamiento(reloj, PerfilRegistroPropuestasSoloPruebas())
	if err != nil {
		t.Fatal(err)
	}
	primera, evidenciaPrimera := propuestaYEvidenciaMemoriaPrueba(
		t, "propuesta:memoria:negocio:0001", "instantanea:memoria:negocio:0001", "NA", "decision:memoria:negocio:0001",
	)
	segunda, evidenciaSegunda := propuestaYEvidenciaMemoriaPrueba(
		t, "propuesta:memoria:negocio:0002", "instantanea:memoria:negocio:0002", "NB", "decision:memoria:negocio:0002",
	)
	if err := registro.GuardarPropuestaLlamamiento(context.Background(), primera, evidenciaPrimera); err != nil {
		t.Fatalf("primer efecto: %v", err)
	}
	if err := registro.GuardarPropuestaLlamamiento(context.Background(), segunda, evidenciaSegunda); !errors.Is(err, puertosbolsa.ErrNecesidadLlamamientoYaPropuesta) {
		t.Fatalf("la misma necesidad/version admitio referencias y decision nuevas: %v", err)
	}
	if registro.NumeroPropuestasParaPruebas() != 1 {
		t.Fatalf("la clave de negocio no fue unica: %d", registro.NumeroPropuestasParaPruebas())
	}
}

func TestRegistroPropuestasLlamamientoMismaNecesidadConcurrenteSoloConfirmaUnEfecto(t *testing.T) {
	reloj := &relojMemoriaLlamamiento{instante: instanteMemoriaLlamamientoPrueba.Add(time.Second)}
	registro, err := NuevoRegistroPropuestasLlamamiento(reloj, PerfilRegistroPropuestasSoloPruebas())
	if err != nil {
		t.Fatal(err)
	}
	const total = 32
	var exitos atomic.Int32
	var denegaciones atomic.Int32
	var grupo sync.WaitGroup
	for indice := 0; indice < total; indice++ {
		indice := indice
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			propuesta, err := propuestaMemoriaPrueba(
				fmt.Sprintf("propuesta:memoria:negocio:%04d", indice+100),
				fmt.Sprintf("instantanea:memoria:negocio:%04d", indice+100),
				fmt.Sprintf("NC%04d", indice),
			)
			if err != nil {
				t.Errorf("propuesta concurrente: %v", err)
				return
			}
			evidencia, err := evidenciaMemoriaPrueba(propuesta, fmt.Sprintf("decision:memoria:negocio:%04d", indice+100))
			if err != nil {
				t.Errorf("evidencia concurrente: %v", err)
				return
			}
			err = registro.GuardarPropuestaLlamamiento(context.Background(), propuesta, evidencia)
			switch {
			case err == nil:
				exitos.Add(1)
			case errors.Is(err, puertosbolsa.ErrNecesidadLlamamientoYaPropuesta):
				denegaciones.Add(1)
			default:
				t.Errorf("error concurrente inesperado: %v", err)
			}
		}()
	}
	grupo.Wait()
	if exitos.Load() != 1 || denegaciones.Load() != total-1 || registro.NumeroPropuestasParaPruebas() != 1 {
		t.Fatalf("unicidad no atomica: exitos=%d denegaciones=%d propuestas=%d", exitos.Load(), denegaciones.Load(), registro.NumeroPropuestasParaPruebas())
	}
}

func TestRegistroPropuestasLlamamientoDeniegaColisionesDePropuestaDecisionYRecibos(t *testing.T) {
	reloj := &relojMemoriaLlamamiento{instante: instanteMemoriaLlamamientoPrueba.Add(time.Second)}
	registro, _ := NuevoRegistroPropuestasLlamamiento(reloj, PerfilRegistroPropuestasSoloPruebas())
	primera, evidenciaPrimera := propuestaYEvidenciaMemoriaPrueba(t, "propuesta:memoria:0010", "instantanea:memoria:0010", "B", "decision:memoria:0010")
	if err := registro.GuardarPropuestaLlamamiento(context.Background(), primera, evidenciaPrimera); err != nil {
		t.Fatal(err)
	}

	mismaPropuestaOtroContenido, evidenciaOtra := propuestaYEvidenciaMemoriaPrueba(
		t, primera.PropuestaRef, "instantanea:memoria:0011", "C", "decision:memoria:0011",
	)
	if err := registro.GuardarPropuestaLlamamiento(context.Background(), mismaPropuestaOtroContenido, evidenciaOtra); !errors.Is(err, puertosbolsa.ErrPropuestaLlamamientoYaExiste) {
		t.Fatalf("referencia de propuesta reutilizada: %v", err)
	}

	otraPropuestaMismaDecision, _ := propuestaYEvidenciaMemoriaPrueba(
		t, "propuesta:memoria:0012", "instantanea:memoria:0012", "D", "decision:memoria:0010",
	)
	if err := registro.GuardarPropuestaLlamamiento(context.Background(), otraPropuestaMismaDecision, evidenciaPrimera); !errors.Is(err, puertosbolsa.ErrDecisionAutorizacionLlamamientoUsada) ||
		!errors.Is(err, puertosvec.ErrDecisionAutorizacionConsumida) {
		t.Fatalf("decision reutilizada: %v", err)
	}

	otraPropuestaMismosRecibos, evidenciaNueva := propuestaYEvidenciaMemoriaPrueba(
		t, "propuesta:memoria:0013", primera.InstantaneaRef, "B", "decision:memoria:0013",
	)
	if err := registro.GuardarPropuestaLlamamiento(context.Background(), otraPropuestaMismosRecibos, evidenciaNueva); !errors.Is(err, puertosbolsa.ErrReferenciaLlamamientoYaUtilizada) {
		t.Fatalf("recibos o instantanea reutilizados: %v", err)
	}
	if registro.NumeroPropuestasParaPruebas() != 1 {
		t.Fatalf("una colision dejo efecto parcial: %d", registro.NumeroPropuestasParaPruebas())
	}
}

func TestRegistroPropuestasLlamamientoDeniegaCancelacionEvidenciaCeroYPerfilProductivo(t *testing.T) {
	reloj := &relojMemoriaLlamamiento{instante: instanteMemoriaLlamamientoPrueba.Add(time.Second)}
	if _, err := NuevoRegistroPropuestasLlamamiento(reloj, PerfilUsoRegistroPropuestasMemoria{}); !errors.Is(err, puertosbolsa.ErrSolicitudPropuestaLlamamientoInvalida) {
		t.Fatalf("perfil no emitido admitido: %v", err)
	}
	var relojNulo *relojMemoriaLlamamiento
	if _, err := NuevoRegistroPropuestasLlamamiento(relojNulo, PerfilRegistroPropuestasSoloPruebas()); !errors.Is(err, puertosbolsa.ErrSolicitudPropuestaLlamamientoInvalida) {
		t.Fatalf("reloj tipado nulo admitido: %v", err)
	}
	registro, _ := NuevoRegistroPropuestasLlamamiento(reloj, PerfilRegistroPropuestasSoloPruebas())
	propuesta, _ := propuestaYEvidenciaMemoriaPrueba(t, "propuesta:memoria:0020", "instantanea:memoria:0020", "E", "decision:memoria:0020")
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if err := registro.GuardarPropuestaLlamamiento(ctx, propuesta, puertosvec.EvidenciaUsoDecisionAutorizacion{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("contexto cancelado no prevalecio: %v", err)
	}
	if err := registro.GuardarPropuestaLlamamiento(context.Background(), propuesta, puertosvec.EvidenciaUsoDecisionAutorizacion{}); !errors.Is(err, puertosvec.ErrEvidenciaUsoDecisionAutorizacionInvalida) {
		t.Fatalf("evidencia cero admitida: %v", err)
	}
	if registro.NumeroPropuestasParaPruebas() != 0 {
		t.Fatal("una denegacion dejo efecto")
	}
}

func TestRegistroPropuestasLlamamientoReintentosConcurrentesProducenUnSoloEfecto(t *testing.T) {
	reloj := &relojMemoriaLlamamiento{instante: instanteMemoriaLlamamientoPrueba.Add(time.Second)}
	registro, _ := NuevoRegistroPropuestasLlamamiento(reloj, PerfilRegistroPropuestasSoloPruebas())
	propuesta, evidencia := propuestaYEvidenciaMemoriaPrueba(t, "propuesta:memoria:0030", "instantanea:memoria:0030", "F", "decision:memoria:0030")
	const total = 64
	var grupo sync.WaitGroup
	errores := make(chan error, total)
	for indice := 0; indice < total; indice++ {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			errores <- registro.GuardarPropuestaLlamamiento(context.Background(), propuesta, evidencia)
		}()
	}
	grupo.Wait()
	close(errores)
	for err := range errores {
		if err != nil {
			t.Fatalf("reintento concurrente no idempotente: %v", err)
		}
	}
	if registro.NumeroPropuestasParaPruebas() != 1 {
		t.Fatalf("efectos concurrentes: %d", registro.NumeroPropuestasParaPruebas())
	}
}

func TestRegistroPropuestasLlamamientoUnaDecisionConcurrenteSoloConfirmaUnaPropuesta(t *testing.T) {
	reloj := &relojMemoriaLlamamiento{instante: instanteMemoriaLlamamientoPrueba.Add(time.Second)}
	registro, _ := NuevoRegistroPropuestasLlamamiento(reloj, PerfilRegistroPropuestasSoloPruebas())
	_, evidencia := propuestaYEvidenciaMemoriaPrueba(t, "propuesta:memoria:0040", "instantanea:memoria:0040", "G", "decision:memoria:0040")
	const total = 32
	var exitos atomic.Int32
	var grupo sync.WaitGroup
	for indice := 0; indice < total; indice++ {
		indice := indice
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			propuesta, _ := propuestaYEvidenciaMemoriaPruebaConcurrente(
				fmt.Sprintf("propuesta:memoria:04%02d", indice+1),
				fmt.Sprintf("instantanea:memoria:04%02d", indice+1),
				fmt.Sprintf("X%02d", indice),
			)
			if err := registro.GuardarPropuestaLlamamiento(context.Background(), propuesta, evidencia); err == nil {
				exitos.Add(1)
			} else if !errors.Is(err, puertosbolsa.ErrDecisionAutorizacionLlamamientoUsada) {
				t.Errorf("error concurrente inesperado: %v", err)
			}
		}()
	}
	grupo.Wait()
	if exitos.Load() != 1 || registro.NumeroPropuestasParaPruebas() != 1 {
		t.Fatalf("consumo no atomico: exitos=%d propuestas=%d", exitos.Load(), registro.NumeroPropuestasParaPruebas())
	}
}

func propuestaYEvidenciaMemoriaPrueba(
	t *testing.T,
	propuestaRef, instantaneaRef, sufijo, decisionRef string,
) (dominiobolsa.PropuestaLlamamiento, puertosvec.EvidenciaUsoDecisionAutorizacion) {
	t.Helper()
	propuesta, err := propuestaMemoriaPrueba(propuestaRef, instantaneaRef, sufijo)
	if err != nil {
		t.Fatalf("propuesta de prueba: %v", err)
	}
	evidencia, err := evidenciaMemoriaPrueba(propuesta, decisionRef)
	if err != nil {
		t.Fatalf("evidencia de prueba: %v", err)
	}
	return propuesta, evidencia
}

func propuestaYEvidenciaMemoriaPruebaConcurrente(
	propuestaRef, instantaneaRef, sufijo string,
) (dominiobolsa.PropuestaLlamamiento, error) {
	return propuestaMemoriaPrueba(propuestaRef, instantaneaRef, sufijo)
}

func propuestaMemoriaPrueba(
	propuestaRef, instantaneaRef, sufijo string,
) (dominiobolsa.PropuestaLlamamiento, error) {
	bolsa, err := dominiobolsa.NuevaBolsaConstituida(dominiobolsa.AltaBolsaConstituida{
		BolsaRef: "bolsa:memoria:0001", Version: 1, ProcesoRef: "proceso:memoria:0001",
		CategoriaRef: "categoria:memoria:0001", ListadoDefinitivoRef: "listado:memoria:0001",
		VersionListado: 1, HuellaListadoSHA256: huellaMemoriaLlamamiento('a'),
		ResolucionConstitucionRef: "resolucion:memoria:0001", HuellaResolucionSHA256: huellaMemoriaLlamamiento('b'),
		ConstituidaEn: instanteMemoriaLlamamientoPrueba.Add(-48 * time.Hour),
		VigenteDesde:  instanteMemoriaLlamamientoPrueba.Add(-24 * time.Hour),
	})
	if err != nil {
		return dominiobolsa.PropuestaLlamamiento{}, err
	}
	huellaBolsa, _ := bolsa.HuellaCanonicaSHA256()
	necesidad, err := dominiobolsa.NuevaNecesidadCobertura(dominiobolsa.AltaNecesidadCobertura{
		NecesidadRef: "necesidad:memoria:0001", Version: 1, BolsaRef: bolsa.BolsaRef, VersionBolsa: bolsa.Version,
		HuellaBolsaSHA256: huellaBolsa, CategoriaRef: bolsa.CategoriaRef, PuestoRef: "puesto:memoria:0001",
		UnidadRef: "unidad:memoria:0001", TipoCoberturaRef: "tipo:memoria:0001", NumeroPuestos: 1,
		InicioPrevisto: instanteMemoriaLlamamientoPrueba.Add(time.Hour),
		FinPrevisto:    instanteMemoriaLlamamientoPrueba.Add(30 * 24 * time.Hour),
		CreadaEn:       instanteMemoriaLlamamientoPrueba.Add(-time.Hour),
	})
	if err != nil {
		return dominiobolsa.PropuestaLlamamiento{}, err
	}
	politica, err := dominiobolsa.NuevaReferenciaPoliticaLlamamiento(dominiobolsa.ReferenciaPoliticaLlamamiento{
		PoliticaRef: "politica:memoria:0001", Clave: "politica_memoria_gobernada", Version: 1,
		HuellaSHA256: huellaMemoriaLlamamiento('c'), PublicadaEn: instanteMemoriaLlamamientoPrueba.Add(-48 * time.Hour),
		VigenteDesde: instanteMemoriaLlamamientoPrueba.Add(-24 * time.Hour),
	})
	if err != nil {
		return dominiobolsa.PropuestaLlamamiento{}, err
	}
	entradas := make([]dominiobolsa.EntradaOrdenBolsa, 2)
	for indice := range entradas {
		orden := indice + 1
		participacion, err := dominiobolsa.NuevaParticipacionBolsa(dominiobolsa.AltaParticipacionBolsa{
			ParticipacionRef: fmt.Sprintf("participacion:memoria:%d", orden), BolsaRef: bolsa.BolsaRef,
			SujetoRef: fmt.Sprintf("sujeto:memoria:%d", orden), Version: 1,
			AltaEn: instanteMemoriaLlamamientoPrueba.Add(-12 * time.Hour),
			Situaciones: []dominiobolsa.SituacionParticipacionBolsa{{
				Secuencia: 1, EstadoClave: "estado_memoria_gobernado", EstadoVersion: 1,
				HuellaEstadoSHA256: huellaMemoriaLlamamiento('d'), CausaClave: "causa_memoria_gobernada",
				CausaVersion: 1, HuellaCausaSHA256: huellaMemoriaLlamamiento('e'),
				DecisionRef: fmt.Sprintf("decision:situacion:memoria:%d", orden), HuellaDecisionSHA256: huellaMemoriaLlamamiento('f'),
				Desde: instanteMemoriaLlamamientoPrueba.Add(-12 * time.Hour),
			}},
		})
		if err != nil {
			return dominiobolsa.PropuestaLlamamiento{}, err
		}
		entradas[indice] = dominiobolsa.EntradaOrdenBolsa{Orden: uint64(orden), Participacion: participacion}
	}
	instantanea, err := dominiobolsa.NuevaInstantaneaOrdenBolsa(dominiobolsa.AltaInstantaneaOrdenBolsa{
		InstantaneaRef: instantaneaRef, Version: 1, Bolsa: bolsa,
		ReferidaEn: instanteMemoriaLlamamientoPrueba, GeneradaEn: instanteMemoriaLlamamientoPrueba, Entradas: entradas,
	})
	if err != nil {
		return dominiobolsa.PropuestaLlamamiento{}, err
	}
	evaluaciones := make([]dominiobolsa.EvaluacionParticipacionLlamamiento, 2)
	for indice := range evaluaciones {
		entrada := instantanea.Entradas[indice]
		situacion, _ := entrada.Participacion.SituacionVigenteEn(instantanea.ReferidaEn)
		huellaNecesidad, _ := necesidad.HuellaCanonicaSHA256()
		resultado := dominiobolsa.ResultadoNoElegible
		if indice == 1 {
			resultado = dominiobolsa.ResultadoElegible
		}
		marca := fmt.Sprintf("%s-%d", sufijo, indice+1)
		evaluaciones[indice] = dominiobolsa.EvaluacionParticipacionLlamamiento{
			ParticipacionRef: entrada.Participacion.ParticipacionRef, SujetoRef: entrada.Participacion.SujetoRef,
			Orden: entrada.Orden, SituacionSecuencia: situacion.Secuencia, EstadoClave: situacion.EstadoClave,
			EstadoVersion: situacion.EstadoVersion, HuellaEstadoSHA256: situacion.HuellaEstadoSHA256,
			NecesidadRef: necesidad.NecesidadRef, VersionNecesidad: necesidad.Version, HuellaNecesidadSHA256: huellaNecesidad,
			InstantaneaRef: instantanea.InstantaneaRef, VersionInstantanea: instantanea.Version,
			HuellaInstantaneaSHA256: instantanea.HuellaContenidoSHA256,
			PoliticaRef:             politica.PoliticaRef, VersionPolitica: politica.Version, HuellaPoliticaSHA256: politica.HuellaSHA256,
			Resultado: resultado, Motivos: []dominiobolsa.MotivoEvaluacionLlamamiento{{
				Clave: "resultado_memoria", ReglaRef: "regla:memoria:" + marca, VersionRegla: 1, HuellaReglaSHA256: huellaMemoriaLlamamiento('1'),
			}},
			EntradaEvaluacionRef: "recibo:entrada:" + marca, HuellaEntradaSHA256: huellaMemoriaLlamamiento('2'),
			ResultadoEvaluacionRef: "recibo:resultado:" + marca, HuellaResultadoSHA256: huellaMemoriaLlamamiento('3'),
			EvaluadaEn: instanteMemoriaLlamamientoPrueba,
		}
	}
	return dominiobolsa.ProponerPrimerLlamamiento(dominiobolsa.OrdenProponerPrimerLlamamiento{
		PropuestaRef: propuestaRef, Bolsa: bolsa, Necesidad: necesidad, Instantanea: instantanea,
		Politica: politica, Evaluaciones: evaluaciones, GeneradaEn: instanteMemoriaLlamamientoPrueba,
	})
}

func evidenciaMemoriaPrueba(
	propuesta dominiobolsa.PropuestaLlamamiento,
	decisionRef string,
) (puertosvec.EvidenciaUsoDecisionAutorizacion, error) {
	recurso := dominiovec.RecursoAutorizable{
		Referencia: propuesta.NecesidadRef, ModuloID: puertosbolsa.ModuloLlamamientos, Tipo: puertosbolsa.TipoRecursoNecesidad,
		Ambitos: map[string]string{"categoria_ref": "categoria:memoria:0001", "unidad_ref": "unidad:memoria:0001"},
	}
	huellaContexto, _ := recurso.HuellaContextoAutorizacionSHA256()
	politicaRef := "politica:seguridad:memoria:v1"
	huellas := map[string]string{politicaRef: huellaMemoriaLlamamiento('4')}
	huellaCatalogo, _ := dominiovec.HuellaEvidenciasCatalogoPoliticasAutorizacion([]string{politicaRef}, huellas)
	principalID := "per_" + strings.Repeat("p", 22)
	perfilActivoRef := "prf_" + strings.Repeat("r", 22)
	vinculo, err := vinculoActorMemoriaPrueba(principalID, perfilActivoRef)
	if err != nil {
		return puertosvec.EvidenciaUsoDecisionAutorizacion{}, err
	}
	decision := dominiovec.DecisionAutorizacion{
		DecisionRef: decisionRef, Concedida: true, Codigo: "concedida", PrincipalID: principalID,
		PerfilActivoRef: perfilActivoRef, Accion: puertosbolsa.AccionProponerLlamamiento,
		RecursoRef: propuesta.NecesidadRef, ModuloID: puertosbolsa.ModuloLlamamientos,
		TipoRecurso: puertosbolsa.TipoRecursoNecesidad, ContextoRecursoHuellaSHA256: huellaContexto,
		Finalidad: puertosbolsa.FinalidadProponerLlamamiento, CorrelacionRef: "correlacion:memoria:0001",
		VinculoAutenticacionActor: vinculo,
		AsignacionRef:             "asignacion:memoria:0001", AsignacionHuellaSHA256: huellaMemoriaLlamamiento('5'),
		VersionRolRef: "rol:memoria:v1", VersionRolHuellaSHA256: huellaMemoriaLlamamiento('6'),
		ControlVigenciaVersionRolRef: "rol:memoria:v1", ControlVigenciaVersionRolRevision: 1,
		ControlVigenciaVersionRolHuellaSHA256: huellaMemoriaLlamamiento('7'), RevisionCatalogoPoliticas: 1,
		CatalogoPoliticasHuellaSHA256: huellaCatalogo, PoliticasEvaluadasRefs: []string{politicaRef},
		PoliticasEvaluadasHuellasSHA256: huellas, PoliticasRefs: []string{politicaRef}, PoliticasHuellasSHA256: huellas,
		GarantiaMinima: dominiovec.AuthAssuranceHigh,
		EmitidaEn:      instanteMemoriaLlamamientoPrueba.Add(-time.Second),
		ValidaHasta:    instanteMemoriaLlamamientoPrueba.Add(2 * time.Minute),
	}
	return puertosvec.NuevaEvidenciaUsoDecisionAutorizacion(decision, propuesta.GeneradaEn)
}

func vinculoActorMemoriaPrueba(
	principalID string,
	perfilActivoRef string,
) (dominiovec.VinculoAutenticacionActorV1, error) {
	token := func(caracter string) string { return strings.Repeat(caracter, 22) }
	cuenta := dominiovec.CuentaAutenticadaContextoActor{
		CuentaRef: "cta_" + token("m"), Metodo: dominiovec.AuthMethodCertificate, Garantia: dominiovec.AuthAssuranceHigh,
	}
	instantanea := dominiovec.InstantaneaContextoActor{
		VinculoRef: "vca_" + token("v"), VinculoVersion: 1, CuentaRef: cuenta.CuentaRef,
		PersonaRef: principalID, PersonaVersion: 1, PerfilActivoRef: perfilActivoRef, PerfilVersion: 1,
		Estado:       dominiovec.EstadoVinculoContextoActorActivo,
		VigenteDesde: instanteMemoriaLlamamientoPrueba.Add(-24 * time.Hour),
		VigenteHasta: instanteMemoriaLlamamientoPrueba.Add(24 * time.Hour),
	}
	actor, err := dominiovec.NuevoContextoActor(cuenta, instantanea, instanteMemoriaLlamamientoPrueba.Add(-time.Minute))
	if err != nil {
		return dominiovec.VinculoAutenticacionActorV1{}, err
	}
	solicitud := dominiovec.SolicitudRevalidacionAutenticacionActorV1{
		AutenticacionRef: "aut_" + token("a"), SesionRef: "ses_" + token("s"),
	}
	autenticacion := dominiovec.AutenticacionRevalidadaV1{
		AutenticacionRef: solicitud.AutenticacionRef, AutenticacionHuellaSHA256: huellaMemoriaLlamamiento('a'),
		AsercionRef: "ase_" + token("e"), SesionRef: solicitud.SesionRef,
		ControlSesionRef: "cse_" + token("c"), ControlSesionRevision: 1,
		ControlSesionHuellaSHA256: huellaMemoriaLlamamiento('b'), CuentaRef: cuenta.CuentaRef,
		CuentaOrdinariaRef: cuenta.CuentaRef, Superficie: dominiovec.SuperficieAutenticacionExternaPersonalV1,
		MetodoObservado: cuenta.Metodo, GarantiaObservada: cuenta.Garantia,
		PoliticaGarantiaRef: "pga_" + token("g"), PoliticaGarantiaHuellaSHA256: huellaMemoriaLlamamiento('c'),
		SesionEmitidaEn:           instanteMemoriaLlamamientoPrueba.Add(-time.Hour),
		AutenticacionVerificadaEn: instanteMemoriaLlamamientoPrueba.Add(-2 * time.Hour),
		SesionRevalidadaEn:        instanteMemoriaLlamamientoPrueba.Add(-90 * time.Second),
		SesionValidaHasta:         instanteMemoriaLlamamientoPrueba.Add(time.Hour),
	}
	return dominiovec.CrearVinculoAutenticacionActorV1(
		context.Background(),
		revalidadorActorMemoriaFunc(func(_ context.Context, recibida dominiovec.SolicitudRevalidacionAutenticacionActorV1) (dominiovec.AutenticacionRevalidadaV1, error) {
			if recibida != solicitud {
				return dominiovec.AutenticacionRevalidadaV1{}, dominiovec.ErrAutenticacionRevalidadaInvalida
			}
			return autenticacion, nil
		}),
		solicitud,
		actor,
		instanteMemoriaLlamamientoPrueba,
	)
}

func huellaMemoriaLlamamiento(caracter byte) string {
	return strings.Repeat(string(caracter), 64)
}
