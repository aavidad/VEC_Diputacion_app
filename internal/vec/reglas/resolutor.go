package reglas

import (
	"context"
	"errors"
	"reflect"
	"strconv"
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
	// Ajustes es opcional: sin él rigen los valores del catálogo base.
	Ajustes ConsultaAjustes
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
	if nulo(cfg.Ajustes) {
		cfg.Ajustes = nil
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
	return r.reglaEn(ctx, clave, time.Time{})
}

func (r *Resolutor) reglaEn(ctx context.Context, clave string, instanteAjustes time.Time) (Regla, error) {
	reglas, err := r.reglasEn(ctx, instanteAjustes)
	if err != nil {
		return Regla{}, err
	}
	for _, regla := range reglas {
		if regla.Clave == clave {
			if regla.AjusteNoAplicable {
				return Regla{}, ErrAjusteInvalido
			}
			return regla, nil
		}
	}
	return Regla{}, ErrReglaNoEncontrada
}

// Reglas devuelve todas las reglas vigentes en el orden del catálogo. Una
// entrada vigente con atributos no válidos invalida la consulta entera.
func (r *Resolutor) Reglas(ctx context.Context) ([]Regla, error) {
	return r.reglasEn(ctx, time.Time{})
}

// ReglasEn es una proyección de lectura para un instante pasado: no fija una
// versión al iniciar un plazo ni sustituye su instantánea durable. La base
// sigue siendo la vigente ahora, pues aún no se consulta su historia.
func (r *Resolutor) ReglasEn(ctx context.Context, instante time.Time) ([]Regla, error) {
	if instante.IsZero() {
		return nil, ErrReglasNoDisponibles
	}
	return r.reglasEn(ctx, instante)
}

// reglasEn resuelve el catálogo base vigente ahora y le aplica los ajustes
// vigentes en instanteAjustes (ahora, si es cero).
func (r *Resolutor) reglasEn(ctx context.Context, instanteAjustes time.Time) ([]Regla, error) {
	lectura, err := r.leerContexto(ctx, instanteAjustes)
	if err != nil {
		return nil, err
	}
	reglas := make([]Regla, 0, len(lectura.catalogo.Entradas))
	for _, entrada := range lectura.catalogo.Entradas {
		if !entrada.VigenteEn(lectura.instante) {
			continue
		}
		regla, err := reglaDesdeEntrada(lectura.catalogo, lectura.huella, lectura.ejemplo, entrada)
		if err != nil {
			return nil, err
		}
		if campos, ajustada := lectura.ajustes.Ajustes[regla.Clave]; lectura.conAjustes && ajustada {
			// Un ajuste que ya no encaja con su regla base deja esa regla
			// fuera de uso; nunca se vuelve en silencio al valor base.
			if ajustadaOK, err := aplicarAjuste(lectura.catalogo, lectura.huella, lectura.ejemplo, entrada, regla, lectura.ajustes, campos); err == nil {
				regla = ajustadaOK
			} else {
				regla.AjusteNoAplicable = true
			}
		}
		reglas = append(reglas, regla)
	}
	return reglas, nil
}

type contextoResolucion struct {
	catalogo   domain.CatalogoConfigurable
	huella     string
	instante   time.Time
	ejemplo    bool
	ajustes    VersionAjustes
	conAjustes bool
}

func (r *Resolutor) leerContexto(ctx context.Context, instanteAjustes time.Time) (contextoResolucion, error) {
	var vacio contextoResolucion
	if r == nil {
		return vacio, ErrReglasNoConfiguradas
	}
	if ctx == nil {
		return vacio, ErrReglasNoDisponibles
	}
	catalogo, instante, err := r.catalogoVigente(ctx)
	if err != nil {
		return vacio, err
	}
	huella, err := catalogo.HuellaSHA256()
	if err != nil {
		return vacio, ErrReglasNoDisponibles
	}
	ejemplo := catalogo.FuenteRef == MarcaPaqueteEjemplo
	if r.cfg.Metadatos != nil {
		metadatos, err := r.cfg.Metadatos.ObtenerMetadatosFuenteCatalogos(ctx)
		if err != nil {
			return vacio, ErrReglasNoDisponibles
		}
		ejemplo = ejemplo || metadatos.Demostracion
	}
	if instanteAjustes.IsZero() {
		instanteAjustes = instante
	}
	ajustes, conAjustes, err := r.ajustesEn(ctx, instanteAjustes.UTC())
	if err != nil {
		return vacio, err
	}
	return contextoResolucion{catalogo: catalogo, huella: huella, instante: instante,
		ejemplo: ejemplo, ajustes: ajustes, conAjustes: conAjustes}, nil
}

// PrepararInstantaneaRegla copia la regla base, su valor resuelto y la versión
// completa de ajustes leída ahora. Prepararla no inicia ni guarda un plazo:
// CT debe persistirla en la misma transacción que el inicio antes de usarla.
func (r *Resolutor) PrepararInstantaneaRegla(ctx context.Context, clave string) (InstantaneaRegla, error) {
	var vacia InstantaneaRegla
	if r == nil {
		return vacia, ErrReglasNoConfiguradas
	}
	if r.cfg.Ajustes == nil {
		return vacia, ErrAjustesNoDisponibles
	}
	lectura, err := r.leerContexto(ctx, time.Time{})
	if err != nil {
		return vacia, err
	}
	canonico, err := CanonicoAjustes(lectura.ajustes.Ajustes)
	if err != nil {
		return vacia, ErrAjustesNoDisponibles
	}
	huellaAjustes := lectura.ajustes.HuellaSHA256
	if !lectura.conAjustes {
		huellaAjustes, err = HuellaAjustes(nil)
		if err != nil {
			return vacia, ErrAjustesNoDisponibles
		}
	}
	for _, entrada := range lectura.catalogo.Entradas {
		if entrada.Clave != clave || !entrada.VigenteEn(lectura.instante) {
			continue
		}
		base, err := reglaDesdeEntrada(lectura.catalogo, lectura.huella, lectura.ejemplo, entrada)
		if err != nil {
			return vacia, err
		}
		efectiva := base
		if campos, ok := lectura.ajustes.Ajustes[clave]; lectura.conAjustes && ok {
			efectiva, err = aplicarAjuste(lectura.catalogo, lectura.huella, lectura.ejemplo, entrada, base, lectura.ajustes, campos)
			if err != nil {
				return vacia, ErrAjusteInvalido
			}
		}
		instantanea := InstantaneaRegla{datos: DatosInstantaneaRegla{
			Base: copiarRegla(base), Efectiva: copiarRegla(efectiva),
			CatalogoAjustesID: CatalogoAjustesDe(lectura.catalogo.ID), AjustesEncontrados: lectura.conAjustes,
			VersionAjustes: lectura.ajustes.Version, HuellaAjustes: huellaAjustes,
			CanonicoAjustes: canonico, AjustesVigenteDesde: lectura.ajustes.VigenteDesde.UTC(),
			PreparadaEn: lectura.instante,
		}}
		if !instantanea.valida() {
			return vacia, ErrReglasNoDisponibles
		}
		return instantanea, nil
	}
	return vacia, ErrReglaNoEncontrada
}

// ajustesEn lee la versión de ajustes vigente en el instante. Sin almacén de
// ajustes o sin ninguna versión todavía, rigen los valores base.
func (r *Resolutor) ajustesEn(ctx context.Context, instante time.Time) (VersionAjustes, bool, error) {
	if r.cfg.Ajustes == nil {
		return VersionAjustes{}, false, nil
	}
	id := CatalogoAjustesDe(r.cfg.CatalogoID)
	version, encontrada, err := r.cfg.Ajustes.AjustesVigentesEn(ctx, id, instante)
	if err != nil {
		if ctx.Err() != nil {
			return VersionAjustes{}, false, ctx.Err()
		}
		if errors.Is(err, ErrAjustesConflicto) {
			return VersionAjustes{}, false, ErrAjustesConflicto
		}
		return VersionAjustes{}, false, ErrAjustesNoDisponibles
	}
	if !encontrada {
		if !versionAjustesVacia(version) {
			return VersionAjustes{}, false, ErrAjustesNoDisponibles
		}
		return VersionAjustes{}, false, nil
	}
	if err := validarVersionAjustes(version, id, instante); err != nil {
		return VersionAjustes{}, false, err
	}
	return version, true, nil
}

// Vencimiento calcula con la base solo si no hay almacén de ajustes compuesto.
// Con ajustes configurados falla cerrado hasta que CT guarde una instantánea
// con el inicio real; consultar AjustesVigentesEn(inicio) no acredita esa versión.
func (r *Resolutor) Vencimiento(ctx context.Context, clave string, inicio time.Time, municipioSede string) (Regla, Vencimiento, error) {
	return r.vencimiento(ctx, clave, inicio, municipioSede, false)
}

// VencimientoUrgente calcula el vencimiento de un asunto urgente: usa el
// atributo cantidad_urgente de la regla si lo tiene y, si no, su cantidad
// ordinaria. Una cantidad urgente mal formada invalida la regla.
func (r *Resolutor) VencimientoUrgente(ctx context.Context, clave string, inicio time.Time, municipioSede string) (Regla, Vencimiento, error) {
	return r.vencimiento(ctx, clave, inicio, municipioSede, true)
}

func (r *Resolutor) vencimiento(ctx context.Context, clave string, inicio time.Time, municipioSede string, urgente bool) (Regla, Vencimiento, error) {
	if inicio.IsZero() {
		return Regla{}, Vencimiento{}, ErrReglaSinPlazo
	}
	if r != nil && r.cfg.Ajustes != nil {
		return Regla{}, Vencimiento{}, ErrAjustesNoDisponibles
	}
	regla, err := r.reglaEn(ctx, clave, time.Time{})
	if err != nil {
		return Regla{}, Vencimiento{}, err
	}
	return r.calcularVencimiento(ctx, regla, inicio, municipioSede, urgente)
}

// CalcularConInstantanea usa únicamente el valor preparado, sin volver a
// consultar el catálogo ni los ajustes. El consumidor futuro deberá acreditar
// que la instantánea se guardó con el inicio real antes de llamar.
func (r *Resolutor) CalcularConInstantanea(ctx context.Context, instantanea InstantaneaRegla, inicio time.Time, municipioSede string, urgente bool) (Regla, Vencimiento, error) {
	if r == nil || ctx == nil || !instantanea.valida() ||
		instantanea.datos.Base.ReferenciaEntrada.CatalogoID != r.cfg.CatalogoID {
		return Regla{}, Vencimiento{}, ErrReglasNoDisponibles
	}
	return r.calcularVencimiento(ctx, copiarRegla(instantanea.datos.Efectiva), inicio, municipioSede, urgente)
}

func (r *Resolutor) calcularVencimiento(ctx context.Context, regla Regla, inicio time.Time, municipioSede string, urgente bool) (Regla, Vencimiento, error) {
	if !regla.Unidad.EsPlazo() || regla.Computo == "" || inicio.IsZero() {
		return Regla{}, Vencimiento{}, ErrReglaSinPlazo
	}
	cantidad, err := cantidadPlazo(regla, urgente)
	if err != nil {
		return Regla{}, Vencimiento{}, err
	}
	if r.cfg.Calculadora == nil {
		return Regla{}, Vencimiento{}, ErrCalculoNoDisponible
	}
	sede := strings.TrimSpace(municipioSede)
	if sede == "" {
		sede = r.cfg.MunicipioSede
	}
	vencimiento, err := r.cfg.Calculadora.CalcularVencimiento(ctx, SolicitudVencimiento{
		Inicio: inicio, Unidad: regla.Unidad, Cantidad: cantidad,
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

// cantidadPlazo es la cantidad ordinaria o, para un asunto urgente con el
// atributo cantidad_urgente, esa cantidad canónica entre 1 y el máximo.
func cantidadPlazo(regla Regla, urgente bool) (int, error) {
	texto, conUrgente := regla.Atributos[AtributoCantidadUrgente]
	if !urgente || !conUrgente {
		return regla.Cantidad, nil
	}
	valor, err := strconv.Atoi(texto)
	if err != nil || valor < 1 || valor > maximoCantidadRegla || strconv.Itoa(valor) != texto {
		return 0, ErrReglaInvalida
	}
	return valor, nil
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
