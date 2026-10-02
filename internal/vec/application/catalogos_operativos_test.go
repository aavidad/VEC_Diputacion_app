package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/memory"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// Este doble prueba la coordinación de la aplicación. No acredita SQL,
// transacciones, ACL ni recuperación tras reinicio de una base real.
type repositorioCatalogosOperativosPrueba struct {
	mu              sync.Mutex
	store           *memory.Store
	cabeza          ports.CabezaCatalogoOperativo
	operaciones     map[string]ports.ResultadoOperacionCatalogoOperativo
	decisiones      []string
	confirmaciones  int
	lecturasRecibo  int
	corromper       func(*ports.ResultadoOperacionCatalogoOperativo)
	perderRespuesta bool
	trasPersistir   func()
}

func (r *repositorioCatalogosOperativosPrueba) ConfirmarAltaBorradorCatalogoOperativo(ctx context.Context, orden ports.ConfirmacionCatalogoOperativo) (ports.ResultadoOperacionCatalogoOperativo, error) {
	return r.confirmar(ctx, orden, ports.AccionCrearCatalogoConfigurable)
}

func (r *repositorioCatalogosOperativosPrueba) ConfirmarPublicacionCatalogoOperativo(ctx context.Context, orden ports.ConfirmacionCatalogoOperativo) (ports.ResultadoOperacionCatalogoOperativo, error) {
	return r.confirmar(ctx, orden, ports.AccionPublicarCatalogoConfigurable)
}

func (r *repositorioCatalogosOperativosPrueba) confirmar(ctx context.Context, orden ports.ConfirmacionCatalogoOperativo, accion string) (ports.ResultadoOperacionCatalogoOperativo, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.confirmaciones++
	datos, err := orden.Autorizacion.Datos()
	if err != nil || datos.Decision.Accion != accion || datos.Decision.RecursoRef != orden.Catalogo.Referencia() {
		return ports.ResultadoOperacionCatalogoOperativo{}, domain.ErrAutorizacionDenegada
	}
	r.decisiones = append(r.decisiones, datos.Decision.DecisionRef)
	suma := sha256.Sum256(orden.MaterialCanonico)
	if hex.EncodeToString(suma[:]) != orden.HuellaMaterialSHA256 {
		return ports.ResultadoOperacionCatalogoOperativo{}, ports.ErrOperacionCatalogoOperativoEnConflicto
	}
	var material materialCatalogoOperativo
	if json.Unmarshal(orden.MaterialCanonico, &material) != nil || material.ActorRef != datos.Decision.PrincipalID || material.Accion != accion {
		return ports.ResultadoOperacionCatalogoOperativo{}, domain.ErrAutorizacionDenegada
	}
	if original, ok := r.operaciones[orden.ClaveIdempotencia]; ok {
		return r.recuperado(original, orden.HuellaMaterialSHA256, accion, datos.Decision.PrincipalID)
	}
	if orden.CabezaEsperada != r.cabeza {
		return ports.ResultadoOperacionCatalogoOperativo{}, ports.ErrRevisionCatalogoEnConflicto
	}
	fecha, actor := orden.Catalogo.CreadoEn, orden.Catalogo.CreadoPor
	switch accion {
	case ports.AccionCrearCatalogoConfigurable:
		err = r.store.ConfirmarAltaBorradorCatalogo(ctx, orden.Catalogo, orden.Auditoria, orden.Evento, orden.Autorizacion)
	case ports.AccionPublicarCatalogoConfigurable:
		err = r.store.ConfirmarPublicacionCatalogo(ctx, orden.HuellaAnteriorSHA256, orden.Catalogo, orden.Auditoria, orden.Evento, orden.Autorizacion)
		fecha, actor = orden.Catalogo.PublicadoEn, orden.Catalogo.PublicadoPor
	}
	if err != nil {
		return ports.ResultadoOperacionCatalogoOperativo{}, err
	}
	huella, err := orden.Catalogo.HuellaSHA256()
	if err != nil {
		return ports.ResultadoOperacionCatalogoOperativo{}, err
	}
	resultado := ports.ResultadoOperacionCatalogoOperativo{
		Catalogo: orden.Catalogo,
		Recibo: ports.ReciboCatalogoOperativo{
			Referencia:           "recibo:" + orden.ClaveIdempotencia,
			ClaveIdempotencia:    orden.ClaveIdempotencia,
			HuellaMaterialSHA256: orden.HuellaMaterialSHA256,
			Accion:               accion,
			CatalogoID:           orden.Catalogo.ID,
			Version:              orden.Catalogo.Version,
			HuellaSHA256:         huella,
			Estado:               orden.Catalogo.Estado,
			ActorRef:             actor,
			AuditoriaRef:         "auditoria:" + orden.ClaveIdempotencia,
			OutboxRef:            "outbox:" + orden.ClaveIdempotencia,
			ConfirmadoEn:         fecha,
		},
	}
	r.operaciones[orden.ClaveIdempotencia] = resultado
	if accion == ports.AccionPublicarCatalogoConfigurable {
		r.cabeza = cabezaCatalogoOperativoPrueba(resultado.Catalogo)
	}
	if r.trasPersistir != nil {
		r.trasPersistir()
	}
	if r.perderRespuesta {
		r.perderRespuesta = false
		return ports.ResultadoOperacionCatalogoOperativo{}, errors.New("respuesta perdida tras confirmar")
	}
	if r.corromper != nil {
		r.corromper(&resultado)
	}
	return resultado, nil
}

func (r *repositorioCatalogosOperativosPrueba) RecuperarOperacionCatalogoOperativo(_ context.Context, orden ports.RecuperacionCatalogoOperativo) (ports.ResultadoOperacionCatalogoOperativo, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	datos, err := orden.Autorizacion.Datos()
	if err != nil || datos.Decision.Accion != orden.Accion || datos.Decision.RecursoRef != fmt.Sprintf("%s:%d", orden.CatalogoID, orden.Version) {
		return ports.ResultadoOperacionCatalogoOperativo{}, domain.ErrAutorizacionDenegada
	}
	r.decisiones = append(r.decisiones, datos.Decision.DecisionRef)
	r.lecturasRecibo++
	suma := sha256.Sum256(orden.MaterialCanonico)
	if hex.EncodeToString(suma[:]) != orden.HuellaMaterialSHA256 {
		return ports.ResultadoOperacionCatalogoOperativo{}, ports.ErrOperacionCatalogoOperativoEnConflicto
	}
	original, ok := r.operaciones[orden.ClaveIdempotencia]
	if !ok {
		return ports.ResultadoOperacionCatalogoOperativo{}, ports.ErrOperacionCatalogoOperativoNoEncontrada
	}
	return r.recuperado(original, orden.HuellaMaterialSHA256, orden.Accion, datos.Decision.PrincipalID)
}

func (r *repositorioCatalogosOperativosPrueba) recuperado(original ports.ResultadoOperacionCatalogoOperativo, material, accion, actor string) (ports.ResultadoOperacionCatalogoOperativo, error) {
	if original.Recibo.ActorRef != actor {
		return ports.ResultadoOperacionCatalogoOperativo{}, domain.ErrAutorizacionDenegada
	}
	if original.Recibo.HuellaMaterialSHA256 != material || original.Recibo.Accion != accion {
		return ports.ResultadoOperacionCatalogoOperativo{}, ports.ErrOperacionCatalogoOperativoEnConflicto
	}
	original.Recuperada = true
	if r.corromper != nil {
		r.corromper(&original)
	}
	return original, nil
}

type gobiernoCatalogoHistoricoCerradoPrueba struct{}

func (gobiernoCatalogoHistoricoCerradoPrueba) ConfirmarAltaBorradorCatalogo(context.Context, domain.CatalogoConfigurable, domain.AuditEntry, domain.Event, ports.EvidenciaUsoDecisionAutorizacion) error {
	return ErrDependenciaCatalogosRequerida
}
func (gobiernoCatalogoHistoricoCerradoPrueba) ConfirmarActualizacionBorradorCatalogo(context.Context, string, domain.CatalogoConfigurable, domain.AuditEntry, domain.Event, ports.EvidenciaUsoDecisionAutorizacion) error {
	return ErrDependenciaCatalogosRequerida
}
func (gobiernoCatalogoHistoricoCerradoPrueba) ConfirmarPublicacionCatalogo(context.Context, string, domain.CatalogoConfigurable, domain.AuditEntry, domain.Event, ports.EvidenciaUsoDecisionAutorizacion) error {
	return ErrDependenciaCatalogosRequerida
}
func (gobiernoCatalogoHistoricoCerradoPrueba) ConfirmarRetiradaCatalogo(context.Context, string, domain.CatalogoConfigurable, domain.AuditEntry, domain.Event, ports.EvidenciaUsoDecisionAutorizacion) error {
	return ErrDependenciaCatalogosRequerida
}

func cabezaCatalogoOperativoPrueba(catalogo domain.CatalogoConfigurable) ports.CabezaCatalogoOperativo {
	huella, _ := catalogo.HuellaSHA256()
	return ports.CabezaCatalogoOperativo{CatalogoID: catalogo.ID, Version: catalogo.Version, HuellaSHA256: huella, Estado: catalogo.Estado}
}

func nuevoCatalogoOperativoPrueba(t *testing.T) (*ServicioCatalogos, *repositorioCatalogosOperativosPrueba, OrdenCrearBorradorCatalogoOperativo) {
	t.Helper()
	s, store := nuevoServicioCatalogosPrueba(t)
	orden := ordenCrearCatalogoPrueba(t)
	orden.ID, orden.ModuloID = "administracion.modulos", "vec.module.administracion"
	orden.Entradas = []domain.EntradaCatalogoConfigurable{{Clave: "vec.module.bolsa", Etiqueta: "Bolsa", VigenteDesde: instanteCatalogosPrueba.Add(-time.Hour), Atributos: map[string]string{"activado": "si"}}}
	borrador, err := s.CrearBorrador(context.Background(), orden)
	if err != nil {
		t.Fatal(err)
	}
	publicado, err := s.Publicar(context.Background(), OrdenPublicarCatalogo{
		Credenciales: credencialesCatalogosPrueba(t, "publicador-inicial"),
		Finalidad:    orden.Finalidad, ID: borrador.ID, Version: 1,
		AprobacionRef: "aprobacion:inicial", Motivo: "Publicación inicial", CorrelacionRef: "corr:inicial",
	})
	if err != nil {
		t.Fatal(err)
	}
	s.gobierno = gobiernoCatalogoHistoricoCerradoPrueba{}
	repo := &repositorioCatalogosOperativosPrueba{store: store, cabeza: cabezaCatalogoOperativoPrueba(publicado), operaciones: map[string]ports.ResultadoOperacionCatalogoOperativo{}}
	orden.Version = 2
	orden.CorrelacionRef = "corr:preparar-v2"
	orden.Entradas[0].Atributos["activado"] = "no"
	return s, repo, OrdenCrearBorradorCatalogoOperativo{Orden: orden, ClaveIdempotencia: "operacion:preparar-v2", CabezaEsperada: repo.cabeza}
}

func publicacionCatalogoOperativoPrueba(t *testing.T, alta OrdenCrearBorradorCatalogoOperativo, resultado ports.ResultadoOperacionCatalogoOperativo) OrdenPublicarCatalogoOperativo {
	t.Helper()
	return OrdenPublicarCatalogoOperativo{
		Orden: OrdenPublicarCatalogo{Credenciales: credencialesCatalogosPrueba(t, "publicador-operativo"), Finalidad: alta.Orden.Finalidad,
			ID: alta.Orden.ID, Version: alta.Orden.Version, AprobacionRef: "aprobacion:operativa", Motivo: "Cambio aprobado", CorrelacionRef: "corr:publicar"},
		ClaveIdempotencia: fmt.Sprintf("operacion:publicar-v%d", alta.Orden.Version), CabezaEsperada: alta.CabezaEsperada,
		HuellaBorradorEsperadaSHA256: resultado.Recibo.HuellaSHA256,
	}
}

func TestCatalogosOperativosVersionesYReplayHistorico(t *testing.T) {
	s, repo, alta := nuevoCatalogoOperativoPrueba(t)
	gobiernoOriginal := s.gobierno
	cabezaOriginal := repo.cabeza
	borrador, err := s.CrearBorradorOperativo(context.Background(), repo, alta)
	if err != nil {
		t.Fatal(err)
	}
	if repo.cabeza != cabezaOriginal || borrador.Catalogo.Estado != domain.EstadoCatalogoBorrador {
		t.Fatal("preparar alteró la cabeza publicada")
	}
	publicacion := publicacionCatalogoOperativoPrueba(t, alta, borrador)
	publicado, err := s.PublicarOperativo(context.Background(), repo, publicacion)
	if err != nil {
		t.Fatal(err)
	}
	if repo.cabeza.Version != 2 || publicado.Catalogo.Entradas[0].Atributos["activado"] != "no" {
		t.Fatal("publicar no actualizó la cabeza")
	}
	altaV3 := alta
	altaV3.Orden.Version = 3
	altaV3.Orden.CorrelacionRef = "corr:v3"
	altaV3.Orden.Entradas = []domain.EntradaCatalogoConfigurable{{Clave: "vec.module.bolsa", Etiqueta: "Bolsa", VigenteDesde: instanteCatalogosPrueba.Add(-time.Hour), Atributos: map[string]string{"activado": "si"}}}
	altaV3.ClaveIdempotencia = "operacion:preparar-v3"
	altaV3.CabezaEsperada = repo.cabeza
	borradorV3, err := s.CrearBorradorOperativo(context.Background(), repo, altaV3)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.PublicarOperativo(context.Background(), repo, publicacionCatalogoOperativoPrueba(t, altaV3, borradorV3)); err != nil {
		t.Fatal(err)
	}
	ahora := instanteCatalogosPrueba.Add(time.Minute)
	s.reloj = relojCatalogosFijo{ahora: ahora}
	s.autorizador.(*autorizadorCatalogosPrueba).ahora = ahora
	alta.Orden.Credenciales = credencialesCatalogosPruebaEn(t, "tecnico-configuracion-1", ahora)
	alta.Orden.CorrelacionRef = "corr:reintento-alta"
	publicacion.Orden.Credenciales = credencialesCatalogosPruebaEn(t, "publicador-operativo", ahora)
	publicacion.Orden.CorrelacionRef = "corr:reintento-publicacion"
	altaRecuperada, err := s.CrearBorradorOperativo(context.Background(), repo, alta)
	if err != nil {
		t.Fatal(err)
	}
	publicacionRecuperada, err := s.PublicarOperativo(context.Background(), repo, publicacion)
	if err != nil {
		t.Fatal(err)
	}
	for _, par := range [][2]ports.ResultadoOperacionCatalogoOperativo{{borrador, altaRecuperada}, {publicado, publicacionRecuperada}} {
		if !par[1].Recuperada || par[0].Recibo != par[1].Recibo || !reflect.DeepEqual(par[0].Catalogo, par[1].Catalogo) {
			t.Fatalf("replay cambió evidencia original: %#v", par)
		}
	}
	if len(repo.operaciones) != 4 || repo.cabeza.Version != 3 || s.gobierno != gobiernoOriginal {
		t.Fatal("replay duplicó efectos o cambió servicio compartido")
	}
	for i, decision := range repo.decisiones {
		for j := 0; j < i; j++ {
			if decision == repo.decisiones[j] {
				t.Fatal("replay reutilizó decisión histórica")
			}
		}
	}
}

func TestCatalogosOperativosDenieganReplaySinPermisoActual(t *testing.T) {
	s, repo, alta := nuevoCatalogoOperativoPrueba(t)
	borrador, err := s.CrearBorradorOperativo(context.Background(), repo, alta)
	if err != nil {
		t.Fatal(err)
	}
	publicacion := publicacionCatalogoOperativoPrueba(t, alta, borrador)
	if _, err := s.PublicarOperativo(context.Background(), repo, publicacion); err != nil {
		t.Fatal(err)
	}
	confirmaciones, lecturas := repo.confirmaciones, repo.lecturasRecibo
	alta.Orden.Credenciales = credencialesCatalogosPrueba(t, "sin-autorizacion")
	publicacion.Orden.Credenciales = alta.Orden.Credenciales
	if _, err := s.CrearBorradorOperativo(context.Background(), repo, alta); !errors.Is(err, domain.ErrAutorizacionDenegada) {
		t.Fatalf("alta = %v", err)
	}
	if _, err := s.PublicarOperativo(context.Background(), repo, publicacion); !errors.Is(err, domain.ErrAutorizacionDenegada) {
		t.Fatalf("publicación = %v", err)
	}
	if repo.confirmaciones != confirmaciones || repo.lecturasRecibo != lecturas {
		t.Fatal("consultó recibo sin permiso vigente")
	}
}

func TestCatalogosOperativosConflictoMaterialYActorOriginal(t *testing.T) {
	s, repo, alta := nuevoCatalogoOperativoPrueba(t)
	borrador, err := s.CrearBorradorOperativo(context.Background(), repo, alta)
	if err != nil {
		t.Fatal(err)
	}
	alterada := alta
	alterada.Orden.Nombre += " cambiado"
	if _, err := s.CrearBorradorOperativo(context.Background(), repo, alterada); !errors.Is(err, ports.ErrOperacionCatalogoOperativoEnConflicto) {
		t.Fatalf("material distinto = %v", err)
	}
	ajena := alta
	ajena.Orden.Credenciales = credencialesCatalogosPrueba(t, "tercero-autorizado")
	if _, err := s.CrearBorradorOperativo(context.Background(), repo, ajena); !errors.Is(err, domain.ErrAutorizacionDenegada) {
		t.Fatalf("actor ajeno = %v", err)
	}
	publicacion := publicacionCatalogoOperativoPrueba(t, alta, borrador)
	if _, err := s.PublicarOperativo(context.Background(), repo, publicacion); err != nil {
		t.Fatal(err)
	}
	publicacion.Orden.Motivo += " cambiado"
	if _, err := s.PublicarOperativo(context.Background(), repo, publicacion); !errors.Is(err, ports.ErrOperacionCatalogoOperativoEnConflicto) {
		t.Fatalf("material publicación = %v", err)
	}
	publicacion.Orden.Credenciales = ajena.Orden.Credenciales
	if _, err := s.PublicarOperativo(context.Background(), repo, publicacion); !errors.Is(err, domain.ErrAutorizacionDenegada) {
		t.Fatalf("publicador ajeno = %v", err)
	}
}

func TestCatalogosOperativosCASExactoYSeparacion(t *testing.T) {
	s, repo, alta := nuevoCatalogoOperativoPrueba(t)
	obsoleta := alta
	obsoleta.CabezaEsperada.HuellaSHA256 = fmt.Sprintf("%064d", 0)
	if _, err := s.CrearBorradorOperativo(context.Background(), repo, obsoleta); !errors.Is(err, ports.ErrRevisionCatalogoEnConflicto) {
		t.Fatalf("CAS alta = %v", err)
	}
	borrador, err := s.CrearBorradorOperativo(context.Background(), repo, alta)
	if err != nil {
		t.Fatal(err)
	}
	publicacion := publicacionCatalogoOperativoPrueba(t, alta, borrador)
	publicacion.HuellaBorradorEsperadaSHA256 = fmt.Sprintf("%064d", 0)
	if _, err := s.PublicarOperativo(context.Background(), repo, publicacion); !errors.Is(err, ports.ErrRevisionCatalogoEnConflicto) {
		t.Fatalf("borrador exacto = %v", err)
	}
	publicacion.HuellaBorradorEsperadaSHA256 = borrador.Recibo.HuellaSHA256
	publicacion.CabezaEsperada = obsoleta.CabezaEsperada
	if _, err := s.PublicarOperativo(context.Background(), repo, publicacion); !errors.Is(err, ports.ErrRevisionCatalogoEnConflicto) {
		t.Fatalf("CAS publicación = %v", err)
	}
	publicacion.CabezaEsperada = alta.CabezaEsperada
	publicacion.Orden.Credenciales = alta.Orden.Credenciales
	if _, err := s.PublicarOperativo(context.Background(), repo, publicacion); !errors.Is(err, domain.ErrTransicionCatalogoInvalida) {
		t.Fatalf("mismo actor = %v", err)
	}
	if len(repo.operaciones) != 1 || repo.cabeza.Version != 1 {
		t.Fatal("fallo dejó efecto parcial")
	}
}

type consultaCatalogoOperativoCarrera struct {
	ports.ConsultaCatalogosConfigurables
	primera *domain.CatalogoConfigurable
}

func (c *consultaCatalogoOperativoCarrera) ObtenerCatalogo(ctx context.Context, id string, version int) (domain.CatalogoConfigurable, error) {
	if c.primera != nil {
		actual := *c.primera
		c.primera = nil
		return actual, nil
	}
	return c.ConsultaCatalogosConfigurables.ObtenerCatalogo(ctx, id, version)
}

func TestCatalogosOperativosRecuperanPublicacionConcurrente(t *testing.T) {
	s, repo, alta := nuevoCatalogoOperativoPrueba(t)
	borrador, err := s.CrearBorradorOperativo(context.Background(), repo, alta)
	if err != nil {
		t.Fatal(err)
	}
	orden := publicacionCatalogoOperativoPrueba(t, alta, borrador)
	original, err := s.PublicarOperativo(context.Background(), repo, orden)
	if err != nil {
		t.Fatal(err)
	}
	s.consulta = &consultaCatalogoOperativoCarrera{ConsultaCatalogosConfigurables: s.consulta, primera: &borrador.Catalogo}
	recuperada, err := s.PublicarOperativo(context.Background(), repo, orden)
	if err != nil {
		t.Fatal(err)
	}
	if !recuperada.Recuperada || recuperada.Recibo != original.Recibo || len(repo.operaciones) != 2 {
		t.Fatal("carrera no recuperó original")
	}
}

func TestCatalogosOperativosRechazanReciboSustituido(t *testing.T) {
	s, repo, alta := nuevoCatalogoOperativoPrueba(t)
	repo.corromper = func(resultado *ports.ResultadoOperacionCatalogoOperativo) {
		resultado.Recibo.ActorRef = "actor:sustituido"
	}
	if _, err := s.CrearBorradorOperativo(context.Background(), repo, alta); !errors.Is(err, ports.ErrReciboCatalogoOperativoInvalido) {
		t.Fatalf("recibo = %v", err)
	}
}

func TestOrdenCatalogoOperativoBloqueaCredencialesEnJSON(t *testing.T) {
	_, _, alta := nuevoCatalogoOperativoPrueba(t)
	for _, orden := range []any{alta, OrdenPublicarCatalogoOperativo{}} {
		if _, err := json.Marshal(orden); !errors.Is(err, ErrSerializacionOrdenCatalogo) {
			t.Fatalf("serialización = %v", err)
		}
		if fmt.Sprint(orden) != "[ORDEN-CATALOGO-INTERNA]" {
			t.Fatal("formateo expuso orden interna")
		}
	}
}

func TestCatalogosOperativosRecuperanRespuestaPerdidaDeBorrador(t *testing.T) {
	s, repo, alta := nuevoCatalogoOperativoPrueba(t)
	repo.perderRespuesta = true
	resultado, err := s.CrearBorradorOperativo(context.Background(), repo, alta)
	if err != nil {
		t.Fatal(err)
	}
	if !resultado.Recuperada || len(repo.operaciones) != 1 || repo.lecturasRecibo != 1 || len(repo.decisiones) != 2 ||
		repo.decisiones[0] == repo.decisiones[1] || repo.cabeza.Version != 1 {
		t.Fatal("no recuperó el único efecto con una decisión nueva")
	}
}

type autorizadorCatalogoOperativoDenegado struct{}

func (autorizadorCatalogoOperativoDenegado) Exigir(context.Context, domain.SolicitudAutorizacion) (domain.DecisionAutorizacion, error) {
	return domain.DecisionAutorizacion{}, domain.ErrAutorizacionDenegada
}

func TestCatalogosOperativosRevocacionDuranteRecuperacion(t *testing.T) {
	s, repo, alta := nuevoCatalogoOperativoPrueba(t)
	repo.perderRespuesta = true
	repo.trasPersistir = func() { s.autorizador = autorizadorCatalogoOperativoDenegado{} }
	resultado, err := s.CrearBorradorOperativo(context.Background(), repo, alta)
	if !errors.Is(err, domain.ErrAutorizacionDenegada) || resultado.Recibo.Referencia != "" || repo.lecturasRecibo != 0 {
		t.Fatalf("recuperación sin autorización actual: resultado=%#v error=%v", resultado, err)
	}
}

type autorizadorCatalogoOperativoConcurrente struct {
	mu   sync.Mutex
	base ports.Autorizador
}

func (a *autorizadorCatalogoOperativoConcurrente) Exigir(ctx context.Context, solicitud domain.SolicitudAutorizacion) (domain.DecisionAutorizacion, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.base.Exigir(ctx, solicitud)
}

func TestCatalogosOperativosInvocacionesConcurrentesNoMutanServicio(t *testing.T) {
	s, repo, alta := nuevoCatalogoOperativoPrueba(t)
	s.autorizador = &autorizadorCatalogoOperativoConcurrente{base: s.autorizador}
	gobierno := s.gobierno
	var wg sync.WaitGroup
	errores := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.CrearBorradorOperativo(context.Background(), repo, alta)
			errores <- err
		}()
	}
	wg.Wait()
	close(errores)
	for err := range errores {
		if err != nil {
			t.Fatal(err)
		}
	}
	if s.gobierno != gobierno || len(repo.operaciones) != 1 || repo.cabeza.Version != 1 {
		t.Fatal("invocaciones compartieron estado mutable o duplicaron efecto")
	}
}

func TestCatalogosOperativosLimiteClaveSemanticaCAT6(t *testing.T) {
	for _, longitud := range []int{1, 2, 3, 160, 161} {
		t.Run(fmt.Sprint(longitud), func(t *testing.T) {
			s, repo, alta := nuevoCatalogoOperativoPrueba(t)
			alta.ClaveIdempotencia = strings.Repeat("k", longitud)
			_, err := s.CrearBorradorOperativo(context.Background(), repo, alta)
			if longitud >= 3 && longitud <= 160 {
				if err != nil {
					t.Fatal(err)
				}
			} else if !errors.Is(err, ErrOrdenCatalogoInvalida) || repo.confirmaciones != 0 {
				t.Fatalf("clave fuera del contrato CAT6 llegó al repositorio: %v", err)
			}
		})
	}
}
