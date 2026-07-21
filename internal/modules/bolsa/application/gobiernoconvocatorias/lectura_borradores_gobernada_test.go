package gobiernoconvocatorias

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"golang.org/x/text/unicode/norm"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type preautorizadorLecturaFallaPrueba struct{ llamadas int }

func (p *preautorizadorLecturaFallaPrueba) PreautorizarLecturaBorrador(
	context.Context, ContextoOperacionBorrador, string,
	puertosbolsa.SelectorVersionConvocatoriaExacta,
) (CapacidadLecturaBorrador, error) {
	p.llamadas++
	return CapacidadLecturaBorrador{}, errors.New("pdp no disponible")
}

type preautorizadorLecturaCapacidadPrueba struct {
	llamadas  int
	capacidad CapacidadLecturaBorrador
}

func (p *preautorizadorLecturaCapacidadPrueba) PreautorizarLecturaBorrador(
	context.Context, ContextoOperacionBorrador, string,
	puertosbolsa.SelectorVersionConvocatoriaExacta,
) (CapacidadLecturaBorrador, error) {
	p.llamadas++
	return p.capacidad, nil
}

type repositorioLecturaContadorPrueba struct {
	listados, detalles int
	lista              ListaBorradores
	detalle            DetalleBorrador
}

func (r *repositorioLecturaContadorPrueba) ListarBorradoresGobernados(
	context.Context, SolicitudListadoBorradoresGobernada,
) (ListaBorradores, error) {
	r.listados++
	return r.lista, nil
}

func (r *repositorioLecturaContadorPrueba) ObtenerBorradorGobernado(
	context.Context, SolicitudDetalleBorradorGobernada,
) (DetalleBorrador, error) {
	r.detalles++
	return r.detalle, nil
}

type generadorCorrelacionLecturaPrueba struct{ valor string }

func (g generadorCorrelacionLecturaPrueba) NuevaReferenciaCorrelacionAutorizacionV2(
	context.Context,
) (string, error) {
	return g.valor, nil
}

func capacidadLecturaGobernadaPrueba(
	t *testing.T, contexto ContextoOperacionBorrador, accion string,
	selector puertosbolsa.SelectorVersionConvocatoriaExacta,
) CapacidadLecturaBorrador {
	t.Helper()
	organizacion, unidad := "org_diputaciongranada", "uni_seleccionexterna"
	referencia, tipo := selector.Referencia(), puertosbolsa.TipoRecursoVersionConvocatoriaGobernada
	if accion == AccionListarBorradoresGobernados {
		referencia, tipo = "borradores:"+organizacion, TipoColeccionBorradoresGobernados
	}
	recurso := dominiovec.RecursoAutorizable{
		Referencia: referencia, ModuloID: puertosbolsa.ModuloGobiernoConvocatorias, Tipo: tipo,
		Ambitos:   map[string]string{"organizacion_ref": organizacion, "unidad_gestion_ref": unidad},
		Atributos: map[string]string{},
	}
	motivo := dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_rrhh", CatalogoVersion: 1,
		CatalogoHuellaSHA256: strings.Repeat("9", 64),
		EntradaClave:         "motivo_0123456789abcdef0123456789abcdef",
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(
		context.Background(), generadorCorrelacionLecturaPrueba{contexto.CorrelacionRef},
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitud, err := dominiovec.NuevaSolicitudAutorizacionLigadaV2(
		dominiovec.DatosSolicitudAutorizacionLigadaV2{
			ContextoActor: contexto.Actor, VinculoAutenticacionActor: contexto.Vinculo,
			ReferenciaMotivo: motivo, Accion: accion, Recurso: recurso,
			Finalidad: FinalidadLecturaBorradoresGobernada, Correlacion: correlacion,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	huellaSolicitud, err := dominiovec.HuellaSHA256SolicitudAutorizacionV2(solicitud)
	if err != nil {
		t.Fatal(err)
	}
	huellaMotivo, err := dominiovec.HuellaSHA256MotivoAutorizacionV2(motivo)
	if err != nil {
		t.Fatal(err)
	}
	huellaRecurso, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	huellaCatalogo, err := dominiovec.HuellaEvidenciasCatalogoPoliticasAutorizacion(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	instante := instanteBorradorPrueba
	decision := dominiovec.DecisionAutorizacion{
		DecisionRef: "decision:lectura-borrador:00000001", Concedida: true, Codigo: "concedida",
		PrincipalID: contexto.Actor.PersonaRef, PerfilActivoRef: contexto.Actor.PerfilActivoRef,
		Accion: accion, RecursoRef: referencia, ModuloID: puertosbolsa.ModuloGobiernoConvocatorias,
		TipoRecurso: tipo, ContextoRecursoHuellaSHA256: huellaRecurso,
		Finalidad: FinalidadLecturaBorradoresGobernada, CorrelacionRef: contexto.CorrelacionRef,
		EsquemaHuellaSolicitud: dominiovec.EsquemaHuellaSolicitudAutorizacionV2,
		SolicitudHuellaSHA256:  huellaSolicitud,
		EsquemaHuellaMotivo:    dominiovec.EsquemaHuellaMotivoAutorizacionV2,
		MotivoHuellaSHA256:     huellaMotivo, VinculoAutenticacionActor: contexto.Vinculo,
		AsignacionRef: "asignacion:lectura-borrador:v1", AsignacionHuellaSHA256: strings.Repeat("1", 64),
		VersionRolRef: "rol:lectura-borrador:v1", VersionRolHuellaSHA256: strings.Repeat("2", 64),
		ControlVigenciaVersionRolRef: "rol:lectura-borrador:v1", ControlVigenciaVersionRolRevision: 1,
		ControlVigenciaVersionRolHuellaSHA256: strings.Repeat("3", 64),
		RevisionCatalogoPoliticas:             1, CatalogoPoliticasHuellaSHA256: huellaCatalogo,
		PoliticasEvaluadasHuellasSHA256: map[string]string{}, GarantiaMinima: dominiovec.AuthAssuranceHigh,
		CamposPermitidos: []string{"version_convocatoria"}, EmitidaEn: instante.Add(-time.Second),
		ValidaHasta: instante.Add(time.Minute),
	}
	evidencia, err := puertosvec.NuevaEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2(decision, instante)
	if err != nil {
		t.Fatal(err)
	}
	return CapacidadLecturaBorrador{
		Solicitud: solicitud, Evidencia: evidencia, Motivo: motivo, Recurso: recurso,
		OrganizacionRef: organizacion, UnidadGestionRef: unidad,
		AtestacionRef: "atestacion:lectura-borrador:00000001", VersionAtestacion: 1,
		EstadoAtestacion: "activa", HuellaAtestacionSHA256: strings.Repeat("4", 64),
	}
}

func contextoLecturaGobernadaPrueba(t *testing.T) ContextoOperacionBorrador {
	t.Helper()
	e := nuevoEscenario(t, confirmarBien)
	return ContextoOperacionBorrador{
		Actor: e.orden.Actor, Vinculo: e.orden.VinculoAutenticacionActor,
		CorrelacionRef: "correlacion_0123456789abcdef0123456789abcdef",
	}
}

func TestLecturaGobernadaNoConsultaRepositorioSiFallaPreautorizacion(t *testing.T) {
	preautorizador := &preautorizadorLecturaFallaPrueba{}
	repositorio := &repositorioLecturaContadorPrueba{}
	servicio, err := NuevoServicioLecturaBorradoresGobernada(preautorizador, repositorio)
	if err != nil {
		t.Fatal(err)
	}
	contexto := contextoLecturaGobernadaPrueba(t)
	_, err = servicio.Listar(context.Background(), contexto, SelectorListaBorradores{Limite: 10})
	if err == nil || preautorizador.llamadas != 1 || repositorio.listados != 0 || repositorio.detalles != 0 {
		t.Fatalf("fallo de cierre previo: err=%v pre=%d repo=%d/%d", err, preautorizador.llamadas, repositorio.listados, repositorio.detalles)
	}
}

func TestLecturaGobernadaNoConsultaRepositorioConCapacidadInvalida(t *testing.T) {
	contexto := contextoLecturaGobernadaPrueba(t)
	preautorizador := &preautorizadorLecturaCapacidadPrueba{}
	repositorio := &repositorioLecturaContadorPrueba{}
	servicio, err := NuevoServicioLecturaBorradoresGobernada(preautorizador, repositorio)
	if err != nil {
		t.Fatal(err)
	}
	_, err = servicio.Listar(context.Background(), contexto, SelectorListaBorradores{Limite: 10})
	if err == nil || preautorizador.llamadas != 1 || repositorio.listados != 0 {
		t.Fatalf("capacidad invalida alcanzo repositorio: err=%v pre=%d repo=%d", err, preautorizador.llamadas, repositorio.listados)
	}
}

func TestLecturaGobernadaCanceladaNoInvocaPreautorizadorNiRepositorio(t *testing.T) {
	contexto := contextoLecturaGobernadaPrueba(t)
	preautorizador := &preautorizadorLecturaCapacidadPrueba{}
	repositorio := &repositorioLecturaContadorPrueba{}
	servicio, err := NuevoServicioLecturaBorradoresGobernada(preautorizador, repositorio)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	_, err = servicio.Listar(ctx, contexto, SelectorListaBorradores{Limite: 10})
	if !errors.Is(err, context.Canceled) || preautorizador.llamadas != 0 || repositorio.listados != 0 {
		t.Fatalf("cancelacion alcanzo dependencias: err=%v pre=%d repo=%d", err, preautorizador.llamadas, repositorio.listados)
	}
}

func TestSelectorListaBorradoresFallaCerradoAntesDelPDP(t *testing.T) {
	validos := []SelectorListaBorradores{
		{Limite: 1},
		{
			Limite: 50, Cursor: "cursor-borrador-" + strings.Repeat("a", 64),
			Texto: "selección externa", Categoria: "auxiliar_administrativo",
		},
	}
	for _, selector := range validos {
		if err := selector.Validar(); err != nil {
			t.Fatalf("selector canonico rechazado: %+v err=%v", selector, err)
		}
	}
	invalidos := map[string]SelectorListaBorradores{
		"limite":             {Limite: 51},
		"cursor libre":       {Limite: 10, Cursor: "cursor-libre"},
		"cursor enorme":      {Limite: 10, Cursor: "cursor-borrador-" + strings.Repeat("a", 65)},
		"texto enorme":       {Limite: 10, Texto: strings.Repeat("a", 181)},
		"texto no NFC":       {Limite: 10, Texto: "seleccio\u0301n"},
		"texto con margen":   {Limite: 10, Texto: " auxiliar"},
		"texto con control":  {Limite: 10, Texto: "auxiliar\nadmin"},
		"categoria enorme":   {Limite: 10, Categoria: strings.Repeat("a", 81)},
		"categoria no canon": {Limite: 10, Categoria: "Auxiliar"},
	}
	for nombre, selector := range invalidos {
		t.Run(nombre, func(t *testing.T) {
			preautorizador := &preautorizadorLecturaCapacidadPrueba{}
			repositorio := &repositorioLecturaContadorPrueba{}
			servicio, err := NuevoServicioLecturaBorradoresGobernada(preautorizador, repositorio)
			if err != nil {
				t.Fatal(err)
			}
			_, err = servicio.Listar(context.Background(), contextoLecturaGobernadaPrueba(t), selector)
			if !errors.Is(err, ErrLecturaBorradoresGobernadaInvalida) ||
				preautorizador.llamadas != 0 || repositorio.listados != 0 {
				t.Fatalf("selector alcanzo PDP/repositorio: err=%v pre=%d repo=%d", err, preautorizador.llamadas, repositorio.listados)
			}
		})
	}
}

func TestSelectorListaBorradoresAplicaPerfilUnicodeComunV1(t *testing.T) {
	t.Helper()
	if norm.Version != versionUnicodeNormalizadorBorradoresV1 {
		t.Fatalf(
			"revisar perfil Unicode: x/text=%s contrato=%s",
			norm.Version, versionUnicodeNormalizadorBorradoresV1,
		)
	}
	type intervalo struct {
		desde rune
		hasta rune
	}
	// Diferencias completas de propiedades de normalizacion entre las tablas
	// Unicode 15, 16 y 17 soportadas por el contrato: 48 del salto 15->16 y
	// 34 CCC del salto 16->17, no solo U+1ACF..U+1ADD.
	incompatibles := []intervalo{
		{0x0897, 0x0897},
		{0x1ACF, 0x1ADD}, {0x1AE0, 0x1AEB},
		{0x105C9, 0x105C9}, {0x105D2, 0x105D2},
		{0x105DA, 0x105DA}, {0x105E4, 0x105E4},
		{0x10D69, 0x10D6D}, {0x10EFA, 0x10EFB},
		{0x11382, 0x11385}, {0x1138B, 0x1138B},
		{0x1138E, 0x1138E}, {0x11390, 0x11391},
		{0x113B8, 0x113B8}, {0x113BB, 0x113BB},
		{0x113C2, 0x113C2}, {0x113C5, 0x113C5},
		{0x113C7, 0x113C9}, {0x113CE, 0x113D0},
		{0x1611E, 0x16129}, {0x1612F, 0x1612F},
		{0x16D63, 0x16D63}, {0x16D67, 0x16D6A},
		{0x1E5EE, 0x1E5EF},
		{0x1E6E3, 0x1E6E3}, {0x1E6E6, 0x1E6E6},
		{0x1E6EE, 0x1E6EF}, {0x1E6F5, 0x1E6F5},
	}
	totalIncompatibles := 0
	for _, tramo := range incompatibles {
		for caracter := tramo.desde; caracter <= tramo.hasta; caracter++ {
			totalIncompatibles++
			selector := SelectorListaBorradores{Limite: 10, Texto: string(caracter)}
			if err := selector.Validar(); !errors.Is(err, ErrSolicitudBorradorInvalida) {
				t.Fatalf("acepto runa fuera del perfil comun: %U err=%v", caracter, err)
			}
		}
	}
	if totalIncompatibles != 82 {
		t.Fatalf("corpus NFC comun incompleto: %d, esperado 82", totalIncompatibles)
	}

	for nombre, texto := range map[string]string{
		"post16 desorden ccc230-220": "x\u1ACF\u1ADD",
		"post16 orden ccc220-230":    "x\u1ADD\u1ACF",
		"post16 desorden ccc234-230": "x\u1AEB\u0301",
		"post16 orden ccc220-230 b":  "x\U00010EFA\U0001E6E3",
		"todhri ei":                  "\U000105D2\u0307",
		"todhri u":                   "\U000105DA\u0307",
		"tulu ii":                    "\U00011382\U000113C9",
		"tulu uu":                    "\U00011384\U000113BB",
		"tulu ai":                    "\U0001138B\U000113C2",
		"tulu au":                    "\U00011390\U000113C9",
		"tulu signo ai":              "\U000113C2\U000113C2",
		"tulu signo oo":              "\U000113C2\U000113B8",
		"tulu signo au":              "\U000113C2\U000113C9",
		"gurung u":                   "\U0001611E\U0001611E",
		"gurung uu":                  "\U0001611E\U00016129",
		"gurung e":                   "\U0001611E\U0001611F",
		"gurung ee":                  "\U00016129\U0001611F",
		"gurung ai":                  "\U0001611E\U00016120",
		"kirat ai":                   "\U00016D67\U00016D67",
		"kirat o":                    "\U00016D63\U00016D67",
	} {
		t.Run(nombre, func(t *testing.T) {
			selector := SelectorListaBorradores{Limite: 10, Texto: texto}
			if err := selector.Validar(); !errors.Is(err, ErrSolicitudBorradorInvalida) {
				t.Fatalf("acepto combinacion fuera del perfil comun: %U err=%v", []rune(texto), err)
			}
		})
	}

	vecinosPermitidos := []rune{
		0x0896, 0x0898,
		0x105C8, 0x105CA, 0x105D1, 0x105D3,
		0x105D9, 0x105DB, 0x105E3, 0x105E5,
		0x10D68, 0x10D6E,
		0x11381, 0x11386, 0x1138A, 0x1138C,
		0x1138D, 0x1138F, 0x11392,
		0x113B7, 0x113B9, 0x113BA, 0x113BC,
		0x113C1, 0x113C3, 0x113C4, 0x113C6, 0x113CA,
		0x113CD, 0x113D1,
		0x1611D, 0x1612A, 0x1612E, 0x16130,
		0x16D62, 0x16D64, 0x16D66, 0x16D6B,
		0x1E5ED, 0x1E5F0,
		0x1ACE, 0x1ADE, 0x1ADF, 0x1AEC,
		0xA7F0, 0xA7F2, 0x10EF9, 0x10EFC,
		0x1E6E2, 0x1E6E4, 0x1E6E5, 0x1E6E7,
		0x1E6ED, 0x1E6F0, 0x1E6F4, 0x1E6F6,
	}
	for _, caracter := range vecinosPermitidos {
		selector := SelectorListaBorradores{Limite: 10, Texto: string(caracter)}
		if err := selector.Validar(); err != nil {
			t.Fatalf("rechazo vecino estable permitido: %U err=%v", caracter, err)
		}
	}
	// Estos 37 escalares cambian solo NFKC, no el NFC prometido por el
	// selector; bloquearlos ampliaria el contrato sin causa.
	compatibilidadPermitida := []rune{0xA7F1}
	for caracter := rune(0x1CCD6); caracter <= 0x1CCF9; caracter++ {
		compatibilidadPermitida = append(compatibilidadPermitida, caracter)
	}
	for _, caracter := range compatibilidadPermitida {
		selector := SelectorListaBorradores{Limite: 10, Texto: string(caracter)}
		if err := selector.Validar(); err != nil {
			t.Fatalf("rechazo runa con cambio solo NFKC: %U err=%v", caracter, err)
		}
	}
	for nombre, texto := range map[string]string{
		"ccc conocido ordenado":      "x\u0323\u0301",
		"vecino anterior post16":     "x\u1ACE\u0301",
		"hueco estable entre rangos": "x\u1ADE\u0301",
	} {
		if err := (SelectorListaBorradores{Limite: 10, Texto: texto}).Validar(); err != nil {
			t.Fatalf("rechazo combinacion estable %s: %U err=%v", nombre, []rune(texto), err)
		}
	}
}

func TestLecturaGobernadaPropagaResultadosTrasCapacidadV2Exacta(t *testing.T) {
	contexto := contextoLecturaGobernadaPrueba(t)
	selectorDetalle := puertosbolsa.SelectorVersionConvocatoriaExacta{ID: "convocatoria-prueba", Secuencia: 1}
	for _, caso := range []struct {
		nombre   string
		accion   string
		selector puertosbolsa.SelectorVersionConvocatoriaExacta
		ejecutar func(*ServicioLecturaBorradoresGobernada) error
		listados int
		detalles int
	}{
		{
			nombre: "listado", accion: AccionListarBorradoresGobernados,
			ejecutar: func(s *ServicioLecturaBorradoresGobernada) error {
				_, err := s.Listar(context.Background(), contexto, SelectorListaBorradores{Limite: 10})
				return err
			}, listados: 1,
		},
		{
			nombre: "detalle", accion: AccionConsultarBorradorGobernado, selector: selectorDetalle,
			ejecutar: func(s *ServicioLecturaBorradoresGobernada) error {
				_, err := s.ObtenerDetalle(context.Background(), contexto, selectorDetalle)
				return err
			}, detalles: 1,
		},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			preautorizador := &preautorizadorLecturaCapacidadPrueba{
				capacidad: capacidadLecturaGobernadaPrueba(t, contexto, caso.accion, caso.selector),
			}
			repositorio := &repositorioLecturaContadorPrueba{}
			servicio, err := NuevoServicioLecturaBorradoresGobernada(preautorizador, repositorio)
			if err != nil {
				t.Fatal(err)
			}
			if err := caso.ejecutar(servicio); err != nil || preautorizador.llamadas != 1 ||
				repositorio.listados != caso.listados || repositorio.detalles != caso.detalles {
				t.Fatalf("recorrido V2 incompleto: err=%v pre=%d repo=%d/%d", err, preautorizador.llamadas, repositorio.listados, repositorio.detalles)
			}
		})
	}
}
