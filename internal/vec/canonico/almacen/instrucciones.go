package almacen

import "time"

// InstruccionesCargaDirecta custodia una concesion temporal y su vinculo sin
// exponer referencias mutables ni credenciales a serializadores genericos.
type InstruccionesCargaDirecta struct {
	datos                  DatosInstruccionesCargaDirecta
	vinculoSolicitudSHA256 string
}

func NuevasInstruccionesCargaDirecta(
	datos DatosInstruccionesCargaDirecta,
) (InstruccionesCargaDirecta, error) {
	datos.Cabeceras = append([]CabeceraCargaDirecta(nil), datos.Cabeceras...)
	instrucciones := InstruccionesCargaDirecta{datos: datos}
	if err := instrucciones.Validar(); err != nil {
		return InstruccionesCargaDirecta{}, err
	}
	return instrucciones, nil
}

// VincularSolicitud devuelve una nueva concesion ligada a la huella exacta
// de la preparacion; nunca modifica el valor del que parte.
func (i InstruccionesCargaDirecta) VincularSolicitud(huellaSHA256 string) (InstruccionesCargaDirecta, error) {
	if err := i.Validar(); err != nil || !SHA256HexadecimalValido(huellaSHA256) {
		return InstruccionesCargaDirecta{}, ErrInstruccionesCargaDirectaNoValidas
	}
	i.vinculoSolicitudSHA256 = huellaSHA256
	return i, nil
}

func (i InstruccionesCargaDirecta) Validar() error {
	if !InstruccionesCargaDirectaValidas(i.datos) {
		return ErrInstruccionesCargaDirectaNoValidas
	}
	return nil
}

func (i InstruccionesCargaDirecta) VigenteEn(instante time.Time) bool {
	instante = instante.UTC()
	return i.Validar() == nil && !instante.Before(i.datos.EmitidaEn) && instante.Before(i.datos.ExpiraEn)
}

func (i InstruccionesCargaDirecta) ValidarContra(capacidades Capacidades) error {
	if err := i.Validar(); err != nil || !capacidades.CargaDirectaTemporal ||
		capacidades.ConectorID != i.datos.ConectorID ||
		capacidades.TamanoMaximoObjeto < i.datos.TamanoMaximo ||
		!OrigenesCargaDirectaValidos(capacidades.OrigenesCargaDirecta) {
		return ErrInstruccionesCargaDirectaNoValidas
	}
	origen := OrigenDestinoCargaDirecta(i.datos.Destino)
	for _, permitido := range capacidades.OrigenesCargaDirecta {
		if origen == permitido && OrigenesCargaDirectaValidos([]string{permitido}) {
			return nil
		}
	}
	return ErrInstruccionesCargaDirectaNoValidas
}

func (i InstruccionesCargaDirecta) ValidarPara(
	tamano int64,
	expiraEn time.Time,
	huellaSolicitudSHA256 string,
	capacidades Capacidades,
) error {
	if i.ValidarContra(capacidades) != nil || i.datos.TamanoMaximo != tamano ||
		!i.datos.ExpiraEn.Equal(expiraEn) || i.vinculoSolicitudSHA256 != huellaSolicitudSHA256 {
		return ErrInstruccionesCargaDirectaNoValidas
	}
	return nil
}

// DatosVerificados devuelve una copia defensiva y el vinculo inmutable solo
// despues de validar la forma completa de la concesion.
func (i InstruccionesCargaDirecta) DatosVerificados() (
	DatosInstruccionesCargaDirecta,
	string,
	error,
) {
	if err := i.Validar(); err != nil {
		return DatosInstruccionesCargaDirecta{}, "", err
	}
	datos := i.datos
	datos.Cabeceras = append([]CabeceraCargaDirecta(nil), i.datos.Cabeceras...)
	return datos, i.vinculoSolicitudSHA256, nil
}
