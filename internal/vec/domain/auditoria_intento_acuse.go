package domain

import "time"

// AcuseIntentoAuditoria procede exclusivamente de la transacción confirmada.
// Referencia, secuencia y huella ubican la entrada en la cadena común.
type AcuseIntentoAuditoria struct {
	AuditoriaRef   string
	Secuencia      int64
	HuellaSHA256   string
	CorrelacionRef string
	RegistradaEn   time.Time
}

func (a AcuseIntentoAuditoria) ValidarPara(o OrdenIntentoAuditoria) error {
	datos, err := o.Datos()
	if err != nil || !referenciaAcuseIntentoAuditoriaValida(a.AuditoriaRef) ||
		a.Secuencia < 1 || a.Secuencia > 9007199254740991 ||
		!huellaAcuseIntentoAuditoriaValida(a.HuellaSHA256) ||
		a.CorrelacionRef != datos.Datos.CorrelacionRef ||
		a.RegistradaEn.IsZero() || a.RegistradaEn.Location() != time.UTC {
		return ErrAcuseIntentoAuditoriaInvalido
	}
	return nil
}

func referenciaAcuseIntentoAuditoriaValida(valor string) bool {
	if valor == "" || len(valor) > 128 {
		return false
	}
	for _, r := range valor {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_' && r != '-' {
			return false
		}
	}
	return true
}

func huellaAcuseIntentoAuditoriaValida(valor string) bool {
	if len(valor) != 64 {
		return false
	}
	for _, r := range valor {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}
