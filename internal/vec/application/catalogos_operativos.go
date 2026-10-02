package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type OrdenCrearBorradorCatalogoOperativo struct {
	Orden             OrdenCrearBorradorCatalogo
	ClaveIdempotencia string
	CabezaEsperada    ports.CabezaCatalogoOperativo
}

func (OrdenCrearBorradorCatalogoOperativo) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionOrdenCatalogo
}
func (*OrdenCrearBorradorCatalogoOperativo) UnmarshalJSON([]byte) error {
	return ErrSerializacionOrdenCatalogo
}
func (OrdenCrearBorradorCatalogoOperativo) String() string { return "[ORDEN-CATALOGO-INTERNA]" }

type OrdenPublicarCatalogoOperativo struct {
	Orden                        OrdenPublicarCatalogo
	ClaveIdempotencia            string
	CabezaEsperada               ports.CabezaCatalogoOperativo
	HuellaBorradorEsperadaSHA256 string
}

func (OrdenPublicarCatalogoOperativo) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionOrdenCatalogo
}
func (*OrdenPublicarCatalogoOperativo) UnmarshalJSON([]byte) error {
	return ErrSerializacionOrdenCatalogo
}
func (OrdenPublicarCatalogoOperativo) String() string { return "[ORDEN-CATALOGO-INTERNA]" }

// CrearBorradorOperativo conserva la transición y la autorización de
// CrearBorrador. El decorador sólo añade CAS y captura el resultado durable;
// ni el servicio compartido ni su repositorio histórico se modifican.
func (s *ServicioCatalogos) CrearBorradorOperativo(
	ctx context.Context,
	repositorio ports.RepositorioCatalogosOperativos,
	orden OrdenCrearBorradorCatalogoOperativo,
) (ports.ResultadoOperacionCatalogoOperativo, error) {
	if dependenciaCatalogoNula(repositorio) || s == nil {
		return ports.ResultadoOperacionCatalogoOperativo{}, ErrDependenciaCatalogosRequerida
	}
	if err := validarOrdenCatalogoOperativo(orden.ClaveIdempotencia, orden.CabezaEsperada, orden.Orden.ID, orden.Orden.Version); err != nil {
		return ports.ResultadoOperacionCatalogoOperativo{}, err
	}
	decorador := &gobiernoCatalogoOperativoInvocacion{
		repositorio: repositorio,
		clave:       orden.ClaveIdempotencia,
		cabeza:      orden.CabezaEsperada,
		finalidad:   orden.Orden.Finalidad,
		motivo:      orden.Orden.Motivo,
		accion:      ports.AccionCrearCatalogoConfigurable,
	}
	local := *s
	local.gobierno = decorador
	if _, err := local.CrearBorrador(ctx, orden.Orden); err != nil {
		if len(decorador.material.bytes) == 0 {
			return ports.ResultadoOperacionCatalogoOperativo{}, err
		}
		resultado, recuperacionErr := s.recuperarBorradorOperativo(ctx, repositorio, orden, decorador)
		if recuperacionErr != nil {
			return ports.ResultadoOperacionCatalogoOperativo{}, errors.Join(err, recuperacionErr)
		}
		return resultado, nil
	}
	return decorador.resultado, nil
}

func (s *ServicioCatalogos) recuperarBorradorOperativo(ctx context.Context, repositorio ports.RepositorioCatalogosOperativos,
	orden OrdenCrearBorradorCatalogoOperativo, decorador *gobiernoCatalogoOperativoInvocacion) (ports.ResultadoOperacionCatalogoOperativo, error) {
	// Leer la versión exacta demuestra que existe tras una respuesta perdida.
	// La autorización sigue refiriéndose al borrador de la creación original,
	// aunque otra etapa haya publicado después esa misma versión.
	actual, err := s.consulta.ObtenerCatalogo(ctx, orden.Orden.ID, orden.Orden.Version)
	if err != nil {
		return ports.ResultadoOperacionCatalogoOperativo{}, err
	}
	actor, err := s.validarContextoGobiernoCatalogo(ctx, orden.Orden.Credenciales, orden.Orden.Finalidad, orden.Orden.Motivo, orden.Orden.CorrelacionRef)
	if err != nil {
		return ports.ResultadoOperacionCatalogoOperativo{}, err
	}
	if actual.Validar() != nil || actual.ID != orden.Orden.ID || actual.Version != orden.Orden.Version ||
		actual.CreadoPor != actor.Principal.ID {
		return ports.ResultadoOperacionCatalogoOperativo{}, ports.ErrReciboCatalogoOperativoInvalido
	}
	decision, err := s.autorizarCatalogo(ctx, actor, orden.Orden.Credenciales.VinculoAutenticacionActor,
		ports.AccionCrearCatalogoConfigurable, decorador.preparado, orden.Orden.Finalidad, orden.Orden.CorrelacionRef, orden.Orden.Motivo)
	if err != nil {
		return ports.ResultadoOperacionCatalogoOperativo{}, err
	}
	evidencia, err := s.evidenciaUsoDecisionCatalogo(ctx, decision)
	if err != nil {
		return ports.ResultadoOperacionCatalogoOperativo{}, err
	}
	resultado, err := repositorio.RecuperarOperacionCatalogoOperativo(ctx, ports.RecuperacionCatalogoOperativo{
		ClaveIdempotencia:    orden.ClaveIdempotencia,
		HuellaMaterialSHA256: decorador.material.huella,
		MaterialCanonico:     append([]byte(nil), decorador.material.bytes...),
		CatalogoID:           orden.Orden.ID,
		Version:              orden.Orden.Version,
		Accion:               ports.AccionCrearCatalogoConfigurable,
		Autorizacion:         evidencia,
	})
	if err != nil {
		return ports.ResultadoOperacionCatalogoOperativo{}, err
	}
	if err := validarResultadoCatalogoOperativo(resultado, orden.ClaveIdempotencia, decorador.material.huella,
		ports.AccionCrearCatalogoConfigurable, orden.Orden.ID, orden.Orden.Version, actor.Principal.ID); err != nil {
		return ports.ResultadoOperacionCatalogoOperativo{}, err
	}
	resultado.Recuperada = true
	return resultado, nil
}

// PublicarOperativo exige el borrador exacto y otra identidad mediante la
// transición Publicar existente. Antes de repetir una publicación ya hecha,
// lee su versión y solicita una autorización actual para el efecto original.
func (s *ServicioCatalogos) PublicarOperativo(
	ctx context.Context,
	repositorio ports.RepositorioCatalogosOperativos,
	orden OrdenPublicarCatalogoOperativo,
) (ports.ResultadoOperacionCatalogoOperativo, error) {
	if dependenciaCatalogoNula(repositorio) || s == nil {
		return ports.ResultadoOperacionCatalogoOperativo{}, ErrDependenciaCatalogosRequerida
	}
	actor, err := s.validarContextoGobiernoCatalogo(ctx, orden.Orden.Credenciales, orden.Orden.Finalidad, orden.Orden.Motivo, orden.Orden.CorrelacionRef)
	if err != nil {
		return ports.ResultadoOperacionCatalogoOperativo{}, err
	}
	if !actor.Principal.AuthAssurance.Cumple(domain.AuthAssuranceHigh) {
		return ports.ResultadoOperacionCatalogoOperativo{}, domain.ErrGarantiaInsuficiente
	}
	if err := validarOrdenCatalogoOperativo(orden.ClaveIdempotencia, orden.CabezaEsperada, orden.Orden.ID, orden.Orden.Version); err != nil {
		return ports.ResultadoOperacionCatalogoOperativo{}, err
	}
	if !huellaCatalogoOperativoValida(orden.HuellaBorradorEsperadaSHA256) ||
		orden.Orden.AprobacionRef == "" || orden.Orden.AprobacionRef != strings.TrimSpace(orden.Orden.AprobacionRef) {
		return ports.ResultadoOperacionCatalogoOperativo{}, ErrOrdenCatalogoInvalida
	}
	material, err := huellaMaterialCatalogoOperativo(materialCatalogoOperativo{
		Esquema:              esquemaMaterialCatalogoOperativo,
		Accion:               ports.AccionPublicarCatalogoConfigurable,
		CatalogoID:           orden.Orden.ID,
		Version:              orden.Orden.Version,
		ActorRef:             actor.Principal.ID,
		Cabeza:               orden.CabezaEsperada,
		HuellaBorradorSHA256: orden.HuellaBorradorEsperadaSHA256,
		Finalidad:            orden.Orden.Finalidad,
		Motivo:               orden.Orden.Motivo,
		AprobacionRef:        orden.Orden.AprobacionRef,
	})
	if err != nil {
		return ports.ResultadoOperacionCatalogoOperativo{}, err
	}
	actual, err := s.consulta.ObtenerCatalogo(ctx, orden.Orden.ID, orden.Orden.Version)
	if err != nil {
		return ports.ResultadoOperacionCatalogoOperativo{}, err
	}
	if actual.Estado == domain.EstadoCatalogoPublicado {
		return s.recuperarPublicacionOperativa(ctx, repositorio, orden, actor, actual, material)
	}
	decorador := &gobiernoCatalogoOperativoInvocacion{
		repositorio:    repositorio,
		clave:          orden.ClaveIdempotencia,
		cabeza:         orden.CabezaEsperada,
		huellaBorrador: orden.HuellaBorradorEsperadaSHA256,
		material:       material,
		accion:         ports.AccionPublicarCatalogoConfigurable,
	}
	local := *s
	local.gobierno = decorador
	if _, err := local.Publicar(ctx, orden.Orden); err != nil {
		// Una publicación concurrente puede llegar entre las dos lecturas, o
		// confirmarse aunque el transporte pierda la respuesta. La recuperación
		// no genera otra publicación ni devuelve un recibo sin nueva decisión.
		publicado, lecturaErr := s.consulta.ObtenerCatalogo(ctx, orden.Orden.ID, orden.Orden.Version)
		if lecturaErr != nil || publicado.Estado != domain.EstadoCatalogoPublicado {
			return ports.ResultadoOperacionCatalogoOperativo{}, err
		}
		resultado, recuperacionErr := s.recuperarPublicacionOperativa(ctx, repositorio, orden, actor, publicado, material)
		if recuperacionErr != nil {
			return ports.ResultadoOperacionCatalogoOperativo{}, errors.Join(err, recuperacionErr)
		}
		return resultado, nil
	}
	return decorador.resultado, nil
}

func (s *ServicioCatalogos) recuperarPublicacionOperativa(
	ctx context.Context,
	repositorio ports.RepositorioCatalogosOperativos,
	orden OrdenPublicarCatalogoOperativo,
	actor domain.ContextoActor,
	publicado domain.CatalogoConfigurable,
	material canonicoMaterialCatalogoOperativo,
) (ports.ResultadoOperacionCatalogoOperativo, error) {
	if publicado.ID != orden.Orden.ID || publicado.Version != orden.Orden.Version ||
		publicado.Estado != domain.EstadoCatalogoPublicado || publicado.Validar() != nil {
		return ports.ResultadoOperacionCatalogoOperativo{}, ports.ErrReciboCatalogoOperativoInvalido
	}
	decision, err := s.autorizarCatalogo(ctx, actor, orden.Orden.Credenciales.VinculoAutenticacionActor,
		ports.AccionPublicarCatalogoConfigurable, publicado, orden.Orden.Finalidad, orden.Orden.CorrelacionRef, orden.Orden.Motivo)
	if err != nil {
		return ports.ResultadoOperacionCatalogoOperativo{}, err
	}
	evidencia, err := s.evidenciaUsoDecisionCatalogo(ctx, decision)
	if err != nil {
		return ports.ResultadoOperacionCatalogoOperativo{}, err
	}
	resultado, err := repositorio.RecuperarOperacionCatalogoOperativo(ctx, ports.RecuperacionCatalogoOperativo{
		ClaveIdempotencia:    orden.ClaveIdempotencia,
		HuellaMaterialSHA256: material.huella,
		MaterialCanonico:     append([]byte(nil), material.bytes...),
		CatalogoID:           orden.Orden.ID,
		Version:              orden.Orden.Version,
		Accion:               ports.AccionPublicarCatalogoConfigurable,
		Autorizacion:         evidencia,
	})
	if err != nil {
		return ports.ResultadoOperacionCatalogoOperativo{}, err
	}
	if err := validarResultadoCatalogoOperativo(resultado, orden.ClaveIdempotencia, material.huella, ports.AccionPublicarCatalogoConfigurable,
		orden.Orden.ID, orden.Orden.Version, actor.Principal.ID); err != nil {
		return ports.ResultadoOperacionCatalogoOperativo{}, err
	}
	resultado.Recuperada = true
	return resultado, nil
}

const esquemaMaterialCatalogoOperativo = "vec.catalogos.operacion.v1"

// El material excluye la correlación, la decisión y los tiempos generados en
// cada intento. Incluye el contenido, motivo, finalidad, actor y precondiciones.
type materialCatalogoOperativo struct {
	Esquema              string
	Accion               string
	CatalogoID           string
	Version              int
	ActorRef             string
	Cabeza               ports.CabezaCatalogoOperativo
	Contenido            *contenidoCatalogoOperativo
	HuellaBorradorSHA256 string
	Finalidad            string
	Motivo               string
	AprobacionRef        string
}

type contenidoCatalogoOperativo struct {
	CatalogoID         string
	Version            int
	Revision           int
	VersionAnteriorRef string
	ModuloID           string
	Nombre             string
	Descripcion        string
	FuenteRef          string
	Entradas           []domain.EntradaCatalogoConfigurable
}

type canonicoMaterialCatalogoOperativo struct {
	bytes  []byte
	huella string
}

func huellaMaterialCatalogoOperativo(material materialCatalogoOperativo) (canonicoMaterialCatalogoOperativo, error) {
	canonico, err := json.Marshal(material)
	if err != nil {
		return canonicoMaterialCatalogoOperativo{}, ErrOrdenCatalogoInvalida
	}
	suma := sha256.Sum256(canonico)
	return canonicoMaterialCatalogoOperativo{bytes: canonico, huella: hex.EncodeToString(suma[:])}, nil
}

func validarOrdenCatalogoOperativo(clave string, cabeza ports.CabezaCatalogoOperativo, id string, version int) error {
	if len(clave) < 3 || len(clave) > 160 || !referenciaCatalogoOperativoValida(clave) || id == "" || id != strings.TrimSpace(id) ||
		cabeza.CatalogoID != id || cabeza.Version < 1 || version < 2 || cabeza.Version != version-1 ||
		cabeza.Estado != domain.EstadoCatalogoPublicado || !huellaCatalogoOperativoValida(cabeza.HuellaSHA256) {
		return ErrOrdenCatalogoInvalida
	}
	return nil
}

func huellaCatalogoOperativoValida(huella string) bool {
	if len(huella) != sha256.Size*2 {
		return false
	}
	for _, caracter := range huella {
		if (caracter < '0' || caracter > '9') && (caracter < 'a' || caracter > 'f') {
			return false
		}
	}
	return true
}

func referenciaCatalogoOperativoValida(ref string) bool {
	if len(ref) < 1 || len(ref) > 512 || ref != strings.TrimSpace(ref) {
		return false
	}
	for _, caracter := range ref {
		if caracter <= 32 || caracter == 127 {
			return false
		}
	}
	return true
}

func validarResultadoCatalogoOperativo(resultado ports.ResultadoOperacionCatalogoOperativo, clave, material, accion, id string, version int, actor string) error {
	recibo, catalogo := resultado.Recibo, resultado.Catalogo
	if catalogo.Validar() != nil || catalogo.ID != id || catalogo.Version != version ||
		recibo.ClaveIdempotencia != clave || recibo.HuellaMaterialSHA256 != material || recibo.Accion != accion ||
		recibo.CatalogoID != id || recibo.Version != version || recibo.Estado != catalogo.Estado || recibo.ActorRef != actor ||
		!referenciaCatalogoOperativoValida(recibo.Referencia) || !referenciaCatalogoOperativoValida(recibo.AuditoriaRef) ||
		!referenciaCatalogoOperativoValida(recibo.OutboxRef) || recibo.ConfirmadoEn.IsZero() ||
		recibo.ConfirmadoEn.Location() != time.UTC || !recibo.ConfirmadoEn.Equal(recibo.ConfirmadoEn.Truncate(time.Microsecond)) {
		return ports.ErrReciboCatalogoOperativoInvalido
	}
	fecha := catalogo.CreadoEn
	switch accion {
	case ports.AccionCrearCatalogoConfigurable:
		if catalogo.Estado != domain.EstadoCatalogoBorrador || catalogo.CreadoPor != actor {
			return ports.ErrReciboCatalogoOperativoInvalido
		}
	case ports.AccionPublicarCatalogoConfigurable:
		fecha = catalogo.PublicadoEn
		if catalogo.Estado != domain.EstadoCatalogoPublicado || catalogo.PublicadoPor != actor {
			return ports.ErrReciboCatalogoOperativoInvalido
		}
	default:
		return ports.ErrReciboCatalogoOperativoInvalido
	}
	huella, err := catalogo.HuellaSHA256()
	if err != nil || huella != recibo.HuellaSHA256 || !fecha.Equal(recibo.ConfirmadoEn) {
		return ports.ErrReciboCatalogoOperativoInvalido
	}
	return nil
}

// Este estado pertenece exclusivamente a una llamada sin goroutines propias.
// Nunca se instala en el ServicioCatalogos compartido ni se reutiliza.
type gobiernoCatalogoOperativoInvocacion struct {
	repositorio    ports.RepositorioCatalogosOperativos
	clave          string
	cabeza         ports.CabezaCatalogoOperativo
	huellaBorrador string
	finalidad      string
	motivo         string
	accion         string
	material       canonicoMaterialCatalogoOperativo
	preparado      domain.CatalogoConfigurable
	resultado      ports.ResultadoOperacionCatalogoOperativo
}

var _ ports.RepositorioGobiernoCatalogos = (*gobiernoCatalogoOperativoInvocacion)(nil)

func (g *gobiernoCatalogoOperativoInvocacion) ConfirmarAltaBorradorCatalogo(ctx context.Context, catalogo domain.CatalogoConfigurable,
	auditoria domain.AuditEntry, evento domain.Event, evidencia ports.EvidenciaUsoDecisionAutorizacion) error {
	if g.accion != ports.AccionCrearCatalogoConfigurable {
		return ErrOrdenCatalogoInvalida
	}
	canonico, err := catalogo.ClonarCanonico()
	if err != nil {
		return err
	}
	material, err := huellaMaterialCatalogoOperativo(materialCatalogoOperativo{
		Esquema:    esquemaMaterialCatalogoOperativo,
		Accion:     g.accion,
		CatalogoID: catalogo.ID,
		Version:    catalogo.Version,
		ActorRef:   catalogo.CreadoPor,
		Cabeza:     g.cabeza,
		Contenido: &contenidoCatalogoOperativo{
			CatalogoID:         canonico.ID,
			Version:            canonico.Version,
			Revision:           canonico.Revision,
			VersionAnteriorRef: canonico.VersionAnteriorRef,
			ModuloID:           canonico.ModuloID,
			Nombre:             canonico.Nombre,
			Descripcion:        canonico.Descripcion,
			FuenteRef:          canonico.FuenteRef,
			Entradas:           canonico.Entradas,
		},
		Finalidad: g.finalidad,
		Motivo:    g.motivo,
	})
	if err != nil {
		return err
	}
	g.material = material
	g.preparado = canonico
	resultado, err := g.repositorio.ConfirmarAltaBorradorCatalogoOperativo(ctx, ports.ConfirmacionCatalogoOperativo{
		ClaveIdempotencia:    g.clave,
		HuellaMaterialSHA256: material.huella,
		MaterialCanonico:     append([]byte(nil), material.bytes...),
		CabezaEsperada:       g.cabeza,
		Catalogo:             catalogo,
		Auditoria:            auditoria,
		Evento:               evento,
		Autorizacion:         evidencia,
	})
	if err != nil {
		return err
	}
	if err := validarResultadoCatalogoOperativo(resultado, g.clave, material.huella, g.accion, catalogo.ID, catalogo.Version, catalogo.CreadoPor); err != nil {
		return err
	}
	g.resultado = resultado
	return nil
}

func (g *gobiernoCatalogoOperativoInvocacion) ConfirmarPublicacionCatalogo(ctx context.Context, anterior string, catalogo domain.CatalogoConfigurable,
	auditoria domain.AuditEntry, evento domain.Event, evidencia ports.EvidenciaUsoDecisionAutorizacion) error {
	if g.accion != ports.AccionPublicarCatalogoConfigurable {
		return ErrOrdenCatalogoInvalida
	}
	if anterior != g.huellaBorrador {
		return ports.ErrRevisionCatalogoEnConflicto
	}
	resultado, err := g.repositorio.ConfirmarPublicacionCatalogoOperativo(ctx, ports.ConfirmacionCatalogoOperativo{
		ClaveIdempotencia:    g.clave,
		HuellaMaterialSHA256: g.material.huella,
		MaterialCanonico:     append([]byte(nil), g.material.bytes...),
		CabezaEsperada:       g.cabeza,
		HuellaAnteriorSHA256: anterior,
		Catalogo:             catalogo,
		Auditoria:            auditoria,
		Evento:               evento,
		Autorizacion:         evidencia,
	})
	if err != nil {
		return err
	}
	if err := validarResultadoCatalogoOperativo(resultado, g.clave, g.material.huella, g.accion, catalogo.ID, catalogo.Version, catalogo.PublicadoPor); err != nil {
		return err
	}
	g.resultado = resultado
	return nil
}

func (*gobiernoCatalogoOperativoInvocacion) ConfirmarActualizacionBorradorCatalogo(context.Context, string, domain.CatalogoConfigurable,
	domain.AuditEntry, domain.Event, ports.EvidenciaUsoDecisionAutorizacion) error {
	return ErrOrdenCatalogoInvalida
}

func (*gobiernoCatalogoOperativoInvocacion) ConfirmarRetiradaCatalogo(context.Context, string, domain.CatalogoConfigurable,
	domain.AuditEntry, domain.Event, ports.EvidenciaUsoDecisionAutorizacion) error {
	return ErrOrdenCatalogoInvalida
}
