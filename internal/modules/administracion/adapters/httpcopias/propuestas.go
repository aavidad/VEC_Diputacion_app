package httpcopias

import p "vec-diputacion-granada/internal/modules/administracion/ports/httpcopias"

func propuestaValida(v p.Propuesta) bool {
	return referencia(v.PropuestaRef) && referencia(v.ConjuntoRef) && referencia(v.DestinoRef) &&
		referencia(v.PoliticaRef) && referencia(v.MotivoRef) && referencia(v.VentanaRef) &&
		v.Version > 0 && v.Estado != "" && huella(v.HuellaSHA256) && huella(v.PreimagenSHA256) &&
		huella(v.PoliticaHuellaSHA256) && huella(v.ConjuntoHuellaSHA256) && utc(v.CaducaEn) && utc(v.VentanaInicio) && utc(v.VentanaFin) &&
		v.VentanaInicio.Before(v.VentanaFin) && v.CopiaPreviaRequerida
}
