package auditoria

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"slices"
	"strings"
	"time"
)

// CapturaLecturaSesion procede de la sesión corporativa y de la evaluación
// positiva del permiso de consulta. No es material de consumo de un acto V3.
type CapturaLecturaSesion struct {
	ContextoCanonico []byte
	VinculoCanonico  []byte
	Orden            json.RawMessage
	OrganizacionRef  string
}

func CamposLecturaSesion() []string {
	campos := append(CamposPermitidos(), "canal", "finalidad_ref", "perfil_ref")
	slices.Sort(campos)
	return campos
}

type ConsultaSesion struct {
	Filtro  Filtro
	Captura CapturaLecturaSesion
}

type Busqueda struct {
	Texto      string
	DespuesRef string
}

type Objetivo struct {
	Referencia string `json:"referencia"`
	Etiqueta   string `json:"etiqueta"`
}

type PaginaObjetivos struct {
	Objetivos  []Objetivo `json:"objetivos"`
	DespuesRef string     `json:"despues_ref"`
}

// Cada método confirma el asiento común en la misma transacción que los datos.
// Las referencias de pertenencia las resuelve el módulo que las conserva.
type FuenteLecturaSesion interface {
	ConsultarAuditoriaSesion(context.Context, ConsultaSesion) (PaginaFuente, error)
	BuscarObjetivos(context.Context, ConsultaSesion, Busqueda) ([]Objetivo, error)
	RegistrarOpciones(context.Context, ConsultaSesion) error
}

type CapturadorLecturaSesion interface {
	CapturarLecturaAuditoria(context.Context, ContextoConsulta, Filtro, string, Busqueda) (CapturaLecturaSesion, error)
}

func NuevoServicioLecturaSesion(c CapturadorLecturaSesion, f FuenteLecturaSesion) (*Servicio, error) {
	if dependenciaNula(c) || dependenciaNula(f) {
		return nil, ErrNoDisponible
	}
	s := &Servicio{capturador: c, lectura: f, ahora: time.Now}
	if _, err := rand.Read(s.claveCursor[:]); err != nil {
		return nil, ErrNoDisponible
	}
	return s, nil
}

func (s *Servicio) prepararLectura(ctx context.Context, p Peticion, operacion string, busqueda Busqueda) (ConsultaSesion, error) {
	if s == nil || ctx == nil || ctx.Err() != nil || dependenciaNula(s.capturador) || dependenciaNula(s.lectura) {
		return ConsultaSesion{}, ErrNoDisponible
	}
	if p.Contexto.Resultado.Validar() != nil || p.Contexto.Vinculo.ValidarPara(p.Contexto.Resultado) != nil ||
		!p.Contexto.Vinculo.VigenteEn(s.ahora().UTC().Truncate(time.Microsecond), p.Contexto.Resultado) ||
		p.Contexto.Motivo.Validar() != nil || p.Contexto.Motivo.Referencia() != p.Filtro.MotivoRef ||
		p.Contexto.Correlacion.Validar() != nil || p.Filtro.Validar() != nil ||
		len(p.Filtro.MotivoConsulta) < 3 || len(p.Filtro.MotivoConsulta) > 500 || !textoMotivoMinimizado(p.Filtro.MotivoConsulta) {
		return ConsultaSesion{}, ErrDenegada
	}
	p.Filtro.AutorLecturaRef = p.Contexto.Resultado.Contexto.Principal.ID + ":" + p.Contexto.Resultado.Contexto.PerfilActivoRef
	if !p.Filtro.Antes.vacia() {
		return ConsultaSesion{}, ErrDenegada
	}
	if p.Cursor != "" {
		pos, err := s.decodificarCursor(p.Cursor, p.Filtro)
		if err != nil || pos.Fuente != p.Filtro.Fuente {
			return ConsultaSesion{}, ErrDenegada
		}
		p.Filtro.Antes = pos
	}
	captura, err := s.capturador.CapturarLecturaAuditoria(ctx, p.Contexto, p.Filtro, operacion, busqueda)
	if err != nil {
		return ConsultaSesion{}, err
	}
	if captura.OrganizacionRef == "" || len(captura.ContextoCanonico) == 0 || len(captura.VinculoCanonico) == 0 || len(captura.Orden) == 0 {
		return ConsultaSesion{}, ErrNoDisponible
	}
	return ConsultaSesion{Filtro: p.Filtro, Captura: captura}, nil
}

func (s *Servicio) consultarSesion(ctx context.Context, p Peticion) (Pagina, error) {
	q, err := s.prepararLectura(ctx, p, "consultar", Busqueda{})
	if err != nil {
		return Pagina{}, err
	}
	filas, err := s.lectura.ConsultarAuditoriaSesion(ctx, q)
	if err != nil {
		return Pagina{}, err
	}
	if ctx.Err() != nil {
		return Pagina{}, ErrNoDisponible
	}
	return s.prepararPagina(q.Filtro, filas, true)
}

func (s *Servicio) BuscarObjetivos(ctx context.Context, p Peticion, b Busqueda) (PaginaObjetivos, error) {
	if len(b.Texto) > 80 || b.Texto != strings.TrimSpace(b.Texto) ||
		(b.Texto != "" && !textoMotivoMinimizado(b.Texto)) ||
		(b.DespuesRef != "" && !referenciaExacta(b.DespuesRef, 512)) || p.Filtro.Limite > 50 {
		return PaginaObjetivos{}, ErrDenegada
	}
	q, err := s.prepararLectura(ctx, p, "buscar", b)
	if err != nil {
		return PaginaObjetivos{}, err
	}
	objetivos, err := s.lectura.BuscarObjetivos(ctx, q, b)
	if err != nil {
		return PaginaObjetivos{}, err
	}
	if ctx.Err() != nil {
		return PaginaObjetivos{}, ErrFuenteInvalida
	}
	if err := ValidarObjetivosFuenteSesion(p.Filtro.Limite, b.DespuesRef, objetivos); err != nil {
		return PaginaObjetivos{}, err
	}
	resultado := PaginaObjetivos{Objetivos: append([]Objetivo{}, objetivos...)}
	if len(objetivos) > int(p.Filtro.Limite) {
		resultado.Objetivos = resultado.Objetivos[:p.Filtro.Limite]
		resultado.DespuesRef = resultado.Objetivos[len(resultado.Objetivos)-1].Referencia
	}
	return resultado, nil
}

func ValidarObjetivosFuenteSesion(limite uint16, despuesRef string, objetivos []Objetivo) error {
	if limite == 0 || limite > 50 || len(objetivos) > int(limite)+1 {
		return ErrFuenteInvalida
	}
	previa := despuesRef
	for _, o := range objetivos {
		if !referenciaExacta(o.Referencia, 512) || o.Referencia <= previa ||
			len(o.Etiqueta) == 0 || len(o.Etiqueta) > 160 || !textoMotivoMinimizado(o.Etiqueta) {
			return ErrFuenteInvalida
		}
		previa = o.Referencia
	}
	return nil
}

func (s *Servicio) registrarOpcionesSesion(ctx context.Context, identidad IdentidadResuelta, o Opciones) error {
	if s == nil || s.lectura == nil {
		return nil
	}
	p := Peticion{Filtro: Filtro{Fuente: "ct", ExpedienteRef: "auditoria:opciones", Desde: time.Unix(0, 0).UTC(),
		Hasta: time.Unix(86400, 0).UTC(), Limite: 1, FinalidadRef: o.FinalidadRef, MotivoRef: o.MotivoRef,
		MotivoConsulta: "opciones_consulta_auditoria"}, Contexto: contextoConsulta(identidad, o)}
	q, err := s.prepararLectura(ctx, p, "opciones", Busqueda{})
	if err != nil {
		return err
	}
	return s.lectura.RegistrarOpciones(ctx, q)
}

func contextoConsulta(i IdentidadResuelta, o Opciones) ContextoConsulta {
	return ContextoConsulta{Vinculo: i.Vinculo, Resultado: i.Resultado, Motivo: o.Motivo, Correlacion: i.Correlacion}
}
