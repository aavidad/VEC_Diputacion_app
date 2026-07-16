package ports_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	ports "vec-diputacion-granada/internal/modules/bolsa/ports"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type consumidorClaveExterno struct{ material []byte }

func (c *consumidorClaveExterno) ConsumirClaveClienteLoteIdempotenciaBaremacion(
	ctx context.Context,
	material []byte,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.material = append([]byte(nil), material...)
	return nil
}

type consumidorIdentidadExterno struct{ material []byte }

func (c *consumidorIdentidadExterno) ConsumirIdentidadInternaEstableIdempotenciaBaremacion(
	ctx context.Context,
	material []byte,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.material = append([]byte(nil), material...)
	return nil
}

var (
	claveSeudonimoExterna            = []byte("clave-seudonimo-externa-ficticia-v1")
	clavePrincipalExterna            = []byte("clave-principal-externa-ficticia-v1")
	claveIndiceExterna               = []byte("clave-indice-externa-ficticia-v1")
	claveAtestacionResolucionExterna = []byte("clave-atestacion-resolucion-externa-v1")
)

const (
	snapshotIdentidadExternoRef        = "snapshot-identidad-externo-v1"
	snapshotIdentidadExternoRevision   = uint64(23)
	formatoAtestacionIdentidadExterno  = "hmac-sha256-v1"
	emisorAtestacionIdentidadExterno   = "servicio-identidad-externo"
	claveAtestacionIdentidadExternoRef = "atestacion-identidad-externa-v1"
)

func identidadInternaExterna(sujeto string) []byte {
	huella := sha256.Sum256([]byte(sujeto))
	identidad := make([]byte, 32)
	for posicion := range identidad {
		identidad[posicion] = 0x80 + byte((posicion+int(huella[0]))%64)
	}
	return identidad
}

type fronteraIdentidadExterna struct {
	identidad []byte
	llamadas  int
}

func (f *fronteraIdentidadExterna) ResolverYEntregarIdentidadInternaEstableIdempotenciaBaremacion(
	ctx context.Context,
	ambito ports.SolicitudResolverSeudonimoSujetoBaremacion,
	seudonimo ports.SeudonimoSujetoBaremacionHMAC,
	receptor ports.ReceptorEfimeroResolucionIdentidadInternaEstableBaremacion,
) error {
	f.llamadas++
	huella, err := ports.CalcularHuellaSnapshotResolucionIdentidadInternaEstableBaremacion(
		ambito, seudonimo, snapshotIdentidadExternoRef,
		snapshotIdentidadExternoRevision, f.identidad,
	)
	if err != nil {
		return err
	}
	preimagen, err := copiarCargaDuranteVisitaExterna(func(visita func(ports.MaterialCanonicoEfimeroBaremacion) error) error {
		return ports.VisitarMaterialCanonicoAtestacionResolucionIdentidadInternaEstableBaremacion(
			ambito, seudonimo, snapshotIdentidadExternoRef,
			snapshotIdentidadExternoRevision, huella, visita,
		)
	})
	if err != nil {
		return err
	}
	defer borrarExterno(preimagen)
	mac := hmac.New(sha256.New, claveAtestacionResolucionExterna)
	_, _ = mac.Write(preimagen)
	return receptor.RegistrarResolucionIdentidadInternaEstableBaremacion(
		ctx, f.identidad, snapshotIdentidadExternoRef,
		snapshotIdentidadExternoRevision, huella, formatoAtestacionIdentidadExterno,
		emisorAtestacionIdentidadExterno, claveAtestacionIdentidadExternoRef, mac.Sum(nil),
	)
}

type productorAtomicoExterno struct {
	claveAtestacion []byte
	llamadas        int
}

func (p *productorAtomicoExterno) ProducirTestimonioAtomicoIdempotenciaBaremacion(
	ctx context.Context,
	solicitud ports.SolicitudTestimonioAtomicoIdempotenciaBaremacion,
	fuenteIdentidad ports.FuenteEfimeraIdentidadInternaEstableIdempotenciaBaremacion,
	fuente ports.FuenteEfimeraClaveClienteIdempotenciaBaremacion,
	receptor ports.ReceptorEfimeroTestimonioAtomicoIdempotenciaBaremacion,
) error {
	p.llamadas++
	consumidor := &consumidorClaveExterno{}
	if err := fuente.EntregarClaveClienteLoteIdempotenciaBaremacion(ctx, consumidor); err != nil {
		return err
	}
	defer borrarExterno(consumidor.material)
	identidad := &consumidorIdentidadExterno{}
	if err := fuenteIdentidad.EntregarIdentidadInternaEstableIdempotenciaBaremacion(ctx, identidad); err != nil {
		return err
	}
	defer borrarExterno(identidad.material)
	preimagenSeudonimo, err := copiarCargaDuranteVisitaExterna(func(visita func(ports.MaterialCanonicoEfimeroBaremacion) error) error {
		return ports.VisitarMaterialCanonicoParaDerivarSeudonimoSujetoBaremacion(
			solicitud, identidad.material, visita,
		)
	})
	if err != nil {
		return err
	}
	defer borrarExterno(preimagenSeudonimo)
	seudonimoCalculado := hmacHexExterno(claveSeudonimoExterna, preimagenSeudonimo)
	seudonimoSolicitud := ""
	if err := solicitud.VisitarAmbitoSujetoBaremacion(func(
		_, _, _ string, _ uint16, _, seudonimo string,
	) error {
		seudonimoSolicitud = seudonimo
		return nil
	}); err != nil || !hmac.Equal([]byte(seudonimoCalculado), []byte(seudonimoSolicitud)) {
		return errors.New("seudonimo externo no corresponde al ancla")
	}
	huellaPrincipal := huellaSnapshotExterna(
		ports.DominioClavePrincipalBaremacion, "llavero-principal-externo", 11,
		ports.VersionPrincipalEstableBaremacionV1, 7, "principal-externo-v1",
	)
	if err := receptor.InmovilizarLlaveroIdentidadesBaremacion(
		"llavero-principal-externo", 11, 1, huellaPrincipal,
	); err != nil {
		return err
	}
	preimagenPrincipal, err := copiarCargaDuranteVisitaExterna(func(visita func(ports.MaterialCanonicoEfimeroBaremacion) error) error {
		return ports.VisitarMaterialCanonicoParaDerivarPrincipalEstableBaremacion(
			solicitud, identidad.material, visita,
		)
	})
	if err != nil {
		return err
	}
	defer borrarExterno(preimagenPrincipal)
	principal := hmacHexExterno(clavePrincipalExterna, preimagenPrincipal)
	if err := receptor.RegistrarPrincipalEstableBaremacion(
		0, ports.VersionPrincipalEstableBaremacionV1, 7, "principal-externo-v1", principal,
	); err != nil {
		return err
	}
	huellaIndice := huellaSnapshotExterna(
		ports.DominioClaveIndiceBaremacion, "llavero-indice-externo", 13,
		ports.VersionIndiceIdempotenciaBaremacionV1, 9, "indice-externo-v1",
	)
	if err := receptor.InmovilizarLlaveroIndicesBaremacion(
		"llavero-indice-externo", 13, 1, huellaIndice,
	); err != nil {
		return err
	}
	preimagenIndice, err := copiarCargaDuranteVisitaExterna(func(visita func(ports.MaterialCanonicoEfimeroBaremacion) error) error {
		return ports.VisitarMaterialCanonicoParaDerivarIndiceIdempotenciaBaremacion(
			solicitud, ports.VersionPrincipalEstableBaremacionV1, 7,
			"principal-externo-v1", principal, consumidor.material, visita,
		)
	})
	if err != nil {
		return err
	}
	defer borrarExterno(preimagenIndice)
	indice := hmacHexExterno(claveIndiceExterna, preimagenIndice)
	if err := receptor.RegistrarIndiceIdempotenciaBaremacion(
		0, 0, ports.VersionIndiceIdempotenciaBaremacionV1, 9, "indice-externo-v1", indice,
	); err != nil {
		return err
	}
	material, err := copiarCargaDuranteVisitaExterna(
		receptor.VisitarMaterialCanonicoParaAtestacionBaremacion,
	)
	if err != nil {
		return err
	}
	defer borrarExterno(material)
	contenido := sha256.Sum256(material)
	mac := hmac.New(sha256.New, p.claveAtestacion)
	_, _ = mac.Write(contenido[:])
	return receptor.RegistrarEvidenciaAtestacionBaremacion(
		"hmac-sha256-v1", "hsm-externo-ficticio", "atestacion-externa-v1", 17,
		hex.EncodeToString(contenido[:]), mac.Sum(nil),
	)
}

type verificadorIndependienteExterno struct {
	claveAtestacion []byte
	llamadas        int
	ultimaVista     ports.VistaEfimeraProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion
}

// raizIndependienteExterna mantiene un tipo concreto distinto del verificador
// de fabrica. Es una barrera nominal de composicion; no acredita por si sola
// proceso, operador ni clave fisica independientes.
type raizIndependienteExterna struct {
	delegado *verificadorIndependienteExterno
}

func (r *raizIndependienteExterna) VerificarTestimonioAtomicoIdempotenciaBaremacion(
	ctx context.Context,
	solicitud ports.SolicitudTestimonioAtomicoIdempotenciaBaremacion,
	identidad ports.FuenteEfimeraIdentidadInternaEstableIdempotenciaBaremacion,
	clave ports.FuenteEfimeraClaveClienteIdempotenciaBaremacion,
	vista ports.VistaEfimeraProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion,
) error {
	return r.delegado.VerificarTestimonioAtomicoIdempotenciaBaremacion(
		ctx, solicitud, identidad, clave, vista,
	)
}

func (v *verificadorIndependienteExterno) VerificarTestimonioAtomicoIdempotenciaBaremacion(
	ctx context.Context,
	solicitud ports.SolicitudTestimonioAtomicoIdempotenciaBaremacion,
	fuenteIdentidad ports.FuenteEfimeraIdentidadInternaEstableIdempotenciaBaremacion,
	fuenteClave ports.FuenteEfimeraClaveClienteIdempotenciaBaremacion,
	vista ports.VistaEfimeraProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion,
) error {
	v.llamadas++
	v.ultimaVista = vista
	if err := ctx.Err(); err != nil {
		return err
	}
	identidad := &consumidorIdentidadExterno{}
	if err := fuenteIdentidad.EntregarIdentidadInternaEstableIdempotenciaBaremacion(ctx, identidad); err != nil {
		return err
	}
	defer borrarExterno(identidad.material)
	claveCliente := &consumidorClaveExterno{}
	if err := fuenteClave.EntregarClaveClienteLoteIdempotenciaBaremacion(ctx, claveCliente); err != nil {
		return err
	}
	defer borrarExterno(claveCliente.material)
	preimagenSeudonimo, err := copiarCargaDuranteVisitaExterna(func(visita func(ports.MaterialCanonicoEfimeroBaremacion) error) error {
		return ports.VisitarMaterialCanonicoParaDerivarSeudonimoSujetoBaremacion(
			solicitud, identidad.material, visita,
		)
	})
	if err != nil {
		return err
	}
	defer borrarExterno(preimagenSeudonimo)
	seudonimoSolicitud := ""
	if err := solicitud.VisitarAmbitoSujetoBaremacion(func(
		_, _, _ string, _ uint16, _, seudonimo string,
	) error {
		seudonimoSolicitud = seudonimo
		return nil
	}); err != nil || !hmac.Equal(
		[]byte(hmacHexExterno(claveSeudonimoExterna, preimagenSeudonimo)),
		[]byte(seudonimoSolicitud),
	) {
		return errors.New("seudonimo externo invalido")
	}
	if err := vista.VisitarResolucionIdentidadInternaEstableBaremacion(func(
		snapshotRef string, revision uint64, huella, formato, emisor, clave string, atestacion []byte,
	) error {
		if snapshotRef != snapshotIdentidadExternoRef || revision != snapshotIdentidadExternoRevision ||
			formato != formatoAtestacionIdentidadExterno || emisor != emisorAtestacionIdentidadExterno ||
			clave != claveAtestacionIdentidadExternoRef {
			return errors.New("resolucion externa inesperada")
		}
		huellaEsperada, err := ports.CalcularHuellaSnapshotResolucionIdentidadInternaEstableBaremacion(
			solicitudAmbitoExterno(solicitud), solicitudSeudonimoExterno(solicitud),
			snapshotRef, revision, identidad.material,
		)
		if err != nil || !hmac.Equal([]byte(huellaEsperada), []byte(huella)) {
			return errors.New("huella de resolucion externa invalida")
		}
		preimagen, err := copiarCargaDuranteVisitaExterna(func(visita func(ports.MaterialCanonicoEfimeroBaremacion) error) error {
			return ports.VisitarMaterialCanonicoAtestacionResolucionIdentidadInternaEstableBaremacion(
				solicitudAmbitoExterno(solicitud), solicitudSeudonimoExterno(solicitud),
				snapshotRef, revision, huella, visita,
			)
		})
		if err != nil {
			return err
		}
		defer borrarExterno(preimagen)
		mac := hmac.New(sha256.New, claveAtestacionResolucionExterna)
		_, _ = mac.Write(preimagen)
		if !hmac.Equal(mac.Sum(nil), atestacion) {
			return errors.New("atestacion de resolucion externa invalida")
		}
		return nil
	}); err != nil {
		return err
	}
	if err := vista.VisitarMaterialCanonicoAtestadoBaremacion(
		func(material ports.MaterialCanonicoEfimeroBaremacion) error {
			return material.VisitarBytes(func([]byte) error { return nil })
		},
	); err != nil {
		return err
	}
	ref, revision, cantidad, huella, err := vista.ResumenLlaveroIdentidadesBaremacion()
	if err != nil || ref != "llavero-principal-externo" || revision != 11 || cantidad != 1 ||
		huella != huellaSnapshotExterna(
			ports.DominioClavePrincipalBaremacion, ref, revision,
			ports.VersionPrincipalEstableBaremacionV1, 7, "principal-externo-v1",
		) {
		return errors.New("identidades externas incompletas")
	}
	if err := vista.VisitarTopologiaIdentidadesBaremacion(func(
		posicion int, version uint16, generacion uint32, clave string,
	) error {
		if posicion != 0 || version != ports.VersionPrincipalEstableBaremacionV1 ||
			generacion != 7 || clave != "principal-externo-v1" {
			return errors.New("topologia principal externa inesperada")
		}
		return nil
	}); err != nil {
		return err
	}
	ref, revision, cantidad, huella, err = vista.ResumenLlaveroIndicesBaremacion()
	if err != nil || ref != "llavero-indice-externo" || revision != 13 || cantidad != 1 ||
		huella != huellaSnapshotExterna(
			ports.DominioClaveIndiceBaremacion, ref, revision,
			ports.VersionIndiceIdempotenciaBaremacionV1, 9, "indice-externo-v1",
		) {
		return errors.New("indices externos incompletos")
	}
	if err := vista.VisitarTopologiaIndicesBaremacion(func(
		posicion int, version uint16, generacion uint32, clave string,
	) error {
		if posicion != 0 || version != ports.VersionIndiceIdempotenciaBaremacionV1 ||
			generacion != 9 || clave != "indice-externo-v1" {
			return errors.New("topologia indice externa inesperada")
		}
		return nil
	}); err != nil {
		return err
	}
	principales, indices := 0, 0
	principalRecibido := ""
	preimagenPrincipal, err := copiarCargaDuranteVisitaExterna(func(visita func(ports.MaterialCanonicoEfimeroBaremacion) error) error {
		return ports.VisitarMaterialCanonicoParaDerivarPrincipalEstableBaremacion(
			solicitud, identidad.material, visita,
		)
	})
	if err != nil {
		return err
	}
	defer borrarExterno(preimagenPrincipal)
	if err := vista.VisitarPrincipalesBaremacion(func(
		posicion int, version uint16, generacion uint32, clave, valor string,
	) error {
		if posicion != 0 || version != ports.VersionPrincipalEstableBaremacionV1 ||
			generacion != 7 || clave != "principal-externo-v1" ||
			!hmac.Equal([]byte(hmacHexExterno(clavePrincipalExterna, preimagenPrincipal)), []byte(valor)) {
			return errors.New("principal externo inesperado")
		}
		principalRecibido = valor
		principales++
		return nil
	}); err != nil {
		return err
	}
	if err := vista.VisitarMatrizIndicesBaremacion(func(
		fila, columna int, version uint16, generacion uint32, clave, valor string,
	) error {
		if fila != 0 || columna != 0 || version != ports.VersionIndiceIdempotenciaBaremacionV1 ||
			generacion != 9 || clave != "indice-externo-v1" {
			return errors.New("indice externo inesperado")
		}
		preimagen, err := copiarCargaDuranteVisitaExterna(func(visita func(ports.MaterialCanonicoEfimeroBaremacion) error) error {
			return ports.VisitarMaterialCanonicoParaDerivarIndiceIdempotenciaBaremacion(
				solicitud, ports.VersionPrincipalEstableBaremacionV1, 7,
				"principal-externo-v1", principalRecibido, claveCliente.material, visita,
			)
		})
		if err != nil {
			return err
		}
		defer borrarExterno(preimagen)
		if !hmac.Equal([]byte(hmacHexExterno(claveIndiceExterna, preimagen)), []byte(valor)) {
			return errors.New("formula de indice externa invalida")
		}
		indices++
		return nil
	}); err != nil {
		return err
	}
	if principales != 1 || indices != 1 {
		return errors.New("matriz externa no completa")
	}
	return vista.VisitarEvidenciaAtestacionBaremacion(func(
		formato, emisor, clave string, revision uint64, huella string, evidencia []byte,
	) error {
		if formato != "hmac-sha256-v1" || emisor != "hsm-externo-ficticio" ||
			clave != "atestacion-externa-v1" || revision != 17 {
			return errors.New("metadatos de atestacion externos inesperados")
		}
		contenido, err := hex.DecodeString(huella)
		if err != nil {
			return err
		}
		mac := hmac.New(sha256.New, v.claveAtestacion)
		_, _ = mac.Write(contenido)
		if !hmac.Equal(mac.Sum(nil), evidencia) {
			return errors.New("atestacion externa no valida")
		}
		return nil
	})
}

type consumidorNominalNoAutoritativoExterno struct {
	llamadas             int
	principales          int
	indices              int
	representaciones     int
	vista                ports.VistaEfimeraProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion
	intencion            *ports.IntencionCambioBaremacion
	motivoVerificado     []byte
	fingerprintRetenido  ports.MaterialCanonicoEfimeroBaremacion
	sobreRetenido        ports.MaterialCanonicoEfimeroBaremacion
	fingerprintAlias     []byte
	sobreAlias           []byte
	fingerprintCopiado   []byte
	sobreCopiado         []byte
	selloFingerprintHMAC string
	panico               bool
}

func (c *consumidorNominalNoAutoritativoExterno) ConsumirProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion(
	ctx context.Context,
	solicitud ports.SolicitudTestimonioAtomicoIdempotenciaBaremacion,
	vista ports.VistaEfimeraProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion,
) error {
	c.llamadas++
	c.vista = vista
	if c.panico {
		panic("panico ficticio del consumidor externo")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := vista.VisitarMaterialCanonicoAtestadoBaremacion(
		func(material ports.MaterialCanonicoEfimeroBaremacion) error {
			return material.VisitarBytes(func([]byte) error { return nil })
		},
	); err != nil {
		return err
	}
	if err := vista.VisitarResolucionIdentidadInternaEstableBaremacion(
		func(string, uint64, string, string, string, string, []byte) error { return nil },
	); err != nil {
		return err
	}
	if _, _, _, _, err := vista.ResumenLlaveroIdentidadesBaremacion(); err != nil {
		return err
	}
	if err := vista.VisitarTopologiaIdentidadesBaremacion(
		func(int, uint16, uint32, string) error { return nil },
	); err != nil {
		return err
	}
	if _, _, _, _, err := vista.ResumenLlaveroIndicesBaremacion(); err != nil {
		return err
	}
	if err := vista.VisitarTopologiaIndicesBaremacion(
		func(int, uint16, uint32, string) error { return nil },
	); err != nil {
		return err
	}
	if err := vista.VisitarPrincipalesBaremacion(func(int, uint16, uint32, string, string) error {
		c.principales++
		return nil
	}); err != nil {
		return err
	}
	if err := vista.VisitarMatrizIndicesBaremacion(func(int, int, uint16, uint32, string, string) error {
		c.indices++
		return nil
	}); err != nil {
		return err
	}
	if c.intencion != nil {
		if err := vista.VisitarRepresentacionesCanonicasIntencionBaremacion(
			solicitud, *c.intencion, c.motivoVerificado,
			func(
				_, _ int,
				fingerprint, sobre ports.MaterialCanonicoEfimeroBaremacion,
			) error {
				c.representaciones++
				c.fingerprintRetenido = fingerprint
				c.sobreRetenido = sobre
				if err := fingerprint.VisitarBytes(func(valor []byte) error {
					c.fingerprintAlias = valor
					c.fingerprintCopiado = append([]byte(nil), valor...)
					return nil
				}); err != nil {
					return err
				}
				if err := sobre.VisitarBytes(func(valor []byte) error {
					c.sobreAlias = valor
					c.sobreCopiado = append([]byte(nil), valor...)
					return nil
				}); err != nil {
					return err
				}
				c.selloFingerprintHMAC = hmacHexExterno(
					[]byte("clave-intencion-privada-externa-ficticia"), c.fingerprintCopiado,
				)
				return nil
			},
		); err != nil {
			return err
		}
	}
	return vista.VisitarEvidenciaAtestacionBaremacion(
		func(string, string, string, uint64, string, []byte) error { return nil },
	)
}

func TestAdaptadorExternoPuedeRevisarYConsumirSoloProductoNominalCompleto(t *testing.T) {
	sujeto := "sujeto-externo-uno"
	solicitud := solicitudExternaPrueba(
		t, "123e4567-e89b-42d3-a456-426614174000", sujeto,
	)
	claveAtestacion := []byte("clave-atestacion-externa-ficticia-0001")
	productor := &productorAtomicoExterno{claveAtestacion: claveAtestacion}
	verificador := &verificadorIndependienteExterno{claveAtestacion: claveAtestacion}
	raizDelegada := &verificadorIndependienteExterno{claveAtestacion: claveAtestacion}
	raizPrivadaSimulada := &raizIndependienteExterna{delegado: raizDelegada}
	consumidor := &consumidorNominalNoAutoritativoExterno{}
	frontera := &fronteraIdentidadExterna{identidad: identidadInternaExterna(sujeto)}
	if err := ports.ConstruirVerificarYConsumirProductoNominalCompletoIdempotenciaBaremacion(
		context.Background(), solicitud, frontera, productor, verificador,
		raizPrivadaSimulada, consumidor,
	); err != nil {
		t.Fatalf("flujo nominal completo externo: %v", err)
	}
	if consumidor.llamadas != 1 || consumidor.principales != 1 || consumidor.indices != 1 ||
		frontera.llamadas != 1 || productor.llamadas != 1 || verificador.llamadas != 1 ||
		raizDelegada.llamadas != 1 {
		t.Fatalf("lote externo incompleto: frontera=%d productor=%d verificador=%d raiz=%d consumidor=%d principales=%d indices=%d",
			frontera.llamadas, productor.llamadas, verificador.llamadas, raizDelegada.llamadas,
			consumidor.llamadas, consumidor.principales, consumidor.indices)
	}
	if _, _, _, _, err := consumidor.vista.ResumenLlaveroIndicesBaremacion(); err == nil {
		t.Fatal("el consumidor externo retuvo una vista util")
	}
	// Todas las copias comparten propietario: el segundo uso falla antes de
	// resolver identidad y no puede repetir ninguna fase.
	fronteraReuso := &fronteraIdentidadExterna{identidad: identidadInternaExterna(sujeto)}
	productorReuso := &productorAtomicoExterno{claveAtestacion: claveAtestacion}
	verificadorReuso := &verificadorIndependienteExterno{claveAtestacion: claveAtestacion}
	raizReusoDelegada := &verificadorIndependienteExterno{claveAtestacion: claveAtestacion}
	consumidorReuso := &consumidorNominalNoAutoritativoExterno{}
	if err := ports.ConstruirVerificarYConsumirProductoNominalCompletoIdempotenciaBaremacion(
		context.Background(), solicitud, fronteraReuso, productorReuso, verificadorReuso,
		&raizIndependienteExterna{delegado: raizReusoDelegada}, consumidorReuso,
	); !errors.Is(err, ports.ErrClaveIdempotenciaBaremacionInvalida) {
		t.Fatalf("una copia de la solicitud consumida fue reutilizada: %v", err)
	}
	if fronteraReuso.llamadas != 0 || productorReuso.llamadas != 0 ||
		verificadorReuso.llamadas != 0 || raizReusoDelegada.llamadas != 0 ||
		consumidorReuso.llamadas != 0 {
		t.Fatal("el reuso one-shot alcanzo una dependencia")
	}
}

func TestServicioAplicacionExternoPuedeVisitarFingerprintYSobreSinRecibirCelda(t *testing.T) {
	sujeto := "sujeto-externo-representaciones"
	solicitud := solicitudExternaPrueba(
		t, "423e4567-e89b-42d3-a456-426614174000", sujeto,
	)
	claveAtestacion := []byte("clave-atestacion-externa-ficticia-0001")
	intencion := intencionExternaLigadaSolicitud(t, solicitud)
	motivo := []byte("motivo exacto ya verificado por el servicio privado")
	consumidor := &consumidorNominalNoAutoritativoExterno{
		intencion: &intencion, motivoVerificado: motivo,
	}
	if err := ports.ConstruirVerificarYConsumirProductoNominalCompletoIdempotenciaBaremacion(
		context.Background(), solicitud,
		&fronteraIdentidadExterna{identidad: identidadInternaExterna(sujeto)},
		&productorAtomicoExterno{claveAtestacion: claveAtestacion},
		&verificadorIndependienteExterno{claveAtestacion: claveAtestacion},
		&raizIndependienteExterna{
			delegado: &verificadorIndependienteExterno{claveAtestacion: claveAtestacion},
		},
		consumidor,
	); err != nil {
		t.Fatalf("puente nominal externo no utilizable: %v", err)
	}
	if consumidor.representaciones != 1 || consumidor.selloFingerprintHMAC == "" ||
		bytes.Equal(consumidor.fingerprintCopiado, consumidor.sobreCopiado) ||
		!bytes.Contains(consumidor.fingerprintCopiado, []byte("vec.bolsa.fingerprint-semantico-intencion.v1")) ||
		!bytes.Contains(consumidor.sobreCopiado, []byte("vec.bolsa.intencion-cambio-baremacion.v1")) {
		t.Fatalf("representaciones externas incompletas: cantidad=%d sello=%q",
			consumidor.representaciones, consumidor.selloFingerprintHMAC)
	}
	for nombre, material := range map[string]ports.MaterialCanonicoEfimeroBaremacion{
		"fingerprint": consumidor.fingerprintRetenido,
		"sobre":       consumidor.sobreRetenido,
	} {
		if material.Validar() == nil || material.VisitarBytes(func([]byte) error { return nil }) == nil {
			t.Fatalf("%s retenido siguio util tras el callback", nombre)
		}
	}
	for nombre, alias := range map[string][]byte{
		"fingerprint": consumidor.fingerprintAlias, "sobre": consumidor.sobreAlias,
	} {
		if !bytes.Equal(alias, make([]byte, len(alias))) {
			t.Fatalf("%s conservo el alias de bytes tras el callback", nombre)
		}
	}
	defer borrarExterno(consumidor.fingerprintCopiado)
	defer borrarExterno(consumidor.sobreCopiado)

	solicitudCruzada := solicitudExternaPrueba(
		t, "423e4567-e89b-42d3-a456-426614174000", sujeto,
	)
	cruzada := intencionExternaLigadaSolicitud(t, solicitudCruzada)
	cruzada.SolicitudRef = referenciaExterna(9991)
	consumidorCruzado := &consumidorNominalNoAutoritativoExterno{
		intencion: &cruzada, motivoVerificado: motivo,
	}
	if err := ports.ConstruirVerificarYConsumirProductoNominalCompletoIdempotenciaBaremacion(
		context.Background(), solicitudCruzada,
		&fronteraIdentidadExterna{identidad: identidadInternaExterna(sujeto)},
		&productorAtomicoExterno{claveAtestacion: claveAtestacion},
		&verificadorIndependienteExterno{claveAtestacion: claveAtestacion},
		&raizIndependienteExterna{
			delegado: &verificadorIndependienteExterno{claveAtestacion: claveAtestacion},
		},
		consumidorCruzado,
	); !errors.Is(err, ports.ErrClaveIdempotenciaBaremacionInvalida) ||
		consumidorCruzado.representaciones != 0 {
		t.Fatalf("intencion externa cruzada aceptada: representaciones=%d err=%v",
			consumidorCruzado.representaciones, err)
	}
}

func TestVistaExternaSeCierraAunqueElConsumidorEntreEnPanico(t *testing.T) {
	sujeto := "sujeto-externo-panico"
	solicitud := solicitudExternaPrueba(
		t, "323e4567-e89b-42d3-a456-426614174000", sujeto,
	)
	claveAtestacion := []byte("clave-atestacion-externa-ficticia-0001")
	consumidor := &consumidorNominalNoAutoritativoExterno{panico: true}
	func() {
		defer func() {
			if recover() == nil {
				t.Error("el panico ficticio no se propago")
			}
		}()
		_ = ports.ConstruirVerificarYConsumirProductoNominalCompletoIdempotenciaBaremacion(
			context.Background(), solicitud,
			&fronteraIdentidadExterna{identidad: identidadInternaExterna(sujeto)},
			&productorAtomicoExterno{claveAtestacion: claveAtestacion},
			&verificadorIndependienteExterno{claveAtestacion: claveAtestacion},
			&raizIndependienteExterna{
				delegado: &verificadorIndependienteExterno{claveAtestacion: claveAtestacion},
			},
			consumidor,
		)
	}()
	if consumidor.vista == nil {
		t.Fatal("el consumidor no alcanzo a recibir la vista")
	}
	if _, _, _, _, err := consumidor.vista.ResumenLlaveroIdentidadesBaremacion(); err == nil {
		t.Fatal("la vista externa siguio util tras panico")
	}
}

type verificadorSeparacionFisicaExterno struct {
	identidadFisicaPorAlias map[string]string
}

func (v *verificadorSeparacionFisicaExterno) VerificarSeparacionDominiosClaveBaremacion(
	ctx context.Context,
	solicitud ports.SolicitudNominalNoAutoritativaSeparacionDominiosClaveBaremacion,
) error {
	identidades := make(map[string]string)
	return solicitud.VisitarReferencias(func(
		dominio ports.DominioClaveHMACBaremacion, _ uint32, referencia string,
	) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		identidadFisica := v.identidadFisicaPorAlias[referencia]
		if identidadFisica == "" {
			identidadFisica = "fisica|" + referencia
		}
		if dominioAnterior, repetida := identidades[identidadFisica]; repetida {
			return fmt.Errorf("alias fisico compartido por %s y %s", dominioAnterior, dominio)
		}
		identidades[identidadFisica] = string(dominio)
		return nil
	})
}

func TestSolicitudExternaRepresentaSeisDominiosHistoricosYElKMSDetectaAliasFisico(t *testing.T) {
	referencias := make([]ports.ReferenciaGeneracionClaveHMACNominalBaremacion, 0, 20)
	crear := func(dominio ports.DominioClaveHMACBaremacion, generacion uint32, referencia string) {
		entrada, err := ports.NuevaReferenciaGeneracionClaveHMACNominalBaremacion(
			dominio, generacion, referencia,
		)
		if err != nil {
			t.Fatal(err)
		}
		referencias = append(referencias, entrada)
	}
	for generacion := uint32(1); generacion <= 8; generacion++ {
		crear(ports.DominioClavePrincipalBaremacion, generacion, fmt.Sprintf("principal-externo-v%d", generacion))
		crear(ports.DominioClaveIndiceBaremacion, generacion, fmt.Sprintf("indice-externo-v%d", generacion))
	}
	crear(ports.DominioClaveSujetoBaremacion, 1, "sujeto-externo-v1")
	crear(ports.DominioClaveMotivoBaremacion, 1, "motivo-externo-v1")
	crear(ports.DominioClaveManifiestoBaremacion, 1, "manifiesto-externo-v1")
	crear(ports.DominioClaveIntencionBaremacion, 1, "intencion-externo-v1")
	solicitud, err := ports.ConstruirSolicitudNominalNoAutoritativaSeparacionDominiosClaveBaremacion(referencias)
	if err != nil {
		t.Fatal(err)
	}
	visitadas := 0
	if err := solicitud.VisitarReferencias(func(
		ports.DominioClaveHMACBaremacion, uint32, string,
	) error {
		visitadas++
		return nil
	}); err != nil || visitadas != 20 {
		t.Fatalf("lote historico externo incompleto: %d, %v", visitadas, err)
	}
	verificador := &verificadorSeparacionFisicaExterno{identidadFisicaPorAlias: map[string]string{
		"principal-externo-v8": "clave-fisica-compartida",
		"indice-externo-v8":    "clave-fisica-compartida",
	}}
	if err := verificador.VerificarSeparacionDominiosClaveBaremacion(context.Background(), solicitud); err == nil {
		t.Fatal("el KMS externo no detecto dos alias de la misma clave fisica")
	}
}

func expresionASTContieneIdentificadorExterno(expresion ast.Expr, nombre string) bool {
	encontrado := false
	ast.Inspect(expresion, func(nodo ast.Node) bool {
		identificador, ok := nodo.(*ast.Ident)
		if ok && identificador.Name == nombre {
			encontrado = true
			return false
		}
		return !encontrado
	})
	return encontrado
}

func listaASTContieneIdentificadorExterno(lista *ast.FieldList, nombre string) bool {
	if lista == nil {
		return false
	}
	for _, campo := range lista.List {
		if expresionASTContieneIdentificadorExterno(campo.Type, nombre) {
			return true
		}
	}
	return false
}

func expresionASTContieneLlamadaExterna(expresion ast.Expr, nombres map[string]bool) bool {
	encontrada := false
	ast.Inspect(expresion, func(nodo ast.Node) bool {
		llamada, ok := nodo.(*ast.CallExpr)
		if !ok {
			return !encontrada
		}
		switch funcion := llamada.Fun.(type) {
		case *ast.Ident:
			encontrada = nombres[funcion.Name]
		case *ast.SelectorExpr:
			encontrada = nombres[funcion.Sel.Name]
		}
		return !encontrada
	})
	return encontrada
}

func TestASTSuperficiePublicaNoDevuelveCargaNiReintroducePuenteEnDosFases(t *testing.T) {
	ruta := filepath.Join(
		localizarRaizRepositorioIdempotencia(t),
		filepath.FromSlash(rutaContratoNominalPermitida),
	)
	unidad, err := parser.ParseFile(token.NewFileSet(), ruta, nil, 0)
	if err != nil {
		t.Fatalf("analizar superficie publica: %v", err)
	}
	prohibidos := map[string]bool{
		"ProductoNominalNoAutoritativoIdempotenciaBaremacion":                            true,
		"ConstruirProductoNominalNoAutoritativoIdempotenciaBaremacion":                   true,
		"RevisarYConsumirProductoNominalNoAutoritativoCompletoIdempotenciaBaremacion":    true,
		"ConstruirMaterialCanonicoParaDerivarSeudonimoSujetoBaremacion":                  true,
		"ConstruirMaterialCanonicoParaDerivarPrincipalEstableBaremacion":                 true,
		"ConstruirMaterialCanonicoParaDerivarIndiceIdempotenciaBaremacion":               true,
		"ConstruirMaterialCanonicoAtestacionResolucionIdentidadInternaEstableBaremacion": true,
		"MaterialCanonicoAtestadoBaremacion":                                             true,
		"MaterialCanonicoParaAtestacionBaremacion":                                       true,
	}
	for _, declaracion := range unidad.Decls {
		switch concreta := declaracion.(type) {
		case *ast.FuncDecl:
			if prohibidos[concreta.Name.Name] {
				t.Errorf("API publica revocable antigua reintroducida: %s", concreta.Name.Name)
			}
			if concreta.Name.IsExported() &&
				listaASTContieneIdentificadorExterno(concreta.Type.Results, "CargaProtegida") {
				t.Errorf("funcion/metodo exportado devuelve CargaProtegida: %s", concreta.Name.Name)
			}
		case *ast.GenDecl:
			for _, especificacion := range concreta.Specs {
				tipo, ok := especificacion.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if prohibidos[tipo.Name.Name] {
					t.Errorf("tipo publico de producto persistible reintroducido: %s", tipo.Name.Name)
				}
				interfaz, ok := tipo.Type.(*ast.InterfaceType)
				if !ok {
					continue
				}
				for _, campo := range interfaz.Methods.List {
					firma, ok := campo.Type.(*ast.FuncType)
					if !ok || len(campo.Names) == 0 || !campo.Names[0].IsExported() {
						continue
					}
					if listaASTContieneIdentificadorExterno(firma.Results, "CargaProtegida") {
						t.Errorf("metodo de interfaz exportado devuelve CargaProtegida: %s.%s",
							tipo.Name.Name, campo.Names[0].Name)
					}
				}
			}
		}
	}
	llamadasTemporalesSensibles := map[string]bool{
		"representacionCanonica":          true,
		"representacionCanonicaSinHuella": true,
		"decodificarHexCanonicoIntencion": true,
		"Revelar":                         true,
	}
	ast.Inspect(unidad, func(nodo ast.Node) bool {
		llamada, ok := nodo.(*ast.CallExpr)
		if !ok {
			return true
		}
		funcion, ok := llamada.Fun.(*ast.Ident)
		if !ok {
			return true
		}
		if funcion.Name == "anexarCampoCanonicoIntencion" && len(llamada.Args) == 3 &&
			expresionASTContieneLlamadaExterna(llamada.Args[2], llamadasTemporalesSensibles) {
			t.Errorf("temporal sensible usado inline en anexarCampoCanonicoIntencion")
		}
		if funcion.Name == "append" && len(llamada.Args) > 0 {
			if identificador, ok := llamada.Args[0].(*ast.Ident); ok && identificador.Name == "material" {
				t.Errorf("append directo puede abandonar un backing canonico sensible")
			}
		}
		return true
	})
}

func solicitudExternaPrueba(
	t *testing.T,
	claveUUID, sujeto string,
) ports.SolicitudTestimonioAtomicoIdempotenciaBaremacion {
	t.Helper()
	clave, err := ports.NuevaClaveClienteIdempotenciaBaremacion(claveUUID)
	if err != nil {
		t.Fatalf("clave externa: %v", err)
	}
	ambito, err := ports.NuevaSolicitudResolverSeudonimoSujetoBaremacion(
		"f47ac10b-58cc-4372-a567-000000000501",
		"f47ac10b-58cc-4372-a567-000000000502",
		"f47ac10b-58cc-4372-a567-000000000503",
	)
	if err != nil {
		t.Fatalf("ambito externo: %v", err)
	}
	seudonimo := ports.SeudonimoSujetoBaremacionHMAC{
		Version:      ports.VersionSeudonimoSujetoBaremacionV1,
		ClaveHMACRef: "seudonimo-externo-v1", ValorHMAC: huellaExterna("provisional-" + sujeto),
	}
	provisional, err := ports.NuevaSolicitudTestimonioAtomicoIdempotenciaBaremacion(
		"despliegue-externo-ficticio", ports.ModuloIdempotenciaBolsa,
		ports.ClaseCambioIncorporarDecision, ambito, seudonimo, clave,
	)
	if err != nil {
		t.Fatalf("solicitud externa provisional: %v", err)
	}
	preimagen, err := copiarCargaDuranteVisitaExterna(
		func(visita func(ports.MaterialCanonicoEfimeroBaremacion) error) error {
			return ports.VisitarMaterialCanonicoParaDerivarSeudonimoSujetoBaremacion(
				provisional, identidadInternaExterna(sujeto), visita,
			)
		},
	)
	if err != nil {
		t.Fatalf("material seudonimo externo: %v", err)
	}
	defer borrarExterno(preimagen)
	seudonimo.ValorHMAC = hmacHexExterno(claveSeudonimoExterna, preimagen)
	solicitud, err := ports.NuevaSolicitudTestimonioAtomicoIdempotenciaBaremacion(
		"despliegue-externo-ficticio", ports.ModuloIdempotenciaBolsa,
		ports.ClaseCambioIncorporarDecision, ambito, seudonimo, clave,
	)
	if err != nil {
		t.Fatalf("solicitud externa: %v", err)
	}
	return solicitud
}

func solicitudAmbitoExterno(
	solicitud ports.SolicitudTestimonioAtomicoIdempotenciaBaremacion,
) ports.SolicitudResolverSeudonimoSujetoBaremacion {
	var ambito ports.SolicitudResolverSeudonimoSujetoBaremacion
	_ = solicitud.VisitarAmbitoSujetoBaremacion(func(
		proceso, expediente, baremacion string, _ uint16, _, _ string,
	) error {
		ambito, _ = ports.NuevaSolicitudResolverSeudonimoSujetoBaremacion(proceso, expediente, baremacion)
		return nil
	})
	return ambito
}

func solicitudSeudonimoExterno(
	solicitud ports.SolicitudTestimonioAtomicoIdempotenciaBaremacion,
) ports.SeudonimoSujetoBaremacionHMAC {
	var seudonimo ports.SeudonimoSujetoBaremacionHMAC
	_ = solicitud.VisitarAmbitoSujetoBaremacion(func(
		_, _, _ string, version uint16, clave, valor string,
	) error {
		seudonimo = ports.SeudonimoSujetoBaremacionHMAC{
			Version: version, ClaveHMACRef: clave, ValorHMAC: valor,
		}
		return nil
	})
	return seudonimo
}

func intencionExternaLigadaSolicitud(
	t *testing.T,
	solicitud ports.SolicitudTestimonioAtomicoIdempotenciaBaremacion,
) ports.IntencionCambioBaremacion {
	t.Helper()
	procesoRef, solicitudRef, baremacionMeritoRef := "", "", ""
	seudonimo := ports.SeudonimoSujetoBaremacionHMAC{}
	if err := solicitud.VisitarAmbitoSujetoBaremacion(func(
		proceso, expediente, baremacion string,
		version uint16, clave, valor string,
	) error {
		procesoRef, solicitudRef, baremacionMeritoRef = proceso, expediente, baremacion
		seudonimo = ports.SeudonimoSujetoBaremacionHMAC{
			Version: version, ClaveHMACRef: clave, ValorHMAC: valor,
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	intencion := ports.IntencionCambioBaremacion{
		Version: ports.VersionIntencionCambioBaremacionV1,
		Clase:   ports.ClaseCambioIncorporarDecision,

		ProcesoRef:          procesoRef,
		SolicitudRef:        solicitudRef,
		SujetoSeudonimoHMAC: seudonimo,
		BaremacionMeritoRef: baremacionMeritoRef,
		VersionBase: ports.ReferenciaVersionBaremacion{
			BaremacionMeritoRef: baremacionMeritoRef,
			Numero:              1,
			HuellaEstadoSHA256:  huellaExterna("estado-base-externo"),
		},
		VersionObjetivo: 2,

		DecisionRef:                          referenciaExterna(4),
		NumeroDecision:                       1,
		ClaseDecision:                        dominiobolsa.ClaseDecisionInicial,
		ResultadoDecision:                    dominiobolsa.ResultadoAceptado,
		HuellaContenidoDecisionSHA256:        huellaExterna("decision-externa"),
		HuellaEstadoResultanteDecisionSHA256: huellaExterna("estado-resultante-externo"),

		PoliticaFirmaRef:             "politica-firma-externa",
		PoliticaFirmaVersion:         3,
		HuellaPoliticaFirmaSHA256:    huellaExterna("politica-firma-externa"),
		EsquemaPlanFirmaDurable:      ports.EsquemaPlanFirmaDurableBaremacionV2,
		VersionPlanFirmaDurable:      ports.VersionPlanFirmaDurableBaremacionV2,
		PlanFirmaDurableRef:          referenciaExterna(5),
		HuellaPlanFirmaDurableSHA256: huellaExterna("plan-firma-externo"),
		EstadoPlanFirmaDurable:       ports.EstadoPlanFirmaDurableCompletado,

		DocumentoFirmableRef:          referenciaExterna(6),
		VersionDocumentoFirmable:      "v1",
		HuellaDocumentoFirmableSHA256: huellaExterna("documento-firmable-externo"),
		FirmaRef:                      referenciaExterna(7),
		HuellaFirmaSHA256:             huellaExterna("firma-externa"),
		DocumentoFirmadoRef:           referenciaExterna(8),
		HuellaDocumentoFirmadoSHA256:  huellaExterna("documento-firmado-externo"),

		EsquemaManifiestoProbatorio:      ports.EsquemaManifiestoMaterialEstableBaremacionV2,
		VersionManifiestoProbatorio:      ports.VersionManifiestoMaterialEstableBaremacionV2,
		ManifiestoProbatorioRef:          referenciaExterna(9),
		HuellaManifiestoProbatorioSHA256: huellaExterna("manifiesto-externo"),
		SelloManifiestoProbatorioHMAC: ports.HMACManifiestoMaterialBaremacionV2{
			Version:      ports.VersionHMACManifiestoMaterialBaremacionV2,
			ClaveHMACRef: "manifiesto-externo-v2",
			ValorHMAC:    huellaExterna("hmac-manifiesto-externo"),
		},

		ObjetoCustodiadoRef:          referenciaExterna(10),
		VersionObjetoCustodiado:      "version-7",
		ConectorCustodiaID:           "almacen-s3-local-externo",
		ZonaCustodia:                 puertosvec.ZonaAlmacenAdmitida,
		HuellaObjetoCustodiadoSHA256: huellaExterna("documento-firmado-externo"),
		FormatoDocumento: ports.InstantaneaCatalogoFormatoDocumentoBaremacion{
			CatalogoRef:          "catalogo-formatos-firma-externo",
			CatalogoVersion:      2,
			HuellaCatalogoSHA256: huellaExterna("catalogo-formatos-externo"),
			FormatoClave:         "pdf_pades",
			MIMECanonico:         "application/pdf",
		},
		ClasificacionDocumento: ports.InstantaneaCatalogoClasificacionDocumentoBaremacion{
			CatalogoRef:          "catalogo-clasificacion-externo",
			CatalogoVersion:      3,
			HuellaCatalogoSHA256: huellaExterna("catalogo-clasificacion-externo"),
			ClasificacionClave:   "datos_personales_alta",
		},
		TamanoDocumentoFirmado:     4096,
		EstadoInmovilizacionObjeto: ports.EstadoInmovilizacionNoAplicada,
		EstadoDisponibilidadObjeto: ports.EstadoDisponibilidadObjetoActivoNoEliminado,

		EsquemaEvidenciaRecuperacion:      ports.EsquemaReciboRecuperacionBaremacionV2,
		VersionEvidenciaRecuperacion:      ports.VersionReciboRecuperacionBaremacionV2,
		EvidenciaRecuperacionFirmadoRef:   referenciaExterna(11),
		HuellaEvidenciaRecuperacionSHA256: huellaExterna("evidencia-recuperacion-externa"),
		EsquemaEvidenciaCustodia:          ports.EsquemaReciboCustodiaBaremacionV2,
		VersionEvidenciaCustodia:          ports.VersionReciboCustodiaBaremacionV2,
		EvidenciaCustodiaFirmadoRef:       referenciaExterna(12),
		HuellaEvidenciaCustodiaSHA256:     huellaExterna("evidencia-custodia-externa"),
		EsquemaEvidenciaRetencion:         ports.EsquemaReciboRetencionBaremacionV2,
		VersionEvidenciaRetencion:         ports.VersionReciboRetencionBaremacionV2,
		EvidenciaRetencionFirmadoRef:      referenciaExterna(13),
		HuellaEvidenciaRetencionSHA256:    huellaExterna("evidencia-retencion-externa"),
		PoliticaRetencionRef:              "politica-retencion-externa",
		PoliticaRetencionVersion:          4,
		HuellaPoliticaRetencionSHA256:     huellaExterna("politica-retencion-externa"),
		RetenidoHasta:                     time.Date(2040, 1, 2, 3, 4, 5, 6000, time.UTC),

		HuellaAgregadoObjetivoSHA256: huellaExterna("agregado-objetivo-externo"),
		MotivoClave:                  "merito_acreditado",
		MotivoHMAC: ports.HMACMotivoBaremacion{
			Version:      ports.VersionHMACMotivoBaremacionV1,
			ClaveHMACRef: "motivo-baremacion-externo-v1",
			ValorHMAC:    huellaExterna("hmac-motivo-externo"),
		},
	}
	if err := intencion.Validar(); err != nil {
		t.Fatalf("intencion externa ligada invalida: %v", err)
	}
	return intencion
}

func referenciaExterna(identificador uint64) string {
	return fmt.Sprintf("f47ac10b-58cc-4372-a567-%012x", identificador)
}

func huellaSnapshotExterna(
	dominio ports.DominioClaveHMACBaremacion,
	referencia string,
	revision uint64,
	version uint16,
	generacion uint32,
	clave string,
) string {
	material := make([]byte, 0, 256)
	material = anexarCampoExterno(material, 1, []byte(dominio))
	material = anexarCampoExterno(material, 2, []byte(referencia))
	material = anexarCampoExterno(material, 3, uint64Externo(revision))
	material = anexarCampoExterno(material, 4, []byte{1})
	material = anexarCampoExterno(material, 100, uint16Externo(version))
	material = anexarCampoExterno(material, 101, uint32Externo(generacion))
	material = anexarCampoExterno(material, 102, []byte(clave))
	huella := sha256.Sum256(material)
	return hex.EncodeToString(huella[:])
}

func huellaExterna(valor string) string {
	huella := sha256.Sum256([]byte(valor))
	return hex.EncodeToString(huella[:])
}

func hmacHexExterno(clave, material []byte) string {
	mac := hmac.New(sha256.New, clave)
	_, _ = mac.Write(material)
	return hex.EncodeToString(mac.Sum(nil))
}

func copiarCargaDuranteVisitaExterna(
	visitar func(func(ports.MaterialCanonicoEfimeroBaremacion) error) error,
) ([]byte, error) {
	var copia []byte
	err := visitar(func(material ports.MaterialCanonicoEfimeroBaremacion) error {
		return material.VisitarBytes(func(valor []byte) error {
			copia = append([]byte(nil), valor...)
			return nil
		})
	})
	return copia, err
}

func borrarExterno(material []byte) {
	for posicion := range material {
		material[posicion] = 0
	}
}

func anexarCampoExterno(destino []byte, etiqueta uint16, valor []byte) []byte {
	var cabecera [10]byte
	binary.BigEndian.PutUint16(cabecera[:2], etiqueta)
	binary.BigEndian.PutUint64(cabecera[2:], uint64(len(valor)))
	destino = append(destino, cabecera[:]...)
	return append(destino, valor...)
}

func uint16Externo(valor uint16) []byte {
	resultado := make([]byte, 2)
	binary.BigEndian.PutUint16(resultado, valor)
	return resultado
}

func uint32Externo(valor uint32) []byte {
	resultado := make([]byte, 4)
	binary.BigEndian.PutUint32(resultado, valor)
	return resultado
}

func uint64Externo(valor uint64) []byte {
	resultado := make([]byte, 8)
	binary.BigEndian.PutUint64(resultado, valor)
	return resultado
}
