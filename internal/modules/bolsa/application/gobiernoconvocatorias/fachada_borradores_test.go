package gobiernoconvocatorias

import (
	"context"
	"errors"
	"testing"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type preparadorContenidoFachadaDoble struct {
	contenido          dominiobolsa.ContenidoPublicableConvocatoria
	motivo             dominiovec.ReferenciaEntradaCatalogo
	err                error
	alterarTitulo      bool
	llamadasAlta       int
	llamadasActualizar int
}

func (p *preparadorContenidoFachadaDoble) PrepararAlta(
	_ context.Context,
	_ ContextoOperacionBorrador,
	solicitud SolicitudAltaBorrador,
) (PreparacionContenidoBorrador, error) {
	p.llamadasAlta++
	if p.err != nil {
		return PreparacionContenidoBorrador{}, p.err
	}
	contenido, err := p.contenido.ClonarCanonico()
	if err != nil {
		return PreparacionContenidoBorrador{}, err
	}
	if p.alterarTitulo {
		contenido.Titulo = "Título sustituido fuera de la intención"
	}
	return PreparacionContenidoBorrador{
		Contenido: contenido, MotivoSolicitado: solicitud.Motivo, MotivoCatalogo: p.motivo,
	}, nil
}

func (p *preparadorContenidoFachadaDoble) PrepararActualizacion(
	_ context.Context,
	_ ContextoOperacionBorrador,
	solicitud SolicitudActualizacionBorrador,
) (PreparacionContenidoBorrador, error) {
	p.llamadasActualizar++
	return PreparacionContenidoBorrador{
		Contenido: p.contenido, MotivoSolicitado: solicitud.Motivo, MotivoCatalogo: p.motivo,
	}, p.err
}

type lectorFachadaBorradoresDoble struct{}

func (lectorFachadaBorradoresDoble) ObtenerOpciones(
	context.Context, ContextoOperacionBorrador,
) (OpcionesBorradores, error) {
	return OpcionesBorradores{}, nil
}

func (lectorFachadaBorradoresDoble) Listar(
	context.Context, ContextoOperacionBorrador, SelectorListaBorradores,
) (ListaBorradores, error) {
	return ListaBorradores{}, nil
}

func (lectorFachadaBorradoresDoble) ObtenerDetalle(
	context.Context, ContextoOperacionBorrador, puertosbolsa.SelectorVersionConvocatoriaExacta,
) (DetalleBorrador, error) {
	return DetalleBorrador{}, nil
}

func solicitudFachadaDesdeEscenario(e escenarioPrueba) SolicitudAltaBorrador {
	contenido := e.orden.Contenido
	return SolicitudAltaBorrador{
		ClaveIdempotencia:    "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		Plantilla:            e.orden.Plantilla,
		CodigoVersionPublica: e.orden.CodigoVersionPublica,
		IdentificadorPublico: contenido.IdentificadorPublico,
		ExpedienteRef:        e.orden.ExpedienteRef,
		Contenido: ContenidoEditableBorrador{
			Tipo: contenido.Tipo, ModalidadInscripcion: contenido.ModalidadInscripcion,
			Categorias: append([]string(nil), contenido.Categorias...),
			Titulo:     contenido.Titulo, Resumen: contenido.Resumen, Descripcion: contenido.Descripcion,
			Plazos:     append([]dominiobolsa.PlazoConvocatoria(nil), contenido.Plazos...),
			Requisitos: append([]dominiobolsa.RequisitoConvocatoria(nil), contenido.Requisitos...),
			Ayuda:      []dominiobolsa.AyudaConvocatoria{},
		},
		Motivo: SelectorMotivoBorrador{
			Referencia: "motivos_rrhh:crear_borrador", Version: e.orden.MotivoCatalogo.CatalogoVersion,
			HuellaSHA256: e.orden.MotivoCatalogo.CatalogoHuellaSHA256,
		},
	}
}

func TestFachadaBorradoresModalidadEditadaCambiaEstadoYRechazaPreparacionDistinta(t *testing.T) {
	escenario := nuevoEscenario(t, confirmarBien, 2, 1)
	esperada, err := puertosbolsa.EstadoVersionConvocatoria(escenario.inicial)
	if err != nil {
		t.Fatal(err)
	}
	motivo := escenario.orden.MotivoCatalogo
	motivo.EntradaClave = "actualizar_borrador"
	preparador := &preparadorContenidoFachadaDoble{contenido: escenario.inicial.Contenido, motivo: motivo}
	preparador.contenido.ModalidadInscripcion = dominiobolsa.ModalidadInscripcionConjunta
	fachada, err := NuevaFachadaBorradoresInternos(escenario.servicio, preparador, lectorFachadaBorradoresDoble{})
	if err != nil {
		t.Fatal(err)
	}
	contexto := ContextoOperacionBorrador{
		Actor: escenario.orden.Actor, Vinculo: escenario.orden.VinculoAutenticacionActor,
		CorrelacionRef: escenario.orden.CorrelacionRef,
	}
	solicitud := SolicitudActualizacionBorrador{
		ClaveIdempotencia: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		Esperada:          esperada, Contenido: solicitudFachadaDesdeEscenario(escenario).Contenido,
		Motivo: SelectorMotivoBorrador{Referencia: motivo.Referencia(), Version: motivo.CatalogoVersion,
			HuellaSHA256: motivo.CatalogoHuellaSHA256},
	}
	solicitud.Contenido.ModalidadInscripcion = "individual"
	if _, err := fachada.Actualizar(context.Background(), contexto, solicitud); !errors.Is(err, ErrSolicitudBorradorInvalida) || preparador.llamadasActualizar != 0 {
		t.Fatalf("modalidad inválida alcanzó preparación: %v, llamadas=%d", err, preparador.llamadasActualizar)
	}
	solicitud.Contenido.ModalidadInscripcion = dominiobolsa.ModalidadInscripcionSeparada
	if _, err := fachada.Actualizar(context.Background(), contexto, solicitud); !errors.Is(err, ErrPreparacionBorradorInsegura) || escenario.confirmador.efectos != 0 {
		t.Fatalf("preparacion con otra modalidad produjo efecto: %v, %d", err, escenario.confirmador.efectos)
	}
	solicitud.Contenido.ModalidadInscripcion = dominiobolsa.ModalidadInscripcionConjunta
	recibo, err := fachada.Actualizar(context.Background(), contexto, solicitud)
	if err != nil {
		t.Fatal(err)
	}
	if recibo.EstadoPrincipal.Revision != esperada.Revision+1 ||
		recibo.EstadoPrincipal.HuellaEstadoSHA256 == esperada.HuellaEstadoSHA256 ||
		escenario.confirmador.ultima == nil ||
		escenario.confirmador.ultima.Version.Contenido.ModalidadInscripcion != dominiobolsa.ModalidadInscripcionConjunta ||
		escenario.confirmador.efectos != 1 {
		t.Fatalf("modalidad sin nueva revision, huella o efecto unico: recibo=%+v, efectos=%d", recibo, escenario.confirmador.efectos)
	}
	huellaAnterior, err := escenario.inicial.HuellaContenidoSHA256()
	if err != nil {
		t.Fatal(err)
	}
	huellaNueva, err := escenario.confirmador.ultima.Version.HuellaContenidoSHA256()
	if err != nil || huellaNueva == huellaAnterior {
		t.Fatalf("la modalidad no cambió la huella del contenido: %q/%q, %v", huellaAnterior, huellaNueva, err)
	}
}

func TestFachadaBorradoresComponeSoloContenidoGobernadoYConservaReplay(t *testing.T) {
	escenario := nuevoEscenario(t, confirmarBien)
	preparador := &preparadorContenidoFachadaDoble{
		contenido: escenario.orden.Contenido, motivo: escenario.orden.MotivoCatalogo,
	}
	fachada, err := NuevaFachadaBorradoresInternos(escenario.servicio, preparador, lectorFachadaBorradoresDoble{})
	if err != nil {
		t.Fatal(err)
	}
	contexto := ContextoOperacionBorrador{
		Actor: escenario.orden.Actor, Vinculo: escenario.orden.VinculoAutenticacionActor,
		CorrelacionRef: escenario.orden.CorrelacionRef,
	}
	solicitud := solicitudFachadaDesdeEscenario(escenario)

	primero, err := fachada.Crear(context.Background(), contexto, solicitud)
	if err != nil {
		t.Fatal(err)
	}
	segundo, err := fachada.Crear(context.Background(), contexto, solicitud)
	if err != nil {
		t.Fatal(err)
	}
	if primero.TransaccionRef != segundo.TransaccionRef || primero.EstadoPrincipal != segundo.EstadoPrincipal ||
		escenario.confirmador.efectos != 1 || preparador.llamadasAlta != 2 {
		t.Fatalf("replay inseguro: txn=%q/%q estado=%+v/%+v efectos=%d preparaciones=%d",
			primero.TransaccionRef, segundo.TransaccionRef, primero.EstadoPrincipal, segundo.EstadoPrincipal,
			escenario.confirmador.efectos, preparador.llamadasAlta)
	}
}

func TestFachadaBorradoresDeniegaContextoAusenteAntesDeResolverContenido(t *testing.T) {
	escenario := nuevoEscenario(t, confirmarBien)
	preparador := &preparadorContenidoFachadaDoble{
		contenido: escenario.orden.Contenido, motivo: escenario.orden.MotivoCatalogo,
	}
	fachada, err := NuevaFachadaBorradoresInternos(escenario.servicio, preparador, lectorFachadaBorradoresDoble{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = fachada.Crear(context.Background(), ContextoOperacionBorrador{}, solicitudFachadaDesdeEscenario(escenario))
	if !errors.Is(err, dominiovec.ErrAutorizacionDenegada) || preparador.llamadasAlta != 0 || escenario.confirmador.efectos != 0 {
		t.Fatalf("default-deny incumplido: err=%v preparaciones=%d efectos=%d", err, preparador.llamadasAlta, escenario.confirmador.efectos)
	}
}

func TestFachadaBorradoresRechazaPreparacionQueAlteraCampoEditable(t *testing.T) {
	escenario := nuevoEscenario(t, confirmarBien)
	preparador := &preparadorContenidoFachadaDoble{
		contenido: escenario.orden.Contenido, motivo: escenario.orden.MotivoCatalogo, alterarTitulo: true,
	}
	fachada, err := NuevaFachadaBorradoresInternos(escenario.servicio, preparador, lectorFachadaBorradoresDoble{})
	if err != nil {
		t.Fatal(err)
	}
	contexto := ContextoOperacionBorrador{
		Actor: escenario.orden.Actor, Vinculo: escenario.orden.VinculoAutenticacionActor,
		CorrelacionRef: escenario.orden.CorrelacionRef,
	}
	_, err = fachada.Crear(context.Background(), contexto, solicitudFachadaDesdeEscenario(escenario))
	if !errors.Is(err, ErrPreparacionBorradorInsegura) || escenario.confirmador.efectos != 0 {
		t.Fatalf("alteración no detenida: err=%v efectos=%d", err, escenario.confirmador.efectos)
	}
}

func TestNuevaFachadaBorradoresFallaCerradaConNulosTipados(t *testing.T) {
	escenario := nuevoEscenario(t, confirmarBien)
	preparador := &preparadorContenidoFachadaDoble{}
	lector := lectorFachadaBorradoresDoble{}
	var servicioNulo *ServicioBorradores
	var preparadorNulo *preparadorContenidoFachadaDoble
	casos := []struct {
		mutador    MutadorBorradores
		preparador PreparadorContenidoBorrador
		lector     LectorBorradoresInternos
	}{
		{nil, preparador, lector}, {escenario.servicio, nil, lector}, {escenario.servicio, preparador, nil},
		{servicioNulo, preparador, lector}, {escenario.servicio, preparadorNulo, lector},
	}
	for indice, caso := range casos {
		if fachada, err := NuevaFachadaBorradoresInternos(caso.mutador, caso.preparador, caso.lector); fachada != nil ||
			!errors.Is(err, ErrFachadaBorradoresInvalida) {
			t.Fatalf("caso %d: fachada=%v err=%v", indice, fachada, err)
		}
	}
}
