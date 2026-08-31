package ports

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/fxamacker/cbor/v2"
	"gopkg.in/yaml.v3"
)

const (
	organizacionCandidaturaPrueba = "organizacion:diputacion-granada"
	actorCandidaturaPrueba        = "actor:tecnica-rrhh-001"
	perfilCandidaturaPrueba       = "perfil:tecnica-rrhh"
)

func TestCandidaturaAltaConstruyeExtraeYConservaCopia(t *testing.T) {
	datos := datosCandidaturaAltaPrueba(3, "a", "b")
	candidatura := nuevaCandidaturaAltaPrueba(t, datos)

	obtenidos, err := candidatura.Datos()
	if err != nil || obtenidos.ReservaRef != datos.ReservaRef ||
		obtenidos.Referencias != datos.Referencias ||
		obtenidos.AmbitoIdempotenciaHMAC != datos.AmbitoIdempotenciaHMAC ||
		obtenidos.HuellaPeticionHMAC != datos.HuellaPeticionHMAC ||
		obtenidos.OrganizacionRef != datos.OrganizacionRef ||
		obtenidos.ActorRef != datos.ActorRef ||
		obtenidos.PerfilRef != datos.PerfilRef ||
		!obtenidos.InstanteEfecto.Equal(datos.InstanteEfecto) {
		t.Fatalf("extracción inesperada: %#v, %v", obtenidos, err)
	}

	obtenidos.ReservaRef = "reserva:alterada"
	deNuevo, err := candidatura.Datos()
	if err != nil || deNuevo.ReservaRef != datos.ReservaRef {
		t.Fatalf("la extracción alteró la candidatura: %#v, %v", deNuevo, err)
	}
	if _, err := (CandidaturaAlta{}).Datos(); !errors.Is(err, ErrPreparacionAltaInvalida) {
		t.Fatalf("la candidatura cero no falló cerrada: %v", err)
	}
}

func TestCandidaturaAltaRechazaCadaCoordenadaInvalida(t *testing.T) {
	base := datosCandidaturaAltaPrueba(3, "a", "b")
	casos := map[string]func(*DatosCandidaturaAlta){
		"reserva": func(d *DatosCandidaturaAlta) { d.ReservaRef = "" },
		"expediente": func(d *DatosCandidaturaAlta) {
			d.Referencias.ExpedienteRef = ""
		},
		"numero visible": func(d *DatosCandidaturaAlta) {
			d.Referencias.NumeroVisible = ""
		},
		"recibo": func(d *DatosCandidaturaAlta) { d.Referencias.ReciboRef = "" },
		"ambito": func(d *DatosCandidaturaAlta) { d.AmbitoIdempotenciaHMAC = "" },
		"huella": func(d *DatosCandidaturaAlta) { d.HuellaPeticionHMAC = "" },
		"dominio ambito": func(d *DatosCandidaturaAlta) {
			d.AmbitoIdempotenciaHMAC = selloCandidaturaAltaPrueba(
				"vec.otro.ambito", 3, "c",
			)
		},
		"dominio huella": func(d *DatosCandidaturaAlta) {
			d.HuellaPeticionHMAC = selloCandidaturaAltaPrueba(
				"vec.otro.huella", 3, "c",
			)
		},
		"generaciones cruzadas": func(d *DatosCandidaturaAlta) {
			d.HuellaPeticionHMAC = selloCandidaturaAltaPrueba(
				dominioHuellaCandidaturaAlta, 2, "c",
			)
		},
		"organizacion": func(d *DatosCandidaturaAlta) { d.OrganizacionRef = "" },
		"actor":        func(d *DatosCandidaturaAlta) { d.ActorRef = "" },
		"perfil":       func(d *DatosCandidaturaAlta) { d.PerfilRef = "" },
		"instante cero": func(d *DatosCandidaturaAlta) {
			d.InstanteEfecto = time.Time{}
		},
		"instante no UTC": func(d *DatosCandidaturaAlta) {
			d.InstanteEfecto = time.Date(
				2026, 8, 31, 12, 0, 0, 0, time.FixedZone("local", 3600),
			)
		},
		"instante submicrosegundo": func(d *DatosCandidaturaAlta) {
			d.InstanteEfecto = d.InstanteEfecto.Add(time.Nanosecond)
		},
	}
	for nombre, alterar := range casos {
		t.Run(nombre, func(t *testing.T) {
			datos := base
			alterar(&datos)
			_, err := NuevaCandidaturaAlta(datos)
			if !errors.Is(err, ErrPreparacionAltaInvalida) {
				t.Fatalf("se aceptó una candidatura inválida: %v", err)
			}
		})
	}
}

func TestSolicitudResolverCandidaturaAltaExigeActivoYMatrizAlineada(t *testing.T) {
	ambitos, huellas := coleccionesCandidaturaAltaPrueba(t)
	propuesta := nuevaCandidaturaAltaPrueba(
		t,
		datosCandidaturaAltaPrueba(3, "a", "b"),
	)
	base := DatosSolicitudResolverCandidaturaAlta{
		AmbitosIdempotenciaHMAC: ambitos,
		HuellasPeticionHMAC:     huellas,
		OrganizacionRef:         organizacionCandidaturaPrueba,
		ActorRef:                actorCandidaturaPrueba,
		PerfilRef:               perfilCandidaturaPrueba,
		Propuesta:               propuesta,
	}
	if _, err := NuevaSolicitudResolverCandidaturaAlta(base); err != nil {
		t.Fatalf("la solicitud válida falló: %v", err)
	}

	matrizCorta, err := NuevaColeccionSellosHMAC(
		selloCandidaturaAltaPrueba(dominioHuellaCandidaturaAlta, 3, "b"),
		[]string{selloCandidaturaAltaPrueba(
			dominioHuellaCandidaturaAlta, 1, "d",
		)},
	)
	if err != nil {
		t.Fatalf("fixture de matriz: %v", err)
	}
	matrizOtraGeneracion, err := NuevaColeccionSellosHMAC(
		selloCandidaturaAltaPrueba(dominioHuellaCandidaturaAlta, 4, "b"),
		[]string{
			selloCandidaturaAltaPrueba(dominioHuellaCandidaturaAlta, 3, "d"),
			selloCandidaturaAltaPrueba(dominioHuellaCandidaturaAlta, 2, "f"),
		},
	)
	if err != nil {
		t.Fatalf("fixture de generación: %v", err)
	}
	propuestaRetenida := nuevaCandidaturaAltaPrueba(
		t,
		datosCandidaturaAltaPrueba(2, "c", "d"),
	)
	propuestaActivaAjena := nuevaCandidaturaAltaPrueba(
		t,
		datosCandidaturaAltaPrueba(3, "e", "f"),
	)
	casos := map[string]func(*DatosSolicitudResolverCandidaturaAlta){
		"ambitos cero": func(d *DatosSolicitudResolverCandidaturaAlta) {
			d.AmbitosIdempotenciaHMAC = ColeccionSellosHMAC{}
		},
		"huellas cero": func(d *DatosSolicitudResolverCandidaturaAlta) {
			d.HuellasPeticionHMAC = ColeccionSellosHMAC{}
		},
		"matrices distintas": func(d *DatosSolicitudResolverCandidaturaAlta) {
			d.HuellasPeticionHMAC = matrizCorta
		},
		"generacion activa distinta": func(d *DatosSolicitudResolverCandidaturaAlta) {
			d.HuellasPeticionHMAC = matrizOtraGeneracion
		},
		"dominios intercambiados": func(d *DatosSolicitudResolverCandidaturaAlta) {
			d.AmbitosIdempotenciaHMAC = huellas
		},
		"organizacion": func(d *DatosSolicitudResolverCandidaturaAlta) {
			d.OrganizacionRef = ""
		},
		"actor": func(d *DatosSolicitudResolverCandidaturaAlta) {
			d.ActorRef = ""
		},
		"perfil": func(d *DatosSolicitudResolverCandidaturaAlta) {
			d.PerfilRef = ""
		},
		"propuesta cero": func(d *DatosSolicitudResolverCandidaturaAlta) {
			d.Propuesta = CandidaturaAlta{}
		},
		"propuesta retenida": func(d *DatosSolicitudResolverCandidaturaAlta) {
			d.Propuesta = propuestaRetenida
		},
		"propuesta activa fuera": func(d *DatosSolicitudResolverCandidaturaAlta) {
			d.Propuesta = propuestaActivaAjena
		},
	}
	for nombre, alterar := range casos {
		t.Run(nombre, func(t *testing.T) {
			datos := base
			alterar(&datos)
			_, err := NuevaSolicitudResolverCandidaturaAlta(datos)
			if !errors.Is(err, ErrPreparacionAltaInvalida) {
				t.Fatalf("se aceptó una solicitud inválida: %v", err)
			}
		})
	}

	for nombre, alterar := range map[string]func(*DatosCandidaturaAlta){
		"organizacion divergente": func(d *DatosCandidaturaAlta) {
			d.OrganizacionRef = "organizacion:otra"
		},
		"actor divergente": func(d *DatosCandidaturaAlta) {
			d.ActorRef = "actor:otro"
		},
		"perfil divergente": func(d *DatosCandidaturaAlta) {
			d.PerfilRef = "perfil:otro"
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			datosPropuesta := datosCandidaturaAltaPrueba(3, "a", "b")
			alterar(&datosPropuesta)
			datos := base
			datos.Propuesta = nuevaCandidaturaAltaPrueba(t, datosPropuesta)
			_, err := NuevaSolicitudResolverCandidaturaAlta(datos)
			if !errors.Is(err, ErrPreparacionAltaInvalida) {
				t.Fatalf("se aceptaron coordenadas divergentes: %v", err)
			}
		})
	}
}

func TestSolicitudResolverCandidaturaAltaClonaColeccionesYPropuesta(t *testing.T) {
	solicitud := nuevaSolicitudCandidaturaAltaPrueba(t)
	primera, err := solicitud.Datos()
	if err != nil {
		t.Fatalf("primera extracción: %v", err)
	}
	segunda, err := solicitud.Datos()
	if err != nil {
		t.Fatalf("segunda extracción: %v", err)
	}
	if primera.AmbitosIdempotenciaHMAC.datos ==
		segunda.AmbitosIdempotenciaHMAC.datos ||
		primera.HuellasPeticionHMAC.datos ==
			segunda.HuellasPeticionHMAC.datos ||
		primera.Propuesta.datos == segunda.Propuesta.datos {
		t.Fatal("Datos reutilizó almacenamiento mutable")
	}
	primera.AmbitosIdempotenciaHMAC.datos.Retenidos[0].Valor = "alterado"
	primera.HuellasPeticionHMAC.datos.Retenidos[0].Valor = "alterado"
	primera.Propuesta.datos.ReservaRef = "reserva:alterada"
	tercera, err := solicitud.Datos()
	if err != nil || tercera.AmbitosIdempotenciaHMAC.datos.Retenidos[0].Valor ==
		"alterado" ||
		tercera.HuellasPeticionHMAC.datos.Retenidos[0].Valor == "alterado" ||
		tercera.Propuesta.datos.ReservaRef == "reserva:alterada" {
		t.Fatalf("la copia defensiva no aisló el original: %#v, %v", tercera, err)
	}
}

func TestSolicitudResolverCandidaturaAltaValidaActivoRetenidosYReplay(t *testing.T) {
	solicitud := nuevaSolicitudCandidaturaAltaPrueba(t)
	datosActivo := datosCandidaturaAltaPrueba(3, "a", "b")
	datosActivo.ReservaRef = "reserva:ct-alta:activa-original"
	datosActivo.Referencias = ReferenciasAlta{
		ExpedienteRef: "expediente:ct:activo-original",
		NumeroVisible: "2026/CT-ACTIVO",
		ReciboRef:     "recibo:ct-alta:activo-original",
	}
	datosActivo.InstanteEfecto = datosActivo.InstanteEfecto.Add(-time.Hour)
	activo := nuevaCandidaturaAltaPrueba(t, datosActivo)
	if err := solicitud.ValidarResultado(activo); err != nil {
		t.Fatalf("resultado activo original rechazado: %v", err)
	}
	if err := solicitud.ValidarResultado(activo); err != nil {
		t.Fatalf("la segunda validación consumió la solicitud: %v", err)
	}

	original := datosCandidaturaAltaPrueba(2, "c", "d")
	original.ReservaRef = "reserva:ct-alta:historica"
	original.Referencias = ReferenciasAlta{
		ExpedienteRef: "expediente:ct:historico",
		NumeroVisible: "2026/CT-HISTORICO",
		ReciboRef:     "recibo:ct-alta:historico",
	}
	original.InstanteEfecto = original.InstanteEfecto.Add(-24 * time.Hour)
	retenido := nuevaCandidaturaAltaPrueba(t, original)
	if err := solicitud.ValidarResultado(retenido); err != nil {
		t.Fatalf("resultado retenido histórico rechazado: %v", err)
	}
	recuperado, err := retenido.Datos()
	if err != nil || recuperado.ReservaRef != original.ReservaRef ||
		recuperado.Referencias != original.Referencias ||
		!recuperado.InstanteEfecto.Equal(original.InstanteEfecto) {
		t.Fatalf("no se conservaron las coordenadas originales: %#v, %v", recuperado, err)
	}

	masAntiguo := nuevaCandidaturaAltaPrueba(
		t,
		datosCandidaturaAltaPrueba(1, "e", "f"),
	)
	if err := solicitud.ValidarResultado(masAntiguo); err != nil {
		t.Fatalf("segundo retenido rechazado: %v", err)
	}
}

func TestSolicitudResolverCandidaturaAltaRechazaResultadoFueraOCruzado(t *testing.T) {
	solicitud := nuevaSolicitudCandidaturaAltaPrueba(t)
	base := datosCandidaturaAltaPrueba(3, "a", "b")
	casos := map[string]CandidaturaAlta{
		"cero": CandidaturaAlta{},
		"par fuera": nuevaCandidaturaAltaPrueba(
			t,
			datosCandidaturaAltaPrueba(3, "e", "f"),
		),
	}
	for nombre, alterar := range map[string]func(*DatosCandidaturaAlta){
		"organizacion": func(d *DatosCandidaturaAlta) {
			d.OrganizacionRef = "organizacion:otra"
		},
		"actor":  func(d *DatosCandidaturaAlta) { d.ActorRef = "actor:otro" },
		"perfil": func(d *DatosCandidaturaAlta) { d.PerfilRef = "perfil:otro" },
	} {
		datos := base
		alterar(&datos)
		casos[nombre] = nuevaCandidaturaAltaPrueba(t, datos)
	}
	cruzado := nuevaCandidaturaAltaPrueba(t, base)
	cruzado.datos.HuellaPeticionHMAC = selloCandidaturaAltaPrueba(
		dominioHuellaCandidaturaAlta, 2, "d",
	)
	casos["generaciones cruzadas"] = cruzado

	for nombre, resultado := range casos {
		t.Run(nombre, func(t *testing.T) {
			if err := solicitud.ValidarResultado(resultado); !errors.Is(err, ErrPreparacionAltaInvalida) {
				t.Fatalf("se aceptó el resultado inválido: %v", err)
			}
		})
	}
}

type codecsMarshalCandidaturaAlta interface {
	MarshalText() ([]byte, error)
	MarshalBinary() ([]byte, error)
	GobEncode() ([]byte, error)
	MarshalCBOR() ([]byte, error)
	MarshalYAML() (any, error)
}

type codecsUnmarshalCandidaturaAlta interface {
	UnmarshalJSON([]byte) error
	UnmarshalText([]byte) error
	UnmarshalBinary([]byte) error
	GobDecode([]byte) error
	UnmarshalCBOR([]byte) error
	UnmarshalYAML(func(any) error) error
}

func TestCandidaturaAltaBloqueaSerializacionBidireccional(t *testing.T) {
	candidatura := nuevaCandidaturaAltaPrueba(
		t,
		datosCandidaturaAltaPrueba(3, "a", "b"),
	)
	datosCandidatura, err := candidatura.Datos()
	if err != nil {
		t.Fatalf("datos candidatura: %v", err)
	}
	solicitud := nuevaSolicitudCandidaturaAltaPrueba(t)
	datosSolicitud, err := solicitud.Datos()
	if err != nil {
		t.Fatalf("datos solicitud: %v", err)
	}
	candidaturaCero := CandidaturaAlta{}
	datosCandidaturaCero := DatosCandidaturaAlta{}
	solicitudCero := SolicitudResolverCandidaturaAlta{}
	datosSolicitudCero := DatosSolicitudResolverCandidaturaAlta{}

	valores := map[string]any{
		"datos":                  datosCandidatura,
		"datos puntero":          &datosCandidatura,
		"candidatura":            candidatura,
		"candidatura puntero":    &candidatura,
		"datos solicitud":        datosSolicitud,
		"datos solicitud ptr":    &datosSolicitud,
		"solicitud":              solicitud,
		"solicitud puntero":      &solicitud,
		"datos cero":             datosCandidaturaCero,
		"datos cero puntero":     &datosCandidaturaCero,
		"candidatura cero":       candidaturaCero,
		"candidatura cero ptr":   &candidaturaCero,
		"datos solicitud cero":   datosSolicitudCero,
		"datos solicitud cero p": &datosSolicitudCero,
		"solicitud cero":         solicitudCero,
		"solicitud cero puntero": &solicitudCero,
	}
	for nombre, valor := range valores {
		t.Run(nombre, func(t *testing.T) {
			comprobarMarshalCandidaturaAltaProhibido(t, valor)
			comprobarRedaccionCandidaturaAlta(t, valor, datosCandidatura)
		})
	}

	destinos := map[string]any{
		"datos":                &datosCandidatura,
		"candidatura":          &candidatura,
		"datos solicitud":      &datosSolicitud,
		"solicitud":            &solicitud,
		"datos cero":           &datosCandidaturaCero,
		"candidatura cero":     &candidaturaCero,
		"datos solicitud cero": &datosSolicitudCero,
		"solicitud cero":       &solicitudCero,
	}
	for nombre, destino := range destinos {
		t.Run(nombre+" decode", func(t *testing.T) {
			comprobarUnmarshalCandidaturaAltaProhibido(t, destino)
		})
	}
}

func TestCandidaturaAltaPunterosNilFallanSinPanico(t *testing.T) {
	var datos *DatosCandidaturaAlta
	var candidatura *CandidaturaAlta
	var datosSolicitud *DatosSolicitudResolverCandidaturaAlta
	var solicitud *SolicitudResolverCandidaturaAlta
	valores := map[string]codecsUnmarshalCandidaturaAlta{
		"datos":           datos,
		"candidatura":     candidatura,
		"datos solicitud": datosSolicitud,
		"solicitud":       solicitud,
	}
	for nombre, valor := range valores {
		t.Run(nombre, func(t *testing.T) {
			comprobarErrorSerializacionCandidaturaAlta(t, "json nil", valor.UnmarshalJSON(nil))
			comprobarErrorSerializacionCandidaturaAlta(t, "gob nil", valor.GobDecode(nil))
			_ = fmt.Sprintf("%v|%+v|%#v", valor, valor, valor)
			var registro bytes.Buffer
			slog.New(slog.NewJSONHandler(&registro, nil)).Info("candidatura_nil", "valor", valor)
		})
	}
}

func comprobarMarshalCandidaturaAltaProhibido(t *testing.T, valor any) {
	t.Helper()
	_, err := json.Marshal(valor)
	comprobarErrorSerializacionCandidaturaAlta(t, "json", err)
	_, err = xml.Marshal(valor)
	comprobarErrorSerializacionCandidaturaAlta(t, "xml", err)
	codecs, ok := valor.(codecsMarshalCandidaturaAlta)
	if !ok {
		t.Fatalf("%T no bloquea los codecs directos", valor)
	}
	_, err = codecs.MarshalText()
	comprobarErrorSerializacionCandidaturaAlta(t, "texto", err)
	_, err = codecs.MarshalBinary()
	comprobarErrorSerializacionCandidaturaAlta(t, "binario", err)
	var destino bytes.Buffer
	comprobarErrorSerializacionCandidaturaAlta(
		t, "gob", gob.NewEncoder(&destino).Encode(valor),
	)
	_, err = codecs.GobEncode()
	comprobarErrorSerializacionCandidaturaAlta(t, "gob directo", err)
	_, err = cbor.Marshal(valor)
	comprobarErrorSerializacionCandidaturaAlta(t, "cbor", err)
	_, err = codecs.MarshalCBOR()
	comprobarErrorSerializacionCandidaturaAlta(t, "cbor directo", err)
	_, err = yaml.Marshal(valor)
	comprobarErrorSerializacionCandidaturaAlta(t, "yaml", err)
	_, err = codecs.MarshalYAML()
	comprobarErrorSerializacionCandidaturaAlta(t, "yaml directo", err)
}

func comprobarUnmarshalCandidaturaAltaProhibido(t *testing.T, destino any) {
	t.Helper()
	comprobarErrorSerializacionCandidaturaAlta(
		t, "json decode", json.Unmarshal([]byte(`{}`), destino),
	)
	comprobarErrorSerializacionCandidaturaAlta(
		t, "xml decode", xml.Unmarshal([]byte(`<candidatura/>`), destino),
	)
	comprobarErrorSerializacionCandidaturaAlta(
		t, "cbor decode", cbor.Unmarshal([]byte{0xa0}, destino),
	)
	comprobarErrorSerializacionCandidaturaAlta(
		t, "yaml decode", yaml.Unmarshal([]byte(`{}`), destino),
	)
	codecs, ok := destino.(codecsUnmarshalCandidaturaAlta)
	if !ok {
		t.Fatalf("%T no bloquea la reconstrucción", destino)
	}
	comprobarErrorSerializacionCandidaturaAlta(
		t, "texto decode", codecs.UnmarshalText([]byte("alterado")),
	)
	comprobarErrorSerializacionCandidaturaAlta(
		t, "binario decode", codecs.UnmarshalBinary([]byte("alterado")),
	)
	comprobarErrorSerializacionCandidaturaAlta(
		t, "gob decode", codecs.GobDecode([]byte("alterado")),
	)
	comprobarErrorSerializacionCandidaturaAlta(
		t, "cbor directo decode", codecs.UnmarshalCBOR([]byte("alterado")),
	)
	comprobarErrorSerializacionCandidaturaAlta(
		t,
		"yaml directo decode",
		codecs.UnmarshalYAML(func(any) error { return nil }),
	)
}

func comprobarRedaccionCandidaturaAlta(t *testing.T, valor any, sensibles DatosCandidaturaAlta) {
	t.Helper()
	formato := fmt.Sprintf("%v|%+v|%#v|%s", valor, valor, valor, valor)
	var registro bytes.Buffer
	slog.New(slog.NewJSONHandler(&registro, nil)).Info(
		"candidatura", "valor", valor,
	)
	log := registro.String()
	if !strings.Contains(formato, redaccionCandidaturaAlta) ||
		!strings.Contains(log, redaccionCandidaturaAlta) {
		t.Fatalf("%T no quedó redactado: %s / %s", valor, formato, log)
	}
	for _, sensible := range []string{
		sensibles.ReservaRef,
		sensibles.Referencias.ExpedienteRef,
		sensibles.Referencias.NumeroVisible,
		sensibles.Referencias.ReciboRef,
		sensibles.AmbitoIdempotenciaHMAC,
		sensibles.HuellaPeticionHMAC,
		sensibles.OrganizacionRef,
		sensibles.ActorRef,
		sensibles.PerfilRef,
		sensibles.InstanteEfecto.Format(time.RFC3339Nano),
	} {
		if strings.Contains(formato, sensible) || strings.Contains(log, sensible) {
			t.Fatalf("%T filtró %q", valor, sensible)
		}
	}
}

func comprobarErrorSerializacionCandidaturaAlta(t *testing.T, nombre string, err error) {
	t.Helper()
	if !errors.Is(err, ErrSerializacionCandidaturaAltaProhibida) {
		t.Fatalf("%s no quedó bloqueado: %v", nombre, err)
	}
}

func nuevaSolicitudCandidaturaAltaPrueba(t *testing.T) SolicitudResolverCandidaturaAlta {
	t.Helper()
	ambitos, huellas := coleccionesCandidaturaAltaPrueba(t)
	propuesta := nuevaCandidaturaAltaPrueba(
		t,
		datosCandidaturaAltaPrueba(3, "a", "b"),
	)
	solicitud, err := NuevaSolicitudResolverCandidaturaAlta(
		DatosSolicitudResolverCandidaturaAlta{
			AmbitosIdempotenciaHMAC: ambitos,
			HuellasPeticionHMAC:     huellas,
			OrganizacionRef:         organizacionCandidaturaPrueba,
			ActorRef:                actorCandidaturaPrueba,
			PerfilRef:               perfilCandidaturaPrueba,
			Propuesta:               propuesta,
		},
	)
	if err != nil {
		t.Fatalf("crear solicitud: %v", err)
	}
	return solicitud
}

func coleccionesCandidaturaAltaPrueba(t *testing.T) (ColeccionSellosHMAC, ColeccionSellosHMAC) {
	t.Helper()
	ambitos, err := NuevaColeccionSellosHMAC(
		selloCandidaturaAltaPrueba(dominioAmbitoCandidaturaAlta, 3, "a"),
		[]string{
			selloCandidaturaAltaPrueba(dominioAmbitoCandidaturaAlta, 2, "c"),
			selloCandidaturaAltaPrueba(dominioAmbitoCandidaturaAlta, 1, "e"),
		},
	)
	if err != nil {
		t.Fatalf("crear ámbitos: %v", err)
	}
	huellas, err := NuevaColeccionSellosHMAC(
		selloCandidaturaAltaPrueba(dominioHuellaCandidaturaAlta, 3, "b"),
		[]string{
			selloCandidaturaAltaPrueba(dominioHuellaCandidaturaAlta, 2, "d"),
			selloCandidaturaAltaPrueba(dominioHuellaCandidaturaAlta, 1, "f"),
		},
	)
	if err != nil {
		t.Fatalf("crear huellas: %v", err)
	}
	return ambitos, huellas
}

func nuevaCandidaturaAltaPrueba(t *testing.T, datos DatosCandidaturaAlta) CandidaturaAlta {
	t.Helper()
	candidatura, err := NuevaCandidaturaAlta(datos)
	if err != nil {
		t.Fatalf("crear candidatura: %v", err)
	}
	return candidatura
}

func datosCandidaturaAltaPrueba(generacion uint32, digestAmbito, digestHuella string) DatosCandidaturaAlta {
	return DatosCandidaturaAlta{
		ReservaRef: "reserva:ct-alta:001",
		Referencias: ReferenciasAlta{
			ExpedienteRef: "expediente:ct:001",
			NumeroVisible: "2026/CT-001",
			ReciboRef:     "recibo:ct-alta:001",
		},
		AmbitoIdempotenciaHMAC: selloCandidaturaAltaPrueba(
			dominioAmbitoCandidaturaAlta,
			generacion,
			digestAmbito,
		),
		HuellaPeticionHMAC: selloCandidaturaAltaPrueba(
			dominioHuellaCandidaturaAlta,
			generacion,
			digestHuella,
		),
		OrganizacionRef: organizacionCandidaturaPrueba,
		ActorRef:        actorCandidaturaPrueba,
		PerfilRef:       perfilCandidaturaPrueba,
		InstanteEfecto: time.Date(
			2026, 8, 31, 12, 34, 56, 123456000, time.UTC,
		),
	}
}

func selloCandidaturaAltaPrueba(dominio string, generacion uint32, digest string) string {
	return fmt.Sprintf(
		"hmac-sha256:%s/v%d:%s",
		dominio,
		generacion,
		strings.Repeat(digest, 64),
	)
}
