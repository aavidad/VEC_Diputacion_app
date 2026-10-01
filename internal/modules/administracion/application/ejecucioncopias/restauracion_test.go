package ejecucioncopias

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	ejadapter "vec-diputacion-granada/internal/modules/administracion/adapters/ejecucioncopias"
	cs07 "vec-diputacion-granada/internal/modules/administracion/adapters/registrocopias"
	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
	puertos "vec-diputacion-granada/internal/modules/administracion/ports/ejecucioncopias"
)

const huellaPrueba = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const preimagenPrueba = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

type fixtureRestauracion struct {
	Manifiesto copias.Manifiesto `json:"manifiesto"`
	Destino    copias.Inventario `json:"destino"`
	Politica   copias.Politica   `json:"politica"`
}

func cargaFixtureRestauracion(t *testing.T) fixtureRestauracion {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("../../../../../cmd/vec-copias-comprobar/testdata", "compatible.json"))
	if err != nil {
		t.Fatal(err)
	}
	var f fixtureRestauracion
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatal(err)
	}
	return f
}

func propuestaPrueba(t *testing.T) (puertos.Propuesta, puertos.Lectura, puertos.Conjunto, puertos.Captura) {
	t.Helper()
	f := cargaFixtureRestauracion(t)
	lectura := puertos.Lectura{Esperado: f.Destino, Observado: f.Destino, Politica: f.Politica, VersionRef: "version:destino", PreimagenSHA256: preimagenPrueba}
	p := puertos.Propuesta{Peticion: puertos.Peticion{OperacionRef: "operacion:restauracion", ActorRef: "actor:operador", OrigenRef: "origen:conjunto", DestinoRef: "destino:sintetico", MotivoRef: "motivo:sintetico", ConjuntoRef: "conjunto:objetivo", PoliticaRef: f.Politica.Ref}, HuellaPropuesta: huellaPrueba, PreimagenSHA256: lectura.PreimagenSHA256, ConjuntoPreviaRef: "conjunto:previo"}
	objetivo := puertos.Conjunto{Ref: p.ConjuntoRef, IndiceAutenticadoRef: "indice:objetivo", ManifiestoSHA256: huellaPrueba, ManifiestoBaseSHA256: huellaPrueba, EjecucionVerificacionRef: "indice:objetivo", Manifiesto: f.Manifiesto, Origen: f.Manifiesto.Verificacion.Fisica}
	objetivo.Manifiesto.ConjuntoRef = objetivo.Ref
	objetivo.Manifiesto.OperacionRef = "operacion:captura-objetivo"
	objetivo.Manifiesto.PoliticaRef = p.PoliticaRef
	objetivo.Manifiesto.Inventario = f.Manifiesto.Inventario
	objetivo.Manifiesto.InventarioSHA256 = copias.HuellaInventario(objetivo.Manifiesto.Inventario)
	previa := objetivo.Manifiesto
	previa.ConjuntoRef = p.ConjuntoPreviaRef
	previa.OperacionRef = p.OperacionRef
	previa.SolicitanteRef = p.ActorRef
	previa.MotivoRef = p.MotivoRef
	previa.Inventario = lectura.Observado
	previa.InventarioSHA256 = copias.HuellaInventario(previa.Inventario)
	previa.Verificacion = copias.Verificacion{Estado: "pendiente_verificacion"}
	return p, lectura, objetivo, puertos.Captura{Manifiesto: previa, Origen: f.Manifiesto.Verificacion.Fisica}
}

type inventarioPrueba struct{ lectura puertos.Lectura }

func (i inventarioPrueba) LeerActual(context.Context, string) (puertos.Lectura, error) {
	return i.lectura, nil
}

type inventarioSecuencia struct {
	primera, segunda puertos.Lectura
	consultas        int
}

func (i *inventarioSecuencia) LeerActual(context.Context, string) (puertos.Lectura, error) {
	i.consultas++
	if i.consultas == 1 {
		return i.primera, nil
	}
	return i.segunda, nil
}

type autorizadorPrueba struct {
	aprobacion puertos.Aprobacion
}

func (a autorizadorPrueba) Autorizar(_ context.Context, p puertos.Peticion, accion string) (puertos.Concesion, error) {
	recurso := p.DestinoRef
	if accion == "copiar" {
		recurso = p.OrigenRef
	}
	return puertos.Concesion{ActorRef: p.ActorRef, Accion: accion, RecursoRef: recurso, DecisionRef: "decision:vigente", Vence: time.Now().Add(time.Hour)}, nil
}

func (a autorizadorPrueba) Aprobar(context.Context, puertos.Propuesta) (puertos.Aprobacion, error) {
	return a.aprobacion, nil
}

type registroPrueba struct {
	op               puertos.Operacion
	copias           map[string]puertos.Operacion
	reservaExistente bool
	eventos          *[]string
}

func (r *registroPrueba) Reservar(_ context.Context, p puertos.Peticion) (puertos.Operacion, error) {
	if !r.reservaExistente {
		r.reservarCopia(p)
	}
	*r.eventos = append(*r.eventos, "reservar_copia")
	return r.op, nil
}

func (r *registroPrueba) reservarCopia(p puertos.Peticion) {
	r.op = puertos.Operacion{Ref: p.OperacionRef, VersionRef: "v1", Estado: "solicitada", ConjuntoRef: p.ConjuntoRef, PoliticaRef: p.PoliticaRef}
}
func (r *registroPrueba) ReservarRestauracion(_ context.Context, p puertos.Propuesta) (puertos.Operacion, error) {
	r.op = puertos.Operacion{Ref: p.OperacionRef, VersionRef: "v1", Estado: "solicitada", ConjuntoRef: p.ConjuntoRef, ConjuntoPreviaPlaneadaRef: p.ConjuntoPreviaRef, PoliticaRef: p.PoliticaRef, PreimagenSHA256: p.PreimagenSHA256, HuellaPropuesta: p.HuellaPropuesta}
	*r.eventos = append(*r.eventos, "reservar")
	return r.op, nil
}
func (r *registroPrueba) Anotar(_ context.Context, _ string, estado, detalle string) error {
	*r.eventos = append(*r.eventos, "anotar:"+estado)
	switch estado {
	case "copia_previa_verificada":
		r.op.CopiaPreviaRef = detalle
	case "plan_preparado":
		r.op.PlanRef = detalle
	case "valida":
		r.op.Estado = estado
		r.op.IndiceAutenticadoRef = detalle
	case "sustitucion_iniciada", "instalado_pendiente_conciliacion", "pendiente_conciliacion", "revertida":
		r.op.Estado = estado
	}
	return nil
}
func (r *registroPrueba) AplicarCopia(context.Context, puertos.EventoCopia) error { return nil }
func (r *registroPrueba) ConciliarRestauracion(ctx context.Context, o puertos.ObservacionRestauracion) error {
	estado := "revertida"
	if o.InstaladoRef == r.op.ConjuntoRef {
		estado = "instalado_pendiente_conciliacion"
	}
	return r.Anotar(ctx, o.OperacionRef, estado, o.InstaladoRef)
}
func (r *registroPrueba) CAS(_ context.Context, _ string, version, preimagen, estado string) (puertos.Operacion, error) {
	*r.eventos = append(*r.eventos, "cas")
	if version != r.op.VersionRef || preimagen != r.op.PreimagenSHA256 {
		return puertos.Operacion{}, errors.New("cas_invalido")
	}
	r.op.Estado = estado
	r.op.VersionRef = "v2"
	return r.op, nil
}
func (r *registroPrueba) Leer(_ context.Context, ref string) (puertos.Operacion, error) {
	if copia, ok := r.copias[ref]; ok {
		return copia, nil
	}
	return r.op, nil
}

type exclusionPrueba struct {
	previa    puertos.Captura
	preimagen string
	eventos   *[]string
}

func (x *exclusionPrueba) CapturarPrevia(_ context.Context, p puertos.Peticion, _ puertos.Lectura) (puertos.Captura, error) {
	*x.eventos = append(*x.eventos, "capturar_previa")
	*x.eventos = append(*x.eventos, "capturar_previa_ref:"+p.ConjuntoRef)
	return x.previa, nil
}
func (x *exclusionPrueba) PreimagenActual(context.Context) (string, error) {
	*x.eventos = append(*x.eventos, "preimagen")
	return x.preimagen, nil
}
func (x *exclusionPrueba) Cerrar(context.Context) error {
	*x.eventos = append(*x.eventos, "cerrar")
	return nil
}
func (x *exclusionPrueba) ConservarMantenimiento(context.Context) error {
	*x.eventos = append(*x.eventos, "conservar_mantenimiento")
	return nil
}

type ventanaPrueba struct {
	exclusion *exclusionPrueba
	captura   puertos.Captura
	eventos   *[]string
}

func (v ventanaPrueba) Capturar(context.Context, puertos.Peticion, puertos.Lectura) (puertos.Captura, error) {
	*v.eventos = append(*v.eventos, "capturar")
	if v.captura.Manifiesto.ConjuntoRef == "" {
		return puertos.Captura{}, errors.New("no_esperado")
	}
	return v.captura, nil
}
func (v ventanaPrueba) AbrirRestauracion(context.Context, puertos.Peticion) (puertos.Exclusion, error) {
	*v.eventos = append(*v.eventos, "abrir_exclusion")
	return v.exclusion, nil
}

type destinoPrueba struct {
	conjuntos map[string]puertos.Conjunto
	eventos   *[]string
}

func (d *destinoPrueba) Publicar(_ context.Context, captura puertos.Captura) (puertos.Conjunto, error) {
	*d.eventos = append(*d.eventos, "publicar_previa")
	c := puertos.Conjunto{Ref: captura.Manifiesto.ConjuntoRef, IndiceAutenticadoRef: "indice:previo", ManifiestoSHA256: huellaPrueba, ManifiestoBaseSHA256: huellaPrueba, EjecucionVerificacionRef: "indice:previo", Manifiesto: captura.Manifiesto, Origen: captura.Origen}
	d.conjuntos[c.Ref] = c
	return c, nil
}
func (d *destinoPrueba) CerrarVerificacion(_ context.Context, c puertos.Conjunto, v copias.Verificacion) (puertos.Conjunto, error) {
	*d.eventos = append(*d.eventos, "cerrar_verificacion")
	c.Manifiesto.Verificacion = v
	d.conjuntos[c.Ref] = c
	return c, nil
}
func (d *destinoPrueba) Recuperar(_ context.Context, ref string) (puertos.Conjunto, error) {
	*d.eventos = append(*d.eventos, "recuperar:"+ref)
	c, ok := d.conjuntos[ref]
	if !ok {
		return puertos.Conjunto{}, errors.New("conjunto_no_encontrado")
	}
	return c, nil
}

type ensayadorPrueba struct {
	evidencia        copias.Evidencia
	eventos          *[]string
	difiereContenido bool
}

func (e ensayadorPrueba) Ensayar(_ context.Context, _ puertos.Conjunto, modo puertos.ModoEnsayo) (puertos.Ensayo, error) {
	*e.eventos = append(*e.eventos, "ensayar:"+string(modo))
	evidencia := e.evidencia
	if e.difiereContenido && modo == puertos.Logico {
		evidencia.ContenidoSHA256 = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	}
	return puertos.Ensayo{Modo: modo, Evidencia: evidencia, VerificadorVersion: "verificador:sintetico:1", ArranqueRef: evidencia.ArranqueRef}, nil
}

type plataformaPrueba struct {
	instalado     string
	sustituciones int
	eventos       *[]string
}

func (p *plataformaPrueba) Preparar(_ context.Context, c puertos.Conjunto) (puertos.Preparado, error) {
	*p.eventos = append(*p.eventos, "preparar")
	return puertos.Preparado{ConjuntoRef: c.Ref, PlanRef: "plan:sintetico"}, nil
}
func (p *plataformaPrueba) Sustituir(_ context.Context, preparado puertos.Preparado, _ string) error {
	*p.eventos = append(*p.eventos, "sustituir")
	p.sustituciones++
	p.instalado = preparado.ConjuntoRef
	return nil
}
func (p *plataformaPrueba) ArrancarAislado(context.Context, puertos.Preparado) (string, error) {
	*p.eventos = append(*p.eventos, "arrancar")
	return "arranque:aislado", nil
}
func (p *plataformaPrueba) Revertir(_ context.Context, c puertos.Conjunto, _ string) error {
	p.instalado = c.Ref
	return nil
}
func (p *plataformaPrueba) IdentificarInstalado(context.Context, string) (string, error) {
	*p.eventos = append(*p.eventos, "identificar")
	return p.instalado, nil
}

func servicioPrueba(t *testing.T, p puertos.Propuesta, lectura puertos.Lectura, objetivo puertos.Conjunto, previa puertos.Captura, aprobacion puertos.Aprobacion, eventos *[]string) (*Servicio, *registroPrueba, *plataformaPrueba) {
	t.Helper()
	destino := &destinoPrueba{conjuntos: map[string]puertos.Conjunto{objetivo.Ref: objetivo}, eventos: eventos}
	registro := &registroPrueba{eventos: eventos, copias: map[string]puertos.Operacion{
		objetivo.Manifiesto.OperacionRef: {Ref: objetivo.Manifiesto.OperacionRef, Estado: "valida", ConjuntoRef: objetivo.Ref, PoliticaRef: objetivo.Manifiesto.PoliticaRef, IndiceAutenticadoRef: objetivo.IndiceAutenticadoRef},
	}}
	plataforma := &plataformaPrueba{eventos: eventos}
	exclusion := &exclusionPrueba{previa: previa, preimagen: p.PreimagenSHA256, eventos: eventos}
	s, err := Nuevo(Dependencias{Inventario: inventarioPrueba{lectura}, Autorizador: autorizadorPrueba{aprobacion}, Registro: registro, Ventana: ventanaPrueba{exclusion: exclusion, eventos: eventos}, Destino: destino, Ensayador: ensayadorPrueba{evidencia: previa.Origen, eventos: eventos}, Plataforma: plataforma})
	if err != nil {
		t.Fatal(err)
	}
	return s, registro, plataforma
}

func aprobacionPrueba(p puertos.Propuesta) puertos.Aprobacion {
	return puertos.Aprobacion{ProponentePersonaRef: "persona:proponente", AprobadorPersonaRef: "persona:aprobadora", HuellaPropuesta: p.HuellaPropuesta, PreimagenSHA256: p.PreimagenSHA256, DecisionRef: "decision:doble-control", Vence: time.Now().Add(time.Hour)}
}

func TestEnsayosValidosRechazaContenidoDistintoConMismoRecuento(t *testing.T) {
	p, _, _, previa := propuestaPrueba(t)
	fisico := puertos.Ensayo{Modo: puertos.Fisico, Evidencia: previa.Origen, VerificadorVersion: "v1", ArranqueRef: previa.Origen.ArranqueRef}
	logico := fisico
	logico.Modo = puertos.Logico
	logico.Evidencia.ContenidoSHA256 = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	origen := previa.Origen
	origen.ArranqueRef = ""
	controlLogico := fisico
	controlLogico.Modo = puertos.Logico
	if !ensayosValidos(origen, fisico, controlLogico) {
		t.Fatal("exigió al origen un arranque reservado a los ensayos")
	}
	sinArranque := controlLogico
	sinArranque.ArranqueRef, sinArranque.Evidencia.ArranqueRef = "", ""
	if ensayosValidos(origen, fisico, sinArranque) {
		t.Fatal("aceptó ensayo sin arranque comprobado")
	}
	if !ensayosValidos(previa.Origen, fisico, puertos.Ensayo{Modo: puertos.Logico, Evidencia: previa.Origen, VerificadorVersion: "v1", ArranqueRef: previa.Origen.ArranqueRef}) {
		t.Fatal("el caso de control debe ser válido")
	}
	if ensayosValidos(previa.Origen, fisico, logico) {
		t.Fatalf("aceptó contenido distinto con recuento %s", p.PreimagenSHA256)
	}
}

func TestCopiarMarcaNoValidaSiContenidoDifiereConMismoRecuento(t *testing.T) {
	p, lectura, objetivo, captura := propuestaPrueba(t)
	captura.Manifiesto.ConjuntoRef = p.ConjuntoRef
	captura.Manifiesto.OperacionRef = p.OperacionRef
	captura.Manifiesto.Verificacion = copias.Verificacion{Estado: "pendiente_verificacion"}
	var eventos []string
	registro := &registroPrueba{eventos: &eventos}
	destino := &destinoPrueba{conjuntos: map[string]puertos.Conjunto{objetivo.Ref: objetivo}, eventos: &eventos}
	ensayador := ensayadorPrueba{evidencia: captura.Origen, eventos: &eventos, difiereContenido: true}
	s, err := NuevoCopia(Dependencias{
		Inventario:  inventarioPrueba{lectura},
		Autorizador: autorizadorPrueba{},
		Registro:    registro,
		Ventana:     ventanaPrueba{captura: captura, eventos: &eventos},
		Destino:     destino,
		Ensayador:   ensayador,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Copiar(context.Background(), p.Peticion); !errors.Is(err, ErrBloqueada) {
		t.Fatalf("error=%v", err)
	}
	for _, evento := range eventos {
		if evento == "anotar:no_valida" {
			return
		}
	}
	t.Fatalf("no registró no_valida: %v", eventos)
}

func TestCopiarRecuperaConjuntoPublicadoSinNuevaCaptura(t *testing.T) {
	for _, tt := range []struct {
		estado    string
		pendiente bool
	}{
		{estado: "capturando", pendiente: true},
		{estado: "capturada", pendiente: true},
		{estado: "verificando", pendiente: true},
		{estado: "verificada_declarada"},
	} {
		t.Run(tt.estado, func(t *testing.T) {
			p, lectura, objetivo, _ := propuestaPrueba(t)
			objetivo.Manifiesto.OperacionRef = p.OperacionRef
			objetivo.Manifiesto.SolicitanteRef = p.ActorRef
			objetivo.Manifiesto.MotivoRef = p.MotivoRef
			if tt.pendiente {
				objetivo.Manifiesto.Verificacion = copias.Verificacion{Estado: "pendiente_verificacion"}
			}
			var eventos []string
			registro := &registroPrueba{
				op:               puertos.Operacion{Ref: p.OperacionRef, VersionRef: "v1", Estado: tt.estado, ConjuntoRef: objetivo.Ref, PoliticaRef: p.PoliticaRef, IndiceAutenticadoRef: objetivo.IndiceAutenticadoRef},
				reservaExistente: true,
				eventos:          &eventos,
			}
			destino := &destinoPrueba{conjuntos: map[string]puertos.Conjunto{objetivo.Ref: objetivo}, eventos: &eventos}
			s, err := NuevoCopia(Dependencias{
				Inventario:  inventarioPrueba{lectura},
				Autorizador: autorizadorPrueba{},
				Registro:    registro,
				Ventana:     ventanaPrueba{eventos: &eventos},
				Destino:     destino,
				Ensayador:   ensayadorPrueba{evidencia: objetivo.Origen, eventos: &eventos},
			})
			if err != nil {
				t.Fatal(err)
			}
			r, err := s.Copiar(context.Background(), p.Peticion)
			if err != nil {
				t.Fatal(err)
			}
			if r.ConjuntoRef != objetivo.Ref || r.Estado != "valida" || r.IndiceAutenticadoRef != objetivo.IndiceAutenticadoRef {
				t.Fatalf("recibo de recuperación: %+v", r)
			}
			contar := func(evento string) int {
				n := 0
				for _, actual := range eventos {
					if actual == evento {
						n++
					}
				}
				return n
			}
			if contar("capturar") != 0 || contar("publicar_previa") != 0 {
				t.Fatalf("repitió captura/publicación: %v", eventos)
			}
			if tt.pendiente {
				if contar("ensayar:fisico") != 1 || contar("ensayar:logico") != 1 || contar("cerrar_verificacion") != 1 {
					t.Fatalf("no completó el ensayo pendiente: %v", eventos)
				}
				pos := func(evento string) int {
					for i, actual := range eventos {
						if actual == evento {
							return i
						}
					}
					return -1
				}
				if pos("cerrar_verificacion") < 0 || pos("anotar:valida") < 0 || pos("cerrar_verificacion") > pos("anotar:valida") {
					t.Fatalf("registró el resultado antes de sellarlo: %v", eventos)
				}
			} else if contar("ensayar:fisico") != 0 || contar("ensayar:logico") != 0 || contar("cerrar_verificacion") != 0 {
				t.Fatalf("repitió un conjunto ya sellado: %v", eventos)
			}
		})
	}
}

func TestCopiarRecuperacionConIndiceCorruptoQuedaEnConciliacion(t *testing.T) {
	p, lectura, objetivo, _ := propuestaPrueba(t)
	objetivo.Manifiesto.Verificacion = copias.Verificacion{Estado: "pendiente_verificacion"}
	objetivo.ManifiestoBaseSHA256 = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	var eventos []string
	registro := &registroPrueba{
		op:               puertos.Operacion{Ref: p.OperacionRef, VersionRef: "v1", Estado: "capturada", ConjuntoRef: objetivo.Ref, PoliticaRef: p.PoliticaRef, IndiceAutenticadoRef: objetivo.IndiceAutenticadoRef},
		reservaExistente: true,
		eventos:          &eventos,
	}
	s, err := NuevoCopia(Dependencias{
		Inventario:  inventarioPrueba{lectura},
		Autorizador: autorizadorPrueba{},
		Registro:    registro,
		Ventana:     ventanaPrueba{eventos: &eventos},
		Destino:     &destinoPrueba{conjuntos: map[string]puertos.Conjunto{objetivo.Ref: objetivo}, eventos: &eventos},
		Ensayador:   ensayadorPrueba{evidencia: objetivo.Origen, eventos: &eventos},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Copiar(context.Background(), p.Peticion); !errors.Is(err, ErrConciliacion) {
		t.Fatalf("error=%v", err)
	}
	for _, evento := range eventos {
		if evento == "capturar" || evento == "publicar_previa" || evento == "anotar:no_valida" {
			t.Fatalf("alteró la copia recuperable: %v", eventos)
		}
	}
}

func TestNuevoCopiaNoHabilitaRestauracionNiConciliacion(t *testing.T) {
	p, lectura, objetivo, previa := propuestaPrueba(t)
	var eventos []string
	completo, _, plataforma := servicioPrueba(t, p, lectura, objetivo, previa, aprobacionPrueba(p), &eventos)
	d := completo.d
	d.Plataforma = nil
	if _, err := Nuevo(d); !errors.Is(err, ErrBloqueada) {
		t.Fatalf("Nuevo aceptó plataforma ausente: %v", err)
	}
	s, err := NuevoCopia(d)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Restaurar(context.Background(), p); !errors.Is(err, ErrBloqueada) {
		t.Fatalf("Restaurar: %v", err)
	}
	if _, err := s.Conciliar(context.Background(), p); !errors.Is(err, ErrBloqueada) {
		t.Fatalf("Conciliar: %v", err)
	}
	if len(eventos) != 0 || plataforma.sustituciones != 0 {
		t.Fatalf("hubo efectos con constructor de copia: %v", eventos)
	}
}

func TestRestaurarBloqueaDobleControlYPreimagenAntesDeSustituir(t *testing.T) {
	for _, tt := range []struct {
		nombre string
		mutar  func(*puertos.Propuesta, *puertos.Aprobacion)
	}{
		{"misma_persona", func(_ *puertos.Propuesta, a *puertos.Aprobacion) { a.AprobadorPersonaRef = a.ProponentePersonaRef }},
		{"preimagen_distinta", func(p *puertos.Propuesta, _ *puertos.Aprobacion) { p.PreimagenSHA256 = huellaPrueba }},
	} {
		t.Run(tt.nombre, func(t *testing.T) {
			p, lectura, objetivo, previa := propuestaPrueba(t)
			tt.mutar(&p, new(puertos.Aprobacion))
			a := aprobacionPrueba(p)
			if tt.nombre == "misma_persona" {
				a.AprobadorPersonaRef = a.ProponentePersonaRef
			}
			var eventos []string
			s, _, plataforma := servicioPrueba(t, p, lectura, objetivo, previa, a, &eventos)
			if _, err := s.Restaurar(context.Background(), p); !errors.Is(err, ErrBloqueada) {
				t.Fatalf("error=%v", err)
			}
			if plataforma.sustituciones != 0 {
				t.Fatal("hubo sustitución pese al bloqueo")
			}
			for _, evento := range eventos {
				if evento == "abrir_exclusion" {
					t.Fatalf("abrió exclusión antes del bloqueo: %v", eventos)
				}
			}
		})
	}
}

func TestRestaurarOrdenaCopiaPreviaCASYSustitucion(t *testing.T) {
	p, lectura, objetivo, previa := propuestaPrueba(t)
	var eventos []string
	s, _, plataforma := servicioPrueba(t, p, lectura, objetivo, previa, aprobacionPrueba(p), &eventos)
	r, err := s.Restaurar(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	if r.Estado != "instalado_pendiente_conciliacion" || plataforma.sustituciones != 1 {
		t.Fatalf("recibo=%+v sustituciones=%d", r, plataforma.sustituciones)
	}
	pos := func(evento string) int {
		for i, actual := range eventos {
			if actual == evento {
				return i
			}
		}
		return -1
	}
	for _, evento := range []string{"capturar_previa", "capturar_previa_ref:" + p.ConjuntoPreviaRef, "cerrar_verificacion", "anotar:copia_previa_verificada", "preparar", "cas", "sustituir"} {
		if pos(evento) < 0 {
			t.Fatalf("falta %s en %v", evento, eventos)
		}
	}
	if !(pos("capturar_previa") < pos("cerrar_verificacion") && pos("cerrar_verificacion") < pos("anotar:copia_previa_verificada") && pos("anotar:copia_previa_verificada") < pos("preparar") && pos("preparar") < pos("cas") && pos("cas") < pos("sustituir")) {
		t.Fatalf("orden incorrecto: %v", eventos)
	}
}

func TestConciliarNoRepiteSustitucion(t *testing.T) {
	p, lectura, objetivo, previa := propuestaPrueba(t)
	var eventos []string
	s, registro, plataforma := servicioPrueba(t, p, lectura, objetivo, previa, aprobacionPrueba(p), &eventos)
	registro.op = puertos.Operacion{Ref: p.OperacionRef, VersionRef: "v2", Estado: "pendiente_conciliacion", ConjuntoRef: p.ConjuntoRef, ConjuntoPreviaPlaneadaRef: p.ConjuntoPreviaRef, PoliticaRef: p.PoliticaRef, CopiaPreviaRef: previa.Manifiesto.ConjuntoRef, PlanRef: "plan:sintetico", PreimagenSHA256: p.PreimagenSHA256, HuellaPropuesta: p.HuellaPropuesta}
	plataforma.instalado = objetivo.Ref
	r, err := s.Conciliar(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	if r.ConjuntoRef != objetivo.Ref || r.Estado != "instalado_pendiente_conciliacion" || plataforma.sustituciones != 0 {
		t.Fatalf("recibo=%+v sustituciones=%d", r, plataforma.sustituciones)
	}
}

func TestConciliarConDiarioRealReabreObservacionSinDuplicados(t *testing.T) {
	for _, inicial := range []string{"sustitucion_iniciada", "pendiente_conciliacion", "instalado_pendiente_conciliacion", "reversion_iniciada", "revertida"} {
		for _, anterior := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/previa=%t", inicial, anterior), func(t *testing.T) {
				ctx := context.Background()
				p, lectura, objetivo, captura := propuestaPrueba(t)
				previa := objetivo
				previa.Ref, previa.Manifiesto, previa.IndiceAutenticadoRef = p.ConjuntoPreviaRef, captura.Manifiesto, "indice:previa"
				previa.Manifiesto.Verificacion = objetivo.Manifiesto.Verificacion
				var eventos []string
				destino := &destinoPrueba{conjuntos: map[string]puertos.Conjunto{objetivo.Ref: objetivo, previa.Ref: previa}, eventos: &eventos}
				base := t.TempDir()
				dirCS07, exterior, restaurada := filepath.Join(base, "cs07"), filepath.Join(base, "exterior"), filepath.Join(base, "restaurada")
				for _, dir := range []string{dirCS07, exterior, restaurada} {
					if err := os.Mkdir(dir, 0700); err != nil {
						t.Fatal(err)
					}
				}
				journal, err := cs07.Abrir(cs07.Config{Directorio: dirCS07, RaicesRestauradas: []string{restaurada}})
				if err != nil {
					t.Fatal(err)
				}
				cfg := ejadapter.ConfigRegistroCS07{Registro: journal, Destino: destino, DirectorioExterior: exterior, RaicesRestauradas: []string{restaurada}}
				registro, err := ejadapter.AbrirRegistroCS07(cfg)
				if err != nil {
					t.Fatal(err)
				}
				if _, err = registro.ReservarRestauracion(ctx, p); err != nil {
					t.Fatal(err)
				}
				for _, etapa := range []struct{ estado, valor string }{{"exclusion_solicitada", p.PreimagenSHA256}, {"copia_previa_verificada", previa.Ref}, {"plan_preparado", "plan:prueba"}} {
					if err = registro.Anotar(ctx, p.OperacionRef, etapa.estado, etapa.valor); err != nil {
						t.Fatal(err)
					}
				}
				op, err := registro.Leer(ctx, p.OperacionRef)
				if err != nil {
					t.Fatal(err)
				}
				if _, err = registro.CAS(ctx, op.Ref, op.VersionRef, p.PreimagenSHA256, "sustitucion_iniciada"); err != nil {
					t.Fatal(err)
				}
				if inicial == "reversion_iniciada" || inicial == "revertida" {
					if err = registro.Anotar(ctx, op.Ref, "reversion_iniciada", previa.Ref); err != nil {
						t.Fatal(err)
					}
				}
				if inicial != "sustitucion_iniciada" && inicial != "reversion_iniciada" {
					valor := objetivo.Ref
					if inicial == "revertida" {
						valor = previa.Ref
					}
					if err = registro.Anotar(ctx, op.Ref, inicial, valor); err != nil {
						t.Fatal(err)
					}
				}
				instalado := objetivo.Ref
				if anterior {
					instalado = previa.Ref
				}
				plataforma := &plataformaPrueba{instalado: instalado, eventos: &eventos}
				s, err := Nuevo(Dependencias{Inventario: inventarioPrueba{lectura}, Autorizador: autorizadorPrueba{}, Registro: registro, Ventana: ventanaPrueba{eventos: &eventos}, Destino: destino, Ensayador: ensayadorPrueba{eventos: &eventos}, Plataforma: plataforma})
				if err != nil {
					t.Fatal(err)
				}
				op, err = registro.Leer(ctx, p.OperacionRef)
				if err != nil {
					t.Fatal(err)
				}
				alterada := p
				alterada.MotivoRef = "motivo:otro"
				if _, err = s.Conciliar(ctx, alterada); !errors.Is(err, ErrConciliacion) {
					t.Fatalf("propuesta distinta: %v", err)
				}
				if err = registro.ConciliarRestauracion(ctx, puertos.ObservacionRestauracion{Propuesta: p, VersionRef: "version:obsoleta", InstaladoRef: instalado, IndiceAutenticadoRef: destino.conjuntos[instalado].IndiceAutenticadoRef}); err == nil {
					t.Fatal("aceptó observación de versión obsoleta")
				}
				if err = registro.ConciliarRestauracion(ctx, puertos.ObservacionRestauracion{Propuesta: p, VersionRef: op.VersionRef, InstaladoRef: instalado, IndiceAutenticadoRef: "indice:ajeno"}); err == nil {
					t.Fatal("aceptó índice ajeno")
				}
				sinCambio, err := registro.Leer(ctx, p.OperacionRef)
				if err != nil || sinCambio != op {
					t.Fatalf("denegación cambió diario: %+v %v", sinCambio, err)
				}
				r, err := s.Conciliar(ctx, p)
				if err != nil {
					t.Fatal(err)
				}
				if r.ConjuntoRef != instalado || r.IndiceAutenticadoRef != destino.conjuntos[instalado].IndiceAutenticadoRef {
					t.Fatalf("resultado ajeno: %+v", r)
				}
				confirmada, err := registro.Leer(ctx, op.Ref)
				if err != nil {
					t.Fatal(err)
				}
				if err = registro.Close(); err != nil {
					t.Fatal(err)
				}
				registro, err = ejadapter.AbrirRegistroCS07(cfg)
				if err != nil {
					t.Fatal(err)
				}
				defer registro.Close()
				s.d.Registro = registro
				replay, err := s.Conciliar(ctx, p)
				if err != nil || replay != r {
					t.Fatalf("recuperación: %+v %v", replay, err)
				}
				final, err := registro.Leer(ctx, op.Ref)
				if err != nil || final != confirmada {
					t.Fatalf("replay añadió versión: %+v %v", final, err)
				}
				if plataforma.sustituciones != 0 {
					t.Fatal("conciliación repitió sustitución")
				}
				for _, evento := range eventos {
					if evento == "abrir_exclusion" || evento == "capturar" || evento == "preparar" || evento == "sustituir" || evento == "arrancar" {
						t.Fatalf("efecto durante observación: %s", evento)
					}
				}
			})
		}
	}
}

func TestRestaurarBloqueaCambioDeContenidoConInventarioIgual(t *testing.T) {
	p, lectura, objetivo, previa := propuestaPrueba(t)
	var eventos []string
	s, _, plataforma := servicioPrueba(t, p, lectura, objetivo, previa, aprobacionPrueba(p), &eventos)
	final := lectura
	final.PreimagenSHA256 = huellaPrueba
	s.d.Inventario = &inventarioSecuencia{primera: lectura, segunda: final}
	if _, err := s.Restaurar(context.Background(), p); !errors.Is(err, ErrBloqueada) {
		t.Fatalf("cambio de contenido aceptado: %v", err)
	}
	if plataforma.sustituciones != 0 {
		t.Fatalf("hubo %d sustituciones con datos distintos", plataforma.sustituciones)
	}
	for _, evento := range eventos {
		if evento == "cas" || evento == "sustituir" {
			t.Fatalf("hubo efecto tras cambiar preimagen: %v", eventos)
		}
	}
}
