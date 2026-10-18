// Package documentosbase conserva el montaje común del almacén documental.
package documentosbase

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	almacenvec "vec-diputacion-granada/internal/vec/adapters/almacen"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrNoDisponible = errors.New("bootstrap: Documentos no disponible")

type ConfiguracionAlmacen struct {
	Tipo         string `json:"tipo"`
	Directorio   string `json:"directorio"`
	TamanoMaximo int64  `json:"tamano_maximo"`
	// RetencionMinimaDias es obligatoria para "ficheros": nil (clave ausente)
	// impide arrancar; 0 solo vale declarado y con catálogo provisional.
	RetencionMinimaDias *int64            `json:"retencion_minima_dias"`
	S3                  map[string]string `json:"s3"`
}

func AbrirAlmacen(ctx context.Context, a ConfiguracionAlmacen, reloj vecports.Reloj) (vecports.AlmacenObjetos, func() error, error) {
	registro := almacenvec.NuevoRegistroConectoresAlmacen()
	requisitos := vecports.RequisitosAlmacenObjetos{EscrituraEnFlujo: true, LecturaEnFlujo: true, ReferenciasOpacas: true,
		IntegridadSHA256: true, Retencion: true}
	var (
		identificador string
		valores       almacenvec.ConfiguracionConectorAlmacen
	)
	switch a.Tipo {
	case "", "ficheros":
		// retencion_minima_dias es explícita: omitirla no equivale a 0. Con 0
		// el conector no fija retención al escribir, como exige un catálogo de
		// conservación provisional.
		if len(a.S3) != 0 || a.TamanoMaximo < 1 || a.RetencionMinimaDias == nil || *a.RetencionMinimaDias < 0 {
			return nil, nil, ErrNoDisponible
		}
		identificador = "ficheros-local"
		if almacenvec.RegistrarFicheros(registro, identificador, reloj) != nil {
			return nil, nil, ErrNoDisponible
		}
		valores = almacenvec.ConfiguracionConectorAlmacen{"directorio": a.Directorio,
			"tamano_maximo": strconv.FormatInt(a.TamanoMaximo, 10), "retencion_minima_dias": strconv.FormatInt(*a.RetencionMinimaDias, 10)}
	case "s3":
		if a.Directorio != "" || a.RetencionMinimaDias != nil || len(a.S3) == 0 {
			return nil, nil, ErrNoDisponible
		}
		identificador = "s3-documentos"
		if almacenvec.RegistrarS3Compatible(registro, identificador) != nil {
			return nil, nil, ErrNoDisponible
		}
		valores = almacenvec.ConfiguracionConectorAlmacen(a.S3)
	default:
		return nil, nil, ErrNoDisponible
	}
	almacen, err := registro.Crear(ctx, identificador, valores, requisitos)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: almacén de originales: %w", ErrNoDisponible, err)
	}
	cerrar := func() error { return nil }
	if cerrable, ok := almacen.(interface{ Cerrar() error }); ok {
		cerrar = cerrable.Cerrar
	}
	return almacen, cerrar, nil
}

// conectorRetencionAlEscribir lo implementa el conector que declara si fija
// retención irreversible al escribir. Uno que no lo declara se presume que sí.
type conectorRetencionAlEscribir interface{ RetencionAlEscribir() bool }

// admitirCatalogoConservacion aplica la decisión de dirección sobre el
// catálogo provisional: sus plazos no son una política aprobada, así que solo
// se admite en el perfil de desarrollo con doble llave y con un conector que
// no fije retención al escribir. S3 (Object Lock) la fija siempre: falla
// cerrado sin llegar a crearlo. Con almacen nil solo se comprueba el perfil y
// el tipo; con el conector ya creado, su declaración. Con catálogo aprobado es
// al revés: un conector que declara no fijar retención al escribir
// (`retencion_minima_dias: 0`) se rechaza al arrancar, no en la primera
// escritura.
func AdmitirCatalogoConservacion(desarrollo bool, provisional bool, tipo string, almacen vecports.AlmacenObjetos) error {
	if !provisional {
		if c, ok := almacen.(conectorRetencionAlEscribir); ok && !c.RetencionAlEscribir() {
			return fmt.Errorf("%w: el catálogo de conservación está aprobado y el conector no fija retención al escribir", ErrNoDisponible)
		}
		return nil
	}
	if !desarrollo {
		return fmt.Errorf("%w: catálogo de conservación provisional fuera del perfil de desarrollo", ErrNoDisponible)
	}
	if tipo == "s3" {
		return fmt.Errorf("%w: el conector S3 fija retención al escribir y el catálogo de conservación es provisional", ErrNoDisponible)
	}
	if almacen == nil {
		return nil
	}
	if c, ok := almacen.(conectorRetencionAlEscribir); !ok || c.RetencionAlEscribir() {
		return fmt.Errorf("%w: el conector fija retención al escribir y el catálogo de conservación es provisional", ErrNoDisponible)
	}
	return nil
}
