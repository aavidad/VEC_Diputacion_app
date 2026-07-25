package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	TiempoMaximoPreparacionGlobalCobertura  = 3 * time.Second
	ConcurrenciaMaximaPreparacionGlobal     = 16
	maximoEvidenciasPorViaPreparacionGlobal = 32
	maximoEnteroSeguroPreparacionGlobal     = uint64(1<<53 - 1)
	redaccionDatosPreparacionGlobal         = "[DATOS-PREPARACION-GLOBAL-COBERTURA-OPACOS]"
)

var (
	ErrPreparadorGlobalCoberturaInvalido = errors.New(
		"contratacion temporal: preparador global de cobertura invalido",
	)
	ErrDatosPreparacionGlobalCoberturaInvalidos = errors.New(
		"contratacion temporal: datos de preparacion global de cobertura invalidos",
	)
	ErrReferenciaPreparacionGlobalCoberturaRepetida = errors.New(
		"contratacion temporal: referencia de comprobacion de cobertura repetida",
	)
	ErrPreparacionGlobalCoberturaNoConfiable = errors.New(
		"contratacion temporal: preparacion global de cobertura no confiable",
	)
	ErrSerializacionDatosPreparacionGlobalCoberturaProhibida = errors.New(
		"contratacion temporal: serializacion de datos de preparacion global prohibida",
	)
)

// GeneradorReferenciasComprobacionCobertura fabrica referencias opacas sin
// recibir material funcional. Así no puede derivarlas de identidad, vía,
// categoría ni otros datos de la petición.
type GeneradorReferenciasComprobacionCobertura interface {
	NuevaReferenciaComprobacionCobertura(context.Context) (string, error)
}

// datosPreparacionGlobalCobertura conserva únicamente una instantánea ya
// resuelta por el servidor. El tipo y su fábrica son privados para que ningún
// adaptador de entrada pueda fabricar autoridad ni tratarlo como DTO de canal.
type datosPreparacionGlobalCobertura struct {
	analisisRef       string
	analisisHuella    string
	catalogo          domain.CatalogoViasCobertura
	politica          domain.PoliticaDecisionCobertura
	organizacionRef   string
	expedienteRef     string
	versionExpediente uint64
	categoriaRef      string
	periodo           domain.PeriodoPrevisto
}

func nuevosDatosPreparacionGlobalCobertura(
	analisisRef string,
	analisisHuellaSHA256 string,
	catalogo domain.CatalogoViasCobertura,
	politica domain.PoliticaDecisionCobertura,
	organizacionRef string,
	expedienteRef string,
	versionExpediente uint64,
	categoriaRef string,
	periodo domain.PeriodoPrevisto,
) (datosPreparacionGlobalCobertura, error) {
	catalogoClonado, errCatalogo := domain.RestaurarCatalogoViasCobertura(
		catalogo.Publicacion(),
	)
	politicaClonada, errPolitica := domain.RestaurarPoliticaDecisionCobertura(
		politica.Publicacion(),
		catalogoClonado,
	)
	datos := datosPreparacionGlobalCobertura{
		analisisRef: analisisRef, analisisHuella: analisisHuellaSHA256,
		catalogo: catalogoClonado, politica: politicaClonada,
		organizacionRef: organizacionRef, expedienteRef: expedienteRef,
		versionExpediente: versionExpediente, categoriaRef: categoriaRef,
		periodo: periodo,
	}
	if errCatalogo != nil || errPolitica != nil ||
		datos.validarEstructura() != nil {
		return datosPreparacionGlobalCobertura{},
			ErrDatosPreparacionGlobalCoberturaInvalidos
	}
	return datos, nil
}

func (d datosPreparacionGlobalCobertura) validarEstructura() error {
	if !domain.ReferenciaOpacaValida(d.analisisRef) ||
		!huellaSHA256PreparacionGlobalValida(d.analisisHuella) ||
		d.catalogo.Validar() != nil ||
		d.politica.Identidad().Validar() != nil ||
		!domain.ReferenciaOpacaValida(d.organizacionRef) ||
		d.politica.Publicacion().OrganizacionRef != d.organizacionRef ||
		!domain.ReferenciaOpacaValida(d.expedienteRef) ||
		d.versionExpediente == 0 ||
		d.versionExpediente > maximoEnteroSeguroPreparacionGlobal ||
		!domain.ReferenciaOpacaValida(d.categoriaRef) ||
		d.periodo.Validar() != nil {
		return ErrDatosPreparacionGlobalCoberturaInvalidos
	}
	return nil
}

func (d datosPreparacionGlobalCobertura) validarEn(instante time.Time) error {
	if d.validarEstructura() != nil ||
		!domain.InstanteUTCCanonico(instante) ||
		!d.catalogo.VigenteEn(instante) {
		return ErrDatosPreparacionGlobalCoberturaInvalidos
	}
	finalidadClave, finalidadRef := d.politica.Finalidad()
	if d.politica.ValidarPara(
		d.catalogo,
		d.organizacionRef,
		finalidadClave,
		finalidadRef,
		instante,
	) != nil {
		return ErrDatosPreparacionGlobalCoberturaInvalidos
	}
	return nil
}

func (datosPreparacionGlobalCobertura) String() string {
	return redaccionDatosPreparacionGlobal
}
func (d datosPreparacionGlobalCobertura) GoString() string { return d.String() }
func (d datosPreparacionGlobalCobertura) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, d.String())
}
func (d datosPreparacionGlobalCobertura) LogValue() slog.Value {
	return slog.StringValue(d.String())
}
func (datosPreparacionGlobalCobertura) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionDatosPreparacionGlobalCoberturaProhibida
}
func (*datosPreparacionGlobalCobertura) UnmarshalJSON([]byte) error {
	return ErrSerializacionDatosPreparacionGlobalCoberturaProhibida
}
func (datosPreparacionGlobalCobertura) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionDatosPreparacionGlobalCoberturaProhibida
}
func (*datosPreparacionGlobalCobertura) UnmarshalText([]byte) error {
	return ErrSerializacionDatosPreparacionGlobalCoberturaProhibida
}

type PreparadorGlobalCobertura struct {
	consultas    *PreparadorConsultaCobertura
	referencias  GeneradorReferenciasComprobacionCobertura
	reloj        ports.Reloj
	concurrencia int
	tiempoMaximo time.Duration
}

func NuevoPreparadorGlobalCobertura(
	consultas *PreparadorConsultaCobertura,
	referencias GeneradorReferenciasComprobacionCobertura,
	reloj ports.Reloj,
	concurrencia int,
	tiempoMaximo time.Duration,
) (*PreparadorGlobalCobertura, error) {
	if dependenciaPreparadorGlobalNula(consultas) ||
		dependenciaPreparadorGlobalNula(referencias) ||
		dependenciaPreparadorGlobalNula(reloj) ||
		concurrencia < 1 ||
		concurrencia > ConcurrenciaMaximaPreparacionGlobal ||
		tiempoMaximo <= 0 ||
		tiempoMaximo > TiempoMaximoPreparacionGlobalCobertura {
		return nil, ErrPreparadorGlobalCoberturaInvalido
	}
	return &PreparadorGlobalCobertura{
		consultas: consultas, referencias: referencias, reloj: reloj,
		concurrencia: concurrencia, tiempoMaximo: tiempoMaximo,
	}, nil
}

func (p *PreparadorGlobalCobertura) Preparar(
	ctx context.Context,
	datos datosPreparacionGlobalCobertura,
) (cobertura.PreparacionConjuntosViasCobertura, error) {
	if ctx == nil || p == nil ||
		dependenciaPreparadorGlobalNula(p.consultas) ||
		dependenciaPreparadorGlobalNula(p.referencias) ||
		dependenciaPreparadorGlobalNula(p.reloj) ||
		p.concurrencia < 1 ||
		p.concurrencia > ConcurrenciaMaximaPreparacionGlobal ||
		p.tiempoMaximo <= 0 ||
		p.tiempoMaximo > TiempoMaximoPreparacionGlobalCobertura {
		return cobertura.PreparacionConjuntosViasCobertura{},
			ErrPreparadorGlobalCoberturaInvalido
	}
	if err := ctx.Err(); err != nil {
		return cobertura.PreparacionConjuntosViasCobertura{}, err
	}
	operacion, cancelar := context.WithTimeout(ctx, p.tiempoMaximo)
	defer cancelar()
	inicio := instanteCanonico(p.reloj.Ahora())
	if datos.validarEn(inicio) != nil {
		return cobertura.PreparacionConjuntosViasCobertura{},
			ErrDatosPreparacionGlobalCoberturaInvalidos
	}
	vias := datos.catalogo.Vias()
	total, err := validarCargaPreparacionGlobal(vias)
	if err != nil {
		return cobertura.PreparacionConjuntosViasCobertura{}, err
	}
	trabajos, evidencias, err := p.prepararTrabajos(
		operacion,
		datos,
		vias,
		total,
		inicio,
	)
	if err != nil {
		return cobertura.PreparacionConjuntosViasCobertura{}, err
	}
	if err := p.ejecutarTrabajos(
		ctx,
		operacion,
		cancelar,
		trabajos,
		evidencias,
	); err != nil {
		return cobertura.PreparacionConjuntosViasCobertura{}, err
	}
	final := instanteCanonico(p.reloj.Ahora())
	if err := operacion.Err(); err != nil {
		return cobertura.PreparacionConjuntosViasCobertura{},
			errorPreparacionGlobalContexto(err)
	}
	if final.Before(inicio) || datos.validarEn(final) != nil {
		return cobertura.PreparacionConjuntosViasCobertura{},
			ErrPreparacionGlobalCoberturaNoConfiable
	}
	return construirPreparacionGlobal(
		operacion,
		datos,
		vias,
		evidencias,
		final,
	)
}

type trabajoPreparacionGlobal struct {
	viaIndice          int
	comprobacionIndice int
	solicitud          ports.SolicitudConsultarCobertura
}

func (p *PreparadorGlobalCobertura) prepararTrabajos(
	ctx context.Context,
	datos datosPreparacionGlobalCobertura,
	vias []domain.DefinicionViaCobertura,
	total int,
	inicio time.Time,
) (
	[]trabajoPreparacionGlobal,
	[][]cobertura.EvidenciaConsultaCobertura,
	error,
) {
	trabajos := make([]trabajoPreparacionGlobal, 0, total)
	evidencias := make([][]cobertura.EvidenciaConsultaCobertura, len(vias))
	referencias := make(map[string]struct{}, total)
	for viaIndice, via := range vias {
		evidencias[viaIndice] = make(
			[]cobertura.EvidenciaConsultaCobertura,
			len(via.Comprobaciones),
		)
		for comprobacionIndice, comprobacion := range via.Comprobaciones {
			if err := ctx.Err(); err != nil {
				return nil, nil, errorPreparacionGlobalContexto(err)
			}
			referencia, err :=
				p.referencias.NuevaReferenciaComprobacionCobertura(ctx)
			if err != nil {
				return nil, nil, errorPreparacionGlobalContexto(ctx.Err())
			}
			if !domain.ReferenciaOpacaValida(referencia) {
				return nil, nil, ErrPreparacionGlobalCoberturaNoConfiable
			}
			if _, repetida := referencias[referencia]; repetida {
				return nil, nil,
					ErrReferenciaPreparacionGlobalCoberturaRepetida
			}
			referencias[referencia] = struct{}{}
			trabajos = append(trabajos, trabajoPreparacionGlobal{
				viaIndice: viaIndice, comprobacionIndice: comprobacionIndice,
				solicitud: ports.SolicitudConsultarCobertura{
					PeticionRef:       referencia,
					OrganizacionRef:   datos.organizacionRef,
					ExpedienteRef:     datos.expedienteRef,
					VersionExpediente: datos.versionExpediente,
					Catalogo:          datos.catalogo.Identidad(),
					ViaClave:          via.Clave, Comprobacion: comprobacion,
					CategoriaRef: datos.categoriaRef, Periodo: datos.periodo,
					SolicitadaEn: inicio,
				},
			})
		}
	}
	return trabajos, evidencias, nil
}

func (p *PreparadorGlobalCobertura) ejecutarTrabajos(
	padre context.Context,
	ctx context.Context,
	cancelar context.CancelFunc,
	trabajos []trabajoPreparacionGlobal,
	evidencias [][]cobertura.EvidenciaConsultaCobertura,
) error {
	var siguiente atomic.Uint64
	var espera sync.WaitGroup
	var unaVez sync.Once
	var primerFallo error
	obreros := p.concurrencia
	if obreros > len(trabajos) {
		obreros = len(trabajos)
	}
	espera.Add(obreros)
	for indice := 0; indice < obreros; indice++ {
		go func() {
			defer espera.Done()
			for {
				if ctx.Err() != nil {
					return
				}
				posicion := int(siguiente.Add(1) - 1)
				if posicion >= len(trabajos) {
					return
				}
				trabajo := trabajos[posicion]
				evidencia, err := p.consultas.ConsultarConEvidencia(
					ctx,
					trabajo.solicitud,
				)
				if err != nil {
					unaVez.Do(func() {
						primerFallo = err
						cancelar()
					})
					return
				}
				evidencias[trabajo.viaIndice][trabajo.comprobacionIndice] =
					evidencia
			}
		}()
	}
	espera.Wait()
	if err := padre.Err(); err != nil {
		return errorPreparacionGlobalContexto(err)
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return errorPreparacionGlobalContexto(context.DeadlineExceeded)
	}
	if primerFallo != nil {
		return ErrPreparacionGlobalCoberturaNoConfiable
	}
	if err := ctx.Err(); err != nil {
		return errorPreparacionGlobalContexto(err)
	}
	return nil
}

func construirPreparacionGlobal(
	ctx context.Context,
	datos datosPreparacionGlobalCobertura,
	vias []domain.DefinicionViaCobertura,
	evidencias [][]cobertura.EvidenciaConsultaCobertura,
	instante time.Time,
) (cobertura.PreparacionConjuntosViasCobertura, error) {
	finalidadClave, finalidadRef := datos.politica.Finalidad()
	conjuntos := make([]cobertura.ConjuntoEvidenciasCobertura, len(vias))
	for indice, via := range vias {
		if err := ctx.Err(); err != nil {
			return cobertura.PreparacionConjuntosViasCobertura{},
				errorPreparacionGlobalContexto(err)
		}
		conjunto, err := cobertura.NuevoConjuntoEvidenciasCobertura(
			cobertura.CoordenadasConjuntoEvidencias{
				OrganizacionRef:   datos.organizacionRef,
				ExpedienteRef:     datos.expedienteRef,
				VersionExpediente: datos.versionExpediente,
				Catalogo:          datos.catalogo.Identidad(),
				Politica:          datos.politica.Identidad(),
				FinalidadClave:    finalidadClave, FinalidadRef: finalidadRef,
				ViaClave: via.Clave, CategoriaRef: datos.categoriaRef,
				Periodo: datos.periodo,
			},
			datos.catalogo,
			datos.politica,
			evidencias[indice],
			instante,
		)
		if err != nil {
			return cobertura.PreparacionConjuntosViasCobertura{},
				ErrPreparacionGlobalCoberturaNoConfiable
		}
		conjuntos[indice] = conjunto
	}
	preparacion, err := cobertura.PrepararConjuntosViasCobertura(
		cobertura.DatosPrepararConjuntosViasCobertura{
			AnalisisRef:          datos.analisisRef,
			AnalisisHuellaSHA256: datos.analisisHuella,
			Catalogo:             datos.catalogo, Politica: datos.politica,
			Conjuntos: conjuntos, PreparadaEn: instante,
		},
	)
	if err != nil {
		return cobertura.PreparacionConjuntosViasCobertura{},
			ErrPreparacionGlobalCoberturaNoConfiable
	}
	if err := ctx.Err(); err != nil {
		return cobertura.PreparacionConjuntosViasCobertura{},
			errorPreparacionGlobalContexto(err)
	}
	return preparacion, nil
}

func validarCargaPreparacionGlobal(
	vias []domain.DefinicionViaCobertura,
) (int, error) {
	if len(vias) == 0 {
		return 0, ErrDatosPreparacionGlobalCoberturaInvalidos
	}
	total := 0
	for _, via := range vias {
		if via.Validar() != nil || len(via.Comprobaciones) == 0 ||
			len(via.Comprobaciones) >
				maximoEvidenciasPorViaPreparacionGlobal {
			return 0, ErrDatosPreparacionGlobalCoberturaInvalidos
		}
		total += len(via.Comprobaciones)
	}
	return total, nil
}

func huellaSHA256PreparacionGlobalValida(valor string) bool {
	if len(valor) != 64 {
		return false
	}
	noCero := false
	for _, caracter := range valor {
		if (caracter < '0' || caracter > '9') &&
			(caracter < 'a' || caracter > 'f') {
			return false
		}
		noCero = noCero || caracter != '0'
	}
	return noCero
}

func errorPreparacionGlobalContexto(err error) error {
	if err == nil {
		return ErrPreparacionGlobalCoberturaNoConfiable
	}
	return errors.Join(ErrPreparacionGlobalCoberturaNoConfiable, err)
}

func dependenciaPreparadorGlobalNula(valor any) bool {
	if valor == nil {
		return true
	}
	reflejo := reflect.ValueOf(valor)
	switch reflejo.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice:
		return reflejo.IsNil()
	default:
		return false
	}
}

var _ json.Marshaler = datosPreparacionGlobalCobertura{}
