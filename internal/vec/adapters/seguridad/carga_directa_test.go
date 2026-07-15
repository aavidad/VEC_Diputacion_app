package seguridad

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
	pruebasvec "vec-diputacion-granada/internal/vec/pruebas"
)

type relojCargaDirectaPrueba struct {
	mu       sync.RWMutex
	instante time.Time
}

func (r *relojCargaDirectaPrueba) Ahora() time.Time {
	if r == nil {
		return time.Time{}
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.instante
}

func (r *relojCargaDirectaPrueba) fijar(instante time.Time) {
	r.mu.Lock()
	r.instante = instante
	r.mu.Unlock()
}

type entradaReciboCargaDirectaPrueba struct {
	registro     ports.RegistroReciboCargaDirecta
	registradoEn time.Time
	consumido    bool
	activo       bool
	orden        ports.OrdenConsumoReciboCargaDirecta
}

// Este doble solo prueba el contrato atomico. No es ni se presenta como
// repositorio productivo: el despliegue real requiere persistencia durable.
type repositorioRecibosCargaDirectaPrueba struct {
	mu              sync.Mutex
	entradas        map[string]entradaReciboCargaDirectaPrueba
	activos         map[string]string
	errorAlta       error
	errorConsumo    error
	conflictosAlta  int
	llamadasAlta    int
	llamadasConsumo int
	resultadosAlta  []ports.ResultadoRegistroReciboCargaDirecta
	horaDurable     time.Time
	altaIniciada    chan struct{}
	desbloquearAlta chan struct{}
}

func nuevoRepositorioRecibosCargaDirectaPrueba() *repositorioRecibosCargaDirectaPrueba {
	return &repositorioRecibosCargaDirectaPrueba{
		entradas:    make(map[string]entradaReciboCargaDirectaPrueba),
		activos:     make(map[string]string),
		horaDurable: time.Date(2026, 7, 15, 8, 30, 0, 123456789, time.UTC),
	}
}

func (r *repositorioRecibosCargaDirectaPrueba) RegistrarReciboCargaDirecta(
	ctx context.Context,
	registro ports.RegistroReciboCargaDirecta,
) (ports.ResultadoRegistroReciboCargaDirecta, error) {
	if err := ctx.Err(); err != nil {
		return ports.ResultadoRegistroReciboCargaDirecta{}, err
	}
	if registro.Validar() != nil {
		return ports.ResultadoRegistroReciboCargaDirecta{}, ports.ErrReciboCargaDirectaNoValido
	}
	if r.altaIniciada != nil {
		select {
		case r.altaIniciada <- struct{}{}:
		default:
		}
	}
	if r.desbloquearAlta != nil {
		select {
		case <-ctx.Done():
			return ports.ResultadoRegistroReciboCargaDirecta{}, ctx.Err()
		case <-r.desbloquearAlta:
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.llamadasAlta++
	if r.errorAlta != nil {
		return ports.ResultadoRegistroReciboCargaDirecta{}, r.errorAlta
	}
	if r.conflictosAlta > 0 {
		r.conflictosAlta--
		return ports.ResultadoRegistroReciboCargaDirecta{}, ports.ErrRegistroReciboCargaDirectaConflicto
	}
	if _, existe := r.entradas[registro.IndiceHMAC]; existe {
		return ports.ResultadoRegistroReciboCargaDirecta{}, ports.ErrRegistroReciboCargaDirectaConflicto
	}
	registradoEn := r.horaDurable
	if registradoEn.IsZero() || registradoEn.Location() != time.UTC ||
		!registradoEn.Before(registro.ExpiraEn) ||
		registro.ExpiraEn.Sub(registradoEn) > duracionMaximaReciboCargaDirecta {
		return ports.ResultadoRegistroReciboCargaDirecta{}, ports.ErrReciboCargaDirectaNoValido
	}
	resultado := ports.ResultadoRegistroReciboCargaDirecta{
		IndiceHMAC: registro.IndiceHMAC, GrupoHMAC: registro.GrupoHMAC,
		AutorizacionEmisionRef: registro.AutorizacionEmisionRef, RegistradoEn: registradoEn,
	}
	if indiceAnterior, existe := r.activos[registro.GrupoHMAC]; existe {
		anterior := r.entradas[indiceAnterior]
		anterior.activo = false
		r.entradas[indiceAnterior] = anterior
		resultado.Predecesor = &ports.PredecesorReciboCargaDirecta{
			IndiceHMAC:             anterior.registro.IndiceHMAC,
			GrupoHMAC:              anterior.registro.GrupoHMAC,
			AutorizacionEmisionRef: anterior.registro.AutorizacionEmisionRef,
			SustituidoEn:           registradoEn,
		}
	}
	r.entradas[registro.IndiceHMAC] = entradaReciboCargaDirectaPrueba{
		registro: registro, registradoEn: registradoEn, activo: true,
	}
	r.activos[registro.GrupoHMAC] = registro.IndiceHMAC
	if resultado.ValidarContra(registro) != nil {
		return ports.ResultadoRegistroReciboCargaDirecta{}, ports.ErrReciboCargaDirectaNoValido
	}
	r.resultadosAlta = append(r.resultadosAlta, resultado)
	return resultado, nil
}

func (r *repositorioRecibosCargaDirectaPrueba) ConsumirReciboCargaDirecta(
	ctx context.Context,
	orden ports.OrdenConsumoReciboCargaDirecta,
) (ports.ResultadoConsumoReciboCargaDirecta, error) {
	if err := ctx.Err(); err != nil {
		return ports.ResultadoConsumoReciboCargaDirecta{}, err
	}
	if orden.Validar() != nil {
		return ports.ResultadoConsumoReciboCargaDirecta{}, ports.ErrConsumoReciboCargaDirectaDenegado
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.llamadasConsumo++
	if r.errorConsumo != nil {
		return ports.ResultadoConsumoReciboCargaDirecta{}, r.errorConsumo
	}
	entrada, existe := r.entradas[orden.IndiceHMAC]
	indiceActivo, grupoActivo := r.activos[orden.GrupoHMAC]
	if !existe || entrada.consumido || !entrada.activo || !grupoActivo || indiceActivo != orden.IndiceHMAC ||
		entrada.registro.GrupoHMAC != orden.GrupoHMAC || entrada.registro.VinculoHMAC != orden.VinculoHMAC ||
		!entrada.registradoEn.Equal(orden.RegistradoEn) {
		return ports.ResultadoConsumoReciboCargaDirecta{}, ports.ErrConsumoReciboCargaDirectaDenegado
	}
	consumidoEn := r.horaDurable
	if consumidoEn.IsZero() || consumidoEn.Location() != time.UTC ||
		consumidoEn.Before(entrada.registradoEn) || !consumidoEn.Before(entrada.registro.ExpiraEn) ||
		!consumidoEn.Before(orden.ValidaHasta) {
		return ports.ResultadoConsumoReciboCargaDirecta{}, ports.ErrConsumoReciboCargaDirectaDenegado
	}
	entrada.consumido = true
	entrada.activo = false
	entrada.orden = orden
	r.entradas[orden.IndiceHMAC] = entrada
	delete(r.activos, orden.GrupoHMAC)
	return ports.ResultadoConsumoReciboCargaDirecta{
		IndiceHMAC: orden.IndiceHMAC, GrupoHMAC: orden.GrupoHMAC, VinculoHMAC: orden.VinculoHMAC,
		EvidenciaConsumoRef:      orden.EvidenciaConsumoRef,
		IntencionConfirmacionRef: orden.IntencionConfirmacionRef,
		HuellaIntencionHMAC:      orden.HuellaIntencionHMAC,
		RegistradoEn:             entrada.registradoEn, ConsumidoEn: consumidoEn, ExpiraEn: entrada.registro.ExpiraEn,
	}, nil
}

func (r *repositorioRecibosCargaDirectaPrueba) estado() (altas, consumos, entradas, usados int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, entrada := range r.entradas {
		if entrada.consumido {
			usados++
		}
	}
	return r.llamadasAlta, r.llamadasConsumo, len(r.entradas), usados
}

func configuracionCargaDirectaPrueba() ConfiguracionCriptografiaCargaDirecta {
	return ConfiguracionCriptografiaCargaDirecta{
		ClaveSeudonimizacion: ConfiguracionClaveHMACCargaDirecta{
			Identificador: "seudonimo_v1",
			Material:      []byte("clave-seudonimo-carga-01-32-bytes!!"),
		},
		ClaveIndiceRecibo: ConfiguracionClaveHMACCargaDirecta{
			Identificador: "indice_recibo_v1",
			Material:      []byte("clave-indice-recibo-02-32-bytes!!!"),
		},
		ClaveVinculoRecibo: ConfiguracionClaveHMACCargaDirecta{
			Identificador: "vinculo_recibo_v1",
			Material:      []byte("clave-vinculo-recibo-03-32-bytes!!"),
		},
		ClaveAtestacion: ConfiguracionClaveHMACCargaDirecta{
			Identificador: "atestacion_recibo_v1",
			Material:      []byte("clave-atestacion-recibo-04-32bytes!!"),
		},
	}
}

func nuevoAdaptadorCargaDirectaPrueba(
	t *testing.T,
) (*AdaptadorCriptograficoCargaDirecta, *repositorioRecibosCargaDirectaPrueba, *relojCargaDirectaPrueba) {
	t.Helper()
	repositorio := nuevoRepositorioRecibosCargaDirectaPrueba()
	reloj := &relojCargaDirectaPrueba{instante: time.Date(2026, 7, 15, 8, 30, 0, 123456789, time.UTC)}
	adaptador, err := NuevoAdaptadorCriptograficoCargaDirecta(
		configuracionCargaDirectaPrueba(), repositorio, reloj,
	)
	if err != nil {
		t.Fatalf("crear adaptador: %v", err)
	}
	return adaptador, repositorio, reloj
}

func contextoPreparacionCargaDirectaPrueba() ports.ContextoOperacionAlmacen {
	return contextoCargaDirectaPrueba(false, nil)
}

func contextoConfirmacionCargaDirectaPrueba() ports.ContextoOperacionAlmacen {
	return contextoCargaDirectaPrueba(true, nil)
}

type mutacionContextoCargaDirectaPrueba func(
	*domain.DecisionAutorizacion,
	*domain.RecursoAutorizable,
	*ports.VinculosOperacionAlmacen,
)

func contextoCargaDirectaPrueba(
	confirmacion bool,
	mutar mutacionContextoCargaDirectaPrueba,
) ports.ContextoOperacionAlmacen {
	instante := time.Date(2026, 7, 15, 8, 30, 0, 0, time.UTC)
	accion := ports.AccionNegocioPrepararCargaDocumental
	campos := []string{"clasificacion", "contenido", "huella_sha256", "mime", "tamano"}
	decisionRef := "autorizacion:preparar:001"
	if confirmacion {
		accion = ports.AccionNegocioConfirmarCargaDocumental
		campos = []string{"contenido_cuarentena", "estado"}
		decisionRef = "autorizacion:confirmar:002"
	}
	vinculos := ports.VinculosOperacionAlmacen{
		OperacionRef: "operacion:carga:001", CargaRef: "carga:001",
		Clasificacion:       "confidencial",
		SujetoSeudonimoHMAC: "hmac-sha256:seudonimo_v1:" + strings.Repeat("a", 64),
		HuellaSolicitudHMAC: "hmac-sha256:solicitud_v1:" + strings.Repeat("b", 64),
		EfectoRef:           "efecto:carga:001",
	}
	recurso := domain.RecursoAutorizable{
		Referencia: "expediente:001", ModuloID: "bolsas", Tipo: "expediente",
		Ambitos:   map[string]string{"organizacion": "diputacion_granada"},
		Atributos: map[string]string{},
	}
	_, vinculoActor, err := pruebasvec.NuevoContextoYVinculo(
		instante, "per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl",
		domain.AuthMethodCertificate, domain.AuthAssuranceHigh,
	)
	if err != nil {
		panic(err)
	}
	huellaCatalogo, err := domain.HuellaEvidenciasCatalogoPoliticasAutorizacion(nil, nil)
	if err != nil {
		panic(err)
	}
	decision := domain.DecisionAutorizacion{
		DecisionRef: decisionRef, Concedida: true, Codigo: "concedida",
		PrincipalID: "per_0123456789abcdefghijkl", PerfilActivoRef: "prf_0123456789abcdefghijkl",
		Accion: accion, RecursoRef: recurso.Referencia, ModuloID: recurso.ModuloID, TipoRecurso: recurso.Tipo,
		Finalidad: "incorporar_documento_expediente", CorrelacionRef: "correlacion:001",
		VinculoAutenticacionActor: vinculoActor,
		AsignacionRef:             "asignacion:carga:v1", AsignacionHuellaSHA256: strings.Repeat("1", 64),
		VersionRolRef: "rol:carga:v1", VersionRolHuellaSHA256: strings.Repeat("2", 64),
		ControlVigenciaVersionRolRef: "rol:carga:v1", ControlVigenciaVersionRolRevision: 1,
		ControlVigenciaVersionRolHuellaSHA256: strings.Repeat("3", 64),
		RevisionCatalogoPoliticas:             1, CatalogoPoliticasHuellaSHA256: huellaCatalogo,
		PoliticasEvaluadasHuellasSHA256: map[string]string{}, GarantiaMinima: domain.AuthAssuranceHigh,
		CamposPermitidos: campos, EmitidaEn: instante, ValidaHasta: instante.Add(5 * time.Minute),
	}
	if mutar != nil {
		mutar(&decision, &recurso, &vinculos)
	}
	recurso.Atributos = map[string]string{
		ports.AtributoAlmacenOperacionRef:        vinculos.OperacionRef,
		ports.AtributoAlmacenCargaRef:            vinculos.CargaRef,
		ports.AtributoAlmacenClasificacion:       vinculos.Clasificacion,
		ports.AtributoAlmacenSujetoSeudonimoHMAC: vinculos.SujetoSeudonimoHMAC,
		ports.AtributoAlmacenHuellaSolicitudHMAC: vinculos.HuellaSolicitudHMAC,
		ports.AtributoAlmacenEfectoRef:           vinculos.EfectoRef,
	}
	decision.RecursoRef, decision.ModuloID, decision.TipoRecurso = recurso.Referencia, recurso.ModuloID, recurso.Tipo
	huellaRecurso, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		panic(err)
	}
	decision.ContextoRecursoHuellaSHA256 = huellaRecurso
	var contexto ports.ContextoOperacionAlmacen
	if confirmacion {
		contexto, err = ports.NuevoContextoConfirmarCargaDirectaAlmacen(decision, recurso, vinculos, instante)
	} else {
		contexto, err = ports.NuevoContextoPrepararCargaDirectaAlmacen(decision, recurso, vinculos, instante)
	}
	if err != nil {
		panic(err)
	}
	return contexto
}

func mutarContextoConsumoCargaDirecta(
	mutar mutacionContextoCargaDirectaPrueba,
) func(*ports.SolicitudConsumirReciboCargaDirecta) {
	return func(solicitud *ports.SolicitudConsumirReciboCargaDirecta) {
		solicitud.Contexto = contextoCargaDirectaPrueba(true, mutar)
	}
}

func emitirReciboCargaDirectaPrueba(
	t *testing.T,
	ctx context.Context,
	adaptador ports.EmisorReciboCargaDirecta,
	reloj *relojCargaDirectaPrueba,
) ports.ReciboCargaDirecta {
	t.Helper()
	return emitirReciboCargaDirectaConContextoPrueba(
		t, ctx, adaptador, reloj, contextoPreparacionCargaDirectaPrueba(),
		"sesion-carga-supersecreta-001",
	)
}

func emitirReciboCargaDirectaConContextoPrueba(
	t *testing.T,
	ctx context.Context,
	adaptador ports.EmisorReciboCargaDirecta,
	reloj *relojCargaDirectaPrueba,
	contexto ports.ContextoOperacionAlmacen,
	sesionRef string,
) ports.ReciboCargaDirecta {
	t.Helper()
	solicitud := ports.SolicitudPrepararCargaDirecta{
		Contexto:          contexto,
		ClaveIdempotencia: "hmac-sha256:idempotencia_v1:" + strings.Repeat("c", 64),
		MIME:              "application/pdf",
		Tamano:            4096,
		HuellaSHA256:      strings.Repeat("d", 64),
		ExpiraEn:          reloj.Ahora().UTC().Add(5 * time.Minute),
	}
	instrucciones, err := ports.NuevasInstruccionesCargaDirectaParaSolicitud(
		solicitud,
		"almacen-prueba",
		sesionRef,
		ports.MetodoCargaDirectaPUT,
		"https://almacen.interno.test/carga/objeto-opaco",
		[]ports.CabeceraCargaDirecta{{Nombre: "content-type", Valor: "application/pdf"}},
		reloj.Ahora().UTC(),
	)
	if err != nil {
		t.Fatalf("crear instrucciones: %v", err)
	}
	capacidades := ports.CapacidadesAlmacenObjetos{
		ConectorID: "almacen-prueba", CargaDirectaTemporal: true,
		TamanoMaximoObjeto:   10 << 20,
		OrigenesCargaDirecta: []string{"https://almacen.interno.test"},
	}
	recibo, err := instrucciones.EmitirReciboConfirmacion(ctx, solicitud, capacidades, adaptador)
	if err != nil {
		t.Fatalf("emitir recibo: %v", err)
	}
	return recibo
}

func solicitudConsumoCargaDirectaPrueba(recibo ports.ReciboCargaDirecta) ports.SolicitudConsumirReciboCargaDirecta {
	return ports.SolicitudConsumirReciboCargaDirecta{
		Contexto:    contextoConfirmacionCargaDirectaPrueba(),
		SesionRef:   "sesion-carga-supersecreta-001",
		Recibo:      recibo,
		ValidaHasta: time.Date(2026, 7, 15, 8, 35, 0, 0, time.UTC),
	}
}

func TestConfiguracionCargaDirectaExigeCuatroClavesSeparadasYDependencias(t *testing.T) {
	base := configuracionCargaDirectaPrueba()
	repositorio := nuevoRepositorioRecibosCargaDirectaPrueba()
	reloj := &relojCargaDirectaPrueba{instante: time.Now().UTC()}

	pruebas := []struct {
		nombre        string
		configuracion ConfiguracionCriptografiaCargaDirecta
		repositorio   ports.RepositorioRecibosCargaDirecta
		reloj         ports.Reloj
	}{
		{nombre: "clave corta", configuracion: func() ConfiguracionCriptografiaCargaDirecta {
			c := base
			c.ClaveIndiceRecibo.Material = []byte("corta")
			return c
		}(), repositorio: repositorio, reloj: reloj},
		{nombre: "identificador repetido", configuracion: func() ConfiguracionCriptografiaCargaDirecta {
			c := base
			c.ClaveIndiceRecibo.Identificador = c.ClaveSeudonimizacion.Identificador
			return c
		}(), repositorio: repositorio, reloj: reloj},
		{nombre: "material repetido", configuracion: func() ConfiguracionCriptografiaCargaDirecta {
			c := base
			c.ClaveAtestacion.Material = append([]byte(nil), c.ClaveIndiceRecibo.Material...)
			return c
		}(), repositorio: repositorio, reloj: reloj},
		{nombre: "identificador no canonico", configuracion: func() ConfiguracionCriptografiaCargaDirecta {
			c := base
			c.ClaveVinculoRecibo.Identificador = "Vinculo:1"
			return c
		}(), repositorio: repositorio, reloj: reloj},
		{nombre: "repositorio ausente", configuracion: base, reloj: reloj},
		{nombre: "reloj ausente", configuracion: base, repositorio: repositorio},
		{nombre: "repositorio nulo tipado", configuracion: base, repositorio: (*repositorioRecibosCargaDirectaPrueba)(nil), reloj: reloj},
		{nombre: "reloj nulo tipado", configuracion: base, repositorio: repositorio, reloj: (*relojCargaDirectaPrueba)(nil)},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			if _, err := NuevoAdaptadorCriptograficoCargaDirecta(
				prueba.configuracion, prueba.repositorio, prueba.reloj,
			); !errors.Is(err, ErrConfiguracionCriptografiaCargaDirectaInvalida) {
				t.Fatalf("se esperaba configuracion denegada, recibido %v", err)
			}
		})
	}
}

func TestAdaptadorCargaDirectaCopiaDefensivamenteLasClaves(t *testing.T) {
	configuracion := configuracionCargaDirectaPrueba()
	repositorio := nuevoRepositorioRecibosCargaDirectaPrueba()
	reloj := &relojCargaDirectaPrueba{instante: time.Date(2026, 7, 15, 8, 30, 0, 0, time.UTC)}
	adaptador, err := NuevoAdaptadorCriptograficoCargaDirecta(configuracion, repositorio, reloj)
	if err != nil {
		t.Fatal(err)
	}
	for _, material := range [][]byte{
		configuracion.ClaveSeudonimizacion.Material,
		configuracion.ClaveIndiceRecibo.Material,
		configuracion.ClaveVinculoRecibo.Material,
		configuracion.ClaveAtestacion.Material,
	} {
		for indice := range material {
			material[indice] = 0
		}
	}

	solicitud, err := ports.NuevaSolicitudSeudonimizarSujetoAlmacen("persona:001", "carga:001")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := adaptador.SeudonimizarSujetoAlmacen(context.Background(), solicitud); err != nil {
		t.Fatalf("la mutacion externa altero las copias internas: %v", err)
	}
	emitirReciboCargaDirectaPrueba(t, context.Background(), adaptador, reloj)
}

func TestSeudonimizacionCargaDirectaEsDeterministaAcotadaYNoRevelaSujeto(t *testing.T) {
	adaptador, _, _ := nuevoAdaptadorCargaDirectaPrueba(t)
	primeraSolicitud, _ := ports.NuevaSolicitudSeudonimizarSujetoAlmacen("dni-interno:12345678Z", "bolsas:carga:001")
	segundaSolicitud, _ := ports.NuevaSolicitudSeudonimizarSujetoAlmacen("dni-interno:12345678Z", "dietas:carga:001")
	primero, err := adaptador.SeudonimizarSujetoAlmacen(context.Background(), primeraSolicitud)
	if err != nil {
		t.Fatal(err)
	}
	repetido, err := adaptador.SeudonimizarSujetoAlmacen(context.Background(), primeraSolicitud)
	if err != nil {
		t.Fatal(err)
	}
	segundoAmbito, err := adaptador.SeudonimizarSujetoAlmacen(context.Background(), segundaSolicitud)
	if err != nil {
		t.Fatal(err)
	}
	if primero != repetido || primero == segundoAmbito {
		t.Fatalf("seudonimos sin determinismo o sin separacion por ambito")
	}
	if !strings.HasPrefix(primero, "hmac-sha256:seudonimo_v1:") ||
		strings.Contains(primero, "12345678Z") {
		t.Fatalf("seudonimo no opaco: %q", primero)
	}
}

func TestSeudonimizacionCargaDirectaFallaCerradaConCancelacionEInvalidos(t *testing.T) {
	adaptador, _, _ := nuevoAdaptadorCargaDirectaPrueba(t)
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	solicitud, _ := ports.NuevaSolicitudSeudonimizarSujetoAlmacen("persona:001", "carga:001")
	if _, err := adaptador.SeudonimizarSujetoAlmacen(ctx, solicitud); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelacion no propagada: %v", err)
	}
	if _, err := adaptador.SeudonimizarSujetoAlmacen(context.Background(), ports.SolicitudSeudonimizarSujetoAlmacen{}); !errors.Is(err, ports.ErrSeudonimizacionAlmacenNoDisponible) {
		t.Fatalf("solicitud vacia aceptada: %v", err)
	}
	if _, err := adaptador.SeudonimizarSujetoAlmacen(nil, solicitud); !errors.Is(err, ErrMaterialCriptograficoCargaDirectaInvalido) {
		t.Fatalf("contexto nulo aceptado: %v", err)
	}
}

func TestCargaDirectaRechazaFormasNoCanonicasSinNormalizarlas(t *testing.T) {
	if _, err := ports.NuevoReciboCargaDirecta(" recibo-opaco "); !errors.Is(err, ports.ErrReciboCargaDirectaNoValido) {
		t.Fatalf("recibo con espacios normalizado: %v", err)
	}
	for _, datos := range [][2]string{
		{" persona:001", "carga:001"},
		{"persona:001", "carga:001 "},
		{"persona:001", "ambito con espacios"},
	} {
		if _, err := ports.NuevaSolicitudSeudonimizarSujetoAlmacen(datos[0], datos[1]); !errors.Is(err, ports.ErrSeudonimizacionAlmacenNoDisponible) {
			t.Fatalf("seudonimizacion no canonica normalizada: %q/%q error=%v", datos[0], datos[1], err)
		}
	}
	instante := time.Date(2026, 7, 15, 8, 30, 0, 0, time.UTC)
	pruebas := []struct {
		nombre, conector, sesion, destino string
		cabeceras                         []ports.CabeceraCargaDirecta
	}{
		{nombre: "conector", conector: " almacen", sesion: "sesion", destino: "https://almacen.test/carga"},
		{nombre: "sesion", conector: "almacen", sesion: "sesion ", destino: "https://almacen.test/carga"},
		{nombre: "destino", conector: "almacen", sesion: "sesion", destino: " https://almacen.test/carga"},
		{nombre: "host mayusculas", conector: "almacen", sesion: "sesion", destino: "https://Almacen.test/carga"},
		{nombre: "cabecera mayusculas", conector: "almacen", sesion: "sesion", destino: "https://almacen.test/carga", cabeceras: []ports.CabeceraCargaDirecta{{Nombre: "Content-Type", Valor: "application/pdf"}}},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			if _, err := ports.NuevasInstruccionesCargaDirecta(
				prueba.conector, prueba.sesion, ports.MetodoCargaDirectaPUT, prueba.destino,
				prueba.cabeceras, instante, instante.Add(time.Minute), 1024,
			); !errors.Is(err, ports.ErrInstruccionesCargaDirectaNoValidas) {
				t.Fatalf("forma no canonica aceptada: %v", err)
			}
		})
	}
}

func TestReciboCargaDirectaAportaAlMenos256BitsAleatoriosYAltaSinSecreto(t *testing.T) {
	adaptador, repositorio, reloj := nuevoAdaptadorCargaDirectaPrueba(t)
	primero := emitirReciboCargaDirectaPrueba(t, context.Background(), adaptador, reloj)
	segundo := emitirReciboCargaDirectaPrueba(t, context.Background(), adaptador, reloj)
	valorPrimero, _ := primero.RevelarParaEntregaOConsumo()
	valorSegundo, _ := segundo.RevelarParaEntregaOConsumo()
	if valorPrimero == valorSegundo {
		t.Fatal("dos recibos compartieron secreto")
	}
	partes := strings.Split(valorPrimero, ".")
	if len(partes) != 8 {
		t.Fatalf("formato inesperado")
	}
	aleatorio, err := base64.RawURLEncoding.DecodeString(partes[2])
	if err != nil || len(aleatorio)*8 < 256 {
		t.Fatalf("entropia aleatoria insuficiente: bytes=%d err=%v", len(aleatorio), err)
	}

	repositorio.mu.Lock()
	defer repositorio.mu.Unlock()
	if len(repositorio.entradas) != 2 {
		t.Fatalf("altas=%d", len(repositorio.entradas))
	}
	proyeccionPreparacion, _ := contextoPreparacionCargaDirectaPrueba().Proyeccion()
	for indice, entrada := range repositorio.entradas {
		persistido := strings.Join([]string{
			indice, entrada.registro.VinculoHMAC, entrada.registro.EvidenciaAltaRef,
		}, "|")
		for _, secreto := range []string{
			valorPrimero, valorSegundo, "sesion-carga-supersecreta-001",
			proyeccionPreparacion.OperacionRef,
		} {
			if strings.Contains(persistido, secreto) {
				t.Fatalf("el repositorio recibio un secreto o contexto en claro")
			}
		}
		if entrada.registro.Validar() != nil {
			t.Fatal("registro persistido no valido")
		}
	}
}

func TestReciboCargaDirectaSeConsumeUnaSolaVezYProduceEvidenciaOpaca(t *testing.T) {
	adaptador, repositorio, reloj := nuevoAdaptadorCargaDirectaPrueba(t)
	recibo := emitirReciboCargaDirectaPrueba(t, context.Background(), adaptador, reloj)
	solicitud := solicitudConsumoCargaDirectaPrueba(recibo)
	comprobante, err := adaptador.ConsumirReciboCargaDirecta(context.Background(), solicitud)
	if err != nil {
		t.Fatalf("primer consumo: %v", err)
	}
	confirmacion, err := ports.NuevaSolicitudConfirmarCargaDirecta(
		context.Background(), solicitud.Contexto, solicitud.SesionRef, comprobante, adaptador,
	)
	if err != nil {
		t.Fatalf("atestacion invalida: %v", err)
	}
	_, _, intencion, huellaIntencion, referencia, emitidoEn, consumidoEn, expiraEn, validaHasta, err := confirmacion.RevelarParaConector()
	if err != nil {
		t.Fatalf("confirmacion no revelable: %v", err)
	}
	if !strings.HasPrefix(intencion, "confirmacion-intencion-v1:") ||
		!strings.HasPrefix(referencia, "recibo-consumo-v1:") ||
		!strings.HasPrefix(huellaIntencion, "hmac-sha256:atestacion_recibo_v1:") ||
		!consumidoEn.Equal(repositorio.horaDurable) || emitidoEn.After(consumidoEn) ||
		!expiraEn.After(consumidoEn) || !validaHasta.After(consumidoEn) {
		t.Fatalf("evidencia no opaca o fecha incorrecta")
	}
	if _, err := adaptador.ConsumirReciboCargaDirecta(context.Background(), solicitud); !errors.Is(err, ports.ErrReciboCargaDirectaNoValido) {
		t.Fatalf("repeticion no denegada uniformemente: %v", err)
	}
	_, consumos, _, usados := repositorio.estado()
	if consumos != 2 || usados != 1 {
		t.Fatalf("consumos=%d usados=%d", consumos, usados)
	}
}

func TestAtestacionImpideForjarComprobanteOCambiarContextoCompleto(t *testing.T) {
	adaptador, _, reloj := nuevoAdaptadorCargaDirectaPrueba(t)
	recibo := emitirReciboCargaDirectaPrueba(t, context.Background(), adaptador, reloj)
	solicitud := solicitudConsumoCargaDirectaPrueba(recibo)
	comprobante, err := adaptador.ConsumirReciboCargaDirecta(context.Background(), solicitud)
	if err != nil {
		t.Fatal(err)
	}
	indice, grupo, vinculo, evidencia, intencion, huellaIntencion,
		emitidoEn, consumidoEn, expiraEn, _, atestacionReal, err := comprobante.RevelarParaVerificacion()
	if err != nil {
		t.Fatal(err)
	}
	resultado := ports.ResultadoConsumoReciboCargaDirecta{
		IndiceHMAC: indice, GrupoHMAC: grupo, VinculoHMAC: vinculo,
		EvidenciaConsumoRef: evidencia, IntencionConfirmacionRef: intencion,
		HuellaIntencionHMAC: huellaIntencion, RegistradoEn: emitidoEn,
		ConsumidoEn: consumidoEn, ExpiraEn: expiraEn,
	}
	atestacionForjada := "hmac-sha256:atestacion_recibo_v1:" + strings.Repeat("0", 64)
	forjado, err := ports.NuevoComprobanteConsumoReciboCargaDirecta(solicitud, resultado, atestacionForjada)
	if err != nil {
		t.Fatalf("precondicion de forma HMAC: %v", err)
	}
	if _, err := ports.NuevaSolicitudConfirmarCargaDirecta(
		context.Background(), solicitud.Contexto, solicitud.SesionRef, forjado, adaptador,
	); !errors.Is(err, ports.ErrAtestacionReciboCargaDirectaNoValida) {
		t.Fatalf("comprobante forjado aceptado: %v", err)
	}

	contextoAlterado := contextoCargaDirectaPrueba(true, func(
		d *domain.DecisionAutorizacion, _ *domain.RecursoAutorizable, _ *ports.VinculosOperacionAlmacen,
	) {
		d.DecisionRef = "autorizacion:confirmar:otra"
	})
	if _, err := ports.NuevaSolicitudConfirmarCargaDirecta(
		context.Background(), contextoAlterado, solicitud.SesionRef, comprobante, adaptador,
	); !errors.Is(err, ports.ErrAtestacionReciboCargaDirectaNoValida) {
		t.Fatalf("autorizacion no atestada aceptada: %v", err)
	}
	contextoAlterado = contextoPreparacionCargaDirectaPrueba()
	if _, err := ports.NuevaSolicitudConfirmarCargaDirecta(
		context.Background(), contextoAlterado, solicitud.SesionRef, comprobante, adaptador,
	); !errors.Is(err, ports.ErrSolicitudAlmacenInvalida) {
		t.Fatalf("accion fuera de lista positiva aceptada: %v", err)
	}

	resultado.EvidenciaConsumoRef = "evidencia:consumo:alterada"
	alterado, err := ports.NuevoComprobanteConsumoReciboCargaDirecta(solicitud, resultado, atestacionReal)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ports.NuevaSolicitudConfirmarCargaDirecta(
		context.Background(), solicitud.Contexto, solicitud.SesionRef, alterado, adaptador,
	); !errors.Is(err, ports.ErrAtestacionReciboCargaDirectaNoValida) {
		t.Fatalf("evidencia durable alterada aceptada: %v", err)
	}
	resultado.EvidenciaConsumoRef = evidencia
	resultado.ConsumidoEn = consumidoEn.Add(time.Nanosecond)
	alterado, err = ports.NuevoComprobanteConsumoReciboCargaDirecta(solicitud, resultado, atestacionReal)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ports.NuevaSolicitudConfirmarCargaDirecta(
		context.Background(), solicitud.Contexto, solicitud.SesionRef, alterado, adaptador,
	); !errors.Is(err, ports.ErrAtestacionReciboCargaDirectaNoValida) {
		t.Fatalf("fecha durable alterada aceptada: %v", err)
	}
	resultado.ConsumidoEn = consumidoEn
	pruebas := []struct {
		nombre string
		mutar  func(*ports.ResultadoConsumoReciboCargaDirecta, *ports.SolicitudConsumirReciboCargaDirecta)
	}{
		{"alta durable", func(r *ports.ResultadoConsumoReciboCargaDirecta, _ *ports.SolicitudConsumirReciboCargaDirecta) {
			r.RegistradoEn = r.RegistradoEn.Add(-time.Nanosecond)
		}},
		{"caducidad", func(r *ports.ResultadoConsumoReciboCargaDirecta, _ *ports.SolicitudConsumirReciboCargaDirecta) {
			r.ExpiraEn = r.ExpiraEn.Add(time.Nanosecond)
		}},
		{"intencion", func(r *ports.ResultadoConsumoReciboCargaDirecta, _ *ports.SolicitudConsumirReciboCargaDirecta) {
			r.IntencionConfirmacionRef = "confirmacion-intencion-v1:alterada"
		}},
		{"huella de intencion", func(r *ports.ResultadoConsumoReciboCargaDirecta, _ *ports.SolicitudConsumirReciboCargaDirecta) {
			r.HuellaIntencionHMAC = "hmac-sha256:atestacion_recibo_v1:" + strings.Repeat("f", 64)
		}},
		{"limite externo", func(_ *ports.ResultadoConsumoReciboCargaDirecta, s *ports.SolicitudConsumirReciboCargaDirecta) {
			s.ValidaHasta = s.ValidaHasta.Add(time.Nanosecond)
		}},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			resultadoAlterado := resultado
			solicitudAlterada := solicitud
			prueba.mutar(&resultadoAlterado, &solicitudAlterada)
			comprobanteAlterado, err := ports.NuevoComprobanteConsumoReciboCargaDirecta(
				solicitudAlterada, resultadoAlterado, atestacionReal,
			)
			if err != nil {
				t.Fatalf("precondicion: %v", err)
			}
			if _, err := ports.NuevaSolicitudConfirmarCargaDirecta(
				context.Background(), solicitud.Contexto, solicitud.SesionRef, comprobanteAlterado, adaptador,
			); !errors.Is(err, ports.ErrAtestacionReciboCargaDirectaNoValida) {
				t.Fatalf("campo alterado aceptado: %v", err)
			}
		})
	}
	if _, err := ports.NuevaSolicitudConfirmarCargaDirecta(
		context.Background(), solicitud.Contexto, solicitud.SesionRef, comprobante, nil,
	); !errors.Is(err, ports.ErrSolicitudAlmacenInvalida) {
		t.Fatalf("fabrica sin verificador aceptada: %v", err)
	}
}

func TestAltaNuevaSustituyeAtomicamenteReciboActivoDelGrupo(t *testing.T) {
	adaptador, repositorio, reloj := nuevoAdaptadorCargaDirectaPrueba(t)
	primero := emitirReciboCargaDirectaPrueba(t, context.Background(), adaptador, reloj)
	mutarSegundo := func(
		d *domain.DecisionAutorizacion, r *domain.RecursoAutorizable, v *ports.VinculosOperacionAlmacen,
	) {
		v.HuellaSolicitudHMAC = "hmac-sha256:solicitud_v1:" + strings.Repeat("e", 64)
		r.Referencia = "expediente:revisado:002"
		if d.Accion == ports.AccionNegocioPrepararCargaDocumental {
			d.DecisionRef = "autorizacion:preparar:reemitida:002"
		} else {
			d.DecisionRef = "autorizacion:confirmar:revisada:003"
		}
	}
	contextoSegundo := contextoCargaDirectaPrueba(false, mutarSegundo)
	segundo := emitirReciboCargaDirectaConContextoPrueba(
		t, context.Background(), adaptador, reloj, contextoSegundo,
		"sesion-carga-supersecreta-001",
	)
	if _, err := adaptador.ConsumirReciboCargaDirecta(
		context.Background(), solicitudConsumoCargaDirectaPrueba(primero),
	); !errors.Is(err, ports.ErrReciboCargaDirectaNoValido) {
		t.Fatalf("recibo sustituido siguio activo: %v", err)
	}
	contextoConfirmacionSegundo := contextoCargaDirectaPrueba(true, mutarSegundo)
	if _, err := adaptador.ConsumirReciboCargaDirecta(
		context.Background(), ports.SolicitudConsumirReciboCargaDirecta{
			Contexto: contextoConfirmacionSegundo, SesionRef: "sesion-carga-supersecreta-001", Recibo: segundo,
			ValidaHasta: time.Date(2026, 7, 15, 8, 35, 0, 0, time.UTC),
		},
	); err != nil {
		t.Fatalf("ultimo recibo del grupo no quedo activo: %v", err)
	}
	_, _, entradas, usados := repositorio.estado()
	if entradas != 2 || usados != 1 {
		t.Fatalf("entradas=%d usados=%d", entradas, usados)
	}
	proyeccionSegundo, _ := contextoSegundo.Proyeccion()
	proyeccionPrimero, _ := contextoPreparacionCargaDirectaPrueba().Proyeccion()
	if len(repositorio.resultadosAlta) != 2 || repositorio.resultadosAlta[1].Predecesor == nil ||
		repositorio.resultadosAlta[1].AutorizacionEmisionRef != proyeccionSegundo.AutorizacionRef ||
		repositorio.resultadosAlta[1].Predecesor.AutorizacionEmisionRef != proyeccionPrimero.AutorizacionRef {
		t.Fatal("la sustitucion no conservo autorizaciones y predecesor tipados")
	}
}

func TestReciboCargaDirectaVinculaSesionYTodosLosInvariantesCompartidos(t *testing.T) {
	adaptador, _, reloj := nuevoAdaptadorCargaDirectaPrueba(t)
	recibo := emitirReciboCargaDirectaPrueba(t, context.Background(), adaptador, reloj)
	base := solicitudConsumoCargaDirectaPrueba(recibo)
	mutaciones := map[string]func(*ports.SolicitudConsumirReciboCargaDirecta){
		"sesion": func(s *ports.SolicitudConsumirReciboCargaDirecta) { s.SesionRef = "sesion-distinta" },
		"operacion": mutarContextoConsumoCargaDirecta(func(_ *domain.DecisionAutorizacion, _ *domain.RecursoAutorizable, v *ports.VinculosOperacionAlmacen) {
			v.OperacionRef = "operacion:otra"
		}),
		"correlacion": mutarContextoConsumoCargaDirecta(func(d *domain.DecisionAutorizacion, _ *domain.RecursoAutorizable, _ *ports.VinculosOperacionAlmacen) {
			d.CorrelacionRef = "correlacion:otra"
		}),
		"finalidad": mutarContextoConsumoCargaDirecta(func(d *domain.DecisionAutorizacion, _ *domain.RecursoAutorizable, _ *ports.VinculosOperacionAlmacen) {
			d.Finalidad = "otra_finalidad"
		}),
		"clasificacion": mutarContextoConsumoCargaDirecta(func(_ *domain.DecisionAutorizacion, _ *domain.RecursoAutorizable, v *ports.VinculosOperacionAlmacen) {
			v.Clasificacion = "restringida"
		}),
		"carga": mutarContextoConsumoCargaDirecta(func(_ *domain.DecisionAutorizacion, _ *domain.RecursoAutorizable, v *ports.VinculosOperacionAlmacen) {
			v.CargaRef = "carga:otra"
		}),
		"seudonimo": mutarContextoConsumoCargaDirecta(func(_ *domain.DecisionAutorizacion, _ *domain.RecursoAutorizable, v *ports.VinculosOperacionAlmacen) {
			v.SujetoSeudonimoHMAC = "hmac-sha256:seudonimo_v1:" + strings.Repeat("e", 64)
		}),
		"recurso": mutarContextoConsumoCargaDirecta(func(_ *domain.DecisionAutorizacion, r *domain.RecursoAutorizable, _ *ports.VinculosOperacionAlmacen) {
			r.Referencia = "expediente:otro"
		}),
		"modulo": mutarContextoConsumoCargaDirecta(func(_ *domain.DecisionAutorizacion, r *domain.RecursoAutorizable, _ *ports.VinculosOperacionAlmacen) {
			r.ModuloID = "seleccion"
		}),
		"huella solicitud": mutarContextoConsumoCargaDirecta(func(_ *domain.DecisionAutorizacion, _ *domain.RecursoAutorizable, v *ports.VinculosOperacionAlmacen) {
			v.HuellaSolicitudHMAC = "hmac-sha256:solicitud_v1:" + strings.Repeat("f", 64)
		}),
	}
	for nombre, mutar := range mutaciones {
		t.Run(nombre, func(t *testing.T) {
			solicitud := base
			mutar(&solicitud)
			if _, err := adaptador.ConsumirReciboCargaDirecta(context.Background(), solicitud); !errors.Is(err, ports.ErrReciboCargaDirectaNoValido) {
				t.Fatalf("mutacion aceptada: %v", err)
			}
		})
	}
	if _, err := adaptador.ConsumirReciboCargaDirecta(context.Background(), base); err != nil {
		t.Fatalf("las denegaciones alteraron el recibo original: %v", err)
	}
}

func TestReciboCargaDirectaSoloAdmiteAccionYContextoPositivosExactos(t *testing.T) {
	adaptador, repositorio, reloj := nuevoAdaptadorCargaDirectaPrueba(t)
	recibo := emitirReciboCargaDirectaPrueba(t, context.Background(), adaptador, reloj)
	base := solicitudConsumoCargaDirectaPrueba(recibo)
	pruebas := map[string]func(*ports.SolicitudConsumirReciboCargaDirecta){
		"accion distinta": func(s *ports.SolicitudConsumirReciboCargaDirecta) {
			s.Contexto = contextoPreparacionCargaDirectaPrueba()
		},
		"accion desconocida": func(s *ports.SolicitudConsumirReciboCargaDirecta) {
			s.Contexto = ports.ContextoOperacionAlmacen{}
		},
		"autorizacion ausente": func(s *ports.SolicitudConsumirReciboCargaDirecta) {
			s.Contexto = ports.ContextoOperacionAlmacen{}
		},
		"modulo ausente": func(s *ports.SolicitudConsumirReciboCargaDirecta) {
			s.Contexto = ports.ContextoOperacionAlmacen{}
		},
		"recibo ausente": func(s *ports.SolicitudConsumirReciboCargaDirecta) {
			s.Recibo = ports.ReciboCargaDirecta{}
		},
	}
	for nombre, mutar := range pruebas {
		t.Run(nombre, func(t *testing.T) {
			solicitud := base
			mutar(&solicitud)
			if _, err := adaptador.ConsumirReciboCargaDirecta(context.Background(), solicitud); !errors.Is(err, ports.ErrReciboCargaDirectaNoValido) {
				t.Fatalf("contexto no permitido aceptado: %v", err)
			}
		})
	}
	_, consumos, _, usados := repositorio.estado()
	if consumos != 0 || usados != 0 {
		t.Fatalf("una denegacion alcanzo el repositorio: consumos=%d usados=%d", consumos, usados)
	}
}

func TestReciboCargaDirectaDeniegaTodaAlteracionDelValor(t *testing.T) {
	adaptador, _, reloj := nuevoAdaptadorCargaDirectaPrueba(t)
	recibo := emitirReciboCargaDirectaPrueba(t, context.Background(), adaptador, reloj)
	valor, _ := recibo.RevelarParaEntregaOConsumo()
	base := strings.Split(valor, ".")
	alteraciones := map[string]func([]string) []string{
		"version":          func(p []string) []string { p[0] = "rcd1"; return p },
		"clave":            func(p []string) []string { p[1] = "otra_clave"; return p },
		"aleatorio":        func(p []string) []string { p[2] = alterarPrimerCaracterCargaDirecta(p[2], 'A', 'B'); return p },
		"huella solicitud": func(p []string) []string { p[3] = alterarPrimerCaracterCargaDirecta(p[3], '0', '1'); return p },
		"caducidad":        func(p []string) []string { p[4] = p[4] + "1"; return p },
		"mac":              func(p []string) []string { p[5] = alterarPrimerCaracterCargaDirecta(p[5], '0', '1'); return p },
		"alta durable":     func(p []string) []string { p[6] = p[6] + "1"; return p },
		"atestacion alta":  func(p []string) []string { p[7] = alterarPrimerCaracterCargaDirecta(p[7], '0', '1'); return p },
		"campo adicional":  func(p []string) []string { return append(p, "extra") },
	}
	for nombre, alterar := range alteraciones {
		t.Run(nombre, func(t *testing.T) {
			partes := append([]string(nil), base...)
			valorAlterado := strings.Join(alterar(partes), ".")
			reciboAlterado, err := ports.NuevoReciboCargaDirecta(valorAlterado)
			if err != nil {
				t.Fatalf("precondicion del recibo opaco: %v", err)
			}
			solicitud := solicitudConsumoCargaDirectaPrueba(reciboAlterado)
			if _, err := adaptador.ConsumirReciboCargaDirecta(context.Background(), solicitud); !errors.Is(err, ports.ErrReciboCargaDirectaNoValido) {
				t.Fatalf("alteracion aceptada: %v", err)
			}
		})
	}
}

func TestReciboCargaDirectaUsaCaducidadYHoraAutoritativaDelRepositorio(t *testing.T) {
	t.Run("alta durable autenticada no procede del reloj del proceso", func(t *testing.T) {
		adaptador, repositorio, reloj := nuevoAdaptadorCargaDirectaPrueba(t)
		repositorio.horaDurable = reloj.Ahora().Add(30 * time.Second)
		recibo := emitirReciboCargaDirectaPrueba(t, context.Background(), adaptador, reloj)
		comprobante, err := adaptador.ConsumirReciboCargaDirecta(
			context.Background(), solicitudConsumoCargaDirectaPrueba(recibo),
		)
		if err != nil {
			t.Fatal(err)
		}
		_, _, _, _, _, _, registradoEn, consumidoEn, _, _, _, err := comprobante.RevelarParaVerificacion()
		if err != nil || !registradoEn.Equal(repositorio.horaDurable) || registradoEn.Equal(reloj.Ahora()) ||
			!consumidoEn.Equal(repositorio.horaDurable) {
			t.Fatalf("alta=%s consumo=%s durable=%s proceso=%s error=%v",
				registradoEn, consumidoEn, repositorio.horaDurable, reloj.Ahora(), err)
		}
	})
	t.Run("desfase del reloj de proceso no amplia diez minutos durables", func(t *testing.T) {
		adaptador, repositorio, reloj := nuevoAdaptadorCargaDirectaPrueba(t)
		reloj.fijar(repositorio.horaDurable.Add(9 * time.Minute))
		solicitud := ports.SolicitudPrepararCargaDirecta{
			Contexto: contextoPreparacionCargaDirectaPrueba(), ClaveIdempotencia: "idempotencia:desfase",
			MIME: "application/pdf", Tamano: 1, HuellaSHA256: strings.Repeat("d", 64),
			ExpiraEn: reloj.Ahora().Add(5 * time.Minute),
		}
		instrucciones, err := ports.NuevasInstruccionesCargaDirectaParaSolicitud(
			solicitud, "almacen-prueba", "sesion-carga-supersecreta-001", ports.MetodoCargaDirectaPUT,
			"https://almacen.interno.test/carga/desfase", nil, reloj.Ahora(),
		)
		if err != nil {
			t.Fatal(err)
		}
		_, err = instrucciones.EmitirReciboConfirmacion(context.Background(), solicitud, ports.CapacidadesAlmacenObjetos{
			ConectorID: "almacen-prueba", CargaDirectaTemporal: true, TamanoMaximoObjeto: 10,
			OrigenesCargaDirecta: []string{"https://almacen.interno.test"},
		}, adaptador)
		if !errors.Is(err, ports.ErrReciboCargaDirectaNoDisponible) {
			t.Fatalf("el reloj de proceso amplio la vigencia durable: %v", err)
		}
	})
	t.Run("revalida la capacidad con el reloj inyectado antes del efecto", func(t *testing.T) {
		adaptador, repositorio, reloj := nuevoAdaptadorCargaDirectaPrueba(t)
		recibo := emitirReciboCargaDirectaPrueba(t, context.Background(), adaptador, reloj)
		reloj.fijar(reloj.Ahora().Add(30 * time.Minute))
		if _, err := adaptador.ConsumirReciboCargaDirecta(
			context.Background(), solicitudConsumoCargaDirectaPrueba(recibo),
		); !errors.Is(err, ports.ErrReciboCargaDirectaNoValido) {
			t.Fatalf("una capacidad caducada alcanzo el efecto: %v", err)
		}
		_, consumos, _, usados := repositorio.estado()
		if consumos != 0 || usados != 0 {
			t.Fatalf("consumos=%d usados=%d", consumos, usados)
		}
	})
	t.Run("condicion durable autoritativa", func(t *testing.T) {
		adaptador, repositorio, reloj := nuevoAdaptadorCargaDirectaPrueba(t)
		recibo := emitirReciboCargaDirectaPrueba(t, context.Background(), adaptador, reloj)
		repositorio.horaDurable = reloj.Ahora().Add(5 * time.Minute)
		if _, err := adaptador.ConsumirReciboCargaDirecta(context.Background(), solicitudConsumoCargaDirectaPrueba(recibo)); !errors.Is(err, ports.ErrReciboCargaDirectaNoValido) {
			t.Fatalf("el repositorio no impuso su condicion atomica: %v", err)
		}
	})
	t.Run("limite de autorizacion y sesion se aplica en la escritura durable", func(t *testing.T) {
		adaptador, repositorio, reloj := nuevoAdaptadorCargaDirectaPrueba(t)
		recibo := emitirReciboCargaDirectaPrueba(t, context.Background(), adaptador, reloj)
		solicitud := solicitudConsumoCargaDirectaPrueba(recibo)
		solicitud.ValidaHasta = repositorio.horaDurable
		if _, err := adaptador.ConsumirReciboCargaDirecta(context.Background(), solicitud); !errors.Is(err, ports.ErrReciboCargaDirectaNoValido) {
			t.Fatalf("el limite externo no se revalido con hora durable: %v", err)
		}
		_, _, _, usados := repositorio.estado()
		if usados != 0 {
			t.Fatal("el recibo se consumio con autorizacion o sesion vencida")
		}
	})
}

func TestConsumoConcurrenteDeReciboCargaDirectaSoloAdmiteUno(t *testing.T) {
	adaptador, repositorio, reloj := nuevoAdaptadorCargaDirectaPrueba(t)
	recibo := emitirReciboCargaDirectaPrueba(t, context.Background(), adaptador, reloj)
	solicitud := solicitudConsumoCargaDirectaPrueba(recibo)
	const trabajadores = 48
	var exitos atomic.Int32
	var fallosInesperados atomic.Int32
	inicio := make(chan struct{})
	var grupo sync.WaitGroup
	for indice := 0; indice < trabajadores; indice++ {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			<-inicio
			_, err := adaptador.ConsumirReciboCargaDirecta(context.Background(), solicitud)
			if err == nil {
				exitos.Add(1)
				return
			}
			if !errors.Is(err, ports.ErrReciboCargaDirectaNoValido) {
				fallosInesperados.Add(1)
			}
		}()
	}
	close(inicio)
	grupo.Wait()
	if exitos.Load() != 1 || fallosInesperados.Load() != 0 {
		t.Fatalf("exitos=%d fallos inesperados=%d", exitos.Load(), fallosInesperados.Load())
	}
	_, _, _, usados := repositorio.estado()
	if usados != 1 {
		t.Fatalf("usos persistidos=%d", usados)
	}
}

func TestAltaReciboCargaDirectaReintentaSoloColisionDeIndice(t *testing.T) {
	adaptador, repositorio, reloj := nuevoAdaptadorCargaDirectaPrueba(t)
	repositorio.conflictosAlta = 1
	emitirReciboCargaDirectaPrueba(t, context.Background(), adaptador, reloj)
	altas, _, entradas, _ := repositorio.estado()
	if altas != 2 || entradas != 1 {
		t.Fatalf("altas=%d entradas=%d", altas, entradas)
	}

	adaptador2, repositorio2, reloj2 := nuevoAdaptadorCargaDirectaPrueba(t)
	repositorio2.conflictosAlta = maximoIntentosAltaRecibo
	// El constructor de peticion publica envuelve el error interno, pero debe
	// seguir sin entregar ningun recibo ni insertar una entrada.
	func() {
		defer func() {
			if recuperado := recover(); recuperado != nil {
				t.Fatalf("panico inesperado: %v", recuperado)
			}
		}()
		solicitud := ports.SolicitudPrepararCargaDirecta{
			Contexto: contextoPreparacionCargaDirectaPrueba(), ClaveIdempotencia: "idempotencia:001",
			MIME: "application/pdf", Tamano: 1, HuellaSHA256: strings.Repeat("d", 64),
			ExpiraEn: reloj2.Ahora().Add(5 * time.Minute),
		}
		instrucciones, err := ports.NuevasInstruccionesCargaDirectaParaSolicitud(
			solicitud, "almacen-prueba", "sesion-carga-supersecreta-001", ports.MetodoCargaDirectaPUT,
			"https://almacen.interno.test/carga/otra", nil, reloj2.Ahora(),
		)
		if err != nil {
			t.Fatal(err)
		}
		_, err = instrucciones.EmitirReciboConfirmacion(context.Background(), solicitud, ports.CapacidadesAlmacenObjetos{
			ConectorID: "almacen-prueba", CargaDirectaTemporal: true, TamanoMaximoObjeto: 10,
			OrigenesCargaDirecta: []string{"https://almacen.interno.test"},
		}, adaptador2)
		if !errors.Is(err, ports.ErrReciboCargaDirectaNoDisponible) {
			t.Fatalf("colisiones agotadas no cerraron la emision: %v", err)
		}
	}()
	altas, _, entradas, _ = repositorio2.estado()
	if altas != maximoIntentosAltaRecibo || entradas != 0 {
		t.Fatalf("altas=%d entradas=%d", altas, entradas)
	}
}

func TestErroresDeRepositorioCargaDirectaFallanCerradosSinFiltrarCausa(t *testing.T) {
	t.Run("alta", func(t *testing.T) {
		adaptador, repositorio, reloj := nuevoAdaptadorCargaDirectaPrueba(t)
		repositorio.errorAlta = errors.New("detalle interno base de datos")
		solicitud := ports.SolicitudPrepararCargaDirecta{
			Contexto: contextoPreparacionCargaDirectaPrueba(), ClaveIdempotencia: "idempotencia:alta",
			MIME: "application/pdf", Tamano: 1, HuellaSHA256: strings.Repeat("d", 64),
			ExpiraEn: reloj.Ahora().Add(5 * time.Minute),
		}
		instrucciones, _ := ports.NuevasInstruccionesCargaDirectaParaSolicitud(
			solicitud, "almacen-prueba", "sesion-carga-supersecreta-001", ports.MetodoCargaDirectaPUT,
			"https://almacen.interno.test/carga/error", nil, reloj.Ahora(),
		)
		_, err := instrucciones.EmitirReciboConfirmacion(context.Background(), solicitud, ports.CapacidadesAlmacenObjetos{
			ConectorID: "almacen-prueba", CargaDirectaTemporal: true, TamanoMaximoObjeto: 10,
			OrigenesCargaDirecta: []string{"https://almacen.interno.test"},
		}, adaptador)
		if !errors.Is(err, ports.ErrReciboCargaDirectaNoDisponible) || strings.Contains(fmt.Sprint(err), "base de datos") {
			t.Fatalf("error no cerrado o filtrado: %v", err)
		}
	})
	t.Run("consumo", func(t *testing.T) {
		adaptador, repositorio, reloj := nuevoAdaptadorCargaDirectaPrueba(t)
		recibo := emitirReciboCargaDirectaPrueba(t, context.Background(), adaptador, reloj)
		repositorio.errorConsumo = errors.New("detalle interno transaccion")
		_, err := adaptador.ConsumirReciboCargaDirecta(context.Background(), solicitudConsumoCargaDirectaPrueba(recibo))
		if !errors.Is(err, ports.ErrReciboCargaDirectaNoDisponible) || strings.Contains(fmt.Sprint(err), "transaccion") {
			t.Fatalf("error no cerrado o filtrado: %v", err)
		}
	})
}

func TestCancelacionCargaDirectaNoCreaNiConsumeRecibos(t *testing.T) {
	adaptador, repositorio, reloj := nuevoAdaptadorCargaDirectaPrueba(t)
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()

	// EmitirReciboConfirmacion oculta deliberadamente el error interno del
	// emisor; la propiedad relevante es que no entrega ni registra un recibo.
	solicitud := ports.SolicitudPrepararCargaDirecta{
		Contexto: contextoPreparacionCargaDirectaPrueba(), ClaveIdempotencia: "idempotencia:cancelada",
		MIME: "application/pdf", Tamano: 1, HuellaSHA256: strings.Repeat("d", 64),
		ExpiraEn: reloj.Ahora().Add(5 * time.Minute),
	}
	instrucciones, _ := ports.NuevasInstruccionesCargaDirectaParaSolicitud(
		solicitud, "almacen-prueba", "sesion-carga-supersecreta-001", ports.MetodoCargaDirectaPUT,
		"https://almacen.interno.test/carga/cancelada", nil, reloj.Ahora(),
	)
	if _, err := instrucciones.EmitirReciboConfirmacion(ctx, solicitud, ports.CapacidadesAlmacenObjetos{
		ConectorID: "almacen-prueba", CargaDirectaTemporal: true, TamanoMaximoObjeto: 10,
		OrigenesCargaDirecta: []string{"https://almacen.interno.test"},
	}, adaptador); !errors.Is(err, ports.ErrReciboCargaDirectaNoDisponible) {
		t.Fatalf("emision cancelada no denegada: %v", err)
	}
	altas, _, entradas, _ := repositorio.estado()
	if altas != 0 || entradas != 0 {
		t.Fatalf("la cancelacion produjo alta")
	}

	recibo := emitirReciboCargaDirectaPrueba(t, context.Background(), adaptador, reloj)
	if _, err := adaptador.ConsumirReciboCargaDirecta(ctx, solicitudConsumoCargaDirectaPrueba(recibo)); !errors.Is(err, context.Canceled) {
		t.Fatalf("consumo cancelado no propagado: %v", err)
	}
	_, consumos, _, usados := repositorio.estado()
	if consumos != 0 || usados != 0 {
		t.Fatalf("la cancelacion produjo consumo")
	}
}

func TestCerrarCargaDirectaBorraCopiasYDeniegaTodaOperacion(t *testing.T) {
	adaptador, _, reloj := nuevoAdaptadorCargaDirectaPrueba(t)
	copiaSeudonimo := adaptador.claveSeudonimizacion.material
	copiaIndice := adaptador.claveIndiceRecibo.material
	copiaVinculo := adaptador.claveVinculoRecibo.material
	copiaAtestacion := adaptador.claveAtestacion.material
	recibo := emitirReciboCargaDirectaPrueba(t, context.Background(), adaptador, reloj)
	adaptador.Cerrar()
	adaptador.Cerrar()
	for nombre, material := range map[string][]byte{
		"seudonimo": copiaSeudonimo, "indice": copiaIndice, "vinculo": copiaVinculo,
		"atestacion": copiaAtestacion,
	} {
		for _, valor := range material {
			if valor != 0 {
				t.Fatalf("clave %s no borrada logicamente", nombre)
			}
		}
	}
	solicitudSeudonimo, _ := ports.NuevaSolicitudSeudonimizarSujetoAlmacen("persona:001", "carga:001")
	if _, err := adaptador.SeudonimizarSujetoAlmacen(context.Background(), solicitudSeudonimo); !errors.Is(err, ErrCriptografiaCargaDirectaCerrada) {
		t.Fatalf("seudonimizacion tras cierre: %v", err)
	}
	if _, err := adaptador.ConsumirReciboCargaDirecta(context.Background(), solicitudConsumoCargaDirectaPrueba(recibo)); !errors.Is(err, ErrCriptografiaCargaDirectaCerrada) {
		t.Fatalf("consumo tras cierre: %v", err)
	}
}

func TestCerrarConcurrenteCargaDirectaNoCausaCarrerasNiReabreAdaptador(t *testing.T) {
	adaptador, _, _ := nuevoAdaptadorCargaDirectaPrueba(t)
	solicitud, _ := ports.NuevaSolicitudSeudonimizarSujetoAlmacen("persona:001", "carga:001")
	const trabajadores = 32
	inicio := make(chan struct{})
	var grupo sync.WaitGroup
	for indice := 0; indice < trabajadores; indice++ {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			<-inicio
			_, err := adaptador.SeudonimizarSujetoAlmacen(context.Background(), solicitud)
			if err != nil && !errors.Is(err, ErrCriptografiaCargaDirectaCerrada) {
				t.Errorf("resultado concurrente inesperado: %v", err)
			}
		}()
	}
	close(inicio)
	adaptador.Cerrar()
	grupo.Wait()
	if _, err := adaptador.SeudonimizarSujetoAlmacen(context.Background(), solicitud); !errors.Is(err, ErrCriptografiaCargaDirectaCerrada) {
		t.Fatalf("adaptador reabierto: %v", err)
	}
}

func TestAdaptadorNoMantieneBloqueoMientrasEsperaRepositorio(t *testing.T) {
	adaptador, repositorio, reloj := nuevoAdaptadorCargaDirectaPrueba(t)
	repositorio.altaIniciada = make(chan struct{}, 1)
	repositorio.desbloquearAlta = make(chan struct{})
	resultado := make(chan error, 1)
	go func() {
		solicitud := ports.SolicitudPrepararCargaDirecta{
			Contexto: contextoPreparacionCargaDirectaPrueba(), ClaveIdempotencia: "idempotencia:bloqueo",
			MIME: "application/pdf", Tamano: 1, HuellaSHA256: strings.Repeat("d", 64),
			ExpiraEn: reloj.Ahora().Add(5 * time.Minute),
		}
		instrucciones, err := ports.NuevasInstruccionesCargaDirectaParaSolicitud(
			solicitud, "almacen-prueba", "sesion-carga-supersecreta-001", ports.MetodoCargaDirectaPUT,
			"https://almacen.interno.test/carga/bloqueo", nil, reloj.Ahora(),
		)
		if err == nil {
			_, err = instrucciones.EmitirReciboConfirmacion(context.Background(), solicitud, ports.CapacidadesAlmacenObjetos{
				ConectorID: "almacen-prueba", CargaDirectaTemporal: true, TamanoMaximoObjeto: 10,
				OrigenesCargaDirecta: []string{"https://almacen.interno.test"},
			}, adaptador)
		}
		resultado <- err
	}()
	select {
	case <-repositorio.altaIniciada:
	case <-time.After(time.Second):
		t.Fatal("el repositorio no recibio el alta")
	}
	cerrado := make(chan struct{})
	go func() {
		adaptador.Cerrar()
		close(cerrado)
	}()
	select {
	case <-cerrado:
	case <-time.After(time.Second):
		t.Fatal("Cerrar quedo bloqueado por una llamada externa al repositorio")
	}
	close(repositorio.desbloquearAlta)
	select {
	case err := <-resultado:
		if err != nil {
			t.Fatalf("operacion iniciada antes del cierre fallo: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("alta no finalizo")
	}
}

func TestSecretosCargaDirectaQuedanOcultosEnFormatoJSONYRegistros(t *testing.T) {
	configuracion := configuracionCargaDirectaPrueba()
	adaptador, _, reloj := nuevoAdaptadorCargaDirectaPrueba(t)
	recibo := emitirReciboCargaDirectaPrueba(t, context.Background(), adaptador, reloj)
	valorRecibo, _ := recibo.RevelarParaEntregaOConsumo()
	solicitud := solicitudConsumoCargaDirectaPrueba(recibo)
	comprobante, err := adaptador.ConsumirReciboCargaDirecta(context.Background(), solicitud)
	if err != nil {
		t.Fatal(err)
	}
	confirmacion, err := ports.NuevaSolicitudConfirmarCargaDirecta(
		context.Background(), solicitud.Contexto, solicitud.SesionRef, comprobante, adaptador,
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitudSeudonimo, _ := ports.NuevaSolicitudSeudonimizarSujetoAlmacen(
		"dni-interno-supersecreto:12345678Z", "bolsas:carga:001",
	)

	var salida bytes.Buffer
	registrador := slog.New(slog.NewTextHandler(&salida, nil))
	registrador.Info("prueba", "configuracion", configuracion, "adaptador", adaptador, "recibo", recibo,
		"solicitud", solicitud, "comprobante", comprobante, "confirmacion", confirmacion,
		"seudonimizacion", solicitudSeudonimo)
	textos := []string{
		fmt.Sprintf("%v %#v %+v", configuracion, configuracion, configuracion),
		fmt.Sprintf("%v %#v %+v", adaptador, adaptador, adaptador),
		fmt.Sprintf("%v %#v %+v", recibo, recibo, recibo),
		fmt.Sprintf("%v %#v %+v", solicitud, solicitud, solicitud),
		fmt.Sprintf("%v %#v %+v", comprobante, comprobante, comprobante),
		fmt.Sprintf("%v %#v %+v", confirmacion, confirmacion, confirmacion),
		fmt.Sprintf("%v %#v %+v", solicitudSeudonimo, solicitudSeudonimo, solicitudSeudonimo),
		salida.String(),
	}
	jsonConfiguracion, err := json.Marshal(configuracion)
	if err != nil {
		t.Fatal(err)
	}
	jsonAdaptador, err := json.Marshal(adaptador)
	if err != nil {
		t.Fatal(err)
	}
	textos = append(textos, string(jsonConfiguracion), string(jsonAdaptador))
	if datos, err := json.Marshal(recibo); err == nil || len(datos) != 0 {
		t.Fatalf("el recibo admitio JSON")
	}
	if datos, err := json.Marshal(solicitud); err == nil || len(datos) != 0 {
		t.Fatalf("la solicitud de consumo admitio JSON")
	}
	if datos, err := json.Marshal(comprobante); err == nil || len(datos) != 0 {
		t.Fatalf("el comprobante admitio JSON")
	}
	if datos, err := json.Marshal(confirmacion); err == nil || len(datos) != 0 {
		t.Fatalf("la confirmacion admitio JSON")
	}
	if datos, err := json.Marshal(solicitudSeudonimo); err == nil || len(datos) != 0 {
		t.Fatalf("la solicitud de seudonimizacion admitio JSON")
	}
	secretos := []string{valorRecibo, solicitud.SesionRef, "dni-interno-supersecreto:12345678Z"}
	for _, clave := range []ConfiguracionClaveHMACCargaDirecta{
		configuracion.ClaveSeudonimizacion,
		configuracion.ClaveIndiceRecibo,
		configuracion.ClaveVinculoRecibo,
		configuracion.ClaveAtestacion,
	} {
		secretos = append(secretos, string(clave.Material))
	}
	for _, texto := range textos {
		for _, secreto := range secretos {
			if strings.Contains(texto, secreto) {
				t.Fatalf("secreto filtrado en representacion: %q", texto)
			}
		}
	}
}

func alterarPrimerCaracterCargaDirecta(valor string, primero, segundo byte) string {
	if valor == "" {
		return valor
	}
	reemplazo := primero
	if valor[0] == primero {
		reemplazo = segundo
	}
	return string(reemplazo) + valor[1:]
}

func TestContratoPersistenciaReciboCargaDirectaValidaSoloDatosCanonicos(t *testing.T) {
	instante := time.Date(2026, 7, 15, 8, 30, 0, 0, time.UTC)
	registro := ports.RegistroReciboCargaDirecta{
		IndiceHMAC:             "hmac-sha256:indice_v1:" + strings.Repeat("a", 64),
		GrupoHMAC:              "hmac-sha256:grupo_v1:" + strings.Repeat("c", 64),
		VinculoHMAC:            "hmac-sha256:vinculo_v1:" + strings.Repeat("b", 64),
		EvidenciaAltaRef:       "recibo-alta-v1:001",
		AutorizacionEmisionRef: "autorizacion:emision:001",
		ExpiraEn:               instante.Add(time.Minute),
	}
	if err := registro.Validar(); err != nil {
		t.Fatalf("registro valido rechazado: %v", err)
	}
	resultadoAlta := ports.ResultadoRegistroReciboCargaDirecta{
		IndiceHMAC: registro.IndiceHMAC, GrupoHMAC: registro.GrupoHMAC,
		AutorizacionEmisionRef: registro.AutorizacionEmisionRef, RegistradoEn: instante,
	}
	if err := resultadoAlta.ValidarContra(registro); err != nil {
		t.Fatalf("alta durable valida rechazada: %v", err)
	}
	resultadoAlta.RegistradoEn = registro.ExpiraEn
	if !errors.Is(resultadoAlta.ValidarContra(registro), ports.ErrReciboCargaDirectaNoValido) {
		t.Fatal("alta durable sin ventana aceptada")
	}
	resultadoAlta.RegistradoEn = instante
	registro.ExpiraEn = instante.Add(10*time.Minute + time.Nanosecond)
	if !errors.Is(resultadoAlta.ValidarContra(registro), ports.ErrReciboCargaDirectaNoValido) {
		t.Fatal("alta durable con vigencia superior a diez minutos aceptada")
	}
	registro.ExpiraEn = instante.Add(time.Minute)
	resultadoAlta.Predecesor = &ports.PredecesorReciboCargaDirecta{
		IndiceHMAC: "hmac-sha256:indice_v1:" + strings.Repeat("e", 64),
		GrupoHMAC:  registro.GrupoHMAC, AutorizacionEmisionRef: "autorizacion:emision:anterior",
		SustituidoEn: instante,
	}
	if err := resultadoAlta.ValidarContra(registro); err != nil {
		t.Fatalf("predecesor exacto rechazado: %v", err)
	}
	resultadoAlta.Predecesor.GrupoHMAC = "hmac-sha256:grupo_v1:" + strings.Repeat("f", 64)
	if !errors.Is(resultadoAlta.ValidarContra(registro), ports.ErrReciboCargaDirectaNoValido) {
		t.Fatal("predecesor de otro grupo aceptado")
	}
	orden := ports.OrdenConsumoReciboCargaDirecta{
		IndiceHMAC:               "hmac-sha256:indice_v1:" + strings.Repeat("a", 64),
		GrupoHMAC:                "hmac-sha256:grupo_v1:" + strings.Repeat("c", 64),
		VinculoHMAC:              "hmac-sha256:vinculo_v1:" + strings.Repeat("b", 64),
		EvidenciaConsumoRef:      "recibo-consumo-v1:001",
		IntencionConfirmacionRef: "confirmacion-intencion-v1:001",
		HuellaIntencionHMAC:      "hmac-sha256:atestacion_v1:" + strings.Repeat("d", 64),
		RegistradoEn:             instante,
		ValidaHasta:              instante.Add(time.Minute),
	}
	if err := orden.Validar(); err != nil {
		t.Fatalf("orden valida rechazada: %v", err)
	}
	resultado := ports.ResultadoConsumoReciboCargaDirecta{
		IndiceHMAC: orden.IndiceHMAC, GrupoHMAC: orden.GrupoHMAC, VinculoHMAC: orden.VinculoHMAC,
		EvidenciaConsumoRef:      orden.EvidenciaConsumoRef,
		IntencionConfirmacionRef: orden.IntencionConfirmacionRef,
		HuellaIntencionHMAC:      orden.HuellaIntencionHMAC,
		RegistradoEn:             instante, ConsumidoEn: instante.Add(time.Second), ExpiraEn: instante.Add(time.Minute),
	}
	if err := resultado.ValidarContra(orden); err != nil {
		t.Fatalf("resultado durable exacto rechazado: %v", err)
	}
	resultado.ExpiraEn = resultado.RegistradoEn.Add(10*time.Minute + time.Nanosecond)
	if !errors.Is(resultado.ValidarContra(orden), ports.ErrReciboCargaDirectaNoValido) {
		t.Fatal("resultado con ventana superior a diez minutos aceptado")
	}
	resultado.ExpiraEn = instante.Add(time.Minute)
	resultado.RegistradoEn = resultado.ConsumidoEn.Add(time.Nanosecond)
	if !errors.Is(resultado.ValidarContra(orden), ports.ErrReciboCargaDirectaNoValido) {
		t.Fatal("resultado consumido antes del alta durable aceptado")
	}
	resultado.RegistradoEn = instante
	resultado.GrupoHMAC = "hmac-sha256:grupo_v1:" + strings.Repeat("d", 64)
	if !errors.Is(resultado.ValidarContra(orden), ports.ErrReciboCargaDirectaNoValido) {
		t.Fatal("resultado durable de otro grupo aceptado")
	}
	orden.IndiceHMAC = "token-en-claro"
	if !errors.Is(orden.Validar(), ports.ErrReciboCargaDirectaNoValido) {
		t.Fatal("indice sin HMAC aceptado")
	}
}

var _ ports.RepositorioRecibosCargaDirecta = (*repositorioRecibosCargaDirectaPrueba)(nil)
var _ ports.Reloj = (*relojCargaDirectaPrueba)(nil)
