package bootstrap

import (
	"context"
	"crypto/subtle"
	"errors"
	"log"

	"vec-diputacion-granada/internal/modules/bolsa/application/constitucion"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

// errNombrePropioMiBolsaNoDisponible no lleva causa: el detalle técnico va
// al registro por etapa y nunca incluye el candidato ni el nombre.
var errNombrePropioMiBolsaNoDisponible = errors.New("bolsa mi bolsa: nombre propio no disponible")

var _ puertosbolsa.FuenteNombrePropioMiBolsa = (*fuentePersonalizacionB7)(nil)

// NombrePropio reutiliza el enlace diferido con las bolsas constituidas que
// ya usa el correo B7: esa fuente es la única de este proceso que guarda la
// clave del staging protegido de CONVOCA. Sin enlace, la cabecera de Mi
// Bolsa queda sin nombre.
func (f *fuentePersonalizacionB7) NombrePropio(ctx context.Context, candidatoRef string, bolsas []string) (string, string, bool, error) {
	if f == nil || ctx == nil {
		return "", "", false, errNombrePropioMiBolsaNoDisponible
	}
	f.mu.RLock()
	fuente := f.fuente
	f.mu.RUnlock()
	if fuente == nil {
		return "", "", false, errNombrePropioMiBolsaNoDisponible
	}
	return fuente.nombrePropio(ctx, candidatoRef, bolsas)
}

// nombrePropio busca, en el acta de cada bolsa vigente de la consulta
// propia, la fila cuya referencia `can_*` recalculada coincide con la del
// candidato autenticado. Las filas ambiguas (dos personas con la misma
// referencia) no se atribuyen a nadie, igual que al vincularlas. Solo mira
// bolsas vigentes: con participaciones solo en bolsas extinguidas no hay
// nombre. Descifra el acta entera en cada lectura, como la lista de RRHH.
func (f *fuenteConstituidaRRHHDesarrollo) nombrePropio(ctx context.Context, candidatoRef string, bolsas []string) (string, string, bool, error) {
	if f == nil || f.repositorio == nil || f.recuperador == nil || f.derivador == nil || candidatoRef == "" || len(bolsas) == 0 {
		return "", "", false, errNombrePropioMiBolsaNoDisponible
	}
	buscadas := make(map[string]bool, len(bolsas))
	for _, bolsa := range bolsas {
		buscadas[bolsa] = true
	}
	vigentes, err := f.repositorio.ListarVigentes(ctx)
	if err != nil {
		return "", "", false, falloNombrePropioMiBolsa("vigentes", err)
	}
	// Un acta ilegible no impide buscar en las demás bolsas de la titular.
	var fallo error
	for _, vigente := range vigentes {
		if !buscadas[vigente.Bolsa.BolsaRef] {
			continue
		}
		lote, _, existe, err := f.recuperador.RecuperarLote(ctx, vigente.Bolsa.HuellaListadoSHA256, vigente.CategoriaRef)
		if err != nil || !existe {
			fallo = falloNombrePropioMiBolsa("recuperar_acta", err)
			continue
		}
		filas, _, err := constitucion.DerivarFilasVinculo(lote, f.derivador)
		if err != nil {
			fallo = falloNombrePropioMiBolsa("derivar_referencias", err)
			continue
		}
		numero := 0
		for _, fila := range filas {
			if subtle.ConstantTimeCompare([]byte(fila.CandidatoRef), []byte(candidatoRef)) == 1 {
				numero = fila.FilaNumero
				break
			}
		}
		if numero == 0 {
			continue
		}
		for _, aceptada := range lote.Aceptadas {
			if aceptada.Numero == numero {
				identidad := aceptada.Identidad
				return identidad.Nombre, identidad.PrimerApellido + " " + identidad.SegundoApellido, true, nil
			}
		}
	}
	return "", "", false, fallo
}

// falloNombrePropioMiBolsa deja la etapa y una causa cerrada; nunca imprime
// el error, que puede envolver mensajes de PostgreSQL.
func falloNombrePropioMiBolsa(etapa string, err error) error {
	log.Printf("bolsa mi bolsa: nombre propio no disponible; etapa=%s causa=%s", etapa, causaFalloPostgreSQLCTDesarrollo(err))
	return errNombrePropioMiBolsaNoDisponible
}
