package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

type situacionBolsaAvisosPrueba struct {
	situacion ports.SituacionBolsaCobertura
	err       error
	llamadas  int
}

func (s *situacionBolsaAvisosPrueba) SituacionBolsaCobertura(context.Context, string) (ports.SituacionBolsaCobertura, error) {
	s.llamadas++
	return s.situacion, s.err
}

type reglasAvisosPrueba struct {
	reglas []reglas.Regla
	err    error
	errVen error
}

func (r *reglasAvisosPrueba) Reglas(context.Context) ([]reglas.Regla, error) { return r.reglas, r.err }

func (r *reglasAvisosPrueba) Vencimiento(_ context.Context, clave string, inicio time.Time, _ string) (reglas.Regla, reglas.Vencimiento, error) {
	if r.errVen != nil {
		return reglas.Regla{}, reglas.Vencimiento{}, r.errVen
	}
	for _, regla := range r.reglas {
		if regla.Clave != clave {
			continue
		}
		meses := regla.Cantidad
		if regla.Unidad == reglas.UnidadAnios {
			meses *= 12
		}
		fin := inicio.UTC().AddDate(0, meses, 0)
		return regla, reglas.Vencimiento{UltimoDia: fin.Format("2006-01-02"), VenceAntesDe: fin.AddDate(0, 0, 1)}, nil
	}
	return reglas.Regla{}, reglas.Vencimiento{}, reglas.ErrReglaNoEncontrada
}

type relojAvisosPrueba struct{ instante time.Time }

func (r relojAvisosPrueba) Ahora() time.Time { return r.instante }

func reglasAvisosEjemplo(umbral string) []reglas.Regla {
	agotamiento := reglas.Regla{Clave: reglas.BolsaAgotamiento, Etiqueta: "Agotamiento", Origen: reglas.OrigenReglamento, Articulo: "arts. 3.3 y 7.c", Atributos: map[string]string{}}
	if umbral != "" {
		agotamiento.Atributos[AtributoUmbralDisponibles] = umbral
	}
	return []reglas.Regla{
		{Clave: reglas.BolsaSAEDuracionMaxima, Etiqueta: "SAE", Unidad: reglas.UnidadMeses, Cantidad: 9, Origen: reglas.OrigenReglamento, Articulo: "art. 3.3"},
		{Clave: reglas.BolsaVigencia, Etiqueta: "Vigencia", Unidad: reglas.UnidadAnios, Cantidad: 5, Origen: reglas.OrigenReglamento, Articulo: "arts. 7.b y 3.4"},
		agotamiento,
	}
}

func periodoAvisosPrueba(inicio, fin string) domain.PeriodoPrevisto {
	desde, _ := time.Parse("2006-01-02", inicio)
	hasta, _ := time.Parse("2006-01-02", fin)
	return domain.PeriodoPrevisto{Inicio: desde, Fin: hasta}
}

func evaluadorAvisosPrueba(t *testing.T, situacion *situacionBolsaAvisosPrueba, resolutor *reglasAvisosPrueba) *EvaluadorAvisosViaCobertura {
	t.Helper()
	evaluador, err := NuevoEvaluadorAvisosViaCobertura(situacion, resolutor, relojAvisosPrueba{time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	return evaluador
}

func clavesAvisos(resultado ResultadoAvisosViaCobertura) []ClaveAvisoViaCobertura {
	claves := make([]ClaveAvisoViaCobertura, 0, len(resultado.Avisos))
	for _, aviso := range resultado.Avisos {
		claves = append(claves, aviso.Clave)
	}
	return claves
}

func TestAvisosViaCoberturaBolsaAgotadaProponeSAEYConvocatoria(t *testing.T) {
	situacion := &situacionBolsaAvisosPrueba{situacion: ports.SituacionBolsaCobertura{
		Existe: true, BolsaRef: "bolsa:1", ConstituidaEn: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC), Integrantes: 4,
	}}
	evaluador := evaluadorAvisosPrueba(t, situacion, &reglasAvisosPrueba{reglas: reglasAvisosEjemplo("0")})
	resultado, evaluado := evaluador.Evaluar(context.Background(), "categoria:rpt:auxiliar", periodoAvisosPrueba("2026-10-01", "2027-12-31"))
	if !evaluado || resultado.Estado != EstadoAvisosEvaluados {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
	claves := clavesAvisos(resultado)
	if len(claves) != 3 || claves[0] != AvisoBolsaAgotadaProvisionalmente || claves[1] != AvisoPropuestaOfertaSAE || claves[2] != AvisoNuevaConvocatoria {
		t.Fatalf("avisos inesperados: %v", claves)
	}
	sae := resultado.Avisos[1]
	if sae.DuracionMaximaMeses != 9 || sae.FinMaximo != "2027-07-01" || !sae.ExcedeDuracion || len(sae.Reglas) != 2 {
		t.Fatalf("propuesta SAE inesperada: %+v", sae)
	}
	convocatoria := resultado.Avisos[2]
	if len(convocatoria.Motivos) != 1 || convocatoria.Motivos[0] != MotivoBolsaAgotada || convocatoria.VigenciaHasta != "2030-03-01" {
		t.Fatalf("aviso de convocatoria inesperado: %+v", convocatoria)
	}
}

func TestAvisosViaCoberturaUmbralDelCatalogo(t *testing.T) {
	situacion := &situacionBolsaAvisosPrueba{situacion: ports.SituacionBolsaCobertura{
		Existe: true, BolsaRef: "bolsa:1", ConstituidaEn: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC), Integrantes: 10, Disponibles: 2,
	}}
	sinAviso := evaluadorAvisosPrueba(t, situacion, &reglasAvisosPrueba{reglas: reglasAvisosEjemplo("0")})
	if resultado, _ := sinAviso.Evaluar(context.Background(), "categoria:rpt:auxiliar", periodoAvisosPrueba("2026-10-01", "2027-01-31")); len(resultado.Avisos) != 0 {
		t.Fatalf("con disponibles no debe haber avisos: %v", clavesAvisos(resultado))
	}
	conAviso := evaluadorAvisosPrueba(t, situacion, &reglasAvisosPrueba{reglas: reglasAvisosEjemplo("2")})
	resultado, _ := conAviso.Evaluar(context.Background(), "categoria:rpt:auxiliar", periodoAvisosPrueba("2026-10-01", "2027-01-31"))
	if len(resultado.Avisos) != 3 || resultado.Avisos[0].Umbral != 2 || resultado.Avisos[1].ExcedeDuracion {
		t.Fatalf("el umbral 2 del catálogo debe agotar la bolsa: %+v", resultado.Avisos)
	}
}

func TestAvisosViaCoberturaVigenciaSuperada(t *testing.T) {
	situacion := &situacionBolsaAvisosPrueba{situacion: ports.SituacionBolsaCobertura{
		Existe: true, BolsaRef: "bolsa:1", ConstituidaEn: time.Date(2021, 1, 10, 0, 0, 0, 0, time.UTC), Integrantes: 10, Disponibles: 6,
	}}
	evaluador := evaluadorAvisosPrueba(t, situacion, &reglasAvisosPrueba{reglas: reglasAvisosEjemplo("0")})
	resultado, _ := evaluador.Evaluar(context.Background(), "categoria:rpt:auxiliar", periodoAvisosPrueba("2026-10-01", "2027-01-31"))
	if len(resultado.Avisos) != 1 || resultado.Avisos[0].Clave != AvisoNuevaConvocatoria ||
		resultado.Avisos[0].Motivos[0] != MotivoVigenciaSuperada || resultado.Avisos[0].Reglas[0].Clave != reglas.BolsaVigencia {
		t.Fatalf("aviso de vigencia inesperado: %+v", resultado.Avisos)
	}
}

func TestAvisosViaCoberturaSinCatalogoMantieneConducta(t *testing.T) {
	situacion := &situacionBolsaAvisosPrueba{}
	evaluador := evaluadorAvisosPrueba(t, situacion, &reglasAvisosPrueba{err: reglas.ErrReglasNoConfiguradas})
	if _, evaluado := evaluador.Evaluar(context.Background(), "categoria:rpt:auxiliar", periodoAvisosPrueba("2026-10-01", "2027-01-31")); evaluado || situacion.llamadas != 0 {
		t.Fatal("sin catálogo no se evalúa ni se consulta Bolsa")
	}
	vacio := evaluadorAvisosPrueba(t, situacion, &reglasAvisosPrueba{})
	if _, evaluado := vacio.Evaluar(context.Background(), "categoria:rpt:auxiliar", periodoAvisosPrueba("2026-10-01", "2027-01-31")); evaluado {
		t.Fatal("un catálogo sin estas reglas no activa avisos")
	}
}

func TestAvisosViaCoberturaIndisponibilidadNoEsResultado(t *testing.T) {
	periodo := periodoAvisosPrueba("2026-10-01", "2027-01-31")
	casos := map[string]*EvaluadorAvisosViaCobertura{
		"bolsa":    evaluadorAvisosPrueba(t, &situacionBolsaAvisosPrueba{err: errors.New("caida")}, &reglasAvisosPrueba{reglas: reglasAvisosEjemplo("0")}),
		"catalogo": evaluadorAvisosPrueba(t, &situacionBolsaAvisosPrueba{}, &reglasAvisosPrueba{err: reglas.ErrReglasNoDisponibles}),
		"umbral": evaluadorAvisosPrueba(t, &situacionBolsaAvisosPrueba{situacion: ports.SituacionBolsaCobertura{Existe: true, BolsaRef: "bolsa:1", ConstituidaEn: time.Now(), Integrantes: 1}},
			&reglasAvisosPrueba{reglas: reglasAvisosEjemplo("-1")}),
		"incoherente": evaluadorAvisosPrueba(t, &situacionBolsaAvisosPrueba{situacion: ports.SituacionBolsaCobertura{Existe: true, BolsaRef: "bolsa:1", ConstituidaEn: time.Now(), Integrantes: 1, Disponibles: 2}},
			&reglasAvisosPrueba{reglas: reglasAvisosEjemplo("0")}),
		"calculo": evaluadorAvisosPrueba(t, &situacionBolsaAvisosPrueba{situacion: ports.SituacionBolsaCobertura{Existe: true, BolsaRef: "bolsa:1", ConstituidaEn: time.Now(), Integrantes: 1, Disponibles: 1}},
			&reglasAvisosPrueba{reglas: reglasAvisosEjemplo("0"), errVen: reglas.ErrCalculoNoDisponible}),
	}
	for nombre, evaluador := range casos {
		resultado, evaluado := evaluador.Evaluar(context.Background(), "categoria:rpt:auxiliar", periodo)
		if !evaluado || resultado.Estado != EstadoAvisosNoDisponible || len(resultado.Avisos) != 0 {
			t.Fatalf("%s: la indisponibilidad debe mostrarse como tal: %+v", nombre, resultado)
		}
	}
	sinBolsa := evaluadorAvisosPrueba(t, &situacionBolsaAvisosPrueba{}, &reglasAvisosPrueba{reglas: reglasAvisosEjemplo("0")})
	if resultado, _ := sinBolsa.Evaluar(context.Background(), "categoria:rpt:auxiliar", periodo); resultado.Estado != EstadoAvisosSinBolsa {
		t.Fatalf("sin bolsa constituida: %+v", resultado)
	}
}

func TestAvisosViaCoberturaConfiguracionUnica(t *testing.T) {
	servicio := &ServicioPresentacionPropuestaCobertura{}
	evaluador := evaluadorAvisosPrueba(t, &situacionBolsaAvisosPrueba{}, &reglasAvisosPrueba{})
	if servicio.ConfigurarAvisosVia(nil) == nil || servicio.ConfigurarAvisosVia(evaluador) != nil || servicio.ConfigurarAvisosVia(evaluador) == nil {
		t.Fatal("el evaluador se configura una sola vez y nunca nulo")
	}
	if _, err := NuevoEvaluadorAvisosViaCobertura(nil, &reglasAvisosPrueba{}, relojAvisosPrueba{}); err == nil {
		t.Fatal("sin consulta de Bolsa el evaluador no se construye")
	}
	original := &ResultadoAvisosViaCobertura{Avisos: []AvisoViaCobertura{{Motivos: []string{"a"}, Reglas: []ProcedenciaReglaAviso{{Clave: "x"}}}}}
	copia := copiarAvisosViaCobertura(original)
	copia.Avisos[0].Motivos[0], copia.Avisos[0].Reglas[0].Clave = "b", "y"
	if original.Avisos[0].Motivos[0] != "a" || original.Avisos[0].Reglas[0].Clave != "x" {
		t.Fatal("la copia comparte listas mutables")
	}
}

func TestProponerCoberturaIncluyeAvisosViaSinAlterarLaPropuesta(t *testing.T) {
	sinAvisos := nuevoEscenarioPresentacionCobertura(t, viasPresentacionCoberturaPrueba(2))
	base, err := sinAvisos.servicio.Proponer(context.Background(), sinAvisos.solicitud)
	if err != nil || base.AvisosVia != nil {
		t.Fatalf("sin evaluador la propuesta es la de hoy: %+v %v", base.AvisosVia, err)
	}
	escenario := nuevoEscenarioPresentacionCobertura(t, viasPresentacionCoberturaPrueba(2))
	situacion := &situacionBolsaAvisosPrueba{err: errors.New("bolsa caida")}
	if err := escenario.servicio.ConfigurarAvisosVia(evaluadorAvisosPrueba(t, situacion, &reglasAvisosPrueba{reglas: reglasAvisosEjemplo("0")})); err != nil {
		t.Fatal(err)
	}
	presentacion, err := escenario.servicio.Proponer(context.Background(), escenario.solicitud)
	if err != nil || presentacion.AvisosVia == nil || presentacion.AvisosVia.Estado != EstadoAvisosNoDisponible || situacion.llamadas != 1 {
		t.Fatalf("los avisos no deben romper la propuesta: %+v %v", presentacion.AvisosVia, err)
	}
	if presentacion.IdentidadSemantica.HuellaSHA256 != base.IdentidadSemantica.HuellaSHA256 || presentacion.ViaRecomendada != base.ViaRecomendada {
		t.Fatal("los avisos alteraron la propuesta firmada")
	}
	resultado, err := nuevaResultadoPropuestaCoberturaParaAdaptador(presentacion)
	if err != nil {
		t.Fatal(err)
	}
	if datos, ok := resultado.DatosParaAdaptador(); !ok || datos.AvisosVia == nil || datos.AvisosVia == presentacion.AvisosVia {
		t.Fatal("el adaptador debe recibir una copia de los avisos")
	}
}
