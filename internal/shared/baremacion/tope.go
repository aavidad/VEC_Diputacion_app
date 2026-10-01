package baremacion

// DesgloseTope conserva los valores exactos antes y despues de un maximo.
// No determina a que regla, seccion o proceso pertenece el limite.
type DesgloseTope struct {
	bruto     Puntos
	tope      Puntos
	resultado Puntos
	exceso    Puntos
}

func (d DesgloseTope) Bruto() Puntos     { return d.bruto }
func (d DesgloseTope) Tope() Puntos      { return d.tope }
func (d DesgloseTope) Resultado() Puntos { return d.resultado }
func (d DesgloseTope) Exceso() Puntos    { return d.exceso }
func (d DesgloseTope) Aplicado() bool    { return d.exceso.Micropuntos() > 0 }

// AplicarTope limita una puntuacion ya calculada y conserva el exceso.
// Cero es un limite explicito valido. No suma ni redondea y no recupera una
// operacion previa que haya fallado por desbordamiento.
func AplicarTope(bruto, tope Puntos) (DesgloseTope, error) {
	comparacion, err := bruto.Comparar(tope)
	if err != nil {
		return DesgloseTope{}, remapearTipoError("tope", err)
	}
	resultado := bruto
	if comparacion > 0 {
		resultado = tope
	}
	exceso, err := bruto.Restar(resultado)
	if err != nil {
		return DesgloseTope{}, remapearTipoError("tope", err)
	}
	return DesgloseTope{bruto: bruto, tope: tope, resultado: resultado, exceso: exceso}, nil
}
