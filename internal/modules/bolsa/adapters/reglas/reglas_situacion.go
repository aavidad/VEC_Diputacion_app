// Package reglas traduce el resolutor común de reglas tipadas a lo que Bolsa
// necesita para cambiar la situación de una participación: destinos
// admitidos, causas de baja y propuesta de reposición (art. 9 del Reglamento).
// No decide nada por su cuenta: sin catálogo responde «no configurado» y la
// aplicación conserva su conducta compilada.
package reglas

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	vecreglas "vec-diputacion-granada/internal/vec/reglas"
)

// ModalidadGeneral es la modalidad por defecto de la propuesta: usa la
// regla general de reposición.
const ModalidadGeneral = "general"

const maximoCodigo = 64

var _ puertosbolsa.ConsultaReglasSituacion = (*ReglasSituacion)(nil)

// ReglasSituacion es válido con resolutor nulo: todo responde «no configurado».
type ReglasSituacion struct {
	resolutor *vecreglas.Resolutor
}

func NuevasReglasSituacion(resolutor *vecreglas.Resolutor) *ReglasSituacion {
	return &ReglasSituacion{resolutor: resolutor}
}

func (r *ReglasSituacion) Configurada() bool { return r != nil && r.resolutor.Disponible() }

// DestinosSituacion devuelve los destinos que el catálogo admite desde
// origen. Una entrada ausente no restringe nada.
func (r *ReglasSituacion) DestinosSituacion(ctx context.Context, origen string) ([]string, bool, error) {
	if !r.Configurada() {
		return nil, false, nil
	}
	regla, err := r.resolutor.Regla(ctx, vecreglas.BolsaPrefijoTransicionesSituacion+origen)
	if errors.Is(err, vecreglas.ErrReglaNoEncontrada) {
		return nil, false, nil
	}
	if err != nil || regla.Unidad != vecreglas.UnidadLista {
		return nil, false, errorReglas(ctx, err)
	}
	destinos := regla.Elementos()
	catalogo := dominiobolsa.SituacionesParticipacion()
	for _, destino := range destinos {
		if !slices.Contains(catalogo, destino) {
			return nil, false, puertosbolsa.ErrReglasSituacionNoDisponibles
		}
	}
	return destinos, true, nil
}

// PoliticaTransiciones compone la política que debe publicarse en la base:
// para cada origen, la lista del catálogo si la tiene o, si no, la tabla
// compilada. hay=false si el catálogo no trae ninguna entrada de
// transiciones; entonces no se publica nada y rige lo que la base ya tenga.
func (r *ReglasSituacion) PoliticaTransiciones(ctx context.Context) (puertosbolsa.PublicacionPoliticaTransicionesSituacion, bool, error) {
	todas, err := r.reglas(ctx)
	if errors.Is(err, puertosbolsa.ErrReglasSituacionNoConfiguradas) {
		return puertosbolsa.PublicacionPoliticaTransicionesSituacion{}, false, nil
	}
	if err != nil {
		return puertosbolsa.PublicacionPoliticaTransicionesSituacion{}, false, err
	}
	tabla := make(map[string][]string, len(dominiobolsa.SituacionesParticipacion()))
	for _, origen := range dominiobolsa.SituacionesParticipacion() {
		tabla[origen] = dominiobolsa.DestinosSituacionParticipacion(origen)
	}
	var publicacion puertosbolsa.PublicacionPoliticaTransicionesSituacion
	hay := false
	for _, regla := range todas {
		origen, ok := strings.CutPrefix(regla.Clave, vecreglas.BolsaPrefijoTransicionesSituacion)
		if !ok {
			continue
		}
		destinos, configurada, err := r.DestinosSituacion(ctx, origen)
		if err != nil || !configurada || !slices.Contains(dominiobolsa.SituacionesParticipacion(), origen) {
			return puertosbolsa.PublicacionPoliticaTransicionesSituacion{}, false, puertosbolsa.ErrReglasSituacionNoDisponibles
		}
		tabla[origen] = destinos
		if !hay {
			// Todas las entradas vienen de la misma versión del catálogo.
			entrada := regla.ReferenciaEntrada
			entrada.EntradaClave = strings.TrimSuffix(vecreglas.BolsaPrefijoTransicionesSituacion, ".")
			publicacion.CatalogoRef = entrada.Referencia()
			publicacion.CatalogoSHA256 = regla.HuellaCatalogo
			hay = true
		}
	}
	if !hay {
		return puertosbolsa.PublicacionPoliticaTransicionesSituacion{}, false, nil
	}
	if publicacion.Politica, err = dominiobolsa.NuevaPoliticaTransicionesSituacion(tabla); err != nil {
		return puertosbolsa.PublicacionPoliticaTransicionesSituacion{}, false, errors.Join(puertosbolsa.ErrReglasSituacionNoDisponibles, err)
	}
	return publicacion, true, nil
}

// CausasBaja lista, en el orden del catálogo, las entradas de causa de baja.
func (r *ReglasSituacion) CausasBaja(ctx context.Context) ([]puertosbolsa.CausaBajaSituacion, error) {
	todas, err := r.reglas(ctx)
	if err != nil {
		return nil, err
	}
	causas := make([]puertosbolsa.CausaBajaSituacion, 0, 8)
	for _, regla := range todas {
		codigo, ok := strings.CutPrefix(regla.Clave, vecreglas.BolsaPrefijoCausaBaja)
		if !ok {
			continue
		}
		if codigo == "" || len(codigo) > maximoCodigo {
			return nil, puertosbolsa.ErrReglasSituacionNoDisponibles
		}
		causas = append(causas, puertosbolsa.CausaBajaSituacion{Codigo: codigo, Etiqueta: regla.Etiqueta, Procedencia: procedencia(regla)})
	}
	return causas, nil
}

// ModalidadesReposicion lista las modalidades con periodo propio.
func (r *ReglasSituacion) ModalidadesReposicion(ctx context.Context) ([]puertosbolsa.ModalidadReposicion, error) {
	todas, err := r.reglas(ctx)
	if err != nil {
		return nil, err
	}
	var modalidades []puertosbolsa.ModalidadReposicion
	for _, regla := range todas {
		for _, modalidad := range modalidadesDe(regla) {
			if modalidad == "" || modalidad == ModalidadGeneral || len(modalidad) > maximoCodigo {
				return nil, puertosbolsa.ErrReglasSituacionNoDisponibles
			}
			modalidades = append(modalidades, puertosbolsa.ModalidadReposicion{Codigo: modalidad, Meses: regla.Cantidad})
		}
	}
	return modalidades, nil
}

// ProponerReposicion calcula, de fecha a fecha según el cómputo de la regla,
// cuándo vuelve a estar disponible quien terminó su relación en finRelacion.
// La modalidad elige la regla que la declara; si ninguna la declara se usa
// la general. La fecha propuesta es el primer instante tras el periodo.
func (r *ReglasSituacion) ProponerReposicion(ctx context.Context, finRelacion time.Time, modalidad string) (puertosbolsa.PropuestaReposicion, error) {
	if finRelacion.IsZero() || len(modalidad) > maximoCodigo {
		return puertosbolsa.PropuestaReposicion{}, puertosbolsa.ErrReposicionNoCalculable
	}
	todas, err := r.reglas(ctx)
	if err != nil {
		return puertosbolsa.PropuestaReposicion{}, err
	}
	clave := vecreglas.BolsaReposicionGeneral
	for _, regla := range todas {
		if modalidad != "" && modalidad != ModalidadGeneral && slices.Contains(modalidadesDe(regla), modalidad) {
			clave = regla.Clave
			break
		}
	}
	regla, vencimiento, err := r.resolutor.Vencimiento(ctx, clave, finRelacion, "")
	if err != nil {
		return puertosbolsa.PropuestaReposicion{}, errorReglas(ctx, err)
	}
	return puertosbolsa.PropuestaReposicion{
		FechaDisponible: vencimiento.VenceAntesDe.UTC(), UltimoDiaNoDisponible: vencimiento.UltimoDia,
		Meses: regla.Cantidad, Procedencia: procedencia(regla),
	}, nil
}

func (r *ReglasSituacion) reglas(ctx context.Context) ([]vecreglas.Regla, error) {
	if !r.Configurada() {
		return nil, puertosbolsa.ErrReglasSituacionNoConfiguradas
	}
	todas, err := r.resolutor.Reglas(ctx)
	if err != nil {
		return nil, errorReglas(ctx, err)
	}
	return todas, nil
}

// modalidadesDe solo reconoce el atributo en reglas de reposición, para que
// otra regla con el mismo atributo no cambie la propuesta por accidente.
func modalidadesDe(regla vecreglas.Regla) []string {
	if !strings.HasPrefix(regla.Clave, "b14.reposicion") || regla.Unidad != vecreglas.UnidadMeses {
		return nil
	}
	valor := regla.Atributos[vecreglas.AtributoModalidades]
	if valor == "" {
		return nil
	}
	return strings.Split(valor, ",")
}

func procedencia(regla vecreglas.Regla) puertosbolsa.ProcedenciaRegla {
	return puertosbolsa.ProcedenciaRegla{
		Clave: regla.Clave, Referencia: regla.Referencia, Articulo: regla.Articulo,
		Norma: regla.Norma, Ejemplo: regla.EsEjemplo(),
	}
}

func errorReglas(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, vecreglas.ErrReglasNoConfiguradas) {
		return puertosbolsa.ErrReglasSituacionNoConfiguradas
	}
	return puertosbolsa.ErrReglasSituacionNoDisponibles
}
