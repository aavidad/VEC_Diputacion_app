package reglas

import (
	"context"
	"reflect"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// Reloj fija el instante con el que se elige la versión vigente.
type Reloj interface {
	Ahora() time.Time
}

// SolicitudVencimiento es neutral respecto a Calendarios: la composición la
// traduce a su contrato. Inicio es el instante del hecho (contacto efectivo,
// notificación, fin de la relación); su fecha civil se toma en hora peninsular.
type SolicitudVencimiento struct {
	Inicio        time.Time
	Unidad        Unidad
	Cantidad      int
	Computo       Computo
	MunicipioSede string
}

// Vencimiento es el último día del plazo y el primer instante en que ya ha
// vencido: 00:00 del día siguiente en Europe/Madrid, de modo que el plazo
// termina a las 23:59:59 hora peninsular.
type Vencimiento struct {
	UltimoDia    string
	VenceAntesDe time.Time
	Prorrogado   bool
	// Calendarios identifica las versiones usadas; vacío en cómputo civil.
	Calendarios []string
}

// CalculadoraPlazos es el puerto hacia el cálculo de plazos. Una
// indisponibilidad nunca se sustituye por una fecha supuesta.
type CalculadoraPlazos interface {
	CalcularVencimiento(context.Context, SolicitudVencimiento) (Vencimiento, error)
}

// Configuracion describe un único catálogo de reglas de un módulo.
type Configuracion struct {
	Consulta   ports.ConsultaCatalogosConfigurablesAcotada
	Metadatos  ports.ConsultaMetadatosFuenteCatalogos
	CatalogoID string
	ModuloID   string
	Reloj      Reloj
	// Calculadora es opcional: sin ella las reglas se resuelven pero no se
	// calculan vencimientos.
	Calculadora CalculadoraPlazos
	// MunicipioSede se usa cuando la solicitud no indica otro.
	MunicipioSede string
}

// Resolutor lee el catálogo en cada consulta; no guarda estado mutable. Un
// puntero nulo es válido y responde ErrReglasNoConfiguradas, para que los
// módulos mantengan su conducta actual sin catálogo.
type Resolutor struct {
	cfg Configuracion
}

func NuevoResolutor(cfg Configuracion) (*Resolutor, error) {
	if nulo(cfg.Consulta) || nulo(cfg.Reloj) || !claveCanonica(cfg.CatalogoID) || !claveCanonica(cfg.ModuloID) {
		return nil, ErrConfiguracion
	}
	if nulo(cfg.Metadatos) {
		cfg.Metadatos = nil
	}
	if nulo(cfg.Calculadora) {
		cfg.Calculadora = nil
	}
	cfg.MunicipioSede = strings.TrimSpace(cfg.MunicipioSede)
	return &Resolutor{cfg: cfg}, nil
}

// Disponible indica si hay catálogo compuesto (no si está vigente).
func (r *Resolutor) Disponible() bool { return r != nil }

// CatalogoID identifica el catálogo que resuelve este resolutor.
func (r *Resolutor) CatalogoID() string {
	if r == nil {
		return ""
	}
	return r.cfg.CatalogoID
}

// Regla devuelve la regla vigente con esa clave.
func (r *Resolutor) Regla(ctx context.Context, clave string) (Regla, error) {
	reglas, err := r.Reglas(ctx)
	if err != nil {
		return Regla{}, err
	}
	for _, regla := range reglas {
		if regla.Clave == clave {
			return regla, nil
		}
	}
	return Regla{}, ErrReglaNoEncontrada
}

// Reglas devuelve todas las reglas vigentes en el orden del catálogo. Una
// entrada vigente con atributos no válidos invalida la consulta entera.
func (r *Resolutor) Reglas(ctx context.Context) ([]Regla, error) {
	if r == nil {
		return nil, ErrReglasNoConfiguradas
	}
	if ctx == nil {
		return nil, ErrReglasNoDisponibles
	}
	catalogo, instante, err := r.catalogoVigente(ctx)
	if err != nil {
		return nil, err
	}
	huella, err := catalogo.HuellaSHA256()
	if err != nil {
		return nil, ErrReglasNoDisponibles
	}
	ejemplo := catalogo.FuenteRef == MarcaPaqueteEjemplo
	if r.cfg.Metadatos != nil {
		metadatos, err := r.cfg.Metadatos.ObtenerMetadatosFuenteCatalogos(ctx)
		if err != nil {
			return nil, ErrReglasNoDisponibles
		}
		ejemplo = ejemplo || metadatos.Demostracion
	}
	reglas := make([]Regla, 0, len(catalogo.Entradas))
	for _, entrada := range catalogo.Entradas {
		if !entrada.VigenteEn(instante) {
			continue
		}
		regla, err := reglaDesdeEntrada(catalogo, huella, ejemplo, entrada)
		if err != nil {
			return nil, err
		}
		reglas = append(reglas, regla)
	}
	return reglas, nil
}

// Vencimiento resuelve la regla y calcula su vencimiento desde inicio con el
// cómputo que declara. Devuelve la regla para que el consumidor conserve su
// referencia y huella junto a la fecha.
func (r *Resolutor) Vencimiento(ctx context.Context, clave string, inicio time.Time, municipioSede string) (Regla, Vencimiento, error) {
	regla, err := r.Regla(ctx, clave)
	if err != nil {
		return Regla{}, Vencimiento{}, err
	}
	if !regla.Unidad.EsPlazo() || regla.Computo == "" || inicio.IsZero() {
		return Regla{}, Vencimiento{}, ErrReglaSinPlazo
	}
	if r.cfg.Calculadora == nil {
		return Regla{}, Vencimiento{}, ErrCalculoNoDisponible
	}
	sede := strings.TrimSpace(municipioSede)
	if sede == "" {
		sede = r.cfg.MunicipioSede
	}
	vencimiento, err := r.cfg.Calculadora.CalcularVencimiento(ctx, SolicitudVencimiento{
		Inicio: inicio, Unidad: regla.Unidad, Cantidad: regla.Cantidad,
		Computo: regla.Computo, MunicipioSede: sede,
	})
	if err != nil {
		if ctx.Err() != nil {
			return Regla{}, Vencimiento{}, ctx.Err()
		}
		return Regla{}, Vencimiento{}, ErrCalculoNoDisponible
	}
	if vencimiento.UltimoDia == "" || vencimiento.VenceAntesDe.IsZero() || !vencimiento.VenceAntesDe.After(inicio) {
		return Regla{}, Vencimiento{}, ErrCalculoNoDisponible
	}
	vencimiento.Calendarios = append([]string(nil), vencimiento.Calendarios...)
	return regla, vencimiento, nil
}

func limitesConsultaReglas() ports.LimitesConsultaCatalogosAcotada {
	return ports.LimitesConsultaCatalogosAcotada{
		Versiones: 16, Entradas: 1_024, Atributos: 4_096, BytesAproximados: 1 << 20,
	}
}

// catalogoVigente elige la versión publicada más alta ya vigente. Una
// versión retirada solo cuenta mientras no haya llegado su retirada.
func (r *Resolutor) catalogoVigente(ctx context.Context) (domain.CatalogoConfigurable, time.Time, error) {
	var vacio domain.CatalogoConfigurable
	if err := ctx.Err(); err != nil {
		return vacio, time.Time{}, err
	}
	instante := r.cfg.Reloj.Ahora().UTC()
	if instante.IsZero() {
		return vacio, time.Time{}, ErrReglasNoDisponibles
	}
	limites := limitesConsultaReglas()
	resultado, err := r.cfg.Consulta.ListarVersionesCatalogoAcotado(ctx, r.cfg.CatalogoID, limites)
	if err != nil {
		if ctx.Err() != nil {
			return vacio, time.Time{}, ctx.Err()
		}
		return vacio, time.Time{}, ErrReglasNoDisponibles
	}
	if resultado.Truncado || len(resultado.Catalogos) == 0 || len(resultado.Catalogos) > limites.Versiones {
		return vacio, time.Time{}, ErrReglasNoDisponibles
	}
	elegido, encontrado := vacio, false
	for _, candidato := range resultado.Catalogos {
		catalogo, err := candidato.ClonarCanonico()
		if err != nil || catalogo.ID != r.cfg.CatalogoID || catalogo.ModuloID != r.cfg.ModuloID {
			return vacio, time.Time{}, ErrReglasNoDisponibles
		}
		if !catalogoVigenteEn(catalogo, instante) || (encontrado && catalogo.Version <= elegido.Version) {
			continue
		}
		elegido, encontrado = catalogo, true
	}
	if !encontrado {
		return vacio, time.Time{}, ErrReglasNoDisponibles
	}
	return elegido, instante, nil
}

func catalogoVigenteEn(catalogo domain.CatalogoConfigurable, instante time.Time) bool {
	switch catalogo.Estado {
	case domain.EstadoCatalogoPublicado:
		return !catalogo.PublicadoEn.After(instante)
	case domain.EstadoCatalogoRetirado:
		return !catalogo.PublicadoEn.After(instante) && catalogo.RetiradoEn.After(instante)
	default:
		return false
	}
}

func claveCanonica(valor string) bool {
	return valor != "" && valor == strings.TrimSpace(valor) &&
		(domain.ReferenciaEntradaCatalogo{
			CatalogoID: valor, CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("0", 64), EntradaClave: valor,
		}).Validar() == nil
}

func nulo(dependencia any) bool {
	if dependencia == nil {
		return true
	}
	valor := reflect.ValueOf(dependencia)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}
